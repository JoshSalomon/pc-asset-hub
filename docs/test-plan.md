# AI Asset Hub — Test Plan

## 1. Overview

This document defines the testing strategy for the AI Asset Hub. It covers the testing layers, frameworks, coverage requirements, and the mapping of feature areas to test types.

---

## 2. Testing Layers

The project uses five testing layers, each building on the one below.

### Layer 1: Unit Tests (Go)

**Scope**: Individual functions and methods in isolation. All business logic in `internal/service/` and `internal/domain/`.

**What's tested**:
- Validation logic (cycle detection, name uniqueness, attribute type checking)
- Version increment logic (copy-on-write for entity types, auto-increment for instances)
- Enum reference enforcement
- Lifecycle state machine transitions (valid and invalid paths)
- DTO mapping (domain models ↔ API request/response types)
- Error handling and domain error types
- CatalogVersion CRD types: DeepCopy correctness for `CatalogVersion` and `CatalogVersionList`, scheme registration
- SanitizeK8sName: version label → valid K8s resource name conversion
- Config: `CLUSTER_ROLE` defaults and `AllowedStages` mapping for each clusterRole value

**Mocking strategy**: Repository interfaces (from `internal/domain/repository/`) are mocked using `testify/mock`. Services never touch a real database in unit tests.

**Framework**: Go `testing` + `testify` (assertions + mocks)

---

### Layer 2: Integration Tests (Go + Database)

**Scope**: Repository implementations (`internal/infrastructure/gorm/repository/`) against a real database.

**What's tested**:
- CRUD operations for all tables (entity types, entity type versions, attributes, associations, enums, enum values, catalog versions, catalog version pins, lifecycle transitions, entity instances, instance attribute values, association links)
- Unique constraint enforcement (entity type names, attribute names within version, enum values)
- Foreign key constraint enforcement
- Cascade behavior (containment deletes at the database level)
- Transaction atomicity (all-or-nothing for multi-table operations)
- Migration correctness (schema created correctly from scratch)
- Optimistic locking (concurrent version conflict detection)
- EAV query patterns (filtering entity instances by attribute values across the join)

**Database**: SQLite in-memory — fast, disposable, each test starts with a fresh schema via GORM auto-migration.

**Framework**: Go `testing` + GORM with SQLite driver

---

### Layer 3: API Tests (Go + HTTP)

**Scope**: Full HTTP request/response cycle through the API layer (`internal/api/`).

**What's tested**:
- Routing: correct handler invoked for each URL pattern (both Meta and Operational APIs)
- Request validation: missing required fields, invalid types, malformed JSON
- Response formats: correct HTTP status codes, JSON structure, pagination headers/links
- Error responses: 400 (bad request), 403 (forbidden), 404 (not found), 409 (conflict), 422 (unprocessable — e.g., cycle detection)
- RBAC enforcement per endpoint: all four roles tested against every endpoint for both allowed and denied operations
- Catalog version scoping: operational API returns only data consistent with the pinned catalog version
- Dynamic routing: operational API URL patterns derived from entity type definitions in the catalog version

**RBAC mocking**: SubjectAccessReview is mocked to simulate each role without requiring a real OpenShift cluster.

**Framework**: Go `testing` + `net/http/httptest` + Echo test utilities

---

### Layer 4: UI Tests (TypeScript/React)

**Scope**: React components and page-level user flows.

**What's tested**:
- **Component tests**: Individual PatternFly components render correctly with various props and states (data tables, forms, modals, dialogs, badges).
- **Page integration tests**: Full page flows including form submission, navigation, state updates, and error handling — with mocked API responses.
- **Role-aware rendering**: Controls hidden for unauthorized roles, disabled for state-restricted operations, with correct tooltips.
- **Validation behavior**: Inline validation errors, cycle detection feedback in association dialogs, attribute name conflict indicators in the copy picker.
- **Association map**: Topology graph renders entity types as nodes and associations as labeled edges, click navigation works, zoom/pan functional.

**Mocking strategy**: API responses mocked via MSW (Mock Service Worker). MSW intercepts HTTP requests at the network level, so React Query caching and refetching work naturally without special test configuration.

**Framework**: Vitest + React Testing Library + MSW

---

### Layer 5: Operator Tests (Go + envtest)

**Scope**: Operator reconciliation logic (`internal/operator/`).

**What's tested**:
- AssetHub CR creation triggers expected Kubernetes resources (Deployment, Service, ConfigMap, etc.)
- CRD/CR generation from catalog version entity type definitions produces valid Kubernetes YAML
- Catalog version promotion to Testing/Production triggers CRD/CR application to the cluster
- Catalog version demotion triggers CRD/CR cleanup from the cluster
- Reconciliation errors are reported via the CR's status conditions and operator logs
- Deletion of the AssetHub CR cleanly removes all managed resources
- **Pure reconciler**: `ReconcileAssetHub` produces ConfigMap with `CLUSTER_ROLE` for all three clusterRole values (development/testing/production); `ReconcileCatalogVersionStatus` returns correct status for valid/invalid lifecycle stages
- **Controller** (fake K8s client): reconcile produces ConfigMap with `CLUSTER_ROLE`; operator sets owner references on existing CatalogVersion CRs; CatalogVersion status updated after reconciliation; no CatalogVersion CRs in namespace → no error
- **K8s CR Manager** (fake K8s client): `CreateOrUpdate` creates new CRs with correct spec and annotations; updates existing CRs idempotently; `Delete` removes CRs; delete of nonexistent CR is idempotent

**Framework**: Go `testing` + `sigs.k8s.io/controller-runtime/pkg/envtest` (simulated Kubernetes API server — no real cluster required)

---

## 3. Feature Area Coverage Matrix

Each feature area is tested at the appropriate layers:

| Feature Area | Unit | Integration | API | UI | Operator |
|---|---|---|---|---|---|
| Entity type CRUD + versioning | X | X | X | X | |
| Attribute management | X | X | X | X | |
| Association management + cycle detection | X | X | X | X | |
| Enum management | X | X | X | X | |
| Copy entity type / copy attributes | X | X | X | X | |
| Catalog version CRUD | X | X | X | X | |
| Lifecycle promotion/demotion + CR management | X | X | X | X | X |
| Entity instance CRUD + versioning | X | X | X | X | |
| Containment traversal + cascade delete | X | X | X | X | |
| Forward/reverse reference queries | X | X | X | X | |
| Filtering/sorting/pagination | X | X | X | X | |
| Optimistic locking (409 conflicts) | X | X | X | | |
| RBAC enforcement (all 4 roles) | X | | X | X | |
| Version history + comparison/diff | X | X | X | X | |
| CRD/CR generation | X | | | | X |
| CatalogVersion CRD types + DeepCopy | X | | | | |
| CatalogVersion CR management (K8s) | X | | | | X |
| ClusterRole / stage filtering | X | | X | | X |
| Operator reconciliation | | | | | X |
| Association map visualization | | | | X | |
| Entity type diagram — UML composition diamond (TD-47) | X | | | X | |
| Model Diagram tab on catalog pages (US-48) | X | | | X | |
| Landing page + unified SPA routing (US-47) | X | | | X | |
| Description fields — ET list, enum, CV (TD-43/45/46) | X | X | X | X | |
| Edit attribute (COW) | X | | X | X | |
| Rename entity type (simple + deep copy) | X | | X | X | |
| Catalog version pins + transitions | X | | X | X | |
| Catalog version stage filter | | | X | X | |
| Catalog version detail page | | | | X | |
| CV create with entity selection | X | | X | X | |
| CV create containment tree + version picker | X | | X | X | |
| Version snapshot + read-only BOM modal | X | | X | X | |
| Association cardinality | X | X | X | X | |
| Edit association (COW) | X | | X | X | |
| Association names + shared namespace | X | X | X | X | |
| Catalog CRUD + name validation | X | X | X | X | |
| Catalog scoping (data isolation) | X | X | X | | |
| Instance CRUD within catalog | X | X | X | X | |
| Instance attribute values (set/get/validate) | X | X | X | X | |
| Entity type pin resolution (catalog → CV → pins) | X | X | X | | |
| Instance optimistic locking | X | X | X | | |
| Catalog validation status reset on mutation | X | X | X | | |
| Contained instance CRUD within catalog | X | X | X | X | |
| Association link CRUD + validation against CV | X | X | X | X | |
| Forward/reverse reference resolution (operational) | X | X | X | X | |
| Containment tree endpoint | X | X | X | | |
| Attribute-based filtering (string/number/enum) | X | X | X | X | |
| Multi-field sorting | X | X | X | X | |
| Pagination (offset/limit/total) | X | X | X | X | |
| Parent chain resolution (breadcrumb) | X | | X | X | |
| Operational UI — catalog list + counts | | | | X | |
| Operational UI — containment tree browser | | | | X | |
| Operational UI — instance detail + attributes | | | | X | |
| Operational UI — reference navigation | | | | X | |
| Operational UI — breadcrumb navigation | | | | X | |
| Operational UI — Vite multi-entry build | | | | X | |
| Catalog-level RBAC — access check + middleware | X | | X | | |
| Catalog-level RBAC — catalog list filtering | X | | X | | |
| Catalog-level RBAC — header mode passthrough | X | | X | | |
| Catalog validation — required attrs + type check | X | X | X | X | |
| Catalog validation — mandatory associations | X | X | X | | |
| Catalog validation — containment consistency | X | X | X | | |
| Catalog validation — status update (valid/invalid) | X | X | X | X | |
| Catalog validation — RBAC (RW+ only) | X | | X | X | |
| Catalog publish/unpublish — service + status | X | X | X | X | |
| Catalog publish — RBAC (Admin+ only) | X | | X | X | |
| Published catalog write protection (SuperAdmin) | X | X | X | X | |
| Catalog CR lifecycle (create/delete on publish) | X | | | | X |
| Catalog CR status.DataVersion (operator bump) | | | | | X |
| CV promotion warnings (draft/invalid catalogs) | X | X | X | X | |
| Copy Catalog — deep clone (service + transaction) | X | X | X | X | |
| Copy Catalog — instance/attr/link/hierarchy remap | X | X | X | | |
| Replace Catalog — atomic swap + archive naming | X | X | X | X | |
| Replace Catalog — published state transfer + CR sync | X | X | X | | X |
| Copy & Replace — RBAC (RW+ copy, Admin+ replace) | X | | X | X | |
| Copy & Replace — validation & error cases | X | X | X | | |
| System attributes — API injection (instance responses) | X | | X | | |
| System attributes — API injection (snapshot + attr list) | X | | X | X | |
| System attributes — reserved name rejection | X | | X | X | |
| System attributes — copy-attributes exclusion | X | | X | X | |
| System attributes — catalog validation (Name non-empty) | X | X | X | | |
| System attributes — UI unified rendering (create/edit) | | | | X | |
| System attributes — UI system badge + edit protection | | | | X | |
| CV metadata edit — label + description (US-49) | X | X | X | X | |
| Catalog metadata edit — name + description (US-50) | X | X | X | X | |
| Catalog re-pinning — change CV (US-51) | X | X | X | X | |
| CV pin add/remove (US-52) | X | X | X | X | |
| Pin editing stage guards (TD-69) | X | | X | | X |

