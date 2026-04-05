# AI Asset Hub — Detailed Test Plan

## Overview

This document specifies test cases organized by implementation milestone. Each test case maps to specific user story acceptance criteria and identifies the testing layer (Unit, Integration, API, UI, Operator). Test case IDs follow the format `T-{milestone}.{sequence}`.

## Development Environment Phases

Development proceeds through three environment phases, each with increasing infrastructure access. Human approval is required before transitioning between phases.

### Environment Phase A: Isolated Development (No Containers)

**Environment**: Development machine with Go, Node.js, and standard tooling. No access to Docker, Podman, or any container runtime.

**What can be developed and tested**:
- All Go backend code (domain models, repository implementations, services, API handlers, middleware)
- All UI code (React components, pages, hooks, state management)
- Operator reconciliation logic and CRD/CR generation code
- All unit tests (mocked dependencies, `testify/mock`)
- All integration tests (SQLite in-memory via GORM — no container needed)
- All API tests (`net/http/httptest` — in-process HTTP, no running server)
- All UI tests (Vitest + JSDOM + MSW — no browser or server needed)
- Operator logic unit tests with envtest (`controller-runtime/pkg/envtest` downloads etcd and kube-apiserver binaries directly — no containers)

**What cannot be tested**:
- PostgreSQL-specific behavior (only SQLite is available)
- Container image builds
- Deployment on a real cluster
- Real OpenShift RBAC (SubjectAccessReview against a live API server)

**Milestones completed**: 1–12 plus CatalogVersion Discovery CRD (all code written and tested)

**Tests that must pass**: All test cases T-1.01 through T-9.09, T-CV.01 through T-CV.31, T-E.01 through T-E.146, T-10.01 through T-10.51, T-11.01 through T-11.58, T-12.01 through T-12.63, T-13.01 through T-13.102 (T-13.78 through T-13.85 retired), T-14.01 through T-14.22, T-15.01 through T-15.81, T-16.01 through T-16.69, T-17.01 through T-17.88, T-24.01 through T-24.28, T-25.01 through T-25.39, T-26.01 through T-26.18, and T-27.01 through T-27.27 (914 test cases), using SQLite and mocked/simulated infrastructure.

**Human checkpoint**: After all 802 tests pass with 100% coverage (documented exceptions). This is the first review point.

---

### Environment Phase B: Local Kubernetes (kind)

**Environment**: Development machine with Docker/Podman access and a kind (Kubernetes in Docker) cluster.

**What is developed and tested**:
- Dockerfiles for API server, UI, and operator
- Container image builds and registry push (local)
- Docker Compose or kind-based local deployment
- PostgreSQL integration tests (PostgreSQL running in a container)
- Full end-to-end tests against running services in kind
- Operator deployment on kind cluster with real CRDs/CRs
- Promotion/demotion lifecycle with actual cluster-side effects

**Additional test cases for Phase B**:

| ID | Test Case | Layer | Notes |
|----|-----------|-------|-------|
| T-B.01 | API server container image builds successfully | Deployment | — |
| T-B.02 | UI container image builds successfully | Deployment | — |
| T-B.03 | Operator container image builds successfully | Deployment | — |
| T-B.04 | All integration tests pass against PostgreSQL (not just SQLite) | Integration | Verifies no SQLite-specific assumptions |
| T-B.05 | API server starts and serves health endpoint in kind | E2E | — |
| T-B.06 | UI serves static assets and connects to API in kind | E2E | — |
| T-B.07 | Operator installs via AssetHub CR in kind | E2E | US-24 |
| T-B.08 | Full meta workflow: create entity type, add attributes, create catalog version, promote to testing → CRDs appear in kind | E2E | US-1, US-2, US-8, US-9 |
| T-B.09 | Full operational workflow: create instance, update, filter, delete with cascade | E2E | US-13–US-18 |
| T-B.10 | Demotion removes CRDs/CRs from kind cluster | E2E | US-12, US-25 |
| T-B.11 | Operator uninstall cleans up all resources | E2E | US-24 |

**Tests that must pass**: All Phase A tests (T-1.* through T-9.*, T-CV.*) plus T-B.01 through T-B.11.

**What cannot be tested**:
- OpenShift-specific features (Routes, OAuth, OLM)
- Real OpenShift RBAC (SubjectAccessReview against OCP identity provider)
- OCP console integration

**Human checkpoint**: After all Phase A + Phase B tests pass. Review before deploying to OCP.

---

### Environment Phase C: Remote OpenShift Cluster

**Environment**: Access to a remote OpenShift cluster.

**What is developed and tested**:
- OLM-based operator installation
- OpenShift Routes for API server and UI
- Real OpenShift RBAC (SubjectAccessReview against OCP identity provider with real users/service accounts)
- OAuth integration for UI authentication
- Production-grade PostgreSQL (via OCP operator or external)

**Additional test cases for Phase C**:

| ID | Test Case | Layer | Notes |
|----|-----------|-------|-------|
| T-C.01 | Operator installable via OLM on OCP | E2E | US-24: OLM installation |
| T-C.02 | API server accessible via OpenShift Route | E2E | — |
| T-C.03 | UI accessible via OpenShift Route | E2E | — |
| T-C.04 | RO user (real OCP service account) can GET, receives 403 on POST | E2E | US-22, US-23: real RBAC |
| T-C.05 | RW user (real OCP service account) can CRUD instances, cannot modify meta | E2E | US-23: real RBAC |
| T-C.06 | Admin user can modify meta, promote to production | E2E | US-23: real RBAC |
| T-C.07 | Super Admin can demote from production, modify production meta | E2E | US-23: real RBAC |
| T-C.08 | Full lifecycle on OCP: dev → test → prod → demote | E2E | US-9, US-10, US-12 |
| T-C.09 | UI OAuth login flow works with OCP identity provider | E2E | — |
| T-C.10 | Operator upgrade (new version) without data loss | E2E | US-24: rolling upgrades |
| T-C.11 | Operator uninstall via OLM cleanly removes all components | E2E | US-24: clean uninstall |

**Tests that must pass**: All Phase A + Phase B + Phase C tests (T-C.01 through T-C.85).

**Human checkpoint**: After all tests pass on OCP. Final acceptance.

---

### Entity Type Management — Backend + UI Tests

These tests cover the full entity type management feature: backend API handlers for attributes, associations, enums, and version history, plus UI pages for entity type detail, enum management, and delete confirmation.

#### Backend: Attribute Handler (T-C.12 through T-C.19)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-C.12 | GET /entity-types/:id/attributes → list attributes | API | 200 + attribute array |
| T-C.13 | POST /entity-types/:id/attributes → add string attribute | API | 201 + new version |
| T-C.14 | POST /entity-types/:id/attributes → add enum attribute with valid enum_id | API | 201 |
| T-C.15 | POST /entity-types/:id/attributes with missing name | API | 400 |
| T-C.16 | POST /entity-types/:id/attributes with duplicate name | API | 409 |
| T-C.17 | DELETE /entity-types/:id/attributes/:name → remove attribute | API | 204 |
| T-C.18 | PUT /entity-types/:id/attributes/reorder | API | 200 |
| T-C.19 | POST /entity-types/:id/attributes as RO | API | 403 |

#### Backend: Association Handler (T-C.20 through T-C.25)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-C.20 | GET /entity-types/:id/associations → list associations | API | 200 + association array |
| T-C.21 | POST /entity-types/:id/associations → create containment | API | 201 + new version |
| T-C.22 | POST /entity-types/:id/associations → create directional | API | 201 |
| T-C.23 | POST /entity-types/:id/associations → containment cycle | API | 422 cycle detected |
| T-C.24 | DELETE /entity-types/:id/associations/:name | API | 204 |
| T-C.25 | POST /entity-types/:id/associations as RO | API | 403 |

#### Backend: Enum Handler (T-C.26 through T-C.36)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-C.26 | GET /enums → list enums | API | 200 + enum array |
| T-C.27 | POST /enums → create enum with values | API | 201 |
| T-C.28 | POST /enums with missing name | API | 400 |
| T-C.29 | POST /enums with duplicate name | API | 409 |
| T-C.30 | GET /enums/:id → get enum by ID | API | 200 |
| T-C.31 | PUT /enums/:id → update enum name | API | 200 |
| T-C.32 | DELETE /enums/:id (unreferenced) | API | 204 |
| T-C.33 | DELETE /enums/:id (referenced by attribute) | API | 422 |
| T-C.34 | GET /enums/:id/values → list enum values | API | 200 + value array |
| T-C.35 | POST /enums/:id/values → add enum value | API | 201 |
| T-C.36 | PUT /enums/:id/values/reorder | API | 200 |

#### Backend: Version History Handler (T-C.37 through T-C.39)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-C.37 | GET /entity-types/:id/versions → version history | API | 200 + version array |
| T-C.38 | GET /entity-types/:id/versions/diff?v1=1&v2=2 → compare versions | API | 200 + diff |
| T-C.39 | GET /entity-types/:id/versions/diff with nonexistent version | API | 404 |

#### Backend: RBAC — Attributes (T-C.40 through T-C.45)

Role hierarchy: RO (0) < RW (1) < Admin (2) < SuperAdmin (3). Write endpoints require Admin+. Read allowed for all authenticated roles.

| ID | Test Case | Role | Method | Expected |
|----|-----------|------|--------|----------|
| T-C.40 | RO can list attributes | RO | GET | 200 |
| T-C.41 | RW cannot add attribute | RW | POST | 403 |
| T-C.42 | Admin can add attribute | Admin | POST | 201 |
| T-C.43 | SuperAdmin can add attribute | SuperAdmin | POST | 201 |
| T-C.44 | RO cannot remove attribute | RO | DELETE | 403 |
| T-C.45 | RO cannot reorder attributes | RO | PUT reorder | 403 |

#### Backend: RBAC — Associations (T-C.46 through T-C.50)

| ID | Test Case | Role | Method | Expected |
|----|-----------|------|--------|----------|
| T-C.46 | RO can list associations | RO | GET | 200 |
| T-C.47 | RW cannot create association | RW | POST | 403 |
| T-C.48 | Admin can create association | Admin | POST | 201 |
| T-C.49 | SuperAdmin can create association | SuperAdmin | POST | 201 |
| T-C.50 | RO cannot delete association | RO | DELETE | 403 |

#### Backend: RBAC — Enums (T-C.51 through T-C.59)

| ID | Test Case | Role | Method | Expected |
|----|-----------|------|--------|----------|
| T-C.51 | RO can list enums | RO | GET | 200 |
| T-C.52 | RO can get enum by ID | RO | GET :id | 200 |
| T-C.53 | RO can list enum values | RO | GET :id/values | 200 |
| T-C.54 | RW cannot create enum | RW | POST | 403 |
| T-C.55 | Admin can create enum | Admin | POST | 201 |
| T-C.56 | SuperAdmin can create enum | SuperAdmin | POST | 201 |
| T-C.57 | RO cannot update enum | RO | PUT | 403 |
| T-C.58 | RO cannot delete enum | RO | DELETE | 403 |
| T-C.59 | RO cannot add enum value | RO | POST values | 403 |

#### Backend: RBAC — Version History (T-C.60 through T-C.61)

Version history endpoints are read-only — all authenticated roles can access.

| ID | Test Case | Role | Method | Expected |
|----|-----------|------|--------|----------|
| T-C.60 | RO can list versions | RO | GET | 200 |
| T-C.61 | RO can compare versions | RO | GET diff | 200 |

#### UI: Delete Confirmation (T-C.62 through T-C.64)

| ID | Test Case | Layer |
|----|-----------|-------|
| T-C.62 | Click Delete → confirmation modal shows entity type name | UI |
| T-C.63 | Cancel confirmation → no deletion | UI |
| T-C.64 | Confirm deletion → API called, entity removed from list | UI |

#### UI: Entity Type Detail Page (T-C.65 through T-C.78)

| ID | Test Case | Layer |
|----|-----------|-------|
| T-C.65 | Navigate to detail page → shows name, description, version | UI |
| T-C.66 | Attributes tab lists attributes | UI |
| T-C.67 | Add string attribute via modal | UI |
| T-C.68 | Add enum attribute — enum selector shown when type=enum | UI |
| T-C.69 | Remove attribute | UI |
| T-C.70 | Reorder attributes with up/down buttons | UI |
| T-C.71 | Associations tab lists associations | UI |
| T-C.72 | Add containment association via modal | UI |
| T-C.73 | Cycle detection error displayed (422) | UI |
| T-C.74 | Remove association | UI |
| T-C.75 | Version history tab shows versions | UI |
| T-C.76 | Compare two versions shows diff | UI |
| T-C.77 | Copy entity type via modal | UI |
| T-C.78 | RO role hides add/remove controls | UI |

#### UI: Enum Management (T-C.79 through T-C.85)

| ID | Test Case | Layer |
|----|-----------|-------|
| T-C.79 | Enum list page shows enums | UI |
| T-C.80 | Create enum with initial values | UI |
| T-C.81 | Navigate to enum detail | UI |
| T-C.82 | Add value to enum | UI |
| T-C.83 | Remove value from enum | UI |
| T-C.84 | Reorder enum values | UI |
| T-C.85 | Delete referenced enum shows error | UI |

---

## Milestone 1: Database Layer — Meta Tables

### Entity Types and Versions (US-1, US-6)

| ID | Test Case | Layer | Acceptance Criteria |
|----|-----------|-------|-------------------|
| T-1.01 | Create entity type with name and description → row created, timestamps set | Integration | US-1: Admin can create a new entity type |
| T-1.02 | Create entity type → initial version row created at version=1 | Integration | US-1: created at version 1 |
| T-1.03 | Create entity type with duplicate name → unique constraint error | Integration | US-1: name must be unique |
| T-1.04 | Create second version of entity type → version=2, previous version intact | Integration | US-6: new version incremented by 1 |
| T-1.05 | List entity types with pagination → correct page size, ordering | Integration | — |
| T-1.06 | Get entity type by ID → returns current data | Integration | — |
| T-1.07 | Get entity type by name → returns matching type | Integration | — |
| T-1.08 | Delete entity type → cascades to all versions | Integration | — |

### Enums (US-5)

| ID | Test Case | Layer | Acceptance Criteria |
|----|-----------|-------|-------------------|
| T-1.09 | Create enum with ordered values → rows created with correct ordinals | Integration | US-5: create named enum with ordered list |
| T-1.10 | Create enum with duplicate name → unique constraint error | Integration | — |
| T-1.11 | Add value to enum → new value row, correct ordinal | Integration | US-5: update enum values (add) |
| T-1.12 | Remove value from enum → value row deleted, ordinals adjusted | Integration | US-5: update enum values (remove) |
| T-1.13 | Reorder enum values → ordinals updated correctly | Integration | US-5: update enum values (reorder) |
| T-1.14 | Create duplicate value within same enum → unique constraint error | Integration | — |

### Attributes (US-1, US-4, US-5, US-6)

| ID | Test Case | Layer | Acceptance Criteria |
|----|-----------|-------|-------------------|
| T-1.15 | Create attribute with type=string → stored correctly | Integration | US-1: attribute with type string |
| T-1.16 | Create attribute with type=number → stored correctly | Integration | US-1: attribute with type number |
| T-1.17 | Create attribute with type=enum referencing existing enum → FK set | Integration | US-1: attribute with type enum |
| T-1.18 | Create attribute with duplicate name within same version → unique constraint error | Integration | — |
| T-1.19 | Same attribute name on different entity type versions → allowed | Integration | — |
| T-1.20 | Reorder attributes → ordinals updated | Integration | — |
| T-1.21 | Copy attributes from version V1 to new version V2 → all attributes duplicated | Integration | US-6: copy-on-write |
| T-1.22 | Delete enum referenced by attribute → rejected (FK constraint or application check) | Integration | US-5: cannot delete if referenced |
| T-1.23 | List attributes by entity type version → returns only that version's attributes | Integration | — |

### Associations (US-2, US-6)

| ID | Test Case | Layer | Acceptance Criteria |
|----|-----------|-------|-------------------|
| T-1.24 | Create containment association → stored with correct type and roles | Integration | US-2: define containment |
| T-1.25 | Create directional reference → stored with source→target direction | Integration | US-2: define directional reference |
| T-1.26 | Create bidirectional reference → stored correctly | Integration | US-2: define bidirectional reference |
| T-1.27 | Copy associations from version V1 to V2 → all associations duplicated | Integration | US-6: copy-on-write includes associations |
| T-1.28 | List associations by entity type version → returns only that version's associations | Integration | — |
| T-1.29 | Delete association → row removed | Integration | — |

### Catalog Versions (US-8, US-9, US-10, US-12)

| ID | Test Case | Layer | Acceptance Criteria |
|----|-----------|-------|-------------------|
| T-1.30 | Create catalog version → created with lifecycle_stage=development | Integration | US-8: created in Development |
| T-1.31 | Create catalog version with duplicate label → unique constraint error | Integration | US-8: unique identifier |
| T-1.32 | Add pin (catalog version + entity type version) → stored | Integration | US-8: records entity type version tuples |
| T-1.33 | Add duplicate pin → unique constraint error | Integration | — |
| T-1.34 | Transition development→testing → stage updated, transition logged | Integration | US-9: status updated to Testing |
| T-1.35 | Transition testing→production → stage updated, transition logged | Integration | US-10: promoted to Production |
| T-1.36 | Transition production→testing → stage updated, transition logged | Integration | US-12: demote from Production |
| T-1.37 | Transition production→development → stage updated, transition logged | Integration | US-12: demote to Development |
| T-1.38 | Invalid transition (development→production directly) → rejected | Integration | — |
| T-1.39 | List transitions for catalog version → returns history in order | Integration | — |

---

## Milestone 2: Database Layer — Data Tables

### Entity Instances (US-13, US-14, US-15, US-16)

| ID | Test Case | Layer | Acceptance Criteria |
|----|-----------|-------|-------------------|
| T-2.01 | Create top-level instance → version=1, ID generated, timestamps set | Integration | US-13: created at version 1 |
| T-2.02 | Create instance with duplicate name (same type, same catalog version, no parent) → unique constraint error | Integration | US-13: name unique within scope |
| T-2.03 | Create contained instance with parent → parent_instance_id set | Integration | US-16: create within containing entity |
| T-2.04 | Create contained instance with same name under different parents → allowed | Integration | US-16: unique within parent namespace |
| T-2.05 | Create contained instance with same name under same parent → rejected | Integration | US-16: unique within parent namespace |
| T-2.06 | Create contained instance with non-existent parent → FK violation | Integration | US-16: parent must exist |
| T-2.07 | Update instance → version incremented by 1 | Integration | US-14: auto-increment version |
| T-2.08 | Soft delete instance → deleted_at set, instance excluded from normal queries | Integration | US-15: delete instance |
| T-2.09 | List instances by type and catalog version with pagination | Integration | — |
| T-2.10 | List instances by parent → returns only children of that parent | Integration | — |

### Instance Attribute Values (US-13, US-14)

| ID | Test Case | Layer | Acceptance Criteria |
|----|-----------|-------|-------------------|
| T-2.11 | Set string attribute value → value_string populated | Integration | US-13: attribute values |
| T-2.12 | Set number attribute value → value_number populated | Integration | US-13: attribute values |
| T-2.13 | Set enum attribute value → value_enum populated | Integration | US-13: attribute values |
| T-2.14 | Set values for version 1, update for version 2 → both versions retained | Integration | US-14: previous version retained |
| T-2.15 | Get current values for instance → returns latest version values | Integration | — |
| T-2.16 | Get values for specific instance version → returns that version's values | Integration | US-14: previous version retained |
| T-2.17 | Unique constraint on (instance_id, instance_version, attribute_id) → prevents duplicate entries | Integration | — |

### Association Links (US-19, US-20)

| ID | Test Case | Layer | Acceptance Criteria |
|----|-----------|-------|-------------------|
| T-2.18 | Create association link between two instances → stored with correct IDs | Integration | — |
| T-2.19 | Get forward references for instance → returns target instances | Integration | US-19: forward references |
| T-2.20 | Get reverse references for instance → returns source instances | Integration | US-20: reverse references |
| T-2.21 | Delete association link → removed | Integration | — |
| T-2.22 | Filter forward references by association type → returns subset | Integration | US-19: filter by type |

---

## Milestone 3: Service Layer — Meta Operations

### Entity Type Service (US-1, US-3, US-6)

| ID | Test Case | Layer | Acceptance Criteria |
|----|-----------|-------|-------------------|
| T-3.01 | CreateEntityType → creates type + V1 in DB, returns domain model | Unit | US-1: Admin can create |
| T-3.02 | CreateEntityType with empty name → validation error | Unit | — |
| T-3.03 | CreateEntityType with duplicate name → conflict error | Unit | US-1: name unique |
| T-3.04 | UpdateEntityType (change description) → creates V2 with all attributes and associations copied from V1 | Unit | US-6: auto-version, copy-on-write |
| T-3.05 | After update, V1 attributes/associations remain unchanged | Unit | US-6: previous version intact |
| T-3.06 | CopyEntityType → new type at V1 with source attributes, no associations | Unit | US-3: copy creates V1, no associations |
| T-3.07 | CopyEntityType → source type unchanged | Unit | US-3: source unchanged |
| T-3.08 | CopyEntityType with duplicate target name → conflict error | Unit | — |
| T-3.09 | DeleteEntityType → calls repository delete | Unit | — |
| T-3.10 | ListEntityTypes with filters → delegates to repository with correct params | Unit | — |

### Attribute Service (US-4, US-6)

| ID | Test Case | Layer | Acceptance Criteria |
|----|-----------|-------|-------------------|
| T-3.11 | AddAttribute → creates new entity type version, copies existing attrs + adds new one | Unit | US-6: auto-version |
| T-3.12 | AddAttribute with duplicate name → conflict error | Unit | US-4: rejected with conflict |
| T-3.13 | AddAttribute with enum type, valid enum_id → accepted | Unit | — |
| T-3.14 | AddAttribute with enum type, invalid enum_id → validation error | Unit | — |
| T-3.15 | RemoveAttribute → creates new version without the removed attribute | Unit | US-6: auto-version |
| T-3.16 | CopyAttributesFromType → copies selected attributes, rejects name conflicts | Unit | US-4: copy, conflict detection |
| T-3.17 | CopyAttributesFromType → target version incremented | Unit | US-4: target version incremented |
| T-3.18 | CopyAttributesFromType → source type unchanged | Unit | US-4: independent copies |
| T-3.19 | ReorderAttributes → updates ordinals on current version | Unit | — |

### Association Service (US-2)

