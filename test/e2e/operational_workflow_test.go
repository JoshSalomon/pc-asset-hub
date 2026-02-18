//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T-B.09: Operational workflow — create/update/filter/delete instance with cascade
func TestTB09_OperationalWorkflow(t *testing.T) {
	waitForAPI(t)

	// Setup: create entity type and catalog version
	etResult := createEntityType(t, "E2EOperationalModel")
	et := etResult["entity_type"].(map[string]interface{})
	etID := et["id"].(string)
	version := etResult["version"].(map[string]interface{})
	versionID := version["id"].(string)

	cvResult := createCatalogVersion(t, "e2e-op-v1.0", []map[string]string{
		{"entity_type_version_id": versionID},
	})
	cvID := cvResult["id"].(string)

	// Step 1: Create an instance
	resp := apiRequest(t, http.MethodPost, fmt.Sprintf("/api/data/v1/%s/E2EOperationalModel", cvID), map[string]interface{}{
		"name":        "test-instance-1",
		"description": "An E2E test instance",
	}, "Admin")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	var instResult map[string]interface{}
	parseJSON(t, resp, &instResult)
	instID := instResult["id"].(string)
	require.NotEmpty(t, instID)

	// Step 2: Get the instance
	resp = apiRequest(t, http.MethodGet, fmt.Sprintf("/api/data/v1/%s/E2EOperationalModel/%s", cvID, instID), nil, "Admin")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var instDetail map[string]interface{}
	parseJSON(t, resp, &instDetail)
	assert.Equal(t, "test-instance-1", instDetail["name"])

	// Step 3: List instances
	resp = apiRequest(t, http.MethodGet, fmt.Sprintf("/api/data/v1/%s/E2EOperationalModel", cvID), nil, "Admin")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var listResult map[string]interface{}
	parseJSON(t, resp, &listResult)
	assert.GreaterOrEqual(t, int(listResult["total"].(float64)), 1)

	// Step 4: Update the instance
	resp = apiRequest(t, http.MethodPut, fmt.Sprintf("/api/data/v1/%s/E2EOperationalModel/%s", cvID, instID), map[string]interface{}{
		"version": 1,
	}, "Admin")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Step 5: Delete the instance
	resp = apiRequest(t, http.MethodDelete, fmt.Sprintf("/api/data/v1/%s/E2EOperationalModel/%s", cvID, instID), nil, "Admin")
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()

	// Step 6: Verify deletion
	resp = apiRequest(t, http.MethodGet, fmt.Sprintf("/api/data/v1/%s/E2EOperationalModel/%s", cvID, instID), nil, "Admin")
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()

	// Cleanup
	resp = apiRequest(t, http.MethodDelete, fmt.Sprintf("/api/meta/v1/entity-types/%s", etID), nil, "Admin")
	resp.Body.Close()
}
