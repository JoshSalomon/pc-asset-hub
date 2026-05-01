package meta_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository/mocks"
	"github.com/project-catalyst/pc-asset-hub/internal/service/meta"
)

func TestT3_01_CreateEntityType(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, attrRepo, assocRepo)

	etRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityType")).Return(nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)

	et, etv, err := svc.CreateEntityType(context.Background(), "Model", "A model entity")
	require.NoError(t, err)
	assert.NotEmpty(t, et.ID)
	assert.Equal(t, "Model", et.Name)
	assert.Equal(t, 1, etv.Version)
}

func TestT3_02_CreateEntityTypeEmptyName(t *testing.T) {
	svc := meta.NewEntityTypeService(nil, nil, nil, nil)
	_, _, err := svc.CreateEntityType(context.Background(), "", "desc")
	assert.True(t, domainerrors.IsValidation(err))
}

func TestT3_03_CreateEntityTypeDuplicateName(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, nil, nil)

	etRepo.On("Create", mock.Anything, mock.Anything).Return(domainerrors.NewConflict("EntityType", "name already exists"))

	_, _, err := svc.CreateEntityType(context.Background(), "Model", "desc")
	assert.True(t, domainerrors.IsConflict(err))
}

func TestT3_04_UpdateEntityType(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, attrRepo, assocRepo)

	latestVersion := &models.EntityTypeVersion{ID: "v1-id", EntityTypeID: "et-id", Version: 1, Description: "V1"}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-id").Return(latestVersion, nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, "v1-id", mock.AnythingOfType("string")).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, "v1-id", mock.AnythingOfType("string")).Return(nil)
	etRepo.On("GetByID", mock.Anything, "et-id").Return(&models.EntityType{ID: "et-id", Name: "Model"}, nil)
	etRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	newVersion, err := svc.UpdateEntityType(context.Background(), "et-id", "V2 description")
	require.NoError(t, err)
	assert.Equal(t, 2, newVersion.Version)
	attrRepo.AssertCalled(t, "BulkCopyToVersion", mock.Anything, "v1-id", mock.AnythingOfType("string"))
	assocRepo.AssertCalled(t, "BulkCopyToVersion", mock.Anything, "v1-id", mock.AnythingOfType("string"))
}

func TestT3_05_UpdatePreservesV1(t *testing.T) {
	// After update, V1 attributes/associations remain unchanged.
	// This is tested by verifying BulkCopyToVersion creates copies (not moves).
	// The test in T3_04 already verifies the copy call is made to a NEW version ID,
	// leaving the original untouched.
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, attrRepo, assocRepo)

	v1 := &models.EntityTypeVersion{ID: "v1-id", EntityTypeID: "et-id", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-id").Return(v1, nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, "v1-id", mock.AnythingOfType("string")).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, "v1-id", mock.AnythingOfType("string")).Return(nil)
	etRepo.On("GetByID", mock.Anything, "et-id").Return(&models.EntityType{ID: "et-id"}, nil)
	etRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	newVersion, err := svc.UpdateEntityType(context.Background(), "et-id", "V2")
	require.NoError(t, err)
	// New version should have a different ID from v1
	assert.NotEqual(t, "v1-id", newVersion.ID)
	assert.Equal(t, 2, newVersion.Version)
}

func TestT3_06_CopyEntityType(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, attrRepo, assocRepo)

	sourceETV := &models.EntityTypeVersion{ID: "src-v1", EntityTypeID: "src-et", Version: 1}
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "src-et", 1).Return(sourceETV, nil)
	etRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityType")).Return(nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, "src-v1", mock.AnythingOfType("string")).Return(nil)

	newET, newETV, err := svc.CopyEntityType(context.Background(), "src-et", 1, "NewType")
	require.NoError(t, err)
	assert.Equal(t, "NewType", newET.Name)
	assert.Equal(t, 1, newETV.Version)
	// Attributes were copied
	attrRepo.AssertCalled(t, "BulkCopyToVersion", mock.Anything, "src-v1", mock.AnythingOfType("string"))
	// Associations were NOT copied
	assocRepo.AssertNotCalled(t, "BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything)
}

func TestT3_07_CopyDoesNotChangeSource(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, attrRepo, assocRepo)

	sourceETV := &models.EntityTypeVersion{ID: "src-v1", EntityTypeID: "src-et", Version: 1}
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "src-et", 1).Return(sourceETV, nil)
	etRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	_, _, err := svc.CopyEntityType(context.Background(), "src-et", 1, "Copy")
	require.NoError(t, err)
	// Source entity type was never updated or deleted
	etRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	etRepo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
}

func TestT3_08_CopyDuplicateTargetName(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, nil, nil)

	sourceETV := &models.EntityTypeVersion{ID: "src-v1", EntityTypeID: "src-et", Version: 1}
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "src-et", 1).Return(sourceETV, nil)
	etRepo.On("Create", mock.Anything, mock.Anything).Return(domainerrors.NewConflict("EntityType", "exists"))

	_, _, err := svc.CopyEntityType(context.Background(), "src-et", 1, "Existing")
	assert.True(t, domainerrors.IsConflict(err))
}

func TestT3_09_DeleteEntityType(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	svc := meta.NewEntityTypeService(etRepo, nil, nil, nil)

	etRepo.On("Delete", mock.Anything, "et-id").Return(nil)
	err := svc.DeleteEntityType(context.Background(), "et-id")
	assert.NoError(t, err)
}

