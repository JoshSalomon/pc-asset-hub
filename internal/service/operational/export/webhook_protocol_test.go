package export_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/project-catalyst/pc-asset-hub/internal/service/operational/export"
)

// T-36.13: WebhookValidateRequest JSON roundtrip — snake_case field names
func TestWebhookValidateRequest_JSON_SnakeCase(t *testing.T) {
	req := export.WebhookValidateRequest{
		Parameters: map[string]string{"server_type": "mcp-server"},
		Schema: export.WebhookSchemaInfo{
			EntityTypes: []export.WebhookSchemaEntityType{
				{
					Name:       "mcp-server",
					Attributes: []string{"endpoint"},
					Associations: []export.WebhookSchemaAssociation{
						{Name: "tools", Type: "containment", TargetEntityType: "mcp-tool"},
					},
				},
			},
		},
	}
	data, err := json.Marshal(req)
	require.NoError(t, err)

	raw := string(data)
	assert.Contains(t, raw, `"parameters"`)
	assert.Contains(t, raw, `"schema"`)
	assert.Contains(t, raw, `"entity_types"`)
	assert.Contains(t, raw, `"target_entity_type"`)
	assert.NotContains(t, raw, `"Parameters"`)
	assert.NotContains(t, raw, `"Schema"`)

	var decoded export.WebhookValidateRequest
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, req, decoded)
}

// T-36.14: WebhookExportRequest JSON roundtrip — snake_case field names
func TestWebhookExportRequest_JSON_SnakeCase(t *testing.T) {
	req := export.WebhookExportRequest{
		CatalogName:        "prod-agents",
		CatalogDescription: "Production AI agents",
		Parameters:         map[string]string{"server_type": "mcp-server"},
		InstancesByType: map[string][]export.WebhookInstance{
			"mcp-server": {
				{ID: "id1", Name: "github", Description: "GitHub MCP", Attributes: map[string]any{"endpoint": "https://example.com"}, ParentID: ""},
			},
		},
		ChildrenOf: map[string][]string{"id1": {"id2"}},
	}
	data, err := json.Marshal(req)
	require.NoError(t, err)

	raw := string(data)
	assert.Contains(t, raw, `"catalog_name"`)
	assert.Contains(t, raw, `"catalog_description"`)
	assert.Contains(t, raw, `"instances_by_type"`)
	assert.Contains(t, raw, `"children_of"`)
	assert.Contains(t, raw, `"parent_id"`)
	assert.NotContains(t, raw, `"CatalogName"`)

	var decoded export.WebhookExportRequest
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, req.CatalogName, decoded.CatalogName)
	assert.Equal(t, req.CatalogDescription, decoded.CatalogDescription)
}

// T-36.15: WebhookArtifact marshals to api_version/kind/name/namespace/yaml
func TestWebhookArtifact_JSON_SnakeCase(t *testing.T) {
	a := export.WebhookArtifact{
		APIVersion: "gateway.assethub.io/v1alpha1",
		Kind:       "MCPServerRegistration",
		Name:       "github",
		Namespace:  "mcp-gateway",
		YAML:       "apiVersion: v1\nkind: test\n",
	}
	data, err := json.Marshal(a)
	require.NoError(t, err)

	raw := string(data)
	assert.Contains(t, raw, `"api_version"`)
	assert.Contains(t, raw, `"kind"`)
	assert.Contains(t, raw, `"name"`)
	assert.Contains(t, raw, `"namespace"`)
	assert.Contains(t, raw, `"yaml"`)
	assert.NotContains(t, raw, `"APIVersion"`)
	assert.NotContains(t, raw, `"YAML"`)
}

