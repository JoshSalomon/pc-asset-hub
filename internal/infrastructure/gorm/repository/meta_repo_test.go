package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/infrastructure/gorm/repository"
	"github.com/project-catalyst/pc-asset-hub/internal/infrastructure/gorm/testutil"
)

func newID() string {
	return uuid.Must(uuid.NewV7()).String()
}

// === Entity Types and Versions (T-1.01 through T-1.08) ===

func TestT1_01_CreateEntityType(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewEntityTypeGormRepo(db)
	ctx := context.Background()

	et := &models.EntityType{
		ID:        newID(),
		Name:      "Model",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err := repo.Create(ctx, et)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, et.ID)
	require.NoError(t, err)
	assert.Equal(t, "Model", found.Name)
	assert.False(t, found.CreatedAt.IsZero())
}

func TestT1_02_CreateEntityTypeVersion(t *testing.T) {
	db := testutil.NewTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	ctx := context.Background()

	et := &models.EntityType{ID: newID(), Name: "Model", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, etRepo.Create(ctx, et))

	etv := &models.EntityTypeVersion{
		ID:           newID(),
		EntityTypeID: et.ID,
		Version:      1,
		Description:  "Initial version",
		CreatedAt:    time.Now(),
	}
	err := etvRepo.Create(ctx, etv)
	require.NoError(t, err)

	found, err := etvRepo.GetByEntityTypeAndVersion(ctx, et.ID, 1)
	require.NoError(t, err)
	assert.Equal(t, 1, found.Version)
}

