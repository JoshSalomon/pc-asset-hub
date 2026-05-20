package export_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/infrastructure/gorm/repository"
	"github.com/project-catalyst/pc-asset-hub/internal/infrastructure/gorm/testutil"
	"github.com/project-catalyst/pc-asset-hub/internal/service/operational/export"
)

func id() string { return uuid.Must(uuid.NewV7()).String() }

type integrationSetup struct {
	svc      *export.ExportBindingService
	registry *export.ExporterRegistry
	catID    string
	cvID     string
}

func setupIntegration(t *testing.T) *integrationSetup {
	t.Helper()
	db := testutil.NewTestDB(t)
	ctx := context.Background()

	catalogRepo := repository.NewCatalogGormRepo(db)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	pinRepo := repository.NewCatalogVersionPinGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	etRepo := repository.NewEntityTypeGormRepo(db)
	attrRepo := repository.NewAttributeGormRepo(db)
	assocRepo := repository.NewAssociationGormRepo(db)
	bindingRepo := repository.NewExportBindingGormRepo(db)

	registry := export.NewExporterRegistry()
	registry.Register(export.NewMCPGatewayExporter())

	// Create entity types
	serverETID := id()
	toolETID := id()
	vsETID := id()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: serverETID, Name: "mcp-server", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: toolETID, Name: "mcp-tool", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: vsETID, Name: "virtual-server", CreatedAt: time.Now(), UpdatedAt: time.Now()}))

	// Create entity type versions
	serverETVID := id()
	toolETVID := id()
	vsETVID := id()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: serverETVID, EntityTypeID: serverETID, Version: 1, CreatedAt: time.Now()}))
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: toolETVID, EntityTypeID: toolETID, Version: 1, CreatedAt: time.Now()}))
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: vsETVID, EntityTypeID: vsETID, Version: 1, CreatedAt: time.Now()}))

	// Create attributes on server type — including required "route_name"
	require.NoError(t, attrRepo.Create(ctx, &models.Attribute{
		ID: id(), EntityTypeVersionID: serverETVID, Name: "route_name",
		Ordinal: 1,}))
	require.NoError(t, attrRepo.Create(ctx, &models.Attribute{
		ID: id(), EntityTypeVersionID: serverETVID, Name: "mcp_path",
		Ordinal: 2,}))

	// Create containment association: server → tool
	require.NoError(t, assocRepo.Create(ctx, &models.Association{
		ID: id(), EntityTypeVersionID: serverETVID, Name: "tools",
		Type: models.AssociationTypeContainment, TargetEntityTypeID: toolETID,}))

	// Create directional association: virtual-server → tool
	require.NoError(t, assocRepo.Create(ctx, &models.Association{
		ID: id(), EntityTypeVersionID: vsETVID, Name: "served-tools",
		Type: models.AssociationTypeDirectional, TargetEntityTypeID: toolETID,}))

	// Create catalog version + pins
	cvID := id()
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{ID: cvID, VersionLabel: "v1.0", CreatedAt: time.Now()}))
	require.NoError(t, pinRepo.Create(ctx, &models.CatalogVersionPin{ID: id(), CatalogVersionID: cvID, EntityTypeVersionID: serverETVID}))
	require.NoError(t, pinRepo.Create(ctx, &models.CatalogVersionPin{ID: id(), CatalogVersionID: cvID, EntityTypeVersionID: toolETVID}))
	require.NoError(t, pinRepo.Create(ctx, &models.CatalogVersionPin{ID: id(), CatalogVersionID: cvID, EntityTypeVersionID: vsETVID}))

	// Create catalog
	catID := id()
	require.NoError(t, catalogRepo.Create(ctx, &models.Catalog{
		ID: catID, Name: "integration-test", CatalogVersionID: cvID,
		ValidationStatus: "draft", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	svc := export.NewExportBindingService(
		bindingRepo, catalogRepo, registry,
		cvRepo, pinRepo, etvRepo, etRepo, attrRepo, assocRepo,
	)

	return &integrationSetup{svc: svc, registry: registry, catID: catID, cvID: cvID}
}

// T-34.19: BuildSchemaInfo integration — resolves real entity types, attributes, associations from DB
func TestT34_19_BuildSchemaInfo_Integration(t *testing.T) {
	s := setupIntegration(t)
	ctx := context.Background()

	schema, err := s.svc.BuildSchemaInfo(ctx, s.cvID)
	require.NoError(t, err)
	require.Len(t, schema.EntityTypes, 3)

	var serverType *export.SchemaEntityType
	for i := range schema.EntityTypes {
		if schema.EntityTypes[i].Name == "mcp-server" {
			serverType = &schema.EntityTypes[i]
		}
	}
	require.NotNil(t, serverType, "mcp-server entity type should be in schema")
	assert.Contains(t, serverType.Attributes, "route_name")
	assert.Contains(t, serverType.Attributes, "mcp_path")
	require.Len(t, serverType.Associations, 1)
	assert.Equal(t, "tools", serverType.Associations[0].Name)
	assert.Equal(t, "containment", serverType.Associations[0].Type)
	assert.Equal(t, "mcp-tool", serverType.Associations[0].TargetEntityType)
}

// T-34.72: ValidateSchema integration — positive: valid schema passes; negative: missing attribute fails
func TestT34_72_ValidateSchema_Integration(t *testing.T) {
	s := setupIntegration(t)
	ctx := context.Background()

	// Positive: create binding with valid params should succeed
	binding, err := s.svc.Create(ctx, "integration-test", "mcp-gateway", map[string]string{
		"server_type":         "mcp-server",
		"tool_type":           "mcp-tool",
		"virtual_server_type": "virtual-server",
	})
	require.NoError(t, err)
	assert.Equal(t, "mcp-gateway", binding.ExporterName)
	assert.Equal(t, "mcp-server", binding.Parameters["server_type"])

	// Negative: entity type not pinned
	_, err = s.svc.Create(ctx, "integration-test", "mcp-gateway", map[string]string{
		"server_type":         "nonexistent-type",
		"tool_type":           "mcp-tool",
		"virtual_server_type": "virtual-server",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent-type")
}

// T-34.72 supplement: ValidateSchema fails when server type lacks route_name attribute
func TestT34_72_ValidateSchema_MissingAttribute(t *testing.T) {
	db := testutil.NewTestDB(t)
	ctx := context.Background()

	catalogRepo := repository.NewCatalogGormRepo(db)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	pinRepo := repository.NewCatalogVersionPinGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	etRepo := repository.NewEntityTypeGormRepo(db)
	attrRepo := repository.NewAttributeGormRepo(db)
	assocRepo := repository.NewAssociationGormRepo(db)
	bindingRepo := repository.NewExportBindingGormRepo(db)

	registry := export.NewExporterRegistry()
	registry.Register(export.NewMCPGatewayExporter())

	// Create entity type WITHOUT route_name attribute
	serverETID := id()
	toolETID := id()
	vsETID := id()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: serverETID, Name: "bad-server", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: toolETID, Name: "bad-tool", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: vsETID, Name: "bad-vs", CreatedAt: time.Now(), UpdatedAt: time.Now()}))

	serverETVID := id()
	toolETVID := id()
	vsETVID := id()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: serverETVID, EntityTypeID: serverETID, Version: 1, CreatedAt: time.Now()}))
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: toolETVID, EntityTypeID: toolETID, Version: 1, CreatedAt: time.Now()}))
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: vsETVID, EntityTypeID: vsETID, Version: 1, CreatedAt: time.Now()}))

	// Only add a non-route_name attribute
	require.NoError(t, attrRepo.Create(ctx, &models.Attribute{
		ID: id(), EntityTypeVersionID: serverETVID, Name: "hostname",
		Ordinal: 1,}))

	// Add containment
	require.NoError(t, assocRepo.Create(ctx, &models.Association{
		ID: id(), EntityTypeVersionID: serverETVID, Name: "tools",
		Type: models.AssociationTypeContainment, TargetEntityTypeID: toolETID,}))

	// Add VS → tool association
	require.NoError(t, assocRepo.Create(ctx, &models.Association{
		ID: id(), EntityTypeVersionID: vsETVID, Name: "served-tools",
		Type: models.AssociationTypeDirectional, TargetEntityTypeID: toolETID,}))

	cvID := id()
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{ID: cvID, VersionLabel: "v1.0", CreatedAt: time.Now()}))
	require.NoError(t, pinRepo.Create(ctx, &models.CatalogVersionPin{ID: id(), CatalogVersionID: cvID, EntityTypeVersionID: serverETVID}))
	require.NoError(t, pinRepo.Create(ctx, &models.CatalogVersionPin{ID: id(), CatalogVersionID: cvID, EntityTypeVersionID: toolETVID}))
	require.NoError(t, pinRepo.Create(ctx, &models.CatalogVersionPin{ID: id(), CatalogVersionID: cvID, EntityTypeVersionID: vsETVID}))

	catID := id()
	require.NoError(t, catalogRepo.Create(ctx, &models.Catalog{
		ID: catID, Name: "bad-schema", CatalogVersionID: cvID,
		ValidationStatus: "draft", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	svc := export.NewExportBindingService(
		bindingRepo, catalogRepo, registry,
		cvRepo, pinRepo, etvRepo, etRepo, attrRepo, assocRepo,
	)

	// ValidateSchema should fail because bad-server lacks route_name
	_, err := svc.Create(ctx, "bad-schema", "mcp-gateway", map[string]string{
		"server_type":         "bad-server",
		"tool_type":           "bad-tool",
		"virtual_server_type": "bad-vs",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "route_name")
}
