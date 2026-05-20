package export_test

import (
	"context"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/project-catalyst/pc-asset-hub/internal/service/operational/export"
)

func mcpInput() export.ExportInput {
	return export.ExportInput{
		CatalogName: "prod-agents",
		CatalogDesc: "Production AI agents",
		Parameters: map[string]string{
			"server_type":      "mcp-server",
			"tool_type":        "mcp-tool",
			"target_namespace": "mcp-system",
		},
		InstancesByType: map[string][]*export.ExportInstance{
			"mcp-server": {
				{
					ID: "s1", EntityType: "mcp-server", Name: "github",
					Attributes: map[string]any{
						"route_name":        "github-mcp-route",
						"mcp_path":          "/mcp",
						"credential_secret": "github-token",
					},
				},
				{
					ID: "s2", EntityType: "mcp-server", Name: "jira",
					Attributes: map[string]any{
						"route_name": "jira-mcp-route",
					},
				},
			},
			"mcp-tool": {
				{ID: "t1", EntityType: "mcp-tool", Name: "list-repos", ParentID: "s1"},
				{ID: "t2", EntityType: "mcp-tool", Name: "create-issue", ParentID: "s1"},
				{ID: "t3", EntityType: "mcp-tool", Name: "search-issues", ParentID: "s2"},
			},
		},
		ChildrenOf: map[string][]*export.ExportInstance{
			"s1": {
				{ID: "t1", EntityType: "mcp-tool", Name: "list-repos"},
				{ID: "t2", EntityType: "mcp-tool", Name: "create-issue"},
			},
			"s2": {
				{ID: "t3", EntityType: "mcp-tool", Name: "search-issues"},
			},
		},
	}
}

// T-34.50: Exporter produces MCPServerRegistration per server instance
func TestMCPGateway_ProducesOnePerServer(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	out, err := e.Export(context.Background(), mcpInput())
	require.NoError(t, err)

	serverCRs := 0
	for _, a := range out.Artifacts {
		if a.Kind == "MCPServerRegistration" {
			serverCRs++
		}
	}
	assert.Equal(t, 2, serverCRs)
}

// T-34.51: MCPServerRegistration has correct apiVersion
func TestMCPGateway_APIVersion(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	out, err := e.Export(context.Background(), mcpInput())
	require.NoError(t, err)

	for _, a := range out.Artifacts {
		assert.Equal(t, "mcp.kuadrant.io/v1alpha1", a.APIVersion)
		assert.Contains(t, a.YAML, "apiVersion: mcp.kuadrant.io/v1alpha1")
	}
}

// T-34.52: MCPServerRegistration has correct kind
func TestMCPGateway_Kind(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	out, err := e.Export(context.Background(), mcpInput())
	require.NoError(t, err)

	assert.Equal(t, "MCPServerRegistration", out.Artifacts[0].Kind)
	assert.Contains(t, out.Artifacts[0].YAML, "kind: MCPServerRegistration")
}

// T-34.53: MCPServerRegistration metadata.name matches server instance name
func TestMCPGateway_MetadataName(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	out, err := e.Export(context.Background(), mcpInput())
	require.NoError(t, err)

	assert.Equal(t, "github", out.Artifacts[0].Name)
	assert.Contains(t, out.Artifacts[0].YAML, "  name: github")
}

// T-34.54: MCPServerRegistration metadata.namespace from target_namespace parameter
func TestMCPGateway_Namespace(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	out, err := e.Export(context.Background(), mcpInput())
	require.NoError(t, err)

	assert.Equal(t, "mcp-system", out.Artifacts[0].Namespace)
	assert.Contains(t, out.Artifacts[0].YAML, "  namespace: mcp-system")
}

// T-34.55: MCPServerRegistration spec.prefix = server_name + "_"
func TestMCPGateway_Prefix(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	out, err := e.Export(context.Background(), mcpInput())
	require.NoError(t, err)

	assert.Contains(t, out.Artifacts[0].YAML, "  prefix: github_")
	assert.Contains(t, out.Artifacts[1].YAML, "  prefix: jira_")
}

