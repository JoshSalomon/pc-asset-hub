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

type validationMocks struct {
	catRepo   *mocks.MockCatalogRepo
	instRepo  *mocks.MockEntityInstanceRepo
	iavRepo   *mocks.MockInstanceAttributeValueRepo
	pinRepo   *mocks.MockCatalogVersionPinRepo
	etvRepo   *mocks.MockEntityTypeVersionRepo
	attrRepo  *mocks.MockAttributeRepo
	assocRepo *mocks.MockAssociationRepo
	tdvRepo   *mocks.MockTypeDefinitionVersionRepo
	tdRepo    *mocks.MockTypeDefinitionRepo
	linkRepo  *mocks.MockAssociationLinkRepo
	etRepo    *mocks.MockEntityTypeRepo
}

func setupValidationService() (*operational.CatalogValidationService, *validationMocks) {
	m := &validationMocks{
		catRepo:   new(mocks.MockCatalogRepo),
		instRepo:  new(mocks.MockEntityInstanceRepo),
		iavRepo:   new(mocks.MockInstanceAttributeValueRepo),
		pinRepo:   new(mocks.MockCatalogVersionPinRepo),
		etvRepo:   new(mocks.MockEntityTypeVersionRepo),
		attrRepo:  new(mocks.MockAttributeRepo),
		assocRepo: new(mocks.MockAssociationRepo),
		tdvRepo:   new(mocks.MockTypeDefinitionVersionRepo),
		tdRepo:    new(mocks.MockTypeDefinitionRepo),
		linkRepo:  new(mocks.MockAssociationLinkRepo),
		etRepo:    new(mocks.MockEntityTypeRepo),
	}
	svc := operational.NewCatalogValidationService(
		m.catRepo, m.instRepo, m.iavRepo, m.pinRepo, m.etvRepo,
		m.attrRepo, m.assocRepo, m.tdvRepo, m.tdRepo, m.linkRepo, m.etRepo,
	)
	return svc, m
}

// setupSingleEntityType sets up common mocks for a catalog with one entity type.
func (m *validationMocks) setupSingleEntityType(ctx context.Context) {
	m.catRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "c1", Name: "my-catalog", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusDraft,
	}, nil)
	m.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
	}, nil)
	m.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{
		ID: "etv1", EntityTypeID: "et1", Version: 1,
	}, nil)
	m.etRepo.On("GetByID", ctx, "et1").Return(&models.EntityType{ID: "et1", Name: "Server"}, nil)
	m.etRepo.On("GetByID", ctx, "et2").Return(&models.EntityType{ID: "et2", Name: "Model"}, nil)
	m.setupCommonTypeDefs(ctx)
}

// setupCommonTypeDefs sets up type definition mocks for string, number, and enum base types.
func (m *validationMocks) setupCommonTypeDefs(ctx context.Context) {
	m.tdvRepo.On("GetByID", ctx, "tdv-string").Return(&models.TypeDefinitionVersion{
		ID: "tdv-string", TypeDefinitionID: "td-string",
	}, nil).Maybe()
	m.tdRepo.On("GetByID", ctx, "td-string").Return(&models.TypeDefinition{
		ID: "td-string", Name: "String", BaseType: models.BaseTypeString,
	}, nil).Maybe()
	m.tdvRepo.On("GetByID", ctx, "tdv-number").Return(&models.TypeDefinitionVersion{
		ID: "tdv-number", TypeDefinitionID: "td-number",
	}, nil).Maybe()
	m.tdRepo.On("GetByID", ctx, "td-number").Return(&models.TypeDefinition{
		ID: "td-number", Name: "Number", BaseType: models.BaseTypeNumber,
	}, nil).Maybe()
}

// T-15.01: Instance missing value for required attribute produces error
func TestT15_01_RequiredAttrMissing(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupSingleEntityType(ctx)

	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "server-1"},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{
		{ID: "attr1", Name: "hostname", TypeDefinitionVersionID: "tdv-string", Required: true},
	}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusInvalid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusInvalid, result.Status)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "Server", result.Errors[0].EntityType)
	assert.Equal(t, "server-1", result.Errors[0].InstanceName)
	assert.Equal(t, "hostname", result.Errors[0].Field)
	assert.Contains(t, result.Errors[0].Violation, "required")
}

// T-15.02: Instance with value for required attribute passes
func TestT15_02_RequiredAttrPresent(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupSingleEntityType(ctx)

	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "server-1"},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{
		{ID: "attr1", Name: "hostname", TypeDefinitionVersionID: "tdv-string", Required: true},
	}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{
		{ID: "v1", InstanceID: "inst1", AttributeID: "attr1", ValueString: "web-01"},
	}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusValid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusValid, result.Status)
	assert.Empty(t, result.Errors)
}

// T-15.03: Instance missing value for optional attribute passes
func TestT15_03_OptionalAttrMissing(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupSingleEntityType(ctx)

	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "server-1"},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{
		{ID: "attr1", Name: "description", TypeDefinitionVersionID: "tdv-string", Required: false},
	}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusValid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusValid, result.Status)
	assert.Empty(t, result.Errors)
}

// T-15.04: Multiple instances missing different required attrs produce separate errors
func TestT15_04_MultipleErrors(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupSingleEntityType(ctx)

	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "server-1"},
		{ID: "inst2", EntityTypeID: "et1", CatalogID: "c1", Name: "server-2"},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{
		{ID: "attr1", Name: "hostname", TypeDefinitionVersionID: "tdv-string", Required: true},
		{ID: "attr2", Name: "ip", TypeDefinitionVersionID: "tdv-string", Required: true},
	}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{
		{ID: "v1", InstanceID: "inst1", AttributeID: "attr1", ValueString: "web-01"},
		// missing attr2 (ip)
	}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst2", mock.Anything).Return([]*models.InstanceAttributeValue{
		// missing both
	}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusInvalid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusInvalid, result.Status)
	// inst1 missing ip, inst2 missing hostname and ip = 3 errors
	assert.Len(t, result.Errors, 3)
}

// T-15.05: String attribute with any value passes
func TestT15_05_StringAttrValid(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupSingleEntityType(ctx)

	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "server-1"},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{
		{ID: "attr1", Name: "hostname", TypeDefinitionVersionID: "tdv-string", Required: true},
	}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{
		{ID: "v1", InstanceID: "inst1", AttributeID: "attr1", ValueString: "anything goes"},
	}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusValid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusValid, result.Status)
	assert.Empty(t, result.Errors)
}

