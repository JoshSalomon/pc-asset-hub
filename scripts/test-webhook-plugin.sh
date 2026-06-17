#!/bin/bash
# Live tests for the webhook MCP Gateway plugin (T-36.125–T-36.134)
# Includes semantic comparison between built-in and webhook export output.
set -uo pipefail

API="${1:-http://host.containers.internal:30080}"
META="$API/api/meta/v1"
DATA="$API/api/data/v1"
KUBE_CMD="${KUBE_CMD:-kubectl --context kind-assethub}"
PASS=0; FAIL=0

pass() { echo "  PASS: $1"; PASS=$((PASS + 1)); }
fail() { echo "  FAIL: $1"; FAIL=$((FAIL + 1)); }

jp() { python3 -c "import json,sys; $1"; }

echo "=== Webhook MCP Gateway Plugin Tests ==="
echo ""

# --- Setup ---
echo "=== Setup ==="
SUFFIX=$(date +%s)

# Create entity types (extract IDs and ETV IDs from creation response)
RESP=$(curl -s -X POST -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  "$META/entity-types" -d "{\"name\":\"wp-server-$SUFFIX\",\"description\":\"test server\"}")
ET_SERVER=$(echo "$RESP" | jp "print(json.load(sys.stdin)['entity_type']['id'])")
ETV_SERVER=$(echo "$RESP" | jp "print(json.load(sys.stdin)['version']['id'])")

RESP=$(curl -s -X POST -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  "$META/entity-types" -d "{\"name\":\"wp-tool-$SUFFIX\",\"description\":\"test tool\"}")
ET_TOOL=$(echo "$RESP" | jp "print(json.load(sys.stdin)['entity_type']['id'])")
ETV_TOOL=$(echo "$RESP" | jp "print(json.load(sys.stdin)['version']['id'])")

RESP=$(curl -s -X POST -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  "$META/entity-types" -d "{\"name\":\"wp-vs-$SUFFIX\",\"description\":\"test VS\"}")
ET_VS=$(echo "$RESP" | jp "print(json.load(sys.stdin)['entity_type']['id'])")
ETV_VS=$(echo "$RESP" | jp "print(json.load(sys.stdin)['version']['id'])")

# Look up string type definition version ID
STRING_TDV=$(curl -s -H 'X-User-Role: Admin' "$META/type-definitions" | jp "d=json.load(sys.stdin); print(next(t['latest_version_id'] for t in d['items'] if t['name']=='string'))")

# Add route_name attribute to server
curl -s -X POST -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  "$META/entity-types/$ET_SERVER/attributes" \
  -d "{\"name\":\"route_name\",\"description\":\"Route\",\"base_type\":\"string\",\"type_definition_version_id\":\"$STRING_TDV\"}" > /dev/null

# Add containment: server → tool
curl -s -X POST -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  "$META/entity-types/$ET_SERVER/associations" \
  -d "{\"name\":\"tools\",\"type\":\"containment\",\"target_entity_type_id\":\"$ET_TOOL\",\"cardinality\":\"0..n\"}" > /dev/null

# Add association: VS → tool
curl -s -X POST -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  "$META/entity-types/$ET_VS/associations" \
  -d "{\"name\":\"allowed-tools\",\"type\":\"directional\",\"target_entity_type_id\":\"$ET_TOOL\",\"cardinality\":\"0..n\"}" > /dev/null

# Re-fetch latest ETVs (version incremented after adding attrs/assocs)
ETV_SERVER=$(curl -s -H 'X-User-Role: Admin' "$META/entity-types/$ET_SERVER/versions" | jp "d=json.load(sys.stdin); print(sorted(d['items'], key=lambda x: x['version'])[-1]['id'])")
ETV_TOOL=$(curl -s -H 'X-User-Role: Admin' "$META/entity-types/$ET_TOOL/versions" | jp "d=json.load(sys.stdin); print(sorted(d['items'], key=lambda x: x['version'])[-1]['id'])")
ETV_VS=$(curl -s -H 'X-User-Role: Admin' "$META/entity-types/$ET_VS/versions" | jp "d=json.load(sys.stdin); print(sorted(d['items'], key=lambda x: x['version'])[-1]['id'])")

# Create CV with pins (using latest versions)
CV=$(curl -s -X POST -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  "$META/catalog-versions" \
  -d "{\"version_label\":\"wp-$SUFFIX\",\"description\":\"WP test\",\"pins\":[{\"entity_type_version_id\":\"$ETV_SERVER\"},{\"entity_type_version_id\":\"$ETV_TOOL\"},{\"entity_type_version_id\":\"$ETV_VS\"}]}" | jp "print(json.load(sys.stdin)['id'])")

