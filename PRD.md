# AI Asset Hub — Product Requirements Document

## 1. Overview

The AI Asset Hub is a metadata-driven management system for AI assets deployed on OpenShift clusters. It is a component of Project Catalyst.

The system manages assets such as models, MCP servers and tools, guardrails and evaluators, and prompts — but the list of asset types is not hardcoded. Entity types, their attributes, and the associations between them are defined dynamically through a configuration layer (the "meta repository"). This makes the system extensible to any future asset type without code changes.

## 2. Core Concepts

### 2.1 Meta Repository (Schema Layer)

The system does not have a fixed list of entity types. Instead, a configuration layer defines:

- **Entity types** (e.g., "Model", "MCP Server", "Tool", "Prompt")
- **Attributes** per entity type (beyond the common attributes)
- **Associations** between entity types (containment, directional reference, bidirectional reference)

This configuration is itself versioned and subject to lifecycle management.

### 2.2 Catalogs

A **catalog** is a named collection of entity instances that uses a catalog version (CV) as its schema. The CV determines which entity types, attributes, and associations are available; the catalog holds the actual data.

- Multiple catalogs can share the same CV (e.g., "Production App A" and "Staging App B" both use CV v2.0 but contain different assets).
- A catalog is pinned to a single CV at creation. In v1, this pin is immutable — to use a new CV, create a new catalog. Re-pinning (upgrading a catalog to a newer CV) is a future capability.
- Each catalog has a **validation status**: `draft` (incomplete, work in progress), `valid` (all constraints satisfied), or `invalid` (constraints violated, e.g., after a CV schema change).
- Entity instances are created freely in a catalog without immediate validation (draft mode). Validation runs on demand or at lifecycle gates.

### 2.3 Entity Instances (Data Layer)

Instances of the configured entity types, with values for their defined attributes. These are the actual AI assets being managed (e.g., a specific model called "llama-3-70b").

Entity instances belong to a **catalog** (not directly to a CV). The catalog's pinned CV determines the schema — which entity types exist, what attributes they have, and what association constraints apply.

### 2.4 Associations

Three types of associations can be defined between entity types:

#### Containment (part-of)
- B is part of A (equivalently, A contains B).
- B lives in the namespace and lifecycle of A.
- B can be created and deleted independently while A exists, but deleting A cascades to delete B.
- Containment can be multi-level (C contained in B contained in A).
- Containment **must not** form cycles (it is a DAG).
- Contained entities are namespaced by their parent — name uniqueness is scoped to the containing entity.

#### Directional Reference
- C refers to D. The forward direction (C → D) must be fast and immediate to resolve. The reverse direction (D → C, "referred by") can be slower and does not require a direct API path.

#### Bidirectional Reference
- E refers to F and F refers to E. Both directions must be fast to resolve.

### 2.5 Attributes

All entities share a set of **common attributes**:

| Attribute   | Description |
|-------------|-------------|
| ID          | System-calculated unique identifier |
| Name        | Unique within scope (global for top-level entities, within parent for contained entities) |
| Description | Free-text description |
| Version     | Auto-incremented on mutation |

Each entity type defines additional **custom attributes** via the meta configuration. Each custom attribute has:

| Property    | Description |
|-------------|-------------|
| ID          | System-assigned identifier |
| Name        | Attribute name |
| Description | Attribute description |
| Type        | Value type: string, number, or enum (closed list of allowed values) |

## 3. Versioning Model

### 3.1 Entity Definition Versioning

Every mutation to an entity type definition (in the meta repository) automatically increments its version. Previous versions are retained. Entity definitions are uniquely identified by `(name, version)`.

### 3.2 Entity Instance Versioning

Mutations to entity instances automatically increment the instance version. This provides an audit trail and reduces the risk of breaking backward compatibility.

### 3.3 Catalog Versioning

A **catalog version** is a snapshot that pins specific entity definition versions together — a bill of materials. For example:

- Catalog V1: Entity A (V1), Entity B (V1), Entity C (V1)
- Catalog V2: Entity A (V1), Entity B (V1), Entity C (V2), Entity D (V1)

Deployments reference a fixed catalog version. Once deployed, changes to the underlying definitions do not affect existing deployments.

### 3.4 Lifecycle

Each catalog version progresses through lifecycle stages:

1. **Development** — Active editing. Meta configuration is stored in the database only. No CRs are created in K8s. Development-stage catalog versions are mutable and managed entirely via the UI/API.
2. **Testing** — Catalog version is promoted. A lightweight `CatalogVersion` CR is created in K8s for discovery (applications find available catalog versions via the K8s API). Entity type CRDs (full schema as K8s resources) are a separate, future feature.
3. **Production** — Catalog version is deployed and frozen. The `CatalogVersion` CR is updated to reflect the production lifecycle stage.

Demoting a catalog version back to Development deletes its `CatalogVersion` CR from K8s.

When a catalog version is promoted, the system warns if any catalogs pinned to that CV have a validation status of `draft` or `invalid`. The user decides whether to proceed with promotion or fix the catalogs first.

### 3.5 Catalog Validation

Entity instances in a catalog are created in **draft mode** — missing required attributes, missing mandatory associations (cardinality `1` or `1..n`), and incomplete containment are allowed. This supports incremental data entry.

**Validation** checks all entity instances in a catalog against the pinned CV's schema:
- All required attributes have values
- Attribute values match their type (string, number, valid enum value)
- All mandatory associations are satisfied (cardinality constraints met)
- Containment hierarchy is consistent (no orphaned contained entities)

Validation can be triggered:
- **On demand** — user explicitly validates a catalog (API or UI)
- **On CV promotion** — system warns about invalid catalogs pinned to the CV being promoted

A catalog's validation status is: `draft` (never validated or has unvalidated changes), `valid` (last validation passed), or `invalid` (last validation found errors).

## 4. Persistence and Storage

### 4.1 Database as Source of Truth

The database is the primary store for all data:

- Meta configuration (entity type definitions, association rules, attribute schemas) — all versions
- Catalogs and their entity instances, attribute values, and association links
- Catalog version definitions and their entity version pins
- Version history

**Production**: PostgreSQL
**Development**: SQLite or MySQL (lower footprint)

Version management is automatic and handled by the system — no manual version management.

### 4.2 CRDs as Deployment Artifacts

Kubernetes Custom Resources are **not** the source of truth. They are generated artifacts. Two types of CRs are distinguished:

**CatalogVersion CRs (discovery artifacts):**
- Lightweight CRs created when a catalog version is promoted to **testing** or **production**.
- Contain version label, description, lifecycle stage, and entity type names — just enough for applications to discover available catalog versions via the K8s API.
- The API server creates CatalogVersion CRs on promotion; the operator reconciles them (sets owner references to the AssetHub CR for garbage collection, updates status conditions).
- During development, no CRs are created — all work happens via the UI/API against the database.

**Catalog CRs (discovery artifacts):**
- Lightweight CRs created only for catalogs with validation status `valid` (or possibly `invalid` — TBD). Draft catalogs are not published to K8s.
- Contain catalog name, pinned CV reference, API endpoint, catalog ID, and validation status — enough for applications to discover available catalogs and query their data via the Operational API.
- Scoping (namespaced vs cluster-scoped) to be determined during implementation. Note: namespaced Catalog CRs would simplify access rights management and data isolation on OpenShift — standard RBAC role bindings per namespace can control which teams/services can access which catalogs, without custom authorization logic in the API server.

**Entity type CRDs (future scope):**
- Full schema-as-CRD artifacts generated from entity type definitions. These would allow entity instances to be represented as native K8s resources.
- This feature is out of scope for the current implementation and will be addressed in a future phase.

This separation supports:
- Iterative development workflows (save-in-progress via UI without cluster side effects)
- Concurrent editing by multiple authors
- Transactional consistency and conflict handling via the database

## 5. API Design

### 5.1 Two API Sets

#### Meta API
Manages the schema layer — entity type definitions, association definitions, attribute schemas, and catalog versions. Used by administrators to configure what the system manages.

#### Operational API
Manages catalogs and entity instances — CRUD operations and queries. Used by operators and consumers to work with actual AI assets.

### 5.2 API Scoping

The operational API is scoped to a **catalog**:
- Catalog CRUD: `POST /api/data/v1/catalogs`, `GET /api/data/v1/catalogs`, `GET /api/data/v1/catalogs/{catalog-name}`
- Entity instances within a catalog: `GET /api/data/v1/catalogs/{catalog-name}/mcp-servers`
- Catalog validation: `POST /api/data/v1/catalogs/{catalog-name}/validate`
- Catalog names must be DNS-label compatible (`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`, max 63 chars) so they are safe for use in URL paths.

This ensures that a consumer always interacts with a named, consistent data set backed by a specific schema (CV).

### 5.3 Query Capabilities

#### Filtering and Sorting
All entity listing endpoints support attribute-based filtering and sorting (mandatory).

#### Containment Traversal
Contained entities are accessible via REST sub-resource URLs that reflect the hierarchy:
```
GET /mcp-servers/{id}/tools          — list tools contained in an MCP server
GET /mcp-servers/{id}/tools/{name}   — get a specific tool by name
```

#### Forward Reference Traversal
Referenced entities are accessible via a generic reference endpoint:
```
GET /mcp-servers/{id}/references             — all references from this entity
GET /mcp-servers/{id}/references/{type}      — references of a specific type
```

#### Reverse Reference Queries
"Referred by" lookups do not require a direct sub-resource API. They can be served through a query endpoint or filter mechanism, and are permitted to have higher latency than forward traversals.

### 5.4 Meta Operations

#### Copy Entity Definition
Users can create a new entity type based on an existing one. The source can be any version of any existing entity type. The copy becomes a new entity type at V1 with its own independent lifecycle. For example, copying the Guardrail entity definition (V2) to create a new Evaluator entity definition (V1).

#### Copy Attributes
Users can add an attribute to an entity type by copying it from another entity type, without redefining it from scratch. This avoids redundant manual entry when multiple entity types share similar attributes.

#### Enum Management
Closed-list (enum) value sets are managed as first-class objects — defined once and reusable across multiple entity types as attribute types. This provides a centralized way to create, view, update, and reuse enums rather than defining them inline per attribute.

## 6. User Interface

The hub includes a UI for both meta operations (schema management) and operational use (entity management).

General requirements:
- The UI communicates with the backend **exclusively** through the APIs. No direct database or cluster access.
- The UI must support the iterative development workflow — authors can build and modify meta configuration over multiple sessions, with in-progress persistence in the database.
- The UI respects the user's role. Controls for actions the user is not authorized to perform are hidden or disabled, not shown and then rejected.

### 6.1 Meta Operations UI

The meta operations UI is the primary workspace for Admins to define and manage the schema layer. It is a configuration-building tool, not a simple form — Admins will spend extended time here constructing entity types, attributes, associations, and catalog versions.

