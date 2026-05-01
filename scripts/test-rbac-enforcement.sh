#!/usr/bin/env bash
# Live system tests for RBAC enforcement across all write endpoints.
# Tests every write endpoint against all 4 roles (RO, RW, Admin, SuperAdmin)
# to verify correct 403/200/201/204 behavior.
#
# Usage: ./scripts/test-rbac-enforcement.sh [API_BASE_URL]
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

# assert_status RESP EXPECTED_STATUS TEST_NAME
assert_status() {
  local resp="$1" expected="$2" name="$3"
  local actual
  actual=$(get_status "$resp")
  if [ "$actual" = "$expected" ]; then
    pass "$name"
  else
    fail "$name" "expected=$expected got=$actual body=$(get_body "$resp" | head -c 200)"
  fi
}

TIMESTAMP=$(date +%s)
PREFIX="rbac-enf"

# Track IDs for cleanup
ET_ID=""
ET2_ID=""
ETV_ID=""
CV_ID=""
CV2_ID=""
CATALOG_NAME="${PREFIX}-cat-${TIMESTAMP}"
TD_ID=""
INSTANCE_ID=""

cleanup() {
  header "Cleanup"

  # Unpublish catalog if published (ignore errors)
  api POST "$DATA_API/catalogs/$CATALOG_NAME/unpublish" SuperAdmin > /dev/null 2>&1 || true

  # Delete catalog
  api DELETE "$DATA_API/catalogs/$CATALOG_NAME" SuperAdmin > /dev/null 2>&1 || true
  echo "  Deleted catalog: $CATALOG_NAME"

  # Delete CVs
  if [ -n "$CV_ID" ]; then
    api DELETE "$META_API/catalog-versions/$CV_ID" Admin > /dev/null 2>&1 || true
    echo "  Deleted CV: $CV_ID"
  fi
  if [ -n "$CV2_ID" ]; then
    api DELETE "$META_API/catalog-versions/$CV2_ID" Admin > /dev/null 2>&1 || true
    echo "  Deleted CV2: $CV2_ID"
  fi

  # Delete type definition
  if [ -n "$TD_ID" ]; then
    api DELETE "$META_API/type-definitions/$TD_ID" Admin > /dev/null 2>&1 || true
    echo "  Deleted TD: $TD_ID"
  fi

  # Delete entity types (must happen after CV deletion since CV pins reference ETVs)
  if [ -n "$ET_ID" ]; then
    api DELETE "$META_API/entity-types/$ET_ID" Admin > /dev/null 2>&1 || true
    echo "  Deleted ET: $ET_ID"
  fi
  if [ -n "$ET2_ID" ]; then
    api DELETE "$META_API/entity-types/$ET2_ID" Admin > /dev/null 2>&1 || true
    echo "  Deleted ET2: $ET2_ID"
  fi

  # Clean up any imported catalogs
  api DELETE "$DATA_API/catalogs/${PREFIX}-imported-${TIMESTAMP}" SuperAdmin > /dev/null 2>&1 || true

  # Clean up any imported CVs
  for lbl_prefix in "${PREFIX}-imp-cv-${TIMESTAMP}"; do
    cvid=$(curl -s "$META_API/catalog-versions" -H 'X-User-Role: Admin' | jq -r ".items[] | select(.version_label | startswith(\"$lbl_prefix\")) | .id" 2>/dev/null | head -5)
    for id in $cvid; do
      api DELETE "$META_API/catalog-versions/$id" Admin > /dev/null 2>&1 || true
    done
  done

  echo "  Cleanup complete"
}
trap cleanup EXIT

# ============================================================
# SETUP: Create base test data as Admin
# ============================================================
header "Setup: Creating test data as Admin"

# Create entity type
RESP=$(api POST "$META_API/entity-types" Admin "{\"name\":\"${PREFIX}-server-${TIMESTAMP}\"}")
ET_STATUS=$(get_status "$RESP")
ET_BODY=$(get_body "$RESP")
ET_ID=$(echo "$ET_BODY" | jq -r '.entity_type.id')
ETV_ID=$(echo "$ET_BODY" | jq -r '.version.id')
echo "  Entity type: $ET_ID (ETV=$ETV_ID, status=$ET_STATUS)"

