#!/usr/bin/env bash
# Live test script for Catalog Import/Export (US-55, US-56, Milestone 20)
# Usage: ./scripts/test-import-export.sh [API_BASE_URL]
set -uo pipefail

API="${1:-http://localhost:30080}"
PASS=0; FAIL=0; TOTAL=0

pass() { ((PASS++)); ((TOTAL++)); echo "  PASS: $1"; }
fail() { ((FAIL++)); ((TOTAL++)); echo "  FAIL: $1 — $2"; }

h() { echo ""; echo "=== $1 ==="; }

# Entity type prefix unlikely to collide with real names
P="zz-ixt"

cleanup() {
  h "Cleanup"
  for name in export-test-cat import-test-cat imported-cat reimport-cat renamed-cat contain-test-cat assoc-test-cat; do
    curl -s -o /dev/null "$API/api/data/v1/catalogs/$name" -X DELETE -H 'X-User-Role: SuperAdmin' 2>/dev/null || true
  done
  if [ -n "${CV_ID:-}" ]; then
    curl -s -o /dev/null "$API/api/meta/v1/catalog-versions/$CV_ID" -X DELETE -H 'X-User-Role: Admin' 2>/dev/null || true
  fi
  for etid in ${ET_IDS:-}; do
    curl -s -o /dev/null "$API/api/meta/v1/entity-types/$etid" -X DELETE -H 'X-User-Role: Admin' 2>/dev/null || true
  done
  # Clean up imported CVs (created by import with suffixed labels)
  for prefix in "${P}-imported-v1" "${P}-renamed-v1" "${P}-test-v1" "${P}-contain-v1" "${P}-assoc-v1"; do
    cvid=$(curl -s "$API/api/meta/v1/catalog-versions" -H 'X-User-Role: Admin' | jq -r ".items[] | select(.version_label | startswith(\"$prefix\")) | .id" 2>/dev/null | head -5)
    for id in $cvid; do
      curl -s -o /dev/null "$API/api/meta/v1/catalog-versions/$id" -X DELETE -H 'X-User-Role: Admin' 2>/dev/null || true
    done
  done
  # Clean up imported entity types
  for name in "${P}-server" "${P}-guard" "${P}-tool" "imported-${P}-server" "imported-${P}-guard" "imported-${P}-tool" "renamed-${P}-server" "renamed-${P}-guard" "renamed-${P}-tool" "assoc-${P}-server" "assoc-${P}-guard" "assoc-${P}-tool"; do
    etid=$(curl -s "$API/api/meta/v1/entity-types?name=$name" -H 'X-User-Role: Admin' | jq -r '.items[0].id // empty')
    if [ -n "$etid" ]; then
      curl -s -o /dev/null "$API/api/meta/v1/entity-types/$etid" -X DELETE -H 'X-User-Role: Admin' 2>/dev/null || true
    fi
  done
  echo "  Cleaned up test data"
}
trap cleanup EXIT

# Pre-cleanup
for name in export-test-cat import-test-cat imported-cat reimport-cat renamed-cat contain-test-cat assoc-test-cat; do
  curl -s -o /dev/null "$API/api/data/v1/catalogs/$name" -X DELETE -H 'X-User-Role: SuperAdmin' 2>/dev/null || true
done

# --- Setup ---
h "Setup"
SUFFIX=$(date +%s)
ET_IDS=""

# Create entity type: server
SERVER_RESP=$(curl -s "$API/api/meta/v1/entity-types" -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d "{\"name\":\"${P}-server\",\"description\":\"Server for import/export test\"}")
SERVER_ET_ID=$(echo "$SERVER_RESP" | jq -r '.entity_type.id')
SERVER_ETV_ID=$(echo "$SERVER_RESP" | jq -r '.version.id')
ET_IDS="$SERVER_ET_ID"

