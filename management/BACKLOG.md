# Backlog

## Purpose
Prioritize upcoming engineering work after M0 foundation.

## Scope
Near-term backlog items. Not a full issue tracker export.

## Dependencies
- [Milestones](./MILESTONES.md)
- [Tasks](../tasks/README.md)

## Owner
Atlas Core Engineering

## Status
Accepted

## Last Updated
2026-08-03

## Related Documents
- [Roadmap](./ROADMAP.md)
- [Changelog](./CHANGELOG.md)

## Now (post-M0)
1. Author `contracts/chat-provider.md` and embedding-provider contracts in detail.
2. Add JSON schemas for conversation, workspace, run under `schemas/`.
3. Scaffold `services/inference-gateway/` documentation pack (mirror orchestrator).
4. Write task `0002`+ for inference gateway implementation slices.
5. Draft `deploy/docker-compose.yml` for Postgres, Redis, Qdrant, MinIO (no app code required).

## Next
- Context builder algorithm ADR (token budgeting strategy).
- Tool SDK interface RFC.
- Prompt registry design.
- Eval dataset fixture format.

## Later
- Kubernetes manifests.
- Multi-region considerations.
- LLM-as-judge optional pipeline.
- Real GitHub/Slack adapters beyond mocks.

## Icebox
- Autonomous multi-agent swarms.
- Self-modifying agents.
- Exactly-once global messaging.
