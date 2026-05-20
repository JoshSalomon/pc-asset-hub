package export_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository/mocks"
	"github.com/project-catalyst/pc-asset-hub/internal/service/operational/export"
)

func setupPublishService() (*export.ExportBindingService, *mocks.MockExportBindingRepo, *mocks.MockCatalogRepo, *export.ExporterRegistry, *export.InMemoryPreviewCache, *mocks.MockCatalogVersionPinRepo) {
	bindingRepo := new(mocks.MockExportBindingRepo)
	catalogRepo := new(mocks.MockCatalogRepo)
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	etRepo := new(mocks.MockEntityTypeRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	registry := export.NewExporterRegistry()
	cache := export.NewInMemoryPreviewCache()

	// Default: empty schema (no pins); allow Update calls from executeBinding
	pinRepo.On("ListByCatalogVersion", mock.Anything, mock.Anything).Maybe().Return([]*models.CatalogVersionPin{}, nil)
	bindingRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.ExportBinding")).Maybe().Return(nil)

	svc := export.NewExportBindingService(
		bindingRepo, catalogRepo, registry,
		cvRepo, pinRepo, etvRepo, etRepo, attrRepo, assocRepo,
		export.WithPreviewCache(cache),
	)
	return svc, bindingRepo, catalogRepo, registry, cache, pinRepo
}

// T-34.77: PublishPreview caches artifacts via PreviewCache with session token
func TestPublishPreview_CachesArtifacts(t *testing.T) {
	svc, bindingRepo, catalogRepo, registry, cache, _ := setupPublishService()
	ctx := context.Background()

	registry.Register(&stubExporter{
		name: "mcp-gateway",
		exportOut: &export.ExportOutput{
			Artifacts: []export.K8sArtifact{{Name: "test", Kind: "MCPServerRegistration", YAML: "test: true"}},
		},
	})

	catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
		UpdatedAt: time.Now(),
	}, nil)
	bindingRepo.On("ListByCatalog", ctx, "cat1").Return([]*models.ExportBinding{
		{ID: "b1", CatalogID: "cat1", ExporterName: "mcp-gateway",
			Parameters: map[string]string{}, Enabled: true},
	}, nil)

	result, err := svc.PublishPreview(ctx, "my-catalog")
	require.NoError(t, err)
	assert.NotEmpty(t, result.SessionToken)
	assert.False(t, result.HasFailures)

	entry, err := cache.Retrieve(result.SessionToken)
	require.NoError(t, err)
	assert.NotEmpty(t, entry.Artifacts["b1"])
}

// T-34.78: PublishPreview returns per-binding results
func TestPublishPreview_PerBindingResults(t *testing.T) {
	svc, bindingRepo, catalogRepo, registry, _, _ := setupPublishService()
	ctx := context.Background()

	registry.Register(&stubExporter{
		name:    "mcp-gateway",
		exportOut: &export.ExportOutput{Artifacts: []export.K8sArtifact{{Name: "a"}}},
	})

	catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	bindingRepo.On("ListByCatalog", ctx, "cat1").Return([]*models.ExportBinding{
		{ID: "b1", CatalogID: "cat1", ExporterName: "mcp-gateway", Parameters: map[string]string{}, Enabled: true},
		{ID: "b2", CatalogID: "cat1", ExporterName: "missing-exporter", Parameters: map[string]string{}, Enabled: true},
	}, nil)

	result, err := svc.PublishPreview(ctx, "my-catalog")
	require.NoError(t, err)
	require.Len(t, result.Bindings, 2)
	assert.Equal(t, "success", result.Bindings[0].Status)
	assert.Equal(t, "failed", result.Bindings[1].Status)
	assert.True(t, result.HasFailures)
}

// T-34.85: PreviewCache stores CatalogUpdatedAt for optimistic lock
func TestPreviewCache_StoresCatalogUpdatedAt(t *testing.T) {
	cache := export.NewInMemoryPreviewCache()
	updatedAt := time.Date(2026, 5, 10, 14, 30, 0, 0, time.UTC)
	entry := export.PreviewCacheEntry{
		CatalogName:      "my-catalog",
		CatalogUpdatedAt: updatedAt,
	}
	require.NoError(t, cache.Store("token1", entry, 5*time.Minute))

	got, err := cache.Retrieve("token1")
	require.NoError(t, err)
	assert.Equal(t, updatedAt, got.CatalogUpdatedAt)
}

// T-34.79: GetCachedArtifacts retrieves cached artifacts by binding ID
func TestGetCachedArtifacts_Success(t *testing.T) {
	svc, bindingRepo, catalogRepo, registry, cache, _ := setupPublishService()
	ctx := context.Background()

	registry.Register(&stubExporter{
		name:    "mcp-gateway",
		exportOut: &export.ExportOutput{Artifacts: []export.K8sArtifact{{Name: "test-cr", YAML: "test: true"}}},
	})

	catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	bindingRepo.On("ListByCatalog", ctx, "cat1").Return([]*models.ExportBinding{
		{ID: "b1", CatalogID: "cat1", ExporterName: "mcp-gateway", Parameters: map[string]string{}, Enabled: true},
	}, nil)

	result, err := svc.PublishPreview(ctx, "my-catalog")
	require.NoError(t, err)

	artifacts, err := svc.GetCachedArtifacts(ctx, "my-catalog", result.SessionToken, "b1")
	require.NoError(t, err)
	assert.Len(t, artifacts, 1)

	_ = cache // used implicitly through svc
}

