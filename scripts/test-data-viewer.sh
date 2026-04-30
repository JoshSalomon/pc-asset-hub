#!/usr/bin/env bash
# Live system tests for Phase 4: Catalog Data Viewer
#
# Usage:
#   ./scripts/test-data-viewer.sh                          # defaults to localhost:30080
#   ./scripts/test-data-viewer.sh http://localhost:30080    # explicit local
#
# Tests:
#   - Containment tree endpoint
#   - Pagination (limit/offset)
#   - Sorting (sort=name:asc/desc)
#   - Filtering (filter.attrName=value)
#   - Parent chain in instance detail
#   - Operational UI served at /operational
#
# Prerequisites:
#   - API server running and healthy
#   - UI server running (for operational UI test)
#   - jq installed
#   - curl installed

set -euo pipefail

API_BASE="${1:-http://localhost:30080}"
UI_BASE="${2:-http://localhost:30000}"
META_URL="${API_BASE}/api/meta/v1"
DATA_URL="${API_BASE}/api/data/v1"

PASS=0
FAIL=0
CLEANUP_IDS=()
PREFIX="dv"  # data viewer test prefix
SUFFIX=$(date +%s)

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
  for name in "${CLEANUP_IDS[@]}"; do
    json_delete "${DATA_URL}/catalogs/${name}" "RW" > /dev/null 2>&1 || true
  done
  if [ -n "${CV:-}" ] && [ "${CV}" != "null" ]; then
    json_delete "${META_URL}/catalog-versions/${CV}" "SuperAdmin" > /dev/null 2>&1 || true
  fi
  for et in "${MODEL_ET:-}" "${TOOL_ET:-}" "${SERVER_ET:-}"; do
    if [ -n "$et" ] && [ "$et" != "null" ]; then
      json_delete "${META_URL}/entity-types/${et}" "Admin" > /dev/null 2>&1 || true
    fi
  done
  echo "Done."
}

trap cleanup EXIT

# --- Setup: Create entity types, associations, CV, catalog with instances ---

echo "=== Setup ==="

# Create entity types with unique names
SERVER_NAME="${PREFIX}---srv-${SUFFIX}"
TOOL_NAME="${PREFIX}---tool-${SUFFIX}"
MODEL_NAME="${PREFIX}---mdl-${SUFFIX}"

SERVER_RESP=$(json_post "${META_URL}/entity-types" Admin "{\"name\":\"${SERVER_NAME}\",\"description\":\"server\"}")
SERVER_ET=$(echo "$SERVER_RESP" | jq -r '.entity_type.id')
TOOL_RESP=$(json_post "${META_URL}/entity-types" Admin "{\"name\":\"${TOOL_NAME}\",\"description\":\"tool\"}")
TOOL_ET=$(echo "$TOOL_RESP" | jq -r '.entity_type.id')
TOOL_ETV=$(echo "$TOOL_RESP" | jq -r '.version.id')
MODEL_RESP=$(json_post "${META_URL}/entity-types" Admin "{\"name\":\"${MODEL_NAME}\",\"description\":\"model\"}")
MODEL_ET=$(echo "$MODEL_RESP" | jq -r '.entity_type.id')
MODEL_ETV=$(echo "$MODEL_RESP" | jq -r '.version.id')
echo "  Entity types: server=${SERVER_ET:0:8}, tool=${TOOL_ET:0:8}, model=${MODEL_ET:0:8}"

# Add string attribute to server
STRING_TDV_ID=$(json_get "${META_URL}/type-definitions" Admin | jq -r '.items[] | select(.name=="string") | .latest_version_id')
json_post "${META_URL}/entity-types/${SERVER_ET}/attributes" Admin \
  "{\"name\":\"endpoint\",\"type_definition_version_id\":\"${STRING_TDV_ID}\",\"required\":true}" > /dev/null

# Add containment: server contains tool
json_post "${META_URL}/entity-types/${SERVER_ET}/associations" Admin \
  "{\"target_entity_type_id\":\"${TOOL_ET}\",\"type\":\"containment\",\"name\":\"tools\"}" > /dev/null

# Add directional: server uses model — capture latest server ETV
SERVER_ETV=$(json_post "${META_URL}/entity-types/${SERVER_ET}/associations" Admin \
  "{\"target_entity_type_id\":\"${MODEL_ET}\",\"type\":\"directional\",\"name\":\"uses-model\"}" | jq -r '.id')

