#!/usr/bin/env bash
# Live system tests for TD-114: Schema Evolution Safety (instance attribute migration)
#
# Usage:
#   ./scripts/test-schema-evolution.sh                          # defaults to localhost:30080
#   ./scripts/test-schema-evolution.sh http://localhost:30080    # explicit
#
# Tests:
#   1. Create entity type with attrs A, B (V1)
#   2. Remove B, add C (required) → V2
#   3. Create CV pinned to V1, catalog, instances with IAVs
#   4. dry_run UpdatePin V1→V2 → verify migration report
#   5. actual UpdatePin V1→V2 → verify IAVs remapped
#   6. GET instance → verify attribute values under V2 schema

set -uo pipefail

API_BASE="${1:-http://localhost:30080}"
META_URL="${API_BASE}/api/meta/v1"
DATA_URL="${API_BASE}/api/data/v1"

PASS=0
FAIL=0
TOTAL=0
TIMESTAMP=$(date +%s)

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

get_status() { echo "$1" | tail -1; }
get_body() { echo "$1" | sed '$d'; }

cleanup() {
  echo ""
  echo "=== Cleanup ==="
  # Delete catalog first, then CV, then entity type
  if [ -n "${CATALOG_NAME:-}" ]; then
    curl -s -X DELETE "${DATA_URL}/catalogs/${CATALOG_NAME}" -H "X-User-Role: RW" > /dev/null 2>&1 || true
  fi
  if [ -n "${CV_ID:-}" ]; then
    curl -s -X DELETE "${META_URL}/catalog-versions/${CV_ID}" -H "X-User-Role: SuperAdmin" > /dev/null 2>&1 || true
  fi
  if [ -n "${ET_ID:-}" ]; then
    curl -s -X DELETE "${META_URL}/entity-types/${ET_ID}" -H "X-User-Role: Admin" > /dev/null 2>&1 || true
  fi
  echo "Done."
}

trap cleanup EXIT

# ===================================================================
# Health check
# ===================================================================
header "Health check"
HEALTH=$(curl -s "${API_BASE}/healthz" | jq -r '.status' 2>/dev/null)
if [ "$HEALTH" = "ok" ]; then
  pass "API healthy"
else
  fail "API health check" "got: $HEALTH"
  echo "Cannot proceed without healthy API."
  exit 1
fi

# ===================================================================
# Setup: Get string type definition version ID
# ===================================================================
header "Setup: Get string type definition"
TD_LIST=$(api GET "${META_URL}/type-definitions" Admin)
TD_BODY=$(get_body "$TD_LIST")
STRING_TDV_ID=$(echo "$TD_BODY" | jq -r '.items[] | select(.base_type=="string" and .name=="string") | .latest_version_id')
INT_TDV_ID=$(echo "$TD_BODY" | jq -r '.items[] | select(.base_type=="integer" and .name=="integer") | .latest_version_id')
echo "  String TDV: $STRING_TDV_ID"
echo "  Integer TDV: $INT_TDV_ID"

# ===================================================================
# Step 1: Create entity type with attrs A, B
# ===================================================================
header "Step 1: Create entity type with attrs A, B"
ET_NAME="se-test-${TIMESTAMP}"

ET_RESP=$(api POST "${META_URL}/entity-types" Admin "{\"name\":\"${ET_NAME}\"}")
ET_BODY=$(get_body "$ET_RESP")
ET_STATUS=$(get_status "$ET_RESP")
ET_ID=$(echo "$ET_BODY" | jq -r '.entity_type.id')

if [ "$ET_STATUS" = "201" ] && [ "$ET_ID" != "null" ]; then
  pass "Created entity type: ${ET_NAME} (${ET_ID})"
else
  fail "Create entity type" "status=$ET_STATUS"
  exit 1
fi

