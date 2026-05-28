# TD Sprint — Design Spec

**Date:** 2026-05-25
**Branch:** `020-td-sprint`
**Scope:** 15 items

---

## Stage 1: Bugs & Security (~2h)

### TD-148: Admin can unpublish published catalogs

**Current behavior:** `CatalogDetailPage.tsx:356` shows Unpublish button when `isAdmin && catalog.published`, where `isAdmin = role === 'Admin' || role === 'SuperAdmin'`. Backend route `POST /:catalog-name/unpublish` (catalog_handler.go:340) uses `requireAdmin, requireCatalogAccess` middleware — but **does not** include `RequireWriteAccess`. The `RequireWriteAccess` middleware (catalog_access.go:70-97) correctly blocks non-SuperAdmin on published catalogs, but it's simply not applied to the Unpublish route.

**Fix — Frontend:** Change `isAdmin && catalog.published` to `role === 'SuperAdmin' && catalog.published` for the Unpublish button condition.

**Fix — Backend:** Add `RequireWriteAccess` to the Unpublish route middleware chain. The `writeGuards` parameter already includes it — the route just needs to use `writeMiddleware` instead of bare `requireAdmin`.

**Decision:** The Publish route (line 339) also lacks `RequireWriteAccess`. Fix both routes for defense-in-depth — prevents future bugs if Publish behavior changes.

### TD-150: Export binding params leak to non-Admin

**Current behavior:** `ListBindings` and `GetBinding` handlers (export_binding_handler.go:49-60, ~62-75) return `bindingToDTO(b)` which unconditionally includes `Parameters map[string]string` containing target namespaces, credential secret names, and route names. The routes use only `requireCatalogAccess` — any user with catalog read access sees infrastructure details.

**Fix — Handler:** In `ListBindings` and `GetBinding`, check `GetRoleFromContext(c)`. If not Admin/SuperAdmin, use a new `bindingToDTOFiltered(b)` that omits the `Parameters` field (or sets it to nil). Non-admin users see binding existence (exporter name, status, timestamps) but not parameter values.

**Decision:** Filter at the handler level (not service level) since this is a presentation concern. Use `omitempty` on the `Parameters` JSON tag so it's omitted entirely for non-admin responses.

### TD-140: Catalog validation doesn't check instance name format

**Current behavior:** `ValidationService.Validate` (validation_service.go:151-161) checks for empty instance names but does not validate format. `ValidateInstanceName` (instance_service.go:20-30) validates DNS-1123 format but is only called at create/update time, not during catalog validation.

**Fix:** After the empty-name check loop, add a format check loop calling a format-validation function (not `ValidateInstanceName` directly, since that checks both empty AND format). Use the same regex `^[a-z0-9]([a-z0-9-]*[a-z0-9])?$` and max 63 chars. Report violation as: `"instance name contains invalid characters (expected lowercase DNS label: [a-z0-9-], max 63 chars)"`.

**Note:** Reuse the existing `instanceNameRegex` by extracting it to a shared location, or import it from instance_service. Prefer extracting to avoid circular dependencies.

---

## Stage 2: Export Polish (~4h)

### TD-152: MCP Gateway exporter hardcodes attribute names

**Current behavior:** `mcp_gateway_exporter.go` hardcodes `route_name` (line 49, 132), `mcp_path` (line 158), and `credential_secret` (line 163) when accessing instance attributes. If the actual schema uses different attribute names, the exporter fails or produces incorrect output.

**Fix:**
1. Add 3 optional binding parameters to `ParameterSchema()`:
   - `route_name_attr` (default: `"route_name"`) — server attribute for HTTPRoute name
   - `mcp_path_attr` (default: `"mcp_path"`) — server attribute for MCP endpoint path
   - `credential_secret_attr` (default: `"credential_secret"`) — server attribute for K8s secret ref
2. In `ValidateSchema()`, use the mapped attribute name instead of the hardcoded one
3. In `Export()`, read the mapped attribute name from binding parameters and use it for instance attribute lookup
4. Update UI: the BindingModal should show these as optional text fields with default values pre-filled

