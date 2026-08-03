# Domain: Agent

## Purpose
Define what an agent is in Atlas and how it relates to planning, tools, and providers.

## Scope
Domain meaning of agent configuration and behavior. Runtime orchestration details live under architecture (planner, workflow, tool runtime).

## Dependencies
- [Workspace](./workspace.md)
- [Prompt](./prompt.md)
- [Tool](./tool.md)
- [Provider](./provider.md)
- [Workflow](./workflow.md)

## Owner
Atlas Domain Engineering

## Status
Accepted

## Last Updated
2026-08-03

## Related Documents
- [Planner Architecture](../architecture/planner.md)
- [Agent Orchestrator](../../services/agent-orchestrator/README.md)

## Definition
An **agent** is a workspace-scoped configuration of goals, prompts, allowed tools, memory policies, model routing preferences, and guardrail settings that the platform uses to plan and act on user requests.

An agent is not “the LLM.” The LLM (via a **provider**) is one dependency the agent uses.

## Boundaries
- In scope: identity, configuration, permissions envelope, default models, tool allowlists.
- Out of scope: raw token streams; those belong to inference adapters.

## Core Attributes (conceptual)
- `agent_id`, name, description, version
- System and developer **prompt** references (versioned)
- Allowed **tools** and approval requirements
- Memory read/write policy
- Model routing profile (which provider/model for plan vs act vs embed)
- Temperature/tool-choice style policies (implementation-specific)

## Relationships
- Belongs to a **workspace**.
- Uses **prompts**, **memory**, **tools**, and **providers**.
- Produces **workflows** / **runs** when handling **conversations**.
- Subject to **evaluation** on its runs.

## Behaviors
- Plan before acting on multi-step goals when configured.
- Call tools only through the tool runtime under guardrails.
- Request human approval for designated side effects.
- Emit observability events for each major stage.

## Invariants
- An agent cannot use a tool not in its allowlist (or workspace allowlist, whichever is stricter).
- Agent configuration changes are versioned or audited enough to explain past runs.

## Failure Modes
- Misconfigured allowlists leading to silent tool denials.
- Prompt/provider mismatch (e.g. vision task with text-only model).
- Runaway cost if budgets and rate limits are unset.
