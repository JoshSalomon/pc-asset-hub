---
id: "mem_7ae0ad1c"
topic: "Kind cluster deployment — complete reference (merged)"
tags:
  - deployment
  - kind
  - kubectl
  - podman
  - crash-recovery
  - containers
phase: 0
difficulty: 0.7
created_at: "2026-03-30T12:23:10.604559+00:00"
created_session: 17
---
## Kind Cluster Deployment for pc-asset-hub

### Quick Reference
- Cluster: `assethub`, context: `kind-assethub`, namespace: `assethub`
- Engine: podman (`export KIND_EXPERIMENTAL_PROVIDER=podman`)
- API: `localhost:30080`, UI: `localhost:30000`
- Images: `assethub/{api-server,ui,operator}:latest`, `imagePullPolicy: Never`

### CRITICAL: Always use --context
```bash
# ALWAYS use --context (user runs multiple clusters in parallel)
kubectl --context=kind-assethub -n assethub get pods
# NEVER use kubectl config use-context
```

### Deploy / Rebuild
```bash
./scripts/kind-deploy.sh rebuild "kubectl --context kind-assethub"
```

### After Laptop Crash / Reboot
```bash
export KIND_EXPERIMENTAL_PROVIDER=podman
podman start assethub-control-plane
# Wait ~10 seconds for K8s API
kubectl --context kind-assethub -n assethub get pods
```
If API server is CrashLoopBackOff (started before Postgres):
```bash
kubectl --context kind-assethub -n assethub delete pod -l app=assethub-api
```

### Podman Image Loading Gotcha
`kind load docker-image` broken with podman. Workaround in `kind-deploy.sh`:
1. Retag with `docker.io/` prefix (strips podman's `localhost/` prefix)
2. `podman save` → `kind load image-archive`

### Pods Don't Pick Up New Images
`rollout restart` is NOT enough with `imagePullPolicy: Never` + `latest` tag. Must **delete pods** to force recreation:
```bash
kubectl --context kind-assethub -n assethub delete pod -l app=assethub-ui
kubectl --context kind-assethub -n assethub delete pod -l app=assethub-api
```

### Verify
```bash
kubectl --context kind-assethub -n assethub get pods
curl -sf http://localhost:30080/healthz
```
