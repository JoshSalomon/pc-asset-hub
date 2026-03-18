package k8s

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/project-catalyst/pc-asset-hub/internal/operator/api/v1alpha1"
	"github.com/project-catalyst/pc-asset-hub/internal/service/meta"
	"github.com/project-catalyst/pc-asset-hub/internal/service/operational"
)

// K8sCRManager implements CatalogVersionCRManager using controller-runtime client.
type K8sCRManager struct {
	client client.Client
}

// NewK8sCRManager creates a new K8sCRManager.
func NewK8sCRManager(c client.Client) *K8sCRManager {
	return &K8sCRManager{client: c}
}

// CreateOrUpdate creates a new CatalogVersion CR or updates an existing one.
func (m *K8sCRManager) CreateOrUpdate(ctx context.Context, spec meta.CatalogVersionCRSpec) error {
	cv := &v1alpha1.CatalogVersion{}
	key := types.NamespacedName{Name: spec.Name, Namespace: spec.Namespace}
	err := m.client.Get(ctx, key, cv)

	if errors.IsNotFound(err) {
		cv = &v1alpha1.CatalogVersion{
			ObjectMeta: metav1.ObjectMeta{
				Name:      spec.Name,
				Namespace: spec.Namespace,
				Annotations: map[string]string{
					"assethub.project-catalyst.io/source-db-id": spec.SourceDBID,
					"assethub.project-catalyst.io/promoted-by":  spec.PromotedBy,
					"assethub.project-catalyst.io/promoted-at":  spec.PromotedAt,
				},
			},
			Spec: v1alpha1.CatalogVersionSpec{
				VersionLabel:   spec.VersionLabel,
				Description:    spec.Description,
				LifecycleStage: spec.LifecycleStage,
				EntityTypes:    spec.EntityTypes,
			},
		}
		return m.client.Create(ctx, cv)
	}
	if err != nil {
		return fmt.Errorf("failed to get CatalogVersion %s: %w", spec.Name, err)
	}

	// Update existing
	cv.Spec.VersionLabel = spec.VersionLabel
	cv.Spec.Description = spec.Description
	cv.Spec.LifecycleStage = spec.LifecycleStage
	cv.Spec.EntityTypes = spec.EntityTypes
	if cv.Annotations == nil {
		cv.Annotations = make(map[string]string)
	}
	cv.Annotations["assethub.project-catalyst.io/source-db-id"] = spec.SourceDBID
	cv.Annotations["assethub.project-catalyst.io/promoted-by"] = spec.PromotedBy
	cv.Annotations["assethub.project-catalyst.io/promoted-at"] = spec.PromotedAt
	return m.client.Update(ctx, cv)
}

// K8sCatalogCRManager adapts K8sCRManager to implement the CatalogCRManager interface.
type K8sCatalogCRManager struct {
	client client.Client
}

// NewK8sCatalogCRManager creates a new K8sCatalogCRManager.
func NewK8sCatalogCRManager(c client.Client) *K8sCatalogCRManager {
	return &K8sCatalogCRManager{client: c}
}

// CreateOrUpdate creates a new Catalog CR or updates an existing one.
func (m *K8sCatalogCRManager) CreateOrUpdate(ctx context.Context, spec operational.CatalogCRSpec) error {
	cat := &v1alpha1.Catalog{}
	key := types.NamespacedName{Name: spec.Name, Namespace: spec.Namespace}
	err := m.client.Get(ctx, key, cat)

	if errors.IsNotFound(err) {
		cat = &v1alpha1.Catalog{
			ObjectMeta: metav1.ObjectMeta{
				Name:      spec.Name,
				Namespace: spec.Namespace,
				Annotations: map[string]string{
					"assethub.project-catalyst.io/source-db-id": spec.SourceDBID,
					"assethub.project-catalyst.io/published-at": spec.PublishedAt,
				},
			},
			Spec: v1alpha1.CatalogSpec{
				CatalogName:         spec.Name,
				CatalogVersionLabel: spec.CatalogVersionLabel,
				ValidationStatus:    spec.ValidationStatus,
				APIEndpoint:         spec.APIEndpoint,
			},
		}
		return m.client.Create(ctx, cat)
	}
	if err != nil {
		return fmt.Errorf("failed to get Catalog %s: %w", spec.Name, err)
	}

	cat.Spec.CatalogName = spec.Name
	cat.Spec.CatalogVersionLabel = spec.CatalogVersionLabel
	cat.Spec.ValidationStatus = spec.ValidationStatus
	cat.Spec.APIEndpoint = spec.APIEndpoint
	cat.Spec.SyncVersion = cat.Spec.SyncVersion + 1
	if cat.Annotations == nil {
		cat.Annotations = make(map[string]string)
	}
	cat.Annotations["assethub.project-catalyst.io/source-db-id"] = spec.SourceDBID
	cat.Annotations["assethub.project-catalyst.io/published-at"] = spec.PublishedAt
	return m.client.Update(ctx, cat)
}

// Delete removes a Catalog CR. Returns nil if the CR does not exist.
func (m *K8sCatalogCRManager) Delete(ctx context.Context, name, namespace string) error {
	cat := &v1alpha1.Catalog{}
	key := types.NamespacedName{Name: name, Namespace: namespace}
	err := m.client.Get(ctx, key, cat)
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get Catalog %s for deletion: %w", name, err)
	}
	return m.client.Delete(ctx, cat)
}

// Delete removes a CatalogVersion CR. Returns nil if the CR does not exist.
func (m *K8sCRManager) Delete(ctx context.Context, name, namespace string) error {
	cv := &v1alpha1.CatalogVersion{}
	key := types.NamespacedName{Name: name, Namespace: namespace}
	err := m.client.Get(ctx, key, cv)
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get CatalogVersion %s for deletion: %w", name, err)
	}
	return m.client.Delete(ctx, cv)
}
