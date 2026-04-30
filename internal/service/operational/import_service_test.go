package operational

import (
	"context"
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository/mocks"
)

func newImportService() (*ImportService, *mocks.MockCatalogRepo, *mocks.MockCatalogVersionRepo, *mocks.MockCatalogVersionPinRepo, *mocks.MockEntityTypeRepo, *mocks.MockEntityTypeVersionRepo, *mocks.MockAttributeRepo, *mocks.MockAssociationRepo, *mocks.MockTypeDefinitionRepo, *mocks.MockTypeDefinitionVersionRepo, *mocks.MockEntityInstanceRepo, *mocks.MockInstanceAttributeValueRepo, *mocks.MockAssociationLinkRepo, *mocks.MockCatalogVersionTypePinRepo) {
	catalogRepo := new(mocks.MockCatalogRepo)
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	instRepo := new(mocks.MockEntityInstanceRepo)
	iavRepo := new(mocks.MockInstanceAttributeValueRepo)
	linkRepo := new(mocks.MockAssociationLinkRepo)
	typePinRepo := new(mocks.MockCatalogVersionTypePinRepo)
	txManager := &mocks.MockTransactionManager{}

	svc := NewImportService(catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, instRepo, iavRepo, linkRepo, typePinRepo, WithImportTransactionManager(txManager))
	return svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, instRepo, iavRepo, linkRepo, typePinRepo
}

func minimalExportData() *ExportData {
	return &ExportData{
		FormatVersion: "1.0",
		ExportedAt:    time.Now(),
		SourceSystem:  "test",
		Catalog:       ExportCatalog{Name: "test-catalog", Description: "test"},
		CatalogVersion: ExportCatalogVersion{Label: "v1.0", Description: "first"},
		TypeDefinitions: []ExportTypeDef{},
		EntityTypes: []ExportEntityType{
			{Name: "server", Description: "A server", Attributes: []ExportAttribute{}, Associations: []ExportAssociation{}},
		},
		Instances: []ExportInstance{},
	}
}

// T-30.26: DryRun — no collisions, all new
func TestT30_26_DryRunNoCollisions(t *testing.T) {
	svc, catalogRepo, cvRepo, _, etRepo, _, _, _, _, _, _, _, _, _ := newImportService()

	catalogRepo.On("GetByName", ctx(), "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", ctx(), "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
	etRepo.On("GetByName", ctx(), "server").Return(nil, domainerrors.NewNotFound("EntityType", "server"))

	req := &ImportRequest{Data: minimalExportData()}
	result, err := svc.DryRun(context.Background(), req)
	require.NoError(t, err)

	assert.Equal(t, "ready", result.Status)
	assert.Equal(t, 3, result.Summary.New) // catalog + CV + entity type
	assert.Equal(t, 0, result.Summary.Conflicts)
	assert.Equal(t, 0, result.Summary.Identical)
}

// T-30.27: DryRun — catalog name conflict
func TestT30_27_DryRunCatalogConflict(t *testing.T) {
	svc, catalogRepo, cvRepo, _, etRepo, _, _, _, _, _, _, _, _, _ := newImportService()

	catalogRepo.On("GetByName", ctx(), "test-catalog").Return(&models.Catalog{ID: "c1", Name: "test-catalog"}, nil)
	cvRepo.On("GetByLabel", ctx(), "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
	etRepo.On("GetByName", ctx(), "server").Return(nil, domainerrors.NewNotFound("EntityType", "server"))

	req := &ImportRequest{Data: minimalExportData()}
	result, err := svc.DryRun(context.Background(), req)
	require.NoError(t, err)

	assert.Equal(t, "conflicts_found", result.Status)
	assert.Equal(t, 1, result.Summary.Conflicts)
	require.Len(t, result.Collisions, 1)
	assert.Equal(t, "catalog", result.Collisions[0].Type)
	assert.Equal(t, "conflict", result.Collisions[0].Resolution)
}

// T-30.28: DryRun — entity type identical at V1
func TestT30_28_DryRunEntityTypeIdentical(t *testing.T) {
	svc, catalogRepo, cvRepo, _, etRepo, etvRepo, attrRepo, assocRepo, _, _, _, _, _, _ := newImportService()

	catalogRepo.On("GetByName", ctx(), "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", ctx(), "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))

	existingET := &models.EntityType{ID: "et-1", Name: "server"}
	existingETV := &models.EntityTypeVersion{ID: "etv-1", EntityTypeID: "et-1", Version: 1}
	etRepo.On("GetByName", ctx(), "server").Return(existingET, nil)
	etvRepo.On("GetLatestByEntityType", ctx(), "et-1").Return(existingETV, nil)
	attrRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Association{}, nil)

	req := &ImportRequest{Data: minimalExportData()}
	result, err := svc.DryRun(context.Background(), req)
	require.NoError(t, err)

	assert.Equal(t, "ready", result.Status) // identical is not a conflict
	assert.Equal(t, 1, result.Summary.Identical)
	var etCollision *Collision
	for i, c := range result.Collisions {
		if c.Type == "entity_type" {
			etCollision = &result.Collisions[i]
			break
		}
	}
	require.NotNil(t, etCollision)
	assert.Equal(t, "identical", etCollision.Resolution)
	assert.Equal(t, 1, etCollision.Version)
}

// T-30.29: DryRun — entity type conflict (different attributes)
func TestT30_29_DryRunEntityTypeConflict(t *testing.T) {
	svc, catalogRepo, cvRepo, _, etRepo, etvRepo, attrRepo, assocRepo, tdvRepo, tdRepo, _, _, _, _ := newImportService()

	catalogRepo.On("GetByName", ctx(), "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", ctx(), "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))

	existingET := &models.EntityType{ID: "et-1", Name: "server"}
	existingETV := &models.EntityTypeVersion{ID: "etv-1", EntityTypeID: "et-1", Version: 1}
	existingAttr := &models.Attribute{ID: "attr-1", Name: "endpoint", TypeDefinitionVersionID: "tdv-url", Required: true}
	systemTD := &models.TypeDefinition{ID: "td-url", Name: "url", BaseType: models.BaseTypeURL, System: true}
	systemTDV := &models.TypeDefinitionVersion{ID: "tdv-url", TypeDefinitionID: "td-url"}

	etRepo.On("GetByName", ctx(), "server").Return(existingET, nil)
	etvRepo.On("GetLatestByEntityType", ctx(), "et-1").Return(existingETV, nil)
	attrRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Attribute{existingAttr}, nil)
	assocRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Association{}, nil)
	tdvRepo.On("GetByID", ctx(), "tdv-url").Return(systemTDV, nil)
	tdRepo.On("GetByID", ctx(), "td-url").Return(systemTD, nil)

	// Import data has 0 attributes, existing has 1 → conflict
	req := &ImportRequest{Data: minimalExportData()}
	result, err := svc.DryRun(context.Background(), req)
	require.NoError(t, err)

	assert.Equal(t, "conflicts_found", result.Status)
	assert.Equal(t, 1, result.Summary.Conflicts)
}

// T-30.30: DryRun — rename_map applied
func TestT30_30_DryRunWithRenameMap(t *testing.T) {
	svc, catalogRepo, cvRepo, _, etRepo, _, _, _, _, _, _, _, _, _ := newImportService()

	catalogRepo.On("GetByName", ctx(), "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", ctx(), "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
	etRepo.On("GetByName", ctx(), "renamed-server").Return(nil, domainerrors.NewNotFound("EntityType", "renamed-server"))

	req := &ImportRequest{
		Data: minimalExportData(),
		RenameMap: &ImportRenameMap{
			EntityTypes: map[string]string{"server": "renamed-server"},
		},
	}
	result, err := svc.DryRun(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "ready", result.Status)
}

// T-30.31: DryRun — catalog_name override
func TestT30_31_DryRunCatalogNameOverride(t *testing.T) {
	svc, catalogRepo, cvRepo, _, etRepo, _, _, _, _, _, _, _, _, _ := newImportService()

	// "my-copy" (override) doesn't exist
	catalogRepo.On("GetByName", ctx(), "my-copy").Return(nil, domainerrors.NewNotFound("Catalog", "my-copy"))
	cvRepo.On("GetByLabel", ctx(), "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
	etRepo.On("GetByName", ctx(), "server").Return(nil, domainerrors.NewNotFound("EntityType", "server"))

	req := &ImportRequest{
		CatalogName: "my-copy",
		Data:        minimalExportData(),
	}
	result, err := svc.DryRun(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "ready", result.Status)
}

// T-30.32: DryRun — invalid format version
func TestT30_32_DryRunInvalidFormatVersion(t *testing.T) {
	svc, _, _, _, _, _, _, _, _, _, _, _, _, _ := newImportService()

	data := minimalExportData()
	data.FormatVersion = "2.0"
	req := &ImportRequest{Data: data}
	_, err := svc.DryRun(context.Background(), req)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

// T-30.33: DryRun — nil data
func TestT30_33_DryRunNilData(t *testing.T) {
	svc, _, _, _, _, _, _, _, _, _, _, _, _, _ := newImportService()

	req := &ImportRequest{Data: nil}
	_, err := svc.DryRun(context.Background(), req)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

// T-30.40: Import — minimal catalog, one entity type, no instances
func TestT30_40_ImportMinimalCatalog(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, instRepo, _, _, typePinRepo := newImportService()

	catalogRepo.On("GetByName", ctx(), "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", ctx(), "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))

	// System types list for resolving attrs
	systemTDs := []*models.TypeDefinition{
		{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true},
	}
	systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}
	tdRepo.On("List", mock.Anything, mock.Anything).Return(systemTDs, 1, nil)
	tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-str").Return(systemTDV, nil)

	// Entity type doesn't exist — will be created
	etRepo.On("GetByName", mock.Anything, "server").Return(nil, domainerrors.NewNotFound("EntityType", "server"))
	etRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityType")).Return(nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)

	// CV creation
	cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)

	// Pin creation
	pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)

	// Type pin creation
	typePinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionTypePin")).Return(nil)

	// No attributes or associations on the ET, so ListByVersion returns empty
	attrRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Association{}, nil)

	// Catalog creation
	catalogRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Catalog")).Return(nil)

	// No instances
	instRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityInstance")).Return(nil)

	req := &ImportRequest{Data: minimalExportData()}
	result, err := svc.Import(context.Background(), req)
	require.NoError(t, err)

	assert.Equal(t, "success", result.Status)
	assert.Equal(t, "test-catalog", result.CatalogName)
	assert.NotEmpty(t, result.CatalogID)
	assert.Equal(t, 1, result.TypesCreated) // entity type
	assert.Equal(t, 0, result.TypesReused)
	assert.Equal(t, 0, result.InstancesCreated)

	catalogRepo.AssertCalled(t, "Create", mock.Anything, mock.AnythingOfType("*models.Catalog"))
	cvRepo.AssertCalled(t, "Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion"))
}

// T-30.34: Import — with type definitions
func TestT30_34_ImportWithTypeDefs(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, _, _, _, typePinRepo := newImportService()

	data := minimalExportData()
	data.TypeDefinitions = []ExportTypeDef{
		{Name: "hex12", Description: "12-char hex", BaseType: "string", System: false, Constraints: map[string]any{"max_length": float64(12)}},
	}
	data.EntityTypes[0].Attributes = []ExportAttribute{
		{Name: "id", TypeDefinition: "hex12", Required: true, Ordinal: 0},
	}

	catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))

	// Type definition doesn't exist
	tdRepo.On("GetByName", mock.Anything, "hex12").Return(nil, domainerrors.NewNotFound("TypeDefinition", "hex12"))
	tdRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.TypeDefinition")).Return(nil)
	tdvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.TypeDefinitionVersion")).Return(nil)

	// System types
	systemTDs := []*models.TypeDefinition{
		{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true},
	}
	systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}
	tdRepo.On("List", mock.Anything, mock.Anything).Return(systemTDs, 1, nil)
	tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-str").Return(systemTDV, nil)

	// Entity type
	etRepo.On("GetByName", mock.Anything, "server").Return(nil, domainerrors.NewNotFound("EntityType", "server"))
	etRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityType")).Return(nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
	attrRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Attribute")).Return(nil)
	assocRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Association{}, nil)
	attrRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Attribute{}, nil)

	cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
	pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)
	typePinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionTypePin")).Return(nil)
	catalogRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Catalog")).Return(nil)

	req := &ImportRequest{Data: data}
	result, err := svc.Import(context.Background(), req)
	require.NoError(t, err)

	assert.Equal(t, "success", result.Status)
	assert.Equal(t, 2, result.TypesCreated) // type def + entity type
	tdRepo.AssertCalled(t, "Create", mock.Anything, mock.AnythingOfType("*models.TypeDefinition"))
	tdvRepo.AssertCalled(t, "Create", mock.Anything, mock.AnythingOfType("*models.TypeDefinitionVersion"))
}

