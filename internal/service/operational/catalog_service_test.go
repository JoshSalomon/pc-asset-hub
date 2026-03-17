package operational_test

import (
	"context"
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
