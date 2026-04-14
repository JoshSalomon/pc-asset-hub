e# AI Asset Hub — Test Coverage Report

Last updated: 2026-04-12

---

## Summary

| Layer | Tests | Pass Rate | Statements | Lines |
|-------|-------|-----------|------------|-------|
| Backend (Go) | 1572 | 100% | 97.6% (4051/4149) | — |
| UI — Unit tests (jsdom) | 75 | 100% | — | — |
| UI — Browser tests (Playwright) | 930 | 100% | 94.7% (2537/2680) | 97.2% (2291/2357) |
| UI — System tests (Playwright + live server) | 121 | 100% | — | — |
| Live system (bash scripts) | 303 | 100% | — | — |
| **Total** | **3001** | **100%** | — | — |

---

## Backend Coverage by Package

| Package | Coverage | Notes |
|---------|----------|-------|
| `internal/api/health` | 90.0% (9/10) | Readyz DB-ping error path |
| `internal/api/meta` | 99.8% (492/493) | `defaultListParams` in version_history_handler: 1 pre-existing uncovered function |
| `internal/api/middleware` | 100.0% (69/69) | |
| `internal/api/operational` | 98.4% (307/312) | Copy/Replace/Update handlers bind-error branches only |
| `internal/domain/errors` | 100.0% (32/32) | |
| `internal/domain/models` | 100.0% (8/8) | |
| `internal/infrastructure/config` | 100.0% (21/21) | |
| `internal/infrastructure/gorm/models` | 89.3% (50/56) | InitDB migration paths (6 lines — one-time legacy schema cleanup, human approved) |
| `internal/infrastructure/gorm/repository` | 92.5% (708/765) | GORM error branches on Delete/Update, partial DB failure paths |
| `internal/infrastructure/k8s` | 92.6% (50/54) | K8s client error paths |
| `internal/operator/api/v1alpha1` | 97.7% (85/87) | `DeepCopyObject` nil-receiver guard |
| `internal/operator/controllers` | 94.3% (198/210) | `SetupWithManager` (envtest — deferred to Phase B), `SetOwnerReference` error branches |
| `internal/operator/crdgen` | 94.3% (33/35) | `json.Marshal` error guards on well-formed inputs |
| `internal/service/meta` | 99.6% (959/963) | BulkCopy error paths, requiresDeepCopy edge cases |
| `internal/service/operational` | 99.8% (987/989) | Cycle guard in resolveParentChain, partial DB failure paths |
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
| `api/meta/catalog_version_handler.go` | `Promote`/`Demote`/`Delete` | — | RoleRO inline switch case in each (unreachable behind RBAC `requireRW` middleware) |
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

**CatalogVersionDetailPage.tsx** (5 statements — Phase 2 handler guards):

| Line | Code | Why unreachable |
|------|------|-----------------|
| L276 | `if (!id) return` | `id` from `useParams()` — always set by route; handler only callable after component renders with valid id |
| L287 | `if (!id) return` | Same — guard in `handleSaveLabel` |
| L321 | `if (!id \|\| !selectedEtvId) return` | Same — guard in `handleAddPin`; `selectedEtvId` set before submit button is enabled |
| L350 | `if (!id) return` | Same — guard in `handleUpdatePinVersion` |
| L361 | `if (!id) return` | Same — guard in `handleRemovePin` |

**CatalogDetailPage.tsx** (4 statements — Phase 2 handler guards + PF6 Select callback):

| Line | Code | Why unreachable |
|------|------|-----------------|
| L276 | `if (!name) return` | `name` from `useParams()` — always set by route |
| L288 | `if (!name \|\| !cvId) return` | Same — guard in `handleChangeCv`; cvId always set by Select onSelect |
| L301 | `if (cvListLoading) return` | Debounce guard — async timing makes deterministic testing impractical |
| L343 | `onOpenChange={(open) => {...}}` | PF6 Select internal callback — fires from PF6's window-level click/key handlers; covered only when PF6's effect listeners are active (not reliably triggered in browser test mode) |

