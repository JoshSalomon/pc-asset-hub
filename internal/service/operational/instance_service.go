package operational

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository"
)

// InstanceService manages entity instances within catalogs.
type InstanceService struct {
	instRepo    repository.EntityInstanceRepository
	iavRepo     repository.InstanceAttributeValueRepository
	catalogRepo repository.CatalogRepository
	cvRepo      repository.CatalogVersionRepository
	pinRepo     repository.CatalogVersionPinRepository
	attrRepo    repository.AttributeRepository
	etvRepo     repository.EntityTypeVersionRepository
	etRepo      repository.EntityTypeRepository
	enumValRepo repository.EnumValueRepository
}

func NewInstanceService(
	instRepo repository.EntityInstanceRepository,
	iavRepo repository.InstanceAttributeValueRepository,
	catalogRepo repository.CatalogRepository,
	cvRepo repository.CatalogVersionRepository,
	pinRepo repository.CatalogVersionPinRepository,
	attrRepo repository.AttributeRepository,
	etvRepo repository.EntityTypeVersionRepository,
	etRepo repository.EntityTypeRepository,
	enumValRepo repository.EnumValueRepository,
) *InstanceService {
	return &InstanceService{
		instRepo:    instRepo,
		iavRepo:     iavRepo,
		catalogRepo: catalogRepo,
		cvRepo:      cvRepo,
		pinRepo:     pinRepo,
		attrRepo:    attrRepo,
		etvRepo:     etvRepo,
		etRepo:      etRepo,
		enumValRepo: enumValRepo,
	}
}

// resolveEntityType resolves catalog name + entity type name to a catalog and pinned entity type version.
func (s *InstanceService) resolveEntityType(ctx context.Context, catalogName, entityTypeName string) (*models.Catalog, *models.EntityTypeVersion, error) {
	catalog, err := s.catalogRepo.GetByName(ctx, catalogName)
	if err != nil {
		return nil, nil, err
	}

	et, err := s.etRepo.GetByName(ctx, entityTypeName)
	if err != nil {
		return nil, nil, domainerrors.NewNotFound("EntityType", entityTypeName)
	}

	pins, err := s.pinRepo.ListByCatalogVersion(ctx, catalog.CatalogVersionID)
	if err != nil {
		return nil, nil, err
	}

	for _, pin := range pins {
		etv, err := s.etvRepo.GetByID(ctx, pin.EntityTypeVersionID)
		if err != nil {
			continue
		}
		if etv.EntityTypeID == et.ID {
			return catalog, etv, nil
		}
	}

	return nil, nil, domainerrors.NewNotFound("EntityType", fmt.Sprintf("%s is not pinned in catalog %s", entityTypeName, catalogName))
}

// InstanceDetail includes the instance and its resolved attribute values.
type InstanceDetail struct {
	Instance   *models.EntityInstance
	Attributes []AttributeValue
}