| ID | Test Case | Layer | Acceptance Criteria |
|----|-----------|-------|-------------------|
| T-3.20 | CreateAssociation (containment) → creates new version with association added | Unit | US-2: define containment |
| T-3.21 | CreateAssociation (containment) that would create direct cycle (A→B→A) → rejected | Unit | US-2: validated for cycles |
| T-3.22 | CreateAssociation (containment) that would create indirect cycle (A→B→C→A) → rejected | Unit | US-2: validated for cycles |
| T-3.23 | CreateAssociation (containment) self-reference → rejected | Unit | US-2: validated for cycles |
| T-3.24 | CreateAssociation (directional reference) → no cycle validation needed, succeeds | Unit | US-2: define directional reference |
| T-3.25 | CreateAssociation (bidirectional reference) → no cycle validation needed, succeeds | Unit | US-2: define bidirectional reference |
| T-3.26 | CreateAssociation → entity type version incremented | Unit | US-2: version incremented |
| T-3.27 | DeleteAssociation → creates new version without the association | Unit | — |
| T-3.28 | Cycle detection algorithm with complex DAG (5+ nodes, valid) → no false positives | Unit | US-2: no false cycle detection |

### Enum Service (US-5)

| ID | Test Case | Layer | Acceptance Criteria |
|----|-----------|-------|-------------------|
| T-3.29 | CreateEnum → stored with values in correct order | Unit | US-5: create named enum |
| T-3.30 | UpdateEnum (add value) → value added, existing unchanged | Unit | US-5: update values |
| T-3.31 | UpdateEnum (remove value) → value removed | Unit | US-5: update values |
| T-3.32 | DeleteEnum with no references → succeeds | Unit | — |
| T-3.33 | DeleteEnum with active attribute references → rejected with list of referencing attributes | Unit | US-5: cannot delete if referenced |
| T-3.34 | GetReferencingAttributes → returns all attributes across all entity type versions that reference this enum | Unit | US-5: multiple attributes can reference same enum |

### Catalog Version Service (US-8, US-9, US-10, US-11, US-12)

| ID | Test Case | Layer | Acceptance Criteria |
|----|-----------|-------|-------------------|
| T-3.35 | CreateCatalogVersion → created in Development stage | Unit | US-8: created in Development |
| T-3.36 | CreateCatalogVersion with pins → pins recorded correctly | Unit | US-8: records entity version tuples |
| T-3.37 | Promote dev→test as RW → succeeds | Unit | US-9: RW can promote |
| T-3.38 | Promote dev→test as RO → rejected (permission error) | Unit | US-9: RO cannot promote |
| T-3.39 | Demote test→dev as RW → succeeds | Unit | US-9: RW can demote |
| T-3.40 | Promote test→prod as Admin → succeeds | Unit | US-10: Admin can promote |
| T-3.41 | Promote test→prod as RW → rejected (permission error) | Unit | US-10: RW cannot promote to prod |
| T-3.42 | Demote prod→test as Super Admin → succeeds | Unit | US-12: Super Admin can demote |
| T-3.43 | Demote prod→test as Admin → rejected (permission error) | Unit | US-12: Admin cannot demote from prod |
| T-3.44 | Promote dev→prod directly → rejected (invalid transition) | Unit | — |
| T-3.45 | All transitions recorded in lifecycle_transitions table | Unit | — |
| T-3.46 | Modify entity definition in production catalog as Super Admin → succeeds | Unit | US-11: Super Admin can modify |
| T-3.47 | Modify entity definition in production catalog as Admin → rejected | Unit | US-11: Admin gets 403 |

### Version History Service (US-36)

| ID | Test Case | Layer | Acceptance Criteria |
|----|-----------|-------|-------------------|
| T-3.48 | GetVersionHistory → returns all versions in chronological order | Unit | US-36: version history |
| T-3.49 | CompareVersions (attribute added) → diff shows addition (green) | Unit | US-36: highlights added |
| T-3.50 | CompareVersions (attribute removed) → diff shows removal (red) | Unit | US-36: highlights removed |
| T-3.51 | CompareVersions (attribute modified) → diff shows modification (yellow) | Unit | US-36: highlights modified |
| T-3.52 | CompareVersions (association added/removed) → diff includes association changes | Unit | US-36: association changes |
| T-3.53 | CompareVersions with same version → empty diff | Unit | — |

---

## Milestone 4: Service Layer — Operational

### Entity Instance Service (US-13, US-14, US-15, US-21)

| ID | Test Case | Layer | Acceptance Criteria |
|----|-----------|-------|-------------------|
| T-4.01 | CreateInstance with valid attributes → instance at V1, ID generated | Unit | US-13: created at V1 |
| T-4.02 | CreateInstance with string value for number attribute → validation error | Unit | US-13: validated against type |
| T-4.03 | CreateInstance with invalid enum value → validation error | Unit | US-13: validated against type |
| T-4.04 | CreateInstance with duplicate name in same scope → conflict error | Unit | US-13: name unique within scope |
| T-4.05 | CreateInstance scoped to non-existent catalog version → error | Unit | US-21: invalid catalog version |
| T-4.06 | UpdateInstance → version incremented, previous values retained | Unit | US-14: auto-increment, history |
| T-4.07 | UpdateInstance with stale version (optimistic lock) → 409 conflict | Unit | — |
| T-4.08 | DeleteInstance (no children) → soft deleted | Unit | US-15: delete by ID |
| T-4.09 | GetInstance scoped to catalog version A → returns data from version A only | Unit | US-21: consistent view |

### Containment Operations (US-15, US-16, US-18)

| ID | Test Case | Layer | Acceptance Criteria |
|----|-----------|-------|-------------------|
| T-4.10 | CreateContainedInstance → parent_instance_id set, name unique within parent | Unit | US-16: create within parent |
| T-4.11 | CreateContainedInstance with non-existent parent → 404 error | Unit | US-16: rejected if parent doesn't exist |
| T-4.12 | ListContainedInstances → returns only direct children of parent | Unit | US-18: accessible via parent URL |
| T-4.13 | CascadeDelete (1 level) → parent and child deleted | Unit | US-15: cascade delete |
| T-4.14 | CascadeDelete (3 levels: A→B→C) → all three deleted | Unit | US-15: multi-level cascade |
| T-4.15 | CascadeDelete atomicity → if child delete fails, nothing is deleted | Unit | US-15: atomic operation |
| T-4.16 | Dangling reference notification → when deleted entity is referenced, system notifies | Unit | US-15: dangling references handled |

### Reference Operations (US-19, US-20)

| ID | Test Case | Layer | Acceptance Criteria |
|----|-----------|-------|-------------------|
| T-4.17 | GetForwardReferences → returns all target instances | Unit | US-19: retrieve referenced entities |
| T-4.18 | GetForwardReferences filtered by type → returns subset | Unit | US-19: filter by type |
| T-4.19 | GetForwardReferences includes both directional and bidirectional | Unit | US-19: both types included |
| T-4.20 | GetReverseReferences → returns all source instances | Unit | US-20: find referring entities |
| T-4.21 | GetReverseReferences includes both directional and bidirectional | Unit | US-20: both types included |
| T-4.22 | Forward reference response includes type, ID, name | Unit | US-19: response includes type, ID, name |

### Filtering and Pagination (US-17)

| ID | Test Case | Layer | Acceptance Criteria |
|----|-----------|-------|-------------------|
| T-4.23 | Filter by string attribute (exact match) → correct results | Unit | US-17: filter by attribute |
| T-4.24 | Filter by number attribute (gt, lt, eq) → correct results | Unit | US-17: filter by attribute |
| T-4.25 | Filter by enum attribute → correct results | Unit | US-17: filter by attribute |
| T-4.26 | Filter by common attribute (name) → correct results | Unit | US-17: common or custom |
| T-4.27 | Multiple filters (AND) → intersection of results | Unit | US-17: combined filters |
| T-4.28 | Sort ascending by attribute → correct order | Unit | US-17: sort ascending |
| T-4.29 | Sort descending by attribute → correct order | Unit | US-17: sort descending |
| T-4.30 | Pagination → correct page boundaries, total count | Unit | US-17: pagination support |
| T-4.31 | Filter by non-existent attribute → error (not empty result) | Unit | US-17: error for non-existent attr |

---

## Milestone 5: API Layer — Meta API

### RBAC Middleware (US-22, US-23)

| ID | Test Case | Layer | Acceptance Criteria |
|----|-----------|-------|-------------------|
| T-5.01 | Request with valid RO token → 200 on GET endpoints | API | US-22: RO can GET |
| T-5.02 | Request with RO token on POST → 403 | API | US-22: RO gets 403 on write |
| T-5.03 | Request with RW token on meta POST → 403 | API | US-23: RW cannot modify meta |
| T-5.04 | Request with Admin token on meta POST → 200 | API | US-23: Admin can modify meta |
| T-5.05 | Request with no token → 401 | API | — |
| T-5.06 | Request with invalid token → 401 | API | — |
| T-5.07 | Request with Super Admin token on production meta modify → 200 | API | US-11: Super Admin can modify prod |
| T-5.08 | Request with Admin token on production meta modify → 403 | API | US-11: Admin gets 403 |

### Entity Type Endpoints (US-1, US-3, US-6)

| ID | Test Case | Layer | Acceptance Criteria |
|----|-----------|-------|-------------------|
| T-5.09 | POST /entity-types with valid body → 201, type returned with V1 | API | US-1: create entity type |
| T-5.10 | POST /entity-types with missing name → 400 | API | — |
| T-5.11 | POST /entity-types with duplicate name → 409 | API | US-1: name unique |
| T-5.12 | GET /entity-types → 200, paginated list | API | — |
| T-5.13 | GET /entity-types?name=foo → 200, filtered results | API | — |
| T-5.14 | GET /entity-types/{id} → 200, full entity type | API | — |
| T-5.15 | GET /entity-types/{nonexistent} → 404 | API | — |
| T-5.16 | PUT /entity-types/{id} → 200, new version created | API | US-6: auto-version |
| T-5.17 | PUT /entity-types/{id} with stale version → 409 | API | — |
| T-5.18 | DELETE /entity-types/{id} → 204 | API | — |
| T-5.19 | POST /entity-types/{id}/copy with new name → 201, new type at V1 | API | US-3: copy entity type |
| T-5.20 | POST /entity-types/{id}/copy with duplicate name → 409 | API | — |
| T-5.21 | POST /entity-types as RO → 403 | API | US-1: RO cannot create |
| T-5.22 | POST /entity-types as RW → 403 | API | US-1: RW cannot create |

### Attribute Endpoints (US-4, US-28)

| ID | Test Case | Layer | Acceptance Criteria |
|----|-----------|-------|-------------------|
| T-5.23 | POST /entity-types/{id}/attributes → 201, attribute added, type version incremented | API | US-6: auto-version |
| T-5.24 | POST /entity-types/{id}/attributes with duplicate name → 409 | API | US-4: conflict error |
| T-5.25 | PUT /entity-types/{id}/attributes/{attrId} → 200, updated | API | — |
| T-5.26 | DELETE /entity-types/{id}/attributes/{attrId} → 204, version incremented | API | — |
| T-5.27 | POST /entity-types/{id}/attributes/copy → 200, attributes copied | API | US-4: copy attributes |
| T-5.28 | PUT /entity-types/{id}/attributes/reorder → 200, ordinals updated | API | — |

### Association Endpoints (US-2)

| ID | Test Case | Layer | Acceptance Criteria |
|----|-----------|-------|-------------------|
| T-5.29 | POST /associations (containment) → 201 | API | US-2: define containment |
| T-5.30 | POST /associations (containment creating cycle) → 422 with error message | API | US-2: cycle rejected |
| T-5.31 | POST /associations (directional) → 201 | API | US-2: define directional |
| T-5.32 | POST /associations (bidirectional) → 201 | API | US-2: define bidirectional |
| T-5.33 | GET /entity-types/{id}/associations → 200, list of associations | API | — |
| T-5.34 | DELETE /associations/{id} → 204, version incremented | API | — |

### Enum Endpoints (US-5)

| ID | Test Case | Layer | Acceptance Criteria |
|----|-----------|-------|-------------------|
| T-5.35 | POST /enums → 201, enum with values | API | US-5: create enum |
| T-5.36 | GET /enums → 200, paginated list with reference counts | API | — |
| T-5.37 | PUT /enums/{id} → 200, values updated | API | US-5: update values |
| T-5.38 | DELETE /enums/{id} (no references) → 204 | API | — |
| T-5.39 | DELETE /enums/{id} (with references) → 422 with list of referencing attributes | API | US-5: cannot delete if referenced |

### Catalog Version Endpoints (US-8, US-9, US-10, US-12)

| ID | Test Case | Layer | Acceptance Criteria |
|----|-----------|-------|-------------------|
| T-5.40 | POST /catalog-versions → 201, created in Development | API | US-8: create catalog version |
| T-5.41 | POST /catalog-versions as RO → 403 | API | US-8: RO cannot create |
| T-5.42 | POST /catalog-versions/{id}/promote (dev→test) as RW → 200 | API | US-9: RW can promote |
| T-5.43 | POST /catalog-versions/{id}/promote (dev→test) as RO → 403 | API | US-9: RO cannot promote |
| T-5.44 | POST /catalog-versions/{id}/demote (test→dev) as RW → 200 | API | US-9: RW can demote |
| T-5.45 | POST /catalog-versions/{id}/promote (test→prod) as Admin → 200 | API | US-10: Admin can promote to prod |
| T-5.46 | POST /catalog-versions/{id}/promote (test→prod) as RW → 403 | API | US-10: RW cannot promote to prod |
| T-5.47 | POST /catalog-versions/{id}/demote (prod→test) as Super Admin → 200 | API | US-12: Super Admin can demote |
| T-5.48 | POST /catalog-versions/{id}/demote (prod→test) as Admin → 403 | API | US-12: Admin cannot demote from prod |
| T-5.49 | GET /catalog-versions/{id}/transitions → 200, transition history | API | — |

### Version History Endpoints

| ID | Test Case | Layer | Acceptance Criteria |
|----|-----------|-------|-------------------|
| T-5.50 | GET /entity-types/{id}/versions → 200, all versions in order | API | — |
| T-5.51 | GET /entity-types/{id}/versions/{v1}/compare/{v2} → 200, diff | API | — |

---

## Milestone 6: API Layer — Operational API

### Instance CRUD (US-13, US-14, US-15, US-21)

| ID | Test Case | Layer | Acceptance Criteria |
|----|-----------|-------|-------------------|
| T-6.01 | POST /catalog/{cv}/{type} → 201, instance at V1 | API | US-13: create instance |
| T-6.02 | POST /catalog/{cv}/{type} with invalid attribute values → 422 | API | US-13: validated against type |
| T-6.03 | POST /catalog/{cv}/{type} as RO → 403 | API | US-13: RO cannot create |
| T-6.04 | GET /catalog/{cv}/{type} → 200, paginated list | API | US-17: list entities |
| T-6.05 | GET /catalog/{cv}/{type}?filter=name:foo&sort=name:asc → 200, filtered+sorted | API | US-17: filter and sort |
| T-6.06 | GET /catalog/{cv}/{type}/{id} → 200, instance with version | API | — |
| T-6.07 | PUT /catalog/{cv}/{type}/{id} → 200, version incremented | API | US-14: auto-increment version |
| T-6.08 | PUT /catalog/{cv}/{type}/{id} with stale version → 409 | API | — |
| T-6.09 | DELETE /catalog/{cv}/{type}/{id} → 204 | API | US-15: delete instance |
| T-6.10 | GET /catalog/{invalid-cv}/{type} → 404 | API | US-21: invalid catalog version |
| T-6.11 | GET /catalog/{demoted-cv}/{type} → appropriate error | API | US-21: demoted catalog version |
| T-6.12 | Instance created in catalog V1 not visible in catalog V2 | API | US-21: consistent view |

### Containment Traversal (US-16, US-18)

| ID | Test Case | Layer | Acceptance Criteria |
|----|-----------|-------|-------------------|
| T-6.13 | POST /catalog/{cv}/{parent}/{id}/{child-type} → 201, contained instance | API | US-16: create contained |
| T-6.14 | GET /catalog/{cv}/{parent}/{id}/{child-type} → 200, list children | API | US-18: access via parent URL |
| T-6.15 | GET /catalog/{cv}/{parent}/{id}/{child-type}/{name} → 200, specific child | API | US-18: individual contained entity |
| T-6.16 | GET /catalog/{cv}/{parent}/{nonexistent}/{child-type} → 404 | API | US-18: 404 for nonexistent parent |
| T-6.17 | GET /catalog/{cv}/a/{id}/b/{id}/c → 200, multi-level containment | API | US-18: multi-level supported |
| T-6.18 | DELETE /catalog/{cv}/{parent}/{id} → cascades to children | API | US-15: cascade delete |
| T-6.19 | Filtering/sorting on contained entity listing → works | API | US-18: filter/sort apply |

### Reference Traversal (US-19, US-20)

| ID | Test Case | Layer | Acceptance Criteria |
|----|-----------|-------|-------------------|
| T-6.20 | GET /catalog/{cv}/{type}/{id}/references → 200, forward refs | API | US-19: retrieve references |
| T-6.21 | GET /catalog/{cv}/{type}/{id}/references/{ref-type} → 200, filtered | API | US-19: filter by type |
| T-6.22 | Response includes type, ID, name for each reference | API | US-19: response includes type, ID, name |
| T-6.23 | Reverse reference query → returns referring entities | API | US-20: find referring entities |
| T-6.24 | Reverse reference response includes type, ID, name | API | US-20: response includes type, ID, name |

---

## Milestone 7: UI — Meta Operations

### Entity Type List (US-26)

| ID | Test Case | Layer | Acceptance Criteria |
|----|-----------|-------|-------------------|
| T-7.01 | List renders all entity types with name, version, description, counts | UI | US-26: list displays all types |
| T-7.02 | Filter by name (text input) → list updates | UI | US-26: filtering by name |
| T-7.03 | Sort by name/version → list reorders | UI | US-26: sorting |
| T-7.04 | Click row → navigates to detail view | UI | US-26: links to detail |
| T-7.05 | Create/copy buttons visible for Admin, hidden for RO/RW | UI | US-26: RO without create/copy; US-38 |

### Entity Type Detail (US-27)

| ID | Test Case | Layer | Acceptance Criteria |
|----|-----------|-------|-------------------|
| T-7.06 | Detail shows name, description, version, attributes, associations | UI | US-27: full definition |
| T-7.07 | Common attributes shown read-only | UI | US-27: read-only common attrs |
| T-7.08 | Inline edit name/description → saves on explicit save | UI | US-27: edit inline |
| T-7.09 | Edit controls disabled for RO/RW users | UI | US-27: read-only for RO/RW |
| T-7.10 | Edit controls disabled with message when in production (non-Super Admin) | UI | US-27: production protection; US-38 |

### Attribute Management (US-28, US-29)

| ID | Test Case | Layer | Acceptance Criteria |
|----|-----------|-------|-------------------|
| T-7.11 | Add attribute dialog: name, description, type dropdown | UI | US-28: add via dialog |
| T-7.12 | Type dropdown shows string, number, and existing enum names | UI | US-28: enum selection |
| T-7.13 | Duplicate name shows inline error before submit | UI | US-28: uniqueness validated; US-37 |
| T-7.14 | Edit attribute inline → saves correctly | UI | US-28: edit existing |
| T-7.15 | Remove attribute shows confirmation dialog | UI | US-28: confirmation on remove |
| T-7.16 | Drag-and-drop reorder updates ordinals | UI | US-28: reorder |
| T-7.17 | Copy picker shows attributes from other types | UI | US-29: browse other types |
| T-7.18 | Copy picker disables conflicting attributes (same name) | UI | US-29: conflict indicator |
| T-7.19 | Inline enum creation from type dropdown | UI | US-28: create enum inline |

### Association Management (US-31, US-32)

| ID | Test Case | Layer | Acceptance Criteria |
|----|-----------|-------|-------------------|
| T-7.20 | Association list grouped by type (containment vs. reference) | UI | US-31: grouped by type |
| T-7.21 | Add association dialog: target type, type, direction | UI | US-31: add via dialog |
| T-7.22 | Containment cycle detection prevents submission in real-time | UI | US-31: real-time cycle detection; US-37 |
| T-7.23 | Remove association shows confirmation | UI | US-31: confirmation on remove |
| T-7.24 | Association map renders nodes and labeled edges | UI | US-32: graph visualization |
| T-7.25 | Containment edges visually distinct from reference edges | UI | US-32: visual distinction |
| T-7.26 | Click node navigates to entity type detail | UI | US-32: interactive |
| T-7.27 | Map handles 10+ entity types (zoom/pan) | UI | US-32: handles 10+ types |

### Enum Management (US-33)

| ID | Test Case | Layer | Acceptance Criteria |
|----|-----------|-------|-------------------|
| T-7.28 | Enum list shows name, value count, referencing attributes | UI | US-33: list view |
| T-7.29 | Create enum form: name, ordered values | UI | US-33: create |
| T-7.30 | Edit enum: add/remove/reorder values | UI | US-33: edit |
| T-7.31 | Delete blocked when referenced (shows referencing attributes) | UI | US-33: delete blocked |
| T-7.32 | Inline creation available from attribute type dropdown | UI | US-33: inline creation |

### Catalog Version Management (US-41, US-35)

| ID | Test Case | Layer | Acceptance Criteria |
|----|-----------|-------|-------------------|
| T-7.33 | Create UI shows all types with version dropdowns, latest pre-selected | UI | US-41: version picker with defaults |
| T-7.34 | Summary/review step shows full bill of materials | UI | US-41: review before confirm |
| T-7.35 | Detail shows identifier, lifecycle badge (color-coded), BOM | UI | US-35: detail view |
| T-7.36 | Development stage: "Promote to Testing" visible for RW+ | UI | US-35: role-based actions |
| T-7.37 | Testing stage: "Promote to Production" visible for Admin+, "Demote" for RW+ | UI | US-35: role-based actions |
| T-7.38 | Production stage: "Demote" visible for Super Admin only | UI | US-35: Super Admin only |
| T-7.39 | Promotion shows confirmation dialog with side effects | UI | US-35: confirmation dialog |
| T-7.40 | Failed promotion shows error, stage unchanged | UI | US-35: error display |
| T-7.41 | Transition history shows who/when | UI | US-35: history |

### Version History (US-36)

| ID | Test Case | Layer | Acceptance Criteria |
|----|-----------|-------|-------------------|
| T-7.42 | Version list shows version number, date, change summary | UI | US-36: version list |
| T-7.43 | Click version → read-only detail | UI | US-36: view past version |
| T-7.44 | Select two versions → side-by-side diff | UI | US-36: comparison |
| T-7.45 | Diff highlights: green (added), red (removed), yellow (modified) | UI | US-36: color-coded diff |
| T-7.46 | Copy from past version action available | UI | US-36: copy from history |

### Validation and Role-Awareness (US-37, US-38)

