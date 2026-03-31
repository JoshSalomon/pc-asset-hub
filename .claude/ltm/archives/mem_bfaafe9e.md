---
id: "mem_bfaafe9e"
topic: "Always use --context=kind-assethub with kubectl"
tags:
  - kubectl
  - context
  - kind
  - critical
phase: 0
difficulty: 0.9
created_at: "2026-02-18T09:46:53.043938+00:00"
created_session: 8
---
## CRITICAL: kubectl context requirement

**ALWAYS** use `kubectl --context=kind-assethub` for every kubectl command. Never use bare `kubectl` or `kubectl config use-context`.

The user works with multiple clusters in parallel (kind + OCP) and switching contexts with `use-context` breaks their other terminal sessions.

This applies to:
- All `kubectl` commands (get, apply, delete, logs, rollout, etc.)
- The `kind-deploy.sh` script (uses bare kubectl — may need manual context flags)

Example:
```bash
# CORRECT
kubectl --context=kind-assethub -n assethub get pods

# WRONG - never do this
kubectl config use-context kind-assethub
kubectl -n assethub get pods
```
