package operational

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	apimw "github.com/project-catalyst/pc-asset-hub/internal/api/middleware"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/service/operational/export"
)

type ExportBindingHandler struct {
	svc           *export.ExportBindingService
	registry      *export.ExporterRegistry
	accessChecker apimw.CatalogAccessChecker
}

func NewExportBindingHandler(svc *export.ExportBindingService, registry *export.ExporterRegistry, accessChecker apimw.CatalogAccessChecker) *ExportBindingHandler {
	return &ExportBindingHandler{svc: svc, registry: registry, accessChecker: accessChecker}
}

func (h *ExportBindingHandler) ListExporters(c echo.Context) error {
	items := h.registry.List()
	return c.JSON(http.StatusOK, map[string]any{"items": items})
}

type createBindingRequest struct {
	ExporterName string            `json:"exporter_name"`
	Parameters   map[string]string `json:"parameters"`
}

func (h *ExportBindingHandler) CreateBinding(c echo.Context) error {
	catalogName := c.Param("catalog-name")
	var req createBindingRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	binding, err := h.svc.Create(c.Request().Context(), catalogName, req.ExporterName, req.Parameters)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(http.StatusCreated, bindingToDTO(binding))
}

func (h *ExportBindingHandler) ListBindings(c echo.Context) error {
	catalogName := c.Param("catalog-name")
	bindings, err := h.svc.List(c.Request().Context(), catalogName)
	if err != nil {
		return mapError(err)
	}
	items := make([]any, len(bindings))
	for i, b := range bindings {
		items[i] = bindingToDTO(b)
	}
	return c.JSON(http.StatusOK, map[string]any{"items": items})
}

func (h *ExportBindingHandler) GetBinding(c echo.Context) error {
	catalogName := c.Param("catalog-name")
	bindingID := c.Param("binding-id")
	binding, err := h.svc.Get(c.Request().Context(), catalogName, bindingID)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(http.StatusOK, bindingToDTO(binding))
}

type updateBindingRequest struct {
	Parameters map[string]string `json:"parameters,omitempty"`
	Enabled    *bool             `json:"enabled,omitempty"`
}

func (h *ExportBindingHandler) UpdateBinding(c echo.Context) error {
	catalogName := c.Param("catalog-name")
	bindingID := c.Param("binding-id")
	var req updateBindingRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	binding, err := h.svc.Update(c.Request().Context(), catalogName, bindingID, req.Parameters, req.Enabled)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(http.StatusOK, bindingToDTO(binding))
}

