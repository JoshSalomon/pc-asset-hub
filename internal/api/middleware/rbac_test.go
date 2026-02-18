package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeaderRBACProvider_ValidRoles(t *testing.T) {
	provider := &HeaderRBACProvider{}
	for _, role := range []Role{RoleRO, RoleRW, RoleAdmin, RoleSuperAdmin} {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-User-Role", string(role))
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		got, err := provider.GetRole(c)
		require.NoError(t, err)
		assert.Equal(t, role, got)
	}
}

func TestHeaderRBACProvider_MissingHeader(t *testing.T) {
	provider := &HeaderRBACProvider{}
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_, err := provider.GetRole(c)
	require.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, he.Code)
}

func TestHeaderRBACProvider_InvalidRole(t *testing.T) {
	provider := &HeaderRBACProvider{}
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-User-Role", "InvalidRole")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_, err := provider.GetRole(c)
	require.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, he.Code)
}

func TestRBACMiddleware_SetsRole(t *testing.T) {
	provider := &HeaderRBACProvider{}
	mw := RBACMiddleware(provider)
	e := echo.New()

	var capturedRole Role
	handler := mw(func(c echo.Context) error {
		capturedRole = c.Get(RoleContextKey).(Role)
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-User-Role", "Admin")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, RoleAdmin, capturedRole)
}

func TestRBACMiddleware_ReturnsErrorOnBadRole(t *testing.T) {
	provider := &HeaderRBACProvider{}
	mw := RBACMiddleware(provider)
	e := echo.New()

	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler(c)
	require.Error(t, err)
}

func TestRequireRole_Sufficient(t *testing.T) {
	mw := RequireRole(RoleRW)
	e := echo.New()

	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(RoleContextKey, RoleAdmin) // Admin >= RW

	err := handler(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireRole_Insufficient(t *testing.T) {
	mw := RequireRole(RoleAdmin)
	e := echo.New()

	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set(RoleContextKey, RoleRO) // RO < Admin

	err := handler(c)
	require.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusForbidden, he.Code)
}

func TestRequireRole_NoRoleInContext(t *testing.T) {
	mw := RequireRole(RoleRO)
	e := echo.New()

	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// Don't set any role

	err := handler(c)
	require.Error(t, err)
	he, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, he.Code)
}

func TestGetRoleFromContext(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// No role set
	assert.Equal(t, Role(""), GetRoleFromContext(c))

	// With role
	c.Set(RoleContextKey, RoleAdmin)
	assert.Equal(t, RoleAdmin, GetRoleFromContext(c))
}

func TestHasMinRole(t *testing.T) {
	tests := []struct {
		actual   Role
		required Role
		expected bool
	}{
		{RoleRO, RoleRO, true},
		{RoleRW, RoleRO, true},
		{RoleAdmin, RoleRO, true},
		{RoleSuperAdmin, RoleRO, true},
		{RoleRO, RoleRW, false},
		{RoleRO, RoleAdmin, false},
		{RoleRO, RoleSuperAdmin, false},
		{RoleRW, RoleRW, true},
		{RoleRW, RoleAdmin, false},
		{RoleAdmin, RoleAdmin, true},
		{RoleAdmin, RoleSuperAdmin, false},
		{RoleSuperAdmin, RoleSuperAdmin, true},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.expected, hasMinRole(tt.actual, tt.required),
			"hasMinRole(%s, %s)", tt.actual, tt.required)
	}
}