# Add attribute A (string, not required)
api POST "${META_URL}/entity-types/${ET_ID}/attributes" Admin \
  "{\"name\":\"attr-a\",\"description\":\"Attribute A\",\"type_definition_version_id\":\"${STRING_TDV_ID}\",\"required\":false}" > /dev/null 2>&1
echo "  Added attr-a (string)"

# Add attribute B (string, not required)
api POST "${META_URL}/entity-types/${ET_ID}/attributes" Admin \
  "{\"name\":\"attr-b\",\"description\":\"Attribute B\",\"type_definition_version_id\":\"${STRING_TDV_ID}\",\"required\":false}" > /dev/null 2>&1
echo "  Added attr-b (string)"

# Get entity type versions — find the latest (V1 for our purposes)
VER_RESP=$(api GET "${META_URL}/entity-types/${ET_ID}/versions" Admin)
VER_BODY=$(get_body "$VER_RESP")
V1_ETV_ID=$(echo "$VER_BODY" | jq -r '.items[-1].id')
V1_VERSION=$(echo "$VER_BODY" | jq -r '.items[-1].version')
echo "  V1 ETV ID: $V1_ETV_ID (version $V1_VERSION)"

# ===================================================================
# Step 2: Evolve schema — remove B, add C (required)
# ===================================================================
header "Step 2: Evolve schema — remove B, add C (required)"

# Remove attribute B
DEL_RESP=$(api DELETE "${META_URL}/entity-types/${ET_ID}/attributes/attr-b" Admin)
DEL_STATUS=$(get_status "$DEL_RESP")
if [ "$DEL_STATUS" = "204" ] || [ "$DEL_STATUS" = "200" ]; then
  echo "  Removed attr-b"
else
  fail "Remove attr-b" "status=$DEL_STATUS"
fi

# Add attribute C (integer, required)
ADD_C_BODY="{\"name\":\"attr-c\",\"description\":\"Attribute C\",\"type_definition_version_id\":\"${INT_TDV_ID}\",\"required\":true}"
ADD_C_RESP=$(api POST "${META_URL}/entity-types/${ET_ID}/attributes" Admin "$ADD_C_BODY")
ADD_C_STATUS=$(get_status "$ADD_C_RESP")
if [ "$ADD_C_STATUS" = "201" ]; then
  echo "  Added attr-c (integer, required)"
else
  fail "Add attr-c" "status=$ADD_C_STATUS, body=$(get_body "$ADD_C_RESP")"
fi

# Get the latest version (V2)
VER_RESP=$(api GET "${META_URL}/entity-types/${ET_ID}/versions" Admin)
VER_BODY=$(get_body "$VER_RESP")
V2_ETV_ID=$(echo "$VER_BODY" | jq -r '.items[-1].id')
V2_VERSION=$(echo "$VER_BODY" | jq -r '.items[-1].version')
echo "  V2 ETV ID: $V2_ETV_ID (version $V2_VERSION)"

if [ "$V1_ETV_ID" != "$V2_ETV_ID" ]; then
  pass "V1 and V2 are different ETVs"
else
  fail "Schema evolution" "V1 and V2 have the same ETV ID"
  exit 1
fi

# ===================================================================
# Step 3: Create CV pinned to V1, catalog, instances
# ===================================================================
header "Step 3: Create CV, catalog, and instances"

# Create catalog version
CV_RESP=$(api POST "${META_URL}/catalog-versions" RW "{\"name\":\"se-cv-${TIMESTAMP}\",\"description\":\"Schema evolution test CV\"}")
CV_BODY=$(get_body "$CV_RESP")
CV_STATUS=$(get_status "$CV_RESP")
CV_ID=$(echo "$CV_BODY" | jq -r '.id')
echo "  Created CV: $CV_ID (status $CV_STATUS)"

