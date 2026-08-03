# Contract: Planner and Run Control

## Purpose
Define the control-plane contract for planning and workflow runs.

## Scope
Orchestrator-facing API concepts (may be internal HTTP).

## Dependencies
- [Contracts Index](./README.md)
- [Domain: Workflow](../docs/domain/workflow.md)
- [Architecture: Planner](../docs/architecture/planner.md)

## Owner
Atlas Architecture

## Status
Draft

## Last Updated
2026-08-03

## Related Documents
- [Agent Orchestrator API](../services/agent-orchestrator/API.md)

## Operations (conceptual)
| Operation | Intent |
|-----------|--------|
| `CreatePlan` | From conversation turn + context → structured plan/DAG |
| `StartRun` | Persist and execute plan |
| `GetRun` | Status, steps, errors |
| `CancelRun` | Cooperative cancel |
| `SubmitApproval` | Resume waiting_approval step |

## Plan artifact
- `steps[]` with ids, type (`model`|`tool`|`approval`|`parallel`), depends_on[], inputs
- Validation: DAG, no cycles, tools allowlisted

## Run artifact
- `run_id`, `status`, `steps[]` state machine, timestamps, budgets
