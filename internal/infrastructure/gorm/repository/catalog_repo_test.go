package repository_test

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
)

func newCatalogID() string { return uuid.Must(uuid.NewV7()).String() }

// T-10.01: Create catalog with valid name and CV ID
func TestT10_01_CreateCatalog(t *testing.T) {
	db := testutil.NewTestDB(t)
	catalogRepo := repository.NewCatalogGormRepo(db)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	ctx := context.Background()

	// Create a CV for FK reference
	cvID := newCatalogID()
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{
		ID: cvID, VersionLabel: "v1", LifecycleStage: "development",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	catalog := &models.Catalog{
		ID:               newCatalogID(),
		Name:             "production-app-a",
		Description:      "Production catalog",
		CatalogVersionID: cvID,
		ValidationStatus: models.ValidationStatusDraft,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	err := catalogRepo.Create(ctx, catalog)
	require.NoError(t, err)

	// Verify it was stored
	found, err := catalogRepo.GetByName(ctx, "production-app-a")
	require.NoError(t, err)
	assert.Equal(t, catalog.ID, found.ID)
	assert.Equal(t, "production-app-a", found.Name)
	assert.Equal(t, "Production catalog", found.Description)
	assert.Equal(t, cvID, found.CatalogVersionID)
	assert.Equal(t, models.ValidationStatusDraft, found.ValidationStatus)
}

// T-10.02: Create catalog with duplicate name
func TestT10_02_DuplicateName(t *testing.T) {
	db := testutil.NewTestDB(t)
	catalogRepo := repository.NewCatalogGormRepo(db)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	ctx := context.Background()

	cvID := newCatalogID()
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{
		ID: cvID, VersionLabel: "v1", LifecycleStage: "development",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	catalog1 := &models.Catalog{
		ID: newCatalogID(), Name: "my-catalog", CatalogVersionID: cvID,
		ValidationStatus: models.ValidationStatusDraft, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, catalogRepo.Create(ctx, catalog1))

	catalog2 := &models.Catalog{
		ID: newCatalogID(), Name: "my-catalog", CatalogVersionID: cvID,
		ValidationStatus: models.ValidationStatusDraft, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	err := catalogRepo.Create(ctx, catalog2)
	assert.Error(t, err)
}

// T-10.04: GetByName retrieves catalog by name
func TestT10_04_GetByName(t *testing.T) {
	db := testutil.NewTestDB(t)
	catalogRepo := repository.NewCatalogGormRepo(db)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	ctx := context.Background()

	cvID := newCatalogID()
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{
		ID: cvID, VersionLabel: "v1", LifecycleStage: "development",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	id := newCatalogID()
	require.NoError(t, catalogRepo.Create(ctx, &models.Catalog{
		ID: id, Name: "test-catalog", CatalogVersionID: cvID,
		ValidationStatus: models.ValidationStatusDraft, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	found, err := catalogRepo.GetByName(ctx, "test-catalog")
	require.NoError(t, err)
	assert.Equal(t, id, found.ID)
	assert.Equal(t, "test-catalog", found.Name)
}

// T-10.05: GetByName for nonexistent name
func TestT10_05_GetByNameNotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	catalogRepo := repository.NewCatalogGormRepo(db)
	ctx := context.Background()

	_, err := catalogRepo.GetByName(ctx, "nonexistent")
	assert.Error(t, err)
}

// T-10.06: List catalogs returns all
func TestT10_06_ListAll(t *testing.T) {
	db := testutil.NewTestDB(t)
	catalogRepo := repository.NewCatalogGormRepo(db)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	ctx := context.Background()

	cvID := newCatalogID()
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{
		ID: cvID, VersionLabel: "v1", LifecycleStage: "development",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	for _, name := range []string{"catalog-a", "catalog-b", "catalog-c"} {
		require.NoError(t, catalogRepo.Create(ctx, &models.Catalog{
			ID: newCatalogID(), Name: name, CatalogVersionID: cvID,
			ValidationStatus: models.ValidationStatusDraft, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}))
	}

	catalogs, total, err := catalogRepo.List(ctx, models.ListParams{Limit: 20})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, catalogs, 3)
}

// T-10.07: List catalogs filtered by catalog_version_id
func TestT10_07_ListFilterByCatalogVersionID(t *testing.T) {
	db := testutil.NewTestDB(t)
	catalogRepo := repository.NewCatalogGormRepo(db)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	ctx := context.Background()

	cv1 := newCatalogID()
	cv2 := newCatalogID()
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{
		ID: cv1, VersionLabel: "v1", LifecycleStage: "development",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{
		ID: cv2, VersionLabel: "v2", LifecycleStage: "development",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	require.NoError(t, catalogRepo.Create(ctx, &models.Catalog{
		ID: newCatalogID(), Name: "cat-a", CatalogVersionID: cv1,
		ValidationStatus: models.ValidationStatusDraft, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	require.NoError(t, catalogRepo.Create(ctx, &models.Catalog{
		ID: newCatalogID(), Name: "cat-b", CatalogVersionID: cv2,
		ValidationStatus: models.ValidationStatusDraft, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	catalogs, total, err := catalogRepo.List(ctx, models.ListParams{
		Limit:   20,
		Filters: map[string]string{"catalog_version_id": cv1},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, catalogs, 1)
	assert.Equal(t, "cat-a", catalogs[0].Name)
}

// T-10.08: List catalogs filtered by validation_status
func TestT10_08_ListFilterByValidationStatus(t *testing.T) {
	db := testutil.NewTestDB(t)
	catalogRepo := repository.NewCatalogGormRepo(db)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	ctx := context.Background()

	cvID := newCatalogID()
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{
		ID: cvID, VersionLabel: "v1", LifecycleStage: "development",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	require.NoError(t, catalogRepo.Create(ctx, &models.Catalog{
		ID: newCatalogID(), Name: "draft-cat", CatalogVersionID: cvID,
		ValidationStatus: models.ValidationStatusDraft, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	require.NoError(t, catalogRepo.Create(ctx, &models.Catalog{
		ID: newCatalogID(), Name: "valid-cat", CatalogVersionID: cvID,
		ValidationStatus: models.ValidationStatusValid, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	catalogs, total, err := catalogRepo.List(ctx, models.ListParams{
		Limit:   20,
		Filters: map[string]string{"validation_status": "valid"},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "valid-cat", catalogs[0].Name)
}

// T-10.09: Delete catalog by ID
func TestT10_09_DeleteCatalog(t *testing.T) {
	db := testutil.NewTestDB(t)
	catalogRepo := repository.NewCatalogGormRepo(db)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	ctx := context.Background()

	cvID := newCatalogID()
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{
		ID: cvID, VersionLabel: "v1", LifecycleStage: "development",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	catID := newCatalogID()
	require.NoError(t, catalogRepo.Create(ctx, &models.Catalog{
		ID: catID, Name: "to-delete", CatalogVersionID: cvID,
		ValidationStatus: models.ValidationStatusDraft, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	err := catalogRepo.Delete(ctx, catID)
	require.NoError(t, err)

	_, err = catalogRepo.GetByName(ctx, "to-delete")
	assert.Error(t, err)
}

// T-10.10: EntityInstance uses catalog_id FK
func TestT10_10_EntityInstanceCatalogFK(t *testing.T) {
	db := testutil.NewTestDB(t)
	catalogRepo := repository.NewCatalogGormRepo(db)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	instRepo := repository.NewEntityInstanceGormRepo(db)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	ctx := context.Background()

	// Create entity type
	etID := newCatalogID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: etID, Name: "model", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	etvID := newCatalogID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID, EntityTypeID: etID, Version: 1, CreatedAt: time.Now()}))

	// Create CV and catalog
	cvID := newCatalogID()
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{
		ID: cvID, VersionLabel: "v1", LifecycleStage: "development",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	catID := newCatalogID()
	require.NoError(t, catalogRepo.Create(ctx, &models.Catalog{
		ID: catID, Name: "test-cat", CatalogVersionID: cvID,
		ValidationStatus: models.ValidationStatusDraft, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	// Create instance with catalog_id
	instID := newCatalogID()
	err := instRepo.Create(ctx, &models.EntityInstance{
		ID: instID, EntityTypeID: etID, CatalogID: catID,
		Name: "my-model", Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})
	require.NoError(t, err)

	// Verify
	inst, err := instRepo.GetByID(ctx, instID)
	require.NoError(t, err)
	assert.Equal(t, catID, inst.CatalogID)
}

// Coverage: GetByID
func TestCatalogRepo_GetByID(t *testing.T) {
	db := testutil.NewTestDB(t)
	catalogRepo := repository.NewCatalogGormRepo(db)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	ctx := context.Background()

	cvID := newCatalogID()
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{
		ID: cvID, VersionLabel: "v1", LifecycleStage: "development",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	catID := newCatalogID()
	require.NoError(t, catalogRepo.Create(ctx, &models.Catalog{
		ID: catID, Name: "test-cat", CatalogVersionID: cvID,
		ValidationStatus: models.ValidationStatusDraft, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	found, err := catalogRepo.GetByID(ctx, catID)
	require.NoError(t, err)
	assert.Equal(t, "test-cat", found.Name)

	_, err = catalogRepo.GetByID(ctx, "nonexistent")
	assert.Error(t, err)
}

// Coverage: UpdateValidationStatus
func TestCatalogRepo_UpdateValidationStatus(t *testing.T) {
	db := testutil.NewTestDB(t)
	catalogRepo := repository.NewCatalogGormRepo(db)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	ctx := context.Background()

	cvID := newCatalogID()
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{
		ID: cvID, VersionLabel: "v1", LifecycleStage: "development",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	catID := newCatalogID()
	now := time.Now()
	require.NoError(t, catalogRepo.Create(ctx, &models.Catalog{
		ID: catID, Name: "status-test", CatalogVersionID: cvID,
		ValidationStatus: models.ValidationStatusDraft, CreatedAt: now, UpdatedAt: now,
	}))

	err := catalogRepo.UpdateValidationStatus(ctx, catID, models.ValidationStatusValid)
	require.NoError(t, err)

	found, err := catalogRepo.GetByName(ctx, "status-test")
	require.NoError(t, err)
	assert.Equal(t, models.ValidationStatusValid, found.ValidationStatus)
	assert.True(t, found.UpdatedAt.After(now) || found.UpdatedAt.Equal(now))

	// Nonexistent ID
	err = catalogRepo.UpdateValidationStatus(ctx, "nonexistent", models.ValidationStatusInvalid)
	assert.Error(t, err)
}

// Coverage: Delete nonexistent
func TestCatalogRepo_DeleteNotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	catalogRepo := repository.NewCatalogGormRepo(db)
	ctx := context.Background()

	err := catalogRepo.Delete(ctx, "nonexistent")
	assert.Error(t, err)
}

// Coverage: DeleteByCatalogID
func TestCatalogRepo_DeleteByCatalogID(t *testing.T) {
	db := testutil.NewTestDB(t)
	catalogRepo := repository.NewCatalogGormRepo(db)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	instRepo := repository.NewEntityInstanceGormRepo(db)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	ctx := context.Background()

	etID := newCatalogID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: etID, Name: "model", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	etvID := newCatalogID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID, EntityTypeID: etID, Version: 1, CreatedAt: time.Now()}))

	cvID := newCatalogID()
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{
		ID: cvID, VersionLabel: "v1", LifecycleStage: "development",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	catID := newCatalogID()
	require.NoError(t, catalogRepo.Create(ctx, &models.Catalog{
		ID: catID, Name: "del-cat", CatalogVersionID: cvID,
		ValidationStatus: models.ValidationStatusDraft, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	// Create two instances in the catalog
	for _, name := range []string{"inst-1", "inst-2"} {
		require.NoError(t, instRepo.Create(ctx, &models.EntityInstance{
			ID: newCatalogID(), EntityTypeID: etID, CatalogID: catID,
			Name: name, Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}))
	}

	// Delete all by catalog ID
	err := instRepo.DeleteByCatalogID(ctx, catID)
	require.NoError(t, err)

	// Instances should be soft-deleted (not visible)
	items, total, err := instRepo.List(ctx, etID, catID, models.ListParams{Limit: 20})
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Len(t, items, 0)

	// DeleteByCatalogID on catalog with no instances is fine
	err = instRepo.DeleteByCatalogID(ctx, "no-such-catalog")
	require.NoError(t, err)
}

// Coverage: List with no results returns empty slice
func TestCatalogRepo_ListEmpty(t *testing.T) {
	db := testutil.NewTestDB(t)
	catalogRepo := repository.NewCatalogGormRepo(db)
	ctx := context.Background()

	catalogs, total, err := catalogRepo.List(ctx, models.ListParams{Limit: 20})
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Len(t, catalogs, 0)
}

// T-16.01/02: Published fields persist correctly
func TestT16_01_UpdatePublished(t *testing.T) {
	db := testutil.NewTestDB(t)
	catalogRepo := repository.NewCatalogGormRepo(db)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	ctx := context.Background()

	cvID := newCatalogID()
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{
		ID: cvID, VersionLabel: "v1", LifecycleStage: "development",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	catID := newCatalogID()
	require.NoError(t, catalogRepo.Create(ctx, &models.Catalog{
		ID: catID, Name: "pub-test", CatalogVersionID: cvID,
		ValidationStatus: models.ValidationStatusValid,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	// Verify default: published=false
	cat, err := catalogRepo.GetByName(ctx, "pub-test")
	require.NoError(t, err)
	assert.False(t, cat.Published)
	assert.Nil(t, cat.PublishedAt)

	// Publish
	now := time.Now()
	require.NoError(t, catalogRepo.UpdatePublished(ctx, catID, true, &now))

	cat, err = catalogRepo.GetByName(ctx, "pub-test")
	require.NoError(t, err)
	assert.True(t, cat.Published)
	assert.NotNil(t, cat.PublishedAt)

	// Unpublish
	require.NoError(t, catalogRepo.UpdatePublished(ctx, catID, false, nil))

	cat, err = catalogRepo.GetByName(ctx, "pub-test")
	require.NoError(t, err)
	assert.False(t, cat.Published)
}

// T-16.44 (partial): ListByCatalogVersionID
func TestListByCatalogVersionID(t *testing.T) {
	db := testutil.NewTestDB(t)
	catalogRepo := repository.NewCatalogGormRepo(db)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	ctx := context.Background()

	cv1ID := newCatalogID()
	cv2ID := newCatalogID()
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{ID: cv1ID, VersionLabel: "v1", LifecycleStage: "development", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{ID: cv2ID, VersionLabel: "v2", LifecycleStage: "development", CreatedAt: time.Now(), UpdatedAt: time.Now()}))

	require.NoError(t, catalogRepo.Create(ctx, &models.Catalog{ID: newCatalogID(), Name: "cat-a", CatalogVersionID: cv1ID, ValidationStatus: models.ValidationStatusDraft, CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, catalogRepo.Create(ctx, &models.Catalog{ID: newCatalogID(), Name: "cat-b", CatalogVersionID: cv1ID, ValidationStatus: models.ValidationStatusValid, CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, catalogRepo.Create(ctx, &models.Catalog{ID: newCatalogID(), Name: "cat-c", CatalogVersionID: cv2ID, ValidationStatus: models.ValidationStatusValid, CreatedAt: time.Now(), UpdatedAt: time.Now()}))

	// Only catalogs pinned to cv1
	cats, err := catalogRepo.ListByCatalogVersionID(ctx, cv1ID)
	require.NoError(t, err)
	assert.Len(t, cats, 2)

	// cv2 has one catalog
	cats, err = catalogRepo.ListByCatalogVersionID(ctx, cv2ID)
	require.NoError(t, err)
	assert.Len(t, cats, 1)
	assert.Equal(t, "cat-c", cats[0].Name)

	// Nonexistent CV — empty result
	cats, err = catalogRepo.ListByCatalogVersionID(ctx, "no-such-cv")
	require.NoError(t, err)
	assert.Len(t, cats, 0)
}

