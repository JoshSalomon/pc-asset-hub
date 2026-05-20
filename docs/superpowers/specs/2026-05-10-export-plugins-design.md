# Catalog Export Plugins — Design Specification

## Overview

Extensible export system that produces consumer-specific output formats from catalog data. Exporters are registered plugins that read catalog data and produce output artifacts (K8s CRs, ConfigMaps, YAML files, etc.) in the consumer's expected format.

**Phase 1 (PoC):** Build the plugin framework with one built-in MCP Gateway CR exporter. The exporter produces CR YAML that the user applies manually to an OCP cluster. No automatic K8s deployment in Phase 1.

**Future phases:** Automatic CR deployment, webhook-based dynamic plugins, multi-cluster support.

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Plugin mechanism (Phase 1) | Compiled-in Go plugins | Simple, fast, type-safe, works in dev/SQLite mode. Phase 2 adds HTTP webhook adapter. |
| Registration (Phase 1) | Code-registered (static) | Zero infrastructure. Exporters register in main.go. Phase 2 adds CRD-based registration. |
| Trigger model | On publish + on demand | Auto-export on publish for all bound exporters. Manual trigger per binding for testing. |
| Output delivery (Phase 1) | Return artifacts as API response (download) | User applies CRs manually. Phase 2 adds system-managed K8s delivery. |
| Configuration model | Export bindings (sub-resource of catalog) | Each catalog has zero or more bindings to registered exporters with per-binding parameters. |
| Entity type mapping | Parameter-based | User configures which entity types map to which roles in the export config. No naming conventions. |
| CR schema source | User-provided CRD YAML | MCP Gateway CRD comes from a working system. Exporter maps catalog data to CR spec fields. |
| RBAC | Admin+ for binding management. RW+ for manual export trigger. Any role for exporter list. | Same as catalog publish permissions. |
| Namespace targeting (Phase 1) | Per-binding parameter, not enforced | Binding stores target_namespace as metadata. Phase 1 just includes it in CR metadata. |
| Parameter validation | At binding creation AND publish preview (two levels) | **Level 1:** Required parameters (e.g., server_type, tool_type) validated against catalog's CV pins — entity types must exist. **Level 2:** Exporter declares required attributes per entity type via `ValidateSchema()`. The MCP Gateway exporter checks that the server type has `route_name` attribute and containment association to tool type. If the schema is missing required attributes, binding creation fails with a clear error listing what's missing. **Schema drift protection:** `ValidateSchema()` is re-run during publish preview to catch CV changes since binding creation (e.g., entity type removed, required attribute deleted). Failed validation produces a clear message: "binding 'X' is invalid: entity type 'mcp-server' no longer has attribute 'route_name'". |
| Multiple bindings per exporter | Allowed | A catalog can have multiple bindings to the same exporter with different parameters (e.g., different target namespaces). |
| Tool prefix source | Derived from server instance name | Exporter uses `server_instance.name + "_"` automatically. No attribute needed — instance names are unique within catalog and DNS-1123 compliant. |
| HTTPRoute handling | Reference only, not generated | MCPServerRegistration references existing HTTPRoutes by name via a `route_name` attribute on the mcp-server instance. Exporter does not generate networking resources. |
| Download format | Multi-document YAML | Single .yaml file with `---` separators. Standard `kubectl apply -f` format. |
| Publish backward compat | POST /publish works without token | Preview is optional. If session token provided, uses cached artifacts. If not, exports run as a fire-and-forget goroutine (30s timeout, errors logged on bindings, not returned to caller). Fully backward compatible. |
| Exporter list access | Any authenticated user | GET /exporters is metadata only, no security risk. |
| Incomplete server handling | Fail entire export (Phase 1) | If any server instance is missing required attributes, the export fails. Future phases may add skip-with-warning for flexibility. |
| Empty catalog export | Success with comment-only file | If the catalog has no instances of the server/tool types, the export succeeds and returns a YAML file containing only a comment: `# No instances found for export — catalog '{name}' has no {server_type} instances`. Not an error — the file is obviously intentional, not a broken download. |
| Prefix convention | Derived from server instance name | No `prefix` attribute needed. The exporter uses `instance.name + "_"` as the prefix automatically. Guarantees uniqueness (instance names are unique within catalog) and requires no user maintenance. |
| Catalog delete with bindings | CASCADE delete with warning | Bindings are deleted when catalog is deleted. UI shows confirmation listing affected bindings. API delete response includes `deleted_bindings_count` so programmatic consumers are aware of the cascade. |
| Run history | Last run only (Phase 1) | Binding stores only the most recent run status. No history — a successful manual `/run` overwrites a failed publish attempt. Known limitation. Future: `export_runs` table with full history per binding. |

## Architecture

### Plugin Interface