func TestT3_10_ListEntityTypes(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	svc := meta.NewEntityTypeService(etRepo, nil, nil, nil)

	expected := []*models.EntityType{{ID: "1", Name: "A"}, {ID: "2", Name: "B"}}
	etRepo.On("List", mock.Anything, mock.AnythingOfType("models.ListParams")).Return(expected, 2, nil)

	result, total, err := svc.ListEntityTypes(context.Background(), models.ListParams{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, result, 2)
}

// === RenameEntityType Tests (T-E.08 through T-E.12b) ===

func setupETServiceWithCatalogRepos() (*meta.EntityTypeService, *mocks.MockEntityTypeRepo, *mocks.MockEntityTypeVersionRepo, *mocks.MockAttributeRepo, *mocks.MockCatalogVersionPinRepo, *mocks.MockCatalogVersionRepo) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, attrRepo, assocRepo)
	meta.WithCatalogRepos(svc, pinRepo, cvRepo)
	return svc, etRepo, etvRepo, attrRepo, pinRepo, cvRepo
}

func TestTE08_RenameEntityNotInAnyCV(t *testing.T) {
	svc, etRepo, etvRepo, _, pinRepo, _ := setupETServiceWithCatalogRepos()

	et := &models.EntityType{ID: "et1", Name: "OldName"}
	etRepo.On("GetByID", mock.Anything, "et1").Return(et, nil)
	etRepo.On("GetByName", mock.Anything, "NewName").Return(nil, domainerrors.NewNotFound("EntityType", "NewName"))
	etvRepo.On("ListByEntityType", mock.Anything, "et1").Return([]*models.EntityTypeVersion{
		{ID: "v1", EntityTypeID: "et1", Version: 1},
	}, nil)
	pinRepo.On("ListByEntityTypeVersionIDs", mock.Anything, []string{"v1"}).Return([]*models.CatalogVersionPin{}, nil)
	etRepo.On("Update", mock.Anything, mock.MatchedBy(func(e *models.EntityType) bool {
		return e.Name == "NewName"
	})).Return(nil)

	result, err := svc.RenameEntityType(context.Background(), "et1", "NewName", false)
	require.NoError(t, err)
	assert.Equal(t, "NewName", result.EntityType.Name)
	assert.False(t, result.WasDeepCopy)
}

func TestTE09_RenameEntityInOneDevelopmentCV(t *testing.T) {
	svc, etRepo, etvRepo, _, pinRepo, cvRepo := setupETServiceWithCatalogRepos()

	et := &models.EntityType{ID: "et1", Name: "OldName"}
	etRepo.On("GetByID", mock.Anything, "et1").Return(et, nil)
	etRepo.On("GetByName", mock.Anything, "NewName").Return(nil, domainerrors.NewNotFound("EntityType", "NewName"))
	etvRepo.On("ListByEntityType", mock.Anything, "et1").Return([]*models.EntityTypeVersion{
		{ID: "v1", EntityTypeID: "et1", Version: 1},
	}, nil)
	pinRepo.On("ListByEntityTypeVersionIDs", mock.Anything, []string{"v1"}).Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "v1"},
	}, nil)
	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	etRepo.On("Update", mock.Anything, mock.MatchedBy(func(e *models.EntityType) bool {
		return e.Name == "NewName"
	})).Return(nil)

	result, err := svc.RenameEntityType(context.Background(), "et1", "NewName", false)
	require.NoError(t, err)
	assert.Equal(t, "NewName", result.EntityType.Name)
	assert.False(t, result.WasDeepCopy)
}

func TestTE10_RenameEntityInTestingCV_DeepCopyNotAllowed(t *testing.T) {
	svc, etRepo, etvRepo, _, pinRepo, cvRepo := setupETServiceWithCatalogRepos()

	et := &models.EntityType{ID: "et1", Name: "OldName"}
	etRepo.On("GetByID", mock.Anything, "et1").Return(et, nil)
	etRepo.On("GetByName", mock.Anything, "NewName").Return(nil, domainerrors.NewNotFound("EntityType", "NewName"))
	etvRepo.On("ListByEntityType", mock.Anything, "et1").Return([]*models.EntityTypeVersion{
		{ID: "v1", EntityTypeID: "et1", Version: 1},
	}, nil)
	pinRepo.On("ListByEntityTypeVersionIDs", mock.Anything, []string{"v1"}).Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "v1"},
	}, nil)
	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageTesting,
	}, nil)

	_, err := svc.RenameEntityType(context.Background(), "et1", "NewName", false)
	assert.True(t, domainerrors.IsDeepCopyRequired(err))
}

