package meta_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository/mocks"
	"github.com/project-catalyst/pc-asset-hub/internal/service/meta"
)

var dbErr = errors.New("db error")

// ============================================================================
// CreateAssociation error branches
// ============================================================================

func TestCreateAssociation_EtvRepoGetLatestError(t *testing.T) {
	svc, _, etvRepo, _ := setupAssocService()

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(nil, dbErr)

	_, err := svc.CreateAssociation(context.Background(), "et-a", "et-b", models.AssociationTypeDirectional, "test_assoc", "", "", "", "")
	assert.ErrorIs(t, err, dbErr)
}

func TestCreateAssociation_EtvRepoCreateError(t *testing.T) {
	svc, assocRepo, etvRepo, attrRepo := setupAssocService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-a", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(dbErr)

	_, err := svc.CreateAssociation(context.Background(), "et-a", "et-b", models.AssociationTypeDirectional, "test_assoc", "", "", "", "")
	assert.ErrorIs(t, err, dbErr)
}

func TestCreateAssociation_AttrBulkCopyError(t *testing.T) {
	svc, assocRepo, etvRepo, attrRepo := setupAssocService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-a", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(dbErr)

	_, err := svc.CreateAssociation(context.Background(), "et-a", "et-b", models.AssociationTypeDirectional, "test_assoc", "", "", "", "")
	assert.ErrorIs(t, err, dbErr)
}

func TestCreateAssociation_AssocBulkCopyError(t *testing.T) {
	svc, assocRepo, etvRepo, attrRepo := setupAssocService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-a", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(dbErr)

	_, err := svc.CreateAssociation(context.Background(), "et-a", "et-b", models.AssociationTypeDirectional, "test_assoc", "", "", "", "")
	assert.ErrorIs(t, err, dbErr)
}

func TestCreateAssociation_AssocCreateError(t *testing.T) {
	svc, assocRepo, etvRepo, attrRepo := setupAssocService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-a", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("Create", mock.Anything, mock.Anything).Return(dbErr)

	_, err := svc.CreateAssociation(context.Background(), "et-a", "et-b", models.AssociationTypeDirectional, "test_assoc", "", "", "", "")
	assert.ErrorIs(t, err, dbErr)
}

// ============================================================================
// DeleteAssociation error branches
// ============================================================================

func TestDeleteAssociation_GetLatestError(t *testing.T) {
	svc, _, etvRepo, _ := setupAssocService()

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(nil, dbErr)

	_, err := svc.DeleteAssociation(context.Background(), "et-a", "test_assoc")
	assert.ErrorIs(t, err, dbErr)
}

func TestDeleteAssociation_EtvCreateError(t *testing.T) {
	svc, _, etvRepo, _ := setupAssocService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-a", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(v1, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(dbErr)

	_, err := svc.DeleteAssociation(context.Background(), "et-a", "test_assoc")
	assert.ErrorIs(t, err, dbErr)
}

func TestDeleteAssociation_AttrBulkCopyError(t *testing.T) {
	svc, _, etvRepo, attrRepo := setupAssocService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-a", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(v1, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(dbErr)

	_, err := svc.DeleteAssociation(context.Background(), "et-a", "test_assoc")
	assert.ErrorIs(t, err, dbErr)
}

func TestDeleteAssociation_AssocBulkCopyError(t *testing.T) {
	svc, assocRepo, etvRepo, attrRepo := setupAssocService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-a", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(v1, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(dbErr)

	_, err := svc.DeleteAssociation(context.Background(), "et-a", "test_assoc")
	assert.ErrorIs(t, err, dbErr)
}

func TestDeleteAssociation_ListByVersionError(t *testing.T) {
	svc, assocRepo, etvRepo, attrRepo := setupAssocService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-a", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(v1, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Association(nil), dbErr)

	_, err := svc.DeleteAssociation(context.Background(), "et-a", "test_assoc")
	assert.ErrorIs(t, err, dbErr)
}

func TestDeleteAssociation_DeleteError(t *testing.T) {
	svc, assocRepo, etvRepo, attrRepo := setupAssocService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-a", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(v1, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Association{
		{ID: "new-assoc-1", Name: "test_assoc", TargetEntityTypeID: "et-b", Type: models.AssociationTypeContainment},
	}, nil)
	assocRepo.On("Delete", mock.Anything, "new-assoc-1").Return(dbErr)

	_, err := svc.DeleteAssociation(context.Background(), "et-a", "test_assoc")
	assert.ErrorIs(t, err, dbErr)
}

// ============================================================================
// AddAttribute error branches
// ============================================================================

