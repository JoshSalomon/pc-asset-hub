package operational

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository"
)

// ValidationError represents a single validation violation.
type ValidationError struct {
	EntityType   string
	InstanceName string
	Field        string
	Violation    string
}

// ValidationResult contains the outcome of a catalog validation.
type ValidationResult struct {
	Status models.ValidationStatus
	Errors []ValidationError
}

// CatalogValidationService validates all instances in a catalog against the pinned CV.
type CatalogValidationService struct {
	catalogRepo repository.CatalogRepository
	instRepo    repository.EntityInstanceRepository
	iavRepo     repository.InstanceAttributeValueRepository
	pinRepo     repository.CatalogVersionPinRepository
	etvRepo     repository.EntityTypeVersionRepository
	attrRepo    repository.AttributeRepository
	assocRepo   repository.AssociationRepository
	enumValRepo repository.EnumValueRepository
	linkRepo    repository.AssociationLinkRepository
	etRepo      repository.EntityTypeRepository
}

func NewCatalogValidationService(
	catalogRepo repository.CatalogRepository,
	instRepo repository.EntityInstanceRepository,
	iavRepo repository.InstanceAttributeValueRepository,
	pinRepo repository.CatalogVersionPinRepository,
	etvRepo repository.EntityTypeVersionRepository,
	attrRepo repository.AttributeRepository,
	assocRepo repository.AssociationRepository,
	enumValRepo repository.EnumValueRepository,
	linkRepo repository.AssociationLinkRepository,
	etRepo repository.EntityTypeRepository,
) *CatalogValidationService {
	return &CatalogValidationService{
		catalogRepo: catalogRepo,
		instRepo:    instRepo,
		iavRepo:     iavRepo,
		pinRepo:     pinRepo,
		etvRepo:     etvRepo,
		attrRepo:    attrRepo,
		assocRepo:   assocRepo,
		enumValRepo: enumValRepo,
		linkRepo:    linkRepo,
		etRepo:      etRepo,
	}
}

