---
id: "mem_4ca1cb84"
topic: "Catalog Import/Export (FF-12) implementation"
tags:
  - import-export
  - FF-12
  - architecture
  - catalog
phase: 0
difficulty: 0.7
created_at: "2026-04-30T15:42:58.266341+00:00"
created_session: 17
---
## Catalog Import/Export (FF-12) — Implemented

**Export**: `GET /api/data/v1/catalogs/{name}/export` with optional `?entities=` filter and `?source_system=` override. Returns JSON v1.0 format with nested containment (children nested by association name, not flat array), parsed list/json attribute values, names as references (not IDs).

**Import**: `POST /api/data/v1/catalogs/import?dry_run=true|false`. Two-transaction model: T1 creates schema (type defs, entity types, CV, pins), T2 creates data (catalog, instances, IAVs, links). Supports `rename_map` (entity types + type defs only), `reuse_existing` for identical+V1 entities, and `catalog_version_label` override.

**Key architectural decisions**:
- References by name not ID — portable across deployments
- Instances nested by containment association name in JSON
- Pins always imported as V1 (fresh start)
- CV lifecycle always `development`, validation status always `draft`
- Collision detection is field-by-field per base type (not JSON string match)
- Bidirectional links can be created from either side (reverse-side lookup added)
- List/JSON attribute values stored as parsed JSON in export file, re-serialized on import

**UI**: `ImportCatalogModal` — 4-step wizard (upload → collisions → confirm → done). Drag-and-drop file upload with counter-based dragLeave. Collision table with reuse toggles and rename preview. 50MB file size limit.

**Test scripts**: `scripts/test-import-export.sh` (30 API tests), `scripts/test-import-export-e2e.sh` (76-assertion round-trip).

**Bug found during coverage**: `vi.stubGlobal('URL')` breaks V8/Istanbul coverage pipeline — overrides the global URL constructor needed by coverage reporters.

**Bug found during system tests**: `getByRole('button', { name: 'Export' })` and `{ name: 'Import' }` need `{ exact: true }` to avoid matching "Import Catalog" button substring.
