package repository

import (
	"context"

	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
)

// ContainmentEdge represents a directed edge in the containment graph.
type ContainmentEdge struct {
	SourceEntityTypeID string
	TargetEntityTypeID string
}

type AssociationRepository interface {
	Create(ctx context.Context, assoc *models.Association) error
	GetByID(ctx context.Context, id string) (*models.Association, error)
	ListByVersion(ctx context.Context, entityTypeVersionID string) ([]*models.Association, error)
	ListByTargetEntityType(ctx context.Context, targetEntityTypeID string) ([]*models.Association, error)
	Update(ctx context.Context, assoc *models.Association) error
	Delete(ctx context.Context, id string) error
	BulkCopyToVersion(ctx context.Context, fromVersionID string, toVersionID string) error
	// GetContainmentGraph returns all containment edges across all entity type versions
	// for cycle detection purposes.
	GetContainmentGraph(ctx context.Context) ([]ContainmentEdge, error)
}
