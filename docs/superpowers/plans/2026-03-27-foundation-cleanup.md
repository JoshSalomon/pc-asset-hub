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

### Task 8: US-49 — CV metadata edit (label + description) ✅

- [x] Added `Update` to CatalogVersionRepository interface, mock, GORM impl
- [x] Added `GetByID` to CatalogVersionPinRepository interface, mock, GORM impl
- [x] Service: `UpdateCatalogVersion(ctx, id, *versionLabel, *description)` — label uniqueness, `*string` pattern
- [x] Handler: `PUT /catalog-versions/:id` with `UpdateCatalogVersionRequest` DTO (requireRW)
- [x] UI: Inline edit for label + description on CatalogVersionDetailPage
- [x] Client: `catalogVersions.update(id, data)`
- [x] Tests: 8 service + 5 handler + 3 integration + 13 browser + 2 client

### Task 9: US-50/US-51 — Catalog metadata edit + re-pinning ✅

- [x] Added `Update` to CatalogRepository interface, mock, GORM impl
- [x] Service: `UpdateMetadata(ctx, name, *newName, *description, *catalogVersionID, role)` — DNS-label validation, published guards, validation reset
- [x] Handler: `PUT /catalogs/:catalog-name` with `UpdateCatalogRequest` DTO (requireRW + requireCatalogAccess)
- [x] UI: Inline edit for description, CV selector dropdown on CatalogDetailPage
- [x] Client: `catalogs.update(name, data)`
- [x] Tests: 15 service + 7 handler + 2 integration + 8 browser + 2 client

### Task 10: US-52 — CV pin editing (add/remove) ✅

- [x] Service: `AddPin(ctx, cvID, entityTypeVersionID)` — validates ETV, 409 on duplicate
- [x] Service: `RemovePin(ctx, cvID, pinID)` — validates ownership
- [x] Handlers: `POST /catalog-versions/:id/pins`, `DELETE /catalog-versions/:id/pins/:pin-id` (requireRW)
- [x] UI: Add Pin button + modal, Remove button per pin on BOM tab
- [x] Client: `catalogVersions.addPin(id, etvId)`, `catalogVersions.removePin(id, pinId)`
- [x] Tests: 11 service + 6 handler + 2 integration + browser tests

### Task 11: Phase 2 Completion ✅

- [x] **Step 1:** Backend tests — all 1392 pass (16 packages)
- [x] **Step 1b:** Browser tests — 768/768 pass (fixed PatternFly select `option`→`menuitem` role, Edit button `exact: true` disambiguation)
- [x] **Step 2:** Coverage — 0 uncovered new Go lines. UI: 8 uncovered new lines (all guard returns + PatternFly internals, approved). Backend: 97.4% (3680/3779). UI: 93.3% (2358/2527).
- [x] **Step 3:** Quality review — completed. Findings:
  - I-1 (description-only edit resets validation on published catalogs): FIXED — description changes no longer set `changed=true`, validation stays valid, CR still synced
  - I-2 (GetByLabel/GetByName DB error silently swallowed): FIXED — added `!domainerrors.IsNotFound(err)` propagation (TDD: RED verified, GREEN verified)
  - I-3 (double write-protection on DELETE): Benign — defense in depth, documented
  - I-4 (rapid CV selector click): FIXED — added `cvListLoading` guard
  - I-5 (aria-label case inconsistency): FIXED — `"Version label"` → `"Version Label"`
  - TD-66 already tracked for role mapping duplication
  - PRD US-43 updated to document description-only edit behavior on published catalogs
- [x] **Step 4:** Quality review fixes applied with TDD (4 new tests: GetByLabelDBError, GetByNameDBError, DescOnlyNoValidationReset, CatalogUpdate_DuplicateName)
- [x] **Step 5:** Deployed to Kind. Live tests: 239/239 pass (8 scripts). System tests: 30/30 pass.
- [x] **Step 6:** `docs/coverage-report.md` updated with measured numbers
- [x] **Step 7:** Human approved
- [x] **Step 8:** Committed with US-53 + TD-69: `ee77d65`

### Task 12: US-53 — CV Pin Management ✅

- [x] **12a:** Fix AddPin entity type duplicate check — now checks entity TYPE ID, not ETV ID (TDD: RED verified, GREEN verified)
- [x] **12b:** UpdatePin service + handler — `PUT /catalog-versions/:id/pins/:pin-id` with entity type mismatch validation (TDD: 5 service + 5 handler tests)
- [x] **12c:** BOM inline version dropdown — PatternFly Select per pin row for Admin+, lazy-loads versions with caching
- [x] **12d:** Add Pin modal filtering — excludes already-pinned entity types from dropdown
- [x] **12e:** Live tests — 3 new tests in `scripts/test-descriptions.sh` (UpdatePin, entity type mismatch 400, duplicate entity type 409)
- [x] **Coverage:** 0 uncovered new Go lines (verified by arithmetic + 4 error path tests added + 1 integration test). Backend: 97.4% (3716/3815).
- [x] **Quality review:** Complete. I-1 (pin Update integration test) already fixed. I-2 (version cache) benign. I-3 (toggle race) low severity. S-1 (same-version test) added. TD-67 added to PRD for S-4 (validate tag enforcement).
- [x] **Browser tests:** 777/777 pass. Fixed PF6 Select-in-Modal aria-hidden issue (extracted PinEntityTypeSelect/PinVersionSelect wrappers). Fixed test data (added unpinned entity types for Add Pin tests).
- [x] **Deploy + live tests:** All 242 live tests pass (28 in test-descriptions.sh including 4 TD-69 stage guard tests). System tests 30/30 pass.

