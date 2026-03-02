package meta_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository/mocks"
	"github.com/project-catalyst/pc-asset-hub/internal/service/meta"
)

func setupAssocService() (*meta.AssociationService, *mocks.MockAssociationRepo, *mocks.MockEntityTypeVersionRepo, *mocks.MockAttributeRepo) {
	assocRepo := new(mocks.MockAssociationRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	svc := meta.NewAssociationService(assocRepo, etvRepo, attrRepo)
	return svc, assocRepo, etvRepo, attrRepo
}

func TestT3_20_CreateContainmentAssociation(t *testing.T) {
	svc, assocRepo, etvRepo, attrRepo := setupAssocService()

	assocRepo.On("GetContainmentGraph", mock.Anything).Return([]repository.ContainmentEdge{}, nil)
	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-a", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Association")).Return(nil)

	newVer, err := svc.CreateAssociation(context.Background(), "et-a", "et-b", models.AssociationTypeContainment, "test_assoc", "contains", "part_of", "", "")
	require.NoError(t, err)
	assert.Equal(t, 2, newVer.Version)
}

func TestT3_21_CycleDetectionDirect(t *testing.T) {
	svc, assocRepo, etvRepo, _ := setupAssocService()

	// A contains B already exists, now trying B contains A
	assocRepo.On("GetContainmentGraph", mock.Anything).Return([]repository.ContainmentEdge{
		{SourceEntityTypeID: "et-a", TargetEntityTypeID: "et-b"},
	}, nil)
	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-b", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-b").Return(v1, nil)

	_, err := svc.CreateAssociation(context.Background(), "et-b", "et-a", models.AssociationTypeContainment, "test_assoc", "", "", "", "")
	assert.True(t, domainerrors.IsCycleDetected(err))
}

func TestT3_22_CycleDetectionIndirect(t *testing.T) {
	svc, assocRepo, etvRepo, _ := setupAssocService()

	// A->B->C exists, trying C->A
	assocRepo.On("GetContainmentGraph", mock.Anything).Return([]repository.ContainmentEdge{
		{SourceEntityTypeID: "et-a", TargetEntityTypeID: "et-b"},
		{SourceEntityTypeID: "et-b", TargetEntityTypeID: "et-c"},
	}, nil)
	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-c", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-c").Return(v1, nil)

	_, err := svc.CreateAssociation(context.Background(), "et-c", "et-a", models.AssociationTypeContainment, "test_assoc", "", "", "", "")
	assert.True(t, domainerrors.IsCycleDetected(err))
}

func TestT3_23_SelfContainment(t *testing.T) {
	svc, assocRepo, _, _ := setupAssocService()

	assocRepo.On("GetContainmentGraph", mock.Anything).Return([]repository.ContainmentEdge{}, nil)

	_, err := svc.CreateAssociation(context.Background(), "et-a", "et-a", models.AssociationTypeContainment, "test_assoc", "", "", "", "")
	assert.True(t, domainerrors.IsCycleDetected(err))
}

func TestT3_24_DirectionalReferenceNoCycleCheck(t *testing.T) {
	svc, assocRepo, etvRepo, attrRepo := setupAssocService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-a", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	newVer, err := svc.CreateAssociation(context.Background(), "et-a", "et-b", models.AssociationTypeDirectional, "test_assoc", "refers_to", "referred_by", "", "")
	require.NoError(t, err)
	assert.Equal(t, 2, newVer.Version)
	// GetContainmentGraph should NOT be called for directional references
	assocRepo.AssertNotCalled(t, "GetContainmentGraph", mock.Anything)
}

func TestT3_25_BidirectionalReferenceNoCycleCheck(t *testing.T) {
	svc, assocRepo, etvRepo, attrRepo := setupAssocService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-a", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	newVer, err := svc.CreateAssociation(context.Background(), "et-a", "et-b", models.AssociationTypeBidirectional, "test_assoc", "", "", "", "")
	require.NoError(t, err)
	assert.Equal(t, 2, newVer.Version)
	assocRepo.AssertNotCalled(t, "GetContainmentGraph", mock.Anything)
}

func TestT3_26_CreateAssociationIncrementsVersion(t *testing.T) {
	// Already verified in T3_20 — newVer.Version == 2
	t.Log("Covered by T3_20")
}

