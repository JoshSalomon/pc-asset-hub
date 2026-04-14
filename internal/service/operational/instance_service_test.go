package operational_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository/mocks"
	"github.com/project-catalyst/pc-asset-hub/internal/service/operational"
)

type instanceTestSetup struct {
	svc         *operational.InstanceService
	instRepo    *mocks.MockEntityInstanceRepo
	iavRepo     *mocks.MockInstanceAttributeValueRepo
	catalogRepo *mocks.MockCatalogRepo
	cvRepo      *mocks.MockCatalogVersionRepo
	pinRepo     *mocks.MockCatalogVersionPinRepo
	attrRepo    *mocks.MockAttributeRepo
	etvRepo     *mocks.MockEntityTypeVersionRepo
	etRepo      *mocks.MockEntityTypeRepo
	tdvRepo     *mocks.MockTypeDefinitionVersionRepo
	tdRepo      *mocks.MockTypeDefinitionRepo
	assocRepo   *mocks.MockAssociationRepo
	linkRepo    *mocks.MockAssociationLinkRepo
}

func setupInstanceService() *instanceTestSetup {
	s := &instanceTestSetup{
		instRepo:    new(mocks.MockEntityInstanceRepo),
		iavRepo:     new(mocks.MockInstanceAttributeValueRepo),
		catalogRepo: new(mocks.MockCatalogRepo),
		cvRepo:      new(mocks.MockCatalogVersionRepo),
		pinRepo:     new(mocks.MockCatalogVersionPinRepo),
		attrRepo:    new(mocks.MockAttributeRepo),
		etvRepo:     new(mocks.MockEntityTypeVersionRepo),
		etRepo:      new(mocks.MockEntityTypeRepo),
		tdvRepo:     new(mocks.MockTypeDefinitionVersionRepo),
		tdRepo:      new(mocks.MockTypeDefinitionRepo),
		assocRepo:   new(mocks.MockAssociationRepo),
		linkRepo:    new(mocks.MockAssociationLinkRepo),
	}
	s.svc = operational.NewInstanceService(
		s.instRepo, s.iavRepo, s.catalogRepo, s.cvRepo,
		s.pinRepo, s.attrRepo, s.etvRepo, s.etRepo, s.tdvRepo,
		s.tdRepo, s.assocRepo, s.linkRepo,
	)
	return s
}

// mockPinResolution sets up the full catalog → CV → pin → entity type version chain.
func (s *instanceTestSetup) mockPinResolution(ctx context.Context) {
	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusDraft,
	}, nil)
	s.etRepo.On("GetByName", ctx, "model").Return(&models.EntityType{ID: "et1", Name: "model"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{
		ID: "etv1", EntityTypeID: "et1", Version: 1,
	}, nil)
}

// mockAttributes sets up attributes for the pinned entity type version.
func (s *instanceTestSetup) mockAttributes(ctx context.Context) {
	s.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{
		{ID: "a1", Name: "hostname", TypeDefinitionVersionID: "tdv-string", EntityTypeVersionID: "etv1"},
		{ID: "a2", Name: "port", TypeDefinitionVersionID: "tdv-number", EntityTypeVersionID: "etv1"},
		{ID: "a3", Name: "status", TypeDefinitionVersionID: "tdv-enum1", EntityTypeVersionID: "etv1"},
	}, nil)
	// Set up type definition resolution
	s.tdvRepo.On("GetByID", ctx, "tdv-string").Return(&models.TypeDefinitionVersion{ID: "tdv-string", TypeDefinitionID: "td-string"}, nil)
	s.tdvRepo.On("GetByID", ctx, "tdv-number").Return(&models.TypeDefinitionVersion{ID: "tdv-number", TypeDefinitionID: "td-number"}, nil)
	s.tdvRepo.On("GetByID", ctx, "tdv-enum1").Return(&models.TypeDefinitionVersion{ID: "tdv-enum1", TypeDefinitionID: "td-enum1", Constraints: map[string]any{"values": []any{"active", "inactive"}}}, nil)
	s.tdRepo.On("GetByID", ctx, "td-string").Return(&models.TypeDefinition{ID: "td-string", BaseType: models.BaseTypeString}, nil)
	s.tdRepo.On("GetByID", ctx, "td-number").Return(&models.TypeDefinition{ID: "td-number", BaseType: models.BaseTypeNumber}, nil)
	s.tdRepo.On("GetByID", ctx, "td-enum1").Return(&models.TypeDefinition{ID: "td-enum1", BaseType: models.BaseTypeEnum}, nil)
}

// T-11.11: Create instance in catalog with pinned entity type
func TestT11_11_CreateInstance(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.mockAttributes(ctx)

	s.instRepo.On("Create", ctx, mock.AnythingOfType("*models.EntityInstance")).Return(nil)
	s.iavRepo.On("SetValues", ctx, mock.Anything).Return(nil)
	s.iavRepo.On("GetCurrentValues", ctx, mock.Anything).Return([]*models.InstanceAttributeValue{
		{ID: "v1", InstanceID: "inst1", InstanceVersion: 1, AttributeID: "a1", ValueString: "myhost"},
	}, nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft).Return(nil)

	detail, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "my-instance", "desc", map[string]any{
		"hostname": "myhost",
	})
	require.NoError(t, err)
	assert.Equal(t, "my-instance", detail.Instance.Name)
	assert.Equal(t, "cat1", detail.Instance.CatalogID)
	assert.Equal(t, "et1", detail.Instance.EntityTypeID)
	assert.Equal(t, 1, detail.Instance.Version)
	s.instRepo.AssertCalled(t, "Create", ctx, mock.AnythingOfType("*models.EntityInstance"))
}

// T-11.12: Create instance with entity type not pinned in CV
func TestT11_12_EntityTypeNotPinned(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", CatalogVersionID: "cv1",
	}, nil)
	s.etRepo.On("GetByName", ctx, "unknown-type").Return(&models.EntityType{ID: "et-other", Name: "unknown-type"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{
		ID: "etv1", EntityTypeID: "et1", Version: 1, // et1 != et-other
	}, nil)

	_, err := s.svc.CreateInstance(ctx, "my-catalog", "unknown-type", "inst", "", nil)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
}

// T-11.13: Create instance in nonexistent catalog
func TestT11_13_NonexistentCatalog(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "no-such").Return(nil, domainerrors.NewNotFound("Catalog", "no-such"))

	_, err := s.svc.CreateInstance(ctx, "no-such", "model", "inst", "", nil)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
}

// T-11.14: Create instance with string attribute
func TestT11_14_StringAttribute(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.mockAttributes(ctx)

	s.instRepo.On("Create", ctx, mock.Anything).Return(nil)
	s.iavRepo.On("SetValues", ctx, mock.MatchedBy(func(vals []*models.InstanceAttributeValue) bool {
		return len(vals) == 1 && vals[0].ValueString == "myhost" && vals[0].AttributeID == "a1"
	})).Return(nil)
	s.iavRepo.On("GetCurrentValues", ctx, mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft).Return(nil)

	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", map[string]any{
		"hostname": "myhost",
	})
	require.NoError(t, err)
	s.iavRepo.AssertCalled(t, "SetValues", ctx, mock.Anything)
}

// T-11.15: Create instance with number attribute
func TestT11_15_NumberAttribute(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.mockAttributes(ctx)

	s.instRepo.On("Create", ctx, mock.Anything).Return(nil)
	s.iavRepo.On("SetValues", ctx, mock.MatchedBy(func(vals []*models.InstanceAttributeValue) bool {
		return len(vals) == 1 && vals[0].ValueNumber != nil && *vals[0].ValueNumber == 8080
	})).Return(nil)
	s.iavRepo.On("GetCurrentValues", ctx, mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft).Return(nil)

	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", map[string]any{
		"port": float64(8080),
	})
	require.NoError(t, err)
}

// T-11.16: Create instance with valid enum value
func TestT11_16_ValidEnumValue(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.mockAttributes(ctx)

	s.instRepo.On("Create", ctx, mock.Anything).Return(nil)
	s.iavRepo.On("SetValues", ctx, mock.MatchedBy(func(vals []*models.InstanceAttributeValue) bool {
		return len(vals) == 1 && vals[0].ValueString == "active"
	})).Return(nil)
	s.iavRepo.On("GetCurrentValues", ctx, mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft).Return(nil)

	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", map[string]any{
		"status": "active",
	})
	require.NoError(t, err)
}

// T-11.17: Create instance with enum value (validation deferred to catalog validation)
func TestT11_17_EnumValueStoredAsString(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.mockAttributes(ctx)

	s.instRepo.On("Create", ctx, mock.Anything).Return(nil)
	s.iavRepo.On("SetValues", ctx, mock.MatchedBy(func(vals []*models.InstanceAttributeValue) bool {
		return len(vals) == 1 && vals[0].ValueString == "bogus"
	})).Return(nil)
	s.iavRepo.On("GetCurrentValues", ctx, mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft).Return(nil)

	// Enum value validation is now deferred to catalog validation, so create succeeds
	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", map[string]any{
		"status": "bogus",
	})
	require.NoError(t, err)
}

// T-11.18: Create instance with non-parseable number
func TestT11_18_NonParseableNumber(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.mockAttributes(ctx)

	s.instRepo.On("Create", ctx, mock.Anything).Return(nil)

	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", map[string]any{
		"port": "not-a-number",
	})
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

// T-11.19: Create instance with missing optional attributes
func TestT11_19_MissingOptionalAttrs(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.mockAttributes(ctx)

	s.instRepo.On("Create", ctx, mock.Anything).Return(nil)
	s.iavRepo.On("GetCurrentValues", ctx, mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft).Return(nil)

	// No attribute values provided — should succeed (draft mode)
	detail, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", nil)
	require.NoError(t, err)
	assert.NotNil(t, detail)
}

// T-11.20: Create instance with missing required attributes (draft mode allows it)
func TestT11_20_MissingRequiredAttrs(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)

	// Override attributes with a required one
	s.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{
		{ID: "a1", Name: "hostname", TypeDefinitionVersionID: "tdv-string", Required: true},
	}, nil)
	s.tdvRepo.On("GetByID", ctx, "tdv-string").Return(&models.TypeDefinitionVersion{ID: "tdv-string", TypeDefinitionID: "td-string"}, nil)
	s.tdRepo.On("GetByID", ctx, "td-string").Return(&models.TypeDefinition{ID: "td-string", BaseType: models.BaseTypeString}, nil)

	s.instRepo.On("Create", ctx, mock.Anything).Return(nil)
	s.iavRepo.On("GetCurrentValues", ctx, mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft).Return(nil)

	// Missing required attr — should still succeed in draft mode
	detail, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", nil)
	require.NoError(t, err)
	assert.NotNil(t, detail)
}

// T-11.21: Create instance with duplicate name
func TestT11_21_DuplicateName(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.mockAttributes(ctx)

	s.instRepo.On("Create", ctx, mock.Anything).Return(domainerrors.NewConflict("EntityInstance", "name exists"))

	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "duplicate", "", nil)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsConflict(err))
}

// T-11.22: Create instance with unknown attribute name
func TestT11_22_UnknownAttribute(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.mockAttributes(ctx)

	s.instRepo.On("Create", ctx, mock.Anything).Return(nil)

	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", map[string]any{
		"nonexistent-attr": "value",
	})
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

// T-11.23: Update instance attribute values
func TestT11_23_UpdateAttributes(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.mockAttributes(ctx)

	s.instRepo.On("GetByID", ctx, "inst1").Return(&models.EntityInstance{
		ID: "inst1", EntityTypeID: "et1", CatalogID: "cat1", Name: "my-inst", Version: 1,
	}, nil)
	s.instRepo.On("Update", ctx, mock.Anything).Return(nil)
	s.iavRepo.On("GetValuesForVersion", ctx, "inst1", 1).Return([]*models.InstanceAttributeValue{
		{AttributeID: "a1", ValueString: "oldhost"},
	}, nil)
	s.iavRepo.On("SetValues", ctx, mock.Anything).Return(nil)
	s.iavRepo.On("GetCurrentValues", ctx, mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft).Return(nil)

	detail, err := s.svc.UpdateInstance(ctx, "my-catalog", "model", "inst1", 1, nil, nil, map[string]any{
		"hostname": "newhost",
	})
	require.NoError(t, err)
	assert.Equal(t, 2, detail.Instance.Version)
}

// T-11.24: Update instance with version mismatch
func TestT11_24_VersionMismatch(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)

	s.instRepo.On("GetByID", ctx, "inst1").Return(&models.EntityInstance{
		ID: "inst1", CatalogID: "cat1", Version: 3,
	}, nil)

	_, err := s.svc.UpdateInstance(ctx, "my-catalog", "model", "inst1", 1, nil, nil, nil)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsConflict(err))
}

// T-11.25: Update instance name and description
func TestT11_25_UpdateNameDesc(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.mockAttributes(ctx)

	s.instRepo.On("GetByID", ctx, "inst1").Return(&models.EntityInstance{
		ID: "inst1", EntityTypeID: "et1", CatalogID: "cat1", Name: "old-name", Description: "old-desc", Version: 1,
	}, nil)
	s.instRepo.On("Update", ctx, mock.Anything).Return(nil)
	s.iavRepo.On("GetValuesForVersion", ctx, "inst1", 1).Return([]*models.InstanceAttributeValue{}, nil)
	s.iavRepo.On("GetCurrentValues", ctx, mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft).Return(nil)

	newName := "new-name"
	newDesc := "new-desc"
	detail, err := s.svc.UpdateInstance(ctx, "my-catalog", "model", "inst1", 1, &newName, &newDesc, nil)
	require.NoError(t, err)
	assert.Equal(t, "new-name", detail.Instance.Name)
	assert.Equal(t, "new-desc", detail.Instance.Description)
	assert.Equal(t, 2, detail.Instance.Version)
}

// T-11.26: Update with invalid attribute value type
func TestT11_26_InvalidAttrType(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.mockAttributes(ctx)

	s.instRepo.On("GetByID", ctx, "inst1").Return(&models.EntityInstance{
		ID: "inst1", CatalogID: "cat1", Version: 1,
	}, nil)

	_, err := s.svc.UpdateInstance(ctx, "my-catalog", "model", "inst1", 1, nil, nil, map[string]any{
		"port": "not-a-number",
	})
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
	// Verify Update was NOT called — validation failed before version increment
	s.instRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

// T-11.27: Update nonexistent instance
func TestT11_27_UpdateNotFound(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)

	s.instRepo.On("GetByID", ctx, "nope").Return(nil, domainerrors.NewNotFound("EntityInstance", "nope"))

	_, err := s.svc.UpdateInstance(ctx, "my-catalog", "model", "nope", 1, nil, nil, nil)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
}

// T-11.28: Get instance returns resolved attribute values
func TestT11_28_GetWithAttributes(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.mockAttributes(ctx)

	s.instRepo.On("GetByID", ctx, "inst1").Return(&models.EntityInstance{
		ID: "inst1", EntityTypeID: "et1", CatalogID: "cat1", Name: "my-inst", Version: 1,
	}, nil)
	s.iavRepo.On("GetCurrentValues", ctx, "inst1").Return([]*models.InstanceAttributeValue{
		{ID: "v1", InstanceID: "inst1", InstanceVersion: 1, AttributeID: "a1", ValueString: "myhost"},
		{ID: "v2", InstanceID: "inst1", InstanceVersion: 1, AttributeID: "a2", ValueNumber: ptrFloat(8080)},
	}, nil)

	detail, err := s.svc.GetInstance(ctx, "my-catalog", "model", "inst1")
	require.NoError(t, err)
	assert.Equal(t, "my-inst", detail.Instance.Name)
	assert.Len(t, detail.Attributes, 3) // 3 schema attrs, 2 with values
	assert.Equal(t, "hostname", detail.Attributes[0].Name)
	assert.Equal(t, "myhost", detail.Attributes[0].Value)
	assert.Equal(t, "port", detail.Attributes[1].Name)
	assert.Equal(t, ptrFloat(8080), detail.Attributes[1].Value)
	assert.Equal(t, "status", detail.Attributes[2].Name)
	assert.Nil(t, detail.Attributes[2].Value) // no value set
}

// T-11.29: List instances returns instances with attribute values
func TestT11_29_ListWithAttributes(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.mockAttributes(ctx)

	s.instRepo.On("List", ctx, "et1", "cat1", mock.AnythingOfType("models.ListParams")).Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "cat1", Name: "inst-a", Version: 1},
	}, 1, nil)
	s.iavRepo.On("GetCurrentValues", ctx, "inst1").Return([]*models.InstanceAttributeValue{}, nil)

	details, total, err := s.svc.ListInstances(ctx, "my-catalog", "model", models.ListParams{Limit: 20})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, details, 1)
	assert.Len(t, details[0].Attributes, 3) // schema has 3 attrs
}

// T-11.30: List instances in catalog with no instances
func TestT11_30_ListEmpty(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.mockAttributes(ctx)

	s.instRepo.On("List", ctx, "et1", "cat1", mock.AnythingOfType("models.ListParams")).Return(
		[]*models.EntityInstance{}, 0, nil)

	details, total, err := s.svc.ListInstances(ctx, "my-catalog", "model", models.ListParams{Limit: 20})
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Len(t, details, 0)
}

