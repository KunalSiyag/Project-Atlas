# Agent Orchestrator API

## Purpose
Document the concrete HTTP control surface implemented by the agent orchestrator.

## Scope
Implemented routes only (see machine-readable [openapi.yaml](./openapi.yaml)). Conceptual draft shapes (`workflow_id`, `execution_id`) are obsolete.

## Dependencies
- [PRD](./PRD.md)
- [Data Model](./DATA_MODEL.md)
- [OpenAPI](./openapi.yaml)
- [Handler](./handler.go)

## Owner
Atlas Agent Runtime Team

## Status
Accepted

## Last Updated
2026-08-05

## Related Documents
- [Test Plan](./TEST_PLAN.md)
- [Events schema](./events.schema.json)
- [Service README](./README.md)

## Base URL
Default local listen: `http://127.0.0.1:8090` (`ATLAS_ORCHESTRATOR_ADDR`).

Optional isolation: when `ATLAS_REQUIRE_WORKSPACE=1`, clients should send `X-Workspace-ID`.

## Endpoints

### Health

| Method | Path | Success | Body |
|--------|------|---------|------|
| `GET` | `/healthz` | 200 | `{"status":"ok","service":"agent-orchestrator"}` |
| `GET` | `/readyz` | 200 / 503 | ready when service is wired |

### Runs

#### `POST /v1/runs` — create (plan + persist pending)

Request:
```json
{ "goal": "calculate 21+21" }
```

Response `201`: full `Run` object (`run_id`, `goal`, `status=pending`, `steps[]`, timestamps).

Errors: `400` missing/invalid goal; `503` service not ready.

#### `GET /v1/runs/{run_id}` — inspect

Response `200`: current `Run` snapshot. `404` if unknown.

#### `GET /v1/runs/{run_id}/events` — list lifecycle events

Response `200`: JSON array of `Event` objects (`event_id`, `run_id`, `step_id?`, `type`, `message`, `created_at`) from `RunStore.ListEvents`. Empty array if the run exists but has no events yet.

Errors: `404` if run unknown.

#### `GET /openapi.yaml` — OpenAPI document

Also `/openapi.yml` and `/v1/openapi.yaml`. Content-Type `application/yaml`; body is the embedded contract.

#### `POST /v1/runs/{run_id}/start` — execute or resume

Runs the plan executor. Response `200`: updated `Run` (may be terminal or `waiting_approval`).

Errors: `409` conflict (not startable status / executor errors).

#### `POST /v1/runs/{run_id}/approve` — approve waiting tool step(s)

Marks steps in `waiting_approval` as approved and continues execution.

Errors: `409` if no step is waiting.

#### `POST /v1/runs/{run_id}/reject` — reject approval

Request (optional):
```json
{ "reason": "operator denied" }
```

Fails the run; pending steps skipped.

#### `POST /v1/runs/{run_id}/cancel` — request cancellation

Sets `cancel_requested`; terminal immediately if pending/waiting, else executor cancels between steps.

### Knowledge & memory

#### `POST /v1/knowledge/ingest`

Request:
```json
{ "id": "optional", "title": "...", "content": "...", "source": "..." }
```

Response `201`: ingested `Document` (may include chunk metadata).

#### `POST /v1/memory`

Request long-term `MemoryFact`:
```json
{
  "workspace_id": "default",
  "user_id": "u1",
  "scope": "long",
  "key": "pref",
  "content": "User prefers concise answers"
}
```

Response `201`: stored fact.

## Run object (response shape)

| Field | Type | Notes |
|-------|------|-------|
| `run_id` | string | durable id |
| `goal` | string | user goal |
| `status` | string | see status table |
| `steps` | array | plan DAG nodes |
| `result` | string | final output when succeeded |
| `error` | string | failure / reject reason |
| `cancel_requested` | bool | cancel flag |
| `created_at` / `updated_at` | RFC3339 | UTC |

### Run statuses
`pending` · `running` · `succeeded` · `failed` · `cancelled` · `waiting_approval`

### Step kinds
`context` · `model` · `tool` · `finalize`

### Step statuses
`pending` · `running` · `succeeded` · `failed` · `skipped` · `cancelled` · `waiting_approval`

## Lifecycle events
Persisted via `RunStore.AppendEvent` (not a separate HTTP list endpoint today). Types are defined in [events.schema.json](./events.schema.json): `run_created`, `run_started`, `step_started`, `step_succeeded`, `step_retry`, `waiting_approval`, `approved`, `rejected`, `run_succeeded`, `run_failed`, `run_cancelled`.

## Failure modes
| Mode | HTTP / behavior |
|------|-----------------|
| Empty goal | 400 on create |
| Run not found | 404 on get |
| Not startable terminal status | 409 on start |
| No waiting step | 409 on approve |
| Unknown tool / schema fail | step fails → run `failed` after retries |
| Gateway unavailable | model step fails (retry if budget) |
| Approval denied | reject → run `failed` |
| Cancel | run `cancelled` |
| Plan stuck | run `failed` with message |

## Machine-readable contract
Canonical OpenAPI 3.0 for implemented routes: [openapi.yaml](./openapi.yaml).
