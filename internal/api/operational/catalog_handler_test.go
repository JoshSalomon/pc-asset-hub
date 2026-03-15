package operational_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/project-catalyst/pc-asset-hub/internal/api/dto"
	apimw "github.com/project-catalyst/pc-asset-hub/internal/api/middleware"
	apiop "github.com/project-catalyst/pc-asset-hub/internal/api/operational"
	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository/mocks"
	svcop "github.com/project-catalyst/pc-asset-hub/internal/service/operational"
)

func setupCatalogServer() (*echo.Echo, *mocks.MockCatalogRepo, *mocks.MockCatalogVersionRepo, *mocks.MockEntityInstanceRepo) {
	catRepo := new(mocks.MockCatalogRepo)
	cvRepo := new(mocks.MockCatalogVersionRepo)
	instRepo := new(mocks.MockEntityInstanceRepo)
	svc := svcop.NewCatalogService(catRepo, cvRepo, instRepo)
	accessChecker := &apimw.HeaderCatalogAccessChecker{}
	handler := apiop.NewCatalogHandler(svc, accessChecker)

	e := echo.New()
	g := e.Group("/api/data/v1/catalogs")
	rbac := &apimw.HeaderRBACProvider{}
	g.Use(apimw.RBACMiddleware(rbac))
	requireRW := apimw.RequireRole(apimw.RoleRW)
	apiop.RegisterCatalogRoutes(g, handler, requireRW)

	return e, catRepo, cvRepo, instRepo
}

func doCatalogRequest(e *echo.Echo, method, path, body string, role apimw.Role) *httptest.ResponseRecorder {
	reader := strings.NewReader(body)
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("X-User-Role", string(role))
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// T-10.27: POST /api/data/v1/catalogs with valid request
func TestT10_27_CreateCatalogSuccess(t *testing.T) {
	e, catRepo, cvRepo, _ := setupCatalogServer()

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{ID: "cv1", VersionLabel: "v1"}, nil)
	catRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Catalog")).Return(nil)

	rec := doCatalogRequest(e, http.MethodPost, "/api/data/v1/catalogs",
		`{"name":"my-catalog","description":"test","catalog_version_id":"cv1"}`, apimw.RoleRW)

	assert.Equal(t, http.StatusCreated, rec.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "my-catalog", resp["name"])
	assert.Equal(t, "draft", resp["validation_status"])
}

// T-10.28: POST /api/data/v1/catalogs with invalid name format
func TestT10_28_CreateInvalidName(t *testing.T) {
	e, _, _, _ := setupCatalogServer()

	rec := doCatalogRequest(e, http.MethodPost, "/api/data/v1/catalogs",
		`{"name":"My-Catalog!","catalog_version_id":"cv1"}`, apimw.RoleRW)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// T-10.29: POST /api/data/v1/catalogs with duplicate name
func TestT10_29_CreateDuplicateName(t *testing.T) {
	e, catRepo, cvRepo, _ := setupCatalogServer()

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{ID: "cv1", VersionLabel: "v1"}, nil)
	catRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Catalog")).Return(domainerrors.NewConflict("Catalog", "exists"))

	rec := doCatalogRequest(e, http.MethodPost, "/api/data/v1/catalogs",
		`{"name":"existing","catalog_version_id":"cv1"}`, apimw.RoleRW)

	assert.Equal(t, http.StatusConflict, rec.Code)
}