// T-34.56: MCPServerRegistration spec.targetRef.name from route_name attribute
func TestMCPGateway_TargetRef(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	out, err := e.Export(context.Background(), mcpInput())
	require.NoError(t, err)

	assert.Contains(t, out.Artifacts[0].YAML, "    name: github-mcp-route")
}

// T-34.57: MCPServerRegistration spec.path from mcp_path attribute, defaults to /mcp
func TestMCPGateway_PathDefault(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	out, err := e.Export(context.Background(), mcpInput())
	require.NoError(t, err)

	assert.Contains(t, out.Artifacts[0].YAML, "  path: /mcp")
	assert.Contains(t, out.Artifacts[1].YAML, "  path: /mcp")
}

// T-34.58: MCPServerRegistration spec.credentialRef from credential_secret attribute
func TestMCPGateway_CredentialRef(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	out, err := e.Export(context.Background(), mcpInput())
	require.NoError(t, err)

	assert.Contains(t, out.Artifacts[0].YAML, "  credentialRef:")
	assert.Contains(t, out.Artifacts[0].YAML, "    name: github-token")
	assert.NotContains(t, out.Artifacts[1].YAML, "credentialRef")
}

// T-34.59: Labels set correctly
func TestMCPGateway_Labels(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	out, err := e.Export(context.Background(), mcpInput())
	require.NoError(t, err)

	for _, a := range out.Artifacts {
		assert.Contains(t, a.YAML, "    assethub.io/catalog: prod-agents")
		assert.Contains(t, a.YAML, "    assethub.io/exporter: mcp-gateway")
	}
}

// T-34.60: Annotations set correctly
func TestMCPGateway_Annotations(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	out, err := e.Export(context.Background(), mcpInput())
	require.NoError(t, err)

	for _, a := range out.Artifacts {
		assert.Contains(t, a.YAML, "    assethub.io/exported-at:")
	}
}

// T-34.61: MCPVirtualServer produced once per export run
// Without VS instance filtering: one VS CR named after catalog (backward compatible)
// With VS instance filtering: tested in T-34.75d/e
func TestMCPGateway_OneVirtualServer(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	out, err := e.Export(context.Background(), mcpInput())
	require.NoError(t, err)

	vsCRs := 0
	for _, a := range out.Artifacts {
		if a.Kind == "MCPVirtualServer" {
			vsCRs++
		}
	}
	assert.Equal(t, 1, vsCRs)
}

// T-34.62: MCPVirtualServer metadata.name — defaults to catalog name when no VS instance specified
// With VS instance: name matches instance (tested in T-34.75e)
func TestMCPGateway_VSName(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	out, err := e.Export(context.Background(), mcpInput())
	require.NoError(t, err)

	vs := out.Artifacts[len(out.Artifacts)-1]
	assert.Equal(t, "prod-agents", vs.Name)
}

// T-34.63: MCPVirtualServer spec.description matches catalog description
func TestMCPGateway_VSDescription(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	out, err := e.Export(context.Background(), mcpInput())
	require.NoError(t, err)

	vs := out.Artifacts[len(out.Artifacts)-1]
	var parsed map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(vs.YAML), &parsed))
	spec := parsed["spec"].(map[string]any)
	assert.Equal(t, "Production AI agents", spec["description"])
}

// T-34.64: MCPVirtualServer spec.tools — without filtering, contains all tools
// With VS instance filtering: only associated tools (tested in T-34.75d)
func TestMCPGateway_VSTools(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	out, err := e.Export(context.Background(), mcpInput())
	require.NoError(t, err)

	vs := out.Artifacts[len(out.Artifacts)-1]
	assert.Contains(t, vs.YAML, "    - github_create-issue")
	assert.Contains(t, vs.YAML, "    - github_list-repos")
	assert.Contains(t, vs.YAML, "    - jira_search-issues")
}

// T-34.65: Tool name format: {server_name}_{tool_name}
func TestMCPGateway_ToolNameFormat(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	out, err := e.Export(context.Background(), mcpInput())
	require.NoError(t, err)

	vs := out.Artifacts[len(out.Artifacts)-1]
	assert.Contains(t, vs.YAML, "github_list-repos")
	assert.Contains(t, vs.YAML, "jira_search-issues")
}

