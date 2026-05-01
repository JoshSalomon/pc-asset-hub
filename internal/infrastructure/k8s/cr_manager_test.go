package k8s

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	v1alpha1 "github.com/project-catalyst/pc-asset-hub/internal/operator/api/v1alpha1"
	"github.com/project-catalyst/pc-asset-hub/internal/service/meta"
	"github.com/project-catalyst/pc-asset-hub/internal/service/operational"
)

func testScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(s)
	return s
}

// T-CV.16: CreateOrUpdate creates new CatalogVersion CR with correct spec
func TestTCV16_CreateOrUpdateCreatesNew(t *testing.T) {
	s := testScheme()
	cl := fake.NewClientBuilder().WithScheme(s).Build()
	mgr := NewK8sCRManager(cl)

	spec := meta.CatalogVersionCRSpec{
		Name:           "release-1",
		Namespace:      "assethub",
		VersionLabel:   "Release 1",
		Description:    "First release",
		LifecycleStage: "testing",
		EntityTypes:    []string{"Device", "Network"},
		SourceDBID:     "uuid-123",
		PromotedBy:     "admin",
		PromotedAt:     "2026-02-17T10:00:00Z",
	}

	err := mgr.CreateOrUpdate(context.Background(), spec)
	require.NoError(t, err)

	cv := &v1alpha1.CatalogVersion{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "release-1", Namespace: "assethub"}, cv)
	require.NoError(t, err)
	assert.Equal(t, "Release 1", cv.Spec.VersionLabel)
	assert.Equal(t, "First release", cv.Spec.Description)
	assert.Equal(t, "testing", cv.Spec.LifecycleStage)
	assert.Equal(t, []string{"Device", "Network"}, cv.Spec.EntityTypes)
}

// T-CV.17: CreateOrUpdate updates existing CatalogVersion CR idempotently
func TestTCV17_CreateOrUpdateUpdatesExisting(t *testing.T) {
	s := testScheme()
	existing := &v1alpha1.CatalogVersion{
		ObjectMeta: metav1.ObjectMeta{Name: "release-1", Namespace: "assethub"},
		Spec: v1alpha1.CatalogVersionSpec{
			VersionLabel:   "Release 1",
			LifecycleStage: "testing",
			EntityTypes:    []string{"Device"},
		},
	}
	cl := fake.NewClientBuilder().WithScheme(s).WithObjects(existing).Build()
	mgr := NewK8sCRManager(cl)

	spec := meta.CatalogVersionCRSpec{
		Name:           "release-1",
		Namespace:      "assethub",
		VersionLabel:   "Release 1",
		Description:    "Updated description",
		LifecycleStage: "production",
		EntityTypes:    []string{"Device", "Network"},
		SourceDBID:     "uuid-123",
		PromotedBy:     "admin",
		PromotedAt:     "2026-02-17T12:00:00Z",
	}

	err := mgr.CreateOrUpdate(context.Background(), spec)
	require.NoError(t, err)

	cv := &v1alpha1.CatalogVersion{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "release-1", Namespace: "assethub"}, cv)
	require.NoError(t, err)
	assert.Equal(t, "production", cv.Spec.LifecycleStage)
	assert.Equal(t, "Updated description", cv.Spec.Description)
	assert.Equal(t, []string{"Device", "Network"}, cv.Spec.EntityTypes)
}

// T-CV.18: Delete removes existing CatalogVersion CR
func TestTCV18_DeleteRemovesExisting(t *testing.T) {
	s := testScheme()
	existing := &v1alpha1.CatalogVersion{
		ObjectMeta: metav1.ObjectMeta{Name: "release-1", Namespace: "assethub"},
		Spec: v1alpha1.CatalogVersionSpec{
			VersionLabel:   "Release 1",
			LifecycleStage: "testing",
		},
	}
	cl := fake.NewClientBuilder().WithScheme(s).WithObjects(existing).Build()
	mgr := NewK8sCRManager(cl)

	err := mgr.Delete(context.Background(), "release-1", "assethub")
	require.NoError(t, err)

	cv := &v1alpha1.CatalogVersion{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "release-1", Namespace: "assethub"}, cv)
	assert.Error(t, err)
}

// T-CV.19: Delete of non-existent CR returns no error
func TestTCV19_DeleteNonExistentReturnsNil(t *testing.T) {
	s := testScheme()
	cl := fake.NewClientBuilder().WithScheme(s).Build()
	mgr := NewK8sCRManager(cl)

	err := mgr.Delete(context.Background(), "nonexistent", "assethub")
	assert.NoError(t, err)
}

