# Domain Language

## Purpose
Define the ubiquitous language for Project Atlas so product, architecture, and code share one vocabulary.

## Scope
Core domain concepts only. Product framing lives under `docs/product/`; subsystem design under `docs/architecture/`.

## Dependencies
- [Product Glossary](../product/GLOSSARY.md)
- [Documentation Standards](../engineering/DOCUMENTATION_STANDARDS.md)

## Owner
Atlas Domain Engineering

## Status
Accepted

## Last Updated
2026-08-03

## Related Documents
- [Architecture Overview](../architecture/OVERVIEW.md)
- [Functional Requirements](../product/FUNCTIONAL_REQUIREMENTS.md)

## Concepts
| Concept | Document |
|---------|----------|
| Workspace | [workspace.md](./workspace.md) |
| Agent | [agent.md](./agent.md) |
| Conversation | [conversation.md](./conversation.md) |
| Memory | [memory.md](./memory.md) |
| Tool | [tool.md](./tool.md) |
| Provider | [provider.md](./provider.md) |
| Evaluation | [evaluation.md](./evaluation.md) |
| Prompt | [prompt.md](./prompt.md) |
| Workflow | [workflow.md](./workflow.md) |

## Relationship Sketch
```text
Workspace
  owns Users, Permissions, Agents, Conversations, Memories, Knowledge
Conversation
  contains Turns
  may start Workflows / Runs
Agent
  uses Prompts, Memory, Tools, Providers
  driven by Planner → Workflow
Tool
  executed by Tool Runtime under Guardrails
Provider
  supplies Inference / Embeddings / Vector / Object / optional Messaging
Evaluation
  scores Runs using deterministic and optional judge metrics
```
