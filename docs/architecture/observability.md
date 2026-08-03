# Architecture: Observability

## Purpose
Describe logging, metrics, and distributed tracing for agent requests.

## Scope
OpenTelemetry-oriented design and correlation across subsystems. Vendor backends are replaceable.

## Dependencies
- [Architecture Overview](./OVERVIEW.md)
- [ADR 0008: OpenTelemetry Direction](../adr/0008-opentelemetry.md)
- [Evaluation](./evaluation.md)

## Owner
Atlas Architecture

## Status
Accepted

## Last Updated
2026-08-03

## Related Documents
- [NFR Observability](../product/NON_FUNCTIONAL_REQUIREMENTS.md)
- [Deployment](./deployment.md)

## Why It Exists
Agent failures span planner, retriever, LLM, tools, and memory. Without end-to-end traces, incidents are un-debuggable and cost is unattributable.

## Problem Solved
- Logs without correlation IDs
- No span covering tool calls under a user request
- Inability to see which stage dominates latency

## Consumers
- Operators / SRE
- Developers
- Evaluation (latency/cost signals)
- Security (audit-adjacent traces with care)

## Public Interfaces (conceptual)
- Trace context propagation on all internal calls
- Standard attributes: `request_id`, `workspace_id`, `conversation_id`, `run_id`, `step_id`, `provider`, `model`
- Metrics: request rate, error rate, latency histograms, token counters, tool success
- Structured log fields matching attributes above
- Example trace path: Request → Planner → Retriever → LLM → GitHub Tool → Memory → Email Tool → Done

## Dependencies
- OpenTelemetry SDK (language-specific when code exists)
- Collector → Prometheus/Grafana (dev defaults free)
- Log pipeline (stdout JSON in containers)

## Failure Modes
| Failure | Handling |
|---------|----------|
| Collector down | Local buffer/drop policy; app must not crash |
| High cardinality labels (raw prompts) | Forbid; use hashes/ids |
| PII in spans | Attribute allowlist; redact |
| Missing context propagation | Broken traces; CI smoke checks later |
