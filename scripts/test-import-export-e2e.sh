#!/usr/bin/env bash
# End-to-end export→import round-trip test
# Creates a realistic catalog with all features, exports, imports with prefix,
# and verifies every piece of data matches between original and imported catalogs.
#
# Usage: ./scripts/test-import-export-e2e.sh [API_BASE_URL]
set -uo pipefail

API="${1:-http://localhost:30080}"
META="$API/api/meta/v1"
DATA="$API/api/data/v1"
PASS=0; FAIL=0; TOTAL=0
SUFFIX=$(date +%s)
P="zz-e2e"  # prefix for test entity types
PREFIX="imp-"  # prefix for imported entity types

pass() { ((PASS++)); ((TOTAL++)); echo "  PASS: $1"; }
fail() { ((FAIL++)); ((TOTAL++)); echo "  FAIL: $1 — $2"; }
h() { echo ""; echo "=== $1 ==="; }

jpost() { curl -s -X POST "$1" -H 'X-User-Role: Admin' -H 'Content-Type: application/json' -d "$2"; }
jget() { curl -s "$1" -H 'X-User-Role: Admin'; }
jdel() { curl -s -X DELETE "$1" -H 'X-User-Role: SuperAdmin' -o /dev/null 2>/dev/null || true; }

ET_IDS=""
ORIG_CAT="${P}-orig-${SUFFIX}"
IMP_CAT="${P}-imported-${SUFFIX}"
ORIG_CV_LABEL="${P}-cv-${SUFFIX}"
IMP_CV_LABEL="${PREFIX}${P}-cv-${SUFFIX}"

cleanup() {
  h "Cleanup"
  for name in "$ORIG_CAT" "$IMP_CAT"; do
    jdel "$DATA/catalogs/$name"
  done
  # Clean CVs
  for lbl in "$ORIG_CV_LABEL" "$IMP_CV_LABEL"; do
    cvid=$(jget "$META/catalog-versions" | jq -r ".items[] | select(.version_label==\"$lbl\") | .id" 2>/dev/null)
    [ -n "$cvid" ] && curl -s -o /dev/null -X DELETE "$META/catalog-versions/$cvid" -H 'X-User-Role: Admin' 2>/dev/null || true
  done
  # Clean entity types
  for etid in ${ET_IDS:-}; do
    curl -s -o /dev/null -X DELETE "$META/entity-types/$etid" -H 'X-User-Role: Admin' 2>/dev/null || true
  done
  # Clean imported entity types (with prefix)
  for base in server tool guardrail model; do
    for pfx in "${P}-" "${PREFIX}${P}-"; do
      etid=$(jget "$META/entity-types?name=${pfx}${base}" | jq -r '.items[0].id // empty')
      [ -n "$etid" ] && curl -s -o /dev/null -X DELETE "$META/entity-types/$etid" -H 'X-User-Role: Admin' 2>/dev/null || true
    done
  done
  # Clean custom type definitions
  for tdname in "${P}-severity" "${P}-multiline"; do
    tdid=$(jget "$META/type-definitions" | jq -r ".items[] | select(.name==\"$tdname\") | .id // empty")
    [ -n "$tdid" ] && curl -s -o /dev/null -X DELETE "$META/type-definitions/$tdid" -H 'X-User-Role: Admin' 2>/dev/null || true
  done
  echo "  Done"
}
trap cleanup EXIT

# Pre-cleanup
cleanup 2>/dev/null

# ============================================================
# SETUP: Build a realistic schema with versioning quirks
# ============================================================
h "Setup: Create entity types with realistic versioning"

# Look up system type definition version IDs
STRING_TDV=$(jget "$META/type-definitions" | jq -r '.items[] | select(.name=="string") | .latest_version_id')
BOOL_TDV=$(jget "$META/type-definitions" | jq -r '.items[] | select(.name=="boolean") | .latest_version_id')
INT_TDV=$(jget "$META/type-definitions" | jq -r '.items[] | select(.name=="integer") | .latest_version_id')
URL_TDV=$(jget "$META/type-definitions" | jq -r '.items[] | select(.name=="url") | .latest_version_id')

# Create custom type definitions
SEV_TD=$(jpost "$META/type-definitions" '{"name":"'"${P}-severity"'","base_type":"enum","constraints":{"values":["low","medium","high","critical"]}}' | jq -r '.id')
ML_TD=$(jpost "$META/type-definitions" '{"name":"'"${P}-multiline"'","base_type":"string","constraints":{"multiline":true}}' | jq -r '.id')
SEV_TDV=$(jget "$META/type-definitions" | jq -r ".items[] | select(.name==\"${P}-severity\") | .latest_version_id")
ML_TDV=$(jget "$META/type-definitions" | jq -r ".items[] | select(.name==\"${P}-multiline\") | .latest_version_id")

# --- Entity type: server ---
# V1: create
SERVER_RESP=$(jpost "$META/entity-types" '{"name":"'"${P}-server"'","description":"MCP Server"}')
SERVER_ET=$(echo "$SERVER_RESP" | jq -r '.entity_type.id')
ET_IDS="$SERVER_ET"