func TestT3_27_DeleteAssociation(t *testing.T) {
	svc, assocRepo, etvRepo, attrRepo := setupAssocService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-a", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(v1, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("ListByVersion", mock.Anything, mock.AnythingOfType("string")).Return([]*models.Association{
		{ID: "new-assoc-1", Name: "test_assoc", TargetEntityTypeID: "et-b", Type: models.AssociationTypeContainment},
	}, nil)
	assocRepo.On("Delete", mock.Anything, "new-assoc-1").Return(nil)

	newVer, err := svc.DeleteAssociation(context.Background(), "et-a", "test_assoc")
	require.NoError(t, err)
	assert.Equal(t, 2, newVer.Version)
}

func TestT3_28_CycleDetectionComplexDAG(t *testing.T) {
	svc, assocRepo, etvRepo, attrRepo := setupAssocService()

	// Complex DAG: A->B, A->C, B->D, C->D, B->E — all valid, no cycles
	assocRepo.On("GetContainmentGraph", mock.Anything).Return([]repository.ContainmentEdge{
		{SourceEntityTypeID: "a", TargetEntityTypeID: "b"},
		{SourceEntityTypeID: "a", TargetEntityTypeID: "c"},
		{SourceEntityTypeID: "b", TargetEntityTypeID: "d"},
		{SourceEntityTypeID: "c", TargetEntityTypeID: "d"},
		{SourceEntityTypeID: "b", TargetEntityTypeID: "e"},
	}, nil)

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "e", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "e").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	// E->F is valid (no cycle)
	newVer, err := svc.CreateAssociation(context.Background(), "e", "f", models.AssociationTypeContainment, "test_assoc", "", "", "", "")
	require.NoError(t, err)
	assert.NotNil(t, newVer)
}

// T-E.75: CreateAssociation with valid cardinality passes through to model
func TestTE75_CreateAssociationWithCardinality(t *testing.T) {
	svc, assocRepo, etvRepo, attrRepo := setupAssocService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-a", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Association")).Return(nil)

	newVer, err := svc.CreateAssociation(context.Background(), "et-a", "et-b", models.AssociationTypeDirectional, "test_assoc", "refers_to", "referred_by", "1", "0..n")
	require.NoError(t, err)
	assert.Equal(t, 2, newVer.Version)

	// Verify cardinality was passed to the created association
	call := assocRepo.Calls[len(assocRepo.Calls)-1]
	createdAssoc := call.Arguments.Get(1).(*models.Association)
	assert.Equal(t, "1", createdAssoc.SourceCardinality)
	assert.Equal(t, "0..n", createdAssoc.TargetCardinality)
}

// T-E.76: CreateAssociation with empty cardinality normalizes to "0..n"
func TestTE76_CreateAssociationEmptyCardinalityNormalized(t *testing.T) {
	svc, assocRepo, etvRepo, attrRepo := setupAssocService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-a", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Association")).Return(nil)

	newVer, err := svc.CreateAssociation(context.Background(), "et-a", "et-b", models.AssociationTypeDirectional, "test_assoc", "", "", "", "")
	require.NoError(t, err)
	assert.Equal(t, 2, newVer.Version)

	call := assocRepo.Calls[len(assocRepo.Calls)-1]
	createdAssoc := call.Arguments.Get(1).(*models.Association)
	assert.Equal(t, "0..n", createdAssoc.SourceCardinality)
	assert.Equal(t, "0..n", createdAssoc.TargetCardinality)
}

// T-E.77: CreateAssociation with invalid cardinality returns error
func TestTE77_CreateAssociationInvalidCardinality(t *testing.T) {
	svc, _, _, _ := setupAssocService()

	_, err := svc.CreateAssociation(context.Background(), "et-a", "et-b", models.AssociationTypeDirectional, "test_assoc", "", "", "bad", "")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
	assert.Contains(t, err.Error(), "source_cardinality")
}

