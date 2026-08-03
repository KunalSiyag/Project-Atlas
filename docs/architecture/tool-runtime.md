# Architecture: Tool Runtime

## Purpose
Describe the uniform framework for registering, authorizing, executing, and observing tools.

## Scope
Runtime semantics and plugin boundaries. Individual SaaS APIs are adapters.

## Dependencies
- [Architecture Overview](./OVERVIEW.md)
- [Domain: Tool](../domain/tool.md)
- [Security](./security.md)
- [Planner](./planner.md)

## Owner
Atlas Architecture

## Status
Accepted

## Last Updated
2026-08-03

## Related Documents
- [Observability](./observability.md)
- [Deployment](./deployment.md)

## Why It Exists
Agents need many capabilities (GitHub, Slack, SQL, browser, filesystem, REST). Without a shared runtime, each integration reinvents validation, timeouts, authz, and logging.

## Problem Solved
- Inconsistent error handling and timeouts
- Side effects without audit
- Hard-to-mock integrations in tests
- Prompt injection into unstructured shell strings

## Consumers
- Workflow engine (primary)
- Developers writing new tools
- Security review
- Evaluation (tool success metrics)

## Public Interfaces (conceptual)
```text
type Tool interface {
    Name() string
    Schema() Schema
    Validate(input) error
    Execute(ctx, input) (output, error)
    Timeout() time.Duration
    Retry() RetryPolicy
}
```
- Registry: `Register(tool)`, `Get(name)`, `List(workspace)`
- Executor: `Run(ctx, toolName, input, authz) → ToolResult`
- Mock mode: in-memory fakes for GitHub/Slack/email/SQL

## Dependencies
- Guardrails (pre/post)
- Secrets for external APIs
- Network egress policy / sandbox for dangerous tools
- Object storage for large artifacts
- Metrics and traces

## Failure Modes
| Failure | Handling |
|---------|----------|
| Validation failure | Do not execute; return structured error to planner |
| Authz deny | 403-equivalent to workflow; audit |
| Timeout | Cancel context; mark failed |
| Partial external write | Prefer idempotency keys; document compensation |
| Injection suspected | Block; escalate to security log |
