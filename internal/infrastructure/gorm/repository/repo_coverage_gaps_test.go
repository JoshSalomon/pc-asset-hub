package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/infrastructure/gorm/repository"
	"github.com/project-catalyst/pc-asset-hub/internal/infrastructure/gorm/testutil"
)

// ============================================================
// #1: AssociationLinkGormRepo.GetByID (0% → covered)
// ============================================================

func TestAssocLinkRepo_GetByID_Success(t *testing.T) {
	db := testutil.NewTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	assocRepo := repository.NewAssociationGormRepo(db)
	linkRepo := repository.NewAssociationLinkGormRepo(db)
	instRepo := repository.NewEntityInstanceGormRepo(db)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	catalogRepo := repository.NewCatalogGormRepo(db)
	ctx := context.Background()
	now := time.Now()

	// Set up entity types
	et1ID, et2ID := newID(), newID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: et1ID, Name: "SrcType", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: et2ID, Name: "TgtType", CreatedAt: now, UpdatedAt: now}))
	etvID := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID, EntityTypeID: et1ID, Version: 1, CreatedAt: now}))

	// Set up CV + catalog for instances
	cvID := newID()
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{ID: cvID, VersionLabel: "link-v1", LifecycleStage: models.LifecycleStageDevelopment, CreatedAt: now, UpdatedAt: now}))
	catID := newID()
	require.NoError(t, catalogRepo.Create(ctx, &models.Catalog{ID: catID, Name: "link-cat", CatalogVersionID: cvID, ValidationStatus: models.ValidationStatusDraft, CreatedAt: now, UpdatedAt: now}))

	// Create instances
	srcInstID, tgtInstID := newID(), newID()
	require.NoError(t, instRepo.Create(ctx, &models.EntityInstance{ID: srcInstID, EntityTypeID: et1ID, CatalogID: catID, Name: "src-inst", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, instRepo.Create(ctx, &models.EntityInstance{ID: tgtInstID, EntityTypeID: et2ID, CatalogID: catID, Name: "tgt-inst", CreatedAt: now, UpdatedAt: now}))

	// Create association
	assocID := newID()
	require.NoError(t, assocRepo.Create(ctx, &models.Association{ID: assocID, EntityTypeVersionID: etvID, TargetEntityTypeID: et2ID, Type: models.AssociationTypeDirectional, CreatedAt: now}))

	// Create link
	linkID := newID()
	require.NoError(t, linkRepo.Create(ctx, &models.AssociationLink{ID: linkID, AssociationID: assocID, SourceInstanceID: srcInstID, TargetInstanceID: tgtInstID, CreatedAt: now}))

	// GetByID — success path
	found, err := linkRepo.GetByID(ctx, linkID)
	require.NoError(t, err)
	assert.Equal(t, linkID, found.ID)
	assert.Equal(t, assocID, found.AssociationID)
	assert.Equal(t, srcInstID, found.SourceInstanceID)
	assert.Equal(t, tgtInstID, found.TargetInstanceID)
}

func TestAssocLinkRepo_GetByID_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	linkRepo := repository.NewAssociationLinkGormRepo(db)
	ctx := context.Background()

	_, err := linkRepo.GetByID(ctx, "nonexistent-link-id")
	require.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
}

// ============================================================
// #2: AssociationLinkGormRepo.DeleteByInstance (0% → covered)
// ============================================================