// Containment source cardinality must be "1" or "0..1"
func TestCreateAssociation_ContainmentRejectsInvalidSourceCardinality(t *testing.T) {
	svc, assocRepo, _, _ := setupAssocService()

	assocRepo.On("GetContainmentGraph", mock.Anything).Return([]repository.ContainmentEdge{}, nil)

	// "0..n" is not valid for containment source (container side)
	_, err := svc.CreateAssociation(context.Background(), "et-a", "et-b", models.AssociationTypeContainment, "test_assoc", "", "", "0..n", "0..n")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
	assert.Contains(t, err.Error(), "source_cardinality")

	// "1..n" is not valid either
	_, err = svc.CreateAssociation(context.Background(), "et-a", "et-b", models.AssociationTypeContainment, "test_assoc2", "", "", "1..n", "0..n")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

func TestCreateAssociation_ContainmentAcceptsValidSourceCardinality(t *testing.T) {
	// "1" and "0..1" are valid for containment source
	for _, srcCard := range []string{"1", "0..1", ""} {
		t.Run(srcCard, func(t *testing.T) {
			svc, assocRepo, etvRepo, attrRepo := setupAssocService()

			assocRepo.On("GetContainmentGraph", mock.Anything).Return([]repository.ContainmentEdge{}, nil)
			v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-a", Version: 1}
			etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(v1, nil)
			attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
			assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
			etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
			attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			assocRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Association")).Return(nil)

			_, err := svc.CreateAssociation(context.Background(), "et-a", "et-b", models.AssociationTypeContainment, "test_assoc", "", "", srcCard, "0..n")
			assert.NoError(t, err)
		})
	}
}

// === EditAssociation Tests ===

// T-E.95: EditAssociation changes source role
func TestTE95_EditAssociationChangesSourceRole(t *testing.T) {
	svc, assocRepo, etvRepo, attrRepo := setupAssocService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-a", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(v1, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, "v1", mock.AnythingOfType("string")).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, "v1", mock.AnythingOfType("string")).Return(nil)
	// ListByVersion on the CURRENT version (v1) — used to find by name
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{
		{ID: "assoc-1", Name: "test_assoc", EntityTypeVersionID: "v1", TargetEntityTypeID: "et-b",
			Type: models.AssociationTypeDirectional, SourceRole: "old_role", TargetRole: "target",
			SourceCardinality: "0..n", TargetCardinality: "0..n"},
	}, nil)
	// ListByVersion on the NEW version — used to find the copy to update
	assocRepo.On("ListByVersion", mock.Anything, mock.MatchedBy(func(id string) bool { return id != "v1" })).Return([]*models.Association{
		{ID: "assoc-1-copy", Name: "test_assoc", EntityTypeVersionID: "new-v", TargetEntityTypeID: "et-b",
			Type: models.AssociationTypeDirectional, SourceRole: "old_role", TargetRole: "target",
			SourceCardinality: "0..n", TargetCardinality: "0..n"},
	}, nil)
	assocRepo.On("Update", mock.Anything, mock.MatchedBy(func(a *models.Association) bool {
		return a.SourceRole == "new_role"
	})).Return(nil)

	newVer, err := svc.EditAssociation(context.Background(), "et-a", "test_assoc", nil, strPtr("new_role"), nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 2, newVer.Version)
}

// T-E.96: EditAssociation changes cardinality
func TestTE96_EditAssociationChangesCardinality(t *testing.T) {
	svc, assocRepo, etvRepo, attrRepo := setupAssocService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-a", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(v1, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, "v1", mock.AnythingOfType("string")).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, "v1", mock.AnythingOfType("string")).Return(nil)
	// ListByVersion on the CURRENT version (v1) — used to find by name
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{
		{ID: "assoc-1", Name: "test_assoc", EntityTypeVersionID: "v1", TargetEntityTypeID: "et-b",
			Type: models.AssociationTypeDirectional, SourceCardinality: "0..n", TargetCardinality: "0..n"},
	}, nil)
	// ListByVersion on the NEW version
	assocRepo.On("ListByVersion", mock.Anything, mock.MatchedBy(func(id string) bool { return id != "v1" })).Return([]*models.Association{
		{ID: "assoc-1-copy", Name: "test_assoc", EntityTypeVersionID: "new-v", TargetEntityTypeID: "et-b",
			Type: models.AssociationTypeDirectional, SourceCardinality: "0..n", TargetCardinality: "0..n"},
	}, nil)
	assocRepo.On("Update", mock.Anything, mock.MatchedBy(func(a *models.Association) bool {
		return a.SourceCardinality == "1" && a.TargetCardinality == "1..n"
	})).Return(nil)

	newVer, err := svc.EditAssociation(context.Background(), "et-a", "test_assoc", nil, nil, nil, strPtr("1"), strPtr("1..n"))
	require.NoError(t, err)
	assert.Equal(t, 2, newVer.Version)
}