func TestT1_03_CreateDuplicateEntityTypeName(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewEntityTypeGormRepo(db)
	ctx := context.Background()

	et1 := &models.EntityType{ID: newID(), Name: "Model", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, repo.Create(ctx, et1))

	et2 := &models.EntityType{ID: newID(), Name: "Model", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	err := repo.Create(ctx, et2)
	require.Error(t, err)
	assert.True(t, domainerrors.IsConflict(err))
}

func TestT1_04_CreateSecondVersion(t *testing.T) {
	db := testutil.NewTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	ctx := context.Background()

	et := &models.EntityType{ID: newID(), Name: "Model", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, etRepo.Create(ctx, et))

	v1 := &models.EntityTypeVersion{ID: newID(), EntityTypeID: et.ID, Version: 1, Description: "V1", CreatedAt: time.Now()}
	require.NoError(t, etvRepo.Create(ctx, v1))

	v2 := &models.EntityTypeVersion{ID: newID(), EntityTypeID: et.ID, Version: 2, Description: "V2", CreatedAt: time.Now()}
	require.NoError(t, etvRepo.Create(ctx, v2))

	// V1 still intact
	foundV1, err := etvRepo.GetByEntityTypeAndVersion(ctx, et.ID, 1)
	require.NoError(t, err)
	assert.Equal(t, "V1", foundV1.Description)

	foundV2, err := etvRepo.GetByEntityTypeAndVersion(ctx, et.ID, 2)
	require.NoError(t, err)
	assert.Equal(t, "V2", foundV2.Description)
}

func TestT1_05_ListEntityTypesWithPagination(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewEntityTypeGormRepo(db)
	ctx := context.Background()

	for _, name := range []string{"Alpha", "Beta", "Charlie", "Delta"} {
		require.NoError(t, repo.Create(ctx, &models.EntityType{ID: newID(), Name: name, CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	}

	items, total, err := repo.List(ctx, models.ListParams{Limit: 2, Offset: 0})
	require.NoError(t, err)
	assert.Equal(t, 4, total)
	assert.Len(t, items, 2)
	assert.Equal(t, "Alpha", items[0].Name)
}

func TestT1_06_GetEntityTypeByID(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewEntityTypeGormRepo(db)
	ctx := context.Background()

	et := &models.EntityType{ID: newID(), Name: "Model", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, repo.Create(ctx, et))

	found, err := repo.GetByID(ctx, et.ID)
	require.NoError(t, err)
	assert.Equal(t, et.Name, found.Name)
}

func TestT1_07_GetEntityTypeByName(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewEntityTypeGormRepo(db)
	ctx := context.Background()

	et := &models.EntityType{ID: newID(), Name: "Model", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, repo.Create(ctx, et))

	found, err := repo.GetByName(ctx, "Model")
	require.NoError(t, err)
	assert.Equal(t, et.ID, found.ID)
}

func TestT1_08_DeleteEntityTypeCascades(t *testing.T) {
	db := testutil.NewTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	ctx := context.Background()

	et := &models.EntityType{ID: newID(), Name: "Model", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, etRepo.Create(ctx, et))
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: newID(), EntityTypeID: et.ID, Version: 1, CreatedAt: time.Now()}))

	require.NoError(t, etRepo.Delete(ctx, et.ID))

	_, err := etvRepo.GetByEntityTypeAndVersion(ctx, et.ID, 1)
	assert.True(t, domainerrors.IsNotFound(err))
}

// === Enums (T-1.09 through T-1.14) ===

func TestT1_09_CreateEnumWithValues(t *testing.T) {
	db := testutil.NewTestDB(t)
	enumRepo := repository.NewEnumGormRepo(db)
	evRepo := repository.NewEnumValueGormRepo(db)
	ctx := context.Background()

	e := &models.Enum{ID: newID(), Name: "Status", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, enumRepo.Create(ctx, e))

	for i, v := range []string{"active", "inactive", "deprecated"} {
		require.NoError(t, evRepo.Create(ctx, &models.EnumValue{ID: newID(), EnumID: e.ID, Value: v, Ordinal: i}))
	}

	values, err := evRepo.ListByEnum(ctx, e.ID)
	require.NoError(t, err)
	assert.Len(t, values, 3)
	assert.Equal(t, "active", values[0].Value)
	assert.Equal(t, 0, values[0].Ordinal)
}

func TestT1_10_CreateDuplicateEnumName(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewEnumGormRepo(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &models.Enum{ID: newID(), Name: "Status", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	err := repo.Create(ctx, &models.Enum{ID: newID(), Name: "Status", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	assert.True(t, domainerrors.IsConflict(err))
}

func TestT1_11_AddValueToEnum(t *testing.T) {
	db := testutil.NewTestDB(t)
	enumRepo := repository.NewEnumGormRepo(db)
	evRepo := repository.NewEnumValueGormRepo(db)
	ctx := context.Background()

	e := &models.Enum{ID: newID(), Name: "Status", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, enumRepo.Create(ctx, e))
	require.NoError(t, evRepo.Create(ctx, &models.EnumValue{ID: newID(), EnumID: e.ID, Value: "active", Ordinal: 0}))
	require.NoError(t, evRepo.Create(ctx, &models.EnumValue{ID: newID(), EnumID: e.ID, Value: "inactive", Ordinal: 1}))

	values, err := evRepo.ListByEnum(ctx, e.ID)
	require.NoError(t, err)
	assert.Len(t, values, 2)
}

func TestT1_12_RemoveValueFromEnum(t *testing.T) {
	db := testutil.NewTestDB(t)
	enumRepo := repository.NewEnumGormRepo(db)
	evRepo := repository.NewEnumValueGormRepo(db)
	ctx := context.Background()

	e := &models.Enum{ID: newID(), Name: "Status", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, enumRepo.Create(ctx, e))

	v1ID := newID()
	require.NoError(t, evRepo.Create(ctx, &models.EnumValue{ID: v1ID, EnumID: e.ID, Value: "active", Ordinal: 0}))
	require.NoError(t, evRepo.Create(ctx, &models.EnumValue{ID: newID(), EnumID: e.ID, Value: "inactive", Ordinal: 1}))

	require.NoError(t, evRepo.Delete(ctx, v1ID))
	values, err := evRepo.ListByEnum(ctx, e.ID)
	require.NoError(t, err)
	assert.Len(t, values, 1)
	assert.Equal(t, "inactive", values[0].Value)
}

func TestT1_13_ReorderEnumValues(t *testing.T) {
	db := testutil.NewTestDB(t)
	enumRepo := repository.NewEnumGormRepo(db)
	evRepo := repository.NewEnumValueGormRepo(db)
	ctx := context.Background()

	e := &models.Enum{ID: newID(), Name: "Status", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, enumRepo.Create(ctx, e))

	id1, id2 := newID(), newID()
	require.NoError(t, evRepo.Create(ctx, &models.EnumValue{ID: id1, EnumID: e.ID, Value: "active", Ordinal: 0}))
	require.NoError(t, evRepo.Create(ctx, &models.EnumValue{ID: id2, EnumID: e.ID, Value: "inactive", Ordinal: 1}))

	// Reverse order
	require.NoError(t, evRepo.Reorder(ctx, e.ID, []string{id2, id1}))
	values, err := evRepo.ListByEnum(ctx, e.ID)
	require.NoError(t, err)
	assert.Equal(t, "inactive", values[0].Value)
	assert.Equal(t, "active", values[1].Value)
}

func TestT1_14_CreateDuplicateEnumValue(t *testing.T) {
	db := testutil.NewTestDB(t)
	enumRepo := repository.NewEnumGormRepo(db)
	evRepo := repository.NewEnumValueGormRepo(db)
	ctx := context.Background()

	e := &models.Enum{ID: newID(), Name: "Status", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, enumRepo.Create(ctx, e))
	require.NoError(t, evRepo.Create(ctx, &models.EnumValue{ID: newID(), EnumID: e.ID, Value: "active", Ordinal: 0}))

	err := evRepo.Create(ctx, &models.EnumValue{ID: newID(), EnumID: e.ID, Value: "active", Ordinal: 1})
	assert.Error(t, err)
	assert.True(t, domainerrors.IsConflict(err))
}

// TD-59: GetLatestByEntityTypes batch query
func TestTD59_GetLatestByEntityTypes(t *testing.T) {
	db := testutil.NewTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	ctx := context.Background()

	now := time.Now()
	et1 := &models.EntityType{ID: newID(), Name: "Server", CreatedAt: now, UpdatedAt: now}
	et2 := &models.EntityType{ID: newID(), Name: "Tool", CreatedAt: now, UpdatedAt: now}
	require.NoError(t, etRepo.Create(ctx, et1))
	require.NoError(t, etRepo.Create(ctx, et2))

	// Server: V1 and V2
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: newID(), EntityTypeID: et1.ID, Version: 1, Description: "V1", CreatedAt: now}))
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: newID(), EntityTypeID: et1.ID, Version: 2, Description: "V2", CreatedAt: now}))
	// Tool: V1 only
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: newID(), EntityTypeID: et2.ID, Version: 1, Description: "Tool V1", CreatedAt: now}))

	result, err := etvRepo.GetLatestByEntityTypes(ctx, []string{et1.ID, et2.ID})
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "V2", result[et1.ID].Description)
	assert.Equal(t, 2, result[et1.ID].Version)
	assert.Equal(t, "Tool V1", result[et2.ID].Description)

	// Empty input
	empty, err := etvRepo.GetLatestByEntityTypes(ctx, []string{})
	require.NoError(t, err)
	assert.Empty(t, empty)
}

// === Attributes (T-1.15 through T-1.23) ===

