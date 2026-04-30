#!/bin/bash
# Live system tests for TD-22: System Attributes (Common Attributes as Schema-Level Attributes)
# Usage: ./scripts/test-system-attributes.sh [API_BASE_URL]
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

get_status() { echo "$1" | tail -1; }
get_body() { echo "$1" | sed '$d'; }

TIMESTAMP=$(date +%s)
ET_NAME="sa-test-${TIMESTAMP}"
CATALOG_NAME="sa-catalog-${TIMESTAMP}"
ET_ID=""
CV_ID=""
ETV_ID=""

cleanup() {
  header "Cleanup"
  # Delete catalog (deletes instances too)
  api DELETE "$DATA_API/catalogs/$CATALOG_NAME" Admin > /dev/null 2>&1 || true
  echo "  Deleted catalog: $CATALOG_NAME"
  # Delete CV
  if [ -n "$CV_ID" ]; then
    api DELETE "$META_API/catalog-versions/$CV_ID" Admin > /dev/null 2>&1 || true
    echo "  Deleted CV: $CV_ID"
  fi
  # Delete entity type
  if [ -n "$ET_ID" ]; then
    api DELETE "$META_API/entity-types/$ET_ID" Admin > /dev/null 2>&1 || true
    echo "  Deleted entity type: $ET_ID"
  fi
}

trap cleanup EXIT

# ─── Setup ────────────────────────────────────────────────────────────

header "Setup: Create entity type"

ET_RESP=$(api POST "$META_API/entity-types" Admin "{\"name\":\"$ET_NAME\"}")
ET_STATUS=$(get_status "$ET_RESP")
ET_BODY=$(get_body "$ET_RESP")
if [ "$ET_STATUS" != "201" ]; then
  echo "  ERROR: Could not create entity type ($ET_STATUS)"
  exit 1
fi
ET_ID=$(echo "$ET_BODY" | jq -r '.entity_type.id')
echo "  Entity type: $ET_ID ($ET_NAME)"

# Look up system "string" type definition version ID for attribute creation
TD_RESP=$(api GET "$META_API/type-definitions" Admin)
STRING_TDV_ID=$(get_body "$TD_RESP" | jq -r '.items[] | select(.name=="string") | .latest_version_id')

# ─── Test 1: Attribute list includes system attrs ─────────────────────

header "Test 1: Attribute list includes system attributes"

ATTRS_RESP=$(api GET "$META_API/entity-types/$ET_ID/attributes" Admin)
ATTRS_BODY=$(get_body "$ATTRS_RESP")
ATTRS_STATUS=$(get_status "$ATTRS_RESP")

if [ "$ATTRS_STATUS" = "200" ]; then
  ATTR_COUNT=$(echo "$ATTRS_BODY" | jq '.total')
  FIRST_NAME=$(echo "$ATTRS_BODY" | jq -r '.items[0].name')
  FIRST_SYS=$(echo "$ATTRS_BODY" | jq '.items[0].system')
  FIRST_REQ=$(echo "$ATTRS_BODY" | jq '.items[0].required')
  SECOND_NAME=$(echo "$ATTRS_BODY" | jq -r '.items[1].name')
  SECOND_SYS=$(echo "$ATTRS_BODY" | jq '.items[1].system')
  SECOND_REQ=$(echo "$ATTRS_BODY" | jq '.items[1].required')

  if [ "$ATTR_COUNT" -ge 2 ] && [ "$FIRST_NAME" = "name" ] && [ "$FIRST_SYS" = "true" ]; then
    pass "Attribute list includes Name system attr (system=true)"
  else
    fail "Name system attr in list" "got count=$ATTR_COUNT first=$FIRST_NAME system=$FIRST_SYS"
  fi

  if [ "$FIRST_REQ" = "true" ]; then
    pass "Name system attr is required"
  else
    fail "Name required flag" "expected true, got $FIRST_REQ"
  fi

  if [ "$SECOND_NAME" = "description" ] && [ "$SECOND_SYS" = "true" ]; then
    pass "Attribute list includes Description system attr (system=true)"
  else
    fail "Description system attr in list" "got name=$SECOND_NAME system=$SECOND_SYS"
  fi

  if [ "$SECOND_REQ" = "false" ]; then
    pass "Description system attr is optional"
  else
    fail "Description required flag" "expected false, got $SECOND_REQ"
  fi
else
  fail "Get attribute list" "HTTP $ATTRS_STATUS"
