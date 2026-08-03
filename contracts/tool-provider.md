# Contract: Tool Provider / Runtime Port

## Purpose
Define the contract for tool registration and execution.

## Scope
Tool runtime port used by the workflow engine.

## Dependencies
- [Contracts Index](./README.md)
- [Domain: Tool](../docs/domain/tool.md)
- [Architecture: Tool Runtime](../docs/architecture/tool-runtime.md)

## Owner
Atlas Architecture

## Status
Draft

## Last Updated
2026-08-03

## Related Documents
- [Security Architecture](../docs/architecture/security.md)

## Interface (conceptual)
```text
ListTools(workspace) → []ToolDescriptor
Execute(ctx, ToolInvocation) → ToolResult
```

### ToolDescriptor
- `name`, `description`, `input_schema`, `side_effect_class`, `timeout_ms`

### ToolInvocation
- `name`, `input`, `principal`, `run_id`, `step_id`

### ToolResult
- `ok`, `output` | `error`, `duration_ms`, `attempts`

## Guarantees
- Validate before execute
- Authz before execute
- Context cancellation honored