# V2: add endpoint attribute (required)
jpost "$META/entity-types/$SERVER_ET/attributes" \
  "{\"name\":\"endpoint\",\"type_definition_version_id\":\"$URL_TDV\",\"required\":true}" > /dev/null

# V3: add optional description attribute (using custom multiline type)
jpost "$META/entity-types/$SERVER_ET/attributes" \
  "{\"name\":\"notes\",\"type_definition_version_id\":\"$ML_TDV\"}" > /dev/null

# --- Entity type: tool ---
# V1: create
TOOL_RESP=$(jpost "$META/entity-types" '{"name":"'"${P}-tool"'","description":"MCP Tool"}')
TOOL_ET=$(echo "$TOOL_RESP" | jq -r '.entity_type.id')
ET_IDS="$ET_IDS $TOOL_ET"

# V2: add attribute
jpost "$META/entity-types/$TOOL_ET/attributes" \
  "{\"name\":\"idempotent\",\"type_definition_version_id\":\"$BOOL_TDV\"}" > /dev/null

# --- Entity type: guardrail ---
# V1: create
GUARD_RESP=$(jpost "$META/entity-types" '{"name":"'"${P}-guardrail"'","description":"Safety guardrail"}')
GUARD_ET=$(echo "$GUARD_RESP" | jq -r '.entity_type.id')
ET_IDS="$ET_IDS $GUARD_ET"

# V2: add attribute using custom enum type
jpost "$META/entity-types/$GUARD_ET/attributes" \
  "{\"name\":\"severity\",\"type_definition_version_id\":\"$SEV_TDV\",\"required\":true}" > /dev/null

# V3: add another attribute (this means associations will be on V4+, not latest attr version)
jpost "$META/entity-types/$GUARD_ET/attributes" \
  "{\"name\":\"notes\",\"type_definition_version_id\":\"$ML_TDV\"}" > /dev/null

# --- Entity type: model ---
# V1: create
MODEL_RESP=$(jpost "$META/entity-types" '{"name":"'"${P}-model"'","description":"AI Model"}')
MODEL_ET=$(echo "$MODEL_RESP" | jq -r '.entity_type.id')
ET_IDS="$ET_IDS $MODEL_ET"

# V2: add attribute
jpost "$META/entity-types/$MODEL_ET/attributes" \
  "{\"name\":\"version\",\"type_definition_version_id\":\"$STRING_TDV\"}" > /dev/null

# V3: add another attribute (so model ends with attribute as last version change)
jpost "$META/entity-types/$MODEL_ET/attributes" \
  "{\"name\":\"context-window\",\"type_definition_version_id\":\"$INT_TDV\"}" > /dev/null

# --- Add associations (these create NEW versions on the source ET) ---

# Containment: server contains tool (server gets a new version)
jpost "$META/entity-types/$SERVER_ET/associations" \
  "{\"name\":\"tools\",\"type\":\"containment\",\"target_entity_type_id\":\"$TOOL_ET\",\"source_cardinality\":\"1\",\"target_cardinality\":\"0..n\"}" > /dev/null

# Directional: server → model (server gets another new version)
jpost "$META/entity-types/$SERVER_ET/associations" \
  "{\"name\":\"uses-model\",\"type\":\"directional\",\"target_entity_type_id\":\"$MODEL_ET\",\"source_cardinality\":\"0..n\",\"target_cardinality\":\"0..1\"}" > /dev/null

# Directional: tool → model (tool gets a new version — tests links on contained instances)
jpost "$META/entity-types/$TOOL_ET/associations" \
  "{\"name\":\"trained-on\",\"type\":\"directional\",\"target_entity_type_id\":\"$MODEL_ET\",\"source_cardinality\":\"0..n\",\"target_cardinality\":\"0..1\"}" > /dev/null

# Bidirectional: guardrail ↔ tool (guardrail gets a new version — V4, AFTER the attribute additions)
jpost "$META/entity-types/$GUARD_ET/associations" \
  "{\"name\":\"pre-execute\",\"type\":\"bidirectional\",\"target_entity_type_id\":\"$TOOL_ET\",\"source_cardinality\":\"0..n\",\"target_cardinality\":\"0..n\"}" > /dev/null

# Directional: guardrail → model (guardrail V5)
jpost "$META/entity-types/$GUARD_ET/associations" \
  "{\"name\":\"uses-model\",\"type\":\"directional\",\"target_entity_type_id\":\"$MODEL_ET\",\"source_cardinality\":\"0..n\",\"target_cardinality\":\"0..1\"}" > /dev/null

# --- NOW add one more attribute to server (creating yet another version AFTER associations) ---
# This is the key scenario: the latest version has an attribute change, not an association change.
# Links will reference association IDs from older versions.
jpost "$META/entity-types/$SERVER_ET/attributes" \
  "{\"name\":\"containerized\",\"type_definition_version_id\":\"$BOOL_TDV\"}" > /dev/null

