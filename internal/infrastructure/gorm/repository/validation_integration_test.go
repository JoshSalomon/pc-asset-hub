package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/infrastructure/gorm/repository"
	"github.com/project-catalyst/pc-asset-hub/internal/infrastructure/gorm/testutil"
	"github.com/project-catalyst/pc-asset-hub/internal/service/operational"
)

// setupValidationIntegration creates all repos and a CatalogValidationService backed by real SQLite.
// Returns the service, repos, and pre-created IDs for a catalog with one entity type (Server)
// that has one required string attribute (hostname) and one optional attribute (description).
func setupValidationIntegration(t *testing.T) (
	*operational.CatalogValidationService,
	context.Context,
	string, // catalogID
	string, // etID (Server)
	string, // etvID (Server v1)
	string, // attrID (hostname)
	*repository.CatalogGormRepo,
	*repository.EntityInstanceGormRepo,
	*repository.InstanceAttributeValueGormRepo,
	*repository.AssociationLinkGormRepo,
) {
	db := testutil.NewTestDB(t)
	ctx := context.Background()

	catRepo := repository.NewCatalogGormRepo(db)
	instRepo := repository.NewEntityInstanceGormRepo(db)
	iavRepo := repository.NewInstanceAttributeValueGormRepo(db)
	pinRepo := repository.NewCatalogVersionPinGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	attrRepo := repository.NewAttributeGormRepo(db)
	assocRepo := repository.NewAssociationGormRepo(db)
	tdvRepo := repository.NewTypeDefinitionVersionGormRepo(db)
	tdRepo := repository.NewTypeDefinitionGormRepo(db)
	linkRepo := repository.NewAssociationLinkGormRepo(db)
	etRepo := repository.NewEntityTypeGormRepo(db)
	cvRepo := repository.NewCatalogVersionGormRepo(db)

	// Seed system type definitions for validation
	now := time.Now()
	tdString := &models.TypeDefinition{ID: "sys-td-string", Name: "string", BaseType: models.BaseTypeString, System: true, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, tdRepo.Create(ctx, tdString))
	tdvString := &models.TypeDefinitionVersion{ID: "sys-tdv-string", TypeDefinitionID: tdString.ID, VersionNumber: 1, Constraints: map[string]any{}, CreatedAt: now}
	require.NoError(t, tdvRepo.Create(ctx, tdvString))

	svc := operational.NewCatalogValidationService(
		catRepo, instRepo, iavRepo, pinRepo, etvRepo,
		attrRepo, assocRepo, tdvRepo, tdRepo, linkRepo, etRepo,
	)

	// Create entity type "Server"
	etID := newID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: etID, Name: "Server", CreatedAt: now, UpdatedAt: now}))

	// Create entity type version
	etvID := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID, EntityTypeID: etID, Version: 1, CreatedAt: now}))

	// Create required attribute "hostname"
	attrID := newID()
	require.NoError(t, attrRepo.Create(ctx, &models.Attribute{
		ID: attrID, EntityTypeVersionID: etvID, Name: "hostname", TypeDefinitionVersionID: tdvString.ID, Required: true, Ordinal: 1,
	}))

	// Create optional attribute "description"
	require.NoError(t, attrRepo.Create(ctx, &models.Attribute{
		ID: newID(), EntityTypeVersionID: etvID, Name: "description", TypeDefinitionVersionID: tdvString.ID, Required: false, Ordinal: 2,
	}))

	// Create catalog version + pin
	cvID := newID()
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{
		ID: cvID, VersionLabel: "v1", LifecycleStage: "development", CreatedAt: now, UpdatedAt: now,
	}))
	require.NoError(t, pinRepo.Create(ctx, &models.CatalogVersionPin{ID: newID(), CatalogVersionID: cvID, EntityTypeVersionID: etvID}))

	// Create catalog
	catalogID := newID()
	require.NoError(t, catRepo.Create(ctx, &models.Catalog{
		ID: catalogID, Name: "test-catalog", CatalogVersionID: cvID,
		ValidationStatus: models.ValidationStatusDraft, CreatedAt: now, UpdatedAt: now,
	}))

	return svc, ctx, catalogID, etID, etvID, attrID, catRepo, instRepo, iavRepo, linkRepo
}

