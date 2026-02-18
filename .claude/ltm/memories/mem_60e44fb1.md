---
id: "mem_60e44fb1"
topic: "GORM ORDER BY SQL injection prevention"
tags:
  - security
  - gorm
  - sql-injection
  - gotcha
phase: 0
difficulty: 0.7
created_at: "2026-02-15T21:17:56.737324+00:00"
created_session: 1
---
## GORM .Order() is NOT parameterized

GORM's `.Order()` method inserts its argument directly into the SQL query as a raw string — it does NOT use parameterized queries for ORDER BY. This means passing user input directly to `.Order()` creates a SQL injection vulnerability.

**Vulnerable pattern:**
```go
query.Order(params.SortBy) // DANGEROUS: user controls the string
```

**Fix:** Validate against an allowlist before passing to `.Order()`:
```go
var allowedSortColumns = map[string]bool{
    "name": true, "created_at": true, "updated_at": true, "version": true,
}

func validateSortBy(sortBy string) error {
    if sortBy != "" && !allowedSortColumns[sortBy] {
        return fmt.Errorf("invalid sort column: %s", sortBy)
    }
    return nil
}
```

This was caught during the Phase A security audit. The fix is in `internal/infrastructure/gorm/repository/helpers.go`.
