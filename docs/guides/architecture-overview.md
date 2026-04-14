# Architecture Overview

The AI Asset Hub is a metadata-driven management system for AI assets deployed on OpenShift clusters. It is a component of Project Catalyst. The system manages assets such as models, MCP servers, tools, guardrails, evaluators, and prompts through a dynamically configurable schema layer -- entity types are not hardcoded but defined at runtime.

## System Architecture

The system consists of four components deployed within an OpenShift cluster.

```
+------------------------------------------------------------+
|                     OpenShift Cluster                       |
|                                                            |
|  +-----------+     +--------------+     +---------------+  |
|  |    UI     |---->|  API Server  |---->|  PostgreSQL   |  |
|  | (React +  |     |  (Go / Echo) |     |  (or SQLite)  |  |
|  |PatternFly)|     |      |       |     |               |  |
|  +-----------+     +------+-------+     +---------------+  |
|                           |                                |
|                           | on promote/demote              |
|                           v                                |
|                   CatalogVersion CRs                       |
|                           |                                |
|                           | watches                        |
|                           v                                |
|                   +--------------+                         |
|                   |   Operator   |-->owner refs, status     |
|                   |(operator-sdk)|                         |
|                   +--------------+                         |
|                                                            |
+------------------------------------------------------------+
```

- **UI**: React + PatternFly single-page application. Serves schema management views and an operational catalog data viewer. Communicates exclusively through the API server.
- **API Server**: Go backend built on the Echo framework. Exposes two API surfaces (Meta and Operational). Creates and manages CatalogVersion custom resources in Kubernetes on lifecycle transitions.
- **Database**: PostgreSQL in production, SQLite for local development. Source of truth for all schema and data.
- **Operator**: Built with operator-sdk. Manages hub installation via an AssetHub custom resource. Watches CatalogVersion CRs, sets owner references for garbage collection, and updates status conditions.

## Design Principles

**Schema-at-runtime.** Entity types, their attributes, and their associations are not hardcoded. Administrators define them through the Meta API or UI. The system stores and enforces structure without compile-time knowledge of what asset types exist.

**Copy-on-write versioning.** Mutating an entity type (adding an attribute, changing an association) creates a new immutable version. Previous versions remain intact. Catalog versions that pin an older entity type version continue to see that version's schema unchanged.

**EAV storage for instance data.** Instance attribute values are stored in a type-specific Entity-Attribute-Value table (`instance_attribute_values`) with columns for string, number, and JSON values. The table structure never changes when entity types are defined -- validation happens at the application layer.

**Catalog version pinning.** A catalog version acts as a bill of materials, pinning specific entity type versions and type definition versions. This guarantees reproducible schemas: a catalog pinned to CV "v1.0" always sees the same attributes and constraints regardless of later schema changes.

## Data Model

The data model is split into a relational meta layer and an EAV data layer.

### Meta Layer (Schema)

- **EntityType** -- Identity record with a unique name. Description is versioned (lives on EntityTypeVersion).
- **EntityTypeVersion** -- Immutable snapshot. Each mutation creates a new version. Carries description, version number, and timestamp.
- **Attribute** -- Belongs to an EntityTypeVersion. Defines name, ordinal, required flag, and references a TypeDefinitionVersion for its data type and constraints.
- **Association** -- Belongs to an EntityTypeVersion. Links to a target entity type with a type (containment, directional, bidirectional), roles, and cardinalities.
- **TypeDefinition** -- Reusable, named type with a base type (one of 9). System types are immutable.
- **TypeDefinitionVersion** -- Versioned constraints (JSONB) for a type definition. Constraints are type-specific (max_length, pattern, min, max, enum values, list element types).
- **CatalogVersion** -- Bill of materials with a lifecycle stage (development, testing, production). Pins entity type versions and type definition versions.

### Data Layer (Instances)