// T-34.66: Without filtering, tools from multiple servers are all included
// With VS instance filtering: only selected VS's servers/tools (tested in T-34.75f)
func TestMCPGateway_MultiServerTools(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	out, err := e.Export(context.Background(), mcpInput())
	require.NoError(t, err)

	vs := out.Artifacts[len(out.Artifacts)-1]
	assert.Contains(t, vs.YAML, "github_")
	assert.Contains(t, vs.YAML, "jira_")
}

// T-34.67: Deterministic ordering: servers sorted by name
func TestMCPGateway_ServerOrdering(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	out, err := e.Export(context.Background(), mcpInput())
	require.NoError(t, err)

	assert.Equal(t, "github", out.Artifacts[0].Name)
	assert.Equal(t, "jira", out.Artifacts[1].Name)
}

// T-34.68: Deterministic ordering: tools sorted alphabetically
func TestMCPGateway_ToolOrdering(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	out, err := e.Export(context.Background(), mcpInput())
	require.NoError(t, err)

	vs := out.Artifacts[len(out.Artifacts)-1]
	lines := strings.Split(vs.YAML, "\n")
	var tools []string
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "- ") && !strings.Contains(line, ":") {
			tools = append(tools, strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "- ")))
		}
	}
	require.Len(t, tools, 3)
	assert.True(t, sort.StringsAreSorted(tools), "tools should be alphabetically sorted: %v", tools)
}

// T-34.69: VirtualServer CR appears last in output
func TestMCPGateway_VSLast(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	out, err := e.Export(context.Background(), mcpInput())
	require.NoError(t, err)

	last := out.Artifacts[len(out.Artifacts)-1]
	assert.Equal(t, "MCPVirtualServer", last.Kind)
}

// T-34.70: Missing route_name fails entire export
func TestMCPGateway_MissingRouteNameFails(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	input := mcpInput()
	input.InstancesByType["mcp-server"][0].Attributes["route_name"] = ""

	_, err := e.Export(context.Background(), input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "github")
	assert.Contains(t, err.Error(), "route_name")
}

// T-34.71: Server with no contained tools
func TestMCPGateway_ServerNoTools(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	input := export.ExportInput{
		CatalogName: "my-catalog",
		Parameters: map[string]string{
			"server_type": "mcp-server", "tool_type": "mcp-tool",
		},
		InstancesByType: map[string][]*export.ExportInstance{
			"mcp-server": {{
				ID: "s1", EntityType: "mcp-server", Name: "lonely",
				Attributes: map[string]any{"route_name": "lonely-route"},
			}},
		},
		ChildrenOf: map[string][]*export.ExportInstance{},
	}

	out, err := e.Export(context.Background(), input)
	require.NoError(t, err)
	assert.Len(t, out.Artifacts, 2)
	assert.Equal(t, "MCPServerRegistration", out.Artifacts[0].Kind)
	assert.Equal(t, "MCPVirtualServer", out.Artifacts[1].Kind)
}

// T-34.41: Run on empty catalog returns success with empty artifacts list
func TestMCPGateway_EmptyCatalog(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	input := export.ExportInput{
		CatalogName: "empty-catalog",
		Parameters: map[string]string{
			"server_type": "mcp-server", "tool_type": "mcp-tool",
		},
		InstancesByType: map[string][]*export.ExportInstance{},
		ChildrenOf:      map[string][]*export.ExportInstance{},
	}

	out, err := e.Export(context.Background(), input)
	require.NoError(t, err)
	assert.Empty(t, out.Artifacts)
	require.Len(t, out.Warnings, 1)
	assert.Contains(t, out.Warnings[0], "No instances found")
}

// T-34.75: Deterministic: same input produces identical output (excluding timestamps)
func TestMCPGateway_Deterministic(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	input := mcpInput()

	out1, err := e.Export(context.Background(), input)
	require.NoError(t, err)
	out2, err := e.Export(context.Background(), input)
	require.NoError(t, err)

	require.Equal(t, len(out1.Artifacts), len(out2.Artifacts))
	for i := range out1.Artifacts {
		assert.Equal(t, out1.Artifacts[i].Name, out2.Artifacts[i].Name)
		assert.Equal(t, out1.Artifacts[i].Kind, out2.Artifacts[i].Kind)
	}
}

