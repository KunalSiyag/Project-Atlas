# Atlas Eval Dataset Fixture Format (v1)

## Purpose
Versioned offline fixtures for deterministic evaluation of planner, tools, and retrieval scorers.

## Scope
JSON datasets consumed by `tools/atlas-eval`. Not live LLM-as-judge.

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

5. **`llm_judge`** — optional LLM-as-judge (requires harness `Judge` / `-judge-stub`).
   - `input.rubric` (string), `input.candidate` (string).
   - `expect.pass` (bool, optional): required judge verdict.
   - `expect.min_score` (number, optional): minimum judge score.
   - If no judge is configured, the case **fails** with `status: "error"` and message `llm_judge requires a Judge` (no panic). Deterministic suites without this type still pass with no judge.
   - CLI: `atlas-eval -dataset ... -judge-stub pass` injects a static pass judge for offline runs.

