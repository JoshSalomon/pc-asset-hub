# Project Memory

## Key Lessons Learned

See [session-001-lessons.md](session-001-lessons.md) for detailed lessons from the first major implementation session.
See [selinux-ltm-fix.md](selinux-ltm-fix.md) for fixing LTM permission errors (SELinux MCS categories).
See [feedback_coverage_json_reporter.md](feedback_coverage_json_reporter.md) for V8 coverage missing new files — requires `json` reporter, not just `json-summary`.
See [feedback_omitted_fields_erase_data.md](feedback_omitted_fields_erase_data.md) for Go API update DTOs: use `*string` not `string` for optional fields to prevent silent data erasure.
See mem_bf8081fc for FF-15 session lessons: Phase 6 before Phase 7, TDD discipline, arithmetic verification, deploy-before-done.
See mem_dbd8c2b8 for coverage agent dispatch: agents must write tests not just measure.

## Infrastructure

- **Kind cluster**: name `assethub`, kubectl context `kind-assethub`, uses podman (set `KIND_EXPERIMENTAL_PROVIDER=podman`)
- **Deploy**: `make -f /home/jsalomon/src/pc-asset-hub/Makefile deploy` (works from any directory)
- **All commands via Makefile** — use absolute path to avoid `cd` issues: `make -f /home/jsalomon/src/pc-asset-hub/Makefile <target>`
- Key targets: `deploy`, `test-backend`, `test-browser`, `test-system`, `test-live`, `test-all`, `coverage-backend`, `coverage-browser`
- **After laptop crash**: `podman start assethub-control-plane`, wait 10s, then check pods — API server may need pod deletion if CrashLoopBackOff (started before Postgres was ready)
- **Ports**: API `localhost:30080`, UI `localhost:30000`

## Skills

- `feat-plan` — Phases 1-3: requirements, HL test plan, detailed test plan. After approval, asks: continue with feat-dev or write handoff file for separate agent.
- `feat-dev` — Phases 4-8: TDD implementation, deploy, quality review, coverage, documentation. Invoked by feat-plan or started from a handoff file.
- `bug-solver` — end-to-end bug resolution
- `coverage-report` — runs on Haiku, orchestrates coverage-generate + coverage-review loop
- `coverage-review` — runs on Sonnet, strict first-review requiring detailed justification for every uncovered line not in allowed exceptions

## Upcoming

- [QA Review Phase plan](project_qa_review_phase.md) — 6-stage test review plan at `.claude/plans/piped-prancing-moore.md`
- FF-15 (Export Plugins) — IMPLEMENTED. Branch `019-export-plugins`. MCP Gateway exporter with VirtualServer instance selection. Design spec: `docs/superpowers/specs/2026-05-10-export-plugins-design.md`
- FF-6 (Operational UI Editing) — IMPLEMENTED. See PRD US-57/58/59.

## Architecture Patterns

- `GetContainmentGraph()` returns one edge per entity type version — deduplicate with `map[string]map[string]bool` to avoid duplicate children in tree
- Version snapshot endpoint resolves enum names and target entity type names in the service layer, not the handler
- Associations need both outgoing (ListByVersion) and incoming (ListByTargetEntityType) with direction metadata
- Bidirectional associations show as "references (mutual)" from both sides — they are symmetric

## Catalog Implementation (Phases 1-3 complete)

- **Catalog** is a named data container pinned to a CatalogVersion. Instances belong to catalogs, not CVs directly.
- Operational API uses catalog **name** (not ID) in URLs: `/api/data/v1/catalogs/{catalog-name}`
- Catalog names must be DNS-label compatible: `^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`, max 63 chars
- Frontend uses `DATA_BASE_URL` env var (`VITE_DATA_API_BASE_URL`) for operational API, separate from meta `BASE_URL`
- PatternFly Tabs renders ALL tab content on mount (hidden via CSS). Call `setAuthRole(role)` in each page's load function.
- Design doc: `docs/plans/2026-03-09-catalog-implementation-design.md` — 7-phase plan (Phase 5 = Catalog RBAC added)
- **Instance CRUD**: `InstanceService` resolves catalog → CV → pins → entity type version → attributes. Attribute values carried forward on update. Draft mode allows missing required attrs.
- **Migration gotcha**: GORM AutoMigrate doesn't drop renamed columns. Use `information_schema.columns` (PG) or `PRAGMA table_info` (SQLite) to detect and `DROP COLUMN` in InitDB.
- **Browser test gotcha**: Use `getByRole('gridcell', ...)` not `getByText(...)` for PatternFly table cells.
- **Containment & Links**: `CreateContainedInstance` validates containment assoc in CV. Links validated against CV definitions. `cascadeDelete` cleans up links via `DeleteByInstance`. Duplicate link prevention via forward ref check.
- **Route gotcha**: Static segments (`/links`, `/references`) must be registered BEFORE parameterized `/:child-type` in Echo.
- **Live test script**: `scripts/test-containment-links.sh [API_BASE_URL]` — 18 parameterized tests
- TD-15 through TD-20, TD-26 through TD-28, TD-35 track deferred issues

