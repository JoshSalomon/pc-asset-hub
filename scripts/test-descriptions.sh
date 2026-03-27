#!/usr/bin/env bash
#
# Live tests for description fields (TD-43, TD-45, TD-46).
# Verifies descriptions on entity types, enums, and catalog versions.
#
# Usage: scripts/test-descriptions.sh [API_BASE_URL]
#

set -euo pipefail

API_BASE="${1:-http://localhost:30080}"

PASSED=0
FAILED=0

pass() { echo "  PASS: $1"; PASSED=$((PASSED + 1)); }
fail() { echo "  FAIL: $1 — $2"; FAILED=$((FAILED + 1)); }

HEADERS=(-H 'Content-Type: application/json' -H 'X-User-Role: Admin')

# Track resources for cleanup
ET_IDS=()
ENUM_IDS=()
CV_IDS=()

cleanup() {
  echo ""
  echo "=== Cleanup ==="
  for id in "${ET_IDS[@]}"; do
    curl -s -X DELETE "${API_BASE}/api/meta/v1/entity-types/${id}" "${HEADERS[@]}" > /dev/null 2>&1 || true
  done
  for id in "${ENUM_IDS[@]}"; do
    curl -s -X DELETE "${API_BASE}/api/meta/v1/enums/${id}" "${HEADERS[@]}" > /dev/null 2>&1 || true
  done
  for id in "${CV_IDS[@]}"; do
    curl -s -X DELETE "${API_BASE}/api/meta/v1/catalog-versions/${id}" "${HEADERS[@]}" > /dev/null 2>&1 || true
  done
  echo "Done."
}
trap cleanup EXIT

echo "=== Description Fields Live Tests ==="
echo ""

# === Entity Type Description (TD-43) ===

echo "=== Test 1: Entity type description from latest version ==="

ET_RES=$(curl -s -X POST "${API_BASE}/api/meta/v1/entity-types" "${HEADERS[@]}" -d '{"name":"desc-test-et","description":"Initial description"}')
ET_ID=$(echo "$ET_RES" | jq -r '.entity_type.id')
ET_IDS+=("$ET_ID")

# List should include description
LIST_DESC=$(curl -s "${API_BASE}/api/meta/v1/entity-types" "${HEADERS[@]}" | jq -r '.items[] | select(.name=="desc-test-et") | .description')
if [ "$LIST_DESC" = "Initial description" ]; then
  pass "Entity type list includes description from latest version"
else
  fail "ET list description" "expected 'Initial description', got '$LIST_DESC'"
fi

# Get by ID should include description
GET_DESC=$(curl -s "${API_BASE}/api/meta/v1/entity-types/${ET_ID}" "${HEADERS[@]}" | jq -r '.description')
if [ "$GET_DESC" = "Initial description" ]; then
  pass "Entity type get includes description"
else
  fail "ET get description" "expected 'Initial description', got '$GET_DESC'"
fi

# === TD-46: Update entity type description ===

echo ""
echo "=== Test 2: Update entity type description (TD-46) ==="

curl -s -X PUT "${API_BASE}/api/meta/v1/entity-types/${ET_ID}" "${HEADERS[@]}" -d '{"description":"Updated description"}' > /dev/null
UPDATED_DESC=$(curl -s "${API_BASE}/api/meta/v1/entity-types/${ET_ID}" "${HEADERS[@]}" | jq -r '.description')
if [ "$UPDATED_DESC" = "Updated description" ]; then
  pass "Entity type description updated via PUT"
else
  fail "ET update description" "expected 'Updated description', got '$UPDATED_DESC'"
fi

# === Enum Description (TD-45) ===

echo ""
echo "=== Test 3: Enum description ==="

ENUM_RES=$(curl -s -X POST "${API_BASE}/api/meta/v1/enums" "${HEADERS[@]}" -d '{"name":"desc-test-enum","description":"Enum for testing","values":["a","b"]}')
ENUM_ID=$(echo "$ENUM_RES" | jq -r '.id')
ENUM_IDS+=("$ENUM_ID")

CREATE_DESC=$(echo "$ENUM_RES" | jq -r '.description')
if [ "$CREATE_DESC" = "Enum for testing" ]; then
  pass "Enum create returns description"
