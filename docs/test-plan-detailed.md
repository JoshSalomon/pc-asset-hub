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

**Milestones completed**: 1–9 (all code written and tested)

**Tests that must pass**: All test cases T-1.01 through T-9.09 (154 test cases), using SQLite and mocked/simulated infrastructure.

**Human checkpoint**: After all 154 tests pass with 100% coverage (documented exceptions). This is the first review point.

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

**Tests that must pass**: All Phase A tests (T-1.* through T-9.*) plus T-B.01 through T-B.11.

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

**Tests that must pass**: All Phase A + Phase B + Phase C tests.

**Human checkpoint**: After all tests pass on OCP. Final acceptance.

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

## Coverage Criteria Per Milestone (Phase A)

Coverage is measured after each milestone. The target is 100% of the code written in that milestone. Lines that cannot be covered must be individually justified.

### Milestone 1: Database Meta Tables
- **Target**: 100% of `internal/infrastructure/gorm/models/` (meta models) and `internal/infrastructure/gorm/repository/` (meta repositories)
- **Target**: 100% of `internal/domain/models/` and `internal/domain/repository/` (interfaces — covered by being implemented)
- **Untestable in Phase A**: None — SQLite in-memory fully exercises all repository code

### Milestone 2: Database Data Tables
- **Target**: 100% of `internal/infrastructure/gorm/repository/` (data repositories: instance, attribute values, association links)
- **Untestable in Phase A**: None

### Milestone 3: Service Layer — Meta
- **Target**: 100% of `internal/service/meta/`, `internal/service/versioning/`, `internal/service/validation/`
- **Untestable in Phase A**: None — all business logic tested with mocked repositories

### Milestone 4: Service Layer — Operational
- **Target**: 100% of `internal/service/operational/`
- **Untestable in Phase A**: None

### Milestone 5: API Layer — Meta API
- **Target**: 100% of `internal/api/meta/`, `internal/api/middleware/`, `internal/api/dto/`
- **Untestable in Phase A**:
  - `internal/api/middleware/rbac.go`: The real SubjectAccessReview call to the k8s API server (the code path that calls `k8s.io/client-go` authorizationv1 API). Covered via mock in Phase A; real path covered in Phase C.
  - Estimated: ~5-10 lines (the actual HTTP call to k8s API and response parsing for real RBAC). Documented per line.

### Milestone 6: API Layer — Operational API
- **Target**: 100% of `internal/api/operational/`
- **Untestable in Phase A**: None — httptest covers all handler code

### Milestone 7: UI — Meta Operations
- **Target**: 100% of `ui/src/pages/meta/`, `ui/src/components/`, `ui/src/hooks/`, `ui/src/context/`
- **Untestable in Phase A**: None — Vitest with JSDOM and MSW covers all React components and hooks

### Milestone 8: UI — Operational Pages
- **Target**: 100% of `ui/src/pages/operational/`
- **Untestable in Phase A**: None

### Milestone 9: Operator
- **Target**: 100% of `internal/operator/controllers/`, `internal/operator/crdgen/`
- **Untestable in Phase A**:
  - `cmd/operator/main.go`: Operator binary entrypoint, manager setup, signal handling (~10-15 lines)
  - `internal/operator/controllers/`: Any code that directly calls k8s API for applying CRDs to a real cluster (as opposed to envtest). envtest covers reconciliation logic but the actual `kubectl apply`-equivalent code path for dynamic CRD registration may differ. Estimated: ~5-10 lines.

### Cross-Milestone Untestable Code (Phase A)

The following code paths cannot be covered in Phase A and are deferred to later phases:

| File/Package | Lines | Reason | Covered In |
|---|---|---|---|
| `cmd/api-server/main.go` | ~15-20 | Server bootstrap: `ListenAndServe`, signal handling, graceful shutdown | Phase B (kind) |
| `cmd/operator/main.go` | ~10-15 | Operator bootstrap: manager setup, leader election, signal handling | Phase B (kind) |
| `internal/api/middleware/rbac.go` (real SAR path) | ~5-10 | Real SubjectAccessReview HTTP call to k8s API server | Phase C (OCP) |
| `internal/infrastructure/gorm/` (PostgreSQL driver init) | ~5-10 | `//go:build postgres` driver initialization code — only SQLite driver exercised in Phase A | Phase B (PostgreSQL container) |
| `internal/infrastructure/config/` (env-based config) | ~5-10 | Production configuration loading from environment variables / ConfigMaps | Phase B (kind) |
| `internal/operator/controllers/` (real CRD apply) | ~5-10 | Dynamic CRD registration against a real k8s API (not envtest) | Phase B (kind) |

**Phase A estimated uncoverable**: ~50-70 lines total across all milestones. All other code must be at 100%.

---

## Phase Exit Criteria

### Phase A Exit Criteria (First Human Checkpoint)

**Tests**:
- All 154 test cases (T-1.01 through T-9.09) pass
- All tests run against SQLite (in-memory) and mocked/simulated infrastructure
- Operator envtest tests pass (envtest downloads and runs etcd/kube-apiserver binaries directly — no containers)
- RBAC tests pass with mocked SubjectAccessReview

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