if [ "$ET_ID" = "null" ] || [ -z "$ET_ID" ]; then
  echo "  FATAL: Setup failed — could not create entity type"
  exit 1
fi

# Get the latest ETV (may differ from initial if attributes were added)
ETV_ID=$(get_body "$(api GET "$META_API/entity-types/$ET_ID/versions" Admin)" | jq -r '.items[-1].id')

# Get string type definition version ID for attribute creation
STRING_TDV_ID=$(get_body "$(api GET "$META_API/type-definitions" Admin)" | jq -r '.items[] | select(.name=="string") | .latest_version_id')
echo "  String TDV: $STRING_TDV_ID"

# Create a second entity type for association tests
RESP=$(api POST "$META_API/entity-types" Admin "{\"name\":\"${PREFIX}-target-${TIMESTAMP}\"}")
ET2_ID=$(echo "$(get_body "$RESP")" | jq -r '.entity_type.id')
ET2_ETV_ID=$(echo "$(get_body "$RESP")" | jq -r '.version.id')
echo "  Entity type 2: $ET2_ID"

# Create CV with pin
RESP=$(api POST "$META_API/catalog-versions" Admin \
  "{\"version_label\":\"${PREFIX}-cv-${TIMESTAMP}\",\"pins\":[{\"entity_type_version_id\":\"$ETV_ID\"}]}")
CV_ID=$(get_body "$RESP" | jq -r '.id')
echo "  CV: $CV_ID"

if [ "$CV_ID" = "null" ] || [ -z "$CV_ID" ]; then
  echo "  FATAL: Setup failed — could not create CV"
  exit 1
fi

# Create catalog
RESP=$(api POST "$DATA_API/catalogs" Admin \
  "{\"name\":\"$CATALOG_NAME\",\"description\":\"RBAC test\",\"catalog_version_id\":\"$CV_ID\"}")
CAT_STATUS=$(get_status "$RESP")
echo "  Catalog: $CATALOG_NAME (status=$CAT_STATUS)"

echo "  Setup complete"

# ============================================================
# SECTION 1: Meta API — Entity Type CRUD (Admin+ required)
# ============================================================
header "Section 1: Entity Type CRUD (Admin+ required)"

# --- POST /entity-types ---
header "1.1: POST /entity-types"

RESP=$(api POST "$META_API/entity-types" RO "{\"name\":\"${PREFIX}-should-fail-ro\"}")
assert_status "$RESP" "403" "RO cannot create entity type (403)"

RESP=$(api POST "$META_API/entity-types" RW "{\"name\":\"${PREFIX}-should-fail-rw\"}")
assert_status "$RESP" "403" "RW cannot create entity type (403)"

# Admin can create — already proven in setup, but test explicitly
RESP=$(api POST "$META_API/entity-types" Admin "{\"name\":\"${PREFIX}-admin-test-${TIMESTAMP}\"}")
STATUS=$(get_status "$RESP")
ADMIN_ET_ID=$(get_body "$RESP" | jq -r '.entity_type.id')
if [ "$STATUS" = "201" ]; then
  pass "Admin can create entity type (201)"
  # Clean up immediately
  api DELETE "$META_API/entity-types/$ADMIN_ET_ID" Admin > /dev/null 2>&1 || true
else
  fail "Admin can create entity type (201)" "expected=201 got=$STATUS"
fi

RESP=$(api POST "$META_API/entity-types" SuperAdmin "{\"name\":\"${PREFIX}-sa-test-${TIMESTAMP}\"}")
STATUS=$(get_status "$RESP")
SA_ET_ID=$(get_body "$RESP" | jq -r '.entity_type.id')
if [ "$STATUS" = "201" ]; then
  pass "SuperAdmin can create entity type (201)"
  api DELETE "$META_API/entity-types/$SA_ET_ID" Admin > /dev/null 2>&1 || true
else
  fail "SuperAdmin can create entity type (201)" "expected=201 got=$STATUS"
fi

# --- PUT /entity-types/:id ---
header "1.2: PUT /entity-types/:id"

RESP=$(api PUT "$META_API/entity-types/$ET_ID" RO "{\"description\":\"updated\"}")
assert_status "$RESP" "403" "RO cannot update entity type (403)"