---

## 4. Coverage Strategy

- **Target**: 100% code coverage across all layers.
- **Per-step measurement**: Coverage is measured at every implementation step, not just at the end. Each step must meet the target before proceeding.
- **Documented exceptions**: Lines that cannot be covered must have a per-line justification. Expected exceptions include:
  - `main()` entrypoint and server bootstrap code
  - Operator binary entrypoint
  - Panic recovery middleware (requires actual panics)
  - Platform-specific build-tag code not exercised in the test environment
- **Backend tooling**: `go test -coverprofile=coverage.out ./...` with `go tool cover -func=coverage.out` for analysis.
- **UI tooling**: Vitest with `--coverage` flag (v8 provider).

---

## 5. Cross-Cutting Test Strategies

### 5.1 RBAC

RBAC is a cross-cutting concern that must be verified at every layer where it applies:

- **Unit tests**: Mock the SubjectAccessReview call. Verify that each service method checks the correct permission for the operation being performed.
- **API tests**: For every endpoint, test all four roles (RO, RW, Admin, Super Admin). Verify both successful access and 403 rejection. Pay special attention to lifecycle transition endpoints where role requirements differ by operation (e.g., RW can promote dev→test but not test→prod).
- **UI tests**: Verify that controls are hidden (role-based restriction) or disabled with tooltip (state-based restriction) per role.

### 5.2 Versioning

Versioning affects entity types, entity instances, and catalog versions:

- **Integration tests**: Verify that entity type mutations produce new versions with correct copy-on-write semantics — all attributes AND associations from the previous version are copied to the new version. Verify that the previous version remains immutable.
- **API tests**: Verify that version numbers appear in all responses. Verify 409 Conflict on optimistic locking violations (update with stale version number).
- **API tests**: Verify the version history endpoint returns all versions in order. Verify the comparison endpoint correctly identifies added, removed, and modified attributes/associations between two versions.

### 5.3 Containment

Containment involves hierarchy management, cascade operations, and namespace scoping:

- **Integration tests**: Verify cascade delete across multiple containment levels (A contains B contains C — deleting A removes B and C). Verify atomicity (partial cascade failure rolls back everything).
- **API tests**: Verify sub-resource URL patterns (`GET /{parent-type}/{id}/{contained-type}`). Verify name uniqueness is enforced within parent scope, not globally. Verify 404 when accessing a contained entity via a non-existent parent.
- **Unit tests**: Verify that the cycle detection algorithm correctly rejects containment associations that would create cycles, including multi-level cycles (A→B→C→A) and self-containment.

### 5.4 CatalogVersion CR Management and Cluster Role

CatalogVersion CRs bridge the database and K8s for discovery. Cluster role controls data visibility.

- **Unit tests**: Verify `CatalogVersionCRManager` interface — `CreateOrUpdate` creates/updates CRs with correct spec, annotations, and entity type names; `Delete` is idempotent. Verify `SanitizeK8sName` produces valid K8s names from arbitrary version labels. Verify `AllowedStages` returns correct lifecycle stages for each clusterRole value.
- **Unit tests (service)**: Verify `Promote` calls `CreateOrUpdate` with correct lifecycle stage; `Demote` to development calls `Delete`; `Demote` to testing calls `CreateOrUpdate` (not `Delete`). Verify crManager=nil gracefully skips CR operations (DB-only mode). Verify `ListCatalogVersions` and `GetCatalogVersion` filter by `allowedStages`.
- **Operator tests (pure reconciler)**: Verify `ReconcileAssetHub` produces ConfigMap with `CLUSTER_ROLE` for all three clusterRole values. Verify `ReconcileCatalogVersionStatus` returns correct ready/message for valid lifecycle stages.
- **Operator tests (controller)**: Verify reconcile sets owner references on CatalogVersion CRs, updates status, and handles empty namespace.

**Coverage targets**:
- 100% on pure reconciler functions, CatalogVersion types (DeepCopy), SanitizeK8sName, K8sCRManager
- ≥90% on controller (excluding SetupWithManager) and CatalogVersionService promotion/demotion with CR operations
- Documented exceptions per uncovered line

### 5.5 Catalog Scoping

Catalog scoping is the core isolation mechanism for the operational API. Entity instances belong to a catalog (named data collection), which is pinned to a catalog version (schema snapshot).

- **API tests**: Verify that every operational API call scoped to a catalog returns only entity instances belonging to that catalog.
- **API tests**: Verify that instances in one catalog are not visible in another catalog, even if both share the same CV.
- **API tests**: Verify that requests with a nonexistent catalog name return 404.
- **API tests**: Verify that the catalog's pinned CV determines which entity types are available for creating instances.

### 5.6 Edit Attribute

EditAttribute uses the same copy-on-write versioning pattern as AddAttribute/RemoveAttribute.

- **Unit tests (service)**: Verify COW version increment on edit. Verify field updates (name, description, type, enumID, required) are applied to the correct attribute in the new version. Verify name conflict detection against other attributes in the same version. Verify enum validation (enum type requires enumID, invalid enumID rejected). Verify NotFound for nonexistent attribute. Verify ordinal is preserved.
- **API tests (handler)**: Verify `PUT /entity-types/:entityTypeId/attributes/:name` returns 200 with new version on success, 403 for RO role, 404 for nonexistent attribute, 409 for name conflict, 400 for invalid enum reference. Verify required field accepted in create and edit requests.
- **UI tests (browser)**: Verify add attribute modal has required checkbox (default unchecked). Verify edit attribute modal has required checkbox pre-filled. Verify attributes table shows required indicator. Verify BOM modal shows required indicator.

### 5.7 Rename Entity Type

Rename is context-sensitive: simple rename when safe, deep copy when the entity type is referenced by non-development catalog versions.

- **Unit tests (service)**: Verify simple rename when entity type is not in any catalog version. Verify simple rename when entity type is in exactly one development-stage catalog version. Verify deep copy is triggered when entity type is in a testing/production catalog version. Verify deep copy is triggered when entity type is in multiple catalog versions. Verify `DeepCopyRequired` error returned when `deepCopyAllowed=false` and deep copy is needed. Verify name uniqueness check. Verify empty name validation.
- **API tests (handler)**: Verify `POST /entity-types/:id/rename` returns 200 with updated entity type on simple rename, 409 with `deep_copy_required` error when deep copy needed and not allowed, 200 with new entity type and `was_deep_copy=true` when deep copy allowed.
- **UI tests**: Verify two-step rename flow — first attempt without deep copy, warning modal on 409 response, retry with deep copy on user confirmation.