```go
type Exporter interface {
    Name() string
    Description() string
    ParameterSchema() []ParameterDef
    ValidateSchema(params map[string]string, schema SchemaInfo) error
    Export(ctx context.Context, input ExportInput) (*ExportOutput, error)
}

// SchemaInfo provides the catalog's CV schema for validation at binding creation time.
// The exporter checks that required entity types, attributes, and associations exist.
type SchemaInfo struct {
    EntityTypes []SchemaEntityType // all entity types pinned in the CV
}

type SchemaEntityType struct {
    Name         string
    Attributes   []string            // attribute names on the pinned version
    Associations []SchemaAssociation // associations on the pinned version
}

type SchemaAssociation struct {
    Name             string // association name
    Type             string // "containment", "directional", "bidirectional"
    TargetEntityType string // target entity type name
}

type ParameterDef struct {
    Name        string `json:"name"`
    Type        string `json:"type"`        // "string", "boolean", "integer"
    Description string `json:"description"`
    Required    bool   `json:"required"`
    Default     string `json:"default,omitempty"`
}

// ExportInput is a structured, pre-indexed view of the catalog data tailored
// for exporters. Built by the framework from the raw catalog data — exporters
// do not need to walk containment trees or filter by entity type.
type ExportInput struct {
    CatalogName    string                          // catalog name
    CatalogDesc    string                          // catalog description
    CVLabel        string                          // catalog version label
    Parameters     map[string]string               // user-configured values from the binding
    EntityTypes    []ExportEntityType              // schema: attributes + associations per type
    InstancesByType map[string][]*ExportInstance    // entity type name → instances of that type
    ChildrenOf      map[string][]*ExportInstance    // parent instance ID → direct children
}

// ExportInstance represents a single entity instance with its attribute values
// and links grouped by association name.
type ExportInstance struct {
    ID          string                        // instance ID
    EntityType  string                        // entity type name
    Name        string                        // instance name (DNS-1123 compliant)
    Description string
    ParentID    string                        // parent instance ID (empty if root)
    Attributes  map[string]any                // attribute name → value
    LinksByAssoc map[string][]ExportLink      // association name → links for that association
}

type ExportOutput struct {
    Artifacts []K8sArtifact
    Warnings  []string // non-fatal issues (e.g., "instance X has no endpoint attribute, skipped")
}

// K8sArtifact represents a single Kubernetes resource produced by an exporter.
// Phase 1 is K8s-specific by design. When non-K8s exporters are added
// (e.g., JSON, XML, env-file), generalize to an Artifact interface with
// Content() []byte and ContentType() string methods. Each exporter declares
// its output format; the download endpoint sets Content-Type accordingly.
type K8sArtifact struct {
    APIVersion string // e.g., "mcp.kuadrant.io/v1alpha1"
    Kind       string // e.g., "MCPServerRegistration", "MCPVirtualServer"
    Name       string // metadata.name (DNS-1123 compliant)
    Namespace  string // metadata.namespace (from binding parameters)
    YAML       string // rendered YAML content, ready to apply
}
```

### Exporter Registry

```go
type ExporterRegistry struct {
    exporters map[string]Exporter // name → exporter
}

func NewExporterRegistry() *ExporterRegistry
func (r *ExporterRegistry) Register(e Exporter)
func (r *ExporterRegistry) Get(name string) (Exporter, bool)
func (r *ExporterRegistry) List() []ExporterInfo // name, description, parameter schema
```

Wiring in main.go:
```go
registry := NewExporterRegistry()
registry.Register(NewMCPGatewayExporter())
// Future: registry.Register(NewConfigMapExporter())
```

### Export Binding Model

```go
type ExportBinding struct {
    ID           string
    CatalogID    string
    ExporterName string
    Parameters   map[string]string // JSON-serialized per-binding config
    Enabled      bool
    LastRunAt    *time.Time
    LastRunStatus string           // "success", "failed", "never"
    LastRunError  string           // error message if failed
    CreatedAt    time.Time
    UpdatedAt    time.Time
}
```

### Export Binding Service

```go
type ExportBindingService struct {
    bindingRepo ExportBindingRepository
    registry    *ExporterRegistry
    exportSvc   *ExportService
    catalogRepo CatalogRepository
}

func (s *ExportBindingService) Create(ctx, catalogName, exporterName, params) (*ExportBinding, error)
func (s *ExportBindingService) List(ctx, catalogName) ([]*ExportBinding, error)
func (s *ExportBindingService) Get(ctx, catalogName, bindingID) (*ExportBinding, error)
func (s *ExportBindingService) Update(ctx, catalogName, bindingID, params, enabled) (*ExportBinding, error)
func (s *ExportBindingService) Delete(ctx, catalogName, bindingID) error
func (s *ExportBindingService) Run(ctx, catalogName, bindingID) (*ExportOutput, error)
func (s *ExportBindingService) RunAll(ctx, catalogName) ([]BindingRunResult, error)
```

### API Endpoints

```
GET    /api/data/v1/exporters                                    — list registered exporters (name, description, params)

GET    /api/data/v1/catalogs/{name}/export-bindings              — list bindings for catalog
POST   /api/data/v1/catalogs/{name}/export-bindings              — create binding (Admin+)
GET    /api/data/v1/catalogs/{name}/export-bindings/{id}         — get binding
PUT    /api/data/v1/catalogs/{name}/export-bindings/{id}         — update binding (Admin+)
DELETE /api/data/v1/catalogs/{name}/export-bindings/{id}         — delete binding (Admin+)
POST   /api/data/v1/catalogs/{name}/export-bindings/{id}/run     — trigger export manually, returns downloadable YAML (RW+, any catalog state)
GET    /api/data/v1/catalogs/{name}/export-bindings/download?token={token} — download combined artifacts from publish preview (RW+)
```

