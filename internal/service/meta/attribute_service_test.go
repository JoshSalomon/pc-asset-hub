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
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, "v1", mock.AnythingOfType("string")).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, "v1", mock.AnythingOfType("string")).Return(nil)
	attrRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Attribute")).Return(nil)

	newVer, err := svc.AddAttribute(context.Background(), "et1", "endpoint", "API endpoint", models.AttributeTypeString, "", false)
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

	_, err := svc.AddAttribute(context.Background(), "et1", "endpoint", "", models.AttributeTypeString, "", false)
	assert.True(t, domainerrors.IsConflict(err))
}

func TestT3_13_AddAttributeEnumValid(t *testing.T) {
	svc, attrRepo, etvRepo, assocRepo, enumRepo := setupAttrService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	enumRepo.On("GetByID", mock.Anything, "enum1").Return(&models.Enum{ID: "enum1", Name: "Status"}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	_, err := svc.AddAttribute(context.Background(), "et1", "status", "", models.AttributeTypeEnum, "enum1", false)
	assert.NoError(t, err)
}

func TestT3_14_AddAttributeEnumInvalid(t *testing.T) {
	svc, attrRepo, etvRepo, _, enumRepo := setupAttrService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	enumRepo.On("GetByID", mock.Anything, "bad-enum").Return(nil, domainerrors.NewNotFound("Enum", "bad-enum"))

	_, err := svc.AddAttribute(context.Background(), "et1", "status", "", models.AttributeTypeEnum, "bad-enum", false)
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

// CopyAttributesFromType preserves Required flag
func TestCopyAttributesPreservesRequired(t *testing.T) {
	svc, attrRepo, etvRepo, assocRepo, _ := setupAttrService()

	srcV1 := &models.EntityTypeVersion{ID: "src-v1", EntityTypeID: "src-et", Version: 1}
	tgtV1 := &models.EntityTypeVersion{ID: "tgt-v1", EntityTypeID: "tgt-et", Version: 1}

	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "src-et", 1).Return(srcV1, nil)
	etvRepo.On("GetLatestByEntityType", mock.Anything, "tgt-et").Return(tgtV1, nil)

	attrRepo.On("ListByVersion", mock.Anything, "src-v1").Return([]*models.Attribute{
		{Name: "hostname", Type: models.AttributeTypeString, Required: true},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "tgt-v1").Return([]*models.Attribute{}, nil)

	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, "tgt-v1", mock.AnythingOfType("string")).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, "tgt-v1", mock.AnythingOfType("string")).Return(nil)
	// The copied attribute MUST have Required=true
	attrRepo.On("Create", mock.Anything, mock.MatchedBy(func(a *models.Attribute) bool {
		return a.Name == "hostname" && a.Required == true
	})).Return(nil)

	_, err := svc.CopyAttributesFromType(context.Background(), "tgt-et", "src-et", 1, []string{"hostname"})
	require.NoError(t, err)
	// Verify the Create was called with Required=true
	attrRepo.AssertCalled(t, "Create", mock.Anything, mock.MatchedBy(func(a *models.Attribute) bool {
		return a.Required == true
	}))
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

// === EditAttribute Tests (T-E.01 through T-E.07) ===

func TestTE01_EditAttributeChangesName(t *testing.T) {
	svc, attrRepo, etvRepo, assocRepo, _ := setupAttrService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{
		{ID: "attr1", Name: "old_name", Description: "desc", Type: models.AttributeTypeString, Ordinal: 0},
		{ID: "attr2", Name: "other", Description: "other desc", Type: models.AttributeTypeNumber, Ordinal: 1},
	}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, "v1", mock.AnythingOfType("string")).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, "v1", mock.AnythingOfType("string")).Return(nil)
	// After BulkCopy, ListByVersion on the new version returns the copied attrs
	attrRepo.On("ListByVersion", mock.Anything, mock.MatchedBy(func(id string) bool { return id != "v1" })).Return([]*models.Attribute{
		{ID: "attr1-copy", Name: "old_name", Description: "desc", Type: models.AttributeTypeString, Ordinal: 0},
		{ID: "attr2-copy", Name: "other", Description: "other desc", Type: models.AttributeTypeNumber, Ordinal: 1},
	}, nil)
	attrRepo.On("Update", mock.Anything, mock.MatchedBy(func(a *models.Attribute) bool {
		return a.Name == "new_name"
	})).Return(nil)

	newVer, err := svc.EditAttribute(context.Background(), "et1", "old_name", strPtr("new_name"), nil, nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 2, newVer.Version)
	attrRepo.AssertCalled(t, "Update", mock.Anything, mock.MatchedBy(func(a *models.Attribute) bool {
		return a.Name == "new_name" && a.Description == "desc"
	}))
}

