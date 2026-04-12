# AI Asset Hub — Architecture Document

## 1. Overview

The AI Asset Hub is a metadata-driven management system for AI assets deployed on OpenShift clusters. It is a component of Project Catalyst. The system manages assets such as models, MCP servers, tools, guardrails, evaluators, and prompts through a dynamically configurable schema layer — entity types are not hardcoded but defined at runtime via configuration.

This document describes the architecture decisions, technology choices, data model, and project structure for the system.

---

## 2. System Architecture

The system consists of four major components:

```
┌──────────────────────────────────────────────────────────┐
│                     OpenShift Cluster                    │
│                                                          │
│  ┌───────────┐     ┌──────────────┐     ┌──────────────┐ │
│  │    UI     │───▶│  API Server  │───▶│  PostgreSQL  │ │
│  │ (React +  │     │  (Go/Echo)   │     │  (or SQLite) │ │
│  │PatternFly)│     │      │       │     │              │ │
│  └───────────┘     └──────┼───────┘     └──────────────┘ │
│                           │                              │
│                           │ on promote/demote            │
│                           ▼                              │
│                   CatalogVersion CRs                     │
│                           │                              │
│                           │ watches                      │
│                           ▼                              │
│                   ┌──────────────┐                       │
│                   │   Operator   │──▶owner refs, status │
│                   │(operator-sdk)│                       │
│                   └──────────────┘                       │
│                                                          │
└──────────────────────────────────────────────────────────┘
```

- **UI**: React + PatternFly single-page application. A unified SPA serves all views: landing page (`/`), schema management (`/schema/*`), and catalog data viewer (`/catalogs/:name`). Communicates exclusively through the API server. Never accesses the database or cluster directly.
- **API Server**: Go backend exposing two API sets (Meta API and Operational API). Creates/updates/deletes `CatalogVersion` CRs on catalog version promotion and demotion. Enforces RBAC via OpenShift SubjectAccessReview.
- **Database**: PostgreSQL (production) or SQLite (development). Source of truth for all data.
- **Operator**: Built with operator-sdk. Manages hub installation. Watches `CatalogVersion` CRs, sets owner references to the AssetHub CR for garbage collection, and updates status conditions.

---

## 3. Technology Stack

| Component | Technology | Rationale |
|-----------|-----------|-----------|
| Backend language | **Go** | operator-sdk requires Go; single language for operator + API server eliminates build complexity. First-class K8s client libraries. Static binary for small container images. |
| Web framework | **Echo** (labstack/echo) | Route grouping cleanly separates Meta and Operational APIs. Idiomatic Go error handling (not panic-based like Gin). Built-in validation, binding, and middleware. |
| ORM | **GORM** | Supports both PostgreSQL and SQLite with build-tag driver switching. Handles migrations, transactions, soft deletes, hooks. |
| Production DB | **PostgreSQL** | Full relational capabilities for the meta schema, EAV queries, and version history. |
| Development DB | **SQLite** | Lower footprint for local development. Same GORM application code via build tags. |
| UI framework | **React + TypeScript** | Largest ecosystem, strong typing for complex form state. |
| UI component library | **PatternFly** | Red Hat's design system — visual consistency with OpenShift console. Enterprise-ready data tables, forms, modals, drag-and-drop. |
| Graph visualization | **@patternfly/react-topology** | Node-edge diagrams for the association map. Multiple layouts (Force, Dagre, Cola), interactive, zoom/pan. |
| UI state management | **React Query + React Context** | React Query for server state (caching, refetching). Context for UI state (role, catalog version). No Redux. |
| UI build tool | **Vite** | Fast development builds, TypeScript compilation, HMR. |
| ID generation | **UUID v7** | Time-ordered (B-tree friendly), no collision risk, portable across PostgreSQL/SQLite. Go library: `google/uuid`. |
| API specification | **OpenAPI 3.0** | Auto-generated documentation, TypeScript client generation, contract testing. |
| Operator framework | **operator-sdk** | Standard OpenShift operator tooling. |

---

## 4. Layered Architecture

