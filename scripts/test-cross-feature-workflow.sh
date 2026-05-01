#!/usr/bin/env bash
# Cross-Feature Workflow Live Test
# Tests the complete end-to-end lifecycle spanning ALL features in a single workflow.
# This is the most important live test because it catches bugs at feature boundaries.
#
# Usage:
#   ./scripts/test-cross-feature-workflow.sh                       # defaults to localhost:30080
#   ./scripts/test-cross-feature-workflow.sh http://localhost:30080 # explicit
#
# Phases:
#   1.  Schema Design (entity types, type defs, associations)
#   2.  Catalog Version Setup (CV + pins)
#   3.  Catalog Population (instances, containment, links)
#   4.  Validation
#   5.  Publishing
#   6.  Export
#   7.  Import into New Catalog
#   8.  Validate Imported
#   9.  Copy
#   10. Replace
#   11. Schema Evolution
#   12. Cleanup

set -uo pipefail

API_BASE="${1:-http://localhost:30080}"
META_API="$API_BASE/api/meta/v1"
DATA_API="$API_BASE/api/data/v1"

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
get_body()   { echo "$1" | sed '$d'; }

# --- Prefix all test data to avoid collisions ---
P="wf-${TIMESTAMP}"
TD_NAME="${P}-status-enum"
ET_SERVER="${P}-server"
ET_APP="${P}-application"
ET_DB="${P}-database"

# Track IDs for cleanup
TD_ID=""
SERVER_ET_ID=""
APP_ET_ID=""
DB_ET_ID=""
CV_ID=""
CATALOG_NAME="workflow-cat-${TIMESTAMP}"
IMPORT_CATALOG="workflow-imported-${TIMESTAMP}"
COPY_CATALOG="workflow-copy-${TIMESTAMP}"

cleanup() {
  header "Phase 12: Cleanup"
  # Delete catalogs (order does not matter among catalogs, but must precede CVs)
  for name in "$CATALOG_NAME" "$IMPORT_CATALOG" "$COPY_CATALOG" "${CATALOG_NAME}-archive"; do
    curl -s -o /dev/null -X DELETE "$DATA_API/catalogs/$name" \
      -H "X-User-Role: SuperAdmin" 2>/dev/null || true
  done
  echo "  Deleted catalogs"

  # Delete imported CV (created by import)
  if [ -n "${IMPORT_CV_ID:-}" ] && [ "${IMPORT_CV_ID:-}" != "null" ]; then
    curl -s -o /dev/null -X DELETE "$META_API/catalog-versions/$IMPORT_CV_ID" \
      -H "X-User-Role: SuperAdmin" 2>/dev/null || true
  fi
  # Also clean up any CV created for the copy catalog (same CV as imported)
  # Delete main CV
  if [ -n "${CV_ID:-}" ] && [ "${CV_ID:-}" != "null" ]; then
    curl -s -o /dev/null -X DELETE "$META_API/catalog-versions/$CV_ID" \
      -H "X-User-Role: SuperAdmin" 2>/dev/null || true
  fi
  echo "  Deleted catalog versions"

  # Delete entity types (reverse order of dependency)
  for etid in "${DB_ET_ID:-}" "${APP_ET_ID:-}" "${SERVER_ET_ID:-}"; do
    if [ -n "$etid" ] && [ "$etid" != "null" ]; then
      curl -s -o /dev/null -X DELETE "$META_API/entity-types/$etid" \
        -H "X-User-Role: Admin" 2>/dev/null || true
    fi
  done
  echo "  Deleted entity types"

  # Delete type definition
  if [ -n "${TD_ID:-}" ] && [ "${TD_ID:-}" != "null" ]; then
    curl -s -o /dev/null -X DELETE "$META_API/type-definitions/$TD_ID" \
      -H "X-User-Role: Admin" 2>/dev/null || true
  fi
  echo "  Deleted type definition"

  # Clean up imported entity types (created by import with matching names)
  for name in "$ET_SERVER" "$ET_APP" "$ET_DB"; do
    etid=$(curl -s "$META_API/entity-types?name=$name" -H "X-User-Role: Admin" 2>/dev/null | jq -r '.items[0].id // empty')
    if [ -n "$etid" ]; then
      curl -s -o /dev/null -X DELETE "$META_API/entity-types/$etid" \
        -H "X-User-Role: Admin" 2>/dev/null || true
    fi
  done

  # Clean up imported CVs by label prefix
  for prefix in "${P}-imported" "${P}-v1"; do
    cvids=$(curl -s "$META_API/catalog-versions" -H "X-User-Role: Admin" 2>/dev/null | \
      jq -r ".items[] | select(.version_label | startswith(\"$prefix\")) | .id" 2>/dev/null | head -10)
    for id in $cvids; do
      curl -s -o /dev/null -X DELETE "$META_API/catalog-versions/$id" \
        -H "X-User-Role: SuperAdmin" 2>/dev/null || true
    done
  done

  echo "  Cleanup complete"
}
trap cleanup EXIT

# ===================================================================
# Pre-cleanup: remove any leftover data from previous runs
# ===================================================================
for name in "$CATALOG_NAME" "$IMPORT_CATALOG" "$COPY_CATALOG" "${CATALOG_NAME}-archive"; do
  curl -s -o /dev/null -X DELETE "$DATA_API/catalogs/$name" \
    -H "X-User-Role: SuperAdmin" 2>/dev/null || true
done

