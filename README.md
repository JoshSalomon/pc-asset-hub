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
| UI | React, TypeScript, PatternFly | Two UIs served from a single build — Meta UI for schema administration, Operational UI for data browsing. |
| Database | PostgreSQL / SQLite | Source of truth. PostgreSQL for production, SQLite for development. |
| Operator | Go, operator-sdk | Manages hub installation. Reconciles AssetHub, CatalogVersion, and Catalog CRs. |

## Key Concepts

- **Entity Types** — dynamically defined asset categories (e.g., "MCP Server", "Model", "Tool") with custom attributes and associations
- **Catalog Versions** — immutable schema snapshots that pin specific entity type versions, with a lifecycle (development → testing → production)
- **Catalogs** — named data containers pinned to a catalog version, holding entity instances with attribute values
- **Associations** — typed relationships between entity types: containment (parent-child hierarchy), directional, and bidirectional
- **Validation** — on-demand schema validation checks required attributes, enum values, mandatory associations, and containment consistency
- **Publishing** — valid catalogs are published as K8s Custom Resources for external discovery, with write protection on published data
- **Copy & Replace** — staging workflow for updating published catalogs without downtime: copy, edit, validate, swap atomically

## Features

### Meta Layer (Schema Management)

- Entity type CRUD with copy-on-write versioning
- Attributes (string, number, enum) with required/optional, reordering, copy-from
- Associations (containment, directional, bidirectional) with UML-style cardinality
- Enum management with value lists
- Catalog version lifecycle (development → testing → production) with K8s CR generation
- UML entity diagram with interactive topology visualization

### Operational Layer (Data Management)

- Catalog CRUD with DNS-label naming and catalog version pinning
- Entity instance CRUD with dynamic attribute forms and optimistic locking
- Containment hierarchy (parent-child) and association link management
- On-demand validation with structured error reporting
- Catalog publishing with K8s Catalog CRs and write protection
- Copy & Replace for atomic catalog updates with archive and rollback
- Per-catalog RBAC via K8s SubjectAccessReview

### UIs

- **Meta UI** (`http://localhost:30000/`) — schema administration: entity types, attributes, associations, enums, catalog versions, catalog management (CRUD, validation, publishing, copy & replace)
- **Operational UI** (`http://localhost:30000/operational`) — read-only data viewer: containment tree browser, instance detail with attributes, reference navigation with breadcrumbs

## Getting Started

See [DEPLOYMENT.md](DEPLOYMENT.md) for full deployment instructions.

Quick start on a local kind cluster:

```bash
./scripts/kind-deploy.sh deploy "kubectl --context kind-assethub"
```

This builds all images, creates a kind cluster, and deploys the full stack. Once complete:

- **API server:** http://localhost:30080
- **Meta UI:** http://localhost:30000
- **Operational UI:** http://localhost:30000/operational

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
make test-system      # System tests against live kind cluster
make test-live        # Live API tests (bash scripts)
make test-all         # All of the above
```

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
│   │   └── operational/     # Operational API handlers
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
│       └── validation/      # Cardinality validation
├── ui/
│   └── src/
│       ├── api/             # API client
│       ├── components/      # Shared components
│       ├── hooks/           # Shared hooks
│       └── pages/
│           ├── meta/        # Meta UI pages
│           └── operational/ # Operational UI pages
├── deploy/                  # K8s manifests
├── scripts/                 # Build, deploy, and test scripts
├── docs/                    # Architecture, test plans, coverage
├── PRD.md                   # Product requirements
└── DEPLOYMENT.md            # Deployment guide
```

## Test Coverage

| Layer | Tests | Coverage |
|-------|-------|----------|
| Backend (Go) | 1262 | 94.0% |
| UI Browser (Playwright) | 453 | 84.6% |
| Live System (bash) | 81 | — |
| System (Playwright + live) | 30 | — |
| **Total** | **1901** | |

## Documentation

| Document | Description |
|----------|-------------|
| [PRD.md](PRD.md) | Product requirements, user stories, acceptance criteria, technical debt |
| [DEPLOYMENT.md](DEPLOYMENT.md) | Deployment guide for kind and OpenShift clusters |
| [docs/architecture.md](docs/architecture.md) | System architecture, data model, layered design, technology stack |
| [docs/test-plan.md](docs/test-plan.md) | Testing strategy, coverage matrix, cross-cutting test approaches |
| [docs/test-plan-detailed.md](docs/test-plan-detailed.md) | Detailed test cases with IDs, layers, and expected outcomes |
| [docs/coverage-report.md](docs/coverage-report.md) | Per-package coverage numbers, uncovered lines, test counts |
| [docs/plans/](docs/plans/) | Implementation design documents |

## Roles

| Role | Permissions |
|------|------------|
| RO | Read all data |
| RW | Read + create/update/delete instances and catalogs |
| Admin | RW + publish/unpublish catalogs, promote/demote catalog versions, replace catalogs |
| SuperAdmin | Admin + edit published catalogs (bypasses write protection) |

## License

See [LICENSE](LICENSE).