# --- Get latest ETV IDs for CV pinning ---
SERVER_ETV=$(jget "$META/entity-types/$SERVER_ET/versions" | jq -r '.items | sort_by(.version) | last | .id')
TOOL_ETV=$(jget "$META/entity-types/$TOOL_ET/versions" | jq -r '.items | sort_by(.version) | last | .id')
GUARD_ETV=$(jget "$META/entity-types/$GUARD_ET/versions" | jq -r '.items | sort_by(.version) | last | .id')
MODEL_ETV=$(jget "$META/entity-types/$MODEL_ET/versions" | jq -r '.items | sort_by(.version) | last | .id')

echo "  Entity types: server=$SERVER_ET, tool=$TOOL_ET, guardrail=$GUARD_ET, model=$MODEL_ET"

# Create CV pinning ALL entity types at latest versions
CV_ID=$(jpost "$META/catalog-versions" \
  "{\"version_label\":\"$ORIG_CV_LABEL\",\"pins\":[{\"entity_type_version_id\":\"$SERVER_ETV\"},{\"entity_type_version_id\":\"$TOOL_ETV\"},{\"entity_type_version_id\":\"$GUARD_ETV\"},{\"entity_type_version_id\":\"$MODEL_ETV\"}]}" | jq -r '.id')

echo "  CV: $CV_ID ($ORIG_CV_LABEL)"

# Create catalog
jpost "$DATA/catalogs" "{\"name\":\"$ORIG_CAT\",\"catalog_version_id\":\"$CV_ID\"}" > /dev/null

# ============================================================
# Create instances with all relationship types
# ============================================================
h "Setup: Create instances"

# Root instances
jpost "$DATA/catalogs/$ORIG_CAT/${P}-server" \
  '{"name":"github","description":"GitHub MCP","attributes":{"endpoint":"https://github.example.com","notes":"Primary server","containerized":"true"}}' > /dev/null

jpost "$DATA/catalogs/$ORIG_CAT/${P}-server" \
  '{"name":"jira","description":"Jira MCP","attributes":{"endpoint":"https://jira.example.com"}}' > /dev/null

jpost "$DATA/catalogs/$ORIG_CAT/${P}-guardrail" \
  '{"name":"pii-filter","attributes":{"severity":"high"}}' > /dev/null

jpost "$DATA/catalogs/$ORIG_CAT/${P}-guardrail" \
  '{"name":"audit-log","attributes":{"severity":"medium","notes":"Logs all actions"}}' > /dev/null

jpost "$DATA/catalogs/$ORIG_CAT/${P}-model" \
  '{"name":"gpt-4o","attributes":{"version":"2024-05","context-window":"128000"}}' > /dev/null

# Contained instances (tools under servers)
GH_ID=$(jget "$DATA/catalogs/$ORIG_CAT/${P}-server" | jq -r '.items[] | select(.name=="github") | .id')
JIRA_ID=$(jget "$DATA/catalogs/$ORIG_CAT/${P}-server" | jq -r '.items[] | select(.name=="jira") | .id')

jpost "$DATA/catalogs/$ORIG_CAT/${P}-server/$GH_ID/${P}-tool" \
  '{"name":"create-pr","attributes":{"idempotent":"false"}}' > /dev/null
jpost "$DATA/catalogs/$ORIG_CAT/${P}-server/$GH_ID/${P}-tool" \
  '{"name":"get-issue","attributes":{"idempotent":"true"}}' > /dev/null
jpost "$DATA/catalogs/$ORIG_CAT/${P}-server/$JIRA_ID/${P}-tool" \
  '{"name":"add-watcher"}' > /dev/null

# Links: directional (server → model)
MODEL_INST_ID=$(jget "$DATA/catalogs/$ORIG_CAT/${P}-model" | jq -r '.items[0].id')
jpost "$DATA/catalogs/$ORIG_CAT/${P}-server/$GH_ID/links" \
  "{\"target_instance_id\":\"$MODEL_INST_ID\",\"association_name\":\"uses-model\"}" > /dev/null

# Links: bidirectional (guardrail ↔ tool) — created from guardrail side
PII_ID=$(jget "$DATA/catalogs/$ORIG_CAT/${P}-guardrail" | jq -r '.items[] | select(.name=="pii-filter") | .id')
CREATE_PR_ID=$(jget "$DATA/catalogs/$ORIG_CAT/${P}-server/$GH_ID/${P}-tool" | jq -r '.items[] | select(.name=="create-pr") | .id')
ADD_WATCHER_ID=$(jget "$DATA/catalogs/$ORIG_CAT/${P}-server/$JIRA_ID/${P}-tool" | jq -r '.items[] | select(.name=="add-watcher") | .id')

jpost "$DATA/catalogs/$ORIG_CAT/${P}-guardrail/$PII_ID/links" \
  "{\"target_instance_id\":\"$CREATE_PR_ID\",\"association_name\":\"pre-execute\"}" > /dev/null
jpost "$DATA/catalogs/$ORIG_CAT/${P}-guardrail/$PII_ID/links" \
  "{\"target_instance_id\":\"$ADD_WATCHER_ID\",\"association_name\":\"pre-execute\"}" > /dev/null

