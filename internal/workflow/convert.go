package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Format identifies which JSON shape a workflow file uses.
type Format string

const (
	// FormatGraph is the API graph format that InvokeAI's enqueue_batch endpoint
	// expects: "nodes" is an object keyed by node ID, "edges" carry
	// source/destination objects.
	FormatGraph Format = "graph"
	// FormatEditor is what the InvokeAI workflow editor writes when you use
	// "Download Workflow" / save to a file: "nodes" is an array of react-flow
	// nodes whose invocation fields live under data.inputs.
	FormatEditor Format = "editor"
)

// Parsed is a workflow file normalized to the API graph format.
type Parsed struct {
	Format Format
	Graph  map[string]any
	// Labels maps node ID → the user-assigned label from the editor. Empty for
	// files that were already in graph format.
	Labels map[string]string
}

// LoadWorkflow reads a workflow file from <dataDir>/workflows and normalizes it
// to the API graph format.
func LoadWorkflow(dataDir, filename string) (*Parsed, error) {
	data, err := os.ReadFile(filepath.Join(dataDir, "workflows", filename))
	if err != nil {
		return nil, fmt.Errorf("read workflow %s: %w", filename, err)
	}
	return ParseWorkflow(data)
}

// ParseWorkflow accepts either an API graph or an editor-exported workflow and
// returns the graph form of it. Editor exports are converted the same way the
// InvokeAI frontend builds a graph before enqueueing.
func ParseWorkflow(data []byte) (*Parsed, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parse workflow: %w", err)
	}

	nodesRaw, ok := root["nodes"]
	if !ok {
		return nil, fmt.Errorf(`parse workflow: no "nodes" key — this is not an InvokeAI workflow or graph`)
	}

	switch firstToken(nodesRaw) {
	case '{':
		var graph map[string]any
		if err := json.Unmarshal(data, &graph); err != nil {
			return nil, fmt.Errorf("parse workflow: %w", err)
		}
		return &Parsed{Format: FormatGraph, Graph: graph, Labels: map[string]string{}}, nil
	case '[':
		return convertEditorWorkflow(nodesRaw, root["edges"])
	default:
		return nil, fmt.Errorf(`parse workflow: "nodes" is neither an object nor an array`)
	}
}

// editorNode is one react-flow node from an editor export.
type editorNode struct {
	ID   string `json:"id"`
	Type string `json:"type"` // "invocation", "notes", "current_image"
	Data struct {
		ID             string                     `json:"id"`
		Type           string                     `json:"type"` // the invocation type
		Label          string                     `json:"label"`
		IsIntermediate *bool                      `json:"isIntermediate"`
		UseCache       *bool                      `json:"useCache"`
		Inputs         map[string]editorInputSpec `json:"inputs"`
	} `json:"data"`
}

// editorInputSpec is one entry of data.inputs. Fields that are fed by an edge
// carry no "value" key at all.
type editorInputSpec struct {
	Value json.RawMessage `json:"value"`
}

type editorEdge struct {
	Type         string `json:"type"` // "default" or "collapsed" (UI-only)
	Source       string `json:"source"`
	Target       string `json:"target"`
	SourceHandle string `json:"sourceHandle"`
	TargetHandle string `json:"targetHandle"`
}

func convertEditorWorkflow(nodesRaw, edgesRaw json.RawMessage) (*Parsed, error) {
	var editorNodes []editorNode
	if err := json.Unmarshal(nodesRaw, &editorNodes); err != nil {
		return nil, fmt.Errorf("parse workflow nodes: %w", err)
	}

	nodes := make(map[string]any, len(editorNodes))
	labels := make(map[string]string)

	for _, n := range editorNodes {
		// Notes and the "current image" placeholder are editor decoration.
		if n.Type == "notes" || n.Type == "current_image" || n.Data.Type == "" || n.Data.Type == "notes" {
			continue
		}

		id := n.ID
		if id == "" {
			id = n.Data.ID
		}
		if id == "" {
			continue
		}

		node := make(map[string]any, len(n.Data.Inputs)+4)
		for name, in := range n.Data.Inputs {
			// No "value" means the field is supplied by an edge; sending null
			// would fail InvokeAI's validation.
			if len(in.Value) == 0 {
				continue
			}
			var v any
			if err := json.Unmarshal(in.Value, &v); err != nil {
				return nil, fmt.Errorf("parse workflow node %s field %s: %w", id, name, err)
			}
			node[name] = v
		}
		// Set last so an input never shadows the invocation identity.
		node["id"] = id
		node["type"] = n.Data.Type
		if n.Data.IsIntermediate != nil {
			node["is_intermediate"] = *n.Data.IsIntermediate
		}
		if n.Data.UseCache != nil {
			node["use_cache"] = *n.Data.UseCache
		}

		nodes[id] = node
		if n.Data.Label != "" {
			labels[id] = n.Data.Label
		}
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("parse workflow: no invocation nodes found")
	}

	var editorEdges []editorEdge
	if len(edgesRaw) > 0 {
		if err := json.Unmarshal(edgesRaw, &editorEdges); err != nil {
			return nil, fmt.Errorf("parse workflow edges: %w", err)
		}
	}

	edges := make([]any, 0, len(editorEdges))
	for _, e := range editorEdges {
		// "collapsed" edges are drawn between collapsed node groups and carry
		// no field handles.
		if e.Type == "collapsed" || e.SourceHandle == "" || e.TargetHandle == "" {
			continue
		}
		if _, ok := nodes[e.Source]; !ok {
			continue
		}
		if _, ok := nodes[e.Target]; !ok {
			continue
		}
		edges = append(edges, map[string]any{
			"source":      map[string]any{"node_id": e.Source, "field": e.SourceHandle},
			"destination": map[string]any{"node_id": e.Target, "field": e.TargetHandle},
		})
	}

	return &Parsed{
		Format: FormatEditor,
		Graph:  map[string]any{"nodes": nodes, "edges": edges},
		Labels: labels,
	}, nil
}

// firstToken returns the first non-whitespace byte of a raw JSON value.
func firstToken(raw json.RawMessage) byte {
	for _, b := range raw {
		switch b {
		case ' ', '\t', '\r', '\n':
			continue
		default:
			return b
		}
	}
	return 0
}
