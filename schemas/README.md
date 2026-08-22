# Schemas

## Purpose
Hold language-agnostic data schemas for core Atlas entities.

## Scope
JSON Schema (or equivalent) documents for shared entities. Service-specific models may extend these.

## Dependencies
- [Contracts](../contracts/README.md)
- [Domain Language](../docs/domain/README.md)
- [Documentation Standards](../docs/engineering/DOCUMENTATION_STANDARDS.md)

## Owner
Atlas Architecture

## Status
Bootstrap (core entity schemas present; extra entities deferred — see task 0003)

## Last Updated
2026-08-05

## Related Documents
- [Storage Architecture](../docs/architecture/storage.md)

## Schemas
| Schema | Entity | Status |
|--------|--------|--------|
| [conversation.schema.json](./conversation.schema.json) | Conversation + turns | Present |
| [workspace.schema.json](./workspace.schema.json) | Workspace | Present |
| [run.schema.json](./run.schema.json) | Workflow run | Present |
| tool.schema.json | Tool descriptor | Deferred (task 0003) |
| memory.schema.json | Memory entry | Deferred (task 0003) |
| prompt.schema.json | Versioned prompt | Deferred (task 0003) |
| provider.schema.json | Provider binding config | Deferred (task 0003) |

## Rules
- Schemas version via `$id` and file naming.
- Implementation DTOs map to these schemas; do not invent silent fields in public APIs.
- M0 provides index and starter schemas for the highest-traffic entities.