# ===================================================================
# Health check
# ===================================================================
header "Health Check"
HEALTH=$(curl -s "$API_BASE/healthz" | jq -r '.status' 2>/dev/null)
if [ "$HEALTH" = "ok" ]; then
  pass "API healthy"
else
  fail "API health check" "got: $HEALTH"
  echo "Cannot proceed without healthy API."
  exit 1
fi

# ===================================================================
# Phase 1: Schema Design
# ===================================================================
header "Phase 1: Schema Design"

# Look up system type definition version IDs
TD_RESP=$(api GET "$META_API/type-definitions" Admin)
TD_BODY=$(get_body "$TD_RESP")
STRING_TDV=$(echo "$TD_BODY" | jq -r '.items[] | select(.name=="string") | .latest_version_id')
INT_TDV=$(echo "$TD_BODY" | jq -r '.items[] | select(.name=="integer") | .latest_version_id')
echo "  String TDV: $STRING_TDV"
echo "  Integer TDV: $INT_TDV"

# 1. Create type definition "wf-status-enum" with values
RESP=$(api POST "$META_API/type-definitions" Admin \
  "{\"name\":\"$TD_NAME\",\"description\":\"Status enum\",\"base_type\":\"enum\",\"constraints\":{\"values\":[\"active\",\"inactive\",\"pending\"]}}")
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
TD_ID=$(echo "$BODY" | jq -r '.id')
TD_TDV_ID=$(echo "$BODY" | jq -r '.latest_version_id')

if [ "$STATUS" = "201" ] && [ "$TD_ID" != "null" ]; then
  pass "1. Created type definition $TD_NAME (id=$TD_ID)"
else
  fail "1. Create type definition" "status=$STATUS"
  exit 1
fi

# 2. Create entity type "wf-server" with attributes
RESP=$(api POST "$META_API/entity-types" Admin "{\"name\":\"$ET_SERVER\",\"description\":\"Workflow server\"}")
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
SERVER_ET_ID=$(echo "$BODY" | jq -r '.entity_type.id')

if [ "$STATUS" = "201" ] && [ "$SERVER_ET_ID" != "null" ]; then
  pass "2a. Created entity type $ET_SERVER"
else
  fail "2a. Create entity type $ET_SERVER" "status=$STATUS"
  exit 1
fi

# Add attributes to wf-server: hostname (string, required), status (enum), cpu-count (integer)
api POST "$META_API/entity-types/$SERVER_ET_ID/attributes" Admin \
  "{\"name\":\"hostname\",\"type_definition_version_id\":\"$STRING_TDV\",\"required\":true}" > /dev/null 2>&1
api POST "$META_API/entity-types/$SERVER_ET_ID/attributes" Admin \
  "{\"name\":\"status\",\"type_definition_version_id\":\"$TD_TDV_ID\",\"required\":false}" > /dev/null 2>&1
api POST "$META_API/entity-types/$SERVER_ET_ID/attributes" Admin \
  "{\"name\":\"cpu-count\",\"type_definition_version_id\":\"$INT_TDV\",\"required\":false}" > /dev/null 2>&1
echo "  Added 3 attributes to $ET_SERVER"

# 3. Create entity type "wf-application"
RESP=$(api POST "$META_API/entity-types" Admin "{\"name\":\"$ET_APP\",\"description\":\"Workflow application\"}")
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
APP_ET_ID=$(echo "$BODY" | jq -r '.entity_type.id')

if [ "$STATUS" = "201" ] && [ "$APP_ET_ID" != "null" ]; then
  pass "3a. Created entity type $ET_APP"
else
  fail "3a. Create entity type $ET_APP" "status=$STATUS"
  exit 1
fi

# Add attributes to wf-application: app-name (string, required), version (string)
api POST "$META_API/entity-types/$APP_ET_ID/attributes" Admin \
  "{\"name\":\"app-name\",\"type_definition_version_id\":\"$STRING_TDV\",\"required\":true}" > /dev/null 2>&1
api POST "$META_API/entity-types/$APP_ET_ID/attributes" Admin \
  "{\"name\":\"version\",\"type_definition_version_id\":\"$STRING_TDV\",\"required\":false}" > /dev/null 2>&1
echo "  Added 2 attributes to $ET_APP"

# 4. Create containment association: wf-server contains wf-application
RESP=$(api POST "$META_API/entity-types/$SERVER_ET_ID/associations" Admin \
  "{\"name\":\"applications\",\"type\":\"containment\",\"target_entity_type_id\":\"$APP_ET_ID\",\"source_cardinality\":\"1\",\"target_cardinality\":\"0..n\"}")
STATUS=$(get_status "$RESP")
if [ "$STATUS" = "201" ]; then
  pass "4. Created containment: $ET_SERVER contains $ET_APP"
else
  fail "4. Create containment association" "status=$STATUS body=$(get_body "$RESP")"
fi

# 5. Create entity type "wf-database"
RESP=$(api POST "$META_API/entity-types" Admin "{\"name\":\"$ET_DB\",\"description\":\"Workflow database\"}")
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
DB_ET_ID=$(echo "$BODY" | jq -r '.entity_type.id')

if [ "$STATUS" = "201" ] && [ "$DB_ET_ID" != "null" ]; then
  pass "5a. Created entity type $ET_DB"
else
  fail "5a. Create entity type $ET_DB" "status=$STATUS"
  exit 1
fi

