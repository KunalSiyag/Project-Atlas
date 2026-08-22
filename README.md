# Project Atlas

## Purpose
Project Atlas is an enterprise-grade AI Agent Platform for building, orchestrating, evaluating, observing, and deploying production AI agents.

## Scope
This repository is the source of truth for Atlas product documentation, architecture, API contracts, data models, operational guidance, and implementation.

## Dependencies
- Git for change history and review.
- Markdown for documentation-first collaboration.
- Future services will declare their own runtime dependencies in service-local documentation.

## Owner
Atlas Core Engineering

## Status
M0–M1 foundation + gateway shipped · M2/M3 orchestrator slice shipped (runs, tools, memory/RAG, eval) · M3–M5 exit work still open (plugin SDK/authz, knowledge SLAs, production hardening)

## Last Updated
2026-08-05

## Related Documents
- [AI Context](./AI_CONTEXT.md)
- [**Complete Beginner Guide (no Go experience required)**](./docs/engineering/COMPLETE_BEGINNER_GUIDE.md)
- [Beginner build record (pointer)](./docs/engineering/BEGINNER_BUILD_RECORD.md)
- [Product Vision](./docs/product/VISION.md)
- [Product Index](./docs/product/README.md)
- [Domain Language](./docs/domain/README.md)
- [Architecture Overview](./docs/architecture/OVERVIEW.md)
- [ADR Index](./docs/adr/README.md)
- [Management](./management/README.md)
- [Engineering Principles](./docs/engineering/PRINCIPLES.md)
- [Documentation Standards](./docs/engineering/DOCUMENTATION_STANDARDS.md)
- [Inference Gateway](./services/inference-gateway/README.md)
- [Atlas CLI](./clients/atlas-cli/README.md)
- [Local deploy / Compose](./deploy/README.md)
- [Agent Orchestrator Service](./services/agent-orchestrator/README.md)

## Mission
Atlas is not another chatbot. It is a modular platform for production agent systems with explicit contracts, observable execution, testable components, and vendor-independent provider adapters.

The LLM is a dependency. The product is everything around it: auth, gateway, context building, planning, workflows, tools, model routing, guardrails, evaluation, and observability.

## Platform status
**M0 documentation foundation is complete.** Product constitution, domain language, subsystem architecture, foundation ADRs, management tracking, and contracts/schemas/tasks indexes are in place.

**M1 core platform is largely shipped:** Go **inference gateway** (chat, embeddings, OTEL), **Docker Compose** data plane, and terminal **atlas-cli**.

**M2/M3 intelligence + tools slice is shipped in-repo:** **agent-orchestrator** (plan DAG, tools/approvals, memory/RAG ports, Postgres runs, HTTP + OpenAPI, list-events), prompt registry, Tool SDK built-ins, and offline **atlas-eval** harness. Still open toward milestone exit: plugin SDK beyond in-process tools and a fuller tool **authz** story (M3), corpus ops / search SLAs (M4), eval gates in CI, security hardening, deploy docs, runbooks, and load/bench dashboards (M5). Multi-region HA and WASM/remote plugins remain icebox.

**Quick try (fake model, no Docker):** `make gateway-run` in one terminal, then `make cli-generate MSG='hello'` in another.

**New to Go / this repo?** Read the [Complete Beginner Guide](./docs/engineering/COMPLETE_BEGINNER_GUIDE.md) first (step-by-step, glossary, architecture, troubleshooting).

## Development Order
All meaningful implementation follows this sequence:

1. Requirements
2. Product Design
3. Architecture
4. Architecture Decision Record
5. API Contracts
6. Data Models
7. Tasks
8. Tests
9. Implementation
10. Documentation Update

If prerequisite documentation is missing, write or update the documentation before implementation.

## Repository Map

```text
docs/
  adr/              Architecture decision records (0001–0008 foundation set)
  architecture/     Overview + per-subsystem design
  engineering/      Standards, principles, beginner build record
  domain/ product/  Language + product constitution
management/         Roadmap, milestones, sprints, backlog, changelog
contracts/ schemas/ Provider contracts + JSON schemas
tasks/              Numbered engineering tasks
deploy/             Docker Compose local data plane (+ fake overlay)
services/
  inference-gateway/   Runnable Go inference HTTP service (chat, embeddings, OTEL)
  agent-orchestrator/  Orchestration slice (runs, tools, memory/RAG, Postgres, OpenAPI)
clients/
  atlas-cli/           Terminal console (ATLAS banner + chat + run)
tools/
  atlas-eval/          Offline eval harness
Makefile            gateway-run / cli-run / test helpers
```

## Short load path (humans and AI)
1. `README.md` (this file)
2. `AI_CONTEXT.md`
3. `docs/engineering/PRINCIPLES.md`
4. Then only the product, domain, architecture, ADR, or service docs for your task

## Current deliverables
- Product constitution and success metrics
- Domain language for core entities
- Subsystem architecture with failure modes
- Foundation ADRs including provider independence and testing philosophy
- Management and task navigation for post-M0 work
