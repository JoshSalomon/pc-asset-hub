package k8s

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1alpha1 "github.com/project-catalyst/pc-asset-hub/internal/operator/api/v1alpha1"
	"github.com/project-catalyst/pc-asset-hub/internal/service/meta"
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
