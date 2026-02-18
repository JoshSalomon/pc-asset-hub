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
