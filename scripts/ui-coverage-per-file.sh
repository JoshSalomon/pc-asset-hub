#!/bin/bash
# Show per-file UI statement coverage from Vitest coverage JSON.
#
# Usage:
#   ./scripts/ui-coverage-per-file.sh [coverage-json]
#   ./scripts/ui-coverage-per-file.sh                    # default: ui/coverage-browser/coverage-final.json
#
# Prerequisites:
#   cd ui && npx vitest run --config vitest.browser.config.ts --coverage --coverage.reporter=json
#
# Output: Per-file coverage table sorted by file name, with raw counts.

set -uo pipefail

COVFILE="${1:-ui/coverage-browser/coverage-final.json}"

if [ ! -f "$COVFILE" ]; then
  echo "ERROR: Coverage file '$COVFILE' not found."
  echo "Run: cd ui && npx vitest run --config vitest.browser.config.ts --coverage --coverage.reporter=json"
  exit 1
fi

echo ""
printf "%-45s %8s %10s\n" "File" "Coverage" "Statements"
printf "%-45s %8s %10s\n" "----" "--------" "----------"

jq -r '
  to_entries[] |
  .key as $path |
  .value |
  (.s | to_entries | length) as $total |
  (.s | to_entries | map(select(.value > 0)) | length) as $covered |
  if $total > 0 then
    "\($path | split("/") | .[-1])|\($covered * 1000 / $total | round / 10)%|\($covered)/\($total)"
  else empty end
' "$COVFILE" | sort | while IFS='|' read -r file pct counts; do
  printf "%-45s %8s %10s\n" "$file" "$pct" "$counts"
done

echo ""

# Totals
jq -r '
  [to_entries[] | .value.s | to_entries[] | .value] |
  { total: length, covered: [.[] | select(. > 0)] | length } |
  "Total: \(.covered)/\(.total) = \(.covered * 1000 / .total | round / 10)%"
' "$COVFILE"
