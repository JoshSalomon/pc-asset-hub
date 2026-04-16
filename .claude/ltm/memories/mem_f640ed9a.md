---
id: "mem_f640ed9a"
topic: "Type system constraint validation architecture (TD-90/91/92)"
tags:
  - type-system
  - validation
  - architecture
  - td-90
  - td-91
  - td-92
phase: 0
difficulty: 0.6
created_at: "2026-04-15T11:20:04.968359+00:00"
created_session: 17
---
## Constraint Validation Architecture

### Backend (TD-90)
- `constraint_validator.go` — pure function `ValidateValueConstraints()`, no repo access
- `CompilePatternConstraint()` pre-compiles regex once per attribute during schema loading
- Bad regex → ONE error per attribute (not per instance), reported even with empty values
- `validateMinMax()` shared helper for integer and number types
- Corrupted constraints (`_raw` key) are skipped — handled separately in main validation loop
- Enum validation handled in main loop, not in constraint validator

### Frontend (TD-91/92)
- `formatAttributeValue.tsx` — type-aware rendering: URL→link (http/https only, XSS defense), boolean→Yes/No, date→formatted, JSON→pre, list→comma-separated
- `validateAttributeValue.ts` — advisory warnings mirroring backend rules (not blocking form submission)
- Inline warnings use PatternFly `HelperText` with `variant="warning"`
- `CreateInstanceModal` initializes mandatory boolean attrs to `"false"` via useEffect
- Date validation uses round-trip check: parse date, rebuild from components, compare — catches impossible dates like Feb 31

### Key differences backend vs frontend
- Frontend is advisory only, backend is authoritative
- Frontend skips `element_base_type` list validation (TD-107)
- Frontend regex uses PCRE, backend uses RE2 — pattern dialect mismatch possible
- Frontend regex recompiled per render (TD-108), backend pre-compiles once

### Test infrastructure
- `vitest.unit.config.ts` — pure function tests run in Node (not browser), much faster
- `scripts/test-type-constraints.sh` — 8 live API tests for constraint validation
- Both added to Makefile (`test-unit`, `test-live`)