RESP=$(api PUT "$META_API/entity-types/$ET_ID" RW "{\"description\":\"updated\"}")
assert_status "$RESP" "403" "RW cannot update entity type (403)"

RESP=$(api PUT "$META_API/entity-types/$ET_ID" Admin "{\"description\":\"updated by admin\"}")
assert_status "$RESP" "200" "Admin can update entity type (200)"

# --- DELETE /entity-types/:id ---
header "1.3: DELETE /entity-types/:id"

# Create a throwaway ET to test delete permissions
RESP=$(api POST "$META_API/entity-types" Admin "{\"name\":\"${PREFIX}-del-test-${TIMESTAMP}\"}")
DEL_ET_ID=$(get_body "$RESP" | jq -r '.entity_type.id')

RESP=$(api DELETE "$META_API/entity-types/$DEL_ET_ID" RO)
assert_status "$RESP" "403" "RO cannot delete entity type (403)"

RESP=$(api DELETE "$META_API/entity-types/$DEL_ET_ID" RW)
assert_status "$RESP" "403" "RW cannot delete entity type (403)"

RESP=$(api DELETE "$META_API/entity-types/$DEL_ET_ID" Admin)
assert_status "$RESP" "204" "Admin can delete entity type (204)"

# ============================================================
# SECTION 2: Meta API — Attributes (Admin+ required)
# ============================================================
header "Section 2: Attributes (Admin+ required)"

header "2.1: POST /entity-types/:id/attributes"

RESP=$(api POST "$META_API/entity-types/$ET_ID/attributes" RO \
  "{\"name\":\"${PREFIX}-attr-ro\",\"type_definition_version_id\":\"$STRING_TDV_ID\"}")
assert_status "$RESP" "403" "RO cannot add attribute (403)"

RESP=$(api POST "$META_API/entity-types/$ET_ID/attributes" RW \
  "{\"name\":\"${PREFIX}-attr-rw\",\"type_definition_version_id\":\"$STRING_TDV_ID\"}")
assert_status "$RESP" "403" "RW cannot add attribute (403)"

RESP=$(api POST "$META_API/entity-types/$ET_ID/attributes" Admin \
  "{\"name\":\"${PREFIX}-attr-admin\",\"type_definition_version_id\":\"$STRING_TDV_ID\"}")
assert_status "$RESP" "201" "Admin can add attribute (201)"

# Re-fetch the latest ETV after attribute addition
ETV_ID=$(get_body "$(api GET "$META_API/entity-types/$ET_ID/versions" Admin)" | jq -r '.items[-1].id')

# ============================================================
# SECTION 3: Meta API — Associations (Admin+ required)
# ============================================================
header "Section 3: Associations (Admin+ required)"

header "3.1: POST /entity-types/:id/associations"

RESP=$(api POST "$META_API/entity-types/$ET_ID/associations" RO \
  "{\"name\":\"${PREFIX}-assoc-ro\",\"type\":\"directional\",\"target_entity_type_id\":\"$ET2_ID\",\"source_cardinality\":\"0..n\",\"target_cardinality\":\"0..n\"}")
assert_status "$RESP" "403" "RO cannot create association (403)"

RESP=$(api POST "$META_API/entity-types/$ET_ID/associations" RW \
  "{\"name\":\"${PREFIX}-assoc-rw\",\"type\":\"directional\",\"target_entity_type_id\":\"$ET2_ID\",\"source_cardinality\":\"0..n\",\"target_cardinality\":\"0..n\"}")
assert_status "$RESP" "403" "RW cannot create association (403)"

RESP=$(api POST "$META_API/entity-types/$ET_ID/associations" Admin \
  "{\"name\":\"${PREFIX}-assoc-admin\",\"type\":\"directional\",\"target_entity_type_id\":\"$ET2_ID\",\"source_cardinality\":\"0..n\",\"target_cardinality\":\"0..n\"}")
assert_status "$RESP" "201" "Admin can create association (201)"

# ============================================================
# SECTION 4: Meta API — Type Definitions (Admin+ required)
# ============================================================
header "Section 4: Type Definitions (Admin+ required)"

header "4.1: POST /type-definitions"

RESP=$(api POST "$META_API/type-definitions" RO \
  "{\"name\":\"${PREFIX}-td-ro-${TIMESTAMP}\",\"base_type\":\"string\"}")
