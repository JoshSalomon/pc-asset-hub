package operational

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository/mocks"
)

func newExportService() (*ExportService, *mocks.MockCatalogRepo, *mocks.MockCatalogVersionRepo, *mocks.MockCatalogVersionPinRepo, *mocks.MockEntityTypeRepo, *mocks.MockEntityTypeVersionRepo, *mocks.MockAttributeRepo, *mocks.MockAssociationRepo, *mocks.MockTypeDefinitionRepo, *mocks.MockTypeDefinitionVersionRepo, *mocks.MockEntityInstanceRepo, *mocks.MockInstanceAttributeValueRepo, *mocks.MockAssociationLinkRepo) {
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

	svc := NewExportService(catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, instRepo, iavRepo, linkRepo)
	return svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, instRepo, iavRepo, linkRepo
}

// T-30.01: Export minimal catalog — one entity type, no instances
func TestT30_01_ExportMinimalCatalog(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, _, _, instRepo, _, _ := newExportService()

	now := time.Now()
	catalog := &models.Catalog{ID: "cat-1", Name: "test-catalog", Description: "Test", CatalogVersionID: "cv-1", ValidationStatus: models.ValidationStatusValid, CreatedAt: now, UpdatedAt: now}
	cv := &models.CatalogVersion{ID: "cv-1", VersionLabel: "v1.0", Description: "First version"}
	et := &models.EntityType{ID: "et-1", Name: "mcp-server"}
	etv := &models.EntityTypeVersion{ID: "etv-1", EntityTypeID: "et-1", Version: 1, Description: "An MCP server"}
	pin := &models.CatalogVersionPin{ID: "pin-1", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-1"}

	catalogRepo.On("GetByName", ctx(), "test-catalog").Return(catalog, nil)
	cvRepo.On("GetByID", ctx(), "cv-1").Return(cv, nil)
	pinRepo.On("ListByCatalogVersion", ctx(), "cv-1").Return([]*models.CatalogVersionPin{pin}, nil)
	etvRepo.On("GetByID", ctx(), "etv-1").Return(etv, nil)
	etRepo.On("GetByID", ctx(), "et-1").Return(et, nil)
	attrRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Association{}, nil)
	instRepo.On("ListByCatalog", ctx(), "cat-1").Return([]*models.EntityInstance{}, nil)

	result, err := svc.ExportCatalog(context.Background(), "test-catalog", nil, "")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "1.0", result.FormatVersion)
	assert.Equal(t, "test-catalog", result.Catalog.Name)
	assert.Equal(t, "Test", result.Catalog.Description)
	assert.Equal(t, "valid", result.Catalog.ValidationStatus)
	assert.Equal(t, "v1.0", result.CatalogVersion.Label)
	assert.Equal(t, "First version", result.CatalogVersion.Description)
	assert.Empty(t, result.TypeDefinitions)
	require.Len(t, result.EntityTypes, 1)
	assert.Equal(t, "mcp-server", result.EntityTypes[0].Name)
	assert.Equal(t, "An MCP server", result.EntityTypes[0].Description)
	assert.Empty(t, result.Instances)
}

// T-30.02: Export with custom type definitions and attributes
func TestT30_02_ExportWithTypeDefs(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, instRepo, _, _ := newExportService()

	now := time.Now()
	catalog := &models.Catalog{ID: "cat-1", Name: "test-catalog", Description: "Test", CatalogVersionID: "cv-1", ValidationStatus: models.ValidationStatusValid, CreatedAt: now, UpdatedAt: now}
	cv := &models.CatalogVersion{ID: "cv-1", VersionLabel: "v1.0", Description: "First"}
	et := &models.EntityType{ID: "et-1", Name: "mcp-server"}
	etv := &models.EntityTypeVersion{ID: "etv-1", EntityTypeID: "et-1", Version: 1, Description: "server"}
	pin := &models.CatalogVersionPin{ID: "pin-1", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-1"}

	// System type (url) — should not be in type_definitions
	systemTD := &models.TypeDefinition{ID: "td-url", Name: "url", BaseType: models.BaseTypeURL, System: true}
	systemTDV := &models.TypeDefinitionVersion{ID: "tdv-url", TypeDefinitionID: "td-url", VersionNumber: 1}
	// Custom type (hex12) — should appear in type_definitions
	customTD := &models.TypeDefinition{ID: "td-hex", Name: "hex12", Description: "12-char hex", BaseType: models.BaseTypeString, System: false}
	customTDV := &models.TypeDefinitionVersion{ID: "tdv-hex", TypeDefinitionID: "td-hex", VersionNumber: 1, Constraints: map[string]any{"max_length": float64(12), "pattern": "[0-9A-F]*"}}

	attrs := []*models.Attribute{
		{ID: "attr-1", EntityTypeVersionID: "etv-1", Name: "endpoint", TypeDefinitionVersionID: "tdv-url", Required: true, Ordinal: 0, Description: "Server URL"},
		{ID: "attr-2", EntityTypeVersionID: "etv-1", Name: "id", TypeDefinitionVersionID: "tdv-hex", Required: true, Ordinal: 1, Description: ""},
	}

	catalogRepo.On("GetByName", ctx(), "test-catalog").Return(catalog, nil)
	cvRepo.On("GetByID", ctx(), "cv-1").Return(cv, nil)
	pinRepo.On("ListByCatalogVersion", ctx(), "cv-1").Return([]*models.CatalogVersionPin{pin}, nil)
	etvRepo.On("GetByID", ctx(), "etv-1").Return(etv, nil)
	etRepo.On("GetByID", ctx(), "et-1").Return(et, nil)
	attrRepo.On("ListByVersion", ctx(), "etv-1").Return(attrs, nil)
	tdvRepo.On("GetByID", ctx(), "tdv-url").Return(systemTDV, nil)
	tdRepo.On("GetByID", ctx(), "td-url").Return(systemTD, nil)
	tdvRepo.On("GetByID", ctx(), "tdv-hex").Return(customTDV, nil)
	tdRepo.On("GetByID", ctx(), "td-hex").Return(customTD, nil)
	assocRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Association{}, nil)
	instRepo.On("ListByCatalog", ctx(), "cat-1").Return([]*models.EntityInstance{}, nil)

	result, err := svc.ExportCatalog(context.Background(), "test-catalog", nil, "")
	require.NoError(t, err)

	// Only custom type definitions should be exported
	require.Len(t, result.TypeDefinitions, 1)
	assert.Equal(t, "hex12", result.TypeDefinitions[0].Name)
	assert.Equal(t, "string", result.TypeDefinitions[0].BaseType)
	assert.False(t, result.TypeDefinitions[0].System)
	assert.Equal(t, float64(12), result.TypeDefinitions[0].Constraints["max_length"])
	assert.Equal(t, "[0-9A-F]*", result.TypeDefinitions[0].Constraints["pattern"])

	// Attributes should reference type definition names
	require.Len(t, result.EntityTypes[0].Attributes, 2)
	assert.Equal(t, "endpoint", result.EntityTypes[0].Attributes[0].Name)
	assert.Equal(t, "url", result.EntityTypes[0].Attributes[0].TypeDefinition)
	assert.True(t, result.EntityTypes[0].Attributes[0].Required)
	assert.Equal(t, "id", result.EntityTypes[0].Attributes[1].Name)
	assert.Equal(t, "hex12", result.EntityTypes[0].Attributes[1].TypeDefinition)
}

// T-30.03: Export with instances and attribute values
func TestT30_03_ExportWithInstances(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, instRepo, iavRepo, linkRepo := newExportService()

	now := time.Now()
	catalog := &models.Catalog{ID: "cat-1", Name: "test-catalog", CatalogVersionID: "cv-1", ValidationStatus: models.ValidationStatusValid, CreatedAt: now, UpdatedAt: now}
	cv := &models.CatalogVersion{ID: "cv-1", VersionLabel: "v1.0"}
	et := &models.EntityType{ID: "et-1", Name: "guardrail"}
	etv := &models.EntityTypeVersion{ID: "etv-1", EntityTypeID: "et-1", Version: 1, Description: "Guard"}
	pin := &models.CatalogVersionPin{ID: "pin-1", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-1"}
	systemTD := &models.TypeDefinition{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}
	systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}

	attrs := []*models.Attribute{
		{ID: "attr-1", EntityTypeVersionID: "etv-1", Name: "label", TypeDefinitionVersionID: "tdv-str", Required: true, Ordinal: 0},
	}

	inst := &models.EntityInstance{ID: "inst-1", EntityTypeID: "et-1", CatalogID: "cat-1", Name: "pii-filter", Description: "Filters PII", Version: 1, CreatedAt: now, UpdatedAt: now}
	iav := &models.InstanceAttributeValue{ID: "iav-1", InstanceID: "inst-1", InstanceVersion: 1, AttributeID: "attr-1", ValueString: "PII Filter"}

	catalogRepo.On("GetByName", ctx(), "test-catalog").Return(catalog, nil)
	cvRepo.On("GetByID", ctx(), "cv-1").Return(cv, nil)
	pinRepo.On("ListByCatalogVersion", ctx(), "cv-1").Return([]*models.CatalogVersionPin{pin}, nil)
	etvRepo.On("GetByID", ctx(), "etv-1").Return(etv, nil)
	etRepo.On("GetByID", ctx(), "et-1").Return(et, nil)
	attrRepo.On("ListByVersion", ctx(), "etv-1").Return(attrs, nil)
	tdvRepo.On("GetByID", ctx(), "tdv-str").Return(systemTDV, nil)
	tdRepo.On("GetByID", ctx(), "td-str").Return(systemTD, nil)
	assocRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Association{}, nil)
	instRepo.On("ListByCatalog", ctx(), "cat-1").Return([]*models.EntityInstance{inst}, nil)
	iavRepo.On("GetValuesForVersion", ctx(), "inst-1", 1).Return([]*models.InstanceAttributeValue{iav}, nil)
	linkRepo.On("GetForwardRefs", ctx(), "inst-1").Return([]*models.AssociationLink{}, nil)

	result, err := svc.ExportCatalog(context.Background(), "test-catalog", nil, "")
	require.NoError(t, err)

	require.Len(t, result.Instances, 1)
	assert.Equal(t, "guardrail", result.Instances[0].EntityType)
	assert.Equal(t, "pii-filter", result.Instances[0].Name)
	assert.Equal(t, "Filters PII", result.Instances[0].Description)
	assert.Equal(t, "PII Filter", result.Instances[0].Attributes["label"])
}

