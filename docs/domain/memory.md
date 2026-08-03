# Domain: Memory

## Purpose
Define memory types and scopes used to personalize and continue agent work beyond a single turn.

## Scope
Domain classification of memory. Physical stores (Redis, Postgres, vector) are architecture/storage decisions.

## Dependencies
- [Workspace](./workspace.md)
- [Conversation](./conversation.md)
- [Agent](./agent.md)

## Owner
Atlas Domain Engineering

## Status
Accepted

## Last Updated
2026-08-03

## Related Documents
- [Memory Architecture](../architecture/memory.md)
- [Context Builder Architecture](../architecture/context-builder.md)

## Definition
**Memory** is persisted information the platform may retrieve when building context or making decisions, beyond the immediate user message.

## Types
| Type | Lifetime | Typical use |
|------|----------|-------------|
| Short-term / session | Conversation or session TTL | Working notes, intermediate facts |
| Long-term user | Across conversations for a user | Preferences, stable facts user opted to store |
| Workspace | Shared within workspace | Team conventions, project facts |
| Ephemeral cache | Seconds–minutes | Hot retrieval cache, rate-limit counters (often not “memory” product term but related) |

## Boundaries
- In scope: scoped entries, write policies, retrieval queries, consent/visibility.
- Out of scope: full document corpora (knowledge base / RAG), though retrieval may combine both.

## Core Attributes (conceptual)
- `memory_id`, scope (`user` | `workspace` | `conversation`), owner references
- Content (text or structured), embeddings optional
- Provenance (who/what wrote it, which run)
- Expiry, sensitivity tags, delete flags

## Relationships
- Scoped under **workspace** (and optionally user or **conversation**).
- Read/written according to **agent** memory policy.
- Consumed by **context building**.
- Distinct from **prompt** artifacts (prompts are versioned instructions; memory is state).

## Invariants
- Writes respect agent and workspace policy (not every tool result becomes long-term memory).
- Retrieval is authorization-checked.
- Users can delete personal memory where product policy requires it.

## Failure Modes
- Memory pollution from unfiltered tool dumps.
- Stale memory overriding fresher conversation facts.
- Privacy leak if workspace memory is too broadly readable.
