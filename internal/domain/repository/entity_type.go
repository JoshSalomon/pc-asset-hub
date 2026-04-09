package repository

import (
	"context"

	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
)

type EntityTypeRepository interface {
	Create(ctx context.Context, et *models.EntityType) error
	GetByID(ctx context.Context, id string) (*models.EntityType, error)
	GetByName(ctx context.Context, name string) (*models.EntityType, error)
	List(ctx context.Context, params models.ListParams) ([]*models.EntityType, int, error)
	Update(ctx context.Context, et *models.EntityType) error
	Delete(ctx context.Context, id string) error
}

type EntityTypeVersionRepository interface {
	Create(ctx context.Context, etv *models.EntityTypeVersion) error
	GetByID(ctx context.Context, id string) (*models.EntityTypeVersion, error)
	GetByIDs(ctx context.Context, ids []string) ([]*models.EntityTypeVersion, error)
	GetByEntityTypeAndVersion(ctx context.Context, entityTypeID string, version int) (*models.EntityTypeVersion, error)
	GetLatestByEntityType(ctx context.Context, entityTypeID string) (*models.EntityTypeVersion, error)
	GetLatestByEntityTypes(ctx context.Context, entityTypeIDs []string) (map[string]*models.EntityTypeVersion, error)
	ListByEntityType(ctx context.Context, entityTypeID string) ([]*models.EntityTypeVersion, error)
}
