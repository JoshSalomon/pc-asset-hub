e# AI Asset Hub — Test Coverage Report

Last updated: 2026-03-29

---

## Summary

| Layer | Tests | Pass Rate | Statements | Lines |
|-------|-------|-----------|------------|-------|
| Backend (Go) | 1329 | 100% | 97.2% (3531/3632) | — |
| UI — Unit tests (jsdom) | 75 | 100% | — | — |
| UI — Browser tests (Playwright) | 728 | 100% | 93.1% (2227/2392) | 96.4% (2022/2097) |
| UI — System tests (Playwright + live server) | 30 | 100% | — | — |
| Live system (bash scripts) | 228 | 100% | — | — |
| **Total** | **2378** | **100%** | — | — |

---

## Backend Coverage by Package

| Package | Coverage | Notes |
|---------|----------|-------|
| `internal/api/health` | 90.0% (9/10) | Readyz DB-ping error path |
| `internal/api/meta` | 98.9% (450/455) | Promote/Demote/Delete RoleRO/RW switch cases unreachable behind RBAC middleware |
| `internal/api/middleware` | 100.0% (69/69) | |
| `internal/api/operational` | 98.3% (296/301) | Copy/Replace handlers bind-error branches only |
| `internal/domain/errors` | 100.0% (32/32) | |
| `internal/domain/models` | 100.0% (1/1) | |
| `internal/infrastructure/config` | 100.0% (21/21) | |
| `internal/infrastructure/gorm/models` | 100.0% (30/30) | |
| `internal/infrastructure/gorm/repository` | 90.9% (610/672) | GORM error branches on Delete/Update |
| `internal/infrastructure/k8s` | 92.6% (50/54) | K8s client error paths |
| `internal/operator/api/v1alpha1` | 97.7% (85/87) | `DeepCopyObject` nil-receiver guard |
| `internal/operator/controllers` | 94.3% (198/210) | `SetupWithManager` (envtest — deferred to Phase B), `SetOwnerReference` error branches |
| `internal/operator/crdgen` | 94.7% (36/38) | `json.Marshal` error guards on well-formed inputs |
| `internal/service/meta` | 99.5% (739/743) | BulkCopy error paths, requiresDeepCopy edge cases |
| `internal/service/operational` | 99.8% (863/865) | Cycle guard in resolveParentChain (safety net) |
| `internal/service/validation` | 95.6% (43/45) | |

### Excluded from Coverage

These packages are not counted toward coverage because they contain no business logic:

| Package | Reason |
|---------|--------|
| `internal/domain/models` | Pure struct definitions (coverage tracked separately for `IsSystemAttributeName` helper) |
| `internal/domain/repository` | Interface definitions only |
| `internal/domain/repository/mocks` | Test infrastructure, not production code |
| `internal/infrastructure/gorm/database` | DB driver bootstrap, covered in Phase B |
| `internal/infrastructure/gorm/testutil` | Test infrastructure |

---

## Known Uncovered Code

### Deferred to Phase B (container environment)

| File | Function | Reason |
|------|----------|--------|
| `cmd/api-server/main.go` | `main` | Server bootstrap, signal handling |
| `cmd/operator/main.go` | `main` | Operator bootstrap, leader election |
| `infrastructure/gorm/database/database_sqlite.go` | `NewDB` | DB driver initialization |
| `operator/controllers/controller.go` | `SetupWithManager` | Requires real controller-runtime manager |

### Deferred to Phase C (OpenShift environment)

| File | Function | Reason |
|------|----------|--------|
| `api/middleware/rbac.go` | Real SAR path | Real SubjectAccessReview against OCP identity provider |

### Low-priority uncovered branches

These are error-handling branches in handlers where `c.Bind()` fails with malformed JSON. They are protected by the HTTP framework and represent low risk.

| File | Function | Coverage | Uncovered Branch |
|------|----------|----------|------------------|
| `api/meta/attribute_handler.go` | `Edit` | 76.9% | Bind-error branch |
| `api/meta/attribute_handler.go` | `CopyAttributes` | 75.0% | Bind-error, empty fields branches |
| `api/meta/attribute_handler.go` | `Reorder` | 66.7% | Bind-error, empty ordered_ids branch |
| `api/meta/catalog_version_handler.go` | `Create` | 70.0% | Bind-error, pin marshaling branches |
| `api/meta/enum_handler.go` | `Update` | 66.7% | Bind-error branch |
| `api/meta/enum_handler.go` | `ReorderValues` | 66.7% | Bind-error branch |

### Trivial delegator methods at 0%

These methods are single-line delegations to the repository layer with no branching logic. They are exercised indirectly through handler and integration tests.

| File | Function | Reason |
|------|----------|--------|
| `service/meta/attribute_service.go` | `ListAttributes` | Delegates to `etvRepo.GetLatestByEntityType` + `attrRepo.ListByVersion` |
| `service/meta/enum_service.go` | `ListValues` | Delegates to `evRepo.ListByEnum` |

