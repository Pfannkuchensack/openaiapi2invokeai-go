package admin

import "github.com/Pfannkuchensack/openaiapi2invokeai-go/internal/workflow"

// Preset defines a bundled workflow + model configuration.
type Preset struct {
	ID           string
	Label        string
	WorkflowFile string
	WorkflowJSON string
	// Optional img2img companion, installed alongside and wired up as the
	// entry's edit/variant workflow.
	EditWorkflowFile string
	EditWorkflowJSON string
	// SubModels are companion models the loader needs beside the main model.
	// Presets that declare them get a picker each in the setup form.
	SubModels []SubModel
	Entry     workflow.ModelEntry
}

// SubModel is a companion model a model_loader node needs, e.g. a text encoder
// or a VAE that is not bundled with the main model.
type SubModel struct {
	Field     string // model_loader field to fill, e.g. "qwen3_vl_encoder_model"
	ModelType string // InvokeAI model type to offer, e.g. "qwen3_vl_encoder"
	Base      string // base recorded in the written reference
	Label     string // shown in the setup form
	Hint      string // extra guidance, e.g. which encoder size to pick
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
	{
		ID:               "krea2",
		Label:            "Krea-2 (1024x1024, txt2img + img2img)",
		WorkflowFile:     "krea2-txt2img.json",
		WorkflowJSON:     krea2WorkflowJSON,
		EditWorkflowFile: "krea2-img2img.json",
		EditWorkflowJSON: krea2ImgWorkflowJSON,
		SubModels: []SubModel{
			{Field: "qwen3_vl_encoder_model", ModelType: "qwen3_vl_encoder", Base: "any",
				Label: "Qwen3-VL Text Encoder", Hint: "Leave empty for Diffusers models, which bundle their own."},
			{Field: "vae_model", ModelType: "vae", Base: "any",
				Label: "VAE", Hint: "Leave empty for Diffusers models, which bundle their own."},
		},
		Entry: workflow.ModelEntry{
			ID:              "krea2",
			Workflow:        "krea2-txt2img.json",
			EditWorkflow:    "krea2-img2img.json",
			VariantWorkflow: "krea2-img2img.json",
			Mapping: workflow.FieldMapping{
				Prompt:  "nodes.text_encoder.prompt",
				Seed:    "nodes.denoise.seed",
				Width:   "nodes.denoise.width",
				Height:  "nodes.denoise.height",
				Steps:   "nodes.denoise.steps",
				CFG:     "nodes.denoise.cfg_scale",
				Image:   "nodes.i2l.image",
				Denoise: "nodes.denoise.denoising_start",
			},
			Defaults:    map[string]any{"steps": 8},
			SizePresets: map[string]workflow.Size{"1024x1024": {1024, 1024}, "1360x768": {1360, 768}, "768x1360": {768, 1360}},
		},
	},
	{
		ID:               "anima",
		Label:            "Anima (1024x1024, txt2img + img2img)",
		WorkflowFile:     "anima-txt2img.json",
		WorkflowJSON:     animaWorkflowJSON,
		EditWorkflowFile: "anima-img2img.json",
		EditWorkflowJSON: animaImgWorkflowJSON,
		SubModels: []SubModel{
			{Field: "qwen3_encoder_model", ModelType: "qwen3_encoder", Base: "any",
				Label: "Qwen3 Text Encoder", Hint: "Must match the model size; anima-base-v1.0 needs Qwen3-0.6B."},
			{Field: "vae_model", ModelType: "vae", Base: "anima", Label: "VAE", Hint: "The Qwen Image VAE."},
		},
		Entry: workflow.ModelEntry{
			ID:              "anima",
			Workflow:        "anima-txt2img.json",
			EditWorkflow:    "anima-img2img.json",
			VariantWorkflow: "anima-img2img.json",
			Mapping: workflow.FieldMapping{
				Prompt:  "nodes.text_encoder.prompt",
				Seed:    "nodes.denoise.seed",
				Width:   "nodes.denoise.width",
				Height:  "nodes.denoise.height",
				Steps:   "nodes.denoise.steps",
				CFG:     "nodes.denoise.guidance_scale",
				Image:   "nodes.i2l.image",
				Denoise: "nodes.denoise.denoising_start",
			},
			Defaults:    map[string]any{"steps": 20, "cfg": 4.5},
			SizePresets: map[string]workflow.Size{"1024x1024": {1024, 1024}, "1360x768": {1360, 768}, "768x1360": {768, 1360}},
		},
	},
	{
		ID:               "qwenimage",
		Label:            "Qwen Image (1024x1024, txt2img + img2img)",
		WorkflowFile:     "qwenimage-txt2img.json",
		WorkflowJSON:     qwenImageWorkflowJSON,
		EditWorkflowFile: "qwenimage-img2img.json",
		EditWorkflowJSON: qwenImageImgWorkflowJSON,
		SubModels: []SubModel{
			{Field: "qwen_vl_encoder_model", ModelType: "qwen_vl_encoder", Base: "any",
				Label: "Qwen2.5-VL Text Encoder", Hint: "Required unless the main model is a Diffusers build that bundles one."},
			{Field: "vae_model", ModelType: "vae", Base: "anima", Label: "VAE", Hint: "The Qwen Image VAE."},
		},
		Entry: workflow.ModelEntry{
			ID:              "qwenimage",
			Workflow:        "qwenimage-txt2img.json",
			EditWorkflow:    "qwenimage-img2img.json",
			VariantWorkflow: "qwenimage-img2img.json",
			Mapping: workflow.FieldMapping{
				Prompt:  "nodes.text_encoder.prompt",
				Seed:    "nodes.denoise.seed",
				Width:   "nodes.denoise.width",
				Height:  "nodes.denoise.height",
				Steps:   "nodes.denoise.steps",
				CFG:     "nodes.denoise.cfg_scale",
				Image:   "nodes.i2l.image",
				Denoise: "nodes.denoise.denoising_start",
			},
			Defaults:    map[string]any{"steps": 20, "cfg": 4.0},
			SizePresets: map[string]workflow.Size{"1024x1024": {1024, 1024}, "1360x768": {1360, 768}, "768x1360": {768, 1360}},
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
