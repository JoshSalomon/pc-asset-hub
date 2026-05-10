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
  for cvid in "${CV:-}" "${DEEP_CV:-}"; do
    if [ -n "$cvid" ] && [ "$cvid" != "null" ]; then
      json_delete "${META_URL}/catalog-versions/${cvid}" "SuperAdmin" > /dev/null 2>&1 || true
    fi
  done
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

# Get the built-in string type definition version ID for hostname attribute (TD-145 test)
STRING_TDV=$(json_get "${META_URL}/type-definitions" Admin | jq -r '.items[] | select(.name=="string") | .latest_version_id')

# Add hostname attribute to Server entity type (for TD-145 attribute preservation test)
json_post "${META_URL}/entity-types/${SERVER_ET}/attributes" Admin \
  "{\"name\":\"hostname\",\"type_definition_version_id\":\"${STRING_TDV}\",\"description\":\"Server hostname\"}" > /dev/null

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

# --- Test 9: Deep containment (3 levels) ---

echo ""
echo "=== Test: Deep containment (3 levels) ==="

# Add containment: tool contains model (creates 3rd level: server → tool → model)
json_post "${META_URL}/entity-types/${TOOL_ET}/associations" Admin \
  "{\"target_entity_type_id\":\"${MODEL_ET}\",\"type\":\"containment\",\"name\":\"sub-models\"}" > /dev/null

# Get updated ETVs after adding the new association
DEEP_TOOL_ETV=$(json_get "${META_URL}/entity-types/${TOOL_ET}/versions" Admin | jq -r '.items[-1].id')
DEEP_MODEL_ETV=$(json_get "${META_URL}/entity-types/${MODEL_ET}/versions" Admin | jq -r '.items[-1].id')

# Update CV pins to use latest versions
DEEP_CV=$(json_post "${META_URL}/catalog-versions" Admin \
  "{\"version_label\":\"${PREFIX}---deep-cv-${SUFFIX}\",\"pins\":[{\"entity_type_version_id\":\"${SERVER_ETV}\"},{\"entity_type_version_id\":\"${DEEP_TOOL_ETV}\"},{\"entity_type_version_id\":\"${DEEP_MODEL_ETV}\"}]}" | jq -r '.id')

# Create a new catalog for the deep containment test
DEEP_CATALOG="${PREFIX}---deep-${SUFFIX}"
json_post "${DATA_URL}/catalogs" RW \
  "{\"name\":\"${DEEP_CATALOG}\",\"description\":\"Deep containment test\",\"catalog_version_id\":\"${DEEP_CV}\"}" > /dev/null
CLEANUP_IDS+=("${DEEP_CATALOG}")

# Level 1: Create server instance (root)
DEEP_A=$(json_post "${DATA_URL}/catalogs/${DEEP_CATALOG}/${SERVER_NAME}" RW \
  '{"name":"deep-root","description":"Level 1 server"}' | jq -r '.id')
if [ -n "$DEEP_A" ] && [ "$DEEP_A" != "null" ]; then
  pass "Create level-1 server (id=${DEEP_A})"
else
  fail "Create level-1 server" "no id returned"
fi

# Level 2: Create tool under server
DEEP_B=$(json_post "${DATA_URL}/catalogs/${DEEP_CATALOG}/${SERVER_NAME}/${DEEP_A}/${TOOL_NAME}" RW \
  '{"name":"deep-tool","description":"Level 2 tool"}' | jq -r '.id')
if [ -n "$DEEP_B" ] && [ "$DEEP_B" != "null" ]; then
  pass "Create level-2 tool (id=${DEEP_B})"
else
  fail "Create level-2 tool" "no id returned"
fi

# Level 3: Create model under tool
DEEP_C=$(json_post "${DATA_URL}/catalogs/${DEEP_CATALOG}/${TOOL_NAME}/${DEEP_B}/${MODEL_NAME}" RW \
  '{"name":"deep-model","description":"Level 3 model"}' | jq -r '.id')
if [ -n "$DEEP_C" ] && [ "$DEEP_C" != "null" ]; then
  pass "Create level-3 model (id=${DEEP_C})"