# Create CV
CV=$(json_post "${META_URL}/catalog-versions" Admin \
  "{\"version_label\":\"${PREFIX}---cv-${SUFFIX}\",\"pins\":[{\"entity_type_version_id\":\"${SERVER_ETV}\"},{\"entity_type_version_id\":\"${TOOL_ETV}\"},{\"entity_type_version_id\":\"${MODEL_ETV}\"}]}" | jq -r '.id')
echo "  CV: ${CV:0:8}"

# Create catalog
CATALOG_NAME="${PREFIX}---cat-${SUFFIX}"
json_post "${DATA_URL}/catalogs" RW \
  "{\"name\":\"${CATALOG_NAME}\",\"catalog_version_id\":\"${CV}\"}" > /dev/null
CLEANUP_IDS+=("${CATALOG_NAME}")
echo "  Catalog: ${CATALOG_NAME}"

# Create instances: 3 servers, 2 tools under first server, 1 model
SRV1=$(json_post "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}" RW \
  "{\"name\":\"alpha-server\",\"description\":\"First server\",\"attributes\":{\"endpoint\":\"https://alpha.example.com\"}}" | jq -r '.id')
SRV2=$(json_post "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}" RW \
  "{\"name\":\"bravo-server\",\"description\":\"Second server\",\"attributes\":{\"endpoint\":\"https://bravo.example.com\"}}" | jq -r '.id')
SRV3=$(json_post "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}" RW \
  "{\"name\":\"charlie-server\",\"description\":\"Third server\",\"attributes\":{\"endpoint\":\"https://charlie.example.com\"}}" | jq -r '.id')

TOOL1=$(json_post "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}/${SRV1}/${TOOL_NAME}" RW \
  "{\"name\":\"tool-one\",\"description\":\"First tool\"}" | jq -r '.id')
TOOL2=$(json_post "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}/${SRV1}/${TOOL_NAME}" RW \
  "{\"name\":\"tool-two\",\"description\":\"Second tool\"}" | jq -r '.id')

MDL1=$(json_post "${DATA_URL}/catalogs/${CATALOG_NAME}/${MODEL_NAME}" RW \
  "{\"name\":\"gpt-4\",\"description\":\"A language model\"}" | jq -r '.id')

# Create link: server uses model
json_post "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}/${SRV1}/links" RW \
  "{\"target_instance_id\":\"${MDL1}\",\"association_name\":\"uses-model\"}" > /dev/null

echo "  Instances: 3 servers, 2 tools, 1 model, 1 link"
echo ""

# ================================================================
# Tests
# ================================================================

echo "=== T1: Containment Tree ==="

# T1.1: Tree returns correct structure
TREE=$(json_get "${DATA_URL}/catalogs/${CATALOG_NAME}/tree" RO)
TREE_LEN=$(echo "$TREE" | jq 'length')
if [ "$TREE_LEN" -ge 3 ]; then
  pass "T1.1 Tree returns root instances (got $TREE_LEN roots)"
else
  fail "T1.1 Tree root count" "expected >=3, got $TREE_LEN"
fi

# T1.2: Tree includes entity type names
ET_NAME=$(echo "$TREE" | jq -r '.[0].entity_type_name')
if [ "$ET_NAME" != "null" ] && [ -n "$ET_NAME" ]; then
  pass "T1.2 Tree nodes include entity type name ($ET_NAME)"
else
  fail "T1.2 Entity type name" "got null or empty"
fi

# T1.3: Tree has children nested under parent
ALPHA_CHILDREN=$(echo "$TREE" | jq "[.[] | select(.instance_name==\"alpha-server\") | .children[]] | length")
if [ "$ALPHA_CHILDREN" -eq 2 ]; then
  pass "T1.3 alpha-server has 2 children (tools)"
else
  fail "T1.3 Children count" "expected 2, got $ALPHA_CHILDREN"
fi

# T1.4: Tree 404 for nonexistent catalog
CODE=$(http_code "${DATA_URL}/catalogs/nonexistent-catalog-xyz/tree" -H "X-User-Role: RO")
if [ "$CODE" = "404" ]; then
  pass "T1.4 Tree returns 404 for nonexistent catalog"
else
  fail "T1.4 Nonexistent catalog" "expected 404, got $CODE"
fi

echo ""
echo "=== T2: Pagination ==="

# T2.1: Default pagination returns up to 20
RESP=$(json_get "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}" RO)
TOTAL=$(echo "$RESP" | jq '.total')
ITEMS=$(echo "$RESP" | jq '.items | length')
if [ "$TOTAL" -eq 3 ] && [ "$ITEMS" -eq 3 ]; then
  pass "T2.1 Default list returns all 3 servers (total=$TOTAL)"
