package meta

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository"
)

// Role represents a user role for lifecycle permission checks.
type Role string

const (
	RoleRO         Role = "RO"
	RoleRW         Role = "RW"
	RoleAdmin      Role = "Admin"
	RoleSuperAdmin Role = "SuperAdmin"
)

// CatalogWarning represents a warning about a catalog's status during CV promotion.
type CatalogWarning struct {
	CatalogName      string
	ValidationStatus string
}

// PromoteResult contains the result of a CV promotion, including any warnings.
type PromoteResult struct {
	Warnings []CatalogWarning
}

type CatalogVersionService struct {
	cvRepo        repository.CatalogVersionRepository
	pinRepo       repository.CatalogVersionPinRepository
	ltRepo        repository.LifecycleTransitionRepository
	crManager     CatalogVersionCRManager // nil when running outside K8s
	namespace     string
	allowedStages []string // empty means all stages allowed
	etRepo        repository.EntityTypeRepository
	etvRepo       repository.EntityTypeVersionRepository
	catalogRepo   repository.CatalogRepository
	attrRepo      repository.AttributeRepository
	instRepo      repository.EntityInstanceRepository
	iavRepo       repository.InstanceAttributeValueRepository
	txManager     repository.TransactionManager
}

type CVServiceOption func(*CatalogVersionService)

func WithMigrationRepos(attrRepo repository.AttributeRepository, instRepo repository.EntityInstanceRepository, iavRepo repository.InstanceAttributeValueRepository) CVServiceOption {
	return func(s *CatalogVersionService) {
		s.attrRepo = attrRepo
		s.instRepo = instRepo
		s.iavRepo = iavRepo
	}
}

func WithTransactionManager(txm repository.TransactionManager) CVServiceOption {
	return func(s *CatalogVersionService) {
		s.txManager = txm
	}
}

func NewCatalogVersionService(
	cvRepo repository.CatalogVersionRepository,
	pinRepo repository.CatalogVersionPinRepository,
	ltRepo repository.LifecycleTransitionRepository,
	crManager CatalogVersionCRManager,
	namespace string,
	allowedStages []string,
	etRepo repository.EntityTypeRepository,
	etvRepo repository.EntityTypeVersionRepository,
	catalogRepo repository.CatalogRepository,
	opts ...CVServiceOption,
) *CatalogVersionService {
	svc := &CatalogVersionService{
		cvRepo:        cvRepo,
		pinRepo:       pinRepo,
		ltRepo:        ltRepo,
		crManager:     crManager,
		namespace:     namespace,
		allowedStages: allowedStages,
		etRepo:        etRepo,
		etvRepo:       etvRepo,
		catalogRepo:   catalogRepo,
	}
	for _, opt := range opts {
		opt(svc)
	}
	return svc
}