# Create entity type: guard
GUARD_RESP=$(curl -s "$API/api/meta/v1/entity-types" -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d "{\"name\":\"${P}-guard\",\"description\":\"Guard for import/export test\"}")
GUARD_ET_ID=$(echo "$GUARD_RESP" | jq -r '.entity_type.id')
GUARD_ETV_ID=$(echo "$GUARD_RESP" | jq -r '.version.id')
ET_IDS="$ET_IDS $GUARD_ET_ID"

# Add attribute to server (using system "string" type)
STRING_TDV_ID=$(curl -s "$API/api/meta/v1/type-definitions" -H 'X-User-Role: Admin' | jq -r '.items[] | select(.name=="string") | .latest_version_id')
curl -s -o /dev/null "$API/api/meta/v1/entity-types/$SERVER_ET_ID/attributes" -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d "{\"name\":\"endpoint\",\"type_definition_version_id\":\"$STRING_TDV_ID\",\"required\":true}"

# Create entity type: tool (contained by server)
TOOL_RESP=$(curl -s "$API/api/meta/v1/entity-types" -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d "{\"name\":\"${P}-tool\",\"description\":\"Tool for import/export test\"}")
TOOL_ET_ID=$(echo "$TOOL_RESP" | jq -r '.entity_type.id')
ET_IDS="$ET_IDS $TOOL_ET_ID"

# Add attribute to tool — capture latest ETV ID from response
TOOL_ETV_ID=$(curl -s "$API/api/meta/v1/entity-types/$TOOL_ET_ID/attributes" -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d "{\"name\":\"command\",\"type_definition_version_id\":\"$STRING_TDV_ID\"}" | jq -r '.id')

# Add directional association: server → guard
curl -s -o /dev/null "$API/api/meta/v1/entity-types/$SERVER_ET_ID/associations" -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d "{\"name\":\"pre-execute\",\"type\":\"directional\",\"target_entity_type_id\":\"$GUARD_ET_ID\",\"source_cardinality\":\"0..n\",\"target_cardinality\":\"0..n\"}"

# Add bidirectional association: server ↔ guard (guardrails)
curl -s -o /dev/null "$API/api/meta/v1/entity-types/$SERVER_ET_ID/associations" -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d "{\"name\":\"guardrails\",\"type\":\"bidirectional\",\"target_entity_type_id\":\"$GUARD_ET_ID\",\"source_cardinality\":\"0..n\",\"target_cardinality\":\"0..n\"}"

# Add containment association: server contains tool — capture latest server ETV ID
SERVER_ETV_ID=$(curl -s "$API/api/meta/v1/entity-types/$SERVER_ET_ID/associations" -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d "{\"name\":\"tools\",\"type\":\"containment\",\"target_entity_type_id\":\"$TOOL_ET_ID\",\"source_cardinality\":\"1\",\"target_cardinality\":\"0..n\"}" | jq -r '.id')

# Create CV and pin all three entity types (latest versions)
CV_ID=$(curl -s "$API/api/meta/v1/catalog-versions" -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d "{\"version_label\":\"${P}-test-v1-$SUFFIX\",\"pins\":[{\"entity_type_version_id\":\"$SERVER_ETV_ID\"},{\"entity_type_version_id\":\"$GUARD_ETV_ID\"},{\"entity_type_version_id\":\"$TOOL_ETV_ID\"}]}" | jq -r '.id')

# Create catalog
curl -s -o /dev/null "$API/api/data/v1/catalogs" -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d "{\"name\":\"export-test-cat\",\"description\":\"Export test catalog\",\"catalog_version_id\":\"$CV_ID\"}"

# Create instances
curl -s -o /dev/null "$API/api/data/v1/catalogs/export-test-cat/${P}-server" -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d '{"name":"github-server","description":"GitHub MCP server","attributes":{"endpoint":"https://github.example.com/mcp"}}'
curl -s -o /dev/null "$API/api/data/v1/catalogs/export-test-cat/${P}-guard" -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d '{"name":"pii-filter","description":"PII filter guard"}'

