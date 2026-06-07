# Adversarial Review: FF-15 Phase 2 — Dynamic Export Plugins Design Spec

**Date:** 2026-06-03
**Reviewed spec:** `docs/plans/2026-05-28-dynamic-export-plugins-design.md`
**Reviewer:** Claude (adversarial review session)

---

## CRITICAL: Factual Errors vs Codebase

### C1. "Registry already has RWMutex for reads" — FALSE

> Design spec says (Dynamic Registry Changes section): "Thread-safety via `sync.RWMutex` (already has it for reads, need write lock for deregister)"

The actual `ExporterRegistry` (`internal/service/operational/export/registry.go:5-7`) has **no mutex at all**. It's a bare `map[string]Exporter` with no synchronization. It works today only because writes happen exclusively during single-threaded startup in `cmd/api-server/main.go`. Adding `Deregister()` without first adding a mutex makes every concurrent `Get()` and `List()` call a data race.

**Fix:** The spec must explicitly state that a `sync.RWMutex` needs to be *added*, not that one already exists. Both reads and writes must be guarded.

---

### C2. `ExportInput` design spec vs actual struct mismatch

The spec says the webhook POST /export receives "ExportInput JSON as built-in exporters receive (catalog name, parameters, instances by type, children, links)."

The actual `ExportInput` struct (`types.go:43-53`) has fields the spec doesn't mention:
- `CatalogDesc` — used by `MCPGatewayExporter` to populate the VirtualServer description
- `CVLabel` — defined but never populated (dead field, see `buildExportInput` line 486-492)
- `EntityTypes` — defined but never populated in `buildExportInput`
- `VirtualServerInstanceName` / `AllowedToolIDs` — set conditionally for VS-scoped exports
- `LinksByAssoc` — exists on `ExportInstance` but is **always an empty map** (line 569: initialized but never populated)

This matters because the webhook protocol JSON schema for `/export` must be a **precise, versioned contract**. The spec needs to decide:
1. Does the webhook receive the full `ExportInput` including dead/empty fields? Or a cleaned-up version?
2. Are `VirtualServerInstanceName` and `AllowedToolIDs` sent to webhook exporters? They're MCP-specific.
3. `LinksByAssoc` is always empty — is this a known bug in the current code, or intentional? If webhook exporters need links, this must be fixed first.

---

### C3. Webhook protocol `/export` response schema contradicts ExportOutput

The spec shows the export response as:
```json
{
  "artifacts": [{ "api_version": "...", "kind": "...", "name": "...", "namespace": "...", "yaml": "..." }],
  "warnings": [...]
}
```

But the actual `K8sArtifact` struct uses `APIVersion` (Go naming), not `api_version`. The JSON serialization of `K8sArtifact` (`types.go:82-88`) has **no json tags at all**. This means Go's default JSON marshaling would produce `APIVersion`, `Kind`, `Name`, `Namespace`, `YAML` — PascalCase, not the snake_case shown in the spec.

**Fix:** Either add json tags to `K8sArtifact` and `ExportOutput`, or use a separate DTO type for the webhook protocol. The spec must match whichever is chosen.

---

## ARCHITECTURAL ISSUES

### A1. Operator and API server share registry — but HOW?

> "The operator and API server share the exporter registry."

Today the operator and API server are **separate binaries** (`cmd/operator/main.go` vs `cmd/api-server/main.go`). They run in separate pods. They cannot share an in-memory Go struct.

The spec then says: "On startup, the API server reads existing CatalogExporter CRs and pre-populates the registry. The operator's watch loop handles subsequent changes."