func TestT1_15_CreateAttributeString(t *testing.T) {
	db := testutil.NewTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	attrRepo := repository.NewAttributeGormRepo(db)
	ctx := context.Background()

	etID := newID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: etID, Name: "Model", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	etvID := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID, EntityTypeID: etID, Version: 1, CreatedAt: time.Now()}))

	attr := &models.Attribute{ID: newID(), EntityTypeVersionID: etvID, Name: "endpoint", Description: "API endpoint", Type: models.AttributeTypeString, Ordinal: 0}
	require.NoError(t, attrRepo.Create(ctx, attr))

	found, err := attrRepo.GetByID(ctx, attr.ID)
	require.NoError(t, err)
	assert.Equal(t, models.AttributeTypeString, found.Type)
}

func TestT1_16_CreateAttributeNumber(t *testing.T) {
	db := testutil.NewTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	attrRepo := repository.NewAttributeGormRepo(db)
	ctx := context.Background()

	etID := newID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: etID, Name: "Model", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	etvID := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID, EntityTypeID: etID, Version: 1, CreatedAt: time.Now()}))

	attr := &models.Attribute{ID: newID(), EntityTypeVersionID: etvID, Name: "max_tokens", Type: models.AttributeTypeNumber, Ordinal: 0}
	require.NoError(t, attrRepo.Create(ctx, attr))

	found, err := attrRepo.GetByID(ctx, attr.ID)
	require.NoError(t, err)
	assert.Equal(t, models.AttributeTypeNumber, found.Type)
}

func TestT1_17_CreateAttributeEnum(t *testing.T) {
	db := testutil.NewTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	attrRepo := repository.NewAttributeGormRepo(db)
	enumRepo := repository.NewEnumGormRepo(db)
	ctx := context.Background()

	enumID := newID()
	require.NoError(t, enumRepo.Create(ctx, &models.Enum{ID: enumID, Name: "Status", CreatedAt: time.Now(), UpdatedAt: time.Now()}))

	etID := newID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: etID, Name: "Model", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	etvID := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID, EntityTypeID: etID, Version: 1, CreatedAt: time.Now()}))

	attr := &models.Attribute{ID: newID(), EntityTypeVersionID: etvID, Name: "status", Type: models.AttributeTypeEnum, EnumID: enumID, Ordinal: 0}
	require.NoError(t, attrRepo.Create(ctx, attr))

	found, err := attrRepo.GetByID(ctx, attr.ID)
	require.NoError(t, err)
	assert.Equal(t, models.AttributeTypeEnum, found.Type)
	assert.Equal(t, enumID, found.EnumID)
}

func TestT1_18_CreateDuplicateAttributeName(t *testing.T) {
	db := testutil.NewTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	attrRepo := repository.NewAttributeGormRepo(db)
	ctx := context.Background()

	etID := newID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: etID, Name: "Model", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	etvID := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID, EntityTypeID: etID, Version: 1, CreatedAt: time.Now()}))

	require.NoError(t, attrRepo.Create(ctx, &models.Attribute{ID: newID(), EntityTypeVersionID: etvID, Name: "endpoint", Type: models.AttributeTypeString, Ordinal: 0}))
	err := attrRepo.Create(ctx, &models.Attribute{ID: newID(), EntityTypeVersionID: etvID, Name: "endpoint", Type: models.AttributeTypeString, Ordinal: 1})
	assert.True(t, domainerrors.IsConflict(err))
}

func TestT1_19_SameAttributeNameDifferentVersions(t *testing.T) {
	db := testutil.NewTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	attrRepo := repository.NewAttributeGormRepo(db)
	ctx := context.Background()

	etID := newID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: etID, Name: "Model", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	etvID1 := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID1, EntityTypeID: etID, Version: 1, CreatedAt: time.Now()}))
	etvID2 := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID2, EntityTypeID: etID, Version: 2, CreatedAt: time.Now()}))

	require.NoError(t, attrRepo.Create(ctx, &models.Attribute{ID: newID(), EntityTypeVersionID: etvID1, Name: "endpoint", Type: models.AttributeTypeString, Ordinal: 0}))
	err := attrRepo.Create(ctx, &models.Attribute{ID: newID(), EntityTypeVersionID: etvID2, Name: "endpoint", Type: models.AttributeTypeString, Ordinal: 0})
	assert.NoError(t, err) // Different versions, same name should be allowed
}

func TestT1_20_ReorderAttributes(t *testing.T) {
	db := testutil.NewTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	attrRepo := repository.NewAttributeGormRepo(db)
	ctx := context.Background()

	etID := newID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: etID, Name: "Model", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	etvID := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID, EntityTypeID: etID, Version: 1, CreatedAt: time.Now()}))

	id1, id2, id3 := newID(), newID(), newID()
	require.NoError(t, attrRepo.Create(ctx, &models.Attribute{ID: id1, EntityTypeVersionID: etvID, Name: "a", Type: models.AttributeTypeString, Ordinal: 0}))
	require.NoError(t, attrRepo.Create(ctx, &models.Attribute{ID: id2, EntityTypeVersionID: etvID, Name: "b", Type: models.AttributeTypeString, Ordinal: 1}))
	require.NoError(t, attrRepo.Create(ctx, &models.Attribute{ID: id3, EntityTypeVersionID: etvID, Name: "c", Type: models.AttributeTypeString, Ordinal: 2}))

	require.NoError(t, attrRepo.Reorder(ctx, etvID, []string{id3, id1, id2}))

	attrs, err := attrRepo.ListByVersion(ctx, etvID)
	require.NoError(t, err)
	assert.Equal(t, "c", attrs[0].Name)
	assert.Equal(t, "a", attrs[1].Name)
	assert.Equal(t, "b", attrs[2].Name)
}

