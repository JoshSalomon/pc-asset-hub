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
    curl -s -X DELETE "${API_BASE}/api/meta/v1/type-definitions/${id}" "${HEADERS[@]}" > /dev/null 2>&1 || true
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
ETV_ID=$(echo "$ET_RES" | jq -r '.version.id')
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

ENUM_RES=$(curl -s -X POST "${API_BASE}/api/meta/v1/type-definitions" "${HEADERS[@]}" -d '{"name":"desc-test-enum","description":"Enum for testing","base_type":"enum","constraints":{"values":["a","b"]}}')
ENUM_ID=$(echo "$ENUM_RES" | jq -r '.id')
ENUM_IDS+=("$ENUM_ID")

CREATE_DESC=$(echo "$ENUM_RES" | jq -r '.description')
if [ "$CREATE_DESC" = "Enum for testing" ]; then
  pass "Enum create returns description"
else
  fail "Enum create description" "expected 'Enum for testing', got '$CREATE_DESC'"
fi

LIST_ENUM_DESC=$(curl -s "${API_BASE}/api/meta/v1/type-definitions" "${HEADERS[@]}" | jq -r '.items[] | select(.name=="desc-test-enum") | .description')
if [ "$LIST_ENUM_DESC" = "Enum for testing" ]; then
  pass "Enum list includes description"
else
  fail "Enum list description" "expected 'Enum for testing', got '$LIST_ENUM_DESC'"
fi

GET_ENUM_DESC=$(curl -s "${API_BASE}/api/meta/v1/type-definitions/${ENUM_ID}" "${HEADERS[@]}" | jq -r '.description')
if [ "$GET_ENUM_DESC" = "Enum for testing" ]; then
  pass "Enum get includes description"
else
  fail "Enum get description" "expected 'Enum for testing', got '$GET_ENUM_DESC'"
fi

# === Enum without description ===

ENUM_NO_DESC=$(curl -s -X POST "${API_BASE}/api/meta/v1/type-definitions" "${HEADERS[@]}" -d '{"name":"desc-test-enum-2","base_type":"enum","constraints":{"values":["x"]}}')
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
# === Phase 2: CV Metadata Edit (US-49) ===

echo ""
echo "=== Test: Update CV version label ==="
CV_UPDATE_RES=$(curl -s -X PUT "${API_BASE}/api/meta/v1/catalog-versions/${CV_ID}" "${HEADERS[@]}" -d '{"version_label":"renamed-cv"}')
CV_NEW_LABEL=$(echo "$CV_UPDATE_RES" | jq -r '.version_label')
if [ "$CV_NEW_LABEL" = "renamed-cv" ]; then
  pass "CV version label updated"
else
  fail "CV label update" "expected 'renamed-cv', got '$CV_NEW_LABEL'"
fi

# Rename back
curl -s -X PUT "${API_BASE}/api/meta/v1/catalog-versions/${CV_ID}" "${HEADERS[@]}" -d '{"version_label":"desc-test-cv"}' > /dev/null

echo ""
echo "=== Test: Update CV description ==="
curl -s -X PUT "${API_BASE}/api/meta/v1/catalog-versions/${CV_ID}" "${HEADERS[@]}" -d '{"description":"Updated CV desc"}' > /dev/null
CV_UPDATED_DESC=$(curl -s "${API_BASE}/api/meta/v1/catalog-versions/${CV_ID}" "${HEADERS[@]}" | jq -r '.description')
if [ "$CV_UPDATED_DESC" = "Updated CV desc" ]; then
  pass "CV description updated"
else
  fail "CV desc update" "expected 'Updated CV desc', got '$CV_UPDATED_DESC'"
fi

echo ""
echo "=== Test: CV label uniqueness enforced ==="
# Create a second CV
CV2_RES=$(curl -s -X POST "${API_BASE}/api/meta/v1/catalog-versions" "${HEADERS[@]}" -d '{"version_label":"desc-test-cv-2","description":"second"}')
CV2_ID=$(echo "$CV2_RES" | jq -r '.id')
CV_IDS+=("$CV2_ID")

DUP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X PUT "${API_BASE}/api/meta/v1/catalog-versions/${CV2_ID}" "${HEADERS[@]}" -d '{"version_label":"desc-test-cv"}')
if [ "$DUP_STATUS" = "409" ]; then
  pass "CV duplicate label returns 409"
else
  fail "CV duplicate label" "expected 409, got $DUP_STATUS"
fi

# === Phase 2: CV Pin Editing (US-52) ===