# Links: directional (guardrail → model)
jpost "$DATA/catalogs/$ORIG_CAT/${P}-guardrail/$PII_ID/links" \
  "{\"target_instance_id\":\"$MODEL_INST_ID\",\"association_name\":\"uses-model\"}" > /dev/null

# Links: directional on CONTAINED instance (tool → model) — tests links on children survive
GET_ISSUE_ID=$(jget "$DATA/catalogs/$ORIG_CAT/${P}-server/$GH_ID/${P}-tool" | jq -r '.items[] | select(.name=="get-issue") | .id')
jpost "$DATA/catalogs/$ORIG_CAT/${P}-tool/$GET_ISSUE_ID/links" \
  "{\"target_instance_id\":\"$MODEL_INST_ID\",\"association_name\":\"trained-on\"}" > /dev/null

echo "  Created: 2 servers, 3 tools (contained), 2 guardrails, 1 model"
echo "  Links: 1 server→model, 2 guardrail↔tool, 1 guardrail→model, 1 tool→model (contained)"

# ============================================================
# Simulate real-world usage: modify entity types AFTER creating links
# This creates new ETVs with new association IDs. Re-pin CV to latest.
# Links still reference OLD association IDs → triggers export version mismatch bug.
# ============================================================
h "Setup: Update entity types after links (triggers version mismatch)"

# Add new attributes to guardrail and tool — creates new ETVs with copied associations (new IDs)
NEW_GUARD_ETV=$(jpost "$META/entity-types/$GUARD_ET/attributes" \
  "{\"name\":\"post-update-attr\",\"type_definition_version_id\":\"$STRING_TDV\"}" | jq -r '.id')
NEW_TOOL_ETV=$(jpost "$META/entity-types/$TOOL_ET/attributes" \
  "{\"name\":\"post-update-attr\",\"type_definition_version_id\":\"$STRING_TDV\"}" | jq -r '.id')

echo "  New ETVs: guardrail=$NEW_GUARD_ETV, tool=$NEW_TOOL_ETV"

# Find pin IDs and update to new ETVs
PINS_JSON=$(jget "$META/catalog-versions/$CV_ID/pins")
GUARD_PIN_ID=$(echo "$PINS_JSON" | jq -r --arg etv "$GUARD_ETV" '.items[] | select(.entity_type_version_id==$etv) | .pin_id')
TOOL_PIN_ID=$(echo "$PINS_JSON" | jq -r --arg etv "$TOOL_ETV" '.items[] | select(.entity_type_version_id==$etv) | .pin_id')

echo "  Pin IDs: guardrail=$GUARD_PIN_ID, tool=$TOOL_PIN_ID"

curl -s -o /dev/null -X PUT "$META/catalog-versions/$CV_ID/pins/$GUARD_PIN_ID" \
  -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d "{\"entity_type_version_id\":\"$NEW_GUARD_ETV\"}"

curl -s -o /dev/null -X PUT "$META/catalog-versions/$CV_ID/pins/$TOOL_PIN_ID" \
  -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d "{\"entity_type_version_id\":\"$NEW_TOOL_ETV\"}"

echo "  Re-pinned CV. Links reference OLD association IDs; CV now pins NEW ETVs"

# ============================================================
# EXPORT
# ============================================================
h "Export original catalog"

CODE=$(curl -s -o /tmp/e2e-export.json -w "%{http_code}" "$DATA/catalogs/$ORIG_CAT/export" -H 'X-User-Role: Admin')
if [ "$CODE" = "200" ]; then
  pass "Export returns 200"
else
  fail "Export returns 200" "got $CODE"
  echo "FATAL: Cannot continue without export"
  exit 1
fi

# ============================================================
# IMPORT with prefix
# ============================================================
h "Import with prefix '${PREFIX}'"

IMP_REQ=$(jq -n --arg cat "$IMP_CAT" --arg lbl "$IMP_CV_LABEL" --arg pfx "$PREFIX" \
  --slurpfile data /tmp/e2e-export.json '{
  catalog_name: $cat,
  catalog_version_label: $lbl,
  rename_map: { entity_types: (
    [$data[0].entity_types[].name] | map({key: ., value: ($pfx + .)}) | from_entries
  ), type_definitions: (
    [$data[0].type_definitions[].name] | map({key: ., value: ($pfx + .)}) | from_entries
  )},
  data: $data[0]
}')

CODE=$(curl -s -o /tmp/e2e-import.json -w "%{http_code}" \
  "$DATA/catalogs/import" \
  -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d "$IMP_REQ")

if [ "$CODE" = "201" ]; then
  pass "Import returns 201"
else
  fail "Import returns 201" "got $CODE — $(cat /tmp/e2e-import.json)"
  echo "FATAL: Cannot continue without import"
  exit 1
fi

IMP_RESULT=$(cat /tmp/e2e-import.json)
IMP_TYPES=$(echo "$IMP_RESULT" | jq -r '.types_created')
IMP_INST=$(echo "$IMP_RESULT" | jq -r '.instances_created')
IMP_LINKS=$(echo "$IMP_RESULT" | jq -r '.links_created')
echo "  Import result: $IMP_TYPES types, $IMP_INST instances, $IMP_LINKS links"

