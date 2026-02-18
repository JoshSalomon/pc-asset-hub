---
id: "mem_416f76ae"
topic: "AI Asset Hub - Tech Stack Decisions"
tags:
  - architecture
  - tech-stack
  - asset-hub
phase: 0
difficulty: 0.7
created_at: "2026-02-12T17:36:23.527040+00:00"
created_session: 1
---
## Tech Stack for AI Asset Hub (Project Catalyst component)

- **Backend**: Go (single language for API server + operator)
- **Web framework**: Echo (labstack/echo) - route grouping, idiomatic error handling
- **ORM**: GORM with build-tag driver switching (`//go:build postgres` / `//go:build sqlite`)
- **Production DB**: PostgreSQL
- **Development DB**: SQLite (lower footprint)
- **API style**: REST with OpenAPI 3.0
- **UI**: React + TypeScript + PatternFly + @patternfly/react-topology
- **UI state**: React Query (TanStack Query) for server state, React Context for UI state. No Redux.
- **UI build**: Vite
- **ID generation**: UUID v7 (time-ordered, B-tree friendly) via `google/uuid`
- **Concurrency**: Optimistic locking with version-based conflict detection (409 on version mismatch)
- **Operator**: operator-sdk with AssetHub CRD + dynamically generated CRDs on catalog promotion
