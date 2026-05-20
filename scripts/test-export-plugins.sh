#!/usr/bin/env bash
# Live system tests for FF-15: Export Plugins
#
# Usage:
#   ./scripts/test-export-plugins.sh                          # defaults to localhost:30080
#   ./scripts/test-export-plugins.sh http://localhost:30080    # explicit

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/test-summary.sh"

API_BASE="${1:-http://localhost:30080}"
META_URL="${API_BASE}/api/meta/v1"
DATA_URL="${API_BASE}/api/data/v1"

PASS=0
FAIL=0
CLEANUP_IDS=()
PREFIX="exp"
SUFFIX=$(date +%s)

pass() { echo "  PASS: $1"; PASS=$((PASS + 1)); }
fail() { echo "  FAIL: $1 — $2"; FAIL=$((FAIL + 1)); }

http_code() {
  curl -s -o /dev/null -w "%{http_code}" "$@"
}

json_post() {
  curl -s -X POST "$1" -H 'Content-Type: application/json' -H "X-User-Role: $2" -d "$3"
}

json_put() {
  curl -s -X PUT "$1" -H 'Content-Type: application/json' -H "X-User-Role: $2" -d "$3"
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
    json_delete "${DATA_URL}/catalogs/${name}" "SuperAdmin" > /dev/null 2>&1 || true
  done
  if [ -n "${CV:-}" ] && [ "$CV" != "null" ]; then
    json_delete "${META_URL}/catalog-versions/${CV}" "SuperAdmin" > /dev/null 2>&1 || true
  fi
  if [ -n "${SERVER_ET_ID:-}" ]; then
    json_delete "${META_URL}/entity-types/${SERVER_ET_ID}" "Admin" > /dev/null 2>&1 || true
  fi
  if [ -n "${TOOL_ET_ID:-}" ]; then
    json_delete "${META_URL}/entity-types/${TOOL_ET_ID}" "Admin" > /dev/null 2>&1 || true
  fi
  if [ -n "${VS_ET_ID:-}" ]; then
    json_delete "${META_URL}/entity-types/${VS_ET_ID}" "Admin" > /dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

echo "=== FF-15 Export Plugins Live Tests ==="
echo "API: $API_BASE"
echo ""

# --- Setup: Create entity types, CV, catalog with instances ---
echo "=== Setup ==="

SERVER_ET=$(json_post "${META_URL}/entity-types" "Admin" "{\"name\":\"${PREFIX}-server-${SUFFIX}\"}")
SERVER_ET_ID=$(echo "$SERVER_ET" | jq -r '.entity_type.id')
SERVER_ETV_ID=$(echo "$SERVER_ET" | jq -r '.version.id')
echo "  Server ET: $SERVER_ET_ID"

TOOL_ET=$(json_post "${META_URL}/entity-types" "Admin" "{\"name\":\"${PREFIX}-tool-${SUFFIX}\"}")
TOOL_ET_ID=$(echo "$TOOL_ET" | jq -r '.entity_type.id')
TOOL_ETV_ID=$(echo "$TOOL_ET" | jq -r '.version.id')
echo "  Tool ET: $TOOL_ET_ID"

VS_ET=$(json_post "${META_URL}/entity-types" "Admin" "{\"name\":\"${PREFIX}-vs-${SUFFIX}\"}")
VS_ET_ID=$(echo "$VS_ET" | jq -r '.entity_type.id')
VS_ETV_ID=$(echo "$VS_ET" | jq -r '.version.id')
echo "  VS ET: $VS_ET_ID"

# Look up system type definition version for "string"
STRING_TD_ID=$(json_get "${META_URL}/type-definitions" "Admin" | jq -r '.items[] | select(.name == "string") | .id')
STRING_TDV=$(json_get "${META_URL}/type-definitions/${STRING_TD_ID}/versions" "Admin" | jq -r '.items[0].id')

# Add route_name attribute to server
json_post "${META_URL}/entity-types/${SERVER_ET_ID}/attributes" "Admin" \
  "{\"name\":\"route_name\",\"type_definition_version_id\":\"${STRING_TDV}\"}" > /dev/null

# Add containment association: server -> tool
json_post "${META_URL}/entity-types/${SERVER_ET_ID}/associations" "Admin" \
  "{\"name\":\"tools\",\"type\":\"containment\",\"target_entity_type_id\":\"${TOOL_ET_ID}\",\"source_cardinality\":\"0..1\",\"target_cardinality\":\"0..n\"}" > /dev/null

# Add directional association: virtual-server -> tool
json_post "${META_URL}/entity-types/${VS_ET_ID}/associations" "Admin" \
  "{\"name\":\"served-tools\",\"type\":\"directional\",\"target_entity_type_id\":\"${TOOL_ET_ID}\"}" > /dev/null

# Create CV with pins
CV_RESP=$(json_post "${META_URL}/catalog-versions" "Admin" "{\"label\":\"exp-v1-${SUFFIX}\"}")
CV=$(echo "$CV_RESP" | jq -r '.id')
echo "  CV: $CV"

# Get latest version IDs (they may have incremented after attribute/assoc add)
SERVER_VERSIONS=$(json_get "${META_URL}/entity-types/${SERVER_ET_ID}/versions" "Admin")
SERVER_ETV_ID=$(echo "$SERVER_VERSIONS" | jq -r '.items | sort_by(.version) | last | .id')
TOOL_VERSIONS=$(json_get "${META_URL}/entity-types/${TOOL_ET_ID}/versions" "Admin")
TOOL_ETV_ID=$(echo "$TOOL_VERSIONS" | jq -r '.items | sort_by(.version) | last | .id')
VS_VERSIONS=$(json_get "${META_URL}/entity-types/${VS_ET_ID}/versions" "Admin")
VS_ETV_ID=$(echo "$VS_VERSIONS" | jq -r '.items | sort_by(.version) | last | .id')

json_post "${META_URL}/catalog-versions/${CV}/pins" "Admin" \
  "{\"entity_type_version_id\":\"${SERVER_ETV_ID}\"}" > /dev/null
json_post "${META_URL}/catalog-versions/${CV}/pins" "Admin" \
  "{\"entity_type_version_id\":\"${TOOL_ETV_ID}\"}" > /dev/null
json_post "${META_URL}/catalog-versions/${CV}/pins" "Admin" \
  "{\"entity_type_version_id\":\"${VS_ETV_ID}\"}" > /dev/null

# Create catalog
CAT_NAME="${PREFIX}-catalog-${SUFFIX}"
json_post "${DATA_URL}/catalogs" "Admin" \
  "{\"name\":\"${CAT_NAME}\",\"description\":\"Export test catalog\",\"catalog_version_id\":\"${CV}\"}" > /dev/null
CLEANUP_IDS+=("$CAT_NAME")
echo "  Catalog: $CAT_NAME"

# Create server instance with route_name
json_post "${DATA_URL}/catalogs/${CAT_NAME}/${PREFIX}-server-${SUFFIX}" "Admin" \
  "{\"name\":\"github\",\"description\":\"GitHub MCP server\",\"attributes\":{\"route_name\":\"github-route\"}}" > /dev/null

# Get server instance ID
INSTANCES=$(json_get "${DATA_URL}/catalogs/${CAT_NAME}/${PREFIX}-server-${SUFFIX}" "Admin")
SERVER_INST_ID=$(echo "$INSTANCES" | jq -r '.items[0].id')

# Create tool instances (contained by server)
json_post "${DATA_URL}/catalogs/${CAT_NAME}/${PREFIX}-server-${SUFFIX}/${SERVER_INST_ID}/${PREFIX}-tool-${SUFFIX}" "Admin" \
  "{\"name\":\"list-repos\"}" > /dev/null
json_post "${DATA_URL}/catalogs/${CAT_NAME}/${PREFIX}-server-${SUFFIX}/${SERVER_INST_ID}/${PREFIX}-tool-${SUFFIX}" "Admin" \
  "{\"name\":\"create-issue\"}" > /dev/null

# Create virtual server instance
json_post "${DATA_URL}/catalogs/${CAT_NAME}/${PREFIX}-vs-${SUFFIX}" "Admin" \
  "{\"name\":\"my-vs\"}" > /dev/null
VS_INSTANCES=$(json_get "${DATA_URL}/catalogs/${CAT_NAME}/${PREFIX}-vs-${SUFFIX}" "Admin")
VS_INST_ID=$(echo "$VS_INSTANCES" | jq -r '.items[0].id')

# Get tool instance IDs for linking
TOOL_INSTANCES=$(json_get "${DATA_URL}/catalogs/${CAT_NAME}/${PREFIX}-tool-${SUFFIX}" "Admin")
TOOL1_ID=$(echo "$TOOL_INSTANCES" | jq -r '.items[] | select(.name=="list-repos") | .id')
TOOL2_ID=$(echo "$TOOL_INSTANCES" | jq -r '.items[] | select(.name=="create-issue") | .id')

# Link VS -> tool instances
json_post "${DATA_URL}/catalogs/${CAT_NAME}/${PREFIX}-vs-${SUFFIX}/${VS_INST_ID}/links" "Admin" \
  "{\"target_instance_id\":\"${TOOL1_ID}\",\"association_name\":\"served-tools\"}" > /dev/null
json_post "${DATA_URL}/catalogs/${CAT_NAME}/${PREFIX}-vs-${SUFFIX}/${VS_INST_ID}/links" "Admin" \
  "{\"target_instance_id\":\"${TOOL2_ID}\",\"association_name\":\"served-tools\"}" > /dev/null

echo "  Setup complete."
echo ""

# --- T-34.100: GET /exporters returns registered exporters ---
echo "=== Exporter List ==="
EXPORTERS=$(json_get "${DATA_URL}/exporters" "RO")
EXPORTER_COUNT=$(echo "$EXPORTERS" | jq '.items | length')
if [ "$EXPORTER_COUNT" -ge 1 ]; then
  pass "T-34.100: GET /exporters returns at least 1 exporter"
else
  fail "T-34.100: GET /exporters" "expected >=1 exporter, got $EXPORTER_COUNT"
fi

MCP_GW=$(echo "$EXPORTERS" | jq '.items[] | select(.name == "mcp-gateway")')
if [ -n "$MCP_GW" ]; then
  pass "T-34.100b: mcp-gateway exporter is registered"
else
  fail "T-34.100b: mcp-gateway exporter" "not found"
fi

# --- T-34.101: Create binding ---
echo ""
echo "=== Binding CRUD ==="
BINDING=$(json_post "${DATA_URL}/catalogs/${CAT_NAME}/export-bindings" "Admin" \
  "{\"exporter_name\":\"mcp-gateway\",\"parameters\":{\"server_type\":\"${PREFIX}-server-${SUFFIX}\",\"tool_type\":\"${PREFIX}-tool-${SUFFIX}\",\"virtual_server_type\":\"${PREFIX}-vs-${SUFFIX}\",\"target_namespace\":\"test-ns\"}}")
BINDING_ID=$(echo "$BINDING" | jq -r '.id')
if [ -n "$BINDING_ID" ] && [ "$BINDING_ID" != "null" ]; then
  pass "T-34.101: Create binding returns ID"
else
  fail "T-34.101: Create binding" "no ID returned"
fi

# --- T-34.102: List bindings ---
BINDINGS=$(json_get "${DATA_URL}/catalogs/${CAT_NAME}/export-bindings" "RO")
BINDING_COUNT=$(echo "$BINDINGS" | jq '.items | length')
if [ "$BINDING_COUNT" -eq 1 ]; then
  pass "T-34.102: List bindings returns 1 binding"
else
  fail "T-34.102: List bindings" "expected 1, got $BINDING_COUNT"
fi

# --- T-34.103: RW cannot create binding ---
CODE=$(http_code -X POST "${DATA_URL}/catalogs/${CAT_NAME}/export-bindings" \
  -H 'Content-Type: application/json' -H 'X-User-Role: RW' \
  -d '{"exporter_name":"mcp-gateway","parameters":{}}')
if [ "$CODE" = "403" ]; then
  pass "T-34.103: RW create binding returns 403"
else
  fail "T-34.103: RW create binding RBAC" "expected 403, got $CODE"
fi

# --- T-34.104: Create binding with invalid exporter returns 400 ---
CODE=$(http_code -X POST "${DATA_URL}/catalogs/${CAT_NAME}/export-bindings" \
  -H 'Content-Type: application/json' -H 'X-User-Role: Admin' \
  -d '{"exporter_name":"no-such","parameters":{}}')
if [ "$CODE" = "400" ]; then
  pass "T-34.104: Invalid exporter returns 400"
else
  fail "T-34.104: Invalid exporter" "expected 400, got $CODE"
fi

# --- T-34.105: Run binding produces YAML ---
echo ""
echo "=== Run & Download ==="
RUN_RESP=$(curl -s -X POST "${DATA_URL}/catalogs/${CAT_NAME}/export-bindings/${BINDING_ID}/run?virtual_server_instance=my-vs" \
  -H "X-User-Role: RW" -D /dev/stderr 2>/tmp/run_headers)
RUN_CT=$(grep -i "content-type" /tmp/run_headers | head -1)
if echo "$RUN_CT" | grep -qi "yaml"; then
  pass "T-34.105: Run returns YAML content-type"
else
  fail "T-34.105: Run content-type" "expected yaml, got: $RUN_CT"
fi

if echo "$RUN_RESP" | grep -q "MCPServerRegistration"; then
  pass "T-34.106: Run produces MCPServerRegistration CRs"
else
  fail "T-34.106: Run MCPServerRegistration" "not found in output"
fi

if echo "$RUN_RESP" | grep -q "MCPVirtualServer"; then
  pass "T-34.107: Run produces MCPVirtualServer CR"
else
  fail "T-34.107: Run MCPVirtualServer" "not found in output"
fi

if echo "$RUN_RESP" | grep -q "github_list-repos"; then
  pass "T-34.108: VirtualServer contains prefixed tool names"
else
  fail "T-34.108: Prefixed tool names" "not found in output"
fi

# T-34.107 supplement: verify both tools are present
if echo "$RUN_RESP" | grep -q "github_create-issue"; then
  pass "T-34.107b: VirtualServer contains all tools (create-issue)"
else
  fail "T-34.107b: Second tool" "github_create-issue not found in output"
fi

# T-34.75m: Verify VirtualServer CR name matches VS instance name, not catalog name
if echo "$RUN_RESP" | grep -q "name: my-vs"; then
  pass "T-34.75n: VirtualServer CR name matches VS instance"
else
  fail "T-34.75n: VS CR name" "expected 'name: my-vs' in output"
fi

# --- T-34.109: RO cannot run ---
CODE=$(http_code -X POST "${DATA_URL}/catalogs/${CAT_NAME}/export-bindings/${BINDING_ID}/run" \
  -H "X-User-Role: RO")
if [ "$CODE" = "403" ]; then
  pass "T-34.109: RO run returns 403"
else
  fail "T-34.109: RO run RBAC" "expected 403, got $CODE"
fi

# --- T-34.110: Update binding ---
json_put "${DATA_URL}/catalogs/${CAT_NAME}/export-bindings/${BINDING_ID}" "Admin" \
  "{\"enabled\":false}" > /dev/null
UPDATED=$(json_get "${DATA_URL}/catalogs/${CAT_NAME}/export-bindings/${BINDING_ID}" "Admin")
ENABLED=$(echo "$UPDATED" | jq '.enabled')
if [ "$ENABLED" = "false" ]; then
  pass "T-34.110: Update binding disabled"
else
  fail "T-34.110: Update binding" "expected enabled=false, got $ENABLED"
fi

# --- T-34.110b: Run on disabled binding returns 400 ---
DISABLED_CODE=$(http_code -X POST "${DATA_URL}/catalogs/${CAT_NAME}/export-bindings/${BINDING_ID}/run" \
  -H "X-User-Role: RW")
if [ "$DISABLED_CODE" = "400" ]; then
  pass "T-34.110b: Run on disabled binding returns 400"
else
  fail "T-34.110b: Run on disabled binding" "expected 400, got $DISABLED_CODE"
fi

# Re-enable for further tests
json_put "${DATA_URL}/catalogs/${CAT_NAME}/export-bindings/${BINDING_ID}" "Admin" \
  "{\"enabled\":true}" > /dev/null

# --- T-34.111: Delete binding ---
CODE=$(http_code -X DELETE "${DATA_URL}/catalogs/${CAT_NAME}/export-bindings/${BINDING_ID}" \
  -H "X-User-Role: Admin")
if [ "$CODE" = "204" ]; then
  pass "T-34.111: Delete binding returns 204"
else
  fail "T-34.111: Delete binding" "expected 204, got $CODE"
fi

# Verify gone
BINDINGS=$(json_get "${DATA_URL}/catalogs/${CAT_NAME}/export-bindings" "RO")
BINDING_COUNT=$(echo "$BINDINGS" | jq '.items | length')
if [ "$BINDING_COUNT" -eq 0 ]; then
  pass "T-34.112: Binding deleted from list"
else
  fail "T-34.112: Binding still in list" "expected 0, got $BINDING_COUNT"
fi

print_summary "test-export-plugins"
