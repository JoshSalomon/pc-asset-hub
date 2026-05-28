---
id: "mem_81fc6f3b"
topic: "Coverage measurement: reproducible method using scripts/go-coverage-table.sh"
tags:
  - coverage
  - measurement
  - baseline
  - scripts
  - reproducibility
phase: 0
difficulty: 0.8
created_at: "2026-05-26T11:48:53.458953+00:00"
created_session: 17
---
Coverage numbers must be measured using `scripts/go-coverage-table.sh` on a fresh `coverage.out` from `go test ./internal/... -count=1 -coverprofile=coverage.out`. 

**Why:** The script has a field-swap bug (reads execution count as statement count), but ALL historical baselines use this script. Switching methods produces incomparable numbers. The uncovered count (91 as of Session 030) is stable across runs; the covered/total counts vary slightly but the uncovered count is deterministic.

**How to apply:** Always regenerate coverage.out fresh. Always use `scripts/go-coverage-table.sh`. Never use `-coverpkg` or custom parsers. See `docs/coverage-measurement.md` for full instructions.

**Key numbers (Session 030):** Main baseline: 5911/6002 (91 uncov). Branch 020-td-sprint: 5956/6047 (91 uncov). The report's previous baseline of 5816/5898 (82 uncov) was from a stale profile and cannot be reproduced.
