# Agent Orchestrator Service

## Purpose
Coordinate agent execution: plan DAGs, tools with approvals, memory, light RAG, and inference-gateway calls.

## Status
M2/M3 runnable slice (2026-08-05)

## Features

| Capability | How |
|------------|-----|
| Dynamic plan | LLMPlanner (gateway JSON) + RulePlanner fallback; keyword rules without gateway (`calculate`, `echo`, `http`, `danger`) |
| Tool steps | `echo`, `calculator`, `http_get`, `danger_echo` (approval) |
| Schema + timeout | `ValidateToolInput` + per-tool deadline |
| Approvals | Run status `waiting_approval`; `POST /v1/runs/{id}/approve` or `/reject` |
| Memory | Short-term per run + long-term Postgres/in-memory facts → context builder |
| RAG | Ingest + top-k retrieve (Qdrant if up, else in-process vectors) |
| Gateway | Model steps call `POST /v1/generate` |

## Run

```bash
# gateway
make gateway-run

# orchestrator
ATLAS_INFERENCE_GATEWAY_URL=http://127.0.0.1:8080 \
ATLAS_ORCHESTRATOR_DATABASE_URL=postgres://atlas:atlas-dev-password@127.0.0.1:5432/atlas?sslmode=disable \
ATLAS_QDRANT_URL=http://127.0.0.1:6333 \
make orch-run

# CLI
cd clients/atlas-cli && go build -o atlas .
./atlas run "calculate 21+21"
./atlas run -auto-approve "danger echo ship payload"
```

## HTTP API

- `POST /v1/runs` `{ "goal": "..." }`
- `POST /v1/runs/{id}/start`
- `POST /v1/runs/{id}/approve` | `/reject` | `/cancel`
- `GET /v1/runs/{id}`
- `POST /v1/knowledge/ingest` `{ "title", "content" }`
- `POST /v1/memory` long-term fact

## Env

| Variable | Default |
|----------|---------|
| `ATLAS_ORCHESTRATOR_ADDR` | `:8090` |
| `ATLAS_INFERENCE_GATEWAY_URL` | `http://127.0.0.1:8080` |
| `ATLAS_ORCHESTRATOR_DATABASE_URL` | empty → memory runs |
| `ATLAS_QDRANT_URL` | `http://127.0.0.1:6333` |
| `ATLAS_REDIS_URL` | empty → in-process short-term; `redis://host:6379/0` for Redis |
| `ATLAS_REQUIRE_WORKSPACE` | empty; `1` requires `X-Workspace-ID` |

## Tests

```bash
go test ./...
```
