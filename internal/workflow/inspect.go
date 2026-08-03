package workflow

import (
	"os"
	"path/filepath"
	"sort"
)

// NodeInfo describes a node found during workflow inspection.
type NodeInfo struct {
	ID     string         `json:"id"`
	Type   string         `json:"type"`
	Label  string         `json:"label,omitempty"`
	Fields map[string]any `json:"fields"`
}

// InspectWorkflow loads a workflow and returns its nodes with their fields,
// useful for the admin UI to suggest field mappings.
func InspectWorkflow(dataDir, filename string) ([]NodeInfo, error) {
	parsed, err := LoadWorkflow(dataDir, filename)
	if err != nil {
		return nil, err
	}

	graphNodes, _ := parsed.Graph["nodes"].(map[string]any)

	var nodes []NodeInfo
	for id, raw := range graphNodes {
		node, ok := raw.(map[string]any)
		if !ok {
			continue
		}

		info := NodeInfo{
			ID:     id,
			Type:   strVal(node, "type"),
			Label:  parsed.Labels[id],
			Fields: make(map[string]any),
		}

		// Collect user-visible fields (skip internal ones)
		for key, val := range node {
			switch key {
			case "id", "type", "is_intermediate", "use_cache":
				continue
			}
			info.Fields[key] = val
		}

		nodes = append(nodes, info)
	}

	// Map iteration is random; keep the inspect page stable between reloads.
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Type != nodes[j].Type {
			return nodes[i].Type < nodes[j].Type
		}
		return nodes[i].ID < nodes[j].ID
	})

	return nodes, nil
}

// ListWorkflows returns the filenames of all workflow JSON files in the data dir.
func ListWorkflows(dataDir string) ([]string, error) {
	dir := filepath.Join(dataDir, "workflows")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

func strVal(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}
