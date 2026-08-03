# Agent Orchestrator Data Model

## Purpose
Define draft domain entities for orchestration.

## Scope
Covers conceptual data models only; storage schemas will be designed later.

## Dependencies
- [PRD](./PRD.md)
- [API](./API.md)

## Owner
Atlas Agent Runtime Team

## Status
Draft

## Last Updated
2026-08-03

## Related Documents
- [Test Plan](./TEST_PLAN.md)

## Entities
### Workflow
A versioned definition of agent steps, tool permissions, and execution policy.

### Execution
A single run of a workflow with status, input, output, and timestamps.

### Step
A discrete unit of orchestration work within an execution.

### Execution Event
An immutable observation emitted during execution lifecycle changes.

## Status Values
- `pending`
- `running`
- `succeeded`
- `failed`
- `cancelled`
