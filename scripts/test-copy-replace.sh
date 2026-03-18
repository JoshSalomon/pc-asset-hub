#!/usr/bin/env bash
# Live test script for Copy & Replace Catalog (Phase 8 / FF-8)
# Usage: ./scripts/test-copy-replace.sh [API_BASE_URL]
set -uo pipefail

API="${1:-http://localhost:30080}"
PASS=0; FAIL=0; TOTAL=0

pass() { ((PASS++)); ((TOTAL++)); echo "  PASS: $1"; }
fail() { ((FAIL++)); ((TOTAL++)); echo "  FAIL: $1 — $2"; }

h() { echo ""; echo "=== $1 ==="; }

cleanup() {
  h "Cleanup"
  for name in source-cat target-cat copy-cat staging-cat prod-cat prod-cat-archive replace-src replace-tgt replace-archive; do
    curl -s -o /dev/null "$API/api/data/v1/catalogs/$name" -X DELETE -H 'X-User-Role: SuperAdmin' 2>/dev/null || true
  done
  # Clean up CV and entity type
  if [ -n "${CV_ID:-}" ]; then
    curl -s -o /dev/null "$API/api/meta/v1/catalog-versions/$CV_ID" -X DELETE -H 'X-User-Role: Admin' 2>/dev/null || true
  fi
  if [ -n "${ET_ID:-}" ]; then
    curl -s -o /dev/null "$API/api/meta/v1/entity-types/$ET_ID" -X DELETE -H 'X-User-Role: Admin' 2>/dev/null || true
  fi
  echo "  Cleaned up test data"
}
trap cleanup EXIT

# --- Setup: Create entity type, CV, pin ---
h "Setup"
ET_SUFFIX=$(date +%s)
ET_RESP=$(curl -s "$API/api/meta/v1/entity-types" -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d "{\"name\":\"copy-test-et-$ET_SUFFIX\"}")
ET_ID=$(echo "$ET_RESP" | jq -r '.entity_type.id')
ETV_ID=$(echo "$ET_RESP" | jq -r '.version.id')
echo "  Entity type version: $ETV_ID"

ET_NAME="copy-test-et-$ET_SUFFIX"

CV_ID=$(curl -s "$API/api/meta/v1/catalog-versions" -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d "{\"version_label\":\"copy-test-v1-$ET_SUFFIX\",\"pins\":[{\"entity_type_version_id\":\"$ETV_ID\"}]}" | jq -r '.id')
echo "  Catalog version: $CV_ID"

if [ "$ET_ID" = "null" ] || [ "$CV_ID" = "null" ]; then
  echo "  FATAL: Setup failed — ET_ID=$ET_ID CV_ID=$CV_ID"
  exit 1
fi

# Pre-cleanup: delete catalogs from previous failed runs
for name in source-cat target-cat copy-cat staging-cat prod-cat prod-cat-archive replace-src replace-tgt replace-archive; do
  curl -s -o /dev/null "$API/api/data/v1/catalogs/$name" -X DELETE -H 'X-User-Role: SuperAdmin' 2>/dev/null || true
done

# --- Test 1: Copy empty catalog ---
h "Test 1: Copy empty catalog"
CREATE_RESP=$(curl -s "$API/api/data/v1/catalogs" -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d "{\"name\":\"source-cat\",\"catalog_version_id\":\"$CV_ID\"}")
echo "  Created source: $(echo "$CREATE_RESP" | jq -r '.name // .message // "error"')"

CODE=$(curl -s -o /tmp/copy-resp.json -w "%{http_code}" "$API/api/data/v1/catalogs/copy" -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d '{"source":"source-cat","name":"copy-cat","description":"copied!"}')

if [ "$CODE" = "201" ]; then
  pass "Copy returns 201"
else
  fail "Copy returns 201" "got $CODE — $(cat /tmp/copy-resp.json)"
fi

NAME=$(jq -r '.name' /tmp/copy-resp.json)
if [ "$NAME" = "copy-cat" ]; then
  pass "Copy catalog has correct name"
else
  fail "Copy catalog has correct name" "got $NAME"
fi

STATUS=$(jq -r '.validation_status' /tmp/copy-resp.json)
if [ "$STATUS" = "draft" ]; then
  pass "Copy catalog is draft"
else
  fail "Copy catalog is draft" "got $STATUS"
fi

DESC=$(jq -r '.description' /tmp/copy-resp.json)
if [ "$DESC" = "copied!" ]; then
  pass "Copy catalog has custom description"
else
  fail "Copy catalog has custom description" "got $DESC"
fi

# --- Test 2: Copy with instances ---
h "Test 2: Copy catalog with instances"
# Create an instance in source-cat
curl -s "$API/api/data/v1/catalogs/source-cat/$ET_NAME" -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d '{"name":"inst-1","description":"first instance"}' > /dev/null

# Copy to target-cat
CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API/api/data/v1/catalogs/copy" -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d '{"source":"source-cat","name":"target-cat"}')

if [ "$CODE" = "201" ]; then
  pass "Copy with instances returns 201"
else
  fail "Copy with instances returns 201" "got $CODE"
fi

# Verify instance exists in target
INST_RESP=$(curl -s "$API/api/data/v1/catalogs/target-cat/$ET_NAME" -H 'X-User-Role: Admin')
INST_COUNT=$(echo "$INST_RESP" | jq '.total')
if [ "$INST_COUNT" = "1" ]; then
  pass "Copied instance exists in target catalog"
