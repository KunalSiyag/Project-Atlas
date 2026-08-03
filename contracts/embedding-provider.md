# Contract: Embedding Provider

## Purpose
Define the conceptual contract for embedding providers.

## Scope
Vector embedding generation only; vector store upsert/query is separate.

## Dependencies
- [Contracts Index](./README.md)
- [Domain: Provider](../docs/domain/provider.md)

## Owner
Atlas Architecture

## Status
Draft

## Last Updated
2026-08-03

## Related Documents
- [Architecture: Storage](../docs/architecture/storage.md)
- [Chat Provider](./chat-provider.md)

## Interface (conceptual)
```text
Embed(ctx, EmbedRequest) → (EmbedResponse, error)
```

### Request
- `inputs[]`: text strings (batch)
- `model`: logical embedding model id
- `metadata`: correlation ids

### Response
- `vectors[]`: float arrays aligned with inputs
- `dimensions`: int
- `usage` optional
- `provider`, `model`

## Implementations (planned)
Local Sentence Transformers / BGE / Nomic via a local runner; optional OpenAI embeddings adapter.
