#!/bin/bash
# Live system tests for type constraint validation (TD-90, TD-91, TD-92)
# Usage: ./scripts/test-type-constraints.sh [API_BASE_URL]
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

get_status() {
  echo "$1" | tail -1
}

get_body() {
  echo "$1" | sed '$d'
}

TIMESTAMP=$(date +%s)

# ===================================================================
# Setup: Create type definitions with constraints
# ===================================================================

header "Setup: Create type definitions with constraints"

# 1. String with max_length=5
TD_RESP=$(api POST "$META_API/type-definitions" Admin \
  "{\"name\":\"tc-short-str-${TIMESTAMP}\",\"base_type\":\"string\",\"constraints\":{\"max_length\":5}}")
TD_STATUS=$(get_status "$TD_RESP")
TD_BODY=$(get_body "$TD_RESP")
SHORT_STR_TDV_ID=$(echo "$TD_BODY" | jq -r '.latest_version_id')
SHORT_STR_TD_ID=$(echo "$TD_BODY" | jq -r '.id')
echo "  Created short-str type (max_length=5): $SHORT_STR_TD_ID (TDV: $SHORT_STR_TDV_ID)"

# 2. String with pattern=^[a-z]+$
TD_RESP=$(api POST "$META_API/type-definitions" Admin \
  "{\"name\":\"tc-lowercase-${TIMESTAMP}\",\"base_type\":\"string\",\"constraints\":{\"pattern\":\"^[a-z]+$\"}}")
TD_BODY=$(get_body "$TD_RESP")
LOWER_TDV_ID=$(echo "$TD_BODY" | jq -r '.latest_version_id')
LOWER_TD_ID=$(echo "$TD_BODY" | jq -r '.id')
echo "  Created lowercase type (pattern=^[a-z]+$): $LOWER_TD_ID"

# 3. Integer with min=0, max=100
TD_RESP=$(api POST "$META_API/type-definitions" Admin \
  "{\"name\":\"tc-percent-${TIMESTAMP}\",\"base_type\":\"integer\",\"constraints\":{\"min\":0,\"max\":100}}")
TD_BODY=$(get_body "$TD_RESP")
PERCENT_TDV_ID=$(echo "$TD_BODY" | jq -r '.latest_version_id')
PERCENT_TD_ID=$(echo "$TD_BODY" | jq -r '.id')
echo "  Created percent type (min=0, max=100): $PERCENT_TD_ID"

# 4. Boolean type (system)
TD_LIST_RESP=$(api GET "$META_API/type-definitions" Admin)
TD_LIST_BODY=$(get_body "$TD_LIST_RESP")
BOOL_TDV_ID=$(echo "$TD_LIST_BODY" | jq -r '.items[] | select(.base_type=="boolean") | .latest_version_id')
echo "  Boolean TDV: $BOOL_TDV_ID"

# 5. URL type (system)
URL_TDV_ID=$(echo "$TD_LIST_BODY" | jq -r '.items[] | select(.base_type=="url") | .latest_version_id')
echo "  URL TDV: $URL_TDV_ID"

# ===================================================================
# Setup: Create entity type with constrained attributes
# ===================================================================

header "Setup: Create entity type with constrained attributes"

ET_RESP=$(api POST "$META_API/entity-types" Admin \
  "{\"name\":\"tc-server-${TIMESTAMP}\"}")
ET_BODY=$(get_body "$ET_RESP")
ET_ID=$(echo "$ET_BODY" | jq -r '.entity_type.id')
echo "  Created entity type: $ET_ID"

# Add attributes with specific type definitions
api POST "$META_API/entity-types/$ET_ID/attributes" Admin \
  "{\"name\":\"code\",\"description\":\"Short code\",\"type_definition_version_id\":\"$SHORT_STR_TDV_ID\",\"required\":false}" > /dev/null 2>&1

api POST "$META_API/entity-types/$ET_ID/attributes" Admin \
  "{\"name\":\"tag\",\"description\":\"Lowercase tag\",\"type_definition_version_id\":\"$LOWER_TDV_ID\",\"required\":false}" > /dev/null 2>&1

api POST "$META_API/entity-types/$ET_ID/attributes" Admin \
  "{\"name\":\"score\",\"description\":\"Percentage\",\"type_definition_version_id\":\"$PERCENT_TDV_ID\",\"required\":false}" > /dev/null 2>&1

api POST "$META_API/entity-types/$ET_ID/attributes" Admin \
  "{\"name\":\"enabled\",\"description\":\"Is enabled\",\"type_definition_version_id\":\"$BOOL_TDV_ID\",\"required\":false}" > /dev/null 2>&1

