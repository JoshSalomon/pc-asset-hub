---
id: "mem_0f96fe43"
topic: "Kind deployment: pods don't pick up new images with rollout restart"
tags:
  - deployment
  - kind
  - kubernetes
  - debugging
phase: 0
difficulty: 0.6
created_at: "2026-02-19T20:10:02.411145+00:00"
created_session: 10
---
## Problem

When deploying to kind with `imagePullPolicy: Never` and `latest` tag, `kubectl rollout restart` does NOT guarantee pods use the new image. The deployment revision changes but since the image tag is the same (`latest`), the pod spec hash doesn't change and existing pods may not be recreated.

## Symptom

After `kind load docker-image` + `rollout restart`, the UI/API still serves old code. Checking `curl` for new strings in the JS bundle shows old content.

## Solution

After loading new images into kind, **delete the pods** to force recreation:

```bash
kubectl --context kind-assethub -n assethub delete pod -l app=assethub-ui
kubectl --context kind-assethub -n assethub delete pod -l app=assethub-api
kubectl --context kind-assethub -n assethub rollout status deployment/assethub-ui deployment/assethub-api --timeout=60s
```

Verify the new bundle is served:
```bash
curl -s http://localhost:30000 | grep -oE 'assets/[a-zA-Z0-9._-]+\.js'
```

## Root Cause

- `imagePullPolicy: Never` means kubelet never pulls — it uses whatever image is in the node's container runtime
- `kind load` replaces the image in the node's runtime, but running containers aren't affected
- `rollout restart` adds an annotation to trigger a new rollout, but if the pod template hash doesn't change, pods may not be replaced
- Deleting pods forces the deployment controller to create new ones, which pull from the node's (now updated) runtime cache
