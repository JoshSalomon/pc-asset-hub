package operational_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	apimw "github.com/project-catalyst/pc-asset-hub/internal/api/middleware"
	apiop "github.com/project-catalyst/pc-asset-hub/internal/api/operational"
	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository/mocks"
	svcop "github.com/project-catalyst/pc-asset-hub/internal/service/operational"
)

func setupExportServer() (*echo.Echo, *mocks.MockCatalogRepo, *mocks.MockCatalogVersionRepo, *mocks.MockCatalogVersionPinRepo, *mocks.MockEntityTypeRepo, *mocks.MockEntityTypeVersionRepo, *mocks.MockAttributeRepo, *mocks.MockAssociationRepo, *mocks.MockTypeDefinitionRepo, *mocks.MockTypeDefinitionVersionRepo, *mocks.MockEntityInstanceRepo, *mocks.MockInstanceAttributeValueRepo, *mocks.MockAssociationLinkRepo) {
	catalogRepo := new(mocks.MockCatalogRepo)
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	instRepo := new(mocks.MockEntityInstanceRepo)
	iavRepo := new(mocks.MockInstanceAttributeValueRepo)
	linkRepo := new(mocks.MockAssociationLinkRepo)

	exportSvc := svcop.NewExportService(catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, instRepo, iavRepo, linkRepo)
	accessChecker := &apimw.HeaderCatalogAccessChecker{}
	exportHandler := apiop.NewExportHandler(exportSvc, accessChecker)

	e := echo.New()
	g := e.Group("/api/data/v1/catalogs")
	rbac := &apimw.HeaderRBACProvider{}
	g.Use(apimw.RBACMiddleware(rbac))
	requireAdmin := apimw.RequireRole(apimw.RoleAdmin)
	apiop.RegisterExportRoutes(g, exportHandler, requireAdmin)

	return e, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, instRepo, iavRepo, linkRepo
}