// T-15.06: Number attribute with valid float value passes
func TestT15_06_NumberAttrValid(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupSingleEntityType(ctx)

	num := 42.5
	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "server-1"},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{
		{ID: "attr1", Name: "cpu_count", TypeDefinitionVersionID: "tdv-number", Required: true},
	}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{
		{ID: "v1", InstanceID: "inst1", AttributeID: "attr1", ValueNumber: &num},
	}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusValid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusValid, result.Status)
	assert.Empty(t, result.Errors)
}

// T-15.08: Enum attribute with value in allowed list passes
func TestT15_08_EnumAttrValid(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupSingleEntityType(ctx)

	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "server-1"},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{
		{ID: "attr1", Name: "status", TypeDefinitionVersionID: "tdv-enum1", Required: true},
	}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{
		{ID: "v1", InstanceID: "inst1", AttributeID: "attr1", ValueString: "active"},
	}, nil)
	// Type resolution mocks for enum attribute
	m.tdvRepo.On("GetByID", ctx, "tdv-enum1").Return(&models.TypeDefinitionVersion{ID: "tdv-enum1", TypeDefinitionID: "td-enum1", Constraints: map[string]any{"values": []any{"active", "inactive"}}}, nil)
	m.tdRepo.On("GetByID", ctx, "td-enum1").Return(&models.TypeDefinition{ID: "td-enum1", BaseType: models.BaseTypeEnum}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusValid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusValid, result.Status)
	assert.Empty(t, result.Errors)
}

// T-15.09: Enum attribute with value not in allowed list produces error
func TestT15_09_EnumAttrInvalid(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupSingleEntityType(ctx)

	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "server-1"},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{
		{ID: "attr1", Name: "status", TypeDefinitionVersionID: "tdv-enum1", Required: true},
	}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{
		{ID: "v1", InstanceID: "inst1", AttributeID: "attr1", ValueString: "bogus"},
	}, nil)
	// Type resolution mocks for enum attribute
	m.tdvRepo.On("GetByID", ctx, "tdv-enum1").Return(&models.TypeDefinitionVersion{ID: "tdv-enum1", TypeDefinitionID: "td-enum1", Constraints: map[string]any{"values": []any{"active", "inactive"}}}, nil)
	m.tdRepo.On("GetByID", ctx, "td-enum1").Return(&models.TypeDefinition{ID: "td-enum1", BaseType: models.BaseTypeEnum}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusInvalid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusInvalid, result.Status)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "status", result.Errors[0].Field)
	assert.Contains(t, result.Errors[0].Violation, "invalid enum value")
}

// T-15.10: Association with target_cardinality "1" — source instance has one link → passes
func TestT15_10_MandatoryAssocSatisfied(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupSingleEntityType(ctx)

	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "server-1"},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{
		{ID: "assoc1", Name: "runs-on", Type: models.AssociationTypeDirectional,
			TargetEntityTypeID: "et2", TargetCardinality: "1"},
	}, nil)
	m.linkRepo.On("GetForwardRefs", ctx, "inst1").Return([]*models.AssociationLink{
		{ID: "link1", AssociationID: "assoc1", SourceInstanceID: "inst1", TargetInstanceID: "inst2"},
	}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusValid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusValid, result.Status)
	assert.Empty(t, result.Errors)
}

// T-15.11: Association with target_cardinality "1" — source instance has no link → error
func TestT15_11_MandatoryAssocMissing(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupSingleEntityType(ctx)

	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "server-1"},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{
		{ID: "assoc1", Name: "runs-on", Type: models.AssociationTypeDirectional,
			TargetEntityTypeID: "et2", TargetCardinality: "1"},
	}, nil)
	m.linkRepo.On("GetForwardRefs", ctx, "inst1").Return([]*models.AssociationLink{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusInvalid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusInvalid, result.Status)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "runs-on", result.Errors[0].Field)
	assert.Contains(t, result.Errors[0].Violation, "mandatory association")
}

// T-15.12: Association with target_cardinality "1..n" — source instance has one link → passes
func TestT15_12_MandatoryAssoc1NWithLink(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupSingleEntityType(ctx)

	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "server-1"},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{
		{ID: "assoc1", Name: "has-tools", Type: models.AssociationTypeDirectional,
			TargetEntityTypeID: "et2", TargetCardinality: "1..n"},
	}, nil)
	m.linkRepo.On("GetForwardRefs", ctx, "inst1").Return([]*models.AssociationLink{
		{ID: "link1", AssociationID: "assoc1", SourceInstanceID: "inst1", TargetInstanceID: "inst2"},
	}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusValid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusValid, result.Status)
	assert.Empty(t, result.Errors)
}

// T-15.13: Association with target_cardinality "1..n" — source instance has no link → error
func TestT15_13_MandatoryAssoc1NMissing(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupSingleEntityType(ctx)

	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "server-1"},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{
		{ID: "assoc1", Name: "has-tools", Type: models.AssociationTypeDirectional,
			TargetEntityTypeID: "et2", TargetCardinality: "1..n"},
	}, nil)
	m.linkRepo.On("GetForwardRefs", ctx, "inst1").Return([]*models.AssociationLink{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusInvalid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusInvalid, result.Status)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0].Violation, "mandatory association")
}

// T-15.14: Association with target_cardinality "0..n" — no link → passes
func TestT15_14_OptionalAssocNoLink(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupSingleEntityType(ctx)

	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "server-1"},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{
		{ID: "assoc1", Name: "relates-to", Type: models.AssociationTypeDirectional,
			TargetEntityTypeID: "et2", TargetCardinality: "0..n"},
	}, nil)
	m.linkRepo.On("GetForwardRefs", ctx, "inst1").Return([]*models.AssociationLink{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusValid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusValid, result.Status)
	assert.Empty(t, result.Errors)
}

// T-15.15: Association with target_cardinality "0..1" — no link → passes
func TestT15_15_OptionalAssoc01NoLink(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupSingleEntityType(ctx)

	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "server-1"},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{
		{ID: "assoc1", Name: "optional-ref", Type: models.AssociationTypeDirectional,
			TargetEntityTypeID: "et2", TargetCardinality: "0..1"},
	}, nil)
	m.linkRepo.On("GetForwardRefs", ctx, "inst1").Return([]*models.AssociationLink{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusValid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusValid, result.Status)
	assert.Empty(t, result.Errors)
}