### 5.8 Copy Attributes Handler

Service method already exists and is tested. This adds the HTTP handler layer.

- **API tests (handler)**: Verify `POST /entity-types/:entityTypeId/attributes/copy` returns 200 with new version on success, 403 for RO role, 409 for name conflict.
- **UI tests**: Verify copy attributes picker — source entity type selection, attribute checkboxes, conflict indicators for same-name attributes, API call on confirm.

### 5.9 Catalog Version Detail — Pins and Transitions

Two new read-only endpoints expose catalog version internals.

- **Unit tests (service)**: Verify `ListPins` resolves pins to entity type names and version numbers. Verify empty pins list for CV with no pins. Verify NotFound for nonexistent CV. Verify `ListTransitions` returns chronologically ordered history. Verify empty history for new CV.
- **API tests (handler)**: Verify `GET /catalog-versions/:id/pins` returns 200 with resolved pin data. Verify `GET /catalog-versions/:id/transitions` returns 200 with ordered transitions.
- **UI tests**: Verify CV detail page renders tabs (Overview, Bill of Materials, Transitions). Verify pins table shows entity type names and versions. Verify transitions table shows ordered history with from/to stages and timestamps.

### 5.10 Catalog Version Create with Entity Selection

The backend already accepts pins in `CreateCatalogVersionRequest`. Tests verify the pin-handling path works correctly end-to-end.

- **Unit tests (service)**: Verify `CreateCatalogVersion` with pins creates pin records linked to the CV. Verify CV creation with empty pins list succeeds (no pins created). Verify pin with invalid entity type version ID is rejected.
- **API tests (handler)**: Verify `POST /catalog-versions` with `pins` array creates CV and associated pins. Verify response reflects created CV. Verify 403 for RO role.
- **UI tests**: Verify entity type selection checkboxes in create modal. Verify containment cascade on check. Verify BOM summary panel. Verify pins are sent in API request on create.

### 5.11 Catalog Version Stage Filter

Stage filtering uses existing GORM filter support. Only the handler needs to pass the query parameter through.

- **API tests (handler)**: Verify `GET /catalog-versions?stage=testing` returns only testing+production CVs (respecting allowedStages). Verify omitting stage returns all CVs.
- **UI tests**: Verify stage filter dropdown in CV list toolbar. Verify list updates on filter change.

### 5.12 CV Create Containment Tree and Version Picker

The CV creation modal shows entity types organized as a containment tree with per-entity version selection.

- **Unit tests (service)**: Verify `GetContainmentTree` builds tree from containment edges. Verify root identification (entities not appearing as target in any containment edge). Verify multi-level nesting. Verify flat entities (no containment) appear as standalone roots. Verify all versions included per node with latest_version set.
- **API tests (handler)**: Verify `GET /entity-types/containment-tree` returns 200 with tree response. Verify empty tree when no entity types.
- **UI tests**: Verify tree structure with indentation and parent/child hierarchy. Verify recursive containment cascade selection: selecting a parent auto-selects all descendants (children, grandchildren, etc.); deselecting a parent deselects all descendants recursively; selecting a child auto-selects all ancestors up to the root; deselecting a child does NOT deselect its parent or ancestors. Verify version dropdown shows all versions per entity type, defaults to latest, selection changes pin.

### 5.13 Version Snapshot and Read-Only BOM Modal

The catalog version BOM tab shows pinned entity types. Clicking an entity type name opens a read-only modal displaying the pinned version's attributes and associations.

- **Unit tests (service)**: Verify `GetVersionSnapshot` returns attributes and associations (both outgoing and incoming) for a specific entity type version. Verify enum names resolved for enum-type attributes. Verify target entity type names resolved for associations. Verify error when entity type or version not found.
- **API tests (handler)**: Verify `GET /entity-types/:id/versions/:version/snapshot` returns 200 with attributes, associations, resolved enum names, and resolved entity type names. Verify 404 for nonexistent entity type or version. Verify 400 for invalid version (negative, zero).
- **UI tests**: Verify clicking an entity type name in the BOM tab opens a modal (not navigates). Verify modal shows entity type name, pinned version, attributes with resolved enum names (e.g., "boolean (enum)"), and associations with contextual relationship labels (contains/contained by/references/referenced by/references (mutual)) and perspective-correct roles. Verify no edit controls are present in the modal.

### 5.14 Association Cardinality

Cardinality adds UML-style multiplicity (`source_cardinality`, `target_cardinality`) to every association. Standard options: `0..1`, `0..n`, `1`, `1..n`. Custom ranges supported (e.g., `2..5`, `2..n`). Default: `0..n` on both ends for backward compatibility. Empty string is normalized to `0..n`.

- **Unit tests (validation)**: Verify `ValidateCardinality` accepts all standard options, custom ranges (e.g., `2..5`, `2..n`), exact values (e.g., `3`), and empty string. Verify rejection of invalid formats: negative numbers, min > max, non-numeric, malformed patterns. Verify `NormalizeCardinality` returns `"0..n"` for empty string and passes through valid values unchanged.
- **Unit tests (service)**: Verify `CreateAssociation` validates cardinality before creating. Verify invalid cardinality returns error. Verify empty cardinality is normalized to `"0..n"` on the created association. Verify cardinality values are passed through to the association model.
- **Integration tests (repository)**: Verify cardinality fields are stored and retrieved correctly. Verify `BulkCopyToVersion` preserves cardinality values on copied associations.
- **API tests (handler)**: Verify `POST /entity-types/:id/associations` accepts `source_cardinality` and `target_cardinality` fields. Verify `GET /entity-types/:id/associations` returns cardinality in response (normalized to `"0..n"` for existing associations). Verify invalid cardinality returns 400. Verify version snapshot includes cardinality.
- **UI tests (browser)**: Verify add association modal includes cardinality dropdowns with standard options and custom input. Verify default is `0..n`. Verify cardinality column in associations table. Verify BOM modal shows cardinality for associations.

### 5.15 Edit Association

EditAssociation uses the same copy-on-write versioning pattern as EditAttribute. Editable fields are source role, target role, source cardinality, and target cardinality. Association type and target entity type are immutable (delete and recreate).

- **Unit tests (service)**: Verify COW version increment on edit. Verify field updates (source role, target role, source cardinality, target cardinality) are applied to the correct association in the new version. Verify containment source cardinality constraint enforced on edit. Verify invalid cardinality rejected. Verify NotFound for nonexistent association.
- **API tests (handler)**: Verify `PUT /entity-types/:entityTypeId/associations/:name` returns 200 with new version on success, 403 for RO role, 404 for nonexistent association, 400 for invalid cardinality.
- **UI tests (browser)**: Verify Edit button in associations table opens modal with pre-filled values. Verify Save triggers API call with changed fields. Verify containment source cardinality restriction in edit modal. Verify custom cardinality option in edit modal (same as add modal). Verify shared EditAssociationModal component used in both entity detail page and diagram.

### 5.16 Association Names and Shared Namespace

Associations have a required name, unique within the entity type version. Names share the same namespace as attributes — no collision allowed between attribute and association names on the same version.

- **Unit tests (service)**: Verify CreateAssociation requires a name. Verify name uniqueness within version. Verify shared namespace — association name conflicts with existing attribute name. Verify attribute name conflicts with existing association name. Verify COW matching by name instead of all-properties. Verify EditAssociation can rename. Verify DeleteAssociation by name.
- **Integration tests (repository)**: Verify name stored and retrieved. Verify unique constraint on (version_id, name). Verify BulkCopy preserves names.
- **API tests (handler)**: Verify create requires name. Verify name in list response. Verify PUT/DELETE routes use `:name`. Verify 409 on duplicate name. Verify snapshot includes name.
- **UI tests (browser)**: Verify name field in add/edit modals. Verify name displayed in associations table.

### 5.17 Entity Type Diagram (US-32)

UML-like graphical diagram of entity types and their associations using `@patternfly/react-topology`. Appears in two locations: main page "Model Diagram" tab (all entity types, interactive) and CV detail page "Diagram" tab (pinned entity types, read-only).

- **UI tests (browser — main page)**: Verify "Model Diagram" tab exists. Verify diagram renders entity type nodes with name, version, and attributes. Verify edges rendered between associated entity types with labels (name, type, cardinality). Verify containment edges visually distinct from reference edges. Verify zoom/pan controls present.
- **UI tests (browser — CV detail)**: Verify "Diagram" tab exists on CV detail page. Verify diagram renders only pinned entity types with attributes and associations.

