package operational_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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

func setupValidationServer() (*echo.Echo, *mocks.MockCatalogRepo, *mocks.MockEntityInstanceRepo,
	*mocks.MockCatalogVersionPinRepo, *mocks.MockEntityTypeVersionRepo,
	*mocks.MockAttributeRepo, *mocks.MockAssociationRepo,
	*mocks.MockEnumValueRepo, *mocks.MockAssociationLinkRepo, *mocks.MockEntityTypeRepo,
	*mocks.MockInstanceAttributeValueRepo) {

	catRepo := new(mocks.MockCatalogRepo)
	instRepo := new(mocks.MockEntityInstanceRepo)
	iavRepo := new(mocks.MockInstanceAttributeValueRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	enumValRepo := new(mocks.MockEnumValueRepo)
	linkRepo := new(mocks.MockAssociationLinkRepo)
	etRepo := new(mocks.MockEntityTypeRepo)
	cvRepo := new(mocks.MockCatalogVersionRepo)

	catalogSvc := svcop.NewCatalogService(catRepo, cvRepo, instRepo)
	validationSvc := svcop.NewCatalogValidationService(
		catRepo, instRepo, iavRepo, pinRepo, etvRepo,
		attrRepo, assocRepo, enumValRepo, linkRepo, etRepo,
	)
	accessChecker := &apimw.HeaderCatalogAccessChecker{}
	handler := apiop.NewCatalogHandler(catalogSvc, validationSvc, accessChecker)

	e := echo.New()
	g := e.Group("/api/data/v1/catalogs")
	rbac := &apimw.HeaderRBACProvider{}
	g.Use(apimw.RBACMiddleware(rbac))
	requireRW := apimw.RequireRole(apimw.RoleRW)
	apiop.RegisterCatalogRoutes(g, handler, requireRW)

	return e, catRepo, instRepo, pinRepo, etvRepo, attrRepo, assocRepo, enumValRepo, linkRepo, etRepo, iavRepo
}

func doValidateRequest(e *echo.Echo, name string, role apimw.Role) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/data/v1/catalogs/"+name+"/validate", strings.NewReader(""))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("X-User-Role", string(role))
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// T-15.33: POST /validate returns 200 with valid catalog
func TestT15_33_ValidateValidCatalog(t *testing.T) {
	e, catRepo, instRepo, pinRepo, etvRepo, attrRepo, assocRepo, _, _, etRepo, iavRepo := setupValidationServer()

	catRepo.On("GetByName", mock.Anything, "test-catalog").Return(&models.Catalog{
		ID: "c1", Name: "test-catalog", CatalogVersionID: "cv1",
	}, nil)
	instRepo.On("ListByCatalog", mock.Anything, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "server-1"},
	}, nil)
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "etv1").Return(&models.EntityTypeVersion{
		ID: "etv1", EntityTypeID: "et1", Version: 1,
	}, nil)
	etRepo.On("GetByID", mock.Anything, "et1").Return(&models.EntityType{ID: "et1", Name: "Server"}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Attribute{}, nil)
	iavRepo.On("GetCurrentValues", mock.Anything, "inst1").Return([]*models.InstanceAttributeValue{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Association{}, nil)
	catRepo.On("UpdateValidationStatus", mock.Anything, "c1", models.ValidationStatusValid).Return(nil)

	rec := doValidateRequest(e, "test-catalog", apimw.RoleRW)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "valid", resp["status"])
	errors, ok := resp["errors"].([]interface{})
	require.True(t, ok)
	assert.Empty(t, errors)
}

// T-15.34: POST /validate returns 200 with invalid catalog
func TestT15_34_ValidateInvalidCatalog(t *testing.T) {
	e, catRepo, instRepo, pinRepo, etvRepo, attrRepo, assocRepo, _, _, etRepo, iavRepo := setupValidationServer()

	catRepo.On("GetByName", mock.Anything, "test-catalog").Return(&models.Catalog{
		ID: "c1", Name: "test-catalog", CatalogVersionID: "cv1",
	}, nil)
	instRepo.On("ListByCatalog", mock.Anything, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "server-1"},
	}, nil)
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "etv1").Return(&models.EntityTypeVersion{
		ID: "etv1", EntityTypeID: "et1", Version: 1,
	}, nil)
	etRepo.On("GetByID", mock.Anything, "et1").Return(&models.EntityType{ID: "et1", Name: "Server"}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Attribute{
		{ID: "attr1", Name: "hostname", Type: models.AttributeTypeString, Required: true},
	}, nil)
	iavRepo.On("GetCurrentValues", mock.Anything, "inst1").Return([]*models.InstanceAttributeValue{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Association{}, nil)
	catRepo.On("UpdateValidationStatus", mock.Anything, "c1", models.ValidationStatusInvalid).Return(nil)

	rec := doValidateRequest(e, "test-catalog", apimw.RoleRW)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "invalid", resp["status"])
	errors, ok := resp["errors"].([]interface{})
	require.True(t, ok)
	assert.Len(t, errors, 1)
}

