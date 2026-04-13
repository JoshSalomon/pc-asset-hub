package meta

import (
	"context"

	"github.com/google/uuid"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository"
)

type AttributeService struct {
	attrRepo repository.AttributeRepository
	etvRepo  repository.EntityTypeVersionRepository
	etRepo   repository.EntityTypeRepository
	assocRepo repository.AssociationRepository
	tdvRepo  repository.TypeDefinitionVersionRepository
}

func NewAttributeService(
	attrRepo repository.AttributeRepository,
	etvRepo repository.EntityTypeVersionRepository,
	etRepo repository.EntityTypeRepository,
	assocRepo repository.AssociationRepository,
	tdvRepo repository.TypeDefinitionVersionRepository,
) *AttributeService {
	return &AttributeService{
		attrRepo:  attrRepo,
		etvRepo:   etvRepo,
		etRepo:    etRepo,
		assocRepo: assocRepo,
		tdvRepo:   tdvRepo,
	}
}

// AddAttribute adds an attribute to an entity type, creating a new version.
func (s *AttributeService) AddAttribute(ctx context.Context, entityTypeID string, name, description string, typeDefinitionVersionID string, required bool) (*models.EntityTypeVersion, error) {
	if name == "" {
		return nil, domainerrors.NewValidation("attribute name is required")
	}
	if models.IsSystemAttributeName(name) {
		return nil, domainerrors.NewValidation("attribute name \"" + name + "\" is reserved for system attributes")
	}

	// Validate type definition version reference
	if typeDefinitionVersionID == "" {
		return nil, domainerrors.NewValidation("type_definition_version_id is required")
	}
	if _, err := s.tdvRepo.GetByID(ctx, typeDefinitionVersionID); err != nil {
		return nil, domainerrors.NewValidation("invalid type_definition_version_id: " + typeDefinitionVersionID)
	}

	// Create new version with copy-on-write
	latest, err := s.etvRepo.GetLatestByEntityType(ctx, entityTypeID)
	if err != nil {
		return nil, err
	}

	// Check for duplicate name in current version (attributes)
	attrs, err := s.attrRepo.ListByVersion(ctx, latest.ID)
	if err != nil {
		return nil, err
	}
	for _, a := range attrs {
		if a.Name == name {
			return nil, domainerrors.NewConflict("Attribute", "attribute name already exists: "+name)
		}
	}

	// Check shared namespace: name must not conflict with association names
	assocs, err := s.assocRepo.ListByVersion(ctx, latest.ID)
	if err != nil {
		return nil, err
	}
	for _, a := range assocs {
		if a.Name == name {
			return nil, domainerrors.NewConflict("Attribute", "name conflicts with association: "+name)
		}
	}

	newVersion := &models.EntityTypeVersion{
		ID:           uuid.Must(uuid.NewV7()).String(),
		EntityTypeID: entityTypeID,
		Version:      latest.Version + 1,
		Description:  latest.Description,
	}
	if err := s.etvRepo.Create(ctx, newVersion); err != nil {
		return nil, err
	}

	if err := s.attrRepo.BulkCopyToVersion(ctx, latest.ID, newVersion.ID); err != nil {
		return nil, err
	}
	if err := s.assocRepo.BulkCopyToVersion(ctx, latest.ID, newVersion.ID); err != nil {
		return nil, err
	}

	// Add the new attribute
	attr := &models.Attribute{
		ID:                      uuid.Must(uuid.NewV7()).String(),
		EntityTypeVersionID:     newVersion.ID,
		Name:                    name,
		Description:             description,
		TypeDefinitionVersionID: typeDefinitionVersionID,
		Ordinal:                 len(attrs),
		Required:                required,
	}
	if err := s.attrRepo.Create(ctx, attr); err != nil {
		return nil, err
	}

	return newVersion, nil
}

