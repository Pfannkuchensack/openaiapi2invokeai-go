# invoke-openai-proxy

OpenAI-compatible image API proxy for [InvokeAI](https://github.com/invoke-ai/InvokeAI). Single static Go binary with an embedded admin web UI.

Allows tools like [Open-WebUI](https://github.com/open-webui/open-webui) and any other OpenAI Image API client to generate images via a local InvokeAI installation.

## Features

- **OpenAI Image API** — `/v1/images/generations`, `/v1/images/edits`, `/v1/images/variations`
- **Model registry** — map OpenAI model names to InvokeAI workflows with parameter injection
- **Admin UI** — embedded HTMX interface for managing workflows, models, and testing
- **Single binary** — no dependencies, no database, no CGO
- **Real-time** — Socket.IO integration for instant completion detection

## Quickstart

```bash
# Download from releases (or build from source)
./invoke-openai-proxy \
  --listen-ip 0.0.0.0 \
  --port 8080 \
  --invoke-url http://localhost:9090

# Visit http://localhost:8080/admin
# 1. Upload a workflow JSON exported from InvokeAI
# 2. Create a model mapping (set prompt/seed/size field paths)
# 3. Configure your OpenAI client to use http://<host>:8080/v1
```

### Open-WebUI Configuration

Set **OpenAI API Base URL** to `http://<proxy-host>:8080/v1` and the models from your registry will appear in the image generation dropdown.

## Build from Source

```bash
git clone https://github.com/Pfannkuchensack/openaiapi2invokeai-go.git
cd openaiapi2invokeai-go
make build        # → bin/invoke-openai-proxy
make cross        # → linux/amd64, linux/arm64, darwin/arm64, windows/amd64
```

Requires Go 1.22+.

## Configuration

All options available as CLI flags, environment variables, or `config.toml`:

| Flag | Env | Default | Description |
|------|-----|---------|-------------|
| `--listen-ip` | `PROXY_LISTEN_IP` | `127.0.0.1` | Bind address |
| `--port` | `PROXY_PORT` | `8080` | Listen port |
| `--invoke-url` | `INVOKE_URL` | `http://127.0.0.1:9090` | InvokeAI base URL |
| `--data-dir` | `PROXY_DATA_DIR` | `~/.invoke-openai-proxy` | Storage for workflows + registry |
| `--api-key` | `PROXY_API_KEY` | _(empty)_ | Bearer token for API auth |
| `--admin-user` | `PROXY_ADMIN_USER` | _(empty)_ | Basic-auth user for `/admin` |
| `--admin-pass` | `PROXY_ADMIN_PASS` | _(empty)_ | Basic-auth password for `/admin` |
| `--timeout` | `PROXY_TIMEOUT` | `300s` | Max wait per generation |
| `--log-level` | `PROXY_LOG_LEVEL` | `info` | `debug`/`info`/`warn`/`error` |

Priority: flag > environment > config file > default.

Config file location: `<data-dir>/config.toml`

## API Endpoints

### `GET /v1/models`
Lists all registered models in OpenAI format.

### `POST /v1/images/generations`
```json
{
  "model": "sdxl-fast",
  "prompt": "a beautiful sunset over mountains",
  "size": "1024x1024",
  "n": 1,
  "response_format": "b64_json"
}
```

### `POST /v1/images/edits`
Multipart form: `image`, `mask` (optional), `prompt`, `model`, `size`, `n`

### `POST /v1/images/variations`
Multipart form: `image`, `model`, `size`, `n`

### `GET /healthz`
Health check endpoint.

## Model Registry

Models are defined in `<data-dir>/registry.json`. Each model maps an ID to a workflow file and defines which graph nodes receive which parameters:

```json
{
  "models": [{
    "id": "sdxl-fast",
    "workflow": "sdxl-workflow.json",
    "mapping": {
      "prompt": "nodes.<node-uuid>.value",
      "negative": "nodes.<node-uuid>.value",
      "seed": "nodes.<noise-uuid>.seed",
      "width": "nodes.<noise-uuid>.width",
      "height": "nodes.<noise-uuid>.height",
      "steps": "nodes.<denoise-uuid>.steps",
      "cfg": "nodes.<denoise-uuid>.cfg_scale"
    },
    "defaults": {"steps": 20, "cfg": 7.5},
    "size_presets": {
      "1024x1024": {"width": 1024, "height": 1024},
      "1792x1024": {"width": 1792, "height": 1024}
    }
  }]
}
```

Use the **Admin UI → Workflows → Inspect** page to find the correct node UUIDs and field names.