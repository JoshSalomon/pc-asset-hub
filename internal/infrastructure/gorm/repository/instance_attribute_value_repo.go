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
	result := r.db.WithContext(ctx).Create(&records)
	if result.Error != nil {
		if isUniqueConstraintError(result.Error) {
			return domainerrors.NewConflict("InstanceAttributeValue", "duplicate attribute value for this instance version")
		}
		return result.Error
	}
	return nil
}

func (r *InstanceAttributeValueGormRepo) GetCurrentValues(ctx context.Context, instanceID string) ([]*models.InstanceAttributeValue, error) {
	// Get the max version for this instance, then get values for that version
	var maxVersion int
	err := r.db.WithContext(ctx).Model(&gormmodels.InstanceAttributeValue{}).
		Where("instance_id = ?", instanceID).
		Select("COALESCE(MAX(instance_version), 0)").
		Scan(&maxVersion).Error
	if err != nil {
		return nil, err
	}
	if maxVersion == 0 {
		return nil, nil
	}
	return r.GetValuesForVersion(ctx, instanceID, maxVersion)
}

func (r *InstanceAttributeValueGormRepo) GetValuesForVersion(ctx context.Context, instanceID string, version int) ([]*models.InstanceAttributeValue, error) {
	var records []gormmodels.InstanceAttributeValue
	result := r.db.WithContext(ctx).
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
