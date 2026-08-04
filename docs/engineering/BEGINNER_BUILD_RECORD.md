# Beginner Build Record — How Atlas Works (Go, CLI, Postgres, and friends)

**Last updated:** 2026-08-04  
**Audience:** someone new to Go, backends, and this repository.  
**Purpose:** a durable “what happened and how it connects” guide that matches the **real** tree (not aspirational services that do not exist yet).

If you only read one teaching doc after `README.md`, make it this file.

---

## 1. What is Project Atlas? (simple story)

Atlas is meant to be an **AI agent platform**, not “just a chat box.”

Imagine two products:

| Simple chatbot | Atlas (goal) |
|----------------|--------------|
| User → OpenAI API → answer | User → platform (auth, history, planning, tools, models, eval) → answer |
| One vendor hard-coded | Models plugged in behind **interfaces** (swap Fake / Ollama / later cloud) |
| Memory is the browser tab | Conversations and runs can live in **Postgres** |

**Today’s reality:** lots of **design documentation** is done (M0). The first **runnable** pieces are:

1. **`services/inference-gateway`** — a Go HTTP service that calls a model provider and can save runs/conversations.
2. **`clients/atlas-cli`** — a colorful terminal client that talks to that service.
3. **`deploy/`** — Docker Compose for local Postgres, Redis, Qdrant, MinIO, Ollama, and the gateway.

**Not built yet as runnable apps:** agent-orchestrator (docs only), full web UI, OpenAI/Anthropic adapters, Kubernetes production stack.

---

## 2. Map of the repository (what actually exists)

```text
Project-Atlas/
  docs/                     Product, architecture, ADRs, domain language
  management/               Roadmap, backlog, milestones
  contracts/                Provider/tool contract docs
  schemas/                  JSON schemas (workspace, conversation, run)
  deploy/                   docker-compose.yml + docker-compose.fake.yml
  services/
    inference-gateway/      ★ Runnable Go service
    agent-orchestrator/     Docs shell only (no main.go yet)
  clients/
    atlas-cli/              ★ Runnable Go CLI
  Makefile                  make gateway-run / cli-run / test
  docs/engineering/
    BEGINNER_BUILD_RECORD.md  ← you are here
```

---

## 3. What has been built so far (timeline of the story)

### Phase A — Documentation first (M0)

Engineers wrote product vision, domain words (workspace, conversation, run…), architecture notes, and **ADRs** (Architecture Decision Records: “we chose X over Y because…”).

**Why?** So code follows contracts instead of inventing a different product every week.

### Phase B — Inference gateway (start of M1)

A real Go module under `services/inference-gateway` that:

- Listens on HTTP (`/healthz`, `/readyz`, `/v1/generate`, conversations, runs)
- Talks to a **Fake** provider (deterministic) or **Ollama** (local models)
- Optionally writes **inference runs** and **conversations** to **PostgreSQL**
- Retries flaky provider errors, attaches request IDs, basic OpenTelemetry hooks

### Phase C — Local data plane

`deploy/docker-compose.yml` runs supporting systems (Postgres, Redis, Qdrant, MinIO, Ollama) plus the gateway.

`deploy/docker-compose.fake.yml` is an overlay that sets the gateway to the **fake** provider so beginners can test **without downloading a model**.

### Phase D — Terminal CLI

`clients/atlas-cli` prints an **ATLAS** Unicode banner and can:

- `health` — check the live gateway
- `generate "…"` — one-shot chat
- interactive `chat` — REPL with `/help`, `/history`, `/quit`, …

If Postgres conversations are unavailable, the CLI falls back to **ephemeral** mode (local history + `/v1/generate`).

---

## 4. Go basics as used in *this* project

You do **not** need a full Go textbook. Learn these ideas as they appear in Atlas.

### 4.1 Packages and `main`

- A **package** is a folder of `.go` files that share a name.
- Library code in the gateway lives in package `inference` (files at the module root).
- A program that *starts* must be `package main` with `func main()`.

Example entry points:

| Path | Role |
|------|------|
| `services/inference-gateway/cmd/inference-gateway/main.go` | Starts the HTTP server |
| `clients/atlas-cli/main.go` | Starts the CLI |

