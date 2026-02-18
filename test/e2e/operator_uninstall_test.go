//go:build e2e

package e2e

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T-B.11: Operator uninstall — delete entity type, verify cleanup
func TestTB11_OperatorUninstall(t *testing.T) {
	waitForAPI(t)

	// Create an entity type
	etResult := createEntityType(t, "E2EUninstallModel")
	et := etResult["entity_type"].(map[string]interface{})
	etID := et["id"].(string)
	require.NotEmpty(t, etID)

	// Verify it exists
	resp := apiRequest(t, http.MethodGet, "/api/meta/v1/entity-types", nil, "Admin")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var listBefore map[string]interface{}
	parseJSON(t, resp, &listBefore)
	totalBefore := int(listBefore["total"].(float64))

	// Delete the entity type
	resp = apiRequest(t, http.MethodDelete, "/api/meta/v1/entity-types/"+etID, nil, "Admin")
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	resp.Body.Close()

	// Verify it's gone
	resp = apiRequest(t, http.MethodGet, "/api/meta/v1/entity-types/"+etID, nil, "Admin")
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()

	// Verify total count decreased
	resp = apiRequest(t, http.MethodGet, "/api/meta/v1/entity-types", nil, "Admin")
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var listAfter map[string]interface{}
	parseJSON(t, resp, &listAfter)
	totalAfter := int(listAfter["total"].(float64))
	assert.Equal(t, totalBefore-1, totalAfter)
}
