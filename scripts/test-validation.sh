#!/bin/bash
# Live system tests for Phase 6: Catalog Validation
# Usage: ./scripts/test-validation.sh [API_BASE_URL]
# Default: http://localhost:30080

set -uo pipefail

API_BASE="${1:-http://localhost:30080}"
META_API="$API_BASE/api/meta/v1"
DATA_API="$API_BASE/api/data/v1"

PASS=0
FAIL=0
TOTAL=0

pass() { PASS=$((PASS+1)); TOTAL=$((TOTAL+1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL+1)); TOTAL=$((TOTAL+1)); echo "  FAIL: $1 — $2"; }

header() { echo ""; echo "=== $1 ==="; }

# Helper: make API call and return HTTP status code + body
api() {
  local method="$1" path="$2" role="${3:-Admin}" body="${4:-}"
  if [ -n "$body" ]; then
    curl -s -w "\n%{http_code}" -X "$method" "$path" \
      -H "X-User-Role: $role" -H "Content-Type: application/json" -d "$body"
  else
    curl -s -w "\n%{http_code}" -X "$method" "$path" \
      -H "X-User-Role: $role" -H "Content-Type: application/json"
  fi
}

# Parse response: last line is HTTP status, rest is body
parse_response() {
  local response="$1"
  local body status
  status=$(echo "$response" | tail -1)
  body=$(echo "$response" | sed '$d')
  echo "$body"
  return 0
}

get_status() {
  echo "$1" | tail -1
}

get_body() {
  echo "$1" | sed '$d'
}

header "Setup: Use existing catalog (test1) or create test data"

# Use the existing catalog if available, otherwise create test data
# We use a unique timestamp-based catalog name to avoid collisions
TIMESTAMP=$(date +%s)
CATALOG_NAME="valtest-${TIMESTAMP}"
CREATED_ETS=""

# Check if there's an existing catalog we can use for basic tests
EXISTING_RESP=$(api GET "$DATA_API/catalogs" Admin)
EXISTING_BODY=$(get_body "$EXISTING_RESP")
EXISTING_COUNT=$(echo "$EXISTING_BODY" | jq '.items | length')

if [ "$EXISTING_COUNT" -gt 0 ]; then
  EXISTING_CATALOG=$(echo "$EXISTING_BODY" | jq -r '.items[0].name')
  EXISTING_CV_ID=$(echo "$EXISTING_BODY" | jq -r '.items[0].catalog_version_id')
  echo "  Found existing catalog: $EXISTING_CATALOG (will use for read-only tests)"
fi

# Look up system type definition version IDs for attribute creation
TD_RESP=$(api GET "$META_API/type-definitions" Admin)
TD_BODY=$(get_body "$TD_RESP")
STRING_TDV=$(echo "$TD_BODY" | jq -r '.items[] | select(.name == "string") | .latest_version_id')
if [ -z "$STRING_TDV" ] || [ "$STRING_TDV" = "null" ]; then
  echo "  ERROR: Could not find string type definition version ID"
  exit 1
fi
echo "  String TDV: $STRING_TDV"

# Create our own test entity types and catalog for mutation tests
echo "Creating test entity types..."
ET_RESP=$(api POST "$META_API/entity-types" Admin "{\"name\":\"vt-server-${TIMESTAMP}\"}")
ET_STATUS=$(get_status "$ET_RESP")
ET_BODY=$(get_body "$ET_RESP")
if [ "$ET_STATUS" = "201" ]; then
  SERVER_ET_ID=$(echo "$ET_BODY" | jq -r '.entity_type.id')
  CREATED_ETS="$SERVER_ET_ID"
else
  echo "  ERROR: Could not create entity type ($ET_STATUS)"
  exit 1
fi

# Add required attribute (using type_definition_version_id)
ATTR_RESP=$(api POST "$META_API/entity-types/$SERVER_ET_ID/attributes" Admin \
  "{\"name\":\"hostname\",\"description\":\"Server hostname\",\"type_definition_version_id\":\"$STRING_TDV\",\"required\":true}")
ATTR_STATUS=$(get_status "$ATTR_RESP")
if [ "$ATTR_STATUS" != "201" ]; then
  echo "  ERROR: Could not create required attribute hostname ($ATTR_STATUS)"
  echo "  $(get_body "$ATTR_RESP")"
  exit 1
fi