// T-34.73 + T-34.74: Round-trip YAML test — verify YAML parses correctly
func TestMCPGateway_YAMLRoundTrip(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	out, err := e.Export(context.Background(), mcpInput())
	require.NoError(t, err)

	for _, a := range out.Artifacts {
		var parsed map[string]any
		err := yaml.Unmarshal([]byte(a.YAML), &parsed)
		require.NoError(t, err, "YAML for %s should parse", a.Name)
		assert.Equal(t, "mcp.kuadrant.io/v1alpha1", parsed["apiVersion"])
		assert.Equal(t, a.Kind, parsed["kind"])
		meta := parsed["metadata"].(map[string]any)
		assert.Equal(t, a.Name, meta["name"])
	}
}

// B1: YAML injection — special characters in descriptions must be safely escaped
func TestMCPGateway_YAMLInjectionSafe(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	input := export.ExportInput{
		CatalogName: "my-catalog",
		CatalogDesc: `Description with "quotes" and colons: here # and comments`,
		Parameters: map[string]string{
			"server_type": "mcp-server", "tool_type": "mcp-tool", "target_namespace": "default",
		},
		InstancesByType: map[string][]*export.ExportInstance{
			"mcp-server": {{
				ID: "s1", EntityType: "mcp-server", Name: "test-server",
				Attributes: map[string]any{"route_name": "route: with-colon"},
			}},
		},
		ChildrenOf: map[string][]*export.ExportInstance{},
	}

	out, err := e.Export(context.Background(), input)
	require.NoError(t, err)

	for _, a := range out.Artifacts {
		var parsed map[string]any
		err := yaml.Unmarshal([]byte(a.YAML), &parsed)
		require.NoError(t, err, "YAML with special chars should parse: %s", a.Name)
	}
}

// ValidateSchema tests
// T-34.11: ValidateSchema checks required attribute exists on entity type
func TestMCPGateway_ValidateSchema_MissingRouteName(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	schema := export.SchemaInfo{
		EntityTypes: []export.SchemaEntityType{
			{Name: "mcp-server", Attributes: []string{"description"}, Associations: []export.SchemaAssociation{
				{Type: "containment", TargetEntityType: "mcp-tool"},
			}},
			{Name: "mcp-tool", Attributes: []string{"description"}},
		},
	}
	err := e.ValidateSchema(map[string]string{"server_type": "mcp-server", "tool_type": "mcp-tool"}, schema)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "route_name")
}

// T-34.12: ValidateSchema checks containment association exists
func TestMCPGateway_ValidateSchema_NoContainment(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	schema := export.SchemaInfo{
		EntityTypes: []export.SchemaEntityType{
			{Name: "mcp-server", Attributes: []string{"route_name"}, Associations: []export.SchemaAssociation{}},
			{Name: "mcp-tool", Attributes: []string{"description"}},
		},
	}
	err := e.ValidateSchema(map[string]string{"server_type": "mcp-server", "tool_type": "mcp-tool"}, schema)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "containment")
}

// T-34.75d: Export with VS instance: only allowed tools appear in output
func TestMCPGateway_Export_FilteredByVSInstance(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	input := mcpInput()
	input.VirtualServerInstanceName = "my-virtual-server"
	// Only allow t1 and t3 — t2 (create-issue) should be excluded
	input.AllowedToolIDs = map[string]bool{"t1": true, "t3": true}

	out, err := e.Export(context.Background(), input)
	require.NoError(t, err)

	// VirtualServer CR should exist
	var vsCR *export.K8sArtifact
	for i, a := range out.Artifacts {
		if a.Kind == "MCPVirtualServer" {
			vsCR = &out.Artifacts[i]
		}
	}
	require.NotNil(t, vsCR, "MCPVirtualServer CR must be produced")

	// T-34.75e: VirtualServer name should be the VS instance name
	assert.Equal(t, "my-virtual-server", vsCR.Name)

	// Should contain only the allowed tools
	assert.Contains(t, vsCR.YAML, "github_list-repos")
	assert.Contains(t, vsCR.YAML, "jira_search-issues")
	assert.NotContains(t, vsCR.YAML, "github_create-issue")
}

