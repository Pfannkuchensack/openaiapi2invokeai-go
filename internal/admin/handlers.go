package admin

import (
	"context"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/Pfannkuchensack/openaiapi2invokeai-go/internal/config"
	"github.com/Pfannkuchensack/openaiapi2invokeai-go/internal/invoke"
	"github.com/Pfannkuchensack/openaiapi2invokeai-go/internal/workflow"
	"github.com/go-chi/chi/v5"
)

//go:embed templates/*.html
var templateFS embed.FS

type Handler struct {
	cfg      *config.Config
	registry *workflow.Registry
	invoke   *invoke.Client
	log      *slog.Logger
	tmpl     *template.Template
}

func NewHandler(cfg *config.Config, registry *workflow.Registry, invokeClient *invoke.Client, log *slog.Logger) *Handler {
	return &Handler{
		cfg:      cfg,
		registry: registry,
		invoke:   invokeClient,
		log:      log,
		tmpl:     nil, // parsed per-request to allow page-specific "content" blocks
	}
}

func (h *Handler) funcMap() template.FuncMap {
	return template.FuncMap{
		"truncVal": func(v any) string {
			s := fmt.Sprintf("%v", v)
			if len(s) > 30 {
				return s[:30] + "..."
			}
			return s
		},
	}
}

func (h *Handler) parseTemplate(page string) *template.Template {
	return template.Must(
		template.New("").Funcs(h.funcMap()).ParseFS(templateFS, "templates/layout.html", "templates/"+page),
	)
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.dashboard)
	r.Get("/workflows", h.workflows)
	r.Post("/workflows/upload", h.workflowUpload)
	r.Delete("/workflows/{name}", h.workflowDelete)
	r.Get("/workflows/inspect/{name}", h.workflowInspect)
	r.Get("/models", h.models)
	r.Post("/models/save", h.modelSave)
	r.Delete("/models/{id}", h.modelDelete)
	r.Get("/setup", h.setup)
	r.Post("/setup/install", h.setupInstall)
	r.Get("/test", h.testPage)
	r.Post("/test/generate", h.testGenerate)
	r.Get("/settings", h.settings)

	return r
}

// --- Dashboard ---

type QueueStatus struct {
	Pending    int `json:"pending"`
	InProgress int `json:"in_progress"`
	Completed  int `json:"completed"`
	Failed     int `json:"failed"`
}

func (h *Handler) dashboard(w http.ResponseWriter, r *http.Request) {
	// Check InvokeAI connection
	invokeOK := false
	invokeVersion := ""
	var queueStatus *QueueStatus

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	if ver, err := h.getInvokeVersion(ctx); err == nil {
		invokeOK = true
		invokeVersion = ver
	}
	if qs, err := h.getQueueStatus(ctx); err == nil {
		queueStatus = qs
	}

	workflows, _ := workflow.ListWorkflows(h.cfg.DataDir)

	h.render(w, "dashboard.html", map[string]any{
		"Title":         "Dashboard",
		"Nav":           "dashboard",
		"InvokeURL":     h.cfg.InvokeURL,
		"InvokeOK":      invokeOK,
		"InvokeVersion": invokeVersion,
		"VersionOK":     CheckInvokeVersion(invokeVersion),
		"MinVersion":    MinInvokeVersion,
		"ModelCount":    len(h.registry.List()),
		"WorkflowCount": len(workflows),
		"QueueStatus":   queueStatus,
	})
}

// --- Workflows ---

type WorkflowInfo struct {
	Name      string
	NodeCount int
}

func (h *Handler) workflows(w http.ResponseWriter, r *http.Request) {
	h.render(w, "workflows.html", map[string]any{
		"Title":     "Workflows",
		"Nav":       "workflows",
		"Workflows": h.listWorkflows(),
	})
}

func (h *Handler) workflowUpload(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(32 << 20)
	file, header, err := r.FormFile("workflow")
	if err != nil {
		http.Error(w, "no file uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "read error", http.StatusInternalServerError)
		return
	}

	// Validate JSON
	var raw json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	dir := filepath.Join(h.cfg.DataDir, "workflows")
	os.MkdirAll(dir, 0o755)
	dest := filepath.Join(dir, header.Filename)
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		http.Error(w, "write error", http.StatusInternalServerError)
		return
	}

	h.log.Info("workflow uploaded", "file", header.Filename)

	// Return updated list (HTMX partial)
	h.renderFragment(w, "workflow-list", "workflows.html", map[string]any{
		"Workflows": h.listWorkflows(),
	})
}

func (h *Handler) workflowDelete(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	path := filepath.Join(h.cfg.DataDir, "workflows", name)
	os.Remove(path)
	h.log.Info("workflow deleted", "file", name)

	h.renderFragment(w, "workflow-list", "workflows.html", map[string]any{
		"Workflows": h.listWorkflows(),
	})
}