assert_status "$RESP" "403" "RO cannot create type definition (403)"

RESP=$(api POST "$META_API/type-definitions" RW \
  "{\"name\":\"${PREFIX}-td-rw-${TIMESTAMP}\",\"base_type\":\"string\"}")
assert_status "$RESP" "403" "RW cannot create type definition (403)"

RESP=$(api POST "$META_API/type-definitions" Admin \
  "{\"name\":\"${PREFIX}-td-${TIMESTAMP}\",\"base_type\":\"string\"}")
STATUS=$(get_status "$RESP")
TD_ID=$(get_body "$RESP" | jq -r '.id')
if [ "$STATUS" = "201" ]; then
  pass "Admin can create type definition (201)"
else
  fail "Admin can create type definition (201)" "expected=201 got=$STATUS"
fi

header "4.2: DELETE /type-definitions/:id"

RESP=$(api DELETE "$META_API/type-definitions/$TD_ID" RO)
assert_status "$RESP" "403" "RO cannot delete type definition (403)"

RESP=$(api DELETE "$META_API/type-definitions/$TD_ID" RW)
assert_status "$RESP" "403" "RW cannot delete type definition (403)"

RESP=$(api DELETE "$META_API/type-definitions/$TD_ID" Admin)
assert_status "$RESP" "204" "Admin can delete type definition (204)"
TD_ID="" # Already deleted

# ============================================================
# SECTION 5: Meta API — Catalog Versions (RW+ required)
# ============================================================
header "Section 5: Catalog Versions (RW+ required)"

header "5.1: POST /catalog-versions"

RESP=$(api POST "$META_API/catalog-versions" RO \
  "{\"version_label\":\"${PREFIX}-cv-ro-${TIMESTAMP}\",\"pins\":[{\"entity_type_version_id\":\"$ETV_ID\"}]}")
assert_status "$RESP" "403" "RO cannot create CV (403)"

RESP=$(api POST "$META_API/catalog-versions" RW \
  "{\"version_label\":\"${PREFIX}-cv-rw-${TIMESTAMP}\",\"pins\":[{\"entity_type_version_id\":\"$ETV_ID\"}]}")
STATUS=$(get_status "$RESP")
CV2_ID=$(get_body "$RESP" | jq -r '.id')
if [ "$STATUS" = "201" ]; then
  pass "RW can create CV (201)"
else
  fail "RW can create CV (201)" "expected=201 got=$STATUS"
fi

header "5.2: POST /catalog-versions/:id/pins"

RESP=$(api POST "$META_API/catalog-versions/$CV2_ID/pins" RO \
  "{\"entity_type_version_id\":\"$ET2_ETV_ID\"}")
assert_status "$RESP" "403" "RO cannot add pin (403)"

RESP=$(api POST "$META_API/catalog-versions/$CV2_ID/pins" RW \
  "{\"entity_type_version_id\":\"$ET2_ETV_ID\"}")
assert_status "$RESP" "201" "RW can add pin (201)"

# ============================================================
# SECTION 6: Operational API — Catalog CRUD (RW+ required)
# ============================================================
header "Section 6: Catalog CRUD (RW+ required)"

header "6.1: POST /catalogs (create)"

RESP=$(api POST "$DATA_API/catalogs" RO \
  "{\"name\":\"${PREFIX}-ro-cat-${TIMESTAMP}\",\"description\":\"test\",\"catalog_version_id\":\"$CV_ID\"}")
assert_status "$RESP" "403" "RO cannot create catalog (403)"

# RW can create — test with a unique name, then clean up
RESP=$(api POST "$DATA_API/catalogs" RW \
  "{\"name\":\"${PREFIX}-rw-cat-${TIMESTAMP}\",\"description\":\"test\",\"catalog_version_id\":\"$CV_ID\"}")
STATUS=$(get_status "$RESP")
if [ "$STATUS" = "201" ]; then
  pass "RW can create catalog (201)"
  api DELETE "$DATA_API/catalogs/${PREFIX}-rw-cat-${TIMESTAMP}" Admin > /dev/null 2>&1 || true
else
  fail "RW can create catalog (201)" "expected=201 got=$STATUS"
fi

header "6.2: DELETE /catalogs/:name"

