# invoke-openai-proxy – Implementation Plan

OpenAI-compatible API proxy for InvokeAI, distributed as a single static Go binary
with an embedded admin web UI for managing workflows and model-name mappings.

## Goal

Allow tools like Open-WebUI (and any other OpenAI-Image-API client) to talk to a
local InvokeAI installation. The proxy translates OpenAI image requests into
InvokeAI workflow-graph queue submissions, waits for completion, and returns the
generated image in OpenAI's response format.

## High-level architecture

```
OpenAI client (Open-WebUI, etc.)
        │  HTTP, OpenAI image schema
        ▼
invoke-openai-proxy  (single Go binary)
  ├─ /v1/images/generations
  ├─ /v1/images/edits
  ├─ /v1/images/variations
  ├─ /v1/models               (derived from registry)
  ├─ /admin/*                  (embedded HTMX UI)
  └─ /healthz
        │  REST + WebSocket
        ▼
InvokeAI  (existing instance, untouched)
```

Mapping flow per request:
1. Receive OpenAI request (prompt, n, size, model, optional image+mask).
2. Look up `model` in the registry → workflow JSON + parameter mapping.
3. Build a queue-batch payload by substituting prompt/seed/size/etc. into the workflow.
4. POST to `/api/v1/queue/{queue_id}/enqueue_batch`.
5. Subscribe to Invoke's socket OR poll the queue item until completion.
6. Fetch the resulting image, base64-encode (or return URL).
7. Respond in OpenAI schema: `{ created, data: [{ b64_json | url, revised_prompt? }] }`.

## Configuration

CLI flags (also via env vars and config file, flags win):

| Flag | Env | Default | Purpose |
|------|-----|---------|---------|
| `--listen-ip` | `PROXY_LISTEN_IP` | `127.0.0.1` | Bind address (use `0.0.0.0` for LAN) |
| `--port` | `PROXY_PORT` | `8080` | Listen port |
| `--invoke-url` | `INVOKE_URL` | `http://127.0.0.1:9090` | InvokeAI base URL |
| `--data-dir` | `PROXY_DATA_DIR` | `~/.invoke-openai-proxy` | Workflows + registry storage |
| `--api-key` | `PROXY_API_KEY` | _empty_ | Optional Bearer token clients must send |
| `--admin-user` | `PROXY_ADMIN_USER` | _empty_ | Basic-auth user for `/admin` (empty = open) |
| `--admin-pass` | `PROXY_ADMIN_PASS` | _empty_ | Basic-auth password for `/admin` |
| `--timeout` | `PROXY_TIMEOUT` | `300s` | Max wait per generation |
| `--no-browser` | — | `false` | Don't auto-open admin on first run |
| `--log-level` | `PROXY_LOG_LEVEL` | `info` | `debug`/`info`/`warn`/`error` |

Config file: `~/.invoke-openai-proxy/config.toml` (optional, same keys).

## Repo layout

```
invoke-openai-proxy/
├── go.mod
├── go.sum
├── Makefile
├── README.md
├── LICENSE
├── .github/workflows/release.yml      # cross-compile + GitHub release
├── cmd/proxy/main.go                   # CLI entry, flag parsing, startup
├── internal/
│   ├── config/                         # flags + env + toml loader
│   ├── server/                         # router, middleware, lifecycle
│   ├── openai/                         # OpenAI schema types + handlers
│   │   ├── types.go
│   │   ├── generations.go
│   │   ├── edits.go
│   │   └── models.go
│   ├── invoke/                         # InvokeAI client
│   │   ├── client.go                   # REST: enqueue, status, image fetch
│   │   ├── socket.go                   # WS: completion subscription
│   │   └── types.go
│   ├── workflow/                       # parse + parameterize workflow JSON
│   │   ├── registry.go                 # CRUD over workflows + model mapping
│   │   ├── parameterize.go             # substitute prompt/seed/size/etc.
│   │   └── inspect.go                  # extract input fields from a graph
│   ├── admin/                          # admin UI handlers
│   │   ├── handlers.go
│   │   └── templates/                  # Go html/template files (HTMX)
│   └── version/version.go
├── web/admin/                          # static assets (CSS, htmx.min.js)
│   ├── htmx.min.js
│   └── style.css
└── testdata/
    └── workflows/sdxl-example.json
```

All UI assets and templates go through `embed.FS` → one binary, no external files
needed at runtime.

## Dependencies (keep minimal)

- `github.com/go-chi/chi/v5` — router
- `github.com/gorilla/websocket` — Invoke socket client
- `github.com/BurntSushi/toml` — config file
- `github.com/spf13/pflag` — POSIX-style flags (optional, stdlib `flag` is fine)
- `log/slog` — stdlib structured logging
- `embed`, `html/template` — stdlib

No CGO, no database. Workflows + registry stored as JSON files on disk.

## Data model

```
~/.invoke-openai-proxy/
├── config.toml                    # optional
├── workflows/
│   ├── sdxl-fast.json             # raw InvokeAI workflow graph
│   └── flux-quality.json
└── registry.json                  # model-name → workflow + param mapping
```

`registry.json`:
```json
{
  "models": [
    {
      "id": "invoke-sdxl-fast",
      "workflow": "sdxl-fast.json",
      "defaults": { "steps": 20, "cfg": 5.5, "scheduler": "dpmpp_2m" },
      "mapping": {
        "prompt":      "nodes.positive_prompt.value",
        "negative":    "nodes.negative_prompt.value",
        "width":       "nodes.image_size.width",
        "height":      "nodes.image_size.height",
        "seed":        "nodes.noise.seed"
      },
      "size_presets": {
        "1024x1024": { "width": 1024, "height": 1024 },
        "1792x1024": { "width": 1792, "height": 1024 },
        "1024x1792": { "width": 1024, "height": 1792 }
      }
    }
  ]
}
```

