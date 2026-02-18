package repository

import (
	"context"

	"gorm.io/gorm"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	gormmodels "github.com/project-catalyst/pc-asset-hub/internal/infrastructure/gorm/models"
)

type EntityTypeGormRepo struct {
	db *gorm.DB
}

func NewEntityTypeGormRepo(db *gorm.DB) *EntityTypeGormRepo {
	return &EntityTypeGormRepo{db: db}
}

func (r *EntityTypeGormRepo) Create(ctx context.Context, et *models.EntityType) error {
	record := gormmodels.EntityTypeFromModel(et)
	result := r.db.WithContext(ctx).Create(record)
	if result.Error != nil {
		if isUniqueConstraintError(result.Error) {
			return domainerrors.NewConflict("EntityType", "name already exists: "+et.Name)
		}
		return result.Error
	}
	return nil
}

func (r *EntityTypeGormRepo) GetByID(ctx context.Context, id string) (*models.EntityType, error) {
	var record gormmodels.EntityType
	result := r.db.WithContext(ctx).First(&record, "id = ?", id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, domainerrors.NewNotFound("EntityType", id)
		}
		return nil, result.Error
	}
	return record.ToModel(), nil
}

func (r *EntityTypeGormRepo) GetByName(ctx context.Context, name string) (*models.EntityType, error) {
	var record gormmodels.EntityType
	result := r.db.WithContext(ctx).First(&record, "name = ?", name)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, domainerrors.NewNotFound("EntityType", name)
		}
		return nil, result.Error
	}
	return record.ToModel(), nil
}

func (r *EntityTypeGormRepo) List(ctx context.Context, params models.ListParams) ([]*models.EntityType, int, error) {
	var records []gormmodels.EntityType
	query := r.db.WithContext(ctx).Model(&gormmodels.EntityType{})

	if name, ok := params.Filters["name"]; ok {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := validateSortBy(params.SortBy); err != nil {
		return nil, 0, err
	}
	if params.SortBy != "" {
		order := params.SortBy
		if params.SortDesc {
			order += " DESC"
		}
		query = query.Order(order)
	} else {
		query = query.Order("name")
	}

	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}
	if params.Offset > 0 {
		query = query.Offset(params.Offset)
	}

	if err := query.Find(&records).Error; err != nil {
		return nil, 0, err
	}

	result := make([]*models.EntityType, len(records))
	for i := range records {
		result[i] = records[i].ToModel()
	}
	return result, int(total), nil
}

func (r *EntityTypeGormRepo) Update(ctx context.Context, et *models.EntityType) error {
	record := gormmodels.EntityTypeFromModel(et)
	result := r.db.WithContext(ctx).Save(record)
	if result.Error != nil {
		if isUniqueConstraintError(result.Error) {
			return domainerrors.NewConflict("EntityType", "name already exists: "+et.Name)
		}
		return result.Error
	}
	return nil
}

func (r *EntityTypeGormRepo) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Select("Versions").Delete(&gormmodels.EntityType{ID: id})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.NewNotFound("EntityType", id)
	}
	return nil
}