#### 6.1.1 Entity Type Management

**Entity type list view**
- Displays all defined entity types with their current (latest) version, description, and the number of attributes and associations.
- Supports filtering by name and sorting by name or version.
- Each entity type links to its detail/edit view.
- Provides actions: create new, copy from existing.

**Entity type detail/edit view**
- Shows the full definition of an entity type: name, description, version, all custom attributes, and all associations.
- Admin can edit name and description. Changes auto-save to the database.
- Shows a read-only list of common attributes (ID, Name, Description, Version) for reference.
- Provides a version history panel showing all previous versions of this entity type, with the ability to view (read-only) any past version.
- When viewing a past version, the Admin can use it as the source for a copy operation.

**Create entity type**
- A form to define a new entity type: name (required), description (optional).
- After creation, the Admin is taken to the detail/edit view to add attributes and associations.

**Copy entity type**
- Admin selects a source entity type and version.
- Admin provides a new name for the copy.
- The system creates the new entity type at V1 with all attributes copied.
- Associations are not copied. The UI communicates this clearly (e.g., a notice or confirmation dialog explaining that associations must be defined separately).

#### 6.1.2 Attribute Management

**Attribute list within entity type**
- The entity type detail view shows all custom attributes in an ordered list.
- Each attribute displays: name, description, type (string/number/enum name), and whether it is required.
- Attributes can be reordered (drag-and-drop or move up/down controls).

**Add attribute**
- Inline form or dialog to add a new attribute: name (required), description (optional), type (required — dropdown of string, number, or existing enums).
- For enum types, the dropdown lists all centrally-defined enums by name. The Admin can also create a new enum inline (see 6.1.4).
- Validation: attribute name must be unique within the entity type.

**Edit attribute**
- Admin can modify an attribute's name, description, or type.
- If the entity type is referenced by a catalog version in production, editing is blocked for non-Super Admin users. The UI disables the controls and shows a message explaining why.

**Remove attribute**
- Admin can remove an attribute from the entity type.
- Confirmation dialog warns that removing an attribute will affect future instances and increments the entity type version.

**Copy attribute from another entity type**
- A picker/dialog that allows the Admin to browse other entity types and select one or more attributes to copy.
- The picker shows the source entity type, version, and attribute details.
- If an attribute with the same name already exists on the target, it is shown as disabled/grayed out with a conflict indicator.
- Copied attributes are independent — the UI does not imply any ongoing link to the source.

#### 6.1.3 Association Management

**Association list within entity type**
- The entity type detail view shows all associations involving this entity type.
- Each association displays: the related entity type, association type (containment/directional reference/bidirectional reference), and direction (e.g., "contains Tool" or "refers to Model").
- Containment associations are visually distinguished from references (e.g., different icons or grouping).

**Add association**
- Dialog or form to define a new association:
  - Select the target entity type (dropdown of all defined entity types).
  - Select the association type (containment, directional reference, bidirectional reference).
  - For containment: select which side is the parent (this entity contains the target, or the target contains this entity).
  - For directional reference: select the direction (this entity refers to target, or target refers to this entity).
- Validation:
  - Containment associations are validated for cycles in real-time. If adding the association would create a cycle, the UI shows an error before submission.
  - An entity type cannot have a containment association with itself.

**Remove association**
- Admin can remove an association. Confirmation dialog explains the implications (e.g., "contained entities of this type will no longer be scoped under this parent").

**Visual association map (optional but recommended)**
- A diagram or graph view showing all entity types and their associations.
- Entity types are nodes; associations are edges with labels indicating type and direction.
- Useful for understanding the overall schema at a glance, especially as the number of entity types grows.

#### 6.1.4 Enum Management

**Enum list view**
- Displays all defined enums with their name, number of values, and a list of entity types/attributes that reference them.
- Supports filtering by name.
- Provides actions: create new, edit, delete.

**Create/edit enum**
- Form to define or modify an enum: name (required), ordered list of allowed values.
- Values can be added, removed, and reordered.
- When editing, the UI shows which attributes currently reference this enum.
- If removing a value that may be in use by existing entity instances, the UI warns the Admin.

**Delete enum**
- Blocked if any attribute in any entity type version references the enum.
- The UI shows which attributes reference it and prevents deletion until all references are removed.

**Inline enum creation**
- When adding an attribute and selecting enum as the type, the Admin can create a new enum inline without navigating away from the entity type view.
- The newly created enum is immediately available for selection.

#### 6.1.5 Catalog Version Management

**Catalog version list view**
- Displays all catalog versions with: version identifier, lifecycle stage (Development/Testing/Production), creation date, and the number of pinned entity definitions.
- Lifecycle stage is visually indicated (e.g., color-coded badges: blue for Development, yellow for Testing, green for Production).
- Supports filtering by lifecycle stage.

**Create catalog version**
- Admin selects which entity definition versions to include.
- The UI presents a selection interface showing all entity types and their available versions, with the latest version pre-selected as a default.
- A summary/review step shows the complete bill of materials before confirmation.
- The catalog version is created in the Development stage.

**Catalog version detail view**
- Shows the complete bill of materials: each entity type name and its pinned version.
- For each pinned entity definition, a link to view that specific version's detail (read-only).
- Shows the current lifecycle stage and the history of stage transitions (who promoted, when).
- Shows available actions based on the current stage and user role:
  - Development: "Promote to Testing" (RW and above)
  - Testing: "Demote to Development" (RW and above), "Promote to Production" (Admin and above)
  - Production: "Demote" (Super Admin only)

**Lifecycle promotion**
- Promotion triggers a confirmation dialog explaining what will happen (e.g., "CRs will be generated and applied to the cluster").
- If promotion fails (e.g., CR generation error), the UI displays the error and the catalog version remains in its current stage.
- Demotion from Production requires Super Admin and shows a warning about impact on consumers currently using this version.

#### 6.1.6 Version History and Comparison

**Entity type version history**
- Accessible from the entity type detail view.
- Lists all versions with: version number, date created, and a summary of what changed (attributes added/removed/modified, associations changed).
- Admin can view any past version in read-only mode.

**Version comparison**
- Admin can select two versions of the same entity type and see a side-by-side diff of attributes and associations.
- Added attributes/associations are highlighted in green, removed in red, modified in yellow.
- Useful for understanding what changed between versions before assembling a catalog version.

#### 6.1.7 Validation and Feedback

- All forms validate input inline (e.g., name uniqueness, required fields, cycle detection) before submission.
- Save operations provide clear success/failure feedback.
- Destructive operations (delete, remove attribute, demote from production) always require explicit confirmation.
- The UI clearly indicates when an entity type or catalog version is in a protected state (production) and which operations are restricted.
- Error messages from the API are displayed in human-readable form, not raw API responses.

## 7. Access Control and Roles

The system leverages OpenShift's native RBAC — no duplication of OCP access control functionality.

### Roles

| Role         | Permissions |
|--------------|-------------|
| **RO**       | Read-only access to entities and configuration. No lifecycle changes. |
| **RW**       | Create, update, and delete entity instances. Create catalog versions in development. Promote dev→test. Demote test→dev. |
| **Admin**    | Modify meta configuration while the catalog version is not in production. All RW lifecycle permissions plus promote test→production. |
| **Super Admin** | Modify meta configuration even when the catalog version is in production. All Admin lifecycle permissions plus demote from production (to test or dev). |

The role model is designed to be extensible for future additional roles.

## 8. Deployment

### 8.1 OpenShift Operator

The system is deployed via an operator built with **operator-sdk**. The operator:

- Installs and manages all system components on OpenShift
- Watches `CatalogVersion` CRs created during catalog version promotion (test/prod)
- Sets owner references on CatalogVersion CRs pointing to the AssetHub CR (enables automatic garbage collection)
- Updates CatalogVersion CR status conditions
- Manages a `clusterRole` configuration that controls which lifecycle stages the API server exposes
- Passes `CLUSTER_ROLE` to the API server ConfigMap
- Leverages existing OpenShift and Kubernetes capabilities (RBAC, networking, storage) — no duplication of OCP functionality

**Note:** `clusterRole` is separate from infrastructure `environment` — these are orthogonal concerns. `environment` (development/openshift) controls infrastructure behavior (NodePort vs ClusterIP, image pull policy, Routes, RBAC mode). `clusterRole` (development/testing/production) controls data visibility (which catalog version lifecycle stages the API serves).

### 8.3 Cluster Role Configuration

The operator manages a `clusterRole` field on the AssetHub CR that controls which catalog version lifecycle stages the API server exposes:

| clusterRole | Visible lifecycle stages |
|-------------|--------------------------|
| `development` (default) | development, testing, production |
| `testing` | testing, production |
| `production` | production only |

This enables:
- **Production clusters** to only serve production catalog versions, preventing accidental exposure of in-progress configurations.
- **Test clusters** to serve testing and production catalog versions for validation workflows.
- **Development clusters** to serve all stages for unrestricted development access.

The operator passes the `CLUSTER_ROLE` value to the API server ConfigMap. The API server filters list/get responses for catalog versions accordingly.

### 8.4 Future Enhancement: Centralized Hub with Remote Consuming Clusters

The current architecture assumes a single-cluster deployment where the API server, database, and operator all run together. A future enhancement will support a **hub-and-spoke topology** where one central AssetHub instance serves an entire organization across multiple clusters.

#### Architecture

```
┌─────────────────────────────────┐
│       Central Hub Cluster       │
│                                 │
│  ┌──────────┐  ┌─────────────┐  │
│  │ Database │◄─│ API Server  │◄──── authoring (UI, Meta API)
│  │ (source  │  │ (full stack)│  │
│  │ of truth)│  │             │  │
│  └──────────┘  └──────┬──────┘  │
│                       │         │
│                       │ Operational API
│                       ▼         │
│               ┌──────────────┐  │
│               │   Operator   │  │
│               └──────────────┘  │
└─────────────────────────────────┘
        │
        │  remote API access
        ▼
┌──────────────────────────┐   ┌──────────────────────────┐
│   Consuming Cluster A    │   │   Consuming Cluster B    │
│                          │   │                          │
│  ┌────────────────────┐  │   │  ┌────────────────────┐  │
│  │ Operator (local)   │  │   │  │ Operator (local)   │  │
│  │ syncs CVs from     │  │   │  │ syncs CVs from     │  │
│  │ central API        │  │   │  │ central API        │  │
│  └────────────────────┘  │   │  └────────────────────┘  │
│                          │   │                          │
│  CatalogVersion CRs      │   │  CatalogVersion CRs      │
│  (auto-created)          │   │  (auto-created)          │
└──────────────────────────┘   └──────────────────────────┘
```

#### Concept