| ID | Test Case | Layer | Acceptance Criteria |
|----|-----------|-------|-------------------|
| T-7.47 | Required field left empty → inline error shown | UI | US-37: inline validation |
| T-7.48 | Successful save → toast notification | UI | US-37: success feedback |
| T-7.49 | Failed API call → human-readable error (not raw response) | UI | US-37: human-readable errors |
| T-7.50 | Destructive operation → confirmation dialog | UI | US-37: confirmation required |
| T-7.51 | RO user: no create/edit/delete/promote controls visible | UI | US-38: RO read-only |
| T-7.52 | RW user: same as RO for meta operations | UI | US-38: RW same as RO for meta |
| T-7.53 | State-disabled controls show grayed out with tooltip | UI | US-38: disabled with tooltip |
| T-7.54 | Role-hidden controls have no placeholder | UI | US-38: no placeholder |

---

## Milestone 8: UI — Operational Pages

| ID | Test Case | Layer | Acceptance Criteria |
|----|-----------|-------|-------------------|
| T-8.01 | Catalog version selector sets context for all operational views | UI | US-21: scoped to catalog version |
| T-8.02 | Entity type navigation sidebar populated from catalog version | UI | — |
| T-8.03 | Instance list with dynamic columns from entity type definition | UI | US-17: list entities |
| T-8.04 | Instance create form generates fields from entity type attributes | UI | US-13: create instance |
| T-8.05 | Enum attributes render as dropdown in form | UI | — |
| T-8.06 | Delete shows cascade warning for containing entities | UI | US-15: cascade delete |
| T-8.07 | Breadcrumb navigation for containment hierarchy | UI | US-18: navigate hierarchy |
| T-8.08 | Contained instances listed within parent detail | UI | US-18: accessible via parent |
| T-8.09 | References tab shows forward references with links | UI | US-19: understand dependencies |
| T-8.10 | References tab shows reverse references (may load separately) | UI | US-20: impact analysis |

---

## Milestone 9: Operator

| ID | Test Case | Layer | Acceptance Criteria |
|----|-----------|-------|-------------------|
| T-9.01 | Create AssetHub CR → Deployment, Service, UI Deployment created | Operator | US-24: deploys all components |
| T-9.02 | Delete AssetHub CR → all managed resources cleaned up | Operator | US-24: clean uninstall |
| T-9.03 | AssetHub CR update (change replicas) → Deployment updated | Operator | US-24: manages lifecycle |
| T-9.04 | Generate CRD from entity type definition → valid K8s CRD YAML | Operator | US-25: CRD generation |
| T-9.05 | Generate CR from entity instance → valid K8s CR YAML | Operator | US-25: CR generation |
| T-9.06 | Promotion to Testing → CRDs/CRs applied to cluster | Operator | US-25: reconciliation on promotion |
| T-9.07 | Demotion → CRDs/CRs removed from cluster | Operator | US-25: cleanup on demotion |
| T-9.08 | Reconciliation failure → error in CR status conditions | Operator | US-25: errors reported |
| T-9.09 | Operator does not modify the database | Operator | US-25: reads CRs only |

---

## CatalogVersion Discovery CRD

This feature spans multiple layers: operator types → pure reconciler → controller → infrastructure (K8s client) → service → config. Test case IDs use the `T-CV.{sequence}` format.

### CatalogVersion CRD Types (`catalogversion_types_test.go`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-CV.01 | CatalogVersion DeepCopy preserves all fields including EntityTypes slice | Unit | All fields match, slices are independent copies |
| T-CV.02 | CatalogVersion DeepCopy of nil returns nil | Unit | Returns nil |
| T-CV.03 | CatalogVersion DeepCopyObject returns valid runtime.Object | Unit | Non-nil, correct type |
| T-CV.04 | CatalogVersionList DeepCopy preserves items | Unit | Items match, mutation of copy doesn't affect original |
| T-CV.05 | CatalogVersionList DeepCopy of nil returns nil | Unit | Returns nil |
| T-CV.06 | AddToScheme registers CatalogVersion and CatalogVersionList | Unit | scheme.New succeeds for both GVKs |

### ClusterRole in Reconciler (`reconciler_test.go`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-CV.07 | ReconcileAssetHub with clusterRole="production" | Operator | ConfigMap has CLUSTER_ROLE=production |
| T-CV.08 | ReconcileAssetHub with clusterRole="" | Operator | Defaults to CLUSTER_ROLE=development |
| T-CV.09 | ReconcileAssetHub with clusterRole="testing" | Operator | ConfigMap has CLUSTER_ROLE=testing |
| T-CV.10 | ReconcileCatalogVersionStatus with lifecycleStage="testing" | Operator | ready=true, message set |
| T-CV.11 | ReconcileCatalogVersionStatus with lifecycleStage="production" | Operator | ready=true, message set |

**Regression:** T-D.06 updated — ConfigMap now has 7 entries (adds CLUSTER_ROLE).

### Controller with CatalogVersion (`controller_test.go`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-CV.12 | Reconcile with clusterRole="production" creates ConfigMap with CLUSTER_ROLE=production | Operator | ConfigMap data correct |
| T-CV.13 | Reconcile sets owner reference on existing CatalogVersion CR | Operator | OwnerReferences non-empty, points to AssetHub |
| T-CV.14 | Reconcile updates CatalogVersion status to ready=true | Operator | Status.Ready=true |
| T-CV.15 | Reconcile with no CatalogVersion CRs in namespace | Operator | Succeeds without error |

### K8s CR Manager (`cr_manager_test.go`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-CV.16 | CreateOrUpdate creates new CatalogVersion CR with correct spec | Unit | CR exists with matching fields |
| T-CV.17 | CreateOrUpdate updates existing CatalogVersion CR | Unit | CR updated idempotently |
| T-CV.18 | Delete removes existing CatalogVersion CR | Unit | CR no longer exists |
| T-CV.19 | Delete of non-existent CR returns no error | Unit | nil error |
| T-CV.20 | CreateOrUpdate sets all three annotations | Unit | Annotations present with correct values |

### SanitizeK8sName (`cr_manager_test.go`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-CV.21 | SanitizeK8sName("Release 2.3") → "release-2-3" | Unit | Valid K8s name |
| T-CV.22 | SanitizeK8sName with uppercase, underscores, leading/trailing special chars | Unit | Valid K8s name |

### CatalogVersionService K8s Integration (`catalog_version_service_test.go`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-CV.23 | Promote dev→testing calls crManager.CreateOrUpdate with lifecycleStage="testing" | Unit | CreateOrUpdate called with correct args |
| T-CV.24 | Promote testing→production calls crManager.CreateOrUpdate with lifecycleStage="production" | Unit | CreateOrUpdate called |
| T-CV.25 | Demote testing→development calls crManager.Delete | Unit | Delete called |
| T-CV.26 | Demote production→testing calls crManager.CreateOrUpdate (not Delete) | Unit | CreateOrUpdate called with stage="testing" |
| T-CV.27 | Promote with crManager=nil does not panic, still updates DB | Unit | DB updated, no panic |
| T-CV.28 | ListCatalogVersions with allowedStages=["production"] returns only production versions | Unit | Only production versions returned |
| T-CV.29 | GetCatalogVersion returns Forbidden when version stage not in allowedStages | Unit | Forbidden error |

### Config (`config_test.go`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-CV.30 | CLUSTER_ROLE env var defaults to "development" | Unit | ClusterRole="development" |
| T-CV.31 | AllowedStages returns correct stages for each clusterRole value | Unit | Correct arrays |

### CatalogVersion Discovery CRD Coverage Targets

| Target | Scope |
|--------|-------|
| 100% | Pure reconciler functions (CLUSTER_ROLE, CatalogVersionStatus) |
| 100% | CatalogVersion types (DeepCopy) |
| 100% | SanitizeK8sName |
| 100% | K8sCRManager |
| ≥90% | Controller (excluding SetupWithManager) |
| ≥90% | CatalogVersionService promotion/demotion with CR operations |
| — | Documented exceptions per uncovered line |

---

## Meta Model CRUD Gaps + Catalog Version Workflow

These tests cover the remaining CRUD gaps identified in the meta model gap analysis: editing attributes, renaming entity types (context-sensitive), wiring the copy-attributes handler, catalog version detail endpoints (pins + transitions), stage filtering, and catalog version creation with entity selection.

### EditAttribute Service (`attribute_service_test.go`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-E.01 | EditAttribute changes name | Unit | New version created, attribute name updated, other attrs preserved |
| T-E.02 | EditAttribute changes description | Unit | Description updated in new version |
| T-E.03 | EditAttribute changes type to enum with valid enumID | Unit | Type and enumID updated |
| T-E.04 | EditAttribute with name conflict | Unit | Conflict error returned |
| T-E.05 | EditAttribute on nonexistent attribute | Unit | NotFound error |
| T-E.06 | EditAttribute with enum type but missing enumID | Unit | Validation error |
| T-E.07 | EditAttribute preserves ordinal | Unit | Ordinal unchanged in new version |

### RenameEntityType Service (`entity_type_service_test.go`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-E.08 | Rename entity not in any CV | Unit | Simple rename, EntityType.Name updated |
| T-E.09 | Rename entity in one development CV | Unit | Simple rename |
| T-E.10 | Rename entity in testing CV, deepCopyAllowed=false | Unit | DeepCopyRequired error returned |
| T-E.11 | Rename entity in multiple CVs, deepCopyAllowed=true | Unit | New entity type created with new name, old unchanged |
| T-E.12a | Rename with duplicate name | Unit | Conflict error |
| T-E.12b | Rename with empty name | Unit | Validation error |

### CopyAttributes Handler (`attribute_handler_test.go`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-E.13 | POST /entity-types/:id/attributes/copy with valid request | API | 200, new version returned |
| T-E.14 | POST /entity-types/:id/attributes/copy as RO | API | 403 |
| T-E.15 | POST /entity-types/:id/attributes/copy with name conflict | API | 409 |

### EditAttribute Handler (`attribute_handler_test.go`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-E.16 | PUT /entity-types/:id/attributes/:name with valid edit | API | 200, new version |
| T-E.17 | PUT /entity-types/:id/attributes/:name as RO | API | 403 |
| T-E.18 | PUT /entity-types/:id/attributes/:name nonexistent | API | 404 |

### RenameEntityType Handler (`entity_type_handler_test.go`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-E.19 | POST /entity-types/:id/rename simple rename | API | 200, updated entity type |
| T-E.20 | POST /entity-types/:id/rename deep copy required, not allowed | API | 409 with deep_copy_required error |
| T-E.21 | POST /entity-types/:id/rename deep copy allowed | API | 200, new entity type, was_deep_copy=true |

### Catalog Version Pins and Transitions (`catalog_version_service_test.go`, `catalog_version_handler_test.go`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-E.22 | ListPins returns resolved entity type names and versions | Unit | Pins with entity type name, ID, version |
| T-E.23 | ListPins for CV with no pins | Unit | Empty list |
| T-E.24 | ListTransitions returns ordered history | Unit | Transitions ordered by performed_at ASC |
| T-E.25 | GET /catalog-versions/:id/pins | API | 200, resolved pins |
| T-E.26 | GET /catalog-versions/:id/pins for nonexistent CV | API | 404 |
| T-E.27 | GET /catalog-versions/:id/transitions | API | 200, ordered transitions |

### Catalog Version Stage Filter (`catalog_version_handler_test.go`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-E.28 | GET /catalog-versions?stage=testing returns filtered | API | Only testing+production CVs returned |
| T-E.29 | GET /catalog-versions without stage returns all | API | All CVs returned |

### Catalog Version Create with Pins (`catalog_version_service_test.go`, `catalog_version_handler_test.go`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-E.30 | CreateCatalogVersion with pins creates pin records | Unit | CV created, pins linked |
| T-E.31 | CreateCatalogVersion with empty pins | Unit | CV created, no pins |
| T-E.32 | POST /catalog-versions with pins array | API | 201, CV with pins created |
| T-E.33 | POST /catalog-versions as RO | API | 403 |

### Rename Navigation (`App.system.test.ts`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-E.34 | Rename entity type and navigate back shows new name in list | System | After renaming via UI and clicking Back, entity types list shows the new name and not the old name |

### Targeted Delete Safety (`App.system.test.ts`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-E.35 | Delete entity type targets correct row, not first | System | Create two entity types, delete the second (not first in list), verify only the targeted one is deleted and the other survives |
| T-E.36 | Delete enum targets correct row, not first | System | Create two enums, delete the second (not first in list), verify only the targeted one is deleted and the other survives |
| T-E.37 | Delete catalog version targets correct row, not first | System | Create two CVs with different timestamps, delete the older one (second in list), verify only the targeted one is deleted and the other survives |

### Copy Attributes Enum Display (`EntityTypeDetailPage.browser.test.tsx`, `App.system.test.ts`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-E.38 | Attributes table shows enum ID for enum-type attributes | Browser | Enum attributes display truncated enum_id in parentheses next to the type label |
| T-E.39 | Copy attributes picker shows enum name for enum-type attributes | System | When selecting a source entity type with enum attributes, the Type column shows "enum (EnumName)" instead of just "enum" |
| T-E.40 | Copy attributes from multi-version entity type works correctly | System | Create a source entity type, add an attribute (creating V2), then copy that attribute to a target — the copy uses the source's latest version, not V1 |

### Bidirectional Association Listing (`coverage_test.go`, `meta_repo_test.go`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-E.41 | ListAllAssociations returns incoming associations | Unit | Entity type targeted by containment from another type shows "incoming" association with source entity type ID |
| T-E.42 | ListAllAssociations returns both outgoing and incoming | Unit | Entity type with outgoing directional and incoming containment shows both with correct directions |
| T-E.43 | ListAllAssociations skips old version associations | Unit | Incoming association from an old version of the source entity type is filtered out |
| T-E.44 | ListAllAssociations error on ListByTargetEntityType | Unit | Error from repo propagates correctly |
| T-E.45 | ListAllAssociations error on GetByID | Unit | Error resolving source version propagates correctly |
| T-E.46 | ListAllAssociations error on GetLatestByEntityType for source | Unit | Error resolving source latest version propagates correctly |
| T-E.47 | ListByTargetEntityType GORM integration | Integration | Query by target entity type ID returns all associations targeting it |

### Role-Aware Lifecycle Buttons (`App.browser.test.tsx`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-E.48 | Admin cannot see Demote on production catalog version | Browser | Production CV row has no Demote button for Admin role; testing CV row does |

### ContainmentTree Service (`entity_type_service_test.go`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-E.49 | GetContainmentTree with no entity types | Unit | Empty tree |
| T-E.50 | GetContainmentTree flat entities (no containment) | Unit | All entity types as roots, no children |
| T-E.51 | GetContainmentTree single parent | Unit | A contains B → A is root with B as child |
| T-E.52 | GetContainmentTree multi-level | Unit | A contains B, B contains C → nested tree |
| T-E.53 | GetContainmentTree includes all versions per node | Unit | Each node has full version list with latest_version set |

### ContainmentTree Handler (`entity_type_handler_test.go`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-E.54 | GET /entity-types/containment-tree returns 200 | API | Tree response with entity types and versions |

### CV Create Tree UI (`App.browser.test.tsx`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-E.55 | CV create modal shows containment tree | Browser | Entity types rendered as tree with indentation |
| T-E.56 | Selecting parent auto-selects all descendants recursively | Browser | Check parent → children and grandchildren become checked |
| T-E.57 | Deselecting parent deselects all descendants recursively | Browser | Uncheck parent → children and grandchildren become unchecked |
| T-E.58 | Version dropdown shows all versions, defaults to latest | Browser | Dropdown lists versions, latest pre-selected |

### Version Snapshot Service (`entity_type_service_test.go`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-E.59 | GetVersionSnapshot returns attributes and associations for specific version | Unit | Returns snapshot with attributes and associations for the requested version |
| T-E.60 | GetVersionSnapshot returns error for nonexistent entity type | Unit | NotFound error |
| T-E.61 | GetVersionSnapshot returns error for nonexistent version | Unit | NotFound error |

### Version Snapshot Handler (`entity_type_handler_test.go`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-E.62 | GET /entity-types/:id/versions/:version/snapshot returns 200 | API | Snapshot with attributes and associations |
| T-E.63 | GET /entity-types/:id/versions/999/snapshot returns 404 | API | 404 for nonexistent version |

### Read-Only BOM Modal (`CatalogVersionDetailPage.browser.test.tsx`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-E.64 | Clicking entity type in BOM opens read-only modal | Browser | Modal opens with entity type name and pinned version |
| T-E.65 | BOM modal shows attributes table with resolved enum names | Browser | Attributes listed with name, type (enum attributes show "EnumName (enum)"), description |
| T-E.66 | BOM modal shows associations with contextual labels | Browser | Associations show relationship label (contains/contained by/references/referenced by/references (mutual)), other entity type name, perspective-correct role |
| T-E.67 | BOM modal has no edit controls | Browser | No Add, Remove, Edit, or Reorder buttons in modal |

### Association Cardinality — Validation (`cardinality_test.go`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-E.68 | ValidateCardinality accepts standard options (0..1, 0..n, 1, 1..n) | Unit | No error |
| T-E.69 | ValidateCardinality accepts custom ranges (2..5, 2..n) | Unit | No error |
| T-E.70 | ValidateCardinality accepts exact values (3) | Unit | No error |
| T-E.71 | ValidateCardinality accepts empty string | Unit | No error |
| T-E.72 | ValidateCardinality rejects invalid formats (negative, min>max, non-numeric, malformed) | Unit | Error returned for each invalid input |
| T-E.73 | NormalizeCardinality returns "0..n" for empty string | Unit | "0..n" |
| T-E.74 | NormalizeCardinality passes through valid values unchanged | Unit | Same value returned |

### Association Cardinality — Service (`association_service_test.go`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-E.75 | CreateAssociation with valid cardinality passes through to model | Unit | Association created with specified cardinality values |
| T-E.76 | CreateAssociation with empty cardinality normalizes to "0..n" | Unit | Association created with "0..n" on both ends |
| T-E.77 | CreateAssociation with invalid cardinality returns error | Unit | Validation error, no version created |

### Association Cardinality — Repository (`meta_repo_test.go`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-E.78 | Create association with cardinality stores and retrieves correctly | Integration | Fields persisted and returned on query |
| T-E.79 | BulkCopyToVersion preserves cardinality on copied associations | Integration | Copied association has same cardinality values as original |

### Association Cardinality — API (`association_handler_test.go`, `entity_type_handler_test.go`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-E.80 | POST /entity-types/:id/associations with cardinality | API | 201, version created, cardinality passed through |
| T-E.81 | POST /entity-types/:id/associations with invalid cardinality | API | 400 |
| T-E.82 | GET /entity-types/:id/associations returns normalized cardinality | API | Response includes source_cardinality and target_cardinality, empty → "0..n" |
| T-E.83 | GET /entity-types/:id/versions/:v/snapshot includes cardinality | API | Snapshot associations include source_cardinality and target_cardinality |

### Association Cardinality — UI (`EntityTypeDetailPage.browser.test.tsx`, `CatalogVersionDetailPage.browser.test.tsx`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-E.84 | Add association modal has cardinality dropdowns defaulting to "0..n" | Browser | Two dropdowns with standard options, default "0..n" |
| T-E.85 | Add association modal custom cardinality reveals min/max inputs | Browser | Selecting "Custom" shows min and max fields |
| T-E.86 | Associations table shows cardinality column | Browser | Column displays source and target cardinality values |
| T-E.87 | BOM modal associations table includes cardinality | Browser | Read-only cardinality values shown for each association |
| T-E.88 | Custom cardinality with empty fields shows client-side error | Browser | Validation error shown before API call when custom selected with empty min/max |
| T-E.89 | Custom cardinality min field only accepts digits | Browser | Non-digit characters rejected in min input field |

### Association Cardinality — System (`App.system.test.ts`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-E.90 | Association cardinality is stored and displayed end-to-end | System | Create association with cardinality via API, verify in list response and UI display |

### Association Cardinality — Containment Constraint (`association_service_test.go`, `EntityTypeDetailPage.browser.test.tsx`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-E.91 | Containment rejects invalid source cardinality (0..n, 1..n) | Unit | Validation error, no version created |
| T-E.92 | Containment accepts valid source cardinality (1, 0..1, empty) | Unit | Association created with valid cardinality |
| T-E.93 | Non-containment allows any source cardinality | Unit | No restriction on directional/bidirectional |
| T-E.94 | Containment source cardinality dropdown restricted to 1 and 0..1 | Browser | Only two options shown, default 0..1 |

### Edit Association — Service (`association_service_test.go`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-E.95 | EditAssociation changes source role | Unit | New version created, source role updated, other fields preserved |
| T-E.96 | EditAssociation changes cardinality | Unit | New version created, cardinality updated |
| T-E.97 | EditAssociation with invalid cardinality | Unit | Validation error, no version created |
| T-E.98 | EditAssociation containment rejects invalid source cardinality | Unit | Validation error when editing containment source to "0..n" |
| T-E.99 | EditAssociation on nonexistent association | Unit | NotFound error |

### Edit Association — API (`association_handler_test.go`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-E.100 | PUT /entity-types/:id/associations/:name with valid edit | API | 200, new version returned |
| T-E.101 | PUT /entity-types/:id/associations/:name as RO | API | 403 |
| T-E.102 | PUT /entity-types/:id/associations/:name nonexistent | API | 404 |
| T-E.103 | PUT /entity-types/:id/associations/:name invalid cardinality | API | 400 |

### Edit Association — UI (`EntityTypeDetailPage.browser.test.tsx`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-E.104 | Edit button opens modal with pre-filled association values | Browser | Modal shows current role and cardinality values |
| T-E.105 | Edit association Save triggers API call | Browser | API called with updated fields, table refreshes |

### Shared EditAssociationModal + Custom Cardinality (`EntityTypeDetailPage.browser.test.tsx`, `App.browser.test.tsx`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-E.137 | Edit modal shows custom cardinality option | Browser | Selecting "Custom" in edit modal reveals min/max inputs |
| T-E.138 | Edit modal custom cardinality sends correct value | Browser | Custom min/max values sent as "min..max" in API call |
| T-E.139 | Edit modal pre-fills custom cardinality correctly | Browser | Association with non-standard cardinality (e.g., "2..5") shows "Custom" selected with min=2, max=5 |

### Association Names — Service (`association_service_test.go`, `attribute_service_test.go`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-E.106 | CreateAssociation requires name | Unit | Validation error when name is empty |
| T-E.107 | CreateAssociation rejects duplicate association name | Unit | Conflict error when name already exists in version |
| T-E.108 | CreateAssociation rejects name that conflicts with attribute | Unit | Conflict error when association name matches existing attribute name |
| T-E.109 | AddAttribute rejects name that conflicts with association | Unit | Conflict error when attribute name matches existing association name |
| T-E.110 | EditAssociation can rename | Unit | New version created, name updated |
| T-E.111 | EditAssociation rename conflicts with attribute | Unit | Conflict error |
| T-E.112 | DeleteAssociation by name | Unit | New version without the named association |
| T-E.113 | COW matching uses name instead of all-properties | Unit | Correct association matched even with reordered list |

