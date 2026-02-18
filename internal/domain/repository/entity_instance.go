package repository

import (
	"context"

	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
)

type EntityInstanceRepository interface {
	Create(ctx context.Context, inst *models.EntityInstance) error
	GetByID(ctx context.Context, id string) (*models.EntityInstance, error)
	GetByNameAndParent(ctx context.Context, entityTypeID, catalogVersionID, parentInstanceID, name string) (*models.EntityInstance, error)
	List(ctx context.Context, entityTypeID, catalogVersionID string, params models.ListParams) ([]*models.EntityInstance, int, error)
	ListByParent(ctx context.Context, parentInstanceID string, params models.ListParams) ([]*models.EntityInstance, int, error)
	Update(ctx context.Context, inst *models.EntityInstance) error
	SoftDelete(ctx context.Context, id string) error
}

type InstanceAttributeValueRepository interface {
	SetValues(ctx context.Context, values []*models.InstanceAttributeValue) error
	GetCurrentValues(ctx context.Context, instanceID string) ([]*models.InstanceAttributeValue, error)
	GetValuesForVersion(ctx context.Context, instanceID string, version int) ([]*models.InstanceAttributeValue, error)
}

type AssociationLinkRepository interface {
	Create(ctx context.Context, link *models.AssociationLink) error
	Delete(ctx context.Context, id string) error
	GetForwardRefs(ctx context.Context, sourceInstanceID string) ([]*models.AssociationLink, error)
	GetReverseRefs(ctx context.Context, targetInstanceID string) ([]*models.AssociationLink, error)
}