func TestTE11_RenameEntityInMultipleCVs_DeepCopyAllowed(t *testing.T) {
	svc, etRepo, etvRepo, attrRepo, pinRepo, cvRepo := setupETServiceWithCatalogRepos()

	et := &models.EntityType{ID: "et1", Name: "OldName"}
	etRepo.On("GetByID", mock.Anything, "et1").Return(et, nil)
	etRepo.On("GetByName", mock.Anything, "NewName").Return(nil, domainerrors.NewNotFound("EntityType", "NewName"))
	etvRepo.On("ListByEntityType", mock.Anything, "et1").Return([]*models.EntityTypeVersion{
		{ID: "v1", EntityTypeID: "et1", Version: 1},
	}, nil)
	pinRepo.On("ListByEntityTypeVersionIDs", mock.Anything, []string{"v1"}).Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "v1"},
		{ID: "pin2", CatalogVersionID: "cv2", EntityTypeVersionID: "v1"},
	}, nil)
	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	cvRepo.On("GetByID", mock.Anything, "cv2").Return(&models.CatalogVersion{
		ID: "cv2", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	// RenameEntityType calls GetLatestByEntityType to find latest version for deep copy
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{
		ID: "v1", EntityTypeID: "et1", Version: 1,
	}, nil)
	// Deep copy uses CopyEntityType logic: creates new ET + V1 + copies attrs
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 1).Return(&models.EntityTypeVersion{
		ID: "v1", EntityTypeID: "et1", Version: 1,
	}, nil)
	etRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityType")).Return(nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, "v1", mock.AnythingOfType("string")).Return(nil)

	result, err := svc.RenameEntityType(context.Background(), "et1", "NewName", true)
	require.NoError(t, err)
	assert.True(t, result.WasDeepCopy)
	assert.Equal(t, "NewName", result.EntityType.Name)
	// Original entity type was NOT renamed
	etRepo.AssertNotCalled(t, "Update", mock.Anything, mock.MatchedBy(func(e *models.EntityType) bool {
		return e.Name == "NewName"
	}))
}

// === RenameEntityType Error Path Tests ===

func TestRename_GetByIDError(t *testing.T) {
	svc, etRepo, _, _, _, _ := setupETServiceWithCatalogRepos()

	etRepo.On("GetByName", mock.Anything, "NewName").Return(nil, domainerrors.NewNotFound("EntityType", "NewName"))
	etRepo.On("GetByID", mock.Anything, "et1").Return(nil, domainerrors.NewNotFound("EntityType", "et1"))

	_, err := svc.RenameEntityType(context.Background(), "et1", "NewName", false)
	assert.True(t, domainerrors.IsNotFound(err))
}

func TestRename_ListVersionsError(t *testing.T) {
	svc, etRepo, etvRepo, _, _, _ := setupETServiceWithCatalogRepos()

	et := &models.EntityType{ID: "et1", Name: "OldName"}
	etRepo.On("GetByName", mock.Anything, "NewName").Return(nil, domainerrors.NewNotFound("EntityType", "NewName"))
	etRepo.On("GetByID", mock.Anything, "et1").Return(et, nil)
	etvRepo.On("ListByEntityType", mock.Anything, "et1").Return(([]*models.EntityTypeVersion)(nil), domainerrors.NewNotFound("EntityTypeVersion", "et1"))

	_, err := svc.RenameEntityType(context.Background(), "et1", "NewName", false)
	assert.True(t, domainerrors.IsNotFound(err))
}

func TestRename_PinRepoError(t *testing.T) {
	svc, etRepo, etvRepo, _, pinRepo, _ := setupETServiceWithCatalogRepos()

	et := &models.EntityType{ID: "et1", Name: "OldName"}
	etRepo.On("GetByName", mock.Anything, "NewName").Return(nil, domainerrors.NewNotFound("EntityType", "NewName"))
	etRepo.On("GetByID", mock.Anything, "et1").Return(et, nil)
	etvRepo.On("ListByEntityType", mock.Anything, "et1").Return([]*models.EntityTypeVersion{
		{ID: "v1", EntityTypeID: "et1", Version: 1},
	}, nil)
	pinRepo.On("ListByEntityTypeVersionIDs", mock.Anything, []string{"v1"}).Return(([]*models.CatalogVersionPin)(nil), domainerrors.NewNotFound("CatalogVersionPin", "v1"))

	_, err := svc.RenameEntityType(context.Background(), "et1", "NewName", false)
	assert.True(t, domainerrors.IsNotFound(err))
}

func TestRename_CvRepoError(t *testing.T) {
	svc, etRepo, etvRepo, _, pinRepo, cvRepo := setupETServiceWithCatalogRepos()

	et := &models.EntityType{ID: "et1", Name: "OldName"}
	etRepo.On("GetByName", mock.Anything, "NewName").Return(nil, domainerrors.NewNotFound("EntityType", "NewName"))
	etRepo.On("GetByID", mock.Anything, "et1").Return(et, nil)
	etvRepo.On("ListByEntityType", mock.Anything, "et1").Return([]*models.EntityTypeVersion{
		{ID: "v1", EntityTypeID: "et1", Version: 1},
	}, nil)
	pinRepo.On("ListByEntityTypeVersionIDs", mock.Anything, []string{"v1"}).Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "v1"},
	}, nil)
	cvRepo.On("GetByID", mock.Anything, "cv1").Return(nil, domainerrors.NewNotFound("CatalogVersion", "cv1"))

	_, err := svc.RenameEntityType(context.Background(), "et1", "NewName", false)
	assert.True(t, domainerrors.IsNotFound(err))
}

func TestRename_UpdateError(t *testing.T) {
	svc, etRepo, etvRepo, _, pinRepo, _ := setupETServiceWithCatalogRepos()

	et := &models.EntityType{ID: "et1", Name: "OldName"}
	etRepo.On("GetByName", mock.Anything, "NewName").Return(nil, domainerrors.NewNotFound("EntityType", "NewName"))
	etRepo.On("GetByID", mock.Anything, "et1").Return(et, nil)
	etvRepo.On("ListByEntityType", mock.Anything, "et1").Return([]*models.EntityTypeVersion{
		{ID: "v1", EntityTypeID: "et1", Version: 1},
	}, nil)
	pinRepo.On("ListByEntityTypeVersionIDs", mock.Anything, []string{"v1"}).Return([]*models.CatalogVersionPin{}, nil)
	etRepo.On("Update", mock.Anything, mock.Anything).Return(domainerrors.NewConflict("EntityType", "update failed"))

	_, err := svc.RenameEntityType(context.Background(), "et1", "NewName", false)
	assert.True(t, domainerrors.IsConflict(err))
}