// setupTwoEntityTypes sets up common mocks for a catalog with two entity types (et1=Server, et2=Tool).
func (m *validationMocks) setupTwoEntityTypes(ctx context.Context) {
	m.catRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "c1", Name: "my-catalog", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusDraft,
	}, nil)
	m.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
		{ID: "pin2", CatalogVersionID: "cv1", EntityTypeVersionID: "etv2"},
	}, nil)
	m.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{
		ID: "etv1", EntityTypeID: "et1", Version: 1,
	}, nil)
	m.etvRepo.On("GetByID", ctx, "etv2").Return(&models.EntityTypeVersion{
		ID: "etv2", EntityTypeID: "et2", Version: 1,
	}, nil)
	m.etRepo.On("GetByID", ctx, "et1").Return(&models.EntityType{ID: "et1", Name: "Server"}, nil)
	m.etRepo.On("GetByID", ctx, "et2").Return(&models.EntityType{ID: "et2", Name: "Tool"}, nil)
	m.setupCommonTypeDefs(ctx)
}

// T-15.16: Containment associations are excluded from mandatory assoc checks
func TestT15_16_ContainmentExcludedFromAssocCheck(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupSingleEntityType(ctx)

	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "server-1"},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{
		{ID: "assoc1", Name: "contains-tool", Type: models.AssociationTypeContainment,
			TargetEntityTypeID: "et2", TargetCardinality: "1..n"},
	}, nil)
	// No GetForwardRefs call should be made for containment assocs
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusValid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusValid, result.Status)
	assert.Empty(t, result.Errors)
}

// T-15.17: Contained instance with valid parent passes
func TestT15_17_ContainmentValid(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupTwoEntityTypes(ctx)

	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "server-1"},
		{ID: "inst2", EntityTypeID: "et2", CatalogID: "c1", Name: "tool-1", ParentInstanceID: "inst1"},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv2").Return([]*models.Attribute{}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst2", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{
		{ID: "assoc1", Name: "contains-tool", Type: models.AssociationTypeContainment, TargetEntityTypeID: "et2"},
	}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv2").Return([]*models.Association{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusValid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusValid, result.Status)
	assert.Empty(t, result.Errors)
}

// T-15.18: Instance with ParentInstanceID pointing to non-existent instance → error
func TestT15_18_OrphanedContainedInstance(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupSingleEntityType(ctx)

	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "server-1", ParentInstanceID: "deleted-parent"},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusInvalid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusInvalid, result.Status)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0].Violation, "orphaned")
}

// T-15.19: Instance with parent whose entity type has no containment assoc to child type → error
func TestT15_19_InvalidContainmentRelationship(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupTwoEntityTypes(ctx)

	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "server-1"},
		{ID: "inst2", EntityTypeID: "et2", CatalogID: "c1", Name: "tool-1", ParentInstanceID: "inst1"},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv2").Return([]*models.Attribute{}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst2", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	// No containment association from et1 to et2
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv2").Return([]*models.Association{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusInvalid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusInvalid, result.Status)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0].Violation, "invalid containment")
}

// T-15.20: Top-level instance (no ParentInstanceID) passes containment check
func TestT15_20_TopLevelInstancePasses(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupSingleEntityType(ctx)

	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "server-1"},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusValid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusValid, result.Status)
	assert.Empty(t, result.Errors)
}

// T-15.21: All checks pass → catalog status set to valid
func TestT15_21_AllPassValid(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupSingleEntityType(ctx)

	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "server-1"},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{
		{ID: "attr1", Name: "hostname", TypeDefinitionVersionID: "tdv-string", Required: true},
	}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{
		{ID: "v1", InstanceID: "inst1", AttributeID: "attr1", ValueString: "web-01"},
	}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusValid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusValid, result.Status)
	m.catRepo.AssertCalled(t, "UpdateValidationStatus", ctx, "c1", models.ValidationStatusValid)
}

// T-15.22: Any check fails → catalog status set to invalid
func TestT15_22_AnyFailInvalid(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupSingleEntityType(ctx)

	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "server-1"},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{
		{ID: "attr1", Name: "hostname", TypeDefinitionVersionID: "tdv-string", Required: true},
	}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusInvalid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusInvalid, result.Status)
	m.catRepo.AssertCalled(t, "UpdateValidationStatus", ctx, "c1", models.ValidationStatusInvalid)
}

// T-15.23: Empty catalog passes validation, status valid
func TestT15_23_EmptyCatalogValid(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()

	m.catRepo.On("GetByName", ctx, "empty-catalog").Return(&models.Catalog{
		ID: "c1", Name: "empty-catalog", CatalogVersionID: "cv1",
	}, nil)
	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusValid).Return(nil)

	result, err := svc.Validate(ctx, "empty-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusValid, result.Status)
	assert.Empty(t, result.Errors)
}

// T-15.24: Nonexistent catalog returns NotFound error
func TestT15_24_NonexistentCatalog(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()

	m.catRepo.On("GetByName", ctx, "nonexistent").Return(nil, domainerrors.NewNotFound("Catalog", "nonexistent"))

	_, err := svc.Validate(ctx, "nonexistent")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
}

// T-15.25: Error list contains all four fields
func TestT15_25_ErrorStructure(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupSingleEntityType(ctx)

	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "my-instance"},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{
		{ID: "attr1", Name: "my-field", TypeDefinitionVersionID: "tdv-string", Required: true},
	}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusInvalid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	require.Len(t, result.Errors, 1)
	e := result.Errors[0]
	assert.Equal(t, "Server", e.EntityType)
	assert.Equal(t, "my-instance", e.InstanceName)
	assert.Equal(t, "my-field", e.Field)
	assert.NotEmpty(t, e.Violation)
}

// T-15.07 (was missing): Number attribute with non-parseable string produces error
// Note: in practice, number values are stored as *float64 in the DB,
// so a non-parseable string can't reach validation. But we test the isEmptyValue path:
// a number attribute with nil ValueNumber is treated as empty/missing.
// This test verifies that a required number attr with nil value is caught.
func TestT15_07_NumberAttrRequiredMissing(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupSingleEntityType(ctx)

	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "server-1"},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{
		{ID: "attr1", Name: "cpu_count", TypeDefinitionVersionID: "tdv-number", Required: true},
	}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{
		{ID: "v1", InstanceID: "inst1", AttributeID: "attr1", ValueNumber: nil},
	}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusInvalid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusInvalid, result.Status)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "cpu_count", result.Errors[0].Field)
	assert.Contains(t, result.Errors[0].Violation, "required")
}

