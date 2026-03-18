package operational_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository/mocks"
	"github.com/project-catalyst/pc-asset-hub/internal/service/operational"
)

type mockCatalogCRManager struct {
	createOrUpdateSpec *operational.CatalogCRSpec
	deleteCalledWith   string
	createErr          error
	deleteErr          error
}

func (m *mockCatalogCRManager) CreateOrUpdate(_ context.Context, spec operational.CatalogCRSpec) error {
	m.createOrUpdateSpec = &spec
	return m.createErr
}

func (m *mockCatalogCRManager) Delete(_ context.Context, name, _ string) error {
	m.deleteCalledWith = name
	return m.deleteErr
}

func setupCatalogService() (*operational.CatalogService, *mocks.MockCatalogRepo, *mocks.MockCatalogVersionRepo, *mocks.MockEntityInstanceRepo) {
	catRepo := new(mocks.MockCatalogRepo)
	cvRepo := new(mocks.MockCatalogVersionRepo)
	instRepo := new(mocks.MockEntityInstanceRepo)
	svc := operational.NewCatalogService(catRepo, cvRepo, instRepo, nil, "")
	return svc, catRepo, cvRepo, instRepo
}

func setupCatalogServiceWithCR() (*operational.CatalogService, *mocks.MockCatalogRepo, *mocks.MockCatalogVersionRepo, *mocks.MockEntityInstanceRepo, *mockCatalogCRManager) {
	catRepo := new(mocks.MockCatalogRepo)
	cvRepo := new(mocks.MockCatalogVersionRepo)
	instRepo := new(mocks.MockEntityInstanceRepo)
	crMgr := &mockCatalogCRManager{}
	svc := operational.NewCatalogService(catRepo, cvRepo, instRepo, crMgr, "assethub")
	return svc, catRepo, cvRepo, instRepo, crMgr
}

func setupCatalogServiceWithCopy() (*operational.CatalogService, *mocks.MockCatalogRepo, *mocks.MockCatalogVersionRepo, *mocks.MockEntityInstanceRepo, *mocks.MockInstanceAttributeValueRepo, *mocks.MockAssociationLinkRepo) {
	catRepo := new(mocks.MockCatalogRepo)
	cvRepo := new(mocks.MockCatalogVersionRepo)
	instRepo := new(mocks.MockEntityInstanceRepo)
	iavRepo := new(mocks.MockInstanceAttributeValueRepo)
	linkRepo := new(mocks.MockAssociationLinkRepo)
	txm := &mocks.MockTransactionManager{}
	svc := operational.NewCatalogService(catRepo, cvRepo, instRepo, nil, "", operational.WithCopyDeps(iavRepo, linkRepo), operational.WithTransactionManager(txm))
	return svc, catRepo, cvRepo, instRepo, iavRepo, linkRepo
}

func setupCatalogServiceWithCopyAndCR() (*operational.CatalogService, *mocks.MockCatalogRepo, *mocks.MockCatalogVersionRepo, *mocks.MockEntityInstanceRepo, *mocks.MockInstanceAttributeValueRepo, *mocks.MockAssociationLinkRepo, *mockCatalogCRManager) {
	catRepo := new(mocks.MockCatalogRepo)
	cvRepo := new(mocks.MockCatalogVersionRepo)
	instRepo := new(mocks.MockEntityInstanceRepo)
	iavRepo := new(mocks.MockInstanceAttributeValueRepo)
	linkRepo := new(mocks.MockAssociationLinkRepo)
	crMgr := &mockCatalogCRManager{}
	txm := &mocks.MockTransactionManager{}
	svc := operational.NewCatalogService(catRepo, cvRepo, instRepo, crMgr, "assethub", operational.WithCopyDeps(iavRepo, linkRepo), operational.WithTransactionManager(txm))
	return svc, catRepo, cvRepo, instRepo, iavRepo, linkRepo, crMgr
}

// T-10.11: CreateCatalog with valid DNS-label name
func TestT10_11_CreateValidName(t *testing.T) {
	svc, catRepo, cvRepo, _ := setupCatalogService()
	ctx := context.Background()

	cvRepo.On("GetByID", ctx, "cv1").Return(&models.CatalogVersion{ID: "cv1", VersionLabel: "v1"}, nil)
	catRepo.On("Create", ctx, mock.AnythingOfType("*models.Catalog")).Return(nil)

	cat, err := svc.CreateCatalog(ctx, "my-catalog-1", "desc", "cv1")
	require.NoError(t, err)
	assert.Equal(t, "my-catalog-1", cat.Name)
	assert.Equal(t, "desc", cat.Description)
	assert.Equal(t, "cv1", cat.CatalogVersionID)
	assert.Equal(t, models.ValidationStatusDraft, cat.ValidationStatus)
	assert.NotEmpty(t, cat.ID)
}

// T-10.12: CreateCatalog with invalid name (uppercase)
func TestT10_12_InvalidNameUppercase(t *testing.T) {
	svc, _, _, _ := setupCatalogService()
	_, err := svc.CreateCatalog(context.Background(), "My-Catalog", "", "cv1")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

// T-10.13: CreateCatalog with invalid name (special chars)
func TestT10_13_InvalidNameSpecialChars(t *testing.T) {
	svc, _, _, _ := setupCatalogService()
	_, err := svc.CreateCatalog(context.Background(), "my_catalog!", "", "cv1")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

// T-10.14: CreateCatalog with invalid name (empty)
func TestT10_14_InvalidNameEmpty(t *testing.T) {
	svc, _, _, _ := setupCatalogService()
	_, err := svc.CreateCatalog(context.Background(), "", "", "cv1")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

// T-10.15: CreateCatalog with invalid name (>63 chars)
func TestT10_15_InvalidNameTooLong(t *testing.T) {
	svc, _, _, _ := setupCatalogService()
	longName := "a"
	for len(longName) <= 63 {
		longName += "a"
	}
	_, err := svc.CreateCatalog(context.Background(), longName, "", "cv1")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

// T-10.16: CreateCatalog with invalid name (starts with hyphen)
func TestT10_16_InvalidNameStartsWithHyphen(t *testing.T) {
	svc, _, _, _ := setupCatalogService()
	_, err := svc.CreateCatalog(context.Background(), "-my-catalog", "", "cv1")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

// T-10.17: CreateCatalog with invalid name (ends with hyphen)
func TestT10_17_InvalidNameEndsWithHyphen(t *testing.T) {
	svc, _, _, _ := setupCatalogService()
	_, err := svc.CreateCatalog(context.Background(), "my-catalog-", "", "cv1")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

// T-10.18: CreateCatalog with duplicate name
func TestT10_18_DuplicateName(t *testing.T) {
	svc, catRepo, cvRepo, _ := setupCatalogService()
	ctx := context.Background()

	cvRepo.On("GetByID", ctx, "cv1").Return(&models.CatalogVersion{ID: "cv1", VersionLabel: "v1"}, nil)
	catRepo.On("Create", ctx, mock.AnythingOfType("*models.Catalog")).Return(domainerrors.NewConflict("Catalog", "name already exists"))

	_, err := svc.CreateCatalog(ctx, "existing-catalog", "", "cv1")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsConflict(err))
}

// M2: CreateCatalog with empty catalog_version_id
func TestT10_M2_EmptyCVID(t *testing.T) {
	svc, _, _, _ := setupCatalogService()
	_, err := svc.CreateCatalog(context.Background(), "valid-name", "", "")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

// T-10.19: CreateCatalog with nonexistent CV ID
func TestT10_19_NonexistentCV(t *testing.T) {
	svc, _, cvRepo, _ := setupCatalogService()
	ctx := context.Background()

	cvRepo.On("GetByID", ctx, "bad-cv").Return(nil, domainerrors.NewNotFound("CatalogVersion", "bad-cv"))

	_, err := svc.CreateCatalog(ctx, "my-catalog", "", "bad-cv")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
}

// T-10.20: GetByName returns catalog with resolved CV label
func TestT10_20_GetByNameWithCVLabel(t *testing.T) {
	svc, catRepo, cvRepo, _ := setupCatalogService()
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "c1", Name: "my-catalog", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusDraft,
	}, nil)
	cvRepo.On("GetByID", ctx, "cv1").Return(&models.CatalogVersion{ID: "cv1", VersionLabel: "release-1.0"}, nil)

	detail, err := svc.GetByName(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, "my-catalog", detail.Name)
	assert.Equal(t, "release-1.0", detail.CatalogVersionLabel)
}

// T-10.21: GetByName for nonexistent name
func TestT10_21_GetByNameNotFound(t *testing.T) {
	svc, catRepo, _, _ := setupCatalogService()
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "nonexistent").Return(nil, domainerrors.NewNotFound("Catalog", "nonexistent"))

	_, err := svc.GetByName(ctx, "nonexistent")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
}

// T-10.22: List catalogs with no filters
func TestT10_22_ListNoFilters(t *testing.T) {
	svc, catRepo, cvRepo, _ := setupCatalogService()
	ctx := context.Background()

	params := models.ListParams{Limit: 20}
	catRepo.On("List", ctx, params).Return([]*models.Catalog{
		{ID: "c1", Name: "cat-a", CatalogVersionID: "cv1"},
		{ID: "c2", Name: "cat-b", CatalogVersionID: "cv1"},
	}, 2, nil)
	cvRepo.On("GetByID", ctx, "cv1").Return(&models.CatalogVersion{ID: "cv1", VersionLabel: "release-1"}, nil)

	details, total, err := svc.List(ctx, params)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, details, 2)
	assert.Equal(t, "release-1", details[0].CatalogVersionLabel)
	assert.Equal(t, "release-1", details[1].CatalogVersionLabel)
}

// T-10.23: List catalogs filtered by catalog_version_id
func TestT10_23_ListFilterByCVID(t *testing.T) {
	svc, catRepo, cvRepo, _ := setupCatalogService()
	ctx := context.Background()

	params := models.ListParams{Limit: 20, Filters: map[string]string{"catalog_version_id": "cv1"}}
	catRepo.On("List", ctx, params).Return([]*models.Catalog{
		{ID: "c1", Name: "cat-a", CatalogVersionID: "cv1"},
	}, 1, nil)
	cvRepo.On("GetByID", ctx, "cv1").Return(&models.CatalogVersion{ID: "cv1", VersionLabel: "v1"}, nil)

	details, total, err := svc.List(ctx, params)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, details, 1)
}

// T-10.24: List catalogs filtered by validation_status
func TestT10_24_ListFilterByStatus(t *testing.T) {
	svc, catRepo, cvRepo, _ := setupCatalogService()
	ctx := context.Background()

	params := models.ListParams{Limit: 20, Filters: map[string]string{"validation_status": "draft"}}
	catRepo.On("List", ctx, params).Return([]*models.Catalog{
		{ID: "c1", Name: "cat-a", CatalogVersionID: "cv1", ValidationStatus: models.ValidationStatusDraft},
	}, 1, nil)
	cvRepo.On("GetByID", ctx, "cv1").Return(&models.CatalogVersion{ID: "cv1", VersionLabel: "v1"}, nil)

	details, total, err := svc.List(ctx, params)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, details, 1)
}