// T-CV.20: CreateOrUpdate sets all three annotations
func TestTCV20_CreateOrUpdateSetsAnnotations(t *testing.T) {
	s := testScheme()
	cl := fake.NewClientBuilder().WithScheme(s).Build()
	mgr := NewK8sCRManager(cl)

	spec := meta.CatalogVersionCRSpec{
		Name:           "release-2",
		Namespace:      "assethub",
		VersionLabel:   "Release 2",
		LifecycleStage: "testing",
		SourceDBID:     "db-uuid-456",
		PromotedBy:     "admin-user",
		PromotedAt:     "2026-02-17T15:30:00Z",
	}

	err := mgr.CreateOrUpdate(context.Background(), spec)
	require.NoError(t, err)

	cv := &v1alpha1.CatalogVersion{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "release-2", Namespace: "assethub"}, cv)
	require.NoError(t, err)
	assert.Equal(t, "db-uuid-456", cv.Annotations["assethub.project-catalyst.io/source-db-id"])
	assert.Equal(t, "admin-user", cv.Annotations["assethub.project-catalyst.io/promoted-by"])
	assert.Equal(t, "2026-02-17T15:30:00Z", cv.Annotations["assethub.project-catalyst.io/promoted-at"])
}

// === Catalog CR Manager Tests ===

// T-16.47: CatalogCRManager.CreateOrUpdate creates Catalog CR with correct spec
func TestT16_47_CatalogCreateOrUpdateNew(t *testing.T) {
	s := testScheme()
	cl := fake.NewClientBuilder().WithScheme(s).Build()
	mgr := NewK8sCatalogCRManager(cl)

	spec := operational.CatalogCRSpec{
		Name:                "prod-catalog",
		Namespace:           "assethub",
		CatalogVersionLabel: "v1.0",
		ValidationStatus:    "valid",
		APIEndpoint:         "/api/data/v1/catalogs/prod-catalog",
		SourceDBID:          "db-uuid-123",
		PublishedAt:         "2026-03-16T10:00:00Z",
	}

	err := mgr.CreateOrUpdate(context.Background(), spec)
	require.NoError(t, err)

	cat := &v1alpha1.Catalog{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "prod-catalog", Namespace: "assethub"}, cat)
	require.NoError(t, err)
	assert.Equal(t, "prod-catalog", cat.Spec.CatalogName)
	assert.Equal(t, "v1.0", cat.Spec.CatalogVersionLabel)
	assert.Equal(t, "valid", cat.Spec.ValidationStatus)
	assert.Equal(t, "/api/data/v1/catalogs/prod-catalog", cat.Spec.APIEndpoint)
}

// T-16.48: CatalogCRManager.CreateOrUpdate sets annotations
func TestT16_48_CatalogAnnotations(t *testing.T) {
	s := testScheme()
	cl := fake.NewClientBuilder().WithScheme(s).Build()
	mgr := NewK8sCatalogCRManager(cl)

	spec := operational.CatalogCRSpec{
		Name:      "prod-catalog",
		Namespace: "assethub",
		SourceDBID:  "db-uuid-123",
		PublishedAt: "2026-03-16T10:00:00Z",
	}

	err := mgr.CreateOrUpdate(context.Background(), spec)
	require.NoError(t, err)

	cat := &v1alpha1.Catalog{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "prod-catalog", Namespace: "assethub"}, cat)
	require.NoError(t, err)
	assert.Equal(t, "db-uuid-123", cat.Annotations["assethub.project-catalyst.io/source-db-id"])
	assert.Equal(t, "2026-03-16T10:00:00Z", cat.Annotations["assethub.project-catalyst.io/published-at"])
}

// T-16.49: CatalogCRManager.CreateOrUpdate updates existing CR
func TestT16_49_CatalogUpdateExisting(t *testing.T) {
	s := testScheme()
	existing := &v1alpha1.Catalog{
		ObjectMeta: metav1.ObjectMeta{Name: "prod-catalog", Namespace: "assethub"},
		Spec: v1alpha1.CatalogSpec{
			CatalogName:      "prod-catalog",
			ValidationStatus: "valid",
		},
	}
	cl := fake.NewClientBuilder().WithScheme(s).WithObjects(existing).Build()
	mgr := NewK8sCatalogCRManager(cl)

	spec := operational.CatalogCRSpec{
		Name:                "prod-catalog",
		Namespace:           "assethub",
		CatalogVersionLabel: "v2.0",
		ValidationStatus:    "draft",
		SourceDBID:          "updated-id",
		PublishedAt:         "2026-03-16T12:00:00Z",
	}

	err := mgr.CreateOrUpdate(context.Background(), spec)
	require.NoError(t, err)

	cat := &v1alpha1.Catalog{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "prod-catalog", Namespace: "assethub"}, cat)
	require.NoError(t, err)
	assert.Equal(t, "v2.0", cat.Spec.CatalogVersionLabel)
	assert.Equal(t, "draft", cat.Spec.ValidationStatus)
}