echo ""
echo "=== Test: Add pin to CV ==="
ADD_PIN_RES=$(curl -s -X POST "${API_BASE}/api/meta/v1/catalog-versions/${CV_ID}/pins" "${HEADERS[@]}" -d "{\"entity_type_version_id\":\"${ETV_ID}\"}")
ADD_PIN_ID=$(echo "$ADD_PIN_RES" | jq -r '.pin_id')
if [ -n "$ADD_PIN_ID" ] && [ "$ADD_PIN_ID" != "null" ]; then
  pass "Pin added to CV with pin_id in response"
else
  fail "Add pin" "expected pin_id in response, got '$ADD_PIN_RES'"
fi

echo ""
echo "=== Test: Duplicate pin returns 409 ==="
DUP_PIN_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${API_BASE}/api/meta/v1/catalog-versions/${CV_ID}/pins" "${HEADERS[@]}" -d "{\"entity_type_version_id\":\"${ETV_ID}\"}")
if [ "$DUP_PIN_STATUS" = "409" ]; then
  pass "Duplicate pin returns 409"
else
  fail "Duplicate pin" "expected 409, got $DUP_PIN_STATUS"
fi

echo ""
echo "=== Test: Remove pin from CV ==="
REMOVE_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "${API_BASE}/api/meta/v1/catalog-versions/${CV_ID}/pins/${ADD_PIN_ID}" "${HEADERS[@]}")
if [ "$REMOVE_STATUS" = "204" ]; then
  pass "Pin removed from CV"
else
  fail "Remove pin" "expected 204, got $REMOVE_STATUS"
fi

# === US-53: UpdatePin and duplicate entity type checks ===

echo ""
echo "=== Test: UpdatePin changes pinned version ==="
# Re-add pin so we have one to update
ADD_PIN2_RES=$(curl -s -X POST "${API_BASE}/api/meta/v1/catalog-versions/${CV_ID}/pins" "${HEADERS[@]}" -d "{\"entity_type_version_id\":\"${ETV_ID}\"}")
ADD_PIN2_ID=$(echo "$ADD_PIN2_RES" | jq -r '.pin_id')

# Create a second version of the same entity type by adding an attribute
# Look up string TDV ID
STRING_TDV_ID=$(curl -s "${API_BASE}/api/meta/v1/type-definitions" "${HEADERS[@]}" | jq -r '.items[] | select(.name=="string") | .latest_version_id')
ETV_V2_RES=$(curl -s -X POST "${API_BASE}/api/meta/v1/entity-types/${ET_ID}/attributes" "${HEADERS[@]}" \
  -d "{\"name\":\"update-test-attr\",\"type_definition_version_id\":\"${STRING_TDV_ID}\",\"description\":\"for update pin test\"}")
ETV_V2_ID=$(echo "$ETV_V2_RES" | jq -r '.id')
ETV_V2_VER=$(echo "$ETV_V2_RES" | jq -r '.version')

UPDATE_PIN_RES=$(curl -s -X PUT "${API_BASE}/api/meta/v1/catalog-versions/${CV_ID}/pins/${ADD_PIN2_ID}" "${HEADERS[@]}" \
  -d "{\"entity_type_version_id\":\"${ETV_V2_ID}\"}")
UPDATE_PIN_ETV=$(echo "$UPDATE_PIN_RES" | jq -r '.pin.entity_type_version_id')
if [ "$UPDATE_PIN_ETV" = "$ETV_V2_ID" ]; then
  pass "UpdatePin changed pinned version to V${ETV_V2_VER}"
else
  fail "UpdatePin" "expected etv=$ETV_V2_ID, got etv=$UPDATE_PIN_ETV"
fi

echo ""
echo "=== Test: UpdatePin entity type mismatch returns 400 ==="
# Create a different entity type
ET2_RES=$(curl -s -X POST "${API_BASE}/api/meta/v1/entity-types" "${HEADERS[@]}" -d '{"name":"update-pin-mismatch-et","description":"different ET"}')
ET2_ID=$(echo "$ET2_RES" | jq -r '.entity_type.id')
ET2_ETV_ID=$(echo "$ET2_RES" | jq -r '.version.id')
ET_IDS+=("$ET2_ID")

# Try to update the pin to point to a version from a different entity type
MISMATCH_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X PUT "${API_BASE}/api/meta/v1/catalog-versions/${CV_ID}/pins/${ADD_PIN2_ID}" "${HEADERS[@]}" \
  -d "{\"entity_type_version_id\":\"${ET2_ETV_ID}\"}")
