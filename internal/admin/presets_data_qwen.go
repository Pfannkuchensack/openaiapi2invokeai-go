package admin

// Workflow graphs for the architectures that encode text with a Qwen model:
// Krea-2, Anima and Qwen Image. They share the qwen_image VAE nodes, so Krea-2
// decodes through qwen_image_l2i rather than the plain l2i, which rejects its
// VAE with an AssertionError.
//
// Krea-2 and Anima take positive_conditioning from a collect node; Qwen Image
// wires it straight from the text encoder. This mirrors InvokeAI's own
// buildKrea2Graph / buildAnimaGraph / buildQwenImageGraph.

const krea2WorkflowJSON = `{
  "id": "krea2-txt2img",
  "nodes": {
    "model_loader": {
      "id": "model_loader",
      "type": "krea2_model_loader",
      "is_intermediate": true,
      "use_cache": true,
      "model": {"key": "REPLACE_WITH_MODEL_KEY", "name": "your-krea2-model", "base": "krea-2", "type": "main"}
    },
    "text_encoder": {
      "id": "text_encoder",
      "type": "krea2_text_encoder",
      "is_intermediate": true,
      "use_cache": true,
      "prompt": "a beautiful landscape"
    },
    "pos_collect": {"id": "pos_collect", "type": "collect", "is_intermediate": true, "use_cache": true},
    "denoise": {
      "id": "denoise",
      "type": "krea2_denoise",
      "is_intermediate": true,
      "use_cache": false,
      "seed": 0,
      "width": 1024,
      "height": 1024,
      "steps": 8,
      "cfg_scale": 1.0,
      "denoising_start": 0.0,
      "denoising_end": 1.0
    },
    "decode": {"id": "decode", "type": "qwen_image_l2i", "is_intermediate": false, "use_cache": false}
  },
  "edges": [
    {"source": {"node_id": "model_loader", "field": "qwen3_vl_encoder"}, "destination": {"node_id": "text_encoder", "field": "qwen3_vl_encoder"}},
    {"source": {"node_id": "model_loader", "field": "transformer"}, "destination": {"node_id": "denoise", "field": "transformer"}},
    {"source": {"node_id": "text_encoder", "field": "conditioning"}, "destination": {"node_id": "pos_collect", "field": "item"}},
    {"source": {"node_id": "pos_collect", "field": "collection"}, "destination": {"node_id": "denoise", "field": "positive_conditioning"}},
    {"source": {"node_id": "denoise", "field": "latents"}, "destination": {"node_id": "decode", "field": "latents"}},
    {"source": {"node_id": "model_loader", "field": "vae"}, "destination": {"node_id": "decode", "field": "vae"}}
  ]
}`

const krea2ImgWorkflowJSON = `{
  "id": "krea2-img2img",
  "nodes": {
    "model_loader": {
      "id": "model_loader",
      "type": "krea2_model_loader",
      "is_intermediate": true,
      "use_cache": true,
      "model": {"key": "REPLACE_WITH_MODEL_KEY", "name": "your-krea2-model", "base": "krea-2", "type": "main"}
    },
    "text_encoder": {
      "id": "text_encoder",
      "type": "krea2_text_encoder",
      "is_intermediate": true,
      "use_cache": true,
      "prompt": "a beautiful landscape"
    },
    "pos_collect": {"id": "pos_collect", "type": "collect", "is_intermediate": true, "use_cache": true},
    "i2l": {"id": "i2l", "type": "qwen_image_i2l", "is_intermediate": true, "use_cache": true},
    "denoise": {
      "id": "denoise",
      "type": "krea2_denoise",
      "is_intermediate": true,
      "use_cache": false,
      "seed": 0,
      "width": 1024,
      "height": 1024,
      "steps": 8,
      "cfg_scale": 1.0,
      "denoising_start": 0.35,
      "denoising_end": 1.0
    },
    "decode": {"id": "decode", "type": "qwen_image_l2i", "is_intermediate": false, "use_cache": false}
  },
  "edges": [
    {"source": {"node_id": "model_loader", "field": "qwen3_vl_encoder"}, "destination": {"node_id": "text_encoder", "field": "qwen3_vl_encoder"}},
    {"source": {"node_id": "model_loader", "field": "transformer"}, "destination": {"node_id": "denoise", "field": "transformer"}},
    {"source": {"node_id": "model_loader", "field": "vae"}, "destination": {"node_id": "i2l", "field": "vae"}},
    {"source": {"node_id": "text_encoder", "field": "conditioning"}, "destination": {"node_id": "pos_collect", "field": "item"}},
    {"source": {"node_id": "pos_collect", "field": "collection"}, "destination": {"node_id": "denoise", "field": "positive_conditioning"}},
    {"source": {"node_id": "i2l", "field": "latents"}, "destination": {"node_id": "denoise", "field": "latents"}},
    {"source": {"node_id": "denoise", "field": "latents"}, "destination": {"node_id": "decode", "field": "latents"}},
    {"source": {"node_id": "model_loader", "field": "vae"}, "destination": {"node_id": "decode", "field": "vae"}}
  ]
}`

