# ADR 0010: Orchestration Run and Step State Machine

## Purpose
Record the status boundaries for agent-orchestrator runs and steps so API, executor, persistence, and eval share one contract.

## Scope
M2/M3 agent-orchestrator plan executor (`services/agent-orchestrator`). Does not define inference-gateway run statuses or multi-workflow product definitions beyond the current `Run` plan DAG.

## Dependencies
- [Architecture: Planner](../architecture/planner.md)
- [Architecture: Tool Runtime](../architecture/tool-runtime.md)
- [Agent Orchestrator model](../../services/agent-orchestrator/model.go)
- [Agent Orchestrator executor](../../services/agent-orchestrator/executor.go)
- [ADR 0003 Architecture Style](./0003-architecture-style.md)

## Owner
Atlas Architecture

## Status
Accepted

## Last Updated
2026-08-05

## Related Documents
- [ADR Index](./README.md)
- [Service API](../../services/agent-orchestrator/API.md)
- [Service DECISIONS](../../services/agent-orchestrator/DECISIONS.md)
- [Events schema](../../services/agent-orchestrator/events.schema.json)

## Context
The orchestrator executes a dynamic plan graph (context → tools/model → finalize) with retries, human approval for dangerous tools, and cooperative cancellation. Early draft docs used abstract `execution.*` event names and incomplete status sets. Implementation constants in `model.go` and transitions in `executor.go` / `service.go` are the source of truth.

## Problem
What statuses may a run and its steps take, which transitions are legal, and where do approval and cancel sit relative to the executor loop?

## Options
1. **Free-form status strings** — callers invent labels; no shared machine.
2. **Workflow engine product** (Temporal/Cadence-style external SM) for every step.
3. **Explicit fixed enums + executor-owned transitions** — small set of run/step statuses, enforced in Go constants and store writes.

## Decision
Adopt **option 3**.

### Run statuses
| Status | Meaning |
|--------|---------|
| `pending` | Created and planned; not yet started |
| `running` | Executor is advancing steps |
| `waiting_approval` | Paused on a tool that requires approval |
| `succeeded` | All required steps done; result available |
| `failed` | Unrecoverable step failure, reject, or stuck plan |
| `cancelled` | Operator cancel honored |

**Legal transitions (summary):**
- `pending` → `running` (start)
- `running` → `waiting_approval` | `succeeded` | `failed` | `cancelled`
- `waiting_approval` → `running` (approve) | `failed` (reject) | `cancelled` (cancel)
- Terminal: `succeeded`, `failed`, `cancelled` (no further start)

### Step statuses
| Status | Meaning |
|--------|---------|
| `pending` | Not yet run (or retry re-queue) |
| `running` | In flight |
| `succeeded` | Output recorded |
| `failed` | Attempts exhausted |
| `skipped` | Not run because run failed/rejected |
| `cancelled` | Cancel before/while step |
| `waiting_approval` | Dangerous tool gated |

**Step kinds:** `context`, `model`, `tool`, `finalize`.

### Approval boundary
- Tools with `RequiresApproval()` (or step flag) return `ErrWaitingApproval` without executing.
- Run and step move to `waiting_approval`; event type `waiting_approval`.
- `ApproveRun` sets `Approved` on waiting steps, emits `approved`, resumes executor.
- `RejectRun` fails waiting steps, skips remaining pending, status `failed`, event `rejected`.

### Cancel boundary
- `CancelRun` sets `cancel_requested`.
- If already `pending` or `waiting_approval`, run becomes `cancelled` immediately.
- If `running`, executor checks between steps and transitions to `cancelled` (event `run_cancelled`).

### Event types (lifecycle telemetry)
Emitted via `RunStore.AppendEvent` (see `events.schema.json`):
`run_created`, `run_started`, `step_started`, `step_succeeded`, `step_retry`, `waiting_approval`, `approved`, `rejected`, `run_succeeded`, `run_failed`, `run_cancelled`.

## Trade-offs
- **Benefit:** Simple, testable, matches in-process executor; works with memory and Postgres stores.
- **Cost:** Not a general BPMN engine; complex multi-party workflows need a later design.
- **Benefit:** Approval is a first-class pause, not a side-channel.

## Consequences
- OpenAPI and API.md must list these status enums only.
- Eval and CLI treat `waiting_approval` as non-terminal pause.
- New step kinds must map into the same step status set.
- Persistence stores opaque step JSON but must preserve status strings for resume.

## Alternatives Considered
External workflow engines rejected for M2/M3 complexity and operational cost. Free-form statuses rejected because CLI/eval cannot reason about pause/approve consistently.
