# ADR 0005: Storage Strategy

## Purpose
Record the multi-store storage approach and development defaults.

## Scope
Logical data stores for platform state, cache, vectors, and objects.

## Dependencies
- [Architecture: Storage](../architecture/storage.md)
- [ADR 0002: Provider Independence](./0002-provider-independence.md)
- [Domain: Memory](../domain/memory.md)

## Owner
Atlas Architecture

## Status
Accepted

## Last Updated
2026-08-03

## Related Documents
- [Deployment Architecture](../architecture/deployment.md)
- [Product Cost Goals](../product/GOALS.md)

## Context
Atlas handles users/sessions, hot cache, embeddings, and binary documents. Different access patterns demand different stores. Budget constraints favor free self-hosted defaults.

## Problem
Which storage systems should Atlas standardize on, and how do we avoid lock-in?

## Options
1. **Single Postgres** for everything including vectors and files (bytea).
2. **Cloud-only managed suite** (e.g. one vendor’s DB + blob + vector).
3. **Polyglot persistence with interfaces:** Postgres + Redis + vector DB + object store; free defaults for dev.

## Decision
Use **polyglot persistence**:
| Need | Store | Dev default |
|------|-------|-------------|
| Relational system of record | PostgreSQL | Postgres |
| Cache, rate limits, short-term | Redis | Redis |
| Semantic search | Vector store interface | Qdrant |
| Blobs | Object store interface | MinIO |

Optional broker later (NATS → Kafka) is not part of M0 storage core.

## Trade-offs
- **Benefit:** Right tool per job; free local stack; clear swap path to managed services.
- **Cost:** More moving parts in Compose; operational knowledge spread.

## Consequences
- Repositories and store ports live behind interfaces.
- No vectors-only-as-Postgres-mandatory decision for v1 (Postgres may still store metadata).
- Backup/restore stories per store in runbooks as ops mature.

## Alternatives Considered
Single-Postgres rejected for blobs/vectors at scale. Cloud-only rejected for cost and onboarding friction.
