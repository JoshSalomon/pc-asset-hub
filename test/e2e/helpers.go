//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func apiBaseURL() string {
	if url := os.Getenv("E2E_API_URL"); url != "" {
		return url
	}
	return "http://localhost:30080"
}

func apiRequest(t *testing.T, method, path string, body interface{}, role string) *http.Response {
	t.Helper()
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		require.NoError(t, err)
		bodyReader = bytes.NewReader(data)
	}

	url := apiBaseURL() + path
	req, err := http.NewRequest(method, url, bodyReader)
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	if role != "" {
		req.Header.Set("X-User-Role", role)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	return resp
}

func parseJSON(t *testing.T, resp *http.Response, target interface{}) {
	t.Helper()
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, target), "body: %s", string(data))
}

func waitForAPI(t *testing.T) {
	t.Helper()
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := client.Get(apiBaseURL() + "/healthz")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatal("API server did not become ready")
}

func createEntityType(t *testing.T, name string) map[string]interface{} {
	t.Helper()
	resp := apiRequest(t, http.MethodPost, "/api/meta/v1/entity-types", map[string]string{
		"name": name,
	}, "Admin")
	require.Equal(t, http.StatusCreated, resp.StatusCode, "failed to create entity type: %s", name)

	var result map[string]interface{}
	parseJSON(t, resp, &result)
	return result
}

func createCatalogVersion(t *testing.T, label string, pins []map[string]string) map[string]interface{} {
	t.Helper()
	body := map[string]interface{}{
		"version_label": label,
	}
	if pins != nil {
		body["pins"] = pins
	}

	resp := apiRequest(t, http.MethodPost, "/api/meta/v1/catalog-versions", body, "Admin")
	require.Equal(t, http.StatusCreated, resp.StatusCode, "failed to create catalog version: %s", label)

	var result map[string]interface{}
	parseJSON(t, resp, &result)
	return result
}

func promoteCatalogVersion(t *testing.T, id string) {
	t.Helper()
	resp := apiRequest(t, http.MethodPost, fmt.Sprintf("/api/meta/v1/catalog-versions/%s/promote", id), nil, "Admin")
	require.Equal(t, http.StatusOK, resp.StatusCode, "failed to promote catalog version")
	resp.Body.Close()
}

func demoteCatalogVersion(t *testing.T, id string, targetStage string) {
	t.Helper()
	resp := apiRequest(t, http.MethodPost, fmt.Sprintf("/api/meta/v1/catalog-versions/%s/demote", id), map[string]string{
		"target_stage": targetStage,
	}, "Admin")
	require.Equal(t, http.StatusOK, resp.StatusCode, "failed to demote catalog version")
	resp.Body.Close()
}