// T-30.05: Export with containment — child instances nested under association name
func TestT30_05_ExportWithContainment(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, instRepo, iavRepo, linkRepo := newExportService()

	now := time.Now()
	catalog := &models.Catalog{ID: "cat-1", Name: "test-catalog", CatalogVersionID: "cv-1", ValidationStatus: models.ValidationStatusValid, CreatedAt: now, UpdatedAt: now}
	cv := &models.CatalogVersion{ID: "cv-1", VersionLabel: "v1.0"}

	parentET := &models.EntityType{ID: "et-parent", Name: "mcp-server"}
	parentETV := &models.EntityTypeVersion{ID: "etv-parent", EntityTypeID: "et-parent", Version: 1, Description: "server"}
	childET := &models.EntityType{ID: "et-child", Name: "mcp-tool"}
	childETV := &models.EntityTypeVersion{ID: "etv-child", EntityTypeID: "et-child", Version: 1, Description: "tool"}

	parentPin := &models.CatalogVersionPin{ID: "pin-1", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-parent"}
	childPin := &models.CatalogVersionPin{ID: "pin-2", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-child"}

	containmentAssoc := &models.Association{ID: "assoc-1", EntityTypeVersionID: "etv-parent", Name: "tools", TargetEntityTypeID: "et-child", Type: models.AssociationTypeContainment, SourceCardinality: "1", TargetCardinality: "0..n"}

	parentInst := &models.EntityInstance{ID: "inst-parent", EntityTypeID: "et-parent", CatalogID: "cat-1", Name: "github", Version: 1, CreatedAt: now, UpdatedAt: now}
	childInst := &models.EntityInstance{ID: "inst-child", EntityTypeID: "et-child", CatalogID: "cat-1", ParentInstanceID: "inst-parent", Name: "list-repos", Version: 1, CreatedAt: now, UpdatedAt: now}

	// System type
	systemTD := &models.TypeDefinition{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}
	systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}

	catalogRepo.On("GetByName", ctx(), "test-catalog").Return(catalog, nil)
	cvRepo.On("GetByID", ctx(), "cv-1").Return(cv, nil)
	pinRepo.On("ListByCatalogVersion", ctx(), "cv-1").Return([]*models.CatalogVersionPin{parentPin, childPin}, nil)
	etvRepo.On("GetByID", ctx(), "etv-parent").Return(parentETV, nil)
	etvRepo.On("GetByID", ctx(), "etv-child").Return(childETV, nil)
	etRepo.On("GetByID", ctx(), "et-parent").Return(parentET, nil)
	etRepo.On("GetByID", ctx(), "et-child").Return(childET, nil)

	// Attributes — parent has one attr
	parentAttr := &models.Attribute{ID: "attr-1", EntityTypeVersionID: "etv-parent", Name: "endpoint", TypeDefinitionVersionID: "tdv-str", Required: true, Ordinal: 0}
	attrRepo.On("ListByVersion", ctx(), "etv-parent").Return([]*models.Attribute{parentAttr}, nil)
	attrRepo.On("ListByVersion", ctx(), "etv-child").Return([]*models.Attribute{}, nil)
	tdvRepo.On("GetByID", ctx(), "tdv-str").Return(systemTDV, nil)
	tdRepo.On("GetByID", ctx(), "td-str").Return(systemTD, nil)

	assocRepo.On("ListByVersion", ctx(), "etv-parent").Return([]*models.Association{containmentAssoc}, nil)
	assocRepo.On("ListByVersion", ctx(), "etv-child").Return([]*models.Association{}, nil)

	instRepo.On("ListByCatalog", ctx(), "cat-1").Return([]*models.EntityInstance{parentInst, childInst}, nil)

	parentIAV := &models.InstanceAttributeValue{ID: "iav-1", InstanceID: "inst-parent", InstanceVersion: 1, AttributeID: "attr-1", ValueString: "https://github.example.com"}
	iavRepo.On("GetValuesForVersion", ctx(), "inst-parent", 1).Return([]*models.InstanceAttributeValue{parentIAV}, nil)
	iavRepo.On("GetValuesForVersion", ctx(), "inst-child", 1).Return([]*models.InstanceAttributeValue{}, nil)

	linkRepo.On("GetForwardRefs", ctx(), "inst-parent").Return([]*models.AssociationLink{}, nil)
	linkRepo.On("GetForwardRefs", ctx(), "inst-child").Return([]*models.AssociationLink{}, nil)

	result, err := svc.ExportCatalog(context.Background(), "test-catalog", nil, "")
	require.NoError(t, err)

	// Only root-level instances in the top-level array
	require.Len(t, result.Instances, 1)
	assert.Equal(t, "mcp-server", result.Instances[0].EntityType)
	assert.Equal(t, "github", result.Instances[0].Name)

	// Child nested under association name "tools"
	require.Len(t, result.Instances[0].Children["tools"], 1)
	assert.Equal(t, "list-repos", result.Instances[0].Children["tools"][0].Name)
}

// T-30.06: Export with association links
func TestT30_06_ExportWithLinks(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, instRepo, iavRepo, linkRepo := newExportService()

	now := time.Now()
	catalog := &models.Catalog{ID: "cat-1", Name: "test-catalog", CatalogVersionID: "cv-1", ValidationStatus: models.ValidationStatusValid, CreatedAt: now, UpdatedAt: now}
	cv := &models.CatalogVersion{ID: "cv-1", VersionLabel: "v1.0"}

	serverET := &models.EntityType{ID: "et-server", Name: "mcp-server"}
	serverETV := &models.EntityTypeVersion{ID: "etv-server", EntityTypeID: "et-server", Version: 1, Description: "server"}
	guardET := &models.EntityType{ID: "et-guard", Name: "guardrail"}
	guardETV := &models.EntityTypeVersion{ID: "etv-guard", EntityTypeID: "et-guard", Version: 1, Description: "guard"}

	serverPin := &models.CatalogVersionPin{ID: "pin-1", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-server"}
	guardPin := &models.CatalogVersionPin{ID: "pin-2", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-guard"}

	dirAssoc := &models.Association{ID: "assoc-dir", EntityTypeVersionID: "etv-server", Name: "pre-execute", TargetEntityTypeID: "et-guard", Type: models.AssociationTypeDirectional, SourceCardinality: "0..n", TargetCardinality: "0..n"}

	serverInst := &models.EntityInstance{ID: "inst-server", EntityTypeID: "et-server", CatalogID: "cat-1", Name: "github", Version: 1, CreatedAt: now, UpdatedAt: now}
	guardInst := &models.EntityInstance{ID: "inst-guard", EntityTypeID: "et-guard", CatalogID: "cat-1", Name: "pii-filter", Version: 1, CreatedAt: now, UpdatedAt: now}

	link := &models.AssociationLink{ID: "link-1", AssociationID: "assoc-dir", SourceInstanceID: "inst-server", TargetInstanceID: "inst-guard"}

	systemTD := &models.TypeDefinition{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}
	systemTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}

	catalogRepo.On("GetByName", ctx(), "test-catalog").Return(catalog, nil)
	cvRepo.On("GetByID", ctx(), "cv-1").Return(cv, nil)
	pinRepo.On("ListByCatalogVersion", ctx(), "cv-1").Return([]*models.CatalogVersionPin{serverPin, guardPin}, nil)
	etvRepo.On("GetByID", ctx(), "etv-server").Return(serverETV, nil)
	etvRepo.On("GetByID", ctx(), "etv-guard").Return(guardETV, nil)
	etRepo.On("GetByID", ctx(), "et-server").Return(serverET, nil)
	etRepo.On("GetByID", ctx(), "et-guard").Return(guardET, nil)

	attrRepo.On("ListByVersion", ctx(), "etv-server").Return([]*models.Attribute{}, nil)
	guardAttr := &models.Attribute{ID: "attr-g1", EntityTypeVersionID: "etv-guard", Name: "level", TypeDefinitionVersionID: "tdv-str", Ordinal: 0}
	attrRepo.On("ListByVersion", ctx(), "etv-guard").Return([]*models.Attribute{guardAttr}, nil)
	tdvRepo.On("GetByID", ctx(), "tdv-str").Return(systemTDV, nil)
	tdRepo.On("GetByID", ctx(), "td-str").Return(systemTD, nil)

	assocRepo.On("ListByVersion", ctx(), "etv-server").Return([]*models.Association{dirAssoc}, nil)
	assocRepo.On("ListByVersion", ctx(), "etv-guard").Return([]*models.Association{}, nil)

	instRepo.On("ListByCatalog", ctx(), "cat-1").Return([]*models.EntityInstance{serverInst, guardInst}, nil)
	iavRepo.On("GetValuesForVersion", ctx(), "inst-server", 1).Return([]*models.InstanceAttributeValue{}, nil)
	guardIAV := &models.InstanceAttributeValue{ID: "iav-g1", InstanceID: "inst-guard", InstanceVersion: 1, AttributeID: "attr-g1", ValueString: "high"}
	iavRepo.On("GetValuesForVersion", ctx(), "inst-guard", 1).Return([]*models.InstanceAttributeValue{guardIAV}, nil)

	linkRepo.On("GetForwardRefs", ctx(), "inst-server").Return([]*models.AssociationLink{link}, nil)
	linkRepo.On("GetForwardRefs", ctx(), "inst-guard").Return([]*models.AssociationLink{}, nil)

	result, err := svc.ExportCatalog(context.Background(), "test-catalog", nil, "")
	require.NoError(t, err)

	// Find server instance and check links
	var serverExport *ExportInstance
	for i := range result.Instances {
		if result.Instances[i].Name == "github" {
			serverExport = &result.Instances[i]
			break
		}
	}
	require.NotNil(t, serverExport)
	require.Len(t, serverExport.Links, 1)
	assert.Equal(t, "pre-execute", serverExport.Links[0].Association)
	assert.Equal(t, "guardrail", serverExport.Links[0].TargetType)
	assert.Equal(t, "pii-filter", serverExport.Links[0].TargetName)
}

// T-30.08: Export with entity filter — only specified entity types included
func TestT30_08_ExportWithEntityFilter(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, _, _, instRepo, _, _ := newExportService()

	now := time.Now()
	catalog := &models.Catalog{ID: "cat-1", Name: "test-catalog", CatalogVersionID: "cv-1", ValidationStatus: models.ValidationStatusValid, CreatedAt: now, UpdatedAt: now}
	cv := &models.CatalogVersion{ID: "cv-1", VersionLabel: "v1.0"}

	et1 := &models.EntityType{ID: "et-1", Name: "server"}
	etv1 := &models.EntityTypeVersion{ID: "etv-1", EntityTypeID: "et-1", Version: 1, Description: "server"}
	et2 := &models.EntityType{ID: "et-2", Name: "guardrail"}
	etv2 := &models.EntityTypeVersion{ID: "etv-2", EntityTypeID: "et-2", Version: 1, Description: "guard"}

	pin1 := &models.CatalogVersionPin{ID: "pin-1", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-1"}
	pin2 := &models.CatalogVersionPin{ID: "pin-2", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-2"}

	catalogRepo.On("GetByName", ctx(), "test-catalog").Return(catalog, nil)
	cvRepo.On("GetByID", ctx(), "cv-1").Return(cv, nil)
	pinRepo.On("ListByCatalogVersion", ctx(), "cv-1").Return([]*models.CatalogVersionPin{pin1, pin2}, nil)
	etvRepo.On("GetByID", ctx(), "etv-1").Return(etv1, nil)
	etvRepo.On("GetByID", ctx(), "etv-2").Return(etv2, nil)
	etRepo.On("GetByID", ctx(), "et-1").Return(et1, nil)
	etRepo.On("GetByID", ctx(), "et-2").Return(et2, nil)
	attrRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Association{}, nil)
	assocRepo.On("ListByVersion", ctx(), "etv-2").Return([]*models.Association{}, nil)
	instRepo.On("ListByCatalog", ctx(), "cat-1").Return([]*models.EntityInstance{}, nil)

	// Filter to only "server"
	result, err := svc.ExportCatalog(context.Background(), "test-catalog", []string{"server"}, "")
	require.NoError(t, err)

	require.Len(t, result.EntityTypes, 1)
	assert.Equal(t, "server", result.EntityTypes[0].Name)
}

// T-30.09: Export with entity filter — associations to excluded types should be dropped
func TestT30_09_ExportFilterDropsAssocToExcludedType(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, _, _, instRepo, _, _ := newExportService()

	now := time.Now()
	catalog := &models.Catalog{ID: "cat-1", Name: "test-catalog", CatalogVersionID: "cv-1", ValidationStatus: models.ValidationStatusValid, CreatedAt: now, UpdatedAt: now}
	cv := &models.CatalogVersion{ID: "cv-1", VersionLabel: "v1.0"}

	serverET := &models.EntityType{ID: "et-1", Name: "server"}
	serverETV := &models.EntityTypeVersion{ID: "etv-1", EntityTypeID: "et-1", Version: 1, Description: "server"}
	guardET := &models.EntityType{ID: "et-2", Name: "guardrail"}
	guardETV := &models.EntityTypeVersion{ID: "etv-2", EntityTypeID: "et-2", Version: 1, Description: "guard"}

	pin1 := &models.CatalogVersionPin{ID: "pin-1", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-1"}
	pin2 := &models.CatalogVersionPin{ID: "pin-2", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-2"}

	dirAssoc := &models.Association{ID: "assoc-1", EntityTypeVersionID: "etv-1", Name: "pre-execute", TargetEntityTypeID: "et-2", Type: models.AssociationTypeDirectional}

	catalogRepo.On("GetByName", ctx(), "test-catalog").Return(catalog, nil)
	cvRepo.On("GetByID", ctx(), "cv-1").Return(cv, nil)
	pinRepo.On("ListByCatalogVersion", ctx(), "cv-1").Return([]*models.CatalogVersionPin{pin1, pin2}, nil)
	etvRepo.On("GetByID", ctx(), "etv-1").Return(serverETV, nil)
	etvRepo.On("GetByID", ctx(), "etv-2").Return(guardETV, nil)
	etRepo.On("GetByID", ctx(), "et-1").Return(serverET, nil)
	etRepo.On("GetByID", ctx(), "et-2").Return(guardET, nil)
	attrRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Association{dirAssoc}, nil)
	assocRepo.On("ListByVersion", ctx(), "etv-2").Return([]*models.Association{}, nil)
	instRepo.On("ListByCatalog", ctx(), "cat-1").Return([]*models.EntityInstance{}, nil)

	// Filter to only "server" — "guardrail" is excluded
	result, err := svc.ExportCatalog(context.Background(), "test-catalog", []string{"server"}, "")
	require.NoError(t, err)

	require.Len(t, result.EntityTypes, 1)
	assert.Equal(t, "server", result.EntityTypes[0].Name)
	// Association to "guardrail" should be dropped since guardrail is excluded
	assert.Empty(t, result.EntityTypes[0].Associations, "associations to excluded entity types should be dropped")
}

// T-30.10: Export source_system override
func TestT30_10_ExportSourceSystem(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, _, _, _, _, _, _, instRepo, _, _ := newExportService()

	now := time.Now()
	catalog := &models.Catalog{ID: "cat-1", Name: "test-catalog", CatalogVersionID: "cv-1", ValidationStatus: models.ValidationStatusValid, CreatedAt: now, UpdatedAt: now}
	cv := &models.CatalogVersion{ID: "cv-1", VersionLabel: "v1.0"}

	catalogRepo.On("GetByName", ctx(), "test-catalog").Return(catalog, nil)
	cvRepo.On("GetByID", ctx(), "cv-1").Return(cv, nil)
	pinRepo.On("ListByCatalogVersion", ctx(), "cv-1").Return([]*models.CatalogVersionPin{}, nil)
	instRepo.On("ListByCatalog", ctx(), "cat-1").Return([]*models.EntityInstance{}, nil)

	result, err := svc.ExportCatalog(context.Background(), "test-catalog", nil, "custom-system")
	require.NoError(t, err)
	assert.Equal(t, "custom-system", result.SourceSystem)
}

// T-30.11: Export catalog not found
func TestT30_11_ExportCatalogNotFound(t *testing.T) {
	svc, catalogRepo, _, _, _, _, _, _, _, _, _, _, _ := newExportService()

	catalogRepo.On("GetByName", ctx(), "nonexistent").Return(nil, domainerrors.NewNotFound("Catalog", "nonexistent"))

	result, err := svc.ExportCatalog(context.Background(), "nonexistent", nil, "")
	assert.Nil(t, result)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
}

// T-30.13: ExportInstance MarshalJSON — children nested under association name
func TestT30_13_ExportInstanceMarshalJSON(t *testing.T) {
	ei := ExportInstance{
		EntityType:  "mcp-server",
		Name:        "github",
		Description: "GitHub server",
		Attributes:  map[string]any{"endpoint": "https://example.com"},
		Links:       nil,
		Children: map[string][]*ExportInstance{
			"tools": {
				{EntityType: "mcp-tool", Name: "list-repos", Description: "List repos", Attributes: map[string]any{}, Children: make(map[string][]*ExportInstance)},
			},
		},
	}

	b, err := json.Marshal(ei)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))

	assert.Equal(t, "mcp-server", m["entity_type"])
	assert.Equal(t, "github", m["name"])
	// "tools" should be a top-level key
	tools, ok := m["tools"].([]any)
	require.True(t, ok, "expected 'tools' key as array")
	require.Len(t, tools, 1)
	tool := tools[0].(map[string]any)
	assert.Equal(t, "list-repos", tool["name"])
}

// T-30.20: Export with number and integer attribute values
func TestT30_20_ExportWithNumericAttributes(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, instRepo, iavRepo, linkRepo := newExportService()

	now := time.Now()
	catalog := &models.Catalog{ID: "cat-1", Name: "test-catalog", CatalogVersionID: "cv-1", ValidationStatus: models.ValidationStatusValid, CreatedAt: now, UpdatedAt: now}
	cv := &models.CatalogVersion{ID: "cv-1", VersionLabel: "v1.0"}
	et := &models.EntityType{ID: "et-1", Name: "metric"}
	etv := &models.EntityTypeVersion{ID: "etv-1", EntityTypeID: "et-1", Version: 1, Description: "Metric"}
	pin := &models.CatalogVersionPin{ID: "pin-1", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-1"}

	numTD := &models.TypeDefinition{ID: "td-num", Name: "number", BaseType: models.BaseTypeNumber, System: true}
	numTDV := &models.TypeDefinitionVersion{ID: "tdv-num", TypeDefinitionID: "td-num", VersionNumber: 1}
	intTD := &models.TypeDefinition{ID: "td-int", Name: "integer", BaseType: models.BaseTypeInteger, System: true}
	intTDV := &models.TypeDefinitionVersion{ID: "tdv-int", TypeDefinitionID: "td-int", VersionNumber: 1}

	attrs := []*models.Attribute{
		{ID: "attr-1", EntityTypeVersionID: "etv-1", Name: "score", TypeDefinitionVersionID: "tdv-num", Required: true, Ordinal: 0},
		{ID: "attr-2", EntityTypeVersionID: "etv-1", Name: "count", TypeDefinitionVersionID: "tdv-int", Required: false, Ordinal: 1},
	}

	inst := &models.EntityInstance{ID: "inst-1", EntityTypeID: "et-1", CatalogID: "cat-1", Name: "m1", Version: 1, CreatedAt: now, UpdatedAt: now}
	numVal := 3.14
	intVal := float64(42)
	iavs := []*models.InstanceAttributeValue{
		{ID: "iav-1", InstanceID: "inst-1", InstanceVersion: 1, AttributeID: "attr-1", ValueString: "3.14", ValueNumber: &numVal},
		{ID: "iav-2", InstanceID: "inst-1", InstanceVersion: 1, AttributeID: "attr-2", ValueString: "", ValueNumber: &intVal},
	}

	catalogRepo.On("GetByName", ctx(), "test-catalog").Return(catalog, nil)
	cvRepo.On("GetByID", ctx(), "cv-1").Return(cv, nil)
	pinRepo.On("ListByCatalogVersion", ctx(), "cv-1").Return([]*models.CatalogVersionPin{pin}, nil)
	etvRepo.On("GetByID", ctx(), "etv-1").Return(etv, nil)
	etRepo.On("GetByID", ctx(), "et-1").Return(et, nil)
	attrRepo.On("ListByVersion", ctx(), "etv-1").Return(attrs, nil)
	tdvRepo.On("GetByID", ctx(), "tdv-num").Return(numTDV, nil)
	tdRepo.On("GetByID", ctx(), "td-num").Return(numTD, nil)
	tdvRepo.On("GetByID", ctx(), "tdv-int").Return(intTDV, nil)
	tdRepo.On("GetByID", ctx(), "td-int").Return(intTD, nil)
	assocRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Association{}, nil)
	instRepo.On("ListByCatalog", ctx(), "cat-1").Return([]*models.EntityInstance{inst}, nil)
	iavRepo.On("GetValuesForVersion", ctx(), "inst-1", 1).Return(iavs, nil)
	linkRepo.On("GetForwardRefs", ctx(), "inst-1").Return([]*models.AssociationLink{}, nil)

	result, err := svc.ExportCatalog(context.Background(), "test-catalog", nil, "")
	require.NoError(t, err)

	require.Len(t, result.Instances, 1)
	// Number with ValueNumber set → should use ValueNumber (preserves JSON number type)
	assert.Equal(t, float64(3.14), result.Instances[0].Attributes["score"])
	// Integer with ValueNumber set → should use ValueNumber
	assert.Equal(t, float64(42), result.Instances[0].Attributes["count"])
}

// T-30.21: Export with JSON/list attribute values
func TestT30_21_ExportWithJSONListAttributes(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, instRepo, iavRepo, linkRepo := newExportService()

	now := time.Now()
	catalog := &models.Catalog{ID: "cat-1", Name: "test-catalog", CatalogVersionID: "cv-1", ValidationStatus: models.ValidationStatusValid, CreatedAt: now, UpdatedAt: now}
	cv := &models.CatalogVersion{ID: "cv-1", VersionLabel: "v1.0"}
	et := &models.EntityType{ID: "et-1", Name: "config"}
	etv := &models.EntityTypeVersion{ID: "etv-1", EntityTypeID: "et-1", Version: 1, Description: "Config"}
	pin := &models.CatalogVersionPin{ID: "pin-1", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-1"}

	listTD := &models.TypeDefinition{ID: "td-list", Name: "list", BaseType: models.BaseTypeList, System: true}
	listTDV := &models.TypeDefinitionVersion{ID: "tdv-list", TypeDefinitionID: "td-list", VersionNumber: 1}
	jsonTD := &models.TypeDefinition{ID: "td-json", Name: "json", BaseType: models.BaseTypeJSON, System: true}
	jsonTDV := &models.TypeDefinitionVersion{ID: "tdv-json", TypeDefinitionID: "td-json", VersionNumber: 1}

	attrs := []*models.Attribute{
		{ID: "attr-1", EntityTypeVersionID: "etv-1", Name: "tags", TypeDefinitionVersionID: "tdv-list", Required: false, Ordinal: 0},
		{ID: "attr-2", EntityTypeVersionID: "etv-1", Name: "metadata", TypeDefinitionVersionID: "tdv-json", Required: false, Ordinal: 1},
		{ID: "attr-3", EntityTypeVersionID: "etv-1", Name: "broken", TypeDefinitionVersionID: "tdv-list", Required: false, Ordinal: 2},
	}

	inst := &models.EntityInstance{ID: "inst-1", EntityTypeID: "et-1", CatalogID: "cat-1", Name: "c1", Version: 1, CreatedAt: now, UpdatedAt: now}
	iavs := []*models.InstanceAttributeValue{
		{ID: "iav-1", InstanceID: "inst-1", InstanceVersion: 1, AttributeID: "attr-1", ValueJSON: `["a","b"]`},
		{ID: "iav-2", InstanceID: "inst-1", InstanceVersion: 1, AttributeID: "attr-2", ValueJSON: `{"key":"val"}`},
		{ID: "iav-3", InstanceID: "inst-1", InstanceVersion: 1, AttributeID: "attr-3", ValueJSON: `{invalid json`},
	}

	catalogRepo.On("GetByName", ctx(), "test-catalog").Return(catalog, nil)
	cvRepo.On("GetByID", ctx(), "cv-1").Return(cv, nil)
	pinRepo.On("ListByCatalogVersion", ctx(), "cv-1").Return([]*models.CatalogVersionPin{pin}, nil)
	etvRepo.On("GetByID", ctx(), "etv-1").Return(etv, nil)
	etRepo.On("GetByID", ctx(), "et-1").Return(et, nil)
	attrRepo.On("ListByVersion", ctx(), "etv-1").Return(attrs, nil)
	tdvRepo.On("GetByID", ctx(), "tdv-list").Return(listTDV, nil)
	tdRepo.On("GetByID", ctx(), "td-list").Return(listTD, nil)
	tdvRepo.On("GetByID", ctx(), "tdv-json").Return(jsonTDV, nil)
	tdRepo.On("GetByID", ctx(), "td-json").Return(jsonTD, nil)
	assocRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Association{}, nil)
	instRepo.On("ListByCatalog", ctx(), "cat-1").Return([]*models.EntityInstance{inst}, nil)
	iavRepo.On("GetValuesForVersion", ctx(), "inst-1", 1).Return(iavs, nil)
	linkRepo.On("GetForwardRefs", ctx(), "inst-1").Return([]*models.AssociationLink{}, nil)

	result, err := svc.ExportCatalog(context.Background(), "test-catalog", nil, "")
	require.NoError(t, err)

	require.Len(t, result.Instances, 1)
	// List → parsed JSON array
	tags, ok := result.Instances[0].Attributes["tags"].([]any)
	require.True(t, ok)
	assert.Len(t, tags, 2)
	// JSON → parsed JSON object
	meta, ok := result.Instances[0].Attributes["metadata"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "val", meta["key"])
	// Broken JSON → raw string fallback
	assert.Equal(t, `{invalid json`, result.Instances[0].Attributes["broken"])
}

// T-30.22: Export with link to contained instance — target_path populated
func TestT30_22_ExportLinkWithTargetPath(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, _, _, instRepo, iavRepo, linkRepo := newExportService()

	now := time.Now()
	catalog := &models.Catalog{ID: "cat-1", Name: "test-catalog", CatalogVersionID: "cv-1", ValidationStatus: models.ValidationStatusValid, CreatedAt: now, UpdatedAt: now}
	cv := &models.CatalogVersion{ID: "cv-1", VersionLabel: "v1.0"}

	serverET := &models.EntityType{ID: "et-server", Name: "server"}
	serverETV := &models.EntityTypeVersion{ID: "etv-server", EntityTypeID: "et-server", Version: 1, Description: "server"}
	toolET := &models.EntityType{ID: "et-tool", Name: "tool"}
	toolETV := &models.EntityTypeVersion{ID: "etv-tool", EntityTypeID: "et-tool", Version: 1, Description: "tool"}

	serverPin := &models.CatalogVersionPin{ID: "pin-1", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-server"}
	toolPin := &models.CatalogVersionPin{ID: "pin-2", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-tool"}

	// Containment: server has tools
	containAssoc := &models.Association{ID: "assoc-contain", EntityTypeVersionID: "etv-server", Name: "tools", TargetEntityTypeID: "et-tool", Type: models.AssociationTypeContainment}
	// Directional: tool references another tool
	dirAssoc := &models.Association{ID: "assoc-dir", EntityTypeVersionID: "etv-tool", Name: "depends-on", TargetEntityTypeID: "et-tool", Type: models.AssociationTypeDirectional}

	parentInst := &models.EntityInstance{ID: "inst-parent", EntityTypeID: "et-server", CatalogID: "cat-1", Name: "github", Version: 1, CreatedAt: now, UpdatedAt: now}
	toolInst := &models.EntityInstance{ID: "inst-tool", EntityTypeID: "et-tool", CatalogID: "cat-1", ParentInstanceID: "inst-parent", Name: "tool-a", Version: 1, CreatedAt: now, UpdatedAt: now}
	targetToolInst := &models.EntityInstance{ID: "inst-target", EntityTypeID: "et-tool", CatalogID: "cat-1", ParentInstanceID: "inst-parent", Name: "tool-b", Version: 1, CreatedAt: now, UpdatedAt: now}

	link := &models.AssociationLink{ID: "link-1", AssociationID: "assoc-dir", SourceInstanceID: "inst-tool", TargetInstanceID: "inst-target"}

	catalogRepo.On("GetByName", ctx(), "test-catalog").Return(catalog, nil)
	cvRepo.On("GetByID", ctx(), "cv-1").Return(cv, nil)
	pinRepo.On("ListByCatalogVersion", ctx(), "cv-1").Return([]*models.CatalogVersionPin{serverPin, toolPin}, nil)
	etvRepo.On("GetByID", ctx(), "etv-server").Return(serverETV, nil)
	etvRepo.On("GetByID", ctx(), "etv-tool").Return(toolETV, nil)
	etRepo.On("GetByID", ctx(), "et-server").Return(serverET, nil)
	etRepo.On("GetByID", ctx(), "et-tool").Return(toolET, nil)
	attrRepo.On("ListByVersion", ctx(), "etv-server").Return([]*models.Attribute{}, nil)
	attrRepo.On("ListByVersion", ctx(), "etv-tool").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", ctx(), "etv-server").Return([]*models.Association{containAssoc}, nil)
	assocRepo.On("ListByVersion", ctx(), "etv-tool").Return([]*models.Association{dirAssoc}, nil)
	instRepo.On("ListByCatalog", ctx(), "cat-1").Return([]*models.EntityInstance{parentInst, toolInst, targetToolInst}, nil)
	iavRepo.On("GetValuesForVersion", ctx(), "inst-parent", 1).Return([]*models.InstanceAttributeValue{}, nil)
	iavRepo.On("GetValuesForVersion", ctx(), "inst-tool", 1).Return([]*models.InstanceAttributeValue{}, nil)
	iavRepo.On("GetValuesForVersion", ctx(), "inst-target", 1).Return([]*models.InstanceAttributeValue{}, nil)
	linkRepo.On("GetForwardRefs", ctx(), "inst-parent").Return([]*models.AssociationLink{}, nil)
	linkRepo.On("GetForwardRefs", ctx(), "inst-tool").Return([]*models.AssociationLink{link}, nil)
	linkRepo.On("GetForwardRefs", ctx(), "inst-target").Return([]*models.AssociationLink{}, nil)

	result, err := svc.ExportCatalog(context.Background(), "test-catalog", nil, "")
	require.NoError(t, err)

	require.Len(t, result.Instances, 1)
	// Parent instance has children
	require.Len(t, result.Instances[0].Children["tools"], 2)
	// Find tool-a's links — it links to tool-b which is contained
	var toolA *ExportInstance
	for _, child := range result.Instances[0].Children["tools"] {
		if child.Name == "tool-a" {
			toolA = child
			break
		}
	}
	require.NotNil(t, toolA)
	require.Len(t, toolA.Links, 1)
	assert.Equal(t, "depends-on", toolA.Links[0].Association)
	assert.Equal(t, "tool-b", toolA.Links[0].TargetName)
	// Target is contained → should have target_path
	assert.Equal(t, []string{"github"}, toolA.Links[0].TargetPath)
}

// T-30.23: Export with link to non-existent target instance — skipped
func TestT30_23_ExportLinkTargetNilSkipped(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, _, _, instRepo, iavRepo, linkRepo := newExportService()

	now := time.Now()
	catalog := &models.Catalog{ID: "cat-1", Name: "test-catalog", CatalogVersionID: "cv-1", ValidationStatus: models.ValidationStatusValid, CreatedAt: now, UpdatedAt: now}
	cv := &models.CatalogVersion{ID: "cv-1", VersionLabel: "v1.0"}
	et := &models.EntityType{ID: "et-1", Name: "server"}
	etv := &models.EntityTypeVersion{ID: "etv-1", EntityTypeID: "et-1", Version: 1, Description: "server"}
	pin := &models.CatalogVersionPin{ID: "pin-1", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-1"}

	dirAssoc := &models.Association{ID: "assoc-1", EntityTypeVersionID: "etv-1", Name: "dep", TargetEntityTypeID: "et-1", Type: models.AssociationTypeDirectional}

	inst := &models.EntityInstance{ID: "inst-1", EntityTypeID: "et-1", CatalogID: "cat-1", Name: "s1", Version: 1, CreatedAt: now, UpdatedAt: now}
	// Link to a target that doesn't exist in the catalog
	link := &models.AssociationLink{ID: "link-1", AssociationID: "assoc-1", SourceInstanceID: "inst-1", TargetInstanceID: "inst-missing"}

	catalogRepo.On("GetByName", ctx(), "test-catalog").Return(catalog, nil)
	cvRepo.On("GetByID", ctx(), "cv-1").Return(cv, nil)
	pinRepo.On("ListByCatalogVersion", ctx(), "cv-1").Return([]*models.CatalogVersionPin{pin}, nil)
	etvRepo.On("GetByID", ctx(), "etv-1").Return(etv, nil)
	etRepo.On("GetByID", ctx(), "et-1").Return(et, nil)
	attrRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Association{dirAssoc}, nil)
	instRepo.On("ListByCatalog", ctx(), "cat-1").Return([]*models.EntityInstance{inst}, nil)
	iavRepo.On("GetValuesForVersion", ctx(), "inst-1", 1).Return([]*models.InstanceAttributeValue{}, nil)
	linkRepo.On("GetForwardRefs", ctx(), "inst-1").Return([]*models.AssociationLink{link}, nil)

	result, err := svc.ExportCatalog(context.Background(), "test-catalog", nil, "")
	require.NoError(t, err)

	require.Len(t, result.Instances, 1)
	assert.Empty(t, result.Instances[0].Links, "link to missing target should be skipped")
}

// T-30.24: Export with containment assoc link skipped (not directional/bidirectional)
func TestT30_24_ExportContainmentLinkSkipped(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, _, _, instRepo, iavRepo, linkRepo := newExportService()

	now := time.Now()
	catalog := &models.Catalog{ID: "cat-1", Name: "test-catalog", CatalogVersionID: "cv-1", ValidationStatus: models.ValidationStatusValid, CreatedAt: now, UpdatedAt: now}
	cv := &models.CatalogVersion{ID: "cv-1", VersionLabel: "v1.0"}
	et := &models.EntityType{ID: "et-1", Name: "server"}
	etv := &models.EntityTypeVersion{ID: "etv-1", EntityTypeID: "et-1", Version: 1, Description: "server"}
	pin := &models.CatalogVersionPin{ID: "pin-1", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-1"}

	containAssoc := &models.Association{ID: "assoc-1", EntityTypeVersionID: "etv-1", Name: "contains", TargetEntityTypeID: "et-1", Type: models.AssociationTypeContainment}

	inst := &models.EntityInstance{ID: "inst-1", EntityTypeID: "et-1", CatalogID: "cat-1", Name: "s1", Version: 1, CreatedAt: now, UpdatedAt: now}
	inst2 := &models.EntityInstance{ID: "inst-2", EntityTypeID: "et-1", CatalogID: "cat-1", Name: "s2", Version: 1, CreatedAt: now, UpdatedAt: now}
	// Containment link — should be skipped in export links (children are nested)
	link := &models.AssociationLink{ID: "link-1", AssociationID: "assoc-1", SourceInstanceID: "inst-1", TargetInstanceID: "inst-2"}

	catalogRepo.On("GetByName", ctx(), "test-catalog").Return(catalog, nil)
	cvRepo.On("GetByID", ctx(), "cv-1").Return(cv, nil)
	pinRepo.On("ListByCatalogVersion", ctx(), "cv-1").Return([]*models.CatalogVersionPin{pin}, nil)
	etvRepo.On("GetByID", ctx(), "etv-1").Return(etv, nil)
	etRepo.On("GetByID", ctx(), "et-1").Return(et, nil)
	attrRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Association{containAssoc}, nil)
	instRepo.On("ListByCatalog", ctx(), "cat-1").Return([]*models.EntityInstance{inst, inst2}, nil)
	iavRepo.On("GetValuesForVersion", ctx(), "inst-1", 1).Return([]*models.InstanceAttributeValue{}, nil)
	iavRepo.On("GetValuesForVersion", ctx(), "inst-2", 1).Return([]*models.InstanceAttributeValue{}, nil)
	linkRepo.On("GetForwardRefs", ctx(), "inst-1").Return([]*models.AssociationLink{link}, nil)
	linkRepo.On("GetForwardRefs", ctx(), "inst-2").Return([]*models.AssociationLink{}, nil)

	result, err := svc.ExportCatalog(context.Background(), "test-catalog", nil, "")
	require.NoError(t, err)

	require.Len(t, result.Instances, 2)
	for _, inst := range result.Instances {
		assert.Empty(t, inst.Links, "containment links should not appear in export links")
	}
}

// T-30.25: Export with link to filtered-out entity type — skipped
func TestT30_25_ExportLinkToFilteredTypeSkipped(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, _, _, instRepo, iavRepo, linkRepo := newExportService()

	now := time.Now()
	catalog := &models.Catalog{ID: "cat-1", Name: "test-catalog", CatalogVersionID: "cv-1", ValidationStatus: models.ValidationStatusValid, CreatedAt: now, UpdatedAt: now}
	cv := &models.CatalogVersion{ID: "cv-1", VersionLabel: "v1.0"}

	serverET := &models.EntityType{ID: "et-server", Name: "server"}
	serverETV := &models.EntityTypeVersion{ID: "etv-server", EntityTypeID: "et-server", Version: 1}
	guardET := &models.EntityType{ID: "et-guard", Name: "guardrail"}
	guardETV := &models.EntityTypeVersion{ID: "etv-guard", EntityTypeID: "et-guard", Version: 1}

	serverPin := &models.CatalogVersionPin{ID: "pin-1", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-server"}
	guardPin := &models.CatalogVersionPin{ID: "pin-2", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-guard"}

	dirAssoc := &models.Association{ID: "assoc-1", EntityTypeVersionID: "etv-server", Name: "dep", TargetEntityTypeID: "et-guard", Type: models.AssociationTypeDirectional}

	serverInst := &models.EntityInstance{ID: "inst-s", EntityTypeID: "et-server", CatalogID: "cat-1", Name: "s1", Version: 1, CreatedAt: now, UpdatedAt: now}
	guardInst := &models.EntityInstance{ID: "inst-g", EntityTypeID: "et-guard", CatalogID: "cat-1", Name: "g1", Version: 1, CreatedAt: now, UpdatedAt: now}
	link := &models.AssociationLink{ID: "link-1", AssociationID: "assoc-1", SourceInstanceID: "inst-s", TargetInstanceID: "inst-g"}

	catalogRepo.On("GetByName", ctx(), "test-catalog").Return(catalog, nil)
	cvRepo.On("GetByID", ctx(), "cv-1").Return(cv, nil)
	pinRepo.On("ListByCatalogVersion", ctx(), "cv-1").Return([]*models.CatalogVersionPin{serverPin, guardPin}, nil)
	etvRepo.On("GetByID", ctx(), "etv-server").Return(serverETV, nil)
	etvRepo.On("GetByID", ctx(), "etv-guard").Return(guardETV, nil)
	etRepo.On("GetByID", ctx(), "et-server").Return(serverET, nil)
	etRepo.On("GetByID", ctx(), "et-guard").Return(guardET, nil)
	attrRepo.On("ListByVersion", ctx(), "etv-server").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", ctx(), "etv-server").Return([]*models.Association{dirAssoc}, nil)
	assocRepo.On("ListByVersion", ctx(), "etv-guard").Return([]*models.Association{}, nil)
	instRepo.On("ListByCatalog", ctx(), "cat-1").Return([]*models.EntityInstance{serverInst, guardInst}, nil)
	iavRepo.On("GetValuesForVersion", ctx(), "inst-s", 1).Return([]*models.InstanceAttributeValue{}, nil)
	linkRepo.On("GetForwardRefs", ctx(), "inst-s").Return([]*models.AssociationLink{link}, nil)

	// Filter to only "server" — guardrail excluded
	result, err := svc.ExportCatalog(context.Background(), "test-catalog", []string{"server"}, "")
	require.NoError(t, err)

	require.Len(t, result.Instances, 1)
	assert.Empty(t, result.Instances[0].Links, "links to filtered-out entity types should be excluded")
}

// T-30.12: Export error paths — CV lookup, pin list, attribute, etc.
func TestT30_12_ExportErrorPaths(t *testing.T) {
	someErr := domainerrors.NewNotFound("x", "y")

	t.Run("cv_lookup_error", func(t *testing.T) {
		svc, catalogRepo, cvRepo, _, _, _, _, _, _, _, _, _, _ := newExportService()
		now := time.Now()
		catalog := &models.Catalog{ID: "cat-1", Name: "test-catalog", CatalogVersionID: "cv-1", CreatedAt: now, UpdatedAt: now}
		catalogRepo.On("GetByName", ctx(), "test-catalog").Return(catalog, nil)
		cvRepo.On("GetByID", ctx(), "cv-1").Return(nil, someErr)
		_, err := svc.ExportCatalog(context.Background(), "test-catalog", nil, "")
		assert.Error(t, err)
	})

	t.Run("pin_list_error", func(t *testing.T) {
		svc, catalogRepo, cvRepo, pinRepo, _, _, _, _, _, _, _, _, _ := newExportService()
		now := time.Now()
		catalog := &models.Catalog{ID: "cat-1", Name: "test-catalog", CatalogVersionID: "cv-1", CreatedAt: now, UpdatedAt: now}
		cv := &models.CatalogVersion{ID: "cv-1", VersionLabel: "v1.0"}
		catalogRepo.On("GetByName", ctx(), "test-catalog").Return(catalog, nil)
		cvRepo.On("GetByID", ctx(), "cv-1").Return(cv, nil)
		pinRepo.On("ListByCatalogVersion", ctx(), "cv-1").Return(nil, someErr)
		_, err := svc.ExportCatalog(context.Background(), "test-catalog", nil, "")
		assert.Error(t, err)
	})

	t.Run("etv_lookup_error", func(t *testing.T) {
		svc, catalogRepo, cvRepo, pinRepo, _, etvRepo, _, _, _, _, _, _, _ := newExportService()
		now := time.Now()
		catalog := &models.Catalog{ID: "cat-1", Name: "test-catalog", CatalogVersionID: "cv-1", CreatedAt: now, UpdatedAt: now}
		cv := &models.CatalogVersion{ID: "cv-1", VersionLabel: "v1.0"}
		pin := &models.CatalogVersionPin{ID: "pin-1", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-1"}
		catalogRepo.On("GetByName", ctx(), "test-catalog").Return(catalog, nil)
		cvRepo.On("GetByID", ctx(), "cv-1").Return(cv, nil)
		pinRepo.On("ListByCatalogVersion", ctx(), "cv-1").Return([]*models.CatalogVersionPin{pin}, nil)
		etvRepo.On("GetByID", ctx(), "etv-1").Return(nil, someErr)
		_, err := svc.ExportCatalog(context.Background(), "test-catalog", nil, "")
		assert.Error(t, err)
	})

	t.Run("et_lookup_error", func(t *testing.T) {
		svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, _, _, _, _, _, _, _ := newExportService()
		now := time.Now()
		catalog := &models.Catalog{ID: "cat-1", Name: "test-catalog", CatalogVersionID: "cv-1", CreatedAt: now, UpdatedAt: now}
		cv := &models.CatalogVersion{ID: "cv-1", VersionLabel: "v1.0"}
		etv := &models.EntityTypeVersion{ID: "etv-1", EntityTypeID: "et-1", Version: 1}
		pin := &models.CatalogVersionPin{ID: "pin-1", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-1"}
		catalogRepo.On("GetByName", ctx(), "test-catalog").Return(catalog, nil)
		cvRepo.On("GetByID", ctx(), "cv-1").Return(cv, nil)
		pinRepo.On("ListByCatalogVersion", ctx(), "cv-1").Return([]*models.CatalogVersionPin{pin}, nil)
		etvRepo.On("GetByID", ctx(), "etv-1").Return(etv, nil)
		etRepo.On("GetByID", ctx(), "et-1").Return(nil, someErr)
		_, err := svc.ExportCatalog(context.Background(), "test-catalog", nil, "")
		assert.Error(t, err)
	})

	t.Run("attr_list_error", func(t *testing.T) {
		svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, _, _, _, _, _, _ := newExportService()
		now := time.Now()
		catalog := &models.Catalog{ID: "cat-1", Name: "test-catalog", CatalogVersionID: "cv-1", CreatedAt: now, UpdatedAt: now}
		cv := &models.CatalogVersion{ID: "cv-1", VersionLabel: "v1.0"}
		et := &models.EntityType{ID: "et-1", Name: "server"}
		etv := &models.EntityTypeVersion{ID: "etv-1", EntityTypeID: "et-1", Version: 1}
		pin := &models.CatalogVersionPin{ID: "pin-1", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-1"}
		catalogRepo.On("GetByName", ctx(), "test-catalog").Return(catalog, nil)
		cvRepo.On("GetByID", ctx(), "cv-1").Return(cv, nil)
		pinRepo.On("ListByCatalogVersion", ctx(), "cv-1").Return([]*models.CatalogVersionPin{pin}, nil)
		etvRepo.On("GetByID", ctx(), "etv-1").Return(etv, nil)
		etRepo.On("GetByID", ctx(), "et-1").Return(et, nil)
		attrRepo.On("ListByVersion", ctx(), "etv-1").Return(nil, someErr)
		_, err := svc.ExportCatalog(context.Background(), "test-catalog", nil, "")
		assert.Error(t, err)
	})

	t.Run("tdv_lookup_error", func(t *testing.T) {
		svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, _, tdvRepo, _, _, _ := newExportService()
		now := time.Now()
		catalog := &models.Catalog{ID: "cat-1", Name: "test-catalog", CatalogVersionID: "cv-1", CreatedAt: now, UpdatedAt: now}
		cv := &models.CatalogVersion{ID: "cv-1", VersionLabel: "v1.0"}
		et := &models.EntityType{ID: "et-1", Name: "server"}
		etv := &models.EntityTypeVersion{ID: "etv-1", EntityTypeID: "et-1", Version: 1}
		pin := &models.CatalogVersionPin{ID: "pin-1", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-1"}
		attr := &models.Attribute{ID: "attr-1", EntityTypeVersionID: "etv-1", Name: "ep", TypeDefinitionVersionID: "tdv-1"}
		catalogRepo.On("GetByName", ctx(), "test-catalog").Return(catalog, nil)
		cvRepo.On("GetByID", ctx(), "cv-1").Return(cv, nil)
		pinRepo.On("ListByCatalogVersion", ctx(), "cv-1").Return([]*models.CatalogVersionPin{pin}, nil)
		etvRepo.On("GetByID", ctx(), "etv-1").Return(etv, nil)
		etRepo.On("GetByID", ctx(), "et-1").Return(et, nil)
		attrRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Attribute{attr}, nil)
		tdvRepo.On("GetByID", ctx(), "tdv-1").Return(nil, someErr)
		assocRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Association{}, nil)
		_, err := svc.ExportCatalog(context.Background(), "test-catalog", nil, "")
		assert.Error(t, err)
	})

	t.Run("td_lookup_error", func(t *testing.T) {
		svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, _, _, _ := newExportService()
		now := time.Now()
		catalog := &models.Catalog{ID: "cat-1", Name: "test-catalog", CatalogVersionID: "cv-1", CreatedAt: now, UpdatedAt: now}
		cv := &models.CatalogVersion{ID: "cv-1", VersionLabel: "v1.0"}
		et := &models.EntityType{ID: "et-1", Name: "server"}
		etv := &models.EntityTypeVersion{ID: "etv-1", EntityTypeID: "et-1", Version: 1}
		pin := &models.CatalogVersionPin{ID: "pin-1", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-1"}
		attr := &models.Attribute{ID: "attr-1", EntityTypeVersionID: "etv-1", Name: "ep", TypeDefinitionVersionID: "tdv-1"}
		tdv := &models.TypeDefinitionVersion{ID: "tdv-1", TypeDefinitionID: "td-1"}
		catalogRepo.On("GetByName", ctx(), "test-catalog").Return(catalog, nil)
		cvRepo.On("GetByID", ctx(), "cv-1").Return(cv, nil)
		pinRepo.On("ListByCatalogVersion", ctx(), "cv-1").Return([]*models.CatalogVersionPin{pin}, nil)
		etvRepo.On("GetByID", ctx(), "etv-1").Return(etv, nil)
		etRepo.On("GetByID", ctx(), "et-1").Return(et, nil)
		attrRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Attribute{attr}, nil)
		tdvRepo.On("GetByID", ctx(), "tdv-1").Return(tdv, nil)
		tdRepo.On("GetByID", ctx(), "td-1").Return(nil, someErr)
		assocRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Association{}, nil)
		_, err := svc.ExportCatalog(context.Background(), "test-catalog", nil, "")
		assert.Error(t, err)
	})

	t.Run("assoc_list_error", func(t *testing.T) {
		svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, _, _, _, _, _ := newExportService()
		now := time.Now()
		catalog := &models.Catalog{ID: "cat-1", Name: "test-catalog", CatalogVersionID: "cv-1", CreatedAt: now, UpdatedAt: now}
		cv := &models.CatalogVersion{ID: "cv-1", VersionLabel: "v1.0"}
		et := &models.EntityType{ID: "et-1", Name: "server"}
		etv := &models.EntityTypeVersion{ID: "etv-1", EntityTypeID: "et-1", Version: 1}
		pin := &models.CatalogVersionPin{ID: "pin-1", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-1"}
		catalogRepo.On("GetByName", ctx(), "test-catalog").Return(catalog, nil)
		cvRepo.On("GetByID", ctx(), "cv-1").Return(cv, nil)
		pinRepo.On("ListByCatalogVersion", ctx(), "cv-1").Return([]*models.CatalogVersionPin{pin}, nil)
		etvRepo.On("GetByID", ctx(), "etv-1").Return(etv, nil)
		etRepo.On("GetByID", ctx(), "et-1").Return(et, nil)
		attrRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Attribute{}, nil)
		assocRepo.On("ListByVersion", ctx(), "etv-1").Return(nil, someErr)
		_, err := svc.ExportCatalog(context.Background(), "test-catalog", nil, "")
		assert.Error(t, err)
	})

	t.Run("assoc_target_et_lookup_error", func(t *testing.T) {
		svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, _, _, _, _, _ := newExportService()
		now := time.Now()
		catalog := &models.Catalog{ID: "cat-1", Name: "test-catalog", CatalogVersionID: "cv-1", CreatedAt: now, UpdatedAt: now}
		cv := &models.CatalogVersion{ID: "cv-1", VersionLabel: "v1.0"}
		et := &models.EntityType{ID: "et-1", Name: "server"}
		etv := &models.EntityTypeVersion{ID: "etv-1", EntityTypeID: "et-1", Version: 1}
		pin := &models.CatalogVersionPin{ID: "pin-1", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-1"}
		assoc := &models.Association{ID: "a1", EntityTypeVersionID: "etv-1", Name: "dep", TargetEntityTypeID: "et-bad", Type: models.AssociationTypeDirectional}
		catalogRepo.On("GetByName", ctx(), "test-catalog").Return(catalog, nil)
		cvRepo.On("GetByID", ctx(), "cv-1").Return(cv, nil)
		pinRepo.On("ListByCatalogVersion", ctx(), "cv-1").Return([]*models.CatalogVersionPin{pin}, nil)
		etvRepo.On("GetByID", ctx(), "etv-1").Return(etv, nil)
		etRepo.On("GetByID", ctx(), "et-1").Return(et, nil)
		attrRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Attribute{}, nil)
		assocRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Association{assoc}, nil)
		etRepo.On("GetByID", ctx(), "et-bad").Return(nil, someErr)
		_, err := svc.ExportCatalog(context.Background(), "test-catalog", nil, "")
		assert.Error(t, err)
	})

	t.Run("list_by_catalog_error", func(t *testing.T) {
		svc, catalogRepo, cvRepo, pinRepo, _, _, _, _, _, _, instRepo, _, _ := newExportService()
		now := time.Now()
		catalog := &models.Catalog{ID: "cat-1", Name: "test-catalog", CatalogVersionID: "cv-1", CreatedAt: now, UpdatedAt: now}
		cv := &models.CatalogVersion{ID: "cv-1", VersionLabel: "v1.0"}
		catalogRepo.On("GetByName", ctx(), "test-catalog").Return(catalog, nil)
		cvRepo.On("GetByID", ctx(), "cv-1").Return(cv, nil)
		pinRepo.On("ListByCatalogVersion", ctx(), "cv-1").Return([]*models.CatalogVersionPin{}, nil)
		instRepo.On("ListByCatalog", ctx(), "cat-1").Return(nil, someErr)
		_, err := svc.ExportCatalog(context.Background(), "test-catalog", nil, "")
		assert.Error(t, err)
	})

	t.Run("type_def_tdv_lookup_error_in_loop", func(t *testing.T) {
		svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, instRepo, _, _ := newExportService()
		now := time.Now()
		catalog := &models.Catalog{ID: "cat-1", Name: "test-catalog", CatalogVersionID: "cv-1", CreatedAt: now, UpdatedAt: now}
		cv := &models.CatalogVersion{ID: "cv-1", VersionLabel: "v1.0"}
		et := &models.EntityType{ID: "et-1", Name: "server"}
		etv := &models.EntityTypeVersion{ID: "etv-1", EntityTypeID: "et-1", Version: 1}
		pin := &models.CatalogVersionPin{ID: "pin-1", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-1"}
		attr := &models.Attribute{ID: "attr-1", EntityTypeVersionID: "etv-1", Name: "id", TypeDefinitionVersionID: "tdv-custom"}
		customTDV := &models.TypeDefinitionVersion{ID: "tdv-custom", TypeDefinitionID: "td-custom"}
		customTD := &models.TypeDefinition{ID: "td-custom", Name: "hex", BaseType: models.BaseTypeString, System: false}

		catalogRepo.On("GetByName", ctx(), "test-catalog").Return(catalog, nil)
		cvRepo.On("GetByID", ctx(), "cv-1").Return(cv, nil)
		pinRepo.On("ListByCatalogVersion", ctx(), "cv-1").Return([]*models.CatalogVersionPin{pin}, nil)
		etvRepo.On("GetByID", ctx(), "etv-1").Return(etv, nil)
		etRepo.On("GetByID", ctx(), "et-1").Return(et, nil)
		attrRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Attribute{attr}, nil)
		// First call resolves attr → success
		tdvRepo.On("GetByID", ctx(), "tdv-custom").Return(customTDV, nil).Once()
		tdRepo.On("GetByID", ctx(), "td-custom").Return(customTD, nil).Once()
		assocRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Association{}, nil)
		instRepo.On("ListByCatalog", ctx(), "cat-1").Return([]*models.EntityInstance{}, nil)
		// Second call in tdvIDSet loop → error
		tdvRepo.On("GetByID", ctx(), "tdv-custom").Return(nil, someErr).Once()
		_, err := svc.ExportCatalog(context.Background(), "test-catalog", nil, "")
		assert.Error(t, err)
	})

	t.Run("type_def_td_lookup_error_in_loop", func(t *testing.T) {
		svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, instRepo, _, _ := newExportService()
		now := time.Now()
		catalog := &models.Catalog{ID: "cat-1", Name: "test-catalog", CatalogVersionID: "cv-1", CreatedAt: now, UpdatedAt: now}
		cv := &models.CatalogVersion{ID: "cv-1", VersionLabel: "v1.0"}
		et := &models.EntityType{ID: "et-1", Name: "server"}
		etv := &models.EntityTypeVersion{ID: "etv-1", EntityTypeID: "et-1", Version: 1}
		pin := &models.CatalogVersionPin{ID: "pin-1", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-1"}
		attr := &models.Attribute{ID: "attr-1", EntityTypeVersionID: "etv-1", Name: "id", TypeDefinitionVersionID: "tdv-custom"}
		customTDV := &models.TypeDefinitionVersion{ID: "tdv-custom", TypeDefinitionID: "td-custom"}
		customTD := &models.TypeDefinition{ID: "td-custom", Name: "hex", BaseType: models.BaseTypeString, System: false}

		catalogRepo.On("GetByName", ctx(), "test-catalog").Return(catalog, nil)
		cvRepo.On("GetByID", ctx(), "cv-1").Return(cv, nil)
		pinRepo.On("ListByCatalogVersion", ctx(), "cv-1").Return([]*models.CatalogVersionPin{pin}, nil)
		etvRepo.On("GetByID", ctx(), "etv-1").Return(etv, nil)
		etRepo.On("GetByID", ctx(), "et-1").Return(et, nil)
		attrRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Attribute{attr}, nil)
		// First call resolves attr → success
		tdvRepo.On("GetByID", ctx(), "tdv-custom").Return(customTDV, nil)
		tdRepo.On("GetByID", ctx(), "td-custom").Return(customTD, nil).Once()
		assocRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Association{}, nil)
		instRepo.On("ListByCatalog", ctx(), "cat-1").Return([]*models.EntityInstance{}, nil)
		// Second call in tdvIDSet loop → td error
		tdRepo.On("GetByID", ctx(), "td-custom").Return(nil, someErr).Once()
		_, err := svc.ExportCatalog(context.Background(), "test-catalog", nil, "")
		assert.Error(t, err)
	})

	t.Run("instance_attr_list_error", func(t *testing.T) {
		svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, _, _, instRepo, _, _ := newExportService()
		now := time.Now()
		catalog := &models.Catalog{ID: "cat-1", Name: "test-catalog", CatalogVersionID: "cv-1", CreatedAt: now, UpdatedAt: now}
		cv := &models.CatalogVersion{ID: "cv-1", VersionLabel: "v1.0"}
		et := &models.EntityType{ID: "et-1", Name: "server"}
		etv := &models.EntityTypeVersion{ID: "etv-1", EntityTypeID: "et-1", Version: 1}
		pin := &models.CatalogVersionPin{ID: "pin-1", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-1"}
		inst := &models.EntityInstance{ID: "inst-1", EntityTypeID: "et-1", CatalogID: "cat-1", Name: "s1", Version: 1, CreatedAt: now, UpdatedAt: now}
		catalogRepo.On("GetByName", ctx(), "test-catalog").Return(catalog, nil)
		cvRepo.On("GetByID", ctx(), "cv-1").Return(cv, nil)
		pinRepo.On("ListByCatalogVersion", ctx(), "cv-1").Return([]*models.CatalogVersionPin{pin}, nil)
		etvRepo.On("GetByID", ctx(), "etv-1").Return(etv, nil)
		etRepo.On("GetByID", ctx(), "et-1").Return(et, nil)
		// First call in entity type loop → success
		attrRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Attribute{}, nil).Once()
		assocRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Association{}, nil)
		instRepo.On("ListByCatalog", ctx(), "cat-1").Return([]*models.EntityInstance{inst}, nil)
		// Second call in buildExportInstances → error
		attrRepo.On("ListByVersion", ctx(), "etv-1").Return(nil, someErr).Once()
		_, err := svc.ExportCatalog(context.Background(), "test-catalog", nil, "")
		assert.Error(t, err)
	})

	t.Run("instance_iav_error", func(t *testing.T) {
		svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, _, _, instRepo, iavRepo, _ := newExportService()
		now := time.Now()
		catalog := &models.Catalog{ID: "cat-1", Name: "test-catalog", CatalogVersionID: "cv-1", CreatedAt: now, UpdatedAt: now}
		cv := &models.CatalogVersion{ID: "cv-1", VersionLabel: "v1.0"}
		et := &models.EntityType{ID: "et-1", Name: "server"}
		etv := &models.EntityTypeVersion{ID: "etv-1", EntityTypeID: "et-1", Version: 1}
		pin := &models.CatalogVersionPin{ID: "pin-1", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-1"}
		inst := &models.EntityInstance{ID: "inst-1", EntityTypeID: "et-1", CatalogID: "cat-1", Name: "s1", Version: 1, CreatedAt: now, UpdatedAt: now}
		catalogRepo.On("GetByName", ctx(), "test-catalog").Return(catalog, nil)
		cvRepo.On("GetByID", ctx(), "cv-1").Return(cv, nil)
		pinRepo.On("ListByCatalogVersion", ctx(), "cv-1").Return([]*models.CatalogVersionPin{pin}, nil)
		etvRepo.On("GetByID", ctx(), "etv-1").Return(etv, nil)
		etRepo.On("GetByID", ctx(), "et-1").Return(et, nil)
		attrRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Attribute{}, nil)
		assocRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Association{}, nil)
		instRepo.On("ListByCatalog", ctx(), "cat-1").Return([]*models.EntityInstance{inst}, nil)
		iavRepo.On("GetValuesForVersion", ctx(), "inst-1", 1).Return(nil, someErr)
		_, err := svc.ExportCatalog(context.Background(), "test-catalog", nil, "")
		assert.Error(t, err)
	})

	t.Run("instance_link_error", func(t *testing.T) {
		svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, _, _, instRepo, iavRepo, linkRepo := newExportService()
		now := time.Now()
		catalog := &models.Catalog{ID: "cat-1", Name: "test-catalog", CatalogVersionID: "cv-1", CreatedAt: now, UpdatedAt: now}
		cv := &models.CatalogVersion{ID: "cv-1", VersionLabel: "v1.0"}
		et := &models.EntityType{ID: "et-1", Name: "server"}
		etv := &models.EntityTypeVersion{ID: "etv-1", EntityTypeID: "et-1", Version: 1}
		pin := &models.CatalogVersionPin{ID: "pin-1", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-1"}
		inst := &models.EntityInstance{ID: "inst-1", EntityTypeID: "et-1", CatalogID: "cat-1", Name: "s1", Version: 1, CreatedAt: now, UpdatedAt: now}
		catalogRepo.On("GetByName", ctx(), "test-catalog").Return(catalog, nil)
		cvRepo.On("GetByID", ctx(), "cv-1").Return(cv, nil)
		pinRepo.On("ListByCatalogVersion", ctx(), "cv-1").Return([]*models.CatalogVersionPin{pin}, nil)
		etvRepo.On("GetByID", ctx(), "etv-1").Return(etv, nil)
		etRepo.On("GetByID", ctx(), "et-1").Return(et, nil)
		attrRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Attribute{}, nil)
		assocRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Association{}, nil)
		instRepo.On("ListByCatalog", ctx(), "cat-1").Return([]*models.EntityInstance{inst}, nil)
		iavRepo.On("GetValuesForVersion", ctx(), "inst-1", 1).Return([]*models.InstanceAttributeValue{}, nil)
		linkRepo.On("GetForwardRefs", ctx(), "inst-1").Return(nil, someErr)
		_, err := svc.ExportCatalog(context.Background(), "test-catalog", nil, "")
		assert.Error(t, err)
	})
}

// T-30.14b: ExportInstance MarshalJSON error paths
func TestT30_14b_MarshalJSON_ErrorPaths(t *testing.T) {
	// MarshalJSON with nil children map should still work
	ei := ExportInstance{
		EntityType:  "server",
		Name:        "s1",
		Description: "test",
		Attributes:  nil,
		Children:    nil,
	}
	b, err := json.Marshal(ei)
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	assert.Equal(t, "server", m["entity_type"])
}

// T-30.19: Export with no matching containment assoc name — fallback to "children"
func TestT30_19_ExportFallbackChildrenAssocName(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, _, _, instRepo, iavRepo, linkRepo := newExportService()

	now := time.Now()
	catalog := &models.Catalog{ID: "cat-1", Name: "test-catalog", CatalogVersionID: "cv-1", ValidationStatus: models.ValidationStatusValid, CreatedAt: now, UpdatedAt: now}
	cv := &models.CatalogVersion{ID: "cv-1", VersionLabel: "v1.0"}

	parentET := &models.EntityType{ID: "et-parent", Name: "parent"}
	parentETV := &models.EntityTypeVersion{ID: "etv-parent", EntityTypeID: "et-parent", Version: 1}
	childET := &models.EntityType{ID: "et-child", Name: "child"}
	childETV := &models.EntityTypeVersion{ID: "etv-child", EntityTypeID: "et-child", Version: 1}

	parentPin := &models.CatalogVersionPin{ID: "pin-1", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-parent"}
	childPin := &models.CatalogVersionPin{ID: "pin-2", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-child"}

	parentInst := &models.EntityInstance{ID: "inst-parent", EntityTypeID: "et-parent", CatalogID: "cat-1", Name: "p1", Version: 1, CreatedAt: now, UpdatedAt: now}
	childInst := &models.EntityInstance{ID: "inst-child", EntityTypeID: "et-child", CatalogID: "cat-1", ParentInstanceID: "inst-parent", Name: "c1", Version: 1, CreatedAt: now, UpdatedAt: now}

	catalogRepo.On("GetByName", ctx(), "test-catalog").Return(catalog, nil)
	cvRepo.On("GetByID", ctx(), "cv-1").Return(cv, nil)
	pinRepo.On("ListByCatalogVersion", ctx(), "cv-1").Return([]*models.CatalogVersionPin{parentPin, childPin}, nil)
	etvRepo.On("GetByID", ctx(), "etv-parent").Return(parentETV, nil)
	etvRepo.On("GetByID", ctx(), "etv-child").Return(childETV, nil)
	etRepo.On("GetByID", ctx(), "et-parent").Return(parentET, nil)
	etRepo.On("GetByID", ctx(), "et-child").Return(childET, nil)
	attrRepo.On("ListByVersion", ctx(), "etv-parent").Return([]*models.Attribute{}, nil)
	attrRepo.On("ListByVersion", ctx(), "etv-child").Return([]*models.Attribute{}, nil)
	// No containment association defined → fallback to "children"
	assocRepo.On("ListByVersion", ctx(), "etv-parent").Return([]*models.Association{}, nil)
	assocRepo.On("ListByVersion", ctx(), "etv-child").Return([]*models.Association{}, nil)
	instRepo.On("ListByCatalog", ctx(), "cat-1").Return([]*models.EntityInstance{parentInst, childInst}, nil)
	iavRepo.On("GetValuesForVersion", ctx(), "inst-parent", 1).Return([]*models.InstanceAttributeValue{}, nil)
	iavRepo.On("GetValuesForVersion", ctx(), "inst-child", 1).Return([]*models.InstanceAttributeValue{}, nil)
	linkRepo.On("GetForwardRefs", ctx(), "inst-parent").Return([]*models.AssociationLink{}, nil)
	linkRepo.On("GetForwardRefs", ctx(), "inst-child").Return([]*models.AssociationLink{}, nil)

	result, err := svc.ExportCatalog(context.Background(), "test-catalog", nil, "")
	require.NoError(t, err)

	require.Len(t, result.Instances, 1)
	assert.Equal(t, "p1", result.Instances[0].Name)
	// Child should be under the "children" fallback key
	require.Len(t, result.Instances[0].Children["children"], 1)
	assert.Equal(t, "c1", result.Instances[0].Children["children"][0].Name)
}

// T-30.15b: Export child build error propagates
func TestT30_15b_ExportChildBuildError(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, _, _, instRepo, iavRepo, linkRepo := newExportService()

	now := time.Now()
	catalog := &models.Catalog{ID: "cat-1", Name: "test-catalog", CatalogVersionID: "cv-1", ValidationStatus: models.ValidationStatusValid, CreatedAt: now, UpdatedAt: now}
	cv := &models.CatalogVersion{ID: "cv-1", VersionLabel: "v1.0"}

	parentET := &models.EntityType{ID: "et-parent", Name: "parent"}
	parentETV := &models.EntityTypeVersion{ID: "etv-parent", EntityTypeID: "et-parent", Version: 1}
	childET := &models.EntityType{ID: "et-child", Name: "child"}
	childETV := &models.EntityTypeVersion{ID: "etv-child", EntityTypeID: "et-child", Version: 1}

	parentPin := &models.CatalogVersionPin{ID: "pin-1", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-parent"}
	childPin := &models.CatalogVersionPin{ID: "pin-2", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-child"}

	parentInst := &models.EntityInstance{ID: "inst-parent", EntityTypeID: "et-parent", CatalogID: "cat-1", Name: "p1", Version: 1, CreatedAt: now, UpdatedAt: now}
	childInst := &models.EntityInstance{ID: "inst-child", EntityTypeID: "et-child", CatalogID: "cat-1", ParentInstanceID: "inst-parent", Name: "c1", Version: 1, CreatedAt: now, UpdatedAt: now}

	catalogRepo.On("GetByName", ctx(), "test-catalog").Return(catalog, nil)
	cvRepo.On("GetByID", ctx(), "cv-1").Return(cv, nil)
	pinRepo.On("ListByCatalogVersion", ctx(), "cv-1").Return([]*models.CatalogVersionPin{parentPin, childPin}, nil)
	etvRepo.On("GetByID", ctx(), "etv-parent").Return(parentETV, nil)
	etvRepo.On("GetByID", ctx(), "etv-child").Return(childETV, nil)
	etRepo.On("GetByID", ctx(), "et-parent").Return(parentET, nil)
	etRepo.On("GetByID", ctx(), "et-child").Return(childET, nil)
	attrRepo.On("ListByVersion", ctx(), "etv-parent").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", ctx(), "etv-parent").Return([]*models.Association{}, nil)
	assocRepo.On("ListByVersion", ctx(), "etv-child").Return([]*models.Association{}, nil)
	// First call to etv-child attrRepo in entity type loop → success
	attrRepo.On("ListByVersion", ctx(), "etv-child").Return([]*models.Attribute{}, nil).Once()
	instRepo.On("ListByCatalog", ctx(), "cat-1").Return([]*models.EntityInstance{parentInst, childInst}, nil)
	iavRepo.On("GetValuesForVersion", ctx(), "inst-parent", 1).Return([]*models.InstanceAttributeValue{}, nil)
	linkRepo.On("GetForwardRefs", ctx(), "inst-parent").Return([]*models.AssociationLink{}, nil)
	// Second call in buildExportInstances → child attr list fails
	attrRepo.On("ListByVersion", ctx(), "etv-child").Return(nil, domainerrors.NewNotFound("x", "y")).Once()
	_, err := svc.ExportCatalog(context.Background(), "test-catalog", nil, "")
	assert.Error(t, err)
}

// T-30.26: Export with attribute that has no stored value — skipped
func TestT30_26_ExportAttrNoValue(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, instRepo, iavRepo, linkRepo := newExportService()

	now := time.Now()
	catalog := &models.Catalog{ID: "cat-1", Name: "test-catalog", CatalogVersionID: "cv-1", ValidationStatus: models.ValidationStatusValid, CreatedAt: now, UpdatedAt: now}
	cv := &models.CatalogVersion{ID: "cv-1", VersionLabel: "v1.0"}
	et := &models.EntityType{ID: "et-1", Name: "server"}
	etv := &models.EntityTypeVersion{ID: "etv-1", EntityTypeID: "et-1", Version: 1}
	pin := &models.CatalogVersionPin{ID: "pin-1", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-1"}
	strTD := &models.TypeDefinition{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}
	strTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}

	attrs := []*models.Attribute{
		{ID: "attr-1", EntityTypeVersionID: "etv-1", Name: "endpoint", TypeDefinitionVersionID: "tdv-str", Required: true, Ordinal: 0},
		{ID: "attr-2", EntityTypeVersionID: "etv-1", Name: "label", TypeDefinitionVersionID: "tdv-str", Required: false, Ordinal: 1},
	}

	inst := &models.EntityInstance{ID: "inst-1", EntityTypeID: "et-1", CatalogID: "cat-1", Name: "s1", Version: 1, CreatedAt: now, UpdatedAt: now}
	// Only attr-1 has a value, attr-2 does NOT → should be skipped
	iavs := []*models.InstanceAttributeValue{
		{ID: "iav-1", InstanceID: "inst-1", InstanceVersion: 1, AttributeID: "attr-1", ValueString: "https://example.com"},
	}

	catalogRepo.On("GetByName", ctx(), "test-catalog").Return(catalog, nil)
	cvRepo.On("GetByID", ctx(), "cv-1").Return(cv, nil)
	pinRepo.On("ListByCatalogVersion", ctx(), "cv-1").Return([]*models.CatalogVersionPin{pin}, nil)
	etvRepo.On("GetByID", ctx(), "etv-1").Return(etv, nil)
	etRepo.On("GetByID", ctx(), "et-1").Return(et, nil)
	attrRepo.On("ListByVersion", ctx(), "etv-1").Return(attrs, nil)
	tdvRepo.On("GetByID", ctx(), "tdv-str").Return(strTDV, nil)
	tdRepo.On("GetByID", ctx(), "td-str").Return(strTD, nil)
	assocRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Association{}, nil)
	instRepo.On("ListByCatalog", ctx(), "cat-1").Return([]*models.EntityInstance{inst}, nil)
	iavRepo.On("GetValuesForVersion", ctx(), "inst-1", 1).Return(iavs, nil)
	linkRepo.On("GetForwardRefs", ctx(), "inst-1").Return([]*models.AssociationLink{}, nil)

	result, err := svc.ExportCatalog(context.Background(), "test-catalog", nil, "")
	require.NoError(t, err)

	require.Len(t, result.Instances, 1)
	assert.Equal(t, "https://example.com", result.Instances[0].Attributes["endpoint"])
	_, hasLabel := result.Instances[0].Attributes["label"]
	assert.False(t, hasLabel, "attribute with no stored value should not appear in export")
}

// T-30.27: Export with ResolveBaseTypes error in instance building
func TestT30_27_ExportResolveBaseTypesError(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, tdRepo, tdvRepo, instRepo, iavRepo, _ := newExportService()

	now := time.Now()
	catalog := &models.Catalog{ID: "cat-1", Name: "test-catalog", CatalogVersionID: "cv-1", ValidationStatus: models.ValidationStatusValid, CreatedAt: now, UpdatedAt: now}
	cv := &models.CatalogVersion{ID: "cv-1", VersionLabel: "v1.0"}
	et := &models.EntityType{ID: "et-1", Name: "server"}
	etv := &models.EntityTypeVersion{ID: "etv-1", EntityTypeID: "et-1", Version: 1}
	pin := &models.CatalogVersionPin{ID: "pin-1", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-1"}
	strTD := &models.TypeDefinition{ID: "td-str", Name: "string", BaseType: models.BaseTypeString, System: true}
	strTDV := &models.TypeDefinitionVersion{ID: "tdv-str", TypeDefinitionID: "td-str", VersionNumber: 1}

	attrs := []*models.Attribute{
		{ID: "attr-1", EntityTypeVersionID: "etv-1", Name: "endpoint", TypeDefinitionVersionID: "tdv-str"},
	}

	inst := &models.EntityInstance{ID: "inst-1", EntityTypeID: "et-1", CatalogID: "cat-1", Name: "s1", Version: 1, CreatedAt: now, UpdatedAt: now}

	catalogRepo.On("GetByName", ctx(), "test-catalog").Return(catalog, nil)
	cvRepo.On("GetByID", ctx(), "cv-1").Return(cv, nil)
	pinRepo.On("ListByCatalogVersion", ctx(), "cv-1").Return([]*models.CatalogVersionPin{pin}, nil)
	etvRepo.On("GetByID", ctx(), "etv-1").Return(etv, nil)
	etRepo.On("GetByID", ctx(), "et-1").Return(et, nil)
	// First call in entity type loop → success
	attrRepo.On("ListByVersion", ctx(), "etv-1").Return(attrs, nil)
	tdvRepo.On("GetByID", ctx(), "tdv-str").Return(strTDV, nil).Once()
	tdRepo.On("GetByID", ctx(), "td-str").Return(strTD, nil).Once()
	assocRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Association{}, nil)
	instRepo.On("ListByCatalog", ctx(), "cat-1").Return([]*models.EntityInstance{inst}, nil)
	iavRepo.On("GetValuesForVersion", ctx(), "inst-1", 1).Return([]*models.InstanceAttributeValue{}, nil)
	// Second call in buildExportInstances → ResolveBaseTypes error
	tdvRepo.On("GetByID", ctx(), "tdv-str").Return(nil, domainerrors.NewNotFound("x", "y")).Once()
	_, err := svc.ExportCatalog(context.Background(), "test-catalog", nil, "")
	assert.Error(t, err)
}

// T-30.28: Export with entity filter — child of filtered-out type returns nil
func TestT30_28_ExportChildFilteredOutReturnsNil(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, _, _, instRepo, iavRepo, linkRepo := newExportService()

	now := time.Now()
	catalog := &models.Catalog{ID: "cat-1", Name: "test-catalog", CatalogVersionID: "cv-1", ValidationStatus: models.ValidationStatusValid, CreatedAt: now, UpdatedAt: now}
	cv := &models.CatalogVersion{ID: "cv-1", VersionLabel: "v1.0"}

	serverET := &models.EntityType{ID: "et-server", Name: "server"}
	serverETV := &models.EntityTypeVersion{ID: "etv-server", EntityTypeID: "et-server", Version: 1}
	toolET := &models.EntityType{ID: "et-tool", Name: "tool"}
	toolETV := &models.EntityTypeVersion{ID: "etv-tool", EntityTypeID: "et-tool", Version: 1}

	serverPin := &models.CatalogVersionPin{ID: "pin-1", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-server"}
	toolPin := &models.CatalogVersionPin{ID: "pin-2", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-tool"}

	containAssoc := &models.Association{ID: "assoc-1", EntityTypeVersionID: "etv-server", Name: "tools", TargetEntityTypeID: "et-tool", Type: models.AssociationTypeContainment}

	serverInst := &models.EntityInstance{ID: "inst-server", EntityTypeID: "et-server", CatalogID: "cat-1", Name: "s1", Version: 1, CreatedAt: now, UpdatedAt: now}
	toolInst := &models.EntityInstance{ID: "inst-tool", EntityTypeID: "et-tool", CatalogID: "cat-1", ParentInstanceID: "inst-server", Name: "t1", Version: 1, CreatedAt: now, UpdatedAt: now}

	catalogRepo.On("GetByName", ctx(), "test-catalog").Return(catalog, nil)
	cvRepo.On("GetByID", ctx(), "cv-1").Return(cv, nil)
	pinRepo.On("ListByCatalogVersion", ctx(), "cv-1").Return([]*models.CatalogVersionPin{serverPin, toolPin}, nil)
	etvRepo.On("GetByID", ctx(), "etv-server").Return(serverETV, nil)
	etvRepo.On("GetByID", ctx(), "etv-tool").Return(toolETV, nil)
	etRepo.On("GetByID", ctx(), "et-server").Return(serverET, nil)
	etRepo.On("GetByID", ctx(), "et-tool").Return(toolET, nil)
	attrRepo.On("ListByVersion", ctx(), "etv-server").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", ctx(), "etv-server").Return([]*models.Association{containAssoc}, nil)
	assocRepo.On("ListByVersion", ctx(), "etv-tool").Return([]*models.Association{}, nil)
	instRepo.On("ListByCatalog", ctx(), "cat-1").Return([]*models.EntityInstance{serverInst, toolInst}, nil)
	iavRepo.On("GetValuesForVersion", ctx(), "inst-server", 1).Return([]*models.InstanceAttributeValue{}, nil)
	linkRepo.On("GetForwardRefs", ctx(), "inst-server").Return([]*models.AssociationLink{}, nil)

	// Filter to only "server" → tool is excluded → child returns nil → continue
	result, err := svc.ExportCatalog(context.Background(), "test-catalog", []string{"server"}, "")
	require.NoError(t, err)

	require.Len(t, result.Instances, 1)
	assert.Equal(t, "s1", result.Instances[0].Name)
	assert.Empty(t, result.Instances[0].Children, "child of filtered-out type should not appear")
}

// T-30.29: Export — pinned ETV not found in cache loop is now a hard error
// (previously silently skipped, fixed per Copilot review PR#18)
func TestT30_29_ExportNilETVInCacheLoop(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, _, _, instRepo, _, _ := newExportService()

	now := time.Now()
	catalog := &models.Catalog{ID: "cat-1", Name: "test-catalog", CatalogVersionID: "cv-1", ValidationStatus: models.ValidationStatusValid, CreatedAt: now, UpdatedAt: now}
	cv := &models.CatalogVersion{ID: "cv-1", VersionLabel: "v1.0"}

	et := &models.EntityType{ID: "et-1", Name: "server"}
	etv := &models.EntityTypeVersion{ID: "etv-1", EntityTypeID: "et-1", Version: 1, Description: "server"}
	badET := &models.EntityType{ID: "et-bad", Name: "bad-type"}
	badETV := &models.EntityTypeVersion{ID: "etv-bad", EntityTypeID: "et-bad", Version: 1}
	goodPin := &models.CatalogVersionPin{ID: "pin-1", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-1"}
	badPin := &models.CatalogVersionPin{ID: "pin-bad", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-bad"}

	catalogRepo.On("GetByName", ctx(), "test-catalog").Return(catalog, nil)
	cvRepo.On("GetByID", ctx(), "cv-1").Return(cv, nil)
	pinRepo.On("ListByCatalogVersion", ctx(), "cv-1").Return([]*models.CatalogVersionPin{goodPin, badPin}, nil)

	etvRepo.On("GetByID", ctx(), "etv-1").Return(etv, nil)
	etRepo.On("GetByID", ctx(), "et-1").Return(et, nil)
	attrRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Association{}, nil)

	// Bad pin resolves in first loop (filtered out), then fails in cache loop
	etvRepo.On("GetByID", ctx(), "etv-bad").Return(badETV, nil).Once()
	etRepo.On("GetByID", ctx(), "et-bad").Return(badET, nil).Once()
	etvRepo.On("GetByID", ctx(), "etv-bad").Return(nil, domainerrors.NewNotFound("ETV", "etv-bad"))

	instRepo.On("ListByCatalog", ctx(), "cat-1").Return([]*models.EntityInstance{}, nil)

	// Even with entity filter excluding "bad-type", the cache loop now propagates
	// the error — a pinned ETV that can't be found is a data integrity issue
	result, err := svc.ExportCatalog(context.Background(), "test-catalog", []string{"server"}, "")
	require.Error(t, err)
	assert.Nil(t, result)
}

// T-30.30: ExportInstance MarshalJSON — json.Marshal error path (unmarshalable attribute value)
func TestT30_30_MarshalJSONError(t *testing.T) {
	ei := ExportInstance{
		EntityType:  "server",
		Name:        "s1",
		Description: "test",
		Attributes:  map[string]any{"bad": math.Inf(1)}, // json.Marshal cannot marshal Inf
		Children:    make(map[string][]*ExportInstance),
	}
	_, err := json.Marshal(ei)
	assert.Error(t, err)
}

// T-30.31: Export logs warning when link association not found (both in map and DB)
func TestT30_31_ExportDroppedLinkWarning(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, _, _, instRepo, iavRepo, linkRepo := newExportService()

	now := time.Now()
	catalog := &models.Catalog{ID: "cat-1", Name: "test", CatalogVersionID: "cv-1", ValidationStatus: models.ValidationStatusValid, CreatedAt: now, UpdatedAt: now}
	cv := &models.CatalogVersion{ID: "cv-1", VersionLabel: "v1"}
	et := &models.EntityType{ID: "et-1", Name: "server"}
	etv := &models.EntityTypeVersion{ID: "etv-1", EntityTypeID: "et-1", Version: 1}
	pin := &models.CatalogVersionPin{ID: "pin-1", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-1"}
	inst := &models.EntityInstance{ID: "inst-1", EntityTypeID: "et-1", CatalogID: "cat-1", Name: "s1", Version: 1, CreatedAt: now, UpdatedAt: now}

	catalogRepo.On("GetByName", ctx(), "test").Return(catalog, nil)
	cvRepo.On("GetByID", ctx(), "cv-1").Return(cv, nil)
	pinRepo.On("ListByCatalogVersion", ctx(), "cv-1").Return([]*models.CatalogVersionPin{pin}, nil)
	etvRepo.On("GetByID", ctx(), "etv-1").Return(etv, nil)
	etRepo.On("GetByID", ctx(), "et-1").Return(et, nil)
	attrRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Association{}, nil)
	instRepo.On("ListByCatalog", ctx(), "cat-1").Return([]*models.EntityInstance{inst}, nil)
	iavRepo.On("GetValuesForVersion", ctx(), "inst-1", 1).Return([]*models.InstanceAttributeValue{}, nil)
	// Link references an association ID not in assocByID AND not in DB
	linkRepo.On("GetForwardRefs", ctx(), "inst-1").Return([]*models.AssociationLink{
		{ID: "link-1", AssociationID: "orphan-assoc-id", SourceInstanceID: "inst-1", TargetInstanceID: "inst-1"},
	}, nil)
	assocRepo.On("GetByID", ctx(), "orphan-assoc-id").Return(nil, assert.AnError)

	result, err := svc.ExportCatalog(context.Background(), "test", nil, "")
	require.NoError(t, err)
	// Link should be dropped (not included) — no crash
	assert.Empty(t, result.Instances[0].Links, "orphan link should be dropped gracefully")
}

func TestT30_30_BuildParentPathCycleDetection(t *testing.T) {
	// Create a cycle: A → B → A
	instances := map[string]*models.EntityInstance{
		"a": {ID: "a", Name: "inst-a", ParentInstanceID: "b"},
		"b": {ID: "b", Name: "inst-b", ParentInstanceID: "a"},
	}
	path := buildParentPath(instances["a"], instances)
	// Should terminate (not infinite loop) and return a path
	assert.NotNil(t, path)
	assert.LessOrEqual(t, len(path), 2, "cycle should be detected and path should be bounded")
}

// T-30.32: Export returns error when cache loop ETV lookup fails (not silently skipped)
// The entity-type building loop (first pass) succeeds, but the cache-building loop
// (second pass over same pins) fails — error must be propagated, not swallowed.
func TestT30_32_ExportCacheLoopETVError(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, _, _, instRepo, _, _ := newExportService()

	now := time.Now()
	catalog := &models.Catalog{ID: "cat-1", Name: "test", CatalogVersionID: "cv-1", ValidationStatus: models.ValidationStatusValid, CreatedAt: now, UpdatedAt: now}
	cv := &models.CatalogVersion{ID: "cv-1", VersionLabel: "v1"}
	et := &models.EntityType{ID: "et-1", Name: "server"}
	etv := &models.EntityTypeVersion{ID: "etv-1", EntityTypeID: "et-1", Version: 1, Description: "desc"}
	pin := &models.CatalogVersionPin{ID: "pin-1", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-1"}

	catalogRepo.On("GetByName", ctx(), "test").Return(catalog, nil)
	cvRepo.On("GetByID", ctx(), "cv-1").Return(cv, nil)
	pinRepo.On("ListByCatalogVersion", ctx(), "cv-1").Return([]*models.CatalogVersionPin{pin}, nil)

	// First pass (entity-type building loop): succeeds
	etvRepo.On("GetByID", ctx(), "etv-1").Return(etv, nil).Once()
	etRepo.On("GetByID", ctx(), "et-1").Return(et, nil).Once()
	attrRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Association{}, nil)

	instRepo.On("ListByCatalog", ctx(), "cat-1").Return([]*models.EntityInstance{}, nil)

	// Subsequent passes (cache loops): ETV lookup fails — transient DB error
	// Currently silently swallowed; should propagate
	etvRepo.On("GetByID", ctx(), "etv-1").Return(nil, fmt.Errorf("connection refused"))

	result, err := svc.ExportCatalog(context.Background(), "test", nil, "")
	require.Error(t, err, "cache loop should propagate ETV lookup error, not silently skip")
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "connection refused")
}

// T-30.33: Export returns error when cache loop ET lookup fails
func TestT30_33_ExportCacheLoopETError(t *testing.T) {
	svc, catalogRepo, cvRepo, pinRepo, etRepo, etvRepo, attrRepo, assocRepo, _, _, instRepo, _, _ := newExportService()

	now := time.Now()
	catalog := &models.Catalog{ID: "cat-1", Name: "test", CatalogVersionID: "cv-1", ValidationStatus: models.ValidationStatusValid, CreatedAt: now, UpdatedAt: now}
	cv := &models.CatalogVersion{ID: "cv-1", VersionLabel: "v1"}
	et := &models.EntityType{ID: "et-1", Name: "server"}
	etv := &models.EntityTypeVersion{ID: "etv-1", EntityTypeID: "et-1", Version: 1, Description: "desc"}
	pin := &models.CatalogVersionPin{ID: "pin-1", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-1"}

	catalogRepo.On("GetByName", ctx(), "test").Return(catalog, nil)
	cvRepo.On("GetByID", ctx(), "cv-1").Return(cv, nil)
	pinRepo.On("ListByCatalogVersion", ctx(), "cv-1").Return([]*models.CatalogVersionPin{pin}, nil)

	// First pass (entity-type building loop): succeeds
	etvRepo.On("GetByID", ctx(), "etv-1").Return(etv, nil).Once()
	etRepo.On("GetByID", ctx(), "et-1").Return(et, nil).Once()
	attrRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", ctx(), "etv-1").Return([]*models.Association{}, nil)

	instRepo.On("ListByCatalog", ctx(), "cat-1").Return([]*models.EntityInstance{}, nil)

	// Subsequent passes (cache loops): ETV succeeds but ET lookup fails
	etvRepo.On("GetByID", ctx(), "etv-1").Return(etv, nil)
	etRepo.On("GetByID", ctx(), "et-1").Return(nil, fmt.Errorf("database locked"))

	result, err := svc.ExportCatalog(context.Background(), "test", nil, "")
	require.Error(t, err, "cache loop should propagate ET lookup error, not silently skip")
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "database locked")
}

func ctx() context.Context {
	return context.Background()
}
