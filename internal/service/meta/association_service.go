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

func (s *AssociationService) ListAssociations(ctx context.Context, entityTypeID string) ([]*models.Association, error) {
	latest, err := s.etvRepo.GetLatestByEntityType(ctx, entityTypeID)
	if err != nil {
		return nil, err
	}
	return s.assocRepo.ListByVersion(ctx, latest.ID)
}
