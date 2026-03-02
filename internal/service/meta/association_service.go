package meta

import (
	"context"

	"fmt"

	"github.com/google/uuid"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository"
	"github.com/project-catalyst/pc-asset-hub/internal/service/validation"
)

type AssociationService struct {
	assocRepo repository.AssociationRepository
	etvRepo   repository.EntityTypeVersionRepository
	attrRepo  repository.AttributeRepository
}

func NewAssociationService(
	assocRepo repository.AssociationRepository,
	etvRepo repository.EntityTypeVersionRepository,
	attrRepo repository.AttributeRepository,
) *AssociationService {
	return &AssociationService{
		assocRepo: assocRepo,
		etvRepo:   etvRepo,
		attrRepo:  attrRepo,
	}
}

// CreateAssociation adds an association, creating a new entity type version.
func (s *AssociationService) CreateAssociation(ctx context.Context, sourceEntityTypeID, targetEntityTypeID string, assocType models.AssociationType, name, sourceRole, targetRole, sourceCardinality, targetCardinality string) (*models.EntityTypeVersion, error) {
	// Validate name
	if name == "" {
		return nil, domainerrors.NewValidation("association name is required")
	}

	// Validate cardinality
	if err := validation.ValidateCardinality(sourceCardinality); err != nil {
		return nil, domainerrors.NewValidation(fmt.Sprintf("source_cardinality: %s", err))
	}
	if err := validation.ValidateCardinality(targetCardinality); err != nil {
		return nil, domainerrors.NewValidation(fmt.Sprintf("target_cardinality: %s", err))
	}

	// Containment: source cardinality (container side) must be "1" or "0..1"
	if assocType == models.AssociationTypeContainment {
		sc := validation.NormalizeSourceCardinality(sourceCardinality, true)
		if sc != "1" && sc != "0..1" {
			return nil, domainerrors.NewValidation("source_cardinality: containment source must be \"1\" or \"0..1\"")
		}
		if err := validation.CheckContainmentCycle(ctx, s.assocRepo, sourceEntityTypeID, targetEntityTypeID); err != nil {
			return nil, err
		}
	}

	latest, err := s.etvRepo.GetLatestByEntityType(ctx, sourceEntityTypeID)
	if err != nil {
		return nil, err
	}

	// Check shared namespace: name must not conflict with attributes or existing associations
	if err := s.checkNameConflict(ctx, latest.ID, name, ""); err != nil {
		return nil, err
	}

	newVersion := &models.EntityTypeVersion{
		ID:           uuid.Must(uuid.NewV7()).String(),
		EntityTypeID: sourceEntityTypeID,
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

	assoc := &models.Association{
		ID:                  uuid.Must(uuid.NewV7()).String(),
		EntityTypeVersionID: newVersion.ID,
		Name:                name,
		TargetEntityTypeID:  targetEntityTypeID,
		Type:                assocType,
		SourceRole:          sourceRole,
		TargetRole:          targetRole,
		SourceCardinality:   validation.NormalizeSourceCardinality(sourceCardinality, assocType == models.AssociationTypeContainment),
		TargetCardinality:   validation.NormalizeCardinality(targetCardinality),
	}
	if err := s.assocRepo.Create(ctx, assoc); err != nil {
		return nil, err
	}

	return newVersion, nil
}

// EditAssociation edits an association's roles, cardinality, and name, creating a new version (copy-on-write).
// Only non-nil fields are updated. The association is identified by currentName.
func (s *AssociationService) EditAssociation(ctx context.Context, entityTypeID, currentName string, newName, sourceRole, targetRole, sourceCardinality, targetCardinality *string) (*models.EntityTypeVersion, error) {
	latest, err := s.etvRepo.GetLatestByEntityType(ctx, entityTypeID)
	if err != nil {
		return nil, err
	}

	// Find the association by name in the current version
	assocs, err := s.assocRepo.ListByVersion(ctx, latest.ID)
	if err != nil {
		return nil, err
	}
	var oldAssoc *models.Association
	for _, a := range assocs {
		if a.Name == currentName {
			oldAssoc = a
			break
		}
	}
	if oldAssoc == nil {
		return nil, domainerrors.NewNotFound("Association", currentName)
	}

	// Validate cardinality if provided
	if sourceCardinality != nil {
		if err := validation.ValidateCardinality(*sourceCardinality); err != nil {
			return nil, domainerrors.NewValidation(fmt.Sprintf("source_cardinality: %s", err))
		}
		if oldAssoc.Type == models.AssociationTypeContainment {
			sc := validation.NormalizeSourceCardinality(*sourceCardinality, true)
			if sc != "1" && sc != "0..1" {
				return nil, domainerrors.NewValidation("source_cardinality: containment source must be \"1\" or \"0..1\"")
			}
		}
	}
	if targetCardinality != nil {
		if err := validation.ValidateCardinality(*targetCardinality); err != nil {
			return nil, domainerrors.NewValidation(fmt.Sprintf("target_cardinality: %s", err))
		}
	}

	// Check name conflict if renaming
	if newName != nil && *newName != currentName {
		if err := s.checkNameConflict(ctx, latest.ID, *newName, currentName); err != nil {
			return nil, err
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

	// Find the copied association by name in the new version and update it
	newAssocs, err := s.assocRepo.ListByVersion(ctx, newVersion.ID)
	if err != nil {
		return nil, err
	}
	for _, a := range newAssocs {
		if a.Name == currentName {
			if newName != nil {
				a.Name = *newName
			}
			if sourceRole != nil {
				a.SourceRole = *sourceRole
			}
			if targetRole != nil {
				a.TargetRole = *targetRole
			}
			if sourceCardinality != nil {
				a.SourceCardinality = validation.NormalizeSourceCardinality(*sourceCardinality, a.Type == models.AssociationTypeContainment)
			}
			if targetCardinality != nil {
				a.TargetCardinality = validation.NormalizeCardinality(*targetCardinality)
			}
			if err := s.assocRepo.Update(ctx, a); err != nil {
				return nil, err
			}
			return newVersion, nil
		}
	}

	return nil, domainerrors.NewNotFound("Association", currentName)
}

// checkNameConflict checks that a name doesn't conflict with existing attributes or associations
// in the given version. excludeName is the current name being renamed (to allow keeping the same name).
func (s *AssociationService) checkNameConflict(ctx context.Context, versionID, name, excludeName string) error {
	// Check against attributes
	attrs, err := s.attrRepo.ListByVersion(ctx, versionID)
	if err != nil {
		return err
	}
	for _, a := range attrs {
		if a.Name == name {
			return domainerrors.NewConflict("Association", "name conflicts with attribute: "+name)
		}
	}
	// Check against existing associations
	assocs, err := s.assocRepo.ListByVersion(ctx, versionID)
	if err != nil {
		return err
	}
	for _, a := range assocs {
		if a.Name == name && a.Name != excludeName {
			return domainerrors.NewConflict("Association", "association name already exists: "+name)
		}
	}
	return nil
}

// DeleteAssociation removes an association by name, creating a new version without it.
func (s *AssociationService) DeleteAssociation(ctx context.Context, entityTypeID, name string) (*models.EntityTypeVersion, error) {
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

	// Find the copied association by name and delete it
	newAssocs, err := s.assocRepo.ListByVersion(ctx, newVersion.ID)
	if err != nil {
		return nil, err
	}

	for _, a := range newAssocs {
		if a.Name == name {
			if err := s.assocRepo.Delete(ctx, a.ID); err != nil {
				return nil, err
			}
			return newVersion, nil
		}
	}

	return nil, domainerrors.NewNotFound("Association", name)
}

// DirectedAssociation wraps an Association with direction metadata.
type DirectedAssociation struct {
	*models.Association
	Direction          string // "outgoing" or "incoming"
	SourceEntityTypeID string // set for incoming associations
}

// ListAssociations returns associations owned by this entity type (outgoing).
func (s *AssociationService) ListAssociations(ctx context.Context, entityTypeID string) ([]*models.Association, error) {
	latest, err := s.etvRepo.GetLatestByEntityType(ctx, entityTypeID)
	if err != nil {
		return nil, err
	}
	return s.assocRepo.ListByVersion(ctx, latest.ID)
}

// ListAllAssociations returns both outgoing (owned) and incoming (targeted) associations
// for an entity type, with direction metadata.
func (s *AssociationService) ListAllAssociations(ctx context.Context, entityTypeID string) ([]*DirectedAssociation, error) {
	latest, err := s.etvRepo.GetLatestByEntityType(ctx, entityTypeID)
	if err != nil {
		return nil, err
	}

	// Outgoing: associations owned by this entity type
	outgoing, err := s.assocRepo.ListByVersion(ctx, latest.ID)
	if err != nil {
		return nil, err
	}

	// Incoming: associations from other entity types that target this one
	incoming, err := s.assocRepo.ListByTargetEntityType(ctx, entityTypeID)
	if err != nil {
		return nil, err
	}

	var result []*DirectedAssociation
	for _, a := range outgoing {
		result = append(result, &DirectedAssociation{
			Association: a,
			Direction:   "outgoing",
		})
	}

	// For incoming, resolve the source entity type ID from the version
	for _, a := range incoming {
		etv, err := s.etvRepo.GetByID(ctx, a.EntityTypeVersionID)
		if err != nil {
			return nil, err
		}
		// Skip if this is actually an outgoing association (same entity type targeting itself)
		if etv.EntityTypeID == entityTypeID {
			continue
		}
		// Only include the latest version's association from each source entity type
		// to avoid showing duplicate associations from old versions
		latestSrc, err := s.etvRepo.GetLatestByEntityType(ctx, etv.EntityTypeID)
		if err != nil {
			return nil, err
		}
		if a.EntityTypeVersionID != latestSrc.ID {
			continue
		}
		result = append(result, &DirectedAssociation{
			Association:        a,
			Direction:          "incoming",
			SourceEntityTypeID: etv.EntityTypeID,
		})
	}

	return result, nil
}
