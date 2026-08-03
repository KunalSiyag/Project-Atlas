# Architecture: Context Builder

## Purpose
Describe how Atlas assembles model context under budgets—context engineering, not only prompt pasting.

## Scope
Selection, ranking, and packaging of context for inference. Prompt version storage is related but separate (domain prompt).

## Dependencies
- [Architecture Overview](./OVERVIEW.md)
- [Memory](./memory.md)
- [Inference](./inference.md)
- [Domain: Conversation](../domain/conversation.md)
- [Domain: Prompt](../domain/prompt.md)

## Owner
Atlas Architecture

## Status
Accepted

## Last Updated
2026-08-03

## Related Documents
- [Evaluation](./evaluation.md)
- [Tool Runtime](./tool-runtime.md)

## Why It Exists
Sending “everything” wastes tokens, increases latency/cost, and can bury relevant facts. The context builder decides what enters the model window.

## Problem Solved
- Context overflow and naive truncation
- Missing provenance (what was included)
- Inconsistent assembly across services

## Consumers
- Planner and response generation paths
- Evaluation (for reproducibility: snapshot of included sources)
- Developers debugging bad answers

## Public Interfaces (conceptual)
- `BuildContext(ctx, ContextRequest) → AssembledContext`
  - Inputs: conversation history, agent prompts, memory hits, retrieved docs, tool results, user preferences, token budget
  - Outputs: ordered messages/sections, token estimate, inclusion manifest (source ids), exclusions summary
- Optional `ExplainContext(run_id)` for debug UIs

## Dependencies
- Conversation store
- Memory engine
- Knowledge retrieval (vector + object metadata)
- Prompt registry
- Tokenizer / budget estimator aligned with target model family
- Observability (span: `context.build`)

## Failure Modes
| Failure | Handling |
|---------|----------|
| Budget too small for system prompt | Fail fast with configuration error |
| Retriever timeout | Proceed with history+memory only; flag degraded |
| Conflicting memories | Prefer higher priority/recency rules; record conflict |
| Manifest omitted | Treat as bug; required for eval reproducibility in strict mode |
