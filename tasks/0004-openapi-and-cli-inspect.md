# Task 0004 — OpenAPI contract + CLI inspect commands

## Goal
Publish an OpenAPI document for implemented inference-gateway routes and extend atlas-cli to inspect runs/conversations and fetch the contract.

## Inputs
- Existing gateway handlers and JSON shapes
- `clients/atlas-cli` HTTP client

## Outputs
- `services/inference-gateway/openapi.yaml`
- `GET /openapi.yaml` (embedded serve)
- CLI: `run`, `conversation`, `openapi`
- `scripts/dev_smoke.sh`, `make smoke`

## Acceptance criteria
- [x] OpenAPI lists only implemented paths (no fake `/v1/embeddings` path)
- [x] Gateway unit tests cover embed + serve
- [x] CLI client tests cover GetRun + OpenAPI
- [x] `make smoke` (or `./scripts/dev_smoke.sh`) passes against fake provider

## Status
Done (2026-08-04)