// T-11.31: Delete instance cascades to children
func TestT11_31_CascadeDelete(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)

	s.instRepo.On("GetByID", ctx, "parent1").Return(&models.EntityInstance{
		ID: "parent1", CatalogID: "cat1",
	}, nil)
	s.instRepo.On("ListByParent", ctx, "parent1", mock.Anything).Return([]*models.EntityInstance{
		{ID: "child1"},
	}, 1, nil)
	s.instRepo.On("ListByParent", ctx, "child1", mock.Anything).Return([]*models.EntityInstance{}, 0, nil)
	s.linkRepo.On("DeleteByInstance", ctx, "child1").Return(nil)
	s.instRepo.On("SoftDelete", ctx, "child1").Return(nil)
	s.linkRepo.On("DeleteByInstance", ctx, "parent1").Return(nil)
	s.instRepo.On("SoftDelete", ctx, "parent1").Return(nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft).Return(nil)

	err := s.svc.DeleteInstance(ctx, "my-catalog", "model", "parent1")
	require.NoError(t, err)
	s.instRepo.AssertCalled(t, "SoftDelete", ctx, "child1")
	s.instRepo.AssertCalled(t, "SoftDelete", ctx, "parent1")
}

// T-11.32: Create instance resets catalog validation status to draft
func TestT11_32_CreateResetsDraft(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	// Use a catalog with "valid" status
	s.catalogRepo.On("GetByName", ctx, "valid-cat").Return(&models.Catalog{
		ID: "cat2", Name: "valid-cat", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusValid,
	}, nil)
	s.etRepo.On("GetByName", ctx, "model").Return(&models.EntityType{ID: "et1", Name: "model"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{
		ID: "etv1", EntityTypeID: "et1", Version: 1,
	}, nil)
	s.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{}, nil)
	s.instRepo.On("Create", ctx, mock.Anything).Return(nil)
	s.iavRepo.On("GetCurrentValues", ctx, mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat2", models.ValidationStatusDraft).Return(nil)

	_, err := s.svc.CreateInstance(ctx, "valid-cat", "model", "inst", "", nil)
	require.NoError(t, err)
	s.catalogRepo.AssertCalled(t, "UpdateValidationStatus", ctx, "cat2", models.ValidationStatusDraft)
}

// T-11.33: Update instance resets catalog validation status to draft
func TestT11_33_UpdateResetsDraft(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.mockAttributes(ctx)

	s.instRepo.On("GetByID", ctx, "inst1").Return(&models.EntityInstance{
		ID: "inst1", EntityTypeID: "et1", CatalogID: "cat1", Version: 1,
	}, nil)
	s.instRepo.On("Update", ctx, mock.Anything).Return(nil)
	s.iavRepo.On("GetValuesForVersion", ctx, "inst1", 1).Return([]*models.InstanceAttributeValue{}, nil)
	s.iavRepo.On("GetCurrentValues", ctx, mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft).Return(nil)

	_, err := s.svc.UpdateInstance(ctx, "my-catalog", "model", "inst1", 1, nil, nil, nil)
	require.NoError(t, err)
	s.catalogRepo.AssertCalled(t, "UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft)
}

// T-11.34: Delete instance resets catalog validation status to draft
func TestT11_34_DeleteResetsDraft(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)

	s.instRepo.On("GetByID", ctx, "inst1").Return(&models.EntityInstance{
		ID: "inst1", CatalogID: "cat1",
	}, nil)
	s.instRepo.On("ListByParent", ctx, "inst1", mock.Anything).Return([]*models.EntityInstance{}, 0, nil)
	s.linkRepo.On("DeleteByInstance", ctx, "inst1").Return(nil)
	s.instRepo.On("SoftDelete", ctx, "inst1").Return(nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft).Return(nil)

	err := s.svc.DeleteInstance(ctx, "my-catalog", "model", "inst1")
	require.NoError(t, err)
	s.catalogRepo.AssertCalled(t, "UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft)
}

// Error propagation: UpdateInstance - instRepo.Update error
func TestCov_UpdateInstance_RepoUpdateError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)

	s.instRepo.On("GetByID", ctx, "i1").Return(&models.EntityInstance{
		ID: "i1", Version: 1,
	}, nil)
	s.iavRepo.On("GetValuesForVersion", ctx, "i1", 1).Return([]*models.InstanceAttributeValue{}, nil)
	s.instRepo.On("Update", ctx, mock.Anything).Return(domainerrors.NewValidation("db error"))

	_, err := s.svc.UpdateInstance(ctx, "my-catalog", "model", "i1", 1, nil, nil, nil)
	assert.Error(t, err)
}

func ptrFloat(f float64) *float64 { return &f }

// === Coverage tests for uncovered lines ===

// Error propagation: etRepo.GetByName returns internal error
func TestCov_ResolveEntityType_GetByNameError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "cat").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	s.etRepo.On("GetByName", ctx, "bad").Return(nil, domainerrors.NewNotFound("EntityType", "bad"))

	_, err := s.svc.CreateInstance(ctx, "cat", "bad", "inst", "", nil)
	assert.Error(t, err)
}

// Error propagation: pinRepo.ListByCatalogVersion error
func TestCov_ResolveEntityType_PinListError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "cat").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	s.etRepo.On("GetByName", ctx, "model").Return(&models.EntityType{ID: "et1"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return(nil, domainerrors.NewValidation("db error"))

	_, err := s.svc.CreateInstance(ctx, "cat", "model", "inst", "", nil)
	assert.Error(t, err)
}

// Error propagation: etvRepo.GetByID error (continue branch)
func TestCov_ResolveEntityType_ETVGetByIDError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "cat").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	s.etRepo.On("GetByName", ctx, "model").Return(&models.EntityType{ID: "et1"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "p1", EntityTypeVersionID: "etv-bad"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv-bad").Return(nil, domainerrors.NewNotFound("ETV", "etv-bad"))

	_, err := s.svc.CreateInstance(ctx, "cat", "model", "inst", "", nil)
	assert.Error(t, err) // Error propagated from GetByID, not silently skipped
}

// Error propagation: resolveAttributeValues - attrRepo error
func TestCov_ResolveAttrValues_AttrRepoError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.attrRepo.ExpectedCalls = nil // clear mockAttributes
	s.attrRepo.On("ListByVersion", ctx, "etv1").Return(nil, domainerrors.NewValidation("db error"))

	s.instRepo.On("Create", ctx, mock.Anything).Return(nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft).Return(nil)

	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", nil)
	assert.Error(t, err)
}

// Error propagation: resolveAttributeValues - iavRepo.GetCurrentValues error
func TestCov_ResolveAttrValues_GetCurrentError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.mockAttributes(ctx)

	s.instRepo.On("Create", ctx, mock.Anything).Return(nil)
	s.iavRepo.On("GetCurrentValues", ctx, mock.Anything).Return(nil, domainerrors.NewValidation("db error"))
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft).Return(nil)

	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", nil)
	assert.Error(t, err)
}

// Error propagation: validateAndBuildAttributeValues - attrRepo error
func TestCov_ValidateAttrs_AttrRepoError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	// First ListByVersion call succeeds (for resolveAttributeValues or CreateInstance flow)
	// But validateAndBuildAttributeValues also calls ListByVersion
	s.attrRepo.On("ListByVersion", ctx, "etv1").Return(nil, domainerrors.NewValidation("db error"))
	s.instRepo.On("Create", ctx, mock.Anything).Return(nil)

	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", map[string]any{"x": "y"})
	assert.Error(t, err)
}

// Error propagation: tdvRepo.GetByID error during type resolution
func TestCov_ValidateAttrs_TDVRepoError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	// Set up attributes but with a bad TDV reference
	s.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{
		{ID: "a1", Name: "hostname", TypeDefinitionVersionID: "tdv-bad", EntityTypeVersionID: "etv1"},
	}, nil)
	s.tdvRepo.On("GetByID", ctx, "tdv-bad").Return(nil, domainerrors.NewValidation("db error"))
	s.instRepo.On("Create", ctx, mock.Anything).Return(nil)

	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", map[string]any{"hostname": "h"})
	assert.Error(t, err)
}

// Error propagation: iavRepo.SetValues error on create
func TestCov_Create_SetValuesError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.mockAttributes(ctx)
	s.instRepo.On("Create", ctx, mock.Anything).Return(nil)
	s.iavRepo.On("SetValues", ctx, mock.Anything).Return(domainerrors.NewValidation("db error"))

	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", map[string]any{"hostname": "h"})
	assert.Error(t, err)
}

// Error propagation: GetInstance - resolveEntityType error
func TestCov_GetInstance_ResolveError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "bad").Return(nil, domainerrors.NewNotFound("Catalog", "bad"))

	_, err := s.svc.GetInstance(ctx, "bad", "model", "i1")
	assert.Error(t, err)
}

// Error propagation: GetInstance - instRepo.GetByID error
func TestCov_GetInstance_GetByIDError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.instRepo.On("GetByID", ctx, "bad").Return(nil, domainerrors.NewNotFound("EntityInstance", "bad"))

	_, err := s.svc.GetInstance(ctx, "my-catalog", "model", "bad")
	assert.Error(t, err)
}

// Error propagation: GetInstance - resolveAttributeValues error
func TestCov_GetInstance_ResolveAttrsError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.attrRepo.On("ListByVersion", ctx, "etv1").Return(nil, domainerrors.NewValidation("db error"))
	s.instRepo.On("GetByID", ctx, "i1").Return(&models.EntityInstance{ID: "i1"}, nil)

	_, err := s.svc.GetInstance(ctx, "my-catalog", "model", "i1")
	assert.Error(t, err)
}

// Error propagation: ListInstances - resolveEntityType error
func TestCov_ListInstances_ResolveError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "bad").Return(nil, domainerrors.NewNotFound("Catalog", "bad"))

	_, _, err := s.svc.ListInstances(ctx, "bad", "model", models.ListParams{Limit: 20})
	assert.Error(t, err)
}

// Error propagation: ListInstances - instRepo.List error
func TestCov_ListInstances_ListError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{}, nil)
	s.instRepo.On("List", ctx, "et1", "cat1", mock.Anything).Return(nil, 0, domainerrors.NewValidation("db error"))

	_, _, err := s.svc.ListInstances(ctx, "my-catalog", "model", models.ListParams{Limit: 20})
	assert.Error(t, err)
}

// Error propagation: ListInstances - attrRepo.ListByVersion error
func TestCov_ListInstances_AttrError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.instRepo.On("List", ctx, "et1", "cat1", mock.Anything).Return([]*models.EntityInstance{
		{ID: "i1"},
	}, 1, nil)
	s.attrRepo.On("ListByVersion", ctx, "etv1").Return(nil, domainerrors.NewValidation("db error"))

	_, _, err := s.svc.ListInstances(ctx, "my-catalog", "model", models.ListParams{Limit: 20})
	assert.Error(t, err)
}

// Error propagation: ListInstances - iavRepo.GetCurrentValues error in loop
func TestCov_ListInstances_ValueError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.mockAttributes(ctx)
	s.instRepo.On("List", ctx, "et1", "cat1", mock.Anything).Return([]*models.EntityInstance{
		{ID: "i1"},
	}, 1, nil)
	s.iavRepo.On("GetCurrentValues", ctx, "i1").Return(nil, domainerrors.NewValidation("db error"))

	_, _, err := s.svc.ListInstances(ctx, "my-catalog", "model", models.ListParams{Limit: 20})
	assert.Error(t, err)
}

// Error propagation: UpdateInstance - resolveEntityType error
func TestCov_UpdateInstance_ResolveError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "bad").Return(nil, domainerrors.NewNotFound("Catalog", "bad"))

	_, err := s.svc.UpdateInstance(ctx, "bad", "model", "i1", 1, nil, nil, nil)
	assert.Error(t, err)
}

// Error propagation: UpdateInstance - GetValuesForVersion error
func TestCov_UpdateInstance_CarryForwardError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.instRepo.On("GetByID", ctx, "i1").Return(&models.EntityInstance{ID: "i1", Version: 1}, nil)
	s.instRepo.On("Update", ctx, mock.Anything).Return(nil)
	s.iavRepo.On("GetValuesForVersion", ctx, "i1", 1).Return(nil, domainerrors.NewValidation("db error"))

	_, err := s.svc.UpdateInstance(ctx, "my-catalog", "model", "i1", 1, nil, nil, nil)
	assert.Error(t, err)
}

// Error propagation: UpdateInstance - validateAndBuildAttributeValues error
func TestCov_UpdateInstance_ValidateError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.attrRepo.On("ListByVersion", ctx, "etv1").Return(nil, domainerrors.NewValidation("db error"))
	s.instRepo.On("GetByID", ctx, "i1").Return(&models.EntityInstance{ID: "i1", Version: 1}, nil)

	_, err := s.svc.UpdateInstance(ctx, "my-catalog", "model", "i1", 1, nil, nil, map[string]any{"x": "y"})
	assert.Error(t, err)
	// Update should NOT be called — validation failed before version increment
	s.instRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

// Error propagation: UpdateInstance - SetValues error
func TestCov_UpdateInstance_SetValuesError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.mockAttributes(ctx)
	s.instRepo.On("GetByID", ctx, "i1").Return(&models.EntityInstance{ID: "i1", Version: 1}, nil)
	s.instRepo.On("Update", ctx, mock.Anything).Return(nil)
	s.iavRepo.On("GetValuesForVersion", ctx, "i1", 1).Return([]*models.InstanceAttributeValue{}, nil)
	s.iavRepo.On("SetValues", ctx, mock.Anything).Return(domainerrors.NewValidation("db error"))

	_, err := s.svc.UpdateInstance(ctx, "my-catalog", "model", "i1", 1, nil, nil, map[string]any{"hostname": "h"})
	assert.Error(t, err)
}

// Error propagation: UpdateInstance - resolveAttributeValues error after update
// This test covers the case where resolveAttributeValues fails AFTER the update succeeds.
// We achieve this by making GetCurrentValues (called by resolveAttributeValues) return an error.
func TestCov_UpdateInstance_ResolveAttrsError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.mockAttributes(ctx)
	s.instRepo.On("GetByID", ctx, "i1").Return(&models.EntityInstance{ID: "i1", CatalogID: "cat1", Version: 1}, nil)
	s.instRepo.On("Update", ctx, mock.Anything).Return(nil)
	s.iavRepo.On("GetValuesForVersion", ctx, "i1", 1).Return([]*models.InstanceAttributeValue{}, nil)
	s.iavRepo.On("SetValues", ctx, mock.Anything).Return(nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft).Return(nil)
	// resolveAttributeValues calls GetCurrentValues — make it fail
	s.iavRepo.On("GetCurrentValues", ctx, mock.Anything).Return(nil, domainerrors.NewValidation("db error"))

	_, err := s.svc.UpdateInstance(ctx, "my-catalog", "model", "i1", 1, nil, nil, map[string]any{"hostname": "h"})
	assert.Error(t, err)
}

// Error propagation: DeleteInstance - resolveEntityType error
func TestCov_DeleteInstance_ResolveError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "bad").Return(nil, domainerrors.NewNotFound("Catalog", "bad"))

	err := s.svc.DeleteInstance(ctx, "bad", "model", "i1")
	assert.Error(t, err)
}

// Error propagation: DeleteInstance - cascadeDelete error
func TestCov_DeleteInstance_CascadeError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.instRepo.On("GetByID", ctx, "i1").Return(&models.EntityInstance{ID: "i1", CatalogID: "cat1"}, nil)
	s.instRepo.On("ListByParent", ctx, "i1", mock.Anything).Return(nil, 0, domainerrors.NewValidation("db error"))

	err := s.svc.DeleteInstance(ctx, "my-catalog", "model", "i1")
	assert.Error(t, err)
}

// Error propagation: cascadeDelete - recursive error
func TestCov_CascadeDelete_RecursiveError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.instRepo.On("GetByID", ctx, "parent").Return(&models.EntityInstance{ID: "parent", CatalogID: "cat1"}, nil)
	s.instRepo.On("ListByParent", ctx, "parent", mock.Anything).Return([]*models.EntityInstance{
		{ID: "child"},
	}, 1, nil)
	s.instRepo.On("ListByParent", ctx, "child", mock.Anything).Return(nil, 0, domainerrors.NewValidation("db error"))

	err := s.svc.DeleteInstance(ctx, "my-catalog", "model", "parent")
	assert.Error(t, err)
}

// Type switch: enum value in resolveAttributeValues
func TestCov_ResolveAttrs_EnumValue(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.mockAttributes(ctx)

	s.instRepo.On("GetByID", ctx, "i1").Return(&models.EntityInstance{ID: "i1", EntityTypeID: "et1", CatalogID: "cat1", Version: 1}, nil)
	s.iavRepo.On("GetCurrentValues", ctx, "i1").Return([]*models.InstanceAttributeValue{
		{AttributeID: "a3", ValueString: "active", InstanceVersion: 1},
	}, nil)

	detail, err := s.svc.GetInstance(ctx, "my-catalog", "model", "i1")
	require.NoError(t, err)
	// Find the status attribute
	for _, av := range detail.Attributes {
		if av.Name == "status" {
			assert.Equal(t, "active", av.Value)
		}
	}
}

