# Export Plugin Architecture

This document describes the export plugin system — how Asset Hub transforms catalog data into consumer-specific artifacts via pluggable exporters. The output format is determined by the plugin: K8s CRs, ConfigMaps, Terraform HCL, Ansible playbooks, JSON, or anything the target system requires.

## Overview

The export system has two layers:

1. **Framework** — bindings, configuration, triggering, and delivery. Common to all exporters.
2. **Plugins** — the actual transformation logic. Each plugin receives catalog data and produces artifacts in whatever format the target system consumes.

Plugins can be **built-in** (compiled Go code) or **webhook-based** (external HTTP services registered via K8s CRDs). The framework treats both identically through the `Exporter` interface.

```
                    ┌─────────────────────────────────────┐
                    │         Export Framework             │
                    │                                     │
  Catalog ──────▶  │  Bindings ──▶ Registry ──▶ Exporter │ ──▶ Artifacts
  (instances,      │     │            │            │      │     (any format)
   attributes,     │     │            │            │      │
   links)          │     ▼            ▼            ▼      │
                    │  Parameters   Built-in   Webhook    │
                    │              Exporters   Exporters   │
                    └─────────────────────────────────────┘
```

## Key Concepts

### Exporter

An exporter transforms catalog data into artifacts for a target system. It implements:

```go
type Exporter interface {
    Name() string
    Description() string
    ParameterSchema() []ParameterDef
    ValidateSchema(params map[string]string, schema SchemaInfo) error
    Export(ctx context.Context, input ExportInput) (*ExportOutput, error)
}
```

- `ValidateSchema` is called when a binding is created — the exporter checks that the catalog's schema is compatible with its requirements (e.g., the required entity types and attributes exist).
- `Export` receives the full catalog data and produces artifacts.

### Binding

A binding connects a catalog to an exporter with specific parameters. It is the unit of configuration — "use exporter X on catalog Y with parameters Z."

```
POST /api/data/v1/catalogs/{name}/export-bindings
{
  "exporter_name": "webhook-mcp-gateway",
  "parameters": {
    "server_type": "mcp-server",
    "tool_type": "mcp-tool",
    "target_namespace": "production"
  }
}
```

Bindings are stored in the database and can be run on demand (`POST .../run`) or automatically on catalog publish.

### Registry

The in-memory exporter registry holds all available exporters. Built-in exporters are registered at startup. Webhook exporters are registered dynamically via K8s CRD watches.

```
Registry
├── mcp-gateway (built-in)
└── webhook-mcp-gateway (webhook, from ExporterPlugin CR)
```

The registry is thread-safe. It tracks the source (`built-in` or `webhook`) and health status of each exporter.

## Built-in Exporters

Built-in exporters are compiled Go types registered in `cmd/api-server/main.go`:

```go
exporterRegistry := export.NewExporterRegistry()
exporterRegistry.Register(export.NewMCPGatewayExporter())
```

The built-in MCP Gateway exporter produces `MCPServerRegistration` and `MCPVirtualServer` CRs from catalog data.

## Webhook Exporters

Webhook exporters are external HTTP services that implement the Asset Hub webhook protocol. They are discovered via `ExporterPlugin` CRDs — deploying a new exporter requires only a Deployment, a Service, and a CR. No Asset Hub code changes.

### ExporterPlugin CRD

```yaml
apiVersion: assethub.project-catalyst.io/v1alpha1
kind: ExporterPlugin
metadata:
  name: my-exporter
  namespace: assethub
spec:
  description: "My custom exporter"
  endpoint: https://my-exporter.assethub.svc.cluster.local
  timeoutSeconds: 15
  parameterSchema:
    - name: target_namespace
      type: string
      description: "K8s namespace for generated CRs"
      required: true
      default: "default"
  trustedSubjects:
    - system:serviceaccount:assethub:assethub-api-server
    - system:serviceaccount:assethub:assethub-operator
status:
  phase: Ready       # Ready | Unhealthy | Error | Unknown
  lastHealthCheck: "2026-06-09T12:00:00Z"
  message: ""
```

| Field | Description |
|-------|-------------|
| `endpoint` | Base URL of the plugin service (scheme + host, no path). |
| `timeoutSeconds` | Export timeout in seconds (minimum 3, default 10). Validate timeout is half the export timeout. |
| `parameterSchema` | Parameter definitions exposed to users when creating bindings. |
| `trustedSubjects` | K8s ServiceAccount subjects allowed to call the plugin. Used by the plugin for TokenReview auth. |
| `status.phase` | Health status set by the operator's health probe controller. |