# Create a throwaway catalog to test delete
api POST "$DATA_API/catalogs" Admin \
  "{\"name\":\"${PREFIX}-del-cat-${TIMESTAMP}\",\"description\":\"delete test\",\"catalog_version_id\":\"$CV_ID\"}" > /dev/null

RESP=$(api DELETE "$DATA_API/catalogs/${PREFIX}-del-cat-${TIMESTAMP}" RO)
assert_status "$RESP" "403" "RO cannot delete catalog (403)"

RESP=$(api DELETE "$DATA_API/catalogs/${PREFIX}-del-cat-${TIMESTAMP}" RW)
assert_status "$RESP" "204" "RW can delete catalog (204)"

header "6.3: POST /catalogs/:name/validate"

RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/validate" RO)
assert_status "$RESP" "403" "RO cannot validate catalog (403)"

RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/validate" RW)
assert_status "$RESP" "200" "RW can validate catalog (200)"

# ============================================================
# SECTION 7: Operational API — Publish/Unpublish (Admin+ required)
# ============================================================
header "Section 7: Publish/Unpublish (Admin+ required)"

# Ensure catalog is validated first
api POST "$DATA_API/catalogs/$CATALOG_NAME/validate" Admin > /dev/null

header "7.1: POST /catalogs/:name/publish"

RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/publish" RO)
assert_status "$RESP" "403" "RO cannot publish catalog (403)"

RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/publish" RW)
assert_status "$RESP" "403" "RW cannot publish catalog (403)"

RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/publish" Admin)
assert_status "$RESP" "200" "Admin can publish catalog (200)"

header "7.2: POST /catalogs/:name/unpublish"

RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/unpublish" RO)
assert_status "$RESP" "403" "RO cannot unpublish catalog (403)"

RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/unpublish" RW)
assert_status "$RESP" "403" "RW cannot unpublish catalog (403)"

RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/unpublish" Admin)
assert_status "$RESP" "200" "Admin can unpublish catalog (200)"

# ============================================================
# SECTION 8: Operational API — Export (Admin+ required)
# ============================================================
header "Section 8: Export (Admin+ required)"

header "8.1: GET /catalogs/:name/export"

RESP=$(api GET "$DATA_API/catalogs/$CATALOG_NAME/export" RO)
assert_status "$RESP" "403" "RO cannot export catalog (403)"

RESP=$(api GET "$DATA_API/catalogs/$CATALOG_NAME/export" RW)
assert_status "$RESP" "403" "RW cannot export catalog (403)"

RESP=$(api GET "$DATA_API/catalogs/$CATALOG_NAME/export" Admin)
assert_status "$RESP" "200" "Admin can export catalog (200)"

# ============================================================
# SECTION 9: Operational API — Import (Admin+ required)
# ============================================================
header "Section 9: Import (Admin+ required)"

# Get export data for import tests
EXPORT_BODY=$(get_body "$(api GET "$DATA_API/catalogs/$CATALOG_NAME/export" Admin)")
ET_NAME="${PREFIX}-server-${TIMESTAMP}"

header "9.1: POST /catalogs/import"

IMPORT_REQ=$(jq -n \
  --arg name "${PREFIX}-imported-${TIMESTAMP}" \
  --arg lbl "${PREFIX}-imp-cv-${TIMESTAMP}" \
  --arg et "$ET_NAME" \
  --argjson data "$EXPORT_BODY" '{
  catalog_name: $name,
  catalog_version_label: $lbl,
  reuse_existing: [$et],
  data: $data
}')

RESP=$(api POST "$DATA_API/catalogs/import" RO "$IMPORT_REQ")
assert_status "$RESP" "403" "RO cannot import catalog (403)"

RESP=$(api POST "$DATA_API/catalogs/import" RW "$IMPORT_REQ")
assert_status "$RESP" "403" "RW cannot import catalog (403)"

RESP=$(api POST "$DATA_API/catalogs/import" Admin "$IMPORT_REQ")
assert_status "$RESP" "201" "Admin can import catalog (201)"

