---
id: "mem_3b7f561e"
topic: "AI Asset Hub — Tech Stack and Project Structure (current as of 2026-03-30)"
tags:
  - architecture
  - tech-stack
  - project-structure
phase: 0
difficulty: 0.5
created_at: "2026-03-30T12:24:15.087420+00:00"
created_session: 17
---
## Tech Stack

- **Backend**: Go 1.25, Echo framework, GORM ORM
- **Databases**: SQLite (dev/test), PostgreSQL 16 (production)
- **APIs**: REST — Meta `/api/meta/v1/`, Operational `/api/data/v1/catalogs/{name}/`
- **UI**: React 19 + TypeScript + PatternFly 6 + Vite 7
- **Testing**: Vitest 4 (browser mode with Playwright), Go testing + testify
- **IDs**: UUID v7 via `google/uuid`
- **Operator**: `controller-runtime`, watches AssetHub + CatalogVersion + Catalog CRDs
- **Containers**: Podman, distroless base images, kind for local K8s
- **RBAC**: Header-based dev mode (`X-User-Role`), OpenShift SAR planned (Phase C)

## Project Structure

```
cmd/api-server/main.go           # Composition root
cmd/operator/main.go              # Operator entrypoint
internal/domain/{models,repository,errors}/  # Domain layer
internal/service/meta/            # Entity type, CV, enum, association services
internal/service/operational/     # Catalog, instance, validation services
internal/infrastructure/gorm/     # GORM repos, models, testutil
internal/api/meta/                # Meta API handlers
internal/api/operational/         # Operational API handlers (catalog + instance)
internal/api/middleware/          # RBAC, catalog access
internal/api/dto/                 # Request/response DTOs
internal/operator/                # CRD types, controllers, reconciler
ui/src/                           # Unified SPA (React + PatternFly)
scripts/                          # Deploy, test, coverage scripts
```

## Unified SPA (US-47)
- Single `index.html`, BrowserRouter
- `/` — Landing page with catalog cards
- `/schema/*` — Schema management (entity types, CVs, enums, catalogs)
- `/catalogs/:name` — Operational data viewer (read-only tree browser)

## RBAC Roles
RO, RW, Admin, SuperAdmin. Group-level middleware extracts role, per-route `RequireRole()` enforces minimum. `RequireWriteAccess` blocks non-SuperAdmin on published catalogs. `RequireCatalogAccess` checks per-catalog permissions.