- **Central hub cluster**: Runs the full AssetHub stack (DB, API server, UI, operator). All schema authoring, catalog version creation, and lifecycle management happens here. This is the single source of truth for an organization's AI asset metadata.

- **Consuming clusters**: Run only the operator. The operator connects to the **remote** central API server (not a local DB) and periodically syncs `CatalogVersion` CRs into the local cluster. Applications on the consuming cluster discover available catalog versions via the local K8s API and use the central Operational API for data access.

- **No local promotion**: Consuming clusters do not perform promotion. The central hub promotes catalog versions through the lifecycle. The consuming cluster's operator polls or watches the central API and automatically creates/updates/deletes local `CatalogVersion` CRs to reflect the central state. The `clusterRole` of the consuming cluster determines which lifecycle stages are synced.

#### Key Design Points

- **External DB connection**: The AssetHub CR on the consuming cluster points to the central API server URL instead of a local database. The operator uses this to sync catalog version state.

- **Automatic CR creation**: On consuming clusters, `CatalogVersion` CRs are created without manual promotion. The operator's sync loop detects new or changed catalog versions on the central hub and creates/updates local CRs accordingly. Deletions on the central hub cascade to local CR deletions.

- **Cluster role filtering at sync time**: A consuming cluster with `clusterRole=production` only syncs production-stage catalog versions. A `clusterRole=testing` cluster syncs testing and production. This ensures consuming clusters only see the stages appropriate to their role.

- **Operational API routing**: Applications on consuming clusters use the central API server's Operational API (`/api/data/v1/:catalog-version/...`) for entity instance access. The local `CatalogVersion` CRs provide discovery only — the data lives in the central database.

- **Network requirements**: Consuming clusters need network access to the central API server. This may require cross-cluster networking, ingress routes, or service mesh configuration depending on the infrastructure.

#### Open Questions

| Item | Notes |
|------|-------|
| Sync mechanism | Polling interval vs. watch/webhook from central hub |
| Authentication | How consuming cluster operators authenticate to the central API |
| Offline resilience | Behavior when the central hub is temporarily unreachable |
| Multi-tenancy | Whether a single hub can serve multiple organizations with isolation |
| Data locality | Whether consuming clusters cache entity instance data locally |

### 8.2 Constraints

- The system must run on OpenShift.
- It must use existing Kubernetes/OpenShift capabilities wherever possible.
- No duplication of functionality already provided by OCP (e.g., RBAC, secrets management, networking).

## 9. User Stories

### Meta Repository Management

**US-1: Define a new entity type**
As an Admin, I want to define a new entity type (e.g., "Guardrail") with a name, description, and a set of custom attributes, so that the system can manage instances of that type.

**Why**: The system's core value proposition is flexibility — the ability to manage any kind of AI asset without code changes. Without dynamic entity type definition, every new asset type would require a code release.

Acceptance Criteria:
- Admin can create a new entity type via the Meta API or UI by specifying a name, description, and zero or more custom attributes.
- The entity type name must be unique within the catalog.
- Each custom attribute must have a name, description, and type (string, number, or enum).
- The new entity type is created at version 1.
- After creation, entity instances of the new type can be created via the Operational API.
- RO and RW users cannot create entity types; the API returns a 403.

---

**US-2: Define associations between entity types**
As an Admin, I want to define associations (containment, directional reference, or bidirectional reference) between entity types, so that relationships between assets are formally modeled and navigable.

**Why**: AI assets have inherent relationships (tools belong to MCP servers, models reference guardrails). Without formal association modeling, consumers would have to maintain these relationships manually and inconsistently.

Acceptance Criteria:
- Admin can define a containment, directional reference, or bidirectional reference between two entity types.
- Containment associations are validated to not create cycles (the system rejects associations that would form a circular containment path).
- For containment: the contained entity type becomes namespaced under the containing type.
- For directional references: the system records the direction (source → target).
- The entity type definition version is incremented when an association is added.
- Associations are reflected in the API URL structure (containment as sub-resources, references via `/references` endpoints).
- Each association has source and target cardinality (UML-style multiplicity). Standard options: `0..1`, `0..n`, `1`, `1..n`. Custom ranges are also supported (e.g., `2..5`). Default cardinality is `0..n` on both ends when not specified.
- Cardinality is stored as strings (`source_cardinality`, `target_cardinality`) on the Association model. Validation ensures min <= max, both non-negative, and max can be `n` (unbounded).
- Each association has a required name that is unique within the entity type version. Association names share the same namespace as attribute names — no association can have the same name as an attribute on the same entity type version.

---

**US-3: Copy an entity definition to create a new type**
As an Admin, I want to create a new entity type by copying an existing entity definition (at any version), so that I can reuse an existing structure as a starting point instead of defining it from scratch.

**Why**: Many AI asset types share similar structures (e.g., guardrails and evaluators). Copying avoids tedious re-entry and reduces configuration errors when creating structurally similar entity types.

Acceptance Criteria:
- Admin can select a source entity type and version to copy from.
- Admin provides a new name for the target entity type.
- The new entity type is created at V1 with all attributes copied from the source.
- Associations are **not** copied (they reference specific entity types and may not apply).
- The new entity type has an independent lifecycle — changes to the source do not affect the copy.
- The source entity type and version remain unchanged.

---

**US-4: Copy attributes between entity types**
As an Admin, I want to add an attribute to an entity type by copying it from another entity type, so that I don't have to re-enter attribute definitions that are shared across types.

**Why**: Common attributes (e.g., "runtime_environment", "max_tokens") appear across multiple entity types. Re-entering them manually is error-prone and leads to inconsistencies in naming and type definitions.

Acceptance Criteria:
- Admin can select one or more attributes from a source entity type and copy them to a target entity type.
- Copied attributes are independent — subsequent changes to the source attribute do not propagate to the copy.
- If an attribute with the same name already exists on the target, the operation is rejected with a conflict error.
- The target entity type version is incremented.

---

**US-5: Manage enums centrally**
As an Admin, I want to define enum value sets as reusable objects and assign them as attribute types across multiple entity types, so that closed lists are consistent and maintained in one place.

**Why**: Without centralized enum management, the same set of values (e.g., supported languages, deployment targets) would be duplicated across entity types. Updates would require finding and changing every copy, risking inconsistency.

Acceptance Criteria:
- Admin can create a named enum with an ordered list of allowed values.
- Admin can update an enum's values (add, remove, reorder).
- Enums can be assigned as the type of any custom attribute on any entity type.
- Multiple attributes across different entity types can reference the same enum.
- When an enum is updated, all attributes referencing it reflect the updated values.
- An enum cannot be deleted if it is referenced by any attribute in any entity type version.

---

**US-6: Modify an entity definition**
As an Admin, I want to modify an entity type definition (add/change/remove attributes or associations), and have the system automatically create a new version of that definition, so that previous versions remain intact.

**Why**: Automatic versioning on mutation is the foundation of backward compatibility. Without it, changes to an entity type could silently break deployments that depend on the previous structure.

Acceptance Criteria:
- When an Admin modifies any aspect of an entity type (attributes, associations), the system creates a new version automatically.
- The previous version remains accessible and unchanged.
- The new version number is the previous version incremented by 1.
- Existing catalog versions that reference the old entity type version continue to reference it — they are not affected by the new version.
- The modification is rejected if the Admin role lacks permission (e.g., catalog version is in production and user is not Super Admin).

---

**US-7: Iterative meta configuration in the UI**
As an Admin, I want to build and modify meta configuration over multiple sessions in the UI, with my work-in-progress saved to the database, so that I can work iteratively without needing to complete everything in one sitting.

**Why**: Building a complete meta configuration (multiple entity types, associations, attributes) is a non-trivial task. Requiring it to be completed in a single session would be impractical and error-prone. No CRs should be generated until the configuration is explicitly promoted.

Acceptance Criteria:
- All meta configuration changes are persisted to the database immediately on save.
- No CRDs or CRs are generated during the Development lifecycle stage.
- The Admin can close the UI, reopen it, and resume editing where they left off.
- Multiple Admins can work on different entity types concurrently.
- The UI shows the current state of all entity type definitions, their versions, and associations.

---

### Catalog Versioning and Lifecycle

**US-8: Create a catalog version**
As an RW user, I want to create a catalog version that pins specific entity definition versions together, so that I have an immutable snapshot of the schema for deployment.

**Why**: Without a pinning mechanism, deployments would be subject to schema drift as entity definitions evolve. Catalog versions provide the stability guarantee that deployed systems need.

Acceptance Criteria:
- RW (and above) users can create a new catalog version by selecting specific entity definition versions to include.
- The catalog version records the exact `(entity type name, version)` tuples it contains.
- Once created, the catalog version's entity version pins cannot be changed (immutable snapshot).
- The catalog version is created in the Development lifecycle stage.
- The catalog version has a unique identifier.
- RO users cannot create catalog versions; the API returns a 403.

---

**US-9: Promote a catalog version to testing**
As an RW user, I want to promote a catalog version from Development to Testing, so that a CatalogVersion CR is created in K8s for discovery and validation.

**Why**: Testing in a real cluster environment is the only way to validate that the meta configuration works correctly with the operator and runtime. Creating a CatalogVersion CR at this stage makes the catalog version discoverable via the K8s API and catches deployment issues before production.

Acceptance Criteria:
- RW (and above) users can promote a catalog version from Development to Testing.
- On promotion, a `CatalogVersion` CR is created in K8s containing: version label, description, lifecycle stage (`testing`), and entity type names from the pinned definitions.
- Applications can discover the promoted catalog version via the K8s API and use it with the operational API (`/api/data/v1/:catalog-version/...`).
- The catalog version status is updated to Testing in the database.
- If CR creation fails, the promotion is rolled back with an error message.
- RW (and above) users can demote a catalog version back from Testing to Development if testing reveals issues (which deletes the CR).
- RO users cannot promote or demote; the API returns a 403.

---

**US-10: Promote a catalog version to production**
As an Admin, I want to promote a catalog version from Testing to Production, so that the schema is deployed and frozen for operational use.

**Why**: Production promotion is the point at which the schema becomes the active contract for API consumers. Freezing it prevents accidental changes that could break running systems. This requires Admin authority because it affects all consumers of the operational API.

Acceptance Criteria:
- Admin (and above) users can promote a catalog version from Testing to Production.
- On promotion, the `CatalogVersion` CR is updated to lifecycle stage `production`.
- Production environments (clusters with `clusterRole=production`) filter API responses to only serve production catalog versions.
- The catalog version is frozen — no modifications are allowed by Admin or lower role users.
- The Operational API begins serving data scoped to this catalog version.
- Only Super Admin users can modify or demote the catalog version after this point.
- RW and RO users cannot promote to production; the API returns a 403.

---