// T-16.50: CatalogCRManager.Delete removes Catalog CR
func TestT16_50_CatalogDelete(t *testing.T) {
	s := testScheme()
	existing := &v1alpha1.Catalog{
		ObjectMeta: metav1.ObjectMeta{Name: "prod-catalog", Namespace: "assethub"},
	}
	cl := fake.NewClientBuilder().WithScheme(s).WithObjects(existing).Build()
	mgr := NewK8sCatalogCRManager(cl)

	err := mgr.Delete(context.Background(), "prod-catalog", "assethub")
	require.NoError(t, err)

	cat := &v1alpha1.Catalog{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "prod-catalog", Namespace: "assethub"}, cat)
	assert.Error(t, err) // not found
}

// T-16.51: CatalogCRManager.Delete on nonexistent is idempotent
func TestT16_51_CatalogDeleteNonexistent(t *testing.T) {
	s := testScheme()
	cl := fake.NewClientBuilder().WithScheme(s).Build()
	mgr := NewK8sCatalogCRManager(cl)

	err := mgr.Delete(context.Background(), "nonexistent", "assethub")
	require.NoError(t, err) // no error
}

// === Error Path Tests ===

// CatalogVersion CreateOrUpdate: Get returns non-NotFound error (line 53-55)
func TestCVCreateOrUpdate_GetError(t *testing.T) {
	s := testScheme()
	cl := fake.NewClientBuilder().WithScheme(s).
		WithInterceptorFuncs(interceptor.Funcs{
			Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				if _, ok := obj.(*v1alpha1.CatalogVersion); ok {
					return fmt.Errorf("injected Get error")
				}
				return c.Get(ctx, key, obj, opts...)
			},
		}).Build()
	mgr := NewK8sCRManager(cl)

	err := mgr.CreateOrUpdate(context.Background(), meta.CatalogVersionCRSpec{
		Name: "test", Namespace: "assethub",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get CatalogVersion")
}

// Catalog CreateOrUpdate: Get returns non-NotFound error (line 106-108)
func TestCatalogCreateOrUpdate_GetError(t *testing.T) {
	s := testScheme()
	cl := fake.NewClientBuilder().WithScheme(s).
		WithInterceptorFuncs(interceptor.Funcs{
			Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				if _, ok := obj.(*v1alpha1.Catalog); ok {
					return fmt.Errorf("injected Catalog Get error")
				}
				return c.Get(ctx, key, obj, opts...)
			},
		}).Build()
	mgr := NewK8sCatalogCRManager(cl)

	err := mgr.CreateOrUpdate(context.Background(), operational.CatalogCRSpec{
		Name: "test", Namespace: "assethub",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get Catalog")
}

// CatalogVersion Delete: Get returns non-NotFound error (line 145-146)
func TestCVDelete_GetError(t *testing.T) {
	s := testScheme()
	cl := fake.NewClientBuilder().WithScheme(s).
		WithInterceptorFuncs(interceptor.Funcs{
			Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				if _, ok := obj.(*v1alpha1.CatalogVersion); ok {
					return fmt.Errorf("injected CV Delete Get error")
				}
				return c.Get(ctx, key, obj, opts...)
			},
		}).Build()
	mgr := NewK8sCRManager(cl)

	err := mgr.Delete(context.Background(), "some-cv", "assethub")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get CatalogVersion")
	assert.Contains(t, err.Error(), "for deletion")
}

// Catalog Delete: Get returns non-NotFound error (line 131-132)
func TestCatalogDelete_GetError(t *testing.T) {
	s := testScheme()
	cl := fake.NewClientBuilder().WithScheme(s).
		WithInterceptorFuncs(interceptor.Funcs{
			Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				if _, ok := obj.(*v1alpha1.Catalog); ok {
					return fmt.Errorf("injected Catalog Delete Get error")
				}
				return c.Get(ctx, key, obj, opts...)
			},
		}).Build()
	mgr := NewK8sCatalogCRManager(cl)

	err := mgr.Delete(context.Background(), "some-cat", "assethub")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get Catalog")
	assert.Contains(t, err.Error(), "for deletion")
}