// AttributeValue is a resolved attribute value with name and type from the schema.
type AttributeValue struct {
	Name  string      `json:"name"`
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

func (s *InstanceService) resolveAttributeValues(ctx context.Context, inst *models.EntityInstance, etv *models.EntityTypeVersion) ([]AttributeValue, error) {
	attrs, err := s.attrRepo.ListByVersion(ctx, etv.ID)
	if err != nil {
		return nil, err
	}

	values, err := s.iavRepo.GetCurrentValues(ctx, inst.ID)
	if err != nil {
		return nil, err
	}

	valueMap := make(map[string]*models.InstanceAttributeValue)
	for _, v := range values {
		valueMap[v.AttributeID] = v
	}

	result := make([]AttributeValue, 0, len(attrs))
	for _, attr := range attrs {
		av := AttributeValue{
			Name: attr.Name,
			Type: string(attr.Type),
		}
		if val, ok := valueMap[attr.ID]; ok {
			switch attr.Type {
			case models.AttributeTypeString:
				av.Value = val.ValueString
			case models.AttributeTypeNumber:
				av.Value = val.ValueNumber
			case models.AttributeTypeEnum:
				av.Value = val.ValueEnum
			}
		}
		result = append(result, av)
	}
	return result, nil
}

func (s *InstanceService) validateAndBuildAttributeValues(ctx context.Context, etv *models.EntityTypeVersion, instanceID string, version int, attrInput map[string]interface{}) ([]*models.InstanceAttributeValue, error) {
	attrs, err := s.attrRepo.ListByVersion(ctx, etv.ID)
	if err != nil {
		return nil, err
	}

	attrByName := make(map[string]*models.Attribute)
	for _, a := range attrs {
		attrByName[a.Name] = a
	}

	var values []*models.InstanceAttributeValue
	for name, rawVal := range attrInput {
		attr, ok := attrByName[name]
		if !ok {
			return nil, domainerrors.NewValidation(fmt.Sprintf("unknown attribute: %s", name))
		}

		iav := &models.InstanceAttributeValue{
			ID:              uuid.Must(uuid.NewV7()).String(),
			InstanceID:      instanceID,
			InstanceVersion: version,
			AttributeID:     attr.ID,
		}

		switch attr.Type {
		case models.AttributeTypeString:
			iav.ValueString = fmt.Sprintf("%v", rawVal)
		case models.AttributeTypeNumber:
			switch v := rawVal.(type) {
			case float64:
				iav.ValueNumber = &v
			case int:
				f := float64(v)
				iav.ValueNumber = &f
			case string:
				f, err := strconv.ParseFloat(v, 64)
				if err != nil {
					return nil, domainerrors.NewValidation(fmt.Sprintf("attribute %s: expected number, got %q", name, v))
				}
				iav.ValueNumber = &f
			default:
				return nil, domainerrors.NewValidation(fmt.Sprintf("attribute %s: expected number", name))
			}
		case models.AttributeTypeEnum:
			strVal := fmt.Sprintf("%v", rawVal)
			// Validate enum value is in allowed list
			enumValues, err := s.enumValRepo.ListByEnum(ctx, attr.EnumID)
			if err != nil {
				return nil, err
			}
			valid := false
			for _, ev := range enumValues {
				if ev.Value == strVal {
					valid = true
					break
				}
			}
			if !valid {
				return nil, domainerrors.NewValidation(fmt.Sprintf("attribute %s: %q is not a valid enum value", name, strVal))
			}
			iav.ValueEnum = strVal
		}

		values = append(values, iav)
	}
	return values, nil
}

func (s *InstanceService) CreateInstance(ctx context.Context, catalogName, entityTypeName, name, description string, attrInput map[string]interface{}) (*InstanceDetail, error) {
	catalog, etv, err := s.resolveEntityType(ctx, catalogName, entityTypeName)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	inst := &models.EntityInstance{
		ID:           uuid.Must(uuid.NewV7()).String(),
		EntityTypeID: etv.EntityTypeID,
		CatalogID:    catalog.ID,
		Name:         name,
		Description:  description,
		Version:      1,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.instRepo.Create(ctx, inst); err != nil {
		return nil, err
	}

	// Store attribute values
	if len(attrInput) > 0 {
		values, err := s.validateAndBuildAttributeValues(ctx, etv, inst.ID, 1, attrInput)
		if err != nil {
			return nil, err
		}
		if err := s.iavRepo.SetValues(ctx, values); err != nil {
			return nil, err
		}
	}

	// Reset catalog validation status to draft
	_ = s.catalogRepo.UpdateValidationStatus(ctx, catalog.ID, models.ValidationStatusDraft)

	// Resolve attribute values for response
	avs, err := s.resolveAttributeValues(ctx, inst, etv)
	if err != nil {
		return nil, err
	}

	return &InstanceDetail{Instance: inst, Attributes: avs}, nil
}

func (s *InstanceService) GetInstance(ctx context.Context, catalogName, entityTypeName, instanceID string) (*InstanceDetail, error) {
	_, etv, err := s.resolveEntityType(ctx, catalogName, entityTypeName)
	if err != nil {
		return nil, err
	}

	inst, err := s.instRepo.GetByID(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	avs, err := s.resolveAttributeValues(ctx, inst, etv)
	if err != nil {
		return nil, err
	}

	return &InstanceDetail{Instance: inst, Attributes: avs}, nil
}

func (s *InstanceService) ListInstances(ctx context.Context, catalogName, entityTypeName string, params models.ListParams) ([]*InstanceDetail, int, error) {
	catalog, etv, err := s.resolveEntityType(ctx, catalogName, entityTypeName)
	if err != nil {
		return nil, 0, err
	}

	instances, total, err := s.instRepo.List(ctx, etv.EntityTypeID, catalog.ID, params)
	if err != nil {
		return nil, 0, err
	}

	// Fetch schema attributes once (not per instance)
	attrs, err := s.attrRepo.ListByVersion(ctx, etv.ID)
	if err != nil {
		return nil, 0, err
	}

	details := make([]*InstanceDetail, len(instances))
	for i, inst := range instances {
		values, err := s.iavRepo.GetCurrentValues(ctx, inst.ID)
		if err != nil {
			return nil, 0, err
		}
		valueMap := make(map[string]*models.InstanceAttributeValue)
		for _, v := range values {
			valueMap[v.AttributeID] = v
		}
		avs := make([]AttributeValue, 0, len(attrs))
		for _, attr := range attrs {
			av := AttributeValue{Name: attr.Name, Type: string(attr.Type)}
			if val, ok := valueMap[attr.ID]; ok {
				switch attr.Type {
				case models.AttributeTypeString:
					av.Value = val.ValueString
				case models.AttributeTypeNumber:
					av.Value = val.ValueNumber
				case models.AttributeTypeEnum:
					av.Value = val.ValueEnum
				}
			}
			avs = append(avs, av)
		}
		details[i] = &InstanceDetail{Instance: inst, Attributes: avs}
	}

	return details, total, nil
}

func (s *InstanceService) UpdateInstance(ctx context.Context, catalogName, entityTypeName, instanceID string, currentVersion int, name, description *string, attrInput map[string]interface{}) (*InstanceDetail, error) {
	catalog, etv, err := s.resolveEntityType(ctx, catalogName, entityTypeName)
	if err != nil {
		return nil, err
	}

	inst, err := s.instRepo.GetByID(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	if inst.Version != currentVersion {
		return nil, domainerrors.NewConflict("EntityInstance", fmt.Sprintf("version mismatch: expected %d but got %d", currentVersion, inst.Version))
	}

	if name != nil {
		inst.Name = *name
	}
	if description != nil {
		inst.Description = *description
	}
	inst.Version++
	inst.UpdatedAt = time.Now()

	if err := s.instRepo.Update(ctx, inst); err != nil {
		return nil, err
	}

	// Carry forward previous version's values and merge with new values
	prevValues, err := s.iavRepo.GetValuesForVersion(ctx, inst.ID, currentVersion)
	if err != nil {
		return nil, err
	}

	// Build new values from input
	var newValues []*models.InstanceAttributeValue
	if len(attrInput) > 0 {
		newValues, err = s.validateAndBuildAttributeValues(ctx, etv, inst.ID, inst.Version, attrInput)
		if err != nil {
			return nil, err
		}
	}

	// Build set of attribute IDs that have new values
	newAttrIDs := make(map[string]bool)
	for _, v := range newValues {
		newAttrIDs[v.AttributeID] = true
	}

	// Carry forward unchanged values from previous version
	for _, prev := range prevValues {
		if !newAttrIDs[prev.AttributeID] {
			carried := &models.InstanceAttributeValue{
				ID:              uuid.Must(uuid.NewV7()).String(),
				InstanceID:      inst.ID,
				InstanceVersion: inst.Version,
				AttributeID:     prev.AttributeID,
				ValueString:     prev.ValueString,
				ValueNumber:     prev.ValueNumber,
				ValueEnum:       prev.ValueEnum,
			}
			newValues = append(newValues, carried)
		}
	}

	if len(newValues) > 0 {
		if err := s.iavRepo.SetValues(ctx, newValues); err != nil {
			return nil, err
		}
	}

	// Reset catalog validation status
	_ = s.catalogRepo.UpdateValidationStatus(ctx, catalog.ID, models.ValidationStatusDraft)

	avs, err := s.resolveAttributeValues(ctx, inst, etv)
	if err != nil {
		return nil, err
	}

	return &InstanceDetail{Instance: inst, Attributes: avs}, nil
}

func (s *InstanceService) DeleteInstance(ctx context.Context, catalogName, entityTypeName, instanceID string) error {
	catalog, _, err := s.resolveEntityType(ctx, catalogName, entityTypeName)
	if err != nil {
		return err
	}

	// Cascade delete children
	if err := s.cascadeDelete(ctx, instanceID); err != nil {
		return err
	}

	// Reset catalog validation status
	_ = s.catalogRepo.UpdateValidationStatus(ctx, catalog.ID, models.ValidationStatusDraft)

	return nil
}

func (s *InstanceService) cascadeDelete(ctx context.Context, id string) error {
	children, _, err := s.instRepo.ListByParent(ctx, id, models.ListParams{Limit: 1000})
	if err != nil {
		return err
	}
	for _, child := range children {
		if err := s.cascadeDelete(ctx, child.ID); err != nil {
			return err
		}
	}
	return s.instRepo.SoftDelete(ctx, id)
}