// T-10.30: POST /api/data/v1/catalogs with nonexistent CV
func TestT10_30_CreateNonexistentCV(t *testing.T) {
	e, _, cvRepo, _ := setupCatalogServer()

	cvRepo.On("GetByID", mock.Anything, "bad-cv").Return(nil, domainerrors.NewNotFound("CatalogVersion", "bad-cv"))

	rec := doCatalogRequest(e, http.MethodPost, "/api/data/v1/catalogs",
		`{"name":"my-catalog","catalog_version_id":"bad-cv"}`, apimw.RoleRW)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// T-10.31: POST /api/data/v1/catalogs as RO
func TestT10_31_CreateAsRO(t *testing.T) {
	e, _, _, _ := setupCatalogServer()

	rec := doCatalogRequest(e, http.MethodPost, "/api/data/v1/catalogs",
		`{"name":"my-catalog","catalog_version_id":"cv1"}`, apimw.RoleRO)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// T-10.32: POST /api/data/v1/catalogs as RW
func TestT10_32_CreateAsRW(t *testing.T) {
	e, catRepo, cvRepo, _ := setupCatalogServer()

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{ID: "cv1", VersionLabel: "v1"}, nil)
	catRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Catalog")).Return(nil)

	rec := doCatalogRequest(e, http.MethodPost, "/api/data/v1/catalogs",
		`{"name":"rw-catalog","catalog_version_id":"cv1"}`, apimw.RoleRW)

	assert.Equal(t, http.StatusCreated, rec.Code)
}

// T-10.33: GET /api/data/v1/catalogs returns list with CV labels
func TestT10_33_ListCatalogs(t *testing.T) {
	e, catRepo, cvRepo, _ := setupCatalogServer()

	now := time.Now()
	catRepo.On("List", mock.Anything, mock.AnythingOfType("models.ListParams")).Return([]*models.Catalog{
		{ID: "c1", Name: "cat-a", CatalogVersionID: "cv1", ValidationStatus: models.ValidationStatusDraft, CreatedAt: now, UpdatedAt: now},
		{ID: "c2", Name: "cat-b", CatalogVersionID: "cv1", ValidationStatus: models.ValidationStatusValid, CreatedAt: now, UpdatedAt: now},
	}, 2, nil)
	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{ID: "cv1", VersionLabel: "release-1"}, nil)

	rec := doCatalogRequest(e, http.MethodGet, "/api/data/v1/catalogs", "", apimw.RoleRO)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, float64(2), resp["total"])
	items := resp["items"].([]any)
	assert.Len(t, items, 2)
	// Verify CV label is resolved
	first := items[0].(map[string]any)
	assert.Equal(t, "release-1", first["catalog_version_label"])
}

// T-10.34: GET /api/data/v1/catalogs?catalog_version_id=X
func TestT10_34_ListFilterByCVID(t *testing.T) {
	e, catRepo, cvRepo, _ := setupCatalogServer()

	catRepo.On("List", mock.Anything, mock.MatchedBy(func(p models.ListParams) bool {
		return p.Filters["catalog_version_id"] == "cv1"
	})).Return([]*models.Catalog{
		{ID: "c1", Name: "cat-a", CatalogVersionID: "cv1", ValidationStatus: models.ValidationStatusDraft},
	}, 1, nil)
	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{ID: "cv1", VersionLabel: "v1"}, nil)

	rec := doCatalogRequest(e, http.MethodGet, "/api/data/v1/catalogs?catalog_version_id=cv1", "", apimw.RoleRO)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, float64(1), resp["total"])
}

// T-10.35: GET /api/data/v1/catalogs?validation_status=draft
func TestT10_35_ListFilterByStatus(t *testing.T) {
	e, catRepo, cvRepo, _ := setupCatalogServer()

	catRepo.On("List", mock.Anything, mock.MatchedBy(func(p models.ListParams) bool {
		return p.Filters["validation_status"] == "draft"
	})).Return([]*models.Catalog{
		{ID: "c1", Name: "cat-a", CatalogVersionID: "cv1", ValidationStatus: models.ValidationStatusDraft},
	}, 1, nil)
	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{ID: "cv1", VersionLabel: "v1"}, nil)

	rec := doCatalogRequest(e, http.MethodGet, "/api/data/v1/catalogs?validation_status=draft", "", apimw.RoleRO)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// Handler: ListCatalogs repo error returns 500
