#!/usr/bin/env bash
# Clean leftover TEST data from the system.
# Only deletes entities matching known test naming patterns.
# User-created entities are NOT touched.
#
# Usage:
#   ./scripts/clean-test-data.sh                          # defaults to localhost:30080
#   ./scripts/clean-test-data.sh http://localhost:30080    # explicit local
#   ./scripts/clean-test-data.sh https://api.ocp.example.com  # remote OCP
#
# Test naming patterns matched:
#   All entities:     tst---*  (standard test prefix)
#
# Prerequisites: curl, jq

set -euo pipefail

API_BASE="${1:-http://localhost:30080}"
META_URL="${API_BASE}/api/meta/v1"
DATA_URL="${API_BASE}/api/data/v1"
ROLE="SuperAdmin"

is_test_name() {
  local name="$1"
  case "$name" in
    tst---*)
      return 0 ;;
    *)
      return 1 ;;
  esac
}

DELETED=0
SKIPPED=0

echo "Cleaning test data from ${API_BASE}"
echo "(Only entities matching test naming patterns)"
echo ""

# 1. Catalogs
echo "=== Catalogs ==="
CATALOGS=$(curl -s "${DATA_URL}/catalogs" -H "X-User-Role: ${ROLE}" | jq -r '.items[]?.name // empty')
if [ -z "$CATALOGS" ]; then
  echo "  (none)"
else
  for name in $CATALOGS; do
    if is_test_name "$name"; then
      printf "  DELETE %-40s " "$name"
      HTTP=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "${DATA_URL}/catalogs/${name}" -H "X-User-Role: ${ROLE}")
      echo "$HTTP"
      DELETED=$((DELETED + 1))
    else
      echo "  SKIP   $name"
      SKIPPED=$((SKIPPED + 1))
    fi
  done
fi

# 2. Catalog versions
echo ""
echo "=== Catalog Versions ==="
CVS_JSON=$(curl -s "${META_URL}/catalog-versions" -H "X-User-Role: ${ROLE}")
CV_COUNT=$(echo "$CVS_JSON" | jq '.total // 0')
if [ "$CV_COUNT" = "0" ]; then
  echo "  (none)"
else
  echo "$CVS_JSON" | jq -r '.items[] | "\(.id) \(.version_label)"' | while read -r id label; do
    if is_test_name "$label"; then
      printf "  DELETE %-40s " "${label} (${id:0:8}...)"
      HTTP=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "${META_URL}/catalog-versions/${id}" -H "X-User-Role: ${ROLE}")
      echo "$HTTP"
      DELETED=$((DELETED + 1))
    else
      echo "  SKIP   ${label} (${id:0:8}...)"
      SKIPPED=$((SKIPPED + 1))
    fi
  done
fi

# 3. Entity types
echo ""
echo "=== Entity Types ==="
ETS_JSON=$(curl -s "${META_URL}/entity-types" -H "X-User-Role: ${ROLE}")
ET_COUNT=$(echo "$ETS_JSON" | jq '.total // 0')
if [ "$ET_COUNT" = "0" ]; then
  echo "  (none)"
else
  echo "$ETS_JSON" | jq -r '.items[] | "\(.id) \(.name)"' | while read -r id name; do
    if is_test_name "$name"; then
      printf "  DELETE %-40s " "${name} (${id:0:8}...)"
      HTTP=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "${META_URL}/entity-types/${id}" -H "X-User-Role: ${ROLE}")
      echo "$HTTP"
    else
      echo "  SKIP   ${name} (${id:0:8}...)"
    fi
  done
fi

echo ""
echo "=== Remaining ==="
echo "  Entity types: $(curl -s "${META_URL}/entity-types" -H "X-User-Role: ${ROLE}" | jq '.total')"
echo "  Catalog versions: $(curl -s "${META_URL}/catalog-versions" -H "X-User-Role: ${ROLE}" | jq '.total')"
echo "  Catalogs: $(curl -s "${DATA_URL}/catalogs" -H "X-User-Role: ${ROLE}" | jq '.total')"
echo "  Enums: $(curl -s "${META_URL}/enums" -H "X-User-Role: ${ROLE}" | jq '.total')"
echo ""
echo "Done."
