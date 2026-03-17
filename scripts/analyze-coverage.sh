#!/bin/bash
# Analyze UI browser test coverage from coverage-summary.json
# Usage: ./scripts/analyze-coverage.sh [filter]
# Examples:
#   ./scripts/analyze-coverage.sh              # show all files < 100%
#   ./scripts/analyze-coverage.sh client       # show only files matching "client"
#   ./scripts/analyze-coverage.sh CatalogDetail # show only CatalogDetail files

COVERAGE_FILE="${1:-}"
SUMMARY_FILE="/home/jsalomon/src/pc-asset-hub/ui/coverage/coverage-summary.json"

if [ ! -f "$SUMMARY_FILE" ]; then
  echo "No coverage summary found. Run: npx vitest run --config vitest.browser.config.ts --coverage"
  exit 1
fi

FILTER="${COVERAGE_FILE:-}"

echo "=== UI Coverage Analysis ==="
echo ""

# Use jq to parse and display
jq -r '
  to_entries[] |
  select(.key != "total") |
  .key as $path |
  .value.statements as $s |
  .value.branches as $b |
  .value.lines as $l |
  ($path | split("/src/") | .[-1]) as $name |
  select($s.pct < 100) |
  "\($name)\t\($s.pct)%\t\($s.covered)/\($s.total)\tbranches=\($b.pct)%"
' "$SUMMARY_FILE" | \
  if [ -n "$FILTER" ]; then grep -i "$FILTER"; else cat; fi | \
  sort -t$'\t' -k2 -n | \
  column -t -s$'\t'

echo ""
echo "=== Totals ==="
jq -r '.total | "Statements: \(.statements.pct)% (\(.statements.covered)/\(.statements.total)) | Branches: \(.branches.pct)% | Lines: \(.lines.pct)%"' "$SUMMARY_FILE"
