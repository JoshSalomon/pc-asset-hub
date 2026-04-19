package repository

import (
	"context"

	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
)

type TypeDefinitionRepository interface {
	Create(ctx context.Context, td *models.TypeDefinition) error
	GetByID(ctx context.Context, id string) (*models.TypeDefinition, error)
	GetByName(ctx context.Context, name string) (*models.TypeDefinition, error)
	List(ctx context.Context, params models.ListParams) ([]*models.TypeDefinition, int, error)
	Update(ctx context.Context, td *models.TypeDefinition) error
	Delete(ctx context.Context, id string) error
}

type TypeDefinitionVersionRepository interface {
	Create(ctx context.Context, tdv *models.TypeDefinitionVersion) error
	GetByID(ctx context.Context, id string) (*models.TypeDefinitionVersion, error)
	GetByIDs(ctx context.Context, ids []string) ([]*models.TypeDefinitionVersion, error)
	GetLatestByTypeDefinition(ctx context.Context, typeDefID string) (*models.TypeDefinitionVersion, error)
	GetLatestByTypeDefinitions(ctx context.Context, typeDefIDs []string) (map[string]*models.TypeDefinitionVersion, error)
	ListByTypeDefinition(ctx context.Context, typeDefID string) ([]*models.TypeDefinitionVersion, error)
	GetByVersion(ctx context.Context, typeDefID string, versionNumber int) (*models.TypeDefinitionVersion, error)
}

type CatalogVersionTypePinRepository interface {
	Create(ctx context.Context, pin *models.CatalogVersionTypePin) error
	GetByID(ctx context.Context, id string) (*models.CatalogVersionTypePin, error)
	ListByCatalogVersion(ctx context.Context, catalogVersionID string) ([]*models.CatalogVersionTypePin, error)
	Delete(ctx context.Context, id string) error
}
