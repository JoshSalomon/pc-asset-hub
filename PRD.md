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
| Name        | **Required.** Unique within scope (global for top-level entities, within parent for contained entities) |
| Description | **Optional.** Free-text description |
| Version     | Auto-incremented on mutation |

**System attributes in API responses:** The common attributes Name (required) and Description (optional) are surfaced as **system attributes** in all API responses — attribute lists, version snapshots, instance responses, and UML diagrams. They carry a `system: true` marker that distinguishes them from user-defined custom attributes. Name is marked as required; Description is optional. System attributes are always present, always appear first in attribute lists, and cannot be created, renamed, or deleted by users. Custom attribute names "name" and "description" are reserved and rejected on creation to prevent conflicts. This is implemented via API-level merge (Approach B from TD-22): common attributes remain as fields on `EntityInstance` in the database, and the API layer injects synthetic attribute entries into responses.

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

**Publishing** is a separate, explicit action (not automatic on validation). Only `valid` catalogs can be published. Published catalogs are write-protected — data mutations require SuperAdmin role. See US-42, US-43.

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
- Lightweight CRs created when an Admin explicitly **publishes** a catalog. Publishing is a manual action, not automatic on validation — a catalog must be `valid` to publish, but not all valid catalogs are published.
- Publishing creates the Catalog CR; unpublishing deletes it. Going to `draft` (after a data mutation) does NOT auto-unpublish — the CR remains until explicitly unpublished, representing the last validated state.
- Data mutations on published catalogs require SuperAdmin role. RW users can only modify unpublished catalogs. This protects production data — use the Copy & Replace workflow (FF-8) for safe edits.
- Contain catalog name, pinned CV reference, API endpoint, catalog ID, validation status, and published timestamp — enough for applications to discover available catalogs and query their data via the Operational API.
- **Scoping: namespaced**, in the same namespace as the AssetHub CR (consistent with CatalogVersion CRs). Owner reference to the AssetHub CR enables garbage collection on operator uninstall. Multi-namespace publishing (creating Catalog CRs in consumer namespaces) is a future enhancement (see FF-9).

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
- System attributes (Name, Description) appear inline in the attribute list with a "System" badge. They cannot be edited, renamed, or deleted.
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
- The entity type detail view shows all attributes (system and custom) in an ordered list.
- System attributes (Name, Description) appear first with a "System" badge and cannot be edited, removed, or reordered.
- Each attribute displays: name, description, type (string/number/enum name), and whether it is required.
- Custom attributes can be reordered (drag-and-drop or move up/down controls).

**Add attribute**
- Inline form or dialog to add a new attribute: name (required), description (optional), type (required — dropdown of string, number, or existing enums).
- For enum types, the dropdown lists all centrally-defined enums by name. The Admin can also create a new enum inline (see 6.1.4).
- Validation: attribute name must be unique within the entity type. Names "name" and "description" are reserved for system attributes and rejected.

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

The system leverages OpenShift's native RBAC — no duplication of OCP access control functionality. Access control is two-dimensional: **meta** (schema) and **catalog** (data) access are controlled independently, with catalog access granular to individual catalogs.

### 7.1 Two Access Domains

| Domain | Scope | K8s Resource | Granularity |
|--------|-------|-------------|-------------|
| **Meta** | Entity types, attributes, associations, enums, catalog versions, lifecycle | `entitytypes` | All-or-nothing (all entity types) |
| **Catalog** | Instances, attribute values, links, validation, publishing | `catalogs` | Per-catalog via `resourceNames` |

A user can have different roles in each domain. For example, user X can be Admin on meta (manages schema) but have no catalog access. User Y can be RO on meta (browses schema) but Admin on catalog "team-alpha-prod" and have no access to catalog "team-beta-staging".

### 7.2 Role Definitions

#### Meta Roles

| Role | Permissions |
|------|-------------|
| **Meta Viewer** | Read-only access to entity types, attributes, associations, enums, catalog versions, version history. |
| **Meta Editor** | Create, update, and delete entity types, attributes, associations, enums. Create catalog versions in development. |
| **Meta Admin** | All Meta Editor permissions plus lifecycle management: promote dev→test→production, demote test→dev. |
| **Meta Super Admin** | All Meta Admin permissions plus demote from production (production→test, production→dev). Modify meta configuration even when the catalog version is in production. |

#### Catalog Roles (per-catalog)

| Role | Permissions |
|------|-------------|
| **Catalog Viewer** | Read-only access to instances, tree, references, attribute values. Can browse but not modify. |
| **Catalog Editor** | Create, update, and delete instances, links. Set parent. All viewer permissions. |
| **Catalog Admin** | All editor permissions plus validate and publish/unpublish. |
| **Catalog Super Admin** | All admin permissions plus modify published catalogs (bypasses write protection). Intended for emergency fixes, security patches, and regulatory changes on specific catalogs. |

There is no global "super admin" role at the application level. Each catalog may be managed by a different team for a different workload — a Catalog Super Admin on catalog X has no implicit access to catalog Y. Platform-wide administrative access (e.g., for disaster recovery) is handled by the K8s cluster-admin role, which bypasses all RBAC checks at the infrastructure level.

### 7.3 K8s RBAC Mapping

Each role maps to a K8s ClusterRole (for meta, which is namespace-independent) or Role (for catalogs, scoped to the AssetHub namespace):

```yaml
# Meta Viewer — browse schema
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: assethub-meta-viewer
rules:
  - apiGroups: ["assethub.example.com"]
    resources: ["entitytypes", "catalogversions", "enums"]
    verbs: ["get", "list"]

# Catalog Editor on specific catalogs — uses resourceNames
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: user-y-catalog-access
  namespace: assethub
rules:
  - apiGroups: ["assethub.example.com"]
    resources: ["catalogs"]
    resourceNames: ["team-alpha-prod"]
    verbs: ["get", "create", "update", "delete"]
  - apiGroups: ["assethub.example.com"]
    resources: ["catalogs"]
    resourceNames: ["team-beta-staging"]
    verbs: ["get"]
```

### 7.4 Application Access

Applications (pipelines, dashboards, monitoring) access catalogs via K8s ServiceAccounts. The same RBAC model applies — a ServiceAccount gets a RoleBinding with `resourceNames` restricting it to specific catalogs:

```yaml
# ML pipeline ServiceAccount: read-only access to "ml-models" catalog only
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: ml-pipeline-catalog-access
  namespace: assethub
subjects:
  - kind: ServiceAccount
    name: ml-pipeline
    namespace: ml-team
roleRef:
  kind: Role
  name: assethub-catalog-viewer-ml-models
```

This ensures application isolation — an application with access to catalog X cannot see or modify any other catalog.

### 7.5 Development Mode

In development mode (`RBAC_MODE=header`), the `X-User-Role` header sets a single global role (RO/RW/Admin/SuperAdmin) that applies to both meta and all catalogs. Per-catalog restrictions are not enforced in development mode. This preserves the existing development workflow.

### 7.6 Example Scenarios

| User | Meta Role | Catalog Access | Result |
|------|-----------|---------------|--------|
| Schema architect | Meta Admin | No catalog access | Can design entity types, promote CVs. Cannot see or modify any catalog data. |
| Data operator | Meta Viewer | Catalog Editor on "prod-catalog" | Can browse schema (read-only). Can create/edit instances in "prod-catalog" only. |
| Dashboard app (SA) | None | Catalog Viewer on "ml-models" | Can read instances from "ml-models". Cannot see schema or other catalogs. |
| Workload owner | Meta Viewer | Catalog Super Admin on "team-a-prod" | Can browse schema. Can modify "team-a-prod" even when published (emergency fixes). Cannot touch any other catalog. |
| Team lead | Meta Viewer | Catalog Admin on "team-a-staging", Catalog Viewer on "team-b-prod" | Can browse schema. Can publish "team-a-staging". Can browse but not modify "team-b-prod". |
| K8s cluster-admin | (bypasses all RBAC) | (bypasses all RBAC) | Full access everywhere via K8s infrastructure role. Not an application-level role. |

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

