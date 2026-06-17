package export_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/project-catalyst/pc-asset-hub/internal/service/operational/export"
)

func newTestWebhookExporter(t *testing.T, server *httptest.Server) *export.WebhookExporter {
	t.Helper()
	we, err := export.NewWebhookExporter(
		"test-webhook",
		"Test webhook exporter",
		server.URL,
		[]export.ParameterDef{{Name: "ns", Type: "string", Required: true}},
		10,
		func() string { return "test-token" },
	)
	require.NoError(t, err)
	return we
}

// T-36.21: Basic getters: Name, Description, ParameterSchema
func TestWebhookExporter_Getters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	we := newTestWebhookExporter(t, server)
	assert.Equal(t, "test-webhook", we.Name())
	assert.Equal(t, "Test webhook exporter", we.Description())
	require.Len(t, we.ParameterSchema(), 1)
	assert.Equal(t, "ns", we.ParameterSchema()[0].Name)
}

// T-36.22: ValidateSchema success (200 {"valid":true})
func TestWebhookExporter_ValidateSchema_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(export.WebhookValidateResponse{Valid: true})
	}))
	defer server.Close()

	we := newTestWebhookExporter(t, server)
	err := we.ValidateSchema(context.Background(), map[string]string{"ns": "default"}, export.SchemaInfo{})
	assert.NoError(t, err)
}

// T-36.23: ValidateSchema error (400 {"valid":false,"error":"msg"})
func TestWebhookExporter_ValidateSchema_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(export.WebhookValidateResponse{Valid: false, Error: "missing required param"})
	}))
	defer server.Close()

	we := newTestWebhookExporter(t, server)
	err := we.ValidateSchema(context.Background(), map[string]string{}, export.SchemaInfo{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required param")
}

// T-36.23b: ValidateSchema captured request body contains parameters + schema with associations
func TestWebhookExporter_ValidateSchema_RequestBody(t *testing.T) {
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(export.WebhookValidateResponse{Valid: true})
	}))
	defer server.Close()

	we := newTestWebhookExporter(t, server)
	schema := export.SchemaInfo{
		EntityTypes: []export.SchemaEntityType{
			{Name: "server", Attributes: []string{"ep"}, Associations: []export.SchemaAssociation{
				{Name: "tools", Type: "containment", TargetEntityType: "tool"},
			}},
		},
	}
	err := we.ValidateSchema(context.Background(), map[string]string{"ns": "prod"}, schema)
	require.NoError(t, err)

	assert.Contains(t, string(capturedBody), `"parameters"`)
	assert.Contains(t, string(capturedBody), `"schema"`)
	assert.Contains(t, string(capturedBody), `"target_entity_type"`)
}

// T-36.24: Export success (200 with artifacts JSON)
func TestWebhookExporter_Export_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(export.WebhookExportResponse{
			Artifacts: []export.WebhookArtifact{
				{APIVersion: "v1", Kind: "ConfigMap", Name: "test", Namespace: "ns", YAML: "data: test"},
			},
			Warnings: []string{"test warning"},
		})
	}))
	defer server.Close()

	we := newTestWebhookExporter(t, server)
	output, err := we.Export(context.Background(), export.ExportInput{
		CatalogName: "test",
		Parameters:  map[string]string{"ns": "default"},
	})
	require.NoError(t, err)
	require.Len(t, output.Artifacts, 1)
	assert.Equal(t, "ConfigMap", output.Artifacts[0].Kind)
	assert.Equal(t, []string{"test warning"}, output.Warnings)
}

// T-36.25: Export 4xx → validation error
func TestWebhookExporter_Export_4xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(export.WebhookErrorResponse{Error: "bad input"})
	}))
	defer server.Close()

	we := newTestWebhookExporter(t, server)
	_, err := we.Export(context.Background(), export.ExportInput{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bad input")
}

// T-36.26: Export 5xx → non-validation error with "webhook plugin error"
func TestWebhookExporter_Export_5xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(export.WebhookErrorResponse{Error: "db down"})
	}))
	defer server.Close()

	we := newTestWebhookExporter(t, server)
	_, err := we.Export(context.Background(), export.ExportInput{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "webhook plugin error")
}

