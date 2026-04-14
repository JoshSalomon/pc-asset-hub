# AI Asset Hub -- Developer Guide

This guide covers how to set up, build, test, and extend the AI Asset Hub codebase.

## Prerequisites

- **Go 1.25+** -- the module requires Go 1.25.7 (see `go.mod`)
- **Node.js 20+** and npm -- for the React/PatternFly frontend
- **Podman** (preferred) or Docker -- for container image builds
- **kind** -- for local Kubernetes cluster development
- **kubectl** -- for cluster management

## Project Structure

The project follows a layered, domain-driven design. Key directories:

```
cmd/
  api-server/          Entry point and composition root for the API server
  operator/            Entry point for the K8s operator

internal/
  domain/
    models/            Pure Go domain model structs (no ORM tags)
    repository/        Storage-agnostic repository interfaces
    errors/            Domain-specific error types
  infrastructure/
    gorm/
      models/          GORM model structs with DB tags, mapping to/from domain
      repository/      GORM implementations of domain repository interfaces
    k8s/               Kubernetes CRManager for CatalogVersion CR lifecycle
    config/            Configuration loading from environment variables
  service/
    meta/              Schema management services (entity types, attributes,
                       associations, type definitions, catalog versions)
    operational/       Data management services (catalogs, instances, links,
                       validation, publishing, copy/replace)
    versioning/        Auto-increment version management, copy-on-write logic
    validation/        Cycle detection, uniqueness checks, constraint enforcement
  api/
    meta/              Meta API HTTP handlers and DTOs
    operational/       Operational API HTTP handlers and DTOs
    middleware/        Auth, RBAC, CORS, error handling middleware
  operator/
    api/v1alpha1/      CRD type definitions (AssetHub, CatalogVersion)
    controllers/       Reconciler implementations

ui/src/
  api/                 TypeScript API client modules
  components/          Reusable React/PatternFly UI components
  pages/
    meta/              Schema management pages (entity types, CVs, type defs)
    operational/       Catalog data viewer and catalog management pages
  hooks/               Custom React hooks (validation, queries, etc.)
  utils/               Shared utility functions
  types/               TypeScript type definitions
```

## Building

### Backend

```bash
cd /path/to/pc-asset-hub
go build ./...
```

The API server binary is built from `cmd/api-server/main.go`. It uses Echo for HTTP routing, GORM for database access (SQLite in dev, PostgreSQL in production), and wires all dependencies via constructor injection in `main()`.

### Frontend

```bash
cd ui
npm install
npm run build
```

Vite builds a multi-entry SPA: the main app (`index.html`) serves both schema management and catalog data viewer pages. PatternFly provides the component library.

## Testing

The project has a four-tier testing strategy. All test commands are available as Makefile targets and can be run from any directory using the absolute path:

```bash
make -f /path/to/pc-asset-hub/Makefile <target>
```

### Tier 1: Go Backend Tests

```bash
make test-backend       # Unit + integration tests (SQLite in-memory)
make coverage-backend   # Same, with coverage report
make test-postgres      # PostgreSQL integration (requires running PG)
```

- Service tests use mock repositories from `internal/domain/repository/mocks/`.
- Handler tests use mock services with `httptest` and Echo context.
- Repository tests use `testutil.NewTestDB(t)` for in-memory SQLite with auto-migration.
- Error-path coverage uses a `closedDB(t)` helper that closes the DB connection to trigger error branches.

### Tier 2: UI Unit Tests (jsdom)

```bash
cd ui && npm test
```

Fast tests that run in jsdom. Good for rendering, data display, and component props. Cannot test PatternFly interactive behavior (modals, tabs, selects) due to missing browser APIs.

### Tier 3: UI Browser Tests (Vitest Browser Mode)

```bash
make test-browser       # Runs in real Chromium
make coverage-browser   # Same, with coverage report
```

Tests run in real Chromium via Vitest Browser Mode. All PatternFly interactive behavior works. Uses `vitest-browser-react` for rendering and `page` locators for assertions.

### Tier 4: System and Live Tests

```bash
make test-system        # Playwright tests against the live Kind cluster UI
make test-live          # Bash script API tests against the live cluster
make test-all           # Runs all four tiers sequentially
```

System tests require a running Kind cluster (`./scripts/kind-deploy.sh deploy`). Live tests exercise the API directly with curl-based scripts covering containment, links, validation, publishing, copy/replace, and more.

## Architecture Patterns

### Domain-Driven Layered Design

The dependency flow is strictly one-directional:

```
API Layer --> Service Layer --> Domain Layer <-- Infrastructure Layer
(handlers)   (business logic)  (models + interfaces)  (GORM implementation)
```

The domain layer has zero external dependencies. Services depend only on domain interfaces. The infrastructure layer implements those interfaces. The composition root (`cmd/api-server/main.go`) wires concrete implementations via constructor injection.

### Copy-on-Write Versioning

Every mutation to an entity type (adding/removing attributes or associations) creates a new `EntityTypeVersion`. All attributes and associations are copied to the new version. Past versions remain immutable, ensuring catalog versions pinned to older entity type versions see unchanged schemas.

### Type Definitions with Constraint Validation

Type definitions replace hardcoded types and enums. Each type definition has a base type (string, integer, number, boolean, date, url, enum, list, json) and versioned constraints (max_length, pattern, min/max, enum values, etc.). Attributes reference a specific type definition version. Validation occurs at the application layer.

### EAV Storage for Instance Data

Entity instances use Entity-Attribute-Value storage via the `instance_attribute_values` table. This allows dynamically defined entity types without schema changes. Values are stored in type-specific columns (`value_string`, `value_number`, `value_json`).

## How to Add a New Feature

A typical feature touches multiple layers. Follow this order:

1. **Domain model** -- Add or modify structs in `internal/domain/models/`. Keep these as pure Go with no external dependencies.

2. **Repository interface** -- Define the storage contract in `internal/domain/repository/`. Use method signatures that accept and return domain models.

3. **GORM implementation** -- Implement the repository interface in `internal/infrastructure/gorm/repository/`. Add GORM model structs in `internal/infrastructure/gorm/models/` with mapping functions to/from domain models.

4. **Mock repository** -- Add a mock in `internal/domain/repository/mocks/` for use in service tests.

5. **Service layer** -- Write the business logic in `internal/service/meta/` or `internal/service/operational/`. Use TDD: write tests first with mock repositories, then implement.

6. **API handler + DTOs** -- Add HTTP handlers in `internal/api/meta/` or `internal/api/operational/`. Define request/response DTOs. Register routes in the appropriate route group.

7. **UI components** -- Add React components in `ui/src/pages/` and `ui/src/components/`. Use PatternFly components. Add API client functions in `ui/src/api/`.

8. **Tests at every tier** -- Backend unit tests, browser tests for UI interactions, and live/system tests for end-to-end validation.

## Code Style

- **Go**: Standard Go formatting (`gofmt`). No global state. Errors are returned, not panicked. Use `golangci-lint` for linting.
- **TypeScript**: Strict mode enabled. PatternFly components for all UI elements. React Query for server state, React Context for UI state.
- **Testing**: 100% coverage is the target. Every new line must be covered. Every modified file must have its coverage measured. See `CLAUDE.md` for the full coverage policy.