// Validate validates all instances in the named catalog against the pinned CV's schema.
func (s *CatalogValidationService) Validate(ctx context.Context, catalogName string) (*ValidationResult, error) {
	catalog, err := s.catalogRepo.GetByName(ctx, catalogName)
	if err != nil {
		return nil, err
	}

	instances, err := s.instRepo.ListByCatalog(ctx, catalog.ID)
	if err != nil {
		return nil, err
	}

	// Empty catalog passes
	if len(instances) == 0 {
		if err := s.catalogRepo.UpdateValidationStatus(ctx, catalog.ID, models.ValidationStatusValid); err != nil {
			return nil, err
		}
		return &ValidationResult{Status: models.ValidationStatusValid, Errors: []ValidationError{}}, nil
	}

	// Resolve pins → entity type version mapping
	pins, err := s.pinRepo.ListByCatalogVersion(ctx, catalog.CatalogVersionID)
	if err != nil {
		return nil, err
	}

	etToETV := make(map[string]string)
	for _, pin := range pins {
		etv, err := s.etvRepo.GetByID(ctx, pin.EntityTypeVersionID)
		if err != nil {
			return nil, err
		}
		etToETV[etv.EntityTypeID] = etv.ID
	}

	// Cache entity type names
	etNames := make(map[string]string)
	resolveETName := func(etID string) string {
		if name, ok := etNames[etID]; ok {
			return name
		}
		et, err := s.etRepo.GetByID(ctx, etID)
		if err != nil {
			etNames[etID] = etID
			return etID
		}
		etNames[etID] = et.Name
		return et.Name
	}

	validationErrors := []ValidationError{}

	// Index instances by ID for parent lookup
	instanceByID := make(map[string]*models.EntityInstance)
	for _, inst := range instances {
		instanceByID[inst.ID] = inst
	}

	// Group instances by entity type
	instancesByET := make(map[string][]*models.EntityInstance)
	for _, inst := range instances {
		instancesByET[inst.EntityTypeID] = append(instancesByET[inst.EntityTypeID], inst)
	}

	// Pre-load associations for all pinned entity types (used by both mandatory assoc and containment checks)
	assocCache := make(map[string][]*models.Association)
	for _, etvID := range etToETV {
		assocs, err := s.assocRepo.ListByVersion(ctx, etvID)
		if err != nil {
			return nil, err
		}
		assocCache[etvID] = assocs
	}

	// Validate each entity type group
	for etID, etInstances := range instancesByET {
		etvID, ok := etToETV[etID]
		if !ok {
			// L1 fix: instances of unpinned entity types are flagged
			etName := resolveETName(etID)
			for _, inst := range etInstances {
				validationErrors = append(validationErrors, ValidationError{
					EntityType:   etName,
					InstanceName: inst.Name,
					Field:        "entity_type",
					Violation:    fmt.Sprintf("entity type %q is not pinned in the catalog version", etName),
				})
			}
			continue
		}

		etName := resolveETName(etID)

		// Load schema attributes
		attrs, err := s.attrRepo.ListByVersion(ctx, etvID)
		if err != nil {
			return nil, err
		}

		// Cache enum values for enum attributes
		enumAllowed := make(map[string]map[string]bool)
		for _, attr := range attrs {
			if attr.Type == models.AttributeTypeEnum && attr.EnumID != "" {
				if _, cached := enumAllowed[attr.EnumID]; !cached {
					evs, err := s.enumValRepo.ListByEnum(ctx, attr.EnumID)
					if err != nil {
						return nil, err
					}
					allowed := make(map[string]bool)
					for _, ev := range evs {
						allowed[ev.Value] = true
					}
					enumAllowed[attr.EnumID] = allowed
				}
			}
		}

		// Collect non-containment associations for cardinality checks
		assocs := assocCache[etvID]
		var nonContainmentAssocs []*models.Association
		for _, assoc := range assocs {
			if assoc.Type == models.AssociationTypeContainment {
				continue
			}
			nonContainmentAssocs = append(nonContainmentAssocs, assoc)
		}

		// Check each instance
		for _, inst := range etInstances {
			values, err := s.iavRepo.GetCurrentValues(ctx, inst.ID)
			if err != nil {
				return nil, err
			}

			valueByAttrID := make(map[string]*models.InstanceAttributeValue)
			for _, v := range values {
				valueByAttrID[v.AttributeID] = v
			}

			for _, attr := range attrs {
				val, hasVal := valueByAttrID[attr.ID]

				// Required check
				if attr.Required && (!hasVal || IsEmptyValue(attr, val)) {
					validationErrors = append(validationErrors, ValidationError{
						EntityType:   etName,
						InstanceName: inst.Name,
						Field:        attr.Name,
						Violation:    fmt.Sprintf("required attribute %q is missing a value", attr.Name),
					})
					continue
				}

				// Type check (only if value present)
				if hasVal && !IsEmptyValue(attr, val) {
					if attr.Type == models.AttributeTypeEnum && attr.EnumID != "" {
						allowed := enumAllowed[attr.EnumID]
						if !allowed[val.ValueEnum] {
							validationErrors = append(validationErrors, ValidationError{
								EntityType:   etName,
								InstanceName: inst.Name,
								Field:        attr.Name,
								Violation:    fmt.Sprintf("invalid enum value %q for attribute %q", val.ValueEnum, attr.Name),
							})
						}
					}
				}
			}

			// Target cardinality check: for each non-containment association,
			// count forward links and verify against target_cardinality (min and max).
			if len(nonContainmentAssocs) > 0 {
				links, err := s.linkRepo.GetForwardRefs(ctx, inst.ID)
				if err != nil {
					return nil, err
				}
				linksByAssoc := make(map[string]int)
				for _, link := range links {
					linksByAssoc[link.AssociationID]++
				}
				for _, assoc := range nonContainmentAssocs {
					count := linksByAssoc[assoc.ID]
					tMin, tMax, tUnbounded := ParseCardinality(assoc.TargetCardinality)
					if count < tMin {
						validationErrors = append(validationErrors, ValidationError{
							EntityType:   etName,
							InstanceName: inst.Name,
							Field:        assoc.Name,
							Violation:    fmt.Sprintf("mandatory association %q requires at least %d link(s) (cardinality %s), has %d", assoc.Name, tMin, assoc.TargetCardinality, count),
						})
					}
					if !tUnbounded && count > tMax {
						validationErrors = append(validationErrors, ValidationError{
							EntityType:   etName,
							InstanceName: inst.Name,
							Field:        assoc.Name,
							Violation:    fmt.Sprintf("association %q exceeds maximum of %d link(s) (cardinality %s), has %d", assoc.Name, tMax, assoc.TargetCardinality, count),
						})
					}
				}
			}
		}
	}

	// Source cardinality check: for associations with source_cardinality min >= 1,
	// each target entity type instance must have enough reverse links.
	// Collect associations with source cardinality constraints.
	// Source cardinality check: for associations with source cardinality constraints
	// (min > 0 or bounded max), verify each target instance has the right number of reverse links.
	var sourceChecks []*models.Association
	for _, assocs := range assocCache {
		for _, assoc := range assocs {
			if assoc.Type == models.AssociationTypeContainment {
				continue
			}
			sMin, sMax, sUnbounded := ParseCardinality(assoc.SourceCardinality)
			if sMin > 0 || (!sUnbounded && sMax > 0) {
				sourceChecks = append(sourceChecks, assoc)
			}
		}
	}

	for _, assoc := range sourceChecks {
		targetInstances := instancesByET[assoc.TargetEntityTypeID]
		for _, inst := range targetInstances {
			links, err := s.linkRepo.GetReverseRefs(ctx, inst.ID)
			if err != nil {
				return nil, err
			}
			count := 0
			for _, link := range links {
				if link.AssociationID == assoc.ID {
					count++
				}
			}
			sMin, sMax, sUnbounded := ParseCardinality(assoc.SourceCardinality)
			targetETName := resolveETName(inst.EntityTypeID)
			if count < sMin {
				validationErrors = append(validationErrors, ValidationError{
					EntityType:   targetETName,
					InstanceName: inst.Name,
					Field:        assoc.Name,
					Violation:    fmt.Sprintf("source cardinality %q requires at least %d link(s) from %s, has %d", assoc.SourceCardinality, sMin, resolveETName(assoc.TargetEntityTypeID), count),
				})
			}
			if !sUnbounded && count > sMax {
				validationErrors = append(validationErrors, ValidationError{
					EntityType:   targetETName,
					InstanceName: inst.Name,
					Field:        assoc.Name,
					Violation:    fmt.Sprintf("source cardinality %q exceeds maximum of %d link(s), has %d", assoc.SourceCardinality, sMax, count),
				})
			}
		}
	}

	// Build set of entity type IDs that are targets of containment associations
	// (i.e., entity types that must be contained by a parent)
	containedETIDs := make(map[string]bool)
	for _, assocs := range assocCache {
		for _, a := range assocs {
			if a.Type == models.AssociationTypeContainment {
				containedETIDs[a.TargetEntityTypeID] = true
			}
		}
	}

	// Containment consistency check
	for _, inst := range instances {
		if inst.ParentInstanceID == "" {
			// Check if this entity type is supposed to be contained
			if containedETIDs[inst.EntityTypeID] {
				etName := resolveETName(inst.EntityTypeID)
				validationErrors = append(validationErrors, ValidationError{
					EntityType:   etName,
					InstanceName: inst.Name,
					Field:        "parent",
					Violation:    fmt.Sprintf("contained entity type %q requires a parent instance", etName),
				})
			}
			continue
		}

		etName := resolveETName(inst.EntityTypeID)

		parent, exists := instanceByID[inst.ParentInstanceID]
		if !exists {
			validationErrors = append(validationErrors, ValidationError{
				EntityType:   etName,
				InstanceName: inst.Name,
				Field:        "parent",
				Violation:    "orphaned contained instance: parent does not exist",
			})
			continue
		}

		parentETVID, ok := etToETV[parent.EntityTypeID]
		if !ok {
			validationErrors = append(validationErrors, ValidationError{
				EntityType:   etName,
				InstanceName: inst.Name,
				Field:        "parent",
				Violation:    "invalid containment: parent entity type not pinned in CV",
			})
			continue
		}

		parentAssocs := assocCache[parentETVID]
		found := false
		for _, a := range parentAssocs {
			if a.Type == models.AssociationTypeContainment && a.TargetEntityTypeID == inst.EntityTypeID {
				found = true
				break
			}
		}
		if !found {
			validationErrors = append(validationErrors, ValidationError{
				EntityType:   etName,
				InstanceName: inst.Name,
				Field:        "parent",
				Violation:    fmt.Sprintf("invalid containment relationship: no containment association from %s to %s", resolveETName(parent.EntityTypeID), etName),
			})
		}
	}

	status := models.ValidationStatusValid
	if len(validationErrors) > 0 {
		status = models.ValidationStatusInvalid
	}

	if err := s.catalogRepo.UpdateValidationStatus(ctx, catalog.ID, status); err != nil {
		return nil, err
	}

	return &ValidationResult{Status: status, Errors: validationErrors}, nil
}