if [ "$MISMATCH_STATUS" = "400" ]; then
  pass "UpdatePin entity type mismatch returns 400"
else
  fail "UpdatePin mismatch" "expected 400, got $MISMATCH_STATUS"
fi

echo ""
echo "=== Test: AddPin duplicate entity type returns 409 ==="
# Try to add a pin for a different version of the already-pinned entity type (desc-test-et)
DUP_ET_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${API_BASE}/api/meta/v1/catalog-versions/${CV_ID}/pins" "${HEADERS[@]}" \
  -d "{\"entity_type_version_id\":\"${ETV_ID}\"}")
if [ "$DUP_ET_STATUS" = "409" ]; then
  pass "AddPin duplicate entity type returns 409"
else
  fail "AddPin duplicate entity type" "expected 409, got $DUP_ET_STATUS"
fi

# Cleanup: remove pin for subsequent tests
curl -s -X DELETE "${API_BASE}/api/meta/v1/catalog-versions/${CV_ID}/pins/${ADD_PIN2_ID}" "${HEADERS[@]}" > /dev/null 2>&1 || true

# === TD-69: Pin editing stage guards ===

echo ""
echo "=== Test: TD-69 AddPin on production CV blocked ==="
# Create a dedicated CV for stage guard tests
STAGE_CV_RES=$(curl -s -X POST "${API_BASE}/api/meta/v1/catalog-versions" "${HEADERS[@]}" \
  -d "{\"version_label\":\"stage-guard-cv\",\"pins\":[{\"entity_type_version_id\":\"${ETV_ID}\"}]}")
STAGE_CV_ID=$(echo "$STAGE_CV_RES" | jq -r '.id')
CV_IDS+=("$STAGE_CV_ID")

# Promote dev → testing → production
curl -s -X POST "${API_BASE}/api/meta/v1/catalog-versions/${STAGE_CV_ID}/promote" \
  -H 'Content-Type: application/json' -H 'X-User-Role: Admin' > /dev/null
curl -s -X POST "${API_BASE}/api/meta/v1/catalog-versions/${STAGE_CV_ID}/promote" \
  -H 'Content-Type: application/json' -H 'X-User-Role: SuperAdmin' > /dev/null

# Try adding a pin as Admin on production CV — should fail
PROD_ADD_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${API_BASE}/api/meta/v1/catalog-versions/${STAGE_CV_ID}/pins" \
  -H 'Content-Type: application/json' -H 'X-User-Role: Admin' \
  -d "{\"entity_type_version_id\":\"${ET2_ETV_ID}\"}")
if [ "$PROD_ADD_STATUS" = "400" ]; then
  pass "TD-69: AddPin on production CV returns 400"
else
  fail "TD-69 production AddPin" "expected 400, got $PROD_ADD_STATUS"
fi

echo ""
echo "=== Test: TD-69 AddPin on production CV blocked even for SuperAdmin ==="
PROD_SA_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${API_BASE}/api/meta/v1/catalog-versions/${STAGE_CV_ID}/pins" \
  -H 'Content-Type: application/json' -H 'X-User-Role: SuperAdmin' \
  -d "{\"entity_type_version_id\":\"${ET2_ETV_ID}\"}")
if [ "$PROD_SA_STATUS" = "400" ]; then
  pass "TD-69: AddPin on production CV blocked for SuperAdmin too"
else
  fail "TD-69 production SuperAdmin AddPin" "expected 400, got $PROD_SA_STATUS"
fi

# Demote back to testing for testing-stage tests
curl -s -X POST "${API_BASE}/api/meta/v1/catalog-versions/${STAGE_CV_ID}/demote" \
  -H 'Content-Type: application/json' -H 'X-User-Role: SuperAdmin' \
  -d '{"target_stage":"testing"}' > /dev/null

echo ""
echo "=== Test: TD-69 AddPin on testing CV blocked for RW ==="
TESTING_RW_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${API_BASE}/api/meta/v1/catalog-versions/${STAGE_CV_ID}/pins" \
  -H 'Content-Type: application/json' -H 'X-User-Role: RW' \
  -d "{\"entity_type_version_id\":\"${ET2_ETV_ID}\"}")
if [ "$TESTING_RW_STATUS" = "400" ]; then
  pass "TD-69: AddPin on testing CV returns 400 for RW"
else
  fail "TD-69 testing RW AddPin" "expected 400, got $TESTING_RW_STATUS"
fi