# Create contained tool instance under server
SERVER_INST_ID=$(curl -s "$API/api/data/v1/catalogs/export-test-cat/${P}-server" -H 'X-User-Role: Admin' | jq -r '.items[0].id')
TOOL_CREATE_RESP=$(curl -s -w "\n%{http_code}" "$API/api/data/v1/catalogs/export-test-cat/${P}-server/$SERVER_INST_ID/${P}-tool" -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d '{"name":"list-repos","description":"List repositories","attributes":{"command":"gh repo list"}}')
TOOL_CREATE_CODE=$(echo "$TOOL_CREATE_RESP" | tail -1)
echo "  Contained tool creation: HTTP $TOOL_CREATE_CODE"
if [ "$TOOL_CREATE_CODE" != "201" ]; then
  echo "  WARNING: $(echo "$TOOL_CREATE_RESP" | head -1)"
fi

# Create link: server → guard (directional)
GUARD_INST_ID=$(curl -s "$API/api/data/v1/catalogs/export-test-cat/${P}-guard" -H 'X-User-Role: Admin' | jq -r '.items[0].id')
curl -s -o /dev/null "$API/api/data/v1/catalogs/export-test-cat/${P}-server/$SERVER_INST_ID/links" -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d "{\"target_instance_id\":\"$GUARD_INST_ID\",\"association_name\":\"pre-execute\"}"

# Create link: guard → server (bidirectional from REVERSE side — tests reverse-side link creation)
curl -s -o /dev/null "$API/api/data/v1/catalogs/export-test-cat/${P}-guard/$GUARD_INST_ID/links" -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d "{\"target_instance_id\":\"$SERVER_INST_ID\",\"association_name\":\"guardrails\"}"

echo "  Setup complete: ET=$SERVER_ET_ID, CV=$CV_ID"

if [ "$SERVER_ET_ID" = "null" ] || [ "$CV_ID" = "null" ]; then
  echo "  FATAL: Setup failed"
  exit 1
fi

# === Test 1: Export catalog ===
h "Test 1: Export catalog"
CODE=$(curl -s -o /tmp/export-resp.json -w "%{http_code}" "$API/api/data/v1/catalogs/export-test-cat/export" -H 'X-User-Role: Admin')

if [ "$CODE" = "200" ]; then
  pass "Export returns 200"
else
  fail "Export returns 200" "got $CODE — $(cat /tmp/export-resp.json)"
fi

FMT=$(jq -r '.format_version' /tmp/export-resp.json)
if [ "$FMT" = "1.0" ]; then
  pass "Export format_version is 1.0"
else
  fail "Export format_version is 1.0" "got $FMT"
fi

CAT_NAME=$(jq -r '.catalog.name' /tmp/export-resp.json)
if [ "$CAT_NAME" = "export-test-cat" ]; then
  pass "Export catalog name correct"
else
  fail "Export catalog name correct" "got $CAT_NAME"
fi

ET_COUNT=$(jq '.entity_types | length' /tmp/export-resp.json)
if [ "$ET_COUNT" = "3" ]; then
  pass "Export has 3 entity types"
else
  fail "Export has 3 entity types" "got $ET_COUNT"
fi

INST_COUNT=$(jq '.instances | length' /tmp/export-resp.json)
if [ "$INST_COUNT" = "2" ]; then
  pass "Export has 2 root instances"
else
  fail "Export has 2 root instances" "got $INST_COUNT"
fi

# === Test 2: Export with entity filter ===
h "Test 2: Export with entity filter"
CODE=$(curl -s -o /tmp/export-filter.json -w "%{http_code}" "$API/api/data/v1/catalogs/export-test-cat/export?entities=${P}-server" -H 'X-User-Role: Admin')

FILTERED_ET_COUNT=$(jq '.entity_types | length' /tmp/export-filter.json)
if [ "$CODE" = "200" ] && [ "$FILTERED_ET_COUNT" = "1" ]; then
  pass "Entity filter returns only selected type"
else
  fail "Entity filter returns only selected type" "code=$CODE, et_count=$FILTERED_ET_COUNT"
