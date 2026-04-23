# Catalog Import/Export — Design Specification

## Overview

Internal import/export for system portability: export a catalog (schema + data) to a JSON file, import it into another Asset Hub deployment. Designed with future external export in mind (FF-15) but Phase 1 is internal-only.

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| File format | JSON (Phase 1) | Native to API, avoids YAML type coercion pitfalls. YAML may be added later — documented in PRD as future option. |
| References in file | By name (not ID) | Human-readable, survives cross-system import, works with rename logic. |
| Instance structure | Nested by containment association name | Avoids name collisions for contained instances. Association name as key eliminates redundant entity_type per child. Supports multiple containments of same type. |
| Pins on import | Always V1 | Fresh start — no version history imported. |
| CV lifecycle on import | Always development | Target system hasn't tested/promoted this CV. |
| Validation status on import | Always draft | Target system should re-validate. |
| Transaction model | Two transactions | T1: schema (type defs, entity types, CV, pins). T2: data (catalog, instances, IAVs, links). Schema survives data failure — re-import auto-detects via identical+V1 check. |
| Collision detection | Structural identity comparison | Field-by-field per type (not JSON string comparison). System types resolved by name. |
| Mass rename | Prefix/suffix on all entity types + type definitions | Applied pre-dry-run, not just for conflicts. Useful for backup imports. |
| Dry-run pattern | Same endpoint with ?dry_run=true | Consistent with UpdatePin pattern. File re-sent on second call (stateless). |
| Export entity selection | Default "export all" + optional tree picker with dependency auto-selection | Common case is full export. Selective export is power feature. Future: graph view option. |
| Partial export links | Warning + user decision | Links to excluded entity types are dropped with explicit warning. User can go back to include missing entities. |
| "Reuse existing" | Entity types and type definitions only | Instances are never reused — always created fresh for the imported catalog. Instances are namespaced by catalog. |
| Identical + V1 detection | Auto-suggest "reuse all" | When all matching schema entities are identical AND at V1, likely a re-import after partial failure. |
| Attribute values for list/json | Parsed JSON in export file | Importer re-serializes based on type definition base type. Readable file, good for future external export. |
| Import dependency order | File structure defines order | type_definitions → entity_types → instances. Import processes top to bottom. Validate references early. |
| Type definition pins | Auto-pinning | Rely on existing auto-pin logic when entity type pins are created. No separate type_pins section in file. |
| Type def "identical" comparison | Field-by-field per base type | Compare known constraint fields (max_length, pattern, values, etc.) individually. Avoids JSON normalization issues. |
| Source system field | Free-form string, user-editable | Default: AssetHub CR name or empty. Purely informational. |
| Access control | Admin+ for both (provisional) | Documented in PRD that roles may diverge for import vs export, internal vs external, as use cases mature. |

## Export

### API

`GET /api/data/v1/catalogs/{name}/export`

Query parameters:
- `entities` (optional): comma-separated entity type names. Default: all pinned entity types.
- `source_system` (optional): override the source system label.

Response: JSON file with `Content-Disposition: attachment; filename="{catalog-name}-export.json"`. Browser Save As dialog with default filename.

### Export Flow

1. User clicks "Export" on catalog detail page
2. Modal opens with two modes:
   - **Export all** (default): one-click download
   - **Select entities**: tree view of pinned entity types with checkboxes. Selecting one auto-checks dependencies (containment + reference targets, recursively). User can uncheck.
3. If catalog is `draft`/`invalid`: warning banner — "This catalog has not been validated. Exported data may be incomplete. Export anyway / Cancel"
4. If partial export drops links: warning — "N links reference entity types not included in this export (type1, type2). These links will be dropped. Go back to include these entity types, or continue without the links."
5. Download triggers browser Save As dialog

### File Format (v1.0)

