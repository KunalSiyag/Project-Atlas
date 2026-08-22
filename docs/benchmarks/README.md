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
Accepted (bench tool + load dashboard seeded)

## Last Updated
2026-08-05

## Related Documents
- [Product Vision](../product/VISION.md)
- [Grafana load/bench dashboard](../../deploy/grafana/dashboards/atlas-load-bench.json)

## Index
| Artifact | Role |
|----------|------|
| `make bench` / `tools/atlas-bench` | Runnable load against a real service path; JSON latency/throughput |
| `deploy/grafana/dashboards/atlas-load-bench.json` | Load/bench product dashboard panels |
| Eval fixtures | Deterministic quality gates (`make eval-gate`) |

## Load / bench

```bash
# Against a live gateway (or any URL implementing the path):
make bench
# or:
cd tools/atlas-bench && go run . -url http://127.0.0.1:8080 -path /healthz -n 40 -c 8 -out metrics.json
```

Metrics fields: `throughput_rps`, `latency_p50_ms`, `latency_p95_ms`, `latency_p99_ms`, `error_ratio`.

## Eval fixtures

Deterministic offline suites live under [`fixtures/eval/v1/`](../../fixtures/eval/v1/FORMAT.md). Run with:

```bash
cd tools/atlas-eval && go run . -dataset ../../fixtures/eval/v1/sample.json -out report.json
# LLM-as-judge product pipeline (fixture sample_with_judge.json):
cd tools/atlas-eval && go run . -dataset ../../fixtures/eval/v1/sample_with_judge.json -judge-stub pass -out judge-report.json
```

