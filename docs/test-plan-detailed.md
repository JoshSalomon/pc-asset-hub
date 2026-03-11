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

**Milestones completed**: 1–9 plus CatalogVersion Discovery CRD (all code written and tested)

**Tests that must pass**: All test cases T-1.01 through T-9.09, T-CV.01 through T-CV.31, and T-E.01 through T-E.146 (331 test cases), using SQLite and mocked/simulated infrastructure.

**Human checkpoint**: After all 331 tests pass with 100% coverage (documented exceptions). This is the first review point.

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

### Catalog Version Management (US-34, US-35)

| ID | Test Case | Layer | Acceptance Criteria |
|----|-----------|-------|-------------------|
| T-7.33 | Create UI shows all types with version dropdowns, latest pre-selected | UI | US-34: version picker with defaults |
| T-7.34 | Summary/review step shows full bill of materials | UI | US-34: review before confirm |
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
- All 384 test cases (T-1.01 through T-9.09, T-CV.01 through T-CV.31, T-E.01 through T-E.146, T-10.01 through T-10.51, and T-11.01 through T-11.58) pass
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
