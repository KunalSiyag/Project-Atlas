# Multi-region considerations (design only)

## Purpose
Capture design considerations for multi-region Atlas deployments **without** implementing a multi-cluster active-active mesh.

## Status
Design only. Production multi-region / HA orchestration remains **icebox task 0006-f**.

## Last Updated
2026-08-05

## Goals (future)
- Survive full regional outage with RPO/RTO targets agreed by product
- Keep provider independence (local adapters still work in each region)

## Non-goals (now)
- Multi-cluster failover automation
- Globally exactly-once messaging (icebox)
- Cross-region active-active write to the same run IDs

## Topology options (when 0006-f leaves icebox)

| Mode | Description | Fit |
|------|-------------|-----|
| Active-passive | Single primary region; warm standby with DB replica promote | First production HA |
| Active-active reads | Writes pin to home region; read replicas elsewhere | Lower latency reads |
| Active-active writes | CRDT/partitioned workspaces | Highest complexity; not recommended early |

## Data plane
- **Postgres runs/events/prompts:** regional primary; logical workspace affinity
- **Redis short-term memory:** regional, ephemeral — do not cross-replicate chat buffers
- **Qdrant / object store:** per-region corpora or async replication with stale-read tolerance
- **Inference:** provider adapters may be global SaaS; local Ollama remains regional

## Control plane
- Sticky workspace → region mapping in the edge gateway
- Run IDs remain region-local; no global run namespace without a federation layer
- Prompt publish is workspace-scoped today — multi-region publish would need outbox replication

## Failure modes
| Failure | Design response |
|---------|-----------------|
| Region down | DNS/ingress fail away; accept RPO of async replica lag |
| Split brain | Prefer fencing primary over dual writers |
| Provider regional outage | Fail over model adapter independently of control plane |

## What is shipped today
- Single-cluster Kubernetes base manifests
- Single-region Compose data plane
- Workspace-scoped prompt store (tenant isolation seed, not geo)

## Exit criteria to leave icebox
- Documented RPO/RTO and chosen topology
- Proven DB promote + orchestrator drain runbook
- Load test of regional failover (bench + dashboards)