# Clean up imported catalog
api DELETE "$DATA_API/catalogs/${PREFIX}-imported-${TIMESTAMP}" Admin > /dev/null 2>&1 || true
# Clean up imported CV
IMP_CV_ID=$(curl -s "$META_API/catalog-versions" -H 'X-User-Role: Admin' | jq -r ".items[] | select(.version_label | startswith(\"${PREFIX}-imp-cv-${TIMESTAMP}\")) | .id" 2>/dev/null | head -1)
if [ -n "$IMP_CV_ID" ]; then
  api DELETE "$META_API/catalog-versions/$IMP_CV_ID" Admin > /dev/null 2>&1 || true
fi

# ============================================================
# SECTION 10: Instance Operations on UNPUBLISHED catalog (RW+ required)
# ============================================================
header "Section 10: Instance CRUD on unpublished catalog (RW+ required)"

header "10.1: POST instances (create)"

RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/${PREFIX}-server-${TIMESTAMP}" RO \
  '{"name":"rbac-test-inst","description":"test"}')
assert_status "$RESP" "403" "RO cannot create instance (403)"

RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/${PREFIX}-server-${TIMESTAMP}" RW \
  '{"name":"rbac-test-inst","description":"test"}')
STATUS=$(get_status "$RESP")
INSTANCE_ID=$(get_body "$RESP" | jq -r '.id')
if [ "$STATUS" = "201" ]; then
  pass "RW can create instance on unpublished catalog (201)"
else
  fail "RW can create instance on unpublished catalog (201)" "expected=201 got=$STATUS"
fi

# Get instance version for update
INST_VER=$(get_body "$(api GET "$DATA_API/catalogs/$CATALOG_NAME/${PREFIX}-server-${TIMESTAMP}/$INSTANCE_ID" Admin)" | jq -r '.version')

header "10.2: PUT instances (update)"

RESP=$(api PUT "$DATA_API/catalogs/$CATALOG_NAME/${PREFIX}-server-${TIMESTAMP}/$INSTANCE_ID" RO \
  "{\"version\":$INST_VER,\"description\":\"updated\"}")
assert_status "$RESP" "403" "RO cannot update instance (403)"

RESP=$(api PUT "$DATA_API/catalogs/$CATALOG_NAME/${PREFIX}-server-${TIMESTAMP}/$INSTANCE_ID" RW \
  "{\"version\":$INST_VER,\"description\":\"updated by rw\"}")
assert_status "$RESP" "200" "RW can update instance on unpublished catalog (200)"

header "10.3: DELETE instances"

# Create another instance to test delete
RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/${PREFIX}-server-${TIMESTAMP}" Admin \
  '{"name":"rbac-del-inst","description":"delete test"}')
DEL_INST_ID=$(get_body "$RESP" | jq -r '.id')

RESP=$(api DELETE "$DATA_API/catalogs/$CATALOG_NAME/${PREFIX}-server-${TIMESTAMP}/$DEL_INST_ID" RO)
assert_status "$RESP" "403" "RO cannot delete instance (403)"

RESP=$(api DELETE "$DATA_API/catalogs/$CATALOG_NAME/${PREFIX}-server-${TIMESTAMP}/$DEL_INST_ID" RW)
assert_status "$RESP" "204" "RW can delete instance on unpublished catalog (204)"

# ============================================================
# SECTION 11: Published catalog — write protection (SuperAdmin only)
# ============================================================
header "Section 11: Published catalog write protection"

# Validate and publish
api POST "$DATA_API/catalogs/$CATALOG_NAME/validate" Admin > /dev/null
api POST "$DATA_API/catalogs/$CATALOG_NAME/publish" Admin > /dev/null

header "11.1: Create instance on published catalog"

RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/${PREFIX}-server-${TIMESTAMP}" RW \
  '{"name":"pub-test-rw","description":"test"}')
assert_status "$RESP" "403" "RW cannot create instance on published catalog (403)"

RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/${PREFIX}-server-${TIMESTAMP}" Admin \
  '{"name":"pub-test-admin","description":"test"}')
assert_status "$RESP" "403" "Admin cannot create instance on published catalog (403)"

RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/${PREFIX}-server-${TIMESTAMP}" SuperAdmin \
  '{"name":"pub-test-sa","description":"test"}')
assert_status "$RESP" "201" "SuperAdmin can create instance on published catalog (201)"
PUB_INST_ID=$(get_body "$RESP" | jq -r '.id')

header "11.2: Update instance on published catalog"