# Add optional attribute
ATTR2_RESP=$(api POST "$META_API/entity-types/$SERVER_ET_ID/attributes" Admin \
  "{\"name\":\"notes\",\"description\":\"Notes\",\"type_definition_version_id\":\"$STRING_TDV\",\"required\":false}")
ATTR2_STATUS=$(get_status "$ATTR2_RESP")
if [ "$ATTR2_STATUS" != "201" ]; then
  echo "  ERROR: Could not create optional attribute notes ($ATTR2_STATUS)"
  echo "  $(get_body "$ATTR2_RESP")"
  exit 1
fi

# Get latest version
SERVER_VERSIONS_RESP=$(api GET "$META_API/entity-types/$SERVER_ET_ID/versions" Admin)
SERVER_ETV_ID=$(get_body "$SERVER_VERSIONS_RESP" | jq -r '.items[-1].id')

echo "  Server ET ID: $SERVER_ET_ID (ETV: $SERVER_ETV_ID)"

# Create catalog version
CV_RESP=$(api POST "$META_API/catalog-versions" Admin \
  "{\"version_label\":\"val-cv-${TIMESTAMP}\",\"pins\":[{\"entity_type_version_id\":\"$SERVER_ETV_ID\"}]}")
CV_STATUS=$(get_status "$CV_RESP")
CV_BODY=$(get_body "$CV_RESP")
CV_ID=$(echo "$CV_BODY" | jq -r '.id')
echo "  Created CV: $CV_ID"

# Create catalog
CAT_RESP=$(api POST "$DATA_API/catalogs" Admin \
  "{\"name\":\"$CATALOG_NAME\",\"description\":\"Validation test catalog\",\"catalog_version_id\":\"$CV_ID\"}")
CAT_STATUS=$(get_status "$CAT_RESP")
echo "  Created catalog: $CATALOG_NAME (status=$CAT_STATUS)"

header "Test 1: Validate empty catalog (should be valid)"

RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/validate" Admin)
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
VAL_STATUS=$(echo "$BODY" | jq -r '.status')

if [ "$STATUS" = "200" ] && [ "$VAL_STATUS" = "valid" ]; then
  pass "Empty catalog validation returns valid"
else
  fail "Empty catalog validation" "status=$STATUS val_status=$VAL_STATUS"
fi

header "Test 2: RO user cannot validate (403)"

RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/validate" RO)
STATUS=$(get_status "$RESP")

if [ "$STATUS" = "403" ]; then
  pass "RO user blocked from validation (403)"
else
  fail "RO user should be blocked" "got status=$STATUS"
fi

header "Test 3: RW user can validate (200)"

RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/validate" RW)
STATUS=$(get_status "$RESP")

if [ "$STATUS" = "200" ]; then
  pass "RW user can validate (200)"
else
  fail "RW user should be allowed" "got status=$STATUS"
fi

header "Test 4: Nonexistent catalog returns 404"

RESP=$(api POST "$DATA_API/catalogs/does-not-exist/validate" Admin)
STATUS=$(get_status "$RESP")

if [ "$STATUS" = "404" ]; then
  pass "Nonexistent catalog returns 404"
else
  fail "Nonexistent catalog should return 404" "got status=$STATUS"
fi

header "Test 5: Create instance without required attr, validate → invalid"

# Create a server instance without setting hostname
INST_RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/vt-server-${TIMESTAMP}" Admin \
  '{"name":"missing-hostname-server","description":"No hostname set"}')
INST_STATUS=$(get_status "$INST_RESP")
INST_BODY=$(get_body "$INST_RESP")

if [ "$INST_STATUS" = "201" ]; then
  INST_ID=$(echo "$INST_BODY" | jq -r '.id')
  echo "  Created instance: $INST_ID"
else
  fail "Create instance" "status=$INST_STATUS"
fi

# Validate — should be invalid
RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/validate" Admin)
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
VAL_STATUS=$(echo "$BODY" | jq -r '.status')
ERROR_COUNT=$(echo "$BODY" | jq '.errors | length')

if [ "$STATUS" = "200" ] && [ "$VAL_STATUS" = "invalid" ] && [ "$ERROR_COUNT" -ge 1 ]; then
  pass "Missing required attr detected (invalid, $ERROR_COUNT error(s))"
else
  fail "Missing required attr" "status=$STATUS val_status=$VAL_STATUS errors=$ERROR_COUNT"
fi

