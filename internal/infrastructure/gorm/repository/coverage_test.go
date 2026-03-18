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
)

// === EntityTypeVersion repo coverage ===

func TestETV_GetByID(t *testing.T) {
	db := testutil.NewTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	ctx := context.Background()

	etID := newID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: etID, Name: "M", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	etvID := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID, EntityTypeID: etID, Version: 1, CreatedAt: time.Now()}))

	found, err := etvRepo.GetByID(ctx, etvID)
	require.NoError(t, err)
	assert.Equal(t, 1, found.Version)

	_, err = etvRepo.GetByID(ctx, "nonexistent")
	assert.Error(t, err)
}

func TestETV_GetLatestByEntityType(t *testing.T) {
	db := testutil.NewTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	ctx := context.Background()

	etID := newID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: etID, Name: "M", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: newID(), EntityTypeID: etID, Version: 1, CreatedAt: time.Now()}))
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: newID(), EntityTypeID: etID, Version: 2, CreatedAt: time.Now()}))

	latest, err := etvRepo.GetLatestByEntityType(ctx, etID)
	require.NoError(t, err)
	assert.Equal(t, 2, latest.Version)

	_, err = etvRepo.GetLatestByEntityType(ctx, "nonexistent")
	assert.Error(t, err)
}

func TestETV_ListByEntityType(t *testing.T) {
	db := testutil.NewTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	ctx := context.Background()

	etID := newID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: etID, Name: "M", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: newID(), EntityTypeID: etID, Version: 1, CreatedAt: time.Now()}))
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: newID(), EntityTypeID: etID, Version: 2, CreatedAt: time.Now()}))

	versions, err := etvRepo.ListByEntityType(ctx, etID)
	require.NoError(t, err)
	assert.Len(t, versions, 2)
	assert.Equal(t, 1, versions[0].Version) // ASC order
}

// === EntityType repo coverage ===

func TestET_Update(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewEntityTypeGormRepo(db)
	ctx := context.Background()

	etID := newID()
	require.NoError(t, repo.Create(ctx, &models.EntityType{ID: etID, Name: "Original", CreatedAt: time.Now(), UpdatedAt: time.Now()}))

	found, _ := repo.GetByID(ctx, etID)
	found.Name = "Updated"
	require.NoError(t, repo.Update(ctx, found))

	updated, err := repo.GetByID(ctx, etID)
	require.NoError(t, err)
	assert.Equal(t, "Updated", updated.Name)
}

func TestET_ListWithSortAndFilter(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewEntityTypeGormRepo(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &models.EntityType{ID: newID(), Name: "Zulu", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, repo.Create(ctx, &models.EntityType{ID: newID(), Name: "Alpha", CreatedAt: time.Now(), UpdatedAt: time.Now()}))

	// Sort descending
	items, _, err := repo.List(ctx, models.ListParams{Limit: 10, SortBy: "name", SortDesc: true})
	require.NoError(t, err)
	assert.Equal(t, "Zulu", items[0].Name)

	// Invalid sort column
	_, _, err = repo.List(ctx, models.ListParams{Limit: 10, SortBy: "invalid_col"})
	assert.Error(t, err)

	// Filter by name
	items, total, err := repo.List(ctx, models.ListParams{Limit: 10, Filters: map[string]string{"name": "Alpha"}})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "Alpha", items[0].Name)
}

// === Enum repo coverage ===

func TestEnum_GetByID(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewEnumGormRepo(db)
	ctx := context.Background()

	eID := newID()
	require.NoError(t, repo.Create(ctx, &models.Enum{ID: eID, Name: "Status", CreatedAt: time.Now(), UpdatedAt: time.Now()}))

	found, err := repo.GetByID(ctx, eID)
	require.NoError(t, err)
	assert.Equal(t, "Status", found.Name)

	_, err = repo.GetByID(ctx, "nonexistent")
	assert.Error(t, err)
}

func TestEnum_GetByName(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewEnumGormRepo(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &models.Enum{ID: newID(), Name: "Priority", CreatedAt: time.Now(), UpdatedAt: time.Now()}))

	found, err := repo.GetByName(ctx, "Priority")
	require.NoError(t, err)
	assert.Equal(t, "Priority", found.Name)

	_, err = repo.GetByName(ctx, "Nonexistent")
	assert.Error(t, err)
}