// T-10.25: Delete catalog cascades to entity instances
func TestT10_25_DeleteCascade(t *testing.T) {
	svc, catRepo, _, instRepo := setupCatalogService()
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{ID: "c1", Name: "my-catalog"}, nil)
	instRepo.On("DeleteByCatalogID", ctx, "c1").Return(nil)
	catRepo.On("Delete", ctx, "c1").Return(nil)

	err := svc.Delete(ctx, "my-catalog")
	require.NoError(t, err)

	instRepo.AssertCalled(t, "DeleteByCatalogID", ctx, "c1")
	catRepo.AssertCalled(t, "Delete", ctx, "c1")
}

// T-10.26: Delete nonexistent catalog
func TestT10_26_DeleteNotFound(t *testing.T) {
	svc, catRepo, _, _ := setupCatalogService()
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "nonexistent").Return(nil, domainerrors.NewNotFound("Catalog", "nonexistent"))

	err := svc.Delete(ctx, "nonexistent")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
}

// --- Error propagation tests ---

// GetByName: CV lookup fails
func TestCatalogService_GetByName_CVLookupError(t *testing.T) {
	svc, catRepo, cvRepo, _ := setupCatalogService()
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "c1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	cvRepo.On("GetByID", ctx, "cv1").Return(nil, domainerrors.NewNotFound("CatalogVersion", "cv1"))

	_, err := svc.GetByName(ctx, "my-catalog")
	assert.Error(t, err)
}

// List: repo error propagates
func TestCatalogService_List_RepoError(t *testing.T) {
	svc, catRepo, _, _ := setupCatalogService()
	ctx := context.Background()

	params := models.ListParams{Limit: 20}
	catRepo.On("List", ctx, params).Return(nil, 0, domainerrors.NewNotFound("Catalog", "error"))

	_, _, err := svc.List(ctx, params)
	assert.Error(t, err)
}

// List: CV lookup error returns empty label (graceful degradation)
func TestCatalogService_List_CVLookupError(t *testing.T) {
	svc, catRepo, cvRepo, _ := setupCatalogService()
	ctx := context.Background()

	params := models.ListParams{Limit: 20}
	catRepo.On("List", ctx, params).Return([]*models.Catalog{
		{ID: "c1", Name: "cat-a", CatalogVersionID: "bad-cv"},
	}, 1, nil)
	cvRepo.On("GetByID", ctx, "bad-cv").Return(nil, domainerrors.NewNotFound("CatalogVersion", "bad-cv"))

	details, total, err := svc.List(ctx, params)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "", details[0].CatalogVersionLabel) // graceful degradation
}

// Delete: DeleteByCatalogID error propagates
func TestCatalogService_Delete_InstanceDeleteError(t *testing.T) {
	svc, catRepo, _, instRepo := setupCatalogService()
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{ID: "c1", Name: "my-catalog"}, nil)
	instRepo.On("DeleteByCatalogID", ctx, "c1").Return(domainerrors.NewValidation("db error"))

	err := svc.Delete(ctx, "my-catalog")
	assert.Error(t, err)
}

// Delete: catalog delete error propagates
func TestCatalogService_Delete_CatalogDeleteError(t *testing.T) {
	svc, catRepo, _, instRepo := setupCatalogService()
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{ID: "c1", Name: "my-catalog"}, nil)
	instRepo.On("DeleteByCatalogID", ctx, "c1").Return(nil)
	catRepo.On("Delete", ctx, "c1").Return(domainerrors.NewValidation("db error"))

	err := svc.Delete(ctx, "my-catalog")
	assert.Error(t, err)
}

// Delete: with txManager exercises transactional path
func TestCatalogService_Delete_WithTxManager(t *testing.T) {
	svc, catRepo, _, instRepo, _, _ := setupCatalogServiceWithCopy()
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{ID: "c1", Name: "my-catalog"}, nil)
	instRepo.On("DeleteByCatalogID", ctx, "c1").Return(nil)
	catRepo.On("Delete", ctx, "c1").Return(nil)

	err := svc.Delete(ctx, "my-catalog")
	require.NoError(t, err)
}

// SyncCR: crManager.CreateOrUpdate error logs warning (line 223)
func TestSyncCR_CreateOrUpdateError(t *testing.T) {
	svc, catRepo, cvRepo, _, _, _, crMgr := setupCatalogServiceWithCopyAndCR()
	ctx := context.Background()
	crMgr.createErr = fmt.Errorf("cr sync error")

	catRepo.On("GetByName", ctx, "pub-cat").Return(&models.Catalog{
		ID: "c1", Name: "pub-cat", CatalogVersionID: "cv1", Published: true,
	}, nil)
	cvRepo.On("GetByID", ctx, "cv1").Return(&models.CatalogVersion{ID: "cv1", VersionLabel: "v1"}, nil)

	// SyncCR should not return error — it logs and continues
	svc.SyncCR(ctx, "pub-cat")
	// Verify CreateOrUpdate was called (and failed)
	assert.NotNil(t, crMgr.createOrUpdateSpec)
}

// ReplaceCatalog: Step 1 UpdateName error (line 468)
func TestReplaceCatalog_Step1Error(t *testing.T) {
	svc, catRepo, _, _, _, _ := setupCatalogServiceWithCopy()
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "staging").Return(&models.Catalog{
		ID: "src-id", Name: "staging", ValidationStatus: models.ValidationStatusValid,
	}, nil)
	catRepo.On("GetByName", ctx, "prod").Return(&models.Catalog{
		ID: "tgt-id", Name: "prod",
	}, nil)
	catRepo.On("UpdateName", ctx, "tgt-id", "prod-archive").Return(fmt.Errorf("rename error"))

	_, err := svc.ReplaceCatalog(ctx, "staging", "prod", "prod-archive")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rename error")
}

// ReplaceCatalog: archive UpdatePublished error (line 483)
func TestReplaceCatalog_ArchiveUnpublishError(t *testing.T) {
	svc, catRepo, _, _, _, _ := setupCatalogServiceWithCopy()
	ctx := context.Background()

	pubTime := time.Now()
	catRepo.On("GetByName", ctx, "staging").Return(&models.Catalog{
		ID: "src-id", Name: "staging", ValidationStatus: models.ValidationStatusValid,
	}, nil)
	catRepo.On("GetByName", ctx, "prod").Return(&models.Catalog{
		ID: "tgt-id", Name: "prod", Published: true, PublishedAt: &pubTime,
	}, nil)
	catRepo.On("UpdateName", ctx, "tgt-id", "prod-archive").Return(nil)
	catRepo.On("UpdateName", ctx, "src-id", "prod").Return(nil)
	catRepo.On("UpdatePublished", ctx, "src-id", true, mock.AnythingOfType("*time.Time")).Return(nil)
	catRepo.On("UpdatePublished", ctx, "tgt-id", false, (*time.Time)(nil)).Return(fmt.Errorf("unpublish error"))

	_, err := svc.ReplaceCatalog(ctx, "staging", "prod", "prod-archive")
	assert.Error(t, err)
}

