package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body map[string]string
	json.NewDecoder(rec.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Fatalf("expected status ok, got %s", body["status"])
	}
}

func TestValidate_Success(t *testing.T) {
	reqBody := ValidateRequest{
		Parameters: map[string]string{"server_type": "mcp-server", "tool_type": "mcp-tool"},
		Schema: SchemaInfo{
			EntityTypes: []SchemaEntityType{
				{
					Name:       "mcp-server",
					Attributes: []string{"route_name", "endpoint"},
					Associations: []SchemaAssociation{
						{Name: "tools", Type: "containment", TargetEntityType: "mcp-tool"},
					},
				},
				{Name: "mcp-tool", Attributes: []string{"type"}},
			},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/validate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleValidate(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp["valid"] != true {
		t.Fatalf("expected valid=true, got %v", resp["valid"])
	}
}

func TestValidate_MissingParams(t *testing.T) {
	reqBody := ValidateRequest{Parameters: map[string]string{}}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/validate", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handleValidate(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestValidate_MissingEntityType(t *testing.T) {
	reqBody := ValidateRequest{
		Parameters: map[string]string{"server_type": "nonexistent", "tool_type": "mcp-tool"},
		Schema:     SchemaInfo{EntityTypes: []SchemaEntityType{{Name: "mcp-tool"}}},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/validate", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handleValidate(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestValidate_MissingRouteNameAttr(t *testing.T) {
	reqBody := ValidateRequest{
		Parameters: map[string]string{"server_type": "mcp-server", "tool_type": "mcp-tool"},
		Schema: SchemaInfo{
			EntityTypes: []SchemaEntityType{
				{Name: "mcp-server", Attributes: []string{"endpoint"}},
			},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/validate", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handleValidate(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestValidate_NoContainment(t *testing.T) {
	reqBody := ValidateRequest{
		Parameters: map[string]string{"server_type": "mcp-server", "tool_type": "mcp-tool"},
		Schema: SchemaInfo{
			EntityTypes: []SchemaEntityType{
				{Name: "mcp-server", Attributes: []string{"route_name"}, Associations: []SchemaAssociation{}},
			},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/validate", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handleValidate(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestExport_Success(t *testing.T) {
	reqBody := ExportRequest{
		CatalogName:        "test-catalog",
		CatalogDescription: "Test",
		Parameters:         map[string]string{"server_type": "mcp-server", "tool_type": "mcp-tool", "target_namespace": "prod"},
		InstancesByType: map[string][]ExportInstance{
			"mcp-server": {
				{ID: "s1", Name: "github", Attributes: map[string]any{"route_name": "route-gh", "mcp_path": "/mcp"}},
			},
			"mcp-tool": {
				{ID: "t1", Name: "create-pr", Attributes: map[string]any{"type": "write"}, ParentID: "s1"},
			},
		},
		ChildrenOf: map[string][]string{"s1": {"t1"}},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/export", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handleExport(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	artifacts := resp["artifacts"].([]any)
	if len(artifacts) != 2 {
		t.Fatalf("expected 2 artifacts, got %d", len(artifacts))
	}

	// First artifact should be MCPServerRegistration
	a0 := artifacts[0].(map[string]any)
	if a0["kind"] != "MCPServerRegistration" {
		t.Fatalf("expected MCPServerRegistration, got %s", a0["kind"])
	}
	if a0["name"] != "github" {
		t.Fatalf("expected name=github, got %s", a0["name"])
	}
	yaml := a0["yaml"].(string)
	if !contains(yaml, "route-gh") {
		t.Fatalf("YAML should contain route name: %s", yaml)
	}

	// Second artifact should be MCPVirtualServer
	a1 := artifacts[1].(map[string]any)
	if a1["kind"] != "MCPVirtualServer" {
		t.Fatalf("expected MCPVirtualServer, got %s", a1["kind"])
	}
}

func TestExport_NoServers(t *testing.T) {
	reqBody := ExportRequest{
		CatalogName: "empty",
		Parameters:  map[string]string{"server_type": "mcp-server", "tool_type": "mcp-tool"},
		InstancesByType: map[string][]ExportInstance{},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/export", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handleExport(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	warnings := resp["warnings"].([]any)
	if len(warnings) == 0 {
		t.Fatal("expected warning about no instances")
	}
}

func TestAuth_MissingToken(t *testing.T) {
	handler := withAuth([]string{"sa"}, handleValidate)
	req := httptest.NewRequest(http.MethodPost, "/validate", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAuth_InvalidFormat(t *testing.T) {
	handler := withAuth([]string{"sa"}, handleValidate)
	req := httptest.NewRequest(http.MethodPost, "/validate", nil)
	req.Header.Set("Authorization", "Basic abc123")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAuth_ValidToken(t *testing.T) {
	reqBody := ValidateRequest{
		Parameters: map[string]string{"server_type": "s", "tool_type": "t"},
		Schema:     SchemaInfo{EntityTypes: []SchemaEntityType{{Name: "s", Attributes: []string{"route_name"}, Associations: []SchemaAssociation{{Type: "containment", TargetEntityType: "t"}}}}},
	}
	body, _ := json.Marshal(reqBody)
	handler := withAuth([]string{"sa"}, handleValidate)
	req := httptest.NewRequest(http.MethodPost, "/validate", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-AssetHub-Protocol-Version", "v1")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAuth_WrongProtocolVersion(t *testing.T) {
	handler := withAuth([]string{"sa"}, handleValidate)
	req := httptest.NewRequest(http.MethodPost, "/validate", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("X-AssetHub-Protocol-Version", "v99")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	if !contains(resp["error"], "unsupported") {
		t.Fatalf("expected 'unsupported' in error, got %s", resp["error"])
	}
}

func TestBuildServerRegistrationYAML_EscapesInjection(t *testing.T) {
	maliciousName := "evil\n  injected: true"
	yaml := buildServerRegistrationYAML(maliciousName, "ns", "cat", "route", "/mcp", "", "2026-01-01T00:00:00Z")

	// Verify the newline is escaped (literal \n), not a real newline producing a new YAML key
	for _, line := range strings.Split(yaml, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "injected: true" {
			t.Fatalf("YAML injection: newline in name produced standalone injected key:\n%s", yaml)
		}
	}
}

func TestBuildVirtualServerYAML_EscapesInjection(t *testing.T) {
	maliciousDesc := "legit\n  malicious_key: pwned"
	yaml := buildVirtualServerYAML("cat", "ns", maliciousDesc, []string{"tool1"}, "2026-01-01T00:00:00Z")

	for _, line := range strings.Split(yaml, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "malicious_key: pwned" {
			t.Fatalf("YAML injection: newline in description produced standalone injected key:\n%s", yaml)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
