# Local Atlas Data Plane

## Purpose

Provide the local development dependencies used by Atlas services.

## Scope

This Compose stack builds the **inference gateway** and **agent orchestrator** and runs them with PostgreSQL, Redis, Qdrant, MinIO, and Ollama.

The orchestrator is always given `ATLAS_ORCHESTRATOR_DATABASE_URL` pointing at Compose `postgres`, so **long-term memory uses Postgres** via `OpenPostgresMemory` (and runs via `OpenPostgresStore`) whenever the stack is up — not an in-memory-only default.

## Start

Full stack (includes Ollama for real local models):

```bash
cd deploy
cp .env.example .env
docker compose up -d
docker compose ps
```

Beginner / offline path (fake provider, no model download — best for CLI testing):

```bash
cd deploy
cp .env.example .env
docker compose -f docker-compose.yml -f docker-compose.fake.yml up --build
```

Or from the repository root: `make compose-fake`.

Without Docker, run only the gateway + fake provider:

```bash
make gateway-run
# other terminal:
make cli-health
make cli-generate MSG='hello atlas'
make cli-run
```

Stop the services without deleting data:

```bash
docker compose down
```

To remove the named development volumes as well, use `docker compose down -v`. This deletes local database, vector, object, and model data.

## Endpoints

| Service | Host endpoint | Purpose |
|---|---|---|
| Inference gateway | `localhost:8080` | Chat inference API |
| Agent orchestrator | `localhost:8090` | Runs, tools, memory, RAG control plane |
| PostgreSQL | `localhost:5432` | Relational state (gateway runs + orchestrator runs/events + long memory) |
| Redis | `localhost:6379` | Cache, rate limits, short-term state |
| Qdrant HTTP | `localhost:6333` | Vector search |
| Qdrant gRPC | `localhost:6334` | Vector client path |
| MinIO API | `localhost:9000` | S3-compatible objects |
| MinIO console | `localhost:9001` | Local object-store UI |
| Ollama | `localhost:11434` | Local model API |

Inside the Compose network, services use hostnames such as `postgres`, `redis`, `qdrant`, `minio`, `ollama`, and `inference-gateway` rather than `localhost`. The gateway is configured with `ATLAS_OLLAMA_URL=http://ollama:11434` for this reason.

The gateway uses PostgreSQL through `ATLAS_DATABASE_URL` and automatically applies the `inference_runs` and `conversations` schemas on startup. Each request returns an `X-Run-ID` that can be used to inspect its persisted lifecycle record.

The **agent orchestrator** uses the same Postgres credentials via `ATLAS_ORCHESTRATOR_DATABASE_URL` (and `ATLAS_DATABASE_URL` fallback). On startup it opens `PostgresStore` for runs/events and `PostgresMemory` for long-term facts, plus optional Redis short-term (`ATLAS_REDIS_URL`) and Qdrant (`ATLAS_QDRANT_URL`). Prompt files are bind-mounted from repo `prompts/` to `/prompts`.

```bash
curl http://localhost:8090/healthz
curl http://localhost:8090/openapi.yaml | head
curl -X POST http://localhost:8090/v1/runs -H 'Content-Type: application/json' -d '{"goal":"echo hello"}'
curl http://localhost:8090/v1/runs/<run-id>/events
```

Inspect a run:

```bash
curl http://localhost:8080/v1/runs/<run-id>
```

Create and inspect a conversation:

```bash
curl -X POST http://localhost:8080/v1/conversations \
  -H 'Content-Type: application/json' \
  -d '{"title":"Local experiment"}'
curl http://localhost:8080/v1/conversations/<conversation-id>
curl -X POST http://localhost:8080/v1/conversations/<conversation-id>/messages \
  -H 'Content-Type: application/json' \
  -d '{"content":"Summarize our discussion"}'
```

## Notes

- Images use development-oriented defaults; pin and scan images before production use.
- Credentials in `.env.example` are intentionally weak local defaults and must be replaced outside a personal workstation.
- Ollama does not automatically download a model. After startup, pull one with `docker compose exec ollama ollama pull gemma3` if the image supports that command.
- The gateway image is built from `services/inference-gateway/Dockerfile` and runs as a non-root user.
- The named volumes are persistent. Backups and restore procedures are not yet implemented.

