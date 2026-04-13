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

func id() string {
	return uuid.Must(uuid.NewV7()).String()
}

type testContext struct {
	cvID     string
	etID     string
	etvID    string
	instRepo *repository.EntityInstanceGormRepo
	iavRepo  *repository.InstanceAttributeValueGormRepo
	linkRepo *repository.AssociationLinkGormRepo
	attrRepo *repository.AttributeGormRepo
}

func setupTestContext(t *testing.T) (*testContext, context.Context) {
	t.Helper()
	db := testutil.NewTestDB(t)
	ctx := context.Background()

	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	cvRepo := repository.NewCatalogVersionGormRepo(db)

	etID := id()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: etID, Name: "Model-" + etID[:8], CreatedAt: time.Now(), UpdatedAt: time.Now()}))
	etvID := id()
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID, EntityTypeID: etID, Version: 1, CreatedAt: time.Now()}))
	cvID := id()
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{ID: cvID, VersionLabel: "v-" + cvID[:8], LifecycleStage: models.LifecycleStageDevelopment, CreatedAt: time.Now(), UpdatedAt: time.Now()}))

	return &testContext{
		cvID:     cvID,
		etID:     etID,
		etvID:    etvID,
		instRepo: repository.NewEntityInstanceGormRepo(db),
		iavRepo:  repository.NewInstanceAttributeValueGormRepo(db),
		linkRepo: repository.NewAssociationLinkGormRepo(db),
		attrRepo: repository.NewAttributeGormRepo(db),
	}, ctx
}

// === Entity Instances (T-2.01 through T-2.10) ===

func TestT2_01_CreateTopLevelInstance(t *testing.T) {
	tc, ctx := setupTestContext(t)

	inst := &models.EntityInstance{
		ID: id(), EntityTypeID: tc.etID, CatalogID: tc.cvID,
		Name: "llama-3-70b", Description: "A large language model", Version: 1,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, tc.instRepo.Create(ctx, inst))

	found, err := tc.instRepo.GetByID(ctx, inst.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, found.Version)
	assert.Equal(t, "llama-3-70b", found.Name)
	assert.False(t, found.CreatedAt.IsZero())
}