### 5.18 Catalog CRUD (US-33)

Catalogs are named data containers pinned to a catalog version. The operational API uses the catalog name in URLs (DNS-label format). This section covers the catalog entity itself — instance management within catalogs is tested separately.

- **Unit tests (service)**: Verify `CreateCatalog` validates name format (DNS-label: `^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`, max 63 chars), rejects invalid names, enforces uniqueness, verifies CV exists, and sets initial validation status to `draft`. Verify `GetByName` retrieves catalog with resolved CV label. Verify `List` supports filtering by `catalog_version_id` and `validation_status`. Verify `Delete` cascades to all entity instances in the catalog.
- **Integration tests (repository)**: Verify CRUD operations on `catalogs` table. Verify unique constraint on name. Verify FK to `catalog_versions`. Verify cascade behavior when catalog is deleted (entity instances removed). Verify `entity_instances.catalog_id` FK works correctly.
- **API tests (handler)**: Verify `POST /api/data/v1/catalogs` returns 201 with catalog data on success. Verify 400 for invalid name format. Verify 409 for duplicate name. Verify 404 for nonexistent CV ID. Verify 403 for RO role. Verify `GET /api/data/v1/catalogs` returns list with filtering. Verify `GET /api/data/v1/catalogs/{name}` returns catalog detail with resolved CV label. Verify `DELETE /api/data/v1/catalogs/{name}` returns 204 and cascades. Verify 403 for RO on delete.
- **UI tests (browser)**: Verify Catalogs nav item in meta UI. Verify catalog list page shows name, CV label, validation status badge, created date. Verify create modal with name input (DNS-label validation), description, CV dropdown. Verify delete with confirmation dialog. Verify RO user sees no create/delete controls.

### 5.19 Instance CRUD with Attributes (US-13, US-14, US-15)

Entity instances are created within a catalog, scoped to an entity type that must be pinned in the catalog's CV. Attribute values are set on create, updated on PUT, and returned with resolved names in all responses. The old CV-scoped instance scaffolding is replaced with catalog-scoped routes.

- **Unit tests (service)**: Verify instance creation resolves catalog name → CV → pins → entity type version. Verify entity type not pinned in CV returns error. Verify attribute value type validation (string accepted, number must be parseable, enum must be in allowed list). Verify missing optional attributes allowed. Verify name uniqueness within catalog scope. Verify optimistic locking on update (version mismatch returns conflict). Verify update increments version and stores previous attribute values. Verify cascade delete removes children. Verify catalog validation status reset to `draft` on create/update/delete.
- **Integration tests (repository)**: Verify `InstanceAttributeValue` CRUD — set values, get current values, get values for specific version. Verify attribute values survive instance version increment (previous values retained). Verify instance creation with `catalog_id` FK. Verify pin resolution query chain (catalog → CV → pins → entity type version) against real DB with multi-table joins. Verify optimistic locking — concurrent update with stale version returns 0 rows affected. Verify catalog validation status reset — `UpdateValidationStatus` called after instance mutation updates the `updated_at` timestamp.
- **API tests (handler)**: Verify `POST /api/data/v1/catalogs/{name}/{entity-type}` creates instance with attributes, returns 201. Verify 404 for nonexistent catalog. Verify 404 for entity type not pinned in CV. Verify 400 for invalid attribute values. Verify 403 for RO. Verify `GET /{name}/{type}` lists instances with attribute values. Verify `GET /{name}/{type}/{id}` returns instance with resolved attributes. Verify `PUT /{name}/{type}/{id}` updates attributes, increments version, returns 409 on version mismatch. Verify `DELETE /{name}/{type}/{id}` returns 204 with cascade.
- **UI tests (browser)**: Verify catalog detail page shows tabs per entity type. Verify instance list table with dynamic columns from attributes. Verify create instance modal with dynamic attribute form (text for string, number input for number, dropdown for enum). Verify edit instance modal pre-fills current values. Verify delete with confirmation. Verify RO user sees no create/edit/delete controls.

### 5.20 Contained Instance CRUD within Catalog (US-16, US-18)

Contained instances are created under a parent instance via sub-resource URLs. The containment relationship must be validated against the pinned CV's association definitions — a containment association must exist between the parent's entity type and the child's entity type. Name uniqueness is enforced within the parent's namespace. Single-level containment routes are supported; multi-level containment URLs (e.g., `/a/{a-id}/b/{b-id}/c`) are deferred to Phase 4.

- **Unit tests (service)**: Verify `CreateContainedInstance` validates parent exists, child entity type is pinned in CV, and a containment association exists between parent and child types in the CV. Verify name uniqueness within parent namespace (same name under different parents allowed, same name under same parent rejected). Verify creation with nonexistent parent returns NotFound. Verify creation with entity type not in a containment relationship returns validation error. Verify `ListContainedInstances` returns only direct children of the specified type under the parent. Verify contained instance creation resets catalog validation status to draft.
- **Integration tests (repository)**: Verify contained instance stored with `parent_instance_id` set. Verify unique constraint `(entity_type_id, catalog_id, parent_instance_id, name)` allows same name under different parents. Verify `ListByParent` returns correct children.
- **API tests (handler)**: Verify `POST /{catalog-name}/{parent-type}/{parent-id}/{child-type}` creates contained instance, returns 201. Verify 404 for nonexistent parent. Verify 400 for entity type not in containment relationship. Verify 403 for RO. Verify `GET /{catalog-name}/{parent-type}/{parent-id}/{child-type}` lists contained instances.
- **UI tests (browser)**: Verify parent instance detail shows contained children. Verify "Add contained instance" action creates child under parent. Verify contained instance appears in parent's children list.

### 5.21 Association Link CRUD and Validation (US-19, US-20)

Association links connect two instances based on a directional or bidirectional association defined in the pinned CV. Link creation validates that the source and target instances match the association definition's entity types. Forward references return all instances referenced by a given instance; reverse references return all instances that reference a given instance.

- **Unit tests (service)**: Verify `CreateAssociationLink` validates association definition exists in CV, source instance's entity type matches association's source entity type, and target instance's entity type matches association's target entity type. Verify link creation with nonexistent association returns NotFound. Verify link creation with mismatched entity types returns validation error. Verify link creation with nonexistent target instance returns NotFound. Verify `DeleteAssociationLink` removes the link. Verify `GetForwardReferences` returns resolved target info (link ID, association name, association type, target instance ID/name/entity type). Verify `GetReverseReferences` returns resolved source info. Verify both directional and bidirectional associations included in forward and reverse queries. Verify link creation resets catalog validation status to draft.
- **Integration tests (repository)**: Verify `AssociationLink` CRUD — create link, get forward refs, get reverse refs, delete link. Verify FK constraints on source/target instance IDs.
- **API tests (handler)**: Verify `POST /{catalog-name}/{entity-type}/{instance-id}/links` creates link, returns 201. Verify 404 for nonexistent instance or target. Verify 400 for invalid association. Verify 403 for RO. Verify `DELETE /{catalog-name}/{entity-type}/{instance-id}/links/{link-id}` returns 204. Verify `GET /{catalog-name}/{entity-type}/{instance-id}/references` returns forward refs with resolved target info. Verify `GET /{catalog-name}/{entity-type}/{instance-id}/referenced-by` returns reverse refs with resolved source info.
- **UI tests (browser)**: Verify instance detail shows references tab with forward and reverse references. Verify "Link to instance" action creates association link. Verify "Unlink" action removes association link. Verify RO user sees references but no link/unlink controls.

### 5.22 Containment Tree Endpoint (US-18, US-40)

The containment tree endpoint returns the full instance hierarchy for a catalog as a nested structure. Each node includes the instance, its entity type name, and its children. This powers the tree browser in the operational UI.

- **Unit tests (service)**: Verify `GetContainmentTree` builds a correct tree from flat instance list. Verify root instances (no parent) appear as top-level nodes. Verify children are nested under their parent, grouped by entity type. Verify multi-level nesting (grandchildren). Verify empty catalog returns empty tree. Verify instances are annotated with entity type name.
- **Integration tests (repository)**: Verify `ListByCatalog` returns all instances in a catalog regardless of entity type. Verify instances from other catalogs are excluded.
- **API tests (handler)**: Verify `GET /api/data/v1/catalogs/{name}/tree` returns 200 with nested tree structure. Verify 404 for nonexistent catalog. Verify tree includes entity type names on each node. Verify tree structure matches containment relationships.

### 5.23 Attribute-Based Filtering (US-17)

Instance list endpoints accept filter query parameters that are applied server-side via SQL JOINs on the EAV attribute value table. Filter semantics are type-aware.