# Add attributes: db-name (string, required), engine (string)
api POST "$META_API/entity-types/$DB_ET_ID/attributes" Admin \
  "{\"name\":\"db-name\",\"type_definition_version_id\":\"$STRING_TDV\",\"required\":true}" > /dev/null 2>&1
api POST "$META_API/entity-types/$DB_ET_ID/attributes" Admin \
  "{\"name\":\"engine\",\"type_definition_version_id\":\"$STRING_TDV\",\"required\":false}" > /dev/null 2>&1
echo "  Added 2 attributes to $ET_DB"

# 6. Create directional association: wf-application -> wf-database
RESP=$(api POST "$META_API/entity-types/$APP_ET_ID/associations" Admin \
  "{\"name\":\"uses-database\",\"type\":\"directional\",\"target_entity_type_id\":\"$DB_ET_ID\",\"source_cardinality\":\"0..n\",\"target_cardinality\":\"0..n\"}")
STATUS=$(get_status "$RESP")
if [ "$STATUS" = "201" ]; then
  pass "6. Created directional: $ET_APP -> $ET_DB (uses-database)"
else
  fail "6. Create directional association" "status=$STATUS body=$(get_body "$RESP")"
fi

# 7. Verify containment tree
RESP=$(api GET "$META_API/entity-types/containment-tree" Admin)
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
HAS_SERVER=$(echo "$BODY" | jq "[.[] | select(.entity_type.name==\"$ET_SERVER\")] | length")
HAS_APP_CHILD=$(echo "$BODY" | jq "[.[] | select(.entity_type.name==\"$ET_SERVER\") | .children[] | select(.entity_type.name==\"$ET_APP\")] | length")

if [ "$STATUS" = "200" ] && [ "$HAS_SERVER" -ge 1 ] && [ "$HAS_APP_CHILD" -ge 1 ]; then
  pass "7. Containment tree: $ET_SERVER -> $ET_APP hierarchy present"
else
  fail "7. Containment tree" "status=$STATUS server=$HAS_SERVER app_child=$HAS_APP_CHILD"
fi

# ===================================================================
# Phase 2: Catalog Version Setup
# ===================================================================
header "Phase 2: Catalog Version Setup"

# Get latest entity type version IDs (after attribute additions, ETVs may have changed)
SERVER_ETV_ID=$(get_body "$(api GET "$META_API/entity-types/$SERVER_ET_ID/versions" Admin)" | jq -r '.items[-1].id')
APP_ETV_ID=$(get_body "$(api GET "$META_API/entity-types/$APP_ET_ID/versions" Admin)" | jq -r '.items[-1].id')
DB_ETV_ID=$(get_body "$(api GET "$META_API/entity-types/$DB_ET_ID/versions" Admin)" | jq -r '.items[-1].id')
echo "  Server ETV: $SERVER_ETV_ID"
echo "  App ETV: $APP_ETV_ID"
echo "  DB ETV: $DB_ETV_ID"

# 8. Create catalog version with all 3 entity types pinned
RESP=$(api POST "$META_API/catalog-versions" Admin \
  "{\"version_label\":\"${P}-v1\",\"description\":\"Workflow CV v1\",\"pins\":[{\"entity_type_version_id\":\"$SERVER_ETV_ID\"},{\"entity_type_version_id\":\"$APP_ETV_ID\"},{\"entity_type_version_id\":\"$DB_ETV_ID\"}]}")
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
CV_ID=$(echo "$BODY" | jq -r '.id')

if [ "$STATUS" = "201" ] && [ "$CV_ID" != "null" ]; then
  pass "8. Created catalog version ${P}-v1 (id=$CV_ID)"
else
  fail "8. Create catalog version" "status=$STATUS body=$BODY"
  exit 1
fi

# 9-10. Verify pins (all 3 were created inline with the CV)
RESP=$(api GET "$META_API/catalog-versions/$CV_ID/pins" Admin)
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
PIN_COUNT=$(echo "$BODY" | jq '.items | length')

if [ "$STATUS" = "200" ] && [ "$PIN_COUNT" = "3" ]; then
  pass "9-10. CV has 3 pins (server, app, database)"
else
  fail "9-10. CV pin count" "expected=3 got=$PIN_COUNT status=$STATUS"
fi

# Extract pin IDs for later (schema evolution)
SERVER_PIN_ID=$(echo "$BODY" | jq -r ".items[] | select(.entity_type_version_id==\"$SERVER_ETV_ID\") | .pin_id")
echo "  Server pin ID: $SERVER_PIN_ID"

# ===================================================================
# Phase 3: Catalog Population
# ===================================================================
header "Phase 3: Catalog Population"

# 12. Create catalog
RESP=$(api POST "$DATA_API/catalogs" Admin \
  "{\"name\":\"$CATALOG_NAME\",\"description\":\"Cross-feature workflow test\",\"catalog_version_id\":\"$CV_ID\"}")
STATUS=$(get_status "$RESP")

if [ "$STATUS" = "201" ]; then
  pass "12. Created catalog $CATALOG_NAME"
else
  fail "12. Create catalog" "status=$STATUS body=$(get_body "$RESP")"
  exit 1
fi

# 13. Create server instance with attributes
RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/$ET_SERVER" Admin \
  "{\"name\":\"web-server-1\",\"description\":\"Primary web server\",\"attributes\":{\"hostname\":\"web1.example.com\",\"status\":\"active\",\"cpu-count\":\"8\"}}")
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
SERVER_INST_ID=$(echo "$BODY" | jq -r '.id')