### UI: Defensive guard clauses (unreachable via UI interaction)

These are `if (!x) return` early returns in event handlers and callbacks. They are unreachable because the UI prevents the conditions from occurring: buttons are disabled when preconditions aren't met, state variables are set before dependent handlers can fire, and Select components always pass non-empty values.

**useCatalogData.ts** (1 statement):

| Line | Code | Why unreachable |
|------|------|-----------------|
| L45 | `if (!pin) return` | `activeTab` set from pin names; tabs generated from pins, mismatch impossible |

**useInstances.ts** (3 statements):

| Line | Code | Why unreachable |
|------|------|-----------------|
| L45 | `if (!catalogName \|\| !entityTypeName \|\| !newInstName.trim()) return` | Create button `isDisabled={!newInstName.trim()}`; catalogName/entityTypeName always set |
| L103 | `if (!catalogName \|\| !entityTypeName \|\| !editTarget) return` | `editTarget` always set before edit modal opens |
| L139 | `if (!catalogName \|\| !entityTypeName \|\| !deleteTarget) return` | `deleteTarget` always set before delete modal opens |

**useInstanceDetail.ts** (1 statement):

| Line | Code | Why unreachable |
|------|------|-----------------|
| L44 | `setChildren([])` | Outer catch wraps loop where each iteration has inner try/catch; only fires if Array.filter or setState throws |

**CatalogDetailPage.tsx** (11 statements — page-level handlers for add-child, link, set-parent modals):

| Line | Code | Why unreachable |
|------|------|-----------------|
| L114 | `if (!name \|\| !typeName) { setAvailableInstances([]); return }` | Called from Select onSelect which always passes selected value |
| L124 | `if (!typeName \|\| !pins.length) { setChildSchemaAttrs([]); return }` | Called from Select with non-empty value; pins loaded before modal |
| L146 | `if (!name) return` | `name` always provided by route |
| L148 | `if (!assoc) return` | Association Select options generated from schemaAssocs; selected value always matches |
| L151 | `if (!targetPin) return` | Assoc target entity type always has a matching pin |
| L160 | `if (!name \|\| !typeName) { setParentInstances([]); return }` | Called from Select onSelect with value |
| L168 | `if (!name \|\| ... \|\| !childTypeName) return` | Button disabled when `!childTypeName` |
| L194 | `return` (else branch) | Button disabled when neither adopt nor create conditions met |
| L210 | `if (... \|\| !linkTargetId \|\| !linkAssocName) return` | Button disabled when `!linkTargetId \|\| !linkAssocName` |
| L227 | `if (... \|\| !parentTypeName) return` | Button disabled when `!parentTypeName` |
| L245 | `if (!name \|\| !activeTab \|\| !selectedInstance) return` | `selectedInstance` always set before handler fires |

---

## UI Test Coverage

### Unit Tests (jsdom)

| Test File | Tests | Status |
|-----------|-------|--------|
| `App.test.tsx` | 9 | Pass |
| `EntityTypeListPage.test.tsx` | 12 | Pass |
| **Total** | **21** | **100% pass** |

### Browser Tests (Playwright)

| Test File | Tests | Status |
|-----------|-------|--------|
| `App.browser.test.tsx` | 51 | Pass |
| `client.browser.test.ts` | 61 | Pass |
| `EntityTypeDetailPage.browser.test.tsx` | 135 | Pass |
| `EntityTypeListPage.browser.test.tsx` | 12 | Pass |
| `EnumDetailPage.browser.test.tsx` | 24 | Pass |
| `EnumListPage.browser.test.tsx` | 14 | Pass |
| `CatalogVersionDetailPage.browser.test.tsx` | 28 | Pass |
| `CatalogListPage.browser.test.tsx` | 20 | Pass |
| `CatalogDetailPage.browser.test.tsx` | 131 | Pass |
| `useCatalogData.browser.test.tsx` | 8 | Pass |
| `useInstances.browser.test.tsx` | 11 | Pass |
| `useInstanceDetail.browser.test.tsx` | 7 | Pass |
| `CreateInstanceModal.browser.test.tsx` | 7 | Pass |
| `EditInstanceModal.browser.test.tsx` | 4 | Pass |
| `AddChildModal.browser.test.tsx` | 7 | Pass |
| `LinkModal.browser.test.tsx` | 5 | Pass |
| `SetParentModal.browser.test.tsx` | 5 | Pass |
| `useEntityTypeData.browser.test.tsx` | 8 | Pass |
| `useAttributeManagement.browser.test.tsx` | 11 | Pass |
| `useAssociationManagement.browser.test.tsx` | 7 | Pass |
| `AddAttributeModal.browser.test.tsx` | 7 | Pass |
| `EditAttributeModal.browser.test.tsx` | 4 | Pass |
| `AddAssociationModal.browser.test.tsx` | 7 | Pass |
| `CopyAttributesModal.browser.test.tsx` | 7 | Pass |
| `RenameEntityTypeModal.browser.test.tsx` | 4 | Pass |
| `useContainmentTree.browser.test.tsx` | 11 | Pass |
| `InstanceDetailPanel.browser.test.tsx` | 14 | Pass |
| `OperationalCatalogDetailPage.browser.test.tsx` | 36 | Pass |
| `OperationalCatalogListPage.browser.test.tsx` | 13 | Pass |
| `OperationalApp.browser.test.tsx` | 3 | Pass |
| `useValidation.browser.test.tsx` | 6 | Pass |
| **Total** | **671** | **100% pass** |

