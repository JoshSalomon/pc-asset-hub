package repository

import (
	"context"

	"gorm.io/gorm"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	gormmodels "github.com/project-catalyst/pc-asset-hub/internal/infrastructure/gorm/models"
)

type EntityTypeVersionGormRepo struct {
	db *gorm.DB
}

func NewEntityTypeVersionGormRepo(db *gorm.DB) *EntityTypeVersionGormRepo {
	return &EntityTypeVersionGormRepo{db: db}
}

func (r *EntityTypeVersionGormRepo) Create(ctx context.Context, etv *models.EntityTypeVersion) error {
	record := gormmodels.EntityTypeVersionFromModel(etv)
	result := r.db.WithContext(ctx).Create(record)
	if result.Error != nil {
		if isUniqueConstraintError(result.Error) {
			return domainerrors.NewConflict("EntityTypeVersion", "version already exists")
		}
		return result.Error
	}
	return nil
}

func (r *EntityTypeVersionGormRepo) GetByID(ctx context.Context, id string) (*models.EntityTypeVersion, error) {
	var record gormmodels.EntityTypeVersion
	result := r.db.WithContext(ctx).First(&record, "id = ?", id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, domainerrors.NewNotFound("EntityTypeVersion", id)
		}
		return nil, result.Error
	}
	return record.ToModel(), nil
}

func (r *EntityTypeVersionGormRepo) GetByEntityTypeAndVersion(ctx context.Context, entityTypeID string, version int) (*models.EntityTypeVersion, error) {
	var record gormmodels.EntityTypeVersion
	result := r.db.WithContext(ctx).Where("entity_type_id = ? AND version = ?", entityTypeID, version).First(&record)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, domainerrors.NewNotFound("EntityTypeVersion", entityTypeID)
		}
		return nil, result.Error
	}
	return record.ToModel(), nil
}

func (r *EntityTypeVersionGormRepo) GetLatestByEntityType(ctx context.Context, entityTypeID string) (*models.EntityTypeVersion, error) {
	var record gormmodels.EntityTypeVersion
	result := r.db.WithContext(ctx).Where("entity_type_id = ?", entityTypeID).Order("version DESC").First(&record)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, domainerrors.NewNotFound("EntityTypeVersion", entityTypeID)
		}
		return nil, result.Error
	}
	return record.ToModel(), nil
}

func (r *EntityTypeVersionGormRepo) ListByEntityType(ctx context.Context, entityTypeID string) ([]*models.EntityTypeVersion, error) {
	var records []gormmodels.EntityTypeVersion
	result := r.db.WithContext(ctx).Where("entity_type_id = ?", entityTypeID).Order("version ASC").Find(&records)
	if result.Error != nil {
		return nil, result.Error
	}
	versions := make([]*models.EntityTypeVersion, len(records))
	for i := range records {
		versions[i] = records[i].ToModel()
	}
	return versions, nil
}