func TestEnum_List(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewEnumGormRepo(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &models.Enum{ID: newID(), Name: "Alpha", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, repo.Create(ctx, &models.Enum{ID: newID(), Name: "Beta", CreatedAt: time.Now(), UpdatedAt: time.Now()}))

	items, total, err := repo.List(ctx, models.ListParams{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, items, 2)

	// With filter
	filtered, total, err := repo.List(ctx, models.ListParams{Limit: 10, Filters: map[string]string{"name": "Alpha"}})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "Alpha", filtered[0].Name)

	// With offset
	items, _, err = repo.List(ctx, models.ListParams{Limit: 10, Offset: 1})
	require.NoError(t, err)
	assert.Len(t, items, 1)
}

func TestEnum_Update(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewEnumGormRepo(db)
	ctx := context.Background()

	eID := newID()
	require.NoError(t, repo.Create(ctx, &models.Enum{ID: eID, Name: "Status", CreatedAt: time.Now(), UpdatedAt: time.Now()}))

	found, _ := repo.GetByID(ctx, eID)
	found.Name = "UpdatedStatus"
	require.NoError(t, repo.Update(ctx, found))

	updated, err := repo.GetByID(ctx, eID)
	require.NoError(t, err)
	assert.Equal(t, "UpdatedStatus", updated.Name)
}

func TestEnum_DeleteNotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewEnumGormRepo(db)
	ctx := context.Background()

	err := repo.Delete(ctx, "nonexistent")
	assert.Error(t, err)
}

// === CatalogVersion repo coverage ===

func TestCV_GetByLabel(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewCatalogVersionGormRepo(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &models.CatalogVersion{ID: newID(), VersionLabel: "v1.0", LifecycleStage: models.LifecycleStageDevelopment, CreatedAt: time.Now(), UpdatedAt: time.Now()}))

	found, err := repo.GetByLabel(ctx, "v1.0")
	require.NoError(t, err)
	assert.Equal(t, "v1.0", found.VersionLabel)

	_, err = repo.GetByLabel(ctx, "nonexistent")
	assert.Error(t, err)
}

func TestCV_List(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewCatalogVersionGormRepo(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &models.CatalogVersion{ID: newID(), VersionLabel: "v1.0", LifecycleStage: models.LifecycleStageDevelopment, CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, repo.Create(ctx, &models.CatalogVersion{ID: newID(), VersionLabel: "v2.0", LifecycleStage: models.LifecycleStageTesting, CreatedAt: time.Now(), UpdatedAt: time.Now()}))

	items, total, err := repo.List(ctx, models.ListParams{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, items, 2)

	// With stage filter
	filtered, total, err := repo.List(ctx, models.ListParams{Limit: 10, Filters: map[string]string{"lifecycle_stage": "testing"}})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, filtered, 1)
}

func TestCVPin_Delete(t *testing.T) {
	db := testutil.NewTestDB(t)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	pinRepo := repository.NewCatalogVersionPinGormRepo(db)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	ctx := context.Background()

	cvID := newID()
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{ID: cvID, VersionLabel: "v1.0", LifecycleStage: models.LifecycleStageDevelopment, CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	etID := newID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: etID, Name: "M", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	etvID := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID, EntityTypeID: etID, Version: 1, CreatedAt: time.Now()}))
	pinID := newID()
	require.NoError(t, pinRepo.Create(ctx, &models.CatalogVersionPin{ID: pinID, CatalogVersionID: cvID, EntityTypeVersionID: etvID}))

	require.NoError(t, pinRepo.Delete(ctx, pinID))
	pins, err := pinRepo.ListByCatalogVersion(ctx, cvID)
	require.NoError(t, err)
	assert.Len(t, pins, 0)
}

// === Attribute repo coverage ===

func TestAttr_Update(t *testing.T) {
	db := testutil.NewTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	attrRepo := repository.NewAttributeGormRepo(db)
	ctx := context.Background()

	etID := newID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: etID, Name: "M", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	etvID := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID, EntityTypeID: etID, Version: 1, CreatedAt: time.Now()}))
	attrID := newID()
	require.NoError(t, attrRepo.Create(ctx, &models.Attribute{ID: attrID, EntityTypeVersionID: etvID, Name: "a", Type: models.AttributeTypeString, Ordinal: 0}))

	found, _ := attrRepo.GetByID(ctx, attrID)
	found.Description = "updated"
	require.NoError(t, attrRepo.Update(ctx, found))

	updated, err := attrRepo.GetByID(ctx, attrID)
	require.NoError(t, err)
	assert.Equal(t, "updated", updated.Description)
}

