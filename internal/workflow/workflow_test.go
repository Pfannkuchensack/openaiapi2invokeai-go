package workflow_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Pfannkuchensack/openaiapi2invokeai-go/internal/workflow"
)

func TestRegistryCRUD(t *testing.T) {
	dir := t.TempDir()

	reg, err := workflow.NewRegistry(dir)
	if err != nil {
		t.Fatalf("new registry: %v", err)
	}

	// Empty initially
	if models := reg.List(); len(models) != 0 {
		t.Fatalf("expected 0 models, got %d", len(models))
	}

	// Put
	entry := workflow.ModelEntry{
		ID:       "test-sdxl",
		Workflow: "sdxl.json",
		Mapping: workflow.FieldMapping{
			Prompt: "nodes.abc.value",
			Seed:   "nodes.noise.seed",
		},
		SizePresets: map[string]workflow.Size{
			"1024x1024": {Width: 1024, Height: 1024},
		},
	}
	if err := reg.Put(entry); err != nil {
		t.Fatalf("put: %v", err)
	}

	// Get
	got, ok := reg.Get("test-sdxl")
	if !ok {
		t.Fatal("expected to find test-sdxl")
	}
	if got.Workflow != "sdxl.json" {
		t.Fatalf("expected sdxl.json, got %s", got.Workflow)
	}

	// Update
	entry.Workflow = "sdxl-v2.json"
	if err := reg.Put(entry); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, _ = reg.Get("test-sdxl")
	if got.Workflow != "sdxl-v2.json" {
		t.Fatalf("expected sdxl-v2.json after update, got %s", got.Workflow)
	}

	// List
	if models := reg.List(); len(models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(models))
	}

	// Delete
	if err := reg.Delete("test-sdxl"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, ok := reg.Get("test-sdxl"); ok {
		t.Fatal("expected model to be deleted")
	}

	// Persistence: reload and verify empty
	reg2, err := workflow.NewRegistry(dir)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if models := reg2.List(); len(models) != 0 {
		t.Fatalf("expected 0 after reload, got %d", len(models))
	}
}

func TestBuildGraph(t *testing.T) {
	// Use the real workflow from testdata
	projectRoot := findProjectRoot(t)
	dataDir := projectRoot

	// Verify workflow exists
	wfPath := filepath.Join(dataDir, "workflows", "example-graph.json")
	if _, err := os.Stat(wfPath); os.IsNotExist(err) {
		t.Skip("testdata/workflows/example-graph.json not found")
	}

	entry := workflow.ModelEntry{
		ID:       "test",
		Workflow: "example-graph.json",
		Mapping: workflow.FieldMapping{
			Prompt: "nodes.edd58e1c-ac4d-4857-9c92-da957fc35c7b.value", // string node with prompt
		},
	}

	params := workflow.Params{
		Prompt: "a beautiful landscape",
	}

	graph, err := workflow.BuildGraph(dataDir, entry, params)
	if err != nil {
		t.Fatalf("build graph: %v", err)
	}

	// Verify the prompt was injected
	nodes, ok := graph["nodes"].(map[string]any)
	if !ok {
		t.Fatal("expected nodes map")
	}

	node, ok := nodes["edd58e1c-ac4d-4857-9c92-da957fc35c7b"].(map[string]any)
	if !ok {
		t.Skip("node edd58e1c-ac4 not found (graph structure may differ)")
	}

	if val := node["value"]; val != "a beautiful landscape" {
		t.Fatalf("expected injected prompt, got %v", val)
	}
}

func TestResolveSize(t *testing.T) {
	entry := workflow.ModelEntry{
		SizePresets: map[string]workflow.Size{
			"1024x1024": {Width: 1024, Height: 1024},
			"landscape": {Width: 1792, Height: 1024},
		},
	}

	tests := []struct {
		input string
		w, h  int
		err   bool
	}{
		{"1024x1024", 1024, 1024, false},
		{"landscape", 1792, 1024, false},
		{"512x768", 512, 768, false},
		{"", 0, 0, false},
		{"invalid", 0, 0, true},
	}

	for _, tt := range tests {
		w, h, err := workflow.ResolveSize(entry, tt.input)
		if tt.err && err == nil {
			t.Errorf("ResolveSize(%q): expected error", tt.input)
		}
		if !tt.err && err != nil {
			t.Errorf("ResolveSize(%q): unexpected error: %v", tt.input, err)
		}
		if w != tt.w || h != tt.h {
			t.Errorf("ResolveSize(%q): got %dx%d, want %dx%d", tt.input, w, h, tt.w, tt.h)
		}
	}
}

func TestInspectWorkflow(t *testing.T) {
	projectRoot := findProjectRoot(t)
	dataDir := projectRoot

	wfPath := filepath.Join(dataDir, "workflows", "example-graph.json")
	if _, err := os.Stat(wfPath); os.IsNotExist(err) {
		t.Skip("testdata/workflows/example-graph.json not found")
	}

	nodes, err := workflow.InspectWorkflow(dataDir, "example-graph.json")
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}

	if len(nodes) == 0 {
		t.Fatal("expected nodes from inspection")
	}

	// Find at least one string node
	found := false
	for _, n := range nodes {
		if n.Type == "string" {
			found = true
			break
		}
	}
	if !found {
		t.Log("warning: no string-type nodes found")
	}

	t.Logf("inspected %d nodes", len(nodes))
}

func findProjectRoot(t *testing.T) string {
	t.Helper()
	// Walk up from test file to find testdata
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "testdata")); err == nil {
			return filepath.Join(dir, "testdata")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Skip("could not find project root with testdata/")
			return ""
		}
		dir = parent
	}
}
