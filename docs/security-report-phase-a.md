# Security Report — Phase A

## Scan Date

Phase A implementation security audit.

## Vulnerabilities Found and Fixed

### Fixed: SQL Injection via SortBy Parameter (HIGH)

**Files**: `internal/infrastructure/gorm/repository/entity_type_repo.go`, `entity_instance_repo.go`

**Issue**: The `SortBy` field from `ListParams` was passed directly into GORM's `.Order()` method as a raw string, which does not parameterize ORDER BY inputs. An attacker could inject arbitrary SQL via the sort parameter.

**Fix**: Added an allowlist of valid sort columns (`name`, `created_at`, `updated_at`, `version`) in `helpers.go`. The `validateSortBy()` function rejects any column not in the allowlist before it reaches the GORM query. Applied to both entity type and entity instance listing.

### Fixed: Missing Role-Based Route Protection (HIGH)

**Files**: `internal/api/meta/entity_type_handler.go`, `internal/api/meta/router.go`

**Issue**: Write endpoints (POST, PUT, DELETE) on meta API had no per-route RBAC enforcement. All roles including RO could perform write operations.

**Fix**: Applied `RequireRole(RoleAdmin)` middleware to all write routes (POST, PUT, DELETE). GET routes remain accessible to all authenticated users. The `RegisterEntityTypeRoutes` function now takes a `requireAdmin` middleware parameter.

### Fixed: Internal Error Details Leaked to Clients (MEDIUM)

**Files**: `internal/api/meta/errors.go`, `internal/api/operational/handler.go`

**Issue**: Unhandled errors returned raw Go error messages to clients, potentially leaking database schema, connection strings, and internal paths.

**Fix**: Replaced catch-all error response with generic "internal server error" message. Actual error details should be logged server-side (logging infrastructure to be added in Phase B).

## Remaining Vulnerabilities with Mitigations

### Header-Based Authentication (CRITICAL — Accepted for Phase A)

**File**: `internal/api/middleware/rbac.go`

**Issue**: The `HeaderRBACProvider` reads the user role from the `X-User-Role` HTTP header, which is trivially spoofable.

**Mitigation**: This is the development/testing RBAC provider. The architecture defines an `RBACProvider` interface specifically so that the real OpenShift SubjectAccessReview implementation can be plugged in during Phase C (OCP deployment). The header-based provider must never be used in production.

**Risk**: Accepted for Phase A (isolated development, no network exposure).

### No CORS Configuration (LOW)

**Mitigation**: Not needed during Phase A (no cross-origin requests in development). CORS middleware will be configured in Phase B when the API server and UI are deployed as separate containers.

### UI API Client Missing Auth Headers (LOW)

**File**: `ui/src/api/client.ts`

**Mitigation**: The UI is not yet connected to a running API server. Auth header injection will be implemented when the full UI-to-API integration is wired in Phase B.

## Categories with No Findings

- Command injection: No use of `os/exec` in the codebase
- Hardcoded secrets: None found. `.gitignore` excludes `.env` files
- XSS: React JSX auto-escaping prevents XSS. No `dangerouslySetInnerHTML` used
- Insecure deserialization: Standard `encoding/json` used throughout