### Task 12f: TD-69 — Pin Editing Stage Guards ✅

- [x] Service: `checkPinEditAllowed(cv, role)` — production blocked, testing SuperAdmin only, development RW+
- [x] Handler: `mapRole` helper + role extraction for AddPin/RemovePin/UpdatePin
- [x] UI: `canEditPins` flag gates Add Pin / Remove / version dropdown visibility
- [x] Tests: 7 service + 3 handler + 4 live tests
- [x] Committed with Phase 2: `ee77d65`

### Phase 2 Final Commit

- **Commit:** `ee77d65` — "Resolve US-49, US-50, US-51, US-52, US-53, TD-69"
- **Followed by:** `b652df7` — "Extract technical debt log from PRD to docs/td-log.md (TD-64)"
- **Tests:** 1416 backend, 777 browser, 30 system, 242 live — all pass
- **Coverage:** backend 97.4% (3737/3838), UI 93.4% (2392/2562)

### Remaining Steps for Phase 2 + US-53 (to be completed in non-isolated environment)

**What was done in isolated environment:**
- All backend code: service, handler, DTO, repo interface + mock + GORM impl
- All frontend code: client.ts `updatePin`, BOM version dropdown, Add Pin filtering
- All backend tests: 17 service + 10 handler + 1 integration = 28 new tests (all pass)
- All browser tests written: 6 new CV detail page tests + 1 client test (compile verified, cannot run without Playwright)
- Live tests written: 3 new tests in `scripts/test-descriptions.sh`
- Coverage verified: 0 uncovered new Go lines, 97.4% overall
- Quality review completed with all issues addressed
- Coverage report and plan updated

**What remains (non-isolated environment):**

1. Run browser tests: `make test-browser`
   - Expect 6 new US-53 tests in `CatalogVersionDetailPage.browser.test.tsx`:
     - T-28.14: BOM version dropdown visible for Admin
     - T-28.16: Selecting version calls updatePin API
     - T-28.18: RO sees plain text, not dropdown
     - T-28.19: Add Pin modal filters pinned entity types
     - Update pin version error shows alert
   - Expect 1 new test in `client.browser.test.ts`:
     - T-28.21: catalogVersions.updatePin sends PUT
   - If PatternFly Select uses `option` instead of `menuitem` role, update test selectors (known issue from Phase 2)
2. Deploy: `./scripts/kind-deploy.sh rebuild "kubectl --context kind-assethub"`
3. Run system tests: `make test-system`
4. Run live tests: `make test-live`
   - `scripts/test-descriptions.sh` has 3 new US-53 tests:
     - UpdatePin changes pinned version
     - UpdatePin entity type mismatch returns 400
     - AddPin duplicate entity type returns 409
5. If browser tests fail on version dropdown interaction, likely causes:
   - PatternFly `Select` role mismatch (`option` vs `menuitem`) — fix: change `getByRole('option', ...)` to `getByRole('menuitem', ...)`
   - Version dropdown not opening — check `aria-label` matches `"Version for Model"`
   - Mock `versions.list` not returning data — verify mock setup in `beforeEach`
6. If all pass, commit: `"Resolve US-49, US-50, US-51, US-52, US-53: Missing CRUD capabilities and pin management"`

### Previous Task 8: TD-61 — Add CV description update endpoint

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

## Phase 2b: CV Pin Management (US-53) — Commit with Phase 2

**Skill:** `/feat-plan` — new feature (change pin version, entity type uniqueness constraint, UI filtering)

**Status:** Phase 1 (PRD) ✅, Phase 2 (HL test plan) ✅, Phase 3 (detailed test plan) ✅

### Task 12a: US-53 — Fix AddPin duplicate entity type check

**Files:**
- Modify: `internal/service/meta/catalog_version_service.go` — AddPin checks entity type ID (via ETV lookup), not just ETV ID
- Test: `internal/service/meta/enum_catalog_service_test.go`

- [ ] **Step 1:** Write test: AddPin with V2 of "Server" when V1 is pinned → 409.
- [ ] **Step 2:** Run test — RED (current code only checks ETV ID match).
- [ ] **Step 3:** Fix AddPin to resolve entity type ID from each existing pin's ETV and compare.
- [ ] **Step 4:** Run test — GREEN.

