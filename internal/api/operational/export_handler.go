package operational

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	svcop "github.com/project-catalyst/pc-asset-hub/internal/service/operational"
)

type ExportHandler struct {
	svc *svcop.ExportService
}

func NewExportHandler(svc *svcop.ExportService) *ExportHandler {
	return &ExportHandler{svc: svc}
}

func (h *ExportHandler) ExportCatalog(c echo.Context) error {
	catalogName := c.Param("catalog-name")

	// Parse query params
	var entityFilter []string
	if entities := c.QueryParam("entities"); entities != "" {
		entityFilter = strings.Split(entities, ",")
	}
	sourceSystem := c.QueryParam("source_system")

	data, err := h.svc.ExportCatalog(c.Request().Context(), catalogName, entityFilter, sourceSystem)
	if err != nil {
		return mapError(err)
	}

	c.Response().Header().Set("Content-Disposition",
		fmt.Sprintf(`attachment; filename="%s-export.json"`, catalogName))
	return c.JSON(http.StatusOK, data)
}

func RegisterExportRoutes(g *echo.Group, h *ExportHandler, requireAdmin echo.MiddlewareFunc) {
	g.GET("/:catalog-name/export", h.ExportCatalog, requireAdmin)
}
