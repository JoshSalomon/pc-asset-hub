---
id: "mem_1cd1b1d4"
topic: "Containment & Association Links — implementation patterns and gotchas"
tags:
  - containment
  - association-links
  - operational-api
  - gotchas
  - routes
phase: 0
difficulty: 0.65
created_at: "2026-03-11T14:40:57.227225+00:00"
created_session: 16
---
## Containment & Association Links (Phase 3 — implemented)

### Architecture
- `CreateContainedInstance` validates containment association exists in CV between parent and child entity types
- Association links validated against CV association definitions — source/target entity types must match
- Forward/reverse references resolved via `resolveLinks` helper (boolean flag for direction)
- `cascadeDelete` cleans up association links via `DeleteByInstance` before soft-deleting instances
- Duplicate link prevention: checks existing forward refs before creating

### Key Validations Added (from quality review)
- Parent instance must belong to the catalog (not cross-catalog containment)
- Source and target instances must be in the same catalog for links
- `DeleteAssociationLink` verifies link ownership (source instance in correct catalog)
- `ListContainedInstances` returns filtered count (not unfiltered `ListByParent` total)

### Route Registration Gotcha
Echo routes: static path segments (`/links`, `/references`, `/referenced-by`) MUST be registered BEFORE parameterized containment route (`/:child-type`). Echo prefers static over parameterized, but registration order matters. Entity types named "links", "references", or "referenced-by" are reserved.

### Repository Interface Changes
- `AssociationLinkRepository` gained `GetByID` and `DeleteByInstance` methods
- `DeleteByInstance` deletes all links where instance is source OR target

### Live System Test Script
- `scripts/test-containment-links.sh` — 18 parameterized tests
- Usage: `./scripts/test-containment-links.sh [API_BASE_URL]`
- Default: `http://localhost:30080`
- Self-cleaning with unique timestamp-based names

### Technical Debt
- TD-26: Extract shared instance creation logic
- TD-27: Fix ListContainedInstances pagination (in-memory filtering)
- TD-28: Code quality (handler duplication, N+1 queries, component decomposition)
