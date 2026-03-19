#!/bin/bash
# Find uncovered NEW lines in Go files changed on the current branch.
# Cross-references git diff (vs base branch) with Go coverage profile.
#
# Usage:
#   ./scripts/uncovered-new-lines.sh [base-branch] [coverage-file]
#   ./scripts/uncovered-new-lines.sh main coverage.out
#   ./scripts/uncovered-new-lines.sh              # defaults: main, coverage.out
#
# Prerequisites:
#   go test ./internal/... -count=1 -coverpkg=./internal/... -coverprofile=coverage.out
#
# Output: For each modified .go file, lists uncovered line ranges that are NEW
#         (added or changed in this branch). Skips test files.

set -uo pipefail

BASE="${1:-main}"
COVFILE="${2:-coverage.out}"

if [ ! -f "$COVFILE" ]; then
  echo "ERROR: Coverage file '$COVFILE' not found."
  echo "Run: go test ./internal/... -count=1 -coverpkg=./internal/... -coverprofile=$COVFILE"
  exit 1
fi

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
  # coverage.out uses full module paths like github.com/org/repo/internal/...
  # We match on the file basename within the path
  BASENAME=$(basename "$FILE")
  DIR=$(dirname "$FILE")

  # Get new line numbers from git diff (only added/changed lines)
  NEW_LINES=$(git diff -U0 "$BASE" -- "$FILE" | awk '
    /^@@/ {
      # Parse @@ -old +new,count @@ format
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
    # Parse startline.col,endline.col
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
  echo "All new lines are covered! (0 uncovered)"
  exit 0
else
  echo "Total uncovered new lines: $TOTAL_UNCOVERED"
  exit 1
fi