### Association Names — Repository (`meta_repo_test.go`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-E.114 | Association name stored and retrieved | Integration | Name persisted and returned |
| T-E.115 | Association unique constraint on (version_id, name) | Integration | Duplicate name in same version rejected |
| T-E.116 | BulkCopyToVersion preserves association names | Integration | Copied associations retain names |

### Association Names — API (`association_handler_test.go`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-E.117 | POST /associations with name returns 201 | API | Name included in create, version returned |
| T-E.118 | POST /associations without name returns 400 | API | Name required |
| T-E.119 | POST /associations duplicate name returns 409 | API | Conflict |
| T-E.120 | GET /associations returns name in response | API | Name field present in list items |
| T-E.121 | PUT /associations/:name with valid edit returns 200 | API | Route uses name param |
| T-E.122 | DELETE /associations/:name returns 204 | API | Route uses name param |
| T-E.123 | GET /versions/:v/snapshot includes association name | API | Name in snapshot response |

### Association Names — UI (`EntityTypeDetailPage.browser.test.tsx`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-E.124 | Add association modal has name field | Browser | Name input visible, required |
| T-E.125 | Associations table shows Name column | Browser | Name displayed in table |
| T-E.126 | Edit association modal shows name pre-filled | Browser | Current name editable |

### Entity Type Diagram — Main Page (`App.browser.test.tsx`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-E.127 | Model Diagram tab exists on main page | Browser | Tab labeled "Model Diagram" is visible |
| T-E.128 | Diagram renders entity type nodes with names | Browser | Node labels show entity type names |
| T-E.129 | Diagram nodes show attributes with types | Browser | Attributes listed as "name : type" (enum shows enum name) |
| T-E.130 | Diagram renders edges with association labels | Browser | Edges show association name and cardinality |
| T-E.131 | Bidirectional edges have two arrowheads (filled target, hollow source) | Browser | Bidirectional edge renders with label and dual arrowhead markers |
| T-E.134 | Double-click entity type node navigates to detail page | Browser | Router navigates to /entity-types/:id |
| T-E.135 | Click association label on diagram opens edit modal | Browser | Edit modal opens with name, type, roles, cardinality; source/target entities read-only |
| T-E.136 | EditAssociation can change association type | Unit | New version created with changed type (e.g., directional → bidirectional) |

### Entity Type Diagram — CV Detail Page (`CatalogVersionDetailPage.browser.test.tsx`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-E.132 | Diagram tab exists on CV detail page | Browser | Tab labeled "Diagram" is visible |
| T-E.133 | CV diagram renders pinned entity types with attributes | Browser | Only pinned entity types shown as nodes with attributes |

### Attribute Required Field (`attribute_service_test.go`, `attribute_handler_test.go`, `EntityTypeDetailPage.browser.test.tsx`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-E.140 | AddAttribute with required=true stores required flag | Unit | Attribute created with required=true |
| T-E.141 | EditAttribute can change required flag | Unit | New version created, required flag updated |
| T-E.142 | POST /attributes with required field | API | 201, attribute created with required=true |
| T-E.143 | PUT /attributes/:name with required field | API | 200, required flag updated |
| T-E.144 | Add attribute modal has required checkbox | Browser | Checkbox visible, default unchecked |
| T-E.145 | Edit attribute modal pre-fills required checkbox | Browser | Checkbox reflects current required value |
| T-E.146 | Attributes table shows required indicator | Browser | Required attributes marked with indicator |

---

## Milestone 10: Catalog Foundation

Catalogs are named data containers pinned to a catalog version. The operational API uses catalog names (DNS-label format) in URLs. This milestone covers the Catalog entity CRUD, domain model refactoring (`EntityInstance.CatalogVersionID` → `CatalogID`), and removal of the old CV-scoped operational scaffolding.

### Catalog Domain Model and Repository

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-10.01 | Create catalog with valid name and CV ID | Integration | Catalog persisted with validation_status=draft |
| T-10.02 | Create catalog with duplicate name | Integration | Unique constraint error |
| T-10.03 | Create catalog with nonexistent CV ID | Integration | FK constraint error |
| T-10.04 | GetByName retrieves catalog by name | Integration | Correct catalog returned |
| T-10.05 | GetByName for nonexistent name | Integration | NotFound error |
| T-10.06 | List catalogs returns all | Integration | All catalogs returned with total count |
| T-10.07 | List catalogs filtered by catalog_version_id | Integration | Only catalogs with matching CV returned |
| T-10.08 | List catalogs filtered by validation_status | Integration | Only catalogs with matching status returned |
| T-10.09 | Delete catalog by ID | Integration | Catalog removed from DB |
| T-10.10 | EntityInstance uses catalog_id FK (not catalog_version_id) | Integration | Instance created with catalog_id, FK enforced |

### Catalog Service

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-10.11 | CreateCatalog with valid DNS-label name | Unit | Catalog created with draft status, UUID assigned |
| T-10.12 | CreateCatalog with invalid name (uppercase) | Unit | Validation error |
| T-10.13 | CreateCatalog with invalid name (special chars) | Unit | Validation error |
| T-10.14 | CreateCatalog with invalid name (empty) | Unit | Validation error |
| T-10.15 | CreateCatalog with invalid name (>63 chars) | Unit | Validation error |
| T-10.16 | CreateCatalog with invalid name (starts with hyphen) | Unit | Validation error |
| T-10.17 | CreateCatalog with invalid name (ends with hyphen) | Unit | Validation error |
| T-10.18 | CreateCatalog with duplicate name | Unit | Conflict error |
| T-10.19 | CreateCatalog with nonexistent CV ID | Unit | NotFound error |
| T-10.20 | GetByName returns catalog with resolved CV label | Unit | Response includes CV version_label |
| T-10.21 | GetByName for nonexistent name | Unit | NotFound error |
| T-10.22 | List catalogs with no filters | Unit | All catalogs returned |
| T-10.23 | List catalogs filtered by catalog_version_id | Unit | Filtered results |
| T-10.24 | List catalogs filtered by validation_status | Unit | Filtered results |
| T-10.25 | Delete catalog cascades to entity instances | Unit | All instances in catalog deleted |
| T-10.26 | Delete nonexistent catalog | Unit | NotFound error |

### Catalog API Handler

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-10.27 | POST /api/data/v1/catalogs with valid request | API | 201, catalog with id, name, status=draft |
| T-10.28 | POST /api/data/v1/catalogs with invalid name format | API | 400, clear error message |
| T-10.29 | POST /api/data/v1/catalogs with duplicate name | API | 409 |
| T-10.30 | POST /api/data/v1/catalogs with nonexistent CV | API | 404 |
| T-10.31 | POST /api/data/v1/catalogs as RO | API | 403 |
| T-10.32 | POST /api/data/v1/catalogs as RW | API | 201 (RW can create) |
| T-10.33 | GET /api/data/v1/catalogs returns list | API | 200, array with total count |
| T-10.34 | GET /api/data/v1/catalogs?catalog_version_id=X | API | 200, filtered list |
| T-10.35 | GET /api/data/v1/catalogs?validation_status=draft | API | 200, filtered list |
| T-10.36 | GET /api/data/v1/catalogs/{name} returns detail | API | 200, catalog with resolved CV label |
| T-10.37 | GET /api/data/v1/catalogs/{name} nonexistent | API | 404 |
| T-10.38 | DELETE /api/data/v1/catalogs/{name} | API | 204, catalog and instances removed |
| T-10.39 | DELETE /api/data/v1/catalogs/{name} as RO | API | 403 |
| T-10.40 | DELETE /api/data/v1/catalogs/{name} nonexistent | API | 404 |

### Catalog UI (Meta UI)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-10.41 | Catalogs nav item visible in sidebar | Browser | Nav item "Catalogs" present |
| T-10.42 | Catalog list page shows name, CV label, status badge, date | Browser | All columns rendered with correct data |
| T-10.43 | Catalog list status badge color-coded (draft=blue, valid=green, invalid=red) | Browser | Correct badge variant per status |
| T-10.44 | Create catalog button visible for RW+, hidden for RO | Browser | Role-aware visibility |
| T-10.45 | Create catalog modal: name input, description, CV dropdown | Browser | All form fields present |
| T-10.46 | Create catalog modal: invalid name shows inline error | Browser | Error shown for uppercase, special chars, etc. |
| T-10.47 | Create catalog modal: submit calls API, list refreshes | Browser | API called, new catalog appears in list |
| T-10.48 | Delete catalog button visible for RW+, hidden for RO | Browser | Role-aware visibility |
| T-10.49 | Delete catalog shows confirmation dialog | Browser | Confirmation required before delete |
| T-10.50 | Delete catalog: confirm removes from list | Browser | Catalog disappears from list after delete |

### Old Operational Scaffolding Removal

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-10.51 | Old CV-scoped routes (/api/catalog/{cv}/...) no longer registered | API | 404 for old route pattern |

---

## Milestone 11: Instance CRUD with Attributes

Entity instances are created within a catalog, scoped to an entity type pinned in the catalog's CV. Attribute values are set on create, validated by type, returned with resolved names, and versioned on update. The old CV-scoped instance scaffolding is replaced with catalog-scoped routes.

### Instance Repository — Attribute Values

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-11.01 | SetValues stores attribute values for instance | Integration | Values persisted and retrievable |
| T-11.02 | GetCurrentValues returns latest version's values | Integration | Returns values for highest version |
| T-11.03 | GetValuesForVersion returns values for specific version | Integration | Returns values for requested version only |
| T-11.04 | SetValues for new version preserves previous version's values | Integration | Both version 1 and version 2 values retrievable independently |
| T-11.05 | Instance creation with catalog_id and attribute values end-to-end | Integration | Instance + values persisted, FK constraints satisfied |

### Pin Resolution Chain

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-11.06 | Catalog → CV → Pin → EntityTypeVersion resolution | Integration | Given catalog with CV that pins entity type, full chain resolves to correct EntityTypeVersion |
| T-11.07 | Pin resolution returns attributes for the pinned version | Integration | Attributes from pinned version (not latest) returned |
| T-11.08 | Pin resolution for entity type not pinned in CV | Integration | No pin found for that entity type |

### Optimistic Locking (DB level)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-11.09 | Update instance with matching version succeeds | Integration | Version incremented, update applied |
| T-11.10 | Update instance with stale version returns 0 rows affected | Integration | RowsAffected=0, no data changed |

### Instance Service — Create with Pin Resolution

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-11.11 | Create instance in catalog with pinned entity type | Unit | Instance created with correct CatalogID and EntityTypeID |
| T-11.12 | Create instance with entity type not pinned in CV | Unit | NotFound error |
| T-11.13 | Create instance in nonexistent catalog | Unit | NotFound error |
| T-11.14 | Create instance with attribute values (string) | Unit | Instance created, string attribute value stored |
| T-11.15 | Create instance with attribute values (number) | Unit | Instance created, number attribute value stored |
| T-11.16 | Create instance with attribute values (enum — valid value) | Unit | Instance created, enum value stored |
| T-11.17 | Create instance with attribute values (enum — invalid value) | Unit | Validation error |
| T-11.18 | Create instance with attribute values (number — non-parseable) | Unit | Validation error |
| T-11.19 | Create instance with missing optional attributes | Unit | Instance created, no error |
| T-11.20 | Create instance with missing required attributes (draft mode) | Unit | Instance created, no error (validation is Phase 5) |
| T-11.21 | Create instance with duplicate name in same catalog | Unit | Conflict error |
| T-11.22 | Create instance with unknown attribute name | Unit | Validation error — attribute not in schema |

### Instance Service — Update

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-11.23 | Update instance attribute values | Unit | Version incremented, new values stored, previous retained |
| T-11.24 | Update instance with version mismatch | Unit | Conflict error |
| T-11.25 | Update instance name and description | Unit | Fields updated, version incremented |
| T-11.26 | Update with invalid attribute value type | Unit | Validation error, no version change |
| T-11.27 | Update nonexistent instance | Unit | NotFound error |

### Instance Service — Get and List

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-11.28 | Get instance returns resolved attribute values (name, type, value) | Unit | Response includes attribute name/type from schema + value |
| T-11.29 | List instances returns instances with attribute values | Unit | Each instance includes its current attribute values |
| T-11.30 | List instances in catalog with no instances | Unit | Empty list, total=0 |

### Instance Service — Delete and Validation Status

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-11.31 | Delete instance cascades to children | Unit | Parent and children soft-deleted |
| T-11.32 | Create instance resets catalog validation status to draft | Unit | Catalog status updated to draft |
| T-11.33 | Update instance resets catalog validation status to draft | Unit | Catalog status updated to draft |
| T-11.34 | Delete instance resets catalog validation status to draft | Unit | Catalog status updated to draft |

### Instance API Handler

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-11.35 | POST /catalogs/{name}/{entity-type} with attributes → 201 | API | Instance created with attribute values in response |
| T-11.36 | POST /catalogs/{name}/{entity-type} nonexistent catalog → 404 | API | Clear error message |
| T-11.37 | POST /catalogs/{name}/{entity-type} entity type not pinned → 404 | API | Clear error message |
| T-11.38 | POST /catalogs/{name}/{entity-type} invalid attribute value → 400 | API | Validation error |
| T-11.39 | POST /catalogs/{name}/{entity-type} as RO → 403 | API | Forbidden |
| T-11.40 | POST /catalogs/{name}/{entity-type} as RW → 201 | API | RW can create instances |
| T-11.41 | GET /catalogs/{name}/{entity-type} → 200 list with attributes | API | Instances with resolved attribute values |
| T-11.42 | GET /catalogs/{name}/{entity-type}/{id} → 200 with attributes | API | Single instance with resolved attribute values |
| T-11.43 | GET /catalogs/{name}/{entity-type}/{id} nonexistent → 404 | API | Not found |
| T-11.44 | PUT /catalogs/{name}/{entity-type}/{id} → 200, version incremented | API | Updated attributes, new version in response |
| T-11.45 | PUT /catalogs/{name}/{entity-type}/{id} version mismatch → 409 | API | Conflict error |
| T-11.46 | DELETE /catalogs/{name}/{entity-type}/{id} → 204 | API | Instance deleted |
| T-11.47 | DELETE /catalogs/{name}/{entity-type}/{id} as RO → 403 | API | Forbidden |

### Instance UI — Catalog Detail Page

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-11.48 | Catalog detail page shows tabs per pinned entity type | Browser | One tab per entity type in CV pins |
| T-11.49 | Entity type tab shows instance list table | Browser | Table with Name, Description columns + dynamic attribute columns |
| T-11.50 | Instance list shows attribute values in columns | Browser | Attribute values rendered in correct columns |
| T-11.51 | Create instance button visible for RW+, hidden for RO | Browser | Role-aware visibility |
| T-11.52 | Create instance modal has dynamic attribute form | Browser | String → text input, number → number input, enum → dropdown |
| T-11.53 | Create instance modal submits with attribute values | Browser | API called with name, description, attributes; list refreshes |
| T-11.54 | Edit instance opens modal with pre-filled values | Browser | Current name, description, attribute values shown |
| T-11.55 | Edit instance submits updated values | Browser | API called with version + changed fields; list refreshes |
| T-11.56 | Delete instance shows confirmation dialog | Browser | Confirmation with instance name |
| T-11.57 | Delete instance removes from list | Browser | Instance disappears after confirm |
| T-11.58 | Empty instance list shows empty state | Browser | "No instances" message |

---

## Milestone 12: Containment & Association Links

Contained instances are created under a parent instance via sub-resource URLs. Association links connect instances based on directional or bidirectional association definitions in the pinned CV. All relationships are validated against the CV's association definitions. Single-level containment routes are supported; multi-level containment URLs are deferred to Phase 4.

### Contained Instance Repository

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-12.01 | Create contained instance with parent_instance_id set | Integration | Instance persisted with correct parent_instance_id |
| T-12.02 | Same name under different parents allowed | Integration | Both instances created, unique constraint satisfied |
| T-12.03 | Same name under same parent rejected | Integration | Unique constraint violation |
| T-12.04 | ListByParent returns only direct children of specified parent | Integration | Only children with matching parent_instance_id returned |

### Contained Instance Service

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-12.05 | CreateContainedInstance with valid parent and containment association in CV | Unit | Instance created with parent_instance_id set, correct entity type |
| T-12.06 | CreateContainedInstance with nonexistent parent | Unit | NotFound error |
| T-12.07 | CreateContainedInstance with child type not in containment relationship with parent type | Unit | Validation error — no containment association exists |
| T-12.08 | CreateContainedInstance with child type not pinned in CV | Unit | NotFound error |
| T-12.09 | CreateContainedInstance same name under different parents | Unit | Both created successfully |
| T-12.10 | CreateContainedInstance duplicate name under same parent | Unit | Conflict error |
| T-12.11 | ListContainedInstances returns only direct children of specified type | Unit | Filtered by parent ID and entity type |
| T-12.12 | CreateContainedInstance resets catalog validation status to draft | Unit | Catalog status updated to draft |
| T-12.13 | CreateContainedInstance with attribute values | Unit | Instance created with validated attribute values |

### Association Link Repository

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-12.14 | Create association link with valid source and target IDs | Integration | Link persisted with correct association_id, source, target |
| T-12.15 | GetForwardRefs returns target instances for source | Integration | All links where source matches returned |
| T-12.16 | GetReverseRefs returns source instances for target | Integration | All links where target matches returned |
| T-12.17 | Delete association link removes it | Integration | Link no longer returned by forward/reverse queries |
| T-12.18 | GetForwardRefs for instance with no links | Integration | Empty list returned |

### Association Link Service

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-12.19 | CreateAssociationLink with valid association definition in CV | Unit | Link created with correct association_id |
| T-12.20 | CreateAssociationLink with nonexistent association name | Unit | NotFound error |
| T-12.21 | CreateAssociationLink source entity type does not match association's source type | Unit | Validation error |
| T-12.22 | CreateAssociationLink target entity type does not match association's target type | Unit | Validation error |
| T-12.23 | CreateAssociationLink with nonexistent target instance | Unit | NotFound error |
| T-12.24 | CreateAssociationLink with nonexistent source instance | Unit | NotFound error |
| T-12.25 | DeleteAssociationLink removes link | Unit | Link deleted successfully |
| T-12.26 | DeleteAssociationLink nonexistent link | Unit | NotFound error |
| T-12.27 | CreateAssociationLink resets catalog validation status to draft | Unit | Catalog status updated to draft |
| T-12.28 | DeleteAssociationLink resets catalog validation status to draft | Unit | Catalog status updated to draft |

### Forward/Reverse Reference Service

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-12.29 | GetForwardReferences returns resolved target info | Unit | Response includes link ID, association name, association type, target instance ID/name/entity type name |
| T-12.30 | GetForwardReferences includes directional associations | Unit | Directional links included in results |
| T-12.31 | GetForwardReferences includes bidirectional associations | Unit | Bidirectional links included in results |
| T-12.32 | GetForwardReferences for instance with no links | Unit | Empty list, no error |
| T-12.33 | GetReverseReferences returns resolved source info | Unit | Response includes link ID, association name, association type, source instance ID/name/entity type name |
| T-12.34 | GetReverseReferences includes directional associations | Unit | Directional links (where this instance is target) included |
| T-12.35 | GetReverseReferences includes bidirectional associations | Unit | Bidirectional links (where this instance is target) included |
| T-12.36 | GetReverseReferences for instance with no incoming links | Unit | Empty list, no error |

### Contained Instance API Handler

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-12.37 | POST /{catalog}/{parent-type}/{parent-id}/{child-type} with valid request | API | 201, contained instance with parent_instance_id in response |
| T-12.38 | POST /{catalog}/{parent-type}/{parent-id}/{child-type} nonexistent parent | API | 404 |
| T-12.39 | POST /{catalog}/{parent-type}/{parent-id}/{child-type} no containment association | API | 400, clear error message |
| T-12.40 | POST /{catalog}/{parent-type}/{parent-id}/{child-type} as RO | API | 403 |
| T-12.41 | GET /{catalog}/{parent-type}/{parent-id}/{child-type} lists children | API | 200, array of contained instances |
| T-12.42 | GET /{catalog}/{parent-type}/{parent-id}/{child-type} nonexistent parent | API | 404 |

### Association Link API Handler

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-12.43 | POST /{catalog}/{type}/{id}/links with valid request | API | 201, link with resolved association info |
| T-12.44 | POST /{catalog}/{type}/{id}/links nonexistent source instance | API | 404 |
| T-12.45 | POST /{catalog}/{type}/{id}/links invalid association name | API | 400 |
| T-12.46 | POST /{catalog}/{type}/{id}/links mismatched entity types | API | 400, validation error |
| T-12.47 | POST /{catalog}/{type}/{id}/links as RO | API | 403 |
| T-12.48 | DELETE /{catalog}/{type}/{id}/links/{link-id} | API | 204 |
| T-12.49 | DELETE /{catalog}/{type}/{id}/links/{link-id} as RO | API | 403 |
| T-12.50 | DELETE /{catalog}/{type}/{id}/links/{link-id} nonexistent | API | 404 |
| T-12.51 | GET /{catalog}/{type}/{id}/references returns forward refs | API | 200, array with resolved target info |
| T-12.52 | GET /{catalog}/{type}/{id}/referenced-by returns reverse refs | API | 200, array with resolved source info |
| T-12.53 | GET /{catalog}/{type}/{id}/references nonexistent instance | API | 404 |

### Containment & Links UI (Meta UI)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-12.54 | Instance detail shows contained children section | Browser | Children listed by entity type under parent |
| T-12.55 | Add contained instance button visible for RW+, hidden for RO | Browser | Role-aware visibility |
| T-12.56 | Add contained instance modal creates child under parent | Browser | API called with parent context, child appears in list |
| T-12.57 | Contained instance appears in parent's children list after creation | Browser | List refreshes with new child |
| T-12.58 | Instance detail shows references tab | Browser | Tab with forward and reverse reference sections |
| T-12.59 | Forward references show target instance info with association name | Browser | Association name, type, target name displayed |
| T-12.60 | Reverse references show source instance info with association name | Browser | Association name, type, source name displayed |
| T-12.61 | Link to instance action creates association link | Browser | Modal for selecting target instance and association, API called |
| T-12.62 | Unlink action removes association link | Browser | Confirmation dialog, link removed from list |
| T-12.63 | RO user sees references but no link/unlink controls | Browser | Read-only view, no action buttons |

## Milestone 13: Catalog Data Viewer

Phase 4 of the catalog implementation plan. Adds a read-only operational UI for browsing catalog data, plus backend enhancements for filtering, sorting, pagination, containment tree, and parent chain resolution. User stories: US-17, US-18, US-19, US-20, US-21, US-40.

