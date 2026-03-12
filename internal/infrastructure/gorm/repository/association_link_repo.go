package repository

import (
	"context"

	"gorm.io/gorm"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	gormmodels "github.com/project-catalyst/pc-asset-hub/internal/infrastructure/gorm/models"
)

type AssociationLinkGormRepo struct {
	db *gorm.DB
}

func NewAssociationLinkGormRepo(db *gorm.DB) *AssociationLinkGormRepo {
	return &AssociationLinkGormRepo{db: db}
}

func (r *AssociationLinkGormRepo) Create(ctx context.Context, link *models.AssociationLink) error {
	record := gormmodels.AssociationLinkFromModel(link)
	result := r.db.WithContext(ctx).Create(record)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *AssociationLinkGormRepo) GetByID(ctx context.Context, id string) (*models.AssociationLink, error) {
	var record gormmodels.AssociationLink
	result := r.db.WithContext(ctx).Where("id = ?", id).First(&record)
	if result.Error != nil {
		if result.Error.Error() == "record not found" {
			return nil, domainerrors.NewNotFound("AssociationLink", id)
		}
		return nil, result.Error
	}
	return record.ToModel(), nil
}

func (r *AssociationLinkGormRepo) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&gormmodels.AssociationLink{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.NewNotFound("AssociationLink", id)
	}
	return nil
}

func (r *AssociationLinkGormRepo) DeleteByInstance(ctx context.Context, instanceID string) error {
	result := r.db.WithContext(ctx).Where("source_instance_id = ? OR target_instance_id = ?", instanceID, instanceID).Delete(&gormmodels.AssociationLink{})
	return result.Error
}

func (r *AssociationLinkGormRepo) GetForwardRefs(ctx context.Context, sourceInstanceID string) ([]*models.AssociationLink, error) {
	var records []gormmodels.AssociationLink
	result := r.db.WithContext(ctx).Where("source_instance_id = ?", sourceInstanceID).Find(&records)
	if result.Error != nil {
		return nil, result.Error
	}
	links := make([]*models.AssociationLink, len(records))
	for i := range records {
		links[i] = records[i].ToModel()
	}
	return links, nil
}

func (r *AssociationLinkGormRepo) GetReverseRefs(ctx context.Context, targetInstanceID string) ([]*models.AssociationLink, error) {
	var records []gormmodels.AssociationLink
	result := r.db.WithContext(ctx).Where("target_instance_id = ?", targetInstanceID).Find(&records)
	if result.Error != nil {
		return nil, result.Error
	}
	links := make([]*models.AssociationLink, len(records))
	for i := range records {
		links[i] = records[i].ToModel()
	}
	return links, nil
}
