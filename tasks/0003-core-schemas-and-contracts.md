# Task 0003: Expand Core Schemas and Provider Contracts

## Purpose
Fill remaining JSON schemas and deepen provider/tool/memory contracts for M1.

## Scope
`schemas/*` and `contracts/*` content expansion.

## Dependencies
- [Task 0001](./0001-repository-bootstrap.md)
- [Domain language](../docs/domain/README.md)

## Owner
Atlas Architecture

## Status
Deferred — conversation/workspace/run schemas and provider/tool/memory/planner contracts are present; tool/memory/prompt/provider `*.schema.json` deferred until code-gen or validation tooling needs them.

## Last Updated
2026-08-05

## Related Documents
- [Schemas README](../schemas/README.md)
- [Contracts README](../contracts/README.md)

## Goal
Have machine-readable schemas for tool, memory, prompt, provider and tightened contract docs ready for code generation or validation.

## Inputs
- Domain docs and starter schemas (conversation, workspace, run)

## Outputs
- Additional `*.schema.json` files; richer contract markdown

## Disposition (2026-08-05)
| Artifact | Status |
|----------|--------|
| `conversation.schema.json`, `workspace.schema.json`, `run.schema.json` | Present |
| `contracts/chat-provider.md`, `embedding-provider.md`, `tool-provider.md`, `memory-provider.md`, `planner.md` | Present |
| `tool.schema.json`, `memory.schema.json`, `prompt.schema.json`, `provider.schema.json` | Deferred — not required by current runtime; add when CI schema validation or multi-language SDKs need them |

## Acceptance Criteria
- [x] Listed planned schemas either exist or are explicitly deferred with reason
- [x] Contracts describe error classes and required fields (provider contract set present)

## Test Requirements
- JSON Schema parse validation in CI when tooling lands
