package export

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"gopkg.in/yaml.v3"
)

type MCPGatewayExporter struct{}

func NewMCPGatewayExporter() *MCPGatewayExporter {
	return &MCPGatewayExporter{}
}

func (e *MCPGatewayExporter) Name() string       { return "mcp-gateway" }
func (e *MCPGatewayExporter) Description() string { return "Exports MCP server/tool instances as MCP Gateway CRs" }

func (e *MCPGatewayExporter) ParameterSchema() []ParameterDef {
	return []ParameterDef{
		{Name: "server_type", Type: "entity_type", Required: true, Description: "Entity type name for MCP servers"},
		{Name: "tool_type", Type: "entity_type", Required: true, Description: "Entity type name for MCP tools"},
		{Name: "virtual_server_type", Type: "entity_type", Required: true, Description: "Entity type name for MCP virtual servers"},
		{Name: "target_namespace", Type: "string", Description: "K8s namespace for output CRs", Default: "default"},
	}
}

func (e *MCPGatewayExporter) ValidateSchema(params map[string]string, schema SchemaInfo) error {
	serverType := params["server_type"]
	toolType := params["tool_type"]

	var serverET *SchemaEntityType
	for i := range schema.EntityTypes {
		if schema.EntityTypes[i].Name == serverType {
			serverET = &schema.EntityTypes[i]
			break
		}
	}
	if serverET == nil {
		return domainerrors.NewValidation(fmt.Sprintf("entity type %q not found in catalog version", serverType))
	}

	hasRouteNameAttr := false
	for _, attr := range serverET.Attributes {
		if attr == "route_name" {
			hasRouteNameAttr = true
			break
		}
	}
	if !hasRouteNameAttr {
		return domainerrors.NewValidation(fmt.Sprintf("entity type %q is missing required attribute 'route_name'", serverType))
	}

	hasContainment := false
	for _, assoc := range serverET.Associations {
		if assoc.Type == "containment" {
			targetName := e.resolveTargetName(assoc.TargetEntityType, schema)
			if targetName == toolType {
				hasContainment = true
				break
			}
		}
	}
	if !hasContainment {
		return domainerrors.NewValidation(fmt.Sprintf("entity type %q has no containment association to %q", serverType, toolType))
	}

	vsType := params["virtual_server_type"]
	var vsET *SchemaEntityType
	for i := range schema.EntityTypes {
		if schema.EntityTypes[i].Name == vsType {
			vsET = &schema.EntityTypes[i]
			break
		}
	}
	if vsET == nil {
		return domainerrors.NewValidation(fmt.Sprintf("entity type %q not found in catalog version", vsType))
	}

	hasToolAssoc := false
	for _, assoc := range vsET.Associations {
		targetName := e.resolveTargetName(assoc.TargetEntityType, schema)
		if targetName == toolType {
			hasToolAssoc = true
			break
		}
	}
	if !hasToolAssoc {
		return domainerrors.NewValidation(fmt.Sprintf("entity type %q has no association to tool type %q", vsType, toolType))
	}

	return nil
}

func (e *MCPGatewayExporter) resolveTargetName(targetID string, schema SchemaInfo) string {
	for _, et := range schema.EntityTypes {
		if et.Name == targetID {
			return et.Name
		}
	}
	return targetID
}

