# Foundation Cleanup Implementation Plan

**Goal:** Fix data integrity bugs, add missing CRUD capabilities, and clean up UI code quality across the schema management layer before adding new features.

**Architecture:** Bottom-up changes: domain models/repos -> service layer -> API handlers -> UI. Each phase is a separate commit.

**Tech Stack:** Go (backend), React/TypeScript/PatternFly (UI), GORM (ORM), SQLite (dev DB), Vitest+Playwright (browser tests)

## Mandatory Process — Skills to Follow

Each phase MUST follow the project's established skills. **These are not optional — follow them to the letter.**

### For feature work (new endpoints, new UI capabilities):
Use `/feat-plan` skill — 8-phase process with approval gates:
1. Requirements & PRD update
2. High-level test plan
3. Detailed test plan
4. Implementation (TDD red/green)
5. Deploy & live test
6. Quality review (3 parallel code reviewers)
7. Coverage & verification (invoke `/coverage-test`)
8. Documentation & manual approval

### For bug fixes (TD items that fix broken behavior):
Use `/bug-solver` skill — end-to-end bug resolution with TDD.

### For coverage verification (mandatory after each phase):
Use `/coverage-test` skill — with the arithmetic check, per-file deltas, and hard gate. **Never skip Step 2c (arithmetic verification).** Delete old coverage files before measuring.

### For quality review (mandatory after each phase):
Dispatch 3 parallel code-reviewer agents. Present findings. Wait for human approval before fixing.

### Skill Selection Guide

| Task | Skill | Why |
|------|-------|-----|
| TD-62 (fix omitted-field bug) | `/bug-solver` | Fixes broken behavior |
| TD-27 (fix broken pagination) | `/bug-solver` | Fixes broken behavior |
| TD-2, TD-3 (verify constraints) | `/bug-solver` | Verify/fix constraints |
| TD-16 (fix cascade deletion) | `/bug-solver` | Fixes data integrity bug |
| TD-59 (N+1 query fix) | `/bug-solver` | Fixes performance bug |
| TD-61 (CV description edit) | `/feat-plan` | New endpoint + UI |
| FF-10 (catalog metadata edit) | `/feat-plan` | New feature |
| TD-12 (catalog re-pinning) | `/feat-plan` | New capability |
| FF-4 (edit CV pins) | `/feat-plan` | New feature |
| TD-60 (replace prompt()) | `/bug-solver` | Fixes UX bug |
| TD-57 (move files) | Neither — pure refactor, just run tests | File move only |
| TD-53/54/55 (DRY) | Neither — pure refactor, just run tests | Code cleanup |
| TD-49/50/51 (hook fixes) | `/bug-solver` | Fixes broken behavior |
| TD-52 (modal refactor) | Neither — pure refactor, just run tests | Architecture cleanup |

### Phase Completion Checklist (MANDATORY for each phase)

Before committing any phase, ALL of these must be done:
- [ ] All tests pass (backend + browser + system + live)
- [ ] `/coverage-test` skill invoked with arithmetic verification
- [ ] Quality review completed (3 code reviewers dispatched)
- [ ] Quality review findings presented to human and approved
- [ ] Coverage report updated with measured numbers (covered/total)
- [ ] Deploy to Kind cluster and verify
- [ ] Human approval received

**Branch:** `009-foundation-cleanup`

**Key commands:**
- Backend tests: `make -f /home/jsalomon/src/pc-asset-hub/Makefile test-backend`
- Browser tests: `make -f /home/jsalomon/src/pc-asset-hub/Makefile test-browser`
- System tests: `make -f /home/jsalomon/src/pc-asset-hub/Makefile test-system`
- Live tests: `make -f /home/jsalomon/src/pc-asset-hub/Makefile test-live`
- Deploy: `./scripts/kind-deploy.sh rebuild "kubectl --context kind-assethub"`
- Coverage (backend): `go test ./internal/... -count=1 -coverprofile=coverage.out && bash scripts/go-coverage-table.sh coverage.out`
- Coverage (UI): `cd ui && npx vitest run --config vitest.browser.config.ts --coverage`
- Uncovered new lines (Go): `bash scripts/uncovered-new-lines.sh --head`
- Uncovered new lines (UI): `bash scripts/uncovered-new-lines-ui.sh --head`