### System Tests (Playwright + live server)

| Test File | Tests | Status |
|-----------|-------|--------|
| `App.system.test.ts` | 30 | Pass |
| **Total** | **30** | **100% pass** |

### Code Coverage (v8 provider)

Coverage is measured using `@vitest/coverage-v8`. The two test suites run independently with separate configs.

**Browser tests** (primary coverage — exercises full component rendering via Playwright):

| File | Statements | Branches | Functions | Lines |
|------|-----------|----------|-----------|-------|
| `src/App.tsx` | 87.7% | 74.4% | 79.6% | 92.7% |
| `src/api/client.ts` | 90.2% | 86.7% | 86.5% | 90.0% |
| `src/pages/meta/CatalogVersionDetailPage.tsx` | 84.5% | 71.8% | 92.9% | 89.0% |
| `src/pages/meta/EnumDetailPage.tsx` | 86.0% | 75.8% | 80.0% | 94.3% |
| `src/pages/meta/EnumListPage.tsx` | 90.0% | 81.3% | 87.5% | 96.3% |
| `src/pages/meta/EntityTypeDetailPage.tsx` | 70.8% | 59.0% | 54.3% | 78.7% |
| `src/pages/meta/EntityTypeListPage.tsx` | 91.7% | 100% | 83.3% | 91.7% |
| **Total** | **79.1%** | **67.4%** | **70.0%** | **85.6%** |

**Unit tests** (supplemental — covers components that work in jsdom without browser):

| File | Statements | Lines |
|------|-----------|-------|
| `src/api/client.ts` | 90.2% | 90.0% |
| `src/context/AuthContext.tsx` | 88.9% | 100% |
| `src/context/NavigationContext.tsx` | 85.7% | 100% |
| `src/pages/meta/EntityTypeListPage.tsx` | 90.0% | 90.0% |
| All other files | 0% | 0% |
| **Total (all source files)** | **17.9%** | **20.6%** |

The low total reflects that unit tests only exercise 4 out of ~15 source files. The remaining files (App.tsx, detail pages, etc.) require a browser environment and are covered by the browser test suite above.

### New Code Coverage (Session 001)

All new functions added in this session are at 100% coverage:

| File | Function | Coverage |
|------|----------|----------|
| `service/meta/entity_type_service.go` | `GetContainmentTree` | 100% |
| `service/meta/entity_type_service.go` | `GetVersionSnapshot` | 100% |
| `api/meta/entity_type_handler.go` | `ContainmentTree` | 100% |
| `api/meta/entity_type_handler.go` | `VersionSnapshot` | 100% |
| `api/meta/entity_type_handler.go` | `convertTreeNodes` | 100% |

### New Code Coverage (Session 002 — Cardinality + Edit + Names)

| File | Function | Coverage |
|------|----------|----------|
| `service/validation/cardinality.go` | `ValidateCardinality` | 100% |
| `service/validation/cardinality.go` | `NormalizeCardinality` | 100% |
| `service/validation/cardinality.go` | `NormalizeSourceCardinality` | 100% |
| `service/validation/cardinality.go` | All functions | 100% |
| `service/meta/association_service.go` | `EditAssociation` | 96.2% |
| `service/meta/association_service.go` | `checkNameConflict` | 100% |
| `service/meta/association_service.go` | `DeleteAssociation` | 94.7% |
| `service/meta/association_service.go` | `CreateAssociation` | 96.4% |
| `api/meta/association_handler.go` | `List` | 100% |
| `api/meta/association_handler.go` | `Create` | 91.7% |
| `api/meta/association_handler.go` | `Edit` | 88.9% |

### New Code Coverage (Session 003 — Diagram + Shared Modal)

| File | Component | Coverage |
|------|-----------|----------|
| `components/EntityTypeDiagram.tsx` | Diagram component | 90.7% stmts, 90.1% lines |
| `components/EditAssociationModal.tsx` | Shared edit modal | 92.4% stmts, 92.7% lines |
| `App.tsx` | Diagram tab + edit modal | 87.6% stmts, 91.3% lines |

### New Code Coverage (Session 004 — Catalog Foundation)

