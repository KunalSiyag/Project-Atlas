# Runbook: Secret and credential rotation

## Symptoms
- Planned rotation of DB passwords, provider API keys, MinIO, Redis, or Slack/GitHub tokens
- Suspected credential leak (treat as emergency)

## Immediate actions (emergency leak)
1. Revoke the leaked credential at the provider first.
2. Issue a new secret and update Kubernetes Secrets / Compose env **without** committing secrets to git.
3. Restart gateway and orchestrator pods/processes so env reloads.
4. Audit recent runs and tool HTTP logs for abuse.

## Planned rotation
1. Create new credentials alongside old ones when dual-write is possible.
2. Update:
   - `atlas-gateway-secrets` / `atlas-orchestrator-secrets` (`database-url`)
   - Provider keys used by the inference gateway
   - Optional `ATLAS_GITHUB_API_URL` / tokens and Slack tokens for integration tools
3. Rolling restart Deployments so readiness probes only pass after the new secret is live.
4. Verify: `make smoke` or health + a canary generate/run.
5. Retire old credentials.

## Never
- Commit `.env` production secrets
- Paste secrets into issue trackers or chat
- Bake secrets into container images

## Related
- [Security audit](../ops/SECURITY_AUDIT.md)
- [Production hardening](../../deploy/README.md)
