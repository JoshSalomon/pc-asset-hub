package repository

import (
	"context"

	"gorm.io/gorm"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	gormmodels "github.com/project-catalyst/pc-asset-hub/internal/infrastructure/gorm/models"
)

type EnumGormRepo struct {
	db *gorm.DB
}

func NewEnumGormRepo(db *gorm.DB) *EnumGormRepo {
	return &EnumGormRepo{db: db}
}

func (r *EnumGormRepo) Create(ctx context.Context, e *models.Enum) error {
	record := gormmodels.EnumFromModel(e)
	result := r.db.WithContext(ctx).Create(record)
	if result.Error != nil {
		if isUniqueConstraintError(result.Error) {
			return domainerrors.NewConflict("Enum", "name already exists: "+e.Name)
		}
		return result.Error
	}
	return nil
}

func (r *EnumGormRepo) GetByID(ctx context.Context, id string) (*models.Enum, error) {
	var record gormmodels.Enum
	result := r.db.WithContext(ctx).First(&record, "id = ?", id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, domainerrors.NewNotFound("Enum", id)
		}
		return nil, result.Error
	}
	return record.ToModel(), nil
}

func (r *EnumGormRepo) GetByName(ctx context.Context, name string) (*models.Enum, error) {
	var record gormmodels.Enum
	result := r.db.WithContext(ctx).First(&record, "name = ?", name)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, domainerrors.NewNotFound("Enum", name)
		}
		return nil, result.Error
	}
	return record.ToModel(), nil
}

func (r *EnumGormRepo) List(ctx context.Context, params models.ListParams) ([]*models.Enum, int, error) {
	var records []gormmodels.Enum
	query := r.db.WithContext(ctx).Model(&gormmodels.Enum{})

	if name, ok := params.Filters["name"]; ok {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query = query.Order("name")
	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}
	if params.Offset > 0 {
		query = query.Offset(params.Offset)
	}

	if err := query.Find(&records).Error; err != nil {
		return nil, 0, err
	}

	result := make([]*models.Enum, len(records))
	for i := range records {
		result[i] = records[i].ToModel()
	}
	return result, int(total), nil
}

func (r *EnumGormRepo) Update(ctx context.Context, e *models.Enum) error {
	record := gormmodels.EnumFromModel(e)
	result := r.db.WithContext(ctx).Save(record)
	if result.Error != nil {
		if isUniqueConstraintError(result.Error) {
			return domainerrors.NewConflict("Enum", "name already exists: "+e.Name)
		}
		return result.Error
	}
	return nil
}

func (r *EnumGormRepo) Delete(ctx context.Context, id string) error {
	// Check if any attributes reference this enum
	var count int64
	if err := r.db.WithContext(ctx).Model(&gormmodels.Attribute{}).Where("enum_id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return domainerrors.NewReferencedEnum(id, []string{"referenced by attributes"})
	}
	result := r.db.WithContext(ctx).Delete(&gormmodels.Enum{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.NewNotFound("Enum", id)
	}
	return nil
}

type EnumValueGormRepo struct {
	db *gorm.DB
}

func NewEnumValueGormRepo(db *gorm.DB) *EnumValueGormRepo {
	return &EnumValueGormRepo{db: db}
}

func (r *EnumValueGormRepo) Create(ctx context.Context, ev *models.EnumValue) error {
	record := gormmodels.EnumValueFromModel(ev)
	result := r.db.WithContext(ctx).Create(record)
	if result.Error != nil {
		if isUniqueConstraintError(result.Error) {
			return domainerrors.NewConflict("EnumValue", "value already exists in this enum: "+ev.Value)
		}
		return result.Error
	}
	return nil
}

func (r *EnumValueGormRepo) ListByEnum(ctx context.Context, enumID string) ([]*models.EnumValue, error) {
	var records []gormmodels.EnumValue
	result := r.db.WithContext(ctx).Where("enum_id = ?", enumID).Order("ordinal ASC").Find(&records)
	if result.Error != nil {
		return nil, result.Error
	}
	values := make([]*models.EnumValue, len(records))
	for i := range records {
		values[i] = records[i].ToModel()
	}
	return values, nil
}

func (r *EnumValueGormRepo) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&gormmodels.EnumValue{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.NewNotFound("EnumValue", id)
	}
	return nil
}

func (r *EnumValueGormRepo) Reorder(ctx context.Context, enumID string, orderedIDs []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i, id := range orderedIDs {
			result := tx.Model(&gormmodels.EnumValue{}).Where("id = ? AND enum_id = ?", id, enumID).Update("ordinal", i)
			if result.Error != nil {
				return result.Error
			}
		}
		return nil
	})
}