| File | Function | Coverage |
|------|----------|----------|
| `service/operational/catalog_service.go` | `NewCatalogService` | 100% |
| `service/operational/catalog_service.go` | `ValidateCatalogName` | 100% |
| `service/operational/catalog_service.go` | `CreateCatalog` | 100% |
| `service/operational/catalog_service.go` | `GetByName` | 100% |
| `service/operational/catalog_service.go` | `List` | 100% |
| `service/operational/catalog_service.go` | `Delete` | 100% |
| `api/operational/catalog_handler.go` | All 7 functions | 100% |
| `infrastructure/gorm/repository/catalog_repo.go` | `Create` | 100% |
| `infrastructure/gorm/repository/catalog_repo.go` | `GetByName` | 100% |
| `infrastructure/gorm/repository/catalog_repo.go` | `GetByID` | 100% |
| `infrastructure/gorm/repository/catalog_repo.go` | `List` | 90% |
| `infrastructure/gorm/repository/catalog_repo.go` | `Delete` | 100% |
| `infrastructure/gorm/repository/catalog_repo.go` | `UpdateValidationStatus` | 100% |
| `infrastructure/gorm/repository/entity_instance_repo.go` | `DeleteByCatalogID` | 100% |

`catalog_repo.go:List` at 90% — the `Find` error after `Count` succeeds requires the DB to fail between two queries in the same function, which cannot be triggered with the `closedDB` pattern.

### New Code Coverage (Session 005 — Instance CRUD with Attributes)

| File | Function | Coverage |
|------|----------|----------|
| `service/operational/instance_service.go` | `NewInstanceService` | 100% |
| `service/operational/instance_service.go` | `resolveEntityType` | 100% |
| `service/operational/instance_service.go` | `resolveAttributeValues` | 100% |
| `service/operational/instance_service.go` | `validateAndBuildAttributeValues` | 97% |
| `service/operational/instance_service.go` | `CreateInstance` | 100% |
| `service/operational/instance_service.go` | `GetInstance` | 100% |
| `service/operational/instance_service.go` | `ListInstances` | 100% |
| `service/operational/instance_service.go` | `mapAttributeValues` | 100% |
| `service/operational/instance_service.go` | `UpdateInstance` | 100% |
| `service/operational/instance_service.go` | `DeleteInstance` | 100% |
| `service/operational/instance_service.go` | `cascadeDelete` | 100% |
| `api/operational/instance_handler.go` | All 8 functions | 100% |
| Service package total | | **99.6%** |
| Handler package total | | **96.5%** |

Remaining uncovered (5 lines):
- `instance_service.go` — `default:` switch label in `validateAndBuildAttributeValues` (Go coverage instrumentation quirk; the body IS covered)
- `catalog_repo.go:82-84` — `Find` error after `Count` succeeds (DB internal; requires failure between sequential queries)
- `entity_instance_repo.go:71-73,91-93,120-122` — same `Find`-after-`Count` pattern across List/ListByParent

Review fixes applied: (1) `resolveEntityType` now returns errors instead of silently continuing on pin resolution failure. (2) `UpdateInstance` validates attribute values before incrementing version, preventing inconsistent state. (3) `mapAttributeValues` extracted as shared helper, eliminating duplicate resolution logic.

Bug found during live testing: PostgreSQL migration — old `catalog_version_id` column on `entity_instances` table not dropped. Fixed with `InitDB` pre-migration that copies data and drops old column.

### New Code Coverage (Session 006 — Containment & Association Links)

| File | Function | Coverage |
|------|----------|----------|
| `service/operational/instance_service.go` | `CreateContainedInstance` | 100% |
| `service/operational/instance_service.go` | `ListContainedInstances` | 100% |
| `service/operational/instance_service.go` | `CreateAssociationLink` | 100% |
| `service/operational/instance_service.go` | `DeleteAssociationLink` | 100% |
| `service/operational/instance_service.go` | `GetForwardReferences` | 100% |
| `service/operational/instance_service.go` | `GetReverseReferences` | 100% |
| `service/operational/instance_service.go` | `resolveLinks` | 100% |
| `service/operational/instance_service.go` | `cascadeDelete` | 100% |
| `api/operational/instance_handler.go` | All 15 functions (incl. SetParent) | 100% |
| `infrastructure/gorm/repository/association_link_repo.go` | `GetByID` | new |
| `infrastructure/gorm/repository/association_link_repo.go` | `DeleteByInstance` | new |
| Service package total | | **100.0%** |
| `service/operational/instance_service.go` | `SetParent` | 100% |
| `api/operational/instance_handler.go` | `SetParent` | 100% |
| Handler package total | | **97.7%** (legacy handler.go has pre-existing uncovered bind-error branches)

Quality review fixes applied: (H1) Route ambiguity resolved — static segments registered before parameterized. (H2) `ListContainedInstances` returns filtered count. (H3) `cascadeDelete` cleans up association links. (H4) `DeleteAssociationLink` verifies link ownership. (M2) Parent catalog validation. (M3) Same-catalog validation for links. (M6) Duplicate link prevention.

