package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
)

// mockCatalogAccessChecker allows configuring per-catalog access for tests.
type mockCatalogAccessChecker struct {
	allowed    map[string]bool // catalogName → allowed
	lastCatalog string
	lastVerb    string
	returnErr   error
}

func (m *mockCatalogAccessChecker) CheckAccess(c echo.Context, catalogName, verb string) (bool, error) {
	m.lastCatalog = catalogName
	m.lastVerb = verb
	if m.returnErr != nil {
		return false, m.returnErr
	}
	if m.allowed == nil {
		return true, nil
	}
	return m.allowed[catalogName], nil
}

// === T-14.01, T-14.02: HeaderCatalogAccessChecker ===

func TestHeaderCatalogAccessChecker_AlwaysAllows(t *testing.T) {
	checker := &HeaderCatalogAccessChecker{}
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c := e.NewContext(req, httptest.NewRecorder())

	allowed, err := checker.CheckAccess(c, "any-catalog", "get")
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestHeaderCatalogAccessChecker_AllowsAllVerbs(t *testing.T) {
	checker := &HeaderCatalogAccessChecker{}
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c := e.NewContext(req, httptest.NewRecorder())

	for _, verb := range []string{"get", "create", "update", "delete"} {
		allowed, err := checker.CheckAccess(c, "test-catalog", verb)
		require.NoError(t, err)
		assert.True(t, allowed, "verb %s should be allowed", verb)
	}
}

// httpMethodToVerb: PUT maps to "update"
func TestHttpMethodToVerb_PutMapsToUpdate(t *testing.T) {
	mock := &mockCatalogAccessChecker{}
	mw := RequireCatalogAccess(mock)
	e := echo.New()

	e.PUT("/catalogs/:catalog-name/items/:id", mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}))

	req := httptest.NewRequest(http.MethodPut, "/catalogs/test/items/123", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, "update", mock.lastVerb)
}

// === T-14.03 through T-14.11: RequireCatalogAccess Middleware ===

func TestRequireCatalogAccess_ExtractsCatalogName(t *testing.T) {
	mock := &mockCatalogAccessChecker{}
	mw := RequireCatalogAccess(mock)
	e := echo.New()

	e.GET("/catalogs/:catalog-name/tree", mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}))

	req := httptest.NewRequest(http.MethodGet, "/catalogs/my-catalog/tree", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, "my-catalog", mock.lastCatalog)
}

func TestRequireCatalogAccess_MapsGETToGet(t *testing.T) {
	mock := &mockCatalogAccessChecker{}
	mw := RequireCatalogAccess(mock)
	e := echo.New()
	e.GET("/catalogs/:catalog-name", mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}))

	req := httptest.NewRequest(http.MethodGet, "/catalogs/test", nil)
	e.ServeHTTP(httptest.NewRecorder(), req)
	assert.Equal(t, "get", mock.lastVerb)
}

func TestRequireCatalogAccess_MapsPOSTToCreate(t *testing.T) {
	mock := &mockCatalogAccessChecker{}
	mw := RequireCatalogAccess(mock)
	e := echo.New()
	e.POST("/catalogs/:catalog-name/instances", mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}))

	req := httptest.NewRequest(http.MethodPost, "/catalogs/test/instances", nil)
	e.ServeHTTP(httptest.NewRecorder(), req)
	assert.Equal(t, "create", mock.lastVerb)
}

func TestRequireCatalogAccess_MapsDELETEToDelete(t *testing.T) {
	mock := &mockCatalogAccessChecker{}
	mw := RequireCatalogAccess(mock)
	e := echo.New()
	e.DELETE("/catalogs/:catalog-name/instances", mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}))

	req := httptest.NewRequest(http.MethodDelete, "/catalogs/test/instances", nil)
	e.ServeHTTP(httptest.NewRecorder(), req)
	assert.Equal(t, "delete", mock.lastVerb)
}

func TestRequireCatalogAccess_Returns403WhenDenied(t *testing.T) {
	mock := &mockCatalogAccessChecker{allowed: map[string]bool{"test": false}}
	mw := RequireCatalogAccess(mock)
	e := echo.New()
	e.GET("/catalogs/:catalog-name", mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}))

	req := httptest.NewRequest(http.MethodGet, "/catalogs/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestRequireCatalogAccess_PassesThroughWhenAllowed(t *testing.T) {
	mock := &mockCatalogAccessChecker{allowed: map[string]bool{"test": true}}
	mw := RequireCatalogAccess(mock)
	e := echo.New()
	e.GET("/catalogs/:catalog-name", mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}))

	req := httptest.NewRequest(http.MethodGet, "/catalogs/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireCatalogAccess_Returns500OnError(t *testing.T) {
	mock := &mockCatalogAccessChecker{returnErr: fmt.Errorf("SAR failed")}
	mw := RequireCatalogAccess(mock)
	e := echo.New()
	e.GET("/catalogs/:catalog-name", mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}))

	req := httptest.NewRequest(http.MethodGet, "/catalogs/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestRequireCatalogAccess_SkipsWhenNoCatalogName(t *testing.T) {
	mock := &mockCatalogAccessChecker{allowed: map[string]bool{}}
	mw := RequireCatalogAccess(mock)
	e := echo.New()
	// Route without :catalog-name param
	e.GET("/catalogs", mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}))

	req := httptest.NewRequest(http.MethodGet, "/catalogs", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "", mock.lastCatalog) // CheckAccess not called
}