func TestT1_21_BulkCopyAttributes(t *testing.T) {
	db := testutil.NewTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	attrRepo := repository.NewAttributeGormRepo(db)
	ctx := context.Background()

	etID := newID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: etID, Name: "Model", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	v1ID := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: v1ID, EntityTypeID: etID, Version: 1, CreatedAt: time.Now()}))
	v2ID := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: v2ID, EntityTypeID: etID, Version: 2, CreatedAt: time.Now()}))

	require.NoError(t, attrRepo.Create(ctx, &models.Attribute{ID: newID(), EntityTypeVersionID: v1ID, Name: "a", Type: models.AttributeTypeString, Ordinal: 0}))
	require.NoError(t, attrRepo.Create(ctx, &models.Attribute{ID: newID(), EntityTypeVersionID: v1ID, Name: "b", Type: models.AttributeTypeNumber, Ordinal: 1}))

	require.NoError(t, attrRepo.BulkCopyToVersion(ctx, v1ID, v2ID))

	v2Attrs, err := attrRepo.ListByVersion(ctx, v2ID)
	require.NoError(t, err)
	assert.Len(t, v2Attrs, 2)
	// Should have different IDs
	v1Attrs, _ := attrRepo.ListByVersion(ctx, v1ID)
	assert.NotEqual(t, v1Attrs[0].ID, v2Attrs[0].ID)
}

func TestT1_22_DeleteEnumReferencedByAttribute(t *testing.T) {
	db := testutil.NewTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	attrRepo := repository.NewAttributeGormRepo(db)
	enumRepo := repository.NewEnumGormRepo(db)
	ctx := context.Background()

	enumID := newID()
	require.NoError(t, enumRepo.Create(ctx, &models.Enum{ID: enumID, Name: "Status", CreatedAt: time.Now(), UpdatedAt: time.Now()}))

	etID := newID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: etID, Name: "Model", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	etvID := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID, EntityTypeID: etID, Version: 1, CreatedAt: time.Now()}))
	require.NoError(t, attrRepo.Create(ctx, &models.Attribute{ID: newID(), EntityTypeVersionID: etvID, Name: "status", Type: models.AttributeTypeEnum, EnumID: enumID, Ordinal: 0}))

	err := enumRepo.Delete(ctx, enumID)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsReferencedEnum(err))
}

func TestT1_23_ListAttributesByVersion(t *testing.T) {
	db := testutil.NewTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	attrRepo := repository.NewAttributeGormRepo(db)
	ctx := context.Background()

	etID := newID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: etID, Name: "Model", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	v1ID := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: v1ID, EntityTypeID: etID, Version: 1, CreatedAt: time.Now()}))
	v2ID := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: v2ID, EntityTypeID: etID, Version: 2, CreatedAt: time.Now()}))

	require.NoError(t, attrRepo.Create(ctx, &models.Attribute{ID: newID(), EntityTypeVersionID: v1ID, Name: "a", Type: models.AttributeTypeString, Ordinal: 0}))
	require.NoError(t, attrRepo.Create(ctx, &models.Attribute{ID: newID(), EntityTypeVersionID: v2ID, Name: "b", Type: models.AttributeTypeString, Ordinal: 0}))

	v1Attrs, err := attrRepo.ListByVersion(ctx, v1ID)
	require.NoError(t, err)
	assert.Len(t, v1Attrs, 1)
	assert.Equal(t, "a", v1Attrs[0].Name)

	v2Attrs, err := attrRepo.ListByVersion(ctx, v2ID)
	require.NoError(t, err)
	assert.Len(t, v2Attrs, 1)
	assert.Equal(t, "b", v2Attrs[0].Name)
}

// === Associations (T-1.24 through T-1.29) ===

func TestT1_24_CreateContainmentAssociation(t *testing.T) {
	db := testutil.NewTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	assocRepo := repository.NewAssociationGormRepo(db)
	ctx := context.Background()

	et1ID, et2ID := newID(), newID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: et1ID, Name: "MCPServer", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: et2ID, Name: "Tool", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	etvID := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID, EntityTypeID: et1ID, Version: 1, CreatedAt: time.Now()}))

	assoc := &models.Association{ID: newID(), EntityTypeVersionID: etvID, TargetEntityTypeID: et2ID, Type: models.AssociationTypeContainment, SourceRole: "contains", TargetRole: "part_of", CreatedAt: time.Now()}
	require.NoError(t, assocRepo.Create(ctx, assoc))

	found, err := assocRepo.GetByID(ctx, assoc.ID)
	require.NoError(t, err)
	assert.Equal(t, models.AssociationTypeContainment, found.Type)
	assert.Equal(t, "contains", found.SourceRole)
}

func TestT1_25_CreateDirectionalReference(t *testing.T) {
	db := testutil.NewTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	assocRepo := repository.NewAssociationGormRepo(db)
	ctx := context.Background()

	et1ID, et2ID := newID(), newID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: et1ID, Name: "Tool", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: et2ID, Name: "Model", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	etvID := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID, EntityTypeID: et1ID, Version: 1, CreatedAt: time.Now()}))

	assoc := &models.Association{ID: newID(), EntityTypeVersionID: etvID, TargetEntityTypeID: et2ID, Type: models.AssociationTypeDirectional, SourceRole: "refers_to", TargetRole: "referred_by", CreatedAt: time.Now()}
	require.NoError(t, assocRepo.Create(ctx, assoc))

	found, err := assocRepo.GetByID(ctx, assoc.ID)
	require.NoError(t, err)
	assert.Equal(t, models.AssociationTypeDirectional, found.Type)
}