**US-11: Modify a production catalog version**
As a Super Admin, I want to modify meta configuration even when the catalog version is in production, so that critical changes can be made when necessary.

**Why**: In rare but critical situations (security patches, regulatory requirements, critical bugs), the ability to modify a production schema is a safety valve. Restricting this to Super Admin ensures it is an intentional, authorized action.

Acceptance Criteria:
- Super Admin can modify entity definitions within a production catalog version.
- The system logs all modifications to production catalog versions for audit purposes.
- Admin, RW, and RO users receive a 403 when attempting to modify a production catalog version.
- Modified CRs are regenerated and reapplied to the cluster.

---

**US-12: Remove a catalog version from production**
As a Super Admin, I want to take a catalog version out of production, so that it can be replaced or retired.

**Why**: Catalog versions may need to be retired when replaced by a newer version, or rolled back if a critical issue is discovered post-deployment. Only Super Admin can perform this action because it directly impacts active consumers.

Acceptance Criteria:
- Super Admin can demote a catalog version from Production to Testing or Development.
- On demotion to Testing, the `CatalogVersion` CR is updated with lifecycle stage `testing`.
- On demotion to Development, the `CatalogVersion` CR is deleted from K8s (development-stage versions do not have CRs).
- The Operational API stops serving data scoped to the demoted catalog version (or returns an appropriate error to consumers still referencing it).
- The catalog version data is retained in the database for audit and potential re-promotion.
- Admin, RW, and RO users cannot demote from production; the API returns a 403.

---

### Catalog Management

**US-33: Create a catalog**
As an RW user, I want to create a named catalog pinned to a catalog version, so that I can start populating it with entity instances.

Acceptance Criteria:
- RW user can create a catalog by specifying a name, description (optional), and a catalog version ID.
- The catalog name must be unique and DNS-label compatible (`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`, max 63 chars).
- The catalog is created with validation status `draft`.
- The catalog's pinned CV determines which entity types and attributes are available for creating instances.
- RO users cannot create catalogs; the API returns a 403.
- `GET /api/data/v1/catalogs` lists all catalogs with support for filtering by `catalog_version_id` and `validation_status`.
- `GET /api/data/v1/catalogs/{catalog-name}` returns catalog detail including resolved CV label (not just CV ID).
- `DELETE /api/data/v1/catalogs/{catalog-name}` deletes the catalog and cascades deletion to all entity instances within it. RW and above can delete; RO returns 403.

---

**US-34: Validate a catalog**
As an RW user, I want to validate all entity instances in a catalog against the pinned CV's schema constraints, so that I can identify and fix errors before the catalog is published.

Acceptance Criteria:
- RW (and above) users can trigger validation. RO users cannot.
- Validation checks: required attributes have values, attribute values match their type, mandatory associations are satisfied (cardinality `1` or `1..n`), containment hierarchy is consistent.
- Validation returns a list of errors (entity name, field, violation) — not just pass/fail.
- Validation status is updated to `valid` (no errors) or `invalid` (errors found).
- Any subsequent data change resets the status to `draft`.
- Only Admin can promote (publish) a catalog — promotion requires validation status `valid`.

---

### Entity Instance Management

**US-13: Create an entity instance**
As an RW user, I want to create a new instance of a configured entity type within a catalog, with values for its attributes, so that I can register a new AI asset in the hub.

**Why**: This is the primary write operation of the system — registering actual AI assets. Without it, the hub would have schema but no data.

Acceptance Criteria:
- RW user can create an entity instance by specifying the catalog, entity type, a name, description, and values for custom attributes.
- The entity type must be one of the types pinned in the catalog's CV.
- The name must be unique within scope (global for top-level entities within the catalog, within the parent for contained entities).
- Attribute values are validated against the attribute type (string, number, enum value from allowed list), but missing optional attributes are allowed (draft mode).
- The instance is created at version 1 with a system-calculated ID.
- RO users cannot create instances; the API returns a 403.
- The response includes the instance with resolved attribute values (attribute name, type, and value — not just raw IDs).
- The API URL uses the entity type name: `POST /api/data/v1/catalogs/{catalog-name}/{entity-type-name}`.
- Any data mutation to instances in a catalog resets the catalog's validation status to `draft`.

---

**US-14: Update an entity instance**
As an RW user, I want to update an entity instance's attribute values and have the system automatically increment its version, so that changes are tracked.

**Why**: Auto-versioning on update ensures that every change to an AI asset is traceable. This is critical for environments where knowing the exact state of a model or guardrail at any point in time matters for compliance and debugging.

Acceptance Criteria:
- RW user can update one or more attribute values on an entity instance.
- The system increments the instance version automatically on each successful update.
- The previous version state is retained in the database.
- Attribute value changes are validated against the attribute type definition.
- The updated instance returns the new version number and all current attribute values in the response.
- Update requires the current version number for optimistic locking (409 on mismatch).
- The update request can include name, description, and/or attribute values — all optional, only provided fields are changed.

---

**US-15: Delete an entity instance**
As an RW user, I want to delete an entity instance, and if it is a containing entity, have all contained entities cascade-deleted, so that orphaned data is not left behind.

**Why**: Cascade deletion enforces the containment contract — contained entities cannot exist without their parent. Without it, deleting a parent would leave orphaned children with broken namespace references.

Acceptance Criteria:
- RW user can delete an entity instance by ID.
- If the entity contains other entities (via containment association), all contained entities are deleted recursively.
- Multi-level containment cascades correctly (deleting A deletes B, which deletes C).
- References to the deleted entity from other entities are handled (at minimum, the system does not leave dangling references without notification).
- The delete operation is atomic — either the entire cascade succeeds or nothing is deleted.

---

**US-16: Create a contained entity**
As an RW user, I want to create an entity instance within a containing entity (e.g., a Tool within an MCP Server), so that the containment hierarchy is maintained.

**Why**: Contained entities are namespaced and lifecycle-bound to their parent. Creating them through the parent ensures the containment relationship is established correctly and name uniqueness is enforced within the right scope.

Acceptance Criteria:
- RW user can create a contained entity by specifying the parent entity and the contained entity type.
- The contained entity's name must be unique within the parent's namespace (not globally).
- The contained entity is accessible via the parent's sub-resource URL (e.g., `GET /mcp-servers/{id}/tools/{name}`).
- The contained entity cannot exist without a parent — creation without specifying a valid parent is rejected.
- If the parent entity does not exist, the creation is rejected with a 404.

---

### Queries and Navigation

**US-17: List and filter entities**
As an RO user, I want to list entity instances with attribute-based filtering and sorting, so that I can find specific assets.

**Why**: As the number of managed assets grows, unfiltered listings become unusable. Filtering and sorting are essential for operational workflows — finding models by status, listing guardrails by type, etc.

Acceptance Criteria:
- RO user can list entity instances of a given type.
- Results can be filtered by any attribute value (common or custom). Filter semantics are type-aware:
  - String attributes: case-insensitive substring match (contains).
  - Number attributes: exact match, or range filter with min/max.
  - Enum attributes: exact match against the enum value.
- Results can be sorted by any attribute (ascending or descending).
- Multiple filters can be combined (AND logic).
- The response includes pagination support (offset/limit with total count). Default page size is 20; configurable up to 100.
- Filtering by non-existent attributes returns an error, not an empty result.

---

**US-18: Navigate containment hierarchy**
As an RO user, I want to access contained entities via the parent entity's URL (e.g., `GET /mcp-servers/{id}/tools`), so that I can browse assets within their natural hierarchy.

**Why**: Containment is a core relationship model. Exposing it through the URL structure makes the API intuitive and self-documenting — the URL itself communicates the relationship between assets.

Acceptance Criteria:
- Contained entities are accessible via `GET /{parent-type}/{parent-id}/{contained-type}`.
- Individual contained entities are accessible via `GET /{parent-type}/{parent-id}/{contained-type}/{name}`.
- Filtering and sorting (from US-17) apply to contained entity listings.
- Accessing a contained entity via a non-existent parent returns 404.
- A containment tree endpoint (`GET /catalogs/{name}/tree`) returns the full instance hierarchy for a catalog as a nested structure, enabling tree-based browsing UIs.
- Each instance in the tree includes its entity type name and a summary of its children.
- Instance detail includes a parent chain (ordered list of ancestors up to root) for breadcrumb navigation.
- Multi-level containment URLs (e.g., `GET /a/{id}/b/{id}/c`) are deferred — single-level parent-child routes and the tree endpoint are sufficient for navigating deep hierarchies.

---

**US-19: Follow forward references**
As an RO user, I want to retrieve all entities referenced by a given entity (e.g., `GET /mcp-servers/{id}/references`), so that I can understand what an asset depends on.

**Why**: Understanding dependencies is critical for impact analysis, deployment planning, and debugging. Forward references answer "what does this asset depend on?"

Acceptance Criteria:
- Forward references are accessible via `GET /{entity-type}/{id}/references`.
- Results can be filtered by reference type via `GET /{entity-type}/{id}/references/{type}`.
- The response includes the referenced entity's type, ID, and name at minimum.
- The lookup is fast — performance comparable to direct entity retrieval.
- Both directional and bidirectional references are included.

---

**US-20: Look up reverse references**
As an RO user, I want to find which entities refer to a given entity, so that I can understand the impact of changing or removing it — accepting that this query may be slower than forward lookups.

**Why**: Reverse reference lookups answer "what would break if I change or remove this asset?" This is essential for safe change management, even if it doesn't need to be as fast as forward lookups.

Acceptance Criteria:
- An API mechanism exists to query reverse references for a given entity.
- The response includes the referring entity's type, ID, and name.
- The query does not need to be a direct sub-resource URL — a query endpoint or filter is acceptable.
- Higher latency than forward reference lookups is acceptable.
- Both directional (referredBy) and bidirectional references are included.

---

**US-21: Interact with a specific catalog version**
As an API consumer, I want all operational API calls scoped to a specific catalog version, so that I get a consistent view of the asset catalog regardless of ongoing changes.

**Why**: Without catalog version scoping, API consumers would be exposed to schema changes mid-flight. This is the core guarantee that makes the versioning model useful — consumers pin to a version and get deterministic behavior.

Acceptance Criteria:
- Every operational API call requires a catalog version identifier (either via URL path or mandatory parameter).
- The API serves entity types and instances consistent with the pinned catalog version.
- Requests with an invalid or non-existent catalog version return a clear error.
- Changes to entity definitions in other catalog versions do not affect responses for the pinned version.
- If a catalog version has been removed from production, the API returns an appropriate error (not stale data).

---

### Access Control

**US-22: Read-only access**
As an RO user, I want to browse entities and configuration without being able to modify anything, so that I can consume asset information safely.

**Why**: Many consumers of the asset hub (dashboards, monitoring systems, downstream services) only need read access. Enforcing read-only access prevents accidental modifications from non-privileged users.

