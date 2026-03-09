package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	gormmodels "github.com/project-catalyst/pc-asset-hub/internal/infrastructure/gorm/models"
)

type CatalogGormRepo struct {
	db *gorm.DB
}

func NewCatalogGormRepo(db *gorm.DB) *CatalogGormRepo {
	return &CatalogGormRepo{db: db}
}

func (r *CatalogGormRepo) Create(ctx context.Context, catalog *models.Catalog) error {
	record := gormmodels.CatalogFromModel(catalog)
	result := r.db.WithContext(ctx).Create(record)
	if result.Error != nil {
		if isUniqueConstraintError(result.Error) {
			return domainerrors.NewConflict("Catalog", "name already exists: "+catalog.Name)
		}
		return result.Error
	}
	return nil
}

func (r *CatalogGormRepo) GetByName(ctx context.Context, name string) (*models.Catalog, error) {
	var record gormmodels.Catalog
	result := r.db.WithContext(ctx).Where("name = ?", name).First(&record)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, domainerrors.NewNotFound("Catalog", name)
		}
		return nil, result.Error
	}
	return record.ToModel(), nil
}

func (r *CatalogGormRepo) GetByID(ctx context.Context, id string) (*models.Catalog, error) {
	var record gormmodels.Catalog
	result := r.db.WithContext(ctx).First(&record, "id = ?", id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, domainerrors.NewNotFound("Catalog", id)
		}
		return nil, result.Error
	}
	return record.ToModel(), nil
}

func (r *CatalogGormRepo) List(ctx context.Context, params models.ListParams) ([]*models.Catalog, int, error) {
	var records []gormmodels.Catalog
	query := r.db.WithContext(ctx).Model(&gormmodels.Catalog{})

	if cvID, ok := params.Filters["catalog_version_id"]; ok {
		query = query.Where("catalog_version_id = ?", cvID)
	}
	if status, ok := params.Filters["validation_status"]; ok {
		query = query.Where("validation_status = ?", status)
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

	result := make([]*models.Catalog, len(records))
	for i := range records {
		result[i] = records[i].ToModel()
	}
	return result, int(total), nil
}

func (r *CatalogGormRepo) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&gormmodels.Catalog{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.NewNotFound("Catalog", id)
	}
	return nil
}

func (r *CatalogGormRepo) UpdateValidationStatus(ctx context.Context, id string, status models.ValidationStatus) error {
	result := r.db.WithContext(ctx).Model(&gormmodels.Catalog{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"validation_status": string(status),
			"updated_at":        time.Now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.NewNotFound("Catalog", id)
	}
	return nil
}
