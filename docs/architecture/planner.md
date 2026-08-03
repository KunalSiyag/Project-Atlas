# Architecture: Planner and Workflow Engine

## Purpose
Describe how Atlas turns goals into structured plans and executes them as workflows.

## Scope
Planner agent behavior and workflow engine semantics. Tool and model details are separate subsystems.

## Dependencies
- [Architecture Overview](./OVERVIEW.md)
- [Inference](./inference.md)
- [Tool Runtime](./tool-runtime.md)
- [Domain: Workflow](../domain/workflow.md)
- [Domain: Agent](../domain/agent.md)

## Owner
Atlas Architecture

## Status
Accepted

## Last Updated
2026-08-03

## Related Documents
- [Agent Orchestrator](../../services/agent-orchestrator/README.md)
- [Security](./security.md)

## Why It Exists
Multi-step user goals (“review repo, plan sprint, email team”) require orchestration beyond a single completion call. Planning and workflow execution make agent work inspectable, parallelizable, and approvable.

## Problem Solved
- Opaque single-prompt agents that hide intermediate work
- No parallelization of independent reads
- No first-class human approval gates
- Difficulty retrying partial failures

## Consumers
- Gateway (start turn / start run)
- Context builder (supplies planner input)
- Tool runtime and inference (step execution)
- Evaluation and observability
- End users (progress and approvals)

## Public Interfaces (conceptual)
- `Plan(ctx, PlanRequest) → Plan` — structured tasks / DAG proposal
- `StartRun(ctx, Plan|Template) → Run`
- `GetRun(run_id) → RunView`
- `Signal(run_id, approval|cancel)` 
- Event stream: step started/finished/failed for UI and OTel

Planner output is data (task graph), not only prose.

## Dependencies
- Inference (planning model via router)
- Tool runtime
- Memory/context for plan input
- Durable store for run state (Postgres-oriented)
- Guardrails before side effects

## Failure Modes
| Failure | Handling |
|---------|----------|
| Planner returns invalid graph | Reject; repair prompt or fail run with error |
| Step tool timeout | Mark step failed; retry policy; optional compensate |
| Approval never arrives | Timeout to `cancelled` or `expired_approval` |
| Cycle in DAG | Validation error; do not execute |
| Model outage during plan | Retry/backoff; surface dependency error |
