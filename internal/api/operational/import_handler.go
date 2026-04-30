package operational

import (
	"net/http"

	"github.com/labstack/echo/v4"

	apimw "github.com/project-catalyst/pc-asset-hub/internal/api/middleware"
	svcop "github.com/project-catalyst/pc-asset-hub/internal/service/operational"
)

type ImportHandler struct {
	svc           *svcop.ImportService
	accessChecker apimw.CatalogAccessChecker
}

func NewImportHandler(svc *svcop.ImportService, accessChecker apimw.CatalogAccessChecker) *ImportHandler {
	return &ImportHandler{svc: svc, accessChecker: accessChecker}
}

func (h *ImportHandler) ImportCatalog(c echo.Context) error {
	dryRunParam := c.QueryParam("dry_run")
	if dryRunParam != "" && dryRunParam != "true" && dryRunParam != "false" {
		return echo.NewHTTPError(http.StatusBadRequest, "dry_run must be 'true', 'false', or absent")
	}

	var req svcop.ImportRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if dryRunParam == "true" {
		result, err := h.svc.DryRun(c.Request().Context(), &req)
		if err != nil {
			return mapError(err)
		}
		return c.JSON(http.StatusOK, result)
	}

	result, err := h.svc.Import(c.Request().Context(), &req)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(http.StatusCreated, result)
}

func RegisterImportRoutes(g *echo.Group, h *ImportHandler, requireAdmin echo.MiddlewareFunc) {
	g.POST("/import", h.ImportCatalog, requireAdmin)
}