Acceptance Criteria:
- RO users can perform all GET operations on both Meta and Operational APIs.
- RO users receive a 403 on any POST, PUT, PATCH, or DELETE operation.
- RO access is enforced at the API layer via OpenShift RBAC, not application-level checks.

---

**US-23: Role-based access enforcement**
As a platform administrator, I want access control enforced via OpenShift's native RBAC, so that there is no duplication of OCP identity and access management.

**Why**: OpenShift already provides a mature, audited RBAC system. Duplicating it would create security gaps (two systems to maintain), inconsistency (different permissions in OCP vs. the hub), and operational overhead.

Acceptance Criteria:
- All four roles (RO, RW, Admin, Super Admin) map to OpenShift RBAC roles/bindings.
- Authentication is handled by OpenShift (no separate user database or login system).
- Authorization decisions use the OCP RBAC API (SubjectAccessReview or equivalent).
- Adding or removing users from roles is done via standard OpenShift tooling (oc, console).
- No custom user management UI or API exists in the hub.
- Per-catalog access control is enforced using K8s RBAC `resourceNames` on the `catalogs` resource (see US-39).

---

### Deployment

**US-24: Install via operator**
As a cluster administrator, I want to deploy the AI Asset Hub by installing an operator via operator-sdk, so that the system is managed as a standard OpenShift workload.

**Why**: Operator-based deployment is the standard pattern for managing applications on OpenShift. It provides lifecycle management (install, upgrade, uninstall), health monitoring, and integration with the OLM (Operator Lifecycle Manager).

Acceptance Criteria:
- The operator is installable via OLM or direct operator-sdk deployment.
- The operator deploys all required components (backend, database, UI).
- The operator manages component lifecycle (restart on failure, rolling upgrades).
- Uninstalling the operator cleanly removes all hub components.
- The operator exposes a CR for hub-level configuration (e.g., database connection, resource limits).

---

**US-25: Operator reconciliation on promotion**
As a cluster administrator, I want the operator to watch for `CatalogVersion` CRs created during catalog version promotion and reconcile their status and ownership, so that the cluster reflects the deployed catalog version.

**Why**: The operator is the bridge between the database (source of truth) and the cluster runtime. Without reconciliation, CatalogVersion CRs would lack owner references (preventing garbage collection) and status conditions (preventing observability).

Acceptance Criteria:
- The operator watches for `CatalogVersion` CRs in its managed namespace.
- When a CatalogVersion CR is detected, the operator sets an owner reference to the AssetHub CR (enabling automatic garbage collection when the AssetHub CR is deleted).
- The operator updates CatalogVersion CR status conditions (ready state, message).
- Reconciliation errors are reported via the CR's status conditions and operator logs.
- The operator does not modify the database — it only manages CatalogVersion CRs on the cluster side.

### Meta Operations UI

**US-26: Entity type list view**
As an Admin, I want to see all defined entity types in a list with their latest version, description, and counts of attributes and associations, so that I can quickly find and navigate to the entity type I need to work on.

**Why**: The list view is the Admin's primary entry point into the meta configuration. Without a clear overview, Admins would have to remember entity type names or search blindly, especially as the number of types grows.

Acceptance Criteria:
- The list displays all entity types with: name, latest version number, description, attribute count, and association count.
- The list supports filtering by name (text search) and sorting by name or version.
- Each row links to the entity type detail/edit view.
- The list provides action buttons to create a new entity type or copy an existing one.
- RO users see the same list but without create/copy actions.

---

**US-27: Entity type detail and editing**
As an Admin, I want a detail view for each entity type where I can see and edit its full definition (attributes, associations, description), so that I have a single workspace for managing an entity type.

**Why**: Entity type configuration involves multiple related elements (attributes, associations, metadata). A unified detail view prevents context-switching between separate pages and reduces the risk of inconsistent edits.

Acceptance Criteria:
- The detail view shows: entity type name, description, current version, all custom attributes (with type and description), and all associations (with type and direction).
- Common attributes (ID, Name, Description, Version) are shown in a read-only section for reference.
- Admin can edit the name and description inline. Changes are saved to the database on explicit save.
- The view provides entry points for adding/editing/removing attributes and associations (see US-28, US-31).
- If the entity type is part of a production catalog version and the user is not Super Admin, all edit controls are disabled with a message explaining the restriction.
- RO and RW users see the detail view in read-only mode with no edit controls.

---

**US-28: Add and edit attributes in the UI**
As an Admin, I want to add, edit, and remove attributes on an entity type through the UI, so that I can define the data structure for each asset type.

**Why**: Attributes define what data each asset type carries. The UI must make attribute management efficient because Admins will frequently add and adjust attributes as requirements evolve during the development lifecycle stage.

Acceptance Criteria:
- Admin can add a new attribute via an inline form or dialog, specifying: name (required), description (optional), and type (string, number, or enum — selected from a dropdown of existing enums).
- Attribute names are validated for uniqueness within the entity type before submission. Duplicate names show an inline error.
- Admin can edit an existing attribute's name, description, or type.
- Admin can remove an attribute. A confirmation dialog warns that this will increment the entity type version.
- Attributes can be reordered (drag-and-drop or move up/down controls).
- When selecting enum as the type, the Admin can create a new enum inline without leaving the page (links to US-33).

---

**US-29: Copy attributes from another entity type in the UI**
As an Admin, I want to browse other entity types and copy selected attributes to the current entity type, so that I can reuse existing attribute definitions without re-entering them.

**Why**: Many entity types share similar attributes. A copy picker is faster and less error-prone than manually recreating attributes, especially when the attribute has an enum type that must be referenced correctly.

Acceptance Criteria:
- A picker/dialog allows the Admin to browse all entity types and their attributes.
- The picker shows each attribute's name, description, and type for informed selection.
- Admin can select one or more attributes to copy.
- Attributes that conflict with existing attributes on the target (same name) are shown as disabled with a conflict indicator.
- On confirmation, selected attributes are added to the current entity type.
- The target entity type version is incremented.

---

**US-30: Copy an entity type in the UI**
As an Admin, I want to create a new entity type by selecting an existing entity type and version as a template, so that I can quickly set up a structurally similar type.

**Why**: Defining entity types from scratch is tedious when a similar type already exists. Copying provides a faster starting point while ensuring the new type is independent.

Acceptance Criteria:
- The Admin can initiate a copy from the entity type list view or from within an entity type's detail view.
- A dialog prompts the Admin to select the source version (defaulting to the latest) and enter a new name.
- The system creates the new entity type at V1 with all attributes copied.
- The dialog clearly states that associations are not copied and must be configured separately.
- After creation, the Admin is navigated to the new entity type's detail view.

---

**US-31: Manage associations in the UI**
As an Admin, I want to add and remove associations between entity types through the UI, with real-time validation for containment cycles, so that I can model relationships without risking invalid configurations.

**Why**: Associations define how entity types relate to each other. Cycle detection must happen before submission — discovering a cycle only after saving would be disruptive, especially in a complex schema with many entity types.

Acceptance Criteria:
- The entity type detail view lists all associations involving this entity type, grouped or labeled by type (containment vs. reference).
- Admin can add a new association via a dialog: select target entity type, select association type (containment, directional reference, bidirectional reference), and select direction where applicable.
- The add association dialog includes cardinality selection for both source and target ends. A dropdown offers standard options (`0..1`, `0..n`, `1`, `1..n`) plus a "Custom" option that reveals min/max input fields. Default is `0..n` on both ends.
- Cardinality is displayed in the association list alongside the relationship label.
- For containment: if the new association would create a cycle, the UI shows an error immediately (before the Admin can submit).
- Admin can edit an existing association's roles and cardinality via a dialog. The association type and target entity type cannot be changed (delete and recreate instead). Editing creates a new entity type version (copy-on-write).
- Admin can remove an association. A confirmation dialog explains the implications.
- Adding, editing, or removing an association increments the entity type version.

---

**US-32: Visual association map**
As an Admin, I want to see a diagram showing all entity types and their associations as a graph, so that I can understand the overall schema structure at a glance.

**Why**: As the number of entity types and associations grows, the relationships become difficult to understand from individual detail views alone. A visual map provides the "big picture" that textual lists cannot.

Acceptance Criteria:
- A graph/diagram view shows entity types as nodes and associations as labeled edges.
- Containment associations are visually distinct from reference associations (e.g., different line styles or colors).
- Entity type nodes display the type name, version number, and a list of attributes with their types (UML class diagram style).
- Edge labels indicate the association name, type, and cardinality.
- The diagram updates automatically when entity types or associations are added or removed.
- The diagram is interactive — double-clicking a node navigates to that entity type's detail view. Double-clicking an edge opens the edit association modal (main page only).
- The diagram handles 10+ entity types without becoming unreadable (supports zoom/pan and automatic layout).
- The diagram appears as a "Model Diagram" tab on the main page (showing all entity types) and as a "Diagram" tab on the catalog version detail page (showing only pinned entity types, read-only).
- Built with `@patternfly/react-topology` (OCP Console native component).

---

**US-33: Enum management in the UI**
As an Admin, I want a dedicated view to create, edit, and delete enums, and I want to see which attributes reference each enum, so that I can manage closed value lists centrally.

**Why**: Enums are shared across entity types. Without a central management view, Admins would not know where an enum is used, risking unintended side effects when modifying or deleting values.

Acceptance Criteria:
- An enum list view displays all enums with: name, number of values, and a list of referencing entity types and attributes.
- Admin can create a new enum with a name and an ordered list of values.
- Admin can edit an enum: add, remove, or reorder values.
- When removing a value, the UI warns if existing entity instances may use that value.
- Admin can delete an enum only if no attributes reference it. If references exist, the UI lists them and blocks deletion.
- Enum creation is also available inline from the attribute type dropdown (without navigating away from the entity type view).

---

**US-34: Catalog version creation in the UI**
As an Admin, I want to create a catalog version by selecting specific entity definition versions from a list, so that I can assemble a bill of materials for deployment.

**Why**: The catalog version is what gets deployed. The selection interface must make it easy to choose the right versions and review the complete snapshot before committing, to avoid deploying unintended entity definition versions.

Acceptance Criteria:
- A creation interface shows all entity types with their available versions.
- The latest version of each entity type is pre-selected as the default.
- Admin can change the selected version for any entity type via a version dropdown.
- The selection interface shows entity types organized as a containment tree. Root entity types (not contained by any other) appear at the top level. Contained entity types appear as indented children of their parent. Entity types with no containment associations appear as standalone roots.
- Selecting a parent entity type auto-selects all its contained descendants recursively (children, grandchildren, etc.). Deselecting a parent deselects all descendants recursively. A contained entity type cannot be selected unless its containing parent is selected — selecting a child auto-selects all ancestors up to the root. Deselecting a child does NOT deselect its parent or ancestors.
- A summary/review step shows the complete bill of materials (all entity type + version pairs) before confirmation.
- On confirmation, the catalog version is created in the Development lifecycle stage.
- The Admin is navigated to the new catalog version's detail view.

