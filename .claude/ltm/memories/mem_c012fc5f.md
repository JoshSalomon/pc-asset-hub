---
id: "mem_c012fc5f"
topic: "TD-22: System Attributes — Approach B (API-level merge)"
tags:
  - td-22
  - system-attributes
  - api-merge
  - architecture
  - quality-review
phase: 0
difficulty: 0.6
created_at: "2026-03-18T17:19:07.697574+00:00"
created_session: 17
---
## TD-22 Implementation: Common Attributes as Schema-Level Attributes

### Approach
Used Approach B (API-level merge): Name and Description remain as fields on `EntityInstance` in the DB. The API layer injects synthetic system attributes (`system: true`) into all responses.

### Key Design Decisions
- System attrs injected in 3 handler locations: `instanceDetailToDTO`, `VersionSnapshot`, `AttributeHandler.List`
- Constants in `domain/models` package (NOT `dto` — avoids service→API layer inversion): `SystemAttrName`, `SystemAttrDescription`, `SystemAttrType`, `SystemAttrNameOrdinal(-2)`, `SystemAttrDescOrdinal(-1)`, `IsSystemAttributeName()` helper
- Reserved name rejection at both handler AND service layer (defense in depth)
- `CopyAttributesFromType` filters system names before any DB calls
- Validation service checks `strings.TrimSpace(inst.Name) == ""` with ID fallback for empty names
- `AttributeValue` service type carries `Required bool` from schema, propagated through handler to response
- Name is required, Description is optional

### Quality Review Lessons
- **Case mismatch bug caught**: Backend sends lowercase `"name"`/`"description"`, UI initially checked for `"Name"`/`"Description"` — would have silently broken create/edit modals
- **DRY**: Extract shared constants to `domain/models` package (not `dto` — keeps dependency direction correct) instead of repeating string literals across 5+ locations
- **Reorder buttons**: Must check `attributes[idx-1]?.system` to prevent swapping custom attrs with system attrs
- **React keys**: System attrs have empty `id` — use `attr.id || attr.name` as key
- **Remove handler**: Needs reserved-name guard same as Add/Edit

### New Scripts
- `scripts/uncovered-new-lines.sh [base] [coverage.out]` — finds uncovered NEW Go lines
- `scripts/uncovered-new-lines-ui.sh [base] [coverage.json]` — finds uncovered NEW UI lines  
- `scripts/coverage-summary.sh` — generates coverage metrics for the report
- `scripts/test-system-attributes.sh` — 14 live system tests
