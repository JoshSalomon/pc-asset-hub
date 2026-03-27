#!/bin/bash
# Find uncovered NEW lines in Go files changed vs a base ref.
# Cross-references git diff with Go coverage profile.
#
# Usage:
#   scripts/uncovered-new-lines.sh --main                          # compare vs main (feature branch workflow)
#   scripts/uncovered-new-lines.sh --head                          # compare vs HEAD (uncommitted changes)
#   scripts/uncovered-new-lines.sh --compare-to <ref>              # compare vs any git ref
#   scripts/uncovered-new-lines.sh                                 # default: --head
#   scripts/uncovered-new-lines.sh --main coverage.out             # explicit coverage file
#
# Common workflows:
#   Before committing (check uncommitted work):
#     scripts/uncovered-new-lines.sh --head
#
#   Before merging branch to main (check all branch changes):
#     scripts/uncovered-new-lines.sh --main
#
#   Compare against specific commit or branch:
#     scripts/uncovered-new-lines.sh --compare-to origin/main
#
# Prerequisites:
#   go test ./internal/... -count=1 -coverprofile=coverage.out
# Note: Do NOT use -coverpkg=./internal/... as it causes cross-package coverage
# measurement issues where lines covered by tests in other packages appear uncovered.
#
# Output: For each modified .go file, lists uncovered line ranges that are NEW
#         (added or changed vs base). Skips test files.

set -uo pipefail

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
      echo "Usage: $0 [--main|--head|--compare-to <ref>] [coverage-file]"
      echo ""
      echo "Options:"
      echo "  --main              Compare vs main branch (for pre-merge checks)"
      echo "  --head              Compare vs HEAD (for uncommitted changes, default)"
      echo "  --compare-to <ref>  Compare vs any git ref (commit, branch, tag)"
      echo ""
      echo "Positional:"
      echo "  coverage-file       Path to coverage.out (default: coverage.out)"
      exit 0
      ;;
    *)
      # Positional arg = coverage file
      COVFILE="$1"
      shift
      ;;
  esac
done

COVFILE="${COVFILE:-coverage.out}"

if [ ! -f "$COVFILE" ]; then
  echo "ERROR: Coverage file '$COVFILE' not found."
  echo "Run: go test ./internal/... -count=1 -coverprofile=$COVFILE"
  exit 1
fi

echo "Comparing vs: $BASE"
echo "Coverage file: $COVFILE"
echo ""

# Get list of modified Go production files (exclude tests)
CHANGED_FILES=$(git diff --name-only "$BASE" -- '*.go' | grep -v '_test\.go$')
if [ -z "$CHANGED_FILES" ]; then
  echo "No modified .go production files found vs $BASE."
  exit 0
fi

TOTAL_UNCOVERED=0

for FILE in $CHANGED_FILES; do
  [ -f "$FILE" ] || continue

  # Get the Go import path suffix for matching in coverage.out
  BASENAME=$(basename "$FILE")
  DIR=$(dirname "$FILE")

  # Get new line numbers from git diff (only added/changed lines)
  NEW_LINES=$(git diff -U0 "$BASE" -- "$FILE" | awk '
    /^@@/ {
      match($0, /\+([0-9]+)(,([0-9]+))?/, arr)
      start = arr[1]
      count = arr[3] == "" ? 1 : arr[3]
      for (i = start; i < start + count; i++) print i
    }
  ' | sort -nu)

  [ -z "$NEW_LINES" ] && continue

  # Get uncovered line ranges from coverage.out for this file
  # Format: path/file.go:startline.col,endline.col numstmts count
  UNCOVERED_RANGES=$(grep "$DIR/$BASENAME" "$COVFILE" | awk '$NF == 0 {
    split($1, parts, ":")
    split(parts[2], range, ",")
    split(range[1], start, ".")
    split(range[2], end, ".")
    for (i = start[1]; i <= end[1]; i++) print i
  }' | sort -nu)

  [ -z "$UNCOVERED_RANGES" ] && continue

  # Find intersection: lines that are both NEW and UNCOVERED
  UNCOVERED_NEW=$(grep -Fxf <(echo "$UNCOVERED_RANGES") <(echo "$NEW_LINES") || true)

  if [ -n "$UNCOVERED_NEW" ]; then
    COUNT=$(echo "$UNCOVERED_NEW" | wc -l)
    TOTAL_UNCOVERED=$((TOTAL_UNCOVERED + COUNT))
    echo "=== $FILE ($COUNT uncovered new lines) ==="
    for LINE in $UNCOVERED_NEW; do
      CODE=$(sed -n "${LINE}p" "$FILE" | sed 's/^\t/  /')
      echo "  L${LINE}: $CODE"
    done
    echo ""
  fi
done

if [ "$TOTAL_UNCOVERED" -eq 0 ]; then
  echo "All new Go lines are covered! (0 uncovered)"
  exit 0
else
  echo "Total uncovered new Go lines: $TOTAL_UNCOVERED"
  exit 1
fi
