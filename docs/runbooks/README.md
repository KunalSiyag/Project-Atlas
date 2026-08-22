# Runbooks

## Purpose
Index operational response guides for Atlas services in production and local stacks.

## Scope
Applies to production operations documentation. Incident-management SaaS (PagerDuty) is out of scope.

## Owner
Atlas Operations

## Status
Accepted (library seeded)

## Last Updated
2026-08-05

## Related Documents
- [Playbooks](../playbooks/README.md)
- [Security audit](../ops/SECURITY_AUDIT.md)
- [Production hardening](../../deploy/README.md)

## Index

| Runbook | When to use |
|---------|-------------|
| [Gateway outage](./gateway-outage.md) | Inference gateway unhealthy, 5xx storms, or generate path down |
| [Orchestrator failed runs](./orchestrator-failed-runs.md) | Runs stuck failed / waiting_approval / cancel storms |
| [Secret and credential rotation](./secret-credential-rotation.md) | Rotating DB URLs, provider keys, workspace secrets |
