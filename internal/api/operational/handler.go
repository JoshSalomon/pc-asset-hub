package operational

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/project-catalyst/pc-asset-hub/internal/api/dto"
	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	svcop "github.com/project-catalyst/pc-asset-hub/internal/service/operational"
)

type Handler struct {
	svc *svcop.EntityInstanceService
}

func NewHandler(svc *svcop.EntityInstanceService) *Handler {
	return &Handler{svc: svc}
}

func mapError(err error) *echo.HTTPError {
	if domainerrors.IsNotFound(err) {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}
	if domainerrors.IsConflict(err) {
		return echo.NewHTTPError(http.StatusConflict, err.Error())
	}
	if domainerrors.IsValidation(err) {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if domainerrors.IsForbidden(err) {
		return echo.NewHTTPError(http.StatusForbidden, err.Error())
	}
	// Do not leak internal error details to clients
	return echo.NewHTTPError(http.StatusInternalServerError, "internal server error")
}

func instanceToResponse(inst *models.EntityInstance) map[string]any {
	return map[string]any{
		"id":                 inst.ID,
		"entity_type_id":     inst.EntityTypeID,
		"catalog_version_id": inst.CatalogVersionID,
		"parent_instance_id": inst.ParentInstanceID,
		"name":               inst.Name,
		"description":        inst.Description,
		"version":            inst.Version,
		"created_at":         inst.CreatedAt,
		"updated_at":         inst.UpdatedAt,
	}
}

func (h *Handler) CreateInstance(c echo.Context) error {
	cv := c.Param("catalog-version")
	entityType := c.Param("entity-type")

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	inst, err := h.svc.CreateInstance(c.Request().Context(), entityType, cv, "", req.Name, req.Description, nil)
	if err != nil {
		return mapError(err)
	}

	return c.JSON(http.StatusCreated, instanceToResponse(inst))
}

func (h *Handler) ListInstances(c echo.Context) error {
	cv := c.Param("catalog-version")
	entityType := c.Param("entity-type")

	params := models.ListParams{
		Limit:  20,
		Offset: 0,
	}

	items, total, err := h.svc.ListInstances(c.Request().Context(), entityType, cv, params)
	if err != nil {
		return mapError(err)
	}

	resp := make([]map[string]any, len(items))
	for i, inst := range items {
		resp[i] = instanceToResponse(inst)
	}
	return c.JSON(http.StatusOK, dto.ListResponse{Items: resp, Total: total})
}

func (h *Handler) GetInstance(c echo.Context) error {
	id := c.Param("id")
	inst, err := h.svc.GetInstance(c.Request().Context(), id)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(http.StatusOK, instanceToResponse(inst))
}

func (h *Handler) UpdateInstance(c echo.Context) error {
	id := c.Param("id")
	var req struct {
		Version int `json:"version"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	inst, err := h.svc.UpdateInstance(c.Request().Context(), id, req.Version)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(http.StatusOK, instanceToResponse(inst))
}

func (h *Handler) DeleteInstance(c echo.Context) error {
	id := c.Param("id")
	if err := h.svc.CascadeDelete(c.Request().Context(), id); err != nil {
		return mapError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) GetForwardReferences(c echo.Context) error {
	id := c.Param("id")
	refs, err := h.svc.GetForwardReferences(c.Request().Context(), id)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(http.StatusOK, refs)
}

// RegisterRoutes registers operational API routes.
func RegisterRoutes(g *echo.Group, h *Handler) {
	g.POST("/:entity-type", h.CreateInstance)
	g.GET("/:entity-type", h.ListInstances)
	g.GET("/:entity-type/:id", h.GetInstance)
	g.PUT("/:entity-type/:id", h.UpdateInstance)
	g.DELETE("/:entity-type/:id", h.DeleteInstance)
	g.GET("/:entity-type/:id/references", h.GetForwardReferences)
}
