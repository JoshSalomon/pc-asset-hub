package repository

import (
	"context"

	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
)

type ExportBindingRepository interface {
	Create(ctx context.Context, binding *models.ExportBinding) error
	GetByID(ctx context.Context, id string) (*models.ExportBinding, error)
	ListByCatalog(ctx context.Context, catalogID string) ([]*models.ExportBinding, error)
	Update(ctx context.Context, binding *models.ExportBinding) error
	Delete(ctx context.Context, id string) error
	CountByCatalog(ctx context.Context, catalogID string) (int, error)
	DeleteByCatalog(ctx context.Context, catalogID string) error
}