### Containment Tree — Repository

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-13.01 | `ListByCatalog` returns all instances in a catalog | Integration | All instances across entity types returned |
| T-13.02 | `ListByCatalog` excludes instances from other catalogs | Integration | Only instances matching catalogID returned |
| T-13.03 | `ListByCatalog` returns empty list for empty catalog | Integration | Empty slice, no error |

### Containment Tree — Service

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-13.04 | `GetContainmentTree` builds tree from flat instance list | Unit | Root instances at top level, children nested |
| T-13.05 | Root instances (no parent) appear as top-level nodes | Unit | Instances with empty ParentInstanceID at root |
| T-13.06 | Children nested under their parent | Unit | Child nodes under correct parent |
| T-13.07 | Multi-level nesting (grandchildren) | Unit | 3+ level hierarchy correctly built |
| T-13.08 | Empty catalog returns empty tree | Unit | Empty slice, no error |
| T-13.09 | Each tree node includes entity type name | Unit | Entity type name resolved via etRepo |
| T-13.10 | Nonexistent catalog returns NotFound | Unit | 404 error |

### Containment Tree — API Handler

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-13.11 | `GET /catalogs/{name}/tree` returns 200 with tree | API | Nested JSON tree structure |
| T-13.12 | `GET /catalogs/{name}/tree` returns 404 for nonexistent catalog | API | 404 response |
| T-13.13 | Tree nodes include instance name, ID, entity type name | API | All fields present in response |
| T-13.14 | Tree structure matches containment relationships | API | Parent-child nesting correct |

### Attribute-Based Filtering — Repository

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-13.15 | String filter applies case-insensitive contains match | Integration | `LIKE '%value%'` behavior |
| T-13.16 | Number filter applies exact match | Integration | Exact float64 comparison |
| T-13.17 | Number range filter with min only | Integration | `>= min` |
| T-13.18 | Number range filter with max only | Integration | `<= max` |
| T-13.19 | Number range filter with min and max | Integration | `>= min AND <= max` |
| T-13.20 | Enum filter applies exact match | Integration | Exact string comparison on enum value |
| T-13.21 | Multiple filters combine with AND logic | Integration | All conditions must match |
| T-13.22 | Filter on attribute with no matching instances returns empty | Integration | Empty result, no error |
| T-13.23 | Filtering works across EAV join | Integration | Correct join on instance_attribute_values |

### Attribute-Based Filtering — Service

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-13.24 | Filter params passed through to repository | Unit | Repository called with correct ListParams |
| T-13.25 | Unknown attribute name in filter returns validation error | Unit | 400-level error |
| T-13.26 | Filter params resolved from attribute name to attribute ID | Unit | Service translates name→ID before passing to repo |

### Attribute-Based Filtering — API Handler

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-13.27 | `GET /{name}/{type}?filter.strattr=hello` returns filtered results | API | Only instances with matching string attr |
| T-13.28 | `GET /{name}/{type}?filter.numattr=5` filters by exact number | API | Only instances with numattr=5 |
| T-13.29 | `GET /{name}/{type}?filter.numattr.min=1&filter.numattr.max=10` range filter | API | Only instances with 1<=numattr<=10 |
| T-13.30 | Multiple filter params combine with AND | API | Intersection of all filters |
| T-13.31 | `filter.unknownattr=x` returns 400 | API | Bad request error |

### Sorting — Repository

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-13.32 | Sort by string attribute ascending | Integration | Alphabetical order |
| T-13.33 | Sort by string attribute descending | Integration | Reverse alphabetical |
| T-13.34 | Sort by number attribute ascending | Integration | Numeric order (1, 2, 10 not 1, 10, 2) |
| T-13.35 | Sort by number attribute descending | Integration | Reverse numeric order |
| T-13.36 | Sort by name (built-in field) ascending | Integration | Name alphabetical |

### Sorting — API Handler

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-13.37 | `?sort=attr:asc` sorts ascending | API | Results in ascending order |
| T-13.38 | `?sort=attr:desc` sorts descending | API | Results in descending order |
| T-13.39 | `?sort=name:asc` sorts by built-in name field | API | Results sorted by name |
| T-13.40 | No sort param uses default order | API | Results in default (created) order |

### Pagination — Repository

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-13.41 | Offset skips correct number of results | Integration | First N results excluded |
| T-13.42 | Limit caps result count | Integration | At most N results returned |
| T-13.43 | Total count unaffected by offset/limit | Integration | Total reflects all matching, not page |
| T-13.44 | Offset beyond total returns empty with correct total | Integration | Empty items, total still accurate |

### Pagination — API Handler

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-13.45 | `?limit=5&offset=10` returns correct page | API | 5 results starting from 11th |
| T-13.46 | Response includes total count | API | `total` field in response |
| T-13.47 | Default limit is 20 when not specified | API | 20 results max |
| T-13.48 | Limit capped at 100 | API | `?limit=500` returns at most 100 |
| T-13.49 | `?offset=0&limit=0` returns count only (no items) | API | Empty items array, total populated |

### Parent Chain Resolution — Service

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-13.50 | Parent chain resolves from instance up to root | Unit | Ordered list of ancestors |
| T-13.51 | Root instance has empty parent chain | Unit | Empty array |
| T-13.52 | Multi-level chain (3+ levels) resolves correctly | Unit | All ancestors in root-first order |
| T-13.53 | Each chain entry includes instance ID, name, entity type name | Unit | All fields populated |

### Parent Chain Resolution — API Handler

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-13.54 | `GET /{name}/{type}/{id}` includes `parent_chain` in response | API | Array field present |
| T-13.55 | Parent chain is ordered root-first | API | First entry is root ancestor |
| T-13.56 | Root instance has empty parent chain | API | Empty array in response |

### Operational UI — Build Infrastructure

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-13.57 | Vite multi-entry build produces `index.html` and `operational.html` | Browser | Both HTML files exist in build output |
| T-13.58 | Operational entry point renders OperationalApp shell | Browser | App mounts and renders |
| T-13.59 | Operational app masthead shows "AI Asset Hub — Data Viewer" | Browser | Brand text correct |
| T-13.60 | Role selector in masthead works | Browser | Role changes propagate to API calls |

### Operational UI — Catalog List Page

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-13.61 | Catalog list page loads and shows catalogs | Browser | Table with catalog rows |
| T-13.62 | Catalog columns: name, CV label, status badge, instance count | Browser | All columns rendered |
| T-13.63 | Search input filters catalogs by name | Browser | Table filters as user types |
| T-13.64 | Sortable column headers | Browser | Click header toggles sort |
| T-13.65 | Pagination controls present | Browser | Page size selector and navigation |
| T-13.66 | Clicking catalog name navigates to catalog detail | Browser | Route changes to /catalogs/{name} |
| T-13.67 | Validation status badge colors (green=valid, blue=draft, red=invalid) | Browser | Correct label colors |

### Operational UI — Catalog Detail Overview

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-13.68 | Catalog detail shows header with name, status badge, CV label | Browser | All header elements rendered |
| T-13.69 | Overview tab lists entity types from pinned CV | Browser | Table with entity type rows |
| T-13.70 | Entity type rows show name, version, instance count | Browser | All columns populated |
| T-13.71 | "Browse Instances" button switches to tree browser for that type | Browser | Tab changes, tree loads |

### Operational UI — Containment Tree Browser (Two-Pane Layout)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-13.72 | Tree browser tab renders two-pane layout with tree and detail | Browser | Tree on left, detail/empty state on right |
| T-13.73 | Tree groups root instances under entity type headers with counts | Browser | "mcp-server (2)" format |
| T-13.74 | Entity type group headers are expandable | Browser | Click expands/collapses to show instances |
| T-13.75 | Clicking a tree instance shows detail in right panel | Browser | Detail panel populates inline (not drawer overlay) |
| T-13.76 | Multi-level tree expands correctly (3+ levels) | Browser | Grandchildren visible after expanding |
| T-13.77 | Empty state shown when no instance selected | Browser | "Select an instance" message in right panel |

### Operational UI — Instance List (REMOVED)

Instance list table with filtering, sorting, and pagination has been removed from the read-only tree browser. The tree is the primary navigation. The backend filtering/sorting/pagination API remains available and is tested at the API layer (T-13.27-49). UI for these controls is deferred to FF-6 (operational editing).

Test cases T-13.78 through T-13.85 are retired.

### Operational UI — Instance Detail

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-13.86 | Instance detail panel shows attributes table | Browser | Name, type, value columns |
| T-13.87 | Enum values show resolved names | Browser | Enum display name, not raw value |
| T-13.88 | Instance description displayed | Browser | Description text visible |
| T-13.89 | Instance version and timestamps displayed | Browser | Version number, created/updated dates |

### Operational UI — Breadcrumb Navigation

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-13.90 | Breadcrumb renders containment path | Browser | Catalog > Parent > Current |
| T-13.91 | Breadcrumb shows entity type and instance name per level | Browser | "MCP Server: my-server" format |
| T-13.92 | Breadcrumb links navigate to ancestor in tree | Browser | Click selects ancestor node |
| T-13.93 | Root instance breadcrumb shows catalog only | Browser | No parent entries |

### Operational UI — Reference Navigation

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-13.94 | References tab shows forward references | Browser | Table with assoc name, type, target |
| T-13.95 | Referenced-by tab shows reverse references | Browser | Table with assoc name, type, source |
| T-13.96 | Clicking referenced instance navigates to it in tree | Browser | Tree node selected, detail updates |
| T-13.97 | No references shows empty state message | Browser | "No references" text |

### Operational UI — Read-Only Enforcement

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-13.98 | No create buttons visible in operational UI | Browser | No "Create" buttons anywhere |
| T-13.99 | No edit buttons visible in operational UI | Browser | No "Edit" buttons anywhere |
| T-13.100 | No delete buttons visible in operational UI | Browser | No "Delete" buttons anywhere |
| T-13.101 | No link/unlink actions in reference tabs | Browser | No write actions on references |
| T-13.102 | Read-only enforcement applies regardless of role (even SuperAdmin) | Browser | SuperAdmin sees same read-only view |

## Milestone 14: Catalog-Level RBAC

Phase 5 of the catalog implementation plan. Adds per-catalog access control via a `CatalogAccessChecker` interface. In header-based dev mode (RBAC_MODE=header), all catalogs are accessible. In SAR mode (Phase C), SubjectAccessReview checks resourceName against K8s RBAC. User stories: US-23, US-39.

### CatalogAccessChecker Interface + HeaderCatalogAccessChecker

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-14.01 | `HeaderCatalogAccessChecker.CheckAccess` returns true for any catalog | Unit | Always allowed in dev mode |
| T-14.02 | `HeaderCatalogAccessChecker.CheckAccess` returns true for any verb | Unit | GET, POST, PUT, DELETE all allowed |

### RequireCatalogAccess Middleware

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-14.03 | Middleware extracts catalog name from `:catalog-name` param | Unit | Correct name passed to CheckAccess |
| T-14.04 | Middleware maps GET to verb "get" | Unit | CheckAccess called with verb "get" |
| T-14.05 | Middleware maps POST to verb "create" | Unit | CheckAccess called with verb "create" |
| T-14.06 | Middleware maps PUT to verb "update" | Unit | CheckAccess called with verb "update" |
| T-14.07 | Middleware maps DELETE to verb "delete" | Unit | CheckAccess called with verb "delete" |
| T-14.08 | Middleware returns 403 when CheckAccess returns false | Unit | 403 Forbidden response |
| T-14.09 | Middleware passes through when CheckAccess returns true | Unit | Next handler called, 200 response |
| T-14.10 | Middleware returns 500 when CheckAccess returns error | Unit | 500 Internal Server Error |
| T-14.11 | Middleware skips check when no catalog name in path (catalog list) | Unit | Next handler called without CheckAccess |

### Catalog List Filtering

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-14.12 | Catalog list filters out denied catalogs | Unit | Only allowed catalogs returned |
| T-14.13 | Catalog list returns all catalogs when all are allowed | Unit | Full list returned |
| T-14.14 | Catalog list returns empty when all catalogs denied | Unit | Empty items, total=0 |
| T-14.15 | `GET /api/data/v1/catalogs` with mock deny returns filtered list | API | Denied catalogs excluded from response |
| T-14.16 | `GET /api/data/v1/catalogs` with header mode returns all catalogs | API | All catalogs in response |

### Catalog Access Enforcement via API

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-14.17 | `GET /api/data/v1/catalogs/{name}/tree` returns 403 when denied | API | 403 response |
| T-14.18 | `GET /api/data/v1/catalogs/{name}/{type}` returns 403 when denied | API | 403 response |
| T-14.19 | `POST /api/data/v1/catalogs/{name}/{type}` returns 403 when denied | API | 403 response |
| T-14.20 | `GET /api/data/v1/catalogs/{name}/{type}/{id}` returns 403 when denied | API | 403 response |
| T-14.21 | All sub-resource operations allowed when catalog access granted | API | 200 response |
| T-14.22 | Header mode: all catalog operations pass regardless of catalog name | API | 200 response for any catalog |

---

## Milestone 15: Catalog Validation

Phase 6 of the catalog implementation plan. On-demand validation of all entity instances in a catalog against the pinned CV's schema. The `CatalogValidationService` checks required attributes, type correctness, full cardinality validation (min and max, both target and source directions), containment consistency (orphaned instances, missing parents for contained types, invalid relationships), and unpinned entity types. Returns a structured error list and updates the catalog's validation status. User story: US-34.

### Validation Service — Required Attributes

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-15.01 | Instance missing value for required attribute produces error | Unit | Error: entity_type, instance_name, field=attr_name, violation="required attribute missing" |
| T-15.02 | Instance with value for required attribute passes | Unit | No error for that attribute |
| T-15.03 | Instance missing value for optional attribute passes | Unit | No error — optional attrs are not required |
| T-15.04 | Multiple instances missing different required attrs produce separate errors | Unit | One error per missing required attr per instance |

### Validation Service — Attribute Type Check

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-15.05 | String attribute with any value passes | Unit | No error |
| T-15.06 | Number attribute with valid float value passes | Unit | No error |
| T-15.07 | Required number attribute with nil value produces error | Unit | Error: violation="required attribute missing" |
| T-15.08 | Enum attribute with value in allowed list passes | Unit | No error |
| T-15.09 | Enum attribute with value not in allowed list produces error | Unit | Error: violation="invalid enum value" |

### Validation Service — Mandatory Associations

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-15.10 | Association with target_cardinality "1" — source instance has one link → passes | Unit | No error |
| T-15.11 | Association with target_cardinality "1" — source instance has no link → error | Unit | Error: violation="mandatory association unsatisfied" |
| T-15.12 | Association with target_cardinality "1..n" — source instance has one link → passes | Unit | No error |
| T-15.13 | Association with target_cardinality "1..n" — source instance has no link → error | Unit | Error: violation="mandatory association unsatisfied" |
| T-15.14 | Association with target_cardinality "0..n" — source instance has no link → passes | Unit | No error — optional association |
| T-15.15 | Association with target_cardinality "0..1" — source instance has no link → passes | Unit | No error — optional association |
| T-15.16 | Containment associations are excluded from mandatory assoc checks | Unit | Containment validated separately |

### Validation Service — Cardinality Max and Source Direction

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-15.49 | Target cardinality "0..1" with 2 links → max exceeded error | Unit | Error: violation="exceeds maximum" |
| T-15.50 | Exact cardinality "1" with 2 links → max exceeded error | Unit | Error: violation="exceeds maximum" |
| T-15.51 | Bidirectional with source_cardinality "1" — target instance has no reverse links → error | Unit | Error: violation="source cardinality" |
| T-15.52 | Directional with source_cardinality "1" — target instance has reverse link → passes | Unit | No error |
| T-15.53 | Source cardinality "0..1" with 2 reverse links → max exceeded error | Unit | Error: violation="exceeds maximum" |
| T-15.54 | Bidirectional with mandatory target_cardinality — source instance has no link → error | Unit | Error: violation="mandatory association" |
| T-15.55 | ParseCardinality parses all formats correctly | Unit | "0..n"→(0,0,true), "1"→(1,1,false), "1..n"→(1,0,true), etc. |
| T-15.56 | ParseCardinality with invalid max returns unbounded | Unit | "1..abc"→(1,0,true) |

### Validation Service — Containment Consistency

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-15.17 | Contained instance with valid parent (correct type, exists) passes | Unit | No error |
| T-15.18 | Instance with ParentInstanceID pointing to non-existent instance → error | Unit | Error: violation="orphaned contained instance" |
| T-15.19 | Instance with parent whose entity type has no containment assoc to child type → error | Unit | Error: violation="invalid containment relationship" |
| T-15.20 | Top-level instance (no ParentInstanceID) passes containment check | Unit | No error |
| T-15.57 | Contained entity type instance without parent → error | Unit | Error: violation="contained entity type requires a parent" |
| T-15.58 | Contained entity type instance with valid parent → passes | Unit | No error |
| T-15.59 | Parent entity type not pinned in CV → error | Unit | Error: violation="parent entity type not pinned" |

### Validation Service — Status Update and Edge Cases

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-15.21 | All checks pass → catalog status set to `valid` | Unit | UpdateValidationStatus called with "valid" |
| T-15.22 | Any check fails → catalog status set to `invalid` | Unit | UpdateValidationStatus called with "invalid" |
| T-15.23 | Empty catalog (no instances) → passes validation, status `valid` | Unit | No errors, status "valid" |
| T-15.24 | Nonexistent catalog returns NotFound error | Unit | NotFound error returned |
| T-15.25 | Error list contains entity_type, instance_name, field, violation for each error | Unit | Structured error with all four fields |
| T-15.60 | Instance of unpinned entity type produces error | Unit | Error: violation="not pinned" |
| T-15.61 | Entity type name fallback to ID when etRepo.GetByID fails | Unit | Uses entity type ID as name |

### Validation Service — Error Propagation

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-15.62 | ListByCatalog error propagated | Unit | Error returned |
| T-15.63 | Empty catalog UpdateValidationStatus error propagated | Unit | Error returned |
| T-15.64 | ListByCatalogVersion error propagated | Unit | Error returned |
| T-15.65 | etvRepo.GetByID error during pin resolution propagated | Unit | Error returned |
| T-15.66 | attrRepo.ListByVersion error propagated | Unit | Error returned |
| T-15.67 | enumValRepo.ListByEnum error propagated | Unit | Error returned |
| T-15.68 | iavRepo.GetCurrentValues error propagated | Unit | Error returned |
| T-15.69 | linkRepo.GetForwardRefs error propagated | Unit | Error returned |
| T-15.70 | linkRepo.GetReverseRefs error propagated | Unit | Error returned |
| T-15.71 | Final UpdateValidationStatus error propagated | Unit | Error returned |
| T-15.72 | assocRepo.ListByVersion error during pre-load propagated | Unit | Error returned |
| T-15.73 | IsEmptyValue returns true for unknown attribute type | Unit | true |

### Validate API Handler — Additional

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-15.74 | Nil validationSvc returns 501 Not Implemented | API | 501 response |

### Instance Service — Clear Attribute Value

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-15.75 | UpdateInstance with empty string clears attribute (not carried forward) | Unit | Attribute value cleared |

### useValidation Hook

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-15.76 | Undefined catalogName does nothing on validate | Browser | API not called |
| T-15.77 | Works without onComplete callback | Browser | Status set to valid |
| T-15.78 | Calls onComplete after validation | Browser | Callback invoked |
| T-15.79 | Handles missing errors field in response | Browser | Empty errors array |
| T-15.80 | API error resets validating state | Browser | validating=false after error |

### Operational UI — Additional

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-15.81 | Validate button hidden for RO in operational UI | Browser | Button not rendered |

### Validation Service — Integration Tests

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-15.26 | Full validation with valid catalog (all attrs set, all mandatory assocs linked, containment correct) | Integration | No errors, status "valid" |
| T-15.27 | Full validation with missing required attribute on one instance | Integration | One error for that instance/attribute |
| T-15.28 | Full validation with invalid enum value on one instance | Integration | One error for that instance/attribute |
| T-15.29 | Full validation with mandatory association missing link | Integration | One error for that instance/association |
| T-15.30 | Full validation with orphaned contained instance | Integration | One error for that instance |
| T-15.31 | Full validation with multiple violations across entity types | Integration | Multiple errors, status "invalid" |
| T-15.32 | Validation status persisted correctly after validation | Integration | DB query confirms status updated |

### Validate API Handler

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-15.33 | `POST /api/data/v1/catalogs/{name}/validate` returns 200 with valid catalog | API | `{status: "valid", errors: []}` |
| T-15.34 | `POST /api/data/v1/catalogs/{name}/validate` returns 200 with invalid catalog | API | `{status: "invalid", errors: [...]}` |
| T-15.35 | `POST /api/data/v1/catalogs/{name}/validate` with nonexistent catalog → 404 | API | 404 Not Found |
| T-15.36 | `POST /api/data/v1/catalogs/{name}/validate` as RO → 403 | API | 403 Forbidden |
| T-15.37 | `POST /api/data/v1/catalogs/{name}/validate` as RW → 200 | API | Allowed |
| T-15.38 | Validate response errors include entity_type, instance_name, field, violation | API | All four fields in each error object |

### Catalog Detail UI — Validate Button (Meta)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-15.39 | Validate button visible for RW user on catalog detail | Browser | Button rendered |
| T-15.40 | Validate button visible for Admin user | Browser | Button rendered |
| T-15.41 | Validate button hidden for RO user | Browser | Button not rendered |
| T-15.42 | Clicking Validate calls POST .../validate API | Browser | API called |
| T-15.43 | Successful validation with no errors shows "valid" status badge | Browser | Green "valid" label |
| T-15.44 | Validation with errors shows "invalid" status badge | Browser | Red "invalid" label |
| T-15.45 | Validation errors displayed grouped by entity type | Browser | Errors grouped under entity type headings |
| T-15.46 | Each error shows instance name, field, and violation | Browser | Error details visible |

### Operational UI — Validate Button

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-15.47 | Validate button visible on operational catalog detail | Browser | Button rendered |
| T-15.48 | Validation results displayed in operational UI | Browser | Errors shown after validation |

---

## Milestone 16: Catalog Publishing, K8s CRs & Promotion Warnings

Phase 7 of the catalog implementation plan. Explicit publish/unpublish operations for catalogs. Publishing creates a namespaced Catalog CR in K8s for discovery. Published catalogs are write-protected — data mutations require SuperAdmin role. The operator reconciles Catalog CRs, sets owner references, and increments `status.DataVersion` for consumer cache invalidation. CV promotion warns about draft/invalid catalogs. User stories: US-42, US-43.

### Catalog Model — Published Field

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-16.01 | Catalog model has `published` boolean field (default false) | Integration | New catalog has published=false |
| T-16.02 | Catalog model has `published_at` timestamp field (default nil) | Integration | New catalog has published_at=nil |

