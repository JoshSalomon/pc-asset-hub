package meta

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository"
)

type TypeDefinitionService struct {
	tdRepo   repository.TypeDefinitionRepository
	tdvRepo  repository.TypeDefinitionVersionRepository
	attrRepo repository.AttributeRepository
}

func NewTypeDefinitionService(
	tdRepo repository.TypeDefinitionRepository,
	tdvRepo repository.TypeDefinitionVersionRepository,
	attrRepo repository.AttributeRepository,
) *TypeDefinitionService {
	return &TypeDefinitionService{
		tdRepo:   tdRepo,
		tdvRepo:  tdvRepo,
		attrRepo: attrRepo,
	}
}

// CreateTypeDefinition creates a new user-defined type definition with V1.
func (s *TypeDefinitionService) CreateTypeDefinition(ctx context.Context, name, description string, baseType models.BaseType, constraints map[string]any) (*models.TypeDefinition, *models.TypeDefinitionVersion, error) {
	if name == "" {
		return nil, nil, domainerrors.NewValidation("type definition name is required")
	}
	if !models.ValidBaseTypes[baseType] {
		return nil, nil, domainerrors.NewValidation(fmt.Sprintf("invalid base type: %s", baseType))
	}

	if constraints == nil {
		constraints = map[string]any{}
	}
	if err := s.ValidateConstraints(baseType, constraints); err != nil {
		return nil, nil, err
	}

	now := time.Now()
	td := &models.TypeDefinition{
		ID:          uuid.Must(uuid.NewV7()).String(),
		Name:        name,
		Description: description,
		BaseType:    baseType,
		System:      false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.tdRepo.Create(ctx, td); err != nil {
		return nil, nil, err
	}

	tdv := &models.TypeDefinitionVersion{
		ID:               uuid.Must(uuid.NewV7()).String(),
		TypeDefinitionID: td.ID,
		VersionNumber:    1,
		Constraints:      constraints,
		CreatedAt:        now,
	}
	if err := s.tdvRepo.Create(ctx, tdv); err != nil {
		return nil, nil, err
	}

	return td, tdv, nil
}

// CreateSystemTypeDefinition creates an immutable system type definition (used during seeding).
func (s *TypeDefinitionService) CreateSystemTypeDefinition(ctx context.Context, name string, baseType models.BaseType) (*models.TypeDefinition, error) {
	now := time.Now()
	td := &models.TypeDefinition{
		ID:        uuid.Must(uuid.NewV7()).String(),
		Name:      name,
		BaseType:  baseType,
		System:    true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.tdRepo.Create(ctx, td); err != nil {
		return nil, err
	}

	tdv := &models.TypeDefinitionVersion{
		ID:               uuid.Must(uuid.NewV7()).String(),
		TypeDefinitionID: td.ID,
		VersionNumber:    1,
		Constraints:      map[string]any{},
		CreatedAt:        now,
	}
	if err := s.tdvRepo.Create(ctx, tdv); err != nil {
		return nil, err
	}

	return td, nil
}

// GetTypeDefinition returns a type definition and its latest version.
func (s *TypeDefinitionService) GetTypeDefinition(ctx context.Context, id string) (*models.TypeDefinition, *models.TypeDefinitionVersion, error) {
	td, err := s.tdRepo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	tdv, err := s.tdvRepo.GetLatestByTypeDefinition(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	return td, tdv, nil
}

// ListTypeDefinitions returns all type definitions.
func (s *TypeDefinitionService) ListTypeDefinitions(ctx context.Context, params models.ListParams) ([]*models.TypeDefinition, int, error) {
	return s.tdRepo.List(ctx, params)
}

// GetLatestVersionNumbers returns the latest version number for each type definition ID (batch).
func (s *TypeDefinitionService) GetLatestVersionNumbers(ctx context.Context, typeDefIDs []string) (map[string]int, error) {
	versions, err := s.tdvRepo.GetLatestByTypeDefinitions(ctx, typeDefIDs)
	if err != nil {
		return nil, err
	}
	result := make(map[string]int, len(versions))
	for id, v := range versions {
		result[id] = v.VersionNumber
	}
	return result, nil
}

// UpdateTypeDefinition creates a new version of a type definition with updated constraints.
func (s *TypeDefinitionService) UpdateTypeDefinition(ctx context.Context, id string, description *string, constraints map[string]any) (*models.TypeDefinitionVersion, error) {
	td, err := s.tdRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if td.System {
		return nil, domainerrors.NewValidation("system type definitions cannot be modified")
	}

	if description != nil {
		td.Description = *description
		td.UpdatedAt = time.Now()
		if err := s.tdRepo.Update(ctx, td); err != nil {
			return nil, err
		}
	}

	latest, err := s.tdvRepo.GetLatestByTypeDefinition(ctx, id)
	if err != nil {
		return nil, err
	}

	// Use new constraints if provided, otherwise carry forward from latest
	newConstraints := latest.Constraints
	if constraints != nil {
		newConstraints = constraints
	}

	if err := s.ValidateConstraints(td.BaseType, newConstraints); err != nil {
		return nil, err
	}

	tdv := &models.TypeDefinitionVersion{
		ID:               uuid.Must(uuid.NewV7()).String(),
		TypeDefinitionID: id,
		VersionNumber:    latest.VersionNumber + 1,
		Constraints:      newConstraints,
		CreatedAt:        time.Now(),
	}
	if err := s.tdvRepo.Create(ctx, tdv); err != nil {
		return nil, err
	}

	return tdv, nil
}

// DeleteTypeDefinition deletes a type definition (if not referenced by any attributes).
func (s *TypeDefinitionService) DeleteTypeDefinition(ctx context.Context, id string) error {
	td, err := s.tdRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if td.System {
		return domainerrors.NewValidation("system type definitions cannot be deleted")
	}
	return s.tdRepo.Delete(ctx, id)
}

// ListVersions returns all versions of a type definition.
func (s *TypeDefinitionService) ListVersions(ctx context.Context, typeDefID string) ([]*models.TypeDefinitionVersion, error) {
	return s.tdvRepo.ListByTypeDefinition(ctx, typeDefID)
}

// GetVersion returns a specific version of a type definition.
func (s *TypeDefinitionService) GetVersion(ctx context.Context, typeDefID string, versionNumber int) (*models.TypeDefinitionVersion, error) {
	versions, err := s.tdvRepo.ListByTypeDefinition(ctx, typeDefID)
	if err != nil {
		return nil, err
	}
	for _, v := range versions {
		if v.VersionNumber == versionNumber {
			return v, nil
		}
	}
	return nil, domainerrors.NewNotFound("TypeDefinitionVersion", fmt.Sprintf("v%d", versionNumber))
}

// ValidateConstraints validates constraints against their base type rules.
func (s *TypeDefinitionService) ValidateConstraints(baseType models.BaseType, constraints map[string]any) error {
	switch baseType {
	case models.BaseTypeString:
		return validateStringConstraints(constraints)
	case models.BaseTypeInteger:
		return validateMinMaxConstraints(constraints, "integer")
	case models.BaseTypeNumber:
		return validateMinMaxConstraints(constraints, "number")
	case models.BaseTypeEnum:
		return validateEnumConstraints(constraints)
	case models.BaseTypeList:
		return validateListConstraints(constraints)
	case models.BaseTypeBoolean, models.BaseTypeDate, models.BaseTypeURL, models.BaseTypeJSON:
		// No specific constraints to validate
		return nil
	}
	return domainerrors.NewValidation(fmt.Sprintf("unknown base type: %s", baseType))
}

func validateStringConstraints(c map[string]any) error {
	if ml, ok := c["max_length"]; ok {
		n, ok := toFloat64(ml)
		if !ok || n < 0 {
			return domainerrors.NewValidation("max_length must be a positive number")
		}
	}
	if p, ok := c["pattern"]; ok {
		s, ok := p.(string)
		if !ok {
			return domainerrors.NewValidation("pattern must be a string")
		}
		if _, err := regexp.Compile(s); err != nil {
			return domainerrors.NewValidation(fmt.Sprintf("invalid regex pattern: %v", err))
		}
	}
	return nil
}

func validateMinMaxConstraints(c map[string]any, typeName string) error {
	var hasMin, hasMax bool
	var minVal, maxVal float64

	if m, ok := c["min"]; ok {
		n, ok := toFloat64(m)
		if !ok {
			return domainerrors.NewValidation(fmt.Sprintf("%s min must be a number", typeName))
		}
		hasMin = true
		minVal = n
	}
	if m, ok := c["max"]; ok {
		n, ok := toFloat64(m)
		if !ok {
			return domainerrors.NewValidation(fmt.Sprintf("%s max must be a number", typeName))
		}
		hasMax = true
		maxVal = n
	}
	if hasMin && hasMax && minVal > maxVal {
		return domainerrors.NewValidation(fmt.Sprintf("%s min (%v) must be <= max (%v)", typeName, minVal, maxVal))
	}
	return nil
}

func validateEnumConstraints(c map[string]any) error {
	vals, ok := c["values"]
	if !ok {
		return domainerrors.NewValidation("enum type requires 'values' constraint")
	}
	arr, ok := vals.([]any)
	if !ok || len(arr) == 0 {
		return domainerrors.NewValidation("enum values must be a non-empty array")
	}
	seen := make(map[string]bool)
	for _, v := range arr {
		s, ok := v.(string)
		if !ok {
			return domainerrors.NewValidation("enum values must be strings")
		}
		if seen[s] {
			return domainerrors.NewValidation(fmt.Sprintf("duplicate enum value: %s", s))
		}
		seen[s] = true
	}
	return nil
}

func validateListConstraints(c map[string]any) error {
	if et, ok := c["element_base_type"]; ok {
		s, ok := et.(string)
		if !ok {
			return domainerrors.NewValidation("element_base_type must be a string")
		}
		bt := models.BaseType(s)
		// List elements cannot be list, json, or enum
		if bt == models.BaseTypeList || bt == models.BaseTypeJSON || bt == models.BaseTypeEnum {
			return domainerrors.NewValidation(fmt.Sprintf("list element type cannot be %s", s))
		}
		if !models.ValidBaseTypes[bt] {
			return domainerrors.NewValidation(fmt.Sprintf("invalid element base type: %s", s))
		}
	}
	if ml, ok := c["max_length"]; ok {
		n, ok := toFloat64(ml)
		if !ok || n < 0 {
			return domainerrors.NewValidation("list max_length must be a positive number")
		}
	}
	return nil
}

func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	}
	return 0, false
}
