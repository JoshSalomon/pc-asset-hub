package repository

import (
	"context"
	"time"

	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
)

type CatalogRepository interface {
	Create(ctx context.Context, catalog *models.Catalog) error
	GetByName(ctx context.Context, name string) (*models.Catalog, error)
	GetByID(ctx context.Context, id string) (*models.Catalog, error)
	List(ctx context.Context, params models.ListParams) ([]*models.Catalog, int, error)
	Delete(ctx context.Context, id string) error
	UpdateValidationStatus(ctx context.Context, id string, status models.ValidationStatus) error
	UpdatePublished(ctx context.Context, id string, published bool, publishedAt *time.Time) error
	UpdateName(ctx context.Context, id string, newName string) error
	ListByCatalogVersionID(ctx context.Context, catalogVersionID string) ([]*models.Catalog, error)
}
