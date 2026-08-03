# Architecture: Storage Layer

## Purpose
Describe the multi-store storage strategy and responsibilities of each store class.

## Scope
Logical stores and ownership of data classes. Vendor picks default to free/local for development.

## Dependencies
- [Architecture Overview](./OVERVIEW.md)
- [ADR 0005: Storage Strategy](../adr/0005-storage-strategy.md)
- [Domain: Workspace](../domain/workspace.md)

## Owner
Atlas Architecture

## Status
Accepted

## Last Updated
2026-08-03

## Related Documents
- [Memory](./memory.md)
- [Deployment](./deployment.md)

## Why It Exists
Agent platforms mix relational state, hot cache, vectors, and blobs. One database for everything creates the wrong access patterns and scaling limits.

## Problem Solved
- Forcing chat history, embeddings, and PDFs into a single store
- Unclear ownership of durability vs ephemerality
- Vendor lock-in to a single cloud database

## Consumers
- All services with durable state
- Backup/restore and retention jobs
- Operators

## Store Map
| Store | Holds | Dev default | Swappable to |
|-------|-------|-------------|--------------|
| PostgreSQL | Users, sessions, workspaces, permissions, runs, prompts metadata | Postgres | Managed Postgres |
| Redis | Short-term memory, cache, rate limits | Redis | Managed Redis |
| Vector store | Embeddings, semantic search | Qdrant | Pinecone / others |
| Object storage | PDFs, images, audio, large artifacts | MinIO | S3-compatible |
| Optional broker | Async events (later milestones) | NATS | Kafka |

## Public Interfaces (conceptual)
- Repository ports per aggregate (WorkspaceRepo, ConversationRepo, RunRepo, …)
- `ObjectStore`, `VectorStore`, `Cache` interfaces
- Migrations owned by services writing to SQL

## Dependencies
- Configuration for connection strings
- Network policy between services and stores
- Encryption at rest (environment-dependent)

## Failure Modes
| Failure | Handling |
|---------|----------|
| Postgres primary down | Service unavailable for writes; health fails ready |
| Redis down | Degrade cache/rate-limit (fail closed on rate limit if required) |
| Vector down | Degrade RAG; core chat may continue |
| Object store down | Upload/download fail; reference integrity errors |
| Migration skew | Prevent deploy via migration checks |
