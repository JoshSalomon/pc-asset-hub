# Adversarial Review: `docs/architecture.md`

**Date:** 2026-06-03  
**Reviewed file:** `docs/architecture.md`  
**Focus:** New local changes vs upstream, plus any high/critical upstream issues

---

## Critical / High Findings (New Changes)

### 1) CRITICAL — FF-15 Phase 2 section is not deployable as written (missing mandatory RBAC dependencies)

The new architecture section states that:

- API server watches `ExporterPlugin` CRs
- plugins validate Bearer tokens via `TokenReview`

But the document does not specify the required RBAC additions, and current API server RBAC does not include `exporterplugins`.

**Why this matters:** this can fail at runtime even with otherwise correct code.

**Required correction:**

- explicitly document RBAC additions for API server and operator for `exporterplugins`
- explicitly document `tokenreviews.authentication.k8s.io` permission requirements for plugin auth flow (or clarify where that auth check runs and who needs it)

---

### 2) HIGH — `trustedSubjects` example uses the wrong API server service account name

The architecture example uses:

- `system:serviceaccount:assethub:assethub-api`

but deployed API SA is:

- `assethub-api-server`

**Why this matters:** plugin authors copying this will reject valid API server calls.

**Required correction:** update examples to actual SA names (`assethub-api-server`, `assethub-operator`) or clearly mark names as placeholders.

---

### 3) HIGH — State ambiguity: doc currently reads as implemented while codebase is pre-Phase-2

The new text says operator now "manages three concerns" and project structure includes `ExporterPlugin` CRD types. In current code:

- operator still wires one reconciler (`AssetHubReconciler`)
- scheme does not register `ExporterPlugin` types

**Why this matters:** this is high-risk documentation drift; implementers/users may treat planned architecture as current behavior.

**Required correction:** clearly label FF-15 Phase 2 content as **target-state / planned** until merged.

---

### 4) MEDIUM — Health semantics are incomplete for stale status

The section says operator downtime does not affect export functionality, but does not define stale-health behavior (e.g., transition from `Ready` to `Unknown` when probe timestamps go stale).

**Required correction:** define staleness policy and phase transitions (`Ready`, `Unhealthy`, `Error`, `Unknown`) explicitly.

---

## High / Critical Findings (Pre-existing in Upstream)

### A) HIGH — Export binding access model is incorrect for `/run`

Architecture states `run` is Admin+, but implementation route uses `requireRW` (RW+).

**Required correction:** update access model text to split mutation permissions:

- create/update/delete: Admin+
- run/download: RW+

---

### B) HIGH — Download endpoint contract is incomplete

Architecture lists:

- `/export-bindings/download?token={token}`

Implementation requires:

- `token` **and** `binding` query params.

**Required correction:** update endpoint contract to:

- `/export-bindings/download?token={token}&binding={binding-id}`

---

## Suggested Edits (Concise)

1. Mark FF-15 Phase 2 section as "planned/target state" until merged.
2. Fix `trustedSubjects` example SA names.
3. Add explicit RBAC requirements subsection for `ExporterPlugin` watchers and token review flow.
4. Add status staleness semantics for exporter health.
5. Correct export binding RBAC and download endpoint query contract in access/API sections.

---

## Bottom Line

The new `ExporterPlugin` architecture direction is solid, but it currently has one critical implementation gap (RBAC dependency omission) and two high-impact correctness issues (wrong SA name example and current-vs-target-state ambiguity). Upstream also still contains two high-severity documentation mismatches in export binding access and download endpoint contract.
