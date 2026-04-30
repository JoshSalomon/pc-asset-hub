package operational

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
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
	tdvRepo     repository.TypeDefinitionVersionRepository
	tdRepo      repository.TypeDefinitionRepository
	assocRepo   repository.AssociationRepository
	linkRepo    repository.AssociationLinkRepository
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
	tdvRepo repository.TypeDefinitionVersionRepository,
	tdRepo repository.TypeDefinitionRepository,
	assocRepo repository.AssociationRepository,
	linkRepo repository.AssociationLinkRepository,
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
		tdvRepo:     tdvRepo,
		tdRepo:      tdRepo,
		assocRepo:   assocRepo,
		linkRepo:    linkRepo,
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
			return nil, nil, fmt.Errorf("failed to resolve pin %s: %w", pin.EntityTypeVersionID, err)
		}
		if etv.EntityTypeID == et.ID {
			return catalog, etv, nil
		}
	}

	return nil, nil, domainerrors.NewNotFound("EntityType", fmt.Sprintf("%s is not pinned in catalog %s", entityTypeName, catalogName))
}

// TreeNode represents an instance in a containment tree.
type TreeNode struct {
	Instance       *models.EntityInstance
	EntityTypeName string
	Children       []TreeNode
}

// ParentChainEntry represents an ancestor in the parent chain for breadcrumb navigation.
type ParentChainEntry struct {
	InstanceID     string
	InstanceName   string
	EntityTypeName string
}

// InstanceDetail includes the instance and its resolved attribute values.
type InstanceDetail struct {
	Instance    *models.EntityInstance
	Attributes  []AttributeValue
	ParentChain []ParentChainEntry
}

// AttributeValue is a resolved attribute value with name and type from the schema.
type AttributeValue struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Value    any    `json:"value"`
	Required bool   `json:"required"`
}

// mapAttributeValues maps raw InstanceAttributeValues to resolved AttributeValues using schema attributes.
// This is the single source of truth for attribute value resolution — used by both single-instance
// and list operations. baseTypeByAttr maps attribute ID to the resolved base type string.
func mapAttributeValues(attrs []*models.Attribute, values []*models.InstanceAttributeValue, baseTypeByAttr map[string]string) []AttributeValue {
	valueMap := make(map[string]*models.InstanceAttributeValue)
	for _, v := range values {
		valueMap[v.AttributeID] = v
	}

	result := make([]AttributeValue, 0, len(attrs))
	for _, attr := range attrs {
		baseType, ok := baseTypeByAttr[attr.ID]
		if !ok {
			baseType = "string" // fallback if type resolution failed
		}
		av := AttributeValue{
			Name:     attr.Name,
			Type:     baseType,
			Required: attr.Required,
		}
		if val, ok := valueMap[attr.ID]; ok {
			switch models.BaseType(baseType) {
			case models.BaseTypeString, models.BaseTypeURL, models.BaseTypeDate, models.BaseTypeBoolean, models.BaseTypeEnum:
				av.Value = val.ValueString
			case models.BaseTypeNumber, models.BaseTypeInteger:
				av.Value = val.ValueNumber
			case models.BaseTypeList, models.BaseTypeJSON:
				if val.ValueJSON != "" {
					av.Value = val.ValueJSON
				}
			}
		}
		result = append(result, av)
	}
	return result
}

// resolveBaseTypes delegates to the shared ResolveBaseTypes helper.
func (s *InstanceService) resolveBaseTypes(ctx context.Context, attrs []*models.Attribute) (map[string]string, error) {
	return ResolveBaseTypes(ctx, attrs, s.tdvRepo, s.tdRepo)
}

