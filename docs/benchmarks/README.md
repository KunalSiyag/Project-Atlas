# Benchmarks

## Purpose
Index benchmark plans and results for Atlas.

## Scope
Applies to performance, reliability, evaluation, and cost benchmarks.

## Dependencies
- [Engineering Principles](../engineering/PRINCIPLES.md)

## Owner
Atlas Evaluation Team

## Status
Bootstrap

## Last Updated
2026-08-03

## Related Documents
- [Product Vision](../product/VISION.md)

## Index
No benchmarks have been authored yet.

## Eval fixtures

Deterministic offline suites live under [`fixtures/eval/v1/`](../../fixtures/eval/v1/FORMAT.md). Run with:

```bash
cd tools/atlas-eval && go run . -dataset ../../fixtures/eval/v1/sample.json -out report.json
```

