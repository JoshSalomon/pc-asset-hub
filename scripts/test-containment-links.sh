#!/usr/bin/env bash
# Live system tests for Phase 3: Containment & Association Links
#
# Usage:
#   ./scripts/test-containment-links.sh                          # defaults to localhost:30080
#   ./scripts/test-containment-links.sh http://localhost:30080    # explicit local
#   ./scripts/test-containment-links.sh https://api.ocp.example.com  # remote OCP
#
# Prerequisites:
#   - API server running and healthy
#   - jq installed
#   - curl installed

set -euo pipefail

API_BASE="${1:-http://localhost:30080}"
META_URL="${API_BASE}/api/meta/v1"
DATA_URL="${API_BASE}/api/data/v1"

PASS=0
FAIL=0
CLEANUP_IDS=()
PREFIX="tst"  # common prefix for all test entities
SUFFIX=$(date +%s)  # unique suffix to avoid name conflicts

# --- Helpers ---

pass() { echo "  PASS: $1"; PASS=$((PASS + 1)); }
fail() { echo "  FAIL: $1 — $2"; FAIL=$((FAIL + 1)); }

http_code() {
  curl -s -o /dev/null -w "%{http_code}" "$@"
}

json_post() {
  curl -s -X POST "$1" -H 'Content-Type: application/json' -H "X-User-Role: $2" -d "$3"
}

json_get() {
  curl -s "$1" -H "X-User-Role: ${2:-RO}"
}

json_delete() {
  curl -s -X DELETE "$1" -H "X-User-Role: ${2:-RW}"
}

cleanup() {
  echo ""
  echo "=== Cleanup ==="
  # Delete in dependency order: catalogs → CVs → entity types
  for name in "${CLEANUP_IDS[@]}"; do
    json_delete "${DATA_URL}/catalogs/${name}" "RW" > /dev/null 2>&1 || true
  done
  if [ -n "${CV:-}" ] && [ "${CV}" != "null" ]; then
    json_delete "${META_URL}/catalog-versions/${CV}" "SuperAdmin" > /dev/null 2>&1 || true
  fi
  # Entity types must be deleted after CV (CV pins reference them).
  # Delete in reverse order: model first (no associations), then tool, then server.
  for et in "${MODEL_ET:-}" "${TOOL_ET:-}" "${SERVER_ET:-}"; do
    if [ -n "$et" ] && [ "$et" != "null" ]; then
      json_delete "${META_URL}/entity-types/${et}" "Admin" > /dev/null 2>&1 || true
    fi
  done
  echo "Done."
}

trap cleanup EXIT

# --- Setup: Create entity types, associations, CV, catalog ---

echo "=== Setup ==="

# Create entity types with unique names
SERVER_NAME="${PREFIX}---srv-${SUFFIX}"
TOOL_NAME="${PREFIX}---tool-${SUFFIX}"
MODEL_NAME="${PREFIX}---mdl-${SUFFIX}"

SERVER_ET=$(json_post "${META_URL}/entity-types" Admin "{\"name\":\"${SERVER_NAME}\",\"description\":\"server\"}" | jq -r '.entity_type.id')
TOOL_ET=$(json_post "${META_URL}/entity-types" Admin "{\"name\":\"${TOOL_NAME}\",\"description\":\"tool\"}" | jq -r '.entity_type.id')
MODEL_ET=$(json_post "${META_URL}/entity-types" Admin "{\"name\":\"${MODEL_NAME}\",\"description\":\"model\"}" | jq -r '.entity_type.id')
echo "  Entity types: server=${SERVER_ET}, tool=${TOOL_ET}, model=${MODEL_ET}"

# Add containment: server contains tool
json_post "${META_URL}/entity-types/${SERVER_ET}/associations" Admin \
  "{\"target_entity_type_id\":\"${TOOL_ET}\",\"type\":\"containment\",\"name\":\"tools\"}" > /dev/null

# Add directional: server uses model
json_post "${META_URL}/entity-types/${SERVER_ET}/associations" Admin \
  "{\"target_entity_type_id\":\"${MODEL_ET}\",\"type\":\"directional\",\"name\":\"uses-model\"}" > /dev/null

# Get latest versions
SERVER_ETV=$(json_get "${META_URL}/entity-types/${SERVER_ET}/versions" Admin | jq -r '.items[-1].id')
TOOL_ETV=$(json_get "${META_URL}/entity-types/${TOOL_ET}/versions" Admin | jq -r '.items[-1].id')
MODEL_ETV=$(json_get "${META_URL}/entity-types/${MODEL_ET}/versions" Admin | jq -r '.items[-1].id')
echo "  Versions: server=${SERVER_ETV}, tool=${TOOL_ETV}, model=${MODEL_ETV}"

# Create CV with all three pinned
CV=$(json_post "${META_URL}/catalog-versions" Admin \
  "{\"version_label\":\"${PREFIX}---cv-${SUFFIX}\",\"pins\":[{\"entity_type_version_id\":\"${SERVER_ETV}\"},{\"entity_type_version_id\":\"${TOOL_ETV}\"},{\"entity_type_version_id\":\"${MODEL_ETV}\"}]}" | jq -r '.id')
