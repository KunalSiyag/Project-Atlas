# Runbook: Orchestrator failed / stuck runs

## Symptoms
- Runs in `failed`, `waiting_approval`, or stuck `running`
- Tool steps denied or timing out
- `GET /v1/runs/{id}/events` shows step_failed / cancel storms

## Immediate actions
1. Fetch run + events:
   ```bash
   curl -sS $ORCH_URL/v1/runs/$RUN_ID
   curl -sS $ORCH_URL/v1/runs/$RUN_ID/events
   ```
2. If `waiting_approval`, approve or reject with a reason; do not leave human gates open overnight.
3. Cancel runaway runs: `POST /v1/runs/$RUN_ID/cancel`.

## Diagnosis
- Tool authz: allowlist denials (`ToolErrDenied`) — check `AllowlistAuthorizer` config.
- Gateway dependency: model step failures usually mean gateway outage (see gateway-outage runbook).
- Postgres store: if `ATLAS_ORCHESTRATOR_DATABASE_URL` is set, verify connectivity and schema migrations applied.
- Prompt admin: workspace isolation errors on `/v1/prompts` when `X-Workspace-ID` is missing.

## Recovery
1. Fix the underlying dependency (gateway, DB, tool remote URL).
2. Re-create the run with the same goal once dependencies are healthy.
3. For knowledge regressions, check `/v1/knowledge/status` and re-ingest corpus docs.

## Escalation
- Persistent state corruption: restore Postgres from backup; multi-region HA is icebox.