// Type switch: int value in validateAndBuildAttributeValues
func TestCov_ValidateAttrs_IntNumber(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.mockAttributes(ctx)
	s.instRepo.On("Create", ctx, mock.Anything).Return(nil)
	s.iavRepo.On("SetValues", ctx, mock.MatchedBy(func(vals []*models.InstanceAttributeValue) bool {
		return len(vals) == 1 && vals[0].ValueNumber != nil && *vals[0].ValueNumber == 42
	})).Return(nil)
	s.iavRepo.On("GetCurrentValues", ctx, mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft).Return(nil)

	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", map[string]any{
		"port": int(42), // int, not float64
	})
	require.NoError(t, err)
}

// Type switch: default (non-numeric) value for number attribute
func TestCov_ValidateAttrs_DefaultNumberType(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.mockAttributes(ctx)
	s.instRepo.On("Create", ctx, mock.Anything).Return(nil)

	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", map[string]any{
		"port": true, // bool — triggers default branch
	})
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

// Type switch: ListInstances with non-empty attribute values (covers inline resolution switch)
func TestCov_ListInstances_WithValues(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.mockAttributes(ctx)

	s.instRepo.On("List", ctx, "et1", "cat1", mock.Anything).Return([]*models.EntityInstance{
		{ID: "i1", EntityTypeID: "et1", CatalogID: "cat1", Name: "inst-a", Version: 1},
	}, 1, nil)
	num := float64(8080)
	s.iavRepo.On("GetCurrentValues", ctx, "i1").Return([]*models.InstanceAttributeValue{
		{AttributeID: "a1", ValueString: "host-a", InstanceVersion: 1},
		{AttributeID: "a2", ValueNumber: &num, InstanceVersion: 1},
		{AttributeID: "a3", ValueString: "active", InstanceVersion: 1},
	}, nil)

	details, total, err := s.svc.ListInstances(ctx, "my-catalog", "model", models.ListParams{Limit: 20})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "host-a", details[0].Attributes[0].Value)
	assert.Equal(t, &num, details[0].Attributes[1].Value)
	assert.Equal(t, "active", details[0].Attributes[2].Value)
}

// === Milestone 12: Containment & Association Links ===

// T-12.05: CreateContainedInstance with valid parent and containment association in CV
func TestT12_05_CreateContainedInstance(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	// Set up catalog with two pinned entity types: "server" (et1/etv1) and "tool" (et2/etv2)
	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusDraft,
	}, nil)
	// Parent entity type "server"
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1", Name: "server"}, nil)
	// Child entity type "tool"
	s.etRepo.On("GetByName", ctx, "tool").Return(&models.EntityType{ID: "et2", Name: "tool"}, nil)
	// Pins for both
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
		{ID: "pin2", CatalogVersionID: "cv1", EntityTypeVersionID: "etv2"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{
		ID: "etv1", EntityTypeID: "et1", Version: 1,
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv2").Return(&models.EntityTypeVersion{
		ID: "etv2", EntityTypeID: "et2", Version: 1,
	}, nil)
	// Parent instance exists
	s.instRepo.On("GetByID", ctx, "parent1").Return(&models.EntityInstance{
		ID: "parent1", EntityTypeID: "et1", CatalogID: "cat1", Name: "my-server", Version: 1,
	}, nil)
	// Containment association exists in CV (server contains tool)
	s.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{
		{ID: "assoc1", EntityTypeVersionID: "etv1", TargetEntityTypeID: "et2",
			Type: models.AssociationTypeContainment, Name: "tools"},
	}, nil)
	// Attributes for child type
	s.attrRepo.On("ListByVersion", ctx, "etv2").Return([]*models.Attribute{}, nil)
	// Instance creation
	s.instRepo.On("Create", ctx, mock.AnythingOfType("*models.EntityInstance")).Return(nil)
	s.iavRepo.On("GetCurrentValues", ctx, mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft).Return(nil)

	detail, err := s.svc.CreateContainedInstance(ctx, "my-catalog", "server", "parent1", "tool", "my-tool", "a tool", nil)
	require.NoError(t, err)
	assert.Equal(t, "my-tool", detail.Instance.Name)
	assert.Equal(t, "parent1", detail.Instance.ParentInstanceID)
	assert.Equal(t, "et2", detail.Instance.EntityTypeID)
	assert.Equal(t, "cat1", detail.Instance.CatalogID)
}

// T-12.06: CreateContainedInstance with nonexistent parent
func TestT12_06_ContainedInstance_NonexistentParent(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1", Name: "server"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{
		ID: "etv1", EntityTypeID: "et1", Version: 1,
	}, nil)
	s.instRepo.On("GetByID", ctx, "nope").Return(nil, domainerrors.NewNotFound("EntityInstance", "nope"))

	_, err := s.svc.CreateContainedInstance(ctx, "my-catalog", "server", "nope", "tool", "my-tool", "", nil)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
}

// T-12.07: CreateContainedInstance with no containment relationship
func TestT12_07_ContainedInstance_NoContainmentRelation(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1", Name: "server"}, nil)
	s.etRepo.On("GetByName", ctx, "model").Return(&models.EntityType{ID: "et3", Name: "model"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
		{ID: "pin3", CatalogVersionID: "cv1", EntityTypeVersionID: "etv3"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{
		ID: "etv1", EntityTypeID: "et1", Version: 1,
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv3").Return(&models.EntityTypeVersion{
		ID: "etv3", EntityTypeID: "et3", Version: 1,
	}, nil)
	s.instRepo.On("GetByID", ctx, "parent1").Return(&models.EntityInstance{
		ID: "parent1", EntityTypeID: "et1", CatalogID: "cat1", Name: "my-server", Version: 1,
	}, nil)
	// No containment association between server and model
	s.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{}, nil)

	_, err := s.svc.CreateContainedInstance(ctx, "my-catalog", "server", "parent1", "model", "my-model", "", nil)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

// T-12.08: CreateContainedInstance with child type not pinned in CV
func TestT12_08_ContainedInstance_ChildNotPinned(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1", Name: "server"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{
		ID: "etv1", EntityTypeID: "et1", Version: 1,
	}, nil)
	s.instRepo.On("GetByID", ctx, "parent1").Return(&models.EntityInstance{
		ID: "parent1", EntityTypeID: "et1", CatalogID: "cat1", Version: 1,
	}, nil)
	// "unpinned" entity type resolves but is not pinned
	s.etRepo.On("GetByName", ctx, "unpinned").Return(&models.EntityType{ID: "et-unpinned", Name: "unpinned"}, nil)

	_, err := s.svc.CreateContainedInstance(ctx, "my-catalog", "server", "parent1", "unpinned", "child", "", nil)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
}