The backend follows a strict layered architecture with clean separation between domain logic and infrastructure.

### Dependency Flow

```
API Layer  ──▶  Service Layer  ──▶  Domain Layer  ◀──  Infrastructure Layer
(handlers)      (business logic)    (models +          (GORM implementation)
                                     interfaces)
```

The domain layer has **zero external dependencies** — it defines pure Go models and repository interfaces. The infrastructure layer implements those interfaces using GORM. The service layer depends only on domain interfaces, never on GORM or any storage technology. The composition root (`cmd/`) wires the concrete implementation via dependency injection.

### Layer Boundary Rules

| Layer | Can import from | Cannot import from |
|-------|----------------|-------------------|
| `domain/` | Standard library only | `service/`, `infrastructure/`, `api/`, any external packages |
| `service/` | `domain/` | `infrastructure/`, `api/` |
| `api/` | `service/`, `domain/` | `infrastructure/` |
| `infrastructure/gorm/` | `domain/`, GORM, database drivers | `service/`, `api/` |
| `cmd/` (composition root) | Everything | — |

### Why This Matters

This separation ensures that the storage backend is pluggable. Swapping from GORM/PostgreSQL to MongoDB, etcd, or any other store requires only implementing the `domain/repository/` interfaces. No changes to the service, API, or UI layers. This is critical for reusability — the component can be plugged into other projects with different storage backends.

---

## 5. Project Structure

```
pc-asset-hub/
  cmd/
    api-server/              # Backend API server entrypoint (composition root)
    operator/                # Operator entrypoint
  internal/
    domain/                  # Domain layer (NO external dependencies)
      models/                # Domain model structs (pure Go, no GORM tags)
      repository/            # Repository interfaces (storage-agnostic)
      errors/                # Domain-specific error types
    service/                 # Service layer (depends on domain only)
      meta/                  # Meta business logic (entity types, attrs, assocs, type defs, catalog)
      operational/           # Operational business logic (instances, queries, refs)
      versioning/            # Version management (auto-increment, copy-on-write)
      validation/            # Cycle detection, uniqueness, constraint enforcement
    infrastructure/          # Infrastructure layer (implements domain interfaces)
      gorm/                  # GORM implementation
        models/              # GORM model structs (with tags, mapping to/from domain)
        repository/          # GORM repository implementations
        migrations/          # Database migration logic
      k8s/                   # K8s CRManager implementation (CatalogVersion CR management)
      config/                # Configuration loading
    api/                     # API layer (depends on service layer)
      meta/                  # Meta API handlers
      operational/           # Operational API handlers
      middleware/            # Auth, RBAC, logging, error handling
      dto/                   # Request/response DTOs
    operator/                # Operator logic
      api/v1alpha1/          # CRD types: AssetHub and CatalogVersion
      controllers/           # Reconciler implementations
      crdgen/                # CRD/CR generation from entity types (future scope)
  pkg/
    types/                   # Shared type definitions (cross-cutting)
  ui/
    src/
      api/                   # TypeScript API client
      components/            # Reusable UI components
      pages/
        LandingPage.tsx      # Landing page (root URL)
        meta/                # Schema management pages (entity types, CVs, type definitions)
        operational/         # Catalog data viewer pages + catalog CRUD
      hooks/                 # Custom React hooks
      utils/                 # Shared utilities
      types/                 # TypeScript types
  config/
    operator/                # Operator bundle, CRDs, RBAC manifests
  docs/                      # Documentation
  test/                      # Integration/E2E test infrastructure
```

---

## 6. API Design

### Two API Sets

The system exposes two distinct API sets with different audiences, middleware chains, and authorization requirements.

**Meta API** (`/api/meta/v1/...`)
Manages the schema layer. Used by Admins and Super Admins.
- Entity type definitions (CRUD, copy)
- Attribute management (add, edit, remove, copy, reorder)
- Association management (with cycle detection)
- Type definition management (CRUD, versioning, reusable across types — replaces enums)
- Catalog version management (create, promote, demote)
- Version history and comparison