else
  fail "Create level-3 model" "no id returned"
fi

# Verify tree has 3 levels
DEEP_TREE=$(json_get "${DATA_URL}/catalogs/${DEEP_CATALOG}/tree" RO)
DEEP_ROOT_COUNT=$(echo "$DEEP_TREE" | jq 'length')
DEEP_L2_COUNT=$(echo "$DEEP_TREE" | jq '[.[] | select(.instance_name=="deep-root") | .children[]] | length')
DEEP_L3_COUNT=$(echo "$DEEP_TREE" | jq '[.[] | select(.instance_name=="deep-root") | .children[] | select(.instance_name=="deep-tool") | .children[]] | length')

if [ "$DEEP_ROOT_COUNT" -ge 1 ] && [ "$DEEP_L2_COUNT" -ge 1 ] && [ "$DEEP_L3_COUNT" -ge 1 ]; then
  pass "Tree has 3 levels (root=$DEEP_ROOT_COUNT, L2=$DEEP_L2_COUNT, L3=$DEEP_L3_COUNT)"
else
  fail "3-level tree" "root=$DEEP_ROOT_COUNT, L2=$DEEP_L2_COUNT, L3=$DEEP_L3_COUNT"
fi

# Delete root → verify cascade deletes levels 2 and 3
DEL_CODE=$(http_code -X DELETE "${DATA_URL}/catalogs/${DEEP_CATALOG}/${SERVER_NAME}/${DEEP_A}" \
  -H 'X-User-Role: RW')
if [ "$DEL_CODE" = "204" ]; then
  pass "Delete root instance (204)"
else
  fail "Delete root instance" "got ${DEL_CODE}, expected 204"
fi

# Verify level-2 tool is gone
L2_CODE=$(http_code "${DATA_URL}/catalogs/${DEEP_CATALOG}/${TOOL_NAME}/${DEEP_B}" -H "X-User-Role: RO")
if [ "$L2_CODE" = "404" ]; then
  pass "Level-2 tool cascade deleted (404)"
else
  fail "Level-2 cascade" "expected 404, got $L2_CODE"
fi

# Verify level-3 model is gone
L3_CODE=$(http_code "${DATA_URL}/catalogs/${DEEP_CATALOG}/${MODEL_NAME}/${DEEP_C}" -H "X-User-Role: RO")
if [ "$L3_CODE" = "404" ]; then
  pass "Level-3 model cascade deleted (404)"
else
  fail "Level-3 cascade" "expected 404, got $L3_CODE"
fi

# Clean up the extra CV (catalog cleanup handled by trap)
json_delete "${META_URL}/catalog-versions/${DEEP_CV}" "SuperAdmin" > /dev/null 2>&1 || true

# === TD-145: Version bumps on structural mutations ===
echo ""
echo "=== Test: Version bumps on structural mutations ==="

# Create a server and verify it starts at version 1
VB_SRV=$(json_post "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}" "RW" \
  '{"name":"vb-server","description":"version bump test"}')
VB_SRV_ID=$(echo "$VB_SRV" | jq -r '.id')
VB_SRV_VER=$(echo "$VB_SRV" | jq -r '.version')
if [ "$VB_SRV_VER" = "1" ]; then
  pass "Server created at version 1"
else
  fail "Server initial version" "expected 1, got $VB_SRV_VER"
fi

# CreateContainedInstance — parent version bumps
json_post "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}/${VB_SRV_ID}/${TOOL_NAME}" "RW" \
  '{"name":"vb-tool","description":"child"}' > /dev/null
VB_SRV_VER2=$(json_get "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}/${VB_SRV_ID}" "RO" | jq -r '.version')
if [ "$VB_SRV_VER2" = "2" ]; then
  pass "CreateContainedInstance bumps parent version (1 → 2)"
else
  fail "Parent version after CreateContained" "expected 2, got $VB_SRV_VER2"
fi

# CreateAssociationLink — source version bumps
VB_MDL=$(json_post "${DATA_URL}/catalogs/${CATALOG_NAME}/${MODEL_NAME}" "RW" \
  '{"name":"vb-model","description":"link target"}')