// T-12.09: Same name under different parents → allowed
func TestT12_09_ContainedInstance_SameNameDiffParents(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusDraft,
	}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1", Name: "server"}, nil)
	s.etRepo.On("GetByName", ctx, "tool").Return(&models.EntityType{ID: "et2", Name: "tool"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
		{ID: "pin2", CatalogVersionID: "cv1", EntityTypeVersionID: "etv2"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.etvRepo.On("GetByID", ctx, "etv2").Return(&models.EntityTypeVersion{ID: "etv2", EntityTypeID: "et2", Version: 1}, nil)
	s.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{
		{ID: "assoc1", EntityTypeVersionID: "etv1", TargetEntityTypeID: "et2", Type: models.AssociationTypeContainment, Name: "tools"},
	}, nil)
	s.attrRepo.On("ListByVersion", ctx, "etv2").Return([]*models.Attribute{}, nil)
	s.instRepo.On("Create", ctx, mock.AnythingOfType("*models.EntityInstance")).Return(nil)
	s.iavRepo.On("GetCurrentValues", ctx, mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft).Return(nil)

	// Create under parent1
	s.instRepo.On("GetByID", ctx, "parent1").Return(&models.EntityInstance{
		ID: "parent1", EntityTypeID: "et1", CatalogID: "cat1", Version: 1,
	}, nil)
	d1, err := s.svc.CreateContainedInstance(ctx, "my-catalog", "server", "parent1", "tool", "same-name", "", nil)
	require.NoError(t, err)
	assert.Equal(t, "parent1", d1.Instance.ParentInstanceID)

	// Create under parent2 with same name — should succeed
	s.instRepo.On("GetByID", ctx, "parent2").Return(&models.EntityInstance{
		ID: "parent2", EntityTypeID: "et1", CatalogID: "cat1", Version: 1,
	}, nil)
	d2, err := s.svc.CreateContainedInstance(ctx, "my-catalog", "server", "parent2", "tool", "same-name", "", nil)
	require.NoError(t, err)
	assert.Equal(t, "parent2", d2.Instance.ParentInstanceID)
}

// T-12.10: Duplicate name under same parent → conflict
func TestT12_10_ContainedInstance_DuplicateNameSameParent(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1", Name: "server"}, nil)
	s.etRepo.On("GetByName", ctx, "tool").Return(&models.EntityType{ID: "et2", Name: "tool"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
		{ID: "pin2", CatalogVersionID: "cv1", EntityTypeVersionID: "etv2"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.etvRepo.On("GetByID", ctx, "etv2").Return(&models.EntityTypeVersion{ID: "etv2", EntityTypeID: "et2", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "parent1").Return(&models.EntityInstance{
		ID: "parent1", EntityTypeID: "et1", CatalogID: "cat1", Version: 1,
	}, nil)
	s.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{
		{ID: "assoc1", EntityTypeVersionID: "etv1", TargetEntityTypeID: "et2", Type: models.AssociationTypeContainment, Name: "tools"},
	}, nil)
	s.attrRepo.On("ListByVersion", ctx, "etv2").Return([]*models.Attribute{}, nil)
	s.instRepo.On("Create", ctx, mock.AnythingOfType("*models.EntityInstance")).Return(
		domainerrors.NewConflict("EntityInstance", "name exists"),
	)

	_, err := s.svc.CreateContainedInstance(ctx, "my-catalog", "server", "parent1", "tool", "dup-name", "", nil)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsConflict(err))
}

// T-12.11: ListContainedInstances returns only direct children of specified type
func TestT12_11_ListContainedInstances(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1", Name: "server"}, nil)
	s.etRepo.On("GetByName", ctx, "tool").Return(&models.EntityType{ID: "et2", Name: "tool"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
		{ID: "pin2", CatalogVersionID: "cv1", EntityTypeVersionID: "etv2"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.etvRepo.On("GetByID", ctx, "etv2").Return(&models.EntityTypeVersion{ID: "etv2", EntityTypeID: "et2", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "parent1").Return(&models.EntityInstance{
		ID: "parent1", EntityTypeID: "et1", CatalogID: "cat1", Version: 1,
	}, nil)
	// Return mixed children — one tool (et2), one other type (et3)
	s.instRepo.On("ListByParent", ctx, "parent1", mock.Anything).Return([]*models.EntityInstance{
		{ID: "child1", EntityTypeID: "et2", CatalogID: "cat1", ParentInstanceID: "parent1", Name: "tool-a"},
		{ID: "child2", EntityTypeID: "et3", CatalogID: "cat1", ParentInstanceID: "parent1", Name: "other"},
	}, 2, nil)
	s.attrRepo.On("ListByVersion", ctx, "etv2").Return([]*models.Attribute{}, nil)
	s.iavRepo.On("GetCurrentValues", ctx, "child1").Return([]*models.InstanceAttributeValue{}, nil)

	details, _, err := s.svc.ListContainedInstances(ctx, "my-catalog", "server", "parent1", "tool", models.ListParams{Limit: 20})
	require.NoError(t, err)
	assert.Len(t, details, 1)
	assert.Equal(t, "tool-a", details[0].Instance.Name)
}

// T-12.12: CreateContainedInstance resets catalog validation status to draft
func TestT12_12_ContainedInstance_ResetsDraft(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "valid-cat").Return(&models.Catalog{
		ID: "cat2", Name: "valid-cat", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusValid,
	}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1", Name: "server"}, nil)
	s.etRepo.On("GetByName", ctx, "tool").Return(&models.EntityType{ID: "et2", Name: "tool"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
		{ID: "pin2", CatalogVersionID: "cv1", EntityTypeVersionID: "etv2"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.etvRepo.On("GetByID", ctx, "etv2").Return(&models.EntityTypeVersion{ID: "etv2", EntityTypeID: "et2", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "parent1").Return(&models.EntityInstance{
		ID: "parent1", EntityTypeID: "et1", CatalogID: "cat2", Version: 1,
	}, nil)
	s.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{
		{ID: "assoc1", EntityTypeVersionID: "etv1", TargetEntityTypeID: "et2", Type: models.AssociationTypeContainment, Name: "tools"},
	}, nil)
	s.attrRepo.On("ListByVersion", ctx, "etv2").Return([]*models.Attribute{}, nil)
	s.instRepo.On("Create", ctx, mock.AnythingOfType("*models.EntityInstance")).Return(nil)
	s.iavRepo.On("GetCurrentValues", ctx, mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat2", models.ValidationStatusDraft).Return(nil)

	_, err := s.svc.CreateContainedInstance(ctx, "valid-cat", "server", "parent1", "tool", "my-tool", "", nil)
	require.NoError(t, err)
	s.catalogRepo.AssertCalled(t, "UpdateValidationStatus", ctx, "cat2", models.ValidationStatusDraft)
}

// T-12.13: CreateContainedInstance with attribute values
func TestT12_13_ContainedInstance_WithAttributes(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusDraft,
	}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1", Name: "server"}, nil)
	s.etRepo.On("GetByName", ctx, "tool").Return(&models.EntityType{ID: "et2", Name: "tool"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
		{ID: "pin2", CatalogVersionID: "cv1", EntityTypeVersionID: "etv2"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.etvRepo.On("GetByID", ctx, "etv2").Return(&models.EntityTypeVersion{ID: "etv2", EntityTypeID: "et2", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "parent1").Return(&models.EntityInstance{
		ID: "parent1", EntityTypeID: "et1", CatalogID: "cat1", Version: 1,
	}, nil)
	s.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{
		{ID: "assoc1", EntityTypeVersionID: "etv1", TargetEntityTypeID: "et2", Type: models.AssociationTypeContainment, Name: "tools"},
	}, nil)
	s.attrRepo.On("ListByVersion", ctx, "etv2").Return([]*models.Attribute{
		{ID: "a1", Name: "description", TypeDefinitionVersionID: "tdv-string", EntityTypeVersionID: "etv2"},
	}, nil)
	s.tdvRepo.On("GetByID", ctx, "tdv-string").Return(&models.TypeDefinitionVersion{ID: "tdv-string", TypeDefinitionID: "td-string"}, nil)
	s.tdRepo.On("GetByID", ctx, "td-string").Return(&models.TypeDefinition{ID: "td-string", BaseType: models.BaseTypeString}, nil)
	s.instRepo.On("Create", ctx, mock.AnythingOfType("*models.EntityInstance")).Return(nil)
	s.iavRepo.On("SetValues", ctx, mock.MatchedBy(func(vals []*models.InstanceAttributeValue) bool {
		return len(vals) == 1 && vals[0].ValueString == "a useful tool"
	})).Return(nil)
	s.iavRepo.On("GetCurrentValues", ctx, mock.Anything).Return([]*models.InstanceAttributeValue{
		{AttributeID: "a1", ValueString: "a useful tool", InstanceVersion: 1},
	}, nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft).Return(nil)

	detail, err := s.svc.CreateContainedInstance(ctx, "my-catalog", "server", "parent1", "tool", "my-tool", "", map[string]any{
		"description": "a useful tool",
	})
	require.NoError(t, err)
	assert.Len(t, detail.Attributes, 1)
	assert.Equal(t, "a useful tool", detail.Attributes[0].Value)
}

// === Association Link Service Tests ===

// T-12.19: CreateAssociationLink with valid association definition in CV
func TestT12_19_CreateAssociationLink(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusDraft,
	}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1", Name: "server"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
		{ID: "pin2", CatalogVersionID: "cv1", EntityTypeVersionID: "etv2"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.etvRepo.On("GetByID", ctx, "etv2").Return(&models.EntityTypeVersion{ID: "etv2", EntityTypeID: "et2", Version: 1}, nil)
	// Source instance
	s.instRepo.On("GetByID", ctx, "inst1").Return(&models.EntityInstance{
		ID: "inst1", EntityTypeID: "et1", CatalogID: "cat1", Version: 1,
	}, nil)
	// Target instance
	s.instRepo.On("GetByID", ctx, "inst2").Return(&models.EntityInstance{
		ID: "inst2", EntityTypeID: "et2", CatalogID: "cat1", Version: 1,
	}, nil)
	// Association definition: server → model (directional)
	s.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{
		{ID: "assoc1", EntityTypeVersionID: "etv1", TargetEntityTypeID: "et2",
			Type: models.AssociationTypeDirectional, Name: "uses-model"},
	}, nil)
	s.linkRepo.On("GetForwardRefs", ctx, "inst1").Return([]*models.AssociationLink{}, nil)
	s.linkRepo.On("Create", ctx, mock.AnythingOfType("*models.AssociationLink")).Return(nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft).Return(nil)

	link, err := s.svc.CreateAssociationLink(ctx, "my-catalog", "server", "inst1", "inst2", "uses-model")
	require.NoError(t, err)
	assert.Equal(t, "assoc1", link.AssociationID)
	assert.Equal(t, "inst1", link.SourceInstanceID)
	assert.Equal(t, "inst2", link.TargetInstanceID)
}

// T-12.20: CreateAssociationLink with nonexistent association name
func TestT12_20_LinkNonexistentAssociation(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1", Name: "server"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "inst1").Return(&models.EntityInstance{
		ID: "inst1", EntityTypeID: "et1", CatalogID: "cat1", Version: 1,
	}, nil)
	s.instRepo.On("GetByID", ctx, "inst2").Return(&models.EntityInstance{
		ID: "inst2", EntityTypeID: "et2", CatalogID: "cat1", Version: 1,
	}, nil)
	s.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{}, nil)

	_, err := s.svc.CreateAssociationLink(ctx, "my-catalog", "server", "inst1", "inst2", "nonexistent")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
}

// T-12.22: CreateAssociationLink target entity type does not match
func TestT12_22_LinkMismatchedTargetType(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1", Name: "server"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "inst1").Return(&models.EntityInstance{
		ID: "inst1", EntityTypeID: "et1", CatalogID: "cat1", Version: 1,
	}, nil)
	// Target instance is et3, but association expects et2
	s.instRepo.On("GetByID", ctx, "inst3").Return(&models.EntityInstance{
		ID: "inst3", EntityTypeID: "et3", CatalogID: "cat1", Version: 1,
	}, nil)
	s.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{
		{ID: "assoc1", EntityTypeVersionID: "etv1", TargetEntityTypeID: "et2",
			Type: models.AssociationTypeDirectional, Name: "uses-model"},
	}, nil)

	_, err := s.svc.CreateAssociationLink(ctx, "my-catalog", "server", "inst1", "inst3", "uses-model")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

// T-12.23: CreateAssociationLink with nonexistent target instance
func TestT12_23_LinkNonexistentTarget(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1", Name: "server"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "inst1").Return(&models.EntityInstance{
		ID: "inst1", EntityTypeID: "et1", CatalogID: "cat1", Version: 1,
	}, nil)
	s.instRepo.On("GetByID", ctx, "nope").Return(nil, domainerrors.NewNotFound("EntityInstance", "nope"))

	_, err := s.svc.CreateAssociationLink(ctx, "my-catalog", "server", "inst1", "nope", "uses-model")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
}

// T-12.25: DeleteAssociationLink removes link
func TestT12_25_DeleteAssociationLink(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1", Name: "server"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.linkRepo.On("GetByID", ctx, "link1").Return(&models.AssociationLink{
		ID: "link1", SourceInstanceID: "inst1",
	}, nil)
	s.instRepo.On("GetByID", ctx, "inst1").Return(&models.EntityInstance{
		ID: "inst1", EntityTypeID: "et1", CatalogID: "cat1",
	}, nil)
	s.linkRepo.On("Delete", ctx, "link1").Return(nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft).Return(nil)

	err := s.svc.DeleteAssociationLink(ctx, "my-catalog", "server", "link1")
	require.NoError(t, err)
	s.linkRepo.AssertCalled(t, "Delete", ctx, "link1")
}

// T-12.26: DeleteAssociationLink nonexistent → NotFound
func TestT12_26_DeleteLinkNotFound(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1", Name: "server"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.linkRepo.On("GetByID", ctx, "nope").Return(nil, domainerrors.NewNotFound("AssociationLink", "nope"))

	err := s.svc.DeleteAssociationLink(ctx, "my-catalog", "server", "nope")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
}

// T-12.27: CreateAssociationLink resets catalog validation status to draft
func TestT12_27_LinkResetsDraft(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "valid-cat").Return(&models.Catalog{
		ID: "cat2", Name: "valid-cat", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusValid,
	}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1", Name: "server"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
		{ID: "pin2", CatalogVersionID: "cv1", EntityTypeVersionID: "etv2"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.etvRepo.On("GetByID", ctx, "etv2").Return(&models.EntityTypeVersion{ID: "etv2", EntityTypeID: "et2", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "inst1").Return(&models.EntityInstance{
		ID: "inst1", EntityTypeID: "et1", CatalogID: "cat2", Version: 1,
	}, nil)
	s.instRepo.On("GetByID", ctx, "inst2").Return(&models.EntityInstance{
		ID: "inst2", EntityTypeID: "et2", CatalogID: "cat2", Version: 1,
	}, nil)
	s.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{
		{ID: "assoc1", EntityTypeVersionID: "etv1", TargetEntityTypeID: "et2",
			Type: models.AssociationTypeDirectional, Name: "uses-model"},
	}, nil)
	s.linkRepo.On("GetForwardRefs", ctx, "inst1").Return([]*models.AssociationLink{}, nil)
	s.linkRepo.On("Create", ctx, mock.AnythingOfType("*models.AssociationLink")).Return(nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat2", models.ValidationStatusDraft).Return(nil)

	_, err := s.svc.CreateAssociationLink(ctx, "valid-cat", "server", "inst1", "inst2", "uses-model")
	require.NoError(t, err)
	s.catalogRepo.AssertCalled(t, "UpdateValidationStatus", ctx, "cat2", models.ValidationStatusDraft)
}

// T-12.28: DeleteAssociationLink resets catalog validation status to draft
func TestT12_28_DeleteLinkResetsDraft(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "valid-cat").Return(&models.Catalog{
		ID: "cat2", Name: "valid-cat", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusValid,
	}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1", Name: "server"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.linkRepo.On("GetByID", ctx, "link1").Return(&models.AssociationLink{
		ID: "link1", SourceInstanceID: "inst1",
	}, nil)
	s.instRepo.On("GetByID", ctx, "inst1").Return(&models.EntityInstance{
		ID: "inst1", EntityTypeID: "et1", CatalogID: "cat2",
	}, nil)
	s.linkRepo.On("Delete", ctx, "link1").Return(nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat2", models.ValidationStatusDraft).Return(nil)

	err := s.svc.DeleteAssociationLink(ctx, "valid-cat", "server", "link1")
	require.NoError(t, err)
	s.catalogRepo.AssertCalled(t, "UpdateValidationStatus", ctx, "cat2", models.ValidationStatusDraft)
}

// === Forward/Reverse Reference Service Tests ===

// T-12.29: GetForwardReferences returns resolved target info
func TestT12_29_GetForwardReferences(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1", Name: "server"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	// Source instance
	s.instRepo.On("GetByID", ctx, "inst1").Return(&models.EntityInstance{
		ID: "inst1", EntityTypeID: "et1", CatalogID: "cat1", Version: 1,
	}, nil)
	// Forward refs return links
	s.linkRepo.On("GetForwardRefs", ctx, "inst1").Return([]*models.AssociationLink{
		{ID: "link1", AssociationID: "assoc1", SourceInstanceID: "inst1", TargetInstanceID: "inst2"},
	}, nil)
	// Resolve association
	s.assocRepo.On("GetByID", ctx, "assoc1").Return(&models.Association{
		ID: "assoc1", Name: "uses-model", Type: models.AssociationTypeDirectional, TargetEntityTypeID: "et2",
	}, nil)
	// Resolve target instance
	s.instRepo.On("GetByID", ctx, "inst2").Return(&models.EntityInstance{
		ID: "inst2", EntityTypeID: "et2", CatalogID: "cat1", Name: "my-model", Version: 1,
	}, nil)
	// Resolve target entity type name
	s.etRepo.On("GetByID", ctx, "et2").Return(&models.EntityType{ID: "et2", Name: "model"}, nil)

	refs, err := s.svc.GetForwardReferences(ctx, "my-catalog", "server", "inst1")
	require.NoError(t, err)
	assert.Len(t, refs, 1)
	assert.Equal(t, "link1", refs[0].LinkID)
	assert.Equal(t, "uses-model", refs[0].AssociationName)
	assert.Equal(t, string(models.AssociationTypeDirectional), refs[0].AssociationType)
	assert.Equal(t, "inst2", refs[0].InstanceID)
	assert.Equal(t, "my-model", refs[0].InstanceName)
	assert.Equal(t, "model", refs[0].EntityTypeName)
}

// T-12.32: GetForwardReferences for instance with no links
func TestT12_32_ForwardRefsEmpty(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1", Name: "server"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "inst1").Return(&models.EntityInstance{
		ID: "inst1", EntityTypeID: "et1", CatalogID: "cat1", Version: 1,
	}, nil)
	s.linkRepo.On("GetForwardRefs", ctx, "inst1").Return([]*models.AssociationLink{}, nil)

	refs, err := s.svc.GetForwardReferences(ctx, "my-catalog", "server", "inst1")
	require.NoError(t, err)
	assert.Len(t, refs, 0)
}

// T-12.33: GetReverseReferences returns resolved source info
func TestT12_33_GetReverseReferences(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.etRepo.On("GetByName", ctx, "model").Return(&models.EntityType{ID: "et2", Name: "model"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin2", CatalogVersionID: "cv1", EntityTypeVersionID: "etv2"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv2").Return(&models.EntityTypeVersion{ID: "etv2", EntityTypeID: "et2", Version: 1}, nil)
	// Target instance (being referenced)
	s.instRepo.On("GetByID", ctx, "inst2").Return(&models.EntityInstance{
		ID: "inst2", EntityTypeID: "et2", CatalogID: "cat1", Name: "my-model", Version: 1,
	}, nil)
	// Reverse refs
	s.linkRepo.On("GetReverseRefs", ctx, "inst2").Return([]*models.AssociationLink{
		{ID: "link1", AssociationID: "assoc1", SourceInstanceID: "inst1", TargetInstanceID: "inst2"},
	}, nil)
	s.assocRepo.On("GetByID", ctx, "assoc1").Return(&models.Association{
		ID: "assoc1", Name: "uses-model", Type: models.AssociationTypeDirectional, TargetEntityTypeID: "et2",
	}, nil)
	// Resolve source instance
	s.instRepo.On("GetByID", ctx, "inst1").Return(&models.EntityInstance{
		ID: "inst1", EntityTypeID: "et1", CatalogID: "cat1", Name: "my-server", Version: 1,
	}, nil)
	s.etRepo.On("GetByID", ctx, "et1").Return(&models.EntityType{ID: "et1", Name: "server"}, nil)

	refs, err := s.svc.GetReverseReferences(ctx, "my-catalog", "model", "inst2")
	require.NoError(t, err)
	assert.Len(t, refs, 1)
	assert.Equal(t, "link1", refs[0].LinkID)
	assert.Equal(t, "uses-model", refs[0].AssociationName)
	assert.Equal(t, "inst1", refs[0].InstanceID)
	assert.Equal(t, "my-server", refs[0].InstanceName)
	assert.Equal(t, "server", refs[0].EntityTypeName)
}

// T-12.36: GetReverseReferences for instance with no incoming links
func TestT12_36_ReverseRefsEmpty(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.etRepo.On("GetByName", ctx, "model").Return(&models.EntityType{ID: "et2", Name: "model"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin2", CatalogVersionID: "cv1", EntityTypeVersionID: "etv2"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv2").Return(&models.EntityTypeVersion{ID: "etv2", EntityTypeID: "et2", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "inst2").Return(&models.EntityInstance{
		ID: "inst2", EntityTypeID: "et2", CatalogID: "cat1", Version: 1,
	}, nil)
	s.linkRepo.On("GetReverseRefs", ctx, "inst2").Return([]*models.AssociationLink{}, nil)

	refs, err := s.svc.GetReverseReferences(ctx, "my-catalog", "model", "inst2")
	require.NoError(t, err)
	assert.Len(t, refs, 0)
}

// Carry-forward: update with non-empty previous values
func TestCov_UpdateInstance_CarryForward(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.mockAttributes(ctx)

	s.instRepo.On("GetByID", ctx, "i1").Return(&models.EntityInstance{
		ID: "i1", EntityTypeID: "et1", CatalogID: "cat1", Version: 1,
	}, nil)
	s.instRepo.On("Update", ctx, mock.Anything).Return(nil)
	// Previous version has hostname="old" and port=80
	oldPort := float64(80)
	s.iavRepo.On("GetValuesForVersion", ctx, "i1", 1).Return([]*models.InstanceAttributeValue{
		{AttributeID: "a1", ValueString: "old-host", InstanceVersion: 1},
		{AttributeID: "a2", ValueNumber: &oldPort, InstanceVersion: 1},
	}, nil)
	// Only updating hostname — port should be carried forward
	s.iavRepo.On("SetValues", ctx, mock.MatchedBy(func(vals []*models.InstanceAttributeValue) bool {
		// Should have 2 values: new hostname + carried-forward port
		return len(vals) == 2
	})).Return(nil)
	s.iavRepo.On("GetCurrentValues", ctx, mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft).Return(nil)

	_, err := s.svc.UpdateInstance(ctx, "my-catalog", "model", "i1", 1, nil, nil, map[string]any{
		"hostname": "new-host",
	})
	require.NoError(t, err)
	s.iavRepo.AssertCalled(t, "SetValues", ctx, mock.MatchedBy(func(vals []*models.InstanceAttributeValue) bool {
		return len(vals) == 2
	}))
}

// === Coverage: Containment & Links error paths ===

func TestCov_CreateContained_AssocRepoError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "cat").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "p1", EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "p1").Return(&models.EntityInstance{ID: "p1", EntityTypeID: "et1", CatalogID: "c1"}, nil)
	s.etRepo.On("GetByName", ctx, "tool").Return(&models.EntityType{ID: "et2"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "p1", EntityTypeVersionID: "etv1"}, {ID: "p2", EntityTypeVersionID: "etv2"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv2").Return(&models.EntityTypeVersion{ID: "etv2", EntityTypeID: "et2", Version: 1}, nil)
	s.assocRepo.On("ListByVersion", ctx, "etv1").Return(nil, domainerrors.NewValidation("db error"))
	_, err := s.svc.CreateContainedInstance(ctx, "cat", "server", "p1", "tool", "child", "", nil)
	assert.Error(t, err)
}

func TestCov_CreateContained_CreateError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "cat").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1"}, nil)
	s.etRepo.On("GetByName", ctx, "tool").Return(&models.EntityType{ID: "et2"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "p1", EntityTypeVersionID: "etv1"}, {ID: "p2", EntityTypeVersionID: "etv2"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.etvRepo.On("GetByID", ctx, "etv2").Return(&models.EntityTypeVersion{ID: "etv2", EntityTypeID: "et2", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "p1").Return(&models.EntityInstance{ID: "p1", EntityTypeID: "et1", CatalogID: "c1"}, nil)
	s.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{
		{ID: "a1", TargetEntityTypeID: "et2", Type: models.AssociationTypeContainment},
	}, nil)
	s.attrRepo.On("ListByVersion", ctx, "etv2").Return([]*models.Attribute{}, nil)
	s.instRepo.On("Create", ctx, mock.Anything).Return(domainerrors.NewValidation("db error"))
	_, err := s.svc.CreateContainedInstance(ctx, "cat", "server", "p1", "tool", "child", "", nil)
	assert.Error(t, err)
}

func TestCov_ListContained_ParentNotFound(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "cat").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "p1", EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "nope").Return(nil, domainerrors.NewNotFound("EntityInstance", "nope"))
	_, _, err := s.svc.ListContainedInstances(ctx, "cat", "server", "nope", "tool", models.ListParams{Limit: 20})
	assert.Error(t, err)
}

func TestCov_ListContained_ChildResolveError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "cat").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "p1", EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "p1").Return(&models.EntityInstance{ID: "p1", EntityTypeID: "et1", CatalogID: "c1"}, nil)
	s.etRepo.On("GetByName", ctx, "bad").Return(nil, domainerrors.NewNotFound("EntityType", "bad"))
	_, _, err := s.svc.ListContainedInstances(ctx, "cat", "server", "p1", "bad", models.ListParams{Limit: 20})
	assert.Error(t, err)
}

func TestCov_ListContained_ListByParentError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "cat").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1"}, nil)
	s.etRepo.On("GetByName", ctx, "tool").Return(&models.EntityType{ID: "et2"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "p1", EntityTypeVersionID: "etv1"}, {ID: "p2", EntityTypeVersionID: "etv2"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.etvRepo.On("GetByID", ctx, "etv2").Return(&models.EntityTypeVersion{ID: "etv2", EntityTypeID: "et2", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "p1").Return(&models.EntityInstance{ID: "p1", EntityTypeID: "et1", CatalogID: "c1"}, nil)
	s.instRepo.On("ListByParent", ctx, "p1", mock.Anything).Return(nil, 0, domainerrors.NewValidation("db error"))
	_, _, err := s.svc.ListContainedInstances(ctx, "cat", "server", "p1", "tool", models.ListParams{Limit: 20})
	assert.Error(t, err)
}

func TestCov_CreateLink_AssocRepoError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "cat").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "p1", EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "i1").Return(&models.EntityInstance{ID: "i1", EntityTypeID: "et1", CatalogID: "c1"}, nil)
	s.instRepo.On("GetByID", ctx, "i2").Return(&models.EntityInstance{ID: "i2", EntityTypeID: "et2", CatalogID: "c1"}, nil)
	s.assocRepo.On("ListByVersion", ctx, "etv1").Return(([]*models.Association)(nil), domainerrors.NewValidation("db error"))
	_, err := s.svc.CreateAssociationLink(ctx, "cat", "server", "i1", "i2", "uses")
	assert.Error(t, err)
}

func TestCov_GetForwardRefs_LinkRepoError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "cat").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "p1", EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "i1").Return(&models.EntityInstance{ID: "i1", EntityTypeID: "et1", CatalogID: "c1"}, nil)
	s.linkRepo.On("GetForwardRefs", ctx, "i1").Return([]*models.AssociationLink{}, domainerrors.NewValidation("db error"))
	_, err := s.svc.GetForwardReferences(ctx, "cat", "server", "i1")
	assert.Error(t, err)
}