**Operational API** (`/api/data/v1/...`)
Manages catalogs and entity instances. Used by all roles. Scoped to a specific catalog.
- Catalog CRUD (create, list, get, delete)
- Entity instance CRUD with auto-versioning (within a catalog)
- Containment traversal via sub-resource URLs
- Forward and reverse reference queries
- Filtering, sorting, pagination

### URL Structure

```
Meta API:
  /api/meta/v1/entity-types
  /api/meta/v1/entity-types/{id}
  /api/meta/v1/entity-types/{id}/attributes
  /api/meta/v1/entity-types/{id}/associations
  /api/meta/v1/entity-types/{id}/versions
  /api/meta/v1/entity-types/{id}/versions/{v1}/compare/{v2}
  /api/meta/v1/entity-types/{id}/versions/{version}/snapshot
  /api/meta/v1/type-definitions
  /api/meta/v1/type-definitions/{id}
  /api/meta/v1/type-definitions/{id}/versions
  /api/meta/v1/type-definitions/{id}/versions/{v}
  /api/meta/v1/catalog-versions
  /api/meta/v1/catalog-versions/{id}
  /api/meta/v1/catalog-versions/{id}/pins
  /api/meta/v1/catalog-versions/{id}/pins/{pin-id}
  /api/meta/v1/catalog-versions/{id}/type-pins
  /api/meta/v1/catalog-versions/{id}/type-pins/{pin-id}
  /api/meta/v1/catalog-versions/{id}/promote
  /api/meta/v1/catalog-versions/{id}/demote

Operational API (catalog-name is DNS-label: [a-z0-9-], max 63 chars):
  /api/data/v1/catalogs
  /api/data/v1/catalogs/{catalog-name}
  /api/data/v1/catalogs/{catalog-name}/{entity-type}
  /api/data/v1/catalogs/{catalog-name}/{entity-type}/{instance-id}
  /api/data/v1/catalogs/{catalog-name}/{entity-type}/{instance-id}/{contained-type}
  /api/data/v1/catalogs/{catalog-name}/{entity-type}/{instance-id}/{contained-type}/{name}
  /api/data/v1/catalogs/{catalog-name}/{entity-type}/{instance-id}/references
  /api/data/v1/catalogs/{catalog-name}/{entity-type}/{instance-id}/references/{ref-type}
  /api/data/v1/catalogs/{catalog-name}/validate
```

### Catalog Scoping

Every operational API call is scoped to a **catalog** via the URL path. A catalog is a named collection of entity instances pinned to a specific catalog version (CV). The CV determines the schema (which entity types, attributes, and associations are available); the catalog holds the actual data. Multiple catalogs can share the same CV. This ensures consumers interact with a named, consistent data set backed by a specific schema.

---

## 7. Data Model

The database uses a hybrid approach: **fixed relational tables** for the meta/schema layer and **EAV (Entity-Attribute-Value) tables** for the dynamic data layer.

### Meta Tables