## Admin UI (HTMX + html/template, embedded)

Pages, all under `/admin`:

1. **Dashboard** — Invoke health, recent requests, version.
2. **Workflows** — list, upload JSON, delete. Upload parses and shows detected
   input fields ready for mapping.
3. **Models** — CRUD on `registry.json` entries: id, workflow, defaults, field
   mapping, size presets.
4. **Test** — prompt + model picker → live generation → show image + raw
   request/response for debugging.
5. **Settings** — read-only view of effective config (where each value came
   from: flag / env / file / default).
6. **Logs** — last N requests: timestamp, model, duration, status, error.

Basic auth via `--admin-user` / `--admin-pass` if set; otherwise open (intended
for local use behind `127.0.0.1`).

## Phased implementation

### Phase 1 — Skeleton (1 session)
- `cmd/proxy/main.go`, flags, slog setup, graceful shutdown.
- Chi router, `/healthz`, `/v1/models` returning stub.
- Config layering: flag > env > toml > default. Print effective config at startup.
- Cross-compile target in Makefile (`linux/amd64`, `linux/arm64`, `darwin/arm64`, `windows/amd64`).
- **Exit criteria:** binary builds, `--listen-ip` + `--port` work, `/healthz` returns 200.

### Phase 2 — Invoke client (1 session)
- `internal/invoke`: enqueue_batch, get queue item, fetch image bytes.
- WebSocket subscriber for completion events (with REST polling fallback).
- Integration test against a real local Invoke instance behind a flag.
- **Exit criteria:** given a hardcoded workflow JSON + prompt, the client can
  enqueue and retrieve the resulting image bytes.

### Phase 3 — Workflow + registry (1 session)
- `internal/workflow`: load workflow files, parameterize via JSONPath-like
  field mappings, validate against `registry.json`.
- File-based registry CRUD with mutex (single-writer, many-reader).
- Unit tests with a real exported Invoke workflow in `testdata/`.
- **Exit criteria:** can take an OpenAI-shaped struct + model id and produce a
  ready-to-enqueue payload.

### Phase 4 — OpenAI endpoints (1 session)
- `/v1/images/generations` — text-to-image. Honor `n`, `size`, `response_format`
  (`b64_json` + `url`), `user`. Reject `quality`/`style` unless mapped.
- `/v1/models` — derived from registry.
- Bearer-token middleware if `--api-key` set.
- **Exit criteria:** Open-WebUI configured against the proxy can generate
  images.

### Phase 5 — Admin UI (1–2 sessions)
- Templates + HTMX, embedded via `embed.FS`.
- Workflows: upload + list + delete + field inspection.
- Models: full CRUD with field-mapping editor.
- Test page with live generation.
- Settings + logs views.
- **Exit criteria:** complete onboarding works in the browser without touching
  files manually.

### Phase 6 — Edits + variations (optional, 1 session)
- `/v1/images/edits` — accepts `image` + optional `mask`, routes to an
  inpainting workflow id (configurable per-model in registry).
- `/v1/images/variations` — same idea, image-to-image with high denoising.
- **Exit criteria:** mask-based inpainting via the OpenAI edits endpoint works.

### Phase 7 — Polish + release (1 session)
- README with quickstart + Open-WebUI configuration screenshot.
- GitHub Actions: build matrix, attach binaries to release tags.
- `LICENSE` (MIT or Apache-2.0).
- Example workflow JSON in `testdata/` so users have something to import.
- **Exit criteria:** `v0.1.0` tag → release page has Linux/macOS/Windows binaries.

## Open questions to resolve early

1. **Sync vs async response** — OpenAI is blocking. Hold the HTTP connection
   for up to `--timeout`? Or stream `Transfer-Encoding: chunked` heartbeats?
   Decision affects reverse-proxy compatibility.
2. **Image hosting for `response_format: url`** — serve from `/v1/images/{id}`
   on the proxy with a TTL cache? Or always force `b64_json`? Start with
   `b64_json` only, add `url` later if needed.
3. **Auth split** — separate token for API vs admin? Plan above does that.
4. **Workflow versioning** — when a user re-uploads with the same filename,
   keep history? Phase 1 says no, just overwrite.
5. **Multi-instance Invoke** — out of scope for v0.1, but the client interface
   should be small enough to wrap with a load balancer later.

## Quickstart you'll want in the README

```bash
# Download from releases, then:
./invoke-openai-proxy \
  --listen-ip 0.0.0.0 \
  --port 8080 \
  --invoke-url http://localhost:9090

# Visit http://localhost:8080/admin
# 1. Upload a workflow JSON exported from Invoke
# 2. Define a model id mapping to that workflow
# 3. In Open-WebUI: set OpenAI base URL to http://<host>:8080/v1
```

## Non-goals (explicit)

- No fork of InvokeAI, no patches to it.
- No support for non-image OpenAI endpoints (chat, embeddings, audio).
- No multi-tenant / user management beyond the single API key + admin login.
- No queue priority, rate limiting, billing accounting in v0.1.
- No SQLite or external DB.

## Starter prompt for the new session

> I want to start a new Go project called `invoke-openai-proxy`. It's a
> single-binary OpenAI-compatible image API that forwards to a local InvokeAI
> instance, with an embedded HTMX admin UI for managing workflows and
> model-name mappings. The full plan is in `PROXY_PLAN.md` — please read it,
> then start with **Phase 1 (Skeleton)**: scaffold the repo, flag parsing
> (especially `--listen-ip` and `--port`), config layering, `/healthz`, and a
> stub `/v1/models`. Use chi, slog, embed.FS, no CGO. Confirm the plan looks
> reasonable before scaffolding.
