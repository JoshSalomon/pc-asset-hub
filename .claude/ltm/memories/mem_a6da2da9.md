---
id: "mem_a6da2da9"
topic: "Catalog Publishing (Phase 7) implementation patterns"
tags:
  - catalog
  - publishing
  - phase-7
  - k8s
  - operator
  - architecture
phase: 0
difficulty: 0.5
created_at: "2026-03-16T17:25:02.992236+00:00"
created_session: 16
---
## Catalog Publishing

- `CatalogService.Publish/Unpublish` in `internal/service/operational/catalog_service.go`
- `Published` bool + `PublishedAt` *time.Time fields on Catalog model
- Publish requires `valid` status; rolls back DB if K8s CR creation fails
- Unpublish checks `catalog.Published` for early return (idempotent)
- Delete cleans up Catalog CR for published catalogs before DB deletion
- `RequireWriteAccess` middleware blocks non-SuperAdmin mutations on published catalogs
- `CatalogPublishChecker` interface accepts `echo.Context` for proper context propagation (not `context.Background()`)

## K8s CR Architecture
- Separate `K8sCatalogCRManager` type (NOT methods on `K8sCRManager`) to avoid Go method name collision — Go doesn't support overloading
- `CatalogCRSpec.Name` serves as both K8s resource name and catalog name (no redundant fields)
- `ReconcileCatalogStatus` pure function — only updates status when NOT Ready (prevents infinite reconciliation loop)
- **CRITICAL**: Never use `currentValue != currentValue + 1` as a change guard — it's always true, causing infinite loops
- `status.DataVersion` increments only on status transitions, not on every reconcile
- Catalog CRs namespaced in AssetHub namespace (same as CatalogVersion CRs)

## CV Promotion Warnings
- `CatalogVersionService.Promote` returns `*PromoteResult` with `Warnings []CatalogWarning`
- Warnings are best-effort — `ListByCatalogVersionID` errors don't block promotion
- Handler returns `warnings: []` (always present, even if empty) for consistent API shape

## Route Registration
- `RegisterCatalogRoutes` takes `requireRW` AND `requireAdmin` middleware params
- `RegisterInstanceRoutes` uses variadic `writeGuards ...echo.MiddlewareFunc` for composable write protection
- `httpMethodToVerb` must map PUT/PATCH → "update" (not default to "get")
