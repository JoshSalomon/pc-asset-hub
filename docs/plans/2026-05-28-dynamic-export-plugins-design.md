# FF-15 Phase 2: Dynamic Export Plugins via HTTP Webhooks — Design Spec

**Date:** 2026-05-28 (revised 2026-06-03 per adversarial review)
**Branch:** `021-dynamic-export-plugins`
**Prerequisite:** FF-15 Phase 1 (merged, PR #23)
**Reviews:** `docs/plans/2026-06-03-dynamic-export-plugins-review.md`, `docs/plans/2026-06-03-dynamic-export-plugins-review-v2.md`

## Core Constraint

**Zero-recompilation extensibility.** Deploying a new exporter must not require any code changes to the Asset Hub. A plugin author deploys a Deployment+Service and creates an ExporterPlugin CR. The Asset Hub discovers it and makes it available for binding.

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Plugin transport | HTTP webhook (POST) | Any language, well-understood K8s pattern (admission webhooks), independent lifecycle |
| Registration | ExporterPlugin CRD (namespace-scoped) | Dynamic, K8s-native. API server watches CRs directly and registers WebhookExporter adapters in the registry. Operator handles health checks and status updates only. |
| Auth | Service account token with subject allowlist | API server sends its SA token in Authorization header for /validate and /export calls. Operator sends its SA token for /health probes. Plugin validates via TokenReview AND checks the token's subject matches an allowlist of Asset Hub service accounts (`system:serviceaccount:{namespace}:{api-sa}` and `system:serviceaccount:{namespace}:{operator-sa}`). Rejects tokens from other SAs even if valid. |
| Schema validation | /validate endpoint required | Plugin must expose POST /validate (called at binding creation and publish preview) |
| Artifact delivery | File download (same as Phase 1) | Plugin returns YAML artifacts. User downloads and applies manually. System-managed delivery deferred to a future phase |
| Timeout | 5s validate, 10s export (default) | Configurable per CR via `spec.timeoutSeconds` (minimum 3). Validate timeout = max(3, timeoutSeconds/2) rounded down. Export timeout = timeoutSeconds. |
| Health checking | Show when CR exists, mark status | Operator probes /health periodically, writes status to CR. API server reads CR status. Exporters with CR present always appear in list (even if unhealthy). Deregistered exporters (CR deleted) disappear from list. |
| MCP Gateway | Keep built-in + add webhook clone | Built-in stays for demo stability. Webhook clone proves the pattern. Removal criteria: webhook clone passes all existing MCP Gateway tests in live and system test suites. |
| Example plugin | MCP Gateway webhook clone | Standalone Go service implementing the same logic. Reference implementation for plugin developers. |
| CR scope | Namespace-scoped | Same namespace as Asset Hub. Matches CatalogVersion CR pattern. Webhook service endpoint can reference any namespace (cross-namespace calls are valid). |
| Protocol versioning | Version header on all requests | `X-AssetHub-Protocol-Version: v1` sent on all webhook calls. Plugin should reject unknown versions with 400. |
| Response size limit | 10 MB max | WebhookExporter adapter wraps response body with `io.LimitReader(resp.Body, 10*1024*1024+1)`, reads into buffer, and returns error if buffer exceeds 10 MB. Prevents runaway plugins. Error classified as `domainerrors.NewValidation("response exceeds 10 MB limit")`. |
| Parameter type validation | Use declared `ParameterDef.Type` field | Fix existing `_type` suffix heuristic in `validateParamEntityTypes()` to use the declared `type: entity_type` field instead. Webhook and built-in exporters validated consistently. |
| CRD naming | ExporterPlugin (not CatalogExporter) | Avoids confusion with ExportBinding. "ExporterPlugin" clearly means a plugin that exports, not an exporter scoped to a catalog. |

## Architecture

### Component Diagram

```
    ExporterPlugin CR                 ExporterPlugin CR
    (status.phase written             (spec read by
     by operator)                      API server)
         ^                                 |
         |                                 | watches (informer)
         |                                 v
+------------------+              +-------------------+
|   Operator       |              | API Server        |
| (health probes,  |              | (ExporterRegistry |
|  status updates) |              |  in-memory)       |
+------------------+              +---+---------------+
                                      |
                                      | WebhookExporter adapter
                                      |
    +---------------------------------+----------------------------+
    |                                 |                            |
    v                                 v                            v
Built-in: mcp-gateway          Webhook: mcp-gw-webhook      Webhook: custom
(compiled Go)                  (HTTP POST to Service)        (HTTP POST)
```

### Registry Synchronization (fix for review A1)

The operator and API server are separate pods and cannot share in-memory state. The synchronization mechanism:

1. **API server watches ExporterPlugin CRs directly** via a K8s informer. This is a new pattern for the API server — it currently uses service-layer DB queries, not informers. The informer is justified here because CR changes must propagate to the in-memory registry with minimal latency. On CR create/update, the API server registers/updates a `WebhookExporter` adapter. On CR delete, it deregisters.
2. **Operator handles health probes only.** The operator's CatalogExporter controller periodically calls `GET /health` on the plugin service and writes `status.phase` (Ready/Unhealthy/Error) to the CR. The API server reads health status from the CR via its informer.
3. **On startup**, the API server lists existing ExporterPlugin CRs and pre-populates the registry. The informer watch handles subsequent changes.

This means the operator does NOT register exporters — it only updates health status on the CR. The API server is the sole owner of the in-memory registry.

### WebhookExporter Adapter

A `WebhookExporter` implements the existing `Exporter` interface by forwarding calls via HTTP:

```go
type WebhookExporter struct {
    name           string
    description    string
    baseURL        string           // e.g., https://my-exporter.ns.svc.cluster.local (NO path)
    paramSchema    []ParameterDef
    validateTimeout time.Duration
    exportTimeout   time.Duration
    healthStatus   string           // "Ready", "Unhealthy", "Error" — read from CR status
    httpClient     *http.Client
}

// Name, Description, ParameterSchema — return stored metadata from CR
// ValidateSchema — POST {baseURL}/validate with request body
// Export — POST {baseURL}/export with ExportInput body
```

`baseURL` is the service URL **without any path**. The adapter appends `/validate`, `/export` as needed. Health probes go to `{baseURL}/health` (done by operator, not adapter).

### Dynamic Registry (fix for review C1)

The current `ExporterRegistry` has **no mutex** — it's a bare `map[string]Exporter` that works only because all writes happen during single-threaded startup. Phase 2 adds concurrent writes (CR watch callbacks) and requires full thread-safety:

```go
type ExporterRegistry struct {
    mu        sync.RWMutex
    exporters map[string]Exporter
    builtIn   map[string]bool       // tracks which are built-in (cannot be deregistered via CR)
}
```

All existing methods (`Register`, `Get`, `List`) must be wrapped with `mu.RLock`/`mu.RUnlock`. New methods:
- `RegisterWebhook(e Exporter)` — adds with write lock, marks as non-built-in
- `Deregister(name string) bool` — removes if not built-in, returns false if built-in or not found
- `IsBuiltIn(name string) bool`
- `GetHealthStatus(name string) string` — returns health from `WebhookExporter.healthStatus`

### Name Collision Policy

Built-in exporter names are reserved. If an ExporterPlugin CR uses a name that matches a built-in exporter (e.g., `mcp-gateway`):
- The API server's informer watch detects the collision via `registry.IsBuiltIn(name)`
- The CR is **not** registered in the registry
- The API server logs a warning: `"ExporterPlugin CR 'mcp-gateway' skipped: name reserved by built-in exporter"`
- The exporter list continues to show the built-in version
- The user discovers the collision via `/exporters` not showing their webhook exporter (the CR status is not updated — the API server has read-only RBAC for ExporterPlugin CRs to avoid requiring write permissions for this rare edge case)

### ExporterPlugin CRD (renamed from CatalogExporter per review N1)

```yaml
apiVersion: assethub.project-catalyst.io/v1alpha1
kind: ExporterPlugin
metadata:
  name: my-custom-exporter
  namespace: assethub
spec:
  description: "Exports catalog data as ConfigMaps"
  endpoint: https://my-exporter.my-namespace.svc.cluster.local
  timeoutSeconds: 15        # optional, default 10, minimum 3
  parameterSchema:
    - name: target_namespace
      type: string
      description: "K8s namespace for output"
      required: true
    - name: server_type
      type: entity_type
      description: "Entity type for servers"
      required: true
status:
  phase: Ready              # Ready, Unhealthy, Error, Unknown
  lastHealthCheck: "2026-05-28T12:00:00Z"
  message: ""
```

Note: `spec.endpoint` is the **base URL** (no path). The adapter appends `/validate`, `/export`. The operator appends `/health`.

**Endpoint trust policy (MVP):** The endpoint must be a cluster-internal service URL (`*.svc.cluster.local` or `*.svc`). The watcher rejects endpoints pointing to external hosts to prevent SA token exfiltration. This can be relaxed in a future version with audience-scoped tokens.

### Auth Contract

The Asset Hub sends a `Bearer` token in the `Authorization` header on every webhook call. Plugin services **must**:

1. Validate the token via K8s `TokenReview` API (confirms token is valid)
2. Check the authenticated subject against an allowlist of Asset Hub service accounts

The ExporterPlugin CR includes the expected SA subjects so the plugin knows who to trust:

```yaml
spec:
  trustedSubjects:
    - system:serviceaccount:assethub:assethub-api-server
    - system:serviceaccount:assethub:assethub-operator
```

If `trustedSubjects` is omitted, the plugin should default to accepting the two SAs from the ExporterPlugin CR's own namespace: `{namespace}:assethub-api` and `{namespace}:assethub-operator`.

The example webhook plugin will include a middleware that performs this validation, serving as a reference for plugin authors.

### Health Status State Machine

| Phase | Meaning | Transition IN | Transition OUT |
|-------|---------|---------------|----------------|
| `Unknown` | Initial state, or probes stale (no successful probe for 3x probe interval) | CR created; probe not run yet; 3x interval without any probe result | → `Ready` on 200; → `Unhealthy` on non-200/timeout; → `Error` on connection failure |
| `Ready` | Plugin is healthy (last probe returned 200) | Successful /health probe (200 OK) | → `Unhealthy` on non-200/timeout; → `Error` on connection failure; → `Unknown` if stale |
| `Unhealthy` | Plugin responded but not healthy (non-200 status or timeout, excluding auth errors) | /health returned non-200 (except 401/403), or request timed out | → `Ready` on 200; → `Error` on connection failure or auth rejection; → `Unknown` if stale |
| (auth rejection) | 401/403 from /health is classified as `Error`, not `Unhealthy` — it indicates misconfiguration (wrong SA, missing TrustedSubjects), not a transient issue | /health returned 401 or 403 | → `Error` with message "health probe authentication rejected" |
| `Error` | Plugin unreachable (DNS failure, TLS error, connection refused) | TCP/TLS/DNS failure on /health probe | → `Ready` on 200; → `Unhealthy` on non-200/timeout; → `Unknown` if stale |

**Probe interval:** 30 seconds (configurable via operator flag).
**Stale threshold:** 3x probe interval (90s default). If no probe result is recorded for this duration (e.g., operator pod restarted), status reverts to `Unknown`.

The API server reads `status.phase` from the CR via its informer. It does NOT run health probes itself.

### Webhook Protocol

All requests include: `X-AssetHub-Protocol-Version: v1`

**POST {baseURL}/validate** — called at binding creation and publish preview

Request:
```json
{
  "parameters": {"server_type": "mcp-server", "target_namespace": "prod"},
  "schema": {
    "entity_types": [
      {
        "name": "mcp-server",
        "attributes": ["endpoint", "route_name", "mcp_path"],
        "associations": [
          {
            "name": "tools",
            "type": "containment",
            "target_entity_type": "mcp-tool"
          }
        ]
      },
      {
        "name": "mcp-tool",
        "attributes": ["type", "idempotent"],
        "associations": []
      }
    ]
  }
}
```

Response (200 OK = valid):
```json
{"valid": true}
```

Response (400 = validation error):
```json
{"valid": false, "error": "Entity type 'mcp-server' missing required attribute 'endpoint'"}
```

**POST {baseURL}/export** — called when a binding runs

Request body is the webhook export contract — a clean JSON representation of catalog data, NOT the raw Go `ExportInput` struct. Dead/internal fields from the Go struct are excluded:

```json
{
  "catalog_name": "prod-agents",
  "catalog_description": "Production AI agents",
  "parameters": {"server_type": "mcp-server", "tool_type": "mcp-tool", "target_namespace": "prod"},
  "instances_by_type": {
    "mcp-server": [
      {
        "id": "019e...",
        "name": "github",
        "description": "GitHub MCP server",
        "attributes": {"endpoint": "https://github-mcp.example.com", "route_name": "route-gh"},
        "parent_id": ""
      }
    ],
    "mcp-tool": [
      {
        "id": "019f...",
        "name": "create-pr",
        "description": "Create a pull request",
        "attributes": {"type": "write"},
        "parent_id": "019e..."
      }
    ]
  },
  "children_of": {
    "019e...": ["019f..."]
  }
}
```

Fields deliberately excluded from the v1 webhook contract (fix for review C2, H4):
- `cv_label` — dead field, never populated in Phase 1
- `entity_types` — dead field, never populated in `buildExportInput`
- `virtual_server_instance_name` / `allowed_tool_ids` — MCP-specific internal filtering, applied before the data reaches the exporter
- `links_by_assoc` — always empty in Phase 1 (`buildInstancesByType` initializes but never populates it). Excluded from v1 contract. Will be added in protocol v2 when the builder is fixed to populate link data.

**Attribute value types in v1 contract:** All attribute values are serialized as JSON native types: strings as `"string"`, numbers as `123` or `3.14`, booleans as `true`/`false`, dates as ISO 8601 strings, lists as JSON arrays, JSON values as parsed JSON. The `attributes` map value type is `any` (JSON value).

Response (200 OK):
```json
{
  "artifacts": [
    {
      "api_version": "gateway.assethub.io/v1alpha1",
      "kind": "MCPServerRegistration",
      "name": "github",
      "namespace": "mcp-gateway",
      "yaml": "apiVersion: gateway.assethub.io/v1alpha1\nkind: MCPServerRegistration\n..."
    }
  ],
  "warnings": ["Server 'test-server' has no tools"]
}
```

Response (4xx = input/validation error):
```json
{"error": "Entity type 'mcp-server' has no instances"}
```

Response (5xx = internal plugin error):
```json
{"error": "Database connection failed"}
```

The WebhookExporter adapter maps HTTP responses to Go errors:

| HTTP Status | Adapter Behavior |
|-------------|-----------------|
| 200 + valid JSON | Success — parse artifacts and warnings |
| 200 + malformed JSON | `BindingStatusFailed`, error: `"plugin returned 200 but response is not valid JSON: {parse error}"` |
| 200 + oversized response | `domainerrors.NewValidation("response exceeds 10 MB limit")` |
| 400-499 | `domainerrors.NewValidation(body.error)` — retryable by fixing input |
| 500-599 | `fmt.Errorf("webhook plugin error: %s", body.error)` — retry later |
| Timeout | `fmt.Errorf("webhook plugin timed out after %s", timeout)` |
| Connection refused / DNS failure | `fmt.Errorf("webhook plugin unreachable: %s", err)` |

Note: All JSON field names use **snake_case** in the webhook protocol. The Go-side DTO types for webhook serialization will have explicit `json:"..."` tags (fix for review C3). These are separate from the internal Go types which use PascalCase.

**GET {baseURL}/health** — probed periodically by the operator

Response (200 OK):
```json
{"status": "ok"}
```

Non-200 or timeout → operator sets `status.phase: Unhealthy` on the CR.

### Operator Reconciliation

The operator has a **new independent controller** (`ExporterPluginReconciler`) separate from `AssetHubReconciler` (fix for review A2). It does NOT use AssetHub owner references — ExporterPlugin CRs are independently deployed by plugin authors.

The controller handles:
1. **Health probes** — periodically calls `GET {baseURL}/health` and updates `status.phase`
2. **Status updates** — writes `status.lastHealthCheck` and `status.message`

The controller does NOT register/deregister exporters in the API server's registry — that's done by the API server's own informer watch (see Registry Synchronization above). This separation means the operator can be down without affecting export functionality — the API server watches CRs independently (fix for review A3).

### Orphaned Bindings

When a webhook exporter is deregistered (CR deleted):
- The exporter **disappears from the list** — `List()` no longer includes it (fix for review E3: "always show" means "even when unhealthy," not "even when deleted")
- `ListBindings` still returns bindings referencing the deleted exporter (with exporter name)
- `RunBinding` returns error: "exporter 'X' is not registered"
- `RunAll` and `PublishPreview` **skip** orphaned bindings with status `"skipped"` and reason `"exporter not registered"`. Skipped bindings do NOT contribute to `has_failures` in the publish preview response — they are a separate category. Only enabled bindings with registered exporters participate.
- New domain constant: `BindingStatusSkipped = "skipped"` added alongside existing `never`/`success`/`failed`. Skipped status appears in `BindingRunResult.Status` and in the publish preview per-binding results. It does NOT persist to `last_run_status` on the binding itself (last_run_status reflects the last actual run, not a skip).
- UI shows warning icon on orphaned bindings
- User can delete orphaned bindings manually

### Import/Export Interaction

FF-12 catalog export does **not** include export bindings (see TD-160). When migrating a catalog to a different system, export bindings must be manually recreated. This applies to both built-in and webhook exporters. Adding bindings to the FF-12 format is deferred to a future phase.

### Parameter Type Validation (fix for review E4)

The existing `validateParamEntityTypes()` uses a `strings.HasSuffix(key, "_type")` heuristic to detect entity type parameters. This is fragile — a parameter named `output_type` of type `string` would incorrectly get entity-type validation.

Fix: change `validateParamEntityTypes()` to iterate `exporter.ParameterSchema()` and check `param.Type == "entity_type"` instead of checking the key name suffix. Both built-in and webhook exporters validated consistently.

### Non-K8s Runtime (Local Development)

When the API server runs outside a cluster (local dev with SQLite, no K8s client), ExporterPlugin CR discovery is disabled:
- The informer watch is not started (K8s client is nil)
- Only built-in exporters are available in the registry
- A warning is logged at startup: `"ExporterPlugin CR watch disabled: no K8s client available. Only built-in exporters will be registered."`
- The `/exporters` endpoint returns only built-in exporters
- No health probes run (operator is not running)

This matches the existing pattern where `K8sCRManager` is nil in non-cluster mode — CR operations are silently skipped.

### RBAC

No changes to existing RBAC model:
- ExporterPlugin CRs managed by cluster admins (standard K8s RBAC)
- Export bindings managed by Asset Hub Admin+ (existing)
- Export runs by RW+ (existing)
- Exporter list visible to all authenticated users (existing)

### ListExporters Response (fix for review A3/D3)

The `/exporters` endpoint response is extended to include health and source information:

```json
{
  "items": [
    {
      "name": "mcp-gateway",
      "description": "Exports MCP server/tool instances as MCP Gateway CRs",
      "parameter_schema": [...],
      "source": "built-in",
      "health": "n/a"
    },
    {
      "name": "my-webhook-exporter",
      "description": "Custom exporter",
      "parameter_schema": [...],
      "source": "webhook",
      "health": "Ready"
    }
  ]
}
```

`source`: `"built-in"` or `"webhook"`. `health`: `"Ready"`, `"Unhealthy"`, `"Error"`, `"Unknown"`, or `"n/a"` (for built-in exporters that don't have health probes).

## Implementation Stages

### Stage 1: WebhookExporter Adapter + Protocol (~3 hours)
- Define webhook protocol DTO types with explicit `json:"snake_case"` tags
- Implement `WebhookExporter` struct implementing `Exporter` interface
- HTTP client with SA token auth, configurable timeout, 10MB response limit
- Protocol version header on all requests
- Unit tests with httptest mock server

### Stage 2: Dynamic Registry + ExporterPlugin CRD (~3 hours)
- Add `sync.RWMutex` to `ExporterRegistry`, guard all methods
- Add `RegisterWebhook`, `Deregister`, `IsBuiltIn`, `GetHealthStatus`
- Define `ExporterPlugin` CRD types in `operator/api/v1alpha1/`
- Add operator `ExporterPluginReconciler` (new independent controller) for health probes + status
- Fix `validateParamEntityTypes` to use declared type instead of name suffix

### Stage 3: API Server Integration (~2 hours)
- Add K8s informer watch for ExporterPlugin CRs in API server startup
- On CR create/update: register/update WebhookExporter in registry; reject name collisions with built-in exporters
- On CR delete: deregister from registry
- Extend `/exporters` response with `source` and `health` fields
- Handle orphaned bindings (skip in RunAll/PublishPreview with `skipped` status)
- Non-K8s mode: log warning, skip informer setup

### Stage 4: Example Webhook Plugin — MCP Gateway Clone (~3 hours)
- Standalone Go service implementing /validate, /export, /health
- Same logic as built-in MCPGatewayExporter but as HTTP service
- Dockerfile + K8s manifests (Deployment + Service)
- ExporterPlugin CR manifest

### Stage 5: UI Updates (~2 hours)
- Exporter list shows health status indicator (green/yellow/red dot)
- Exporter list shows source badge (built-in / webhook)
- Orphaned binding warning icon
- No other UI changes needed (webhook exporters use the same binding flow)

### Stage 6: Live Tests + Documentation (~2 hours)
- Deploy example webhook plugin to Kind cluster
- Create ExporterPlugin CR, verify API server registers it
- Create binding, run export, verify YAML output
- Live test script: `scripts/test-webhook-exporters.sh`
- System tests: Playwright tests for health indicator + orphaned warning

## Cross-Feature Interactions

| Feature | Interaction |
|---------|-------------|
| Publishing | Webhook exporters participate in publish preview (same as built-in). Orphaned bindings skipped with "skipped" status. |
| RBAC | No change — webhook exporters use the same binding permissions |
| Import/Export | FF-12 does not include export bindings (TD-160). Bindings must be manually recreated on target system. |
| Catalog delete | Cascade deletes bindings (same as built-in) |
| K8s operator | New ExporterPluginReconciler (independent controller, no AssetHub owner refs). Health probes + status only. |

## Files to Create/Modify

**New files:**
- `internal/service/operational/export/webhook_exporter.go` — WebhookExporter adapter
- `internal/service/operational/export/webhook_protocol.go` — Webhook DTO types with explicit `json:"snake_case"` tags
- `internal/operator/api/v1alpha1/exporterplugin_types.go` — CRD Go types + DeepCopy
- `internal/operator/controllers/exporterplugin_controller.go` — Health probe reconciler (independent controller)
- `deploy/k8s/operator/exporterplugin-crd.yaml` — CRD manifest
- `deploy/k8s/operator/role.yaml` — Updated: add ExporterPlugin RBAC rules (get, list, watch, update/status)
- `examples/webhook-exporter/` — Example MCP Gateway webhook service (main.go, Dockerfile, K8s manifests, ExporterPlugin CR)
- `scripts/test-webhook-exporters.sh` — Live test script

**Modified files:**
- `internal/operator/api/v1alpha1/assethub_types.go` — Add ExporterPlugin to `addKnownTypes` / `SchemeBuilder`
- `internal/service/operational/export/registry.go` — Add `sync.RWMutex`, Deregister, IsBuiltIn, GetHealthStatus
- `internal/service/operational/export/binding_service.go` — Fix `validateParamEntityTypes` to use declared `param.Type`; skip orphaned bindings in RunAll; handle malformed plugin responses
- `internal/api/operational/export_binding_handler.go` — Health + source in exporter list response
- `cmd/api-server/main.go` — ExporterPlugin CR informer watch, registry sync on startup
- `cmd/operator/main.go` — Register ExporterPluginReconciler with manager
- `ui/src/components/ExportBindingsPanel.tsx` — Health indicator, source badge, orphaned warning
