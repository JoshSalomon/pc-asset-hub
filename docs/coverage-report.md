# AI Asset Hub — Test Coverage Report

Last updated: 2026-03-09

---

## Summary

| Layer | Tests | Pass Rate | Statements | Lines |
|-------|-------|-----------|------------|-------|
| Backend (Go) | 804+ | 100% | ~87% | — |
| UI — Unit tests | 62 | 100% | 17.9% | 20.6% |
| UI — Browser tests (Playwright) | 254 | 100% | ~81% | ~86% |
| UI — System tests (Playwright + live server) | 30 | 100% | — | — |
| **Total** | **1150+** | **100%** | — | — |

---

## Backend Coverage by Package

| Package | Coverage | Notes |
|---------|----------|-------|
| `internal/api/health` | 90.0% | Readyz DB-ping error path |
| `internal/api/meta` | 88.8% | Handler bind-error branches in some handlers |
| `internal/api/middleware` | 100.0% | |
| `internal/api/operational` | 94.6% | Catalog handler at 100%; legacy handler bind-error branches |
| `internal/domain/errors` | 100.0% | |
| `internal/infrastructure/config` | 100.0% | |
| `internal/infrastructure/gorm/models` | 100.0% | |
| `internal/infrastructure/gorm/repository` | 90.5% | GORM error branches on Delete/Update, `CatalogVersionGormRepo.Delete` at 0%; new `CatalogGormRepo` at 90-100% |
| `internal/infrastructure/k8s` | 92.6% | K8s client error paths |
| `internal/operator/api/v1alpha1` | 98.2% | `DeepCopyObject` nil-receiver guard |
| `internal/operator/controllers` | 85.5% | `SetupWithManager`, Route reconciliation, complex controller paths |
| `internal/operator/crdgen` | 84.2% | `GenerateCRDJSON`, `GenerateCR` error paths |
| `internal/service/meta` | 94.6% | `ListAttributes` and `ListValues` at 0% (trivial delegators) |
| `internal/service/operational` | 100.0% | |
| `internal/service/validation` | 95.6% | |

### Excluded from Coverage

These packages are not counted toward coverage because they contain no business logic:

| Package | Reason |
|---------|--------|
| `internal/api/dto` | Pure struct definitions, no test files |
| `internal/domain/models` | Pure struct definitions, no statements |
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
| `client.browser.test.ts` | 36 | Pass |
| `EntityTypeDetailPage.browser.test.tsx` | 77 | Pass |
| `EntityTypeListPage.browser.test.tsx` | 12 | Pass |
| `EnumDetailPage.browser.test.tsx` | 24 | Pass |
| `EnumListPage.browser.test.tsx` | 14 | Pass |
| `CatalogVersionDetailPage.browser.test.tsx` | 27 | Pass |
| `CatalogListPage.browser.test.tsx` | 11 | Pass |
| **Total** | **254** | **100% pass** |

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
