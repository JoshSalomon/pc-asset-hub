---
id: "mem_f1f6c181"
topic: "Catalog Foundation — implementation patterns and gotchas"
tags:
  - catalog
  - architecture
  - implementation
  - operational-api
  - gotchas
phase: 0
difficulty: 0.6
created_at: "2026-03-09T15:25:16.184113+00:00"
created_session: 16
---
## Catalog Foundation (Phase 1 — implemented)

### Architecture
- **Catalog** is a named data container pinned to a CatalogVersion. Instances belong to catalogs, not CVs.
- `EntityInstance.CatalogVersionID` renamed to `CatalogID` across entire codebase.
- Operational API uses catalog **name** (not ID) in URLs: `/api/data/v1/catalogs/{catalog-name}`
- Catalog names must be DNS-label: `^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`, max 63 chars

### Key Files
- Domain: `internal/domain/models/models.go` (Catalog struct), `internal/domain/repository/catalog.go`
- GORM: `internal/infrastructure/gorm/repository/catalog_repo.go`
- Service: `internal/service/operational/catalog_service.go` — returns `CatalogDetail` with resolved CV label
- Handler: `internal/api/operational/catalog_handler.go` — uses `dto.CatalogResponse`
- UI: `ui/src/pages/operational/CatalogListPage.tsx`
- Design doc: `docs/plans/2026-03-09-catalog-implementation-design.md` (6-phase plan)

### Gotchas
- **PatternFly Tabs render ALL content on mount** (hidden via CSS). Child `useEffect` runs before parent. Call `setAuthRole(role)` in each page's load function, not just in App.tsx.
- **Frontend uses separate `DATA_BASE_URL`** env var (`VITE_DATA_API_BASE_URL`) for operational API.
- **`CatalogService.List`** returns `[]*CatalogDetail` with batch-cached CV label resolution.
- **Cascade delete** is not transactional (TD-15). Instances are soft-deleted, catalog is hard-deleted (TD-16).

### Technical Debt
- TD-14: Catalogs using this CV (cross-reference on CV detail page)
- TD-15: Cascade delete needs transaction
- TD-16: Mixed soft-delete/hard-delete
- TD-17: List pagination (limit/offset params)
- TD-18: UI props style inconsistency