else
  fail "Copied instance exists in target catalog" "got count=$INST_COUNT"
fi

INST_NAME=$(echo "$INST_RESP" | jq -r '.items[0].name')
if [ "$INST_NAME" = "inst-1" ]; then
  pass "Copied instance has correct name"
else
  fail "Copied instance has correct name" "got $INST_NAME"
fi

# --- Test 3: Copy nonexistent source → 404 ---
h "Test 3: Copy nonexistent source"
CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API/api/data/v1/catalogs/copy" -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d '{"source":"nonexistent","name":"target2"}')
if [ "$CODE" = "404" ]; then
  pass "Copy nonexistent source returns 404"
else
  fail "Copy nonexistent source returns 404" "got $CODE"
fi

# --- Test 4: Copy duplicate target → 409 ---
h "Test 4: Copy duplicate target"
CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API/api/data/v1/catalogs/copy" -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d '{"source":"source-cat","name":"copy-cat"}')
if [ "$CODE" = "409" ]; then
  pass "Copy duplicate target returns 409"
else
  fail "Copy duplicate target returns 409" "got $CODE"
fi

# --- Test 5: Copy as RO → 403 ---
h "Test 5: Copy RBAC"
CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API/api/data/v1/catalogs/copy" -H 'X-User-Role: RO' -H 'Content-Type: application/json' \
  -d '{"source":"source-cat","name":"ro-copy"}')
if [ "$CODE" = "403" ]; then
  pass "Copy as RO returns 403"
else
  fail "Copy as RO returns 403" "got $CODE"
fi

# --- Test 6: Replace — basic swap ---
h "Test 6: Replace basic swap"
# Create staging (valid) and prod catalogs
curl -s "$API/api/data/v1/catalogs" -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d "{\"name\":\"staging-cat\",\"catalog_version_id\":\"$CV_ID\"}" > /dev/null
curl -s "$API/api/data/v1/catalogs" -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d "{\"name\":\"prod-cat\",\"catalog_version_id\":\"$CV_ID\"}" > /dev/null

# Validate staging to make it valid
curl -s "$API/api/data/v1/catalogs/staging-cat/validate" -X POST -H 'X-User-Role: Admin' > /dev/null

CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API/api/data/v1/catalogs/replace" -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d '{"source":"staging-cat","target":"prod-cat","archive_name":"prod-cat-archive"}')

if [ "$CODE" = "200" ]; then
  pass "Replace returns 200"
else
  fail "Replace returns 200" "got $CODE"
fi

# Verify staging-cat no longer exists (was renamed to prod-cat)
CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API/api/data/v1/catalogs/staging-cat" -H 'X-User-Role: Admin')
if [ "$CODE" = "404" ]; then
  pass "Old source name no longer exists"
else
  fail "Old source name no longer exists" "got $CODE"
fi

# Verify prod-cat exists (was the source, now renamed)
CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API/api/data/v1/catalogs/prod-cat" -H 'X-User-Role: Admin')
if [ "$CODE" = "200" ]; then
  pass "Target name now serves source data"
else
  fail "Target name now serves source data" "got $CODE"
fi

# Verify archive exists
CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API/api/data/v1/catalogs/prod-cat-archive" -H 'X-User-Role: Admin')
if [ "$CODE" = "200" ]; then
  pass "Archive catalog exists"
else
  fail "Archive catalog exists" "got $CODE"
fi

# --- Test 7: Replace — source must be valid ---
h "Test 7: Replace requires valid source"
curl -s "$API/api/data/v1/catalogs" -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d "{\"name\":\"replace-src\",\"catalog_version_id\":\"$CV_ID\"}" > /dev/null
curl -s "$API/api/data/v1/catalogs" -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d "{\"name\":\"replace-tgt\",\"catalog_version_id\":\"$CV_ID\"}" > /dev/null

# replace-src is draft, should fail
CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API/api/data/v1/catalogs/replace" -H 'X-User-Role: Admin' -H 'Content-Type: application/json' \
  -d '{"source":"replace-src","target":"replace-tgt","archive_name":"replace-archive"}')
if [ "$CODE" = "400" ]; then
  pass "Replace draft source returns 400"
else
  fail "Replace draft source returns 400" "got $CODE"
fi

# --- Test 8: Replace RBAC ---
h "Test 8: Replace RBAC"
CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API/api/data/v1/catalogs/replace" -H 'X-User-Role: RW' -H 'Content-Type: application/json' \
  -d '{"source":"replace-src","target":"replace-tgt"}')
if [ "$CODE" = "403" ]; then
  pass "Replace as RW returns 403"
else
  fail "Replace as RW returns 403" "got $CODE"
fi

CODE=$(curl -s -o /dev/null -w "%{http_code}" "$API/api/data/v1/catalogs/replace" -H 'X-User-Role: RO' -H 'Content-Type: application/json' \
  -d '{"source":"replace-src","target":"replace-tgt"}')
if [ "$CODE" = "403" ]; then
  pass "Replace as RO returns 403"
else
  fail "Replace as RO returns 403" "got $CODE"
fi

# --- Summary ---
h "Results"
echo "  Total: $TOTAL  Pass: $PASS  Fail: $FAIL"
if [ "$FAIL" -gt 0 ]; then
  echo "  SOME TESTS FAILED"
  exit 1
else
  echo "  ALL TESTS PASSED"
fi