func TestCov_ResolveLinks_AssocError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "cat").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "p1", EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "i1").Return(&models.EntityInstance{ID: "i1", EntityTypeID: "et1", CatalogID: "c1"}, nil)
	s.linkRepo.On("GetForwardRefs", ctx, "i1").Return([]*models.AssociationLink{
		{ID: "l1", AssociationID: "a1", SourceInstanceID: "i1", TargetInstanceID: "i2"},
	}, nil)
	s.assocRepo.On("GetByID", ctx, "a1").Return(nil, domainerrors.NewNotFound("Association", "a1"))
	_, err := s.svc.GetForwardReferences(ctx, "cat", "server", "i1")
	assert.Error(t, err)
}

func TestCov_ResolveLinks_InstanceError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "cat").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "p1", EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "i1").Return(&models.EntityInstance{ID: "i1", EntityTypeID: "et1", CatalogID: "c1"}, nil)
	s.linkRepo.On("GetForwardRefs", ctx, "i1").Return([]*models.AssociationLink{
		{ID: "l1", AssociationID: "a1", SourceInstanceID: "i1", TargetInstanceID: "i2"},
	}, nil)
	s.assocRepo.On("GetByID", ctx, "a1").Return(&models.Association{ID: "a1", Name: "x", Type: "directional"}, nil)
	s.instRepo.On("GetByID", ctx, "i2").Return(nil, domainerrors.NewNotFound("EntityInstance", "i2"))
	_, err := s.svc.GetForwardReferences(ctx, "cat", "server", "i1")
	assert.Error(t, err)
}

func TestCov_ResolveLinks_ETError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "cat").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "p1", EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "i1").Return(&models.EntityInstance{ID: "i1", EntityTypeID: "et1", CatalogID: "c1"}, nil)
	s.linkRepo.On("GetForwardRefs", ctx, "i1").Return([]*models.AssociationLink{
		{ID: "l1", AssociationID: "a1", SourceInstanceID: "i1", TargetInstanceID: "i2"},
	}, nil)
	s.assocRepo.On("GetByID", ctx, "a1").Return(&models.Association{ID: "a1", Name: "x", Type: "directional"}, nil)
	s.instRepo.On("GetByID", ctx, "i2").Return(&models.EntityInstance{ID: "i2", EntityTypeID: "et-bad"}, nil)
	s.etRepo.On("GetByID", ctx, "et-bad").Return(nil, domainerrors.NewNotFound("EntityType", "et-bad"))
	_, err := s.svc.GetForwardReferences(ctx, "cat", "server", "i1")
	assert.Error(t, err)
}

func TestCov_DeleteLink_SourceInstanceError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "cat").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "p1", EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.linkRepo.On("GetByID", ctx, "l1").Return(&models.AssociationLink{ID: "l1", SourceInstanceID: "i-gone"}, nil)
	s.instRepo.On("GetByID", ctx, "i-gone").Return(nil, domainerrors.NewNotFound("EntityInstance", "i-gone"))
	err := s.svc.DeleteAssociationLink(ctx, "cat", "server", "l1")
	assert.Error(t, err)
}

func TestCov_CascadeDelete_LinkDeleteError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.instRepo.On("GetByID", ctx, "i1").Return(&models.EntityInstance{ID: "i1", CatalogID: "cat1"}, nil)
	s.instRepo.On("ListByParent", ctx, "i1", mock.Anything).Return([]*models.EntityInstance{}, 0, nil)
	s.linkRepo.On("DeleteByInstance", ctx, "i1").Return(domainerrors.NewValidation("db error"))
	err := s.svc.DeleteInstance(ctx, "my-catalog", "model", "i1")
	assert.Error(t, err)
}

func TestCov_CreateLink_GetForwardRefsError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "cat").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "p1", EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "i1").Return(&models.EntityInstance{ID: "i1", EntityTypeID: "et1", CatalogID: "c1"}, nil)
	s.instRepo.On("GetByID", ctx, "i2").Return(&models.EntityInstance{ID: "i2", EntityTypeID: "et2", CatalogID: "c1"}, nil)
	s.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{
		{ID: "a1", TargetEntityTypeID: "et2", Type: models.AssociationTypeDirectional, Name: "uses"},
	}, nil)
	s.linkRepo.On("GetForwardRefs", ctx, "i1").Return([]*models.AssociationLink{}, domainerrors.NewValidation("db error"))
	_, err := s.svc.CreateAssociationLink(ctx, "cat", "server", "i1", "i2", "uses")
	assert.Error(t, err)
}

// Coverage: CreateContainedInstance — resolveEntityType error (parent)
func TestCov_CreateContained_ResolveParentError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "bad").Return(nil, domainerrors.NewNotFound("Catalog", "bad"))
	_, err := s.svc.CreateContainedInstance(ctx, "bad", "server", "p1", "tool", "child", "", nil)
	assert.Error(t, err)
}

// Coverage: CreateContainedInstance — attr validation error
func TestCov_CreateContained_AttrValidationError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "cat").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1"}, nil)
	s.etRepo.On("GetByName", ctx, "tool").Return(&models.EntityType{ID: "et2"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "p1", EntityTypeVersionID: "etv1"}, {ID: "p2", EntityTypeVersionID: "etv2"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.etvRepo.On("GetByID", ctx, "etv2").Return(&models.EntityTypeVersion{ID: "etv2", EntityTypeID: "et2", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "p1").Return(&models.EntityInstance{ID: "p1", EntityTypeID: "et1", CatalogID: "c1"}, nil)
	s.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{
		{ID: "a1", TargetEntityTypeID: "et2", Type: models.AssociationTypeContainment},
	}, nil)
	// Attribute validation will fail — unknown attribute
	s.attrRepo.On("ListByVersion", ctx, "etv2").Return([]*models.Attribute{}, nil)
	s.instRepo.On("Create", ctx, mock.Anything).Return(nil)
	_, err := s.svc.CreateContainedInstance(ctx, "cat", "server", "p1", "tool", "child", "", map[string]any{"bad": "val"})
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

// Coverage: CreateContainedInstance — SetValues error
func TestCov_CreateContained_SetValuesError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "cat").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1"}, nil)
	s.etRepo.On("GetByName", ctx, "tool").Return(&models.EntityType{ID: "et2"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "p1", EntityTypeVersionID: "etv1"}, {ID: "p2", EntityTypeVersionID: "etv2"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.etvRepo.On("GetByID", ctx, "etv2").Return(&models.EntityTypeVersion{ID: "etv2", EntityTypeID: "et2", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "p1").Return(&models.EntityInstance{ID: "p1", EntityTypeID: "et1", CatalogID: "c1"}, nil)
	s.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{
		{ID: "a1", TargetEntityTypeID: "et2", Type: models.AssociationTypeContainment},
	}, nil)
	s.attrRepo.On("ListByVersion", ctx, "etv2").Return([]*models.Attribute{
		{ID: "a1", Name: "desc", TypeDefinitionVersionID: "tdv-string"},
	}, nil)
	s.tdvRepo.On("GetByID", ctx, "tdv-string").Return(&models.TypeDefinitionVersion{ID: "tdv-string", TypeDefinitionID: "td-string"}, nil)
	s.tdRepo.On("GetByID", ctx, "td-string").Return(&models.TypeDefinition{ID: "td-string", BaseType: models.BaseTypeString}, nil)
	s.instRepo.On("Create", ctx, mock.Anything).Return(nil)
	s.iavRepo.On("SetValues", ctx, mock.Anything).Return(domainerrors.NewValidation("db error"))
	_, err := s.svc.CreateContainedInstance(ctx, "cat", "server", "p1", "tool", "child", "", map[string]any{"desc": "val"})
	assert.Error(t, err)
}

// Coverage: CreateContainedInstance — resolveAttributeValues error
func TestCov_CreateContained_ResolveAttrsError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "cat").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1"}, nil)
	s.etRepo.On("GetByName", ctx, "tool").Return(&models.EntityType{ID: "et2"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "p1", EntityTypeVersionID: "etv1"}, {ID: "p2", EntityTypeVersionID: "etv2"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.etvRepo.On("GetByID", ctx, "etv2").Return(&models.EntityTypeVersion{ID: "etv2", EntityTypeID: "et2", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "p1").Return(&models.EntityInstance{ID: "p1", EntityTypeID: "et1", CatalogID: "c1"}, nil)
	s.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{
		{ID: "a1", TargetEntityTypeID: "et2", Type: models.AssociationTypeContainment},
	}, nil)
	// First ListByVersion call succeeds (for validateAndBuild), second fails (for resolveAttrs)
	s.attrRepo.On("ListByVersion", ctx, "etv2").Return(([]*models.Attribute)(nil), domainerrors.NewValidation("db error")).Once()
	s.instRepo.On("Create", ctx, mock.Anything).Return(nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusDraft).Return(nil)
	_, err := s.svc.CreateContainedInstance(ctx, "cat", "server", "p1", "tool", "child", "", nil)
	assert.Error(t, err)
}

// Coverage: ListContainedInstances — resolveEntityType error (parent)
func TestCov_ListContained_ResolveParentError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "bad").Return(nil, domainerrors.NewNotFound("Catalog", "bad"))
	_, _, err := s.svc.ListContainedInstances(ctx, "bad", "server", "p1", "tool", models.ListParams{Limit: 20})
	assert.Error(t, err)
}

// Coverage: ListContainedInstances — attrRepo error
func TestCov_ListContained_AttrRepoError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "cat").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1"}, nil)
	s.etRepo.On("GetByName", ctx, "tool").Return(&models.EntityType{ID: "et2"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "p1", EntityTypeVersionID: "etv1"}, {ID: "p2", EntityTypeVersionID: "etv2"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.etvRepo.On("GetByID", ctx, "etv2").Return(&models.EntityTypeVersion{ID: "etv2", EntityTypeID: "et2", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "p1").Return(&models.EntityInstance{ID: "p1", EntityTypeID: "et1", CatalogID: "c1"}, nil)
	s.instRepo.On("ListByParent", ctx, "p1", mock.Anything).Return([]*models.EntityInstance{
		{ID: "c1", EntityTypeID: "et2"},
	}, 1, nil)
	s.attrRepo.On("ListByVersion", ctx, "etv2").Return(([]*models.Attribute)(nil), domainerrors.NewValidation("db error"))
	_, _, err := s.svc.ListContainedInstances(ctx, "cat", "server", "p1", "tool", models.ListParams{Limit: 20})
	assert.Error(t, err)
}

// Coverage: ListContainedInstances — iavRepo.GetCurrentValues error
func TestCov_ListContained_GetValuesError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "cat").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1"}, nil)
	s.etRepo.On("GetByName", ctx, "tool").Return(&models.EntityType{ID: "et2"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "p1", EntityTypeVersionID: "etv1"}, {ID: "p2", EntityTypeVersionID: "etv2"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.etvRepo.On("GetByID", ctx, "etv2").Return(&models.EntityTypeVersion{ID: "etv2", EntityTypeID: "et2", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "p1").Return(&models.EntityInstance{ID: "p1", EntityTypeID: "et1", CatalogID: "c1"}, nil)
	s.instRepo.On("ListByParent", ctx, "p1", mock.Anything).Return([]*models.EntityInstance{
		{ID: "c1", EntityTypeID: "et2"},
	}, 1, nil)
	s.attrRepo.On("ListByVersion", ctx, "etv2").Return([]*models.Attribute{}, nil)
	s.iavRepo.On("GetCurrentValues", ctx, "c1").Return(nil, domainerrors.NewValidation("db error"))
	_, _, err := s.svc.ListContainedInstances(ctx, "cat", "server", "p1", "tool", models.ListParams{Limit: 20})
	assert.Error(t, err)
}

// Coverage: CreateAssociationLink — resolveEntityType error
func TestCov_CreateLink_ResolveError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "bad").Return(nil, domainerrors.NewNotFound("Catalog", "bad"))
	_, err := s.svc.CreateAssociationLink(ctx, "bad", "server", "i1", "i2", "uses")
	assert.Error(t, err)
}

// Coverage: CreateAssociationLink — source instance error
func TestCov_CreateLink_SourceError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "cat").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "p1", EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "i1").Return(nil, domainerrors.NewNotFound("EntityInstance", "i1"))
	_, err := s.svc.CreateAssociationLink(ctx, "cat", "server", "i1", "i2", "uses")
	assert.Error(t, err)
}

// Coverage: CreateAssociationLink — source not in catalog
func TestCov_CreateLink_SourceWrongCatalog(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "cat").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "p1", EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "i1").Return(&models.EntityInstance{ID: "i1", EntityTypeID: "et1", CatalogID: "other"}, nil)
	_, err := s.svc.CreateAssociationLink(ctx, "cat", "server", "i1", "i2", "uses")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

// Coverage: CreateAssociationLink — linkRepo.Create error
func TestCov_CreateLink_LinkRepoCreateError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "cat").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "p1", EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "i1").Return(&models.EntityInstance{ID: "i1", EntityTypeID: "et1", CatalogID: "c1"}, nil)
	s.instRepo.On("GetByID", ctx, "i2").Return(&models.EntityInstance{ID: "i2", EntityTypeID: "et2", CatalogID: "c1"}, nil)
	s.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{
		{ID: "a1", TargetEntityTypeID: "et2", Type: models.AssociationTypeDirectional, Name: "uses"},
	}, nil)
	s.linkRepo.On("GetForwardRefs", ctx, "i1").Return([]*models.AssociationLink{}, nil)
	s.linkRepo.On("Create", ctx, mock.Anything).Return(domainerrors.NewValidation("db error"))
	_, err := s.svc.CreateAssociationLink(ctx, "cat", "server", "i1", "i2", "uses")
	assert.Error(t, err)
}

// Coverage: DeleteAssociationLink — resolveEntityType error
func TestCov_DeleteLink_ResolveError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "bad").Return(nil, domainerrors.NewNotFound("Catalog", "bad"))
	err := s.svc.DeleteAssociationLink(ctx, "bad", "server", "l1")
	assert.Error(t, err)
}

// Coverage: GetForwardReferences — resolveEntityType error
func TestCov_GetForwardRefs_ResolveError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "bad").Return(nil, domainerrors.NewNotFound("Catalog", "bad"))
	_, err := s.svc.GetForwardReferences(ctx, "bad", "server", "i1")
	assert.Error(t, err)
}

// Coverage: GetForwardReferences — instRepo error
func TestCov_GetForwardRefs_InstanceError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "cat").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "p1", EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "i1").Return(nil, domainerrors.NewNotFound("EntityInstance", "i1"))
	_, err := s.svc.GetForwardReferences(ctx, "cat", "server", "i1")
	assert.Error(t, err)
}

// Coverage: GetReverseReferences — all paths (resolveEntityType, instRepo, linkRepo)
func TestCov_GetReverseRefs_ResolveError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "bad").Return(nil, domainerrors.NewNotFound("Catalog", "bad"))
	_, err := s.svc.GetReverseReferences(ctx, "bad", "server", "i1")
	assert.Error(t, err)
}

func TestCov_GetReverseRefs_InstanceError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "cat").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "p1", EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "i1").Return(nil, domainerrors.NewNotFound("EntityInstance", "i1"))
	_, err := s.svc.GetReverseReferences(ctx, "cat", "server", "i1")
	assert.Error(t, err)
}

func TestCov_GetReverseRefs_LinkRepoError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "cat").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "p1", EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "i1").Return(&models.EntityInstance{ID: "i1", EntityTypeID: "et1", CatalogID: "c1"}, nil)
	s.linkRepo.On("GetReverseRefs", ctx, "i1").Return([]*models.AssociationLink{}, domainerrors.NewValidation("db error"))
	_, err := s.svc.GetReverseReferences(ctx, "cat", "server", "i1")
	assert.Error(t, err)
}

// Bug: empty enum value should be skipped (optional attribute not set)
func TestBug_EmptyEnumValueSkipped(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.mockAttributes(ctx)

	s.instRepo.On("Create", ctx, mock.Anything).Return(nil)
	s.iavRepo.On("SetValues", ctx, mock.MatchedBy(func(vals []*models.InstanceAttributeValue) bool {
		// Empty enum should NOT be in the values list
		for _, v := range vals {
			if v.AttributeID == "a3" { // status (enum) attribute
				return false // should not be present
			}
		}
		return true
	})).Return(nil)
	s.iavRepo.On("GetCurrentValues", ctx, mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft).Return(nil)

	// Pass empty string for enum attribute — should succeed (skip it)
	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", map[string]any{
		"hostname": "myhost",
		"status":   "", // empty enum — should be skipped, not validated
	})
	require.NoError(t, err)
}

// === SetParent (reparent instance) ===

