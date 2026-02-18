package meta

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/project-catalyst/pc-asset-hub/internal/api/dto"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	svcmeta "github.com/project-catalyst/pc-asset-hub/internal/service/meta"
)

type VersionHistoryHandler struct {
	svc *svcmeta.VersionHistoryService
}

func NewVersionHistoryHandler(svc *svcmeta.VersionHistoryService) *VersionHistoryHandler {
	return &VersionHistoryHandler{svc: svc}
}

func (h *VersionHistoryHandler) List(c echo.Context) error {
	entityTypeID := c.Param("entityTypeId")
	versions, err := h.svc.GetVersionHistory(c.Request().Context(), entityTypeID)
	if err != nil {
		return mapError(err)
	}

	resp := make([]dto.EntityTypeVersionResponse, len(versions))
	for i, v := range versions {
		resp[i] = dto.EntityTypeVersionResponse{
			ID: v.ID, EntityTypeID: v.EntityTypeID,
			Version: v.Version, Description: v.Description, CreatedAt: v.CreatedAt,
		}
	}
	return c.JSON(http.StatusOK, dto.ListResponse{Items: resp, Total: len(resp)})
}

func (h *VersionHistoryHandler) Diff(c echo.Context) error {
	entityTypeID := c.Param("entityTypeId")

	v1Str := c.QueryParam("v1")
	v2Str := c.QueryParam("v2")
	if v1Str == "" || v2Str == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "v1 and v2 query parameters are required")
	}

	v1, err := strconv.Atoi(v1Str)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "v1 must be an integer")
	}
	v2, err := strconv.Atoi(v2Str)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "v2 must be an integer")
	}

	diff, err := h.svc.CompareVersions(c.Request().Context(), entityTypeID, v1, v2)
	if err != nil {
		return mapError(err)
	}

	changes := make([]dto.VersionDiffItemDTO, len(diff.Changes))
	for i, ch := range diff.Changes {
		changes[i] = dto.VersionDiffItemDTO{
			Name: ch.Name, ChangeType: ch.ChangeType, Category: ch.Category,
			OldValue: ch.OldValue, NewValue: ch.NewValue,
		}
	}

	return c.JSON(http.StatusOK, dto.VersionDiffResponse{
		FromVersion: diff.FromVersion, ToVersion: diff.ToVersion, Changes: changes,
	})
}

func RegisterVersionHistoryRoutes(g *echo.Group, h *VersionHistoryHandler) {
	g.GET("/entity-types/:entityTypeId/versions", h.List)
	g.GET("/entity-types/:entityTypeId/versions/diff", h.Diff)
}

// defaultListParams returns sensible defaults for list operations.
func defaultListParams() models.ListParams {
	return models.ListParams{Limit: 100, Offset: 0}
}
