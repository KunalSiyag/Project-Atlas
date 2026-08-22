# Complete Beginner Guide to Project Atlas

**Who this is for:** you, if you do **not** know Go, backends, Docker, or this repo.  
**Goal:** explain *what* this project is, *what tools* it uses, *how the pieces fit*, and *exactly what to type* to try it.  
**Last updated:** 2026-08-05

You do **not** need to become a Go expert to understand Atlas. Treat this like a map of a building: rooms, doors, and light switches—not the blueprint of every brick.

---

## Table of contents

1. [The 60-second story](#1-the-60-second-story)
2. [Words you will see (glossary)](#2-words-you-will-see-glossary)
3. [What software is this built with?](#3-what-software-is-this-built-with)
4. [Map of the repository](#4-map-of-the-repository)
5. [How the system fits together](#5-how-the-system-fits-together)
6. [What you need installed](#6-what-you-need-installed)
7. [Step-by-step: try Atlas with zero Docker](#7-step-by-step-try-atlas-with-zero-docker)
8. [Step-by-step: full local stack with Docker](#8-step-by-step-full-local-stack-with-docker)
9. [What each major piece does](#9-what-each-major-piece-does)
10. [How a “run” works (agent orchestrator)](#10-how-a-run-works-agent-orchestrator)
11. [Tools, prompts, knowledge, and eval (plain language)](#11-tools-prompts-knowledge-and-eval-plain-language)
12. [Common commands cheat sheet](#12-common-commands-cheat-sheet)
13. [How to read Go code without panic](#13-how-to-read-go-code-without-panic)
14. [How testing works here](#14-how-testing-works-here)
15. [Where docs and “status” live](#15-where-docs-and-status-live)
16. [What is intentionally *not* built yet](#16-what-is-intentionally-not-built-yet)
17. [If something breaks](#17-if-something-breaks)
18. [Suggested learning path](#18-suggested-learning-path)

---

## 1. The 60-second story

**Project Atlas** is software for running **AI agents** in a controlled way.

A normal chatbot is:

```text
You type → one AI API → answer
```

Atlas aims for:

```text
You (or a system) ask for a goal
    → Platform plans steps
    → Calls a model when needed
    → May use tools (calculator, HTTP, GitHub-style APIs, …)
    → Remembers useful facts
    → Can search knowledge documents
    → Records what happened (so you can debug and evaluate)
```

Important idea used everywhere in this repo:

> **The LLM is a plug-in, not the whole product.**  
> The product is planning, tools, memory, APIs, tests, deploy, and operations.

### Two main “servers” today

| Name | Port (local) | Job in one sentence |
|------|----------------|---------------------|
| **Inference gateway** | `8080` | Talks to AI models (or a fake model for testing) |
| **Agent orchestrator** | `8090` | Turns a goal into a multi-step **run** (plan → tools → model → result) |

There is also a **terminal program** (`atlas-cli`) so you can chat without a web UI.

---

## 2. Words you will see (glossary)

| Word | Plain meaning |
|------|----------------|
| **Repository / repo** | This whole project folder, tracked by Git |
| **Service** | A long-running program that answers network requests (like a small website API) |
| **HTTP / API** | Rules for how programs talk over the network using URLs and methods like GET/POST |
| **Endpoint** | One URL action, e.g. `GET /healthz` means “are you alive?” |
| **JSON** | Text format for structured data: `{"status":"ok"}` |
| **Go (Golang)** | The programming language most of the runnable Atlas code is written in |
| **Module** | A Go package group with a `go.mod` file (dependencies listed there) |
| **CLI** | Command-line interface — a program you run in the terminal |
| **Docker** | Runs apps in isolated “containers” so Postgres/Redis/etc. install consistently |
| **Compose** | A Docker tool that starts many containers together from a YAML file |
| **Postgres** | A database (stores runs, events, memory, prompts long-term) |
| **Redis** | Fast temporary store (short-term memory / cache) |
| **Qdrant** | Vector database used for “search similar text” (RAG) |
| **RAG** | Retrieval-Augmented Generation: search docs, then let the model use them |
| **Orchestrator** | The service that **plans and executes** agent runs |
| **Run** | One attempt to achieve a **goal** (has steps, status, result) |
| **Step / plan DAG** | Ordered steps with dependencies (A before B) |
| **Tool** | A capability the agent can call (echo, calculator, remote HTTP, …) |
| **Authz** | Authorization: “is this caller allowed to use that tool?” |
| **Prompt registry** | Versioned text templates for the model (load from files or database) |
| **Workspace** | A tenant/isolation key (like a company or project id) |
| **Eval** | Automated tests of quality (fixtures with expected outcomes) |
| **Bench** | Load test: hit the service many times, measure speed |
| **OpenAPI** | Machine-readable description of an HTTP API (`openapi.yaml`) |
| **Makefile** | File of short aliases: `make gateway-run` runs a longer command for you |
| **Icebox** | Work we deliberately **do not** build yet (too big / too early) |
| **Milestone (M0–M5)** | Big project phases (docs → core → intelligence → tools → knowledge → production) |

---

## 3. What software is this built with?

### Go (the language)

- Files end in `.go`
- You **compile** or **run** them with the `go` command
- Atlas services are mostly **HTTP servers** written in Go

You do **not** need to write Go to *use* Atlas. You need Go installed to *run* local commands like `make gateway-run` and `go test`.

### Supporting systems (when using Docker)

| System | Why Atlas uses it |
|--------|-------------------|
| **PostgreSQL** | Durable storage for runs, events, long memory, published prompts |
| **Redis** | Short-term notes during a run |
| **Qdrant** | Vector search for knowledge documents |
| **MinIO** | S3-like object storage (local stand-in) |
| **Ollama** | Run open models on your machine (optional; beginners can use **fake**) |
| **Grafana / OTEL** | Metrics and dashboards (optional overlay) |

### Fake model (beginner mode)

Instead of downloading a real AI model, the gateway can use a **fake provider**:

- Always available
- Deterministic answers
- Perfect for learning the APIs and running tests

---

## 4. Map of the repository

```text
Project-Atlas/
│
├── README.md                 ← Project home page
├── AI_CONTEXT.md             ← Short context for humans and AIs
├── Makefile                  ← Shortcuts: make gateway-run, make test, make bench
│
├── docs/
│   ├── product/              ← What we are building (vision, goals)
│   ├── domain/               ← Shared vocabulary (workspace, run, …)
│   ├── architecture/         ← How systems connect (incl. multi-region design)
│   ├── adr/                  ← Why we chose design X over Y
│   ├── engineering/          ← THIS GUIDE + principles
│   ├── runbooks/             ← “What to do when production breaks”
│   └── ops/                  ← Security audit notes
│
├── management/
│   ├── BACKLOG.md            ← What is done vs next vs icebox
│   ├── MILESTONES.md         ← M0–M5 status
│   └── ROADMAP.md            ← Long-term ladder
│
├── contracts/                ← Interface descriptions (chat, tools, memory)
├── schemas/                  ← JSON shapes for core entities
├── prompts/                  ← File-backed prompt templates
├── fixtures/eval/            ← Eval test cases (JSON)
│
├── services/
│   ├── inference-gateway/    ← Model HTTP service (port 8080)
│   └── agent-orchestrator/   ← Agent run service (port 8090)
│
├── clients/
│   └── atlas-cli/            ← Terminal client
│
├── tools/
│   ├── atlas-eval/           ← Offline evaluation harness
│   └── atlas-bench/          ← Load/latency bench tool
│
├── deploy/
│   ├── docker-compose.yml    ← Full local stack
│   ├── docker-compose.fake.yml
│   ├── k8s/                  ← Kubernetes starter manifests
│   └── grafana/              ← Dashboards
│
└── tasks/                    ← Numbered engineering tasks (0001, 0006, …)
```

**Rule of thumb**

| If you want to… | Look here |
|-----------------|-----------|
| Understand the product | `docs/product/` |
| Run something | `Makefile`, `deploy/`, `services/*/cmd/` |
| See what’s left to build | `management/BACKLOG.md` |
| Learn without prior context | **this file** |

---

## 5. How the system fits together

```text
                    ┌─────────────────────┐
   You / CLI / curl │                     │
  ─────────────────►│  Agent orchestrator │  :8090
                    │  (plans + tools)    │
                    └──────────┬──────────┘
                               │  “please generate text”
                               ▼
                    ┌─────────────────────┐
                    │  Inference gateway  │  :8080
                    │  (model access)     │
                    └──────────┬──────────┘
                               │
               ┌───────────────┼───────────────┐
               ▼               ▼               ▼
           Fake model      Ollama         (later: cloud)
                               │
        ┌──────────────────────┼──────────────────────┐
        ▼                      ▼                      ▼
   PostgreSQL               Redis                  Qdrant
   (runs, memory,           (short notes)          (doc search)
    prompts)
```

**Gateway alone** is enough for simple chat-like generate.  
**Orchestrator** is what makes multi-step “agent” behavior (plan, tools, approvals).

---

## 6. What you need installed

### Minimum (no Docker)

1. **Git** — already how you got the repo  
2. **Go** (version roughly 1.22+; this repo targets modern Go)  
   - Check: `go version`  
   - If missing, install from https://go.dev/dl/  
3. A terminal (bash, zsh, etc.)

Optional but nice:

- **make** (`make --version`) — runs Makefile targets  
  - Without make, you can still copy the commands from the Makefile by hand

### Full local stack

4. **Docker** + **Docker Compose**  
   - Check: `docker version` and `docker compose version`

### You do **not** need

- A paid OpenAI/Anthropic key for the beginner path  
- Kubernetes for local learning  
- Deep knowledge of Postgres SQL

---

## 7. Step-by-step: try Atlas with zero Docker

This uses the **fake model**. Two terminals.

### Terminal A — start the gateway

From the **repository root** (the folder that contains `Makefile` and `README.md`):

```bash
cd /path/to/Project-Atlas

# Ensure Go is on PATH (Linux often needs this)
export PATH=$PATH:/usr/local/go/bin

make gateway-run
```

What this does:

- Starts the inference gateway on `http://127.0.0.1:8080`
- Uses the **fake** provider (no real model download)
- Leaves the process running in the foreground (Ctrl+C stops it)

You should see a log line about listening on an address.

### Terminal B — talk to it

```bash
cd /path/to/Project-Atlas
export PATH=$PATH:/usr/local/go/bin

# Is the gateway up?
make cli-health

# One-shot generate
make cli-generate MSG='hello atlas'
```

Or interactive CLI:

```bash
make cli-run
```

### Optional: start the orchestrator (also no Docker)

Terminal C:

```bash
cd /path/to/Project-Atlas
export PATH=$PATH:/usr/local/go/bin

# Gateway must already be running on 8080
make orch-run
```

Then, in another terminal, create a run:

```bash
curl -sS -X POST http://127.0.0.1:8090/v1/runs \
  -H 'Content-Type: application/json' \
  -d '{"goal":"calculate 2+2"}'
```

You’ll get JSON back with a `run_id` and planned `steps`. Start it:

```bash
# Replace RUN_ID with the id from the previous response
curl -sS -X POST http://127.0.0.1:8090/v1/runs/RUN_ID/start
```

Check health:

```bash
curl -sS http://127.0.0.1:8090/healthz
curl -sS http://127.0.0.1:8090/openapi.yaml | head
```

---

## 8. Step-by-step: full local stack with Docker

Use this when you want Postgres, Redis, Qdrant, both services, etc., without installing them by hand.

### Beginner / offline (fake model — recommended first)

```bash
cd /path/to/Project-Atlas/deploy
cp -n .env.example .env   # only if .env does not exist yet
docker compose -f docker-compose.yml -f docker-compose.fake.yml up --build
```

Or from repo root:

```bash
make compose-fake
```

Wait until containers are healthy (`docker compose ps`).

Then:

| URL | What |
|-----|------|
| http://localhost:8080/healthz | Gateway alive? |
| http://localhost:8090/healthz | Orchestrator alive? |
| http://localhost:5432 | Postgres (app credentials from `.env`) |

### Stop

```bash
cd deploy
docker compose down
```

To also wipe databases:

```bash
docker compose down -v
```

### Full stack with Ollama (real local models)

```bash
cd deploy
cp -n .env.example .env
docker compose up -d
# Later: pull a model inside Ollama if needed
```

This is heavier (downloads models). Prefer fake mode while learning.

---

## 9. What each major piece does

### 9.1 Inference gateway (`services/inference-gateway`)

**Job:** be the only door to models.

Typical endpoints:

| Method + path | Meaning |
|---------------|---------|
| `GET /healthz` | Liveness (“process is up”) |
| `GET /readyz` | Readiness (“safe to send traffic”) |
| `POST /v1/generate` | Ask the model for a completion |
| `POST /v1/embeddings` | Turn text into vectors |
| `GET /openapi.yaml` | API description |

**Why separate from the orchestrator?**  
So many products (CLI, future UI, orchestrator) share one model access layer with logging, retries, and provider swapping.

### 9.2 Agent orchestrator (`services/agent-orchestrator`)

**Job:** turn a **goal** string into a **run** with steps.

Important endpoints (simplified):

| Method + path | Meaning |
|---------------|---------|
| `POST /v1/runs` | Create a planned run |
| `POST /v1/runs/{id}/start` | Execute the plan |
| `GET /v1/runs/{id}` | Get status/result |
| `GET /v1/runs/{id}/events` | Timeline of what happened |
| `POST /v1/runs/{id}/approve` | Human allows a dangerous tool step |
| `POST /v1/runs/{id}/reject` | Human rejects |
| `POST /v1/runs/{id}/cancel` | Stop |
| `POST /v1/knowledge/ingest` | Add a document for search |
| `GET /v1/knowledge/docs` | List documents |
| `DELETE /v1/knowledge/docs/{id}` | Delete a document |
| `GET /v1/knowledge/status` | Corpus counts |
| `POST /v1/knowledge/search` | Search + SLA metrics |
| `POST /v1/prompts` | Publish a prompt (needs workspace header) |
| `GET /v1/prompts` | List prompts for a workspace |
| `GET /v1/plugins` | List registered remote tools |
| `POST /v1/plugins` | Register a remote HTTP tool |

**Workspace header (multi-tenant prompts):**

```bash
curl -sS -X POST http://127.0.0.1:8090/v1/prompts \
  -H 'Content-Type: application/json' \
  -H 'X-Workspace-ID: my-team' \
  -d '{"id":"demo.hello","version":"1","status":"active","text":"Hello {{name}}"}'
```

Without `X-Workspace-ID`, prompt admin APIs refuse the request. That is intentional isolation practice.

### 9.3 Atlas CLI (`clients/atlas-cli`)

Terminal program for humans:

- Banner + chat against the gateway  
- Health / generate helpers via `make cli-*`

### 9.4 Eval (`tools/atlas-eval`)

Reads JSON **fixtures** under `fixtures/eval/v1/` and scores them:

- Did the planner pick a calculator for a math goal?  
- Did the right document rank first in search?  
- Did a tool return the expected value?  
- Optional: **LLM-as-judge** cases (model scores quality)

```bash
make eval-gate          # fails CI if fixtures fail
make eval               # run sample suite, write report
```

Judge offline without a real model:

```bash
cd tools/atlas-eval
go run . -dataset ../../fixtures/eval/v1/sample_with_judge.json \
  -judge-stub pass -out /tmp/judge-report.json
```

### 9.5 Bench (`tools/atlas-bench`)

Hits a real HTTP path many times and prints **JSON metrics** (latency, throughput).

```bash
make bench
# or with gateway down, unit tests still prove the tool works via a tiny fake server
```

### 9.6 Deploy (`deploy/`)

| Path | Purpose |
|------|---------|
| `docker-compose*.yml` | Local multi-container environment |
| `k8s/` | Starter Kubernetes YAML (images, probes, resources) |
| `grafana/` | Dashboards including load/bench panels |
| `README.md` | Production hardening notes |

---

## 10. How a “run” works (agent orchestrator)

Think of a **run** as a ticket:

1. **Create** — “I want to calculate 12+30”  
2. **Plan** — software invents steps (prepare context → maybe tool → call model → finalize)  
3. **Start** — execute steps in dependency order  
4. **Maybe pause** — if a tool needs human approval  
5. **Finish** — `succeeded` / `failed` / `cancelled`  
6. **Inspect** — get run + events

```text
CreateRun(goal)
    │
    ▼
Planner builds steps (rule planner or LLM planner)
    │
    ▼
Save run as pending
    │
    ▼
StartRun
    │
    ├─► context step (history + memory + RAG snippets)
    ├─► tool step (Invoke tool registry; authz; approval)
    ├─► model step (call inference gateway)
    └─► finalize step (produce result)
```

Statuses you might see:

| Status | Meaning |
|--------|---------|
| `pending` | Planned, not started |
| `running` | Executing |
| `waiting_approval` | Human must approve a tool |
| `succeeded` | Done OK |
| `failed` | Error |
| `cancelled` | Stopped on purpose |

---

## 11. Tools, prompts, knowledge, and eval (plain language)

### Tools

A **tool** is a named skill with:

- Input schema (required fields)  
- Optional approval requirement  
- Execute function  

Built-ins include: `echo`, `calculator`, `http_get`, `danger_echo` (needs approval).

Also available:

- **Remote HTTP tools** — POST JSON to a URL you configure (plugin style)  
- **GitHub-style / Slack-style tools** — real HTTP shape, but tests use a fake server (no need for live credentials)

**Authz (authorization):** an allowlist can block tools so their body **never runs** if not allowed.

### Prompts

Prompts are versioned templates, e.g. system instructions for planning.

Sources:

1. Files under `prompts/` (reloadable without restart)  
2. Published rows in Postgres / memory store (multi-tenant by workspace)

### Knowledge (RAG + corpus)

1. **Ingest** documents  
2. **Search** by meaning/keywords  
3. **List / delete / status** for operations  
4. Search returns **SLA metrics** (how long it took, how many hits)

Even when “Qdrant” is configured, a local fallback keeps a copy so list/delete still work when Qdrant is down.

### Eval vs bench

| | Eval | Bench |
|---|------|-------|
| Question | “Is behavior correct?” | “Is it fast enough / stable?” |
| Input | Fixtures with expectations | Many HTTP requests |
| Output | Pass/fail report | Latency & throughput JSON |

---

## 12. Common commands cheat sheet

Run from **repo root** unless noted. Ensure Go is on `PATH`.

```bash
export PATH=$PATH:/usr/local/go/bin

make help              # list targets

# Tests
make test              # gateway + orchestrator + cli + eval + bench tests
make orch-test         # only orchestrator
make eval-gate         # must-pass eval fixtures
make bench-test        # bench unit tests

# Run locally (no Docker)
make gateway-run       # :8080 fake model
make orch-run          # :8090 (expects gateway)
make cli-health
make cli-generate MSG='hi'
make cli-run

# Docker
make compose-fake      # full stack, fake model

# Load metrics (needs something listening on GATEWAY_URL)
make bench
```

### Useful curl patterns

```bash
# Gateway
curl -sS http://127.0.0.1:8080/healthz

# Orchestrator
curl -sS http://127.0.0.1:8090/healthz
curl -sS http://127.0.0.1:8090/openapi.yaml | head -20

# Create + start a run
curl -sS -X POST http://127.0.0.1:8090/v1/runs \
  -H 'Content-Type: application/json' \
  -d '{"goal":"echo hello"}'
# then POST .../v1/runs/<id>/start
```

---

## 13. How to read Go code without panic

You are not expected to write production Go yet. When you open a file:

### File roles (orchestrator examples)

| File | Role |
|------|------|
| `cmd/agent-orchestrator/main.go` | Program entry: wire config, start HTTP server |
| `handler.go` | URL routing: which path calls which function |
| `service.go` | Application use-cases: CreateRun, StartRun, … |
| `executor.go` | Runs steps of a plan |
| `tools.go` | Tool registry + built-in tools |
| `rag.go` | Document ingest/search |
| `prompt_registry.go` | Load/resolve prompt templates |
| `*_test.go` | Automated tests (run with `go test`) |

### Patterns you will see

```go
// A function attached to a type (method)
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // h = handler, w = response writer, r = incoming request
}

// Error handling is explicit — no hidden exceptions
if err != nil {
    return err
}
```

### Tests drive the real code

Good tests in this repo:

- Call the **real** `Handler` / `ToolRegistry.Invoke` / `Judge`  
- Do **not** re-implement the feature inside the test  
- Prefer `httptest` (fake HTTP server in memory) over live internet

---

## 14. How testing works here

```bash
# From repo root
make orch-test

# Or inside a module
cd services/agent-orchestrator
go test ./...
```

Meaning of output:

| Line | Meaning |
|------|---------|
| `PASS` / `ok` | Tests succeeded |
| `FAIL` | Something broke |
| `--- SKIP:` | Test skipped (e.g. Postgres not running) |

**Eval gate** (`make eval-gate`) is special: it fails the pipeline if quality fixtures fail—like a quality seatbelt.

---

## 15. Where docs and “status” live

| Question | File |
|----------|------|
| What should we build overall? | `docs/product/VISION.md`, `GOALS.md` |
| What words mean (domain)? | `docs/domain/` |
| What’s done / next / icebox? | `management/BACKLOG.md` |
| Milestone status M0–M5 | `management/MILESTONES.md` |
| Why a design choice? | `docs/adr/` |
| Ops “what if production breaks?” | `docs/runbooks/` |
| Security self-review | `docs/ops/SECURITY_AUDIT.md` |
| Multi-region (design only) | `docs/architecture/multi-region.md` |
| Older short beginner notes | `docs/engineering/BEGINNER_BUILD_RECORD.md` |

**Icebox** means: written down so we don’t forget, but **not** scheduled as current work (e.g. multi-region HA mesh, WASM guest runtime, multi-agent swarms).

---

## 16. What is intentionally *not* built yet

These are **not** missing by accident; they are deferred:

| Topic | Status |
|-------|--------|
| Full multi-region active-active HA | Icebox (design doc only) |
| WASM sandbox for untrusted plugins | Icebox (HTTP plugins exist) |
| Autonomous multi-agent swarms | Icebox |
| Fancy web UI | Not the current focus |
| Third-party SOC2 / pen-test certificate | Outside this repo’s engineering scope |
| Live production GitHub/Slack credentials in CI | Not required; tools tested with fake servers |

---

## 17. If something breaks

### `go: command not found`

```bash
export PATH=$PATH:/usr/local/go/bin
go version
```

Install Go from https://go.dev/dl/ if still missing.

### `connection refused` on 8080 / 8090

The service is not running. Start `make gateway-run` and/or `make orch-run`, or start Docker Compose.

### Docker “permission denied” or daemon not running

Start Docker Desktop / the docker service. On Linux you may need to be in the `docker` group.

### Postgres tests skip

Normal if Postgres is not up. Core unit tests still run in memory. Compose starts Postgres for the full stack.

### Orchestrator run fails with gateway errors

Orchestrator model steps need the gateway. Start gateway first, or set:

```bash
export ATLAS_INFERENCE_GATEWAY_URL=http://127.0.0.1:8080
```

### Prompt API returns 401

Add header:

```text
X-Workspace-ID: some-workspace-name
```

### Port already in use

Something else is bound to 8080/8090. Stop the other process or change:

```bash
ATLAS_INFERENCE_ADDR=127.0.0.1:18080 make gateway-run
```

---

## 18. Suggested learning path

Do these in order; each step builds on the last.

| Step | Action | Success looks like |
|------|--------|--------------------|
| 1 | Read this guide (you are here) | You can explain gateway vs orchestrator |
| 2 | `make gateway-run` + `make cli-generate MSG='hi'` | You get a fake model reply |
| 3 | `make orch-run` + create/start a run with curl | JSON shows steps and a result |
| 4 | `make orch-test` | All (or almost all) tests PASS |
| 5 | Read `services/agent-orchestrator/handler.go` top-to-bottom | You recognize URL routes |
| 6 | Read one runbook in `docs/runbooks/` | You see how ops thinks |
| 7 | Optional: `make compose-fake` | Full stack health endpoints work |
| 8 | Skim `management/BACKLOG.md` | You know what’s done vs icebox |

---

## Appendix A — Environment variables (most useful)

| Variable | Used by | Meaning |
|----------|---------|---------|
| `ATLAS_INFERENCE_ADDR` | Gateway | Listen address (default often `:8080`) |
| `ATLAS_INFERENCE_PROVIDER` | Gateway | `fake` / ollama / … |
| `ATLAS_INFERENCE_MODEL` | Gateway / orchestrator | Model name string |
| `ATLAS_INFERENCE_GATEWAY_URL` | Orchestrator | Where to call the gateway |
| `ATLAS_ORCHESTRATOR_ADDR` | Orchestrator | Listen address (`:8090`) |
| `ATLAS_ORCHESTRATOR_DATABASE_URL` | Orchestrator | Postgres connection string |
| `ATLAS_PROMPTS_DIR` | Orchestrator | Folder of prompt files |
| `ATLAS_REQUIRE_WORKSPACE` | Orchestrator | `1` = require `X-Workspace-ID` on most routes |
| `ATLAS_REDIS_URL` | Orchestrator | Optional short-term memory |
| `ATLAS_QDRANT_URL` | Orchestrator | Vector DB URL |
| `ATLAS_GITHUB_API_URL` | Orchestrator tools | Override GitHub API base (tests/local) |
| `ATLAS_SLACK_API_URL` | Orchestrator tools | Override Slack API base |

---

## Appendix B — Mental model of “documentation first”

This repo often writes docs **before** or **with** code:

1. Product says *what* and *why*  
2. Architecture / ADR says *how at a high level*  
3. Contracts/schemas say *shapes of data*  
4. Tasks track *implementable chunks*  
5. Tests prove *shipped code*  
6. Runbooks/ops cover *when it breaks*

That is why you see many Markdown files. They are the product memory—not decoration.

---

## Appendix C — Where to ask next

1. **This guide** for orientation  
2. **Service README** for that service’s details:  
   - `services/inference-gateway/README.md`  
   - `services/agent-orchestrator/README.md`  
3. **`deploy/README.md`** for Compose and production hardening  
4. **`management/BACKLOG.md`** for remaining product work  

If you only remember three ideas:

1. **Gateway** = models  
2. **Orchestrator** = plans + tools + runs  
3. **Fake mode** = learn without downloading AI models  

---

*End of Complete Beginner Guide.*