// T-E.97: EditAssociation with invalid cardinality
func TestTE97_EditAssociationInvalidCardinality(t *testing.T) {
	svc, assocRepo, etvRepo, _ := setupAssocService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-a", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(v1, nil)
	// ListByVersion on current version to find by name
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{
		{ID: "assoc-1", Name: "test_assoc", Type: models.AssociationTypeDirectional},
	}, nil)

	_, err := svc.EditAssociation(context.Background(), "et-a", "test_assoc", nil, nil, nil, strPtr("bad"), nil)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

// T-E.98: EditAssociation containment rejects invalid source cardinality
func TestTE98_EditAssociationContainmentRejectsInvalidSource(t *testing.T) {
	svc, assocRepo, etvRepo, _ := setupAssocService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-a", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(v1, nil)
	// ListByVersion on current version to find by name
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{
		{ID: "assoc-1", Name: "test_assoc", Type: models.AssociationTypeContainment, SourceCardinality: "0..1"},
	}, nil)

	_, err := svc.EditAssociation(context.Background(), "et-a", "test_assoc", nil, nil, nil, strPtr("0..n"), nil)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
	assert.Contains(t, err.Error(), "source_cardinality")
}

// T-E.99: EditAssociation on nonexistent association
func TestTE99_EditAssociationNotFound(t *testing.T) {
	svc, assocRepo, etvRepo, _ := setupAssocService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-a", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(v1, nil)
	// ListByVersion returns no matching name
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)

	_, err := svc.EditAssociation(context.Background(), "et-a", "bad_name", nil, nil, nil, nil, nil)
	assert.True(t, domainerrors.IsNotFound(err))
}

// EditAssociation matches the correct association by name
func TestEditAssociation_MatchesCorrectDuplicate(t *testing.T) {
	svc, assocRepo, etvRepo, attrRepo := setupAssocService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-a", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(v1, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, "v1", mock.AnythingOfType("string")).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, "v1", mock.AnythingOfType("string")).Return(nil)

	// ListByVersion on the CURRENT version (v1) — finds "primary_ref" by name
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{
		{ID: "assoc-2", Name: "secondary_ref", EntityTypeVersionID: "v1", TargetEntityTypeID: "et-b",
			Type: models.AssociationTypeDirectional, SourceRole: "secondary", TargetRole: "target2",
			SourceCardinality: "0..n", TargetCardinality: "0..n"},
		{ID: "assoc-1", Name: "primary_ref", EntityTypeVersionID: "v1", TargetEntityTypeID: "et-b",
			Type: models.AssociationTypeDirectional, SourceRole: "primary", TargetRole: "target1",
			SourceCardinality: "0..n", TargetCardinality: "0..n"},
	}, nil)

	// New version has TWO associations to same target with same type but different names
	// Put the WRONG one first to expose the bug if matching only by target+type
	assocRepo.On("ListByVersion", mock.Anything, mock.MatchedBy(func(id string) bool { return id != "v1" })).Return([]*models.Association{
		{ID: "copy-2", Name: "secondary_ref", TargetEntityTypeID: "et-b", Type: models.AssociationTypeDirectional,
			SourceRole: "secondary", TargetRole: "target2", SourceCardinality: "0..n", TargetCardinality: "0..n"},
		{ID: "copy-1", Name: "primary_ref", TargetEntityTypeID: "et-b", Type: models.AssociationTypeDirectional,
			SourceRole: "primary", TargetRole: "target1", SourceCardinality: "0..n", TargetCardinality: "0..n"},
	}, nil)

	// Should update "copy-1" (the one with matching name "primary_ref"), NOT "copy-2"
	assocRepo.On("Update", mock.Anything, mock.MatchedBy(func(a *models.Association) bool {
		return a.ID == "copy-1" && a.SourceRole == "updated_primary"
	})).Return(nil)

	newVer, err := svc.EditAssociation(context.Background(), "et-a", "primary_ref", nil, strPtr("updated_primary"), nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 2, newVer.Version)
	// Verify that Update was called with copy-1 (not copy-2)
	assocRepo.AssertCalled(t, "Update", mock.Anything, mock.MatchedBy(func(a *models.Association) bool {
		return a.ID == "copy-1"
	}))
}

// Directional/bidirectional have no source cardinality restriction
func TestCreateAssociation_NonContainmentAllowsAnySourceCardinality(t *testing.T) {
	svc, assocRepo, etvRepo, attrRepo := setupAssocService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-a", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	// "0..n" is fine for directional
	_, err := svc.CreateAssociation(context.Background(), "et-a", "et-b", models.AssociationTypeDirectional, "test_assoc", "", "", "0..n", "0..n")
	assert.NoError(t, err)
}