func TestTE02_EditAttributeChangesDescription(t *testing.T) {
	svc, attrRepo, etvRepo, assocRepo, _ := setupAttrService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{
		{ID: "attr1", Name: "endpoint", Description: "old desc", Type: models.AttributeTypeString, Ordinal: 0},
	}, nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, "v1", mock.AnythingOfType("string")).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, "v1", mock.AnythingOfType("string")).Return(nil)
	attrRepo.On("ListByVersion", mock.Anything, mock.MatchedBy(func(id string) bool { return id != "v1" })).Return([]*models.Attribute{
		{ID: "attr1-copy", Name: "endpoint", Description: "old desc", Type: models.AttributeTypeString, Ordinal: 0},
	}, nil)
	attrRepo.On("Update", mock.Anything, mock.MatchedBy(func(a *models.Attribute) bool {
		return a.Description == "new desc"
	})).Return(nil)

	newVer, err := svc.EditAttribute(context.Background(), "et1", "endpoint", nil, strPtr("new desc"), nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 2, newVer.Version)
}

func TestTE03_EditAttributeChangesTypeToEnum(t *testing.T) {
	svc, attrRepo, etvRepo, assocRepo, enumRepo := setupAttrService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{
		{ID: "attr1", Name: "status", Description: "desc", Type: models.AttributeTypeString, Ordinal: 0},
	}, nil)
	enumRepo.On("GetByID", mock.Anything, "enum1").Return(&models.Enum{ID: "enum1", Name: "Status"}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("ListByVersion", mock.Anything, mock.MatchedBy(func(id string) bool { return id != "v1" })).Return([]*models.Attribute{
		{ID: "attr1-copy", Name: "status", Description: "desc", Type: models.AttributeTypeString, Ordinal: 0},
	}, nil)
	enumType := models.AttributeTypeEnum
	enumID := "enum1"
	attrRepo.On("Update", mock.Anything, mock.MatchedBy(func(a *models.Attribute) bool {
		return a.Type == models.AttributeTypeEnum && a.EnumID == "enum1"
	})).Return(nil)

	newVer, err := svc.EditAttribute(context.Background(), "et1", "status", nil, nil, &enumType, &enumID, nil)
	require.NoError(t, err)
	assert.Equal(t, 2, newVer.Version)
}

func TestTE04_EditAttributeNameConflict(t *testing.T) {
	svc, attrRepo, etvRepo, _, _ := setupAttrService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{
		{ID: "attr1", Name: "endpoint", Type: models.AttributeTypeString, Ordinal: 0},
		{ID: "attr2", Name: "conflict_name", Type: models.AttributeTypeString, Ordinal: 1},
	}, nil)

	_, err := svc.EditAttribute(context.Background(), "et1", "endpoint", strPtr("conflict_name"), nil, nil, nil, nil)
	assert.True(t, domainerrors.IsConflict(err))
}