The current architecture assumes a single-cluster deployment where the API server, database, and operator all run together. A future enhancementr will support a **hub-and-spoke topology** where one central AssetHub instance serves an entire organization across multiple clusters.

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
- Every entity type automatically includes system attributes: Name (required) and Description (optional). These are visible in all views (attribute lists, diagrams, instance modals). Custom attributes named "name" or "description" are rejected to prevent conflicts with system attributes.
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
- Admin can create a named enum with an optional description and an ordered list of allowed values.
- Admin can update an enum's description and values (add, remove, reorder).
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
- RW (and above) users can create a new catalog version by specifying a version label, an optional description, and selecting specific entity definition versions to include.
- The catalog version records the exact `(entity type name, version)` tuples it contains.
- Once created, the catalog version's entity version pins cannot be changed (immutable snapshot). The description can be updated.
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
- Validation checks: required attributes have values, attribute values match their type, mandatory associations are satisfied (cardinality `1` or `1..n`), containment hierarchy is consistent, and the Name system attribute is non-empty for all instances.
- Validation returns a list of errors (entity name, field, violation) — not just pass/fail.
- Validation status is updated to `valid` (no errors) or `invalid` (errors found).
- Any subsequent data change resets the status to `draft`.
- Publishing a catalog is a separate action (see US-42) — validation is a prerequisite, not a trigger.

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
- The response includes the instance with resolved attribute values (attribute name, type, and value — not just raw IDs). System attributes — Name (required) and Description (optional) — are included in the attribute list with `system: true`, alongside custom attributes.
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
As a Meta Viewer or Catalog Viewer, I want to browse entities, configuration, and catalog data without being able to modify anything, so that I can consume asset information safely.

**Why**: Many consumers of the asset hub (dashboards, monitoring systems, downstream services) only need read access. Enforcing read-only access prevents accidental modifications from non-privileged users.

Acceptance Criteria:
- Meta Viewers can perform all GET operations on the Meta API (entity types, attributes, associations, enums, catalog versions). They receive 403 on any POST, PUT, PATCH, or DELETE operation on meta resources.
- Catalog Viewers can perform all GET operations on catalogs they are authorized to access. They receive 403 on any POST, PUT, PATCH, or DELETE operation on catalog data.
- A user with Meta Viewer role and no catalog role cannot access any catalog data (receives 403 on all `/api/data/v1/catalogs/*` requests).
- A user with Catalog Viewer role and no meta role cannot access any meta data (receives 403 on all `/api/meta/v1/*` requests).
- Read-only access is enforced via OpenShift RBAC (SubjectAccessReview), not application-level checks.

---

**US-23: Role-based access enforcement via OpenShift RBAC**
As a platform administrator, I want access control enforced via OpenShift's native RBAC with independent meta and catalog roles, so that there is no duplication of OCP identity and access management and users get the minimum privileges they need.

**Why**: OpenShift already provides a mature, audited RBAC system. Duplicating it would create security gaps. Separating meta and catalog roles allows schema architects to work without accessing production data, and data operators to work without modifying the schema.

Acceptance Criteria:
- Meta roles (Meta Viewer, Meta Editor, Meta Admin) map to K8s ClusterRoles on the `entitytypes`, `catalogversions`, and `enums` resources in the `assethub.example.com` API group.
- Catalog roles (Catalog Viewer, Catalog Editor, Catalog Admin) map to K8s Roles with per-catalog granularity via `resourceNames` on the `catalogs` resource.
- There is no application-level global super admin. Catalog Super Admin is per-catalog, scoped via `resourceNames`. Platform-wide access uses the K8s cluster-admin role.
- A user can hold different roles in meta vs catalogs (e.g., Meta Viewer + Catalog Admin on "prod-catalog").
- A user can hold different catalog roles on different catalogs (e.g., Catalog Admin on "team-a", Catalog Viewer on "team-b").
- Authentication is handled by OpenShift (no separate user database or login system).
- Authorization decisions use the OCP RBAC API (SubjectAccessReview). The API server checks both the resource type (`entitytypes` or `catalogs`) and the resource name (catalog name) for each request.
- Adding or removing users from roles is done via standard OpenShift tooling (`oc`, `kubectl`, or the OCP console) by creating/modifying RoleBindings.
- No custom user management UI or API exists in the hub.
- The operator ships predefined ClusterRoles/Roles for all role levels. Cluster admins only need to create RoleBindings to assign users.
- In development mode (`RBAC_MODE=header`), the `X-User-Role` header sets a single global role (RO/RW/Admin/SuperAdmin) that applies to both meta and all catalogs. Per-catalog and per-domain restrictions are not enforced in development mode.

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
- System attributes (Name, Description) appear inline in the attribute list with a `system` indicator. They cannot be deleted or renamed by users.
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
- Attribute names are validated for uniqueness within the entity type before submission. Duplicate names show an inline error. Names "name" and "description" are reserved for system attributes and rejected.
- Admin can edit an existing attribute's name, description, or type.
- Admin can remove an attribute. A confirmation dialog warns that this will increment the entity type version.
- System attributes (Name, Description) cannot be edited, removed, or reordered. The UI disables these controls for system attributes.
- Custom attributes can be reordered (drag-and-drop or move up/down controls).
- When selecting enum as the type, the Admin can create a new enum inline without leaving the page (links to US-33).

---

**US-29: Copy attributes from another entity type in the UI**
As an Admin, I want to browse other entity types and copy selected attributes to the current entity type, so that I can reuse existing attribute definitions without re-entering them.

**Why**: Many entity types share similar attributes. A copy picker is faster and less error-prone than manually recreating attributes, especially when the attribute has an enum type that must be referenced correctly.

Acceptance Criteria:
- A picker/dialog allows the Admin to browse all entity types and their attributes.
- The picker shows each attribute's name, description, and type for informed selection.
- System attributes (Name, Description) are excluded from the copy picker since they already exist on all entity types.
- Admin can select one or more custom attributes to copy.
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
- An enum list view displays all enums with: name, description, number of values, and a list of referencing entity types and attributes.
- Admin can create a new enum with a name, optional description, and an ordered list of values.
- Admin can edit an enum: update description, add, remove, or reorder values.
- When removing a value, the UI warns if existing entity instances may use that value.
- Admin can delete an enum only if no attributes reference it. If references exist, the UI lists them and blocks deletion.
- Enum creation is also available inline from the attribute type dropdown (without navigating away from the entity type view).

---

**US-41: Catalog version creation in the UI**
As an Admin, I want to create a catalog version by selecting specific entity definition versions from a list, so that I can assemble a bill of materials for deployment.

**Why**: The catalog version is what gets deployed. The selection interface must make it easy to choose the right versions and review the complete snapshot before committing, to avoid deploying unintended entity definition versions.

Acceptance Criteria:
- The creation modal includes a version label (required) and an optional description.
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
- The detail view shows: catalog version identifier, description, current lifecycle stage (visually indicated with color-coded badge), creation date, and the full bill of materials (entity type name + pinned version).
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
As a user of the meta and operational UIs, I want the interface to show or hide controls based on my effective role in each domain (meta and per-catalog), so that I only see actions I am authorized to perform.

**Why**: Showing controls that the user cannot use (and returning 403s when clicked) is a poor user experience. Role-aware controls reduce confusion, prevent wasted clicks, and make the UI feel intentional rather than restrictive. With the two-domain role model, the UI must adapt to the user's meta role AND their per-catalog role independently.

Acceptance Criteria:
- **Meta UI**: Controls adapt to the user's meta role. Meta Viewers see all configuration in read-only mode (no create/edit/delete/promote buttons). Meta Editors see schema editing controls but not lifecycle management. Meta Admins see all controls except Super Admin actions.
- **Operational UI**: Controls adapt to the user's catalog role for the current catalog. Catalog Viewers see browse-only interface (no create/edit/delete). Catalog Editors see instance CRUD controls. Catalog Admins see validate and publish buttons. Catalog Super Admins see all controls including editing published catalog data.
- A user with no meta role sees an empty or inaccessible meta UI. A user with no catalog access sees an empty catalog list.
- When a control is hidden due to role restrictions, no placeholder or "locked" indicator is shown — the control simply does not exist in the UI.
- When a control is disabled due to state restrictions (e.g., entity type in production, published catalog), the control is visible but grayed out with a tooltip explaining why.
- In development mode, the role dropdown continues to set a single global role that applies to both meta and all catalogs.

