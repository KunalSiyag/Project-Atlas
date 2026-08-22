# Task 0005 — Agent orchestrator docs, contracts, and task breakdown

## Goal
Align agent-orchestrator service pack documentation and contracts with the **implemented** HTTP surface, state machine, persistence ports, and lifecycle events; close remaining service TODO checkboxes and RFC 0001 acceptance leftovers.

## Inputs
- Live code: `handler.go`, `model.go`, `store.go`, `executor.go`, `tools.go`, `memory.go`, `rag.go`
- Draft `TODO.md` / draft PRD and API concepts

## Outputs / files
- `services/agent-orchestrator/PRD.md` (Accepted)
- `services/agent-orchestrator/API.md` + `openapi.yaml`
- `docs/adr/0010-orchestration-state-machine.md` + ADR index row
- `services/agent-orchestrator/DATA_MODEL.md` (ports)
- `services/agent-orchestrator/events.schema.json`
- `services/agent-orchestrator/DECISIONS.md` + `TODO.md` all checked
- RFC 0001 acceptance closed; eval `tool_success` case type
- This task file + [0006](./0006-agent-orchestrator-remaining.md) remaining work

## Dependencies
- Working orchestrator package tests
- RFC 0001 interface direction (Accepted)

## Acceptance criteria
- [x] PRD status Accepted and matches shipped slice
- [x] OpenAPI lists create/start/get/approve/reject/cancel, knowledge, memory, healthz/readyz
- [x] ADR 0010 Decision section covers run/step SM + approval/cancel
- [x] DATA_MODEL documents `RunStore` / `MemoryStore` / `Retriever`
- [x] Event schema lists real executor/service event types
- [x] Built-ins use explicit `Tool` interface; eval can assert tool success by name
- [x] `go test` in agent-orchestrator (and atlas-eval when fixtures change) passes

## Status
Done (2026-08-05)

## Test requirements
```bash
export PATH=$PATH:/usr/local/go/bin
cd services/agent-orchestrator && go test ./...
cd tools/atlas-eval && go test ./...
```