// ParameterDef with AttributeMappings serializes correctly
func TestParameterDef_AttributeMappings_JSON(t *testing.T) {
	pd := export.ParameterDef{
		Name:     "server_type",
		Type:     "entity_type",
		Required: true,
		AttributeMappings: []export.AttributeMapping{
			{Name: "route_name_attr", Description: "HTTPRoute name", Required: true, Default: "route_name"},
			{Name: "mcp_path_attr", Description: "MCP path", Default: "mcp_path"},
		},
	}

	data, err := json.Marshal(pd)
	require.NoError(t, err)
	raw := string(data)
	assert.Contains(t, raw, `"attribute_mappings"`)
	assert.Contains(t, raw, `"route_name_attr"`)
	assert.Contains(t, raw, `"mcp_path_attr"`)

	var parsed export.ParameterDef
	require.NoError(t, json.Unmarshal(data, &parsed))
	require.Len(t, parsed.AttributeMappings, 2)
	assert.Equal(t, "route_name_attr", parsed.AttributeMappings[0].Name)
	assert.True(t, parsed.AttributeMappings[0].Required)
	assert.Equal(t, "route_name", parsed.AttributeMappings[0].Default)
	assert.Equal(t, "mcp_path_attr", parsed.AttributeMappings[1].Name)
	assert.False(t, parsed.AttributeMappings[1].Required)
}

// ParameterDef without AttributeMappings omits the field
func TestParameterDef_NoAttributeMappings_OmitsField(t *testing.T) {
	pd := export.ParameterDef{Name: "target_namespace", Type: "string"}
	data, err := json.Marshal(pd)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "attribute_mappings")
}

// T-36.16: ExportInputToWebhookRequest excludes dead fields, includes VS instance name,
// and correctly serializes multi-type, multi-instance data with links
func TestExportInputToWebhookRequest_ExcludesDeadFields(t *testing.T) {
	serverInst := &export.ExportInstance{
		ID: "s1", EntityType: "Server", Name: "alpha",
		Description: "Alpha server",
		Attributes:  map[string]any{"route_name": "route-a", "port": 8080, "enabled": true},
		LinksByAssoc: map[string][]export.ExportLink{
			"tools": {{TargetInstanceID: "t1", TargetInstanceName: "tool-a", TargetEntityType: "Tool"}},
		},
	}
	toolInst := &export.ExportInstance{
		ID: "t1", EntityType: "Tool", Name: "tool-a", ParentID: "s1",
		Attributes: map[string]any{"type": "read", "tags": []string{"safe", "public"}},
	}

	input := export.ExportInput{
		CatalogName:               "test",
		CatalogDesc:               "Test catalog",
		CVLabel:                   "dead-field",
		Parameters:                map[string]string{"key": "val"},
		EntityTypes:               []export.ExportEntityType{{Name: "dead-field"}},
		InstancesByType: map[string][]*export.ExportInstance{
			"Server": {serverInst},
			"Tool":   {toolInst},
		},
		ChildrenOf: map[string][]*export.ExportInstance{
			"s1": {toolInst},
		},
		VirtualServerInstanceName: "my-vs-instance",
		AllowedToolIDs:            map[string]bool{"id1": true},
	}

	req := export.ExportInputToWebhookRequest(input)

	assert.Equal(t, "test", req.CatalogName)
	assert.Equal(t, "Test catalog", req.CatalogDescription)
	assert.Equal(t, map[string]string{"key": "val"}, req.Parameters)
	assert.Equal(t, "my-vs-instance", req.VirtualServerInstanceName)

	// Verify dead fields excluded from JSON
	data, err := json.Marshal(req)
	require.NoError(t, err)
	raw := string(data)
	assert.NotContains(t, raw, "dead-field")
	assert.NotContains(t, raw, "cv_label")
	assert.NotContains(t, raw, "entity_types")
	assert.NotContains(t, raw, "allowed_tool_ids")
	assert.Contains(t, raw, "virtual_server_instance_name")
	assert.Contains(t, raw, "my-vs-instance")

	// Verify live data serialized with full depth
	assert.Len(t, req.InstancesByType, 2)
	assert.Len(t, req.InstancesByType["Server"], 1)
	assert.Len(t, req.InstancesByType["Tool"], 1)
	assert.Equal(t, "alpha", req.InstancesByType["Server"][0].Name)
	assert.Equal(t, "route-a", req.InstancesByType["Server"][0].Attributes["route_name"])
	assert.Equal(t, 8080, req.InstancesByType["Server"][0].Attributes["port"])
	assert.Equal(t, true, req.InstancesByType["Server"][0].Attributes["enabled"])
	assert.Len(t, req.InstancesByType["Server"][0].Links, 1)
	assert.Equal(t, "tool-a", req.InstancesByType["Server"][0].Links["tools"][0].TargetInstanceName)
	assert.Equal(t, "s1", req.InstancesByType["Tool"][0].ParentID)
	assert.Len(t, req.ChildrenOf["s1"], 1)
	assert.Equal(t, "t1", req.ChildrenOf["s1"][0])
}

