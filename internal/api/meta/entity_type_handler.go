package meta

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/project-catalyst/pc-asset-hub/internal/api/dto"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	svcmeta "github.com/project-catalyst/pc-asset-hub/internal/service/meta"
	"github.com/project-catalyst/pc-asset-hub/internal/service/validation"
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

func (h *EntityTypeHandler) Rename(c echo.Context) error {
	id := c.Param("id")
	var req dto.RenameEntityTypeRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if req.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "name is required")
	}

	result, err := h.svc.RenameEntityType(c.Request().Context(), id, req.Name, req.DeepCopyAllowed)
	if err != nil {
		return mapError(err)
	}

	return c.JSON(http.StatusOK, dto.RenameEntityTypeResponse{
		EntityType: dto.EntityTypeResponse{
			ID: result.EntityType.ID, Name: result.EntityType.Name,
			CreatedAt: result.EntityType.CreatedAt, UpdatedAt: result.EntityType.UpdatedAt,
		},
		WasDeepCopy: result.WasDeepCopy,
	})
}

func (h *EntityTypeHandler) ContainmentTree(c echo.Context) error {
	tree, err := h.svc.GetContainmentTree(c.Request().Context())
	if err != nil {
		return mapError(err)
	}
	return c.JSON(http.StatusOK, convertTreeNodes(tree))
}

func convertTreeNodes(nodes []*svcmeta.ContainmentTreeNode) []dto.ContainmentTreeNodeDTO {
	result := make([]dto.ContainmentTreeNodeDTO, len(nodes))
	for i, node := range nodes {
		versions := make([]dto.EntityTypeVersionResponse, len(node.Versions))
		for j, v := range node.Versions {
			versions[j] = dto.EntityTypeVersionResponse{
				ID: v.ID, EntityTypeID: v.EntityTypeID, Version: v.Version,
				Description: v.Description, CreatedAt: v.CreatedAt,
			}
		}
		result[i] = dto.ContainmentTreeNodeDTO{
			EntityType: dto.EntityTypeResponse{
				ID: node.EntityType.ID, Name: node.EntityType.Name,
				CreatedAt: node.EntityType.CreatedAt, UpdatedAt: node.EntityType.UpdatedAt,
			},
			Versions:      versions,
			LatestVersion: node.LatestVersion,
			Children:      convertTreeNodes(node.Children),
		}
	}
	return result
}

func (h *EntityTypeHandler) VersionSnapshot(c echo.Context) error {
	entityTypeID := c.Param("id")
	versionStr := c.Param("version")
	version, err := strconv.Atoi(versionStr)
	if err != nil || version <= 0 {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid version number")
	}

	snapshot, err := h.svc.GetVersionSnapshot(c.Request().Context(), entityTypeID, version)
	if err != nil {
		return mapError(err)
	}

	attrs := make([]dto.SnapshotAttributeResponse, len(snapshot.Attributes))
	for i, a := range snapshot.Attributes {
		attrs[i] = dto.SnapshotAttributeResponse{
			ID: a.ID, Name: a.Name, Description: a.Description,
			Type: string(a.Type), EnumID: a.EnumID, EnumName: snapshot.EnumNames[a.EnumID],
			Ordinal: a.Ordinal, Required: a.Required,
		}
	}

	assocs := make([]dto.SnapshotAssociationResponse, len(snapshot.Associations))
	for i, da := range snapshot.Associations {
		resp := dto.SnapshotAssociationResponse{
			ID: da.ID, Name: da.Name, Type: string(da.Type),
			TargetEntityTypeID:   da.TargetEntityTypeID,
			TargetEntityTypeName: snapshot.TargetEntityTypeNames[da.TargetEntityTypeID],
			SourceRole:           da.SourceRole,
			TargetRole:           da.TargetRole,
			SourceCardinality:    validation.NormalizeSourceCardinality(da.SourceCardinality, da.Type == models.AssociationTypeContainment),
			TargetCardinality:    validation.NormalizeCardinality(da.TargetCardinality),
			Direction:            da.Direction,
		}
		if da.Direction == "incoming" {
			resp.SourceEntityTypeID = da.SourceEntityTypeID
			resp.SourceEntityTypeName = snapshot.TargetEntityTypeNames[da.SourceEntityTypeID]
		}
		assocs[i] = resp
	}

	return c.JSON(http.StatusOK, dto.VersionSnapshotResponse{
		EntityType: dto.EntityTypeResponse{
			ID: snapshot.EntityType.ID, Name: snapshot.EntityType.Name,
			CreatedAt: snapshot.EntityType.CreatedAt, UpdatedAt: snapshot.EntityType.UpdatedAt,
		},
		Version: dto.EntityTypeVersionResponse{
			ID: snapshot.Version.ID, EntityTypeID: snapshot.Version.EntityTypeID,
			Version: snapshot.Version.Version, Description: snapshot.Version.Description,
			CreatedAt: snapshot.Version.CreatedAt,
		},
		Attributes:   attrs,
		Associations: assocs,
	})
}

// RegisterEntityTypeRoutes registers entity type routes on the given Echo group.
func RegisterEntityTypeRoutes(g *echo.Group, h *EntityTypeHandler, requireAdmin echo.MiddlewareFunc) {
	g.GET("/entity-types/containment-tree", h.ContainmentTree)
	g.GET("/entity-types/:id/versions/:version/snapshot", h.VersionSnapshot)
	g.GET("/entity-types", h.List)
	g.GET("/entity-types/:id", h.GetByID)
	g.POST("/entity-types", h.Create, requireAdmin)
	g.PUT("/entity-types/:id", h.Update, requireAdmin)
	g.DELETE("/entity-types/:id", h.Delete, requireAdmin)
	g.POST("/entity-types/:id/copy", h.Copy, requireAdmin)
	g.POST("/entity-types/:id/rename", h.Rename, requireAdmin)
}