func TestAddAttribute_EmptyName(t *testing.T) {
	svc, _, _, _, _ := setupAttrService()

	_, err := svc.AddAttribute(context.Background(), "et1", "", "", models.AttributeTypeString, "", false)
	assert.Error(t, err)
}

func TestAddAttribute_EnumMissingEnumID(t *testing.T) {
	svc, _, _, _, _ := setupAttrService()

	_, err := svc.AddAttribute(context.Background(), "et1", "status", "", models.AttributeTypeEnum, "", false)
	assert.Error(t, err)
}

func TestAddAttribute_GetLatestError(t *testing.T) {
	svc, _, etvRepo, _, _ := setupAttrService()

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(nil, dbErr)

	_, err := svc.AddAttribute(context.Background(), "et1", "attr", "", models.AttributeTypeString, "", false)
	assert.ErrorIs(t, err, dbErr)
}

func TestAddAttribute_ListByVersionError(t *testing.T) {
	svc, attrRepo, etvRepo, _, _ := setupAttrService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute(nil), dbErr)

	_, err := svc.AddAttribute(context.Background(), "et1", "attr", "", models.AttributeTypeString, "", false)
	assert.ErrorIs(t, err, dbErr)
}

func TestAddAttribute_EtvCreateError(t *testing.T) {
	svc, attrRepo, etvRepo, assocRepo, _ := setupAttrService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(dbErr)

	_, err := svc.AddAttribute(context.Background(), "et1", "attr", "", models.AttributeTypeString, "", false)
	assert.ErrorIs(t, err, dbErr)
}

func TestAddAttribute_AttrBulkCopyError(t *testing.T) {
	svc, attrRepo, etvRepo, assocRepo, _ := setupAttrService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(dbErr)

	_, err := svc.AddAttribute(context.Background(), "et1", "attr", "", models.AttributeTypeString, "", false)
	assert.ErrorIs(t, err, dbErr)
}

func TestAddAttribute_AssocBulkCopyError(t *testing.T) {
	svc, attrRepo, etvRepo, assocRepo, _ := setupAttrService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(dbErr)

	_, err := svc.AddAttribute(context.Background(), "et1", "attr", "", models.AttributeTypeString, "", false)
	assert.ErrorIs(t, err, dbErr)
}

func TestAddAttribute_AttrCreateError(t *testing.T) {
	svc, attrRepo, etvRepo, assocRepo, _ := setupAttrService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("Create", mock.Anything, mock.Anything).Return(dbErr)

	_, err := svc.AddAttribute(context.Background(), "et1", "attr", "", models.AttributeTypeString, "", false)
	assert.ErrorIs(t, err, dbErr)
}

// ============================================================================
// RemoveAttribute error branches
// ============================================================================

func TestRemoveAttribute_GetLatestError(t *testing.T) {
	svc, _, etvRepo, _, _ := setupAttrService()

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(nil, dbErr)

	_, err := svc.RemoveAttribute(context.Background(), "et1", "attr")
	assert.ErrorIs(t, err, dbErr)
}