---

**US-35: Catalog version detail and lifecycle management in the UI**
As an Admin, I want to see a catalog version's full bill of materials and promote or demote it through lifecycle stages, so that I can manage the deployment pipeline from the UI.

**Why**: Lifecycle promotion is a critical operation with real cluster-side effects (CR generation and application). The UI must provide clear context and confirmation to prevent accidental promotions and help Admins understand the current state.

Acceptance Criteria:
- The detail view shows: catalog version identifier, current lifecycle stage (visually indicated with color-coded badge), creation date, and the full bill of materials (entity type name + pinned version).
- Clicking an entity type name in the bill of materials opens a read-only modal showing the pinned version's attributes and associations. The modal displays the entity type name, pinned version number, and two sections. Attributes show name, type (with resolved enum name for enum types, e.g., "boolean (enum)"), and description. Associations show both outgoing and incoming with contextual relationship labels: "contains"/"contained by" for containment, "references"/"referenced by" for directional, "references (mutual)" for bidirectional — each color-coded. The other entity type name and the perspective-correct role are shown (target role for outgoing, source role for incoming). No edit controls are shown — the modal is purely informational.
- The view shows a history of lifecycle transitions (who promoted/demoted, when).
- Available actions are displayed based on current stage and user role:
  - Development: "Promote to Testing" (RW and above).
  - Testing: "Demote to Development" (RW and above), "Promote to Production" (Admin and above).
  - Production: "Demote" (Super Admin only).
- Promotion triggers a confirmation dialog explaining the side effects (e.g., "CRs will be generated and applied to the cluster").
- If promotion fails, the error is displayed and the catalog version remains in its current stage.
- Demotion from Production shows a warning about impact on active API consumers.

---

**US-36: Entity type version history in the UI**
As an Admin, I want to see the version history of an entity type and compare any two versions side-by-side, so that I can understand what changed and make informed decisions when assembling catalog versions.

**Why**: When selecting which entity definition version to pin in a catalog version, the Admin needs to understand the differences between versions. Without a comparison tool, the Admin would have to manually inspect each version and track changes mentally.

Acceptance Criteria:
- The entity type detail view includes a version history panel listing all versions with: version number, date created, and a summary of changes (attributes added/removed/modified, associations changed).
- Admin can click any version to view its full definition in read-only mode.
- Admin can select two versions for side-by-side comparison.
- The comparison view highlights: added items (green), removed items (red), and modified items (yellow).
- From the history view, the Admin can initiate a copy operation using any past version as the source.

---

**US-37: UI validation and error feedback**
As an Admin, I want the UI to validate my inputs inline before submission and show clear feedback on success or failure, so that I catch errors early and understand the outcome of every action.

**Why**: Meta configuration errors (invalid names, cycle-creating associations, type conflicts) are easier and cheaper to fix when caught at input time rather than after an API round-trip or, worse, during deployment. Clear feedback builds confidence in the tool.

Acceptance Criteria:
- Required fields show validation errors inline when left empty or when the form is submitted.
- Name uniqueness is validated before submission (inline check against existing names).
- Containment cycle detection happens in real-time when adding associations — the UI prevents submission of cycle-creating associations.
- Successful save operations show a brief, non-blocking success indicator (e.g., toast notification).
- Failed API calls display human-readable error messages, not raw API responses or status codes.
- Destructive operations (delete entity type, remove attribute, demote from production) always require a confirmation dialog with a description of the consequences.

---

**US-38: Role-aware UI controls**
As a user of the meta operations UI, I want the interface to show or hide controls based on my role, so that I only see actions I am authorized to perform.

**Why**: Showing controls that the user cannot use (and returning 403s when clicked) is a poor user experience. Role-aware controls reduce confusion, prevent wasted clicks, and make the UI feel intentional rather than restrictive.

Acceptance Criteria:
- RO users see all meta configuration in read-only mode. No create, edit, delete, or promote buttons are visible.
- RW users see the same as RO for meta operations (RW only applies to entity instances, not meta configuration).
- Admin users see all edit controls except those restricted to Super Admin (e.g., modifying production catalog versions, demoting from production).
- Super Admin users see all controls.
- When a control is hidden due to role restrictions, no placeholder or "locked" indicator is shown — the control simply does not exist in the UI.
- When a control is disabled due to state restrictions (e.g., entity type in production, not a role issue), the control is visible but grayed out with a tooltip explaining why.

---

**US-39: Catalog-level access control**
As a platform administrator, I want to grant users read or write access to specific catalogs (not all catalogs globally), so that teams can only access the data they own or are responsible for.

**Why**: In a multi-team environment, different teams manage different catalogs (e.g., "team-alpha-prod", "team-beta-staging"). A global RW role that grants write access to all catalogs violates the principle of least privilege. Catalog-level access control ensures data isolation between teams without requiring separate Asset Hub deployments.

Acceptance Criteria:
- Per-catalog access is controlled via K8s RBAC using `resourceNames` on the `catalogs` resource. No custom ACL tables or user management APIs are introduced.
- Cluster admins can grant a user read or write access to specific catalogs by creating a RoleBinding with `resourceNames` listing the allowed catalog names.
- Users without a `resourceNames` restriction (i.e., a Role that grants access to all `catalogs` resources) retain access to all catalogs, preserving backward compatibility.
- The catalog list API (`GET /api/data/v1/catalogs`) returns only catalogs the requesting user is authorized to access.
- Accessing a catalog the user is not authorized for returns 403 with a clear error message.
- All sub-resource operations (instance CRUD, links, references) under a catalog inherit the catalog's access check — no separate per-instance authorization.
- In development mode (`RBAC_MODE=header`), the global role header applies to all catalogs (no per-catalog restriction), preserving the existing development workflow.
- No catalog-level permission management UI exists in the hub — admins use `oc`, `kubectl`, or the OCP console to manage RoleBindings.

---

**US-40: Operational data viewer UI**
As an operator or consumer, I want a dedicated read-only UI for browsing catalog data, separate from the admin/meta UI, so that I can discover and navigate assets without being exposed to schema management concerns.

**Why**: The meta UI is designed for administrators building and managing schemas and populating data. Operators and consumers need a simpler, read-optimized interface focused on browsing — finding assets, navigating containment hierarchies, following references, and filtering by attributes. Separating these concerns avoids overloading the admin UI and allows the operational viewer to be deployed independently with its own access controls.

Acceptance Criteria:
- The operational UI is a separate web application served at `/operational` (path-based routing on the same port as the meta UI), built from the same codebase with its own Vite entry point.
- The operational UI is read-only — no create, edit, or delete actions are available. All data modification is performed through the meta UI (see FF-6 for future editing support).
- The operational UI provides a catalog list page showing catalog name, pinned CV label, validation status, and instance counts.
- The operational UI provides a catalog detail page with an entity type overview (types with instance counts) and a containment tree browser.
- The containment tree browser uses a two-pane layout: the left pane shows the containment tree grouped by entity type with expandable headers; the right pane shows the selected instance's detail. No separate instance list table — the tree is the primary navigation for browsing instances.
- Instance detail shows all attribute values (with resolved enum names), description, version, and timestamps.
- Instance detail shows forward references ("References") and reverse references ("Referenced By") with clickable links that navigate to the referenced instance in the tree.
- Breadcrumb navigation shows the containment path from catalog root to the current instance.
- The backend supports attribute-based filtering (US-17), column sorting, and pagination via API query parameters, available for future use by the operational editing UI (FF-6).
- The operational UI shares types, API client, and utility code with the meta UI — no duplication of shared infrastructure.
- The meta UI's catalog detail page includes a link to open the same catalog in the operational data viewer, providing a seamless transition from editing to browsing.
- Deployment: a single nginx pod serves both the meta UI (at `/`) and the operational UI (at `/operational`) via path-based routing.
- Deployment: a single nginx pod serves both the meta UI (on port 30000) and the operational UI (on port 30001) via separate location blocks.

## 10. Open Design Decisions

The following items are acknowledged but not yet fully specified:

| Item | Notes |
|------|-------|
| ID calculation strategy | How entity IDs are generated (UUID, hash, deterministic from name+version, etc.) |
| Exact CRD schema | Partially resolved: `CatalogVersion` CRD defined for discovery. `Catalog` CRD for data discovery (scope TBD). Entity type CRD schema (full schema-as-CRD) remains open. |
| Entity type CRDs | Full schema-as-CRD feature where entity type definitions become native K8s CRDs. Future scope — separate from CatalogVersion discovery CRs. |
| Catalog version creation workflow | How an author assembles a catalog version from entity definition versions — manual selection vs. automatic "snapshot current state" |
| Concurrent editing model | Optimistic locking, pessimistic locking, or merge-based conflict resolution |
| Ad-hoc query language | Whether complex cross-entity queries (beyond filter+sort) will be needed in the future |
| Predefined queries | Which standard queries are provided out of the box |
| Entity instance versioning depth | Whether full version history is retained or only N recent versions |
| Technology choices | Backend language/framework, UI framework, API style (REST vs. GraphQL) |
| Centralized hub topology | Hub-and-spoke deployment where consuming clusters sync CatalogVersion CRs from a central API. See Section 8.4. |

## 11. Technical Debt

Items where the current implementation diverges from the intended behavior described in this PRD. These should be addressed in priority order.

