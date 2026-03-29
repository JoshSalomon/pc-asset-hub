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
		m.assocRepo, m.linkRepo,
	)
	handler := apiop.NewInstanceHandler(svc, nil)

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
	assocRepo   *mocks.MockAssociationRepo
	linkRepo    *mocks.MockAssociationLinkRepo
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
		assocRepo:   new(mocks.MockAssociationRepo),
		linkRepo:    new(mocks.MockAssociationLinkRepo),
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
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "my-inst", resp["name"])
	attrs := resp["attributes"].([]any)
	assert.Len(t, attrs, 3) // 2 system + 1 custom
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
	var resp map[string]any
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
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "inst-a", resp["name"])
	attrs := resp["attributes"].([]any)
	assert.Len(t, attrs, 3) // 2 system + 1 custom
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
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, float64(2), resp["version"])
}

// T-11.45: PUT version mismatch → 409
func TestT11_45_UpdateVersionMismatch(t *testing.T) {
	e, m := setupInstanceServer()
	m.mockPinResolution()

	m.instRepo.On("GetByID", mock.Anything, "i1").Return(&models.EntityInstance{
		ID: "i1", CatalogID: "cat1", Version: 3,
	}, nil)

	rec := doInstanceRequest(e, http.MethodPut, "/api/data/v1/catalogs/my-catalog/model/i1",
		`{"version":1}`, apimw.RoleRW)

	assert.Equal(t, http.StatusConflict, rec.Code)
}

