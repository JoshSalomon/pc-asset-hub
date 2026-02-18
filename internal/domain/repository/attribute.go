package repository

import (
	"context"

	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
)

type AttributeRepository interface {
	Create(ctx context.Context, attr *models.Attribute) error
	GetByID(ctx context.Context, id string) (*models.Attribute, error)
	ListByVersion(ctx context.Context, entityTypeVersionID string) ([]*models.Attribute, error)
	Update(ctx context.Context, attr *models.Attribute) error
	Delete(ctx context.Context, id string) error
	Reorder(ctx context.Context, entityTypeVersionID string, orderedIDs []string) error
	BulkCopyToVersion(ctx context.Context, fromVersionID string, toVersionID string) error
}
