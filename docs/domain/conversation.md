# Domain: Conversation

## Purpose
Define conversation and turn structure as the user-facing interaction record.

## Scope
Conversation lifecycle and relationship to runs and memory. Transport (WebSocket vs SSE) is an ADR/architecture concern.

## Dependencies
- [Workspace](./workspace.md)
- [Agent](./agent.md)
- [Memory](./memory.md)
- [Workflow](./workflow.md)

## Owner
Atlas Domain Engineering

## Status
Accepted

## Last Updated
2026-08-03

## Related Documents
- [Context Builder Architecture](../architecture/context-builder.md)
- [Gateway Architecture](../architecture/gateway.md)

## Definition
A **conversation** is a durable, ordered series of **turns** between a human user (and optionally other actors) and one or more agents within a workspace.

A **turn** is a unit of interaction: typically a user message plus the agent’s planning, tool activity, and response artifacts for that request.

## Boundaries
- In scope: identity, participants, message history references, linked runs, status.
- Out of scope: full blob storage of large uploads (object storage references instead).

## Core Attributes (conceptual)
- `conversation_id`, `workspace_id`, title, status (open/archived)
- Participant user ids; bound **agent** (or agent selection rules)
- Ordered turns with roles (user, assistant, system, tool)
- Links to **runs**/workflows started from turns
- Timestamps and retention metadata

## Relationships
- Belongs to a **workspace**.
- Uses an **agent** configuration (fixed or per-turn).
- Feeds **context building** and may write **memory**.
- Spawns **workflows** when multi-step work is required.
- Produces artifacts scored by **evaluation**.

## Invariants
- Turns are append-oriented; rewriting history is an explicit admin operation if supported.
- Conversation data is workspace-scoped.
- Every side-effecting agent action on a turn is attributable to a run/step for audit.

## Failure Modes
- Context explosion if history is unbounded and unprioritized.
- Orphan runs if conversation is deleted without retention policy.
- Split-brain if multiple clients append without concurrency control.