**CatalogDetailPage.tsx** (11 statements — pre-existing page-level handlers for add-child, link, set-parent modals):

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
| `EntityTypeDetailPage.browser.test.tsx` | 136 | Pass |
| `EntityTypeListPage.browser.test.tsx` | 12 | Pass |
| `TypeDefinitionListPage.browser.test.tsx` | 60 | Pass |
| `TypeDefinitionDetailPage.browser.test.tsx` | 47 | Pass |
| `CatalogVersionDetailPage.browser.test.tsx` | 57 | Pass |
| `CatalogListPage.browser.test.tsx` | 20 | Pass |
| `CatalogDetailPage.browser.test.tsx` | 137 | Pass |
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
| `AddAttributeModal.browser.test.tsx` | 9 | Pass |
| `EditAttributeModal.browser.test.tsx` | 6 | Pass |
| `AddAssociationModal.browser.test.tsx` | 7 | Pass |
| `CopyAttributesModal.browser.test.tsx` | 7 | Pass |
| `RenameEntityTypeModal.browser.test.tsx` | 4 | Pass |
| `useContainmentTree.browser.test.tsx` | 11 | Pass |
| `InstanceDetailPanel.browser.test.tsx` | 14 | Pass |
| `OperationalCatalogDetailPage.browser.test.tsx` | 36 | Pass |
| `OperationalCatalogListPage.browser.test.tsx` | 13 | Pass |
| `OperationalApp.browser.test.tsx` | 3 | Pass |
| `useValidation.browser.test.tsx` | 6 | Pass |
| `useCatalogDiagram.browser.test.tsx` | 5 | Pass |
| `CopyCatalogModal.browser.test.tsx` | 5 | Pass |
| `AttributeFormFields.browser.test.tsx` | 15 | Pass |
| `EntityTypeDiagram.browser.test.tsx` | 3 | Pass |
| `LandingPage.browser.test.tsx` | 12 | Pass |
| **Total** | **930** | **100% pass** |

### System Tests (Playwright + live server)

| Test File | Tests | Status |
|-----------|-------|--------|
| `App.system.test.ts` | 30 | Pass |
| `CatalogDetail.system.test.ts` | 15 | Pass |
| `CatalogVersionDetail.system.test.ts` | 12 | Pass |
| `DataViewer.system.test.ts` | 17 | Pass |
| `LandingPage.system.test.ts` | 4 | Pass |
| `SecurityFlows.system.test.ts` | 21 | Pass |
| `TypeSystem.system.test.ts` | 22 | Pass |
| **Total** | **121** | **100% pass** |

### Code Coverage (v8 provider)

Coverage is measured using `@vitest/coverage-v8`. The two test suites run independently with separate configs.

**Browser tests** (primary coverage — exercises full component rendering via Playwright):

