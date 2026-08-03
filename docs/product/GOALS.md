# Product Goals

## Purpose
State what Project Atlas optimizes for so architecture and implementation stay aligned.

## Scope
Product outcomes and engineering outcomes for the platform foundation through early milestones. Not a sprint backlog.

## Dependencies
- [Vision](./VISION.md)
- [Non-Goals](./NON_GOALS.md)
- [Success Metrics](./SUCCESS_METRICS.md)

## Owner
Atlas Product Engineering

## Status
Accepted

## Last Updated
2026-08-03

## Related Documents
- [Functional Requirements](./FUNCTIONAL_REQUIREMENTS.md)
- [Milestones](../../management/MILESTONES.md)

## Primary Goals

### G1 — Reliable agent platform, not a demo chatbot
Deliver a modular platform for production agent systems with explicit contracts, observable execution, testable components, and vendor-independent provider adapters.

### G2 — Context and workflow as first-class capabilities
Support multi-stage request handling: auth and workspace, session load, context building, planning, DAG workflow execution, tool runtime, model routing, guardrails, evaluation, and tracing.

### G3 — Provider and infrastructure independence
All external inference, embedding, vector, object storage, messaging, and tool backends are accessed through interfaces so free/local implementations can power development and commercial providers can plug in later.

### G4 — Engineering-grade repository artifacts
Ship Architecture Decision Records, evaluation and benchmark structure, prompt versioning strategy, security and operational docs, and deployment-oriented documentation—not code alone.

### G5 — Cost-conscious development path
Design so a student or early-career budget can run the platform locally (order of ₹0–₹2,000/month) by defaulting to free/local stacks while keeping commercial paths open.

### G6 — AI-assisted development efficiency
Documentation is modular and small enough that humans and AI assistants load only the relevant files for a task, with service-local context packs and a clear onboarding path.

## Secondary Goals
- Interview- and portfolio-ready narrative: documented trade-offs, not “we used X because everyone uses X.”
- Parallel flagship with distributed-systems work (e.g. North Star): same engineering discipline applied to AI systems.
- Clear milestone ladder (foundation → core platform → intelligence → tools → knowledge → production).