# Pin to V1
PIN_RESP=$(api POST "${META_URL}/catalog-versions/${CV_ID}/pins" RW "{\"entity_type_version_id\":\"${V1_ETV_ID}\"}")
PIN_BODY=$(get_body "$PIN_RESP")
PIN_STATUS=$(get_status "$PIN_RESP")
PIN_ID=$(echo "$PIN_BODY" | jq -r '.pin_id')
echo "  Added pin to V1: $PIN_ID (status $PIN_STATUS)"

if [ "$PIN_STATUS" = "201" ] && [ "$PIN_ID" != "null" ]; then
  pass "CV pinned to V1"
else
  fail "Add pin" "status=$PIN_STATUS, body=$(get_body "$PIN_RESP")"
  exit 1
fi

# Create catalog
CATALOG_NAME="se-cat-${TIMESTAMP}"
CAT_RESP=$(api POST "${DATA_URL}/catalogs" RW "{\"name\":\"${CATALOG_NAME}\",\"catalog_version_id\":\"${CV_ID}\"}")
CAT_STATUS=$(get_status "$CAT_RESP")
echo "  Created catalog: ${CATALOG_NAME} (status $CAT_STATUS)"

# Create two instances with attributes
INST1_RESP=$(api POST "${DATA_URL}/catalogs/${CATALOG_NAME}/${ET_NAME}" RW \
  "{\"name\":\"inst-1\",\"description\":\"Instance 1\",\"attributes\":{\"attr-a\":\"alpha\",\"attr-b\":\"bravo\"}}")
INST1_BODY=$(get_body "$INST1_RESP")
INST1_STATUS=$(get_status "$INST1_RESP")
INST1_ID=$(echo "$INST1_BODY" | jq -r '.id')
echo "  Created inst-1: $INST1_ID (status $INST1_STATUS)"

INST2_RESP=$(api POST "${DATA_URL}/catalogs/${CATALOG_NAME}/${ET_NAME}" RW \
  "{\"name\":\"inst-2\",\"description\":\"Instance 2\",\"attributes\":{\"attr-a\":\"apple\",\"attr-b\":\"banana\"}}")
INST2_BODY=$(get_body "$INST2_RESP")
INST2_STATUS=$(get_status "$INST2_RESP")
INST2_ID=$(echo "$INST2_BODY" | jq -r '.id')
echo "  Created inst-2: $INST2_ID (status $INST2_STATUS)"

if [ "$INST1_STATUS" = "201" ] && [ "$INST2_STATUS" = "201" ]; then
  pass "Created 2 instances with V1 attributes"
else
  fail "Create instances" "inst1=$INST1_STATUS, inst2=$INST2_STATUS"
fi

# Verify instance 1 has attr-a and attr-b (attributes returned as array of {name,value,...})
GET1_RESP=$(api GET "${DATA_URL}/catalogs/${CATALOG_NAME}/${ET_NAME}/${INST1_ID}" RO)
GET1_BODY=$(get_body "$GET1_RESP")
ATTR_A_VAL=$(echo "$GET1_BODY" | jq -r '.attributes[] | select(.name=="attr-a") | .value')
ATTR_B_VAL=$(echo "$GET1_BODY" | jq -r '.attributes[] | select(.name=="attr-b") | .value')
if [ "$ATTR_A_VAL" = "alpha" ] && [ "$ATTR_B_VAL" = "bravo" ]; then
  pass "Instance 1 has correct V1 attribute values"
else
  fail "Instance 1 attrs" "attr-a=$ATTR_A_VAL, attr-b=$ATTR_B_VAL"
fi

# ===================================================================
# Step 4: Dry-run UpdatePin V1→V2
# ===================================================================
header "Step 4: Dry-run UpdatePin V1→V2"

DRY_RESP=$(api PUT "${META_URL}/catalog-versions/${CV_ID}/pins/${PIN_ID}?dry_run=true" RW \
  "{\"entity_type_version_id\":\"${V2_ETV_ID}\"}")
DRY_STATUS=$(get_status "$DRY_RESP")
DRY_BODY=$(get_body "$DRY_RESP")