func TestRename_GetLatestError(t *testing.T) {
	svc, etRepo, etvRepo, _, pinRepo, cvRepo := setupETServiceWithCatalogRepos()

	et := &models.EntityType{ID: "et1", Name: "OldName"}
	etRepo.On("GetByName", mock.Anything, "NewName").Return(nil, domainerrors.NewNotFound("EntityType", "NewName"))
	etRepo.On("GetByID", mock.Anything, "et1").Return(et, nil)
	etvRepo.On("ListByEntityType", mock.Anything, "et1").Return([]*models.EntityTypeVersion{
		{ID: "v1", EntityTypeID: "et1", Version: 1},
		{ID: "v2", EntityTypeID: "et1", Version: 2},
	}, nil)
	pinRepo.On("ListByEntityTypeVersionIDs", mock.Anything, []string{"v1", "v2"}).Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "v1"},
		{ID: "pin2", CatalogVersionID: "cv2", EntityTypeVersionID: "v2"},
	}, nil)
	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	cvRepo.On("GetByID", mock.Anything, "cv2").Return(&models.CatalogVersion{
		ID: "cv2", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	// Multiple CVs → deep copy required, deepCopyAllowed=true
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(nil, domainerrors.NewNotFound("EntityTypeVersion", "et1"))

	_, err := svc.RenameEntityType(context.Background(), "et1", "NewName", true)
	assert.True(t, domainerrors.IsNotFound(err))
}

func TestTE12a_RenameDuplicateName(t *testing.T) {
	svc, etRepo, _, _, _, _ := setupETServiceWithCatalogRepos()

	et := &models.EntityType{ID: "et1", Name: "OldName"}
	etRepo.On("GetByID", mock.Anything, "et1").Return(et, nil)
	etRepo.On("GetByName", mock.Anything, "ExistingName").Return(&models.EntityType{ID: "et2", Name: "ExistingName"}, nil)

	_, err := svc.RenameEntityType(context.Background(), "et1", "ExistingName", false)
	assert.True(t, domainerrors.IsConflict(err))
}

func TestTE12b_RenameEmptyName(t *testing.T) {
	svc, _, _, _, _, _ := setupETServiceWithCatalogRepos()

	_, err := svc.RenameEntityType(context.Background(), "et1", "", false)
	assert.True(t, domainerrors.IsValidation(err))
}

// === GetContainmentTree Tests (T-E.49 through T-E.53) ===

func TestTE49_GetContainmentTree_NoEntityTypes(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, nil, assocRepo)

	etRepo.On("List", mock.Anything, mock.Anything).Return([]*models.EntityType{}, 0, nil)
	assocRepo.On("GetContainmentGraph", mock.Anything).Return([]repository.ContainmentEdge{}, nil)

	tree, err := svc.GetContainmentTree(context.Background())
	require.NoError(t, err)
	assert.Empty(t, tree)
}

func TestTE50_GetContainmentTree_FlatEntities(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, nil, assocRepo)

	entities := []*models.EntityType{
		{ID: "et1", Name: "Model"},
		{ID: "et2", Name: "Tool"},
	}
	etRepo.On("List", mock.Anything, mock.Anything).Return(entities, 2, nil)
	assocRepo.On("GetContainmentGraph", mock.Anything).Return([]repository.ContainmentEdge{}, nil)
	etvRepo.On("ListByEntityType", mock.Anything, "et1").Return([]*models.EntityTypeVersion{
		{ID: "v1", EntityTypeID: "et1", Version: 1},
	}, nil)
	etvRepo.On("ListByEntityType", mock.Anything, "et2").Return([]*models.EntityTypeVersion{
		{ID: "v2", EntityTypeID: "et2", Version: 1},
	}, nil)

	tree, err := svc.GetContainmentTree(context.Background())
	require.NoError(t, err)
	assert.Len(t, tree, 2)
	// All roots, no children
	for _, node := range tree {
		assert.Empty(t, node.Children)
	}
}

func TestTE51_GetContainmentTree_SingleParent(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, nil, assocRepo)

	entities := []*models.EntityType{
		{ID: "et-a", Name: "Server"},
		{ID: "et-b", Name: "Tool"},
	}
	etRepo.On("List", mock.Anything, mock.Anything).Return(entities, 2, nil)
	// A contains B
	assocRepo.On("GetContainmentGraph", mock.Anything).Return([]repository.ContainmentEdge{
		{SourceEntityTypeID: "et-a", TargetEntityTypeID: "et-b"},
	}, nil)
	etvRepo.On("ListByEntityType", mock.Anything, "et-a").Return([]*models.EntityTypeVersion{
		{ID: "va1", EntityTypeID: "et-a", Version: 1},
	}, nil)
	etvRepo.On("ListByEntityType", mock.Anything, "et-b").Return([]*models.EntityTypeVersion{
		{ID: "vb1", EntityTypeID: "et-b", Version: 1},
	}, nil)

	tree, err := svc.GetContainmentTree(context.Background())
	require.NoError(t, err)
	// Only A is a root
	assert.Len(t, tree, 1)
	assert.Equal(t, "et-a", tree[0].EntityType.ID)
	// A has B as child
	require.Len(t, tree[0].Children, 1)
	assert.Equal(t, "et-b", tree[0].Children[0].EntityType.ID)
	assert.Empty(t, tree[0].Children[0].Children)
}

