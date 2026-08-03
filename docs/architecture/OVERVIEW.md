# Architecture Overview

## Purpose
Describe the system architecture direction for Project Atlas and index subsystem documents.

## Scope
Repository-level architecture and service boundaries before implementation. Subsystem details live in linked docs.

## Dependencies
- [Engineering Principles](../engineering/PRINCIPLES.md)
- [Product Vision](../product/VISION.md)
- [ADR 0001](../adr/0001-documentation-first-modular-monorepo.md)
- [ADR 0003: Architecture Style](../adr/0003-architecture-style.md)

## Owner
Atlas Architecture

## Status
Accepted

## Last Updated
2026-08-03

## Related Documents
- [Domain Language](../domain/README.md)
- [Functional Requirements](../product/FUNCTIONAL_REQUIREMENTS.md)
- [Agent Orchestrator README](../../services/agent-orchestrator/README.md)

## Platform Capabilities
Atlas evolves around:
- Agent orchestration and workflow execution
- Tool and provider abstraction
- Context building and memory
- Evaluation and regression testing
- Observability for agent behavior
- Policy, safety, and governance
- Deployment and runtime operations

## High-Level Request Path
```text
User → Authentication → API Gateway → Conversation API
         → Context Builder + Memory Engine
         → Planner Agent → Workflow Engine
         → Tool Runtime | Knowledge Base | Model Router
         → Response Generator → Evaluation → Observability
```

The LLM appears only inside inference/model routing—not as the whole application.

## Architectural Style
Modular services with hexagonal (ports and adapters) boundaries:
- Domain logic owns use cases and business rules
- Ports define required capabilities
- Adapters integrate external systems
- API contracts are defined before service implementation

See [ADR 0003](../adr/0003-architecture-style.md).

## Vendor Independence
External inference, embeddings, vector stores, object storage, identity, message brokers, and tools are accessed through interfaces. See [ADR 0002](../adr/0002-provider-independence.md).

## Subsystem Index
| Subsystem | Document |
|-----------|----------|
| Gateway / API surface | [gateway.md](./gateway.md) |
| Planner / workflow orchestration | [planner.md](./planner.md) |
| Memory | [memory.md](./memory.md) |
| Context builder | [context-builder.md](./context-builder.md) |
| Inference / model routing | [inference.md](./inference.md) |
| Tool runtime | [tool-runtime.md](./tool-runtime.md) |
| Storage | [storage.md](./storage.md) |
| Security / guardrails | [security.md](./security.md) |
| Observability | [observability.md](./observability.md) |
| Evaluation | [evaluation.md](./evaluation.md) |
| Deployment | [deployment.md](./deployment.md) |

## Diagrams
Repository root may include high-level images (`HLD.png`, `System-design.png`) as visual companions to this overview.

## What We Intentionally Defer
Autonomous coding agents, self-improving loops, and large multi-agent swarms are non-goals until fundamentals are solid. See [NON_GOALS](../product/NON_GOALS.md).
