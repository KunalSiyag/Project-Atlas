# ADR 0002: Provider Independence

## Purpose
Record the decision that all external capability backends are accessed through interfaces/adapters.

## Scope
Inference, embeddings, vector stores, object storage, optional brokers, and tool SaaS integrations.

## Dependencies
- [Engineering Principles](../engineering/PRINCIPLES.md)
- [Domain: Provider](../domain/provider.md)
- [Architecture: Inference](../architecture/inference.md)

## Owner
Atlas Architecture

## Status
Accepted

## Last Updated
2026-08-03

## Related Documents
- [ADR 0001](./0001-documentation-first-modular-monorepo.md)
- [Product Goals G3](../product/GOALS.md)

## Context
Atlas must run on a near-zero development budget while remaining credible for commercial providers later. Hard-coding a single vendor SDK into domain logic would freeze cost and portability decisions incorrectly.

## Problem
How should the platform depend on LLMs, embeddings, vector DBs, object storage, and similar services without coupling the core to one vendor?

## Options
1. **Call vendor SDKs directly** from application services (fastest demo).
2. **Single “Atlas cloud” backend** only—no local path.
3. **Ports and adapters** with at least one free/local implementation per paid category.

## Decision
Use **provider independence via ports and adapters**. Domain and use-case code depend on interfaces (e.g. `ModelProvider`, `EmbeddingProvider`, `VectorStore`, `ObjectStore`). Adapters implement vendors. Development defaults favor free/local (Ollama, Qdrant, MinIO, etc.). Commercial providers are optional adapters for benchmarks and production.

## Trade-offs
- **Benefit:** Swap providers with minimal core change; clone-and-run without cloud signups; stronger engineering narrative.
- **Cost:** More interface design up front; occasional lowest-common-denominator API design.

## Consequences
- No `OpenAI.ChatCompletion` (or equivalent) in domain packages.
- Contracts under `contracts/` describe provider ports before implementations.
- Benchmarks compare providers through the same interface.
- Configuration selects adapters (see ADR 0006).

## Alternatives Considered
Direct SDK usage rejected for lock-in and cost. Cloud-only rejected as incompatible with the cost and learning goals.
