# Architecture: Deployment

## Purpose
Describe how Atlas is packaged and run across environments, starting local-first.

## Scope
Deployment topology principles for foundation and early milestones. Full Kubernetes production hardening is later.

## Dependencies
- [Architecture Overview](./OVERVIEW.md)
- [Storage](./storage.md)
- [Observability](./observability.md)
- [ADR 0001](../adr/0001-documentation-first-modular-monorepo.md)

## Owner
Atlas Architecture

## Status
Accepted

## Last Updated
2026-08-03

## Related Documents
- [Configuration ADR](../adr/0006-configuration-strategy.md)
- [management/ROADMAP](../../management/ROADMAP.md)

## Why It Exists
A platform that only runs on a developer laptop with ad-hoc processes is not operable. Deployment design must exist even before all services are implemented so boundaries stay deployable.

## Problem Solved
- Hidden dependencies (“works on my machine”)
- Premature Kubernetes complexity before Compose proves the graph
- Secrets baked into images

## Consumers
- Developers (`docker compose up` goal for M1+)
- Operators
- CI environments

## Public Interfaces (conceptual)
- Compose file(s) under `deploy/` for the local data plane
- Per-service container entrypoints and health checks
- Config via environment / files (see ADR 0006)
- Later: Kubernetes manifests or Helm after Compose path is stable

## Target Topology (development)
```text
[gateway] [orchestrator] [inference-gateway]
    |          |                |
    +-----+----+------+---------+
          |           |
     Postgres      Redis
     Qdrant        MinIO
     Ollama (optional host network)
     Prometheus / Grafana / OTel collector
```

## Dependencies
- Docker / Compose for local
- Free/local data plane defaults
- No paid cloud required for clone-and-run development

The initial data-plane stack is documented in [`deploy/README.md`](../../deploy/README.md) and defined in [`deploy/docker-compose.yml`](../../deploy/docker-compose.yml). The inference gateway is the first containerized Atlas service; additional services will be added after their container entrypoints and migrations exist.

## Failure Modes
| Failure | Handling |
|---------|----------|
| Service start order | Healthchecks + depends_on conditions |
| Volume data loss | Document named volumes; backup notes in runbooks |
| Port conflicts | Configurable host ports |
| Secret leakage in compose overrides | `.env` gitignored; examples only in `.env.example` |

## Non-Goals for Early Deployment
- Multi-region active-active
- Service mesh mandatory
- Custom schedulers