// L1 fix: Instance of unpinned entity type produces validation error
func TestValidation_UnpinnedEntityTypeProducesError(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()

	m.catRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "c1", Name: "my-catalog", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusDraft,
	}, nil)
	// Instance belongs to et1, but CV only pins et2
	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "orphan-server"},
	}, nil)
	m.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv2"},
	}, nil)
	m.etvRepo.On("GetByID", ctx, "etv2").Return(&models.EntityTypeVersion{
		ID: "etv2", EntityTypeID: "et2", Version: 1,
	}, nil)
	m.etRepo.On("GetByID", ctx, "et1").Return(&models.EntityType{ID: "et1", Name: "Server"}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv2").Return([]*models.Association{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusInvalid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusInvalid, result.Status)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "Server", result.Errors[0].EntityType)
	assert.Contains(t, result.Errors[0].Violation, "not pinned")
}

// L9: Bidirectional association with mandatory cardinality
func TestValidation_BidirectionalMandatoryAssoc(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupSingleEntityType(ctx)

	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "server-1"},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{
		{ID: "assoc1", Name: "peers-with", Type: models.AssociationTypeBidirectional,
			TargetEntityTypeID: "et2", TargetCardinality: "1"},
	}, nil)
	m.linkRepo.On("GetForwardRefs", ctx, "inst1").Return([]*models.AssociationLink{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusInvalid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusInvalid, result.Status)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0].Violation, "mandatory association")
}

// === Error propagation and edge case tests for full coverage ===

func TestValidation_ListByCatalogError(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.catRepo.On("GetByName", ctx, "c").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	m.instRepo.On("ListByCatalog", ctx, "c1").Return(nil, domainerrors.NewValidation("db"))
	_, err := svc.Validate(ctx, "c")
	assert.Error(t, err)
}

func TestValidation_EmptyCatalogUpdateStatusError(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.catRepo.On("GetByName", ctx, "c").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusValid).Return(domainerrors.NewValidation("db"))
	_, err := svc.Validate(ctx, "c")
	assert.Error(t, err)
}

func TestValidation_ListPinsError(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.catRepo.On("GetByName", ctx, "c").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{{ID: "i1", EntityTypeID: "et1"}}, nil)
	m.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return(nil, domainerrors.NewValidation("db"))
	_, err := svc.Validate(ctx, "c")
	assert.Error(t, err)
}

func TestValidation_ETVResolveError(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.catRepo.On("GetByName", ctx, "c").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{{ID: "i1", EntityTypeID: "et1"}}, nil)
	m.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{{EntityTypeVersionID: "etv1"}}, nil)
	m.etvRepo.On("GetByID", ctx, "etv1").Return(nil, domainerrors.NewNotFound("ETV", "etv1"))
	_, err := svc.Validate(ctx, "c")
	assert.Error(t, err)
}

func TestValidation_ETNameFallback(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.catRepo.On("GetByName", ctx, "c").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	m.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{{EntityTypeVersionID: "etv1"}}, nil)
	m.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1"}, nil)
	m.etRepo.On("GetByID", ctx, "et1").Return(nil, domainerrors.NewNotFound("ET", "et1"))
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{}, nil)
	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{{ID: "i1", EntityTypeID: "et1", Name: "s1"}}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{{ID: "a1", Name: "h", TypeDefinitionVersionID: "tdv-string", Required: true}}, nil)
	m.tdvRepo.On("GetByID", ctx, "tdv-string").Return(&models.TypeDefinitionVersion{ID: "tdv-string", TypeDefinitionID: "td-string"}, nil)
	m.tdRepo.On("GetByID", ctx, "td-string").Return(&models.TypeDefinition{ID: "td-string", BaseType: models.BaseTypeString}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "i1", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusInvalid).Return(nil)
	result, err := svc.Validate(ctx, "c")
	require.NoError(t, err)
	assert.Equal(t, "et1", result.Errors[0].EntityType) // fallback to ID
}

func TestValidation_AttrListError(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupSingleEntityType(ctx)
	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{{ID: "i1", EntityTypeID: "et1"}}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return(nil, domainerrors.NewValidation("db"))
	_, err := svc.Validate(ctx, "my-catalog")
	assert.Error(t, err)
}

func TestValidation_EnumValuesListError(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupSingleEntityType(ctx)
	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{{ID: "i1", EntityTypeID: "et1"}}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{{ID: "a1", Name: "s", TypeDefinitionVersionID: "tdv-e1"}}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{}, nil)
	m.tdvRepo.On("GetByID", ctx, "tdv-e1").Return(nil, domainerrors.NewValidation("db"))
	_, err := svc.Validate(ctx, "my-catalog")
	assert.Error(t, err)
}

func TestValidation_GetValuesForVersionError(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupSingleEntityType(ctx)
	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{{ID: "i1", EntityTypeID: "et1"}}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "i1", mock.Anything).Return(nil, domainerrors.NewValidation("db"))
	_, err := svc.Validate(ctx, "my-catalog")
	assert.Error(t, err)
}

func TestValidation_GetForwardRefsError(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupSingleEntityType(ctx)
	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{{ID: "i1", EntityTypeID: "et1"}}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "i1", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{
		{ID: "a1", Name: "r", Type: models.AssociationTypeDirectional, TargetEntityTypeID: "et2", TargetCardinality: "1"},
	}, nil)
	m.linkRepo.On("GetForwardRefs", ctx, "i1").Return([]*models.AssociationLink{}, domainerrors.NewValidation("db"))
	_, err := svc.Validate(ctx, "my-catalog")
	assert.Error(t, err)
}

func TestValidation_ContainmentParentNotPinned(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.catRepo.On("GetByName", ctx, "c").Return(&models.Catalog{ID: "c1", CatalogVersionID: "cv1"}, nil)
	m.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{{EntityTypeVersionID: "etv2"}}, nil)
	m.etvRepo.On("GetByID", ctx, "etv2").Return(&models.EntityTypeVersion{ID: "etv2", EntityTypeID: "et2"}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv2").Return([]*models.Association{}, nil)
	m.etRepo.On("GetByID", ctx, "et1").Return(&models.EntityType{ID: "et1", Name: "Server"}, nil)
	m.etRepo.On("GetByID", ctx, "et2").Return(&models.EntityType{ID: "et2", Name: "Tool"}, nil)
	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "i1", EntityTypeID: "et1", Name: "server-1"},
		{ID: "i2", EntityTypeID: "et2", Name: "tool-1", ParentInstanceID: "i1"},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv2").Return([]*models.Attribute{}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "i2", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusInvalid).Return(nil)
	result, err := svc.Validate(ctx, "c")
	require.NoError(t, err)
	var found bool
	for _, e := range result.Errors {
		if e.InstanceName == "tool-1" && e.Field == "parent" && e.Violation == "invalid containment: parent entity type not pinned in CV" {
			found = true
		}
	}
	assert.True(t, found, "should have parent-not-pinned error")
}