// T-36.27: Export timeout → error with "timed out"
func TestWebhookExporter_Export_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
	}))
	defer server.Close()

	we, err := export.NewWebhookExporter("test", "test", server.URL, nil, 3, func() string { return "" })
	require.NoError(t, err)
	_, exportErr := we.Export(context.Background(), export.ExportInput{})
	require.Error(t, exportErr)
	assert.Contains(t, exportErr.Error(), "timed out")
}

// T-36.28: Export connection refused → error with "unreachable"
func TestWebhookExporter_Export_ConnectionRefused(t *testing.T) {
	we, err := export.NewWebhookExporter("test", "test", "http://127.0.0.1:1", nil, 3, func() string { return "" })
	require.NoError(t, err)
	_, exportErr := we.Export(context.Background(), export.ExportInput{})
	require.Error(t, exportErr)
	assert.Contains(t, exportErr.Error(), "unreachable")
}

// T-36.29: Export oversized response (>10MB) → validation error
func TestWebhookExporter_Export_OversizedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"artifacts":[{"yaml":"`))
		for i := 0; i < 11*1024*1024; i++ {
			w.Write([]byte("x"))
		}
		w.Write([]byte(`"}]}`))
	}))
	defer server.Close()

	we := newTestWebhookExporter(t, server)
	_, err := we.Export(context.Background(), export.ExportInput{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "10 MB")
}

// T-36.30: Export malformed JSON (200 with invalid body) → error
func TestWebhookExporter_Export_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`not json`))
	}))
	defer server.Close()

	we := newTestWebhookExporter(t, server)
	_, err := we.Export(context.Background(), export.ExportInput{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not valid JSON")
}

// T-36.31: Export empty 200 response body → clear error
func TestWebhookExporter_Export_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
	}))
	defer server.Close()

	we := newTestWebhookExporter(t, server)
	_, err := we.Export(context.Background(), export.ExportInput{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty response")
}

// T-36.32: Export HTTP 302 redirect → rejected with clear error
func TestWebhookExporter_Export_Redirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://evil.example.com", http.StatusFound)
	}))
	defer server.Close()

	we := newTestWebhookExporter(t, server)
	_, err := we.Export(context.Background(), export.ExportInput{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "redirect")
}