- **Catalog** -- Named data container pinned to a CatalogVersion. Has a validation status (draft, valid, invalid) and a published flag.
- **EntityInstance** -- Belongs to a catalog and an entity type. Supports containment hierarchy via `parent_instance_id`. Soft-deleted via `deleted_at`.
- **InstanceAttributeValue** -- EAV row storing a value for one attribute of one instance version. Columns: `value_string`, `value_number`, `value_json`.
- **AssociationLink** -- Connects two entity instances via an association definition.

## API Design

The system exposes two distinct API surfaces with separate middleware chains and authorization rules.

**Meta API** (`/api/meta/v1/`) manages the schema layer. Entity types, attributes, associations, and type definitions require Admin role for writes. Catalog version lifecycle operations require RW minimum, with production transitions requiring higher roles.

**Operational API** (`/api/data/v1/`) manages catalogs and entity instance data. All operations are scoped to a catalog via the URL path (e.g., `/api/data/v1/catalogs/{catalog-name}/{entity-type}`). Catalog names must be DNS-label compatible (lowercase alphanumeric with hyphens, max 63 characters).

Both APIs follow RESTful conventions with a DTO layer that separates internal domain models from request/response structures.

## Type System

The type system provides 9 base types, each with optional constraints:

| Base Type | Constraint Examples |
|-----------|-------------------|
| string | `max_length`, `multiline`, `pattern` (regex) |
| integer | `min`, `max` |
| number | `min`, `max` |
| boolean | (none) |
| date | (none) |
| url | (none) |
| enum | `values` (ordered list of allowed strings) |
| list | `element_base_type`, `max_length` |
| json | `schema` (optional JSON Schema) |

Six **system types** (string, integer, number, boolean, date, url) are seeded on startup with `system=true` and a single immutable version (V1). They cannot be modified or deleted. Custom type definitions are versioned with copy-on-write semantics and pinned in catalog versions, ensuring that constraint changes do not retroactively affect existing catalogs.

## Versioning Strategy

The system maintains four levels of versioning:

1. **Type definition versioning** -- Mutating constraints creates a new version. System types are immutable at V1.
2. **Entity type versioning** -- Adding, removing, or changing attributes or associations creates a new version via copy-on-write. All attributes and associations are copied to the new version.
3. **Instance versioning** -- Each mutation to an entity instance increments its version. Previous attribute values are retained per version for history.
4. **Catalog versioning** -- A catalog version pins specific entity type versions and type definition versions. Lifecycle stages progress from development through testing to production, with role-gated transitions.

## Layered Architecture

The backend follows strict layer separation:

```
API Layer  -->  Service Layer  -->  Domain Layer  <--  Infrastructure Layer
(handlers)     (business logic)    (models +          (GORM implementation)
                                    interfaces)
```

The domain layer has zero external dependencies. Repository interfaces are defined in the domain; the infrastructure layer provides GORM implementations. The service layer depends only on domain interfaces. The composition root (`cmd/api-server/main.go`) wires concrete implementations via dependency injection. This makes the storage backend pluggable -- swapping databases requires only new repository implementations.

## Testing Strategy

The project targets 100% code coverage and employs multiple testing layers:

- **Unit tests** -- Service and handler logic tested with mock repositories (testify/mock). Fast, isolated, run on every change.
- **Integration tests** -- Repository implementations tested against SQLite. Verify SQL correctness and GORM behavior.
- **Browser tests** -- UI components and pages tested with Playwright via Vitest. PatternFly interactions, form validation, navigation flows.
- **System tests** -- Go test suite running against a live Kind cluster deployment. End-to-end API validation.
- **Bash script API tests** -- Shell scripts exercising operational flows (containment, links, data viewer, validation, publishing, copy/replace) against a running API server.

## Quality Assurance

- **Coverage reports** maintained in `docs/coverage-report.md` with per-file metrics.
- **Quality reviews** performed by code-reviewer agents before merging.
- **Technical debt** tracked in `docs/td-log.md` with severity ratings and deferred-item identifiers (TD-nn).
- **Coverage tooling**: `scripts/coverage-summary.sh` for overall metrics, `scripts/uncovered-new-lines.sh` and `scripts/uncovered-new-lines-ui.sh` for new-line checks, `scripts/analyze-coverage.sh` for UI coverage analysis.