fi

# === Test 3: Export with source_system override ===
h "Test 3: Export with source_system override"
CODE=$(curl -s -o /tmp/export-src.json -w "%{http_code}" "$API/api/data/v1/catalogs/export-test-cat/export?source_system=prod-cluster" -H 'X-User-Role: Admin')

SRC=$(jq -r '.source_system' /tmp/export-src.json)
if [ "$SRC" = "prod-cluster" ]; then
  pass "Source system override works"
else
  fail "Source system override works" "got $SRC"
fi

# === Test 4: Export requires Admin ===
h "Test 4: Export requires Admin"
CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API/api/data/v1/catalogs/export-test-cat/export" -H 'X-User-Role: RW')
if [ "$CODE" = "403" ]; then
  pass "Export denied for RW role"
else
  fail "Export denied for RW role" "got $CODE"
fi

# === Test 5: Export non-existent catalog ===
h "Test 5: Export non-existent catalog"
CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API/api/data/v1/catalogs/nonexistent/export" -H 'X-User-Role: Admin')
if [ "$CODE" = "404" ]; then
  pass "Export non-existent catalog returns 404"
else
  fail "Export non-existent catalog returns 404" "got $CODE"
fi

# === Test 6: Import dry-run — no conflicts ===
h "Test 6: Import dry-run — no conflicts"
# Modify exported file with new names
jq --arg lbl "${P}-imported-v1-$SUFFIX" '.catalog.name = "import-test-cat" | .catalog_version.label = $lbl' /tmp/export-resp.json > /tmp/import-data.json

IMPORT_REQ=$(jq -n --arg lbl "${P}-imported-v1-$SUFFIX" \
  --arg s "${P}-server" --arg g "${P}-guard" --arg t "${P}-tool" \
  --slurpfile data /tmp/import-data.json '{
  catalog_name: "import-test-cat",
  catalog_version_label: $lbl,
  reuse_existing: [$s, $g, $t],
  data: $data[0]
}')

CODE=$(curl -s -o /tmp/dryrun-resp.json -w "%{http_code}" \
  "$API/api/data/v1/catalogs/import?dry_run=true" \
  -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d "$IMPORT_REQ")

if [ "$CODE" = "200" ]; then
  pass "Dry run returns 200"
else
  fail "Dry run returns 200" "got $CODE — $(cat /tmp/dryrun-resp.json)"
fi

DR_STATUS=$(jq -r '.status' /tmp/dryrun-resp.json)
if [ "$DR_STATUS" = "ready" ]; then
  pass "Dry run status is 'ready'"
else
  fail "Dry run status is 'ready'" "got $DR_STATUS"
fi

# === Test 7: Import dry-run — catalog name conflict ===
h "Test 7: Import dry-run — catalog name conflict"
CONFLICT_REQ=$(jq -n \
  --arg s "${P}-server" --arg g "${P}-guard" --arg t "${P}-tool" \
  --slurpfile data /tmp/import-data.json '{
  catalog_name: "export-test-cat",
  reuse_existing: [$s, $g, $t],
  data: $data[0]
}')

CODE=$(curl -s -o /tmp/dryrun-conflict.json -w "%{http_code}" \
  "$API/api/data/v1/catalogs/import?dry_run=true" \
  -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d "$CONFLICT_REQ")

DR_STATUS=$(jq -r '.status' /tmp/dryrun-conflict.json)
if [ "$DR_STATUS" = "conflicts_found" ]; then
  pass "Dry run detects catalog name conflict"
else
  fail "Dry run detects catalog name conflict" "status=$DR_STATUS"
fi

# === Test 8: Import with reuse_existing ===
h "Test 8: Import with reuse_existing"
CODE=$(curl -s -o /tmp/import-resp.json -w "%{http_code}" \
  "$API/api/data/v1/catalogs/import" \
  -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d "$IMPORT_REQ")

if [ "$CODE" = "201" ]; then
  pass "Import returns 201"
