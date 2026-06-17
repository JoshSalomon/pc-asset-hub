#!/bin/bash
# Check that every test case ID in the detailed test plan has a corresponding test.
# Usage: scripts/check-test-plan-coverage.sh [milestone-number]
# Example: scripts/check-test-plan-coverage.sh 36

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TEST_PLAN="$PROJECT_ROOT/docs/test-plan-detailed.md"

if [[ ! -f "$TEST_PLAN" ]]; then
  echo "ERROR: Test plan not found at $TEST_PLAN"
  exit 1
fi

MILESTONE="${1:-}"
EXCLUDE_PATTERN="${2:-}"

if [[ -z "$MILESTONE" ]]; then
  echo "Usage: $0 <milestone-number> [exclude-pattern]"
  echo "Example: $0 36"
  echo "Example: $0 36 'T-36\\.(59|6[0-4]|1[23][0-9]|14[0-4])'"
  echo ""
  echo "The exclude pattern filters out test IDs that are deferred (e.g., live/system tests)."
  exit 1
fi

# Extract all test case IDs for this milestone from the test plan
# Matches T-{milestone}.{number} with optional letter suffix (e.g., T-36.23b, T-36.83a)
PLANNED=$(grep -oP "T-${MILESTONE}\.\d+[a-z]*" "$TEST_PLAN" | sort -u)

# Apply exclusion pattern if provided
if [[ -n "$EXCLUDE_PATTERN" ]]; then
  PLANNED=$(echo "$PLANNED" | grep -vP "$EXCLUDE_PATTERN")
fi

PLANNED_COUNT=$(echo "$PLANNED" | wc -l)

# Find all test case IDs referenced in test files and test scripts
IMPLEMENTED=$(grep -rhoP "T-${MILESTONE}\.\d+[a-z]*" \
  "$PROJECT_ROOT/internal/" \
  "$PROJECT_ROOT/ui/src/" \
  "$PROJECT_ROOT/scripts/" \
  --include="*_test*" --include="*.test.*" --include="test-*.sh" 2>/dev/null | sort -u)
IMPLEMENTED_COUNT=$(echo "$IMPLEMENTED" | wc -l)

# Find missing IDs
MISSING=$(comm -23 <(echo "$PLANNED") <(echo "$IMPLEMENTED"))
MISSING_COUNT=$(echo "$MISSING" | grep -c "T-" 2>/dev/null || true)
MISSING_COUNT=${MISSING_COUNT:-0}

echo "=== Test Plan Coverage: Milestone $MILESTONE ==="
echo ""
echo "  Planned test cases:     $PLANNED_COUNT"
echo "  Implemented test cases: $IMPLEMENTED_COUNT"
echo "  Missing test cases:     $MISSING_COUNT"
echo ""

if [[ "$MISSING_COUNT" -gt 0 ]]; then
  echo "MISSING TEST CASES:"
  echo "$MISSING" | while read -r id; do
    # Show the test plan description for context
    desc=$(grep -P "^\| $id " "$TEST_PLAN" | head -1 | sed 's/|/\t/g' | awk -F'\t' '{print $3}' | xargs)
    echo "  $id: $desc"
  done
  echo ""
  echo "FAIL: $MISSING_COUNT test case(s) missing. Do not proceed."
  exit 1
else
  echo "PASS: All $PLANNED_COUNT test cases have implementations."
  exit 0
fi
