package export

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
)

type PublishPreviewResult struct {
	SessionToken string
	ExpiresAt    time.Time
	Bindings     []BindingRunResult
	HasFailures  bool
}

func (s *ExportBindingService) PublishPreview(ctx context.Context, catalogName string) (*PublishPreviewResult, error) {
	catalog, err := s.catalogRepo.GetByName(ctx, catalogName)
	if err != nil {
		return nil, err
	}

	bindings, err := s.bindingRepo.ListByCatalog(ctx, catalog.ID)
	if err != nil {
		return nil, err
	}

	schema, err := s.buildSchemaInfo(ctx, catalog.CatalogVersionID)
	if err != nil {
		return nil, err
	}

	artifactsByBinding := map[string][]K8sArtifact{}
	var results []BindingRunResult
	hasFailures := false

	for _, binding := range bindings {
		if !binding.Enabled {
			continue
		}

		if exporter, ok := s.registry.Get(binding.ExporterName); ok {
			if err := exporter.ValidateSchema(binding.Parameters, schema); err != nil {
				results = append(results, BindingRunResult{
					BindingID:    binding.ID,
					ExporterName: binding.ExporterName,
					Status:       "failed",
					Error:        err.Error(),
				})
				hasFailures = true
				continue
			}
		}

		result := s.executeBinding(ctx, catalog, binding)
		if result.Status == "failed" {
			hasFailures = true
		} else {
			artifactsByBinding[binding.ID] = result.Artifacts
		}
		results = append(results, result)
	}

	token := uuid.Must(uuid.NewV7()).String()
	ttl := s.getPreviewTTL()
	expiresAt := time.Now().Add(ttl)

	if s.previewCache != nil {
		entry := PreviewCacheEntry{
			CatalogName:      catalogName,
			Artifacts:        artifactsByBinding,
			BindingResults:   results,
			CatalogUpdatedAt: catalog.UpdatedAt,
		}
		if err := s.previewCache.Store(token, entry, ttl); err != nil {
			return nil, err
		}
	}

	return &PublishPreviewResult{
		SessionToken: token,
		ExpiresAt:    expiresAt,
		Bindings:     results,
		HasFailures:  hasFailures,
	}, nil
}

func (s *ExportBindingService) GetCachedArtifacts(ctx context.Context, catalogName, token, bindingID string) ([]K8sArtifact, error) {
	if s.previewCache == nil {
		return nil, domainerrors.NewValidation("preview cache not configured")
	}

	entry, err := s.previewCache.Retrieve(token)
	if err != nil {
		return nil, domainerrors.NewNotFound("preview session", token)
	}

	if entry.CatalogName != catalogName {
		return nil, domainerrors.NewNotFound("preview session", token)
	}

	artifacts, ok := entry.Artifacts[bindingID]
	if !ok {
		return nil, domainerrors.NewNotFound("binding artifacts", bindingID)
	}
	return artifacts, nil
}

func (s *ExportBindingService) GetPreviewEntry(token string) (*PreviewCacheEntry, error) {
	if s.previewCache == nil {
		return nil, domainerrors.NewValidation("preview cache not configured")
	}
	return s.previewCache.Retrieve(token)
}

func (s *ExportBindingService) getPreviewTTL() time.Duration {
	if v := os.Getenv("PUBLISH_PREVIEW_TTL"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil {
			return time.Duration(secs) * time.Second
		}
	}
	return 5 * time.Minute
}