// T-30.35: DryRun — type definition identical
func TestT30_35_DryRunTypeDefIdentical(t *testing.T) {
	svc, catalogRepo, cvRepo, _, etRepo, _, _, _, tdRepo, tdvRepo, _, _, _, _ := newImportService()

	data := minimalExportData()
	data.TypeDefinitions = []ExportTypeDef{
		{Name: "hex12", Description: "12-char hex", BaseType: "string", Constraints: map[string]any{"max_length": float64(12)}},
	}

	catalogRepo.On("GetByName", ctx(), "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", ctx(), "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
	etRepo.On("GetByName", ctx(), "server").Return(nil, domainerrors.NewNotFound("EntityType", "server"))

	existingTD := &models.TypeDefinition{ID: "td-1", Name: "hex12", Description: "12-char hex", BaseType: models.BaseTypeString}
	existingTDV := &models.TypeDefinitionVersion{ID: "tdv-1", TypeDefinitionID: "td-1", VersionNumber: 1, Constraints: map[string]any{"max_length": float64(12)}}
	tdRepo.On("GetByName", ctx(), "hex12").Return(existingTD, nil)
	tdvRepo.On("GetLatestByTypeDefinition", ctx(), "td-1").Return(existingTDV, nil)

	req := &ImportRequest{Data: data}
	result, err := svc.DryRun(context.Background(), req)
	require.NoError(t, err)

	assert.Equal(t, "ready", result.Status) // identical is not a conflict
	assert.Equal(t, 1, result.Summary.Identical)
}

// T-30.36: ApplyMassRename
func TestT30_36_ApplyMassRename(t *testing.T) {
	entityTypes := []ExportEntityType{
		{Name: "server"},
		{Name: "tool"},
	}
	typeDefs := []ExportTypeDef{
		{Name: "hex12"},
	}

	renameMap := ApplyMassRename(nil, entityTypes, typeDefs, "imported-", "")
	assert.Equal(t, "imported-server", renameMap.EntityTypes["server"])
	assert.Equal(t, "imported-tool", renameMap.EntityTypes["tool"])
	assert.Equal(t, "imported-hex12", renameMap.TypeDefinitions["hex12"])
}

// T-30.37: Import — invalid catalog name
func TestT30_37_ImportInvalidCatalogName(t *testing.T) {
	svc, _, _, _, _, _, _, _, _, _, _, _, _, _ := newImportService()

	data := minimalExportData()
	data.Catalog.Name = "INVALID_NAME!"
	req := &ImportRequest{Data: data}
	_, err := svc.Import(context.Background(), req)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

// T-30.39: Import with instances and links — linksCreated counter
func TestT30_39_ImportLinksCreatedCounter(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, instRepo, iavRepo, linkRepo, typePinRepo := newImportService()

	data := minimalExportData()
	data.EntityTypes = []ExportEntityType{
		{Name: "server", Description: "server", Attributes: []ExportAttribute{}, Associations: []ExportAssociation{
			{Name: "pre-execute", Type: "directional", Target: "guard", SourceCardinality: "0..n", TargetCardinality: "0..n"},
		}},
		{Name: "guard", Description: "guard", Attributes: []ExportAttribute{}, Associations: []ExportAssociation{}},
	}
	data.Instances = []ExportInstance{
		{EntityType: "server", Name: "s1", Attributes: map[string]any{}, Links: []ExportLink{
			{Association: "pre-execute", TargetType: "guard", TargetName: "g1"},
		}, Children: make(map[string][]*ExportInstance)},
		{EntityType: "guard", Name: "g1", Attributes: map[string]any{}, Links: nil, Children: make(map[string][]*ExportInstance)},
	}

	catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))

	systemTDs := []*models.TypeDefinition{{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}}
	systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}
	tdRepo.On("List", mock.Anything, mock.Anything).Return(systemTDs, 1, nil)
	tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-str").Return(systemTDV, nil)

	etRepo.On("GetByName", mock.Anything, "server").Return(nil, domainerrors.NewNotFound("EntityType", "server"))
	etRepo.On("GetByName", mock.Anything, "guard").Return(nil, domainerrors.NewNotFound("EntityType", "guard"))
	etRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityType")).Return(nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
	cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
	pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)
	typePinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionTypePin")).Return(nil)
	assocRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Association")).Return(nil)
	// ListByVersion for link resolution — return the created association
	assocRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Association{
		{ID: "assoc-created", Name: "pre-execute", Type: models.AssociationTypeDirectional, TargetEntityTypeID: "et-guard"},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Attribute{}, nil)
	catalogRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Catalog")).Return(nil)
	instRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityInstance")).Return(nil)
	iavRepo.On("SetValues", mock.Anything, mock.Anything).Return(nil)
	linkRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.AssociationLink")).Return(nil)

	req := &ImportRequest{Data: data}
	result, err := svc.Import(context.Background(), req)
	require.NoError(t, err)

	assert.Equal(t, "success", result.Status)
	assert.Equal(t, 2, result.InstancesCreated)
	assert.Equal(t, 1, result.LinksCreated, "LinksCreated should be 1, not 0")
}

// T-30.38: constraintsEqual
func TestT30_38_ConstraintsEqual(t *testing.T) {
	assert.True(t, constraintsEqual(nil, nil))
	assert.True(t, constraintsEqual(map[string]any{}, map[string]any{}))
	assert.True(t, constraintsEqual(
		map[string]any{"max_length": float64(12)},
		map[string]any{"max_length": float64(12)},
	))
	assert.False(t, constraintsEqual(
		map[string]any{"max_length": float64(12)},
		map[string]any{"max_length": float64(24)},
	))
	assert.True(t, constraintsEqual(nil, map[string]any{}))
	assert.True(t, constraintsEqual(map[string]any{}, nil))

	// json.Marshal error paths — math.Inf is not marshalable
	assert.False(t, constraintsEqual(
		map[string]any{"bad": math.Inf(1)},
		map[string]any{"ok": float64(1)},
	))
	assert.False(t, constraintsEqual(
		map[string]any{"ok": float64(1)},
		map[string]any{"bad": math.Inf(1)},
	))
}

// T-30.41: applyRenames — entity types + type definitions + associations + instances + links + children
func TestT30_41_ApplyRenames(t *testing.T) {
	svc, _, _, _, _, _, _, _, _, _, _, _, _, _ := newImportService()

	data := &ExportData{
		FormatVersion: "1.0",
		TypeDefinitions: []ExportTypeDef{
			{Name: "hex12", BaseType: "string"},
		},
		EntityTypes: []ExportEntityType{
			{
				Name: "server",
				Attributes: []ExportAttribute{
					{Name: "id", TypeDefinition: "hex12"},
				},
				Associations: []ExportAssociation{
					{Name: "deploys-to", Target: "platform"},
				},
			},
			{Name: "platform"},
		},
		Instances: []ExportInstance{
			{
				EntityType: "server",
				Name:       "s1",
				Links: []ExportLink{
					{Association: "deploys-to", TargetType: "platform", TargetName: "p1"},
				},
				Children: map[string][]*ExportInstance{
					"tools": {
						{EntityType: "server", Name: "sub1", Children: make(map[string][]*ExportInstance)},
					},
				},
			},
			{EntityType: "platform", Name: "p1", Children: make(map[string][]*ExportInstance)},
		},
	}

	renameMap := &ImportRenameMap{
		EntityTypes:     map[string]string{"server": "svc", "platform": "infra"},
		TypeDefinitions: map[string]string{"hex12": "hex-custom"},
	}

	svc.applyRenames(data, renameMap)

	// Type definitions renamed
	assert.Equal(t, "hex-custom", data.TypeDefinitions[0].Name)
	// Attribute type definition references renamed
	assert.Equal(t, "hex-custom", data.EntityTypes[0].Attributes[0].TypeDefinition)
	// Entity types renamed
	assert.Equal(t, "svc", data.EntityTypes[0].Name)
	assert.Equal(t, "infra", data.EntityTypes[1].Name)
	// Association targets renamed
	assert.Equal(t, "infra", data.EntityTypes[0].Associations[0].Target)
	// Instance entity_type renamed
	assert.Equal(t, "svc", data.Instances[0].EntityType)
	assert.Equal(t, "infra", data.Instances[1].EntityType)
	// Instance link target types renamed
	assert.Equal(t, "infra", data.Instances[0].Links[0].TargetType)
	// Children entity types renamed recursively
	assert.Equal(t, "svc", data.Instances[0].Children["tools"][0].EntityType)
}

// T-30.42: applyRenames — nil renameMap is noop
func TestT30_42_ApplyRenamesNil(t *testing.T) {
	svc, _, _, _, _, _, _, _, _, _, _, _, _, _ := newImportService()
	data := minimalExportData()
	svc.applyRenames(data, nil)
	assert.Equal(t, "server", data.EntityTypes[0].Name)
}

// T-30.43: DryRun — CV label collision
func TestT30_43_DryRunCVLabelCollision(t *testing.T) {
	svc, catalogRepo, cvRepo, _, etRepo, _, _, _, _, _, _, _, _, _ := newImportService()

	catalogRepo.On("GetByName", ctx(), "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", ctx(), "v1.0").Return(&models.CatalogVersion{ID: "cv-1"}, nil) // EXISTS
	etRepo.On("GetByName", ctx(), "server").Return(nil, domainerrors.NewNotFound("EntityType", "server"))

	req := &ImportRequest{Data: minimalExportData()}
	result, err := svc.DryRun(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "conflicts_found", result.Status)
	assert.Equal(t, 1, result.Summary.Conflicts)
}

// T-30.44: DryRun — cv_label override
func TestT30_44_DryRunCVLabelOverride(t *testing.T) {
	svc, catalogRepo, cvRepo, _, etRepo, _, _, _, _, _, _, _, _, _ := newImportService()

	catalogRepo.On("GetByName", ctx(), "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", ctx(), "custom-label").Return(nil, domainerrors.NewNotFound("CatalogVersion", "custom-label"))
	etRepo.On("GetByName", ctx(), "server").Return(nil, domainerrors.NewNotFound("EntityType", "server"))

	req := &ImportRequest{
		CatalogVersionLabel: "custom-label",
		Data:                minimalExportData(),
	}
	result, err := svc.DryRun(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "ready", result.Status)
}