func TestAssocLinkRepo_DeleteByInstance(t *testing.T) {
	db := testutil.NewTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	assocRepo := repository.NewAssociationGormRepo(db)
	linkRepo := repository.NewAssociationLinkGormRepo(db)
	instRepo := repository.NewEntityInstanceGormRepo(db)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	catalogRepo := repository.NewCatalogGormRepo(db)
	ctx := context.Background()
	now := time.Now()

	// Set up entity types
	et1ID, et2ID := newID(), newID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: et1ID, Name: "DelSrcType", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: et2ID, Name: "DelTgtType", CreatedAt: now, UpdatedAt: now}))
	etvID := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID, EntityTypeID: et1ID, Version: 1, CreatedAt: now}))

	// Set up CV + catalog
	cvID := newID()
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{ID: cvID, VersionLabel: "dbi-v1", LifecycleStage: models.LifecycleStageDevelopment, CreatedAt: now, UpdatedAt: now}))
	catID := newID()
	require.NoError(t, catalogRepo.Create(ctx, &models.Catalog{ID: catID, Name: "dbi-cat", CatalogVersionID: cvID, ValidationStatus: models.ValidationStatusDraft, CreatedAt: now, UpdatedAt: now}))

	// Create instances
	srcInstID, tgtInstID, otherInstID := newID(), newID(), newID()
	require.NoError(t, instRepo.Create(ctx, &models.EntityInstance{ID: srcInstID, EntityTypeID: et1ID, CatalogID: catID, Name: "dbi-src", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, instRepo.Create(ctx, &models.EntityInstance{ID: tgtInstID, EntityTypeID: et2ID, CatalogID: catID, Name: "dbi-tgt", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, instRepo.Create(ctx, &models.EntityInstance{ID: otherInstID, EntityTypeID: et2ID, CatalogID: catID, Name: "dbi-other", CreatedAt: now, UpdatedAt: now}))

	// Create association
	assocID := newID()
	require.NoError(t, assocRepo.Create(ctx, &models.Association{ID: assocID, EntityTypeVersionID: etvID, TargetEntityTypeID: et2ID, Type: models.AssociationTypeDirectional, CreatedAt: now}))

	// Create links: srcInstID→tgtInstID and otherInstID→srcInstID (srcInstID is target)
	link1ID := newID()
	require.NoError(t, linkRepo.Create(ctx, &models.AssociationLink{ID: link1ID, AssociationID: assocID, SourceInstanceID: srcInstID, TargetInstanceID: tgtInstID, CreatedAt: now}))
	link2ID := newID()
	require.NoError(t, linkRepo.Create(ctx, &models.AssociationLink{ID: link2ID, AssociationID: assocID, SourceInstanceID: otherInstID, TargetInstanceID: srcInstID, CreatedAt: now}))

	// DeleteByInstance should remove both links involving srcInstID
	err := linkRepo.DeleteByInstance(ctx, srcInstID)
	require.NoError(t, err)

	// Both links should be gone
	_, err = linkRepo.GetByID(ctx, link1ID)
	assert.True(t, domainerrors.IsNotFound(err))
	_, err = linkRepo.GetByID(ctx, link2ID)
	assert.True(t, domainerrors.IsNotFound(err))
}

func TestAssocLinkRepo_DeleteByInstance_NoLinks(t *testing.T) {
	db := testutil.NewTestDB(t)
	linkRepo := repository.NewAssociationLinkGormRepo(db)
	ctx := context.Background()

	// DeleteByInstance with an ID that has no links — should not error
	err := linkRepo.DeleteByInstance(ctx, "no-such-instance")
	require.NoError(t, err)
}

// ============================================================
// #3: AssociationGormRepo.Update (0% → covered)
// ============================================================

func TestAssocRepo_Update(t *testing.T) {
	db := testutil.NewTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	assocRepo := repository.NewAssociationGormRepo(db)
	ctx := context.Background()
	now := time.Now()

	et1ID, et2ID := newID(), newID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: et1ID, Name: "UpdSrc", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: et2ID, Name: "UpdTgt", CreatedAt: now, UpdatedAt: now}))
	etvID := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID, EntityTypeID: et1ID, Version: 1, CreatedAt: now}))

	assocID := newID()
	require.NoError(t, assocRepo.Create(ctx, &models.Association{
		ID: assocID, EntityTypeVersionID: etvID, Name: "orig-assoc",
		TargetEntityTypeID: et2ID, Type: models.AssociationTypeDirectional,
		SourceRole: "refers_to", TargetRole: "referred_by",
		CreatedAt: now,
	}))

	// Update the association
	found, err := assocRepo.GetByID(ctx, assocID)
	require.NoError(t, err)
	found.SourceRole = "updated_role"
	found.SourceCardinality = "1..n"
	require.NoError(t, assocRepo.Update(ctx, found))

	// Verify
	updated, err := assocRepo.GetByID(ctx, assocID)
	require.NoError(t, err)
	assert.Equal(t, "updated_role", updated.SourceRole)
	assert.Equal(t, "1..n", updated.SourceCardinality)
}

