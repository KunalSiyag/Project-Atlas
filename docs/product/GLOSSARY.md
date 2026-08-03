# Glossary

## Purpose
Provide shared product vocabulary for Atlas so product, architecture, and implementation use the same terms.

## Scope
Product-facing and cross-cutting terms. Domain entities are defined in depth under `docs/domain/`; this glossary summarizes and points to them.

## Dependencies
- [Vision](./VISION.md)
- [Domain Language](../domain/README.md)

## Owner
Atlas Product Engineering

## Status
Accepted

## Last Updated
2026-08-03

## Related Documents
- [Functional Requirements](./FUNCTIONAL_REQUIREMENTS.md)
- [Architecture Overview](../architecture/OVERVIEW.md)

## Terms

### Agent
A configured reasoning actor that plans, calls tools, uses memory, and produces outputs within workspace policy. See [domain/agent](../domain/agent.md).

### Context building
The process of selecting and assembling history, memories, documents, tool results, preferences, and system instructions into a bounded model context. Distinct from raw prompt concatenation.

### Conversation
A durable thread of user and agent turns within a workspace. See [domain/conversation](../domain/conversation.md).

### Evaluation
Measurement of run quality and operations signals (latency, cost, tool success, grounding, optional human or LLM judges). See [domain/evaluation](../domain/evaluation.md).

### Guardrails
Controls that constrain inputs, tool execution, and outputs (permissions, injection defense, secret scanning, policy).

### Memory
Stored facts or state used to personalize or continue work beyond a single turn; scoped to user, workspace, or conversation. See [domain/memory](../domain/memory.md).

### Model router
Logic that chooses which model/provider handles a given task class (plan, code, vision, embed).

### Planner
Component that turns a user goal into structured tasks or a workflow rather than only a free-text answer.

### Prompt
Versioned instruction material (system, developer, tool schemas, few-shots) managed as artifacts. See [domain/prompt](../domain/prompt.md).

### Provider
An external or local backend for inference, embeddings, vector search, storage, or tools, accessed via interfaces. See [domain/provider](../domain/provider.md).

### Run / execution
One attempt to process a user request or workflow from acceptance through completion or failure, with logs and evaluation hooks.

### Tool
A capability with schema and execution semantics (GitHub, Slack, SQL, browser, etc.). See [domain/tool](../domain/tool.md).

### Tool runtime
The subsystem that registers, validates, authorizes, executes, and observes tools.

### Workflow
A structured execution graph (sequence or DAG) of steps including tool calls and human approvals. See [domain/workflow](../domain/workflow.md).

### Workspace
Isolation boundary for users, permissions, data, agents, and configuration. See [domain/workspace](../domain/workspace.md).

### Vendor independence
Architectural property that domain logic depends on ports/interfaces, not concrete vendor SDKs.

### ADR
Architecture Decision Record: durable decision with context, options, and consequences.

### M0 (Foundation)
Documentation-first foundation milestone: constitution, domain, architecture, foundation ADRs, navigation—without requiring application service binaries.
