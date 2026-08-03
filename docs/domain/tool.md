# Domain: Tool

## Purpose
Define tools as first-class, interface-uniform capabilities agents may invoke.

## Scope
Tool identity, schema, and execution semantics at the domain level. Implementation adapters live under tool runtime architecture.

## Dependencies
- [Agent](./agent.md)
- [Workflow](./workflow.md)
- [Provider](./provider.md)

## Owner
Atlas Domain Engineering

## Status
Accepted

## Last Updated
2026-08-03

## Related Documents
- [Tool Runtime Architecture](../architecture/tool-runtime.md)
- [Security Architecture](../architecture/security.md)

## Definition
A **tool** is a named capability with a machine-readable input schema, validation rules, execution behavior, timeout/retry policy, and authorization requirements.

Examples: GitHub repository read, Slack post, SQL query, browser navigation, filesystem read, generic REST call.

## Conceptual Interface
Every tool satisfies the same conceptual contract:
- `Name()` — stable identifier
- `Schema()` — input (and optionally output) schema
- `Validate(input)` — structural and policy validation
- `Execute(ctx, input)` — perform work
- `Timeout()` / `Retry()` — operational policy

## Boundaries
- In scope: registration, schemas, authz metadata, side-effect classification (read vs write).
- Out of scope: proprietary SDK details inside domain types (those belong in adapters).

## Relationships
- Allowed by **agent** and **workspace** policy.
- Invoked as steps inside **workflows**.
- May call external systems via **providers** or direct integrations.
- Guardrails run before and after execution.
- Outcomes feed **evaluation** (tool success rate) and **observability**.

## Side-Effect Classes
| Class | Examples | Typical control |
|-------|----------|-----------------|
| Read-only | List issues, fetch doc | Authed allowlist |
| Soft write | Draft comment | Often auto with audit |
| Hard write | Send email, merge PR | Human approval often required |
| Dangerous | Shell, arbitrary code | Strict deny or isolated sandbox |

## Invariants
- No tool execution without authorization check.
- Every execution is attributable to workspace, user, agent, run, and step.
- Timeouts are mandatory; unbounded waits are invalid configuration.

## Failure Modes
- Schema drift vs model tool-calling format.
- Partial external side effects without idempotency keys.
- Prompt injection inducing unsafe tool arguments.