// T-34.75f: Servers not related to selected VS tools are excluded
func TestMCPGateway_Export_FilteredExcludesUnrelatedServers(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	input := mcpInput()
	input.VirtualServerInstanceName = "my-vs"
	// Only allow t3 (from jira server) — github server's tools excluded
	input.AllowedToolIDs = map[string]bool{"t3": true}

	out, err := e.Export(context.Background(), input)
	require.NoError(t, err)

	// Only jira server should produce a ServerRegistration (github has no allowed tools)
	serverCRs := 0
	for _, a := range out.Artifacts {
		if a.Kind == "MCPServerRegistration" {
			serverCRs++
			assert.Equal(t, "jira", a.Name)
		}
	}
	assert.Equal(t, 1, serverCRs)
}

func TestMCPGateway_ValidateSchema_Success(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	schema := export.SchemaInfo{
		EntityTypes: []export.SchemaEntityType{
			{Name: "mcp-server", Attributes: []string{"route_name"}, Associations: []export.SchemaAssociation{
				{Type: "containment", TargetEntityType: "mcp-tool"},
			}},
			{Name: "mcp-tool", Attributes: []string{"description"}},
			{Name: "virtual-server", Associations: []export.SchemaAssociation{
				{Type: "directional", TargetEntityType: "mcp-tool"},
			}},
		},
	}
	err := e.ValidateSchema(map[string]string{"server_type": "mcp-server", "tool_type": "mcp-tool", "virtual_server_type": "virtual-server"}, schema)
	assert.NoError(t, err)
}

// T-34.40: Run binding produces K8sArtifact list with APIVersion/Kind/Name/YAML
func TestMCPGateway_RunProducesK8sArtifactList(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	out, err := e.Export(context.Background(), mcpInput())
	require.NoError(t, err)
	require.NotEmpty(t, out.Artifacts, "artifacts list must be non-empty")

	for _, a := range out.Artifacts {
		assert.NotEmpty(t, a.APIVersion, "artifact %q must have APIVersion", a.Name)
		assert.NotEmpty(t, a.Kind, "artifact %q must have Kind", a.Name)
		assert.NotEmpty(t, a.Name, "artifact must have Name")
		assert.NotEmpty(t, a.YAML, "artifact %q must have YAML", a.Name)

		// Verify YAML parses and contains the declared fields
		var parsed map[string]any
		require.NoError(t, yaml.Unmarshal([]byte(a.YAML), &parsed), "YAML for %q must parse", a.Name)
		assert.Equal(t, a.APIVersion, parsed["apiVersion"])
		assert.Equal(t, a.Kind, parsed["kind"])
		meta := parsed["metadata"].(map[string]any)
		assert.Equal(t, a.Name, meta["name"])
	}
}

// T-34.42: Run with missing required attributes fails with descriptive error
// Input where a server has NO route_name attribute at all (key absent from map).
func TestMCPGateway_MissingRouteNameKeyFails(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	input := export.ExportInput{
		CatalogName: "test-catalog",
		Parameters: map[string]string{
			"server_type":      "mcp-server",
			"tool_type":        "mcp-tool",
			"target_namespace": "default",
		},
		InstancesByType: map[string][]*export.ExportInstance{
			"mcp-server": {
				{
					ID:         "s1",
					EntityType: "mcp-server",
					Name:       "broken-server",
					// route_name key is entirely absent from Attributes
					Attributes: map[string]any{
						"mcp_path": "/mcp",
					},
				},
			},
		},
		ChildrenOf: map[string][]*export.ExportInstance{},
	}

	_, err := e.Export(context.Background(), input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "route_name", "error should mention the missing attribute")
	assert.Contains(t, err.Error(), "broken-server", "error should mention the server name")
}

// T-34.43: Note — schema re-validation on publish is tested in publish_service_test.go
// (TestPublishPreview_RevalidatesSchema). The PublishPreview path calls
// ValidateSchema before Export, catching schema drift since binding creation.