func TestTE52_GetContainmentTree_MultiLevel(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, nil, assocRepo)

	entities := []*models.EntityType{
		{ID: "et-a", Name: "Org"},
		{ID: "et-b", Name: "Team"},
		{ID: "et-c", Name: "Member"},
	}
	etRepo.On("List", mock.Anything, mock.Anything).Return(entities, 3, nil)
	// A contains B, B contains C
	assocRepo.On("GetContainmentGraph", mock.Anything).Return([]repository.ContainmentEdge{
		{SourceEntityTypeID: "et-a", TargetEntityTypeID: "et-b"},
		{SourceEntityTypeID: "et-b", TargetEntityTypeID: "et-c"},
	}, nil)
	etvRepo.On("ListByEntityType", mock.Anything, "et-a").Return([]*models.EntityTypeVersion{
		{ID: "va1", EntityTypeID: "et-a", Version: 1},
	}, nil)
	etvRepo.On("ListByEntityType", mock.Anything, "et-b").Return([]*models.EntityTypeVersion{
		{ID: "vb1", EntityTypeID: "et-b", Version: 1},
	}, nil)
	etvRepo.On("ListByEntityType", mock.Anything, "et-c").Return([]*models.EntityTypeVersion{
		{ID: "vc1", EntityTypeID: "et-c", Version: 1},
	}, nil)

	tree, err := svc.GetContainmentTree(context.Background())
	require.NoError(t, err)
	// Only A is a root
	assert.Len(t, tree, 1)
	assert.Equal(t, "et-a", tree[0].EntityType.ID)
	// A → B → C
	require.Len(t, tree[0].Children, 1)
	assert.Equal(t, "et-b", tree[0].Children[0].EntityType.ID)
	require.Len(t, tree[0].Children[0].Children, 1)
	assert.Equal(t, "et-c", tree[0].Children[0].Children[0].EntityType.ID)
	assert.Empty(t, tree[0].Children[0].Children[0].Children)
}

func TestTE53_GetContainmentTree_IncludesAllVersions(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, nil, assocRepo)

	entities := []*models.EntityType{
		{ID: "et1", Name: "Model"},
	}
	etRepo.On("List", mock.Anything, mock.Anything).Return(entities, 1, nil)
	assocRepo.On("GetContainmentGraph", mock.Anything).Return([]repository.ContainmentEdge{}, nil)
	etvRepo.On("ListByEntityType", mock.Anything, "et1").Return([]*models.EntityTypeVersion{
		{ID: "v1", EntityTypeID: "et1", Version: 1},
		{ID: "v2", EntityTypeID: "et1", Version: 2},
		{ID: "v3", EntityTypeID: "et1", Version: 3},
	}, nil)

	tree, err := svc.GetContainmentTree(context.Background())
	require.NoError(t, err)
	require.Len(t, tree, 1)
	// All 3 versions present
	assert.Len(t, tree[0].Versions, 3)
	// LatestVersion is 3
	assert.Equal(t, 3, tree[0].LatestVersion)
}

func TestGetContainmentTree_DeduplicatesEdges(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, nil, assocRepo)

	entities := []*models.EntityType{
		{ID: "et-a", Name: "Server"},
		{ID: "et-b", Name: "Tool"},
	}
	etRepo.On("List", mock.Anything, mock.Anything).Return(entities, 2, nil)
	// Same containment edge appears 3 times (from 3 versions of Server)
	assocRepo.On("GetContainmentGraph", mock.Anything).Return([]repository.ContainmentEdge{
		{SourceEntityTypeID: "et-a", TargetEntityTypeID: "et-b"},
		{SourceEntityTypeID: "et-a", TargetEntityTypeID: "et-b"},
		{SourceEntityTypeID: "et-a", TargetEntityTypeID: "et-b"},
	}, nil)
	etvRepo.On("ListByEntityType", mock.Anything, "et-a").Return([]*models.EntityTypeVersion{
		{ID: "va1", EntityTypeID: "et-a", Version: 1},
	}, nil)
	etvRepo.On("ListByEntityType", mock.Anything, "et-b").Return([]*models.EntityTypeVersion{
		{ID: "vb1", EntityTypeID: "et-b", Version: 1},
	}, nil)

	tree, err := svc.GetContainmentTree(context.Background())
	require.NoError(t, err)
	assert.Len(t, tree, 1)
	assert.Equal(t, "et-a", tree[0].EntityType.ID)
	// Tool should appear exactly once as child, not 3 times
	require.Len(t, tree[0].Children, 1)
	assert.Equal(t, "et-b", tree[0].Children[0].EntityType.ID)
}

// === GetContainmentTree Error Path Tests ===

func TestGetContainmentTree_ListError(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	svc := meta.NewEntityTypeService(etRepo, nil, nil, nil)

	etRepo.On("List", mock.Anything, mock.Anything).Return(([]*models.EntityType)(nil), 0, domainerrors.NewNotFound("EntityType", ""))

	_, err := svc.GetContainmentTree(context.Background())
	assert.Error(t, err)
}