// ReplaceCatalog: nil txManager error fallback (line 500)
func TestReplaceCatalog_NilTxManager_Error(t *testing.T) {
	catRepo := new(mocks.MockCatalogRepo)
	cvRepo := new(mocks.MockCatalogVersionRepo)
	instRepo := new(mocks.MockEntityInstanceRepo)
	iavRepo := new(mocks.MockInstanceAttributeValueRepo)
	linkRepo := new(mocks.MockAssociationLinkRepo)
	svc := operational.NewCatalogService(catRepo, cvRepo, instRepo, nil, "", operational.WithCopyDeps(iavRepo, linkRepo))
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "staging").Return(&models.Catalog{
		ID: "src-id", Name: "staging", ValidationStatus: models.ValidationStatusValid,
	}, nil)
	catRepo.On("GetByName", ctx, "prod").Return(&models.Catalog{
		ID: "tgt-id", Name: "prod",
	}, nil)
	catRepo.On("UpdateName", ctx, "tgt-id", "prod-archive").Return(nil)
	catRepo.On("UpdateName", ctx, "src-id", "prod").Return(fmt.Errorf("step2 error"))

	_, err := svc.ReplaceCatalog(ctx, "staging", "prod", "prod-archive")
	assert.Error(t, err)
}

// Handler: ListCatalogs error propagates
func TestCatalogHandler_List_ServiceError(t *testing.T) {
	// This is tested via the handler test file but adding explicit service error coverage
	svc, catRepo, _, _ := setupCatalogService()
	ctx := context.Background()

	params := models.ListParams{Limit: 20}
	catRepo.On("List", ctx, params).Return(nil, 0, domainerrors.NewValidation("internal"))

	_, _, err := svc.List(ctx, params)
	assert.Error(t, err)
}

// === Publish/Unpublish Tests ===

// T-16.03: Publish a valid catalog
func TestT16_03_PublishValid(t *testing.T) {
	svc, catRepo, cvRepo, _ := setupCatalogService()
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "c1", Name: "my-catalog", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusValid,
	}, nil)
	catRepo.On("UpdatePublished", ctx, "c1", true, mock.AnythingOfType("*time.Time")).Return(nil)
	cvRepo.On("GetByID", ctx, "cv1").Return(&models.CatalogVersion{ID: "cv1", VersionLabel: "v1.0"}, nil)

	err := svc.Publish(ctx, "my-catalog")
	require.NoError(t, err)
	catRepo.AssertCalled(t, "UpdatePublished", ctx, "c1", true, mock.AnythingOfType("*time.Time"))
}

// T-16.04: Publish a draft catalog → error
func TestT16_04_PublishDraft(t *testing.T) {
	svc, catRepo, _, _ := setupCatalogService()
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "c1", Name: "my-catalog", ValidationStatus: models.ValidationStatusDraft,
	}, nil)

	err := svc.Publish(ctx, "my-catalog")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

// T-16.05: Publish an invalid catalog → error
func TestT16_05_PublishInvalid(t *testing.T) {
	svc, catRepo, _, _ := setupCatalogService()
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "c1", Name: "my-catalog", ValidationStatus: models.ValidationStatusInvalid,
	}, nil)

	err := svc.Publish(ctx, "my-catalog")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

// T-16.06: Publish nonexistent catalog → NotFound
func TestT16_06_PublishNotFound(t *testing.T) {
	svc, catRepo, _, _ := setupCatalogService()
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "nonexistent").Return(nil, domainerrors.NewNotFound("Catalog", "nonexistent"))

	err := svc.Publish(ctx, "nonexistent")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
}

// T-16.08: Publish with nil crManager (DB-only)
func TestT16_08_PublishNilCRManager(t *testing.T) {
	svc, catRepo, cvRepo, _ := setupCatalogService()
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "c1", Name: "my-catalog", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusValid,
	}, nil)
	catRepo.On("UpdatePublished", ctx, "c1", true, mock.AnythingOfType("*time.Time")).Return(nil)
	cvRepo.On("GetByID", ctx, "cv1").Return(&models.CatalogVersion{ID: "cv1", VersionLabel: "v1.0"}, nil)

	err := svc.Publish(ctx, "my-catalog")
	require.NoError(t, err)
}

// T-16.11: Unpublish sets published=false
func TestT16_11_Unpublish(t *testing.T) {
	svc, catRepo, _, _ := setupCatalogService()
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "c1", Name: "my-catalog", Published: true,
	}, nil)
	catRepo.On("UpdatePublished", ctx, "c1", false, (*time.Time)(nil)).Return(nil)

	err := svc.Unpublish(ctx, "my-catalog")
	require.NoError(t, err)
	catRepo.AssertCalled(t, "UpdatePublished", ctx, "c1", false, (*time.Time)(nil))
}

// T-16.14: Unpublish already-unpublished → idempotent
func TestT16_14_UnpublishIdempotent(t *testing.T) {
	svc, catRepo, _, _ := setupCatalogService()
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "c1", Name: "my-catalog", Published: false,
	}, nil)
	catRepo.On("UpdatePublished", ctx, "c1", false, (*time.Time)(nil)).Return(nil)

	err := svc.Unpublish(ctx, "my-catalog")
	require.NoError(t, err)
}

// T-16.09: Publish already-published is idempotent (re-publishes)
func TestT16_09_PublishAlreadyPublished(t *testing.T) {
	svc, catRepo, cvRepo, _ := setupCatalogService()
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "c1", Name: "my-catalog", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusValid, Published: true,
	}, nil)
	catRepo.On("UpdatePublished", ctx, "c1", true, mock.AnythingOfType("*time.Time")).Return(nil)
	cvRepo.On("GetByID", ctx, "cv1").Return(&models.CatalogVersion{ID: "cv1", VersionLabel: "v1.0"}, nil)

	err := svc.Publish(ctx, "my-catalog")
	require.NoError(t, err)
}

// Publish: UpdatePublished error propagated
func TestPublish_UpdatePublishedError(t *testing.T) {
	svc, catRepo, _, _ := setupCatalogService()
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "c1", Name: "my-catalog", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusValid,
	}, nil)
	catRepo.On("UpdatePublished", ctx, "c1", true, mock.AnythingOfType("*time.Time")).Return(domainerrors.NewValidation("db"))

	err := svc.Publish(ctx, "my-catalog")
	assert.Error(t, err)
}

// Unpublish: UpdatePublished error propagated
func TestUnpublish_UpdatePublishedError(t *testing.T) {
	svc, catRepo, _, _ := setupCatalogService()
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "c1", Name: "my-catalog", Published: true,
	}, nil)
	catRepo.On("UpdatePublished", ctx, "c1", false, (*time.Time)(nil)).Return(domainerrors.NewValidation("db"))

	err := svc.Unpublish(ctx, "my-catalog")
	assert.Error(t, err)
}

// Delete: published catalog cleans up CR
func TestDelete_PublishedCatalogCleansUpCR(t *testing.T) {
	svc, catRepo, _, instRepo := setupCatalogService()
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "c1", Name: "my-catalog", Published: true,
	}, nil)
	// crManager is nil (from setupCatalogService), so CR cleanup is skipped
	instRepo.On("DeleteByCatalogID", ctx, "c1").Return(nil)
	catRepo.On("Delete", ctx, "c1").Return(nil)

	err := svc.Delete(ctx, "my-catalog")
	require.NoError(t, err)
}

// IsPublished tests
func TestIsPublished_True(t *testing.T) {
	svc, catRepo, _, _ := setupCatalogService()

	catRepo.On("GetByName", mock.Anything, "my-catalog").Return(&models.Catalog{
		ID: "c1", Name: "my-catalog", Published: true,
	}, nil)

	e := echo.New()
	req, _ := http.NewRequest("GET", "/", nil)
	c := e.NewContext(req, nil)

	published, err := svc.IsPublished(c, "my-catalog")
	require.NoError(t, err)
	assert.True(t, published)
}

func TestIsPublished_False(t *testing.T) {
	svc, catRepo, _, _ := setupCatalogService()

	catRepo.On("GetByName", mock.Anything, "my-catalog").Return(&models.Catalog{
		ID: "c1", Name: "my-catalog", Published: false,
	}, nil)

	e := echo.New()
	req, _ := http.NewRequest("GET", "/", nil)
	c := e.NewContext(req, nil)

	published, err := svc.IsPublished(c, "my-catalog")
	require.NoError(t, err)
	assert.False(t, published)
}

