//go:build postgres_integration

package repository_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	gormmodels "github.com/project-catalyst/pc-asset-hub/internal/infrastructure/gorm/models"
	"github.com/project-catalyst/pc-asset-hub/internal/infrastructure/gorm/repository"
)

func pgID() string {
	return uuid.Must(uuid.NewV7()).String()
}

func newPGTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("PG_DSN")
	if dsn == "" {
		dsn = "host=localhost user=assethub password=assethub dbname=assethub port=5432 sslmode=disable"
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	require.NoError(t, gormmodels.InitDB(db))

	// Clean all tables before each test
	tables := []string{
		"association_links", "instance_attribute_values", "entity_instances",
		"lifecycle_transitions", "catalog_version_pins", "catalog_versions",
		"associations", "attributes", "enum_values", "enums",
		"entity_type_versions", "entity_types",
	}
	for _, table := range tables {
		db.Exec("DELETE FROM " + table)
	}

	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

// T-B.04: PostgreSQL integration — runs the same repo tests against real PostgreSQL

func TestPG_CreateEntityType(t *testing.T) {
	db := newPGTestDB(t)
	repo := repository.NewEntityTypeGormRepo(db)
	ctx := context.Background()

	et := &models.EntityType{
		ID:        pgID(),
		Name:      "Model",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := repo.Create(ctx, et)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, et.ID)
	require.NoError(t, err)
	assert.Equal(t, "Model", found.Name)
}

func TestPG_CreateDuplicateEntityType(t *testing.T) {
	db := newPGTestDB(t)
	repo := repository.NewEntityTypeGormRepo(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &models.EntityType{ID: pgID(), Name: "Model", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	err := repo.Create(ctx, &models.EntityType{ID: pgID(), Name: "Model", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	assert.True(t, domainerrors.IsConflict(err))
}

func TestPG_ListEntityTypes(t *testing.T) {
	db := newPGTestDB(t)
	repo := repository.NewEntityTypeGormRepo(db)
	ctx := context.Background()

	for _, name := range []string{"Alpha", "Beta", "Charlie"} {
		require.NoError(t, repo.Create(ctx, &models.EntityType{ID: pgID(), Name: name, CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	}

	items, total, err := repo.List(ctx, models.ListParams{Limit: 10, Offset: 0})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, items, 3)
}

func TestPG_CreateEntityTypeVersion(t *testing.T) {
	db := newPGTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	ctx := context.Background()

	etID := pgID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: etID, Name: "Model", CreatedAt: time.Now(), UpdatedAt: time.Now()}))

	etvID := pgID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID, EntityTypeID: etID, Version: 1, CreatedAt: time.Now()}))

	found, err := etvRepo.GetByEntityTypeAndVersion(ctx, etID, 1)
	require.NoError(t, err)
	assert.Equal(t, 1, found.Version)
}

func TestPG_CreateAttribute(t *testing.T) {
	db := newPGTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	attrRepo := repository.NewAttributeGormRepo(db)
	ctx := context.Background()

	etID := pgID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: etID, Name: "Model", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	etvID := pgID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID, EntityTypeID: etID, Version: 1, CreatedAt: time.Now()}))

	attrID := pgID()
	require.NoError(t, attrRepo.Create(ctx, &models.Attribute{ID: attrID, EntityTypeVersionID: etvID, Name: "endpoint", Type: models.AttributeTypeString, Ordinal: 0}))

	found, err := attrRepo.GetByID(ctx, attrID)
	require.NoError(t, err)
	assert.Equal(t, models.AttributeTypeString, found.Type)
}

func TestPG_CatalogVersionLifecycle(t *testing.T) {
	db := newPGTestDB(t)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	ctx := context.Background()

	cvID := pgID()
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{ID: cvID, VersionLabel: "v1.0", LifecycleStage: models.LifecycleStageDevelopment, CreatedAt: time.Now(), UpdatedAt: time.Now()}))

	require.NoError(t, cvRepo.UpdateLifecycle(ctx, cvID, models.LifecycleStageTesting))
	found, err := cvRepo.GetByID(ctx, cvID)
	require.NoError(t, err)
	assert.Equal(t, models.LifecycleStageTesting, found.LifecycleStage)
}

func TestPG_EntityInstance(t *testing.T) {
	db := newPGTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	pinRepo := repository.NewCatalogVersionPinGormRepo(db)
	instRepo := repository.NewEntityInstanceGormRepo(db)
	ctx := context.Background()

	etID := pgID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: etID, Name: "Model", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	etvID := pgID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID, EntityTypeID: etID, Version: 1, CreatedAt: time.Now()}))
	cvID := pgID()
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{ID: cvID, VersionLabel: "v1.0", LifecycleStage: models.LifecycleStageDevelopment, CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, pinRepo.Create(ctx, &models.CatalogVersionPin{ID: pgID(), CatalogVersionID: cvID, EntityTypeVersionID: etvID}))

	instID := pgID()
	require.NoError(t, instRepo.Create(ctx, &models.EntityInstance{ID: instID, EntityTypeVersionID: etvID, CatalogVersionID: cvID, Name: "llama-3", CreatedAt: time.Now(), UpdatedAt: time.Now()}))

	found, err := instRepo.GetByID(ctx, instID)
	require.NoError(t, err)
	assert.Equal(t, "llama-3", found.Name)
}

func TestPG_DeleteCascade(t *testing.T) {
	db := newPGTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	attrRepo := repository.NewAttributeGormRepo(db)
	ctx := context.Background()

	etID := pgID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: etID, Name: "Model", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	etvID := pgID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID, EntityTypeID: etID, Version: 1, CreatedAt: time.Now()}))
	require.NoError(t, attrRepo.Create(ctx, &models.Attribute{ID: pgID(), EntityTypeVersionID: etvID, Name: "a", Type: models.AttributeTypeString, Ordinal: 0}))

	require.NoError(t, etRepo.Delete(ctx, etID))

	_, err := etvRepo.GetByEntityTypeAndVersion(ctx, etID, 1)
	assert.True(t, domainerrors.IsNotFound(err))
}

func TestPG_EnumCRUD(t *testing.T) {
	db := newPGTestDB(t)
	enumRepo := repository.NewEnumGormRepo(db)
	evRepo := repository.NewEnumValueGormRepo(db)
	ctx := context.Background()

	eID := pgID()
	require.NoError(t, enumRepo.Create(ctx, &models.Enum{ID: eID, Name: "Status", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, evRepo.Create(ctx, &models.EnumValue{ID: pgID(), EnumID: eID, Value: "active", Ordinal: 0}))
	require.NoError(t, evRepo.Create(ctx, &models.EnumValue{ID: pgID(), EnumID: eID, Value: "inactive", Ordinal: 1}))

	values, err := evRepo.ListByEnum(ctx, eID)
	require.NoError(t, err)
	assert.Len(t, values, 2)
}