```
┌──────────────┐        ┌──────────────────────┐        ┌──────────────┐
│ entity_types │──1:N─▶│ entity_type_versions │◀─N:1──│  attributes  │
│              │        │                      │        │              │
│ id           │        │ id                   │        │ id           │
│ name (unique)│        │ entity_type_id (FK)  │        │ etv_id (FK)  │
│ created_at   │        │ version              │        │ name         │
│ updated_at   │        │ description          │        │ description  │
└──────────────┘        │ created_at           │        │ type_def_ver │
                        │ UNIQUE(et_id, ver)   │        │   _id (FK)   │
                        └──────────┬───────────┘        │ ordinal      │
                                   │                    │ required     │
                                   │                    └──────────────┘
                                   │
                            ┌──────┴───────┐
                            │ associations │
                            │              │
                            │ id           │
                            │ etv_id (FK)  │
                            │ target_et_id │
                            │ type         │
                            │ source_role  │
                            │ target_role  │
                            └──────────────┘

┌──────────────────┐        ┌────────────────────────┐
│type_definitions  │──1:N─▶│type_definition_versions│
│                  │        │                        │
│ id               │        │ id                     │
│ name (unique)    │        │ type_def_id (FK)       │
│ description      │        │ version_number         │
│ base_type        │        │ constraints (JSONB)    │
│ system           │        └────────────────────────┘
│ created_at       │
│ updated_at       │
└──────────────────┘

┌──────────────────┐        ┌────────────────────┐      ┌───────────────────────┐
│ catalog_versions │──1:N─▶│catalog_version_pins│      │lifecycle_transitions  │
│                  │        │ (entity types)     │      │                       │
│ id               │        │ id                 │      │ id                    │
│ version_label    │──1:N─▶│ catalog_ver_id(FK) │      │ catalog_ver_id (FK)   │
│ lifecycle_stage  │        │ etv_id (FK)        │      │ from_stage            │
│ created_at       │        └────────────────────┘      │ to_stage              │
│ updated_at       │                                    │ performed_by          │
│                  │        ┌────────────────────────┐  │ performed_at          │
│                  │──1:N─▶│catalog_version_type_pins│ │ notes                 │
│                  │        │ (type definitions)     │  └───────────────────────┘
│                  │        │ id                     │
│                  │        │ catalog_ver_id (FK)    │
│                  │        │ type_def_ver_id (FK)   │
│                  │        └────────────────────────┘
└──────────────────┘
```

### Data Tables

```
┌──────────────────┐
│    catalogs      │
│                  │
│ id               │
│ name (unique)    │
│ description      │
│ catalog_ver_id   │──FK──▶ catalog_versions.id
│ validation_status│        (draft|valid|invalid)
│ created_at       │
│ updated_at       │
└────────┬─────────┘
         │
         │ 1:N
         ▼
┌──────────────────────┐        ┌───────────────────────────┐
│  entity_instances    │──1:N─▶│ instance_attribute_values │
│                      │        │                           │
│ id                   │        │ id                        │
│ entity_type_id (FK)  │        │ instance_id (FK)          │
│ catalog_id (FK)      │        │ instance_version          │
│ parent_inst_id (FK)  │        │ attribute_id (FK)         │
│ name                 │        │ value_string              │
│ description          │        │ value_number              │
│ version              │        │ value_enum                │
│ created_at           │        └───────────────────────────┘
│ updated_at           │
│ deleted_at           │        ┌───────────────────┐
│                      │        │ association_links │
│ UNIQUE(et_id,        │        │                   │
│  cat_id, parent,name)│──────▶│ id                │
└──────────────────────┘        │ association_id    │
                                │ source_inst_id    │
                                │ target_inst_id    │
                                │ created_at        │
                                └───────────────────┘
```

### Schema Design Details