```json
{
  "format_version": "1.0",
  "exported_at": "2026-04-23T14:30:00Z",
  "source_system": "assethub-dev",

  "catalog": {
    "name": "production-agents",
    "description": "Production MCP agent configurations",
    "validation_status": "valid"
  },

  "catalog_version": {
    "label": "v2.0",
    "description": "Spring 2026 release"
  },

  "type_definitions": [
    {
      "name": "hex12",
      "description": "12-char hex ID",
      "base_type": "string",
      "system": false,
      "constraints": { "max_length": 12, "pattern": "[0-9A-F]*" }
    },
    {
      "name": "tool-type",
      "description": "Tool classification",
      "base_type": "enum",
      "system": false,
      "constraints": { "values": ["read", "write", "admin"] }
    }
  ],

  "entity_types": [
    {
      "name": "mcp-server",
      "description": "An MCP server endpoint",
      "attributes": [
        { "name": "endpoint", "type_definition": "url", "required": true, "ordinal": 0, "description": "Server URL" },
        { "name": "containerized", "type_definition": "boolean", "required": false, "ordinal": 1, "description": "" }
      ],
      "associations": [
        {
          "name": "tools",
          "type": "containment",
          "target": "mcp-tool",
          "source_cardinality": "1",
          "target_cardinality": "0..n",
          "source_role": "",
          "target_role": ""
        },
        {
          "name": "pre-execute",
          "type": "directional",
          "target": "guardrail",
          "source_cardinality": "0..n",
          "target_cardinality": "0..n",
          "source_role": "",
          "target_role": ""
        }
      ]
    },
    {
      "name": "mcp-tool",
      "description": "A tool provided by an MCP server",
      "attributes": [
        { "name": "type", "type_definition": "tool-type", "required": false, "ordinal": 0, "description": "" },
        { "name": "idempotent", "type_definition": "boolean", "required": false, "ordinal": 1, "description": "" }
      ],
      "associations": []
    },
    {
      "name": "guardrail",
      "description": "A safety guardrail",
      "attributes": [
        { "name": "id", "type_definition": "hex12", "required": true, "ordinal": 0, "description": "" }
      ],
      "associations": []
    }
  ],

  "instances": [
    {
      "entity_type": "mcp-server",
      "name": "github",
      "description": "GitHub MCP server",
      "attributes": {
        "endpoint": "https://github.example.com/mcp",
        "containerized": "true"
      },
      "links": [
        { "association": "pre-execute", "target_type": "guardrail", "target_name": "pii-filter" }
      ],
      "tools": [
        {
          "name": "list-repos",
          "description": "List repositories",
          "attributes": { "type": "read", "idempotent": "true" },
          "links": []
        },
        {
          "name": "create-pr",
          "description": "Create pull request",
          "attributes": { "type": "write", "idempotent": "false" },
          "links": []
        }
      ]
    },
    {
      "entity_type": "guardrail",
      "name": "pii-filter",
      "description": "Filters PII from responses",
      "attributes": { "id": "A1B2C3D4E5F6" },
      "links": []
    }
  ]
}
```

**Conventions:**
- `type_definition` in attributes: references by name. System types (`url`, `boolean`, `string`, etc.) resolved to target system's built-ins on import — not included in `type_definitions` section.
- `target` in associations: entity type name.
- Attribute values: always strings for scalar types. List and JSON values are parsed (not escaped strings) — importer re-serializes based on base type.
- Contained instances: nested under association name keys (e.g., `"tools": [...]`). No `entity_type` per child — determined by association definition. Recursive for deeper nesting.
- `links`: `target_path` omitted for root-level targets, present as ancestry array for contained targets (e.g., `"target_path": ["github"]` for a tool inside github server).
- `children` and `links` can be omitted entirely (treated as empty).
- System type definitions not included in `type_definitions` — only custom types.
- `format_version` "1.0" — future versions add fields; importers ignore unknown fields.
- `source_system`: free-form, user-editable, default from AssetHub CR name. Informational only.

## Import

### API

`POST /api/data/v1/catalogs/import`

Request body (JSON):
```json
{
  "catalog_name": "my-copy",
  "catalog_version_label": "v1-imported",
  "rename_map": {
    "entity_types": { "mcp-server": "imported-mcp-server" },
    "type_definitions": { "hex12": "imported-hex12" }
  },
  "reuse_existing": ["guardrail"],
  "data": { "format_version": "1.0", ... }
}
```

