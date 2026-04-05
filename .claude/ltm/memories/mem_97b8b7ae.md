---
id: "mem_97b8b7ae"
topic: "Coverage skills: block-by-block reconciliation for arithmetic discrepancies"
tags:
  - coverage
  - skills
  - testing
  - debugging
phase: 0
difficulty: 0.7
created_at: "2026-04-03T19:23:12.011920+00:00"
created_session: 17
---
## Lesson: "0 new uncovered" can be wrong even when scripts agree

When `uncovered-new-lines.sh` reports 0 but arithmetic shows `delta_uncovered > 0`, the scripts are checking the wrong thing — they only check if **diff lines** are covered. But new uncovered blocks can appear in **unmodified code** when:
1. New code adds a call path through an existing function with uncovered branches (e.g., calling `mapRole` from a new handler)
2. Test changes remove coverage that previously existed (e.g., new tests cover bind-error but stop exercising other paths)
3. Go recompilation changes statement boundaries

## Fix applied to skills (2026-04-03)
- `/coverage-test`: Added Step 2d "Block-by-block reconciliation" — list ALL current uncovered blocks, map each to previous report's documented blocks, any unmapped block is NEW
- `/coverage-review`: Added Phase 2b "Arithmetic Discrepancy Audit" — reviewer must independently do the reconciliation when delta > 0

## Real example
Previous: 5 uncov (Promote RO, Demote RO, Delete RO, Update bind-error, enum bind-error)
Current: 7 uncov (mapRole RO, mapRole default, Promote RO, Promote warnings loop, Demote RO, Delete RO, enum bind-error)
Update bind-error newly covered (-1), 3 new blocks (+3), net +2. Agent initially dismissed as "pre-existing recount."