### TD-151: Export binding modal allows same entity type for multiple params

**Current behavior:** `ExportBindingsPanel.tsx` BindingModal (lines 314-339) renders entity_type parameter dropdowns independently. No validation checks if two params share the same entity type value.

**Fix:** Add a client-side warning (PatternFly `Alert variant="warning"`) that appears when two or more `entity_type` parameters have the same value. Do not block submission — future exporters may legitimately allow it. The warning text: `"Parameters {param1} and {param2} use the same entity type. For MCP Gateway, server and tool types should be distinct."`.

### TD-131: Export file field ordering

**Current behavior:** Multiple non-deterministic orderings in `export_service.go`:
1. Type definitions iterate `map[string]bool` (line 293) — random order
2. Associations have no sort (line 264) — DB query order
3. Instances depend on `ListByCatalog` order (line 313) — DB order
4. `ExportInstance.MarshalJSON()` iterates `Children` map (line 109) — random order
5. Attributes carry ordinal from DB — already correct

**Fix (DB-level ordering where possible, Go-level for map-derived data):**
1. **Associations**: Add `ORDER BY name` to `ListByVersion` repo query — benefits all callers, not just export
2. **Instances**: Add `ORDER BY name` to `ListByCatalog` repo query for stable instance ordering. Entity type grouping sorted in Go after name resolution (type name requires JOIN — not worth for this)
3. **Type definitions**: Sort slice by `Name` in Go after building (data assembled from individual `GetByID` calls on map iteration — no single query to order)
4. **Children map keys**: Sort in `MarshalJSON()` before iterating (Go map — must sort explicitly)
5. **Attributes**: Sort by `Ordinal` (should already be ordered from DB, but add explicit sort for safety)

---

## Stage 3: UX Papercuts (~3h)

### TD-142: Add Child modal pre-select single containment type (operational page)

**Finding:** The meta CatalogDetailPage already has this fix (lines 494-496 compute `initialChildType` and pass it to AddChildModal). But `OperationalCatalogDetailPage.tsx` is missing the same logic — line 515 opens the modal without setting `initialChildType`, and the AddChildModal at line 662 doesn't pass the prop.

**Fix:** In `OperationalCatalogDetailPage.tsx`:
1. Add `initialChildType` state (like the meta page does)
2. In the "Add Child" button onClick, check `outgoingContainment.length === 1` and set `initialChildType` to the single target type name
3. Pass `initialChildType` prop to the `AddChildModal` component

### TD-144: Set Parent modal filter current parent from dropdown

**Current behavior:** `SetParentModal.tsx` loads all instances of the parent type via `api.instances.list()` (line 48-55). The dropdown includes the current parent. Selecting it is a silent no-op.

**Fix:**
1. Add `currentParentInstanceId?: string` prop to `SetParentModal`
2. Pass it from CatalogDetailPage where the modal is invoked
3. Filter `currentParentInstanceId` from the instances list after loading

### TD-146: Orphaned containment targets visual distinction

**Current behavior:** Root-level entity type groups in the tree browser look identical regardless of whether the entity type is a legitimate root or an orphaned containment target.

**Fix:** In the tree rendering (CatalogDetailPage.tsx), check if an entity type group at root level has that entity type appearing as `target_entity_type_name` in any containment association in `schemaAssocs`. If yes, add a warning icon and "(orphaned)" suffix to the group header. This uses data already available in the component.

### TD-79: Add Pin modal version dropdown default to latest

**Current behavior:** `usePinManagement.ts:50` sets `setSelectedEtvId('')` after loading versions, forcing the user to manually select.

**Fix:** After loading versions in `handleSelectEntityType`, auto-select the version with the highest `version` number:
```typescript
const latest = res.items.reduce((a, b) => a.version > b.version ? a : b)
setSelectedEtvId(latest.id)
```

### TD-33: "Contained by" flickers UUID

