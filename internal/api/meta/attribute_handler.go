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

	// Prepend system attributes (Name — required, Description — optional)
	systemAttrs := []dto.AttributeResponse{
		{Name: models.SystemAttrName, Type: models.SystemAttrType, Ordinal: models.SystemAttrNameOrdinal, Required: true, System: true},
		{Name: models.SystemAttrDescription, Type: models.SystemAttrType, Ordinal: models.SystemAttrDescOrdinal, Required: false, System: true},
	}
	resp := make([]dto.AttributeResponse, 0, len(systemAttrs)+len(attrs))
	resp = append(resp, systemAttrs...)
	for _, a := range attrs {
		resp = append(resp, dto.AttributeResponse{
			ID: a.ID, Name: a.Name, Description: a.Description,
			Type: string(a.Type), EnumID: a.EnumID, Ordinal: a.Ordinal, Required: a.Required,
		})
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
	if models.IsSystemAttributeName(req.Name) {
		return echo.NewHTTPError(http.StatusBadRequest, "attribute name \""+req.Name+"\" is reserved for system attributes")
	}
	if req.Type == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "type is required")
	}

	newVersion, err := h.svc.AddAttribute(c.Request().Context(), entityTypeID, req.Name, req.Description, models.AttributeType(req.Type), req.EnumID, req.Required)
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

	if models.IsSystemAttributeName(name) {
		return echo.NewHTTPError(http.StatusBadRequest, "cannot remove system attribute \""+name+"\"")
	}

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

func (h *AttributeHandler) Edit(c echo.Context) error {
	entityTypeID := c.Param("entityTypeId")
	name := c.Param("name")

	var req dto.UpdateAttributeRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if req.Name != nil && models.IsSystemAttributeName(*req.Name) {
		return echo.NewHTTPError(http.StatusBadRequest, "attribute name \""+*req.Name+"\" is reserved for system attributes")
	}

	var newType *models.AttributeType
	if req.Type != nil {
		t := models.AttributeType(*req.Type)
		newType = &t
	}

	newVersion, err := h.svc.EditAttribute(c.Request().Context(), entityTypeID, name, req.Name, req.Description, newType, req.EnumID, req.Required)
	if err != nil {
		return mapError(err)
	}

	return c.JSON(http.StatusOK, dto.EntityTypeVersionResponse{
		ID: newVersion.ID, EntityTypeID: newVersion.EntityTypeID,
		Version: newVersion.Version, Description: newVersion.Description, CreatedAt: newVersion.CreatedAt,
	})
}

func (h *AttributeHandler) CopyAttributes(c echo.Context) error {
	entityTypeID := c.Param("entityTypeId")
	var req dto.CopyAttributesRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.SourceEntityTypeID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "source_entity_type_id is required")
	}
	if len(req.AttributeNames) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "attribute_names is required")
	}

	newVersion, err := h.svc.CopyAttributesFromType(c.Request().Context(), entityTypeID, req.SourceEntityTypeID, req.SourceVersion, req.AttributeNames)
	if err != nil {
		return mapError(err)
	}

	return c.JSON(http.StatusOK, dto.EntityTypeVersionResponse{
		ID: newVersion.ID, EntityTypeID: newVersion.EntityTypeID,
		Version: newVersion.Version, Description: newVersion.Description, CreatedAt: newVersion.CreatedAt,
	})
}

func RegisterAttributeRoutes(g *echo.Group, h *AttributeHandler, requireAdmin echo.MiddlewareFunc) {
	g.GET("/entity-types/:entityTypeId/attributes", h.List)
	g.POST("/entity-types/:entityTypeId/attributes", h.Add, requireAdmin)
	g.PUT("/entity-types/:entityTypeId/attributes/:name", h.Edit, requireAdmin)
	g.DELETE("/entity-types/:entityTypeId/attributes/:name", h.Remove, requireAdmin)
	g.PUT("/entity-types/:entityTypeId/attributes/reorder", h.Reorder, requireAdmin)
	g.POST("/entity-types/:entityTypeId/attributes/copy", h.CopyAttributes, requireAdmin)
}