# ============================================================
# RE-EXPORT imported catalog for comparison
# ============================================================
h "Re-export imported catalog"

CODE=$(curl -s -o /tmp/e2e-reimport-export.json -w "%{http_code}" "$DATA/catalogs/$IMP_CAT/export" -H 'X-User-Role: Admin')
if [ "$CODE" = "200" ]; then
  pass "Re-export returns 200"
else
  fail "Re-export returns 200" "got $CODE"
  echo "FATAL: Cannot continue without re-export"
  exit 1
fi

# ============================================================
# COMPARE: Entity types
# ============================================================
h "Compare: Entity types"

ORIG_ET_COUNT=$(jq '.entity_types | length' /tmp/e2e-export.json)
IMP_ET_COUNT=$(jq '.entity_types | length' /tmp/e2e-reimport-export.json)
if [ "$ORIG_ET_COUNT" = "$IMP_ET_COUNT" ]; then
  pass "Entity type count matches ($ORIG_ET_COUNT)"
else
  fail "Entity type count" "orig=$ORIG_ET_COUNT, imported=$IMP_ET_COUNT"
fi

# Compare each entity type's attributes and associations
for ET_NAME in $(jq -r '.entity_types[].name' /tmp/e2e-export.json); do
  IMP_ET_NAME="${PREFIX}${ET_NAME}"

  # Attribute count
  ORIG_ATTRS=$(jq --arg n "$ET_NAME" '[.entity_types[] | select(.name==$n) | (.attributes // [])[]] | length' /tmp/e2e-export.json)
  IMP_ATTRS=$(jq --arg n "$IMP_ET_NAME" '[.entity_types[] | select(.name==$n) | (.attributes // [])[]] | length' /tmp/e2e-reimport-export.json)
  if [ "$ORIG_ATTRS" = "$IMP_ATTRS" ]; then
    pass "ET $ET_NAME: attribute count matches ($ORIG_ATTRS)"
  else
    fail "ET $ET_NAME: attribute count" "orig=$ORIG_ATTRS, imported=$IMP_ATTRS"
  fi

  # Association count
  ORIG_ASSOCS=$(jq --arg n "$ET_NAME" '[.entity_types[] | select(.name==$n) | (.associations // [])[]] | length' /tmp/e2e-export.json)
  IMP_ASSOCS=$(jq --arg n "$IMP_ET_NAME" '[.entity_types[] | select(.name==$n) | (.associations // [])[]] | length' /tmp/e2e-reimport-export.json)
  if [ "$ORIG_ASSOCS" = "$IMP_ASSOCS" ]; then
    pass "ET $ET_NAME: association count matches ($ORIG_ASSOCS)"
  else
    fail "ET $ET_NAME: association count" "orig=$ORIG_ASSOCS, imported=$IMP_ASSOCS"
  fi

  # Association types (sorted)
  ORIG_ASSOC_TYPES=$(jq -r --arg n "$ET_NAME" '[.entity_types[] | select(.name==$n) | (.associations // [])[].type] | sort | join(",")' /tmp/e2e-export.json)
  IMP_ASSOC_TYPES=$(jq -r --arg n "$IMP_ET_NAME" '[.entity_types[] | select(.name==$n) | (.associations // [])[].type] | sort | join(",")' /tmp/e2e-reimport-export.json)
  if [ "$ORIG_ASSOC_TYPES" = "$IMP_ASSOC_TYPES" ]; then
    pass "ET $ET_NAME: association types match ($ORIG_ASSOC_TYPES)"
  else
    fail "ET $ET_NAME: association types" "orig=$ORIG_ASSOC_TYPES, imported=$IMP_ASSOC_TYPES"
  fi
done

# ============================================================
# COMPARE: Type definitions
# ============================================================
h "Compare: Type definitions"

ORIG_TD_COUNT=$(jq '.type_definitions | length' /tmp/e2e-export.json)
IMP_TD_COUNT=$(jq '.type_definitions | length' /tmp/e2e-reimport-export.json)
if [ "$ORIG_TD_COUNT" = "$IMP_TD_COUNT" ]; then
  pass "Type definition count matches ($ORIG_TD_COUNT)"
else
  fail "Type definition count" "orig=$ORIG_TD_COUNT, imported=$IMP_TD_COUNT"
fi

