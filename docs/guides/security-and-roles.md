# Security and Access Control

This document describes the authentication model, role hierarchy, and authorization enforcement in the AI Asset Hub.

## Authentication Model

The API server supports two RBAC modes, selected via the `RBAC_MODE` environment variable.

**Development mode (`RBAC_MODE=header`)**
The user's role is read from the `X-User-Role` HTTP header. Any caller can claim any role. This mode is insecure and must never be used in production. The server prints a warning at startup when header-based RBAC is active.

**Production mode (`RBAC_MODE=sar` -- Phase C, not yet implemented)**
The server will authenticate callers via their ServiceAccount or Bearer token and perform a Kubernetes SubjectAccessReview (SAR) to map the authenticated identity to an application role. The `SARRBACProvider` and `SARCatalogAccessChecker` interfaces are defined but not yet wired.

## Role Hierarchy

Four roles are defined in order of increasing privilege. Each role inherits the permissions of all lower roles.

| Level | Role | Description |
|-------|------|-------------|
| 0 | **RO** (Read-Only) | Can view all schema and data. Cannot create, modify, or delete anything. |
| 1 | **RW** (Read-Write) | Can create and modify catalogs and instances. Can create catalog versions and promote dev to testing (or demote back). Cannot modify schema objects (entity types, attributes, associations, type definitions). |
| 2 | **Admin** | Full schema management (CRUD entity types, attributes, associations, type definitions). Can promote catalog versions through all lifecycle stages (testing to production). Cannot modify published catalogs or perform SuperAdmin actions. |
| 3 | **SuperAdmin** | All Admin permissions plus: modify published catalogs, demote from production, delete production catalog versions, bypass write protection on published catalogs. |

## Role Permissions Matrix

### Meta API (`/api/meta/v1/`)

| Operation | RO | RW | Admin | SuperAdmin |
|-----------|:--:|:--:|:-----:|:----------:|
| List / Get entity types, attributes, associations, type definitions, versions | Y | Y | Y | Y |
| Create / Update / Delete entity types | - | - | Y | Y |
| Add / Edit / Remove / Reorder attributes | - | - | Y | Y |
| Create / Edit / Delete associations | - | - | Y | Y |
| Create / Update / Delete type definitions | - | - | Y | Y |
| Create catalog version | - | Y | Y | Y |
| Promote dev to testing / Demote testing to dev | - | Y | Y | Y |
| Promote testing to production | - | - | Y | Y |
| Demote from production | - | - | - | Y |
| Delete production catalog version | - | - | - | Y |

### Operational API (`/api/data/v1/`)

| Operation | RO | RW | Admin | SuperAdmin |
|-----------|:--:|:--:|:-----:|:----------:|
| List / Get catalogs and instances | Y | Y | Y | Y |
| Create / Update / Delete catalogs | - | Y | Y | Y |
| Create / Update / Delete instances | - | Y | Y | Y |
| Create / Delete association links | - | Y | Y | Y |
| Set parent (containment) | - | Y | Y | Y |
| Validate catalog | - | Y | Y | Y |
| Publish / Unpublish catalog | - | - | Y | Y |
| Copy catalog | - | Y | Y | Y |
| Replace catalog | - | - | Y | Y |
| Mutate data in a published catalog | - | - | - | Y |

## Role Enforcement

Authorization is enforced at three layers of middleware, applied during route registration in `cmd/api-server/main.go`.

### 1. Global RBAC Middleware

`RBACMiddleware(provider)` is applied to both the Meta and Operational API groups. It calls the configured `RBACProvider` to extract the user's role from the request and stores it in the Echo context under the key `user_role`. If the header is missing or the role value is invalid, the request is rejected with HTTP 401.

### 2. Route-Level Role Guards

Two role-gate middlewares are created and applied to specific route registrations:

- **`requireAdmin`** (`RequireRole(RoleAdmin)`) -- Applied to all schema-write routes: entity type create/update/delete/copy/rename, attribute add/edit/remove/reorder/copy, association create/edit/delete, type definition create/update/delete, catalog publish/unpublish, and catalog replace.
- **`requireRW`** (`RequireRole(RoleRW)`) -- Applied to catalog version lifecycle routes (create, promote, demote, delete, pin management) and to all catalog and instance write routes (create, update, delete, validate, copy).

`RequireRole` compares the caller's role level against the minimum required level. Requests below the threshold receive HTTP 403.

### 3. Per-Catalog Access Control

`RequireCatalogAccess(checker)` is applied as group-level middleware on the instance route group and as per-route middleware on individual catalog endpoints. It extracts the catalog name from the `:catalog-name` URL parameter and calls the `CatalogAccessChecker` interface with a verb derived from the HTTP method (`GET` maps to `get`, `POST` to `create`, `PUT`/`PATCH` to `update`, `DELETE` to `delete`).

In development mode, `HeaderCatalogAccessChecker` always returns `true`. In production, the planned `SARCatalogAccessChecker` will perform a Kubernetes SubjectAccessReview scoped to the specific catalog resource.

The generic helper `FilterAccessible[T]` filters list results so that users only see catalogs they are authorized to access.

## Published Catalog Protection

`RequireWriteAccess(checker)` is applied to all write routes under a catalog (instance CRUD, catalog update/delete, validation). It queries whether the target catalog is published:

- If the catalog is **not published** (or not found), the request proceeds normally.
- If the catalog **is published**, only callers with the **SuperAdmin** role may proceed. All other roles receive HTTP 403 with the message "published catalog requires SuperAdmin for data mutations."

This prevents accidental data corruption in production catalogs. The check is deliberately combined with the not-found path to avoid timing-based catalog existence probing.

## API Security Configuration

**CORS**: Configured via the `CORS_ALLOWED_ORIGINS` environment variable (comma-separated list of origins). When set, the server allows `GET`, `POST`, `PUT`, `DELETE`, and `OPTIONS` methods, and permits `Content-Type`, `Authorization`, and `X-User-Role` headers. When the variable is empty, CORS middleware is a no-op.

**No authentication tokens in dev mode**: Header-based RBAC provides no real authentication. Any HTTP client can set `X-User-Role: SuperAdmin`. This mode exists solely for local development and testing.

## Security Boundaries

The system enforces a clear separation between two API surfaces:

- **Meta API** (`/api/meta/v1/`) manages the schema layer (entity types, attributes, associations, type definitions, catalog versions). Schema writes require Admin. Catalog version lifecycle requires RW minimum, with production transitions requiring higher roles.
- **Operational API** (`/api/data/v1/`) manages data (catalogs, instances, attribute values, links). Data writes require RW minimum, with published catalog mutations requiring SuperAdmin.

This separation ensures that data operators (RW) cannot accidentally alter the schema, and that published production data is protected from modification by anyone below SuperAdmin.