## Observability (OTEL + Grafana)

Optional overlay starts an OpenTelemetry Collector and Grafana with a provisioned Prometheus datasource and an Atlas gateway dashboard.

```bash
cd deploy
cp -n .env.example .env 2>/dev/null || true
# Full stack with Ollama + OTEL:
docker compose -f docker-compose.yml -f docker-compose.otel.yml up -d
# Offline fake provider + OTEL:
docker compose -f docker-compose.yml -f docker-compose.fake.yml -f docker-compose.otel.yml up -d
```

| Service | URL / port | Notes |
|---------|------------|--------|
| Grafana | http://localhost:3000 | user `admin` / password `atlas-dev-password` (override via env) |
| OTLP gRPC | localhost:4317 | traces/metrics from apps |
| OTLP HTTP | localhost:4318 | used by gateway when `ATLAS_OTEL_EXPORTER=otlp` |
| Prometheus scrape | localhost:8889 | collector exporter scraped by Grafana |

Config files:

- `deploy/otel/otel-collector-config.yaml`
- `deploy/grafana/provisioning/datasources/datasources.yaml`
- `deploy/grafana/provisioning/dashboards/dashboards.yaml`
- `deploy/grafana/dashboards/atlas-gateway.json`

Without Docker, validate the compose graph:

```bash
docker compose -f docker-compose.yml -f docker-compose.otel.yml config
```

## Kubernetes

Static manifests for gateway + orchestrator live under [`k8s/`](./k8s/):

| File | Workload |
|------|----------|
| [`k8s/inference-gateway.yaml`](./k8s/inference-gateway.yaml) | Deployment + Service (port 8080, probes, resources) |
| [`k8s/agent-orchestrator.yaml`](./k8s/agent-orchestrator.yaml) | Deployment + Service (port 8090, probes, resources) |
| [`k8s/README.md`](./k8s/README.md) | Apply notes and secret bootstrap |

```bash
kubectl apply -f deploy/k8s/inference-gateway.yaml
kubectl apply -f deploy/k8s/agent-orchestrator.yaml
```

These are a **single-cluster base**, not multi-region / active-active HA (that remains icebox task 0006-f).

## Production hardening

Before exposing Atlas outside a personal workstation:

1. **Secrets** — never bake DB URLs, provider API keys, or MinIO credentials into images. Use Kubernetes Secrets (or your secret manager) for `ATLAS_DATABASE_URL` / `ATLAS_ORCHESTRATOR_DATABASE_URL` and provider keys. Rotate local Compose `.env` defaults.
2. **Config** — pin image digests; set `ATLAS_REQUIRE_WORKSPACE=1` on the orchestrator; mount prompts read-only (`ATLAS_PROMPTS_DIR` / ConfigMap). Prefer file hot-reload + Postgres prompt publish over baking prompts into the binary.
3. **Probes** — readiness and liveness hit `/healthz` on both services (see k8s manifests). Do not route traffic until ready.
4. **Resources** — set CPU/memory requests and limits (manifests include baseline requests). Right-size under load; add HPA only after measuring.
5. **Security** — run as non-root (UID 10001 in Dockerfiles), drop capabilities, read-only root filesystem where possible, NetworkPolicies limiting egress to gateway/DB/Redis/Qdrant. Enable TLS at the ingress; do not expose Postgres/Redis/MinIO publicly.
6. **Authz** — tool allowlists (`AllowlistAuthorizer` on the orchestrator tool registry) and approval gates for dangerous tools; do not leave unrestricted remote HTTP tools pointed at untrusted networks.
7. **Eval gate** — run `make eval-gate` (or CI workflow) so fixture regressions fail the pipeline before promote.
8. **Observability** — OTEL exporter + dashboards for sample load; retain traces for failed runs. Full load/bench product dashboards and incident runbook library remain M5 residual.

This section is intentional hardening guidance, not a claim that the full M5 production suite (runbooks library, multi-region mesh, enterprise audit) is complete.