func TestRemoveAttribute_EtvCreateError(t *testing.T) {
	svc, _, etvRepo, _, _ := setupAttrService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(v1, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(dbErr)

	_, err := svc.RemoveAttribute(context.Background(), "et1", "attr")
	assert.ErrorIs(t, err, dbErr)
}

func TestRemoveAttribute_AttrBulkCopyError(t *testing.T) {
	svc, attrRepo, etvRepo, _, _ := setupAttrService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(v1, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(dbErr)

	_, err := svc.RemoveAttribute(context.Background(), "et1", "attr")
	assert.ErrorIs(t, err, dbErr)
}

func TestRemoveAttribute_AssocBulkCopyError(t *testing.T) {
	svc, attrRepo, etvRepo, assocRepo, _ := setupAttrService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(v1, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(dbErr)

	_, err := svc.RemoveAttribute(context.Background(), "et1", "attr")
	assert.ErrorIs(t, err, dbErr)
}

func TestRemoveAttribute_ListByVersionError(t *testing.T) {
	svc, attrRepo, etvRepo, assocRepo, _ := setupAttrService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(v1, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Attribute(nil), dbErr)

	_, err := svc.RemoveAttribute(context.Background(), "et1", "attr")
	assert.ErrorIs(t, err, dbErr)
}

func TestRemoveAttribute_DeleteError(t *testing.T) {
	svc, attrRepo, etvRepo, assocRepo, _ := setupAttrService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(v1, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Attribute{
		{ID: "a1", Name: "attr"},
	}, nil)
	attrRepo.On("Delete", mock.Anything, "a1").Return(dbErr)

	_, err := svc.RemoveAttribute(context.Background(), "et1", "attr")
	assert.ErrorIs(t, err, dbErr)
}

// ============================================================================
// CopyAttributesFromType error branches
// ============================================================================

func TestCopyAttributesFromType_GetSourceVersionError(t *testing.T) {
	svc, _, etvRepo, _, _ := setupAttrService()

	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "src-et", 1).Return(nil, dbErr)

	_, err := svc.CopyAttributesFromType(context.Background(), "tgt-et", "src-et", 1, []string{"a"})
	assert.ErrorIs(t, err, dbErr)
}

func TestCopyAttributesFromType_ListSourceAttrsError(t *testing.T) {
	svc, attrRepo, etvRepo, _, _ := setupAttrService()

	srcV1 := &models.EntityTypeVersion{ID: "src-v1", EntityTypeID: "src-et", Version: 1}
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "src-et", 1).Return(srcV1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "src-v1").Return([]*models.Attribute(nil), dbErr)

	_, err := svc.CopyAttributesFromType(context.Background(), "tgt-et", "src-et", 1, []string{"a"})
	assert.ErrorIs(t, err, dbErr)
}

func TestCopyAttributesFromType_GetTargetLatestError(t *testing.T) {
	svc, attrRepo, etvRepo, _, _ := setupAttrService()

	srcV1 := &models.EntityTypeVersion{ID: "src-v1", EntityTypeID: "src-et", Version: 1}
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "src-et", 1).Return(srcV1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "src-v1").Return([]*models.Attribute{
		{Name: "a", Type: models.AttributeTypeString},
	}, nil)
	etvRepo.On("GetLatestByEntityType", mock.Anything, "tgt-et").Return(nil, dbErr)

	_, err := svc.CopyAttributesFromType(context.Background(), "tgt-et", "src-et", 1, []string{"a"})
	assert.ErrorIs(t, err, dbErr)
}

func TestCopyAttributesFromType_ListTargetAttrsError(t *testing.T) {
	svc, attrRepo, etvRepo, _, _ := setupAttrService()

	srcV1 := &models.EntityTypeVersion{ID: "src-v1", EntityTypeID: "src-et", Version: 1}
	tgtV1 := &models.EntityTypeVersion{ID: "tgt-v1", EntityTypeID: "tgt-et", Version: 1}
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "src-et", 1).Return(srcV1, nil)
	etvRepo.On("GetLatestByEntityType", mock.Anything, "tgt-et").Return(tgtV1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "src-v1").Return([]*models.Attribute{
		{Name: "a", Type: models.AttributeTypeString},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "tgt-v1").Return([]*models.Attribute(nil), dbErr)

	_, err := svc.CopyAttributesFromType(context.Background(), "tgt-et", "src-et", 1, []string{"a"})
	assert.ErrorIs(t, err, dbErr)
}

