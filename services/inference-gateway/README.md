# Inference Gateway Service

## Purpose

Provide a provider-independent entry point for chat/completion and embedding inference used by Atlas services.

## Scope

This service owns request validation, logical model resolution, provider adapter invocation, bounded retry/fallback behavior, usage metering, and inference telemetry. It does not own agent planning, workflow state, tool execution, or prompt business logic.

## Dependencies

- [Service AI Context](./AI_CONTEXT.md)
- [Service PRD](./PRD.md)
- [Service API](./API.md)
- [Service Data Model](./DATA_MODEL.md)
- [Configuration](./CONFIGURATION.md)
- [Chat Provider Contract](../../contracts/chat-provider.md)
- [Embedding Provider Contract](../../contracts/embedding-provider.md)

## Owner

Atlas Core Engineering

## Status

Planning

## Last Updated

2026-08-03

## Related Documents

- [Inference Architecture](../../docs/architecture/inference.md)
- [Provider Independence ADR](../../docs/adr/0002-provider-independence.md)
- [Configuration ADR](../../docs/adr/0006-configuration-strategy.md)

## Current State

The first implementation slice now exists as a small Go module with normalized chat types, `ModelProvider`, `ModelRouter`, and `RunStore` ports, deterministic fake and optional Ollama providers, normalized errors with bounded retries, request IDs and safe structured logs, OpenTelemetry spans/metrics, PostgreSQL-backed run persistence, unit tests, a standard-library HTTP server, and environment-based startup configuration.