func (e *MCPGatewayExporter) Export(ctx context.Context, input ExportInput) (*ExportOutput, error) {
	serverType := input.Parameters["server_type"]
	toolType := input.Parameters["tool_type"]
	namespace := input.Parameters["target_namespace"]
	if namespace == "" {
		namespace = "default"
	}

	servers := input.InstancesByType[serverType]
	if len(servers) == 0 {
		return &ExportOutput{
			Artifacts: nil,
			Warnings:  []string{fmt.Sprintf("No instances found for export — catalog '%s' has no %s instances", input.CatalogName, serverType)},
		}, nil
	}

	sort.Slice(servers, func(i, j int) bool { return servers[i].Name < servers[j].Name })

	now := time.Now().UTC().Format(time.RFC3339)
	var artifacts []K8sArtifact
	var allTools []string
	var missingRouteName []string

	for _, server := range servers {
		routeName, ok := server.Attributes["route_name"].(string)
		if !ok || routeName == "" {
			missingRouteName = append(missingRouteName, server.Name)
			continue
		}

		// Collect tools for this server, filtering by AllowedToolIDs if set
		var serverTools []string
		children := input.ChildrenOf[server.ID]
		for _, child := range children {
			if child.EntityType == toolType {
				if input.AllowedToolIDs != nil && !input.AllowedToolIDs[child.ID] {
					continue
				}
				serverTools = append(serverTools, server.Name+"_"+child.Name)
			}
		}

		// Skip server entirely if filtering is active and no tools match
		if input.AllowedToolIDs != nil && len(serverTools) == 0 {
			continue
		}

		allTools = append(allTools, serverTools...)

		mcpPath := "/mcp"
		if p, ok := server.Attributes["mcp_path"].(string); ok && p != "" {
			mcpPath = p
		}

		credentialRef := ""
		if cr, ok := server.Attributes["credential_secret"].(string); ok && cr != "" {
			credentialRef = cr
		}

		cr := buildServerRegistration(server.Name, namespace, input.CatalogName, routeName, mcpPath, credentialRef, now)
		yamlBytes, err := yaml.Marshal(cr)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal MCPServerRegistration %q: %w", server.Name, err)
		}
		artifacts = append(artifacts, K8sArtifact{
			APIVersion: "mcp.kuadrant.io/v1alpha1",
			Kind:       "MCPServerRegistration",
			Name:       server.Name,
			Namespace:  namespace,
			YAML:       string(yamlBytes),
		})
	}

	if len(missingRouteName) > 0 {
		return nil, domainerrors.NewValidation(fmt.Sprintf("export failed: instances missing required attribute 'route_name': %s", strings.Join(missingRouteName, ", ")))
	}

	sort.Strings(allTools)

	vsName := input.CatalogName
	if input.VirtualServerInstanceName != "" {
		vsName = input.VirtualServerInstanceName
	}

	vs := buildVirtualServer(vsName, namespace, input.CatalogDesc, allTools, now)
	vsBytes, err := yaml.Marshal(vs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal MCPVirtualServer: %w", err)
	}
	artifacts = append(artifacts, K8sArtifact{
		APIVersion: "mcp.kuadrant.io/v1alpha1",
		Kind:       "MCPVirtualServer",
		Name:       vsName,
		Namespace:  namespace,
		YAML:       string(vsBytes),
	})

	return &ExportOutput{Artifacts: artifacts}, nil
}

type k8sResource struct {
	APIVersion string            `yaml:"apiVersion"`
	Kind       string            `yaml:"kind"`
	Metadata   k8sMetadata       `yaml:"metadata"`
	Spec       map[string]any    `yaml:"spec"`
}

type k8sMetadata struct {
	Name        string            `yaml:"name"`
	Namespace   string            `yaml:"namespace"`
	Labels      map[string]string `yaml:"labels"`
	Annotations map[string]string `yaml:"annotations"`
}

func buildServerRegistration(name, namespace, catalogName, routeName, path, credentialRef, exportedAt string) k8sResource {
	spec := map[string]any{
		"prefix": name + "_",
		"targetRef": map[string]string{
			"group": "gateway.networking.k8s.io",
			"kind":  "HTTPRoute",
			"name":  routeName,
		},
		"path": path,
	}
	if credentialRef != "" {
		spec["credentialRef"] = map[string]string{"name": credentialRef}
	}

	return k8sResource{
		APIVersion: "mcp.kuadrant.io/v1alpha1",
		Kind:       "MCPServerRegistration",
		Metadata: k8sMetadata{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"assethub.io/catalog":  catalogName,
				"assethub.io/exporter": "mcp-gateway",
			},
			Annotations: map[string]string{
				"assethub.io/exported-at": exportedAt,
			},
		},
		Spec: spec,
	}
}

func buildVirtualServer(catalogName, namespace, description string, tools []string, exportedAt string) k8sResource {
	spec := map[string]any{}
	if description != "" {
		spec["description"] = description
	}
	if len(tools) > 0 {
		spec["tools"] = tools
	}

	return k8sResource{
		APIVersion: "mcp.kuadrant.io/v1alpha1",
		Kind:       "MCPVirtualServer",
		Metadata: k8sMetadata{
			Name:      catalogName,
			Namespace: namespace,
			Labels: map[string]string{
				"assethub.io/catalog":  catalogName,
				"assethub.io/exporter": "mcp-gateway",
			},
			Annotations: map[string]string{
				"assethub.io/exported-at": exportedAt,
			},
		},
		Spec: spec,
	}
}