func TestT1_26_CreateBidirectionalReference(t *testing.T) {
	db := testutil.NewTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	assocRepo := repository.NewAssociationGormRepo(db)
	ctx := context.Background()

	et1ID, et2ID := newID(), newID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: et1ID, Name: "Guardrail", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: et2ID, Name: "Evaluator", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	etvID := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID, EntityTypeID: et1ID, Version: 1, CreatedAt: time.Now()}))

	assoc := &models.Association{ID: newID(), EntityTypeVersionID: etvID, TargetEntityTypeID: et2ID, Type: models.AssociationTypeBidirectional, CreatedAt: time.Now()}
	require.NoError(t, assocRepo.Create(ctx, assoc))

	found, err := assocRepo.GetByID(ctx, assoc.ID)
	require.NoError(t, err)
	assert.Equal(t, models.AssociationTypeBidirectional, found.Type)
}

func TestT1_27_BulkCopyAssociations(t *testing.T) {
	db := testutil.NewTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	assocRepo := repository.NewAssociationGormRepo(db)
	ctx := context.Background()

	et1ID, et2ID := newID(), newID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: et1ID, Name: "MCPServer", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: et2ID, Name: "Tool", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	v1ID := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: v1ID, EntityTypeID: et1ID, Version: 1, CreatedAt: time.Now()}))
	v2ID := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: v2ID, EntityTypeID: et1ID, Version: 2, CreatedAt: time.Now()}))

	require.NoError(t, assocRepo.Create(ctx, &models.Association{ID: newID(), EntityTypeVersionID: v1ID, TargetEntityTypeID: et2ID, Type: models.AssociationTypeContainment, CreatedAt: time.Now()}))

	require.NoError(t, assocRepo.BulkCopyToVersion(ctx, v1ID, v2ID))

	v2Assocs, err := assocRepo.ListByVersion(ctx, v2ID)
	require.NoError(t, err)
	assert.Len(t, v2Assocs, 1)

	v1Assocs, _ := assocRepo.ListByVersion(ctx, v1ID)
	assert.NotEqual(t, v1Assocs[0].ID, v2Assocs[0].ID) // Different IDs
}

func TestT1_28_ListAssociationsByVersion(t *testing.T) {
	db := testutil.NewTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	assocRepo := repository.NewAssociationGormRepo(db)
	ctx := context.Background()

	et1ID, et2ID := newID(), newID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: et1ID, Name: "MCPServer", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: et2ID, Name: "Tool", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	etvID := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID, EntityTypeID: et1ID, Version: 1, CreatedAt: time.Now()}))

	require.NoError(t, assocRepo.Create(ctx, &models.Association{ID: newID(), EntityTypeVersionID: etvID, TargetEntityTypeID: et2ID, Type: models.AssociationTypeContainment, CreatedAt: time.Now()}))

	assocs, err := assocRepo.ListByVersion(ctx, etvID)
	require.NoError(t, err)
	assert.Len(t, assocs, 1)

	// Different version returns empty
	otherEtvID := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: otherEtvID, EntityTypeID: et1ID, Version: 2, CreatedAt: time.Now()}))
	assocs2, err := assocRepo.ListByVersion(ctx, otherEtvID)
	require.NoError(t, err)
	assert.Len(t, assocs2, 0)
}

func TestListByTargetEntityType(t *testing.T) {
	db := testutil.NewTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	assocRepo := repository.NewAssociationGormRepo(db)
	ctx := context.Background()

	et1ID, et2ID, et3ID := newID(), newID(), newID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: et1ID, Name: "A", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: et2ID, Name: "B", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: et3ID, Name: "C", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	etvID1 := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID1, EntityTypeID: et1ID, Version: 1, CreatedAt: time.Now()}))
	etvID3 := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID3, EntityTypeID: et3ID, Version: 1, CreatedAt: time.Now()}))

	// A targets B, C targets B
	require.NoError(t, assocRepo.Create(ctx, &models.Association{ID: newID(), EntityTypeVersionID: etvID1, TargetEntityTypeID: et2ID, Type: models.AssociationTypeContainment, CreatedAt: time.Now()}))
	require.NoError(t, assocRepo.Create(ctx, &models.Association{ID: newID(), EntityTypeVersionID: etvID3, TargetEntityTypeID: et2ID, Type: models.AssociationTypeDirectional, CreatedAt: time.Now()}))

	// Query by target B — should return both
	assocs, err := assocRepo.ListByTargetEntityType(ctx, et2ID)
	require.NoError(t, err)
	assert.Len(t, assocs, 2)

	// Query by target A — should return empty
	assocsA, err := assocRepo.ListByTargetEntityType(ctx, et1ID)
	require.NoError(t, err)
	assert.Len(t, assocsA, 0)
}

func TestT1_29_DeleteAssociation(t *testing.T) {
	db := testutil.NewTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	assocRepo := repository.NewAssociationGormRepo(db)
	ctx := context.Background()

	et1ID, et2ID := newID(), newID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: et1ID, Name: "MCPServer", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: et2ID, Name: "Tool", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	etvID := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID, EntityTypeID: et1ID, Version: 1, CreatedAt: time.Now()}))

	assocID := newID()
	require.NoError(t, assocRepo.Create(ctx, &models.Association{ID: assocID, EntityTypeVersionID: etvID, TargetEntityTypeID: et2ID, Type: models.AssociationTypeContainment, CreatedAt: time.Now()}))

	require.NoError(t, assocRepo.Delete(ctx, assocID))
	_, err := assocRepo.GetByID(ctx, assocID)
	assert.True(t, domainerrors.IsNotFound(err))
}

