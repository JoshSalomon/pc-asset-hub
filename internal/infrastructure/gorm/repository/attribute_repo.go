package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	gormmodels "github.com/project-catalyst/pc-asset-hub/internal/infrastructure/gorm/models"
)

type AttributeGormRepo struct {
	db *gorm.DB
}

func NewAttributeGormRepo(db *gorm.DB) *AttributeGormRepo {
	return &AttributeGormRepo{db: db}
}

func (r *AttributeGormRepo) Create(ctx context.Context, attr *models.Attribute) error {
	record := gormmodels.AttributeFromModel(attr)
	result := r.db.WithContext(ctx).Create(record)
	if result.Error != nil {
		if isUniqueConstraintError(result.Error) {
			return domainerrors.NewConflict("Attribute", "attribute name already exists in this version: "+attr.Name)
		}
		return result.Error
	}
	return nil
}

func (r *AttributeGormRepo) GetByID(ctx context.Context, id string) (*models.Attribute, error) {
	var record gormmodels.Attribute
	result := r.db.WithContext(ctx).First(&record, "id = ?", id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, domainerrors.NewNotFound("Attribute", id)
		}
		return nil, result.Error
	}
	return record.ToModel(), nil
}

func (r *AttributeGormRepo) ListByVersion(ctx context.Context, entityTypeVersionID string) ([]*models.Attribute, error) {
	var records []gormmodels.Attribute
	result := r.db.WithContext(ctx).Where("entity_type_version_id = ?", entityTypeVersionID).Order("ordinal ASC").Find(&records)
	if result.Error != nil {
		return nil, result.Error
	}
	attrs := make([]*models.Attribute, len(records))
	for i := range records {
		attrs[i] = records[i].ToModel()
	}
	return attrs, nil
}

func (r *AttributeGormRepo) Update(ctx context.Context, attr *models.Attribute) error {
	record := gormmodels.AttributeFromModel(attr)
	result := r.db.WithContext(ctx).Save(record)
	if result.Error != nil {
		if isUniqueConstraintError(result.Error) {
			return domainerrors.NewConflict("Attribute", "attribute name already exists in this version: "+attr.Name)
		}
		return result.Error
	}
	return nil
}

func (r *AttributeGormRepo) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&gormmodels.Attribute{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.NewNotFound("Attribute", id)
	}
	return nil
}

func (r *AttributeGormRepo) Reorder(ctx context.Context, entityTypeVersionID string, orderedIDs []string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i, id := range orderedIDs {
			result := tx.Model(&gormmodels.Attribute{}).Where("id = ? AND entity_type_version_id = ?", id, entityTypeVersionID).Update("ordinal", i)
			if result.Error != nil {
				return result.Error
			}
		}
		return nil
	})
}

func (r *AttributeGormRepo) BulkCopyToVersion(ctx context.Context, fromVersionID string, toVersionID string) error {
	var records []gormmodels.Attribute
	if err := r.db.WithContext(ctx).Where("entity_type_version_id = ?", fromVersionID).Find(&records).Error; err != nil {
		return err
	}
	for i := range records {
		records[i].ID = uuid.Must(uuid.NewV7()).String()
		records[i].EntityTypeVersionID = toVersionID
	}
	if len(records) > 0 {
		return r.db.WithContext(ctx).Create(&records).Error
	}
	return nil
}
