---
id: "mem_0daf29f3"
topic: "Phase 4: Catalog Data Viewer — Implementation Patterns"
tags:
  - vite
  - filtering
  - containment-tree
  - operational-ui
  - worktree
  - patterns
phase: 0
difficulty: 0.5
created_at: "2026-03-12T16:25:31.157817+00:00"
created_session: 16
---
## Vite Multi-Entry Build
- Add `rollupOptions.input` with named entries in `vite.config.ts`
- Each entry needs its own HTML file (`index.html`, `operational.html`)
- BrowserRouter needs `basename="/operational"` when served under a subpath
- Single nginx serves both via path-based routing (`/` and `/operational`)
- No need for separate ports — avoids kind port mapping and CORS complexity

## EAV Attribute Filtering (SQL JOIN approach)
- Filters use aliased JOINs: `JOIN instance_attribute_values AS iav0 ON iav0.instance_id = entity_instances.id AND iav0.attribute_id = ?`
- Each filter gets its own alias (iav0, iav1, ...) to avoid conflicts
- String: `LOWER(value_string) LIKE '%val%'`
- Enum: `value_enum = 'val'`
- Number range: `.min` / `.max` suffixes → `value_number >= ?` / `value_number <= ?`
- Service layer translates filter attribute **names** to **IDs** before calling repo
- Must use `entity_instances.*` in SELECT to avoid JOIN columns interfering with GORM scan
- Count query must use `Session(&gorm.Session{})` to clone the base query — don't rebuild it

## Containment Tree Building
- `ListByCatalog` returns all instances flat, then tree built in-memory
- Two-pass: (1) build `childrenMap[parentID] → children`, (2) recursive `buildNodes("")` from root
- Entity type names cached in a `map[string]string` to avoid N+1
- Falls back to entity type ID if name resolution fails

## Parent Chain Resolution
- Walk up via `ParentInstanceID`, collect entries, reverse for root-first order
- Add cycle guard (`visited` set + `maxDepth=50`) to prevent infinite loops
- Service-layer types should NOT have json tags — serialization belongs in DTOs only

## Worktree Agent Pitfall
- Worktree agents start from the committed branch state, not the working tree
- If implementation changes aren't committed, worktree agents re-implement from scratch with potentially wrong patterns (e.g., old CatalogVersionID vs current CatalogID)
- Better approach: do backend work directly, or commit before dispatching agents