**Important patterns:**
- Update DTOs with optional fields MUST use `*string` pointers, not `string` — see LTM memory `feedback_omitted_fields_erase_data.md`
- Coverage `json` reporter required in `vitest.browser.config.ts` — see LTM memory `feedback_coverage_json_reporter.md`
- Run `scripts/ui-coverage-check-file.sh` to verify new files appear in coverage data

---

## Phase 1: Data Integrity & API Hardening (Commit 1)

### Task 1: TD-62 — Audit and fix UpdateEntityType for omitted-field data loss ✅

- [x] Changed `UpdateEntityTypeRequest.Description` from `string` to `*string` in `dto.go`
- [x] Handler checks `req.Description != nil`; if nil and `etvRepo != nil`, fetches current description from latest version
- [x] Service signature unchanged (`string`) — resolved in handler
- [x] Tests: `TestTD62_UpdatePreservesDescriptionWhenOmitted`, `TestTD62_UpdateOmittedDescriptionLatestVersionError`
- [x] All backend tests pass

### Task 2: TD-27 — Fix ListContainedInstances pagination ✅

- [x] Extracted `parseListParams(c)` helper from `ListInstances` (DRY, not copy)
- [x] `ListContainedInstances` now uses `parseListParams(c)` instead of hardcoded `ListParams{Limit: 20}`
- [x] Tests: `TestTD27_ListContainedWithPagination`, `TestTD27_ListContainedWithFilter`
- [x] All backend tests pass

### Task 3: TD-2 — Catalog version label uniqueness ✅

- [x] Verified: GORM model has `uniqueIndex` on `VersionLabel`, DB constraint enforced
- [x] Test: `TestTD2_DuplicateCatalogVersionLabel` confirms conflict error propagates
- [x] No code change needed — constraint already works

### Task 4: TD-3 — Association name uniqueness ✅

- [x] Verified: GORM model has `uniqueIndex:idx_assoc_version_name` on `(EntityTypeVersionID, Name)`
- [x] Existing test `TestTE107_CreateAssociationDuplicateName` already covers this
- [x] Service-level validation also present (checks `ListByVersion` for name collision before DB insert)
- [x] No code change needed

### Task 5: TD-16 — Fix catalog deletion cascade ✅

- [x] Added `DeleteByInstanceID` to `InstanceAttributeValueRepository` interface, mock, and GORM impl
- [x] `CatalogService.Delete` now: `ListByCatalog` → per-instance `linkRepo.DeleteByInstance` + `iavRepo.DeleteByInstanceID` → `DeleteByCatalogID` → `catalogRepo.Delete`
- [x] Nil guards on `iavRepo`/`linkRepo` preserve backward compatibility
- [x] Tests: `TestTD16_DeleteCascadesIAVsAndLinks`, `TestTD16_DeleteCascadeListByCatalogError`, `TestTD16_DeleteCascadeLinkDeleteError`, `TestTD16_DeleteCascadeIAVDeleteError`
- [x] Integration test: `TestTD16_DeleteByInstanceID` (real SQLite)
- [x] All backend tests pass

### Task 6: TD-59 — Fix N+1 query in entity type list ✅

- [x] Added `GetLatestByEntityTypes(ctx, []string) (map[string]*EntityTypeVersion, error)` to interface, mock, GORM impl
- [x] GORM impl uses `WHERE entity_type_id IN ? ORDER BY version DESC` with app-level dedup (SQLite-compatible)
- [x] Handler `List` collects all IDs, makes single batch call instead of N individual calls
- [x] Tests: `TestTD59_ListUsesGetLatestByEntityTypes` (handler), `TestTD59_GetLatestByEntityTypes` (integration), error branch via `closedDB`
- [x] Updated existing `TestT23_14_ListEntityTypesWithDescription` to use batch mock
- [x] All backend tests pass

### Task 7: Phase 1 Completion ✅

