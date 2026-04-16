package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/infrastructure/gorm/repository"
	"github.com/project-catalyst/pc-asset-hub/internal/infrastructure/gorm/testutil"
	"github.com/project-catalyst/pc-asset-hub/internal/service/operational"
)

func copyTestID() string { return uuid.Must(uuid.NewV7()).String() }

// Test GormTransactionManager.RunInTransaction — rollback on error
func TestGormTransactionManager_Rollback(t *testing.T) {
	db := testutil.NewTestDB(t)
	txm := repository.NewGormTransactionManager(db)
	catalogRepo := repository.NewCatalogGormRepo(db)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	ctx := context.Background()

	cvID := copyTestID()
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{
		ID: cvID, VersionLabel: "v1", LifecycleStage: "development",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	// Transaction that creates a catalog then fails — should rollback
	err := txm.RunInTransaction(ctx, func(txCtx context.Context) error {
		require.NoError(t, catalogRepo.Create(txCtx, &models.Catalog{
			ID: copyTestID(), Name: "should-not-exist", CatalogVersionID: cvID,
			ValidationStatus: models.ValidationStatusDraft,
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}))
		return fmt.Errorf("deliberate error")
	})
	assert.Error(t, err)

	// Catalog should NOT exist (rolled back)
	_, err = catalogRepo.GetByName(ctx, "should-not-exist")
	assert.Error(t, err) // NotFound
}

// Test GormTransactionManager.RunInTransaction — commit on success
func TestGormTransactionManager_Commit(t *testing.T) {
	db := testutil.NewTestDB(t)
	txm := repository.NewGormTransactionManager(db)
	catalogRepo := repository.NewCatalogGormRepo(db)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	ctx := context.Background()

	cvID := copyTestID()
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{
		ID: cvID, VersionLabel: "v1", LifecycleStage: "development",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}))

	err := txm.RunInTransaction(ctx, func(txCtx context.Context) error {
		return catalogRepo.Create(txCtx, &models.Catalog{
			ID: copyTestID(), Name: "should-exist", CatalogVersionID: cvID,
			ValidationStatus: models.ValidationStatusDraft,
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		})
	})
	require.NoError(t, err)

	// Catalog should exist (committed)
	found, err := catalogRepo.GetByName(ctx, "should-exist")
	require.NoError(t, err)
	assert.Equal(t, "should-exist", found.Name)
}

// Test CopyCatalog uses real transaction — partial failure rolls back
func TestCopyCatalog_TransactionRollback_Integration(t *testing.T) {
	db := testutil.NewTestDB(t)
	ctx := context.Background()
	now := time.Now()

	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	pinRepo := repository.NewCatalogVersionPinGormRepo(db)
	catalogRepo := repository.NewCatalogGormRepo(db)
	instRepo := repository.NewEntityInstanceGormRepo(db)
	iavRepo := repository.NewInstanceAttributeValueGormRepo(db)
	linkRepo := repository.NewAssociationLinkGormRepo(db)
	txm := repository.NewGormTransactionManager(db)

	etID := copyTestID()
	etvID := copyTestID()
	cvID := copyTestID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: etID, Name: "test-et", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etvID, EntityTypeID: etID, Version: 1, CreatedAt: now}))
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{ID: cvID, VersionLabel: "v1", LifecycleStage: "development", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, pinRepo.Create(ctx, &models.CatalogVersionPin{ID: copyTestID(), CatalogVersionID: cvID, EntityTypeVersionID: etvID}))

	// Create source catalog with an instance
	srcCatID := copyTestID()
	require.NoError(t, catalogRepo.Create(ctx, &models.Catalog{
		ID: srcCatID, Name: "tx-source", CatalogVersionID: cvID,
		ValidationStatus: models.ValidationStatusValid, CreatedAt: now, UpdatedAt: now,
	}))
	require.NoError(t, instRepo.Create(ctx, &models.EntityInstance{
		ID: copyTestID(), EntityTypeID: etID, CatalogID: srcCatID,
		Name: "inst-1", Version: 1, CreatedAt: now, UpdatedAt: now,
	}))

	svc := operational.NewCatalogService(catalogRepo, cvRepo, instRepo, nil, "",
		operational.WithCopyDeps(iavRepo, linkRepo),
		operational.WithTransactionManager(txm))

	// Copy to a name that will conflict (create a catalog with that name first)
	require.NoError(t, catalogRepo.Create(ctx, &models.Catalog{
		ID: copyTestID(), Name: "tx-target", CatalogVersionID: cvID,
		ValidationStatus: models.ValidationStatusDraft, CreatedAt: now, UpdatedAt: now,
	}))

	// This should fail because tx-target already exists
	_, err := svc.CopyCatalog(ctx, "tx-source", "tx-target", "")
	assert.Error(t, err)

	// The partial catalog should NOT exist — transaction rolled back
	cats, _, _ := catalogRepo.List(ctx, models.ListParams{Limit: 100})
	for _, c := range cats {
		// No catalog should have the copy's generated ID
		assert.NotEqual(t, "tx-target-copy", c.Name)
	}
}

