---
id: "mem_4cc0673f"
topic: "Kind cluster deployment and crash recovery for pc-asset-hub"
tags:
  - deployment
  - kind
  - kubectl
  - kubernetes
  - crash-recovery
  - podman
phase: 0
difficulty: 0.4
created_at: "2026-03-01T17:35:40.030753+00:00"
created_session: 12
---
# Kind Cluster Deployment

## Quick rebuild
```bash
./scripts/kind-deploy.sh rebuild "kubectl --context kind-assethub"
```

## Key facts
- Cluster name: `assethub`, context: `kind-assethub`, namespace: `assethub`
- Container engine: podman (`KIND_EXPERIMENTAL_PROVIDER=podman`)
- API: `localhost:30080`, UI: `localhost:30000`
- Images: `assethub/{api-server,ui,operator}:latest`, `imagePullPolicy: Never`

## After laptop crash / reboot
```bash
export KIND_EXPERIMENTAL_PROVIDER=podman
podman start assethub-control-plane
# Wait ~10 seconds for K8s API server
kubectl --context kind-assethub get nodes
kubectl --context kind-assethub -n assethub get pods
```
If API server is in CrashLoopBackOff (started before Postgres was ready):
```bash
kubectl --context kind-assethub -n assethub delete pod -l app=assethub-api
kubectl --context kind-assethub -n assethub rollout status deployment/assethub-api --timeout=60s
```

## Verify
```bash
kubectl --context kind-assethub -n assethub get pods
curl -sf http://localhost:30080/healthz
```
