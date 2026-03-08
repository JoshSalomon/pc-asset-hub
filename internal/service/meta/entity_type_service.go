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
	pinRepo   repository.CatalogVersionPinRepository
	cvRepo    repository.CatalogVersionRepository
	enumRepo  repository.EnumRepository
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

// WithEnumRepo adds an enum repository for resolving enum names in snapshots.
func WithEnumRepo(svc *EntityTypeService, enumRepo repository.EnumRepository) *EntityTypeService {
	svc.enumRepo = enumRepo
	return svc
}

// WithCatalogRepos adds catalog version repositories needed for rename operations.
func WithCatalogRepos(svc *EntityTypeService, pinRepo repository.CatalogVersionPinRepository, cvRepo repository.CatalogVersionRepository) *EntityTypeService {
	svc.pinRepo = pinRepo
	svc.cvRepo = cvRepo
	return svc
}

// RenameResult holds the result of a rename operation.
type RenameResult struct {
	EntityType  *models.EntityType
	WasDeepCopy bool
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

// RenameEntityType renames an entity type with context-sensitive behavior.
// If the entity type is not in any catalog version, or is only in one development-stage CV,
// it performs a simple rename. Otherwise, if deepCopyAllowed is false, it returns a
// DeepCopyRequired error. If deepCopyAllowed is true, it creates a new entity type with
// the new name (deep copy).
func (s *EntityTypeService) RenameEntityType(ctx context.Context, id, newName string, deepCopyAllowed bool) (*RenameResult, error) {
	if newName == "" {
		return nil, domainerrors.NewValidation("name is required")
	}

	// Check name uniqueness
	existing, err := s.etRepo.GetByName(ctx, newName)
	if err == nil && existing != nil {
		return nil, domainerrors.NewConflict("EntityType", "name already exists: "+newName)
	}
	if err != nil && !domainerrors.IsNotFound(err) {
		return nil, err
	}

	et, err := s.etRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Determine if simple rename or deep copy is needed
	needsDeepCopy, err := s.requiresDeepCopy(ctx, id)
	if err != nil {
		return nil, err
	}

	if !needsDeepCopy {
		// Simple rename
		et.Name = newName
		et.UpdatedAt = time.Now()
		if err := s.etRepo.Update(ctx, et); err != nil {
			return nil, err
		}
		return &RenameResult{EntityType: et, WasDeepCopy: false}, nil
	}

	// Deep copy required
	if !deepCopyAllowed {
		return nil, domainerrors.NewDeepCopyRequired("entity type is referenced by catalog versions in testing/production or multiple catalog versions")
	}

	// Get latest version for deep copy
	latest, err := s.etvRepo.GetLatestByEntityType(ctx, id)
	if err != nil {
		return nil, err
	}

	// Use existing CopyEntityType logic
	newET, _, err := s.CopyEntityType(ctx, id, latest.Version, newName)
	if err != nil {
		return nil, err
	}

	return &RenameResult{EntityType: newET, WasDeepCopy: true}, nil
}

// requiresDeepCopy checks if an entity type rename requires a deep copy.
// Returns true if the entity type is in multiple CVs or any non-development CV.
func (s *EntityTypeService) requiresDeepCopy(ctx context.Context, entityTypeID string) (bool, error) {
	if s.pinRepo == nil || s.cvRepo == nil {
		// Without catalog repos, assume no CVs reference this entity type
		return false, nil
	}

	versions, err := s.etvRepo.ListByEntityType(ctx, entityTypeID)
	if err != nil {
		return false, err
	}
	if len(versions) == 0 {
		return false, nil
	}

	etvIDs := make([]string, len(versions))
	for i, v := range versions {
		etvIDs[i] = v.ID
	}

	pins, err := s.pinRepo.ListByEntityTypeVersionIDs(ctx, etvIDs)
	if err != nil {
		return false, err
	}
	if len(pins) == 0 {
		return false, nil
	}

	// Multiple CVs → deep copy
	cvIDs := make(map[string]bool)
	for _, pin := range pins {
		cvIDs[pin.CatalogVersionID] = true
	}
	if len(cvIDs) > 1 {
		return true, nil
	}

	// Single CV — check if it's in development
	for cvID := range cvIDs {
		cv, err := s.cvRepo.GetByID(ctx, cvID)
		if err != nil {
			return false, err
		}
		if cv.LifecycleStage != models.LifecycleStageDevelopment {
			return true, nil
		}
	}

	return false, nil
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

// ContainmentTreeNode represents an entity type in the containment tree,
// with its available versions and contained children.
type ContainmentTreeNode struct {
	EntityType    *models.EntityType
	Versions      []*models.EntityTypeVersion
	LatestVersion int
	Children      []*ContainmentTreeNode
}

// GetContainmentTree returns all entity types organized as a containment tree.
// Root entity types (not contained by any other) appear at the top level.
// Contained entity types appear as children of their parent.
func (s *EntityTypeService) GetContainmentTree(ctx context.Context) ([]*ContainmentTreeNode, error) {
	// Get all entity types
	allET, _, err := s.etRepo.List(ctx, models.ListParams{Limit: 10000})
	if err != nil {
		return nil, err
	}
	if len(allET) == 0 {
		return []*ContainmentTreeNode{}, nil
	}

	// Get containment edges
	edges, err := s.assocRepo.GetContainmentGraph(ctx)
	if err != nil {
		return nil, err
	}

	// Build nodes map with versions
	nodes := make(map[string]*ContainmentTreeNode, len(allET))
	for _, et := range allET {
		versions, err := s.etvRepo.ListByEntityType(ctx, et.ID)
		if err != nil {
			return nil, err
		}
		latestVersion := 0
		for _, v := range versions {
			if v.Version > latestVersion {
				latestVersion = v.Version
			}
		}
		nodes[et.ID] = &ContainmentTreeNode{
			EntityType:    et,
			Versions:      versions,
			LatestVersion: latestVersion,
			Children:      []*ContainmentTreeNode{},
		}
	}

	// Build parent→children map and track which entities are children (deduplicate edges)
	childIDs := make(map[string]bool)
	parentToChildren := make(map[string]map[string]bool)
	for _, edge := range edges {
		if parentToChildren[edge.SourceEntityTypeID] == nil {
			parentToChildren[edge.SourceEntityTypeID] = make(map[string]bool)
		}
		parentToChildren[edge.SourceEntityTypeID][edge.TargetEntityTypeID] = true
		childIDs[edge.TargetEntityTypeID] = true
	}

	// Attach children to parent nodes
	for parentID, childIDSet := range parentToChildren {
		parentNode, ok := nodes[parentID]
		if !ok {
			continue
		}
		for childID := range childIDSet {
			childNode, ok := nodes[childID]
			if !ok {
				continue
			}
			parentNode.Children = append(parentNode.Children, childNode)
		}
	}

	// Collect roots (entities not appearing as target in any edge)
	var roots []*ContainmentTreeNode
	for _, et := range allET {
		if !childIDs[et.ID] {
			roots = append(roots, nodes[et.ID])
		}
	}

	return roots, nil
}

// VersionSnapshot holds the attributes and associations for a specific entity type version,
// with resolved names for enum types and association targets.
type VersionSnapshot struct {
	EntityType            *models.EntityType
	Version               *models.EntityTypeVersion
	Attributes            []*models.Attribute
	Associations          []*DirectedAssociation
	EnumNames             map[string]string // enum_id → enum name
	TargetEntityTypeNames map[string]string // entity_type_id → entity type name
}

// GetVersionSnapshot returns the attributes and associations for a specific entity type version.
func (s *EntityTypeService) GetVersionSnapshot(ctx context.Context, entityTypeID string, version int) (*VersionSnapshot, error) {
	et, err := s.etRepo.GetByID(ctx, entityTypeID)
	if err != nil {
		return nil, err
	}

	etv, err := s.etvRepo.GetByEntityTypeAndVersion(ctx, entityTypeID, version)
	if err != nil {
		return nil, err
	}

	attrs, err := s.attrRepo.ListByVersion(ctx, etv.ID)
	if err != nil {
		return nil, err
	}

	outgoing, err := s.assocRepo.ListByVersion(ctx, etv.ID)
	if err != nil {
		return nil, err
	}

	// Build directed associations: outgoing first
	var directedAssocs []*DirectedAssociation
	for _, a := range outgoing {
		directedAssocs = append(directedAssocs, &DirectedAssociation{
			Association: a,
			Direction:   "outgoing",
		})
	}

	// Incoming: associations from other entity types that target this one
	incoming, err := s.assocRepo.ListByTargetEntityType(ctx, entityTypeID)
	if err != nil {
		return nil, err
	}
	for _, a := range incoming {
		incEtv, err := s.etvRepo.GetByID(ctx, a.EntityTypeVersionID)
		if err != nil {
			return nil, err
		}
		if incEtv.EntityTypeID == entityTypeID {
			continue // skip self-references
		}
		latestSrc, err := s.etvRepo.GetLatestByEntityType(ctx, incEtv.EntityTypeID)
		if err != nil {
			return nil, err
		}
		if a.EntityTypeVersionID != latestSrc.ID {
			continue // skip old versions
		}
		directedAssocs = append(directedAssocs, &DirectedAssociation{
			Association:    a,
			Direction:      "incoming",
			SourceEntityTypeID: incEtv.EntityTypeID,
		})
	}

	// Resolve enum names for enum-type attributes
	enumNames := make(map[string]string)
	if s.enumRepo != nil {
		for _, attr := range attrs {
			if attr.Type == "enum" && attr.EnumID != "" {
				if _, exists := enumNames[attr.EnumID]; !exists {
					if e, err := s.enumRepo.GetByID(ctx, attr.EnumID); err == nil {
						enumNames[attr.EnumID] = e.Name
					}
				}
			}
		}
	}

	// Resolve target entity type names for associations (both outgoing targets and incoming sources)
	targetNames := make(map[string]string)
	for _, da := range directedAssocs {
		if da.Direction == "outgoing" {
			if _, exists := targetNames[da.TargetEntityTypeID]; !exists {
				if targetET, err := s.etRepo.GetByID(ctx, da.TargetEntityTypeID); err == nil {
					targetNames[da.TargetEntityTypeID] = targetET.Name
				}
			}
		} else {
			if _, exists := targetNames[da.SourceEntityTypeID]; !exists {
				if sourceET, err := s.etRepo.GetByID(ctx, da.SourceEntityTypeID); err == nil {
					targetNames[da.SourceEntityTypeID] = sourceET.Name
				}
			}
		}
	}

	return &VersionSnapshot{
		EntityType:            et,
		Version:               etv,
		Attributes:            attrs,
		Associations:          directedAssocs,
		EnumNames:             enumNames,
		TargetEntityTypeNames: targetNames,
	}, nil
}
