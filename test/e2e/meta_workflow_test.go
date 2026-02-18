//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T-B.08: Full meta workflow — create entity type → add attributes → create catalog version → promote → verify
func TestTB08_MetaWorkflow(t *testing.T) {
	waitForAPI(t)

	// Step 1: Create an entity type
	etResult := createEntityType(t, "E2EModel")
	et := etResult["entity_type"].(map[string]interface{})
	etID := et["id"].(string)
	require.NotEmpty(t, etID)

	// Verify entity type version was created
	version := etResult["version"].(map[string]interface{})
	versionID := version["id"].(string)
	require.NotEmpty(t, versionID)

	// Step 2: Verify the entity type can be retrieved
	resp := apiRequest(t, http.MethodGet, fmt.Sprintf("/api/meta/v1/entity-types/%s", etID), nil, "Admin")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var etDetail map[string]interface{}
	parseJSON(t, resp, &etDetail)
	assert.Equal(t, "E2EModel", etDetail["name"])

	// Step 3: List entity types
	resp = apiRequest(t, http.MethodGet, "/api/meta/v1/entity-types", nil, "Admin")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var listResult map[string]interface{}
	parseJSON(t, resp, &listResult)
	assert.GreaterOrEqual(t, int(listResult["total"].(float64)), 1)

	// Step 4: Create a catalog version with the entity type pinned
	cvResult := createCatalogVersion(t, "e2e-v1.0", []map[string]string{
		{"entity_type_version_id": versionID},
	})
	cvID := cvResult["id"].(string)
	require.NotEmpty(t, cvID)

	// Step 5: Verify catalog version is in development
	resp = apiRequest(t, http.MethodGet, fmt.Sprintf("/api/meta/v1/catalog-versions/%s", cvID), nil, "Admin")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var cvDetail map[string]interface{}
	parseJSON(t, resp, &cvDetail)
	assert.Equal(t, "development", cvDetail["lifecycle_stage"])

	// Step 6: Promote to testing
	promoteCatalogVersion(t, cvID)

	// Step 7: Verify it's now in testing
	resp = apiRequest(t, http.MethodGet, fmt.Sprintf("/api/meta/v1/catalog-versions/%s", cvID), nil, "Admin")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	parseJSON(t, resp, &cvDetail)
	assert.Equal(t, "testing", cvDetail["lifecycle_stage"])

	// Step 8: Promote to production
	promoteCatalogVersion(t, cvID)

	resp = apiRequest(t, http.MethodGet, fmt.Sprintf("/api/meta/v1/catalog-versions/%s", cvID), nil, "Admin")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	parseJSON(t, resp, &cvDetail)
	assert.Equal(t, "production", cvDetail["lifecycle_stage"])

	// Cleanup: delete entity type
	resp = apiRequest(t, http.MethodDelete, fmt.Sprintf("/api/meta/v1/entity-types/%s", etID), nil, "Admin")
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()
}