### Task 12b: US-53 — Add UpdatePin endpoint (change version)

**Files:**
- Modify: `internal/service/meta/catalog_version_service.go` — add `UpdatePin(ctx, cvID, pinID, newETVID)` method
- Modify: `internal/api/meta/catalog_version_handler.go` — add `UpdatePin` handler + `PUT /catalog-versions/:id/pins/:pin-id` route
- Modify: `internal/api/dto/dto.go` — add `UpdatePinRequest` DTO
- Modify: `ui/src/api/client.ts` — add `catalogVersions.updatePin(cvId, pinId, etvId)`
- Test: Service + handler + integration + client tests

- [ ] **Step 1:** Write service test: UpdatePin changes ETV, validates same entity type.
- [ ] **Step 2:** Implement service method.
- [ ] **Step 3:** Write handler test: `PUT /catalog-versions/:id/pins/:pin-id` returns 200.
- [ ] **Step 4:** Implement handler + DTO + route.
- [ ] **Step 5:** Write test: entity type mismatch returns 400.
- [ ] **Step 6:** Add `catalogVersions.updatePin` to `client.ts`.

### Task 12c: US-53 — BOM tab inline version dropdown

**Files:**
- Modify: `ui/src/pages/meta/CatalogVersionDetailPage.tsx` — version column becomes dropdown for Admin+, loads versions on open, calls updatePin on select
- Test: Browser tests for dropdown, version change, RO plain text

- [ ] **Step 1:** Write browser test: version column is dropdown for Admin.
- [ ] **Step 2:** Replace plain text version with PatternFly Select dropdown.
- [ ] **Step 3:** Write browser test: selecting version calls updatePin API.
- [ ] **Step 4:** Implement onSelect handler.
- [ ] **Step 5:** Write browser test: RO sees plain text.

### Task 12d: US-53 — Add Pin modal filters to unpinned entities

**Files:**
- Modify: `ui/src/pages/meta/CatalogVersionDetailPage.tsx` — filter entity type list in Add Pin modal
- Test: Browser tests for filtering

- [ ] **Step 1:** Write browser test: Add Pin modal does not show already-pinned entity types.
- [ ] **Step 2:** Filter `entityTypes` list by excluding those whose ID matches any `pin.entity_type_id`.
- [ ] **Step 3:** Run test — GREEN.

### Task 12e: US-53 — Live tests

- [ ] **Step 1:** Add live test cases to `scripts/test-descriptions.sh`: add pin of same entity type different version → 409, update pin version, verify updated version in response.
- [ ] **Step 2:** Run live tests.

---

## Phase 2c: Security Fix — Published Catalog Validate Bypass (Commit with Phase 2)

**Skill:** `/bug-solver` — fixes authorization bypass (CWE-863)

**Severity:** High — RW user can mutate published catalog's `validation_status` without SuperAdmin role.

**Root cause:** `POST /:catalog-name/validate` route lacks `RequireWriteAccess` middleware. The PRD (US-43) incorrectly classified validation as a "read operation", but `CatalogValidationService.Validate` calls `catalogRepo.UpdateValidationStatus()` which WRITES to the database.

**Impact:**
- RW user can flip a published catalog from `valid` to `invalid` (or vice versa)
- Catalog CR in K8s gets desynchronized status, potentially triggering operator alerts
- Undermines trust model: consumers relying on published catalog `valid` status

### Task 12f: Fix validate endpoint write protection

**Option B (split approach):** Allow RW users to RUN validation and SEE results on published catalogs, but only persist the status update if the user has write access. This preserves the diagnostic utility while closing the authorization gap.

**Files:**
- Modify: `internal/api/operational/catalog_handler.go` — `ValidateCatalog` handler checks `RequireWriteAccess` OR splits read/write
- Modify: route registration — add `writeMiddleware` to validate route
- Test: Handler + live tests

- [ ] **Step 1:** Write test: RW user calls validate on published catalog → status NOT updated (or returns 403).
- [ ] **Step 2:** Run test — RED (current code allows RW to mutate status).
- [ ] **Step 3:** Apply fix: add `RequireWriteAccess` middleware to validate route, OR modify handler to skip `UpdateValidationStatus` when user lacks write access on published catalogs.
- [ ] **Step 4:** Run test — GREEN.
- [ ] **Step 5:** Write live test: verify published catalog validate is blocked for RW user.
- [ ] **Step 6:** Update PRD US-43 to correct the "read operation" characterization.
- [ ] **Step 7:** Run all tests (backend + browser + live).

**Decision needed from human:** Which fix option?
- **Option A (strict):** Add `writeMiddleware` to validate route — RW users cannot validate published catalogs at all
- **Option B (split):** Return validation results but skip status write on published catalogs for non-SuperAdmin
- **Option C (document):** Accept risk, update PRD to explicitly allow RW status mutation on published catalogs

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
