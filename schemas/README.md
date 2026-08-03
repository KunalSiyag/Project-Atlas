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
Bootstrap

## Last Updated
2026-08-03

## Related Documents
- [Storage Architecture](../docs/architecture/storage.md)

## Planned Schemas (M1+)
| Schema | Entity |
|--------|--------|
| [conversation.schema.json](./conversation.schema.json) | Conversation + turns |
| [workspace.schema.json](./workspace.schema.json) | Workspace |
| [tool.schema.json](./tool.schema.json) | Tool descriptor |
| [memory.schema.json](./memory.schema.json) | Memory entry |
| [prompt.schema.json](./prompt.schema.json) | Versioned prompt |
| [provider.schema.json](./provider.schema.json) | Provider binding config |
| [run.schema.json](./run.schema.json) | Workflow run |

## Rules
- Schemas version via `$id` and file naming.
- Implementation DTOs map to these schemas; do not invent silent fields in public APIs.
- M0 provides index and starter schemas for the highest-traffic entities.