func TestAttr_Delete(t *testing.T) {
	db := testutil.NewTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	attrRepo := repository.NewAttributeGormRepo(db)
	ctx := context.Background()

	etID := newID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: etID, Name: "M", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	etvID := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID, EntityTypeID: etID, Version: 1, CreatedAt: time.Now()}))
	attrID := newID()
	require.NoError(t, attrRepo.Create(ctx, &models.Attribute{ID: attrID, EntityTypeVersionID: etvID, Name: "a", Type: models.AttributeTypeString, Ordinal: 0}))

	require.NoError(t, attrRepo.Delete(ctx, attrID))
	_, err := attrRepo.GetByID(ctx, attrID)
	assert.Error(t, err)
}

// === Association repo coverage ===

func TestAssoc_GetContainmentGraph(t *testing.T) {
	db := testutil.NewTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	assocRepo := repository.NewAssociationGormRepo(db)
	ctx := context.Background()

	et1ID, et2ID := newID(), newID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: et1ID, Name: "Parent", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: et2ID, Name: "Child", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	etvID := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID, EntityTypeID: et1ID, Version: 1, CreatedAt: time.Now()}))

	require.NoError(t, assocRepo.Create(ctx, &models.Association{ID: newID(), EntityTypeVersionID: etvID, TargetEntityTypeID: et2ID, Type: models.AssociationTypeContainment, CreatedAt: time.Now()}))

	edges, err := assocRepo.GetContainmentGraph(ctx)
	require.NoError(t, err)
	assert.Len(t, edges, 1)
	assert.Equal(t, et1ID, edges[0].SourceEntityTypeID)
	assert.Equal(t, et2ID, edges[0].TargetEntityTypeID)
}

// === EntityInstance repo coverage ===

func TestInstance_GetByNameAndParent(t *testing.T) {
	db := testutil.NewTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	pinRepo := repository.NewCatalogVersionPinGormRepo(db)
	instRepo := repository.NewEntityInstanceGormRepo(db)
	ctx := context.Background()

	etID := newID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: etID, Name: "M", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	etvID := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID, EntityTypeID: etID, Version: 1, CreatedAt: time.Now()}))
	cvID := newID()
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{ID: cvID, VersionLabel: "v1", LifecycleStage: models.LifecycleStageDevelopment, CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, pinRepo.Create(ctx, &models.CatalogVersionPin{ID: newID(), CatalogVersionID: cvID, EntityTypeVersionID: etvID}))

	instID := newID()
	require.NoError(t, instRepo.Create(ctx, &models.EntityInstance{ID: instID, EntityTypeID: etID, CatalogID: cvID, Name: "test", CreatedAt: time.Now(), UpdatedAt: time.Now()}))

	found, err := instRepo.GetByNameAndParent(ctx, etID, cvID, "", "test")
	require.NoError(t, err)
	assert.Equal(t, "test", found.Name)

	_, err = instRepo.GetByNameAndParent(ctx, etID, cvID, "", "nonexistent")
	assert.Error(t, err)
}

// === helpers coverage ===

func TestValidateSortBy(t *testing.T) {
	// We can't call validateSortBy directly from external test since it's unexported,
	// but we exercise it via List with invalid sort column above.
	// This test verifies the list function behavior.
	db := testutil.NewTestDB(t)
	repo := repository.NewEntityTypeGormRepo(db)
	ctx := context.Background()

	// Valid column
	_, _, err := repo.List(ctx, models.ListParams{Limit: 10, SortBy: "name"})
	assert.NoError(t, err)

	// Another valid column
	_, _, err = repo.List(ctx, models.ListParams{Limit: 10, SortBy: "created_at"})
	assert.NoError(t, err)

	// Empty is fine
	_, _, err = repo.List(ctx, models.ListParams{Limit: 10, SortBy: ""})
	assert.NoError(t, err)

	// Invalid
	_, _, err = repo.List(ctx, models.ListParams{Limit: 10, SortBy: "DROP TABLE"})
	assert.Error(t, err)
}