echo "  CV: ${CV}"

# Create catalog
CATALOG_NAME="${PREFIX}---cat-${SUFFIX}"
json_post "${DATA_URL}/catalogs" RW \
  "{\"name\":\"${CATALOG_NAME}\",\"description\":\"Phase 3 test\",\"catalog_version_id\":\"${CV}\"}" > /dev/null
CLEANUP_IDS+=("${CATALOG_NAME}")
echo "  Catalog: ${CATALOG_NAME}"
echo ""

# --- Test 1: Create top-level instances ---

echo "=== Test: Create top-level instances ==="

PARENT=$(json_post "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}" RW \
  '{"name":"srv-1","description":"test server"}' | jq -r '.id')
if [ -n "$PARENT" ] && [ "$PARENT" != "null" ]; then
  pass "Create server instance (id=${PARENT})"
else
  fail "Create server instance" "no id returned"
fi

MODEL_INST=$(json_post "${DATA_URL}/catalogs/${CATALOG_NAME}/${MODEL_NAME}" RW \
  '{"name":"gpt-4","description":"a model"}' | jq -r '.id')
if [ -n "$MODEL_INST" ] && [ "$MODEL_INST" != "null" ]; then
  pass "Create model instance (id=${MODEL_INST})"
else
  fail "Create model instance" "no id returned"
fi

# --- Test 2: Create contained instance ---

echo ""
echo "=== Test: Containment ==="

CHILD_RESP=$(json_post "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}/${PARENT}/${TOOL_NAME}" RW \
  '{"name":"my-tool","description":"contained tool"}')
CHILD_ID=$(echo "$CHILD_RESP" | jq -r '.id')
CHILD_PARENT=$(echo "$CHILD_RESP" | jq -r '.parent_instance_id')

if [ "$CHILD_PARENT" = "$PARENT" ]; then
  pass "Create contained instance (parent_instance_id matches)"
else
  fail "Create contained instance" "parent_instance_id=${CHILD_PARENT}, expected=${PARENT}"
fi

# Create second contained instance
json_post "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}/${PARENT}/${TOOL_NAME}" RW \
  '{"name":"tool-2","description":"another tool"}' > /dev/null

# List contained instances
LIST_RESP=$(json_get "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}/${PARENT}/${TOOL_NAME}")
LIST_COUNT=$(echo "$LIST_RESP" | jq '.items | length')
if [ "$LIST_COUNT" = "2" ]; then
  pass "List contained instances (count=2)"
else
  fail "List contained instances" "count=${LIST_COUNT}, expected=2"
fi

# --- Test 3: Containment validation ---

echo ""
echo "=== Test: Containment validation ==="

# No containment relationship: server does not contain model
CODE=$(http_code -X POST "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}/${PARENT}/${MODEL_NAME}" \
  -H 'Content-Type: application/json' -H 'X-User-Role: RW' -d '{"name":"bad-child"}')
if [ "$CODE" = "400" ]; then
  pass "Reject non-containment child type (400)"
else
  fail "Reject non-containment child type" "got ${CODE}, expected 400"
fi

# Nonexistent parent
CODE=$(http_code -X POST "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}/nonexistent-id/${TOOL_NAME}" \
  -H 'Content-Type: application/json' -H 'X-User-Role: RW' -d '{"name":"orphan"}')
if [ "$CODE" = "404" ]; then
  pass "Reject nonexistent parent (404)"
else
  fail "Reject nonexistent parent" "got ${CODE}, expected 404"
fi

# RO cannot create contained
CODE=$(http_code -X POST "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}/${PARENT}/${TOOL_NAME}" \
  -H 'Content-Type: application/json' -H 'X-User-Role: RO' -d '{"name":"ro-child"}')
if [ "$CODE" = "403" ]; then
  pass "RO cannot create contained instance (403)"
else
  fail "RO cannot create contained instance" "got ${CODE}, expected 403"
fi

# --- Test 4: Association links ---

echo ""
echo "=== Test: Association links ==="

LINK_RESP=$(json_post "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}/${PARENT}/links" RW \
  "{\"target_instance_id\":\"${MODEL_INST}\",\"association_name\":\"uses-model\"}")
LINK_ID=$(echo "$LINK_RESP" | jq -r '.id')
LINK_TARGET=$(echo "$LINK_RESP" | jq -r '.target_instance_id')

if [ "$LINK_TARGET" = "$MODEL_INST" ]; then
  pass "Create association link (target matches)"
else
  fail "Create association link" "target=${LINK_TARGET}, expected=${MODEL_INST}"
fi

# Duplicate link prevention
DUP_CODE=$(http_code -X POST "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}/${PARENT}/links" \
  -H 'Content-Type: application/json' -H 'X-User-Role: RW' \
  -d "{\"target_instance_id\":\"${MODEL_INST}\",\"association_name\":\"uses-model\"}")
