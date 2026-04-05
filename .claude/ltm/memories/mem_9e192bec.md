---
id: "mem_9e192bec"
topic: "Stage guards: checkCVEditAllowed pattern for CV metadata + pin editing"
tags:
  - stage-guard
  - catalog-version
  - RBAC
  - security
phase: 0
difficulty: 0.4
created_at: "2026-04-03T19:23:11.935055+00:00"
created_session: 17
---
## Pattern: checkCVEditAllowed(cv, role, operation)

Unified stage guard for catalog version operations (replaces old `checkPinEditAllowed`):
- **development**: RW+ allowed
- **testing**: SuperAdmin only
- **production**: blocked entirely

Used by: `UpdateCatalogVersion`, `AddPin`, `UpdatePin`, `RemovePin`.

The handler must call `mapRole(middleware.GetRoleFromContext(c))` to convert middleware role to service role before passing to the service.

## UI side
- `CatalogVersionDetailPage`: `canEdit = hasWriteRole && stage != production && (stage != testing || SuperAdmin)`. `canEditPins = canEdit`.
- `CatalogDetailPage` (operational): `canValidate = canWrite && (!published || SuperAdmin)` — prevents validate button on published catalogs for non-SuperAdmin.