if [ "$STATUS" = "201" ] && [ "$SERVER_INST_ID" != "null" ]; then
  pass "13. Created server instance web-server-1"
else
  fail "13. Create server instance" "status=$STATUS body=$BODY"
fi

# 14. Create contained application instance under web-server-1
RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/$ET_SERVER/$SERVER_INST_ID/$ET_APP" Admin \
  "{\"name\":\"frontend-app\",\"description\":\"Frontend application\",\"attributes\":{\"app-name\":\"Frontend\",\"version\":\"2.0\"}}")
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
APP_INST_ID=$(echo "$BODY" | jq -r '.id')

if [ "$STATUS" = "201" ] && [ "$APP_INST_ID" != "null" ]; then
  pass "14. Created contained app instance frontend-app under web-server-1"
else
  fail "14. Create contained instance" "status=$STATUS body=$BODY"
fi

# 15. Create database instance
RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/$ET_DB" Admin \
  "{\"name\":\"main-db\",\"description\":\"Production database\",\"attributes\":{\"db-name\":\"production-db\",\"engine\":\"postgres\"}}")
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
DB_INST_ID=$(echo "$BODY" | jq -r '.id')

if [ "$STATUS" = "201" ] && [ "$DB_INST_ID" != "null" ]; then
  pass "15. Created database instance main-db"
else
  fail "15. Create database instance" "status=$STATUS body=$BODY"
fi

# 16. Create association link: frontend-app -> main-db via "uses-database"
RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/$ET_APP/$APP_INST_ID/links" Admin \
  "{\"target_instance_id\":\"$DB_INST_ID\",\"association_name\":\"uses-database\"}")
STATUS=$(get_status "$RESP")

if [ "$STATUS" = "201" ]; then
  pass "16. Created link: frontend-app -> main-db (uses-database)"
else
  fail "16. Create association link" "status=$STATUS body=$(get_body "$RESP")"
fi

# 17. Verify tree shows correct hierarchy
RESP=$(api GET "$DATA_API/catalogs/$CATALOG_NAME/tree" RO)
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
TREE_SERVER=$(echo "$BODY" | jq "[.[] | select(.instance_name==\"web-server-1\")] | length")
TREE_APP_CHILD=$(echo "$BODY" | jq "[.[] | select(.instance_name==\"web-server-1\") | .children[] | select(.instance_name==\"frontend-app\")] | length")

if [ "$STATUS" = "200" ] && [ "$TREE_SERVER" -ge 1 ] && [ "$TREE_APP_CHILD" -ge 1 ]; then
  pass "17. Catalog tree: web-server-1 -> frontend-app hierarchy correct"
else
  fail "17. Catalog tree" "status=$STATUS server=$TREE_SERVER app_child=$TREE_APP_CHILD"
fi

# 18. Verify forward references from frontend-app
RESP=$(api GET "$DATA_API/catalogs/$CATALOG_NAME/$ET_APP/$APP_INST_ID/references" RO)
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
FWD_COUNT=$(echo "$BODY" | jq 'length')
FWD_TARGET=$(echo "$BODY" | jq -r '.[0].instance_name')

if [ "$STATUS" = "200" ] && [ "$FWD_COUNT" -ge 1 ] && [ "$FWD_TARGET" = "main-db" ]; then
  pass "18. Forward references: frontend-app -> main-db"
else
  fail "18. Forward references" "status=$STATUS count=$FWD_COUNT target=$FWD_TARGET"
fi

# 19. Verify reverse references from main-db
RESP=$(api GET "$DATA_API/catalogs/$CATALOG_NAME/$ET_DB/$DB_INST_ID/referenced-by" RO)
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
REV_COUNT=$(echo "$BODY" | jq 'length')
REV_SOURCE=$(echo "$BODY" | jq -r '.[0].instance_name')

if [ "$STATUS" = "200" ] && [ "$REV_COUNT" -ge 1 ] && [ "$REV_SOURCE" = "frontend-app" ]; then
  pass "19. Reverse references: main-db <- frontend-app"
else
  fail "19. Reverse references" "status=$STATUS count=$REV_COUNT source=$REV_SOURCE"
fi

# ===================================================================
# Phase 4: Validation
# ===================================================================
header "Phase 4: Validation"

# 20. Validate catalog
RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/validate" Admin)
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
VAL_STATUS=$(echo "$BODY" | jq -r '.status')

if [ "$STATUS" = "200" ] && [ "$VAL_STATUS" = "valid" ]; then
  pass "20. Catalog validation: valid (all required attrs set)"
else
  fail "20. Catalog validation" "status=$STATUS val_status=$VAL_STATUS errors=$(echo "$BODY" | jq '.errors')"
fi

# 21. Verify validation status persisted
RESP=$(api GET "$DATA_API/catalogs/$CATALOG_NAME" Admin)
BODY=$(get_body "$RESP")
PERSISTED=$(echo "$BODY" | jq -r '.validation_status')

if [ "$PERSISTED" = "valid" ]; then
  pass "21. Validation status persisted as valid"
else
  fail "21. Validation status persistence" "expected=valid got=$PERSISTED"
fi

# ===================================================================
# Phase 5: Publishing
# ===================================================================
header "Phase 5: Publishing"

# 22. Publish catalog
RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/publish" Admin)
STATUS=$(get_status "$RESP")

if [ "$STATUS" = "200" ]; then
  pass "22. Published catalog successfully"
else
  fail "22. Publish catalog" "status=$STATUS body=$(get_body "$RESP")"
fi

