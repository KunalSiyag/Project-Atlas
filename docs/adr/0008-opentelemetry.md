# ADR 0008: OpenTelemetry for Observability

## Purpose
Record the decision to standardize on OpenTelemetry for traces, metrics, and related context propagation.

## Scope
Observability instrumentation approach across Atlas services.

## Dependencies
- [Architecture: Observability](../architecture/observability.md)
- [ADR 0003](./0003-architecture-style.md)
- [NFR Observability](../product/NON_FUNCTIONAL_REQUIREMENTS.md)

## Owner
Atlas Architecture

## Status
Accepted

## Last Updated
2026-08-03

## Related Documents
- [Deployment Architecture](../architecture/deployment.md)
- [Evaluation Architecture](../architecture/evaluation.md)

## Context
Agent requests cross many components. Proprietary APM lock-in conflicts with free/local development and provider independence ethos. Structured correlation is mandatory for production AI systems.

## Problem
Which observability standard should Atlas adopt for traces and metrics?

## Options
1. **Ad-hoc logs only.**
2. **Single commercial APM agent** embedded in domain code.
3. **OpenTelemetry** instrumentation with replaceable backends (Collector → Prometheus/Grafana/Jaeger or vendors).

## Decision
Standardize on **OpenTelemetry** for distributed traces and metrics (and log correlation where supported). Export via collector; default local backends are free (e.g. Prometheus, Grafana, and a trace backend compatible with OTLP). Business logic depends on OTel APIs/patterns, not a single SaaS agent SDK.

## Trade-offs
- **Benefit:** Portable, industry-standard, free local stack, clear span model for agent stages.
- **Cost:** Collector configuration; discipline on attribute cardinality.

## Consequences
- Span naming conventions document request → planner → retrieve → inference → tool → memory.
- Forbidden high-cardinality raw prompt text as metric labels.
- Runbooks reference trace ids for incidents.

## Alternatives Considered
Logs-only rejected for multi-hop agent debugging. Single APM SDK rejected for lock-in and cost during development.