// SetParent with valid containment association
func TestSetParent_Valid(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusDraft,
	}, nil)
	s.etRepo.On("GetByName", ctx, "tool").Return(&models.EntityType{ID: "et2", Name: "tool"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
		{ID: "pin2", CatalogVersionID: "cv1", EntityTypeVersionID: "etv2"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.etvRepo.On("GetByID", ctx, "etv2").Return(&models.EntityTypeVersion{ID: "etv2", EntityTypeID: "et2", Version: 1}, nil)
	// Child instance (tool, currently no parent)
	s.instRepo.On("GetByID", ctx, "child1").Return(&models.EntityInstance{
		ID: "child1", EntityTypeID: "et2", CatalogID: "cat1", Name: "my-tool", Version: 1,
	}, nil)
	// Parent instance (server)
	s.instRepo.On("GetByID", ctx, "parent1").Return(&models.EntityInstance{
		ID: "parent1", EntityTypeID: "et1", CatalogID: "cat1", Name: "my-server", Version: 1,
	}, nil)
	// Containment association: server (et1) contains tool (et2) — check on parent's ETV
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1", Name: "server"}, nil)
	s.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{
		{ID: "assoc1", EntityTypeVersionID: "etv1", TargetEntityTypeID: "et2",
			Type: models.AssociationTypeContainment, Name: "tools"},
	}, nil)
	s.instRepo.On("Update", ctx, mock.Anything).Return(nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft).Return(nil)

	err := s.svc.SetParent(ctx, "my-catalog", "tool", "child1", "server", "parent1")
	require.NoError(t, err)
	s.instRepo.AssertCalled(t, "Update", ctx, mock.MatchedBy(func(inst *models.EntityInstance) bool {
		return inst.ID == "child1" && inst.ParentInstanceID == "parent1"
	}))
}

// SetParent with no containment association → validation error
func TestSetParent_NoContainment(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.etRepo.On("GetByName", ctx, "tool").Return(&models.EntityType{ID: "et2", Name: "tool"}, nil)
	s.etRepo.On("GetByName", ctx, "model").Return(&models.EntityType{ID: "et3", Name: "model"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin2", CatalogVersionID: "cv1", EntityTypeVersionID: "etv2"},
		{ID: "pin3", CatalogVersionID: "cv1", EntityTypeVersionID: "etv3"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv2").Return(&models.EntityTypeVersion{ID: "etv2", EntityTypeID: "et2", Version: 1}, nil)
	s.etvRepo.On("GetByID", ctx, "etv3").Return(&models.EntityTypeVersion{ID: "etv3", EntityTypeID: "et3", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "child1").Return(&models.EntityInstance{
		ID: "child1", EntityTypeID: "et2", CatalogID: "cat1", Version: 1,
	}, nil)
	s.instRepo.On("GetByID", ctx, "parent1").Return(&models.EntityInstance{
		ID: "parent1", EntityTypeID: "et3", CatalogID: "cat1", Version: 1,
	}, nil)
	// No containment from model to tool
	s.assocRepo.On("ListByVersion", ctx, "etv3").Return([]*models.Association{}, nil)

	err := s.svc.SetParent(ctx, "my-catalog", "tool", "child1", "model", "parent1")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

// SetParent with empty parent ID → clears parent (unset)
func TestSetParent_ClearParent(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusDraft,
	}, nil)
	s.etRepo.On("GetByName", ctx, "tool").Return(&models.EntityType{ID: "et2", Name: "tool"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin2", CatalogVersionID: "cv1", EntityTypeVersionID: "etv2"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv2").Return(&models.EntityTypeVersion{ID: "etv2", EntityTypeID: "et2", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "child1").Return(&models.EntityInstance{
		ID: "child1", EntityTypeID: "et2", CatalogID: "cat1", ParentInstanceID: "old-parent", Version: 1,
	}, nil)
	s.instRepo.On("Update", ctx, mock.Anything).Return(nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft).Return(nil)

	err := s.svc.SetParent(ctx, "my-catalog", "tool", "child1", "", "")
	require.NoError(t, err)
	s.instRepo.AssertCalled(t, "Update", ctx, mock.MatchedBy(func(inst *models.EntityInstance) bool {
		return inst.ID == "child1" && inst.ParentInstanceID == ""
	}))
}

// SetParent error paths
func TestSetParent_ResolveError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "bad").Return(nil, domainerrors.NewNotFound("Catalog", "bad"))
	err := s.svc.SetParent(ctx, "bad", "tool", "c1", "server", "p1")
	assert.Error(t, err)
}

func TestSetParent_ChildNotFound(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.instRepo.On("GetByID", ctx, "nope").Return(nil, domainerrors.NewNotFound("EntityInstance", "nope"))
	err := s.svc.SetParent(ctx, "my-catalog", "model", "nope", "server", "p1")
	assert.Error(t, err)
}

func TestSetParent_ChildWrongCatalog(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "cat").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	s.etRepo.On("GetByName", ctx, "tool").Return(&models.EntityType{ID: "et2"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "p2", EntityTypeVersionID: "etv2"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv2").Return(&models.EntityTypeVersion{ID: "etv2", EntityTypeID: "et2", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "c1").Return(&models.EntityInstance{ID: "c1", CatalogID: "other"}, nil)
	err := s.svc.SetParent(ctx, "cat", "tool", "c1", "server", "p1")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

func TestSetParent_ClearUpdateError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "cat").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	s.etRepo.On("GetByName", ctx, "tool").Return(&models.EntityType{ID: "et2"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "p2", EntityTypeVersionID: "etv2"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv2").Return(&models.EntityTypeVersion{ID: "etv2", EntityTypeID: "et2", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "c1").Return(&models.EntityInstance{ID: "c1", CatalogID: "c1", ParentInstanceID: "old"}, nil)
	s.instRepo.On("Update", ctx, mock.Anything).Return(domainerrors.NewValidation("db error"))
	err := s.svc.SetParent(ctx, "cat", "tool", "c1", "", "")
	assert.Error(t, err)
}

func TestSetParent_ParentResolveError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "cat").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	s.etRepo.On("GetByName", ctx, "tool").Return(&models.EntityType{ID: "et2"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "p2", EntityTypeVersionID: "etv2"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv2").Return(&models.EntityTypeVersion{ID: "etv2", EntityTypeID: "et2", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "c1").Return(&models.EntityInstance{ID: "c1", EntityTypeID: "et2", CatalogID: "c1"}, nil)
	s.etRepo.On("GetByName", ctx, "bad-type").Return(nil, domainerrors.NewNotFound("EntityType", "bad-type"))
	err := s.svc.SetParent(ctx, "cat", "tool", "c1", "bad-type", "p1")
	assert.Error(t, err)
}

func TestSetParent_ParentNotFound(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "cat").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	s.etRepo.On("GetByName", ctx, "tool").Return(&models.EntityType{ID: "et2"}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "p1", EntityTypeVersionID: "etv1"}, {ID: "p2", EntityTypeVersionID: "etv2"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.etvRepo.On("GetByID", ctx, "etv2").Return(&models.EntityTypeVersion{ID: "etv2", EntityTypeID: "et2", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "c1").Return(&models.EntityInstance{ID: "c1", EntityTypeID: "et2", CatalogID: "c1"}, nil)
	s.instRepo.On("GetByID", ctx, "nope").Return(nil, domainerrors.NewNotFound("EntityInstance", "nope"))
	err := s.svc.SetParent(ctx, "cat", "tool", "c1", "server", "nope")
	assert.Error(t, err)
}

func TestSetParent_ParentWrongCatalog(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "cat").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	s.etRepo.On("GetByName", ctx, "tool").Return(&models.EntityType{ID: "et2"}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "p1", EntityTypeVersionID: "etv1"}, {ID: "p2", EntityTypeVersionID: "etv2"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.etvRepo.On("GetByID", ctx, "etv2").Return(&models.EntityTypeVersion{ID: "etv2", EntityTypeID: "et2", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "c1").Return(&models.EntityInstance{ID: "c1", EntityTypeID: "et2", CatalogID: "c1"}, nil)
	s.instRepo.On("GetByID", ctx, "p1").Return(&models.EntityInstance{ID: "p1", EntityTypeID: "et1", CatalogID: "other"}, nil)
	err := s.svc.SetParent(ctx, "cat", "tool", "c1", "server", "p1")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

func TestSetParent_AssocRepoError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "cat").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	s.etRepo.On("GetByName", ctx, "tool").Return(&models.EntityType{ID: "et2"}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "p1", EntityTypeVersionID: "etv1"}, {ID: "p2", EntityTypeVersionID: "etv2"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.etvRepo.On("GetByID", ctx, "etv2").Return(&models.EntityTypeVersion{ID: "etv2", EntityTypeID: "et2", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "c1").Return(&models.EntityInstance{ID: "c1", EntityTypeID: "et2", CatalogID: "c1"}, nil)
	s.instRepo.On("GetByID", ctx, "p1").Return(&models.EntityInstance{ID: "p1", EntityTypeID: "et1", CatalogID: "c1"}, nil)
	s.assocRepo.On("ListByVersion", ctx, "etv1").Return(([]*models.Association)(nil), domainerrors.NewValidation("db error"))
	err := s.svc.SetParent(ctx, "cat", "tool", "c1", "server", "p1")
	assert.Error(t, err)
}

func TestSetParent_UpdateError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "cat").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	s.etRepo.On("GetByName", ctx, "tool").Return(&models.EntityType{ID: "et2"}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "p1", EntityTypeVersionID: "etv1"}, {ID: "p2", EntityTypeVersionID: "etv2"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.etvRepo.On("GetByID", ctx, "etv2").Return(&models.EntityTypeVersion{ID: "etv2", EntityTypeID: "et2", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "c1").Return(&models.EntityInstance{ID: "c1", EntityTypeID: "et2", CatalogID: "c1"}, nil)
	s.instRepo.On("GetByID", ctx, "p1").Return(&models.EntityInstance{ID: "p1", EntityTypeID: "et1", CatalogID: "c1"}, nil)
	s.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{
		{ID: "a1", TargetEntityTypeID: "et2", Type: models.AssociationTypeContainment},
	}, nil)
	s.instRepo.On("Update", ctx, mock.Anything).Return(domainerrors.NewValidation("db error"))
	err := s.svc.SetParent(ctx, "cat", "tool", "c1", "server", "p1")
	assert.Error(t, err)
}

// Coverage: string parsed as number (success case, line 186)
func TestCov_ValidateAttrs_StringAsNumber(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.mockAttributes(ctx)
	s.instRepo.On("Create", ctx, mock.Anything).Return(nil)
	s.iavRepo.On("SetValues", ctx, mock.MatchedBy(func(vals []*models.InstanceAttributeValue) bool {
		return len(vals) == 1 && vals[0].ValueNumber != nil && *vals[0].ValueNumber == 42.5
	})).Return(nil)
	s.iavRepo.On("GetCurrentValues", ctx, mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft).Return(nil)

	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", map[string]any{
		"port": "42.5", // string that parses as number
	})
	require.NoError(t, err)
}

// Coverage: CreateContainedInstance — assocRepo.ListByVersion error (line 428)
func TestCov_CreateContained_AssocListError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "cat").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1"}, nil)
	s.etRepo.On("GetByName", ctx, "tool").Return(&models.EntityType{ID: "et2"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "p1", EntityTypeVersionID: "etv1"}, {ID: "p2", EntityTypeVersionID: "etv2"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.etvRepo.On("GetByID", ctx, "etv2").Return(&models.EntityTypeVersion{ID: "etv2", EntityTypeID: "et2", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "p1").Return(&models.EntityInstance{ID: "p1", EntityTypeID: "et1", CatalogID: "c1"}, nil)
	s.assocRepo.On("ListByVersion", ctx, "etv1").Return(([]*models.Association)(nil), domainerrors.NewValidation("db error"))

	_, err := s.svc.CreateContainedInstance(ctx, "cat", "server", "p1", "tool", "child", "", nil)
	assert.Error(t, err)
}

// Coverage: DeleteAssociationLink — linkRepo.Delete error (line 631)
func TestCov_DeleteLink_DeleteError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "cat").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "p1", EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.linkRepo.On("GetByID", ctx, "l1").Return(&models.AssociationLink{ID: "l1", SourceInstanceID: "i1"}, nil)
	s.instRepo.On("GetByID", ctx, "i1").Return(&models.EntityInstance{ID: "i1", EntityTypeID: "et1", CatalogID: "c1"}, nil)
	s.linkRepo.On("Delete", ctx, "l1").Return(domainerrors.NewValidation("db error"))

	err := s.svc.DeleteAssociationLink(ctx, "cat", "server", "l1")
	assert.Error(t, err)
}

// === Quality Review Fixes ===

// H2: ListContainedInstances returns correct total (filtered count, not unfiltered)
func TestQR_H2_ListContainedCorrectTotal(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1", Name: "server"}, nil)
	s.etRepo.On("GetByName", ctx, "tool").Return(&models.EntityType{ID: "et2", Name: "tool"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
		{ID: "pin2", CatalogVersionID: "cv1", EntityTypeVersionID: "etv2"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.etvRepo.On("GetByID", ctx, "etv2").Return(&models.EntityTypeVersion{ID: "etv2", EntityTypeID: "et2", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "parent1").Return(&models.EntityInstance{
		ID: "parent1", EntityTypeID: "et1", CatalogID: "cat1", Version: 1,
	}, nil)
	// Parent has 2 tools and 1 other type
	s.instRepo.On("ListByParent", ctx, "parent1", mock.Anything).Return([]*models.EntityInstance{
		{ID: "c1", EntityTypeID: "et2", Name: "tool-a"},
		{ID: "c2", EntityTypeID: "et3", Name: "other"},
		{ID: "c3", EntityTypeID: "et2", Name: "tool-b"},
	}, 3, nil)
	s.attrRepo.On("ListByVersion", ctx, "etv2").Return([]*models.Attribute{}, nil)
	s.iavRepo.On("GetCurrentValues", ctx, mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)

	details, total, err := s.svc.ListContainedInstances(ctx, "my-catalog", "server", "parent1", "tool", models.ListParams{Limit: 20})
	require.NoError(t, err)
	assert.Len(t, details, 2)       // Only tools
	assert.Equal(t, 2, total)       // Total should match filtered count
}

// H3: cascadeDelete cleans up association links
func TestQR_H3_CascadeDeleteCleansLinks(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)

	s.instRepo.On("GetByID", ctx, "inst1").Return(&models.EntityInstance{ID: "inst1", CatalogID: "cat1"}, nil)
	s.instRepo.On("ListByParent", ctx, "inst1", mock.Anything).Return([]*models.EntityInstance{}, 0, nil)
	s.linkRepo.On("DeleteByInstance", ctx, "inst1").Return(nil)
	s.instRepo.On("SoftDelete", ctx, "inst1").Return(nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft).Return(nil)

	err := s.svc.DeleteInstance(ctx, "my-catalog", "model", "inst1")
	require.NoError(t, err)
	s.linkRepo.AssertCalled(t, "DeleteByInstance", ctx, "inst1")
}

// H4: DeleteAssociationLink verifies link ownership
func TestQR_H4_DeleteLinkVerifiesOwnership(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1", Name: "server"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	// Link belongs to an instance in a different catalog
	s.linkRepo.On("GetByID", ctx, "link1").Return(&models.AssociationLink{
		ID: "link1", SourceInstanceID: "inst-other",
	}, nil)
	s.instRepo.On("GetByID", ctx, "inst-other").Return(&models.EntityInstance{
		ID: "inst-other", EntityTypeID: "et1", CatalogID: "cat-other", // different catalog!
	}, nil)

	err := s.svc.DeleteAssociationLink(ctx, "my-catalog", "server", "link1")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

// M2: CreateContainedInstance verifies parent belongs to catalog
func TestQR_M2_ContainedVerifiesParentCatalog(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1", Name: "server"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	// Parent instance belongs to a different catalog
	s.instRepo.On("GetByID", ctx, "parent1").Return(&models.EntityInstance{
		ID: "parent1", EntityTypeID: "et1", CatalogID: "cat-other", Version: 1,
	}, nil)

	_, err := s.svc.CreateContainedInstance(ctx, "my-catalog", "server", "parent1", "tool", "child", "", nil)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

// M3: CreateAssociationLink verifies source and target in same catalog
func TestQR_M3_LinkVerifiesSameCatalog(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1", Name: "server"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "inst1").Return(&models.EntityInstance{
		ID: "inst1", EntityTypeID: "et1", CatalogID: "cat1", Version: 1,
	}, nil)
	// Target is in a different catalog
	s.instRepo.On("GetByID", ctx, "inst2").Return(&models.EntityInstance{
		ID: "inst2", EntityTypeID: "et2", CatalogID: "cat-other", Version: 1,
	}, nil)
	s.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{
		{ID: "assoc1", EntityTypeVersionID: "etv1", TargetEntityTypeID: "et2",
			Type: models.AssociationTypeDirectional, Name: "uses"},
	}, nil)

	_, err := s.svc.CreateAssociationLink(ctx, "my-catalog", "server", "inst1", "inst2", "uses")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

// M6: No duplicate link prevention
func TestQR_M6_DuplicateLinkPrevention(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusDraft,
	}, nil)
	s.etRepo.On("GetByName", ctx, "server").Return(&models.EntityType{ID: "et1", Name: "server"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	s.instRepo.On("GetByID", ctx, "inst1").Return(&models.EntityInstance{
		ID: "inst1", EntityTypeID: "et1", CatalogID: "cat1", Version: 1,
	}, nil)
	s.instRepo.On("GetByID", ctx, "inst2").Return(&models.EntityInstance{
		ID: "inst2", EntityTypeID: "et2", CatalogID: "cat1", Version: 1,
	}, nil)
	s.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{
		{ID: "assoc1", EntityTypeVersionID: "etv1", TargetEntityTypeID: "et2",
			Type: models.AssociationTypeDirectional, Name: "uses"},
	}, nil)
	// Existing link already present
	s.linkRepo.On("GetForwardRefs", ctx, "inst1").Return([]*models.AssociationLink{
		{ID: "existing-link", AssociationID: "assoc1", SourceInstanceID: "inst1", TargetInstanceID: "inst2"},
	}, nil)

	_, err := s.svc.CreateAssociationLink(ctx, "my-catalog", "server", "inst1", "inst2", "uses")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsConflict(err))
}

// === Containment Tree (T-13.04 through T-13.10) ===

func TestGetContainmentTree_BuildsCorrectTree(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)

	parent := &models.EntityInstance{ID: "p1", EntityTypeID: "et1", CatalogID: "cat1", Name: "parent"}
	child := &models.EntityInstance{ID: "c1", EntityTypeID: "et2", CatalogID: "cat1", ParentInstanceID: "p1", Name: "child"}

	s.instRepo.On("ListByCatalog", ctx, "cat1").Return([]*models.EntityInstance{parent, child}, nil)
	s.etRepo.On("GetByID", ctx, "et1").Return(&models.EntityType{ID: "et1", Name: "Server"}, nil)
	s.etRepo.On("GetByID", ctx, "et2").Return(&models.EntityType{ID: "et2", Name: "Tool"}, nil)

	tree, err := s.svc.GetContainmentTree(ctx, "my-catalog")
	require.NoError(t, err)
	require.Len(t, tree, 1)
	assert.Equal(t, "parent", tree[0].Instance.Name)
	assert.Equal(t, "Server", tree[0].EntityTypeName)
	require.Len(t, tree[0].Children, 1)
	assert.Equal(t, "child", tree[0].Children[0].Instance.Name)
	assert.Equal(t, "Tool", tree[0].Children[0].EntityTypeName)
}

func TestGetContainmentTree_EmptyCatalog(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "empty").Return(&models.Catalog{
		ID: "cat-empty", Name: "empty", CatalogVersionID: "cv1",
	}, nil)
	s.instRepo.On("ListByCatalog", ctx, "cat-empty").Return([]*models.EntityInstance{}, nil)

	tree, err := s.svc.GetContainmentTree(ctx, "empty")
	require.NoError(t, err)
	assert.Empty(t, tree)
}

func TestGetContainmentTree_NonexistentCatalog(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "nope").Return(nil, domainerrors.NewNotFound("Catalog", "nope"))

	_, err := s.svc.GetContainmentTree(ctx, "nope")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
}

func TestGetContainmentTree_MultiLevelNesting(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)

	root := &models.EntityInstance{ID: "r1", EntityTypeID: "et1", CatalogID: "cat1", Name: "root"}
	child := &models.EntityInstance{ID: "c1", EntityTypeID: "et2", CatalogID: "cat1", ParentInstanceID: "r1", Name: "child"}
	grandchild := &models.EntityInstance{ID: "g1", EntityTypeID: "et3", CatalogID: "cat1", ParentInstanceID: "c1", Name: "grandchild"}

	s.instRepo.On("ListByCatalog", ctx, "cat1").Return([]*models.EntityInstance{root, child, grandchild}, nil)
	s.etRepo.On("GetByID", ctx, "et1").Return(&models.EntityType{ID: "et1", Name: "Server"}, nil)
	s.etRepo.On("GetByID", ctx, "et2").Return(&models.EntityType{ID: "et2", Name: "Tool"}, nil)
	s.etRepo.On("GetByID", ctx, "et3").Return(&models.EntityType{ID: "et3", Name: "Prompt"}, nil)

	tree, err := s.svc.GetContainmentTree(ctx, "my-catalog")
	require.NoError(t, err)
	require.Len(t, tree, 1)
	require.Len(t, tree[0].Children, 1)
	require.Len(t, tree[0].Children[0].Children, 1)
	assert.Equal(t, "grandchild", tree[0].Children[0].Children[0].Instance.Name)
	assert.Equal(t, "Prompt", tree[0].Children[0].Children[0].EntityTypeName)
}

// === Parent Chain Resolution (T-13.50 through T-13.53) ===

func TestGetInstance_WithParentChain(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.mockPinResolution(ctx)

	childInst := &models.EntityInstance{
		ID: "child1", EntityTypeID: "et1", CatalogID: "cat1",
		ParentInstanceID: "parent1", Name: "child", Version: 1,
	}
	parentInst := &models.EntityInstance{
		ID: "parent1", EntityTypeID: "et1", CatalogID: "cat1",
		Name: "parent", Version: 1,
	}

	s.instRepo.On("GetByID", ctx, "child1").Return(childInst, nil)
	s.instRepo.On("GetByID", ctx, "parent1").Return(parentInst, nil)
	s.iavRepo.On("GetCurrentValues", ctx, "child1").Return([]*models.InstanceAttributeValue{}, nil)
	s.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{}, nil)
	s.etRepo.On("GetByID", ctx, "et1").Return(&models.EntityType{ID: "et1", Name: "model"}, nil)

	detail, err := s.svc.GetInstance(ctx, "my-catalog", "model", "child1")
	require.NoError(t, err)
	require.Len(t, detail.ParentChain, 1)
	assert.Equal(t, "parent1", detail.ParentChain[0].InstanceID)
	assert.Equal(t, "parent", detail.ParentChain[0].InstanceName)
	assert.Equal(t, "model", detail.ParentChain[0].EntityTypeName)
}

func TestGetInstance_RootInstance_EmptyChain(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.mockPinResolution(ctx)

	rootInst := &models.EntityInstance{
		ID: "root1", EntityTypeID: "et1", CatalogID: "cat1",
		Name: "root", Version: 1,
	}

	s.instRepo.On("GetByID", ctx, "root1").Return(rootInst, nil)
	s.iavRepo.On("GetCurrentValues", ctx, "root1").Return([]*models.InstanceAttributeValue{}, nil)
	s.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{}, nil)

	detail, err := s.svc.GetInstance(ctx, "my-catalog", "model", "root1")
	require.NoError(t, err)
	assert.Nil(t, detail.ParentChain)
}

