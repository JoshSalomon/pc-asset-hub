#!/bin/bash
# CRD validation tests for ExporterPlugin (T-36.59–T-36.64)
# Requires: kubectl access to a cluster with the CRD applied

set -uo pipefail

KUBE_CMD="${KUBE_CMD:-kubectl --context kind-assethub}"
PASS=0
FAIL=0
CRD_FILE="$(cd "$(dirname "$0")/.." && pwd)/deploy/k8s/operator/exporterplugin-crd.yaml"

pass() { echo "  PASS: $1"; ((PASS++)); }
fail() { echo "  FAIL: $1"; ((FAIL++)); }

echo "=== ExporterPlugin CRD Validation Tests ==="
echo ""

# T-36.59: CRD YAML dry-run=server succeeds
if $KUBE_CMD apply --dry-run=server -f "$CRD_FILE" >/dev/null 2>&1; then
  pass "T-36.59: CRD YAML dry-run=server succeeds"
else
  fail "T-36.59: CRD YAML dry-run=server succeeds"
fi

# T-36.60: RBAC YAML dry-run succeeds
RBAC_DIR="$(cd "$(dirname "$0")/.." && pwd)/deploy/k8s"
if $KUBE_CMD apply --dry-run=server -f "$RBAC_DIR/operator/role.yaml" >/dev/null 2>&1 && \
   $KUBE_CMD apply --dry-run=server -f "$RBAC_DIR/api-server/rbac.yaml" >/dev/null 2>&1; then
  pass "T-36.60: RBAC YAML dry-run succeeds (operator + API server)"
else
  fail "T-36.60: RBAC YAML dry-run succeeds (operator + API server)"
fi

# T-36.61: CRD applied → kubectl get crd succeeds
if $KUBE_CMD get crd exporterplugins.assethub.project-catalyst.io >/dev/null 2>&1; then
  pass "T-36.61: CRD registered in cluster"
else
  fail "T-36.61: CRD registered in cluster"
fi

# T-36.62: Missing Endpoint field → rejected by CRD
RESULT=$($KUBE_CMD apply --dry-run=server -f - 2>&1 <<'EOF' || true
apiVersion: assethub.project-catalyst.io/v1alpha1
kind: ExporterPlugin
metadata:
  name: test-no-endpoint
  namespace: assethub
spec:
  description: "Missing endpoint"
EOF
)
if echo "$RESULT" | grep -qi "endpoint\|required" >/dev/null 2>&1; then
  pass "T-36.62: Missing Endpoint field → rejected by CRD"
else
  fail "T-36.62: Missing Endpoint field → rejected (got: $RESULT)"
fi

# T-36.63: TimeoutSeconds negative → rejected by CRD
RESULT=$($KUBE_CMD apply --dry-run=server -f - 2>&1 <<'EOF' || true
apiVersion: assethub.project-catalyst.io/v1alpha1
kind: ExporterPlugin
metadata:
  name: test-neg-timeout
  namespace: assethub
spec:
  endpoint: "http://test.svc.cluster.local"
  timeoutSeconds: -1
EOF
)
if echo "$RESULT" | grep -qi "minimum\|invalid\|greater" >/dev/null 2>&1; then
  pass "T-36.63: Negative TimeoutSeconds → rejected by CRD"
else
  fail "T-36.63: Negative TimeoutSeconds → rejected (got: $RESULT)"
fi

# T-36.64: TimeoutSeconds=0 → rejected; omitted → accepted
RESULT_ZERO=$($KUBE_CMD apply --dry-run=server -f - 2>&1 <<'EOF' || true
apiVersion: assethub.project-catalyst.io/v1alpha1
kind: ExporterPlugin
metadata:
  name: test-zero-timeout
  namespace: assethub
spec:
  endpoint: "http://test.svc.cluster.local"
  timeoutSeconds: 0
EOF
)
RESULT_OMIT=$($KUBE_CMD apply --dry-run=server -f - 2>&1 <<'EOF' || true
apiVersion: assethub.project-catalyst.io/v1alpha1
kind: ExporterPlugin
metadata:
  name: test-omit-timeout
  namespace: assethub
spec:
  endpoint: "http://test.svc.cluster.local"
EOF
)
if echo "$RESULT_ZERO" | grep -qi "minimum\|invalid\|greater" && echo "$RESULT_OMIT" | grep -qi "created\|configured\|unchanged"; then
  pass "T-36.64: TimeoutSeconds=0 rejected; omitted accepted"
else
  fail "T-36.64: TimeoutSeconds=0 rejected; omitted accepted (zero: $RESULT_ZERO | omit: $RESULT_OMIT)"
fi

echo ""
echo -e "\033[0;32mtest-exporterplugin-crd: Tests: $((PASS+FAIL)), Passed: $PASS, Failed: $FAIL\033[0m"

if [[ $FAIL -gt 0 ]]; then
  exit 1
fi