Newbie analogy: `main` is the light switch; other files are rooms full of furniture.

### 4.2 Modules (`go.mod`)

```text
module github.com/KunalSiyag/Project-Atlas/services/inference-gateway
go 1.25.0
```

This is like `package.json` / Cargo.toml: **module path + dependencies**.

Commands you will see:

```bash
go test ./...     # run tests in this module
go build -o atlas .   # compile a binary
go run ./cmd/inference-gateway   # compile+run without keeping the binary
```

### 4.3 Structs (data bags)

```go
type Message struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}
```

- `struct` = “object with named fields.”
- `` `json:"role"` `` = when converting to/from JSON, use the key `"role"`.

### 4.4 Interfaces (plugs)

```go
type ModelProvider interface {
    Generate(ctx context.Context, request ChatRequest) (ChatResponse, error)
}
```

Anything that implements `Generate` with that signature can be a provider:

- `FakeProvider` — returns `"fake response to: " + lastMessage`
- Ollama adapter — HTTP-calls a local model server

The gateway depends on the **interface**, not on OpenAI’s SDK. That is **provider independence**.

### 4.5 Methods

```go
func (p FakeProvider) Generate(ctx context.Context, request ChatRequest) (ChatResponse, error) {
    // ...
}
```

`(p FakeProvider)` means this function is a **method** on `FakeProvider`. That is how the type “implements” the interface (no `implements` keyword in Go).

### 4.6 Errors

Go usually returns errors as values:

```go
result, err := provider.Generate(ctx, req)
if err != nil {
    return err
}
```

There is no exception try/catch style as the default.

### 4.7 Context

`context.Context` carries cancellation and deadlines for one request. If the client disconnects, work can stop.

### 4.8 HTTP in the standard library

The gateway implements `http.Handler` via `ServeHTTP` on `Gateway`.  
The CLI uses `net/http` as a **client** (`http.Client.Do`).

### 4.9 Tests

Files named `*_test.go` run with `go test`.  
Good tests call the **real shipped functions** (e.g. `client.Generate`) against a tiny `httptest` server — not a fake reimplementation of the client.

---

## 5. How the pieces connect

```text
┌────────────────────┐     HTTP/JSON      ┌──────────────────────────┐
│  atlas-cli         │ ─────────────────► │  inference-gateway       │
│  clients/atlas-cli │ ◄───────────────── │  services/inference-…    │
└────────────────────┘   response JSON    └────────────┬─────────────┘
                                                       │
                         ┌─────────────────────────────┼─────────────────────────────┐
                         │                             │                             │
                         ▼                             ▼                             ▼
                  ModelProvider                 RunStore /                 (optional)
                  Fake or Ollama                ConversationStore          OTEL metrics
                         │                      Postgres (if URL set)
                         ▼
                   Fake: in-process
                   Ollama: http://ollama:11434
```

| Piece | Talks to | How |
|-------|----------|-----|
| CLI | Gateway | `POST /v1/generate`, `GET /healthz`, conversation routes |
| Gateway | Provider | Go interface `ModelProvider.Generate` |
| Gateway | Postgres | `database/sql` + `pgx` driver; SQL migrations embedded |
| Compose | All containers | Shared Docker network; hostnames `postgres`, `ollama`, … |

**Config is environment variables**, for example:

- `ATLAS_INFERENCE_ADDR` — listen address (`:8080`)
- `ATLAS_INFERENCE_PROVIDER` — `fake` or `ollama`
- `ATLAS_INFERENCE_MODEL` — model name
- `ATLAS_DATABASE_URL` — empty = no durable store; set = Postgres on
- `ATLAS_GATEWAY_URL` — CLI → gateway base URL

---

## 6. Request lifecycle (CLI → answer)

### Path A — one-shot generate (always available if gateway is up)

1. You run: `./atlas generate "hello atlas"`
2. CLI prints the ATLAS banner, then `POST /v1/generate` with JSON messages.
3. Gateway assigns/propagates `X-Request-ID` and `X-Run-ID`.
4. Router picks provider + model.
5. If Postgres is configured, gateway inserts a **running** inference run.
6. Provider `Generate` runs (fake returns a string immediately).
7. Gateway marks the run **succeeded** (or **failed**) and returns JSON.
8. CLI prints `atlas ›` plus the assistant content and short metadata.