PUB_INST_VER=$(get_body "$(api GET "$DATA_API/catalogs/$CATALOG_NAME/${PREFIX}-server-${TIMESTAMP}/$PUB_INST_ID" Admin)" | jq -r '.version')

RESP=$(api PUT "$DATA_API/catalogs/$CATALOG_NAME/${PREFIX}-server-${TIMESTAMP}/$PUB_INST_ID" RW \
  "{\"version\":$PUB_INST_VER,\"description\":\"updated\"}")
assert_status "$RESP" "403" "RW cannot update instance on published catalog (403)"

RESP=$(api PUT "$DATA_API/catalogs/$CATALOG_NAME/${PREFIX}-server-${TIMESTAMP}/$PUB_INST_ID" Admin \
  "{\"version\":$PUB_INST_VER,\"description\":\"updated\"}")
assert_status "$RESP" "403" "Admin cannot update instance on published catalog (403)"

RESP=$(api PUT "$DATA_API/catalogs/$CATALOG_NAME/${PREFIX}-server-${TIMESTAMP}/$PUB_INST_ID" SuperAdmin \
  "{\"version\":$PUB_INST_VER,\"description\":\"updated by sa\"}")
assert_status "$RESP" "200" "SuperAdmin can update instance on published catalog (200)"

header "11.3: Delete instance on published catalog"

# Create another instance as SuperAdmin so we can test delete
RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/${PREFIX}-server-${TIMESTAMP}" SuperAdmin \
  '{"name":"pub-del-sa","description":"delete test"}')
PUB_DEL_INST_ID=$(get_body "$RESP" | jq -r '.id')

RESP=$(api DELETE "$DATA_API/catalogs/$CATALOG_NAME/${PREFIX}-server-${TIMESTAMP}/$PUB_DEL_INST_ID" RW)
assert_status "$RESP" "403" "RW cannot delete instance on published catalog (403)"

RESP=$(api DELETE "$DATA_API/catalogs/$CATALOG_NAME/${PREFIX}-server-${TIMESTAMP}/$PUB_DEL_INST_ID" Admin)
assert_status "$RESP" "403" "Admin cannot delete instance on published catalog (403)"

RESP=$(api DELETE "$DATA_API/catalogs/$CATALOG_NAME/${PREFIX}-server-${TIMESTAMP}/$PUB_DEL_INST_ID" SuperAdmin)
assert_status "$RESP" "204" "SuperAdmin can delete instance on published catalog (204)"

header "11.4: Validate on published catalog"

RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/validate" RW)
assert_status "$RESP" "403" "RW cannot validate published catalog (403)"

RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/validate" SuperAdmin)
assert_status "$RESP" "200" "SuperAdmin can validate published catalog (200)"

header "11.5: Delete published catalog"

RESP=$(api DELETE "$DATA_API/catalogs/$CATALOG_NAME" RW)
assert_status "$RESP" "403" "RW cannot delete published catalog (403)"

RESP=$(api DELETE "$DATA_API/catalogs/$CATALOG_NAME" Admin)
assert_status "$RESP" "403" "Admin cannot delete published catalog (403)"

# Unpublish before cleanup
api POST "$DATA_API/catalogs/$CATALOG_NAME/unpublish" Admin > /dev/null

# ============================================================
# SECTION 12: Catalog Copy/Replace RBAC
# ============================================================
header "Section 12: Catalog Copy/Replace"

header "12.1: POST /catalogs/copy (RW+ required)"

RESP=$(api POST "$DATA_API/catalogs/copy" RO \
  "{\"source\":\"$CATALOG_NAME\",\"name\":\"${PREFIX}-copy-ro-${TIMESTAMP}\"}")
assert_status "$RESP" "403" "RO cannot copy catalog (403)"

RESP=$(api POST "$DATA_API/catalogs/copy" RW \
  "{\"source\":\"$CATALOG_NAME\",\"name\":\"${PREFIX}-copy-rw-${TIMESTAMP}\"}")
STATUS=$(get_status "$RESP")
if [ "$STATUS" = "201" ]; then
  pass "RW can copy catalog (201)"
  api DELETE "$DATA_API/catalogs/${PREFIX}-copy-rw-${TIMESTAMP}" Admin > /dev/null 2>&1 || true