// === SyncCR Tests ===

// SyncCR updates CR when catalog is published
func TestSyncCR_PublishedCatalogUpdatesCR(t *testing.T) {
	svc, catRepo, cvRepo, _, crMgr := setupCatalogServiceWithCR()
	ctx := context.Background()

	catRepo.On("GetByName", mock.Anything, "my-catalog").Return(&models.Catalog{
		ID: "c1", Name: "my-catalog", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusDraft, Published: true,
	}, nil)
	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{ID: "cv1", VersionLabel: "v1.0"}, nil)

	svc.SyncCR(ctx, "my-catalog")

	require.NotNil(t, crMgr.createOrUpdateSpec)
	assert.Equal(t, "my-catalog", crMgr.createOrUpdateSpec.Name)
	assert.Equal(t, "draft", crMgr.createOrUpdateSpec.ValidationStatus)
	assert.Equal(t, "v1.0", crMgr.createOrUpdateSpec.CatalogVersionLabel)
}

// SyncCR does nothing when catalog is not published
func TestSyncCR_UnpublishedCatalogNoOp(t *testing.T) {
	svc, catRepo, _, _, crMgr := setupCatalogServiceWithCR()
	ctx := context.Background()

	catRepo.On("GetByName", mock.Anything, "my-catalog").Return(&models.Catalog{
		ID: "c1", Name: "my-catalog", Published: false,
	}, nil)

	svc.SyncCR(ctx, "my-catalog")

	assert.Nil(t, crMgr.createOrUpdateSpec, "should not call CreateOrUpdate for unpublished catalog")
}

// SyncCR does nothing when crManager is nil
func TestSyncCR_NilCRManager(t *testing.T) {
	svc, _, _, _ := setupCatalogService() // nil crManager
	ctx := context.Background()

	// Should not panic
	svc.SyncCR(ctx, "my-catalog")
}

// SyncCR does nothing when catalog not found
func TestSyncCR_CatalogNotFound(t *testing.T) {
	svc, catRepo, _, _, crMgr := setupCatalogServiceWithCR()
	ctx := context.Background()

	catRepo.On("GetByName", mock.Anything, "nonexistent").Return(nil, domainerrors.NewNotFound("Catalog", "nonexistent"))

	svc.SyncCR(ctx, "nonexistent")

	assert.Nil(t, crMgr.createOrUpdateSpec)
}

// T-16.07: Publish calls CRManager.CreateOrUpdate with correct spec
func TestT16_07_PublishCallsCRManager(t *testing.T) {
	svc, catRepo, cvRepo, _, crMgr := setupCatalogServiceWithCR()
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "c1", Name: "my-catalog", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusValid,
	}, nil)
	catRepo.On("UpdatePublished", ctx, "c1", true, mock.AnythingOfType("*time.Time")).Return(nil)
	cvRepo.On("GetByID", ctx, "cv1").Return(&models.CatalogVersion{ID: "cv1", VersionLabel: "v1.0"}, nil)

	err := svc.Publish(ctx, "my-catalog")
	require.NoError(t, err)
	require.NotNil(t, crMgr.createOrUpdateSpec)
	assert.Equal(t, "my-catalog", crMgr.createOrUpdateSpec.Name)
	assert.Equal(t, "assethub", crMgr.createOrUpdateSpec.Namespace)
	assert.Equal(t, "v1.0", crMgr.createOrUpdateSpec.CatalogVersionLabel)
	assert.Equal(t, "valid", crMgr.createOrUpdateSpec.ValidationStatus)
}

// Publish: CR creation fails → rollback DB
func TestPublish_CRFailureRollsBack(t *testing.T) {
	svc, catRepo, cvRepo, _, crMgr := setupCatalogServiceWithCR()
	ctx := context.Background()
	crMgr.createErr = domainerrors.NewValidation("k8s error")

	catRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "c1", Name: "my-catalog", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusValid,
	}, nil)
	catRepo.On("UpdatePublished", ctx, "c1", true, mock.AnythingOfType("*time.Time")).Return(nil)
	cvRepo.On("GetByID", ctx, "cv1").Return(&models.CatalogVersion{ID: "cv1", VersionLabel: "v1.0"}, nil)
	catRepo.On("UpdatePublished", ctx, "c1", false, (*time.Time)(nil)).Return(nil) // rollback

	err := svc.Publish(ctx, "my-catalog")
	assert.Error(t, err)
	// Verify rollback was called
	catRepo.AssertCalled(t, "UpdatePublished", ctx, "c1", false, (*time.Time)(nil))
}

// T-16.12: Unpublish calls CRManager.Delete
func TestT16_12_UnpublishCallsCRManager(t *testing.T) {
	svc, catRepo, _, _, crMgr := setupCatalogServiceWithCR()
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "c1", Name: "my-catalog", Published: true,
	}, nil)
	catRepo.On("UpdatePublished", ctx, "c1", false, (*time.Time)(nil)).Return(nil)

	err := svc.Unpublish(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, "my-catalog", crMgr.deleteCalledWith)
}

// Unpublish: CR deletion fails → error propagated
func TestUnpublish_CRDeleteError(t *testing.T) {
	svc, catRepo, _, _, crMgr := setupCatalogServiceWithCR()
	ctx := context.Background()
	crMgr.deleteErr = domainerrors.NewValidation("k8s error")

	catRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "c1", Name: "my-catalog", Published: true,
	}, nil)
	catRepo.On("UpdatePublished", ctx, "c1", false, (*time.Time)(nil)).Return(nil)

	err := svc.Unpublish(ctx, "my-catalog")
	assert.Error(t, err)
}

// Delete: published catalog with crManager cleans up CR
func TestDelete_PublishedWithCRManager(t *testing.T) {
	svc, catRepo, _, instRepo, crMgr := setupCatalogServiceWithCR()
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "c1", Name: "my-catalog", Published: true,
	}, nil)
	instRepo.On("DeleteByCatalogID", ctx, "c1").Return(nil)
	catRepo.On("Delete", ctx, "c1").Return(nil)

	err := svc.Delete(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Equal(t, "my-catalog", crMgr.deleteCalledWith)
}

// Delete: CR cleanup fails → error propagated
func TestDelete_CRCleanupError(t *testing.T) {
	svc, catRepo, _, _, crMgr := setupCatalogServiceWithCR()
	ctx := context.Background()
	crMgr.deleteErr = domainerrors.NewValidation("k8s error")

	catRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "c1", Name: "my-catalog", Published: true,
	}, nil)

	err := svc.Delete(ctx, "my-catalog")
	assert.Error(t, err)
}

func TestIsPublished_NotFound(t *testing.T) {
	svc, catRepo, _, _ := setupCatalogService()

	catRepo.On("GetByName", mock.Anything, "nonexistent").Return(nil, domainerrors.NewNotFound("Catalog", "nonexistent"))

	e := echo.New()
	req, _ := http.NewRequest("GET", "/", nil)
	c := e.NewContext(req, nil)

	_, err := svc.IsPublished(c, "nonexistent")
	assert.Error(t, err)
}

// T-16.15: Unpublish nonexistent → NotFound
func TestT16_15_UnpublishNotFound(t *testing.T) {
	svc, catRepo, _, _ := setupCatalogService()
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "nonexistent").Return(nil, domainerrors.NewNotFound("Catalog", "nonexistent"))

	err := svc.Unpublish(ctx, "nonexistent")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
}

// ---- Copy Catalog Tests ----

// T-17.05: Copy creates new catalog with same CV pin and draft status
func TestT17_05_CopyCatalog_BasicFields(t *testing.T) {
	svc, catRepo, _, instRepo, iavRepo, linkRepo := setupCatalogServiceWithCopy()
	ctx := context.Background()

	sourceCat := &models.Catalog{
		ID: "src-id", Name: "source", Description: "source desc",
		CatalogVersionID: "cv1", ValidationStatus: models.ValidationStatusValid,
	}
	catRepo.On("GetByName", ctx, "source").Return(sourceCat, nil)
	catRepo.On("Create", ctx, mock.AnythingOfType("*models.Catalog")).Return(nil)
	instRepo.On("ListByCatalog", ctx, "src-id").Return([]*models.EntityInstance{}, nil)
	// No instances → no attribute/link calls needed
	_ = iavRepo
	_ = linkRepo

	result, err := svc.CopyCatalog(ctx, "source", "target", "")
	require.NoError(t, err)
	assert.Equal(t, "target", result.Name)
	assert.Equal(t, "cv1", result.CatalogVersionID)
	assert.Equal(t, models.ValidationStatusDraft, result.ValidationStatus)
	assert.Equal(t, "source desc", result.Description)
	assert.NotEqual(t, "src-id", result.ID) // New ID
}

