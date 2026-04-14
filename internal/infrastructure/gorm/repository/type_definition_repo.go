package repository

import (
	"context"

	"gorm.io/gorm"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	gormmodels "github.com/project-catalyst/pc-asset-hub/internal/infrastructure/gorm/models"
)

type TypeDefinitionGormRepo struct {
	db *gorm.DB
}

func NewTypeDefinitionGormRepo(db *gorm.DB) *TypeDefinitionGormRepo {
	return &TypeDefinitionGormRepo{db: db}
}

func (r *TypeDefinitionGormRepo) Create(ctx context.Context, td *models.TypeDefinition) error {
	record := gormmodels.TypeDefinitionFromModel(td)
	result := r.db.WithContext(ctx).Create(record)
	if result.Error != nil {
		if isUniqueConstraintError(result.Error) {
			return domainerrors.NewConflict("TypeDefinition", "name already exists: "+td.Name)
		}
		return result.Error
	}
	return nil
}

func (r *TypeDefinitionGormRepo) GetByID(ctx context.Context, id string) (*models.TypeDefinition, error) {
	var record gormmodels.TypeDefinition
	result := r.db.WithContext(ctx).First(&record, "id = ?", id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, domainerrors.NewNotFound("TypeDefinition", id)
		}
		return nil, result.Error
	}
	return record.ToModel(), nil
}

func (r *TypeDefinitionGormRepo) GetByName(ctx context.Context, name string) (*models.TypeDefinition, error) {
	var record gormmodels.TypeDefinition
	result := r.db.WithContext(ctx).First(&record, "name = ?", name)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, domainerrors.NewNotFound("TypeDefinition", name)
		}
		return nil, result.Error
	}
	return record.ToModel(), nil
}

func (r *TypeDefinitionGormRepo) List(ctx context.Context, params models.ListParams) ([]*models.TypeDefinition, int, error) {
	var records []gormmodels.TypeDefinition
	query := r.db.WithContext(ctx).Model(&gormmodels.TypeDefinition{})

	if name, ok := params.Filters["name"]; ok {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if baseType, ok := params.Filters["base_type"]; ok {
		query = query.Where("base_type = ?", baseType)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query = query.Order("system DESC, name ASC")
	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}
	if params.Offset > 0 {
		query = query.Offset(params.Offset)
	}

	if err := query.Find(&records).Error; err != nil {
		return nil, 0, err
	}

	result := make([]*models.TypeDefinition, len(records))
	for i := range records {
		result[i] = records[i].ToModel()
	}
	return result, int(total), nil
}

func (r *TypeDefinitionGormRepo) Update(ctx context.Context, td *models.TypeDefinition) error {
	record := gormmodels.TypeDefinitionFromModel(td)
	result := r.db.WithContext(ctx).Save(record)
	if result.Error != nil {
		if isUniqueConstraintError(result.Error) {
			return domainerrors.NewConflict("TypeDefinition", "name already exists: "+td.Name)
		}
		return result.Error
	}
	return nil
}

func (r *TypeDefinitionGormRepo) Delete(ctx context.Context, id string) error {
	// Check if any attributes reference versions of this type definition
	var count int64
	if err := r.db.WithContext(ctx).Model(&gormmodels.Attribute{}).
		Where("type_definition_version_id IN (?)",
			r.db.Model(&gormmodels.TypeDefinitionVersion{}).Select("id").Where("type_definition_id = ?", id)).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return domainerrors.NewValidation("type definition is referenced by attributes and cannot be deleted")
	}

	result := r.db.WithContext(ctx).Delete(&gormmodels.TypeDefinition{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.NewNotFound("TypeDefinition", id)
	}
	return nil
}

// TypeDefinitionVersionGormRepo

type TypeDefinitionVersionGormRepo struct {
	db *gorm.DB
}

func NewTypeDefinitionVersionGormRepo(db *gorm.DB) *TypeDefinitionVersionGormRepo {
	return &TypeDefinitionVersionGormRepo{db: db}
}

func (r *TypeDefinitionVersionGormRepo) Create(ctx context.Context, tdv *models.TypeDefinitionVersion) error {
	record := gormmodels.TypeDefinitionVersionFromModel(tdv)
	result := r.db.WithContext(ctx).Create(record)
	if result.Error != nil {
		if isUniqueConstraintError(result.Error) {
			return domainerrors.NewConflict("TypeDefinitionVersion", "version already exists")
		}
		return result.Error
	}
	return nil
}

func (r *TypeDefinitionVersionGormRepo) GetByID(ctx context.Context, id string) (*models.TypeDefinitionVersion, error) {
	var record gormmodels.TypeDefinitionVersion
	result := r.db.WithContext(ctx).First(&record, "id = ?", id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, domainerrors.NewNotFound("TypeDefinitionVersion", id)
		}
		return nil, result.Error
	}
	return record.ToModel(), nil
}