```sql
-- Entity type definitions (identity)
entity_types (
  id UUID PK,
  name TEXT UNIQUE NOT NULL,
  created_at TIMESTAMP,
  updated_at TIMESTAMP
)

-- Immutable version snapshots
entity_type_versions (
  id UUID PK,
  entity_type_id UUID FK → entity_types.id,
  version INTEGER NOT NULL,
  description TEXT,
  created_at TIMESTAMP,
  UNIQUE(entity_type_id, version)
)

-- Attributes belong to a specific entity type version
attributes (
  id UUID PK,
  entity_type_version_id UUID FK → entity_type_versions.id,
  name TEXT NOT NULL,
  description TEXT,
  type_definition_version_id UUID FK → type_definition_versions.id NOT NULL,
  ordinal INTEGER NOT NULL,
  required BOOLEAN DEFAULT FALSE,
  UNIQUE(entity_type_version_id, name)
)

-- Associations belong to a specific entity type version
associations (
  id UUID PK,
  entity_type_version_id UUID FK → entity_type_versions.id,
  target_entity_type_id UUID FK → entity_types.id,
  type TEXT NOT NULL,          -- 'containment', 'directional', 'bidirectional'
  source_role TEXT,
  target_role TEXT,
  created_at TIMESTAMP
)

-- Reusable type definitions (replaces enums)
type_definitions (
  id UUID PK,
  name TEXT UNIQUE NOT NULL,
  description TEXT,
  base_type TEXT NOT NULL,    -- 'string', 'integer', 'number', 'boolean', 'date', 'url', 'enum', 'list', 'json'
  system BOOLEAN DEFAULT FALSE,
  created_at TIMESTAMP,
  updated_at TIMESTAMP
)

-- Versioned constraints for type definitions
type_definition_versions (
  id UUID PK,
  type_definition_id UUID FK → type_definitions.id,
  version_number INTEGER NOT NULL,
  constraints JSONB,          -- type-specific constraints (max_length, pattern, min/max, values, etc.)
  UNIQUE(type_definition_id, version_number)
)

-- Catalog version snapshots
catalog_versions (
  id UUID PK,
  version_label TEXT UNIQUE NOT NULL,
  lifecycle_stage TEXT NOT NULL DEFAULT 'development',
  created_at TIMESTAMP,
  updated_at TIMESTAMP
)

-- Pins entity type versions to a catalog version
catalog_version_pins (
  id UUID PK,
  catalog_version_id UUID FK → catalog_versions.id,
  entity_type_version_id UUID FK → entity_type_versions.id,
  UNIQUE(catalog_version_id, entity_type_version_id)
)

-- Pins type definition versions to a catalog version
catalog_version_type_pins (
  id UUID PK,
  catalog_version_id UUID FK → catalog_versions.id,
  type_definition_version_id UUID FK → type_definition_versions.id,
  UNIQUE(catalog_version_id, type_definition_version_id)
)

-- Audit trail for lifecycle transitions
lifecycle_transitions (
  id UUID PK,
  catalog_version_id UUID FK → catalog_versions.id,
  from_stage TEXT,
  to_stage TEXT NOT NULL,
  performed_by TEXT NOT NULL,
  performed_at TIMESTAMP NOT NULL,
  notes TEXT
)

-- Catalogs (named data collections pinned to a CV)
catalogs (
  id UUID PK,
  name TEXT UNIQUE NOT NULL,
  description TEXT,
  catalog_version_id UUID FK → catalog_versions.id NOT NULL,
  validation_status TEXT NOT NULL DEFAULT 'draft',  -- draft | valid | invalid
  created_at TIMESTAMP,
  updated_at TIMESTAMP
)

-- Entity instances (EAV pattern)
entity_instances (
  id UUID PK,
  entity_type_id UUID FK → entity_types.id,
  catalog_id UUID FK → catalogs.id,
  parent_instance_id UUID FK → entity_instances.id NULLABLE,
  name TEXT NOT NULL,
  description TEXT,
  version INTEGER NOT NULL DEFAULT 1,
  created_at TIMESTAMP,
  updated_at TIMESTAMP,
  deleted_at TIMESTAMP NULLABLE,
  UNIQUE(entity_type_id, catalog_id, parent_instance_id, name)
)

-- Attribute values per instance version
instance_attribute_values (
  id UUID PK,
  instance_id UUID FK → entity_instances.id,
  instance_version INTEGER NOT NULL,
  attribute_id UUID FK → attributes.id,
  value_string TEXT,           -- string, url, date, boolean, enum
  value_number DOUBLE,         -- number, integer
  value_json TEXT,             -- list (JSON array), json (JSON object)
  UNIQUE(instance_id, instance_version, attribute_id)
)

-- Links between entity instances via associations
association_links (
  id UUID PK,
  association_id UUID FK → associations.id,
  source_instance_id UUID FK → entity_instances.id,
  target_instance_id UUID FK → entity_instances.id,
  created_at TIMESTAMP
)
```

### Key Design Decisions in the Schema

**Copy-on-write versioning**: `entity_type_versions` holds immutable snapshots. When an entity type is mutated, a new version row is created and all attributes and associations are copied to the new version. Past versions remain intact. Catalog versions pinning an older entity type version continue to see that version's attributes and associations unchanged.

**Entity type description is versioned**: The `entity_types` table has no `description` column. The description lives on `entity_type_versions.description`, making it part of the versioned snapshot. When the API returns an entity type list or detail, the handler resolves the description from the latest version via `GetLatestByEntityType`. This means updating a description creates a new version (COW), which is the intended behavior — the description change is tracked in version history.