// T-15.35: POST /validate with nonexistent catalog → 404
func TestT15_35_ValidateNotFound(t *testing.T) {
	e, catRepo, _, _, _, _, _, _, _, _, _ := setupValidationServer()

	catRepo.On("GetByName", mock.Anything, "nonexistent").Return(nil, domainerrors.NewNotFound("Catalog", "nonexistent"))

	rec := doValidateRequest(e, "nonexistent", apimw.RoleRW)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// T-15.36: POST /validate as RO → 403
func TestT15_36_ValidateRO(t *testing.T) {
	e, _, _, _, _, _, _, _, _, _, _ := setupValidationServer()

	rec := doValidateRequest(e, "test-catalog", apimw.RoleRO)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// T-15.37: POST /validate as RW → 200
func TestT15_37_ValidateRW(t *testing.T) {
	e, catRepo, instRepo, _, _, _, _, _, _, _, _ := setupValidationServer()

	catRepo.On("GetByName", mock.Anything, "test-catalog").Return(&models.Catalog{
		ID: "c1", Name: "test-catalog", CatalogVersionID: "cv1",
	}, nil)
	instRepo.On("ListByCatalog", mock.Anything, "c1").Return([]*models.EntityInstance{}, nil)
	catRepo.On("UpdateValidationStatus", mock.Anything, "c1", models.ValidationStatusValid).Return(nil)

	rec := doValidateRequest(e, "test-catalog", apimw.RoleRW)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// T-15.38: Validate response errors include all four fields
func TestT15_38_ValidateErrorFields(t *testing.T) {
	e, catRepo, instRepo, pinRepo, etvRepo, attrRepo, assocRepo, _, _, etRepo, iavRepo := setupValidationServer()

	catRepo.On("GetByName", mock.Anything, "test-catalog").Return(&models.Catalog{
		ID: "c1", Name: "test-catalog", CatalogVersionID: "cv1",
	}, nil)
	instRepo.On("ListByCatalog", mock.Anything, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "server-1"},
	}, nil)
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "etv1").Return(&models.EntityTypeVersion{
		ID: "etv1", EntityTypeID: "et1", Version: 1,
	}, nil)
	etRepo.On("GetByID", mock.Anything, "et1").Return(&models.EntityType{ID: "et1", Name: "Server"}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Attribute{
		{ID: "attr1", Name: "hostname", Type: models.AttributeTypeString, Required: true},
	}, nil)
	iavRepo.On("GetCurrentValues", mock.Anything, "inst1").Return([]*models.InstanceAttributeValue{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Association{}, nil)
	catRepo.On("UpdateValidationStatus", mock.Anything, "c1", models.ValidationStatusInvalid).Return(nil)

	rec := doValidateRequest(e, "test-catalog", apimw.RoleRW)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	errors := resp["errors"].([]interface{})
	require.Len(t, errors, 1)
	errObj := errors[0].(map[string]interface{})
	assert.Equal(t, "Server", errObj["entity_type"])
	assert.Equal(t, "server-1", errObj["instance_name"])
	assert.Equal(t, "hostname", errObj["field"])
	assert.NotEmpty(t, errObj["violation"])
}

// Cover: nil validationSvc returns 501
func TestValidateCatalog_NilService(t *testing.T) {
	catRepo := new(mocks.MockCatalogRepo)
	cvRepo := new(mocks.MockCatalogVersionRepo)
	instRepo := new(mocks.MockEntityInstanceRepo)
	catalogSvc := svcop.NewCatalogService(catRepo, cvRepo, instRepo)
	accessChecker := &apimw.HeaderCatalogAccessChecker{}
	handler := apiop.NewCatalogHandler(catalogSvc, nil, accessChecker)

	e := echo.New()
	g := e.Group("/api/data/v1/catalogs")
	rbac := &apimw.HeaderRBACProvider{}
	g.Use(apimw.RBACMiddleware(rbac))
	requireRW := apimw.RequireRole(apimw.RoleRW)
	apiop.RegisterCatalogRoutes(g, handler, requireRW)

	rec := doValidateRequest(e, "test-catalog", apimw.RoleRW)
	assert.Equal(t, http.StatusNotImplemented, rec.Code)
}
