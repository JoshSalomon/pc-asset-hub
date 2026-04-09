package operational

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/project-catalyst/pc-asset-hub/internal/api/dto"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	svcop "github.com/project-catalyst/pc-asset-hub/internal/service/operational"
)

type InstanceHandler struct {
	svc        *svcop.InstanceService
	catalogSvc *svcop.CatalogService
}

func NewInstanceHandler(svc *svcop.InstanceService, catalogSvc *svcop.CatalogService) *InstanceHandler {
	return &InstanceHandler{svc: svc, catalogSvc: catalogSvc}
}

// syncCR updates the Catalog CR if the catalog is published (best-effort, fire-and-forget).
func (h *InstanceHandler) syncCR(c echo.Context) {
	if h.catalogSvc != nil {
		h.catalogSvc.SyncCR(c.Request().Context(), c.Param("catalog-name"))
	}
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

	h.syncCR(c)
	return c.JSON(http.StatusCreated, instanceDetailToDTO(detail))
}

// parseListParams extracts pagination, sort, and filter query params from the request.
func parseListParams(c echo.Context) models.ListParams {
	params := models.ListParams{Limit: 20}

	// Parse pagination
	if limitStr := c.QueryParam("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			if l > 100 {
				l = 100
			}
			params.Limit = l
		}
	}
	if offsetStr := c.QueryParam("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			params.Offset = o
		}
	}

	// Parse sort: ?sort=attr:asc or ?sort=attr:desc
	if sortStr := c.QueryParam("sort"); sortStr != "" {
		parts := strings.SplitN(sortStr, ":", 2)
		params.SortBy = parts[0]
		if len(parts) == 2 && parts[1] == "desc" {
			params.SortDesc = true
		}
	}

	// Parse filters: ?filter.attrName=value, ?filter.attrName.min=5, ?filter.attrName.max=10
	filters := make(map[string]string)
	for key, values := range c.QueryParams() {
		if strings.HasPrefix(key, "filter.") && len(values) > 0 {
			filterKey := strings.TrimPrefix(key, "filter.")
			filters[filterKey] = values[0]
		}
	}
	if len(filters) > 0 {
		params.Filters = filters
	}

	return params
}

func (h *InstanceHandler) ListInstances(c echo.Context) error {
	catalogName := c.Param("catalog-name")
	entityType := c.Param("entity-type")

	params := parseListParams(c)

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

	h.syncCR(c)
	return c.JSON(http.StatusOK, instanceDetailToDTO(detail))
}

func (h *InstanceHandler) DeleteInstance(c echo.Context) error {
	catalogName := c.Param("catalog-name")
	entityType := c.Param("entity-type")
	instanceID := c.Param("instance-id")

	if err := h.svc.DeleteInstance(c.Request().Context(), catalogName, entityType, instanceID); err != nil {
		return mapError(err)
	}

	h.syncCR(c)
	return c.NoContent(http.StatusNoContent)
}