func (r *TypeDefinitionVersionGormRepo) GetByIDs(ctx context.Context, ids []string) ([]*models.TypeDefinitionVersion, error) {
	if len(ids) == 0 {
		return []*models.TypeDefinitionVersion{}, nil
	}
	var records []gormmodels.TypeDefinitionVersion
	result := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&records)
	if result.Error != nil {
		return nil, result.Error
	}
	versions := make([]*models.TypeDefinitionVersion, len(records))
	for i := range records {
		versions[i] = records[i].ToModel()
	}
	return versions, nil
}

func (r *TypeDefinitionVersionGormRepo) GetLatestByTypeDefinition(ctx context.Context, typeDefID string) (*models.TypeDefinitionVersion, error) {
	var record gormmodels.TypeDefinitionVersion
	result := r.db.WithContext(ctx).Where("type_definition_id = ?", typeDefID).Order("version_number DESC").First(&record)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, domainerrors.NewNotFound("TypeDefinitionVersion", typeDefID)
		}
		return nil, result.Error
	}
	return record.ToModel(), nil
}

func (r *TypeDefinitionVersionGormRepo) GetLatestByTypeDefinitions(ctx context.Context, typeDefIDs []string) (map[string]*models.TypeDefinitionVersion, error) {
	if len(typeDefIDs) == 0 {
		return map[string]*models.TypeDefinitionVersion{}, nil
	}
	// Subquery: max version_number per type_definition_id
	var records []gormmodels.TypeDefinitionVersion
	subq := r.db.Model(&gormmodels.TypeDefinitionVersion{}).
		Select("type_definition_id, MAX(version_number) as max_v").
		Where("type_definition_id IN ?", typeDefIDs).
		Group("type_definition_id")
	result := r.db.WithContext(ctx).
		Where("(type_definition_id, version_number) IN (?)",
			r.db.Table("(?) as sub", subq).Select("sub.type_definition_id, sub.max_v")).
		Find(&records)
	if result.Error != nil {
		// Fallback for SQLite which doesn't support tuple IN subquery
		records = nil
		for _, tdID := range typeDefIDs {
			var rec gormmodels.TypeDefinitionVersion
			err := r.db.WithContext(ctx).Where("type_definition_id = ?", tdID).Order("version_number DESC").First(&rec).Error
			if err != nil {
				continue
			}
			records = append(records, rec)
		}
	}
	m := make(map[string]*models.TypeDefinitionVersion, len(records))
	for i := range records {
		m[records[i].TypeDefinitionID] = records[i].ToModel()
	}
	return m, nil
}

func (r *TypeDefinitionVersionGormRepo) ListByTypeDefinition(ctx context.Context, typeDefID string) ([]*models.TypeDefinitionVersion, error) {
	var records []gormmodels.TypeDefinitionVersion
	result := r.db.WithContext(ctx).Where("type_definition_id = ?", typeDefID).Order("version_number ASC").Find(&records)
	if result.Error != nil {
		return nil, result.Error
	}
	versions := make([]*models.TypeDefinitionVersion, len(records))
	for i := range records {
		versions[i] = records[i].ToModel()
	}
	return versions, nil
}

// CatalogVersionTypePinGormRepo

type CatalogVersionTypePinGormRepo struct {
	db *gorm.DB
}

func NewCatalogVersionTypePinGormRepo(db *gorm.DB) *CatalogVersionTypePinGormRepo {
	return &CatalogVersionTypePinGormRepo{db: db}
}

func (r *CatalogVersionTypePinGormRepo) Create(ctx context.Context, pin *models.CatalogVersionTypePin) error {
	record := gormmodels.CatalogVersionTypePinFromModel(pin)
	result := r.db.WithContext(ctx).Create(record)
	if result.Error != nil {
		if isUniqueConstraintError(result.Error) {
			return domainerrors.NewConflict("CatalogVersionTypePin", "type definition version already pinned")
		}
		return result.Error
	}
	return nil
}

func (r *CatalogVersionTypePinGormRepo) GetByID(ctx context.Context, id string) (*models.CatalogVersionTypePin, error) {
	var record gormmodels.CatalogVersionTypePin
	result := r.db.WithContext(ctx).First(&record, "id = ?", id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, domainerrors.NewNotFound("CatalogVersionTypePin", id)
		}
		return nil, result.Error
	}
	return record.ToModel(), nil
}

func (r *CatalogVersionTypePinGormRepo) ListByCatalogVersion(ctx context.Context, catalogVersionID string) ([]*models.CatalogVersionTypePin, error) {
	var records []gormmodels.CatalogVersionTypePin
	result := r.db.WithContext(ctx).Where("catalog_version_id = ?", catalogVersionID).Find(&records)
	if result.Error != nil {
		return nil, result.Error
	}
	pins := make([]*models.CatalogVersionTypePin, len(records))
	for i := range records {
		pins[i] = records[i].ToModel()
	}
	return pins, nil
}

func (r *CatalogVersionTypePinGormRepo) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&gormmodels.CatalogVersionTypePin{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.NewNotFound("CatalogVersionTypePin", id)
	}
	return nil
}
