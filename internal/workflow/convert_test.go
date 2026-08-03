package workflow_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Pfannkuchensack/openaiapi2invokeai-go/internal/workflow"
)

// A trimmed-down export from the InvokeAI workflow editor: "nodes" is an array
// of react-flow nodes, invocation fields live under data.inputs, and there is a
// notes node plus a collapsed edge that must be dropped.
const editorWorkflowJSON = `{
  "name": "Flux Text to Image",
  "meta": {"version": "3.0.0", "category": "user"},
  "nodes": [
    {
      "id": "note-1",
      "type": "notes",
      "position": {"x": 0, "y": 0},
      "data": {"id": "note-1", "type": "notes", "notes": "hello", "isOpen": true}
    },
    {
      "id": "prompt-node",
      "type": "invocation",
      "position": {"x": 100, "y": 0},
      "data": {
        "id": "prompt-node",
        "type": "flux_text_encoder",
        "label": "Positive Prompt",
        "isIntermediate": true,
        "useCache": true,
        "inputs": {
          "clip": {"name": "clip", "label": ""},
          "t5_encoder": {"name": "t5_encoder", "label": ""},
          "prompt": {"name": "prompt", "label": "", "value": "a cat"},
          "mask": {"name": "mask", "label": "", "value": null}
        }
      }
    },
    {
      "id": "noise-node",
      "type": "invocation",
      "position": {"x": 200, "y": 0},
      "data": {
        "id": "noise-node",
        "type": "noise",
        "label": "",
        "isIntermediate": true,
        "useCache": false,
        "inputs": {
          "seed": {"name": "seed", "label": "", "value": 12345},
          "width": {"name": "width", "label": "", "value": 1024},
          "height": {"name": "height", "label": "", "value": 1024}
        }
      }
    },
    {
      "id": "output-node",
      "type": "invocation",
      "position": {"x": 300, "y": 0},
      "data": {
        "id": "output-node",
        "type": "flux_vae_decode",
        "label": "",
        "isIntermediate": false,
        "useCache": false,
        "inputs": {
          "latents": {"name": "latents", "label": ""},
          "vae": {"name": "vae", "label": ""}
        }
      }
    }
  ],
  "edges": [
    {
      "id": "edge-1",
      "type": "default",
      "source": "prompt-node",
      "target": "output-node",
      "sourceHandle": "conditioning",
      "targetHandle": "latents"
    },
    {
      "id": "edge-collapsed",
      "type": "collapsed",
      "source": "noise-node",
      "target": "output-node"
    },
    {
      "id": "edge-dangling",
      "type": "default",
      "source": "note-1",
      "target": "output-node",
      "sourceHandle": "x",
      "targetHandle": "y"
    }
  ]
}`

const graphWorkflowJSON = `{
  "id": "g",
  "nodes": {"n1": {"id": "n1", "type": "noise", "seed": 1}},
  "edges": []
}`