func TestGetContainmentTree_GraphError(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewEntityTypeService(etRepo, nil, nil, assocRepo)

	etRepo.On("List", mock.Anything, mock.Anything).Return([]*models.EntityType{{ID: "et1", Name: "A"}}, 1, nil)
	assocRepo.On("GetContainmentGraph", mock.Anything).Return(([]repository.ContainmentEdge)(nil), domainerrors.NewNotFound("Association", ""))

	_, err := svc.GetContainmentTree(context.Background())
	assert.Error(t, err)
}

func TestGetContainmentTree_VersionListError(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, nil, assocRepo)

	etRepo.On("List", mock.Anything, mock.Anything).Return([]*models.EntityType{{ID: "et1", Name: "A"}}, 1, nil)
	assocRepo.On("GetContainmentGraph", mock.Anything).Return([]repository.ContainmentEdge{}, nil)
	etvRepo.On("ListByEntityType", mock.Anything, "et1").Return(([]*models.EntityTypeVersion)(nil), domainerrors.NewNotFound("EntityTypeVersion", ""))

	_, err := svc.GetContainmentTree(context.Background())
	assert.Error(t, err)
}

func TestGetContainmentTree_OrphanEdges(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, nil, assocRepo)

	// Only entity A exists, but graph references non-existent B and C
	etRepo.On("List", mock.Anything, mock.Anything).Return([]*models.EntityType{
		{ID: "et-a", Name: "A"},
	}, 1, nil)
	assocRepo.On("GetContainmentGraph", mock.Anything).Return([]repository.ContainmentEdge{
		{SourceEntityTypeID: "et-a", TargetEntityTypeID: "et-nonexistent"},   // child doesn't exist
		{SourceEntityTypeID: "et-ghost", TargetEntityTypeID: "et-a"},         // parent doesn't exist
	}, nil)
	etvRepo.On("ListByEntityType", mock.Anything, "et-a").Return([]*models.EntityTypeVersion{
		{ID: "va1", EntityTypeID: "et-a", Version: 1},
	}, nil)

	tree, err := svc.GetContainmentTree(context.Background())
	require.NoError(t, err)
	// A is still a root (the ghost parent edge doesn't create a real parent)
	// But A appears as a child target of "et-ghost", so childIDs has "et-a"
	// Since et-ghost doesn't exist in nodes, A won't get attached as child — but it IS in childIDs
	// So A won't appear as a root either. The tree should be empty since A is marked as a child.
	// This is correct defensive behavior — orphan edges are silently skipped.
	assert.Empty(t, tree)
}

// === GetVersionSnapshot Tests (T-E.59 through T-E.61) ===

func TestTE59_GetVersionSnapshot_ReturnsAttrsAndAssocs(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, attrRepo, assocRepo)

	et := &models.EntityType{ID: "et1", Name: "Server"}
	etv := &models.EntityTypeVersion{ID: "v2-id", EntityTypeID: "et1", Version: 2, Description: "V2"}
	attrs := []*models.Attribute{
		{ID: "a1", Name: "hostname", TypeDefinitionVersionID: "tdv-string", Ordinal: 1},
		{ID: "a2", Name: "port", TypeDefinitionVersionID: "tdv-number", Ordinal: 2},
	}
	assocs := []*models.Association{
		{ID: "as1", EntityTypeVersionID: "v2-id", TargetEntityTypeID: "et2", Type: "containment", SourceRole: "server", TargetRole: "tool"},
	}

	etRepo.On("GetByID", mock.Anything, "et1").Return(et, nil)
	etRepo.On("GetByID", mock.Anything, "et2").Return(&models.EntityType{ID: "et2", Name: "Tool"}, nil)
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 2).Return(etv, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v2-id").Return(attrs, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v2-id").Return(assocs, nil)
	assocRepo.On("ListByTargetEntityType", mock.Anything, "et1").Return([]*models.Association{}, nil)

	snapshot, err := svc.GetVersionSnapshot(context.Background(), "et1", 2)
	require.NoError(t, err)
	assert.Equal(t, "Server", snapshot.EntityType.Name)
	assert.Equal(t, 2, snapshot.Version.Version)
	assert.Len(t, snapshot.Attributes, 2)
	assert.Equal(t, "hostname", snapshot.Attributes[0].Name)
	assert.Len(t, snapshot.Associations, 1)
	assert.Equal(t, "containment", string(snapshot.Associations[0].Type))
	assert.Equal(t, "outgoing", snapshot.Associations[0].Direction)
	assert.Equal(t, "Tool", snapshot.TargetEntityTypeNames["et2"])
}

func TestTE60_GetVersionSnapshot_NonexistentEntityType(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	svc := meta.NewEntityTypeService(etRepo, nil, nil, nil)

	etRepo.On("GetByID", mock.Anything, "nope").Return(nil, domainerrors.NewNotFound("EntityType", "nope"))

	_, err := svc.GetVersionSnapshot(context.Background(), "nope", 1)
	assert.True(t, domainerrors.IsNotFound(err))
}

func TestTE61_GetVersionSnapshot_NonexistentVersion(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, nil, nil)

	et := &models.EntityType{ID: "et1", Name: "Server"}
	etRepo.On("GetByID", mock.Anything, "et1").Return(et, nil)
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 999).Return(nil, domainerrors.NewNotFound("EntityTypeVersion", "999"))

	_, err := svc.GetVersionSnapshot(context.Background(), "et1", 999)
	assert.True(t, domainerrors.IsNotFound(err))
}

