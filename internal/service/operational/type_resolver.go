package operational

import (
	"context"
	"fmt"

	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository"
)

// AttrTypeInfo holds resolved type information for an attribute.
type AttrTypeInfo struct {
	BaseType    models.BaseType
	Constraints map[string]any
}

// ResolveBaseTypes resolves the base type for each attribute by looking up its TypeDefinitionVersion.
// Returns a map from attribute ID to base type string.
func ResolveBaseTypes(ctx context.Context, attrs []*models.Attribute, tdvRepo repository.TypeDefinitionVersionRepository, tdRepo repository.TypeDefinitionRepository) (map[string]string, error) {
	baseTypeByAttr := make(map[string]string, len(attrs))
	tdvCache := make(map[string]*models.TypeDefinitionVersion)
	tdCache := make(map[string]*models.TypeDefinition)

	for _, attr := range attrs {
		tdv, ok := tdvCache[attr.TypeDefinitionVersionID]
		if !ok {
			var err error
			tdv, err = tdvRepo.GetByID(ctx, attr.TypeDefinitionVersionID)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve type for attribute %s: %w", attr.Name, err)
			}
			tdvCache[attr.TypeDefinitionVersionID] = tdv
		}
		td, ok := tdCache[tdv.TypeDefinitionID]
		if !ok {
			var err error
			td, err = tdRepo.GetByID(ctx, tdv.TypeDefinitionID)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve type definition for attribute %s: %w", attr.Name, err)
			}
			tdCache[tdv.TypeDefinitionID] = td
		}
		baseTypeByAttr[attr.ID] = string(td.BaseType)
	}
	return baseTypeByAttr, nil
}

// ResolveAttrTypeInfo resolves full type information (base type + constraints) for each attribute.
// Returns a map from attribute ID to AttrTypeInfo.
func ResolveAttrTypeInfo(ctx context.Context, attrs []*models.Attribute, tdvRepo repository.TypeDefinitionVersionRepository, tdRepo repository.TypeDefinitionRepository) (map[string]*AttrTypeInfo, error) {
	typeInfo := make(map[string]*AttrTypeInfo, len(attrs))
	tdvCache := make(map[string]*models.TypeDefinitionVersion)
	tdCache := make(map[string]*models.TypeDefinition)

	for _, attr := range attrs {
		tdv, ok := tdvCache[attr.TypeDefinitionVersionID]
		if !ok {
			var err error
			tdv, err = tdvRepo.GetByID(ctx, attr.TypeDefinitionVersionID)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve type for attribute %s: %w", attr.Name, err)
			}
			tdvCache[attr.TypeDefinitionVersionID] = tdv
		}
		td, ok := tdCache[tdv.TypeDefinitionID]
		if !ok {
			var err error
			td, err = tdRepo.GetByID(ctx, tdv.TypeDefinitionID)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve type definition for attribute %s: %w", attr.Name, err)
			}
			tdCache[tdv.TypeDefinitionID] = td
		}
		typeInfo[attr.ID] = &AttrTypeInfo{
			BaseType:    td.BaseType,
			Constraints: tdv.Constraints,
		}
	}
	return typeInfo, nil
}
