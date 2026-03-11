package operational

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/project-catalyst/pc-asset-hub/internal/api/dto"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	svcop "github.com/project-catalyst/pc-asset-hub/internal/service/operational"
)

type InstanceHandler struct {
	svc *svcop.InstanceService
}

func NewInstanceHandler(svc *svcop.InstanceService) *InstanceHandler {
	return &InstanceHandler{svc: svc}
}

func (h *InstanceHandler) CreateInstance(c echo.Context) error {
	catalogName := c.Param("catalog-name")
	entityType := c.Param("entity-type")

	var req dto.CreateInstanceRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	detail, err := h.svc.CreateInstance(c.Request().Context(), catalogName, entityType, req.Name, req.Description, req.Attributes)
	if err != nil {
		return mapError(err)
	}

	return c.JSON(http.StatusCreated, instanceDetailToDTO(detail))
}

func (h *InstanceHandler) ListInstances(c echo.Context) error {
	catalogName := c.Param("catalog-name")
	entityType := c.Param("entity-type")

	params := models.ListParams{Limit: 20}

	details, total, err := h.svc.ListInstances(c.Request().Context(), catalogName, entityType, params)
	if err != nil {
		return mapError(err)
	}

	items := make([]dto.InstanceResponse, len(details))
	for i, d := range details {
		items[i] = instanceDetailToDTO(d)
	}

	return c.JSON(http.StatusOK, dto.ListResponse{Items: items, Total: total})
}

func (h *InstanceHandler) GetInstance(c echo.Context) error {
	catalogName := c.Param("catalog-name")
	entityType := c.Param("entity-type")
	instanceID := c.Param("instance-id")

	detail, err := h.svc.GetInstance(c.Request().Context(), catalogName, entityType, instanceID)
	if err != nil {
		return mapError(err)
	}

	return c.JSON(http.StatusOK, instanceDetailToDTO(detail))
}

func (h *InstanceHandler) UpdateInstance(c echo.Context) error {
	catalogName := c.Param("catalog-name")
	entityType := c.Param("entity-type")
	instanceID := c.Param("instance-id")

	var req dto.UpdateInstanceRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	detail, err := h.svc.UpdateInstance(c.Request().Context(), catalogName, entityType, instanceID, req.Version, req.Name, req.Description, req.Attributes)
	if err != nil {
		return mapError(err)
	}

	return c.JSON(http.StatusOK, instanceDetailToDTO(detail))
}

func (h *InstanceHandler) DeleteInstance(c echo.Context) error {
	catalogName := c.Param("catalog-name")
	entityType := c.Param("entity-type")
	instanceID := c.Param("instance-id")

	if err := h.svc.DeleteInstance(c.Request().Context(), catalogName, entityType, instanceID); err != nil {
		return mapError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

func instanceDetailToDTO(d *svcop.InstanceDetail) dto.InstanceResponse {
	attrs := make([]dto.AttributeValueResponse, len(d.Attributes))
	for i, av := range d.Attributes {
		attrs[i] = dto.AttributeValueResponse{
			Name:  av.Name,
			Type:  av.Type,
			Value: av.Value,
		}
	}
	return dto.InstanceResponse{
		ID:               d.Instance.ID,
		EntityTypeID:     d.Instance.EntityTypeID,
		CatalogID:        d.Instance.CatalogID,
		ParentInstanceID: d.Instance.ParentInstanceID,
		Name:             d.Instance.Name,
		Description:      d.Instance.Description,
		Version:          d.Instance.Version,
		Attributes:       attrs,
		CreatedAt:        d.Instance.CreatedAt,
		UpdatedAt:        d.Instance.UpdatedAt,
	}
}

func RegisterInstanceRoutes(g *echo.Group, h *InstanceHandler, requireRW echo.MiddlewareFunc) {
	g.POST("/:entity-type", h.CreateInstance, requireRW)
	g.GET("/:entity-type", h.ListInstances)
	g.GET("/:entity-type/:instance-id", h.GetInstance)
	g.PUT("/:entity-type/:instance-id", h.UpdateInstance, requireRW)
	g.DELETE("/:entity-type/:instance-id", h.DeleteInstance, requireRW)
}