- **Unit tests (service)**: Verify filter params are passed through to repository. Verify unknown attribute name in filter returns validation error.
- **Integration tests (repository)**: Verify string filter applies case-insensitive contains match. Verify number filter applies exact match. Verify number range filter with min/max. Verify enum filter applies exact match. Verify multiple filters combine with AND logic. Verify filter on nonexistent attribute returns empty result (not error at repo level). Verify filtering works correctly across the EAV join (instance ↔ attribute value ↔ attribute).
- **API tests (handler)**: Verify `GET /{name}/{type}?filter.attr=value` returns filtered results. Verify `filter.numattr=5` filters by exact number. Verify `filter.numattr.min=1&filter.numattr.max=10` filters by range. Verify multiple filter params combine. Verify 400 for filter on unknown attribute.

### 5.24 Sorting and Pagination (US-17)

Instance list endpoints accept sort and pagination query parameters. Sorting is applied server-side. Pagination uses offset/limit with total count in the response.

- **Unit tests (service)**: Verify sort params are passed through to repository. Verify pagination params (offset, limit) are passed through.
- **Integration tests (repository)**: Verify sorting by string attribute (alphabetical). Verify sorting by number attribute (numeric order). Verify ascending and descending sort. Verify offset skips correct number of results. Verify limit caps result count. Verify total count is unaffected by offset/limit.
- **API tests (handler)**: Verify `?sort=attr:asc` sorts ascending. Verify `?sort=attr:desc` sorts descending. Verify `?limit=5&offset=10` returns correct page. Verify response includes total count. Verify default limit is 20 when not specified. Verify limit capped at 100.
- **UI tests (browser)**: Sorting and pagination UI controls are deferred to FF-6 (operational editing). The backend API supports these via query parameters and is tested at the API layer.

### 5.25 Parent Chain Resolution (US-18, US-40)

Instance detail responses include a parent chain — an ordered list of ancestors from root to immediate parent — for breadcrumb navigation.

- **Unit tests (service)**: Verify parent chain resolves from instance up to root. Verify root instance has empty parent chain. Verify multi-level chain (3+ levels). Verify each entry includes instance ID, instance name, and entity type name.
- **API tests (handler)**: Verify `GET /{name}/{type}/{id}` includes `parent_chain` array in response. Verify chain is ordered root-first. Verify chain is empty for root instances.
- **UI tests (browser)**: Verify breadcrumb renders containment path from catalog to current instance. Verify breadcrumb shows entity type and instance name for each level. Verify breadcrumb links are clickable and navigate to the ancestor in the tree. Verify root instance shows breadcrumb with catalog only (no parent entries).

### 5.26 Operational UI — Catalog Data Viewer (US-40)

The operational UI is a separate read-only web application for browsing catalog data. It shares types and API client with the meta UI but has its own Vite entry point and app shell.

- **UI tests (browser — build infrastructure)**: Verify Vite multi-entry build produces both `index.html` (meta) and `operational.html` (operational). Verify operational entry point renders the operational app shell. Verify operational app masthead shows "AI Asset Hub — Data Viewer". Verify role selector in masthead works.
- **UI tests (browser — catalog list)**: Verify catalog list page loads and shows catalogs. Verify catalog name, CV label, validation status badge, and instance count columns. Verify search input filters catalogs by name. Verify sortable columns. Verify pagination controls. Verify clicking catalog name navigates to catalog detail.
- **UI tests (browser — catalog detail overview)**: Verify catalog detail page shows catalog header with name, status badge, and CV label. Verify overview tab lists entity types with instance counts. Verify "Browse Instances" navigates to tree browser for that type.
- **UI tests (browser — containment tree)**: Verify tree browser tab shows two-pane layout: tree on left, detail on right. Verify tree groups root instances under entity type headers with counts. Verify clicking a tree node loads instance detail in the right panel. Verify multi-level tree expands correctly. Verify empty state when no instance selected.
- **UI tests (browser — instance detail)**: Verify instance detail panel shows attributes table with name, type, and value. Verify enum values show resolved names. Verify description, version, and timestamps displayed. Verify breadcrumb shows containment path.
- **UI tests (browser — reference navigation)**: Verify references tab shows forward references with association name, type, target instance, and entity type. Verify referenced-by tab shows reverse references. Verify clicking a referenced instance navigates to it in the tree.
- **UI tests (browser — read-only)**: Verify no create, edit, or delete buttons are visible in the operational UI regardless of role. Verify no write-action modals exist.

### 5.27 Catalog-Level RBAC (US-23, US-39)

Per-catalog access control using a `CatalogAccessChecker` interface. In header-based dev mode, all catalogs are accessible (passthrough). In SAR mode (future Phase C), SubjectAccessReview checks `resourceName` against K8s RBAC. The middleware extracts the catalog name from the URL path and checks access before the handler runs.

- **Unit tests (middleware)**: Verify `RequireCatalogAccess` middleware extracts catalog name from `:catalog-name` URL param. Verify middleware calls `CatalogAccessChecker.CheckAccess` with correct catalog name and verb. Verify middleware returns 403 when access is denied. Verify middleware passes through when access is allowed. Verify verb mapping: GET→get, POST→create, PUT/PATCH→update, DELETE→delete.
- **Unit tests (HeaderCatalogAccessChecker)**: Verify always returns true (passthrough in dev mode).
- **API tests (handler)**: Verify `GET /api/data/v1/catalogs` filters results through access checker. Verify catalog list with denied catalogs excludes them. Verify `GET /api/data/v1/catalogs/{name}/...` returns 403 when access denied. Verify all sub-resource operations (instances, tree, links, references) inherit catalog access check.
- **Integration note**: Real SAR-based checking is deferred to Phase C (OCP cluster required). Unit tests use a mock `CatalogAccessChecker` that can be configured to allow/deny specific catalogs.

### 5.28 Catalog Validation (US-34)

On-demand validation of all entity instances in a catalog against the pinned CV's schema. The `CatalogValidationService` loads all instances, resolves the schema for each entity type via the CV's pins, and checks four constraint categories. Returns a structured error list and updates the catalog's validation status.

- **Unit tests (service)**: Verify required attribute check — instance missing a value for a `Required=true` attribute produces a validation error. Verify type check — number attribute with non-parseable value, enum attribute with value not in allowed list. Verify mandatory association check — association with target cardinality min >= 1 (e.g., `1` or `1..n`) requires each source instance to have at least one link; missing link produces error. Verify containment check — instance with `ParentInstanceID` pointing to a non-existent instance, or parent whose entity type has no containment association to the child's type. Verify status update — no errors sets status to `valid`; errors set status to `invalid`. Verify empty catalog (no instances) passes validation. Verify error structure includes entity type name, instance name, field name, and violation description.
- **Integration tests (end-to-end validation against real DB)**: Set up a complete catalog with a CV, pins, entity types with required attributes, mandatory associations, and containment. Create instances with valid and invalid data. Run the full validation service against the real SQLite database and verify it detects the expected violations. Verify a fully valid catalog produces no errors. Verify status is persisted correctly after validation.
- **API tests (handler)**: Verify `POST /api/data/v1/catalogs/{name}/validate` returns 200 with validation results. Verify response includes `status` (`valid` or `invalid`) and `errors` array. Verify 404 for nonexistent catalog. Verify 403 for RO role. Verify catalog status is updated in the database after validation.
- **UI tests (browser — meta)**: Verify "Validate" button appears on catalog detail page for RW+ users. Verify RO users do not see the Validate button. Verify clicking Validate calls the API and displays results. Verify validation errors are displayed grouped by entity type with per-instance details. Verify status badge updates after validation.
- **UI tests (browser — operational)**: Verify "Validate" button appears on operational catalog detail page. Verify validation results display in the operational UI.

### 5.29 Catalog Publishing (US-42, US-43)

Explicit publish/unpublish operations for catalogs. Publishing creates a namespaced Catalog CR in K8s for discovery. Published catalogs are write-protected — data mutations require SuperAdmin role. CV promotion warns about draft/invalid catalogs.