// T-30.45: DryRun — type definition conflict (different constraints)
func TestT30_45_DryRunTypeDefConflict(t *testing.T) {
	svc, catalogRepo, cvRepo, _, etRepo, _, _, _, tdRepo, tdvRepo, _, _, _, _ := newImportService()

	data := minimalExportData()
	data.TypeDefinitions = []ExportTypeDef{
		{Name: "hex12", Description: "12-char hex", BaseType: "string", Constraints: map[string]any{"max_length": float64(12)}},
	}

	catalogRepo.On("GetByName", ctx(), "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", ctx(), "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
	etRepo.On("GetByName", ctx(), "server").Return(nil, domainerrors.NewNotFound("EntityType", "server"))

	existingTD := &models.TypeDefinition{ID: "td-1", Name: "hex12", Description: "12-char hex", BaseType: models.BaseTypeString}
	existingTDV := &models.TypeDefinitionVersion{ID: "tdv-1", TypeDefinitionID: "td-1", VersionNumber: 1, Constraints: map[string]any{"max_length": float64(24)}} // DIFFERENT
	tdRepo.On("GetByName", ctx(), "hex12").Return(existingTD, nil)
	tdvRepo.On("GetLatestByTypeDefinition", ctx(), "td-1").Return(existingTDV, nil)

	req := &ImportRequest{Data: data}
	result, err := svc.DryRun(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "conflicts_found", result.Status)
	assert.Equal(t, 1, result.Summary.Conflicts)
}

// T-30.46: DryRun — repo error paths
func TestT30_46_DryRunRepoErrors(t *testing.T) {
	someErr := domainerrors.NewNotFound("x", "y")

	t.Run("catalog_repo_error", func(t *testing.T) {
		svc, catalogRepo, _, _, _, _, _, _, _, _, _, _, _, _ := newImportService()
		catalogRepo.On("GetByName", ctx(), "test-catalog").Return(nil, assert.AnError)
		req := &ImportRequest{Data: minimalExportData()}
		_, err := svc.DryRun(context.Background(), req)
		assert.Error(t, err)
	})

	t.Run("cv_repo_error", func(t *testing.T) {
		svc, catalogRepo, cvRepo, _, _, _, _, _, _, _, _, _, _, _ := newImportService()
		catalogRepo.On("GetByName", ctx(), "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
		cvRepo.On("GetByLabel", ctx(), "v1.0").Return(nil, assert.AnError)
		req := &ImportRequest{Data: minimalExportData()}
		_, err := svc.DryRun(context.Background(), req)
		assert.Error(t, err)
	})

	t.Run("td_repo_error", func(t *testing.T) {
		svc, catalogRepo, cvRepo, _, etRepo, _, _, _, tdRepo, _, _, _, _, _ := newImportService()
		data := minimalExportData()
		data.TypeDefinitions = []ExportTypeDef{{Name: "hex", BaseType: "string"}}
		catalogRepo.On("GetByName", ctx(), "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
		cvRepo.On("GetByLabel", ctx(), "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
		etRepo.On("GetByName", ctx(), "server").Return(nil, domainerrors.NewNotFound("EntityType", "server"))
		tdRepo.On("GetByName", ctx(), "hex").Return(nil, assert.AnError) // Non-NotFound error
		req := &ImportRequest{Data: data}
		_, err := svc.DryRun(context.Background(), req)
		assert.Error(t, err)
	})

	t.Run("tdv_repo_error", func(t *testing.T) {
		svc, catalogRepo, cvRepo, _, etRepo, _, _, _, tdRepo, tdvRepo, _, _, _, _ := newImportService()
		data := minimalExportData()
		data.TypeDefinitions = []ExportTypeDef{{Name: "hex", BaseType: "string"}}
		catalogRepo.On("GetByName", ctx(), "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
		cvRepo.On("GetByLabel", ctx(), "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
		etRepo.On("GetByName", ctx(), "server").Return(nil, domainerrors.NewNotFound("EntityType", "server"))
		tdRepo.On("GetByName", ctx(), "hex").Return(&models.TypeDefinition{ID: "td-1"}, nil)
		tdvRepo.On("GetLatestByTypeDefinition", ctx(), "td-1").Return(nil, someErr)
		req := &ImportRequest{Data: data}
		_, err := svc.DryRun(context.Background(), req)
		assert.Error(t, err)
	})

	t.Run("et_repo_error", func(t *testing.T) {
		svc, catalogRepo, cvRepo, _, etRepo, _, _, _, _, _, _, _, _, _ := newImportService()
		catalogRepo.On("GetByName", ctx(), "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
		cvRepo.On("GetByLabel", ctx(), "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
		etRepo.On("GetByName", ctx(), "server").Return(nil, assert.AnError) // non-notfound
		req := &ImportRequest{Data: minimalExportData()}
		_, err := svc.DryRun(context.Background(), req)
		assert.Error(t, err)
	})

	t.Run("etv_repo_error", func(t *testing.T) {
		svc, catalogRepo, cvRepo, _, etRepo, etvRepo, _, _, _, _, _, _, _, _ := newImportService()
		catalogRepo.On("GetByName", ctx(), "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
		cvRepo.On("GetByLabel", ctx(), "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
		etRepo.On("GetByName", ctx(), "server").Return(&models.EntityType{ID: "et-1"}, nil)
		etvRepo.On("GetLatestByEntityType", ctx(), "et-1").Return(nil, someErr)
		req := &ImportRequest{Data: minimalExportData()}
		_, err := svc.DryRun(context.Background(), req)
		assert.Error(t, err)
	})

	t.Run("attr_repo_error", func(t *testing.T) {
		svc, catalogRepo, cvRepo, _, etRepo, etvRepo, attrRepo, _, _, _, _, _, _, _ := newImportService()
		catalogRepo.On("GetByName", ctx(), "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
		cvRepo.On("GetByLabel", ctx(), "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
		etRepo.On("GetByName", ctx(), "server").Return(&models.EntityType{ID: "et-1"}, nil)
		etvRepo.On("GetLatestByEntityType", ctx(), "et-1").Return(&models.EntityTypeVersion{ID: "etv-1"}, nil)
		attrRepo.On("ListByVersion", ctx(), "etv-1").Return(nil, someErr)
		req := &ImportRequest{Data: minimalExportData()}
		_, err := svc.DryRun(context.Background(), req)
		assert.Error(t, err)
	})

	t.Run("assoc_repo_error", func(t *testing.T) {
		svc, catalogRepo, cvRepo, _, etRepo, etvRepo, attrRepo, assocRepo, _, _, _, _, _, _ := newImportService()
		catalogRepo.On("GetByName", ctx(), "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
		cvRepo.On("GetByLabel", ctx(), "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
		etRepo.On("GetByName", ctx(), "server").Return(&models.EntityType{ID: "et-1"}, nil)
		etvRepo.On("GetLatestByEntityType", ctx(), "et-1").Return(&models.EntityTypeVersion{ID: "etv-1"}, nil)
		attrRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Attribute{}, nil)
		assocRepo.On("ListByVersion", ctx(), "etv-1").Return(nil, someErr)
		req := &ImportRequest{Data: minimalExportData()}
		_, err := svc.DryRun(context.Background(), req)
		assert.Error(t, err)
	})
}

// T-30.47: DryRun — reuse_existing skips entity type check
func TestT30_47_DryRunReuseExisting(t *testing.T) {
	svc, catalogRepo, cvRepo, _, _, _, _, _, _, _, _, _, _, _ := newImportService()

	catalogRepo.On("GetByName", ctx(), "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", ctx(), "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
	// No etRepo.GetByName mock needed — it should be skipped

	req := &ImportRequest{
		ReuseExisting: []string{"server"},
		Data:          minimalExportData(),
	}
	result, err := svc.DryRun(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "ready", result.Status)
}

// T-30.48: isTypeDefIdentical — base type mismatch and description mismatch
func TestT30_48_IsTypeDefIdentical(t *testing.T) {
	td := &models.TypeDefinition{ID: "td-1", Name: "hex", Description: "desc", BaseType: models.BaseTypeString}
	tdv := &models.TypeDefinitionVersion{ID: "tdv-1", TypeDefinitionID: "td-1", VersionNumber: 1, Constraints: map[string]any{"max_length": float64(12)}}

	// Identical
	assert.True(t, isTypeDefIdentical(td, tdv, ExportTypeDef{Name: "hex", Description: "desc", BaseType: "string", Constraints: map[string]any{"max_length": float64(12)}}))
	// Base type mismatch
	assert.False(t, isTypeDefIdentical(td, tdv, ExportTypeDef{Name: "hex", Description: "desc", BaseType: "number", Constraints: map[string]any{"max_length": float64(12)}}))
	// Description mismatch
	assert.False(t, isTypeDefIdentical(td, tdv, ExportTypeDef{Name: "hex", Description: "different", BaseType: "string", Constraints: map[string]any{"max_length": float64(12)}}))
}

// T-30.49: isEntityTypeIdentical — full comparison paths
func TestT30_49_IsEntityTypeIdentical(t *testing.T) {
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	etRepo := new(mocks.MockEntityTypeRepo)

	strTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str"}
	strTD := &models.TypeDefinition{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}
	tdvRepo.On("GetByID", mock.Anything, "tdv-str").Return(strTDV, nil)
	tdRepo.On("GetByID", mock.Anything, "td-str").Return(strTD, nil)

	targetET := &models.EntityType{ID: "et-target", Name: "tool"}
	etRepo.On("GetByID", mock.Anything, "et-target").Return(targetET, nil)

	existingAttrs := []*models.Attribute{
		{ID: "attr-1", Name: "endpoint", TypeDefinitionVersionID: "tdv-str", Required: true},
	}
	existingAssocs := []*models.Association{
		{ID: "assoc-1", Name: "dep", TargetEntityTypeID: "et-target", Type: models.AssociationTypeDirectional},
	}

	importET := ExportEntityType{
		Name: "server",
		Attributes: []ExportAttribute{
			{Name: "endpoint", TypeDefinition: "string", Required: true},
		},
		Associations: []ExportAssociation{
			{Name: "dep", Type: "directional", Target: "tool"},
		},
	}

	// Identical
	assert.True(t, isEntityTypeIdentical(existingAttrs, existingAssocs, importET, tdvRepo, tdRepo, etRepo, context.Background()))

	// Attribute count mismatch
	assert.False(t, isEntityTypeIdentical([]*models.Attribute{}, existingAssocs, importET, tdvRepo, tdRepo, etRepo, context.Background()))

	// Association count mismatch
	assert.False(t, isEntityTypeIdentical(existingAttrs, []*models.Association{}, importET, tdvRepo, tdRepo, etRepo, context.Background()))

	// Attribute name mismatch
	differentAttrET := ExportEntityType{
		Name: "server",
		Attributes: []ExportAttribute{
			{Name: "url", TypeDefinition: "string", Required: true},
		},
		Associations: []ExportAssociation{
			{Name: "dep", Type: "directional", Target: "tool"},
		},
	}
	assert.False(t, isEntityTypeIdentical(existingAttrs, existingAssocs, differentAttrET, tdvRepo, tdRepo, etRepo, context.Background()))

	// Attribute required mismatch
	differentReqET := ExportEntityType{
		Name: "server",
		Attributes: []ExportAttribute{
			{Name: "endpoint", TypeDefinition: "string", Required: false},
		},
		Associations: []ExportAssociation{
			{Name: "dep", Type: "directional", Target: "tool"},
		},
	}
	assert.False(t, isEntityTypeIdentical(existingAttrs, existingAssocs, differentReqET, tdvRepo, tdRepo, etRepo, context.Background()))

	// Attribute type mismatch
	differentTypeET := ExportEntityType{
		Name: "server",
		Attributes: []ExportAttribute{
			{Name: "endpoint", TypeDefinition: "number", Required: true},
		},
		Associations: []ExportAssociation{
			{Name: "dep", Type: "directional", Target: "tool"},
		},
	}
	assert.False(t, isEntityTypeIdentical(existingAttrs, existingAssocs, differentTypeET, tdvRepo, tdRepo, etRepo, context.Background()))

	// Association name mismatch
	differentAssocET := ExportEntityType{
		Name: "server",
		Attributes: []ExportAttribute{
			{Name: "endpoint", TypeDefinition: "string", Required: true},
		},
		Associations: []ExportAssociation{
			{Name: "other", Type: "directional", Target: "tool"},
		},
	}
	assert.False(t, isEntityTypeIdentical(existingAttrs, existingAssocs, differentAssocET, tdvRepo, tdRepo, etRepo, context.Background()))

	// Association type mismatch
	differentAssocTypeET := ExportEntityType{
		Name: "server",
		Attributes: []ExportAttribute{
			{Name: "endpoint", TypeDefinition: "string", Required: true},
		},
		Associations: []ExportAssociation{
			{Name: "dep", Type: "bidirectional", Target: "tool"},
		},
	}
	assert.False(t, isEntityTypeIdentical(existingAttrs, existingAssocs, differentAssocTypeET, tdvRepo, tdRepo, etRepo, context.Background()))

	// Association target mismatch
	differentAssocTargetET := ExportEntityType{
		Name: "server",
		Attributes: []ExportAttribute{
			{Name: "endpoint", TypeDefinition: "string", Required: true},
		},
		Associations: []ExportAssociation{
			{Name: "dep", Type: "directional", Target: "other-entity"},
		},
	}
	etRepo.On("GetByID", mock.Anything, "et-target").Return(targetET, nil)
	assert.False(t, isEntityTypeIdentical(existingAttrs, existingAssocs, differentAssocTargetET, tdvRepo, tdRepo, etRepo, context.Background()))

	// TDV lookup error → false
	badTDVRepo := new(mocks.MockTypeDefinitionVersionRepo)
	badTDVRepo.On("GetByID", mock.Anything, "tdv-str").Return(nil, assert.AnError)
	assert.False(t, isEntityTypeIdentical(existingAttrs, existingAssocs, importET, badTDVRepo, tdRepo, etRepo, context.Background()))

	// TD lookup error → false
	badTDRepo := new(mocks.MockTypeDefinitionRepo)
	tdvRepo2 := new(mocks.MockTypeDefinitionVersionRepo)
	tdvRepo2.On("GetByID", mock.Anything, "tdv-str").Return(strTDV, nil)
	badTDRepo.On("GetByID", mock.Anything, "td-str").Return(nil, assert.AnError)
	assert.False(t, isEntityTypeIdentical(existingAttrs, existingAssocs, importET, tdvRepo2, badTDRepo, etRepo, context.Background()))

	// ET target lookup error → false
	badETRepo := new(mocks.MockEntityTypeRepo)
	badETRepo.On("GetByID", mock.Anything, "et-target").Return(nil, assert.AnError)
	tdvRepo3 := new(mocks.MockTypeDefinitionVersionRepo)
	tdvRepo3.On("GetByID", mock.Anything, "tdv-str").Return(strTDV, nil)
	tdRepo3 := new(mocks.MockTypeDefinitionRepo)
	tdRepo3.On("GetByID", mock.Anything, "td-str").Return(strTD, nil)
	assert.False(t, isEntityTypeIdentical(existingAttrs, existingAssocs, importET, tdvRepo3, tdRepo3, badETRepo, context.Background()))
}

// T-30.50b: ParseExportInstances — parses instances with containment children
func TestT30_50b_ParseExportInstances(t *testing.T) {
	entityTypes := []ExportEntityType{
		{
			Name: "server",
			Associations: []ExportAssociation{
				{Name: "tools", Type: "containment", Target: "tool"},
			},
		},
		{Name: "tool"},
	}

	raw := []json.RawMessage{
		json.RawMessage(`{
			"entity_type": "server",
			"name": "github",
			"description": "GitHub server",
			"attributes": {"endpoint": "https://example.com"},
			"links": [{"association": "dep", "target_type": "guard", "target_name": "g1"}],
			"tools": [
				{"name": "list-repos", "description": "List repos", "attributes": {}}
			]
		}`),
	}

	result, err := ParseExportInstances(raw, entityTypes)
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "server", result[0].EntityType)
	assert.Equal(t, "github", result[0].Name)
	assert.Equal(t, "GitHub server", result[0].Description)
	assert.Equal(t, "https://example.com", result[0].Attributes["endpoint"])
	require.Len(t, result[0].Links, 1)
	assert.Equal(t, "dep", result[0].Links[0].Association)

	// Children parsed from containment association key "tools"
	require.Len(t, result[0].Children["tools"], 1)
	assert.Equal(t, "list-repos", result[0].Children["tools"][0].Name)
	// Entity type inferred from association target
	assert.Equal(t, "tool", result[0].Children["tools"][0].EntityType)
}

// T-30.50c: ParseExportInstances — invalid JSON returns error
func TestT30_50c_ParseExportInstancesInvalidJSON(t *testing.T) {
	_, err := ParseExportInstances([]json.RawMessage{json.RawMessage(`{invalid}`)}, nil)
	assert.Error(t, err)
}

// T-30.50d: ParseExportInstances — child with entity_type already set keeps it
func TestT30_50d_ParseExportInstancesChildWithEntityType(t *testing.T) {
	entityTypes := []ExportEntityType{
		{
			Name: "server",
			Associations: []ExportAssociation{
				{Name: "tools", Type: "containment", Target: "tool"},
			},
		},
	}

	raw := []json.RawMessage{
		json.RawMessage(`{
			"entity_type": "server",
			"name": "s1",
			"tools": [
				{"entity_type": "custom-tool", "name": "t1"}
			]
		}`),
	}

	result, err := ParseExportInstances(raw, entityTypes)
	require.NoError(t, err)
	// Entity type explicitly set → should NOT be overridden
	assert.Equal(t, "custom-tool", result[0].Children["tools"][0].EntityType)
}

// T-30.51b: Import with instances, attributes, containment, and links
func TestT30_51b_ImportWithInstancesAndLinks(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, instRepo, iavRepo, linkRepo, typePinRepo := newImportService()

	data := minimalExportData()
	data.EntityTypes = []ExportEntityType{
		{
			Name:        "server",
			Description: "A server",
			Attributes: []ExportAttribute{
				{Name: "endpoint", TypeDefinition: "string", Required: true, Ordinal: 0},
			},
			Associations: []ExportAssociation{
				{Name: "tools", Type: "containment", Target: "tool", SourceCardinality: "1", TargetCardinality: "0..n"},
				{Name: "pre-execute", Type: "directional", Target: "guard", SourceCardinality: "0..n", TargetCardinality: "0..n"},
			},
		},
		{Name: "tool", Description: "A tool", Attributes: []ExportAttribute{}, Associations: []ExportAssociation{}},
		{Name: "guard", Description: "A guard", Attributes: []ExportAttribute{}, Associations: []ExportAssociation{}},
	}
	data.Instances = []ExportInstance{
		{
			EntityType:  "server",
			Name:        "github",
			Description: "GitHub server",
			Attributes:  map[string]any{"endpoint": "https://example.com"},
			Links: []ExportLink{
				{Association: "pre-execute", TargetType: "guard", TargetName: "g1"},
			},
			Children: map[string][]*ExportInstance{
				"tools": {
					{EntityType: "tool", Name: "list-repos", Description: "List repos", Attributes: map[string]any{}, Children: make(map[string][]*ExportInstance)},
				},
			},
		},
		{EntityType: "guard", Name: "g1", Attributes: map[string]any{}, Children: make(map[string][]*ExportInstance)},
	}

	catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))

	systemTDs := []*models.TypeDefinition{{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}}
	systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}
	tdRepo.On("List", mock.Anything, mock.Anything).Return(systemTDs, 1, nil)
	tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-str").Return(systemTDV, nil)

	etRepo.On("GetByName", mock.Anything, "server").Return(nil, domainerrors.NewNotFound("EntityType", "server"))
	etRepo.On("GetByName", mock.Anything, "tool").Return(nil, domainerrors.NewNotFound("EntityType", "tool"))
	etRepo.On("GetByName", mock.Anything, "guard").Return(nil, domainerrors.NewNotFound("EntityType", "guard"))
	etRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityType")).Return(nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
	cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
	pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)
	typePinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionTypePin")).Return(nil)
	attrRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Attribute")).Return(nil)
	assocRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Association")).Return(nil)
	assocRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Association{
		{ID: "assoc-created", Name: "pre-execute", Type: models.AssociationTypeDirectional, TargetEntityTypeID: "et-guard"},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Attribute{
		{ID: "attr-created", Name: "endpoint", TypeDefinitionVersionID: "tdv-str"},
	}, nil)
	// ResolveBaseTypes needs tdv/td lookups
	tdvRepo.On("GetByID", mock.Anything, "tdv-str").Return(systemTDV, nil)
	tdRepo.On("GetByID", mock.Anything, "td-str").Return(systemTDs[0], nil)
	catalogRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Catalog")).Return(nil)
	instRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityInstance")).Return(nil)
	iavRepo.On("SetValues", mock.Anything, mock.Anything).Return(nil)
	linkRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.AssociationLink")).Return(nil)

	req := &ImportRequest{Data: data}
	result, err := svc.Import(context.Background(), req)
	require.NoError(t, err)

	assert.Equal(t, "success", result.Status)
	assert.Equal(t, 3, result.TypesCreated)
	assert.Equal(t, 3, result.InstancesCreated)   // server + tool + guard
	assert.Equal(t, 1, result.LinksCreated)         // pre-execute link
}

// T-30.52b: Import — reuse_existing entity type
func TestT30_52b_ImportReuseExisting(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, _, _, _, typePinRepo := newImportService()

	catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))

	systemTDs := []*models.TypeDefinition{{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}}
	systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}
	tdRepo.On("List", mock.Anything, mock.Anything).Return(systemTDs, 1, nil)
	tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-str").Return(systemTDV, nil)

	existingET := &models.EntityType{ID: "et-existing", Name: "server"}
	existingETV := &models.EntityTypeVersion{ID: "etv-existing", EntityTypeID: "et-existing", Version: 1}
	etRepo.On("GetByName", mock.Anything, "server").Return(existingET, nil)
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-existing").Return(existingETV, nil)

	cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
	pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)
	typePinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionTypePin")).Return(nil)
	assocRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Association{}, nil)
	attrRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Attribute{}, nil)
	catalogRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Catalog")).Return(nil)

	req := &ImportRequest{
		ReuseExisting: []string{"server"},
		Data:          minimalExportData(),
	}
	result, err := svc.Import(context.Background(), req)
	require.NoError(t, err)

	assert.Equal(t, "success", result.Status)
	assert.Equal(t, 0, result.TypesCreated)
	assert.Equal(t, 1, result.TypesReused) // reused existing
}

// T-30.53b: Import — identical existing entity type reused automatically
func TestT30_53b_ImportIdenticalETReused(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, _, _, _, typePinRepo := newImportService()

	catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))

	systemTDs := []*models.TypeDefinition{{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}}
	systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}
	tdRepo.On("List", mock.Anything, mock.Anything).Return(systemTDs, 1, nil)
	tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-str").Return(systemTDV, nil)

	// Entity type exists and is identical (0 attrs, 0 assocs)
	existingET := &models.EntityType{ID: "et-existing", Name: "server"}
	existingETV := &models.EntityTypeVersion{ID: "etv-existing", EntityTypeID: "et-existing", Version: 1}
	etRepo.On("GetByName", mock.Anything, "server").Return(existingET, nil)
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-existing").Return(existingETV, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv-existing").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "etv-existing").Return([]*models.Association{}, nil)

	cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
	pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)
	typePinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionTypePin")).Return(nil)
	catalogRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Catalog")).Return(nil)

	req := &ImportRequest{Data: minimalExportData()}
	result, err := svc.Import(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, 0, result.TypesCreated)
	assert.Equal(t, 1, result.TypesReused)
}

