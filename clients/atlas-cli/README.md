# atlas-cli

Terminal console for Project Atlas. Talks to the **inference gateway** over HTTP.

```
  █████╗ ████████╗██╗      █████╗ ███████╗
 ██╔══██╗╚══██╔══╝██║     ██╔══██╗██╔════╝
 ███████║   ██║   ██║     ███████║███████╗
 ██╔══██║   ██║   ██║     ██╔══██║╚════██║
 ██║  ██║   ██║   ███████╗██║  ██║███████║
 ╚═╝  ╚═╝   ╚═╝   ╚══════╝╚═╝  ╚═╝╚══════╝
```

## What it is

A small Go CLI with:

- Unicode **ATLAS** splash banner
- Interactive **chat** REPL (you › / atlas ›)
- One-shot **generate**
- **health** check against the gateway
- Optional durable conversations when Postgres is configured on the gateway
- Ephemeral fallback via `/v1/generate` when conversations are unavailable

## Prerequisites

1. Inference gateway running (default `http://localhost:8080`)
2. Go 1.22+ to build the CLI

```bash
# Option A — gateway only (fake model, no Postgres)
cd services/inference-gateway
export PATH=/usr/local/go/bin:$PATH
go run ./cmd/inference-gateway

# Option B — full local stack
cd deploy
docker compose up --build
```

## Build & run

```bash
cd clients/atlas-cli
export PATH=/usr/local/go/bin:$PATH

go build -o atlas .

./atlas                    # interactive chat
./atlas health             # /healthz + /readyz
./atlas generate "hello"   # one-shot
./atlas run <run_id>       # inspect persisted run (Postgres required)
./atlas conversation <id>  # show durable conversation
./atlas openapi            # print gateway OpenAPI YAML
./atlas -model gemma3      # when gateway uses Ollama
./atlas -url http://localhost:8080
./atlas -no-color          # plain text

# from repo root — one-command proof:
# make smoke
```

Install into your PATH (optional):

```bash
go install .
# binary lands in $(go env GOPATH)/bin — often ~/go/bin
```

## Chat slash commands

| Command | Action |
|---------|--------|
| `/help` | Show commands |
| `/new` | New conversation (or clear local history) |
| `/history` | Print messages so far |
| `/id` | Show conversation id |
| `/model <name>` | Switch model mid-session |
| `/clear` | Clear screen + redraw banner |
| `/quit` | Exit |

## Environment

| Variable | Default | Meaning |
|----------|---------|---------|
| `ATLAS_GATEWAY_URL` | `http://localhost:8080` | Gateway base URL |
| `ATLAS_INFERENCE_MODEL` | `fake-model` | Model name sent to the gateway |

## Modes

| Mode | When | API used |
|------|------|----------|
| **Durable** | Gateway has Postgres + conversation store | `POST /v1/conversations` + `/messages` |
| **Ephemeral** | No DB / conversations not ready | Local history + `POST /v1/generate` |

The CLI detects this automatically on startup.

## Status

Prototype client for local testing. Not a full agent console (no tools, approvals, or multi-agent views yet).

## Learn more

For a plain-language tour of Go, the gateway, Postgres, and the request path, read:

[docs/engineering/BEGINNER_BUILD_RECORD.md](../../docs/engineering/BEGINNER_BUILD_RECORD.md)

Root helpers: `make gateway-run`, `make cli-run`, `make test`.