- **Unit tests (service — publish/unpublish)**: Verify `Publish` requires validation status `valid` — returns error for `draft` or `invalid`. Verify `Publish` sets `published=true` and `published_at` timestamp. Verify `Publish` calls CatalogCRManager.CreateOrUpdate with correct spec. Verify `Unpublish` sets `published=false` and calls CatalogCRManager.Delete. Verify `Publish` on already-published catalog is idempotent. Verify `Unpublish` on unpublished catalog is idempotent. Verify `Publish`/`Unpublish` with nil crManager skips CR operations (DB-only mode).
- **Unit tests (service — write protection)**: Verify data mutations (CreateInstance, UpdateInstance, DeleteInstance, CreateContainedInstance, CreateAssociationLink, DeleteAssociationLink, SetParent) on a published catalog return 403 for RW role. Verify same mutations succeed for SuperAdmin. Verify mutations still reset validation status to `draft` even on published catalogs. Verify `draft` does not auto-unpublish — `published` stays `true`.
- **Unit tests (service — CV promotion warnings)**: Verify `Promote` returns warnings for catalogs pinned to the CV with `draft` or `invalid` status. Verify promotion proceeds despite warnings. Verify no warnings when all pinned catalogs are `valid`. Verify no warnings when no catalogs are pinned.
- **Integration tests (publish/unpublish)**: Verify publish persists `published=true` and `published_at` in the database. Verify unpublish persists `published=false`. Verify data mutation on published catalog by SuperAdmin resets status to `draft` but keeps `published=true`.
- **Integration tests (write protection)**: Set up a published catalog with instances in real SQLite. Verify instance creation on published catalog fails for non-SuperAdmin. Verify instance creation succeeds for SuperAdmin. Verify the `published` field survives the full create→publish→mutate→query round-trip.
- **Integration tests (CV promotion warnings)**: Create a CV with multiple pinned catalogs at different validation statuses (draft, valid, invalid) in real SQLite. Promote the CV and verify the response includes warnings for draft/invalid catalogs. Verify no warnings when all catalogs are valid.
- **API tests (handler)**: Verify `POST /catalogs/{name}/publish` returns 200 for Admin. Verify 403 for RW and RO. Verify 400 for `draft` or `invalid` catalog. Verify 404 for nonexistent catalog. Verify `POST /catalogs/{name}/unpublish` returns 200 for Admin, 403 for RW/RO. Verify instance create/update/delete on published catalog returns 403 for RW, 200 for SuperAdmin.
- **Unit tests (K8s CR manager)**: Verify `CatalogCRManager.CreateOrUpdate` creates Catalog CR with correct spec, annotations, and namespace. Verify `CatalogCRManager.Delete` removes Catalog CR. Verify Delete is idempotent (nonexistent CR returns nil).
- **Operator tests**: Verify reconciler sets owner reference on Catalog CRs. Verify reconciler updates Catalog CR status. Verify `status.DataVersion` is incremented on each reconciliation. Verify new Catalog CR has `DataVersion: 0` before first reconciliation. Verify `DataVersion` becomes 1 after first reconciliation. Verify `DataVersion` increments from existing value on subsequent reconciliations.
- **UI tests (browser — meta)**: Verify "Publish" button visible for Admin on `valid` unpublished catalog. Verify "Publish" button hidden for RW. Verify "Publish" button hidden when catalog is `draft` or `invalid`. Verify "Unpublish" button visible on published catalog for Admin. Verify published badge on catalog list and detail. Verify warning banner on published catalog for RW users. Verify instance create/edit/delete controls disabled on published catalogs for RW. Verify controls enabled for SuperAdmin on published catalogs.
- **UI tests (browser — CV promotion)**: Verify promotion dialog shows warnings for draft/invalid catalogs pinned to the CV being promoted.

### 5.30 Copy & Replace Catalog (US-44, US-45, US-46)

Copy Catalog deep-clones all data (instances, attribute values, association links, containment hierarchy) from a source catalog into a new catalog with new UUIDs, remapped references, and `draft` status. Replace Catalog atomically swaps a staging catalog into the name of a published one by renaming both catalogs in a single transaction. Both operations require transactional guarantees — all-or-nothing.

**What is tested at each layer:**

- **Unit tests (service — CopyCatalog)**: Verify copy creates a new catalog with same CV pin, `draft` status, and new description. Verify all instances are cloned with new UUIDs, same entity type/name/description, version reset to 1. Verify attribute values are cloned and remapped to new instance IDs. Verify association links are cloned with remapped source/target instance IDs. Verify containment hierarchy is preserved — parent references remapped to new instance IDs. Verify source catalog name must exist (returns NotFound). Verify target name validated (DNS-label, uniqueness — returns ConflictError). Verify copy is transactional — if any step fails, the new catalog is not created. Verify copy of an empty catalog (no instances) creates empty catalog. Verify self-referential links (source and target in same catalog) are correctly remapped.
- **Unit tests (service — ReplaceCatalog)**: Verify source validation status must be `valid` (returns error for `draft` or `invalid`). Verify source and target must exist (404 for each). Verify target is renamed to archive name (default: `{target}-archive-{timestamp}`). Verify source is renamed to target's original name. Verify archive name is validated (DNS-label format). Verify custom archive name is used when provided. Verify replace is transactional — both renames succeed or neither does. Verify published state transfer: if target was published, source (now named as target) inherits `published=true` and `published_at`; archive becomes `published=false`. Verify SyncCR is called after replace to update the Catalog CR spec with new data. Verify CR DataVersion is bumped so consumers detect the swap. Verify archive catalog's CR is deleted (it has a new name). Verify source cannot equal target (returns error). Verify archive name collision (returns ConflictError if archive name already taken).
- **Unit tests (handler — DTO)**: Verify `CopyCatalogRequest` binding with `source`, `name`, `description`. Verify `ReplaceCatalogRequest` binding with `source`, `target`, `archive_name`. Verify correct HTTP status codes (201 for copy, 200 for replace).
- **Integration tests (repository — UpdateName)**: Verify `UpdateName` updates the catalog name in the database. Verify `UpdateName` returns ConflictError when new name already exists. Verify `UpdateName` returns NotFoundError when catalog ID doesn't exist.
- **Integration tests (end-to-end copy)**: Set up a complete catalog in real SQLite with instances, attribute values, association links, and containment. Copy it. Verify the new catalog has all data with new IDs. Verify original catalog is unchanged. Verify association links point to the new instances. Verify containment hierarchy is intact.
- **Integration tests (end-to-end replace)**: Set up source and target catalogs in real SQLite. Replace. Verify names swapped correctly. Verify published state transferred. Verify data integrity — instances in the renamed catalogs still have correct catalog IDs.
- **Integration tests (validation error paths)**: Verify replace with `draft` source returns error and no names are changed. Verify replace with nonexistent source returns NotFound. Verify replace with archive name that already exists returns ConflictError. Verify copy with duplicate target name returns ConflictError. Verify all error cases leave the database unchanged (transactional rollback).
- **API tests (handler — copy)**: Verify `POST /api/data/v1/catalogs/copy` returns 201 with new catalog. Verify 404 for nonexistent source. Verify 409 for duplicate target name. Verify 400 for invalid target name. Verify 403 for RO role. Verify response includes full catalog detail with resolved CV label.
- **API tests (handler — replace)**: Verify `POST /api/data/v1/catalogs/replace` returns 200 with updated catalog. Verify 400 for non-valid source. Verify 404 for nonexistent source or target. Verify 400 for invalid archive name. Verify 403 for non-Admin roles (RO, RW). Verify response includes updated catalog.
- **UI tests (browser — copy modal)**: Verify "Copy" button visible on catalog detail page for RW+ users. Verify Copy button hidden for RO. Verify copy modal opens with name input. Verify name input validates DNS-label format (shows error for invalid names). Verify successful copy navigates to the new catalog or refreshes the list. Verify copy error (409 duplicate name) shows alert.
- **UI tests (browser — replace modal)**: Verify "Replace" button visible on `valid` staging catalogs for Admin+ users. Verify Replace button hidden for RW and RO. Verify Replace button hidden for `draft` or `invalid` catalogs. Verify replace modal shows target catalog dropdown. Verify optional archive name input with DNS-label validation. Verify successful replace refreshes catalog list. Verify replace error shows alert.
- **Operator tests**: Verify Catalog CR DataVersion bumped after replace (via SyncCR → CreateOrUpdate with incremented SyncVersion, then operator reconciliation increments DataVersion).
- **Live system test**: Verify full staging workflow — copy published catalog, edit staging copy, validate, replace back. Verify original data archived. Verify rollback by replacing from archive.

### 5.31 System Attributes — Common Attributes as Schema-Level Attributes (TD-22)

Common attributes (Name — required, Description — optional) are hardcoded fields on `EntityInstance` but are surfaced as synthetic system attributes (`system: true`) in all API responses. This makes them visible in attribute lists, UML diagrams, version snapshots, and instance create/edit modals. The API layer injects them; no DB schema changes are made (Approach B).

**What is tested at each layer:**

