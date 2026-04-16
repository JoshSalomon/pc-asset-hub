package repository

import (
	"context"

	"gorm.io/gorm"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	gormmodels "github.com/project-catalyst/pc-asset-hub/internal/infrastructure/gorm/models"
)

type InstanceAttributeValueGormRepo struct {
	db *gorm.DB
}

func NewInstanceAttributeValueGormRepo(db *gorm.DB) *InstanceAttributeValueGormRepo {
	return &InstanceAttributeValueGormRepo{db: db}
}

func (r *InstanceAttributeValueGormRepo) SetValues(ctx context.Context, values []*models.InstanceAttributeValue) error {
	if len(values) == 0 {
		return nil
	}
	records := make([]gormmodels.InstanceAttributeValue, len(values))
	for i, v := range values {
		records[i] = *gormmodels.InstanceAttributeValueFromModel(v)
	}
	result := getDB(ctx, r.db).Create(&records)
	if result.Error != nil {
		if isUniqueConstraintError(result.Error) {
			return domainerrors.NewConflict("InstanceAttributeValue", "duplicate attribute value for this instance version")
		}
		return result.Error
	}
	return nil
}

func (r *InstanceAttributeValueGormRepo) GetValuesForVersion(ctx context.Context, instanceID string, version int) ([]*models.InstanceAttributeValue, error) {
	var records []gormmodels.InstanceAttributeValue
	result := getDB(ctx, r.db).
		Where("instance_id = ? AND instance_version = ?", instanceID, version).
		Find(&records)
	if result.Error != nil {
		return nil, result.Error
	}
	values := make([]*models.InstanceAttributeValue, len(records))
	for i := range records {
		values[i] = records[i].ToModel()
	}
	return values, nil
}

func (r *InstanceAttributeValueGormRepo) DeleteByInstanceID(ctx context.Context, instanceID string) error {
	return getDB(ctx, r.db).Where("instance_id = ?", instanceID).Delete(&gormmodels.InstanceAttributeValue{}).Error
}
