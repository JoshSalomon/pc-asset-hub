# Meta Model Operations — Gap Analysis

## Priority 1: Core Missing CRUD (blocks other features)

| Gap | Backend | UI | Notes |
|-----|---------|-----|-------|
| **Edit attribute** (name/desc/type) | No endpoint | No UI | Can only delete and re-add today |
| **Edit entity type name/description** | PUT exists but only description | No inline edit UI | PRD says inline auto-save |
| **Copy attributes from another type** | Service exists, no handler/route | No UI | PRD US-4, US-29 |

## Priority 2: Catalog Version Creation Workflow (high-impact UX)

| Gap | Status |
|-----|--------|
| Entity type selection during creation | Only label input today, no pin selection |
| BOM display (bill of materials) | Not shown on create or detail |
| Catalog version detail page | Doesn't exist — no way to see what's in a version |
| Lifecycle transition history | Not displayed anywhere |
| Filter catalog versions by stage | Not implemented |

## Priority 3: Model Visualization (new feature, zero implementation)

| Gap | Status |
|-----|--------|
| UML-style class diagram component | Nothing exists |
| Live model view (current state) | Nothing exists |
| Catalog version model view | Nothing exists |
| Interactive diagram (double-click, zoom/pan) | Nothing exists |
| Diagram-based entity selection for CV creation | Nothing exists |

## Priority 4: Polish and Completeness

| Gap | Status |
|-----|--------|
| Enum: show referencing attributes/entity types | Not displayed |
| Enum: inline creation from attribute add form | Must navigate to Enums page |
| Enum: value count in list view | Not shown |
| Attribute: "required" field display | Not in UI |
| Association: visual distinction (icons/grouping by type) | All in one flat table |
| Association: real-time cycle detection feedback | Only on submit |
| Entity type list: attribute/association counts | Not displayed |
| Version history: view past version detail (read-only) | Can't click to view |
| Version history: copy from past version | Only current version |
| Role-aware: RW vs Admin distinction in meta UI | No difference today |
| Role-aware: disabled+tooltip instead of hidden controls | Controls just hidden |
| Role-aware: production state enforcement in UI | Not checked |

## Technical Debt

| Item | Current Behavior | Required Behavior |
|------|-----------------|-------------------|
| Enum deletion safety | Enum delete checks if any attribute references it (flat check across all versions) | Enum cannot be deleted if it is used by any attribute in a **used entity version**. A used entity version is: (1) any version pinned by a catalog version, or (2) the latest version of any entity type (which belongs to an implicit pre-production catalog). Unused historical versions should not block deletion. |
| Catalog version timestamp uniqueness | Two catalog versions can have the same `created_at` timestamp | `created_at` must be unique across catalog versions. The backend should enforce this (e.g., retry with a small delay if a conflict is detected). This ensures deterministic sort order in the CV list (`ORDER BY created_at DESC`). |
| Association target+role uniqueness | No uniqueness constraint on target entity type + target role per source entity type | Target entity type + target role must be unique per source entity type version. Empty target role is valid (one allowed per target). API should reject duplicates with 409. |
| Copy attributes modal: enum name display | Enum attributes show type label "enum" without the enum name | Enum attributes should display the enum name (e.g., "enum (Month)") so users can distinguish between different enum types |

## Recommended Implementation Order

1. **Priority 1** first — basic CRUD gaps that affect daily use
2. **Priority 2** next — catalog version creation is unusable without entity selection
3. **Priority 3** depends on Priority 2 (diagram-based selection needs the selection logic first)
4. **Priority 4** can be interleaved as polish

---

## Architecture Decisions (Approved 2026-02-18)

### AD-1: EditAttribute uses copy-on-write versioning

Same pattern as AddAttribute/RemoveAttribute: new EntityTypeVersion, BulkCopyToVersion, then modify the target attribute via existing `attrRepo.Update()`. Type changes (e.g., string to number) are allowed during development — operational validation is future scope.

### AD-2: Entity type rename is context-sensitive (simple or deep copy)

`EntityType.Name` lives on the EntityType model, not EntityTypeVersion. Renaming behavior depends on how the entity type is referenced by catalog versions:

**Simple rename** (direct update on EntityType record):
- Entity type is not part of any catalog version, OR
- Entity type exists in exactly one catalog version that is in development stage

**Deep copy rename** (fork — new entity type with new name):
- Entity type is part of multiple catalog versions, OR
- Entity type is part of any catalog version in testing or production stage

Deep copy creates a new EntityType with the new name and copies the latest version's attributes. The old entity type remains unchanged in existing catalog versions.

The API uses a two-step flow:
1. Client calls `POST /entity-types/:id/rename` with `{name, deep_copy_allowed: false}`
2. If simple rename is possible: performs it, returns the updated entity type
3. If deep copy is required: returns 409 with `{"error": "deep_copy_required", "message": "..."}`
4. If approved: client re-calls with `{name, deep_copy_allowed: true}` — performs deep copy, returns new entity type

**Implementation notes:**
- Requires new pin repo method `ListByEntityTypeVersionIDs(ctx, []string)` to find catalog versions referencing an entity type
- Requires adding `pinRepo` and `cvRepo` to `EntityTypeService` constructor

### AD-3: CopyAttributes wires existing service to new handler

`CopyAttributesFromType` service method and `CopyAttributesRequest` DTO already exist. Only needs: handler method + route registration.

### AD-4: Catalog version detail exposes pins and transitions

Two new read-only endpoints:
- `GET /catalog-versions/:id/pins` — resolved entity type names + version numbers
- `GET /catalog-versions/:id/transitions` — lifecycle transition history

Uses existing repo methods: `pinRepo.ListByCatalogVersion()`, `ltRepo.ListByCatalogVersion()`.

### AD-5: Stage filter uses existing GORM filter support

GORM repo already handles `params.Filters["lifecycle_stage"]` (catalog_version_repo.go:62-64). Handler reads `?stage=` query param and passes through. No service/repo changes needed.

### AD-6: CV creation entity selection is UI-only

Backend already accepts `pins` in `CreateCatalogVersionRequest`. UI currently ignores it. Change is purely frontend.
