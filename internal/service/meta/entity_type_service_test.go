package meta_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
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
