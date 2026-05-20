package operational_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	apimw "github.com/project-catalyst/pc-asset-hub/internal/api/middleware"
	apiop "github.com/project-catalyst/pc-asset-hub/internal/api/operational"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository/mocks"
	"github.com/project-catalyst/pc-asset-hub/internal/service/operational/export"
)

func setupExportBindingServer() (*echo.Echo, *export.ExporterRegistry, *mocks.MockExportBindingRepo, *mocks.MockCatalogRepo, *mocks.MockCatalogVersionPinRepo, *mocks.MockEntityTypeVersionRepo, *mocks.MockEntityTypeRepo, *mocks.MockAttributeRepo, *mocks.MockAssociationRepo) {
	registry := export.NewExporterRegistry()
	bindingRepo := new(mocks.MockExportBindingRepo)
	catalogRepo := new(mocks.MockCatalogRepo)
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	etRepo := new(mocks.MockEntityTypeRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)

	bindingSvc := export.NewExportBindingService(
		bindingRepo, catalogRepo, registry,
		cvRepo, pinRepo, etvRepo, etRepo,
		attrRepo, assocRepo,
	)
	accessChecker := &apimw.HeaderCatalogAccessChecker{}
	handler := apiop.NewExportBindingHandler(bindingSvc, registry, accessChecker)

	e := echo.New()
	g := e.Group("/api/data/v1/catalogs")
	rbac := &apimw.HeaderRBACProvider{}
	g.Use(apimw.RBACMiddleware(rbac))
	requireRW := apimw.RequireRole(apimw.RoleRW)
	requireAdmin := apimw.RequireRole(apimw.RoleAdmin)

	// Register exporters endpoint at the top level
	exportersGroup := e.Group("/api/data/v1")
	exportersGroup.Use(apimw.RBACMiddleware(rbac))
	exportersGroup.GET("/exporters", handler.ListExporters)

	apiop.RegisterExportBindingRoutes(g, handler, requireRW, requireAdmin)

	return e, registry, bindingRepo, catalogRepo, pinRepo, etvRepo, etRepo, attrRepo, assocRepo
}