func TestAssocRepo_Update_DuplicateName(t *testing.T) {
	db := testutil.NewTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	assocRepo := repository.NewAssociationGormRepo(db)
	ctx := context.Background()
	now := time.Now()

	et1ID, et2ID := newID(), newID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: et1ID, Name: "DupSrc", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: et2ID, Name: "DupTgt", CreatedAt: now, UpdatedAt: now}))
	etvID := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID, EntityTypeID: et1ID, Version: 1, CreatedAt: now}))

	// Create two associations with different names
	assoc1ID := newID()
	require.NoError(t, assocRepo.Create(ctx, &models.Association{
		ID: assoc1ID, EntityTypeVersionID: etvID, Name: "assoc-a",
		TargetEntityTypeID: et2ID, Type: models.AssociationTypeDirectional, CreatedAt: now,
	}))
	assoc2ID := newID()
	require.NoError(t, assocRepo.Create(ctx, &models.Association{
		ID: assoc2ID, EntityTypeVersionID: etvID, Name: "assoc-b",
		TargetEntityTypeID: et2ID, Type: models.AssociationTypeDirectional, CreatedAt: now,
	}))

	// Try to update assoc-b's name to assoc-a — should conflict
	found, err := assocRepo.GetByID(ctx, assoc2ID)
	require.NoError(t, err)
	found.Name = "assoc-a"
	err = assocRepo.Update(ctx, found)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsConflict(err))
}

// ============================================================
// #4: AttributeGormRepo.ListByTypeDefinitionVersionIDs (0% → covered)
// ============================================================

func TestAttrRepo_ListByTypeDefinitionVersionIDs(t *testing.T) {
	db := testutil.NewTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	attrRepo := repository.NewAttributeGormRepo(db)
	tdRepo := repository.NewTypeDefinitionGormRepo(db)
	tdvRepo := repository.NewTypeDefinitionVersionGormRepo(db)
	ctx := context.Background()
	now := time.Now()

	// Create type definitions
	td1 := &models.TypeDefinition{ID: newID(), Name: "lbtdv-str", BaseType: models.BaseTypeString, CreatedAt: now, UpdatedAt: now}
	td2 := &models.TypeDefinition{ID: newID(), Name: "lbtdv-int", BaseType: models.BaseTypeInteger, CreatedAt: now, UpdatedAt: now}
	require.NoError(t, tdRepo.Create(ctx, td1))
	require.NoError(t, tdRepo.Create(ctx, td2))

	tdv1 := &models.TypeDefinitionVersion{ID: newID(), TypeDefinitionID: td1.ID, VersionNumber: 1, Constraints: map[string]any{}, CreatedAt: now}
	tdv2 := &models.TypeDefinitionVersion{ID: newID(), TypeDefinitionID: td2.ID, VersionNumber: 1, Constraints: map[string]any{}, CreatedAt: now}
	require.NoError(t, tdvRepo.Create(ctx, tdv1))
	require.NoError(t, tdvRepo.Create(ctx, tdv2))

	// Create entity type + version
	etID := newID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: etID, Name: "LbtdvET", CreatedAt: now, UpdatedAt: now}))
	etvID := newID()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID, EntityTypeID: etID, Version: 1, CreatedAt: now}))

	// Create attributes using those TDV IDs
	require.NoError(t, attrRepo.Create(ctx, &models.Attribute{ID: newID(), EntityTypeVersionID: etvID, Name: "attr-str", TypeDefinitionVersionID: tdv1.ID, Ordinal: 0}))
	require.NoError(t, attrRepo.Create(ctx, &models.Attribute{ID: newID(), EntityTypeVersionID: etvID, Name: "attr-int", TypeDefinitionVersionID: tdv2.ID, Ordinal: 1}))

	// Query by both TDV IDs
	attrs, err := attrRepo.ListByTypeDefinitionVersionIDs(ctx, []string{tdv1.ID, tdv2.ID})
	require.NoError(t, err)
	assert.Len(t, attrs, 2)

	// Query by single TDV ID
	attrs, err = attrRepo.ListByTypeDefinitionVersionIDs(ctx, []string{tdv1.ID})
	require.NoError(t, err)
	assert.Len(t, attrs, 1)
	assert.Equal(t, "attr-str", attrs[0].Name)

	// Empty slice → nil result
	attrs, err = attrRepo.ListByTypeDefinitionVersionIDs(ctx, []string{})
	require.NoError(t, err)
	assert.Nil(t, attrs)

	// No matches
	attrs, err = attrRepo.ListByTypeDefinitionVersionIDs(ctx, []string{"nonexistent-tdv"})
	require.NoError(t, err)
	assert.Len(t, attrs, 0)
}

// ============================================================
// #5: CatalogVersionPinGormRepo.Delete (0% for pin Delete → covered)
// Note: CatalogVersionGormRepo.Delete at line 118 is a DIFFERENT Delete
// from the CatalogVersionPinGormRepo.Delete. The pin Delete is at line 196.
// The CatalogVersion Delete (line 118) is also at 0%.
// ============================================================

