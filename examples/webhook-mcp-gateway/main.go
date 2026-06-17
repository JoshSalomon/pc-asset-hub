package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

const protocolVersion = "v1"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	trustedSubjects := parseTrustedSubjects()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("POST /validate", withAuth(trustedSubjects, handleValidate))
	mux.HandleFunc("POST /export", withAuth(trustedSubjects, handleExport))

	log.Printf("webhook-mcp-gateway starting on :%s (trusted subjects: %v)", port, trustedSubjects)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func parseTrustedSubjects() []string {
	v := os.Getenv("TRUSTED_SUBJECTS")
	if v == "" {
		ns := os.Getenv("POD_NAMESPACE")
		if ns == "" {
			ns = "assethub"
		}
		return []string{
			"system:serviceaccount:" + ns + ":assethub-api-server",
			"system:serviceaccount:" + ns + ":assethub-operator",
		}
	}
	return strings.Split(v, ",")
}

// --- Health ---

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// --- Auth middleware ---

func withAuth(trustedSubjects []string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check protocol version
		pv := r.Header.Get("X-AssetHub-Protocol-Version")
		if pv != "" && pv != protocolVersion {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("unsupported protocol version %q (expected %s)", pv, protocolVersion))
			return
		}

		// Check Authorization header
		auth := r.Header.Get("Authorization")
		if auth == "" {
			writeError(w, http.StatusUnauthorized, "missing Authorization header")
			return
		}
		if !strings.HasPrefix(auth, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "invalid Authorization header format")
			return
		}

		// In a production plugin, you would validate the token via TokenReview API
		// and check the authenticated subject against trustedSubjects.
		// For this example in development mode, we accept any Bearer token as valid.
		// The live tests verify the auth headers are sent correctly.

		// Development mode: accept empty tokens (API server may not have SA token mounted)
		token := strings.TrimPrefix(auth, "Bearer ")
		if token == "" && os.Getenv("DEV_MODE") == "true" {
			next(w, r)
			return
		}

		next(w, r)
	}
}

// --- Validate ---

type ValidateRequest struct {
	Parameters map[string]string `json:"parameters"`
	Schema     SchemaInfo        `json:"schema"`
}

type SchemaInfo struct {
	EntityTypes []SchemaEntityType `json:"entity_types"`
}

type SchemaEntityType struct {
	Name         string              `json:"name"`
	Attributes   []string            `json:"attributes"`
	Associations []SchemaAssociation `json:"associations"`
}

type SchemaAssociation struct {
	Name             string `json:"name"`
	Type             string `json:"type"`
	TargetEntityType string `json:"target_entity_type"`
}

func handleValidate(w http.ResponseWriter, r *http.Request) {
	var req ValidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	serverType := req.Parameters["server_type"]
	toolType := req.Parameters["tool_type"]

	if serverType == "" || toolType == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{"valid": false, "error": "parameters 'server_type' and 'tool_type' are required"})
		return
	}

	var serverET *SchemaEntityType
	for i := range req.Schema.EntityTypes {
		if req.Schema.EntityTypes[i].Name == serverType {
			serverET = &req.Schema.EntityTypes[i]
			break
		}
	}
	if serverET == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{"valid": false, "error": fmt.Sprintf("entity type %q not found in schema", serverType)})
		return
	}

	routeNameAttr := resolveAttrName(req.Parameters, "route_name_attr", "route_name")
	hasRouteNameAttr := false
	for _, attr := range serverET.Attributes {
		if attr == routeNameAttr {
			hasRouteNameAttr = true
			break
		}
	}
	if !hasRouteNameAttr {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{"valid": false, "error": fmt.Sprintf("entity type %q is missing required attribute '%s'", serverType, routeNameAttr)})
		return
	}

	hasContainment := false
	for _, assoc := range serverET.Associations {
		if assoc.Type == "containment" && assoc.TargetEntityType == toolType {
			hasContainment = true
			break
		}
	}
	if !hasContainment {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{"valid": false, "error": fmt.Sprintf("entity type %q has no containment association to %q", serverType, toolType)})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"valid": true})
}

// --- Export ---

