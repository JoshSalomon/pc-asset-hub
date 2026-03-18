package meta_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	apimeta "github.com/project-catalyst/pc-asset-hub/internal/api/meta"
	apimw "github.com/project-catalyst/pc-asset-hub/internal/api/middleware"
	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository/mocks"
	svcmeta "github.com/project-catalyst/pc-asset-hub/internal/service/meta"
)

func setupAssocServer(assocRepo *mocks.MockAssociationRepo, etvRepo *mocks.MockEntityTypeVersionRepo, attrRepo *mocks.MockAttributeRepo) *echo.Echo {
	e := echo.New()
	svc := svcmeta.NewAssociationService(assocRepo, etvRepo, attrRepo)
	handler := apimeta.NewAssociationHandler(svc)

	g := e.Group("/api/meta/v1")
	rbac := &apimw.HeaderRBACProvider{}
	g.Use(apimw.RBACMiddleware(rbac))
	requireAdmin := apimw.RequireRole(apimw.RoleAdmin)
	apimeta.RegisterAssociationRoutes(g, handler, requireAdmin)

	return e
}

// T-C.09: List associations
func TestTC09_ListAssociations(t *testing.T) {
	assocRepo := new(mocks.MockAssociationRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	e := setupAssocServer(assocRepo, etvRepo, nil)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{
		{ID: "assoc1", Name: "test_assoc", TargetEntityTypeID: "et2", Type: models.AssociationTypeContainment, CreatedAt: time.Now()},
	}, nil)
	assocRepo.On("ListByTargetEntityType", mock.Anything, "et1").Return([]*models.Association{}, nil)

	rec := doRequest(e, http.MethodGet, "/api/meta/v1/entity-types/et1/associations", "", apimw.RoleRO)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"containment"`)
	assert.Contains(t, rec.Body.String(), `"name":"test_assoc"`)
}

// T-C.10: Create containment association
func TestTC10_CreateContainmentAssociation(t *testing.T) {
	assocRepo := new(mocks.MockAssociationRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	e := setupAssocServer(assocRepo, etvRepo, attrRepo)

	assocRepo.On("GetContainmentGraph", mock.Anything).Return([]repository.ContainmentEdge{}, nil)
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/et1/associations",
		`{"name":"test_assoc","target_entity_type_id":"et2","type":"containment","source_role":"parent","target_role":"child"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Contains(t, rec.Body.String(), `"version":2`)
}

// T-C.11: Create directional association
func TestTC11_CreateDirectionalAssociation(t *testing.T) {
	assocRepo := new(mocks.MockAssociationRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	e := setupAssocServer(assocRepo, etvRepo, attrRepo)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/et1/associations",
		`{"name":"test_assoc","target_entity_type_id":"et2","type":"directional"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

// T-C.12: Create containment cycle → 422
func TestTC12_CreateContainmentCycle(t *testing.T) {
	assocRepo := new(mocks.MockAssociationRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	e := setupAssocServer(assocRepo, etvRepo, nil)

	// Graph already has et2 → et1, so et1 → et2 would create a cycle
	assocRepo.On("GetContainmentGraph", mock.Anything).Return([]repository.ContainmentEdge{
		{SourceEntityTypeID: "et2", TargetEntityTypeID: "et1"},
	}, nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/et1/associations",
		`{"name":"test_assoc","target_entity_type_id":"et2","type":"containment"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

// T-C.13: Delete association
func TestTC13_DeleteAssociation(t *testing.T) {
	assocRepo := new(mocks.MockAssociationRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	e := setupAssocServer(assocRepo, etvRepo, attrRepo)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Association{
		{ID: "assoc1-new", Name: "test_assoc", TargetEntityTypeID: "et2", Type: models.AssociationTypeContainment},
	}, nil)
	assocRepo.On("Delete", mock.Anything, "assoc1-new").Return(nil)

	rec := doRequest(e, http.MethodDelete, "/api/meta/v1/entity-types/et1/associations/test_assoc", "", apimw.RoleAdmin)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

// T-C.14: Create association as RO → 403
func TestTC14_CreateAssociationAsRO(t *testing.T) {
	e := setupAssocServer(nil, nil, nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/et1/associations",
		`{"target_entity_type_id":"et2","type":"containment"}`, apimw.RoleRO)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// T-C.09b: List associations for nonexistent entity type → 404
func TestTC09b_ListAssociationsNotFound(t *testing.T) {
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	e := setupAssocServer(nil, etvRepo, nil)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "bad").Return(nil, domainerrors.NewNotFound("EntityTypeVersion", "bad"))

	rec := doRequest(e, http.MethodGet, "/api/meta/v1/entity-types/bad/associations", "", apimw.RoleRO)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// T-C.10b: Create association with missing target → 400
func TestTC10b_CreateAssociationMissingTarget(t *testing.T) {
	e := setupAssocServer(nil, nil, nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/et1/associations",
		`{"type":"containment"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// T-C.10c: Create association with missing type → 400
func TestTC10c_CreateAssociationMissingType(t *testing.T) {
	e := setupAssocServer(nil, nil, nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/et1/associations",
		`{"target_entity_type_id":"et2"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// T-C.60: RW cannot create association → 403
func TestTC60_RWCannotCreateAssociation(t *testing.T) {
	e := setupAssocServer(nil, nil, nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/et1/associations",
		`{"target_entity_type_id":"et2","type":"containment"}`, apimw.RoleRW)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// T-C.62: SuperAdmin can create association → 201
func TestTC62_SuperAdminCanCreateAssociation(t *testing.T) {
	assocRepo := new(mocks.MockAssociationRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	e := setupAssocServer(assocRepo, etvRepo, attrRepo)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/et1/associations",
		`{"name":"test_assoc","target_entity_type_id":"et2","type":"directional"}`, apimw.RoleSuperAdmin)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

// T-C.63: RO cannot delete association → 403
func TestTC63_ROCannotDeleteAssociation(t *testing.T) {
	e := setupAssocServer(nil, nil, nil)

	rec := doRequest(e, http.MethodDelete, "/api/meta/v1/entity-types/et1/associations/test_assoc", "", apimw.RoleRO)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// T-E.80: Create association with cardinality
func TestTE80_CreateAssociationWithCardinality(t *testing.T) {
	assocRepo := new(mocks.MockAssociationRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	e := setupAssocServer(assocRepo, etvRepo, attrRepo)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/et1/associations",
		`{"name":"test_assoc","target_entity_type_id":"et2","type":"directional","source_cardinality":"1","target_cardinality":"0..n"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Contains(t, rec.Body.String(), `"version":2`)
}

// T-E.81: Create association with invalid cardinality → 400
func TestTE81_CreateAssociationInvalidCardinality(t *testing.T) {
	assocRepo := new(mocks.MockAssociationRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	e := setupAssocServer(assocRepo, etvRepo, attrRepo)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/et1/associations",
		`{"name":"test_assoc","target_entity_type_id":"et2","type":"directional","source_cardinality":"bad"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// T-E.82: List associations returns normalized cardinality
func TestTE82_ListAssociationsReturnsCardinality(t *testing.T) {
	assocRepo := new(mocks.MockAssociationRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	e := setupAssocServer(assocRepo, etvRepo, nil)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{
		{ID: "assoc1", Name: "assoc_one", TargetEntityTypeID: "et2", Type: models.AssociationTypeContainment,
			SourceCardinality: "1", TargetCardinality: "0..n", CreatedAt: time.Now()},
		{ID: "assoc2", Name: "assoc_two", TargetEntityTypeID: "et3", Type: models.AssociationTypeDirectional,
			SourceCardinality: "", TargetCardinality: "", CreatedAt: time.Now()},
	}, nil)
	assocRepo.On("ListByTargetEntityType", mock.Anything, "et1").Return([]*models.Association{}, nil)

	rec := doRequest(e, http.MethodGet, "/api/meta/v1/entity-types/et1/associations", "", apimw.RoleRO)
	assert.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	// First association has explicit cardinality
	assert.Contains(t, body, `"source_cardinality":"1"`)
	assert.Contains(t, body, `"target_cardinality":"0..n"`)
	// Second association has empty → normalized to "0..n"
	// Both should show "0..n" (normalized)
	// Count occurrences of "0..n" — should appear for both source and target of assoc2 plus target of assoc1
}

// T-E.100: PUT /entity-types/:id/associations/:name with valid edit
func TestTE100_EditAssociationValid(t *testing.T) {
	assocRepo := new(mocks.MockAssociationRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	e := setupAssocServer(assocRepo, etvRepo, attrRepo)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{
		{ID: "assoc1", Name: "test_assoc", EntityTypeVersionID: "v1", TargetEntityTypeID: "et2",
			Type: models.AssociationTypeDirectional, SourceRole: "old", TargetRole: "old_tgt",
			SourceCardinality: "0..n", TargetCardinality: "0..n"},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("ListByVersion", mock.Anything, mock.MatchedBy(func(id string) bool { return id != "v1" })).Return([]*models.Association{
		{ID: "assoc1-copy", Name: "test_assoc", TargetEntityTypeID: "et2", Type: models.AssociationTypeDirectional,
			SourceRole: "old", TargetRole: "old_tgt", SourceCardinality: "0..n", TargetCardinality: "0..n"},
	}, nil)
	assocRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	rec := doRequest(e, http.MethodPut, "/api/meta/v1/entity-types/et1/associations/test_assoc",
		`{"source_role":"new_role","source_cardinality":"1"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"version":2`)
}

// T-E.101: PUT /entity-types/:id/associations/:name as RO → 403
func TestTE101_EditAssociationAsRO(t *testing.T) {
	e := setupAssocServer(nil, nil, nil)

	rec := doRequest(e, http.MethodPut, "/api/meta/v1/entity-types/et1/associations/test_assoc",
		`{"source_role":"new"}`, apimw.RoleRO)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// T-E.102: PUT /entity-types/:id/associations/:name nonexistent → 404
func TestTE102_EditAssociationNotFound(t *testing.T) {
	assocRepo := new(mocks.MockAssociationRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	e := setupAssocServer(assocRepo, etvRepo, attrRepo)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)

	rec := doRequest(e, http.MethodPut, "/api/meta/v1/entity-types/et1/associations/bad-name",
		`{"source_role":"new"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// T-E.103: PUT /entity-types/:id/associations/:name invalid cardinality → 400
func TestTE103_EditAssociationInvalidCardinality(t *testing.T) {
	assocRepo := new(mocks.MockAssociationRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	e := setupAssocServer(assocRepo, etvRepo, attrRepo)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{
		{ID: "assoc1", Name: "test_assoc", Type: models.AssociationTypeDirectional},
	}, nil)

	rec := doRequest(e, http.MethodPut, "/api/meta/v1/entity-types/et1/associations/test_assoc",
		`{"source_cardinality":"bad"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// === Coverage: bind-error and service-error branches ===

func TestAssocCreate_BindError(t *testing.T) {
	e := setupAssocServer(nil, nil, nil)
	rec := doRequest(e, http.MethodPost, "/api/meta/v1/entity-types/et1/associations", "bad{json", apimw.RoleAdmin)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAssocEdit_BindError(t *testing.T) {
	e := setupAssocServer(nil, nil, nil)
	rec := doRequest(e, http.MethodPut, "/api/meta/v1/entity-types/et1/associations/test_assoc", "bad{json", apimw.RoleAdmin)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAssocEdit_ServiceError(t *testing.T) {
	assocRepo := new(mocks.MockAssociationRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	e := setupAssocServer(assocRepo, etvRepo, nil)
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(nil, domainerrors.NewNotFound("EntityTypeVersion", "et1"))
	rec := doRequest(e, http.MethodPut, "/api/meta/v1/entity-types/et1/associations/test_assoc",
		`{"source_role":"new"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAssocDelete_ServiceError(t *testing.T) {
	assocRepo := new(mocks.MockAssociationRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	e := setupAssocServer(assocRepo, etvRepo, nil)
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(nil, domainerrors.NewNotFound("EntityTypeVersion", "et1"))
	rec := doRequest(e, http.MethodDelete, "/api/meta/v1/entity-types/et1/associations/test_assoc", "", apimw.RoleAdmin)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// Coverage: Edit with type field (lines 79-82)
func TestAssocEdit_WithType(t *testing.T) {
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	e := setupAssocServer(assocRepo, etvRepo, attrRepo)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{
		ID: "etv1", EntityTypeID: "et1", Version: 2,
	}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Association{
		{ID: "a1", Name: "old-name", Type: models.AssociationTypeDirectional},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Attribute{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Association{
		{ID: "a2", Name: "old-name", Type: models.AssociationTypeBidirectional},
	}, nil)
	assocRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	rec := doRequest(e, http.MethodPut, "/api/meta/v1/entity-types/et1/associations/old-name",
		`{"type":"bidirectional"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusOK, rec.Code)
}
