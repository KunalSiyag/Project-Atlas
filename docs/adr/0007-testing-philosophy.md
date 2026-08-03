# ADR 0007: Testing Philosophy

## Purpose
Record how Atlas tests platform behavior without theatrical or vendor-locked tests.

## Scope
Unit, contract, integration, and evaluation testing principles across the monorepo.

## Dependencies
- [Engineering Principles](../engineering/PRINCIPLES.md)
- [Domain: Evaluation](../domain/evaluation.md)
- [Architecture: Evaluation](../architecture/evaluation.md)

## Owner
Atlas Architecture

## Status
Accepted

## Last Updated
2026-08-03

## Related Documents
- [Success Metrics](../product/SUCCESS_METRICS.md)
- [Agent Orchestrator Test Plan](../../services/agent-orchestrator/TEST_PLAN.md)

## Context
AI systems are nondeterministic at the model boundary. Tests that hard-code model prose or reimplement the system under test give false confidence. Atlas still needs rigorous tests for orchestration, authz, tools, and contracts.

## Problem
What testing strategy balances confidence with flaky model outputs and free/local development?

## Options
1. **Only manual chat testing.**
2. **Only LLM-as-judge in CI.**
3. **Layered tests:** pure domain and contract tests with fakes; integration with local doubles; optional recorded/golden provider tests; deterministic evaluators first; models out of default unit CI.

## Decision
Adopt a **layered testing philosophy**:
1. **Unit tests** for domain logic and pure functions with fake ports (no network).
2. **Contract tests** validating API/schema shapes and adapter interface compliance.
3. **Integration tests** against Compose dependencies (Postgres, Redis, etc.) when services exist.
4. **Tool tests** with mock tool backends (Mailpit, fake GitHub, etc.).
5. **Evaluation suites** versioned under benchmarks; deterministic metrics gate CI; LLM judges optional and non-blocking unless stabilized.
6. **No test theater:** tests must exercise shipped entry points; no hard-coded “expected essay” as the only assert for orchestration correctness.

## Trade-offs
- **Benefit:** Reliable CI; real confidence in platform code; provider swap safety.
- **Cost:** More fakes and fixtures to maintain; model quality tracked separately from unit green builds.

## Consequences
- Service `TEST_PLAN.md` files list acceptance tests before implementation.
- Provider adapters have conformance tests against the port.
- Flaky network model calls are not required for merge of pure platform changes.

## Alternatives Considered
Manual-only rejected. Judge-only CI rejected as expensive and flaky for foundation work.