func TestCopyAttributesFromType_EtvCreateError(t *testing.T) {
	svc, attrRepo, etvRepo, _, _ := setupAttrService()

	srcV1 := &models.EntityTypeVersion{ID: "src-v1", EntityTypeID: "src-et", Version: 1}
	tgtV1 := &models.EntityTypeVersion{ID: "tgt-v1", EntityTypeID: "tgt-et", Version: 1}
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "src-et", 1).Return(srcV1, nil)
	etvRepo.On("GetLatestByEntityType", mock.Anything, "tgt-et").Return(tgtV1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "src-v1").Return([]*models.Attribute{
		{Name: "a", Type: models.AttributeTypeString},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "tgt-v1").Return([]*models.Attribute{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(dbErr)

	_, err := svc.CopyAttributesFromType(context.Background(), "tgt-et", "src-et", 1, []string{"a"})
	assert.ErrorIs(t, err, dbErr)
}

func TestCopyAttributesFromType_AttrBulkCopyError(t *testing.T) {
	svc, attrRepo, etvRepo, _, _ := setupAttrService()

	srcV1 := &models.EntityTypeVersion{ID: "src-v1", EntityTypeID: "src-et", Version: 1}
	tgtV1 := &models.EntityTypeVersion{ID: "tgt-v1", EntityTypeID: "tgt-et", Version: 1}
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "src-et", 1).Return(srcV1, nil)
	etvRepo.On("GetLatestByEntityType", mock.Anything, "tgt-et").Return(tgtV1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "src-v1").Return([]*models.Attribute{
		{Name: "a", Type: models.AttributeTypeString},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "tgt-v1").Return([]*models.Attribute{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(dbErr)

	_, err := svc.CopyAttributesFromType(context.Background(), "tgt-et", "src-et", 1, []string{"a"})
	assert.ErrorIs(t, err, dbErr)
}

func TestCopyAttributesFromType_AssocBulkCopyError(t *testing.T) {
	svc, attrRepo, etvRepo, assocRepo, _ := setupAttrService()

	srcV1 := &models.EntityTypeVersion{ID: "src-v1", EntityTypeID: "src-et", Version: 1}
	tgtV1 := &models.EntityTypeVersion{ID: "tgt-v1", EntityTypeID: "tgt-et", Version: 1}
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "src-et", 1).Return(srcV1, nil)
	etvRepo.On("GetLatestByEntityType", mock.Anything, "tgt-et").Return(tgtV1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "src-v1").Return([]*models.Attribute{
		{Name: "a", Type: models.AttributeTypeString},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "tgt-v1").Return([]*models.Attribute{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(dbErr)

	_, err := svc.CopyAttributesFromType(context.Background(), "tgt-et", "src-et", 1, []string{"a"})
	assert.ErrorIs(t, err, dbErr)
}

func TestCopyAttributesFromType_AttrCreateError(t *testing.T) {
	svc, attrRepo, etvRepo, assocRepo, _ := setupAttrService()

	srcV1 := &models.EntityTypeVersion{ID: "src-v1", EntityTypeID: "src-et", Version: 1}
	tgtV1 := &models.EntityTypeVersion{ID: "tgt-v1", EntityTypeID: "tgt-et", Version: 1}
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "src-et", 1).Return(srcV1, nil)
	etvRepo.On("GetLatestByEntityType", mock.Anything, "tgt-et").Return(tgtV1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "src-v1").Return([]*models.Attribute{
		{Name: "a", Type: models.AttributeTypeString},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "tgt-v1").Return([]*models.Attribute{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("Create", mock.Anything, mock.Anything).Return(dbErr)

	_, err := svc.CopyAttributesFromType(context.Background(), "tgt-et", "src-et", 1, []string{"a"})
	assert.ErrorIs(t, err, dbErr)
}

// ============================================================================
// ReorderAttributes error branches
// ============================================================================

func TestReorderAttributes_GetLatestError(t *testing.T) {
	svc, _, etvRepo, _, _ := setupAttrService()

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(nil, dbErr)

	err := svc.ReorderAttributes(context.Background(), "et1", []string{"a", "b"})
	assert.ErrorIs(t, err, dbErr)
}

func TestReorderAttributes_ReorderError(t *testing.T) {
	svc, attrRepo, etvRepo, _, _ := setupAttrService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(v1, nil)
	attrRepo.On("Reorder", mock.Anything, "v1", []string{"a", "b"}).Return(dbErr)

	err := svc.ReorderAttributes(context.Background(), "et1", []string{"a", "b"})
	assert.ErrorIs(t, err, dbErr)
}

// ============================================================================
// UpdateEntityType error branches
// ============================================================================

func TestUpdateEntityType_GetLatestError(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, nil, nil)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-id").Return(nil, dbErr)

	_, err := svc.UpdateEntityType(context.Background(), "et-id", "V2")
	assert.ErrorIs(t, err, dbErr)
}

func TestUpdateEntityType_EtvCreateError(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, nil, nil)

	v1 := &models.EntityTypeVersion{ID: "v1-id", EntityTypeID: "et-id", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-id").Return(v1, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(dbErr)

	_, err := svc.UpdateEntityType(context.Background(), "et-id", "V2")
	assert.ErrorIs(t, err, dbErr)
}

func TestUpdateEntityType_AttrBulkCopyError(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, attrRepo, nil)

	v1 := &models.EntityTypeVersion{ID: "v1-id", EntityTypeID: "et-id", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-id").Return(v1, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(dbErr)

	_, err := svc.UpdateEntityType(context.Background(), "et-id", "V2")
	assert.ErrorIs(t, err, dbErr)
}

func TestUpdateEntityType_AssocBulkCopyError(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, attrRepo, assocRepo)

	v1 := &models.EntityTypeVersion{ID: "v1-id", EntityTypeID: "et-id", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-id").Return(v1, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(dbErr)

	_, err := svc.UpdateEntityType(context.Background(), "et-id", "V2")
	assert.ErrorIs(t, err, dbErr)
}

func TestUpdateEntityType_GetByIDError(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, attrRepo, assocRepo)

	v1 := &models.EntityTypeVersion{ID: "v1-id", EntityTypeID: "et-id", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-id").Return(v1, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	etRepo.On("GetByID", mock.Anything, "et-id").Return(nil, dbErr)

	_, err := svc.UpdateEntityType(context.Background(), "et-id", "V2")
	assert.ErrorIs(t, err, dbErr)
}

func TestUpdateEntityType_UpdateError(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, attrRepo, assocRepo)

	v1 := &models.EntityTypeVersion{ID: "v1-id", EntityTypeID: "et-id", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-id").Return(v1, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	etRepo.On("GetByID", mock.Anything, "et-id").Return(&models.EntityType{ID: "et-id", Name: "Model"}, nil)
	etRepo.On("Update", mock.Anything, mock.Anything).Return(dbErr)

	_, err := svc.UpdateEntityType(context.Background(), "et-id", "V2")
	assert.ErrorIs(t, err, dbErr)
}

// ============================================================================
// CopyEntityType error branches
// ============================================================================

func TestCopyEntityType_EmptyName(t *testing.T) {
	svc := meta.NewEntityTypeService(nil, nil, nil, nil)

	_, _, err := svc.CopyEntityType(context.Background(), "src-et", 1, "")
	assert.Error(t, err)
}

func TestCopyEntityType_GetSourceVersionError(t *testing.T) {
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := meta.NewEntityTypeService(nil, etvRepo, nil, nil)

	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "src-et", 1).Return(nil, dbErr)

	_, _, err := svc.CopyEntityType(context.Background(), "src-et", 1, "NewType")
	assert.ErrorIs(t, err, dbErr)
}

func TestCopyEntityType_EtCreateError(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, nil, nil)

	srcETV := &models.EntityTypeVersion{ID: "src-v1", EntityTypeID: "src-et", Version: 1}
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "src-et", 1).Return(srcETV, nil)
	etRepo.On("Create", mock.Anything, mock.Anything).Return(dbErr)

	_, _, err := svc.CopyEntityType(context.Background(), "src-et", 1, "NewType")
	assert.ErrorIs(t, err, dbErr)
}

func TestCopyEntityType_EtvCreateError(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, nil, nil)

	srcETV := &models.EntityTypeVersion{ID: "src-v1", EntityTypeID: "src-et", Version: 1}
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "src-et", 1).Return(srcETV, nil)
	etRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(dbErr)

	_, _, err := svc.CopyEntityType(context.Background(), "src-et", 1, "NewType")
	assert.ErrorIs(t, err, dbErr)
}

func TestCopyEntityType_AttrBulkCopyError(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, attrRepo, nil)

	srcETV := &models.EntityTypeVersion{ID: "src-v1", EntityTypeID: "src-et", Version: 1}
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "src-et", 1).Return(srcETV, nil)
	etRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(dbErr)

	_, _, err := svc.CopyEntityType(context.Background(), "src-et", 1, "NewType")
	assert.ErrorIs(t, err, dbErr)
}