// T-17.06: Copy uses provided description
func TestT17_06_CopyCatalog_CustomDescription(t *testing.T) {
	svc, catRepo, _, instRepo, _, _ := setupCatalogServiceWithCopy()
	ctx := context.Background()

	sourceCat := &models.Catalog{
		ID: "src-id", Name: "source", Description: "old desc",
		CatalogVersionID: "cv1", ValidationStatus: models.ValidationStatusValid,
	}
	catRepo.On("GetByName", ctx, "source").Return(sourceCat, nil)
	catRepo.On("Create", ctx, mock.AnythingOfType("*models.Catalog")).Return(nil)
	instRepo.On("ListByCatalog", ctx, "src-id").Return([]*models.EntityInstance{}, nil)

	result, err := svc.CopyCatalog(ctx, "source", "target", "custom desc")
	require.NoError(t, err)
	assert.Equal(t, "custom desc", result.Description)
}

// T-17.07/08/09/10/11/12: Copy clones instances with remapped IDs, attrs, links, hierarchy
func TestT17_07_CopyCatalog_ClonesInstances(t *testing.T) {
	svc, catRepo, _, instRepo, iavRepo, linkRepo := setupCatalogServiceWithCopy()
	ctx := context.Background()

	sourceCat := &models.Catalog{
		ID: "src-id", Name: "source", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusValid,
	}
	catRepo.On("GetByName", ctx, "source").Return(sourceCat, nil)
	catRepo.On("Create", ctx, mock.AnythingOfType("*models.Catalog")).Return(nil)

	// Source has a parent instance and a child
	parentInst := &models.EntityInstance{
		ID: "parent-1", EntityTypeID: "et-1", CatalogID: "src-id",
		Name: "parent", Description: "parent desc", Version: 3,
	}
	childInst := &models.EntityInstance{
		ID: "child-1", EntityTypeID: "et-2", CatalogID: "src-id",
		ParentInstanceID: "parent-1", Name: "child", Description: "child desc", Version: 5,
	}
	instRepo.On("ListByCatalog", ctx, "src-id").Return([]*models.EntityInstance{parentInst, childInst}, nil)

	// Attribute values for parent and child
	parentAttrs := []*models.InstanceAttributeValue{
		{ID: "av1", InstanceID: "parent-1", InstanceVersion: 3, AttributeID: "attr1", ValueString: "hello"},
	}
	childAttrs := []*models.InstanceAttributeValue{
		{ID: "av2", InstanceID: "child-1", InstanceVersion: 5, AttributeID: "attr2", ValueString: "world"},
	}
	iavRepo.On("GetCurrentValues", ctx, "parent-1").Return(parentAttrs, nil)
	iavRepo.On("GetCurrentValues", ctx, "child-1").Return(childAttrs, nil)

	// Association links — parent references child
	links := []*models.AssociationLink{
		{ID: "link-1", AssociationID: "assoc-1", SourceInstanceID: "parent-1", TargetInstanceID: "child-1"},
	}
	linkRepo.On("GetForwardRefs", ctx, "parent-1").Return(links, nil)
	linkRepo.On("GetForwardRefs", ctx, "child-1").Return([]*models.AssociationLink{}, nil)

	// Expect creates for new instances, attrs, links
	instRepo.On("Create", ctx, mock.AnythingOfType("*models.EntityInstance")).Return(nil)
	iavRepo.On("SetValues", ctx, mock.AnythingOfType("[]*models.InstanceAttributeValue")).Return(nil)
	linkRepo.On("Create", ctx, mock.AnythingOfType("*models.AssociationLink")).Return(nil)

	result, err := svc.CopyCatalog(ctx, "source", "target", "")
	require.NoError(t, err)
	assert.Equal(t, "target", result.Name)

	// Verify instances were created with new IDs
	instCalls := instRepo.Calls
	var createdInstances []*models.EntityInstance
	for _, call := range instCalls {
		if call.Method == "Create" {
			inst := call.Arguments[1].(*models.EntityInstance)
			createdInstances = append(createdInstances, inst)
		}
	}
	require.Len(t, createdInstances, 2)

	// Both should have new IDs, version=1
	for _, inst := range createdInstances {
		assert.NotEqual(t, "parent-1", inst.ID)
		assert.NotEqual(t, "child-1", inst.ID)
		assert.Equal(t, 1, inst.Version)
	}

	// Find the cloned child — it should have remapped parent ID
	var clonedChild *models.EntityInstance
	for _, inst := range createdInstances {
		if inst.Name == "child" {
			clonedChild = inst
			break
		}
	}
	require.NotNil(t, clonedChild)
	assert.NotEmpty(t, clonedChild.ParentInstanceID)
	assert.NotEqual(t, "parent-1", clonedChild.ParentInstanceID) // Remapped

	// The cloned child's parent should be the cloned parent's ID
	var clonedParent *models.EntityInstance
	for _, inst := range createdInstances {
		if inst.Name == "parent" {
			clonedParent = inst
			break
		}
	}
	require.NotNil(t, clonedParent)
	assert.Equal(t, clonedParent.ID, clonedChild.ParentInstanceID)

	// Verify attribute values were set with remapped instance IDs
	attrCalls := iavRepo.Calls
	var setValuesCalls []mock.Call
	for _, call := range attrCalls {
		if call.Method == "SetValues" {
			setValuesCalls = append(setValuesCalls, call)
		}
	}
	assert.Len(t, setValuesCalls, 2) // One per instance

	// Verify link was created with remapped IDs
	linkCalls := linkRepo.Calls
	var linkCreateCalls []mock.Call
	for _, call := range linkCalls {
		if call.Method == "Create" {
			linkCreateCalls = append(linkCreateCalls, call)
		}
	}
	require.Len(t, linkCreateCalls, 1)
	createdLink := linkCreateCalls[0].Arguments[1].(*models.AssociationLink)
	assert.NotEqual(t, "parent-1", createdLink.SourceInstanceID)
	assert.NotEqual(t, "child-1", createdLink.TargetInstanceID)
	assert.Equal(t, clonedParent.ID, createdLink.SourceInstanceID)
	assert.Equal(t, clonedChild.ID, createdLink.TargetInstanceID)
}

// T-17.13: Copy with nonexistent source returns NotFoundError
func TestT17_13_CopyCatalog_SourceNotFound(t *testing.T) {
	svc, catRepo, _, _, _, _ := setupCatalogServiceWithCopy()
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "nonexistent").Return(nil, domainerrors.NewNotFound("Catalog", "nonexistent"))

	_, err := svc.CopyCatalog(ctx, "nonexistent", "target", "")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
}

// T-17.14: Copy with invalid target name returns validation error
func TestT17_14_CopyCatalog_InvalidTargetName(t *testing.T) {
	svc, catRepo, _, _, _, _ := setupCatalogServiceWithCopy()
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "source").Return(&models.Catalog{
		ID: "src-id", Name: "source", CatalogVersionID: "cv1",
	}, nil)

	_, err := svc.CopyCatalog(ctx, "source", "INVALID_NAME", "")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

// T-17.15: Copy with duplicate target name returns ConflictError
func TestT17_15_CopyCatalog_DuplicateTarget(t *testing.T) {
	svc, catRepo, _, instRepo, _, _ := setupCatalogServiceWithCopy()
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "source").Return(&models.Catalog{
		ID: "src-id", Name: "source", CatalogVersionID: "cv1",
	}, nil)
	instRepo.On("ListByCatalog", ctx, "src-id").Return([]*models.EntityInstance{}, nil)
	catRepo.On("Create", ctx, mock.AnythingOfType("*models.Catalog")).Return(
		domainerrors.NewConflict("Catalog", "name already exists: target"),
	)

	_, err := svc.CopyCatalog(ctx, "source", "target", "")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsConflict(err))
}

// T-17.16: Copy of empty catalog creates empty catalog
func TestT17_16_CopyCatalog_EmptyCatalog(t *testing.T) {
	svc, catRepo, _, instRepo, _, _ := setupCatalogServiceWithCopy()
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "source").Return(&models.Catalog{
		ID: "src-id", Name: "source", CatalogVersionID: "cv1",
	}, nil)
	catRepo.On("Create", ctx, mock.AnythingOfType("*models.Catalog")).Return(nil)
	instRepo.On("ListByCatalog", ctx, "src-id").Return([]*models.EntityInstance{}, nil)

	result, err := svc.CopyCatalog(ctx, "source", "empty-copy", "")
	require.NoError(t, err)
	assert.Equal(t, "empty-copy", result.Name)
}

