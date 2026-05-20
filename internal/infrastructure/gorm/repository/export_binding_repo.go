package repository

import (
	"context"

	"gorm.io/gorm"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	gormmodels "github.com/project-catalyst/pc-asset-hub/internal/infrastructure/gorm/models"
)

type ExportBindingGormRepo struct {
	db *gorm.DB
}

func NewExportBindingGormRepo(db *gorm.DB) *ExportBindingGormRepo {
	return &ExportBindingGormRepo{db: db}
}

func (r *ExportBindingGormRepo) Create(ctx context.Context, binding *models.ExportBinding) error {
	record := gormmodels.ExportBindingFromModel(binding)
	result := getDB(ctx, r.db).Create(record)
	return result.Error
}

func (r *ExportBindingGormRepo) GetByID(ctx context.Context, id string) (*models.ExportBinding, error) {
	var record gormmodels.ExportBinding
	result := getDB(ctx, r.db).First(&record, "id = ?", id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, domainerrors.NewNotFound("ExportBinding", id)
		}
		return nil, result.Error
	}
	return record.ToModel(), nil
}

func (r *ExportBindingGormRepo) ListByCatalog(ctx context.Context, catalogID string) ([]*models.ExportBinding, error) {
	var records []gormmodels.ExportBinding
	result := getDB(ctx, r.db).Where("catalog_id = ?", catalogID).Order("created_at ASC").Find(&records)
	if result.Error != nil {
		return nil, result.Error
	}
	out := make([]*models.ExportBinding, len(records))
	for i := range records {
		out[i] = records[i].ToModel()
	}
	return out, nil
}

func (r *ExportBindingGormRepo) Update(ctx context.Context, binding *models.ExportBinding) error {
	record := gormmodels.ExportBindingFromModel(binding)
	result := getDB(ctx, r.db).Save(record)
	return result.Error
}

func (r *ExportBindingGormRepo) Delete(ctx context.Context, id string) error {
	result := getDB(ctx, r.db).Delete(&gormmodels.ExportBinding{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.NewNotFound("ExportBinding", id)
	}
	return nil
}

func (r *ExportBindingGormRepo) CountByCatalog(ctx context.Context, catalogID string) (int, error) {
	var count int64
	result := getDB(ctx, r.db).Model(&gormmodels.ExportBinding{}).Where("catalog_id = ?", catalogID).Count(&count)
	if result.Error != nil {
		return 0, result.Error
	}
	return int(count), nil
}

func (r *ExportBindingGormRepo) DeleteByCatalog(ctx context.Context, catalogID string) error {
	return getDB(ctx, r.db).Where("catalog_id = ?", catalogID).Delete(&gormmodels.ExportBinding{}).Error
}
