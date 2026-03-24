#!/bin/bash
# Show all uncovered lines in a specific UI file from Vitest coverage JSON.
#
# Usage:
#   ./scripts/ui-uncovered-lines.sh <file-pattern> [coverage-json]
#   ./scripts/ui-uncovered-lines.sh EntityTypeDetailPage
#   ./scripts/ui-uncovered-lines.sh CatalogDetailPage ui/coverage-browser/coverage-final.json
#
# Prerequisites:
#   cd ui && npx vitest run --config vitest.browser.config.ts --coverage --coverage.reporter=json

set -uo pipefail

PATTERN="${1:?Usage: $0 <file-pattern> [coverage-json]}"
COVFILE="${2:-ui/coverage-browser/coverage-final.json}"

if [ ! -f "$COVFILE" ]; then
  echo "ERROR: Coverage file '$COVFILE' not found."
  echo "Run: cd ui && npx vitest run --config vitest.browser.config.ts --coverage --coverage.reporter=json"
  exit 1
fi

# Find matching file(s) in coverage JSON
MATCHES=$(jq -r "to_entries[] | select(.key | test(\"$PATTERN\")) | .key" "$COVFILE")

if [ -z "$MATCHES" ]; then
  echo "No files matching '$PATTERN' found in coverage data."
  exit 1
fi

for FILEPATH in $MATCHES; do
  FILENAME=$(basename "$FILEPATH")

  # Get coverage stats
  STATS=$(jq -r --arg fp "$FILEPATH" '
    .[$fp] |
    (.s | to_entries | length) as $total |
    (.s | to_entries | map(select(.value > 0)) | length) as $covered |
    "\($covered)/\($total) = \($covered * 1000 / $total | round / 10)%"
  ' "$COVFILE")

  echo "=== $FILENAME ($STATS) ==="
  echo ""

  # Get uncovered statement start lines
  UNCOVERED=$(jq -r --arg fp "$FILEPATH" '
    .[$fp] |
    .statementMap as $sm |
    .s | to_entries[] | select(.value == 0) |
    $sm[.key] | .start.line
  ' "$COVFILE" | sort -nu)

  if [ -z "$UNCOVERED" ]; then
    echo "  All lines covered!"
    continue
  fi

  COUNT=$(echo "$UNCOVERED" | wc -l)
  echo "  $COUNT uncovered statements:"
  echo ""

  for LINE in $UNCOVERED; do
    if [ -f "$FILEPATH" ]; then
      CODE=$(sed -n "${LINE}p" "$FILEPATH" | sed 's/^[[:space:]]*//')
    else
      CODE="(file not found at $FILEPATH)"
    fi
    printf "  L%-5d %s\n" "$LINE" "$CODE"
  done
  echo ""
done
