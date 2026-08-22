# Enterprise security audit (in-repo)

## Purpose
Engineering self-audit of the Atlas security surface as of 2026-08-05. This is **not** a third-party pen-test, SOC2 report, or formal certification.

## Scope
Authn/authz, secrets, tools, network exposure, multi-tenant isolation for the shipped gateway + orchestrator slice.

## Status
Accepted (living document)

## Findings

| ID | Area | Severity | Finding | Status / mitigation |
|----|------|----------|---------|---------------------|
| SA-01 | Authn | **High** | No end-user OIDC/JWT product auth; services trust network + optional workspace header | Documented; production must put gateway/orchestrator behind authenticated ingress. Not claimed Done as enterprise SSO. |
| SA-02 | Authz / multi-tenant | **Medium** | Prompt admin requires `X-Workspace-ID` and store keys by workspace; cross-workspace list/get isolated. Run APIs are not fully tenant-partitioned in the run store. | Prompt path mitigated. Run-level tenant isolation remains residual hardening. |
| SA-03 | Secrets | **High** | Compose `.env.example` uses weak local defaults; K8s manifests use secretKeyRef (optional) | Production hardening docs require secret manager; never use example passwords outside laptop. |
| SA-04 | Tools | **Medium** | Remote HTTP tools and GitHub/Slack adapters can call arbitrary base URLs if misconfigured; allowlist authz available but not default-on for all deployments | Default registry has no allowlist; operators should set `AllowlistAuthorizer` and constrain remote bases. Dangerous tools require approval. |
| SA-05 | Tools | **Low** | WASM guest sandbox not present (icebox) — reduces guest escape surface but also means no hard isolation for untrusted plugins | HTTP plugins only; do not load untrusted WASM. |
| SA-06 | Network | **Medium** | Local Compose exposes Postgres/Redis/MinIO on host ports for DX | Production hardening: do not expose data plane publicly; use NetworkPolicies. |
| SA-07 | Knowledge | **Low** | Corpus list/delete/search has no per-workspace ACL yet | Documented residual; treat knowledge as single-tenant until ACL lands. |
| SA-08 | Eval / data | **Low** | LLM-as-judge may send candidate text to a model provider | Use stub judge offline; redacted fixtures for CI. |

## Severity legend
- **Critical** — active exploit path in default config (none open in this audit after documented controls)
- **High** — production block without compensating control
- **Medium** — must address before multi-tenant production
- **Low** — defense in depth / residual

## Recommendations (next)
1. Mandatory authenticated ingress + mTLS between gateway and orchestrator.
2. Default-on tool allowlists per workspace.
3. Run store workspace partition matching prompt isolation.
4. External pen-test before enterprise launch (out of scope for this document).
