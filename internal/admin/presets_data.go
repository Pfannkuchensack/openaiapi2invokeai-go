package admin

const sdxlWorkflowJSON = `{
  "id": "sdxl-txt2img",
  "nodes": {
    "model_loader": {
      "id": "model_loader",
      "type": "sdxl_model_loader",
      "is_intermediate": true,
      "use_cache": true,
      "model": {"key": "REPLACE_WITH_MODEL_KEY", "name": "your-sdxl-model", "base": "sdxl", "type": "main"}
    },
    "positive_prompt": {
      "id": "positive_prompt",
      "type": "sdxl_compel_prompt",
      "is_intermediate": true,
      "use_cache": true,
      "prompt": "a beautiful landscape",
      "style": ""
    },
    "negative_prompt": {
      "id": "negative_prompt",
      "type": "sdxl_compel_prompt",
      "is_intermediate": true,
      "use_cache": true,
      "prompt": "bad quality, worst quality, lowres",
      "style": ""
    },
    "noise": {
      "id": "noise",
      "type": "noise",
      "is_intermediate": true,
      "use_cache": false,
      "seed": 0,
      "width": 1024,
      "height": 1024,
      "use_cpu": true
    },
    "denoise": {
      "id": "denoise",
      "type": "denoise_latents",
      "is_intermediate": true,
      "use_cache": false,
      "steps": 20,
      "cfg_scale": 7.5,
      "denoising_start": 0.0,
      "denoising_end": 1.0,
      "scheduler": "dpmpp_2m"
    },
    "decode": {
      "id": "decode",
      "type": "l2i",
      "is_intermediate": false,
      "use_cache": false
    }
  },
  "edges": [
    {"source": {"node_id": "model_loader", "field": "unet"}, "destination": {"node_id": "denoise", "field": "unet"}},
    {"source": {"node_id": "model_loader", "field": "clip"}, "destination": {"node_id": "positive_prompt", "field": "clip"}},
    {"source": {"node_id": "model_loader", "field": "clip"}, "destination": {"node_id": "negative_prompt", "field": "clip"}},
    {"source": {"node_id": "model_loader", "field": "clip2"}, "destination": {"node_id": "positive_prompt", "field": "clip2"}},
    {"source": {"node_id": "model_loader", "field": "clip2"}, "destination": {"node_id": "negative_prompt", "field": "clip2"}},
    {"source": {"node_id": "positive_prompt", "field": "conditioning"}, "destination": {"node_id": "denoise", "field": "positive_conditioning"}},
    {"source": {"node_id": "negative_prompt", "field": "conditioning"}, "destination": {"node_id": "denoise", "field": "negative_conditioning"}},
    {"source": {"node_id": "noise", "field": "noise"}, "destination": {"node_id": "denoise", "field": "noise"}},
    {"source": {"node_id": "denoise", "field": "latents"}, "destination": {"node_id": "decode", "field": "latents"}},
    {"source": {"node_id": "model_loader", "field": "vae"}, "destination": {"node_id": "decode", "field": "vae"}}
  ]
}`

const fluxWorkflowJSON = `{
  "id": "flux-txt2img",
  "nodes": {
    "model_loader": {
      "id": "model_loader",
      "type": "flux_model_loader",
      "is_intermediate": true,
      "use_cache": true,
      "model": {"key": "REPLACE_WITH_MODEL_KEY", "name": "your-flux-model", "base": "flux", "type": "main"}
    },
    "text_encoder": {
      "id": "text_encoder",
      "type": "flux_text_encoder",
      "is_intermediate": true,
      "use_cache": true,
      "prompt": "a beautiful landscape",
      "t5_max_seq_len": 256
    },
    "denoise": {
      "id": "denoise",
      "type": "flux_denoise",
      "is_intermediate": true,
      "use_cache": false,
      "width": 1024,
      "height": 1024,
      "num_steps": 20,
      "guidance": 4.0,
      "seed": 0,
      "scheduler": "euler",
      "denoising_start": 0.0,
      "denoising_end": 1.0
    },
    "decode": {
      "id": "decode",
      "type": "flux_vae_decode",
      "is_intermediate": false,
      "use_cache": false
    }
  },
  "edges": [
    {"source": {"node_id": "model_loader", "field": "transformer"}, "destination": {"node_id": "denoise", "field": "transformer"}},
    {"source": {"node_id": "model_loader", "field": "clip"}, "destination": {"node_id": "text_encoder", "field": "clip"}},
    {"source": {"node_id": "model_loader", "field": "t5_encoder"}, "destination": {"node_id": "text_encoder", "field": "t5_encoder"}},
    {"source": {"node_id": "text_encoder", "field": "conditioning"}, "destination": {"node_id": "denoise", "field": "positive_text_conditioning"}},
    {"source": {"node_id": "denoise", "field": "latents"}, "destination": {"node_id": "decode", "field": "latents"}},
    {"source": {"node_id": "model_loader", "field": "vae"}, "destination": {"node_id": "decode", "field": "vae"}}
  ]
}`