func TestGetInstance_MultiLevelChain(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.mockPinResolution(ctx)

	grandchild := &models.EntityInstance{
		ID: "gc1", EntityTypeID: "et1", CatalogID: "cat1",
		ParentInstanceID: "c1", Name: "grandchild", Version: 1,
	}
	child := &models.EntityInstance{
		ID: "c1", EntityTypeID: "et1", CatalogID: "cat1",
		ParentInstanceID: "r1", Name: "child", Version: 1,
	}
	root := &models.EntityInstance{
		ID: "r1", EntityTypeID: "et1", CatalogID: "cat1",
		Name: "root", Version: 1,
	}

	s.instRepo.On("GetByID", ctx, "gc1").Return(grandchild, nil)
	s.instRepo.On("GetByID", ctx, "c1").Return(child, nil)
	s.instRepo.On("GetByID", ctx, "r1").Return(root, nil)
	s.iavRepo.On("GetCurrentValues", ctx, "gc1").Return([]*models.InstanceAttributeValue{}, nil)
	s.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{}, nil)
	s.etRepo.On("GetByID", ctx, "et1").Return(&models.EntityType{ID: "et1", Name: "model"}, nil)

	detail, err := s.svc.GetInstance(ctx, "my-catalog", "model", "gc1")
	require.NoError(t, err)
	require.Len(t, detail.ParentChain, 2)
	// Root-first order
	assert.Equal(t, "root", detail.ParentChain[0].InstanceName)
	assert.Equal(t, "child", detail.ParentChain[1].InstanceName)
}

// === ListInstances filter name→ID resolution ===

func TestListInstances_FilterResolvesNameToID(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)

	s.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{
		{ID: "attr-abc", Name: "hostname", TypeDefinitionVersionID: "tdv-string"},
	}, nil)
	s.tdvRepo.On("GetByID", ctx, "tdv-string").Return(&models.TypeDefinitionVersion{ID: "tdv-string", TypeDefinitionID: "td-string"}, nil)
	s.tdRepo.On("GetByID", ctx, "td-string").Return(&models.TypeDefinition{ID: "td-string", BaseType: models.BaseTypeString}, nil)
	// Expect the filter to use the attribute ID, not the name
	s.instRepo.On("List", ctx, "et1", "cat1", mock.MatchedBy(func(p models.ListParams) bool {
		return p.Filters["attr-abc"] == "hello"
	})).Return([]*models.EntityInstance{}, 0, nil)

	_, _, err := s.svc.ListInstances(ctx, "my-catalog", "model", models.ListParams{
		Limit:   20,
		Filters: map[string]string{"hostname": "hello"},
	})
	assert.NoError(t, err)
}

func TestListInstances_FilterUnknownAttrReturnsError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)

	s.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{
		{ID: "attr-abc", Name: "hostname", TypeDefinitionVersionID: "tdv-string"},
	}, nil)
	s.tdvRepo.On("GetByID", ctx, "tdv-string").Return(&models.TypeDefinitionVersion{ID: "tdv-string", TypeDefinitionID: "td-string"}, nil)
	s.tdRepo.On("GetByID", ctx, "td-string").Return(&models.TypeDefinition{ID: "td-string", BaseType: models.BaseTypeString}, nil)

	_, _, err := s.svc.ListInstances(ctx, "my-catalog", "model", models.ListParams{
		Limit:   20,
		Filters: map[string]string{"nonexistent": "value"},
	})
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

func TestListInstances_FilterMinMaxResolvesNameToID(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)

	s.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{
		{ID: "attr-num", Name: "score", TypeDefinitionVersionID: "tdv-number"},
	}, nil)
	s.tdvRepo.On("GetByID", ctx, "tdv-number").Return(&models.TypeDefinitionVersion{ID: "tdv-number", TypeDefinitionID: "td-number"}, nil)
	s.tdRepo.On("GetByID", ctx, "td-number").Return(&models.TypeDefinition{ID: "td-number", BaseType: models.BaseTypeNumber}, nil)
	s.instRepo.On("List", ctx, "et1", "cat1", mock.MatchedBy(func(p models.ListParams) bool {
		return p.Filters["attr-num.min"] == "5" && p.Filters["attr-num.max"] == "10"
	})).Return([]*models.EntityInstance{}, 0, nil)

	_, _, err := s.svc.ListInstances(ctx, "my-catalog", "model", models.ListParams{
		Limit:   20,
		Filters: map[string]string{"score.min": "5", "score.max": "10"},
	})
	assert.NoError(t, err)
}

// === GetContainmentTree entity type name fallback ===

func TestGetContainmentTree_MultipleRootsOrdered(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)

	root1 := &models.EntityInstance{ID: "r1", EntityTypeID: "et1", CatalogID: "cat1", Name: "root-a"}
	root2 := &models.EntityInstance{ID: "r2", EntityTypeID: "et1", CatalogID: "cat1", Name: "root-b"}
	child := &models.EntityInstance{ID: "c1", EntityTypeID: "et1", CatalogID: "cat1", ParentInstanceID: "r1", Name: "child-of-a"}

	s.instRepo.On("ListByCatalog", ctx, "cat1").Return([]*models.EntityInstance{root1, root2, child}, nil)
	s.etRepo.On("GetByID", ctx, "et1").Return(&models.EntityType{ID: "et1", Name: "Server"}, nil)

	tree, err := s.svc.GetContainmentTree(ctx, "my-catalog")
	require.NoError(t, err)
	require.Len(t, tree, 2) // Both roots present
	assert.Equal(t, "root-a", tree[0].Instance.Name)
	assert.Equal(t, "root-b", tree[1].Instance.Name)
	// child is under root-a
	require.Len(t, tree[0].Children, 1)
	assert.Equal(t, "child-of-a", tree[0].Children[0].Instance.Name)
	// root-b has no children
	assert.Len(t, tree[1].Children, 0)
}

func TestGetContainmentTree_EntityTypeNameCaching(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)

	// Two instances with the same EntityTypeID
	inst1 := &models.EntityInstance{ID: "i1", EntityTypeID: "et1", CatalogID: "cat1", Name: "inst-a"}
	inst2 := &models.EntityInstance{ID: "i2", EntityTypeID: "et1", CatalogID: "cat1", Name: "inst-b"}

	s.instRepo.On("ListByCatalog", ctx, "cat1").Return([]*models.EntityInstance{inst1, inst2}, nil)
	s.etRepo.On("GetByID", ctx, "et1").Return(&models.EntityType{ID: "et1", Name: "Server"}, nil)

	tree, err := s.svc.GetContainmentTree(ctx, "my-catalog")
	require.NoError(t, err)
	require.Len(t, tree, 2)
	assert.Equal(t, "Server", tree[0].EntityTypeName)
	assert.Equal(t, "Server", tree[1].EntityTypeName)

	// etRepo.GetByID("et1") should have been called exactly once (cached)
	s.etRepo.AssertNumberOfCalls(t, "GetByID", 1)
}

func TestGetContainmentTree_ETNameFallback(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.instRepo.On("ListByCatalog", ctx, "cat1").Return([]*models.EntityInstance{
		{ID: "p1", EntityTypeID: "et-unknown", CatalogID: "cat1", Name: "orphan"},
	}, nil)
	s.etRepo.On("GetByID", ctx, "et-unknown").Return(nil, domainerrors.NewNotFound("EntityType", "et-unknown"))

	tree, err := s.svc.GetContainmentTree(ctx, "my-catalog")
	require.NoError(t, err)
	require.Len(t, tree, 1)
	// Falls back to using the entity type ID as name
	assert.Equal(t, "et-unknown", tree[0].EntityTypeName)
}

// Bug fix: Clearing a required attribute value in draft mode should work.
// Sending an empty string for an attribute should clear it (not carry forward the old value).
func TestUpdateInstance_ClearAttributeValue(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.mockAttributes(ctx)

	s.instRepo.On("GetByID", ctx, "inst1").Return(&models.EntityInstance{
		ID: "inst1", EntityTypeID: "et1", CatalogID: "cat1", Name: "server-1", Version: 1,
	}, nil)
	s.instRepo.On("Update", ctx, mock.Anything).Return(nil)
	// Previous version had hostname = "web-01"
	s.iavRepo.On("GetValuesForVersion", ctx, "inst1", 1).Return([]*models.InstanceAttributeValue{
		{ID: "v1", InstanceID: "inst1", InstanceVersion: 1, AttributeID: "a1", ValueString: "web-01"},
	}, nil)
	// We expect SetValues to be called with NO hostname value (it was cleared)
	s.iavRepo.On("SetValues", ctx, mock.MatchedBy(func(vals []*models.InstanceAttributeValue) bool {
		for _, v := range vals {
			if v.AttributeID == "a1" {
				return false // hostname should NOT be carried forward
			}
		}
		return true
	})).Return(nil)
	s.iavRepo.On("GetCurrentValues", ctx, mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft).Return(nil)

	// Send empty string to clear hostname
	detail, err := s.svc.UpdateInstance(ctx, "my-catalog", "model", "inst1", 1, nil, nil, map[string]any{
		"hostname": "",
	})
	require.NoError(t, err)
	assert.Equal(t, 2, detail.Instance.Version)
	// Verify hostname was NOT carried forward
	for _, av := range detail.Attributes {
		if av.Name == "hostname" {
			assert.Nil(t, av.Value, "hostname should have been cleared, not carried forward")
		}
	}
}

// TD-20: Empty name validation
func TestCreateInstance_EmptyNameRejected(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "", "", nil)
	require.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
	assert.Contains(t, err.Error(), "name is required")
}

func TestCreateInstance_WhitespaceOnlyNameRejected(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "   ", "", nil)
	require.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

func TestCreateContainedInstance_EmptyNameRejected(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	_, err := s.svc.CreateContainedInstance(ctx, "my-catalog", "server", "p1", "tool", "", "", nil)
	require.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
	assert.Contains(t, err.Error(), "name is required")
}

// TD-30: Catalog ownership check on instance read/update/delete
func TestGetInstance_WrongCatalogReturnsNotFound(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{ID: "cat1", CatalogVersionID: "cv1"}, nil)
	s.etRepo.On("GetByName", ctx, "model").Return(&models.EntityType{ID: "et1", Name: "model"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1"}, nil)
	// Instance belongs to a different catalog
	s.instRepo.On("GetByID", ctx, "i1").Return(&models.EntityInstance{
		ID: "i1", CatalogID: "other-catalog-id", EntityTypeID: "et1",
	}, nil)

	_, err := s.svc.GetInstance(ctx, "my-catalog", "model", "i1")
	require.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
}

func TestGetInstance_ResolveAttrError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)

	s.instRepo.On("GetByID", ctx, "i1").Return(&models.EntityInstance{
		ID: "i1", CatalogID: "cat1", EntityTypeID: "et1", Version: 1,
	}, nil)
	s.attrRepo.On("ListByVersion", ctx, "etv1").Return(nil, fmt.Errorf("db error"))

	_, err := s.svc.GetInstance(ctx, "my-catalog", "model", "i1")
	assert.Error(t, err)
}

func TestDeleteInstance_GetByIDError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)

	s.instRepo.On("GetByID", ctx, "i1").Return(nil, domainerrors.NewNotFound("EntityInstance", "i1"))

	err := s.svc.DeleteInstance(ctx, "my-catalog", "model", "i1")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
}