func TestRequireCatalogAccess_MapsHEADToGet(t *testing.T) {
	mock := &mockCatalogAccessChecker{}
	mw := RequireCatalogAccess(mock)
	e := echo.New()
	e.Add(http.MethodHead, "/catalogs/:catalog-name", mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}))

	req := httptest.NewRequest(http.MethodHead, "/catalogs/test", nil)
	e.ServeHTTP(httptest.NewRecorder(), req)
	assert.Equal(t, "get", mock.lastVerb)
}

func TestHttpMethodToVerb_DefaultReturnsGet(t *testing.T) {
	assert.Equal(t, "get", httpMethodToVerb("OPTIONS"))
	assert.Equal(t, "get", httpMethodToVerb("UNKNOWN"))
}

// === T-14.12 through T-14.14: FilterAccessibleCatalogs ===

func TestFilterAccessibleCatalogs_FiltersDenied(t *testing.T) {
	mock := &mockCatalogAccessChecker{allowed: map[string]bool{
		"allowed-1": true, "denied-1": false, "allowed-2": true,
	}}
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c := e.NewContext(req, httptest.NewRecorder())

	identity := func(s string) string { return s }

	result, err := FilterAccessible(c, mock, []string{"allowed-1", "denied-1", "allowed-2"}, identity)
	require.NoError(t, err)
	assert.Equal(t, []string{"allowed-1", "allowed-2"}, result)
}

func TestFilterAccessibleCatalogs_AllAllowed(t *testing.T) {
	mock := &mockCatalogAccessChecker{} // nil allowed map = all allowed
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c := e.NewContext(req, httptest.NewRecorder())

	identity := func(s string) string { return s }
	result, err := FilterAccessible(c, mock, []string{"a", "b", "c"}, identity)
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, result)
}