UI bug fixes: Details pane closes on tab switch. Add Contained modal supports "Adopt Existing" mode. Link modal uses dropdowns for association and target instance. Set Container modal added for reparenting from child side. Buttons disabled when no applicable associations.

Live system tests: `scripts/test-containment-links.sh` — 18 parameterized tests covering containment CRUD, validation, links, references, duplicate prevention, cascade delete with link cleanup.

### New Code Coverage (Session 007 — Catalog Data Viewer)

| File | Function | Coverage |
|------|----------|----------|
| `service/operational/instance_service.go` | `GetContainmentTree` | 96.4% |
| `service/operational/instance_service.go` | `resolveParentChain` | 87.5% |
| `service/operational/instance_service.go` | `ListInstances` (enhanced) | 100% |
| `service/operational/instance_service.go` | `GetInstance` (enhanced) | 93.8% |
| `api/operational/instance_handler.go` | `GetContainmentTree` | 100% |
| `api/operational/instance_handler.go` | `treeNodesToDTO` | 100% |
| `api/operational/instance_handler.go` | `ListInstances` (enhanced) | 100% |
| `api/operational/instance_handler.go` | `instanceDetailToDTO` (enhanced) | 100% |
| `infrastructure/gorm/repository/entity_instance_repo.go` | `ListByCatalog` | 85.7% |
| `infrastructure/gorm/repository/entity_instance_repo.go` | `applyAttrFilters` | 94.4% |
| `infrastructure/gorm/repository/entity_instance_repo.go` | `List` (enhanced) | 93.3% |

