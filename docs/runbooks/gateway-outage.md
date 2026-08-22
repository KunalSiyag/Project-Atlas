# Runbook: Inference gateway outage

## Symptoms
- `GET /healthz` or `/readyz` on the gateway fails or returns non-200
- Clients see timeouts or 5xx on `POST /v1/generate` / embeddings
- Orchestrator runs fail at model steps with gateway connection errors

## Immediate actions
1. Confirm blast radius: `curl -sS $GATEWAY_URL/healthz` and `/readyz`.
2. Check process/container logs for panic, OOM, or provider auth failures.
3. If using Compose: `docker compose ps` and restart only `inference-gateway` if the data plane is healthy.
4. Fail open for non-critical traffic: point local clients at the fake provider (`ATLAS_INFERENCE_PROVIDER=fake`) while restoring the primary provider.

## Diagnosis
- Provider path: Ollama/OpenAI-style adapter errors vs local fake.
- Database: gateway optional Postgres (`ATLAS_DATABASE_URL`) — a down DB should not block pure generate if designed as optional; if it does, open a defect.
- Resource limits: CPU throttle or memory limits in Kubernetes.

## Recovery
1. Restore provider credentials from secret store (never paste into tickets).
2. Roll forward to last known good image digests.
3. Re-run `make smoke` or `make cli-health` / `cli-generate` against the restored endpoint.
4. Confirm orchestrator model steps succeed on a canary run.

## Escalation
- If multi-region failover is required, that is **icebox 0006-f** (not shipped). Page platform and fail traffic to a single healthy region manually.