else
  fail "T2.1 Default list" "total=$TOTAL, items=$ITEMS"
fi

# T2.2: limit=1 returns 1 item with correct total
RESP=$(json_get "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}?limit=1" RO)
TOTAL=$(echo "$RESP" | jq '.total')
ITEMS=$(echo "$RESP" | jq '.items | length')
if [ "$TOTAL" -eq 3 ] && [ "$ITEMS" -eq 1 ]; then
  pass "T2.2 limit=1 returns 1 item, total=3"
else
  fail "T2.2 Limit" "total=$TOTAL, items=$ITEMS"
fi

# T2.3: offset=1&limit=1 skips first item
RESP=$(json_get "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}?offset=1&limit=1" RO)
NAME=$(echo "$RESP" | jq -r '.items[0].name')
if [ "$NAME" = "bravo-server" ]; then
  pass "T2.3 offset=1 returns bravo-server"
else
  fail "T2.3 Offset" "expected bravo-server, got $NAME"
fi

# T2.4: limit capped at 100
RESP=$(json_get "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}?limit=500" RO)
ITEMS=$(echo "$RESP" | jq '.items | length')
if [ "$ITEMS" -le 100 ]; then
  pass "T2.4 limit=500 capped (got $ITEMS items)"
else
  fail "T2.4 Limit cap" "got $ITEMS items, expected <=100"
fi

echo ""
echo "=== T3: Sorting ==="

# T3.1: sort=name:asc (default should already be ascending)
RESP=$(json_get "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}?sort=name:asc" RO)
FIRST=$(echo "$RESP" | jq -r '.items[0].name')
LAST=$(echo "$RESP" | jq -r '.items[-1].name')
if [ "$FIRST" = "alpha-server" ] && [ "$LAST" = "charlie-server" ]; then
  pass "T3.1 sort=name:asc returns alpha first, charlie last"
else
  fail "T3.1 Ascending sort" "first=$FIRST, last=$LAST"
fi

# T3.2: sort=name:desc reverses order
RESP=$(json_get "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}?sort=name:desc" RO)
FIRST=$(echo "$RESP" | jq -r '.items[0].name')
LAST=$(echo "$RESP" | jq -r '.items[-1].name')
if [ "$FIRST" = "charlie-server" ] && [ "$LAST" = "alpha-server" ]; then
  pass "T3.2 sort=name:desc returns charlie first, alpha last"
else
  fail "T3.2 Descending sort" "first=$FIRST, last=$LAST"
fi

echo ""
echo "=== T4: Filtering ==="

# T4.1: Filter by string attribute (endpoint contains "bravo")
RESP=$(json_get "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}?filter.endpoint=bravo" RO)
TOTAL=$(echo "$RESP" | jq '.total')
NAME=$(echo "$RESP" | jq -r '.items[0].name // empty')
if [ "$TOTAL" -eq 1 ] && [ "$NAME" = "bravo-server" ]; then
  pass "T4.1 filter.endpoint=bravo returns bravo-server"
else
  fail "T4.1 String filter" "total=$TOTAL, name=$NAME"
fi

# T4.2: Filter with no matches returns empty
RESP=$(json_get "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}?filter.endpoint=nonexistent" RO)
TOTAL=$(echo "$RESP" | jq '.total')
if [ "$TOTAL" -eq 0 ]; then
  pass "T4.2 Filter with no match returns total=0"
else
  fail "T4.2 No match filter" "total=$TOTAL"
fi

echo ""
echo "=== T5: Parent Chain ==="

# T5.1: Child instance has parent chain
RESP=$(json_get "${DATA_URL}/catalogs/${CATALOG_NAME}/${TOOL_NAME}/${TOOL1}" RO)
CHAIN_LEN=$(echo "$RESP" | jq '.parent_chain | length')
CHAIN_NAME=$(echo "$RESP" | jq -r '.parent_chain[0].instance_name // empty')
if [ "$CHAIN_LEN" -eq 1 ] && [ "$CHAIN_NAME" = "alpha-server" ]; then
  pass "T5.1 Tool has parent chain [alpha-server]"
else
  fail "T5.1 Parent chain" "length=$CHAIN_LEN, name=$CHAIN_NAME"
fi