**Associations are versioned**: Associations are tied to `entity_type_versions`, not `entity_types`. This ensures that adding or removing an association only affects the new version. Without this, modifying an association would retroactively affect older catalog versions that reference previous entity type versions.

**EAV for dynamic data**: The `instance_attribute_values` table stores attribute values using type-specific columns (`value_string`, `value_number`, `value_json`). `value_string` stores string, url, date, boolean, and enum values. `value_number` stores number and integer values. `value_json` stores list (JSON array) and json (JSON object) values. The structure of this table never changes when entity types are defined — validation of values against the type definition happens at the application layer.

**Type definitions replace enums**: Enums are unified under the type definition system as type definitions with `base_type=enum` and an ordered list of values in their constraints. Type definitions are versioned (copy-on-write) and pinned in catalog versions, solving the enum mutation problem (TD-58) where changing enum values retroactively affected all CVs.

**Instance version history**: `instance_attribute_values` includes `instance_version`, so every historical state of an instance's attributes is preserved.

**Catalog as data container**: Entity instances belong to a `Catalog`, not directly to a `CatalogVersion`. The catalog is pinned to a CV at creation; the pin can be changed via re-pinning (`PUT /catalogs/{name}` with `catalog_version_id`), which resets validation status to `draft`. This separates schema (CV) from data (catalog) and allows multiple named datasets to share the same schema.

**Containment via self-reference**: `parent_instance_id` on `entity_instances` models the containment hierarchy. Name uniqueness is scoped to `(entity_type_id, catalog_id, parent_instance_id, name)`, enforcing the namespace rule.

**Soft deletes**: `deleted_at` on `entity_instances` supports audit trails while allowing cascade delete logic at the application layer.

---

## 8. Versioning Model

### Four Levels of Versioning

```
Catalog Version (bill of materials)
  │
  ├── Entity Type Pins:
  │   ├── Entity Type A (V1)
  │   │     ├── attribute: name (string)
  │   │     ├── attribute: guardrail_id (guardrailID V1)
  │   │     └── association: contains → Tool
  │   ├── Entity Type B (V2)
  │   │     ├── attribute: endpoint (url)
  │   │     └── attribute: max_tokens (integer)
  │   └── Entity Type C (V1)
  │         └── attribute: config (json)
  │
  └── Type Definition Pins:
      ├── guardrailID (V1) — string, max_length: 12, pattern: [0-9A-F]*
      └── statusEnum (V1) — enum, values: [active, inactive, deprecated]
```

1. **Type definition versioning**: Every mutation to a type definition's constraints creates a new version. Previous versions are retained. System type definitions (string, integer, number, boolean, date, url) are immutable V1.

2. **Entity definition versioning**: Every mutation to an entity type definition (add/remove/change attributes or associations) creates a new version automatically. Previous versions are immutable. Unique key: `(name, version)`.

3. **Entity instance versioning**: Every mutation to an entity instance auto-increments its version. Previous attribute values are retained per version.

4. **Catalog versioning**: A catalog version pins specific entity definition versions and type definition versions together as a bill of materials. Pins can be added, removed, or changed. Each entity type can appear at most once in a CV. Custom type definitions used by pinned entity type attributes are auto-pinned. Deployments reference a fixed catalog version.

### Lifecycle States

Each catalog version progresses through:

```
              RW+              Admin+            
Development  ─────▶  Testing  ─────▶  Production
     ◀──────────     ◀──────────────────────
           RW+              Super Admin
```

- **Development → Testing**: RW and above can promote. RW and above can demote back.
- **Testing → Production**: Admin and above can promote.
- **Production → Testing/Development**: Super Admin only can demote.

Stage descriptions:
- **Development**: Active editing in the database. No CRs created in K8s. Work-in-progress via UI.
- **Testing**: A `CatalogVersion` CR is created in K8s for discovery. Applications find available catalog versions via the K8s API.
- **Production**: The `CatalogVersion` CR is updated to production stage. Frozen — only Super Admin can modify or demote.