func (h *Handler) workflowInspect(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	nodes, err := workflow.InspectWorkflow(h.cfg.DataDir, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.render(w, "workflow_inspect.html", map[string]any{
		"Title":    "Inspect " + name,
		"Nav":      "workflows",
		"Filename": name,
		"Nodes":    nodes,
	})
}

// --- Models ---

func (h *Handler) models(w http.ResponseWriter, r *http.Request) {
	workflows, _ := workflow.ListWorkflows(h.cfg.DataDir)
	h.render(w, "models.html", map[string]any{
		"Title":     "Models",
		"Nav":       "models",
		"Models":    h.registry.List(),
		"Workflows": workflows,
	})
}

func (h *Handler) modelSave(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()

	entry := workflow.ModelEntry{
		ID:       r.FormValue("id"),
		Workflow: r.FormValue("workflow"),
		Mapping: workflow.FieldMapping{
			Prompt:   r.FormValue("map_prompt"),
			Negative: r.FormValue("map_negative"),
			Width:    r.FormValue("map_width"),
			Height:   r.FormValue("map_height"),
			Seed:     r.FormValue("map_seed"),
			Steps:    r.FormValue("map_steps"),
			CFG:      r.FormValue("map_cfg"),
		},
	}

	// Parse size presets
	if sp := r.FormValue("size_presets"); sp != "" {
		var presets map[string]workflow.Size
		if err := json.Unmarshal([]byte(sp), &presets); err == nil {
			entry.SizePresets = presets
		}
	}

	// Parse defaults
	if d := r.FormValue("defaults"); d != "" {
		var defaults map[string]any
		if err := json.Unmarshal([]byte(d), &defaults); err == nil {
			entry.Defaults = defaults
		}
	}

	if err := h.registry.Put(entry); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.log.Info("model saved", "id", entry.ID)

	h.renderFragment(w, "model-list", "models.html", map[string]any{
		"Models": h.registry.List(),
	})
}

func (h *Handler) modelDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	h.registry.Delete(id)
	h.log.Info("model deleted", "id", id)

	h.renderFragment(w, "model-list", "models.html", map[string]any{
		"Models": h.registry.List(),
	})
}

// --- Test ---

func (h *Handler) testPage(w http.ResponseWriter, r *http.Request) {
	h.render(w, "test.html", map[string]any{
		"Title":  "Test",
		"Nav":    "test",
		"Models": h.registry.List(),
	})
}

func (h *Handler) testGenerate(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	model := r.FormValue("model")
	prompt := r.FormValue("prompt")
	size := r.FormValue("size")

	entry, ok := h.registry.Get(model)
	if !ok {
		h.renderFragment(w, "test-result", "test.html", map[string]any{"Error": "model not found"})
		return
	}

	width, height, _ := workflow.ResolveSize(entry, size)

	params := workflow.Params{
		Prompt: prompt,
		Width:  width,
		Height: height,
		Seed:   -1,
	}

	graph, err := workflow.BuildGraph(h.cfg.DataDir, entry, params)
	if err != nil {
		h.renderFragment(w, "test-result", "test.html", map[string]any{"Error": "build graph: " + err.Error()})
		return
	}

	start := time.Now()

	resp, err := h.invoke.EnqueueBatch(r.Context(), invoke.Graph(graph))
	if err != nil {
		h.renderFragment(w, "test-result", "test.html", map[string]any{"Error": "enqueue: " + err.Error()})
		return
	}
	if len(resp.ItemIDs) == 0 {
		h.renderFragment(w, "test-result", "test.html", map[string]any{"Error": "no items enqueued"})
		return
	}

	status, err := h.invoke.WaitForCompletion(r.Context(), resp.Batch.BatchID, resp.ItemIDs[0])
	if err != nil {
		h.renderFragment(w, "test-result", "test.html", map[string]any{"Error": "wait: " + err.Error()})
		return
	}
	if status.Status != "completed" {
		h.renderFragment(w, "test-result", "test.html", map[string]any{"Error": fmt.Sprintf("generation %s: %s", status.Status, status.Error)})
		return
	}

	detail, err := h.invoke.GetQueueItemDetail(r.Context(), resp.ItemIDs[0])
	if err != nil {
		h.renderFragment(w, "test-result", "test.html", map[string]any{"Error": "get results: " + err.Error()})
		return
	}

	names := h.invoke.GetImageNames(detail)
	if len(names) == 0 {
		h.renderFragment(w, "test-result", "test.html", map[string]any{"Error": "no images in output"})
		return
	}

	imgBytes, _, err := h.invoke.GetImageBytes(r.Context(), names[len(names)-1])
	if err != nil {
		h.renderFragment(w, "test-result", "test.html", map[string]any{"Error": "fetch image: " + err.Error()})
		return
	}

	duration := time.Since(start).Round(time.Millisecond)

	h.renderFragment(w, "test-result", "test.html", map[string]any{
		"Duration": duration.String(),
		"ImageB64": base64.StdEncoding.EncodeToString(imgBytes),
	})
}

