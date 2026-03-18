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

func instTestID() string { return uuid.Must(uuid.NewV7()).String() }

// setupInstanceTestData creates the full chain: entity type → version → attributes → CV → pin → catalog
// Returns IDs needed for instance tests.
type instanceTestData struct {
	etID, etvID, cvID, catID string
	attrStringID, attrNumID  string
}

func setupInstanceTestData(t *testing.T) (*instanceTestData, *repository.EntityInstanceGormRepo, *repository.InstanceAttributeValueGormRepo) {
	t.Helper()
	db := testutil.NewTestDB(t)
	ctx := context.Background()

	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	attrRepo := repository.NewAttributeGormRepo(db)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	pinRepo := repository.NewCatalogVersionPinGormRepo(db)
	catalogRepo := repository.NewCatalogGormRepo(db)
	instRepo := repository.NewEntityInstanceGormRepo(db)
	iavRepo := repository.NewInstanceAttributeValueGormRepo(db)

	now := time.Now()
	data := &instanceTestData{
		etID: instTestID(), etvID: instTestID(), cvID: instTestID(), catID: instTestID(),
		attrStringID: instTestID(), attrNumID: instTestID(),
	}

	// Entity type + version
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: data.etID, Name: "model", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: data.etvID, EntityTypeID: data.etID, Version: 1, CreatedAt: now}))

	// Attributes
	require.NoError(t, attrRepo.Create(ctx, &models.Attribute{
		ID: data.attrStringID, EntityTypeVersionID: data.etvID, Name: "hostname", Type: models.AttributeTypeString, Ordinal: 1,
	}))
	require.NoError(t, attrRepo.Create(ctx, &models.Attribute{
		ID: data.attrNumID, EntityTypeVersionID: data.etvID, Name: "port", Type: models.AttributeTypeNumber, Ordinal: 2,
	}))

	// CV + pin
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{ID: data.cvID, VersionLabel: "v1", LifecycleStage: "development", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, pinRepo.Create(ctx, &models.CatalogVersionPin{ID: instTestID(), CatalogVersionID: data.cvID, EntityTypeVersionID: data.etvID}))

	// Catalog
	require.NoError(t, catalogRepo.Create(ctx, &models.Catalog{
		ID: data.catID, Name: "test-catalog", CatalogVersionID: data.cvID,
		ValidationStatus: models.ValidationStatusDraft, CreatedAt: now, UpdatedAt: now,
	}))

	return data, instRepo, iavRepo
}

