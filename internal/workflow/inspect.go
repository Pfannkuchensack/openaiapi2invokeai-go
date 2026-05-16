package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// NodeInfo describes a node found during workflow inspection.
type NodeInfo struct {
	ID   string         `json:"id"`
	Type string         `json:"type"`
	Fields map[string]any `json:"fields"`
}

// InspectWorkflow loads a workflow and returns its nodes with their fields,
// useful for the admin UI to suggest field mappings.
func InspectWorkflow(dataDir, filename string) ([]NodeInfo, error) {
	wfPath := filepath.Join(dataDir, "workflows", filename)
	data, err := os.ReadFile(wfPath)
	if err != nil {
		return nil, fmt.Errorf("read workflow: %w", err)
	}

	var graph struct {
		Nodes map[string]map[string]any `json:"nodes"`
	}
	if err := json.Unmarshal(data, &graph); err != nil {
		return nil, fmt.Errorf("parse workflow: %w", err)
	}

	var nodes []NodeInfo
	for id, node := range graph.Nodes {
		info := NodeInfo{
			ID:     id,
			Type:   strVal(node, "type"),
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