# T5.2: Root instance has no parent chain
RESP=$(json_get "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}/${SRV1}" RO)
CHAIN=$(echo "$RESP" | jq '.parent_chain // []')
CHAIN_LEN=$(echo "$CHAIN" | jq 'length')
if [ "$CHAIN_LEN" -eq 0 ]; then
  pass "T5.2 Root instance has empty parent chain"
else
  fail "T5.2 Root chain" "expected 0, got $CHAIN_LEN"
fi

# T5.3: Parent chain includes entity type name
RESP=$(json_get "${DATA_URL}/catalogs/${CATALOG_NAME}/${TOOL_NAME}/${TOOL1}" RO)
ET=$(echo "$RESP" | jq -r '.parent_chain[0].entity_type_name // empty')
if [ -n "$ET" ] && [ "$ET" != "null" ]; then
  pass "T5.3 Parent chain has entity type name ($ET)"
else
  fail "T5.3 Entity type in chain" "got empty or null"
fi

echo ""
echo "=== T6: Operational UI ==="

# T6.1: Operational HTML is served
CODE=$(http_code "${UI_BASE}/operational")
if [ "$CODE" = "200" ]; then
  pass "T6.1 /operational returns 200"
else
  fail "T6.1 Operational UI" "expected 200, got $CODE"
fi

# T6.2: Unified SPA title
TITLE=$(curl -s "${UI_BASE}/operational" | grep -o '<title>.*</title>')
if echo "$TITLE" | grep -q "AI Asset Hub"; then
  pass "T6.2 Unified SPA has 'AI Asset Hub' in title"
else
  fail "T6.2 Title" "got $TITLE"
fi

# T6.3: Root URL returns unified SPA
CODE=$(http_code "${UI_BASE}/")
if [ "$CODE" = "200" ]; then
  pass "T6.3 Root URL returns 200"
else
  fail "T6.3 Root URL" "expected 200, got $CODE"
fi

echo ""
echo "=== T7: Combined Queries ==="

# T7.1: Pagination + sort combined
RESP=$(json_get "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}?sort=name:desc&limit=2" RO)
ITEMS=$(echo "$RESP" | jq '.items | length')
FIRST=$(echo "$RESP" | jq -r '.items[0].name')
if [ "$ITEMS" -eq 2 ] && [ "$FIRST" = "charlie-server" ]; then
  pass "T7.1 sort=name:desc&limit=2 returns charlie first, 2 items"
else
  fail "T7.1 Combined" "items=$ITEMS, first=$FIRST"
fi

# T7.2: Filter + pagination combined
RESP=$(json_get "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}?filter.endpoint=example&limit=2" RO)
TOTAL=$(echo "$RESP" | jq '.total')
ITEMS=$(echo "$RESP" | jq '.items | length')
if [ "$TOTAL" -eq 3 ] && [ "$ITEMS" -eq 2 ]; then
  pass "T7.2 filter.endpoint=example&limit=2 returns total=3, items=2"
else
  fail "T7.2 Filter+pagination" "total=$TOTAL, items=$ITEMS"
fi

echo ""
echo "=== T8: References in Detail ==="

# T8.1: Forward references for server with link
REFS=$(json_get "${DATA_URL}/catalogs/${CATALOG_NAME}/${SERVER_NAME}/${SRV1}/references" RO)
REFS_LEN=$(echo "$REFS" | jq 'length')
if [ "$REFS_LEN" -ge 1 ]; then
  pass "T8.1 Forward references returns >= 1 ref"
else
  fail "T8.1 Forward refs" "got $REFS_LEN"
fi

# T8.2: Forward ref includes entity type name
REF_ET=$(echo "$REFS" | jq -r '.[0].entity_type_name // empty')
if [ -n "$REF_ET" ] && [ "$REF_ET" != "null" ]; then
  pass "T8.2 Reference includes entity type name ($REF_ET)"
else
  fail "T8.2 Ref entity type" "got empty"
fi

# T8.3: Reverse ref on model
RVREFS=$(json_get "${DATA_URL}/catalogs/${CATALOG_NAME}/${MODEL_NAME}/${MDL1}/referenced-by" RO)
RVREFS_LEN=$(echo "$RVREFS" | jq 'length')
if [ "$RVREFS_LEN" -ge 1 ]; then
  pass "T8.3 Reverse references on model returns >= 1"
else
  fail "T8.3 Reverse refs" "got $RVREFS_LEN"
fi

# ================================================================
# Summary
# ================================================================

echo ""
echo "========================================"
echo "  Results: $PASS passed, $FAIL failed"
echo "========================================"

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
