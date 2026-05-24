# AI Asset Hub

> **Work in Progress.** This project is under active development and currently runs only on local [kind](https://kind.sigs.k8s.io/) clusters. It is not yet ready for production use.

AI Asset Hub is a metadata-driven management system for AI assets deployed on OpenShift clusters. It is a component of [Project Catalyst](https://github.com/project-catalyst).

The system manages assets such as models, MCP servers, tools, guardrails, evaluators, and prompts — but the list of asset types is not hardcoded. Entity types, their attributes, and the associations between them are defined dynamically through a configuration layer, making the system extensible to any asset type without code changes.

## Architecture

```
                        OpenShift / kind Cluster
 ┌──────────────────────────────────────────────────────────┐
 │                                                          │
 │  ┌───────────┐     ┌──────────────┐     ┌──────────────┐ │
 │  │    UI     │────▶│  API Server  │────▶│  PostgreSQL  │ │
 │  │ React +   │     │   Go/Echo    │     │  (or SQLite) │ │
 │  │PatternFly │     │              │     │              │ │
 │  └───────────┘     └──────┬───────┘     └──────────────┘ │
 │   :30000                  │ :30080                       │
 │                           ▼                              │
 │                  CatalogVersion CRs ◀── Operator         │
 │                  Catalog CRs            (operator-sdk)   │
 │                                                          │
 └──────────────────────────────────────────────────────────┘
```

| Component | Technology | Description |
|-----------|-----------|-------------|
| API Server | Go, Echo, GORM | REST API with two API sets: Meta (schema management) and Operational (data management). RBAC via OpenShift SubjectAccessReview. |
| UI | React, TypeScript, PatternFly | Two UIs served from a single build — Meta UI for schema administration, Operational UI for data browsing and editing. |
| Database | PostgreSQL / SQLite | Source of truth. PostgreSQL for production, SQLite for development. |
| Operator | Go, operator-sdk | Manages hub installation. Reconciles AssetHub, CatalogVersion, and Catalog CRs. |

## Key Concepts

- **Entity Types** — dynamically defined asset categories (e.g., "MCP Server", "Model", "Tool") with custom attributes and associations
- **Type Definitions** — reusable data types (string, number, integer, boolean, url, date, enum, list, json) with constraints (max_length, pattern, min/max, enum values)
- **Catalog Versions** — immutable schema snapshots that pin specific entity type versions, with a lifecycle (development → testing → production)
- **Catalogs** — named data containers pinned to a catalog version, holding entity instances with attribute values
- **Associations** — typed relationships between entity types: containment (parent-child hierarchy), directional, and bidirectional
- **Validation** — on-demand schema validation checks required attributes, enum values, mandatory associations, and containment consistency
- **Publishing** — valid catalogs are published as K8s Custom Resources for external discovery, with write protection on published data
- **Copy & Replace** — staging workflow for updating published catalogs without downtime: copy, edit, validate, swap atomically
- **Export Plugins** — extensible system for producing consumer-specific output (K8s CRs, ConfigMaps, YAML) from catalog data via registered exporter plugins

## Features

### Meta Layer (Schema Management)

- Entity type CRUD with copy-on-write versioning
- 9 base types (string, number, integer, boolean, url, date, enum, list, json) with type-specific constraints
- Type definitions with versioning and constraint validation (max_length, pattern, min/max, enum values)
- Attributes with required/optional, reordering, copy-from, type-aware forms
- Associations (containment, directional, bidirectional) with UML-style cardinality
- Catalog version lifecycle (development → testing → production) with K8s CR generation
- UML entity diagram with interactive topology visualization and composition diamonds

### Operational Layer (Data Management)

- Catalog CRUD with DNS-label naming and catalog version pinning
- Entity instance CRUD with dynamic attribute forms, inline validation, and optimistic locking
- Instance names validated as Kubernetes resource names (DNS-1123)
- Containment hierarchy (parent-child) and association link management
- Full editing in the operational data viewer: create, edit, delete instances, manage containment and links
- On-demand validation with structured error reporting
- Catalog publishing with K8s Catalog CRs, write protection, and export plugin preview
- Copy & Replace for atomic catalog updates with archive and rollback
- Export/Import catalogs to portable JSON format (schema + data round-trip)
- Export plugins: attach exporter bindings to catalogs, run on demand or automatically on publish
- Built-in MCP Gateway CR exporter (MCPServerRegistration + MCPVirtualServer)
- Per-catalog RBAC via K8s SubjectAccessReview

### UIs

- **Meta UI** (`/schema`) — schema administration: entity types, attributes, associations, type definitions, catalog versions, catalog management (CRUD, validation, publishing, copy & replace, import/export, export plugins)
- **Operational UI** (`/catalogs/{name}`) — data viewer and editor: containment tree browser, instance detail with attributes, reference navigation, create/edit/delete instances, containment and link management, export plugins

## Getting Started

See [DEPLOYMENT.md](DEPLOYMENT.md) for full deployment instructions.

Quick start on a local kind cluster:

```bash
./scripts/kind-deploy.sh deploy "kubectl --context kind-assethub"
```

This builds all images, creates a kind cluster, and deploys the full stack. Once complete:

- **API server:** http://localhost:30080
- **Meta UI:** http://localhost:30000/schema
- **Operational UI:** http://localhost:30000/catalogs

### API Examples

```bash
# Create an entity type
curl -s -X POST http://localhost:30080/api/meta/v1/entity-types \
  -H 'Content-Type: application/json' -H 'X-User-Role: Admin' \
  -d '{"name": "mcp-server"}' | jq .

# Create a catalog
curl -s -X POST http://localhost:30080/api/data/v1/catalogs \
  -H 'Content-Type: application/json' -H 'X-User-Role: Admin' \
  -d '{"name": "my-catalog", "catalog_version_id": "<CV_ID>"}' | jq .

# Create an instance
curl -s -X POST http://localhost:30080/api/data/v1/catalogs/my-catalog/mcp-server \
  -H 'Content-Type: application/json' -H 'X-User-Role: Admin' \
  -d '{"name": "my-server", "attributes": {"hostname": "localhost"}}' | jq .

# List registered export plugins
curl -s http://localhost:30080/api/data/v1/exporters -H 'X-User-Role: Admin' | jq .
```

## Development

### Prerequisites

| Tool | Version |
|------|---------|
| Go | 1.25+ |
| Node.js | 22+ |
| Docker or Podman | Docker 24+ / Podman 4+ |
| kind | 0.20+ |
| kubectl | 1.28+ |

### Running Tests

```bash
make test-backend     # Go unit + integration tests (SQLite)
make test-browser     # UI browser tests (Playwright via Vitest)
make test-system      # Live browser tests against kind cluster (Playwright)
make test-e2e         # Alias for test-system
make test-live        # Live API tests (bash scripts)
make test-all         # All of the above
```

Run a single test file or a specific test:

```bash
cd ui && npx vitest run --config vitest.system.config.ts src/LandingPage.system.test.ts
cd ui && npx vitest run --config vitest.system.config.ts -t "role selector shows all 4 roles"
```

Watch live browser tests in a visible browser window:

```bash
HEADLESS=false make test-e2e                          # show browser
HEADLESS=false SLOWMO=500 make test-e2e               # show browser + slow motion (ms)
```

These env vars also work with single-file runs:

```bash
HEADLESS=false SLOWMO=300 npx vitest run --config vitest.system.config.ts src/CatalogDetail.system.test.ts
```

First-time setup for live browser tests: `scripts/install-playwright.sh`

### Coverage

```bash
make coverage-backend   # Go coverage report
make coverage-browser   # UI coverage report
```

### Build & Deploy

```bash
make build                    # Build Go binaries
make docker-build-all         # Build all container images

# Deploy to kind
./scripts/kind-deploy.sh deploy "kubectl --context kind-assethub"

# Rebuild after code changes
./scripts/kind-deploy.sh rebuild "kubectl --context kind-assethub"

# Teardown
./scripts/kind-deploy.sh teardown
```

## Project Structure

```
pc-asset-hub/
├── cmd/
│   ├── api-server/          # API server entrypoint
│   └── operator/            # Operator entrypoint
├── internal/
│   ├── api/
│   │   ├── dto/             # Request/response types
│   │   ├── meta/            # Meta API handlers
│   │   ├── middleware/      # RBAC, catalog access
│   │   └── operational/     # Operational API handlers (catalog, instance, export)
│   ├── domain/
│   │   ├── errors/          # Domain error types
│   │   ├── models/          # Domain models
│   │   └── repository/      # Repository interfaces
│   ├── infrastructure/
│   │   ├── gorm/            # GORM repository implementations
│   │   └── k8s/             # K8s CR managers
│   ├── operator/            # Operator controllers and CRD types
│   └── service/
│       ├── meta/            # Meta service layer
│       ├── operational/     # Operational service layer
│       │   └── export/      # Export plugin framework (registry, bindings, exporters)
│       └── validation/      # Cardinality validation
├── ui/
│   └── src/
│       ├── api/             # API client
│       ├── components/      # Shared components (modals, panels, forms)
│       ├── hooks/           # Shared hooks
│       └── pages/
│           ├── meta/        # Meta UI pages
│           └── operational/ # Operational UI pages
├── deploy/                  # K8s manifests
├── scripts/                 # Build, deploy, and test scripts
├── docs/                    # Architecture, test plans, coverage, design specs
├── PRD.md                   # Product requirements
└── DEPLOYMENT.md            # Deployment guide
```

## Test Coverage

| Layer | Tests | Coverage |
|-------|-------|----------|
| Backend (Go) | ~2100 | 97%+ |
| UI Browser (Playwright) | ~1274 | 95%+ |
| Live System (bash scripts) | ~539 | — |
| System (Playwright + live) | ~202 | — |
| **Total** | **~4100** | |

## Documentation

| Document | Description |
|----------|-------------|
| [PRD.md](PRD.md) | Product requirements, user stories, acceptance criteria, future features |
| [docs/td-log.md](docs/td-log.md) | Technical debt log (critical, normal, resolved) |
| [DEPLOYMENT.md](DEPLOYMENT.md) | Deployment guide for kind and OpenShift clusters |
| [docs/architecture.md](docs/architecture.md) | System architecture, data model, layered design, technology stack |
| [docs/test-plan.md](docs/test-plan.md) | Testing strategy, coverage matrix, cross-cutting test approaches |
| [docs/test-plan-detailed.md](docs/test-plan-detailed.md) | Detailed test cases with IDs, layers, and expected outcomes |
| [docs/coverage-report.md](docs/coverage-report.md) | Per-package coverage numbers, uncovered lines, test counts |
| [docs/superpowers/specs/](docs/superpowers/specs/) | Design specifications for major features |

## Roles

| Role | Permissions |
|------|------------|
| RO | Read all data |
| RW | Read + create/update/delete instances and catalogs |
| Admin | RW + publish/unpublish catalogs, promote/demote catalog versions, replace catalogs, manage export bindings |
| SuperAdmin | Admin + edit published catalogs (bypasses write protection) |

## License

See [LICENSE](LICENSE).