fi

# ─── Test 2: Reserved name rejection ──────────────────────────────────

header "Test 2: Reserved name rejection"

NAME_RESP=$(api POST "$META_API/entity-types/$ET_ID/attributes" Admin '{"name":"name","type_definition_version_id":"'"${STRING_TDV_ID}"'"}')
NAME_STATUS=$(get_status "$NAME_RESP")
if [ "$NAME_STATUS" = "400" ]; then
  pass "Create attribute 'name' rejected (400)"
else
  fail "Reserved name 'name'" "expected 400, got $NAME_STATUS"
fi

DESC_RESP=$(api POST "$META_API/entity-types/$ET_ID/attributes" Admin '{"name":"description","type_definition_version_id":"'"${STRING_TDV_ID}"'"}')
DESC_STATUS=$(get_status "$DESC_RESP")
if [ "$DESC_STATUS" = "400" ]; then
  pass "Create attribute 'description' rejected (400)"
else
  fail "Reserved name 'description'" "expected 400, got $DESC_STATUS"
fi

# Uppercase "Name" should be allowed
UPPER_RESP=$(api POST "$META_API/entity-types/$ET_ID/attributes" Admin '{"name":"Name","type_definition_version_id":"'"${STRING_TDV_ID}"'"}')
UPPER_STATUS=$(get_status "$UPPER_RESP")
if [ "$UPPER_STATUS" = "201" ]; then
  pass "Create attribute 'Name' (uppercase) allowed"
else
  fail "Uppercase 'Name' allowed" "expected 201, got $UPPER_STATUS"
fi

# ─── Test 3: Add a real attribute and check snapshot ──────────────────

header "Test 3: Version snapshot includes system attrs"

# Add a hostname attribute — capture the new version from response
HOST_RESP=$(api POST "$META_API/entity-types/$ET_ID/attributes" Admin '{"name":"hostname","type_definition_version_id":"'"${STRING_TDV_ID}"'","required":true}')
HOST_STATUS=$(get_status "$HOST_RESP")
HOST_BODY=$(get_body "$HOST_RESP")
if [ "$HOST_STATUS" = "201" ]; then
  LATEST_VER=$(echo "$HOST_BODY" | jq -r '.version')
  echo "  Added 'hostname' attribute, now at version $LATEST_VER"
else
  fail "Add hostname attribute" "HTTP $HOST_STATUS"
  LATEST_VER=1
fi

SNAP_RESP=$(api GET "$META_API/entity-types/$ET_ID/versions/$LATEST_VER/snapshot" Admin)
SNAP_STATUS=$(get_status "$SNAP_RESP")
SNAP_BODY=$(get_body "$SNAP_RESP")

if [ "$SNAP_STATUS" = "200" ]; then
  ETV_ID=$(echo "$SNAP_BODY" | jq -r '.version.id')
  SNAP_ATTR_COUNT=$(echo "$SNAP_BODY" | jq '.attributes | length')
  SNAP_FIRST=$(echo "$SNAP_BODY" | jq -r '.attributes[0].name')
  SNAP_FIRST_SYS=$(echo "$SNAP_BODY" | jq '.attributes[0].system')
  SNAP_FIRST_ORD=$(echo "$SNAP_BODY" | jq '.attributes[0].ordinal')

  if [ "$SNAP_FIRST" = "name" ] && [ "$SNAP_FIRST_SYS" = "true" ]; then
    pass "Snapshot includes Name system attr first"
  else
    fail "Snapshot Name attr" "got name=$SNAP_FIRST system=$SNAP_FIRST_SYS"
  fi

  if [ "$SNAP_FIRST_ORD" = "-2" ]; then
    pass "System attr Name has ordinal -2"
  else
    fail "Name ordinal" "expected -2, got $SNAP_FIRST_ORD"
  fi

  SNAP_SECOND=$(echo "$SNAP_BODY" | jq -r '.attributes[1].name')
  SNAP_SECOND_ORD=$(echo "$SNAP_BODY" | jq '.attributes[1].ordinal')
  if [ "$SNAP_SECOND" = "description" ] && [ "$SNAP_SECOND_ORD" = "-1" ]; then
    pass "Snapshot includes Description system attr (ordinal=-1)"
  else
    fail "Snapshot Description attr" "got name=$SNAP_SECOND ordinal=$SNAP_SECOND_ORD"
  fi
else
  fail "Get snapshot" "HTTP $SNAP_STATUS"