**Current behavior:** `CatalogDetailPage.tsx:471` shows `detail.parentName || detail.selectedInstance.parent_instance_id`. `useInstanceDetail.ts:38-42` fetches parent name asynchronously — between instance load and parent name resolution, the UUID flickers.

**Fix — Backend:** Include `parent_instance_name` in all instance API responses (list, get, contained-list) via a self-JOIN at the repository level: `LEFT JOIN entity_instances parent ON parent.id = ei.parent_instance_id`. Add `ParentInstanceName *string` to the domain model and DTO response. This avoids N+1 lookups on list endpoints. Consistent across all endpoints to prevent similar issues in other views.

**Fix — Frontend:** Use `parent_instance_name` from the API response directly, removing the async lookup in `useInstanceDetail.ts`. Show "loading..." instead of UUID as fallback during the brief window where the response hasn't arrived yet.

### TD-137: Types tab missing name filter text field

**Current behavior:** `TypeDefinitionListPage.tsx` has a base type dropdown filter but no name text search. The Entity Types tab has a `SearchInput` for name filtering.

**Fix:** Add a `SearchInput` component to the Types tab toolbar, following the same pattern as EntityTypeListPage. Filter `typeDefs` by `td.name.toLowerCase().includes(filterText.toLowerCase())` before applying the base type filter.

---

## Stage 4: Code Quality (~1h)

### TD-87: App.system.test.ts refactor to shared helpers

**Current behavior:** `App.system.test.ts` defines inline `visible()`, `hidden()`, `apiCall()`, `getTypeVersionId()`, `cleanupTestData()` that duplicate equivalents in `test-helpers/system.ts`.

**Fix:** Replace inline helpers with imports from `test-helpers/system.ts`. Key differences to reconcile:
- `apiCall()` — inline version has no `role` param (hardcodes Admin). Use shared version with explicit role.
- `cleanupTestData()` — inline version has different `TEST_PREFIXES`. Merge prefixes into shared `cleanupE2EData()` or parameterize it.
- Keep test-specific helpers (`navigateToEntityType`, `navigateToTypeDefDetail`, `trackResource`) if they're not generalizable.

### TD-154: Preview cache TTL env var caching

**Current behavior:** `ExportBindingService.getPreviewTTL()` (publish_service.go:120-127) reads `PUBLISH_PREVIEW_TTL` env var on every call via `os.Getenv()`.

**Fix:** Add a `previewTTL time.Duration` field to `ExportBindingService`. Initialize it in `NewExportBindingService()`:
```go
ttl := 5 * time.Minute
if v := os.Getenv("PUBLISH_PREVIEW_TTL"); v != "" {
    if secs, err := strconv.Atoi(v); err == nil {
        ttl = time.Duration(secs) * time.Second
    }
}
```
Change `getPreviewTTL()` to return `s.previewTTL`.

### TD-132: Remove unused accessChecker field

**Current behavior:** `ExportHandler` (export_handler.go:14-17) and `ImportHandler` (import_handler.go:12-15) both declare `accessChecker CatalogAccessChecker` fields that are never used. Access control is enforced at the middleware level.

**Fix:** Remove the field from both structs, remove from constructors, update all call sites (main.go, tests).

---

## Cross-Cutting Concerns

- **TD-148 + TD-150:** Both are security fixes. Deploy and verify with live tests immediately after Stage 1.
- **TD-131 + TD-152:** Both affect export. TD-131 sorting should be done first since it affects test baselines.
- **TD-33:** Backend change (add `parent_instance_name` to response) needs both backend and frontend changes. Run backend tests before moving to UI.

## Revised Item Count

| Stage | Items | Notes |
|-------|-------|-------|
| Stage 1: Bugs & Security | 3 | TD-148, TD-150, TD-140 |
| Stage 2: Export Polish | 3 | TD-152, TD-151, TD-131 |
| Stage 3: UX Papercuts | 6 | TD-142, TD-144, TD-146, TD-79, TD-33, TD-137 |
| Stage 4: Code Quality | 3 | TD-87, TD-154, TD-132 |
| **Total** | **15** | |
