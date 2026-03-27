#!/bin/bash
# Find uncovered NEW lines in UI (.ts/.tsx) files changed vs a base ref.
# Cross-references git diff with Vitest coverage JSON.
#
# Usage:
#   scripts/uncovered-new-lines-ui.sh --main                          # compare vs main (feature branch workflow)
#   scripts/uncovered-new-lines-ui.sh --head                          # compare vs HEAD (uncommitted changes)
#   scripts/uncovered-new-lines-ui.sh --compare-to <ref>              # compare vs any git ref
#   scripts/uncovered-new-lines-ui.sh                                 # default: --head
#   scripts/uncovered-new-lines-ui.sh --main path/to/coverage.json    # explicit coverage file
#
# Common workflows:
#   Before committing (check uncommitted work):
#     scripts/uncovered-new-lines-ui.sh --head
#
#   Before merging branch to main (check all branch changes):
#     scripts/uncovered-new-lines-ui.sh --main
#
#   Compare against specific commit or branch:
#     scripts/uncovered-new-lines-ui.sh --compare-to origin/main
#     scripts/uncovered-new-lines-ui.sh --compare-to abc1234
#
# Prerequisites:
#   cd ui && npx vitest run --config vitest.browser.config.ts --coverage
#
# Output: For each modified .ts/.tsx file, lists uncovered line numbers that are NEW
#         (added or changed vs base). Skips test files.

set -uo pipefail

# Must run from project root
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

# Parse arguments
BASE="HEAD"  # default: compare uncommitted changes
COVFILE=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --main)
      BASE="main"
      shift
      ;;
    --head)
      BASE="HEAD"
      shift
      ;;
    --compare-to)
      BASE="${2:?--compare-to requires a git ref argument}"
      shift 2
      ;;
    --help|-h)
      echo "Usage: $0 [--main|--head|--compare-to <ref>] [coverage-json]"
      echo ""
      echo "Options:"
      echo "  --main              Compare vs main branch (for pre-merge checks)"
      echo "  --head              Compare vs HEAD (for uncommitted changes, default)"
      echo "  --compare-to <ref>  Compare vs any git ref (commit, branch, tag)"
      echo ""
      echo "Positional:"
      echo "  coverage-json       Path to coverage-final.json (default: ui/coverage/coverage-final.json)"
      exit 0
      ;;
    *)
      # Positional arg = coverage file
      COVFILE="$1"
      shift
      ;;
  esac
done

COVFILE="${COVFILE:-ui/coverage/coverage-final.json}"

if [ ! -f "$COVFILE" ]; then
  echo "ERROR: Coverage file '$COVFILE' not found."
  echo "Run: cd ui && npx vitest run --config vitest.browser.config.ts --coverage"
  exit 1
fi

echo "Comparing vs: $BASE"
echo "Coverage file: $COVFILE"
echo ""

# Get list of modified UI production files (exclude tests)
CHANGED_FILES=$(git diff --name-only "$BASE" -- '*.ts' '*.tsx' | grep -v '\.test\.' | grep -v '\.spec\.' | grep -v 'types/')
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
  FULL_PATH="${PROJECT_ROOT}/${FILE}"

  # Extract uncovered statement start lines from coverage JSON
  UNCOV=$(jq -r --arg fp "$FULL_PATH" '
    .[$fp] // empty |
    .statementMap as $sm |
    .s | to_entries[] | select(.value == 0) | $sm[.key] | .start.line
  ' "$COVFILE" 2>/dev/null | sort -nu)

  if [ -z "$UNCOV" ]; then
    # File not in coverage data — might be new file not yet instrumented
    NEW_COUNT=$(echo "$NEW_LINES" | wc -l)
    if [ "$NEW_COUNT" -gt 0 ]; then
      echo "WARNING: $FILE has $NEW_COUNT new lines but is NOT in coverage data!"
      echo "  This file may not be instrumented by V8 coverage."
      echo ""
    fi
    continue
  fi

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
