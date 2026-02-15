# AI Asset Hub — Phase A Implementation Prompt

You are implementing the AI Asset Hub backend, UI, and operator logic. This is Phase A: isolated development with no containers. All tests use SQLite in-memory and mocked infrastructure.

## Reference Documents (READ THESE)

- `docs/architecture.md` — Tech stack, layered architecture, project structure, database schema, API design
- `docs/test-plan-detailed.md` — All test cases (T-1.01 through T-9.09) with acceptance criteria, plus coverage targets per milestone
- `PRD.md` — Product requirements and 38 user stories with acceptance criteria
- `.claude/commands/plan.md` — Planning methodology and final steps

## How This Works

This prompt runs in a Ralph Loop. Each iteration, you:
1. Determine which step you're on (see below)
2. Work on that step
3. Verify the step's exit criteria
4. If the step passes, move to the next step in the same iteration if possible
5. Only output `<promise>PHASE A COMPLETE</promise>` when ALL steps are done

## Step Detection

Check the current state of the project to determine which step to work on:

- **Step 0 (Scaffolding)**: Not done if `go.mod` doesn't exist or `make test` fails or `cd ui && npm test` fails
- **Step 1 (Domain)**: Not done if `internal/domain/models/` has no Go files or `internal/domain/repository/` has no interface files
- **Step 2 (GORM Meta Repos)**: Not done if `internal/infrastructure/gorm/repository/` has no meta repo files or tests T-1.01 through T-1.39 don't all pass
- **Step 3 (GORM Data Repos)**: Not done if tests T-2.01 through T-2.22 don't all pass
- **Step 4 (Meta Services)**: Not done if `internal/service/meta/` has no Go files or tests T-3.01 through T-3.53 don't all pass
- **Step 5 (Operational Services)**: Not done if `internal/service/operational/` has no Go files or tests T-4.01 through T-4.31 don't all pass
- **Step 6 (Meta API)**: Not done if `internal/api/meta/` has no Go files or tests T-5.01 through T-5.51 don't all pass
- **Step 7 (Operational API)**: Not done if `internal/api/operational/` has no Go files or tests T-6.01 through T-6.24 don't all pass
- **Step 8 (Meta UI)**: Not done if `ui/src/pages/meta/` has no .tsx files or UI tests T-7.01 through T-7.54 don't all pass
- **Step 9 (Operational UI)**: Not done if `ui/src/pages/operational/` has no .tsx files or UI tests T-8.01 through T-8.10 don't all pass
- **Step 10 (Operator)**: Not done if `internal/operator/` has no Go files or tests T-9.01 through T-9.09 don't all pass
- **Step 11 (Final)**: Not done if `docs/security-report-phase-a.md` doesn't exist or `docs/commit-message-phase-a.txt` doesn't exist

Work on the FIRST incomplete step. Do not skip steps.

---

## Step 0: Project Scaffolding

Initialize the project:
1. Go module: `go mod init github.com/project-catalyst/pc-asset-hub`
2. Create the full directory structure from `docs/architecture.md` section 5
3. Makefile with targets: build, test, lint, coverage
4. golangci-lint config (`.golangci.yml`)
5. Initialize UI: `cd ui && npm create vite@latest . -- --template react-ts` (or equivalent)
6. Install: `@patternfly/react-core`, `@patternfly/react-table`, `@patternfly/react-topology`, `@patternfly/react-icons`
7. Set up ESLint + Prettier, Vitest + React Testing Library + MSW for UI
8. Basic PatternFly app shell (renders "AI Asset Hub")
9. One trivial Go test, one trivial UI test

**Exit criteria**: `make build` succeeds, `make test` passes, `make lint` clean, `cd ui && npm test` passes, `cd ui && npm run build` succeeds.

---

## Step 1: Domain Models and Repository Interfaces

Implement `internal/domain/` with ZERO external dependencies (standard library only).

**Domain models** (`internal/domain/models/`): EntityType, EntityTypeVersion, Attribute, Association, Enum, EnumValue, CatalogVersion, CatalogVersionPin, LifecycleTransition, EntityInstance, InstanceAttributeValue, AssociationLink. See `docs/architecture.md` section 7 for all fields.