func TestValidation_FinalUpdateStatusError(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupSingleEntityType(ctx)
	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{{ID: "i1", EntityTypeID: "et1", Name: "server-1"}}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "i1", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusValid).Return(domainerrors.NewValidation("db"))
	_, err := svc.Validate(ctx, "my-catalog")
	assert.Error(t, err)
}

func TestValidation_AssocPreloadError(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupSingleEntityType(ctx)
	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{{ID: "i1", EntityTypeID: "et1"}}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{}, domainerrors.NewValidation("db"))
	_, err := svc.Validate(ctx, "my-catalog")
	assert.Error(t, err)
}

// === Exported helper function tests ===

func TestCardinalityMinGE1_EmptyString(t *testing.T) {
	assert.False(t, operational.CardinalityMinGE1(""))
}

func TestCardinalityMinGE1_NonNumeric(t *testing.T) {
	assert.False(t, operational.CardinalityMinGE1("abc"))
}

func TestCardinalityMinGE1_ValidValues(t *testing.T) {
	assert.True(t, operational.CardinalityMinGE1("1"))
	assert.True(t, operational.CardinalityMinGE1("1..n"))
	assert.True(t, operational.CardinalityMinGE1("2..5"))
	assert.False(t, operational.CardinalityMinGE1("0..n"))
	assert.False(t, operational.CardinalityMinGE1("0..1"))
}

func TestIsEmptyValue_UnknownType(t *testing.T) {
	val := &models.InstanceAttributeValue{ValueString: "something"}
	assert.True(t, operational.IsEmptyValue("unknown", val))
}

// === Coverage: IsEmptyValue — all base type branches ===

func TestIsEmptyValue_StringNonEmpty(t *testing.T) {
	val := &models.InstanceAttributeValue{ValueString: "hello"}
	assert.False(t, operational.IsEmptyValue(models.BaseTypeString, val))
}

func TestIsEmptyValue_StringEmpty(t *testing.T) {
	val := &models.InstanceAttributeValue{ValueString: ""}
	assert.True(t, operational.IsEmptyValue(models.BaseTypeString, val))
}

func TestIsEmptyValue_BooleanNonEmpty(t *testing.T) {
	val := &models.InstanceAttributeValue{ValueString: "true"}
	assert.False(t, operational.IsEmptyValue(models.BaseTypeBoolean, val))
}

func TestIsEmptyValue_BooleanEmpty(t *testing.T) {
	val := &models.InstanceAttributeValue{ValueString: ""}
	assert.True(t, operational.IsEmptyValue(models.BaseTypeBoolean, val))
}

func TestIsEmptyValue_DateNonEmpty(t *testing.T) {
	val := &models.InstanceAttributeValue{ValueString: "2026-04-12"}
	assert.False(t, operational.IsEmptyValue(models.BaseTypeDate, val))
}

func TestIsEmptyValue_DateEmpty(t *testing.T) {
	val := &models.InstanceAttributeValue{ValueString: ""}
	assert.True(t, operational.IsEmptyValue(models.BaseTypeDate, val))
}

func TestIsEmptyValue_URLNonEmpty(t *testing.T) {
	val := &models.InstanceAttributeValue{ValueString: "https://example.com"}
	assert.False(t, operational.IsEmptyValue(models.BaseTypeURL, val))
}

func TestIsEmptyValue_URLEmpty(t *testing.T) {
	val := &models.InstanceAttributeValue{ValueString: ""}
	assert.True(t, operational.IsEmptyValue(models.BaseTypeURL, val))
}

func TestIsEmptyValue_EnumNonEmpty(t *testing.T) {
	val := &models.InstanceAttributeValue{ValueString: "active"}
	assert.False(t, operational.IsEmptyValue(models.BaseTypeEnum, val))
}

func TestIsEmptyValue_EnumEmpty(t *testing.T) {
	val := &models.InstanceAttributeValue{ValueString: ""}
	assert.True(t, operational.IsEmptyValue(models.BaseTypeEnum, val))
}

func TestIsEmptyValue_NumberNonEmpty(t *testing.T) {
	num := 42.0
	val := &models.InstanceAttributeValue{ValueNumber: &num}
	assert.False(t, operational.IsEmptyValue(models.BaseTypeNumber, val))
}

func TestIsEmptyValue_NumberEmpty(t *testing.T) {
	val := &models.InstanceAttributeValue{ValueNumber: nil}
	assert.True(t, operational.IsEmptyValue(models.BaseTypeNumber, val))
}

func TestIsEmptyValue_IntegerNonEmpty(t *testing.T) {
	num := 99.0
	val := &models.InstanceAttributeValue{ValueNumber: &num}
	assert.False(t, operational.IsEmptyValue(models.BaseTypeInteger, val))
}

func TestIsEmptyValue_IntegerEmpty(t *testing.T) {
	val := &models.InstanceAttributeValue{ValueNumber: nil}
	assert.True(t, operational.IsEmptyValue(models.BaseTypeInteger, val))
}

func TestIsEmptyValue_ListNonEmpty(t *testing.T) {
	val := &models.InstanceAttributeValue{ValueJSON: `["a"]`}
	assert.False(t, operational.IsEmptyValue(models.BaseTypeList, val))
}

func TestIsEmptyValue_ListEmpty(t *testing.T) {
	val := &models.InstanceAttributeValue{ValueJSON: ""}
	assert.True(t, operational.IsEmptyValue(models.BaseTypeList, val))
}

func TestIsEmptyValue_JSONNonEmpty(t *testing.T) {
	val := &models.InstanceAttributeValue{ValueJSON: `{"k":"v"}`}
	assert.False(t, operational.IsEmptyValue(models.BaseTypeJSON, val))
}

func TestIsEmptyValue_JSONEmpty(t *testing.T) {
	val := &models.InstanceAttributeValue{ValueJSON: ""}
	assert.True(t, operational.IsEmptyValue(models.BaseTypeJSON, val))
}

// === Bug 2: Full cardinality validation (min + max, both directions) ===