VB_MDL_ID=$(echo "$VB_MDL" | jq -r '.id')
json_post "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}/${VB_SRV_ID}/links" "RW" \
  "{\"target_instance_id\":\"${VB_MDL_ID}\",\"association_name\":\"uses-model\"}" > /dev/null
VB_SRV_VER3=$(json_get "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}/${VB_SRV_ID}" "RO" | jq -r '.version')
if [ "$VB_SRV_VER3" = "3" ]; then
  pass "CreateAssociationLink bumps source version (2 → 3)"
else
  fail "Source version after CreateLink" "expected 3, got $VB_SRV_VER3"
fi

# DeleteAssociationLink — source version bumps again
VB_LINK_ID=$(json_get "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}/${VB_SRV_ID}/references" "RO" | jq -r '.[0].link_id')
json_delete "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}/${VB_SRV_ID}/links/${VB_LINK_ID}" "RW" > /dev/null
VB_SRV_VER4=$(json_get "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}/${VB_SRV_ID}" "RO" | jq -r '.version')
if [ "$VB_SRV_VER4" = "4" ]; then
  pass "DeleteAssociationLink bumps source version (3 → 4)"
else
  fail "Source version after DeleteLink" "expected 4, got $VB_SRV_VER4"
fi

# SetParent — bumps child and new parent versions
VB_TOOL_ID=$(json_get "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}/${VB_SRV_ID}/${TOOL_NAME}" "RO" | jq -r '.items[0].id')
VB_TOOL_VER=$(json_get "${DATA_URL}/catalogs/${CATALOG_NAME}/${TOOL_NAME}/${VB_TOOL_ID}" "RO" | jq -r '.version')
# Create a second server to reparent to
VB_SRV2=$(json_post "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}" "RW" \
  '{"name":"vb-server-2","description":"new parent"}')
VB_SRV2_ID=$(echo "$VB_SRV2" | jq -r '.id')
VB_SRV2_VER=$(echo "$VB_SRV2" | jq -r '.version')
curl -s -X PUT "${DATA_URL}/catalogs/${CATALOG_NAME}/${TOOL_NAME}/${VB_TOOL_ID}/parent" \
  -H "X-User-Role: RW" -H "Content-Type: application/json" \
  -d "{\"parent_type\":\"${SERVER_NAME}\",\"parent_instance_id\":\"${VB_SRV2_ID}\"}" > /dev/null
VB_TOOL_VER2=$(json_get "${DATA_URL}/catalogs/${CATALOG_NAME}/${TOOL_NAME}/${VB_TOOL_ID}" "RO" | jq -r '.version')
VB_SRV2_VER2=$(json_get "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}/${VB_SRV2_ID}" "RO" | jq -r '.version')
if [ "$VB_TOOL_VER2" = "$((VB_TOOL_VER + 1))" ]; then
  pass "SetParent bumps child version (${VB_TOOL_VER} → ${VB_TOOL_VER2})"
else
  fail "Child version after SetParent" "expected $((VB_TOOL_VER + 1)), got $VB_TOOL_VER2"
fi
if [ "$VB_SRV2_VER2" = "$((VB_SRV2_VER + 1))" ]; then
  pass "SetParent bumps new parent version (${VB_SRV2_VER} → ${VB_SRV2_VER2})"
else
  fail "New parent version after SetParent" "expected $((VB_SRV2_VER + 1)), got $VB_SRV2_VER2"
fi

# Test attribute preservation after version bump (bug fix verification)
# Create a server with attributes, add a child, verify parent attributes still visible
ATTR_SRV=$(json_post "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}" "RW" \
  '{"name":"attr-test-server","attributes":{"hostname":"web1.example.com"}}')
ATTR_SRV_ID=$(echo "$ATTR_SRV" | jq -r '.id')
# Verify attribute is set at version 1
ATTR_SRV_DATA=$(json_get "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}/${ATTR_SRV_ID}" "RO")
ATTR_HOSTNAME=$(echo "$ATTR_SRV_DATA" | jq -r '.attributes[] | select(.name=="hostname") | .value')
if [ "$ATTR_HOSTNAME" = "web1.example.com" ]; then
  pass "Server attribute set at version 1"
