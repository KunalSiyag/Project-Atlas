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
M0 Foundation Complete

## Last Updated
2026-08-03

## Related Documents
- [AI Context](./AI_CONTEXT.md)
- [Product Vision](./docs/product/VISION.md)
- [Product Index](./docs/product/README.md)
- [Domain Language](./docs/domain/README.md)
- [Architecture Overview](./docs/architecture/OVERVIEW.md)
- [ADR Index](./docs/adr/README.md)
- [Management](./management/README.md)
- [Engineering Principles](./docs/engineering/PRINCIPLES.md)
- [Documentation Standards](./docs/engineering/DOCUMENTATION_STANDARDS.md)
- [Agent Orchestrator Service](./services/agent-orchestrator/README.md)

## Mission
Atlas is not another chatbot. It is a modular platform for production agent systems with explicit contracts, observable execution, testable components, and vendor-independent provider adapters.

The LLM is a dependency. The product is everything around it: auth, gateway, context building, planning, workflows, tools, model routing, guardrails, evaluation, and observability.

## Foundation status (M0)
**M0 documentation foundation is complete.** The repository is set up as an engineering organization: product constitution, domain language, subsystem architecture, foundation ADRs, management tracking, and contracts/schemas/tasks indexes.

**Not required for M0 (and not present as application code):** Go services, live LLM wiring, Docker Compose runtime, or Kubernetes.

**What comes next (M1):** deepen inference-gateway docs and contracts, local data-plane Compose, then first implementation behind provider interfaces. See [management/ROADMAP.md](./management/ROADMAP.md) and [management/BACKLOG.md](./management/BACKLOG.md).

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
  benchmarks/       Benchmark plans and results
  domain/           Ubiquitous language
  engineering/      Standards and principles
  kb/               Knowledge base notes
  playbooks/        Repeatable engineering procedures
  product/          Product constitution
  rfcs/             RFC process and proposals
  runbooks/         Operational response guides
management/         Roadmap, milestones, sprints, backlog, changelog
contracts/          Provider and control-plane contracts
schemas/            JSON schemas for core entities
tasks/              Numbered engineering tasks
services/
  agent-orchestrator/  First service documentation shell
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