// The Anima encoder size has to match the model: anima-base-v1.0 expects
// Qwen3-0.6B, a 4B encoder fails with a shape mismatch during denoising.

const animaWorkflowJSON = `{
  "id": "anima-txt2img",
  "nodes": {
    "model_loader": {
      "id": "model_loader",
      "type": "anima_model_loader",
      "is_intermediate": true,
      "use_cache": true,
      "model": {"key": "REPLACE_WITH_MODEL_KEY", "name": "your-anima-model", "base": "anima", "type": "main"}
    },
    "text_encoder": {
      "id": "text_encoder",
      "type": "anima_text_encoder",
      "is_intermediate": true,
      "use_cache": true,
      "prompt": "a beautiful landscape"
    },
    "pos_collect": {"id": "pos_collect", "type": "collect", "is_intermediate": true, "use_cache": true},
    "denoise": {
      "id": "denoise",
      "type": "anima_denoise",
      "is_intermediate": true,
      "use_cache": false,
      "seed": 0,
      "width": 1024,
      "height": 1024,
      "steps": 20,
      "guidance_scale": 4.5,
      "scheduler": "euler",
      "add_noise": true,
      "denoising_start": 0.0,
      "denoising_end": 1.0
    },
    "decode": {"id": "decode", "type": "anima_l2i", "is_intermediate": false, "use_cache": false}
  },
  "edges": [
    {"source": {"node_id": "model_loader", "field": "qwen3_encoder"}, "destination": {"node_id": "text_encoder", "field": "qwen3_encoder"}},
    {"source": {"node_id": "model_loader", "field": "transformer"}, "destination": {"node_id": "denoise", "field": "transformer"}},
    {"source": {"node_id": "text_encoder", "field": "conditioning"}, "destination": {"node_id": "pos_collect", "field": "item"}},
    {"source": {"node_id": "pos_collect", "field": "collection"}, "destination": {"node_id": "denoise", "field": "positive_conditioning"}},
    {"source": {"node_id": "denoise", "field": "latents"}, "destination": {"node_id": "decode", "field": "latents"}},
    {"source": {"node_id": "model_loader", "field": "vae"}, "destination": {"node_id": "decode", "field": "vae"}}
  ]
}`

const animaImgWorkflowJSON = `{
  "id": "anima-img2img",
  "nodes": {
    "model_loader": {
      "id": "model_loader",
      "type": "anima_model_loader",
      "is_intermediate": true,
      "use_cache": true,
      "model": {"key": "REPLACE_WITH_MODEL_KEY", "name": "your-anima-model", "base": "anima", "type": "main"}
    },
    "text_encoder": {
      "id": "text_encoder",
      "type": "anima_text_encoder",
      "is_intermediate": true,
      "use_cache": true,
      "prompt": "a beautiful landscape"
    },
    "pos_collect": {"id": "pos_collect", "type": "collect", "is_intermediate": true, "use_cache": true},
    "i2l": {"id": "i2l", "type": "anima_i2l", "is_intermediate": true, "use_cache": true},
    "denoise": {
      "id": "denoise",
      "type": "anima_denoise",
      "is_intermediate": true,
      "use_cache": false,
      "seed": 0,
      "width": 1024,
      "height": 1024,
      "steps": 20,
      "guidance_scale": 4.5,
      "scheduler": "euler",
      "add_noise": true,
      "denoising_start": 0.35,
      "denoising_end": 1.0
    },
    "decode": {"id": "decode", "type": "anima_l2i", "is_intermediate": false, "use_cache": false}
  },
  "edges": [
    {"source": {"node_id": "model_loader", "field": "qwen3_encoder"}, "destination": {"node_id": "text_encoder", "field": "qwen3_encoder"}},
    {"source": {"node_id": "model_loader", "field": "transformer"}, "destination": {"node_id": "denoise", "field": "transformer"}},
    {"source": {"node_id": "model_loader", "field": "vae"}, "destination": {"node_id": "i2l", "field": "vae"}},
    {"source": {"node_id": "text_encoder", "field": "conditioning"}, "destination": {"node_id": "pos_collect", "field": "item"}},
    {"source": {"node_id": "pos_collect", "field": "collection"}, "destination": {"node_id": "denoise", "field": "positive_conditioning"}},
    {"source": {"node_id": "i2l", "field": "latents"}, "destination": {"node_id": "denoise", "field": "latents"}},
    {"source": {"node_id": "denoise", "field": "latents"}, "destination": {"node_id": "decode", "field": "latents"}},
    {"source": {"node_id": "model_loader", "field": "vae"}, "destination": {"node_id": "decode", "field": "vae"}}
  ]
}`