The `clusterRole` configuration on the AssetHub CR controls which lifecycle stages the API server exposes: `production` clusters only serve production catalog versions, `testing` clusters serve testing and production, and `development` clusters (default) serve all stages.

---

## 9. Concurrency Model

**Optimistic locking with version-based conflict detection.**

Every mutable entity carries a version number (already mandated by the PRD). Update requests include the expected current version. The server rejects updates where the version has changed since the client's last read.

```
Client A reads Entity X (version 3)
Client B reads Entity X (version 3)
Client A updates Entity X with version=3 → succeeds, version becomes 4
Client B updates Entity X with version=3 → fails with 409 Conflict
Client B refreshes, reads version 4, retries
```

Implementation: `UPDATE ... WHERE id = ? AND version = ?`. Zero rows affected returns HTTP 409.

The UI handles 409 responses with a conflict notification and data refresh.

---

## 10. Access Control

The system leverages OpenShift's native RBAC. No custom user database or authentication system.

### Roles

| Role | Meta API | Operational API | Lifecycle |
|------|----------|----------------|-----------|
| **RO** | GET only | GET only | — |
| **RW** | GET only | Full CRUD | Create catalog version (dev). Promote dev→test. Demote test→dev. |
| **Admin** | Full access (non-production) | Full CRUD | All RW lifecycle permissions. Promote test→production. |
| **Super Admin** | Full access (including production) | Full CRUD | All Admin lifecycle permissions. Demote from production (to test or dev). |

### Implementation

- Authentication: Extract identity from request (ServiceAccount token or Bearer token via OpenShift OAuth).
- Authorization: SubjectAccessReview via `k8s.io/client-go` maps the authenticated user to application roles.
- The API middleware enforces role checks before handlers execute.
- Development mode: Configurable mock RBAC for local development outside a cluster.

---

## 11. Operator Architecture

Built with **operator-sdk**. Manages two concerns:

### Hub Installation (AssetHub CRD)

A single `AssetHub` custom resource configures the hub installation:
- Database connection settings
- Resource limits and replicas
- Feature flags

The operator watches this CR and reconciles the backend Deployment, Service, UI Deployment, and database setup.

### Catalog Version Discovery (CatalogVersion CRs)

When a catalog version is promoted to Testing or Production, a lightweight `CatalogVersion` CR is created for discovery:

1. The API server `Promote()` updates the DB lifecycle stage.
2. The API server creates or updates a `CatalogVersion` CR in K8s with version label, description, lifecycle stage, and entity type names.
3. The operator watches `CatalogVersion` CRs, sets owner references to the AssetHub CR (enabling garbage collection), and updates status conditions.
4. On demotion to Development, the API server deletes the `CatalogVersion` CR (development-stage versions don't exist in K8s).
5. On demotion from Production to Testing, the API server updates the CR with the new lifecycle stage.

The database remains the source of truth. `CatalogVersion` CRs are discovery artifacts — lightweight projections enabling applications to find available catalog versions via the K8s API.

### Entity Type CRDs (Future Scope)

Full schema-as-CRD artifacts — where entity type definitions become native K8s CRDs — are a separate feature planned for a future phase. The `crdgen/` package contains the generation logic for this future capability.

---

## 12. Future Extensibility: Entity Type Inheritance

The architecture accommodates entity type inheritance as an additive change if needed in the future. This would allow entity type B to extend entity type A, inheriting A's attributes and associations.

**Required changes (all additive, no rearchitecture):**
- Add a nullable `parent_entity_type_version_id` FK on `entity_type_versions` — one column.
- Attribute resolution walks up the inheritance chain (own + inherited attributes). The repository interface doesn't change — only the implementation.
- Cycle detection extends to the inheritance hierarchy (same algorithm pattern as containment).
- API adds optional `?includeSubtypes=true` for instance listings.
- UI distinguishes inherited vs. own attributes visually.

The clean layer separation, copy-on-write versioning, and EAV data tables all accommodate this without structural changes.