func (s *InstanceService) resolveAttributeValues(ctx context.Context, inst *models.EntityInstance, etv *models.EntityTypeVersion) ([]AttributeValue, error) {
	attrs, err := s.attrRepo.ListByVersion(ctx, etv.ID)
	if err != nil {
		return nil, err
	}

	baseTypeByAttr, err := s.resolveBaseTypes(ctx, attrs)
	if err != nil {
		return nil, err
	}

	// Use inst.Version (authoritative) instead of GetCurrentValues (MAX query).
	// When all custom attrs are cleared, no IAVs exist at the new version —
	// MAX would fall back to the previous version, returning stale data.
	values, err := s.iavRepo.GetValuesForVersion(ctx, inst.ID, inst.Version)
	if err != nil {
		return nil, err
	}

	return mapAttributeValues(attrs, values, baseTypeByAttr), nil
}

// validateAndBuildAttributeValues validates and builds attribute values from input.
// Returns the built IAV records and a set of attribute IDs that were explicitly
// provided in the input (even if their value was empty/nil — used by UpdateInstance
// to avoid carrying forward cleared values).
func (s *InstanceService) validateAndBuildAttributeValues(ctx context.Context, etv *models.EntityTypeVersion, instanceID string, version int, attrInput map[string]any) ([]*models.InstanceAttributeValue, map[string]bool, error) {
	attrs, err := s.attrRepo.ListByVersion(ctx, etv.ID)
	if err != nil {
		return nil, nil, err
	}

	attrByName := make(map[string]*models.Attribute)
	for _, a := range attrs {
		attrByName[a.Name] = a
	}

	// Resolve base types for all attributes
	baseTypeByAttr, err := s.resolveBaseTypes(ctx, attrs)
	if err != nil {
		return nil, nil, err
	}

	touchedAttrIDs := make(map[string]bool)
	var values []*models.InstanceAttributeValue
	for name, rawVal := range attrInput {
		attr, ok := attrByName[name]
		if !ok {
			return nil, nil, domainerrors.NewValidation(fmt.Sprintf("unknown attribute: %s", name))
		}

		// Mark as explicitly touched (even if value is empty)
		touchedAttrIDs[attr.ID] = true

		// Skip empty values (draft mode allows clearing attributes)
		if rawVal == nil || rawVal == "" {
			continue
		}

		iav := &models.InstanceAttributeValue{
			ID:              uuid.Must(uuid.NewV7()).String(),
			InstanceID:      instanceID,
			InstanceVersion: version,
			AttributeID:     attr.ID,
		}

		baseType := models.BaseType(baseTypeByAttr[attr.ID])
		switch baseType {
		case models.BaseTypeString, models.BaseTypeURL, models.BaseTypeDate, models.BaseTypeEnum:
			iav.ValueString = fmt.Sprintf("%v", rawVal)
		case models.BaseTypeBoolean:
			iav.ValueString = fmt.Sprintf("%v", rawVal)
		case models.BaseTypeInteger:
			switch v := rawVal.(type) {
			case float64:
				iav.ValueNumber = &v
			case int:
				f := float64(v)
				iav.ValueNumber = &f
			case string:
				f, err := strconv.ParseFloat(v, 64)
				if err != nil {
					return nil, nil, domainerrors.NewValidation(fmt.Sprintf("attribute %s: expected integer, got %q", name, v))
				}
				iav.ValueNumber = &f
			default:
				return nil, nil, domainerrors.NewValidation(fmt.Sprintf("attribute %s: expected integer", name))
			}
		case models.BaseTypeNumber:
			switch v := rawVal.(type) {
			case float64:
				iav.ValueNumber = &v
			case int:
				f := float64(v)
				iav.ValueNumber = &f
			case string:
				f, err := strconv.ParseFloat(v, 64)
				if err != nil {
					return nil, nil, domainerrors.NewValidation(fmt.Sprintf("attribute %s: expected number, got %q", name, v))
				}
				iav.ValueNumber = &f
			default:
				return nil, nil, domainerrors.NewValidation(fmt.Sprintf("attribute %s: expected number", name))
			}
		case models.BaseTypeList, models.BaseTypeJSON:
			switch v := rawVal.(type) {
			case string:
				iav.ValueJSON = v
			default:
				b, err := json.Marshal(v)
				if err != nil {
					return nil, nil, domainerrors.NewValidation(fmt.Sprintf("attribute %s: failed to marshal JSON value", name))
				}
				iav.ValueJSON = string(b)
			}
		default:
			iav.ValueString = fmt.Sprintf("%v", rawVal)
		}

		values = append(values, iav)
	}
	return values, touchedAttrIDs, nil
}