| File | Stmts (covered/total) | Stmts % | Lines (covered/total) | Lines % |
|------|-----------------------|---------|-----------------------|---------|
| `App.tsx` | 275/309 | 89.0% | 250/269 | 92.9% |
| `api/client.ts` | 90/97 | 92.8% | 84/91 | 92.3% |
| `components/AddAssociationModal.tsx` | 87/87 | 100.0% | 79/79 | 100.0% |
| `components/AddAttributeModal.tsx` | 41/42 | 97.6% | 35/35 | 100.0% |
| `components/AddChildModal.tsx` | 89/90 | 98.9% | 75/75 | 100.0% |
| `components/AttributeFormFields.tsx` | 29/29 | 100.0% | 27/27 | 100.0% |
| `components/CopyAttributesModal.tsx` | 49/49 | 100.0% | 39/39 | 100.0% |
| `components/CopyCatalogModal.tsx` | 12/12 | 100.0% | 12/12 | 100.0% |
| `components/CreateInstanceModal.tsx` | 14/14 | 100.0% | 13/13 | 100.0% |
| `components/DiagramTabContent.tsx` | 4/4 | 100.0% | 3/3 | 100.0% |
| `components/EditAssociationModal.tsx` | 85/92 | 92.4% | 76/82 | 92.7% |
| `components/EditAttributeModal.tsx` | 38/38 | 100.0% | 31/31 | 100.0% |
| `components/EditInstanceModal.tsx` | 19/19 | 100.0% | 17/17 | 100.0% |
| `components/EntityTypeDiagram.tsx` | 94/102 | 92.2% | 89/97 | 91.8% |
| `components/InstanceDetailPanel.tsx` | 8/8 | 100.0% | 8/8 | 100.0% |
| `components/LinkModal.tsx` | 43/44 | 97.7% | 36/36 | 100.0% |
| `components/RenameEntityTypeModal.tsx` | 12/12 | 100.0% | 12/12 | 100.0% |
| `components/ReplaceCatalogModal.tsx` | 17/17 | 100.0% | 15/15 | 100.0% |
| `components/SetParentModal.tsx` | 27/27 | 100.0% | 23/23 | 100.0% |
| `components/ValidationResults.tsx` | 12/12 | 100.0% | 10/10 | 100.0% |
| `context/AuthContext.tsx` | 8/9 | 88.9% | 7/7 | 100.0% |
| `hooks/useAssociationManagement.ts` | 49/52 | 94.2% | 48/48 | 100.0% |
| `hooks/useAttributeManagement.ts` | 83/91 | 91.2% | 78/79 | 98.7% |
| `hooks/useCatalogData.ts` | 48/48 | 100.0% | 41/41 | 100.0% |
| `hooks/useCatalogDiagram.ts` | 25/25 | 100.0% | 24/24 | 100.0% |
| `hooks/useContainmentTree.ts` | 60/60 | 100.0% | 57/57 | 100.0% |
| `hooks/useEntityTypeData.ts` | 61/64 | 95.3% | 57/57 | 100.0% |
| `hooks/useInlineEdit.ts` | 32/32 | 100.0% | 30/30 | 100.0% |
| `hooks/useInstanceDetail.ts` | 56/57 | 98.2% | 55/56 | 98.2% |
| `hooks/useInstances.ts` | 66/69 | 95.7% | 65/65 | 100.0% |
| `hooks/usePinManagement.ts` | 72/72 | 100.0% | 66/66 | 100.0% |
| `hooks/useValidation.ts` | 21/21 | 100.0% | 19/19 | 100.0% |
| `pages/LandingPage.tsx` | 21/21 | 100.0% | 20/20 | 100.0% |
| `pages/meta/CatalogDetailPage.tsx` | 208/220 | 94.5% | 179/180 | 99.4% |
| `pages/meta/CatalogListPage.tsx` | 74/90 | 82.2% | 67/75 | 89.3% |
| `pages/meta/CatalogVersionDetailPage.tsx` | 144/165 | 87.3% | 130/142 | 91.5% |
| `pages/meta/EntityTypeDetailPage.tsx` | 155/161 | 96.3% | 134/134 | 100.0% |
| `pages/meta/EntityTypeListPage.tsx` | 11/12 | 91.7% | 11/12 | 91.7% |
| `pages/meta/TypeDefinitionDetailPage.tsx` | 86/90 | 95.6% | 76/76 | 100.0% |
| `pages/meta/TypeDefinitionListPage.tsx` | 129/132 | 97.7% | 121/122 | 99.2% |
| `pages/operational/OperationalCatalogDetailPage.tsx` | 60/61 | 98.4% | 52/52 | 100.0% |
| `utils/buildTypedAttrs.ts` | 15/15 | 100.0% | 13/13 | 100.0% |
| `utils/dnsLabel.ts` | 3/3 | 100.0% | 2/2 | 100.0% |
| `utils/statusColor.ts` | 5/6 | 83.3% | 5/6 | 83.3% |
| **All files (44)** | **2537/2680** | **94.7%** | **2291/2357** | **97.2%** |

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

### New Code Coverage (Session 013 — Phase 2: Missing CRUD Capabilities)