### Dual-Controller Architecture

Two independent controllers manage ExporterPlugin CRs:

```
                ExporterPlugin CR
                ┌──────────────┐
         watch  │  spec:       │  update status
    ┌──────────▶│    endpoint  │◀──────────┐
    │           │  status:     │           │
    │           │    phase     │           │
    │           └──────────────┘           │
    │                                      │
┌───┴────────┐                     ┌───────┴──────┐
│ API Server │                     │   Operator   │
│            │                     │              │
│ Watcher:   │                     │ Reconciler:  │
│ - handleAdd│                     │ - GET /health│
│ - handleUpd│                     │ - classify   │
│ - handleDel│                     │ - write      │
│            │                     │   status     │
│ Registers/ │                     │              │
│ deregisters│                     │ Every 30s    │
│ WebhookExp │                     │              │
└────────────┘                     └──────────────┘
```

**API server** watches CRs via a K8s informer. On add/update, it creates a `WebhookExporter` adapter and registers it in the exporter registry. On delete, it deregisters. The `WebhookExporter` implements the same `Exporter` interface as built-in exporters — the rest of the system doesn't distinguish between them.

**Operator** runs an `ExporterPluginReconciler` that periodically probes `GET {endpoint}/health` and writes `status.phase` to the CR. The API server reads health via its informer.

This separation ensures the operator can be down without affecting export functionality.

### Health Status Phases

| Phase | Meaning | Trigger |
|-------|---------|---------|
| `Unknown` | Initial state, no probe yet | CR just created, or no probe for 3x interval |
| `Ready` | Plugin is healthy | `GET /health` returns 200 |
| `Unhealthy` | Plugin responded but not healthy | Non-200 status or timeout |
| `Error` | Plugin unreachable | Connection refused, DNS/TLS failure, auth rejected |

### Orphaned Bindings

When an ExporterPlugin CR is deleted, bindings referencing that exporter become orphaned:

- `RunAll` (publish-triggered) skips orphaned bindings with status `skipped`
- Single `Run` on an orphaned binding returns 404
- The UI shows a warning icon with tooltip "Exporter not registered"
- Orphaned bindings can be disabled but not updated with new parameters

## Webhook Protocol (v1)

The plugin service exposes three HTTP endpoints. All requests include the `X-AssetHub-Protocol-Version: v1` header.

### POST /validate

Called when a binding is created. The plugin verifies that the catalog schema is compatible with its requirements.

**Request:**
```json
{
  "parameters": {"server_type": "mcp-server", "tool_type": "mcp-tool"},
  "schema": {
    "entity_types": [
      {
        "name": "mcp-server",
        "attributes": ["route_name", "endpoint"],
        "associations": [
          {"name": "tools", "type": "containment", "target_entity_type": "mcp-tool"}
        ]
      }
    ]
  }
}
```

**Response (200):**
```json
{"valid": true}
```

**Response (400):**
```json
{"valid": false, "error": "entity type 'mcp-server' missing required attribute 'route_name'"}
```

### POST /export

Called to produce artifacts from catalog data. The plugin receives the full, unfiltered catalog model and is responsible for any filtering (e.g., virtual server instance selection).

**Request:**
```json
{
  "catalog_name": "my-catalog",
  "catalog_description": "Production catalog",
  "parameters": {"server_type": "mcp-server", "target_namespace": "prod"},
  "instances_by_type": {
    "mcp-server": [
      {
        "id": "inst-1",
        "name": "github-server",
        "description": "GitHub MCP server",
        "attributes": {"route_name": "github-route", "endpoint": "/mcp"},
        "parent_id": "",
        "links": {
          "allowed-tools": [
            {"target_instance_id": "t1", "target_instance_name": "create-pr", "target_entity_type": "mcp-tool"}
          ]
        }
      }
    ]
  },
  "children_of": {"inst-1": ["tool-1", "tool-2"]},
  "virtual_server_instance_name": "my-vs"
}
```

**Response (200):**
```json
{
  "artifacts": [
    {
      "api_version": "mcp.kuadrant.io/v1alpha1",
      "kind": "MCPServerRegistration",
      "name": "github-server",
      "namespace": "prod",
      "yaml": "apiVersion: mcp.kuadrant.io/v1alpha1\nkind: MCPServerRegistration\n..."
    }
  ],
  "warnings": ["Server 'test-server' has no 'route_name' attribute, skipped"]
}
```

