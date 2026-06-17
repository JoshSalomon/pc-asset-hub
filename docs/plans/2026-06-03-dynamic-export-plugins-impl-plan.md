# FF-15 Phase 2: Dynamic Export Plugins — Implementation Plan

**Revised:** 2026-06-03 (incorporated findings from two adversarial reviews)

<!--
## Persona & Goals

I am a craftsman of reliable software. Every line of code is backed by a test
that proves it works. TDD is how I think — I write the test first because
understanding what "correct" means comes before writing the solution.

**Goal for this plan:** Break the feature into small, self-contained units that
can each be tested in isolation BEFORE anything depends on them. Bottom-up
ordering ensures every foundation is proven before we build on it. The goal is
zero surprises at integration time — every component has been verified against
its contract before it meets its neighbors.

**What this means practically:**
- Registry mutex is tested with -race before any concurrent code uses it
- Webhook protocol DTOs are roundtrip-tested before the adapter serializes them
- WebhookExporter is tested with httptest before the informer creates instances
- CRD types have DeepCopy tests before the operator controller uses them
- Every step ends with a green test suite and zero regressions
-->

## Context

FF-15 Phase 1 (PR #23) shipped a compiled-in export plugin framework: one Go
`Exporter` interface, one registry, one MCP Gateway exporter, binding CRUD, and
publish preview. Phase 2 adds **zero-recompilation extensibility** — plugin
authors deploy a Deployment+Service and create an ExporterPlugin CR. The Asset
Hub discovers it via K8s informer and makes it available through the existing
binding flow.

Design spec: `docs/plans/2026-05-28-dynamic-export-plugins-design.md`
Design reviews: `docs/plans/2026-06-03-dynamic-export-plugins-review.md`,
`docs/plans/2026-06-03-dynamic-export-plugins-review-v2.md`
Plan reviews: `docs/plans/2026-06-03-dynamic-export-plugins-impl-plan-adversarial-review.txt`,
`docs/plans/2026-06-03-impl-plan-review.txt`

## Implementation Steps

### Step 1: Foundation — `BindingStatusSkipped` + Fix `validateParamEntityTypes` (T-36.01–T-36.05)

Two independent base-level changes that everything else builds on.

**1a. Add `BindingStatusSkipped = "skipped"` constant**
- File: `internal/service/operational/export/types.go` (line 8, after existing constants)
- Test: assert constant value equals `"skipped"`

**1b. Fix `validateParamEntityTypes` to use declared type, not name suffix**

Current code (`binding_service.go:393-406`) uses `strings.HasSuffix(key, "_type")`.

**Signature change required** (review R2#1): the current function signature is
`validateParamEntityTypes(params map[string]string, schema SchemaInfo) error` —
it has no access to the exporter's ParameterSchema. Change the signature to:
```go
func (s *ExportBindingService) validateParamEntityTypes(
    params map[string]string, schema SchemaInfo, paramDefs []ParameterDef) error
```

Update the caller `validateBindingParams` (line 132) to pass `exporter.ParameterSchema()`.

Fix: iterate `paramDefs`, build a set of param names where
`param.Type == "entity_type"`, then validate only those params against CV pins.

Tests (RED first):
- Param `output_type` with type `"string"` → must NOT trigger entity-type validation (current code wrongly validates it)
- Param `server_ref` with type `"entity_type"` → must validate against CV pins (current code misses it because no `_type` suffix)
- Existing behavior preserved for params like `server_type` with type `"entity_type"`

**Verify:** `go test ./internal/service/operational/export/... -count=1` — all existing + new tests pass.

---

### Step 2: Thread-Safe Registry (T-36.06–T-36.12)

Current `ExporterRegistry` (`registry.go:5-7`) is a bare `map[string]Exporter`
with **no mutex**. Phase 2 adds concurrent writes from the informer watch.

**Changes to `registry.go`:**
- Add `mu sync.RWMutex` and `builtIn map[string]bool` fields
- Wrap `Register()` with write lock; mark `builtIn[name] = true`
- Wrap `Get()` with read lock
- Wrap `List()` with read lock
- Add `RegisterWebhook(e Exporter)` — write lock, `builtIn[name] = false`
- Add `Deregister(name string) bool` — write lock, refuse if built-in, return false if not found
- Add `IsBuiltIn(name string) bool` — read lock

Tests (RED first):
- `Register` marks built-in; `RegisterWebhook` does not
- `Deregister` succeeds for webhook, refused for built-in, false for not-found
- `RegisterWebhook` twice overwrites (last wins)
- **Concurrent access** — 50 goroutines doing Register/Get/List/Deregister. Must pass `go test -race`.

**Verify:** `go test -race ./internal/service/operational/export/... -run TestRegistry -count=1`

---

### Step 3: Webhook Protocol DTOs (T-36.13–T-36.20)

New file: `internal/service/operational/export/webhook_protocol.go`

Separate serialization types with explicit `json:"snake_case"` tags. These are
the **webhook wire contract** — they must NOT change without bumping the
protocol version.

Types:
- `WebhookValidateRequest` — `Parameters`, `Schema` (with full association detail: Name, Type, TargetEntityType)
- `WebhookValidateResponse` — `Valid`, `Error`
- `WebhookExportRequest` — `CatalogName`, `CatalogDescription`, `Parameters`, `InstancesByType`, `ChildrenOf`
- `WebhookInstance` — `ID`, `Name`, `Description`, `Attributes map[string]any`, `ParentID`
- `WebhookExportResponse` — `Artifacts []WebhookArtifact`, `Warnings`
- `WebhookArtifact` — `APIVersion`, `Kind`, `Name`, `Namespace`, `YAML` (all snake_case json tags)
- `WebhookErrorResponse` — `Error`
- `WebhookHealthResponse` — `Status`

Conversion helpers:
- `SchemaInfoToWebhook(SchemaInfo) WebhookSchemaInfo`
- `ExportInputToWebhookRequest(ExportInput) WebhookExportRequest` — excludes dead fields: `CVLabel`, `EntityTypes`, `VirtualServerInstanceName`, `AllowedToolIDs`, `LinksByAssoc`
- `WebhookExportResponseToOutput(WebhookExportResponse) *ExportOutput`

**VS filtering parity note** (review H3): `VirtualServerInstanceName` and
`AllowedToolIDs` are excluded from the webhook contract because the filtering
is applied BEFORE the exporter sees the data. In `binding_service.go:Run()`
(lines 238-244), `resolveVSInstanceTools()` populates `AllowedToolIDs`, and
`buildExportInput()` uses it to pre-filter `InstancesByType` and `ChildrenOf`.
The webhook exporter receives already-filtered data — it does not need these
fields. Document this in the protocol spec.

**EntityTypes exclusion note** (review R2#2): `EntityTypes` is excluded from v1
because it's dead (never populated by `buildExportInput`). This limits webhook
plugins that need schema context. Accepted for v1 — document as known
limitation. Can be added in v2 by populating the field in `buildExportInput`.

Tests:
- JSON roundtrip for each type — verify snake_case field names in marshaled output
- `ExportInputToWebhookRequest` excludes dead fields (set all dead fields, verify absent in result)
- `WebhookArtifact` marshals to `api_version`, not `APIVersion`
- Mixed attribute types (string, int, bool, array) roundtrip correctly

**Verify:** `go test ./internal/service/operational/export/... -run TestWebhook -count=1`

**Depends on:** Step 1 (uses ExportInput type definition)

---

### Step 4: `WebhookExporter` Adapter (T-36.21–T-36.46)

New file: `internal/service/operational/export/webhook_exporter.go`

Implements `Exporter` interface by forwarding to HTTP endpoints. All tests use
`httptest.NewServer` — no real network.

```go
type WebhookExporter struct {
    name, description, baseURL string
    paramSchema    []ParameterDef
    validateTimeout, exportTimeout time.Duration
    healthStatus   string
    httpClient     *http.Client
    tokenGetter    func() string    // returns SA bearer token; injected at construction
}
```

`baseURL` is the service URL **without path** (verified against design spec P1 fix).
Adapter appends `/validate` and `/export`.

All requests include:
- `X-AssetHub-Protocol-Version: v1` header
- `Authorization: Bearer {token}` header (token from `tokenGetter()`)

Response handling (from design spec, verified):

| HTTP Status | Behavior |
|-------------|----------|
| 200 + valid JSON | Success |
| 200 + malformed JSON | Error: "not valid JSON" |
| 200 + >10MB | `domainerrors.NewValidation("exceeds 10 MB")` |
| 400-499 | `domainerrors.NewValidation(body.error)` |
| 500-599 | `fmt.Errorf("webhook plugin error: %s")` |
| Timeout | `fmt.Errorf("timed out after %s")` |
| Connection refused | `fmt.Errorf("unreachable: %s")` |

Response size enforced via `io.LimitReader(resp.Body, 10*1024*1024+1)` — read
into buffer, check length. NOT `http.MaxBytesReader` (that's server-side,
per review C1).

HTTP redirects disabled: set `http.Client.CheckRedirect` to reject all
redirects. A redirect from a webhook plugin endpoint is always a
misconfiguration (auth proxy, wrong service). Return clear error
"unexpected redirect to {url}" instead of following and getting HTML.

Timeout calculation: `exportTimeout = max(3, timeoutSeconds)`,
`validateTimeout = max(3, timeoutSeconds/2)` (integer division, min floor 3s,
per review P4).

**Deletion safety note** (review R2#9): Once a caller has a reference from
`registry.Get()`, the `WebhookExporter` remains usable even after
deregistration. Go's map delete does not invalidate existing pointers. Only
future `Get()` calls return not-found. This is safe by design.

Tests (RED first, using httptest):
- Basic getters (Name, Description, ParameterSchema)
- ValidateSchema success (200 `{"valid":true}`)
- ValidateSchema error (400 `{"valid":false,"error":"msg"}`)
- Protocol version header present on requests
- Authorization header present
- Export success (200 with artifacts)
- Export 4xx, 5xx, timeout, connection refused, oversized, malformed JSON
- Dead fields excluded from request body
- Timeout calculation (timeoutSeconds=15 → validate=7, export=15; timeoutSeconds=3 → both=3)
- HealthStatus get/set

**Verify:** `go test ./internal/service/operational/export/... -run TestWebhookExporter -count=1`

**Depends on:** Steps 2 (registry), 3 (protocol DTOs)

---

### Step 5: `ExporterInfo` with Source + Health (T-36.47–T-36.51)

Extend `ExporterInfo` (`registry.go:9-13`) with two new fields:
```go
Source string `json:"source"` // "built-in" or "webhook"
Health string `json:"health"` // "Ready", "Unhealthy", "Error", "Unknown", "n/a"
```

In `List()`: for built-in exporters → `Source: "built-in"`, `Health: "n/a"`.
For webhook exporters (type-assert to `*WebhookExporter`) → `Source: "webhook"`,
`Health: we.HealthStatus()`.

Tests:
- List with built-in → source="built-in", health="n/a"
- List with webhook (health "Ready") → source="webhook", health="Ready"
- List with mixed → each correct
- Handler test: GET /exporters returns `source` and `health` in JSON

**Verify:** `go test ./internal/service/operational/export/... -run TestRegistry_List -count=1` + `go test ./internal/api/operational/... -run TestListExporters -count=1`

**Depends on:** Steps 2, 4

---

### Step 6: ExporterPlugin CRD Go Types (T-36.52–T-36.58)

New file: `internal/operator/api/v1alpha1/exporterplugin_types.go`

Follow exact pattern of `catalog_types.go`:
- `ExporterPluginSpec`: Description, Endpoint (required), TimeoutSeconds (default 10, min 3), ParameterSchema, TrustedSubjects
- `ExporterPluginStatus`: Phase (Ready/Unhealthy/Error/Unknown), LastHealthCheck (ISO 8601), Message
- `ExporterPlugin`, `ExporterPluginList` with TypeMeta, ObjectMeta, Spec, Status
- DeepCopyInto, DeepCopy, DeepCopyObject for both (follow catalog_types.go pattern exactly)
- API group: `assethub.project-catalyst.io` (verified from `assethub_types.go:10`)

**TrustedSubjects purpose** (review R2#3): This field tells the plugin service
which K8s service account subjects to accept via TokenReview. It's consumed by
the plugin (not the Asset Hub). The CRD carries it so the plugin can read its
own CR to get the allowlist at startup. Default (if omitted): the plugin should
accept `{namespace}:assethub-api-server` and `{namespace}:assethub-operator`.

Modify `assethub_types.go:addKnownTypes()` (line 17-28): add `&ExporterPlugin{}`, `&ExporterPluginList{}`

Tests:
- DeepCopy independence (modify copy, original unchanged)
- DeepCopy nil returns nil
- DeepCopyObject returns non-nil runtime.Object
- Nil slices handled (ParameterSchema, TrustedSubjects)
- List DeepCopy
- AddToScheme registers ExporterPlugin kind

**Verify:** `go test ./internal/operator/api/v1alpha1/... -count=1`

**Depends on:** Nothing (independent of export package)

---

### Step 7: CRD YAML + RBAC Manifests (T-36.59–T-36.64)

New file: `deploy/k8s/operator/exporterplugin-crd.yaml`
- Group: `assethub.project-catalyst.io`, version: `v1alpha1`
- Kind: `ExporterPlugin`, plural: `exporterplugins`, shortNames: `ep`
- Scope: Namespaced
- Follow exact structure of `catalog-crd.yaml`
- Status subresource enabled

Modify: `deploy/k8s/operator/role.yaml` (operator RBAC)
- Add rule: apiGroups `["assethub.project-catalyst.io"]`, resources `["exporterplugins", "exporterplugins/status"]`, verbs `["get", "list", "watch", "update", "patch"]`

**Modify: `deploy/k8s/api-server/rbac.yaml`** (API server RBAC — review C2)
- Add rule: apiGroups `["assethub.project-catalyst.io"]`, resources `["exporterplugins"]`, verbs `["get", "list", "watch"]`
- The API server only needs read access (watch for registry sync). It does NOT need update/patch (status updates are done by the operator).

**Update: `scripts/kind-deploy.sh` or Makefile** (review R2#13)
- Apply `exporterplugin-crd.yaml` BEFORE deploying the operator, so the operator's controller setup doesn't fail on a missing CRD.

**Verify:** `kubectl apply --dry-run=client -f deploy/k8s/operator/exporterplugin-crd.yaml` succeeds. Deploy to Kind and verify CRD is registered: `kubectl get crd exporterplugins.assethub.project-catalyst.io`.

**Depends on:** Step 6 (types must match schema)

---

### Step 8: Operator Health Probe Controller (T-36.65–T-36.77)

New file: `internal/operator/controllers/exporterplugin_controller.go`

New **independent** controller (NOT extending AssetHubReconciler — per review A2).

```go
type ExporterPluginReconciler struct {
    client.Client
    Scheme        *runtime.Scheme
    HTTPClient    *http.Client
    ProbeInterval time.Duration   // default 30s (review R2#4)
    TokenGetter   func() string   // returns operator SA bearer token (review H4)
}
```

`Reconcile()`: Get CR → probe `GET {spec.endpoint}/health` with
`Authorization: Bearer {token}` header (review H4 — health probes MUST send
auth, otherwise plugins enforcing subject allowlist will reject probes and
appear permanently Unhealthy) → classify result → update `status.phase`,
`status.lastHealthCheck`, `status.message` → requeue after `ProbeInterval`.

Pure function `classifyHealthResult(statusCode int, err error) (phase, message string)`:
- 200 → Ready, ""
- 401/403 → Error, "health probe authentication rejected" (distinguishes auth failure from unhealthy)
- Non-200 → Unhealthy, "health probe returned HTTP {code}"
- Timeout (context.DeadlineExceeded) → Unhealthy, "health probe timed out"
- Connection/DNS/TLS error → Error, "plugin unreachable: {err}"

Stale-to-Unknown transition: in `Reconcile()`, before probing, check if
`status.lastHealthCheck` is older than 3x `ProbeInterval` (default 90s). If
stale and current phase is not Unknown, set `status.phase=Unknown` with message
"health probe stale — no result for >90s". This covers operator downtime.

`SetupWithManager()`: `ctrl.NewControllerManagedBy(mgr).For(&v1alpha1.ExporterPlugin{}).Complete(r)`

Modify `cmd/operator/main.go`: register `ExporterPluginReconciler` after `AssetHubReconciler`, with `ProbeInterval: 30 * time.Second`.

Tests (using fake client + httptest):
- `classifyHealthResult` pure function tests (5 cases including 401)
- Reconcile with CR not found → no error (idempotent)
- Reconcile with healthy plugin (httptest 200) → status.phase = Ready
- Reconcile with unhealthy plugin (httptest 503) → status.phase = Unhealthy
- Reconcile with unreachable plugin → status.phase = Error
- Reconcile sends Authorization header

**Verify:** `go test ./internal/operator/controllers/... -run TestExporterPlugin -count=1`

**Depends on:** Step 6 (CRD types)

---

### Step 9: Orphaned Binding Handling (T-36.78–T-36.90)

Modify `binding_service.go` — `RunAll()` (line 294-312):
Before calling `executeBinding()`, check `registry.Get(binding.ExporterName)`.
If not found, append `BindingRunResult{Status: BindingStatusSkipped, Error: "exporter not registered"}` and **continue** (do NOT call `updateBindingStatus` — `last_run_status` is NOT changed for skipped bindings, review M8).

Modify `publish_service.go` — `PublishPreview()` (line 39-64):
Add early continue **before line 44** that catches missing exporters (review R2#6).
This must prevent BOTH the ValidateSchema block (lines 44-55) AND `executeBinding`
(line 57) from running. Without this, the current code falls through the
`if exporter, ok` block (because !ok) and reaches `executeBinding`, which
produces a BindingStatusFailed result instead of BindingStatusSkipped.

```go
// Before the existing exporter check (line 44):
if _, ok := s.registry.Get(binding.ExporterName); !ok {
    results = append(results, BindingRunResult{
        BindingID:    binding.ID,
        ExporterName: binding.ExporterName,
        Status:       BindingStatusSkipped,
        Error:        "exporter not registered",
    })
    continue  // skip both validation AND executeBinding
}
```

Skipped bindings do NOT set `hasFailures = true`.

**Persistence semantics** (review M8): `BindingStatusSkipped` appears only in
`BindingRunResult.Status` (transient run result). It is NOT persisted to
`binding.LastRunStatus` — that field retains whatever value it had from the last
actual run (success/failed/never). The UI shows the current-run skipped state
from the result, not from the persisted field.

Tests (RED first):
- `RunAll` with one registered and one orphaned exporter → registered runs (success), orphaned returns skipped. Orphaned binding's `LastRunStatus` NOT updated.
- `PublishPreview` with orphaned binding → `HasFailures` is false. Result includes skipped entry.
- `PublishPreview` with one orphaned + one failed → `HasFailures` is true (from the failed one, not the skipped one).

**Verify:** `go test ./internal/service/operational/export/... -run "TestRunAll_Skips|TestPublishPreview_Skips" -count=1`

**Depends on:** Steps 1 (BindingStatusSkipped), 2 (registry.Get)

---

### Step 10: API Server ExporterPlugin CR Watch (T-36.106–T-36.117)

New file: `internal/infrastructure/k8s/exporterplugin_watcher.go`

**Watch mechanism** (review R2#7, H5): `client.Client` from controller-runtime
is request-response only — it cannot watch. Use a **controller-runtime
`cache.Cache`** with an informer. The cache handles reconnection, backoff, and
resourceVersion tracking automatically. Approach:

```go
type ExporterPluginWatcher struct {
    registry  *export.ExporterRegistry
    namespace string
    cache     cache.Cache  // controller-runtime cache with built-in informer
}

func NewExporterPluginWatcher(cfg *rest.Config, registry *export.ExporterRegistry, namespace string) (*ExporterPluginWatcher, error) {
    scheme := runtime.NewScheme()
    _ = v1alpha1.AddToScheme(scheme)

    c, err := cache.New(cfg, cache.Options{
        Scheme: scheme,
        DefaultNamespaces: map[string]cache.Config{namespace: {}},
    })
    if err != nil {
        return nil, err
    }
    return &ExporterPluginWatcher{registry: registry, namespace: namespace, cache: c}, nil
}

func (w *ExporterPluginWatcher) Start(ctx context.Context) error {
    // Get informer for ExporterPlugin
    informer, err := w.cache.GetInformer(ctx, &v1alpha1.ExporterPlugin{})
    if err != nil {
        return err
    }
    // Register event handlers
    informer.AddEventHandler(toolscache.ResourceEventHandlerFuncs{
        AddFunc:    w.handleAdd,
        UpdateFunc: w.handleUpdate,
        DeleteFunc: w.handleDelete,
    })
    // Start cache (blocks until ctx is cancelled)
    return w.cache.Start(ctx)
}
```

This gives us: automatic list-watch with resourceVersion continuity, reconnect
with exponential backoff, resync on cache expiry — all handled by
controller-runtime's cache implementation. No manual watch loop needed.

**Startup wiring in `cmd/api-server/main.go`** (review C1):

Current code creates `k8sClient` inside a block-scoped `if k8sErr == nil`.
Required refactor:
1. Lift `k8sRestConfig` to outer scope (it's already `var` but assigned with `:=` inside the block)
2. Create `exporterRegistry` BEFORE the K8s client setup (move line 110-111 up)
3. After K8s client setup, if `k8sRestConfig != nil`:

```go
// After existing K8s client setup and AFTER exporterRegistry creation:
if k8sRestConfig != nil {
    watcher, err := k8sinfra.NewExporterPluginWatcher(k8sRestConfig, exporterRegistry, watchNamespace)
    if err != nil {
        log.Printf("warning: ExporterPlugin watcher setup failed: %v", err)
    } else {
        go func() {
            if err := watcher.Start(ctx); err != nil {
                log.Printf("warning: ExporterPlugin watcher stopped: %v", err)
            }
        }()
    }
} else {
    log.Println("ExporterPlugin CR watch disabled: no K8s client. Only built-in exporters available.")
}
```

4. Create a `ctx` with cancel tied to the existing signal handler / graceful shutdown

**Event handlers:**

`handleAdd(obj)`:
- Type-assert to `*v1alpha1.ExporterPlugin`
- Check `registry.IsBuiltIn(cr.Name)` → if collision, log warning `"ExporterPlugin CR '%s' skipped: name reserved by built-in exporter"` and skip registration. The API server has read-only RBAC for ExporterPlugin CRs, so it cannot update CR status. The user discovers the collision via `/exporters` not showing their webhook exporter.
- Otherwise: create `NewWebhookExporter(...)` from CR spec, read `cr.Status.Phase` to set initial health status (review R2#8), call `registry.RegisterWebhook(...)`

`handleUpdate(oldObj, newObj)`:
- **Atomic swap** (review M7): build the new `WebhookExporter` adapter first, validate it can be constructed. Then in a single write-locked operation: deregister old name, register new. If new construction fails, keep old registration.
- Read `cr.Status.Phase` and set health status on the new adapter (review R2#8)

`handleDelete(obj)`:
- `registry.Deregister(cr.Name)`

Conversion: `ExporterPluginParameterDef` (CRD type) → `export.ParameterDef` (export type).

Timeout semantics: CRD schema enforces `minimum: 3`, so values 1-2 are rejected by K8s admission. Omitted or 0 → default to 10 at application level. Values ≥3 → use as-is.

Tests:
- handleAdd registers webhook with correct baseURL, description, paramSchema, timeout, healthStatus
- handleDelete deregisters
- handleAdd with built-in name collision → not registered, built-in preserved (+ CR status set to Error if possible in test)
- handleUpdate replaces registration atomically (old gone, new present)
- handleUpdate with invalid spec → old registration preserved
- Default timeout (no spec.TimeoutSeconds) → 10s
- Custom timeout (spec.TimeoutSeconds=20) → export=20s, validate=10s
- Minimum timeout (spec.TimeoutSeconds=1) → clamped to 3
- Health status read from CR status.Phase on add/update

**Verify:** `go test ./internal/infrastructure/k8s/... -run TestExporterPluginWatcher -count=1`

**Depends on:** Steps 2, 4, 6

---

### Step 11: UI Updates (T-36.118–T-36.124)

Modify `ui/src/components/ExportBindingsPanel.tsx`:
- Update inline exporter type to include `source: string` and `health: string`
- In exporter list/dropdown: show source badge (`<Label color="blue">built-in</Label>` or `<Label color="cyan">webhook</Label>`)
- Add health dot next to exporter name: green=Ready, yellow=Unhealthy, red=Error, grey=Unknown
- Orphaned binding detection: if `binding.exporter_name` not in exporters list → warning icon + tooltip "Exporter not registered"
- Disable "Export Now" button for orphaned bindings

Modify `ui/src/api/client.ts`:
- Update `exporters.list` return type to include `source` and `health`

Browser tests:
- Exporter list with built-in shows "built-in" badge
- Exporter list with webhook shows "webhook" badge + health dot
- Orphaned binding shows warning icon
- Export Now disabled for orphaned binding

**Verify:** `cd ui && npx vitest run --config vitest.browser.config.ts src/components/ExportBindingsPanel.browser.test.tsx` + `npx tsc --noEmit`

**Depends on:** Steps 5, 9

---

## Dependency Graph

```
Step 1 (constants + param fix) ──┐
Step 2 (thread-safe registry) ───┤
Step 3 (protocol DTOs) ──────────┤──── Step 4 (WebhookExporter) ──── Step 5 (ExporterInfo)
                                 │                                        │
Step 6 (CRD types) ──────────────┤──── Step 7 (CRD YAML + RBAC)         │
       │                         │                                        │
       ├── Step 8 (health ctrl)  │                                        │
       │                         │                                        │
       └──────────────────────── Step 10 (informer watch) ────────────────┤
                                                                          │
Step 9 (orphaned bindings) ── Steps 1,2 ──────────────────────────────────┤
                                                                          │
Step 11 (UI) ─────────────────────────────────────────────────────────────┘
```

Parallelizable: Steps 1+2+3+6 can be done simultaneously.
Then: 4+7+8+9 can be done simultaneously.
Then: 5+10 (need 4+6).
Finally: 11.

## Verification — End to End

After all steps, deploy to Kind and run:

1. `go test ./internal/... -count=1` — all backend tests pass
2. `cd ui && npx vitest run --config vitest.browser.config.ts` — all browser tests pass
3. Apply CRD first: `kubectl apply -f deploy/k8s/operator/exporterplugin-crd.yaml` (review R2#13 — CRD must exist before operator starts)
4. `make deploy` — deploy to Kind cluster (operator + API server)
5. Deploy example webhook exporter service
6. `kubectl apply -f examples/webhook-exporter/exporterplugin.yaml` — create CR
7. Verify: `kubectl get exporterplugins` shows CR with status Ready
8. Verify: `curl localhost:30080/api/data/v1/exporters` shows webhook exporter with source="webhook", health="Ready"
9. Create binding via UI, run export, verify YAML output
10. Delete CR → verify exporter disappears from list, binding shows orphaned warning
11. `make test-live` — all live tests pass
12. `cd ui && npx vitest run --config vitest.system.config.ts` — all system tests pass

## Not Addressed (Accepted Limitations for Phase 2 MVP)

- **No retry for transient failures** (review R2#11): user can retry manually for single runs. For PublishPreview, a transient 503 marks the binding as failed. Acceptable for MVP.
- **EntityTypes field excluded from v1 contract** (review R2#2): webhook plugins don't get schema context in the export request. Accepted — plugins use ValidateSchema at binding creation to understand the schema. Can be added in v2.
- **No import path for orphaned bindings** (review R2#10): FF-12 export doesn't include bindings (TD-160), so there's nothing to import. Non-issue.
