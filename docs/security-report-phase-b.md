# Security Report — Phase B

## Date: 2026-02-16

## 1. Go Standard Library Vulnerabilities (govulncheck)

**No vulnerabilities found.** Running Go 1.25.7.

Previous scan on Go 1.25.4 found 5 vulnerabilities (GO-2026-4341, GO-2026-4340, GO-2026-4337, GO-2025-4175, GO-2025-4155) in net/url, crypto/tls, and crypto/x509. All resolved by upgrading to Go 1.25.7.

## 2. Third-Party Dependencies

No vulnerabilities found in imported third-party packages.

## 3. Docker Image Security

- **Base images**: `gcr.io/distroless/static-debian12:nonroot` for Go binaries (minimal attack surface, non-root user)
- **UI base**: `nginx:alpine` (minimal footprint)
- **Build practice**: Multi-stage builds prevent build toolchain leaks
- **CGO_ENABLED=0**: Static binaries with no C library dependency

## 4. Kubernetes Manifest Security

- **Operator RBAC**: Scoped Role (not ClusterRole) — limited to `assethub` namespace only
- **Least privilege**: Operator only has permissions for AssetHub CRDs, Deployments, Services, and Events
- **ImagePullPolicy**: Set to `Never` for kind cluster (local images only)
- **PostgreSQL credentials**: Stored in Kubernetes Secret (base64 encoded, not plaintext in manifests)

## 5. CORS Configuration

- CORS middleware is properly configured with configurable allowed origins
- X-User-Role header is explicitly allowed in CORS config
- No wildcard origins — must be explicitly listed
- No-op middleware when no origins configured (secure default)

## 6. API Security

- **RBAC enforcement**: All meta and operational API routes require X-User-Role header
- **Role hierarchy**: RO < RW < Admin < SuperAdmin, enforced at middleware level
- **Input validation**: Request binding via Echo framework with JSON content type enforcement
- **Error handling**: Domain errors mapped to appropriate HTTP status codes; internal errors do not leak details to clients
- **No SQL injection risk**: GORM parameterized queries throughout; ORDER BY columns validated against allowlist

## 7. PostgreSQL DSN Handling

- **docker-compose**: DSN passed via environment variables (not embedded in image)
- **Kubernetes**: DSN stored in ConfigMap (should be migrated to Secret for production)
- **Recommendation**: For production deployment, move DB_CONNECTION_STRING to a Kubernetes Secret

## 8. Areas for Future Improvement

1. **Move DB DSN to Secret** in K8s manifests for production deployments
2. **Add network policies** to restrict pod-to-pod communication in K8s
3. **TLS termination** for API server (currently HTTP only, suitable for development/kind)
4. **Rate limiting** on API endpoints for production deployment
5. **Audit logging** for RBAC-protected operations
