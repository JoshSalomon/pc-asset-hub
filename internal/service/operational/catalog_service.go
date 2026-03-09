package operational

import (
	"context"
	"regexp"
	"time"

	"github.com/google/uuid"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository"
)

var dnsLabelRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

type CatalogService struct {
	catalogRepo repository.CatalogRepository
	cvRepo      repository.CatalogVersionRepository
	instRepo    repository.EntityInstanceRepository
}

func NewCatalogService(
	catalogRepo repository.CatalogRepository,
	cvRepo repository.CatalogVersionRepository,
	instRepo repository.EntityInstanceRepository,
) *CatalogService {
	return &CatalogService{
		catalogRepo: catalogRepo,
		cvRepo:      cvRepo,
		instRepo:    instRepo,
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

	// Cascade delete all instances in this catalog
	if err := s.instRepo.DeleteByCatalogID(ctx, catalog.ID); err != nil {
		return err
	}

	return s.catalogRepo.Delete(ctx, catalog.ID)
}
