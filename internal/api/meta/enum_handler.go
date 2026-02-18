package meta

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/project-catalyst/pc-asset-hub/internal/api/dto"
	svcmeta "github.com/project-catalyst/pc-asset-hub/internal/service/meta"
)

type EnumHandler struct {
	svc *svcmeta.EnumService
}

func NewEnumHandler(svc *svcmeta.EnumService) *EnumHandler {
	return &EnumHandler{svc: svc}
}

func (h *EnumHandler) List(c echo.Context) error {
	enums, total, err := h.svc.ListEnums(c.Request().Context(), defaultListParams())
	if err != nil {
		return mapError(err)
	}

	resp := make([]dto.EnumResponse, len(enums))
	for i, e := range enums {
		resp[i] = dto.EnumResponse{
			ID: e.ID, Name: e.Name, CreatedAt: e.CreatedAt, UpdatedAt: e.UpdatedAt,
		}
	}
	return c.JSON(http.StatusOK, dto.ListResponse{Items: resp, Total: total})
}

func (h *EnumHandler) Create(c echo.Context) error {
	var req dto.CreateEnumRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}

	e, err := h.svc.CreateEnum(c.Request().Context(), req.Name, req.Values)
	if err != nil {
		return mapError(err)
	}

	return c.JSON(http.StatusCreated, dto.EnumResponse{
		ID: e.ID, Name: e.Name, CreatedAt: e.CreatedAt, UpdatedAt: e.UpdatedAt,
	})
}

func (h *EnumHandler) GetByID(c echo.Context) error {
	id := c.Param("id")
	e, err := h.svc.GetEnum(c.Request().Context(), id)
	if err != nil {
		return mapError(err)
	}

	return c.JSON(http.StatusOK, dto.EnumResponse{
		ID: e.ID, Name: e.Name, CreatedAt: e.CreatedAt, UpdatedAt: e.UpdatedAt,
	})
}

func (h *EnumHandler) Update(c echo.Context) error {
	id := c.Param("id")
	var req dto.UpdateEnumRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}

	if err := h.svc.UpdateEnum(c.Request().Context(), id, req.Name); err != nil {
		return mapError(err)
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "updated"})
}

func (h *EnumHandler) Delete(c echo.Context) error {
	id := c.Param("id")
	if err := h.svc.DeleteEnum(c.Request().Context(), id); err != nil {
		return mapError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *EnumHandler) ListValues(c echo.Context) error {
	enumID := c.Param("id")
	values, err := h.svc.ListValues(c.Request().Context(), enumID)
	if err != nil {
		return mapError(err)
	}

	resp := make([]dto.EnumValueResponse, len(values))
	for i, v := range values {
		resp[i] = dto.EnumValueResponse{
			ID: v.ID, Value: v.Value, Ordinal: v.Ordinal,
		}
	}
	return c.JSON(http.StatusOK, dto.ListResponse{Items: resp, Total: len(resp)})
}

func (h *EnumHandler) AddValue(c echo.Context) error {
	enumID := c.Param("id")
	var req dto.AddEnumValueRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.Value == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "value is required")
	}

	if err := h.svc.AddValue(c.Request().Context(), enumID, req.Value); err != nil {
		return mapError(err)
	}
	return c.JSON(http.StatusCreated, map[string]string{"status": "added"})
}

func (h *EnumHandler) RemoveValue(c echo.Context) error {
	valueID := c.Param("valueId")
	if err := h.svc.RemoveValue(c.Request().Context(), valueID); err != nil {
		return mapError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *EnumHandler) ReorderValues(c echo.Context) error {
	enumID := c.Param("id")
	var req dto.ReorderEnumValuesRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if len(req.OrderedIDs) == 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "ordered_ids is required")
	}

	if err := h.svc.ReorderValues(c.Request().Context(), enumID, req.OrderedIDs); err != nil {
		return mapError(err)
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "reordered"})
}

func RegisterEnumRoutes(g *echo.Group, h *EnumHandler, requireAdmin echo.MiddlewareFunc) {
	g.GET("/enums", h.List)
	g.POST("/enums", h.Create, requireAdmin)
	g.GET("/enums/:id", h.GetByID)
	g.PUT("/enums/:id", h.Update, requireAdmin)
	g.DELETE("/enums/:id", h.Delete, requireAdmin)
	g.GET("/enums/:id/values", h.ListValues)
	g.POST("/enums/:id/values", h.AddValue, requireAdmin)
	g.DELETE("/enums/:id/values/:valueId", h.RemoveValue, requireAdmin)
	g.PUT("/enums/:id/values/reorder", h.ReorderValues, requireAdmin)
}