// === Catalog Versions (T-1.30 through T-1.39) ===

func TestT1_30_CreateCatalogVersion(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewCatalogVersionGormRepo(db)
	ctx := context.Background()

	cv := &models.CatalogVersion{ID: newID(), VersionLabel: "v1.0", LifecycleStage: models.LifecycleStageDevelopment, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, repo.Create(ctx, cv))

	found, err := repo.GetByID(ctx, cv.ID)
	require.NoError(t, err)
	assert.Equal(t, models.LifecycleStageDevelopment, found.LifecycleStage)
}

func TestT1_31_CreateDuplicateCatalogVersionLabel(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := repository.NewCatalogVersionGormRepo(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &models.CatalogVersion{ID: newID(), VersionLabel: "v1.0", LifecycleStage: models.LifecycleStageDevelopment, CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	err := repo.Create(ctx, &models.CatalogVersion{ID: newID(), VersionLabel: "v1.0", LifecycleStage: models.LifecycleStageDevelopment, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	assert.True(t, domainerrors.IsConflict(err))
}

func TestT1_32_AddPin(t *testing.T) {
	db := testutil.NewTestDB(t)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	pinRepo := repository.NewCatalogVersionPinGormRepo(db)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	ctx := context.Background()

	cvID := newID()
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{ID: cvID, VersionLabel: "v1.0", LifecycleStage: models.LifecycleStageDevelopment, CreatedAt: time.Now(), UpdatedAt: time.Now()}))

	etID := newID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: etID, Name: "Model", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	etvID := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID, EntityTypeID: etID, Version: 1, CreatedAt: time.Now()}))

	pin := &models.CatalogVersionPin{ID: newID(), CatalogVersionID: cvID, EntityTypeVersionID: etvID}
	require.NoError(t, pinRepo.Create(ctx, pin))

	pins, err := pinRepo.ListByCatalogVersion(ctx, cvID)
	require.NoError(t, err)
	assert.Len(t, pins, 1)
}

func TestT1_33_AddDuplicatePin(t *testing.T) {
	db := testutil.NewTestDB(t)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	pinRepo := repository.NewCatalogVersionPinGormRepo(db)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	ctx := context.Background()

	cvID := newID()
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{ID: cvID, VersionLabel: "v1.0", LifecycleStage: models.LifecycleStageDevelopment, CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	etID := newID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: etID, Name: "Model", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	etvID := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID, EntityTypeID: etID, Version: 1, CreatedAt: time.Now()}))

	require.NoError(t, pinRepo.Create(ctx, &models.CatalogVersionPin{ID: newID(), CatalogVersionID: cvID, EntityTypeVersionID: etvID}))
	err := pinRepo.Create(ctx, &models.CatalogVersionPin{ID: newID(), CatalogVersionID: cvID, EntityTypeVersionID: etvID})
	assert.True(t, domainerrors.IsConflict(err))
}

func TestT1_34_TransitionDevToTesting(t *testing.T) {
	db := testutil.NewTestDB(t)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	ltRepo := repository.NewLifecycleTransitionGormRepo(db)
	ctx := context.Background()

	cvID := newID()
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{ID: cvID, VersionLabel: "v1.0", LifecycleStage: models.LifecycleStageDevelopment, CreatedAt: time.Now(), UpdatedAt: time.Now()}))

	require.NoError(t, cvRepo.UpdateLifecycle(ctx, cvID, models.LifecycleStageTesting))
	require.NoError(t, ltRepo.Create(ctx, &models.LifecycleTransition{ID: newID(), CatalogVersionID: cvID, FromStage: "development", ToStage: "testing", PerformedBy: "admin", PerformedAt: time.Now()}))

	found, err := cvRepo.GetByID(ctx, cvID)
	require.NoError(t, err)
	assert.Equal(t, models.LifecycleStageTesting, found.LifecycleStage)
}

func TestT1_35_TransitionTestingToProduction(t *testing.T) {
	db := testutil.NewTestDB(t)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	ltRepo := repository.NewLifecycleTransitionGormRepo(db)
	ctx := context.Background()

	cvID := newID()
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{ID: cvID, VersionLabel: "v1.0", LifecycleStage: models.LifecycleStageTesting, CreatedAt: time.Now(), UpdatedAt: time.Now()}))

	require.NoError(t, cvRepo.UpdateLifecycle(ctx, cvID, models.LifecycleStageProduction))
	require.NoError(t, ltRepo.Create(ctx, &models.LifecycleTransition{ID: newID(), CatalogVersionID: cvID, FromStage: "testing", ToStage: "production", PerformedBy: "admin", PerformedAt: time.Now()}))

	found, err := cvRepo.GetByID(ctx, cvID)
	require.NoError(t, err)
	assert.Equal(t, models.LifecycleStageProduction, found.LifecycleStage)
}

func TestT1_36_TransitionProductionToTesting(t *testing.T) {
	db := testutil.NewTestDB(t)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	ctx := context.Background()

	cvID := newID()
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{ID: cvID, VersionLabel: "v1.0", LifecycleStage: models.LifecycleStageProduction, CreatedAt: time.Now(), UpdatedAt: time.Now()}))

	require.NoError(t, cvRepo.UpdateLifecycle(ctx, cvID, models.LifecycleStageTesting))
	found, _ := cvRepo.GetByID(ctx, cvID)
	assert.Equal(t, models.LifecycleStageTesting, found.LifecycleStage)
}