### Path B — interactive chat with durable conversations

1. CLI `POST /v1/conversations` (creates a row if store is ready).
2. Each user line: `POST /v1/conversations/{id}/messages`.
3. Gateway loads history, appends user message, calls provider with **full history**, appends assistant message, links `run_id`.
4. `/history` in the CLI re-fetches the conversation from the server.

### Path C — ephemeral fallback

If conversation create fails (no DB), CLI keeps messages in memory and uses Path A each turn.

---

## 7. PostgreSQL in plain language

Postgres is a **database**: durable tables of rows.

Gateway migrations (SQL files under `services/inference-gateway/migrations/`):

| File | Tables |
|------|--------|
| `001_inference_runs.sql` | One row per model call attempt |
| `002_conversations.sql` | Conversations + ordered messages |
| `003_message_idempotency.sql` | Safer retries for the same message key |

On startup, if `ATLAS_DATABASE_URL` is set, the gateway opens a connection, pings, and applies these schemas.

**Analogy:** Postgres is a filing cabinet; Go store code is the clerk; HTTP handlers are the front desk.

---

## 8. How to run it yourself (beginner path)

### Fastest: fake provider, no Docker

Terminal 1:

```bash
export PATH=/usr/local/go/bin:$PATH
make gateway-run
```

Terminal 2:

```bash
export PATH=/usr/local/go/bin:$PATH
make cli-health
make cli-generate MSG='hello atlas'
make cli-run
```

You should see the **ATLAS** banner and a line like `fake response to: hello atlas`.

### With Docker (fake overlay)

```bash
make compose-fake
# then make cli-health against localhost:8080
```

### Tests (prove the code, not just “it worked once”)

```bash
make test
# or:
cd services/inference-gateway && go test ./...
cd clients/atlas-cli && go test ./...
```

---

## 9. Important files to open while learning

| File | Why |
|------|-----|
| `clients/atlas-cli/main.go` | CLI commands and flags |
| `clients/atlas-cli/client.go` | HTTP client (how Go calls the gateway) |
| `clients/atlas-cli/banner.go` | Unicode ATLAS art + colors |
| `clients/atlas-cli/chat.go` | Interactive REPL |
| `services/inference-gateway/provider.go` | Interface + message types |
| `services/inference-gateway/fake_provider.go` | Simplest provider |
| `services/inference-gateway/handler.go` | HTTP routes |
| `services/inference-gateway/cmd/inference-gateway/main.go` | Process startup |
| `services/inference-gateway/run_store.go` | Postgres runs |
| `services/inference-gateway/conversation_store.go` | Postgres chats |
| `deploy/docker-compose.yml` | Local multi-service stack |

---

## 10. Glossary (tiny)

| Word | Meaning here |
|------|----------------|
| **Gateway** | Front door HTTP service for inference |
| **Provider** | Code that actually talks to a model (or fakes it) |
| **Run** | One recorded attempt to generate |
| **Conversation** | Durable thread of messages |
| **Compose** | Tool that runs multiple containers together |
| **CLI** | Command-line interface (terminal app) |
| **Idempotency** | Doing the same request twice doesn’t double-create data |

---

## 11. What is still open (honest)

From `services/inference-gateway/TODO.md` and the roadmap:

- OpenAPI formalization
- Full OpenTelemetry exporter wiring for production
- Hosted model adapters (OpenAI, etc.) after conformance
- Agent orchestrator implementation
- Web UI (CLI is the intentional first surface)

---

## 12. One-sentence takeaway

**Atlas is a documentation-first platform where a Go inference gateway (pluggable models + optional Postgres) is already runnable, and a colorful terminal CLI is the beginner way to poke it—while the bigger agent brain is designed but not fully coded yet.**

---

## 13. Continuation log (what we kept building)

This section is a simple diary so you can see progress over sessions.

### Session: CLI + beginner path (earlier)

- Built `clients/atlas-cli` with Unicode ATLAS banner, health/generate/chat.
- Added fake Compose overlay and Makefile helpers.
- Wrote this beginner record.

