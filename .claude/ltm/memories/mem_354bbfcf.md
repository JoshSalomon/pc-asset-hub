---
id: "mem_354bbfcf"
topic: "No podman/docker in sandbox — deploy via SSH to host, use host.containers.internal for API/UI"
tags:
  - no podman
  - no docker
  - container engine not found
  - kind-deploy.sh
  - sandbox
  - host.containers.internal
  - localhost not working
  - ssh deploy
phase: 0
difficulty: 0.5
created_at: "2026-04-06T08:19:07.493759+00:00"
created_session: 17
---
## Problem
Claude Code runs in a container sandbox without podman or docker. `kind-deploy.sh rebuild` fails with "no working container engine found." But kubectl works — the cluster is accessible, just can't build images.

## Solution
Deploy via SSH to the host where podman is available:
```bash
ssh -i ~/.ssh/container-dev -o StrictHostKeyChecking=no jsalomon@host.containers.internal deploy
```

## Important: localhost doesn't work
From the sandbox, `localhost` does NOT reach the kind cluster ports. Always use `host.containers.internal`:
- API: `http://host.containers.internal:30080`
- UI: `http://host.containers.internal:30000`

Live test scripts need the host URL:
```bash
bash scripts/test-publishing.sh http://host.containers.internal:30080
```