This implies the operator must **notify** the API server of changes. How? The spec offers no mechanism. Options:
1. API server also watches CatalogExporter CRs (duplicating the operator's watch)
2. Operator calls an internal API server endpoint to register/deregister
3. API server re-reads CRs periodically

Option 1 means the "operator reconciler" for CatalogExporter is actually in the API server, not the operator — which contradicts "Add operator reconciler for CatalogExporter CRs" (Stage 2). Option 2 requires a new internal API endpoint, not mentioned in the spec. Option 3 has staleness.

**Fix:** The spec must pick one mechanism and describe it explicitly. The simplest is: the **API server** watches CatalogExporter CRs directly (no operator involvement for registry sync). The operator only handles health checks and status updates.

---

### A2. CatalogExporter controller — new controller or part of AssetHubReconciler?

The existing operator has a single `AssetHubReconciler` that handles AssetHub, CatalogVersion, and Catalog CRs via `SetupWithManager`. The spec says "Add operator reconciler for CatalogExporter CRs" but doesn't specify whether this is:

1. A new, independent controller (separate `Reconcile` method, separate `SetupWithManager`)
2. An extension of `AssetHubReconciler` (like CatalogVersion and Catalog are today)

If it's option 2, CatalogExporter CRs would need an AssetHub owner reference, which doesn't make sense for plugin-deployed exporters (the plugin author doesn't know the AssetHub CR name). If option 1, the spec should say so explicitly and note it's a different pattern from existing controllers.

---

### A3. Health check loop — who runs it, where?

> "Health check — operator periodically probes /health, updates status.phase"

But if the API server is the one maintaining the in-memory registry (per A1), the health status needs to reach the API server's `/exporters` response too. The spec says "Exporter list shows health status indicator" but `ListExporters` currently returns data from the in-memory `ExporterRegistry`, which has no health field.

Either:
- The API server watches CatalogExporter CR status (where the operator writes health) and includes it in the `/exporters` response
- The API server runs its own health probes
- `ExporterInfo` gets a health field sourced from... somewhere

The spec needs to close this loop explicitly.

---

## PROTOCOL AMBIGUITIES

### P1. Endpoint field — base URL or full path?

The CRD spec shows:
```yaml
endpoint: https://my-exporter.my-namespace.svc.cluster.local/export
```

But the webhook protocol describes three endpoints: `/validate`, `/export`, `/health`. If `endpoint` includes `/export`, how does the adapter construct the `/validate` and `/health` URLs? Does it strip the path and append? Replace the last segment?

The `WebhookExporter` struct has a single `endpoint` field described as `"e.g., https://my-exporter.ns.svc.cluster.local"` (no path), but the CRD example includes `/export` in the path.

**Fix:** Spec must clearly state: `endpoint` is the **base URL** (no path), and the adapter appends `/validate`, `/export`, `/health`. Update the CRD example accordingly.

---

### P2. Validate request body uses "schema" — shape incomplete

The `/validate` request body has two top-level keys: `parameters` and `schema`. But the existing `ValidateSchema` interface method signature is:
```go
ValidateSchema(params map[string]string, schema SchemaInfo) error
```

The webhook protocol's `schema` field uses a different shape than `SchemaInfo`. The protocol shows:
```json
{"entity_types": [{"name": "...", "attributes": [...], "associations": [...]}]}
```

But `SchemaInfo` has `EntityTypes []SchemaEntityType` where `SchemaAssociation` includes `Name`, `Type`, and `TargetEntityType`. The webhook protocol example doesn't show all these fields. Is the webhook schema a subset? The full `SchemaInfo`? Something different?

**Fix:** The spec should show the complete JSON schema for the `/validate` request, including all association fields, so plugin authors know exactly what they receive.

---

### P3. Auth: "Asset Hub sends its SA token" — which SA token?

> "Asset Hub sends its SA token in Authorization header. Plugin validates via TokenReview."

But which process sends the request? If the operator probes `/health`, it uses the operator's SA token. If the API server calls `/validate` and `/export`, it uses the API server's SA token. These are different pods with different service accounts.

The plugin must accept **both** tokens (or the spec must say they share a service account). This isn't mentioned.

---

### P4. Timeout "proportional scaling" is contradictory

> "5s validate, 10s export (default). Configurable per CR via `spec.timeoutSeconds`. Proportional scaling: validate = timeout/2, export = timeout"

So if `timeoutSeconds: 15`, then validate = 7.5s and export = 15s? A fractional timeout is unusual. And for `timeoutSeconds: 5`, validate = 2.5s. For `timeoutSeconds: 3`, validate = 1.5s — which is probably too short for a network round-trip.

**Fix:** Consider a minimum timeout floor, and clarify whether the division is integer or float.

---

## CORNER CASES

### E1. Orphaned bindings: RunAll / PublishPreview behavior

The spec says orphaned bindings (exporter deregistered) return an error on `RunBinding`. But what about `RunAll` and `PublishPreview`?

Looking at `executeBinding` (line 314-349), it already handles "exporter not found" by returning `BindingStatusFailed` in the result. But `PublishPreview` is used for **publishing**, and a failed binding currently sets `hasFailures: true`. Does an orphaned binding block publishing? The spec doesn't say.

**Fix:** Clarify: should orphaned bindings be skipped in RunAll/PublishPreview, or should they fail? If skipped, should there be a separate "skipped" status?

---

### E2. Race between CR deletion and in-flight export

If a CatalogExporter CR is deleted while an export is in progress:
1. Operator deregisters the exporter from the registry
2. An in-flight `Export()` call on the `WebhookExporter` is still running
3. The HTTP request may succeed (the plugin service is still running during graceful shutdown)
4. But the `WebhookExporter` instance was removed from the registry

The `WebhookExporter` adapter is a struct held by value in the registry map. Once `Deregister` removes it, any goroutine still holding a pointer to it can still use it. But if the underlying HTTP client is shared, and we close it on deregister, we'd interrupt in-flight requests.

**Fix:** Spec should state that deregistration is lazy — the `WebhookExporter` reference remains valid for in-flight calls. Only future `Get()` calls return not-found.

---

### E3. "Exporter always appears in list" vs deregistration

The spec says under Health checking: "Exporter always appears in list." But under Dynamic Registry Changes: "`Deregister(name string)` — removes an exporter."

These are contradictory. If the exporter is deregistered (CR deleted), it's removed from the registry and won't appear in `List()`. But the spec also says "Show always, mark status."

**Fix:** Clarify: does "always" mean even after CR deletion? If so, the registry needs a "tombstone" state, not true deregistration. Or does "always" only mean "even when unhealthy" (i.e., don't hide unhealthy exporters)?

---

### E4. CRD parameter schema — `type: entity_type` is project-specific

The `ParameterDef.Type` field can be `"entity_type"` or `"string"`. The `entity_type` type has special behavior: the UI renders a dropdown of pinned entity types, and the binding service validates the value against CV pins.

For webhook exporters, this validation still happens in `validateParamEntityTypes()` (line 393-405) based on `strings.HasSuffix(key, "_type")`, **not** based on `ParameterDef.Type`. So a webhook exporter with a parameter named `output_type` of type `string` would still get entity-type validation because the key ends in `_type`.

**Fix:** The spec should clarify whether webhook exporters inherit the `_type` suffix heuristic, or whether only the explicit `type: entity_type` field matters. Ideally, fix the heuristic to use the declared type.

---

### E5. Import/Export interaction — incomplete

The spec says: "On import to a system without the webhook exporter, bindings are orphaned (warning)."

But the current import service (`internal/service/operational/import_service.go`) handles export bindings by exporter name. When importing, it would try to create bindings referencing an exporter that doesn't exist. Does `CreateBinding` fail? Looking at `validateBindingParams` (line 117-137): yes, it calls `registry.Get()` and returns "exporter not found."

So import would **fail**, not create orphaned bindings. The spec's claim is wrong.

**Fix:** Either the import service must skip binding validation for missing exporters (creating them as orphaned), or the spec must say import fails for bindings referencing missing webhook exporters.

---

### E6. Namespace scope inconsistency

> "CR scope: Namespace-scoped. Same namespace as Asset Hub."

But the CRD example shows:
```yaml
endpoint: https://my-exporter.my-namespace.svc.cluster.local/export
```

The endpoint references `my-namespace`, which can be a different namespace from the CatalogExporter CR's namespace. This is fine (cross-namespace service calls work), but the spec should explicitly state whether the webhook service must be in the same namespace as the CR, or can be in any namespace.

---

## DESIGN GAPS

### D1. No versioning on the webhook protocol

The spec defines a protocol but doesn't version it. If the `/export` request body gains new fields in a future release, existing plugins would receive unexpected JSON keys. If the response format changes, the adapter would break.

**Fix:** Add a protocol version header (e.g., `X-AssetHub-Protocol-Version: v1`) or version the endpoint paths (e.g., `/v1/export`).

---

### D2. No error body contract for /export

The spec defines the success response for `/export` (200 with artifacts) but not the error response. What status code? What body shape? Is it the same `{"valid": false, "error": "..."}` as `/validate`? Or a different shape?

Looking at the existing code, `Export()` returns `(*ExportOutput, error)`. The adapter needs to map HTTP errors to Go errors. The spec should define: 4xx = validation error (retryable by fixing input), 5xx = internal error (retry later), and specify the body format for both.

---

### D3. `ListExporters` response needs health + source fields

Currently `ExporterInfo` has only `name`, `description`, and `parameter_schema`. The spec says the UI should show health status indicators and differentiate built-in from webhook. But `ExporterInfo` has no `health` or `source` field.

The spec lists this under Stage 3 ("health status in exporter list") but doesn't define the new response shape. The API contract should be specified.

---

### D4. No retry/backoff on webhook calls

The spec defines timeouts but no retry policy. If the webhook service returns 503 or times out, does the adapter retry? How many times? With backoff? The MCPGatewayExporter (built-in) never fails due to network issues, so this is a new failure mode unique to webhook exporters.

---

### D5. No size limit on webhook response

A malicious or buggy webhook plugin could return a 100MB response. The adapter should enforce a maximum response body size. Not mentioned in the spec.

---

### D6. MCP Gateway "webhook clone" — dual maintenance burden

> "Built-in stays for demo stability. Webhook clone proves the pattern. Built-in removed later after webhook is validated."

The spec calls for implementing the same logic twice — once as compiled Go, once as a standalone webhook service. This creates a maintenance window where bugs must be fixed in both places. The spec doesn't define criteria for "validated" (when is the built-in removed?) or who maintains the clone during the overlap period.

---

## NAMING / TERMINOLOGY ISSUES

### N1. "CatalogExporter" vs "ExportBinding" — confusing naming

The system already has `ExportBinding` (a binding of an exporter to a catalog with parameters). Now we add `CatalogExporter` (a CR that registers a webhook exporter). The word "Catalog" in "CatalogExporter" suggests it's catalog-scoped, but it's actually a global exporter registration that can be bound to any catalog.

A name like `ExporterPlugin` or `WebhookExporter` for the CRD would be less ambiguous.

---

### N2. Endpoint field confusion

The `WebhookExporter` struct has `endpoint string` described as the base URL. But the CRD `spec.endpoint` includes `/export` in the example. The word "endpoint" is overloaded — it means "base URL of the service" in one place and "full URL including path" in another.

---

## SUMMARY TABLE

| # | Severity | Issue | Required Action |
|---|----------|-------|-----------------|
| C1 | **Critical** | Registry has no mutex — spec claims it does | Fix factual error, describe full mutex addition |
| C2 | **Critical** | ExportInput fields don't match spec's description | Define exact webhook JSON schema |
| C3 | **Critical** | K8sArtifact has no json tags — PascalCase vs snake_case | Add json tags or define DTO |
| A1 | **Critical** | Operator/API server can't share in-memory registry | Pick and describe sync mechanism |
| A2 | **High** | New controller vs extending AssetHubReconciler unclear | Specify controller architecture |
| A3 | **High** | Health status data flow to API response undefined | Close the loop |
| P1 | **High** | Endpoint = base URL or full path? | Clarify, fix CRD example |
| P2 | **Medium** | Validate request schema incomplete | Show complete JSON schema |
| P3 | **Medium** | SA token — operator vs API server | Clarify auth identity |
| P4 | **Low** | Fractional timeout, no minimum floor | Add floor, clarify rounding |
| E1 | **High** | Orphaned bindings in RunAll/PublishPreview | Define behavior |
| E2 | **Medium** | Race on deregister during in-flight export | Specify lazy deregistration |
| E3 | **High** | "Always show" contradicts Deregister | Resolve contradiction |
| E4 | **Medium** | `_type` suffix heuristic vs declared type | Clarify validation rule |
| E5 | **High** | Import fails, doesn't create orphans | Fix spec or fix import |
| E6 | **Low** | Cross-namespace endpoint not addressed | Clarify namespace constraints |
| D1 | **Medium** | No protocol versioning | Add version header/path |
| D2 | **Medium** | No error response contract | Define error body format |
| D3 | **Medium** | ExporterInfo missing health/source fields | Define new API response shape |
| D4 | **Low** | No retry policy for webhook calls | Define retry behavior |
| D5 | **Low** | No response size limit | Add max body size |
| D6 | **Low** | Dual maintenance of MCP Gateway logic | Define removal criteria |