func (s *InstanceService) CreateInstance(ctx context.Context, catalogName, entityTypeName, name, description string, attrInput map[string]any) (*InstanceDetail, error) {
	if strings.TrimSpace(name) == "" {
		return nil, domainerrors.NewValidation("instance name is required")
	}

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
		values, _, err := s.validateAndBuildAttributeValues(ctx, etv, inst.ID, 1, attrInput)
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
	catalog, etv, err := s.resolveEntityType(ctx, catalogName, entityTypeName)
	if err != nil {
		return nil, err
	}

	inst, err := s.instRepo.GetByID(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	if inst.CatalogID != catalog.ID {
		return nil, domainerrors.NewNotFound("EntityInstance", instanceID)
	}

	avs, err := s.resolveAttributeValues(ctx, inst, etv)
	if err != nil {
		return nil, err
	}

	detail := &InstanceDetail{Instance: inst, Attributes: avs}

	// Resolve parent chain for breadcrumb navigation
	if inst.ParentInstanceID != "" {
		chain, err := s.resolveParentChain(ctx, inst)
		if err != nil {
			return nil, err
		}
		detail.ParentChain = chain
	}

	return detail, nil
}

func (s *InstanceService) ListInstances(ctx context.Context, catalogName, entityTypeName string, params models.ListParams) ([]*InstanceDetail, int, error) {
	catalog, etv, err := s.resolveEntityType(ctx, catalogName, entityTypeName)
	if err != nil {
		return nil, 0, err
	}

	// Fetch schema attributes for filter name→ID translation and value resolution
	attrs, err := s.attrRepo.ListByVersion(ctx, etv.ID)
	if err != nil {
		return nil, 0, err
	}

	// Resolve base types once for all instances
	baseTypeByAttr, err := s.resolveBaseTypes(ctx, attrs)
	if err != nil {
		return nil, 0, err
	}

	// Translate filter attribute names to IDs
	if len(params.Filters) > 0 {
		attrByName := make(map[string]*models.Attribute)
		for _, a := range attrs {
			attrByName[a.Name] = a
		}
		resolved := make(map[string]string)
		for key, val := range params.Filters {
			// Handle .min/.max suffixes
			baseName := key
			suffix := ""
			if strings.HasSuffix(key, ".min") {
				baseName = strings.TrimSuffix(key, ".min")
				suffix = ".min"
			} else if strings.HasSuffix(key, ".max") {
				baseName = strings.TrimSuffix(key, ".max")
				suffix = ".max"
			}
			attr, ok := attrByName[baseName]
			if !ok {
				return nil, 0, domainerrors.NewValidation(fmt.Sprintf("unknown filter attribute: %s", baseName))
			}
			resolved[attr.ID+suffix] = val
		}
		params.Filters = resolved
	}

	instances, total, err := s.instRepo.List(ctx, etv.EntityTypeID, catalog.ID, params)
	if err != nil {
		return nil, 0, err
	}

	details := make([]*InstanceDetail, len(instances))
	for i, inst := range instances {
		values, err := s.iavRepo.GetValuesForVersion(ctx, inst.ID, inst.Version)
		if err != nil {
			return nil, 0, err
		}
		details[i] = &InstanceDetail{Instance: inst, Attributes: mapAttributeValues(attrs, values, baseTypeByAttr)}
	}

	return details, total, nil
}

func (s *InstanceService) UpdateInstance(ctx context.Context, catalogName, entityTypeName, instanceID string, currentVersion int, name, description *string, attrInput map[string]any) (*InstanceDetail, error) {
	catalog, etv, err := s.resolveEntityType(ctx, catalogName, entityTypeName)
	if err != nil {
		return nil, err
	}

	inst, err := s.instRepo.GetByID(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	if inst.CatalogID != catalog.ID {
		return nil, domainerrors.NewNotFound("EntityInstance", instanceID)
	}

	if inst.Version != currentVersion {
		return nil, domainerrors.NewConflict("EntityInstance", fmt.Sprintf("version mismatch: expected %d but got %d", currentVersion, inst.Version))
	}

	// Validate attribute values BEFORE incrementing version to avoid inconsistent state
	newVersion := currentVersion + 1
	var newValues []*models.InstanceAttributeValue
	touchedAttrIDs := make(map[string]bool)
	if len(attrInput) > 0 {
		var touched map[string]bool
		newValues, touched, err = s.validateAndBuildAttributeValues(ctx, etv, inst.ID, newVersion, attrInput)
		if err != nil {
			return nil, err
		}
		touchedAttrIDs = touched
	}

	// Carry forward previous version's values (skip explicitly touched attributes)
	prevValues, err := s.iavRepo.GetValuesForVersion(ctx, inst.ID, currentVersion)
	if err != nil {
		return nil, err
	}
	for _, prev := range prevValues {
		if !touchedAttrIDs[prev.AttributeID] {
			newValues = append(newValues, &models.InstanceAttributeValue{
				ID:              uuid.Must(uuid.NewV7()).String(),
				InstanceID:      inst.ID,
				InstanceVersion: newVersion,
				AttributeID:     prev.AttributeID,
				ValueString:     prev.ValueString,
				ValueNumber:     prev.ValueNumber,
				ValueJSON:       prev.ValueJSON,
			})
		}
	}

	// Now safe to update — validation passed
	if name != nil {
		inst.Name = *name
	}
	if description != nil {
		inst.Description = *description
	}
	inst.Version = newVersion
	inst.UpdatedAt = time.Now()

	if err := s.instRepo.Update(ctx, inst); err != nil {
		return nil, err
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

	// Verify instance belongs to this catalog
	inst, err := s.instRepo.GetByID(ctx, instanceID)
	if err != nil {
		return err
	}
	if inst.CatalogID != catalog.ID {
		return domainerrors.NewNotFound("EntityInstance", instanceID)
	}

	// Cascade delete children
	if err := s.cascadeDelete(ctx, instanceID); err != nil {
		return err
	}

	// Reset catalog validation status
	_ = s.catalogRepo.UpdateValidationStatus(ctx, catalog.ID, models.ValidationStatusDraft)

	return nil
}

func (s *InstanceService) CreateContainedInstance(ctx context.Context, catalogName, parentType, parentID, childType, name, description string, attrInput map[string]any) (*InstanceDetail, error) {
	if strings.TrimSpace(name) == "" {
		return nil, domainerrors.NewValidation("instance name is required")
	}

	// Resolve parent entity type
	catalog, parentETV, err := s.resolveEntityType(ctx, catalogName, parentType)
	if err != nil {
		return nil, err
	}

	// Verify parent instance exists and belongs to this catalog
	parentInst, err := s.instRepo.GetByID(ctx, parentID)
	if err != nil {
		return nil, err
	}
	if parentInst.CatalogID != catalog.ID {
		return nil, domainerrors.NewValidation("parent instance does not belong to this catalog")
	}

	// Resolve child entity type
	_, childETV, err := s.resolveEntityType(ctx, catalogName, childType)
	if err != nil {
		return nil, err
	}

	// Verify containment association exists between parent and child types in the CV
	assocs, err := s.assocRepo.ListByVersion(ctx, parentETV.ID)
	if err != nil {
		return nil, err
	}
	found := false
	for _, a := range assocs {
		if a.Type == models.AssociationTypeContainment && a.TargetEntityTypeID == childETV.EntityTypeID {
			found = true
			break
		}
	}
	if !found {
		return nil, domainerrors.NewValidation(fmt.Sprintf("no containment association from %s to %s", parentType, childType))
	}

	now := time.Now()
	inst := &models.EntityInstance{
		ID:               uuid.Must(uuid.NewV7()).String(),
		EntityTypeID:     childETV.EntityTypeID,
		CatalogID:        catalog.ID,
		ParentInstanceID: parentID,
		Name:             name,
		Description:      description,
		Version:          1,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.instRepo.Create(ctx, inst); err != nil {
		return nil, err
	}

	if len(attrInput) > 0 {
		values, _, err := s.validateAndBuildAttributeValues(ctx, childETV, inst.ID, 1, attrInput)
		if err != nil {
			return nil, err
		}
		if err := s.iavRepo.SetValues(ctx, values); err != nil {
			return nil, err
		}
	}

	_ = s.catalogRepo.UpdateValidationStatus(ctx, catalog.ID, models.ValidationStatusDraft)

	avs, err := s.resolveAttributeValues(ctx, inst, childETV)
	if err != nil {
		return nil, err
	}

	return &InstanceDetail{Instance: inst, Attributes: avs}, nil
}

func (s *InstanceService) ListContainedInstances(ctx context.Context, catalogName, parentType, parentID, childType string, params models.ListParams) ([]*InstanceDetail, int, error) {
	// Resolve parent to validate catalog/type
	_, _, err := s.resolveEntityType(ctx, catalogName, parentType)
	if err != nil {
		return nil, 0, err
	}

	// Verify parent exists
	if _, err := s.instRepo.GetByID(ctx, parentID); err != nil {
		return nil, 0, err
	}

	// Resolve child entity type
	_, childETV, err := s.resolveEntityType(ctx, catalogName, childType)
	if err != nil {
		return nil, 0, err
	}

	// List children by parent, filtered by entity type
	children, _, err := s.instRepo.ListByParent(ctx, parentID, params)
	if err != nil {
		return nil, 0, err
	}

	// Filter by child entity type
	var filtered []*models.EntityInstance
	for _, child := range children {
		if child.EntityTypeID == childETV.EntityTypeID {
			filtered = append(filtered, child)
		}
	}

	// Fetch schema attributes once
	attrs, err := s.attrRepo.ListByVersion(ctx, childETV.ID)
	if err != nil {
		return nil, 0, err
	}

	baseTypeByAttr, err := s.resolveBaseTypes(ctx, attrs)
	if err != nil {
		return nil, 0, err
	}

	details := make([]*InstanceDetail, len(filtered))
	for i, inst := range filtered {
		values, err := s.iavRepo.GetValuesForVersion(ctx, inst.ID, inst.Version)
		if err != nil {
			return nil, 0, err
		}
		details[i] = &InstanceDetail{Instance: inst, Attributes: mapAttributeValues(attrs, values, baseTypeByAttr)}
	}

	return details, len(filtered), nil
}

// ReferenceDetail represents a resolved forward or reverse reference.
type ReferenceDetail struct {
	LinkID          string `json:"link_id"`
	AssociationName string `json:"association_name"`
	AssociationType string `json:"association_type"`
	InstanceID      string `json:"instance_id"`
	InstanceName    string `json:"instance_name"`
	EntityTypeName  string `json:"entity_type_name"`
}

func (s *InstanceService) CreateAssociationLink(ctx context.Context, catalogName, entityTypeName, sourceInstanceID, targetInstanceID, associationName string) (*models.AssociationLink, error) {
	catalog, sourceETV, err := s.resolveEntityType(ctx, catalogName, entityTypeName)
	if err != nil {
		return nil, err
	}

	// Verify source instance exists and belongs to this catalog
	sourceInst, err := s.instRepo.GetByID(ctx, sourceInstanceID)
	if err != nil {
		return nil, err
	}
	if sourceInst.CatalogID != catalog.ID {
		return nil, domainerrors.NewValidation("source instance does not belong to this catalog")
	}

	// Verify target instance exists and belongs to same catalog
	targetInst, err := s.instRepo.GetByID(ctx, targetInstanceID)
	if err != nil {
		return nil, err
	}
	if targetInst.CatalogID != catalog.ID {
		return nil, domainerrors.NewValidation("target instance does not belong to this catalog")
	}

	// Find association definition by name in the source entity type version
	assocs, err := s.assocRepo.ListByVersion(ctx, sourceETV.ID)
	if err != nil {
		return nil, err
	}
	var assoc *models.Association
	reverse := false
	for _, a := range assocs {
		if a.Name == associationName {
			assoc = a
			break
		}
	}
	// For bidirectional associations, check the target's entity type versions too
	if assoc == nil {
		pins, err := s.pinRepo.ListByCatalogVersion(ctx, catalog.CatalogVersionID)
		if err != nil {
			return nil, err
		}
		for _, pin := range pins {
			etv, err := s.etvRepo.GetByID(ctx, pin.EntityTypeVersionID)
			if err != nil {
				continue
			}
			if etv.EntityTypeID != targetInst.EntityTypeID {
				continue
			}
			targetAssocs, err := s.assocRepo.ListByVersion(ctx, etv.ID)
			if err != nil {
				return nil, err
			}
			for _, a := range targetAssocs {
				if a.Name == associationName && a.Type == models.AssociationTypeBidirectional {
					assoc = a
					reverse = true
					break
				}
			}
			break
		}
	}
	if assoc == nil {
		return nil, domainerrors.NewNotFound("Association", associationName)
	}

	// Validate entity type match (forward: target matches assoc target; reverse: source matches assoc target)
	if reverse {
		if sourceInst.EntityTypeID != assoc.TargetEntityTypeID {
			return nil, domainerrors.NewValidation(fmt.Sprintf("source instance entity type %s does not match association target %s", sourceInst.EntityTypeID, assoc.TargetEntityTypeID))
		}
	} else {
		if targetInst.EntityTypeID != assoc.TargetEntityTypeID {
			return nil, domainerrors.NewValidation(fmt.Sprintf("target instance entity type %s does not match association target %s", targetInst.EntityTypeID, assoc.TargetEntityTypeID))
		}
	}

	// Check for duplicate link
	existingLinks, err := s.linkRepo.GetForwardRefs(ctx, sourceInstanceID)
	if err != nil {
		return nil, err
	}
	for _, existing := range existingLinks {
		if existing.AssociationID == assoc.ID && existing.TargetInstanceID == targetInstanceID {
			return nil, domainerrors.NewConflict("AssociationLink", "link already exists")
		}
	}

	link := &models.AssociationLink{
		ID:               uuid.Must(uuid.NewV7()).String(),
		AssociationID:    assoc.ID,
		SourceInstanceID: sourceInstanceID,
		TargetInstanceID: targetInstanceID,
		CreatedAt:        time.Now(),
	}

	if err := s.linkRepo.Create(ctx, link); err != nil {
		return nil, err
	}

	_ = s.catalogRepo.UpdateValidationStatus(ctx, catalog.ID, models.ValidationStatusDraft)

	return link, nil
}

func (s *InstanceService) DeleteAssociationLink(ctx context.Context, catalogName, entityTypeName, linkID string) error {
	catalog, _, err := s.resolveEntityType(ctx, catalogName, entityTypeName)
	if err != nil {
		return err
	}

	// Verify link exists and belongs to this catalog
	link, err := s.linkRepo.GetByID(ctx, linkID)
	if err != nil {
		return err
	}
	sourceInst, err := s.instRepo.GetByID(ctx, link.SourceInstanceID)
	if err != nil {
		return err
	}
	if sourceInst.CatalogID != catalog.ID {
		return domainerrors.NewValidation("link does not belong to this catalog")
	}

	if err := s.linkRepo.Delete(ctx, linkID); err != nil {
		return err
	}

	_ = s.catalogRepo.UpdateValidationStatus(ctx, catalog.ID, models.ValidationStatusDraft)
	return nil
}

func (s *InstanceService) GetForwardReferences(ctx context.Context, catalogName, entityTypeName, instanceID string) ([]ReferenceDetail, error) {
	_, _, err := s.resolveEntityType(ctx, catalogName, entityTypeName)
	if err != nil {
		return nil, err
	}

	// Verify instance exists
	if _, err := s.instRepo.GetByID(ctx, instanceID); err != nil {
		return nil, err
	}

	links, err := s.linkRepo.GetForwardRefs(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	return s.resolveLinks(ctx, links, false)
}

func (s *InstanceService) GetReverseReferences(ctx context.Context, catalogName, entityTypeName, instanceID string) ([]ReferenceDetail, error) {
	_, _, err := s.resolveEntityType(ctx, catalogName, entityTypeName)
	if err != nil {
		return nil, err
	}

	if _, err := s.instRepo.GetByID(ctx, instanceID); err != nil {
		return nil, err
	}

	links, err := s.linkRepo.GetReverseRefs(ctx, instanceID)
	if err != nil {
		return nil, err
	}

	return s.resolveLinks(ctx, links, true)
}

// resolveLinks resolves association links to ReferenceDetails.
// If reverse is true, the "other" instance is the source; otherwise it's the target.
func (s *InstanceService) resolveLinks(ctx context.Context, links []*models.AssociationLink, reverse bool) ([]ReferenceDetail, error) {
	refs := make([]ReferenceDetail, 0, len(links))
	for _, link := range links {
		assoc, err := s.assocRepo.GetByID(ctx, link.AssociationID)
		if err != nil {
			return nil, err
		}

		otherID := link.TargetInstanceID
		if reverse {
			otherID = link.SourceInstanceID
		}

		otherInst, err := s.instRepo.GetByID(ctx, otherID)
		if err != nil {
			return nil, err
		}

		et, err := s.etRepo.GetByID(ctx, otherInst.EntityTypeID)
		if err != nil {
			return nil, err
		}

		refs = append(refs, ReferenceDetail{
			LinkID:          link.ID,
			AssociationName: assoc.Name,
			AssociationType: string(assoc.Type),
			InstanceID:      otherInst.ID,
			InstanceName:    otherInst.Name,
			EntityTypeName:  et.Name,
		})
	}
	return refs, nil
}

func (s *InstanceService) SetParent(ctx context.Context, catalogName, childTypeName, childID, parentTypeName, parentID string) error {
	catalog, _, err := s.resolveEntityType(ctx, catalogName, childTypeName)
	if err != nil {
		return err
	}

	// Verify child instance exists and belongs to this catalog
	childInst, err := s.instRepo.GetByID(ctx, childID)
	if err != nil {
		return err
	}
	if childInst.CatalogID != catalog.ID {
		return domainerrors.NewValidation("instance does not belong to this catalog")
	}

	if parentID == "" {
		// Clear parent
		childInst.ParentInstanceID = ""
		childInst.UpdatedAt = time.Now()
		if err := s.instRepo.Update(ctx, childInst); err != nil {
			return err
		}
		_ = s.catalogRepo.UpdateValidationStatus(ctx, catalog.ID, models.ValidationStatusDraft)
		return nil
	}

	// Resolve parent entity type
	_, parentETV, err := s.resolveEntityType(ctx, catalogName, parentTypeName)
	if err != nil {
		return err
	}

	// Verify parent instance exists and belongs to this catalog
	parentInst, err := s.instRepo.GetByID(ctx, parentID)
	if err != nil {
		return err
	}
	if parentInst.CatalogID != catalog.ID {
		return domainerrors.NewValidation("parent instance does not belong to this catalog")
	}

	// Verify containment association exists from parent type to child type
	assocs, err := s.assocRepo.ListByVersion(ctx, parentETV.ID)
	if err != nil {
		return err
	}
	found := false
	for _, a := range assocs {
		if a.Type == models.AssociationTypeContainment && a.TargetEntityTypeID == childInst.EntityTypeID {
			found = true
			break
		}
	}
	if !found {
		return domainerrors.NewValidation(fmt.Sprintf("no containment association from %s to %s", parentTypeName, childTypeName))
	}

	childInst.ParentInstanceID = parentID
	childInst.UpdatedAt = time.Now()
	if err := s.instRepo.Update(ctx, childInst); err != nil {
		return err
	}

	_ = s.catalogRepo.UpdateValidationStatus(ctx, catalog.ID, models.ValidationStatusDraft)
	return nil
}

// GetContainmentTree builds a containment tree for all instances in a catalog.
func (s *InstanceService) GetContainmentTree(ctx context.Context, catalogName string) ([]TreeNode, error) {
	catalog, err := s.catalogRepo.GetByName(ctx, catalogName)
	if err != nil {
		return nil, err
	}

	instances, err := s.instRepo.ListByCatalog(ctx, catalog.ID)
	if err != nil {
		return nil, err
	}

	if len(instances) == 0 {
		return []TreeNode{}, nil
	}

	// Resolve entity type names (cache by ID)
	etNames := make(map[string]string)
	for _, inst := range instances {
		if _, ok := etNames[inst.EntityTypeID]; !ok {
			et, err := s.etRepo.GetByID(ctx, inst.EntityTypeID)
			if err != nil {
				etNames[inst.EntityTypeID] = inst.EntityTypeID // fallback
			} else {
				etNames[inst.EntityTypeID] = et.Name
			}
		}
	}

	// Index children by parent ID
	childrenMap := make(map[string][]*models.EntityInstance)
	for _, inst := range instances {
		childrenMap[inst.ParentInstanceID] = append(childrenMap[inst.ParentInstanceID], inst)
	}

	// Build nodes recursively
	var buildNodes func(parentID string) []TreeNode
	buildNodes = func(parentID string) []TreeNode {
		children := childrenMap[parentID]
		if len(children) == 0 {
			return nil
		}
		nodes := make([]TreeNode, len(children))
		for i, child := range children {
			nodes[i] = TreeNode{
				Instance:       child,
				EntityTypeName: etNames[child.EntityTypeID],
				Children:       buildNodes(child.ID),
			}
		}
		return nodes
	}

	return buildNodes(""), nil
}

// resolveParentChain walks up from an instance to the root, collecting ancestors.
// Returns entries in root-first order.
func (s *InstanceService) resolveParentChain(ctx context.Context, inst *models.EntityInstance) ([]ParentChainEntry, error) {
	const maxDepth = 50
	var chain []ParentChainEntry
	currentID := inst.ParentInstanceID
	etNames := make(map[string]string)
	visited := make(map[string]bool)

	for currentID != "" {
		if visited[currentID] || len(chain) >= maxDepth {
			break
		}
		visited[currentID] = true
		parent, err := s.instRepo.GetByID(ctx, currentID)
		if err != nil {
			return nil, err
		}

		etName := etNames[parent.EntityTypeID]
		if etName == "" {
			et, err := s.etRepo.GetByID(ctx, parent.EntityTypeID)
			if err == nil {
				etName = et.Name
			} else {
				etName = parent.EntityTypeID
			}
			etNames[parent.EntityTypeID] = etName
		}

		chain = append(chain, ParentChainEntry{
			InstanceID:     parent.ID,
			InstanceName:   parent.Name,
			EntityTypeName: etName,
		})
		currentID = parent.ParentInstanceID
	}

	// Reverse to root-first order
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}

	return chain, nil
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
	// Clean up association links where this instance is source or target
	if err := s.linkRepo.DeleteByInstance(ctx, id); err != nil {
		return err
	}
	return s.instRepo.Delete(ctx, id)
}
