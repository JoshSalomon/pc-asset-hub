package meta_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/project-catalyst/pc-asset-hub/internal/api/dto"
	apimeta "github.com/project-catalyst/pc-asset-hub/internal/api/meta"
	apimw "github.com/project-catalyst/pc-asset-hub/internal/api/middleware"
	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository/mocks"
	svcmeta "github.com/project-catalyst/pc-asset-hub/internal/service/meta"
)

type typeDefTestServer struct {
	echo     *echo.Echo
	tdRepo   *mocks.MockTypeDefinitionRepo
	tdvRepo  *mocks.MockTypeDefinitionVersionRepo
	attrRepo *mocks.MockAttributeRepo
	pinRepo  *mocks.MockCatalogVersionPinRepo
	etvRepo  *mocks.MockEntityTypeVersionRepo
}

func setupTypeDefServer(tdRepo *mocks.MockTypeDefinitionRepo, tdvRepo *mocks.MockTypeDefinitionVersionRepo) *typeDefTestServer {
	attrRepo := new(mocks.MockAttributeRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := svcmeta.NewTypeDefinitionService(tdRepo, tdvRepo, attrRepo, svcmeta.WithPinRepo(pinRepo), svcmeta.WithETVRepo(etvRepo))
	handler := apimeta.NewTypeDefinitionHandler(svc)

	e := echo.New()
	g := e.Group("/api/meta/v1")
	rbac := &apimw.HeaderRBACProvider{}
	g.Use(apimw.RBACMiddleware(rbac))
	requireAdmin := apimw.RequireRole(apimw.RoleAdmin)
	apimeta.RegisterTypeDefinitionRoutes(g, handler, requireAdmin)

	return &typeDefTestServer{echo: e, tdRepo: tdRepo, tdvRepo: tdvRepo, attrRepo: attrRepo, pinRepo: pinRepo, etvRepo: etvRepo}
}

func TestTypeDefHandler_Create(t *testing.T) {
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	ts := setupTypeDefServer(tdRepo, tdvRepo)

	tdRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	tdvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	body := `{"name":"guardrailID","description":"Hex ID","base_type":"string","constraints":{"max_length":12}}`
	rec := doRequest(ts.echo, http.MethodPost, "/api/meta/v1/type-definitions", body, apimw.RoleAdmin)

	assert.Equal(t, http.StatusCreated, rec.Code)
	var resp dto.TypeDefinitionResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "guardrailID", resp.Name)
	assert.Equal(t, "string", resp.BaseType)
	assert.Equal(t, 1, resp.LatestVersion)
}

func TestTypeDefHandler_Create_RequiresAdmin(t *testing.T) {
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	ts := setupTypeDefServer(tdRepo, tdvRepo)

	body := `{"name":"test","base_type":"string"}`
	rec := doRequest(ts.echo, http.MethodPost, "/api/meta/v1/type-definitions", body, apimw.RoleRO)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestTypeDefHandler_List(t *testing.T) {
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	ts := setupTypeDefServer(tdRepo, tdvRepo)

	tdRepo.On("List", mock.Anything, mock.Anything).Return([]*models.TypeDefinition{
		{ID: "td-1", Name: "string", BaseType: models.BaseTypeString, System: true},
		{ID: "td-2", Name: "status", BaseType: models.BaseTypeEnum},
	}, 2, nil)
	// Batch version lookup
	tdvRepo.On("GetLatestByTypeDefinitions", mock.Anything, []string{"td-1", "td-2"}).Return(map[string]*models.TypeDefinitionVersion{
		"td-1": {ID: "tdv-1", VersionNumber: 1},
		"td-2": {ID: "tdv-2", VersionNumber: 3},
	}, nil)

	rec := doRequest(ts.echo, http.MethodGet, "/api/meta/v1/type-definitions", "", apimw.RoleRO)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp dto.ListResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 2, resp.Total)

	// Verify latest_version_id is present in response
	body := rec.Body.String()
	assert.Contains(t, body, `"latest_version_id":"tdv-1"`)
	assert.Contains(t, body, `"latest_version_id":"tdv-2"`)
}

