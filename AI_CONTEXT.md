# AI Context

## Purpose
Provide future AI assistants with the minimum durable context required to work safely in Project Atlas.

## Scope
Applies to repository-wide planning, documentation, architecture, and implementation assistance.

## Dependencies
- [README](./README.md)
- [Engineering Principles](./docs/engineering/PRINCIPLES.md)
- [Documentation Standards](./docs/engineering/DOCUMENTATION_STANDARDS.md)

## Owner
Atlas Core Engineering

## Status
M0–M1 + M2/M3 orchestrator slice shipped · M3–M5 residual / production still open

## Last Updated
2026-08-05

## Related Documents
- [Beginner Build Record](./docs/engineering/BEGINNER_BUILD_RECORD.md)
- [Product Index](./docs/product/README.md)
- [Domain Index](./docs/domain/README.md)
- [Architecture Overview](./docs/architecture/OVERVIEW.md)
- [ADR Index](./docs/adr/README.md)
- [Management](./management/README.md)
- [RFC Process](./docs/rfcs/README.md)
- [Roadmap](./management/ROADMAP.md)
- [Inference Gateway](./services/inference-gateway/README.md)
- [Atlas CLI](./clients/atlas-cli/README.md)

## What Atlas is
An enterprise-oriented **AI agent platform** (not a single chatbot demo). The LLM is one dependency behind interfaces. Core concerns: workspace auth, conversations, context building, planning/workflows, tool runtime, model routing, guardrails, evaluation, observability, multi-store storage.

## Repository philosophy
- Documentation-first modular monorepo ([ADR 0001](./docs/adr/0001-documentation-first-modular-monorepo.md))
- Provider independence ([ADR 0002](./docs/adr/0002-provider-independence.md))
- Hexagonal modular services ([ADR 0003](./docs/adr/0003-architecture-style.md))
- Contracts and schemas before implementation code

## Operating Model
Atlas is documentation-first. Do not implement features until requirements, product design, architecture, ADRs, API contracts, data models, tasks, and test plans exist for the relevant scope.

## Status and next steps
- **M0 is complete:** product constitution, domain language, subsystem architecture, ADRs 0001–0008 (+ later ADRs), management layer, contracts/schemas/tasks indexes.
- **M1 largely shipped:** `services/inference-gateway` (fake/Ollama/OpenAI-style chat, embeddings, OTEL, OpenAPI), `deploy/` Compose, `clients/atlas-cli`, root `Makefile`.
- **M2/M3 slice shipped:** agent-orchestrator (plan DAG, tools+approvals, memory/RAG ports, Postgres runs, list-events, OpenAPI), prompt registry, Tool SDK built-ins, `tools/atlas-eval`, streaming SSE and Compose OTEL/Grafana as available.
- **Still open (not done):** M3 plugin SDK beyond in-process + tool authz; M4 corpus ops / search SLAs; M5 eval CI gates, security hardening, deploy docs, runbooks, load/bench dashboards; enterprise auth; web UI. Icebox: multi-region HA, WASM/remote plugins, multi-agent swarms.
- Teach-yourself path: [docs/engineering/COMPLETE_BEGINNER_GUIDE.md](./docs/engineering/COMPLETE_BEGINNER_GUIDE.md) (no Go experience required).
- Track work in [management/BACKLOG.md](./management/BACKLOG.md) and [tasks/](./tasks/README.md).

## Core Constraints
- Keep business logic independent from vendors.
- Prefer small documents over large context-heavy files.
- Use service-local documentation for service-specific decisions.
- Do not make silent architectural decisions.
- Update documentation with every meaningful change.
- Prefer free/local adapters for development; commercial providers optional.

## Recommended Context Loading
For repository-wide work, read only:
1. `README.md`
2. `AI_CONTEXT.md`
3. `docs/engineering/PRINCIPLES.md`
4. The specific product, domain, architecture, ADR, or service document being changed

For product work, read:
1. `docs/product/README.md` then the specific constitution doc

For domain work, read:
1. `docs/domain/README.md` then the entity doc

For architecture work, read:
1. `docs/architecture/OVERVIEW.md` then the subsystem doc
2. Related ADR(s)

For service work, read only:
1. `services/<service>/AI_CONTEXT.md`
2. `services/<service>/PRD.md`
3. `services/<service>/API.md`
4. `services/<service>/README.md`

## Non-goals reminder
No autonomous coding agents, self-improving loops, or multi-agent swarms as near-term defaults. See `docs/product/NON_GOALS.md`.