// T-17.19: Copy does not modify source catalog
func TestT17_19_CopyCatalog_SourceUnchanged(t *testing.T) {
	svc, catRepo, _, instRepo, _, _ := setupCatalogServiceWithCopy()
	ctx := context.Background()

	sourceCat := &models.Catalog{
		ID: "src-id", Name: "source", Description: "src desc",
		CatalogVersionID: "cv1", ValidationStatus: models.ValidationStatusValid,
		Published: true,
	}
	catRepo.On("GetByName", ctx, "source").Return(sourceCat, nil)
	catRepo.On("Create", ctx, mock.AnythingOfType("*models.Catalog")).Return(nil)
	instRepo.On("ListByCatalog", ctx, "src-id").Return([]*models.EntityInstance{}, nil)

	_, err := svc.CopyCatalog(ctx, "source", "target", "")
	require.NoError(t, err)

	// Source should not be modified
	assert.Equal(t, "source", sourceCat.Name)
	assert.Equal(t, models.ValidationStatusValid, sourceCat.ValidationStatus)
	assert.True(t, sourceCat.Published)

	// No UpdateName, UpdatePublished, or UpdateValidationStatus calls on source
	catRepo.AssertNotCalled(t, "UpdateName", mock.Anything, mock.Anything, mock.Anything)
	catRepo.AssertNotCalled(t, "UpdatePublished", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// T-17.18: Copy transactional — instance create error returns error, no partial data
func TestT17_18_CopyCatalog_InstanceCreateError(t *testing.T) {
	svc, catRepo, _, instRepo, _, _ := setupCatalogServiceWithCopy()
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "source").Return(&models.Catalog{
		ID: "src-id", Name: "source", CatalogVersionID: "cv1",
	}, nil)
	instRepo.On("ListByCatalog", ctx, "src-id").Return([]*models.EntityInstance{
		{ID: "i1", EntityTypeID: "et1", CatalogID: "src-id", Name: "inst", Version: 1},
	}, nil)
	catRepo.On("Create", ctx, mock.AnythingOfType("*models.Catalog")).Return(nil)
	instRepo.On("Create", ctx, mock.AnythingOfType("*models.EntityInstance")).Return(fmt.Errorf("db error"))

	_, err := svc.CopyCatalog(ctx, "source", "target", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

// CopyCatalog — GetCurrentValues error
func TestCopyCatalog_AttrError(t *testing.T) {
	svc, catRepo, _, instRepo, iavRepo, _ := setupCatalogServiceWithCopy()
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "source").Return(&models.Catalog{
		ID: "src-id", Name: "source", CatalogVersionID: "cv1",
	}, nil)
	instRepo.On("ListByCatalog", ctx, "src-id").Return([]*models.EntityInstance{
		{ID: "i1", EntityTypeID: "et1", CatalogID: "src-id", Name: "inst", Version: 1},
	}, nil)
	catRepo.On("Create", ctx, mock.AnythingOfType("*models.Catalog")).Return(nil)
	instRepo.On("Create", ctx, mock.AnythingOfType("*models.EntityInstance")).Return(nil)
	iavRepo.On("GetCurrentValues", ctx, "i1").Return(nil, fmt.Errorf("attr error"))

	_, err := svc.CopyCatalog(ctx, "source", "target", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "attr error")
}

// CopyCatalog — SetValues error
func TestCopyCatalog_SetValuesError(t *testing.T) {
	svc, catRepo, _, instRepo, iavRepo, _ := setupCatalogServiceWithCopy()
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "source").Return(&models.Catalog{
		ID: "src-id", Name: "source", CatalogVersionID: "cv1",
	}, nil)
	instRepo.On("ListByCatalog", ctx, "src-id").Return([]*models.EntityInstance{
		{ID: "i1", EntityTypeID: "et1", CatalogID: "src-id", Name: "inst", Version: 1},
	}, nil)
	catRepo.On("Create", ctx, mock.AnythingOfType("*models.Catalog")).Return(nil)
	instRepo.On("Create", ctx, mock.AnythingOfType("*models.EntityInstance")).Return(nil)
	iavRepo.On("GetCurrentValues", ctx, "i1").Return([]*models.InstanceAttributeValue{
		{ID: "av1", InstanceID: "i1", AttributeID: "a1", ValueString: "v"},
	}, nil)
	iavRepo.On("SetValues", ctx, mock.Anything).Return(fmt.Errorf("set error"))

	_, err := svc.CopyCatalog(ctx, "source", "target", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "set error")
}

// CopyCatalog — GetForwardRefs error
func TestCopyCatalog_LinkError(t *testing.T) {
	svc, catRepo, _, instRepo, iavRepo, linkRepo := setupCatalogServiceWithCopy()
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "source").Return(&models.Catalog{
		ID: "src-id", Name: "source", CatalogVersionID: "cv1",
	}, nil)
	instRepo.On("ListByCatalog", ctx, "src-id").Return([]*models.EntityInstance{
		{ID: "i1", EntityTypeID: "et1", CatalogID: "src-id", Name: "inst", Version: 1},
	}, nil)
	catRepo.On("Create", ctx, mock.AnythingOfType("*models.Catalog")).Return(nil)
	instRepo.On("Create", ctx, mock.AnythingOfType("*models.EntityInstance")).Return(nil)
	iavRepo.On("GetCurrentValues", ctx, "i1").Return([]*models.InstanceAttributeValue{}, nil)
	linkRepo.On("GetForwardRefs", ctx, "i1").Return(nil, fmt.Errorf("link error"))

	_, err := svc.CopyCatalog(ctx, "source", "target", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "link error")
}

// CopyCatalog — link Create error
func TestCopyCatalog_LinkCreateError(t *testing.T) {
	svc, catRepo, _, instRepo, iavRepo, linkRepo := setupCatalogServiceWithCopy()
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "source").Return(&models.Catalog{
		ID: "src-id", Name: "source", CatalogVersionID: "cv1",
	}, nil)
	instRepo.On("ListByCatalog", ctx, "src-id").Return([]*models.EntityInstance{
		{ID: "i1", EntityTypeID: "et1", CatalogID: "src-id", Name: "inst", Version: 1},
	}, nil)
	catRepo.On("Create", ctx, mock.AnythingOfType("*models.Catalog")).Return(nil)
	instRepo.On("Create", ctx, mock.AnythingOfType("*models.EntityInstance")).Return(nil)
	iavRepo.On("GetCurrentValues", ctx, "i1").Return([]*models.InstanceAttributeValue{}, nil)
	linkRepo.On("GetForwardRefs", ctx, "i1").Return([]*models.AssociationLink{
		{ID: "l1", AssociationID: "a1", SourceInstanceID: "i1", TargetInstanceID: "i1"},
	}, nil)
	linkRepo.On("Create", ctx, mock.AnythingOfType("*models.AssociationLink")).Return(fmt.Errorf("link create error"))

	_, err := svc.CopyCatalog(ctx, "source", "target", "")
	assert.Error(t, err)
}

// CopyCatalog — link with target outside catalog is skipped
func TestCopyCatalog_LinkOutsideCatalogSkipped(t *testing.T) {
	svc, catRepo, _, instRepo, iavRepo, linkRepo := setupCatalogServiceWithCopy()
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "source").Return(&models.Catalog{
		ID: "src-id", Name: "source", CatalogVersionID: "cv1",
	}, nil)
	instRepo.On("ListByCatalog", ctx, "src-id").Return([]*models.EntityInstance{
		{ID: "i1", EntityTypeID: "et1", CatalogID: "src-id", Name: "inst", Version: 1},
	}, nil)
	catRepo.On("Create", ctx, mock.AnythingOfType("*models.Catalog")).Return(nil)
	instRepo.On("Create", ctx, mock.AnythingOfType("*models.EntityInstance")).Return(nil)
	iavRepo.On("GetCurrentValues", ctx, "i1").Return([]*models.InstanceAttributeValue{}, nil)
	// Link points to instance outside catalog — should be skipped
	linkRepo.On("GetForwardRefs", ctx, "i1").Return([]*models.AssociationLink{
		{ID: "l1", AssociationID: "a1", SourceInstanceID: "i1", TargetInstanceID: "external-inst"},
	}, nil)

	result, err := svc.CopyCatalog(ctx, "source", "target", "")
	require.NoError(t, err)
	assert.Equal(t, "target", result.Name)
	// linkRepo.Create should NOT have been called
	linkRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

// CopyCatalog — without TransactionManager (nil fallback, success)
func TestCopyCatalog_NilTxManager(t *testing.T) {
	catRepo := new(mocks.MockCatalogRepo)
	cvRepo := new(mocks.MockCatalogVersionRepo)
	instRepo := new(mocks.MockEntityInstanceRepo)
	iavRepo := new(mocks.MockInstanceAttributeValueRepo)
	linkRepo := new(mocks.MockAssociationLinkRepo)
	svc := operational.NewCatalogService(catRepo, cvRepo, instRepo, nil, "", operational.WithCopyDeps(iavRepo, linkRepo))
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "source").Return(&models.Catalog{
		ID: "src-id", Name: "source", CatalogVersionID: "cv1",
	}, nil)
	catRepo.On("Create", ctx, mock.AnythingOfType("*models.Catalog")).Return(nil)
	instRepo.On("ListByCatalog", ctx, "src-id").Return([]*models.EntityInstance{}, nil)

	result, err := svc.CopyCatalog(ctx, "source", "target", "")
	require.NoError(t, err)
	assert.Equal(t, "target", result.Name)
}

