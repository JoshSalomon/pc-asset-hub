package meta_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/project-catalyst/pc-asset-hub/internal/api/dto"
	apimeta "github.com/project-catalyst/pc-asset-hub/internal/api/meta"
	apimw "github.com/project-catalyst/pc-asset-hub/internal/api/middleware"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository"
	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository/mocks"
	svcmeta "github.com/project-catalyst/pc-asset-hub/internal/service/meta"
)

func setupAttrServer(attrRepo *mocks.MockAttributeRepo, etvRepo *mocks.MockEntityTypeVersionRepo, assocRepo *mocks.MockAssociationRepo, tdvRepo *mocks.MockTypeDefinitionVersionRepo) *echo.Echo {
	return setupAttrServerWithTypeRepos(attrRepo, etvRepo, assocRepo, tdvRepo, nil, nil)
}

func setupAttrServerWithTypeRepos(attrRepo *mocks.MockAttributeRepo, etvRepo *mocks.MockEntityTypeVersionRepo, assocRepo *mocks.MockAssociationRepo, tdvRepo *mocks.MockTypeDefinitionVersionRepo, resolveTdvRepo *mocks.MockTypeDefinitionVersionRepo, tdRepo *mocks.MockTypeDefinitionRepo) *echo.Echo {
	e := echo.New()
	svc := svcmeta.NewAttributeService(attrRepo, etvRepo, nil, assocRepo, tdvRepo)
	// Avoid Go typed-nil interface trap: only pass non-nil repos
	var resolveRepo repository.TypeDefinitionVersionRepository
	var typeRepo repository.TypeDefinitionRepository
	if resolveTdvRepo != nil {
		resolveRepo = resolveTdvRepo
	}
	if tdRepo != nil {
		typeRepo = tdRepo
	}
	handler := apimeta.NewAttributeHandler(svc, resolveRepo, typeRepo)

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

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{
		{ID: "a1", Name: "hostname", TypeDefinitionVersionID: "tdv-string", Ordinal: 0},
		{ID: "a2", Name: "cpu_count", TypeDefinitionVersionID: "tdv-number", Ordinal: 1},
	}, nil)

	// Set up with type definition repos for type info resolution
	tdvRepoResolve := new(mocks.MockTypeDefinitionVersionRepo)
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	tdvRepoResolve.On("GetByIDs", mock.Anything, mock.Anything).Return([]*models.TypeDefinitionVersion{
		{ID: "tdv-string", TypeDefinitionID: "td-string", VersionNumber: 1},
		{ID: "tdv-number", TypeDefinitionID: "td-number", VersionNumber: 1},
	}, nil)
	tdRepo.On("GetByID", mock.Anything, "td-string").Return(&models.TypeDefinition{ID: "td-string", Name: "string", BaseType: models.BaseTypeString}, nil)
	tdRepo.On("GetByID", mock.Anything, "td-number").Return(&models.TypeDefinition{ID: "td-number", Name: "number", BaseType: models.BaseTypeNumber}, nil)

	e2 := setupAttrServerWithTypeRepos(attrRepo, etvRepo, nil, nil, tdvRepoResolve, tdRepo)
	rec := doRequest(e2, http.MethodGet, "/api/meta/v1/entity-types/et1/attributes", "", apimw.RoleRO)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"hostname"`)
	assert.Contains(t, rec.Body.String(), `"cpu_count"`)
	// Type info must be resolved — type_name and base_type must be present
	assert.Contains(t, rec.Body.String(), `"type_name":"string"`)
	assert.Contains(t, rec.Body.String(), `"base_type":"string"`)
	assert.Contains(t, rec.Body.String(), `"base_type":"number"`)
}

// T-C.02: Add string attribute
func TestTC02_AddStringAttribute(t *testing.T) {
	attrRepo := new(mocks.MockAttributeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	e := setupAttrServer(attrRepo, etvRepo, assocRepo, tdvRepo)

	tdvRepo.On("GetByID", mock.Anything, "tdv-string").Return(&models.TypeDefinitionVersion{ID: "tdv-string"}, nil)
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/et1/attributes",
		`{"name":"hostname","type_definition_version_id":"tdv-string","description":"Host name"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Contains(t, rec.Body.String(), `"version":2`)
}

