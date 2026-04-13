package meta_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository/mocks"
	"github.com/project-catalyst/pc-asset-hub/internal/service/meta"
)

func setupVersionHistoryService() (*meta.VersionHistoryService, *mocks.MockEntityTypeVersionRepo, *mocks.MockAttributeRepo, *mocks.MockAssociationRepo) {
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewVersionHistoryService(etvRepo, attrRepo, assocRepo)
	return svc, etvRepo, attrRepo, assocRepo
}

func TestT3_48_GetVersionHistory(t *testing.T) {
	svc, etvRepo, _, _ := setupVersionHistoryService()

	versions := []*models.EntityTypeVersion{
		{ID: "v1", Version: 1},
		{ID: "v2", Version: 2},
	}
	etvRepo.On("ListByEntityType", mock.Anything, "et1").Return(versions, nil)

	result, err := svc.GetVersionHistory(context.Background(), "et1")
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, 1, result[0].Version)
	assert.Equal(t, 2, result[1].Version)
}

func TestT3_49_CompareVersionsAttributeAdded(t *testing.T) {
	svc, etvRepo, attrRepo, assocRepo := setupVersionHistoryService()

	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 1).Return(&models.EntityTypeVersion{ID: "v1"}, nil)
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 2).Return(&models.EntityTypeVersion{ID: "v2"}, nil)

	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v2").Return([]*models.Attribute{
		{Name: "endpoint", TypeDefinitionVersionID: "tdv-string"},
	}, nil)

	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v2").Return([]*models.Association{}, nil)

	diff, err := svc.CompareVersions(context.Background(), "et1", 1, 2)
	require.NoError(t, err)
	assert.Len(t, diff.Changes, 1)
	assert.Equal(t, "added", diff.Changes[0].ChangeType)
	assert.Equal(t, "endpoint", diff.Changes[0].Name)
}

func TestT3_50_CompareVersionsAttributeRemoved(t *testing.T) {
	svc, etvRepo, attrRepo, assocRepo := setupVersionHistoryService()

	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 1).Return(&models.EntityTypeVersion{ID: "v1"}, nil)
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 2).Return(&models.EntityTypeVersion{ID: "v2"}, nil)

	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{
		{Name: "endpoint", TypeDefinitionVersionID: "tdv-string"},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v2").Return([]*models.Attribute{}, nil)

	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v2").Return([]*models.Association{}, nil)

	diff, err := svc.CompareVersions(context.Background(), "et1", 1, 2)
	require.NoError(t, err)
	assert.Len(t, diff.Changes, 1)
	assert.Equal(t, "removed", diff.Changes[0].ChangeType)
}

func TestT3_51_CompareVersionsAttributeModified(t *testing.T) {
	svc, etvRepo, attrRepo, assocRepo := setupVersionHistoryService()

	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 1).Return(&models.EntityTypeVersion{ID: "v1"}, nil)
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 2).Return(&models.EntityTypeVersion{ID: "v2"}, nil)

	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{
		{Name: "endpoint", TypeDefinitionVersionID: "tdv-string", Description: "old desc"},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v2").Return([]*models.Attribute{
		{Name: "endpoint", TypeDefinitionVersionID: "tdv-string", Description: "new desc"},
	}, nil)

	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v2").Return([]*models.Association{}, nil)

	diff, err := svc.CompareVersions(context.Background(), "et1", 1, 2)
	require.NoError(t, err)
	assert.Len(t, diff.Changes, 1)
	assert.Equal(t, "modified", diff.Changes[0].ChangeType)
}

func TestT3_52_CompareVersionsAssociationChanged(t *testing.T) {
	svc, etvRepo, attrRepo, assocRepo := setupVersionHistoryService()

	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 1).Return(&models.EntityTypeVersion{ID: "v1"}, nil)
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 2).Return(&models.EntityTypeVersion{ID: "v2"}, nil)

	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v2").Return([]*models.Attribute{}, nil)

	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v2").Return([]*models.Association{
		{TargetEntityTypeID: "et-b", Type: models.AssociationTypeContainment},
	}, nil)

	diff, err := svc.CompareVersions(context.Background(), "et1", 1, 2)
	require.NoError(t, err)
	hasAssocChange := false
	for _, c := range diff.Changes {
		if c.Category == "association" {
			hasAssocChange = true
		}
	}
	assert.True(t, hasAssocChange)
}

func TestT3_53_CompareVersionsSameVersion(t *testing.T) {
	svc, etvRepo, attrRepo, assocRepo := setupVersionHistoryService()

	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 1).Return(&models.EntityTypeVersion{ID: "v1"}, nil)

	attrs := []*models.Attribute{{Name: "a", TypeDefinitionVersionID: "tdv-string"}}
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return(attrs, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)

	diff, err := svc.CompareVersions(context.Background(), "et1", 1, 1)
	require.NoError(t, err)
	assert.Len(t, diff.Changes, 0) // No changes
}
