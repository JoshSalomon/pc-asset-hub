#!/bin/bash
# Find uncovered NEW lines in UI (.ts/.tsx) files changed on the current branch.
# Cross-references git diff (vs base branch) with Vitest coverage JSON.
#
# Usage:
#   ./scripts/uncovered-new-lines-ui.sh [base-branch] [coverage-json]
#   ./scripts/uncovered-new-lines-ui.sh main ui/coverage-browser/coverage-final.json
#   ./scripts/uncovered-new-lines-ui.sh          # defaults: main, ui/coverage-browser/coverage-final.json
#
# Prerequisites:
#   cd ui && npx vitest run --config vitest.browser.config.ts --coverage --coverage.reporter=json
#
# Output: For each modified .ts/.tsx file, lists uncovered line numbers that are NEW
#         (added or changed in this branch). Skips test files.

set -uo pipefail

BASE="${1:-main}"
COVFILE="${2:-ui/coverage-browser/coverage-final.json}"

if [ ! -f "$COVFILE" ]; then
  echo "ERROR: Coverage file '$COVFILE' not found."
  echo "Run: cd ui && npx vitest run --config vitest.browser.config.ts --coverage --coverage.reporter=json"
  exit 1
fi

# Get list of modified UI production files (exclude tests)
CHANGED_FILES=$(git diff --name-only "$BASE" -- '*.ts' '*.tsx' | grep -v '\.test\.' | grep -v '\.spec\.')
if [ -z "$CHANGED_FILES" ]; then
  echo "No modified .ts/.tsx production files found vs $BASE."
  exit 0
fi

TOTAL_UNCOVERED=0

for FILE in $CHANGED_FILES; do
  [ -f "$FILE" ] || continue

  # Get new line numbers from git diff
  NEW_LINES=$(git diff -U0 "$BASE" -- "$FILE" | awk '
    /^@@/ {
      match($0, /\+([0-9]+)(,([0-9]+))?/, arr)
      start = arr[1]
      count = arr[3] == "" ? 1 : arr[3]
      for (i = start; i < start + count; i++) print i
    }
  ' | sort -nu)

  [ -z "$NEW_LINES" ] && continue

  # Get full absolute path for matching in coverage JSON
  FULL_PATH="$(cd "$(dirname "$FILE")" && pwd)/$(basename "$FILE")"

  # Extract uncovered statement start lines from coverage JSON
  UNCOV=$(jq -r --arg fp "$FULL_PATH" '
    .[$fp] // empty |
    .statementMap as $sm |
    .s | to_entries[] | select(.value == 0) | $sm[.key] | .start.line
  ' "$COVFILE" 2>/dev/null | sort -nu)

  [ -z "$UNCOV" ] && continue

  # Find intersection: lines that are both NEW and UNCOVERED
  OVERLAP=$(grep -Fxf <(echo "$UNCOV") <(echo "$NEW_LINES") || true)

  if [ -n "$OVERLAP" ]; then
    COUNT=$(echo "$OVERLAP" | wc -l)
    TOTAL_UNCOVERED=$((TOTAL_UNCOVERED + COUNT))
    echo "=== $FILE ($COUNT uncovered new lines) ==="
    for LINE in $OVERLAP; do
      CODE=$(sed -n "${LINE}p" "$FILE" | sed 's/^[[:space:]]*/  /')
      echo "  L${LINE}: $CODE"
    done
    echo ""
  fi
done

if [ "$TOTAL_UNCOVERED" -eq 0 ]; then
  echo "All new UI lines are covered! (0 uncovered)"
  exit 0
else
  echo "Total uncovered new UI lines: $TOTAL_UNCOVERED"
  exit 1
fi