else
  fail "Import returns 201" "got $CODE — $(cat /tmp/import-resp.json)"
fi

IMP_STATUS=$(jq -r '.status' /tmp/import-resp.json)
if [ "$IMP_STATUS" = "success" ]; then
  pass "Import status is 'success'"
else
  fail "Import status is 'success'" "got $IMP_STATUS"
fi

IMP_CAT=$(jq -r '.catalog_name' /tmp/import-resp.json)
if [ "$IMP_CAT" = "import-test-cat" ]; then
  pass "Imported catalog has correct name"
else
  fail "Imported catalog has correct name" "got $IMP_CAT"
fi

# === Test 9: Verify imported catalog exists ===
h "Test 9: Verify imported catalog"
CODE=$(curl -s -o /tmp/imported-cat.json -w "%{http_code}" "$API/api/data/v1/catalogs/import-test-cat" -H 'X-User-Role: Admin')
if [ "$CODE" = "200" ]; then
  pass "Imported catalog accessible"
else
  fail "Imported catalog accessible" "got $CODE"
fi

IMP_VS=$(jq -r '.validation_status' /tmp/imported-cat.json)
if [ "$IMP_VS" = "draft" ]; then
  pass "Imported catalog is draft status"
else
  fail "Imported catalog is draft status" "got $IMP_VS"
fi

# === Test 10: Import requires Admin ===
h "Test 10: Import requires Admin"
CODE=$(curl -s -o /dev/null -w "%{http_code}" \
  "$API/api/data/v1/catalogs/import" \
  -H 'X-User-Role: RW' -H 'Content-Type: application/json' \
  -d '{}')
if [ "$CODE" = "403" ]; then
  pass "Import denied for RW role"
else
  fail "Import denied for RW role" "got $CODE"
fi

# === Test 10b: Import rejects non-catalog JSON ===
h "Test 10b: Import rejects non-catalog JSON"
CODE=$(curl -s -o /tmp/noncatalog-resp.json -w "%{http_code}" \
  "$API/api/data/v1/catalogs/import" \
  -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d '{"data":{"name":"not-a-catalog","version":"1.0"}}')
if [ "$CODE" = "400" ]; then
  pass "Non-catalog JSON returns 400"
else
  fail "Non-catalog JSON returns 400" "got $CODE — $(cat /tmp/noncatalog-resp.json)"
fi

CODE=$(curl -s -o /tmp/noversion-resp.json -w "%{http_code}" \
  "$API/api/data/v1/catalogs/import" \
  -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d '{"data":{"format_version":"1.0","catalog":{"name":"test"},"catalog_version":{}}}')
if [ "$CODE" = "400" ]; then
  pass "Missing CV label returns 400"
else
  fail "Missing CV label returns 400" "got $CODE — $(cat /tmp/noversion-resp.json)"
fi

CODE=$(curl -s -o /tmp/badname-resp.json -w "%{http_code}" \
  "$API/api/data/v1/catalogs/import" \
  -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d '{"data":{"format_version":"1.0","catalog":{"name":"INVALID!"},"catalog_version":{"label":"v1"},"entity_types":[]}}')
if [ "$CODE" = "400" ]; then
  pass "Invalid catalog name returns 400"
else
  fail "Invalid catalog name returns 400" "got $CODE — $(cat /tmp/badname-resp.json)"
fi

# === Test 11: Invalid dry_run param ===
h "Test 11: Invalid dry_run param"
CODE=$(curl -s -o /dev/null -w "%{http_code}" \
  "$API/api/data/v1/catalogs/import?dry_run=invalid" \
  -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d '{}')
if [ "$CODE" = "400" ]; then
  pass "Invalid dry_run param returns 400"
else
  fail "Invalid dry_run param returns 400" "got $CODE"
fi

