package operational_test

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

	apimw "github.com/project-catalyst/pc-asset-hub/internal/api/middleware"
	apiop "github.com/project-catalyst/pc-asset-hub/internal/api/operational"
	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository/mocks"
	svcop "github.com/project-catalyst/pc-asset-hub/internal/service/operational"
)

func setupImportServer() (*echo.Echo, *mocks.MockCatalogRepo, *mocks.MockCatalogVersionRepo, *mocks.MockCatalogVersionPinRepo, *mocks.MockEntityTypeRepo, *mocks.MockEntityTypeVersionRepo, *mocks.MockAttributeRepo, *mocks.MockAssociationRepo, *mocks.MockTypeDefinitionRepo, *mocks.MockTypeDefinitionVersionRepo, *mocks.MockEntityInstanceRepo, *mocks.MockInstanceAttributeValueRepo, *mocks.MockAssociationLinkRepo, *mocks.MockCatalogVersionTypePinRepo) {
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
	typePinRepo := new(mocks.MockCatalogVersionTypePinRepo)
	txManager := &mocks.MockTransactionManager{}

	importSvc := svcop.NewImportService(catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, instRepo, iavRepo, linkRepo, typePinRepo, svcop.WithImportTransactionManager(txManager))
	accessChecker := &apimw.HeaderCatalogAccessChecker{}
	importHandler := apiop.NewImportHandler(importSvc, accessChecker)

	e := echo.New()
	g := e.Group("/api/data/v1/catalogs")
	rbac := &apimw.HeaderRBACProvider{}
	g.Use(apimw.RBACMiddleware(rbac))
	requireAdmin := apimw.RequireRole(apimw.RoleAdmin)
	apiop.RegisterImportRoutes(g, importHandler, requireAdmin)

	return e, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, instRepo, iavRepo, linkRepo, typePinRepo
}

func doImportRequest(e *echo.Echo, path, body string, role apimw.Role) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("X-User-Role", string(role))
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// T-30.50: POST /catalogs/import?dry_run=true — dry run success
func TestT30_50_ImportDryRunEndpoint(t *testing.T) {
	e, catalogRepo, cvRepo, _, etRepo, _, _, _, _, _, _, _, _, _ := setupImportServer()

	catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
	etRepo.On("GetByName", mock.Anything, "server").Return(nil, domainerrors.NewNotFound("EntityType", "server"))

	body := `{
		"data": {
			"format_version": "1.0",
			"exported_at": "2026-04-23T14:30:00Z",
			"source_system": "test",
			"catalog": {"name": "test-catalog", "description": "test", "validation_status": "valid"},
			"catalog_version": {"label": "v1.0", "description": "first"},
			"type_definitions": [],
			"entity_types": [{"name": "server", "description": "A server", "attributes": [], "associations": []}],
			"instances": []
		}
	}`

	rec := doImportRequest(e, "/api/data/v1/catalogs/import?dry_run=true", body, apimw.RoleAdmin)

	assert.Equal(t, http.StatusOK, rec.Code)
	var result svcop.DryRunResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	assert.Equal(t, "ready", result.Status)
}

// T-30.51: POST /catalogs/import — import success
func TestT30_51_ImportEndpointSuccess(t *testing.T) {
	e, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, _, assocRepo, tdRepo, tdvRepo, _, _, _, typePinRepo := setupImportServer()

	catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))

	systemTDs := []*models.TypeDefinition{{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}}
	systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}
	tdRepo.On("List", mock.Anything, mock.Anything).Return(systemTDs, 1, nil)
	tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-str").Return(systemTDV, nil)

	etRepo.On("GetByName", mock.Anything, "server").Return(nil, domainerrors.NewNotFound("EntityType", "server"))
	etRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityType")).Return(nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
	cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
	pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)
	typePinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionTypePin")).Return(nil)
	assocRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Association{}, nil)
	catalogRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Catalog")).Return(nil)

	body := `{
		"data": {
			"format_version": "1.0",
			"exported_at": "2026-04-23T14:30:00Z",
			"source_system": "test",
			"catalog": {"name": "test-catalog", "description": "test", "validation_status": "valid"},
			"catalog_version": {"label": "v1.0", "description": "first"},
			"type_definitions": [],
			"entity_types": [{"name": "server", "description": "A server", "attributes": [], "associations": []}],
			"instances": []
		}
	}`

	rec := doImportRequest(e, "/api/data/v1/catalogs/import", body, apimw.RoleAdmin)

	assert.Equal(t, http.StatusCreated, rec.Code)
	var result svcop.ImportResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	assert.Equal(t, "success", result.Status)
	assert.Equal(t, "test-catalog", result.CatalogName)
}

// T-30.52: POST /catalogs/import — non-admin gets 403
func TestT30_52_ImportEndpointForbidden(t *testing.T) {
	e, _, _, _, _, _, _, _, _, _, _, _, _, _ := setupImportServer()

	rec := doImportRequest(e, "/api/data/v1/catalogs/import", `{}`, apimw.RoleRW)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// T-30.53: POST /catalogs/import?dry_run=invalid — bad dry_run param
func TestT30_53_ImportInvalidDryRunParam(t *testing.T) {
	e, _, _, _, _, _, _, _, _, _, _, _, _, _ := setupImportServer()

	rec := doImportRequest(e, "/api/data/v1/catalogs/import?dry_run=invalid", `{}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// T-30.54b: POST /catalogs/import?dry_run=true — service error → mapped error
func TestT30_54b_ImportDryRunServiceError(t *testing.T) {
	e, catalogRepo, _, _, _, _, _, _, _, _, _, _, _, _ := setupImportServer()

	catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, assert.AnError) // non-notfound → will propagate

	body := `{
		"data": {
			"format_version": "1.0",
			"exported_at": "2026-04-23T14:30:00Z",
			"source_system": "test",
			"catalog": {"name": "test-catalog", "description": "test", "validation_status": "valid"},
			"catalog_version": {"label": "v1.0", "description": "first"},
			"type_definitions": [],
			"entity_types": [{"name": "server", "description": "A server", "attributes": [], "associations": []}],
			"instances": []
		}
	}`

	rec := doImportRequest(e, "/api/data/v1/catalogs/import?dry_run=true", body, apimw.RoleAdmin)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// T-30.55b: POST /catalogs/import — service error → mapped error
func TestT30_55b_ImportServiceError(t *testing.T) {
	e, _, _, _, _, _, _, _, _, _, _, _, _, _ := setupImportServer()

	// Invalid format version will cause validation error
	body := `{
		"data": {
			"format_version": "2.0",
			"exported_at": "2026-04-23T14:30:00Z",
			"source_system": "test",
			"catalog": {"name": "test-catalog"},
			"catalog_version": {"label": "v1.0"},
			"type_definitions": [],
			"entity_types": [],
			"instances": []
		}
	}`

	rec := doImportRequest(e, "/api/data/v1/catalogs/import", body, apimw.RoleAdmin)
	assert.Equal(t, http.StatusBadRequest, rec.Code) // Validation error → 400
}

// T-30.54c: POST /catalogs/import — invalid JSON body → 400
func TestT30_54c_ImportInvalidBody(t *testing.T) {
	e, _, _, _, _, _, _, _, _, _, _, _, _, _ := setupImportServer()

	rec := doImportRequest(e, "/api/data/v1/catalogs/import", `{not valid json`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// Ensure unused import
var _ = time.Now