// T-36.17: Mixed attribute types (string, int, bool, array) roundtrip correctly
func TestWebhookInstance_MixedAttributeTypes(t *testing.T) {
	inst := export.WebhookInstance{
		ID:          "id1",
		Name:        "test",
		Description: "desc",
		Attributes: map[string]any{
			"str_attr":  "hello",
			"int_attr":  float64(42),
			"bool_attr": true,
			"arr_attr":  []any{"a", "b"},
		},
		ParentID: "",
	}

	data, err := json.Marshal(inst)
	require.NoError(t, err)

	var decoded export.WebhookInstance
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, "hello", decoded.Attributes["str_attr"])
	assert.Equal(t, float64(42), decoded.Attributes["int_attr"])
	assert.Equal(t, true, decoded.Attributes["bool_attr"])
	assert.IsType(t, []any{}, decoded.Attributes["arr_attr"])
}

// T-36.18: SchemaInfoToWebhook converts associations with all fields
func TestSchemaInfoToWebhook(t *testing.T) {
	schema := export.SchemaInfo{
		EntityTypes: []export.SchemaEntityType{
			{
				Name:       "mcp-server",
				Attributes: []string{"endpoint", "route_name"},
				Associations: []export.SchemaAssociation{
					{Name: "tools", Type: "containment", TargetEntityType: "mcp-tool"},
				},
			},
			{
				Name:       "mcp-tool",
				Attributes: []string{"type"},
			},
		},
	}

	result := export.SchemaInfoToWebhook(schema)
	require.Len(t, result.EntityTypes, 2)

	server := result.EntityTypes[0]
	assert.Equal(t, "mcp-server", server.Name)
	assert.Equal(t, []string{"endpoint", "route_name"}, server.Attributes)
	require.Len(t, server.Associations, 1)
	assert.Equal(t, "tools", server.Associations[0].Name)
	assert.Equal(t, "containment", server.Associations[0].Type)
	assert.Equal(t, "mcp-tool", server.Associations[0].TargetEntityType)

	tool := result.EntityTypes[1]
	assert.Equal(t, "mcp-tool", tool.Name)
	assert.Empty(t, tool.Associations)
}

// T-36.19: WebhookExportResponse JSON roundtrip
func TestWebhookExportResponse_JSON_Roundtrip(t *testing.T) {
	resp := export.WebhookExportResponse{
		Artifacts: []export.WebhookArtifact{
			{APIVersion: "v1", Kind: "ConfigMap", Name: "test", Namespace: "default", YAML: "data: test"},
		},
		Warnings: []string{"warn1"},
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	raw := string(data)
	assert.Contains(t, raw, `"artifacts"`)
	assert.Contains(t, raw, `"warnings"`)

	var decoded export.WebhookExportResponse
	require.NoError(t, json.Unmarshal(data, &decoded))
	require.Len(t, decoded.Artifacts, 1)
	assert.Equal(t, "ConfigMap", decoded.Artifacts[0].Kind)
	assert.Equal(t, []string{"warn1"}, decoded.Warnings)
}

// T-36.20: WebhookExportResponseToOutput converts artifacts correctly
func TestWebhookExportResponseToOutput(t *testing.T) {
	resp := export.WebhookExportResponse{
		Artifacts: []export.WebhookArtifact{
			{APIVersion: "v1", Kind: "ConfigMap", Name: "test", Namespace: "default", YAML: "data: test"},
		},
		Warnings: []string{"warn1"},
	}

	output := export.WebhookExportResponseToOutput(resp)
	require.Len(t, output.Artifacts, 1)
	assert.Equal(t, "v1", output.Artifacts[0].APIVersion)
	assert.Equal(t, "ConfigMap", output.Artifacts[0].Kind)
	assert.Equal(t, "test", output.Artifacts[0].Name)
	assert.Equal(t, "default", output.Artifacts[0].Namespace)
	assert.Equal(t, "data: test", output.Artifacts[0].YAML)
	assert.Equal(t, []string{"warn1"}, output.Warnings)
}
