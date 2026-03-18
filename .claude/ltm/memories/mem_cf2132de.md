---
id: "mem_cf2132de"
topic: "Phase 5: Catalog-Level RBAC — patterns and quality review lessons"
tags:
  - rbac
  - catalog-access
  - middleware
  - security
  - quality-review
phase: 0
difficulty: 0.4
created_at: "2026-03-15T16:27:44.576837+00:00"
created_session: 16
---
## CatalogAccessChecker Interface
- `CheckAccess(c echo.Context, catalogName, verb string) (bool, error)`
- Separate from RBACProvider (role-based vs resource-based are orthogonal)
- `HeaderCatalogAccessChecker` always returns true (dev mode)
- `SARCatalogAccessChecker` (future Phase C) will call K8s SubjectAccessReview with resourceName

## RequireCatalogAccess Middleware
- Extracts `:catalog-name` from URL path param
- Maps HTTP method to K8s verb: GET→get, POST→create, DELETE→delete
- Returns 403 if denied, 500 if error, passes through if no catalog name
- Applied to BOTH catalog routes (GET/DELETE) AND instance routes
- Must be applied via RegisterCatalogRoutes for catalog-level routes AND as group middleware for sub-routes

## Critical Lesson: Apply access checks to ALL routes with catalog name
- Quality review found catalog GET/DELETE bypassed access check — only instance routes had the middleware
- Fix: apply RequireCatalogAccess in RegisterCatalogRoutes for /:catalog-name routes
- Also: CreateCatalog must check access for the catalog name being created (no :param but name is in request body)

## FilterAccessible Generic Helper
- `FilterAccessible[T any](c, checker, items, nameFunc)` — Go generics for type-safe filtering
- Avoids duplication between filtering []string vs []*CatalogDetail
- Error returns generic "access check failed" (not mapError which leaks internals)

## Verb Mapping
- Only map verbs that have actual endpoints (GET, POST, DELETE)
- Don't pre-map PUT/PATCH if no update endpoints exist — it's dead code
