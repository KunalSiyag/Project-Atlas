# Agent Orchestrator PRD

## Purpose
Define product requirements for the agent orchestration service slice shipped in this repository.

## Scope
Covers the runnable M2/M3 orchestrator: plan DAGs, tools with approvals, memory, light RAG, and inference-gateway model calls. Does not cover multi-agent swarms, web UI, WASM plugins, or multi-region production hardening.

## Dependencies
- [Product Vision](../../docs/product/VISION.md)
- [Service README](./README.md)
- [Architecture Overview](../../docs/architecture/OVERVIEW.md)

## Owner
Atlas Product Engineering

## Status
Accepted

## Last Updated
2026-08-05

## Related Documents
- [API](./API.md)
- [OpenAPI](./openapi.yaml)
- [Data Model](./DATA_MODEL.md)
- [Test Plan](./TEST_PLAN.md)
- [TODO](./TODO.md)
- [ADR 0010 Orchestration State Machine](../../docs/adr/0010-orchestration-state-machine.md)

## Goal
Provide a reliable orchestration layer for executing agent workflows with observable state, provider-independent integrations, and deterministic failure handling.

## Reasoning
Agent systems fail in complex ways. The platform needs orchestration boundaries before integrating providers or adding advanced agent behavior. Implementation evidence now exists: HTTP control surface, `RunStore` backends, plan executor, tool registry (RFC 0001), and eval fixtures.

## Deliverables (shipped slice)
- Execution lifecycle: run statuses `pending` → `running` → terminal (`succeeded` | `failed` | `cancelled`) with optional `waiting_approval`.
- HTTP API for create/start/get/approve/reject/cancel runs, knowledge ingest, memory put, health probes (see [API](./API.md) / [openapi.yaml](./openapi.yaml)).
- Data model for runs, steps, and lifecycle events (see [DATA_MODEL](./DATA_MODEL.md)).
- Plan graph executor with context/model/tool/finalize step kinds, retries, cancellation.
- Built-in tools via explicit `Tool` interface: `echo`, `calculator`, `http_get`, `danger_echo`.
- Memory (short per-run + long-term) and RAG ingest/search ports.
- Inference via gateway client (`POST /v1/generate`), not direct provider SDKs.
- Test plan coverage in package tests and offline eval harness.

## Acceptance Criteria
- [x] Service-local documentation is sufficient to operate and extend the shipped slice.
- [x] API contract documents request and response shapes for implemented routes (OpenAPI).
- [x] Failure modes are explicitly documented (invalid goal, unknown tool, gateway failure, approval deny, cancel, plan stuck).
- [x] State machine boundaries are recorded in ADR 0010.
- [x] Persistence ports match live `RunStore` / `MemoryStore` / `Retriever` surfaces.
- [x] Package tests pass for the orchestrator module.

## Non-goals (this PRD)
- Kubernetes operators, multi-region active-active, SaaS tool marketplace.
- Hot-reload prompt publish path beyond file-backed prompt registry.
- Replacing the inference gateway or embedding provider contracts.

## Risks
- Docs lagging code (mitigated by OpenAPI + DATA_MODEL aligned to handler/store).
- Coupling orchestration to a specific inference provider (mitigated: gateway client only).
- Dangerous tools without approval (mitigated: `RequiresApproval` + `waiting_approval`).

## Engineering acceptance note
Accepted 2026-08-05 against the implemented Go service under `services/agent-orchestrator` (handlers, executor, stores, tools). Product-owner sign-off beyond engineering alignment is out of band for this slice.