else
  fail "RW can copy catalog (201)" "expected=201 got=$STATUS body=$(get_body "$RESP" | head -c 200)"
fi

header "12.2: POST /catalogs/replace (Admin+ required)"

# Create source and target for replace test, validate source (required for replace)
api POST "$DATA_API/catalogs" Admin \
  "{\"name\":\"${PREFIX}-rep-src-${TIMESTAMP}\",\"description\":\"replace src\",\"catalog_version_id\":\"$CV_ID\"}" > /dev/null
api POST "$DATA_API/catalogs/${PREFIX}-rep-src-${TIMESTAMP}/validate" Admin > /dev/null
api POST "$DATA_API/catalogs" Admin \
  "{\"name\":\"${PREFIX}-rep-tgt-${TIMESTAMP}\",\"description\":\"replace tgt\",\"catalog_version_id\":\"$CV_ID\"}" > /dev/null

RESP=$(api POST "$DATA_API/catalogs/replace" RO \
  "{\"source\":\"${PREFIX}-rep-src-${TIMESTAMP}\",\"target\":\"${PREFIX}-rep-tgt-${TIMESTAMP}\"}")
assert_status "$RESP" "403" "RO cannot replace catalog (403)"

RESP=$(api POST "$DATA_API/catalogs/replace" RW \
  "{\"source\":\"${PREFIX}-rep-src-${TIMESTAMP}\",\"target\":\"${PREFIX}-rep-tgt-${TIMESTAMP}\"}")
assert_status "$RESP" "403" "RW cannot replace catalog (403)"

RESP=$(api POST "$DATA_API/catalogs/replace" Admin \
  "{\"source\":\"${PREFIX}-rep-src-${TIMESTAMP}\",\"target\":\"${PREFIX}-rep-tgt-${TIMESTAMP}\"}")
assert_status "$RESP" "200" "Admin can replace catalog (200)"

# Clean up replace test catalogs
api DELETE "$DATA_API/catalogs/${PREFIX}-rep-src-${TIMESTAMP}" Admin > /dev/null 2>&1 || true
api DELETE "$DATA_API/catalogs/${PREFIX}-rep-tgt-${TIMESTAMP}" Admin > /dev/null 2>&1 || true

# ============================================================
# SECTION 13: Read endpoints accessible by all roles
# ============================================================
header "Section 13: Read endpoints accessible by all roles"

header "13.1: GET /entity-types (all roles)"

RESP=$(api GET "$META_API/entity-types" RO)
assert_status "$RESP" "200" "RO can list entity types (200)"

RESP=$(api GET "$META_API/entity-types" RW)
assert_status "$RESP" "200" "RW can list entity types (200)"

header "13.2: GET /catalogs (all roles)"

RESP=$(api GET "$DATA_API/catalogs" RO)
assert_status "$RESP" "200" "RO can list catalogs (200)"

RESP=$(api GET "$DATA_API/catalogs" RW)
assert_status "$RESP" "200" "RW can list catalogs (200)"

header "13.3: GET /catalogs/:name (all roles)"

RESP=$(api GET "$DATA_API/catalogs/$CATALOG_NAME" RO)
assert_status "$RESP" "200" "RO can get catalog details (200)"

header "13.4: GET /catalog-versions (all roles)"

RESP=$(api GET "$META_API/catalog-versions" RO)
assert_status "$RESP" "200" "RO can list CVs (200)"

header "13.5: GET instances (all roles)"

RESP=$(api GET "$DATA_API/catalogs/$CATALOG_NAME/${PREFIX}-server-${TIMESTAMP}" RO)
assert_status "$RESP" "200" "RO can list instances (200)"

# ============================================================
# SECTION 14: Missing role header (401)
# ============================================================
header "Section 14: Missing/invalid role header"

RESP=$(curl -s -w "\n%{http_code}" -X GET "$META_API/entity-types" -H "Content-Type: application/json")
assert_status "$RESP" "401" "Missing X-User-Role header returns 401"

RESP=$(curl -s -w "\n%{http_code}" -X GET "$META_API/entity-types" \
  -H "X-User-Role: InvalidRole" -H "Content-Type: application/json")
assert_status "$RESP" "401" "Invalid X-User-Role header returns 401"

# ============================================================
# Results
# ============================================================
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
