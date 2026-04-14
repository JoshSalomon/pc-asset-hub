# AI Asset Hub Documentation

Welcome to the AI Asset Hub project documentation. This guide will help you find the right information based on your role and needs.

## What is AI Asset Hub?

AI Asset Hub is a metadata-driven management system for AI assets (models, MCP servers, tools, guardrails, prompts) deployed on OpenShift clusters. Unlike traditional asset registries with fixed schemas, AI Asset Hub lets you define entity types, attributes, and relationships at runtime — making it extensible to any asset type without code changes.

It is a component of **Project Catalyst**.

## Documentation by Role

### I'm a stakeholder or product manager
- [Project Overview](overview.md) — What it does, why it matters, key capabilities

### I'm a user (schema admin or data operator)
- [User Guide](user-guide.md) — How to use the Schema Management and Data Viewer UIs

### I'm a developer
- [Developer Guide](developer-guide.md) — Setup, build, test, code structure, how to extend
- [Architecture Overview](architecture-overview.md) — Design decisions, data model, API design, testing strategy

### I'm deploying or operating this system
- [Deployment Guide](deployment-guide.md) — Kind cluster, OpenShift, configuration, troubleshooting

### I'm reviewing security or access control
- [Security and Roles](security-and-roles.md) — RBAC model, 4 user roles, per-catalog access, API security

### I'm evaluating the project
Start with the [Project Overview](overview.md), then read [Architecture Overview](architecture-overview.md) and [Security and Roles](security-and-roles.md).

## Quick Links

| Resource | Location |
|----------|----------|
| Product Requirements | [`PRD.md`](../PRD.md) |
| Architecture Document | [`docs/architecture.md`](../architecture.md) |
| Test Plan | [`docs/test-plan.md`](../test-plan.md) |
| Coverage Report | [`docs/coverage-report.md`](../coverage-report.md) |
| Technical Debt Log | [`docs/td-log.md`](../td-log.md) |
| Type System Design | [`docs/superpowers/specs/2026-04-11-type-system-design.md`](../superpowers/specs/2026-04-11-type-system-design.md) |

## System at a Glance

```
                    +-----------+
                    |    UI     |  :30000
                    | React +   |  Schema Management (/schema/*)
                    | PatternFly |  Data Viewer (/operational/*)
                    +-----+-----+
                          |
                          v
                    +-----+-----+
                    | API Server|  :30080
                    |  Go/Echo  |  Meta API: /api/meta/v1/*
                    |           |  Data API: /api/data/v1/*
                    +-----+-----+
                          |
                    +-----+-----+
                    | PostgreSQL|  Entity types, instances,
                    |           |  type definitions, catalogs
                    +-----------+
                          ^
                    +-----+-----+
                    |  Operator |  Watches CatalogVersion CRs
                    |  Go/SDK   |  Manages Catalog CRs
                    +-----------+
```

## Roles Summary

| Role | Schema | Data | Catalogs | Publishing |
|------|--------|------|----------|------------|
| RO | View | View | View | - |
| RW | View | Create/Edit | Create/Edit/Validate | - |
| Admin | Full CRUD | Create/Edit | Full + Lifecycle | Publish/Unpublish |
| SuperAdmin | Full CRUD | Full (incl. published) | Full | Full + Override |