// ParseCardinality parses a cardinality string into min, max, and whether the max is unbounded.
// Examples: "0..n" → (0, 0, true), "1" → (1, 1, false), "1..5" → (1, 5, false), "" → (0, 0, true).
func ParseCardinality(cardinality string) (min int, max int, unbounded bool) {
	if cardinality == "" {
		return 0, 0, true // default: 0..n
	}
	parts := strings.SplitN(cardinality, "..", 2)
	min, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, true
	}
	if len(parts) == 1 {
		return min, min, false // exact value: "1" means 1..1
	}
	if parts[1] == "n" {
		return min, 0, true
	}
	max, err = strconv.Atoi(parts[1])
	if err != nil {
		return min, 0, true
	}
	return min, max, false
}

// CardinalityMinGE1 returns true if the cardinality string has a minimum >= 1.
func CardinalityMinGE1(cardinality string) bool {
	min, _, _ := ParseCardinality(cardinality)
	return min >= 1
}

func IsEmptyValue(attr *models.Attribute, val *models.InstanceAttributeValue) bool {
	switch attr.Type {
	case models.AttributeTypeString:
		return val.ValueString == ""
	case models.AttributeTypeNumber:
		return val.ValueNumber == nil
	case models.AttributeTypeEnum:
		return val.ValueEnum == ""
	}
	return true
}