# Create catalog
CATALOG="wp-test-$SUFFIX"
curl -s -X POST -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  "$DATA/catalogs" -d "{\"name\":\"$CATALOG\",\"description\":\"Webhook test\",\"catalog_version_id\":\"$CV\"}" > /dev/null

# Create instances: 2 servers, each with 2 tools (4 tools total)
# VS will link to only 2 of the 4 tools — the other 2 must NOT appear in output

SRV1_RESP=$(curl -s -X POST -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  "$DATA/catalogs/$CATALOG/wp-server-$SUFFIX" \
  -d "{\"name\":\"alpha-server\",\"description\":\"Server A\",\"attributes\":{\"route_name\":\"route-alpha\"}}")
SRV1_ID=$(echo "$SRV1_RESP" | jp "print(json.load(sys.stdin).get('id','FAIL'))")

SRV2_RESP=$(curl -s -X POST -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  "$DATA/catalogs/$CATALOG/wp-server-$SUFFIX" \
  -d "{\"name\":\"beta-server\",\"description\":\"Server B\",\"attributes\":{\"route_name\":\"route-beta\"}}")
SRV2_ID=$(echo "$SRV2_RESP" | jp "print(json.load(sys.stdin).get('id','FAIL'))")

# Server alpha: 2 tools (allowed-tool, excluded-tool-a)
curl -s -X POST -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  "$DATA/catalogs/$CATALOG/wp-server-$SUFFIX/$SRV1_ID/wp-tool-$SUFFIX" \
  -d "{\"name\":\"allowed-tool\",\"description\":\"This tool is linked to VS\",\"attributes\":{}}" > /dev/null
curl -s -X POST -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  "$DATA/catalogs/$CATALOG/wp-server-$SUFFIX/$SRV1_ID/wp-tool-$SUFFIX" \
  -d "{\"name\":\"excluded-tool-a\",\"description\":\"NOT linked to VS\",\"attributes\":{}}" > /dev/null

# Server beta: 2 tools (selected-tool, excluded-tool-b)
curl -s -X POST -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  "$DATA/catalogs/$CATALOG/wp-server-$SUFFIX/$SRV2_ID/wp-tool-$SUFFIX" \
  -d "{\"name\":\"selected-tool\",\"description\":\"This tool is linked to VS\",\"attributes\":{}}" > /dev/null
curl -s -X POST -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  "$DATA/catalogs/$CATALOG/wp-server-$SUFFIX/$SRV2_ID/wp-tool-$SUFFIX" \
  -d "{\"name\":\"excluded-tool-b\",\"description\":\"NOT linked to VS\",\"attributes\":{}}" > /dev/null

# Create VS instance
VS_RESP=$(curl -s -X POST -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  "$DATA/catalogs/$CATALOG/wp-vs-$SUFFIX" \
  -d "{\"name\":\"test-vs\",\"description\":\"Test VS\",\"attributes\":{}}")
VS_ID=$(echo "$VS_RESP" | jp "print(json.load(sys.stdin).get('id','FAIL'))")

# Get tool instance IDs — link only allowed-tool and selected-tool to VS
ALLOWED_TOOL_ID=$(curl -s -H 'X-User-Role: Admin' "$DATA/catalogs/$CATALOG/wp-tool-$SUFFIX" | jp "d=json.load(sys.stdin); print(next(i['id'] for i in d['items'] if i['name']=='allowed-tool'))")
SELECTED_TOOL_ID=$(curl -s -H 'X-User-Role: Admin' "$DATA/catalogs/$CATALOG/wp-tool-$SUFFIX" | jp "d=json.load(sys.stdin); print(next(i['id'] for i in d['items'] if i['name']=='selected-tool'))")

# Link VS → only 2 of 4 tools
curl -s -X POST -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  "$DATA/catalogs/$CATALOG/wp-vs-$SUFFIX/$VS_ID/links" \
  -d "{\"target_instance_id\":\"$ALLOWED_TOOL_ID\",\"association_name\":\"allowed-tools\"}" > /dev/null
curl -s -X POST -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  "$DATA/catalogs/$CATALOG/wp-vs-$SUFFIX/$VS_ID/links" \
  -d "{\"target_instance_id\":\"$SELECTED_TOOL_ID\",\"association_name\":\"allowed-tools\"}" > /dev/null

echo "  CV: $CV"
echo "  Catalog: $CATALOG"
echo "  Setup complete."
echo ""

# --- T-36.125: Webhook exporter visible ---
echo "=== Webhook Plugin Registration ==="

