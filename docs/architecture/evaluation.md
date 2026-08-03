# Architecture: Evaluation Layer

## Purpose
Describe how runs become measurable experiments online and offline.

## Scope
Evaluation service boundaries, deterministic vs judge evaluators, and benchmark integration.

## Dependencies
- [Architecture Overview](./OVERVIEW.md)
- [Domain: Evaluation](../domain/evaluation.md)
- [Observability](./observability.md)
- [Inference](./inference.md)

## Owner
Atlas Architecture

## Status
Accepted

## Last Updated
2026-08-03

## Related Documents
- [Success Metrics](../product/SUCCESS_METRICS.md)
- [Benchmarks](../benchmarks/README.md)
- [ADR 0007: Testing Philosophy](../adr/0007-testing-philosophy.md)

## Why It Exists
Production AI systems differ from demos by measuring quality, cost, and reliability continuously—not only by subjective chat.

## Problem Solved
- No regression signal when swapping models or prompts
- Cost and latency invisible at the product layer
- Over-reliance on LLM-as-judge before deterministic checks exist

## Consumers
- Release process (gates)
- Product/quality owners
- Model routing experiments
- CI (subset of offline evals)

## Public Interfaces (conceptual)
- `RecordRunMetrics(run_id, metrics)`
- `Evaluate(run_id, evaluator_set) → Report`
- `Compare(baseline_id, candidate_id) → Diff`
- Dataset registry: versioned cases with expected tool traces / citations
- Benchmark report documents under `docs/benchmarks/`

## Evaluator Priority
1. Deterministic: latency, tokens/cost, tool success, schema/citation checks, retrieval proxies on fixed sets  
2. Optional LLM judges (provider-independent interface)  
3. Human ratings  

## Dependencies
- Run store and observability exports
- Object storage for datasets/artifacts
- Inference (only for optional judges)
- Configuration for which evaluators are online vs offline

## Failure Modes
| Failure | Handling |
|---------|----------|
| Judge provider down | Skip optional judges; keep deterministic |
| Flaky non-deterministic eval in CI | Quarantine; do not gate on flake |
| Missing baseline | Refuse “improved” claims; require baseline id |
| Dataset drift | Version pin; changelog |