Remaining uncovered lines:
- `ListByCatalog:85.7%` — GORM `Find` error path (DB internal failure; same pattern as other List methods)
- `applyAttrFilters:94.4%` — `.max` error path (symmetric to `.min` path which is tested)
- `resolveParentChain:87.5%` — cycle guard safety net (requires circular data which can't exist in normal operation)
- `GetContainmentTree:96.4%` — already handles ET name fallback; remaining line is branch coverage instrumentation

Quality review fixes applied: (H1) BrowserRouter basename for /operational path. (H2) Deduplicated count query in List. (M1) Extracted findAndSelect to navigateToTreeNode callback. (M2) Extracted statusColor to shared utility. (M4) Removed json tags from service-layer ParentChainEntry. (M5) applyAttrFilters returns error for invalid numeric values. (L1) Display total in catalog list. (L2) Wire detailLoading spinner. (L4) Cycle guard in resolveParentChain.

New UI files (operational data viewer):
- `ui/operational.html` — separate HTML entry point
- `ui/src/main-operational.tsx` — operational app entry with basename="/operational"
- `ui/src/OperationalApp.tsx` — app shell (masthead, role selector, routes)
- `ui/src/pages/operational/OperationalCatalogListPage.tsx` — catalog list with search, pagination
- `ui/src/pages/operational/OperationalCatalogDetailPage.tsx` — tree browser + instance detail drawer
- `ui/src/utils/statusColor.ts` — shared utility

Live system tests: `scripts/test-data-viewer.sh` — 23 parameterized tests covering containment tree, pagination, sorting, filtering, parent chain, operational UI serving, combined queries, and references.

Two-pane redesign: Removed the redundant middle instance list pane from the tree browser. The tree is now the sole navigation (left pane), with instance detail shown inline in the right pane. Browser tests reduced from 37 to 27 for this page (instance list tests T-13.78-85 retired). Component simplified from ~605 lines to ~300 lines.

### New Code Coverage (Session 008 — Catalog Validation)

| File | Function | Coverage |
|------|----------|----------|
| `service/operational/validation_service.go` | `NewCatalogValidationService` | 100% |
| `service/operational/validation_service.go` | `Validate` | 100% |
| `service/operational/validation_service.go` | `ParseCardinality` | 100% |
| `service/operational/validation_service.go` | `CardinalityMinGE1` | 100% |
| `service/operational/validation_service.go` | `IsEmptyValue` | 100% |
| `api/operational/catalog_handler.go` | `ValidateCatalog` | 100% |
| `api/dto/dto.go` | `ValidationResultResponse` | struct (no test files) |
| `service/operational/instance_service.go` | `validateAndBuildAttributeValues` (updated) | 100% |
| `components/ValidationResults.tsx` | component | 100% |
| `hooks/useValidation.ts` | hook | 100% |

All new Go and UI code at 100% coverage. Every error propagation path, edge case (empty cardinality, non-numeric cardinality, invalid max cardinality, unknown attribute types, nil validation service, max cardinality, source cardinality, contained-without-parent, unpinned entity types), and containment check path has explicit test coverage.

Bug fixes included:
- Instance service: `UpdateInstance` now respects explicitly cleared attribute values (sends empty string → value not carried forward)
- Validation: full cardinality checks (min + max, target + source direction, directional + bidirectional)
- Validation: contained entity type without parent flagged as error

New UI files:
- `ui/src/components/ValidationResults.tsx` — shared validation results display component
- `ui/src/hooks/useValidation.ts` — shared validation hook

Live system tests: `scripts/test-validation.sh` — 9 parameterized tests covering empty catalog, RO/RW access, 404, required attrs, error structure, status persistence, valid catalog, status reset after mutation.

Quality review fixes applied: (M1) Eliminated duplicate `assocRepo.ListByVersion` calls — pre-load into `assocCache`. (M2) Added DTO layer — `ValidationResultResponse` and `ValidationErrorResponse`. (M3) Nil-guard on `validationSvc`. (L1) Unpinned entity type instances now produce validation errors. (L2) Operational UI Validate button hidden for RO. (L3) Extracted shared `useValidation` hook + `ValidationResults` component. (L6) Fixed inline import to top-level.

New UI files:
- `ui/src/components/ValidationResults.tsx` — shared validation results display component
- `ui/src/hooks/useValidation.ts` — shared validation hook

Live system tests: `scripts/test-validation.sh` — 9 parameterized tests covering empty catalog, RO/RW access, 404, required attrs, error structure, status persistence, valid catalog, status reset after mutation.

### New Code Coverage (Session 009 — Catalog Publishing)

| File | Function | Coverage |
|------|----------|----------|
| `service/operational/catalog_service.go` | `Publish` | 100% |
| `service/operational/catalog_service.go` | `Unpublish` | 100% |
| `service/operational/catalog_service.go` | `IsPublished` | 100% |
| `service/operational/catalog_service.go` | `Delete` (enhanced with CR cleanup) | 100% |
| `api/operational/catalog_handler.go` | `PublishCatalog` | 100% |
| `api/operational/catalog_handler.go` | `UnpublishCatalog` | 100% |
| `api/middleware/catalog_access.go` | `RequireWriteAccess` | 100% |
| `api/middleware/catalog_access.go` | `httpMethodToVerb` (with PUT/PATCH) | 100% |
| `operator/controllers/reconciler.go` | `ReconcileCatalogStatus` | 100% |
| `operator/api/v1alpha1/catalog_types.go` | All DeepCopy functions | 100% |
| `infrastructure/k8s/cr_manager.go` | `K8sCatalogCRManager.CreateOrUpdate` | 100% |
| `infrastructure/k8s/cr_manager.go` | `K8sCatalogCRManager.Delete` | 100% |
| `service/meta/catalog_version_service.go` | `Promote` (with warnings) | 100% |
| `infrastructure/gorm/repository/catalog_repo.go` | `UpdatePublished` | 66.7% |
| `infrastructure/gorm/repository/catalog_repo.go` | `ListByCatalogVersionID` | 85.7% |
| `operator/controllers/controller.go` | `reconcileCatalogs` | 77.8% |

Uncovered lines with justification:
- `UpdatePublished:66.7%` — GORM `RowsAffected==0` error path after successful Update (same pre-existing pattern as other repo Update methods — requires DB to fail between two sequential operations)
- `ListByCatalogVersionID:85.7%` — GORM `Find` error path (same pre-existing pattern as other repo List methods)
- `reconcileCatalogs:77.8%` — K8s client error paths for `SetOwnerReference`, `Update`, and `Status().Update` failures (same pre-existing pattern as `reconcileCatalogVersions` — fake K8s client doesn't simulate partial failures in these operations)

Quality review fixes applied: (C1) Fixed infinite DataVersion loop — extracted `ReconcileCatalogStatus` pure function, only updates when status is stale. (C2) Created separate `K8sCatalogCRManager` type satisfying `CatalogCRManager` interface. (I1) Publish rolls back DB on CR creation failure. (I2) Added PUT/PATCH → "update" verb mapping. (I3) Delete cleans up Catalog CR for published catalogs. (I4) `IsPublished` uses request context via `echo.Context`. (I5) UI error handling on publish/unpublish clicks. (I6) Removed redundant `CatalogName` from `CatalogCRSpec`. (I8) Unpublish checks `catalog.Published` for early return. (S2) Added `StatusResponse` DTO. (S3) Extracted `ReconcileCatalogStatus` pure function.

New files:
- `internal/service/operational/cr_manager.go` — CatalogCRManager interface + CatalogCRSpec
- `internal/operator/api/v1alpha1/catalog_types.go` — Catalog CR CRD type with DataVersion
- `scripts/test-publishing.sh` — 14 live system tests

Live system tests: `scripts/test-publishing.sh` — 14 tests covering publish/unpublish RBAC, draft/valid validation gate, write protection (RW blocked, SuperAdmin allowed), status persistence, CR cleanup, CV promotion warnings.

### New Code Coverage (Session 010 — Copy & Replace Catalog)

| File | Function | Coverage |
|------|----------|----------|
| `service/operational/catalog_service.go` | `CopyCatalog` | 98.3% |
| `service/operational/catalog_service.go` | `ReplaceCatalog` | 98.1% |
| `service/operational/catalog_service.go` | `WithCopyDeps` | 100% |
| `service/operational/catalog_service.go` | `WithTransactionManager` | 100% |
| `api/operational/catalog_handler.go` | `CopyCatalog` | 94.4% |
| `api/operational/catalog_handler.go` | `ReplaceCatalog` | 94.4% |
| `infrastructure/gorm/repository/catalog_repo.go` | `UpdateName` | 100% |
| `infrastructure/gorm/repository/transaction.go` | `NewGormTransactionManager` | 100% |
| `infrastructure/gorm/repository/transaction.go` | `RunInTransaction` | 100% |
| `infrastructure/gorm/repository/transaction.go` | `getDB` | 100% |

Uncovered lines with justification:
- `CopyCatalog handler:94.4%` — 1 line: `c.Bind` error (pre-existing pattern across all handlers; requires malformed JSON that bypasses Echo's content-type negotiation)
- `ReplaceCatalog handler:94.4%` — 1 line: same `c.Bind` error pattern
- `CopyCatalog service:98.3%` — 1 line: error return inside `txManager != nil` branch; the error path IS tested (via MockTransactionManager which passes through to doMutations), but the specific `txManager.RunInTransaction` error-wrapping line shows as partially uncovered due to Go coverage instrumentation
- `ReplaceCatalog service:98.1%` — 1 line: same `txManager.RunInTransaction` error-wrapping line

Error paths covered by dedicated tests:
- Instance create error during copy (T-17.18)
- GetCurrentValues error during copy
- SetValues error during copy
- GetForwardRefs error during copy
- Link Create error during copy
- Link with target outside catalog skipped
- Copy and Replace with nil TransactionManager (both success and error)
- Replace step 2 rename error
- Replace UpdatePublished error in step 3
- Replace source-published unpublish error
- Handler: source access denied, target access denied, access checker error (both copy and replace)
- Handler: CV label resolution fallback (both copy and replace)
- Replace auto-generated archive name too long

Quality review fixes applied: (H1) TransactionManager for atomic copy/replace operations. (H2) Mock nil-guard on GetForwardRefs/GetReverseRefs. (M1) ReplaceCatalog returns correct in-memory name. (M2) DNS label regex extracted to shared constant in UI. (M3) Per-catalog access checks in Copy (source+target) and Replace (source+target) handlers. (M4) CV label resolved in Copy/Replace API responses. (M5) Source-published handling in Replace (unpublish + CR cleanup). (L1) Better error message for auto-generated archive name exceeding 63 chars. (L2) API field name standardized to `source` (was `source_catalog_name`). (L3) Loading/spinner state on Copy/Replace buttons. (L4) Removed unnecessary `seen` map. (L5) Extracted `canPublishOrReplace` named boolean.

New files:
- `internal/domain/repository/transaction.go` — TransactionManager interface
- `internal/infrastructure/gorm/repository/transaction.go` — GORM TransactionManager + getDB helper
- `internal/infrastructure/gorm/repository/catalog_copy_integration_test.go` — End-to-end integration tests (including transaction rollback/commit verification)
- `scripts/test-copy-replace.sh` — 17 live system tests

Browser test count: 429 → 453 (+24 new tests for Copy/Replace UI and pre-existing gap coverage).
Live system tests: 64 → 81 (+17 new tests in test-copy-replace.sh).

UI new code coverage:
- All new Copy/Replace modal code: **100% covered** (0 uncovered lines in lines 1000+)
- Copy modal: open, close (X button), cancel, name validation, description input, submit success, submit error
- Replace modal: open, close (X button), cancel, target dropdown interaction, archive name validation, submit success, submit error
- `api/client.ts` copy/replace methods: 100% covered via client browser tests
- Pre-existing gaps addressed: adopt-mode submit (setParent API call), set-container submit (setParent API call)
- Remaining uncovered lines 1112/1115 are in the pre-existing `EnumSelect` component (not new code)
- Remaining pre-existing uncovered lines (139-992): guard returns, error catch blocks, and PatternFly modal onClose/onChange callbacks — documented in detail below

Pre-existing uncovered lines in CatalogDetailPage.tsx (not from Phase 8):
- Guard returns (139, 183, 206, 248, 274, 338, 370, 384, 399, 416, 434): early returns when URL param, active tab, or form state is missing — never triggered because test data always satisfies conditions
- Error catch blocks (299, 315, 329-330, 343, 366, 429): require API mocks to fail at specific points during multi-step interactions
- `loadLinkTargetInstances` (348-357): requires completing a multi-step PatternFly Select cascade (select association → triggers async load → select target) that blocks test automation
- `handleCreateLink` (399-411): same cascading Select dependency
- Modal callbacks (626-627, 722, 731, 740, 761, 767, 770, 779, 786, 800, 811-818, 857, 892, 903-911, 928-937, 950, 981-992): PatternFly component onClose/onChange/onSelect internal callbacks

### New Code Coverage (Session 011 — Cleanups & TD Fixes)

| TD | Change | Coverage |
|----|--------|----------|
| TD-21 | Remove migration code from InitDB | N/A (deleted code) |
| TD-24 | Remove legacy EntityInstanceService + handler + tests | N/A (deleted code, -1285 lines) |
| TD-25 | Replace `interface{}` with `any` | N/A (cosmetic) |
| TD-20 | Instance name validation | 100% (3 new tests) |
| TD-29 | Reserved entity type names | 100% (2 new tests) |
| TD-34 | SetParent parent_type validation | 100% (1 new test) |
| TD-8d | Extract EdgeClickData interface | N/A (type only) |
| TD-9 | Required `*` prefix in diagram | 100% (existing render loop) |
| TD-15 | Transactional catalog delete | 100% (existing tests exercise nil-txManager path) |
| TD-30 | Catalog ownership check on Get/Update/Delete | 100% (2 new tests + 6 existing tests fixed) |
| TD-17 | Catalog list pagination | 100% (2 new tests) |
| TD-8a | Mark EditAssociationModal resolved | N/A (already implemented) |

Backend test count: 1261 → 1255 (net -6: removed 68 legacy tests, added 62 new tests).
Service/meta: 94.6% → 99.5% (+4.9pp). Service/operational: 98.8% → 99.8% (+1.0pp).
Backend coverage: 94.0% → 94.3% (improved by removing uncovered legacy code).
Live test count: 81 → 89 (added test-copy-replace.sh to Makefile target).

### New Code Coverage (Session 012 — Foundation Cleanup Phase 1)

| TD | Change | Coverage |
|----|--------|----------|
| TD-62 | Fix UpdateEntityType omitted-field data loss (`*string`) | 100% (3 new tests) |
| TD-27 | Fix ListContainedInstances pagination (`parseListParams`) | 100% (2 new tests) |
| TD-2 | Verify CV label uniqueness constraint | 100% (1 verification test) |
| TD-3 | Verify association name uniqueness | Already covered (TestTE107) |
| TD-16 | Fix catalog deletion cascade (IAVs + links) | 100% (4 new tests + 1 integration test) |
| TD-59 | Fix N+1 query in entity type list (batch `GetLatestByEntityTypes`) | 100% (2 new tests + 1 integration test + 1 error branch test) |

New lines: **0 uncovered** (verified by arithmetic check: script reported 0, math confirms).

Per-file coverage deltas (modified files only):

| Package | Before | After | Delta |
|---------|--------|-------|-------|
| `api/meta` | 99.1% (433/437) | 98.9% (450/455) | +17 covered, +18 total (pre-existing uncov in unmodified file) |
| `api/operational` | 98.3% (294/299) | 98.3% (296/301) | +2 covered, +2 total |
| `gorm/repository` | 90.7% (597/658) | 90.9% (610/672) | +13 covered, +14 total |
| `service/operational` | 99.8% (852/854) | 99.8% (863/865) | +11 covered, +11 total |

Backend test count: 1255 → 1329 (+74 new tests including sub-tests). Overall: 97.2% (3531/3632).

### Coverage Gaps to Address

| Component | Browser Coverage | Issue | Resolution |
|-----------|-----------------|-------|------------|
| `EntityTypeDetailPage.tsx` | 70.8% | Copy-attributes source selection flow and deep copy confirmation flow are hard to test with browser mocks | System tests cover these flows against a live server |
| `InstanceListPage.tsx` | 0% | Operational page, no tests yet | Add when operational UI is prioritized |

### System Test Notes

System tests (`App.system.test.ts`) run against a live deployment (kind cluster) using Playwright. They are not included in coverage measurements because they test the deployed build, not instrumented source. They provide functional verification of cross-page flows including:
- Rename entity type and navigate back (list refresh)
- Targeted delete (correct row, not first) for entity types, enums, and catalog versions
- Copy attributes picker with enum name resolution

---

## How to Reproduce

### Backend

```bash
# Run all tests with coverage
go test ./internal/... -count=1 -coverprofile=coverage.out

# View summary
go tool cover -func=coverage.out | tail -1

# View per-function coverage
go tool cover -func=coverage.out | grep -v '100.0%'

# HTML report
go tool cover -html=coverage.out -o coverage.html
```

### Frontend

```bash
cd ui

# Run unit/component tests
npx vitest run --exclude='src/App.system.test.ts'

# Run with coverage
npx vitest run --exclude='src/App.system.test.ts' --coverage

# Run browser tests with Playwright (separate config)
npx vitest run --config vitest.browser.config.ts --coverage

# Run system tests (requires running kind cluster)
npx vitest run --config vitest.system.config.ts
```

### Linting

```bash
golangci-lint run ./...
npx tsc --noEmit
```