EXPORTERS=$(curl -s -H 'X-User-Role: Admin' "$DATA/exporters")
WH_SOURCE=$(echo "$EXPORTERS" | jp "d=json.load(sys.stdin); exps=[e for e in d['items'] if e['name']=='webhook-mcp-gateway']; print(exps[0]['source'] if exps else 'NOT_FOUND')")
WH_HEALTH=$(echo "$EXPORTERS" | jp "d=json.load(sys.stdin); exps=[e for e in d['items'] if e['name']=='webhook-mcp-gateway']; print(exps[0]['health'] if exps else 'NOT_FOUND')")

if [ "$WH_SOURCE" = "webhook" ]; then pass "T-36.125: source=webhook"; else fail "T-36.125: Expected webhook, got $WH_SOURCE"; fi
if [ "$WH_HEALTH" = "Ready" ]; then pass "T-36.125b: health=Ready"; else fail "T-36.125b: Expected Ready, got $WH_HEALTH"; fi

# --- Create bindings to BOTH exporters ---
echo ""
echo "=== Create Bindings & Export ==="

# Create bindings WITHOUT virtual_server_type for fair comparison
# (both exporters get the full unfiltered model)
BUILTIN_BINDING=$(curl -s -X POST -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  "$DATA/catalogs/$CATALOG/export-bindings" \
  -d "{\"exporter_name\":\"mcp-gateway\",\"parameters\":{\"server_type\":\"wp-server-$SUFFIX\",\"tool_type\":\"wp-tool-$SUFFIX\",\"virtual_server_type\":\"wp-vs-$SUFFIX\",\"target_namespace\":\"default\"}}" | jp "print(json.load(sys.stdin).get('id','FAIL'))")

WEBHOOK_BINDING=$(curl -s -X POST -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  "$DATA/catalogs/$CATALOG/export-bindings" \
  -d "{\"exporter_name\":\"webhook-mcp-gateway\",\"parameters\":{\"server_type\":\"wp-server-$SUFFIX\",\"tool_type\":\"wp-tool-$SUFFIX\",\"virtual_server_type\":\"wp-vs-$SUFFIX\",\"target_namespace\":\"default\"}}" | jp "print(json.load(sys.stdin).get('id','FAIL'))")

[ "$BUILTIN_BINDING" != "FAIL" ] && pass "T-36.126a: Built-in binding created" || fail "T-36.126a: Built-in binding failed"
[ "$WEBHOOK_BINDING" != "FAIL" ] && pass "T-36.126b: Webhook binding created" || fail "T-36.126b: Webhook binding failed"

# --- Run exports WITH VS instance (both get full model + VS name, each filters independently) ---
BUILTIN_YAML=$(curl -s -X POST -H 'X-User-Role: Admin' "$DATA/catalogs/$CATALOG/export-bindings/$BUILTIN_BINDING/run?virtual_server_instance=test-vs")
echo "  Built-in run response: $(echo "$BUILTIN_YAML" | head -c 200)"
WEBHOOK_YAML=$(curl -s -X POST -H 'X-User-Role: Admin' "$DATA/catalogs/$CATALOG/export-bindings/$WEBHOOK_BINDING/run?virtual_server_instance=test-vs")
echo "  Webhook run response: $(echo "$WEBHOOK_YAML" | head -c 200)"
echo "  Webhook run response: $(echo "$WEBHOOK_YAML" | head -c 200)"

# Both must produce MCPServerRegistration AND MCPVirtualServer
echo "$BUILTIN_YAML" | grep -q "MCPServerRegistration" && pass "T-36.126c: Built-in → MCPServerRegistration" || fail "T-36.126c: Built-in missing MCPServerRegistration"
echo "$WEBHOOK_YAML" | grep -q "MCPServerRegistration" && pass "T-36.126d: Webhook → MCPServerRegistration" || fail "T-36.126d: Webhook missing MCPServerRegistration"
echo "$BUILTIN_YAML" | grep -q "MCPVirtualServer" && pass "T-36.126e: Built-in → MCPVirtualServer" || fail "T-36.126e: Built-in missing MCPVirtualServer"
echo "$WEBHOOK_YAML" | grep -q "MCPVirtualServer" && pass "T-36.126f: Webhook → MCPVirtualServer" || fail "T-36.126f: Webhook missing MCPVirtualServer"