- **Unit tests (handler — instance DTO injection)**: Verify `instanceDetailToDTO` prepends two system attributes (name, description) before custom attributes. Verify system attrs have `system: true`, correct types (`string`), and correct required flags (name=required, description=optional). Verify system attr values match the instance's `Name` and `Description` fields. Verify custom attributes retain `system: false` (or omitted). Verify injection works for instances with zero custom attributes.
- **Unit tests (handler — snapshot injection)**: Verify version snapshot response includes system attributes (name, description) at the start of the attributes array with `system: true`. Verify ordinals are negative (name=-2, description=-1) to sort before custom attrs (ordinal >= 0).
- **Unit tests (handler — attribute list injection)**: Verify attribute list endpoint for an entity type version prepends system attributes. Verify system attrs have `system: true`, correct names, types, and required flags.
- **Unit tests (handler — reserved name rejection)**: Verify creating an attribute named "name" returns 409/validation error. Verify creating an attribute named "description" returns 409/validation error. Verify creating an attribute named "Name" (case variation) is allowed (names are case-sensitive). Verify renaming an attribute to "name" or "description" is rejected.
- **Unit tests (service — copy attributes exclusion)**: Verify `CopyAttributes` with "name" in the attribute list silently skips it. Verify `CopyAttributes` with "description" in the list silently skips it. Verify `CopyAttributes` with a mix of system and custom names copies only the custom ones.
- **Unit tests (service — validation Name check)**: Verify `Validate` returns an error for instances with empty Name. Verify `Validate` returns an error for instances with whitespace-only Name. Verify `Validate` passes for instances with non-empty Name. Verify the validation error has field="name", entity type name resolved, and a clear violation message.
- **Integration tests (validation Name check)**: Verify end-to-end validation against real SQLite with an empty-named instance returns `invalid` status with the name error.
- **API tests (instance CRUD — system attrs in response)**: Verify `POST` create instance response includes system attrs in the attributes array. Verify `GET` instance response includes system attrs. Verify `GET` list instances response includes system attrs per instance. Verify `PUT` update instance response includes system attrs with updated values.
- **API tests (meta — reserved name rejection)**: Verify `POST` create attribute with name "name" returns 400/409. Verify `POST` create attribute with name "description" returns 400/409.
- **API tests (meta — snapshot includes system attrs)**: Verify `GET` version snapshot includes system attrs at the start of the attributes array.
- **UI tests (browser — meta attribute list)**: Verify entity type detail page shows "Name" and "Description" system attributes with a "System" badge. Verify system attributes appear before custom attributes. Verify delete/edit controls are disabled for system attributes.
- **UI tests (browser — meta copy attributes picker)**: Verify copy-attributes dialog excludes system attributes from the source list.
- **UI tests (browser — operational create modal)**: Verify create instance modal renders Name and Description fields from schema attributes (not hardcoded). Verify Name field is required. Verify Description field is optional. Verify custom attributes render after system attributes.
- **UI tests (browser — operational edit modal)**: Verify edit instance modal shows Name and Description as editable fields from schema. Verify saving updates Name and Description via top-level request fields.
- **UI tests (browser — API client)**: Verify `client.ts` functions pass system attribute data correctly in requests and responses.
- **Live system test**: Verify end-to-end — create entity type, view its attributes (system attrs visible), create instance with name/description, verify instance response includes system attrs, validate catalog with empty-named instance fails, verify UML diagram shows system attrs.

### 5.32 Component Decomposition (TD-23, TD-35) — Refactoring

Pure refactoring of three oversized page components into custom hooks and modal sub-components. Zero behavior changes. Existing browser tests serve as regression safety net; new hook and component tests improve coverage and testability.

**What is tested at each layer:**

- **Existing browser tests (537+) must pass unchanged.** The refactoring moves code between files but does not change any behavior, API calls, or rendered output. These serve as integration-level regression tests.
- **New hook unit tests** (`renderHook` with mocked API): Each extracted hook (`useCatalogData`, `useInstances`, `useInstanceDetail`, `useEntityTypeData`, `useAttributeManagement`, `useAssociationManagement`, `useContainmentTree`) is tested in isolation. Tests verify data loading, error handling, state management, and guard clause behavior. Testing hooks in isolation makes previously-uncoverable guard clauses (e.g., `if (!name) return`) coverable — call the hook without providing the dependency.
- **New modal component tests** (browser): Each extracted modal (`CreateInstanceModal`, `EditInstanceModal`, `AddChildModal`, `LinkModal`, `SetParentModal`, `AddAttributeModal`, `EditAttributeModal`, `AddAssociationModal`, `CopyAttributesModal`, `RenameEntityTypeModal`) is tested in isolation with mock props. Tests verify form rendering, field validation, submit behavior, and close callbacks — without the complex page-level setup currently required.
- **Coverage must not regress** for any affected file. Per-file coverage deltas are reported. Target: coverage improvement due to simplified, isolated testability.

### 5.33 Modal State Internalization + Shared Components (TD-23 Phase 4)

Modals internalize their form state (own `useState`, pass values up via `onSubmit`). Shared `AttributeFormFields` component and `buildTypedAttrs` utility extracted. Copy/Replace modals extracted from CatalogDetailPage.

**What is tested at each layer:**

- **Existing page-level browser tests (671+) must pass unchanged.** Modal interface changes are internal — page tests interact with the rendered UI, not prop interfaces.
- **Updated modal component tests** (browser): All 10 modal tests rewritten to match the new interface — modals own form state, tests fill form fields and verify `onSubmit` callback receives correct typed values.
- **New `AttributeFormFields` component tests** (browser): Renders system attrs with required indicators, custom attrs from schema, enum selects, number inputs. Tests `includeSystem` prop to control system attr visibility. Verifies `onChange` callback.
- **New `buildTypedAttrs` utility tests** (unit): Converts string→number for number-type attrs, passes through string/enum, skips empty values, handles edge cases.
- **New `CopyCatalogModal` component tests** (browser): DNS-label name validation, disabled submit when empty, `onSubmit` with correct args, error display.
- **New `ReplaceCatalogModal` component tests** (browser): Target catalog dropdown, archive name input, disabled submit when target not selected, `onSubmit` with correct args, error display.

### 5.34 UML Composition Diamond + Model Diagram Tab (TD-47, US-48)

TD-47 adds UML composition notation (filled diamond on parent end) to containment edges in the entity type diagram. US-48 adds a read-only "Model Diagram" tab to both meta and operational catalog detail pages, showing the entity type model from the catalog's pinned CV.

- **Unit tests (TypeScript — buildModel)**: Verify containment edges in the built model include diamond marker data (`markerStart` type). Verify non-containment edges (directional, bidirectional) do not include diamond marker data. Verify bidirectional edges retain their existing marker configuration.
- **Unit tests (TypeScript — useCatalogDiagram hook)**: Verify hook loads pins and snapshots when tab becomes active. Verify hook returns loading state during fetch. Verify hook returns diagram data after successful fetch. Verify hook does not re-fetch if data is already loaded. Verify hook handles API errors gracefully.
- **Browser tests (EntityTypeDiagram rendering)**: Verify containment edges render with a filled diamond SVG marker on the source (parent) end. Verify the diamond uses the containment color (`#3e8635`). Verify non-containment edges do not render a diamond. Verify bidirectional edges retain their existing hollow/filled arrow markers.
- **Browser tests (meta CatalogDetailPage)**: Verify "Model Diagram" tab exists on the catalog detail page. Verify clicking the tab loads and renders the entity type diagram with pinned entity types. Verify diagram shows entity types, attributes, and associations from the CV. Verify the diagram is read-only (no edit interactions). Verify empty state when no entity types are pinned.
- **Browser tests (operational OperationalCatalogDetailPage)**: Verify "Model Diagram" tab exists on the operational catalog detail page. Verify clicking the tab loads and renders the entity type diagram. Verify the diagram is read-only. Verify empty state when no entity types are pinned.

### 5.35 Landing Page + Unified SPA (US-47)

US-47 merges the two separate SPAs (meta + operational) into a single SPA with route-based views. A landing page at `/` provides navigation to schema management (`/schema`) and catalog data viewers (`/catalogs/:name`).

- **Unit tests (TypeScript — catalog card rendering)**: Verify catalog card displays name, CV label, validation status badge (draft/valid/invalid with correct colors), published indicator. Verify card with no description renders cleanly. Verify card with long name/description truncates or wraps.
- **Browser tests (LandingPage)**: Verify landing page renders at root URL. Verify Schema Management card is visible and links to `/schema`. Verify catalog cards are rendered for each accessible catalog with name, CV label, validation status, and published indicator. Verify clicking a catalog card navigates to `/catalogs/:name`. Verify empty state when no catalogs are accessible. Verify loading state while fetching catalogs. Verify error state on API failure.
- **Browser tests (App routing)**: Verify `/schema` renders the schema management tabs (entity types, catalog versions, enums, model diagram). Verify `/schema/entity-types/:id` renders entity type detail page. Verify `/schema/catalog-versions/:id` renders CV detail page. Verify `/schema/catalogs/:name` renders catalog detail page. Verify `/catalogs/:name` renders the operational catalog data viewer. Verify masthead shows "Schema" on schema pages. Verify masthead shows "Data Viewer" on catalog viewer pages. Verify masthead brand link navigates back to landing page.
- **Browser tests (regression)**: All existing App.tsx tests pass with updated `/schema` routes. All existing OperationalCatalogDetailPage tests pass at the new `/catalogs/:name` route. All existing CatalogDetailPage tests pass at `/schema/catalogs/:name`.
- **System tests**: Verify landing page loads in live deployment. Verify navigation from landing page to schema management works end-to-end. Verify navigation from landing page to catalog data viewer works end-to-end. Verify `/schema` routes serve correctly through nginx. Verify `/catalogs/:name` routes serve correctly through nginx (no separate `operational.html`).