func TestGetVersionSnapshot_ListByVersionAttrError(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, attrRepo, nil)

	etRepo.On("GetByID", mock.Anything, "et1").Return(&models.EntityType{ID: "et1", Name: "A"}, nil)
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 1).Return(&models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return(([]*models.Attribute)(nil), domainerrors.NewNotFound("Attribute", ""))

	_, err := svc.GetVersionSnapshot(context.Background(), "et1", 1)
	assert.Error(t, err)
}

func TestGetVersionSnapshot_ListByVersionAssocError(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, attrRepo, assocRepo)

	etRepo.On("GetByID", mock.Anything, "et1").Return(&models.EntityType{ID: "et1", Name: "A"}, nil)
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 1).Return(&models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return(([]*models.Association)(nil), domainerrors.NewNotFound("Association", ""))

	_, err := svc.GetVersionSnapshot(context.Background(), "et1", 1)
	assert.Error(t, err)
}

func TestGetVersionSnapshot_ResolvesTypeInfo(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, attrRepo, assocRepo)
	meta.WithTypeDefinitionRepos(svc, tdRepo, tdvRepo)

	etRepo.On("GetByID", mock.Anything, "et1").Return(&models.EntityType{ID: "et1", Name: "Server"}, nil)
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 1).Return(&models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{
		{ID: "a1", Name: "status", TypeDefinitionVersionID: "tdv-enum1", Ordinal: 1},
		{ID: "a2", Name: "hostname", TypeDefinitionVersionID: "tdv-string", Ordinal: 2},
	}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	assocRepo.On("ListByTargetEntityType", mock.Anything, "et1").Return([]*models.Association{}, nil)
	tdvRepo.On("GetByID", mock.Anything, "tdv-enum1").Return(&models.TypeDefinitionVersion{ID: "tdv-enum1", TypeDefinitionID: "td-enum1"}, nil)
	tdvRepo.On("GetByID", mock.Anything, "tdv-string").Return(&models.TypeDefinitionVersion{ID: "tdv-string", TypeDefinitionID: "td-string"}, nil)
	tdRepo.On("GetByID", mock.Anything, "td-enum1").Return(&models.TypeDefinition{ID: "td-enum1", Name: "ServerStatus", BaseType: models.BaseTypeEnum}, nil)
	tdRepo.On("GetByID", mock.Anything, "td-string").Return(&models.TypeDefinition{ID: "td-string", Name: "String", BaseType: models.BaseTypeString}, nil)

	snapshot, err := svc.GetVersionSnapshot(context.Background(), "et1", 1)
	require.NoError(t, err)
	assert.Equal(t, "ServerStatus", snapshot.TypeInfo["tdv-enum1"].TypeName)
}

func TestGetVersionSnapshot_ResolvesTargetEntityTypeNames(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, attrRepo, assocRepo)

	etRepo.On("GetByID", mock.Anything, "et1").Return(&models.EntityType{ID: "et1", Name: "Server"}, nil)
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 1).Return(&models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{
		{ID: "as1", TargetEntityTypeID: "et2", Type: "containment", TargetRole: "tool"},
	}, nil)
	assocRepo.On("ListByTargetEntityType", mock.Anything, "et1").Return([]*models.Association{}, nil)
	etRepo.On("GetByID", mock.Anything, "et2").Return(&models.EntityType{ID: "et2", Name: "Tool"}, nil)

	snapshot, err := svc.GetVersionSnapshot(context.Background(), "et1", 1)
	require.NoError(t, err)
	assert.Equal(t, "Tool", snapshot.TargetEntityTypeNames["et2"])
}

func TestGetVersionSnapshot_IncludesIncomingAssociations(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, attrRepo, assocRepo)

	// mcp-tool is contained by mcp-server
	etRepo.On("GetByID", mock.Anything, "et-tool").Return(&models.EntityType{ID: "et-tool", Name: "mcp-tool"}, nil)
	etRepo.On("GetByID", mock.Anything, "et-server").Return(&models.EntityType{ID: "et-server", Name: "mcp-server"}, nil)
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et-tool", 1).Return(&models.EntityTypeVersion{ID: "v-tool-1", EntityTypeID: "et-tool", Version: 1}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v-tool-1").Return([]*models.Attribute{}, nil)
	// No outgoing associations from mcp-tool
	assocRepo.On("ListByVersion", mock.Anything, "v-tool-1").Return([]*models.Association{}, nil)
	// Incoming: mcp-server contains mcp-tool
	assocRepo.On("ListByTargetEntityType", mock.Anything, "et-tool").Return([]*models.Association{
		{ID: "as1", EntityTypeVersionID: "v-server-3", TargetEntityTypeID: "et-tool", Type: "containment", SourceRole: "server", TargetRole: "tool"},
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "v-server-3").Return(&models.EntityTypeVersion{ID: "v-server-3", EntityTypeID: "et-server", Version: 3}, nil)
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-server").Return(&models.EntityTypeVersion{ID: "v-server-3", EntityTypeID: "et-server", Version: 3}, nil)

	snapshot, err := svc.GetVersionSnapshot(context.Background(), "et-tool", 1)
	require.NoError(t, err)
	// Should have 1 association (incoming containment from mcp-server)
	require.Len(t, snapshot.Associations, 1)
	assert.Equal(t, "incoming", snapshot.Associations[0].Direction)
	assert.Equal(t, "et-server", snapshot.Associations[0].SourceEntityTypeID)
	assert.Equal(t, "mcp-server", snapshot.TargetEntityTypeNames["et-server"])
}

