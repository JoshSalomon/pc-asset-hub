package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	gormmodels "github.com/project-catalyst/pc-asset-hub/internal/infrastructure/gorm/models"
	"github.com/project-catalyst/pc-asset-hub/internal/infrastructure/gorm/repository"
)

// closedDB creates a DB, migrates it, then closes the underlying connection.
// All subsequent operations will fail with a non-ErrRecordNotFound error,
// exercising the `return nil, result.Error` branches.
func closedDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&_busy_timeout=1"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := gormmodels.InitDB(db); err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	_ = sqlDB.Close()
	return db
}

func TestEntityType_ErrorBranches(t *testing.T) {
	db := closedDB(t)
	repo := repository.NewEntityTypeGormRepo(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "x")
	assert.Error(t, err)

	_, err = repo.GetByName(ctx, "x")
	assert.Error(t, err)

	_, _, err = repo.List(ctx, models.ListParams{Limit: 10})
	assert.Error(t, err)

	err = repo.Create(ctx, &models.EntityType{ID: "x", Name: "x", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	assert.Error(t, err)

	err = repo.Update(ctx, &models.EntityType{ID: "x", Name: "x", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	assert.Error(t, err)

	err = repo.Delete(ctx, "x")
	assert.Error(t, err)
}

func TestEntityTypeVersion_ErrorBranches(t *testing.T) {
	db := closedDB(t)
	repo := repository.NewEntityTypeVersionGormRepo(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "x")
	assert.Error(t, err)

	_, err = repo.GetByEntityTypeAndVersion(ctx, "x", 1)
	assert.Error(t, err)

	_, err = repo.GetLatestByEntityType(ctx, "x")
	assert.Error(t, err)

	_, err = repo.ListByEntityType(ctx, "x")
	assert.Error(t, err)

	_, err = repo.GetLatestByEntityTypes(ctx, []string{"x"})
	assert.Error(t, err)

	err = repo.Create(ctx, &models.EntityTypeVersion{ID: "x", EntityTypeID: "et1", Version: 1, CreatedAt: time.Now()})
	assert.Error(t, err)
}

func TestAttribute_ErrorBranches(t *testing.T) {
	db := closedDB(t)
	repo := repository.NewAttributeGormRepo(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "x")
	assert.Error(t, err)

	_, err = repo.ListByVersion(ctx, "x")
	assert.Error(t, err)

	err = repo.Create(ctx, &models.Attribute{ID: "x", EntityTypeVersionID: "v1", Name: "a", Type: models.AttributeTypeString})
	assert.Error(t, err)

	err = repo.Update(ctx, &models.Attribute{ID: "x", EntityTypeVersionID: "v1", Name: "a", Type: models.AttributeTypeString})
	assert.Error(t, err)

	err = repo.Delete(ctx, "x")
	assert.Error(t, err)

	err = repo.BulkCopyToVersion(ctx, "x", "y")
	assert.Error(t, err)

	err = repo.Reorder(ctx, "v1", []string{"a1"})
	assert.Error(t, err)
}

func TestAssociation_ErrorBranches(t *testing.T) {
	db := closedDB(t)
	repo := repository.NewAssociationGormRepo(db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "x")
	assert.Error(t, err)

	_, err = repo.ListByVersion(ctx, "x")
	assert.Error(t, err)

	err = repo.Create(ctx, &models.Association{ID: "x", EntityTypeVersionID: "v1", TargetEntityTypeID: "et2", Type: models.AssociationTypeContainment, CreatedAt: time.Now()})
	assert.Error(t, err)

	err = repo.Delete(ctx, "x")
	assert.Error(t, err)

	err = repo.BulkCopyToVersion(ctx, "x", "y")
	assert.Error(t, err)

	_, err = repo.GetContainmentGraph(ctx)
	assert.Error(t, err)
}

func TestEnum_ErrorBranches(t *testing.T) {
	db := closedDB(t)
	enumRepo := repository.NewEnumGormRepo(db)
	evRepo := repository.NewEnumValueGormRepo(db)
	ctx := context.Background()

	_, err := enumRepo.GetByID(ctx, "x")
	assert.Error(t, err)

	_, err = enumRepo.GetByName(ctx, "x")
	assert.Error(t, err)

	_, _, err = enumRepo.List(ctx, models.ListParams{Limit: 10})
	assert.Error(t, err)

	err = enumRepo.Create(ctx, &models.Enum{ID: "x", Name: "x", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	assert.Error(t, err)

	err = enumRepo.Update(ctx, &models.Enum{ID: "x", Name: "x", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	assert.Error(t, err)

	err = enumRepo.Delete(ctx, "x")
	assert.Error(t, err)

	err = evRepo.Create(ctx, &models.EnumValue{ID: "x", EnumID: "e1", Value: "v", Ordinal: 0})
	assert.Error(t, err)

	_, err = evRepo.ListByEnum(ctx, "x")
	assert.Error(t, err)

	err = evRepo.Delete(ctx, "x")
	assert.Error(t, err)

	err = evRepo.Reorder(ctx, "e1", []string{"v1"})
	assert.Error(t, err)
}

func TestCatalogVersion_ErrorBranches(t *testing.T) {
	db := closedDB(t)
	cvRepo := repository.NewCatalogVersionGormRepo(db)
	pinRepo := repository.NewCatalogVersionPinGormRepo(db)
	ltRepo := repository.NewLifecycleTransitionGormRepo(db)
	ctx := context.Background()

	_, err := cvRepo.GetByID(ctx, "x")
	assert.Error(t, err)

	_, err = cvRepo.GetByLabel(ctx, "x")
	assert.Error(t, err)

	_, _, err = cvRepo.List(ctx, models.ListParams{Limit: 10})
	assert.Error(t, err)

	err = cvRepo.Create(ctx, &models.CatalogVersion{ID: "x", VersionLabel: "v", LifecycleStage: "development", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	assert.Error(t, err)

	err = cvRepo.UpdateLifecycle(ctx, "x", models.LifecycleStageTesting)
	assert.Error(t, err)

	err = pinRepo.Create(ctx, &models.CatalogVersionPin{ID: "x", CatalogVersionID: "cv1", EntityTypeVersionID: "v1"})
	assert.Error(t, err)

	_, err = pinRepo.ListByCatalogVersion(ctx, "x")
	assert.Error(t, err)

	err = pinRepo.Delete(ctx, "x")
	assert.Error(t, err)

	err = ltRepo.Create(ctx, &models.LifecycleTransition{ID: "x", CatalogVersionID: "cv1", ToStage: "testing", PerformedBy: "admin", PerformedAt: time.Now()})
	assert.Error(t, err)

	_, err = ltRepo.ListByCatalogVersion(ctx, "x")
	assert.Error(t, err)

	err = cvRepo.Update(ctx, &models.CatalogVersion{ID: "x", VersionLabel: "v", LifecycleStage: "development", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	assert.Error(t, err)

	_, err = pinRepo.GetByID(ctx, "x")
	assert.Error(t, err)

	_, err = pinRepo.ListByEntityTypeVersionIDs(ctx, []string{"x"})
	assert.Error(t, err)
}

func TestEntityInstance_ErrorBranches(t *testing.T) {
	db := closedDB(t)
	instRepo := repository.NewEntityInstanceGormRepo(db)
	iavRepo := repository.NewInstanceAttributeValueGormRepo(db)
	linkRepo := repository.NewAssociationLinkGormRepo(db)
	ctx := context.Background()

	_, err := instRepo.GetByID(ctx, "x")
	assert.Error(t, err)

	_, err = instRepo.GetByNameAndParent(ctx, "et1", "cv1", "", "x")
	assert.Error(t, err)

	_, _, err = instRepo.List(ctx, "et1", "cv1", models.ListParams{Limit: 10})
	assert.Error(t, err)

	_, _, err = instRepo.ListByParent(ctx, "x", models.ListParams{Limit: 10})
	assert.Error(t, err)

	err = instRepo.Create(ctx, &models.EntityInstance{ID: "x", EntityTypeID: "et1", CatalogID: "cv1", Name: "n", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	assert.Error(t, err)

	err = instRepo.Update(ctx, &models.EntityInstance{ID: "x", EntityTypeID: "et1", CatalogID: "cv1", Name: "n", CreatedAt: time.Now(), UpdatedAt: time.Now()})
	assert.Error(t, err)

	err = instRepo.SoftDelete(ctx, "x")
	assert.Error(t, err)

	err = iavRepo.SetValues(ctx, []*models.InstanceAttributeValue{{ID: "x", InstanceID: "i1", InstanceVersion: 1, AttributeID: "a1"}})
	assert.Error(t, err)

	_, err = iavRepo.GetCurrentValues(ctx, "x")
	assert.Error(t, err)

	_, err = iavRepo.GetValuesForVersion(ctx, "x", 1)
	assert.Error(t, err)

	err = linkRepo.Create(ctx, &models.AssociationLink{ID: "x", AssociationID: "a1", SourceInstanceID: "i1", TargetInstanceID: "i2", CreatedAt: time.Now()})
	assert.Error(t, err)

	err = linkRepo.Delete(ctx, "x")
	assert.Error(t, err)

	_, err = linkRepo.GetForwardRefs(ctx, "x")
	assert.Error(t, err)

	_, err = linkRepo.GetReverseRefs(ctx, "x")
	assert.Error(t, err)

	err = instRepo.DeleteByCatalogID(ctx, "x")
	assert.Error(t, err)
}

func TestCatalog_ErrorBranches(t *testing.T) {
	db := closedDB(t)
	repo := repository.NewCatalogGormRepo(db)
	ctx := context.Background()

	err := repo.Create(ctx, &models.Catalog{ID: "x", Name: "n", CatalogVersionID: "cv1", ValidationStatus: models.ValidationStatusDraft, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	assert.Error(t, err)

	_, err = repo.GetByName(ctx, "x")
	assert.Error(t, err)

	_, err = repo.GetByID(ctx, "x")
	assert.Error(t, err)

	_, _, err = repo.List(ctx, models.ListParams{Limit: 10})
	assert.Error(t, err)

	err = repo.Delete(ctx, "x")
	assert.Error(t, err)

	err = repo.UpdateValidationStatus(ctx, "x", models.ValidationStatusValid)
	assert.Error(t, err)

	err = repo.Update(ctx, &models.Catalog{ID: "x", Name: "n", CatalogVersionID: "cv1", ValidationStatus: models.ValidationStatusDraft, CreatedAt: time.Now(), UpdatedAt: time.Now()})
	assert.Error(t, err)
}
