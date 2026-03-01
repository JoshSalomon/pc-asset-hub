---
id: "mem_7a79ce47"
topic: "Containment tree and version snapshot architecture patterns"
tags:
  - architecture
  - containment-tree
  - version-snapshot
  - associations
  - deduplication
phase: 0
difficulty: 0.6
created_at: "2026-03-01T17:36:02.100452+00:00"
created_session: 12
---
# Architecture Patterns - Containment Tree & Version Snapshot

## Containment tree deduplication
`assocRepo.GetContainmentGraph()` returns one edge per entity type version. Multi-version entity types produce duplicate edges. Deduplicate with `map[string]map[string]bool`:
```go
parentToChildren := make(map[string]map[string]bool)
for _, edge := range edges {
    if parentToChildren[edge.SourceEntityTypeID] == nil {
        parentToChildren[edge.SourceEntityTypeID] = make(map[string]bool)
    }
    parentToChildren[edge.SourceEntityTypeID][edge.TargetEntityTypeID] = true
}
```

## Version snapshot — resolve names in service layer
`GetVersionSnapshot` resolves enum names and target entity type names before returning. Don't resolve in handler — the service has access to repos.

Add optional repos via `WithEnumRepo(svc, enumRepo)` pattern (same as `WithCatalogRepos`).

## Associations need both directions
`GetVersionSnapshot` calls both:
- `assocRepo.ListByVersion(versionID)` — outgoing associations
- `assocRepo.ListByTargetEntityType(entityTypeID)` — incoming associations

Wrap each in `DirectedAssociation` with direction metadata. For incoming, filter to latest version of source entity type only (skip old versions).

## Association display labels
| Type | Outgoing | Incoming |
|------|----------|----------|
| containment | contains (green) | contained by (grey) |
| directional | references (blue) | referenced by (grey) |
| bidirectional | references (mutual) (purple) | references (mutual) (purple) |

Role shown is perspective-dependent: outgoing → target_role, incoming → source_role.

