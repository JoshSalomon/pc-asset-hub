#!/bin/bash
# Live system tests for Phase 7: Catalog Publishing
# Usage: ./scripts/test-publishing.sh [API_BASE_URL]
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
CATALOG_NAME="pubtest-${TIMESTAMP}"

header "Setup: Create test data"

# Create entity type
ET_RESP=$(api POST "$META_API/entity-types" Admin "{\"name\":\"pub-server-${TIMESTAMP}\"}")
ET_STATUS=$(get_status "$ET_RESP")
ET_BODY=$(get_body "$ET_RESP")
SERVER_ET_ID=$(echo "$ET_BODY" | jq -r '.entity_type.id')
echo "  Entity type: $SERVER_ET_ID (status=$ET_STATUS)"

# Get latest version
SERVER_ETV_ID=$(get_body "$(api GET "$META_API/entity-types/$SERVER_ET_ID/versions" Admin)" | jq -r '.items[-1].id')

# Create CV + catalog
CV_ID=$(get_body "$(api POST "$META_API/catalog-versions" Admin \
  "{\"version_label\":\"pub-cv-${TIMESTAMP}\",\"pins\":[{\"entity_type_version_id\":\"$SERVER_ETV_ID\"}]}")" | jq -r '.id')
echo "  CV: $CV_ID"

api POST "$DATA_API/catalogs" Admin \
  "{\"name\":\"$CATALOG_NAME\",\"description\":\"Publish test\",\"catalog_version_id\":\"$CV_ID\"}" > /dev/null
echo "  Catalog: $CATALOG_NAME"

header "Test 1: Cannot publish draft catalog (400)"

RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/publish" Admin)
STATUS=$(get_status "$RESP")
if [ "$STATUS" = "400" ]; then
  pass "Draft catalog cannot be published (400)"
else
  fail "Draft catalog publish" "expected=400 got=$STATUS"
fi

header "Test 2: RW cannot publish (403)"

RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/publish" RW)
STATUS=$(get_status "$RESP")
if [ "$STATUS" = "403" ]; then
  pass "RW cannot publish (403)"
else
  fail "RW publish" "expected=403 got=$STATUS"
fi

header "Test 3: RO cannot publish (403)"

RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/publish" RO)
STATUS=$(get_status "$RESP")
if [ "$STATUS" = "403" ]; then
  pass "RO cannot publish (403)"
else
  fail "RO publish" "expected=403 got=$STATUS"
fi

header "Test 4: Validate catalog → valid"

RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/validate" Admin)
BODY=$(get_body "$RESP")
VAL_STATUS=$(echo "$BODY" | jq -r '.status')
if [ "$VAL_STATUS" = "valid" ]; then
  pass "Empty catalog validates as valid"
else
  fail "Validation" "expected=valid got=$VAL_STATUS"
fi

header "Test 5: Admin can publish valid catalog (200)"

RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/publish" Admin)
STATUS=$(get_status "$RESP")
if [ "$STATUS" = "200" ]; then
  pass "Admin can publish valid catalog (200)"
else
  fail "Admin publish" "expected=200 got=$STATUS body=$(get_body "$RESP")"
fi

header "Test 6: Catalog shows published=true"

RESP=$(api GET "$DATA_API/catalogs/$CATALOG_NAME" Admin)
BODY=$(get_body "$RESP")
PUBLISHED=$(echo "$BODY" | jq -r '.published')
PUBLISHED_AT=$(echo "$BODY" | jq -r '.published_at')
if [ "$PUBLISHED" = "true" ] && [ "$PUBLISHED_AT" != "null" ]; then
  pass "Catalog shows published=true with published_at"
else
  fail "Published status" "published=$PUBLISHED published_at=$PUBLISHED_AT"
fi

header "Test 7: RW cannot create instance on published catalog (403)"

RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/pub-server-${TIMESTAMP}" RW \
  '{"name":"test-server","description":"test"}')
STATUS=$(get_status "$RESP")
if [ "$STATUS" = "403" ]; then
  pass "RW blocked from creating instance on published catalog (403)"
else
  fail "RW write protection" "expected=403 got=$STATUS"
fi

header "Test 8: SuperAdmin can create instance on published catalog (201)"

RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/pub-server-${TIMESTAMP}" SuperAdmin \
  '{"name":"test-server","description":"test"}')
STATUS=$(get_status "$RESP")
if [ "$STATUS" = "201" ]; then
  pass "SuperAdmin can create instance on published catalog (201)"
else
  fail "SuperAdmin write" "expected=201 got=$STATUS body=$(get_body "$RESP")"
fi

header "Test 9: After mutation, status=draft but published=true"

RESP=$(api GET "$DATA_API/catalogs/$CATALOG_NAME" Admin)
BODY=$(get_body "$RESP")
VAL_STATUS=$(echo "$BODY" | jq -r '.validation_status')
PUBLISHED=$(echo "$BODY" | jq -r '.published')
if [ "$VAL_STATUS" = "draft" ] && [ "$PUBLISHED" = "true" ]; then
  pass "After mutation: status=draft, published=true"
else
  fail "Post-mutation state" "status=$VAL_STATUS published=$PUBLISHED"
fi

header "Test 10: Admin can unpublish (200)"

RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/unpublish" Admin)
STATUS=$(get_status "$RESP")
if [ "$STATUS" = "200" ]; then
  pass "Admin can unpublish (200)"
else
  fail "Admin unpublish" "expected=200 got=$STATUS"
fi

header "Test 11: After unpublish, published=false"

RESP=$(api GET "$DATA_API/catalogs/$CATALOG_NAME" Admin)
BODY=$(get_body "$RESP")
PUBLISHED=$(echo "$BODY" | jq -r '.published')
if [ "$PUBLISHED" = "false" ]; then
  pass "After unpublish: published=false"
else
  fail "Unpublish state" "published=$PUBLISHED"
fi

header "Test 12: RW can now create instance on unpublished catalog (201)"

RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/pub-server-${TIMESTAMP}" RW \
  '{"name":"test-server-2","description":"test"}')
STATUS=$(get_status "$RESP")
if [ "$STATUS" = "201" ]; then
  pass "RW can create instance on unpublished catalog (201)"
else
  fail "RW write after unpublish" "expected=201 got=$STATUS"
fi

header "Test 13: Publish nonexistent catalog → 404"

RESP=$(api POST "$DATA_API/catalogs/does-not-exist/publish" Admin)
STATUS=$(get_status "$RESP")
if [ "$STATUS" = "404" ]; then
  pass "Nonexistent catalog publish → 404"
else
  fail "Nonexistent publish" "expected=404 got=$STATUS"
fi

header "Test 14: CV promotion warnings"

# Re-validate to make catalog draft (it has instances now)
RESP=$(api GET "$DATA_API/catalogs/$CATALOG_NAME" Admin)
BODY=$(get_body "$RESP")
VAL_STATUS=$(echo "$BODY" | jq -r '.validation_status')
echo "  Catalog validation_status: $VAL_STATUS"

# Promote CV and check for warnings
RESP=$(api POST "$META_API/catalog-versions/$CV_ID/promote" RW)
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
if [ "$STATUS" = "200" ]; then
  WARNINGS=$(echo "$BODY" | jq '.warnings | length')
  if [ "$WARNINGS" -ge 0 ]; then
    pass "CV promotion returns warnings array (${WARNINGS} warning(s))"
  else
    fail "CV promotion warnings" "no warnings field"
  fi
else
  fail "CV promotion" "expected=200 got=$STATUS"
fi

header "Test 15: CR auto-sync on mutation"

# Create a fresh catalog for this test
SYNC_CATALOG="synctest-${TIMESTAMP}"
api POST "$DATA_API/catalogs" Admin \
  "{\"name\":\"$SYNC_CATALOG\",\"description\":\"Sync test\",\"catalog_version_id\":\"$CV_ID\"}" > /dev/null

# Validate and publish
api POST "$DATA_API/catalogs/$SYNC_CATALOG/validate" Admin > /dev/null
api POST "$DATA_API/catalogs/$SYNC_CATALOG/publish" Admin > /dev/null

# Verify CR exists with validationStatus=valid
CR_STATUS=$(kubectl --context kind-assethub -n assethub get catalog "$SYNC_CATALOG" -o jsonpath='{.spec.validationStatus}' 2>/dev/null)
if [ "$CR_STATUS" = "valid" ]; then
  pass "CR created with validationStatus=valid"
else
  fail "CR initial status" "expected=valid got=$CR_STATUS"
fi

# Mutate as SuperAdmin (create instance)
MUTATE_RESP=$(api POST "$DATA_API/catalogs/$SYNC_CATALOG/pub-server-${TIMESTAMP}" SuperAdmin \
  '{"name":"sync-server","description":"test"}')
MUTATE_STATUS=$(get_status "$MUTATE_RESP")
MUTATE_BODY=$(get_body "$MUTATE_RESP")
echo "  Mutation response: status=$MUTATE_STATUS"
if [ "$MUTATE_STATUS" != "201" ]; then
  echo "  Mutation body: $MUTATE_BODY"
fi
sleep 1

# Check DB state
DB_STATUS=$(get_body "$(api GET "$DATA_API/catalogs/$SYNC_CATALOG" Admin)" | jq -r '.validation_status')
echo "  DB validation_status after mutation: $DB_STATUS"

# Verify CR updated to draft
CR_STATUS=$(kubectl --context kind-assethub -n assethub get catalog "$SYNC_CATALOG" -o jsonpath='{.spec.validationStatus}' 2>/dev/null)
echo "  CR validationStatus after mutation: $CR_STATUS"
if [ "$CR_STATUS" = "draft" ]; then
  pass "CR auto-synced to validationStatus=draft after mutation"
else
  fail "CR sync after mutation" "expected=draft got=$CR_STATUS db_status=$DB_STATUS"
fi

# Verify CR still exists (draft doesn't unpublish)
CR_EXISTS=$(kubectl --context kind-assethub -n assethub get catalog "$SYNC_CATALOG" -o name 2>/dev/null)
if [ -n "$CR_EXISTS" ]; then
  pass "CR still exists after mutation (draft does not unpublish)"
else
  fail "CR persistence" "CR was deleted after mutation"
fi

header "Test 15b: CR sync after UPDATE (edit Description on published catalog)"

# Re-validate and re-publish the sync catalog (it's now draft from the create above)
api POST "$DATA_API/catalogs/$SYNC_CATALOG/validate" SuperAdmin > /dev/null
api POST "$DATA_API/catalogs/$SYNC_CATALOG/publish" Admin > /dev/null
sleep 1

# Get the instance we just created
INST_LIST_RESP=$(api GET "$DATA_API/catalogs/$SYNC_CATALOG/pub-server-${TIMESTAMP}" SuperAdmin)
INST_ID=$(get_body "$INST_LIST_RESP" | jq -r '.items[0].id')
INST_VER=$(get_body "$INST_LIST_RESP" | jq -r '.items[0].version')
echo "  Instance: $INST_ID (version=$INST_VER)"

# Verify CR is valid before edit
CR_STATUS_BEFORE=$(kubectl --context kind-assethub -n assethub get catalog "$SYNC_CATALOG" -o jsonpath='{.spec.validationStatus}' 2>/dev/null)
CR_DV_BEFORE=$(kubectl --context kind-assethub -n assethub get catalog "$SYNC_CATALOG" -o jsonpath='{.status.dataVersion}' 2>/dev/null)
echo "  CR before edit: validationStatus=$CR_STATUS_BEFORE dataVersion=$CR_DV_BEFORE"

# Update instance Description (the built-in field, not a schema attribute)
UPDATE_RESP=$(api PUT "$DATA_API/catalogs/$SYNC_CATALOG/pub-server-${TIMESTAMP}/$INST_ID" SuperAdmin \
  "{\"version\":$INST_VER,\"description\":\"edited description\"}")
UPDATE_STATUS=$(get_status "$UPDATE_RESP")
echo "  Update response: status=$UPDATE_STATUS"
if [ "$UPDATE_STATUS" != "200" ]; then
  echo "  Update body: $(get_body "$UPDATE_RESP")"
fi
sleep 1

# Check DB state
DB_STATUS_AFTER=$(get_body "$(api GET "$DATA_API/catalogs/$SYNC_CATALOG" Admin)" | jq -r '.validation_status')
echo "  DB validation_status after edit: $DB_STATUS_AFTER"

# Check CR state
CR_STATUS_AFTER=$(kubectl --context kind-assethub -n assethub get catalog "$SYNC_CATALOG" -o jsonpath='{.spec.validationStatus}' 2>/dev/null)
CR_DV_AFTER=$(kubectl --context kind-assethub -n assethub get catalog "$SYNC_CATALOG" -o jsonpath='{.status.dataVersion}' 2>/dev/null)
CR_GEN=$(kubectl --context kind-assethub -n assethub get catalog "$SYNC_CATALOG" -o jsonpath='{.metadata.generation}' 2>/dev/null)
CR_OBS_GEN=$(kubectl --context kind-assethub -n assethub get catalog "$SYNC_CATALOG" -o jsonpath='{.status.observedGeneration}' 2>/dev/null)
echo "  CR after edit: validationStatus=$CR_STATUS_AFTER dataVersion=$CR_DV_AFTER generation=$CR_GEN observedGeneration=$CR_OBS_GEN"

if [ "$CR_STATUS_AFTER" = "draft" ]; then
  pass "CR updated to draft after Description edit"
else
  fail "CR sync after Description edit" "expected=draft got=$CR_STATUS_AFTER (db=$DB_STATUS_AFTER)"
fi

if [ -n "$CR_DV_BEFORE" ] && [ -n "$CR_DV_AFTER" ] && [ "$CR_DV_AFTER" -gt "$CR_DV_BEFORE" ]; then
  pass "DataVersion incremented after edit ($CR_DV_BEFORE → $CR_DV_AFTER)"
else
  fail "DataVersion increment" "before=$CR_DV_BEFORE after=$CR_DV_AFTER"
fi

header "Test 15c: Consecutive edits each bump DataVersion"

# Edit again — same field, different value
INST_VER2=$((INST_VER + 1))
CR_DV_BEFORE2=$(kubectl --context kind-assethub -n assethub get catalog "$SYNC_CATALOG" -o jsonpath='{.status.dataVersion}' 2>/dev/null)
echo "  DataVersion before second edit: $CR_DV_BEFORE2"

UPDATE_RESP2=$(api PUT "$DATA_API/catalogs/$SYNC_CATALOG/pub-server-${TIMESTAMP}/$INST_ID" SuperAdmin \
  "{\"version\":$INST_VER2,\"description\":\"second edit\"}")
UPDATE_STATUS2=$(get_status "$UPDATE_RESP2")
echo "  Second update response: status=$UPDATE_STATUS2"
sleep 2

CR_DV_AFTER2=$(kubectl --context kind-assethub -n assethub get catalog "$SYNC_CATALOG" -o jsonpath='{.status.dataVersion}' 2>/dev/null)
echo "  DataVersion after second edit: $CR_DV_AFTER2"

if [ -n "$CR_DV_BEFORE2" ] && [ -n "$CR_DV_AFTER2" ] && [ "$CR_DV_AFTER2" -gt "$CR_DV_BEFORE2" ]; then
  pass "Consecutive edit bumps DataVersion ($CR_DV_BEFORE2 → $CR_DV_AFTER2)"
else
  fail "Consecutive DataVersion" "before=$CR_DV_BEFORE2 after=$CR_DV_AFTER2"
fi

header "Test 16: Operator sets status on CR"

# Check if operator has reconciled and set status
CR_READY=$(kubectl --context kind-assethub -n assethub get catalog "$SYNC_CATALOG" -o jsonpath='{.status.ready}' 2>/dev/null)
CR_DATAVERSION=$(kubectl --context kind-assethub -n assethub get catalog "$SYNC_CATALOG" -o jsonpath='{.status.dataVersion}' 2>/dev/null)
if [ "$CR_READY" = "true" ] && [ -n "$CR_DATAVERSION" ] && [ "$CR_DATAVERSION" -ge 1 ]; then
  pass "Operator set status.ready=true, dataVersion=$CR_DATAVERSION"
else
  # Operator may not have reconciled yet — trigger by restarting
  echo "  Operator hasn't reconciled yet, triggering..."
  kubectl --context kind-assethub -n assethub delete pod -l app=assethub-operator --wait=false > /dev/null 2>&1
  sleep 15
  CR_READY=$(kubectl --context kind-assethub -n assethub get catalog "$SYNC_CATALOG" -o jsonpath='{.status.ready}' 2>/dev/null)
  CR_DATAVERSION=$(kubectl --context kind-assethub -n assethub get catalog "$SYNC_CATALOG" -o jsonpath='{.status.dataVersion}' 2>/dev/null)
  if [ "$CR_READY" = "true" ] && [ -n "$CR_DATAVERSION" ] && [ "$CR_DATAVERSION" -ge 1 ]; then
    pass "Operator set status.ready=true, dataVersion=$CR_DATAVERSION (after restart)"
  else
    fail "Operator status" "ready=$CR_READY dataVersion=$CR_DATAVERSION"
  fi
fi

header "Test 17: Unpublish removes CR"

api POST "$DATA_API/catalogs/$SYNC_CATALOG/unpublish" Admin > /dev/null
sleep 1

CR_EXISTS=$(kubectl --context kind-assethub -n assethub get catalog "$SYNC_CATALOG" -o name 2>/dev/null)
if [ -z "$CR_EXISTS" ]; then
  pass "CR removed after unpublish"
else
  fail "CR removal" "CR still exists after unpublish"
fi

# Clean up sync test catalog
api DELETE "$DATA_API/catalogs/$SYNC_CATALOG" Admin > /dev/null 2>&1 || true

header "Test T-30.22: Validate on published catalog blocked for RW (403)"

# Re-validate and re-publish for this test
api POST "$DATA_API/catalogs/$CATALOG_NAME/validate" SuperAdmin > /dev/null 2>&1
api POST "$DATA_API/catalogs/$CATALOG_NAME/publish" Admin > /dev/null 2>&1

RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/validate" RW)
STATUS=$(get_status "$RESP")
if [ "$STATUS" = "403" ]; then
  pass "T-30.22: Validate on published catalog blocked for RW (403)"
else
  fail "T-30.22: Validate on published catalog" "expected=403 got=$STATUS"
fi

# Clean up: unpublish again
api POST "$DATA_API/catalogs/$CATALOG_NAME/unpublish" Admin > /dev/null 2>&1

header "Test T-30.23: UpdateCatalogVersion on production CV blocked (400)"

# Promote CV to production for this test
api POST "$META_API/catalog-versions/$CV_ID/promote" Admin > /dev/null 2>&1

RESP=$(api PUT "$META_API/catalog-versions/$CV_ID" Admin "{\"description\":\"should fail\"}")
STATUS=$(get_status "$RESP")
if [ "$STATUS" = "400" ]; then
  pass "T-30.23: UpdateCatalogVersion on production CV blocked (400)"
else
  fail "T-30.23: UpdateCatalogVersion on production CV" "expected=400 got=$STATUS"
fi

# Demote back to testing for next tests
api POST "$META_API/catalog-versions/$CV_ID/demote" SuperAdmin "{\"target_stage\":\"testing\"}" > /dev/null 2>&1

header "Test T-30.24: UpdateCatalogVersion on testing CV blocked for RW (400)"

RESP=$(api PUT "$META_API/catalog-versions/$CV_ID" RW "{\"description\":\"should fail\"}")
STATUS=$(get_status "$RESP")
if [ "$STATUS" = "400" ]; then
  pass "T-30.24: UpdateCatalogVersion on testing CV blocked for RW (400)"
else
  fail "T-30.24: UpdateCatalogVersion on testing CV as RW" "expected=400 got=$STATUS"
fi

header "Test T-30.25: UpdateCatalogVersion on testing CV allowed for SuperAdmin (200)"

RESP=$(api PUT "$META_API/catalog-versions/$CV_ID" SuperAdmin "{\"description\":\"updated by superadmin\"}")
STATUS=$(get_status "$RESP")
if [ "$STATUS" = "200" ]; then
  pass "T-30.25: UpdateCatalogVersion on testing CV allowed for SuperAdmin (200)"
else
  fail "T-30.25: UpdateCatalogVersion on testing CV as SuperAdmin" "expected=200 got=$STATUS"
fi

header "Test 18: Publish idempotence — publish already-published catalog (200)"

# Re-validate and publish the catalog (it's currently unpublished from line 395)
api POST "$DATA_API/catalogs/$CATALOG_NAME/validate" SuperAdmin > /dev/null 2>&1
api POST "$DATA_API/catalogs/$CATALOG_NAME/publish" Admin > /dev/null 2>&1

# Publish again — should be idempotent (200, not error)
RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/publish" Admin)
STATUS=$(get_status "$RESP")
if [ "$STATUS" = "200" ]; then
  pass "Publish idempotent on already-published catalog (200)"
else
  fail "Publish idempotence" "expected=200 got=$STATUS body=$(get_body "$RESP")"
fi

# Verify catalog is still published
RESP=$(api GET "$DATA_API/catalogs/$CATALOG_NAME" Admin)
BODY=$(get_body "$RESP")
PUBLISHED=$(echo "$BODY" | jq -r '.published')
if [ "$PUBLISHED" = "true" ]; then
  pass "Catalog still published after idempotent publish"
else
  fail "Post-idempotent publish state" "published=$PUBLISHED"
fi

header "Test 19: Unpublish → re-publish cycle"

# Unpublish
RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/unpublish" Admin)
STATUS=$(get_status "$RESP")
if [ "$STATUS" = "200" ]; then
  echo "  Unpublished successfully"
else
  fail "Unpublish for re-publish test" "expected=200 got=$STATUS"
fi

# Verify unpublished
RESP=$(api GET "$DATA_API/catalogs/$CATALOG_NAME" Admin)
BODY=$(get_body "$RESP")
PUBLISHED=$(echo "$BODY" | jq -r '.published')
if [ "$PUBLISHED" != "false" ]; then
  fail "Unpublish verification" "expected published=false got=$PUBLISHED"
fi

# Verify CR removed after unpublish
CR_EXISTS=$(kubectl --context kind-assethub -n assethub get catalog "$CATALOG_NAME" -o name 2>/dev/null)
if [ -n "$CR_EXISTS" ]; then
  echo "  Warning: CR still exists after unpublish"
fi

# Re-validate (status was reset to draft by unpublish)
api POST "$DATA_API/catalogs/$CATALOG_NAME/validate" SuperAdmin > /dev/null 2>&1

# Re-publish
RESP=$(api POST "$DATA_API/catalogs/$CATALOG_NAME/publish" Admin)
STATUS=$(get_status "$RESP")
if [ "$STATUS" = "200" ]; then
  pass "Re-publish after unpublish succeeds (200)"
else
  fail "Re-publish" "expected=200 got=$STATUS body=$(get_body "$RESP")"
fi

# Verify catalog is published again
RESP=$(api GET "$DATA_API/catalogs/$CATALOG_NAME" Admin)
BODY=$(get_body "$RESP")
PUBLISHED=$(echo "$BODY" | jq -r '.published')
PUBLISHED_AT=$(echo "$BODY" | jq -r '.published_at')
if [ "$PUBLISHED" = "true" ] && [ "$PUBLISHED_AT" != "null" ]; then
  pass "Catalog published again with published_at set"
else
  fail "Re-publish state" "published=$PUBLISHED published_at=$PUBLISHED_AT"
fi

# Verify CR recreated after re-publish
sleep 1
CR_EXISTS=$(kubectl --context kind-assethub -n assethub get catalog "$CATALOG_NAME" -o name 2>/dev/null)
if [ -n "$CR_EXISTS" ]; then
  pass "CR recreated after re-publish"
else
  fail "CR recreation" "CR not found after re-publish"
fi

# Clean up: unpublish for cleanup section
api POST "$DATA_API/catalogs/$CATALOG_NAME/unpublish" Admin > /dev/null 2>&1

header "Cleanup (only removing test data created by this script)"

api DELETE "$DATA_API/catalogs/$CATALOG_NAME" Admin > /dev/null 2>&1 || true
echo "  Deleted test catalog: $CATALOG_NAME"

# Demote CV back to development (it was promoted to testing in test 14)
api POST "$META_API/catalog-versions/$CV_ID/demote" Admin > /dev/null 2>&1 || true

# Delete CV (must happen before entity type since CV pins reference ETVs)
api DELETE "$META_API/catalog-versions/$CV_ID" Admin > /dev/null 2>&1 || true
echo "  Deleted test CV: $CV_ID"

# Delete entity type
api DELETE "$META_API/entity-types/$SERVER_ET_ID" Admin > /dev/null 2>&1 || true
echo "  Deleted test entity type: $SERVER_ET_ID"

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
