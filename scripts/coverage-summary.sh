#!/bin/bash
# Generate coverage summary for the coverage report.
# Collects test counts and coverage percentages from all layers.
#
# Usage:
#   ./scripts/coverage-summary.sh
#
# Prerequisites:
#   - Backend tests must pass (go test ./internal/...)
#   - Browser tests must have been run with coverage (npx vitest --coverage)
#   - coverage_combined.out should exist (or will be generated)
#
# Output: Formatted summary table for docs/coverage-report.md

set -uo pipefail

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$PROJECT_ROOT"

echo "=== Collecting Test & Coverage Metrics ==="
echo ""

# ─── Backend ──────────────────────────────────────────────────────────

echo "--- Backend (Go) ---"

# Run tests with coverage if profile doesn't exist
COVFILE="coverage_combined.out"
if [ ! -f "$COVFILE" ] || [ "$COVFILE" -ot "internal" ]; then
  echo "  Generating coverage profile..."
  go test ./internal/... -count=1 -coverprofile="$COVFILE" > /dev/null 2>&1
fi

# Count tests
BACKEND_TESTS=$(go test ./internal/... -count=1 -v 2>&1 | grep -c "^--- PASS:")
echo "  Tests: $BACKEND_TESTS"

# Overall coverage
BACKEND_COV=$(go tool cover -func="$COVFILE" | tail -1 | awk '{print $NF}')
echo "  Coverage: $BACKEND_COV"

# Per-package coverage
echo ""
echo "  Per-package:"
go test ./internal/... -count=1 -coverprofile="$COVFILE" 2>&1 | grep "^ok" | grep "coverage:" | while read -r _ pkg _ _ cov _; do
  short=$(echo "$pkg" | sed 's|github.com/project-catalyst/pc-asset-hub/||')
  pct=$(echo "$cov" | sed 's/coverage: //' | sed 's/ of statements//')
  printf "    %-55s %s\n" "$short" "$pct"
done

echo ""

# ─── UI Browser Tests ────────────────────────────────────────────────

echo "--- UI Browser Tests ---"

COVJSON="ui/coverage-browser/coverage-final.json"
if [ -f "$COVJSON" ]; then
  # Count tests from last run
  BROWSER_TESTS=$(cd ui && npx vitest run --config vitest.browser.config.ts 2>&1 | grep "Tests" | grep -oP '\d+ passed' | grep -oP '\d+')
  echo "  Tests: $BROWSER_TESTS"

  # Per-file coverage
  echo "  Per-file statement coverage:"
  jq -r '
    to_entries[] |
    .key as $path |
    .value |
    (.s | to_entries | length) as $total |
    (.s | to_entries | map(select(.value > 0)) | length) as $covered |
    if $total > 0 then
      "\($path | split("/") | .[-1])\t\($covered)/\($total)\t\(($covered * 1000 / $total + 5) / 10 | floor)%"
    else empty end
  ' "$COVJSON" | sort | while IFS=$'\t' read -r file ratio pct; do
    printf "    %-45s %s (%s)\n" "$file" "$pct" "$ratio"
  done
else
  echo "  Coverage JSON not found. Run: cd ui && npx vitest run --config vitest.browser.config.ts --coverage"
fi

echo ""

# ─── UI System Tests ─────────────────────────────────────────────────

echo "--- UI System Tests ---"
SYSTEM_TESTS=$(cd ui && npx vitest run --config vitest.system.config.ts 2>&1 | grep -oP '\d+ passed' | head -1 | grep -oP '^\d+' || echo "0")
echo "  Tests: $SYSTEM_TESTS"

echo ""

# ─── Live Scripts ─────────────────────────────────────────────────────

echo "--- Live System Tests (bash scripts) ---"
LIVE_TOTAL=0
for f in scripts/test-*.sh; do
  [ -f "$f" ] || continue
  NAME=$(basename "$f" .sh)
  # Count pass/fail calls as approximate test count
  COUNT=$(grep -cE '^\s+(pass|fail)\b' "$f" 2>/dev/null || echo 0)
  LIVE_TOTAL=$((LIVE_TOTAL + COUNT))
  printf "    %-40s %d tests\n" "$NAME" "$COUNT"
done
echo "  Total: $LIVE_TOTAL"

echo ""

# ─── Grand Total ──────────────────────────────────────────────────────

echo "=== Summary ==="
echo ""
echo "| Layer | Tests | Coverage |"
echo "|-------|-------|----------|"
echo "| Backend (Go) | $BACKEND_TESTS | $BACKEND_COV |"
echo "| Browser tests | ${BROWSER_TESTS:-?} | see per-file |"
echo "| System tests | ${SYSTEM_TESTS:-?} | — |"
echo "| Live scripts | $LIVE_TOTAL | — |"
SYS_COUNT=${SYSTEM_TESTS%%[!0-9]*}  # strip any non-numeric chars
BROWSER_COUNT=${BROWSER_TESTS%%[!0-9]*}
GRAND_TOTAL=$((BACKEND_TESTS + ${BROWSER_COUNT:-0} + ${SYS_COUNT:-0} + LIVE_TOTAL))
echo "| **Total** | **$GRAND_TOTAL** | — |"
