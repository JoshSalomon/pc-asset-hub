package operational

import (
	"context"
	"log"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository"
)

var dnsLabelRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

type CatalogService struct {
	catalogRepo repository.CatalogRepository
	cvRepo      repository.CatalogVersionRepository
	instRepo    repository.EntityInstanceRepository
	crManager   CatalogCRManager
	namespace   string
}

func NewCatalogService(
	catalogRepo repository.CatalogRepository,
	cvRepo repository.CatalogVersionRepository,
	instRepo repository.EntityInstanceRepository,
	crManager CatalogCRManager,
	namespace string,
) *CatalogService {
	return &CatalogService{
		catalogRepo: catalogRepo,
		cvRepo:      cvRepo,
		instRepo:    instRepo,
		crManager:   crManager,
		namespace:   namespace,
	}
}

func ValidateCatalogName(name string) error {
	if name == "" {
		return domainerrors.NewValidation("catalog name is required")
	}
	if len(name) > 63 {
		return domainerrors.NewValidation("catalog name must be at most 63 characters")
	}
	if !dnsLabelRegex.MatchString(name) {
		return domainerrors.NewValidation("catalog name must be a valid DNS label: lowercase alphanumeric and hyphens, must start and end with alphanumeric")
	}
	return nil
}

func (s *CatalogService) CreateCatalog(ctx context.Context, name, description, catalogVersionID string) (*models.Catalog, error) {
	if err := ValidateCatalogName(name); err != nil {
		return nil, err
	}

	if catalogVersionID == "" {
		return nil, domainerrors.NewValidation("catalog_version_id is required")
	}

	// Verify CV exists
	if _, err := s.cvRepo.GetByID(ctx, catalogVersionID); err != nil {
		return nil, err
	}

	now := time.Now()
	catalog := &models.Catalog{
		ID:               uuid.Must(uuid.NewV7()).String(),
		Name:             name,
		Description:      description,
		CatalogVersionID: catalogVersionID,
		ValidationStatus: models.ValidationStatusDraft,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.catalogRepo.Create(ctx, catalog); err != nil {
		return nil, err
	}

	return catalog, nil
}

// CatalogDetail includes the resolved CV label.
type CatalogDetail struct {
	*models.Catalog
	CatalogVersionLabel string
}

func (s *CatalogService) GetByName(ctx context.Context, name string) (*CatalogDetail, error) {
	catalog, err := s.catalogRepo.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}

	cv, err := s.cvRepo.GetByID(ctx, catalog.CatalogVersionID)
	if err != nil {
		return nil, err
	}

	return &CatalogDetail{
		Catalog:             catalog,
		CatalogVersionLabel: cv.VersionLabel,
	}, nil
}

func (s *CatalogService) List(ctx context.Context, params models.ListParams) ([]*CatalogDetail, int, error) {
	catalogs, total, err := s.catalogRepo.List(ctx, params)
	if err != nil {
		return nil, 0, err
	}

	// Batch-resolve CV labels
	cvCache := make(map[string]string)
	details := make([]*CatalogDetail, len(catalogs))
	for i, cat := range catalogs {
		label, ok := cvCache[cat.CatalogVersionID]
		if !ok {
			cv, err := s.cvRepo.GetByID(ctx, cat.CatalogVersionID)
			if err != nil {
				label = ""
			} else {
				label = cv.VersionLabel
			}
			cvCache[cat.CatalogVersionID] = label
		}
		details[i] = &CatalogDetail{
			Catalog:             cat,
			CatalogVersionLabel: label,
		}
	}
	return details, total, nil
}

func (s *CatalogService) Delete(ctx context.Context, name string) error {
	catalog, err := s.catalogRepo.GetByName(ctx, name)
	if err != nil {
		return err
	}

	// Clean up Catalog CR if published
	if catalog.Published && s.crManager != nil {
		if err := s.crManager.Delete(ctx, catalog.Name, s.namespace); err != nil {
			return err
		}
	}

	// Cascade delete all instances in this catalog
	if err := s.instRepo.DeleteByCatalogID(ctx, catalog.ID); err != nil {
		return err
	}

	return s.catalogRepo.Delete(ctx, catalog.ID)
}

// IsPublished checks if a catalog is published (implements CatalogPublishChecker for middleware).
func (s *CatalogService) IsPublished(c echo.Context, catalogName string) (bool, error) {
	catalog, err := s.catalogRepo.GetByName(c.Request().Context(), catalogName)
	if err != nil {
		return false, err
	}
	return catalog.Published, nil
}

// SyncCR updates the Catalog CR if the catalog is published.
// Called after data mutations to keep the CR spec in sync with DB state.
func (s *CatalogService) SyncCR(ctx context.Context, catalogName string) {
	if s.crManager == nil {
		return
	}
	catalog, err := s.catalogRepo.GetByName(ctx, catalogName)
	if err != nil || !catalog.Published {
		return
	}
	cvLabel := ""
	cv, err := s.cvRepo.GetByID(ctx, catalog.CatalogVersionID)
	if err == nil {
		cvLabel = cv.VersionLabel
	}
	if err := s.crManager.CreateOrUpdate(ctx, CatalogCRSpec{
		Name:                catalog.Name,
		Namespace:           s.namespace,
		CatalogVersionLabel: cvLabel,
		ValidationStatus:    string(catalog.ValidationStatus),
		APIEndpoint:         "/api/data/v1/catalogs/" + catalog.Name,
		SourceDBID:          catalog.ID,
	}); err != nil {
		log.Printf("warning: failed to sync Catalog CR %s: %v", catalog.Name, err)
	}
}

func (s *CatalogService) Publish(ctx context.Context, name string) error {
	catalog, err := s.catalogRepo.GetByName(ctx, name)
	if err != nil {
		return err
	}

	if catalog.ValidationStatus != models.ValidationStatusValid {
		return domainerrors.NewValidation("catalog must be valid to publish (current status: " + string(catalog.ValidationStatus) + ")")
	}

	now := time.Now()
	if err := s.catalogRepo.UpdatePublished(ctx, catalog.ID, true, &now); err != nil {
		return err
	}

	// Resolve CV label for CR spec
	cvLabel := ""
	cv, err := s.cvRepo.GetByID(ctx, catalog.CatalogVersionID)
	if err == nil {
		cvLabel = cv.VersionLabel
	}

	if s.crManager != nil {
		if err := s.crManager.CreateOrUpdate(ctx, CatalogCRSpec{
			Name:                catalog.Name,
			Namespace:           s.namespace,
			CatalogVersionLabel: cvLabel,
			ValidationStatus:    string(catalog.ValidationStatus),
			APIEndpoint:         "/api/data/v1/catalogs/" + catalog.Name,
			SourceDBID:          catalog.ID,
			PublishedAt:         now.Format(time.RFC3339),
		}); err != nil {
			// Rollback DB state on CR creation failure
			_ = s.catalogRepo.UpdatePublished(ctx, catalog.ID, false, nil)
			return err
		}
	}

	return nil
}

func (s *CatalogService) Unpublish(ctx context.Context, name string) error {
	catalog, err := s.catalogRepo.GetByName(ctx, name)
	if err != nil {
		return err
	}

	if !catalog.Published {
		return nil // already unpublished — idempotent
	}

	if err := s.catalogRepo.UpdatePublished(ctx, catalog.ID, false, nil); err != nil {
		return err
	}

	if s.crManager != nil {
		if err := s.crManager.Delete(ctx, catalog.Name, s.namespace); err != nil {
			return err
		}
	}

	return nil
}
