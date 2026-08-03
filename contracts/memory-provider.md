# Contract: Memory Provider

## Purpose
Define the port for memory put/query/delete operations.

## Scope
Memory engine API used by context builder and agents.

## Dependencies
- [Contracts Index](./README.md)
- [Domain: Memory](../docs/domain/memory.md)
- [Architecture: Memory](../docs/architecture/memory.md)

## Owner
Atlas Architecture

## Status
Draft

## Last Updated
2026-08-03

## Related Documents
- [Storage ADR](../docs/adr/0005-storage-strategy.md)

## Interface (conceptual)
```text
Put(ctx, MemoryWrite) → MemoryRef
Query(ctx, MemoryQuery) → []MemoryHit
Delete(ctx, MemoryRef) → error
```

### Scopes
`user`, `workspace`, `conversation`

### MemoryQuery
- scope + ids, text query, limit, optional semantic flag

## Guarantees
- Authorization applied inside provider or mandatory middleware
- Provenance stored on write