**Domain errors** (`internal/domain/errors/`): ErrNotFound, ErrConflict, ErrValidation, ErrForbidden, ErrCycleDetected, ErrReferencedEnum.

**Repository interfaces** (`internal/domain/repository/`): One interface per entity — see `docs/architecture.md` section 4. All methods use domain models only. See Step 1 in the plan for the full list of interfaces and methods.

**Exit criteria**: All domain code compiles, no external imports in `internal/domain/`, `make test` passes, `make lint` clean.

---

## Step 2: GORM Models and Meta Repositories

See `docs/architecture.md` section 7 for the full database schema. Implement test cases T-1.01 through T-1.39 from `docs/test-plan-detailed.md`.

1. GORM-tagged model structs (`internal/infrastructure/gorm/models/`) with `ToModel()`/`FromModel()` conversion methods
2. Database init with SQLite driver, auto-migration
3. GORM repository implementations for ALL meta tables: entity types, versions, attributes, associations, enums, enum values, catalog versions, pins, lifecycle transitions
4. Integration tests for T-1.01 through T-1.39 using SQLite in-memory

**Exit criteria**: All 39 test cases pass, 100% coverage of `internal/infrastructure/gorm/`, `make test` passes.

---

## Step 3: GORM Data Repositories

Implement test cases T-2.01 through T-2.22.

1. GORM models for EntityInstance, InstanceAttributeValue, AssociationLink
2. Repository implementations for all data tables
3. Integration tests for T-2.01 through T-2.22

**Exit criteria**: T-2.01 through T-2.22 pass, all previous tests still pass (61 total), 100% coverage of all repos.

---

## Step 4: Meta Services

Implement test cases T-3.01 through T-3.53. Services import ONLY from `internal/domain/`.

1. **EntityTypeService**: CRUD + CopyEntityType. Update creates new version with copy-on-write (copies attributes AND associations)
2. **AttributeService**: Add/Update/Remove/Reorder/CopyFromType. Every mutation triggers version increment
3. **AssociationService**: Create/Delete/List. Containment cycle detection (DAG). Version increment on mutation
4. **EnumService**: CRUD + value management. Delete blocked when referenced
5. **CatalogVersionService**: Create/Promote/Demote. Lifecycle state machine: RW=dev↔test, Admin=test→prod, SuperAdmin=prod→test/dev
6. **VersionHistoryService**: GetHistory, CompareVersions (diff: added/removed/modified)

All services use constructor injection of repository interfaces. Unit tests mock repos with `testify/mock`.

**Exit criteria**: T-3.01 through T-3.53 pass, all previous tests pass (92 total), 100% coverage of `internal/service/`.

---

## Step 5: Operational Services

Implement test cases T-4.01 through T-4.31.

1. **EntityInstanceService**: CRUD scoped to catalog version, attribute validation, auto-versioning, optimistic locking (409 on stale version)
2. **ContainmentService**: Create/List/Get contained, cascade delete (multi-level, atomic), dangling reference notification
3. **ReferenceService**: Create/Delete/GetForward/GetReverse, filter by type, both directional and bidirectional
4. **QueryService**: Filter by any attribute (EAV join), sort, paginate, error on non-existent attribute

**Exit criteria**: T-4.01 through T-4.31 pass, all previous tests pass (123 total), 100% coverage.

---

## Step 6: Meta API Handlers

Implement test cases T-5.01 through T-5.51. See `docs/architecture.md` section 6 for URLs.

1. **RBAC middleware**: Mock provider reading role from `X-User-Role` header. Interface for real SubjectAccessReview later
2. **DTOs** with validation tags
3. **Echo router**: Meta API group at `/api/meta/v1/`
4. **Handlers**: Entity types (CRUD+copy), Attributes (CRUD+copy+reorder), Associations (CRUD), Enums (CRUD), Catalog versions (CRUD+promote+demote+transitions), Version history (list+compare)
5. **API tests** using httptest for all four roles

**Exit criteria**: T-5.01 through T-5.51 pass, all previous tests pass (143 total), 100% of `internal/api/` covered (except ~5 lines of real SAR, documented).

---

## Step 7: Operational API Handlers

Implement test cases T-6.01 through T-6.24.