## Phase 4: Catalog Data Viewer (complete)

- **Operational UI** served at `/operational` (same port 30000, path-based routing via nginx)
- Vite multi-entry build: `index.html` + `operational.html` with shared assets
- BrowserRouter needs `basename="/operational"` for subpath serving
- **Read-only** — no create/edit/delete actions (editing is FF-6)
- Tree endpoint: `GET /catalogs/{name}/tree` — builds in-memory from flat `ListByCatalog`
- EAV filtering: aliased JOINs on `instance_attribute_values`, service translates attr names→IDs
- Parent chain: walk up ParentInstanceID, reverse for root-first, with cycle guard
- **Live test script**: `scripts/test-data-viewer.sh [API_BASE_URL]` — 23 tests
- PRD additions: US-39 (catalog RBAC), US-40 (operational UI), FF-6 (operational editing)

## Phase 5: Catalog-Level RBAC (complete)

- `CatalogAccessChecker` interface — `CheckAccess(c, catalogName, verb) (bool, error)`
- `HeaderCatalogAccessChecker` always allows (dev mode); `SARCatalogAccessChecker` deferred to Phase C (OCP)
- `RequireCatalogAccess` middleware on catalog GET/DELETE routes AND instance group
- `CreateCatalog` checks access for the catalog name in the request body
- `ListCatalogs` filters through `FilterAccessible[T]` generic helper
- **Gotcha**: Must apply access check to ALL routes with catalog name — not just sub-routes
- TD-35 through TD-37 track deferred issues

## Phase 6: Catalog Validation (complete)

- `CatalogValidationService` validates required attrs, enum values, mandatory associations, containment consistency, unpinned entity types
- API: `POST /api/data/v1/catalogs/{name}/validate` — RW+ only
- Service types have NO json tags — DTO layer (`ValidationResultResponse`) converts in handler
- Pre-load associations into `assocCache` once before both mandatory-assoc and containment loops
- Shared UI: `useValidation` hook + `ValidationResults` component (both meta and operational pages)
- `cardinalityMinGE1` checks if cardinality string min >= 1 (e.g., "1", "1..n" → true)
- Bidirectional reverse-direction cardinality check deferred (only forward refs checked)
- **Live test script**: `scripts/test-validation.sh [API_BASE_URL]` — 9 tests
- **PRD fix**: Duplicate US-34 renamed to US-41 (catalog version creation in UI)
- **Browser test gotcha**: `getByText('valid')` matches "Validate" button substring — use `{ exact: true }`

## Phase 8: Copy & Replace Catalog (complete)

- `CatalogService.CopyCatalog` deep-clones instances, attrs, links, containment with ID remapping
- `CatalogService.ReplaceCatalog` atomically swaps names, transfers published state, syncs CR
- `TransactionManager` interface (`domain/repository/transaction.go`) — GORM impl uses context-propagated `*gorm.DB`
- Repos use `getDB(ctx, r.db)` to participate in transactions (catalog, instance, iav, link repos)
- `WithCopyDeps` + `WithTransactionManager` — functional options on CatalogService
- Routes: `/copy` and `/replace` registered BEFORE `/:catalog-name`
- Handler access checks: Copy checks source (get) + target (create); Replace checks both (update)
- DNS label regex extracted to `isValidDnsLabel()` in CatalogDetailPage.tsx
- **Live test script**: `scripts/test-copy-replace.sh` — 17 tests
- See [phase8-deferred-items.md](phase8-deferred-items.md) for low-severity deferred items

## Phase 7: Catalog Publishing (complete)

- `CatalogService.Publish/Unpublish` — Admin+ only, requires `valid` status
- `Published` bool + `PublishedAt` on Catalog model
- `RequireWriteAccess` middleware blocks non-SuperAdmin on published catalogs
- `K8sCatalogCRManager` — separate type (not methods on K8sCRManager) to avoid Go name collision
- `ReconcileCatalogStatus` pure function — increments DataVersion only on status transitions (NOT every reconcile)
- CV promotion returns `*PromoteResult` with `Warnings []CatalogWarning` for draft/invalid catalogs
- Publish rolls back DB if K8s CR creation fails
- `httpMethodToVerb` maps PUT/PATCH → "update"
- **Live test script**: `scripts/test-publishing.sh [API_BASE_URL]` — 14 tests
- **Coverage tool**: `scripts/analyze-coverage.sh [pattern]` — UI coverage analysis
- **Coverage lesson**: Page tests mock `api` → `client.ts` stays uncovered. Always add direct tests in `client.browser.test.ts` using `mockFetch`
- Design doc: Phase 8 (Copy & Replace Catalog) added; FF-7 (snapshots), FF-8 (copy & replace), FF-9 (multi-namespace publishing)
- PRD: US-42 (publish), US-43 (write protection), Catalog CR scoping resolved (namespaced)