# 23. Verify published=true and published_at set
RESP=$(api GET "$DATA_API/catalogs/$CATALOG_NAME" Admin)
BODY=$(get_body "$RESP")
PUBLISHED=$(echo "$BODY" | jq -r '.published')
PUBLISHED_AT=$(echo "$BODY" | jq -r '.published_at')

if [ "$PUBLISHED" = "true" ] && [ "$PUBLISHED_AT" != "null" ] && [ "$PUBLISHED_AT" != "" ]; then
  pass "23. published=true, published_at=$PUBLISHED_AT"
else
  fail "23. Published state" "published=$PUBLISHED published_at=$PUBLISHED_AT"
fi

# 24. Verify RW cannot create instances on published catalog
RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/$ET_DB" RW \
  '{"name":"rw-blocked-db","description":"should be blocked"}')
STATUS=$(get_status "$RESP")

if [ "$STATUS" = "403" ]; then
  pass "24. RW blocked from creating instance on published catalog (403)"
else
  fail "24. RW write protection" "expected=403 got=$STATUS"
fi

# 25. Verify SuperAdmin CAN create instances on published catalog
RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/$ET_DB" SuperAdmin \
  "{\"name\":\"superadmin-db\",\"description\":\"SuperAdmin override\",\"attributes\":{\"db-name\":\"sa-db\"}}")
STATUS=$(get_status "$RESP")
SA_DB_INST_ID=$(get_body "$RESP" | jq -r '.id')

if [ "$STATUS" = "201" ]; then
  pass "25. SuperAdmin can create instance on published catalog (201)"
  # Clean up the extra instance to not affect export counts
  api DELETE "$DATA_API/catalogs/$CATALOG_NAME/$ET_DB/$SA_DB_INST_ID" SuperAdmin > /dev/null 2>&1
else
  fail "25. SuperAdmin write" "status=$STATUS body=$(get_body "$RESP")"
fi

# Unpublish for remaining phases (need to mutate data)
api POST "$DATA_API/catalogs/$CATALOG_NAME/unpublish" Admin > /dev/null 2>&1

# Re-validate after mutation (unpublish + instance delete set status to draft)
api POST "$DATA_API/catalogs/$CATALOG_NAME/validate" Admin > /dev/null 2>&1

# ===================================================================
# Phase 6: Export
# ===================================================================
header "Phase 6: Export"

# 26. Export catalog
RESP=$(api GET "$DATA_API/catalogs/$CATALOG_NAME/export" Admin)
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
echo "$BODY" > /tmp/wf-export-${TIMESTAMP}.json

if [ "$STATUS" = "200" ]; then
  pass "26. Export catalog returned 200"
else
  fail "26. Export catalog" "status=$STATUS"
fi

# 27. Verify export has all 3 entity types, instances, links
EXPORT_ET_COUNT=$(echo "$BODY" | jq '.entity_types | length')
EXPORT_INST_COUNT=$(echo "$BODY" | jq '[.instances[]] | length')
# Count link references across all instances (including nested children)
EXPORT_LINK_COUNT=$(echo "$BODY" | jq '[.instances[] | (.links // []), (to_entries[] | select(.key != "name" and .key != "entity_type" and .key != "attributes" and .key != "links") | .value[]? | (.links // [])) | .[] ] | length')
# Check containment (children nested under association name on parent)
EXPORT_CONTAINED=$(echo "$BODY" | jq "[.instances[] | select(.entity_type==\"$ET_SERVER\") | to_entries[] | select(.key != \"name\" and .key != \"entity_type\" and .key != \"attributes\" and .key != \"links\") | .value[]? ] | length")

if [ "$EXPORT_ET_COUNT" = "3" ]; then
  pass "27a. Export has 3 entity types"
else
  fail "27a. Export entity type count" "expected=3 got=$EXPORT_ET_COUNT"
fi

if [ "$EXPORT_INST_COUNT" -ge 2 ]; then
  pass "27b. Export has >= 2 root instances"
else
  fail "27b. Export instance count" "expected>=2 got=$EXPORT_INST_COUNT"
fi

if [ "$EXPORT_LINK_COUNT" -ge 1 ]; then
  pass "27c. Export has >= 1 association link"
else
  fail "27c. Export link count" "expected>=1 got=$EXPORT_LINK_COUNT"
fi

if [ "$EXPORT_CONTAINED" -ge 1 ]; then
  pass "27d. Export has contained instances"
else
  fail "27d. Export contained instances" "expected>=1 got=$EXPORT_CONTAINED"
fi

# 28. Verify export format_version
FMT_VER=$(echo "$BODY" | jq -r '.format_version')

if [ "$FMT_VER" = "1.0" ]; then
  pass "28. Export format_version is 1.0"
else
  fail "28. Export format_version" "expected=1.0 got=$FMT_VER"
fi

# ===================================================================
# Phase 7: Import into New Catalog
# ===================================================================
header "Phase 7: Import into New Catalog"

# 29. Import exported data with new catalog name
IMPORT_LABEL="${P}-imported-v1"
IMPORT_REQ=$(jq -n \
  --arg cat "$IMPORT_CATALOG" \
  --arg lbl "$IMPORT_LABEL" \
  --arg s "$ET_SERVER" --arg a "$ET_APP" --arg d "$ET_DB" \
  --slurpfile data "/tmp/wf-export-${TIMESTAMP}.json" '{
  catalog_name: $cat,
  catalog_version_label: $lbl,
  reuse_existing: [$s, $a, $d],
  data: $data[0]
}')