func TestCatalogHandler_ListError(t *testing.T) {
	e, catRepo, _, _ := setupCatalogServer()

	catRepo.On("List", mock.Anything, mock.AnythingOfType("models.ListParams")).Return(
		nil, 0, fmt.Errorf("db connection failed"))

	rec := doCatalogRequest(e, http.MethodGet, "/api/data/v1/catalogs", "", apimw.RoleRO)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// Handler: CreateCatalog bind error
func TestCatalogHandler_CreateBindError(t *testing.T) {
	e, _, _, _ := setupCatalogServer()

	rec := doCatalogRequest(e, http.MethodPost, "/api/data/v1/catalogs", "not-json{{{", apimw.RoleRW)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// T-10.36: GET /api/data/v1/catalogs/{name} returns detail
func TestT10_36_GetCatalog(t *testing.T) {
	e, catRepo, cvRepo, _ := setupCatalogServer()

	catRepo.On("GetByName", mock.Anything, "my-catalog").Return(&models.Catalog{
		ID: "c1", Name: "my-catalog", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusDraft,
	}, nil)
	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{ID: "cv1", VersionLabel: "release-1.0"}, nil)

	rec := doCatalogRequest(e, http.MethodGet, "/api/data/v1/catalogs/my-catalog", "", apimw.RoleRO)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "my-catalog", resp["name"])
	assert.Equal(t, "release-1.0", resp["catalog_version_label"])
}

// T-10.37: GET /api/data/v1/catalogs/{name} nonexistent
func TestT10_37_GetCatalogNotFound(t *testing.T) {
	e, catRepo, _, _ := setupCatalogServer()

	catRepo.On("GetByName", mock.Anything, "nonexistent").Return(nil, domainerrors.NewNotFound("Catalog", "nonexistent"))

	rec := doCatalogRequest(e, http.MethodGet, "/api/data/v1/catalogs/nonexistent", "", apimw.RoleRO)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// T-10.38: DELETE /api/data/v1/catalogs/{name}
func TestT10_38_DeleteCatalog(t *testing.T) {
	e, catRepo, _, instRepo := setupCatalogServer()

	catRepo.On("GetByName", mock.Anything, "my-catalog").Return(&models.Catalog{ID: "c1", Name: "my-catalog"}, nil)
	instRepo.On("DeleteByCatalogID", mock.Anything, "c1").Return(nil)
	catRepo.On("Delete", mock.Anything, "c1").Return(nil)

	rec := doCatalogRequest(e, http.MethodDelete, "/api/data/v1/catalogs/my-catalog", "", apimw.RoleRW)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

// T-10.39: DELETE /api/data/v1/catalogs/{name} as RO
func TestT10_39_DeleteAsRO(t *testing.T) {
	e, _, _, _ := setupCatalogServer()

	rec := doCatalogRequest(e, http.MethodDelete, "/api/data/v1/catalogs/my-catalog", "", apimw.RoleRO)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// T-10.40: DELETE /api/data/v1/catalogs/{name} nonexistent
func TestT10_40_DeleteNotFound(t *testing.T) {
	e, catRepo, _, _ := setupCatalogServer()

	catRepo.On("GetByName", mock.Anything, "nonexistent").Return(nil, domainerrors.NewNotFound("Catalog", "nonexistent"))

	rec := doCatalogRequest(e, http.MethodDelete, "/api/data/v1/catalogs/nonexistent", "", apimw.RoleRW)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// T-10.51: Old CV-scoped routes no longer registered
func TestT10_51_OldRoutesNotRegistered(t *testing.T) {
	e, _, _, _ := setupCatalogServer()

	rec := doCatalogRequest(e, http.MethodGet, "/api/catalog/cv1/models", "", apimw.RoleRO)

	// Should get 404 or 405 since these routes aren't on this server
	assert.NotEqual(t, http.StatusOK, rec.Code)
}

// === Phase 5: Catalog-Level RBAC API Tests ===

// mockAccessChecker allows per-catalog access control in tests.
type mockAccessChecker struct {
	allowed map[string]bool
}

func (m *mockAccessChecker) CheckAccess(_ echo.Context, catalogName, _ string) (bool, error) {
	if m.allowed == nil {
		return true, nil
	}
	return m.allowed[catalogName], nil
}

func setupCatalogServerWithAccessChecker(checker apimw.CatalogAccessChecker) (*echo.Echo, *mocks.MockCatalogRepo, *mocks.MockCatalogVersionRepo) {
	catRepo := new(mocks.MockCatalogRepo)
	cvRepo := new(mocks.MockCatalogVersionRepo)
	instRepo := new(mocks.MockEntityInstanceRepo)
	svc := svcop.NewCatalogService(catRepo, cvRepo, instRepo)
	handler := apiop.NewCatalogHandler(svc, checker)

	e := echo.New()
	g := e.Group("/api/data/v1/catalogs")
	rbac := &apimw.HeaderRBACProvider{}
	g.Use(apimw.RBACMiddleware(rbac))
	requireRW := apimw.RequireRole(apimw.RoleRW)
	apiop.RegisterCatalogRoutes(g, handler, requireRW)

	return e, catRepo, cvRepo
}

// T-14.15: Catalog list with mock deny returns filtered list
func TestT14_15_CatalogListFiltersDenied(t *testing.T) {
	checker := &mockAccessChecker{allowed: map[string]bool{
		"allowed-cat": true, "denied-cat": false,
	}}
	e, catRepo, cvRepo := setupCatalogServerWithAccessChecker(checker)

	now := time.Now()
	catRepo.On("List", mock.Anything, mock.Anything).Return([]*models.Catalog{
		{ID: "c1", Name: "allowed-cat", CatalogVersionID: "cv1", ValidationStatus: models.ValidationStatusDraft, CreatedAt: now, UpdatedAt: now},
		{ID: "c2", Name: "denied-cat", CatalogVersionID: "cv1", ValidationStatus: models.ValidationStatusDraft, CreatedAt: now, UpdatedAt: now},
	}, 2, nil)
	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{ID: "cv1", VersionLabel: "v1"}, nil)

	rec := doCatalogRequest(e, http.MethodGet, "/api/data/v1/catalogs", "", apimw.RoleRO)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp dto.ListResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, 1, resp.Total)

	items, _ := json.Marshal(resp.Items)
	assert.Contains(t, string(items), "allowed-cat")
	assert.NotContains(t, string(items), "denied-cat")
}

// T-14.16: Header mode returns all catalogs
func TestT14_16_HeaderModeReturnsAll(t *testing.T) {
	e, catRepo, cvRepo, _ := setupCatalogServer() // uses HeaderCatalogAccessChecker

	now := time.Now()
	catRepo.On("List", mock.Anything, mock.Anything).Return([]*models.Catalog{
		{ID: "c1", Name: "cat-a", CatalogVersionID: "cv1", ValidationStatus: models.ValidationStatusDraft, CreatedAt: now, UpdatedAt: now},
		{ID: "c2", Name: "cat-b", CatalogVersionID: "cv1", ValidationStatus: models.ValidationStatusDraft, CreatedAt: now, UpdatedAt: now},
	}, 2, nil)
	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{ID: "cv1", VersionLabel: "v1"}, nil)

	rec := doCatalogRequest(e, http.MethodGet, "/api/data/v1/catalogs", "", apimw.RoleRO)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp dto.ListResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, 2, resp.Total)
}

