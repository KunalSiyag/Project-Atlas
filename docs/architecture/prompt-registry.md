# Prompt Registry Design

## Purpose
Define how Atlas stores, versions, and resolves named prompts (system templates, plan prompts, tool preambles) across services.

## Scope
Design for M2–M4. Implementation may start as files on disk; this doc sets the contract before a database-backed registry.

## Dependencies
- [Domain: Prompt](../domain/prompt.md)
- [ADR 0009 Context Builder Token Budgeting](../adr/0009-context-builder-token-budgeting.md)
- [Architecture: Planner](./planner.md)

## Owner
Atlas Core Engineering

## Status
Accepted

## Last Updated
2026-08-05

## Related Documents
- [Evaluation architecture](./evaluation.md)
- [Context builder architecture](./context-builder.md)

## Context / Problem
Prompts are currently string literals inside the orchestrator (e.g. LLM planner system messages). Scattered literals make eval diffs hard, prevent A/B of prompt packs, and couple releases to code. Atlas needs a **prompt registry**: named, versioned templates with variable slots and metadata (owner, model class, approx tokens).

## Options
1. **Code-only constants** — status quo.
2. **File-backed registry** — YAML/JSON under `prompts/` loaded at process start.
3. **Database registry** — Postgres rows with draft/publish workflow.

## Decision
**Phase 1 (now–M3):** file-backed registry under `prompts/` (or `schemas/prompts/`) with the data model below.  
**Phase 2 (M4+):** optional Postgres publish path reusing the same logical IDs; files remain fixtures for eval.

## Data model

| Field | Description |
|-------|-------------|
| `id` | Stable name, e.g. `orchestrator.plan.system` |
| `version` | Semver or monotonic int |
| `text` | Template body with `{{var}}` placeholders |
| `variables` | Declared var names + descriptions |
| `model_class` | Optional hint: `plan`, `act`, `embed` |
| `approx_tokens` | Integer for context budgeting |
| `status` | `draft` \| `active` \| `retired` |

## Resolution algorithm
1. Client requests `(id, version?)`.
2. If version omitted, pick highest **active** version.
3. Render variables; missing required vars → error (do not silently empty).
4. Return rendered text + metadata for telemetry (`prompt_id`, `prompt_version`).

## Integration
- **Context builder:** system prompts resolved from registry before budget trim.
- **LLM planner:** plan system/user wrappers registered as `orchestrator.plan.*`.
- **Eval:** fixtures pin `prompt_id@version` so scores stay stable across code moves.

## Non-goals
- Prompt self-modification by agents.
- Multi-tenant prompt marketplaces.

## Open questions
- Hot reload vs process restart for file registry.
- Whether gateway should resolve prompts or only orchestrator.