RESP=$(api POST "$DATA_API/catalogs/import" Admin "$IMPORT_REQ")
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")

if [ "$STATUS" = "201" ]; then
  pass "29. Import into $IMPORT_CATALOG returned 201"
else
  fail "29. Import catalog" "status=$STATUS body=$BODY"
fi

IMP_STATUS=$(echo "$BODY" | jq -r '.status')
IMP_INSTANCES=$(echo "$BODY" | jq -r '.instances_created // 0')
IMPORT_CV_ID=$(echo "$BODY" | jq -r '.catalog_version_id // empty')

# 30. Verify imported catalog exists with draft status
RESP=$(api GET "$DATA_API/catalogs/$IMPORT_CATALOG" Admin)
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
IMP_VAL_STATUS=$(echo "$BODY" | jq -r '.validation_status')

if [ "$STATUS" = "200" ] && [ "$IMP_VAL_STATUS" = "draft" ]; then
  pass "30. Imported catalog exists with draft status"
else
  fail "30. Imported catalog status" "status=$STATUS val_status=$IMP_VAL_STATUS"
fi

# 31. Verify imported catalog has correct instance count
# Should have at least 3: server, app (contained), database
if [ "$IMP_INSTANCES" -ge 3 ] 2>/dev/null; then
  pass "31. Imported catalog has >= 3 instances ($IMP_INSTANCES)"
else
  fail "31. Imported instance count" "expected>=3 got=$IMP_INSTANCES"
fi

# 32. Verify containment hierarchy preserved in imported catalog
IMP_SERVER_ID=$(get_body "$(api GET "$DATA_API/catalogs/$IMPORT_CATALOG/$ET_SERVER" Admin)" | jq -r '.items[0].id // empty')

if [ -n "$IMP_SERVER_ID" ] && [ "$IMP_SERVER_ID" != "null" ]; then
  RESP=$(api GET "$DATA_API/catalogs/$IMPORT_CATALOG/$ET_SERVER/$IMP_SERVER_ID/$ET_APP" Admin)
  IMP_APP_COUNT=$(get_body "$RESP" | jq '.items | length')

  if [ "$IMP_APP_COUNT" -ge 1 ] 2>/dev/null; then
    pass "32. Containment preserved: imported server has contained app ($IMP_APP_COUNT)"
  else
    fail "32. Containment preservation" "app count=$IMP_APP_COUNT"
  fi
else
  fail "32. Containment preservation" "no server instance found in imported catalog"
fi

# 33. Verify association links preserved
IMP_APP_ID=$(get_body "$(api GET "$DATA_API/catalogs/$IMPORT_CATALOG/$ET_SERVER/$IMP_SERVER_ID/$ET_APP" Admin)" | jq -r '.items[0].id // empty')

if [ -n "$IMP_APP_ID" ] && [ "$IMP_APP_ID" != "null" ]; then
  RESP=$(api GET "$DATA_API/catalogs/$IMPORT_CATALOG/$ET_APP/$IMP_APP_ID/references" RO)
  IMP_LINK_COUNT=$(get_body "$RESP" | jq 'length')

  if [ "$IMP_LINK_COUNT" -ge 1 ] 2>/dev/null; then
    pass "33. Association links preserved in imported catalog ($IMP_LINK_COUNT)"
  else
    fail "33. Association link preservation" "link count=$IMP_LINK_COUNT"
  fi
else
  fail "33. Association link preservation" "no app instance found in imported catalog"
fi

# ===================================================================
# Phase 8: Validate Imported
# ===================================================================
header "Phase 8: Validate Imported Catalog"

# 34. Validate imported catalog
RESP=$(api POST "$DATA_API/catalogs/$IMPORT_CATALOG/validate" Admin)
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
IMP_VAL=$(echo "$BODY" | jq -r '.status')

if [ "$STATUS" = "200" ] && [ "$IMP_VAL" = "valid" ]; then
  pass "34. Imported catalog validation: valid"
else
  fail "34. Imported catalog validation" "status=$STATUS val=$IMP_VAL errors=$(echo "$BODY" | jq '.errors // empty')"
fi

# ===================================================================
# Phase 9: Copy
# ===================================================================
header "Phase 9: Copy"

# 35. Copy "workflow-imported" to "workflow-copy"
RESP=$(api POST "$DATA_API/catalogs/copy" Admin \
  "{\"source\":\"$IMPORT_CATALOG\",\"name\":\"$COPY_CATALOG\",\"description\":\"Copied from imported\"}")
STATUS=$(get_status "$RESP")

if [ "$STATUS" = "201" ]; then
  pass "35. Copied $IMPORT_CATALOG to $COPY_CATALOG (201)"
else
  fail "35. Copy catalog" "status=$STATUS body=$(get_body "$RESP")"
fi

# 36. Verify copy has same instance count
COPY_SERVER_COUNT=$(get_body "$(api GET "$DATA_API/catalogs/$COPY_CATALOG/$ET_SERVER" Admin)" | jq '.total')
IMP_SERVER_COUNT=$(get_body "$(api GET "$DATA_API/catalogs/$IMPORT_CATALOG/$ET_SERVER" Admin)" | jq '.total')

if [ "$COPY_SERVER_COUNT" = "$IMP_SERVER_COUNT" ]; then
  pass "36. Copy has same server instance count ($COPY_SERVER_COUNT)"