// Max cardinality: association with target_cardinality "0..1" — 2 links → error
func TestValidation_MaxCardinalityExceeded(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupSingleEntityType(ctx)

	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "server-1"},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{
		{ID: "assoc1", Name: "primary-db", Type: models.AssociationTypeDirectional,
			TargetEntityTypeID: "et2", TargetCardinality: "0..1"},
	}, nil)
	m.linkRepo.On("GetForwardRefs", ctx, "inst1").Return([]*models.AssociationLink{
		{ID: "link1", AssociationID: "assoc1", SourceInstanceID: "inst1", TargetInstanceID: "inst2"},
		{ID: "link2", AssociationID: "assoc1", SourceInstanceID: "inst1", TargetInstanceID: "inst3"},
	}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusInvalid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusInvalid, result.Status)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0].Violation, "exceeds maximum")
}

// Max cardinality: exactly "1" means exactly one — 2 links → error
func TestValidation_ExactCardinalityExceeded(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupSingleEntityType(ctx)

	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "server-1"},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{
		{ID: "assoc1", Name: "runs-on", Type: models.AssociationTypeDirectional,
			TargetEntityTypeID: "et2", TargetCardinality: "1"},
	}, nil)
	m.linkRepo.On("GetForwardRefs", ctx, "inst1").Return([]*models.AssociationLink{
		{ID: "link1", AssociationID: "assoc1", SourceInstanceID: "inst1", TargetInstanceID: "inst2"},
		{ID: "link2", AssociationID: "assoc1", SourceInstanceID: "inst1", TargetInstanceID: "inst3"},
	}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusInvalid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusInvalid, result.Status)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0].Violation, "exceeds maximum")
}

// Source cardinality: bidirectional with source_cardinality "1" — target instance has no reverse links → error
func TestValidation_SourceCardinalityBidirectional(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupTwoEntityTypes(ctx)

	// et1 (Server) has bidirectional association to et2 (Tool) with source_cardinality "1"
	// meaning each Tool instance must be linked to at least one Server
	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "server-1"},
		{ID: "inst2", EntityTypeID: "et2", CatalogID: "c1", Name: "tool-1"},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv2").Return([]*models.Attribute{}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst2", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{
		{ID: "assoc1", Name: "uses-tool", Type: models.AssociationTypeBidirectional,
			TargetEntityTypeID: "et2", TargetCardinality: "0..n", SourceCardinality: "1"},
	}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv2").Return([]*models.Association{}, nil)
	// server-1 has no forward links
	m.linkRepo.On("GetForwardRefs", ctx, "inst1").Return([]*models.AssociationLink{}, nil)
	// tool-1 has no reverse links
	m.linkRepo.On("GetReverseRefs", ctx, "inst2").Return([]*models.AssociationLink{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusInvalid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusInvalid, result.Status)
	// tool-1 should have an error: source_cardinality "1" not satisfied
	var found bool
	for _, e := range result.Errors {
		if e.InstanceName == "tool-1" && e.Field == "uses-tool" {
			found = true
			assert.Contains(t, e.Violation, "source cardinality")
		}
	}
	assert.True(t, found, "should have source cardinality error for tool-1")
}

// Source cardinality: directional with source_cardinality "1" — target instance has reverse link → passes
func TestValidation_SourceCardinalitySatisfied(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupTwoEntityTypes(ctx)

	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "server-1"},
		{ID: "inst2", EntityTypeID: "et2", CatalogID: "c1", Name: "tool-1"},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv2").Return([]*models.Attribute{}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst2", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{
		{ID: "assoc1", Name: "uses-tool", Type: models.AssociationTypeDirectional,
			TargetEntityTypeID: "et2", TargetCardinality: "0..n", SourceCardinality: "1"},
	}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv2").Return([]*models.Association{}, nil)
	m.linkRepo.On("GetForwardRefs", ctx, "inst1").Return([]*models.AssociationLink{}, nil)
	// tool-1 has a reverse link (some server links to it)
	m.linkRepo.On("GetReverseRefs", ctx, "inst2").Return([]*models.AssociationLink{
		{ID: "link1", AssociationID: "assoc1", SourceInstanceID: "inst1", TargetInstanceID: "inst2"},
	}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusValid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusValid, result.Status)
	assert.Empty(t, result.Errors)
}

// ParseCardinality helper
func TestParseCardinality(t *testing.T) {
	tests := []struct {
		input    string
		wantMin  int
		wantMax  int
		wantUnbounded bool
	}{
		{"0..n", 0, 0, true},
		{"0..1", 0, 1, false},
		{"1", 1, 1, false},
		{"1..n", 1, 0, true},
		{"1..5", 1, 5, false},
		{"2..n", 2, 0, true},
		{"", 0, 0, true}, // default
	}
	for _, tt := range tests {
		min, max, unbounded := operational.ParseCardinality(tt.input)
		assert.Equal(t, tt.wantMin, min, "min for %q", tt.input)
		assert.Equal(t, tt.wantMax, max, "max for %q", tt.input)
		assert.Equal(t, tt.wantUnbounded, unbounded, "unbounded for %q", tt.input)
	}
}

// Source cardinality max exceeded
func TestValidation_SourceCardinalityMaxExceeded(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupTwoEntityTypes(ctx)

	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "server-1"},
		{ID: "inst2", EntityTypeID: "et1", CatalogID: "c1", Name: "server-2"},
		{ID: "inst3", EntityTypeID: "et2", CatalogID: "c1", Name: "tool-1"},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv2").Return([]*models.Attribute{}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst2", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst3", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	// Assoc: Server -> Tool with source_cardinality "0..1" (each Tool can be linked to by at most 1 Server)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{
		{ID: "assoc1", Name: "uses-tool", Type: models.AssociationTypeDirectional,
			TargetEntityTypeID: "et2", TargetCardinality: "0..n", SourceCardinality: "0..1"},
	}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv2").Return([]*models.Association{}, nil)
	m.linkRepo.On("GetForwardRefs", ctx, "inst1").Return([]*models.AssociationLink{}, nil)
	m.linkRepo.On("GetForwardRefs", ctx, "inst2").Return([]*models.AssociationLink{}, nil)
	// tool-1 has 2 reverse links (from server-1 and server-2) — exceeds max of 1
	m.linkRepo.On("GetReverseRefs", ctx, "inst3").Return([]*models.AssociationLink{
		{ID: "link1", AssociationID: "assoc1", SourceInstanceID: "inst1", TargetInstanceID: "inst3"},
		{ID: "link2", AssociationID: "assoc1", SourceInstanceID: "inst2", TargetInstanceID: "inst3"},
	}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusInvalid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusInvalid, result.Status)
	var found bool
	for _, e := range result.Errors {
		if e.InstanceName == "tool-1" && e.Field == "uses-tool" {
			found = true
			assert.Contains(t, e.Violation, "exceeds maximum")
		}
	}
	assert.True(t, found, "should have source cardinality max exceeded error for tool-1")
}

