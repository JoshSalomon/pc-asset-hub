package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	gormmodels "github.com/project-catalyst/pc-asset-hub/internal/infrastructure/gorm/models"
)

type CatalogVersionGormRepo struct {
	db *gorm.DB
}

func NewCatalogVersionGormRepo(db *gorm.DB) *CatalogVersionGormRepo {
	return &CatalogVersionGormRepo{db: db}
}

func (r *CatalogVersionGormRepo) Create(ctx context.Context, cv *models.CatalogVersion) error {
	record := gormmodels.CatalogVersionFromModel(cv)
	result := r.db.WithContext(ctx).Create(record)
	if result.Error != nil {
		if isUniqueConstraintError(result.Error) {
			return domainerrors.NewConflict("CatalogVersion", "version label already exists: "+cv.VersionLabel)
		}
		return result.Error
	}
	return nil
}

func (r *CatalogVersionGormRepo) GetByID(ctx context.Context, id string) (*models.CatalogVersion, error) {
	var record gormmodels.CatalogVersion
	result := r.db.WithContext(ctx).First(&record, "id = ?", id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, domainerrors.NewNotFound("CatalogVersion", id)
		}
		return nil, result.Error
	}
	return record.ToModel(), nil
}

func (r *CatalogVersionGormRepo) GetByLabel(ctx context.Context, label string) (*models.CatalogVersion, error) {
	var record gormmodels.CatalogVersion
	result := r.db.WithContext(ctx).First(&record, "version_label = ?", label)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, domainerrors.NewNotFound("CatalogVersion", label)
		}
		return nil, result.Error
	}
	return record.ToModel(), nil
}

func (r *CatalogVersionGormRepo) List(ctx context.Context, params models.ListParams) ([]*models.CatalogVersion, int, error) {
	var records []gormmodels.CatalogVersion
	query := r.db.WithContext(ctx).Model(&gormmodels.CatalogVersion{})

	if stage, ok := params.Filters["lifecycle_stage"]; ok {
		query = query.Where("lifecycle_stage = ?", stage)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query = query.Order("created_at DESC")
	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}
	if params.Offset > 0 {
		query = query.Offset(params.Offset)
	}

	if err := query.Find(&records).Error; err != nil {
		return nil, 0, err
	}

	result := make([]*models.CatalogVersion, len(records))
	for i := range records {
		result[i] = records[i].ToModel()
	}
	return result, int(total), nil
}

func (r *CatalogVersionGormRepo) UpdateLifecycle(ctx context.Context, id string, stage models.LifecycleStage) error {
	result := r.db.WithContext(ctx).Model(&gormmodels.CatalogVersion{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"lifecycle_stage": string(stage),
			"updated_at":      time.Now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.NewNotFound("CatalogVersion", id)
	}
	return nil
}

func (r *CatalogVersionGormRepo) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&gormmodels.CatalogVersion{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.NewNotFound("CatalogVersion", id)
	}
	return nil
}

type CatalogVersionPinGormRepo struct {
	db *gorm.DB
}

func NewCatalogVersionPinGormRepo(db *gorm.DB) *CatalogVersionPinGormRepo {
	return &CatalogVersionPinGormRepo{db: db}
}

func (r *CatalogVersionPinGormRepo) Create(ctx context.Context, pin *models.CatalogVersionPin) error {
	record := gormmodels.CatalogVersionPinFromModel(pin)
	result := r.db.WithContext(ctx).Create(record)
	if result.Error != nil {
		if isUniqueConstraintError(result.Error) {
			return domainerrors.NewConflict("CatalogVersionPin", "pin already exists")
		}
		return result.Error
	}
	return nil
}

func (r *CatalogVersionPinGormRepo) ListByCatalogVersion(ctx context.Context, catalogVersionID string) ([]*models.CatalogVersionPin, error) {
	var records []gormmodels.CatalogVersionPin
	result := r.db.WithContext(ctx).Where("catalog_version_id = ?", catalogVersionID).Find(&records)
	if result.Error != nil {
		return nil, result.Error
	}
	pins := make([]*models.CatalogVersionPin, len(records))
	for i := range records {
		pins[i] = records[i].ToModel()
	}
	return pins, nil
}

func (r *CatalogVersionPinGormRepo) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&gormmodels.CatalogVersionPin{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.NewNotFound("CatalogVersionPin", id)
	}
	return nil
}

type LifecycleTransitionGormRepo struct {
	db *gorm.DB
}

func NewLifecycleTransitionGormRepo(db *gorm.DB) *LifecycleTransitionGormRepo {
	return &LifecycleTransitionGormRepo{db: db}
}

func (r *LifecycleTransitionGormRepo) Create(ctx context.Context, lt *models.LifecycleTransition) error {
	record := gormmodels.LifecycleTransitionFromModel(lt)
	result := r.db.WithContext(ctx).Create(record)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *LifecycleTransitionGormRepo) ListByCatalogVersion(ctx context.Context, catalogVersionID string) ([]*models.LifecycleTransition, error) {
	var records []gormmodels.LifecycleTransition
	result := r.db.WithContext(ctx).Where("catalog_version_id = ?", catalogVersionID).Order("performed_at ASC").Find(&records)
	if result.Error != nil {
		return nil, result.Error
	}
	transitions := make([]*models.LifecycleTransition, len(records))
	for i := range records {
		transitions[i] = records[i].ToModel()
	}
	return transitions, nil
}