func instanceDetailToDTO(d *svcop.InstanceDetail) dto.InstanceResponse {
	// Prepend system attributes (Name — required, Description — optional)
	systemAttrs := []dto.AttributeValueResponse{
		{Name: models.SystemAttrName, Type: models.SystemAttrType, Value: d.Instance.Name, System: true, Required: true},
		{Name: models.SystemAttrDescription, Type: models.SystemAttrType, Value: d.Instance.Description, System: true, Required: false},
	}
	attrs := make([]dto.AttributeValueResponse, 0, len(systemAttrs)+len(d.Attributes))
	attrs = append(attrs, systemAttrs...)
	for _, av := range d.Attributes {
		attrs = append(attrs, dto.AttributeValueResponse{
			Name:     av.Name,
			Type:     av.Type,
			Value:    av.Value,
			Required: av.Required,
		})
	}
	resp := dto.InstanceResponse{
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

	if len(d.ParentChain) > 0 {
		chain := make([]dto.ParentChainEntryResponse, len(d.ParentChain))
		for i, entry := range d.ParentChain {
			chain[i] = dto.ParentChainEntryResponse{
				InstanceID:     entry.InstanceID,
				InstanceName:   entry.InstanceName,
				EntityTypeName: entry.EntityTypeName,
			}
		}
		resp.ParentChain = chain
	}

	return resp
}

func (h *InstanceHandler) CreateContainedInstance(c echo.Context) error {
	catalogName := c.Param("catalog-name")
	parentType := c.Param("entity-type")
	parentID := c.Param("instance-id")
	childType := c.Param("child-type")

	var req dto.CreateInstanceRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	detail, err := h.svc.CreateContainedInstance(c.Request().Context(), catalogName, parentType, parentID, childType, req.Name, req.Description, req.Attributes)
	if err != nil {
		return mapError(err)
	}

	h.syncCR(c)
	return c.JSON(http.StatusCreated, instanceDetailToDTO(detail))
}

func (h *InstanceHandler) ListContainedInstances(c echo.Context) error {
	catalogName := c.Param("catalog-name")
	parentType := c.Param("entity-type")
	parentID := c.Param("instance-id")
	childType := c.Param("child-type")

	params := parseListParams(c)

	details, total, err := h.svc.ListContainedInstances(c.Request().Context(), catalogName, parentType, parentID, childType, params)
	if err != nil {
		return mapError(err)
	}

	items := make([]dto.InstanceResponse, len(details))
	for i, d := range details {
		items[i] = instanceDetailToDTO(d)
	}

	return c.JSON(http.StatusOK, dto.ListResponse{Items: items, Total: total})
}

func (h *InstanceHandler) CreateLink(c echo.Context) error {
	catalogName := c.Param("catalog-name")
	entityType := c.Param("entity-type")
	instanceID := c.Param("instance-id")

	var req dto.CreateAssociationLinkRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	link, err := h.svc.CreateAssociationLink(c.Request().Context(), catalogName, entityType, instanceID, req.TargetInstanceID, req.AssociationName)
	if err != nil {
		return mapError(err)
	}

	h.syncCR(c)
	return c.JSON(http.StatusCreated, dto.AssociationLinkResponse{
		ID:               link.ID,
		AssociationID:    link.AssociationID,
		SourceInstanceID: link.SourceInstanceID,
		TargetInstanceID: link.TargetInstanceID,
		CreatedAt:        link.CreatedAt,
	})
}

func (h *InstanceHandler) DeleteLink(c echo.Context) error {
	catalogName := c.Param("catalog-name")
	entityType := c.Param("entity-type")
	linkID := c.Param("link-id")

	if err := h.svc.DeleteAssociationLink(c.Request().Context(), catalogName, entityType, linkID); err != nil {
		return mapError(err)
	}

	h.syncCR(c)
	return c.NoContent(http.StatusNoContent)
}

func (h *InstanceHandler) GetForwardReferences(c echo.Context) error {
	catalogName := c.Param("catalog-name")
	entityType := c.Param("entity-type")
	instanceID := c.Param("instance-id")

	refs, err := h.svc.GetForwardReferences(c.Request().Context(), catalogName, entityType, instanceID)
	if err != nil {
		return mapError(err)
	}

	result := make([]dto.ReferenceResponse, len(refs))
	for i, r := range refs {
		result[i] = dto.ReferenceResponse{
			LinkID:          r.LinkID,
			AssociationName: r.AssociationName,
			AssociationType: r.AssociationType,
			InstanceID:      r.InstanceID,
			InstanceName:    r.InstanceName,
			EntityTypeName:  r.EntityTypeName,
		}
	}

	return c.JSON(http.StatusOK, result)
}

func (h *InstanceHandler) GetReverseReferences(c echo.Context) error {
	catalogName := c.Param("catalog-name")
	entityType := c.Param("entity-type")
	instanceID := c.Param("instance-id")

	refs, err := h.svc.GetReverseReferences(c.Request().Context(), catalogName, entityType, instanceID)
	if err != nil {
		return mapError(err)
	}

	result := make([]dto.ReferenceResponse, len(refs))
	for i, r := range refs {
		result[i] = dto.ReferenceResponse{
			LinkID:          r.LinkID,
			AssociationName: r.AssociationName,
			AssociationType: r.AssociationType,
			InstanceID:      r.InstanceID,
			InstanceName:    r.InstanceName,
			EntityTypeName:  r.EntityTypeName,
		}
	}

	return c.JSON(http.StatusOK, result)
}

func (h *InstanceHandler) SetParent(c echo.Context) error {
	catalogName := c.Param("catalog-name")
	entityType := c.Param("entity-type")
	instanceID := c.Param("instance-id")

	var req dto.SetParentRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.ParentType == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "parent_type is required")
	}

	if err := h.svc.SetParent(c.Request().Context(), catalogName, entityType, instanceID, req.ParentType, req.ParentInstanceID); err != nil {
		return mapError(err)
	}

	h.syncCR(c)
	return c.JSON(http.StatusOK, map[string]string{"status": "updated"})
}

