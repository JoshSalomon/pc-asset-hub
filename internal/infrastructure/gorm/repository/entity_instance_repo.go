package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	gormmodels "github.com/project-catalyst/pc-asset-hub/internal/infrastructure/gorm/models"
)

type EntityInstanceGormRepo struct {
	db *gorm.DB
}

func NewEntityInstanceGormRepo(db *gorm.DB) *EntityInstanceGormRepo {
	return &EntityInstanceGormRepo{db: db}
}

func (r *EntityInstanceGormRepo) Create(ctx context.Context, inst *models.EntityInstance) error {
	record := gormmodels.EntityInstanceFromModel(inst)
	result := r.db.WithContext(ctx).Create(record)
	if result.Error != nil {
		if isUniqueConstraintError(result.Error) {
			return domainerrors.NewConflict("EntityInstance", "name already exists in this scope: "+inst.Name)
		}
		return result.Error
	}
	return nil
}

func (r *EntityInstanceGormRepo) GetByID(ctx context.Context, id string) (*models.EntityInstance, error) {
	var record gormmodels.EntityInstance
	result := r.db.WithContext(ctx).Where("deleted_at IS NULL").First(&record, "id = ?", id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, domainerrors.NewNotFound("EntityInstance", id)
		}
		return nil, result.Error
	}
	return record.ToModel(), nil
}

func (r *EntityInstanceGormRepo) GetByNameAndParent(ctx context.Context, entityTypeID, catalogVersionID, parentInstanceID, name string) (*models.EntityInstance, error) {
	var record gormmodels.EntityInstance
	result := r.db.WithContext(ctx).
		Where("entity_type_id = ? AND catalog_version_id = ? AND parent_instance_id = ? AND name = ? AND deleted_at IS NULL",
			entityTypeID, catalogVersionID, parentInstanceID, name).
		First(&record)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, domainerrors.NewNotFound("EntityInstance", name)
		}
		return nil, result.Error
	}
	return record.ToModel(), nil
}

func (r *EntityInstanceGormRepo) List(ctx context.Context, entityTypeID, catalogVersionID string, params models.ListParams) ([]*models.EntityInstance, int, error) {
	var records []gormmodels.EntityInstance
	query := r.db.WithContext(ctx).Model(&gormmodels.EntityInstance{}).
		Where("entity_type_id = ? AND catalog_version_id = ? AND deleted_at IS NULL", entityTypeID, catalogVersionID)

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

	result := make([]*models.EntityInstance, len(records))
	for i := range records {
		result[i] = records[i].ToModel()
	}
	return result, int(total), nil
}

func (r *EntityInstanceGormRepo) ListByParent(ctx context.Context, parentInstanceID string, params models.ListParams) ([]*models.EntityInstance, int, error) {
	var records []gormmodels.EntityInstance
	query := r.db.WithContext(ctx).Model(&gormmodels.EntityInstance{}).
		Where("parent_instance_id = ? AND deleted_at IS NULL", parentInstanceID)

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

	result := make([]*models.EntityInstance, len(records))
	for i := range records {
		result[i] = records[i].ToModel()
	}
	return result, int(total), nil
}

func (r *EntityInstanceGormRepo) Update(ctx context.Context, inst *models.EntityInstance) error {
	record := gormmodels.EntityInstanceFromModel(inst)
	result := r.db.WithContext(ctx).Save(record)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *EntityInstanceGormRepo) SoftDelete(ctx context.Context, id string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&gormmodels.EntityInstance{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", now)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.NewNotFound("EntityInstance", id)
	}
	return nil
}
