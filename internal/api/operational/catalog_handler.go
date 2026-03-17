package operational

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/project-catalyst/pc-asset-hub/internal/api/dto"
	apimw "github.com/project-catalyst/pc-asset-hub/internal/api/middleware"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	svcop "github.com/project-catalyst/pc-asset-hub/internal/service/operational"
)

type CatalogHandler struct {
	svc           *svcop.CatalogService
	validationSvc *svcop.CatalogValidationService
	accessChecker apimw.CatalogAccessChecker
}

func NewCatalogHandler(svc *svcop.CatalogService, validationSvc *svcop.CatalogValidationService, accessChecker apimw.CatalogAccessChecker) *CatalogHandler {
	return &CatalogHandler{svc: svc, validationSvc: validationSvc, accessChecker: accessChecker}
}

func (h *CatalogHandler) CreateCatalog(c echo.Context) error {
	var req dto.CreateCatalogRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	// Check if user can create this catalog
	allowed, err := h.accessChecker.CheckAccess(c, req.Name, "create")
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "access check failed")
	}
	if !allowed {
		return echo.NewHTTPError(http.StatusForbidden, "access denied to catalog: "+req.Name)
	}

	catalog, err := h.svc.CreateCatalog(c.Request().Context(), req.Name, req.Description, req.CatalogVersionID)
	if err != nil {
		return mapError(err)
	}

	return c.JSON(http.StatusCreated, catalogToDTO(catalog, ""))
}

func (h *CatalogHandler) ListCatalogs(c echo.Context) error {
	params := models.ListParams{
		Limit:   20,
		Filters: make(map[string]string),
	}

	if cvID := c.QueryParam("catalog_version_id"); cvID != "" {
		params.Filters["catalog_version_id"] = cvID
	}
	if status := c.QueryParam("validation_status"); status != "" {
		params.Filters["validation_status"] = status
	}

	details, _, err := h.svc.List(c.Request().Context(), params)
	if err != nil {
		return mapError(err)
	}

	// Filter by catalog-level access
	accessible, err := apimw.FilterAccessible(c, h.accessChecker, details, func(d *svcop.CatalogDetail) string {
		return d.Catalog.Name
	})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "access check failed")
	}

	items := make([]dto.CatalogResponse, len(accessible))
	for i, d := range accessible {
		items[i] = catalogToDTO(d.Catalog, d.CatalogVersionLabel)
	}

	return c.JSON(http.StatusOK, dto.ListResponse{Items: items, Total: len(accessible)})
}

func (h *CatalogHandler) GetCatalog(c echo.Context) error {
	name := c.Param("catalog-name")

	detail, err := h.svc.GetByName(c.Request().Context(), name)
	if err != nil {
		return mapError(err)
	}

	return c.JSON(http.StatusOK, catalogToDTO(detail.Catalog, detail.CatalogVersionLabel))
}

func (h *CatalogHandler) DeleteCatalog(c echo.Context) error {
	name := c.Param("catalog-name")

	if err := h.svc.Delete(c.Request().Context(), name); err != nil {
		return mapError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *CatalogHandler) ValidateCatalog(c echo.Context) error {
	if h.validationSvc == nil {
		return echo.NewHTTPError(http.StatusNotImplemented, "validation service not configured")
	}

	name := c.Param("catalog-name")

	result, err := h.validationSvc.Validate(c.Request().Context(), name)
	if err != nil {
		return mapError(err)
	}

	errors := make([]dto.ValidationErrorResponse, len(result.Errors))
	for i, e := range result.Errors {
		errors[i] = dto.ValidationErrorResponse{
			EntityType:   e.EntityType,
			InstanceName: e.InstanceName,
			Field:        e.Field,
			Violation:    e.Violation,
		}
	}

	return c.JSON(http.StatusOK, dto.ValidationResultResponse{
		Status: string(result.Status),
		Errors: errors,
	})
}

func (h *CatalogHandler) PublishCatalog(c echo.Context) error {
	name := c.Param("catalog-name")
	if err := h.svc.Publish(c.Request().Context(), name); err != nil {
		return mapError(err)
	}
	return c.JSON(http.StatusOK, dto.StatusResponse{Status: "published"})
}

func (h *CatalogHandler) UnpublishCatalog(c echo.Context) error {
	name := c.Param("catalog-name")
	if err := h.svc.Unpublish(c.Request().Context(), name); err != nil {
		return mapError(err)
	}
	return c.JSON(http.StatusOK, dto.StatusResponse{Status: "unpublished"})
}

func catalogToDTO(cat *models.Catalog, cvLabel string) dto.CatalogResponse {
	return dto.CatalogResponse{
		ID:                  cat.ID,
		Name:                cat.Name,
		Description:         cat.Description,
		CatalogVersionID:    cat.CatalogVersionID,
		CatalogVersionLabel: cvLabel,
		ValidationStatus:    string(cat.ValidationStatus),
		Published:           cat.Published,
		PublishedAt:         cat.PublishedAt,
		CreatedAt:           cat.CreatedAt,
		UpdatedAt:           cat.UpdatedAt,
	}
}

func RegisterCatalogRoutes(g *echo.Group, h *CatalogHandler, requireRW, requireAdmin echo.MiddlewareFunc) {
	requireCatalogAccess := apimw.RequireCatalogAccess(h.accessChecker)
	g.POST("", h.CreateCatalog, requireRW)
	g.GET("", h.ListCatalogs)
	g.GET("/:catalog-name", h.GetCatalog, requireCatalogAccess)
	g.DELETE("/:catalog-name", h.DeleteCatalog, requireRW, requireCatalogAccess)
	g.POST("/:catalog-name/validate", h.ValidateCatalog, requireRW, requireCatalogAccess)
	g.POST("/:catalog-name/publish", h.PublishCatalog, requireAdmin, requireCatalogAccess)
	g.POST("/:catalog-name/unpublish", h.UnpublishCatalog, requireAdmin, requireCatalogAccess)
}