else
  fail "Enum create description" "expected 'Enum for testing', got '$CREATE_DESC'"
fi

LIST_ENUM_DESC=$(curl -s "${API_BASE}/api/meta/v1/enums" "${HEADERS[@]}" | jq -r '.items[] | select(.name=="desc-test-enum") | .description')
if [ "$LIST_ENUM_DESC" = "Enum for testing" ]; then
  pass "Enum list includes description"
else
  fail "Enum list description" "expected 'Enum for testing', got '$LIST_ENUM_DESC'"
fi

GET_ENUM_DESC=$(curl -s "${API_BASE}/api/meta/v1/enums/${ENUM_ID}" "${HEADERS[@]}" | jq -r '.description')
if [ "$GET_ENUM_DESC" = "Enum for testing" ]; then
  pass "Enum get includes description"
else
  fail "Enum get description" "expected 'Enum for testing', got '$GET_ENUM_DESC'"
fi

# === Enum without description ===

ENUM_NO_DESC=$(curl -s -X POST "${API_BASE}/api/meta/v1/enums" "${HEADERS[@]}" -d '{"name":"desc-test-enum-2","values":["x"]}')
ENUM_NO_DESC_ID=$(echo "$ENUM_NO_DESC" | jq -r '.id')
ENUM_IDS+=("$ENUM_NO_DESC_ID")
NO_DESC_VAL=$(echo "$ENUM_NO_DESC" | jq -r '.description')
if [ "$NO_DESC_VAL" = "" ]; then
  pass "Enum without description defaults to empty string"
else
  fail "Enum no description" "expected empty, got '$NO_DESC_VAL'"
fi

# === Catalog Version Description ===

echo ""
echo "=== Test 4: Catalog version description ==="

CV_RES=$(curl -s -X POST "${API_BASE}/api/meta/v1/catalog-versions" "${HEADERS[@]}" -d '{"version_label":"vdesc-test","description":"CV for testing"}')
CV_ID=$(echo "$CV_RES" | jq -r '.id')
CV_IDS+=("$CV_ID")

CV_CREATE_DESC=$(echo "$CV_RES" | jq -r '.description')
if [ "$CV_CREATE_DESC" = "CV for testing" ]; then
  pass "CV create returns description"
else
  fail "CV create description" "expected 'CV for testing', got '$CV_CREATE_DESC'"
fi

CV_LIST_DESC=$(curl -s "${API_BASE}/api/meta/v1/catalog-versions" "${HEADERS[@]}" | jq -r '.items[] | select(.version_label=="vdesc-test") | .description')
if [ "$CV_LIST_DESC" = "CV for testing" ]; then
  pass "CV list includes description"
else
  fail "CV list description" "expected 'CV for testing', got '$CV_LIST_DESC'"
fi

CV_GET_DESC=$(curl -s "${API_BASE}/api/meta/v1/catalog-versions/${CV_ID}" "${HEADERS[@]}" | jq -r '.description')
if [ "$CV_GET_DESC" = "CV for testing" ]; then
  pass "CV get includes description"
else
  fail "CV get description" "expected 'CV for testing', got '$CV_GET_DESC'"
fi

# === CV without description ===

CV_NO_DESC=$(curl -s -X POST "${API_BASE}/api/meta/v1/catalog-versions" "${HEADERS[@]}" -d '{"version_label":"vdesc-test-2"}')
CV_NO_DESC_ID=$(echo "$CV_NO_DESC" | jq -r '.id')
CV_IDS+=("$CV_NO_DESC_ID")
CV_NO_DESC_VAL=$(echo "$CV_NO_DESC" | jq -r '.description')
if [ "$CV_NO_DESC_VAL" = "" ]; then
  pass "CV without description defaults to empty string"
else
  fail "CV no description" "expected empty, got '$CV_NO_DESC_VAL'"
fi

# === Results ===

echo ""
echo "=== Results ==="
echo ""
echo "  Total: $((PASSED + FAILED))"
echo "  Passed: $PASSED"
echo "  Failed: $FAILED"

if [ "$FAILED" -gt 0 ]; then
  echo ""
  echo "  SOME TESTS FAILED"
  exit 1
else
  echo ""
  echo "  ALL TESTS PASSED"
fi
