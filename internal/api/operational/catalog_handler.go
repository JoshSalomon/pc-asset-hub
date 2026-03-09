package operational

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/project-catalyst/pc-asset-hub/internal/api/dto"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	svcop "github.com/project-catalyst/pc-asset-hub/internal/service/operational"
)

type CatalogHandler struct {
	svc *svcop.CatalogService
}

func NewCatalogHandler(svc *svcop.CatalogService) *CatalogHandler {
	return &CatalogHandler{svc: svc}
}

func (h *CatalogHandler) CreateCatalog(c echo.Context) error {
	var req dto.CreateCatalogRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
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

	details, total, err := h.svc.List(c.Request().Context(), params)
	if err != nil {
		return mapError(err)
	}

	items := make([]dto.CatalogResponse, len(details))
	for i, d := range details {
		items[i] = catalogToDTO(d.Catalog, d.CatalogVersionLabel)
	}

	return c.JSON(http.StatusOK, dto.ListResponse{Items: items, Total: total})
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

func catalogToDTO(cat *models.Catalog, cvLabel string) dto.CatalogResponse {
	return dto.CatalogResponse{
		ID:                  cat.ID,
		Name:                cat.Name,
		Description:         cat.Description,
		CatalogVersionID:    cat.CatalogVersionID,
		CatalogVersionLabel: cvLabel,
		ValidationStatus:    string(cat.ValidationStatus),
		CreatedAt:           cat.CreatedAt,
		UpdatedAt:           cat.UpdatedAt,
	}
}

func RegisterCatalogRoutes(g *echo.Group, h *CatalogHandler, requireRW echo.MiddlewareFunc) {
	g.POST("", h.CreateCatalog, requireRW)
	g.GET("", h.ListCatalogs)
	g.GET("/:catalog-name", h.GetCatalog)
	g.DELETE("/:catalog-name", h.DeleteCatalog, requireRW)
}
