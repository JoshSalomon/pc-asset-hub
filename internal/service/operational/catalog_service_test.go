package operational_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository/mocks"
	"github.com/project-catalyst/pc-asset-hub/internal/service/operational"
)

func setupCatalogService() (*operational.CatalogService, *mocks.MockCatalogRepo, *mocks.MockCatalogVersionRepo, *mocks.MockEntityInstanceRepo) {
	catRepo := new(mocks.MockCatalogRepo)
	cvRepo := new(mocks.MockCatalogVersionRepo)
	instRepo := new(mocks.MockEntityInstanceRepo)
	svc := operational.NewCatalogService(catRepo, cvRepo, instRepo)
	return svc, catRepo, cvRepo, instRepo
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