for TD_NAME in $(jq -r '.type_definitions[].name' /tmp/e2e-export.json); do
  IMP_TD_NAME="${PREFIX}${TD_NAME}"

  # base_type
  ORIG_BT=$(jq -r --arg n "$TD_NAME" '.type_definitions[] | select(.name==$n) | .base_type' /tmp/e2e-export.json)
  IMP_BT=$(jq -r --arg n "$IMP_TD_NAME" '.type_definitions[] | select(.name==$n) | .base_type' /tmp/e2e-reimport-export.json)
  if [ "$ORIG_BT" = "$IMP_BT" ]; then
    pass "TD $TD_NAME: base_type matches ($ORIG_BT)"
  else
    fail "TD $TD_NAME: base_type" "orig=$ORIG_BT, imported=$IMP_BT"
  fi

  # description
  ORIG_DESC=$(jq -r --arg n "$TD_NAME" '.type_definitions[] | select(.name==$n) | .description' /tmp/e2e-export.json)
  IMP_DESC=$(jq -r --arg n "$IMP_TD_NAME" '.type_definitions[] | select(.name==$n) | .description' /tmp/e2e-reimport-export.json)
  if [ "$ORIG_DESC" = "$IMP_DESC" ]; then
    pass "TD $TD_NAME: description matches"
  else
    fail "TD $TD_NAME: description" "orig=$ORIG_DESC, imported=$IMP_DESC"
  fi

  # constraints (sorted JSON)
  ORIG_CONS=$(jq -cS --arg n "$TD_NAME" '.type_definitions[] | select(.name==$n) | .constraints // {}' /tmp/e2e-export.json)
  IMP_CONS=$(jq -cS --arg n "$IMP_TD_NAME" '.type_definitions[] | select(.name==$n) | .constraints // {}' /tmp/e2e-reimport-export.json)
  if [ "$ORIG_CONS" = "$IMP_CONS" ]; then
    pass "TD $TD_NAME: constraints match"
  else
    fail "TD $TD_NAME: constraints" "orig=$ORIG_CONS, imported=$IMP_CONS"
  fi
done

# Compare each entity type's attribute details (name, type_definition, required, ordinal, description)
for ET_NAME in $(jq -r '.entity_types[].name' /tmp/e2e-export.json); do
  IMP_ET_NAME="${PREFIX}${ET_NAME}"
  for ATTR_NAME in $(jq -r --arg n "$ET_NAME" '.entity_types[] | select(.name==$n) | (.attributes // [])[].name' /tmp/e2e-export.json); do
    # required flag
    ORIG_REQ=$(jq -r --arg n "$ET_NAME" --arg a "$ATTR_NAME" '.entity_types[] | select(.name==$n) | (.attributes // [])[] | select(.name==$a) | .required' /tmp/e2e-export.json)
    IMP_REQ=$(jq -r --arg n "$IMP_ET_NAME" --arg a "$ATTR_NAME" '.entity_types[] | select(.name==$n) | (.attributes // [])[] | select(.name==$a) | .required' /tmp/e2e-reimport-export.json)
    if [ "$ORIG_REQ" = "$IMP_REQ" ]; then
      pass "ET $ET_NAME attr $ATTR_NAME: required=$ORIG_REQ matches"
    else
      fail "ET $ET_NAME attr $ATTR_NAME: required" "orig=$ORIG_REQ, imported=$IMP_REQ"
    fi

    # type_definition (with prefix applied)
    ORIG_TD=$(jq -r --arg n "$ET_NAME" --arg a "$ATTR_NAME" '.entity_types[] | select(.name==$n) | (.attributes // [])[] | select(.name==$a) | .type_definition' /tmp/e2e-export.json)
    IMP_TD=$(jq -r --arg n "$IMP_ET_NAME" --arg a "$ATTR_NAME" '.entity_types[] | select(.name==$n) | (.attributes // [])[] | select(.name==$a) | .type_definition' /tmp/e2e-reimport-export.json)
    # System types keep their name; custom types get prefix
    EXPECTED_TD="$ORIG_TD"
    if jq -e --arg n "$ORIG_TD" '.type_definitions[] | select(.name==$n)' /tmp/e2e-export.json > /dev/null 2>&1; then
      EXPECTED_TD="${PREFIX}${ORIG_TD}"
    fi
    if [ "$IMP_TD" = "$EXPECTED_TD" ]; then
      pass "ET $ET_NAME attr $ATTR_NAME: type_definition matches ($IMP_TD)"
    else
      fail "ET $ET_NAME attr $ATTR_NAME: type_definition" "expected=$EXPECTED_TD, got=$IMP_TD"
    fi

    # ordinal
    ORIG_ORD=$(jq -r --arg n "$ET_NAME" --arg a "$ATTR_NAME" '.entity_types[] | select(.name==$n) | (.attributes // [])[] | select(.name==$a) | .ordinal' /tmp/e2e-export.json)
    IMP_ORD=$(jq -r --arg n "$IMP_ET_NAME" --arg a "$ATTR_NAME" '.entity_types[] | select(.name==$n) | (.attributes // [])[] | select(.name==$a) | .ordinal' /tmp/e2e-reimport-export.json)
    if [ "$ORIG_ORD" = "$IMP_ORD" ]; then
      pass "ET $ET_NAME attr $ATTR_NAME: ordinal=$ORIG_ORD matches"
    else
      fail "ET $ET_NAME attr $ATTR_NAME: ordinal" "orig=$ORIG_ORD, imported=$IMP_ORD"
    fi
  done
done

# ============================================================
# COMPARE: Instances (root-level)
# ============================================================
h "Compare: Root instances"

