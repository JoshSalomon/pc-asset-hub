# Catalog Implementation Design

## Context

The meta layer (entity types, attributes, associations, enums, catalog versions with lifecycle) is fully implemented. The next major milestone is the **catalog and operational layer** — enabling users to create named data collections, populate them with entity instances, and browse/consume that data.

The current operational code (`internal/api/operational/`, `internal/service/operational/`) is scaffolding that scopes instances directly to a CatalogVersion ID. The PRD defines a different model: instances belong to a **Catalog**, which is a named collection pinned to a CatalogVersion. This scaffolding will be replaced (Option C).

## Key Design Decisions

### Catalog vs. CatalogVersion scoping

Instances belong to a **Catalog**, not directly to a CatalogVersion. The Catalog knows its pinned CV; the CV determines the schema. This enables:

- Multiple named datasets sharing the same schema (e.g., "Production App A" and "Staging App B" both on CV v2.0)
- Validation status tracked per dataset, independent of schema lifecycle
- Clean separation: CV = schema shape, Catalog = actual data

The `EntityInstance.CatalogVersionID` field will be replaced with `EntityInstance.CatalogID`.

### Separate Operational UI

The operational UI (catalog data viewer) runs on a **separate port** from the meta UI:

| | Meta UI | Operational UI |
|---|---------|---------------|
| Persona | Admin building schemas | Operator/consumer browsing assets |
| Port | 30000 | 30001 |
| API consumed | `/api/meta/v1/...` | `/api/data/v1/...` |
| RBAC focus | Admin/SuperAdmin | RO-friendly, RW for edits |

Both UIs live in the same `ui/` codebase with two Vite entry points (`main-meta.tsx` and `main-operational.tsx`), producing two separate builds. Shared types, components, and API client are reused. The operator deploys them on separate ports.

## Phased Plan

### Phase 1: Catalog Foundation

**Goal:** Catalog as a first-class entity with CRUD, and the domain model refactoring.

**Backend:**

- New `Catalog` domain model:
  ```
  Catalog {
    ID               string
    Name             string
    Description      string
    CatalogVersionID string    // pinned CV
    ValidationStatus string    // draft | valid | invalid
    CreatedAt        time.Time
    UpdatedAt        time.Time
  }
  ```
- `CatalogRepository` interface + GORM implementation
- `CatalogService` — create, get, list, delete
- Change `EntityInstance.CatalogVersionID` to `EntityInstance.CatalogID`
- Remove old CV-scoped operational handler, service, and tests

**API:**

- `POST /api/data/v1/catalogs` — create catalog (name, description, catalog_version_id). Name must be DNS-label compatible.
- `GET /api/data/v1/catalogs` — list catalogs (filter by CV, validation status)
- `GET /api/data/v1/catalogs/{catalog-name}` — get catalog detail (includes resolved CV info)
- `DELETE /api/data/v1/catalogs/{catalog-name}` — delete catalog (cascades all instances)

**UI (meta UI — catalog management is an admin concern at this stage):**

- Catalog list page (name, pinned CV label, validation status badge, created date)
- Create catalog modal (name, description, select CV from dropdown)
- Delete catalog with confirmation

**User stories:** US-33, US-21

---

### Phase 2: Instance CRUD with Attributes

**Goal:** Create, read, update, delete entity instances within a catalog, including attribute values.

**Backend:**

- Rework `EntityInstanceService` — catalog-scoped instance creation
- On create: verify entity type is pinned in the catalog's CV
- Set attribute values on create and update
- Type validation: string (any), number (parseable), enum (value in allowed list)
- Missing optional attributes allowed (draft mode)
- Name uniqueness: global within catalog for top-level, within parent for contained
- Optimistic locking on update (version mismatch returns 409)
- Instance response includes resolved attribute values (attribute name, type, value)

**API:**

- `POST /api/data/v1/catalogs/{catalog-name}/{entity-type}` — create instance with attributes
- `GET /api/data/v1/catalogs/{catalog-name}/{entity-type}` — list instances of a type
- `GET /api/data/v1/catalogs/{catalog-name}/{entity-type}/{instance-id}` — get instance with attributes
- `PUT /api/data/v1/catalogs/{catalog-name}/{entity-type}/{instance-id}` — update instance attributes
- `DELETE /api/data/v1/catalogs/{catalog-name}/{entity-type}/{instance-id}` — delete (cascade contained)

**UI (meta UI — data entry is still admin/RW workflow):**

