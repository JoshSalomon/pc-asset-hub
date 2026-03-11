package operational_test

import (
	"context"
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
	enumValRepo *mocks.MockEnumValueRepo
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
		enumValRepo: new(mocks.MockEnumValueRepo),
	}
	s.svc = operational.NewInstanceService(
		s.instRepo, s.iavRepo, s.catalogRepo, s.cvRepo,
		s.pinRepo, s.attrRepo, s.etvRepo, s.etRepo, s.enumValRepo,
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
		{ID: "a1", Name: "hostname", Type: models.AttributeTypeString, EntityTypeVersionID: "etv1"},
		{ID: "a2", Name: "port", Type: models.AttributeTypeNumber, EntityTypeVersionID: "etv1"},
		{ID: "a3", Name: "status", Type: models.AttributeTypeEnum, EnumID: "enum1", EntityTypeVersionID: "etv1"},
	}, nil)
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

	detail, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "my-instance", "desc", map[string]interface{}{
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

	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", map[string]interface{}{
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

	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", map[string]interface{}{
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

	s.enumValRepo.On("ListByEnum", ctx, "enum1").Return([]*models.EnumValue{
		{ID: "ev1", Value: "active"},
		{ID: "ev2", Value: "inactive"},
	}, nil)
	s.instRepo.On("Create", ctx, mock.Anything).Return(nil)
	s.iavRepo.On("SetValues", ctx, mock.MatchedBy(func(vals []*models.InstanceAttributeValue) bool {
		return len(vals) == 1 && vals[0].ValueEnum == "active"
	})).Return(nil)
	s.iavRepo.On("GetCurrentValues", ctx, mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	s.catalogRepo.On("UpdateValidationStatus", ctx, "cat1", models.ValidationStatusDraft).Return(nil)

	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", map[string]interface{}{
		"status": "active",
	})
	require.NoError(t, err)
}

// T-11.17: Create instance with invalid enum value
func TestT11_17_InvalidEnumValue(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.mockAttributes(ctx)

	s.enumValRepo.On("ListByEnum", ctx, "enum1").Return([]*models.EnumValue{
		{ID: "ev1", Value: "active"},
	}, nil)
	s.instRepo.On("Create", ctx, mock.Anything).Return(nil)

	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", map[string]interface{}{
		"status": "bogus",
	})
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

// T-11.18: Create instance with non-parseable number
func TestT11_18_NonParseableNumber(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.mockAttributes(ctx)

	s.instRepo.On("Create", ctx, mock.Anything).Return(nil)

	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", map[string]interface{}{
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
		{ID: "a1", Name: "hostname", Type: models.AttributeTypeString, Required: true},
	}, nil)

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

	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", map[string]interface{}{
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

	detail, err := s.svc.UpdateInstance(ctx, "my-catalog", "model", "inst1", 1, nil, nil, map[string]interface{}{
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
		ID: "inst1", Version: 3,
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
		ID: "inst1", Version: 1,
	}, nil)

	_, err := s.svc.UpdateInstance(ctx, "my-catalog", "model", "inst1", 1, nil, nil, map[string]interface{}{
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

	s.instRepo.On("ListByParent", ctx, "parent1", mock.Anything).Return([]*models.EntityInstance{
		{ID: "child1"},
	}, 1, nil)
	s.instRepo.On("ListByParent", ctx, "child1", mock.Anything).Return([]*models.EntityInstance{}, 0, nil)
	s.instRepo.On("SoftDelete", ctx, "child1").Return(nil)
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

	s.instRepo.On("ListByParent", ctx, "inst1", mock.Anything).Return([]*models.EntityInstance{}, 0, nil)
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

	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", map[string]interface{}{"x": "y"})
	assert.Error(t, err)
}

// Error propagation: enumValRepo.ListByEnum error
func TestCov_ValidateAttrs_EnumRepoError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
	s.mockAttributes(ctx)
	s.enumValRepo.On("ListByEnum", ctx, "enum1").Return(nil, domainerrors.NewValidation("db error"))
	s.instRepo.On("Create", ctx, mock.Anything).Return(nil)

	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", map[string]interface{}{"status": "active"})
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

	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", map[string]interface{}{"hostname": "h"})
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

	_, err := s.svc.UpdateInstance(ctx, "my-catalog", "model", "i1", 1, nil, nil, map[string]interface{}{"x": "y"})
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

	_, err := s.svc.UpdateInstance(ctx, "my-catalog", "model", "i1", 1, nil, nil, map[string]interface{}{"hostname": "h"})
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

	_, err := s.svc.UpdateInstance(ctx, "my-catalog", "model", "i1", 1, nil, nil, map[string]interface{}{"hostname": "h"})
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
	s.instRepo.On("ListByParent", ctx, "i1", mock.Anything).Return(nil, 0, domainerrors.NewValidation("db error"))

	err := s.svc.DeleteInstance(ctx, "my-catalog", "model", "i1")
	assert.Error(t, err)
}

// Error propagation: cascadeDelete - recursive error
func TestCov_CascadeDelete_RecursiveError(t *testing.T) {
	s := setupInstanceService()
	ctx := context.Background()
	s.mockPinResolution(ctx)
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
		{AttributeID: "a3", ValueEnum: "active", InstanceVersion: 1},
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

	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", map[string]interface{}{
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

	_, err := s.svc.CreateInstance(ctx, "my-catalog", "model", "inst", "", map[string]interface{}{
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
		{AttributeID: "a3", ValueEnum: "active", InstanceVersion: 1},
	}, nil)

	details, total, err := s.svc.ListInstances(ctx, "my-catalog", "model", models.ListParams{Limit: 20})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "host-a", details[0].Attributes[0].Value)
	assert.Equal(t, &num, details[0].Attributes[1].Value)
	assert.Equal(t, "active", details[0].Attributes[2].Value)
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

	_, err := s.svc.UpdateInstance(ctx, "my-catalog", "model", "i1", 1, nil, nil, map[string]interface{}{
		"hostname": "new-host",
	})
	require.NoError(t, err)
	s.iavRepo.AssertCalled(t, "SetValues", ctx, mock.MatchedBy(func(vals []*models.InstanceAttributeValue) bool {
		return len(vals) == 2
	}))
}