else
  fail "36. Copy instance count" "copy=$COPY_SERVER_COUNT imported=$IMP_SERVER_COUNT"
fi

# 37. Verify copy is independent (delete instance from copy, verify original unchanged)
COPY_DB_RESP=$(api GET "$DATA_API/catalogs/$COPY_CATALOG/$ET_DB" Admin)
COPY_DB_ID=$(get_body "$COPY_DB_RESP" | jq -r '.items[0].id // empty')

if [ -n "$COPY_DB_ID" ] && [ "$COPY_DB_ID" != "null" ]; then
  api DELETE "$DATA_API/catalogs/$COPY_CATALOG/$ET_DB/$COPY_DB_ID" Admin > /dev/null 2>&1

  # Verify original still has the database instance
  ORIG_DB_COUNT=$(get_body "$(api GET "$DATA_API/catalogs/$IMPORT_CATALOG/$ET_DB" Admin)" | jq '.total')
  COPY_DB_COUNT=$(get_body "$(api GET "$DATA_API/catalogs/$COPY_CATALOG/$ET_DB" Admin)" | jq '.total')

  if [ "$ORIG_DB_COUNT" -ge 1 ] && [ "$COPY_DB_COUNT" -eq 0 ]; then
    pass "37. Copy is independent (deleted from copy, original unchanged)"
  else
    fail "37. Copy independence" "orig_db=$ORIG_DB_COUNT copy_db=$COPY_DB_COUNT"
  fi
else
  fail "37. Copy independence" "no database instance found in copy"
fi

# ===================================================================
# Phase 10: Replace
# ===================================================================
header "Phase 10: Replace"

# 38. Validate workflow-copy (it may be draft after the delete above)
RESP=$(api POST "$DATA_API/catalogs/$COPY_CATALOG/validate" Admin)
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
COPY_VAL=$(echo "$BODY" | jq -r '.status')

if [ "$STATUS" = "200" ] && [ "$COPY_VAL" = "valid" ]; then
  pass "38. Copy catalog validation: valid"
else
  fail "38. Copy catalog validation" "status=$STATUS val=$COPY_VAL"
fi

# 39. Replace original catalog with copy
# The original catalog ($CATALOG_NAME) may still be published, unpublish it first
api POST "$DATA_API/catalogs/$CATALOG_NAME/unpublish" Admin > /dev/null 2>&1

ARCHIVE_NAME="${CATALOG_NAME}-archive"
RESP=$(api POST "$DATA_API/catalogs/replace" Admin \
  "{\"source\":\"$COPY_CATALOG\",\"target\":\"$CATALOG_NAME\",\"archive_name\":\"$ARCHIVE_NAME\"}")
STATUS=$(get_status "$RESP")

if [ "$STATUS" = "200" ]; then
  pass "39. Replaced $CATALOG_NAME with $COPY_CATALOG (200)"
else
  fail "39. Replace catalog" "status=$STATUS body=$(get_body "$RESP")"
fi

# 40. Verify target catalog now has the copied data (db instance was deleted from copy)
REPLACED_DB_COUNT=$(get_body "$(api GET "$DATA_API/catalogs/$CATALOG_NAME/$ET_DB" Admin)" | jq '.total')

if [ "$REPLACED_DB_COUNT" -eq 0 ]; then
  pass "40. Replaced catalog has copy's data (0 database instances)"
else
  fail "40. Replaced catalog data" "expected 0 db instances, got $REPLACED_DB_COUNT"
fi

# 41. Verify archive catalog exists
RESP=$(api GET "$DATA_API/catalogs/$ARCHIVE_NAME" Admin)
STATUS=$(get_status "$RESP")

if [ "$STATUS" = "200" ]; then
  pass "41. Archive catalog $ARCHIVE_NAME exists"
else
  fail "41. Archive catalog" "status=$STATUS"
fi

# ===================================================================
# Phase 11: Schema Evolution
# ===================================================================
header "Phase 11: Schema Evolution"

# First, re-populate the catalog with a server instance for schema evolution testing
# (The replace may have moved instances around, ensure we have a server)
EXISTING_SERVERS=$(get_body "$(api GET "$DATA_API/catalogs/$CATALOG_NAME/$ET_SERVER" Admin)" | jq '.total')
if [ "$EXISTING_SERVERS" -eq 0 ]; then
  api POST "$DATA_API/catalogs/$CATALOG_NAME/$ET_SERVER" Admin \
    "{\"name\":\"evo-server\",\"description\":\"Schema evolution test\",\"attributes\":{\"hostname\":\"evo.example.com\",\"status\":\"active\",\"cpu-count\":\"4\"}}" > /dev/null 2>&1
fi

# Get current server instance IDs
EVO_RESP=$(api GET "$DATA_API/catalogs/$CATALOG_NAME/$ET_SERVER" Admin)
EVO_BODY=$(get_body "$EVO_RESP")
EVO_INST_COUNT=$(echo "$EVO_BODY" | jq '.total')
EVO_INST_ID=$(echo "$EVO_BODY" | jq -r '.items[0].id')
echo "  Server instances in catalog: $EVO_INST_COUNT"
echo "  Test instance: $EVO_INST_ID"

# 42. Create V2 of wf-server: add "memory-gb" (integer), remove "cpu-count"
RESP=$(api DELETE "$META_API/entity-types/$SERVER_ET_ID/attributes/cpu-count" Admin)
STATUS=$(get_status "$RESP")
if [ "$STATUS" = "204" ] || [ "$STATUS" = "200" ]; then
  echo "  Removed cpu-count attribute"