# Check error structure
ERROR_ET=$(echo "$BODY" | jq -r '.errors[0].entity_type')
ERROR_INST=$(echo "$BODY" | jq -r '.errors[0].instance_name')
ERROR_FIELD=$(echo "$BODY" | jq -r '.errors[0].field')
ERROR_VIOL=$(echo "$BODY" | jq -r '.errors[0].violation')

if [ "$ERROR_ET" != "null" ] && [ "$ERROR_INST" != "null" ] && [ "$ERROR_FIELD" != "null" ] && [ "$ERROR_VIOL" != "null" ]; then
  pass "Error structure has entity_type, instance_name, field, violation"
else
  fail "Error structure" "entity_type=$ERROR_ET instance_name=$ERROR_INST field=$ERROR_FIELD violation=$ERROR_VIOL"
fi

header "Test 6: Catalog status persisted after validation"

RESP=$(api GET "$DATA_API/catalogs/$CATALOG_NAME" Admin)
BODY=$(get_body "$RESP")
PERSISTED_STATUS=$(echo "$BODY" | jq -r '.validation_status')

if [ "$PERSISTED_STATUS" = "invalid" ]; then
  pass "Catalog status persisted as 'invalid'"
else
  fail "Catalog status persistence" "expected=invalid got=$PERSISTED_STATUS"
fi

header "Test 7: Create instance WITH required attr, validate → valid"

# Delete the instance without hostname first
if [ -n "${INST_ID:-}" ]; then
  api DELETE "$DATA_API/catalogs/$CATALOG_NAME/vt-server-${TIMESTAMP}/$INST_ID" Admin > /dev/null 2>&1 || true
fi

# Create a server instance with hostname set
INST_RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/vt-server-${TIMESTAMP}" Admin \
  '{"name":"complete-server","description":"Has hostname","attributes":{"hostname":"web-01"}}')
INST_STATUS=$(get_status "$INST_RESP")

if [ "$INST_STATUS" = "201" ]; then
  echo "  Created instance with hostname"
else
  fail "Create instance with hostname" "status=$INST_STATUS"
fi

RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/validate" Admin)
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
VAL_STATUS=$(echo "$BODY" | jq -r '.status')

if [ "$STATUS" = "200" ] && [ "$VAL_STATUS" = "valid" ]; then
  pass "Instance with required attrs passes validation"
else
  fail "Valid instance validation" "status=$STATUS val_status=$VAL_STATUS body=$BODY"
fi

header "Test 8: Status reset to draft after data mutation"

# Update the instance
INST_LIST=$(api GET "$DATA_API/catalogs/$CATALOG_NAME/vt-server-${TIMESTAMP}" Admin)
INST_LIST_BODY=$(get_body "$INST_LIST")
FIRST_INST_ID=$(echo "$INST_LIST_BODY" | jq -r '.items[0].id')
FIRST_INST_VER=$(echo "$INST_LIST_BODY" | jq -r '.items[0].version')

api PUT "$DATA_API/catalogs/$CATALOG_NAME/vt-server-${TIMESTAMP}/$FIRST_INST_ID" Admin \
  "{\"version\":$FIRST_INST_VER,\"description\":\"Updated\"}" > /dev/null 2>&1

RESP=$(api GET "$DATA_API/catalogs/$CATALOG_NAME" Admin)
BODY=$(get_body "$RESP")
STATUS_AFTER=$(echo "$BODY" | jq -r '.validation_status')

if [ "$STATUS_AFTER" = "draft" ]; then
  pass "Status reset to draft after mutation"
else
  fail "Status reset" "expected=draft got=$STATUS_AFTER"
fi

header "Test 9: Constraint violation — string exceeding max_length → invalid"

# Create a type definition with max_length constraint
CTYPE_RESP=$(api POST "$META_API/type-definitions" Admin \
  "{\"name\":\"vt-shortstr-${TIMESTAMP}\",\"base_type\":\"string\",\"constraints\":{\"max_length\":5}}")
CTYPE_STATUS=$(get_status "$CTYPE_RESP")
CTYPE_BODY=$(get_body "$CTYPE_RESP")
CTYPE_TDV_ID=$(echo "$CTYPE_BODY" | jq -r '.latest_version_id')
echo "  Created type definition with max_length=5 (TDV: $CTYPE_TDV_ID, status=$CTYPE_STATUS)"