### 5.36 Description Fields — Entity Type List, Enum, Catalog Version (TD-43, TD-45, TD-46)

Adds description fields across the schema management layer. Entity type list resolves the latest version's description into the API response. Enum and CatalogVersion models gain a new `description` field with full CRUD support. Entity type detail page gains an inline editable description.

- **Unit tests (Go — service)**: Verify enum create with description stores it. Verify enum update description. Verify CV create with description stores it.
- **Integration tests (Go — repository)**: Verify Enum description field stored and retrieved. Verify CatalogVersion description field stored and retrieved. Verify GORM migration adds the column without data loss.
- **API tests (Go — handler)**: Verify entity type list response includes `description` field resolved from latest version. Verify entity type with no versions returns empty description. Verify enum create accepts description. Verify enum response includes description. Verify CV create accepts description. Verify CV response includes description.
- **Browser tests (App.tsx — entity type list)**: Verify Description column visible in entity type list. Verify description text shown for entity types that have one.
- **Browser tests (EntityTypeDetailPage — TD-46)**: Verify description shown in overview section. Verify edit description triggers PUT and creates new version. Verify updated description visible after save.
- **Browser tests (EnumListPage)**: Verify Description column visible. Verify create modal has description field. Verify description shown in list after creation.
- **Browser tests (EnumDetailPage)**: Verify description shown in detail view. Verify description editable.
- **Browser tests (App.tsx — CV list)**: Verify Description column visible in CV list. Verify create modal has description field.
- **Browser tests (CatalogVersionDetailPage)**: Verify description shown in overview section.

### 5.37 Catalog Version Metadata Edit (US-49, TD-61)

Update a catalog version's version label and/or description after creation via `PUT /catalog-versions/:id`. Uses `*string` pattern for optional field preservation.

- **Unit tests (service)**: Verify `UpdateCatalogVersion` updates description when provided. Verify label update when provided. Verify omitted fields preserved (nil pointer = no change). Verify label uniqueness enforced (409 on duplicate). Verify NotFound for nonexistent CV.
- **Integration tests (repository)**: Verify label update persists correctly in real SQLite. Verify `uniqueIndex` on `VersionLabel` rejects duplicate labels at DB level. Verify description update persists. Verify updating a nonexistent CV returns error.
- **API tests (handler)**: Verify `PUT /catalog-versions/:id` with `{"description":"new"}` returns 200 with updated CV. Verify `{"version_label":"v2.1"}` renames the CV. Verify `{}` (empty body) preserves all fields. Verify 409 for duplicate label. Verify 404 for nonexistent CV. Verify 403 for RO.
- **UI tests (browser)**: Verify inline Edit button next to description on CV detail page. Verify edit flow: click Edit → TextInput appears → type → Save → API called → value updated. Verify Cancel restores original value. Verify inline Edit button next to version label. Verify label edit triggers PUT. Verify RO user sees no Edit buttons.

### 5.38 Catalog Metadata Edit (US-50, FF-10)

Update a catalog's name and/or description after creation via `PUT /catalogs/{name}`. Published catalogs restrict editing (SuperAdmin only for description, rename blocked).

- **Unit tests (service)**: Verify `UpdateMetadata` updates description. Verify name change with DNS-label validation and uniqueness check. Verify omitted fields preserved (`*string`). Verify published catalog: SuperAdmin can edit description, rename blocked (returns 400). Verify non-SuperAdmin on published catalog returns 403. Verify validation status reset to `draft` on any change. Verify SyncCR called after metadata change on published catalog.
- **Integration tests (repository)**: Verify catalog name update persists and old name no longer resolves. Verify unique constraint on catalog name rejects duplicates at DB level. Verify description update persists. Verify `UpdateValidationStatus` to `draft` after metadata change. Verify catalog with instances — rename preserves all instance `catalog_id` FK references (instances still belong to the same catalog entity).
- **API tests (handler)**: Verify `PUT /catalogs/{name}` with `{"description":"new"}` returns 200. Verify `{"name":"new-name"}` renames catalog and redirects response to new name. Verify `{}` preserves all fields. Verify 400 for invalid DNS-label name. Verify 409 for duplicate name. Verify 404 for nonexistent catalog. Verify 403 for RO. Verify `RequireWriteAccess` + `RequireCatalogAccess` middleware applied. Verify published catalog: 403 for RW/Admin on any edit, 400 for SuperAdmin rename, 200 for SuperAdmin description edit.
- **UI tests (browser)**: Verify inline Edit button for description on catalog detail page (same pattern as EntityTypeDetailPage). Verify edit flow: click Edit → input → Save → API called → value updated. Verify Cancel restores. Verify RO user sees no Edit button. Verify published catalog shows disabled Edit for non-SuperAdmin.
- **Live system tests**: Add test cases to `scripts/test-descriptions.sh` — update catalog description, verify persisted; update catalog name, verify redirect; attempt rename on published catalog, verify rejection.

### 5.39 Catalog Re-pinning (US-51, TD-12)

Change a catalog's pinned CV via `PUT /catalogs/{name}` with `catalog_version_id`. Resets validation to `draft`. Published catalogs must unpublish first.

- **Unit tests (service)**: Verify re-pin updates `catalog_version_id`. Verify new CV must exist (404 if not). Verify validation status reset to `draft` on re-pin. Verify published catalog re-pin blocked (returns 400 with "unpublish first"). Verify unpublished catalog re-pin succeeds.
- **Integration tests (repository)**: Verify `catalog_version_id` FK update persists in real SQLite. Verify FK constraint — re-pin to nonexistent CV ID rejected at DB level. Verify catalog's validation status updated to `draft` after re-pin. Verify instances remain associated with the catalog after re-pin (only the CV reference changes, not the catalog ID on instances).
- **API tests (handler)**: Verify `PUT /catalogs/{name}` with `{"catalog_version_id":"new-cv"}` returns 200. Verify 404 for nonexistent CV. Verify 400 for published catalog. Verify 403 for RO.
- **UI tests (browser)**: Verify CV selector dropdown on catalog detail page for Admin+ users. Verify dropdown lists available CVs. Verify selecting a new CV triggers PUT. Verify dropdown disabled on published catalogs.

### 5.40 Catalog Version Pin Editing (US-52, FF-4)

Add, remove, or change version of entity type pins in a catalog version. Each entity type can appear at most once.

- **Unit tests (service — AddPin)**: Verify `AddPin` validates ETV exists (404 if not). Verify duplicate entity type (not just ETV) returns 409 — adding V2 of "Server" when V1 is already pinned must fail. Verify adding a different entity type succeeds. Verify RW+ role required.
- **Unit tests (service — UpdatePin)**: Verify `UpdatePin` changes the pinned ETV. Verify new ETV must belong to the same entity type as the existing pin (returns 400 if mismatched). Verify 404 for nonexistent pin or ETV. Verify pin ownership — pin must belong to the specified CV.
- **Unit tests (service — RemovePin)**: Verify `RemovePin` removes pin by ID. Verify 404 for nonexistent pin.
- **Integration tests (repository)**: Verify pin creation persists. Verify pin update persists. Verify `ListByCatalogVersion` returns updated pins. Verify FK constraints.
- **API tests (handler)**: Verify `POST /pins` returns 201. Verify 409 for duplicate entity type. Verify `PUT /pins/:pin-id` returns 200 with updated ETV. Verify 400 for entity type mismatch on update. Verify `DELETE /pins/:pin-id` returns 204. Verify 403 for RO on all endpoints.
- **Browser tests (BOM tab — inline version change)**: Verify version column is a dropdown for Admin+. Verify dropdown lists all versions of the entity type. Verify selecting a different version calls updatePin API. Verify version updates in the table after successful change. Verify RO user sees plain text, not dropdown.
- **Browser tests (Add Pin modal — entity type filtering)**: Verify Add Pin modal only shows entity types NOT already pinned. Verify after adding a pin, the entity type disappears from the Add Pin dropdown. Verify after removing a pin, the entity type reappears in the Add Pin dropdown.
