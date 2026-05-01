#!/usr/bin/env bash
# Live test script for API error handling: malformed inputs, wrong Content-Types,
# boundary values, and error response format consistency.
# Usage: ./scripts/test-error-handling.sh [API_BASE_URL]
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

# Like api() but with custom Content-Type
api_ct() {
  local method="$1" path="$2" role="$3" ct="$4" body="${5:-}"
  if [ -n "$body" ]; then
    curl -s -w "\n%{http_code}" -X "$method" "$path" \
      -H "X-User-Role: $role" -H "Content-Type: $ct" -d "$body"
  else
    curl -s -w "\n%{http_code}" -X "$method" "$path" \
      -H "X-User-Role: $role" -H "Content-Type: $ct"
  fi
}

# Like api() but with NO Content-Type header
api_noct() {
  local method="$1" path="$2" role="$3" body="${4:-}"
  if [ -n "$body" ]; then
    curl -s -w "\n%{http_code}" -X "$method" "$path" \
      -H "X-User-Role: $role" -d "$body"
  else
    curl -s -w "\n%{http_code}" -X "$method" "$path" \
      -H "X-User-Role: $role"
  fi
}

get_status() { echo "$1" | tail -1; }
get_body()   { echo "$1" | sed '$d'; }

# Assert HTTP status code
assert_status() {
  local label="$1" expected="$2" actual="$3"
  if [ "$actual" = "$expected" ]; then
    pass "$label (HTTP $expected)"
  else
    fail "$label" "expected=$expected got=$actual"
  fi
}

# Assert that the response body is valid JSON with a "message" field
assert_json_error() {
  local label="$1" body="$2"
  local msg
  msg=$(echo "$body" | jq -r '.message // empty' 2>/dev/null)
  if [ -n "$msg" ]; then
    pass "$label — has JSON message: $msg"
  else
    # Check if body is HTML (error page leak)
    if echo "$body" | grep -qi '<html\|<!doctype'; then
      fail "$label" "got HTML error page instead of JSON"
    else
      fail "$label" "missing 'message' field in: $(echo "$body" | head -c 120)"
    fi
  fi
}

# Assert body is valid JSON (not HTML)
assert_json_body() {
  local label="$1" body="$2"
  if echo "$body" | jq . >/dev/null 2>&1; then
    pass "$label — response is valid JSON"
  else
    if echo "$body" | grep -qi '<html\|<!doctype'; then
      fail "$label" "got HTML error page instead of JSON"
    else
      fail "$label" "response is not valid JSON: $(echo "$body" | head -c 120)"
    fi
  fi
}

TIMESTAMP=$(date +%s)
PREFIX="err-test"

# Track IDs for cleanup
CLEANUP_ET_IDS=""
CLEANUP_CAT_NAMES=""
CLEANUP_CV_IDS=""

cleanup() {
  header "Cleanup"
  for name in $CLEANUP_CAT_NAMES; do
    curl -s -o /dev/null "$DATA_API/catalogs/$name" -X DELETE -H 'X-User-Role: SuperAdmin' 2>/dev/null || true
  done
  for cvid in $CLEANUP_CV_IDS; do
    curl -s -o /dev/null "$META_API/catalog-versions/$cvid" -X DELETE -H 'X-User-Role: Admin' 2>/dev/null || true
  done
  for etid in $CLEANUP_ET_IDS; do
    curl -s -o /dev/null "$META_API/entity-types/$etid" -X DELETE -H 'X-User-Role: Admin' 2>/dev/null || true
  done
  echo "  Cleaned up test data"
}
trap cleanup EXIT

# ============================================================================
# Setup: Create minimal test data
# ============================================================================
header "Setup: Create minimal test data"

