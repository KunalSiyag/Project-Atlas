# Agent Orchestrator Data Model

## Purpose
Define domain entities and persistence ports for the shipped agent orchestrator.

## Scope
Live Go types and store interfaces under `services/agent-orchestrator`. Storage backends: in-memory (default) and PostgreSQL for runs/events; in-memory, Redis (short-term), and Postgres-backed long memory as configured.

## Dependencies
- [PRD](./PRD.md)
- [API](./API.md)
- [model.go](./model.go)
- [store.go](./store.go)
- [postgres_store.go](./postgres_store.go)
- [memory.go](./memory.go)
- [rag.go](./rag.go)

## Owner
Atlas Agent Runtime Team

## Status
Accepted

## Last Updated
2026-08-05

## Related Documents
- [Events schema](./events.schema.json)
- [ADR 0010 State Machine](../../docs/adr/0010-orchestration-state-machine.md)
- [Migration](./migrations/001_orchestrator_runs.sql)

## Entities

### Run
A durable orchestration execution record.

| Field | Type | Notes |
|-------|------|-------|
| `run_id` | string | primary key |
| `goal` | string | user goal |
| `status` | string | run status enum |
| `steps` | Step[] | plan DAG snapshot |
| `result` | string | final answer when succeeded |
| `error` | string | failure message |
| `cancel_requested` | bool | cooperative cancel |
| `created_at` / `updated_at` | time | UTC |

### Step
One node in the run plan DAG.

| Field | Type | Notes |
|-------|------|-------|
| `step_id` | string | unique within run |
| `name` | string | display/debug name |
| `kind` | string | `context` \| `model` \| `tool` \| `finalize` |
| `depends_on` | string[] | predecessor step_ids |
| `status` | string | step status enum |
| `attempts` / `max_attempts` | int | retry budget |
| `output` / `error` | string | result or failure |
| `prompt` | string | optional model override |
| `tool_name` / `tool_input` | string / object | tool steps |
| `requires_approval` / `approved` | bool | human gate |

### Event
Immutable lifecycle observation for a run (and optional step).

| Field | Type | Notes |
|-------|------|-------|
| `event_id` | string | primary key |
| `run_id` | string | parent run |
| `step_id` | string | optional |
| `type` | string | see events.schema.json |
| `message` | string | human-readable detail |
| `created_at` | time | UTC |

### MemoryFact
Scoped knowledge for context assembly.

| Field | Type | Notes |
|-------|------|-------|
| `id` | string | |
| `workspace_id` / `user_id` | string | tenancy scope |
| `scope` | string | `short` (per run) or `long` |
| `key` / `content` | string | |
| `run_id` | string | short-term association |
| `created_at` | time | |

### Document
RAG knowledge chunk / parent document (see `rag.go`).

## Status values

**Run:** `pending`, `running`, `succeeded`, `failed`, `cancelled`, `waiting_approval`

**Step:** `pending`, `running`, `succeeded`, `failed`, `skipped`, `cancelled`, `waiting_approval`

## Persistence ports (interfaces)

These ports match the shipped Go interfaces. Implementations: `InMemoryRunStore`, `PostgresStore`, `InMemoryMemory`, Redis short-term wrapper, `InProcessRetriever` / Qdrant-backed retriever.

### RunStore (`store.go`)

```text
RunStore
  SaveRun(ctx, run Run) error
  GetRun(ctx, runID string) (Run, error)
  AppendEvent(ctx, event Event) error
  ListEvents(ctx, runID string) ([]Event, error)
```

- **InMemoryRunStore:** process-local maps; deep-copies steps on save/get.
- **PostgresStore:** tables `orchestrator_runs` (steps as JSONB) and `orchestrator_events` (`migrations/001_orchestrator_runs.sql`). Selected when `ATLAS_ORCHESTRATOR_DATABASE_URL` is set.

### MemoryStore (`memory.go`)

```text
MemoryStore
  PutShort(ctx, runID, content string) error
  ListShort(ctx, runID string) ([]MemoryFact, error)
  PutLong(ctx, fact MemoryFact) error
  ListLong(ctx, workspaceID, userID string, limit int) ([]MemoryFact, error)
```

- Default: `InMemoryMemory`.
- Optional Redis for short-term (`ATLAS_REDIS_URL`); long facts may use Postgres-backed paths when configured.

### Retriever (`rag.go`)

```text
Retriever
  Ingest(ctx, doc Document) (Document, error)
  Search(ctx, query string, topK int) ([]Document, error)
```

- In-process vector index or Qdrant when `ATLAS_QDRANT_URL` is reachable.
- Embeddings via gateway embedder or fake for tests.

## Design notes
- Ports stay pure interfaces so the executor and HTTP handler do not import SQL or Redis clients directly.
- Event append is best-effort in several call sites (`_ = store.AppendEvent`); run durability is authoritative for control plane.
- No separate Workflow entity table yet: each create-run plans a fresh step graph from the goal.