// RemoveAttribute removes an attribute, creating a new version without it.
func (s *AttributeService) RemoveAttribute(ctx context.Context, entityTypeID, attributeName string) (*models.EntityTypeVersion, error) {
	latest, err := s.etvRepo.GetLatestByEntityType(ctx, entityTypeID)
	if err != nil {
		return nil, err
	}

	newVersion := &models.EntityTypeVersion{
		ID:           uuid.Must(uuid.NewV7()).String(),
		EntityTypeID: entityTypeID,
		Version:      latest.Version + 1,
		Description:  latest.Description,
	}
	if err := s.etvRepo.Create(ctx, newVersion); err != nil {
		return nil, err
	}

	if err := s.attrRepo.BulkCopyToVersion(ctx, latest.ID, newVersion.ID); err != nil {
		return nil, err
	}
	if err := s.assocRepo.BulkCopyToVersion(ctx, latest.ID, newVersion.ID); err != nil {
		return nil, err
	}

	// Find and delete the attribute from the new version
	newAttrs, err := s.attrRepo.ListByVersion(ctx, newVersion.ID)
	if err != nil {
		return nil, err
	}
	for _, a := range newAttrs {
		if a.Name == attributeName {
			if err := s.attrRepo.Delete(ctx, a.ID); err != nil {
				return nil, err
			}
			return newVersion, nil
		}
	}
	return nil, domainerrors.NewNotFound("Attribute", attributeName)
}

// CopyAttributesFromType copies selected attributes from a source entity type to a target.
func (s *AttributeService) CopyAttributesFromType(ctx context.Context, targetEntityTypeID, sourceEntityTypeID string, sourceVersion int, attributeNames []string) (*models.EntityTypeVersion, error) {
	// Filter out system attribute names (Name, Description are synthetic, not real DB attrs)
	var filteredNames []string
	for _, name := range attributeNames {
		if models.IsSystemAttributeName(name) {
			continue
		}
		filteredNames = append(filteredNames, name)
	}
	if len(filteredNames) == 0 {
		return nil, domainerrors.NewValidation("no custom attributes to copy (system attributes are excluded)")
	}
	attributeNames = filteredNames

	sourceETV, err := s.etvRepo.GetByEntityTypeAndVersion(ctx, sourceEntityTypeID, sourceVersion)
	if err != nil {
		return nil, err
	}

	sourceAttrs, err := s.attrRepo.ListByVersion(ctx, sourceETV.ID)
	if err != nil {
		return nil, err
	}

	latest, err := s.etvRepo.GetLatestByEntityType(ctx, targetEntityTypeID)
	if err != nil {
		return nil, err
	}

	targetAttrs, err := s.attrRepo.ListByVersion(ctx, latest.ID)
	if err != nil {
		return nil, err
	}

	// Check for conflicts
	existingNames := make(map[string]bool)
	for _, a := range targetAttrs {
		existingNames[a.Name] = true
	}

	var toCopy []*models.Attribute
	for _, name := range attributeNames {
		if existingNames[name] {
			return nil, domainerrors.NewConflict("Attribute", "attribute name already exists on target: "+name)
		}
		found := false
		for _, sa := range sourceAttrs {
			if sa.Name == name {
				toCopy = append(toCopy, sa)
				found = true
				break
			}
		}
		if !found {
			return nil, domainerrors.NewNotFound("Attribute", name)
		}
	}

	// Create new version
	newVersion := &models.EntityTypeVersion{
		ID:           uuid.Must(uuid.NewV7()).String(),
		EntityTypeID: targetEntityTypeID,
		Version:      latest.Version + 1,
		Description:  latest.Description,
	}
	if err := s.etvRepo.Create(ctx, newVersion); err != nil {
		return nil, err
	}
	if err := s.attrRepo.BulkCopyToVersion(ctx, latest.ID, newVersion.ID); err != nil {
		return nil, err
	}
	if err := s.assocRepo.BulkCopyToVersion(ctx, latest.ID, newVersion.ID); err != nil {
		return nil, err
	}

	// Add copied attributes
	for i, src := range toCopy {
		attr := &models.Attribute{
			ID:                      uuid.Must(uuid.NewV7()).String(),
			EntityTypeVersionID:     newVersion.ID,
			Name:                    src.Name,
			Description:             src.Description,
			TypeDefinitionVersionID: src.TypeDefinitionVersionID,
			Ordinal:                 len(targetAttrs) + i,
			Required:                src.Required,
		}
		if err := s.attrRepo.Create(ctx, attr); err != nil {
			return nil, err
		}
	}

	return newVersion, nil
}