// GetReverseRefs error propagation in source cardinality check
func TestValidation_ReverseRefsError(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupTwoEntityTypes(ctx)

	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "server-1"},
		{ID: "inst2", EntityTypeID: "et2", CatalogID: "c1", Name: "tool-1"},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv2").Return([]*models.Attribute{}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst2", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{
		{ID: "assoc1", Name: "uses-tool", Type: models.AssociationTypeDirectional,
			TargetEntityTypeID: "et2", TargetCardinality: "0..n", SourceCardinality: "1"},
	}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv2").Return([]*models.Association{}, nil)
	m.linkRepo.On("GetForwardRefs", ctx, "inst1").Return([]*models.AssociationLink{}, nil)
	m.linkRepo.On("GetReverseRefs", ctx, "inst2").Return([]*models.AssociationLink{}, domainerrors.NewValidation("db"))

	_, err := svc.Validate(ctx, "my-catalog")
	assert.Error(t, err)
}

// ParseCardinality with invalid max (e.g., "1..abc")
func TestParseCardinality_InvalidMax(t *testing.T) {
	min, max, unbounded := operational.ParseCardinality("1..abc")
	assert.Equal(t, 1, min)
	assert.Equal(t, 0, max)
	assert.True(t, unbounded)
}

// Bug fix: Contained entity type instance without a parent should be flagged
func TestValidation_ContainedTypeWithoutParent(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupTwoEntityTypes(ctx)

	// et1 (Server) contains et2 (Tool) via containment association
	// Tool instance exists WITHOUT a parent → should produce error
	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et2", CatalogID: "c1", Name: "orphan-tool"},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv2").Return([]*models.Attribute{}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	// Server (et1/etv1) has containment association to Tool (et2)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{
		{ID: "assoc1", Name: "contains-tool", Type: models.AssociationTypeContainment, TargetEntityTypeID: "et2"},
	}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv2").Return([]*models.Association{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusInvalid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusInvalid, result.Status)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "Tool", result.Errors[0].EntityType)
	assert.Equal(t, "orphan-tool", result.Errors[0].InstanceName)
	assert.Equal(t, "parent", result.Errors[0].Field)
	assert.Contains(t, result.Errors[0].Violation, "contained entity type")
}

// === TD-22: System Attributes — Name Non-Empty Validation ===

// T-18.22: Validate returns error for instance with empty Name
func TestT18_22_ValidateEmptyName(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupSingleEntityType(ctx)

	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: ""},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusInvalid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusInvalid, result.Status)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "name", result.Errors[0].Field)
	assert.Contains(t, result.Errors[0].Violation, "required")
}

// T-18.23: Validate returns error for instance with whitespace-only Name
func TestT18_23_ValidateWhitespaceName(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupSingleEntityType(ctx)

	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "   "},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusInvalid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusInvalid, result.Status)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "name", result.Errors[0].Field)
}

// T-18.24: Validate passes for instance with non-empty Name
func TestT18_24_ValidateNonEmptyName(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupSingleEntityType(ctx)

	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "server-1"},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusValid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusValid, result.Status)
	assert.Empty(t, result.Errors)
}

// S2: Validation error for empty name uses instance ID as fallback
func TestValidation_EmptyNameUsesIDFallback(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupSingleEntityType(ctx)

	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst-abc", EntityTypeID: "et1", CatalogID: "c1", Name: ""},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst-abc", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusInvalid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	require.Len(t, result.Errors, 1)
	// InstanceName should contain the ID as fallback when name is empty
	assert.Contains(t, result.Errors[0].InstanceName, "inst-abc")
}

// T-18.25: Validation error includes correct entity type name
func TestT18_25_ValidateEmptyNameEntityType(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupSingleEntityType(ctx)

	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: ""},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusInvalid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "Server", result.Errors[0].EntityType)
}

// Verify: contained type WITH a parent does NOT produce this error
func TestValidation_ContainedTypeWithParentPasses(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupTwoEntityTypes(ctx)

	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "server-1"},
		{ID: "inst2", EntityTypeID: "et2", CatalogID: "c1", Name: "tool-1", ParentInstanceID: "inst1"},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv2").Return([]*models.Attribute{}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst2", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{
		{ID: "assoc1", Name: "contains-tool", Type: models.AssociationTypeContainment, TargetEntityTypeID: "et2"},
	}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv2").Return([]*models.Association{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusValid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusValid, result.Status)
	assert.Empty(t, result.Errors)
}

// T-31.XX: Corrupted type definition constraints produce a validation error
func TestT31_CorruptedConstraintsFlaggedByValidation(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()

	m.catRepo.On("GetByName", ctx, "corrupt-catalog").Return(&models.Catalog{
		ID: "c1", Name: "corrupt-catalog", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusDraft,
	}, nil)
	m.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
	}, nil)
	m.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{
		ID: "etv1", EntityTypeID: "et1", Version: 1,
	}, nil)
	m.etRepo.On("GetByID", ctx, "et1").Return(&models.EntityType{ID: "et1", Name: "Server"}, nil)

	// Attribute referencing a type definition with corrupted constraints
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{
		{ID: "a1", Name: "status", TypeDefinitionVersionID: "tdv-corrupt", Required: false},
	}, nil)
	m.tdvRepo.On("GetByID", ctx, "tdv-corrupt").Return(&models.TypeDefinitionVersion{
		ID: "tdv-corrupt", TypeDefinitionID: "td-corrupt",
		Constraints: map[string]any{"_raw": "not{valid-json"},
	}, nil)
	m.tdRepo.On("GetByID", ctx, "td-corrupt").Return(&models.TypeDefinition{
		ID: "td-corrupt", Name: "BadEnum", BaseType: models.BaseTypeEnum,
	}, nil)

	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{}, nil)
	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "i1", EntityTypeID: "et1", Name: "srv-1", Version: 1},
	}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "i1", mock.Anything).Return([]*models.InstanceAttributeValue{}, nil)
	m.linkRepo.On("GetForwardRefs", ctx, "i1").Return([]*models.AssociationLink{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusInvalid).Return(nil)

	result, err := svc.Validate(ctx, "corrupt-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusInvalid, result.Status)
	require.NotEmpty(t, result.Errors)
	assert.Contains(t, result.Errors[0].Violation, "corrupted")
}

// === TD-90: Constraint validation through Validate() ===

// T-31.47 integration: String exceeding max_length produces validation error through Validate()
func TestT31_47_StringMaxLengthViolation(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupSingleEntityType(ctx)

	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "server-1"},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{
		{ID: "attr1", Name: "hostname", TypeDefinitionVersionID: "tdv-maxlen", Required: false},
	}, nil)
	m.tdvRepo.On("GetByID", ctx, "tdv-maxlen").Return(&models.TypeDefinitionVersion{
		ID: "tdv-maxlen", TypeDefinitionID: "td-string",
		Constraints: map[string]any{"max_length": float64(5)},
	}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{
		{ID: "v1", InstanceID: "inst1", AttributeID: "attr1", ValueString: "toolong"},
	}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusInvalid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusInvalid, result.Status)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "hostname", result.Errors[0].Field)
	assert.Contains(t, result.Errors[0].Violation, "exceeds maximum length")
}

