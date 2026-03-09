---
id: "mem_9f3c1baa"
topic: "Catalog concept — data layer architecture"
tags:
  - catalog
  - architecture
  - data-layer
  - validation
  - CRD
phase: 0
difficulty: 0.7
created_at: "2026-03-08T15:37:43.440185+00:00"
created_session: 14
---
## Catalog Entity — Key Architectural Decision

### What
A **Catalog** is a named collection of entity instances that uses a CatalogVersion (CV) as its schema. This separates schema (CV) from data (Catalog).

### Why
- Multiple catalogs can share the same CV (different apps, different data sets)
- CV is the "bill of materials" (schema); Catalog is the actual data container
- Without this separation, only one set of data per CV is possible

### Model
```
CatalogVersion (schema) ←── Catalog (data container) ←── Entity Instances (data)
     1                          *                              *
```

### Validation
- Entity instances are created in **draft mode** — incomplete data allowed
- Validation runs **on demand** or warned at **CV promotion**
- Validation status: `draft` | `valid` | `invalid`
- Checks: required attributes, attribute types, mandatory associations (cardinality 1/1..n), containment consistency

### V1 Constraints
- Catalog CV pin is **immutable** (re-pinning is TD-9)
- CV lookup by name is TD-10

### CRDs
- CatalogVersion CR: schema discovery (existing)
- Catalog CR: data discovery — name, pinned CV ref, API endpoint, catalog ID, validation status
- Catalog CR scoping (namespaced vs cluster-scoped) TBD

### API
- Operational API scoped to catalog: `/api/data/v1/catalogs/{catalog-id}/...`
- Catalog CRUD: `POST/GET /api/data/v1/catalogs`
- Validation: `POST /api/data/v1/catalogs/{id}/validate`
