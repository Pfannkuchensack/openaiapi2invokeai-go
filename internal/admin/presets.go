package admin

import "github.com/Pfannkuchensack/openaiapi2invokeai-go/internal/workflow"

// Preset defines a bundled workflow + model configuration.
type Preset struct {
	ID           string
	Label        string
	WorkflowFile string
	WorkflowJSON string
	Entry        workflow.ModelEntry
}

// Presets contains the built-in workflow templates.
var Presets = []Preset{
	{
		ID:           "sdxl",
		Label:        "SDXL (1024x1024)",
		WorkflowFile: "sdxl-txt2img.json",
		WorkflowJSON: sdxlWorkflowJSON,
		Entry: workflow.ModelEntry{
			ID:       "sdxl",
			Workflow: "sdxl-txt2img.json",
			Mapping: workflow.FieldMapping{
				Prompt:   "nodes.positive_prompt.prompt",
				Negative: "nodes.negative_prompt.prompt",
				Seed:     "nodes.noise.seed",
				Width:    "nodes.noise.width",
				Height:   "nodes.noise.height",
				Steps:    "nodes.denoise.steps",
				CFG:      "nodes.denoise.cfg_scale",
			},
			Defaults:    map[string]any{"steps": 20, "cfg": 7.5},
			SizePresets: map[string]workflow.Size{"1024x1024": {1024, 1024}, "1792x1024": {1792, 1024}, "1024x1792": {1024, 1792}},
		},
	},
	{
		ID:           "flux",
		Label:        "Flux (1024x1024)",
		WorkflowFile: "flux-txt2img.json",
		WorkflowJSON: fluxWorkflowJSON,
		Entry: workflow.ModelEntry{
			ID:       "flux",
			Workflow: "flux-txt2img.json",
			Mapping: workflow.FieldMapping{
				Prompt: "nodes.text_encoder.prompt",
				Seed:   "nodes.denoise.seed",
				Width:  "nodes.denoise.width",
				Height: "nodes.denoise.height",
				Steps:  "nodes.denoise.num_steps",
			},
			Defaults:    map[string]any{"steps": 20},
			SizePresets: map[string]workflow.Size{"1024x1024": {1024, 1024}, "1360x768": {1360, 768}, "768x1360": {768, 1360}},
		},
	},
	{
		ID:           "zimage",
		Label:        "Z-Image (1024x1024)",
		WorkflowFile: "zimage-txt2img.json",
		WorkflowJSON: zimageWorkflowJSON,
		Entry: workflow.ModelEntry{
			ID:       "zimage",
			Workflow: "zimage-txt2img.json",
			Mapping: workflow.FieldMapping{
				Prompt: "nodes.text_encoder.prompt",
				Seed:   "nodes.denoise.seed",
				Width:  "nodes.denoise.width",
				Height: "nodes.denoise.height",
				Steps:  "nodes.denoise.steps",
			},
			Defaults:    map[string]any{"steps": 8},
			SizePresets: map[string]workflow.Size{"1024x1024": {1024, 1024}, "1360x768": {1360, 768}, "768x1360": {768, 1360}},
		},
	},
	{
		ID:           "flux2klein",
		Label:        "Flux2 Klein (1024x1024)",
		WorkflowFile: "flux2klein-txt2img.json",
		WorkflowJSON: flux2kleinWorkflowJSON,
		Entry: workflow.ModelEntry{
			ID:       "flux2klein",
			Workflow: "flux2klein-txt2img.json",
			Mapping: workflow.FieldMapping{
				Prompt: "nodes.text_encoder.prompt",
				Seed:   "nodes.denoise.seed",
				Width:  "nodes.denoise.width",
				Height: "nodes.denoise.height",
				Steps:  "nodes.denoise.num_steps",
			},
			Defaults:    map[string]any{"steps": 4},
			SizePresets: map[string]workflow.Size{"1024x1024": {1024, 1024}, "1360x768": {1360, 768}, "768x1360": {768, 1360}},
		},
	},
	{
		ID:           "sd15",
		Label:        "SD 1.5 (512x512)",
		WorkflowFile: "sd15-txt2img.json",
		WorkflowJSON: sd15WorkflowJSON,
		Entry: workflow.ModelEntry{
			ID:       "sd15",
			Workflow: "sd15-txt2img.json",
			Mapping: workflow.FieldMapping{
				Prompt:   "nodes.positive_prompt.prompt",
				Negative: "nodes.negative_prompt.prompt",
				Seed:     "nodes.noise.seed",
				Width:    "nodes.noise.width",
				Height:   "nodes.noise.height",
				Steps:    "nodes.denoise.steps",
				CFG:      "nodes.denoise.cfg_scale",
			},
			Defaults:    map[string]any{"steps": 20, "cfg": 7.5},
			SizePresets: map[string]workflow.Size{"512x512": {512, 512}, "768x512": {768, 512}, "512x768": {512, 768}},
		},
	},
}

// PresetByID returns a preset by its ID.
func PresetByID(id string) (Preset, bool) {
	for _, p := range Presets {
		if p.ID == id {
			return p, true
		}
	}
	return Preset{}, false
}
