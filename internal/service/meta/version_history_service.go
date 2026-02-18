package meta

import (
	"context"

	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository"
)

// VersionDiffItem represents a single change in a version diff.
type VersionDiffItem struct {
	Name       string // attribute or association name
	ChangeType string // "added", "removed", "modified"
	Category   string // "attribute" or "association"
	OldValue   string // for modified items
	NewValue   string // for modified items
}

// VersionDiff holds the comparison result between two versions.
type VersionDiff struct {
	FromVersion int
	ToVersion   int
	Changes     []VersionDiffItem
}

type VersionHistoryService struct {
	etvRepo   repository.EntityTypeVersionRepository
	attrRepo  repository.AttributeRepository
	assocRepo repository.AssociationRepository
}

func NewVersionHistoryService(
	etvRepo repository.EntityTypeVersionRepository,
	attrRepo repository.AttributeRepository,
	assocRepo repository.AssociationRepository,
) *VersionHistoryService {
	return &VersionHistoryService{
		etvRepo:   etvRepo,
		attrRepo:  attrRepo,
		assocRepo: assocRepo,
	}
}

func (s *VersionHistoryService) GetVersionHistory(ctx context.Context, entityTypeID string) ([]*models.EntityTypeVersion, error) {
	return s.etvRepo.ListByEntityType(ctx, entityTypeID)
}

func (s *VersionHistoryService) CompareVersions(ctx context.Context, entityTypeID string, v1, v2 int) (*VersionDiff, error) {
	etv1, err := s.etvRepo.GetByEntityTypeAndVersion(ctx, entityTypeID, v1)
	if err != nil {
		return nil, err
	}
	etv2, err := s.etvRepo.GetByEntityTypeAndVersion(ctx, entityTypeID, v2)
	if err != nil {
		return nil, err
	}

	attrs1, err := s.attrRepo.ListByVersion(ctx, etv1.ID)
	if err != nil {
		return nil, err
	}
	attrs2, err := s.attrRepo.ListByVersion(ctx, etv2.ID)
	if err != nil {
		return nil, err
	}

	assocs1, err := s.assocRepo.ListByVersion(ctx, etv1.ID)
	if err != nil {
		return nil, err
	}
	assocs2, err := s.assocRepo.ListByVersion(ctx, etv2.ID)
	if err != nil {
		return nil, err
	}

	diff := &VersionDiff{FromVersion: v1, ToVersion: v2}

	// Diff attributes
	attrMap1 := make(map[string]*models.Attribute)
	for _, a := range attrs1 {
		attrMap1[a.Name] = a
	}
	attrMap2 := make(map[string]*models.Attribute)
	for _, a := range attrs2 {
		attrMap2[a.Name] = a
	}

	for name, a2 := range attrMap2 {
		a1, exists := attrMap1[name]
		if !exists {
			diff.Changes = append(diff.Changes, VersionDiffItem{
				Name: name, ChangeType: "added", Category: "attribute",
			})
		} else if a1.Type != a2.Type || a1.Description != a2.Description || a1.EnumID != a2.EnumID {
			diff.Changes = append(diff.Changes, VersionDiffItem{
				Name: name, ChangeType: "modified", Category: "attribute",
				OldValue: string(a1.Type), NewValue: string(a2.Type),
			})
		}
	}
	for name := range attrMap1 {
		if _, exists := attrMap2[name]; !exists {
			diff.Changes = append(diff.Changes, VersionDiffItem{
				Name: name, ChangeType: "removed", Category: "attribute",
			})
		}
	}

	// Diff associations
	assocKey := func(a *models.Association) string {
		return a.TargetEntityTypeID + ":" + string(a.Type)
	}
	assocMap1 := make(map[string]*models.Association)
	for _, a := range assocs1 {
		assocMap1[assocKey(a)] = a
	}
	assocMap2 := make(map[string]*models.Association)
	for _, a := range assocs2 {
		assocMap2[assocKey(a)] = a
	}

	for key := range assocMap2 {
		if _, exists := assocMap1[key]; !exists {
			diff.Changes = append(diff.Changes, VersionDiffItem{
				Name: key, ChangeType: "added", Category: "association",
			})
		}
	}
	for key := range assocMap1 {
		if _, exists := assocMap2[key]; !exists {
			diff.Changes = append(diff.Changes, VersionDiffItem{
				Name: key, ChangeType: "removed", Category: "association",
			})
		}
	}

	return diff, nil
}
