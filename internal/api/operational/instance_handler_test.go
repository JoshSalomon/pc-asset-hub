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

func setupInstanceServer() (*echo.Echo, *instanceMocks) {
	m := newInstanceMocks()
	svc := svcop.NewInstanceService(
		m.instRepo, m.iavRepo, m.catalogRepo, m.cvRepo,
		m.pinRepo, m.attrRepo, m.etvRepo, m.etRepo, m.enumValRepo,
	)
	handler := apiop.NewInstanceHandler(svc)

	e := echo.New()
	g := e.Group("/api/data/v1/catalogs/:catalog-name")
	rbac := &apimw.HeaderRBACProvider{}
	g.Use(apimw.RBACMiddleware(rbac))
	requireRW := apimw.RequireRole(apimw.RoleRW)
	apiop.RegisterInstanceRoutes(g, handler, requireRW)

	return e, m
}

type instanceMocks struct {
	instRepo    *mocks.MockEntityInstanceRepo
	iavRepo     *mocks.MockInstanceAttributeValueRepo
	catalogRepo *mocks.MockCatalogRepo
	cvRepo      *mocks.MockCatalogVersionRepo
	pinRepo     *mocks.MockCatalogVersionPinRepo
	attrRepo    *mocks.MockAttributeRepo
	etvRepo     *mocks.MockEntityTypeVersionRepo
	etRepo      *mocks.MockEntityTypeRepo
	enumValRepo *mocks.MockEnumValueRepo
}

func newInstanceMocks() *instanceMocks {
	return &instanceMocks{
		instRepo:    new(mocks.MockEntityInstanceRepo),
		iavRepo:     new(mocks.MockInstanceAttributeValueRepo),
		catalogRepo: new(mocks.MockCatalogRepo),
		cvRepo:      new(mocks.MockCatalogVersionRepo),
		pinRepo:     new(mocks.MockCatalogVersionPinRepo),
		attrRepo:    new(mocks.MockAttributeRepo),
		etvRepo:     new(mocks.MockEntityTypeVersionRepo),
		etRepo:      new(mocks.MockEntityTypeRepo),
		enumValRepo: new(mocks.MockEnumValueRepo),
	}
}

func (m *instanceMocks) mockPinResolution() {
	m.catalogRepo.On("GetByName", mock.Anything, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	m.etRepo.On("GetByName", mock.Anything, "model").Return(&models.EntityType{ID: "et1", Name: "model"}, nil)
	m.pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
	}, nil)
	m.etvRepo.On("GetByID", mock.Anything, "etv1").Return(&models.EntityTypeVersion{
		ID: "etv1", EntityTypeID: "et1", Version: 1,
	}, nil)
	m.attrRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Attribute{
		{ID: "a1", Name: "hostname", Type: models.AttributeTypeString},
	}, nil)
}

