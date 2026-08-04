# Changelog

## Purpose
Record notable changes to the Atlas repository for humans and release notes.

## Scope
Repository-level changes. Prefer ADRs for architectural decisions; this file summarizes.

## Dependencies
- [Releases](./RELEASES.md)
- [README](../README.md)

## Owner
Atlas Core Engineering

## Status
Accepted

## Last Updated
2026-08-05

## Related Documents
- [Milestones](./MILESTONES.md)

## Format
Keep a human-readable list. When versioned releases begin, adopt Keep a Changelog style under version headings.

## Unreleased

### Added
- Hybrid **LLMPlanner** (gateway JSON plan + ValidatePlan) with **RulePlanner** fallback; Redis short-term memory (`ATLAS_REDIS_URL`); RAG chunking/ranking/score metadata.
- Offline **eval harness** `tools/atlas-eval` + fixture format `fixtures/eval/v1/`.
- Compose observability overlay `deploy/docker-compose.otel.yml` (OTEL collector + Grafana provisioning/dashboard).
- ADR 0009 context-builder token budgeting; RFC 0001 Tool SDK interface; prompt registry design (`docs/architecture/prompt-registry.md`).
- SSE streaming generate (`stream: true` on `POST /v1/generate`) with native fake/Ollama/OpenAI streams and CLI `-stream`.
- Runnable inference gateway (`services/inference-gateway`) with fake/Ollama providers, HTTP APIs, retries, run + conversation Postgres stores.
- OpenAPI 3 contract for implemented routes (`openapi.yaml`, served at `GET /openapi.yaml`), including embeddings.
- OpenTelemetry SDK export (`ATLAS_OTEL_EXPORTER=none|console|otlp`).
- OpenAI-compatible chat + embeddings adapters; chat provider conformance suite.
- `POST /v1/embeddings` with deterministic fake embedder.
- First agent-orchestrator Go module (in-memory run lifecycle + HTTP).
- Orchestrator executes plan DAG against inference gateway, context budget builder, Postgres run store, cancel/retry.
- Tool steps (echo/calculator/http_get/danger_echo), schema validation, timeouts, approval/reject APIs.
- Thin memory (short+long), dynamic rule planner, in-process/Qdrant RAG, CLI `atlas run` agent mode.
- Terminal client `clients/atlas-cli` (ATLAS Unicode banner, health/generate/chat, run inspect, conversation show, openapi fetch).
- Local Compose data plane under `deploy/` plus `docker-compose.fake.yml` for offline fake-provider testing.
- Root `Makefile` (`gateway-run`, `cli-run`, `test`, `smoke`, `compose-fake`).
- `scripts/dev_smoke.sh` one-command local smoke.
- Beginner teaching doc: `docs/engineering/BEGINNER_BUILD_RECORD.md`.
- Empty-body support for `POST /v1/conversations`.
- Product constitution under `docs/product/` (vision, goals, non-goals, personas, FR, NFR, success metrics, glossary).
- Domain language under `docs/domain/` (workspace, agent, conversation, memory, tool, provider, evaluation, prompt, workflow).
- Subsystem architecture docs under `docs/architecture/`.
- Foundation ADRs 0002–0008 (provider independence, architecture style, protocols, storage, configuration, testing, OpenTelemetry).
- Management layer under `management/`.
- Indexes for `contracts/`, `schemas/`, `tasks/`.
- Updated root README and AI_CONTEXT for M0-complete navigation.

### Previously
- Bootstrap documentation-first monorepo, ADR 0001, engineering standards, agent-orchestrator service doc shell.