func TestFilterAccessibleCatalogs_AllDenied(t *testing.T) {
	mock := &mockCatalogAccessChecker{allowed: map[string]bool{
		"a": false, "b": false,
	}}
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c := e.NewContext(req, httptest.NewRecorder())

	identity := func(s string) string { return s }
	result, err := FilterAccessible(c, mock, []string{"a", "b"}, identity)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestFilterAccessibleCatalogs_ErrorPropagates(t *testing.T) {
	mock := &mockCatalogAccessChecker{returnErr: fmt.Errorf("SAR failed")}
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	c := e.NewContext(req, httptest.NewRecorder())

	identity := func(s string) string { return s }
	_, err := FilterAccessible(c, mock, []string{"a"}, identity)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SAR failed")
}

// === RequireWriteAccess Middleware Tests ===

type mockPublishChecker struct {
	published map[string]bool
	returnErr error
}

func (m *mockPublishChecker) IsPublished(_ echo.Context, catalogName string) (bool, error) {
	if m.returnErr != nil {
		return false, m.returnErr
	}
	return m.published[catalogName], nil
}

// T-16.17: RW blocked on published catalog
func TestT16_17_WriteProtection_RWBlocked(t *testing.T) {
	checker := &mockPublishChecker{published: map[string]bool{"prod-catalog": true}}
	e := echo.New()
	e.Use(RBACMiddleware(&HeaderRBACProvider{}))
	g := e.Group("/catalogs/:catalog-name")
	g.POST("/items", func(c echo.Context) error {
		return c.String(http.StatusCreated, "created")
	}, RequireRole(RoleRW), RequireWriteAccess(checker))

	req := httptest.NewRequest(http.MethodPost, "/catalogs/prod-catalog/items", nil)
	req.Header.Set("X-User-Role", "RW")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// T-16.18: SuperAdmin allowed on published catalog
func TestT16_18_WriteProtection_SuperAdminAllowed(t *testing.T) {
	checker := &mockPublishChecker{published: map[string]bool{"prod-catalog": true}}
	e := echo.New()
	e.Use(RBACMiddleware(&HeaderRBACProvider{}))
	g := e.Group("/catalogs/:catalog-name")
	g.POST("/items", func(c echo.Context) error {
		return c.String(http.StatusCreated, "created")
	}, RequireRole(RoleRW), RequireWriteAccess(checker))

	req := httptest.NewRequest(http.MethodPost, "/catalogs/prod-catalog/items", nil)
	req.Header.Set("X-User-Role", "SuperAdmin")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

// T-16.26: Unpublished catalog — RW allowed
func TestT16_26_WriteProtection_UnpublishedRWAllowed(t *testing.T) {
	checker := &mockPublishChecker{published: map[string]bool{"dev-catalog": false}}
	e := echo.New()
	e.Use(RBACMiddleware(&HeaderRBACProvider{}))
	g := e.Group("/catalogs/:catalog-name")
	g.POST("/items", func(c echo.Context) error {
		return c.String(http.StatusCreated, "created")
	}, RequireRole(RoleRW), RequireWriteAccess(checker))

	req := httptest.NewRequest(http.MethodPost, "/catalogs/dev-catalog/items", nil)
	req.Header.Set("X-User-Role", "RW")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

// Write protection: no catalog name → passthrough
func TestWriteProtection_NoCatalogName(t *testing.T) {
	checker := &mockPublishChecker{}
	e := echo.New()
	e.Use(RBACMiddleware(&HeaderRBACProvider{}))
	e.POST("/items", func(c echo.Context) error {
		return c.String(http.StatusCreated, "created")
	}, RequireRole(RoleRW), RequireWriteAccess(checker))

	req := httptest.NewRequest(http.MethodPost, "/items", nil)
	req.Header.Set("X-User-Role", "RW")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

// Write protection: not-found error → passthrough to handler (handler returns 404)
func TestWriteProtection_NotFoundError(t *testing.T) {
	checker := &mockPublishChecker{returnErr: domainerrors.NewNotFound("Catalog", "missing")}
	e := echo.New()
	e.Use(RBACMiddleware(&HeaderRBACProvider{}))
	g := e.Group("/catalogs/:catalog-name")
	g.POST("/items", func(c echo.Context) error {
		return c.String(http.StatusCreated, "created")
	}, RequireRole(RoleRW), RequireWriteAccess(checker))

	req := httptest.NewRequest(http.MethodPost, "/catalogs/missing/items", nil)
	req.Header.Set("X-User-Role", "RW")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	// Not-found → passthrough; handler decides the response
	assert.Equal(t, http.StatusCreated, rec.Code)
}

// Write protection: not-found path executes same logic as not-published (timing consistency)
// The middleware should not have a separate early-return for not-found — both paths
// should go through the same conditional to prevent timing-based catalog existence probing.
func TestWriteProtection_NotFound_SamePathAsUnpublished(t *testing.T) {
	// Track how many times the handler is called and with what role context
	var handlerCalled int
	handler := func(c echo.Context) error {
		handlerCalled++
		return c.String(http.StatusCreated, "created")
	}

	// Test 1: not-found catalog
	checker1 := &mockPublishChecker{returnErr: domainerrors.NewNotFound("Catalog", "missing")}
	e1 := echo.New()
	e1.Use(RBACMiddleware(&HeaderRBACProvider{}))
	g1 := e1.Group("/catalogs/:catalog-name")
	g1.POST("/items", handler, RequireRole(RoleRW), RequireWriteAccess(checker1))

	handlerCalled = 0
	req1 := httptest.NewRequest(http.MethodPost, "/catalogs/missing/items", nil)
	req1.Header.Set("X-User-Role", "RW")
	rec1 := httptest.NewRecorder()
	e1.ServeHTTP(rec1, req1)
	assert.Equal(t, http.StatusCreated, rec1.Code)
	assert.Equal(t, 1, handlerCalled, "not-found should call handler exactly once")

	// Test 2: unpublished catalog — must behave identically
	checker2 := &mockPublishChecker{published: map[string]bool{"dev-catalog": false}}
	e2 := echo.New()
	e2.Use(RBACMiddleware(&HeaderRBACProvider{}))
	g2 := e2.Group("/catalogs/:catalog-name")
	g2.POST("/items", handler, RequireRole(RoleRW), RequireWriteAccess(checker2))

	handlerCalled = 0
	req2 := httptest.NewRequest(http.MethodPost, "/catalogs/dev-catalog/items", nil)
	req2.Header.Set("X-User-Role", "RW")
	rec2 := httptest.NewRecorder()
	e2.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusCreated, rec2.Code)
	assert.Equal(t, 1, handlerCalled, "unpublished should call handler exactly once")
}

// Write protection: genuine DB error → 500 (do NOT silently allow mutations)
func TestWriteProtection_DBError_Returns500(t *testing.T) {
	checker := &mockPublishChecker{returnErr: fmt.Errorf("connection refused")}
	e := echo.New()
	e.Use(RBACMiddleware(&HeaderRBACProvider{}))
	g := e.Group("/catalogs/:catalog-name")
	g.POST("/items", func(c echo.Context) error {
		return c.String(http.StatusCreated, "created")
	}, RequireRole(RoleRW), RequireWriteAccess(checker))

	req := httptest.NewRequest(http.MethodPost, "/catalogs/prod-catalog/items", nil)
	req.Header.Set("X-User-Role", "RW")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	// DB error must NOT passthrough — return 500 to prevent silent mutation of published catalogs
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