# Add an attribute using the constrained type definition to the server entity type
CATTR_RESP=$(api POST "$META_API/entity-types/$SERVER_ET_ID/attributes" Admin \
  "{\"name\":\"shortcode\",\"description\":\"Short code (max 5 chars)\",\"type_definition_version_id\":\"$CTYPE_TDV_ID\",\"required\":false}")
CATTR_STATUS=$(get_status "$CATTR_RESP")
echo "  Added constrained attribute 'shortcode' (status=$CATTR_STATUS)"

# Get the latest entity type version after adding the new attribute
NEW_SERVER_ETV_ID=$(get_body "$(api GET "$META_API/entity-types/$SERVER_ET_ID/versions" Admin)" | jq -r '.items[-1].id')
echo "  New server ETV: $NEW_SERVER_ETV_ID"

# Update CV pin to use the new entity type version
api PUT "$META_API/catalog-versions/$CV_ID" Admin \
  "{\"pins\":[{\"entity_type_version_id\":\"$NEW_SERVER_ETV_ID\"}]}" > /dev/null 2>&1

# Delete existing instance from Test 7 so we start clean
INST_LIST=$(api GET "$DATA_API/catalogs/$CATALOG_NAME/vt-server-${TIMESTAMP}" Admin)
INST_LIST_BODY=$(get_body "$INST_LIST")
EXISTING_INST_ID=$(echo "$INST_LIST_BODY" | jq -r '.items[0].id // empty')
if [ -n "$EXISTING_INST_ID" ]; then
  api DELETE "$DATA_API/catalogs/$CATALOG_NAME/vt-server-${TIMESTAMP}/$EXISTING_INST_ID" Admin > /dev/null 2>&1 || true
fi

# Create instance with shortcode exceeding max_length (6 chars > 5 max)
INST_RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/vt-server-${TIMESTAMP}" Admin \
  '{"name":"constraint-server","description":"Has long shortcode","attributes":{"hostname":"ok-host","shortcode":"TOOLONG"}}')
INST_STATUS=$(get_status "$INST_RESP")
if [ "$INST_STATUS" = "201" ]; then
  echo "  Created instance with oversized shortcode"
else
  echo "  Instance creation status: $INST_STATUS ($(get_body "$INST_RESP"))"
fi

# Validate — should be invalid with constraint error
RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/validate" Admin)
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
VAL_STATUS=$(echo "$BODY" | jq -r '.status')
ERROR_COUNT=$(echo "$BODY" | jq '.errors | length')

if [ "$STATUS" = "200" ] && [ "$VAL_STATUS" = "invalid" ] && [ "$ERROR_COUNT" -ge 1 ]; then
  # Check that the error mentions max_length constraint
  ERROR_VIOL=$(echo "$BODY" | jq -r '[.errors[].violation] | join(" ")')
  if echo "$ERROR_VIOL" | grep -qi "maximum length\|max.length\|exceeds"; then
    pass "Constraint violation detected: string exceeds max_length ($ERROR_COUNT error(s))"
  else
    pass "Validation returned invalid with $ERROR_COUNT error(s) (violation: $ERROR_VIOL)"
  fi
else
  fail "Constraint violation" "status=$STATUS val_status=$VAL_STATUS errors=$ERROR_COUNT body=$BODY"
fi

# Clean up the type definition
CTYPE_TD_ID=$(echo "$CTYPE_BODY" | jq -r '.id')
api DELETE "$META_API/type-definitions/$CTYPE_TD_ID" Admin > /dev/null 2>&1 || true

header "Cleanup (only removing test data created by this script)"

api DELETE "$DATA_API/catalogs/$CATALOG_NAME" Admin > /dev/null 2>&1 || true
echo "  Deleted test catalog: $CATALOG_NAME"

# Delete CV (must happen before entity type deletion since CV pins reference ETVs)
api DELETE "$META_API/catalog-versions/$CV_ID" Admin > /dev/null 2>&1 || true
echo "  Deleted test CV: $CV_ID"

# Delete entity type
api DELETE "$META_API/entity-types/$SERVER_ET_ID" Admin > /dev/null 2>&1 || true
echo "  Deleted test entity type: $SERVER_ET_ID"

header "Results"

echo ""
echo "  Total: $TOTAL"
echo "  Passed: $PASS"
echo "  Failed: $FAIL"
echo ""

if [ "$FAIL" -gt 0 ]; then
  echo "  SOME TESTS FAILED"
  exit 1
else
  echo "  ALL TESTS PASSED"
  exit 0
fi