# === Test 12: Import with rename_map ===
h "Test 12: Import with rename_map"
RENAME_REQ=$(jq -n --arg lbl "${P}-renamed-v1-$SUFFIX" \
  --arg s "${P}-server" --arg g "${P}-guard" --arg t "${P}-tool" \
  --arg rs "renamed-${P}-server" --arg rg "renamed-${P}-guard" --arg rt "renamed-${P}-tool" \
  --slurpfile data /tmp/import-data.json '{
  catalog_name: "renamed-cat",
  catalog_version_label: $lbl,
  rename_map: {
    entity_types: {($s): $rs, ($g): $rg, ($t): $rt}
  },
  data: $data[0]
}')

CODE=$(curl -s -o /tmp/rename-resp.json -w "%{http_code}" \
  "$API/api/data/v1/catalogs/import" \
  -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d "$RENAME_REQ")

if [ "$CODE" = "201" ]; then
  pass "Import with rename_map returns 201"
else
  fail "Import with rename_map returns 201" "got $CODE — $(cat /tmp/rename-resp.json)"
fi

# === Test 13: Export → import preserves contained instances ===
h "Test 13: Export → import preserves contained instances"
# Verify export has contained tool nested under server
TOOL_IN_EXPORT=$(jq --arg et "${P}-server" \
  '.instances[] | select(.entity_type==$et) | .tools | length' /tmp/export-resp.json 2>/dev/null || echo 0)
if [ "$TOOL_IN_EXPORT" -ge 1 ] 2>/dev/null; then
  pass "Export nests contained tool under server ($TOOL_IN_EXPORT)"
else
  fail "Export nests contained tool under server" "found $TOOL_IN_EXPORT tools in export"
fi

# Import the exported data into a new catalog and verify children
CONTAIN_REQ=$(jq -n --arg lbl "${P}-contain-v1-$SUFFIX" \
  --arg s "${P}-server" --arg g "${P}-guard" --arg t "${P}-tool" \
  --slurpfile data /tmp/import-data.json '{
  catalog_name: "contain-test-cat",
  catalog_version_label: $lbl,
  reuse_existing: [$s, $g, $t],
  data: $data[0]
}')
CODE=$(curl -s -o /tmp/contain-resp.json -w "%{http_code}" \
  "$API/api/data/v1/catalogs/import" \
  -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d "$CONTAIN_REQ")

if [ "$CODE" = "201" ]; then
  pass "Import with containment returns 201"
else
  fail "Import with containment returns 201" "got $CODE — $(cat /tmp/contain-resp.json)"
fi

CONTAIN_INST=$(jq -r '.instances_created // 0' /tmp/contain-resp.json)
if [ "$CONTAIN_INST" -ge 3 ] 2>/dev/null; then
  pass "Import created parent + child instances ($CONTAIN_INST total)"
else
  fail "Import created parent + child instances" "got $CONTAIN_INST instances (expected ≥3: server+tool+guard)"
fi

# Verify child instance exists via API
IMPORTED_SERVER_ID=$(curl -s "$API/api/data/v1/catalogs/contain-test-cat/${P}-server" -H 'X-User-Role: Admin' | jq -r '.items[0].id // empty')
if [ -n "$IMPORTED_SERVER_ID" ]; then
  IMPORTED_TOOL_COUNT=$(curl -s "$API/api/data/v1/catalogs/contain-test-cat/${P}-server/$IMPORTED_SERVER_ID/${P}-tool" -H 'X-User-Role: Admin' | jq '.items | length')
  if [ "$IMPORTED_TOOL_COUNT" -ge 1 ] 2>/dev/null; then
    pass "Imported catalog has contained tool instance"
  else
    fail "Imported catalog has contained tool instance" "got $IMPORTED_TOOL_COUNT tools"
  fi
else
  fail "Imported catalog has contained tool instance" "server instance not found in imported catalog"
fi