func TestTE05_EditAttributeNotFound(t *testing.T) {
	svc, attrRepo, etvRepo, _, _ := setupAttrService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{
		{ID: "attr1", Name: "endpoint", Type: models.AttributeTypeString},
	}, nil)

	_, err := svc.EditAttribute(context.Background(), "et1", "nonexistent", strPtr("new_name"), nil, nil, nil, nil)
	assert.True(t, domainerrors.IsNotFound(err))
}

func TestTE06_EditAttributeEnumTypeMissingEnumID(t *testing.T) {
	svc, attrRepo, etvRepo, _, _ := setupAttrService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{
		{ID: "attr1", Name: "status", Type: models.AttributeTypeString, Ordinal: 0},
	}, nil)

	enumType := models.AttributeTypeEnum
	_, err := svc.EditAttribute(context.Background(), "et1", "status", nil, nil, &enumType, nil, nil)
	assert.True(t, domainerrors.IsValidation(err))
}

func TestTE07_EditAttributePreservesOrdinal(t *testing.T) {
	svc, attrRepo, etvRepo, assocRepo, _ := setupAttrService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{
		{ID: "attr1", Name: "first", Type: models.AttributeTypeString, Ordinal: 0},
		{ID: "attr2", Name: "second", Type: models.AttributeTypeString, Ordinal: 1},
		{ID: "attr3", Name: "third", Type: models.AttributeTypeString, Ordinal: 2},
	}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("ListByVersion", mock.Anything, mock.MatchedBy(func(id string) bool { return id != "v1" })).Return([]*models.Attribute{
		{ID: "attr1-copy", Name: "first", Type: models.AttributeTypeString, Ordinal: 0},
		{ID: "attr2-copy", Name: "second", Type: models.AttributeTypeString, Ordinal: 1},
		{ID: "attr3-copy", Name: "third", Type: models.AttributeTypeString, Ordinal: 2},
	}, nil)
	attrRepo.On("Update", mock.Anything, mock.MatchedBy(func(a *models.Attribute) bool {
		return a.Ordinal == 1 && a.Name == "renamed_second"
	})).Return(nil)

	newVer, err := svc.EditAttribute(context.Background(), "et1", "second", strPtr("renamed_second"), nil, nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 2, newVer.Version)
}

// === EditAttribute Error Path Tests ===

func TestEditAttribute_GetLatestError(t *testing.T) {
	svc, _, etvRepo, _, _ := setupAttrService()

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(nil, domainerrors.NewNotFound("EntityTypeVersion", "et1"))

	_, err := svc.EditAttribute(context.Background(), "et1", "attr", nil, strPtr("new desc"), nil, nil, nil)
	assert.True(t, domainerrors.IsNotFound(err))
}

func TestEditAttribute_ListByVersionError(t *testing.T) {
	svc, attrRepo, etvRepo, _, _ := setupAttrService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return(([]*models.Attribute)(nil), domainerrors.NewNotFound("Attribute", "v1"))

	_, err := svc.EditAttribute(context.Background(), "et1", "attr", nil, strPtr("new desc"), nil, nil, nil)
	assert.True(t, domainerrors.IsNotFound(err))
}

func TestEditAttribute_EtvCreateError(t *testing.T) {
	svc, attrRepo, etvRepo, assocRepo, _ := setupAttrService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{
		{ID: "attr1", Name: "endpoint", Type: models.AttributeTypeString, Ordinal: 0},
	}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(domainerrors.NewConflict("EntityTypeVersion", "create failed"))

	_, err := svc.EditAttribute(context.Background(), "et1", "endpoint", strPtr("new_name"), nil, nil, nil, nil)
	assert.True(t, domainerrors.IsConflict(err))
}

func TestEditAttribute_BulkCopyAttrError(t *testing.T) {
	svc, attrRepo, etvRepo, assocRepo, _ := setupAttrService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{
		{ID: "attr1", Name: "endpoint", Type: models.AttributeTypeString, Ordinal: 0},
	}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, "v1", mock.AnythingOfType("string")).Return(domainerrors.NewNotFound("Attribute", "bulk copy failed"))

	_, err := svc.EditAttribute(context.Background(), "et1", "endpoint", strPtr("new_name"), nil, nil, nil, nil)
	assert.True(t, domainerrors.IsNotFound(err))
}