# Create an entity type for tests that need a valid ID
ET_RESP=$(api POST "$META_API/entity-types" Admin "{\"name\":\"${PREFIX}-server-${TIMESTAMP}\"}")
ET_STATUS=$(get_status "$ET_RESP")
ET_BODY=$(get_body "$ET_RESP")
SETUP_ET_ID=$(echo "$ET_BODY" | jq -r '.entity_type.id')
SETUP_ETV_ID=$(echo "$ET_BODY" | jq -r '.version.id')
CLEANUP_ET_IDS="$SETUP_ET_ID"
echo "  Entity type: $SETUP_ET_ID (status=$ET_STATUS)"

# Create a CV for catalog tests
CV_RESP=$(api POST "$META_API/catalog-versions" Admin \
  "{\"version_label\":\"${PREFIX}-cv-${TIMESTAMP}\",\"pins\":[{\"entity_type_version_id\":\"$SETUP_ETV_ID\"}]}")
SETUP_CV_ID=$(get_body "$CV_RESP" | jq -r '.id')
CLEANUP_CV_IDS="$SETUP_CV_ID"
echo "  CV: $SETUP_CV_ID"

# Create a catalog for instance-level tests
SETUP_CAT="${PREFIX}-cat-${TIMESTAMP}"
api POST "$DATA_API/catalogs" Admin \
  "{\"name\":\"$SETUP_CAT\",\"description\":\"Error handling test\",\"catalog_version_id\":\"$SETUP_CV_ID\"}" > /dev/null
CLEANUP_CAT_NAMES="$SETUP_CAT"
echo "  Catalog: $SETUP_CAT"

# ============================================================================
# Category 1: Malformed JSON
# ============================================================================
header "Category 1: Malformed JSON (expect 400, not 500)"

RESP=$(api POST "$META_API/entity-types" Admin '{invalid json')
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
assert_status "POST /entity-types with malformed JSON" "400" "$STATUS"
assert_json_error "POST /entity-types malformed JSON response format" "$BODY"

RESP=$(api POST "$DATA_API/catalogs" Admin '{invalid json')
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
assert_status "POST /catalogs with malformed JSON" "400" "$STATUS"
assert_json_error "POST /catalogs malformed JSON response format" "$BODY"

RESP=$(api POST "$DATA_API/catalogs/import" Admin '{invalid json')
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
assert_status "POST /catalogs/import with malformed JSON" "400" "$STATUS"
assert_json_error "POST /catalogs/import malformed JSON response format" "$BODY"

RESP=$(api POST "$META_API/type-definitions" Admin '{invalid json')
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
assert_status "POST /type-definitions with malformed JSON" "400" "$STATUS"
assert_json_error "POST /type-definitions malformed JSON response format" "$BODY"

# ============================================================================
# Category 2: Wrong Content-Type
# ============================================================================
header "Category 2: Wrong Content-Type"

RESP=$(api_ct POST "$META_API/entity-types" Admin "text/plain" '{"name":"should-not-create"}')
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
# Echo framework may still parse it or reject it; either 400 or 415 is acceptable
if [ "$STATUS" = "400" ] || [ "$STATUS" = "415" ] || [ "$STATUS" = "201" ]; then
  pass "POST /entity-types with text/plain Content-Type — HTTP $STATUS (graceful)"
  # Clean up if it was accidentally created
  if [ "$STATUS" = "201" ]; then
    ACCIDENTAL_ID=$(echo "$BODY" | jq -r '.entity_type.id // empty')
    if [ -n "$ACCIDENTAL_ID" ]; then
      CLEANUP_ET_IDS="$CLEANUP_ET_IDS $ACCIDENTAL_ID"
    fi
  fi
else
  fail "POST /entity-types with text/plain Content-Type" "expected 400/415/201, got=$STATUS"
fi

RESP=$(api_ct POST "$DATA_API/catalogs" Admin "application/xml" '{"name":"should-not-create"}')
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
if [ "$STATUS" = "400" ] || [ "$STATUS" = "415" ] || [ "$STATUS" = "201" ]; then
  pass "POST /catalogs with application/xml Content-Type — HTTP $STATUS (graceful)"
  if [ "$STATUS" = "201" ]; then
    ACCIDENTAL_NAME=$(echo "$BODY" | jq -r '.name // empty')
    if [ -n "$ACCIDENTAL_NAME" ]; then
      CLEANUP_CAT_NAMES="$CLEANUP_CAT_NAMES $ACCIDENTAL_NAME"
    fi
  fi
