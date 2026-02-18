//go:build e2e

package e2e

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T-B.10: Demotion workflow — promote catalog version, then demote, verify state changes
func TestTB10_DemotionWorkflow(t *testing.T) {
	waitForAPI(t)

	// Setup: create entity type and catalog version
	etResult := createEntityType(t, "E2EDemotionModel")
	et := etResult["entity_type"].(map[string]interface{})
	etID := et["id"].(string)
	version := etResult["version"].(map[string]interface{})
	versionID := version["id"].(string)

	cvResult := createCatalogVersion(t, "e2e-demo-v1.0", []map[string]string{
		{"entity_type_version_id": versionID},
	})
	cvID := cvResult["id"].(string)

	// Promote to testing
	promoteCatalogVersion(t, cvID)

	// Promote to production
	promoteCatalogVersion(t, cvID)

	// Verify production
	resp := apiRequest(t, http.MethodGet, fmt.Sprintf("/api/meta/v1/catalog-versions/%s", cvID), nil, "Admin")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var cvDetail map[string]interface{}
	parseJSON(t, resp, &cvDetail)
	assert.Equal(t, "production", cvDetail["lifecycle_stage"])

	// Demote to testing
	demoteCatalogVersion(t, cvID, "testing")

	// Verify testing
	resp = apiRequest(t, http.MethodGet, fmt.Sprintf("/api/meta/v1/catalog-versions/%s", cvID), nil, "Admin")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	parseJSON(t, resp, &cvDetail)
	assert.Equal(t, "testing", cvDetail["lifecycle_stage"])

	// Demote to development
	demoteCatalogVersion(t, cvID, "development")

	// Verify development
	resp = apiRequest(t, http.MethodGet, fmt.Sprintf("/api/meta/v1/catalog-versions/%s", cvID), nil, "Admin")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	parseJSON(t, resp, &cvDetail)
	assert.Equal(t, "development", cvDetail["lifecycle_stage"])

	// Cleanup
	resp = apiRequest(t, http.MethodDelete, fmt.Sprintf("/api/meta/v1/entity-types/%s", etID), nil, "Admin")
	resp.Body.Close()
}
