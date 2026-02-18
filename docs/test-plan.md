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

### 5.5 Catalog Version Scoping

Catalog version scoping is the core isolation mechanism for the operational API:

- **API tests**: Verify that every operational API call scoped to a catalog version returns only entity types and instances consistent with that version's pinned entity type definitions.
- **API tests**: Verify that creating or modifying entities in one catalog version does not affect another.
- **API tests**: Verify that requests with an invalid catalog version return a clear error. Verify that requests against a demoted (no longer in production) catalog version return an appropriate error rather than stale data.