func doInstanceRequest(e *echo.Echo, method, path, body string, role apimw.Role) *httptest.ResponseRecorder {
	reader := strings.NewReader(body)
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("X-User-Role", string(role))
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// T-11.35: POST /catalogs/{name}/{entity-type} with attributes → 201
func TestT11_35_CreateInstance(t *testing.T) {
	e, m := setupInstanceServer()
	m.mockPinResolution()

	m.instRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityInstance")).Return(nil)
	m.iavRepo.On("SetValues", mock.Anything, mock.Anything).Return(nil)
	m.iavRepo.On("GetCurrentValues", mock.Anything, mock.Anything).Return([]*models.InstanceAttributeValue{
		{ID: "v1", AttributeID: "a1", ValueString: "myhost", InstanceVersion: 1},
	}, nil)
	m.catalogRepo.On("UpdateValidationStatus", mock.Anything, "cat1", models.ValidationStatusDraft).Return(nil)

	rec := doInstanceRequest(e, http.MethodPost, "/api/data/v1/catalogs/my-catalog/model",
		`{"name":"my-inst","description":"desc","attributes":{"hostname":"myhost"}}`, apimw.RoleRW)

	assert.Equal(t, http.StatusCreated, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "my-inst", resp["name"])
	attrs := resp["attributes"].([]interface{})
	assert.Len(t, attrs, 1)
}

// T-11.36: POST nonexistent catalog → 404
func TestT11_36_NonexistentCatalog(t *testing.T) {
	e, m := setupInstanceServer()
	m.catalogRepo.On("GetByName", mock.Anything, "no-such").Return(nil, domainerrors.NewNotFound("Catalog", "no-such"))

	rec := doInstanceRequest(e, http.MethodPost, "/api/data/v1/catalogs/no-such/model",
		`{"name":"inst"}`, apimw.RoleRW)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// T-11.37: POST entity type not pinned → 404
func TestT11_37_EntityTypeNotPinned(t *testing.T) {
	e, m := setupInstanceServer()

	m.catalogRepo.On("GetByName", mock.Anything, "my-catalog").Return(&models.Catalog{
		ID: "cat1", CatalogVersionID: "cv1",
	}, nil)
	m.etRepo.On("GetByName", mock.Anything, "not-pinned").Return(&models.EntityType{ID: "et-other"}, nil)
	m.pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
	}, nil)
	m.etvRepo.On("GetByID", mock.Anything, "etv1").Return(&models.EntityTypeVersion{
		ID: "etv1", EntityTypeID: "et1", // et1 != et-other
	}, nil)

	rec := doInstanceRequest(e, http.MethodPost, "/api/data/v1/catalogs/my-catalog/not-pinned",
		`{"name":"inst"}`, apimw.RoleRW)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// T-11.38: POST invalid attribute value → 400 (unknown attribute)
func TestT11_38_InvalidAttributeValue(t *testing.T) {
	e, m := setupInstanceServer()
	m.mockPinResolution()
	m.instRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	rec := doInstanceRequest(e, http.MethodPost, "/api/data/v1/catalogs/my-catalog/model",
		`{"name":"inst","attributes":{"bogus":"val"}}`, apimw.RoleRW)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// T-11.39: POST as RO → 403
func TestT11_39_CreateAsRO(t *testing.T) {
	e, _ := setupInstanceServer()

	rec := doInstanceRequest(e, http.MethodPost, "/api/data/v1/catalogs/my-catalog/model",
		`{"name":"inst"}`, apimw.RoleRO)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// T-11.40: POST as RW → 201
func TestT11_40_CreateAsRW(t *testing.T) {
	e, m := setupInstanceServer()
	m.mockPinResolution()

	m.instRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	m.iavRepo.On("GetCurrentValues", mock.Anything, mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.catalogRepo.On("UpdateValidationStatus", mock.Anything, "cat1", models.ValidationStatusDraft).Return(nil)

	rec := doInstanceRequest(e, http.MethodPost, "/api/data/v1/catalogs/my-catalog/model",
		`{"name":"inst"}`, apimw.RoleRW)

	assert.Equal(t, http.StatusCreated, rec.Code)
}

// T-11.41: GET /catalogs/{name}/{entity-type} → 200 list with attributes
func TestT11_41_ListInstances(t *testing.T) {
	e, m := setupInstanceServer()
	m.mockPinResolution()

	m.instRepo.On("List", mock.Anything, "et1", "cat1", mock.AnythingOfType("models.ListParams")).Return([]*models.EntityInstance{
		{ID: "i1", EntityTypeID: "et1", CatalogID: "cat1", Name: "inst-a", Version: 1},
	}, 1, nil)
	m.iavRepo.On("GetCurrentValues", mock.Anything, "i1").Return([]*models.InstanceAttributeValue{
		{AttributeID: "a1", ValueString: "host-a"},
	}, nil)

	rec := doInstanceRequest(e, http.MethodGet, "/api/data/v1/catalogs/my-catalog/model", "", apimw.RoleRO)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, float64(1), resp["total"])
}

// T-11.42: GET /catalogs/{name}/{entity-type}/{id} → 200 with attributes
func TestT11_42_GetInstance(t *testing.T) {
	e, m := setupInstanceServer()
	m.mockPinResolution()

	m.instRepo.On("GetByID", mock.Anything, "i1").Return(&models.EntityInstance{
		ID: "i1", EntityTypeID: "et1", CatalogID: "cat1", Name: "inst-a", Version: 1,
	}, nil)
	m.iavRepo.On("GetCurrentValues", mock.Anything, "i1").Return([]*models.InstanceAttributeValue{
		{AttributeID: "a1", ValueString: "host-a"},
	}, nil)

	rec := doInstanceRequest(e, http.MethodGet, "/api/data/v1/catalogs/my-catalog/model/i1", "", apimw.RoleRO)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "inst-a", resp["name"])
	attrs := resp["attributes"].([]interface{})
	assert.Len(t, attrs, 1)
}

// T-11.43: GET nonexistent instance → 404
func TestT11_43_GetNotFound(t *testing.T) {
	e, m := setupInstanceServer()
	m.mockPinResolution()

	m.instRepo.On("GetByID", mock.Anything, "nope").Return(nil, domainerrors.NewNotFound("EntityInstance", "nope"))

	rec := doInstanceRequest(e, http.MethodGet, "/api/data/v1/catalogs/my-catalog/model/nope", "", apimw.RoleRO)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// T-11.44: PUT /catalogs/{name}/{entity-type}/{id} → 200, version incremented
func TestT11_44_UpdateInstance(t *testing.T) {
	e, m := setupInstanceServer()
	m.mockPinResolution()

	m.instRepo.On("GetByID", mock.Anything, "i1").Return(&models.EntityInstance{
		ID: "i1", EntityTypeID: "et1", CatalogID: "cat1", Name: "inst-a", Version: 1,
	}, nil)
	m.instRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
	m.iavRepo.On("GetValuesForVersion", mock.Anything, "i1", 1).Return([]*models.InstanceAttributeValue{}, nil)
	m.iavRepo.On("SetValues", mock.Anything, mock.Anything).Return(nil)
	m.iavRepo.On("GetCurrentValues", mock.Anything, mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.catalogRepo.On("UpdateValidationStatus", mock.Anything, "cat1", models.ValidationStatusDraft).Return(nil)

	rec := doInstanceRequest(e, http.MethodPut, "/api/data/v1/catalogs/my-catalog/model/i1",
		`{"version":1,"attributes":{"hostname":"newhost"}}`, apimw.RoleRW)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, float64(2), resp["version"])
}

// T-11.45: PUT version mismatch → 409
func TestT11_45_UpdateVersionMismatch(t *testing.T) {
	e, m := setupInstanceServer()
	m.mockPinResolution()

	m.instRepo.On("GetByID", mock.Anything, "i1").Return(&models.EntityInstance{
		ID: "i1", Version: 3,
	}, nil)

	rec := doInstanceRequest(e, http.MethodPut, "/api/data/v1/catalogs/my-catalog/model/i1",
		`{"version":1}`, apimw.RoleRW)

	assert.Equal(t, http.StatusConflict, rec.Code)
}

// T-11.46: DELETE /catalogs/{name}/{entity-type}/{id} → 204
func TestT11_46_DeleteInstance(t *testing.T) {
	e, m := setupInstanceServer()
	m.mockPinResolution()

	m.instRepo.On("ListByParent", mock.Anything, "i1", mock.Anything).Return([]*models.EntityInstance{}, 0, nil)
	m.instRepo.On("SoftDelete", mock.Anything, "i1").Return(nil)
	m.catalogRepo.On("UpdateValidationStatus", mock.Anything, "cat1", models.ValidationStatusDraft).Return(nil)

	rec := doInstanceRequest(e, http.MethodDelete, "/api/data/v1/catalogs/my-catalog/model/i1", "", apimw.RoleRW)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

// T-11.47: DELETE as RO → 403
func TestT11_47_DeleteAsRO(t *testing.T) {
	e, _ := setupInstanceServer()

	rec := doInstanceRequest(e, http.MethodDelete, "/api/data/v1/catalogs/my-catalog/model/i1", "", apimw.RoleRO)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// === Coverage tests for uncovered handler lines ===

// Bind error: CreateInstance with malformed JSON
func TestCov_CreateInstance_BindError(t *testing.T) {
	e, _ := setupInstanceServer()
	rec := doInstanceRequest(e, http.MethodPost, "/api/data/v1/catalogs/my-catalog/model", "not-json{{{", apimw.RoleRW)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// Bind error: UpdateInstance with malformed JSON
func TestCov_UpdateInstance_BindError(t *testing.T) {
	e, _ := setupInstanceServer()
	rec := doInstanceRequest(e, http.MethodPut, "/api/data/v1/catalogs/my-catalog/model/i1", "not-json{{{", apimw.RoleRW)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// Error propagation: ListInstances service error
func TestCov_ListInstances_ServiceError(t *testing.T) {
	e, m := setupInstanceServer()
	m.catalogRepo.On("GetByName", mock.Anything, "my-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "bad"))

	rec := doInstanceRequest(e, http.MethodGet, "/api/data/v1/catalogs/my-catalog/model", "", apimw.RoleRO)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// Error propagation: DeleteInstance service error
func TestCov_DeleteInstance_ServiceError(t *testing.T) {
	e, m := setupInstanceServer()
	m.catalogRepo.On("GetByName", mock.Anything, "my-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "bad"))

	rec := doInstanceRequest(e, http.MethodDelete, "/api/data/v1/catalogs/my-catalog/model/i1", "", apimw.RoleRW)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}
