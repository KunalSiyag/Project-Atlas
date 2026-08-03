# Architecture: Security and Guardrails

## Purpose
Describe authentication, authorization, and guardrail controls around agent execution.

## Scope
Platform security architecture for M0 design. Full compliance certifications are out of scope.

## Dependencies
- [Architecture Overview](./OVERVIEW.md)
- [Gateway](./gateway.md)
- [Tool Runtime](./tool-runtime.md)
- [Domain: Workspace](../domain/workspace.md)

## Owner
Atlas Architecture

## Status
Accepted

## Last Updated
2026-08-03

## Related Documents
- [NFR Security](../product/NON_FUNCTIONAL_REQUIREMENTS.md)
- [Runbooks](../runbooks/README.md)

## Why It Exists
Agents combine untrusted natural language with powerful tools. Without guardrails, the platform becomes an injection and data-exfiltration engine.

## Problem Solved
- Prompt injection driving unsafe tool calls
- Cross-workspace data access
- Secret leakage in model outputs or logs
- Unapproved hard side effects (email, merge, spend)

## Consumers
- Gateway (authn)
- Tool runtime (authz + validation)
- Response path (egress filters)
- Operators and security reviewers

## Public Interfaces (conceptual)
- Authn middleware → principal
- `Authorize(principal, action, resource) → allow|deny`
- Pre-tool guard: `CheckToolCall(principal, tool, input)`
- Post-response guard: `ScanOutput(text) → findings`
- Policy config per workspace (allowlists, approval requirements)

## Control Points
1. **Before tools:** permissions, schema validation, injection heuristics, dangerous-action blocks
2. **Before response:** secret patterns, policy text, optional citation requirements
3. **Always:** audit log for denials and hard writes

## Dependencies
- Identity provider
- Workspace membership store
- Secret stores (env/vault)—never prompts
- Optional browser/code sandbox for dangerous tools

## Failure Modes
| Failure | Handling |
|---------|----------|
| Auth service down | Fail closed at gateway |
| False positive injection block | User-visible deny with code; allow operator override path later |
| Secret in output | Redact/block response; alert |
| Policy misconfiguration | Prefer deny on hard writes; alert on parse errors |
