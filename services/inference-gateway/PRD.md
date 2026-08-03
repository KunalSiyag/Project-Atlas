# Inference Gateway PRD

## Purpose

Define the first runtime service that centralizes model and embedding invocation for Atlas.

## Scope

The service provides synchronous and streaming-capable chat generation plus batch embedding calls through provider-independent ports. Routing, retries, metering, and telemetry are in scope; model training and agent planning are not.

## Dependencies

- [Inference Architecture](../../docs/architecture/inference.md)
- [Functional Requirements](../../docs/product/FUNCTIONAL_REQUIREMENTS.md)
- [Chat Provider Contract](../../contracts/chat-provider.md)
- [Embedding Provider Contract](../../contracts/embedding-provider.md)

## Owner

Atlas Product Engineering

## Status

Draft

## Last Updated

2026-08-03

## Related Documents

- [API](./API.md)
- [Data Model](./DATA_MODEL.md)
- [Test Plan](./TEST_PLAN.md)

## Goal

Make model selection and invocation a replaceable platform capability so planners, response generators, evaluators, and embedding pipelines do not depend on one vendor.

## Users and consumers

- Planner and workflow services requesting plan or action generation.
- Response generation paths requesting chat output or tool-call proposals.
- Memory and knowledge pipelines requesting embeddings.
- Evaluation and benchmark jobs comparing providers and models.
- Operators inspecting latency, errors, usage, and provider health.

## Requirements

- Accept logical task classes such as `plan`, `act`, `code`, `vision`, `embed`, and `judge`.
- Resolve a configured provider and concrete model for each request.
- Route by logical task class so planning and acting can use different models.
- Support local/fake provider adapters first; the initial Ollama adapter is local and hosted adapters remain optional.
- Validate request size, supported capabilities, and model configuration before provider calls.
- Return normalized content, tool calls, finish reason, usage, provider/model identity, and error class.
- Emit trace spans and metrics through the OpenTelemetry API without requiring a specific backend.
- Support bounded timeout, retry, backoff, and optional failover policy.
- Preserve workspace/run correlation metadata without exporting prompt secrets.
- Expose health/readiness information for configured dependencies.

## Non-goals

- Owning prompts, conversations, plans, workflows, or tools.
- Training or fine-tuning models.
- Hiding provider-specific limitations; unsupported capabilities must be explicit.
- Making LLM-as-judge mandatory for normal requests.

## Acceptance criteria

- A fake provider can complete a chat request through the same port used by real adapters.
- A local provider path can be selected through configuration without domain-code changes.
- Provider errors normalize into documented classes and preserve retryability semantics.
- Usage and latency metadata are available to evaluation and observability paths.
- Contract and integration tests cover timeout, rate limit, invalid response, and fallback behavior.
