# AI Asset Hub User Guide

This guide walks you through the AI Asset Hub user interface, covering Schema Management (defining what the system manages) and the Data Viewer (browsing catalog data).

---

## 1. Getting Started

1. Open your browser and navigate to `http://localhost:30000`.
2. The **Landing Page** shows links to Schema Management and your available Catalogs.
3. Use the **Role** dropdown in the top-right corner to select your role: RO, RW, Admin, or SuperAdmin. Your role determines which actions are available.

> In production OpenShift deployments, your role is determined by cluster RBAC permissions rather than the dropdown.

---

## 2. Schema Management

Access Schema Management by clicking **Explore Schema** on the landing page. It contains five tabs: Entity Types, Catalog Versions, Catalogs, Types, and Model Diagram.

### 2.1 Entity Types

Entity types define the kinds of assets the system manages (for example, "Model", "MCP Server", "Tool").

**Creating an entity type** (Admin+ only):
1. Click **Create Entity Type**.
2. Enter a name and optional description, then click **Create**.
3. You are taken to the detail page to add attributes and associations.

Click any entity type name to open its detail page, where you can update the description, manage attributes and associations, and review version history. Each change automatically creates a new version -- previous versions remain intact and accessible.

### 2.2 Type Definitions

Type definitions specify data types for attributes. Built-in system types (string, integer, number, boolean, date, url) are always available. Custom types add constraints.

**Creating a custom type:**
1. Go to the **Types** tab, click **Create Type Definition**.
2. Enter a name, select a base type, and configure constraints:
   - **string**: max length, multiline toggle, regex pattern
   - **integer/number**: min and max values
   - **enum**: ordered list of allowed values
   - **list**: element type and max length
3. Click **Create**. Editing constraints later creates a new version automatically.

### 2.3 Attributes

Attributes define data fields on an entity type. Every entity type includes system attributes (Name, Description) that cannot be removed.

**Adding an attribute:**
1. On the entity type detail page, click **Add Attribute**.
2. Enter a name (unique within the entity type; "name" and "description" are reserved).
3. Select a type definition and optionally mark it as **Required**.
4. Click **Add**.

Custom attributes can be reordered using move up/down controls. You can also copy attributes from another entity type to avoid redefining common fields.

### 2.4 Associations

Associations define relationships between entity types:

- **Containment**: parent-child (deleting the parent cascades to children)
- **Directional reference**: one-way link (source to target)
- **Bidirectional reference**: mutual link navigable from either side

**Adding an association:**
1. Click **Add Association** on the entity type detail page.
2. Enter a name, select the target entity type, choose the type, and set cardinality (for example, `1` to `0..n`).
3. Click **Add**. Containment cycles are detected and blocked automatically.

### 2.5 Model Diagram

The **Model Diagram** tab shows a visual graph of all entity types (nodes) and their associations (edges). Double-click a node to open that entity type. Click an edge to view or edit the association.

---

## 3. Catalog Versions

A catalog version pins specific entity type versions together as a "bill of materials."

**Creating a catalog version:**
1. Go to the **Catalog Versions** tab, click **Create Catalog Version**.
2. Enter a version label (e.g., "v1.0") and optional description.
3. Select entity types to include from the containment tree. Selecting a parent auto-selects its children and vice versa.
4. Choose which version to pin for each entity type (latest is pre-selected).
5. Click **Create**. It starts in the **Development** stage.

**Lifecycle stages** progress with role-based gates:
- **Development** (blue): active editing, no cluster resources created.
- **Testing** (orange): a CatalogVersion custom resource is created for discovery. Promoted by RW+.
- **Production** (green): frozen for consumers. Promoted by Admin+. Demoting requires SuperAdmin.

Use the **Promote** and **Demote** buttons to move between stages.

---

## 4. Catalogs

A catalog is a named collection of entity instances using a catalog version as its schema.

**Creating a catalog:**
1. Go to the **Catalogs** tab, click **Create Catalog**.
2. Enter a DNS-compatible name (lowercase, hyphens, max 63 characters), select a catalog version, and click **Create**.

**Working with instances:** On the catalog detail page, create instances for each pinned entity type. Instances support draft mode -- required attributes and mandatory associations can be filled in incrementally across sessions. For containment associations, create child instances within a parent to build the hierarchy.

**Publishing:** Click **Validate** to check all instances against the schema. Only **valid** catalogs can be published. Publishing creates a Kubernetes custom resource for discovery and write-protects the catalog.

---

## 5. Data Viewer

Access the Data Viewer by clicking a catalog name on the landing page. It provides a read-only browsing experience showing:

- A **containment tree** on the left with instances organized by entity type
- **Instance details** on the right with attribute values and references
- **Filtering** to search instances by attribute values

---

## 6. Copy and Replace

To update a published catalog safely:
1. **Copy** the published catalog to create a staging copy (all data is duplicated).
2. Make changes in the staging copy.
3. **Validate** the staging copy.
4. **Replace** to atomically swap the staging copy into the original's name. The original is renamed as a backup.

---

## 7. Validation

Validation checks all instances in a catalog against its schema. Common violations and fixes:

| Check | How to Fix |
|-------|-----------|
| Required attribute missing | Open the instance and provide a value |
| Invalid attribute value | Edit the value to match the type's constraints |
| Mandatory association missing | Create the required link between instances |
| Containment inconsistency | Assign the instance to a parent or remove it |
| Unpinned entity type | Pin the entity type in the catalog version or remove orphaned instances |

Results list each violation with the affected instance and entity type. Re-validate at any time to confirm all issues are resolved.
