# Agent Orchestrator API

## Purpose
Define initial API contract concepts for the Agent Orchestrator Service.

## Scope
Covers draft request models, response models, events, and failure modes.

## Dependencies
- [PRD](./PRD.md)
- [Data Model](./DATA_MODEL.md)

## Owner
Atlas Agent Runtime Team

## Status
Draft

## Last Updated
2026-08-03

## Related Documents
- [Test Plan](./TEST_PLAN.md)

## Draft Commands
### Start Execution
Request fields:
- `workflow_id`
- `input`
- `metadata`
- `idempotency_key`

Response fields:
- `execution_id`
- `status`
- `created_at`

### Get Execution
Request fields:
- `execution_id`

Response fields:
- `execution_id`
- `workflow_id`
- `status`
- `steps`
- `events`

## Draft Events
- `execution.started`
- `execution.step.started`
- `execution.step.completed`
- `execution.step.failed`
- `execution.completed`
- `execution.failed`

## Failure Modes
- Invalid workflow definition.
- Duplicate idempotency key with conflicting request body.
- Provider timeout through an adapter.
- Tool invocation failure.
- Persistence failure.
- Execution cancellation.