- Catalog detail page with tabs per entity type (driven by pinned CV's pins)
- Instance list table per entity type
- Create instance modal (name, description, dynamic attribute form based on schema)
- Edit instance modal (update attribute values)
- Delete instance with confirmation

**User stories:** US-13, US-14, US-15, US-17 (basic list)

---

### Phase 3: Containment & Association Links

**Goal:** Hierarchical instance creation and association linking between instances.

**Backend:**

- Contained instance creation scoped to parent
- Name uniqueness within parent namespace
- Containment traversal queries (list children by entity type)
- Association link CRUD — create/delete links between instances
- Validate links against association definitions in the CV (correct entity types, correct direction)
- Forward and reverse reference queries with resolved target info

**API:**

- `POST /api/data/v1/catalogs/{catalog-name}/{parent-type}/{parent-id}/{child-type}` — create contained instance
- `GET /api/data/v1/catalogs/{catalog-name}/{parent-type}/{parent-id}/{child-type}` — list contained instances
- `POST /api/data/v1/catalogs/{catalog-name}/{entity-type}/{instance-id}/links` — create association link
- `DELETE /api/data/v1/catalogs/{catalog-name}/{entity-type}/{instance-id}/links/{link-id}` — delete link
- `GET /api/data/v1/catalogs/{catalog-name}/{entity-type}/{instance-id}/references` — forward refs (resolved)
- `GET /api/data/v1/catalogs/{catalog-name}/{entity-type}/{instance-id}/referenced-by` — reverse refs (resolved)

**UI (meta UI):**

- Instance detail shows containment children (expandable)
- "Add contained instance" action from parent
- Association link management (link to existing instance, unlink)
- References tab on instance detail

**Deferred to Phase 4:** Multi-level containment URLs (e.g., `GET /{catalog}/a/{a-id}/b/{b-id}/c`) are not implemented in Phase 3. Single-level parent-child routes (`/{parent-type}/{parent-id}/{child-type}`) are sufficient for creating and navigating multi-level hierarchies — each level is addressed through its immediate parent. The deep URL path pattern is a browsing convenience that fits naturally with the Phase 4 containment tree endpoint.

**User stories:** US-16, US-18, US-19, US-20

---

### Phase 4: Catalog Data Viewer

**Goal:** A read-optimized operational UI on a separate port for browsing and consuming catalog data. Includes filtering and sorting.

**Backend:**

- Attribute-based filtering on instance list queries
  - String: contains (case-insensitive)
  - Number: exact, range (min/max)
  - Enum: exact match
- Multi-field sorting (ascending/descending)
- Pagination (offset/limit with total count)
- Containment tree endpoint — returns full instance hierarchy for a catalog
- Rich instance detail — resolved attributes, parent chain (for breadcrumb), children summary

**API:**

- `GET /api/data/v1/catalogs/{catalog-name}/tree` — containment tree for the catalog
- Query params on list endpoints: `?filter=attr:value&sort=attr:asc&limit=20&offset=0`
- Instance detail already returns resolved data (from Phase 2), enhanced with parent chain

**UI (new operational UI on port 30001):**

- Vite multi-entry setup: `main-meta.tsx` (existing) and `main-operational.tsx` (new)
- Shared components, types, and API client between the two apps
- Operator/deploy changes to serve operational UI on port 30001

Operational UI pages:
- **Catalog list** — browse available catalogs with name, CV label, status, instance counts
- **Catalog detail** — overview of entity types with instance counts
- **Containment tree browser** — expandable tree showing the full hierarchy; click a node to view detail
- **Instance detail panel** — all attribute values (resolved enum names), description, version, timestamps
- **Reference navigation** — "References" and "Referenced by" tabs with clickable links to target instances
- **Breadcrumb navigation** — containment path (Catalog > MCP Server "my-server" > Tool "my-tool")
- **Filtering controls** — per-attribute filters on instance lists
- **Sort controls** — column header click to sort
- **Pagination** — page size selector, page navigation
- **Role-aware** — RO users see everything without edit/create/delete controls; RW users get edit actions

**User stories:** US-17, US-18, US-19, US-20, US-21

---

### Phase 5: Catalog Validation

**Goal:** On-demand schema validation of catalog data.

**Backend:**

- `CatalogValidationService` — validates all instances in a catalog against the pinned CV:
  - Required attributes have values
  - Attribute values match declared type (string, number, valid enum value)
  - Mandatory associations satisfied (cardinality `1` or `1..*`)
  - Containment hierarchy consistent (no orphaned contained instances)
- Returns structured error list: `[{entity_type, instance_name, field, violation}]`
- Updates catalog validation status: pass → `valid`, fail → `invalid`
- Any data mutation (create/update/delete instance, set attribute, link/unlink) resets status to `draft`

**API:**

- `POST /api/data/v1/catalogs/{catalog-name}/validate` — trigger validation, returns errors list

**UI (both meta and operational):**

- Validate button on catalog detail
- Validation results display (grouped by entity type, per-instance errors)
- Validation status badge updates after validation
- CV promotion dialog warns about catalogs with `draft` or `invalid` status

**User stories:** US-34

---

### Phase 6: Catalog K8s CRs & Promotion Warnings

**Goal:** Publish valid catalogs as K8s discovery artifacts.

**Backend:**

- Catalog CR type definition (catalog name, CV reference, API endpoint, catalog ID, validation status)
- CR lifecycle: create when validation status becomes `valid`, update on status change, delete on catalog deletion
- CV promotion check: warn if any pinned catalogs are `draft` or `invalid`

**K8s / Operator:**

- Operator watches Catalog CRs, sets owner references to AssetHub CR
- Reconciler updates status conditions

**UI:**

- CV promotion dialog shows catalog validation warnings with list of affected catalogs

**User stories:** PRD section 4.2 (Catalog CRs), section 3.4 (promotion warnings)

---

## Phase Dependencies

```
Phase 1 (Foundation)
  |
  v
Phase 2 (Instance CRUD)
  |
  v
Phase 3 (Containment & Links)
  |
  v
Phase 4 (Data Viewer)    -- also depends on Phase 2 for attribute display
  |
  v
Phase 5 (Validation)     -- needs Phases 2-3 for complete validation
  |
  v
Phase 6 (K8s CRs)        -- needs Phase 5 for validation status
```

## Out of Scope

- Catalog re-pinning (upgrading a catalog to a newer CV) — PRD TD-12, future capability
- Entity type CRDs (full schema as K8s resources) — PRD future scope
- Hub-and-spoke topology — PRD section 8.4, future enhancement
- Catalog CR scoping (namespaced vs cluster-scoped) — TBD during Phase 6
