# MCP Gateway CRD Reference — For Export Plugin PoC

Source: [github.com/Kuadrant/mcp-gateway](https://github.com/Kuadrant/mcp-gateway)

## Overview

The MCP Gateway is an Envoy-based gateway for aggregating and routing multiple MCP (Model Context Protocol) servers behind a single endpoint. It uses three CRDs:

| CRD | Short Name | Purpose |
|-----|-----------|---------|
| `MCPGatewayExtension` | — | Extends a K8s Gateway with MCP protocol support. One per namespace. |
| `MCPServerRegistration` | `mcpsr` | Registers a backend MCP server with the gateway via HTTPRoute reference. |
| `MCPVirtualServer` | `mcpvs` | Defines a virtual server exposing a curated subset of tools. |

**API Group:** `mcp.kuadrant.io/v1alpha1`

## CRD 1: MCPServerRegistration

Registers a backend MCP server. The gateway discovers its tools and makes them available for federation.

```yaml
apiVersion: mcp.kuadrant.io/v1alpha1
kind: MCPServerRegistration
metadata:
  name: github-mcp
  namespace: mcp-system
spec:
  # HTTPRoute pointing to the backend MCP server's Service
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: github-mcp-route        # must exist in same or referenced namespace
    namespace: ""                  # optional, defaults to same namespace

  # Prefix for all federated tools from this server (avoids naming conflicts)
  prefix: github_                  # e.g., tools become github_search, github_create_issue

  # MCP endpoint path on the backend server
  path: /mcp                       # default: /mcp

  # Optional: authentication credentials for the backend MCP server
  credentialRef:
    name: github-mcp-token         # Secret name
    key: token                     # Key within the Secret (default: "token")
```

**Status fields:**
- `conditions[].type: Ready` — whether the server is reachable and tools discovered
- `discoveredTools` — count of tools found on this server

**Key points for the exporter:**
- Each `MCPServerRegistration` maps to one backend MCP server
- The `targetRef` points to an `HTTPRoute` (Gateway API), NOT directly to a Service
- The `prefix` is important for tool namespacing when aggregating multiple servers
- Credentials are optional — stored in a referenced Secret

## CRD 2: MCPVirtualServer

Defines a virtual endpoint that exposes a curated subset of tools from registered servers. This is the **primary target for the PoC** — it lets you control which tools are accessible to a specific consumer.

```yaml
apiVersion: mcp.kuadrant.io/v1alpha1
kind: MCPVirtualServer
metadata:
  name: reporting-agents-vs
  namespace: mcp-system
spec:
  description: "Virtual server for reporting agents — read-only tools only"
  tools:
    - github_list_repos            # prefixed tool names from MCPServerRegistrations
    - github_get_issue
    - jira_search_issues
    - jira_get_board
```

**Key points for the exporter:**
- `tools` is a flat list of prefixed tool names (strings)
- Tools must be available from registered `MCPServerRegistration` resources
- The virtual server acts as a filter — clients connecting to this endpoint only see listed tools
- No `targetRef` — the virtual server is served by the gateway broker directly

## CRD 3: MCPGatewayExtension

Extends a K8s Gateway to handle MCP traffic. Infrastructure-level — typically one per namespace, configured by a platform admin.

```yaml
apiVersion: mcp.kuadrant.io/v1alpha1
kind: MCPGatewayExtension
metadata:
  name: mcp-gateway-ext
  namespace: mcp-system
spec:
  targetRef:
    group: gateway.networking.k8s.io
    kind: Gateway
    name: mcp-gateway
    namespace: gateway-system
    sectionName: mcp               # listener name on the Gateway
  publicHost: mcp.example.com      # optional override
  backendPingIntervalSeconds: 60   # health check interval
  httpRouteManagement: Enabled     # operator manages the HTTPRoute
```

**Not relevant for the exporter** — this is gateway infrastructure. The exporter only produces `MCPServerRegistration` and `MCPVirtualServer` CRs.

## PoC Mapping: Asset Hub Catalog → MCP Gateway CRs

### What the exporter produces

For a catalog with MCP servers and tools, the exporter produces:

1. **One `MCPServerRegistration` per mcp-server instance** — registers the backend server with the gateway
2. **One `MCPVirtualServer` per catalog (or per view, future)** — curates which tools are exposed

### Mapping: Asset Hub → MCPServerRegistration

| Asset Hub data | MCPServerRegistration field |
|---------------|---------------------------|
| mcp-server instance name | `metadata.name` |
| catalog name | `metadata.labels["assethub.io/catalog"]` |
| mcp-server `route_name` attribute | `spec.targetRef.name` — references existing HTTPRoute by name |
| (derived: instance name + `_`) | `spec.prefix` — no attribute needed, derived from server instance name |
| mcp-server `mcp_path` attribute (optional) | `spec.path` (default: /mcp) |
| mcp-server `credential_secret` attribute (optional) | `spec.credentialRef.name` |

**Open question:** `MCPServerRegistration.spec.targetRef` references an `HTTPRoute`, not a direct endpoint URL. The exporter needs to either:
- (a) Also generate the `HTTPRoute` + `Service` resources that point to the MCP server endpoint
- (b) Assume the HTTPRoute/Service already exist and just reference them by name
- (c) For the PoC, generate all three resources (Service, HTTPRoute, MCPServerRegistration) as a bundle

### Mapping: Asset Hub → MCPVirtualServer

| Asset Hub data | MCPVirtualServer field |
|---------------|----------------------|
| catalog name (or view name) | `metadata.name` |
| catalog description | `spec.description` |
| All mcp-tool instances (with server prefix) | `spec.tools[]` — prefixed tool names |

**Tool name construction:** Each tool in the virtual server is `{server_instance_name}_{tool_instance_name}`. The prefix is derived from the server instance name (no separate attribute). The exporter walks the containment tree to find each tool's parent server.

### Example output

Given a catalog `reporting-agents` with:
- mcp-server `github` (route_name: `github-mcp-route`)
  - mcp-tool `list-repos`
  - mcp-tool `create-issue`
- mcp-server `jira` (route_name: `jira-mcp-route`)
  - mcp-tool `search-issues`

The exporter produces:

```yaml
---
apiVersion: mcp.kuadrant.io/v1alpha1
kind: MCPServerRegistration
metadata:
  name: github
  namespace: mcp-system
  labels:
    assethub.io/catalog: reporting-agents
    assethub.io/exporter: mcp-gateway
spec:
  prefix: github_
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: github-mcp-route
  path: /mcp
---
apiVersion: mcp.kuadrant.io/v1alpha1
kind: MCPServerRegistration
metadata:
  name: jira
  namespace: mcp-system
  labels:
    assethub.io/catalog: reporting-agents
    assethub.io/exporter: mcp-gateway
spec:
  prefix: jira_
  targetRef:
    group: gateway.networking.k8s.io
    kind: HTTPRoute
    name: jira-mcp-route
  path: /mcp
---
apiVersion: mcp.kuadrant.io/v1alpha1
kind: MCPVirtualServer
metadata:
  name: reporting-agents
  namespace: mcp-system
  labels:
    assethub.io/catalog: reporting-agents
    assethub.io/exporter: mcp-gateway
spec:
  description: "Reporting agents - MCP tools for data analysis"
  tools:
    - github_list-repos
    - github_create-issue
    - jira_search-issues
```

## Resolved Design Decisions

These questions were resolved during design review. See `2026-05-10-export-plugins-design.md` for full context.

1. **HTTPRoute generation:** Resolved — **reference only, not generated.** The exporter references existing HTTPRoutes by name via the `route_name` attribute on mcp-server instances. HTTPRoutes and Services are managed by the platform admin, not Asset Hub.

2. **Prefix source:** Resolved — **derived from server instance name.** No separate `prefix` attribute needed. The exporter uses `server_instance.name + "_"` automatically. Instance names are unique within the catalog and DNS-1123 compliant (enforced by Stage 0 validation).

3. **Credential handling:** Resolved — **reference by attribute, don't generate Secrets.** If the mcp-server instance has a `credential_secret` attribute, its value is used as `spec.credentialRef.name` in the generated CR. The platform admin provisions the Secret in the target namespace. If the attribute is absent, `credentialRef` is omitted from the CR.

4. **Tool name format:** Resolved — **DNS-1123 enforcement on instance names.** Instance names are tightened to DNS-1123 format (Stage 0 prerequisite), so all names are valid for K8s and MCP tool identifiers. No runtime sanitization needed.