func TestParseWorkflowEditorFormat(t *testing.T) {
	parsed, err := workflow.ParseWorkflow([]byte(editorWorkflowJSON))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.Format != workflow.FormatEditor {
		t.Fatalf("format: got %q, want %q", parsed.Format, workflow.FormatEditor)
	}

	nodes, ok := parsed.Graph["nodes"].(map[string]any)
	if !ok {
		t.Fatal("nodes is not a map")
	}
	if len(nodes) != 3 {
		t.Fatalf("expected 3 invocation nodes (notes dropped), got %d: %v", len(nodes), nodes)
	}
	if _, ok := nodes["note-1"]; ok {
		t.Error("notes node should not be in the graph")
	}

	prompt, _ := nodes["prompt-node"].(map[string]any)
	if prompt["type"] != "flux_text_encoder" {
		t.Errorf("type: got %v", prompt["type"])
	}
	if prompt["prompt"] != "a cat" {
		t.Errorf("prompt: got %v", prompt["prompt"])
	}
	if prompt["is_intermediate"] != true || prompt["use_cache"] != true {
		t.Errorf("is_intermediate/use_cache not carried over: %v", prompt)
	}
	// Connection-only inputs carry no "value" key and must be omitted.
	if _, ok := prompt["clip"]; ok {
		t.Error("clip is edge-fed and should not be a literal field")
	}
	// An explicit null is a real value and is kept, matching the editor.
	if v, ok := prompt["mask"]; !ok || v != nil {
		t.Errorf("mask: got %v (present=%v), want explicit nil", v, ok)
	}

	if parsed.Labels["prompt-node"] != "Positive Prompt" {
		t.Errorf("label: got %q", parsed.Labels["prompt-node"])
	}

	edges, ok := parsed.Graph["edges"].([]any)
	if !ok {
		t.Fatal("edges is not a slice")
	}
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge (collapsed + dangling dropped), got %d: %v", len(edges), edges)
	}
	e := edges[0].(map[string]any)
	src := e["source"].(map[string]any)
	dst := e["destination"].(map[string]any)
	if src["node_id"] != "prompt-node" || src["field"] != "conditioning" {
		t.Errorf("source: got %v", src)
	}
	if dst["node_id"] != "output-node" || dst["field"] != "latents" {
		t.Errorf("destination: got %v", dst)
	}
}

func TestParseWorkflowGraphFormatUnchanged(t *testing.T) {
	parsed, err := workflow.ParseWorkflow([]byte(graphWorkflowJSON))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.Format != workflow.FormatGraph {
		t.Fatalf("format: got %q, want %q", parsed.Format, workflow.FormatGraph)
	}
	if parsed.Graph["id"] != "g" {
		t.Errorf("graph passed through lossily: %v", parsed.Graph)
	}
	nodes := parsed.Graph["nodes"].(map[string]any)
	if len(nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(nodes))
	}
}

func TestParseWorkflowErrors(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{"not json", `not json at all`},
		{"no nodes key", `{"name": "x"}`},
		{"nodes wrong type", `{"nodes": "abc"}`},
		{"editor with no invocations", `{"nodes": [{"id":"n","type":"notes","data":{"id":"n","type":"notes"}}]}`},
	}
	for _, tt := range tests {
		if _, err := workflow.ParseWorkflow([]byte(tt.json)); err == nil {
			t.Errorf("%s: expected an error", tt.name)
		}
	}
}

// An editor export must be usable end-to-end: inspect finds the nodes, and the
// mapping paths shown there resolve during graph building.
func TestEditorWorkflowBuildAndInspect(t *testing.T) {
	dir := t.TempDir()
	wfDir := filepath.Join(dir, "workflows")
	if err := os.MkdirAll(wfDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wfDir, "flux.json"), []byte(editorWorkflowJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	nodes, err := workflow.InspectWorkflow(dir, "flux.json")
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if len(nodes) != 3 {
		t.Fatalf("inspect: expected 3 nodes, got %d", len(nodes))
	}

	entry := workflow.ModelEntry{
		ID:       "flux",
		Workflow: "flux.json",
		Mapping: workflow.FieldMapping{
			Prompt: "nodes.prompt-node.prompt",
			Seed:   "nodes.noise-node.seed",
			Width:  "nodes.noise-node.width",
		},
	}
	graph, err := workflow.BuildGraph(dir, entry, workflow.Params{
		Prompt: "a dog on a skateboard",
		Width:  768,
		Seed:   42,
	})
	if err != nil {
		t.Fatalf("build graph: %v", err)
	}

	gn := graph["nodes"].(map[string]any)
	if got := gn["prompt-node"].(map[string]any)["prompt"]; got != "a dog on a skateboard" {
		t.Errorf("prompt not injected: %v", got)
	}
	noise := gn["noise-node"].(map[string]any)
	if noise["width"] != 768 {
		t.Errorf("width not injected: %v", noise["width"])
	}
	if noise["seed"] != int64(42) {
		t.Errorf("seed not injected: %v", noise["seed"])
	}
}