else
  echo "  Warning: remove cpu-count returned $STATUS"
fi

RESP=$(api POST "$META_API/entity-types/$SERVER_ET_ID/attributes" Admin \
  "{\"name\":\"memory-gb\",\"type_definition_version_id\":\"$INT_TDV\",\"required\":false}")
STATUS=$(get_status "$RESP")
if [ "$STATUS" = "201" ]; then
  echo "  Added memory-gb attribute"
else
  echo "  Warning: add memory-gb returned $STATUS"
fi

# Get the V2 ETV ID
V2_ETV_ID=$(get_body "$(api GET "$META_API/entity-types/$SERVER_ET_ID/versions" Admin)" | jq -r '.items[-1].id')
echo "  V2 ETV ID: $V2_ETV_ID"

if [ "$V2_ETV_ID" != "$SERVER_ETV_ID" ]; then
  pass "42. Created V2 of $ET_SERVER (V1=$SERVER_ETV_ID, V2=$V2_ETV_ID)"
else
  fail "42. Schema evolution" "V1 and V2 have the same ETV ID"
fi

# Re-read the pin ID (it might have changed format/been recreated)
PIN_RESP=$(api GET "$META_API/catalog-versions/$CV_ID/pins" Admin)
PIN_BODY=$(get_body "$PIN_RESP")
SERVER_PIN_ID=$(echo "$PIN_BODY" | jq -r ".items[] | select(.entity_type_version_id==\"$SERVER_ETV_ID\") | .pin_id")
echo "  Server pin ID for V1: $SERVER_PIN_ID"

# 43. Dry-run UpdatePin V1->V2
RESP=$(api PUT "$META_API/catalog-versions/$CV_ID/pins/$SERVER_PIN_ID?dry_run=true" Admin \
  "{\"entity_type_version_id\":\"$V2_ETV_ID\"}")
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")

if [ "$STATUS" = "200" ]; then
  pass "43. Dry-run UpdatePin V1->V2 returned 200"
else
  fail "43. Dry-run UpdatePin" "status=$STATUS body=$BODY"
fi

# 44. Verify dry-run shows affected instances and attribute mapping
DRY_AFFECTED=$(echo "$BODY" | jq '.migration.affected_instances // 0')
DRY_HAS_MIGRATION=$(echo "$BODY" | jq 'has("migration")')
DRY_PIN_ETV=$(echo "$BODY" | jq -r '.pin.entity_type_version_id')

if [ "$DRY_HAS_MIGRATION" = "true" ] && [ "$DRY_AFFECTED" -ge 1 ]; then
  pass "44a. Dry-run: migration report present, affected_instances=$DRY_AFFECTED"
else
  fail "44a. Dry-run migration report" "has_migration=$DRY_HAS_MIGRATION affected=$DRY_AFFECTED"
fi

# Verify pin NOT changed (still V1)
if [ "$DRY_PIN_ETV" = "$SERVER_ETV_ID" ]; then
  pass "44b. Dry-run: pin still points to V1 (read-only)"
else
  fail "44b. Dry-run pin unchanged" "expected=$SERVER_ETV_ID got=$DRY_PIN_ETV"
fi

# 45. Apply UpdatePin V1->V2
RESP=$(api PUT "$META_API/catalog-versions/$CV_ID/pins/$SERVER_PIN_ID" Admin \
  "{\"entity_type_version_id\":\"$V2_ETV_ID\"}")
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
REAL_PIN_ETV=$(echo "$BODY" | jq -r '.pin.entity_type_version_id')
REAL_AFFECTED=$(echo "$BODY" | jq '.migration.affected_instances // 0')

if [ "$STATUS" = "200" ] && [ "$REAL_PIN_ETV" = "$V2_ETV_ID" ]; then
  pass "45a. UpdatePin applied: pin now points to V2"
else
  fail "45a. UpdatePin" "status=$STATUS pin=$REAL_PIN_ETV"
fi

if [ "$REAL_AFFECTED" -ge 1 ]; then
  pass "45b. Migration affected $REAL_AFFECTED instance(s)"
else
  fail "45b. Migration affected count" "expected>=1 got=$REAL_AFFECTED"
fi

# 46. Verify instances migrated
RESP=$(api GET "$DATA_API/catalogs/$CATALOG_NAME/$ET_SERVER/$EVO_INST_ID" RO)
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")

# attr-a (hostname) should still be present
HOSTNAME_VAL=$(echo "$BODY" | jq -r '.attributes[] | select(.name=="hostname") | .value')
# Verify V2 schema does not include cpu-count
RESP2=$(api GET "$META_API/entity-types/$SERVER_ET_ID/attributes" Admin)
SCHEMA_CPU=$(echo "$(get_body "$RESP2")" | jq '[.items[] | select(.name=="cpu-count")] | length')

if [ "$HOSTNAME_VAL" != "" ] && [ "$HOSTNAME_VAL" != "null" ]; then
  pass "46a. hostname attribute preserved after migration"
else
  fail "46a. hostname preservation" "got=$HOSTNAME_VAL"
fi

if [ "$SCHEMA_CPU" = "0" ]; then
  pass "46b. cpu-count no longer in V2 schema"
else
  fail "46b. cpu-count should be absent from V2 schema" "found $SCHEMA_CPU occurrences"
fi

# ===================================================================
# Summary
# ===================================================================
header "Results"

echo ""
echo "  Total:  $TOTAL"
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