ORIG_INST_COUNT=$(jq '.instances | length' /tmp/e2e-export.json)
IMP_INST_COUNT=$(jq '.instances | length' /tmp/e2e-reimport-export.json)
if [ "$ORIG_INST_COUNT" = "$IMP_INST_COUNT" ]; then
  pass "Root instance count matches ($ORIG_INST_COUNT)"
else
  fail "Root instance count" "orig=$ORIG_INST_COUNT, imported=$IMP_INST_COUNT"
fi

# Compare each root instance
for INST_NAME in $(jq -r '.instances[].name' /tmp/e2e-export.json); do
  ORIG_ET_TYPE=$(jq -r --arg n "$INST_NAME" '.instances[] | select(.name==$n) | .entity_type' /tmp/e2e-export.json)
  IMP_ET_TYPE=$(jq -r --arg n "$INST_NAME" '.instances[] | select(.name==$n) | .entity_type' /tmp/e2e-reimport-export.json)
  EXPECTED_IMP_ET="${PREFIX}${ORIG_ET_TYPE}"

  if [ "$IMP_ET_TYPE" = "$EXPECTED_IMP_ET" ]; then
    pass "Instance $INST_NAME: entity type matches ($IMP_ET_TYPE)"
  else
    fail "Instance $INST_NAME: entity type" "expected=$EXPECTED_IMP_ET, got=$IMP_ET_TYPE"
  fi

  # Compare attributes (sorted JSON keys)
  ORIG_ATTRS=$(jq -cS --arg n "$INST_NAME" '.instances[] | select(.name==$n) | .attributes // {}' /tmp/e2e-export.json)
  IMP_ATTRS=$(jq -cS --arg n "$INST_NAME" '.instances[] | select(.name==$n) | .attributes // {}' /tmp/e2e-reimport-export.json)
  if [ "$ORIG_ATTRS" = "$IMP_ATTRS" ]; then
    pass "Instance $INST_NAME: attributes match"
  else
    fail "Instance $INST_NAME: attributes" "orig=$ORIG_ATTRS, imported=$IMP_ATTRS"
  fi

  # Compare links (count and association names)
  ORIG_LINKS=$(jq -r --arg n "$INST_NAME" '[.instances[] | select(.name==$n) | (.links // [])[].association] | sort | join(",")' /tmp/e2e-export.json)
  IMP_LINKS=$(jq -r --arg n "$INST_NAME" '[.instances[] | select(.name==$n) | (.links // [])[].association] | sort | join(",")' /tmp/e2e-reimport-export.json)
  if [ "$ORIG_LINKS" = "$IMP_LINKS" ]; then
    if [ -n "$ORIG_LINKS" ]; then
      pass "Instance $INST_NAME: links match ($ORIG_LINKS)"
    else
      pass "Instance $INST_NAME: no links (correct)"
    fi
  else
    fail "Instance $INST_NAME: links" "orig=$ORIG_LINKS, imported=$IMP_LINKS"
  fi

  # Compare link targets (with renamed entity types)
  ORIG_LINK_TARGETS=$(jq -r --arg n "$INST_NAME" '[.instances[] | select(.name==$n) | (.links // [])[] | "\(.association)→\(.target_type)/\(.target_name)"] | sort | join(";")' /tmp/e2e-export.json)
  IMP_LINK_TARGETS=$(jq -r --arg n "$INST_NAME" --arg pfx "$PREFIX" '[.instances[] | select(.name==$n) | (.links // [])[] | "\(.association)→\(.target_type)/\(.target_name)"] | sort | join(";")' /tmp/e2e-reimport-export.json)
  # Imported targets should have PREFIX on entity type names
  EXPECTED_TARGETS=$(echo "$ORIG_LINK_TARGETS" | sed "s|→${P}-|→${PREFIX}${P}-|g")
  if [ "$IMP_LINK_TARGETS" = "$EXPECTED_TARGETS" ]; then
    if [ -n "$ORIG_LINK_TARGETS" ]; then
      pass "Instance $INST_NAME: link targets match"
    fi
  else
    fail "Instance $INST_NAME: link targets" "expected=$EXPECTED_TARGETS, got=$IMP_LINK_TARGETS"
  fi

  # Compare contained children
  ORIG_CHILDREN=$(jq -r --arg n "$INST_NAME" '.instances[] | select(.name==$n) | keys_unsorted[] | select(. != "entity_type" and . != "name" and . != "description" and . != "attributes" and . != "links")' /tmp/e2e-export.json 2>/dev/null || true)
  for CHILD_KEY in $ORIG_CHILDREN; do
    ORIG_CHILD_COUNT=$(jq --arg n "$INST_NAME" --arg k "$CHILD_KEY" '.instances[] | select(.name==$n) | .[$k] | length' /tmp/e2e-export.json)
    IMP_CHILD_COUNT=$(jq --arg n "$INST_NAME" --arg k "$CHILD_KEY" '.instances[] | select(.name==$n) | .[$k] | length' /tmp/e2e-reimport-export.json)
    if [ "$ORIG_CHILD_COUNT" = "$IMP_CHILD_COUNT" ]; then
      pass "Instance $INST_NAME/$CHILD_KEY: child count matches ($ORIG_CHILD_COUNT)"
    else
      fail "Instance $INST_NAME/$CHILD_KEY: child count" "orig=$ORIG_CHILD_COUNT, imported=$IMP_CHILD_COUNT"
    fi

    # Compare each child's attributes AND links
    for CHILD_NAME in $(jq -r --arg n "$INST_NAME" --arg k "$CHILD_KEY" '.instances[] | select(.name==$n) | .[$k][].name' /tmp/e2e-export.json); do
      ORIG_CHILD_ATTRS=$(jq -cS --arg n "$INST_NAME" --arg k "$CHILD_KEY" --arg cn "$CHILD_NAME" '.instances[] | select(.name==$n) | .[$k][] | select(.name==$cn) | .attributes // {}' /tmp/e2e-export.json)
      IMP_CHILD_ATTRS=$(jq -cS --arg n "$INST_NAME" --arg k "$CHILD_KEY" --arg cn "$CHILD_NAME" '.instances[] | select(.name==$n) | .[$k][] | select(.name==$cn) | .attributes // {}' /tmp/e2e-reimport-export.json)
      if [ "$ORIG_CHILD_ATTRS" = "$IMP_CHILD_ATTRS" ]; then
        pass "Instance $INST_NAME/$CHILD_KEY/$CHILD_NAME: attributes match"
      else
        fail "Instance $INST_NAME/$CHILD_KEY/$CHILD_NAME: attributes" "orig=$ORIG_CHILD_ATTRS, imported=$IMP_CHILD_ATTRS"
      fi

      # Links on contained instances
      ORIG_CHILD_LINKS=$(jq -r --arg n "$INST_NAME" --arg k "$CHILD_KEY" --arg cn "$CHILD_NAME" '[.instances[] | select(.name==$n) | .[$k][] | select(.name==$cn) | (.links // [])[].association] | sort | join(",")' /tmp/e2e-export.json)
      IMP_CHILD_LINKS=$(jq -r --arg n "$INST_NAME" --arg k "$CHILD_KEY" --arg cn "$CHILD_NAME" '[.instances[] | select(.name==$n) | .[$k][] | select(.name==$cn) | (.links // [])[].association] | sort | join(",")' /tmp/e2e-reimport-export.json)
      if [ "$ORIG_CHILD_LINKS" = "$IMP_CHILD_LINKS" ]; then
        if [ -n "$ORIG_CHILD_LINKS" ]; then
          pass "Instance $INST_NAME/$CHILD_KEY/$CHILD_NAME: links match ($ORIG_CHILD_LINKS)"
        fi
      else
        fail "Instance $INST_NAME/$CHILD_KEY/$CHILD_NAME: links" "orig=$ORIG_CHILD_LINKS, imported=$IMP_CHILD_LINKS"
      fi
    done
  done
