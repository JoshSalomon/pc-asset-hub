package meta

import (
	"context"
	"time"

	"github.com/google/uuid"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository"
)

type EntityTypeService struct {
	etRepo    repository.EntityTypeRepository
	etvRepo   repository.EntityTypeVersionRepository
	attrRepo  repository.AttributeRepository
	assocRepo repository.AssociationRepository
}

func NewEntityTypeService(
	etRepo repository.EntityTypeRepository,
	etvRepo repository.EntityTypeVersionRepository,
	attrRepo repository.AttributeRepository,
	assocRepo repository.AssociationRepository,
) *EntityTypeService {
	return &EntityTypeService{
		etRepo:    etRepo,
		etvRepo:   etvRepo,
		attrRepo:  attrRepo,
		assocRepo: assocRepo,
	}
}

func (s *EntityTypeService) CreateEntityType(ctx context.Context, name, description string) (*models.EntityType, *models.EntityTypeVersion, error) {
	if name == "" {
		return nil, nil, domainerrors.NewValidation("name is required")
	}

	now := time.Now()
	et := &models.EntityType{
		ID:        uuid.Must(uuid.NewV7()).String(),
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.etRepo.Create(ctx, et); err != nil {
		return nil, nil, err
	}

	etv := &models.EntityTypeVersion{
		ID:           uuid.Must(uuid.NewV7()).String(),
		EntityTypeID: et.ID,
		Version:      1,
		Description:  description,
		CreatedAt:    now,
	}
	if err := s.etvRepo.Create(ctx, etv); err != nil {
		return nil, nil, err
	}

	return et, etv, nil
}

func (s *EntityTypeService) GetEntityType(ctx context.Context, id string) (*models.EntityType, error) {
	return s.etRepo.GetByID(ctx, id)
}

func (s *EntityTypeService) ListEntityTypes(ctx context.Context, params models.ListParams) ([]*models.EntityType, int, error) {
	return s.etRepo.List(ctx, params)
}

// UpdateEntityType creates a new version with copy-on-write semantics.
// All attributes and associations from the current latest version are copied to the new version.
func (s *EntityTypeService) UpdateEntityType(ctx context.Context, id, description string) (*models.EntityTypeVersion, error) {
	latest, err := s.etvRepo.GetLatestByEntityType(ctx, id)
	if err != nil {
		return nil, err
	}

	newVersion := &models.EntityTypeVersion{
		ID:           uuid.Must(uuid.NewV7()).String(),
		EntityTypeID: id,
		Version:      latest.Version + 1,
		Description:  description,
		CreatedAt:    time.Now(),
	}
	if err := s.etvRepo.Create(ctx, newVersion); err != nil {
		return nil, err
	}

	// Copy-on-write: copy attributes and associations from previous version
	if err := s.attrRepo.BulkCopyToVersion(ctx, latest.ID, newVersion.ID); err != nil {
		return nil, err
	}
	if err := s.assocRepo.BulkCopyToVersion(ctx, latest.ID, newVersion.ID); err != nil {
		return nil, err
	}

	// Update the entity type's UpdatedAt
	et, err := s.etRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	et.UpdatedAt = time.Now()
	if err := s.etRepo.Update(ctx, et); err != nil {
		return nil, err
	}

	return newVersion, nil
}

func (s *EntityTypeService) DeleteEntityType(ctx context.Context, id string) error {
	return s.etRepo.Delete(ctx, id)
}

// CopyEntityType creates a new entity type by copying attributes from the source version.
// Associations are NOT copied.
func (s *EntityTypeService) CopyEntityType(ctx context.Context, sourceEntityTypeID string, sourceVersion int, newName string) (*models.EntityType, *models.EntityTypeVersion, error) {
	if newName == "" {
		return nil, nil, domainerrors.NewValidation("name is required")
	}

	// Get source version
	sourceETV, err := s.etvRepo.GetByEntityTypeAndVersion(ctx, sourceEntityTypeID, sourceVersion)
	if err != nil {
		return nil, nil, err
	}

	// Create new entity type
	now := time.Now()
	newET := &models.EntityType{
		ID:        uuid.Must(uuid.NewV7()).String(),
		Name:      newName,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.etRepo.Create(ctx, newET); err != nil {
		return nil, nil, err
	}

	// Create V1 for the new type
	newETV := &models.EntityTypeVersion{
		ID:           uuid.Must(uuid.NewV7()).String(),
		EntityTypeID: newET.ID,
		Version:      1,
		Description:  "Copied from " + sourceEntityTypeID,
		CreatedAt:    now,
	}
	if err := s.etvRepo.Create(ctx, newETV); err != nil {
		return nil, nil, err
	}

	// Copy attributes (but NOT associations)
	if err := s.attrRepo.BulkCopyToVersion(ctx, sourceETV.ID, newETV.ID); err != nil {
		return nil, nil, err
	}

	return newET, newETV, nil
}
