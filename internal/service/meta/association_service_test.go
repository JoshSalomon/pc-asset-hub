package meta_test

import (
	"context"
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

func setupAssocService() (*meta.AssociationService, *mocks.MockAssociationRepo, *mocks.MockEntityTypeVersionRepo, *mocks.MockAttributeRepo) {
	assocRepo := new(mocks.MockAssociationRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	svc := meta.NewAssociationService(assocRepo, etvRepo, attrRepo)
	return svc, assocRepo, etvRepo, attrRepo
}

func TestT3_20_CreateContainmentAssociation(t *testing.T) {
	svc, assocRepo, etvRepo, attrRepo := setupAssocService()

	assocRepo.On("GetContainmentGraph", mock.Anything).Return([]repository.ContainmentEdge{}, nil)
	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-a", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(v1, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Association")).Return(nil)

	newVer, err := svc.CreateAssociation(context.Background(), "et-a", "et-b", models.AssociationTypeContainment, "contains", "part_of")
	require.NoError(t, err)
	assert.Equal(t, 2, newVer.Version)
}

func TestT3_21_CycleDetectionDirect(t *testing.T) {
	svc, assocRepo, etvRepo, _ := setupAssocService()

	// A contains B already exists, now trying B contains A
	assocRepo.On("GetContainmentGraph", mock.Anything).Return([]repository.ContainmentEdge{
		{SourceEntityTypeID: "et-a", TargetEntityTypeID: "et-b"},
	}, nil)
	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-b", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-b").Return(v1, nil)

	_, err := svc.CreateAssociation(context.Background(), "et-b", "et-a", models.AssociationTypeContainment, "", "")
	assert.True(t, domainerrors.IsCycleDetected(err))
}

func TestT3_22_CycleDetectionIndirect(t *testing.T) {
	svc, assocRepo, etvRepo, _ := setupAssocService()

	// A->B->C exists, trying C->A
	assocRepo.On("GetContainmentGraph", mock.Anything).Return([]repository.ContainmentEdge{
		{SourceEntityTypeID: "et-a", TargetEntityTypeID: "et-b"},
		{SourceEntityTypeID: "et-b", TargetEntityTypeID: "et-c"},
	}, nil)
	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-c", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-c").Return(v1, nil)

	_, err := svc.CreateAssociation(context.Background(), "et-c", "et-a", models.AssociationTypeContainment, "", "")
	assert.True(t, domainerrors.IsCycleDetected(err))
}

func TestT3_23_SelfContainment(t *testing.T) {
	svc, assocRepo, _, _ := setupAssocService()

	assocRepo.On("GetContainmentGraph", mock.Anything).Return([]repository.ContainmentEdge{}, nil)

	_, err := svc.CreateAssociation(context.Background(), "et-a", "et-a", models.AssociationTypeContainment, "", "")
	assert.True(t, domainerrors.IsCycleDetected(err))
}

func TestT3_24_DirectionalReferenceNoCycleCheck(t *testing.T) {
	svc, assocRepo, etvRepo, attrRepo := setupAssocService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-a", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(v1, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	newVer, err := svc.CreateAssociation(context.Background(), "et-a", "et-b", models.AssociationTypeDirectional, "refers_to", "referred_by")
	require.NoError(t, err)
	assert.Equal(t, 2, newVer.Version)
	// GetContainmentGraph should NOT be called for directional references
	assocRepo.AssertNotCalled(t, "GetContainmentGraph", mock.Anything)
}

func TestT3_25_BidirectionalReferenceNoCycleCheck(t *testing.T) {
	svc, assocRepo, etvRepo, attrRepo := setupAssocService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-a", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(v1, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	newVer, err := svc.CreateAssociation(context.Background(), "et-a", "et-b", models.AssociationTypeBidirectional, "", "")
	require.NoError(t, err)
	assert.Equal(t, 2, newVer.Version)
	assocRepo.AssertNotCalled(t, "GetContainmentGraph", mock.Anything)
}

func TestT3_26_CreateAssociationIncrementsVersion(t *testing.T) {
	// Already verified in T3_20 — newVer.Version == 2
	t.Log("Covered by T3_20")
}

func TestT3_27_DeleteAssociation(t *testing.T) {
	svc, assocRepo, etvRepo, attrRepo := setupAssocService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-a", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(v1, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("GetByID", mock.Anything, "assoc-1").Return(&models.Association{
		ID: "assoc-1", TargetEntityTypeID: "et-b", Type: models.AssociationTypeContainment,
	}, nil)
	assocRepo.On("ListByVersion", mock.Anything, mock.AnythingOfType("string")).Return([]*models.Association{
		{ID: "new-assoc-1", TargetEntityTypeID: "et-b", Type: models.AssociationTypeContainment},
	}, nil)
	assocRepo.On("Delete", mock.Anything, "new-assoc-1").Return(nil)

	newVer, err := svc.DeleteAssociation(context.Background(), "et-a", "assoc-1")
	require.NoError(t, err)
	assert.Equal(t, 2, newVer.Version)
}

func TestT3_28_CycleDetectionComplexDAG(t *testing.T) {
	svc, assocRepo, etvRepo, attrRepo := setupAssocService()

	// Complex DAG: A->B, A->C, B->D, C->D, B->E — all valid, no cycles
	assocRepo.On("GetContainmentGraph", mock.Anything).Return([]repository.ContainmentEdge{
		{SourceEntityTypeID: "a", TargetEntityTypeID: "b"},
		{SourceEntityTypeID: "a", TargetEntityTypeID: "c"},
		{SourceEntityTypeID: "b", TargetEntityTypeID: "d"},
		{SourceEntityTypeID: "c", TargetEntityTypeID: "d"},
		{SourceEntityTypeID: "b", TargetEntityTypeID: "e"},
	}, nil)

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "e", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "e").Return(v1, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	// E->F is valid (no cycle)
	newVer, err := svc.CreateAssociation(context.Background(), "e", "f", models.AssociationTypeContainment, "", "")
	require.NoError(t, err)
	assert.NotNil(t, newVer)
}
