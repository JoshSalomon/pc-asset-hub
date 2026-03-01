package meta

import (
	"context"

	"github.com/google/uuid"

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
func (s *AssociationService) CreateAssociation(ctx context.Context, sourceEntityTypeID, targetEntityTypeID string, assocType models.AssociationType, sourceRole, targetRole string) (*models.EntityTypeVersion, error) {
	// Cycle detection for containment
	if assocType == models.AssociationTypeContainment {
		if err := validation.CheckContainmentCycle(ctx, s.assocRepo, sourceEntityTypeID, targetEntityTypeID); err != nil {
			return nil, err
		}
	}

	latest, err := s.etvRepo.GetLatestByEntityType(ctx, sourceEntityTypeID)
	if err != nil {
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
		TargetEntityTypeID:  targetEntityTypeID,
		Type:                assocType,
		SourceRole:          sourceRole,
		TargetRole:          targetRole,
	}
	if err := s.assocRepo.Create(ctx, assoc); err != nil {
		return nil, err
	}

	return newVersion, nil
}

// DeleteAssociation removes an association, creating a new version without it.
func (s *AssociationService) DeleteAssociation(ctx context.Context, entityTypeID, associationID string) (*models.EntityTypeVersion, error) {
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

	// Find the copied association and delete it
	newAssocs, err := s.assocRepo.ListByVersion(ctx, newVersion.ID)
	if err != nil {
		return nil, err
	}

	// Find the association in the old version to match by properties
	oldAssoc, err := s.assocRepo.GetByID(ctx, associationID)
	if err != nil {
		return nil, err
	}

	for _, a := range newAssocs {
		if a.TargetEntityTypeID == oldAssoc.TargetEntityTypeID && a.Type == oldAssoc.Type {
			if err := s.assocRepo.Delete(ctx, a.ID); err != nil {
				return nil, err
			}
			return newVersion, nil
		}
	}

	return newVersion, nil
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