# Verify reused entity types still have their associations
REUSE_EXPORT_CODE=$(curl -s -o /tmp/reuse-export.json -w "%{http_code}" "$API/api/data/v1/catalogs/contain-test-cat/export" -H 'X-User-Role: Admin')
if [ "$REUSE_EXPORT_CODE" = "200" ]; then
  REUSE_ASSOC=$(jq -r --arg et "${P}-server" \
    '[.entity_types[] | select(.name==$et) | .associations[].type] | sort | join(",")' /tmp/reuse-export.json)
  if echo "$REUSE_ASSOC" | grep -q "containment" && echo "$REUSE_ASSOC" | grep -q "directional"; then
    pass "Reused server keeps containment AND directional associations ($REUSE_ASSOC)"
  else
    fail "Reused server keeps both association types" "got: $REUSE_ASSOC"
  fi
fi

# === Test 14: Import with NEW entity types preserves ALL association types ===
h "Test 14: Import with new entity types preserves associations"
# Import with rename_map — forces creation of new entity types (not reuse)
ASSOC_REQ=$(jq -n --arg lbl "${P}-assoc-v1-$SUFFIX" \
  --arg s "${P}-server" --arg g "${P}-guard" --arg t "${P}-tool" \
  --arg rs "assoc-${P}-server" --arg rg "assoc-${P}-guard" --arg rt "assoc-${P}-tool" \
  --slurpfile data /tmp/import-data.json '{
  catalog_name: "assoc-test-cat",
  catalog_version_label: $lbl,
  rename_map: { entity_types: {($s): $rs, ($g): $rg, ($t): $rt} },
  data: $data[0]
}')
CODE=$(curl -s -o /tmp/assoc-resp.json -w "%{http_code}" \
  "$API/api/data/v1/catalogs/import" \
  -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d "$ASSOC_REQ")
if [ "$CODE" = "201" ]; then
  pass "Import with renamed entity types returns 201"
else
  fail "Import with renamed entity types returns 201" "got $CODE — $(cat /tmp/assoc-resp.json)"
fi

# Re-export and verify associations on the newly created entity types
CODE=$(curl -s -o /tmp/assoc-export.json -w "%{http_code}" "$API/api/data/v1/catalogs/assoc-test-cat/export" -H 'X-User-Role: Admin')
if [ "$CODE" = "200" ]; then
  ASSOC_TYPES=$(jq -r --arg et "assoc-${P}-server" '[.entity_types[] | select(.name==$et) | (.associations // [])[].type] | sort | join(",")' /tmp/assoc-export.json)
  if echo "$ASSOC_TYPES" | grep -q "containment" && echo "$ASSOC_TYPES" | grep -q "directional" && echo "$ASSOC_TYPES" | grep -q "bidirectional"; then
    pass "Newly created server has all 3 association types ($ASSOC_TYPES)"
  else
    fail "Newly created server has all association types" "got: $ASSOC_TYPES"
  fi
  # Verify bidirectional link from reverse side survived import
  # The link was created guard→server, so it appears on the guard instance
  GUARD_LINKS=$(jq --arg et "assoc-${P}-guard" '[.instances[] | select(.entity_type==$et) | (.links // [])[]] | length' /tmp/assoc-export.json)
  if [ "$GUARD_LINKS" -ge 1 ] 2>/dev/null; then
    pass "Bidirectional link from reverse side survived import ($GUARD_LINKS)"
  else
    fail "Bidirectional link from reverse side survived import" "got $GUARD_LINKS links on guard"
  fi
  # Verify contained instances also survived with new entity types
  NEW_TOOL_COUNT=$(jq --arg et "assoc-${P}-server" '[.instances[] | select(.entity_type==$et) | (.tools // []) | length] | add // 0' /tmp/assoc-export.json)
  if [ "$NEW_TOOL_COUNT" -ge 1 ] 2>/dev/null; then
    pass "Newly created server has contained tool instances ($NEW_TOOL_COUNT)"
  else
    fail "Newly created server has contained tool instances" "got $NEW_TOOL_COUNT"
  fi
else
  fail "Re-export of assoc-test-cat" "got $CODE"
fi

# === Summary ===
h "Summary"
echo "  Total: $TOTAL, Passed: $PASS, Failed: $FAIL"
if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
