# Security Report — Phase D: CatalogVersion Discovery CRD + Cluster Role Filtering

## Scan Date: 2026-02-18

## Scope

Files added or modified in this phase:
- `internal/service/meta/cr_manager.go` (SanitizeK8sName)
- `internal/service/meta/catalog_version_service.go` (CR management on promote/demote)
- `internal/infrastructure/k8s/cr_manager.go` (K8s CRManager implementation)
- `internal/operator/api/v1alpha1/catalogversion_types.go` (CRD types)
- `internal/operator/controllers/controller.go` (CatalogVersion reconciliation)
- `internal/operator/controllers/reconciler.go` (CLUSTER_ROLE, CatalogVersion status)
- `internal/infrastructure/config/config.go` (ClusterRole, AllowedStages)
- `cmd/api-server/main.go` (K8s client wiring)
- `deploy/k8s/` manifests

## Issues Found and Resolved

### RESOLVED: SanitizeK8sName empty string vulnerability (HIGH)

**Problem:** `SanitizeK8sName` could return an empty string for inputs containing only special characters (e.g., `"!!!###$$$"`), leading to invalid K8s resource names.

**Fix:**
- Added validation in `Promote()` and `Demote()` to reject empty sanitized names with a clear error message.
- Added length truncation (253 char limit) to prevent exceeding K8s DNS subdomain name limits.
- Added tests for empty input and long input edge cases.

## Pre-existing Issues (Not Part of This Phase)

| Severity | Issue | Notes |
|----------|-------|-------|
| HIGH | Header-based RBAC spoofing in development mode | Pre-existing. Development mode uses `X-User-Role` header. Production should use token-based auth. |
| MEDIUM | DB credentials in ConfigMap | Pre-existing. Development setup uses ConfigMap for DB connection string. |
| LOW | No rate limiting | Pre-existing. No rate limiting middleware configured. |
| LOW | Weak default credentials | Pre-existing. Default postgres password is `assethub`. |

## No New Vulnerabilities Introduced

The following aspects of the new code were verified:
- **K8s annotations**: `PromotedBy` comes from the validated role string (not arbitrary user input). `PromotedAt` is generated server-side via `time.Now().Format(time.RFC3339)`.
- **RBAC permissions**: Operator role and API server RBAC follow least-privilege for CatalogVersion resources within the namespace.
- **Nil safety**: `crManager` nil checks prevent panics when running outside K8s.
- **Input validation**: Version labels are validated as required at the API layer. SanitizeK8sName handles edge cases.
- **Error handling**: Errors are returned as domain errors (validation/forbidden), not raw K8s errors.
