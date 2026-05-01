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
	iavRepo     repository.InstanceAttributeValueRepository
	linkRepo    repository.AssociationLinkRepository
	txManager   repository.TransactionManager
	crManager   CatalogCRManager
	namespace   string
}

func NewCatalogService(
	catalogRepo repository.CatalogRepository,
	cvRepo repository.CatalogVersionRepository,
	instRepo repository.EntityInstanceRepository,
	crManager CatalogCRManager,
	namespace string,
	opts ...CatalogServiceOption,
) *CatalogService {
	s := &CatalogService{
		catalogRepo: catalogRepo,
		cvRepo:      cvRepo,
		instRepo:    instRepo,
		crManager:   crManager,
		namespace:   namespace,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// CatalogServiceOption configures optional dependencies for CatalogService.
type CatalogServiceOption func(*CatalogService)

// WithCopyDeps adds the repositories needed for Copy & Replace operations.
func WithCopyDeps(iavRepo repository.InstanceAttributeValueRepository, linkRepo repository.AssociationLinkRepository) CatalogServiceOption {
	return func(s *CatalogService) {
		s.iavRepo = iavRepo
		s.linkRepo = linkRepo
	}
}

// WithTransactionManager adds transaction support for atomic operations.
func WithTransactionManager(txm repository.TransactionManager) CatalogServiceOption {
	return func(s *CatalogService) {
		s.txManager = txm
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

	// Clean up Catalog CR if published (K8s operation, outside transaction)
	if catalog.Published && s.crManager != nil {
		if err := s.crManager.Delete(ctx, catalog.Name, s.namespace); err != nil {
			return err
		}
	}

	// Cascade delete: IAVs + links per instance, then instances, then catalog
	doDelete := func(txCtx context.Context) error {
		if s.iavRepo != nil || s.linkRepo != nil {
			instances, err := s.instRepo.ListByCatalog(txCtx, catalog.ID)
			if err != nil {
				return err
			}
			for _, inst := range instances {
				if s.linkRepo != nil {
					if err := s.linkRepo.DeleteByInstance(txCtx, inst.ID); err != nil {
						return err
					}
				}
				if s.iavRepo != nil {
					if err := s.iavRepo.DeleteByInstanceID(txCtx, inst.ID); err != nil {
						return err
					}
				}
			}
		}
		if err := s.instRepo.DeleteByCatalogID(txCtx, catalog.ID); err != nil {
			return err
		}
		return s.catalogRepo.Delete(txCtx, catalog.ID)
	}

	if s.txManager != nil {
		return s.txManager.RunInTransaction(ctx, doDelete)
	}
	return doDelete(ctx)
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

	if catalog.Published {
		return nil
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

// UpdateMetadata updates a catalog's name, description, and/or catalog version.
func (s *CatalogService) UpdateMetadata(ctx context.Context, name string, newName, description, catalogVersionID *string, role string) (*CatalogDetail, error) {
	catalog, err := s.catalogRepo.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}

	// Published catalog restrictions
	if catalog.Published {
		if role != "SuperAdmin" {
			return nil, domainerrors.NewForbidden("only SuperAdmin can edit published catalogs")
		}
		if newName != nil {
			return nil, domainerrors.NewValidation("cannot rename published catalog")
		}
		if catalogVersionID != nil {
			return nil, domainerrors.NewValidation("cannot change catalog version of published catalog; unpublish first")
		}
	}

	changed := false
	needsUpdate := false

	// Handle rename (uses dedicated UpdateName — separate from general Update)
	if newName != nil && *newName != catalog.Name {
		if err := ValidateCatalogName(*newName); err != nil {
			return nil, err
		}
		// Check uniqueness — propagate non-NotFound DB errors
		existing, err := s.catalogRepo.GetByName(ctx, *newName)
		if err == nil && existing.ID != catalog.ID {
			return nil, domainerrors.NewConflict("Catalog", "name already exists: "+*newName)
		} else if err != nil && !domainerrors.IsNotFound(err) {
			return nil, err
		}
		if err := s.catalogRepo.UpdateName(ctx, catalog.ID, *newName); err != nil {
			return nil, err
		}
		catalog.Name = *newName
		catalog.UpdatedAt = time.Now()
		changed = true
	}

	// Handle description update — cosmetic change, does not reset validation
	if description != nil && *description != catalog.Description {
		catalog.Description = *description
		needsUpdate = true
	}

	// Handle catalog version change
	if catalogVersionID != nil && *catalogVersionID != catalog.CatalogVersionID {
		// Verify new CV exists
		if _, err := s.cvRepo.GetByID(ctx, *catalogVersionID); err != nil {
			return nil, err
		}
		catalog.CatalogVersionID = *catalogVersionID
		needsUpdate = true
		changed = true
	}

	// Single Update call for description and/or CV change
	if needsUpdate {
		catalog.UpdatedAt = time.Now()
		if err := s.catalogRepo.Update(ctx, catalog); err != nil {
			return nil, err
		}
	}

	// Reset validation status only for structural changes (name, CV re-pin).
	// Description-only edits are cosmetic and should not invalidate a published catalog.
	if changed {
		if err := s.catalogRepo.UpdateValidationStatus(ctx, catalog.ID, models.ValidationStatusDraft); err != nil {
			return nil, err
		}
		catalog.ValidationStatus = models.ValidationStatusDraft
	}

	// Sync CR if published — covers both structural changes and description edits.
	// For description-only edits on published catalogs, this refreshes the CR
	// without resetting validation status (the catalog stays valid+published).
	if catalog.Published && (changed || needsUpdate) {
		s.SyncCR(ctx, catalog.Name)
	}

	// Resolve CV label for response
	cvLabel := ""
	cv, err := s.cvRepo.GetByID(ctx, catalog.CatalogVersionID)
	if err == nil {
		cvLabel = cv.VersionLabel
	}

	return &CatalogDetail{
		Catalog:             catalog,
		CatalogVersionLabel: cvLabel,
	}, nil
}

// CopyCatalog deep-clones all data from a source catalog into a new catalog.
func (s *CatalogService) CopyCatalog(ctx context.Context, sourceName, targetName, description string) (*models.Catalog, error) {
	// Validate target name
	if err := ValidateCatalogName(targetName); err != nil {
		return nil, err
	}

	// Get source catalog
	source, err := s.catalogRepo.GetByName(ctx, sourceName)
	if err != nil {
		return nil, err
	}

	// Use source description if none provided
	desc := description
	if desc == "" {
		desc = source.Description
	}

	// Load all source instances
	sourceInstances, err := s.instRepo.ListByCatalog(ctx, source.ID)
	if err != nil {
		return nil, err
	}

	// Create new catalog
	now := time.Now()
	newCatalog := &models.Catalog{
		ID:               uuid.Must(uuid.NewV7()).String(),
		Name:             targetName,
		Description:      desc,
		CatalogVersionID: source.CatalogVersionID,
		ValidationStatus: models.ValidationStatusDraft,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	// Build old→new instance ID mapping
	idMap := make(map[string]string, len(sourceInstances))
	newInstances := make([]*models.EntityInstance, len(sourceInstances))
	for i, inst := range sourceInstances {
		newID := uuid.Must(uuid.NewV7()).String()
		idMap[inst.ID] = newID
		newInstances[i] = &models.EntityInstance{
			ID:           newID,
			EntityTypeID: inst.EntityTypeID,
			CatalogID:    newCatalog.ID,
			Name:         inst.Name,
			Description:  inst.Description,
			Version:      1,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
	}

	// Remap parent references
	for i, inst := range sourceInstances {
		if inst.ParentInstanceID != "" {
			if newParentID, ok := idMap[inst.ParentInstanceID]; ok {
				newInstances[i].ParentInstanceID = newParentID
			}
		}
	}

	// All DB mutations in a transaction for atomicity
	doMutations := func(txCtx context.Context) error {
		if err := s.catalogRepo.Create(txCtx, newCatalog); err != nil {
			return err
		}
		for _, inst := range newInstances {
			if err := s.instRepo.Create(txCtx, inst); err != nil {
				return err
			}
		}
		if s.iavRepo != nil {
			for _, srcInst := range sourceInstances {
				values, err := s.iavRepo.GetValuesForVersion(txCtx, srcInst.ID, srcInst.Version)
				if err != nil {
					return err
				}
				if len(values) == 0 {
					continue
				}
				newValues := make([]*models.InstanceAttributeValue, len(values))
				for j, v := range values {
					newValues[j] = &models.InstanceAttributeValue{
						ID:              uuid.Must(uuid.NewV7()).String(),
						InstanceID:      idMap[srcInst.ID],
						InstanceVersion: 1,
						AttributeID:     v.AttributeID,
						ValueString:     v.ValueString,
						ValueNumber:     v.ValueNumber,
						ValueJSON:       v.ValueJSON,
					}
				}
				if err := s.iavRepo.SetValues(txCtx, newValues); err != nil {
					return err
				}
			}
		}
		if s.linkRepo != nil {
			for _, srcInst := range sourceInstances {
				links, err := s.linkRepo.GetForwardRefs(txCtx, srcInst.ID)
				if err != nil {
					return err
				}
				for _, link := range links {
					newSourceID, srcOk := idMap[link.SourceInstanceID]
					newTargetID, tgtOk := idMap[link.TargetInstanceID]
					if !srcOk || !tgtOk {
						continue
					}
					if err := s.linkRepo.Create(txCtx, &models.AssociationLink{
						ID:               uuid.Must(uuid.NewV7()).String(),
						AssociationID:    link.AssociationID,
						SourceInstanceID: newSourceID,
						TargetInstanceID: newTargetID,
						CreatedAt:        now,
					}); err != nil {
						return err
					}
				}
			}
		}
		return nil
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

	return newCatalog, nil
}

// ReplaceCatalog atomically swaps a staging catalog into the name of an existing one.
func (s *CatalogService) ReplaceCatalog(ctx context.Context, sourceName, targetName, archiveName string) (*models.Catalog, error) {
	if sourceName == targetName {
		return nil, domainerrors.NewValidation("source and target catalog names must be different")
	}

	// Get source and target
	source, err := s.catalogRepo.GetByName(ctx, sourceName)
	if err != nil {
		return nil, err
	}
	target, err := s.catalogRepo.GetByName(ctx, targetName)
	if err != nil {
		return nil, err
	}

	// Source must be valid
	if source.ValidationStatus != models.ValidationStatusValid {
		return nil, domainerrors.NewValidation("source catalog must be valid to replace (current status: " + string(source.ValidationStatus) + ")")
	}

	// Generate or validate archive name
	archive := archiveName
	if archive == "" {
		archive = targetName + "-archive-" + time.Now().Format("20060102")
	}
	if err := ValidateCatalogName(archive); err != nil {
		if archiveName == "" {
			return nil, domainerrors.NewValidation("auto-generated archive name '" + archive + "' is invalid (too long); provide a shorter archive_name explicitly")
		}
		return nil, err
	}

	// All DB mutations in a transaction for atomicity
	doMutations := func(txCtx context.Context) error {
		// Step 1: Rename target → archive
		if err := s.catalogRepo.UpdateName(txCtx, target.ID, archive); err != nil {
			return err
		}
		// Step 2: Rename source → target
		if err := s.catalogRepo.UpdateName(txCtx, source.ID, targetName); err != nil {
			return err
		}
		// Step 3: Transfer published state
		if target.Published {
			now := time.Now()
			// Source (now named as target) inherits published state
			if err := s.catalogRepo.UpdatePublished(txCtx, source.ID, true, &now); err != nil {
				return err
			}
			// Archive becomes unpublished
			if err := s.catalogRepo.UpdatePublished(txCtx, target.ID, false, nil); err != nil {
				return err
			}
		} else if source.Published {
			// Source was published under old name — unpublish since it has a new name now
			if err := s.catalogRepo.UpdatePublished(txCtx, source.ID, false, nil); err != nil {
				return err
			}
		}
		return nil
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

	// CR operations outside transaction (K8s, not DB)
	if target.Published {
		if s.crManager != nil {
			_ = s.crManager.Delete(ctx, archive, s.namespace)
		}
		s.SyncCR(ctx, targetName)
	}
	if source.Published && s.crManager != nil {
		_ = s.crManager.Delete(ctx, sourceName, s.namespace)
	}

	// Update in-memory fields to match DB state for accurate API response
	source.Name = targetName
	if target.Published {
		now := time.Now()
		source.Published = true
		source.PublishedAt = &now
	} else if source.Published {
		source.Published = false
		source.PublishedAt = nil
	}

	return source, nil
}
