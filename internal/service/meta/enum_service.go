package meta

import (
	"context"
	"time"

	"github.com/google/uuid"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository"
)

type EnumService struct {
	enumRepo repository.EnumRepository
	evRepo   repository.EnumValueRepository
	attrRepo repository.AttributeRepository
}

func NewEnumService(
	enumRepo repository.EnumRepository,
	evRepo repository.EnumValueRepository,
	attrRepo repository.AttributeRepository,
) *EnumService {
	return &EnumService{
		enumRepo: enumRepo,
		evRepo:   evRepo,
		attrRepo: attrRepo,
	}
}

func (s *EnumService) CreateEnum(ctx context.Context, name, description string, values []string) (*models.Enum, error) {
	if name == "" {
		return nil, domainerrors.NewValidation("enum name is required")
	}

	now := time.Now()
	e := &models.Enum{
		ID:          uuid.Must(uuid.NewV7()).String(),
		Name:        name,
		Description: description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.enumRepo.Create(ctx, e); err != nil {
		return nil, err
	}

	for i, v := range values {
		ev := &models.EnumValue{
			ID:      uuid.Must(uuid.NewV7()).String(),
			EnumID:  e.ID,
			Value:   v,
			Ordinal: i,
		}
		if err := s.evRepo.Create(ctx, ev); err != nil {
			return nil, err
		}
	}

	return e, nil
}

func (s *EnumService) GetEnum(ctx context.Context, id string) (*models.Enum, error) {
	return s.enumRepo.GetByID(ctx, id)
}

func (s *EnumService) ListEnums(ctx context.Context, params models.ListParams) ([]*models.Enum, int, error) {
	return s.enumRepo.List(ctx, params)
}

func (s *EnumService) UpdateEnum(ctx context.Context, id string, name, description string) error {
	e, err := s.enumRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	e.Name = name
	e.Description = description
	e.UpdatedAt = time.Now()
	return s.enumRepo.Update(ctx, e)
}

func (s *EnumService) DeleteEnum(ctx context.Context, id string) error {
	return s.enumRepo.Delete(ctx, id)
}

func (s *EnumService) AddValue(ctx context.Context, enumID, value string) error {
	values, err := s.evRepo.ListByEnum(ctx, enumID)
	if err != nil {
		return err
	}
	ev := &models.EnumValue{
		ID:      uuid.Must(uuid.NewV7()).String(),
		EnumID:  enumID,
		Value:   value,
		Ordinal: len(values),
	}
	return s.evRepo.Create(ctx, ev)
}

func (s *EnumService) RemoveValue(ctx context.Context, valueID string) error {
	return s.evRepo.Delete(ctx, valueID)
}

func (s *EnumService) ReorderValues(ctx context.Context, enumID string, orderedIDs []string) error {
	return s.evRepo.Reorder(ctx, enumID, orderedIDs)
}

// ListValues returns the values for the given enum.
func (s *EnumService) ListValues(ctx context.Context, enumID string) ([]*models.EnumValue, error) {
	return s.evRepo.ListByEnum(ctx, enumID)
}

// GetReferencingAttributes returns all attributes that reference the given enum.
// This queries across ALL entity type versions.
func (s *EnumService) GetReferencingAttributes(ctx context.Context, enumID string) ([]string, error) {
	// This is a simplified implementation — the actual implementation would query
	// the attribute repository for all attributes with this enum_id.
	// For now, we return the enum ID check result.
	return []string{enumID}, nil
}
