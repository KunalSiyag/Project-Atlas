# Domain: Evaluation

## Purpose
Define evaluation as a first-class product concern for every meaningful run.

## Scope
What is measured, how runs become experiments, and relationship to benchmarks. Not a full statistics methodology guide.

## Dependencies
- [Workflow](./workflow.md)
- [Conversation](./conversation.md)
- [Provider](./provider.md)

## Owner
Atlas Domain Engineering

## Status
Accepted

## Last Updated
2026-08-03

## Related Documents
- [Evaluation Architecture](../architecture/evaluation.md)
- [Success Metrics](../product/SUCCESS_METRICS.md)
- [Benchmarks](../benchmarks/README.md)

## Definition
**Evaluation** is the systematic capture and scoring of agent runs so the platform can compare quality, cost, latency, and reliability across prompts, models, and versions.

Every request can be treated as an experiment with recorded metrics—not only offline benchmark suites.

## Signal Classes
| Signal | Deterministic? | Notes |
|--------|----------------|-------|
| Latency | Yes | End-to-end and per stage |
| Cost / tokens | Yes (when reported) | Per provider call |
| Tool success rate | Yes | Success vs error/timeout |
| Retrieval precision proxies | Often | Offline sets preferred |
| Citation shape/correctness | Often | Schema and link checks |
| Grounding / hallucination scores | Optional judge | LLM-as-judge is optional |
| Human rating | Human | Sampled UX feedback |

## Boundaries
- In scope: metrics definitions, dataset references, run attachments, regression gates.
- Out of scope: training base models.

## Relationships
- Attaches to **runs** / **workflows** and **conversations**.
- Compares **providers** and **prompt** versions.
- Consumes observability data; may add dedicated eval stores.

## Invariants
- Prefer deterministic evaluators before paid judges.
- Eval datasets are versioned.
- A “pass” in CI should mean a real check on shipped behavior, not a hard-coded mock of the metric.

## Failure Modes
- Goodharting metrics (optimizing scores that stop correlating with user value).
- Non-reproducible judges without temperature/seed control.
- Missing baseline when swapping providers.
