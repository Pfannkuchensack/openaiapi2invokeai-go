package workflow

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
)

// Params are the OpenAI-facing parameters to inject into a workflow graph.
type Params struct {
	Prompt   string
	Negative string
	Width    int
	Height   int
	Seed     int64 // -1 or 0 means random
	Steps    int
	CFG      float64
}

// BuildGraph loads a workflow file and applies parameter substitution based on
// the model entry's field mapping. Returns the ready-to-enqueue graph.
func BuildGraph(dataDir string, entry ModelEntry, params Params) (map[string]any, error) {
	wfPath := filepath.Join(dataDir, "workflows", entry.Workflow)
	data, err := os.ReadFile(wfPath)
	if err != nil {
		return nil, fmt.Errorf("read workflow %s: %w", entry.Workflow, err)
	}

	var graph map[string]any
	if err := json.Unmarshal(data, &graph); err != nil {
		return nil, fmt.Errorf("parse workflow: %w", err)
	}

	// Apply defaults first, then explicit params override
	if entry.Defaults != nil {
		for key, val := range entry.Defaults {
			path := mappingPathForKey(entry.Mapping, key)
			if path != "" {
				setField(graph, path, val)
			}
		}
	}

	// Apply explicit parameters
	if params.Prompt != "" && entry.Mapping.Prompt != "" {
		setField(graph, entry.Mapping.Prompt, params.Prompt)
	}
	if params.Negative != "" && entry.Mapping.Negative != "" {
		setField(graph, entry.Mapping.Negative, params.Negative)
	}
	if params.Width > 0 && entry.Mapping.Width != "" {
		setField(graph, entry.Mapping.Width, params.Width)
	}
	if params.Height > 0 && entry.Mapping.Height != "" {
		setField(graph, entry.Mapping.Height, params.Height)
	}
	if entry.Mapping.Seed != "" {
		seed := params.Seed
		if seed <= 0 {
			seed = rand.Int64N(2147483647)
		}
		setField(graph, entry.Mapping.Seed, seed)
	}
	if params.Steps > 0 && entry.Mapping.Steps != "" {
		setField(graph, entry.Mapping.Steps, params.Steps)
	}
	if params.CFG > 0 && entry.Mapping.CFG != "" {
		setField(graph, entry.Mapping.CFG, params.CFG)
	}

	return graph, nil
}

// setField sets a value at a dot-separated path in a nested map.
// Path format: "nodes.<node-id>.<field>" e.g. "nodes.abc123.value"
func setField(graph map[string]any, path string, value any) {
	parts := splitPath(path)
	if len(parts) < 2 {
		return
	}

	current := any(graph)
	for i := 0; i < len(parts)-1; i++ {
		switch m := current.(type) {
		case map[string]any:
			next, ok := m[parts[i]]
			if !ok {
				return
			}
			current = next
		default:
			return
		}
	}

	if m, ok := current.(map[string]any); ok {
		m[parts[len(parts)-1]] = value
	}
}

// splitPath splits a dot-path but preserves dots inside node UUIDs.
// Format: "nodes.<uuid>.field" → ["nodes", "<uuid>", "field"]
func splitPath(path string) []string {
	// Split on dots, but reassemble UUID parts (contains dashes, not dots)
	return strings.Split(path, ".")
}

// mappingPathForKey maps a defaults key name to a field mapping path.
func mappingPathForKey(m FieldMapping, key string) string {
	switch key {
	case "steps":
		return m.Steps
	case "cfg":
		return m.CFG
	case "seed":
		return m.Seed
	case "width":
		return m.Width
	case "height":
		return m.Height
	case "prompt":
		return m.Prompt
	case "negative":
		return m.Negative
	}
	return ""
}

// BuildGraphFromFile is like BuildGraph but allows specifying the workflow file directly
// (used when edit/variant workflows differ from the default).
func BuildGraphFromFile(dataDir string, workflowFile string, entry ModelEntry, params Params) (map[string]any, error) {
	e := entry
	e.Workflow = workflowFile
	return BuildGraph(dataDir, e, params)
}

// SetGraphField sets a value at a dot-path in a graph. Exported for use by handlers.
func SetGraphField(graph map[string]any, path string, value any) {
	setField(graph, path, value)
}

// ResolveSize resolves an OpenAI size string (e.g. "1024x1024") to width/height
// using the model's size presets, or parses directly.
func ResolveSize(entry ModelEntry, sizeStr string) (int, int, error) {
	if sizeStr == "" {
		return 0, 0, nil
	}

	// Check presets first
	if preset, ok := entry.SizePresets[sizeStr]; ok {
		return preset.Width, preset.Height, nil
	}

	// Try to parse "WxH"
	var w, h int
	if _, err := fmt.Sscanf(sizeStr, "%dx%d", &w, &h); err == nil && w > 0 && h > 0 {
		return w, h, nil
	}

	return 0, 0, fmt.Errorf("unknown size %q (not in presets and not WxH format)", sizeStr)
}
