# Domain: Workflow

## Purpose
Define workflows as structured, inspectable executions of multi-step agent work.

## Scope
Workflow and step semantics including DAGs and human approval. Engine implementation details live under architecture.

## Dependencies
- [Agent](./agent.md)
- [Tool](./tool.md)
- [Conversation](./conversation.md)
- [Evaluation](./evaluation.md)

## Owner
Atlas Domain Engineering

## Status
Accepted

## Last Updated
2026-08-03

## Related Documents
- [Planner Architecture](../architecture/planner.md)
- [Agent Orchestrator PRD](../../services/agent-orchestrator/PRD.md)

## Definition
A **workflow** is a directed structure of **steps** that achieves a user goal. Steps may run sequentially or in parallel (DAG), call tools, wait for human approval, or invoke models.

A **run** is one execution instance of a workflow (or of a simpler single-shot path recorded in the same lifecycle model).

## Boundaries
- In scope: graph structure, step states, inputs/outputs, retries, cancellation, approval gates.
- Out of scope: arbitrary Turing-complete user scripting as a day-one requirement.

## Step States (conceptual)
`pending` → `running` → `succeeded` | `failed` | `cancelled` | `waiting_approval` → (resume) …

## Core Attributes (conceptual)
- `workflow_id` / `run_id`, parent **conversation** and **agent**
- Graph definition (nodes, edges) or linear plan
- Per-step: type (model, tool, approval, fan-out join), I/O, error, attempts
- Timestamps, budget consumed

## Relationships
- Created by **planner** output or predefined templates.
- Invokes **tools** and **providers**.
- Visible in **conversation** UI as progress.
- Scored by **evaluation**; traced by observability.

## Example Shape
```text
          Root
           │
   ┌───────┴───────┐
Read GitHub     Read Docs
   └───────┬───────┘
        Summarize
           │
   Create Sprint Plan
           │
    Human Approval
           │
       Send Email
```

## Invariants
- Independent steps may run in parallel only when the graph allows (no hidden data race on shared unscoped state).
- Side-effecting steps record enough for audit and compensation/retry.
- Approval gates cannot be skipped without authorization.

## Failure Modes
- Deadlocks in poorly formed graphs (engine must reject cycles if DAG-only).
- Retrying non-idempotent hard writes without keys.
- Lost approvals if notification channel fails.
