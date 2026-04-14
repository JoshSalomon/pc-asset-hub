# AI Asset Hub — Technical Debt Log

Items where the current implementation diverges from the intended behavior described in the PRD.

**Legend:** ✅ Resolved | 🔴 Critical | ⚠️ Important | 🔒 Security | 🐛 Bug

---

## 🔴 Critical / ⚠️ Important

| ID | Item | Current Behavior | Required Behavior |
|----|------|-----------------|-------------------|
| TD-1 | 🔴 Enum deletion safety | Enum delete checks if any attribute references it across all entity type versions (flat check) | Enum cannot be deleted if it is used by any attribute in a **used entity version**. A used entity version is defined as: (1) any entity type version pinned by a catalog version, or (2) the latest version of any entity type (which belongs to an implicit pre-production catalog). Unused historical versions that are not pinned by any CV and are not the latest version should not block deletion. |
| TD-10 | ⚠️ Mutable CVs in development mode | **PARTIALLY RESOLVED.** CV pins are now mutable via US-52 (add/remove/change version). Stage guards added via TD-69: production = blocked, testing = SuperAdmin only, development = RW+. Remaining: pin changes on demoted CVs should trigger re-validation of pinned catalogs. |
| TD-99 | 🐛 Containment cardinality `1..n` not enforced during catalog validation | A container entity type with a containment association of source cardinality `1` or `1..n` to a child type can pass validation even when no contained instances exist. For example, if MCP-Server contains Tools with cardinality `1..n`, creating an MCP-Server instance with zero Tools should fail validation — but it passes. The validation service checks mandatory associations for reference links but does not check containment parent-child counts against the source cardinality constraint. | Extend `CatalogValidationService` to count contained instances (children) per parent instance for each containment association. If the containment cardinality minimum is >= 1 (e.g., `1`, `1..n`) and the parent has zero children of the contained type, report a validation error. |

---

## Normal Priority