// --- Quick Setup ---

func (h *Handler) setup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	invokeModels := h.getInvokeModels(ctx)

	// Fetch sub-model options for the setup form
	allModels := h.fetchInvokeModels(ctx, "")
	var qwen3Encoders, fluxVAEs []InvokeModel
	for _, m := range allModels {
		switch m.Type {
		case "qwen3_encoder":
			qwen3Encoders = append(qwen3Encoders, m)
		case "vae":
			if m.Base == "flux" || m.Base == "flux2" || m.Base == "z-image" || m.Base == "any" {
				fluxVAEs = append(fluxVAEs, m)
			}
		}
	}

	h.render(w, "setup.html", map[string]any{
		"Title":          "Quick Setup",
		"Nav":            "setup",
		"Presets":        Presets,
		"InvokeModels":   invokeModels,
		"Qwen3Encoders":  qwen3Encoders,
		"FluxVAEs":       fluxVAEs,
	})
}

func (h *Handler) setupInstall(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	presetID := r.FormValue("preset")
	modelKey := r.FormValue("model_key")
	modelName := r.FormValue("model_name")
	modelHash := r.FormValue("model_hash")
	modelBase := r.FormValue("model_base")
	customID := r.FormValue("custom_id")

	preset, ok := PresetByID(presetID)
	if !ok {
		http.Error(w, "unknown preset", http.StatusBadRequest)
		return
	}

	// Write workflow file
	dir := filepath.Join(h.cfg.DataDir, "workflows")
	os.MkdirAll(dir, 0o755)

	// Patch model reference into workflow JSON
	wfData := preset.WorkflowJSON
	if modelKey != "" {
		wfData = patchModelRef(wfData, modelKey, modelName, modelHash, modelBase)
	}

	// Patch sub-models based on architecture
	switch presetID {
	case "flux":
		wfData = h.patchSubModels(r.Context(), wfData, []subModelSpec{
			{"t5_encoder_model", "t5_encoder", "flux"},
			{"clip_embed_model", "clip_embed", "flux"},
			{"vae_model", "vae", "flux"},
		})
	case "zimage", "flux2klein":
		// Use user-selected qwen3 encoder and VAE
		qwen3Key := r.FormValue("qwen3_key")
		qwen3Name := r.FormValue("qwen3_name")
		qwen3Hash := r.FormValue("qwen3_hash")
		vaeKey := r.FormValue("vae_key")
		vaeName := r.FormValue("vae_name")
		vaeHash := r.FormValue("vae_hash")
		wfData = h.patchManualSubModels(wfData, qwen3Key, qwen3Name, qwen3Hash, vaeKey, vaeName, vaeHash)
	}

	os.WriteFile(filepath.Join(dir, preset.WorkflowFile), []byte(wfData), 0o644)

	// Register model
	entry := preset.Entry
	if customID != "" {
		entry.ID = customID
	}

	h.registry.Put(entry)
	h.log.Info("preset installed", "preset", presetID, "model_id", entry.ID, "model_key", modelKey)

	// Redirect to models page
	http.Redirect(w, r, "/admin/models", http.StatusSeeOther)
}

type InvokeModel struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	Base string `json:"base"`
	Type string `json:"type"`
	Hash string `json:"hash"`
}

func (h *Handler) getInvokeModels(ctx context.Context) []InvokeModel {
	return h.fetchInvokeModels(ctx, "main")
}

