# Product Vision

## Purpose
Define the product direction for Project Atlas.

## Scope
Covers the product framing, positioning, and long-term intent for the platform. Does not specify service APIs or implementation.

## Dependencies
- [README](../../README.md)
- [Goals](./GOALS.md)
- [Non-Goals](./NON_GOALS.md)
- [Architecture Overview](../architecture/OVERVIEW.md)

## Owner
Atlas Product Engineering

## Status
Accepted

## Last Updated
2026-08-03

## Related Documents
- [Functional Requirements](./FUNCTIONAL_REQUIREMENTS.md)
- [Success Metrics](./SUCCESS_METRICS.md)
- [Agent Orchestrator PRD](../../services/agent-orchestrator/PRD.md)

## Vision
Atlas is an enterprise-grade AI agent **platform**: teams use it to build, orchestrate, evaluate, observe, and deploy production agents with modular boundaries and replaceable providers.

The LLM is a dependency, not the application. The product is authentication, conversation and session management, context construction, planning and workflow execution, tool runtime, model routing, guardrails, evaluation, and observability—coordinated as one system.

When someone opens the repository, the intended reaction is not “this person knows LangChain.” It is “this person understands how to build reliable AI platforms.”

## Positioning
Atlas is to production AI systems engineering what a disciplined distributed-systems case study is to backend infrastructure work: documentation-first, interface-driven, and designed to evolve under real constraints (cost, reliability, auditability).

## Core Product Beliefs
- Context engineering matters more than one-shot prompt engineering.
- Agents plan and execute workflows (including DAGs with human approval), not only single chat replies.
- Every paid or vendor-specific capability has a free or local alternative behind an interface for development.
- Evaluation and observability are product features, not afterthoughts.
- Explicit non-goals protect focus until fundamentals are solid.

## Success Criteria (Product-Level)
- Teams can reason about each service from local documentation.
- Provider changes are handled through configuration and adapters where practical.
- Agent executions are observable, testable, and auditable.
- A new engineer or AI assistant can load a short path list and contribute safely.