echo ""
echo "=== Test: TD-69 AddPin on testing CV allowed for SuperAdmin ==="
TESTING_SA_RES=$(curl -s -X POST "${API_BASE}/api/meta/v1/catalog-versions/${STAGE_CV_ID}/pins" \
  -H 'Content-Type: application/json' -H 'X-User-Role: SuperAdmin' \
  -d "{\"entity_type_version_id\":\"${ET2_ETV_ID}\"}")
TESTING_SA_PIN=$(echo "$TESTING_SA_RES" | jq -r '.pin_id')
if [ -n "$TESTING_SA_PIN" ] && [ "$TESTING_SA_PIN" != "null" ]; then
  pass "TD-69: AddPin on testing CV allowed for SuperAdmin"
else
  fail "TD-69 testing SuperAdmin AddPin" "expected pin_id, got '$TESTING_SA_RES'"
fi

# Demote to development — cleanup
curl -s -X POST "${API_BASE}/api/meta/v1/catalog-versions/${STAGE_CV_ID}/demote" \
  -H 'Content-Type: application/json' -H 'X-User-Role: SuperAdmin' \
  -d '{"target_stage":"development"}' > /dev/null

# === Phase 2: Catalog Metadata Edit (US-50) ===

echo ""
echo "=== Test: Create catalog for metadata edit tests ==="
CAT_RES=$(curl -s -X POST "${API_BASE}/api/data/v1/catalogs" "${HEADERS[@]}" -d "{\"name\":\"desc-test-cat\",\"description\":\"initial\",\"catalog_version_id\":\"${CV_ID}\"}")
CAT_NAME=$(echo "$CAT_RES" | jq -r '.name')
CATALOG_NAMES=("$CAT_NAME")

echo ""
echo "=== Test: Update catalog description ==="
curl -s -X PUT "${API_BASE}/api/data/v1/catalogs/desc-test-cat" "${HEADERS[@]}" -d '{"description":"updated catalog desc"}' > /dev/null
CAT_DESC=$(curl -s "${API_BASE}/api/data/v1/catalogs/desc-test-cat" "${HEADERS[@]}" | jq -r '.description')
if [ "$CAT_DESC" = "updated catalog desc" ]; then
  pass "Catalog description updated"
else
  fail "Catalog desc update" "expected 'updated catalog desc', got '$CAT_DESC'"
fi

echo ""
echo "=== Test: Rename catalog ==="
RENAME_RES=$(curl -s -X PUT "${API_BASE}/api/data/v1/catalogs/desc-test-cat" "${HEADERS[@]}" -d '{"name":"desc-test-cat-renamed"}')
RENAMED_NAME=$(echo "$RENAME_RES" | jq -r '.name')
if [ "$RENAMED_NAME" = "desc-test-cat-renamed" ]; then
  pass "Catalog renamed"
  CATALOG_NAMES=("desc-test-cat-renamed")
else
  fail "Catalog rename" "expected 'desc-test-cat-renamed', got '$RENAMED_NAME'"
fi

# Old name should 404
OLD_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "${API_BASE}/api/data/v1/catalogs/desc-test-cat" "${HEADERS[@]}")
if [ "$OLD_STATUS" = "404" ]; then
  pass "Old catalog name returns 404 after rename"
else
  fail "Old name check" "expected 404, got $OLD_STATUS"
fi

# === Phase 2: Catalog Re-pinning (US-51) ===

echo ""
echo "=== Test: Re-pin catalog to different CV ==="
REPIN_RES=$(curl -s -X PUT "${API_BASE}/api/data/v1/catalogs/${CATALOG_NAMES[0]}" "${HEADERS[@]}" -d "{\"catalog_version_id\":\"${CV2_ID}\"}")
REPIN_CV=$(echo "$REPIN_RES" | jq -r '.catalog_version_id')
REPIN_STATUS=$(echo "$REPIN_RES" | jq -r '.validation_status')
if [ "$REPIN_CV" = "$CV2_ID" ] && [ "$REPIN_STATUS" = "draft" ]; then
  pass "Catalog re-pinned to new CV, status reset to draft"
else
  fail "Re-pin" "expected cv=$CV2_ID status=draft, got cv=$REPIN_CV status=$REPIN_STATUS"
fi

# Cleanup catalogs
echo ""
echo "=== Cleanup Phase 2 catalogs ==="
for name in "${CATALOG_NAMES[@]}"; do
  curl -s -X DELETE "${API_BASE}/api/data/v1/catalogs/${name}" "${HEADERS[@]}" > /dev/null 2>&1 || true
done

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