---

**US-39: Per-catalog access control for users and applications**
As a platform administrator, I want to grant users and application ServiceAccounts read or write access to specific catalogs (not all catalogs globally), so that teams and applications can only access the data they own or are authorized to use.

**Why**: In a multi-team environment, different teams manage different catalogs (e.g., "team-alpha-prod", "team-beta-staging"). A global RW role that grants write access to all catalogs violates the principle of least privilege. Catalog-level access control ensures data isolation between teams and between applications without requiring separate Asset Hub deployments.

Acceptance Criteria:
- Per-catalog access is controlled via K8s RBAC using `resourceNames` on the `catalogs` resource in the `assethub.example.com` API group. No custom ACL tables or user management APIs are introduced.
- Cluster admins can grant a user or ServiceAccount read, write, or admin access to specific catalogs by creating a RoleBinding with the appropriate catalog role and `resourceNames` listing the allowed catalog names.
- Users without a `resourceNames` restriction (i.e., a Role that grants access to all `catalogs` resources) retain access to all catalogs, preserving backward compatibility.
- The catalog list API (`GET /api/data/v1/catalogs`) returns only catalogs the requesting user/ServiceAccount is authorized to access. The API server performs a SubjectAccessReview for each catalog and filters the result set. Catalogs the user cannot access are excluded silently (not returned with 403).
- Accessing a specific catalog the user is not authorized for returns 403 with a clear error message.
- All sub-resource operations (instance CRUD, links, references, validation, publishing) under a catalog inherit the catalog's access check — no separate per-instance authorization.
- **Application isolation**: A ServiceAccount bound to a catalog role with `resourceNames: ["catalog-x"]` can only access catalog X. It cannot list, read, or modify any other catalog. This is the standard pattern for granting API access to pipelines, dashboards, and downstream services.
- The API server maps HTTP verbs to K8s verbs for SAR checks: GET → `get`, POST → `create`, PUT/PATCH → `update`, DELETE → `delete`. Validate maps to `create` on the `catalogs/validate` sub-resource. Publish/unpublish maps to `create` on `catalogs/publish`.
- In development mode (`RBAC_MODE=header`), the global role header applies to all catalogs (no per-catalog restriction), preserving the existing development workflow.
- No catalog-level permission management UI exists in the hub — admins use `oc`, `kubectl`, or the OCP console to manage RoleBindings.
- The operator ships predefined Roles for each catalog access level (Catalog Viewer, Catalog Editor, Catalog Admin). Cluster admins create RoleBindings that bind users/ServiceAccounts to these roles with `resourceNames` for the specific catalogs.

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
- The operational catalog detail page includes a "Schema Diagram" tab showing the catalog version's entity type diagram (same `EntityTypeDiagram` component used in the meta UI). The diagram shows all pinned entity types with their attributes, associations, and cardinality — read-only, no edit interactions. This helps operational users understand the data model without needing meta UI access.
- The backend supports attribute-based filtering (US-17), column sorting, and pagination via API query parameters, available for future use by the operational editing UI (FF-6).
- The operational UI shares types, API client, and utility code with the meta UI — no duplication of shared infrastructure.
- The meta UI's catalog detail page includes a link to open the same catalog in the operational data viewer, providing a seamless transition from editing to browsing.
- Deployment: a single nginx pod serves both the meta UI (at `/`) and the operational UI (at `/operational`) via path-based routing.
- Deployment: a single nginx pod serves both the meta UI (on port 30000) and the operational UI (on port 30001) via separate location blocks.

---

**US-42: Publish a catalog**
As an Admin, I want to explicitly publish a validated catalog so that it becomes discoverable via a K8s Catalog CR, and I want to control when this happens rather than it being automatic.

**Why**: Publishing makes catalog data visible to external consumers via K8s discovery. This should be a deliberate action — not every valid catalog should be automatically published (e.g., staging catalogs, test data). Explicit publishing gives administrators control over what is exposed to the cluster.

Acceptance Criteria:
- Admin (and above) can publish a catalog. RW and RO users cannot.
- Publishing requires validation status `valid`. Attempting to publish a `draft` or `invalid` catalog returns 400.
- Publishing creates a Catalog CR in K8s with catalog name, CV reference, API endpoint, catalog ID, validation status, and published timestamp.
- A published catalog that goes to `draft` (due to data mutation) does NOT auto-unpublish. The Catalog CR remains, representing the last validated state.
- Admin can explicitly unpublish a catalog, which deletes the Catalog CR.
- The catalog list and detail pages show a "published" indicator.
- A "Publish" button appears on the catalog detail page for Admin+ users when the catalog is `valid` and not yet published.
- An "Unpublish" button appears on published catalogs for Admin+ users.

---

**US-43: Published catalog write protection**
As a platform administrator, I want data mutations on published catalogs restricted to Catalog Super Admin only, so that production data is protected from accidental edits by Catalog Editors and Catalog Admins.

**Why**: Published catalogs serve as the source of truth for external consumers. Accidental edits could corrupt production data. Restricting writes to Catalog Super Admin ensures that only deliberate, authorized changes are made — and only by users who are Super Admin on that specific catalog (not globally). For routine catalog updates, the Copy & Replace workflow (FF-8) provides a safe staging pattern that doesn't require Super Admin access. Note: Catalog Super Admin is per-catalog — being Super Admin on catalog X grants no privileges on catalog Y.

Acceptance Criteria:
- Data mutations (create/update/delete instance, create/delete link, set parent) on a published catalog require SuperAdmin role. RW and Admin users receive 403.
- RW users see instance create/edit/delete controls disabled on published catalogs with a tooltip explaining the restriction.
- Published catalogs show a warning banner: "This catalog is published. Editing requires SuperAdmin privileges."
- SuperAdmin edits still reset validation status to `draft` (same behavior as unpublished catalogs).
- Validation (POST .../validate) remains available to RW+ users on published catalogs — it is a read operation.
- Catalog-level operations (delete catalog, unpublish) require Admin+ regardless of published state.

---

### Copy & Replace Catalog

Two operations that enable a staging workflow for extending published catalogs without disrupting them.

**Problem:** Adding instances to a `valid` published catalog resets it to `draft`, temporarily removing it from K8s discovery. Users need a way to prepare changes in isolation and swap atomically.

**US-44: Copy Catalog**

As an RW user, I want to deep-clone an existing catalog into a new one so that I can prepare changes without affecting the original.

**Why**: Published catalogs are write-protected (US-43). Users need to copy a published catalog into a staging copy, make changes there, validate, and then swap it back in. Without Copy, users would have to manually recreate all instances, attributes, and links — error-prone and time-consuming for catalogs with hundreds of instances.

Acceptance Criteria:
- AC-44.1: `POST /api/data/v1/catalogs/copy` with `{source, name, description?}` creates a new catalog with the same CV pin, `draft` status
- AC-44.2: All entity instances are cloned with new UUIDs, same entity type, name, description, version reset to 1
- AC-44.3: All instance attribute values are cloned and remapped to new instance IDs
- AC-44.4: All association links are cloned and remapped to new source/target instance IDs
- AC-44.5: Containment hierarchy is preserved — parent references remapped to new instance IDs
- AC-44.6: Copy is transactional — all-or-nothing (if any step fails, no partial data is created)
- AC-44.7: Source catalog name must exist; returns 404 if not found
- AC-44.8: Target catalog name must be DNS-label compatible and unique; returns 409 if name taken
- AC-44.9: Requires RW+ access (create verb checked on the new catalog name)
- AC-44.10: Returns 201 with the new catalog details

---

**US-45: Replace Catalog**

As an Admin user, I want to atomically swap a staging catalog into the name of a published one so that I can update production data without downtime.

**Why**: The Copy & Replace workflow is the safe alternative to editing published catalogs directly. Replace performs an atomic name swap — the published catalog name now serves the staging catalog's data, and the old data is archived under a new name for rollback. External consumers (watching the Catalog CR) see a DataVersion bump and know to refresh.

