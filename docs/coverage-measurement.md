# Coverage Measurement — Reproducible Method

## Backend Coverage

### Step 1: Generate coverprofile

```bash
go test ./internal/... -count=1 -coverprofile=coverage.out
```

This produces a `mode: set` profile. The `-count=1` disables test caching.

### Step 2: Per-package table with statement counts

```bash
scripts/go-coverage-table.sh coverage.out
```

This script parses the coverprofile and outputs a markdown table with `covered/total` per package plus the production total and uncovered count.

Go coverprofile format: `file:startLine.startCol,endLine.endCol numStmts count`
- `numStmts` = number of statements in the block
- `count` = execution count (0 = not covered, >0 = covered)

The script correctly reads `group(4)` as `numStmts` and `group(5)` as `count`.

### Step 3: Verify

The per-package uncovered counts are stable across runs. The total covered/total numbers vary slightly between runs because different test execution orders can cause minor differences in which blocks get exercised. The uncovered count is the reliable metric for comparison.

### What NOT to do

- Do NOT use `go test -coverpkg=./internal/...` — this produces different statement counts (cross-package coverage) that are not comparable with the baseline.
- Do NOT compare percentages without statement counts. Always report `covered/total`.
- Do NOT use a stale `coverage.out` from a previous run. Always regenerate.

## UI Coverage

```bash
cd ui
NODE_OPTIONS="--max-old-space-size=4096" npx vitest run --config vitest.browser.config.ts --coverage --reporter=dot
```

This takes 15-30 minutes in container environments. Coverage data lands in `ui/coverage/coverage-final.json`.

Per-file statement counts:
```bash
scripts/ui-coverage-per-file.sh
```

## New-line coverage check

Backend:
```bash
scripts/uncovered-new-lines.sh --compare-to <base-ref>
```

UI:
```bash
scripts/uncovered-new-lines-ui.sh
```

Note: `uncovered-new-lines.sh` has awk syntax errors on some systems (gawk vs mawk). If it reports errors but concludes "0 uncovered", verify manually.