api POST "$META_API/entity-types/$ET_ID/attributes" Admin \
  "{\"name\":\"homepage\",\"description\":\"Homepage URL\",\"type_definition_version_id\":\"$URL_TDV_ID\",\"required\":false}" > /dev/null 2>&1

echo "  Added 5 constrained attributes"

# Get latest ETV
ETV_RESP=$(api GET "$META_API/entity-types/$ET_ID/versions" Admin)
ETV_ID=$(get_body "$ETV_RESP" | jq -r '.items[-1].id')
echo "  ETV: $ETV_ID"

# ===================================================================
# Setup: Create catalog version and catalog
# ===================================================================

header "Setup: Create catalog and catalog version"

CV_RESP=$(api POST "$META_API/catalog-versions" Admin \
  "{\"version_label\":\"tc-cv-${TIMESTAMP}\",\"pins\":[{\"entity_type_version_id\":\"$ETV_ID\"}]}")
CV_BODY=$(get_body "$CV_RESP")
CV_ID=$(echo "$CV_BODY" | jq -r '.id')
echo "  Created CV: $CV_ID"

CATALOG_NAME="tc-catalog-${TIMESTAMP}"
api POST "$DATA_API/catalogs" Admin \
  "{\"name\":\"$CATALOG_NAME\",\"description\":\"Type constraint tests\",\"catalog_version_id\":\"$CV_ID\"}" > /dev/null 2>&1
echo "  Created catalog: $CATALOG_NAME"

ET_NAME="tc-server-${TIMESTAMP}"

# ===================================================================
# Test 1: T-31.162 — Valid values pass constraint validation
# ===================================================================

header "Test 1: T-31.162 — Valid values pass constraint validation"

INST_RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/$ET_NAME" Admin \
  '{"name":"valid-server","description":"All valid values","attributes":{"code":"abcde","tag":"hello","score":50,"enabled":"true","homepage":"https://example.com"}}')
INST_STATUS=$(get_status "$INST_RESP")

if [ "$INST_STATUS" = "201" ]; then
  echo "  Created instance with valid values"
else
  fail "Create valid instance" "status=$INST_STATUS"
fi

RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/validate" Admin)
BODY=$(get_body "$RESP")
VAL_STATUS=$(echo "$BODY" | jq -r '.status')

if [ "$VAL_STATUS" = "valid" ]; then
  pass "Valid values pass constraint validation"
else
  fail "Valid values should pass" "status=$VAL_STATUS errors=$(echo "$BODY" | jq '.errors')"
fi

# ===================================================================
# Test 2: T-31.163 — String exceeding max_length → invalid
# ===================================================================

header "Test 2: T-31.163 — String exceeding max_length → constraint error"

INST_RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/$ET_NAME" Admin \
  '{"name":"long-code-server","description":"Code too long","attributes":{"code":"toolong"}}')
INST_STATUS=$(get_status "$INST_RESP")

if [ "$INST_STATUS" = "201" ]; then
  echo "  Created instance with long code value"
fi

RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/validate" Admin)
BODY=$(get_body "$RESP")
VAL_STATUS=$(echo "$BODY" | jq -r '.status')
ERRORS=$(echo "$BODY" | jq -r '.errors[] | select(.field=="code") | .violation')

if [ "$VAL_STATUS" = "invalid" ] && echo "$ERRORS" | grep -q "maximum length"; then
  pass "String exceeding max_length detected as constraint error"
else
  fail "max_length violation" "status=$VAL_STATUS errors=$ERRORS"
fi

# ===================================================================
# Test 3: String not matching pattern → invalid
# ===================================================================

header "Test 3: String pattern mismatch → constraint error"

INST_RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/$ET_NAME" Admin \
  '{"name":"bad-tag-server","description":"Tag has uppercase","attributes":{"tag":"ABC123"}}')
INST_STATUS=$(get_status "$INST_RESP")

if [ "$INST_STATUS" = "201" ]; then
  echo "  Created instance with uppercase tag"
fi

RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/validate" Admin)
BODY=$(get_body "$RESP")
ERRORS=$(echo "$BODY" | jq -r '.errors[] | select(.field=="tag") | .violation')

if echo "$ERRORS" | grep -q "pattern"; then
  pass "String pattern mismatch detected"
else
  fail "Pattern violation" "errors=$ERRORS"
fi

# ===================================================================
# Test 4: Integer below min → invalid
# ===================================================================