// T-11.01: SetValues stores attribute values
func TestT11_01_SetValues(t *testing.T) {
	data, instRepo, iavRepo := setupInstanceTestData(t)
	ctx := context.Background()

	instID := instTestID()
	require.NoError(t, instRepo.Create(ctx, &models.EntityInstance{
		ID: instID, EntityTypeID: data.etID, CatalogID: data.catID,
		Name: "inst1", Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	values := []*models.InstanceAttributeValue{
		{ID: instTestID(), InstanceID: instID, InstanceVersion: 1, AttributeID: data.attrStringID, ValueString: "myhost"},
	}
	require.NoError(t, iavRepo.SetValues(ctx, values))

	got, err := iavRepo.GetCurrentValues(ctx, instID)
	require.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, "myhost", got[0].ValueString)
}

// T-11.02: GetCurrentValues returns latest version
func TestT11_02_GetCurrentValues(t *testing.T) {
	data, instRepo, iavRepo := setupInstanceTestData(t)
	ctx := context.Background()

	instID := instTestID()
	require.NoError(t, instRepo.Create(ctx, &models.EntityInstance{
		ID: instID, EntityTypeID: data.etID, CatalogID: data.catID,
		Name: "inst1", Version: 2, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	// Version 1 values
	require.NoError(t, iavRepo.SetValues(ctx, []*models.InstanceAttributeValue{
		{ID: instTestID(), InstanceID: instID, InstanceVersion: 1, AttributeID: data.attrStringID, ValueString: "old"},
	}))
	// Version 2 values
	require.NoError(t, iavRepo.SetValues(ctx, []*models.InstanceAttributeValue{
		{ID: instTestID(), InstanceID: instID, InstanceVersion: 2, AttributeID: data.attrStringID, ValueString: "new"},
	}))

	got, err := iavRepo.GetCurrentValues(ctx, instID)
	require.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, "new", got[0].ValueString)
}

// T-11.03: GetValuesForVersion returns specific version
func TestT11_03_GetValuesForVersion(t *testing.T) {
	data, instRepo, iavRepo := setupInstanceTestData(t)
	ctx := context.Background()

	instID := instTestID()
	require.NoError(t, instRepo.Create(ctx, &models.EntityInstance{
		ID: instID, EntityTypeID: data.etID, CatalogID: data.catID,
		Name: "inst1", Version: 2, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	require.NoError(t, iavRepo.SetValues(ctx, []*models.InstanceAttributeValue{
		{ID: instTestID(), InstanceID: instID, InstanceVersion: 1, AttributeID: data.attrStringID, ValueString: "v1val"},
	}))
	require.NoError(t, iavRepo.SetValues(ctx, []*models.InstanceAttributeValue{
		{ID: instTestID(), InstanceID: instID, InstanceVersion: 2, AttributeID: data.attrStringID, ValueString: "v2val"},
	}))

	got, err := iavRepo.GetValuesForVersion(ctx, instID, 1)
	require.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, "v1val", got[0].ValueString)
}

// T-11.04: Previous version values preserved after new version
func TestT11_04_VersionPreservation(t *testing.T) {
	data, instRepo, iavRepo := setupInstanceTestData(t)
	ctx := context.Background()

	instID := instTestID()
	require.NoError(t, instRepo.Create(ctx, &models.EntityInstance{
		ID: instID, EntityTypeID: data.etID, CatalogID: data.catID,
		Name: "inst1", Version: 2, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	require.NoError(t, iavRepo.SetValues(ctx, []*models.InstanceAttributeValue{
		{ID: instTestID(), InstanceID: instID, InstanceVersion: 1, AttributeID: data.attrStringID, ValueString: "original"},
	}))
	require.NoError(t, iavRepo.SetValues(ctx, []*models.InstanceAttributeValue{
		{ID: instTestID(), InstanceID: instID, InstanceVersion: 2, AttributeID: data.attrStringID, ValueString: "updated"},
	}))

	// Both versions should be independently retrievable
	v1, err := iavRepo.GetValuesForVersion(ctx, instID, 1)
	require.NoError(t, err)
	assert.Equal(t, "original", v1[0].ValueString)

	v2, err := iavRepo.GetValuesForVersion(ctx, instID, 2)
	require.NoError(t, err)
	assert.Equal(t, "updated", v2[0].ValueString)
}

// T-11.05: Instance creation with catalog_id and attribute values end-to-end
func TestT11_05_EndToEndInstanceWithValues(t *testing.T) {
	data, instRepo, iavRepo := setupInstanceTestData(t)
	ctx := context.Background()

	instID := instTestID()
	num := float64(8080)
	require.NoError(t, instRepo.Create(ctx, &models.EntityInstance{
		ID: instID, EntityTypeID: data.etID, CatalogID: data.catID,
		Name: "full-inst", Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	require.NoError(t, iavRepo.SetValues(ctx, []*models.InstanceAttributeValue{
		{ID: instTestID(), InstanceID: instID, InstanceVersion: 1, AttributeID: data.attrStringID, ValueString: "host.example.com"},
		{ID: instTestID(), InstanceID: instID, InstanceVersion: 1, AttributeID: data.attrNumID, ValueNumber: &num},
	}))

	// Verify via GetByID + GetCurrentValues
	inst, err := instRepo.GetByID(ctx, instID)
	require.NoError(t, err)
	assert.Equal(t, data.catID, inst.CatalogID)

	vals, err := iavRepo.GetCurrentValues(ctx, instID)
	require.NoError(t, err)
	assert.Len(t, vals, 2)
}

// T-11.06: Pin resolution chain
func TestT11_06_PinResolutionChain(t *testing.T) {
	db := testutil.NewTestDB(t)
	ctx := context.Background()
	now := time.Now()

	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	pinRepo := repository.NewCatalogVersionPinGormRepo(db)
	catalogRepo := repository.NewCatalogGormRepo(db)

	etID, etvID, cvID, catID := instTestID(), instTestID(), instTestID(), instTestID()

	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: etID, Name: "tool", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID, EntityTypeID: etID, Version: 2, CreatedAt: now}))
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{ID: cvID, VersionLabel: "v1", LifecycleStage: "development", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, pinRepo.Create(ctx, &models.CatalogVersionPin{ID: instTestID(), CatalogVersionID: cvID, EntityTypeVersionID: etvID}))
	require.NoError(t, catalogRepo.Create(ctx, &models.Catalog{
		ID: catID, Name: "pin-test", CatalogVersionID: cvID,
		ValidationStatus: models.ValidationStatusDraft, CreatedAt: now, UpdatedAt: now,
	}))

	// Resolve chain: catalog → CV → pin → entity type version
	cat, err := catalogRepo.GetByName(ctx, "pin-test")
	require.NoError(t, err)
	pins, err := pinRepo.ListByCatalogVersion(ctx, cat.CatalogVersionID)
	require.NoError(t, err)
	require.Len(t, pins, 1)
	etv, err := etvRepo.GetByID(ctx, pins[0].EntityTypeVersionID)
	require.NoError(t, err)
	assert.Equal(t, etID, etv.EntityTypeID)
	assert.Equal(t, 2, etv.Version)
}

// T-11.07: Pin resolution returns attributes for pinned version
func TestT11_07_PinResolutionAttributes(t *testing.T) {
	db := testutil.NewTestDB(t)
	ctx := context.Background()
	now := time.Now()

	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	attrRepo := repository.NewAttributeGormRepo(db)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	pinRepo := repository.NewCatalogVersionPinGormRepo(db)

	etID := instTestID()
	etv1ID, etv2ID := instTestID(), instTestID()
	cvID := instTestID()

	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: etID, Name: "svc", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etv1ID, EntityTypeID: etID, Version: 1, CreatedAt: now}))
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etv2ID, EntityTypeID: etID, Version: 2, CreatedAt: now}))

	// V1 has attr "name", V2 has attr "name" + "url"
	require.NoError(t, attrRepo.Create(ctx, &models.Attribute{ID: instTestID(), EntityTypeVersionID: etv1ID, Name: "name", Type: models.AttributeTypeString, Ordinal: 1}))
	require.NoError(t, attrRepo.Create(ctx, &models.Attribute{ID: instTestID(), EntityTypeVersionID: etv2ID, Name: "name", Type: models.AttributeTypeString, Ordinal: 1}))
	require.NoError(t, attrRepo.Create(ctx, &models.Attribute{ID: instTestID(), EntityTypeVersionID: etv2ID, Name: "url", Type: models.AttributeTypeString, Ordinal: 2}))

	// Pin V1 (not latest V2)
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{ID: cvID, VersionLabel: "v1", LifecycleStage: "development", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, pinRepo.Create(ctx, &models.CatalogVersionPin{ID: instTestID(), CatalogVersionID: cvID, EntityTypeVersionID: etv1ID}))

	// Resolve and get attributes — should get V1's attributes (1), not V2's (2)
	pins, _ := pinRepo.ListByCatalogVersion(ctx, cvID)
	attrs, err := attrRepo.ListByVersion(ctx, pins[0].EntityTypeVersionID)
	require.NoError(t, err)
	assert.Len(t, attrs, 1) // Only V1's "name", not V2's "name"+"url"
}

// T-11.08: Pin resolution for unpinned entity type
func TestT11_08_UnpinnedEntityType(t *testing.T) {
	db := testutil.NewTestDB(t)
	ctx := context.Background()
	now := time.Now()

	etRepo := repository.NewEntityTypeGormRepo(db)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	pinRepo := repository.NewCatalogVersionPinGormRepo(db)

	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: instTestID(), Name: "unpinned", CreatedAt: now, UpdatedAt: now}))
	cvID := instTestID()
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{ID: cvID, VersionLabel: "v1", LifecycleStage: "development", CreatedAt: now, UpdatedAt: now}))

	pins, err := pinRepo.ListByCatalogVersion(ctx, cvID)
	require.NoError(t, err)
	assert.Len(t, pins, 0) // No pins for this CV
}

