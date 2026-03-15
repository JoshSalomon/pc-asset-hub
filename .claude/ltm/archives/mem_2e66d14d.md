---
id: "mem_2e66d14d"
topic: "Phase 5: Catalog-Level RBAC — CatalogAccessChecker pattern"
tags:
  - rbac
  - catalog-access
  - middleware
  - patterns
phase: 0
difficulty: 0.3
created_at: "2026-03-15T15:39:19.887026+00:00"
created_session: 16
---
## CatalogAccessChecker Interface
- `CheckAccess(c echo.Context, catalogName, verb string) (bool, error)`
- Separate from RBACProvider (role-based vs resource-based are orthogonal)
- `HeaderCatalogAccessChecker` always returns true (dev mode)
- `SARCatalogAccessChecker` (future Phase C) will call K8s SubjectAccessReview with resourceName

## RequireCatalogAccess Middleware
- Extracts `:catalog-name` from URL path param
- Maps HTTP method to K8s verb: GET→get, POST→create, PUT→update, DELETE→delete
- Returns 403 if denied, 500 if error, passes through if allowed
- Skips check when no catalog name in path (catalog list endpoint)
- Applied to the `/:catalog-name` group in main.go, so all sub-routes inherit it

## Catalog List Filtering
- Done in CatalogHandler.ListCatalogs, not middleware (middleware doesn't have the list)
- Fetches all catalogs from DB, then filters through CatalogAccessChecker
- Returns only accessible catalogs with updated total count
- Acceptable performance since catalogs are bounded (<100)