// ============================================================================
// CreateEntityType error branches
// ============================================================================

func TestCreateEntityType_EtvCreateError(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, nil, nil)

	etRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(dbErr)

	_, _, err := svc.CreateEntityType(context.Background(), "Model", "desc")
	assert.ErrorIs(t, err, dbErr)
}

// ============================================================================
// CreateCatalogVersion error branches
// ============================================================================

func TestCreateCatalogVersion_CvCreateError(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil)

	cvRepo.On("Create", mock.Anything, mock.Anything).Return(dbErr)

	_, err := svc.CreateCatalogVersion(context.Background(), "v1.0", nil)
	assert.ErrorIs(t, err, dbErr)
}

func TestCreateCatalogVersion_PinCreateError(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, nil, nil)

	cvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	pinRepo.On("Create", mock.Anything, mock.Anything).Return(dbErr)

	pins := []models.CatalogVersionPin{{EntityTypeVersionID: "etv1"}}
	_, err := svc.CreateCatalogVersion(context.Background(), "v1.0", pins)
	assert.ErrorIs(t, err, dbErr)
}

func TestCreateCatalogVersion_LtCreateError(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	ltRepo := new(mocks.MockLifecycleTransitionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, ltRepo, nil, "", nil, nil, nil)

	cvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	pinRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	ltRepo.On("Create", mock.Anything, mock.Anything).Return(dbErr)

	pins := []models.CatalogVersionPin{{EntityTypeVersionID: "etv1"}}
	_, err := svc.CreateCatalogVersion(context.Background(), "v1.0", pins)
	assert.ErrorIs(t, err, dbErr)
}

func TestCreateCatalogVersion_NoPins_LtCreateError(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	ltRepo := new(mocks.MockLifecycleTransitionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, ltRepo, nil, "", nil, nil, nil)

	cvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	ltRepo.On("Create", mock.Anything, mock.Anything).Return(dbErr)

	_, err := svc.CreateCatalogVersion(context.Background(), "v1.0", nil)
	assert.ErrorIs(t, err, dbErr)
}

// ============================================================================
// CreateEnum error branches
// ============================================================================