echo "  Dry-run response (status $DRY_STATUS):"
echo "$DRY_BODY" | jq '.' 2>/dev/null || echo "$DRY_BODY"

if [ "$DRY_STATUS" = "200" ]; then
  pass "Dry-run returned 200"
else
  fail "Dry-run status" "expected 200, got $DRY_STATUS"
fi

# Verify pin NOT changed (still points to V1)
DRY_PIN_ETV=$(echo "$DRY_BODY" | jq -r '.pin.entity_type_version_id')
if [ "$DRY_PIN_ETV" = "$V1_ETV_ID" ]; then
  pass "Dry-run: pin still points to V1"
else
  fail "Dry-run pin" "expected $V1_ETV_ID, got $DRY_PIN_ETV"
fi

# Verify migration report present
HAS_MIGRATION=$(echo "$DRY_BODY" | jq 'has("migration")')
if [ "$HAS_MIGRATION" = "true" ]; then
  pass "Dry-run: migration report present"
else
  fail "Dry-run migration" "no migration field in response"
fi

# Verify affected instances = 2
AFFECTED=$(echo "$DRY_BODY" | jq '.migration.affected_instances')
if [ "$AFFECTED" = "2" ]; then
  pass "Dry-run: affected_instances = 2"
else
  fail "Dry-run affected" "expected 2, got $AFFECTED"
fi

# Verify attr-a is remapped
A_ACTION=$(echo "$DRY_BODY" | jq -r '.migration.attribute_mappings[] | select(.old_name=="attr-a") | .action')
if [ "$A_ACTION" = "remap" ]; then
  pass "Dry-run: attr-a action = remap"
else
  fail "Dry-run attr-a action" "expected remap, got $A_ACTION"
fi

# Verify attr-b → attr-c treated as rename (same ordinal position)
B_ACTION=$(echo "$DRY_BODY" | jq -r '.migration.attribute_mappings[] | select(.old_name=="attr-b") | .action')
if [ "$B_ACTION" = "remap" ]; then
  pass "Dry-run: attr-b action = remap (ordinal-based rename to attr-c)"
else
  fail "Dry-run attr-b action" "expected remap, got $B_ACTION"
fi

B_NEW_NAME=$(echo "$DRY_BODY" | jq -r '.migration.attribute_mappings[] | select(.old_name=="attr-b") | .new_name')
if [ "$B_NEW_NAME" = "attr-c" ]; then
  pass "Dry-run: attr-b remapped to attr-c"
else
  fail "Dry-run attr-b new_name" "expected attr-c, got $B_NEW_NAME"
fi

# Verify renamed warning for attr-c (ordinal-based rename detection)
HAS_RENAME_WARNING=$(echo "$DRY_BODY" | jq '[.migration.warnings[] | select(.type=="renamed" and .attribute=="attr-c")] | length')
if [ "$HAS_RENAME_WARNING" = "1" ]; then
  pass "Dry-run: renamed warning for attr-b→attr-c"
else
  fail "Dry-run warning" "missing renamed warning for attr-c"
fi

# Verify instances NOT modified (dry-run should be read-only)
GET1_AFTER_DRY=$(api GET "${DATA_URL}/catalogs/${CATALOG_NAME}/${ET_NAME}/${INST1_ID}" RO)
GET1_AFTER_BODY=$(get_body "$GET1_AFTER_DRY")
ATTR_A_AFTER=$(echo "$GET1_AFTER_BODY" | jq -r '.attributes[] | select(.name=="attr-a") | .value')
if [ "$ATTR_A_AFTER" = "alpha" ]; then
  pass "Dry-run: instance data unchanged"
else
  fail "Dry-run side effects" "attr-a changed to $ATTR_A_AFTER"
fi

# ===================================================================
# Step 5: Actual UpdatePin V1→V2
# ===================================================================
header "Step 5: Actual UpdatePin V1→V2"

