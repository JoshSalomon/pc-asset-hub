---
id: "mem_dabfbb44"
topic: "Instance CRUD with Attributes — implementation patterns and migration gotcha"
tags:
  - instance
  - attributes
  - migration
  - gotchas
  - postgresql
phase: 0
difficulty: 0.7
created_at: "2026-03-10T15:42:43.801262+00:00"
created_session: 16
---
## Instance CRUD with Attributes (Phase 2 — implemented)

### Architecture
- `InstanceService` resolves catalog name → CV → pins → entity type version → attributes
- Pin resolution iterates pins and matches `EntityTypeVersion.EntityTypeID` against the resolved entity type
- Attribute values are validated by type (string=any, number=parseable, enum=in allowed list)
- Missing required attributes allowed in draft mode (enforcement is Phase 5 Validation)
- On update, previous version's attribute values are **carried forward** — only changed attrs need to be sent

### Key Files
- Service: `internal/service/operational/instance_service.go`
- Handler: `internal/api/operational/instance_handler.go`
- DTOs: `internal/api/dto/dto.go` (InstanceResponse, CreateInstanceRequest, UpdateInstanceRequest)
- UI: `ui/src/pages/operational/CatalogDetailPage.tsx`

### Gotchas
- **PostgreSQL migration**: When renaming a GORM column (e.g., `catalog_version_id` → `catalog_id`), AutoMigrate adds the new column but does NOT drop the old one. Must add explicit pre-migration in `InitDB` to copy data and `DROP COLUMN`. Use `information_schema.columns` for PostgreSQL and `PRAGMA table_info` for SQLite to detect old columns.
- **GORM `HasColumn` checks the model, not the DB**: `db.Migrator().HasColumn(&Model{}, "old_column")` returns false if the field was removed from the Go struct, even if the DB column still exists. Query `information_schema` directly instead.
- **Browser test `getByText` doesn't work inside PatternFly tables**: Use `getByRole('gridcell', { name: '...' })` instead of `getByText('...')` for table cell content.
- **Update must carry forward unchanged attrs**: Without this, `GetCurrentValues` (which queries by max version) loses previous version's values for unchanged attributes.
