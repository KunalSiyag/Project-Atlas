# Agent Orchestrator AI Context

## Purpose
Provide minimal context for AI-assisted work on the Agent Orchestrator Service.

## Scope
Applies only to files under `services/agent-orchestrator`.

## Dependencies
- [README](./README.md)
- [PRD](./PRD.md)
- [API](./API.md)
- [Data Model](./DATA_MODEL.md)

## Owner
Atlas Agent Runtime Team

## Status
Bootstrap

## Last Updated
2026-08-03

## Related Documents
- [Test Plan](./TEST_PLAN.md)
- [TODO](./TODO.md)

## Rules
- Do not call provider SDKs directly from orchestration domain logic.
- Define ports for inference, tools, persistence, messaging, and telemetry.
- Keep execution state transitions deterministic and testable.
- Emit structured execution events for observability.
