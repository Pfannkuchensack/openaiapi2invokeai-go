package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Registry manages model-name → workflow mappings persisted as JSON.
type Registry struct {
	mu       sync.RWMutex
	filePath string
	data     RegistryData
}

// RegistryData is the top-level structure of registry.json.
type RegistryData struct {
	Models []ModelEntry `json:"models"`
}

// ModelEntry maps an OpenAI model ID to an InvokeAI workflow + parameters.
type ModelEntry struct {
	ID              string            `json:"id"`
	Workflow        string            `json:"workflow"`
	EditWorkflow    string            `json:"edit_workflow,omitempty"`    // inpainting workflow
	VariantWorkflow string            `json:"variant_workflow,omitempty"` // img2img workflow
	Defaults        map[string]any    `json:"defaults,omitempty"`
	Mapping         FieldMapping      `json:"mapping"`
	SizePresets     map[string]Size   `json:"size_presets,omitempty"`
}

// FieldMapping defines which graph fields to substitute for OpenAI parameters.
type FieldMapping struct {
	Prompt   string `json:"prompt"`
	Negative string `json:"negative,omitempty"`
	Width    string `json:"width,omitempty"`
	Height   string `json:"height,omitempty"`
	Seed     string `json:"seed,omitempty"`
	Steps    string `json:"steps,omitempty"`
	CFG      string `json:"cfg,omitempty"`
	Image    string `json:"image,omitempty"`    // input image (for edits/variations)
	Mask     string `json:"mask,omitempty"`     // mask image (for edits/inpainting)
	Denoise  string `json:"denoise,omitempty"`  // denoising strength (for variations)
}

// Size holds width/height for a size preset.
type Size struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// NewRegistry loads or creates the registry at the given data directory.
func NewRegistry(dataDir string) (*Registry, error) {
	fp := filepath.Join(dataDir, "registry.json")
	r := &Registry{filePath: fp}

	if _, err := os.Stat(fp); os.IsNotExist(err) {
		r.data = RegistryData{Models: []ModelEntry{}}
		return r, nil
	}

	data, err := os.ReadFile(fp)
	if err != nil {
		return nil, fmt.Errorf("read registry: %w", err)
	}

	if err := json.Unmarshal(data, &r.data); err != nil {
		return nil, fmt.Errorf("parse registry: %w", err)
	}

	return r, nil
}

// List returns all registered models.
func (r *Registry) List() []ModelEntry {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ModelEntry, len(r.data.Models))
	copy(out, r.data.Models)
	return out
}

// Get returns a model entry by ID.
func (r *Registry) Get(id string) (ModelEntry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, m := range r.data.Models {
		if m.ID == id {
			return m, true
		}
	}
	return ModelEntry{}, false
}

// Put adds or updates a model entry.
func (r *Registry) Put(entry ModelEntry) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	found := false
	for i, m := range r.data.Models {
		if m.ID == entry.ID {
			r.data.Models[i] = entry
			found = true
			break
		}
	}
	if !found {
		r.data.Models = append(r.data.Models, entry)
	}

	return r.save()
}

// Delete removes a model entry by ID.
func (r *Registry) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, m := range r.data.Models {
		if m.ID == id {
			r.data.Models = append(r.data.Models[:i], r.data.Models[i+1:]...)
			return r.save()
		}
	}
	return fmt.Errorf("model %q not found", id)
}

func (r *Registry) save() error {
	dir := filepath.Dir(r.filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(r.data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(r.filePath, data, 0o644)
}