func (h *ExportBindingHandler) DeleteBinding(c echo.Context) error {
	catalogName := c.Param("catalog-name")
	bindingID := c.Param("binding-id")
	if err := h.svc.Delete(c.Request().Context(), catalogName, bindingID); err != nil {
		return mapError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *ExportBindingHandler) RunBinding(c echo.Context) error {
	catalogName := c.Param("catalog-name")
	bindingID := c.Param("binding-id")
	vsInstanceName := c.QueryParam("virtual_server_instance")
	output, err := h.svc.Run(c.Request().Context(), catalogName, bindingID, vsInstanceName)
	if err != nil {
		return mapError(err)
	}
	yamlContent := renderMultiDocYAML(output.Artifacts)
	c.Response().Header().Set("Content-Disposition",
		fmt.Sprintf(`attachment; filename="%s-export.yaml"`, catalogName))
	return c.Blob(http.StatusOK, "application/x-yaml", []byte(yamlContent))
}

func renderMultiDocYAML(artifacts []export.K8sArtifact) string {
	if len(artifacts) == 0 {
		return ""
	}
	var parts []string
	for _, a := range artifacts {
		parts = append(parts, a.YAML)
	}
	return "---\n" + strings.Join(parts, "\n---\n")
}

type bindingDTO struct {
	ID            string            `json:"id"`
	CatalogID     string            `json:"catalog_id"`
	ExporterName  string            `json:"exporter_name"`
	Parameters    map[string]string `json:"parameters"`
	Enabled       bool              `json:"enabled"`
	LastRunAt     *time.Time        `json:"last_run_at"`
	LastRunStatus string            `json:"last_run_status"`
	LastRunError  string            `json:"last_run_error,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

func bindingToDTO(b *models.ExportBinding) *bindingDTO {
	return &bindingDTO{
		ID:            b.ID,
		CatalogID:     b.CatalogID,
		ExporterName:  b.ExporterName,
		Parameters:    b.Parameters,
		Enabled:       b.Enabled,
		LastRunAt:     b.LastRunAt,
		LastRunStatus: b.LastRunStatus,
		LastRunError:  b.LastRunError,
		CreatedAt:     b.CreatedAt,
		UpdatedAt:     b.UpdatedAt,
	}
}

func (h *ExportBindingHandler) PublishPreview(c echo.Context) error {
	catalogName := c.Param("catalog-name")
	result, err := h.svc.PublishPreview(c.Request().Context(), catalogName)
	if err != nil {
		return mapError(err)
	}

	bindings := make([]map[string]any, len(result.Bindings))
	for i, b := range result.Bindings {
		bindings[i] = map[string]any{
			"binding_id":     b.BindingID,
			"exporter_name":  b.ExporterName,
			"status":         b.Status,
			"artifact_count": b.ArtifactCount,
			"error":          b.Error,
		}
	}

	return c.JSON(http.StatusOK, map[string]any{
		"session_token": result.SessionToken,
		"expires_at":    result.ExpiresAt,
		"bindings":      bindings,
		"has_failures":  result.HasFailures,
	})
}

func (h *ExportBindingHandler) DownloadBindingArtifacts(c echo.Context) error {
	catalogName := c.Param("catalog-name")
	token := c.QueryParam("token")
	bindingID := c.QueryParam("binding")

	if token == "" || bindingID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "token and binding query parameters are required")
	}

	artifacts, err := h.svc.GetCachedArtifacts(c.Request().Context(), catalogName, token, bindingID)
	if err != nil {
		return mapError(err)
	}

	var yamlContent string
	if len(artifacts) == 0 {
		yamlContent = fmt.Sprintf("# No instances found for export — catalog '%s'\n", catalogName)
	} else {
		yamlContent = renderMultiDocYAML(artifacts)
	}

	c.Response().Header().Set("Content-Disposition",
		fmt.Sprintf(`attachment; filename="%s-export.yaml"`, catalogName))
	return c.Blob(http.StatusOK, "application/x-yaml", []byte(yamlContent))
}

func RegisterExportBindingRoutes(g *echo.Group, h *ExportBindingHandler, requireRW, requireAdmin echo.MiddlewareFunc, writeGuards ...echo.MiddlewareFunc) {
	requireCatalogAccess := apimw.RequireCatalogAccess(h.accessChecker)
	adminWrite := append([]echo.MiddlewareFunc{requireAdmin}, writeGuards...)
	g.GET("/:catalog-name/export-bindings", h.ListBindings, requireCatalogAccess)
	g.POST("/:catalog-name/export-bindings", h.CreateBinding, append(adminWrite, requireCatalogAccess)...)
	g.GET("/:catalog-name/export-bindings/download", h.DownloadBindingArtifacts, requireRW, requireCatalogAccess)
	g.GET("/:catalog-name/export-bindings/:binding-id", h.GetBinding, requireCatalogAccess)
	g.PUT("/:catalog-name/export-bindings/:binding-id", h.UpdateBinding, append(adminWrite, requireCatalogAccess)...)
	g.DELETE("/:catalog-name/export-bindings/:binding-id", h.DeleteBinding, append(adminWrite, requireCatalogAccess)...)
	g.POST("/:catalog-name/export-bindings/:binding-id/run", h.RunBinding, requireRW, requireCatalogAccess)
	g.POST("/:catalog-name/publish/preview", h.PublishPreview, append(adminWrite, requireCatalogAccess)...)
}
