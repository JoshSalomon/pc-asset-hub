package repository

import (
	"context"
	"fmt"
	"strconv"
	"strings"
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
	result := getDB(ctx, r.db).Create(record)
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
	result := getDB(ctx, r.db).Where("deleted_at IS NULL").First(&record, "id = ?", id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, domainerrors.NewNotFound("EntityInstance", id)
		}
		return nil, result.Error
	}
	return record.ToModel(), nil
}

func (r *EntityInstanceGormRepo) GetByNameAndParent(ctx context.Context, entityTypeID, catalogID, parentInstanceID, name string) (*models.EntityInstance, error) {
	var record gormmodels.EntityInstance
	result := getDB(ctx, r.db).
		Where("entity_type_id = ? AND catalog_id = ? AND parent_instance_id = ? AND name = ? AND deleted_at IS NULL",
			entityTypeID, catalogID, parentInstanceID, name).
		First(&record)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, domainerrors.NewNotFound("EntityInstance", name)
		}
		return nil, result.Error
	}
	return record.ToModel(), nil
}

// applyAttrFilters adds JOIN and WHERE clauses for attribute-based filtering.
// Filter keys are attribute IDs (translated from names by the service layer).
// Suffixes: .min for number minimum, .max for number maximum.
// Plain keys: case-insensitive contains for strings, exact match for enums.
func applyAttrFilters(query *gorm.DB, filters map[string]string) (*gorm.DB, error) {
	joinIdx := 0
	for key, val := range filters {
		alias := fmt.Sprintf("iav%d", joinIdx)
		joinIdx++

		if strings.HasSuffix(key, ".min") {
			attrID := strings.TrimSuffix(key, ".min")
			numVal, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return nil, domainerrors.NewValidation(fmt.Sprintf("invalid number for filter %s: %s", key, val))
			}
			query = query.Joins(
				fmt.Sprintf("JOIN instance_attribute_values AS %s ON %s.instance_id = entity_instances.id AND %s.attribute_id = ?", alias, alias, alias), attrID).
				Where(fmt.Sprintf("%s.value_number >= ?", alias), numVal)
		} else if strings.HasSuffix(key, ".max") {
			attrID := strings.TrimSuffix(key, ".max")
			numVal, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return nil, domainerrors.NewValidation(fmt.Sprintf("invalid number for filter %s: %s", key, val))
			}
			query = query.Joins(
				fmt.Sprintf("JOIN instance_attribute_values AS %s ON %s.instance_id = entity_instances.id AND %s.attribute_id = ?", alias, alias, alias), attrID).
				Where(fmt.Sprintf("%s.value_number <= ?", alias), numVal)
		} else {
			query = query.Joins(
				fmt.Sprintf("JOIN instance_attribute_values AS %s ON %s.instance_id = entity_instances.id AND %s.attribute_id = ?", alias, alias, alias), key).
				Where(fmt.Sprintf("(LOWER(%s.value_string) LIKE ? OR %s.value_enum = ?)", alias, alias), "%"+strings.ToLower(val)+"%", val)
		}
	}
	return query, nil
}

func (r *EntityInstanceGormRepo) List(ctx context.Context, entityTypeID, catalogID string, params models.ListParams) ([]*models.EntityInstance, int, error) {
	base := getDB(ctx, r.db).Table("entity_instances").
		Where("entity_instances.entity_type_id = ? AND entity_instances.catalog_id = ? AND entity_instances.deleted_at IS NULL", entityTypeID, catalogID)

	// Apply attribute filters
	if len(params.Filters) > 0 {
		var err error
		base, err = applyAttrFilters(base, params.Filters)
		if err != nil {
			return nil, 0, err
		}
	}

	// Count (uses same base query with filters applied)
	var total int64
	if err := base.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query := base

	// Sorting
	if params.SortBy != "" {
		if err := validateSortBy(params.SortBy); err != nil {
			return nil, 0, err
		}
		order := "entity_instances." + params.SortBy
		if params.SortDesc {
			order += " DESC"
		}
		query = query.Order(order)
	} else {
		query = query.Order("entity_instances.name")
	}

	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}
	if params.Offset > 0 {
		query = query.Offset(params.Offset)
	}

	// Select only entity_instances columns to avoid JOIN columns interfering
	query = query.Select("entity_instances.*")

	var records []gormmodels.EntityInstance
	if err := query.Find(&records).Error; err != nil {
		return nil, 0, err
	}

	result := make([]*models.EntityInstance, len(records))
	for i := range records {
		result[i] = records[i].ToModel()
	}
	return result, int(total), nil
}

func (r *EntityInstanceGormRepo) ListByCatalog(ctx context.Context, catalogID string) ([]*models.EntityInstance, error) {
	var records []gormmodels.EntityInstance
	if err := getDB(ctx, r.db).
		Where("catalog_id = ? AND deleted_at IS NULL", catalogID).
		Order("name").
		Find(&records).Error; err != nil {
		return nil, err
	}
	result := make([]*models.EntityInstance, len(records))
	for i := range records {
		result[i] = records[i].ToModel()
	}
	return result, nil
}

func (r *EntityInstanceGormRepo) ListByParent(ctx context.Context, parentInstanceID string, params models.ListParams) ([]*models.EntityInstance, int, error) {
	var records []gormmodels.EntityInstance
	query := getDB(ctx, r.db).Model(&gormmodels.EntityInstance{}).
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
	result := getDB(ctx, r.db).Save(record)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *EntityInstanceGormRepo) DeleteByCatalogID(ctx context.Context, catalogID string) error {
	now := time.Now()
	result := getDB(ctx, r.db).Model(&gormmodels.EntityInstance{}).
		Where("catalog_id = ? AND deleted_at IS NULL", catalogID).
		Update("deleted_at", now)
	return result.Error
}

func (r *EntityInstanceGormRepo) SoftDelete(ctx context.Context, id string) error {
	now := time.Now()
	result := getDB(ctx, r.db).Model(&gormmodels.EntityInstance{}).
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