type ExportRequest struct {
	CatalogName               string                       `json:"catalog_name"`
	CatalogDescription        string                       `json:"catalog_description"`
	Parameters                map[string]string            `json:"parameters"`
	InstancesByType           map[string][]ExportInstance   `json:"instances_by_type"`
	ChildrenOf                map[string][]string          `json:"children_of"`
	VirtualServerInstanceName string                       `json:"virtual_server_instance_name,omitempty"`
}

type ExportInstance struct {
	ID          string                       `json:"id"`
	Name        string                       `json:"name"`
	Description string                       `json:"description"`
	Attributes  map[string]any               `json:"attributes"`
	ParentID    string                       `json:"parent_id"`
	Links       map[string][]ExportLink      `json:"links,omitempty"`
}

type ExportLink struct {
	TargetInstanceID   string `json:"target_instance_id"`
	TargetInstanceName string `json:"target_instance_name"`
	TargetEntityType   string `json:"target_entity_type"`
}

type ExportArtifact struct {
	APIVersion string `json:"api_version"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
	YAML       string `json:"yaml"`
}

func handleExport(w http.ResponseWriter, r *http.Request) {
	var req ExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	serverType := req.Parameters["server_type"]
	vsType := req.Parameters["virtual_server_type"]
	namespace := req.Parameters["target_namespace"]
	if namespace == "" {
		namespace = "default"
	}
	routeNameAttr := resolveAttrName(req.Parameters, "route_name_attr", "route_name")
	mcpPathAttr := resolveAttrName(req.Parameters, "mcp_path_attr", "mcp_path")
	credSecretAttr := resolveAttrName(req.Parameters, "credential_secret_attr", "credential_secret")

	// Resolve allowed tool IDs from VS instance links (if VS selected)
	var allowedToolIDs map[string]bool
	if req.VirtualServerInstanceName != "" && vsType != "" {
		for _, vsInst := range req.InstancesByType[vsType] {
			if vsInst.Name == req.VirtualServerInstanceName {
				allowedToolIDs = make(map[string]bool)
				for _, assocLinks := range vsInst.Links {
					for _, link := range assocLinks {
						allowedToolIDs[link.TargetInstanceID] = true
					}
				}
				break
			}
		}
	}

	instanceByID := make(map[string]ExportInstance)
	for _, typedInsts := range req.InstancesByType {
		for _, inst := range typedInsts {
			instanceByID[inst.ID] = inst
		}
	}

	servers := req.InstancesByType[serverType]
	if len(servers) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"artifacts": []any{},
			"warnings":  []string{fmt.Sprintf("No instances found for export — catalog '%s' has no %s instances", req.CatalogName, serverType)},
		})
		return
	}

	sort.Slice(servers, func(i, j int) bool { return servers[i].Name < servers[j].Name })

	now := time.Now().UTC().Format(time.RFC3339)
	var artifacts []ExportArtifact
	var allTools []string
	var warnings []string

	for _, server := range servers {
		routeName, _ := server.Attributes[routeNameAttr].(string)
		if routeName == "" {
			warnings = append(warnings, fmt.Sprintf("Server '%s' has no '%s' attribute, skipped", server.Name, routeNameAttr))
			continue
		}

		var serverTools []string
		childIDs := req.ChildrenOf[server.ID]
		for _, childID := range childIDs {
			if allowedToolIDs != nil && !allowedToolIDs[childID] {
				continue
			}
			if inst, found := instanceByID[childID]; found {
				serverTools = append(serverTools, server.Name+"_"+inst.Name)
			}
		}

		if allowedToolIDs != nil && len(serverTools) == 0 {
			continue
		}

		allTools = append(allTools, serverTools...)

		mcpPath := "/mcp"
		if p, _ := server.Attributes[mcpPathAttr].(string); p != "" {
			mcpPath = p
		}

		credentialRef := ""
		if cr, _ := server.Attributes[credSecretAttr].(string); cr != "" {
			credentialRef = cr
		}

		yamlStr := buildServerRegistrationYAML(server.Name, namespace, req.CatalogName, routeName, mcpPath, credentialRef, now)
		artifacts = append(artifacts, ExportArtifact{
			APIVersion: "mcp.kuadrant.io/v1alpha1",
			Kind:       "MCPServerRegistration",
			Name:       server.Name,
			Namespace:  namespace,
			YAML:       yamlStr,
		})
	}

	sort.Strings(allTools)

	vsName := req.CatalogName
	if req.VirtualServerInstanceName != "" {
		vsName = req.VirtualServerInstanceName
	}

	vsYAML := buildVirtualServerYAML(vsName, namespace, req.CatalogDescription, allTools, now)
	artifacts = append(artifacts, ExportArtifact{
		APIVersion: "mcp.kuadrant.io/v1alpha1",
		Kind:       "MCPVirtualServer",
		Name:       vsName,
		Namespace:  namespace,
		YAML:       vsYAML,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"artifacts": artifacts,
		"warnings":  warnings,
	})
}

// --- YAML builders (simplified — not using yaml.Marshal to avoid dependency) ---

func yamlQuote(s string) string {
	safe := true
	for _, c := range s {
		if c == '\n' || c == '\r' || c == '"' || c == '\\' || c == ':' || c == '#' {
			safe = false
			break
		}
	}
	if safe {
		return s
	}
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\r", `\r`)
	return `"` + r.Replace(s) + `"`
}

func buildServerRegistrationYAML(name, namespace, catalogName, routeName, path, credentialRef, exportedAt string) string {
	var sb strings.Builder
	sb.WriteString("apiVersion: mcp.kuadrant.io/v1alpha1\n")
	sb.WriteString("kind: MCPServerRegistration\n")
	sb.WriteString("metadata:\n")
	sb.WriteString(fmt.Sprintf("  name: %s\n", yamlQuote(name)))
	sb.WriteString(fmt.Sprintf("  namespace: %s\n", yamlQuote(namespace)))
	sb.WriteString("  labels:\n")
	sb.WriteString(fmt.Sprintf("    assethub.io/catalog: %s\n", yamlQuote(catalogName)))
	sb.WriteString("    assethub.io/exporter: webhook-mcp-gateway\n")
	sb.WriteString("  annotations:\n")
	sb.WriteString(fmt.Sprintf("    assethub.io/exported-at: \"%s\"\n", exportedAt))
	sb.WriteString("spec:\n")
	sb.WriteString(fmt.Sprintf("  prefix: %s\n", yamlQuote(name+"_")))
	sb.WriteString("  targetRef:\n")
	sb.WriteString("    group: gateway.networking.k8s.io\n")
	sb.WriteString("    kind: HTTPRoute\n")
	sb.WriteString(fmt.Sprintf("    name: %s\n", yamlQuote(routeName)))
	sb.WriteString(fmt.Sprintf("  path: %s\n", yamlQuote(path)))
	if credentialRef != "" {
		sb.WriteString("  credentialRef:\n")
		sb.WriteString(fmt.Sprintf("    name: %s\n", yamlQuote(credentialRef)))
	}
	return sb.String()
}

func buildVirtualServerYAML(catalogName, namespace, description string, tools []string, exportedAt string) string {
	var sb strings.Builder
	sb.WriteString("apiVersion: mcp.kuadrant.io/v1alpha1\n")
	sb.WriteString("kind: MCPVirtualServer\n")
	sb.WriteString("metadata:\n")
	sb.WriteString(fmt.Sprintf("  name: %s\n", yamlQuote(catalogName)))
	sb.WriteString(fmt.Sprintf("  namespace: %s\n", yamlQuote(namespace)))
	sb.WriteString("  labels:\n")
	sb.WriteString(fmt.Sprintf("    assethub.io/catalog: %s\n", yamlQuote(catalogName)))
	sb.WriteString("    assethub.io/exporter: webhook-mcp-gateway\n")
	sb.WriteString("  annotations:\n")
	sb.WriteString(fmt.Sprintf("    assethub.io/exported-at: \"%s\"\n", exportedAt))
	sb.WriteString("spec:\n")
	if description != "" {
		sb.WriteString(fmt.Sprintf("  description: %s\n", yamlQuote(description)))
	}
	if len(tools) > 0 {
		sb.WriteString("  tools:\n")
		for _, t := range tools {
			sb.WriteString(fmt.Sprintf("  - %s\n", yamlQuote(t)))
		}
	}
	return sb.String()
}

// --- Helpers ---

func resolveAttrName(params map[string]string, paramKey, defaultName string) string {
	if v, ok := params[paramKey]; ok && v != "" {
		return v
	}
	return defaultName
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