func TestEditAttribute_BulkCopyAssocError(t *testing.T) {
	svc, attrRepo, etvRepo, assocRepo, _ := setupAttrService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{
		{ID: "attr1", Name: "endpoint", Type: models.AttributeTypeString, Ordinal: 0},
	}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, "v1", mock.AnythingOfType("string")).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, "v1", mock.AnythingOfType("string")).Return(domainerrors.NewNotFound("Association", "bulk copy failed"))

	_, err := svc.EditAttribute(context.Background(), "et1", "endpoint", strPtr("new_name"), nil, nil, nil, nil)
	assert.True(t, domainerrors.IsNotFound(err))
}

func TestEditAttribute_ListNewVersionError(t *testing.T) {
	svc, attrRepo, etvRepo, assocRepo, _ := setupAttrService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{
		{ID: "attr1", Name: "endpoint", Type: models.AttributeTypeString, Ordinal: 0},
	}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, "v1", mock.AnythingOfType("string")).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, "v1", mock.AnythingOfType("string")).Return(nil)
	attrRepo.On("ListByVersion", mock.Anything, mock.MatchedBy(func(id string) bool { return id != "v1" })).Return(([]*models.Attribute)(nil), domainerrors.NewNotFound("Attribute", "list failed"))

	_, err := svc.EditAttribute(context.Background(), "et1", "endpoint", strPtr("new_name"), nil, nil, nil, nil)
	assert.True(t, domainerrors.IsNotFound(err))
}

func TestEditAttribute_UpdateError(t *testing.T) {
	svc, attrRepo, etvRepo, assocRepo, _ := setupAttrService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{
		{ID: "attr1", Name: "endpoint", Type: models.AttributeTypeString, Ordinal: 0},
	}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, "v1", mock.AnythingOfType("string")).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, "v1", mock.AnythingOfType("string")).Return(nil)
	attrRepo.On("ListByVersion", mock.Anything, mock.MatchedBy(func(id string) bool { return id != "v1" })).Return([]*models.Attribute{
		{ID: "attr1-copy", Name: "endpoint", Type: models.AttributeTypeString, Ordinal: 0},
	}, nil)
	attrRepo.On("Update", mock.Anything, mock.Anything).Return(domainerrors.NewConflict("Attribute", "update failed"))

	_, err := svc.EditAttribute(context.Background(), "et1", "endpoint", strPtr("new_name"), nil, nil, nil, nil)
	assert.True(t, domainerrors.IsConflict(err))
}

func TestEditAttribute_EnumValidation(t *testing.T) {
	svc, _, _, _, enumRepo := setupAttrService()

	enumType := models.AttributeTypeEnum
	enumID := "bad-enum"
	enumRepo.On("GetByID", mock.Anything, "bad-enum").Return(nil, domainerrors.NewNotFound("Enum", "bad-enum"))

	_, err := svc.EditAttribute(context.Background(), "et1", "status", nil, nil, &enumType, &enumID, nil)
	assert.True(t, domainerrors.IsValidation(err))
}

// strPtr is a helper to create a pointer to a string.
func strPtr(s string) *string { return &s }

func TestT3_19_ReorderAttributes(t *testing.T) {
	svc, attrRepo, etvRepo, _, _ := setupAttrService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(v1, nil)
	attrRepo.On("Reorder", mock.Anything, "v1", []string{"c", "a", "b"}).Return(nil)

	err := svc.ReorderAttributes(context.Background(), "et1", []string{"c", "a", "b"})
	assert.NoError(t, err)
}