| US | Change | Coverage |
|----|--------|----------|
| US-49 | CV metadata edit (label + description) — `UpdateCatalogVersion` service + handler | 100% (8 service + 5 handler tests) |
| US-50 | Catalog metadata edit (name + description) — `UpdateMetadata` service + handler | 100% (15 service + 7 handler tests) |
| US-51 | Catalog re-pinning (change CV) — extends `UpdateMetadata` | 100% (included in US-50 tests) |
| US-52 | CV pin editing (add/remove) — `AddPin`/`RemovePin` service + handler | 100% (11 service + 6 handler tests) |

New lines: **0 uncovered** (verified by arithmetic: script + manual per-package delta).

Per-file coverage deltas (modified files only):

| Package | Phase 1 (before) | Phase 2 (after) | Delta |
|---------|---------|--------|-------|
| `api/meta` | 98.9% (450/455) | 99.0% (474/479) | +24 covered, +24 total |
| `api/operational` | 98.3% (296/301) | 98.4% (307/312) | +11 covered, +11 total |
| `gorm/repository` | 90.9% (610/672) | 91.2% (631/692) | +21 covered, +20 total (improved) |
| `service/meta` | 99.5% (739/743) | 99.5% (780/784) | +41 covered, +41 total |
| `service/operational` | 99.8% (863/865) | 99.8% (910/912) | +47 covered, +47 total |

Backend test count: 1329 → 1388 (+59 new tests). Overall: 97.4% (3676/3776).

Quality review fixes: C-1 (RemovePin pin_id), I-1 (RequireWriteAccess middleware), I-2 (AddPin response), I-3 (single Update call), I-4 (11 live test cases), S-3 (param rename). Added TD-65, TD-66 to PRD.

### New Code Coverage (Session 014 — US-53: CV Pin Management)

| Change | Coverage |
|--------|----------|
| Fix AddPin entity type check (check TYPE ID, not ETV ID) | 100% (1 new test + 1 error path test) |
| UpdatePin service (change pinned version, entity type mismatch guard) | 100% (8 service tests) |
| UpdatePin handler (PUT /catalog-versions/:id/pins/:pin-id) | 100% (5 handler tests) |
| Pin Update GORM impl | 100% (1 integration test) |

New lines: **0 uncovered** (verified by arithmetic).

Per-file coverage deltas:

| Package | Before | After | Delta |
|---------|--------|-------|-------|
| `api/meta` | 474/479 (5 uncov) | 484/489 (5 uncov) | +10 covered, +10 total |
| `gorm/repository` | 632/692 (60 uncov) | 635/695 (60 uncov) | +3 covered, +3 total |
| `service/meta` | 782/786 (4 uncov) | 805/809 (4 uncov) | +23 covered, +23 total |

Backend test count: 1392 → 1408 (+16 new tests). Overall: 97.4% (3716/3815).

### New Code Coverage (Session 015 — Phase 2 UI + Browser Test Fixes)

**UI changes (CatalogVersionDetailPage.tsx, CatalogDetailPage.tsx, client.ts, types/index.ts):**

| Component | Change | Coverage |
|-----------|--------|----------|
| `CatalogVersionDetailPage.tsx` | Inline edit (description, label), Add/Remove/Update Pin, BOM version dropdown | 96.2% stmts |
| `CatalogDetailPage.tsx` | Inline edit (description), CV selector dropdown | 92.6% stmts |
| `client.ts` | `catalogVersions.update`, `addPin`, `updatePin`, `removePin`, `catalogs.update` | Covered via page tests + client tests |
| `types/index.ts` | `pin_id` field on `CatalogVersionPin` | Type only |

New UI lines: **9 uncovered** (5 `useParams` guards + 1 debounce guard + 1 PF6 callback + 2 `useParams`/param guards in CatalogDetailPage).

Browser test count: 774 → 777 (+3 coverage tests for BOM version dropdown toggle, Escape close, and version load error).

**PF6 Select-in-Modal fix:** Extracted `PinEntityTypeSelect` and `PinVersionSelect` wrapper components that manage their own `isOpen` state. This prevents the parent component (and Modal) from re-rendering when the dropdown opens, avoiding PF6 Modal's `toggleSiblingsFromScreenReaders` from setting `aria-hidden` on the Popper portal.