### Session: OpenAPI + richer CLI (continued)

**Goal:** make the HTTP contract explicit and give the CLI more “inspector” power.

| Change | Why it helps a beginner |
|--------|-------------------------|
| `services/inference-gateway/openapi.yaml` | A machine-readable menu of real endpoints (not dream features) |
| `GET /openapi.yaml` on the running gateway | You can curl the contract from the live server |
| CLI `atlas openapi` | Download/print that contract |
| CLI `atlas run <run_id>` | Inspect a saved inference run (when Postgres is on) |
| CLI `atlas conversation <id>` | Print a saved chat thread |
| `./scripts/dev_smoke.sh` / `make smoke` | One command proves gateway + CLI still work |

**Still not done (honest):**

- OpenTelemetry exporter wiring for production dashboards
- Hosted model providers (OpenAI, etc.)
- Embeddings HTTP endpoint (mentioned in older API notes, not coded)
- Full agent orchestrator

**How the new OpenAPI piece fits in Go:**

```text
openapi.yaml  ──embed──►  openapi.go (//go:embed)
                              │
                              ▼
                     Gateway.ServeHTTP
                     GET /openapi.yaml  → raw YAML bytes
```

`//go:embed` means: at compile time, Go bakes the file into the binary. No separate copy step when you deploy the gateway.

**Try it:**

```bash
make smoke
# or after make gateway-run:
curl -s http://127.0.0.1:8080/openapi.yaml | head
./clients/atlas-cli/atlas openapi | head
```

### Session: OTEL + OpenAI-style + embeddings + orchestrator code

| Piece | What it means in plain English |
|-------|--------------------------------|
| `ATLAS_OTEL_EXPORTER=console` | Gateway prints real OpenTelemetry spans/metrics to the terminal (telemetry *leaves* the process) |
| `ATLAS_OTEL_EXPORTER=otlp` | Same, but sends to an OTLP collector (`ATLAS_OTEL_ENDPOINT`) |
| `ATLAS_INFERENCE_PROVIDER=openai` | Chat via OpenAI-compatible `/v1/chat/completions` |
| `POST /v1/embeddings` | Turn text into vectors (fake = deterministic local vectors) |
| `services/agent-orchestrator` | **Code** now: create/start/complete in-memory runs + `/healthz` |

**Conformance suite:** `go test` in the gateway package includes `TestChatProviderConformance` — same checks against **fake** and **OpenAI-style** (with `httptest` fake API).

```bash
make gateway-test   # includes OTEL, openai, embeddings, conformance
make orch-test
make orch-run       # :8090
```

### Session: Orchestrator → gateway + Postgres + plan DAG + context builder

| Capability | Meaning |
|------------|---------|
| **Start run** | No longer only flips `running` — executes steps and calls the gateway |
| **Plan graph** | `prepare` → `infer` (depends on prepare, 2 attempts) → `finalize` |
| **Retry** | Failed model step retries until `max_attempts` |
| **Cancel** | `POST /v1/runs/{id}/cancel` stops later steps |
| **Postgres** | `ATLAS_ORCHESTRATOR_DATABASE_URL` keeps runs after restart |
| **Context builder** | Drops old history so estimated tokens stay under budget |

Flow:

```text
CreateRun → pending + plan steps
StartRun  → running
  prepare (context) → infer (POST gateway /v1/generate) → finalize
         → succeeded + result  OR  failed + error  OR  cancelled
```

### Session: tools, memory, planner, CLI agent run, RAG

| Piece | What it does |
|-------|----------------|
| **Tools** | `echo`, `calculator`, `http_get`, `danger_echo` with schema + timeout |
| **Approvals** | Dangerous tools pause as `waiting_approval` until `/approve` or `/reject` |
| **Planner** | Goal keywords build a real step graph (not only the old fixed three steps) |
| **Memory** | Short-term per run + long-term facts → fed into context builder |
| **RAG** | Ingest docs, embed, top-k search (Qdrant or in-process), inject snippets |
| **CLI** | `atlas run "calculate 2+2"` create → start → show steps → result |

```bash
./atlas run "calculate 21+21"
./atlas run -auto-approve "danger echo do the thing"
```