// T-E.109: AddAttribute rejects name that conflicts with association
func TestTE109_AddAttributeConflictsWithAssociation(t *testing.T) {
	svc, attrRepo, etvRepo, assocRepo, _ := setupAttrService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{
		{ID: "assoc1", Name: "tools", TargetEntityTypeID: "et2", Type: models.AssociationTypeContainment},
	}, nil)

	_, err := svc.AddAttribute(context.Background(), "et1", "tools", "", models.AttributeTypeString, "", false)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsConflict(err))
	assert.Contains(t, err.Error(), "association")
}

// EditAttribute rename conflicts with association name
func TestEditAttribute_RenameConflictsWithAssociation(t *testing.T) {
	svc, attrRepo, etvRepo, assocRepo, _ := setupAttrService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{
		{ID: "a1", Name: "hostname", Type: models.AttributeTypeString, Ordinal: 0},
	}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{
		{ID: "assoc1", Name: "tools", TargetEntityTypeID: "et2", Type: models.AssociationTypeContainment},
	}, nil)

	_, err := svc.EditAttribute(context.Background(), "et1", "hostname", strPtr("tools"), nil, nil, nil, nil)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsConflict(err))
	assert.Contains(t, err.Error(), "association")
}

// T-E.140: AddAttribute with required=true stores required flag
func TestTE140_AddAttributeWithRequired(t *testing.T) {
	svc, attrRepo, etvRepo, assocRepo, _ := setupAttrService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, "v1", mock.AnythingOfType("string")).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, "v1", mock.AnythingOfType("string")).Return(nil)
	attrRepo.On("Create", mock.Anything, mock.MatchedBy(func(a *models.Attribute) bool {
		return a.Required == true
	})).Return(nil)

	newVer, err := svc.AddAttribute(context.Background(), "et1", "hostname", "Host", models.AttributeTypeString, "", true)
	require.NoError(t, err)
	assert.Equal(t, 2, newVer.Version)
}

// T-E.141: EditAttribute can change required flag
func TestTE141_EditAttributeChangeRequired(t *testing.T) {
	svc, attrRepo, etvRepo, assocRepo, _ := setupAttrService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{
		{ID: "a1", Name: "hostname", Type: models.AttributeTypeString, Ordinal: 0, Required: false},
	}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, "v1", mock.AnythingOfType("string")).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, "v1", mock.AnythingOfType("string")).Return(nil)
	attrRepo.On("ListByVersion", mock.Anything, mock.MatchedBy(func(id string) bool { return id != "v1" })).Return([]*models.Attribute{
		{ID: "a1-copy", Name: "hostname", Type: models.AttributeTypeString, Ordinal: 0, Required: false},
	}, nil)
	reqTrue := true
	attrRepo.On("Update", mock.Anything, mock.MatchedBy(func(a *models.Attribute) bool {
		return a.Required == true
	})).Return(nil)

	newVer, err := svc.EditAttribute(context.Background(), "et1", "hostname", nil, nil, nil, nil, &reqTrue)
	require.NoError(t, err)
	assert.Equal(t, 2, newVer.Version)
}

// === TD-22: System Attributes — Copy Attributes Exclusion ===

// T-18.19: CopyAttributes with "name" in list silently skips it
func TestT18_19_CopySkipsSystemName(t *testing.T) {
	svc, attrRepo, etvRepo, assocRepo, _ := setupAttrService()

	srcV1 := &models.EntityTypeVersion{ID: "src-v1", EntityTypeID: "src-et", Version: 1}
	tgtV1 := &models.EntityTypeVersion{ID: "tgt-v1", EntityTypeID: "tgt-et", Version: 1}

	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "src-et", 1).Return(srcV1, nil)
	etvRepo.On("GetLatestByEntityType", mock.Anything, "tgt-et").Return(tgtV1, nil)

	attrRepo.On("ListByVersion", mock.Anything, "src-v1").Return([]*models.Attribute{
		{Name: "hostname", Type: models.AttributeTypeString},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "tgt-v1").Return([]*models.Attribute{}, nil)

	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, "tgt-v1", mock.AnythingOfType("string")).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, "tgt-v1", mock.AnythingOfType("string")).Return(nil)
	attrRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Attribute")).Return(nil)

	// Request both "name" (system) and "hostname" (custom) — only hostname should be copied
	newVer, err := svc.CopyAttributesFromType(context.Background(), "tgt-et", "src-et", 1, []string{"name", "hostname"})
	require.NoError(t, err)
	assert.Equal(t, 2, newVer.Version)
	// Verify Create was called exactly once — for "hostname", not "name"
	attrRepo.AssertNumberOfCalls(t, "Create", 1)
}