// CopyCatalog — without TransactionManager (nil fallback, error)
func TestCopyCatalog_NilTxManager_Error(t *testing.T) {
	catRepo := new(mocks.MockCatalogRepo)
	cvRepo := new(mocks.MockCatalogVersionRepo)
	instRepo := new(mocks.MockEntityInstanceRepo)
	iavRepo := new(mocks.MockInstanceAttributeValueRepo)
	linkRepo := new(mocks.MockAssociationLinkRepo)
	svc := operational.NewCatalogService(catRepo, cvRepo, instRepo, nil, "", operational.WithCopyDeps(iavRepo, linkRepo))
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "source").Return(&models.Catalog{
		ID: "src-id", Name: "source", CatalogVersionID: "cv1",
	}, nil)
	instRepo.On("ListByCatalog", ctx, "src-id").Return([]*models.EntityInstance{}, nil)
	catRepo.On("Create", ctx, mock.AnythingOfType("*models.Catalog")).Return(fmt.Errorf("create error"))

	_, err := svc.CopyCatalog(ctx, "source", "target", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create error")
}

// ---- Replace Catalog Tests ----

// T-17.26/27: Replace renames target to archive, source to target name
func TestT17_26_ReplaceCatalog_BasicSwap(t *testing.T) {
	svc, catRepo, _, _, _, _ := setupCatalogServiceWithCopy()
	ctx := context.Background()

	sourceCat := &models.Catalog{
		ID: "src-id", Name: "staging", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusValid,
	}
	targetCat := &models.Catalog{
		ID: "tgt-id", Name: "prod", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusValid,
	}
	catRepo.On("GetByName", ctx, "staging").Return(sourceCat, nil)
	catRepo.On("GetByName", ctx, "prod").Return(targetCat, nil)
	catRepo.On("UpdateName", ctx, "tgt-id", mock.MatchedBy(func(name string) bool {
		return name == "prod-archive" // custom archive name
	})).Return(nil)
	catRepo.On("UpdateName", ctx, "src-id", "prod").Return(nil)
	catRepo.On("UpdatePublished", ctx, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	result, err := svc.ReplaceCatalog(ctx, "staging", "prod", "prod-archive")
	require.NoError(t, err)
	assert.Equal(t, "src-id", result.ID)

	// Verify rename calls
	catRepo.AssertCalled(t, "UpdateName", ctx, "tgt-id", "prod-archive")
	catRepo.AssertCalled(t, "UpdateName", ctx, "src-id", "prod")
}

// T-17.28: Replace with default archive name uses {target}-archive-{timestamp}
func TestT17_28_ReplaceCatalog_DefaultArchiveName(t *testing.T) {
	svc, catRepo, _, _, _, _ := setupCatalogServiceWithCopy()
	ctx := context.Background()

	sourceCat := &models.Catalog{
		ID: "src-id", Name: "staging", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusValid,
	}
	targetCat := &models.Catalog{
		ID: "tgt-id", Name: "prod", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusValid,
	}
	catRepo.On("GetByName", ctx, "staging").Return(sourceCat, nil)
	catRepo.On("GetByName", ctx, "prod").Return(targetCat, nil)
	catRepo.On("UpdateName", ctx, "tgt-id", mock.MatchedBy(func(name string) bool {
		// Should start with "prod-archive-" and be DNS-label compatible
		return len(name) > len("prod-archive-") && name[:14] == "prod-archive-2"
	})).Return(nil)
	catRepo.On("UpdateName", ctx, "src-id", "prod").Return(nil)
	catRepo.On("UpdatePublished", ctx, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	_, err := svc.ReplaceCatalog(ctx, "staging", "prod", "")
	require.NoError(t, err)
}

// T-17.30: Replace requires source valid status — draft returns error
func TestT17_30_ReplaceCatalog_DraftSource(t *testing.T) {
	svc, catRepo, _, _, _, _ := setupCatalogServiceWithCopy()
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "staging").Return(&models.Catalog{
		ID: "src-id", Name: "staging", ValidationStatus: models.ValidationStatusDraft,
	}, nil)
	catRepo.On("GetByName", ctx, "prod").Return(&models.Catalog{
		ID: "tgt-id", Name: "prod",
	}, nil)

	_, err := svc.ReplaceCatalog(ctx, "staging", "prod", "")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

// T-17.32: Replace with nonexistent source returns NotFound
func TestT17_32_ReplaceCatalog_SourceNotFound(t *testing.T) {
	svc, catRepo, _, _, _, _ := setupCatalogServiceWithCopy()
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "nonexistent").Return(nil, domainerrors.NewNotFound("Catalog", "nonexistent"))

	_, err := svc.ReplaceCatalog(ctx, "nonexistent", "prod", "")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
}

// T-17.33: Replace with nonexistent target returns NotFound
func TestT17_33_ReplaceCatalog_TargetNotFound(t *testing.T) {
	svc, catRepo, _, _, _, _ := setupCatalogServiceWithCopy()
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "staging").Return(&models.Catalog{
		ID: "src-id", Name: "staging", ValidationStatus: models.ValidationStatusValid,
	}, nil)
	catRepo.On("GetByName", ctx, "nonexistent").Return(nil, domainerrors.NewNotFound("Catalog", "nonexistent"))

	_, err := svc.ReplaceCatalog(ctx, "staging", "nonexistent", "")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
}

