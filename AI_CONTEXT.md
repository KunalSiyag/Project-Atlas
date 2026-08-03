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
M0 Foundation Complete

## Last Updated
2026-08-03

## Related Documents
- [Product Index](./docs/product/README.md)
- [Domain Index](./docs/domain/README.md)
- [Architecture Overview](./docs/architecture/OVERVIEW.md)
- [ADR Index](./docs/adr/README.md)
- [Management](./management/README.md)
- [RFC Process](./docs/rfcs/README.md)
- [Roadmap](./management/ROADMAP.md)

## What Atlas is
An enterprise-oriented **AI agent platform** (not a single chatbot demo). The LLM is one dependency behind interfaces. Core concerns: workspace auth, conversations, context building, planning/workflows, tool runtime, model routing, guardrails, evaluation, observability, multi-store storage.

## Repository philosophy
- Documentation-first modular monorepo ([ADR 0001](./docs/adr/0001-documentation-first-modular-monorepo.md))
- Provider independence ([ADR 0002](./docs/adr/0002-provider-independence.md))
- Hexagonal modular services ([ADR 0003](./docs/adr/0003-architecture-style.md))
- Contracts and schemas before implementation code

## Operating Model
Atlas is documentation-first. Do not implement features until requirements, product design, architecture, ADRs, API contracts, data models, tasks, and test plans exist for the relevant scope.

## M0 status and next steps
- **M0 is complete:** product constitution, domain language, subsystem architecture, ADRs 0001–0008, management layer, contracts/schemas/tasks indexes.
- **Do not** treat missing Go/application code as incomplete M0.
- **Next (M1):** inference gateway service pack + contracts depth + local Compose data plane; then first code behind `ModelProvider`.
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