func (s *CatalogVersionService) CreateCatalogVersion(ctx context.Context, label, description string, pins []models.CatalogVersionPin) (*models.CatalogVersion, error) {
	now := time.Now()
	cv := &models.CatalogVersion{
		ID:             uuid.Must(uuid.NewV7()).String(),
		VersionLabel:   label,
		Description:    description,
		LifecycleStage: models.LifecycleStageDevelopment,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.cvRepo.Create(ctx, cv); err != nil {
		return nil, err
	}

	for i := range pins {
		pins[i].ID = uuid.Must(uuid.NewV7()).String()
		pins[i].CatalogVersionID = cv.ID
		if err := s.pinRepo.Create(ctx, &pins[i]); err != nil {
			return nil, err
		}
	}

	// Record initial transition
	if err := s.ltRepo.Create(ctx, &models.LifecycleTransition{
		ID:               uuid.Must(uuid.NewV7()).String(),
		CatalogVersionID: cv.ID,
		ToStage:          string(models.LifecycleStageDevelopment),
		PerformedBy:      "system",
		PerformedAt:      now,
	}); err != nil {
		return nil, err
	}

	return cv, nil
}

func (s *CatalogVersionService) GetCatalogVersion(ctx context.Context, id string) (*models.CatalogVersion, error) {
	cv, err := s.cvRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !s.isStageAllowed(cv.LifecycleStage) {
		return nil, domainerrors.NewForbidden("catalog version stage not available in this cluster role")
	}
	return cv, nil
}

func (s *CatalogVersionService) ListCatalogVersions(ctx context.Context, params models.ListParams) ([]*models.CatalogVersion, int, error) {
	items, total, err := s.cvRepo.List(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	if len(s.allowedStages) == 0 {
		return items, total, nil
	}
	var filtered []*models.CatalogVersion
	for _, cv := range items {
		if s.isStageAllowed(cv.LifecycleStage) {
			filtered = append(filtered, cv)
		}
	}
	return filtered, len(filtered), nil
}

func (s *CatalogVersionService) isStageAllowed(stage models.LifecycleStage) bool {
	if len(s.allowedStages) == 0 {
		return true
	}
	for _, allowed := range s.allowedStages {
		if allowed == string(stage) {
			return true
		}
	}
	return false
}

// Promote advances the lifecycle stage forward.
func (s *CatalogVersionService) Promote(ctx context.Context, id string, role Role, performedBy string) (*PromoteResult, error) {
	cv, err := s.cvRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	var targetStage models.LifecycleStage
	switch cv.LifecycleStage {
	case models.LifecycleStageDevelopment:
		targetStage = models.LifecycleStageTesting
		if role == RoleRO {
			return nil, domainerrors.NewForbidden("RO users cannot promote catalog versions")
		}
	case models.LifecycleStageTesting:
		targetStage = models.LifecycleStageProduction
		if role == RoleRO || role == RoleRW {
			return nil, domainerrors.NewForbidden("only Admin and above can promote to production")
		}
	case models.LifecycleStageProduction:
		return nil, domainerrors.NewValidation("cannot promote beyond production")
	default:
		return nil, domainerrors.NewValidation("invalid lifecycle stage: " + string(cv.LifecycleStage))
	}

	if err := s.cvRepo.UpdateLifecycle(ctx, id, targetStage); err != nil {
		return nil, err
	}

	now := time.Now()
	if err := s.ltRepo.Create(ctx, &models.LifecycleTransition{
		ID:               uuid.Must(uuid.NewV7()).String(),
		CatalogVersionID: id,
		FromStage:        string(cv.LifecycleStage),
		ToStage:          string(targetStage),
		PerformedBy:      performedBy,
		PerformedAt:      now,
	}); err != nil {
		return nil, err
	}

	// Create/update CatalogVersion CR in K8s
	if s.crManager != nil {
		k8sName := SanitizeK8sName(cv.VersionLabel)
		if k8sName == "" {
			return nil, domainerrors.NewValidation("version label must contain at least one alphanumeric character")
		}
		entityTypeNames, err := s.getEntityTypeNamesForCV(ctx, id)
		if err != nil {
			return nil, err
		}
		if err := s.crManager.CreateOrUpdate(ctx, CatalogVersionCRSpec{
			Name:           k8sName,
			Namespace:      s.namespace,
			VersionLabel:   cv.VersionLabel,
			Description:    "",
			LifecycleStage: string(targetStage),
			EntityTypes:    entityTypeNames,
			SourceDBID:     cv.ID,
			PromotedBy:     performedBy,
			PromotedAt:     now.Format(time.RFC3339),
		}); err != nil {
			return nil, err
		}
	}

	// Collect catalog warnings (draft/invalid catalogs pinned to this CV)
	result := &PromoteResult{}
	if s.catalogRepo != nil {
		catalogs, err := s.catalogRepo.ListByCatalogVersionID(ctx, id)
		if err == nil {
			for _, cat := range catalogs {
				if cat.ValidationStatus != models.ValidationStatusValid {
					result.Warnings = append(result.Warnings, CatalogWarning{
						CatalogName:      cat.Name,
						ValidationStatus: string(cat.ValidationStatus),
					})
				}
			}
		}
		// Warnings are best-effort — don't fail promotion if catalog lookup fails
	}

	return result, nil
}

// DeleteCatalogVersion deletes a catalog version.
// Admin can delete non-production versions. SuperAdmin can delete any version.
func (s *CatalogVersionService) DeleteCatalogVersion(ctx context.Context, id string, role Role) error {
	cv, err := s.cvRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if cv.LifecycleStage == models.LifecycleStageProduction {
		if role != RoleSuperAdmin {
			return domainerrors.NewForbidden("only Super Admin can delete production catalog versions")
		}
	} else {
		if role != RoleAdmin && role != RoleSuperAdmin {
			return domainerrors.NewForbidden("only Admin and above can delete catalog versions")
		}
	}

	// Block deletion if any catalogs are pinned to this CV
	if s.catalogRepo != nil {
		catalogs, err := s.catalogRepo.ListByCatalogVersionID(ctx, id)
		if err != nil {
			return err
		}
		if len(catalogs) > 0 {
			names := make([]string, len(catalogs))
			for i, c := range catalogs {
				names[i] = c.Name
			}
			return domainerrors.NewConflict("CatalogVersion",
				fmt.Sprintf("cannot delete: %d catalog(s) are pinned to this version (%s)", len(catalogs), fmt.Sprintf("%v", names)))
		}
	}

	if err := s.cvRepo.Delete(ctx, id); err != nil {
		return err
	}

	// Delete CatalogVersion CR from K8s
	if s.crManager != nil {
		return s.crManager.Delete(ctx, SanitizeK8sName(cv.VersionLabel), s.namespace)
	}
	return nil
}

// Demote moves the lifecycle stage backward.
func (s *CatalogVersionService) Demote(ctx context.Context, id string, role Role, performedBy string, targetStage models.LifecycleStage) error {
	cv, err := s.cvRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	switch cv.LifecycleStage {
	case models.LifecycleStageDevelopment:
		return domainerrors.NewValidation("cannot demote from development")
	case models.LifecycleStageTesting:
		if targetStage != models.LifecycleStageDevelopment {
			return domainerrors.NewValidation("testing can only be demoted to development")
		}
		// RW and above can demote test→dev
		if role == RoleRO {
			return domainerrors.NewForbidden("RO users cannot demote catalog versions")
		}
	case models.LifecycleStageProduction:
		if targetStage != models.LifecycleStageTesting && targetStage != models.LifecycleStageDevelopment {
			return domainerrors.NewValidation("production can only be demoted to testing or development")
		}
		// Only Super Admin can demote from production
		if role != RoleSuperAdmin {
			return domainerrors.NewForbidden("only Super Admin can demote from production")
		}
	default:
		return domainerrors.NewValidation("invalid lifecycle stage")
	}

	if err := s.cvRepo.UpdateLifecycle(ctx, id, targetStage); err != nil {
		return err
	}

	now := time.Now()
	if err := s.ltRepo.Create(ctx, &models.LifecycleTransition{
		ID:               uuid.Must(uuid.NewV7()).String(),
		CatalogVersionID: id,
		FromStage:        string(cv.LifecycleStage),
		ToStage:          string(targetStage),
		PerformedBy:      performedBy,
		PerformedAt:      now,
	}); err != nil {
		return err
	}

	// Manage CatalogVersion CR in K8s
	if s.crManager != nil {
		k8sName := SanitizeK8sName(cv.VersionLabel)
		if k8sName == "" {
			return domainerrors.NewValidation("version label must contain at least one alphanumeric character")
		}
		if targetStage == models.LifecycleStageDevelopment {
			// Development versions don't have CRs
			return s.crManager.Delete(ctx, k8sName, s.namespace)
		}
		// Update CR for testing stage
		entityTypeNames, err := s.getEntityTypeNamesForCV(ctx, id)
		if err != nil {
			return err
		}
		return s.crManager.CreateOrUpdate(ctx, CatalogVersionCRSpec{
			Name:           k8sName,
			Namespace:      s.namespace,
			VersionLabel:   cv.VersionLabel,
			LifecycleStage: string(targetStage),
			EntityTypes:    entityTypeNames,
			SourceDBID:     cv.ID,
			PromotedBy:     performedBy,
			PromotedAt:     now.Format(time.RFC3339),
		})
	}

	return nil
}

// ResolvedPin represents a catalog version pin with resolved entity type information.
type ResolvedPin struct {
	PinID               string
	EntityTypeName      string
	EntityTypeID        string
	EntityTypeVersionID string
	Version             int
	Description         string
}

// ListPins returns resolved pins for a catalog version.
func (s *CatalogVersionService) ListPins(ctx context.Context, cvID string) ([]ResolvedPin, error) {
	if _, err := s.cvRepo.GetByID(ctx, cvID); err != nil {
		return nil, err
	}

	pins, err := s.pinRepo.ListByCatalogVersion(ctx, cvID)
	if err != nil {
		return nil, err
	}

	resolved := make([]ResolvedPin, 0, len(pins))
	for _, pin := range pins {
		etv, err := s.etvRepo.GetByID(ctx, pin.EntityTypeVersionID)
		if err != nil {
			return nil, err
		}
		et, err := s.etRepo.GetByID(ctx, etv.EntityTypeID)
		if err != nil {
			return nil, err
		}
		resolved = append(resolved, ResolvedPin{
			PinID:               pin.ID,
			EntityTypeName:      et.Name,
			EntityTypeID:        et.ID,
			EntityTypeVersionID: pin.EntityTypeVersionID,
			Version:             etv.Version,
			Description:         etv.Description,
		})
	}
	return resolved, nil
}

// ListTransitions returns lifecycle transition history for a catalog version.
func (s *CatalogVersionService) ListTransitions(ctx context.Context, cvID string) ([]*models.LifecycleTransition, error) {
	if _, err := s.cvRepo.GetByID(ctx, cvID); err != nil {
		return nil, err
	}
	return s.ltRepo.ListByCatalogVersion(ctx, cvID)
}

// UpdateCatalogVersion updates the version label and/or description of a catalog version.
func (s *CatalogVersionService) UpdateCatalogVersion(ctx context.Context, id string, versionLabel, description *string, role Role) (*models.CatalogVersion, error) {
	cv, err := s.cvRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := checkCVEditAllowed(cv, role, "metadata editing"); err != nil {
		return nil, err
	}

	changed := false

	if versionLabel != nil && *versionLabel != cv.VersionLabel {
		// Check uniqueness — propagate non-NotFound DB errors
		existing, err := s.cvRepo.GetByLabel(ctx, *versionLabel)
		if err == nil && existing.ID != cv.ID {
			return nil, domainerrors.NewConflict("CatalogVersion", "version label already exists: "+*versionLabel)
		} else if err != nil && !domainerrors.IsNotFound(err) {
			return nil, err
		}
		cv.VersionLabel = *versionLabel
		changed = true
	}

	if description != nil && *description != cv.Description {
		cv.Description = *description
		changed = true
	}

	if !changed {
		return cv, nil
	}

	cv.UpdatedAt = time.Now()
	if err := s.cvRepo.Update(ctx, cv); err != nil {
		return nil, err
	}

	return cv, nil
}

// checkCVEditAllowed verifies that the CV stage permits the given operation for the given role.
// development: RW+ allowed, testing: SuperAdmin only, production: blocked entirely.
func checkCVEditAllowed(cv *models.CatalogVersion, role Role, operation string) error {
	switch cv.LifecycleStage {
	case models.LifecycleStageProduction:
		return domainerrors.NewValidation(operation + " is not allowed on production catalog versions")
	case models.LifecycleStageTesting:
		if role != RoleSuperAdmin {
			return domainerrors.NewValidation("only SuperAdmin can perform " + operation + " on testing catalog versions")
		}
	}
	return nil
}

// AddPin adds a new entity type version pin to a catalog version.

func (s *CatalogVersionService) AddPin(ctx context.Context, cvID, entityTypeVersionID string, role Role) (*models.CatalogVersionPin, error) {
	// Verify CV exists and check stage permissions
	cv, err := s.cvRepo.GetByID(ctx, cvID)
	if err != nil {
		return nil, err
	}
	if err := checkCVEditAllowed(cv, role, "pin editing"); err != nil {
		return nil, err
	}

	// Verify ETV exists and get its entity type ID
	newETV, err := s.etvRepo.GetByID(ctx, entityTypeVersionID)
	if err != nil {
		return nil, err
	}

	// Check for duplicate pin — reject if ANY version of the same entity type is already pinned
	existingPins, err := s.pinRepo.ListByCatalogVersion(ctx, cvID)
	if err != nil {
		return nil, err
	}
	if len(existingPins) > 0 {
		// Batch-fetch all existing pin ETVs in one query to avoid N+1
		etvIDs := make([]string, len(existingPins))
		for i, pin := range existingPins {
			etvIDs[i] = pin.EntityTypeVersionID
		}
		existingETVs, err := s.etvRepo.GetByIDs(ctx, etvIDs)
		if err != nil {
			return nil, err
		}
		if len(existingETVs) < len(etvIDs) {
			return nil, domainerrors.NewValidation("data integrity error: orphaned pin references a deleted entity type version")
		}
		for _, etv := range existingETVs {
			if etv.EntityTypeID == newETV.EntityTypeID {
				return nil, domainerrors.NewConflict("CatalogVersionPin", "entity type already pinned; use UpdatePin to change version")
			}
		}
	}

	pin := &models.CatalogVersionPin{
		ID:                  uuid.Must(uuid.NewV7()).String(),
		CatalogVersionID:    cvID,
		EntityTypeVersionID: entityTypeVersionID,
	}
	if err := s.pinRepo.Create(ctx, pin); err != nil {
		return nil, err
	}

	if err := s.resetDependentCatalogs(ctx, cvID); err != nil {
		return nil, err
	}

	return pin, nil
}

// UpdatePin changes the pinned entity type version on an existing pin.
// The new ETV must belong to the same entity type as the current pin.
// If migration repos are configured and the version actually changes,
// IAVs are remapped from old to new attribute IDs. In dry-run mode,
// the migration report is returned without applying changes.
func (s *CatalogVersionService) UpdatePin(ctx context.Context, cvID, pinID, newEntityTypeVersionID string, role Role, dryRun bool) (*models.UpdatePinResult, error) {
	cv, err := s.cvRepo.GetByID(ctx, cvID)
	if err != nil {
		return nil, err
	}
	if err := checkCVEditAllowed(cv, role, "pin editing"); err != nil {
		return nil, err
	}

	pin, err := s.pinRepo.GetByID(ctx, pinID)
	if err != nil {
		return nil, err
	}
	if pin.CatalogVersionID != cvID {
		return nil, domainerrors.NewNotFound("CatalogVersionPin", pinID)
	}

	currentETV, err := s.etvRepo.GetByID(ctx, pin.EntityTypeVersionID)
	if err != nil {
		return nil, err
	}

	newETV, err := s.etvRepo.GetByID(ctx, newEntityTypeVersionID)
	if err != nil {
		return nil, err
	}
	if newETV.EntityTypeID != currentETV.EntityTypeID {
		return nil, domainerrors.NewValidation("entity type mismatch: new version must belong to the same entity type")
	}

	oldETVID := pin.EntityTypeVersionID
	var migration *models.MigrationReport

	if oldETVID != newEntityTypeVersionID && s.attrRepo != nil {
		migration, err = s.buildMigrationReport(ctx, oldETVID, newEntityTypeVersionID, cvID, currentETV.EntityTypeID)
		if err != nil {
			return nil, err
		}
	}

	if !dryRun {
		doMutations := func(txCtx context.Context) error {
			if migration != nil && len(migration.InstanceIDs) > 0 && len(migration.IDMapping) > 0 {
				if _, err := s.iavRepo.RemapAttributeIDs(txCtx, migration.InstanceIDs, migration.IDMapping); err != nil {
					return err
				}
			}
			pin.EntityTypeVersionID = newEntityTypeVersionID
			if err := s.pinRepo.Update(txCtx, pin); err != nil {
				return err
			}
			return s.resetDependentCatalogs(txCtx, cvID)
		}

		if s.txManager != nil {
			if err := s.txManager.RunInTransaction(ctx, doMutations); err != nil {
				return nil, err
			}
		} else {
			if err := doMutations(ctx); err != nil {
				return nil, err
			}
		}
	}

	return &models.UpdatePinResult{Pin: pin, Migration: migration}, nil
}

func (s *CatalogVersionService) buildMigrationReport(ctx context.Context, oldETVID, newETVID, cvID, entityTypeID string) (*models.MigrationReport, error) {
	oldAttrs, err := s.attrRepo.ListByVersion(ctx, oldETVID)
	if err != nil {
		return nil, err
	}
	newAttrs, err := s.attrRepo.ListByVersion(ctx, newETVID)
	if err != nil {
		return nil, err
	}

	mapping, mappings, warnings := buildAttributeMapping(oldAttrs, newAttrs)

	instanceIDs, breakdown, err := s.collectAffectedInstances(ctx, cvID, entityTypeID)
	if err != nil {
		return nil, err
	}

	affectedCount := len(instanceIDs)
	for i := range warnings {
		warnings[i].AffectedInstances = affectedCount
	}

	return &models.MigrationReport{
		AffectedCatalogs:  len(breakdown),
		AffectedInstances: affectedCount,
		CatalogBreakdown:  breakdown,
		AttributeMappings: mappings,
		Warnings:          warnings,
		IDMapping:         mapping,
		InstanceIDs:       instanceIDs,
	}, nil
}

func buildAttributeMapping(oldAttrs, newAttrs []*models.Attribute) (map[string]string, []models.AttributeMapping, []models.MigrationWarning) {
	newByName := make(map[string]*models.Attribute, len(newAttrs))
	for _, a := range newAttrs {
		newByName[a.Name] = a
	}

	oldByOrdinal := make(map[int]*models.Attribute, len(oldAttrs))
	for _, a := range oldAttrs {
		oldByOrdinal[a.Ordinal] = a
	}

	mapping := make(map[string]string)
	var mappings []models.AttributeMapping
	var warnings []models.MigrationWarning
	matchedOld := make(map[string]bool)

	// Match by name
	for _, oldAttr := range oldAttrs {
		if newAttr, ok := newByName[oldAttr.Name]; ok {
			mapping[oldAttr.ID] = newAttr.ID
			mappings = append(mappings, models.AttributeMapping{
				OldName: oldAttr.Name, NewName: newAttr.Name, Action: "remap",
			})
			matchedOld[oldAttr.ID] = true

			if oldAttr.TypeDefinitionVersionID != newAttr.TypeDefinitionVersionID {
				warnings = append(warnings, models.MigrationWarning{
					Type: "type_changed", Attribute: oldAttr.Name,
					OldType: oldAttr.TypeDefinitionVersionID, NewType: newAttr.TypeDefinitionVersionID,
				})
			}
		} else {
			mappings = append(mappings, models.AttributeMapping{OldName: oldAttr.Name, Action: "orphaned"})
			warnings = append(warnings, models.MigrationWarning{Type: "deleted_attribute", Attribute: oldAttr.Name})
		}
	}

	// Detect renames (same ordinal, different name, unmatched)
	for _, newAttr := range newAttrs {
		if oldAttr, ok := findOldByName(oldAttrs, newAttr.Name); ok && matchedOld[oldAttr.ID] {
			continue
		}
		if oldAttr, ok := oldByOrdinal[newAttr.Ordinal]; ok && !matchedOld[oldAttr.ID] {
			mapping[oldAttr.ID] = newAttr.ID
			matchedOld[oldAttr.ID] = true
			warnings = append(warnings, models.MigrationWarning{
				Type: "renamed", Attribute: newAttr.Name, OldType: oldAttr.Name, NewType: newAttr.Name,
			})
			if oldAttr.TypeDefinitionVersionID != newAttr.TypeDefinitionVersionID {
				warnings = append(warnings, models.MigrationWarning{
					Type: "type_changed", Attribute: newAttr.Name,
					OldType: oldAttr.TypeDefinitionVersionID, NewType: newAttr.TypeDefinitionVersionID,
				})
			}
			for i, m := range mappings {
				if m.OldName == oldAttr.Name && m.Action == "orphaned" {
					mappings[i] = models.AttributeMapping{OldName: oldAttr.Name, NewName: newAttr.Name, Action: "remap"}
					break
				}
			}
			filtered := warnings[:0]
			for _, w := range warnings {
				if !(w.Type == "deleted_attribute" && w.Attribute == oldAttr.Name) {
					filtered = append(filtered, w)
				}
			}
			warnings = filtered
		}
	}

	// Detect new attributes (not matched by name or rename)
	renamedNames := make(map[string]bool)
	for _, w := range warnings {
		if w.Type == "renamed" {
			renamedNames[w.NewType] = true
		}
	}
	for _, newAttr := range newAttrs {
		if oldAttr, ok := findOldByName(oldAttrs, newAttr.Name); ok && matchedOld[oldAttr.ID] {
			continue
		}
		if renamedNames[newAttr.Name] {
			continue
		}
		mappings = append(mappings, models.AttributeMapping{NewName: newAttr.Name, Action: "added"})
		if newAttr.Required {
			warnings = append(warnings, models.MigrationWarning{Type: "new_required", Attribute: newAttr.Name})
		}
	}

	return mapping, mappings, warnings
}

const migrationLimit = 10000

func (s *CatalogVersionService) collectAffectedInstances(ctx context.Context, cvID, entityTypeID string) ([]string, []models.CatalogImpact, error) {
	catalogs, err := s.catalogRepo.ListByCatalogVersionID(ctx, cvID)
	if err != nil {
		return nil, nil, err
	}

	var instanceIDs []string
	var breakdown []models.CatalogImpact
	for _, cat := range catalogs {
		instances, total, err := s.instRepo.List(ctx, entityTypeID, cat.ID, models.ListParams{Limit: migrationLimit})
		if err != nil {
			return nil, nil, err
		}
		if total > migrationLimit {
			return nil, nil, domainerrors.NewValidation(
				fmt.Sprintf("catalog %s has %d instances of this entity type, exceeds migration limit of %d",
					cat.Name, total, migrationLimit))
		}
		if total > 0 {
			breakdown = append(breakdown, models.CatalogImpact{CatalogName: cat.Name, InstanceCount: total})
		}
		for _, inst := range instances {
			instanceIDs = append(instanceIDs, inst.ID)
		}
	}

	return instanceIDs, breakdown, nil
}

func findOldByName(attrs []*models.Attribute, name string) (*models.Attribute, bool) {
	for _, a := range attrs {
		if a.Name == name {
			return a, true
		}
	}
	return nil, false
}

// RemovePin removes a pin from a catalog version.
func (s *CatalogVersionService) RemovePin(ctx context.Context, cvID, pinID string, role Role) error {
	// Verify CV exists and check stage permissions
	cv, err := s.cvRepo.GetByID(ctx, cvID)
	if err != nil {
		return err
	}
	if err := checkCVEditAllowed(cv, role, "pin editing"); err != nil {
		return err
	}

	// Get pin and verify it belongs to this CV
	pin, err := s.pinRepo.GetByID(ctx, pinID)
	if err != nil {
		return err
	}
	if pin.CatalogVersionID != cvID {
		return domainerrors.NewNotFound("CatalogVersionPin", pinID)
	}

	if err := s.pinRepo.Delete(ctx, pinID); err != nil {
		return err
	}

	return s.resetDependentCatalogs(ctx, cvID)
}

// resetDependentCatalogs resets validation status to draft for all catalogs pinned to a CV.
func (s *CatalogVersionService) resetDependentCatalogs(ctx context.Context, cvID string) error {
	if s.catalogRepo == nil {
		return nil
	}
	catalogs, err := s.catalogRepo.ListByCatalogVersionID(ctx, cvID)
	if err != nil {
		return err
	}
	for _, cat := range catalogs {
		if cat.ValidationStatus == models.ValidationStatusDraft {
			continue // already draft, skip unnecessary write
		}
		if err := s.catalogRepo.UpdateValidationStatus(ctx, cat.ID, models.ValidationStatusDraft); err != nil {
			return err
		}
	}
	return nil
}

// getEntityTypeNamesForCV resolves entity type names from catalog version pins.
func (s *CatalogVersionService) getEntityTypeNamesForCV(ctx context.Context, cvID string) ([]string, error) {
	pins, err := s.pinRepo.ListByCatalogVersion(ctx, cvID)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, pin := range pins {
		etv, err := s.etvRepo.GetByID(ctx, pin.EntityTypeVersionID)
		if err != nil {
			return nil, err
		}
		et, err := s.etRepo.GetByID(ctx, etv.EntityTypeID)
		if err != nil {
			return nil, err
		}
		names = append(names, et.Name)
	}
	return names, nil
}