### Publish Catalog — Service

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-16.03 | Publish a `valid` catalog sets published=true and published_at | Unit | Catalog updated |
| T-16.04 | Publish a `draft` catalog returns error | Unit | Validation error |
| T-16.05 | Publish an `invalid` catalog returns error | Unit | Validation error |
| T-16.06 | Publish nonexistent catalog returns NotFound | Unit | NotFound error |
| T-16.07 | Publish calls CatalogCRManager.CreateOrUpdate with correct spec | Unit | CR created with catalog name, CV label, validation status |
| T-16.08 | Publish with nil crManager skips CR operation (DB-only) | Unit | No panic, published=true in DB |
| T-16.09 | Publish already-published catalog is idempotent | Unit | No error, published stays true |
| T-16.10 | Publish persists published=true and published_at in database | Integration | DB query confirms fields |

### Unpublish Catalog — Service

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-16.11 | Unpublish sets published=false | Unit | Catalog updated |
| T-16.12 | Unpublish calls CatalogCRManager.Delete | Unit | CR deleted |
| T-16.13 | Unpublish with nil crManager skips CR operation (DB-only) | Unit | No panic, published=false in DB |
| T-16.14 | Unpublish already-unpublished catalog is idempotent | Unit | No error |
| T-16.15 | Unpublish nonexistent catalog returns NotFound | Unit | NotFound error |
| T-16.16 | Unpublish persists published=false in database | Integration | DB query confirms field |

### Published Catalog Write Protection — Service

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-16.17 | CreateInstance on published catalog as RW → 403 | Unit | Forbidden error |
| T-16.18 | CreateInstance on published catalog as SuperAdmin → succeeds | Unit | Instance created |
| T-16.19 | UpdateInstance on published catalog as RW → 403 | Unit | Forbidden error |
| T-16.20 | DeleteInstance on published catalog as RW → 403 | Unit | Forbidden error |
| T-16.21 | CreateContainedInstance on published catalog as RW → 403 | Unit | Forbidden error |
| T-16.22 | CreateAssociationLink on published catalog as RW → 403 | Unit | Forbidden error |
| T-16.23 | DeleteAssociationLink on published catalog as RW → 403 | Unit | Forbidden error |
| T-16.24 | SetParent on published catalog as RW → 403 | Unit | Forbidden error |
| T-16.25 | SuperAdmin mutation on published catalog resets status to draft | Unit | Status=draft, published=true |
| T-16.26 | Draft does not auto-unpublish — published stays true after mutation | Unit | published=true after mutation |

### Published Catalog Write Protection — Integration

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-16.27 | Publish catalog, create instance as SuperAdmin, verify published=true and status=draft | Integration | Full round-trip in DB |
| T-16.28 | Published field survives create→publish→mutate→query round-trip | Integration | published=true persisted through mutations |

### Publish/Unpublish API Handler

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-16.29 | `POST /catalogs/{name}/publish` as Admin → 200 | API | Published successfully |
| T-16.30 | `POST /catalogs/{name}/publish` as RW → 403 | API | Forbidden |
| T-16.31 | `POST /catalogs/{name}/publish` as RO → 403 | API | Forbidden |
| T-16.32 | `POST /catalogs/{name}/publish` on draft catalog → 400 | API | Bad request |
| T-16.33 | `POST /catalogs/{name}/publish` on nonexistent → 404 | API | Not found |
| T-16.34 | `POST /catalogs/{name}/unpublish` as Admin → 200 | API | Unpublished successfully |
| T-16.35 | `POST /catalogs/{name}/unpublish` as RW → 403 | API | Forbidden |
| T-16.36 | Instance create on published catalog as RW → 403 | API | Forbidden |
| T-16.37 | Instance create on published catalog as SuperAdmin → 201 | API | Instance created |
| T-16.38 | Catalog response includes `published` and `published_at` fields | API | Fields in JSON response |

### CV Promotion Warnings — Service

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-16.39 | Promote CV with draft catalog pinned → warning in response | Unit | Warning includes catalog name and status |
| T-16.40 | Promote CV with invalid catalog pinned → warning in response | Unit | Warning includes catalog name and status |
| T-16.41 | Promote CV with all valid catalogs → no warnings | Unit | Empty warnings list |
| T-16.42 | Promote CV with no catalogs pinned → no warnings | Unit | Empty warnings list |
| T-16.43 | Promotion proceeds despite warnings (not blocked) | Unit | Lifecycle stage updated |

### CV Promotion Warnings — Integration

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-16.44 | Create CV, pin catalogs with draft/valid/invalid statuses, promote → correct warnings | Integration | Warnings for draft and invalid only |

### CV Promotion Warnings — API

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-16.45 | `POST /catalog-versions/{id}/promote` response includes `warnings` array | API | Warnings in JSON response |
| T-16.46 | Promotion response with no warnings has empty array | API | `warnings: []` |

### Catalog CR Manager

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-16.47 | CatalogCRManager.CreateOrUpdate creates Catalog CR with correct spec | Unit | CR has catalog name, CV label, status |
| T-16.48 | CatalogCRManager.CreateOrUpdate sets annotations (source-db-id, published-at) | Unit | Annotations present |
| T-16.49 | CatalogCRManager.CreateOrUpdate updates existing CR idempotently | Unit | Spec updated, no duplicate |
| T-16.50 | CatalogCRManager.Delete removes Catalog CR | Unit | CR deleted |
| T-16.51 | CatalogCRManager.Delete on nonexistent CR returns nil (idempotent) | Unit | No error |

### Operator — Catalog CR Reconciliation

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-16.52 | Reconciler sets owner reference on Catalog CR to AssetHub CR | Operator | OwnerReference set |
| T-16.53 | Reconciler updates Catalog CR status.Ready=true | Operator | Status updated |
| T-16.54 | New Catalog CR has DataVersion=0 before reconciliation | Operator | DataVersion is zero value |
| T-16.55 | First reconciliation sets DataVersion=1 | Operator | DataVersion incremented from 0 to 1 |
| T-16.56 | Subsequent reconciliation increments DataVersion | Operator | DataVersion incremented from existing value |

### Catalog Detail UI — Publish Button (Meta)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-16.57 | Publish button visible for Admin on valid unpublished catalog | Browser | Button rendered |
| T-16.58 | Publish button hidden for RW user | Browser | Button not rendered |
| T-16.59 | Publish button hidden when catalog is draft | Browser | Button not rendered |
| T-16.60 | Publish button hidden when catalog is invalid | Browser | Button not rendered |
| T-16.61 | Unpublish button visible on published catalog for Admin | Browser | Button rendered |
| T-16.62 | Clicking Publish calls POST .../publish API | Browser | API called |
| T-16.63 | Published badge shown on catalog detail after publish | Browser | "published" indicator visible |
| T-16.64 | Published badge shown in catalog list | Browser | "published" indicator in list |
| T-16.65 | Warning banner shown on published catalog for RW user | Browser | Banner visible |
| T-16.66 | Instance create/edit/delete controls disabled for RW on published catalog | Browser | Controls disabled |
| T-16.67 | Instance controls enabled for SuperAdmin on published catalog | Browser | Controls enabled |

### CV Promotion UI — Warnings

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-16.68 | Promote dialog shows warnings for draft/invalid catalogs | Browser | Warning text visible |
| T-16.69 | Promote dialog shows no warnings when all catalogs valid | Browser | No warning shown |

---

## Milestone 17: Copy & Replace Catalog

Phase 8 of the catalog implementation plan. Copy Catalog deep-clones all data from a source catalog (instances, attribute values, association links, containment hierarchy) into a new catalog with new UUIDs and remapped references. Replace Catalog atomically swaps a staging catalog into the name of a published one, archiving the old catalog. Both operations are transactional. User stories: US-44, US-45, US-46.

### Repository — UpdateName

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-17.01 | UpdateName changes catalog name in database | Integration | Name updated, other fields unchanged |
| T-17.02 | UpdateName with name that already exists returns ConflictError | Integration | ConflictError returned |
| T-17.03 | UpdateName with nonexistent catalog ID returns NotFoundError | Integration | NotFoundError returned |
| T-17.04 | UpdatePublished via UpdateName preserves published state | Integration | published and published_at unchanged after rename |

### Copy Catalog — Service

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-17.05 | Copy creates new catalog with same CV pin and draft status | Unit | New catalog with source's CV ID, status=draft |
| T-17.06 | Copy uses provided description (or source description if empty) | Unit | Description set correctly |
| T-17.07 | Copy clones all instances with new UUIDs | Unit | Instance count matches, IDs differ |
| T-17.08 | Copy preserves instance entity type, name, and description | Unit | Fields match source instances |
| T-17.09 | Copy resets instance version to 1 | Unit | All cloned instances at version 1 |
| T-17.10 | Copy clones attribute values remapped to new instance IDs | Unit | Values match, instance IDs remapped |
| T-17.11 | Copy clones association links remapped to new source/target IDs | Unit | Links match, source/target IDs remapped |
| T-17.12 | Copy preserves containment hierarchy — parent refs remapped | Unit | Parent-child relationships intact with new IDs |
| T-17.13 | Copy with nonexistent source returns NotFoundError | Unit | NotFoundError |
| T-17.14 | Copy with invalid target name returns validation error | Unit | Validation error |
| T-17.15 | Copy with duplicate target name returns ConflictError | Unit | ConflictError |
| T-17.16 | Copy of empty catalog (no instances) creates empty catalog | Unit | Catalog created, zero instances |
| T-17.17 | Copy with self-referential links (src and tgt in same catalog) remaps correctly | Unit | Both ends of link remapped to new IDs |
| T-17.18 | Copy is transactional — partial failure creates no catalog | Unit | No catalog/instances on error |
| T-17.19 | Copy does not modify source catalog | Unit | Source unchanged after copy |

### Copy Catalog — Integration (End-to-End)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-17.20 | Copy catalog with instances, attributes, links, containment in real DB | Integration | All data cloned with new IDs |
| T-17.21 | Copied instances have correct catalog_id pointing to new catalog | Integration | FK integrity maintained |
| T-17.22 | Copied attribute values retrievable via GetCurrentValues on new instances | Integration | Values match source |
| T-17.23 | Copied links retrievable via GetForwardRefs/GetReverseRefs on new instances | Integration | Links point to new instances |
| T-17.24 | Containment hierarchy intact — ListByParent on new parent returns new children | Integration | Children found under new parent |
| T-17.25 | Original catalog data unchanged after copy | Integration | Source instances/attrs/links untouched |

### Replace Catalog — Service

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-17.26 | Replace renames target to archive name | Unit | Target gets archive name |
| T-17.27 | Replace renames source to target's original name | Unit | Source gets target's name |
| T-17.28 | Replace with default archive name uses `{target}-archive-{timestamp}` | Unit | Name matches pattern |
| T-17.29 | Replace with custom archive name uses provided name | Unit | Custom archive name used |
| T-17.30 | Replace requires source validation status `valid` | Unit | Error for draft source |
| T-17.31 | Replace requires source validation status `valid` (invalid) | Unit | Error for invalid source |
| T-17.32 | Replace with nonexistent source returns NotFoundError | Unit | NotFoundError |
| T-17.33 | Replace with nonexistent target returns NotFoundError | Unit | NotFoundError |
| T-17.34 | Replace where source equals target returns error | Unit | Validation error |
| T-17.35 | Replace with invalid archive name (non-DNS-label) returns error | Unit | Validation error |
| T-17.36 | Replace with archive name that already exists returns ConflictError | Unit | ConflictError |
| T-17.37 | Replace transfers published state: target was published → source inherits published=true | Unit | Source gets published=true, published_at |
| T-17.38 | Replace transfers published state: archive becomes unpublished | Unit | Archive gets published=false, published_at=nil |
| T-17.39 | Replace on unpublished target: both source and archive remain unpublished | Unit | No published state change |
| T-17.40 | Replace calls SyncCR after swap to update Catalog CR spec | Unit | SyncCR called with target name |
| T-17.41 | Replace deletes archive catalog's CR (old name no longer valid) | Unit | CRManager.Delete called for archive |
| T-17.42 | Replace bumps CR DataVersion so consumers detect the swap | Unit | CreateOrUpdate called with incremented SyncVersion |
| T-17.43 | Replace is transactional — failure in second rename rolls back first | Unit | Both catalogs retain original names |
| T-17.44 | Replace with nil crManager skips CR operations (DB-only) | Unit | No panic, names swapped |

### Replace Catalog — Integration (End-to-End)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-17.45 | Replace swaps names in real DB | Integration | Names swapped correctly |
| T-17.46 | Replace transfers published state in real DB | Integration | published=true on renamed source |
| T-17.47 | Instances in renamed catalogs retain correct catalog_id | Integration | FK integrity maintained |
| T-17.48 | Replace with draft source leaves DB unchanged | Integration | No names changed |
| T-17.49 | Replace with nonexistent source leaves DB unchanged | Integration | No names changed |
| T-17.50 | Replace with archive name collision leaves DB unchanged | Integration | No names changed |

### Copy Catalog — API Handler

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-17.51 | `POST /api/data/v1/catalogs/copy` returns 201 with new catalog | API | CatalogResponse with new ID |
| T-17.52 | Copy response includes resolved CV label | API | catalog_version_label present |
| T-17.53 | Copy with nonexistent source → 404 | API | Not found |
| T-17.54 | Copy with duplicate target name → 409 | API | Conflict |
| T-17.55 | Copy with invalid target name → 400 | API | Bad request |
| T-17.56 | Copy as RO → 403 | API | Forbidden |
| T-17.57 | Copy as RW → 201 (RW can create catalogs) | API | Success |
| T-17.58 | Copy request binds source, name, description | API | Fields correctly extracted |

### Replace Catalog — API Handler

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-17.59 | `POST /api/data/v1/catalogs/replace` returns 200 with updated catalog | API | CatalogResponse |
| T-17.60 | Replace with non-valid source → 400 | API | Bad request |
| T-17.61 | Replace with nonexistent source → 404 | API | Not found |
| T-17.62 | Replace with nonexistent target → 404 | API | Not found |
| T-17.63 | Replace with invalid archive name → 400 | API | Bad request |
| T-17.64 | Replace as RO → 403 | API | Forbidden |
| T-17.65 | Replace as RW → 403 | API | Forbidden |
| T-17.66 | Replace as Admin → 200 | API | Success |
| T-17.67 | Replace request binds source, target, archive_name | API | Fields correctly extracted |

### Copy Catalog — UI (Meta)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-17.68 | Copy button visible on catalog detail for RW+ users | Browser | Button rendered |
| T-17.69 | Copy button hidden for RO users | Browser | Button not rendered |
| T-17.70 | Copy modal opens with name input | Browser | Modal with text field |
| T-17.71 | Copy modal validates DNS-label format (error for invalid) | Browser | Inline validation error |
| T-17.72 | Successful copy calls POST /catalogs/copy API | Browser | API called with correct body |
| T-17.73 | Copy error (409 conflict) shows alert | Browser | Error alert displayed |
| T-17.74 | Successful copy refreshes catalog list or navigates to new catalog | Browser | List refreshed / navigation occurs |

### Replace Catalog — UI (Meta)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-17.75 | Replace button visible on valid catalog for Admin+ users | Browser | Button rendered |
| T-17.76 | Replace button hidden for RW users | Browser | Button not rendered |
| T-17.77 | Replace button hidden for RO users | Browser | Button not rendered |
| T-17.78 | Replace button hidden for draft catalogs | Browser | Button not rendered |
| T-17.79 | Replace button hidden for invalid catalogs | Browser | Button not rendered |
| T-17.80 | Replace modal opens with target dropdown and archive name input | Browser | Modal with dropdown + input |
| T-17.81 | Replace modal target dropdown shows existing catalogs | Browser | Catalog names listed |
| T-17.82 | Replace modal archive name validates DNS-label format | Browser | Inline validation error for invalid |
| T-17.83 | Successful replace calls POST /catalogs/replace API | Browser | API called with correct body |
| T-17.84 | Replace error shows alert | Browser | Error alert displayed |
| T-17.85 | Successful replace refreshes catalog list | Browser | List refreshed |

### Operator — CR DataVersion After Replace

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-17.86 | SyncCR after replace triggers CreateOrUpdate with incremented SyncVersion | Operator | SyncVersion bumped in CR spec |

### Copy & Replace — API Client (Browser)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-17.87 | copyCatalog client function sends POST with correct body | Browser | mockFetch called correctly |
| T-17.88 | replaceCatalog client function sends POST with correct body | Browser | mockFetch called correctly |

---

## 18. System Attributes — Common Attributes as Schema-Level Attributes (TD-22)

Common attributes (Name — required, Description — optional) are hardcoded fields on `EntityInstance` but are surfaced as synthetic system attributes (`system: true`) in all API responses via Approach B (API-level merge). No DB schema changes.

### Instance DTO — System Attribute Injection

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-18.01 | instanceDetailToDTO prepends Name system attr (type=string, required=true, system=true) | Unit | First attr: name="name", type="string", required=true, system=true |
| T-18.02 | instanceDetailToDTO prepends Description system attr (type=string, required=false, system=true) | Unit | Second attr: name="description", type="string", required=false, system=true |
| T-18.03 | System attr values match instance Name and Description fields | Unit | name attr value = inst.Name, description attr value = inst.Description |
| T-18.04 | Custom attributes follow system attrs and have system=false | Unit | Custom attrs start at index 2, system=false |
| T-18.05 | Instance with zero custom attrs still has 2 system attrs | Unit | Attributes array length = 2 |
| T-18.06 | System attrs injected in list instances response (each item) | Unit | Every item in list has system attrs prepended |

### Version Snapshot — System Attribute Injection

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-18.07 | Version snapshot prepends Name system attr (ordinal=-2, system=true, required=true) | Unit | First attr in snapshot |
| T-18.08 | Version snapshot prepends Description system attr (ordinal=-1, system=true, required=false) | Unit | Second attr in snapshot |
| T-18.09 | Custom attrs in snapshot retain original ordinals (>= 0) | Unit | Custom attr ordinals unchanged |

### Attribute List — System Attribute Injection

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-18.10 | Attribute list endpoint prepends Name and Description system attrs | Unit | First two entries are system attrs |
| T-18.11 | System attrs in attribute list have correct types and required flags | Unit | name: required=true, description: required=false |

### Reserved Name Rejection

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-18.12 | Create attribute with name "name" is rejected | Unit | Validation error returned |
| T-18.13 | Create attribute with name "description" is rejected | Unit | Validation error returned |
| T-18.14 | Create attribute with name "Name" (uppercase) is allowed | Unit | Attribute created (names are case-sensitive) |
| T-18.15 | Rename attribute to "name" is rejected | Unit | Validation error returned |
| T-18.16 | Rename attribute to "description" is rejected | Unit | Validation error returned |
| T-18.17 | Create attribute "name" returns 400 via API | API | HTTP 400 with error message |
| T-18.18 | Create attribute "description" returns 400 via API | API | HTTP 400 with error message |

### Copy Attributes — System Attribute Exclusion

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-18.19 | CopyAttributes with "name" in list silently skips it | Unit | No error, only custom attrs copied |
| T-18.20 | CopyAttributes with "description" in list silently skips it | Unit | No error, only custom attrs copied |
| T-18.21 | CopyAttributes with mix of system and custom names copies only custom | Unit | Custom attrs copied, system skipped |

### Catalog Validation — Name Non-Empty Check

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-18.22 | Validate returns error for instance with empty Name | Unit | ValidationError: field="name", violation contains "required" |
| T-18.23 | Validate returns error for instance with whitespace-only Name | Unit | ValidationError: field="name" |
| T-18.24 | Validate passes for instance with non-empty Name | Unit | No name-related error |
| T-18.25 | Validation error includes correct entity type name | Unit | EntityType field resolved to name |
| T-18.26 | Validate empty-named instance end-to-end in SQLite | Integration | Status = invalid, error for name field |

### Instance CRUD API — System Attrs in Response

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-18.27 | POST create instance response includes system attrs in attributes array | API | System attrs present with correct values |
| T-18.28 | GET instance response includes system attrs | API | System attrs present |
| T-18.29 | GET list instances includes system attrs per item | API | Every item has system attrs |
| T-18.30 | PUT update instance response reflects updated name/description in system attrs | API | System attr values match updated fields |

### Meta API — Snapshot and Attribute List

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-18.31 | GET version snapshot includes system attrs at start of attributes array | API | First two attrs are system, system=true |
| T-18.32 | GET attribute list includes system attrs at start | API | First two entries are system attrs |

### Meta UI — Attribute List with System Badge

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-18.33 | Entity type detail shows "Name" system attribute with System badge | Browser | Row with "Name" and badge visible |
| T-18.34 | Entity type detail shows "Description" system attribute with System badge | Browser | Row with "Description" and badge visible |
| T-18.35 | System attributes appear before custom attributes | Browser | Name and Description rows first |
| T-18.36 | Delete button disabled/hidden for system attributes | Browser | No delete action for system rows |
| T-18.37 | Edit button disabled/hidden for system attributes | Browser | No edit action for system rows |

### Meta UI — Copy Attributes Picker

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-18.38 | Copy attributes picker excludes system attributes from source list | Browser | "Name" and "Description" not in picker |

### Operational UI — Create Instance Modal

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-18.39 | Create modal renders Name field from schema attrs (not hardcoded) | Browser | Name field rendered with required indicator |
| T-18.40 | Create modal renders Description field from schema attrs | Browser | Description field rendered without required indicator |
| T-18.41 | Create modal renders custom attributes after system attributes | Browser | System attrs first, then custom |
| T-18.42 | Create submits name and description as top-level request fields | Browser | API called with {name, description, attributes} |

### Operational UI — Edit Instance Modal

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-18.43 | Edit modal shows Name and Description from schema attrs | Browser | Fields populated with instance values |
| T-18.44 | Edit submits updated name/description as top-level request fields | Browser | API called with correct body structure |

### API Client — System Attribute Passthrough

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-18.45 | getSnapshot client function returns system attrs in response | Browser | mockFetch response includes system attrs |
| T-18.46 | listAttributes client function returns system attrs in response | Browser | mockFetch response includes system attrs |

---

## 19. Component Decomposition — CatalogDetailPage (TD-23, Phase 1)

Pure refactoring of `CatalogDetailPage.tsx` into 3 custom hooks + 5 modal components. Zero behavior changes.

### useCatalogData Hook

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-19.01 | Loads catalog and pins on mount | Hook | `catalog`, `pins` populated from mock API |
| T-19.02 | Sets first pin as activeTab when no tab selected | Hook | `activeTab` = first pin's entity_type_name |
| T-19.03 | Loads schema attrs when activeTab changes | Hook | `schemaAttrs`, `schemaAssocs` populated from snapshot |
| T-19.04 | Loads enum values for enum-type attributes | Hook | `enumValues` populated for each enum attr |
| T-19.05 | Returns early when name is undefined | Hook | No API calls, no errors |
| T-19.06 | Handles catalog load error | Hook | `error` set, `loading` false |
| T-19.07 | Handles schema load error gracefully | Hook | `schemaAttrs` stays empty, no crash |
| T-19.08 | Reloads on loadCatalog call | Hook | API called again, state refreshed |

