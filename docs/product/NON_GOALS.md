# Product Non-Goals

## Purpose
Record what Atlas will not build or prioritize in the near term so scope stays honest.

## Scope
Explicit exclusions for M0–M3 timeframe unless a later ADR or product revision accepts them.

## Dependencies
- [Vision](./VISION.md)
- [Goals](./GOALS.md)

## Owner
Atlas Product Engineering

## Status
Accepted

## Last Updated
2026-08-03

## Related Documents
- [Functional Requirements](./FUNCTIONAL_REQUIREMENTS.md)
- [Milestones](../../management/MILESTONES.md)

## Non-Goals

### NG1 — Single-purpose chatbot product
Atlas is not a consumer chatbot app with a fixed UI and one hard-coded model. Chat is a surface; the product is the platform underneath.

### NG2 — Vendor lock-in
Atlas is not tied to one model vendor, vector database, cloud provider, UI framework, or agent library (LangChain/LangGraph or otherwise). Libraries may be used behind adapters; they must not own domain logic.

### NG3 — Demo-over-architecture delivery
Atlas does not prioritize impressive demos at the expense of maintainable production architecture, contracts, tests, and observability.

### NG4 — Autonomous research agents first
The following are out of scope until fundamentals are solid:
- Fully autonomous coding agents that self-modify the platform
- Self-improving agents with unbounded recursive loops
- Large multi-agent swarms (dozens of peer agents) as the default runtime

### NG5 — Day-one multi-region and enterprise identity full stack
Geo replication, multi-region active-active, and full enterprise IdP federation are not required for early milestones. Design for replaceable identity later; do not block M0–M2 on them.

### NG6 — Exactly-once global semantics for every path
Prefer at-least-once with idempotent handlers where appropriate. Exactly-once is not a global product requirement for V1 messaging or tool execution.

### NG7 — Building every integration for real cloud on day one
GitHub, Slack, email, browser, and SQL tools may be mocked or local during development. Real SaaS credentials are optional for foundation and early intelligence work.

### NG8 — Full M1–M5 runtime under documentation foundation
Completing documentation, domain language, architecture, and ADRs does not require live inference, Kubernetes, or production SLOs in the same milestone.
