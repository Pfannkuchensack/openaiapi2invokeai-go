package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/Pfannkuchensack/openaiapi2invokeai-go/internal/workflow"
)

// handleImageEdits implements POST /v1/images/edits (inpainting).
// Accepts multipart/form-data with: image, mask (optional), prompt, model, n, size.
func (s *Server) handleImageEdits(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid_request_error", "invalid multipart form: "+err.Error())
		return
	}

	prompt := r.FormValue("prompt")
	if prompt == "" {
		s.writeError(w, http.StatusBadRequest, "invalid_request_error", "prompt is required")
		return
	}

	modelID := r.FormValue("model")
	n := parseIntOr(r.FormValue("n"), 1)
	size := r.FormValue("size")

	// Read image
	imageData, err := readFormFile(r, "image")
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid_request_error", "image is required: "+err.Error())
		return
	}

	// Read mask (optional)
	maskData, _ := readFormFile(r, "mask")

	// Resolve model
	entry, ok := s.resolveModel(modelID)
	if !ok {
		s.writeError(w, http.StatusNotFound, "invalid_request_error", fmt.Sprintf("model %q not found", modelID))
		return
	}

	// Use edit workflow if available, otherwise fall back to default
	workflowFile := entry.EditWorkflow
	if workflowFile == "" {
		workflowFile = entry.Workflow
	}

	width, height, err := workflow.ResolveSize(entry, size)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}

	s.log.Info("image edit", "model", entry.ID, "prompt", prompt, "n", n, "has_mask", maskData != nil)

	var images []ImageData
	for i := 0; i < n; i++ {
		params := workflow.Params{
			Prompt: prompt,
			Width:  width,
			Height: height,
			Seed:   -1,
		}

		graph, err := workflow.BuildGraph(s.cfg.DataDir, entry, params)
		if err != nil {
			// Try with edit workflow file directly
			graph, err = workflow.BuildGraphFromFile(s.cfg.DataDir, workflowFile, entry, params)
			if err != nil {
				s.writeError(w, http.StatusInternalServerError, "server_error", "build graph: "+err.Error())
				return
			}
		}

		// Inject image and mask as base64 into the graph
		if entry.Mapping.Image != "" {
			workflow.SetGraphField(graph, entry.Mapping.Image, base64.StdEncoding.EncodeToString(imageData))
		}
		if maskData != nil && entry.Mapping.Mask != "" {
			workflow.SetGraphField(graph, entry.Mapping.Mask, base64.StdEncoding.EncodeToString(maskData))
		}

		imgData, err := s.generateImage(r.Context(), graph)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "server_error", "generation failed: "+err.Error())
			return
		}

		images = append(images, ImageData{
			B64JSON:       base64.StdEncoding.EncodeToString(imgData),
			RevisedPrompt: prompt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ImageResponse{
		Created: time.Now().Unix(),
		Data:    images,
	})
}

// handleImageVariations implements POST /v1/images/variations (img2img).
// Accepts multipart/form-data with: image, model, n, size.
func (s *Server) handleImageVariations(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid_request_error", "invalid multipart form: "+err.Error())
		return
	}

	modelID := r.FormValue("model")
	n := parseIntOr(r.FormValue("n"), 1)
	size := r.FormValue("size")

	// Read image
	imageData, err := readFormFile(r, "image")
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid_request_error", "image is required: "+err.Error())
		return
	}

	// Resolve model
	entry, ok := s.resolveModel(modelID)
	if !ok {
		s.writeError(w, http.StatusNotFound, "invalid_request_error", fmt.Sprintf("model %q not found", modelID))
		return
	}

	// Use variant workflow if available
	workflowFile := entry.VariantWorkflow
	if workflowFile == "" {
		workflowFile = entry.Workflow
	}

	width, height, err := workflow.ResolveSize(entry, size)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}

	s.log.Info("image variation", "model", entry.ID, "n", n)

	var images []ImageData
	for i := 0; i < n; i++ {
		params := workflow.Params{
			Prompt: "", // no prompt for variations
			Width:  width,
			Height: height,
			Seed:   -1,
		}

		graph, err := workflow.BuildGraphFromFile(s.cfg.DataDir, workflowFile, entry, params)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "server_error", "build graph: "+err.Error())
			return
		}

		// Inject image
		if entry.Mapping.Image != "" {
			workflow.SetGraphField(graph, entry.Mapping.Image, base64.StdEncoding.EncodeToString(imageData))
		}

		// Set high denoising for variations
		if entry.Mapping.Denoise != "" {
			workflow.SetGraphField(graph, entry.Mapping.Denoise, 0.75)
		}

		imgData, err := s.generateImage(r.Context(), graph)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "server_error", "generation failed: "+err.Error())
			return
		}

		images = append(images, ImageData{
			B64JSON: base64.StdEncoding.EncodeToString(imgData),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ImageResponse{
		Created: time.Now().Unix(),
		Data:    images,
	})
}

// --- Helpers ---

func (s *Server) resolveModel(modelID string) (workflow.ModelEntry, bool) {
	if modelID == "" {
		models := s.registry.List()
		if len(models) == 0 {
			return workflow.ModelEntry{}, false
		}
		return models[0], true
	}
	return s.registry.Get(modelID)
}

func readFormFile(r *http.Request, field string) ([]byte, error) {
	file, _, err := r.FormFile(field)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(file)
}

func parseIntOr(s string, fallback int) int {
	if s == "" {
		return fallback
	}
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}