func (h *InstanceHandler) GetContainmentTree(c echo.Context) error {
	catalogName := c.Param("catalog-name")

	tree, err := h.svc.GetContainmentTree(c.Request().Context(), catalogName)
	if err != nil {
		return mapError(err)
	}

	return c.JSON(http.StatusOK, treeNodesToDTO(tree))
}

func treeNodesToDTO(nodes []svcop.TreeNode) []dto.TreeNodeResponse {
	if len(nodes) == 0 {
		return []dto.TreeNodeResponse{}
	}
	result := make([]dto.TreeNodeResponse, len(nodes))
	for i, node := range nodes {
		result[i] = dto.TreeNodeResponse{
			InstanceID:     node.Instance.ID,
			InstanceName:   node.Instance.Name,
			EntityTypeName: node.EntityTypeName,
			Description:    node.Instance.Description,
			Children:       treeNodesToDTO(node.Children),
		}
	}
	return result
}

func RegisterInstanceRoutes(g *echo.Group, h *InstanceHandler, requireRW echo.MiddlewareFunc, writeGuards ...echo.MiddlewareFunc) {
	// writeMiddleware combines requireRW with any additional write guards (e.g., published catalog protection)
	writeMiddleware := append([]echo.MiddlewareFunc{requireRW}, writeGuards...)

	// Containment tree — static path before parameterized :entity-type
	g.GET("/tree", h.GetContainmentTree)

	// Instance CRUD
	g.POST("/:entity-type", h.CreateInstance, writeMiddleware...)
	g.GET("/:entity-type", h.ListInstances)
	g.GET("/:entity-type/:instance-id", h.GetInstance)
	g.PUT("/:entity-type/:instance-id", h.UpdateInstance, writeMiddleware...)
	g.DELETE("/:entity-type/:instance-id", h.DeleteInstance, writeMiddleware...)

	// Parent management
	g.PUT("/:entity-type/:instance-id/parent", h.SetParent, writeMiddleware...)

	// Association links and references — static path segments registered before
	// the parameterized containment routes so Echo matches them first.
	// Entity types named "links", "references", or "referenced-by" are reserved.
	g.POST("/:entity-type/:instance-id/links", h.CreateLink, writeMiddleware...)
	g.DELETE("/:entity-type/:instance-id/links/:link-id", h.DeleteLink, writeMiddleware...)
	g.GET("/:entity-type/:instance-id/references", h.GetForwardReferences)
	g.GET("/:entity-type/:instance-id/referenced-by", h.GetReverseReferences)

	// Containment — parameterized :child-type after static segments above
	g.POST("/:entity-type/:instance-id/:child-type", h.CreateContainedInstance, writeMiddleware...)
	g.GET("/:entity-type/:instance-id/:child-type", h.ListContainedInstances)
}