Acceptance Criteria:
- AC-45.1: `POST /api/data/v1/catalogs/replace` with `{source, target, archive_name?}` swaps the catalogs
- AC-45.2: Source catalog must be `valid`; returns 400 if `draft` or `invalid`
- AC-45.3: Target catalog must exist; returns 404 if not found
- AC-45.4: Source catalog must exist; returns 404 if not found
- AC-45.5: Target is renamed to archive name (default: `{target}-archive-{timestamp}`)
- AC-45.6: Source is renamed to target's original name
- AC-45.7: Replace is transactional — both renames succeed or neither does
- AC-45.8: If target was published, the source (now with target's name) inherits published state; archive is unpublished
- AC-45.9: The Catalog CR (if target was published) continues to reference the same name — now serves new data; CR spec is updated via SyncCR
- AC-45.10: The Catalog CR's DataVersion is bumped after replace so that consumers watching the CR know to invalidate their cache
- AC-45.11: Archive name must be DNS-label compatible; returns 400 if invalid
- AC-45.12: Requires Admin+ access
- AC-45.13: Returns 200 with the updated catalog (source, now renamed to target)

---

**US-46: Copy & Replace UI**

As an admin using the meta UI, I want Copy and Replace buttons on the catalog detail page so that I can perform the staging workflow through the UI.

**Why**: The staging workflow (copy → edit → validate → replace) should be accessible through the UI, not just via API. The Copy button is available on any catalog; the Replace button is available on valid staging catalogs to swap them into a published catalog's name.

Acceptance Criteria:
- AC-46.1: "Copy" button on catalog detail page opens a modal with new name input
- AC-46.2: Copy modal validates DNS-label format before submitting
- AC-46.3: "Replace" button visible on `valid` staging catalogs opens a modal to select target catalog and optional archive name
- AC-46.4: Replace modal shows target catalog dropdown (filtered to existing catalogs)
- AC-46.5: Archive catalogs are visible in the catalog list as normal catalogs

**Staging workflow:**
1. `prod-catalog` is `valid` and published.
2. Copy: `prod-catalog` → `prod-catalog-next`
3. Edit `prod-catalog-next`: add/modify/delete instances.
4. Validate `prod-catalog-next` — must reach `valid`.
5. Replace: `source=prod-catalog-next, target=prod-catalog` → atomically swaps them. Old data archived as `prod-catalog-archive-20260316`.
6. The Catalog CR for `prod-catalog` now serves the updated data. Rollback: replace back from the archive.

**Rollback:**
- The archived catalog is a normal catalog — it can be browsed, validated, and used as a replace source to roll back.

---

**US-47: Landing page + unified SPA**
As a user, I want a landing page at the root URL (`/`) that shows me available actions based on my permissions, so that I can navigate directly to schema management or to a specific catalog without memorizing URL paths.

**Why**: The current root URL goes directly to the meta UI (entity type list), which is irrelevant for users who only have catalog access and confusing as a first experience for all users. A landing page that adapts to the user's permissions provides a clear, role-appropriate entry point — schema architects see meta tools, data operators see their catalogs, and users with both see everything in one place.

**Architecture decision (resolved):** Merge the two separate SPAs (meta `App.tsx` + operational `OperationalApp.tsx`) into a single SPA with route-based views. The separate `operational.html` entry point, `OperationalApp.tsx`, and `main-operational.tsx` are removed. The nginx config simplifies to API proxy + single SPA catch-all. This eliminates the artificial meta/operational split and gives users a unified experience.

**Design decision (resolved):** Minimal navigation cards (Option A). Dashboard stats deferred to future iteration.

URL structure:
- `/` — Landing page
- `/schema` — Schema management (entity types, catalog versions, enums, model diagram)
- `/schema/entity-types/:id` — Entity type detail
- `/schema/catalog-versions/:id` — Catalog version detail
- `/schema/catalogs/:name` — Catalog detail (with instance CRUD)
- `/catalogs/:name` — Catalog data viewer (read-only tree browser + model diagram)

Acceptance Criteria:
- The root URL (`/`) renders a landing page with navigation cards.
- **Schema Management card**: Links to `/schema`. Visible for all roles in development mode. In production mode, visible only if the user has a meta role (future — deferred to RBAC Phase C).
- **Catalogs section**: Shows a card for each accessible catalog displaying: catalog name, description, pinned CV label, validation status badge, and published indicator. Clicking navigates to `/catalogs/{catalog-name}`.
- A user with no meta role and no catalog access sees an appropriate empty state (e.g., "No resources available. Contact your administrator.").
- In development mode, the landing page shows all sections (meta + all catalogs) since the global role grants access to everything.
- The landing page loads quickly — catalog list is fetched with a single API call and filtered by access on the server side (see US-39).
- The separate operational SPA (`OperationalApp.tsx`, `main-operational.tsx`, `operational.html`) is removed. All routes are served by a single SPA.
- The masthead shows "AI Asset Hub" with a role selector. On schema pages, the masthead shows "AI Asset Hub — Schema". On catalog viewer pages, the masthead shows "AI Asset Hub — Data Viewer". A home icon or "AI Asset Hub" link in the masthead navigates back to the landing page.
- Nginx config simplifies to: `/api/` proxy + `try_files $uri $uri/ /index.html` catch-all.
- Vite build produces a single `index.html` entry point (no `operational.html`).

---

**US-48: Model Diagram tab on Catalog Detail Pages**
As a user viewing a catalog (in either the meta or operational UI), I want a "Model Diagram" tab that shows the entity type model diagram for the catalog's pinned catalog version, so that I can understand the data model without navigating to the catalog version detail page.

**Why**: Catalogs are the primary working context for data operators. The entity type model (entity types, attributes, associations, containment) is essential context when creating or browsing instances. Currently, viewing the model diagram requires navigating away to the catalog version detail page and finding the Diagram tab. A diagram tab directly on the catalog page keeps this context one click away.

Acceptance Criteria:
- Both the meta catalog detail page (`CatalogDetailPage`) and the operational catalog detail page (`OperationalCatalogDetailPage`) have a "Model Diagram" tab.
- The tab renders the `EntityTypeDiagram` component showing all entity types pinned in the catalog's catalog version, with their attributes and associations.
- The diagram is **read-only** — no edit interactions (no node double-click navigation, no edge click editing).
- Diagram data is loaded on-demand when the tab is first selected (not on page load).
- The diagram includes the TD-47 UML composition diamond notation for containment edges.
- If the catalog has no pinned entity types, the tab shows an appropriate empty state.

---

## 10. Open Design Decisions

The following items are acknowledged but not yet fully specified:

| Item | Notes |
|------|-------|
| ID calculation strategy | How entity IDs are generated (UUID, hash, deterministic from name+version, etc.) |
| Exact CRD schema | Partially resolved: `CatalogVersion` CRD defined for discovery. `Catalog` CRD for data discovery — namespaced in AssetHub namespace (resolved), multi-namespace via FF-9 (future). Entity type CRD schema (full schema-as-CRD) remains open. |
| Entity type CRDs | Full schema-as-CRD feature where entity type definitions become native K8s CRDs. Future scope — separate from CatalogVersion discovery CRs. |
| Catalog version creation workflow | How an author assembles a catalog version from entity definition versions — manual selection vs. automatic "snapshot current state" |
| Concurrent editing model | Optimistic locking, pessimistic locking, or merge-based conflict resolution |
| Ad-hoc query language | Whether complex cross-entity queries (beyond filter+sort) will be needed in the future |
| Predefined queries | Which standard queries are provided out of the box |
| Entity instance versioning depth | Whether full version history is retained or only N recent versions |
| Technology choices | Backend language/framework, UI framework, API style (REST vs. GraphQL) |
| Centralized hub topology | Hub-and-spoke deployment where consuming clusters sync CatalogVersion CRs from a central API. See Section 8.4. |
| ~~Landing page content~~ | **RESOLVED.** Minimal navigation cards (Option A). Single unified SPA with route-based views. See US-47. |

## 11. Technical Debt

Items where the current implementation diverges from the intended behavior described in this PRD. These should be addressed in priority order.

| ID | Item | Current Behavior | Required Behavior |
|----|------|-----------------|-------------------|
| TD-1 | Enum deletion safety | Enum delete checks if any attribute references it across all entity type versions (flat check) | Enum cannot be deleted if it is used by any attribute in a **used entity version**. A used entity version is defined as: (1) any entity type version pinned by a catalog version, or (2) the latest version of any entity type (which belongs to an implicit pre-production catalog). Unused historical versions that are not pinned by any CV and are not the latest version should not block deletion. |
| TD-2 | Catalog version timestamp uniqueness | Two catalog versions can have the same `created_at` timestamp, causing non-deterministic sort order | `created_at` must be unique across catalog versions. The backend should enforce this (e.g., retry with a small delay if a timestamp collision is detected). This ensures deterministic sort order in the CV list (`ORDER BY created_at DESC`). |
| TD-3 | Association target+role uniqueness | No uniqueness constraint on (target entity type, target role) per source entity type version | Target entity type + target role must be unique per source entity type version. Empty target role is valid (one allowed per target). API should reject duplicates with 409 Conflict. |
| ~~TD-4~~ | ~~Copy attributes dialog: enum name display~~ | **RESOLVED.** Copy picker now uses `enum_name` from snapshot to show "enum (MonthName)" instead of just "enum". |
| TD-5 | Version lineage tracking | Entity type versions are sequential integers with no parent tracking. Version 4 is created from version 3, but this relationship is not recorded. | Each entity type version should record which version it was derived from (`parent_version_id`). This enables: (1) understanding the edit history as a DAG rather than a flat list, (2) supporting future scenarios where editing from a catalog version context creates a branch, (3) detecting when two catalog versions diverge from a common base version. **Decision: deferred for v1.** The current simple incrementing scheme is sufficient for the initial release. Revisit when implementing edit-from-CV-context or version branching features (see FF-3). |
| TD-6 | Duplicate DTO mapping logic | Attribute and Association model-to-DTO conversion is duplicated across handlers (attribute_handler, association_handler, entity_type_handler VersionSnapshot) | Extract shared helper functions (e.g., `dto.ToAttributeResponse`, `dto.ToAssociationResponse`) to eliminate duplication. All handlers should use these helpers instead of inline conversion loops. |
| TD-7 | Bidirectional association removal only from source | A bidirectional association can only be removed from the entity type that created it (the source/outgoing side). From the target entity type's Associations tab, the Remove button is hidden for incoming associations, including bidirectional ones. | Since bidirectional associations are symmetric, the Remove button should be available from either side. Removing from the target side should delete the same association record. The UI currently hides Remove for all incoming associations — bidirectional should be an exception. |
| TD-8a | Extract shared EditAssociationModal component | Edit association modal is duplicated between `App.tsx` (diagram edit) and `EntityTypeDetailPage.tsx` (associations tab edit) — ~110 lines of duplication | Extract into shared `ui/src/components/EditAssociationModal.tsx` with props for `showEntityTypeNames`, `allowTypeChange`, `onSave`. |
| TD-8b | Consolidate edit modal state into a single object | Diagram edit modal in `App.tsx` uses 12 separate `useState` calls for one form | Group into a single state object or move into the shared component from TD-8a. |
| TD-8c | Extract diagram data loading into a custom hook | `App.tsx` and `CatalogVersionDetailPage.tsx` both have `loadDiagramData` functions that fetch snapshots and build `DiagramEntityType[]` | Extract into `ui/src/hooks/useDiagramData.ts` with `loadFromAllEntityTypes()` and `loadFromPins(pins)` methods. |
| ~~TD-8d~~ | ~~Extract EdgeClickData interface~~ | **RESOLVED.** Exported `EdgeClickData` interface from `EntityTypeDiagram.tsx`, imported in `App.tsx`. | |
| ~~TD-9~~ | ~~Show required attributes in diagram~~ | **RESOLVED.** Required attributes prefixed with `*` in diagram UML nodes. | |
| TD-10 | Mutable CVs in development mode | CV pins are immutable — entity types are pinned at creation and cannot be changed | In development stage, CVs should be mutable: add/remove entity type pins, change pinned versions. Pins are frozen only on promotion to testing. Catalogs cannot be created against a development-stage CV. If a CV with existing catalogs is demoted back to development, modified, and re-promoted, all catalogs pinned to that CV must be re-validated — they may become invalid if entity types were removed or attribute schemas changed. |
| TD-11 | Show mandatory associations in UI | Associations with cardinality `1` or `1..n` are not visually distinguished from optional ones | On the entity detail page, BOM modal, and diagram, show a mandatory indicator (e.g., `*` or bold) on the side of the association where cardinality starts with `1`. For example, mcp-tool's containment by mcp-server (cardinality `1` on source) shows as mandatory on the mcp-tool detail screen but NOT on the mcp-server detail screen (where the target cardinality `0..n` is optional). The indicator appears only from the perspective of the entity that is required to have the association. |
| TD-12 | Catalog re-pinning | A catalog's CV pin is immutable — to use a new CV, create a new catalog | Allow upgrading a catalog to a newer CV. Requires data migration validation: check that all entity instances are still valid under the new CV's schema. Report incompatibilities and let the user resolve them before completing the re-pin. |
| TD-13 | Get catalog version by name | CV can only be retrieved by ID; no lookup by version_label | Add `GET /api/meta/v1/catalog-versions/by-name/:label` endpoint for name-based lookup. The K8s CR already uses the label as its name. |
| TD-14 | Catalogs using this CV | CatalogVersion detail page does not show which catalogs are pinned to it | Add a "Catalogs" section on the CatalogVersion detail page listing catalogs pinned to that CV, with name, validation status, and link to catalog detail. |
| ~~TD-15~~ | ~~Catalog cascade delete needs transaction~~ | **RESOLVED.** Wrapped instance soft-delete + catalog hard-delete in `TransactionManager.RunInTransaction`. | |
| TD-16 | Mixed soft-delete/hard-delete on catalog deletion | Instances are soft-deleted (`deleted_at` set) but the catalog itself is hard-deleted. Soft-deleted instances with no parent catalog accumulate as dead rows. | Either hard-delete instances when the catalog is deleted (since the catalog is hard-deleted, there's no recovery path anyway), or soft-delete the catalog too. Also consider a periodic cleanup job for orphaned soft-deleted instances. |
| ~~TD-17~~ | ~~Catalog list pagination~~ | **RESOLVED.** Added `limit` and `offset` query params to `ListCatalogs` handler. Default 20, max 100. Instance list handler already had pagination. | |
| TD-18 | UI component props style inconsistency | Minor style issue: some components use a named `interface Props { ... }` for their parameter type (e.g., `EnumListPage`), while others use inline destructured types (e.g., `CatalogListPage`: `{ role }: { role: Role }`). Both are functionally identical. | Pick one convention and apply it consistently across all page components. The named `Props` interface is more common in the codebase and scales better when props grow. Low priority — a future style alignment pass. |
| TD-19 | N+1 query in resolveEntityType | `InstanceService.resolveEntityType` iterates all CV pins and calls `etvRepo.GetByID` for each to find the matching entity type | Replace the per-pin query loop with a batch fetch or a join query that resolves entity type ID → pinned version in one call. Acceptable for now since CVs typically have 3-5 pins; becomes a problem at 20+. |
| ~~TD-20~~ | ~~Missing name validation on instance creation~~ | **RESOLVED.** Added `strings.TrimSpace(name) == ""` validation in both `CreateInstance` and `CreateContainedInstance`. Returns `"instance name is required"` error. | |
| ~~TD-21~~ | ~~Remove catalog_version_id migration code~~ | **RESOLVED.** Migration code removed from `InitDB`. All environments have been migrated. | |
| ~~TD-23~~ | ~~CatalogDetailPage component too large~~ | **RESOLVED (Phases 1-3 complete).** Decomposed 3 page components (2834 lines total) into 6 hooks + 12 components (18 new files). CatalogDetailPage: 1208→740 lines. EntityTypeDetailPage: 1198→583 lines. OperationalCatalogDetailPage: 428→257 lines. 134 new tests, 92.9% UI coverage. Phase 4 (modal internalization) deferred. See `docs/plans/2026-03-24-component-decomposition.md`. |
| ~~TD-24~~ | ~~Remove legacy EntityInstanceService~~ | **RESOLVED.** Removed `entity_instance_service.go`, `entity_instance_service_test.go`, legacy `Handler` from `handler.go`, `handler_test.go`, `additional_tests_test.go`, `coverage_test.go`, and legacy route registration from `main.go`. | |
| ~~TD-25~~ | ~~Replace `interface{}` with `any` across codebase~~ | **RESOLVED.** Replaced in 9 files. | |
| TD-26 | Extract shared instance creation logic (M5) | `CreateInstance` and `CreateContainedInstance` share ~70% of logic (instance model creation, attribute validation, persistence, validation status reset, attribute resolution). | Extract a private `createInstanceInternal` method that both call, passing `parentID` (empty for root) and containment validation as optional steps. |
| TD-27 | ListContainedInstances pagination broken by in-memory filtering (M7) | `ListContainedInstances` fetches children with `ListByParent` (which respects limit/offset), then filters by entity type in memory. This can return fewer results than limit or miss results entirely when the parent has children of multiple types. | Push the entity type filter into the repository query (add `ListByParentAndType` method), or fetch all children without pagination and paginate after filtering. |
| ~~TD-29~~ | ~~Reject reserved entity type names~~ | **RESOLVED.** Added `reservedEntityTypeNames` blocklist (`links`, `references`, `referenced-by`, `copy`, `replace`, `tree`, `validate`, `publish`, `unpublish`) checked in both `CreateEntityType` and `RenameEntityType`. | |
| ~~TD-30~~ | ~~Add catalog ownership check on instance read/update/delete~~ | **RESOLVED.** Added `inst.CatalogID != catalog.ID` check in `GetInstance`, `UpdateInstance`, and `DeleteInstance`. Returns NotFound if mismatch. | |
| TD-31 | Create new container from contained instance's Set Container modal | The Set Container modal only allows selecting existing parent instances. Users cannot create a new container directly from the child side — they must first create the container via the parent entity type tab, then come back and set it. | Add a "Create New" mode to the Set Container modal (similar to the "Create New / Adopt Existing" toggle in the Add Contained Instance modal). The natural flow is parent-first, so this is a convenience feature, not a critical gap. |
| TD-32 | Diagram: overlapping edges between same entity pair | When two or more associations exist between the same pair of entity types (e.g., mcp-tool → guardrail with both "uses" and "validates"), the edges overlap into a single line with two labels stacked on top of each other. | Add edge offset or curvature so multiple edges between the same pair are visually distinct. Dagre layout doesn't natively support parallel edges — options include: (a) adding a small vertical offset per duplicate edge, (b) using quadratic bezier curves with different control points, or (c) bundling labels into a single edge with a multi-line label. |
| TD-33 | "Contained by" flickers UUID before showing parent name | When opening instance details for a contained entity, the parent UUID briefly flashes before the async API call resolves the parent name. | Either (a) include `parent_instance_name` in the instance list API response so no extra fetch is needed, or (b) show a spinner/placeholder instead of the raw UUID while loading. Option (a) is cleaner — resolve the parent name server-side in `ListInstances`/`GetInstance`. |
| ~~TD-34~~ | ~~`SetParentRequest.ParentType` missing validation~~ | **RESOLVED.** Added explicit `parent_type is required` check in `SetParent` handler. Returns 400 with clear error message. | |
| ~~TD-35~~ | ~~Operational catalog detail page too large~~ | **RESOLVED.** Extracted `useContainmentTree` hook + `InstanceDetailPanel` component. 428→257 lines, 15→5 useState. See TD-23 resolution. |
| TD-37 | Reference direction unclear in tree browser detail panel | In the instance detail panel, directional associations show under "Forward References" and "Referenced By" sections with a "Type" column showing "directional". It is not clear which direction the association goes — the user cannot tell whether the selected instance depends on the target or vice versa. The association name alone may not convey direction (e.g., "uses-model" is clear, but "related-to" is not). | Show an arrow or directional indicator in the reference table: e.g., "my-server → gpt-4" for forward refs and "monitor-1 → my-server" for reverse. Alternatively, use role labels from the association definition (source_role/target_role) to clarify the relationship semantics. Consider replacing the generic "directional" type label with the actual role or a "depends on" / "depended by" phrasing. |
| TD-38 | Entity type tab selector doesn't scale in meta catalog detail | The meta UI `CatalogDetailPage` uses PatternFly Tabs with one tab per entity type. When a catalog has many entity types (10+), the tabs overflow a single row and become hard to navigate. | Options: (A) Replace tabs with a sidebar or dropdown selector. (B) Add a search/filter input above the tabs. (C) Use PatternFly's scrollable tabs variant (`isOverflowHorizontal`). (D) Switch to a two-pane layout similar to the operational UI's tree browser. |
| ~~TD-36~~ | ~~Review usefulness of Overview tab in operational catalog view~~ | **RESOLVED.** Overview tab removed (TD-56). Tree Browser is now the default tab. See TD-56 for future re-addition with useful content. |
| TD-28 | Phase 3 code quality improvements (L1-L5, L7) | Multiple low-severity issues from quality review: (L1) duplicated forward/reverse reference handler conversion logic, (L2) dead `_ = parentInst`/`_ = sourceInst` assignments, (L3) JSON tags on service-layer `ReferenceDetail`, (L4) N+1 queries in `resolveLinks`, (L5) CatalogDetailPage now has ~30 state variables and should be decomposed, (L7) silently swallowed `UpdateValidationStatus` errors. | Extract `refsToDTO` helper in handler. Clean up dead assignments. Remove JSON tags from service types. Add batch fetch for links resolution. Decompose CatalogDetailPage into sub-components. Log validation status update failures. |
| TD-39 | CopyCatalog sequential instance creation doesn't scale | `CopyCatalog` creates instances one at a time via N individual `instRepo.Create` calls, plus N `GetCurrentValues` and N `GetForwardRefs` calls. For catalogs with 1000+ instances this is slow. | Add `CreateBatch` method to `EntityInstanceRepository` that inserts multiple instances in a single query. Similarly batch `SetValues` and link creation. Low priority — catalogs currently have <100 instances. |
| TD-40 | `SyncCR` uses unstructured logging | `SyncCR` in `catalog_service.go` uses `log.Printf("warning: ...")` (Go's default logger). In production this produces unstructured text logs that are hard to filter and correlate in centralized logging systems. | Replace `log.Printf` with structured logging (e.g., `slog.Warn` or a project-standard logger) that includes catalog name, error, and operation context as structured fields. Apply the same fix to any other `log.Printf` calls in the codebase. |
| ~~TD-41~~ | ~~Show entity description in table views~~ | **PARTIALLY RESOLVED.** Operational catalog list shows Description column (catalog has its own description). Entity type list and BOM pins table have the UI column but the API does not return description — see TD-43 and TD-44. |
| TD-43 | Entity type list missing description in API response | `EntityTypeResponse` DTO does not include a `description` field. The `EntityType` model has no description — it lives on `EntityTypeVersion`. The entity type list page shows an empty Description column. | Add `description` to `EntityTypeResponse` by resolving the latest version's description in the handler (or service). Alternatively, add a `description` field to the `EntityType` model itself (separate from version description). The version description describes the version change; the entity type description should describe what the entity type represents. |
| ~~TD-44~~ | ~~BOM pins table missing description in API response~~ | **RESOLVED.** Added `Description` to `ResolvedPin`, `CatalogVersionPinResponse`, and handler. BOM tab now shows entity type version description. |
| TD-45 | Enum list page missing description column | The enum list page does not show a Description column. Enums have a `name` but no description field in the model. | Either add a `description` field to the Enum model, or accept that enums don't have descriptions. If adding, update the create/edit API and UI. |
| TD-46 | No UI to edit entity type version description | The entity type version description is set at creation and carried forward on COW. The backend `UpdateEntityType` endpoint (`PUT /entity-types/:id`) creates a new version with a new description, but the UI has no control to invoke it. Users cannot change the description after initial creation. | Add a "Description" editable field (inline or modal) on the entity type detail page. Editing the description calls `PUT /entity-types/:id` with `{"description": "new desc"}`, which creates a new version via COW. Show the current version's description on the detail page header area. |
| ~~TD-47~~ | ~~Diagram: containment edges should use UML composition notation~~ | **RESOLVED.** Filled diamond SVG marker added on the parent (source) end of containment edges. Arrowhead retained on target end. Non-containment edges unchanged. Implemented in `EntityTypeDiagram.tsx` `AssociationEdge` component. |
| ~~TD-48~~ | ~~Duplicate number-parsing logic in attribute submission~~ | **RESOLVED.** Extracted `buildTypedAttrs` utility in `utils/buildTypedAttrs.ts`, used by all three call sites. |
| TD-49 | `useInstanceDetail.selectInstance` missing `setAuthRole` call | `useCatalogData` and `useInstances` both call `setAuthRole(role)` before API requests, but `useInstanceDetail.selectInstance` makes API calls (get parent, list children, get refs) without setting the auth role. If the role changes while an instance is selected, the wrong role could be sent. | Pass `role` to `useInstanceDetail` and call `setAuthRole(role)` at the start of `selectInstance`. |
| TD-50 | `selectInstance` passes stale instance object after mutations | After mutations (handleAddChild, handleCreateLink, handleSetParent), the code calls `detail.selectInstance(detail.selectedInstance)` with the render-time snapshot. The instance's version may have changed server-side. | Pass only the instance ID and re-fetch the instance inside `selectInstance`, or accept that only `inst.id` is used (current behavior is safe but fragile). |
| TD-51 | `onRemoveParent` swallows errors silently | The "Remove Container" inline handler uses `.catch(() => {})` which silently discards API errors. The user gets no feedback if the operation fails. | Show an error alert or set `setParentError` in the catch block. |
| TD-52 | Modal data-loading still managed by page | `AddChildModal`, `LinkModal`, and `SetParentModal` receive dependent data (child schema attrs, enum values, available instances, link target instances, parent instances) as props from the page. The page manages ~60 lines of data-loading orchestration for these modals. | Move data loading into the modals — each modal imports the API client and loads its own data on open/selection change. This eliminates tight coupling and reduces the page by ~60 lines. Modals would accept `catalogName`, `pins`, and `schemaAssocs` as props, then load everything else internally. |
| TD-53 | Diagram tab JSX duplicated across catalog pages | The Model Diagram tab rendering (loading spinner, error alert, empty state, diagram component) is duplicated between `CatalogDetailPage.tsx` and `OperationalCatalogDetailPage.tsx` (~10 lines each). | Extract a `DiagramTabContent` component that accepts the `useCatalogDiagram` hook return value and renders the loading/error/empty/diagram states. Both pages import and use this component. |
| TD-54 | `CatalogVersionDetailPage` does not use `useCatalogDiagram` hook | `CatalogVersionDetailPage.tsx` manually manages `diagramData` and `diagramLoading` state with ~25 lines of inline fetch-pins-then-snapshots logic. This duplicates the exact pattern now encapsulated in `useCatalogDiagram`. | Refactor `CatalogVersionDetailPage` to use `useCatalogDiagram(id)` and remove the duplicated state and logic. |
| TD-55 | Edge click handler object construction duplicated in `AssociationEdge` | The `onClick` handler in `AssociationEdge` (EntityTypeDiagram.tsx) constructs an identical `EdgeClickData` object at two locations (transparent clickable path and label group). Each has ~10 fields with `|| ''` fallbacks. | Extract a helper function (e.g., `buildEdgeClickData(data)`) within the component scope that both click handlers call. |
| TD-56 | Operational catalog viewer Overview tab removed — consider re-adding with useful content | The Overview tab on `OperationalCatalogDetailPage` was hidden because it showed only entity type names/versions with no actionable information. If a meaningful Overview tab is designed later (e.g., catalog stats, instance counts per entity type, last-modified timestamps, data quality summary), the tab should be re-added with useful content. | Design and implement a useful Overview tab, or confirm it is not needed and clean up any remaining dead code. |
| TD-57 | Move `CatalogDetailPage` and `CatalogListPage` from `pages/operational/` to `pages/meta/` | Both `CatalogDetailPage.tsx` (schema catalog detail with instance CRUD) and `CatalogListPage.tsx` (schema catalog list) live in `pages/operational/` but serve schema-management functions and are rendered under `/schema/*` routes. All other schema pages (EntityTypeDetailPage, EnumDetailPage, CatalogVersionDetailPage) are in `pages/meta/`. | Move both files and their test files to `pages/meta/`. Update imports in `App.tsx`. This is a pure file-move refactor with no behavior changes. |
| TD-59 | N+1 query in entity type list description resolution | The entity type list handler calls `GetLatestByEntityType()` once per entity type in a loop to resolve descriptions. With many entity types, this generates N additional database queries. | Add a batch method `GetLatestByEntityTypes(ctx, entityTypeIDs []string) (map[string]*EntityTypeVersion, error)` to resolve all descriptions in a single query. |
| TD-60 | Enum description edit uses `window.prompt()` instead of inline edit | The enum detail page description edit uses `window.prompt()`, while the entity type detail page uses a proper inline TextInput with Save/Cancel buttons. This is inconsistent UX and makes the enum edit untestable in browser tests (Playwright cannot mock native `window.prompt`). | Replace `window.prompt()` with inline TextInput edit (same pattern as EntityTypeDetailPage description edit). This also makes the catch block coverable in tests. |
| TD-61 | CatalogVersion description is not editable after creation | The CV description is set at creation but there is no `PUT /catalog-versions/:id` endpoint to update it. The CV detail page shows the description but has no edit control. | Add `UpdateCatalogVersion(ctx, id, description)` to service, `PUT /catalog-versions/:id` handler accepting `{description}`, and inline edit on the CV detail page (same pattern as EntityTypeDetailPage). |
| TD-58 | Enum values are not versioned — mutations are destructive | Enum values (add, remove, reorder) mutate in place. If an enum is referenced by attributes across multiple catalog versions, changing its values affects all of them retroactively. Entity types use copy-on-write versioning for attributes and associations, but enums have no equivalent mechanism. Removing an enum value that is in use by existing entity instances could leave invalid data. | Options: (A) Add versioning to enums (copy-on-write on mutation, enum versions pinned in CVs alongside entity type versions). (B) Add validation that prevents removing enum values that are in use. (C) Accept the current behavior and document it as a known limitation. Option B is the minimum viable fix. |
| ~~TD-42~~ | ~~[IMPORTANT] Add Contained Instance modal missing custom attributes~~ | **RESOLVED.** Add Contained Instance modal now loads child entity type schema attributes when child type is selected. Renders attribute fields (string, number, enum) same as root-level Create Instance modal. Attributes passed in API request body. |
| TD-62 | [IMPORTANT] Audit all update/PUT endpoints for omitted-field data loss | The `UpdateEnum` handler was silently erasing the `description` field when the caller omitted it from the JSON body, because the DTO used `string` (zero value `""`) rather than `*string` (nil = omitted). This was fixed for enums by switching to `*string` and preserving the current value when nil. **The same bug pattern may exist on other update endpoints** — any PUT/PATCH handler that unconditionally overwrites a field from a DTO `string` will erase data when the caller omits that field. | Audit every update endpoint in the Meta and Operational APIs. For each optional field on an update DTO, verify that omitting the field in the JSON request body does NOT erase the current value. Use `*string` (pointer) for optional fields and preserve current values when nil. Endpoints to check: `UpdateEntityType`, `UpdateInstance`, `SetParent`, catalog update (if added), CV update (TD-61), and any future update handlers. |
| ~~TD-22~~ | ~~[CRITICAL] Common attributes as schema-level attributes~~ | **RESOLVED (Approach B — API-level merge).** Common attributes — Name (required) and Description (optional) — are injected as synthetic system attributes (`system: true`) into all API responses: instance detail, attribute lists, version snapshots, and UML diagrams. The UI renders them uniformly alongside custom attributes. System attributes cannot be created, edited, renamed, or deleted by users. Custom attribute names "name"/"description" are rejected. Copy-attributes excludes system attributes. Catalog validation checks Name is non-empty. No DB schema changes — common attributes remain as fields on `EntityInstance`, with the API layer handling the merge. |

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

Catalog versions are currently immutable after creation — pins (entity type version selections) cannot be changed, and there is no way to add or remove entity types from an existing CV. This forces users to delete and recreate CVs when entity types evolve, which is disruptive when catalogs already reference the CV.

**Capabilities:**
- **Add entity type pin** — pin a new entity type (at a specific version) to an existing CV
- **Remove entity type pin** — unpin an entity type from the CV
- **Change pinned version** — upgrade or downgrade the pinned version for an entity type (e.g., V2 → V3)

**Editing constraints based on CV usage:**

| CV Usage | Allowed Changes |
|----------|----------------|
| **Unused** (no catalogs reference it) | All changes — add, remove, change pins freely |
| **Used by draft-only catalogs** (all catalogs are unpublished with status `draft` or `invalid`) | All changes with warnings — removing an entity type that has instances in a catalog resets that catalog's validation status to `draft`. The UI warns but allows the operation. |
| **Used by published catalogs** (at least one catalog is published) | **Non-destructive changes only:** (1) Add new entity type pins (new types have no instances yet, so no data impact). (2) Change pinned version to a newer version IF the new version is backward-compatible — meaning it only adds optional attributes, adds optional associations (`0..n` cardinality), or changes descriptions. Renaming attributes, removing attributes, changing types, adding required attributes/associations, or removing entity type pins are **blocked** with a clear error explaining why. |

**Backward-compatible version change detection:** When upgrading a pin from V_old to V_new, the system computes a version diff (same logic as the existing version comparison feature) and classifies each change as compatible or breaking:
- **Compatible:** added optional attribute, added optional association (`0..n`), changed attribute/association description, changed entity type description
- **Breaking:** removed attribute, renamed attribute, changed attribute type, added required attribute, added mandatory association (`1` or `1..n` cardinality), removed association, changed association type or target

**API:**
- `PUT /api/meta/v1/catalog-versions/:id/pins` — replace the full pin set (validates constraints before applying)
- `POST /api/meta/v1/catalog-versions/:id/pins` — add a single pin
- `DELETE /api/meta/v1/catalog-versions/:id/pins/:entity-type-id` — remove a pin
- `PUT /api/meta/v1/catalog-versions/:id/pins/:entity-type-id` — change pinned version for one entity type

**UI:**
- CV detail page gains an "Edit Pins" mode with add/remove/upgrade controls
- When upgrading a pin version, a diff preview shows what changed (reuses existing version diff component)
- Breaking changes on published CVs show a blocked indicator with explanation
- Removing a pin that has catalog instances shows a warning with affected catalog names and instance counts

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

### FF-7: Catalog Versioning (Snapshots)

Catalogs currently have a single mutable set of instances with a validation status (`draft`/`valid`/`invalid`). Any data mutation resets the status to `draft`. This creates a problem for published catalogs: adding a new instance to a `valid` catalog resets it to `draft`, which could cause it to be unpublished (once Catalog CRs are implemented in Phase 7).

**Proposed solution:** Catalog snapshots — immutable, versioned copies of a catalog's data at a point in time. Similar to how CatalogVersions snapshot entity type schemas, catalog snapshots would freeze instance data.

**Workflow:**
1. A catalog has a working copy (mutable, always `draft` after changes) and zero or more published snapshots.
2. When validation passes, the user can "publish" the current state as a numbered snapshot (V1, V2, ...).
3. Published snapshots are immutable and retain their `valid` status.
4. The Catalog CR references the latest published snapshot, not the mutable working copy.
5. Users continue editing the working copy. The published snapshot is unaffected until a new snapshot is explicitly published.

**Benefits:**
- Published catalogs are never disrupted by ongoing edits
- Rollback to a previous snapshot is possible
- Consumers always see a consistent, validated dataset
- Diff between snapshots enables change tracking

**Considerations:**
- Storage cost: each snapshot duplicates instance data (or uses COW/reference counting)
- Migration: existing catalogs would need an initial V1 snapshot created from their current state
- UI: snapshot list, publish action, rollback, diff view

**Decision:** Deferred. Phase 7 will use a simpler approach (Option A: `draft` does not unpublish — the Catalog CR represents the last validated state). Catalog versioning can be added when the need for immutable published snapshots is validated with users.

### FF-8: Copy & Replace Catalog — IMPLEMENTED

See US-44 (Copy Catalog), US-45 (Replace Catalog), US-46 (Copy & Replace UI) in section 9.

### FF-9: Multi-Namespace Catalog Publishing

In Phase 7, Catalog CRs are created in the AssetHub's own namespace. This means all published catalogs are visible to any application with access to that namespace. FF-9 extends publishing to support target namespaces, so catalogs are discoverable only by apps in specific namespaces.

**Proposed change:** The `publish` API accepts an optional `namespace` parameter. When specified, the Catalog CR is created in that namespace instead of the AssetHub's namespace. A catalog can be published to multiple namespaces simultaneously.

**API:**
- `POST /api/data/v1/catalogs/{name}/publish` — `{namespace?: string}` (default: AssetHub namespace)
- `POST /api/data/v1/catalogs/{name}/unpublish` — `{namespace?: string}` (unpublish from specific namespace, or all if omitted)
- `GET /api/data/v1/catalogs/{name}/publications` — list namespaces where the catalog is published

**Backend changes:**
- `CatalogPublication` table: `(catalog_id, namespace, published_at)` — tracks where each catalog is published
- Operator needs ClusterRole to create/delete CRs in arbitrary namespaces
- Owner references cannot cross namespaces — use finalizers on the AssetHub CR or a controller-based cleanup for GC
- Namespace must exist; publishing to a non-existent namespace returns 404

**Benefits:**
- K8s RBAC "just works" — apps see only Catalog CRs in their own namespace
- Multi-audience: one catalog published to many namespaces for different teams
- Clean tenant isolation without custom cross-namespace authorization

**Decision:** Deferred. Phase 7 publishes to the AssetHub's namespace only. Multi-namespace publishing can be added when multi-tenancy requirements are validated.

### FF-10: Edit Catalog Metadata

Allow editing a catalog's name and description after creation. Currently catalogs are immutable once created — the only way to change metadata is to delete and recreate.

**API:**
- `PUT /api/data/v1/catalogs/{catalog-name}` — `{name?, description?}` → 200 with updated catalog
- If `name` is provided and differs from current, the catalog is renamed (same DNS-label validation as create)

**Access control:**
- **Unpublished catalogs:** RW+ can edit both name and description
- **Published catalogs:** Only SuperAdmin can edit, and only the description — renaming a published catalog is not allowed (returns 400). The Catalog CR name is immutable while published.
- **Workaround for renaming published catalogs:** Unpublish → rename → republish. This ensures consumers watching the CR name are not surprised by a silent rename.

**UI:**
- "Edit" button on catalog detail page (RW+ for unpublished, SuperAdmin for published)
- Modal with name and description fields, pre-filled with current values
- Name field disabled on published catalogs with tooltip: "Unpublish to rename"
- Name field validates DNS-label format

**Note:** Changing the pinned catalog version (re-pinning to a different CV) is a separate concern tracked in TD-12.

### FF-11: Catalog Import from External Systems

Import catalog data (entity instances, attribute values, association links, containment hierarchy) from external sources into a new or existing catalog. Supports both file-based and API-based import.

**Import sources:**
- **File import:** Upload YAML or JSON files containing instance data structured according to the catalog's pinned CV schema. The UI provides a file upload dialog; the backend validates the data against the CV's entity type definitions before creating instances.
- **API import:** Pull data from external systems via configured API endpoints. The import configuration specifies the source URL, authentication, and a mapping from the external data format to the Asset Hub entity model.

**Scope:** Details on file format, API mapping configuration, conflict resolution (merge vs. overwrite), and validation behavior will be discussed and specified in a future design session.

