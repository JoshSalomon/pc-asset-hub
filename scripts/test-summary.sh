#!/usr/bin/env bash
# Shared test summary output — sourced by all test-*.sh scripts.
# Expects: PASS (or PASSED), FAIL (or FAILED), and optionally SKIP.
# Usage: source scripts/test-summary.sh; ... ; print_summary "test-foo"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

print_summary() {
  local name="${1:-$(basename "$0" .sh)}"
  local p=${PASS:-${PASSED:-0}}
  local f=${FAIL:-${FAILED:-0}}
  local s=${SKIP:-${SKIPPED:-0}}
  local total=$((p + f + s))

  echo ""
  if [ "$f" -gt 0 ]; then
    printf "${RED}%s: Tests: %d, Passed: %d, Failed: %d${NC}\n" "$name" "$total" "$p" "$f"
    exit 1
  elif [ "$s" -gt 0 ]; then
    printf "${YELLOW}%s: Tests: %d, Passed: %d, Skipped: %d, Failed: %d${NC}\n" "$name" "$total" "$p" "$s" "$f"
    exit 0
  else
    printf "${GREEN}%s: Tests: %d, Passed: %d, Failed: %d${NC}\n" "$name" "$total" "$p" "$f"
    exit 0
  fi
}