// T-30.54: Import — entity type conflict error
func TestT30_54_ImportETConflict(t *testing.T) {
	svc, catalogRepo, cvRepo, _, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, _, _, _, _ := newImportService()

	catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))

	systemTDs := []*models.TypeDefinition{{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}}
	systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}
	tdRepo.On("List", mock.Anything, mock.Anything).Return(systemTDs, 1, nil)
	tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-str").Return(systemTDV, nil)

	// Existing ET with 1 attr — import has 0 attrs → conflict
	existingET := &models.EntityType{ID: "et-1", Name: "server"}
	existingETV := &models.EntityTypeVersion{ID: "etv-1", EntityTypeID: "et-1", Version: 1}
	etRepo.On("GetByName", mock.Anything, "server").Return(existingET, nil)
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-1").Return(existingETV, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv-1").Return([]*models.Attribute{{ID: "a1", Name: "ep", TypeDefinitionVersionID: "tdv-str", Required: true}}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "etv-1").Return([]*models.Association{}, nil)
	tdvRepo.On("GetByID", mock.Anything, "tdv-str").Return(systemTDV, nil)
	tdRepo.On("GetByID", mock.Anything, "td-str").Return(systemTDs[0], nil)
	cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)

	req := &ImportRequest{Data: minimalExportData()}
	_, err := svc.Import(context.Background(), req)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsConflict(err))
}

// T-30.55: Import — type definition conflict error
func TestT30_55_ImportTDConflict(t *testing.T) {
	svc, catalogRepo, cvRepo, _, _, _, _, _, tdRepo, tdvRepo, _, _, _, _ := newImportService()

	data := minimalExportData()
	data.TypeDefinitions = []ExportTypeDef{
		{Name: "hex12", Description: "12-char hex", BaseType: "string", Constraints: map[string]any{"max_length": float64(12)}},
	}

	catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))

	existingTD := &models.TypeDefinition{ID: "td-1", Name: "hex12", Description: "12-char hex", BaseType: models.BaseTypeString}
	existingTDV := &models.TypeDefinitionVersion{ID: "tdv-1", TypeDefinitionID: "td-1", VersionNumber: 1, Constraints: map[string]any{"max_length": float64(24)}} // DIFFERENT
	tdRepo.On("GetByName", mock.Anything, "hex12").Return(existingTD, nil)
	tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-1").Return(existingTDV, nil)

	req := &ImportRequest{Data: data}
	_, err := svc.Import(context.Background(), req)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsConflict(err))
}

// T-30.56: Import — nil data
func TestT30_56_ImportNilData(t *testing.T) {
	svc, _, _, _, _, _, _, _, _, _, _, _, _, _ := newImportService()
	req := &ImportRequest{Data: nil}
	_, err := svc.Import(context.Background(), req)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

// T-30.57: Import — invalid format version
func TestT30_57_ImportInvalidFormatVersion(t *testing.T) {
	svc, _, _, _, _, _, _, _, _, _, _, _, _, _ := newImportService()
	data := minimalExportData()
	data.FormatVersion = "2.0"
	req := &ImportRequest{Data: data}
	_, err := svc.Import(context.Background(), req)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

// T-30.58: Import — catalog_name and cv_label overrides
func TestT30_58_ImportOverrides(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, _, _, _, typePinRepo := newImportService()

	catalogRepo.On("GetByName", mock.Anything, "my-import").Return(nil, domainerrors.NewNotFound("Catalog", "my-import"))
	cvRepo.On("GetByLabel", mock.Anything, "custom-label").Return(nil, domainerrors.NewNotFound("CatalogVersion", "custom-label"))

	systemTDs := []*models.TypeDefinition{{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}}
	systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}
	tdRepo.On("List", mock.Anything, mock.Anything).Return(systemTDs, 1, nil)
	tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-str").Return(systemTDV, nil)
	etRepo.On("GetByName", mock.Anything, "server").Return(nil, domainerrors.NewNotFound("EntityType", "server"))
	etRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityType")).Return(nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
	cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
	pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)
	typePinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionTypePin")).Return(nil)
	assocRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Association{}, nil)
	attrRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Attribute{}, nil)
	catalogRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Catalog")).Return(nil)

	req := &ImportRequest{
		CatalogName:         "my-import",
		CatalogVersionLabel: "custom-label",
		Data:                minimalExportData(),
	}
	result, err := svc.Import(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "my-import", result.CatalogName)
}

// T-30.59: Import — missing type definition for attribute
func TestT30_59_ImportMissingTypeDefForAttr(t *testing.T) {
	svc, catalogRepo, cvRepo, _, etRepo, etvRepo, _, assocRepo, tdRepo, tdvRepo, _, _, _, _ := newImportService()

	data := minimalExportData()
	data.EntityTypes[0].Attributes = []ExportAttribute{
		{Name: "id", TypeDefinition: "nonexistent-type", Required: true, Ordinal: 0},
	}

	catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
	systemTDs := []*models.TypeDefinition{{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}}
	systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}
	tdRepo.On("List", mock.Anything, mock.Anything).Return(systemTDs, 1, nil)
	tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-str").Return(systemTDV, nil)
	etRepo.On("GetByName", mock.Anything, "server").Return(nil, domainerrors.NewNotFound("EntityType", "server"))
	etRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityType")).Return(nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
	cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
	assocRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Association{}, nil)

	req := &ImportRequest{Data: data}
	_, err := svc.Import(context.Background(), req)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

// T-30.60: Import — missing association target entity type
func TestT30_60_ImportMissingAssocTarget(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, _, assocRepo, tdRepo, tdvRepo, _, _, _, typePinRepo := newImportService()

	data := minimalExportData()
	data.EntityTypes = []ExportEntityType{
		{
			Name: "server",
			Associations: []ExportAssociation{
				{Name: "dep", Type: "directional", Target: "unknown-type"},
			},
		},
	}

	catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
	systemTDs := []*models.TypeDefinition{{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}}
	systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}
	tdRepo.On("List", mock.Anything, mock.Anything).Return(systemTDs, 1, nil)
	tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-str").Return(systemTDV, nil)
	etRepo.On("GetByName", mock.Anything, "server").Return(nil, domainerrors.NewNotFound("EntityType", "server"))
	etRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityType")).Return(nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
	cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
	pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)
	typePinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionTypePin")).Return(nil)
	assocRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Association{}, nil)

	req := &ImportRequest{Data: data}
	_, err := svc.Import(context.Background(), req)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

// T-30.61: Import — instance with unknown entity type
func TestT30_61_ImportInstanceUnknownET(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, _, assocRepo, tdRepo, tdvRepo, _, _, _, typePinRepo := newImportService()

	data := minimalExportData()
	data.Instances = []ExportInstance{
		{EntityType: "unknown-type", Name: "inst1", Attributes: map[string]any{}, Children: make(map[string][]*ExportInstance)},
	}

	catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
	systemTDs := []*models.TypeDefinition{{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}}
	systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}
	tdRepo.On("List", mock.Anything, mock.Anything).Return(systemTDs, 1, nil)
	tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-str").Return(systemTDV, nil)
	etRepo.On("GetByName", mock.Anything, "server").Return(nil, domainerrors.NewNotFound("EntityType", "server"))
	etRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityType")).Return(nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
	cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
	pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)
	typePinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionTypePin")).Return(nil)
	assocRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Association{}, nil)
	catalogRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Catalog")).Return(nil)

	req := &ImportRequest{Data: data}
	_, err := svc.Import(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "data import failed")
}

// T-30.62: Import — JSON/list attribute values in instances
func TestT30_62_ImportJSONListAttributes(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, instRepo, iavRepo, _, typePinRepo := newImportService()

	data := minimalExportData()
	data.EntityTypes[0].Attributes = []ExportAttribute{
		{Name: "tags", TypeDefinition: "list", Required: false, Ordinal: 0},
		{Name: "meta", TypeDefinition: "json", Required: false, Ordinal: 1},
		{Name: "raw", TypeDefinition: "list", Required: false, Ordinal: 2},
	}
	data.Instances = []ExportInstance{
		{
			EntityType: "server",
			Name:       "s1",
			Attributes: map[string]any{
				"tags": []any{"a", "b"},          // list value (non-string)
				"meta": map[string]any{"k": "v"}, // JSON value (non-string)
				"raw":  "already-a-string",        // string value for list type
			},
			Children: make(map[string][]*ExportInstance),
		},
	}

	catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))

	listTD := &models.TypeDefinition{ID: "td-list", Name: "list", BaseType: models.BaseTypeList, System: true}
	listTDV := &models.TypeDefinitionVersion{ID: "tdv-list", TypeDefinitionID: "td-list", VersionNumber: 1}
	jsonTD := &models.TypeDefinition{ID: "td-json", Name: "json", BaseType: models.BaseTypeJSON, System: true}
	jsonTDV := &models.TypeDefinitionVersion{ID: "tdv-json", TypeDefinitionID: "td-json", VersionNumber: 1}
	systemTDs := []*models.TypeDefinition{
		{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true},
		listTD,
		jsonTD,
	}
	systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}
	tdRepo.On("List", mock.Anything, mock.Anything).Return(systemTDs, 3, nil)
	tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-str").Return(systemTDV, nil)
	tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-list").Return(listTDV, nil)
	tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-json").Return(jsonTDV, nil)
	tdvRepo.On("GetByID", mock.Anything, "tdv-list").Return(listTDV, nil)
	tdRepo.On("GetByID", mock.Anything, "td-list").Return(listTD, nil)
	tdvRepo.On("GetByID", mock.Anything, "tdv-json").Return(jsonTDV, nil)
	tdRepo.On("GetByID", mock.Anything, "td-json").Return(jsonTD, nil)

	etRepo.On("GetByName", mock.Anything, "server").Return(nil, domainerrors.NewNotFound("EntityType", "server"))
	etRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityType")).Return(nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
	attrRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Attribute")).Return(nil)
	cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
	pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)
	typePinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionTypePin")).Return(nil)
	assocRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Association{}, nil)
	attrRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Attribute{
		{ID: "attr-tags", Name: "tags", TypeDefinitionVersionID: "tdv-list"},
		{ID: "attr-meta", Name: "meta", TypeDefinitionVersionID: "tdv-json"},
		{ID: "attr-raw", Name: "raw", TypeDefinitionVersionID: "tdv-list"},
	}, nil)
	catalogRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Catalog")).Return(nil)
	instRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityInstance")).Return(nil)
	iavRepo.On("SetValues", mock.Anything, mock.Anything).Return(nil)

	req := &ImportRequest{Data: data}
	result, err := svc.Import(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, 1, result.InstancesCreated)
}

// T-30.63: Import — reuse_existing ET not found
func TestT30_63_ImportReuseExistingNotFound(t *testing.T) {
	svc, catalogRepo, cvRepo, _, etRepo, _, _, _, tdRepo, tdvRepo, _, _, _, _ := newImportService()

	catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
	systemTDs := []*models.TypeDefinition{{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}}
	systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}
	tdRepo.On("List", mock.Anything, mock.Anything).Return(systemTDs, 1, nil)
	tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-str").Return(systemTDV, nil)
	etRepo.On("GetByName", mock.Anything, "server").Return(nil, domainerrors.NewNotFound("EntityType", "server"))
	cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)

	req := &ImportRequest{
		ReuseExisting: []string{"server"},
		Data:          minimalExportData(),
	}
	_, err := svc.Import(context.Background(), req)
	assert.Error(t, err)
}