else
  fail "POST /catalogs with application/xml Content-Type" "expected 400/415/201, got=$STATUS"
fi

# ============================================================================
# Category 3: Empty bodies
# ============================================================================
header "Category 3: Empty bodies (expect 400 with descriptive error)"

RESP=$(api POST "$META_API/entity-types" Admin "")
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
assert_status "POST /entity-types with empty body" "400" "$STATUS"
assert_json_error "POST /entity-types empty body response format" "$BODY"

RESP=$(api POST "$DATA_API/catalogs" Admin "")
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
assert_status "POST /catalogs with empty body" "400" "$STATUS"
assert_json_error "POST /catalogs empty body response format" "$BODY"

# PUT /entity-types/:id with empty body — may be OK (no changes) or 400
RESP=$(api PUT "$META_API/entity-types/$SETUP_ET_ID" Admin "")
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
if [ "$STATUS" = "200" ] || [ "$STATUS" = "400" ]; then
  pass "PUT /entity-types/:id with empty body — HTTP $STATUS (acceptable)"
else
  fail "PUT /entity-types/:id with empty body" "expected 200 or 400, got=$STATUS"
fi

RESP=$(api POST "$META_API/type-definitions" Admin "")
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
assert_status "POST /type-definitions with empty body" "400" "$STATUS"
assert_json_error "POST /type-definitions empty body response format" "$BODY"

# ============================================================================
# Category 4: Boundary-length strings
# ============================================================================
header "Category 4: Boundary-length strings"

# 4a. Entity type with empty name
RESP=$(api POST "$META_API/entity-types" Admin '{"name":""}')
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
assert_status "Entity type with empty name" "400" "$STATUS"
assert_json_error "Entity type empty name response format" "$BODY"

# 4b. Entity type with 256-char name
LONG_NAME=$(printf 'a%.0s' $(seq 1 256))
RESP=$(api POST "$META_API/entity-types" Admin "{\"name\":\"$LONG_NAME\"}")
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
if [ "$STATUS" = "400" ] || [ "$STATUS" = "201" ]; then
  pass "Entity type with 256-char name — HTTP $STATUS (handled)"
  if [ "$STATUS" = "201" ]; then
    LONG_ET_ID=$(echo "$BODY" | jq -r '.entity_type.id // empty')
    if [ -n "$LONG_ET_ID" ]; then
      CLEANUP_ET_IDS="$CLEANUP_ET_IDS $LONG_ET_ID"
    fi
  fi
else
  fail "Entity type with 256-char name" "expected 400 or 201, got=$STATUS"
fi

# 4c. Catalog name exceeding 63 chars (DNS label limit)
LONG_CAT_NAME=$(printf 'a%.0s' $(seq 1 64))
RESP=$(api POST "$DATA_API/catalogs" Admin \
  "{\"name\":\"$LONG_CAT_NAME\",\"description\":\"too long\",\"catalog_version_id\":\"$SETUP_CV_ID\"}")
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
assert_status "Catalog name exceeding 63 chars" "400" "$STATUS"
assert_json_error "Catalog name 64-char response format" "$BODY"

# 4d. Catalog name of exactly 63 chars (should succeed)
EXACT_CAT_NAME=$(printf 'a%.0s' $(seq 1 63))
RESP=$(api POST "$DATA_API/catalogs" Admin \
  "{\"name\":\"$EXACT_CAT_NAME\",\"description\":\"exact limit\",\"catalog_version_id\":\"$SETUP_CV_ID\"}")
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
assert_status "Catalog name of exactly 63 chars" "201" "$STATUS"
CLEANUP_CAT_NAMES="$CLEANUP_CAT_NAMES $EXACT_CAT_NAME"