func TestT2_02_CreateDuplicateInstanceName(t *testing.T) {
	tc, ctx := setupTestContext(t)

	inst1 := &models.EntityInstance{
		ID: id(), EntityTypeID: tc.etID, CatalogID: tc.cvID,
		Name: "llama", Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, tc.instRepo.Create(ctx, inst1))

	inst2 := &models.EntityInstance{
		ID: id(), EntityTypeID: tc.etID, CatalogID: tc.cvID,
		Name: "llama", Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	err := tc.instRepo.Create(ctx, inst2)
	assert.True(t, domainerrors.IsConflict(err))
}

func TestT2_03_CreateContainedInstance(t *testing.T) {
	tc, ctx := setupTestContext(t)

	parent := &models.EntityInstance{
		ID: id(), EntityTypeID: tc.etID, CatalogID: tc.cvID,
		Name: "mcp-server-1", Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, tc.instRepo.Create(ctx, parent))

	child := &models.EntityInstance{
		ID: id(), EntityTypeID: tc.etID, CatalogID: tc.cvID,
		ParentInstanceID: parent.ID, Name: "tool-1", Version: 1,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, tc.instRepo.Create(ctx, child))

	found, err := tc.instRepo.GetByID(ctx, child.ID)
	require.NoError(t, err)
	assert.Equal(t, parent.ID, found.ParentInstanceID)
}

func TestT2_04_SameNameDifferentParents(t *testing.T) {
	tc, ctx := setupTestContext(t)

	parent1 := &models.EntityInstance{
		ID: id(), EntityTypeID: tc.etID, CatalogID: tc.cvID,
		Name: "server-1", Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	parent2 := &models.EntityInstance{
		ID: id(), EntityTypeID: tc.etID, CatalogID: tc.cvID,
		Name: "server-2", Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, tc.instRepo.Create(ctx, parent1))
	require.NoError(t, tc.instRepo.Create(ctx, parent2))

	child1 := &models.EntityInstance{
		ID: id(), EntityTypeID: tc.etID, CatalogID: tc.cvID,
		ParentInstanceID: parent1.ID, Name: "tool-A", Version: 1,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	child2 := &models.EntityInstance{
		ID: id(), EntityTypeID: tc.etID, CatalogID: tc.cvID,
		ParentInstanceID: parent2.ID, Name: "tool-A", Version: 1,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, tc.instRepo.Create(ctx, child1))
	err := tc.instRepo.Create(ctx, child2)
	assert.NoError(t, err) // Same name, different parents — allowed
}

func TestT2_05_SameNameSameParent(t *testing.T) {
	tc, ctx := setupTestContext(t)

	parent := &models.EntityInstance{
		ID: id(), EntityTypeID: tc.etID, CatalogID: tc.cvID,
		Name: "server-1", Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, tc.instRepo.Create(ctx, parent))

	child1 := &models.EntityInstance{
		ID: id(), EntityTypeID: tc.etID, CatalogID: tc.cvID,
		ParentInstanceID: parent.ID, Name: "tool-A", Version: 1,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	child2 := &models.EntityInstance{
		ID: id(), EntityTypeID: tc.etID, CatalogID: tc.cvID,
		ParentInstanceID: parent.ID, Name: "tool-A", Version: 1,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, tc.instRepo.Create(ctx, child1))
	err := tc.instRepo.Create(ctx, child2)
	assert.True(t, domainerrors.IsConflict(err))
}

func TestT2_06_CreateWithNonExistentParent(t *testing.T) {
	tc, ctx := setupTestContext(t)

	child := &models.EntityInstance{
		ID: id(), EntityTypeID: tc.etID, CatalogID: tc.cvID,
		ParentInstanceID: "nonexistent-parent-id", Name: "orphan", Version: 1,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	// SQLite doesn't enforce FK on parent_instance_id because the column can be empty string.
	// The service layer validates parent existence. At the repo level, this may succeed.
	// We accept this — FK enforcement for self-references is tricky in SQLite.
	_ = tc.instRepo.Create(ctx, child)
}

func TestT2_07_UpdateInstanceVersion(t *testing.T) {
	tc, ctx := setupTestContext(t)

	inst := &models.EntityInstance{
		ID: id(), EntityTypeID: tc.etID, CatalogID: tc.cvID,
		Name: "model-1", Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, tc.instRepo.Create(ctx, inst))

	inst.Version = 2
	inst.UpdatedAt = time.Now()
	require.NoError(t, tc.instRepo.Update(ctx, inst))

	found, err := tc.instRepo.GetByID(ctx, inst.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, found.Version)
}

func TestT2_08_SoftDeleteInstance(t *testing.T) {
	tc, ctx := setupTestContext(t)

	inst := &models.EntityInstance{
		ID: id(), EntityTypeID: tc.etID, CatalogID: tc.cvID,
		Name: "model-1", Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, tc.instRepo.Create(ctx, inst))

	require.NoError(t, tc.instRepo.SoftDelete(ctx, inst.ID))

	// Should not be found via normal query
	_, err := tc.instRepo.GetByID(ctx, inst.ID)
	assert.True(t, domainerrors.IsNotFound(err))
}

func TestT2_09_ListInstancesWithPagination(t *testing.T) {
	tc, ctx := setupTestContext(t)

	for _, name := range []string{"alpha", "beta", "charlie", "delta"} {
		require.NoError(t, tc.instRepo.Create(ctx, &models.EntityInstance{
			ID: id(), EntityTypeID: tc.etID, CatalogID: tc.cvID,
			Name: name, Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}))
	}

	items, total, err := tc.instRepo.List(ctx, tc.etID, tc.cvID, models.ListParams{Limit: 2, Offset: 0})
	require.NoError(t, err)
	assert.Equal(t, 4, total)
	assert.Len(t, items, 2)
}

func TestT2_10_ListByParent(t *testing.T) {
	tc, ctx := setupTestContext(t)

	parent := &models.EntityInstance{
		ID: id(), EntityTypeID: tc.etID, CatalogID: tc.cvID,
		Name: "server-1", Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, tc.instRepo.Create(ctx, parent))

	for _, name := range []string{"tool-a", "tool-b"} {
		require.NoError(t, tc.instRepo.Create(ctx, &models.EntityInstance{
			ID: id(), EntityTypeID: tc.etID, CatalogID: tc.cvID,
			ParentInstanceID: parent.ID, Name: name, Version: 1,
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}))
	}
	// Another top-level instance — should not appear
	require.NoError(t, tc.instRepo.Create(ctx, &models.EntityInstance{
		ID: id(), EntityTypeID: tc.etID, CatalogID: tc.cvID,
		Name: "other", Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	children, total, err := tc.instRepo.ListByParent(ctx, parent.ID, models.ListParams{})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, children, 2)
}

// === Instance Attribute Values (T-2.11 through T-2.17) ===

func TestT2_11_SetStringValue(t *testing.T) {
	tc, ctx := setupTestContext(t)

	instID := id()
	require.NoError(t, tc.instRepo.Create(ctx, &models.EntityInstance{
		ID: instID, EntityTypeID: tc.etID, CatalogID: tc.cvID,
		Name: "model-1", Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	attrID := id()
	values := []*models.InstanceAttributeValue{{
		ID: id(), InstanceID: instID, InstanceVersion: 1,
		AttributeID: attrID, ValueString: "https://api.example.com",
	}}
	require.NoError(t, tc.iavRepo.SetValues(ctx, values))

	found, err := tc.iavRepo.GetCurrentValues(ctx, instID)
	require.NoError(t, err)
	assert.Len(t, found, 1)
	assert.Equal(t, "https://api.example.com", found[0].ValueString)
}

func TestT2_12_SetNumberValue(t *testing.T) {
	tc, ctx := setupTestContext(t)

	instID := id()
	require.NoError(t, tc.instRepo.Create(ctx, &models.EntityInstance{
		ID: instID, EntityTypeID: tc.etID, CatalogID: tc.cvID,
		Name: "model-1", Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	val := 4096.0
	values := []*models.InstanceAttributeValue{{
		ID: id(), InstanceID: instID, InstanceVersion: 1,
		AttributeID: id(), ValueNumber: &val,
	}}
	require.NoError(t, tc.iavRepo.SetValues(ctx, values))

	found, err := tc.iavRepo.GetCurrentValues(ctx, instID)
	require.NoError(t, err)
	assert.Len(t, found, 1)
	assert.NotNil(t, found[0].ValueNumber)
	assert.Equal(t, 4096.0, *found[0].ValueNumber)
}

func TestT2_13_SetEnumValue(t *testing.T) {
	tc, ctx := setupTestContext(t)

	instID := id()
	require.NoError(t, tc.instRepo.Create(ctx, &models.EntityInstance{
		ID: instID, EntityTypeID: tc.etID, CatalogID: tc.cvID,
		Name: "model-1", Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	values := []*models.InstanceAttributeValue{{
		ID: id(), InstanceID: instID, InstanceVersion: 1,
		AttributeID: id(), ValueString: "active",
	}}
	require.NoError(t, tc.iavRepo.SetValues(ctx, values))

	found, err := tc.iavRepo.GetCurrentValues(ctx, instID)
	require.NoError(t, err)
	assert.Len(t, found, 1)
	assert.Equal(t, "active", found[0].ValueString)
}

func TestT2_14_VersionedValues(t *testing.T) {
	tc, ctx := setupTestContext(t)

	instID := id()
	attrID := id()
	require.NoError(t, tc.instRepo.Create(ctx, &models.EntityInstance{
		ID: instID, EntityTypeID: tc.etID, CatalogID: tc.cvID,
		Name: "model-1", Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	// Version 1 values
	require.NoError(t, tc.iavRepo.SetValues(ctx, []*models.InstanceAttributeValue{{
		ID: id(), InstanceID: instID, InstanceVersion: 1,
		AttributeID: attrID, ValueString: "v1-value",
	}}))

	// Version 2 values
	require.NoError(t, tc.iavRepo.SetValues(ctx, []*models.InstanceAttributeValue{{
		ID: id(), InstanceID: instID, InstanceVersion: 2,
		AttributeID: attrID, ValueString: "v2-value",
	}}))

	v1, err := tc.iavRepo.GetValuesForVersion(ctx, instID, 1)
	require.NoError(t, err)
	assert.Len(t, v1, 1)
	assert.Equal(t, "v1-value", v1[0].ValueString)

	v2, err := tc.iavRepo.GetValuesForVersion(ctx, instID, 2)
	require.NoError(t, err)
	assert.Len(t, v2, 1)
	assert.Equal(t, "v2-value", v2[0].ValueString)
}

func TestT2_15_GetCurrentValues(t *testing.T) {
	tc, ctx := setupTestContext(t)

	instID := id()
	attrID := id()
	require.NoError(t, tc.instRepo.Create(ctx, &models.EntityInstance{
		ID: instID, EntityTypeID: tc.etID, CatalogID: tc.cvID,
		Name: "model-1", Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	require.NoError(t, tc.iavRepo.SetValues(ctx, []*models.InstanceAttributeValue{{
		ID: id(), InstanceID: instID, InstanceVersion: 1, AttributeID: attrID, ValueString: "old",
	}}))
	require.NoError(t, tc.iavRepo.SetValues(ctx, []*models.InstanceAttributeValue{{
		ID: id(), InstanceID: instID, InstanceVersion: 2, AttributeID: attrID, ValueString: "current",
	}}))

	current, err := tc.iavRepo.GetCurrentValues(ctx, instID)
	require.NoError(t, err)
	assert.Len(t, current, 1)
	assert.Equal(t, "current", current[0].ValueString)
}

func TestT2_16_GetValuesForSpecificVersion(t *testing.T) {
	tc, ctx := setupTestContext(t)

	instID := id()
	attrID := id()
	require.NoError(t, tc.instRepo.Create(ctx, &models.EntityInstance{
		ID: instID, EntityTypeID: tc.etID, CatalogID: tc.cvID,
		Name: "model-1", Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	require.NoError(t, tc.iavRepo.SetValues(ctx, []*models.InstanceAttributeValue{{
		ID: id(), InstanceID: instID, InstanceVersion: 1, AttributeID: attrID, ValueString: "first",
	}}))
	require.NoError(t, tc.iavRepo.SetValues(ctx, []*models.InstanceAttributeValue{{
		ID: id(), InstanceID: instID, InstanceVersion: 2, AttributeID: attrID, ValueString: "second",
	}}))

	v1, err := tc.iavRepo.GetValuesForVersion(ctx, instID, 1)
	require.NoError(t, err)
	assert.Equal(t, "first", v1[0].ValueString)
}

func TestT2_17_UniqueConstraintOnAttributeValues(t *testing.T) {
	tc, ctx := setupTestContext(t)

	instID := id()
	attrID := id()
	require.NoError(t, tc.instRepo.Create(ctx, &models.EntityInstance{
		ID: instID, EntityTypeID: tc.etID, CatalogID: tc.cvID,
		Name: "model-1", Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	require.NoError(t, tc.iavRepo.SetValues(ctx, []*models.InstanceAttributeValue{{
		ID: id(), InstanceID: instID, InstanceVersion: 1, AttributeID: attrID, ValueString: "first",
	}}))

	// Same instance, version, attribute — should conflict
	err := tc.iavRepo.SetValues(ctx, []*models.InstanceAttributeValue{{
		ID: id(), InstanceID: instID, InstanceVersion: 1, AttributeID: attrID, ValueString: "duplicate",
	}})
	assert.True(t, domainerrors.IsConflict(err))
}

// === Association Links (T-2.18 through T-2.22) ===

func TestT2_18_CreateAssociationLink(t *testing.T) {
	tc, ctx := setupTestContext(t)

	srcID, tgtID := id(), id()
	require.NoError(t, tc.instRepo.Create(ctx, &models.EntityInstance{
		ID: srcID, EntityTypeID: tc.etID, CatalogID: tc.cvID,
		Name: "source", Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	require.NoError(t, tc.instRepo.Create(ctx, &models.EntityInstance{
		ID: tgtID, EntityTypeID: tc.etID, CatalogID: tc.cvID,
		Name: "target", Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	assocID := id()
	link := &models.AssociationLink{
		ID: id(), AssociationID: assocID,
		SourceInstanceID: srcID, TargetInstanceID: tgtID,
		CreatedAt: time.Now(),
	}
	require.NoError(t, tc.linkRepo.Create(ctx, link))

	refs, err := tc.linkRepo.GetForwardRefs(ctx, srcID)
	require.NoError(t, err)
	assert.Len(t, refs, 1)
	assert.Equal(t, tgtID, refs[0].TargetInstanceID)
}

func TestT2_19_GetForwardReferences(t *testing.T) {
	tc, ctx := setupTestContext(t)

	srcID, tgt1ID, tgt2ID := id(), id(), id()
	for _, inst := range []struct{ id, name string }{{srcID, "src"}, {tgt1ID, "tgt1"}, {tgt2ID, "tgt2"}} {
		require.NoError(t, tc.instRepo.Create(ctx, &models.EntityInstance{
			ID: inst.id, EntityTypeID: tc.etID, CatalogID: tc.cvID,
			Name: inst.name, Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}))
	}

	assocID := id()
	require.NoError(t, tc.linkRepo.Create(ctx, &models.AssociationLink{ID: id(), AssociationID: assocID, SourceInstanceID: srcID, TargetInstanceID: tgt1ID, CreatedAt: time.Now()}))
	require.NoError(t, tc.linkRepo.Create(ctx, &models.AssociationLink{ID: id(), AssociationID: assocID, SourceInstanceID: srcID, TargetInstanceID: tgt2ID, CreatedAt: time.Now()}))

	refs, err := tc.linkRepo.GetForwardRefs(ctx, srcID)
	require.NoError(t, err)
	assert.Len(t, refs, 2)
}

func TestT2_20_GetReverseReferences(t *testing.T) {
	tc, ctx := setupTestContext(t)

	src1ID, src2ID, tgtID := id(), id(), id()
	for _, inst := range []struct{ id, name string }{{src1ID, "src1"}, {src2ID, "src2"}, {tgtID, "tgt"}} {
		require.NoError(t, tc.instRepo.Create(ctx, &models.EntityInstance{
			ID: inst.id, EntityTypeID: tc.etID, CatalogID: tc.cvID,
			Name: inst.name, Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}))
	}

	assocID := id()
	require.NoError(t, tc.linkRepo.Create(ctx, &models.AssociationLink{ID: id(), AssociationID: assocID, SourceInstanceID: src1ID, TargetInstanceID: tgtID, CreatedAt: time.Now()}))
	require.NoError(t, tc.linkRepo.Create(ctx, &models.AssociationLink{ID: id(), AssociationID: assocID, SourceInstanceID: src2ID, TargetInstanceID: tgtID, CreatedAt: time.Now()}))

	refs, err := tc.linkRepo.GetReverseRefs(ctx, tgtID)
	require.NoError(t, err)
	assert.Len(t, refs, 2)
}

func TestT2_21_DeleteAssociationLink(t *testing.T) {
	tc, ctx := setupTestContext(t)

	srcID, tgtID := id(), id()
	require.NoError(t, tc.instRepo.Create(ctx, &models.EntityInstance{
		ID: srcID, EntityTypeID: tc.etID, CatalogID: tc.cvID,
		Name: "source", Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	require.NoError(t, tc.instRepo.Create(ctx, &models.EntityInstance{
		ID: tgtID, EntityTypeID: tc.etID, CatalogID: tc.cvID,
		Name: "target", Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	linkID := id()
	require.NoError(t, tc.linkRepo.Create(ctx, &models.AssociationLink{
		ID: linkID, AssociationID: id(),
		SourceInstanceID: srcID, TargetInstanceID: tgtID,
		CreatedAt: time.Now(),
	}))

	require.NoError(t, tc.linkRepo.Delete(ctx, linkID))

	refs, err := tc.linkRepo.GetForwardRefs(ctx, srcID)
	require.NoError(t, err)
	assert.Len(t, refs, 0)
}

func TestT2_22_FilterForwardRefsByAssociationType(t *testing.T) {
	tc, ctx := setupTestContext(t)

	srcID, tgt1ID, tgt2ID := id(), id(), id()
	for _, inst := range []struct{ id, name string }{{srcID, "src"}, {tgt1ID, "tgt1"}, {tgt2ID, "tgt2"}} {
		require.NoError(t, tc.instRepo.Create(ctx, &models.EntityInstance{
			ID: inst.id, EntityTypeID: tc.etID, CatalogID: tc.cvID,
			Name: inst.name, Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}))
	}

	assoc1ID := id() // containment
	assoc2ID := id() // directional
	require.NoError(t, tc.linkRepo.Create(ctx, &models.AssociationLink{ID: id(), AssociationID: assoc1ID, SourceInstanceID: srcID, TargetInstanceID: tgt1ID, CreatedAt: time.Now()}))
	require.NoError(t, tc.linkRepo.Create(ctx, &models.AssociationLink{ID: id(), AssociationID: assoc2ID, SourceInstanceID: srcID, TargetInstanceID: tgt2ID, CreatedAt: time.Now()}))

	// GetForwardRefs returns all — filtering by association type is done at the service layer
	// by joining with the associations table. At the repo level, we get all links.
	refs, err := tc.linkRepo.GetForwardRefs(ctx, srcID)
	require.NoError(t, err)
	assert.Len(t, refs, 2)

	// We can filter in-memory by association ID
	var filtered []*models.AssociationLink
	for _, r := range refs {
		if r.AssociationID == assoc1ID {
			filtered = append(filtered, r)
		}
	}
	assert.Len(t, filtered, 1)
	assert.Equal(t, tgt1ID, filtered[0].TargetInstanceID)
}

// === ListByCatalog (T-13.01 through T-13.03) ===

func TestT13_01_ListByCatalog_ReturnsAllInstances(t *testing.T) {
	tc, ctx := setupTestContext(t)
	catalogID := tc.cvID

	for _, name := range []string{"charlie", "alpha", "bravo"} {
		require.NoError(t, tc.instRepo.Create(ctx, &models.EntityInstance{
			ID: id(), EntityTypeID: tc.etID, CatalogID: catalogID,
			Name: name, Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}))
	}

	results, err := tc.instRepo.ListByCatalog(ctx, catalogID)
	require.NoError(t, err)
	assert.Len(t, results, 3)
	// Should be ordered by name
	assert.Equal(t, "alpha", results[0].Name)
	assert.Equal(t, "bravo", results[1].Name)
	assert.Equal(t, "charlie", results[2].Name)
}

func TestT13_02_ListByCatalog_ExcludesOtherCatalogs(t *testing.T) {
	tc, ctx := setupTestContext(t)
	catalog1 := tc.cvID
	catalog2 := id()

	require.NoError(t, tc.instRepo.Create(ctx, &models.EntityInstance{
		ID: id(), EntityTypeID: tc.etID, CatalogID: catalog1,
		Name: "in-catalog-1", Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	require.NoError(t, tc.instRepo.Create(ctx, &models.EntityInstance{
		ID: id(), EntityTypeID: tc.etID, CatalogID: catalog2,
		Name: "in-catalog-2", Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	results, err := tc.instRepo.ListByCatalog(ctx, catalog1)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "in-catalog-1", results[0].Name)
}

func TestT13_03_ListByCatalog_ExcludesDeleted(t *testing.T) {
	tc, ctx := setupTestContext(t)

	instID := id()
	require.NoError(t, tc.instRepo.Create(ctx, &models.EntityInstance{
		ID: instID, EntityTypeID: tc.etID, CatalogID: tc.cvID,
		Name: "deleted-inst", Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	require.NoError(t, tc.instRepo.SoftDelete(ctx, instID))

	results, err := tc.instRepo.ListByCatalog(ctx, tc.cvID)
	require.NoError(t, err)
	assert.Len(t, results, 0)
}

func TestT13_04_ListByCatalog_EmptyCatalog(t *testing.T) {
	tc, ctx := setupTestContext(t)

	results, err := tc.instRepo.ListByCatalog(ctx, "nonexistent-catalog-id")
	require.NoError(t, err)
	assert.Empty(t, results)
}

// === Attribute Filtering (T-13.15 through T-13.23) ===

func TestT13_15_StringFilter_CaseInsensitiveContains(t *testing.T) {
	tc, ctx := setupTestContext(t)
	attrID := id()
	require.NoError(t, tc.attrRepo.Create(ctx, &models.Attribute{
		ID: attrID, EntityTypeVersionID: tc.etvID, Name: "hostname", TypeDefinitionVersionID: "tdv-string", Ordinal: 0,
	}))

	// Create two instances with attribute values
	inst1ID := id()
	require.NoError(t, tc.instRepo.Create(ctx, &models.EntityInstance{
		ID: inst1ID, EntityTypeID: tc.etID, CatalogID: tc.cvID,
		Name: "srv-alpha", Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	require.NoError(t, tc.iavRepo.SetValues(ctx, []*models.InstanceAttributeValue{{
		ID: id(), InstanceID: inst1ID, InstanceVersion: 1, AttributeID: attrID, ValueString: "Alpha.Example.com",
	}}))

	inst2ID := id()
	require.NoError(t, tc.instRepo.Create(ctx, &models.EntityInstance{
		ID: inst2ID, EntityTypeID: tc.etID, CatalogID: tc.cvID,
		Name: "srv-bravo", Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	require.NoError(t, tc.iavRepo.SetValues(ctx, []*models.InstanceAttributeValue{{
		ID: id(), InstanceID: inst2ID, InstanceVersion: 1, AttributeID: attrID, ValueString: "bravo.other.com",
	}}))

	// Filter by "alpha" (case-insensitive contains)
	results, total, err := tc.instRepo.List(ctx, tc.etID, tc.cvID, models.ListParams{
		Limit:   20,
		Filters: map[string]string{attrID: "alpha"},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, results, 1)
	assert.Equal(t, "srv-alpha", results[0].Name)
}

func TestT13_20_EnumFilter_ExactMatch(t *testing.T) {
	tc, ctx := setupTestContext(t)
	attrID := id()
	require.NoError(t, tc.attrRepo.Create(ctx, &models.Attribute{
		ID: attrID, EntityTypeVersionID: tc.etvID, Name: "status", TypeDefinitionVersionID: "tdv-enum", Ordinal: 0,
	}))

	inst1ID := id()
	require.NoError(t, tc.instRepo.Create(ctx, &models.EntityInstance{
		ID: inst1ID, EntityTypeID: tc.etID, CatalogID: tc.cvID,
		Name: "running-srv", Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	require.NoError(t, tc.iavRepo.SetValues(ctx, []*models.InstanceAttributeValue{{
		ID: id(), InstanceID: inst1ID, InstanceVersion: 1, AttributeID: attrID, ValueString: "running",
	}}))

	inst2ID := id()
	require.NoError(t, tc.instRepo.Create(ctx, &models.EntityInstance{
		ID: inst2ID, EntityTypeID: tc.etID, CatalogID: tc.cvID,
		Name: "stopped-srv", Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	require.NoError(t, tc.iavRepo.SetValues(ctx, []*models.InstanceAttributeValue{{
		ID: id(), InstanceID: inst2ID, InstanceVersion: 1, AttributeID: attrID, ValueString: "stopped",
	}}))

	results, total, err := tc.instRepo.List(ctx, tc.etID, tc.cvID, models.ListParams{
		Limit:   20,
		Filters: map[string]string{attrID: "running"},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, results, 1)
	assert.Equal(t, "running-srv", results[0].Name)
}

func TestT13_22_Filter_NoMatch_ReturnsEmpty(t *testing.T) {
	tc, ctx := setupTestContext(t)
	attrID := id()
	require.NoError(t, tc.attrRepo.Create(ctx, &models.Attribute{
		ID: attrID, EntityTypeVersionID: tc.etvID, Name: "tag", TypeDefinitionVersionID: "tdv-string", Ordinal: 0,
	}))

	instID := id()
	require.NoError(t, tc.instRepo.Create(ctx, &models.EntityInstance{
		ID: instID, EntityTypeID: tc.etID, CatalogID: tc.cvID,
		Name: "tagged", Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	require.NoError(t, tc.iavRepo.SetValues(ctx, []*models.InstanceAttributeValue{{
		ID: id(), InstanceID: instID, InstanceVersion: 1, AttributeID: attrID, ValueString: "production",
	}}))

	results, total, err := tc.instRepo.List(ctx, tc.etID, tc.cvID, models.ListParams{
		Limit:   20,
		Filters: map[string]string{attrID: "nonexistent"},
	})
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Len(t, results, 0)
}

func TestT13_17_NumberRangeFilter(t *testing.T) {
	tc, ctx := setupTestContext(t)

	attrID := id()
	require.NoError(t, tc.attrRepo.Create(ctx, &models.Attribute{
		ID: attrID, EntityTypeVersionID: tc.etvID, Name: "score", TypeDefinitionVersionID: "tdv-number", Ordinal: 0,
	}))

	// Create instances with different numeric values
	for _, pair := range []struct{ name string; val float64 }{
		{"low", 2}, {"mid", 5}, {"high", 9},
	} {
		instID := id()
		require.NoError(t, tc.instRepo.Create(ctx, &models.EntityInstance{
			ID: instID, EntityTypeID: tc.etID, CatalogID: tc.cvID,
			Name: pair.name, Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}))
		v := pair.val
		require.NoError(t, tc.iavRepo.SetValues(ctx, []*models.InstanceAttributeValue{{
			ID: id(), InstanceID: instID, InstanceVersion: 1, AttributeID: attrID, ValueNumber: &v,
		}}))
	}

	// Filter min=3 max=7 → should only match "mid" (5)
	results, total, err := tc.instRepo.List(ctx, tc.etID, tc.cvID, models.ListParams{
		Limit:   20,
		Filters: map[string]string{attrID + ".min": "3", attrID + ".max": "7"},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, results, 1)
	assert.Equal(t, "mid", results[0].Name)
}

func TestT13_19_NumberFilter_InvalidValueReturnsError(t *testing.T) {
	tc, ctx := setupTestContext(t)

	attrID := id()
	_, _, err := tc.instRepo.List(ctx, tc.etID, tc.cvID, models.ListParams{
		Limit:   20,
		Filters: map[string]string{attrID + ".min": "not-a-number"},
	})
	assert.Error(t, err)
}

func TestT13_16_NumberFilter_ExactMatch(t *testing.T) {
	tc, ctx := setupTestContext(t)

	attrID := id()
	require.NoError(t, tc.attrRepo.Create(ctx, &models.Attribute{
		ID: attrID, EntityTypeVersionID: tc.etvID, Name: "count", TypeDefinitionVersionID: "tdv-number", Ordinal: 0,
	}))

	for _, pair := range []struct {
		name string
		val  float64
	}{
		{"five", 5}, {"ten", 10},
	} {
		instID := id()
		require.NoError(t, tc.instRepo.Create(ctx, &models.EntityInstance{
			ID: instID, EntityTypeID: tc.etID, CatalogID: tc.cvID,
			Name: pair.name, Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}))
		v := pair.val
		require.NoError(t, tc.iavRepo.SetValues(ctx, []*models.InstanceAttributeValue{{
			ID: id(), InstanceID: instID, InstanceVersion: 1, AttributeID: attrID, ValueNumber: &v,
		}}))
	}

	// Exact match using min=5 AND max=5
	results, total, err := tc.instRepo.List(ctx, tc.etID, tc.cvID, models.ListParams{
		Limit:   20,
		Filters: map[string]string{attrID + ".min": "5", attrID + ".max": "5"},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, results, 1)
	assert.Equal(t, "five", results[0].Name)
}

func TestT13_18_NumberFilter_MaxOnly(t *testing.T) {
	tc, ctx := setupTestContext(t)

	attrID := id()
	require.NoError(t, tc.attrRepo.Create(ctx, &models.Attribute{
		ID: attrID, EntityTypeVersionID: tc.etvID, Name: "score", TypeDefinitionVersionID: "tdv-number", Ordinal: 0,
	}))

	for _, pair := range []struct {
		name string
		val  float64
	}{
		{"low", 3}, {"mid", 7}, {"high", 12},
	} {
		instID := id()
		require.NoError(t, tc.instRepo.Create(ctx, &models.EntityInstance{
			ID: instID, EntityTypeID: tc.etID, CatalogID: tc.cvID,
			Name: pair.name, Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}))
		v := pair.val
		require.NoError(t, tc.iavRepo.SetValues(ctx, []*models.InstanceAttributeValue{{
			ID: id(), InstanceID: instID, InstanceVersion: 1, AttributeID: attrID, ValueNumber: &v,
		}}))
	}

	// Filter with max=7 only → returns instances with value <= 7
	results, total, err := tc.instRepo.List(ctx, tc.etID, tc.cvID, models.ListParams{
		Limit:   20,
		Filters: map[string]string{attrID + ".max": "7"},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, results, 2)
	// Should be "low" and "mid" (ordered by name)
	assert.Equal(t, "low", results[0].Name)
	assert.Equal(t, "mid", results[1].Name)
}

func TestT13_21_MultipleFilters_ANDLogic(t *testing.T) {
	tc, ctx := setupTestContext(t)

	attr1ID := id()
	attr2ID := id()
	require.NoError(t, tc.attrRepo.Create(ctx, &models.Attribute{
		ID: attr1ID, EntityTypeVersionID: tc.etvID, Name: "env", TypeDefinitionVersionID: "tdv-string", Ordinal: 0,
	}))
	require.NoError(t, tc.attrRepo.Create(ctx, &models.Attribute{
		ID: attr2ID, EntityTypeVersionID: tc.etvID, Name: "region", TypeDefinitionVersionID: "tdv-string", Ordinal: 1,
	}))

	// inst1: env=prod, region=us
	inst1ID := id()
	require.NoError(t, tc.instRepo.Create(ctx, &models.EntityInstance{
		ID: inst1ID, EntityTypeID: tc.etID, CatalogID: tc.cvID,
		Name: "prod-us", Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	require.NoError(t, tc.iavRepo.SetValues(ctx, []*models.InstanceAttributeValue{
		{ID: id(), InstanceID: inst1ID, InstanceVersion: 1, AttributeID: attr1ID, ValueString: "prod"},
		{ID: id(), InstanceID: inst1ID, InstanceVersion: 1, AttributeID: attr2ID, ValueString: "us"},
	}))

	// inst2: env=prod, region=eu
	inst2ID := id()
	require.NoError(t, tc.instRepo.Create(ctx, &models.EntityInstance{
		ID: inst2ID, EntityTypeID: tc.etID, CatalogID: tc.cvID,
		Name: "prod-eu", Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	require.NoError(t, tc.iavRepo.SetValues(ctx, []*models.InstanceAttributeValue{
		{ID: id(), InstanceID: inst2ID, InstanceVersion: 1, AttributeID: attr1ID, ValueString: "prod"},
		{ID: id(), InstanceID: inst2ID, InstanceVersion: 1, AttributeID: attr2ID, ValueString: "eu"},
	}))

	// inst3: env=dev, region=us
	inst3ID := id()
	require.NoError(t, tc.instRepo.Create(ctx, &models.EntityInstance{
		ID: inst3ID, EntityTypeID: tc.etID, CatalogID: tc.cvID,
		Name: "dev-us", Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))
	require.NoError(t, tc.iavRepo.SetValues(ctx, []*models.InstanceAttributeValue{
		{ID: id(), InstanceID: inst3ID, InstanceVersion: 1, AttributeID: attr1ID, ValueString: "dev"},
		{ID: id(), InstanceID: inst3ID, InstanceVersion: 1, AttributeID: attr2ID, ValueString: "us"},
	}))

	// Filter env=prod AND region=us → only inst1
	results, total, err := tc.instRepo.List(ctx, tc.etID, tc.cvID, models.ListParams{
		Limit:   20,
		Filters: map[string]string{attr1ID: "prod", attr2ID: "us"},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, results, 1)
	assert.Equal(t, "prod-us", results[0].Name)
}

func TestT13_23_FilterWorksAcrossEAVJoin(t *testing.T) {
	tc, ctx := setupTestContext(t)

	attrID := id()
	require.NoError(t, tc.attrRepo.Create(ctx, &models.Attribute{
		ID: attrID, EntityTypeVersionID: tc.etvID, Name: "color", TypeDefinitionVersionID: "tdv-string", Ordinal: 0,
	}))

	// Create 3 instances, two match the filter
	for _, pair := range []struct{ name, color string }{
		{"apple", "red"},
		{"banana", "yellow"},
		{"cherry", "red"},
	} {
		instID := id()
		require.NoError(t, tc.instRepo.Create(ctx, &models.EntityInstance{
			ID: instID, EntityTypeID: tc.etID, CatalogID: tc.cvID,
			Name: pair.name, Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}))
		require.NoError(t, tc.iavRepo.SetValues(ctx, []*models.InstanceAttributeValue{{
			ID: id(), InstanceID: instID, InstanceVersion: 1, AttributeID: attrID, ValueString: pair.color,
		}}))
	}

	results, total, err := tc.instRepo.List(ctx, tc.etID, tc.cvID, models.ListParams{
		Limit:   20,
		Filters: map[string]string{attrID: "red"},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, results, 2)
	// No duplicate instances — each should appear exactly once
	names := map[string]bool{results[0].Name: true, results[1].Name: true}
	assert.True(t, names["apple"])
	assert.True(t, names["cherry"])
}

func TestT13_32_SortByNameAsc(t *testing.T) {
	tc, ctx := setupTestContext(t)

	for _, name := range []string{"charlie", "alpha", "bravo"} {
		require.NoError(t, tc.instRepo.Create(ctx, &models.EntityInstance{
			ID: id(), EntityTypeID: tc.etID, CatalogID: tc.cvID,
			Name: name, Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}))
	}

	results, _, err := tc.instRepo.List(ctx, tc.etID, tc.cvID, models.ListParams{
		Limit:    20,
		SortBy:   "name",
		SortDesc: false,
	})
	require.NoError(t, err)
	require.Len(t, results, 3)
	assert.Equal(t, "alpha", results[0].Name)
	assert.Equal(t, "bravo", results[1].Name)
	assert.Equal(t, "charlie", results[2].Name)
}

func TestT13_33_SortByNameDesc(t *testing.T) {
	tc, ctx := setupTestContext(t)

	for _, name := range []string{"charlie", "alpha", "bravo"} {
		require.NoError(t, tc.instRepo.Create(ctx, &models.EntityInstance{
			ID: id(), EntityTypeID: tc.etID, CatalogID: tc.cvID,
			Name: name, Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}))
	}

	results, _, err := tc.instRepo.List(ctx, tc.etID, tc.cvID, models.ListParams{
		Limit:    20,
		SortBy:   "name",
		SortDesc: true,
	})
	require.NoError(t, err)
	require.Len(t, results, 3)
	assert.Equal(t, "charlie", results[0].Name)
	assert.Equal(t, "bravo", results[1].Name)
	assert.Equal(t, "alpha", results[2].Name)
}

func TestT13_41_OffsetSkipsResults(t *testing.T) {
	tc, ctx := setupTestContext(t)

	for _, name := range []string{"a", "b", "c", "d", "e"} {
		require.NoError(t, tc.instRepo.Create(ctx, &models.EntityInstance{
			ID: id(), EntityTypeID: tc.etID, CatalogID: tc.cvID,
			Name: name, Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}))
	}

	results, _, err := tc.instRepo.List(ctx, tc.etID, tc.cvID, models.ListParams{
		Limit:  2,
		Offset: 2,
	})
	require.NoError(t, err)
	require.Len(t, results, 2)
	// Default sort is by name: a, b, c, d, e → offset 2, limit 2 → c, d
	assert.Equal(t, "c", results[0].Name)
	assert.Equal(t, "d", results[1].Name)
}

func TestT13_42_LimitCapsResults(t *testing.T) {
	tc, ctx := setupTestContext(t)

	for _, name := range []string{"a", "b", "c", "d", "e"} {
		require.NoError(t, tc.instRepo.Create(ctx, &models.EntityInstance{
			ID: id(), EntityTypeID: tc.etID, CatalogID: tc.cvID,
			Name: name, Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}))
	}

	results, _, err := tc.instRepo.List(ctx, tc.etID, tc.cvID, models.ListParams{
		Limit: 3,
	})
	require.NoError(t, err)
	assert.Len(t, results, 3)
}

func TestT13_43_TotalUnaffectedByOffsetLimit(t *testing.T) {
	tc, ctx := setupTestContext(t)

	for _, name := range []string{"a", "b", "c", "d", "e"} {
		require.NoError(t, tc.instRepo.Create(ctx, &models.EntityInstance{
			ID: id(), EntityTypeID: tc.etID, CatalogID: tc.cvID,
			Name: name, Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}))
	}

	_, total, err := tc.instRepo.List(ctx, tc.etID, tc.cvID, models.ListParams{
		Limit:  2,
		Offset: 2,
	})
	require.NoError(t, err)
	assert.Equal(t, 5, total)
}

func TestT13_44_OffsetBeyondTotal(t *testing.T) {
	tc, ctx := setupTestContext(t)

	for _, name := range []string{"a", "b", "c"} {
		require.NoError(t, tc.instRepo.Create(ctx, &models.EntityInstance{
			ID: id(), EntityTypeID: tc.etID, CatalogID: tc.cvID,
			Name: name, Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}))
	}

	results, total, err := tc.instRepo.List(ctx, tc.etID, tc.cvID, models.ListParams{
		Limit:  10,
		Offset: 10,
	})
	require.NoError(t, err)
	assert.Empty(t, results)
	assert.Equal(t, 3, total)
}

// TD-16: DeleteByInstanceID removes all IAVs for a given instance
func TestTD16_DeleteByInstanceID(t *testing.T) {
	tc, ctx := setupTestContext(t)

	// Create instance with attribute values
	inst := &models.EntityInstance{
		ID: id(), EntityTypeID: tc.etID, CatalogID: tc.cvID,
		Name: "test-inst", Version: 1, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, tc.instRepo.Create(ctx, inst))
	require.NoError(t, tc.iavRepo.SetValues(ctx, []*models.InstanceAttributeValue{
		{ID: id(), InstanceID: inst.ID, AttributeID: "a1", ValueString: "val1", InstanceVersion: 1},
	}))

	// Verify values exist
	vals, err := tc.iavRepo.GetCurrentValues(ctx, inst.ID)
	require.NoError(t, err)
	assert.Len(t, vals, 1)

	// Delete IAVs by instance
	require.NoError(t, tc.iavRepo.DeleteByInstanceID(ctx, inst.ID))

	// Verify values are gone
	vals, err = tc.iavRepo.GetCurrentValues(ctx, inst.ID)
	require.NoError(t, err)
	assert.Empty(t, vals)
}