### GET /health

Health probe endpoint called by the operator every 30 seconds.

**Response (200):**
```json
{"status": "ok"}
```

### Authentication

All requests from Asset Hub include a `Authorization: Bearer <token>` header with the calling component's ServiceAccount token. The plugin validates tokens via K8s TokenReview against the `trustedSubjects` list in its CR spec.

### Error Handling

| HTTP Status | Meaning | Framework Behavior |
|------------|---------|-------------------|
| 200 | Success | Parse response body |
| 400-499 | Client error (bad params, schema mismatch) | Return validation error to user |
| 500-599 | Server error | Return "webhook plugin error" |
| Timeout | Plugin didn't respond in time | Return timeout error |
| Connection refused | Plugin unreachable | Return "webhook plugin unreachable" |
| Redirect (3xx) | Unexpected | Rejected — plugins must not redirect |

Response body size is limited to 10 MB.

## Writing a Plugin

### Minimal Plugin

A plugin is any HTTP server that implements `/validate`, `/export`, and `/health`. Here's the minimal contract:

1. `GET /health` returns `{"status": "ok"}` with 200
2. `POST /validate` checks schema compatibility, returns `{"valid": true/false}`
3. `POST /export` transforms instances into artifacts

### Example: webhook-mcp-gateway

The `examples/webhook-mcp-gateway/` directory contains a complete reference implementation that produces MCP Gateway CRs. It demonstrates:

- Token-based authentication via `Authorization: Bearer` header
- Protocol version checking via `X-AssetHub-Protocol-Version` header
- Schema validation (required entity types, attributes, associations)
- Virtual server instance filtering from link data
- YAML generation with proper escaping (`yamlQuote`)
- K8s Deployment + Service + ExporterPlugin CR manifests

Deploy the example:
```bash
# Included in kind-deploy.sh — built and deployed automatically
./scripts/kind-deploy.sh rebuild "kubectl --context kind-assethub"
```

### Deployment Checklist

1. Build and deploy your plugin as a Deployment + Service in the `assethub` namespace
2. Create an `ExporterPlugin` CR with your service endpoint
3. Verify health: `kubectl get exporterplugin -n assethub` should show `phase: Ready`
4. Create a binding: `POST /api/data/v1/catalogs/{name}/export-bindings`
5. Run the export: `POST /api/data/v1/catalogs/{name}/export-bindings/{id}/run`

## RBAC

| Component | Resource | Verbs |
|-----------|----------|-------|
| API server SA | `exporterplugins` | `get, list, watch` |
| API server SA | `exporterplugins/status` | (none — read via informer) |
| Operator SA | `exporterplugins` | `get, list, watch` |
| Operator SA | `exporterplugins/status` | `get, update, patch` |
| Plugin SA | `tokenreviews` | `create` (for validating incoming tokens) |

Binding RBAC follows catalog permissions: Admin+ can create/update/delete bindings, RW+ can run exports, RO can list bindings (parameters hidden).

## Non-K8s Mode

When running outside a cluster (local development without `rest.InClusterConfig`), ExporterPlugin discovery is disabled. Only built-in exporters are available. The API server logs a warning and continues without the watcher.

## Source Files

| File | Purpose |
|------|---------|
| `internal/service/operational/export/types.go` | `Exporter` interface and data types |
| `internal/service/operational/export/registry.go` | Thread-safe exporter registry |
| `internal/service/operational/export/binding_service.go` | Binding CRUD, schema building, export execution |
| `internal/service/operational/export/publish_service.go` | Publish-triggered export with preview |
| `internal/service/operational/export/webhook_exporter.go` | `WebhookExporter` adapter (HTTP client) |
| `internal/service/operational/export/webhook_protocol.go` | Wire protocol types and conversion |
| `internal/service/operational/export/mcp_gateway_exporter.go` | Built-in MCP Gateway exporter |
| `internal/infrastructure/k8s/exporterplugin_watcher.go` | K8s informer for ExporterPlugin CRs |
| `internal/operator/controllers/exporterplugin_controller.go` | Health probe reconciler |
| `internal/operator/api/v1alpha1/exporterplugin_types.go` | CRD Go types |
| `deploy/k8s/operator/exporterplugin-crd.yaml` | CRD manifest |
| `examples/webhook-mcp-gateway/` | Reference plugin implementation |
