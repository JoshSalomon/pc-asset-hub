---
id: "mem_8934acc7"
topic: "FF-1 Association Cardinality implementation"
tags:
  - association
  - cardinality
  - FF-1
  - feature
phase: 0
difficulty: 0.6
created_at: "2026-03-02T12:42:57.457650+00:00"
created_session: 13
---
## FF-1 Association Cardinality

Implemented UML-style multiplicity (`source_cardinality`, `target_cardinality`) on associations.

### Key decisions
- Standard options: `0..1`, `0..n`, `1`, `1..n` + custom ranges (e.g., `2..5`, `2..n`)
- Default: `0..n` on both ends, except containment source defaults to `0..1`
- **Containment constraint**: source cardinality restricted to `1` or `0..1` (entity contained by at most one parent)
- Validation via regex `^(\d+)(\.\.(\d+|n))?$` in `internal/service/validation/cardinality.go`
- `NormalizeSourceCardinality(s, isContainment)` for type-aware defaults
- Handler-level normalization kept as safety net for legacy DB rows
- DB migration in `InitDB` fixes existing containment associations (empty/0..n → 0..1)

### Files added
- `internal/service/validation/cardinality.go` + `cardinality_test.go`

### Test IDs: T-E.68 through T-E.94 (27 tests)

