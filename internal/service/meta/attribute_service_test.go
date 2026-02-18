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

func setupAttrService() (*meta.AttributeService, *mocks.MockAttributeRepo, *mocks.MockEntityTypeVersionRepo, *mocks.MockAssociationRepo, *mocks.MockEnumRepo) {
	attrRepo := new(mocks.MockAttributeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	etRepo := new(mocks.MockEntityTypeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	enumRepo := new(mocks.MockEnumRepo)
	svc := meta.NewAttributeService(attrRepo, etvRepo, etRepo, assocRepo, enumRepo)
	return svc, attrRepo, etvRepo, assocRepo, enumRepo
}

func TestT3_11_AddAttribute(t *testing.T) {
	svc, attrRepo, etvRepo, assocRepo, _ := setupAttrService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, "v1", mock.AnythingOfType("string")).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, "v1", mock.AnythingOfType("string")).Return(nil)
	attrRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Attribute")).Return(nil)

	newVer, err := svc.AddAttribute(context.Background(), "et1", "endpoint", "API endpoint", models.AttributeTypeString, "")
	require.NoError(t, err)
	assert.Equal(t, 2, newVer.Version)
}

func TestT3_12_AddAttributeDuplicateName(t *testing.T) {
	svc, attrRepo, etvRepo, _, _ := setupAttrService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{
		{Name: "endpoint", Type: models.AttributeTypeString},
	}, nil)

	_, err := svc.AddAttribute(context.Background(), "et1", "endpoint", "", models.AttributeTypeString, "")
	assert.True(t, domainerrors.IsConflict(err))
}

func TestT3_13_AddAttributeEnumValid(t *testing.T) {
	svc, attrRepo, etvRepo, assocRepo, enumRepo := setupAttrService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	enumRepo.On("GetByID", mock.Anything, "enum1").Return(&models.Enum{ID: "enum1", Name: "Status"}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	_, err := svc.AddAttribute(context.Background(), "et1", "status", "", models.AttributeTypeEnum, "enum1")
	assert.NoError(t, err)
}

func TestT3_14_AddAttributeEnumInvalid(t *testing.T) {
	svc, attrRepo, etvRepo, _, enumRepo := setupAttrService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	enumRepo.On("GetByID", mock.Anything, "bad-enum").Return(nil, domainerrors.NewNotFound("Enum", "bad-enum"))

	_, err := svc.AddAttribute(context.Background(), "et1", "status", "", models.AttributeTypeEnum, "bad-enum")
	assert.True(t, domainerrors.IsValidation(err))
}

func TestT3_15_RemoveAttribute(t *testing.T) {
	svc, attrRepo, etvRepo, assocRepo, _ := setupAttrService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(v1, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, "v1", mock.AnythingOfType("string")).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, "v1", mock.AnythingOfType("string")).Return(nil)
	attrRepo.On("ListByVersion", mock.Anything, mock.AnythingOfType("string")).Return([]*models.Attribute{
		{ID: "attr1", Name: "endpoint", Type: models.AttributeTypeString},
	}, nil)
	attrRepo.On("Delete", mock.Anything, "attr1").Return(nil)

	newVer, err := svc.RemoveAttribute(context.Background(), "et1", "endpoint")
	require.NoError(t, err)
	assert.Equal(t, 2, newVer.Version)
}

func TestT3_16_CopyAttributesFromType(t *testing.T) {
	svc, attrRepo, etvRepo, assocRepo, _ := setupAttrService()

	srcV1 := &models.EntityTypeVersion{ID: "src-v1", EntityTypeID: "src-et", Version: 1}
	tgtV1 := &models.EntityTypeVersion{ID: "tgt-v1", EntityTypeID: "tgt-et", Version: 1}

	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "src-et", 1).Return(srcV1, nil)
	etvRepo.On("GetLatestByEntityType", mock.Anything, "tgt-et").Return(tgtV1, nil)

	attrRepo.On("ListByVersion", mock.Anything, "src-v1").Return([]*models.Attribute{
		{Name: "endpoint", Type: models.AttributeTypeString, Description: "API endpoint"},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "tgt-v1").Return([]*models.Attribute{}, nil)

	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, "tgt-v1", mock.AnythingOfType("string")).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, "tgt-v1", mock.AnythingOfType("string")).Return(nil)
	attrRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Attribute")).Return(nil)

	newVer, err := svc.CopyAttributesFromType(context.Background(), "tgt-et", "src-et", 1, []string{"endpoint"})
	require.NoError(t, err)
	assert.Equal(t, 2, newVer.Version)
}

func TestT3_17_CopyAttributesIncrementsVersion(t *testing.T) {
	// Already verified in T3_16 — newVer.Version == 2
	t.Log("Covered by T3_16")
}

func TestT3_18_CopyAttributesSourceUnchanged(t *testing.T) {
	svc, attrRepo, etvRepo, assocRepo, _ := setupAttrService()

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

	_, err := svc.CopyAttributesFromType(context.Background(), "tgt-et", "src-et", 1, []string{"attr1"})
	require.NoError(t, err)
	// Source was never modified
	attrRepo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
}

func TestT3_19_ReorderAttributes(t *testing.T) {
	svc, attrRepo, etvRepo, _, _ := setupAttrService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(v1, nil)
	attrRepo.On("Reorder", mock.Anything, "v1", []string{"c", "a", "b"}).Return(nil)

	err := svc.ReorderAttributes(context.Background(), "et1", []string{"c", "a", "b"})
	assert.NoError(t, err)
}