// T-30.64: Import with no transaction manager
func TestT30_64_ImportNoTransactionManager(t *testing.T) {
	catalogRepo := new(mocks.MockCatalogRepo)
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	instRepo := new(mocks.MockEntityInstanceRepo)
	iavRepo := new(mocks.MockInstanceAttributeValueRepo)
	linkRepo := new(mocks.MockAssociationLinkRepo)
	typePinRepo := new(mocks.MockCatalogVersionTypePinRepo)

	// No txManager option → uses direct execution
	svc := NewImportService(catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, instRepo, iavRepo, linkRepo, typePinRepo)

	catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
	systemTDs := []*models.TypeDefinition{{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}}
	systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}
	tdRepo.On("List", mock.Anything, mock.Anything).Return(systemTDs, 1, nil)
	tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-str").Return(systemTDV, nil)
	etRepo.On("GetByName", mock.Anything, "server").Return(nil, domainerrors.NewNotFound("EntityType", "server"))
	etRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityType")).Return(nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
	cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
	pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)
	typePinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionTypePin")).Return(nil)
	assocRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Association{}, nil)
	attrRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Attribute{}, nil)
	catalogRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Catalog")).Return(nil)
	instRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityInstance")).Return(nil)
	iavRepo.On("SetValues", mock.Anything, mock.Anything).Return(nil)

	req := &ImportRequest{Data: minimalExportData()}
	result, err := svc.Import(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "success", result.Status)
}

// T-30.65: ApplyMassRename — various cases
func TestT30_65_ApplyMassRenameCases(t *testing.T) {
	t.Run("no_prefix_suffix", func(t *testing.T) {
		result := ApplyMassRename(nil, nil, nil, "", "")
		assert.Nil(t, result)
	})

	t.Run("suffix_only", func(t *testing.T) {
		entityTypes := []ExportEntityType{{Name: "server"}}
		typeDefs := []ExportTypeDef{{Name: "hex"}}
		result := ApplyMassRename(nil, entityTypes, typeDefs, "", "-v2")
		assert.Equal(t, "server-v2", result.EntityTypes["server"])
		assert.Equal(t, "hex-v2", result.TypeDefinitions["hex"])
	})

	t.Run("existing_rename_not_overridden", func(t *testing.T) {
		existing := &ImportRenameMap{
			EntityTypes:     map[string]string{"server": "custom"},
			TypeDefinitions: map[string]string{"hex": "custom-hex"},
		}
		entityTypes := []ExportEntityType{{Name: "server"}}
		typeDefs := []ExportTypeDef{{Name: "hex"}}
		result := ApplyMassRename(existing, entityTypes, typeDefs, "pre-", "")
		assert.Equal(t, "custom", result.EntityTypes["server"])     // Not overridden
		assert.Equal(t, "custom-hex", result.TypeDefinitions["hex"]) // Not overridden
	})
}

// T-30.66: Import — reuse_existing with etvRepo error
func TestT30_66_ImportReuseExistingETVError(t *testing.T) {
	svc, catalogRepo, cvRepo, _, etRepo, etvRepo, _, _, tdRepo, tdvRepo, _, _, _, _ := newImportService()

	catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
	systemTDs := []*models.TypeDefinition{{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}}
	systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}
	tdRepo.On("List", mock.Anything, mock.Anything).Return(systemTDs, 1, nil)
	tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-str").Return(systemTDV, nil)
	existingET := &models.EntityType{ID: "et-1", Name: "server"}
	etRepo.On("GetByName", mock.Anything, "server").Return(existingET, nil)
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-1").Return(nil, assert.AnError)
	cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)

	req := &ImportRequest{
		ReuseExisting: []string{"server"},
		Data:          minimalExportData(),
	}
	_, err := svc.Import(context.Background(), req)
	assert.Error(t, err)
}

// T-30.68: Import error paths in schema transaction
func TestT30_68_ImportSchemaErrorPaths(t *testing.T) {
	t.Run("td_repo_get_error", func(t *testing.T) {
		svc, catalogRepo, cvRepo, _, _, _, _, _, tdRepo, tdvRepo, _, _, _, _ := newImportService()
		data := minimalExportData()
		data.TypeDefinitions = []ExportTypeDef{{Name: "hex", BaseType: "string"}}
		catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
		cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
		tdRepo.On("GetByName", mock.Anything, "hex").Return(nil, assert.AnError) // non-notfound
		tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, mock.Anything).Return(nil, nil)
		_, err := svc.Import(context.Background(), &ImportRequest{Data: data})
		assert.Error(t, err)
	})

	t.Run("td_create_error", func(t *testing.T) {
		svc, catalogRepo, cvRepo, _, _, _, _, _, tdRepo, _, _, _, _, _ := newImportService()
		data := minimalExportData()
		data.TypeDefinitions = []ExportTypeDef{{Name: "hex", BaseType: "string"}}
		catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
		cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
		tdRepo.On("GetByName", mock.Anything, "hex").Return(nil, domainerrors.NewNotFound("TypeDefinition", "hex"))
		tdRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.TypeDefinition")).Return(assert.AnError)
		_, err := svc.Import(context.Background(), &ImportRequest{Data: data})
		assert.Error(t, err)
	})

	t.Run("tdv_create_error", func(t *testing.T) {
		svc, catalogRepo, cvRepo, _, _, _, _, _, tdRepo, tdvRepo, _, _, _, _ := newImportService()
		data := minimalExportData()
		data.TypeDefinitions = []ExportTypeDef{{Name: "hex", BaseType: "string"}}
		catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
		cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
		tdRepo.On("GetByName", mock.Anything, "hex").Return(nil, domainerrors.NewNotFound("TypeDefinition", "hex"))
		tdRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.TypeDefinition")).Return(nil)
		tdvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.TypeDefinitionVersion")).Return(assert.AnError)
		_, err := svc.Import(context.Background(), &ImportRequest{Data: data})
		assert.Error(t, err)
	})

	t.Run("td_getlatestversion_error", func(t *testing.T) {
		svc, catalogRepo, cvRepo, _, _, _, _, _, tdRepo, tdvRepo, _, _, _, _ := newImportService()
		data := minimalExportData()
		data.TypeDefinitions = []ExportTypeDef{{Name: "hex", BaseType: "string"}}
		catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
		cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
		existingTD := &models.TypeDefinition{ID: "td-1", Name: "hex", BaseType: models.BaseTypeString}
		tdRepo.On("GetByName", mock.Anything, "hex").Return(existingTD, nil)
		tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-1").Return(nil, assert.AnError)
		_, err := svc.Import(context.Background(), &ImportRequest{Data: data})
		assert.Error(t, err)
	})

	t.Run("system_types_list_error", func(t *testing.T) {
		svc, catalogRepo, cvRepo, _, _, _, _, _, tdRepo, _, _, _, _, _ := newImportService()
		catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
		cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
		tdRepo.On("List", mock.Anything, mock.Anything).Return([]*models.TypeDefinition(nil), 0, assert.AnError)
		_, err := svc.Import(context.Background(), &ImportRequest{Data: minimalExportData()})
		assert.Error(t, err)
	})

	t.Run("cv_create_error", func(t *testing.T) {
		svc, catalogRepo, cvRepo, _, _, _, _, _, tdRepo, tdvRepo, _, _, _, _ := newImportService()
		catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
		cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
		systemTDs := []*models.TypeDefinition{{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}}
		systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}
		tdRepo.On("List", mock.Anything, mock.Anything).Return(systemTDs, 1, nil)
		tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-str").Return(systemTDV, nil)
		cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(assert.AnError)
		_, err := svc.Import(context.Background(), &ImportRequest{Data: minimalExportData()})
		assert.Error(t, err)
	})

	t.Run("et_create_error", func(t *testing.T) {
		svc, catalogRepo, cvRepo, _, etRepo, _, _, _, tdRepo, tdvRepo, _, _, _, _ := newImportService()
		catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
		cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
		systemTDs := []*models.TypeDefinition{{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}}
		systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}
		tdRepo.On("List", mock.Anything, mock.Anything).Return(systemTDs, 1, nil)
		tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-str").Return(systemTDV, nil)
		cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
		etRepo.On("GetByName", mock.Anything, "server").Return(nil, domainerrors.NewNotFound("EntityType", "server"))
		etRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityType")).Return(assert.AnError)
		_, err := svc.Import(context.Background(), &ImportRequest{Data: minimalExportData()})
		assert.Error(t, err)
	})

	t.Run("etv_create_error", func(t *testing.T) {
		svc, catalogRepo, cvRepo, _, etRepo, etvRepo, _, _, tdRepo, tdvRepo, _, _, _, _ := newImportService()
		catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
		cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
		systemTDs := []*models.TypeDefinition{{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}}
		systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}
		tdRepo.On("List", mock.Anything, mock.Anything).Return(systemTDs, 1, nil)
		tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-str").Return(systemTDV, nil)
		cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
		etRepo.On("GetByName", mock.Anything, "server").Return(nil, domainerrors.NewNotFound("EntityType", "server"))
		etRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityType")).Return(nil)
		etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(assert.AnError)
		_, err := svc.Import(context.Background(), &ImportRequest{Data: minimalExportData()})
		assert.Error(t, err)
	})

	t.Run("attr_create_error", func(t *testing.T) {
		svc, catalogRepo, cvRepo, _, etRepo, etvRepo, attrRepo, _, tdRepo, tdvRepo, _, _, _, _ := newImportService()
		data := minimalExportData()
		data.EntityTypes[0].Attributes = []ExportAttribute{{Name: "ep", TypeDefinition: "string", Ordinal: 0}}
		catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
		cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
		systemTDs := []*models.TypeDefinition{{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}}
		systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}
		tdRepo.On("List", mock.Anything, mock.Anything).Return(systemTDs, 1, nil)
		tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-str").Return(systemTDV, nil)
		cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
		etRepo.On("GetByName", mock.Anything, "server").Return(nil, domainerrors.NewNotFound("EntityType", "server"))
		etRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityType")).Return(nil)
		etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
		attrRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Attribute")).Return(assert.AnError)
		_, err := svc.Import(context.Background(), &ImportRequest{Data: data})
		assert.Error(t, err)
	})

	t.Run("pin_create_error", func(t *testing.T) {
		svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, _, _, tdRepo, tdvRepo, _, _, _, _ := newImportService()
		catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
		cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
		systemTDs := []*models.TypeDefinition{{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}}
		systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}
		tdRepo.On("List", mock.Anything, mock.Anything).Return(systemTDs, 1, nil)
		tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-str").Return(systemTDV, nil)
		cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
		etRepo.On("GetByName", mock.Anything, "server").Return(nil, domainerrors.NewNotFound("EntityType", "server"))
		etRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityType")).Return(nil)
		etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
		pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(assert.AnError)
		_, err := svc.Import(context.Background(), &ImportRequest{Data: minimalExportData()})
		assert.Error(t, err)
	})

	t.Run("assoc_create_error", func(t *testing.T) {
		svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, _, assocRepo, tdRepo, tdvRepo, _, _, _, typePinRepo := newImportService()
		data := minimalExportData()
		data.EntityTypes[0].Associations = []ExportAssociation{{Name: "dep", Type: "directional", Target: "server"}}
		catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
		cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
		systemTDs := []*models.TypeDefinition{{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}}
		systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}
		tdRepo.On("List", mock.Anything, mock.Anything).Return(systemTDs, 1, nil)
		tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-str").Return(systemTDV, nil)
		cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
		etRepo.On("GetByName", mock.Anything, "server").Return(nil, domainerrors.NewNotFound("EntityType", "server"))
		etRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityType")).Return(nil)
		etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
		pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)
		typePinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionTypePin")).Return(nil)
		assocRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Association{}, nil)
		assocRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Association")).Return(assert.AnError)
		_, err := svc.Import(context.Background(), &ImportRequest{Data: data})
		assert.Error(t, err)
	})

	t.Run("type_pin_create_error", func(t *testing.T) {
		svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, _, assocRepo, tdRepo, tdvRepo, _, _, _, typePinRepo := newImportService()
		catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
		cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
		systemTDs := []*models.TypeDefinition{{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}}
		systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}
		tdRepo.On("List", mock.Anything, mock.Anything).Return(systemTDs, 1, nil)
		tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-str").Return(systemTDV, nil)
		cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
		etRepo.On("GetByName", mock.Anything, "server").Return(nil, domainerrors.NewNotFound("EntityType", "server"))
		etRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityType")).Return(nil)
		etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
		pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)
		assocRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Association{}, nil)
		typePinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionTypePin")).Return(assert.AnError)
		_, err := svc.Import(context.Background(), &ImportRequest{Data: minimalExportData()})
		assert.Error(t, err)
	})

	t.Run("catalog_create_error", func(t *testing.T) {
		svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, _, assocRepo, tdRepo, tdvRepo, _, _, _, typePinRepo := newImportService()
		catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
		cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
		systemTDs := []*models.TypeDefinition{{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}}
		systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}
		tdRepo.On("List", mock.Anything, mock.Anything).Return(systemTDs, 1, nil)
		tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-str").Return(systemTDV, nil)
		cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
		etRepo.On("GetByName", mock.Anything, "server").Return(nil, domainerrors.NewNotFound("EntityType", "server"))
		etRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityType")).Return(nil)
		etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
		pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)
		typePinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionTypePin")).Return(nil)
		assocRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Association{}, nil)
		catalogRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Catalog")).Return(assert.AnError)
		_, err := svc.Import(context.Background(), &ImportRequest{Data: minimalExportData()})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "data import failed")
	})

	t.Run("instance_create_error", func(t *testing.T) {
		svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, instRepo, _, _, typePinRepo := newImportService()
		data := minimalExportData()
		data.Instances = []ExportInstance{
			{EntityType: "server", Name: "s1", Attributes: map[string]any{}, Children: make(map[string][]*ExportInstance)},
		}
		catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
		cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
		systemTDs := []*models.TypeDefinition{{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}}
		systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}
		tdRepo.On("List", mock.Anything, mock.Anything).Return(systemTDs, 1, nil)
		tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-str").Return(systemTDV, nil)
		cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
		etRepo.On("GetByName", mock.Anything, "server").Return(nil, domainerrors.NewNotFound("EntityType", "server"))
		etRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityType")).Return(nil)
		etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
		pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)
		typePinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionTypePin")).Return(nil)
		assocRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Association{}, nil)
		attrRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Attribute{}, nil)
		catalogRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Catalog")).Return(nil)
		instRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityInstance")).Return(assert.AnError)
		_, err := svc.Import(context.Background(), &ImportRequest{Data: data})
		assert.Error(t, err)
	})

	t.Run("reuse_existing_pin_create_error", func(t *testing.T) {
		svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, _, _, tdRepo, tdvRepo, _, _, _, _ := newImportService()
		catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
		cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
		systemTDs := []*models.TypeDefinition{{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}}
		systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}
		tdRepo.On("List", mock.Anything, mock.Anything).Return(systemTDs, 1, nil)
		tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-str").Return(systemTDV, nil)
		cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
		etRepo.On("GetByName", mock.Anything, "server").Return(&models.EntityType{ID: "et-1", Name: "server"}, nil)
		etvRepo.On("GetLatestByEntityType", mock.Anything, "et-1").Return(&models.EntityTypeVersion{ID: "etv-1", EntityTypeID: "et-1", Version: 1}, nil)
		pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(assert.AnError)
		_, err := svc.Import(context.Background(), &ImportRequest{ReuseExisting: []string{"server"}, Data: minimalExportData()})
		assert.Error(t, err)
	})

	t.Run("identical_et_etv_lookup_error", func(t *testing.T) {
		svc, catalogRepo, cvRepo, _, etRepo, etvRepo, _, _, tdRepo, tdvRepo, _, _, _, _ := newImportService()
		catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
		cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
		systemTDs := []*models.TypeDefinition{{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}}
		systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}
		tdRepo.On("List", mock.Anything, mock.Anything).Return(systemTDs, 1, nil)
		tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-str").Return(systemTDV, nil)
		cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
		etRepo.On("GetByName", mock.Anything, "server").Return(&models.EntityType{ID: "et-1", Name: "server"}, nil)
		etvRepo.On("GetLatestByEntityType", mock.Anything, "et-1").Return(nil, assert.AnError)
		_, err := svc.Import(context.Background(), &ImportRequest{Data: minimalExportData()})
		assert.Error(t, err)
	})

	t.Run("identical_et_attr_list_error", func(t *testing.T) {
		svc, catalogRepo, cvRepo, _, etRepo, etvRepo, attrRepo, _, tdRepo, tdvRepo, _, _, _, _ := newImportService()
		catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
		cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
		systemTDs := []*models.TypeDefinition{{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}}
		systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}
		tdRepo.On("List", mock.Anything, mock.Anything).Return(systemTDs, 1, nil)
		tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-str").Return(systemTDV, nil)
		cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
		etRepo.On("GetByName", mock.Anything, "server").Return(&models.EntityType{ID: "et-1", Name: "server"}, nil)
		etvRepo.On("GetLatestByEntityType", mock.Anything, "et-1").Return(&models.EntityTypeVersion{ID: "etv-1", EntityTypeID: "et-1", Version: 1}, nil)
		attrRepo.On("ListByVersion", mock.Anything, "etv-1").Return(nil, assert.AnError)
		_, err := svc.Import(context.Background(), &ImportRequest{Data: minimalExportData()})
		assert.Error(t, err)
	})

	t.Run("identical_et_assoc_list_error", func(t *testing.T) {
		svc, catalogRepo, cvRepo, _, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, _, _, _, _ := newImportService()
		catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
		cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
		systemTDs := []*models.TypeDefinition{{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}}
		systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}
		tdRepo.On("List", mock.Anything, mock.Anything).Return(systemTDs, 1, nil)
		tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-str").Return(systemTDV, nil)
		cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
		etRepo.On("GetByName", mock.Anything, "server").Return(&models.EntityType{ID: "et-1", Name: "server"}, nil)
		etvRepo.On("GetLatestByEntityType", mock.Anything, "et-1").Return(&models.EntityTypeVersion{ID: "etv-1", EntityTypeID: "et-1", Version: 1}, nil)
		attrRepo.On("ListByVersion", mock.Anything, "etv-1").Return([]*models.Attribute{}, nil)
		assocRepo.On("ListByVersion", mock.Anything, "etv-1").Return(nil, assert.AnError)
		_, err := svc.Import(context.Background(), &ImportRequest{Data: minimalExportData()})
		assert.Error(t, err)
	})

	t.Run("identical_et_pin_create_error", func(t *testing.T) {
		svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, _, _, _, _ := newImportService()
		catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
		cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
		systemTDs := []*models.TypeDefinition{{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}}
		systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}
		tdRepo.On("List", mock.Anything, mock.Anything).Return(systemTDs, 1, nil)
		tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-str").Return(systemTDV, nil)
		cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
		etRepo.On("GetByName", mock.Anything, "server").Return(&models.EntityType{ID: "et-1", Name: "server"}, nil)
		etvRepo.On("GetLatestByEntityType", mock.Anything, "et-1").Return(&models.EntityTypeVersion{ID: "etv-1", EntityTypeID: "et-1", Version: 1}, nil)
		attrRepo.On("ListByVersion", mock.Anything, "etv-1").Return([]*models.Attribute{}, nil)
		assocRepo.On("ListByVersion", mock.Anything, "etv-1").Return([]*models.Association{}, nil)
		pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(assert.AnError)
		_, err := svc.Import(context.Background(), &ImportRequest{Data: minimalExportData()})
		assert.Error(t, err)
	})

	t.Run("et_getbyname_non_notfound_error", func(t *testing.T) {
		svc, catalogRepo, cvRepo, _, etRepo, _, _, _, tdRepo, tdvRepo, _, _, _, _ := newImportService()
		catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
		cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
		systemTDs := []*models.TypeDefinition{{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}}
		systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}
		tdRepo.On("List", mock.Anything, mock.Anything).Return(systemTDs, 1, nil)
		tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-str").Return(systemTDV, nil)
		cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
		etRepo.On("GetByName", mock.Anything, "server").Return(nil, assert.AnError) // non-notfound
		_, err := svc.Import(context.Background(), &ImportRequest{Data: minimalExportData()})
		assert.Error(t, err)
	})

	// assoc_second_pass_et_getbyname_error and assoc_etv_lookup_error_in_second_pass
	// were removed — the association loop no longer re-queries the DB to check if
	// entity types were reused. It uses the createdETNames set instead.
}