// T-11.09: Optimistic locking — update with matching version
func TestT11_09_OptimisticLockingSuccess(t *testing.T) {
	data, instRepo, _ := setupInstanceTestData(t)
	ctx := context.Background()

	instID := instTestID()
	require.NoError(t, instRepo.Create(ctx, &models.EntityInstance{
		ID: instID, EntityTypeID: data.etID, CatalogID: data.catID,
		Name: "lock-test", Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	inst, _ := instRepo.GetByID(ctx, instID)
	inst.Version = 2
	inst.UpdatedAt = time.Now()
	require.NoError(t, instRepo.Update(ctx, inst))

	updated, _ := instRepo.GetByID(ctx, instID)
	assert.Equal(t, 2, updated.Version)
}

// T-11.10: Optimistic locking — concurrent update
func TestT11_10_OptimisticLockingConcurrent(t *testing.T) {
	data, instRepo, _ := setupInstanceTestData(t)
	ctx := context.Background()

	instID := instTestID()
	require.NoError(t, instRepo.Create(ctx, &models.EntityInstance{
		ID: instID, EntityTypeID: data.etID, CatalogID: data.catID,
		Name: "lock-test", Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	// Read twice (simulating two clients)
	inst1, _ := instRepo.GetByID(ctx, instID)
	inst2, _ := instRepo.GetByID(ctx, instID)

	// First update succeeds
	inst1.Version = 2
	inst1.UpdatedAt = time.Now()
	require.NoError(t, instRepo.Update(ctx, inst1))

	// Second update with stale version — the repo uses Save which doesn't check version
	// The optimistic locking is at the service layer, not the repo layer
	// So at the repo level, both updates succeed (repo doesn't enforce version checks)
	inst2.Version = 2
	inst2.UpdatedAt = time.Now()
	err := instRepo.Update(ctx, inst2)
	// This actually succeeds at the repo level — optimistic locking is service-level
	assert.NoError(t, err)
}

// === Coverage tests for repo uncovered lines ===

// Pagination: List with SortBy, SortDesc, and Offset
func TestCov_List_SortAndOffset(t *testing.T) {
	data, instRepo, _ := setupInstanceTestData(t)
	ctx := context.Background()

	for _, name := range []string{"alpha", "beta", "gamma"} {
		require.NoError(t, instRepo.Create(ctx, &models.EntityInstance{
			ID: instTestID(), EntityTypeID: data.etID, CatalogID: data.catID,
			Name: name, Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}))
	}

	// Sort by name descending with offset
	items, total, err := instRepo.List(ctx, data.etID, data.catID, models.ListParams{
		Limit:    2,
		Offset:   1,
		SortBy:   "name",
		SortDesc: true,
	})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, items, 2) // 3 total, offset 1, limit 2 = 2 items
}

// Pagination: ListByParent with Limit and Offset
func TestCov_ListByParent_Pagination(t *testing.T) {
	data, instRepo, _ := setupInstanceTestData(t)
	ctx := context.Background()

	parentID := instTestID()
	require.NoError(t, instRepo.Create(ctx, &models.EntityInstance{
		ID: parentID, EntityTypeID: data.etID, CatalogID: data.catID,
		Name: "parent", Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	for _, name := range []string{"child-a", "child-b", "child-c"} {
		require.NoError(t, instRepo.Create(ctx, &models.EntityInstance{
			ID: instTestID(), EntityTypeID: data.etID, CatalogID: data.catID,
			ParentInstanceID: parentID, Name: name, Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}))
	}

	items, total, err := instRepo.ListByParent(ctx, parentID, models.ListParams{Limit: 2, Offset: 1})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, items, 2)
}

// SoftDelete nonexistent instance
func TestCov_SoftDelete_NotFound(t *testing.T) {
	data, instRepo, _ := setupInstanceTestData(t)
	_ = data
	ctx := context.Background()

	err := instRepo.SoftDelete(ctx, "nonexistent-id")
	assert.Error(t, err)
}

// Catalog List with offset
func TestCov_CatalogList_Offset(t *testing.T) {
	db := testutil.NewTestDB(t)
	catalogRepo := repository.NewCatalogGormRepo(db)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	ctx := context.Background()

	cvID := instTestID()
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{
		ID: cvID, VersionLabel: "v1", LifecycleStage: "development",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	for _, name := range []string{"cat-a", "cat-b", "cat-c"} {
		require.NoError(t, catalogRepo.Create(ctx, &models.Catalog{
			ID: instTestID(), Name: name, CatalogVersionID: cvID,
			ValidationStatus: models.ValidationStatusDraft, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}))
	}

	items, total, err := catalogRepo.List(ctx, models.ListParams{Limit: 2, Offset: 1})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, items, 2)
}

// Bug: attribute filter JOIN must constrain to current instance version.
// Without the version constraint, filtering matches historical attribute values
// from previous versions, producing incorrect results.
func TestAttrFilter_OnlyMatchesCurrentVersion(t *testing.T) {
	data, instRepo, iavRepo := setupInstanceTestData(t)
	ctx := context.Background()

	// Create an instance at version 1 with hostname="old-host"
	instID := instTestID()
	require.NoError(t, instRepo.Create(ctx, &models.EntityInstance{
		ID: instID, EntityTypeID: data.etID, CatalogID: data.catID,
		Name: "versioned-inst", Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	require.NoError(t, iavRepo.SetValues(ctx, []*models.InstanceAttributeValue{
		{ID: instTestID(), InstanceID: instID, InstanceVersion: 1, AttributeID: data.attrStringID, ValueString: "old-host"},
	}))

	// Simulate update: bump to version 2 with hostname="new-host"
	require.NoError(t, instRepo.Update(ctx, &models.EntityInstance{
		ID: instID, EntityTypeID: data.etID, CatalogID: data.catID,
		Name: "versioned-inst", Version: 2, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	require.NoError(t, iavRepo.SetValues(ctx, []*models.InstanceAttributeValue{
		{ID: instTestID(), InstanceID: instID, InstanceVersion: 2, AttributeID: data.attrStringID, ValueString: "new-host"},
	}))

	// Filter by "old-host" — should NOT match (only version 1 had this value)
	items, total, err := instRepo.List(ctx, data.etID, data.catID, models.ListParams{
		Limit:   20,
		Filters: map[string]string{data.attrStringID: "old-host"},
	})
	require.NoError(t, err)
	assert.Equal(t, 0, total, "filter by old version's value should return 0 results")
	assert.Len(t, items, 0)

	// Filter by "new-host" — should match (current version 2 has this value)
	items, total, err = instRepo.List(ctx, data.etID, data.catID, models.ListParams{
		Limit:   20,
		Filters: map[string]string{data.attrStringID: "new-host"},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, total, "filter by current version's value should return 1 result")
	assert.Len(t, items, 1)
}
