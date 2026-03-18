---
id: "mem_dd81fc7a"
topic: "Catalog Validation (Phase 6) implementation patterns"
tags:
  - catalog
  - validation
  - phase-6
  - architecture
phase: 0
difficulty: 0.5
created_at: "2026-03-15T23:16:40.548454+00:00"
created_session: 16
---
## Catalog Validation Service

- `CatalogValidationService` in `internal/service/operational/validation_service.go`
- Validates: required attrs, enum values, mandatory associations (target_cardinality min >= 1), containment consistency, unpinned entity types
- Returns `ValidationResult{Status, Errors[]ValidationError}` — service types have NO json tags; DTO layer converts
- API: `POST /api/data/v1/catalogs/{name}/validate` — requires RW+ role
- Route registered in `RegisterCatalogRoutes` with `requireRW` and `requireCatalogAccess`

## Key Design Decisions
- Associations pre-loaded into `assocCache` once (not per-entity-type loop) to avoid duplicate DB calls
- Unpinned entity type instances produce validation errors (not silently skipped)
- Containment associations excluded from mandatory assoc checks (validated separately)
- `cardinalityMinGE1` parses cardinality string to check if min >= 1
- Bidirectional association reverse-direction check deferred (only forward refs checked)

## UI Patterns
- Shared `useValidation(name, loadCatalog)` hook in `ui/src/hooks/useValidation.ts`
- Shared `ValidationResults` component in `ui/src/components/ValidationResults.tsx`
- Both meta and operational catalog detail pages use the shared hook/component
- Validate button guarded by `canWrite` check in both UIs

## Live Tests
- `scripts/test-validation.sh` — 9 tests, uses timestamp-based names to avoid collisions with existing data
