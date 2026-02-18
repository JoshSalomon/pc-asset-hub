package meta

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/project-catalyst/pc-asset-hub/internal/api/dto"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	svcmeta "github.com/project-catalyst/pc-asset-hub/internal/service/meta"
)

type AttributeHandler struct {
	svc *svcmeta.AttributeService
}

func NewAttributeHandler(svc *svcmeta.AttributeService) *AttributeHandler {
	return &AttributeHandler{svc: svc}
}

func (h *AttributeHandler) List(c echo.Context) error {
	entityTypeID := c.Param("entityTypeId")
	attrs, err := h.svc.ListAttributes(c.Request().Context(), entityTypeID)
	if err != nil {
		return mapError(err)
	}

	resp := make([]dto.AttributeResponse, len(attrs))
	for i, a := range attrs {
		resp[i] = dto.AttributeResponse{
			ID: a.ID, Name: a.Name, Description: a.Description,
			Type: string(a.Type), EnumID: a.EnumID, Ordinal: a.Ordinal, Required: a.Required,
		}
	}
	return c.JSON(http.StatusOK, dto.ListResponse{Items: resp, Total: len(resp)})
}

func (h *AttributeHandler) Add(c echo.Context) error {
	entityTypeID := c.Param("entityTypeId")
	var req dto.CreateAttributeRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}
	if req.Type == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "type is required")
	}

	newVersion, err := h.svc.AddAttribute(c.Request().Context(), entityTypeID, req.Name, req.Description, models.AttributeType(req.Type), req.EnumID)
	if err != nil {
		return mapError(err)
	}

	return c.JSON(http.StatusCreated, dto.EntityTypeVersionResponse{
		ID: newVersion.ID, EntityTypeID: newVersion.EntityTypeID,
		Version: newVersion.Version, Description: newVersion.Description, CreatedAt: newVersion.CreatedAt,
	})
}

func (h *AttributeHandler) Remove(c echo.Context) error {
	entityTypeID := c.Param("entityTypeId")
	name := c.Param("name")

	_, err := h.svc.RemoveAttribute(c.Request().Context(), entityTypeID, name)
	if err != nil {
		return mapError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *AttributeHandler) Reorder(c echo.Context) error {
	entityTypeID := c.Param("entityTypeId")
	var req dto.ReorderAttributesRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if len(req.OrderedIDs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "ordered_ids is required")
	}

	if err := h.svc.ReorderAttributes(c.Request().Context(), entityTypeID, req.OrderedIDs); err != nil {
		return mapError(err)
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "reordered"})
}

func RegisterAttributeRoutes(g *echo.Group, h *AttributeHandler, requireAdmin echo.MiddlewareFunc) {
	g.GET("/entity-types/:entityTypeId/attributes", h.List)
	g.POST("/entity-types/:entityTypeId/attributes", h.Add, requireAdmin)
	g.DELETE("/entity-types/:entityTypeId/attributes/:name", h.Remove, requireAdmin)
	g.PUT("/entity-types/:entityTypeId/attributes/reorder", h.Reorder, requireAdmin)
}