# T-36.126 depth: verify attribute values and structure in webhook YAML
WH_CR_COUNT=$(echo "$WEBHOOK_YAML" | grep -c "kind: MCPServerRegistration")
[ "$WH_CR_COUNT" -eq 2 ] && pass "T-36.126g: Webhook has 2 server CRs" || fail "T-36.126g: Expected 2 server CRs, got $WH_CR_COUNT"
echo "$WEBHOOK_YAML" | grep -q "route-alpha" && pass "T-36.126h: route_name attribute present" || fail "T-36.126h: route_name missing"
echo "$WEBHOOK_YAML" | grep -q "assethub.io/catalog: $CATALOG" && pass "T-36.126i: catalog label correct" || fail "T-36.126i: catalog label missing"

# --- Semantic comparison using compare tool ---
echo ""
echo "=== Semantic Comparison (built-in vs webhook) ==="

# Save both outputs to temp files for comparison
echo "$BUILTIN_YAML" > /tmp/wp-builtin-export.yaml
echo "$WEBHOOK_YAML" > /tmp/wp-webhook-export.yaml

# Run semantic comparison — should be EQUAL (ignoring timestamps and exporter labels)
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CMP_RESULT=$("$SCRIPT_DIR/compare-export-yaml.sh" /tmp/wp-builtin-export.yaml /tmp/wp-webhook-export.yaml 2>&1)
CMP_EXIT=$?
echo "$CMP_RESULT"

if [ $CMP_EXIT -eq 0 ]; then
  pass "Semantic comparison: built-in and webhook produce equivalent CRs"
else
  fail "Semantic comparison: built-in and webhook produce DIFFERENT CRs"
fi

# Also verify reverse direction
CMP_REV=$("$SCRIPT_DIR/compare-export-yaml.sh" /tmp/wp-webhook-export.yaml /tmp/wp-builtin-export.yaml 2>&1)
CMP_REV_EXIT=$?

if [ $CMP_REV_EXIT -eq 0 ]; then
  pass "Reverse comparison: also equal"
else
  fail "Reverse comparison: DIFFERENT"
fi

# Verify VirtualServer is named after the VS instance (not the catalog)
VS_NAME=$(echo "$WEBHOOK_YAML" | sed -n '/kind: MCPVirtualServer/,/^---/p' | grep "^  name:" | head -1 | awk '{print $2}')
if [ "$VS_NAME" = "test-vs" ]; then
  pass "Webhook VirtualServer named after VS instance 'test-vs'"
else
  fail "Webhook VirtualServer named '$VS_NAME' (expected 'test-vs')"
fi

# Verify allowed tools ARE in VirtualServer
echo "$WEBHOOK_YAML" | grep -q "alpha-server_allowed-tool" && pass "VirtualServer contains 'alpha-server_allowed-tool'" || fail "VirtualServer missing 'alpha-server_allowed-tool'"
echo "$WEBHOOK_YAML" | grep -q "beta-server_selected-tool" && pass "VirtualServer contains 'beta-server_selected-tool'" || fail "VirtualServer missing 'beta-server_selected-tool'"

# Verify EXCLUDED tools are NOT in VirtualServer (VS filtering must work)
echo "$WEBHOOK_YAML" | grep -q "excluded-tool-a" && fail "VirtualServer contains excluded-tool-a (should be filtered out)" || pass "excluded-tool-a correctly filtered out"
echo "$WEBHOOK_YAML" | grep -q "excluded-tool-b" && fail "VirtualServer contains excluded-tool-b (should be filtered out)" || pass "excluded-tool-b correctly filtered out"

# Verify both servers produced MCPServerRegistration CRs
echo "$WEBHOOK_YAML" | grep -q "alpha-server" && pass "Webhook contains alpha-server CR" || fail "Webhook missing alpha-server CR"
echo "$WEBHOOK_YAML" | grep -q "beta-server" && pass "Webhook contains beta-server CR" || fail "Webhook missing beta-server CR"

# Clean up temp files
rm -f /tmp/wp-builtin-export.yaml /tmp/wp-webhook-export.yaml

# --- T-36.127: Delete CR → exporter disappears ---
echo ""
echo "=== CR Lifecycle ==="

$KUBE_CMD -n assethub delete exporterplugin webhook-mcp-gateway --timeout=10s 2>/dev/null
sleep 3
AFTER=$(curl -s -H 'X-User-Role: Admin' "$DATA/exporters" | jp "d=json.load(sys.stdin); print([e['name'] for e in d['items']])")
echo "$AFTER" | grep -qv "webhook-mcp-gateway" && pass "T-36.127: Exporter removed after CR delete" || fail "T-36.127: Still present"

# --- T-36.128: Name collision ---
$KUBE_CMD apply -f - <<'EOF' 2>/dev/null
apiVersion: assethub.project-catalyst.io/v1alpha1
kind: ExporterPlugin
metadata:
  name: mcp-gateway
  namespace: assethub
spec:
  endpoint: http://fake.svc.cluster.local