// T-30.69: createLinksRecursive error paths
func TestT30_69_CreateLinksRecursiveErrors(t *testing.T) {
	t.Run("assoc_list_error", func(t *testing.T) {
		svc, _, _, _, _, _, _, assocRepo, _, _, _, _, _, _ := newImportService()
		inst := ExportInstance{
			EntityType: "server",
			Name:       "s1",
			Links:      []ExportLink{{Association: "dep", TargetType: "guard", TargetName: "g1"}},
			Children:   make(map[string][]*ExportInstance),
		}
		etNameToVersionID := map[string]string{"server": "etv-1"}
		instanceNameToID := map[string]string{"server/s1": "inst-1"}
		assocRepo.On("ListByVersion", mock.Anything, "etv-1").Return(nil, assert.AnError)
		_, err := svc.createLinksRecursive(context.Background(), inst, etNameToVersionID, instanceNameToID, nil)
		assert.Error(t, err)
	})

	t.Run("link_create_error", func(t *testing.T) {
		svc, _, _, _, _, _, _, assocRepo, _, _, _, _, linkRepo, _ := newImportService()
		inst := ExportInstance{
			EntityType: "server",
			Name:       "s1",
			Links:      []ExportLink{{Association: "dep", TargetType: "guard", TargetName: "g1"}},
			Children:   make(map[string][]*ExportInstance),
		}
		etNameToVersionID := map[string]string{"server": "etv-1"}
		instanceNameToID := map[string]string{"server/s1": "inst-1", "guard/g1": "inst-2"}
		assocRepo.On("ListByVersion", mock.Anything, "etv-1").Return([]*models.Association{
			{ID: "assoc-1", Name: "dep", Type: models.AssociationTypeDirectional},
		}, nil)
		linkRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.AssociationLink")).Return(assert.AnError)
		_, err := svc.createLinksRecursive(context.Background(), inst, etNameToVersionID, instanceNameToID, nil)
		assert.Error(t, err)
	})

	t.Run("child_link_error", func(t *testing.T) {
		svc, _, _, _, _, _, _, assocRepo, _, _, _, _, _, _ := newImportService()
		child := ExportInstance{
			EntityType: "tool",
			Name:       "t1",
			Links:      []ExportLink{{Association: "dep", TargetType: "guard", TargetName: "g1"}},
			Children:   make(map[string][]*ExportInstance),
		}
		inst := ExportInstance{
			EntityType: "server",
			Name:       "s1",
			Children: map[string][]*ExportInstance{
				"tools": {&child},
			},
		}
		etNameToVersionID := map[string]string{"server": "etv-1", "tool": "etv-2"}
		instanceNameToID := map[string]string{"server/s1": "inst-1", "tool/t1": "inst-2"}
		assocRepo.On("ListByVersion", mock.Anything, "etv-2").Return(nil, assert.AnError)
		_, err := svc.createLinksRecursive(context.Background(), inst, etNameToVersionID, instanceNameToID, nil)
		assert.Error(t, err)
	})

	t.Run("missing_assoc_and_target_skipped", func(t *testing.T) {
		svc, _, _, _, _, _, _, assocRepo, _, _, _, _, _, _ := newImportService()
		inst := ExportInstance{
			EntityType: "server",
			Name:       "s1",
			Links: []ExportLink{
				{Association: "unknown-assoc", TargetType: "guard", TargetName: "g1"},
				{Association: "dep", TargetType: "guard", TargetName: "missing-target"},
			},
			Children: make(map[string][]*ExportInstance),
		}
		etNameToVersionID := map[string]string{"server": "etv-1"}
		instanceNameToID := map[string]string{"server/s1": "inst-1"}
		assocRepo.On("ListByVersion", mock.Anything, "etv-1").Return([]*models.Association{
			{ID: "assoc-1", Name: "dep", Type: models.AssociationTypeDirectional},
		}, nil)
		n, err := svc.createLinksRecursive(context.Background(), inst, etNameToVersionID, instanceNameToID, nil)
		require.NoError(t, err)
		assert.Equal(t, 0, n, "both links should be skipped: unknown assoc + missing target")
	})
}

// T-30.70: ParseExportInstances — recursive child parse error
func TestT30_70_ParseExportInstancesChildParseError(t *testing.T) {
	entityTypes := []ExportEntityType{
		{Name: "server", Associations: []ExportAssociation{{Name: "tools", Type: "containment", Target: "tool"}}},
	}
	raw := []json.RawMessage{
		json.RawMessage(`{
			"entity_type": "server",
			"name": "s1",
			"tools": [{"invalid json that will fail to unmarshal nested"}]
		}`),
	}
	_, err := ParseExportInstances(raw, entityTypes)
	assert.Error(t, err)
}

// T-30.71: Import — instance attr list error in data transaction
func TestT30_71_ImportInstanceAttrListError(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, instRepo, _, _, typePinRepo := newImportService()

	data := minimalExportData()
	data.Instances = []ExportInstance{
		{EntityType: "server", Name: "s1", Attributes: map[string]any{"endpoint": "https://example.com"}, Children: make(map[string][]*ExportInstance)},
	}

	catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
	systemTDs := []*models.TypeDefinition{{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}}
	systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}
	tdRepo.On("List", mock.Anything, mock.Anything).Return(systemTDs, 1, nil)
	tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-str").Return(systemTDV, nil)
	cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
	etRepo.On("GetByName", mock.Anything, "server").Return(nil, domainerrors.NewNotFound("EntityType", "server"))
	etRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityType")).Return(nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
	pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)
	typePinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionTypePin")).Return(nil)
	// Schema phase: assoc list succeed
	assocRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Association{}, nil)
	// attrRepo.ListByVersion always fails — data phase will hit it for instance attrs
	attrRepo.On("ListByVersion", mock.Anything, mock.Anything).Return(nil, assert.AnError)
	catalogRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Catalog")).Return(nil)
	instRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityInstance")).Return(nil)

	_, err := svc.Import(context.Background(), &ImportRequest{Data: data})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "data import failed")
}

// T-30.72: Import — iavRepo.SetValues error in data transaction
func TestT30_72_ImportIAVSetValuesError(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, instRepo, iavRepo, _, typePinRepo := newImportService()

	data := minimalExportData()
	data.EntityTypes[0].Attributes = []ExportAttribute{
		{Name: "endpoint", TypeDefinition: "string", Required: true, Ordinal: 0},
	}
	data.Instances = []ExportInstance{
		{EntityType: "server", Name: "s1", Attributes: map[string]any{"endpoint": "https://example.com"}, Children: make(map[string][]*ExportInstance)},
	}

	catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
	systemTDs := []*models.TypeDefinition{{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}}
	systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}
	tdRepo.On("List", mock.Anything, mock.Anything).Return(systemTDs, 1, nil)
	tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-str").Return(systemTDV, nil)
	cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
	etRepo.On("GetByName", mock.Anything, "server").Return(nil, domainerrors.NewNotFound("EntityType", "server"))
	etRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityType")).Return(nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
	attrRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Attribute")).Return(nil)
	pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)
	typePinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionTypePin")).Return(nil)
	assocRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Association{}, nil)
	attrRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Attribute{
		{ID: "attr-1", Name: "endpoint", TypeDefinitionVersionID: "tdv-str"},
	}, nil)
	tdvRepo.On("GetByID", mock.Anything, "tdv-str").Return(systemTDV, nil)
	tdRepo.On("GetByID", mock.Anything, "td-str").Return(systemTDs[0], nil)
	catalogRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Catalog")).Return(nil)
	instRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityInstance")).Return(nil)
	iavRepo.On("SetValues", mock.Anything, mock.Anything).Return(assert.AnError)

	_, err := svc.Import(context.Background(), &ImportRequest{Data: data})
	assert.Error(t, err)
}

// T-30.73: Import — child instance create error
func TestT30_73_ImportChildCreateError(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, instRepo, _, _, typePinRepo := newImportService()

	data := minimalExportData()
	data.Instances = []ExportInstance{
		{
			EntityType: "server",
			Name:       "s1",
			Attributes: map[string]any{},
			Children: map[string][]*ExportInstance{
				"tools": {{EntityType: "server", Name: "child1", Attributes: map[string]any{}, Children: make(map[string][]*ExportInstance)}},
			},
		},
	}

	catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
	systemTDs := []*models.TypeDefinition{{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}}
	systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}
	tdRepo.On("List", mock.Anything, mock.Anything).Return(systemTDs, 1, nil)
	tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-str").Return(systemTDV, nil)
	cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
	etRepo.On("GetByName", mock.Anything, "server").Return(nil, domainerrors.NewNotFound("EntityType", "server"))
	etRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityType")).Return(nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
	pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)
	typePinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionTypePin")).Return(nil)
	assocRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Association{}, nil)
	attrRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Attribute{}, nil)
	catalogRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Catalog")).Return(nil)
	// Parent instance succeeds, child fails
	instRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityInstance")).Return(nil).Once()
	instRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityInstance")).Return(assert.AnError).Once()

	_, err := svc.Import(context.Background(), &ImportRequest{Data: data})
	assert.Error(t, err)
}

