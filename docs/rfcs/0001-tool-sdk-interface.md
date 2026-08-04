# RFC 0001: Tool SDK Interface

## Purpose
Propose a stable interface for registering, validating, and executing tools inside the Atlas agent orchestrator and future plugin loaders.

## Scope
In-process tool runtime for M3. Covers schema validation, timeouts, approvals, and discovery. Does not implement remote tool mesh or marketplace packaging.

## Dependencies
- [RFC Process](./README.md)
- [Architecture: Tool Runtime](../architecture/tool-runtime.md)
- [Contract: Tool Provider](../../contracts/tool-provider.md)
- [Agent Orchestrator Tools](../../services/agent-orchestrator/tools.go)

## Owner
Atlas Product Engineering

## Status
Accepted

## Last Updated
2026-08-05

## Related Documents
- [ADR 0003 Architecture Style](../adr/0003-architecture-style.md)
- [Domain: Tool](../domain/tool.md)

## Goal
Give tool authors a single Go-facing (and later multi-language) SDK shape so calculators, HTTP getters, and enterprise adapters plug into plan DAGs with validation, timeout, and human approval without forking the orchestrator.

## Reasoning
The orchestrator already has `ToolRegistry`, `ValidateToolInput`, and approval gates for `danger_echo`. Without an RFC, each new tool invents its own input map and error classes. Codifying the interface unblocks third-party tools and eval fixtures that assert tool success.

## Deliverables
1. Conceptual **Tool** interface: `Name()`, `Schema() JSONSchema`, `RequiresApproval() bool`, `Execute(ctx, input) (output, error)`.
2. **Registry** API: register, list, get, invoke-with-timeout.
3. **Error classes:** `invalid_input`, `timeout`, `denied`, `tool_error` (stable strings for eval/reports).
4. **Approval hook:** tools may mark `RequiresApproval`; orchestrator pauses run at `waiting_approval`.
5. Documentation: this RFC + updates to tool-runtime architecture when implementations land beyond built-ins.

## Design (interface sketch)

```text
Tool
  Name() string
  Description() string
  InputSchema() map[string]any   // JSON Schema object
  RequiresApproval() bool
  Execute(ctx context.Context, input map[string]any) (string, error)

ToolRegistry
  Register(Tool) error
  Get(name string) (Tool, bool)
  List() []Tool
  Invoke(ctx, name, input, timeout) (string, error)  // validate → optional approve → execute
```

Built-ins today (`echo`, `calculator`, `http_get`, `danger_echo`) should be re-expressible as this interface without behavior change.

## Risks
- Over-abstracting before multi-process tools exist.
- JSON Schema dialects diverge; pin draft-07 subset for M3.
- Side-effect tools without approval cause production incidents — default safe tools to no approval, dangerous tools must opt in.

## Dependencies
- Working agent-orchestrator plan executor (present).
- Optional: prompt registry for tool-use system prompts.

## Acceptance Criteria
- [x] RFC documents interface, error classes, approval semantics.
- [ ] At least one built-in tool refactored to explicit Tool interface (follow-up implementation task).
- [ ] Eval fixture type can assert tool success by name (orchestrator tests cover calculator today).

## Non-Goals
- WASM/remote tool hosting.
- OAuth for third-party SaaS tools.
- Automatic tool synthesis by the LLM without registry allow-list.

## Open Questions
1. Should schemas live in `schemas/tools/*.json` in-repo for every built-in?
2. Cross-language SDK (Python) timing vs Go-only for M3?

## Decision (interface direction)
Proceed with the Go interface sketch above as the v1 Tool SDK contract. Remote/plugin loaders must adapt into this interface at the process boundary.