func TestTypeDefHandler_GetByID(t *testing.T) {
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	ts := setupTypeDefServer(tdRepo, tdvRepo)

	tdRepo.On("GetByID", mock.Anything, "td-1").Return(&models.TypeDefinition{ID: "td-1", Name: "status", BaseType: models.BaseTypeEnum}, nil)
	tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-1").Return(&models.TypeDefinitionVersion{
		ID: "tdv-1", VersionNumber: 2, Constraints: map[string]any{"values": []any{"a", "b"}},
	}, nil)

	rec := doRequest(ts.echo, http.MethodGet, "/api/meta/v1/type-definitions/td-1", "", apimw.RoleRO)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "status")
}

func TestTypeDefHandler_Update(t *testing.T) {
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	ts := setupTypeDefServer(tdRepo, tdvRepo)

	tdRepo.On("GetByID", mock.Anything, "td-1").Return(&models.TypeDefinition{ID: "td-1", Name: "guardrailID", BaseType: models.BaseTypeString}, nil)
	tdRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
	tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-1").Return(&models.TypeDefinitionVersion{
		ID: "tdv-1", VersionNumber: 1, Constraints: map[string]any{"max_length": float64(12)},
	}, nil)
	tdvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	body := `{"description":"Updated","constraints":{"max_length":16}}`
	rec := doRequest(ts.echo, http.MethodPut, "/api/meta/v1/type-definitions/td-1", body, apimw.RoleAdmin)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestTypeDefHandler_Delete(t *testing.T) {
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	ts := setupTypeDefServer(tdRepo, tdvRepo)

	tdRepo.On("GetByID", mock.Anything, "td-1").Return(&models.TypeDefinition{ID: "td-1", Name: "guardrailID", BaseType: models.BaseTypeString}, nil)
	// Reference check: no versions → no attributes → safe to delete
	tdvRepo.On("ListByTypeDefinition", mock.Anything, "td-1").Return([]*models.TypeDefinitionVersion{
		{ID: "tdv-1", TypeDefinitionID: "td-1", VersionNumber: 1},
	}, nil)
	ts.attrRepo.On("ListByTypeDefinitionVersionIDs", mock.Anything, []string{"tdv-1"}).Return([]*models.Attribute{}, nil)
	tdRepo.On("Delete", mock.Anything, "td-1").Return(nil)

	rec := doRequest(ts.echo, http.MethodDelete, "/api/meta/v1/type-definitions/td-1", "", apimw.RoleAdmin)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

// T-29.23: DELETE /type-definitions/{id} with type in use by pinned version → 409
func TestTypeDefHandler_Delete_InUse(t *testing.T) {
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	ts := setupTypeDefServer(tdRepo, tdvRepo)

	tdRepo.On("GetByID", mock.Anything, "td-1").Return(&models.TypeDefinition{ID: "td-1", Name: "guardrailID", BaseType: models.BaseTypeString}, nil)
	tdvRepo.On("ListByTypeDefinition", mock.Anything, "td-1").Return([]*models.TypeDefinitionVersion{
		{ID: "tdv-1", TypeDefinitionID: "td-1", VersionNumber: 1},
	}, nil)
	ts.attrRepo.On("ListByTypeDefinitionVersionIDs", mock.Anything, []string{"tdv-1"}).Return([]*models.Attribute{
		{ID: "attr-1", Name: "guard_id", EntityTypeVersionID: "etv-1", TypeDefinitionVersionID: "tdv-1"},
	}, nil)
	ts.pinRepo.On("ListByEntityTypeVersionIDs", mock.Anything, []string{"etv-1"}).Return([]*models.CatalogVersionPin{
		{ID: "pin-1", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-1"},
	}, nil)

	rec := doRequest(ts.echo, http.MethodDelete, "/api/meta/v1/type-definitions/td-1", "", apimw.RoleAdmin)
	assert.Equal(t, http.StatusConflict, rec.Code)
	assert.Contains(t, rec.Body.String(), "in use")
}

func TestTypeDefHandler_ListVersions(t *testing.T) {
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	ts := setupTypeDefServer(tdRepo, tdvRepo)

	tdvRepo.On("ListByTypeDefinition", mock.Anything, "td-1").Return([]*models.TypeDefinitionVersion{
		{ID: "tdv-1", VersionNumber: 1, Constraints: map[string]any{}},
		{ID: "tdv-2", VersionNumber: 2, Constraints: map[string]any{"max_length": float64(16)}},
	}, nil)

	rec := doRequest(ts.echo, http.MethodGet, "/api/meta/v1/type-definitions/td-1/versions", "", apimw.RoleRO)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// === Create error paths ===

func TestTypeDefHandler_Create_BindError(t *testing.T) {
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	ts := setupTypeDefServer(tdRepo, tdvRepo)

	// Send invalid JSON
	rec := doRequest(ts.echo, http.MethodPost, "/api/meta/v1/type-definitions", "{invalid json", apimw.RoleAdmin)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTypeDefHandler_Create_EmptyName(t *testing.T) {
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	ts := setupTypeDefServer(tdRepo, tdvRepo)

	body := `{"name":"","base_type":"string"}`
	rec := doRequest(ts.echo, http.MethodPost, "/api/meta/v1/type-definitions", body, apimw.RoleAdmin)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "name is required")
}

func TestTypeDefHandler_Create_EmptyBaseType(t *testing.T) {
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	ts := setupTypeDefServer(tdRepo, tdvRepo)

	body := `{"name":"test","base_type":""}`
	rec := doRequest(ts.echo, http.MethodPost, "/api/meta/v1/type-definitions", body, apimw.RoleAdmin)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "base_type is required")
}

func TestTypeDefHandler_Create_ServiceError(t *testing.T) {
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	ts := setupTypeDefServer(tdRepo, tdvRepo)

	tdRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("internal error"))

	body := `{"name":"test","base_type":"string"}`
	rec := doRequest(ts.echo, http.MethodPost, "/api/meta/v1/type-definitions", body, apimw.RoleAdmin)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// === List error paths ===

func TestTypeDefHandler_List_ServiceError(t *testing.T) {
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	ts := setupTypeDefServer(tdRepo, tdvRepo)

	tdRepo.On("List", mock.Anything, mock.Anything).Return([]*models.TypeDefinition(nil), 0, errors.New("list error"))

	rec := doRequest(ts.echo, http.MethodGet, "/api/meta/v1/type-definitions", "", apimw.RoleRO)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestTypeDefHandler_List_VersionInfoError(t *testing.T) {
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	ts := setupTypeDefServer(tdRepo, tdvRepo)

	tdRepo.On("List", mock.Anything, mock.Anything).Return([]*models.TypeDefinition{
		{ID: "td-1", Name: "string", BaseType: models.BaseTypeString, System: true},
	}, 1, nil)
	tdvRepo.On("GetLatestByTypeDefinitions", mock.Anything, []string{"td-1"}).Return(nil, errors.New("version lookup failed"))

	rec := doRequest(ts.echo, http.MethodGet, "/api/meta/v1/type-definitions", "", apimw.RoleRO)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestTypeDefHandler_List_WithFilters(t *testing.T) {
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	ts := setupTypeDefServer(tdRepo, tdvRepo)

	tdRepo.On("List", mock.Anything, mock.MatchedBy(func(p models.ListParams) bool {
		return p.Filters["base_type"] == "string" && p.Filters["name"] == "test"
	})).Return([]*models.TypeDefinition{}, 0, nil)
	tdvRepo.On("GetLatestByTypeDefinitions", mock.Anything, []string{}).Return(map[string]*models.TypeDefinitionVersion{}, nil)

	rec := doRequest(ts.echo, http.MethodGet, "/api/meta/v1/type-definitions?base_type=string&name=test", "", apimw.RoleRO)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// === GetByID error paths ===

func TestTypeDefHandler_GetByID_NotFound(t *testing.T) {
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	ts := setupTypeDefServer(tdRepo, tdvRepo)

	tdRepo.On("GetByID", mock.Anything, "td-missing").Return(nil, domainerrors.NewNotFound("TypeDefinition", "td-missing"))

	rec := doRequest(ts.echo, http.MethodGet, "/api/meta/v1/type-definitions/td-missing", "", apimw.RoleRO)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// === Update error paths ===

func TestTypeDefHandler_Update_BindError(t *testing.T) {
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	ts := setupTypeDefServer(tdRepo, tdvRepo)

	rec := doRequest(ts.echo, http.MethodPut, "/api/meta/v1/type-definitions/td-1", "{invalid", apimw.RoleAdmin)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTypeDefHandler_Update_ServiceError(t *testing.T) {
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	ts := setupTypeDefServer(tdRepo, tdvRepo)

	tdRepo.On("GetByID", mock.Anything, "td-1").Return(nil, domainerrors.NewNotFound("TypeDefinition", "td-1"))

	body := `{"constraints":{"max_length":16}}`
	rec := doRequest(ts.echo, http.MethodPut, "/api/meta/v1/type-definitions/td-1", body, apimw.RoleAdmin)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// === Delete error paths ===

func TestTypeDefHandler_Delete_ServiceError(t *testing.T) {
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	ts := setupTypeDefServer(tdRepo, tdvRepo)

	tdRepo.On("GetByID", mock.Anything, "td-1").Return(nil, domainerrors.NewNotFound("TypeDefinition", "td-1"))

	rec := doRequest(ts.echo, http.MethodDelete, "/api/meta/v1/type-definitions/td-1", "", apimw.RoleAdmin)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// === ListVersions error paths ===

func TestTypeDefHandler_ListVersions_ServiceError(t *testing.T) {
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	ts := setupTypeDefServer(tdRepo, tdvRepo)

	tdvRepo.On("ListByTypeDefinition", mock.Anything, "td-1").Return(nil, errors.New("version list error"))

	rec := doRequest(ts.echo, http.MethodGet, "/api/meta/v1/type-definitions/td-1/versions", "", apimw.RoleRO)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// === GetVersion ===

func TestTypeDefHandler_GetVersion_Success(t *testing.T) {
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	ts := setupTypeDefServer(tdRepo, tdvRepo)

	tdvRepo.On("GetByVersion", mock.Anything, "td-1", 2).Return(&models.TypeDefinitionVersion{
		ID: "tdv-2", TypeDefinitionID: "td-1", VersionNumber: 2, Constraints: map[string]any{"max_length": float64(16)},
	}, nil)

	rec := doRequest(ts.echo, http.MethodGet, "/api/meta/v1/type-definitions/td-1/versions/2", "", apimw.RoleRO)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp dto.TypeDefinitionVersionResponse
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, 2, resp.VersionNumber)
	assert.Equal(t, "tdv-2", resp.ID)
}

func TestTypeDefHandler_GetVersion_InvalidVersionNumber(t *testing.T) {
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	ts := setupTypeDefServer(tdRepo, tdvRepo)

	rec := doRequest(ts.echo, http.MethodGet, "/api/meta/v1/type-definitions/td-1/versions/notanumber", "", apimw.RoleRO)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestTypeDefHandler_GetVersion_NotFound(t *testing.T) {
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	ts := setupTypeDefServer(tdRepo, tdvRepo)

	tdvRepo.On("GetByVersion", mock.Anything, "td-1", 99).Return(nil, domainerrors.NewNotFound("TypeDefinitionVersion", "v99"))

	rec := doRequest(ts.echo, http.MethodGet, "/api/meta/v1/type-definitions/td-1/versions/99", "", apimw.RoleRO)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestTypeDefHandler_GetVersion_ServiceError(t *testing.T) {
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	ts := setupTypeDefServer(tdRepo, tdvRepo)

	tdvRepo.On("GetByVersion", mock.Anything, "td-1", 1).Return(nil, errors.New("version error"))

	rec := doRequest(ts.echo, http.MethodGet, "/api/meta/v1/type-definitions/td-1/versions/1", "", apimw.RoleRO)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}