// T-17.34: Replace where source equals target returns error
func TestT17_34_ReplaceCatalog_SameSourceTarget(t *testing.T) {
	svc, _, _, _, _, _ := setupCatalogServiceWithCopy()
	ctx := context.Background()

	_, err := svc.ReplaceCatalog(ctx, "same", "same", "")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

// T-17.35: Replace with invalid archive name returns error
func TestT17_35_ReplaceCatalog_InvalidArchiveName(t *testing.T) {
	svc, catRepo, _, _, _, _ := setupCatalogServiceWithCopy()
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "staging").Return(&models.Catalog{
		ID: "src-id", Name: "staging", ValidationStatus: models.ValidationStatusValid,
	}, nil)
	catRepo.On("GetByName", ctx, "prod").Return(&models.Catalog{
		ID: "tgt-id", Name: "prod",
	}, nil)

	_, err := svc.ReplaceCatalog(ctx, "staging", "prod", "INVALID_NAME")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

// T-17.37/38: Replace transfers published state from target to source, archive unpublished
func TestT17_37_ReplaceCatalog_PublishedStateTransfer(t *testing.T) {
	svc, catRepo, cvRepo, _, _, _, crMgr := setupCatalogServiceWithCopyAndCR()
	ctx := context.Background()

	pubTime := time.Now()
	sourceCat := &models.Catalog{
		ID: "src-id", Name: "staging", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusValid,
	}
	targetCat := &models.Catalog{
		ID: "tgt-id", Name: "prod", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusValid,
		Published: true, PublishedAt: &pubTime,
	}
	catRepo.On("GetByName", ctx, "staging").Return(sourceCat, nil)
	catRepo.On("GetByName", ctx, "prod").Return(targetCat, nil)
	catRepo.On("UpdateName", ctx, "tgt-id", "prod-archive").Return(nil)
	catRepo.On("UpdateName", ctx, "src-id", "prod").Return(nil)
	catRepo.On("UpdatePublished", ctx, "src-id", true, mock.AnythingOfType("*time.Time")).Return(nil)
	catRepo.On("UpdatePublished", ctx, "tgt-id", false, (*time.Time)(nil)).Return(nil)
	cvRepo.On("GetByID", ctx, "cv1").Return(&models.CatalogVersion{ID: "cv1", VersionLabel: "v1"}, nil)

	result, err := svc.ReplaceCatalog(ctx, "staging", "prod", "prod-archive")
	require.NoError(t, err)
	assert.Equal(t, "src-id", result.ID)

	// Source (now "prod") should be published
	catRepo.AssertCalled(t, "UpdatePublished", ctx, "src-id", true, mock.AnythingOfType("*time.Time"))
	// Archive should be unpublished
	catRepo.AssertCalled(t, "UpdatePublished", ctx, "tgt-id", false, (*time.Time)(nil))
	// SyncCR should have been called
	assert.NotNil(t, crMgr.createOrUpdateSpec)
	// Archive CR should be deleted
	assert.Equal(t, "prod-archive", crMgr.deleteCalledWith)
}

// T-17.39: Replace on unpublished target — both remain unpublished
func TestT17_39_ReplaceCatalog_UnpublishedTarget(t *testing.T) {
	svc, catRepo, _, _, _, _ := setupCatalogServiceWithCopy()
	ctx := context.Background()

	sourceCat := &models.Catalog{
		ID: "src-id", Name: "staging", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusValid,
	}
	targetCat := &models.Catalog{
		ID: "tgt-id", Name: "prod", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusValid,
		Published: false,
	}
	catRepo.On("GetByName", ctx, "staging").Return(sourceCat, nil)
	catRepo.On("GetByName", ctx, "prod").Return(targetCat, nil)
	catRepo.On("UpdateName", ctx, "tgt-id", "prod-archive").Return(nil)
	catRepo.On("UpdateName", ctx, "src-id", "prod").Return(nil)

	_, err := svc.ReplaceCatalog(ctx, "staging", "prod", "prod-archive")
	require.NoError(t, err)

	// No UpdatePublished calls since target was not published
	catRepo.AssertNotCalled(t, "UpdatePublished", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// T-17.xx: Replace with long target name — auto-generated archive name exceeds 63 chars
func TestReplaceCatalog_ArchiveNameTooLong(t *testing.T) {
	svc, catRepo, _, _, _, _ := setupCatalogServiceWithCopy()
	ctx := context.Background()

	longName := "a]aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" // 53 chars — + "-archive-20260317" = 70 chars
	longName = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"   // 54 chars
	catRepo.On("GetByName", ctx, "staging").Return(&models.Catalog{
		ID: "src-id", Name: "staging", ValidationStatus: models.ValidationStatusValid,
	}, nil)
	catRepo.On("GetByName", ctx, longName).Return(&models.Catalog{
		ID: "tgt-id", Name: longName,
	}, nil)

	_, err := svc.ReplaceCatalog(ctx, "staging", longName, "")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
	assert.Contains(t, err.Error(), "archive_name")
}

// ReplaceCatalog — source was published, gets unpublished and CR cleaned up
func TestReplaceCatalog_SourcePublished(t *testing.T) {
	svc, catRepo, _, _, _, _, crMgr := setupCatalogServiceWithCopyAndCR()
	ctx := context.Background()

	sourceCat := &models.Catalog{
		ID: "src-id", Name: "staging", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusValid,
		Published: true,
	}
	targetCat := &models.Catalog{
		ID: "tgt-id", Name: "prod", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusValid,
		Published: false,
	}
	catRepo.On("GetByName", ctx, "staging").Return(sourceCat, nil)
	catRepo.On("GetByName", ctx, "prod").Return(targetCat, nil)
	catRepo.On("UpdateName", ctx, "tgt-id", "prod-archive").Return(nil)
	catRepo.On("UpdateName", ctx, "src-id", "prod").Return(nil)
	catRepo.On("UpdatePublished", ctx, "src-id", false, (*time.Time)(nil)).Return(nil)

	result, err := svc.ReplaceCatalog(ctx, "staging", "prod", "prod-archive")
	require.NoError(t, err)
	// Source should be unpublished in the response
	assert.False(t, result.Published)
	assert.Nil(t, result.PublishedAt)
	// Source's old CR should be deleted
	assert.Equal(t, "staging", crMgr.deleteCalledWith)
}

// ReplaceCatalog — source published, UpdatePublished error
func TestReplaceCatalog_SourcePublishedUnpublishError(t *testing.T) {
	svc, catRepo, _, _, _, _ := setupCatalogServiceWithCopy()
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "staging").Return(&models.Catalog{
		ID: "src-id", Name: "staging", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusValid,
		Published: true,
	}, nil)
	catRepo.On("GetByName", ctx, "prod").Return(&models.Catalog{
		ID: "tgt-id", Name: "prod", CatalogVersionID: "cv1",
		Published: false, // target NOT published → enters else-if branch
	}, nil)
	catRepo.On("UpdateName", ctx, "tgt-id", "prod-archive").Return(nil)
	catRepo.On("UpdateName", ctx, "src-id", "prod").Return(nil)
	catRepo.On("UpdatePublished", ctx, "src-id", false, (*time.Time)(nil)).Return(fmt.Errorf("unpublish error"))

	_, err := svc.ReplaceCatalog(ctx, "staging", "prod", "prod-archive")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unpublish error")
}

// ReplaceCatalog — step 2 rename error (rollback tested by existing test, but ensure error propagates)
func TestReplaceCatalog_Step2Error(t *testing.T) {
	svc, catRepo, _, _, _, _ := setupCatalogServiceWithCopy()
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "staging").Return(&models.Catalog{
		ID: "src-id", Name: "staging", ValidationStatus: models.ValidationStatusValid,
	}, nil)
	catRepo.On("GetByName", ctx, "prod").Return(&models.Catalog{
		ID: "tgt-id", Name: "prod",
	}, nil)
	catRepo.On("UpdateName", ctx, "tgt-id", "prod-archive").Return(nil) // Step 1 succeeds
	catRepo.On("UpdateName", ctx, "src-id", "prod").Return(fmt.Errorf("step2 error")) // Step 2 fails

	_, err := svc.ReplaceCatalog(ctx, "staging", "prod", "prod-archive")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "step2 error")
}

// ReplaceCatalog — UpdatePublished error in step 3
func TestReplaceCatalog_PublishTransferError(t *testing.T) {
	svc, catRepo, _, _, _, _ := setupCatalogServiceWithCopy()
	ctx := context.Background()

	pubTime := time.Now()
	catRepo.On("GetByName", ctx, "staging").Return(&models.Catalog{
		ID: "src-id", Name: "staging", ValidationStatus: models.ValidationStatusValid,
	}, nil)
	catRepo.On("GetByName", ctx, "prod").Return(&models.Catalog{
		ID: "tgt-id", Name: "prod", Published: true, PublishedAt: &pubTime,
	}, nil)
	catRepo.On("UpdateName", ctx, "tgt-id", "prod-archive").Return(nil)
	catRepo.On("UpdateName", ctx, "src-id", "prod").Return(nil)
	catRepo.On("UpdatePublished", ctx, "src-id", true, mock.AnythingOfType("*time.Time")).Return(fmt.Errorf("publish error"))

	_, err := svc.ReplaceCatalog(ctx, "staging", "prod", "prod-archive")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "publish error")
}

// ReplaceCatalog — without TransactionManager (nil fallback)
func TestReplaceCatalog_NilTxManager(t *testing.T) {
	catRepo := new(mocks.MockCatalogRepo)
	cvRepo := new(mocks.MockCatalogVersionRepo)
	instRepo := new(mocks.MockEntityInstanceRepo)
	iavRepo := new(mocks.MockInstanceAttributeValueRepo)
	linkRepo := new(mocks.MockAssociationLinkRepo)
	svc := operational.NewCatalogService(catRepo, cvRepo, instRepo, nil, "", operational.WithCopyDeps(iavRepo, linkRepo))
	ctx := context.Background()

	catRepo.On("GetByName", ctx, "staging").Return(&models.Catalog{
		ID: "src-id", Name: "staging", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusValid,
	}, nil)
	catRepo.On("GetByName", ctx, "prod").Return(&models.Catalog{
		ID: "tgt-id", Name: "prod", CatalogVersionID: "cv1",
	}, nil)
	catRepo.On("UpdateName", ctx, "tgt-id", "prod-archive").Return(nil)
	catRepo.On("UpdateName", ctx, "src-id", "prod").Return(nil)

	_, err := svc.ReplaceCatalog(ctx, "staging", "prod", "prod-archive")
	require.NoError(t, err)
}

// T-17.44: Replace with nil crManager skips CR operations
func TestT17_44_ReplaceCatalog_NilCRManager(t *testing.T) {
	svc, catRepo, _, _, _, _ := setupCatalogServiceWithCopy()
	ctx := context.Background()

	pubTime := time.Now()
	sourceCat := &models.Catalog{
		ID: "src-id", Name: "staging", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusValid,
	}
	targetCat := &models.Catalog{
		ID: "tgt-id", Name: "prod", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusValid,
		Published: true, PublishedAt: &pubTime,
	}
	catRepo.On("GetByName", ctx, "staging").Return(sourceCat, nil)
	catRepo.On("GetByName", ctx, "prod").Return(targetCat, nil)
	catRepo.On("UpdateName", ctx, "tgt-id", "prod-archive").Return(nil)
	catRepo.On("UpdateName", ctx, "src-id", "prod").Return(nil)
	catRepo.On("UpdatePublished", ctx, "src-id", true, mock.AnythingOfType("*time.Time")).Return(nil)
	catRepo.On("UpdatePublished", ctx, "tgt-id", false, (*time.Time)(nil)).Return(nil)

	_, err := svc.ReplaceCatalog(ctx, "staging", "prod", "prod-archive")
	require.NoError(t, err)
}