func TestCreateEnum_EmptyName(t *testing.T) {
	svc := meta.NewEnumService(nil, nil, nil)

	_, err := svc.CreateEnum(context.Background(), "", []string{"a", "b"})
	assert.Error(t, err)
}

func TestCreateEnum_EnumCreateError(t *testing.T) {
	enumRepo := new(mocks.MockEnumRepo)
	svc := meta.NewEnumService(enumRepo, nil, nil)

	enumRepo.On("Create", mock.Anything, mock.Anything).Return(dbErr)

	_, err := svc.CreateEnum(context.Background(), "Status", []string{"active"})
	assert.ErrorIs(t, err, dbErr)
}

func TestCreateEnum_ValueCreateError(t *testing.T) {
	enumRepo := new(mocks.MockEnumRepo)
	evRepo := new(mocks.MockEnumValueRepo)
	svc := meta.NewEnumService(enumRepo, evRepo, nil)

	enumRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	evRepo.On("Create", mock.Anything, mock.Anything).Return(dbErr)

	_, err := svc.CreateEnum(context.Background(), "Status", []string{"active", "inactive"})
	assert.ErrorIs(t, err, dbErr)
}

// ============================================================================
// Promote/Demote error branches
// ============================================================================

func TestPromote_GetByIDError(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(nil, dbErr)

	err := svc.Promote(context.Background(), "cv1", meta.RoleAdmin, "admin")
	assert.ErrorIs(t, err, dbErr)
}

func TestPromote_UpdateLifecycleError(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	cvRepo.On("UpdateLifecycle", mock.Anything, "cv1", models.LifecycleStageTesting).Return(dbErr)

	err := svc.Promote(context.Background(), "cv1", meta.RoleRW, "user")
	assert.ErrorIs(t, err, dbErr)
}

func TestPromote_LtCreateError(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	ltRepo := new(mocks.MockLifecycleTransitionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, ltRepo, nil, "", nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	cvRepo.On("UpdateLifecycle", mock.Anything, "cv1", models.LifecycleStageTesting).Return(nil)
	ltRepo.On("Create", mock.Anything, mock.Anything).Return(dbErr)

	err := svc.Promote(context.Background(), "cv1", meta.RoleRW, "user")
	assert.ErrorIs(t, err, dbErr)
}

func TestDemote_GetByIDError(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(nil, dbErr)

	err := svc.Demote(context.Background(), "cv1", meta.RoleSuperAdmin, "sa", models.LifecycleStageDevelopment)
	assert.ErrorIs(t, err, dbErr)
}

func TestDemote_UpdateLifecycleError(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageTesting,
	}, nil)
	cvRepo.On("UpdateLifecycle", mock.Anything, "cv1", models.LifecycleStageDevelopment).Return(dbErr)

	err := svc.Demote(context.Background(), "cv1", meta.RoleRW, "user", models.LifecycleStageDevelopment)
	assert.ErrorIs(t, err, dbErr)
}

func TestDemote_LtCreateError(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	ltRepo := new(mocks.MockLifecycleTransitionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, ltRepo, nil, "", nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageTesting,
	}, nil)
	cvRepo.On("UpdateLifecycle", mock.Anything, "cv1", models.LifecycleStageDevelopment).Return(nil)
	ltRepo.On("Create", mock.Anything, mock.Anything).Return(dbErr)

	err := svc.Demote(context.Background(), "cv1", meta.RoleRW, "user", models.LifecycleStageDevelopment)
	assert.ErrorIs(t, err, dbErr)
}

// ============================================================================
// EnumService additional error branches
// ============================================================================

func TestUpdateEnum_GetByIDError(t *testing.T) {
	enumRepo := new(mocks.MockEnumRepo)
	svc := meta.NewEnumService(enumRepo, nil, nil)

	enumRepo.On("GetByID", mock.Anything, "e1").Return(nil, dbErr)

	err := svc.UpdateEnum(context.Background(), "e1", "Updated")
	assert.ErrorIs(t, err, dbErr)
}

func TestUpdateEnum_UpdateError(t *testing.T) {
	enumRepo := new(mocks.MockEnumRepo)
	svc := meta.NewEnumService(enumRepo, nil, nil)

	enumRepo.On("GetByID", mock.Anything, "e1").Return(&models.Enum{ID: "e1", Name: "Old"}, nil)
	enumRepo.On("Update", mock.Anything, mock.Anything).Return(dbErr)

	err := svc.UpdateEnum(context.Background(), "e1", "Updated")
	assert.ErrorIs(t, err, dbErr)
}