// T-30.74: Import — link create error in data transaction
func TestT30_74_ImportLinkCreateError(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, instRepo, _, linkRepo, typePinRepo := newImportService()

	data := minimalExportData()
	data.EntityTypes[0].Associations = []ExportAssociation{
		{Name: "dep", Type: "directional", Target: "server", SourceCardinality: "0..n", TargetCardinality: "0..n"},
	}
	data.Instances = []ExportInstance{
		{EntityType: "server", Name: "s1", Attributes: map[string]any{}, Links: []ExportLink{
			{Association: "dep", TargetType: "server", TargetName: "s2"},
		}, Children: make(map[string][]*ExportInstance)},
		{EntityType: "server", Name: "s2", Attributes: map[string]any{}, Children: make(map[string][]*ExportInstance)},
	}

	catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
	systemTDs := []*models.TypeDefinition{{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}}
	systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}
	tdRepo.On("List", mock.Anything, mock.Anything).Return(systemTDs, 1, nil)
	tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-str").Return(systemTDV, nil)
	cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
	etRepo.On("GetByName", mock.Anything, "server").Return(nil, domainerrors.NewNotFound("EntityType", "server"))
	etRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityType")).Return(nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
	pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)
	typePinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionTypePin")).Return(nil)
	assocRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Association")).Return(nil)
	assocRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Association{
		{ID: "assoc-1", Name: "dep", Type: models.AssociationTypeDirectional},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Attribute{}, nil)
	catalogRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Catalog")).Return(nil)
	instRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityInstance")).Return(nil)
	linkRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.AssociationLink")).Return(assert.AnError)

	_, err := svc.Import(context.Background(), &ImportRequest{Data: data})
	assert.Error(t, err)
}

// T-30.75: DryRun — new type definition (not found → newCount)
func TestT30_75_DryRunNewTypeDef(t *testing.T) {
	svc, catalogRepo, cvRepo, _, etRepo, _, _, _, tdRepo, _, _, _, _, _ := newImportService()

	data := minimalExportData()
	data.TypeDefinitions = []ExportTypeDef{{Name: "custom-type", BaseType: "string"}}

	catalogRepo.On("GetByName", ctx(), "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", ctx(), "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
	etRepo.On("GetByName", ctx(), "server").Return(nil, domainerrors.NewNotFound("EntityType", "server"))
	tdRepo.On("GetByName", ctx(), "custom-type").Return(nil, domainerrors.NewNotFound("TypeDefinition", "custom-type"))

	result, err := svc.DryRun(context.Background(), &ImportRequest{Data: data})
	require.NoError(t, err)
	assert.Equal(t, "ready", result.Status)
	assert.Equal(t, 4, result.Summary.New) // catalog + CV + ET + TD
}

// T-30.76: Import — system type GetLatestByTypeDefinition error → skipped
func TestT30_76_ImportSystemTypeLatestVersionError(t *testing.T) {
	svc, catalogRepo, cvRepo, _, _, _, _, _, tdRepo, tdvRepo, _, _, _, _ := newImportService()

	catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
	// System type whose latest version lookup fails → should return error (not silently skip)
	systemTDs := []*models.TypeDefinition{
		{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true},
		{ID: "td-bad", Name: "bad-system", BaseType: models.BaseTypeString, System: true},
	}
	systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}
	tdRepo.On("List", mock.Anything, mock.Anything).Return(systemTDs, 2, nil)
	tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-str").Return(systemTDV, nil)
	tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-bad").Return(nil, assert.AnError)

	req := &ImportRequest{Data: minimalExportData()}
	_, err := svc.Import(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bad-system")
}

// T-30.77: Import — schema error via no-txManager path
func TestT30_77_ImportSchemaErrorNoTxManager(t *testing.T) {
	catalogRepo := new(mocks.MockCatalogRepo)
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	instRepo := new(mocks.MockEntityInstanceRepo)
	iavRepo := new(mocks.MockInstanceAttributeValueRepo)
	linkRepo := new(mocks.MockAssociationLinkRepo)
	typePinRepo := new(mocks.MockCatalogVersionTypePinRepo)

	svc := NewImportService(catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, instRepo, iavRepo, linkRepo, typePinRepo)

	catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
	tdRepo.On("List", mock.Anything, mock.Anything).Return([]*models.TypeDefinition(nil), 0, assert.AnError)

	_, err := svc.Import(context.Background(), &ImportRequest{Data: minimalExportData()})
	assert.Error(t, err)
}

// T-30.78: Import — data error via no-txManager path
func TestT30_78_ImportDataErrorNoTxManager(t *testing.T) {
	catalogRepo := new(mocks.MockCatalogRepo)
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	instRepo := new(mocks.MockEntityInstanceRepo)
	iavRepo := new(mocks.MockInstanceAttributeValueRepo)
	linkRepo := new(mocks.MockAssociationLinkRepo)
	typePinRepo := new(mocks.MockCatalogVersionTypePinRepo)

	svc := NewImportService(catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, instRepo, iavRepo, linkRepo, typePinRepo)

	catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
	systemTDs := []*models.TypeDefinition{{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}}
	systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}
	tdRepo.On("List", mock.Anything, mock.Anything).Return(systemTDs, 1, nil)
	tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-str").Return(systemTDV, nil)
	cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
	etRepo.On("GetByName", mock.Anything, "server").Return(nil, domainerrors.NewNotFound("EntityType", "server"))
	etRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityType")).Return(nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
	pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)
	typePinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionTypePin")).Return(nil)
	assocRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Association{}, nil)
	catalogRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Catalog")).Return(assert.AnError) // catalog create fails in data tx

	_, err := svc.Import(context.Background(), &ImportRequest{Data: minimalExportData()})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "data import failed")
}

// T-30.79: Import — instance with unknown attr name → skipped
func TestT30_79_ImportInstanceUnknownAttrSkipped(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, instRepo, iavRepo, _, typePinRepo := newImportService()

	data := minimalExportData()
	data.EntityTypes[0].Attributes = []ExportAttribute{
		{Name: "endpoint", TypeDefinition: "string", Required: true, Ordinal: 0},
	}
	data.Instances = []ExportInstance{
		{EntityType: "server", Name: "s1", Attributes: map[string]any{
			"endpoint":    "https://example.com",
			"nonexistent": "should be skipped", // Unknown attr
		}, Children: make(map[string][]*ExportInstance)},
	}

	catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
	systemTDs := []*models.TypeDefinition{{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}}
	systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}
	tdRepo.On("List", mock.Anything, mock.Anything).Return(systemTDs, 1, nil)
	tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-str").Return(systemTDV, nil)
	cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
	etRepo.On("GetByName", mock.Anything, "server").Return(nil, domainerrors.NewNotFound("EntityType", "server"))
	etRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityType")).Return(nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
	attrRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Attribute")).Return(nil)
	pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)
	typePinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionTypePin")).Return(nil)
	assocRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Association{}, nil)
	attrRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Attribute{
		{ID: "attr-1", Name: "endpoint", TypeDefinitionVersionID: "tdv-str"},
	}, nil)
	tdvRepo.On("GetByID", mock.Anything, "tdv-str").Return(systemTDV, nil)
	tdRepo.On("GetByID", mock.Anything, "td-str").Return(systemTDs[0], nil)
	catalogRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Catalog")).Return(nil)
	instRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityInstance")).Return(nil)
	iavRepo.On("SetValues", mock.Anything, mock.Anything).Return(nil)

	result, err := svc.Import(context.Background(), &ImportRequest{Data: data})
	require.NoError(t, err)
	assert.Equal(t, 1, result.InstancesCreated)
}

// T-30.80: Import — ResolveBaseTypes error during instance attribute resolution
func TestT30_80_ImportResolveBaseTypesError(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, instRepo, _, _, typePinRepo := newImportService()

	data := minimalExportData()
	data.EntityTypes[0].Attributes = []ExportAttribute{
		{Name: "endpoint", TypeDefinition: "string", Required: true, Ordinal: 0},
	}
	data.Instances = []ExportInstance{
		{EntityType: "server", Name: "s1", Attributes: map[string]any{"endpoint": "value"}, Children: make(map[string][]*ExportInstance)},
	}

	catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
	systemTDs := []*models.TypeDefinition{{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}}
	systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}
	tdRepo.On("List", mock.Anything, mock.Anything).Return(systemTDs, 1, nil)
	tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-str").Return(systemTDV, nil)
	cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
	etRepo.On("GetByName", mock.Anything, "server").Return(nil, domainerrors.NewNotFound("EntityType", "server"))
	etRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityType")).Return(nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
	attrRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Attribute")).Return(nil)
	pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)
	typePinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionTypePin")).Return(nil)
	assocRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Association{}, nil)
	attrRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Attribute{
		{ID: "attr-1", Name: "endpoint", TypeDefinitionVersionID: "tdv-str"},
	}, nil)
	// ResolveBaseTypes will call tdvRepo.GetByID → error
	tdvRepo.On("GetByID", mock.Anything, "tdv-str").Return(nil, assert.AnError)
	catalogRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Catalog")).Return(nil)
	instRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityInstance")).Return(nil)

	_, err := svc.Import(context.Background(), &ImportRequest{Data: data})
	assert.Error(t, err)
}

// T-30.81: ParseExportInstances — invalid child JSON array → continue
func TestT30_81_ParseExportInstancesInvalidChildArray(t *testing.T) {
	entityTypes := []ExportEntityType{
		{Name: "server", Associations: []ExportAssociation{{Name: "tools", Type: "containment", Target: "tool"}}},
	}
	raw := []json.RawMessage{
		json.RawMessage(`{
			"entity_type": "server",
			"name": "s1",
			"tools": "not-an-array"
		}`),
	}
	result, err := ParseExportInstances(raw, entityTypes)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Empty(t, result[0].Children, "invalid child array should be skipped via continue")
}

// T-30.82: ParseExportInstances — recursive child parse error (valid array with invalid child object)
func TestT30_82_ParseExportInstancesRecursiveChildError(t *testing.T) {
	entityTypes := []ExportEntityType{
		{Name: "server", Associations: []ExportAssociation{{Name: "tools", Type: "containment", Target: "tool"}}},
	}
	// tools is a valid JSON array, but each element is not a valid JSON object
	raw := []json.RawMessage{
		json.RawMessage(`{
			"entity_type": "server",
			"name": "s1",
			"tools": [123]
		}`),
	}
	_, err := ParseExportInstances(raw, entityTypes)
	assert.Error(t, err) // 123 is not an object → json.Unmarshal to map[string]json.RawMessage fails
}

// T-30.91: Import returns error when reused entity type's assoc listing fails
func TestT30_91_ImportReusedETAssocListError(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, _, assocRepo, tdRepo, _, _, _, _, typePinRepo := newImportService()

	data := minimalExportData()
	existingET := &models.EntityType{ID: "et-1", Name: "server"}
	existingETV := &models.EntityTypeVersion{ID: "etv-1", EntityTypeID: "et-1", Version: 1}

	catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
	tdRepo.On("List", mock.Anything, mock.Anything).Return([]*models.TypeDefinition{}, 0, nil)
	cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
	// Entity type exists and is in reuse_existing
	etRepo.On("GetByName", mock.Anything, "server").Return(existingET, nil)
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-1").Return(existingETV, nil)
	pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)
	typePinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionTypePin")).Return(nil)
	// ListByVersion for reused ET fails
	assocRepo.On("ListByVersion", mock.Anything, "etv-1").Return(nil, assert.AnError)

	req := &ImportRequest{Data: data, ReuseExisting: []string{"server"}}
	_, err := svc.Import(context.Background(), req)
	assert.Error(t, err, "should return error when reused ET assoc listing fails")
}

// T-30.89: Import with duplicate instance names creates both
func TestT30_89_ImportDuplicateInstanceNames(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, _, assocRepo, tdRepo, _, instRepo, _, _, typePinRepo := newImportService()

	data := minimalExportData()
	data.Instances = []ExportInstance{
		{EntityType: "server", Name: "dup-name", Attributes: map[string]any{}, Children: make(map[string][]*ExportInstance)},
		{EntityType: "server", Name: "dup-name", Attributes: map[string]any{}, Children: make(map[string][]*ExportInstance)},
	}

	catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
	tdRepo.On("List", mock.Anything, mock.Anything).Return([]*models.TypeDefinition{}, 0, nil)
	etRepo.On("GetByName", mock.Anything, mock.Anything).Return(nil, domainerrors.NewNotFound("EntityType", ""))
	etRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityType")).Return(nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
	cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
	pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)
	typePinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionTypePin")).Return(nil)
	assocRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Association")).Return(nil)
	assocRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Association{}, nil)
	catalogRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Catalog")).Return(nil)
	instRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityInstance")).Return(nil)

	req := &ImportRequest{Data: data}
	result, err := svc.Import(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, 2, result.InstancesCreated, "duplicate names should both be created")
}

// T-30.90: Import with missing association target entity type
func TestT30_90_ImportMissingAssocTarget(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, _, _, tdRepo, _, _, _, _, typePinRepo := newImportService()

	data := minimalExportData()
	data.EntityTypes = []ExportEntityType{
		{
			Name: "server",
			Associations: []ExportAssociation{
				{Name: "uses", Type: "directional", Target: "nonexistent-type"},
			},
		},
	}

	catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
	tdRepo.On("List", mock.Anything, mock.Anything).Return([]*models.TypeDefinition{}, 0, nil)
	etRepo.On("GetByName", mock.Anything, mock.Anything).Return(nil, domainerrors.NewNotFound("EntityType", ""))
	etRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityType")).Return(nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
	cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
	pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)
	typePinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionTypePin")).Return(nil)

	req := &ImportRequest{Data: data}
	_, err := svc.Import(context.Background(), req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent-type")
}