EOF
sleep 2
COLLISION=$(curl -s -H 'X-User-Role: Admin' "$DATA/exporters" | jp "d=json.load(sys.stdin); exps=[e for e in d['items'] if e['name']=='mcp-gateway']; print(exps[0]['source'] if exps else 'NOT_FOUND')")
[ "$COLLISION" = "built-in" ] && pass "T-36.128: Name collision → built-in preserved" || fail "T-36.128: Got $COLLISION"
$KUBE_CMD -n assethub delete exporterplugin mcp-gateway --timeout=10s 2>/dev/null

# --- RBAC ---
echo ""
echo "=== RBAC ==="
RBAC1=$($KUBE_CMD -n assethub auth can-i list exporterplugins.assethub.project-catalyst.io --as=system:serviceaccount:assethub:assethub-api-server 2>&1)
[ "$RBAC1" = "yes" ] && pass "T-36.129: API SA can list" || fail "T-36.129: $RBAC1"
RBAC2=$($KUBE_CMD -n assethub auth can-i update exporterplugins.assethub.project-catalyst.io/status --as=system:serviceaccount:assethub:assethub-operator 2>&1)
[ "$RBAC2" = "yes" ] && pass "T-36.130: Operator SA can update status" || fail "T-36.130: $RBAC2"
RBAC3=$($KUBE_CMD -n assethub auth can-i update exporterplugins.assethub.project-catalyst.io/status --as=system:serviceaccount:assethub:assethub-api-server 2>&1)
[ "$RBAC3" = "no" ] && pass "T-36.131: API SA cannot update status" || fail "T-36.131: $RBAC3"

# Restore webhook exporter CR before API-level RBAC tests
$KUBE_CMD apply -f /home/jsalomon/src/pc-asset-hub/examples/webhook-mcp-gateway/deploy/exporterplugin.yaml 2>/dev/null

# Wait for webhook exporter to re-register and become healthy after CR lifecycle tests
for i in $(seq 1 15); do
  WH_CHECK=$(curl -s -H 'X-User-Role: Admin' "$DATA/exporters" | jp "d=json.load(sys.stdin); exps=[e for e in d.get('items',[]) if e['name']=='webhook-mcp-gateway']; print(exps[0].get('health','') if exps else 'no')")
  [ "$WH_CHECK" = "Ready" ] && break
  sleep 3
done

# T-36.132: RO cannot create binding to webhook exporter
RBAC4=$(curl -s -o /dev/null -w '%{http_code}' -X POST -H 'X-User-Role: RO' -H 'Content-Type: application/json' \
  "$DATA/catalogs/$CATALOG/export-bindings" \
  -d '{"exporter_name":"webhook-mcp-gateway","parameters":{"server_type":"s","tool_type":"t"}}')
[ "$RBAC4" = "403" ] && pass "T-36.132: RO cannot create webhook binding" || fail "T-36.132: Expected 403, got $RBAC4"

# T-36.133: RW can run export on webhook binding
RBAC5=$(curl -s -o /dev/null -w '%{http_code}' -X POST -H 'X-User-Role: RW' "$DATA/catalogs/$CATALOG/export-bindings/$WEBHOOK_BINDING/run?virtual_server_instance=test-vs")
[ "$RBAC5" = "200" ] && pass "T-36.133: RW can run webhook export" || fail "T-36.133: Expected 200, got $RBAC5"

# T-36.134: Admin can create/update/delete binding to webhook exporter
RBAC6_BIND=$(curl -s -X POST -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  "$DATA/catalogs/$CATALOG/export-bindings" \
  -d "{\"exporter_name\":\"webhook-mcp-gateway\",\"parameters\":{\"server_type\":\"wp-server-$SUFFIX\",\"tool_type\":\"wp-tool-$SUFFIX\",\"virtual_server_type\":\"wp-vs-$SUFFIX\",\"target_namespace\":\"default\"}}" | jp "print(json.load(sys.stdin).get('id','FAIL'))")
RBAC6_UPD=$(curl -s -o /dev/null -w '%{http_code}' -X PUT -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  "$DATA/catalogs/$CATALOG/export-bindings/$RBAC6_BIND" \
  -d "{\"parameters\":{\"server_type\":\"wp-server-$SUFFIX\",\"tool_type\":\"wp-tool-$SUFFIX\",\"virtual_server_type\":\"wp-vs-$SUFFIX\",\"target_namespace\":\"prod\"}}")