header "Test 4: Integer below min → constraint error"

INST_RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/$ET_NAME" Admin \
  '{"name":"low-score-server","description":"Score below zero","attributes":{"score":-10}}')

RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/validate" Admin)
BODY=$(get_body "$RESP")
ERRORS=$(echo "$BODY" | jq -r '.errors[] | select(.field=="score") | .violation')

if echo "$ERRORS" | grep -q "below minimum"; then
  pass "Integer below min detected"
else
  fail "Min violation" "errors=$ERRORS"
fi

# ===================================================================
# Test 5: Integer above max → invalid
# ===================================================================

header "Test 5: Integer above max → constraint error"

INST_RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/$ET_NAME" Admin \
  '{"name":"high-score-server","description":"Score above 100","attributes":{"score":150}}')

RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/validate" Admin)
BODY=$(get_body "$RESP")
ERRORS=$(echo "$BODY" | jq -r '.errors[] | select(.field=="score") | .violation')

if echo "$ERRORS" | grep -q "above maximum"; then
  pass "Integer above max detected"
else
  fail "Max violation" "errors=$ERRORS"
fi

# ===================================================================
# Test 6: Boolean invalid format → invalid
# ===================================================================

header "Test 6: Boolean invalid format → constraint error"

INST_RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/$ET_NAME" Admin \
  '{"name":"bad-bool-server","description":"Boolean not true/false","attributes":{"enabled":"yes"}}')

RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/validate" Admin)
BODY=$(get_body "$RESP")
ERRORS=$(echo "$BODY" | jq -r '.errors[] | select(.field=="enabled") | .violation')

if echo "$ERRORS" | grep -q '"true" or "false"'; then
  pass "Boolean invalid format detected"
else
  fail "Boolean format violation" "errors=$ERRORS"
fi

# ===================================================================
# Test 7: URL invalid format → invalid
# ===================================================================

header "Test 7: URL invalid format → constraint error"

INST_RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/$ET_NAME" Admin \
  '{"name":"bad-url-server","description":"Invalid URL","attributes":{"homepage":"not a url"}}')

RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/validate" Admin)
BODY=$(get_body "$RESP")
ERRORS=$(echo "$BODY" | jq -r '.errors[] | select(.field=="homepage") | .violation')

if echo "$ERRORS" | grep -q "invalid URL"; then
  pass "URL invalid format detected"
else
  fail "URL format violation" "errors=$ERRORS"
fi

# ===================================================================
# Test 8: T-31.154 — Validation uses pinned type version constraints
# ===================================================================

header "Test 8: T-31.154 — Constraint from pinned type version is enforced"

# Update the type definition to relax max_length to 100 (creates version 2)
api PUT "$META_API/type-definitions/$SHORT_STR_TD_ID" Admin \
  "{\"description\":\"Relaxed\",\"constraints\":{\"max_length\":100}}" > /dev/null 2>&1
echo "  Updated short-str type to max_length=100 (version 2)"

# Validate again — the catalog is STILL pinned to version 1 (max_length=5)
RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/validate" Admin)
BODY=$(get_body "$RESP")
ERRORS=$(echo "$BODY" | jq -r '.errors[] | select(.field=="code") | .violation')

if echo "$ERRORS" | grep -q "maximum length"; then
  pass "Validation uses pinned type version (v1, max_length=5), not latest (v2, max_length=100)"
else
  fail "Pinned type version" "expected max_length=5 violation, got: $ERRORS"
fi

# ===================================================================
# Cleanup
# ===================================================================

header "Cleanup"

api DELETE "$DATA_API/catalogs/$CATALOG_NAME" Admin > /dev/null 2>&1 || true
echo "  Deleted catalog: $CATALOG_NAME"

api DELETE "$META_API/catalog-versions/$CV_ID" Admin > /dev/null 2>&1 || true
echo "  Deleted CV: $CV_ID"

api DELETE "$META_API/entity-types/$ET_ID" Admin > /dev/null 2>&1 || true
echo "  Deleted entity type: $ET_ID"

api DELETE "$META_API/type-definitions/$SHORT_STR_TD_ID" Admin > /dev/null 2>&1 || true
api DELETE "$META_API/type-definitions/$LOWER_TD_ID" Admin > /dev/null 2>&1 || true
api DELETE "$META_API/type-definitions/$PERCENT_TD_ID" Admin > /dev/null 2>&1 || true
echo "  Deleted custom type definitions"

# ===================================================================
# Results
# ===================================================================

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
