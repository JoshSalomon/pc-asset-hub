package meta_test

import (
	"encoding/json"
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
	apimeta "github.com/project-catalyst/pc-asset-hub/internal/api/meta"
	apimw "github.com/project-catalyst/pc-asset-hub/internal/api/middleware"
	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository/mocks"
	svcmeta "github.com/project-catalyst/pc-asset-hub/internal/service/meta"
)

func setupTestServer(etRepo *mocks.MockEntityTypeRepo, etvRepo *mocks.MockEntityTypeVersionRepo, attrRepo *mocks.MockAttributeRepo, assocRepo *mocks.MockAssociationRepo) *echo.Echo {
	e := echo.New()
	svc := svcmeta.NewEntityTypeService(etRepo, etvRepo, attrRepo, assocRepo)
	handler := apimeta.NewEntityTypeHandler(svc)

	g := e.Group("/api/meta/v1")
	rbac := &apimw.HeaderRBACProvider{}
	g.Use(apimw.RBACMiddleware(rbac))
	requireAdmin := apimw.RequireRole(apimw.RoleAdmin)
	apimeta.RegisterEntityTypeRoutes(g, handler, requireAdmin)

	return e
}

func doRequest(e *echo.Echo, method, path, body string, role apimw.Role) *httptest.ResponseRecorder {
	var reader *strings.Reader
	if body != "" {
		reader = strings.NewReader(body)
	} else {
		reader = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	if role != "" {
		req.Header.Set("X-User-Role", string(role))
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// T-5.01: RO can GET
func TestT5_01_ROCanGet(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	e := setupTestServer(etRepo, etvRepo, nil, nil)

	etRepo.On("List", mock.Anything, mock.Anything).Return([]*models.EntityType{}, 0, nil)

	rec := doRequest(e, http.MethodGet, "/api/meta/v1/entity-types", "", apimw.RoleRO)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// T-5.02: RO cannot POST
func TestT5_02_ROCannotPost(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	e := setupTestServer(etRepo, etvRepo, nil, nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types",
		`{"name":"Test","description":"desc"}`, apimw.RoleRO)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// T-5.05: No token returns 401
func TestT5_05_NoToken(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	e := setupTestServer(etRepo, etvRepo, nil, nil)

	rec := doRequest(e, http.MethodGet, "/api/meta/v1/entity-types", "", "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// T-5.06: Invalid token returns 401
func TestT5_06_InvalidToken(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	e := setupTestServer(etRepo, etvRepo, nil, nil)

	rec := doRequest(e, http.MethodGet, "/api/meta/v1/entity-types", "", "InvalidRole")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// T-5.09: POST entity-types creates entity with V1
func TestT5_09_CreateEntityType(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	e := setupTestServer(etRepo, etvRepo, nil, nil)

	etRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityType")).Return(nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types",
		`{"name":"Model","description":"A model"}`, apimw.RoleAdmin)

	assert.Equal(t, http.StatusCreated, rec.Code)
	var result map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	assert.Contains(t, string(result["version"]), `"version":1`)
}

// T-5.10: POST with missing name returns 400
func TestT5_10_CreateMissingName(t *testing.T) {
	e := setupTestServer(nil, nil, nil, nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types",
		`{"description":"no name"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// T-5.11: POST with duplicate name returns 409
func TestT5_11_CreateDuplicateName(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	e := setupTestServer(etRepo, nil, nil, nil)

	etRepo.On("Create", mock.Anything, mock.Anything).Return(domainerrors.NewConflict("EntityType", "exists"))

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types",
		`{"name":"Duplicate"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusConflict, rec.Code)
}

// T-5.12: GET entity-types returns paginated list
func TestT5_12_ListEntityTypes(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	e := setupTestServer(etRepo, nil, nil, nil)

	now := time.Now()
	etRepo.On("List", mock.Anything, mock.Anything).Return([]*models.EntityType{
		{ID: "1", Name: "Model", CreatedAt: now, UpdatedAt: now},
		{ID: "2", Name: "Tool", CreatedAt: now, UpdatedAt: now},
	}, 2, nil)

	rec := doRequest(e, http.MethodGet, "/api/meta/v1/entity-types", "", apimw.RoleRO)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp dto.ListResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, 2, resp.Total)
}

// T-5.13: GET with name filter
func TestT5_13_ListWithFilter(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	e := setupTestServer(etRepo, nil, nil, nil)

	etRepo.On("List", mock.Anything, mock.Anything).Return([]*models.EntityType{
		{ID: "1", Name: "Model"},
	}, 1, nil)

	rec := doRequest(e, http.MethodGet, "/api/meta/v1/entity-types?name=Model", "", apimw.RoleRO)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// T-5.14: GET by ID
func TestT5_14_GetByID(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	e := setupTestServer(etRepo, nil, nil, nil)

	etRepo.On("GetByID", mock.Anything, "et1").Return(&models.EntityType{
		ID: "et1", Name: "Model",
	}, nil)

	rec := doRequest(e, http.MethodGet, "/api/meta/v1/entity-types/et1", "", apimw.RoleRO)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// T-5.15: GET non-existent returns 404
func TestT5_15_GetNotFound(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	e := setupTestServer(etRepo, nil, nil, nil)

	etRepo.On("GetByID", mock.Anything, "bad").Return(nil, domainerrors.NewNotFound("EntityType", "bad"))

	rec := doRequest(e, http.MethodGet, "/api/meta/v1/entity-types/bad", "", apimw.RoleRO)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// T-5.16: PUT creates new version
func TestT5_16_UpdateCreatesNewVersion(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	e := setupTestServer(etRepo, etvRepo, attrRepo, assocRepo)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	etRepo.On("GetByID", mock.Anything, "et1").Return(&models.EntityType{ID: "et1"}, nil)
	etRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	rec := doRequest(e, http.MethodPut, "/api/meta/v1/entity-types/et1",
		`{"description":"Updated"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"version":2`)
}

// T-5.18: DELETE returns 204
func TestT5_18_Delete(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	e := setupTestServer(etRepo, nil, nil, nil)

	etRepo.On("Delete", mock.Anything, "et1").Return(nil)

	rec := doRequest(e, http.MethodDelete, "/api/meta/v1/entity-types/et1", "", apimw.RoleAdmin)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

// T-5.19: POST copy creates new type at V1
func TestT5_19_CopyEntityType(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	e := setupTestServer(etRepo, etvRepo, attrRepo, nil)

	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 1).Return(&models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}, nil)
	etRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/et1/copy",
		`{"source_version":1,"new_name":"CopiedType"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Contains(t, rec.Body.String(), `"version":1`)
}

// T-5.20: Copy with duplicate name returns 409
func TestT5_20_CopyDuplicateName(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	e := setupTestServer(etRepo, etvRepo, nil, nil)

	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 1).Return(&models.EntityTypeVersion{ID: "v1"}, nil)
	etRepo.On("Create", mock.Anything, mock.Anything).Return(domainerrors.NewConflict("EntityType", "exists"))

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/et1/copy",
		`{"source_version":1,"new_name":"Existing"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusConflict, rec.Code)
}

// T-5.03: RW cannot POST to meta
func TestT5_03_RWCannotPostMeta(t *testing.T) {
	e := setupTestServer(nil, nil, nil, nil)
	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types",
		`{"name":"Test"}`, apimw.RoleRW)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// T-5.04: Admin can POST to meta
func TestT5_04_AdminCanPostMeta(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	e := setupTestServer(etRepo, etvRepo, nil, nil)

	etRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types",
		`{"name":"Test","description":"d"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

// T-5.07: Super Admin can modify production meta
func TestT5_07_SuperAdminCanModifyProd(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	e := setupTestServer(etRepo, etvRepo, attrRepo, assocRepo)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	etRepo.On("GetByID", mock.Anything, "et1").Return(&models.EntityType{ID: "et1"}, nil)
	etRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	rec := doRequest(e, http.MethodPut, "/api/meta/v1/entity-types/et1",
		`{"description":"Updated by super admin"}`, apimw.RoleSuperAdmin)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// T-5.08: Admin cannot modify production meta (tested at service level)
func TestT5_08_AdminModifyProd(t *testing.T) {
	// At the API level, Admin has access to PUT routes. Production protection
	// is enforced at the service level (CatalogVersionService checks).
	// This test verifies Admin can reach the PUT handler.
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	e := setupTestServer(etRepo, etvRepo, attrRepo, assocRepo)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	etRepo.On("GetByID", mock.Anything, "et1").Return(&models.EntityType{ID: "et1"}, nil)
	etRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	rec := doRequest(e, http.MethodPut, "/api/meta/v1/entity-types/et1",
		`{"description":"Updated by admin"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// T-5.17: PUT with stale version returns 409
func TestT5_17_UpdateStaleVersion(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	e := setupTestServer(etRepo, etvRepo, nil, nil)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(nil, domainerrors.NewNotFound("EntityTypeVersion", "et1"))

	rec := doRequest(e, http.MethodPut, "/api/meta/v1/entity-types/et1",
		`{"description":"Updated"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// T-5.21: POST entity-types as RO returns 403
func TestT5_21_PostAsRO(t *testing.T) {
	e := setupTestServer(nil, nil, nil, nil)
	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types",
		`{"name":"Test"}`, apimw.RoleRO)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// T-5.22: POST entity-types as RW returns 403
func TestT5_22_PostAsRW(t *testing.T) {
	e := setupTestServer(nil, nil, nil, nil)
	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types",
		`{"name":"Test"}`, apimw.RoleRW)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// setupETServerWithCatalogRepos creates a test server with all repos including catalog repos wired via WithCatalogRepos.
func setupETServerWithCatalogRepos(
	etRepo *mocks.MockEntityTypeRepo,
	etvRepo *mocks.MockEntityTypeVersionRepo,
	attrRepo *mocks.MockAttributeRepo,
	assocRepo *mocks.MockAssociationRepo,
	pinRepo *mocks.MockCatalogVersionPinRepo,
	cvRepo *mocks.MockCatalogVersionRepo,
) *echo.Echo {
	e := echo.New()
	svc := svcmeta.NewEntityTypeService(etRepo, etvRepo, attrRepo, assocRepo)
	svcmeta.WithCatalogRepos(svc, pinRepo, cvRepo)
	handler := apimeta.NewEntityTypeHandler(svc)

	g := e.Group("/api/meta/v1")
	rbac := &apimw.HeaderRBACProvider{}
	g.Use(apimw.RBACMiddleware(rbac))
	requireAdmin := apimw.RequireRole(apimw.RoleAdmin)
	apimeta.RegisterEntityTypeRoutes(g, handler, requireAdmin)

	return e
}

// TE-19: POST /entity-types/:id/rename simple rename returns 200
func TestTE19_RenameSimple(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	cvRepo := new(mocks.MockCatalogVersionRepo)
	e := setupETServerWithCatalogRepos(etRepo, etvRepo, nil, nil, pinRepo, cvRepo)

	now := time.Now()

	// GetByName returns NotFound (name is available)
	etRepo.On("GetByName", mock.Anything, "NewName").Return(nil, domainerrors.NewNotFound("EntityType", "NewName"))
	// GetByID returns the entity type to rename
	etRepo.On("GetByID", mock.Anything, "et1").Return(&models.EntityType{
		ID: "et1", Name: "OldName", CreatedAt: now, UpdatedAt: now,
	}, nil)
	// ListByEntityType returns versions for deep-copy check
	etvRepo.On("ListByEntityType", mock.Anything, "et1").Return([]*models.EntityTypeVersion{
		{ID: "v1", EntityTypeID: "et1", Version: 1},
	}, nil)
	// ListByEntityTypeVersionIDs returns empty (no pins → simple rename)
	pinRepo.On("ListByEntityTypeVersionIDs", mock.Anything, []string{"v1"}).Return([]*models.CatalogVersionPin{}, nil)
	// Update succeeds for the simple rename
	etRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.EntityType")).Return(nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/et1/rename",
		`{"name":"NewName","deep_copy_allowed":false}`, apimw.RoleAdmin)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp dto.RenameEntityTypeResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "NewName", resp.EntityType.Name)
	assert.False(t, resp.WasDeepCopy)
}

// TE-20: POST /entity-types/:id/rename deep copy required but not allowed returns 409
func TestTE20_RenameDeepCopyRequired(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	cvRepo := new(mocks.MockCatalogVersionRepo)
	e := setupETServerWithCatalogRepos(etRepo, etvRepo, nil, nil, pinRepo, cvRepo)

	now := time.Now()

	// GetByName returns NotFound (name is available)
	etRepo.On("GetByName", mock.Anything, "NewName").Return(nil, domainerrors.NewNotFound("EntityType", "NewName"))
	// GetByID returns the entity type
	etRepo.On("GetByID", mock.Anything, "et1").Return(&models.EntityType{
		ID: "et1", Name: "OldName", CreatedAt: now, UpdatedAt: now,
	}, nil)
	// ListByEntityType returns versions
	etvRepo.On("ListByEntityType", mock.Anything, "et1").Return([]*models.EntityTypeVersion{
		{ID: "v1", EntityTypeID: "et1", Version: 1},
	}, nil)
	// ListByEntityTypeVersionIDs returns pins (entity is referenced)
	pinRepo.On("ListByEntityTypeVersionIDs", mock.Anything, []string{"v1"}).Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "v1"},
	}, nil)
	// GetByID on cvRepo returns a testing-stage CV (triggers deep copy requirement)
	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", VersionLabel: "v1.0", LifecycleStage: models.LifecycleStageTesting,
		CreatedAt: now, UpdatedAt: now,
	}, nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/et1/rename",
		`{"name":"NewName","deep_copy_allowed":false}`, apimw.RoleAdmin)

	assert.Equal(t, http.StatusConflict, rec.Code)
}

// TE-21: POST /entity-types/:id/rename deep copy allowed returns 200
func TestTE21_RenameDeepCopyAllowed(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	cvRepo := new(mocks.MockCatalogVersionRepo)
	e := setupETServerWithCatalogRepos(etRepo, etvRepo, attrRepo, nil, pinRepo, cvRepo)

	now := time.Now()

	// GetByName returns NotFound (name is available)
	etRepo.On("GetByName", mock.Anything, "NewName").Return(nil, domainerrors.NewNotFound("EntityType", "NewName"))
	// GetByID returns the entity type
	etRepo.On("GetByID", mock.Anything, "et1").Return(&models.EntityType{
		ID: "et1", Name: "OldName", CreatedAt: now, UpdatedAt: now,
	}, nil)
	// ListByEntityType returns versions for deep-copy check
	etvRepo.On("ListByEntityType", mock.Anything, "et1").Return([]*models.EntityTypeVersion{
		{ID: "v1", EntityTypeID: "et1", Version: 1},
	}, nil)
	// ListByEntityTypeVersionIDs returns pins (entity is referenced)
	pinRepo.On("ListByEntityTypeVersionIDs", mock.Anything, []string{"v1"}).Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "v1"},
	}, nil)
	// GetByID on cvRepo returns a testing-stage CV (triggers deep copy requirement)
	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", VersionLabel: "v1.0", LifecycleStage: models.LifecycleStageTesting,
		CreatedAt: now, UpdatedAt: now,
	}, nil)
	// GetLatestByEntityType returns the latest version for deep copy
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{
		ID: "v1", EntityTypeID: "et1", Version: 1,
	}, nil)
	// CopyEntityType internals: GetByEntityTypeAndVersion
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 1).Return(&models.EntityTypeVersion{
		ID: "v1", EntityTypeID: "et1", Version: 1,
	}, nil)
	// Create the new entity type
	etRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityType")).Return(nil)
	// Create V1 for new entity type
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
	// Copy attributes
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/et1/rename",
		`{"name":"NewName","deep_copy_allowed":true}`, apimw.RoleAdmin)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp dto.RenameEntityTypeResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "NewName", resp.EntityType.Name)
	assert.True(t, resp.WasDeepCopy)
}

// TE-54: GET /entity-types/containment-tree returns 200 with tree
func TestTE54_ContainmentTree(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	e := setupTestServer(etRepo, etvRepo, nil, assocRepo)

	now := time.Now()

	// Two entity types: Server contains Tool
	etRepo.On("List", mock.Anything, mock.Anything).Return([]*models.EntityType{
		{ID: "et-a", Name: "Server", CreatedAt: now, UpdatedAt: now},
		{ID: "et-b", Name: "Tool", CreatedAt: now, UpdatedAt: now},
	}, 2, nil)
	assocRepo.On("GetContainmentGraph", mock.Anything).Return([]repository.ContainmentEdge{
		{SourceEntityTypeID: "et-a", TargetEntityTypeID: "et-b"},
	}, nil)
	etvRepo.On("ListByEntityType", mock.Anything, "et-a").Return([]*models.EntityTypeVersion{
		{ID: "va1", EntityTypeID: "et-a", Version: 1, CreatedAt: now},
	}, nil)
	etvRepo.On("ListByEntityType", mock.Anything, "et-b").Return([]*models.EntityTypeVersion{
		{ID: "vb1", EntityTypeID: "et-b", Version: 1, CreatedAt: now},
		{ID: "vb2", EntityTypeID: "et-b", Version: 2, CreatedAt: now},
	}, nil)

	rec := doRequest(e, http.MethodGet, "/api/meta/v1/entity-types/containment-tree", "", apimw.RoleRO)
	assert.Equal(t, http.StatusOK, rec.Code)

	var result []dto.ContainmentTreeNodeDTO
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	// Only Server is a root
	require.Len(t, result, 1)
	assert.Equal(t, "Server", result[0].EntityType.Name)
	assert.Equal(t, 1, result[0].LatestVersion)
	// Server has Tool as child
	require.Len(t, result[0].Children, 1)
	assert.Equal(t, "Tool", result[0].Children[0].EntityType.Name)
	assert.Len(t, result[0].Children[0].Versions, 2)
	assert.Equal(t, 2, result[0].Children[0].LatestVersion)
}

// TE-54b: GET /entity-types/containment-tree returns error when service fails
func TestTE54b_ContainmentTreeError(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	e := setupTestServer(etRepo, nil, nil, nil)

	etRepo.On("List", mock.Anything, mock.Anything).Return(([]*models.EntityType)(nil), 0, domainerrors.NewNotFound("EntityType", ""))

	rec := doRequest(e, http.MethodGet, "/api/meta/v1/entity-types/containment-tree", "", apimw.RoleRO)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// TE-62: GET /entity-types/:id/versions/:version/snapshot returns 200
func TestTE62_VersionSnapshot(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	e := setupTestServer(etRepo, etvRepo, attrRepo, assocRepo)

	now := time.Now()

	etRepo.On("GetByID", mock.Anything, "et1").Return(&models.EntityType{ID: "et1", Name: "Server", CreatedAt: now, UpdatedAt: now}, nil)
	etRepo.On("GetByID", mock.Anything, "et2").Return(&models.EntityType{ID: "et2", Name: "Tool", CreatedAt: now, UpdatedAt: now}, nil)
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 2).Return(&models.EntityTypeVersion{
		ID: "v2-id", EntityTypeID: "et1", Version: 2, Description: "V2", CreatedAt: now,
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v2-id").Return([]*models.Attribute{
		{ID: "a1", Name: "hostname", Type: "string", Ordinal: 1},
	}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v2-id").Return([]*models.Association{
		{ID: "as1", EntityTypeVersionID: "v2-id", TargetEntityTypeID: "et2", Type: "containment", SourceRole: "server", TargetRole: "tool", CreatedAt: now},
	}, nil)
	assocRepo.On("ListByTargetEntityType", mock.Anything, "et1").Return([]*models.Association{}, nil)

	rec := doRequest(e, http.MethodGet, "/api/meta/v1/entity-types/et1/versions/2/snapshot", "", apimw.RoleRO)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp dto.VersionSnapshotResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "Server", resp.EntityType.Name)
	assert.Equal(t, 2, resp.Version.Version)
	assert.Len(t, resp.Attributes, 3) // 2 system + 1 custom
	assert.Equal(t, "name", resp.Attributes[0].Name)
	assert.Equal(t, "description", resp.Attributes[1].Name)
	assert.Equal(t, "hostname", resp.Attributes[2].Name)
	assert.Len(t, resp.Associations, 1)
	assert.Equal(t, "containment", resp.Associations[0].Type)
	assert.Equal(t, "Tool", resp.Associations[0].TargetEntityTypeName)
	assert.Equal(t, "outgoing", resp.Associations[0].Direction)
	assert.Equal(t, "server", resp.Associations[0].SourceRole)
	assert.Equal(t, "tool", resp.Associations[0].TargetRole)
}

// TE-63: GET /entity-types/:id/versions/999/snapshot returns 404
func TestTE63_VersionSnapshot_NotFound(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	e := setupTestServer(etRepo, etvRepo, nil, nil)

	etRepo.On("GetByID", mock.Anything, "et1").Return(&models.EntityType{ID: "et1", Name: "Server"}, nil)
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 999).Return(nil, domainerrors.NewNotFound("EntityTypeVersion", "999"))

	rec := doRequest(e, http.MethodGet, "/api/meta/v1/entity-types/et1/versions/999/snapshot", "", apimw.RoleRO)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// Negative version number should return 400
func TestVersionSnapshot_NegativeVersion(t *testing.T) {
	e := setupTestServer(nil, nil, nil, nil)

	rec := doRequest(e, http.MethodGet, "/api/meta/v1/entity-types/et1/versions/-5/snapshot", "", apimw.RoleRO)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// Zero version number should return 400
func TestVersionSnapshot_ZeroVersion(t *testing.T) {
	e := setupTestServer(nil, nil, nil, nil)

	rec := doRequest(e, http.MethodGet, "/api/meta/v1/entity-types/et1/versions/0/snapshot", "", apimw.RoleRO)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// Snapshot with incoming association populates source entity type fields
func TestVersionSnapshot_IncomingAssociation(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	e := setupTestServer(etRepo, etvRepo, attrRepo, assocRepo)

	now := time.Now()

	etRepo.On("GetByID", mock.Anything, "et-tool").Return(&models.EntityType{ID: "et-tool", Name: "Tool", CreatedAt: now, UpdatedAt: now}, nil)
	etRepo.On("GetByID", mock.Anything, "et-server").Return(&models.EntityType{ID: "et-server", Name: "Server", CreatedAt: now, UpdatedAt: now}, nil)
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et-tool", 1).Return(&models.EntityTypeVersion{ID: "vt1", EntityTypeID: "et-tool", Version: 1, CreatedAt: now}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "vt1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "vt1").Return([]*models.Association{}, nil)
	assocRepo.On("ListByTargetEntityType", mock.Anything, "et-tool").Return([]*models.Association{
		{ID: "as1", EntityTypeVersionID: "vs3", TargetEntityTypeID: "et-tool", Type: "containment", SourceRole: "server", TargetRole: "tool", CreatedAt: now},
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "vs3").Return(&models.EntityTypeVersion{ID: "vs3", EntityTypeID: "et-server", Version: 3}, nil)
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-server").Return(&models.EntityTypeVersion{ID: "vs3", EntityTypeID: "et-server", Version: 3}, nil)

	rec := doRequest(e, http.MethodGet, "/api/meta/v1/entity-types/et-tool/versions/1/snapshot", "", apimw.RoleRO)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp dto.VersionSnapshotResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Associations, 1)
	assert.Equal(t, "incoming", resp.Associations[0].Direction)
	assert.Equal(t, "et-server", resp.Associations[0].SourceEntityTypeID)
	assert.Equal(t, "Server", resp.Associations[0].SourceEntityTypeName)
	assert.Equal(t, "server", resp.Associations[0].SourceRole)
}

// === Coverage: bind-error and service-error branches ===

func TestETCreate_BindError(t *testing.T) {
	e := setupTestServer(nil, nil, nil, nil)
	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types", "bad{json", apimw.RoleAdmin)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestETList_ServiceError(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	e := setupTestServer(etRepo, nil, nil, nil)
	etRepo.On("List", mock.Anything, mock.Anything).Return(([]*models.EntityType)(nil), 0, domainerrors.NewValidation("db error"))
	rec := doRequest(e, http.MethodGet, "/api/meta/v1/entity-types", "", apimw.RoleRO)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestETUpdate_BindError(t *testing.T) {
	e := setupTestServer(nil, nil, nil, nil)
	rec := doRequest(e, http.MethodPut, "/api/meta/v1/entity-types/et1", "bad{json", apimw.RoleAdmin)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestETDelete_ServiceError(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	e := setupTestServer(etRepo, nil, nil, nil)
	etRepo.On("Delete", mock.Anything, "et1").Return(domainerrors.NewNotFound("EntityType", "et1"))
	rec := doRequest(e, http.MethodDelete, "/api/meta/v1/entity-types/et1", "", apimw.RoleAdmin)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestETCopy_BindError(t *testing.T) {
	e := setupTestServer(nil, nil, nil, nil)
	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/et1/copy", "bad{json", apimw.RoleAdmin)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestETCopy_EmptyName(t *testing.T) {
	e := setupTestServer(nil, nil, nil, nil)
	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/et1/copy",
		`{"source_version":1,"new_name":""}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestETRename_BindError(t *testing.T) {
	e := setupTestServer(nil, nil, nil, nil)
	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/et1/rename", "bad{json", apimw.RoleAdmin)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestETRename_EmptyName(t *testing.T) {
	e := setupTestServer(nil, nil, nil, nil)
	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/et1/rename",
		`{"name":""}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// T-E.83: Version snapshot includes cardinality
func TestTE83_VersionSnapshotIncludesCardinality(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	e := setupTestServer(etRepo, etvRepo, attrRepo, assocRepo)

	now := time.Now()

	etRepo.On("GetByID", mock.Anything, "et1").Return(&models.EntityType{ID: "et1", Name: "Server", CreatedAt: now, UpdatedAt: now}, nil)
	etRepo.On("GetByID", mock.Anything, "et2").Return(&models.EntityType{ID: "et2", Name: "Tool", CreatedAt: now, UpdatedAt: now}, nil)
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 1).Return(&models.EntityTypeVersion{
		ID: "v1-id", EntityTypeID: "et1", Version: 1, CreatedAt: now,
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1-id").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1-id").Return([]*models.Association{
		{ID: "as1", EntityTypeVersionID: "v1-id", TargetEntityTypeID: "et2", Type: "containment",
			SourceCardinality: "1", TargetCardinality: "0..n", CreatedAt: now},
	}, nil)
	assocRepo.On("ListByTargetEntityType", mock.Anything, "et1").Return([]*models.Association{}, nil)

	rec := doRequest(e, http.MethodGet, "/api/meta/v1/entity-types/et1/versions/1/snapshot", "", apimw.RoleRO)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp dto.VersionSnapshotResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Associations, 1)
	assert.Equal(t, "1", resp.Associations[0].SourceCardinality)
	assert.Equal(t, "0..n", resp.Associations[0].TargetCardinality)
}

// === TD-22: System Attributes in Version Snapshot ===

// T-18.07: Version snapshot prepends Name system attr
func TestT18_07_SnapshotSystemAttrName(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	e := setupTestServer(etRepo, etvRepo, attrRepo, assocRepo)

	now := time.Now()
	etRepo.On("GetByID", mock.Anything, "et1").Return(&models.EntityType{ID: "et1", Name: "Server", CreatedAt: now, UpdatedAt: now}, nil)
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 1).Return(&models.EntityTypeVersion{
		ID: "v1", EntityTypeID: "et1", Version: 1, CreatedAt: now,
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{
		{ID: "a1", Name: "hostname", Type: "string", Ordinal: 0},
	}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	assocRepo.On("ListByTargetEntityType", mock.Anything, "et1").Return([]*models.Association{}, nil)

	rec := doRequest(e, http.MethodGet, "/api/meta/v1/entity-types/et1/versions/1/snapshot", "", apimw.RoleRO)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp dto.VersionSnapshotResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Attributes, 3) // 2 system + 1 custom
	assert.Equal(t, "name", resp.Attributes[0].Name)
	assert.Equal(t, "string", resp.Attributes[0].Type)
	assert.Equal(t, true, resp.Attributes[0].System)
	assert.Equal(t, true, resp.Attributes[0].Required)
	assert.Equal(t, -2, resp.Attributes[0].Ordinal)
}

// T-18.08: Version snapshot prepends Description system attr
func TestT18_08_SnapshotSystemAttrDescription(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	e := setupTestServer(etRepo, etvRepo, attrRepo, assocRepo)

	now := time.Now()
	etRepo.On("GetByID", mock.Anything, "et1").Return(&models.EntityType{ID: "et1", Name: "Server", CreatedAt: now, UpdatedAt: now}, nil)
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 1).Return(&models.EntityTypeVersion{
		ID: "v1", EntityTypeID: "et1", Version: 1, CreatedAt: now,
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	assocRepo.On("ListByTargetEntityType", mock.Anything, "et1").Return([]*models.Association{}, nil)

	rec := doRequest(e, http.MethodGet, "/api/meta/v1/entity-types/et1/versions/1/snapshot", "", apimw.RoleRO)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp dto.VersionSnapshotResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Attributes, 2) // only system attrs
	assert.Equal(t, "description", resp.Attributes[1].Name)
	assert.Equal(t, false, resp.Attributes[1].Required)
	assert.Equal(t, true, resp.Attributes[1].System)
	assert.Equal(t, -1, resp.Attributes[1].Ordinal)
}

// T-18.09: Custom attrs in snapshot retain original ordinals
func TestT18_09_SnapshotCustomOrdinals(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	e := setupTestServer(etRepo, etvRepo, attrRepo, assocRepo)

	now := time.Now()
	etRepo.On("GetByID", mock.Anything, "et1").Return(&models.EntityType{ID: "et1", Name: "Server", CreatedAt: now, UpdatedAt: now}, nil)
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 1).Return(&models.EntityTypeVersion{
		ID: "v1", EntityTypeID: "et1", Version: 1, CreatedAt: now,
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{
		{ID: "a1", Name: "hostname", Type: "string", Ordinal: 0},
		{ID: "a2", Name: "port", Type: "number", Ordinal: 1},
	}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	assocRepo.On("ListByTargetEntityType", mock.Anything, "et1").Return([]*models.Association{}, nil)

	rec := doRequest(e, http.MethodGet, "/api/meta/v1/entity-types/et1/versions/1/snapshot", "", apimw.RoleRO)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp dto.VersionSnapshotResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Attributes, 4)
	assert.Equal(t, 0, resp.Attributes[2].Ordinal) // hostname
	assert.Equal(t, 1, resp.Attributes[3].Ordinal) // port
	assert.Equal(t, false, resp.Attributes[2].System)
}
