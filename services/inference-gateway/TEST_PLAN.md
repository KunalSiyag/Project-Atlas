# Inference Gateway Test Plan

## Purpose

Define tests that prove normalized inference behavior without coupling the gateway to a live commercial provider.

## Scope

Unit, contract, integration, and optional provider-conformance tests for chat generation, embeddings, routing, retries, errors, and telemetry.

## Dependencies

- [PRD](./PRD.md)
- [API](./API.md)
- [Data Model](./DATA_MODEL.md)
- [Testing Philosophy ADR](../../docs/adr/0007-testing-philosophy.md)

## Owner

Atlas Core Engineering

## Status

Draft

## Last Updated

2026-08-03

## Test categories

- **Unit:** request validation, task-class routing, budget checks, error mapping, retry policy, and usage normalization using fakes.
- **Port contract:** every adapter satisfies chat and embedding interfaces for success, malformed response, timeout, rate limit, and unavailable cases.
- **HTTP contract:** request/response/error JSON shapes, health endpoints, correlation headers, and streaming framing when enabled.
- **Integration:** gateway with a local fake server or local provider, configuration overlays, and readiness behavior.
- **Resilience:** bounded retries, fallback selection, circuit behavior, cancellation, and no retry on invalid credentials or invalid requests.
- **Security:** secret redaction, metadata isolation, request-size limits, and rejection of unauthorized model/provider overrides.
- **Telemetry:** spans and metrics contain request/run/provider/model identifiers without raw prompt labels.

## Acceptance criteria

- Tests pass with no network or commercial credentials for the default suite.
- Provider swap tests exercise the same normalized contract.
- Every documented error class has deterministic coverage.
- Streaming cancellation does not leave an unbounded provider call.
- Token/cost fields remain correct when a provider reports partial or no usage.