| ID | Item | Current Behavior | Required Behavior |
|----|------|-----------------|-------------------|
| TD-1 | Enum deletion safety | Enum delete checks if any attribute references it across all entity type versions (flat check) | Enum cannot be deleted if it is used by any attribute in a **used entity version**. A used entity version is defined as: (1) any entity type version pinned by a catalog version, or (2) the latest version of any entity type (which belongs to an implicit pre-production catalog). Unused historical versions that are not pinned by any CV and are not the latest version should not block deletion. |
| TD-2 | Catalog version timestamp uniqueness | Two catalog versions can have the same `created_at` timestamp, causing non-deterministic sort order | `created_at` must be unique across catalog versions. The backend should enforce this (e.g., retry with a small delay if a timestamp collision is detected). This ensures deterministic sort order in the CV list (`ORDER BY created_at DESC`). |
| TD-3 | Association target+role uniqueness | No uniqueness constraint on (target entity type, target role) per source entity type version | Target entity type + target role must be unique per source entity type version. Empty target role is valid (one allowed per target). API should reject duplicates with 409 Conflict. |
| TD-4 | Copy attributes dialog: enum name display | Enum attributes in the copy-from picker show type label "enum" without the enum name | Enum attributes should display the enum name alongside the type (e.g., "enum (Month)") so users can distinguish between different enum types when deciding which attributes to copy. |
| TD-5 | Version lineage tracking | Entity type versions are sequential integers with no parent tracking. Version 4 is created from version 3, but this relationship is not recorded. | Each entity type version should record which version it was derived from (`parent_version_id`). This enables: (1) understanding the edit history as a DAG rather than a flat list, (2) supporting future scenarios where editing from a catalog version context creates a branch, (3) detecting when two catalog versions diverge from a common base version. **Decision: deferred for v1.** The current simple incrementing scheme is sufficient for the initial release. Revisit when implementing edit-from-CV-context or version branching features (see FF-3). |
| TD-6 | Duplicate DTO mapping logic | Attribute and Association model-to-DTO conversion is duplicated across handlers (attribute_handler, association_handler, entity_type_handler VersionSnapshot) | Extract shared helper functions (e.g., `dto.ToAttributeResponse`, `dto.ToAssociationResponse`) to eliminate duplication. All handlers should use these helpers instead of inline conversion loops. |
| TD-7 | Bidirectional association removal only from source | A bidirectional association can only be removed from the entity type that created it (the source/outgoing side). From the target entity type's Associations tab, the Remove button is hidden for incoming associations, including bidirectional ones. | Since bidirectional associations are symmetric, the Remove button should be available from either side. Removing from the target side should delete the same association record. The UI currently hides Remove for all incoming associations — bidirectional should be an exception. |
| TD-8a | Extract shared EditAssociationModal component | Edit association modal is duplicated between `App.tsx` (diagram edit) and `EntityTypeDetailPage.tsx` (associations tab edit) — ~110 lines of duplication | Extract into shared `ui/src/components/EditAssociationModal.tsx` with props for `showEntityTypeNames`, `allowTypeChange`, `onSave`. |
| TD-8b | Consolidate edit modal state into a single object | Diagram edit modal in `App.tsx` uses 12 separate `useState` calls for one form | Group into a single state object or move into the shared component from TD-8a. |
| TD-8c | Extract diagram data loading into a custom hook | `App.tsx` and `CatalogVersionDetailPage.tsx` both have `loadDiagramData` functions that fetch snapshots and build `DiagramEntityType[]` | Extract into `ui/src/hooks/useDiagramData.ts` with `loadFromAllEntityTypes()` and `loadFromPins(pins)` methods. |
| TD-8d | Extract EdgeClickData interface | `onEdgeClick` prop on `EntityTypeDiagramProps` uses inline type with 9 fields | Extract to a named `EdgeClickData` interface for reuse and readability. |
| TD-9 | Show required attributes in diagram | Diagram nodes list attributes without required indicator | Required attributes in diagram UML nodes should show `*` or bold to distinguish mandatory from optional attributes. |
| TD-10 | Mutable CVs in development mode | CV pins are immutable — entity types are pinned at creation and cannot be changed | In development stage, CVs should be mutable: add/remove entity type pins, change pinned versions. Pins are frozen only on promotion to testing. Catalogs cannot be created against a development-stage CV. If a CV with existing catalogs is demoted back to development, modified, and re-promoted, all catalogs pinned to that CV must be re-validated — they may become invalid if entity types were removed or attribute schemas changed. |
| TD-11 | Show mandatory associations in UI | Associations with cardinality `1` or `1..n` are not visually distinguished from optional ones | On the entity detail page, BOM modal, and diagram, show a mandatory indicator (e.g., `*` or bold) on the side of the association where cardinality starts with `1`. For example, mcp-tool's containment by mcp-server (cardinality `1` on source) shows as mandatory on the mcp-tool detail screen but NOT on the mcp-server detail screen (where the target cardinality `0..n` is optional). The indicator appears only from the perspective of the entity that is required to have the association. |
| TD-12 | Catalog re-pinning | A catalog's CV pin is immutable — to use a new CV, create a new catalog | Allow upgrading a catalog to a newer CV. Requires data migration validation: check that all entity instances are still valid under the new CV's schema. Report incompatibilities and let the user resolve them before completing the re-pin. |
| TD-13 | Get catalog version by name | CV can only be retrieved by ID; no lookup by version_label | Add `GET /api/meta/v1/catalog-versions/by-name/:label` endpoint for name-based lookup. The K8s CR already uses the label as its name. |
| TD-14 | Catalogs using this CV | CatalogVersion detail page does not show which catalogs are pinned to it | Add a "Catalogs" section on the CatalogVersion detail page listing catalogs pinned to that CV, with name, validation status, and link to catalog detail. |
| TD-15 | Catalog cascade delete needs transaction | `CatalogService.Delete` performs soft-delete of instances then hard-delete of catalog as two separate DB operations without a transaction | Wrap both operations in a database transaction. If the catalog delete fails after instances are soft-deleted, the system is left in an inconsistent state. Requires either passing a `*gorm.DB` transaction through context or introducing a unit-of-work pattern. The retry path currently works (soft-delete is idempotent on already-deleted rows), so this is a correctness improvement, not a data-loss risk. |
| TD-16 | Mixed soft-delete/hard-delete on catalog deletion | Instances are soft-deleted (`deleted_at` set) but the catalog itself is hard-deleted. Soft-deleted instances with no parent catalog accumulate as dead rows. | Either hard-delete instances when the catalog is deleted (since the catalog is hard-deleted, there's no recovery path anyway), or soft-delete the catalog too. Also consider a periodic cleanup job for orphaned soft-deleted instances. |
| TD-17 | Catalog list pagination | `ListCatalogs` handler hardcodes `Limit: 20` with no `offset`/`limit` query parameters | Accept `limit` and `offset` query parameters so clients can paginate through large catalog lists. The `total` count is already returned in the response. Apply the same fix to the instance list handler. |
| TD-18 | UI component props style inconsistency | Minor style issue: some components use a named `interface Props { ... }` for their parameter type (e.g., `EnumListPage`), while others use inline destructured types (e.g., `CatalogListPage`: `{ role }: { role: Role }`). Both are functionally identical. | Pick one convention and apply it consistently across all page components. The named `Props` interface is more common in the codebase and scales better when props grow. Low priority — a future style alignment pass. |
| TD-19 | N+1 query in resolveEntityType | `InstanceService.resolveEntityType` iterates all CV pins and calls `etvRepo.GetByID` for each to find the matching entity type | Replace the per-pin query loop with a batch fetch or a join query that resolves entity type ID → pinned version in one call. Acceptable for now since CVs typically have 3-5 pins; becomes a problem at 20+. |
| TD-20 | Missing name validation on instance creation | `CreateInstanceRequest.Name` has no `validate:"required"` tag and the handler does not validate the name before passing to the service. An empty-name instance can be created. | Add explicit name validation in the service layer (`name is required`, `name must not be empty`). Also consider a codebase-wide pass to add consistent `validate` tags across all DTOs — currently only `CreateEntityTypeRequest` uses them. |
| TD-21 | Remove catalog_version_id migration code | `InitDB` in `models.go` contains one-time migration code that detects and drops the old `catalog_version_id` column from `entity_instances`, copying data to `catalog_id`. This runs on every startup. | Once all environments (dev, staging, production) have been migrated, remove the migration block from `InitDB`. It is safe to remove after all databases have been started at least once with the current code. The migration is idempotent (no-ops if the old column doesn't exist) so there is no urgency. |
| TD-23 | CatalogDetailPage component too large | `CatalogDetailPage.tsx` is 429 lines with 14 `useState` calls managing catalog loading, instance loading, schema loading, enum caching, create/edit/delete modals — all in one file | Extract custom hooks (`useInstances`, `useSchema`) and split modals into sub-components. Follows the same pattern as `EntityTypeDetailPage` which is similarly large but could also benefit from extraction. |
| TD-24 | Remove legacy EntityInstanceService | Both `entity_instance_service.go` (legacy, CV-scoped) and `instance_service.go` (new, catalog-scoped) exist in the same `operational` package. The legacy service is still wired in `main.go` and its routes registered under `/api/data/v1/:catalog-version`. | Remove `entity_instance_service.go`, its tests, the legacy `Handler` in `handler.go`, and the legacy route registration in `main.go`. The new `InstanceService` and `InstanceHandler` replace them entirely. Also remove the legacy handler tests. |
| TD-25 | Replace `interface{}` with `any` across codebase | Multiple files use `interface{}` instead of the Go 1.18+ alias `any`. The `modernize` linter flags 15+ occurrences across DTOs, service, and test files. | Run a codebase-wide find-and-replace of `interface{}` → `any`. Low priority — purely cosmetic, no behavior change. |
| TD-26 | Extract shared instance creation logic (M5) | `CreateInstance` and `CreateContainedInstance` share ~70% of logic (instance model creation, attribute validation, persistence, validation status reset, attribute resolution). | Extract a private `createInstanceInternal` method that both call, passing `parentID` (empty for root) and containment validation as optional steps. |
| TD-27 | ListContainedInstances pagination broken by in-memory filtering (M7) | `ListContainedInstances` fetches children with `ListByParent` (which respects limit/offset), then filters by entity type in memory. This can return fewer results than limit or miss results entirely when the parent has children of multiple types. | Push the entity type filter into the repository query (add `ListByParentAndType` method), or fetch all children without pagination and paginate after filtering. |
| TD-29 | Reject reserved entity type names | Entity types named `links`, `references`, or `referenced-by` would shadow the static route segments used for association links and reference queries, making containment routes for those types unreachable. No server-side validation prevents creating such names. | Add validation in `CreateEntityType` and `RenameEntityType` to reject entity type names that collide with reserved operational API path segments: `links`, `references`, `referenced-by`. |
| TD-30 | Add catalog ownership check on instance read/update/delete | `GetInstance`, `UpdateInstance`, and `DeleteInstance` resolve the catalog from the URL but do not verify the fetched instance's `CatalogID` matches the resolved catalog. This means a request with a mismatched catalog name succeeds and resets the wrong catalog's validation status. Write operations (`CreateContainedInstance`, `CreateAssociationLink`) already perform this check. | Add `if inst.CatalogID != catalog.ID { return NotFound }` after `instRepo.GetByID` in `GetInstance`, `UpdateInstance`, and `DeleteInstance`. |
| TD-31 | Create new container from contained instance's Set Container modal | The Set Container modal only allows selecting existing parent instances. Users cannot create a new container directly from the child side — they must first create the container via the parent entity type tab, then come back and set it. | Add a "Create New" mode to the Set Container modal (similar to the "Create New / Adopt Existing" toggle in the Add Contained Instance modal). The natural flow is parent-first, so this is a convenience feature, not a critical gap. |
| TD-32 | Diagram: overlapping edges between same entity pair | When two or more associations exist between the same pair of entity types (e.g., mcp-tool → guardrail with both "uses" and "validates"), the edges overlap into a single line with two labels stacked on top of each other. | Add edge offset or curvature so multiple edges between the same pair are visually distinct. Dagre layout doesn't natively support parallel edges — options include: (a) adding a small vertical offset per duplicate edge, (b) using quadratic bezier curves with different control points, or (c) bundling labels into a single edge with a multi-line label. |
| TD-33 | "Contained by" flickers UUID before showing parent name | When opening instance details for a contained entity, the parent UUID briefly flashes before the async API call resolves the parent name. | Either (a) include `parent_instance_name` in the instance list API response so no extra fetch is needed, or (b) show a spinner/placeholder instead of the raw UUID while loading. Option (a) is cleaner — resolve the parent name server-side in `ListInstances`/`GetInstance`. |
| TD-34 | `SetParentRequest.ParentType` missing `validate:"required"` | When setting a parent, `ParentType` is logically required but lacks the `validate:"required"` tag. Empty `ParentType` with non-empty `ParentInstanceID` reaches the service and fails with a confusing `EntityType not found: ""` error instead of a clear 400. | Add `validate:"required"` to `ParentType` in `SetParentRequest`, or add explicit validation in the handler/service. |
| TD-35 | Operational catalog detail page too large | `OperationalCatalogDetailPage.tsx` manages 17+ state variables across 4 concerns (catalog metadata, tree state, instance detail/refs, instance list) in a single 500+ line component. | Extract the containment tree panel, instance detail drawer, and instance list table into separate sub-components to reduce cognitive load and improve testability. |
| TD-37 | Reference direction unclear in tree browser detail panel | In the instance detail panel, directional associations show under "Forward References" and "Referenced By" sections with a "Type" column showing "directional". It is not clear which direction the association goes — the user cannot tell whether the selected instance depends on the target or vice versa. The association name alone may not convey direction (e.g., "uses-model" is clear, but "related-to" is not). | Show an arrow or directional indicator in the reference table: e.g., "my-server → gpt-4" for forward refs and "monitor-1 → my-server" for reverse. Alternatively, use role labels from the association definition (source_role/target_role) to clarify the relationship semantics. Consider replacing the generic "directional" type label with the actual role or a "depends on" / "depended by" phrasing. |
| TD-38 | Entity type tab selector doesn't scale in meta catalog detail | The meta UI `CatalogDetailPage` uses PatternFly Tabs with one tab per entity type. When a catalog has many entity types (10+), the tabs overflow a single row and become hard to navigate. | Options: (A) Replace tabs with a sidebar or dropdown selector. (B) Add a search/filter input above the tabs. (C) Use PatternFly's scrollable tabs variant (`isOverflowHorizontal`). (D) Switch to a two-pane layout similar to the operational UI's tree browser. |
| TD-36 | Review usefulness of Overview tab in operational catalog view | The Overview tab shows entity type names, pinned versions, and a "Browse Instances" button per type. With the two-pane tree browser now grouping instances under entity type headers, the Overview tab is largely redundant — the only unique information it provides is the meta entity type version number. | Options: (A) Remove the Overview tab entirely and make the tree browser the default (and only) tab. Show entity type version info in the tree group headers (e.g., "mcp-server V3 (2)"). (B) Repurpose the Overview tab as a catalog dashboard with useful aggregate info: instance counts per type, validation summary, catalog metadata, recent changes. (C) Keep as-is for users who want a quick summary before diving into the tree. |
| TD-28 | Phase 3 code quality improvements (L1-L5, L7) | Multiple low-severity issues from quality review: (L1) duplicated forward/reverse reference handler conversion logic, (L2) dead `_ = parentInst`/`_ = sourceInst` assignments, (L3) JSON tags on service-layer `ReferenceDetail`, (L4) N+1 queries in `resolveLinks`, (L5) CatalogDetailPage now has ~30 state variables and should be decomposed, (L7) silently swallowed `UpdateValidationStatus` errors. | Extract `refsToDTO` helper in handler. Clean up dead assignments. Remove JSON tags from service types. Add batch fetch for links resolution. Decompose CatalogDetailPage into sub-components. Log validation status update failures. |
| **TD-22** | **[CRITICAL] Common attributes as schema-level attributes** | Common attributes (Name, Description) are fields on `EntityInstance` but are NOT represented as `Attribute` records in the entity type schema. They are invisible in attribute lists, diagrams, and BOM modals. If an entity type manually creates custom attributes named `name`/`description`, the create instance modal shows duplicate fields. | **Approach A (DB-level):** Make common attributes into real `Attribute` records: (1) Auto-create them when an entity type is created, marked with a `system: true` flag. (2) Prevent deletion of system attributes. (3) Show them in all views — attribute tabs, diagrams, BOM modals, create/edit modals. (4) Remove the hardcoded Name/Description fields from the instance create/edit modals — use the schema attributes instead. (5) Design for extensibility: future common attributes like `Version` (auto-incremented) and `State` (enum lifecycle) should follow the same pattern. **Approach B (API-level merge):** Keep common attributes as hardcoded fields on `EntityInstance` (no DB schema change). The API layer merges them into the dynamic attribute list when returning responses — injecting synthetic `name`, `description`, `version` entries at the top of the attributes array with a `system: true` marker. The UI renders all attributes uniformly from this merged list and prevents editing/removing system-flagged ones. Meta API endpoints (attribute list, version snapshot, diagram data) similarly inject common attributes into their responses. Simpler to implement (no migration, no COW implications), but common attributes are never real DB records — they exist only as API-level projections. Prevents the duplicate-fields bug by having a single source of truth for what attributes exist (the merged list). |

## 12. Future Features

Features planned for future implementation. These are not yet specified in detail and are not part of the current scope.

### FF-1: Association Cardinality (IMPLEMENTED — see US-2 and US-31)

Associations should support cardinality constraints on both ends (UML-style multiplicity notation).

**Standard cardinality options:**
- `0..1` — optional, at most one
- `0..n` — optional, any number
- `1` — exactly one (required)
- `1..n` — at least one (required)

**Custom cardinality:** In some cases, non-standard ranges are needed (e.g., `2..n`, `1..2`, `3..5`). The UI should support:
- A dropdown for the four standard options above
- A "Custom" option that reveals min/max input fields for arbitrary ranges
- Validation that min <= max, both are non-negative integers, and max can be `n` (unbounded)

Cardinality applies to both the source and target ends of the association. For example, a containment association between `Server` and `Tool` might have cardinality `1` on the source (a tool belongs to exactly one server) and `0..n` on the target (a server can have zero or more tools).

**Model changes:** Add `source_cardinality` and `target_cardinality` fields to the Association model (string, e.g., `"0..1"`, `"1..n"`, `"2..5"`). Default: `0..n` on both ends (no constraint) for backward compatibility.

### FF-2: Entity Version Labels

Entity type versions should support user-defined labels for easy identification, grouping, and catalog version assembly.

**Use cases:**
- Mark a version as "stable", "draft", "reviewed", "release-candidate"
- Group versions across entity types by label (e.g., label all versions as "Q1-2026-release") for easy catalog version creation
- Filter entity type versions by label when selecting which versions to pin in a catalog version

**Model changes:** Add a `labels` field to EntityTypeVersion — a set of string tags (e.g., `["stable", "Q1-2026"]`). Labels are free-form text, not predefined.

**UI:** Label badges on the version history table, label filter in the CV creation entity selection dialog, bulk-label operation to tag multiple entity type versions at once.

**API:**
- `PUT /entity-types/:id/versions/:version/labels` — set labels on a version
- `GET /entity-types?version_label=stable` — filter entity types by version label
- CV creation dialog: filter entity types by label to quickly select a coherent set of versions

### FF-3: Version Lineage and Edit-from-CV Context

Entity type versions currently form a flat sequence (V1, V2, V3...). A future enhancement would track version lineage — recording which version each new version was derived from — enabling a DAG-based version history.

**Motivation:** When viewing an entity type from a catalog version's BOM, the user may want to edit it in-place. Today, clicking an entity type from the BOM should show a read-only view of the pinned version (attributes + associations). Editing requires navigating to the entity type management area, making changes (which creates a new version), then updating the CV pin.

A future edit-from-CV workflow could:
- Create a new version derived from the pinned version (not necessarily the latest)
- Automatically update the CV pin to the new version
- Track that V5 was branched from V3 (not V4), enabling divergence detection

**Model changes:** Add `parent_version_id` (nullable UUID FK) to `EntityTypeVersion`. For the initial version (V1), this is null. For subsequent versions, it points to the version that was the base for the copy-on-write operation.

**Depends on:** TD-5 (version lineage tracking).

**Decision:** Deferred. The v1 read-only BOM view is sufficient. Revisit when user feedback indicates demand for in-place editing from the CV context.

### FF-4: Edit Catalog Version

Catalog versions are currently immutable after creation — pins (entity type version selections) cannot be changed, and there is no way to add or remove entity types from an existing CV. A future enhancement would allow editing a catalog version's pins.

**Potential capabilities (details TBD):**
- Add or remove entity type pins from an existing catalog version
- Change the pinned version for an entity type (e.g., upgrade from V2 to V3)
- Possibly restricted by lifecycle stage (e.g., only development-stage CVs are editable)

**Decision:** Deferred. Requirements not yet clear. Revisit when usage patterns emerge.

### FF-5: Configurable Diagram Layout

The entity type diagram currently uses a hardcoded Dagre (hierarchical top-to-bottom) layout algorithm. A future enhancement would make the layout algorithm configurable at runtime — either per-user preference or as a UI dropdown — without requiring recompilation.

**Supported layouts** (all available in `@patternfly/react-topology`):
- Dagre (current default — hierarchical, instant positioning)
- Cola (force-directed, iterative settling)
- Force (D3 force simulation)
- Concentric, Grid, BreadthFirst

**Decision:** Deferred. Current Dagre layout works well for the UML class diagram use case.

### FF-6: Operational UI Editing

The operational UI (Phase 4) is initially read-only — a data viewer for operators and consumers to browse, filter, and navigate catalog data. A future enhancement would add write capabilities to the operational UI, allowing authorized users to create, edit, and delete instances, manage containment, and create association links directly from the operational interface.

**Motivation:** Some teams may prefer a single UI for both browsing and editing catalog data, rather than switching between the meta UI (editing) and operational UI (browsing). This is especially relevant when catalog-level RBAC (US-39) is in place — an operator with write access to a specific catalog should be able to edit it from the same interface they use to browse it.

**Scope:** Reuse the existing create/edit/delete modals from the meta UI's `CatalogDetailPage`, adapted for the operational app shell. Role-aware rendering (RO vs RW) determines which controls are visible.

**Decision:** Deferred. The read-only viewer must be validated with users first. Editing features can be added incrementally once the browsing UX is stable.