func TestCatalogVersionRepo_Delete(t *testing.T) {
	db := testutil.NewTestDB(t)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	ctx := context.Background()
	now := time.Now()

	cvID := newID()
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{
		ID: cvID, VersionLabel: "cv-del-v1", LifecycleStage: models.LifecycleStageDevelopment,
		CreatedAt: now, UpdatedAt: now,
	}))

	// Delete — success
	require.NoError(t, cvRepo.Delete(ctx, cvID))

	// Verify it's gone
	_, err := cvRepo.GetByID(ctx, cvID)
	assert.True(t, domainerrors.IsNotFound(err))
}

func TestCatalogVersionRepo_Delete_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	ctx := context.Background()

	err := cvRepo.Delete(ctx, "nonexistent-cv-id")
	require.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
}

func TestCatalogVersionPinRepo_Delete_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	pinRepo := repository.NewCatalogVersionPinGormRepo(db)
	ctx := context.Background()

	err := pinRepo.Delete(ctx, "nonexistent-pin-id")
	require.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
}

// ============================================================
// #6: CatalogGormRepo.UpdatePublished — not-found branch (66.7% → higher)
// ============================================================

func TestCatalogRepo_UpdatePublished_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	catalogRepo := repository.NewCatalogGormRepo(db)
	ctx := context.Background()

	now := time.Now()
	err := catalogRepo.UpdatePublished(ctx, "nonexistent-catalog-id", true, &now)
	require.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
}

// ============================================================
// #7: EntityTypeGormRepo.Delete — not-found branch (83.3% → higher)
// ============================================================

func TestEntityTypeRepo_Delete_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	ctx := context.Background()

	err := etRepo.Delete(ctx, "nonexistent-et-id")
	require.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
}

// ============================================================
// #8: AttributeGormRepo.Delete — not-found branch (83.3% → higher)
// ============================================================

func TestAttrRepo_Delete_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	attrRepo := repository.NewAttributeGormRepo(db)
	ctx := context.Background()

	err := attrRepo.Delete(ctx, "nonexistent-attr-id")
	require.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
}

// ============================================================
// #9: AssociationGormRepo.Delete — not-found branch (83.3% → higher)
// ============================================================

func TestAssocRepo_Delete_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	assocRepo := repository.NewAssociationGormRepo(db)
	ctx := context.Background()

	err := assocRepo.Delete(ctx, "nonexistent-assoc-id")
	require.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
}

// ============================================================
// #10: CatalogVersionGormRepo.UpdateLifecycle — not-found branch (83.3% → higher)
// ============================================================

func TestCatalogVersionRepo_UpdateLifecycle_NotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	ctx := context.Background()

	err := cvRepo.UpdateLifecycle(ctx, "nonexistent-cv-id", models.LifecycleStageTesting)
	require.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
}

// ============================================================
// #11: EntityTypeGormRepo.Update — not-found / conflict branch (85.7% → higher)
// ============================================================

func TestEntityTypeRepo_Update_DuplicateName(t *testing.T) {
	db := testutil.NewTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	ctx := context.Background()
	now := time.Now()

	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: newID(), Name: "ET-Alpha", CreatedAt: now, UpdatedAt: now}))
	et2ID := newID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: et2ID, Name: "ET-Beta", CreatedAt: now, UpdatedAt: now}))

	// Try to rename ET-Beta to ET-Alpha (duplicate)
	found, err := etRepo.GetByID(ctx, et2ID)
	require.NoError(t, err)
	found.Name = "ET-Alpha"
	err = etRepo.Update(ctx, found)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsConflict(err))
}

// ============================================================
// #12: EntityTypeVersionGormRepo.Create — unique constraint branch (85.7% → higher)
// ============================================================

func TestEntityTypeVersionRepo_Create_DuplicateVersion(t *testing.T) {
	db := testutil.NewTestDB(t)
	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	ctx := context.Background()
	now := time.Now()

	etID := newID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: etID, Name: "DupVerET", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: newID(), EntityTypeID: etID, Version: 1, CreatedAt: now}))

	// Duplicate: same entity_type_id + version → unique constraint
	err := etvRepo.Create(ctx, &models.EntityTypeVersion{ID: newID(), EntityTypeID: etID, Version: 1, CreatedAt: now})
	assert.Error(t, err)
	assert.True(t, domainerrors.IsConflict(err))
}
