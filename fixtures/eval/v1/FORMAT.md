# Atlas Eval Dataset Fixture Format (v1)

## Purpose
Versioned offline fixtures for deterministic evaluation of planner, tools, and retrieval scorers.

Shipped suites under this directory:
- `sample.json` — smoke suite (planner + RAG + tool_success + string)
- `extended.json` — expanded suite (more tool_success, rule_plan, plan_json cases)
- `sample_with_judge.json` — optional LLM-as-judge cases

## Scope
JSON datasets consumed by `tools/atlas-eval`. LLM-as-judge is a first-class product pipeline path (`llm_judge` cases) via the shipped `orchestrator.Judge` interface (static stub, `GatewayJudge`, or custom).

## Schema
Top-level object:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `version` | string | yes | Fixture format version; currently `"1"`. |
| `name` | string | yes | Human-readable suite name. |
| `description` | string | no | Suite intent. |
| `cases` | array | yes | Ordered list of cases. |

Each **case**:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | yes | Stable case id. |
| `type` | string | yes | Scorer type (see below). |
| `description` | string | no | What is being checked. |
| `input` | object | yes | Type-specific input. |
| `expect` | object | yes | Type-specific expectations. |

### Case types

1. **`rule_plan`** — run `RulePlanner.Plan` on `input.goal`.  
   - `expect.has_tool` (string, optional): tool name must appear.  
   - `expect.min_steps` (int, optional): minimum step count.  
   - `expect.kinds` (string[], optional): required step kinds.

2. **`plan_json`** — parse `input.plan_json` via `ParsePlanJSON` + `ValidatePlan`.  
   - `expect.valid` (bool): whether validation should succeed.

3. **`rag_rank`** — ingest `input.docs[]` then `Search(input.query, top_k)`.  
   - Doc: `{id, title, content, source}`.  
   - `expect.top_id` (string): first hit id must match.

4. **`string_contains`** — check `input.text` contains all of `expect.contains[]`.

5. **`tool_success`** — invoke a built-in tool **by name** through the shipped `ToolRegistry.Invoke` path (RFC 0001). Does not re-implement tools.
   - `input.tool` (string, required): registered tool name (e.g. `calculator`, `echo`).
   - `input.input` (object): tool input map (schema-validated by the registry).
   - `input.approve` (bool, optional): set true for tools with `RequiresApproval`.
   - `expect.equals` (string, optional): exact output match.
   - `expect.contains` (string[], optional): substrings that must appear in output.
   - Case **fails** if the tool errors (schema, timeout, denied, tool_error) or output mismatches.

6. **`llm_judge`** — LLM-as-judge **product pipeline** (requires harness `Judge` / `-judge-stub` / gateway judge).
   - Drives the shipped `orchestrator.Judge.Judge` end-to-end (not only `ParseJudgeResponse`).
   - `input.rubric` (string), `input.candidate` (string).
   - `expect.pass` (bool, optional): required judge verdict.
   - `expect.min_score` (number, optional): minimum judge score.
   - If no judge is configured, the case **fails** with `status: "error"` and message `llm_judge requires a Judge` (no panic). Deterministic suites without this type still pass with no judge.
   - CLI offline: `atlas-eval -dataset fixtures/eval/v1/sample_with_judge.json -judge-stub pass -out report.json`
   - Gateway path: wire `orchestrator.GatewayJudge{Client: ..., Model: ...}` into `RunDatasetWithOptions` (see harness tests).

## Report output
Machine-readable JSON written by the harness:

```json
{
  "version": "1",
  "suite": "...",
  "passed": true,
  "total": 3,
  "passed_count": 3,
  "failed_count": 0,
  "cases": [
    {"id": "...", "type": "...", "pass": true, "score": 1.0, "message": ""}
  ]
}
```

`score` is `1.0` on pass and `0.0` on fail for deterministic gates.