REAL_RESP=$(api PUT "${META_URL}/catalog-versions/${CV_ID}/pins/${PIN_ID}" RW \
  "{\"entity_type_version_id\":\"${V2_ETV_ID}\"}")
REAL_STATUS=$(get_status "$REAL_RESP")
REAL_BODY=$(get_body "$REAL_RESP")

echo "  UpdatePin response (status $REAL_STATUS):"
echo "$REAL_BODY" | jq '.' 2>/dev/null || echo "$REAL_BODY"

if [ "$REAL_STATUS" = "200" ]; then
  pass "UpdatePin returned 200"
else
  fail "UpdatePin status" "expected 200, got $REAL_STATUS"
fi

# Verify pin now points to V2
REAL_PIN_ETV=$(echo "$REAL_BODY" | jq -r '.pin.entity_type_version_id')
if [ "$REAL_PIN_ETV" = "$V2_ETV_ID" ]; then
  pass "Pin now points to V2"
else
  fail "Pin update" "expected $V2_ETV_ID, got $REAL_PIN_ETV"
fi

# Verify migration report matches dry-run
REAL_AFFECTED=$(echo "$REAL_BODY" | jq '.migration.affected_instances')
if [ "$REAL_AFFECTED" = "2" ]; then
  pass "Actual migration: affected_instances = 2"
else
  fail "Actual migration affected" "expected 2, got $REAL_AFFECTED"
fi

# ===================================================================
# Step 6: Verify IAVs migrated — GET instances
# ===================================================================
header "Step 6: Verify instance attributes after migration"

# Instance 1: attr-a should still have value "alpha", attr-b should be gone
GET1_FINAL=$(api GET "${DATA_URL}/catalogs/${CATALOG_NAME}/${ET_NAME}/${INST1_ID}" RO)
GET1_FINAL_BODY=$(get_body "$GET1_FINAL")
GET1_FINAL_STATUS=$(get_status "$GET1_FINAL")

echo "  Instance 1 after migration (status $GET1_FINAL_STATUS):"
echo "$GET1_FINAL_BODY" | jq '.attributes' 2>/dev/null || echo "$GET1_FINAL_BODY"

FINAL_A=$(echo "$GET1_FINAL_BODY" | jq -r '.attributes[] | select(.name=="attr-a") | .value')
if [ "$FINAL_A" = "alpha" ]; then
  pass "Instance 1: attr-a preserved with value 'alpha'"
else
  fail "Instance 1 attr-a" "expected 'alpha', got '$FINAL_A'"
fi

# attr-b should no longer appear (orphaned IAVs stay in DB but aren't returned for V2 schema)
FINAL_B=$(echo "$GET1_FINAL_BODY" | jq -r '[.attributes[] | select(.name=="attr-b")] | length')
if [ "$FINAL_B" = "0" ]; then
  pass "Instance 1: attr-b no longer in V2 attributes"
else
  fail "Instance 1 attr-b" "expected absent, found $FINAL_B matches"
fi

# Instance 2: verify similarly
GET2_FINAL=$(api GET "${DATA_URL}/catalogs/${CATALOG_NAME}/${ET_NAME}/${INST2_ID}" RO)
GET2_FINAL_BODY=$(get_body "$GET2_FINAL")

FINAL2_A=$(echo "$GET2_FINAL_BODY" | jq -r '.attributes[] | select(.name=="attr-a") | .value')
if [ "$FINAL2_A" = "apple" ]; then
  pass "Instance 2: attr-a preserved with value 'apple'"
else
  fail "Instance 2 attr-a" "expected 'apple', got '$FINAL2_A'"
fi

# ===================================================================
# Summary
# ===================================================================
echo ""
echo "============================================"
echo "  Schema Evolution Tests: $PASS passed, $FAIL failed (of $TOTAL)"
echo "============================================"

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