const zimageWorkflowJSON = `{
  "id": "zimage-txt2img",
  "nodes": {
    "model_loader": {
      "id": "model_loader",
      "type": "z_image_model_loader",
      "is_intermediate": true,
      "use_cache": true,
      "model": {"key": "REPLACE_WITH_MODEL_KEY", "name": "your-zimage-model", "base": "z-image", "type": "main"}
    },
    "text_encoder": {
      "id": "text_encoder",
      "type": "z_image_text_encoder",
      "is_intermediate": true,
      "use_cache": true,
      "prompt": "a beautiful landscape"
    },
    "denoise": {
      "id": "denoise",
      "type": "z_image_denoise",
      "is_intermediate": true,
      "use_cache": false,
      "width": 1024,
      "height": 1024,
      "steps": 8,
      "guidance_scale": 1.0,
      "seed": 0,
      "scheduler": "euler",
      "denoising_start": 0.0,
      "denoising_end": 1.0
    },
    "decode": {
      "id": "decode",
      "type": "z_image_l2i",
      "is_intermediate": false,
      "use_cache": false
    }
  },
  "edges": [
    {"source": {"node_id": "model_loader", "field": "transformer"}, "destination": {"node_id": "denoise", "field": "transformer"}},
    {"source": {"node_id": "model_loader", "field": "qwen3_encoder"}, "destination": {"node_id": "text_encoder", "field": "qwen3_encoder"}},
    {"source": {"node_id": "text_encoder", "field": "conditioning"}, "destination": {"node_id": "denoise", "field": "positive_conditioning"}},
    {"source": {"node_id": "denoise", "field": "latents"}, "destination": {"node_id": "decode", "field": "latents"}},
    {"source": {"node_id": "model_loader", "field": "vae"}, "destination": {"node_id": "decode", "field": "vae"}},
    {"source": {"node_id": "model_loader", "field": "vae"}, "destination": {"node_id": "denoise", "field": "vae"}}
  ]
}`

const flux2kleinWorkflowJSON = `{
  "id": "flux2klein-txt2img",
  "nodes": {
    "model_loader": {
      "id": "model_loader",
      "type": "flux2_klein_model_loader",
      "is_intermediate": true,
      "use_cache": true,
      "model": {"key": "REPLACE_WITH_MODEL_KEY", "name": "your-flux2klein-model", "base": "flux", "type": "main"},
      "max_seq_len": 512
    },
    "text_encoder": {
      "id": "text_encoder",
      "type": "flux2_klein_text_encoder",
      "is_intermediate": true,
      "use_cache": true,
      "prompt": "a beautiful landscape",
      "max_seq_len": 512
    },
    "denoise": {
      "id": "denoise",
      "type": "flux2_denoise",
      "is_intermediate": true,
      "use_cache": false,
      "width": 1024,
      "height": 1024,
      "num_steps": 4,
      "guidance": 4.0,
      "seed": 0,
      "scheduler": "euler",
      "denoising_start": 0.0,
      "denoising_end": 1.0
    },
    "decode": {
      "id": "decode",
      "type": "flux2_vae_decode",
      "is_intermediate": false,
      "use_cache": false
    }
  },
  "edges": [
    {"source": {"node_id": "model_loader", "field": "transformer"}, "destination": {"node_id": "denoise", "field": "transformer"}},
    {"source": {"node_id": "model_loader", "field": "qwen3_encoder"}, "destination": {"node_id": "text_encoder", "field": "qwen3_encoder"}},
    {"source": {"node_id": "text_encoder", "field": "conditioning"}, "destination": {"node_id": "denoise", "field": "positive_text_conditioning"}},
    {"source": {"node_id": "denoise", "field": "latents"}, "destination": {"node_id": "decode", "field": "latents"}},
    {"source": {"node_id": "model_loader", "field": "vae"}, "destination": {"node_id": "decode", "field": "vae"}},
    {"source": {"node_id": "model_loader", "field": "vae"}, "destination": {"node_id": "denoise", "field": "vae"}}
  ]
}`

const sd15WorkflowJSON = `{
  "id": "sd15-txt2img",
  "nodes": {
    "model_loader": {
      "id": "model_loader",
      "type": "main_model_loader",
      "is_intermediate": true,
      "use_cache": true,
      "model": {"key": "REPLACE_WITH_MODEL_KEY", "name": "your-sd15-model", "base": "sd-1", "type": "main"}
    },
    "positive_prompt": {
      "id": "positive_prompt",
      "type": "compel",
      "is_intermediate": true,
      "use_cache": true,
      "prompt": "a beautiful landscape"
    },
    "negative_prompt": {
      "id": "negative_prompt",
      "type": "compel",
      "is_intermediate": true,
      "use_cache": true,
      "prompt": "bad quality, worst quality, lowres"
    },
    "noise": {
      "id": "noise",
      "type": "noise",
      "is_intermediate": true,
      "use_cache": false,
      "seed": 0,
      "width": 512,
      "height": 512,
      "use_cpu": true
    },
    "denoise": {
      "id": "denoise",
      "type": "denoise_latents",
      "is_intermediate": true,
      "use_cache": false,
      "steps": 20,
      "cfg_scale": 7.5,
      "denoising_start": 0.0,
      "denoising_end": 1.0,
      "scheduler": "euler"
    },
    "decode": {
      "id": "decode",
      "type": "l2i",
      "is_intermediate": false,
      "use_cache": false
    }
  },
  "edges": [
    {"source": {"node_id": "model_loader", "field": "unet"}, "destination": {"node_id": "denoise", "field": "unet"}},
    {"source": {"node_id": "model_loader", "field": "clip"}, "destination": {"node_id": "positive_prompt", "field": "clip"}},
    {"source": {"node_id": "model_loader", "field": "clip"}, "destination": {"node_id": "negative_prompt", "field": "clip"}},
    {"source": {"node_id": "positive_prompt", "field": "conditioning"}, "destination": {"node_id": "denoise", "field": "positive_conditioning"}},
    {"source": {"node_id": "negative_prompt", "field": "conditioning"}, "destination": {"node_id": "denoise", "field": "negative_conditioning"}},
    {"source": {"node_id": "noise", "field": "noise"}, "destination": {"node_id": "denoise", "field": "noise"}},
    {"source": {"node_id": "denoise", "field": "latents"}, "destination": {"node_id": "decode", "field": "latents"}},
    {"source": {"node_id": "model_loader", "field": "vae"}, "destination": {"node_id": "decode", "field": "vae"}}
  ]
}`