**Test fixes:** Added `exact: true` to `getByRole('button', { name: 'Model' })` to disambiguate from "Version for Model" aria-label. Fixed Add Pin test data to include unpinned entity types (filtering removes already-pinned). Used `data-testid` on SelectOptions to bypass `aria-hidden` for Select dropdowns inside Modals.

Per-file coverage deltas:

| File | Before | After | Delta |
|------|--------|-------|-------|
| `CatalogVersionDetailPage.tsx` | 84.5% stmts | 96.2% stmts | +11.7pp |
| `CatalogDetailPage.tsx` | 92.6% stmts | 92.6% stmts | 0pp (new lines offset by new guards) |
| `client.ts` | 90.2% stmts | 90.2% stmts | 0pp (new methods covered by page/client tests) |

Live test count: 239 → 242 (+3 new US-53 tests: UpdatePin, entity type mismatch 400, AddPin duplicate entity type 409).

### New Code Coverage (Session 016 — Phase 2c Security Fixes + Phase 3 Cleanup)

**Backend changes:**

| File | Change | Coverage |
|------|--------|----------|
| `catalog_version_handler.go` | Add `mapRole` call in `Update` for stage guard (TD-71) | 100% new lines |
| `catalog_version_service.go` | Rename `checkPinEditAllowed` → `checkCVEditAllowed`, add stage guard to `UpdateCatalogVersion` | 100% new lines |
| `catalog_handler.go` | Fix validate route to use `writeMiddleware` (published catalog bypass fix) | Routing only |

**UI changes:**

| File | Change | Coverage |
|------|--------|----------|
| `CatalogVersionDetailPage.tsx` | Unify `canEdit`/`canEditPins` with stage guard logic | 100% new lines |
| `CatalogDetailPage.tsx` | Add `canValidate` guard (block validate on published unless SuperAdmin) | 100% new lines |

New backend tests: **+3 coverage tests** for previously uncovered handler branches:
- `TestCVUpdate_MapRole_RO` — exercises `mapRole` RO case by bypassing `requireRW` middleware
- `TestCVUpdate_MapRole_Unknown` — exercises `mapRole` default case by injecting unknown role into context
- `TestCVPromote_WithWarnings` — exercises Promote warnings loop body with mock catalog repo returning draft/invalid catalogs

Per-file coverage deltas:

| Package/File | Before | After | Delta |
|-------------|--------|-------|-------|
| `api/meta` | 484/489 (5 uncov) | 495/499 (4 uncov) | +11 covered, +10 total, **-1 uncov** |
| `api/operational` | 307/312 (5 uncov) | 307/312 (5 uncov) | unchanged |
| `service/meta` | 805/809 (4 uncov) | 821/825 (4 uncov) | +16 covered, +16 total |
| `CatalogVersionDetailPage.tsx` | 230/256 (89.8%) | 232/258 (89.9%) | +2 covered, +2 total |
| `CatalogDetailPage.tsx` (op) | 261/282 (92.6%) | 262/283 (92.6%) | +1 covered, +1 total |

Backend test count: 1409 → 1450 (+41 new tests including TD-71 stage guard + coverage tests).
Browser test count: 777 → 784 (+7 tests).
Live test count: 242 → 303 (+61 tests across multiple scripts).

### New Code Coverage (Session 017 — Type System)

**Backend:** Enums replaced by versioned type definitions. 9 base types. All new Go code at 100% coverage except 3 GORM partial-DB-failure lines and 6 InitDB migration lines (human approved).

| File | Function | Coverage |
|------|----------|----------|
| `service/meta/type_definition_service.go` | All 13 functions | 100% |
| `service/meta/seed_system_types.go` | `SeedSystemTypes` | 100% |
| `service/operational/type_resolver.go` | `ResolveBaseTypes`, `ResolveAttrTypeInfo` | 100% |
| `api/meta/type_definition_handler.go` | All 8 functions | 100% |
| `gorm/repository/type_definition_repo.go` | 16 functions | 97% (3 partial-DB-failure lines) |
| `gorm/models/models.go` | `TypeDefinitionVersion.ToModel` (corruption handling) | 100% |
| `service/operational/instance_service.go` | `mapAttributeValues`, `validateAndBuildAttributeValues` (all 9 base types) | 100% |
| `service/operational/validation_service.go` | `IsEmptyValue` (all base types), corrupted constraints check | 100% |