// T-34.80: Expired token returns error
func TestGetCachedArtifacts_ExpiredToken(t *testing.T) {
	cache := export.NewInMemoryPreviewCache()
	entry := export.PreviewCacheEntry{CatalogName: "c"}
	require.NoError(t, cache.Store("token1", entry, 1*time.Millisecond))
	time.Sleep(5 * time.Millisecond)

	_, err := cache.Retrieve("token1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

// T-34.76: PublishPreview re-validates bindings via ValidateSchema
// Register an exporter whose ValidateSchema returns an error; verify preview has the failure.
func TestPublishPreview_RevalidatesSchema(t *testing.T) {
	svc, bindingRepo, catalogRepo, registry, _, _ := setupPublishService()
	ctx := context.Background()

	registry.Register(&stubExporter{
		name:        "schema-drifted",
		validateErr: fmt.Errorf("entity type %q is missing required attribute 'route_name'", "mcp-server"),
	})

	catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	bindingRepo.On("ListByCatalog", ctx, "cat1").Return([]*models.ExportBinding{
		{ID: "b1", CatalogID: "cat1", ExporterName: "schema-drifted",
			Parameters: map[string]string{}, Enabled: true},
	}, nil)

	result, err := svc.PublishPreview(ctx, "my-catalog")
	require.NoError(t, err) // Preview itself succeeds; individual bindings may fail
	require.Len(t, result.Bindings, 1)
	assert.Equal(t, "failed", result.Bindings[0].Status)
	assert.Contains(t, result.Bindings[0].Error, "route_name")
	assert.True(t, result.HasFailures)
}

// T-34.81: Publish with modified catalog — optimistic lock via CatalogUpdatedAt
// Store a preview with CatalogUpdatedAt = T1, verify GetPreviewEntry returns matching timestamp.
func TestPublishPreview_OptimisticLockTimestamp(t *testing.T) {
	svc, bindingRepo, catalogRepo, registry, _, _ := setupPublishService()
	ctx := context.Background()

	catalogUpdatedAt := time.Date(2026, 5, 10, 14, 30, 0, 0, time.UTC)

	registry.Register(&stubExporter{
		name:    "mcp-gateway",
		exportOut: &export.ExportOutput{Artifacts: []export.K8sArtifact{{Name: "a"}}},
	})

	catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
		UpdatedAt: catalogUpdatedAt,
	}, nil)
	bindingRepo.On("ListByCatalog", ctx, "cat1").Return([]*models.ExportBinding{
		{ID: "b1", CatalogID: "cat1", ExporterName: "mcp-gateway",
			Parameters: map[string]string{}, Enabled: true},
	}, nil)

	result, err := svc.PublishPreview(ctx, "my-catalog")
	require.NoError(t, err)

	// Retrieve the cached entry and verify the timestamp matches the catalog's UpdatedAt
	entry, err := svc.GetPreviewEntry(result.SessionToken)
	require.NoError(t, err)
	assert.Equal(t, catalogUpdatedAt, entry.CatalogUpdatedAt,
		"cached CatalogUpdatedAt must match catalog.UpdatedAt for optimistic lock comparison")
}

// T-34.82: RunAll works correctly for fire-and-forget (backward compat without token)
// Verifies RunAll executes all enabled bindings and returns per-binding results.
func TestRunAll_FireAndForget(t *testing.T) {
	svc, bindingRepo, catalogRepo, registry, _, _ := setupPublishService()
	ctx := context.Background()

	registry.Register(&stubExporter{
		name: "mcp-gateway",
		exportOut: &export.ExportOutput{
			Artifacts: []export.K8sArtifact{
				{Name: "server1", Kind: "MCPServerRegistration", YAML: "test: true"},
			},
		},
	})

	catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	bindingRepo.On("ListByCatalog", ctx, "cat1").Return([]*models.ExportBinding{
		{ID: "b1", CatalogID: "cat1", ExporterName: "mcp-gateway",
			Parameters: map[string]string{}, Enabled: true},
		{ID: "b2", CatalogID: "cat1", ExporterName: "mcp-gateway",
			Parameters: map[string]string{}, Enabled: true},
		{ID: "b3", CatalogID: "cat1", ExporterName: "mcp-gateway",
			Parameters: map[string]string{}, Enabled: false}, // disabled — should be skipped
	}, nil)
	bindingRepo.On("Update", ctx, mock.AnythingOfType("*models.ExportBinding")).Return(nil)

	results, err := svc.RunAll(ctx, "my-catalog")
	require.NoError(t, err)
	// Only enabled bindings should run (b1 and b2; b3 is disabled)
	assert.Len(t, results, 2)
	for _, r := range results {
		assert.Equal(t, "success", r.Status)
		assert.Equal(t, 1, r.ArtifactCount)
	}
}