RBAC6_DEL=$(curl -s -o /dev/null -w '%{http_code}' -X DELETE -H 'X-User-Role: Admin' "$DATA/catalogs/$CATALOG/export-bindings/$RBAC6_BIND")
if [ "$RBAC6_BIND" != "FAIL" ] && [ "$RBAC6_UPD" = "200" ] && [ "$RBAC6_DEL" = "204" ]; then
  pass "T-36.134: Admin full CRUD on webhook binding"
else
  fail "T-36.134: create=$RBAC6_BIND update=$RBAC6_UPD delete=$RBAC6_DEL"
fi

# --- Plugin-level auth tests (T-36.135–T-36.138) ---
# Uses a temporary curl pod since API server is distroless (no curl)
echo ""
echo "=== Plugin Auth ==="
PLUGIN_URL="http://webhook-mcp-gateway.assethub.svc.cluster.local"
CURL_IMG="curlimages/curl:latest"

plugin_curl() {
  $KUBE_CMD run -n assethub "plugin-auth-$$" --rm -i --restart=Never --image="$CURL_IMG" -- \
    curl -s -o /dev/null -w '%{http_code}' "$@" 2>&1 | grep -oE '^[0-9]{3}'
}

# T-36.135: Missing token → 401
P_AUTH1=$(plugin_curl -X POST \
  -H 'Content-Type: application/json' -H 'X-AssetHub-Protocol-Version: v1' \
  "$PLUGIN_URL/validate" -d '{"parameters":{},"schema":{"entity_types":[]}}')
[ "$P_AUTH1" = "401" ] && pass "T-36.135: Plugin rejects missing token" || fail "T-36.135: Expected 401, got $P_AUTH1"

# T-36.136: Wrong auth format → 401
P_AUTH2=$(plugin_curl -X POST \
  -H 'Content-Type: application/json' -H 'X-AssetHub-Protocol-Version: v1' \
  -H 'Authorization: Basic wrongformat' \
  "$PLUGIN_URL/validate" -d '{"parameters":{},"schema":{"entity_types":[]}}')
[ "$P_AUTH2" = "401" ] && pass "T-36.136: Plugin rejects wrong auth format" || fail "T-36.136: Expected 401, got $P_AUTH2"

# T-36.137: Valid Bearer token → accepted
P_AUTH3=$(plugin_curl -X POST \
  -H 'Content-Type: application/json' -H 'X-AssetHub-Protocol-Version: v1' \
  -H 'Authorization: Bearer test-token' \
  "$PLUGIN_URL/validate" -d '{"parameters":{"server_type":"s","tool_type":"t"},"schema":{"entity_types":[{"name":"s","attributes":["route_name"],"associations":[{"name":"tools","type":"containment","target_entity_type":"t"}]},{"name":"t","attributes":[]}]}}')
[ "$P_AUTH3" = "200" ] && pass "T-36.137: Plugin accepts valid token" || fail "T-36.137: Expected 200, got $P_AUTH3"

# T-36.138: Wrong protocol version → 400
P_AUTH4=$(plugin_curl -X POST \
  -H 'Content-Type: application/json' -H 'X-AssetHub-Protocol-Version: v99' \
  -H 'Authorization: Bearer test-token' \
  "$PLUGIN_URL/validate" -d '{"parameters":{}}')
[ "$P_AUTH4" = "400" ] && pass "T-36.138: Plugin rejects v99 protocol" || fail "T-36.138: Expected 400, got $P_AUTH4"

# --- Section 1: attribute_mappings in API response ---
echo ""
echo "=== Attribute Mappings ==="
AM_COUNT=$(echo "$EXPORTERS" | jp "d=json.load(sys.stdin); wh=[e for e in d['items'] if e['name']=='webhook-mcp-gateway'][0]; st=[p for p in wh['parameter_schema'] if p['name']=='server_type'][0]; print(len(st.get('attribute_mappings',[])))")
[ "$AM_COUNT" -ge 2 ] && pass "attribute_mappings present on server_type ($AM_COUNT)" || fail "attribute_mappings missing or empty ($AM_COUNT)"

AM_REQUIRED=$(echo "$EXPORTERS" | jp "d=json.load(sys.stdin); wh=[e for e in d['items'] if e['name']=='webhook-mcp-gateway'][0]; st=[p for p in wh['parameter_schema'] if p['name']=='server_type'][0]; ams=st.get('attribute_mappings',[]); rna=[a for a in ams if a['name']=='route_name_attr']; print(rna[0].get('required',False) if rna else 'NOT_FOUND')")
[ "$AM_REQUIRED" = "True" ] && pass "route_name_attr is required" || fail "route_name_attr required=$AM_REQUIRED"

