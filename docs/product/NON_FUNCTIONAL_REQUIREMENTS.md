# Non-Functional Requirements

## Purpose
Define quality attributes Atlas must meet as a production-oriented platform.

## Scope
Cross-cutting NFRs. Concrete SLO numbers may be refined per environment; early milestones use design targets.

## Dependencies
- [Vision](./VISION.md)
- [Functional Requirements](./FUNCTIONAL_REQUIREMENTS.md)
- [Success Metrics](./SUCCESS_METRICS.md)

## Owner
Atlas Product Engineering

## Status
Accepted

## Last Updated
2026-08-03

## Related Documents
- [Architecture Overview](../architecture/OVERVIEW.md)
- [ADR: Testing Philosophy](../adr/0007-testing-philosophy.md)
- [Observability Architecture](../architecture/observability.md)

## Reliability
- NFR-REL-1: Tool and model failures are isolated; a single provider outage does not corrupt durable conversation state.
- NFR-REL-2: Workflow steps are retryable with bounded retries and backoff.
- NFR-REL-3: Partial failure leaves inspectable state (which steps completed, which failed).

## Performance and latency
- NFR-PERF-1: Interactive chat path targets p95 latency dominated by model time, not platform overhead; platform overhead budget is documented per service as implementations land.
- NFR-PERF-2: Context building and retrieval have explicit time budgets so they cannot unbounded-stall a turn.
- NFR-PERF-3: Streaming responses are supported or planned for user-facing channels where UX requires them (channel decision via ADR).

## Scalability
- NFR-SCALE-1: Stateless service instances can scale horizontally behind the gateway where practical.
- NFR-SCALE-2: Durable state lives in dedicated stores (SQL, cache, vector, object), not local disk of a single pod.
- NFR-SCALE-3: Early design does not require multi-region; single-region horizontal scale is the first target.

## Security
- NFR-SEC-1: Secrets never appear in logs, traces attributes that are exported broadly, or client-visible errors by default.
- NFR-SEC-2: Least-privilege tool execution; dangerous actions require explicit permission and often human approval.
- NFR-SEC-3: Workspace isolation is enforced at data access layers, not only UI.
- NFR-SEC-4: Dependency and container images follow a documented baseline when deployment manifests land.

## Privacy and data handling
- NFR-PRIV-1: User content and documents are workspace-scoped.
- NFR-PRIV-2: Deletion and retention policies are documentable per store (conversation, memory, objects, embeddings).
- NFR-PRIV-3: Provider adapters can be configured to avoid sending data to third-party APIs when local providers are selected.

## Cost control
- NFR-COST-1: Token usage and estimated cost are attributable per workspace and run when providers report them.
- NFR-COST-2: Rate limits and quotas are first-class for inference and expensive tools.
- NFR-COST-3: Default local/free providers keep development cost near zero.

## Observability
- NFR-OBS-1: Every request has a trace spanning major subsystems.
- NFR-OBS-2: Structured logs use consistent field names for request_id, workspace_id, conversation_id, and run_id.
- NFR-OBS-3: Metrics distinguish client errors, server errors, and dependency errors.

## Maintainability
- NFR-MAINT-1: Domain logic does not import vendor SDKs directly.
- NFR-MAINT-2: Documentation metadata standards apply to durable docs.
- NFR-MAINT-3: ADRs record significant technical choices.

## Testability
- NFR-TEST-1: Provider and tool ports are mockable in unit tests.
- NFR-TEST-2: Deterministic evaluators run without network where possible.
- NFR-TEST-3: Contracts and schemas can be validated without full runtime stack.

## Operability
- NFR-OPS-1: Health and readiness endpoints for services when implementation exists.
- NFR-OPS-2: Configuration and version identity exposed for support.
- NFR-OPS-3: Runbooks exist for common failure classes as production nears.

## Portability
- NFR-PORT-1: Local Docker Compose-style development is the first deployment target (before Kubernetes).
- NFR-PORT-2: Storage and providers are swappable via configuration and adapters.