**UI:** Enum pages replaced by TypeDefinition pages. All new UI code covered.

| File | Coverage |
|------|----------|
| `TypeDefinitionListPage.tsx` | 97.7% (129/132) — 3 defensive guards |
| `TypeDefinitionDetailPage.tsx` | 95.6% (86/90) — 4 useParams guards |
| `AttributeFormFields.tsx` | 100% (29/29) — all 9 base type controls + multiline |
| `AddAttributeModal.tsx` | 97.6% (41/42) — 1 Select-state guard |
| `EditAttributeModal.tsx` | 100% (38/38) |
| `CopyAttributesModal.tsx` | 100% (49/49) |
| `buildTypedAttrs.ts` | 100% (15/15) |

Backend test count: 1460 → 1570 (+110). Browser test count: 777 → 926 (+149).

UI coverage delta vs git baseline (2391/2561 → 2521/2664): **-27 uncovered** (improvement).
Per-file regressions from 100%: AddAttributeModal.tsx (39/39 → 41/42, +1 unreachable guard), LinkModal.tsx (27/27 → 43/44, +1 unreachable guard).
Deleted: EnumSelect.tsx, EnumDetailPage.tsx, EnumListPage.tsx. Excluded from coverage: test-helpers/system.ts.

Quality review fixes applied: (1) N+1 query in List handler → batch `GetLatestByTypeDefinitions`. (2) `resolveBaseTypes` duplication → extracted to `type_resolver.go`. (3) Corrupted JSON constraints → `{"_raw": ...}` wrapper + `IsCorruptedConstraints`/`ExtractRawConstraints` + validation flags it. (4) Missing nil check in `mapAttributeValues`.

### New Code Coverage (Session 018 — Type System: latest_version_id, multiline string)

**Backend:** Added `GetLatestVersionInfo()` service method and `LatestVersionID` field to DTO. All new Go lines covered.

| File | Function | Coverage |
|------|----------|----------|
| `service/meta/type_definition_service.go` | `GetLatestVersionInfo` | 100% |
| `api/meta/type_definition_handler.go` | `Create`, `List`, `GetByID` (populate LatestVersionID) | 100% |
| `api/dto/dto.go` | `TypeDefinitionResponse.LatestVersionID` field | N/A (struct field) |

**UI:** Fixed AddAttributeModal/EditAttributeModal to use `td.latest_version_id` instead of `td.id`. Added multiline string TextArea in AttributeFormFields.

| File | Coverage |
|------|----------|
| `AttributeFormFields.tsx` | 100% (29/29) — multiline TextArea onChange covered |
| `EditAttributeModal.tsx` | 100% (38/38) — new guard + td.latest_version_id covered |
| `AddAttributeModal.tsx` | 97.6% (41/42) — 1 pre-existing guard (unchanged) |
| `types/index.ts` | N/A (type definition only) |

Backend test count: 1572. Browser test count: 926 → 928 (+2 coverage tests).

Per-file coverage deltas:

| File | Before | After | Delta |
|------|--------|-------|-------|
| `service/meta` | 99.6% (950/954) 4 uncov | 99.6% (959/963) 4 uncov | +9 covered, +9 total |
| `api/meta` | 99.8% (473/474) 1 uncov | 99.8% (473/474) 1 uncov | unchanged |
| `EditAttributeModal.tsx` | 100% (34/34) | 100% (38/38) | +4 covered, +4 total |
| `AttributeFormFields.tsx` | 100% (28/28) | 100% (29/29) | +1 covered, +1 total |
| `AddAttributeModal.tsx` | 97.6% (41/42) | 97.6% (41/42) | unchanged |