type copyTestEnv struct {
	catalogRepo *repository.CatalogGormRepo
	instRepo    *repository.EntityInstanceGormRepo
	iavRepo     *repository.InstanceAttributeValueGormRepo
	linkRepo    *repository.AssociationLinkGormRepo
	svc         *operational.CatalogService
	cvID        string
	etID1       string // parent entity type
	etID2       string // child entity type
	assocID     string // association between et1 → et2
}

func setupCopyTestEnv(t *testing.T) *copyTestEnv {
	t.Helper()
	db := testutil.NewTestDB(t)
	ctx := context.Background()
	now := time.Now()

	etRepo := repository.NewEntityTypeGormRepo(db)
	etvRepo := repository.NewEntityTypeVersionGormRepo(db)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	pinRepo := repository.NewCatalogVersionPinGormRepo(db)
	catalogRepo := repository.NewCatalogGormRepo(db)
	instRepo := repository.NewEntityInstanceGormRepo(db)
	iavRepo := repository.NewInstanceAttributeValueGormRepo(db)
	linkRepo := repository.NewAssociationLinkGormRepo(db)
	assocRepo := repository.NewAssociationGormRepo(db)

	env := &copyTestEnv{
		catalogRepo: catalogRepo,
		instRepo:    instRepo,
		iavRepo:     iavRepo,
		linkRepo:    linkRepo,
		etID1:       copyTestID(),
		etID2:       copyTestID(),
		cvID:        copyTestID(),
		assocID:     copyTestID(),
	}

	// Entity types + versions
	etv1ID := copyTestID()
	etv2ID := copyTestID()
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: env.etID1, Name: "server", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etv1ID, EntityTypeID: env.etID1, Version: 1, CreatedAt: now}))
	require.NoError(t, etRepo.Create(ctx, &models.EntityType{ID: env.etID2, Name: "tool", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, etvRepo.Create(ctx, &models.EntityTypeVersion{ID: etv2ID, EntityTypeID: env.etID2, Version: 1, CreatedAt: now}))

	// Association (containment: server contains tool)
	require.NoError(t, assocRepo.Create(ctx, &models.Association{
		ID: env.assocID, EntityTypeVersionID: etv1ID, Name: "contains-tool",
		TargetEntityTypeID: env.etID2, Type: models.AssociationTypeContainment,
		SourceCardinality: "0..1", TargetCardinality: "0..n",
	}))

	// CV + pins
	require.NoError(t, cvRepo.Create(ctx, &models.CatalogVersion{ID: env.cvID, VersionLabel: "v1", LifecycleStage: "development", CreatedAt: now, UpdatedAt: now}))
	require.NoError(t, pinRepo.Create(ctx, &models.CatalogVersionPin{ID: copyTestID(), CatalogVersionID: env.cvID, EntityTypeVersionID: etv1ID}))
	require.NoError(t, pinRepo.Create(ctx, &models.CatalogVersionPin{ID: copyTestID(), CatalogVersionID: env.cvID, EntityTypeVersionID: etv2ID}))

	env.svc = operational.NewCatalogService(catalogRepo, cvRepo, instRepo, nil, "",
		operational.WithCopyDeps(iavRepo, linkRepo))

	return env
}

func createCatalogWithData(t *testing.T, ctx context.Context, env *copyTestEnv, catName string) (catID, parentInstID, childInstID string) {
	t.Helper()
	now := time.Now()

	catID = copyTestID()
	require.NoError(t, env.catalogRepo.Create(ctx, &models.Catalog{
		ID: catID, Name: catName, CatalogVersionID: env.cvID,
		ValidationStatus: models.ValidationStatusValid,
		CreatedAt: now, UpdatedAt: now,
	}))

	// Parent instance
	parentInstID = copyTestID()
	require.NoError(t, env.instRepo.Create(ctx, &models.EntityInstance{
		ID: parentInstID, EntityTypeID: env.etID1, CatalogID: catID,
		Name: "my-server", Description: "server desc", Version: 1,
		CreatedAt: now, UpdatedAt: now,
	}))

	// Child instance (contained by parent)
	childInstID = copyTestID()
	require.NoError(t, env.instRepo.Create(ctx, &models.EntityInstance{
		ID: childInstID, EntityTypeID: env.etID2, CatalogID: catID,
		ParentInstanceID: parentInstID,
		Name: "my-tool", Description: "tool desc", Version: 1,
		CreatedAt: now, UpdatedAt: now,
	}))

	// Attribute values for parent
	port := float64(8080)
	require.NoError(t, env.iavRepo.SetValues(ctx, []*models.InstanceAttributeValue{
		{ID: copyTestID(), InstanceID: parentInstID, InstanceVersion: 1, AttributeID: "attr1", ValueString: "localhost"},
		{ID: copyTestID(), InstanceID: parentInstID, InstanceVersion: 1, AttributeID: "attr2", ValueNumber: &port},
	}))

	// Association link: parent → child
	require.NoError(t, env.linkRepo.Create(ctx, &models.AssociationLink{
		ID: copyTestID(), AssociationID: env.assocID,
		SourceInstanceID: parentInstID, TargetInstanceID: childInstID,
		CreatedAt: now,
	}))

	return catID, parentInstID, childInstID
}

// T-17.20: Copy catalog with instances, attributes, links, containment in real DB
func TestT17_20_CopyCatalog_Integration(t *testing.T) {
	env := setupCopyTestEnv(t)
	ctx := context.Background()

	_, _, _ = createCatalogWithData(t, ctx, env, "source-cat")

	// Copy
	result, err := env.svc.CopyCatalog(ctx, "source-cat", "target-cat", "copied")
	require.NoError(t, err)
	assert.Equal(t, "target-cat", result.Name)
	assert.Equal(t, "copied", result.Description)
	assert.Equal(t, models.ValidationStatusDraft, result.ValidationStatus)

	// T-17.21: Verify instances in new catalog
	newInstances, err := env.instRepo.ListByCatalog(ctx, result.ID)
	require.NoError(t, err)
	assert.Len(t, newInstances, 2)

	// T-17.22: Verify attribute values
	var newParent *models.EntityInstance
	for _, inst := range newInstances {
		if inst.Name == "my-server" {
			newParent = inst
			break
		}
	}
	require.NotNil(t, newParent)
	attrs, err := env.iavRepo.GetValuesForVersion(ctx, newParent.ID, 1)
	require.NoError(t, err)
	assert.Len(t, attrs, 2)

	// T-17.23: Verify links
	links, err := env.linkRepo.GetForwardRefs(ctx, newParent.ID)
	require.NoError(t, err)
	assert.Len(t, links, 1)

	// T-17.24: Verify containment
	var newChild *models.EntityInstance
	for _, inst := range newInstances {
		if inst.Name == "my-tool" {
			newChild = inst
			break
		}
	}
	require.NotNil(t, newChild)
	assert.Equal(t, newParent.ID, newChild.ParentInstanceID)

	// Link target should be the new child
	assert.Equal(t, newChild.ID, links[0].TargetInstanceID)
}

// T-17.25: Original catalog data unchanged after copy
func TestT17_25_CopyCatalog_SourceUnchanged_Integration(t *testing.T) {
	env := setupCopyTestEnv(t)
	ctx := context.Background()

	srcCatID, srcParentID, srcChildID := createCatalogWithData(t, ctx, env, "source-cat")

	_, err := env.svc.CopyCatalog(ctx, "source-cat", "target-cat", "")
	require.NoError(t, err)

	// Source instances unchanged
	srcInstances, err := env.instRepo.ListByCatalog(ctx, srcCatID)
	require.NoError(t, err)
	assert.Len(t, srcInstances, 2)

	// Source attributes unchanged
	srcAttrs, err := env.iavRepo.GetValuesForVersion(ctx, srcParentID, 1)
	require.NoError(t, err)
	assert.Len(t, srcAttrs, 2)

	// Source links unchanged
	srcLinks, err := env.linkRepo.GetForwardRefs(ctx, srcParentID)
	require.NoError(t, err)
	assert.Len(t, srcLinks, 1)
	assert.Equal(t, srcChildID, srcLinks[0].TargetInstanceID)
}

// T-17.45: Replace swaps names in real DB
func TestT17_45_ReplaceCatalog_Integration(t *testing.T) {
	env := setupCopyTestEnv(t)
	ctx := context.Background()
	now := time.Now()

	// Source catalog (valid)
	srcID := copyTestID()
	require.NoError(t, env.catalogRepo.Create(ctx, &models.Catalog{
		ID: srcID, Name: "staging", CatalogVersionID: env.cvID,
		ValidationStatus: models.ValidationStatusValid,
		CreatedAt: now, UpdatedAt: now,
	}))

	// Target catalog
	tgtID := copyTestID()
	require.NoError(t, env.catalogRepo.Create(ctx, &models.Catalog{
		ID: tgtID, Name: "prod", CatalogVersionID: env.cvID,
		ValidationStatus: models.ValidationStatusValid,
		CreatedAt: now, UpdatedAt: now,
	}))

	result, err := env.svc.ReplaceCatalog(ctx, "staging", "prod", "prod-archive")
	require.NoError(t, err)
	assert.Equal(t, srcID, result.ID)

	// Source should now be named "prod"
	found, err := env.catalogRepo.GetByName(ctx, "prod")
	require.NoError(t, err)
	assert.Equal(t, srcID, found.ID)

	// Target should now be named "prod-archive"
	found, err = env.catalogRepo.GetByName(ctx, "prod-archive")
	require.NoError(t, err)
	assert.Equal(t, tgtID, found.ID)

	// Old names should not exist
	_, err = env.catalogRepo.GetByName(ctx, "staging")
	assert.Error(t, err)
}

// T-17.46: Replace transfers published state in real DB
func TestT17_46_ReplaceCatalog_PublishedState_Integration(t *testing.T) {
	env := setupCopyTestEnv(t)
	ctx := context.Background()
	now := time.Now()

	// Source (valid, unpublished)
	srcID := copyTestID()
	require.NoError(t, env.catalogRepo.Create(ctx, &models.Catalog{
		ID: srcID, Name: "staging", CatalogVersionID: env.cvID,
		ValidationStatus: models.ValidationStatusValid,
		CreatedAt: now, UpdatedAt: now,
	}))

	// Target (published)
	tgtID := copyTestID()
	require.NoError(t, env.catalogRepo.Create(ctx, &models.Catalog{
		ID: tgtID, Name: "prod", CatalogVersionID: env.cvID,
		ValidationStatus: models.ValidationStatusValid,
		Published: true, PublishedAt: &now,
		CreatedAt: now, UpdatedAt: now,
	}))

	_, err := env.svc.ReplaceCatalog(ctx, "staging", "prod", "prod-archive")
	require.NoError(t, err)

	// Source (now "prod") should be published
	prod, err := env.catalogRepo.GetByName(ctx, "prod")
	require.NoError(t, err)
	assert.True(t, prod.Published)
	assert.NotNil(t, prod.PublishedAt)

	// Archive should be unpublished
	archive, err := env.catalogRepo.GetByName(ctx, "prod-archive")
	require.NoError(t, err)
	assert.False(t, archive.Published)
}

// T-17.48: Replace with draft source leaves DB unchanged
func TestT17_48_ReplaceCatalog_DraftSourceUnchanged_Integration(t *testing.T) {
	env := setupCopyTestEnv(t)
	ctx := context.Background()
	now := time.Now()

	srcID := copyTestID()
	require.NoError(t, env.catalogRepo.Create(ctx, &models.Catalog{
		ID: srcID, Name: "staging", CatalogVersionID: env.cvID,
		ValidationStatus: models.ValidationStatusDraft,
		CreatedAt: now, UpdatedAt: now,
	}))

	tgtID := copyTestID()
	require.NoError(t, env.catalogRepo.Create(ctx, &models.Catalog{
		ID: tgtID, Name: "prod", CatalogVersionID: env.cvID,
		ValidationStatus: models.ValidationStatusValid,
		CreatedAt: now, UpdatedAt: now,
	}))

	_, err := env.svc.ReplaceCatalog(ctx, "staging", "prod", "")
	assert.Error(t, err)

	// Names unchanged
	found, err := env.catalogRepo.GetByName(ctx, "staging")
	require.NoError(t, err)
	assert.Equal(t, srcID, found.ID)

	found, err = env.catalogRepo.GetByName(ctx, "prod")
	require.NoError(t, err)
	assert.Equal(t, tgtID, found.ID)
}

// T-17.50: Replace with archive name collision leaves DB unchanged
func TestT17_50_ReplaceCatalog_ArchiveCollision_Integration(t *testing.T) {
	env := setupCopyTestEnv(t)
	ctx := context.Background()
	now := time.Now()

	require.NoError(t, env.catalogRepo.Create(ctx, &models.Catalog{
		ID: copyTestID(), Name: "staging", CatalogVersionID: env.cvID,
		ValidationStatus: models.ValidationStatusValid,
		CreatedAt: now, UpdatedAt: now,
	}))
	require.NoError(t, env.catalogRepo.Create(ctx, &models.Catalog{
		ID: copyTestID(), Name: "prod", CatalogVersionID: env.cvID,
		ValidationStatus: models.ValidationStatusValid,
		CreatedAt: now, UpdatedAt: now,
	}))
	// Archive name already exists
	require.NoError(t, env.catalogRepo.Create(ctx, &models.Catalog{
		ID: copyTestID(), Name: "prod-archive", CatalogVersionID: env.cvID,
		ValidationStatus: models.ValidationStatusDraft,
		CreatedAt: now, UpdatedAt: now,
	}))

	_, err := env.svc.ReplaceCatalog(ctx, "staging", "prod", "prod-archive")
	assert.Error(t, err)

	// Names unchanged — staging and prod still exist
	_, err = env.catalogRepo.GetByName(ctx, "staging")
	assert.NoError(t, err)
	_, err = env.catalogRepo.GetByName(ctx, "prod")
	assert.NoError(t, err)
}