// T-14.17: GET catalog returns 403 when access denied
func TestT14_17_GetCatalog_DeniedReturns403(t *testing.T) {
	checker := &mockAccessChecker{allowed: map[string]bool{"denied-cat": false}}
	e, catRepo, _ := setupCatalogServerWithAccessChecker(checker)

	now := time.Now()
	catRepo.On("GetByName", mock.Anything, "denied-cat").Return(&models.Catalog{
		ID: "c1", Name: "denied-cat", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusDraft, CreatedAt: now, UpdatedAt: now,
	}, nil)

	rec := doCatalogRequest(e, http.MethodGet, "/api/data/v1/catalogs/denied-cat", "", apimw.RoleRO)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// T-14.18: DELETE catalog returns 403 when access denied
func TestT14_18_DeleteCatalog_DeniedReturns403(t *testing.T) {
	checker := &mockAccessChecker{allowed: map[string]bool{"denied-cat": false}}
	e, _, _ := setupCatalogServerWithAccessChecker(checker)

	rec := doCatalogRequest(e, http.MethodDelete, "/api/data/v1/catalogs/denied-cat", "", apimw.RoleRW)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// T-14.19: CreateCatalog returns 403 when access denied for that catalog name
func TestT14_19_CreateCatalog_DeniedReturns403(t *testing.T) {
	checker := &mockAccessChecker{allowed: map[string]bool{"forbidden-cat": false}}
	e, _, _ := setupCatalogServerWithAccessChecker(checker)

	body := `{"name":"forbidden-cat","catalog_version_id":"cv1"}`
	rec := doCatalogRequest(e, http.MethodPost, "/api/data/v1/catalogs", body, apimw.RoleRW)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}
