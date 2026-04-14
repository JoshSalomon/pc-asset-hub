# AI Asset Hub Overview

## What is AI Asset Hub?

AI Asset Hub is a metadata management system for AI assets on OpenShift. It provides a centralized place to define, organize, track, and publish AI building blocks -- models, MCP servers, tools, guardrails, evaluators, and prompts -- without code changes when new asset types emerge.

## The Problem

Organizations deploying AI on OpenShift face a growing inventory of interconnected assets. Models depend on guardrails; MCP servers contain tools. Tracking these typically involves spreadsheets or systems that hardcode a fixed set of asset types. When a new kind of asset appears, these rigid systems require engineering to extend, and there is no single source of truth for what exists or which combinations are production-ready.

## Key Capabilities

**Dynamic Schema Definition.** Administrators define entity types, attributes, and relationships through the UI. New asset types are configured, not coded.

**Versioned Everything.** Every change to an entity type, attribute, or data record is versioned automatically with full history and side-by-side comparison.

**Catalog Versions.** A catalog version pins entity type versions together into a validated combination that progresses through lifecycle stages (Development, Testing, Production) with role-based gates.

**Validation and Publishing.** Catalogs are validated for required attributes, type constraints, and mandatory relationships. Only valid catalogs can be published, and published catalogs are write-protected.

**Containment and References.** An MCP server contains tools, a model references a guardrail, or two assets reference each other mutually. These relationships are formally defined, validated, and navigable.

**Copy and Replace.** Update a published catalog safely by copying, editing, validating, and atomically replacing the original.

## Use Cases

- Cataloging AI models with their configurations, guardrails, and version history
- Tracking MCP servers and their contained tools with references to the models they serve
- Managing prompt libraries with versioned templates tied to model configurations
- Publishing validated asset configurations discoverable by pipelines and dashboards via Kubernetes APIs

## Project Catalyst

AI Asset Hub is a component of Project Catalyst, Red Hat's initiative for AI on OpenShift. It provides the metadata layer that other Catalyst components use to discover AI assets. Promoted configurations are surfaced as Kubernetes custom resources for ecosystem-wide discovery.

## The Key Differentiator

Traditional asset management systems define their schema at compile time -- database tables and UI forms are fixed when the software is built. AI Asset Hub defines its schema at runtime. The same software manages models today, MCP servers tomorrow, and whatever emerges next -- without a code change.