- [x] **Step 1:** Backend tests — all 1329 tests pass (16 packages)
- [x] **Step 1b:** Browser tests — 730/730 pass
- [x] **Step 2:** `/coverage-test` — 0 uncovered new lines (verified by arithmetic check against main). Per-file deltas: no regressions. `service/operational` 99.4% → 99.8%. Overall backend: 97.2% (3531/3632). Report updated in `docs/coverage-report.md`.
- [x] **Step 3:** Code review — 2 findings. I-1 (nil guard on etvRepo in Update handler) fixed. I-2 (cascade O(n) performance) acknowledged as acceptable tech debt.
- [x] **Step 4:** Review fixes applied with tests
- [x] **Step 5:** Deployed to Kind cluster
- [x] **Step 6:** System tests 30/30 pass. Live tests all pass (228+ tests across 7 scripts + 11 new Phase 1 tests).
- [x] **Step 7:** `docs/coverage-report.md` updated with measured numbers
- [x] **Step 8:** Human approved
- [x] **Step 9:** Committed: `d697857` — `"Resolve TD-62, TD-27, TD-59, TD-16, audit TD-2/TD-3: Phase 1 data integrity fixes"`

---

## Phase 2: Missing CRUD Capabilities (Commit 2)

### Task 8: TD-61 — Add CV description update endpoint

**Files:**
- Modify: `internal/service/meta/catalog_version_service.go` — add `UpdateDescription(ctx, id, description)` method
- Modify: `internal/api/meta/catalog_version_handler.go` — add `Update` handler
- Modify: `internal/api/dto/dto.go` — add `UpdateCatalogVersionRequest` DTO with `Description *string`
- Modify: `internal/api/meta/catalog_version_handler.go` — register `PUT /catalog-versions/:id` route
- Modify: `ui/src/api/client.ts` — add `catalogVersions.update` method
- Modify: `ui/src/pages/meta/CatalogVersionDetailPage.tsx` — add inline edit for description
- Test: Backend handler + service tests
- Test: Browser test for CV detail page edit

- [ ] **Step 1:** Write service test: `UpdateDescription` changes description.
- [ ] **Step 2:** Implement service method.
- [ ] **Step 3:** Write handler test: `PUT /catalog-versions/:id` with `{"description":"new"}` returns 200.
- [ ] **Step 4:** Implement handler + DTO + route registration.
- [ ] **Step 5:** Add `catalogVersions.update` to `client.ts`.
- [ ] **Step 6:** Add inline edit button on CV detail page (same pattern as EntityTypeDetailPage).
- [ ] **Step 7:** Write browser test for edit button + API call.
- [ ] **Step 8:** Run all tests.

### Task 9: FF-10 — Add catalog metadata edit endpoint

**Files:**
- Modify: `internal/service/operational/catalog_service.go` — add `UpdateMetadata(ctx, name, description)` or extend existing update
- Modify: `internal/api/operational/catalog_handler.go` — add `Update` handler
- Modify: `internal/api/dto/dto.go` — add `UpdateCatalogRequest` DTO
- Modify: route registration
- Modify: `ui/src/api/client.ts` — add `catalogs.update` method
- Modify: `ui/src/pages/operational/CatalogDetailPage.tsx` — add editable description
- Test: Backend + browser tests

- [ ] **Step 1:** Write service test: update catalog description.
- [ ] **Step 2:** Implement service method.
- [ ] **Step 3:** Write handler test: `PUT /catalogs/:name` with `{"description":"new"}`.
- [ ] **Step 4:** Implement handler + DTO + route. Note: name changes require DNS-label validation + uniqueness check. Published catalogs: only SuperAdmin can edit, name change not allowed while published. **Ensure `RequireWriteAccess` + `RequireCatalogAccess` middleware applied to the PUT route.**
- [ ] **Step 5:** Add UI edit control on catalog detail page.
- [ ] **Step 6:** Write browser tests.
- [ ] **Step 7:** Add live test cases to `scripts/test-descriptions.sh`.
- [ ] **Step 8:** Run all tests.

### Task 10: TD-12 — Catalog re-pinning (change CV)

> **Depends on Task 9** — extends `UpdateCatalogRequest` DTO and `PUT /catalogs/:name` route created in Task 9.

**Files:**
- Modify: `internal/service/operational/catalog_service.go` — add re-pin logic
- Modify: `internal/api/operational/catalog_handler.go` — accept `catalog_version_id` in update
- Modify: `ui/src/pages/operational/CatalogDetailPage.tsx` — add CV picker

- [ ] **Step 1:** Write service test: re-pin catalog to different CV.
- [ ] **Step 2:** Implement: validate new CV exists, update `catalog_version_id`, reset validation status to `draft`.
- [ ] **Step 3:** Write handler test.
- [ ] **Step 4:** Implement handler (extend UpdateCatalogRequest with optional `catalog_version_id *string`).
- [ ] **Step 5:** Add UI — CV selector dropdown on catalog detail page.
- [ ] **Step 6:** Write browser test.
- [ ] **Step 7:** Run all tests.

