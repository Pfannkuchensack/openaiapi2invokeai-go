package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Pfannkuchensack/openaiapi2invokeai-go/internal/invoke"
	"github.com/Pfannkuchensack/openaiapi2invokeai-go/internal/workflow"
)

// --- OpenAI Types ---

type ImageGenerationRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	N              int    `json:"n,omitempty"`
	Size           string `json:"size,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"` // b64_json or url
	User           string `json:"user,omitempty"`
}

type ImageResponse struct {
	Created int64       `json:"created"`
	Data    []ImageData `json:"data"`
}

type ImageData struct {
	B64JSON       string `json:"b64_json,omitempty"`
	URL           string `json:"url,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code,omitempty"`
}

// --- Handlers ---

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	type Model struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		OwnedBy string `json:"owned_by"`
	}
	type Response struct {
		Object string  `json:"object"`
		Data   []Model `json:"data"`
	}

	models := s.registry.List()
	data := make([]Model, len(models))
	for i, m := range models {
		data[i] = Model{
			ID:      m.ID,
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: "invokeai",
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Response{Object: "list", Data: data})
}

func (s *Server) handleImageGenerations(w http.ResponseWriter, r *http.Request) {
	var req ImageGenerationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid_request_error", "invalid JSON: "+err.Error())
		return
	}

	if req.Prompt == "" {
		s.writeError(w, http.StatusBadRequest, "invalid_request_error", "prompt is required")
		return
	}

	if req.N <= 0 {
		req.N = 1
	}
	if req.ResponseFormat == "" {
		req.ResponseFormat = "b64_json"
	}
	if req.ResponseFormat != "b64_json" {
		s.writeError(w, http.StatusBadRequest, "invalid_request_error", "only b64_json response_format is currently supported")
		return
	}

	// Look up model in registry
	modelID := req.Model
	if modelID == "" {
		// Use first registered model as default
		models := s.registry.List()
		if len(models) == 0 {
			s.writeError(w, http.StatusBadRequest, "invalid_request_error", "no models configured, please set up a model in the admin UI")
			return
		}
		modelID = models[0].ID
	}

	entry, ok := s.registry.Get(modelID)
	if !ok {
		s.writeError(w, http.StatusNotFound, "invalid_request_error", fmt.Sprintf("model %q not found", modelID))
		return
	}

	// Resolve size
	width, height, err := workflow.ResolveSize(entry, req.Size)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}

	s.log.Info("image generation",
		"model", modelID,
		"prompt", req.Prompt,
		"n", req.N,
		"size", req.Size,
	)

	var images []ImageData

	for i := 0; i < req.N; i++ {
		params := workflow.Params{
			Prompt: req.Prompt,
			Width:  width,
			Height: height,
			Seed:   -1, // random
		}

		graph, err := workflow.BuildGraph(s.cfg.DataDir, entry, params)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "server_error", "build graph: "+err.Error())
			return
		}

		imgData, err := s.generateImage(r.Context(), graph)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "server_error", "generation failed: "+err.Error())
			return
		}

		images = append(images, ImageData{
			B64JSON:       base64.StdEncoding.EncodeToString(imgData),
			RevisedPrompt: req.Prompt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ImageResponse{
		Created: time.Now().Unix(),
		Data:    images,
	})
}

// generateImage enqueues a graph, waits for completion, and returns image bytes.
func (s *Server) generateImage(ctx context.Context, graph map[string]any) ([]byte, error) {
	resp, err := s.invoke.EnqueueBatch(ctx, invoke.Graph(graph))
	if err != nil {
		return nil, fmt.Errorf("enqueue: %w", err)
	}

	if len(resp.ItemIDs) == 0 {
		return nil, fmt.Errorf("no items enqueued")
	}

	itemID := resp.ItemIDs[0]
	batchID := resp.Batch.BatchID
	s.log.Debug("enqueued", "item_id", itemID, "batch_id", batchID)

	// Wait for completion
	status, err := s.invoke.WaitForCompletion(ctx, batchID, itemID)
	if err != nil {
		return nil, fmt.Errorf("wait: %w", err)
	}

	if status.Status != "completed" {
		return nil, fmt.Errorf("generation %s: %s", status.Status, status.Error)
	}

	// Get results and find image
	detail, err := s.invoke.GetQueueItemDetail(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("get results: %w", err)
	}

	names := s.invoke.GetImageNames(detail)
	if len(names) == 0 {
		return nil, fmt.Errorf("no images in output")
	}

	// Fetch the last image (typically the final output)
	imgBytes, _, err := s.invoke.GetImageBytes(ctx, names[len(names)-1])
	if err != nil {
		return nil, fmt.Errorf("fetch image: %w", err)
	}

	return imgBytes, nil
}

func (s *Server) writeError(w http.ResponseWriter, status int, errType, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error: ErrorDetail{
			Message: message,
			Type:    errType,
		},
	})
}