// T-31.48 integration: String not matching pattern produces validation error
func TestT31_48_StringPatternViolation(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupSingleEntityType(ctx)

	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "server-1"},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{
		{ID: "attr1", Name: "hostname", TypeDefinitionVersionID: "tdv-pattern", Required: false},
	}, nil)
	m.tdvRepo.On("GetByID", ctx, "tdv-pattern").Return(&models.TypeDefinitionVersion{
		ID: "tdv-pattern", TypeDefinitionID: "td-string",
		Constraints: map[string]any{"pattern": "^[a-z]+$"},
	}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{
		{ID: "v1", InstanceID: "inst1", AttributeID: "attr1", ValueString: "ABC123"},
	}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusInvalid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusInvalid, result.Status)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0].Violation, "does not match pattern")
}

// Bad regex pattern produces ONE error per attribute (not per instance)
func TestT31_BadPatternReportsOnce(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupSingleEntityType(ctx)

	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "server-1"},
		{ID: "inst2", EntityTypeID: "et1", CatalogID: "c1", Name: "server-2"},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{
		{ID: "attr1", Name: "hostname", TypeDefinitionVersionID: "tdv-badpat", Required: false},
	}, nil)
	m.tdvRepo.On("GetByID", ctx, "tdv-badpat").Return(&models.TypeDefinitionVersion{
		ID: "tdv-badpat", TypeDefinitionID: "td-string",
		Constraints: map[string]any{"pattern": "[invalid"},
	}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{
		{ID: "v1", InstanceID: "inst1", AttributeID: "attr1", ValueString: "abc"},
	}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst2", mock.Anything).Return([]*models.InstanceAttributeValue{
		{ID: "v2", InstanceID: "inst2", AttributeID: "attr1", ValueString: "def"},
	}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusInvalid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusInvalid, result.Status)
	// Only 1 error for the bad pattern, not 2 (one per instance)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0].Violation, "invalid pattern constraint")
	assert.Equal(t, "(schema)", result.Errors[0].InstanceName) // attribute-level, not instance-level
}

// T-31.50 integration: Integer not whole number through Validate()
func TestT31_50_IntegerNotWholeNumber(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupSingleEntityType(ctx)

	num := 3.14
	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "server-1"},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{
		{ID: "attr1", Name: "port", TypeDefinitionVersionID: "tdv-int", Required: false},
	}, nil)
	m.tdvRepo.On("GetByID", ctx, "tdv-int").Return(&models.TypeDefinitionVersion{
		ID: "tdv-int", TypeDefinitionID: "td-int",
	}, nil)
	m.tdRepo.On("GetByID", ctx, "td-int").Return(&models.TypeDefinition{
		ID: "td-int", Name: "Integer", BaseType: models.BaseTypeInteger,
	}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{
		{ID: "v1", InstanceID: "inst1", AttributeID: "attr1", ValueNumber: &num},
	}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusInvalid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusInvalid, result.Status)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0].Violation, "whole number")
}

// T-31.56 integration: Boolean invalid value through Validate()
func TestT31_56_BooleanInvalidValue(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupSingleEntityType(ctx)

	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "server-1"},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{
		{ID: "attr1", Name: "enabled", TypeDefinitionVersionID: "tdv-bool", Required: false},
	}, nil)
	m.tdvRepo.On("GetByID", ctx, "tdv-bool").Return(&models.TypeDefinitionVersion{
		ID: "tdv-bool", TypeDefinitionID: "td-bool",
	}, nil)
	m.tdRepo.On("GetByID", ctx, "td-bool").Return(&models.TypeDefinition{
		ID: "td-bool", Name: "Boolean", BaseType: models.BaseTypeBoolean,
	}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{
		{ID: "v1", InstanceID: "inst1", AttributeID: "attr1", ValueString: "yes"},
	}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusInvalid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusInvalid, result.Status)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0].Violation, "must be \"true\" or \"false\"")
}

// Constraint validation skipped for corrupted constraints
func TestT31_ConstraintsSkippedForCorrupted(t *testing.T) {
	svc, m := setupValidationService()
	ctx := context.Background()
	m.setupSingleEntityType(ctx)

	m.instRepo.On("ListByCatalog", ctx, "c1").Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1", CatalogID: "c1", Name: "server-1"},
	}, nil)
	m.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{
		{ID: "attr1", Name: "hostname", TypeDefinitionVersionID: "tdv-corrupt-str", Required: false},
	}, nil)
	m.tdvRepo.On("GetByID", ctx, "tdv-corrupt-str").Return(&models.TypeDefinitionVersion{
		ID: "tdv-corrupt-str", TypeDefinitionID: "td-string",
		Constraints: map[string]any{"_raw": "bad", "max_length": float64(1)},
	}, nil)
	m.iavRepo.On("GetValuesForVersion", ctx, "inst1", mock.Anything).Return([]*models.InstanceAttributeValue{
		{ID: "v1", InstanceID: "inst1", AttributeID: "attr1", ValueString: "toolong"},
	}, nil)
	m.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{}, nil)
	m.catRepo.On("UpdateValidationStatus", ctx, "c1", models.ValidationStatusInvalid).Return(nil)

	result, err := svc.Validate(ctx, "my-catalog")
	require.NoError(t, err)
	// Should have corrupted error but NOT a max_length error
	found := false
	for _, e := range result.Errors {
		if e.Field == "hostname" {
			assert.Contains(t, e.Violation, "corrupted")
			found = true
		}
	}
	assert.True(t, found, "should have corrupted constraint error")
	for _, e := range result.Errors {
		assert.NotContains(t, e.Violation, "exceeds maximum length")
	}
}
