package meta

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/project-catalyst/pc-asset-hub/internal/api/dto"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	svcmeta "github.com/project-catalyst/pc-asset-hub/internal/service/meta"
)

type EntityTypeHandler struct {
	svc *svcmeta.EntityTypeService
}

func NewEntityTypeHandler(svc *svcmeta.EntityTypeService) *EntityTypeHandler {
	return &EntityTypeHandler{svc: svc}
}

func (h *EntityTypeHandler) Create(c echo.Context) error {
	var req dto.CreateEntityTypeRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}

	et, etv, err := h.svc.CreateEntityType(c.Request().Context(), req.Name, req.Description)
	if err != nil {
		return mapError(err)
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"entity_type": dto.EntityTypeResponse{
			ID: et.ID, Name: et.Name, CreatedAt: et.CreatedAt, UpdatedAt: et.UpdatedAt,
		},
		"version": dto.EntityTypeVersionResponse{
			ID: etv.ID, EntityTypeID: etv.EntityTypeID, Version: etv.Version, Description: etv.Description, CreatedAt: etv.CreatedAt,
		},
	})
}

func (h *EntityTypeHandler) List(c echo.Context) error {
	params := models.ListParams{
		Limit:  20,
		Offset: 0,
	}
	if name := c.QueryParam("name"); name != "" {
		params.Filters = map[string]string{"name": name}
	}

	items, total, err := h.svc.ListEntityTypes(c.Request().Context(), params)
	if err != nil {
		return mapError(err)
	}

	resp := make([]dto.EntityTypeResponse, len(items))
	for i, et := range items {
		resp[i] = dto.EntityTypeResponse{
			ID: et.ID, Name: et.Name, CreatedAt: et.CreatedAt, UpdatedAt: et.UpdatedAt,
		}
	}
	return c.JSON(http.StatusOK, dto.ListResponse{Items: resp, Total: total})
}

func (h *EntityTypeHandler) GetByID(c echo.Context) error {
	id := c.Param("id")
	et, err := h.svc.GetEntityType(c.Request().Context(), id)
	if err != nil {
		return mapError(err)
	}

	return c.JSON(http.StatusOK, dto.EntityTypeResponse{
		ID: et.ID, Name: et.Name, CreatedAt: et.CreatedAt, UpdatedAt: et.UpdatedAt,
	})
}

func (h *EntityTypeHandler) Update(c echo.Context) error {
	id := c.Param("id")
	var req dto.UpdateEntityTypeRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	newVersion, err := h.svc.UpdateEntityType(c.Request().Context(), id, req.Description)
	if err != nil {
		return mapError(err)
	}

	return c.JSON(http.StatusOK, dto.EntityTypeVersionResponse{
		ID: newVersion.ID, EntityTypeID: newVersion.EntityTypeID, Version: newVersion.Version,
		Description: newVersion.Description, CreatedAt: newVersion.CreatedAt,
	})
}

func (h *EntityTypeHandler) Delete(c echo.Context) error {
	id := c.Param("id")
	if err := h.svc.DeleteEntityType(c.Request().Context(), id); err != nil {
		return mapError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *EntityTypeHandler) Copy(c echo.Context) error {
	id := c.Param("id")
	var req dto.CopyEntityTypeRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.NewName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "new_name is required")
	}

	et, etv, err := h.svc.CopyEntityType(c.Request().Context(), id, req.SourceVersion, req.NewName)
	if err != nil {
		return mapError(err)
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"entity_type": dto.EntityTypeResponse{
			ID: et.ID, Name: et.Name, CreatedAt: et.CreatedAt, UpdatedAt: et.UpdatedAt,
		},
		"version": dto.EntityTypeVersionResponse{
			ID: etv.ID, EntityTypeID: etv.EntityTypeID, Version: etv.Version,
		},
	})
}

// RegisterEntityTypeRoutes registers entity type routes on the given Echo group.
func RegisterEntityTypeRoutes(g *echo.Group, h *EntityTypeHandler, requireAdmin echo.MiddlewareFunc) {
	g.GET("/entity-types", h.List)
	g.GET("/entity-types/:id", h.GetByID)
	g.POST("/entity-types", h.Create, requireAdmin)
	g.PUT("/entity-types/:id", h.Update, requireAdmin)
	g.DELETE("/entity-types/:id", h.Delete, requireAdmin)
	g.POST("/entity-types/:id/copy", h.Copy, requireAdmin)
}