// T-C.03: Add attribute with valid type_definition_version_id
func TestTC03_AddAttributeWithTDV(t *testing.T) {
	attrRepo := new(mocks.MockAttributeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	e := setupAttrServer(attrRepo, etvRepo, assocRepo, tdvRepo)

	tdvRepo.On("GetByID", mock.Anything, "tdv-enum1").Return(&models.TypeDefinitionVersion{ID: "tdv-enum1", TypeDefinitionID: "td-status"}, nil)
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/et1/attributes",
		`{"name":"status","type_definition_version_id":"tdv-enum1"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

// T-C.04: Add attribute missing name → 400
func TestTC04_AddAttributeMissingName(t *testing.T) {
	e := setupAttrServer(nil, nil, nil, nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/et1/attributes",
		`{"type_definition_version_id":"tdv-string"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// T-C.05: Add duplicate attribute name → 409
func TestTC05_AddDuplicateAttribute(t *testing.T) {
	attrRepo := new(mocks.MockAttributeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	e := setupAttrServer(attrRepo, etvRepo, nil, tdvRepo)

	tdvRepo.On("GetByID", mock.Anything, "tdv-string").Return(&models.TypeDefinitionVersion{ID: "tdv-string"}, nil)
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{
		{ID: "a1", Name: "hostname", TypeDefinitionVersionID: "tdv-string"},
	}, nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/et1/attributes",
		`{"name":"hostname","type_definition_version_id":"tdv-string"}`, apimw.RoleAdmin)
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
		{ID: "a1-new", Name: "hostname", TypeDefinitionVersionID: "tdv-string"},
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
		`{"name":"hostname","type_definition_version_id":"tdv-string"}`, apimw.RoleRO)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// T-C.04b: Add attribute missing type_definition_version_id → 400
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
		`{"name":"hostname","type_definition_version_id":"tdv-string"}`, apimw.RoleRW)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// T-C.56: SuperAdmin can add attribute → 201
func TestTC56_SuperAdminCanAddAttribute(t *testing.T) {
	attrRepo := new(mocks.MockAttributeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	e := setupAttrServer(attrRepo, etvRepo, assocRepo, tdvRepo)

	tdvRepo.On("GetByID", mock.Anything, "tdv-string").Return(&models.TypeDefinitionVersion{ID: "tdv-string"}, nil)
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/et1/attributes",
		`{"name":"hostname","type_definition_version_id":"tdv-string"}`, apimw.RoleSuperAdmin)
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
		{ID: "a1", Name: "hostname", Description: "old", TypeDefinitionVersionID: "tdv-string", Ordinal: 0},
	}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("ListByVersion", mock.Anything, mock.MatchedBy(func(id string) bool { return id != "v1" })).Return([]*models.Attribute{
		{ID: "a1-copy", Name: "hostname", Description: "old", TypeDefinitionVersionID: "tdv-string", Ordinal: 0},
	}, nil)
	attrRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	rec := doRequest(e, http.MethodPut, "/api/meta/v1/entity-types/et1/attributes/hostname",
		`{"name":"host","description":"updated"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"version":2`)
}

// Coverage: Edit with type_definition_version_id change covers req.TypeDefinitionVersionID != nil branch
func TestEditAttribute_WithTypeChange(t *testing.T) {
	attrRepo := new(mocks.MockAttributeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	e := setupAttrServer(attrRepo, etvRepo, assocRepo, tdvRepo)

	tdvRepo.On("GetByID", mock.Anything, "tdv-number").Return(&models.TypeDefinitionVersion{ID: "tdv-number"}, nil)
	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{
		{ID: "a1", Name: "hostname", TypeDefinitionVersionID: "tdv-string", Ordinal: 0},
	}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("ListByVersion", mock.Anything, mock.MatchedBy(func(id string) bool { return id != "v1" })).Return([]*models.Attribute{
		{ID: "a1-copy", Name: "hostname", TypeDefinitionVersionID: "tdv-string", Ordinal: 0},
	}, nil)
	attrRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	rec := doRequest(e, http.MethodPut, "/api/meta/v1/entity-types/et1/attributes/hostname",
		`{"type_definition_version_id":"tdv-number"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusOK, rec.Code)
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
		{Name: "attr1", TypeDefinitionVersionID: "tdv-string"},
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
		{Name: "conflict", TypeDefinitionVersionID: "tdv-string"},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "tgt-v1").Return([]*models.Attribute{
		{Name: "conflict", TypeDefinitionVersionID: "tdv-string"},
	}, nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/tgt-et/attributes/copy",
		`{"source_entity_type_id":"src-et","source_version":1,"attribute_names":["conflict"]}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusConflict, rec.Code)
}

// === Coverage: bind-error and service-error branches ===

func TestAttrAdd_BindError(t *testing.T) {
	e := setupAttrServer(nil, nil, nil, nil)
	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/et1/attributes", "bad{json", apimw.RoleAdmin)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAttrReorder_BindError(t *testing.T) {
	e := setupAttrServer(nil, nil, nil, nil)
	rec := doRequest(e, http.MethodPut, "/api/meta/v1/entity-types/et1/attributes/reorder", "bad{json", apimw.RoleAdmin)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAttrReorder_EmptyIDs(t *testing.T) {
	e := setupAttrServer(nil, nil, nil, nil)
	rec := doRequest(e, http.MethodPut, "/api/meta/v1/entity-types/et1/attributes/reorder", `{"ordered_ids":[]}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAttrReorder_ServiceError(t *testing.T) {
	attrRepo := new(mocks.MockAttributeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	e := setupAttrServer(attrRepo, etvRepo, nil, nil)
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(nil, domainerrors.NewNotFound("EntityTypeVersion", "et1"))
	rec := doRequest(e, http.MethodPut, "/api/meta/v1/entity-types/et1/attributes/reorder",
		`{"ordered_ids":["a1","a2"]}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAttrEdit_BindError(t *testing.T) {
	e := setupAttrServer(nil, nil, nil, nil)
	rec := doRequest(e, http.MethodPut, "/api/meta/v1/entity-types/et1/attributes/hostname", "bad{json", apimw.RoleAdmin)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAttrEdit_ServiceError(t *testing.T) {
	attrRepo := new(mocks.MockAttributeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	e := setupAttrServer(attrRepo, etvRepo, nil, nil)
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(nil, domainerrors.NewNotFound("EntityTypeVersion", "et1"))
	rec := doRequest(e, http.MethodPut, "/api/meta/v1/entity-types/et1/attributes/hostname",
		`{"name":"new"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAttrCopy_BindError(t *testing.T) {
	e := setupAttrServer(nil, nil, nil, nil)
	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/et1/attributes/copy", "bad{json", apimw.RoleAdmin)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAttrCopy_EmptySourceEntityTypeID(t *testing.T) {
	e := setupAttrServer(nil, nil, nil, nil)
	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/et1/attributes/copy",
		`{"source_entity_type_id":"","source_version":1,"attribute_names":["a"]}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAttrCopy_EmptyAttributeNames(t *testing.T) {
	e := setupAttrServer(nil, nil, nil, nil)
	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/et1/attributes/copy",
		`{"source_entity_type_id":"src","source_version":1,"attribute_names":[]}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// === TD-22: System Attributes in Attribute List ===

// T-18.10: Attribute list prepends Name and Description system attrs
func TestT18_10_AttrListSystemAttrs(t *testing.T) {
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	e := setupAttrServer(attrRepo, etvRepo, nil, nil)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "v1"}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{
		{ID: "a1", Name: "hostname", TypeDefinitionVersionID: "tdv-string", Ordinal: 0},
	}, nil)

	rec := doRequest(e, http.MethodGet, "/api/meta/v1/entity-types/et1/attributes", "", apimw.RoleRO)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp dto.ListResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, 3, resp.Total) // 2 system + 1 custom
	items := resp.Items.([]any)
	nameAttr := items[0].(map[string]any)
	assert.Equal(t, "name", nameAttr["name"])
	assert.Equal(t, true, nameAttr["system"])
	assert.Equal(t, true, nameAttr["required"])
	descAttr := items[1].(map[string]any)
	assert.Equal(t, "description", descAttr["name"])
	assert.Equal(t, true, descAttr["system"])
	assert.Equal(t, false, descAttr["required"])
}

// T-18.11: System attrs have correct ordinals
func TestT18_11_AttrListSystemTypes(t *testing.T) {
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	e := setupAttrServer(attrRepo, etvRepo, nil, nil)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "v1"}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)

	rec := doRequest(e, http.MethodGet, "/api/meta/v1/entity-types/et1/attributes", "", apimw.RoleRO)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp dto.ListResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, 2, resp.Total) // only system attrs
	items := resp.Items.([]any)
	nameAttr := items[0].(map[string]any)
	assert.Equal(t, float64(-2), nameAttr["ordinal"])
	descAttr := items[1].(map[string]any)
	assert.Equal(t, float64(-1), descAttr["ordinal"])
}

// T-18.12: Create attribute with name "name" is rejected
func TestT18_12_ReservedNameRejected(t *testing.T) {
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	e := setupAttrServer(attrRepo, etvRepo, nil, nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/et1/attributes",
		`{"name":"name","type_definition_version_id":"tdv-string"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "reserved")
}

// T-18.13: Create attribute with name "description" is rejected
func TestT18_13_ReservedDescriptionRejected(t *testing.T) {
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	e := setupAttrServer(attrRepo, etvRepo, nil, nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/et1/attributes",
		`{"name":"description","type_definition_version_id":"tdv-string"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "reserved")
}

// T-18.14: Create attribute with name "Name" (uppercase) is allowed
func TestT18_14_UppercaseNameAllowed(t *testing.T) {
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	e := setupAttrServer(attrRepo, etvRepo, assocRepo, tdvRepo)

	tdvRepo.On("GetByID", mock.Anything, "tdv-string").Return(&models.TypeDefinitionVersion{ID: "tdv-string"}, nil)
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	attrRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/et1/attributes",
		`{"name":"Name","type_definition_version_id":"tdv-string"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

// T-18.15: Rename attribute to "name" is rejected
func TestT18_15_RenameToNameRejected(t *testing.T) {
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	e := setupAttrServer(attrRepo, etvRepo, nil, nil)

	rec := doRequest(e, http.MethodPut, "/api/meta/v1/entity-types/et1/attributes/hostname",
		`{"name":"name"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "reserved")
}

// T-18.16: Rename attribute to "description" is rejected
func TestT18_16_RenameToDescriptionRejected(t *testing.T) {
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	e := setupAttrServer(attrRepo, etvRepo, nil, nil)

	rec := doRequest(e, http.MethodPut, "/api/meta/v1/entity-types/et1/attributes/hostname",
		`{"name":"description"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "reserved")
}

// I3: Remove handler rejects system attribute names
func TestRemove_RejectsReservedName(t *testing.T) {
	e := setupAttrServer(nil, nil, nil, nil)

	rec := doRequest(e, http.MethodDelete, "/api/meta/v1/entity-types/et1/attributes/name", "", apimw.RoleAdmin)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "system attribute")
}

func TestRemove_RejectsReservedDescription(t *testing.T) {
	e := setupAttrServer(nil, nil, nil, nil)

	rec := doRequest(e, http.MethodDelete, "/api/meta/v1/entity-types/et1/attributes/description", "", apimw.RoleAdmin)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "system attribute")
}

// Coverage: Edit service error (lines 99-102)
func TestAttrEdit_ServiceErrorPath(t *testing.T) {
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	e := setupAttrServer(attrRepo, etvRepo, nil, nil)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(nil, domainerrors.NewNotFound("ETV", "et1"))

	rec := doRequest(e, http.MethodPut, "/api/meta/v1/entity-types/et1/attributes/hostname",
		`{"description":"new"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}
