package meta

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/project-catalyst/pc-asset-hub/internal/api/dto"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	svcmeta "github.com/project-catalyst/pc-asset-hub/internal/service/meta"
	"github.com/project-catalyst/pc-asset-hub/internal/service/validation"
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
			Name: a.Name,
			TargetEntityTypeID: a.TargetEntityTypeID, Type: string(a.Type),
			SourceRole: a.SourceRole, TargetRole: a.TargetRole,
			SourceCardinality: validation.NormalizeSourceCardinality(a.SourceCardinality, a.Type == models.AssociationTypeContainment),
			TargetCardinality: validation.NormalizeCardinality(a.TargetCardinality),
			CreatedAt: a.CreatedAt,
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

	newVersion, err := h.svc.CreateAssociation(c.Request().Context(), entityTypeID, req.TargetEntityTypeID, models.AssociationType(req.Type), req.Name, req.SourceRole, req.TargetRole, req.SourceCardinality, req.TargetCardinality)
	if err != nil {
		return mapError(err)
	}

	return c.JSON(http.StatusCreated, dto.EntityTypeVersionResponse{
		ID: newVersion.ID, EntityTypeID: newVersion.EntityTypeID,
		Version: newVersion.Version, Description: newVersion.Description, CreatedAt: newVersion.CreatedAt,
	})
}

func (h *AssociationHandler) Edit(c echo.Context) error {
	entityTypeID := c.Param("entityTypeId")
	name := c.Param("name")

	var req dto.UpdateAssociationRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	var newType *models.AssociationType
	if req.Type != nil {
		t := models.AssociationType(*req.Type)
		newType = &t
	}
	newVersion, err := h.svc.EditAssociation(c.Request().Context(), entityTypeID, name, req.Name, req.SourceRole, req.TargetRole, req.SourceCardinality, req.TargetCardinality, newType)
	if err != nil {
		return mapError(err)
	}

	return c.JSON(http.StatusOK, dto.EntityTypeVersionResponse{
		ID: newVersion.ID, EntityTypeID: newVersion.EntityTypeID,
		Version: newVersion.Version, Description: newVersion.Description, CreatedAt: newVersion.CreatedAt,
	})
}

func (h *AssociationHandler) Delete(c echo.Context) error {
	entityTypeID := c.Param("entityTypeId")
	name := c.Param("name")

	_, err := h.svc.DeleteAssociation(c.Request().Context(), entityTypeID, name)
	if err != nil {
		return mapError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func RegisterAssociationRoutes(g *echo.Group, h *AssociationHandler, requireAdmin echo.MiddlewareFunc) {
	g.GET("/entity-types/:entityTypeId/associations", h.List)
	g.POST("/entity-types/:entityTypeId/associations", h.Create, requireAdmin)
	g.PUT("/entity-types/:entityTypeId/associations/:name", h.Edit, requireAdmin)
	g.DELETE("/entity-types/:entityTypeId/associations/:name", h.Delete, requireAdmin)
}
