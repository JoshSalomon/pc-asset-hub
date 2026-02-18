package health

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/project-catalyst/pc-asset-hub/internal/infrastructure/gorm/testutil"
)

func TestHealthz(t *testing.T) {
	db := testutil.NewTestDB(t)
	h := NewHandler(db)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.Healthz(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"status":"ok"`)
}

func TestReadyz_Healthy(t *testing.T) {
	db := testutil.NewTestDB(t)
	h := NewHandler(db)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.Readyz(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"status":"ready"`)
}

func TestReadyz_Unhealthy(t *testing.T) {
	db := testutil.NewTestDB(t)
	// Close the DB to simulate an unreachable database
	sqlDB, err := db.DB()
	require.NoError(t, err)
	_ = sqlDB.Close()

	h := NewHandler(db)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err = h.Readyz(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestRegisterRoutes(t *testing.T) {
	db := testutil.NewTestDB(t)
	h := NewHandler(db)
	e := echo.New()
	RegisterRoutes(e, h)

	// Verify routes are registered
	routes := e.Routes()
	var paths []string
	for _, r := range routes {
		paths = append(paths, r.Path)
	}
	assert.Contains(t, paths, "/healthz")
	assert.Contains(t, paths, "/readyz")
}