func TestAddValue_ListByEnumError(t *testing.T) {
	evRepo := new(mocks.MockEnumValueRepo)
	svc := meta.NewEnumService(nil, evRepo, nil)

	evRepo.On("ListByEnum", mock.Anything, "e1").Return([]*models.EnumValue(nil), dbErr)

	err := svc.AddValue(context.Background(), "e1", "newval")
	assert.ErrorIs(t, err, dbErr)
}

func TestAddValue_CreateError(t *testing.T) {
	evRepo := new(mocks.MockEnumValueRepo)
	svc := meta.NewEnumService(nil, evRepo, nil)

	evRepo.On("ListByEnum", mock.Anything, "e1").Return([]*models.EnumValue{}, nil)
	evRepo.On("Create", mock.Anything, mock.Anything).Return(dbErr)

	err := svc.AddValue(context.Background(), "e1", "newval")
	assert.ErrorIs(t, err, dbErr)
}

// ============================================================================
// VersionHistoryService error branches
// ============================================================================

func TestCompareVersions_GetV1Error(t *testing.T) {
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := meta.NewVersionHistoryService(etvRepo, nil, nil)

	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 1).Return(nil, dbErr)

	_, err := svc.CompareVersions(context.Background(), "et1", 1, 2)
	assert.ErrorIs(t, err, dbErr)
}

func TestCompareVersions_GetV2Error(t *testing.T) {
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := meta.NewVersionHistoryService(etvRepo, nil, nil)

	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 1).Return(&models.EntityTypeVersion{ID: "v1"}, nil)
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 2).Return(nil, dbErr)

	_, err := svc.CompareVersions(context.Background(), "et1", 1, 2)
	assert.ErrorIs(t, err, dbErr)
}

func TestCompareVersions_ListAttrsV1Error(t *testing.T) {
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	svc := meta.NewVersionHistoryService(etvRepo, attrRepo, nil)

	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 1).Return(&models.EntityTypeVersion{ID: "v1"}, nil)
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 2).Return(&models.EntityTypeVersion{ID: "v2"}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute(nil), dbErr)

	_, err := svc.CompareVersions(context.Background(), "et1", 1, 2)
	assert.ErrorIs(t, err, dbErr)
}

func TestCompareVersions_ListAttrsV2Error(t *testing.T) {
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	svc := meta.NewVersionHistoryService(etvRepo, attrRepo, nil)

	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 1).Return(&models.EntityTypeVersion{ID: "v1"}, nil)
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 2).Return(&models.EntityTypeVersion{ID: "v2"}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v2").Return([]*models.Attribute(nil), dbErr)

	_, err := svc.CompareVersions(context.Background(), "et1", 1, 2)
	assert.ErrorIs(t, err, dbErr)
}

func TestCompareVersions_ListAssocsV1Error(t *testing.T) {
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewVersionHistoryService(etvRepo, attrRepo, assocRepo)

	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 1).Return(&models.EntityTypeVersion{ID: "v1"}, nil)
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 2).Return(&models.EntityTypeVersion{ID: "v2"}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v2").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association(nil), dbErr)

	_, err := svc.CompareVersions(context.Background(), "et1", 1, 2)
	assert.ErrorIs(t, err, dbErr)
}

func TestCompareVersions_ListAssocsV2Error(t *testing.T) {
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewVersionHistoryService(etvRepo, attrRepo, assocRepo)

	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 1).Return(&models.EntityTypeVersion{ID: "v1"}, nil)
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 2).Return(&models.EntityTypeVersion{ID: "v2"}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v2").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v2").Return([]*models.Association(nil), dbErr)

	_, err := svc.CompareVersions(context.Background(), "et1", 1, 2)
	assert.ErrorIs(t, err, dbErr)
}

// ============================================================================
// ListAssociations error branches
// ============================================================================