### useInstances Hook

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-19.09 | Loads instances for active entity type | Hook | `instances`, `instTotal` populated |
| T-19.10 | Returns early when catalogName is undefined | Hook | No API calls |
| T-19.11 | handleCreate calls API with name, description, attributes | Hook | `api.instances.create` called with correct args |
| T-19.12 | handleCreate with number attribute passes parseFloat value | Hook | Number attribute submitted as float |
| T-19.13 | handleCreate error sets createError | Hook | `createError` contains message |
| T-19.14 | handleEdit calls API with version, changed fields | Hook | `api.instances.update` called with correct args |
| T-19.15 | handleEdit error sets editError | Hook | `editError` contains message |
| T-19.16 | handleDelete calls API and refreshes list | Hook | `api.instances.delete` called, `loadInstances` triggered |
| T-19.17 | handleDelete error sets deleteError | Hook | `deleteError` contains message |
| T-19.18 | openCreate resets form state | Hook | `newInstName`, `newInstDesc`, `newInstAttrs` cleared |
| T-19.19 | openEdit populates form from instance (skips system attrs) | Hook | `editName`, `editDesc`, `editAttrs` set from instance |

### useInstanceDetail Hook

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-19.20 | selectInstance loads parent name, children, refs | Hook | All three populated from API |
| T-19.21 | selectInstance with no parent skips parent name load | Hook | `parentName` empty, no parent API call |
| T-19.22 | selectInstance handles parent name load error (falls back to ID) | Hook | `parentName` = parent UUID |
| T-19.23 | selectInstance handles children load error | Hook | `children` = empty array |
| T-19.24 | selectInstance handles refs load error | Hook | `forwardRefs`, `reverseRefs` = empty arrays |
| T-19.25 | clearSelection resets all detail state | Hook | All detail state cleared |
| T-19.26 | selectInstance with null clears selection | Hook | `selectedInstance` = null |

### CreateInstanceModal Component

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-19.27 | Renders system attrs (Name required, Description optional) | Browser | Name with required indicator, Description without |
| T-19.28 | Renders custom attrs from schemaAttrs | Browser | Fields for each non-system attr |
| T-19.29 | Renders enum select for enum attributes | Browser | Dropdown with enum values |
| T-19.30 | Submit button disabled when name empty | Browser | Button disabled |
| T-19.31 | Calls onSubmit with correct args on submit | Browser | name, description, attributes passed |
| T-19.32 | Shows error when provided | Browser | Alert visible with error message |
| T-19.33 | Resets form on close | Browser | Fields cleared after close |

### EditInstanceModal Component

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-19.34 | Pre-fills form from instance data | Browser | Name, description, attrs populated |
| T-19.35 | Calls onSubmit with updated fields | Browser | Changed values passed |
| T-19.36 | Shows error when provided | Browser | Alert visible |
| T-19.37 | Closed when instance is null | Browser | Modal not rendered |

### AddChildModal Component

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-19.38 | Shows child type selector from containment assocs | Browser | Dropdown with containment targets |
| T-19.39 | Loads child schema on type selection | Browser | Attribute fields appear |
| T-19.40 | Create mode: shows name, description, attr fields | Browser | Form fields rendered |
| T-19.41 | Adopt mode: shows instance selector | Browser | Instance dropdown visible |
| T-19.42 | Calls onSubmit with create data | Browser | Type, mode, name, attrs passed |
| T-19.43 | Calls onSubmit with adopt data | Browser | Type, mode, instanceId passed |
| T-19.44 | Shows error when provided | Browser | Alert visible |

### LinkModal Component

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-19.45 | Shows association selector from outgoing assocs | Browser | Dropdown with non-containment assocs |
| T-19.46 | Loads target instances on association selection | Browser | Target dropdown populated |
| T-19.47 | Submit button disabled until assoc and target selected | Browser | Button disabled |
| T-19.48 | Calls onSubmit with targetId and assocName | Browser | Correct args passed |
| T-19.49 | Shows error when provided | Browser | Alert visible |

### SetParentModal Component

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-19.50 | Shows parent type selector from pins | Browser | Dropdown with entity types |
| T-19.51 | Loads parent instances on type selection | Browser | Instance dropdown populated |
| T-19.52 | Calls onSubmit with parentType and parentId | Browser | Correct args passed |
| T-19.53 | Clear parent button calls onClearParent | Browser | Clear handler called |
| T-19.54 | Shows error when provided | Browser | Alert visible |

### Regression — Existing Page Tests

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-19.55 | All existing CatalogDetailPage browser tests pass | Browser | 131 tests pass unchanged |

---

## 20. Modal State Internalization + Shared Components (TD-23 Phase 4)

Modals internalize form state. Shared `AttributeFormFields` component and `buildTypedAttrs` utility extracted. Copy/Replace modals extracted.

### buildTypedAttrs Utility

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-20.01 | Converts string value to parseFloat for number-type attr | Unit | `{ weight: 3.14 }` |
| T-20.02 | Passes through string value for string-type attr | Unit | `{ hostname: "foo" }` |
| T-20.03 | Passes through string value for enum-type attr | Unit | `{ status: "active" }` |
| T-20.04 | Skips empty string values | Unit | `{}` |
| T-20.05 | Returns empty object for empty input | Unit | `{}` |
| T-20.06 | Handles mix of types correctly | Unit | Number parsed, strings kept, empty skipped |

### AttributeFormFields Component

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-20.07 | Renders Name system attr with required indicator when includeSystem=true | Browser | Name field with required marker |
| T-20.08 | Renders Description system attr without required when includeSystem=true | Browser | Description field, no required |
| T-20.09 | Does not render system attrs when includeSystem=false | Browser | Only custom attrs visible |
| T-20.10 | Renders custom text attr with text input | Browser | TextInput rendered |
| T-20.11 | Renders custom number attr with number input | Browser | Input type=number |
| T-20.12 | Renders enum attr with EnumSelect dropdown | Browser | Select/dropdown rendered |
| T-20.13 | Calls onChange when text input changes | Browser | onChange called with (name, value) |
| T-20.14 | Shows required indicator for required custom attrs | Browser | `*` in label |

### CopyCatalogModal Component

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-20.15 | Renders name and description inputs | Browser | Both fields visible |
| T-20.16 | Submit disabled when name empty | Browser | Button disabled |
| T-20.17 | Calls onSubmit with name and description | Browser | Correct values passed |
| T-20.18 | Shows error alert when error prop set | Browser | Alert visible |
| T-20.19 | Closes and resets on close | Browser | Modal closed, fields cleared |

### ReplaceCatalogModal Component

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-20.20 | Renders target catalog dropdown | Browser | Dropdown with catalog names |
| T-20.21 | Submit disabled when target not selected | Browser | Button disabled |
| T-20.22 | Calls onSubmit with source, target, archiveName | Browser | Correct values passed |
| T-20.23 | Shows error alert when error prop set | Browser | Alert visible |
| T-20.24 | Archive name input is optional | Browser | Can submit without archive name |

### Updated Modal Tests — Internalized State

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-20.25 | CreateInstanceModal: fills form and onSubmit receives (name, desc, attrs) | Browser | Typed values in callback |
| T-20.26 | CreateInstanceModal: resets form on reopen | Browser | Fields cleared |
| T-20.27 | EditInstanceModal: pre-fills from initialValues prop | Browser | Fields populated |
| T-20.28 | EditInstanceModal: onSubmit receives updated values | Browser | Changed values in callback |
| T-20.29 | AddChildModal: onSubmit receives (childType, mode, data) | Browser | Correct typed args |
| T-20.30 | LinkModal: onSubmit receives (targetId, assocName) | Browser | Correct args |
| T-20.31 | SetParentModal: onSubmit receives (parentType, parentId) | Browser | Correct args |
| T-20.32 | AddAttributeModal: onSubmit receives (name, desc, type, enumId, required) | Browser | Correct args |
| T-20.33 | AddAssociationModal: onSubmit receives all assoc fields | Browser | Name, target, type, roles, cardinality |
| T-20.34 | CopyAttributesModal: onSubmit receives (sourceId, version, attrNames) | Browser | Correct args |
| T-20.35 | RenameEntityTypeModal: onSubmit receives (newName, deepCopyAllowed) | Browser | Correct args |

### Regression

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-20.36 | All existing page-level browser tests pass | Browser | 671 tests unchanged |

---

## 21. UML Composition Diamond + Model Diagram Tab (TD-47, US-48)

TD-47 adds UML composition notation (filled diamond marker on parent end) to containment edges in the entity type diagram. US-48 adds a read-only "Model Diagram" tab to both meta and operational catalog detail pages showing the CV's entity type model.

### EntityTypeDiagram — Composition Diamond (TD-47)

#### Unit Tests (buildModel / edge data)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-21.01 | Containment edge data includes diamond marker type | Unit | Edge data has `markerStart: 'diamond'` or equivalent |
| T-21.02 | Directional (reference) edge data does not include diamond marker | Unit | No diamond marker on reference edges |
| T-21.03 | Bidirectional edge data retains existing marker configuration | Unit | Hollow source arrow + filled target arrow preserved |

