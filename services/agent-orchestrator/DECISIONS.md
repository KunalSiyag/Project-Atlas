# Agent Orchestrator Decisions

## Purpose
Track service-local decisions that do not require a repository-level ADR, and link accepted ADRs that bound this service.

## Scope
Applies to Agent Orchestrator Service planning and shipped slice.

## Dependencies
- [Service README](./README.md)
- [Repository ADR Index](../../docs/adr/README.md)

## Owner
Atlas Agent Runtime Team

## Status
Accepted

## Last Updated
2026-08-05

## Related Documents
- [ADR 0010 Orchestration State Machine](../../docs/adr/0010-orchestration-state-machine.md)
- [ADR 0009 Context Builder Token Budgeting](../../docs/adr/0009-context-builder-token-budgeting.md)
- [RFC 0001 Tool SDK Interface](../../docs/rfcs/0001-tool-sdk-interface.md)

## Decisions

### D1 — Run/step state machine (repository ADR)
**Decision:** Run and step statuses, approval pause, cancel, and lifecycle event types are fixed by [ADR 0010](../../docs/adr/0010-orchestration-state-machine.md).  
**Rationale:** Executor, API OpenAPI, and stores must share one enum set.

### D2 — Persistence via ports
**Decision:** Application code depends on `RunStore`, `MemoryStore`, and `Retriever` interfaces; default in-memory backends, optional Postgres/Redis/Qdrant via env.  
**Rationale:** Matches hexagonal style (ADR 0003); enables tests without Docker.

### D3 — Inference only through gateway client
**Decision:** Model steps call the inference gateway HTTP client, never a vendor SDK inside the orchestrator.  
**Rationale:** Provider independence (ADR 0002).

### D4 — Tools implement RFC 0001 Tool interface
**Decision:** Built-ins (`echo`, `calculator`, `http_get`, `danger_echo`) register as `Tool` values on `ToolRegistry`; invoke path validates schema, timeout, and approval.  
**Rationale:** Eval and future plugins share error classes and discovery.
