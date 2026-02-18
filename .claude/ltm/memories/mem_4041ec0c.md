---
id: "mem_4041ec0c"
topic: "AI Asset Hub - Tech Stack and Architecture (current)"
tags:
  - architecture
  - tech-stack
  - api
  - rbac
  - project-structure
phase: 0
difficulty: 0.7
created_at: "2026-02-16T15:48:17.526342+00:00"
created_session: 3
---
## Tech Stack

- **Backend**: Go 1.25.7, Echo framework, GORM ORM
- **Databases**: SQLite (dev/test), PostgreSQL 16 (production). Runtime selection via `internal/infrastructure/config/`
- **API**: REST — Meta at `/api/meta/v1/`, Operational at `/api/data/v1/:catalog-version/`
- **UI**: React 19 + TypeScript + PatternFly 6 + Vite 7
- **UI state**: useState/useCallback hooks (no Redux, no React Query yet)
- **Testing**: Vitest 4, Vitest Browser Mode with Playwright, React Testing Library
- **IDs**: UUID v7 via `google/uuid`
- **Operator**: `controller-runtime` (NOT operator-sdk), watches AssetHub CRD
- **Containers**: Podman/Docker, distroless base images, kind for local K8s
- **RBAC**: Header-based (`X-User-Role`) via `HeaderRBACProvider`. OpenShift SAR planned but not implemented.

## Project Structure

```
cmd/api-server/main.go          # Composition root
cmd/operator/main.go             # Operator entrypoint
internal/domain/models/          # Pure Go domain models
internal/domain/repository/      # Interfaces + mocks/
internal/domain/errors/          # Domain error types
internal/service/meta/           # Entity type, catalog, enum, association, attribute, version history services
internal/service/operational/    # Instance service
internal/service/validation/     # Cycle detection
internal/infrastructure/config/  # Env-based configuration
internal/infrastructure/gorm/    # database/, models/, repository/, testutil/
internal/api/meta/               # Meta API handlers + router
internal/api/operational/        # Operational API handlers
internal/api/middleware/          # CORS, RBAC (HeaderRBACProvider)
internal/api/dto/                # Request/response DTOs
internal/api/health/             # /healthz, /readyz
internal/operator/api/v1alpha1/  # CRD types + DeepCopy + scheme
internal/operator/controllers/   # Reconciler (pure) + Controller (controller-runtime)
internal/operator/crdgen/        # CRD/CR generation from entity types
build/                           # Dockerfiles (api-server, ui, operator)
deploy/kind/                     # kind cluster config
deploy/k8s/                      # K8s manifests (namespace, postgres, api-server, ui, operator)
scripts/kind-deploy.sh           # Deployment automation
ui/src/                          # React + PatternFly UI
```

## API Routes (actual)

Meta: GET/POST `/entity-types`, GET/PUT/DELETE `/entity-types/:id`, POST `/entity-types/:id/copy`, GET/POST `/catalog-versions`, GET `/catalog-versions/:id`, POST `/:id/promote`, POST `/:id/demote`

Operational: POST/GET `/:entity-type`, GET/PUT/DELETE `/:entity-type/:id`, GET `/:entity-type/:id/references`

## RBAC

Header-based. Roles: RO, RW, Admin, SuperAdmin. Group-level middleware extracts role, per-route `RequireRole()` enforces minimum. Admin required for entity type mutations, RW for catalog version mutations.