done

# ============================================================
# COMPARE: Verify via operational API (not just export JSON)
# ============================================================
h "Compare: Verify via operational API"

# Check instance counts per entity type
for ET_NAME in $(jq -r '.entity_types[].name' /tmp/e2e-export.json); do
  IMP_ET_NAME="${PREFIX}${ET_NAME}"
  ORIG_COUNT=$(jget "$DATA/catalogs/$ORIG_CAT/$ET_NAME" | jq '.total // 0')
  IMP_COUNT=$(jget "$DATA/catalogs/$IMP_CAT/$IMP_ET_NAME" | jq '.total // 0')
  if [ "$ORIG_COUNT" = "$IMP_COUNT" ]; then
    pass "API: $ET_NAME instance count matches ($ORIG_COUNT)"
  else
    fail "API: $ET_NAME instance count" "orig=$ORIG_COUNT, imported=$IMP_COUNT"
  fi
done

# Check forward references on guardrail pii-filter
ORIG_PII_ID=$(jget "$DATA/catalogs/$ORIG_CAT/${P}-guardrail" | jq -r '.items[] | select(.name=="pii-filter") | .id')
IMP_PII_ID=$(jget "$DATA/catalogs/$IMP_CAT/${PREFIX}${P}-guardrail" | jq -r '.items[] | select(.name=="pii-filter") | .id')

ORIG_PII_REFS=$(jget "$DATA/catalogs/$ORIG_CAT/${P}-guardrail/$ORIG_PII_ID/references" | jq 'length')
IMP_PII_REFS=$(jget "$DATA/catalogs/$IMP_CAT/${PREFIX}${P}-guardrail/$IMP_PII_ID/references" | jq 'length')
if [ "$ORIG_PII_REFS" = "$IMP_PII_REFS" ] 2>/dev/null; then
  pass "API: pii-filter forward refs match ($ORIG_PII_REFS)"
else
  fail "API: pii-filter forward refs" "orig=$ORIG_PII_REFS, imported=$IMP_PII_REFS"
fi

# ============================================================
# Summary
# ============================================================
h "Summary"
echo "  Total: $TOTAL, Passed: $PASS, Failed: $FAIL"
if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