// T-34.05: GET /exporters returns registered exporters as JSON
func TestListExporters_ReturnsRegistered(t *testing.T) {
	e, registry, _, _, _, _, _, _, _ := setupExportBindingServer()
	registry.Register(&testExporter{
		name: "mcp-gateway",
		desc: "MCP Gateway Exporter",
		params: []export.ParameterDef{
			{Name: "server_type", Type: "string", Required: true, Description: "Server entity type"},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/data/v1/exporters", nil)
	req.Header.Set("X-User-Role", "RO")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string][]map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body["items"], 1)
	assert.Equal(t, "mcp-gateway", body["items"][0]["name"])
}

// T-34.06: GET /exporters with no auth still succeeds (any authenticated user)
func TestListExporters_AnyRole(t *testing.T) {
	e, _, _, _, _, _, _, _, _ := setupExportBindingServer()

	req := httptest.NewRequest(http.MethodGet, "/api/data/v1/exporters", nil)
	req.Header.Set("X-User-Role", "RO")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// T-34.20: POST /export-bindings with valid data returns 201
func TestCreateBinding_API_Success(t *testing.T) {
	e, registry, bindingRepo, catalogRepo, pinRepo, etvRepo, etRepo, attrRepo, assocRepo := setupExportBindingServer()
	registry.Register(&testExporter{
		name: "mcp-gateway",
		params: []export.ParameterDef{
			{Name: "server_type", Type: "string", Required: true},
			{Name: "tool_type", Type: "string", Required: true},
		},
	})

	catalogRepo.On("GetByName", mock.Anything, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return([]*models.CatalogVersionPin{
		{EntityTypeVersionID: "etv1"}, {EntityTypeVersionID: "etv2"},
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1"}, nil)
	etvRepo.On("GetByID", mock.Anything, "etv2").Return(&models.EntityTypeVersion{ID: "etv2", EntityTypeID: "et2"}, nil)
	etRepo.On("GetByID", mock.Anything, "et1").Return(&models.EntityType{ID: "et1", Name: "mcp-server"}, nil)
	etRepo.On("GetByID", mock.Anything, "et2").Return(&models.EntityType{ID: "et2", Name: "mcp-tool"}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Attribute{{Name: "route_name"}}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv2").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Association{
		{Type: "containment", TargetEntityTypeID: "et2"},
	}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "etv2").Return([]*models.Association{}, nil)
	bindingRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.ExportBinding")).Return(nil)

	body, _ := json.Marshal(map[string]any{
		"exporter_name": "mcp-gateway",
		"parameters":    map[string]string{"server_type": "mcp-server", "tool_type": "mcp-tool"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/data/v1/catalogs/my-catalog/export-bindings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Role", "Admin")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
}

// T-34.21: POST /export-bindings with invalid exporter returns 400
func TestCreateBinding_API_InvalidExporter(t *testing.T) {
	e, _, _, catalogRepo, _, _, _, _, _ := setupExportBindingServer()
	catalogRepo.On("GetByName", mock.Anything, "my-catalog").Return(&models.Catalog{
		ID: "cat1", CatalogVersionID: "cv1",
	}, nil)

	body, _ := json.Marshal(map[string]any{"exporter_name": "no-such"})
	req := httptest.NewRequest(http.MethodPost, "/api/data/v1/catalogs/my-catalog/export-bindings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Role", "Admin")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// T-34.22: POST /export-bindings with missing required params returns 400
func TestCreateBinding_API_MissingParams(t *testing.T) {
	e, registry, _, catalogRepo, _, _, _, _, _ := setupExportBindingServer()
	registry.Register(&testExporter{
		name: "mcp-gateway",
		params: []export.ParameterDef{
			{Name: "server_type", Type: "string", Required: true},
		},
	})
	catalogRepo.On("GetByName", mock.Anything, "my-catalog").Return(&models.Catalog{
		ID: "cat1", CatalogVersionID: "cv1",
	}, nil)

	body, _ := json.Marshal(map[string]any{"exporter_name": "mcp-gateway", "parameters": map[string]string{}})
	req := httptest.NewRequest(http.MethodPost, "/api/data/v1/catalogs/my-catalog/export-bindings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Role", "Admin")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "server_type")
}

// T-34.23: POST /export-bindings with entity type not in CV returns 400
func TestCreateBinding_API_EntityTypeNotPinned(t *testing.T) {
	e, registry, _, catalogRepo, pinRepo, etvRepo, etRepo, attrRepo, assocRepo := setupExportBindingServer()
	registry.Register(&testExporter{
		name: "mcp-gateway",
		params: []export.ParameterDef{
			{Name: "server_type", Type: "string", Required: true},
			{Name: "tool_type", Type: "string", Required: true},
		},
	})
	catalogRepo.On("GetByName", mock.Anything, "my-catalog").Return(&models.Catalog{
		ID: "cat1", CatalogVersionID: "cv1",
	}, nil)
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return([]*models.CatalogVersionPin{
		{EntityTypeVersionID: "etv1"},
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1"}, nil)
	etRepo.On("GetByID", mock.Anything, "et1").Return(&models.EntityType{ID: "et1", Name: "other-type"}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Association{}, nil)

	body, _ := json.Marshal(map[string]any{
		"exporter_name": "mcp-gateway",
		"parameters":    map[string]string{"server_type": "mcp-server", "tool_type": "mcp-tool"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/data/v1/catalogs/my-catalog/export-bindings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Role", "Admin")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	body2 := rec.Body.String()
	assert.True(t, strings.Contains(body2, "mcp-server") || strings.Contains(body2, "mcp-tool"),
		"error should mention one of the unpinned entity types, got: %s", body2)
}

// T-34.24: GET /export-bindings returns all bindings for catalog
func TestListBindings_API(t *testing.T) {
	e, _, bindingRepo, catalogRepo, _, _, _, _, _ := setupExportBindingServer()
	catalogRepo.On("GetByName", mock.Anything, "my-catalog").Return(&models.Catalog{ID: "cat1", CatalogVersionID: "cv1"}, nil)
	bindingRepo.On("ListByCatalog", mock.Anything, "cat1").Return([]*models.ExportBinding{
		{ID: "b1", CatalogID: "cat1", ExporterName: "mcp-gateway", Parameters: map[string]string{}, LastRunStatus: "never"},
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/data/v1/catalogs/my-catalog/export-bindings", nil)
	req.Header.Set("X-User-Role", "RO")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string][]map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Len(t, body["items"], 1)
}

// T-34.25: PUT /export-bindings/{id} updates enabled
func TestUpdateBinding_API(t *testing.T) {
	e, _, bindingRepo, catalogRepo, _, _, _, _, _ := setupExportBindingServer()
	catalogRepo.On("GetByName", mock.Anything, "my-catalog").Return(&models.Catalog{ID: "cat1", CatalogVersionID: "cv1"}, nil)
	bindingRepo.On("GetByID", mock.Anything, "b1").Return(&models.ExportBinding{
		ID: "b1", CatalogID: "cat1", ExporterName: "mcp-gateway",
		Parameters: map[string]string{"server_type": "mcp-server"}, Enabled: true, LastRunStatus: "never",
	}, nil)
	bindingRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.ExportBinding")).Return(nil)

	disabled := false
	body, _ := json.Marshal(map[string]any{
		"enabled": disabled,
	})
	req := httptest.NewRequest(http.MethodPut, "/api/data/v1/catalogs/my-catalog/export-bindings/b1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Role", "Admin")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// T-34.26: DELETE /export-bindings/{id} removes binding
func TestDeleteBinding_API(t *testing.T) {
	e, _, bindingRepo, catalogRepo, _, _, _, _, _ := setupExportBindingServer()
	catalogRepo.On("GetByName", mock.Anything, "my-catalog").Return(&models.Catalog{ID: "cat1", CatalogVersionID: "cv1"}, nil)
	bindingRepo.On("GetByID", mock.Anything, "b1").Return(&models.ExportBinding{
		ID: "b1", CatalogID: "cat1",
	}, nil)
	bindingRepo.On("Delete", mock.Anything, "b1").Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/api/data/v1/catalogs/my-catalog/export-bindings/b1", nil)
	req.Header.Set("X-User-Role", "Admin")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

// T-34.27: RBAC: RO can list bindings
func TestListBindings_API_RO(t *testing.T) {
	e, _, bindingRepo, catalogRepo, _, _, _, _, _ := setupExportBindingServer()
	catalogRepo.On("GetByName", mock.Anything, "my-catalog").Return(&models.Catalog{ID: "cat1", CatalogVersionID: "cv1"}, nil)
	bindingRepo.On("ListByCatalog", mock.Anything, "cat1").Return([]*models.ExportBinding{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/data/v1/catalogs/my-catalog/export-bindings", nil)
	req.Header.Set("X-User-Role", "RO")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// T-34.28: RBAC: RW cannot create binding (Admin+ required)
func TestCreateBinding_API_RW_Forbidden(t *testing.T) {
	e, _, _, _, _, _, _, _, _ := setupExportBindingServer()

	body, _ := json.Marshal(map[string]any{"exporter_name": "mcp-gateway"})
	req := httptest.NewRequest(http.MethodPost, "/api/data/v1/catalogs/my-catalog/export-bindings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Role", "RW")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// T-34.29: RBAC: Admin can create/update/delete bindings
func TestBinding_API_Admin_FullCRUD(t *testing.T) {
	e, registry, bindingRepo, catalogRepo, pinRepo, etvRepo, etRepo, attrRepo, assocRepo := setupExportBindingServer()
	registry.Register(&testExporter{
		name: "mcp-gateway",
		params: []export.ParameterDef{
			{Name: "server_type", Type: "string", Required: true},
			{Name: "tool_type", Type: "string", Required: true},
		},
	})

	catalogRepo.On("GetByName", mock.Anything, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return([]*models.CatalogVersionPin{
		{EntityTypeVersionID: "etv1"}, {EntityTypeVersionID: "etv2"},
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1"}, nil)
	etvRepo.On("GetByID", mock.Anything, "etv2").Return(&models.EntityTypeVersion{ID: "etv2", EntityTypeID: "et2"}, nil)
	etRepo.On("GetByID", mock.Anything, "et1").Return(&models.EntityType{ID: "et1", Name: "mcp-server"}, nil)
	etRepo.On("GetByID", mock.Anything, "et2").Return(&models.EntityType{ID: "et2", Name: "mcp-tool"}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Attribute{{Name: "route_name"}}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv2").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Association{
		{Type: "containment", TargetEntityTypeID: "et2"},
	}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "etv2").Return([]*models.Association{}, nil)
	bindingRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.ExportBinding")).Return(nil)
	bindingRepo.On("GetByID", mock.Anything, mock.Anything).Return(&models.ExportBinding{
		ID: "b1", CatalogID: "cat1", ExporterName: "mcp-gateway",
		Parameters: map[string]string{}, Enabled: true, LastRunStatus: "never",
	}, nil)
	bindingRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.ExportBinding")).Return(nil)
	bindingRepo.On("Delete", mock.Anything, mock.Anything).Return(nil)

	// Create
	body, _ := json.Marshal(map[string]any{
		"exporter_name": "mcp-gateway",
		"parameters":    map[string]string{"server_type": "mcp-server", "tool_type": "mcp-tool"},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/data/v1/catalogs/my-catalog/export-bindings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Role", "Admin")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusCreated, rec.Code)

	// Update (toggle enabled, no param validation needed)
	body, _ = json.Marshal(map[string]any{"enabled": false})
	req = httptest.NewRequest(http.MethodPut, "/api/data/v1/catalogs/my-catalog/export-bindings/b1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Role", "Admin")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Delete
	req = httptest.NewRequest(http.MethodDelete, "/api/data/v1/catalogs/my-catalog/export-bindings/b1", nil)
	req.Header.Set("X-User-Role", "Admin")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

// setupExportBindingServerWithCache returns an additional preview cache for
// publish-preview and download tests.
func setupExportBindingServerWithCache() (*echo.Echo, *export.ExporterRegistry, *mocks.MockExportBindingRepo, *mocks.MockCatalogRepo, *mocks.MockCatalogVersionPinRepo, *mocks.MockEntityTypeVersionRepo, *mocks.MockEntityTypeRepo, *mocks.MockAttributeRepo, *mocks.MockAssociationRepo, *export.InMemoryPreviewCache) {
	registry := export.NewExporterRegistry()
	bindingRepo := new(mocks.MockExportBindingRepo)
	catalogRepo := new(mocks.MockCatalogRepo)
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	etRepo := new(mocks.MockEntityTypeRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	cache := export.NewInMemoryPreviewCache()

	// Allow Update calls from executeBinding (status updates)
	bindingRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.ExportBinding")).Maybe().Return(nil)

	bindingSvc := export.NewExportBindingService(
		bindingRepo, catalogRepo, registry,
		cvRepo, pinRepo, etvRepo, etRepo,
		attrRepo, assocRepo,
		export.WithPreviewCache(cache),
	)
	accessChecker := &apimw.HeaderCatalogAccessChecker{}
	handler := apiop.NewExportBindingHandler(bindingSvc, registry, accessChecker)

	e := echo.New()
	g := e.Group("/api/data/v1/catalogs")
	rbac := &apimw.HeaderRBACProvider{}
	g.Use(apimw.RBACMiddleware(rbac))
	requireRW := apimw.RequireRole(apimw.RoleRW)
	requireAdmin := apimw.RequireRole(apimw.RoleAdmin)

	exportersGroup := e.Group("/api/data/v1")
	exportersGroup.Use(apimw.RBACMiddleware(rbac))
	exportersGroup.GET("/exporters", handler.ListExporters)

	apiop.RegisterExportBindingRoutes(g, handler, requireRW, requireAdmin)

	return e, registry, bindingRepo, catalogRepo, pinRepo, etvRepo, etRepo, attrRepo, assocRepo, cache
}

// T-34.44: POST /export-bindings/{id}/run returns YAML with Content-Disposition header
func TestT34_44_RunBinding_ReturnsYAML(t *testing.T) {
	e, registry, bindingRepo, catalogRepo, _, _, _, _, _ := setupExportBindingServer()
	registry.Register(&testExporter{
		name: "mcp-gateway",
		exportOut: &export.ExportOutput{
			Artifacts: []export.K8sArtifact{
				{APIVersion: "v1", Kind: "ConfigMap", Name: "cm1", YAML: "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: cm1\n"},
			},
		},
	})

	catalogRepo.On("GetByName", mock.Anything, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	bindingRepo.On("GetByID", mock.Anything, "b1").Return(&models.ExportBinding{
		ID: "b1", CatalogID: "cat1", ExporterName: "mcp-gateway",
		Parameters: map[string]string{}, Enabled: true, LastRunStatus: "never",
	}, nil)
	bindingRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.ExportBinding")).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/data/v1/catalogs/my-catalog/export-bindings/b1/run", nil)
	req.Header.Set("X-User-Role", "RW")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Disposition"), "my-catalog-export.yaml")
	assert.Contains(t, rec.Body.String(), "apiVersion: v1")
	assert.Contains(t, rec.Body.String(), "---")
}

// T-34.45: POST /run as RO returns 403
func TestT34_45_RunBinding_RO_Forbidden(t *testing.T) {
	e, _, _, _, _, _, _, _, _ := setupExportBindingServer()

	req := httptest.NewRequest(http.MethodPost, "/api/data/v1/catalogs/my-catalog/export-bindings/b1/run", nil)
	req.Header.Set("X-User-Role", "RO")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// T-34.46: POST /run as RW returns 200
func TestT34_46_RunBinding_RW_Allowed(t *testing.T) {
	e, registry, bindingRepo, catalogRepo, _, _, _, _, _ := setupExportBindingServer()
	registry.Register(&testExporter{
		name: "mcp-gateway",
		exportOut: &export.ExportOutput{
			Artifacts: []export.K8sArtifact{
				{APIVersion: "v1", Kind: "ConfigMap", Name: "cm1", YAML: "apiVersion: v1\nkind: ConfigMap\n"},
			},
		},
	})

	catalogRepo.On("GetByName", mock.Anything, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	bindingRepo.On("GetByID", mock.Anything, "b1").Return(&models.ExportBinding{
		ID: "b1", CatalogID: "cat1", ExporterName: "mcp-gateway",
		Parameters: map[string]string{}, Enabled: true, LastRunStatus: "never",
	}, nil)
	bindingRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.ExportBinding")).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/data/v1/catalogs/my-catalog/export-bindings/b1/run", nil)
	req.Header.Set("X-User-Role", "RW")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// T-34.47: POST /run on draft catalog succeeds (run works on any state)
func TestT34_47_RunBinding_DraftCatalog(t *testing.T) {
	e, registry, bindingRepo, catalogRepo, _, _, _, _, _ := setupExportBindingServer()
	registry.Register(&testExporter{
		name: "mcp-gateway",
		exportOut: &export.ExportOutput{
			Artifacts: []export.K8sArtifact{
				{APIVersion: "v1", Kind: "ConfigMap", Name: "cm1", YAML: "apiVersion: v1\nkind: ConfigMap\n"},
			},
		},
	})

	catalogRepo.On("GetByName", mock.Anything, "draft-cat").Return(&models.Catalog{
		ID: "cat1", Name: "draft-cat", CatalogVersionID: "cv1",
		ValidationStatus: models.ValidationStatusDraft,
	}, nil)
	bindingRepo.On("GetByID", mock.Anything, "b1").Return(&models.ExportBinding{
		ID: "b1", CatalogID: "cat1", ExporterName: "mcp-gateway",
		Parameters: map[string]string{}, Enabled: true, LastRunStatus: "never",
	}, nil)
	bindingRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.ExportBinding")).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/data/v1/catalogs/draft-cat/export-bindings/b1/run", nil)
	req.Header.Set("X-User-Role", "RW")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

// T-34.47b: POST /run on empty catalog returns empty content
func TestT34_47b_RunBinding_EmptyCatalog(t *testing.T) {
	e, registry, bindingRepo, catalogRepo, _, _, _, _, _ := setupExportBindingServer()
	registry.Register(&testExporter{
		name: "mcp-gateway",
		exportOut: &export.ExportOutput{
			Artifacts: []export.K8sArtifact{},
			Warnings:  []string{"no instances found"},
		},
	})

	catalogRepo.On("GetByName", mock.Anything, "empty-cat").Return(&models.Catalog{
		ID: "cat1", Name: "empty-cat", CatalogVersionID: "cv1",
	}, nil)
	bindingRepo.On("GetByID", mock.Anything, "b1").Return(&models.ExportBinding{
		ID: "b1", CatalogID: "cat1", ExporterName: "mcp-gateway",
		Parameters: map[string]string{}, Enabled: true, LastRunStatus: "never",
	}, nil)
	bindingRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.ExportBinding")).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/api/data/v1/catalogs/empty-cat/export-bindings/b1/run", nil)
	req.Header.Set("X-User-Role", "RW")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	// Empty artifacts → renderMultiDocYAML returns ""
	assert.Equal(t, "", rec.Body.String())
}

// T-34.86: POST /publish/preview returns session token and binding results
func TestT34_86_PublishPreview_ReturnsSessionToken(t *testing.T) {
	e, registry, bindingRepo, catalogRepo, pinRepo, _, _, _, _, _ := setupExportBindingServerWithCache()
	registry.Register(&testExporter{
		name: "mcp-gateway",
		exportOut: &export.ExportOutput{
			Artifacts: []export.K8sArtifact{
				{APIVersion: "v1", Kind: "ConfigMap", Name: "cm1", YAML: "apiVersion: v1\nkind: ConfigMap\n"},
			},
		},
	})

	catalogRepo.On("GetByName", mock.Anything, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	bindingRepo.On("ListByCatalog", mock.Anything, "cat1").Return([]*models.ExportBinding{
		{
			ID: "b1", CatalogID: "cat1", ExporterName: "mcp-gateway",
			Parameters: map[string]string{}, Enabled: true,
		},
	}, nil)
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return([]*models.CatalogVersionPin{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/data/v1/catalogs/my-catalog/publish/preview", nil)
	req.Header.Set("X-User-Role", "Admin")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.NotEmpty(t, body["session_token"])
	assert.NotNil(t, body["bindings"])
	bindings := body["bindings"].([]any)
	require.Len(t, bindings, 1)
	b0 := bindings[0].(map[string]any)
	assert.Equal(t, "success", b0["status"])
	assert.Equal(t, "b1", b0["binding_id"])
	_, hasFailures := body["has_failures"]
	assert.True(t, hasFailures)
	assert.Equal(t, false, body["has_failures"])
}

// T-34.88: GET /export-bindings/download?token=X&binding=Y returns YAML
func TestT34_88_DownloadArtifacts_ReturnsYAML(t *testing.T) {
	e, _, _, _, _, _, _, _, _, cache := setupExportBindingServerWithCache()

	// Pre-populate the preview cache with artifacts
	token := "test-token-123"
	err := cache.Store(token, export.PreviewCacheEntry{
		CatalogName: "my-catalog",
		Artifacts: map[string][]export.K8sArtifact{
			"b1": {
				{APIVersion: "v1", Kind: "ConfigMap", Name: "cm1", YAML: "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: cm1\n"},
			},
		},
	}, 5*time.Minute)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/data/v1/catalogs/my-catalog/export-bindings/download?token="+token+"&binding=b1", nil)
	req.Header.Set("X-User-Role", "RW")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Disposition"), "my-catalog-export.yaml")
	assert.Contains(t, rec.Body.String(), "apiVersion: v1")
}

// T-34.89: GET /download with expired/missing token returns 404
func TestT34_89_DownloadArtifacts_ExpiredToken(t *testing.T) {
	e, _, _, _, _, _, _, _, _, _ := setupExportBindingServerWithCache()

	req := httptest.NewRequest(http.MethodGet, "/api/data/v1/catalogs/my-catalog/export-bindings/download?token=nonexistent-token&binding=b1", nil)
	req.Header.Set("X-User-Role", "RW")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// T-34.90: RBAC: publish/preview requires Admin+
func TestT34_90_PublishPreview_RW_Forbidden(t *testing.T) {
	e, _, _, _, _, _, _, _, _, _ := setupExportBindingServerWithCache()

	req := httptest.NewRequest(http.MethodPost, "/api/data/v1/catalogs/my-catalog/publish/preview", nil)
	req.Header.Set("X-User-Role", "RW")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// T-34.87: Preview + download flow — preview returns token, download uses it
func TestT34_87_PreviewThenDownload(t *testing.T) {
	e, registry, bindingRepo, catalogRepo, pinRepo, _, _, _, _, _ := setupExportBindingServerWithCache()

	registry.Register(&testExporter{
		name:      "mcp-gateway",
		exportOut: &export.ExportOutput{Artifacts: []export.K8sArtifact{{Name: "test-cr", Kind: "MCPServerRegistration", YAML: "apiVersion: v1\nkind: Test"}}},
	})

	catalogRepo.On("GetByName", mock.Anything, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return([]*models.CatalogVersionPin{}, nil)
	bindingRepo.On("ListByCatalog", mock.Anything, "cat1").Return([]*models.ExportBinding{
		{ID: "b1", CatalogID: "cat1", ExporterName: "mcp-gateway", Parameters: map[string]string{}, Enabled: true},
	}, nil)

	// Step 1: Preview
	req := httptest.NewRequest(http.MethodPost, "/api/data/v1/catalogs/my-catalog/publish/preview", nil)
	req.Header.Set("X-User-Role", "Admin")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var preview map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &preview))
	token := preview["session_token"].(string)
	require.NotEmpty(t, token)

	// Step 2: Download using the token
	req = httptest.NewRequest(http.MethodGet, "/api/data/v1/catalogs/my-catalog/export-bindings/download?token="+token+"&binding=b1", nil)
	req.Header.Set("X-User-Role", "RW")
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "apiVersion")
}

// T-34.91: RBAC: download requires RW+
func TestT34_91_DownloadArtifacts_RO_Forbidden(t *testing.T) {
	e, _, _, _, _, _, _, _, _, _ := setupExportBindingServerWithCache()

	req := httptest.NewRequest(http.MethodGet, "/api/data/v1/catalogs/my-catalog/export-bindings/download?token=some-token&binding=b1", nil)
	req.Header.Set("X-User-Role", "RO")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// Coverage: GET /export-bindings/{id} success
func TestGetBinding_API_Success(t *testing.T) {
	e, _, bindingRepo, catalogRepo, _, _, _, _, _ := setupExportBindingServer()
	catalogRepo.On("GetByName", mock.Anything, "my-catalog").Return(&models.Catalog{
		ID: "cat1", CatalogVersionID: "cv1",
	}, nil)
	bindingRepo.On("GetByID", mock.Anything, "b1").Return(&models.ExportBinding{
		ID: "b1", CatalogID: "cat1", ExporterName: "mcp-gateway",
		Parameters: map[string]string{}, LastRunStatus: "never",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/data/v1/catalogs/my-catalog/export-bindings/b1", nil)
	req.Header.Set("X-User-Role", "RO")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "b1", body["id"])
}

// Coverage: GET /export-bindings/{id} not found
func TestGetBinding_API_NotFound(t *testing.T) {
	e, _, bindingRepo, catalogRepo, _, _, _, _, _ := setupExportBindingServer()
	catalogRepo.On("GetByName", mock.Anything, "my-catalog").Return(&models.Catalog{
		ID: "cat1", CatalogVersionID: "cv1",
	}, nil)
	bindingRepo.On("GetByID", mock.Anything, "nonexistent").Return(nil, fmt.Errorf("not found"))

	req := httptest.NewRequest(http.MethodGet, "/api/data/v1/catalogs/my-catalog/export-bindings/nonexistent", nil)
	req.Header.Set("X-User-Role", "RO")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.True(t, rec.Code >= 400)
}

// Coverage: GET /export-bindings/{id} catalog not found → error
func TestGetBinding_API_CatalogNotFound(t *testing.T) {
	e, _, bindingRepo, catalogRepo, _, _, _, _, _ := setupExportBindingServer()
	_ = bindingRepo

	catalogRepo.On("GetByName", mock.Anything, "missing").Return(nil, fmt.Errorf("not found"))

	req := httptest.NewRequest(http.MethodGet, "/api/data/v1/catalogs/missing/export-bindings/b1", nil)
	req.Header.Set("X-User-Role", "RO")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// Should return an error status
	assert.True(t, rec.Code >= 400)
}

// Coverage: POST /export-bindings with bind error
func TestCreateBinding_API_BindError(t *testing.T) {
	e, _, _, _, _, _, _, _, _ := setupExportBindingServer()

	req := httptest.NewRequest(http.MethodPost, "/api/data/v1/catalogs/my-catalog/export-bindings", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Role", "Admin")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// Coverage: GET /export-bindings service error
func TestListBindings_API_Error(t *testing.T) {
	e, _, _, catalogRepo, _, _, _, _, _ := setupExportBindingServer()
	catalogRepo.On("GetByName", mock.Anything, "missing").Return(nil, fmt.Errorf("not found"))

	req := httptest.NewRequest(http.MethodGet, "/api/data/v1/catalogs/missing/export-bindings", nil)
	req.Header.Set("X-User-Role", "RO")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.True(t, rec.Code >= 400)
}

// Coverage: PUT /export-bindings/{id} bind error
func TestUpdateBinding_API_BindError(t *testing.T) {
	e, _, _, _, _, _, _, _, _ := setupExportBindingServer()

	req := httptest.NewRequest(http.MethodPut, "/api/data/v1/catalogs/my-catalog/export-bindings/b1", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Role", "Admin")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// Coverage: PUT /export-bindings/{id} service error
func TestUpdateBinding_API_ServiceError(t *testing.T) {
	e, _, _, catalogRepo, _, _, _, _, _ := setupExportBindingServer()
	catalogRepo.On("GetByName", mock.Anything, "missing").Return(nil, fmt.Errorf("not found"))

	body, _ := json.Marshal(map[string]any{"enabled": false})
	req := httptest.NewRequest(http.MethodPut, "/api/data/v1/catalogs/missing/export-bindings/b1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Role", "Admin")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.True(t, rec.Code >= 400)
}

// Coverage: DELETE /export-bindings/{id} service error
func TestDeleteBinding_API_Error(t *testing.T) {
	e, _, _, catalogRepo, _, _, _, _, _ := setupExportBindingServer()
	catalogRepo.On("GetByName", mock.Anything, "missing").Return(nil, fmt.Errorf("not found"))

	req := httptest.NewRequest(http.MethodDelete, "/api/data/v1/catalogs/missing/export-bindings/b1", nil)
	req.Header.Set("X-User-Role", "Admin")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.True(t, rec.Code >= 400)
}

// Coverage: POST /run service error
func TestRunBinding_API_Error(t *testing.T) {
	e, _, _, catalogRepo, _, _, _, _, _ := setupExportBindingServer()
	catalogRepo.On("GetByName", mock.Anything, "missing").Return(nil, fmt.Errorf("not found"))

	req := httptest.NewRequest(http.MethodPost, "/api/data/v1/catalogs/missing/export-bindings/b1/run", nil)
	req.Header.Set("X-User-Role", "RW")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.True(t, rec.Code >= 400)
}

// Coverage: POST /publish/preview service error
func TestPublishPreview_API_Error(t *testing.T) {
	e, _, _, catalogRepo, _, _, _, _, _, _ := setupExportBindingServerWithCache()
	catalogRepo.On("GetByName", mock.Anything, "missing").Return(nil, fmt.Errorf("not found"))

	req := httptest.NewRequest(http.MethodPost, "/api/data/v1/catalogs/missing/publish/preview", nil)
	req.Header.Set("X-User-Role", "Admin")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.True(t, rec.Code >= 400)
}

// Coverage: GET /download missing query params
func TestDownloadArtifacts_MissingParams(t *testing.T) {
	e, _, _, _, _, _, _, _, _, _ := setupExportBindingServerWithCache()

	// Missing both token and binding
	req := httptest.NewRequest(http.MethodGet, "/api/data/v1/catalogs/my-catalog/export-bindings/download", nil)
	req.Header.Set("X-User-Role", "RW")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// Coverage: GET /download empty result (no instances)
func TestDownloadArtifacts_EmptyArtifacts(t *testing.T) {
	e, _, _, _, _, _, _, _, _, cache := setupExportBindingServerWithCache()

	token := "empty-token"
	require.NoError(t, cache.Store(token, export.PreviewCacheEntry{
		CatalogName: "my-catalog",
		Artifacts:   map[string][]export.K8sArtifact{"b1": {}},
	}, 5*time.Minute))

	req := httptest.NewRequest(http.MethodGet, "/api/data/v1/catalogs/my-catalog/export-bindings/download?token="+token+"&binding=b1", nil)
	req.Header.Set("X-User-Role", "RW")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "No instances found")
}

// Coverage: GET /download service error (e.g., wrong catalog)
func TestDownloadArtifacts_ServiceError(t *testing.T) {
	e, _, _, _, _, _, _, _, _, cache := setupExportBindingServerWithCache()

	token := "test-token"
	require.NoError(t, cache.Store(token, export.PreviewCacheEntry{
		CatalogName: "other-catalog", // won't match my-catalog
		Artifacts:   map[string][]export.K8sArtifact{"b1": {{Name: "a"}}},
	}, 5*time.Minute))

	req := httptest.NewRequest(http.MethodGet, "/api/data/v1/catalogs/my-catalog/export-bindings/download?token="+token+"&binding=b1", nil)
	req.Header.Set("X-User-Role", "RW")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// T-34.75h: POST /run without virtual_server_instance when binding has virtual_server_type returns 400
func TestT34_75h_RunBinding_MissingVSInstance(t *testing.T) {
	e, registry, bindingRepo, catalogRepo, _, _, _, _, _ := setupExportBindingServer()
	registry.Register(&testExporter{name: "mcp-gateway"})

	catalogRepo.On("GetByName", mock.Anything, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	bindingRepo.On("GetByID", mock.Anything, "b1").Return(&models.ExportBinding{
		ID: "b1", CatalogID: "cat1", ExporterName: "mcp-gateway",
		Parameters: map[string]string{"virtual_server_type": "virtual-server"}, Enabled: true,
	}, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/data/v1/catalogs/my-catalog/export-bindings/b1/run", nil)
	req.Header.Set("X-User-Role", "RW")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "virtual_server_instance")
}

// T-34.75i: Run with non-existent VS instance returns NotFound (unit test in binding_service_coverage_test.go)
// T-34.75j: Run with valid VS instance returns filtered output (unit test in mcp_gateway_exporter_test.go T-34.75d/e/f)
