package meta

import (
	"context"
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
) *CatalogVersionService {
	return &CatalogVersionService{
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
}

func (s *CatalogVersionService) CreateCatalogVersion(ctx context.Context, label string, pins []models.CatalogVersionPin) (*models.CatalogVersion, error) {
	now := time.Now()
	cv := &models.CatalogVersion{
		ID:             uuid.Must(uuid.NewV7()).String(),
		VersionLabel:   label,
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
	EntityTypeName      string
	EntityTypeID        string
	EntityTypeVersionID string
	Version             int
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
			EntityTypeName:      et.Name,
			EntityTypeID:        et.ID,
			EntityTypeVersionID: pin.EntityTypeVersionID,
			Version:             etv.Version,
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
