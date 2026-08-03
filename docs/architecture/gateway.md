# Architecture: Gateway / API Surface

## Purpose
Define the edge of the Atlas platform: authentication, routing, and conversation APIs.

## Scope
Conceptual architecture for the API gateway and public/platform HTTP (and future gRPC) surfaces.

## Dependencies
- [Architecture Overview](./OVERVIEW.md)
- [Security](./security.md)
- [ADR 0004: Communication Protocols](../adr/0004-communication-protocols.md)
- [Domain: Workspace](../domain/workspace.md)
- [Domain: Conversation](../domain/conversation.md)

## Owner
Atlas Architecture

## Status
Accepted

## Last Updated
2026-08-03

## Related Documents
- [Functional Requirements FR-AUTH, FR-API](../product/FUNCTIONAL_REQUIREMENTS.md)
- [Contracts Index](../../contracts/README.md)

## Why It Exists
Clients need a single, versioned entry point that authenticates users, resolves workspace context, enforces rate limits, and routes to internal services without exposing every internal detail.

## Problem Solved
- Unauthenticated or cross-tenant access
- Ad-hoc per-service auth duplication
- Unbounded request rates toward expensive models/tools

## Consumers
- Web/desktop clients, CLI, automation
- Internal services calling public APIs (limited)
- Operators (health, metrics scrape endpoints)

## Public Interfaces (conceptual)
- `POST /v1/conversations` — create conversation
- `POST /v1/conversations/{id}/messages` — user turn (sync or stream)
- `GET /v1/conversations/{id}` — fetch history/metadata
- `GET /v1/runs/{id}` — inspect workflow run
- `POST /v1/runs/{id}/approve` — human approval
- Auth middleware: validate JWT (or session), attach `user_id`, `workspace_id`, roles
- Health: `/healthz`, `/readyz`

Exact schemas live under `contracts/` as they are written.

## Dependencies
- Identity issuer (dev: local JWT; prod: replaceable IdP)
- Session/conversation service
- Planner/orchestrator
- Rate limiting (often Redis)
- Observability stack

## Failure Modes
| Failure | Handling |
|---------|----------|
| Invalid/expired token | 401; no partial execution |
| Missing workspace membership | 403 |
| Downstream orchestrator timeout | 504/408 with correlation id; no silent drop |
| Rate limit exceeded | 429 with retry guidance |
| Body too large | 413 before model spend |