# 4e. Catalog name with uppercase
RESP=$(api POST "$DATA_API/catalogs" Admin \
  "{\"name\":\"UpperCase\",\"description\":\"invalid\",\"catalog_version_id\":\"$SETUP_CV_ID\"}")
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
assert_status "Catalog name with uppercase" "400" "$STATUS"
assert_json_error "Catalog uppercase name response format" "$BODY"

# 4f. Catalog name with spaces
RESP=$(api POST "$DATA_API/catalogs" Admin \
  "{\"name\":\"has spaces\",\"description\":\"invalid\",\"catalog_version_id\":\"$SETUP_CV_ID\"}")
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
assert_status "Catalog name with spaces" "400" "$STATUS"
assert_json_error "Catalog name with spaces response format" "$BODY"

# 4g. Catalog name with special chars
RESP=$(api POST "$DATA_API/catalogs" Admin \
  "{\"name\":\"bad!@#name\",\"description\":\"invalid\",\"catalog_version_id\":\"$SETUP_CV_ID\"}")
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
assert_status "Catalog name with special chars" "400" "$STATUS"
assert_json_error "Catalog special chars name response format" "$BODY"

# 4h. Catalog name starting with hyphen
RESP=$(api POST "$DATA_API/catalogs" Admin \
  "{\"name\":\"-leading-hyphen\",\"description\":\"invalid\",\"catalog_version_id\":\"$SETUP_CV_ID\"}")
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
assert_status "Catalog name starting with hyphen" "400" "$STATUS"
assert_json_error "Catalog leading-hyphen name response format" "$BODY"

# ============================================================================
# Category 5: Error response format consistency
# ============================================================================
header "Category 5: Error response format consistency"

# 5a. 400 responses should have JSON body with "message" field
RESP=$(api POST "$META_API/entity-types" Admin '{"name":""}')
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
if [ "$STATUS" = "400" ]; then
  assert_json_error "400 response has JSON 'message' field" "$BODY"
fi

# 5b. 404 responses should have JSON body
RESP=$(api GET "$META_API/entity-types/00000000-0000-0000-0000-000000000000" Admin)
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
assert_status "GET nonexistent entity type" "404" "$STATUS"
assert_json_body "404 entity type response is JSON" "$BODY"

RESP=$(api GET "$DATA_API/catalogs/nonexistent-catalog-xyz" Admin)
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
assert_status "GET nonexistent catalog" "404" "$STATUS"
assert_json_body "404 catalog response is JSON" "$BODY"

# 5c. 403 responses should have JSON body
RESP=$(api POST "$META_API/entity-types" RO '{"name":"should-fail-rbac"}')
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
assert_status "POST entity type as RO (forbidden)" "403" "$STATUS"
assert_json_body "403 response is JSON" "$BODY"

# 5d. 409 responses should have JSON body (try creating duplicate entity type)
RESP=$(api POST "$META_API/entity-types" Admin "{\"name\":\"${PREFIX}-server-${TIMESTAMP}\"}")
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
assert_status "Create duplicate entity type" "409" "$STATUS"
assert_json_body "409 response is JSON" "$BODY"

# 5e. No HTML error pages for bad routes
RESP=$(api GET "$META_API/nonexistent-endpoint" Admin)
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
if echo "$BODY" | grep -qi '<html\|<!doctype'; then
  fail "No HTML for unknown meta endpoint" "got HTML response"
else
  pass "No HTML for unknown meta endpoint (HTTP $STATUS)"
fi

RESP=$(api GET "$DATA_API/nonexistent-endpoint" Admin)
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
if echo "$BODY" | grep -qi '<html\|<!doctype'; then
  fail "No HTML for unknown data endpoint" "got HTML response"
else
  pass "No HTML for unknown data endpoint (HTTP $STATUS)"
fi

# ============================================================================
# Category 6: Numeric boundary values
# ============================================================================
header "Category 6: Numeric boundary values (pagination)"

ET_NAME="${PREFIX}-server-${TIMESTAMP}"

