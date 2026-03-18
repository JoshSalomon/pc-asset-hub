package operational

import (
	"context"
	"time"

	"github.com/google/uuid"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository"
)

type EntityInstanceService struct {
	instRepo repository.EntityInstanceRepository
	iavRepo  repository.InstanceAttributeValueRepository
	attrRepo repository.AttributeRepository
	cvRepo   repository.CatalogVersionRepository
	linkRepo repository.AssociationLinkRepository
}

func NewEntityInstanceService(
	instRepo repository.EntityInstanceRepository,
	iavRepo repository.InstanceAttributeValueRepository,
	attrRepo repository.AttributeRepository,
	cvRepo repository.CatalogVersionRepository,
	linkRepo repository.AssociationLinkRepository,
) *EntityInstanceService {
	return &EntityInstanceService{
		instRepo: instRepo,
		iavRepo:  iavRepo,
		attrRepo: attrRepo,
		cvRepo:   cvRepo,
		linkRepo: linkRepo,
	}
}

func (s *EntityInstanceService) CreateInstance(ctx context.Context, entityTypeID, catalogVersionID, parentInstanceID, name, description string, attributeValues map[string]interface{}) (*models.EntityInstance, error) {
	// Verify catalog version exists
	if _, err := s.cvRepo.GetByID(ctx, catalogVersionID); err != nil {
		return nil, err
	}

	now := time.Now()
	inst := &models.EntityInstance{
		ID:               uuid.Must(uuid.NewV7()).String(),
		EntityTypeID:     entityTypeID,
		CatalogID:        catalogVersionID,
		ParentInstanceID: parentInstanceID,
		Name:             name,
		Description:      description,
		Version:          1,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.instRepo.Create(ctx, inst); err != nil {
		return nil, err
	}

	return inst, nil
}

func (s *EntityInstanceService) GetInstance(ctx context.Context, id string) (*models.EntityInstance, error) {
	return s.instRepo.GetByID(ctx, id)
}

func (s *EntityInstanceService) UpdateInstance(ctx context.Context, id string, currentVersion int) (*models.EntityInstance, error) {
	inst, err := s.instRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Optimistic locking
	if inst.Version != currentVersion {
		return nil, domainerrors.NewConflict("EntityInstance", "version mismatch: expected "+string(rune(currentVersion+'0'))+" but got "+string(rune(inst.Version+'0')))
	}

	inst.Version++
	inst.UpdatedAt = time.Now()

	if err := s.instRepo.Update(ctx, inst); err != nil {
		return nil, err
	}

	return inst, nil
}

func (s *EntityInstanceService) DeleteInstance(ctx context.Context, id string) error {
	return s.instRepo.SoftDelete(ctx, id)
}

func (s *EntityInstanceService) ListInstances(ctx context.Context, entityTypeID, catalogVersionID string, params models.ListParams) ([]*models.EntityInstance, int, error) {
	return s.instRepo.List(ctx, entityTypeID, catalogVersionID, params)
}

// CascadeDelete deletes an instance and all its contained children recursively.
func (s *EntityInstanceService) CascadeDelete(ctx context.Context, id string) error {
	// Get all children
	children, _, err := s.instRepo.ListByParent(ctx, id, models.ListParams{Limit: 1000})
	if err != nil {
		return err
	}

	// Recursively delete children first
	for _, child := range children {
		if err := s.CascadeDelete(ctx, child.ID); err != nil {
			return err
		}
	}

	// Delete self
	return s.instRepo.SoftDelete(ctx, id)
}

// CreateContainedInstance creates an instance within a parent.
func (s *EntityInstanceService) CreateContainedInstance(ctx context.Context, parentID, entityTypeID, catalogVersionID, name, description string) (*models.EntityInstance, error) {
	// Verify parent exists
	if _, err := s.instRepo.GetByID(ctx, parentID); err != nil {
		return nil, err
	}

	return s.CreateInstance(ctx, entityTypeID, catalogVersionID, parentID, name, description, nil)
}

// ListContainedInstances lists children of a parent.
func (s *EntityInstanceService) ListContainedInstances(ctx context.Context, parentID string, params models.ListParams) ([]*models.EntityInstance, int, error) {
	return s.instRepo.ListByParent(ctx, parentID, params)
}

// GetForwardReferences returns all forward reference links from an instance.
func (s *EntityInstanceService) GetForwardReferences(ctx context.Context, instanceID string) ([]*models.AssociationLink, error) {
	return s.linkRepo.GetForwardRefs(ctx, instanceID)
}

// GetReverseReferences returns all reverse reference links to an instance.
func (s *EntityInstanceService) GetReverseReferences(ctx context.Context, instanceID string) ([]*models.AssociationLink, error) {
	return s.linkRepo.GetReverseRefs(ctx, instanceID)
}
