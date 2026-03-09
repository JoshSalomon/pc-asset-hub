package repository

import (
	"context"

	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
)

type CatalogRepository interface {
	Create(ctx context.Context, catalog *models.Catalog) error
	GetByName(ctx context.Context, name string) (*models.Catalog, error)
	GetByID(ctx context.Context, id string) (*models.Catalog, error)
	List(ctx context.Context, params models.ListParams) ([]*models.Catalog, int, error)
	Delete(ctx context.Context, id string) error
	UpdateValidationStatus(ctx context.Context, id string, status models.ValidationStatus) error
}
