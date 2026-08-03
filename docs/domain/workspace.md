# Domain: Workspace

## Purpose
Define the workspace as the primary isolation and tenancy boundary in Atlas.

## Scope
Conceptual domain model for workspace. Storage schema details live in service data models.

## Dependencies
- [Domain Index](./README.md)
- [Agent](./agent.md)
- [Conversation](./conversation.md)

## Owner
Atlas Domain Engineering

## Status
Accepted

## Last Updated
2026-08-03

## Related Documents
- [FR-AUTH](../product/FUNCTIONAL_REQUIREMENTS.md)
- [Storage Architecture](../architecture/storage.md)

## Definition
A **workspace** is a named isolation boundary that groups users, permissions, agents, conversations, memories, documents, tools configuration, and provider settings.

## Boundaries
- In scope: membership, roles, resource ownership, configuration defaults, quotas.
- Out of scope: global platform admin concerns that span all workspaces (platform operators).

## Core Attributes (conceptual)
- `workspace_id`, display name, status (active/suspended)
- Owner and members with roles (e.g. owner, developer, viewer)
- Default agent configuration and allowed tools
- Quotas (rate limits, token budgets) when configured
- Linked knowledge collections and secrets references (not raw secrets in domain objects)

## Relationships
- Has many **users** (membership).
- Owns **conversations**, **agents**, **memories**, **workflows** history, and **evaluation** datasets scoped to the workspace.
- Constrains which **providers** and **tools** may be used.

## Invariants
- Data belonging to workspace A is not readable by workspace B without explicit cross-workspace grant (default: none).
- Tool execution authorization is evaluated in workspace context.
- Deleting or suspending a workspace must have a documented cascade or retention policy (implementation detail).

## Failure Modes
- Orphaned resources if cascade rules are incomplete.
- Privilege escalation if membership checks only exist at the UI.
- Config bleed if provider keys are global-only without workspace override policy.