1. **Echo router**: Operational group at `/api/catalog/{catalog-version}/`
2. **Dynamic routing**: entity type names from catalog version become URL segments
3. **Handlers**: Instance CRUD, containment sub-resources, multi-level traversal, reference traversal, filtering/sorting/pagination
4. **API tests**: catalog version scoping, 404 on invalid/demoted versions

**Exit criteria**: T-6.01 through T-6.24 pass, all previous tests pass (148 total), 100% of `internal/api/operational/`.

---

## Step 8: UI — Meta Operations

Implement test cases T-7.01 through T-7.54. See `PRD.md` section 6.1 for UI requirements.

1. **API client** (`ui/src/api/`): Typed fetch client for all Meta API endpoints. React Query provider
2. **Auth context**: AuthContext with mock role provider
3. **Router**: React Router for entity types, enums, catalog versions
4. **Pages**: EntityTypeList, EntityTypeDetail, AttributeManagement, AssociationManagement, AssociationMap (@patternfly/react-topology), EnumList, EnumDetail, CatalogVersionList, CatalogVersionCreate, CatalogVersionDetail, VersionHistory with diff
5. **Shared**: ConfirmationDialog, InlineValidation, ToastNotifications, RoleAwareControl
6. **Tests**: Vitest + React Testing Library + MSW

**Exit criteria**: T-7.01 through T-7.54 pass, all previous Go tests pass, 100% UI coverage.

---

## Step 9: UI — Operational Pages

Implement test cases T-8.01 through T-8.10.

1. **CatalogVersionContext**: Global selected catalog version
2. **Pages**: Instance list (dynamic columns), instance create (dynamic form), instance detail (values + children + references), containment breadcrumb
3. **Tests**: T-8.01 through T-8.10

**Exit criteria**: T-8.01 through T-8.10 pass, all previous tests pass, 100% UI coverage.

---

## Step 10: Operator Logic

Implement test cases T-9.01 through T-9.09. See `docs/architecture.md` section 11.

1. **Operator scaffolding** with operator-sdk
2. **AssetHub CRD**: Reconciler creates Deployment, Service, UI Deployment. Delete cleans up
3. **CRD/CR generation** (`internal/operator/crdgen/`): Generate valid K8s YAML from entity type definitions and instances
4. **Promotion reconciler**: Apply on promote, remove on demote, report status, never modify DB
5. **Tests**: envtest for T-9.01 through T-9.09

**Exit criteria**: T-9.01 through T-9.09 pass, all previous tests pass, 100% of `internal/operator/` (except `cmd/operator/main.go` ~10-15 lines, documented).

---

## Step 11: Security Scan, Cleanup, Final Verification

1. **Security scan**: Scan all Go and UI code for OWASP Top 10, injection, XSS, secrets exposure, auth bypass. Fix all issues. Run full test suite after fixes. Write report to `docs/security-report-phase-a.md`
2. **Code cleanup**: golangci-lint + ESLint/Prettier, eliminate duplication, add comments only where non-obvious
3. **Full test run**: `make test` (all Go), `cd ui && npm test` (all UI), coverage ≥99% backend (document every uncovered line), 100% UI
4. **Documentation**: Update docs/ if architecture changed during implementation
5. **LTM**: Store implementation learnings, remove stale memories
6. **Stage files**: List all files by category, list unstaged files with reasons. Write commit message to `docs/commit-message-phase-a.txt`
7. **DO NOT commit to git**

**Exit criteria**: Security report exists, all tests pass, coverage targets met, lint clean, commit message prepared.

---

## IMPORTANT RULES

- Follow the layered architecture strictly: `domain/` has NO external imports, `service/` imports only `domain/`, `api/` imports `service/`+`domain/`, `infrastructure/` imports `domain/`+GORM
- The `cmd/` package is the composition root — the only place that wires concrete implementations
- Every test from previous steps must continue to pass at every subsequent step
- Use UUID v7 for all IDs (`google/uuid`)
- Use optimistic locking on all updates (`WHERE id = ? AND version = ?`)
- Association versioning: associations belong to entity_type_versions, not entity_types (copy-on-write)

## COMPLETION

Only when ALL steps 0-11 are complete — all tests pass, coverage targets met, security report written, commit message prepared — output:

<promise>PHASE A COMPLETE</promise>
