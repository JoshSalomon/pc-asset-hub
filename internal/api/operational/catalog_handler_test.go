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
	svc := svcop.NewCatalogService(catRepo, cvRepo, instRepo, nil, "")
	accessChecker := &apimw.HeaderCatalogAccessChecker{}
	handler := apiop.NewCatalogHandler(svc, nil, accessChecker)

	e := echo.New()
	g := e.Group("/api/data/v1/catalogs")
	rbac := &apimw.HeaderRBACProvider{}
	g.Use(apimw.RBACMiddleware(rbac))
	requireRW := apimw.RequireRole(apimw.RoleRW)
	apiop.RegisterCatalogRoutes(g, handler, requireRW, apimw.RequireRole(apimw.RoleAdmin))

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

// TD-17: Pagination query params
func TestListCatalogs_WithPagination(t *testing.T) {
	e, catRepo, cvRepo, _ := setupCatalogServer()

	catRepo.On("List", mock.Anything, mock.MatchedBy(func(p models.ListParams) bool {
		return p.Limit == 5 && p.Offset == 10
	})).Return([]*models.Catalog{
		{ID: "c1", Name: "cat-a", CatalogVersionID: "cv1", ValidationStatus: models.ValidationStatusDraft},
	}, 20, nil)
	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{ID: "cv1", VersionLabel: "v1"}, nil)

	rec := doCatalogRequest(e, http.MethodGet, "/api/data/v1/catalogs?limit=5&offset=10", "", apimw.RoleRO)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestListCatalogs_LimitCappedAt100(t *testing.T) {
	e, catRepo, cvRepo, _ := setupCatalogServer()

	catRepo.On("List", mock.Anything, mock.MatchedBy(func(p models.ListParams) bool {
		return p.Limit == 100 // capped from 999
	})).Return([]*models.Catalog{}, 0, nil)
	_ = cvRepo

	rec := doCatalogRequest(e, http.MethodGet, "/api/data/v1/catalogs?limit=999", "", apimw.RoleRO)
	assert.Equal(t, http.StatusOK, rec.Code)
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
	svc := svcop.NewCatalogService(catRepo, cvRepo, instRepo, nil, "")
	handler := apiop.NewCatalogHandler(svc, nil, checker)

	e := echo.New()
	g := e.Group("/api/data/v1/catalogs")
	rbac := &apimw.HeaderRBACProvider{}
	g.Use(apimw.RBACMiddleware(rbac))
	requireRW := apimw.RequireRole(apimw.RoleRW)
	apiop.RegisterCatalogRoutes(g, handler, requireRW, apimw.RequireRole(apimw.RoleAdmin))

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

// === Publish/Unpublish Handler Tests ===

// T-16.29: Publish as Admin → 200
func TestT16_29_PublishAdmin(t *testing.T) {
	e, catRepo, cvRepo, _ := setupCatalogServer()
	catRepo.On("GetByName", mock.Anything, "my-catalog").Return(&models.Catalog{
		ID: "c1", Name: "my-catalog", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusValid,
	}, nil)
	catRepo.On("UpdatePublished", mock.Anything, "c1", true, mock.AnythingOfType("*time.Time")).Return(nil)
	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{ID: "cv1", VersionLabel: "v1"}, nil)

	rec := doCatalogRequest(e, http.MethodPost, "/api/data/v1/catalogs/my-catalog/publish", "", apimw.RoleAdmin)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// T-16.30: Publish as RW → 403
func TestT16_30_PublishRW(t *testing.T) {
	e, _, _, _ := setupCatalogServer()
	rec := doCatalogRequest(e, http.MethodPost, "/api/data/v1/catalogs/my-catalog/publish", "", apimw.RoleRW)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// T-16.31: Publish as RO → 403
func TestT16_31_PublishRO(t *testing.T) {
	e, _, _, _ := setupCatalogServer()
	rec := doCatalogRequest(e, http.MethodPost, "/api/data/v1/catalogs/my-catalog/publish", "", apimw.RoleRO)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// T-16.32: Publish draft catalog → 400
func TestT16_32_PublishDraft(t *testing.T) {
	e, catRepo, _, _ := setupCatalogServer()
	catRepo.On("GetByName", mock.Anything, "my-catalog").Return(&models.Catalog{
		ID: "c1", Name: "my-catalog", ValidationStatus: models.ValidationStatusDraft,
	}, nil)

	rec := doCatalogRequest(e, http.MethodPost, "/api/data/v1/catalogs/my-catalog/publish", "", apimw.RoleAdmin)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// T-16.33: Publish nonexistent → 404
func TestT16_33_PublishNotFound(t *testing.T) {
	e, catRepo, _, _ := setupCatalogServer()
	catRepo.On("GetByName", mock.Anything, "nonexistent").Return(nil, domainerrors.NewNotFound("Catalog", "nonexistent"))

	rec := doCatalogRequest(e, http.MethodPost, "/api/data/v1/catalogs/nonexistent/publish", "", apimw.RoleAdmin)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// T-16.34: Unpublish as Admin → 200
func TestT16_34_UnpublishAdmin(t *testing.T) {
	e, catRepo, _, _ := setupCatalogServer()
	catRepo.On("GetByName", mock.Anything, "my-catalog").Return(&models.Catalog{
		ID: "c1", Name: "my-catalog", Published: true,
	}, nil)
	catRepo.On("UpdatePublished", mock.Anything, "c1", false, (*time.Time)(nil)).Return(nil)

	rec := doCatalogRequest(e, http.MethodPost, "/api/data/v1/catalogs/my-catalog/unpublish", "", apimw.RoleAdmin)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// Unpublish nonexistent → 404
func TestUnpublish_NotFound(t *testing.T) {
	e, catRepo, _, _ := setupCatalogServer()
	catRepo.On("GetByName", mock.Anything, "nonexistent").Return(nil, domainerrors.NewNotFound("Catalog", "nonexistent"))
	rec := doCatalogRequest(e, http.MethodPost, "/api/data/v1/catalogs/nonexistent/unpublish", "", apimw.RoleAdmin)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// T-16.35: Unpublish as RW → 403
func TestT16_35_UnpublishRW(t *testing.T) {
	e, _, _, _ := setupCatalogServer()
	rec := doCatalogRequest(e, http.MethodPost, "/api/data/v1/catalogs/my-catalog/unpublish", "", apimw.RoleRW)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// T-16.38: Catalog response includes published fields
func TestT16_38_CatalogResponseHasPublishedFields(t *testing.T) {
	e, catRepo, cvRepo, _ := setupCatalogServer()
	now := time.Now()
	catRepo.On("GetByName", mock.Anything, "my-catalog").Return(&models.Catalog{
		ID: "c1", Name: "my-catalog", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusValid,
		Published: true, PublishedAt: &now,
		CreatedAt: now, UpdatedAt: now,
	}, nil)
	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{ID: "cv1", VersionLabel: "v1"}, nil)

	rec := doCatalogRequest(e, http.MethodGet, "/api/data/v1/catalogs/my-catalog", "", apimw.RoleAdmin)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, true, resp["published"])
	assert.NotNil(t, resp["published_at"])
}

// ---- Copy & Replace Handler Tests ----

func setupCatalogServerWithCopy() (*echo.Echo, *mocks.MockCatalogRepo, *mocks.MockCatalogVersionRepo, *mocks.MockEntityInstanceRepo, *mocks.MockInstanceAttributeValueRepo, *mocks.MockAssociationLinkRepo) {
	catRepo := new(mocks.MockCatalogRepo)
	cvRepo := new(mocks.MockCatalogVersionRepo)
	instRepo := new(mocks.MockEntityInstanceRepo)
	iavRepo := new(mocks.MockInstanceAttributeValueRepo)
	linkRepo := new(mocks.MockAssociationLinkRepo)
	txm := &mocks.MockTransactionManager{}
	svc := svcop.NewCatalogService(catRepo, cvRepo, instRepo, nil, "", svcop.WithCopyDeps(iavRepo, linkRepo), svcop.WithTransactionManager(txm))
	accessChecker := &apimw.HeaderCatalogAccessChecker{}
	handler := apiop.NewCatalogHandler(svc, nil, accessChecker)

	e := echo.New()
	g := e.Group("/api/data/v1/catalogs")
	rbac := &apimw.HeaderRBACProvider{}
	g.Use(apimw.RBACMiddleware(rbac))
	requireRW := apimw.RequireRole(apimw.RoleRW)
	apiop.RegisterCatalogRoutes(g, handler, requireRW, apimw.RequireRole(apimw.RoleAdmin))

	return e, catRepo, cvRepo, instRepo, iavRepo, linkRepo
}

// T-17.51: POST /catalogs/copy returns 201
func TestT17_51_CopyCatalog_Success(t *testing.T) {
	e, catRepo, cvRepo, instRepo, _, _ := setupCatalogServerWithCopy()

	catRepo.On("GetByName", mock.Anything, "source").Return(&models.Catalog{
		ID: "src-id", Name: "source", Description: "desc", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusValid,
	}, nil)
	catRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Catalog")).Return(nil)
	instRepo.On("ListByCatalog", mock.Anything, "src-id").Return([]*models.EntityInstance{}, nil)
	// For CV label resolution in response
	catRepo.On("GetByName", mock.Anything, "target").Return(&models.Catalog{
		ID: "new-id", Name: "target", CatalogVersionID: "cv1",
	}, nil)
	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{ID: "cv1", VersionLabel: "v1.0"}, nil)

	body := `{"source":"source","name":"target","description":"copied"}`
	rec := doCatalogRequest(e, http.MethodPost, "/api/data/v1/catalogs/copy", body, apimw.RoleRW)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp dto.CatalogResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "target", resp.Name)
	assert.Equal(t, "draft", resp.ValidationStatus)
	assert.Equal(t, "v1.0", resp.CatalogVersionLabel) // M4: CV label resolved
}

// T-17.53: Copy with nonexistent source → 404
func TestT17_53_CopyCatalog_SourceNotFound(t *testing.T) {
	e, catRepo, _, _, _, _ := setupCatalogServerWithCopy()

	catRepo.On("GetByName", mock.Anything, "nonexistent").Return(nil, domainerrors.NewNotFound("Catalog", "nonexistent"))

	body := `{"source":"nonexistent","name":"target"}`
	rec := doCatalogRequest(e, http.MethodPost, "/api/data/v1/catalogs/copy", body, apimw.RoleRW)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// T-17.54: Copy with duplicate target → 409
func TestT17_54_CopyCatalog_DuplicateTarget(t *testing.T) {
	e, catRepo, _, instRepo, _, _ := setupCatalogServerWithCopy()

	catRepo.On("GetByName", mock.Anything, "source").Return(&models.Catalog{
		ID: "src-id", Name: "source", CatalogVersionID: "cv1",
	}, nil)
	instRepo.On("ListByCatalog", mock.Anything, "src-id").Return([]*models.EntityInstance{}, nil)
	catRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Catalog")).Return(
		domainerrors.NewConflict("Catalog", "name already exists"),
	)

	body := `{"source":"source","name":"existing"}`
	rec := doCatalogRequest(e, http.MethodPost, "/api/data/v1/catalogs/copy", body, apimw.RoleRW)
	assert.Equal(t, http.StatusConflict, rec.Code)
}

// T-17.55: Copy with invalid target name → 400
func TestT17_55_CopyCatalog_InvalidName(t *testing.T) {
	e, catRepo, _, _, _, _ := setupCatalogServerWithCopy()

	catRepo.On("GetByName", mock.Anything, "source").Return(&models.Catalog{
		ID: "src-id", Name: "source", CatalogVersionID: "cv1",
	}, nil)

	body := `{"source":"source","name":"INVALID"}`
	rec := doCatalogRequest(e, http.MethodPost, "/api/data/v1/catalogs/copy", body, apimw.RoleRW)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// T-17.56: Copy as RO → 403
func TestT17_56_CopyCatalog_RO_Forbidden(t *testing.T) {
	e, _, _, _, _, _ := setupCatalogServerWithCopy()

	body := `{"source":"source","name":"target"}`
	rec := doCatalogRequest(e, http.MethodPost, "/api/data/v1/catalogs/copy", body, apimw.RoleRO)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// T-17.57: Copy as RW → 201
func TestT17_57_CopyCatalog_RW_Allowed(t *testing.T) {
	e, catRepo, cvRepo, instRepo, _, _ := setupCatalogServerWithCopy()

	catRepo.On("GetByName", mock.Anything, "source").Return(&models.Catalog{
		ID: "src-id", Name: "source", CatalogVersionID: "cv1",
	}, nil)
	catRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Catalog")).Return(nil)
	instRepo.On("ListByCatalog", mock.Anything, "src-id").Return([]*models.EntityInstance{}, nil)
	catRepo.On("GetByName", mock.Anything, "target").Return(&models.Catalog{ID: "new-id", Name: "target", CatalogVersionID: "cv1"}, nil)
	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{ID: "cv1", VersionLabel: "v1"}, nil)

	body := `{"source":"source","name":"target"}`
	rec := doCatalogRequest(e, http.MethodPost, "/api/data/v1/catalogs/copy", body, apimw.RoleRW)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

// T-17.59: POST /catalogs/replace returns 200
func TestT17_59_ReplaceCatalog_Success(t *testing.T) {
	e, catRepo, cvRepo, _, _, _ := setupCatalogServerWithCopy()

	catRepo.On("GetByName", mock.Anything, "staging").Return(&models.Catalog{
		ID: "src-id", Name: "staging", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusValid,
	}, nil)
	catRepo.On("GetByName", mock.Anything, "prod").Return(&models.Catalog{
		ID: "tgt-id", Name: "prod", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusValid,
	}, nil)
	catRepo.On("UpdateName", mock.Anything, "tgt-id", "prod-archive").Return(nil)
	catRepo.On("UpdateName", mock.Anything, "src-id", "prod").Return(nil)
	// For CV label resolution — after replace, source is named "prod"
	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{ID: "cv1", VersionLabel: "v1.0"}, nil)

	body := `{"source":"staging","target":"prod","archive_name":"prod-archive"}`
	rec := doCatalogRequest(e, http.MethodPost, "/api/data/v1/catalogs/replace", body, apimw.RoleAdmin)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp dto.CatalogResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "prod", resp.Name) // M1: correct name in response
}

// T-17.60: Replace with non-valid source → 400
func TestT17_60_ReplaceCatalog_InvalidSource(t *testing.T) {
	e, catRepo, _, _, _, _ := setupCatalogServerWithCopy()

	catRepo.On("GetByName", mock.Anything, "staging").Return(&models.Catalog{
		ID: "src-id", Name: "staging", ValidationStatus: models.ValidationStatusDraft,
	}, nil)
	catRepo.On("GetByName", mock.Anything, "prod").Return(&models.Catalog{
		ID: "tgt-id", Name: "prod",
	}, nil)

	body := `{"source":"staging","target":"prod"}`
	rec := doCatalogRequest(e, http.MethodPost, "/api/data/v1/catalogs/replace", body, apimw.RoleAdmin)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// T-17.64: Replace as RO → 403
func TestT17_64_ReplaceCatalog_RO_Forbidden(t *testing.T) {
	e, _, _, _, _, _ := setupCatalogServerWithCopy()

	body := `{"source":"staging","target":"prod"}`
	rec := doCatalogRequest(e, http.MethodPost, "/api/data/v1/catalogs/replace", body, apimw.RoleRO)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// T-17.65: Replace as RW → 403
func TestT17_65_ReplaceCatalog_RW_Forbidden(t *testing.T) {
	e, _, _, _, _, _ := setupCatalogServerWithCopy()

	body := `{"source":"staging","target":"prod"}`
	rec := doCatalogRequest(e, http.MethodPost, "/api/data/v1/catalogs/replace", body, apimw.RoleRW)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// --- Copy/Replace access checker error coverage ---

func setupCopyServerWithAccessChecker(checker apimw.CatalogAccessChecker) (*echo.Echo, *mocks.MockCatalogRepo, *mocks.MockCatalogVersionRepo, *mocks.MockEntityInstanceRepo) {
	catRepo := new(mocks.MockCatalogRepo)
	cvRepo := new(mocks.MockCatalogVersionRepo)
	instRepo := new(mocks.MockEntityInstanceRepo)
	iavRepo := new(mocks.MockInstanceAttributeValueRepo)
	linkRepo := new(mocks.MockAssociationLinkRepo)
	txm := &mocks.MockTransactionManager{}
	svc := svcop.NewCatalogService(catRepo, cvRepo, instRepo, nil, "", svcop.WithCopyDeps(iavRepo, linkRepo), svcop.WithTransactionManager(txm))
	handler := apiop.NewCatalogHandler(svc, nil, checker)

	e := echo.New()
	g := e.Group("/api/data/v1/catalogs")
	rbac := &apimw.HeaderRBACProvider{}
	g.Use(apimw.RBACMiddleware(rbac))
	requireRW := apimw.RequireRole(apimw.RoleRW)
	apiop.RegisterCatalogRoutes(g, handler, requireRW, apimw.RequireRole(apimw.RoleAdmin))
	return e, catRepo, cvRepo, instRepo
}

// Copy: source access denied → 403
func TestCopyCatalog_SourceAccessDenied(t *testing.T) {
	checker := &mockAccessChecker{allowed: map[string]bool{"source": false, "target": true}}
	e, _, _, _ := setupCopyServerWithAccessChecker(checker)

	body := `{"source":"source","name":"target"}`
	rec := doCatalogRequest(e, http.MethodPost, "/api/data/v1/catalogs/copy", body, apimw.RoleRW)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// Copy: target access denied → 403
func TestCopyCatalog_TargetAccessDenied(t *testing.T) {
	checker := &mockAccessChecker{allowed: map[string]bool{"source": true, "target": false}}
	e, _, _, _ := setupCopyServerWithAccessChecker(checker)

	body := `{"source":"source","name":"target"}`
	rec := doCatalogRequest(e, http.MethodPost, "/api/data/v1/catalogs/copy", body, apimw.RoleRW)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// Replace: source access denied → 403
func TestReplaceCatalog_SourceAccessDenied(t *testing.T) {
	checker := &mockAccessChecker{allowed: map[string]bool{"staging": false, "prod": true}}
	e, _, _, _ := setupCopyServerWithAccessChecker(checker)

	body := `{"source":"staging","target":"prod"}`
	rec := doCatalogRequest(e, http.MethodPost, "/api/data/v1/catalogs/replace", body, apimw.RoleAdmin)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// Replace: target access denied → 403
func TestReplaceCatalog_TargetAccessDenied(t *testing.T) {
	checker := &mockAccessChecker{allowed: map[string]bool{"staging": true, "prod": false}}
	e, _, _, _ := setupCopyServerWithAccessChecker(checker)

	body := `{"source":"staging","target":"prod"}`
	rec := doCatalogRequest(e, http.MethodPost, "/api/data/v1/catalogs/replace", body, apimw.RoleAdmin)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// errorAccessChecker always returns an error
type errorAccessChecker struct{}

func (e *errorAccessChecker) CheckAccess(_ echo.Context, _ string, _ string) (bool, error) {
	return false, fmt.Errorf("access check error")
}

// Copy: access checker error → 500
func TestCopyCatalog_AccessCheckError(t *testing.T) {
	e, _, _, _ := setupCopyServerWithAccessChecker(&errorAccessChecker{})

	body := `{"source":"source","name":"target"}`
	rec := doCatalogRequest(e, http.MethodPost, "/api/data/v1/catalogs/copy", body, apimw.RoleRW)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// Replace: access checker error → 500
func TestReplaceCatalog_AccessCheckError(t *testing.T) {
	e, _, _, _ := setupCopyServerWithAccessChecker(&errorAccessChecker{})

	body := `{"source":"staging","target":"prod"}`
	rec := doCatalogRequest(e, http.MethodPost, "/api/data/v1/catalogs/replace", body, apimw.RoleAdmin)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// targetErrorAccessChecker allows source but errors on target
type targetErrorAccessChecker struct{}

func (t *targetErrorAccessChecker) CheckAccess(_ echo.Context, name string, _ string) (bool, error) {
	if name == "source" || name == "staging" {
		return true, nil
	}
	return false, fmt.Errorf("access check error for target")
}

// Copy: target access check error → 500
func TestCopyCatalog_TargetAccessCheckError(t *testing.T) {
	e, _, _, _ := setupCopyServerWithAccessChecker(&targetErrorAccessChecker{})

	body := `{"source":"source","name":"target"}`
	rec := doCatalogRequest(e, http.MethodPost, "/api/data/v1/catalogs/copy", body, apimw.RoleRW)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// Replace: target access check error → 500
func TestReplaceCatalog_TargetAccessCheckError(t *testing.T) {
	e, _, _, _ := setupCopyServerWithAccessChecker(&targetErrorAccessChecker{})

	body := `{"source":"staging","target":"prod"}`
	rec := doCatalogRequest(e, http.MethodPost, "/api/data/v1/catalogs/replace", body, apimw.RoleAdmin)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// Copy: CV label resolution failure — fallback to empty label
func TestCopyCatalog_CVLabelFallback(t *testing.T) {
	e, catRepo, cvRepo, instRepo, _, _ := setupCatalogServerWithCopy()

	catRepo.On("GetByName", mock.Anything, "source").Return(&models.Catalog{
		ID: "src-id", Name: "source", CatalogVersionID: "cv1",
	}, nil)
	catRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Catalog")).Return(nil)
	instRepo.On("ListByCatalog", mock.Anything, "src-id").Return([]*models.EntityInstance{}, nil)
	// GetByName for CV label resolution fails
	catRepo.On("GetByName", mock.Anything, "target").Return(nil, fmt.Errorf("not found"))
	_ = cvRepo // not called since GetByName fails first

	body := `{"source":"source","name":"target"}`
	rec := doCatalogRequest(e, http.MethodPost, "/api/data/v1/catalogs/copy", body, apimw.RoleRW)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp dto.CatalogResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "", resp.CatalogVersionLabel) // fallback
}

// Replace: CV label resolution failure — fallback to empty label
func TestReplaceCatalog_CVLabelFallback(t *testing.T) {
	e, catRepo, cvRepo, _, _, _ := setupCatalogServerWithCopy()

	catRepo.On("GetByName", mock.Anything, "staging").Return(&models.Catalog{
		ID: "src-id", Name: "staging", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusValid,
	}, nil)
	catRepo.On("GetByName", mock.Anything, "prod").Return(&models.Catalog{
		ID: "tgt-id", Name: "prod", CatalogVersionID: "cv1",
	}, nil)
	catRepo.On("UpdateName", mock.Anything, "tgt-id", "prod-archive").Return(nil)
	catRepo.On("UpdateName", mock.Anything, "src-id", "prod").Return(nil)
	// CV label resolution will fail — cvRepo.GetByID returns error
	cvRepo.On("GetByID", mock.Anything, "cv1").Return(nil, fmt.Errorf("cv not found"))

	body := `{"source":"staging","target":"prod","archive_name":"prod-archive"}`
	rec := doCatalogRequest(e, http.MethodPost, "/api/data/v1/catalogs/replace", body, apimw.RoleAdmin)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp dto.CatalogResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "", resp.CatalogVersionLabel) // fallback
}
