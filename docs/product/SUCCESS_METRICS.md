# Success Metrics

## Purpose
Define how Atlas knows the product and engineering approach are working.

## Scope
Metrics for product foundation quality, platform behavior once implemented, and repository health. Not billing KPIs.

## Dependencies
- [Goals](./GOALS.md)
- [Functional Requirements](./FUNCTIONAL_REQUIREMENTS.md)
- [Non-Functional Requirements](./NON_FUNCTIONAL_REQUIREMENTS.md)

## Owner
Atlas Product Engineering

## Status
Accepted

## Last Updated
2026-08-03

## Related Documents
- [Evaluation Domain](../domain/evaluation.md)
- [Benchmarks](../benchmarks/README.md)
- [Milestones](../../management/MILESTONES.md)

## Foundation (M0) Success Metrics
These apply before application runtime code ships.

| ID | Metric | Target |
|----|--------|--------|
| SM-M0-1 | Product constitution coverage | Vision, goals, non-goals, personas, FR, NFR, success metrics, glossary exist with substantive content |
| SM-M0-2 | Domain language coverage | Workspace, agent, conversation, memory, tool, provider, evaluation, prompt, workflow defined |
| SM-M0-3 | Architecture subsystem coverage | Overview plus gateway, planner, memory, context, inference, tools, storage, security, observability, evaluation, deployment |
| SM-M0-4 | Foundation ADRs | Provider independence, architecture style, protocols, storage, configuration, testing (plus monorepo/docs ADR) |
| SM-M0-5 | Navigation | README and AI_CONTEXT list short load paths; management and contracts/schemas/tasks indexes exist |
| SM-M0-6 | No premature app code mandate | M0 complete without requiring production service binaries |

## Platform Runtime Metrics (post-implementation)
Tracked when corresponding milestones land.

| ID | Metric | Intent |
|----|--------|--------|
| SM-RT-1 | Tool success rate | Share of tool calls that complete within policy (success vs error/timeout) |
| SM-RT-2 | End-to-end latency | p50/p95 per surface (chat, tool-heavy workflows) |
| SM-RT-3 | Cost per successful run | Tokens + estimated $ or local compute proxy by workspace |
| SM-RT-4 | Grounding / citation correctness | Deterministic checks where citations are required |
| SM-RT-5 | Hallucination proxy scores | Optional judge or deterministic fact checks on eval sets |
| SM-RT-6 | Human rating | Optional thumbs / rubric on sampled runs |
| SM-RT-7 | Provider swap cost | Engineering effort (or test pass rate) when switching model provider via config |
| SM-RT-8 | Trace completeness | % of requests with full span tree across planner → model → tools |

## Engineering Process Metrics
| ID | Metric | Intent |
|----|--------|--------|
| SM-ENG-1 | Contract-before-code adherence | New public surfaces have contract docs before merge of implementation |
| SM-ENG-2 | ADR coverage | Material decisions have ADRs linked from architecture |
| SM-ENG-3 | Doc freshness | Status and Last Updated maintained on durable docs when content changes |
| SM-ENG-4 | Onboarding time | New contributor loads AI_CONTEXT path list and can locate correct subsystem in minutes |

## How Metrics Are Stored
- Foundation metrics: verified by checklist against the repository tree (see management milestones).
- Runtime metrics: evaluation subsystem and observability backends (Prometheus/Grafana/OTel-oriented).
- Benchmarks: `docs/benchmarks/` for plans and later results.
