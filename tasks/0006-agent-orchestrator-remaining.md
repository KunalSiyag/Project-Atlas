# Task 0006 — Agent orchestrator remaining implementation (post-contracts)

## Goal
Track product/engineering work after documentation and contracts match the shipped M2/M3 slice.

## Inputs
- [0005](./0005-agent-orchestrator-docs-and-contracts.md) (done)
- [Service README](../services/agent-orchestrator/README.md)
- [Management backlog](../management/BACKLOG.md)

## Outputs / files
- List-events HTTP + OpenAPI serve + workspace middleware tests
- Compose orchestrator service with Postgres long memory
- Expanded eval fixture `fixtures/eval/v1/extended.json`

## Dependencies
- Accepted contracts from task 0005
- Inference gateway (M1) available for model steps

## Work status

| ID | Item | Status |
|----|------|--------|
| 0006-a | HTTP list-events endpoint (`GET /v1/runs/{id}/events`) using `ListEvents` | Done |
| 0006-b | Serve orchestrator OpenAPI at `GET /openapi.yaml` (parity with gateway) | Done |
| 0006-c | Workspace middleware enforcement tests for `ATLAS_REQUIRE_WORKSPACE` | Done |
| 0006-d | Postgres long-memory path always-on in Compose profile | Done |
| 0006-e | Expanded eval suites beyond smoke fixture | Done |
| 0006-f | Production multi-region / HA orchestration | Icebox (see backlog) — not a multi-cluster mesh |
| 0006-g | WASM / remote tool plugins (WASM guest runtime) | Icebox — HTTP `RegisterDescriptor` / `RemoteHTTPTool` shipped as M3 residual, not WASM |

## Done in prior slices (reference)

| Item | Where |
|------|--------|
| Create/start/get/approve/reject/cancel HTTP | `handler.go` |
| Plan executor + retries + cancel | `executor.go` |
| Tool SDK built-ins | `tools.go` |
| Memory + RAG ports | `memory.go`, `rag.go` |
| Postgres run store | `postgres_store.go` |
| LLM + rule planners | `planner.go` |
| Context builder | `context_builder.go` |
| Offline eval harness | `tools/atlas-eval` |

## Delivered (0006-a…e)

| Item | Where |
|------|--------|
| List events | `Service.ListEvents`, `GET /v1/runs/{id}/events`, `handler_events_test.go` |
| OpenAPI serve | `openapi.go` embed, `GET /openapi.yaml` (+ `.yml`, `/v1/openapi.yaml`) |
| Workspace middleware | `RequireWorkspace` in `middleware.go` + `middleware_test.go`; main uses package helper |
| Compose long memory | `deploy/docker-compose.yml` `agent-orchestrator` with `ATLAS_ORCHESTRATOR_DATABASE_URL` → Postgres |
| Expanded eval | `fixtures/eval/v1/extended.json` + `TestRunExtendedFixture` |

## Acceptance criteria
- [x] Planned rows 0006-a…e marked Done with landing code
- [x] Icebox items stay out of sprint until product prioritizes them

## Status
Partial complete (a–e Done; f–g Icebox)

## Test requirements
```bash
export PATH=$PATH:/usr/local/go/bin
cd services/agent-orchestrator && go test ./...
cd tools/atlas-eval && go test ./...
```