else
  fail "Initial attribute value" "expected web1.example.com, got $ATTR_HOSTNAME"
fi
# Add a child (bumps parent version 1 → 2)
json_post "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}/${ATTR_SRV_ID}/${TOOL_NAME}" "RW" \
  '{"name":"attr-tool","description":"triggers parent version bump"}' > /dev/null
# Verify parent is now at version 2
ATTR_SRV_VER=$(json_get "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}/${ATTR_SRV_ID}" "RO" | jq -r '.version')
if [ "$ATTR_SRV_VER" = "2" ]; then
  pass "Parent bumped to version 2 after CreateContainedInstance"
else
  fail "Parent version after child creation" "expected 2, got $ATTR_SRV_VER"
fi
# CRITICAL: Verify attribute still visible at version 2 (bug fix)
ATTR_SRV_DATA2=$(json_get "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}/${ATTR_SRV_ID}" "RO")
ATTR_HOSTNAME2=$(echo "$ATTR_SRV_DATA2" | jq -r '.attributes[] | select(.name=="hostname") | .value')
if [ "$ATTR_HOSTNAME2" = "web1.example.com" ]; then
  pass "Attribute preserved after version bump (bug fix verified)"
else
  fail "Attribute after version bump" "expected web1.example.com, got $ATTR_HOSTNAME2 (attribute disappeared)"
fi

# === Optimistic locking (version conflict) ===
echo ""
echo "=== Test: Optimistic locking (version conflict) ==="

# Create an instance
OL_INST=$(json_post "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}" "RW" \
  "{\"name\":\"ol-test\",\"description\":\"optimistic locking test\"}")
OL_ID=$(echo "$OL_INST" | jq -r '.id')
OL_VER=$(echo "$OL_INST" | jq -r '.version')
pass "Create instance for locking test (id=$OL_ID, version=$OL_VER)"

# First update succeeds (correct version)
OL_UPD1=$(curl -s -w "\n%{http_code}" -X PUT \
  "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}/${OL_ID}" \
  -H "X-User-Role: RW" -H "Content-Type: application/json" \
  -d "{\"version\":${OL_VER},\"description\":\"updated once\"}")
OL_UPD1_CODE=$(echo "$OL_UPD1" | tail -1)
OL_UPD1_BODY=$(echo "$OL_UPD1" | sed '$d')
if [ "$OL_UPD1_CODE" = "200" ]; then
  pass "First update succeeds (200)"
else
  fail "First update" "expected 200, got $OL_UPD1_CODE"
fi

# Second update with ORIGINAL (stale) version → 409 conflict
OL_UPD2=$(curl -s -w "\n%{http_code}" -X PUT \
  "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}/${OL_ID}" \
  -H "X-User-Role: RW" -H "Content-Type: application/json" \
  -d "{\"version\":${OL_VER},\"description\":\"should conflict\"}")
OL_UPD2_CODE=$(echo "$OL_UPD2" | tail -1)
if [ "$OL_UPD2_CODE" = "409" ]; then
  pass "Stale version update rejected (409)"
else
  fail "Stale version update" "expected 409, got $OL_UPD2_CODE"
fi

# Verify instance has the first update's data, not the stale one
OL_GET=$(json_get "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}/${OL_ID}" "RO")
OL_DESC=$(echo "$OL_GET" | jq -r '.description')
OL_FINAL_VER=$(echo "$OL_GET" | jq -r '.version')
if [ "$OL_DESC" = "updated once" ]; then
  pass "Instance has correct data after conflict (description='updated once')"
else
  fail "Instance data after conflict" "expected 'updated once', got '$OL_DESC'"
fi
if [ "$OL_FINAL_VER" = "2" ]; then
  pass "Version incremented only once (version=2)"
else
  fail "Version after conflict" "expected 2, got $OL_FINAL_VER"
fi

# --- Summary ---

echo ""
echo "============================================"
echo "  Results: ${PASS} passed, ${FAIL} failed"
echo "============================================"

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