func (h *Handler) fetchInvokeModels(ctx context.Context, modelType string) []InvokeModel {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.cfg.InvokeURL+"/api/v2/models/", nil)
	if err != nil {
		return nil
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	var result struct {
		Models []InvokeModel `json:"models"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if modelType == "" {
		return result.Models
	}

	var filtered []InvokeModel
	for _, m := range result.Models {
		if m.Type == modelType {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

func (h *Handler) findSubModel(ctx context.Context, modelType string, preferredBase string) *InvokeModel {
	all := h.fetchInvokeModels(ctx, "")
	var candidates []InvokeModel
	for _, m := range all {
		if m.Type == modelType && (m.Base == preferredBase || m.Base == "any") {
			candidates = append(candidates, m)
		}
	}
	if len(candidates) == 0 {
		// Fallback: just match type
		for _, m := range all {
			if m.Type == modelType {
				candidates = append(candidates, m)
			}
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	// Prefer the last candidate (typically the largest/most capable model)
	result := candidates[len(candidates)-1]
	return &result
}

func (h *Handler) patchManualSubModels(wfJSON, qwen3Key, qwen3Name, qwen3Hash, vaeKey, vaeName, vaeHash string) string {
	var graph map[string]any
	if err := json.Unmarshal([]byte(wfJSON), &graph); err != nil {
		return wfJSON
	}
	nodes, _ := graph["nodes"].(map[string]any)
	loader, _ := nodes["model_loader"].(map[string]any)
	if loader == nil {
		return wfJSON
	}
	if qwen3Key != "" {
		loader["qwen3_encoder_model"] = map[string]any{
			"key": qwen3Key, "name": qwen3Name, "base": "any", "type": "qwen3_encoder", "hash": qwen3Hash,
		}
	}
	if vaeKey != "" {
		loader["vae_model"] = map[string]any{
			"key": vaeKey, "name": vaeName, "base": "flux", "type": "vae", "hash": vaeHash,
		}
	}
	patched, _ := json.MarshalIndent(graph, "", "  ")
	return string(patched)
}

type subModelSpec struct {
	field         string // field name in the model_loader node
	modelType     string // type to search for in InvokeAI
	preferredBase string // preferred base to match
}

func (h *Handler) patchSubModels(ctx context.Context, wfJSON string, specs []subModelSpec) string {
	var graph map[string]any
	if err := json.Unmarshal([]byte(wfJSON), &graph); err != nil {
		return wfJSON
	}

	nodes, ok := graph["nodes"].(map[string]any)
	if !ok {
		return wfJSON
	}

	loader, ok := nodes["model_loader"].(map[string]any)
	if !ok {
		return wfJSON
	}

	for _, spec := range specs {
		if m := h.findSubModel(ctx, spec.modelType, spec.preferredBase); m != nil {
			loader[spec.field] = map[string]any{
				"key": m.Key, "name": m.Name, "base": m.Base, "type": m.Type, "hash": m.Hash,
			}
		}
	}

	patched, err := json.MarshalIndent(graph, "", "  ")
	if err != nil {
		return wfJSON
	}
	return string(patched)
}

func patchModelRef(wfJSON, key, name, hash, base string) string {
	// Parse the workflow, find the model_loader node, and replace the model reference
	var graph map[string]any
	if err := json.Unmarshal([]byte(wfJSON), &graph); err != nil {
		return wfJSON
	}

	nodes, ok := graph["nodes"].(map[string]any)
	if !ok {
		return wfJSON
	}

	// Find model_loader node and update its model field
	if loader, ok := nodes["model_loader"].(map[string]any); ok {
		modelRef := map[string]any{
			"key":  key,
			"name": name,
			"base": base,
			"type": "main",
			"hash": hash,
		}
		loader["model"] = modelRef
	}

	patched, err := json.MarshalIndent(graph, "", "  ")
	if err != nil {
		return wfJSON
	}
	return string(patched)
}

// --- Settings ---

func (h *Handler) settings(w http.ResponseWriter, r *http.Request) {
	h.render(w, "settings.html", map[string]any{
		"Title":  "Settings",
		"Nav":    "settings",
		"Config": h.cfg,
	})
}

// --- Helpers ---

func (h *Handler) render(w http.ResponseWriter, page string, data map[string]any) {
	tmpl := h.parseTemplate(page)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		h.log.Error("template render", "error", err, "template", page)
	}
}

func (h *Handler) renderFragment(w http.ResponseWriter, name string, page string, data map[string]any) {
	tmpl := h.parseTemplate(page)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		h.log.Error("template render fragment", "error", err, "template", name)
	}
}

func (h *Handler) listWorkflows() []WorkflowInfo {
	names, _ := workflow.ListWorkflows(h.cfg.DataDir)
	var infos []WorkflowInfo
	for _, name := range names {
		nodes, _ := workflow.InspectWorkflow(h.cfg.DataDir, name)
		infos = append(infos, WorkflowInfo{Name: name, NodeCount: len(nodes)})
	}
	return infos
}

func (h *Handler) getInvokeVersion(ctx context.Context) (string, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, h.cfg.InvokeURL+"/api/v1/app/version", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var v struct {
		Version string `json:"version"`
	}
	json.NewDecoder(resp.Body).Decode(&v)
	return v.Version, nil
}

func (h *Handler) getQueueStatus(ctx context.Context) (*QueueStatus, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, h.cfg.InvokeURL+"/api/v1/queue/default/status", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var s struct {
		Queue QueueStatus `json:"queue"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return nil, err
	}
	return &s.Queue, nil
}