// T-36.33: Response Content-Type mismatch (text/html) → hints at Content-Type
func TestWebhookExporter_Export_ContentTypeMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html>Not Found</html>`))
	}))
	defer server.Close()

	we := newTestWebhookExporter(t, server)
	_, err := we.Export(context.Background(), export.ExportInput{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Content-Type")
}

// T-36.34: X-AssetHub-Protocol-Version: v1 header on all outgoing requests
func TestWebhookExporter_ProtocolVersionHeader(t *testing.T) {
	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(export.WebhookValidateResponse{Valid: true})
	}))
	defer server.Close()

	we := newTestWebhookExporter(t, server)
	we.ValidateSchema(context.Background(), map[string]string{}, export.SchemaInfo{})
	assert.Equal(t, "v1", capturedHeaders.Get("X-AssetHub-Protocol-Version"))
}

// T-36.35: Authorization: Bearer {token} header on all outgoing requests
func TestWebhookExporter_AuthorizationHeader(t *testing.T) {
	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(export.WebhookValidateResponse{Valid: true})
	}))
	defer server.Close()

	we := newTestWebhookExporter(t, server)
	we.ValidateSchema(context.Background(), map[string]string{}, export.SchemaInfo{})
	assert.Equal(t, "Bearer test-token", capturedHeaders.Get("Authorization"))
}

// T-36.36: Content-Type: application/json header on all POST requests
func TestWebhookExporter_ContentTypeHeader(t *testing.T) {
	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(export.WebhookValidateResponse{Valid: true})
	}))
	defer server.Close()

	we := newTestWebhookExporter(t, server)
	we.ValidateSchema(context.Background(), map[string]string{}, export.SchemaInfo{})
	assert.Equal(t, "application/json", capturedHeaders.Get("Content-Type"))
}

// T-36.38: Timeout calculation: timeoutSeconds=15 → validate=7s, export=15s
func TestWebhookExporter_Timeout_15(t *testing.T) {
	we, err := export.NewWebhookExporter("test", "test", "http://localhost", nil, 15, func() string { return "" })
	require.NoError(t, err)
	vt, et := we.Timeouts()
	assert.Equal(t, 7*time.Second, vt)
	assert.Equal(t, 15*time.Second, et)
}

// T-36.39: Timeout calculation: timeoutSeconds=3 → both=3s
func TestWebhookExporter_Timeout_3(t *testing.T) {
	we, err := export.NewWebhookExporter("test", "test", "http://localhost", nil, 3, func() string { return "" })
	require.NoError(t, err)
	vt, et := we.Timeouts()
	assert.Equal(t, 3*time.Second, vt)
	assert.Equal(t, 3*time.Second, et)
}

// T-36.39b: Timeout calculation: timeoutSeconds=1 → both clamped to 3s
func TestWebhookExporter_Timeout_BelowFloor(t *testing.T) {
	we, err := export.NewWebhookExporter("test", "test", "http://localhost", nil, 1, func() string { return "" })
	require.NoError(t, err)
	vt, et := we.Timeouts()
	assert.Equal(t, 3*time.Second, vt)
	assert.Equal(t, 3*time.Second, et)
}

// T-36.40: HealthStatus get/set
func TestWebhookExporter_HealthStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	we := newTestWebhookExporter(t, server)
	assert.Equal(t, "Unknown", we.HealthStatus())
	we.SetHealthStatus("Ready")
	assert.Equal(t, "Ready", we.HealthStatus())
}

// T-36.41: Zero ParameterSchema — registration succeeds, Export with empty params
func TestWebhookExporter_ZeroParamSchema(t *testing.T) {
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(export.WebhookExportResponse{})
	}))
	defer server.Close()

	we, err := export.NewWebhookExporter("test", "test", server.URL, nil, 10, func() string { return "" })
	require.NoError(t, err)
	assert.Empty(t, we.ParameterSchema())

	_, err = we.Export(context.Background(), export.ExportInput{Parameters: map[string]string{}})
	require.NoError(t, err)
	assert.Contains(t, string(capturedBody), `"parameters":{}`)
}

// T-36.42: Dead fields excluded from export request body
func TestWebhookExporter_DeadFieldsExcluded(t *testing.T) {
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(export.WebhookExportResponse{})
	}))
	defer server.Close()

	we := newTestWebhookExporter(t, server)
	input := export.ExportInput{
		CatalogName:               "test",
		CVLabel:                   "dead-field",
		EntityTypes:               []export.ExportEntityType{{Name: "dead-field"}},
		VirtualServerInstanceName: "my-vs",
		AllowedToolIDs:            map[string]bool{"id": true},
		Parameters:                map[string]string{},
		InstancesByType:           map[string][]*export.ExportInstance{},
		ChildrenOf:                map[string][]*export.ExportInstance{},
	}
	we.Export(context.Background(), input)

	raw := string(capturedBody)
	assert.NotContains(t, raw, "dead-field")
	assert.NotContains(t, raw, "cv_label")
	assert.Contains(t, raw, "my-vs", "virtual_server_instance_name should be included")
	assert.NotContains(t, raw, "entity_types")
}

// T-36.44: Base URL with path rejected at construction
func TestWebhookExporter_BaseURLWithPath_Rejected(t *testing.T) {
	_, err := export.NewWebhookExporter("test", "test", "http://localhost/api/v1", nil, 10, func() string { return "" })
	require.Error(t, err)
	assert.Contains(t, err.Error(), "path")
}

// T-36.44b: Webhook plugin returns warnings → propagated
func TestWebhookExporter_WarningsPropagated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(export.WebhookExportResponse{
			Artifacts: []export.WebhookArtifact{{Kind: "ConfigMap", Name: "test", YAML: "data: x"}},
			Warnings:  []string{"warn1", "warn2"},
		})
	}))
	defer server.Close()

	we := newTestWebhookExporter(t, server)
	output, err := we.Export(context.Background(), export.ExportInput{Parameters: map[string]string{}})
	require.NoError(t, err)
	assert.Equal(t, []string{"warn1", "warn2"}, output.Warnings)
}

// T-36.45: Round-trip: ExportInput → WebhookExportRequest → httptest → WebhookExportResponse → ExportOutput
func TestWebhookExporter_RoundTrip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req export.WebhookExportRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Echo back catalog name and instance count as artifacts
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(export.WebhookExportResponse{
			Artifacts: []export.WebhookArtifact{
				{APIVersion: "v1", Kind: "Echo", Name: req.CatalogName, YAML: fmt.Sprintf("instances: %d", len(req.InstancesByType))},
			},
			Warnings: []string{"roundtrip-ok"},
		})
	}))
	defer server.Close()

	we := newTestWebhookExporter(t, server)
	input := export.ExportInput{
		CatalogName: "test-catalog",
		CatalogDesc: "Test",
		Parameters:  map[string]string{"ns": "default"},
		InstancesByType: map[string][]*export.ExportInstance{
			"server": {{ID: "1", Name: "s1"}},
		},
		ChildrenOf: map[string][]*export.ExportInstance{},
	}

	output, err := we.Export(context.Background(), input)
	require.NoError(t, err)
	require.Len(t, output.Artifacts, 1)
	assert.Equal(t, "test-catalog", output.Artifacts[0].Name)
	assert.Contains(t, output.Artifacts[0].YAML, "instances: 1")
	assert.Equal(t, []string{"roundtrip-ok"}, output.Warnings)
}

// T-36.37: Protocol version mismatch: plugin returns v2 response header → no rejection
func TestWebhookExporter_ProtocolVersionMismatch_NoRejection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-AssetHub-Protocol-Version", "v2")
		json.NewEncoder(w).Encode(export.WebhookExportResponse{
			Artifacts: []export.WebhookArtifact{{Kind: "Test", Name: "ok", YAML: "data: ok"}},
		})
	}))
	defer server.Close()

	we := newTestWebhookExporter(t, server)
	output, err := we.Export(context.Background(), export.ExportInput{Parameters: map[string]string{}})
	require.NoError(t, err)
	require.Len(t, output.Artifacts, 1)
}

// T-36.43: VS filtering parity: pre-filtered ExportInput → faithfully serialized
func TestWebhookExporter_VSFilteringParity(t *testing.T) {
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(export.WebhookExportResponse{})
	}))
	defer server.Close()

	we := newTestWebhookExporter(t, server)
	input := export.ExportInput{
		CatalogName: "test",
		Parameters:  map[string]string{"ns": "prod"},
		InstancesByType: map[string][]*export.ExportInstance{
			"tool": {{ID: "t1", Name: "allowed-tool"}},
		},
		ChildrenOf: map[string][]*export.ExportInstance{
			"s1": {{ID: "t1", Name: "allowed-tool"}},
		},
	}
	we.Export(context.Background(), input)

	var req export.WebhookExportRequest
	json.Unmarshal(capturedBody, &req)
	require.Len(t, req.InstancesByType["tool"], 1)
	assert.Equal(t, "allowed-tool", req.InstancesByType["tool"][0].Name)
	require.Len(t, req.ChildrenOf["s1"], 1)
}

// Validate timeout error reports validateTimeout, not exportTimeout
func TestWebhookExporter_ValidateSchema_Timeout_ReportsCorrectDuration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
	}))
	defer server.Close()

	// timeoutSeconds=10 → validateTimeout=5s, exportTimeout=10s
	we, err := export.NewWebhookExporter("test", "test", server.URL, nil, 10, func() string { return "" })
	require.NoError(t, err)
	valErr := we.ValidateSchema(context.Background(), map[string]string{}, export.SchemaInfo{})
	require.Error(t, valErr)
	assert.Contains(t, valErr.Error(), "5s", "should report validateTimeout (5s), not exportTimeout (10s)")
	assert.NotContains(t, valErr.Error(), "10s", "should not report exportTimeout")
}

// Fix #2 verification: new WebhookExporter defaults healthStatus to "Unknown"
func TestWebhookExporter_DefaultHealthStatus_Unknown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()
	we := newTestWebhookExporter(t, server)
	assert.Equal(t, "Unknown", we.HealthStatus(), "new WebhookExporter should default to Unknown health")
}

// T-36.44c: NewWebhookExporter with unparseable URL → error
func TestWebhookExporter_InvalidURL_Rejected(t *testing.T) {
	_, err := export.NewWebhookExporter("test", "test", "://invalid", nil, 10, func() string { return "" })
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

// T-36.46: Export 4xx with non-JSON body → generic "plugin returned HTTP 4xx" error
func TestWebhookExporter_Export_4xx_NonJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("plain text error"))
	}))
	defer server.Close()

	we := newTestWebhookExporter(t, server)
	_, err := we.Export(context.Background(), export.ExportInput{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 400")
}

// T-36.47: Export 5xx with non-JSON body → generic "webhook plugin error: HTTP 5xx"
func TestWebhookExporter_Export_5xx_NonJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("plain text error"))
	}))
	defer server.Close()

	we := newTestWebhookExporter(t, server)
	_, err := we.Export(context.Background(), export.ExportInput{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 500")
}

// T-36.48: ValidateSchema 200 with non-JSON body → "not valid JSON" error
func TestWebhookExporter_ValidateSchema_200_NonJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not json"))
	}))
	defer server.Close()

	we := newTestWebhookExporter(t, server)
	err := we.ValidateSchema(context.Background(), map[string]string{}, export.SchemaInfo{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not valid JSON")
}

// T-36.49: ValidateSchema 200 with valid:false → validation error with message
func TestWebhookExporter_ValidateSchema_200_Invalid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(export.WebhookValidateResponse{Valid: false, Error: "schema mismatch"})
	}))
	defer server.Close()

	we := newTestWebhookExporter(t, server)
	err := we.ValidateSchema(context.Background(), map[string]string{}, export.SchemaInfo{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "schema mismatch")
}

// Concurrent HealthStatus get/set must not race
func TestWebhookExporter_HealthStatus_ConcurrentAccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	we := newTestWebhookExporter(t, server)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			we.SetHealthStatus("Ready")
		}()
		go func() {
			defer wg.Done()
			_ = we.HealthStatus()
		}()
	}
	wg.Wait()
}

// Coverage: lines 109-111 — readLimitedBody error in ValidateSchema via server abort
func TestWebhookExporter_ValidateSchema_ReadError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Length", "1000000")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"valid":`))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		hj, ok := w.(http.Hijacker)
		if !ok {
			return
		}
		conn, _, _ := hj.Hijack()
		conn.Close()
	}))
	defer server.Close()

	we := newTestWebhookExporter(t, server)
	err := we.ValidateSchema(context.Background(), map[string]string{}, export.SchemaInfo{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read response body")
}

// Coverage attempt: io.ReadAll error via server that aborts mid-response
func TestWebhookExporter_Export_ReadError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Length", "1000000")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"artifacts":[`))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		hj, ok := w.(http.Hijacker)
		if !ok {
			return
		}
		conn, _, _ := hj.Hijack()
		conn.Close()
	}))
	defer server.Close()

	we := newTestWebhookExporter(t, server)
	_, err := we.Export(context.Background(), export.ExportInput{Parameters: map[string]string{}})
	require.Error(t, err)
}

// ValidateSchema respects caller's context cancellation
func TestWebhookExporter_ValidateSchema_RespectsCallerContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(export.WebhookValidateResponse{Valid: true})
	}))
	defer server.Close()

	we := newTestWebhookExporter(t, server)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := we.ValidateSchema(ctx, map[string]string{}, export.SchemaInfo{})
	require.Error(t, err)
}