**Middleware stacks per endpoint:**

The existing codebase uses three middleware functions for access control: `RequireCatalogAccess` (catalog-scoped RBAC), `requireRW` (blocks RO role), and `requireAdmin` (blocks RO and RW roles). Published catalog write protection is enforced by `RequireWriteAccess` (blocks non-SuperAdmin on published catalogs). The `/exporters` endpoint is not catalog-scoped and has no middleware.

| Endpoint | Middleware | Who can access |
|----------|-----------|----------------|
| `GET /exporters` | none | Any authenticated user |
| `GET .../export-bindings` | `RequireCatalogAccess` | Any role with catalog access |
| `GET .../export-bindings/{id}` | `RequireCatalogAccess` | Any role with catalog access |
| `POST .../export-bindings` | `RequireCatalogAccess` + `requireAdmin` | Admin+ |
| `PUT .../export-bindings/{id}` | `RequireCatalogAccess` + `requireAdmin` | Admin+ |
| `DELETE .../export-bindings/{id}` | `RequireCatalogAccess` + `requireAdmin` | Admin+ |
| `POST .../export-bindings/{id}/run` | `RequireCatalogAccess` + `requireRW` | RW+ (any catalog state — download only) |
| `GET .../export-bindings/download?token=` | `RequireCatalogAccess` + `requireRW` | RW+ (download cached artifacts after publish) |
| `POST .../publish/preview` | `RequireCatalogAccess` + `requireAdmin` | Admin+ (same guards as existing publish) |

Notes:
- Binding list/get is open to any role with catalog access (including RO) so the operational UI can show bindings in read-only mode.
- The `/run` endpoint uses `requireRW` not `requireAdmin` because it's a non-destructive download operation. It does NOT use `RequireWriteAccess` because it works on any catalog state (draft/valid/published) — it's a debug/preview tool.
- `publish/preview` uses the same middleware as the existing `publish` endpoint.
- Route registration pattern: `/exporters` is registered on the top-level data API group. All `/export-bindings` routes are sub-routes of the catalog group (inheriting `RequireCatalogAccess`).

**Run endpoint response:**

The `/run` endpoint returns the artifacts directly as a downloadable multi-document YAML file with `Content-Disposition: attachment` header (same pattern as the existing catalog export endpoint `GET /catalogs/{name}/export`). No separate metadata or download endpoint — keep it simple for Phase 1.

### Publish Integration

When a catalog is published, all enabled export bindings are triggered automatically:

**API flow — two-step publish when bindings exist:**

```
Step 1: POST /catalogs/{name}/publish/preview
  → Validates catalog is publishable (valid status, not already published)
  → Runs all enabled export bindings — produces artifacts
  → Validates delivery permissions (RBAC for CR deploy, ConfigMap write, etc.)
  → Returns export results + pre-computed artifacts (kept server-side)
  → Does NOT commit publish state

Step 2: POST /catalogs/{name}/publish
  → Commits publish state to DB + creates Catalog CR
  → Deploys pre-computed artifacts from step 1 (no DB re-read, no re-export)
  → Updates binding run statuses
```

**Key insight:** Step 1 does all the heavy lifting — reads catalog data, runs exporters, produces artifacts, and validates delivery permissions. Step 2 only commits state and deploys the already-computed artifacts. This eliminates the window between "dry run succeeded" and "actual run failed due to transient DB/permission issue."

**What step 1 validates:**
- Catalog is in `valid` state and not already published
- All enabled export bindings produce artifacts successfully
- Delivery permissions are valid (future Phase 2: K8s RBAC for target namespace, ConfigMap write access, etc.)

**What step 2 does:**
- Commits publish state to DB
- Creates Catalog CR (existing logic)
- Deploys pre-computed artifacts from step 1 (method depends on binding config — see below)
- Updates each binding's LastRunAt/LastRunStatus/LastRunError

**Deployment methods** (per binding, determined by exporter type or binding parameter):
- **Download to local machine (Phase 1/PoC):** Artifacts are returned as a downloadable file (multi-document YAML). The UI triggers a browser Save As dialog. This is the primary method for the PoC — user downloads the CR YAML and applies it manually to OCP.
- **Apply to K8s cluster (Phase 2+):** System applies artifacts directly via K8s API client. See F1.
- **Push to Git repo (Phase 2+):** System commits artifacts to a configured Git repository. See F3.

For Phase 1, all bindings use the download method. The `/publish/preview` step 1 produces the artifacts; step 2 commits the publish and makes the artifacts available for download. The artifacts can also be downloaded independently of publish via the manual run endpoint (`POST /export-bindings/{id}/run`).

The UI calls step 1 first. If all exports succeed, it immediately calls step 2. If any fail, it shows a confirmation dialog with the failures and "Publish Anyway" / "Abort" buttons. "Publish Anyway" calls step 2. "Abort" does nothing — catalog stays unpublished.

