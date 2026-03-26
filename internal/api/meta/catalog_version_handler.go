package meta

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/project-catalyst/pc-asset-hub/internal/api/dto"
	"github.com/project-catalyst/pc-asset-hub/internal/api/middleware"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	svcmeta "github.com/project-catalyst/pc-asset-hub/internal/service/meta"
)

type CatalogVersionHandler struct {
	svc *svcmeta.CatalogVersionService
}

func NewCatalogVersionHandler(svc *svcmeta.CatalogVersionService) *CatalogVersionHandler {
	return &CatalogVersionHandler{svc: svc}
}

func (h *CatalogVersionHandler) Create(c echo.Context) error {
	var req dto.CreateCatalogVersionRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	var pins []models.CatalogVersionPin
	for _, p := range req.Pins {
		pins = append(pins, models.CatalogVersionPin{EntityTypeVersionID: p.EntityTypeVersionID})
	}

	cv, err := h.svc.CreateCatalogVersion(c.Request().Context(), req.VersionLabel, req.Description, pins)
	if err != nil {
		return mapError(err)
	}

	return c.JSON(http.StatusCreated, dto.CatalogVersionResponse{
		ID: cv.ID, VersionLabel: cv.VersionLabel, Description: cv.Description, LifecycleStage: string(cv.LifecycleStage),
		CreatedAt: cv.CreatedAt, UpdatedAt: cv.UpdatedAt,
	})
}

func (h *CatalogVersionHandler) List(c echo.Context) error {
	params := models.ListParams{Limit: 20}
	if stage := c.QueryParam("stage"); stage != "" {
		params.Filters = map[string]string{"lifecycle_stage": stage}
	}
	items, total, err := h.svc.ListCatalogVersions(c.Request().Context(), params)
	if err != nil {
		return mapError(err)
	}
	resp := make([]dto.CatalogVersionResponse, len(items))
	for i, cv := range items {
		resp[i] = dto.CatalogVersionResponse{
			ID: cv.ID, VersionLabel: cv.VersionLabel, Description: cv.Description, LifecycleStage: string(cv.LifecycleStage),
			CreatedAt: cv.CreatedAt, UpdatedAt: cv.UpdatedAt,
		}
	}
	return c.JSON(http.StatusOK, dto.ListResponse{Items: resp, Total: total})
}

func (h *CatalogVersionHandler) GetByID(c echo.Context) error {
	id := c.Param("id")
	cv, err := h.svc.GetCatalogVersion(c.Request().Context(), id)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(http.StatusOK, dto.CatalogVersionResponse{
		ID: cv.ID, VersionLabel: cv.VersionLabel, Description: cv.Description, LifecycleStage: string(cv.LifecycleStage),
		CreatedAt: cv.CreatedAt, UpdatedAt: cv.UpdatedAt,
	})
}

func (h *CatalogVersionHandler) Promote(c echo.Context) error {
	id := c.Param("id")
	role := middleware.GetRoleFromContext(c)

	var svcRole svcmeta.Role
	switch role {
	case middleware.RoleRO:
		svcRole = svcmeta.RoleRO
	case middleware.RoleRW:
		svcRole = svcmeta.RoleRW
	case middleware.RoleAdmin:
		svcRole = svcmeta.RoleAdmin
	case middleware.RoleSuperAdmin:
		svcRole = svcmeta.RoleSuperAdmin
	}

	result, err := h.svc.Promote(c.Request().Context(), id, svcRole, string(role))
	if err != nil {
		return mapError(err)
	}
	warnings := make([]dto.CatalogWarningResponse, len(result.Warnings))
	for i, w := range result.Warnings {
		warnings[i] = dto.CatalogWarningResponse{
			CatalogName:      w.CatalogName,
			ValidationStatus: w.ValidationStatus,
		}
	}
	return c.JSON(http.StatusOK, dto.PromoteResponse{
		Status:   "promoted",
		Warnings: warnings,
	})
}

func (h *CatalogVersionHandler) Demote(c echo.Context) error {
	id := c.Param("id")
	role := middleware.GetRoleFromContext(c)

	var svcRole svcmeta.Role
	switch role {
	case middleware.RoleRO:
		svcRole = svcmeta.RoleRO
	case middleware.RoleRW:
		svcRole = svcmeta.RoleRW
	case middleware.RoleAdmin:
		svcRole = svcmeta.RoleAdmin
	case middleware.RoleSuperAdmin:
		svcRole = svcmeta.RoleSuperAdmin
	}

	var req struct {
		TargetStage string `json:"target_stage"`
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if err := h.svc.Demote(c.Request().Context(), id, svcRole, string(role), models.LifecycleStage(req.TargetStage)); err != nil {
		return mapError(err)
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "demoted"})
}

func (h *CatalogVersionHandler) Delete(c echo.Context) error {
	id := c.Param("id")
	role := middleware.GetRoleFromContext(c)

	var svcRole svcmeta.Role
	switch role {
	case middleware.RoleRO:
		svcRole = svcmeta.RoleRO
	case middleware.RoleRW:
		svcRole = svcmeta.RoleRW
	case middleware.RoleAdmin:
		svcRole = svcmeta.RoleAdmin
	case middleware.RoleSuperAdmin:
		svcRole = svcmeta.RoleSuperAdmin
	}

	if err := h.svc.DeleteCatalogVersion(c.Request().Context(), id, svcRole); err != nil {
		return mapError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *CatalogVersionHandler) ListPins(c echo.Context) error {
	id := c.Param("id")
	pins, err := h.svc.ListPins(c.Request().Context(), id)
	if err != nil {
		return mapError(err)
	}

	resp := make([]dto.CatalogVersionPinResponse, len(pins))
	for i, p := range pins {
		resp[i] = dto.CatalogVersionPinResponse{
			EntityTypeName:      p.EntityTypeName,
			EntityTypeID:        p.EntityTypeID,
			EntityTypeVersionID: p.EntityTypeVersionID,
			Version:             p.Version,
			Description:         p.Description,
		}
	}
	return c.JSON(http.StatusOK, dto.ListResponse{Items: resp, Total: len(resp)})
}

func (h *CatalogVersionHandler) ListTransitions(c echo.Context) error {
	id := c.Param("id")
	transitions, err := h.svc.ListTransitions(c.Request().Context(), id)
	if err != nil {
		return mapError(err)
	}

	resp := make([]dto.LifecycleTransitionResponse, len(transitions))
	for i, lt := range transitions {
		resp[i] = dto.LifecycleTransitionResponse{
			ID: lt.ID, FromStage: lt.FromStage, ToStage: lt.ToStage,
			PerformedBy: lt.PerformedBy, PerformedAt: lt.PerformedAt, Notes: lt.Notes,
		}
	}
	return c.JSON(http.StatusOK, dto.ListResponse{Items: resp, Total: len(resp)})
}

// RegisterCatalogVersionRoutes registers catalog version routes.
func RegisterCatalogVersionRoutes(g *echo.Group, h *CatalogVersionHandler, requireRW echo.MiddlewareFunc) {
	g.GET("/catalog-versions", h.List)
	g.GET("/catalog-versions/:id", h.GetByID)
	g.POST("/catalog-versions", h.Create, requireRW)
	g.POST("/catalog-versions/:id/promote", h.Promote, requireRW)
	g.POST("/catalog-versions/:id/demote", h.Demote, requireRW)
	g.DELETE("/catalog-versions/:id", h.Delete, requireRW)
	g.GET("/catalog-versions/:id/pins", h.ListPins)
	g.GET("/catalog-versions/:id/transitions", h.ListTransitions)
}
