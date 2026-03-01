package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository"
	gormmodels "github.com/project-catalyst/pc-asset-hub/internal/infrastructure/gorm/models"
)

type AssociationGormRepo struct {
	db *gorm.DB
}

func NewAssociationGormRepo(db *gorm.DB) *AssociationGormRepo {
	return &AssociationGormRepo{db: db}
}

func (r *AssociationGormRepo) Create(ctx context.Context, assoc *models.Association) error {
	record := gormmodels.AssociationFromModel(assoc)
	result := r.db.WithContext(ctx).Create(record)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *AssociationGormRepo) GetByID(ctx context.Context, id string) (*models.Association, error) {
	var record gormmodels.Association
	result := r.db.WithContext(ctx).First(&record, "id = ?", id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, domainerrors.NewNotFound("Association", id)
		}
		return nil, result.Error
	}
	return record.ToModel(), nil
}

func (r *AssociationGormRepo) ListByVersion(ctx context.Context, entityTypeVersionID string) ([]*models.Association, error) {
	var records []gormmodels.Association
	result := r.db.WithContext(ctx).Where("entity_type_version_id = ?", entityTypeVersionID).Find(&records)
	if result.Error != nil {
		return nil, result.Error
	}
	assocs := make([]*models.Association, len(records))
	for i := range records {
		assocs[i] = records[i].ToModel()
	}
	return assocs, nil
}

func (r *AssociationGormRepo) ListByTargetEntityType(ctx context.Context, targetEntityTypeID string) ([]*models.Association, error) {
	var records []gormmodels.Association
	result := r.db.WithContext(ctx).Where("target_entity_type_id = ?", targetEntityTypeID).Find(&records)
	if result.Error != nil {
		return nil, result.Error
	}
	assocs := make([]*models.Association, len(records))
	for i := range records {
		assocs[i] = records[i].ToModel()
	}
	return assocs, nil
}

func (r *AssociationGormRepo) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&gormmodels.Association{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.NewNotFound("Association", id)
	}
	return nil
}

func (r *AssociationGormRepo) BulkCopyToVersion(ctx context.Context, fromVersionID string, toVersionID string) error {
	var records []gormmodels.Association
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

func (r *AssociationGormRepo) GetContainmentGraph(ctx context.Context) ([]repository.ContainmentEdge, error) {
	type row struct {
		EntityTypeID       string
		TargetEntityTypeID string
	}
	var rows []row
	// Join associations with entity_type_versions to get the source entity type ID
	result := r.db.WithContext(ctx).
		Table("associations").
		Select("entity_type_versions.entity_type_id, associations.target_entity_type_id").
		Joins("JOIN entity_type_versions ON entity_type_versions.id = associations.entity_type_version_id").
		Where("associations.type = ?", "containment").
		Find(&rows)
	if result.Error != nil {
		return nil, result.Error
	}
	edges := make([]repository.ContainmentEdge, len(rows))
	for i, r := range rows {
		edges[i] = repository.ContainmentEdge{
			SourceEntityTypeID: r.EntityTypeID,
			TargetEntityTypeID: r.TargetEntityTypeID,
		}
	}
	return edges, nil
}