### Task 11: FF-4 — Edit Catalog Version pins (add/remove)

**Files:**
- Modify: `internal/service/meta/catalog_version_service.go` — add `AddPin`, `RemovePin` methods
- Modify: `internal/api/meta/catalog_version_handler.go` — add pin CRUD handlers
- Modify: route registration — `POST /catalog-versions/:id/pins`, `DELETE /catalog-versions/:id/pins/:pin-id`
- Modify: `ui/src/pages/meta/CatalogVersionDetailPage.tsx` — add/remove pin controls on BOM tab

- [ ] **Step 1:** Write service test: add pin to existing CV.
- [ ] **Step 2:** Implement `AddPin` — validate entity type version exists, check no duplicate.
- [ ] **Step 3:** Write service test: remove pin from CV.
- [ ] **Step 4:** Implement `RemovePin`.
- [ ] **Step 5:** Write handler tests for both endpoints.
- [ ] **Step 6:** Implement handlers + route registration.
- [ ] **Step 7:** Add UI controls (add button + remove button per pin on BOM tab).
- [ ] **Step 8:** Write browser tests.
- [ ] **Step 9:** Add live test cases.
- [ ] **Step 10:** Run all tests.

### Task 12: Phase 2 Completion (follow Phase Completion Checklist)

- [ ] **Step 1:** Run full test suite: `make test-backend && make test-browser && make test-system && make test-live`
- [ ] **Step 2:** Invoke `/coverage-test` skill — delete old files, run fresh, arithmetic check, per-file deltas
- [ ] **Step 3:** Dispatch 3 parallel code-reviewer agents — present findings, wait for human approval
- [ ] **Step 4:** Fix any quality review issues (with TDD)
- [ ] **Step 5:** Deploy to Kind and run live + system tests
- [ ] **Step 6:** Update `docs/coverage-report.md` with measured numbers
- [ ] **Step 7:** Get human approval
- [ ] **Step 8:** Commit: `"Resolve TD-61, FF-10, TD-12, FF-4: Missing CRUD capabilities"`

---

## Phase 3: UI Polish & Code Quality (Commit 3)

### Task 13: TD-60 — Replace enum window.prompt() with inline edit

**Files:**
- Modify: `ui/src/pages/meta/EnumDetailPage.tsx` — replace `window.prompt()` with inline TextInput (same pattern as EntityTypeDetailPage)
- Modify: `ui/src/pages/meta/EnumDetailPage.browser.test.tsx` — update tests

- [ ] **Step 1:** Write browser test: click Edit description → inline input appears → type → Save → API called.
- [ ] **Step 2:** Run test — expect FAIL.
- [ ] **Step 3:** Replace `window.prompt()` block with inline edit state + TextInput + Save/Cancel buttons.
- [ ] **Step 4:** Run test — expect PASS.
- [ ] **Step 5:** Remove the `window.prompt` mock tests, replace with inline edit tests.

### Task 14: TD-57 — Move CatalogDetailPage and CatalogListPage to pages/meta/

**Files:**
- Move: `ui/src/pages/operational/CatalogDetailPage.tsx` → `ui/src/pages/meta/CatalogDetailPage.tsx`
- Move: `ui/src/pages/operational/CatalogDetailPage.browser.test.tsx` → `ui/src/pages/meta/CatalogDetailPage.browser.test.tsx`
- Move: `ui/src/pages/operational/CatalogListPage.tsx` → `ui/src/pages/meta/CatalogListPage.tsx`
- Move: `ui/src/pages/operational/CatalogListPage.browser.test.tsx` → `ui/src/pages/meta/CatalogListPage.browser.test.tsx`
- Modify: `ui/src/App.tsx` — update import paths

- [ ] **Step 1:** Move files using `git mv`.
- [ ] **Step 2:** Update all import paths in App.tsx and test files.
- [ ] **Step 3:** Run full browser tests — expect all PASS (no behavior change).

### Task 15: TD-53/54/55 — Diagram component DRY