// EditAttribute edits an attribute on an entity type, creating a new version (copy-on-write).
// Only non-nil fields are updated. The attribute is identified by currentName.
func (s *AttributeService) EditAttribute(ctx context.Context, entityTypeID, currentName string, newName, newDesc *string, newTypeDefVersionID *string, newRequired *bool) (*models.EntityTypeVersion, error) {
	if newName != nil && models.IsSystemAttributeName(*newName) {
		return nil, domainerrors.NewValidation("attribute name \"" + *newName + "\" is reserved for system attributes")
	}
	// Validate type definition version reference if changing
	if newTypeDefVersionID != nil && *newTypeDefVersionID != "" {
		if _, err := s.tdvRepo.GetByID(ctx, *newTypeDefVersionID); err != nil {
			return nil, domainerrors.NewValidation("invalid type_definition_version_id: " + *newTypeDefVersionID)
		}
	}

	latest, err := s.etvRepo.GetLatestByEntityType(ctx, entityTypeID)
	if err != nil {
		return nil, err
	}

	attrs, err := s.attrRepo.ListByVersion(ctx, latest.ID)
	if err != nil {
		return nil, err
	}

	// Find the target attribute and check for name conflicts
	var targetFound bool
	for _, a := range attrs {
		if a.Name == currentName {
			targetFound = true
		}
		if newName != nil && a.Name == *newName && a.Name != currentName {
			return nil, domainerrors.NewConflict("Attribute", "attribute name already exists: "+*newName)
		}
	}
	if !targetFound {
		return nil, domainerrors.NewNotFound("Attribute", currentName)
	}

	// Check shared namespace: renamed attribute must not conflict with association names
	if newName != nil && *newName != currentName {
		assocs, err := s.assocRepo.ListByVersion(ctx, latest.ID)
		if err != nil {
			return nil, err
		}
		for _, a := range assocs {
			if a.Name == *newName {
				return nil, domainerrors.NewConflict("Attribute", "name conflicts with association: "+*newName)
			}
		}
	}

	// Create new version with copy-on-write
	newVersion := &models.EntityTypeVersion{
		ID:           uuid.Must(uuid.NewV7()).String(),
		EntityTypeID: entityTypeID,
		Version:      latest.Version + 1,
		Description:  latest.Description,
	}
	if err := s.etvRepo.Create(ctx, newVersion); err != nil {
		return nil, err
	}

	if err := s.attrRepo.BulkCopyToVersion(ctx, latest.ID, newVersion.ID); err != nil {
		return nil, err
	}
	if err := s.assocRepo.BulkCopyToVersion(ctx, latest.ID, newVersion.ID); err != nil {
		return nil, err
	}

	// Find the copied attribute in the new version and update it
	newAttrs, err := s.attrRepo.ListByVersion(ctx, newVersion.ID)
	if err != nil {
		return nil, err
	}
	for _, a := range newAttrs {
		if a.Name == currentName {
			if newName != nil {
				a.Name = *newName
			}
			if newDesc != nil {
				a.Description = *newDesc
			}
			if newTypeDefVersionID != nil {
				a.TypeDefinitionVersionID = *newTypeDefVersionID
			}
			if newRequired != nil {
				a.Required = *newRequired
			}
			if err := s.attrRepo.Update(ctx, a); err != nil {
				return nil, err
			}
			return newVersion, nil
		}
	}

	return nil, domainerrors.NewNotFound("Attribute", currentName)
}

// ListAttributes returns the attributes for the latest version of the given entity type.
func (s *AttributeService) ListAttributes(ctx context.Context, entityTypeID string) ([]*models.Attribute, error) {
	latest, err := s.etvRepo.GetLatestByEntityType(ctx, entityTypeID)
	if err != nil {
		return nil, err
	}
	return s.attrRepo.ListByVersion(ctx, latest.ID)
}

// ReorderAttributes reorders attributes within the latest version.
func (s *AttributeService) ReorderAttributes(ctx context.Context, entityTypeID string, orderedIDs []string) error {
	latest, err := s.etvRepo.GetLatestByEntityType(ctx, entityTypeID)
	if err != nil {
		return err
	}
	return s.attrRepo.Reorder(ctx, latest.ID, orderedIDs)
}
