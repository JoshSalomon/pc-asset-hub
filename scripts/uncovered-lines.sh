#!/bin/bash
# Show uncovered lines for a specific file from v8 coverage
# Usage: ./scripts/uncovered-lines.sh <file-pattern> [line-start] [line-end]
# Example: ./scripts/uncovered-lines.sh CatalogDetailPage 440 480

PATTERN="${1:?Usage: uncovered-lines.sh <file-pattern> [line-start] [line-end]}"
LINE_START="${2:-0}"
LINE_END="${3:-99999}"
COVERAGE_DIR="/home/jsalomon/src/pc-asset-hub/ui/coverage"

# Find the v8 coverage JSON file
V8_FILE=$(ls "$COVERAGE_DIR"/coverage-*.json 2>/dev/null | head -1)
if [ -z "$V8_FILE" ]; then
  echo "No v8 coverage data found. Run with --coverage first."
  exit 1
fi

# Use the summary file instead — it has per-file stats but not line detail
# The line-level data is in the lcov report or the html
# Let's parse the text table from the coverage output

SUMMARY_FILE="$COVERAGE_DIR/coverage-summary.json"
if [ ! -f "$SUMMARY_FILE" ]; then
  echo "No coverage summary found."
  exit 1
fi

# Show matching files from summary
echo "=== Coverage for files matching '$PATTERN' ==="
jq -r "
  to_entries[] |
  select(.key | test(\"$PATTERN\"; \"i\")) |
  select(.key != \"total\") |
  .key as \$path |
  (\$path | split(\"/src/\") | .[-1]) as \$name |
  \"\(\$name): stmts=\(.value.statements.pct)% branches=\(.value.branches.pct)% lines=\(.value.lines.pct)%\"
" "$SUMMARY_FILE"

echo ""
echo "To see exact uncovered lines, check: $COVERAGE_DIR/index.html"
