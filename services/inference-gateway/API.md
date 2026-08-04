# Inference Gateway API

## Purpose

Define the initial service-facing contract for normalized chat and embedding requests.

## Scope

These are draft internal HTTP/JSON shapes. The provider ports remain the authoritative abstraction boundary; an OpenAPI contract should be generated or formalized before implementation is merged.

Requests may include `X-Request-ID`. If absent, the gateway generates one and returns it in the response header. The identifier is also added to provider metadata and structured logs.

## Dependencies

- [PRD](./PRD.md)
- [Data Model](./DATA_MODEL.md)
- [Chat Provider Contract](../../contracts/chat-provider.md)
- [Embedding Provider Contract](../../contracts/embedding-provider.md)

## Owner

Atlas Core Engineering

## Status

Implemented subset documented in OpenAPI (including SSE streaming generate)

## Last Updated

2026-08-04

## Related Documents

- [OpenAPI 3 (implemented routes)](./openapi.yaml) — also `GET /openapi.yaml` on a running gateway
- [Test Plan](./TEST_PLAN.md)
- [Communication Protocols ADR](../../docs/adr/0004-communication-protocols.md)

## Generate chat

`POST /v1/generate`

Request fields:

- `messages[]`: role and content parts.
- `task_class` (optional): logical class such as `plan` or `act`; defaults to `act`.
- `model` (optional): logical model override subject to policy.
- `tools[]` (optional): names and JSON schemas available to the model.
- `temperature`, `max_tokens` (optional): bounded generation controls.
- `metadata`: `workspace_id`, `conversation_id`, `run_id`, `step_id`, and request correlation data.
- `stream` (optional): when `true`, the response is **SSE** (`text/event-stream`) instead of a single JSON body.

Response fields (non-streaming JSON):

- `message`: normalized assistant content and optional tool calls.
- `usage`: input/output token counts when known.
- `provider`, `model`: resolved concrete identifiers.
- `finish_reason`.
- `request_id` and `latency_ms` (headers / telemetry; not always duplicated in body).

### Streaming (`stream: true`)

Content-Type: `text/event-stream`. Events:

| Event | Data payload |
|-------|----------------|
| `delta` | `{"delta":"...","provider":"...","model":"..."}` content fragment |
| `done` | Full `ChatResponse` JSON (final assembled message + usage) |
| `error` | `{"error":{"code":"...","message":"..."}}` if generation fails after the stream starts |

Headers `X-Request-ID` and `X-Run-ID` are still set. Fake, Ollama, and OpenAI-compatible providers stream natively; other adapters synthesize chunks after a full generate.

## Create embeddings

`POST /v1/embeddings`

Request fields:

- `inputs[]`: text strings, with a configured batch limit.
- `model` (optional): logical embedding model id.
- `metadata`: correlation data.

Response fields:

- `vectors[]`: vectors aligned with the input order.
- `dimensions`.
- `usage` (optional).
- `provider`, `model`, `latency_ms`, and `request_id`.

## Conversations

`POST /v1/conversations` creates a durable conversation. The optional request body is `{ "title": "..." }`.

`GET /v1/conversations/{conversation_id}` returns conversation metadata and ordered persisted messages.

`POST /v1/conversations/{conversation_id}/messages` accepts `{ "content": "...", "task_class": "...", "model": "...", "metadata": {} }`. It appends the user message, generates an assistant response from the full history, and appends that response with the resulting run ID.

## Health and readiness

- `GET /healthz`: process health.
- `GET /readyz`: configuration and provider readiness without making an unbounded generation call.
- `GET /v1/runs/{run_id}`: inspect a persisted inference run.
- `POST /v1/conversations`: create a conversation when PostgreSQL persistence is configured.
- `GET /v1/conversations/{conversation_id}`: fetch conversation metadata and messages.
- `POST /v1/conversations/{conversation_id}/messages`: append a user message and generate an assistant response.

## Error classes

`invalid_request`, `unsupported_capability`, `auth`, `rate_limit`, `timeout`, `unavailable`, `content_filter`, `provider_error`, and `internal`. `rate_limit`, `timeout`, and `unavailable` are retryable; the remaining classes are not retryable by default.

Errors must include a safe public message, stable class/code, retryability indication, and correlation id. Raw provider secrets, prompts, and credentials must not be returned.
