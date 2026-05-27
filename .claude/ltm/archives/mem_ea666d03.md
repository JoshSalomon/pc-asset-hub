---
id: "mem_ea666d03"
topic: "go-coverage-table.sh field-swap bug fixed in Session 030 — re-baselined all coverage numbers"
tags:
  - coverage
  - bug-fix
  - go-coverage-table
  - baseline
  - measurement
phase: 0
difficulty: 0.9
created_at: "2026-05-26T11:55:52.757506+00:00"
created_session: 17
---
`scripts/go-coverage-table.sh` had a bug since its creation: it swapped `count` (execution count, group 4) and `num_stmts` (statement count, group 5) from Go's coverprofile format. This inflated totals (summing execution counts instead of statement counts) and created phantom "uncovered" units (covered blocks with 0 statements counted as uncovered).

**Fixed in Session 030.** The swap was corrected. All numbers in `docs/coverage-report.md` were re-baselined.

**Before fix:** Main reported 5816/5898 (82 uncov). Could not be reproduced — fresh runs gave 5911/6002 (91 uncov). The 91 "uncovered" were all phantom.

**After fix:** Main = 4150/4151 (1 uncov). Branch 020-td-sprint = 4177/4178 (1 uncov). The 1 uncovered is `preview_cache.go:86` — an empty `case <-ticker.C:` select body with 0 statements.

**How to measure:** See `docs/coverage-measurement.md`. Always use `go test ./internal/... -count=1 -coverprofile=coverage.out` then `scripts/go-coverage-table.sh coverage.out`. Never use `-coverpkg`, never use custom parsers.