Test fixture fixes: Updated 14 browser test expectations to use `latest_version_id` instead of type definition ID for `typeDefinitionVersionId`. Added `type_definition_version_id` to mock attributes in EntityTypeDetailPage tests. Added system type definitions (string, number) to mock data.

### New Code Coverage (Session 019 — TD-94: Number min/max leading zeros fix)

**Bug fix:** Extracted `NumericConstraintFields` component from `ConstraintsForm` in `TypeDefinitionListPage.tsx`. Uses local string state and parses on blur instead of onChange, preventing leading zeros after decimal point from being dropped during typing.

| File | Change | Coverage |
|------|--------|----------|
| `TypeDefinitionListPage.tsx` | Extracted `NumericConstraintFields` component (+45 lines, -25 lines) | 97.7% (129/132) |
| `AttributeFormFields.tsx` | Added multiline TextArea branch | 100% (29/29) |
| `EditAttributeModal.tsx` | Updated to use `td.latest_version_id` | 100% (38/38) |
| `AddAttributeModal.tsx` | Updated to use `td.latest_version_id` | 97.6% (41/42) |

New lines: **0 uncovered** (verified by arithmetic: baseline 143 uncovered stmts, current 143 uncovered stmts, delta = 0).

Per-file coverage deltas:

| File | Before | After | Delta |
|------|--------|-------|-------|
| `TypeDefinitionListPage.tsx` | 97.5% (118/121) 3 uncov | 97.7% (129/132) 3 uncov | +11 covered, +11 total |
| `AttributeFormFields.tsx` | 100% (28/28) 0 uncov | 100% (29/29) 0 uncov | +1 covered, +1 total |
| `EditAttributeModal.tsx` | 100% (34/34) 0 uncov | 100% (38/38) 0 uncov | +4 covered, +4 total |
| `AddAttributeModal.tsx` | 97.6% (41/42) 1 uncov | 97.6% (41/42) 1 uncov | unchanged |

Browser test count: 926 -> 929 (+3 tests: TD-94 keystroke test + 2 from session 018).

### New Code Coverage (Session 020 — Attribute List type_name/base_type resolution)

**Bug fix:** `AttributeHandler.List` now resolves `type_name` and `base_type` from type definitions for each attribute. Added `tdvRepo` and `tdRepo` to `AttributeHandler` struct. `cmd/api-server/main.go` updated to pass the new repos.

| File | Function | Coverage |
|------|----------|----------|
| `api/meta/attribute_handler.go` | `NewAttributeHandler` | 100% |
| `api/meta/attribute_handler.go` | `List` (type resolution) | 100% |
| `api/meta/attribute_handler.go` | All 8 functions | 100% |

New lines: **0 uncovered** (verified by arithmetic: baseline 98 uncovered, current 98 uncovered, delta = 0).

Per-file coverage deltas:

| Package | Before | After | Delta |
|---------|--------|-------|-------|
| `api/meta` | 99.8% (473/474) 1 uncov | 99.8% (492/493) 1 uncov | +19 covered, +19 total |

Backend test count: 1572 (unchanged). Overall: 97.6% (4051/4149).

`cmd/api-server/main.go` is excluded from coverage (deferred to Phase B — server bootstrap).

### New Code Coverage (Session 021 — System attributes "unknown" type fix)

**Bug fix:** Added conditional in `EntityTypeDetailPage.tsx` to show "string" for system attributes instead of resolving `type_name`/`base_type` (which are empty for system attrs, causing "unknown" display).

| File | Change | Coverage |
|------|--------|----------|
| `EntityTypeDetailPage.tsx` | System attr type conditional (4 lines changed) | 96.3% (155/161) — all new lines covered |

New lines: **0 uncovered** (verified by arithmetic: baseline 143 uncovered stmts, current 143 uncovered stmts, delta = 0).

Per-file coverage deltas:

| File | Before | After | Delta |
|------|--------|-------|-------|
| `EntityTypeDetailPage.tsx` | 96.3% (155/161) 6 uncov | 96.3% (155/161) 6 uncov | unchanged (new code replaces old code with same stmt count) |

Browser test count: 929 -> 930 (+1 test for system attribute type display).

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