// T-11.46: DELETE /catalogs/{name}/{entity-type}/{id} → 204
func TestT11_46_DeleteInstance(t *testing.T) {
	e, m := setupInstanceServer()
	m.mockPinResolution()

	m.instRepo.On("GetByID", mock.Anything, "i1").Return(&models.EntityInstance{
		ID: "i1", CatalogID: "cat1",
	}, nil)
	m.instRepo.On("ListByParent", mock.Anything, "i1", mock.Anything).Return([]*models.EntityInstance{}, 0, nil)
	m.linkRepo.On("DeleteByInstance", mock.Anything, "i1").Return(nil)
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

// === SetParent Handler Tests ===

func TestSetParent_Handler_Success(t *testing.T) {
	e, m := setupInstanceServer()
	mockTwoTypePinResolution(m)

	m.instRepo.On("GetByID", mock.Anything, "c1").Return(&models.EntityInstance{
		ID: "c1", EntityTypeID: "et2", CatalogID: "cat1", Version: 1,
	}, nil)
	m.instRepo.On("GetByID", mock.Anything, "p1").Return(&models.EntityInstance{
		ID: "p1", EntityTypeID: "et1", CatalogID: "cat1", Version: 1,
	}, nil)
	m.assocRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Association{
		{ID: "a1", TargetEntityTypeID: "et2", Type: models.AssociationTypeContainment},
	}, nil)
	m.instRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
	m.catalogRepo.On("UpdateValidationStatus", mock.Anything, "cat1", models.ValidationStatusDraft).Return(nil)

	rec := doInstanceRequest(e, http.MethodPut, "/api/data/v1/catalogs/my-catalog/tool/c1/parent",
		`{"parent_type":"server","parent_instance_id":"p1"}`, apimw.RoleRW)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestSetParent_Handler_BindError(t *testing.T) {
	e, _ := setupInstanceServer()
	rec := doInstanceRequest(e, http.MethodPut, "/api/data/v1/catalogs/my-catalog/tool/c1/parent", "bad{json", apimw.RoleRW)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSetParent_Handler_ServiceError(t *testing.T) {
	e, m := setupInstanceServer()
	m.catalogRepo.On("GetByName", mock.Anything, "my-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "bad"))
	rec := doInstanceRequest(e, http.MethodPut, "/api/data/v1/catalogs/my-catalog/tool/c1/parent",
		`{"parent_type":"server","parent_instance_id":"p1"}`, apimw.RoleRW)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestSetParent_Handler_AsRO(t *testing.T) {
	e, _ := setupInstanceServer()
	rec := doInstanceRequest(e, http.MethodPut, "/api/data/v1/catalogs/my-catalog/tool/c1/parent",
		`{"parent_type":"server","parent_instance_id":"p1"}`, apimw.RoleRO)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// === Milestone 12: Handler Coverage Tests ===

func TestCov_CreateContained_BindError(t *testing.T) {
	e, _ := setupInstanceServer()
	rec := doInstanceRequest(e, http.MethodPost, "/api/data/v1/catalogs/my-catalog/server/p1/tool", "bad{json", apimw.RoleRW)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCov_CreateContained_ServiceError(t *testing.T) {
	e, m := setupInstanceServer()
	m.catalogRepo.On("GetByName", mock.Anything, "my-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "bad"))
	rec := doInstanceRequest(e, http.MethodPost, "/api/data/v1/catalogs/my-catalog/server/p1/tool", `{"name":"c"}`, apimw.RoleRW)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestCov_ListContained_ServiceError(t *testing.T) {
	e, m := setupInstanceServer()
	m.catalogRepo.On("GetByName", mock.Anything, "my-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "bad"))
	rec := doInstanceRequest(e, http.MethodGet, "/api/data/v1/catalogs/my-catalog/server/p1/tool", "", apimw.RoleRO)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestCov_CreateLink_BindError(t *testing.T) {
	e, _ := setupInstanceServer()
	rec := doInstanceRequest(e, http.MethodPost, "/api/data/v1/catalogs/my-catalog/server/i1/links", "bad{json", apimw.RoleRW)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestCov_CreateLink_ServiceError(t *testing.T) {
	e, m := setupInstanceServer()
	m.catalogRepo.On("GetByName", mock.Anything, "my-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "bad"))
	rec := doInstanceRequest(e, http.MethodPost, "/api/data/v1/catalogs/my-catalog/server/i1/links",
		`{"target_instance_id":"i2","association_name":"uses"}`, apimw.RoleRW)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestCov_DeleteLink_ServiceError(t *testing.T) {
	e, m := setupInstanceServer()
	m.catalogRepo.On("GetByName", mock.Anything, "my-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "bad"))
	rec := doInstanceRequest(e, http.MethodDelete, "/api/data/v1/catalogs/my-catalog/server/i1/links/l1", "", apimw.RoleRW)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestCov_GetForwardRefs_ServiceError(t *testing.T) {
	e, m := setupInstanceServer()
	m.catalogRepo.On("GetByName", mock.Anything, "my-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "bad"))
	rec := doInstanceRequest(e, http.MethodGet, "/api/data/v1/catalogs/my-catalog/server/i1/references", "", apimw.RoleRO)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestCov_GetReverseRefs_ServiceError(t *testing.T) {
	e, m := setupInstanceServer()
	m.catalogRepo.On("GetByName", mock.Anything, "my-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "bad"))
	rec := doInstanceRequest(e, http.MethodGet, "/api/data/v1/catalogs/my-catalog/server/i1/referenced-by", "", apimw.RoleRO)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// === Milestone 12: Handler Tests ===

func mockTwoTypePinResolution(m *instanceMocks) {
	m.catalogRepo.On("GetByName", mock.Anything, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusDraft,
	}, nil)
	m.etRepo.On("GetByName", mock.Anything, "server").Return(&models.EntityType{ID: "et1", Name: "server"}, nil)
	m.etRepo.On("GetByName", mock.Anything, "tool").Return(&models.EntityType{ID: "et2", Name: "tool"}, nil)
	m.pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
		{ID: "pin2", CatalogVersionID: "cv1", EntityTypeVersionID: "etv2"},
	}, nil)
	m.etvRepo.On("GetByID", mock.Anything, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	m.etvRepo.On("GetByID", mock.Anything, "etv2").Return(&models.EntityTypeVersion{ID: "etv2", EntityTypeID: "et2", Version: 1}, nil)
	m.attrRepo.On("ListByVersion", mock.Anything, "etv2").Return([]*models.Attribute{}, nil)
}

// T-12.37: POST /{catalog}/{parent-type}/{parent-id}/{child-type} → 201
func TestT12_37_CreateContainedInstance(t *testing.T) {
	e, m := setupInstanceServer()
	mockTwoTypePinResolution(m)

	m.instRepo.On("GetByID", mock.Anything, "p1").Return(&models.EntityInstance{
		ID: "p1", EntityTypeID: "et1", CatalogID: "cat1", Version: 1,
	}, nil)
	m.assocRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Association{
		{ID: "assoc1", EntityTypeVersionID: "etv1", TargetEntityTypeID: "et2", Type: models.AssociationTypeContainment, Name: "tools"},
	}, nil)
	m.instRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityInstance")).Return(nil)
	m.iavRepo.On("GetCurrentValues", mock.Anything, mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.catalogRepo.On("UpdateValidationStatus", mock.Anything, "cat1", models.ValidationStatusDraft).Return(nil)

	rec := doInstanceRequest(e, http.MethodPost, "/api/data/v1/catalogs/my-catalog/server/p1/tool",
		`{"name":"my-tool","description":"desc"}`, apimw.RoleRW)

	assert.Equal(t, http.StatusCreated, rec.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "my-tool", resp["name"])
	assert.Equal(t, "p1", resp["parent_instance_id"])
}

// T-12.38: POST contained with nonexistent parent → 404
func TestT12_38_ContainedNonexistentParent(t *testing.T) {
	e, m := setupInstanceServer()
	mockTwoTypePinResolution(m)

	m.instRepo.On("GetByID", mock.Anything, "nope").Return(nil, domainerrors.NewNotFound("EntityInstance", "nope"))

	rec := doInstanceRequest(e, http.MethodPost, "/api/data/v1/catalogs/my-catalog/server/nope/tool",
		`{"name":"child"}`, apimw.RoleRW)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// T-12.40: POST contained as RO → 403
func TestT12_40_ContainedAsRO(t *testing.T) {
	e, _ := setupInstanceServer()

	rec := doInstanceRequest(e, http.MethodPost, "/api/data/v1/catalogs/my-catalog/server/p1/tool",
		`{"name":"child"}`, apimw.RoleRO)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// T-12.41: GET /{catalog}/{parent-type}/{parent-id}/{child-type} → 200
func TestT12_41_ListContainedInstances(t *testing.T) {
	e, m := setupInstanceServer()
	mockTwoTypePinResolution(m)

	m.instRepo.On("GetByID", mock.Anything, "p1").Return(&models.EntityInstance{
		ID: "p1", EntityTypeID: "et1", CatalogID: "cat1", Version: 1,
	}, nil)
	m.instRepo.On("ListByParent", mock.Anything, "p1", mock.Anything).Return([]*models.EntityInstance{
		{ID: "c1", EntityTypeID: "et2", CatalogID: "cat1", ParentInstanceID: "p1", Name: "tool-a"},
	}, 1, nil)
	m.iavRepo.On("GetCurrentValues", mock.Anything, "c1").Return([]*models.InstanceAttributeValue{}, nil)

	rec := doInstanceRequest(e, http.MethodGet, "/api/data/v1/catalogs/my-catalog/server/p1/tool", "", apimw.RoleRO)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	items := resp["items"].([]any)
	assert.Len(t, items, 1)
}

// TD-27: ListContainedInstances respects pagination query params
func TestTD27_ListContainedWithPagination(t *testing.T) {
	e, m := setupInstanceServer()
	mockTwoTypePinResolution(m)

	m.instRepo.On("GetByID", mock.Anything, "p1").Return(&models.EntityInstance{
		ID: "p1", EntityTypeID: "et1", CatalogID: "cat1", Version: 1,
	}, nil)
	m.instRepo.On("ListByParent", mock.Anything, "p1", mock.MatchedBy(func(p models.ListParams) bool {
		return p.Limit == 5 && p.Offset == 10 && p.SortBy == "name" && p.SortDesc == true
	})).Return([]*models.EntityInstance{}, 0, nil)

	rec := doInstanceRequest(e, http.MethodGet,
		"/api/data/v1/catalogs/my-catalog/server/p1/tool?limit=5&offset=10&sort=name:desc", "", apimw.RoleRO)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// TD-27: ListContainedInstances respects filter query params
func TestTD27_ListContainedWithFilter(t *testing.T) {
	e, m := setupInstanceServer()
	mockTwoTypePinResolution(m)

	m.instRepo.On("GetByID", mock.Anything, "p1").Return(&models.EntityInstance{
		ID: "p1", EntityTypeID: "et1", CatalogID: "cat1", Version: 1,
	}, nil)
	m.instRepo.On("ListByParent", mock.Anything, "p1", mock.MatchedBy(func(p models.ListParams) bool {
		return p.Filters != nil && p.Filters["hostname"] == "web"
	})).Return([]*models.EntityInstance{}, 0, nil)

	rec := doInstanceRequest(e, http.MethodGet,
		"/api/data/v1/catalogs/my-catalog/server/p1/tool?filter.hostname=web", "", apimw.RoleRO)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// T-12.43: POST /{catalog}/{type}/{id}/links → 201
func TestT12_43_CreateLink(t *testing.T) {
	e, m := setupInstanceServer()
	mockTwoTypePinResolution(m)

	m.instRepo.On("GetByID", mock.Anything, "inst1").Return(&models.EntityInstance{
		ID: "inst1", EntityTypeID: "et1", CatalogID: "cat1", Version: 1,
	}, nil)
	m.instRepo.On("GetByID", mock.Anything, "inst2").Return(&models.EntityInstance{
		ID: "inst2", EntityTypeID: "et2", CatalogID: "cat1", Version: 1,
	}, nil)
	m.assocRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Association{
		{ID: "assoc1", EntityTypeVersionID: "etv1", TargetEntityTypeID: "et2",
			Type: models.AssociationTypeDirectional, Name: "uses"},
	}, nil)
	m.linkRepo.On("GetForwardRefs", mock.Anything, "inst1").Return([]*models.AssociationLink{}, nil)
	m.linkRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.AssociationLink")).Return(nil)
	m.catalogRepo.On("UpdateValidationStatus", mock.Anything, "cat1", models.ValidationStatusDraft).Return(nil)

	rec := doInstanceRequest(e, http.MethodPost, "/api/data/v1/catalogs/my-catalog/server/inst1/links",
		`{"target_instance_id":"inst2","association_name":"uses"}`, apimw.RoleRW)

	assert.Equal(t, http.StatusCreated, rec.Code)
}

// T-12.47: POST link as RO → 403
func TestT12_47_LinkAsRO(t *testing.T) {
	e, _ := setupInstanceServer()

	rec := doInstanceRequest(e, http.MethodPost, "/api/data/v1/catalogs/my-catalog/server/inst1/links",
		`{"target_instance_id":"inst2","association_name":"uses"}`, apimw.RoleRO)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// T-12.48: DELETE /{catalog}/{type}/{id}/links/{link-id} → 204
func TestT12_48_DeleteLink(t *testing.T) {
	e, m := setupInstanceServer()
	mockTwoTypePinResolution(m)

	m.linkRepo.On("GetByID", mock.Anything, "link1").Return(&models.AssociationLink{
		ID: "link1", SourceInstanceID: "inst1",
	}, nil)
	m.instRepo.On("GetByID", mock.Anything, "inst1").Return(&models.EntityInstance{
		ID: "inst1", EntityTypeID: "et1", CatalogID: "cat1",
	}, nil)
	m.linkRepo.On("Delete", mock.Anything, "link1").Return(nil)
	m.catalogRepo.On("UpdateValidationStatus", mock.Anything, "cat1", models.ValidationStatusDraft).Return(nil)

	rec := doInstanceRequest(e, http.MethodDelete, "/api/data/v1/catalogs/my-catalog/server/inst1/links/link1", "", apimw.RoleRW)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

// T-12.49: DELETE link as RO → 403
func TestT12_49_DeleteLinkAsRO(t *testing.T) {
	e, _ := setupInstanceServer()

	rec := doInstanceRequest(e, http.MethodDelete, "/api/data/v1/catalogs/my-catalog/server/inst1/links/link1", "", apimw.RoleRO)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// T-12.51: GET /{catalog}/{type}/{id}/references → 200
func TestT12_51_GetForwardRefs(t *testing.T) {
	e, m := setupInstanceServer()
	mockTwoTypePinResolution(m)

	m.instRepo.On("GetByID", mock.Anything, "inst1").Return(&models.EntityInstance{
		ID: "inst1", EntityTypeID: "et1", CatalogID: "cat1", Version: 1,
	}, nil)
	m.linkRepo.On("GetForwardRefs", mock.Anything, "inst1").Return([]*models.AssociationLink{
		{ID: "link1", AssociationID: "assoc1", SourceInstanceID: "inst1", TargetInstanceID: "inst2"},
	}, nil)
	m.assocRepo.On("GetByID", mock.Anything, "assoc1").Return(&models.Association{
		ID: "assoc1", Name: "uses", Type: models.AssociationTypeDirectional, TargetEntityTypeID: "et2",
	}, nil)
	m.instRepo.On("GetByID", mock.Anything, "inst2").Return(&models.EntityInstance{
		ID: "inst2", EntityTypeID: "et2", CatalogID: "cat1", Name: "my-model", Version: 1,
	}, nil)
	m.etRepo.On("GetByID", mock.Anything, "et2").Return(&models.EntityType{ID: "et2", Name: "model"}, nil)

	rec := doInstanceRequest(e, http.MethodGet, "/api/data/v1/catalogs/my-catalog/server/inst1/references", "", apimw.RoleRO)

	assert.Equal(t, http.StatusOK, rec.Code)
	var refs []map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &refs))
	assert.Len(t, refs, 1)
	assert.Equal(t, "uses", refs[0]["association_name"])
	assert.Equal(t, "my-model", refs[0]["instance_name"])
}

// T-12.52: GET /{catalog}/{type}/{id}/referenced-by → 200
func TestT12_52_GetReverseRefs(t *testing.T) {
	e, m := setupInstanceServer()
	mockTwoTypePinResolution(m)

	m.instRepo.On("GetByID", mock.Anything, "inst2").Return(&models.EntityInstance{
		ID: "inst2", EntityTypeID: "et2", CatalogID: "cat1", Name: "my-tool", Version: 1,
	}, nil)
	m.linkRepo.On("GetReverseRefs", mock.Anything, "inst2").Return([]*models.AssociationLink{
		{ID: "link1", AssociationID: "assoc1", SourceInstanceID: "inst1", TargetInstanceID: "inst2"},
	}, nil)
	m.assocRepo.On("GetByID", mock.Anything, "assoc1").Return(&models.Association{
		ID: "assoc1", Name: "uses", Type: models.AssociationTypeDirectional,
	}, nil)
	m.instRepo.On("GetByID", mock.Anything, "inst1").Return(&models.EntityInstance{
		ID: "inst1", EntityTypeID: "et1", CatalogID: "cat1", Name: "my-server", Version: 1,
	}, nil)
	m.etRepo.On("GetByID", mock.Anything, "et1").Return(&models.EntityType{ID: "et1", Name: "server"}, nil)

	rec := doInstanceRequest(e, http.MethodGet, "/api/data/v1/catalogs/my-catalog/tool/inst2/referenced-by", "", apimw.RoleRO)

	assert.Equal(t, http.StatusOK, rec.Code)
	var refs []map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &refs))
	assert.Len(t, refs, 1)
	assert.Equal(t, "my-server", refs[0]["instance_name"])
}

// === Phase 4: Containment Tree Handler Tests ===

func TestGetContainmentTree_Success(t *testing.T) {
	e, m := setupInstanceServer()

	m.catalogRepo.On("GetByName", mock.Anything, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	m.instRepo.On("ListByCatalog", mock.Anything, "cat1").Return([]*models.EntityInstance{
		{ID: "p1", EntityTypeID: "et1", CatalogID: "cat1", Name: "parent"},
		{ID: "c1", EntityTypeID: "et2", CatalogID: "cat1", ParentInstanceID: "p1", Name: "child"},
	}, nil)
	m.etRepo.On("GetByID", mock.Anything, "et1").Return(&models.EntityType{ID: "et1", Name: "Server"}, nil)
	m.etRepo.On("GetByID", mock.Anything, "et2").Return(&models.EntityType{ID: "et2", Name: "Tool"}, nil)

	rec := doInstanceRequest(e, http.MethodGet, "/api/data/v1/catalogs/my-catalog/tree", "", apimw.RoleRO)
	assert.Equal(t, http.StatusOK, rec.Code)

	var tree []map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &tree))
	require.Len(t, tree, 1)
	assert.Equal(t, "parent", tree[0]["instance_name"])
	assert.Equal(t, "Server", tree[0]["entity_type_name"])
	children := tree[0]["children"].([]any)
	require.Len(t, children, 1)
	child := children[0].(map[string]any)
	assert.Equal(t, "child", child["instance_name"])
}

func TestGetContainmentTree_NotFound(t *testing.T) {
	e, m := setupInstanceServer()

	m.catalogRepo.On("GetByName", mock.Anything, "nope").Return(nil, domainerrors.NewNotFound("Catalog", "nope"))

	rec := doInstanceRequest(e, http.MethodGet, "/api/data/v1/catalogs/nope/tree", "", apimw.RoleRO)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetContainmentTree_EmptyCatalog(t *testing.T) {
	e, m := setupInstanceServer()

	m.catalogRepo.On("GetByName", mock.Anything, "empty").Return(&models.Catalog{
		ID: "cat-empty", Name: "empty", CatalogVersionID: "cv1",
	}, nil)
	m.instRepo.On("ListByCatalog", mock.Anything, "cat-empty").Return([]*models.EntityInstance{}, nil)

	rec := doInstanceRequest(e, http.MethodGet, "/api/data/v1/catalogs/empty/tree", "", apimw.RoleRO)
	assert.Equal(t, http.StatusOK, rec.Code)

	var tree []map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &tree))
	assert.Len(t, tree, 0)
}

// === Phase 4: ListInstances Query Params ===

func TestListInstances_WithPagination(t *testing.T) {
	e, m := setupInstanceServer()
	m.mockPinResolution()
	m.instRepo.On("List", mock.Anything, "et1", "cat1", mock.MatchedBy(func(p models.ListParams) bool {
		return p.Limit == 5 && p.Offset == 10
	})).Return([]*models.EntityInstance{}, 0, nil)

	rec := doInstanceRequest(e, http.MethodGet, "/api/data/v1/catalogs/my-catalog/model?limit=5&offset=10", "", apimw.RoleRO)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestListInstances_WithSort(t *testing.T) {
	e, m := setupInstanceServer()
	m.mockPinResolution()
	m.instRepo.On("List", mock.Anything, "et1", "cat1", mock.MatchedBy(func(p models.ListParams) bool {
		return p.SortBy == "name" && p.SortDesc == true
	})).Return([]*models.EntityInstance{}, 0, nil)

	rec := doInstanceRequest(e, http.MethodGet, "/api/data/v1/catalogs/my-catalog/model?sort=name:desc", "", apimw.RoleRO)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestListInstances_WithFilter(t *testing.T) {
	e, m := setupInstanceServer()
	m.mockPinResolution()
	m.instRepo.On("List", mock.Anything, "et1", "cat1", mock.MatchedBy(func(p models.ListParams) bool {
		return p.Filters != nil && p.Filters["a1"] == "hello"
	})).Return([]*models.EntityInstance{}, 0, nil)

	rec := doInstanceRequest(e, http.MethodGet, "/api/data/v1/catalogs/my-catalog/model?filter.hostname=hello", "", apimw.RoleRO)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestListInstances_LimitCappedAt100(t *testing.T) {
	e, m := setupInstanceServer()
	m.mockPinResolution()
	m.instRepo.On("List", mock.Anything, "et1", "cat1", mock.MatchedBy(func(p models.ListParams) bool {
		return p.Limit == 100
	})).Return([]*models.EntityInstance{}, 0, nil)

	rec := doInstanceRequest(e, http.MethodGet, "/api/data/v1/catalogs/my-catalog/model?limit=500", "", apimw.RoleRO)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// === Phase 4: GetInstance with Parent Chain ===

func TestListInstances_DefaultLimit20(t *testing.T) {
	e, m := setupInstanceServer()
	m.mockPinResolution()
	m.instRepo.On("List", mock.Anything, "et1", "cat1", mock.MatchedBy(func(p models.ListParams) bool {
		return p.Limit == 20
	})).Return([]*models.EntityInstance{}, 0, nil)

	rec := doInstanceRequest(e, http.MethodGet, "/api/data/v1/catalogs/my-catalog/model", "", apimw.RoleRO)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestListInstances_SortAsc(t *testing.T) {
	e, m := setupInstanceServer()
	m.mockPinResolution()
	m.instRepo.On("List", mock.Anything, "et1", "cat1", mock.MatchedBy(func(p models.ListParams) bool {
		return p.SortBy == "name" && p.SortDesc == false
	})).Return([]*models.EntityInstance{}, 0, nil)

	rec := doInstanceRequest(e, http.MethodGet, "/api/data/v1/catalogs/my-catalog/model?sort=name:asc", "", apimw.RoleRO)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestListInstances_NoSortDefault(t *testing.T) {
	e, m := setupInstanceServer()
	m.mockPinResolution()
	m.instRepo.On("List", mock.Anything, "et1", "cat1", mock.MatchedBy(func(p models.ListParams) bool {
		return p.SortBy == ""
	})).Return([]*models.EntityInstance{}, 0, nil)

	rec := doInstanceRequest(e, http.MethodGet, "/api/data/v1/catalogs/my-catalog/model", "", apimw.RoleRO)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestListInstances_MultipleFilters(t *testing.T) {
	e, m := setupInstanceServer()
	// Need two attributes in the mock
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
		{ID: "a2", Name: "region", Type: models.AttributeTypeString},
	}, nil)

	m.instRepo.On("List", mock.Anything, "et1", "cat1", mock.MatchedBy(func(p models.ListParams) bool {
		return p.Filters != nil && p.Filters["a1"] == "web" && p.Filters["a2"] == "us"
	})).Return([]*models.EntityInstance{}, 0, nil)

	rec := doInstanceRequest(e, http.MethodGet, "/api/data/v1/catalogs/my-catalog/model?filter.hostname=web&filter.region=us", "", apimw.RoleRO)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetInstance_RootNoParentChain(t *testing.T) {
	e, m := setupInstanceServer()
	m.mockPinResolution()

	m.instRepo.On("GetByID", mock.Anything, "root1").Return(&models.EntityInstance{
		ID: "root1", EntityTypeID: "et1", CatalogID: "cat1",
		Name: "root-instance", Version: 1,
	}, nil)
	m.iavRepo.On("GetCurrentValues", mock.Anything, "root1").Return([]*models.InstanceAttributeValue{}, nil)

	rec := doInstanceRequest(e, http.MethodGet, "/api/data/v1/catalogs/my-catalog/model/root1", "", apimw.RoleRO)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	// parent_chain should be absent or null for root instances
	chain, exists := resp["parent_chain"]
	if exists {
		assert.Nil(t, chain)
	}
}

func TestGetContainmentTree_TreeStructure(t *testing.T) {
	e, m := setupInstanceServer()

	m.catalogRepo.On("GetByName", mock.Anything, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	m.instRepo.On("ListByCatalog", mock.Anything, "cat1").Return([]*models.EntityInstance{
		{ID: "p1", EntityTypeID: "et1", CatalogID: "cat1", Name: "parent", Description: "parent desc"},
		{ID: "c1", EntityTypeID: "et2", CatalogID: "cat1", ParentInstanceID: "p1", Name: "child", Description: "child desc"},
	}, nil)
	m.etRepo.On("GetByID", mock.Anything, "et1").Return(&models.EntityType{ID: "et1", Name: "Server"}, nil)
	m.etRepo.On("GetByID", mock.Anything, "et2").Return(&models.EntityType{ID: "et2", Name: "Tool"}, nil)

	rec := doInstanceRequest(e, http.MethodGet, "/api/data/v1/catalogs/my-catalog/tree", "", apimw.RoleRO)
	assert.Equal(t, http.StatusOK, rec.Code)

	var tree []map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &tree))
	require.Len(t, tree, 1)

	// Verify root node has all expected fields
	root := tree[0]
	assert.NotEmpty(t, root["instance_id"])
	assert.Equal(t, "parent", root["instance_name"])
	assert.Equal(t, "Server", root["entity_type_name"])
	assert.Equal(t, "parent desc", root["description"])
	assert.NotNil(t, root["children"])

	// Verify child node structure
	children := root["children"].([]any)
	require.Len(t, children, 1)
	child := children[0].(map[string]any)
	assert.NotEmpty(t, child["instance_id"])
	assert.Equal(t, "child", child["instance_name"])
	assert.Equal(t, "Tool", child["entity_type_name"])
	assert.Equal(t, "child desc", child["description"])
	// Leaf node should have empty children
	childChildren := child["children"].([]any)
	assert.Len(t, childChildren, 0)
}

func TestGetInstance_IncludesParentChain(t *testing.T) {
	e, m := setupInstanceServer()
	m.mockPinResolution()

	childInst := &models.EntityInstance{
		ID: "child1", EntityTypeID: "et1", CatalogID: "cat1",
		ParentInstanceID: "parent1", Name: "child", Version: 1,
	}
	parentInst := &models.EntityInstance{
		ID: "parent1", EntityTypeID: "et1", CatalogID: "cat1",
		Name: "parent", Version: 1,
	}
	m.instRepo.On("GetByID", mock.Anything, "child1").Return(childInst, nil)
	m.instRepo.On("GetByID", mock.Anything, "parent1").Return(parentInst, nil)
	m.iavRepo.On("GetCurrentValues", mock.Anything, "child1").Return([]*models.InstanceAttributeValue{}, nil)
	m.etRepo.On("GetByID", mock.Anything, "et1").Return(&models.EntityType{ID: "et1", Name: "model"}, nil)

	rec := doInstanceRequest(e, http.MethodGet, "/api/data/v1/catalogs/my-catalog/model/child1", "", apimw.RoleRO)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	chain, ok := resp["parent_chain"].([]any)
	require.True(t, ok)
	require.Len(t, chain, 1)
	entry := chain[0].(map[string]any)
	assert.Equal(t, "parent", entry["instance_name"])
}

// TD-34: SetParent with empty parent_type returns 400
func TestSetParent_EmptyParentType(t *testing.T) {
	e, _ := setupInstanceServer()

	body := `{"parent_type":"","parent_instance_id":"p1"}`
	rec := doInstanceRequest(e, http.MethodPut, "/api/data/v1/catalogs/my-catalog/model/i1/parent", body, apimw.RoleRW)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "parent_type is required")
}

// === TD-22: System Attributes — Instance DTO Injection ===

// T-18.01: instanceDetailToDTO prepends Name system attr
func TestT18_01_SystemAttr_Name(t *testing.T) {
	e, m := setupInstanceServer()
	m.mockPinResolution()

	m.instRepo.On("GetByID", mock.Anything, "i1").Return(&models.EntityInstance{
		ID: "i1", EntityTypeID: "et1", CatalogID: "cat1", Name: "my-inst", Description: "desc", Version: 1,
	}, nil)
	m.iavRepo.On("GetCurrentValues", mock.Anything, "i1").Return([]*models.InstanceAttributeValue{
		{AttributeID: "a1", ValueString: "host-a"},
	}, nil)

	rec := doInstanceRequest(e, http.MethodGet, "/api/data/v1/catalogs/my-catalog/model/i1", "", apimw.RoleRO)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	attrs := resp["attributes"].([]any)
	require.True(t, len(attrs) >= 2, "expected at least 2 attrs (system), got %d", len(attrs))
	nameAttr := attrs[0].(map[string]any)
	assert.Equal(t, "name", nameAttr["name"])
	assert.Equal(t, "string", nameAttr["type"])
	assert.Equal(t, true, nameAttr["system"])
	assert.Equal(t, true, nameAttr["required"])
	assert.Equal(t, "my-inst", nameAttr["value"])
}

// T-18.02: instanceDetailToDTO prepends Description system attr
func TestT18_02_SystemAttr_Description(t *testing.T) {
	e, m := setupInstanceServer()
	m.mockPinResolution()

	m.instRepo.On("GetByID", mock.Anything, "i1").Return(&models.EntityInstance{
		ID: "i1", EntityTypeID: "et1", CatalogID: "cat1", Name: "my-inst", Description: "my-desc", Version: 1,
	}, nil)
	m.iavRepo.On("GetCurrentValues", mock.Anything, "i1").Return([]*models.InstanceAttributeValue{}, nil)

	rec := doInstanceRequest(e, http.MethodGet, "/api/data/v1/catalogs/my-catalog/model/i1", "", apimw.RoleRO)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	attrs := resp["attributes"].([]any)
	require.True(t, len(attrs) >= 2)
	descAttr := attrs[1].(map[string]any)
	assert.Equal(t, "description", descAttr["name"])
	assert.Equal(t, "string", descAttr["type"])
	assert.Equal(t, true, descAttr["system"])
	assert.Equal(t, false, descAttr["required"])
	assert.Equal(t, "my-desc", descAttr["value"])
}

// T-18.04: Custom attributes follow system attrs and have system=false
func TestT18_04_CustomAttrsAfterSystem(t *testing.T) {
	e, m := setupInstanceServer()
	m.mockPinResolution()

	m.instRepo.On("GetByID", mock.Anything, "i1").Return(&models.EntityInstance{
		ID: "i1", EntityTypeID: "et1", CatalogID: "cat1", Name: "inst", Version: 1,
	}, nil)
	m.iavRepo.On("GetCurrentValues", mock.Anything, "i1").Return([]*models.InstanceAttributeValue{
		{AttributeID: "a1", ValueString: "host-val"},
	}, nil)

	rec := doInstanceRequest(e, http.MethodGet, "/api/data/v1/catalogs/my-catalog/model/i1", "", apimw.RoleRO)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	attrs := resp["attributes"].([]any)
	require.Len(t, attrs, 3) // 2 system + 1 custom
	customAttr := attrs[2].(map[string]any)
	assert.Equal(t, "hostname", customAttr["name"])
	assert.Equal(t, false, customAttr["system"])
}

// T-18.05: Instance with zero custom attrs still has 2 system attrs
func TestT18_05_ZeroCustomAttrsHasSystemAttrs(t *testing.T) {
	e, m := setupInstanceServer()
	// Use a pin resolution with no custom attrs
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
	m.attrRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Attribute{}, nil)

	m.instRepo.On("GetByID", mock.Anything, "i1").Return(&models.EntityInstance{
		ID: "i1", EntityTypeID: "et1", CatalogID: "cat1", Name: "inst", Version: 1,
	}, nil)
	m.iavRepo.On("GetCurrentValues", mock.Anything, "i1").Return([]*models.InstanceAttributeValue{}, nil)

	rec := doInstanceRequest(e, http.MethodGet, "/api/data/v1/catalogs/my-catalog/model/i1", "", apimw.RoleRO)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	attrs := resp["attributes"].([]any)
	assert.Len(t, attrs, 2) // only system attrs
}

// S1: Custom attrs include Required flag from schema
func TestSystemAttrs_CustomAttrRequiredFlag(t *testing.T) {
	e, m := setupInstanceServer()
	// Use a pin with a required attribute
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
		{ID: "a1", Name: "hostname", Type: models.AttributeTypeString, Required: true},
	}, nil)

	m.instRepo.On("GetByID", mock.Anything, "i1").Return(&models.EntityInstance{
		ID: "i1", EntityTypeID: "et1", CatalogID: "cat1", Name: "inst", Version: 1,
	}, nil)
	m.iavRepo.On("GetCurrentValues", mock.Anything, "i1").Return([]*models.InstanceAttributeValue{
		{AttributeID: "a1", ValueString: "host-val"},
	}, nil)

	rec := doInstanceRequest(e, http.MethodGet, "/api/data/v1/catalogs/my-catalog/model/i1", "", apimw.RoleRO)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	attrs := resp["attributes"].([]any)
	// Third attr (hostname) should have required=true
	customAttr := attrs[2].(map[string]any)
	assert.Equal(t, "hostname", customAttr["name"])
	assert.Equal(t, true, customAttr["required"])
}

// T-18.06: System attrs injected in list instances response
func TestT18_06_SystemAttrsInListResponse(t *testing.T) {
	e, m := setupInstanceServer()
	m.mockPinResolution()

	m.instRepo.On("List", mock.Anything, "et1", "cat1", mock.AnythingOfType("models.ListParams")).Return([]*models.EntityInstance{
		{ID: "i1", EntityTypeID: "et1", CatalogID: "cat1", Name: "inst-a", Version: 1},
	}, 1, nil)
	m.iavRepo.On("GetCurrentValues", mock.Anything, "i1").Return([]*models.InstanceAttributeValue{
		{AttributeID: "a1", ValueString: "host-a"},
	}, nil)

	rec := doInstanceRequest(e, http.MethodGet, "/api/data/v1/catalogs/my-catalog/model", "", apimw.RoleRO)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	items := resp["items"].([]any)
	require.Len(t, items, 1)
	item := items[0].(map[string]any)
	attrs := item["attributes"].([]any)
	require.Len(t, attrs, 3) // 2 system + 1 custom
	assert.Equal(t, "name", attrs[0].(map[string]any)["name"])
	assert.Equal(t, "inst-a", attrs[0].(map[string]any)["value"])
}
