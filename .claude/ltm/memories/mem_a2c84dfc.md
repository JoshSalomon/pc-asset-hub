---
id: "mem_a2c84dfc"
topic: "SQLite in-memory testing with GORM - key configuration"
tags:
  - go
  - testing
  - sqlite
  - gorm
  - gotcha
phase: 0
difficulty: 0.4
created_at: "2026-02-15T21:17:45.861365+00:00"
created_session: 1
---
## SQLite In-Memory Test DB Setup

The test helper at `internal/infrastructure/gorm/testutil/testdb.go` uses:

```go
db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&_foreign_keys=on"), &gorm.Config{
    Logger: logger.Default.LogMode(logger.Silent),
})
```

Key settings:
- `cache=shared` — required for SQLite in-memory DBs to be shared across connections within the same process
- `_foreign_keys=on` — SQLite doesn't enforce foreign keys by default; this pragma enables them
- `LogMode(logger.Silent)` — suppresses GORM SQL logging noise in test output

Each test gets a fresh DB via `testutil.NewTestDB(t)` with auto-migration. The `t.Cleanup` closes the connection after the test.

**Gotcha**: SQLite doesn't enforce FK constraints on self-referencing columns (like `parent_instance_id` on `entity_instances`) the same way PostgreSQL does. Parent existence validation must happen at the service layer, not rely on DB constraints alone.