# tool_type should NOT have attribute_mappings
AM_TOOL=$(echo "$EXPORTERS" | jp "d=json.load(sys.stdin); wh=[e for e in d['items'] if e['name']=='webhook-mcp-gateway'][0]; tt=[p for p in wh['parameter_schema'] if p['name']=='tool_type'][0]; print(len(tt.get('attribute_mappings',[])))")
[ "$AM_TOOL" = "0" ] && pass "tool_type has no attribute_mappings" || fail "tool_type has $AM_TOOL attribute_mappings"

# --- Section 8: Publish preview with orphaned binding ---
echo ""
echo "=== Publish Preview (Orphaned) ==="
# Create a binding, then orphan it
PP_BIND=$(curl -s -X POST -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  "$DATA/catalogs/$CATALOG/export-bindings" \
  -d "{\"exporter_name\":\"webhook-mcp-gateway\",\"parameters\":{\"server_type\":\"wp-server-$SUFFIX\",\"tool_type\":\"wp-tool-$SUFFIX\",\"virtual_server_type\":\"wp-vs-$SUFFIX\",\"target_namespace\":\"default\"}}" | jp "print(json.load(sys.stdin).get('id','FAIL'))")

$KUBE_CMD -n assethub delete exporterplugin webhook-mcp-gateway --timeout=10s 2>/dev/null
sleep 3

PP_RESP=$(curl -s -X POST -H 'X-User-Role: Admin' "$DATA/catalogs/$CATALOG/publish/preview")
PP_STATUS=$(echo "$PP_RESP" | jp "d=json.load(sys.stdin); bs=[b for b in d.get('bindings',[]) if b['binding_id']=='$PP_BIND']; print(bs[0]['status'] if bs else 'NOT_FOUND')")
PP_FAILURES=$(echo "$PP_RESP" | jp "print(json.load(sys.stdin).get('has_failures', 'NOT_FOUND'))")

[ "$PP_STATUS" = "skipped" ] && pass "Orphaned binding status=skipped" || fail "Orphaned binding status=$PP_STATUS (expected skipped)"
[ "$PP_FAILURES" = "False" ] && pass "has_failures=false (skipped != failure)" || fail "has_failures=$PP_FAILURES"

# Restore CR
$KUBE_CMD apply -f /home/jsalomon/src/pc-asset-hub/examples/webhook-mcp-gateway/deploy/exporterplugin.yaml 2>/dev/null
# Wait for re-registration + health
for i in $(seq 1 15); do
  WH_BACK=$(curl -s -H 'X-User-Role: Admin' "$DATA/exporters" | jp "d=json.load(sys.stdin); exps=[e for e in d.get('items',[]) if e['name']=='webhook-mcp-gateway']; print(exps[0].get('health','') if exps else 'no')")
  [ "$WH_BACK" = "Ready" ] && break
  sleep 3
done
# Clean up preview binding
curl -s -X DELETE -H 'X-User-Role: Admin' "$DATA/catalogs/$CATALOG/export-bindings/$PP_BIND" > /dev/null 2>&1

# --- Section 10: Binding CRUD lifecycle ---
echo ""
echo "=== Binding CRUD Lifecycle ==="
# Create
CRUD_BIND=$(curl -s -X POST -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  "$DATA/catalogs/$CATALOG/export-bindings" \
  -d "{\"exporter_name\":\"webhook-mcp-gateway\",\"parameters\":{\"server_type\":\"wp-server-$SUFFIX\",\"tool_type\":\"wp-tool-$SUFFIX\",\"virtual_server_type\":\"wp-vs-$SUFFIX\",\"target_namespace\":\"default\"}}" | jp "print(json.load(sys.stdin).get('id','FAIL'))")
[ "$CRUD_BIND" != "FAIL" ] && pass "CRUD: create binding" || fail "CRUD: create failed"

# Read (verify in list)
CRUD_LIST=$(curl -s -H 'X-User-Role: Admin' "$DATA/catalogs/$CATALOG/export-bindings" | jp "d=json.load(sys.stdin); print('yes' if any(b['id']=='$CRUD_BIND' for b in d.get('items',[])) else 'no')")
[ "$CRUD_LIST" = "yes" ] && pass "CRUD: binding in list" || fail "CRUD: binding not in list"

# Update parameters
CRUD_UPD=$(curl -s -o /dev/null -w '%{http_code}' -X PUT -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  "$DATA/catalogs/$CATALOG/export-bindings/$CRUD_BIND" \
  -d "{\"parameters\":{\"server_type\":\"wp-server-$SUFFIX\",\"tool_type\":\"wp-tool-$SUFFIX\",\"virtual_server_type\":\"wp-vs-$SUFFIX\",\"target_namespace\":\"prod\"}}")