// === Association Name Tests ===

// T-E.106: CreateAssociation requires name
func TestTE106_CreateAssociationRequiresName(t *testing.T) {
	svc, _, _, _ := setupAssocService()

	_, err := svc.CreateAssociation(context.Background(), "et-a", "et-b", models.AssociationTypeDirectional, "", "", "", "", "")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
	assert.Contains(t, err.Error(), "name")
}

// T-E.107: CreateAssociation rejects duplicate association name
func TestTE107_CreateAssociationDuplicateName(t *testing.T) {
	svc, assocRepo, etvRepo, attrRepo := setupAssocService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-a", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{
		{ID: "existing", Name: "tools", TargetEntityTypeID: "et-b", Type: models.AssociationTypeContainment},
	}, nil)

	_, err := svc.CreateAssociation(context.Background(), "et-a", "et-c", models.AssociationTypeDirectional, "tools", "", "", "", "")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsConflict(err))
}

// T-E.108: CreateAssociation rejects name that conflicts with attribute
func TestTE108_CreateAssociationNameConflictsWithAttribute(t *testing.T) {
	svc, assocRepo, etvRepo, attrRepo := setupAssocService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-a", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(v1, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{
		{ID: "a1", Name: "hostname", Type: models.AttributeTypeString},
	}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)

	_, err := svc.CreateAssociation(context.Background(), "et-a", "et-b", models.AssociationTypeDirectional, "hostname", "", "", "", "")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsConflict(err))
	assert.Contains(t, err.Error(), "attribute")
}

// T-E.110: EditAssociation can rename
func TestTE110_EditAssociationRename(t *testing.T) {
	svc, assocRepo, etvRepo, attrRepo := setupAssocService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-a", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(v1, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, "v1", mock.AnythingOfType("string")).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, "v1", mock.AnythingOfType("string")).Return(nil)
	// ListByVersion on current version (for finding and validation)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{
		{ID: "assoc-1", Name: "old_name", EntityTypeVersionID: "v1", TargetEntityTypeID: "et-b",
			Type: models.AssociationTypeDirectional, SourceCardinality: "0..n", TargetCardinality: "0..n"},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	// ListByVersion on new version (for finding the copy)
	assocRepo.On("ListByVersion", mock.Anything, mock.MatchedBy(func(id string) bool { return id != "v1" })).Return([]*models.Association{
		{ID: "assoc-1-copy", Name: "old_name", EntityTypeVersionID: "new-v", TargetEntityTypeID: "et-b",
			Type: models.AssociationTypeDirectional, SourceCardinality: "0..n", TargetCardinality: "0..n"},
	}, nil)
	assocRepo.On("Update", mock.Anything, mock.MatchedBy(func(a *models.Association) bool {
		return a.Name == "new_name"
	})).Return(nil)

	newVer, err := svc.EditAssociation(context.Background(), "et-a", "old_name", strPtr("new_name"), nil, nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 2, newVer.Version)
}

// T-E.111: EditAssociation rename conflicts with attribute
func TestTE111_EditAssociationRenameConflictsWithAttribute(t *testing.T) {
	svc, assocRepo, etvRepo, attrRepo := setupAssocService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-a", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(v1, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{
		{ID: "assoc-1", Name: "tools", EntityTypeVersionID: "v1", TargetEntityTypeID: "et-b", Type: models.AssociationTypeDirectional},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{
		{ID: "a1", Name: "hostname", Type: models.AttributeTypeString},
	}, nil)

	_, err := svc.EditAssociation(context.Background(), "et-a", "tools", strPtr("hostname"), nil, nil, nil, nil)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsConflict(err))
}

// T-E.112: DeleteAssociation by name
func TestTE112_DeleteAssociationByName(t *testing.T) {
	svc, assocRepo, etvRepo, attrRepo := setupAssocService()

	v1 := &models.EntityTypeVersion{ID: "v1", EntityTypeID: "et-a", Version: 1}
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(v1, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("ListByVersion", mock.Anything, mock.AnythingOfType("string")).Return([]*models.Association{
		{ID: "new-assoc-1", Name: "tools", TargetEntityTypeID: "et-b", Type: models.AssociationTypeContainment},
	}, nil)
	assocRepo.On("Delete", mock.Anything, "new-assoc-1").Return(nil)

	newVer, err := svc.DeleteAssociation(context.Background(), "et-a", "tools")
	require.NoError(t, err)
	assert.Equal(t, 2, newVer.Version)
}
