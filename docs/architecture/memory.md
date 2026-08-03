# Architecture: Memory Engine

## Purpose
Describe how Atlas stores and retrieves short- and long-term memory for agents.

## Scope
Memory service boundaries and interactions with context building. Physical multi-store mapping is summarized; see storage architecture.

## Dependencies
- [Architecture Overview](./OVERVIEW.md)
- [Context Builder](./context-builder.md)
- [Storage](./storage.md)
- [Domain: Memory](../domain/memory.md)

## Owner
Atlas Architecture

## Status
Accepted

## Last Updated
2026-08-03

## Related Documents
- [Security](./security.md)
- [ADR 0005: Storage Strategy](../adr/0005-storage-strategy.md)

## Why It Exists
Agents need continuity: preferences, prior decisions, and working notes without stuffing entire history into every prompt.

## Problem Solved
- Stateless completions that forget user preferences
- Unstructured “memory” dumped into system prompts without scope or expiry
- Missing authorization on remembered facts

## Consumers
- Context builder (primary reader)
- Agents/workflows (writers under policy)
- Users (list/delete personal memory)
- Evaluation (optional memory-hit metrics)

## Public Interfaces (conceptual)
- `PutMemory(scope, content, metadata)`
- `QueryMemory(scope, query, limit)` — keyword and/or semantic
- `ListMemory(scope, filter)`
- `DeleteMemory(id)` / `DeleteByScope(...)`
- Policy check: can this agent write long-term memory?

## Dependencies
- Postgres (durable structured memory metadata/content)
- Redis (optional hot session memory / cache)
- Vector store (semantic memory retrieval)
- Authz from workspace membership

## Failure Modes
| Failure | Handling |
|---------|----------|
| Vector store down | Fall back to metadata/keyword; degrade gracefully |
| Write policy deny | No-op with audit log; do not crash run |
| Over-retention | TTL workers; explicit retention policy |
| Poisoned content | Size limits; optional classifiers; user delete |