| ID | Item | Current Behavior | Required Behavior |
|----|------|-----------------|-------------------|
| TD-2 | Catalog version timestamp uniqueness | Two catalog versions can have the same `created_at` timestamp, causing non-deterministic sort order | `created_at` must be unique across catalog versions. The backend should enforce this (e.g., retry with a small delay if a timestamp collision is detected). This ensures deterministic sort order in the CV list (`ORDER BY created_at DESC`). |
| TD-3 | Association target+role uniqueness | No uniqueness constraint on (target entity type, target role) per source entity type version | Target entity type + target role must be unique per source entity type version. Empty target role is valid (one allowed per target). API should reject duplicates with 409 Conflict. |
| TD-5 | Version lineage tracking | Entity type versions are sequential integers with no parent tracking. Version 4 is created from version 3, but this relationship is not recorded. | Each entity type version should record which version it was derived from (`parent_version_id`). **Decision: deferred for v1.** |
| TD-6 | Duplicate DTO mapping logic | Attribute and Association model-to-DTO conversion is duplicated across handlers | Extract shared helper functions (e.g., `dto.ToAttributeResponse`, `dto.ToAssociationResponse`). |
| TD-7 | Bidirectional association removal only from source | A bidirectional association can only be removed from the entity type that created it (the source/outgoing side). | Since bidirectional associations are symmetric, the Remove button should be available from either side. |
| TD-8a | Extract shared EditAssociationModal component | Edit association modal is duplicated between `App.tsx` and `EntityTypeDetailPage.tsx` — ~110 lines of duplication | Extract into shared `ui/src/components/EditAssociationModal.tsx`. |
| TD-8b | Consolidate edit modal state into a single object | Diagram edit modal in `App.tsx` uses 12 separate `useState` calls for one form | Group into a single state object or move into the shared component from TD-8a. |
| TD-8c | Extract diagram data loading into a custom hook | `App.tsx` and `CatalogVersionDetailPage.tsx` both have `loadDiagramData` functions | Extract into `ui/src/hooks/useDiagramData.ts`. |
| TD-11 | Show mandatory associations in UI | Associations with cardinality `1` or `1..n` are not visually distinguished from optional ones | Show a mandatory indicator (e.g., `*` or bold) on the side of the association where cardinality starts with `1`. |
| TD-13 | Get catalog version by name | CV can only be retrieved by ID; no lookup by version_label | Add `GET /api/meta/v1/catalog-versions/by-name/:label` endpoint. |
| TD-14 | Catalogs using this CV | CatalogVersion detail page does not show which catalogs are pinned to it | Add a "Catalogs" section on the CatalogVersion detail page. |
| TD-18 | UI component props style inconsistency | Some components use `interface Props`, others use inline destructured types | Pick one convention and apply consistently. Low priority — cosmetic. |
| TD-19 | N+1 query in resolveEntityType | `InstanceService.resolveEntityType` iterates all CV pins and calls `etvRepo.GetByID` for each | Replace with a batch fetch or join query. Acceptable for now with 3-5 pins. |
| TD-26 | Extract shared instance creation logic (M5) | `CreateInstance` and `CreateContainedInstance` share ~70% of logic | Extract a private `createInstanceInternal` method. |
| TD-28 | Phase 3 code quality improvements (L1-L5, L7) | Multiple low-severity issues from quality review | Extract `refsToDTO` helper, clean up dead assignments, remove JSON tags from service types, batch link resolution, log validation status update failures. |
| TD-31 | Create new container from Set Container modal | The Set Container modal only allows selecting existing parent instances | Add a "Create New" mode to the Set Container modal. |
| TD-32 | Diagram: overlapping edges between same entity pair | Multiple associations between the same pair overlap into a single line | Add edge offset or curvature so multiple edges are visually distinct. |
| TD-33 | "Contained by" flickers UUID before showing parent name | Parent UUID briefly flashes before the async API call resolves the parent name | Include `parent_instance_name` in API response, or show spinner instead of UUID. |
| TD-37 | Reference direction unclear in tree browser detail panel | Directional associations show "directional" type but no arrow/direction | Show directional indicator: e.g., "my-server → gpt-4" for forward refs. |
| TD-38 | Entity type tab selector doesn't scale in meta catalog detail | Tabs overflow with 10+ entity types | Use scrollable tabs, sidebar, or dropdown selector. |
| TD-39 | CopyCatalog sequential instance creation doesn't scale | N individual `instRepo.Create` calls for catalog copy | Add `CreateBatch` method. Low priority — catalogs currently have <100 instances. |
| TD-40 | `SyncCR` uses unstructured logging | Uses `log.Printf` instead of structured logging | Replace with `slog.Warn` or project-standard logger. |
| TD-43 | Entity type list missing description in API response | `EntityTypeResponse` DTO does not include a `description` field | Add `description` by resolving the latest version's description. |
| TD-45 | Enum list page missing description column | Enums have no description field in the model | Either add a `description` field to the Enum model, or accept as-is. |
| TD-46 | No UI to edit entity type version description | Description is set at creation and carried forward on COW, no UI to change it | Add inline editable description field on entity type detail page. |
| TD-56 | Operational catalog viewer Overview tab removed — consider re-adding | Overview tab hidden; may be useful with stats/counts | Design useful Overview tab, or confirm not needed. |
| TD-63 | Inconsistent edit UX on entity type detail overview | Name uses modal, Description uses inline edit | Align both to inline edit pattern. |
| TD-65 | CV selector on catalog detail is Admin-only, not RW+ | RW users cannot change the CV | Show for RW+, or document as intentional. |
| TD-67 | `validate:"required"` struct tags not enforced by handler | No struct validator runs; empty fields reach service | Register `go-playground/validator` and call `c.Validate`. |
| TD-79 | Add Pin modal: version dropdown should default to latest version | Version dropdown empty after entity type selection | Auto-select highest version number. |
| TD-80 | No UI to rename unpublished catalogs | Catalog names are set at creation and cannot be changed from the UI. The backend `PUT /catalogs/{name}` supports updating the name, but the catalog detail page has no inline edit for it. | Add inline name edit on catalog detail page, guarded by `canMutate`. Validate DNS-label format client-side. Published catalogs should not be renamable (name is part of CR identity). |
| TD-81 | Missing Unlink/Remove buttons and obscure containment removal UX | Multiple gaps: (1) Schema page has Unlink on Forward References but NOT on Referenced By. (2) Operational data viewer has no Unlink buttons at all. (3) Containment removal is only possible from the contained entity (child) side via "Set Container" modal → then "Remove" inside the modal — this is obscure and non-discoverable. There is no way to remove a child from the container (parent) side. | Add Unlink to Referenced By tables on both pages. For containment: add a "Remove" action on children listed in the parent's Contained Instances section (calls `setParent` with empty parent). On the child side, add a visible "Remove Container" button directly in the detail panel (not hidden inside the Set Container modal). |
| TD-83 | 🐛 Diagram node selection frame has visual glitches | When selecting an entity type node in the diagram, the blue selection border does not properly enclose the full node. The frame appears clipped or misaligned relative to the node boundary, especially when the node has dynamic width (TD-72 fix). | The selection highlight is likely using the default PatternFly Topology node dimensions rather than the dynamically computed width. Ensure the node's `width`/`height` in the topology model match the actual rendered dimensions so the selection frame aligns correctly. |
| TD-84 | `handleUnlink` swallows errors silently | `CatalogDetailPage.tsx:177` has `catch { /* ignore */ }` on the link delete API call. The user gets no feedback if unlinking fails. Same class of bug as TD-51 (which was fixed for remove-parent). | Show an error alert in the catch block, e.g., `setError(e instanceof Error ? e.message : 'Failed to unlink')`. |
| TD-85 | `GetByIDs` return order not guaranteed | `EntityTypeVersionRepo.GetByIDs` uses `WHERE id IN ?` without `ORDER BY`. The returned slice order may not match the input `ids` order. Currently only used by `AddPin` (which doesn't need ordering), but future callers might expect order preservation. | Either sort results in Go to match input order, or keep the comment-only fix and accept non-deterministic order. If a caller needs ordering, add an `ORDER BY` or post-sort at that time. |
| TD-86 | `append(writeMiddleware, ...)` slice mutation risk in route registration | `RegisterCatalogRoutes` builds `writeMiddleware := append([]echo.MiddlewareFunc{requireRW}, writeGuards...)` then calls `append(writeMiddleware, requireCatalogAccess)...` for each route. Currently safe because capacity=2 forces a new allocation each time, but adding a third `writeGuard` would cause the underlying array to be shared — later appends would corrupt earlier routes' middleware chains. | Replace `append(writeMiddleware, requireCatalogAccess)...` with `slices.Concat(writeMiddleware, []echo.MiddlewareFunc{requireCatalogAccess})...` or pre-build per-route slices explicitly. Low priority — only triggers if a third writeGuard is added. |
| TD-87 | `App.system.test.ts` has inline helpers instead of importing from `test-helpers/system.ts` | `App.system.test.ts` defines its own `navigateToUI`, `apiCall`, `visible`, `hidden`, `setRole`, `cleanupTestData` inline. The shared `test-helpers/system.ts` module provides equivalent helpers used by all other system test files. | Refactor `App.system.test.ts` to import from `test-helpers/system.ts` and remove inline duplicates. Low priority — functional duplication only, no behavioral divergence. |
| TD-88 | System tests use magic timeout numbers | `visible()` defaults to 15s but tests override with various values (500, 1000, 2000, 5000, 10000, 25000). `waitForTimeout()` uses arbitrary delays (300–2000ms). No named constants, inconsistent across files. | Extract shared timeout constants (e.g., `NAVIGATION_TIMEOUT`, `LOAD_TIMEOUT`) in `test-helpers/system.ts`. Replace `waitForTimeout()` with deterministic waits (`waitForLoadState`, `waitForResponse`, `waitForFunction`) where possible. Low priority — tests pass reliably as-is. |
| TD-89 | Types tab: no sort or filter | TypeDefinitionListPage shows all type definitions in a flat list with no sorting controls or filtering (e.g., by base type, system/custom). The old EnumListPage also lacked this. | Add sort-by-name/base-type column headers and a base type filter dropdown (similar to entity type list filtering). |
| TD-90 | Catalog validation does not check type definition constraints (min/max, max_length, pattern) | Validation service checks required attrs, enum values, and cardinality but does NOT validate integer min/max, string max_length, string pattern, or other type definition constraints. Instance values that violate constraints pass validation. | Extend `CatalogValidationService` to resolve type definition constraints for each attribute and validate instance values against them (min/max for integer/number, max_length/pattern for string, valid URL format, valid ISO date, list element types, etc.). |
| TD-91 | Data viewer does not render type-aware value formatting | `InstanceDetailPanel` displays all attribute values as plain text regardless of base type. URLs are not clickable links, booleans show as "true"/"false" instead of "Yes"/"No", dates are not formatted, JSON is not collapsible. | Add type-aware rendering in `InstanceDetailPanel`: URLs as `<a>` tags, booleans as "Yes"/"No", dates formatted, JSON as collapsible/formatted block, lists as bullet points. Requires passing `base_type` to the panel. |
| TD-92 | Instance forms have no inline field-level validation warnings | `AttributeFormFields.tsx` renders type-appropriate controls but does NOT validate values against type definition constraints during editing. No warnings for: string exceeding max_length, string not matching pattern, integer/number out of min/max range, invalid URL format, invalid date format, invalid JSON syntax. Per US-54, warnings should be advisory only — form remains submittable (draft mode). | Add client-side constraint validation in `AttributeFormFields.tsx`. On blur or change, check the value against `attr.constraints` and show a PatternFly `helperText` warning (not error) on the FormGroup. Form submission is never blocked — warnings are advisory. |
| TD-93 | No option to rename a type definition | Type definitions have a `name` set at creation that cannot be changed. The `PUT /type-definitions/:id` endpoint updates description and constraints (creating a new version) but does not accept a `name` field. The Types tab detail page has no rename control. | Add `name` as an optional field on `UpdateTypeDefinitionRequest`. Renaming does NOT create a new version (it changes the identity, like entity type rename). Block renaming system types. Add inline edit for name on `TypeDefinitionDetailPage`. |
| TD-95 | Cannot mark a string attribute as multiline when adding it to an entity type | The Add Attribute modal only lets the user select a type definition — there is no option to set `multiline: true` on the attribute. Multiline is a constraint on the string type definition, not a per-attribute property. To get a multiline string field, the user must first create a separate type definition with `base_type: string` and `constraints: { multiline: true }`, then select it when adding the attribute. This is non-obvious and cumbersome for a common use case. | Either (a) add a "Multiline" checkbox in the Add Attribute modal when a string-based type is selected (override the constraint per-attribute), or (b) pre-create a system type "multiline-string" with `{ multiline: true }` so users can pick it directly, or (c) improve discoverability by showing a hint in the type selector that the user should create a custom string type with multiline enabled. |
| TD-96 | 🐛 Add Attribute modal Save button does not work | When adding an attribute to an entity type, clicking the Add/Save button does not close the modal and the attribute is not saved. The modal stays open with no visible error. Likely the `handleSubmit` in `AddAttributeModal.tsx` is failing silently — either the API call returns an error that is not displayed, or `td.latest_version_id` is undefined/empty causing the request to be rejected by the backend. | Debug `AddAttributeModal.handleSubmit`: check if `td.latest_version_id` is populated in the type definitions list, check if the API call succeeds or fails, and ensure errors are surfaced in the modal. |
| TD-97 | Type selector dropdown in Add Attribute modal is not scrollable and has no search | The type definition dropdown in `AddAttributeModal.tsx` renders all types in a flat `<Select>` list with no max height, no scroll, and no type-ahead filter. When many custom type definitions exist, the dropdown extends beyond the browser viewport and types at the bottom are not selectable. | Replace the plain `<Select>` with a PatternFly `<Select>` that has `maxHeight` set (e.g., 300px) for scrollability, and add a `hasSearchInput` / typeahead filter so users can type part of the type name to narrow the list. Apply the same fix to `EditAttributeModal.tsx`. |
| TD-98 | 🐛 Cannot clear an optional string attribute value on instance edit | When editing an instance in a catalog, clearing an optional string attribute (deleting all text so the field is empty) and saving — the change is silently ignored. The attribute retains its previous value. The backend likely treats an empty string `""` the same as "not provided" and skips the update, or the UI doesn't include attributes with empty values in the update payload. | Distinguish between "not provided" (omit from payload) and "explicitly cleared" (send empty string or null). The backend `UpdateInstance` should accept `""` as a valid value for optional string attributes and store it, clearing the previous value. The UI should include empty-string attributes in the update payload when the user explicitly clears the field. |

---

## ✅ Resolved

| ID | Item | Resolution |
|----|------|------------|
| ✅ ~~TD-4~~ | ~~Copy attributes dialog: enum name display~~ | Copy picker now uses `enum_name` from snapshot. |
| ✅ ~~TD-8d~~ | ~~Extract EdgeClickData interface~~ | Exported from `EntityTypeDiagram.tsx`, imported in `App.tsx`. |
| ✅ ~~TD-9~~ | ~~Show required attributes in diagram~~ | Required attributes prefixed with `*` in diagram UML nodes. |
| ✅ ~~TD-12~~ | ~~Catalog re-pinning~~ | `PUT /catalogs/{name}` accepts `catalog_version_id`. See US-51. |
| ✅ ~~TD-15~~ | ~~Catalog cascade delete needs transaction~~ | Wrapped in `TransactionManager.RunInTransaction`. |
| ✅ ~~TD-16~~ | ~~Catalog deletion cascade leaves orphaned IAVs and links~~ | Deletes IAVs + links before instances and catalog. |
| ✅ ~~TD-17~~ | ~~Catalog list pagination~~ | Added `limit` and `offset` query params. |
| ✅ ~~TD-20~~ | ~~Missing name validation on instance creation~~ | Added `strings.TrimSpace(name) == ""` validation. |
| ✅ ~~TD-21~~ | ~~Remove catalog_version_id migration code~~ | Migration code removed from `InitDB`. |
| ✅ ~~TD-22~~ | ~~🔴 Common attributes as schema-level attributes~~ | API-level merge of Name/Description as system attributes. |
| ✅ ~~TD-23~~ | ~~CatalogDetailPage component too large~~ | Decomposed into 6 hooks + 12 components (18 new files). |
| ✅ ~~TD-24~~ | ~~Remove legacy EntityInstanceService~~ | Removed service, handler, and tests. |
| ✅ ~~TD-25~~ | ~~Replace `interface{}` with `any`~~ | Replaced in 9 files. |
| ✅ ~~TD-27~~ | ~~ListContainedInstances pagination params ignored~~ | Extracted `parseListParams()` helper. |
| ✅ ~~TD-29~~ | ~~Reject reserved entity type names~~ | Added `reservedEntityTypeNames` blocklist. |
| ✅ ~~TD-30~~ | ~~Add catalog ownership check on instance read/update/delete~~ | Added `inst.CatalogID != catalog.ID` check. |
| ✅ ~~TD-34~~ | ~~`SetParentRequest.ParentType` missing validation~~ | Added `parent_type is required` check. |
| ✅ ~~TD-35~~ | ~~Operational catalog detail page too large~~ | Extracted `useContainmentTree` + `InstanceDetailPanel`. |
| ✅ ~~TD-36~~ | ~~Review usefulness of Overview tab~~ | Overview tab removed. See TD-56. |
| ✅ ~~TD-41~~ | ~~Show entity description in table views~~ | **Partially resolved.** See TD-43/TD-44. |
| ✅ ~~TD-42~~ | ~~⚠️ Add Contained Instance modal missing custom attributes~~ | Modal loads child schema attributes on type selection. |
| ✅ ~~TD-44~~ | ~~BOM pins table missing description~~ | Added `Description` to `ResolvedPin` and DTO. |
| ✅ ~~TD-47~~ | ~~Diagram: containment edges UML composition~~ | Filled diamond SVG marker on parent end. |
| ✅ ~~TD-48~~ | ~~Duplicate number-parsing logic~~ | Extracted `buildTypedAttrs` utility. |
| ✅ ~~TD-49~~ | ~~`useInstanceDetail.selectInstance` missing `setAuthRole` call~~ | Pass `role` to hook, call `setAuthRole(role)` at start of `selectInstance`. |
| ✅ ~~TD-50~~ | ~~`selectInstance` passes stale instance object~~ | Changed to accept ID string, re-fetches instance internally. |
| ✅ ~~TD-51~~ | ~~`onRemoveParent` swallows errors silently~~ | Catch block now sets `setParentError` for user feedback. |
| ✅ ~~TD-52~~ | ~~Modal data-loading still managed by page~~ | Modals now import `api` and load their own data. Page reduced by ~60 lines. |
| ✅ ~~TD-53~~ | ~~Diagram tab JSX duplicated across catalog pages~~ | Extracted `DiagramTabContent` component. |
| ✅ ~~TD-54~~ | ~~`CatalogVersionDetailPage` does not use `useCatalogDiagram`~~ | Refactored to use shared hook. Removed inline state. |
| ✅ ~~TD-55~~ | ~~Edge click handler object construction duplicated~~ | Extracted `buildEdgeClickData(data)` helper. +4 tests. |
| ✅ ~~TD-57~~ | ~~Move `CatalogDetailPage`/`CatalogListPage` to `pages/meta/`~~ | Moved via `git mv`. Updated imports in `App.tsx`. |
| ✅ ~~TD-58~~ | ~~🔴 Enum values are not versioned — mutations are destructive~~ | Enums replaced by versioned type definitions (FF-14). Type definition versions are pinned in CVs via `CatalogVersionTypePin`. Mutations create new versions without affecting existing CVs. |
| ✅ ~~TD-59~~ | ~~N+1 query in entity type list~~ | Added `GetLatestByEntityTypes` batch method. |
| ✅ ~~TD-60~~ | ~~Enum description edit uses `window.prompt()`~~ | Replaced with inline TextInput + Save/Cancel. +4 browser tests. |
| ✅ ~~TD-61~~ | ~~CatalogVersion metadata not editable~~ | Added `PUT /catalog-versions/:id` with `*string` pattern. See US-49. |
| ✅ ~~TD-62~~ | ~~⚠️ Audit update/PUT for data loss~~ | Fixed `Description` from `string` to `*string`. |
| ✅ ~~TD-64~~ | ~~Move TD table from PRD to `docs/td-log.md`~~ | **Done** — this file. |
| ✅ ~~TD-66~~ | ~~Duplicated role-to-service-role mapping in CV handler~~ | Replaced inline switches with `mapRole` calls. See TD-73. |
| ✅ ~~TD-68~~ | ~~Inline edit field size mismatch~~ | Removed `maxWidth: 300px`, kept `width: 100%`. +3 browser tests. |
| ✅ ~~TD-69~~ | ~~🔒 CV BOM pin editing in production~~ | Added `checkCVEditAllowed` stage guard. Extended in TD-71. |
| ✅ ~~TD-70~~ | ~~🐛 BOM table not sorted~~ | Case-insensitive sort in `loadPins()`. |
| ✅ ~~TD-71~~ | ~~⚠️🔒 UpdateCatalogVersion no stage guard~~ | Stage guard + validate bypass fix + UI canMutate guard. |
| ✅ ~~TD-72~~ | ~~🐛 Diagram node overflow~~ | Dynamic node width from longest attribute label. |
| ✅ ~~TD-73~~ | ~~`mapRole` dead code + inline switch duplication~~ | Replaced 3 inline switches with `mapRole`. Fixed missing default case. +4 tests. |
| ✅ ~~TD-74~~ | ~~CatalogVersionDetailPage too large~~ | Extracted `useInlineEdit` + `usePinManagement` hooks. 724→621 lines. +31 tests. |
| ✅ ~~TD-75~~ | ~~`handleOpenPinVersionSelect` mixes concerns~~ | Split into toggle + separate data-loading. See TD-74. |
| ✅ ~~TD-76~~ | ~~Missing browser tests for `canEditPins` visibility~~ | Added T-29.15/16/17 + regression test. +4 browser tests. |
| ✅ ~~TD-77~~ | ~~AddPin O(n) entity type resolution~~ | Added `GetByIDs` batch method. Single query for duplicate check. +5 tests. |
| ✅ ~~TD-78~~ | ~~⚠️ Association tables: Entity Type buried at end~~ | Merged into first Target column as `instance (type)`. +3 browser tests. |
| ✅ ~~TD-82~~ | ~~🐛 Diagram double-click no back path~~ | Navigate with `{ state: { from } }`. Back button reads `location.state.from`. |
| ✅ ~~TD-94~~ | ~~🐛 Number type min/max input drops leading zeros after decimal point~~ | Extracted `NumericConstraintFields` component with local string state. Parse on blur, not on every keystroke. `type="text"` instead of `type="number"`. |
