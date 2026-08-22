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
2026-08-05

## Related Documents
- [Sprints](./SPRINTS.md)
- [Backlog](./BACKLOG.md)

## Direction
Atlas is built documentation-first, then infrastructure, then intelligence, tools, knowledge, and production hardening. The LLM remains a replaceable dependency throughout.

## Milestone Ladder
1. **M0 — Foundation** (complete): product constitution, domain language, subsystem architecture, foundation ADRs, management/contracts/schemas/tasks navigation.
2. **M1 — Core Platform** (largely complete): gateway, inference gateway, configuration, logging, metrics, local Compose.
3. **M2 — Intelligence** (slice shipped): planner, context builder, memory engine.
4. **M3 — Tools** (slice shipped; residual open): tool runtime, plugin SDK patterns, workflow execution with approvals — in-process built-ins done; external plugin SDK + authz still open.
5. **M4 — Knowledge** (first path shipped; residual open): RAG, embeddings, document ingest, search SLAs/corpus ops.
6. **M5 — Production** (ahead): deeper security, evaluation gates, deployment hardening, dashboards.

## Near-Term Themes
- Keep every paid capability abstractable to a free local adapter.
- Close M3 residual (plugin SDK / tool authz) and M4 production search quality without regressing local free adapters.
- Production hardening (M5): eval gates in CI, security, deploy docs, incident runbooks, load/bench dashboards.

## Explicitly Later
Autonomous coding agents, self-improving loops, multi-agent swarms, multi-region active-active.