func doExportRequest(e *echo.Echo, path string, role apimw.Role) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("X-User-Role", string(role))
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// T-30.14: GET /catalogs/{name}/export — success
func TestT30_14_ExportEndpointSuccess(t *testing.T) {
	e, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, _, _, instRepo, _, _ := setupExportServer()

	now := time.Now()
	catalog := &models.Catalog{ID: "cat-1", Name: "prod-agents", CatalogVersionID: "cv-1", ValidationStatus: models.ValidationStatusValid, CreatedAt: now, UpdatedAt: now}
	cv := &models.CatalogVersion{ID: "cv-1", VersionLabel: "v2.0", Description: "Spring release"}
	et := &models.EntityType{ID: "et-1", Name: "agent"}
	etv := &models.EntityTypeVersion{ID: "etv-1", EntityTypeID: "et-1", Version: 1, Description: "Agent entity"}
	pin := &models.CatalogVersionPin{ID: "pin-1", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-1"}

	catalogRepo.On("GetByName", mock.Anything, "prod-agents").Return(catalog, nil)
	cvRepo.On("GetByID", mock.Anything, "cv-1").Return(cv, nil)
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv-1").Return([]*models.CatalogVersionPin{pin}, nil)
	etvRepo.On("GetByID", mock.Anything, "etv-1").Return(etv, nil)
	etRepo.On("GetByID", mock.Anything, "et-1").Return(et, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv-1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "etv-1").Return([]*models.Association{}, nil)
	instRepo.On("ListByCatalog", mock.Anything, "cat-1").Return([]*models.EntityInstance{}, nil)

	rec := doExportRequest(e, "/api/data/v1/catalogs/prod-agents/export", apimw.RoleAdmin)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Disposition"), `attachment; filename="prod-agents-export.json"`)

	var result svcop.ExportData
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	assert.Equal(t, "1.0", result.FormatVersion)
	assert.Equal(t, "prod-agents", result.Catalog.Name)
	assert.Equal(t, "v2.0", result.CatalogVersion.Label)
}

// T-30.15: GET /catalogs/{name}/export — catalog not found → 404
func TestT30_15_ExportEndpointNotFound(t *testing.T) {
	e, catalogRepo, _, _, _, _, _, _, _, _, _, _, _ := setupExportServer()

	catalogRepo.On("GetByName", mock.Anything, "nonexistent").Return(nil, domainerrors.NewNotFound("Catalog", "nonexistent"))

	rec := doExportRequest(e, "/api/data/v1/catalogs/nonexistent/export", apimw.RoleAdmin)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// T-30.16: GET /catalogs/{name}/export — non-admin gets 403
func TestT30_16_ExportEndpointForbidden(t *testing.T) {
	e, _, _, _, _, _, _, _, _, _, _, _, _ := setupExportServer()

	rec := doExportRequest(e, "/api/data/v1/catalogs/prod-agents/export", apimw.RoleRW)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// T-30.17: GET /catalogs/{name}/export?entities=server — entity filter passed to service
func TestT30_17_ExportEndpointWithEntityFilter(t *testing.T) {
	e, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, _, _, instRepo, _, _ := setupExportServer()

	now := time.Now()
	catalog := &models.Catalog{ID: "cat-1", Name: "my-catalog", CatalogVersionID: "cv-1", ValidationStatus: models.ValidationStatusValid, CreatedAt: now, UpdatedAt: now}
	cv := &models.CatalogVersion{ID: "cv-1", VersionLabel: "v1.0"}

	et1 := &models.EntityType{ID: "et-1", Name: "server"}
	etv1 := &models.EntityTypeVersion{ID: "etv-1", EntityTypeID: "et-1", Version: 1, Description: "server"}
	et2 := &models.EntityType{ID: "et-2", Name: "guardrail"}
	etv2 := &models.EntityTypeVersion{ID: "etv-2", EntityTypeID: "et-2", Version: 1, Description: "guard"}

	pin1 := &models.CatalogVersionPin{ID: "pin-1", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-1"}
	pin2 := &models.CatalogVersionPin{ID: "pin-2", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-2"}

	catalogRepo.On("GetByName", mock.Anything, "my-catalog").Return(catalog, nil)
	cvRepo.On("GetByID", mock.Anything, "cv-1").Return(cv, nil)
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv-1").Return([]*models.CatalogVersionPin{pin1, pin2}, nil)
	etvRepo.On("GetByID", mock.Anything, "etv-1").Return(etv1, nil)
	etvRepo.On("GetByID", mock.Anything, "etv-2").Return(etv2, nil)
	etRepo.On("GetByID", mock.Anything, "et-1").Return(et1, nil)
	etRepo.On("GetByID", mock.Anything, "et-2").Return(et2, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv-1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "etv-1").Return([]*models.Association{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "etv-2").Return([]*models.Association{}, nil)
	instRepo.On("ListByCatalog", mock.Anything, "cat-1").Return([]*models.EntityInstance{}, nil)

	rec := doExportRequest(e, "/api/data/v1/catalogs/my-catalog/export?entities=server", apimw.RoleAdmin)

	assert.Equal(t, http.StatusOK, rec.Code)
	var result svcop.ExportData
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	require.Len(t, result.EntityTypes, 1)
	assert.Equal(t, "server", result.EntityTypes[0].Name)
}

// T-30.18: GET /catalogs/{name}/export?source_system=custom — source system override
func TestT30_18_ExportEndpointSourceSystem(t *testing.T) {
	e, catalogRepo, cvRepo, pinRepo, _, _, _, _, _, _, instRepo, _, _ := setupExportServer()

	now := time.Now()
	catalog := &models.Catalog{ID: "cat-1", Name: "my-catalog", CatalogVersionID: "cv-1", ValidationStatus: models.ValidationStatusValid, CreatedAt: now, UpdatedAt: now}
	cv := &models.CatalogVersion{ID: "cv-1", VersionLabel: "v1.0"}

	catalogRepo.On("GetByName", mock.Anything, "my-catalog").Return(catalog, nil)
	cvRepo.On("GetByID", mock.Anything, "cv-1").Return(cv, nil)
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv-1").Return([]*models.CatalogVersionPin{}, nil)
	instRepo.On("ListByCatalog", mock.Anything, "cat-1").Return([]*models.EntityInstance{}, nil)

	rec := doExportRequest(e, "/api/data/v1/catalogs/my-catalog/export?source_system=prod-cluster", apimw.RoleAdmin)

	assert.Equal(t, http.StatusOK, rec.Code)
	var result svcop.ExportData
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	assert.Equal(t, "prod-cluster", result.SourceSystem)
}
