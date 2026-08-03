# Contract: Chat / Completion Provider

## Purpose
Define the conceptual contract for chat/completion model providers.

## Scope
Provider port only—not HTTP gateway shapes for end users.

## Dependencies
- [Contracts Index](./README.md)
- [Domain: Provider](../docs/domain/provider.md)
- [Architecture: Inference](../docs/architecture/inference.md)

## Owner
Atlas Architecture

## Status
Draft

## Last Updated
2026-08-03

## Related Documents
- [ADR 0002](../docs/adr/0002-provider-independence.md)
- [Embedding Provider](./embedding-provider.md)

## Interface (conceptual)
```text
Generate(ctx, Request) → (Response, error)
```

### Request (fields)
- `messages[]`: role + content parts
- `model`: logical model id (router-resolved)
- `tools[]` optional: name, description, JSON schema
- `temperature`, `max_tokens` optional
- `metadata`: workspace/run correlation ids

### Response (fields)
- `message`: assistant content and optional tool_calls
- `usage`: input/output tokens when known
- `provider`, `model`: concrete backend ids
- `finish_reason`

## Error Classes
`invalid_request`, `auth`, `rate_limit`, `timeout`, `unavailable`, `content_filter`

## Implementations (planned)
Ollama, Gemini, OpenAI, Anthropic—selected by configuration.
