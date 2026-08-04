# ADR 0009: Context Builder Token Budgeting Strategy

## Purpose
Record how Atlas assembles bounded model context from system prompts, history, memory, and retrieved documents.

## Scope
Agent-orchestrator context assembly for M1–M3. Does not cover prompt registry storage formats (see prompt registry design) or provider-side context windows.

## Dependencies
- [Architecture: Context Builder](../architecture/context-builder.md)
- [ADR 0005 Storage Strategy](./0005-storage-strategy.md)
- [Agent Orchestrator](../../services/agent-orchestrator/README.md)

## Owner
Atlas Architecture

## Status
Accepted

## Last Updated
2026-08-05

## Related Documents
- [ADR Index](./README.md)
- [Prompt Registry Design](../architecture/prompt-registry.md)

## Context
Agent runs combine system instructions, conversation history, short/long memory, and RAG snippets. Model context windows are finite. Early Atlas used a simple char/4 token estimator and priority drops; this ADR freezes the budgeting algorithm so orchestrator, eval, and future UIs share one contract.

## Problem
How should Atlas rank and trim context material under a token budget without unbounded prompt growth or silent loss of critical instructions?

## Options
1. **Provider-native tokenizers only** — call each model’s tokenizer API before every generate.
2. **Fixed sliding window of last N messages** — ignore memory/RAG priority distinctions.
3. **Priority layers + cheap estimator** — estimate tokens as `ceil(utf8_runes/4)`, always keep system if it fits, then newest history, then memory, then docs; drop lowest priority first.

## Decision
Adopt **option 3** for M1–M3:

1. **Estimate** tokens as `max(1, ceil(utf8_runes/4))` for non-empty strings (documented approximation, not BPE).
2. **Priority order (high → low):** system prompt → recent history (newest-first selection, chronological emit) → memory notes → document snippets.
3. **Hard fail** if the system prompt alone exceeds the budget (caller must raise budget or shorten system text).
4. **Soft drop** lower-priority material with a `dropped` count and inclusion summary for debugging.
5. **Budget source:** request `MaxTokens`, else builder default (orchestrator config / env), never unlimited.

Implementation lives in `services/agent-orchestrator` `ContextBuilder.Build`. Future tokenizer adapters may refine estimates without changing priority order.

## Trade-offs
- **Benefit:** Deterministic, testable, provider-independent, fast enough for every step.
- **Cost:** Token estimates diverge from true BPE counts; may under/over fill real windows. Mitigate with conservative budgets and later optional true tokenizers.

## Consequences
- Eval and unit tests must assert priority and drop behavior, not exact vendor token counts.
- Prompt registry entries should declare approximate sizes so packs fit budgets.
- Streaming generate does not change budgeting; budget applies to the assembled request.

## Alternatives Considered
Provider-only tokenizers rejected for offline/fake-provider and multi-vendor cost. Pure last-N window rejected because memory and RAG would be starved unpredictably.
