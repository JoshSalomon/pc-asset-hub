package repository

import (
	"context"

	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
)

type CatalogVersionRepository interface {
	Create(ctx context.Context, cv *models.CatalogVersion) error
	GetByID(ctx context.Context, id string) (*models.CatalogVersion, error)
	GetByLabel(ctx context.Context, label string) (*models.CatalogVersion, error)
	List(ctx context.Context, params models.ListParams) ([]*models.CatalogVersion, int, error)
	Update(ctx context.Context, cv *models.CatalogVersion) error
	UpdateLifecycle(ctx context.Context, id string, stage models.LifecycleStage) error
	Delete(ctx context.Context, id string) error
}

type CatalogVersionPinRepository interface {
	Create(ctx context.Context, pin *models.CatalogVersionPin) error
	GetByID(ctx context.Context, id string) (*models.CatalogVersionPin, error)
	Update(ctx context.Context, pin *models.CatalogVersionPin) error
	ListByCatalogVersion(ctx context.Context, catalogVersionID string) ([]*models.CatalogVersionPin, error)
	ListByEntityTypeVersionIDs(ctx context.Context, etvIDs []string) ([]*models.CatalogVersionPin, error)
	Delete(ctx context.Context, id string) error
}

type LifecycleTransitionRepository interface {
	Create(ctx context.Context, lt *models.LifecycleTransition) error
	ListByCatalogVersion(ctx context.Context, catalogVersionID string) ([]*models.LifecycleTransition, error)
}