#### Browser Tests (SVG rendering)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-21.04 | Containment edge renders filled diamond SVG marker on source end | Browser | `<path>` element with diamond shape in containment color |
| T-21.05 | Diamond marker uses containment color (#3e8635) | Browser | Fill attribute is `#3e8635` |
| T-21.06 | Containment edge renders arrowhead on target end | Browser | Arrow marker present on target |
| T-21.07 | Reference edge does not render diamond marker | Browser | No diamond `<path>` on reference edges |
| T-21.08 | Bidirectional edge retains hollow source arrow and filled target arrow | Browser | Existing bidirectional markers unchanged |

### useCatalogDiagram Hook

#### Unit Tests (renderHook)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-21.09 | Returns empty diagramData and loading=false initially | Unit | `{ diagramData: [], diagramLoading: false }` |
| T-21.10 | Loads pins and snapshots when loadDiagram is called | Unit | API calls made, diagramData populated with DiagramEntityType[] |
| T-21.11 | Sets diagramLoading=true during fetch | Unit | Loading state true while promises pending |
| T-21.12 | Returns diagramData after successful fetch | Unit | Array of DiagramEntityType with correct entity types, attributes, associations |
| T-21.13 | Does not re-fetch if diagramData is already loaded | Unit | Second call to loadDiagram does not trigger API calls |
| T-21.14 | Handles API error gracefully — sets error, clears loading | Unit | `diagramLoading: false`, error message set |
| T-21.15 | Handles empty pins list — returns empty diagramData | Unit | `diagramData: []`, no snapshot calls made |

### Meta CatalogDetailPage — Model Diagram Tab

#### Browser Tests

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-21.16 | "Model Diagram" tab exists on catalog detail page | Browser | Tab with text "Model Diagram" rendered |
| T-21.17 | Clicking Model Diagram tab loads diagram data | Browser | API calls for pins + snapshots triggered |
| T-21.18 | Diagram renders entity type nodes with names and attributes | Browser | Entity type names visible in diagram area |
| T-21.19 | Diagram is read-only — no node double-click navigation | Browser | No navigation on double-click |
| T-21.20 | Empty state shown when no entity types are pinned | Browser | Empty state message displayed |

### Operational OperationalCatalogDetailPage — Model Diagram Tab

#### Browser Tests

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-21.21 | "Model Diagram" tab exists on operational catalog detail page | Browser | Tab with text "Model Diagram" rendered |
| T-21.22 | Clicking Model Diagram tab loads diagram data | Browser | API calls for pins + snapshots triggered |
| T-21.23 | Diagram renders entity type nodes from CV | Browser | Entity type names visible in diagram area |
| T-21.24 | Diagram is read-only — no edit interactions | Browser | No edit callbacks triggered |
| T-21.25 | Empty state shown when no entity types are pinned | Browser | Empty state message displayed |

### API Client Tests

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-21.26 | Existing client functions (listPins, snapshot) work for diagram data loading | Browser | mockFetch verifies correct URLs called |

### Regression

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-21.27 | All existing EntityTypeDiagram tests pass | Browser | No regressions in diagram rendering |
| T-21.28 | All existing CatalogDetailPage tests pass | Browser | No regressions in catalog page |
| T-21.29 | All existing OperationalCatalogDetailPage tests pass | Browser | No regressions in operational page |
| T-21.30 | All existing CatalogVersionDetailPage diagram tab tests pass | Browser | No regressions in CV detail diagram |

---

## 22. Landing Page + Unified SPA (US-47)

Merges the two separate SPAs (meta + operational) into a single SPA with route-based views. Landing page at `/` provides navigation to schema management (`/schema`) and catalog data viewers (`/catalogs/:name`).

### LandingPage Component

#### Unit Tests (catalog card rendering)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-22.01 | Catalog card shows name, CV label, validation status badge | Unit | Name, label, colored badge visible |
| T-22.02 | Draft status badge renders blue | Unit | Blue badge with "draft" text |
| T-22.03 | Valid status badge renders green | Unit | Green badge with "valid" text |
| T-22.04 | Invalid status badge renders red | Unit | Red badge with "invalid" text |
| T-22.05 | Published catalog shows published indicator | Unit | Published badge or icon visible |
| T-22.06 | Card with no description renders cleanly | Unit | No crash, no empty description area |

#### Browser Tests (LandingPage)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-22.07 | Landing page renders at root URL | Browser | Landing page content visible |
| T-22.08 | Schema Management card is visible | Browser | Card with "Schema Management" text |
| T-22.09 | Schema Management card links to /schema | Browser | Click navigates to /schema |
| T-22.10 | Catalog cards rendered for each accessible catalog | Browser | One card per catalog from API |
| T-22.11 | Catalog card shows name, CV label, status badge | Browser | All fields visible on card |
| T-22.12 | Clicking catalog card navigates to /catalogs/:name | Browser | URL changes to /catalogs/{name} |
| T-22.13 | Empty state when no catalogs accessible | Browser | "No catalogs" message displayed |
| T-22.14 | Loading state while fetching catalogs | Browser | Spinner visible during fetch |
| T-22.15 | Error state on API failure | Browser | Error alert displayed |

### Unified SPA Routing

#### Browser Tests (App routing)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-22.16 | /schema renders schema management tabs | Browser | Entity Types, Catalog Versions, Enums, Model Diagram tabs visible |
| T-22.17 | /schema/entity-types/:id renders entity type detail | Browser | Entity type detail page content |
| T-22.18 | /schema/catalog-versions/:id renders CV detail | Browser | CV detail page content |
| T-22.19 | /schema/catalogs/:name renders catalog detail (meta) | Browser | Catalog detail page with instance CRUD |
| T-22.20 | /catalogs/:name renders catalog data viewer | Browser | Catalog data viewer with tree browser |
| T-22.21 | Masthead shows "Schema" on /schema pages | Browser | "Schema" text in masthead |
| T-22.22 | Masthead shows "Data Viewer" on /catalogs pages | Browser | "Data Viewer" text in masthead |
| T-22.23 | Masthead brand link navigates to landing page | Browser | Click navigates to / |

### Regression Tests

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-22.24 | All existing App.tsx tests pass with /schema routes | Browser | No regressions |
| T-22.25 | All existing OperationalCatalogDetailPage tests pass at /catalogs/:name | Browser | No regressions |
| T-22.26 | All existing CatalogDetailPage tests pass at /schema/catalogs/:name | Browser | No regressions |
| T-22.27 | All existing EntityTypeDetailPage tests pass at /schema/entity-types/:id | Browser | No regressions |
| T-22.28 | All existing CatalogVersionDetailPage tests pass at /schema/catalog-versions/:id | Browser | No regressions |

### System Tests (Live Deployment)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-22.29 | Landing page loads at root URL in live deployment | System | Page renders with catalog cards |
| T-22.30 | Navigate from landing page to schema management | System | /schema loads, tabs visible |
| T-22.31 | Navigate from landing page to catalog data viewer | System | /catalogs/:name loads, tree browser visible |
| T-22.32 | /schema routes serve correctly through nginx | System | No 404, SPA routing works |
| T-22.33 | /catalogs/:name routes serve correctly through nginx | System | No 404, SPA routing works |
| T-22.34 | Masthead brand link returns to landing page | System | Navigation works end-to-end |

---

## 23. Description Fields — Entity Type List, Enum, Catalog Version (TD-43, TD-45, TD-46)

Adds description fields to Enum and CatalogVersion models, resolves entity type description from latest version into list API, and adds editable description on entity type detail page.

### Backend — Enum Description (TD-45)

#### Unit Tests (service)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-23.01 | CreateEnum with description stores it | Unit | Description persisted |
| T-23.02 | CreateEnum without description defaults to empty | Unit | Empty string stored |
| T-23.03 | UpdateEnum description updates it | Unit | New description persisted |

#### Integration Tests (repository)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-23.04 | Enum description stored and retrieved | Integration | Round-trip matches |
| T-23.05 | Enum with empty description retrieved correctly | Integration | Empty string, not null |

#### API Tests (handler)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-23.06 | POST /enums with description returns it in response | API | Description in response body |
| T-23.07 | GET /enums list includes description field | API | Each enum has description |
| T-23.08 | PUT /enums/:id updates description | API | Updated description returned |

### Backend — CatalogVersion Description

#### Unit Tests (service)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-23.09 | CreateCatalogVersion with description stores it | Unit | Description persisted |
| T-23.10 | CreateCatalogVersion without description defaults to empty | Unit | Empty string stored |

#### Integration Tests (repository)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-23.11 | CatalogVersion description stored and retrieved | Integration | Round-trip matches |

#### API Tests (handler)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-23.12 | POST /catalog-versions with description returns it | API | Description in response body |
| T-23.13 | GET /catalog-versions list includes description | API | Each CV has description |

### Backend — Entity Type List Description (TD-43)

#### API Tests (handler)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-23.14 | GET /entity-types list includes description from latest version | API | Description matches latest EntityTypeVersion.Description |
| T-23.15 | Entity type with no description returns empty string | API | `description: ""` |
| T-23.16 | Entity type description updates after new version created | API | Description reflects new version |

### UI — Entity Type List Description Column

#### Browser Tests

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-23.17 | Description column visible in entity type list | Browser | Column header present |
| T-23.18 | Description text shown for entity types | Browser | Description value in cell |

### UI — Entity Type Detail Editable Description (TD-46)

#### Browser Tests

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-23.19 | Description shown in entity type detail overview | Browser | Current version description visible |
| T-23.20 | Edit description button visible for Admin | Browser | Edit button present |
| T-23.21 | Edit description hidden for RO | Browser | No edit button |
| T-23.22 | Editing description calls PUT API | Browser | API called with new description |
| T-23.23 | Description updates after successful edit | Browser | New description visible |

### UI — Enum Description

#### Browser Tests

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-23.24 | Description column visible in enum list | Browser | Column header present |
| T-23.25 | Create enum modal has description field | Browser | Description input visible |
| T-23.26 | Creating enum with description shows it in list | Browser | Description in table cell |
| T-23.27 | Enum detail page shows description | Browser | Description in overview |

### UI — Catalog Version Description

#### Browser Tests

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-23.28 | Description column visible in CV list | Browser | Column header present |
| T-23.29 | Create CV modal has description field | Browser | Description input visible |
| T-23.30 | Creating CV with description shows it in list | Browser | Description in table cell |
| T-23.31 | CV detail page shows description in overview | Browser | Description in overview section |

### Regression

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-23.32 | All existing entity type tests pass | Browser | No regressions |
| T-23.33 | All existing enum tests pass | Browser | No regressions |
| T-23.34 | All existing CV tests pass | Browser | No regressions |
| T-23.35 | All existing backend tests pass | Unit/API | No regressions |

---

## 24. Catalog Version Metadata Edit (US-49, TD-61)

Update a catalog version's version label and/or description after creation.

### Backend — Service

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-24.01 | UpdateCatalogVersion updates description when provided | Unit | Description changed, label unchanged |
| T-24.02 | UpdateCatalogVersion updates version_label when provided | Unit | Label changed, description unchanged |
| T-24.03 | UpdateCatalogVersion with both fields updates both | Unit | Both fields changed |
| T-24.04 | UpdateCatalogVersion with nil fields preserves existing values | Unit | No change to either field |
| T-24.05 | UpdateCatalogVersion with duplicate label returns 409 | Unit | ConflictError |
| T-24.06 | UpdateCatalogVersion with nonexistent CV returns 404 | Unit | NotFoundError |

### Backend — Integration (Repository)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-24.07 | CV label update persists in real SQLite | Integration | Updated label retrievable |
| T-24.08 | CV description update persists in real SQLite | Integration | Updated description retrievable |
| T-24.09 | Duplicate label rejected by DB unique constraint | Integration | Error returned |

### Backend — Handler

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-24.10 | PUT /catalog-versions/:id with {"description":"new"} returns 200 | API | Updated CV in response |
| T-24.11 | PUT /catalog-versions/:id with {"version_label":"v2.1"} returns 200 | API | Renamed CV in response |
| T-24.12 | PUT /catalog-versions/:id with {} preserves all fields | API | 200, unchanged values |
| T-24.13 | PUT /catalog-versions/:id with duplicate label returns 409 | API | ConflictError |
| T-24.14 | PUT /catalog-versions/:id nonexistent returns 404 | API | NotFoundError |
| T-24.15 | PUT /catalog-versions/:id as RO returns 403 | API | Forbidden |
| T-24.16 | PUT /catalog-versions/:id as RW returns 200 | API | Success |
| T-24.17 | PUT /catalog-versions/:id bind error returns 400 | API | BadRequest |

### Frontend — UI

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-24.18 | CV detail page shows Edit button next to description | Browser | Button visible for RW+ |
| T-24.19 | Click Edit → TextInput appears with current value | Browser | Inline edit mode |
| T-24.20 | Type new value → Save → API PUT called | Browser | API call with description |
| T-24.21 | Cancel restores original value | Browser | No API call, original shown |
| T-24.22 | CV detail page shows Edit button next to version label | Browser | Button visible for RW+ |
| T-24.23 | Label edit triggers PUT with version_label | Browser | API call with version_label |
| T-24.24 | RO user sees no Edit buttons | Browser | Buttons hidden |
| T-24.25 | client.ts catalogVersions.update calls PUT /catalog-versions/:id | Browser | Correct URL and method |

### Frontend — API Client

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-24.26 | catalogVersions.update(id, {description}) sends PUT | Browser | PUT with JSON body |
| T-24.27 | catalogVersions.update(id, {version_label}) sends PUT | Browser | PUT with JSON body |

### Regression

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-24.28 | All existing CV tests pass | Unit/API/Browser | No regressions |

---

## 25. Catalog Metadata Edit (US-50, FF-10)

Update a catalog's name and/or description after creation. Published catalogs restrict editing.

### Backend — Service

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-25.01 | UpdateMetadata updates description | Unit | Description changed |
| T-25.02 | UpdateMetadata updates name with DNS-label validation | Unit | Name changed |
| T-25.03 | UpdateMetadata with invalid DNS-label name returns 400 | Unit | ValidationError |
| T-25.04 | UpdateMetadata with duplicate name returns 409 | Unit | ConflictError |
| T-25.05 | UpdateMetadata with nil fields preserves existing values | Unit | No change |
| T-25.06 | UpdateMetadata resets validation status to draft | Unit | Status = draft |
| T-25.07 | UpdateMetadata on published catalog: SuperAdmin can edit description | Unit | Success |
| T-25.08 | UpdateMetadata on published catalog: rename blocked returns 400 | Unit | "cannot rename published catalog" |
| T-25.09 | UpdateMetadata on published catalog: non-SuperAdmin returns 403 | Unit | ForbiddenError |
| T-25.10 | UpdateMetadata on nonexistent catalog returns 404 | Unit | NotFoundError |
| T-25.11 | UpdateMetadata on published catalog calls SyncCR | Unit | SyncCR invoked |

### Backend — Integration (Repository)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-25.12 | Catalog name update persists, old name no longer resolves | Integration | GetByName(old) = NotFound |
| T-25.13 | Catalog description update persists | Integration | Updated description retrievable |
| T-25.14 | Duplicate catalog name rejected by DB unique constraint | Integration | Error returned |
| T-25.15 | Validation status reset to draft after metadata change | Integration | Status = draft |
| T-25.16 | Rename preserves instance catalog_id FK references | Integration | Instances still belong to catalog |

### Backend — Handler

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-25.17 | PUT /catalogs/{name} with {"description":"new"} returns 200 | API | Updated catalog |
| T-25.18 | PUT /catalogs/{name} with {"name":"new-name"} returns 200 | API | Renamed catalog |
| T-25.19 | PUT /catalogs/{name} with {} preserves all fields | API | 200, unchanged |
| T-25.20 | PUT /catalogs/{name} invalid DNS-label returns 400 | API | BadRequest |
| T-25.21 | PUT /catalogs/{name} duplicate name returns 409 | API | ConflictError |
| T-25.22 | PUT /catalogs/{name} nonexistent returns 404 | API | NotFoundError |
| T-25.23 | PUT /catalogs/{name} as RO returns 403 | API | Forbidden |
| T-25.24 | PUT /catalogs/{name} RequireWriteAccess middleware applied | API | Published catalog: 403 for RW |
| T-25.25 | PUT /catalogs/{name} RequireCatalogAccess middleware applied | API | Access check invoked |
| T-25.26 | PUT /catalogs/{name} published catalog: SuperAdmin desc edit → 200 | API | Success |
| T-25.27 | PUT /catalogs/{name} published catalog: SuperAdmin rename → 400 | API | BadRequest |
| T-25.28 | PUT /catalogs/{name} bind error returns 400 | API | BadRequest |

### Frontend — UI

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-25.29 | Catalog detail page shows Edit button for description | Browser | Button visible for RW+ |
| T-25.30 | Description edit: click Edit → input → Save → API PUT called | Browser | API call with description |
| T-25.31 | Description edit: Cancel restores original | Browser | No API call |
| T-25.32 | RO user sees no Edit button | Browser | Button hidden |
| T-25.33 | Published catalog: Edit disabled for non-SuperAdmin | Browser | Button disabled with tooltip |

### Frontend — API Client

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-25.34 | catalogs.update(name, {description}) sends PUT | Browser | PUT to /catalogs/{name} |
| T-25.35 | catalogs.update(name, {name}) sends PUT | Browser | PUT with new name |

### Live System

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-25.36 | Update catalog description via API, verify persisted | Live | GET returns updated description |
| T-25.37 | Rename catalog via API, verify old name 404 | Live | Old name returns 404 |
| T-25.38 | Rename published catalog via API returns 400 | Live | Cannot rename published |

### Regression

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-25.39 | All existing catalog tests pass | Unit/API/Browser | No regressions |

---

## 26. Catalog Re-pinning (US-51, TD-12)

Change a catalog's pinned CV via PUT /catalogs/{name}.

### Backend — Service

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-26.01 | Re-pin updates catalog_version_id | Unit | New CV ID set |
| T-26.02 | Re-pin to nonexistent CV returns 404 | Unit | NotFoundError |
| T-26.03 | Re-pin resets validation status to draft | Unit | Status = draft |
| T-26.04 | Re-pin on published catalog returns 400 | Unit | "unpublish first" |
| T-26.05 | Re-pin on unpublished catalog succeeds | Unit | Success |

### Backend — Integration (Repository)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-26.06 | catalog_version_id FK update persists | Integration | New CV ID retrievable |
| T-26.07 | Re-pin to nonexistent CV rejected by FK constraint | Integration | Error returned |
| T-26.08 | Validation status reset to draft after re-pin | Integration | Status = draft |
| T-26.09 | Instances remain associated with catalog after re-pin | Integration | Instance catalog_id unchanged |

### Backend — Handler

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-26.10 | PUT /catalogs/{name} with {"catalog_version_id":"new-cv"} returns 200 | API | Updated catalog with new CV |
| T-26.11 | PUT /catalogs/{name} with nonexistent CV returns 404 | API | NotFoundError |
| T-26.12 | PUT /catalogs/{name} re-pin on published catalog returns 400 | API | BadRequest |
| T-26.13 | PUT /catalogs/{name} re-pin as RO returns 403 | API | Forbidden |

### Frontend — UI

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-26.14 | Catalog detail page shows CV selector dropdown | Browser | Dropdown with CVs listed |
| T-26.15 | Selecting new CV triggers PUT with catalog_version_id | Browser | API call with new CV ID |
| T-26.16 | CV dropdown disabled on published catalogs | Browser | Dropdown disabled |
| T-26.17 | RO user sees no CV dropdown | Browser | Dropdown hidden |

### Regression

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-26.18 | All existing catalog tests pass | Unit/API/Browser | No regressions |

---

## 27. Catalog Version Pin Editing (US-52, FF-4)

Add or remove entity type pins from a catalog version.

### Backend — Service

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-27.01 | AddPin with valid ETV ID creates pin | Unit | Pin created, linked to CV |
| T-27.02 | AddPin with nonexistent ETV returns 404 | Unit | NotFoundError |
| T-27.03 | AddPin with duplicate ETV (already pinned) returns 409 | Unit | ConflictError |
| T-27.04 | RemovePin with valid pin ID removes pin | Unit | Pin deleted |
| T-27.05 | RemovePin with nonexistent pin returns 404 | Unit | NotFoundError |

### Backend — Integration (Repository)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-27.06 | Pin creation persists with correct CV ID and ETV ID FKs | Integration | Pin retrievable via ListByCatalogVersion |
| T-27.07 | Duplicate pin (same ETV on same CV) rejected by DB constraint | Integration | Error returned |
| T-27.08 | Pin deletion removes the row | Integration | ListByCatalogVersion excludes deleted pin |
| T-27.09 | Pin with nonexistent ETV ID rejected by FK constraint | Integration | Error returned |

### Backend — Handler

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-27.10 | POST /catalog-versions/:id/pins with {"entity_type_version_id":"etv"} returns 201 | API | Pin in response |
| T-27.11 | POST /catalog-versions/:id/pins nonexistent ETV returns 404 | API | NotFoundError |
| T-27.12 | POST /catalog-versions/:id/pins duplicate returns 409 | API | ConflictError |
| T-27.13 | POST /catalog-versions/:id/pins as RO returns 403 | API | Forbidden |
| T-27.14 | DELETE /catalog-versions/:id/pins/:pin-id returns 204 | API | No content |
| T-27.15 | DELETE /catalog-versions/:id/pins/:pin-id nonexistent returns 404 | API | NotFoundError |
| T-27.16 | DELETE /catalog-versions/:id/pins/:pin-id as RO returns 403 | API | Forbidden |
| T-27.17 | POST /catalog-versions/:id/pins bind error returns 400 | API | BadRequest |

### Frontend — UI

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-27.18 | BOM tab shows Add Pin button for RW+ users | Browser | Button visible |
| T-27.19 | Add Pin opens picker with available entity type versions | Browser | Picker/modal with ETV list |
| T-27.20 | Selecting ETV and confirming triggers POST /pins | Browser | API call, pin added to list |
| T-27.21 | Remove button visible per pin row for RW+ users | Browser | Button on each row |
| T-27.22 | Click Remove triggers DELETE /pins/:pin-id | Browser | API call, pin removed from list |
| T-27.23 | RO user sees no Add/Remove controls | Browser | Controls hidden |

### Frontend — API Client

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-27.24 | catalogVersions.addPin(id, etvId) sends POST | Browser | POST to /catalog-versions/:id/pins |
| T-27.25 | catalogVersions.removePin(id, pinId) sends DELETE | Browser | DELETE to /catalog-versions/:id/pins/:pin-id |

### Regression

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-27.26 | All existing CV tests pass | Unit/API/Browser | No regressions |
| T-27.27 | All existing catalog tests pass (pin changes don't break catalogs) | Unit/API | No regressions |

---

## 28. CV Pin Management — Unique Entity Type + Version Change (US-53)

Prevents duplicate entity type pins, adds inline version change in BOM table, and filters Add Pin modal to unpinned entities.

### Backend — AddPin Duplicate Entity Type Check

#### Unit Tests (service)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-28.01 | AddPin with same entity type (different version) returns 409 | Unit | 409 conflict with entity type name in message |
| T-28.02 | AddPin with different entity type succeeds | Unit | Pin created successfully |
| T-28.03 | AddPin with exact same ETV returns 409 (existing behavior) | Unit | 409 conflict |

### Backend — UpdatePin (Change Version)

#### Unit Tests (service)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-28.04 | UpdatePin changes ETV on existing pin | Unit | Pin updated, new ETV set |
| T-28.05 | UpdatePin with ETV from different entity type returns 400 | Unit | 400 entity type mismatch |
| T-28.06 | UpdatePin with nonexistent pin returns 404 | Unit | 404 not found |
| T-28.07 | UpdatePin with nonexistent ETV returns 404 | Unit | 404 not found |
| T-28.08 | UpdatePin verifies pin belongs to specified CV | Unit | 404 if pin belongs to different CV |

#### API Tests (handler)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-28.09 | PUT /catalog-versions/:id/pins/:pin-id returns 200 | API | Updated pin in response |
| T-28.10 | PUT with entity type mismatch returns 400 | API | 400 error message |
| T-28.11 | PUT with nonexistent pin returns 404 | API | 404 |
| T-28.12 | PUT as RO returns 403 | API | 403 forbidden |

#### Integration Tests (repository)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-28.13 | Pin update persists new ETV ID | Integration | DB reflects new ETV |

### UI — BOM Tab Inline Version Dropdown

#### Browser Tests

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-28.14 | Version column is dropdown for Admin+ | Browser | Select/dropdown visible per pin row |
| T-28.15 | Version dropdown lists all versions of entity type | Browser | Versions from API shown |
| T-28.16 | Selecting different version calls updatePin API | Browser | PUT called with new ETV ID |
| T-28.17 | Version updates in table after change | Browser | New version shown |
| T-28.18 | RO user sees plain text, not dropdown | Browser | No dropdown controls |

### UI — Add Pin Modal Entity Type Filtering

#### Browser Tests

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-28.19 | Add Pin modal only shows unpinned entity types | Browser | Already-pinned types filtered out |
| T-28.20 | After removing pin, entity type reappears in Add Pin | Browser | Removed type available again |

### API Client Tests

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-28.21 | catalogVersions.updatePin(id, pinId, etvId) sends PUT | Browser | PUT to correct URL with body |

### Regression

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-28.22 | All existing CV pin tests pass | Browser/Unit/API | No regressions |
| T-28.23 | Diagram renders correctly after pin version change | Browser | Diagram shows updated entity types |

## 29. Pin Editing Stage Guards (TD-69)

Pin editing (AddPin, RemovePin, UpdatePin) is restricted by catalog version lifecycle stage:
- **development**: RW+ allowed (standard permission)
- **testing**: SuperAdmin only (this policy is provisional and may be relaxed to Admin+ in the future)
- **production**: blocked entirely, regardless of role

### Backend — Service Stage Guards

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-29.01 | AddPin on production CV as Admin → rejected | Unit | Validation error mentioning "production" |
| T-29.02 | AddPin on production CV as SuperAdmin → rejected | Unit | Validation error mentioning "production" |
| T-29.03 | AddPin on testing CV as RW → rejected | Unit | Validation error mentioning "SuperAdmin" |
| T-29.04 | AddPin on testing CV as Admin → rejected | Unit | Validation error mentioning "SuperAdmin" |
| T-29.05 | AddPin on testing CV as SuperAdmin → allowed | Unit | Pin created successfully |
| T-29.06 | RemovePin on production CV as Admin → rejected | Unit | Validation error mentioning "production" |
| T-29.07 | UpdatePin on testing CV as RW → rejected | Unit | Validation error mentioning "SuperAdmin" |

### API Handler Tests

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-29.08 | AddPin as RW on development CV → allowed | API | 201 created |
| T-29.09 | AddPin as SuperAdmin on testing CV → allowed | API | 201 created |
| T-29.10 | AddPin as Admin on production CV → blocked | API | 400 with "production" message |

### Live System Tests (`scripts/test-descriptions.sh`)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-29.11 | AddPin on production CV returns 400 | Live | 400 (Admin role) |
| T-29.12 | AddPin on production CV blocked for SuperAdmin | Live | 400 |
| T-29.13 | AddPin on testing CV returns 400 for RW | Live | 400 |
| T-29.14 | AddPin on testing CV allowed for SuperAdmin | Live | 201 with pin_id |

### UI — Pin Control Visibility

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-29.15 | BOM tab hides Add Pin / Remove / version dropdown for production CV | Browser | No pin editing controls visible |
| T-29.16 | BOM tab hides pin controls for testing CV as Admin | Browser | No pin editing controls visible |
| T-29.17 | BOM tab shows pin controls for testing CV as SuperAdmin | Browser | Add Pin, Remove, version dropdown visible |

---

## 30. Security Fixes — Validate Write Protection & CV Metadata Stage Guards (TD-78, TD-71)

Two authorization fixes: (1) `POST /catalogs/{name}/validate` on published catalogs now requires SuperAdmin (closes write protection bypass where RW users could mutate `validation_status`); (2) `UpdateCatalogVersion` now enforces the same lifecycle stage guards as pin editing — blocked on production, SuperAdmin-only on testing, RW+ on development.

### Backend — Validate Write Protection (TD-78)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-30.01 | Validate on published catalog as RW → blocked by writeMiddleware | API | 403 Forbidden |
| T-30.02 | Validate on published catalog as Admin → blocked by writeMiddleware | API | 403 Forbidden |
| T-30.03 | Validate on published catalog as SuperAdmin → allowed | API | 200 with validation results |
| T-30.04 | Validate on unpublished catalog as RW → allowed (no regression) | API | 200 with validation results |

### Backend — CV Metadata Stage Guard (TD-71)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-30.05 | UpdateCatalogVersion on production CV as SuperAdmin → rejected | Unit | Validation error mentioning "production" |
| T-30.06 | UpdateCatalogVersion on production CV as RW → rejected | Unit | Validation error mentioning "production" |
| T-30.07 | UpdateCatalogVersion on testing CV as RW → rejected | Unit | Validation error mentioning "SuperAdmin" |
| T-30.08 | UpdateCatalogVersion on testing CV as Admin → rejected | Unit | Validation error mentioning "SuperAdmin" |
| T-30.09 | UpdateCatalogVersion on testing CV as SuperAdmin → allowed | Unit | CV updated successfully |
| T-30.10 | UpdateCatalogVersion on development CV as RW → allowed (no regression) | Unit | CV updated successfully |

### API Handler — CV Metadata Stage Guard (TD-71)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-30.11 | PUT /catalog-versions/:id on production CV → 400 | API | 400 with "production" message |
| T-30.12 | PUT /catalog-versions/:id on testing CV as RW → 400 | API | 400 with stage guard message |
| T-30.13 | PUT /catalog-versions/:id on testing CV as SuperAdmin → 200 | API | 200 with updated CV |
| T-30.14 | PUT /catalog-versions/:id on development CV as RW → 200 (no regression) | API | 200 with updated CV |

### UI — Validate Button Visibility (TD-78)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-30.15 | Validate button hidden on published catalog for RW user | Browser | Button not rendered |
| T-30.16 | Validate button visible on published catalog for SuperAdmin | Browser | Button rendered and clickable |
| T-30.17 | Validate button visible on unpublished catalog for RW (no regression) | Browser | Button rendered |

### UI — CV Edit Controls Visibility (TD-71)

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-30.18 | Edit buttons hidden on production CV for all roles | Browser | No Edit buttons for label or description |
| T-30.19 | Edit buttons hidden on testing CV for RW | Browser | No Edit buttons |
| T-30.20 | Edit buttons visible on testing CV for SuperAdmin | Browser | Edit buttons rendered |
| T-30.21 | Edit buttons visible on development CV for RW (no regression) | Browser | Edit buttons rendered |

### Live System Tests

| ID | Test Case | Layer | Expected |
|----|-----------|-------|----------|
| T-30.22 | Validate on published catalog as RW → 403 | Live | 403 Forbidden |
| T-30.23 | UpdateCatalogVersion on production CV → 400 | Live | 400 with stage guard message |
| T-30.24 | UpdateCatalogVersion on testing CV as RW → 400 | Live | 400 with stage guard message |
| T-30.25 | UpdateCatalogVersion on testing CV as SuperAdmin → 200 | Live | 200 with updated CV |

---

## Coverage Criteria

### Pass Rate

- **Target**: 100% of all test cases must pass before any code is considered complete.
- Tests are run continuously during implementation. No code is merged or committed with failing tests.

### Code Coverage

- **Target**: Near-100% code coverage across all layers (backend, UI, operator).
- **Backend**: `go test -coverprofile=coverage.out ./...` with `go tool cover -func=coverage.out` for analysis.
- **UI**: Vitest with `--coverage` flag (v8 provider).
- Any uncovered lines must be individually documented with a justification explaining why the line cannot be reached in the current test environment.

### Known Untestable Code (Phase A)

The following code paths cannot be covered in Phase A (no container runtime) and are deferred to later environment phases:

| File/Package | Lines | Reason | Covered In |
|---|---|---|---|
| `cmd/api-server/main.go` | ~15-20 | Server bootstrap: `ListenAndServe`, signal handling, graceful shutdown | Phase B (kind) |
| `cmd/operator/main.go` | ~10-15 | Operator bootstrap: manager setup, leader election, signal handling | Phase B (kind) |
| `internal/api/middleware/rbac.go` (real SAR path) | ~5-10 | Real SubjectAccessReview HTTP call to k8s API server | Phase C (OCP) |
| `internal/infrastructure/gorm/` (PostgreSQL driver init) | ~5-10 | `//go:build postgres` driver initialization code — only SQLite driver exercised in Phase A | Phase B (PostgreSQL container) |
| `internal/infrastructure/config/` (env-based config) | ~5-10 | Production configuration loading from environment variables / ConfigMaps | Phase B (kind) |
| `internal/operator/controllers/` (real CRD apply) | ~5-10 | Dynamic CRD registration against a real k8s API (not envtest) | Phase B (kind) |

**Phase A estimated uncoverable**: ~50-70 lines total. All other code must be at 100%.

---

## Phase Exit Criteria

### Phase A Exit Criteria (First Human Checkpoint)

**Tests**:
- All 1173 test cases (T-1.01 through T-28.23; T-13.78 through T-13.85 retired) pass
- All tests run against SQLite (in-memory) and mocked/simulated infrastructure
- Operator envtest tests pass (envtest downloads and runs etcd/kube-apiserver binaries directly — no containers)
- RBAC tests pass with mocked SubjectAccessReview
- CatalogVersion CRD tests pass with fake K8s client

**Coverage**:
- 100% code coverage of all business logic, repository implementations, API handlers, UI components, and operator reconciliation logic
- Maximum ~50-70 lines uncovered, each individually documented with justification (see table above)
- Backend: `go test -coverprofile` shows ≥99% (the ~50 uncoverable lines out of an estimated codebase of ~5,000-10,000 lines)
- UI: Vitest `--coverage` shows 100% of components, hooks, pages, and context providers

**No containers required at any point.**

---

### Phase B Exit Criteria (Second Human Checkpoint)

**Tests**:
- All Phase A tests (T-1.01 through T-9.09) continue to pass
- All Phase B tests (T-B.01 through T-B.11) pass
- Integration tests pass against PostgreSQL (running in container) — confirms no SQLite-specific behavior assumptions

**Coverage**:
- All previously uncoverable `cmd/` entrypoint code now covered via E2E tests against running containers
- PostgreSQL driver initialization code (`//go:build postgres`) now covered
- Production configuration loading now covered
- Dynamic CRD registration against real k8s API (kind) now covered
- Backend coverage: ≥99.5%

**Remaining uncoverable (deferred to Phase C)**:

| File/Package | Lines | Reason | Covered In |
|---|---|---|---|
| `internal/api/middleware/rbac.go` (real OCP SAR) | ~5-10 | Real SubjectAccessReview against OCP identity provider (kind uses basic RBAC, not OCP OAuth) | Phase C |
| OLM integration code | ~10-15 | OLM bundle registration, ClusterServiceVersion handling — OLM not available in kind | Phase C |
| OpenShift Route / OAuth proxy setup | ~5-10 | OCP-specific networking and auth configuration | Phase C |

**Phase B estimated uncoverable**: ~20-35 lines.

---

### Phase C Exit Criteria (Final Acceptance)

**Tests**:
- All Phase A + Phase B tests continue to pass
- All Phase C tests (T-C.01 through T-C.11) pass on the remote OCP cluster
- Real OpenShift RBAC verified with actual OCP users/service accounts (not mocked)
- OLM installation and uninstallation verified
- Operator upgrade path verified

**Coverage**:
- All previously uncoverable OCP-specific code now covered
- Real SubjectAccessReview path covered with actual OCP identity provider
- OLM integration code covered
- OpenShift Route / OAuth setup covered
- **Target: 100% coverage** across all code
- Any remaining uncoverable lines (expected: 0-5, e.g., panic recovery catch-all) must be individually documented

---

### General Notes
- Test cases within each milestone are executed as part of that milestone's implementation step.
- All tests from previous milestones must continue to pass at every subsequent step.
- Coverage is measured after each milestone within Phase A. Any lines not covered must be justified.
- The final full test run (planning methodology step 3) runs all tests across all milestones.
