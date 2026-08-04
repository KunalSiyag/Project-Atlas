# Inference Gateway TODO

## Purpose

Track work required to move this service from planning to implementation.

## Scope

Applies to the M1 inference gateway service pack.

## Dependencies

- [PRD](./PRD.md)
- [API](./API.md)
- [Test Plan](./TEST_PLAN.md)

## Owner

Atlas Core Engineering

## Status

Draft

## Last Updated

2026-08-04

## Tasks

- [x] Convert draft API shapes into versioned OpenAPI or equivalent contracts.
- [x] Define Go module, service layout, and configuration key reference.
- [x] Define provider adapter and router ports in code.
- [x] Implement fake provider and deterministic test fixtures.
- [x] Implement local provider path and provider-aware health/readiness checks.
- [x] Add normalized errors, bounded retry, timeout, and cancellation behavior.
- [x] Add request IDs and safe structured logging.
- [x] Add OpenTelemetry spans and metrics API instrumentation.
- [x] Configure an OpenTelemetry SDK/exporter for local Compose and production deployment.
- [x] Add PostgreSQL-backed inference-run persistence and migration.
- [x] Add persisted run inspection endpoint.
- [x] Add conversation persistence schema and create/get endpoints.
- [x] Connect conversation messages to full-history inference and run IDs.
- [x] Add Compose wiring after the local data-plane design is accepted.
- [x] Accept empty-body conversation create for simple local clients.
- [x] Add optional hosted adapters only after the conformance suite is green.
- [x] Add embeddings HTTP path with fake (and OpenAI-compatible) embedders.
- [x] Chat provider conformance suite (fake + OpenAI-style).

## Notes

- Compose lives under `deploy/docker-compose.yml` (gateway + Postgres + Redis + Qdrant + MinIO + Ollama).
- Beginner fake-provider overlay: `deploy/docker-compose.fake.yml` (no Ollama required).
- Terminal client: `clients/atlas-cli/` (`make cli-run` after `make gateway-run`).
- OpenAPI (implemented routes only): `services/inference-gateway/openapi.yaml`, also served at `GET /openapi.yaml`.
- Local smoke: `./scripts/dev_smoke.sh`