// T-15.26: Full validation with valid catalog (all attrs set)
func TestT15_26_IntegrationValid(t *testing.T) {
	svc, ctx, catalogID, etID, _, attrID, catRepo, instRepo, iavRepo, _ := setupValidationIntegration(t)

	instID := newID()
	require.NoError(t, instRepo.Create(ctx, &models.EntityInstance{
		ID: instID, EntityTypeID: etID, CatalogID: catalogID, Name: "server-1",
		Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	require.NoError(t, iavRepo.SetValues(ctx, []*models.InstanceAttributeValue{
		{ID: newID(), InstanceID: instID, InstanceVersion: 1, AttributeID: attrID, ValueString: "web-01"},
	}))

	result, err := svc.Validate(ctx, "test-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusValid, result.Status)
	assert.Empty(t, result.Errors)

	// Verify status persisted
	cat, err := catRepo.GetByName(ctx, "test-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusValid, cat.ValidationStatus)
}

// T-15.27: Full validation with missing required attribute
func TestT15_27_IntegrationMissingRequired(t *testing.T) {
	svc, ctx, catalogID, etID, _, _, _, instRepo, _, _ := setupValidationIntegration(t)

	instID := newID()
	require.NoError(t, instRepo.Create(ctx, &models.EntityInstance{
		ID: instID, EntityTypeID: etID, CatalogID: catalogID, Name: "server-1",
		Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	// No attribute values set — hostname is required

	result, err := svc.Validate(ctx, "test-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusInvalid, result.Status)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "hostname", result.Errors[0].Field)
	assert.Contains(t, result.Errors[0].Violation, "required")
}

// T-15.28: Full validation with invalid enum value
func TestT15_28_IntegrationInvalidEnum(t *testing.T) {
	db := testutil.NewTestDB(t)
	ctx := context.Background()
	now := time.Now()

	catRepo := repository.NewCatalogGormRepo(db)
	instRepo := repository.NewEntityInstanceGormRepo(db)
	iavRepo := repository.NewInstanceAttributeValueGormRepo(db)
	pinRepo := repository.NewCatalogVersionPinGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	attrRepo := repository.NewAttributeGormRepo(db)
	assocRepo := repository.NewAssociationGormRepo(db)
	tdvRepo := repository.NewTypeDefinitionVersionGormRepo(db)
	tdRepo := repository.NewTypeDefinitionGormRepo(db)
	linkRepo := repository.NewAssociationLinkGormRepo(db)
	etRepo := repository.NewEntityTypeGormRepo(db)
	cvRepo := repository.NewCatalogVersionGormRepo(db)

	// Create enum type definition with values
	tdEnum := &models.TypeDefinition{ID: newID(), Name: "status", BaseType: models.BaseTypeEnum, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, tdRepo.Create(ctx, tdEnum))
	tdvEnum := &models.TypeDefinitionVersion{ID: newID(), TypeDefinitionID: tdEnum.ID, VersionNumber: 1,
		Constraints: map[string]any{"values": []any{"active", "inactive"}}, CreatedAt: now}
	require.NoError(t, tdvRepo.Create(ctx, tdvEnum))

	svc := operational.NewCatalogValidationService(
		catRepo, instRepo, iavRepo, pinRepo, etvRepo,
		attrRepo, assocRepo, tdvRepo, tdRepo, linkRepo, etRepo,
	)

	// Create entity type + version
	etID := newID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: etID, Name: "Server", CreatedAt: now, UpdatedAt: now}))
	etvID := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID, EntityTypeID: etID, Version: 1, CreatedAt: now}))

	// Create enum attribute referencing the type definition version
	attrID := newID()
	require.NoError(t, attrRepo.Create(ctx, &models.Attribute{
		ID: attrID, EntityTypeVersionID: etvID, Name: "status", TypeDefinitionVersionID: tdvEnum.ID, Required: true, Ordinal: 1,
	}))

	// Create CV + pin + catalog
	cvID := newID()
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{ID: cvID, VersionLabel: "v1", LifecycleStage: "development", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, pinRepo.Create(ctx, &models.CatalogVersionPin{ID: newID(), CatalogVersionID: cvID, EntityTypeVersionID: etvID}))
	catalogID := newID()
	require.NoError(t, catRepo.Create(ctx, &models.Catalog{ID: catalogID, Name: "enum-test", CatalogVersionID: cvID, ValidationStatus: models.ValidationStatusDraft, CreatedAt: now, UpdatedAt: now}))

	// Create instance with invalid enum value (stored in ValueString)
	instID := newID()
	require.NoError(t, instRepo.Create(ctx, &models.EntityInstance{ID: instID, EntityTypeID: etID, CatalogID: catalogID, Name: "server-1", Version: 1, CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, iavRepo.SetValues(ctx, []*models.InstanceAttributeValue{
		{ID: newID(), InstanceID: instID, InstanceVersion: 1, AttributeID: attrID, ValueString: "bogus"},
	}))

	result, err := svc.Validate(ctx, "enum-test")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusInvalid, result.Status)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0].Violation, "invalid enum value")
}

// T-15.31: Multiple violations across entity types
func TestT15_31_IntegrationMultipleViolations(t *testing.T) {
	svc, ctx, catalogID, etID, _, _, _, instRepo, _, _ := setupValidationIntegration(t)

	// Create two instances with no required attrs set
	require.NoError(t, instRepo.Create(ctx, &models.EntityInstance{
		ID: newID(), EntityTypeID: etID, CatalogID: catalogID, Name: "server-1",
		Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	require.NoError(t, instRepo.Create(ctx, &models.EntityInstance{
		ID: newID(), EntityTypeID: etID, CatalogID: catalogID, Name: "server-2",
		Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	result, err := svc.Validate(ctx, "test-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusInvalid, result.Status)
	assert.Len(t, result.Errors, 2) // one per instance for missing hostname
}

// T-15.32: Validation status persisted correctly
func TestT15_32_IntegrationStatusPersisted(t *testing.T) {
	svc, ctx, catalogID, etID, _, attrID, catRepo, instRepo, iavRepo, _ := setupValidationIntegration(t)

	instID := newID()
	require.NoError(t, instRepo.Create(ctx, &models.EntityInstance{
		ID: instID, EntityTypeID: etID, CatalogID: catalogID, Name: "server-1",
		Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	require.NoError(t, iavRepo.SetValues(ctx, []*models.InstanceAttributeValue{
		{ID: newID(), InstanceID: instID, InstanceVersion: 1, AttributeID: attrID, ValueString: "web-01"},
	}))

	// Validate — should be valid
	result, err := svc.Validate(ctx, "test-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusValid, result.Status)

	cat, err := catRepo.GetByName(ctx, "test-catalog")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusValid, cat.ValidationStatus)
}
