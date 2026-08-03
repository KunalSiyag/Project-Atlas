# Personas

## Purpose
Define who Atlas serves so requirements and UX decisions stay grounded.

## Scope
Primary and secondary personas for the platform product. Not end-customer marketing personas for third-party apps built on Atlas.

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
- [Glossary](./GLOSSARY.md)

## Primary Personas

### P1 — Platform Engineer
**Job:** Build and operate Atlas (or a fork) as a reliable internal platform.

**Needs:** Clear service boundaries, ADRs, deployability, observability, config strategy, testing philosophy.

**Pain if missing:** Hidden coupling to vendors; unreviewable agent runs; no way to swap models or stores.

### P2 — Application / Agent Developer
**Job:** Build agent workflows, tools, and prompts on top of Atlas APIs and contracts.

**Needs:** Stable APIs, tool SDK patterns, workflow semantics, memory and context behavior, examples.

**Pain if missing:** Guessing planner behavior; reimplementing RAG and tools ad hoc; no eval hooks.

### P3 — Operator / SRE
**Job:** Run Atlas in an environment, respond to incidents, capacity and cost.

**Needs:** Runbooks, metrics, traces, health checks, rate limits, clear failure modes.

**Pain if missing:** Opaque LLM hangs; unbounded tool spend; no correlation IDs across planner and tools.

### P4 — Evaluator / Quality Owner
**Job:** Measure agent quality, regressions, cost, and grounding over time.

**Needs:** Evaluation datasets, score definitions, benchmark reports, prompt versioning.

**Pain if missing:** “It feels better” as the only signal; silent quality regressions after model swaps.

## Secondary Personas

### P5 — Security / Compliance Reviewer
**Needs:** Guardrails, permission model, audit logs, secret scanning, policy documentation.

### P6 — End User (Human in the loop)
**Job:** Starts conversations, uploads files, approves sensitive steps, rates outcomes.

**Needs:** Clear approvals, transparent agent progress, ability to stop or revise.

### P7 — AI Coding Assistant
**Job:** Contribute to the Atlas repository itself using modular docs.

**Needs:** `AI_CONTEXT.md`, service-local packs, short load paths, explicit non-goals.

## Actors in the Runtime Ecosystem
Separate from product personas, the runtime ecosystem includes: Human User, AI Agent, Tool Runtime, Knowledge Layer, and Platform (coordination). Domain docs define these entities precisely.
