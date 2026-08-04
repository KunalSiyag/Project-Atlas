# Backlog

## Purpose
Prioritize upcoming engineering work after M0 foundation.

## Scope
Near-term backlog items. Not a full issue tracker export.

## Dependencies
- [Milestones](./MILESTONES.md)
- [Tasks](../tasks/README.md)

## Owner
Atlas Core Engineering

## Status
Accepted

## Last Updated
2026-08-05

## Related Documents
- [Roadmap](./ROADMAP.md)
- [Changelog](./CHANGELOG.md)

## Now (M1 in progress)
1. ~~Author chat/embedding contracts and core schemas~~ (present under `contracts/`, `schemas/`).
2. ~~Inference gateway docs + first Go implementation~~ (`services/inference-gateway`).
3. ~~Local Compose data plane~~ (`deploy/`).
4. ~~Terminal client + OpenAPI for implemented routes~~ (`clients/atlas-cli`, `openapi.yaml`).
5. ~~OpenTelemetry SDK/exporter~~ (`ATLAS_OTEL_EXPORTER`).
6. ~~OpenAI-style chat + conformance suite~~.
7. ~~Embeddings HTTP path~~ (`POST /v1/embeddings`).
8. ~~Agent-orchestrator first code slice~~ (in-memory runs).
9. ~~Wire orchestrator → inference gateway~~ (StartRun → `/v1/generate`).
10. ~~Persist orchestrator runs (Postgres) + real plan DAG (deps/retry/cancel) + context builder~~.
11. ~~Tool steps / approvals~~ (echo, calculator, http_get, danger_echo + approve API).
12. ~~Dynamic rule planner + memory + first RAG path + CLI `atlas run`~~.
13. ~~Richer LLM planner, real Redis short-term, production RAG quality~~.
14. ~~Streaming generate~~ (SSE); ~~eval harness~~ (`tools/atlas-eval`); ~~Compose OTEL/Grafana~~ (`deploy/docker-compose.otel.yml`).
15. ~~Context-builder ADR, Tool SDK RFC, prompt registry, eval fixture format~~.
16. ~~Prompt registry file loader; Tool SDK refactor; LLM-as-judge eval path~~.

## Next
- Kubernetes manifests / production hardening (M5).
- Prompt hot-reload and Postgres publish path for registry.
- Remote/WASM tool plugins.

## Later
- Kubernetes manifests.
- Multi-region considerations.
- LLM-as-judge optional pipeline.
- Real GitHub/Slack adapters beyond mocks.

## Icebox
- Autonomous multi-agent swarms.
- Self-modifying agents.
- Exactly-once global messaging.