func TestListAssociations_GetLatestError(t *testing.T) {
	assocRepo := new(mocks.MockAssociationRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := meta.NewAssociationService(assocRepo, etvRepo, nil)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(nil, dbErr)

	_, err := svc.ListAssociations(context.Background(), "et1")
	assert.ErrorIs(t, err, dbErr)
}

func TestListAssociations_ListByVersionError(t *testing.T) {
	assocRepo := new(mocks.MockAssociationRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := meta.NewAssociationService(assocRepo, etvRepo, nil)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(
		&models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association(nil), dbErr)

	_, err := svc.ListAssociations(context.Background(), "et1")
	assert.ErrorIs(t, err, dbErr)
}

// ============================================================================
// EditAssociation error branches
// ============================================================================

func TestEditAssociation_GetLatestError(t *testing.T) {
	svc, _, etvRepo, _ := setupAssocService()
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(nil, dbErr)
	_, err := svc.EditAssociation(context.Background(), "et-a", "test", nil, nil, nil, nil, nil, nil)
	assert.ErrorIs(t, err, dbErr)
}

func TestEditAssociation_ListByVersionError(t *testing.T) {
	svc, assocRepo, etvRepo, _ := setupAssocService()
	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-a", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(v1, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return(([]*models.Association)(nil), dbErr)
	_, err := svc.EditAssociation(context.Background(), "et-a", "test", nil, nil, nil, nil, nil, nil)
	assert.ErrorIs(t, err, dbErr)
}

func TestEditAssociation_TargetCardinalityInvalid(t *testing.T) {
	svc, assocRepo, etvRepo, _ := setupAssocService()
	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-a", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(v1, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{
		{ID: "a1", Name: "test", Type: models.AssociationTypeDirectional},
	}, nil)
	_, err := svc.EditAssociation(context.Background(), "et-a", "test", nil, nil, nil, nil, strPtr("bad"), nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "target_cardinality")
}

func TestEditAssociation_EtvCreateError(t *testing.T) {
	svc, assocRepo, etvRepo, attrRepo := setupAssocService()
	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-a", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(v1, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{
		{ID: "a1", Name: "test", Type: models.AssociationTypeDirectional},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(dbErr)
	_, err := svc.EditAssociation(context.Background(), "et-a", "test", nil, strPtr("role"), nil, nil, nil, nil)
	assert.ErrorIs(t, err, dbErr)
}

func TestEditAssociation_AttrBulkCopyError(t *testing.T) {
	svc, assocRepo, etvRepo, attrRepo := setupAssocService()
	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-a", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(v1, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{
		{ID: "a1", Name: "test", Type: models.AssociationTypeDirectional},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(dbErr)
	_, err := svc.EditAssociation(context.Background(), "et-a", "test", nil, strPtr("role"), nil, nil, nil, nil)
	assert.ErrorIs(t, err, dbErr)
}

func TestEditAssociation_AssocBulkCopyError(t *testing.T) {
	svc, assocRepo, etvRepo, attrRepo := setupAssocService()
	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-a", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(v1, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{
		{ID: "a1", Name: "test", Type: models.AssociationTypeDirectional},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(dbErr)
	_, err := svc.EditAssociation(context.Background(), "et-a", "test", nil, strPtr("role"), nil, nil, nil, nil)
	assert.ErrorIs(t, err, dbErr)
}

func TestEditAssociation_ListNewVersionError(t *testing.T) {
	svc, assocRepo, etvRepo, attrRepo := setupAssocService()
	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-a", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(v1, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{
		{ID: "a1", Name: "test", Type: models.AssociationTypeDirectional},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("ListByVersion", mock.Anything, mock.MatchedBy(func(id string) bool { return id != "v1" })).Return(([]*models.Association)(nil), dbErr)
	_, err := svc.EditAssociation(context.Background(), "et-a", "test", nil, strPtr("role"), nil, nil, nil, nil)
	assert.ErrorIs(t, err, dbErr)
}

func TestEditAssociation_UpdateError(t *testing.T) {
	svc, assocRepo, etvRepo, attrRepo := setupAssocService()
	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-a", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(v1, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{
		{ID: "a1", Name: "test", Type: models.AssociationTypeDirectional},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("ListByVersion", mock.Anything, mock.MatchedBy(func(id string) bool { return id != "v1" })).Return([]*models.Association{
		{ID: "a1-copy", Name: "test", Type: models.AssociationTypeDirectional},
	}, nil)
	assocRepo.On("Update", mock.Anything, mock.Anything).Return(dbErr)
	_, err := svc.EditAssociation(context.Background(), "et-a", "test", nil, strPtr("role"), nil, nil, nil, nil)
	assert.ErrorIs(t, err, dbErr)
}

// ============================================================================
// checkNameConflict error branches
// ============================================================================

func TestCheckNameConflict_AttrListError(t *testing.T) {
	svc, _, etvRepo, attrRepo := setupAssocService()
	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-a", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return(([]*models.Attribute)(nil), dbErr)
	_, err := svc.CreateAssociation(context.Background(), "et-a", "et-b", models.AssociationTypeDirectional, "test", "", "", "", "")
	assert.ErrorIs(t, err, dbErr)
}

func TestCheckNameConflict_AssocListError(t *testing.T) {
	svc, assocRepo, etvRepo, attrRepo := setupAssocService()
	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-a", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return(([]*models.Association)(nil), dbErr)
	_, err := svc.CreateAssociation(context.Background(), "et-a", "et-b", models.AssociationTypeDirectional, "test", "", "", "", "")
	assert.ErrorIs(t, err, dbErr)
}