[ "$CRUD_UPD" = "200" ] && pass "CRUD: update parameters" || fail "CRUD: update got $CRUD_UPD"

# Toggle enabled=false
CRUD_DIS=$(curl -s -o /dev/null -w '%{http_code}' -X PUT -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  "$DATA/catalogs/$CATALOG/export-bindings/$CRUD_BIND" -d '{"enabled":false}')
[ "$CRUD_DIS" = "200" ] && pass "CRUD: disable binding" || fail "CRUD: disable got $CRUD_DIS"

# Run on disabled → should fail
CRUD_RUN_DIS=$(curl -s -o /dev/null -w '%{http_code}' -X POST -H 'X-User-Role: RW' "$DATA/catalogs/$CATALOG/export-bindings/$CRUD_BIND/run?virtual_server_instance=test-vs")
[ "$CRUD_RUN_DIS" = "400" ] && pass "CRUD: disabled binding cannot run" || fail "CRUD: disabled run got $CRUD_RUN_DIS"

# Re-enable
curl -s -X PUT -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  "$DATA/catalogs/$CATALOG/export-bindings/$CRUD_BIND" -d '{"enabled":true}' > /dev/null

# Delete
CRUD_DEL=$(curl -s -o /dev/null -w '%{http_code}' -X DELETE -H 'X-User-Role: Admin' "$DATA/catalogs/$CATALOG/export-bindings/$CRUD_BIND")
[ "$CRUD_DEL" = "204" ] && pass "CRUD: delete binding" || fail "CRUD: delete got $CRUD_DEL"

# Verify gone
CRUD_GONE=$(curl -s -H 'X-User-Role: Admin' "$DATA/catalogs/$CATALOG/export-bindings" | jp "d=json.load(sys.stdin); print('yes' if any(b['id']=='$CRUD_BIND' for b in d.get('items',[])) else 'no')")
[ "$CRUD_GONE" = "no" ] && pass "CRUD: binding removed from list" || fail "CRUD: binding still in list"

# --- Section 12: Name collision with log check ---
echo ""
echo "=== Name Collision ==="
$KUBE_CMD apply -f - <<'COLLEOF' 2>/dev/null
apiVersion: assethub.project-catalyst.io/v1alpha1
kind: ExporterPlugin
metadata:
  name: mcp-gateway
  namespace: assethub
spec:
  endpoint: "http://fake.svc.cluster.local"
COLLEOF
sleep 3

# Verify built-in is preserved
COLL_SRC=$(curl -s -H 'X-User-Role: Admin' "$DATA/exporters" | jp "d=json.load(sys.stdin); exps=[e for e in d['items'] if e['name']=='mcp-gateway']; print(exps[0]['source'] if exps else 'NOT_FOUND')")
[ "$COLL_SRC" = "built-in" ] && pass "Name collision: built-in preserved" || fail "Name collision: source=$COLL_SRC"

# Check log for warning
API_POD=$($KUBE_CMD -n assethub get pod -l app=assethub-api -o jsonpath='{.items[0].metadata.name}')
COLL_LOG=$($KUBE_CMD -n assethub logs "$API_POD" --since=30s 2>/dev/null | grep -c "mcp-gateway.*skipped\|reserved")
[ "$COLL_LOG" -ge 1 ] && pass "Name collision: warning in API server log" || fail "Name collision: no warning in log"

# Clean up collision CR
$KUBE_CMD -n assethub delete exporterplugin mcp-gateway --timeout=10s 2>/dev/null

# --- Restore & Cleanup ---
$KUBE_CMD apply -f /home/jsalomon/src/pc-asset-hub/examples/webhook-mcp-gateway/deploy/exporterplugin.yaml 2>/dev/null
echo ""
echo "=== Cleanup ==="
curl -s -X DELETE -H 'X-User-Role: SuperAdmin' "$DATA/catalogs/$CATALOG" > /dev/null 2>&1
curl -s -X DELETE -H 'X-User-Role: Admin' "$META/catalog-versions/$CV" > /dev/null 2>&1
curl -s -X DELETE -H 'X-User-Role: Admin' "$META/entity-types/$ET_SERVER" > /dev/null 2>&1
curl -s -X DELETE -H 'X-User-Role: Admin' "$META/entity-types/$ET_TOOL" > /dev/null 2>&1
curl -s -X DELETE -H 'X-User-Role: Admin' "$META/entity-types/$ET_VS" > /dev/null 2>&1

echo ""
echo -e "\033[0;32mtest-webhook-plugin: Tests: $((PASS+FAIL)), Passed: $PASS, Failed: $FAIL\033[0m"
[ "$FAIL" -gt 0 ] && exit 1 || exit 0
