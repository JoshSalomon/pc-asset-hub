package meta

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/project-catalyst/pc-asset-hub/internal/api/dto"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	svcmeta "github.com/project-catalyst/pc-asset-hub/internal/service/meta"
)

type AssociationHandler struct {
	svc *svcmeta.AssociationService
}

func NewAssociationHandler(svc *svcmeta.AssociationService) *AssociationHandler {
	return &AssociationHandler{svc: svc}
}

func (h *AssociationHandler) List(c echo.Context) error {
	entityTypeID := c.Param("entityTypeId")
	assocs, err := h.svc.ListAllAssociations(c.Request().Context(), entityTypeID)
	if err != nil {
		return mapError(err)
	}

	resp := make([]dto.AssociationResponse, len(assocs))
	for i, a := range assocs {
		resp[i] = dto.AssociationResponse{
			ID: a.ID, EntityTypeVersionID: a.EntityTypeVersionID,
			TargetEntityTypeID: a.TargetEntityTypeID, Type: string(a.Type),
			SourceRole: a.SourceRole, TargetRole: a.TargetRole, CreatedAt: a.CreatedAt,
			Direction: a.Direction, SourceEntityTypeID: a.SourceEntityTypeID,
		}
	}
	return c.JSON(http.StatusOK, dto.ListResponse{Items: resp, Total: len(resp)})
}

func (h *AssociationHandler) Create(c echo.Context) error {
	entityTypeID := c.Param("entityTypeId")
	var req dto.CreateAssociationRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.TargetEntityTypeID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "target_entity_type_id is required")
	}
	if req.Type == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "type is required")
	}

	newVersion, err := h.svc.CreateAssociation(c.Request().Context(), entityTypeID, req.TargetEntityTypeID, models.AssociationType(req.Type), req.SourceRole, req.TargetRole)
	if err != nil {
		return mapError(err)
	}

	return c.JSON(http.StatusCreated, dto.EntityTypeVersionResponse{
		ID: newVersion.ID, EntityTypeID: newVersion.EntityTypeID,
		Version: newVersion.Version, Description: newVersion.Description, CreatedAt: newVersion.CreatedAt,
	})
}

func (h *AssociationHandler) Delete(c echo.Context) error {
	entityTypeID := c.Param("entityTypeId")
	assocID := c.Param("associationId")

	_, err := h.svc.DeleteAssociation(c.Request().Context(), entityTypeID, assocID)
	if err != nil {
		return mapError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func RegisterAssociationRoutes(g *echo.Group, h *AssociationHandler, requireAdmin echo.MiddlewareFunc) {
	g.GET("/entity-types/:entityTypeId/associations", h.List)
	g.POST("/entity-types/:entityTypeId/associations", h.Create, requireAdmin)
	g.DELETE("/entity-types/:entityTypeId/associations/:associationId", h.Delete, requireAdmin)
}