If there are no export bindings, step 1 returns an empty result and the UI proceeds directly to step 2 (same as today's publish flow).

**Artifact storage between steps:** Pre-computed artifacts are stored via a `PreviewCache` interface, keyed by a publish-session token (UUID). The token is returned in step 1 and passed to step 2. Artifacts expire after a configurable TTL (system-level setting, default 5 minutes) to prevent stale deploys. The TTL is configured via the `PUBLISH_PREVIEW_TTL` environment variable or the AssetHub CR spec.

```go
type PreviewCache interface {
    Store(token string, entry PreviewCacheEntry, ttl time.Duration) error
    Retrieve(token string) (*PreviewCacheEntry, error)  // returns error if expired/missing
    Delete(token string)
}

type PreviewCacheEntry struct {
    Artifacts        []K8sArtifact
    CatalogUpdatedAt time.Time  // catalog's updated_at at preview time — used to detect modifications between preview and commit
}
```

**Phase 1 (Kind/PoC):** `InMemoryPreviewCache` using `sync.Map` with TTL-based expiry. Sufficient for single-replica deployment. Proves the preview → token → commit mechanism works.

**OCP deployment:** Swap to `RedisPreviewCache` (or OCP-native cache such as Red Hat Data Grid / Infinispan) for reliability across pod restarts, failovers, and horizontal scaling. The `PreviewCache` interface stays the same — only the storage backend changes. No modifications to the publish flow code.

```go
func (s *CatalogService) PublishPreview(ctx, catalogName) (*PublishPreview, error) {
    // Validate publishable state
    // For each enabled binding: re-run ValidateSchema() against current CV
    //   (catches schema drift since binding creation)
    // Run all validated export bindings → produce artifacts
    // Validate delivery permissions for each binding
    // Store artifacts in cache with session token (UUID)
    // Return preview with per-binding results + session token
}

func (s *CatalogService) Publish(ctx, catalogName, sessionToken) (*PublishResult, error) {
    // Retrieve pre-computed artifacts from cache (410 Gone if expired/missing)
    // Compare catalog's updated_at against the timestamp stored in cache
    //   → if changed: return 409 "catalog modified since preview — re-run preview"
    // Commit publish state to DB
    // Create Catalog CR
    // Update binding statuses
    // Return PublishResult with download token for artifacts
}
```

**Publish response schema (when session token provided):**
```json
{
  "status": "published",
  "download_token": "550e8400-e29b-41d4-a716-446655440000",
  "bindings": [
    { "binding_id": "bind-1", "exporter_name": "mcp-gateway", "status": "success", "artifact_count": 3 },
    { "binding_id": "bind-2", "exporter_name": "configmap", "status": "success", "artifact_count": 1 }
  ]
}
```

The `download_token` is the same session token (artifacts are still in cache). The UI uses it to trigger downloads — **per binding, not combined** — since different bindings may target different namespaces or serve different purposes:

```
GET /api/data/v1/catalogs/{name}/export-bindings/download?token={token}&binding={binding-id}
```

Returns the artifacts for that single binding. Format depends on the exporter (YAML, JSON, XML, etc.). `Content-Disposition: attachment; filename="{catalog-name}-{exporter-name}.{ext}"`. For K8s exporters producing multiple CRs, multi-document YAML (separated by `---`).

The `binding` parameter is **required** — there is no combined download across bindings. Different exporters may produce different formats (YAML, JSON, XML) which cannot be meaningfully concatenated. One download per binding, always.

**UX flow:**
1. User clicks Publish → UI calls `/publish/preview`
2. Preview results shown (per binding) → user confirms
3. UI calls `/publish` with session token → gets `download_token` + per-binding results back
4. UI shows a download button per binding (e.g., "Download MCP Gateway CRs", "Download ConfigMaps")
5. Each click downloads that binding's YAML file
```

**PublishPreview response schema:**
```json
{
  "session_token": "550e8400-e29b-41d4-a716-446655440000",
  "expires_at": "2026-05-10T14:35:00Z",
  "bindings": [
    {
      "binding_id": "bind-1",
      "exporter_name": "mcp-gateway",
      "status": "success",
      "artifact_count": 3,
      "error": ""
    },
    {
      "binding_id": "bind-2",
      "exporter_name": "configmap",
      "status": "failed",
      "artifact_count": 0,
      "error": "entity type 'config' has no instances"
    }
  ],
  "has_failures": true
}
```

If `has_failures` is true, the UI shows a confirmation dialog. The user can proceed with `POST /publish` (passing `session_token` in the request body) or abort.

**Publish request body (when using preview):**
```json
{
  "session_token": "550e8400-e29b-41d4-a716-446655440000"
}
```

If called without a body (no session token), exports run as a fire-and-forget side effect:
- Exports run in a **goroutine with a timeout** (30 seconds). Publish does NOT wait indefinitely.
- Export errors are **logged and recorded on each binding** (`last_run_status = "failed"`, `last_run_error = "..."`) but **not returned to the caller**. The publish response is identical to today's — fully backward compatible.
- If an exporter panics, the goroutine's `recover` catches it. Logged as error, binding marked failed. Publish unaffected.
- The publish response includes an optional `export_warnings` field (array of strings) for callers that choose to inspect it, but it's informational only.

**Important:** Publish and export serve different consumers:
- **Publish** is for pull consumers (using the Asset Hub APIs directly)
- **Export** is for push consumers (integration via CRs — no change needed on the client side)

Export failures do NOT automatically block publish, but the user must be informed before the publish is finalized. The publish flow becomes two-phase when export bindings exist:

1. **Pre-publish exports:** Run all enabled export bindings BEFORE committing the publish state
2. **Present results:** If any export fails, show results to the user (success/failure per binding with error details)
3. **User decides:** "Publish anyway" or "Abort" — no SuperAdmin needed for abort since the publish hasn't been committed yet
4. **Commit:** Only if the user confirms, commit the publish state to DB and create the Catalog CR

If all exports succeed, the publish completes without interruption (no confirmation needed). If there are no export bindings, the publish flow is unchanged from today.

This avoids the need for unpublish-as-rollback (which requires SuperAdmin) and keeps the security model clean — the user who initiated the publish has full authority to abort it.

**Export availability by catalog state:**
- **Manual export to file (download):** Available on ALL catalog states (draft, valid, published). Useful as a debug/preview feature during development.
- **Automatic deploy to K8s (future Phase 2+):** Available on published catalogs ONLY. Unpublished catalogs should not push CRs to live clusters.

## MCP Gateway CR Exporter (First Plugin)

### Input Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `server_type` | string | yes | — | Entity type name for MCP servers (e.g., "mcp-server") |
| `tool_type` | string | yes | — | Entity type name for MCP tools (e.g., "mcp-tool") |
| `target_namespace` | string | no | "default" | K8s namespace for output CRs |

### Mapping Logic

1. Find all instances of `server_type` in the catalog
2. For each server instance:
   a. Map instance attributes → CR spec fields (endpoint, description, etc.)
   b. Find all contained `tool_type` instances (children via containment association)
   c. Map each tool's attributes → nested spec.tools entries
3. Produce one CR per server instance

### Output CR Examples

The exporter produces two types of CRs matching the real MCP Gateway CRDs (`mcp.kuadrant.io/v1alpha1`):

**MCPServerRegistration (one per mcp-server instance):**
```yaml
apiVersion: mcp.kuadrant.io/v1alpha1
kind: MCPServerRegistration
metadata:
  name: github
  namespace: mcp-system
  labels:
    assethub.io/catalog: prod-agents
    assethub.io/exporter: mcp-gateway
    assethub.io/binding-id: bind-123
  annotations:
    assethub.io/exported-at: "2026-05-10T14:30:00Z"
    assethub.io/source-system: assethub-dev
spec:
  prefix: github_
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: github-mcp-route
  path: /mcp
```

**MCPVirtualServer (one per catalog):**
```yaml
apiVersion: mcp.kuadrant.io/v1alpha1
kind: MCPVirtualServer
metadata:
  name: prod-agents
  namespace: mcp-system
  labels:
    assethub.io/catalog: prod-agents
    assethub.io/exporter: mcp-gateway
    assethub.io/binding-id: bind-123
  annotations:
    assethub.io/exported-at: "2026-05-10T14:30:00Z"
spec:
  description: "Production AI agents - MCP tools"
  tools:
    - github_list-repos
    - github_create-issue
    - jira_search-issues
```

**CRD reference:** See `docs/superpowers/specs/2026-05-10-mcp-gateway-crd-reference.md` for full CRD schema details.

### Attribute Mapping

The exporter maps mcp-server instance attributes to `MCPServerRegistration` CR fields and uses containment to find tools for `MCPVirtualServer`:

**MCPServerRegistration (one per mcp-server instance):**

| Instance attribute | CR field | Required |
|-------------------|----------|----------|
| (instance name) | `metadata.name` | yes (implicit) |
| (instance name + `_`) | `spec.prefix` | yes (derived, no attribute needed) |
| `route_name` | `spec.targetRef.name` | yes — HTTPRoute must exist on cluster |
| `mcp_path` | `spec.path` | no — default: `/mcp` |
| `credential_secret` | `spec.credentialRef.name` | no — omitted if not set |

**MCPVirtualServer (one per catalog):**

| Source | CR field |
|--------|----------|
| catalog name | `metadata.name` |
| catalog description | `spec.description` |
| all tool instances, prefixed by parent server's `prefix` attribute | `spec.tools[]` |

**Tool name construction:** For each mcp-tool instance, the tool name in the virtual server is `{parent_server.name}_{tool.name}`. The exporter walks the containment tree to find each tool's parent server. The prefix is derived from the server instance name — no separate attribute needed.

### Instance Name Validation — Tighten to DNS-1123

**Prerequisite for FF-15:** Tighten instance name validation from the current permissive regex (`^[a-zA-Z0-9_][a-zA-Z0-9 ._-]*$`) to DNS-1123 subdomain format (`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`, max 63 chars). Same rule already used for catalog names (`isValidDnsLabel`).

**Why:** Export plugins generate K8s CRs with `metadata.name` derived from instance names. DNS-1123 compliance ensures every instance name is valid as a K8s resource name without per-exporter sanitization. This benefits all future exporters, not just the MCP Gateway exporter.

**What changes:**
- **Backend:** `ValidateInstanceName()` in `instance_service.go` switches to the DNS-1123 regex
- `CreateInstance`, `CreateContainedInstance`, and `UpdateInstance` reject names with uppercase, underscores, spaces, or dots with a clear error message: `"instance name must be a valid Kubernetes resource name: lowercase letters, numbers, and hyphens only, must start and end with a letter or number, max 63 characters"`
- **UI:** `AttributeFormFields.tsx` (or wherever instance name input validation lives) must apply the same DNS-1123 regex client-side, showing inline validation feedback before submission. Both `CreateInstanceModal` and `EditInstanceModal` in the meta and operational UIs.
- Existing instances with non-conforming names are not migrated automatically — the user renames them manually if needed
- Implement as Stage 0 (before the plugin framework), since it affects the core instance service and UI

**Deterministic output ordering:** Generated YAML must have stable ordering for diff stability and GitOps workflows. The exporter sorts:
- `MCPServerRegistration` CRs by `metadata.name` (server instance name)
- `spec.tools[]` entries in `MCPVirtualServer` by tool name (alphabetical)
- The `MCPVirtualServer` CR appears last in the multi-document YAML (after all server registrations)

This ensures identical catalog data always produces byte-identical YAML output. Related: TD-131 tracks the same concern for the existing FF-12 export service.

**Missing required attributes:** If any mcp-server instance is missing `route_name`, the entire export fails with an error listing which instances are incomplete. Future phases may add skip-with-warning for flexibility.

## UI Integration

### Catalog Detail Page — Export Bindings Section

Add an "Export Plugins" tab or section to the meta CatalogDetailPage:

- **List view:** Shows all bindings with exporter name, enabled/disabled toggle, last run status/timestamp, parameters summary
- **Add Binding:** Button opens modal with exporter dropdown, parameter form (generated from ParameterSchema), target namespace field
- **Edit Binding:** Click binding row to edit parameters or enable/disable
- **Run:** "Export Now" button per binding. Downloads the output artifacts. Shows success/failure inline.
- **Delete:** Remove binding with confirmation

### Artifact Preview (nice-to-have, can be deferred)

Before downloading, the user can preview the generated artifacts in a read-only YAML viewer pane (syntax-highlighted, similar to the K8s dashboard YAML view). This lets the user inspect the CRs before applying them to a cluster. The preview is especially valuable during initial setup when the user is verifying the mapping is correct.

If deferred: the "Run" button downloads the file directly. The user inspects with their own YAML viewer/editor.

### Operational Catalog Detail Page

Show export bindings in read-only mode (same as meta page but no edit controls). The "Export Now" button is available for RW+ users.

## Data Model

### New Table: `export_bindings`

```sql
CREATE TABLE export_bindings (
    id              TEXT PRIMARY KEY,
    catalog_id      TEXT NOT NULL REFERENCES catalogs(id) ON DELETE CASCADE,
    exporter_name   TEXT NOT NULL,
    parameters      TEXT NOT NULL DEFAULT '{}',  -- JSON
    enabled         BOOLEAN NOT NULL DEFAULT true,
    last_run_at     TIMESTAMP,
    last_run_status TEXT DEFAULT 'never',        -- 'success', 'failed', 'never'
    last_run_error  TEXT DEFAULT '',
    created_at      TIMESTAMP NOT NULL,
    updated_at      TIMESTAMP NOT NULL
);
```

### Repository Interface

```go
type ExportBindingRepository interface {
    Create(ctx, binding) error
    GetByID(ctx, id) (*ExportBinding, error)
    ListByCatalog(ctx, catalogID) ([]*ExportBinding, error)
    Update(ctx, binding) error
    Delete(ctx, id) error
}
```

## Future Architecture — Documented Open Questions

### F1: Automatic K8s Deployment (Phase 2+)

In Phase 1, the exporter produces CR YAML and the user downloads and applies it manually. In Phase 2+, the system should apply CRs directly to the K8s cluster.

**Requirements:**
- Create/update CRs in the target namespace
- Delete stale CRs when instances are removed from the catalog
- Track which CRs were created by which binding (via labels)

**Approach:** Extend the existing `K8sCRManager` pattern — use the controller-runtime K8s client to create/update CRs. The `Artifact` struct already carries Kind, Name, Namespace, and Content. A `K8sArtifactApplier` adapter would apply artifacts to the cluster.

**Constraint:** Automatic deployment is restricted to published catalogs only. Unpublished/draft catalogs can use manual export (download) but should not push CRs to live clusters.

**Deferred because:** Requires deciding CR ownership model (owner references? labels? finalizers?) and cleanup strategy.

### F2: Namespace Binding Architecture

Catalogs need to be associated with K8s namespaces for automatic export deployment.

**Proposed model:**
- A catalog is bound to one or more namespaces (via a `catalog_namespaces` table or JSON field)
- Each exporter binding specifies which of the catalog's bound namespaces to target
- This aligns with the workload model: a catalog serves a workload, a workload runs in a small number of namespaces, an exporter targets a specific component in a specific namespace

**Interaction with FF-16 (Catalog Views):**
- A view represents how a catalog is seen by a specific workload
- A view could be bound to a namespace, and export bindings on the view export to that namespace
- This gives per-workload, per-namespace export control

**Deferred because:** Namespace binding requires changes to the Catalog model and RBAC model. Design together with FF-16.

### F3: Multi-Cluster Architecture

Asset Hub may be a centralized component serving multiple OCP clusters.

**Open questions:**
- Which components run per-cluster? (at minimum: the operator that watches for CRs and reconciles)
- Which components are centralized? (at minimum: the database, API server, UI)
- How does data propagate from central to per-cluster? Options:
  - **Push model:** Central API pushes CRs to each cluster's API server (requires cross-cluster auth)
  - **Pull model:** Per-cluster operator polls central API for changes (simpler auth, eventual consistency)
  - **GitOps model:** Central exports to a Git repo, per-cluster ArgoCD/Flux syncs from it
  - **Federation model:** CRs are created in a hub cluster, federation controller distributes to managed clusters

**Impact on export plugins:** Once multi-cluster is solved, the export plugin's artifact delivery becomes "apply to the correct cluster's namespace" rather than "apply to this cluster." The plugin interface doesn't change — only the delivery adapter.

**Deferred because:** Architecturally significant. Requires decisions about the overall deployment topology of Asset Hub across clusters.

### F4: CRD-Based Exporter Registration

Phase 1 uses code-registered exporters (`registry.Register()` in main.go). When CRD-based registration is needed (for demos or dynamic plugins), two options exist:

**Option A: Add `exporters` field to existing `AssetHub` CR spec**
```yaml
apiVersion: assethub.project-catalyst.io/v1alpha1
kind: AssetHub
spec:
  exporters:
    - name: mcp-gateway
      description: "Exports MCP server/tool instances as MCP Gateway CRs"
      parameterSchema:
        - name: server_type
          type: string
          required: true
        - name: tool_type
          type: string
          required: true
    - name: custom-webhook
      endpoint: https://my-exporter.svc.cluster.local/export
```

Pros: No new CRD. Central configuration — one `AssetHub` CR per deployment. Simple for a small number of exporters.

**Option B: Separate `CatalogExporter` CRD — one CR per exporter**

Pros: Each exporter has its own lifecycle (create/delete independently). Better for dynamic webhook plugins where exporters come and go. More granular RBAC (different teams can manage their own exporter CRs).

**Recommendation:** Option A for Phase 1 (1-2 built-in exporters, demo scenarios). Option B when dynamic webhook plugins are added — that's when independent lifecycle matters.

### F5: Exporter Unregistration and Binding Lifecycle (Phase 2+)

When a dynamic exporter is unregistered (CRD deleted or webhook service removed), all bindings referencing that exporter across all catalogs should be **disabled, not deleted**. This preserves the binding configuration (parameters, catalog association) so that if the exporter is re-registered later, the user doesn't have to recreate all bindings — just re-enable them.

- On unregister: find all bindings with `exporter_name == <removed exporter>`, set `enabled = false`, set `last_run_status = "exporter_unavailable"`
- On re-register with same name: existing disabled bindings become eligible for re-enablement (user must explicitly re-enable)
- UI: disabled bindings show "exporter unavailable" status with a re-enable button if the exporter is re-registered

Not applicable for Phase 1 (compiled-in exporters are always registered).

### F6: Dynamic Plugins via HTTP Webhooks (Phase 2+)

Phase 1 plugins are compiled into the binary. Phase 2 adds webhook-based dynamic plugins.

**Approach:**
- `WebhookExporter` adapter implements the `Exporter` interface
- Forwards `ExportInput` as JSON POST to the webhook endpoint
- Receives `ExportOutput` as JSON response
- Registered via `CatalogExporter` CRD:

```yaml
apiVersion: assethub.io/v1alpha1
kind: CatalogExporter
metadata:
  name: custom-exporter
spec:
  endpoint: https://my-exporter.my-ns.svc.cluster.local/export
  parameterSchema:
    - name: output_format
      type: string
      required: true
```

**Prerequisite: configurable attribute field mapping (TD-152).** Phase 1's MCP Gateway exporter hardcodes attribute names (`route_name`, `mcp_path`, `credential_secret`). This works for the PoC where we control the schema, but dynamic plugins will export to consumer systems we don't control. Those systems may name their attributes differently (e.g., `route-name`, `httproute-ref`, `routeName`). Before implementing dynamic plugins, exporters must support configurable field mapping — binding parameters that let users map exporter-expected field names to actual attribute names in the catalog schema. Without this, every schema mismatch requires a new exporter or schema changes on the catalog side, which defeats the purpose of a plugin architecture.

**Deferred because:** Phase 1 built-in plugins cover the first use case. Webhook support adds when third-party/dynamic exporters are needed.

### F7: Export Bindings in Catalog Import/Export (FF-12)

**Phase 1 decision:** Export bindings are NOT included in FF-12 catalog export. Bindings contain environment-specific data (target namespaces, route names) that don't transfer between deployments.

**Future: generic export classification mechanism (tracked as TD).** The broader question is which metadata items (attributes, entities, associations, bindings) are "portable" vs "environment-specific." A backup export should include everything; a portability export should strip environment-specific data. This requires:
- A way to tag data as portable vs environment-specific
- Export modes: "backup" (everything) vs "portability" (strip environment-specific)
- Per-attribute or per-entity classification

This is tracked as TD-149 (Important) on the current export design, not as a separate FF. May escalate to FF if scope grows.

Export bindings reference exporters by name. When a catalog is exported (FF-12), its bindings should be included. When imported into a different system:
- If the exporter exists on the target: binding is active
- If the exporter doesn't exist: several options (skip, warn, create inactive, fail)

**Deferred because:** The right behavior depends on the use case (backup vs live restore). Needs further discussion.

### F8: CR Metadata — Source System and Binding ID (Phase 2+)

Phase 1 CRs include `assethub.io/exported-at` (annotation), `assethub.io/catalog`, and `assethub.io/exporter` (labels). Two additional metadata fields from the design spec are deferred:

- **`assethub.io/source-system`** (annotation): Identifies which Asset Hub instance produced the CR (e.g., `assethub-dev` vs `assethub-prod`). Useful when multiple Asset Hub deployments export to the same cluster. Requires a system-level identity config — either an environment variable (`ASSETHUB_INSTANCE_NAME`) or a value from the Asset Hub operator CR.
- **`assethub.io/binding-id`** (label): Identifies which export binding produced the CR. Useful for selective cleanup (delete all CRs from a specific binding). Requires passing the binding ID through `ExportInput` to the exporter.

**Deferred because:** Not needed for PoC. Both are operational convenience features for multi-deployment and CR lifecycle management scenarios. Will decide whether to implement or create a TD when Phase 2 work begins.

### F9: Named Export Bindings (Phase 2+)

Phase 1 export bindings are identified by their exporter name and parameter values. When a catalog has multiple bindings (e.g., two MCP Gateway bindings for different virtual servers, or bindings to different exporters), they are hard to distinguish in the UI — the user sees rows like "mcp-gateway / server_type=mcp-server" repeated with only subtle parameter differences.

**Proposed change:** Add an optional `name` field to `ExportBinding` (model, GORM model, API request/response). The name is user-assigned, free-text, and displayed as the primary identifier in the binding list and export modals. When omitted, the UI falls back to showing the exporter name + parameter summary (current behavior).

**Benefits:**
- Easier to identify bindings at a glance: "Production MCP Export" vs "Staging MCP Export" instead of comparing parameter values
- Better UX when the same exporter is used multiple times with different configurations
- Natural label for the downloaded YAML file name (e.g., `production-mcp-export.yaml` instead of `catalog-name-export.yaml`)

**Deferred because:** Single-binding scenarios work fine without names. Naming becomes valuable when users have 3+ bindings per catalog, which is uncommon in the PoC phase.

## Implementation Stages

### Stage 0: DNS-1123 Instance Name Validation (prerequisite)
- Tighten `ValidateInstanceName()` in `instance_service.go` to DNS-1123 regex
- Update UI client-side validation in `CreateInstanceModal` and `EditInstanceModal`
- Clear error message: "instance name must be a valid Kubernetes resource name: lowercase letters, numbers, and hyphens only, must start and end with a letter or number, max 63 characters"
- Update existing tests that use non-DNS-1123 instance names

### Stage 1: Plugin Framework (backend)
- `Exporter` interface (including `ValidateSchema()`) + `ExporterRegistry`
- `ExportBinding` model + repository + GORM migration (ON DELETE CASCADE for catalog FK)
- `ExportBindingService` with CRUD + Run + RunAll
- `ExportBindingHandler` with API endpoints + middleware stacks (see middleware table in design)
- `GET /exporters` endpoint (no auth required)
- `PreviewCache` interface + `InMemoryPreviewCache` implementation (configurable TTL)
- Wire in main.go
- No UI yet

### Stage 2: MCP Gateway CR Exporter (backend)
- `MCPGatewayExporter` implementing `Exporter` interface
- `ValidateSchema()`: check server_type has `route_name` attribute, containment association to tool_type exists
- Parameter validation at binding creation (server_type, tool_type entity types exist in CV)
- Instance-to-CR mapping: server name → CR name + prefix, tool containment → virtual server tools
- Multi-document YAML output with deterministic ordering (servers by name, tools alphabetical, virtual server last)
- Register in main.go

### Stage 3: Publish Integration (backend)
- `POST /publish/preview` endpoint: re-validate bindings via `ValidateSchema()`, run exports, cache artifacts via `PreviewCache`
- `POST /publish` accepts optional session token: if provided, retrieves cached artifacts and validates catalog not modified (optimistic lock); if not, runs exports as fire-and-forget goroutine (30s timeout, errors logged on bindings, not returned to caller)
- Update binding run status after each execution
- Catalog delete shows warning listing affected bindings before CASCADE

### Stage 4: Export Bindings UI (frontend)
- Export Plugins tab/section on CatalogDetailPage
- Binding list with status, enable/disable, parameters, last run info
- Add/Edit/Delete binding modals (parameter form generated from ParameterSchema)
- "Export Now" button with YAML download (Content-Disposition)
- Exporter list dropdown populated from `GET /exporters`
- Publish flow: call preview first, show results if failures, confirm/abort dialog. On successful publish, auto-download artifacts for each binding — create hidden `<a>` elements with download URLs and trigger `.click()` programmatically (same pattern as existing catalog export button). One file per binding, no extra user clicks needed.
- Operational page: bindings visible read-only, "Export Now" for RW+

### Stage 5: Live Test Scripts
- `scripts/test-export-plugins.sh` — binding CRUD, run, download, RBAC, parameter validation
- Publish-triggered export verification (preview + commit flow)
- Schema drift detection (change CV after binding creation, verify preview catches it)

### Stage 6: System Tests (Playwright)
- Export binding management via UI (add, edit, delete, enable/disable)
- Manual export trigger and YAML download
- Publish with bindings: preview flow, failure confirmation, abort
- Role-based visibility (RO sees bindings read-only, Admin sees CRUD controls)
