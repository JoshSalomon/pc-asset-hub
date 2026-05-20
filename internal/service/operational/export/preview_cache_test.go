package export_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/project-catalyst/pc-asset-hub/internal/service/operational/export"
)

func TestPreviewCache_StoreAndRetrieve(t *testing.T) {
	cache := export.NewInMemoryPreviewCache()
	entry := export.PreviewCacheEntry{
		CatalogName: "my-catalog",
		Artifacts:   map[string][]export.K8sArtifact{"b1": {{Name: "test", YAML: "test: true"}}},
	}
	require.NoError(t, cache.Store("token1", entry, 5*time.Minute))

	got, err := cache.Retrieve("token1")
	require.NoError(t, err)
	assert.Equal(t, "my-catalog", got.CatalogName)
	assert.Len(t, got.Artifacts["b1"], 1)
}

func TestPreviewCache_ExpiredReturnsError(t *testing.T) {
	cache := export.NewInMemoryPreviewCache()
	entry := export.PreviewCacheEntry{CatalogName: "my-catalog"}
	require.NoError(t, cache.Store("token1", entry, 1*time.Millisecond))

	time.Sleep(5 * time.Millisecond)
	_, err := cache.Retrieve("token1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestPreviewCache_NotFoundReturnsError(t *testing.T) {
	cache := export.NewInMemoryPreviewCache()
	_, err := cache.Retrieve("no-such")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestPreviewCache_Delete(t *testing.T) {
	cache := export.NewInMemoryPreviewCache()
	entry := export.PreviewCacheEntry{CatalogName: "my-catalog"}
	require.NoError(t, cache.Store("token1", entry, 5*time.Minute))

	cache.Delete("token1")
	_, err := cache.Retrieve("token1")
	require.Error(t, err)
}