# 6a. limit=0
RESP=$(api GET "$DATA_API/catalogs/$SETUP_CAT/$ET_NAME?limit=0" Admin)
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
if [ "$STATUS" = "200" ] || [ "$STATUS" = "400" ]; then
  pass "List instances with limit=0 — HTTP $STATUS (handled gracefully)"
else
  fail "List instances with limit=0" "expected 200 or 400, got=$STATUS"
fi

# 6b. limit=-1
RESP=$(api GET "$DATA_API/catalogs/$SETUP_CAT/$ET_NAME?limit=-1" Admin)
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
if [ "$STATUS" = "200" ] || [ "$STATUS" = "400" ]; then
  pass "List instances with limit=-1 — HTTP $STATUS (handled gracefully)"
else
  fail "List instances with limit=-1" "expected 200 or 400, got=$STATUS"
fi

# 6c. limit=101 — should be capped at 100
RESP=$(api GET "$DATA_API/catalogs/$SETUP_CAT/$ET_NAME?limit=101" Admin)
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
if [ "$STATUS" = "200" ]; then
  pass "List instances with limit=101 — HTTP 200 (capped at 100)"
else
  fail "List instances with limit=101" "expected 200, got=$STATUS"
fi

# 6d. offset=-1
RESP=$(api GET "$DATA_API/catalogs/$SETUP_CAT/$ET_NAME?offset=-1" Admin)
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
if [ "$STATUS" = "200" ] || [ "$STATUS" = "400" ]; then
  pass "List instances with offset=-1 — HTTP $STATUS (handled gracefully)"
else
  fail "List instances with offset=-1" "expected 200 or 400, got=$STATUS"
fi

# 6e. limit=abc (non-numeric)
RESP=$(api GET "$DATA_API/catalogs/$SETUP_CAT/$ET_NAME?limit=abc" Admin)
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
if [ "$STATUS" = "200" ] || [ "$STATUS" = "400" ]; then
  pass "List instances with limit=abc — HTTP $STATUS (handled gracefully)"
else
  fail "List instances with limit=abc" "expected 200 or 400, got=$STATUS"
fi

# 6f. Extremely large limit
RESP=$(api GET "$DATA_API/catalogs/$SETUP_CAT/$ET_NAME?limit=999999999" Admin)
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
if [ "$STATUS" = "200" ]; then
  pass "List instances with limit=999999999 — HTTP 200 (capped)"
else
  fail "List instances with limit=999999999" "expected 200, got=$STATUS"
fi

# ============================================================================
# Category 7: Special characters in names
# ============================================================================
header "Category 7: Special characters in names"

