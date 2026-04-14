package meta

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/project-catalyst/pc-asset-hub/internal/api/dto"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	svcmeta "github.com/project-catalyst/pc-asset-hub/internal/service/meta"
)

type TypeDefinitionHandler struct {
	svc *svcmeta.TypeDefinitionService
}

func NewTypeDefinitionHandler(svc *svcmeta.TypeDefinitionService) *TypeDefinitionHandler {
	return &TypeDefinitionHandler{svc: svc}
}

func (h *TypeDefinitionHandler) Create(c echo.Context) error {
	var req dto.CreateTypeDefinitionRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}
	if req.BaseType == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "base_type is required")
	}

	td, tdv, err := h.svc.CreateTypeDefinition(c.Request().Context(), req.Name, req.Description, models.BaseType(req.BaseType), req.Constraints)
	if err != nil {
		return mapError(err)
	}

	return c.JSON(http.StatusCreated, dto.TypeDefinitionResponse{
		ID:              td.ID,
		Name:            td.Name,
		Description:     td.Description,
		BaseType:        string(td.BaseType),
		System:          td.System,
		LatestVersion:   tdv.VersionNumber,
		LatestVersionID: tdv.ID,
		CreatedAt:       td.CreatedAt,
		UpdatedAt:       td.UpdatedAt,
	})
}

func (h *TypeDefinitionHandler) List(c echo.Context) error {
	params := models.ListParams{Filters: make(map[string]string)}
	if bt := c.QueryParam("base_type"); bt != "" {
		params.Filters["base_type"] = bt
	}
	if name := c.QueryParam("name"); name != "" {
		params.Filters["name"] = name
	}

	items, total, err := h.svc.ListTypeDefinitions(c.Request().Context(), params)
	if err != nil {
		return mapError(err)
	}

	// Batch-fetch latest version info to avoid N+1 queries
	typeDefIDs := make([]string, len(items))
	for i, td := range items {
		typeDefIDs[i] = td.ID
	}
	versionNumbers, versionIDs, _ := h.svc.GetLatestVersionInfo(c.Request().Context(), typeDefIDs)

	resp := make([]dto.TypeDefinitionResponse, len(items))
	for i, td := range items {
		resp[i] = dto.TypeDefinitionResponse{
			ID:              td.ID,
			Name:            td.Name,
			Description:     td.Description,
			BaseType:        string(td.BaseType),
			System:          td.System,
			LatestVersion:   versionNumbers[td.ID],
			LatestVersionID: versionIDs[td.ID],
			CreatedAt:       td.CreatedAt,
			UpdatedAt:       td.UpdatedAt,
		}
	}

	return c.JSON(http.StatusOK, dto.ListResponse{Items: resp, Total: total})
}

func (h *TypeDefinitionHandler) GetByID(c echo.Context) error {
	id := c.Param("id")

	td, tdv, err := h.svc.GetTypeDefinition(c.Request().Context(), id)
	if err != nil {
		return mapError(err)
	}

	return c.JSON(http.StatusOK, dto.TypeDefinitionResponse{
		ID:              td.ID,
		Name:            td.Name,
		Description:     td.Description,
		BaseType:        string(td.BaseType),
		System:          td.System,
		LatestVersion:   tdv.VersionNumber,
		LatestVersionID: tdv.ID,
		CreatedAt:       td.CreatedAt,
		UpdatedAt:       td.UpdatedAt,
	})
}

func (h *TypeDefinitionHandler) Update(c echo.Context) error {
	id := c.Param("id")

	var req dto.UpdateTypeDefinitionRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	newTDV, err := h.svc.UpdateTypeDefinition(c.Request().Context(), id, req.Description, req.Constraints)
	if err != nil {
		return mapError(err)
	}

	return c.JSON(http.StatusOK, dto.TypeDefinitionVersionResponse{
		ID:               newTDV.ID,
		TypeDefinitionID: newTDV.TypeDefinitionID,
		VersionNumber:    newTDV.VersionNumber,
		Constraints:      newTDV.Constraints,
		CreatedAt:        newTDV.CreatedAt,
	})
}

func (h *TypeDefinitionHandler) Delete(c echo.Context) error {
	id := c.Param("id")

	if err := h.svc.DeleteTypeDefinition(c.Request().Context(), id); err != nil {
		return mapError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

func (h *TypeDefinitionHandler) ListVersions(c echo.Context) error {
	id := c.Param("id")

	versions, err := h.svc.ListVersions(c.Request().Context(), id)
	if err != nil {
		return mapError(err)
	}

	resp := make([]dto.TypeDefinitionVersionResponse, len(versions))
	for i, v := range versions {
		resp[i] = dto.TypeDefinitionVersionResponse{
			ID:               v.ID,
			TypeDefinitionID: v.TypeDefinitionID,
			VersionNumber:    v.VersionNumber,
			Constraints:      v.Constraints,
			CreatedAt:        v.CreatedAt,
		}
	}

	return c.JSON(http.StatusOK, dto.ListResponse{Items: resp, Total: len(resp)})
}

func (h *TypeDefinitionHandler) GetVersion(c echo.Context) error {
	id := c.Param("id")
	var versionNum int
	if err := echo.PathParamsBinder(c).Int("v", &versionNum).BindError(); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid version number")
	}

	version, err := h.svc.GetVersion(c.Request().Context(), id, versionNum)
	if err != nil {
		return mapError(err)
	}

	return c.JSON(http.StatusOK, dto.TypeDefinitionVersionResponse{
		ID:               version.ID,
		TypeDefinitionID: version.TypeDefinitionID,
		VersionNumber:    version.VersionNumber,
		Constraints:      version.Constraints,
		CreatedAt:        version.CreatedAt,
	})
}

func RegisterTypeDefinitionRoutes(g *echo.Group, h *TypeDefinitionHandler, requireAdmin echo.MiddlewareFunc) {
	g.POST("/type-definitions", h.Create, requireAdmin)
	g.GET("/type-definitions", h.List)
	g.GET("/type-definitions/:id", h.GetByID)
	g.PUT("/type-definitions/:id", h.Update, requireAdmin)
	g.DELETE("/type-definitions/:id", h.Delete, requireAdmin)
	g.GET("/type-definitions/:id/versions", h.ListVersions)
	g.GET("/type-definitions/:id/versions/:v", h.GetVersion)
}
