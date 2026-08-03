# Inference Gateway Data Model

## Purpose

Define the normalized request, response, routing, and usage records used by the inference gateway.

## Scope

Conceptual service data; durable storage is not required for every record. Run-level usage and error metadata must be available to observability/evaluation sinks.

## Dependencies

- [PRD](./PRD.md)
- [API](./API.md)
- [Provider Domain](../../docs/domain/provider.md)
- [Observability Architecture](../../docs/architecture/observability.md)

## Owner

Atlas Core Engineering

## Status

Draft

## Last Updated

2026-08-03

## Entities

### InferenceRequest

An authenticated, correlated request for chat generation or embeddings. It contains a task class, normalized inputs, routing hints, capability requirements, budget, and metadata.

### RoutingProfile

Configuration that maps task classes to a logical model and provider policy, including allowed providers, fallback order, timeout, retry budget, and rate limits.

### ProviderBinding

The resolved adapter and concrete model identifier selected for one request. It must retain logical and concrete identity for debugging and comparison.

### InferenceResult

Normalized content or vectors plus finish status, usage, provider/model identity, latency, and optional warnings.

### InferenceFailure

A normalized error class with provider-safe detail, retryability, attempt count, and correlation ids.

## Invariants

- A request has one correlation id and one resolved routing decision per attempt.
- Embedding output order matches input order.
- Retry and fallback are bounded by request and policy budgets.
- Provider-specific response types are converted at the adapter boundary.
- Usage values may be unknown, but unknown usage is represented explicitly rather than guessed.
- Sensitive input content is not stored in ordinary metrics labels or error messages.