# 7a. SQL injection attempt in entity type name
RESP=$(api POST "$META_API/entity-types" Admin '{"name":"'\''; DROP TABLE --"}')
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
if [ "$STATUS" = "400" ] || [ "$STATUS" = "201" ]; then
  pass "SQL injection in entity type name — HTTP $STATUS (no crash)"
  if [ "$STATUS" = "201" ]; then
    SQLI_ID=$(echo "$BODY" | jq -r '.entity_type.id // empty')
    if [ -n "$SQLI_ID" ]; then
      CLEANUP_ET_IDS="$CLEANUP_ET_IDS $SQLI_ID"
    fi
  fi
else
  fail "SQL injection in entity type name" "expected 400 or 201, got=$STATUS"
fi

# 7b. XSS attempt in entity type name
RESP=$(api POST "$META_API/entity-types" Admin '{"name":"<script>alert(1)</script>"}')
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
if [ "$STATUS" = "400" ] || [ "$STATUS" = "201" ]; then
  pass "XSS in entity type name — HTTP $STATUS (no crash)"
  if [ "$STATUS" = "201" ]; then
    XSS_ID=$(echo "$BODY" | jq -r '.entity_type.id // empty')
    if [ -n "$XSS_ID" ]; then
      CLEANUP_ET_IDS="$CLEANUP_ET_IDS $XSS_ID"
    fi
  fi
else
  fail "XSS in entity type name" "expected 400 or 201, got=$STATUS"
fi

# 7c. Null byte in entity type name
RESP=$(curl -s -w "\n%{http_code}" -X POST "$META_API/entity-types" \
  -H "X-User-Role: Admin" -H "Content-Type: application/json" \
  -d '{"name":"null byte"}')
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
if [ "$STATUS" = "400" ] || [ "$STATUS" = "201" ]; then
  pass "Null byte in entity type name — HTTP $STATUS (no crash)"
  if [ "$STATUS" = "201" ]; then
    NULL_ID=$(echo "$BODY" | jq -r '.entity_type.id // empty')
    if [ -n "$NULL_ID" ]; then
      CLEANUP_ET_IDS="$CLEANUP_ET_IDS $NULL_ID"
    fi
  fi
else
  fail "Null byte in entity type name" "expected 400 or 201, got=$STATUS"
fi

# 7d. Unicode emoji in entity type name
RESP=$(api POST "$META_API/entity-types" Admin '{"name":"test-😀-emoji"}')
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
if [ "$STATUS" = "400" ] || [ "$STATUS" = "201" ]; then
  pass "Unicode emoji in entity type name — HTTP $STATUS (no crash)"
  if [ "$STATUS" = "201" ]; then
    EMOJI_ID=$(echo "$BODY" | jq -r '.entity_type.id // empty')
    if [ -n "$EMOJI_ID" ]; then
      CLEANUP_ET_IDS="$CLEANUP_ET_IDS $EMOJI_ID"
    fi
  fi
else
  fail "Unicode emoji in entity type name" "expected 400 or 201, got=$STATUS"
fi

# 7e. SQL injection in catalog name (DNS label validation should reject)
RESP=$(api POST "$DATA_API/catalogs" Admin \
  "{\"name\":\"'; DROP TABLE --\",\"description\":\"sqli\",\"catalog_version_id\":\"$SETUP_CV_ID\"}")
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
assert_status "SQL injection in catalog name (DNS label rejects)" "400" "$STATUS"

# 7f. Very long JSON value (not name, but description)
LONG_DESC=$(printf 'x%.0s' $(seq 1 10000))
RESP=$(api POST "$META_API/entity-types" Admin "{\"name\":\"${PREFIX}-longdesc-${TIMESTAMP}\",\"description\":\"$LONG_DESC\"}")
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
if [ "$STATUS" = "201" ] || [ "$STATUS" = "400" ]; then
  pass "Entity type with 10K-char description — HTTP $STATUS (handled)"
  if [ "$STATUS" = "201" ]; then
    LONGDESC_ID=$(echo "$BODY" | jq -r '.entity_type.id // empty')
    if [ -n "$LONGDESC_ID" ]; then
      CLEANUP_ET_IDS="$CLEANUP_ET_IDS $LONGDESC_ID"
    fi
  fi
else
  fail "Entity type with 10K-char description" "expected 201 or 400, got=$STATUS"
fi

# ============================================================================
# Category 8: Additional edge cases
# ============================================================================
header "Category 8: Additional edge cases"

# 8a. GET with request body (should be ignored or harmless)
RESP=$(api GET "$META_API/entity-types" Admin '{"unexpected":"body"}')
STATUS=$(get_status "$RESP")
if [ "$STATUS" = "200" ]; then
  pass "GET /entity-types with body — HTTP 200 (body ignored)"
else
  fail "GET /entity-types with body" "expected 200, got=$STATUS"
fi

# 8b. DELETE nonexistent entity type
RESP=$(api DELETE "$META_API/entity-types/00000000-0000-0000-0000-000000000000" Admin)
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
assert_status "DELETE nonexistent entity type" "404" "$STATUS"
assert_json_body "DELETE 404 response is JSON" "$BODY"

# 8c. DELETE nonexistent catalog
RESP=$(api DELETE "$DATA_API/catalogs/does-not-exist-xyz" Admin)
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
assert_status "DELETE nonexistent catalog" "404" "$STATUS"
assert_json_body "DELETE catalog 404 response is JSON" "$BODY"

# 8d. PUT with invalid UUID path param
RESP=$(api PUT "$META_API/entity-types/not-a-uuid" Admin '{"description":"test"}')
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
if [ "$STATUS" = "400" ] || [ "$STATUS" = "404" ]; then
  pass "PUT with invalid UUID — HTTP $STATUS (handled)"
else
  fail "PUT with invalid UUID" "expected 400 or 404, got=$STATUS"
fi

# 8e. POST to a GET-only endpoint
RESP=$(api POST "$META_API/entity-types/$SETUP_ET_ID" Admin '{}')
STATUS=$(get_status "$RESP")
if [ "$STATUS" = "405" ] || [ "$STATUS" = "404" ]; then
  pass "POST to GET-only entity type endpoint — HTTP $STATUS"
else
  fail "POST to GET-only endpoint" "expected 405 or 404, got=$STATUS"
fi

# 8f. Create catalog with missing catalog_version_id
RESP=$(api POST "$DATA_API/catalogs" Admin '{"name":"missing-cv","description":"no cv"}')
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
assert_status "Create catalog without catalog_version_id" "400" "$STATUS"
assert_json_error "Missing CV ID response format" "$BODY"

# 8g. Create catalog with nonexistent catalog_version_id
RESP=$(api POST "$DATA_API/catalogs" Admin \
  '{"name":"bad-cv-ref","description":"bad cv","catalog_version_id":"00000000-0000-0000-0000-000000000000"}')
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
if [ "$STATUS" = "400" ] || [ "$STATUS" = "404" ]; then
  pass "Create catalog with nonexistent CV ID — HTTP $STATUS"
else
  fail "Create catalog with nonexistent CV ID" "expected 400 or 404, got=$STATUS"
fi

# 8h. Create entity type with extra unexpected fields (should be ignored or rejected)
RESP=$(api POST "$META_API/entity-types" Admin \
  "{\"name\":\"${PREFIX}-extra-${TIMESTAMP}\",\"extra_field\":\"value\",\"another\":123}")
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
if [ "$STATUS" = "201" ] || [ "$STATUS" = "400" ]; then
  pass "Entity type with extra fields — HTTP $STATUS (handled)"
  if [ "$STATUS" = "201" ]; then
    EXTRA_ID=$(echo "$BODY" | jq -r '.entity_type.id // empty')
    if [ -n "$EXTRA_ID" ]; then
      CLEANUP_ET_IDS="$CLEANUP_ET_IDS $EXTRA_ID"
    fi
  fi
else
  fail "Entity type with extra fields" "expected 201 or 400, got=$STATUS"
fi

# 8i. Double-create same catalog (expect 409 conflict)
RESP=$(api POST "$DATA_API/catalogs" Admin \
  "{\"name\":\"$SETUP_CAT\",\"description\":\"duplicate\",\"catalog_version_id\":\"$SETUP_CV_ID\"}")
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
assert_status "Create duplicate catalog" "409" "$STATUS"
assert_json_body "409 duplicate catalog response is JSON" "$BODY"

# 8j. Create instance on nonexistent catalog
RESP=$(api POST "$DATA_API/catalogs/nonexistent-cat/some-type" Admin \
  '{"name":"test","description":"test"}')
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
assert_status "Create instance on nonexistent catalog" "404" "$STATUS"
assert_json_body "404 instance-on-missing-catalog is JSON" "$BODY"

# 8k. Create instance for nonexistent entity type on valid catalog
RESP=$(api POST "$DATA_API/catalogs/$SETUP_CAT/nonexistent-entity-type" Admin \
  '{"name":"test","description":"test"}')
STATUS=$(get_status "$RESP")
BODY=$(get_body "$RESP")
if [ "$STATUS" = "400" ] || [ "$STATUS" = "404" ]; then
  pass "Create instance for unknown entity type — HTTP $STATUS"
  assert_json_body "Unknown entity type response is JSON" "$BODY"
else
  fail "Create instance for unknown entity type" "expected 400 or 404, got=$STATUS"
fi

# ============================================================================
# Results
# ============================================================================
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
