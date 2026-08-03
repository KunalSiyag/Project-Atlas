# Architecture: Inference and Model Router

## Purpose
Describe provider-independent model invocation and routing by task class.

## Scope
Chat/completion and embedding calls, routing, and metering. Training models is out of scope.

## Dependencies
- [Architecture Overview](./OVERVIEW.md)
- [Domain: Provider](../domain/provider.md)
- [ADR 0002: Provider Independence](../adr/0002-provider-independence.md)
- [Context Builder](./context-builder.md)

## Owner
Atlas Architecture

## Status
Accepted

## Last Updated
2026-08-03

## Related Documents
- [Evaluation](./evaluation.md)
- [Observability](./observability.md)
- [Contracts: chat-provider](../../contracts/README.md)

## Why It Exists
Almost every subsystem needs models. Centralizing inference behind interfaces enables local free development, commercial benchmarks, and per-task routing (plan vs code vs vision vs embed).

## Problem Solved
- Hard-coded `OpenAI.ChatCompletion` in business logic
- No cost/latency comparison across providers
- Inability to run offline

## Consumers
- Planner, response generator, optional judges
- Embedding pipelines for RAG/memory
- Benchmarks

## Public Interfaces (conceptual)
```text
type ModelProvider interface {
    Generate(ctx, Request) (Response, error)
}

type EmbeddingProvider interface {
    Embed(ctx, EmbedRequest) (EmbedResponse, error)
}

type ModelRouter interface {
    Resolve(taskClass, AgentRoutingProfile) (Provider, ModelID)
}
```
- Task classes (examples): `plan`, `act`, `code`, `vision`, `embed`, `judge`
- Response includes text/tool-calls, usage tokens, provider/model ids, latency

## Dependencies
- Provider adapters (Ollama, Gemini, OpenAI, Anthropic, …)
- Configuration (ADR 0006)
- Secrets management
- Rate limits / quotas
- OpenTelemetry instrumentation

## Failure Modes
| Failure | Handling |
|---------|----------|
| Provider 429 | Backoff; optional failover model if configured |
| Provider 5xx | Retry budget; circuit breaker |
| Auth misconfig | Fast fail; do not retry forever |
| Tool-call JSON invalid | Repair attempt once; then step failure |
| Local GPU OOM | Error with guidance to smaller/quantized model |