func TestT1_37_TransitionProductionToDevelopment(t *testing.T) {
	db := testutil.NewTestDB(t)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	ctx := context.Background()

	cvID := newID()
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{ID: cvID, VersionLabel: "v1.0", LifecycleStage: models.LifecycleStageProduction, CreatedAt: time.Now(), UpdatedAt: time.Now()}))

	require.NoError(t, cvRepo.UpdateLifecycle(ctx, cvID, models.LifecycleStageDevelopment))
	found, _ := cvRepo.GetByID(ctx, cvID)
	assert.Equal(t, models.LifecycleStageDevelopment, found.LifecycleStage)
}

func TestT1_38_InvalidTransitionValidation(t *testing.T) {
	// Note: The repository layer doesn't enforce transition rules — that's the service layer's responsibility.
	// At the DB level, any lifecycle_stage string can be set. This test documents that the repo
	// stores whatever stage is given, and the service layer will validate transitions.
	db := testutil.NewTestDB(t)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	ctx := context.Background()

	cvID := newID()
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{ID: cvID, VersionLabel: "v1.0", LifecycleStage: models.LifecycleStageDevelopment, CreatedAt: time.Now(), UpdatedAt: time.Now()}))

	// At the repo level, this updates. The service layer will reject dev→prod.
	err := cvRepo.UpdateLifecycle(ctx, cvID, models.LifecycleStageProduction)
	assert.NoError(t, err) // Repo allows it; service validates
}

func TestListByEntityTypeVersionIDs(t *testing.T) {
	db := testutil.NewTestDB(t)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	pinRepo := repository.NewCatalogVersionPinGormRepo(db)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	ctx := context.Background()

	// Create a catalog version
	cvID := newID()
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{ID: cvID, VersionLabel: "v1.0", LifecycleStage: models.LifecycleStageDevelopment, CreatedAt: time.Now(), UpdatedAt: time.Now()}))

	// Create two entity types with versions
	et1ID, et2ID := newID(), newID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: et1ID, Name: "Model", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: et2ID, Name: "Tool", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	etv1ID, etv2ID := newID(), newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etv1ID, EntityTypeID: et1ID, Version: 1, CreatedAt: time.Now()}))
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etv2ID, EntityTypeID: et2ID, Version: 1, CreatedAt: time.Now()}))

	// Pin both entity type versions to the catalog version
	require.NoError(t, pinRepo.Create(ctx, &models.CatalogVersionPin{ID: newID(), CatalogVersionID: cvID, EntityTypeVersionID: etv1ID}))
	require.NoError(t, pinRepo.Create(ctx, &models.CatalogVersionPin{ID: newID(), CatalogVersionID: cvID, EntityTypeVersionID: etv2ID}))

	// Query by both entity type version IDs
	pins, err := pinRepo.ListByEntityTypeVersionIDs(ctx, []string{etv1ID, etv2ID})
	require.NoError(t, err)
	assert.Len(t, pins, 2)

	// Query by single entity type version ID
	pins, err = pinRepo.ListByEntityTypeVersionIDs(ctx, []string{etv1ID})
	require.NoError(t, err)
	assert.Len(t, pins, 1)
	assert.Equal(t, etv1ID, pins[0].EntityTypeVersionID)
}

func TestListByEntityTypeVersionIDs_Empty(t *testing.T) {
	db := testutil.NewTestDB(t)
	pinRepo := repository.NewCatalogVersionPinGormRepo(db)
	ctx := context.Background()

	// Query with empty slice returns empty result
	pins, err := pinRepo.ListByEntityTypeVersionIDs(ctx, []string{})
	require.NoError(t, err)
	assert.Len(t, pins, 0)
}

func TestListByEntityTypeVersionIDs_NoMatch(t *testing.T) {
	db := testutil.NewTestDB(t)
	pinRepo := repository.NewCatalogVersionPinGormRepo(db)
	ctx := context.Background()

	// Query with non-matching IDs returns empty result
	pins, err := pinRepo.ListByEntityTypeVersionIDs(ctx, []string{newID(), newID()})
	require.NoError(t, err)
	assert.Len(t, pins, 0)
}

func TestT1_39_ListTransitions(t *testing.T) {
	db := testutil.NewTestDB(t)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	ltRepo := repository.NewLifecycleTransitionGormRepo(db)
	ctx := context.Background()

	cvID := newID()
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{ID: cvID, VersionLabel: "v1.0", LifecycleStage: models.LifecycleStageDevelopment, CreatedAt: time.Now(), UpdatedAt: time.Now()}))

	t1 := time.Now()
	t2 := t1.Add(time.Minute)
	require.NoError(t, ltRepo.Create(ctx, &models.LifecycleTransition{ID: newID(), CatalogVersionID: cvID, FromStage: "", ToStage: "development", PerformedBy: "system", PerformedAt: t1}))
	require.NoError(t, ltRepo.Create(ctx, &models.LifecycleTransition{ID: newID(), CatalogVersionID: cvID, FromStage: "development", ToStage: "testing", PerformedBy: "admin", PerformedAt: t2}))

	transitions, err := ltRepo.ListByCatalogVersion(ctx, cvID)
	require.NoError(t, err)
	assert.Len(t, transitions, 2)
	assert.Equal(t, "development", transitions[0].ToStage)
	assert.Equal(t, "testing", transitions[1].ToStage)
}

