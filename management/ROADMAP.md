# Roadmap

## Purpose
Describe the multi-milestone path from documentation foundation to production-ready platform capabilities.

## Scope
Strategic roadmap only. Ticket-level detail lives in `tasks/` and `BACKLOG.md`.

## Dependencies
- [Milestones](./MILESTONES.md)
- [Product Goals](../docs/product/GOALS.md)
- [Non-Goals](../docs/product/NON_GOALS.md)

## Owner
Atlas Core Engineering

## Status
Accepted

## Last Updated
2026-08-03

## Related Documents
- [Sprints](./SPRINTS.md)
- [Backlog](./BACKLOG.md)

## Direction
Atlas is built documentation-first, then infrastructure, then intelligence, tools, knowledge, and production hardening. The LLM remains a replaceable dependency throughout.

## Milestone Ladder
1. **M0 — Foundation** (current target for this goal): product constitution, domain language, subsystem architecture, foundation ADRs, management/contracts/schemas/tasks navigation.
2. **M1 — Core Platform**: gateway, inference gateway, configuration, logging, metrics, local Compose.
3. **M2 — Intelligence**: planner, context builder, memory engine.
4. **M3 — Tools**: tool runtime, plugin SDK patterns, workflow execution with approvals.
5. **M4 — Knowledge**: RAG, embeddings, document ingest, search.
6. **M5 — Production**: deeper security, evaluation gates, deployment hardening, dashboards.

## Near-Term Themes
- Keep every paid capability abstractable to a free local adapter.
- Contracts and schemas before service code.
- First implementation service: **inference gateway** (shared dependency), then orchestrator paths.

## Explicitly Later
Autonomous coding agents, self-improving loops, multi-agent swarms, multi-region active-active.
