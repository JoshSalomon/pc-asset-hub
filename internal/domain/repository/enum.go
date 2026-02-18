package repository

import (
	"context"

	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
)

type EnumRepository interface {
	Create(ctx context.Context, e *models.Enum) error
	GetByID(ctx context.Context, id string) (*models.Enum, error)
	GetByName(ctx context.Context, name string) (*models.Enum, error)
	List(ctx context.Context, params models.ListParams) ([]*models.Enum, int, error)
	Update(ctx context.Context, e *models.Enum) error
	Delete(ctx context.Context, id string) error
}

type EnumValueRepository interface {
	Create(ctx context.Context, ev *models.EnumValue) error
	ListByEnum(ctx context.Context, enumID string) ([]*models.EnumValue, error)
	Delete(ctx context.Context, id string) error
	Reorder(ctx context.Context, enumID string, orderedIDs []string) error
}
