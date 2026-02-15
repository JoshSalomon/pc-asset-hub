---
id: "mem_ee0bef4e"
topic: "AI Asset Hub - Layered Architecture and Project Structure"
tags:
  - architecture
  - project-structure
  - asset-hub
phase: 0
difficulty: 0.7
created_at: "2026-02-12T17:36:23.838858+00:00"
created_session: 1
---
## Layered Architecture

Strict separation: `API → Service → Domain ← Infrastructure`

### Layer Boundary Rules
- `domain/` (models + repository interfaces): Standard library only. NO external dependencies.
- `service/`: Imports only from `domain/`. Never from `infrastructure/` or `api/`.
- `api/`: Imports from `service/` and `domain/`. Never from `infrastructure/`.
- `infrastructure/gorm/`: Imports from `domain/`, GORM, DB drivers. Never from `service/` or `api/`.
- `cmd/` (composition root): Wires GORM implementation into services via dependency injection.

### Project Structure
```
pc-asset-hub/
  cmd/api-server/          # API server entrypoint (composition root)
  cmd/operator/            # Operator entrypoint
  internal/domain/models/  # Pure Go domain models (no GORM tags)
  internal/domain/repository/  # Storage-agnostic interfaces
  internal/domain/errors/  # Domain error types
  internal/service/meta/   # Meta business logic
  internal/service/operational/  # Operational business logic
  internal/service/versioning/   # Version management
  internal/service/validation/   # Cycle detection, uniqueness
  internal/infrastructure/gorm/models/      # GORM model structs
  internal/infrastructure/gorm/repository/  # GORM implementations
  internal/infrastructure/gorm/migrations/  # Migrations
  internal/api/meta/       # Meta API handlers
  internal/api/operational/ # Operational API handlers
  internal/api/middleware/  # Auth, RBAC, logging
  internal/api/dto/        # Request/response DTOs
  internal/operator/controllers/  # Reconcilers
  internal/operator/crdgen/      # CRD/CR generation
  ui/src/                  # React + PatternFly UI
```

### Key Design Principle
The GORM implementation is pluggable. Swapping to MongoDB, etcd, or another store requires only implementing `domain/repository/` interfaces — no changes to service, API, or UI layers.