func TestDeleteInstance_WrongCatalogReturnsNotFound(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{ID: "cat1", CatalogVersionID: "cv1"}, nil)
	s.etRepo.On("GetByName", ctx, "model").Return(&models.EntityType{ID: "et1", Name: "model"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1"}, nil)
	s.instRepo.On("GetByID", ctx, "i1").Return(&models.EntityInstance{
		ID: "i1", CatalogID: "other-catalog-id", EntityTypeID: "et1",
	}, nil)

	err := s.svc.DeleteInstance(ctx, "my-catalog", "model", "i1")
	require.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
}

// === Coverage: remaining error paths in service/operational ===

// GetInstance: resolveParentChain error (line 317)
func TestGetInstance_ParentChainError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.mockAttributes(ctx)

	s.instRepo.On("GetByID", ctx, "i1").Return(&models.EntityInstance{
		ID: "i1", CatalogID: "cat1", EntityTypeID: "et1", Version: 1,
		ParentInstanceID: "parent1",
	}, nil)
	s.iavRepo.On("GetCurrentValues", ctx, "i1").Return([]*models.InstanceAttributeValue{}, nil)
	// Parent lookup fails
	s.instRepo.On("GetByID", ctx, "parent1").Return(nil, fmt.Errorf("parent not found"))

	_, err := s.svc.GetInstance(ctx, "my-catalog", "model", "i1")
	assert.Error(t, err)
}

// UpdateInstance: instRepo.Update error (line 442)
func TestUpdateInstance_RepoUpdateError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)

	s.instRepo.On("GetByID", ctx, "i1").Return(&models.EntityInstance{
		ID: "i1", CatalogID: "cat1", EntityTypeID: "et1", Version: 1,
	}, nil)
	s.iavRepo.On("GetValuesForVersion", ctx, "i1", 1).Return([]*models.InstanceAttributeValue{}, nil)
	s.instRepo.On("Update", ctx, mock.Anything).Return(fmt.Errorf("update failed"))

	newName := "new-name"
	_, err := s.svc.UpdateInstance(ctx, "my-catalog", "model", "i1", 1, &newName, nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update failed")
}

// UpdateInstance: iavRepo.SetValues error (line 447)
func TestUpdateInstance_SetValuesError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.mockAttributes(ctx)

	s.instRepo.On("GetByID", ctx, "i1").Return(&models.EntityInstance{
		ID: "i1", CatalogID: "cat1", EntityTypeID: "et1", Version: 1,
	}, nil)
	s.iavRepo.On("GetValuesForVersion", ctx, "i1", 1).Return([]*models.InstanceAttributeValue{}, nil)
	s.instRepo.On("Update", ctx, mock.Anything).Return(nil)
	s.iavRepo.On("SetValues", ctx, mock.Anything).Return(fmt.Errorf("set values failed"))

	_, err := s.svc.UpdateInstance(ctx, "my-catalog", "model", "i1", 1, nil, nil, map[string]any{"hostname": "h"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "set values failed")
}

// UpdateInstance: GetValuesForVersion error (line 415)
func TestUpdateInstance_GetPrevValuesError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)

	s.instRepo.On("GetByID", ctx, "i1").Return(&models.EntityInstance{
		ID: "i1", CatalogID: "cat1", EntityTypeID: "et1", Version: 1,
	}, nil)
	s.iavRepo.On("GetValuesForVersion", ctx, "i1", 1).Return(nil, fmt.Errorf("prev values error"))

	newName := "x"
	_, err := s.svc.UpdateInstance(ctx, "my-catalog", "model", "i1", 1, &newName, nil, nil)
	assert.Error(t, err)
}

// GetContainmentTree: ListByCatalog error (line 877)
func TestGetContainmentTree_ListError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{ID: "cat1"}, nil)
	s.instRepo.On("ListByCatalog", ctx, "cat1").Return(nil, fmt.Errorf("list error"))

	_, err := s.svc.GetContainmentTree(ctx, "my-catalog")
	assert.Error(t, err)
}

// === Coverage: validateAndBuildAttributeValues — all base type branches ===

// helper to set up a single-attribute scenario with a specific base type
func setupSingleAttrBaseType(s *instanceTestSetup, ctx context.Context, attrName string, baseType models.BaseType) {
	s.mockPinResolution(ctx)
	tdvID := "tdv-" + string(baseType)
	tdID := "td-" + string(baseType)
	s.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{
		{ID: "a1", Name: attrName, TypeDefinitionVersionID: tdvID, EntityTypeVersionID: "etv1"},
	}, nil)
	s.tdvRepo.On("GetByID", ctx, tdvID).Return(&models.TypeDefinitionVersion{ID: tdvID, TypeDefinitionID: tdID}, nil)
	s.tdRepo.On("GetByID", ctx, tdID).Return(&models.TypeDefinition{ID: tdID, BaseType: baseType}, nil)
}

func TestCov_ValidateAttrs_BooleanBaseType(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	setupSingleAttrBaseType(s, ctx, "enabled", models.BaseTypeBoolean)

	s.instRepo.On("Create", ctx, mock.Anything).Return(nil)
	s.iavRepo.On("SetValues", ctx, mock.MatchedBy(func(vals []*models.InstanceAttributeValue) bool {
		return len(vals) == 1 && vals[0].ValueString == "true"
	})).Return(nil)
	s.iavRepo.On("GetCurrentValues", ctx, mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft).Return(nil)

	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", map[string]any{
		"enabled": true,
	})
	require.NoError(t, err)
}

func TestCov_ValidateAttrs_IntegerFloat64(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	setupSingleAttrBaseType(s, ctx, "count", models.BaseTypeInteger)

	s.instRepo.On("Create", ctx, mock.Anything).Return(nil)
	s.iavRepo.On("SetValues", ctx, mock.MatchedBy(func(vals []*models.InstanceAttributeValue) bool {
		return len(vals) == 1 && vals[0].ValueNumber != nil && *vals[0].ValueNumber == 7
	})).Return(nil)
	s.iavRepo.On("GetCurrentValues", ctx, mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft).Return(nil)

	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", map[string]any{
		"count": float64(7),
	})
	require.NoError(t, err)
}

func TestCov_ValidateAttrs_IntegerInt(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	setupSingleAttrBaseType(s, ctx, "count", models.BaseTypeInteger)

	s.instRepo.On("Create", ctx, mock.Anything).Return(nil)
	s.iavRepo.On("SetValues", ctx, mock.MatchedBy(func(vals []*models.InstanceAttributeValue) bool {
		return len(vals) == 1 && vals[0].ValueNumber != nil && *vals[0].ValueNumber == 99
	})).Return(nil)
	s.iavRepo.On("GetCurrentValues", ctx, mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft).Return(nil)

	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", map[string]any{
		"count": int(99),
	})
	require.NoError(t, err)
}

func TestCov_ValidateAttrs_IntegerStringValid(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	setupSingleAttrBaseType(s, ctx, "count", models.BaseTypeInteger)

	s.instRepo.On("Create", ctx, mock.Anything).Return(nil)
	s.iavRepo.On("SetValues", ctx, mock.MatchedBy(func(vals []*models.InstanceAttributeValue) bool {
		return len(vals) == 1 && vals[0].ValueNumber != nil && *vals[0].ValueNumber == 55
	})).Return(nil)
	s.iavRepo.On("GetCurrentValues", ctx, mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft).Return(nil)

	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", map[string]any{
		"count": "55",
	})
	require.NoError(t, err)
}

func TestCov_ValidateAttrs_IntegerStringInvalid(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	setupSingleAttrBaseType(s, ctx, "count", models.BaseTypeInteger)

	s.instRepo.On("Create", ctx, mock.Anything).Return(nil)

	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", map[string]any{
		"count": "not-a-number",
	})
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
	assert.Contains(t, err.Error(), "expected integer")
}

func TestCov_ValidateAttrs_IntegerInvalidType(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	setupSingleAttrBaseType(s, ctx, "count", models.BaseTypeInteger)

	s.instRepo.On("Create", ctx, mock.Anything).Return(nil)

	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", map[string]any{
		"count": true, // bool is not valid for integer
	})
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
	assert.Contains(t, err.Error(), "expected integer")
}

func TestCov_ValidateAttrs_NumberStringValid(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	setupSingleAttrBaseType(s, ctx, "rate", models.BaseTypeNumber)

	s.instRepo.On("Create", ctx, mock.Anything).Return(nil)
	s.iavRepo.On("SetValues", ctx, mock.MatchedBy(func(vals []*models.InstanceAttributeValue) bool {
		return len(vals) == 1 && vals[0].ValueNumber != nil && *vals[0].ValueNumber == 3.14
	})).Return(nil)
	s.iavRepo.On("GetCurrentValues", ctx, mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft).Return(nil)

	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", map[string]any{
		"rate": "3.14",
	})
	require.NoError(t, err)
}

func TestCov_ValidateAttrs_ListString(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	setupSingleAttrBaseType(s, ctx, "tags", models.BaseTypeList)

	s.instRepo.On("Create", ctx, mock.Anything).Return(nil)
	s.iavRepo.On("SetValues", ctx, mock.MatchedBy(func(vals []*models.InstanceAttributeValue) bool {
		return len(vals) == 1 && vals[0].ValueJSON == `["a","b"]`
	})).Return(nil)
	s.iavRepo.On("GetCurrentValues", ctx, mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft).Return(nil)

	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", map[string]any{
		"tags": `["a","b"]`,
	})
	require.NoError(t, err)
}

func TestCov_ValidateAttrs_ListSliceMarshal(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	setupSingleAttrBaseType(s, ctx, "tags", models.BaseTypeList)

	s.instRepo.On("Create", ctx, mock.Anything).Return(nil)
	s.iavRepo.On("SetValues", ctx, mock.MatchedBy(func(vals []*models.InstanceAttributeValue) bool {
		return len(vals) == 1 && vals[0].ValueJSON == `["x","y"]`
	})).Return(nil)
	s.iavRepo.On("GetCurrentValues", ctx, mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft).Return(nil)

	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", map[string]any{
		"tags": []string{"x", "y"}, // slice → marshal path
	})
	require.NoError(t, err)
}

func TestCov_ValidateAttrs_JSONMapMarshal(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	setupSingleAttrBaseType(s, ctx, "meta", models.BaseTypeJSON)

	s.instRepo.On("Create", ctx, mock.Anything).Return(nil)
	s.iavRepo.On("SetValues", ctx, mock.MatchedBy(func(vals []*models.InstanceAttributeValue) bool {
		return len(vals) == 1 && vals[0].ValueJSON == `{"k":"v"}`
	})).Return(nil)
	s.iavRepo.On("GetCurrentValues", ctx, mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft).Return(nil)

	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", map[string]any{
		"meta": map[string]any{"k": "v"}, // map → marshal path
	})
	require.NoError(t, err)
}

func TestCov_ValidateAttrs_URLBaseType(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	setupSingleAttrBaseType(s, ctx, "homepage", models.BaseTypeURL)

	s.instRepo.On("Create", ctx, mock.Anything).Return(nil)
	s.iavRepo.On("SetValues", ctx, mock.MatchedBy(func(vals []*models.InstanceAttributeValue) bool {
		return len(vals) == 1 && vals[0].ValueString == "https://example.com"
	})).Return(nil)
	s.iavRepo.On("GetCurrentValues", ctx, mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft).Return(nil)

	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", map[string]any{
		"homepage": "https://example.com",
	})
	require.NoError(t, err)
}

func TestCov_ValidateAttrs_DateBaseType(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	setupSingleAttrBaseType(s, ctx, "birthdate", models.BaseTypeDate)

	s.instRepo.On("Create", ctx, mock.Anything).Return(nil)
	s.iavRepo.On("SetValues", ctx, mock.MatchedBy(func(vals []*models.InstanceAttributeValue) bool {
		return len(vals) == 1 && vals[0].ValueString == "2026-04-12"
	})).Return(nil)
	s.iavRepo.On("GetCurrentValues", ctx, mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft).Return(nil)

	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", map[string]any{
		"birthdate": "2026-04-12",
	})
	require.NoError(t, err)
}

func TestCov_ValidateAttrs_DefaultBaseType(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	// Set up with an unknown base type by mocking a custom type
	s.mockPinResolution(ctx)
	s.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{
		{ID: "a1", Name: "custom", TypeDefinitionVersionID: "tdv-weird", EntityTypeVersionID: "etv1"},
	}, nil)
	s.tdvRepo.On("GetByID", ctx, "tdv-weird").Return(&models.TypeDefinitionVersion{ID: "tdv-weird", TypeDefinitionID: "td-weird"}, nil)
	s.tdRepo.On("GetByID", ctx, "td-weird").Return(&models.TypeDefinition{ID: "td-weird", BaseType: "xyzzy"}, nil)

	s.instRepo.On("Create", ctx, mock.Anything).Return(nil)
	s.iavRepo.On("SetValues", ctx, mock.MatchedBy(func(vals []*models.InstanceAttributeValue) bool {
		return len(vals) == 1 && vals[0].ValueString == "fallback-value"
	})).Return(nil)
	s.iavRepo.On("GetCurrentValues", ctx, mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft).Return(nil)

	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", map[string]any{
		"custom": "fallback-value",
	})
	require.NoError(t, err)
}

func TestCov_ValidateAttrs_JSONMarshalError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	setupSingleAttrBaseType(s, ctx, "meta", models.BaseTypeJSON)

	s.instRepo.On("Create", ctx, mock.Anything).Return(nil)

	// Channels cannot be marshaled to JSON
	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", map[string]any{
		"meta": make(chan int), // triggers json.Marshal error
	})
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
	assert.Contains(t, err.Error(), "failed to marshal JSON value")
}

func TestCov_ResolveAttrValues_BaseTypesError(t *testing.T) {
	// resolveAttributeValues → resolveBaseTypes returns error (line 172)
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{
		{ID: "a1", Name: "broken", TypeDefinitionVersionID: "tdv-missing", EntityTypeVersionID: "etv1"},
	}, nil)
	s.tdvRepo.On("GetByID", ctx, "tdv-missing").Return(nil, domainerrors.NewValidation("tdv not found"))

	s.instRepo.On("GetByID", ctx, "i1").Return(&models.EntityInstance{
		ID: "i1", CatalogID: "cat1", EntityTypeID: "et1", Version: 1,
	}, nil)

	_, err := s.svc.GetInstance(ctx, "my-catalog", "model", "i1")
	assert.Error(t, err)
}

// resolveParentChain: GetByID error during walk (line 940) and ET name fallback (line 949)
func TestGetInstance_ParentChainETNameFallback(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.mockAttributes(ctx)

	s.instRepo.On("GetByID", ctx, "i1").Return(&models.EntityInstance{
		ID: "i1", CatalogID: "cat1", EntityTypeID: "et1", Version: 1,
		ParentInstanceID: "parent1",
	}, nil)
	s.iavRepo.On("GetCurrentValues", ctx, "i1").Return([]*models.InstanceAttributeValue{}, nil)
	s.instRepo.On("GetByID", ctx, "parent1").Return(&models.EntityInstance{
		ID: "parent1", CatalogID: "cat1", EntityTypeID: "et-unknown", Version: 1,
	}, nil)
	// ET lookup fails — should fallback to EntityTypeID
	s.etRepo.On("GetByID", ctx, "et-unknown").Return(nil, fmt.Errorf("et not found"))

	detail, err := s.svc.GetInstance(ctx, "my-catalog", "model", "i1")
	require.NoError(t, err)
	assert.Len(t, detail.ParentChain, 1)
	assert.Equal(t, "et-unknown", detail.ParentChain[0].EntityTypeName) // fallback to ID
}

// Cover resolveBaseTypes error in ListInstances
func TestListInstances_ResolveBaseTypesError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "cat").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	s.etRepo.On("GetByName", ctx, "Server").Return(&models.EntityType{ID: "et1"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "p1", EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1"}, nil)
	s.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{
		{ID: "a1", Name: "x", TypeDefinitionVersionID: "tdv-missing"},
	}, nil)
	s.tdvRepo.On("GetByID", ctx, "tdv-missing").Return(nil, errors.New("tdv not found"))

	_, _, err := s.svc.ListInstances(ctx, "cat", "Server", models.ListParams{Limit: 10})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tdv not found")
}

// Cover resolveBaseTypes error in ListContainedInstances
func TestListContainedInstances_ResolveBaseTypesError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "cat").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	s.etRepo.On("GetByName", ctx, "Parent").Return(&models.EntityType{ID: "et1"}, nil)
	s.etRepo.On("GetByName", ctx, "Child").Return(&models.EntityType{ID: "et2"}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "p1", EntityTypeVersionID: "etv1"},
		{ID: "p2", EntityTypeVersionID: "etv2"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1"}, nil)
	s.etvRepo.On("GetByID", ctx, "etv2").Return(&models.EntityTypeVersion{ID: "etv2", EntityTypeID: "et2"}, nil)
	s.instRepo.On("GetByID", ctx, "parent1").Return(&models.EntityInstance{ID: "parent1", CatalogID: "c1"}, nil)
	s.instRepo.On("ListByParent", ctx, "parent1", mock.Anything).Return([]*models.EntityInstance{}, 0, nil)
	s.attrRepo.On("ListByVersion", ctx, "etv2").Return([]*models.Attribute{
		{ID: "a1", Name: "x", TypeDefinitionVersionID: "tdv-missing"},
	}, nil)
	s.tdvRepo.On("GetByID", ctx, "tdv-missing").Return(nil, errors.New("tdv not found"))

	_, _, err := s.svc.ListContainedInstances(ctx, "cat", "Parent", "parent1", "Child", models.ListParams{Limit: 10})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tdv not found")
}
