package repository

import (
	"context"

	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
)

type EntityInstanceRepository interface {
	Create(ctx context.Context, inst *models.EntityInstance) error
	GetByID(ctx context.Context, id string) (*models.EntityInstance, error)
	GetByNameAndParent(ctx context.Context, entityTypeID, catalogID, parentInstanceID, name string) (*models.EntityInstance, error)
	List(ctx context.Context, entityTypeID, catalogID string, params models.ListParams) ([]*models.EntityInstance, int, error)
	ListByCatalog(ctx context.Context, catalogID string) ([]*models.EntityInstance, error)
	DeleteByCatalogID(ctx context.Context, catalogID string) error
	ListByParent(ctx context.Context, parentInstanceID string, params models.ListParams) ([]*models.EntityInstance, int, error)
	Update(ctx context.Context, inst *models.EntityInstance) error
	SoftDelete(ctx context.Context, id string) error
}

type InstanceAttributeValueRepository interface {
	SetValues(ctx context.Context, values []*models.InstanceAttributeValue) error
	GetCurrentValues(ctx context.Context, instanceID string) ([]*models.InstanceAttributeValue, error)
	GetValuesForVersion(ctx context.Context, instanceID string, version int) ([]*models.InstanceAttributeValue, error)
	DeleteByInstanceID(ctx context.Context, instanceID string) error
}

type AssociationLinkRepository interface {
	Create(ctx context.Context, link *models.AssociationLink) error
	GetByID(ctx context.Context, id string) (*models.AssociationLink, error)
	Delete(ctx context.Context, id string) error
	DeleteByInstance(ctx context.Context, instanceID string) error
	GetForwardRefs(ctx context.Context, sourceInstanceID string) ([]*models.AssociationLink, error)
	GetReverseRefs(ctx context.Context, targetInstanceID string) ([]*models.AssociationLink, error)
}