All fields except `data` are optional. `catalog_name` and `catalog_version_label` override values from the file. `rename_map` applies individual renames. `reuse_existing` lists entity type names to reuse instead of creating — the CV pins the **latest version** of the reused entity type on the target system. Associated type definitions are also reused (resolved by name from the reused entity type's attributes).

Query parameter: `?dry_run=true` returns collision report without importing.

### Dry-Run Response

```json
{
  "status": "conflicts_found",
  "collisions": [
    { "type": "entity_type", "name": "mcp-server", "resolution": "conflict", "detail": "exists with different attributes" },
    { "type": "entity_type", "name": "guardrail", "resolution": "identical", "version": 1, "detail": "structurally identical at V1, likely previous import" },
    { "type": "type_definition", "name": "hex12", "resolution": "conflict", "detail": "different constraints" },
    { "type": "catalog", "name": "production-agents", "resolution": "conflict", "detail": "catalog name already exists" }
  ],
  "summary": {
    "total_entities": 5,
    "conflicts": 2,
    "identical": 1,
    "new": 2
  }
}
```

When `status` is `"ready"`, no collisions found — import can proceed directly.

### Import Flow

1. Parse and validate the export file (format version, required fields, reference integrity)
2. Apply rename_map to entity type and type definition names throughout the file
3. Resolve system type definitions by name to target system's built-ins
4. **Transaction 1 — Schema:**
   - Create custom type definitions (V1) — skip those in `reuse_existing` or identified as identical
   - Create entity types (V1) with attributes and associations — skip reused/identical
   - Create CV with label, description, lifecycle=development
   - Create entity type pins + auto-pin type definitions
   - Commit
5. **Transaction 2 — Data:**
   - Create catalog with name, description, validation_status=draft
   - Walk instance tree recursively: create instances, set IAVs, establish containment (parent references)
   - Create association links (resolve target by name + target_path within imported data)
   - Commit
6. Return success with summary (created counts)

### Collision Rules

| Scenario | Dry-Run Resolution |
|----------|-------------------|
| Entity type — same name, identical schema, V1 | `identical` — auto-suggest "reuse" (likely re-import) |
| Entity type — same name, identical schema, V>1 | `identical` — user chooses "reuse" or "create new" |
| Entity type — same name, different schema | `conflict` — must rename |
| Type definition — same name, identical constraints (field-by-field) | `identical` — same logic as entity types |
| Type definition — same name, different constraints | `conflict` — must rename |
| Catalog — same name | `conflict` — must rename via `catalog_name` override |
| CV — same label | `conflict` — must rename via `catalog_version_label` override |

### Partial Failure Recovery

If Transaction 1 succeeds but Transaction 2 fails:
- Error response explains: "Schema created successfully. Data import failed: {error}. Re-import the same file to retry."
- On re-import, dry-run detects all schema entities as identical+V1 → auto-suggests "reuse all"
- Only Transaction 2 runs (schema entities reused, not recreated)

## UI

### Export UI (Catalog Detail Page)

- **"Export" button** in catalog header, visible to Admin+
- Modal with:
  - Source system field (editable, pre-filled from default)
  - Two modes:
    - **Export all** (default): one-click, triggers Save As dialog
    - **Select entities**: tree view of pinned entity types with dependency auto-check. Unchecking a dependency triggers dangling link warning.
  - If draft/invalid: warning banner with "Export anyway" / "Cancel"
  - If dropped links: warning with "Go back" / "Continue without links"
- Browser Save As dialog with default filename `{catalog-name}-export.json`

### Import UI (Catalog List Page)

- **"Import Catalog" button** in toolbar, visible to Admin+
- Multi-step wizard modal:

**Step 1 — Upload & Options:**
- File upload dropzone
- Catalog name (pre-filled from file, editable)
- CV label (pre-filled from file, editable)
- Mass rename: prefix + suffix fields (applied to all entity type and type definition names)
- "Analyze" button → dry-run

**Step 2 — Collision Resolution** (skipped if no collisions):
- Summary bar: "5 entities: 2 new, 1 identical, 2 conflicts"
- Table: Name, Type, Status (new/identical/conflict), Action
  - New: "Create" (no action needed)
  - Identical + V1: "Reuse existing" (auto-selected, note: "appears to be from a previous import")
  - Identical + V>1: "Reuse existing" / "Create new" toggle
  - Conflict: rename field (pre-filled with mass-renamed name)

**Step 3 — Confirm:**
- Summary: X entity types, Y type definitions, Z instances, W links to be created
- "Import" / "Cancel"

**Result:**
- Success: "Catalog '{name}' imported successfully" with link to catalog
- Partial failure: error explaining schema OK + data failed, suggest re-import

## Access Control (Provisional)

Phase 1: Admin+ for both import and export. Documented in PRD that this may change:
- Different roles for import vs export
- Different roles for internal vs external export
- To be refined when use cases are better understood

## File Format Versioning

`format_version: "1.0"` — importers ignore unknown fields for forward compatibility. Breaking changes require a new major version. The import endpoint validates the format version and rejects unsupported versions.

## Future Enhancements (Documented, Not Implemented)

- YAML export format option (alongside JSON)
- Graph view for entity selection (instead of tree view)
- External export plugins (FF-15) — transformation layer producing consumer-specific formats (K8s CRs, ConfigMaps, etc.)
- Streaming/pagination for large catalog exports
- Compression for export files
- Schema-only export/import mode (no instance data)