// T-E.78: Create association with cardinality stores and retrieves correctly
func TestTE78_AssociationCardinalityStoredAndRetrieved(t *testing.T) {
	db := testutil.NewTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	assocRepo := repository.NewAssociationGormRepo(db)
	ctx := context.Background()

	et1ID, et2ID := newID(), newID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: et1ID, Name: "Server", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: et2ID, Name: "Tool", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	etvID := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID, EntityTypeID: et1ID, Version: 1, CreatedAt: time.Now()}))

	assoc := &models.Association{
		ID: newID(), EntityTypeVersionID: etvID, TargetEntityTypeID: et2ID,
		Type: models.AssociationTypeContainment, SourceRole: "contains", TargetRole: "part_of",
		SourceCardinality: "1", TargetCardinality: "0..n",
		CreatedAt: time.Now(),
	}
	require.NoError(t, assocRepo.Create(ctx, assoc))

	found, err := assocRepo.GetByID(ctx, assoc.ID)
	require.NoError(t, err)
	assert.Equal(t, "1", found.SourceCardinality)
	assert.Equal(t, "0..n", found.TargetCardinality)
}

// T-E.79: BulkCopyToVersion preserves cardinality on copied associations
func TestTE79_BulkCopyPreservesCardinality(t *testing.T) {
	db := testutil.NewTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	assocRepo := repository.NewAssociationGormRepo(db)
	ctx := context.Background()

	et1ID, et2ID := newID(), newID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: et1ID, Name: "Server", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: et2ID, Name: "Tool", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	v1ID := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: v1ID, EntityTypeID: et1ID, Version: 1, CreatedAt: time.Now()}))
	v2ID := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: v2ID, EntityTypeID: et1ID, Version: 2, CreatedAt: time.Now()}))

	require.NoError(t, assocRepo.Create(ctx, &models.Association{
		ID: newID(), EntityTypeVersionID: v1ID, TargetEntityTypeID: et2ID,
		Type: models.AssociationTypeContainment,
		SourceCardinality: "1..n", TargetCardinality: "0..1",
		CreatedAt: time.Now(),
	}))

	require.NoError(t, assocRepo.BulkCopyToVersion(ctx, v1ID, v2ID))

	v2Assocs, err := assocRepo.ListByVersion(ctx, v2ID)
	require.NoError(t, err)
	require.Len(t, v2Assocs, 1)
	assert.Equal(t, "1..n", v2Assocs[0].SourceCardinality)
	assert.Equal(t, "0..1", v2Assocs[0].TargetCardinality)
}

// T-E.114: Association name stored and retrieved
func TestTE114_AssociationNameStoredAndRetrieved(t *testing.T) {
	db := testutil.NewTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	assocRepo := repository.NewAssociationGormRepo(db)
	ctx := context.Background()

	et1ID, et2ID := newID(), newID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: et1ID, Name: "Server", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: et2ID, Name: "Tool", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	etvID := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID, EntityTypeID: et1ID, Version: 1, CreatedAt: time.Now()}))

	assoc := &models.Association{
		ID: newID(), EntityTypeVersionID: etvID, Name: "tools",
		TargetEntityTypeID: et2ID, Type: models.AssociationTypeContainment,
		CreatedAt: time.Now(),
	}
	require.NoError(t, assocRepo.Create(ctx, assoc))

	found, err := assocRepo.GetByID(ctx, assoc.ID)
	require.NoError(t, err)
	assert.Equal(t, "tools", found.Name)
}

// T-E.115: Association unique constraint on (version_id, name)
func TestTE115_AssociationNameUniqueConstraint(t *testing.T) {
	db := testutil.NewTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	assocRepo := repository.NewAssociationGormRepo(db)
	ctx := context.Background()

	et1ID, et2ID := newID(), newID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: et1ID, Name: "Server", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: et2ID, Name: "Tool", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	etvID := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID, EntityTypeID: et1ID, Version: 1, CreatedAt: time.Now()}))

	require.NoError(t, assocRepo.Create(ctx, &models.Association{
		ID: newID(), EntityTypeVersionID: etvID, Name: "tools",
		TargetEntityTypeID: et2ID, Type: models.AssociationTypeContainment, CreatedAt: time.Now(),
	}))

	// Duplicate name in same version should fail
	err := assocRepo.Create(ctx, &models.Association{
		ID: newID(), EntityTypeVersionID: etvID, Name: "tools",
		TargetEntityTypeID: et2ID, Type: models.AssociationTypeDirectional, CreatedAt: time.Now(),
	})
	assert.Error(t, err)
	assert.True(t, domainerrors.IsConflict(err))
}

// T-E.116: BulkCopyToVersion preserves association names
func TestTE116_BulkCopyPreservesAssociationNames(t *testing.T) {
	db := testutil.NewTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	assocRepo := repository.NewAssociationGormRepo(db)
	ctx := context.Background()

	et1ID, et2ID := newID(), newID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: et1ID, Name: "Server", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: et2ID, Name: "Tool", CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	v1ID, v2ID := newID(), newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: v1ID, EntityTypeID: et1ID, Version: 1, CreatedAt: time.Now()}))
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: v2ID, EntityTypeID: et1ID, Version: 2, CreatedAt: time.Now()}))

	require.NoError(t, assocRepo.Create(ctx, &models.Association{
		ID: newID(), EntityTypeVersionID: v1ID, Name: "my_tools",
		TargetEntityTypeID: et2ID, Type: models.AssociationTypeContainment, CreatedAt: time.Now(),
	}))

	require.NoError(t, assocRepo.BulkCopyToVersion(ctx, v1ID, v2ID))

	v2Assocs, err := assocRepo.ListByVersion(ctx, v2ID)
	require.NoError(t, err)
	require.Len(t, v2Assocs, 1)
	assert.Equal(t, "my_tools", v2Assocs[0].Name)
}