// T-18.20: CopyAttributes with "description" in list silently skips it
func TestT18_20_CopySkipsSystemDescription(t *testing.T) {
	svc, attrRepo, etvRepo, assocRepo, _ := setupAttrService()

	srcV1 := &models.EntityTypeVersion{ID: "src-v1", EntityTypeID: "src-et", Version: 1}
	tgtV1 := &models.EntityTypeVersion{ID: "tgt-v1", EntityTypeID: "tgt-et", Version: 1}

	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "src-et", 1).Return(srcV1, nil)
	etvRepo.On("GetLatestByEntityType", mock.Anything, "tgt-et").Return(tgtV1, nil)

	attrRepo.On("ListByVersion", mock.Anything, "src-v1").Return([]*models.Attribute{
		{Name: "hostname", Type: models.AttributeTypeString},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "tgt-v1").Return([]*models.Attribute{}, nil)

	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, "tgt-v1", mock.AnythingOfType("string")).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, "tgt-v1", mock.AnythingOfType("string")).Return(nil)
	attrRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Attribute")).Return(nil)

	newVer, err := svc.CopyAttributesFromType(context.Background(), "tgt-et", "src-et", 1, []string{"description", "hostname"})
	require.NoError(t, err)
	assert.Equal(t, 2, newVer.Version)
	attrRepo.AssertNumberOfCalls(t, "Create", 1)
}

// T-18.21: CopyAttributes with only system names results in no new attrs
func TestT18_21_CopyOnlySystemNamesNoOp(t *testing.T) {
	svc, attrRepo, etvRepo, _, _ := setupAttrService()

	srcV1 := &models.EntityTypeVersion{ID: "src-v1", EntityTypeID: "src-et", Version: 1}

	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "src-et", 1).Return(srcV1, nil)

	attrRepo.On("ListByVersion", mock.Anything, "src-v1").Return([]*models.Attribute{}, nil)

	// Requesting only system names — should return error "attribute not found" since "name" isn't a real attr
	_, err := svc.CopyAttributesFromType(context.Background(), "tgt-et", "src-et", 1, []string{"name", "description"})
	// After filtering system names, the list is empty. The service should handle this gracefully.
	// Current behavior: it would try to find attrs named "name"/"description" in source and fail.
	// With the fix: system names are skipped, and if nothing remains, we should get an error or no-op.
	assert.Error(t, err) // no custom attrs to copy
}

// === I5: Service-level reserved name guard ===

// AddAttribute rejects reserved name "name" at service level
func TestAddAttribute_RejectsReservedName(t *testing.T) {
	svc, _, _, _, _ := setupAttrService()
	_, err := svc.AddAttribute(context.Background(), "et1", "name", "", models.AttributeTypeString, "", false)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

// AddAttribute rejects reserved name "description" at service level
func TestAddAttribute_RejectsReservedDescription(t *testing.T) {
	svc, _, _, _, _ := setupAttrService()
	_, err := svc.AddAttribute(context.Background(), "et1", "description", "", models.AttributeTypeString, "", false)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

// EditAttribute rejects renaming to reserved name at service level
func TestEditAttribute_RejectsRenameToReserved(t *testing.T) {
	svc, _, _, _, _ := setupAttrService()
	reservedName := "name"
	_, err := svc.EditAttribute(context.Background(), "et1", "hostname", &reservedName, nil, nil, nil, nil)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}