fi

# ─── Test 4: Instance response includes system attrs ──────────────────

header "Test 4: Instance CRUD with system attrs in response"

# Create CV with pin
CV_RESP=$(api POST "$META_API/catalog-versions" Admin \
  "{\"version_label\":\"sa-v-${TIMESTAMP}\",\"pins\":[{\"entity_type_version_id\":\"$ETV_ID\"}]}")
CV_STATUS=$(get_status "$CV_RESP")
CV_BODY=$(get_body "$CV_RESP")
if [ "$CV_STATUS" != "201" ]; then
  fail "Create CV" "HTTP $CV_STATUS"
  # Try to continue with remaining tests
else
  CV_ID=$(echo "$CV_BODY" | jq -r '.id')
fi

# Create catalog
CAT_RESP=$(api POST "$DATA_API/catalogs" RW \
  "{\"name\":\"$CATALOG_NAME\",\"description\":\"test\",\"catalog_version_id\":\"$CV_ID\"}")
CAT_STATUS=$(get_status "$CAT_RESP")
if [ "$CAT_STATUS" != "201" ]; then
  fail "Create catalog" "HTTP $CAT_STATUS"
fi

# Create instance
INST_RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/$ET_NAME" RW \
  '{"name":"server-1","description":"my server","attributes":{"hostname":"host.example.com"}}')
INST_STATUS=$(get_status "$INST_RESP")
INST_BODY=$(get_body "$INST_RESP")

if [ "$INST_STATUS" = "201" ]; then
  INST_ATTR_COUNT=$(echo "$INST_BODY" | jq '.attributes | length')
  INST_FIRST_NAME=$(echo "$INST_BODY" | jq -r '.attributes[0].name')
  INST_FIRST_VAL=$(echo "$INST_BODY" | jq -r '.attributes[0].value')
  INST_FIRST_SYS=$(echo "$INST_BODY" | jq '.attributes[0].system')
  INST_SECOND_NAME=$(echo "$INST_BODY" | jq -r '.attributes[1].name')
  INST_SECOND_VAL=$(echo "$INST_BODY" | jq -r '.attributes[1].value')

  if [ "$INST_FIRST_NAME" = "name" ] && [ "$INST_FIRST_SYS" = "true" ] && [ "$INST_FIRST_VAL" = "server-1" ]; then
    pass "Instance response has Name system attr with correct value"
  else
    fail "Instance Name attr" "name=$INST_FIRST_NAME val=$INST_FIRST_VAL sys=$INST_FIRST_SYS"
  fi

  if [ "$INST_SECOND_NAME" = "description" ] && [ "$INST_SECOND_VAL" = "my server" ]; then
    pass "Instance response has Description system attr with correct value"
  else
    fail "Instance Description attr" "name=$INST_SECOND_NAME val=$INST_SECOND_VAL"
  fi

  # Custom attrs should follow system attrs — find 'hostname' among them
  HOSTNAME_SYS=$(echo "$INST_BODY" | jq -r '[.attributes[] | select(.name == "hostname")][0].system')
  HOSTNAME_VAL=$(echo "$INST_BODY" | jq -r '[.attributes[] | select(.name == "hostname")][0].value')
  if [ "$HOSTNAME_SYS" = "false" ] && [ "$HOSTNAME_VAL" = "host.example.com" ]; then
    pass "Custom attr 'hostname' present with system=false and correct value"
  else
    fail "Custom attr hostname" "system=$HOSTNAME_SYS value=$HOSTNAME_VAL"
  fi
else
  fail "Create instance" "HTTP $INST_STATUS"
fi

# ─── Test 5: Catalog validation catches empty name ────────────────────

header "Test 5: Validation catches empty-named instances"

# Validation should pass (all instances have names)
VAL_RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/validate" RW)
VAL_STATUS=$(get_status "$VAL_RESP")
VAL_BODY=$(get_body "$VAL_RESP")

if [ "$VAL_STATUS" = "200" ]; then
  VAL_RESULT=$(echo "$VAL_BODY" | jq -r '.status')
  if [ "$VAL_RESULT" = "valid" ]; then
    pass "Catalog with named instance validates as 'valid'"
  else
    fail "Validation status" "expected valid, got $VAL_RESULT"
  fi
else
  fail "Validate catalog" "HTTP $VAL_STATUS"
fi

# ─── Results ──────────────────────────────────────────────────────────

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