**Files:**
- Create: `ui/src/components/DiagramTabContent.tsx` — shared loading/error/empty/diagram renderer
- Modify: `ui/src/pages/meta/CatalogVersionDetailPage.tsx` — use `useCatalogDiagram` hook + `DiagramTabContent`
- Modify: `ui/src/pages/meta/CatalogDetailPage.tsx` — use `DiagramTabContent`
- Modify: `ui/src/pages/operational/OperationalCatalogDetailPage.tsx` — use `DiagramTabContent`
- Modify: `ui/src/components/EntityTypeDiagram.tsx` — extract `buildEdgeClickData` helper (TD-55)

- [ ] **Step 1:** Create `DiagramTabContent` component accepting `{diagramData, diagramLoading, diagramError}`.
- [ ] **Step 2:** Replace inline diagram tab JSX in all 3 pages with `<DiagramTabContent>`.
- [ ] **Step 3:** Refactor `CatalogVersionDetailPage` to use `useCatalogDiagram` hook instead of inline loading (TD-54).
- [ ] **Step 4:** Extract `buildEdgeClickData(data)` helper in EntityTypeDiagram (TD-55).
- [ ] **Step 5:** Run all browser tests — expect PASS.

### Task 16: TD-49/50/51 — Instance detail hook fixes

**Files:**
- Modify: `ui/src/hooks/useInstanceDetail.ts` — add `role` param, call `setAuthRole` (TD-49)
- Modify: `ui/src/hooks/useInstanceDetail.ts` — `selectInstance` re-fetches by ID (TD-50)
- Modify: `ui/src/pages/meta/CatalogDetailPage.tsx` — fix `onRemoveParent` error handling (TD-51)
- Modify: `ui/src/pages/meta/CatalogDetailPage.tsx` — pass `role` to `useInstanceDetail`

- [ ] **Step 1:** Write hook test: `useInstanceDetail` calls `setAuthRole` before API requests.
- [ ] **Step 2:** Add `role` parameter to hook, call `setAuthRole(role)` at start of `selectInstance`.
- [ ] **Step 3:** Change `selectInstance` to accept instance ID (not full object), re-fetch inside.
- [ ] **Step 4:** Update all callers to pass `instance.id` instead of `instance`.
- [ ] **Step 5:** Fix `onRemoveParent` catch block: `setSetParentError(e instanceof Error ? e.message : 'Failed')`.
- [ ] **Step 6:** Run all browser tests.

### Task 17: TD-52 — Modal data-loading internalization

**Files:**
- Modify: `ui/src/pages/meta/CatalogDetailPage.tsx` — move data loading into modals
- Modify: `ui/src/components/AddChildModal.tsx` — accept `catalogName`, `pins`, load own data
- Modify: `ui/src/components/LinkModal.tsx` — accept minimal props, load own data
- Modify: `ui/src/components/SetParentModal.tsx` — accept minimal props, load own data

- [ ] **Step 1:** Identify data-loading code in CatalogDetailPage that serves modals (~60 lines).
- [ ] **Step 2:** Move `loadAvailableInstances` into AddChildModal.
- [ ] **Step 3:** Move `loadLinkTargetInstances` into LinkModal.
- [ ] **Step 4:** Move `loadParentInstances` into SetParentModal.
- [ ] **Step 5:** Update CatalogDetailPage to pass minimal props (catalogName, pins, schemaAssocs).
- [ ] **Step 6:** Run all browser tests — expect PASS (no behavior change).

### Task 18: Phase 3 Completion (follow Phase Completion Checklist)

- [ ] **Step 1:** Run full test suite: `make test-backend && make test-browser && make test-system && make test-live`
- [ ] **Step 2:** Invoke `/coverage-test` skill — delete old files, run fresh, arithmetic check, per-file deltas
- [ ] **Step 3:** Dispatch 3 parallel code-reviewer agents — present findings, wait for human approval
- [ ] **Step 4:** Fix any quality review issues (with TDD)
- [ ] **Step 5:** Deploy to Kind and run live + system tests
- [ ] **Step 6:** Update `docs/coverage-report.md` with measured numbers
- [ ] **Step 7:** Get human approval
- [ ] **Step 8:** Commit: `"Resolve TD-49/50/51/52/53/54/55/57/60: UI polish and code quality"`

---

## Post-Plan: Final PR

After all 3 phases committed:
- [ ] Update `docs/session-start-prompt.txt` for next session
- [ ] Push branch and create PR
- [ ] Run `/superpowers:requesting-code-review` on the full PR
- [ ] Run security review
- [ ] Get human approval to merge
