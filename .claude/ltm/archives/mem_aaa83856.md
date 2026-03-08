---
id: "mem_aaa83856"
topic: "kubectl context for kind cluster"
tags:
  - kubectl
  - kind
  - context
  - gotcha
phase: 0
difficulty: 0.3
created_at: "2026-02-17T10:05:17.027738+00:00"
created_session: 5
---
When running kubectl commands against the local kind cluster, always use `--context=kind-assethub` to avoid accidentally hitting an external cluster. The user's default kubectl context may point to a different cluster.

Example:
```bash
kubectl --context=kind-assethub -n assethub get pods
```

This applies to all kubectl commands in scripts, debugging, and system test verification.
