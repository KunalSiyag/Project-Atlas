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

## Now
No active sprint-critical engineering checkboxes. Prior Next/Later product slices below are shipped; only Icebox and optional deeper residual remain.

### Completed (reference — struck work formerly “Now” / “Next” / “Later”)
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
17. ~~Orchestrator list-events HTTP, OpenAPI serve, workspace middleware tests, Compose Postgres long memory, expanded eval fixture~~ (task 0006-a…e).
18. ~~Prompt hot-reload + Postgres publish path~~ (`PromptRegistry.Reload`, `PostgresPromptStore` / `MemoryPromptStore`).
19. ~~Tool authz on Invoke + registerable HTTP remote plugin tools~~ (`AllowlistAuthorizer`, `RegisterDescriptor` / `RemoteHTTPTool`) — not a WASM runtime.
20. ~~Kubernetes base manifests + production hardening notes~~ (`deploy/k8s/*`, deploy README).
21. ~~Eval CI-style gate~~ (`make eval-gate`, `.github/workflows/eval-gate.yml`).
22. ~~Prompt admin HTTP multi-tenant publish/list/get~~ (`/v1/prompts`, workspace isolation).
23. ~~M5 deeper package~~: runbooks library, load/bench dashboard + `make bench`, in-repo security audit.
24. ~~LLM-as-judge product pipeline~~ (`GatewayJudge` + fixture `sample_with_judge.json` through harness).
25. ~~GitHub/Slack HTTP adapters~~ (`github_create_issue`, `slack_post_message`, configurable base URL).
26. ~~M4 corpus ops + search SLA metrics~~ (list/delete/status, `MeteredRetriever`).
27. ~~Multi-region design doc~~ (design only; 0006-f remains icebox).
28. ~~Plugin marketplace catalog~~ (`/v1/plugins` discovery + register; WASM VM still icebox).

## Next
- Optional: run-store workspace partition + default-on tool allowlists (security audit residual).
- Optional: authenticated ingress / OIDC product path (SA-01).

## Later
- External pen-test / SOC2 (out of engineering backlog).
- Active multi-region HA implementation (see icebox 0006-f; design doc exists).

## Icebox
- Production multi-region / HA orchestration (task 0006-f).
- WASM / remote tool plugins as WASM guest runtime (task 0006-g; HTTP remote register + catalog shipped separately).
- Autonomous multi-agent swarms.
- Self-modifying agents.
- Exactly-once global messaging.
