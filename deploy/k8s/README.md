# Kubernetes manifests (Atlas)

Static base manifests for **inference-gateway** and **agent-orchestrator**. They are production-oriented (probes, resource requests, non-root securityContext, secret-backed env) but are not a full multi-region/HA mesh.

## Apply (local or cluster)

```bash
# From repository root (requires images built/pushed and secrets created)
kubectl apply -f deploy/k8s/inference-gateway.yaml
kubectl apply -f deploy/k8s/agent-orchestrator.yaml
```

Create secrets before production traffic:

```bash
kubectl create secret generic atlas-gateway-secrets \
  --from-literal=database-url='postgres://...'
kubectl create secret generic atlas-orchestrator-secrets \
  --from-literal=database-url='postgres://...'
```

Optional: mount file-backed prompts as a ConfigMap named `atlas-prompts`.

## Images

Build service images from each Dockerfile:

```bash
docker build -t atlas/inference-gateway:latest services/inference-gateway
docker build -t atlas/agent-orchestrator:latest services/agent-orchestrator
```

## Production hardening

See [deploy/README.md](../README.md) § Production hardening.