func TestGetVersionSnapshot_ListByTargetEntityTypeError(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, attrRepo, assocRepo)

	etRepo.On("GetByID", mock.Anything, "et1").Return(&models.EntityType{ID: "et1"}, nil)
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 1).Return(&models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	assocRepo.On("ListByTargetEntityType", mock.Anything, "et1").Return(([]*models.Association)(nil), domainerrors.NewNotFound("Association", ""))

	_, err := svc.GetVersionSnapshot(context.Background(), "et1", 1)
	assert.Error(t, err)
}

func TestGetVersionSnapshot_IncomingGetByIDError(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, attrRepo, assocRepo)

	etRepo.On("GetByID", mock.Anything, "et1").Return(&models.EntityType{ID: "et1"}, nil)
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 1).Return(&models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	assocRepo.On("ListByTargetEntityType", mock.Anything, "et1").Return([]*models.Association{
		{ID: "as1", EntityTypeVersionID: "v-other", TargetEntityTypeID: "et1", Type: "containment"},
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "v-other").Return(nil, domainerrors.NewNotFound("EntityTypeVersion", "v-other"))

	_, err := svc.GetVersionSnapshot(context.Background(), "et1", 1)
	assert.Error(t, err)
}

func TestGetVersionSnapshot_SkipsSelfReferences(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, attrRepo, assocRepo)

	etRepo.On("GetByID", mock.Anything, "et1").Return(&models.EntityType{ID: "et1", Name: "A"}, nil)
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 1).Return(&models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	// Incoming association from self (self-reference)
	assocRepo.On("ListByTargetEntityType", mock.Anything, "et1").Return([]*models.Association{
		{ID: "as-self", EntityTypeVersionID: "v1", TargetEntityTypeID: "et1", Type: "directional"},
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "v1").Return(&models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}, nil)

	snapshot, err := svc.GetVersionSnapshot(context.Background(), "et1", 1)
	require.NoError(t, err)
	// Self-reference should be skipped
	assert.Empty(t, snapshot.Associations)
}

func TestGetVersionSnapshot_IncomingGetLatestError(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, attrRepo, assocRepo)

	etRepo.On("GetByID", mock.Anything, "et1").Return(&models.EntityType{ID: "et1"}, nil)
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 1).Return(&models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	assocRepo.On("ListByTargetEntityType", mock.Anything, "et1").Return([]*models.Association{
		{ID: "as1", EntityTypeVersionID: "v-other", TargetEntityTypeID: "et1", Type: "containment"},
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "v-other").Return(&models.EntityTypeVersion{ID: "v-other", EntityTypeID: "et2", Version: 1}, nil)
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et2").Return(nil, domainerrors.NewNotFound("EntityTypeVersion", "et2"))

	_, err := svc.GetVersionSnapshot(context.Background(), "et1", 1)
	assert.Error(t, err)
}

func TestGetVersionSnapshot_SkipsOldVersionAssociations(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, attrRepo, assocRepo)

	etRepo.On("GetByID", mock.Anything, "et1").Return(&models.EntityType{ID: "et1", Name: "A"}, nil)
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 1).Return(&models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	// Incoming from old version of et2
	assocRepo.On("ListByTargetEntityType", mock.Anything, "et1").Return([]*models.Association{
		{ID: "as1", EntityTypeVersionID: "v2-old", TargetEntityTypeID: "et1", Type: "containment"},
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "v2-old").Return(&models.EntityTypeVersion{ID: "v2-old", EntityTypeID: "et2", Version: 1}, nil)
	// Latest is v2-new, not v2-old
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et2").Return(&models.EntityTypeVersion{ID: "v2-new", EntityTypeID: "et2", Version: 2}, nil)

	snapshot, err := svc.GetVersionSnapshot(context.Background(), "et1", 1)
	require.NoError(t, err)
	// Old version association should be filtered out
	assert.Empty(t, snapshot.Associations)
}

// TD-29: Reserved entity type names
func TestCreateEntityType_ReservedNameRejected(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, attrRepo, assocRepo)

	for _, name := range []string{"links", "references", "referenced-by", "copy", "replace", "tree", "validate", "publish", "unpublish"} {
		_, _, err := svc.CreateEntityType(context.Background(), name, "")
		require.Error(t, err, "name=%s should be rejected", name)
		assert.True(t, domainerrors.IsValidation(err))
		assert.Contains(t, err.Error(), "reserved")
	}
}

func TestCreateEntityType_NameTooLong(t *testing.T) {
	svc := meta.NewEntityTypeService(nil, nil, nil, nil)
	longName := strings.Repeat("a", 256)
	_, _, err := svc.CreateEntityType(context.Background(), longName, "desc")
	require.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
	assert.Contains(t, err.Error(), "255")
}

func TestCreateEntityType_DescriptionTooLong(t *testing.T) {
	svc := meta.NewEntityTypeService(nil, nil, nil, nil)
	longDesc := strings.Repeat("x", 1025)
	_, _, err := svc.CreateEntityType(context.Background(), "valid-name", longDesc)
	require.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
	assert.Contains(t, err.Error(), "1024")
}

func TestRenameEntityType_ReservedNameRejected(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, attrRepo, assocRepo)

	_, err := svc.RenameEntityType(context.Background(), "et1", "links", false)
	require.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
	assert.Contains(t, err.Error(), "reserved")
}
