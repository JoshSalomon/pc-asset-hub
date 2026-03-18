#!/bin/bash
# Show uncovered line ranges from coverage-final.json
# Usage: ./scripts/analyze-coverage-lines.sh <file-pattern> [min-line]
# Examples:
#   ./scripts/analyze-coverage-lines.sh CatalogDetailPage        # all uncovered lines
#   ./scripts/analyze-coverage-lines.sh CatalogDetailPage 1000   # only lines >= 1000

PATTERN="${1:?Usage: $0 <file-pattern> [min-line]}"
MIN_LINE="${2:-0}"
COVERAGE_FILE="/home/jsalomon/src/pc-asset-hub/ui/coverage/coverage-final.json"

if [ ! -f "$COVERAGE_FILE" ]; then
  echo "No coverage data found. Run: npx vitest run --config vitest.browser.config.ts --coverage --coverage.reporter=json"
  exit 1
fi

jq -r --arg pat "$PATTERN" --argjson min "$MIN_LINE" '
  to_entries[] |
  select(.key | test($pat)) |
  .key as $path |
  .value.statementMap as $stmts |
  .value.s as $counts |
  ($path | split("/src/") | .[-1]) as $name |
  "\($name)",
  ($counts | to_entries[] |
    select(.value == 0) |
    .key as $sid |
    $stmts[$sid] |
    select(.start.line >= $min) |
    "  Line \(.start.line)-\(.end.line)"
  )
' "$COVERAGE_FILE" | sort -t'-' -k1 -n -u
