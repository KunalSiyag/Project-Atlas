# Milestones

## Purpose
Define milestone outcomes and exit criteria for Atlas.

## Scope
M0 through M5 as engineering milestones. Not marketing release names.

## Dependencies
- [Roadmap](./ROADMAP.md)
- [Success Metrics](../docs/product/SUCCESS_METRICS.md)

## Owner
Atlas Core Engineering

## Status
Accepted

## Last Updated
2026-08-03

## Related Documents
- [Sprints](./SPRINTS.md)
- [Backlog](./BACKLOG.md)

## M0 — Foundation
**Status:** Complete when verification below passes.

**Outcomes:**
- Product constitution (vision, goals, non-goals, personas, FR, NFR, success metrics, glossary)
- Domain docs for core concepts
- Subsystem architecture docs
- Foundation ADRs (docs monorepo, provider independence, architecture style, protocols, storage, configuration, testing, OpenTelemetry)
- Management layer + contracts/schemas/tasks indexes
- README and AI_CONTEXT navigation for M0-complete state

**Exit criteria:** Acceptance criteria in the M0 plan; no application service binaries required.

## M1 — Core Platform
**Outcomes:** Inference gateway contracts and first implementation path; configuration loading; structured logging; metrics hooks; Docker Compose skeleton for data plane.

**Exit criteria:** Local stack boots data dependencies; a hello-path generation works via provider interface (local model optional).

## M2 — Intelligence
**Outcomes:** Planner and context builder designs implemented at MVP; memory read/write paths.

**Exit criteria:** Multi-step plan artifact produced and stored for a sample conversation.

## M3 — Tools
**Outcomes:** Tool runtime with ≥2 mock tools and one real optional integration; workflow step execution with timeout/retry.

**Exit criteria:** End-to-end run calling tools under authz with traces.

## M4 — Knowledge
**Outcomes:** Document ingest, embeddings, retrieval in context builder.

**Exit criteria:** Query retrieves relevant chunk with source metadata on a fixture corpus.

## M5 — Production
**Outcomes:** Evaluation gates, security hardening, deploy docs, incident runbooks, dashboards.

**Exit criteria:** Benchmark report + trace dashboard for sample load; release checklist exists.