if [ "$DUP_CODE" = "409" ]; then
  pass "Reject duplicate link (409)"
else
  fail "Reject duplicate link" "got ${DUP_CODE}, expected 409"
fi

# RO cannot create link
CODE=$(http_code -X POST "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}/${PARENT}/links" \
  -H 'Content-Type: application/json' -H 'X-User-Role: RO' \
  -d "{\"target_instance_id\":\"${MODEL_INST}\",\"association_name\":\"uses-model\"}")
if [ "$CODE" = "403" ]; then
  pass "RO cannot create link (403)"
else
  fail "RO cannot create link" "got ${CODE}, expected 403"
fi

# --- Test 5: Forward references ---

echo ""
echo "=== Test: Forward references ==="

FWD_RESP=$(json_get "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}/${PARENT}/references")
FWD_COUNT=$(echo "$FWD_RESP" | jq 'length')
FWD_NAME=$(echo "$FWD_RESP" | jq -r '.[0].association_name')
FWD_TARGET=$(echo "$FWD_RESP" | jq -r '.[0].instance_name')
FWD_TYPE=$(echo "$FWD_RESP" | jq -r '.[0].entity_type_name')

if [ "$FWD_COUNT" = "1" ] && [ "$FWD_NAME" = "uses-model" ]; then
  pass "Forward references (count=1, assoc=uses-model)"
else
  fail "Forward references" "count=${FWD_COUNT}, assoc=${FWD_NAME}"
fi

if [ "$FWD_TARGET" = "gpt-4" ] && [ "$FWD_TYPE" = "${MODEL_NAME}" ]; then
  pass "Forward ref resolved target (name=gpt-4, type=${MODEL_NAME})"
else
  fail "Forward ref resolved target" "name=${FWD_TARGET}, type=${FWD_TYPE}"
fi

# --- Test 6: Reverse references ---

echo ""
echo "=== Test: Reverse references ==="

REV_RESP=$(json_get "${DATA_URL}/catalogs/${CATALOG_NAME}/${MODEL_NAME}/${MODEL_INST}/referenced-by")
REV_COUNT=$(echo "$REV_RESP" | jq 'length')
REV_SOURCE=$(echo "$REV_RESP" | jq -r '.[0].instance_name')

if [ "$REV_COUNT" = "1" ] && [ "$REV_SOURCE" = "srv-1" ]; then
  pass "Reverse references (count=1, source=srv-1)"
else
  fail "Reverse references" "count=${REV_COUNT}, source=${REV_SOURCE}"
fi

# --- Test 7: Delete link ---

echo ""
echo "=== Test: Delete link ==="

DEL_CODE=$(http_code -X DELETE "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}/${PARENT}/links/${LINK_ID}" \
  -H 'X-User-Role: RW')
if [ "$DEL_CODE" = "204" ]; then
  pass "Delete link (204)"
else
  fail "Delete link" "got ${DEL_CODE}, expected 204"
fi

# Verify forward refs empty after delete
FWD_AFTER=$(json_get "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}/${PARENT}/references")
FWD_AFTER_COUNT=$(echo "$FWD_AFTER" | jq 'length')
if [ "$FWD_AFTER_COUNT" = "0" ]; then
  pass "Forward refs empty after delete"
else
  fail "Forward refs after delete" "count=${FWD_AFTER_COUNT}, expected 0"
fi

# --- Test 8: Cascade delete cleans up links ---

echo ""
echo "=== Test: Cascade delete with links ==="

# Create a new link
json_post "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}/${PARENT}/links" RW \
  "{\"target_instance_id\":\"${MODEL_INST}\",\"association_name\":\"uses-model\"}" > /dev/null

# Verify link exists
PRE_DEL=$(json_get "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}/${PARENT}/references" | jq 'length')
if [ "$PRE_DEL" = "1" ]; then
  pass "Link exists before cascade delete"
else
  fail "Link before cascade delete" "count=${PRE_DEL}"
fi

# Delete the server instance (should cascade delete children + links)
DEL_CODE=$(http_code -X DELETE "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}/${PARENT}" \
  -H 'X-User-Role: RW')
if [ "$DEL_CODE" = "204" ]; then
  pass "Cascade delete server instance (204)"
else
  fail "Cascade delete" "got ${DEL_CODE}, expected 204"
fi

# Verify model's reverse refs are now empty (link was cleaned up)
REV_AFTER=$(json_get "${DATA_URL}/catalogs/${CATALOG_NAME}/${MODEL_NAME}/${MODEL_INST}/referenced-by")
REV_AFTER_COUNT=$(echo "$REV_AFTER" | jq 'length')
if [ "$REV_AFTER_COUNT" = "0" ]; then
  pass "Reverse refs empty after cascade delete (links cleaned up)"
else
  fail "Reverse refs after cascade" "count=${REV_AFTER_COUNT}, expected 0"
fi

# --- Summary ---

echo ""
echo "============================================"
echo "  Results: ${PASS} passed, ${FAIL} failed"
echo "============================================"

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
