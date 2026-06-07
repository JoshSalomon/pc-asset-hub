# Adversarial Review: FF-15 Phase 2 Dynamic Export Plugins

**Date:** 2026-06-03  
**Reviewed spec:** `docs/plans/2026-05-28-dynamic-export-plugins-design.md`  
**Reviewer:** Codex (adversarial pass)

---

## Review Goal

Stress-test the design for correctness before implementation by finding:

- internal inconsistencies,
- codebase/spec mismatches,
- hidden edge cases,
- and ambiguous dual meanings that can lead to wrong implementation.

---

## Critical Findings

### C1. Response size limit is specified with the wrong API

**Issue**  
The spec says the webhook client should enforce a 10 MB response limit with `http.MaxBytesReader(...)`.

**Why this is critical**  
`http.MaxBytesReader` is a server-side helper for request bodies, not a client-side response reader. Implementing this as written will fail or be silently wrong.

**Fix required**  
Define client-side enforcement explicitly (for example, bounded read with overflow detection), and define behavior on overflow (`validation` error vs `internal` error).

---

### C2. ExporterPlugin CRD API group conflicts with existing project group

**Issue**  
Spec uses `apiVersion: assethub.io/v1alpha1`, but project CRDs and scheme use `assethub.project-catalyst.io/v1alpha1`.

**Why this is critical**  
If implemented as written, CRs will not be recognized by existing scheme/RBAC and watches/controllers will fail.

**Fix required**  
Use the existing group `assethub.project-catalyst.io/v1alpha1` unless there is an intentional migration plan (which would require explicit compatibility and rollout sections).

---

### C3. Import/orphaned-binding behavior conflicts with current FF-12 scope

**Issue**  
Spec says import should create orphaned bindings when exporter is missing, but FF-12 currently excludes export bindings from catalog export/import payloads.

**Why this is critical**  
This is a direct scope contradiction: there are no imported bindings to orphan unless FF-12 format changes first.

**Fix required**  
Pick one:

1. defer orphaned import behavior to the phase that adds bindings to FF-12 payloads, or
2. explicitly expand FF-12 format in this phase and document migration/versioning impact.

---

## High-Severity Findings

### H1. Exporter name collision policy is undefined

The design keeps built-in `mcp-gateway` while adding dynamic plugins, but does not define behavior when a plugin CR uses a built-in name.

**Fix:** Reserve built-in names; reject colliding CRs with clear status/message.

---

### H2. Health status model is underdefined

Phases include `Ready`, `Unhealthy`, `Error`, `Unknown`, but only one transition is specified (non-200 => `Unhealthy`).

Missing:

- when to set `Error` vs `Unhealthy`,
- when to use `Unknown`,
- stale status behavior when probes stop,
- recovery transitions.

**Fix:** Add explicit state machine and transition table.

---

### H3. Protocol version has dual meaning (header and body)

Spec uses both:

- header `X-AssetHub-Protocol-Version: v1`
- body field `protocol_version: "v1"`

No mismatch behavior is defined.

**Fix:** Make one canonical source (prefer header), or require strict equality and reject mismatch.

---

### H4. Payload typing and link completeness are ambiguous

The protocol claims a clean contract, but current builder behavior has known gaps:

- `links_by_assoc` is currently empty in Phase 1 input construction,
- value typing behavior (string/number/json) is not frozen as a strict wire contract.

**Fix:**  
Either:

1. remove unsupported fields from v1 contract, or
2. make link population a hard prerequisite and define a strict type matrix.

---

### H5. Auth trust boundary is too broad

Spec says plugin validates tokens and accepts any valid cluster token.

**Risk:** Any cluster service account token (not just Asset Hub components) may be accepted if network path exists.

**Fix:** Require subject allowlist (API server SA + operator SA), optionally with audience checks.

---

## Medium-Severity Findings

### M1. "Same pattern as CatalogVersion informer" is factually misleading

The spec says ExporterPlugin watch is "same pattern as CatalogVersion stage filtering", but current stage filtering is service-layer DB filtering, not informer-driven.

**Fix:** Reword to avoid implying an existing informer pattern that does not exist.

---

### M2. Implementation file list is incomplete/inaccurate

The design lists `config/operator/crd/exporterplugin.yaml`, but this repo uses `deploy/k8s/operator/*` for operator CRD manifests.  
Also missing explicit mention of required edits for:

- scheme registration (`AddToScheme`) for `ExporterPlugin`,
- API server and operator RBAC additions for new resource watches/updates.

**Fix:** Update file plan to match repo structure and include all required wiring files.

---

### M3. New `skipped` status is not formalized in core status model

Spec introduces orphan skip behavior (`status: "skipped"`), but current binding status model is only `never/success/failed`.

**Fix:** Add `BindingStatusSkipped` as a first-class constant and document where it appears (run, preview, last_run_status) and whether it contributes to `has_failures`.

---

### M4. Non-K8s runtime behavior is unspecified for dynamic plugin discovery

Current API server only creates cluster clients in-cluster. The design does not define behavior in local/non-cluster runs where CR watching is unavailable.

**Fix:** Document expected behavior in non-K8s mode (feature disabled with warning vs fallback).

---

## Design Dualities / Ambiguous Meanings

1. **"Always appears in list" vs "deregister on delete"**  
   Clarified partially in spec, but needs strict API contract wording: "always while CR exists, even if unhealthy."

2. **"Import creates orphaned bindings" vs "bindings not in FF-12 export"**  
   This remains unresolved and must be reconciled before implementation.

3. **Protocol version in two places**  
   Needs a single source of truth.

---

## Corner Cases Not Fully Nailed Down

1. CR deleted during in-flight export (what outcome is guaranteed for current run?)
2. Plugin endpoint DNS resolves but TLS/handshake fails repeatedly (phase and retry semantics)
3. Health probe auth mismatch (operator token rejected while API token accepted)
4. Plugin returns malformed JSON with 200 status (error classification)
5. Same exporter recreated quickly after delete (watch event ordering and registry final state)

---

## Required Spec Corrections Before Build

1. Fix CRD API group and RBAC plan.
2. Correct client-side response-size enforcement mechanism.
3. Resolve FF-12 import/binding scope contradiction.
4. Define exporter name collision policy.
5. Define health phase state machine and stale status handling.
6. Freeze protocol v1 contract (typing, links, version source of truth).
7. Formalize `skipped` status in domain/API/UI contracts.
8. Update implementation file map to real repo paths and mandatory wiring changes.

---

## Optional Hardening (Recommended)

- Add `spec.caBundle` / TLS trust model for webhook services.
- Add endpoint scheme policy (`https` only by default).
- Add retry/backoff policy for transient 503/timeouts.
- Add idempotency guidance for plugin `/export` handlers.
- Add conformance test suite for third-party webhook plugins.

---

## Bottom Line

The revised design is significantly stronger than the initial draft, but there are still a few blocker-level correctness issues and several high-risk ambiguities. Addressing the critical and high-severity findings above will materially reduce implementation churn and production risk.
