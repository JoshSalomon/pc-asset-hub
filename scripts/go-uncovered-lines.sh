#!/bin/bash
# Show uncovered Go lines from a coverage profile
# Usage: ./scripts/go-uncovered-lines.sh <coverprofile> [file-pattern]
# Examples:
#   ./scripts/go-uncovered-lines.sh coverage.out                    # all uncovered lines
#   ./scripts/go-uncovered-lines.sh coverage.out catalog_service    # only matching files

PROFILE="${1:?Usage: $0 <coverprofile> [file-pattern]}"
PATTERN="${2:-}"

if [ ! -f "$PROFILE" ]; then
  echo "Coverage profile not found: $PROFILE"
  exit 1
fi

# Coverage profile format: file:startLine.startCol,endLine.endCol numStatements count
# Lines with count=0 are uncovered
grep ' 0$' "$PROFILE" | while IFS= read -r line; do
  # Parse: github.com/...file.go:startLine.col,endLine.col N 0
  file=$(echo "$line" | cut -d: -f1)
  rest=$(echo "$line" | cut -d: -f2)
  startLine=$(echo "$rest" | cut -d. -f1)
  endLine=$(echo "$rest" | cut -d, -f2 | cut -d. -f1)

  # Short filename
  short=$(basename "$file")

  # Check pattern filter
  if [ -n "$PATTERN" ] && ! echo "$short" | grep -qi "$PATTERN"; then
    continue
  fi

  # Get relative path
  relPath="${file#github.com/project-catalyst/pc-asset-hub/}"

  if [ -f "$relPath" ]; then
    code=$(sed -n "${startLine},${endLine}p" "$relPath" | head -3 | tr '\n' ' ' | sed 's/[[:space:]]\+/ /g' | cut -c1-120)
    echo "  $short:$startLine: $code"
  else
    echo "  $short:$startLine-$endLine"
  fi
done | sort -t: -k1,1 -k2 -n -u
