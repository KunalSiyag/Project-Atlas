# Agent Orchestrator TODO

## Purpose
Track documentation-first work required before and alongside implementation.

## Scope
Applies to Agent Orchestrator planning and implementation readiness.

## Dependencies
- [PRD](./PRD.md)
- [API](./API.md)
- [OpenAPI](./openapi.yaml)
- [Test Plan](./TEST_PLAN.md)

## Owner
Atlas Agent Runtime Team

## Status
Accepted

## Last Updated
2026-08-05

## Related Documents
- [Decisions](./DECISIONS.md)
- [ADR 0010](../../docs/adr/0010-orchestration-state-machine.md)
- [Task 0005](../../tasks/0005-agent-orchestrator-docs-and-contracts.md)
- [Task 0006](../../tasks/0006-agent-orchestrator-remaining.md)

## Tasks
- [x] Review and accept service PRD.
- [x] Convert API concepts into concrete OpenAPI or protobuf contracts.
- [x] Create ADR for orchestration state machine boundaries.
- [x] Define persistence port interfaces.
- [x] Define telemetry event schema.
- [x] Create implementation task breakdown after contracts are accepted.

## Deliverable map
| Task | Artifact |
|------|----------|
| PRD accepted | [PRD.md](./PRD.md) Status Accepted |
| OpenAPI | [openapi.yaml](./openapi.yaml), [API.md](./API.md) |
| State machine ADR | [docs/adr/0010-orchestration-state-machine.md](../../docs/adr/0010-orchestration-state-machine.md) |
| Persistence ports | [DATA_MODEL.md](./DATA_MODEL.md) (`RunStore`, `MemoryStore`, `Retriever`) |
| Telemetry events | [events.schema.json](./events.schema.json) |
| Task breakdown | [tasks/0005](../../tasks/0005-agent-orchestrator-docs-and-contracts.md) done; [tasks/0006](../../tasks/0006-agent-orchestrator-remaining.md) a–e Done, f–g Icebox |