const qwenImageWorkflowJSON = `{
  "id": "qwenimage-txt2img",
  "nodes": {
    "model_loader": {
      "id": "model_loader",
      "type": "qwen_image_model_loader",
      "is_intermediate": true,
      "use_cache": true,
      "model": {"key": "REPLACE_WITH_MODEL_KEY", "name": "your-qwen-image-model", "base": "qwen-image", "type": "main"}
    },
    "text_encoder": {
      "id": "text_encoder",
      "type": "qwen_image_text_encoder",
      "is_intermediate": true,
      "use_cache": true,
      "prompt": "a beautiful landscape"
    },
    "denoise": {
      "id": "denoise",
      "type": "qwen_image_denoise",
      "is_intermediate": true,
      "use_cache": false,
      "seed": 0,
      "width": 1024,
      "height": 1024,
      "steps": 20,
      "cfg_scale": 4.0,
      "denoising_start": 0.0,
      "denoising_end": 1.0
    },
    "decode": {"id": "decode", "type": "qwen_image_l2i", "is_intermediate": false, "use_cache": false}
  },
  "edges": [
    {"source": {"node_id": "model_loader", "field": "qwen_vl_encoder"}, "destination": {"node_id": "text_encoder", "field": "qwen_vl_encoder"}},
    {"source": {"node_id": "model_loader", "field": "transformer"}, "destination": {"node_id": "denoise", "field": "transformer"}},
    {"source": {"node_id": "text_encoder", "field": "conditioning"}, "destination": {"node_id": "denoise", "field": "positive_conditioning"}},
    {"source": {"node_id": "denoise", "field": "latents"}, "destination": {"node_id": "decode", "field": "latents"}},
    {"source": {"node_id": "model_loader", "field": "vae"}, "destination": {"node_id": "decode", "field": "vae"}}
  ]
}`

const qwenImageImgWorkflowJSON = `{
  "id": "qwenimage-img2img",
  "nodes": {
    "model_loader": {
      "id": "model_loader",
      "type": "qwen_image_model_loader",
      "is_intermediate": true,
      "use_cache": true,
      "model": {"key": "REPLACE_WITH_MODEL_KEY", "name": "your-qwen-image-model", "base": "qwen-image", "type": "main"}
    },
    "text_encoder": {
      "id": "text_encoder",
      "type": "qwen_image_text_encoder",
      "is_intermediate": true,
      "use_cache": true,
      "prompt": "a beautiful landscape"
    },
    "i2l": {"id": "i2l", "type": "qwen_image_i2l", "is_intermediate": true, "use_cache": true},
    "denoise": {
      "id": "denoise",
      "type": "qwen_image_denoise",
      "is_intermediate": true,
      "use_cache": false,
      "seed": 0,
      "width": 1024,
      "height": 1024,
      "steps": 20,
      "cfg_scale": 4.0,
      "denoising_start": 0.35,
      "denoising_end": 1.0
    },
    "decode": {"id": "decode", "type": "qwen_image_l2i", "is_intermediate": false, "use_cache": false}
  },
  "edges": [
    {"source": {"node_id": "model_loader", "field": "qwen_vl_encoder"}, "destination": {"node_id": "text_encoder", "field": "qwen_vl_encoder"}},
    {"source": {"node_id": "model_loader", "field": "transformer"}, "destination": {"node_id": "denoise", "field": "transformer"}},
    {"source": {"node_id": "model_loader", "field": "vae"}, "destination": {"node_id": "i2l", "field": "vae"}},
    {"source": {"node_id": "text_encoder", "field": "conditioning"}, "destination": {"node_id": "denoise", "field": "positive_conditioning"}},
    {"source": {"node_id": "i2l", "field": "latents"}, "destination": {"node_id": "denoise", "field": "latents"}},
    {"source": {"node_id": "denoise", "field": "latents"}, "destination": {"node_id": "decode", "field": "latents"}},
    {"source": {"node_id": "model_loader", "field": "vae"}, "destination": {"node_id": "decode", "field": "vae"}}
  ]
}`