// T-30.88: Import sets ValueNumber for integer/number attributes
func TestT30_88_ImportSetsValueNumberForIntegerAttributes(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, instRepo, iavRepo, _, typePinRepo := newImportService()

	data := minimalExportData()
	data.EntityTypes = []ExportEntityType{
		{
			Name: "model",
			Attributes: []ExportAttribute{
				{Name: "context-window", TypeDefinition: "integer", Required: false, Ordinal: 0},
			},
		},
	}
	data.Instances = []ExportInstance{
		{EntityType: "model", Name: "gpt-4", Attributes: map[string]any{"context-window": float64(128000)}, Children: make(map[string][]*ExportInstance)},
	}

	catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
	systemTDs := []*models.TypeDefinition{
		{ID: "td-int", Name: "integer", BaseType: models.BaseTypeInteger, System: true},
	}
	systemTDV := &models.TypeDefinitionVersion{ID: "tdv-int", TypeDefinitionID: "td-int", VersionNumber: 1}
	tdRepo.On("List", mock.Anything, mock.Anything).Return(systemTDs, 1, nil)
	tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-int").Return(systemTDV, nil)
	etRepo.On("GetByName", mock.Anything, mock.Anything).Return(nil, domainerrors.NewNotFound("EntityType", ""))
	etRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityType")).Return(nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
	cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
	pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)
	typePinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionTypePin")).Return(nil)
	attrRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Attribute")).Return(nil)
	assocRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Association")).Return(nil)
	attrRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Attribute{
		{ID: "attr-cw", Name: "context-window", TypeDefinitionVersionID: "tdv-int"},
	}, nil)
	assocRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Association{}, nil)
	tdvRepo.On("GetByID", mock.Anything, "tdv-int").Return(systemTDV, nil)
	tdRepo.On("GetByID", mock.Anything, "td-int").Return(systemTDs[0], nil)
	catalogRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Catalog")).Return(nil)
	instRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityInstance")).Return(nil)
	iavRepo.On("SetValues", mock.Anything, mock.Anything).Return(nil)

	req := &ImportRequest{Data: data}
	result, err := svc.Import(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "success", result.Status)

	// Verify SetValues was called with ValueNumber set
	setValuesCall := iavRepo.Calls[0]
	values := setValuesCall.Arguments.Get(1).([]*models.InstanceAttributeValue)
	require.Len(t, values, 1)
	assert.Equal(t, "attr-cw", values[0].AttributeID)
	require.NotNil(t, values[0].ValueNumber, "ValueNumber must be set for integer attributes")
	assert.Equal(t, float64(128000), *values[0].ValueNumber)
	assert.Equal(t, "128000", values[0].ValueString)
}

// T-30.86: Import creates links for bidirectional associations from reverse side
func TestT30_86_ImportBidirectionalLinkFromReverseSide(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, _, instRepo, _, linkRepo, typePinRepo := newImportService()

	data := minimalExportData()
	data.EntityTypes = []ExportEntityType{
		{
			Name: "server",
			Associations: []ExportAssociation{
				{Name: "guardrails", Type: "bidirectional", Target: "guard", SourceCardinality: "0..n", TargetCardinality: "0..n"},
			},
		},
		{Name: "guard"},
	}
	// The guard instance has the link (reverse side — association defined on server, not guard)
	data.Instances = []ExportInstance{
		{EntityType: "server", Name: "s1", Attributes: map[string]any{}, Children: make(map[string][]*ExportInstance)},
		{EntityType: "guard", Name: "g1", Attributes: map[string]any{}, Children: make(map[string][]*ExportInstance),
			Links: []ExportLink{
				{Association: "guardrails", TargetType: "server", TargetName: "s1"},
			},
		},
	}

	catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
	tdRepo.On("List", mock.Anything, mock.Anything).Return([]*models.TypeDefinition{}, 0, nil)
	etRepo.On("GetByName", mock.Anything, mock.Anything).Return(nil, domainerrors.NewNotFound("EntityType", ""))
	etRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityType")).Return(nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
	cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
	pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)
	typePinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionTypePin")).Return(nil)
	assocRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Association")).Return(nil)
	attrRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Attribute{}, nil)
	// Guard's ETV has no associations — ListByVersion returns empty.
	// The "guardrails" association is on server's ETV, not guard's.
	assocRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Association{}, nil)
	catalogRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Catalog")).Return(nil)
	instRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityInstance")).Return(nil)
	linkRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.AssociationLink")).Return(nil)

	req := &ImportRequest{Data: data}
	result, err := svc.Import(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, 1, result.LinksCreated, "bidirectional link from reverse side should be created")
}

// T-30.87: Import rejects data missing required catalog export fields
func TestT30_87_ImportRejectsMissingCatalogFields(t *testing.T) {
	t.Run("missing catalog name", func(t *testing.T) {
		svc, _, _, _, _, _, _, _, _, _, _, _, _, _ := newImportService()
		data := &ExportData{FormatVersion: "1.0", Catalog: ExportCatalog{}, CatalogVersion: ExportCatalogVersion{Label: "v1"}}
		_, err := svc.Import(context.Background(), &ImportRequest{Data: data})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "catalog name")
	})

	t.Run("missing catalog version label", func(t *testing.T) {
		svc, _, _, _, _, _, _, _, _, _, _, _, _, _ := newImportService()
		data := &ExportData{FormatVersion: "1.0", Catalog: ExportCatalog{Name: "test"}, CatalogVersion: ExportCatalogVersion{}}
		_, err := svc.Import(context.Background(), &ImportRequest{Data: data})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "catalog version label")
	})

	t.Run("DryRun missing catalog name", func(t *testing.T) {
		svc, _, _, _, _, _, _, _, _, _, _, _, _, _ := newImportService()
		data := &ExportData{FormatVersion: "1.0", Catalog: ExportCatalog{}, CatalogVersion: ExportCatalogVersion{Label: "v1"}}
		_, err := svc.DryRun(context.Background(), &ImportRequest{Data: data})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "catalog name")
	})

	t.Run("DryRun missing CV label", func(t *testing.T) {
		svc, _, _, _, _, _, _, _, _, _, _, _, _, _ := newImportService()
		data := &ExportData{FormatVersion: "1.0", Catalog: ExportCatalog{Name: "test"}, CatalogVersion: ExportCatalogVersion{}}
		_, err := svc.DryRun(context.Background(), &ImportRequest{Data: data})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "catalog version label")
	})
}

// T-30.85: Import creates associations for newly created entity types
func TestT30_85_ImportCreatesAssociationsForNewEntityTypes(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, _, instRepo, _, _, typePinRepo := newImportService()

	data := minimalExportData()
	data.EntityTypes = []ExportEntityType{
		{
			Name: "server",
			Associations: []ExportAssociation{
				{Name: "tools", Type: "containment", Target: "tool", SourceCardinality: "1", TargetCardinality: "0..n"},
				{Name: "pre-execute", Type: "directional", Target: "guard", SourceCardinality: "0..n", TargetCardinality: "0..n"},
			},
		},
		{Name: "tool"},
		{Name: "guard"},
	}
	data.Instances = nil

	catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
	tdRepo.On("List", mock.Anything, mock.Anything).Return([]*models.TypeDefinition{}, 0, nil)
	etRepo.On("GetByName", mock.Anything, mock.Anything).Return(nil, domainerrors.NewNotFound("EntityType", ""))
	etRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityType")).Return(nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
	cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
	pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)
	typePinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionTypePin")).Return(nil)
	assocRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Association")).Return(nil)
	attrRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Association{}, nil)
	catalogRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Catalog")).Return(nil)
	instRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityInstance")).Return(nil)

	req := &ImportRequest{Data: data}
	result, err := svc.Import(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "success", result.Status)

	// Verify associations were created — both containment AND directional
	var assocNames []string
	for _, call := range assocRepo.Calls {
		if call.Method == "Create" {
			a := call.Arguments.Get(1).(*models.Association)
			assocNames = append(assocNames, a.Name)
		}
	}
	assert.Contains(t, assocNames, "tools", "containment association should be created")
	assert.Contains(t, assocNames, "pre-execute", "directional association should be created")
	assert.Len(t, assocNames, 2, "should create exactly 2 associations")
}

// T-30.83: ExportData JSON round-trip preserves contained children
func TestT30_83_ExportDataJSONRoundTripPreservesChildren(t *testing.T) {
	data := ExportData{
		FormatVersion: "1.0",
		Catalog:       ExportCatalog{Name: "test", Description: "test"},
		CatalogVersion: ExportCatalogVersion{Label: "v1", Description: "v1"},
		EntityTypes: []ExportEntityType{
			{
				Name: "server",
				Associations: []ExportAssociation{
					{Name: "tools", Type: "containment", Target: "tool"},
				},
			},
			{Name: "tool"},
		},
		Instances: []ExportInstance{
			{
				EntityType: "server",
				Name:       "github",
				Attributes: map[string]any{},
				Children: map[string][]*ExportInstance{
					"tools": {
						{EntityType: "tool", Name: "list-repos", Attributes: map[string]any{}, Children: make(map[string][]*ExportInstance)},
						{EntityType: "tool", Name: "create-pr", Attributes: map[string]any{}, Children: make(map[string][]*ExportInstance)},
					},
				},
			},
		},
	}

	// Marshal to JSON (uses custom MarshalJSON — includes children)
	b, err := json.Marshal(data)
	require.NoError(t, err)

	// Unmarshal back (this is the code path used by the import handler)
	var parsed ExportData
	err = json.Unmarshal(b, &parsed)
	require.NoError(t, err)

	require.Len(t, parsed.Instances, 1)
	require.Len(t, parsed.Instances[0].Children["tools"], 2, "children must survive JSON round-trip")
	assert.Equal(t, "list-repos", parsed.Instances[0].Children["tools"][0].Name)
	assert.Equal(t, "create-pr", parsed.Instances[0].Children["tools"][1].Name)
}

// T-30.84: Import via JSON — contained children are created
func TestT30_84_ImportViaJSONCreatesContainedChildren(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, _, instRepo, iavRepo, _, typePinRepo := newImportService()

	data := minimalExportData()
	data.EntityTypes = []ExportEntityType{
		{
			Name: "server",
			Associations: []ExportAssociation{
				{Name: "tools", Type: "containment", Target: "tool", SourceCardinality: "1", TargetCardinality: "0..n"},
			},
		},
		{Name: "tool"},
	}
	data.Instances = []ExportInstance{
		{
			EntityType: "server",
			Name:       "github",
			Attributes: map[string]any{},
			Children: map[string][]*ExportInstance{
				"tools": {
					{EntityType: "tool", Name: "list-repos", Attributes: map[string]any{}, Children: make(map[string][]*ExportInstance)},
					{EntityType: "tool", Name: "create-pr", Attributes: map[string]any{}, Children: make(map[string][]*ExportInstance)},
				},
			},
		},
	}

	// Marshal to JSON and unmarshal back — simulates the handler's c.Bind(&req) path
	jsonBytes, err := json.Marshal(data)
	require.NoError(t, err)
	var parsed ExportData
	require.NoError(t, json.Unmarshal(jsonBytes, &parsed))

	// Set up mocks
	catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))
	tdRepo.On("List", mock.Anything, mock.Anything).Return([]*models.TypeDefinition{}, 0, nil)
	etRepo.On("GetByName", mock.Anything, "server").Return(nil, domainerrors.NewNotFound("EntityType", "server"))
	etRepo.On("GetByName", mock.Anything, "tool").Return(nil, domainerrors.NewNotFound("EntityType", "tool"))
	etRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityType")).Return(nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
	cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
	pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)
	typePinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionTypePin")).Return(nil)
	assocRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Association")).Return(nil)
	attrRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Association{}, nil)
	catalogRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Catalog")).Return(nil)
	instRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityInstance")).Return(nil)
	iavRepo.On("SetValues", mock.Anything, mock.Anything).Return(nil)

	req := &ImportRequest{Data: &parsed}
	result, err := svc.Import(context.Background(), req)
	require.NoError(t, err)

	assert.Equal(t, "success", result.Status)
	assert.Equal(t, 3, result.InstancesCreated, "should create server + 2 contained tools")

	// Verify parent-child relationship: 3rd instance create call should have ParentInstanceID set
	createCalls := instRepo.Calls
	var childCalls int
	for _, call := range createCalls {
		if call.Method == "Create" {
			inst := call.Arguments.Get(1).(*models.EntityInstance)
			if inst.ParentInstanceID != "" {
				childCalls++
			}
		}
	}
	assert.Equal(t, 2, childCalls, "2 child instances should have ParentInstanceID set")
}

// T-30.67: Import — identical existing TD reused
func TestT30_67_ImportIdenticalTDReused(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, _, _, _, typePinRepo := newImportService()

	data := minimalExportData()
	data.TypeDefinitions = []ExportTypeDef{
		{Name: "hex12", Description: "12-char hex", BaseType: "string", Constraints: map[string]any{"max_length": float64(12)}},
	}
	data.EntityTypes[0].Attributes = []ExportAttribute{
		{Name: "id", TypeDefinition: "hex12", Required: true, Ordinal: 0},
	}

	catalogRepo.On("GetByName", mock.Anything, "test-catalog").Return(nil, domainerrors.NewNotFound("Catalog", "test-catalog"))
	cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v1.0"))

	// Identical existing TD
	existingTD := &models.TypeDefinition{ID: "td-existing", Name: "hex12", Description: "12-char hex", BaseType: models.BaseTypeString}
	existingTDV := &models.TypeDefinitionVersion{ID: "tdv-existing", TypeDefinitionID: "td-existing", VersionNumber: 1, Constraints: map[string]any{"max_length": float64(12)}}
	tdRepo.On("GetByName", mock.Anything, "hex12").Return(existingTD, nil)
	tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-existing").Return(existingTDV, nil)

	systemTDs := []*models.TypeDefinition{{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}}
	systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}
	tdRepo.On("List", mock.Anything, mock.Anything).Return(systemTDs, 1, nil)
	tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-str").Return(systemTDV, nil)

	etRepo.On("GetByName", mock.Anything, "server").Return(nil, domainerrors.NewNotFound("EntityType", "server"))
	etRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityType")).Return(nil)
	etvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EntityTypeVersion")).Return(nil)
	attrRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Attribute")).Return(nil)
	cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
	pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)
	typePinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionTypePin")).Return(nil)
	assocRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Association{}, nil)
	attrRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Attribute{}, nil)
	catalogRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Catalog")).Return(nil)

	req := &ImportRequest{Data: data}
	result, err := svc.Import(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, 1, result.TypesReused) // hex12 reused
	assert.Equal(t, 1, result.TypesCreated) // entity type created
}
