package meta_test

import (
	"net/http"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	apimeta "github.com/project-catalyst/pc-asset-hub/internal/api/meta"
	apimw "github.com/project-catalyst/pc-asset-hub/internal/api/middleware"
	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository/mocks"
	svcmeta "github.com/project-catalyst/pc-asset-hub/internal/service/meta"
)

func setupAttrServer(attrRepo *mocks.MockAttributeRepo, etvRepo *mocks.MockEntityTypeVersionRepo, assocRepo *mocks.MockAssociationRepo, enumRepo *mocks.MockEnumRepo) *echo.Echo {
	e := echo.New()
	svc := svcmeta.NewAttributeService(attrRepo, etvRepo, nil, assocRepo, enumRepo)
	handler := apimeta.NewAttributeHandler(svc)

	g := e.Group("/api/meta/v1")
	rbac := &apimw.HeaderRBACProvider{}
	g.Use(apimw.RBACMiddleware(rbac))
	requireAdmin := apimw.RequireRole(apimw.RoleAdmin)
	apimeta.RegisterAttributeRoutes(g, handler, requireAdmin)

	return e
}

// T-C.01: List attributes for entity type
func TestTC01_ListAttributes(t *testing.T) {
	attrRepo := new(mocks.MockAttributeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	e := setupAttrServer(attrRepo, etvRepo, nil, nil)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{
		{ID: "a1", Name: "hostname", Type: models.AttributeTypeString, Ordinal: 0},
		{ID: "a2", Name: "cpu_count", Type: models.AttributeTypeNumber, Ordinal: 1},
	}, nil)

	rec := doRequest(e, http.MethodGet, "/api/meta/v1/entity-types/et1/attributes", "", apimw.RoleRO)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"hostname"`)
	assert.Contains(t, rec.Body.String(), `"cpu_count"`)
}

// T-C.02: Add string attribute
func TestTC02_AddStringAttribute(t *testing.T) {
	attrRepo := new(mocks.MockAttributeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	e := setupAttrServer(attrRepo, etvRepo, assocRepo, nil)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/et1/attributes",
		`{"name":"hostname","type":"string","description":"Host name"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Contains(t, rec.Body.String(), `"version":2`)
}

// T-C.03: Add enum attribute with valid enum_id
func TestTC03_AddEnumAttribute(t *testing.T) {
	attrRepo := new(mocks.MockAttributeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	enumRepo := new(mocks.MockEnumRepo)
	e := setupAttrServer(attrRepo, etvRepo, assocRepo, enumRepo)

	enumRepo.On("GetByID", mock.Anything, "enum1").Return(&models.Enum{ID: "enum1", Name: "Status"}, nil)
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/et1/attributes",
		`{"name":"status","type":"enum","enum_id":"enum1"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

// T-C.04: Add attribute missing name → 400
func TestTC04_AddAttributeMissingName(t *testing.T) {
	e := setupAttrServer(nil, nil, nil, nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/et1/attributes",
		`{"type":"string"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// T-C.05: Add duplicate attribute name → 409
func TestTC05_AddDuplicateAttribute(t *testing.T) {
	attrRepo := new(mocks.MockAttributeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	e := setupAttrServer(attrRepo, etvRepo, nil, nil)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{
		{ID: "a1", Name: "hostname", Type: models.AttributeTypeString},
	}, nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/et1/attributes",
		`{"name":"hostname","type":"string"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusConflict, rec.Code)
}

// T-C.06: Remove attribute by name
func TestTC06_RemoveAttribute(t *testing.T) {
	attrRepo := new(mocks.MockAttributeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	e := setupAttrServer(attrRepo, etvRepo, assocRepo, nil)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Attribute{
		{ID: "a1-new", Name: "hostname", Type: models.AttributeTypeString},
	}, nil)
	attrRepo.On("Delete", mock.Anything, "a1-new").Return(nil)

	rec := doRequest(e, http.MethodDelete, "/api/meta/v1/entity-types/et1/attributes/hostname", "", apimw.RoleAdmin)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

// T-C.07: Reorder attributes
func TestTC07_ReorderAttributes(t *testing.T) {
	attrRepo := new(mocks.MockAttributeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	e := setupAttrServer(attrRepo, etvRepo, nil, nil)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}, nil)
	attrRepo.On("Reorder", mock.Anything, "v1", []string{"a2", "a1"}).Return(nil)

	rec := doRequest(e, http.MethodPut, "/api/meta/v1/entity-types/et1/attributes/reorder",
		`{"ordered_ids":["a2","a1"]}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// T-C.08: Add attribute as RO → 403
func TestTC08_AddAttributeAsRO(t *testing.T) {
	e := setupAttrServer(nil, nil, nil, nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/et1/attributes",
		`{"name":"hostname","type":"string"}`, apimw.RoleRO)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// T-C.04b: Add attribute missing type → 400
func TestTC04b_AddAttributeMissingType(t *testing.T) {
	e := setupAttrServer(nil, nil, nil, nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/et1/attributes",
		`{"name":"hostname"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// T-C.06b: Remove nonexistent attribute → 404
func TestTC06b_RemoveNonexistentAttribute(t *testing.T) {
	attrRepo := new(mocks.MockAttributeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	e := setupAttrServer(attrRepo, etvRepo, assocRepo, nil)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Attribute{}, nil)

	rec := doRequest(e, http.MethodDelete, "/api/meta/v1/entity-types/et1/attributes/nonexistent", "", apimw.RoleAdmin)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// T-C.01b: List attributes for nonexistent entity type → 404
func TestTC01b_ListAttributesNotFound(t *testing.T) {
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	e := setupAttrServer(nil, etvRepo, nil, nil)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "bad").Return(nil, domainerrors.NewNotFound("EntityTypeVersion", "bad"))

	rec := doRequest(e, http.MethodGet, "/api/meta/v1/entity-types/bad/attributes", "", apimw.RoleRO)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// T-C.54: RW cannot add attribute → 403
func TestTC54_RWCannotAddAttribute(t *testing.T) {
	e := setupAttrServer(nil, nil, nil, nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/et1/attributes",
		`{"name":"hostname","type":"string"}`, apimw.RoleRW)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// T-C.56: SuperAdmin can add attribute → 201
func TestTC56_SuperAdminCanAddAttribute(t *testing.T) {
	attrRepo := new(mocks.MockAttributeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	e := setupAttrServer(attrRepo, etvRepo, assocRepo, nil)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/et1/attributes",
		`{"name":"hostname","type":"string"}`, apimw.RoleSuperAdmin)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

// T-C.57: RO cannot remove attribute → 403
func TestTC57_ROCannotRemoveAttribute(t *testing.T) {
	e := setupAttrServer(nil, nil, nil, nil)

	rec := doRequest(e, http.MethodDelete, "/api/meta/v1/entity-types/et1/attributes/hostname", "", apimw.RoleRO)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// T-C.58: RO cannot reorder attributes → 403
func TestTC58_ROCannotReorderAttributes(t *testing.T) {
	e := setupAttrServer(nil, nil, nil, nil)

	rec := doRequest(e, http.MethodPut, "/api/meta/v1/entity-types/et1/attributes/reorder",
		`{"ordered_ids":["a2","a1"]}`, apimw.RoleRO)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// === EditAttribute Handler Tests (T-E.16 through T-E.18) ===

// T-E.16: PUT /attributes/:name with valid edit → 200
func TestTE16_EditAttributeValid(t *testing.T) {
	attrRepo := new(mocks.MockAttributeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	e := setupAttrServer(attrRepo, etvRepo, assocRepo, nil)

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{
		{ID: "a1", Name: "hostname", Description: "old", Type: models.AttributeTypeString, Ordinal: 0},
	}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("ListByVersion", mock.Anything, mock.MatchedBy(func(id string) bool { return id != "v1" })).Return([]*models.Attribute{
		{ID: "a1-copy", Name: "hostname", Description: "old", Type: models.AttributeTypeString, Ordinal: 0},
	}, nil)
	attrRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	rec := doRequest(e, http.MethodPut, "/api/meta/v1/entity-types/et1/attributes/hostname",
		`{"name":"host","description":"updated"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"version":2`)
}

// T-E.17: PUT /attributes/:name as RO → 403
func TestTE17_EditAttributeAsRO(t *testing.T) {
	e := setupAttrServer(nil, nil, nil, nil)

	rec := doRequest(e, http.MethodPut, "/api/meta/v1/entity-types/et1/attributes/hostname",
		`{"name":"new"}`, apimw.RoleRO)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// T-E.18: PUT /attributes/:name nonexistent → 404
func TestTE18_EditAttributeNotFound(t *testing.T) {
	attrRepo := new(mocks.MockAttributeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	e := setupAttrServer(attrRepo, etvRepo, nil, nil)

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)

	rec := doRequest(e, http.MethodPut, "/api/meta/v1/entity-types/et1/attributes/nonexistent",
		`{"name":"new"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// === CopyAttributes Handler Tests (T-E.13 through T-E.15) ===

// T-E.13: POST /attributes/copy with valid request → 200
func TestTE13_CopyAttributesValid(t *testing.T) {
	attrRepo := new(mocks.MockAttributeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	e := setupAttrServer(attrRepo, etvRepo, assocRepo, nil)

	srcV1 := &models.EntityTypeVersion{ID: "src-v1", EntityTypeID: "src-et", Version: 1}
	tgtV1 := &models.EntityTypeVersion{ID: "tgt-v1", EntityTypeID: "tgt-et", Version: 1}
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "src-et", 1).Return(srcV1, nil)
	etvRepo.On("GetLatestByEntityType", mock.Anything, "tgt-et").Return(tgtV1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "src-v1").Return([]*models.Attribute{
		{Name: "attr1", Type: models.AttributeTypeString},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "tgt-v1").Return([]*models.Attribute{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/tgt-et/attributes/copy",
		`{"source_entity_type_id":"src-et","source_version":1,"attribute_names":["attr1"]}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"version":2`)
}

// T-E.14: POST /attributes/copy as RO → 403
func TestTE14_CopyAttributesAsRO(t *testing.T) {
	e := setupAttrServer(nil, nil, nil, nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/tgt-et/attributes/copy",
		`{"source_entity_type_id":"src-et","source_version":1,"attribute_names":["attr1"]}`, apimw.RoleRO)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// T-E.15: POST /attributes/copy with name conflict → 409
func TestTE15_CopyAttributesConflict(t *testing.T) {
	attrRepo := new(mocks.MockAttributeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	e := setupAttrServer(attrRepo, etvRepo, nil, nil)

	srcV1 := &models.EntityTypeVersion{ID: "src-v1", EntityTypeID: "src-et", Version: 1}
	tgtV1 := &models.EntityTypeVersion{ID: "tgt-v1", EntityTypeID: "tgt-et", Version: 1}
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "src-et", 1).Return(srcV1, nil)
	etvRepo.On("GetLatestByEntityType", mock.Anything, "tgt-et").Return(tgtV1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "src-v1").Return([]*models.Attribute{
		{Name: "conflict", Type: models.AttributeTypeString},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "tgt-v1").Return([]*models.Attribute{
		{Name: "conflict", Type: models.AttributeTypeString},
	}, nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/tgt-et/attributes/copy",
		`{"source_entity_type_id":"src-et","source_version":1,"attribute_names":["conflict"]}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusConflict, rec.Code)
}
