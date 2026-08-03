# Domain: Provider

## Purpose
Define providers as swappable backends behind stable interfaces.

## Scope
Provider categories and independence rules. Concrete vendor choices are ADRs or configuration, not hard domain types.

## Dependencies
- [Agent](./agent.md)
- [Tool](./tool.md)
- [Evaluation](./evaluation.md)

## Owner
Atlas Domain Engineering

## Status
Accepted

## Last Updated
2026-08-03

## Related Documents
- [ADR: Provider Independence](../adr/0002-provider-independence.md)
- [Inference Architecture](../architecture/inference.md)

## Definition
A **provider** is an implementation of a capability port such as chat completion, embeddings, vector search, object storage, or messaging. Domain logic depends on the port, not the vendor.

## Categories
| Category | Port examples | Free/local examples | Commercial examples |
|----------|---------------|---------------------|---------------------|
| Chat / completion | `ModelProvider.Generate` | Ollama, vLLM, llama.cpp | OpenAI, Anthropic, Gemini |
| Embeddings | `EmbeddingProvider.Embed` | BGE, Nomic, Sentence Transformers | OpenAI embeddings |
| Vector store | `VectorStore.Upsert/Query` | Qdrant | Pinecone |
| Object storage | `ObjectStore.Put/Get` | MinIO | S3 |
| Optional broker | `Broker.Publish/Subscribe` | NATS | Kafka |

## Boundaries
- In scope: capability contracts, health, cost metadata, model identifiers.
- Out of scope: vendor-specific fine-tuning UIs.

## Relationships
- Selected by **model router** / configuration for an **agent** or task class.
- Used during **workflows** and **context building** (embeddings/retrieval).
- Compared via **evaluation** and benchmarks.

## Invariants
- Business logic must not import a single vendor SDK as the only path.
- Each paid provider category has at least one free/local adapter planned for development.
- Failures are typed (timeout, rate limit, auth, invalid request) for retry policy.

## Failure Modes
- Silent quality change when default model IDs shift upstream.
- Cost blowups without per-call metering.
- Data residency violations if remote providers are used contrary to workspace policy.
