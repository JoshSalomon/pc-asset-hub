package export_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository/mocks"
	"github.com/project-catalyst/pc-asset-hub/internal/service/operational/export"
)

type bindingTestSetup struct {
	svc         *export.ExportBindingService
	bindingRepo *mocks.MockExportBindingRepo
	catalogRepo *mocks.MockCatalogRepo
	registry    *export.ExporterRegistry
	cvRepo      *mocks.MockCatalogVersionRepo
	pinRepo     *mocks.MockCatalogVersionPinRepo
	etvRepo     *mocks.MockEntityTypeVersionRepo
	etRepo      *mocks.MockEntityTypeRepo
	attrRepo    *mocks.MockAttributeRepo
	assocRepo   *mocks.MockAssociationRepo
}

func setupBindingService() *bindingTestSetup {
	s := &bindingTestSetup{
		bindingRepo: new(mocks.MockExportBindingRepo),
		catalogRepo: new(mocks.MockCatalogRepo),
		registry:    export.NewExporterRegistry(),
		cvRepo:      new(mocks.MockCatalogVersionRepo),
		pinRepo:     new(mocks.MockCatalogVersionPinRepo),
		etvRepo:     new(mocks.MockEntityTypeVersionRepo),
		etRepo:      new(mocks.MockEntityTypeRepo),
		attrRepo:    new(mocks.MockAttributeRepo),
		assocRepo:   new(mocks.MockAssociationRepo),
	}
	s.svc = export.NewExportBindingService(
		s.bindingRepo, s.catalogRepo, s.registry,
		s.cvRepo, s.pinRepo, s.etvRepo, s.etRepo,
		s.attrRepo, s.assocRepo,
	)
	return s
}

// T-34.07: Create binding with valid exporter and parameters
func TestCreateBinding_Success(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.registry.Register(&stubExporter{
		name: "mcp-gateway",
		desc: "MCP Gateway Exporter",
		params: []export.ParameterDef{
			{Name: "server_type", Type: "string", Required: true},
			{Name: "tool_type", Type: "string", Required: true},
			{Name: "target_namespace", Type: "string", Default: "default"},
		},
	})

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{EntityTypeVersionID: "etv1"},
		{EntityTypeVersionID: "etv2"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1"}, nil)
	s.etvRepo.On("GetByID", ctx, "etv2").Return(&models.EntityTypeVersion{ID: "etv2", EntityTypeID: "et2"}, nil)
	s.etRepo.On("GetByID", ctx, "et1").Return(&models.EntityType{ID: "et1", Name: "mcp-server"}, nil)
	s.etRepo.On("GetByID", ctx, "et2").Return(&models.EntityType{ID: "et2", Name: "mcp-tool"}, nil)
	s.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{
		{Name: "route_name"},
	}, nil)
	s.attrRepo.On("ListByVersion", ctx, "etv2").Return([]*models.Attribute{}, nil)
	s.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{
		{Type: "containment", TargetEntityTypeID: "et2", Name: "tools"},
	}, nil)
	s.assocRepo.On("ListByVersion", ctx, "etv2").Return([]*models.Association{}, nil)
	s.bindingRepo.On("Create", ctx, mock.AnythingOfType("*models.ExportBinding")).Return(nil)

	binding, err := s.svc.Create(ctx, "my-catalog", "mcp-gateway", map[string]string{
		"server_type": "mcp-server",
		"tool_type":   "mcp-tool",
	})
	require.NoError(t, err)
	require.NotNil(t, binding)
	assert.NotEmpty(t, binding.ID)
	assert.Equal(t, "cat1", binding.CatalogID)
	assert.Equal(t, "mcp-gateway", binding.ExporterName)
	assert.True(t, binding.Enabled)
	assert.Equal(t, "never", binding.LastRunStatus)
}

// T-34.08: Create binding with non-existent exporter returns error
func TestCreateBinding_ExporterNotFound(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", CatalogVersionID: "cv1",
	}, nil)

	_, err := s.svc.Create(ctx, "my-catalog", "no-such-exporter", map[string]string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exporter")
}

// T-34.09: Create binding with missing required parameter returns error
func TestCreateBinding_MissingRequiredParam(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.registry.Register(&stubExporter{
		name: "mcp-gateway",
		params: []export.ParameterDef{
			{Name: "server_type", Type: "string", Required: true},
			{Name: "tool_type", Type: "string", Required: true},
		},
	})

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", CatalogVersionID: "cv1",
	}, nil)

	_, err := s.svc.Create(ctx, "my-catalog", "mcp-gateway", map[string]string{
		"server_type": "mcp-server",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tool_type")
}

// T-34.10: Create binding — ValidateSchema checks entity type exists in CV
func TestCreateBinding_EntityTypeNotPinned(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.registry.Register(&stubExporter{
		name: "mcp-gateway",
		params: []export.ParameterDef{
			{Name: "server_type", Type: "string", Required: true},
			{Name: "tool_type", Type: "string", Required: true},
		},
	})

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", CatalogVersionID: "cv1",
	}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1"}, nil)
	s.etRepo.On("GetByID", ctx, "et1").Return(&models.EntityType{ID: "et1", Name: "other-type"}, nil)
	s.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{}, nil)
	s.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{}, nil)

	_, err := s.svc.Create(ctx, "my-catalog", "mcp-gateway", map[string]string{
		"server_type": "mcp-server",
		"tool_type":   "mcp-tool",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not pinned")
}

// T-34.13: Create multiple bindings to same exporter with different params
func TestCreateBinding_MultipleSameExporter(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.registry.Register(&stubExporter{
		name: "mcp-gateway",
		params: []export.ParameterDef{
			{Name: "server_type", Type: "string", Required: true},
			{Name: "tool_type", Type: "string", Required: true},
		},
	})

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", CatalogVersionID: "cv1",
	}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{EntityTypeVersionID: "etv1"},
		{EntityTypeVersionID: "etv2"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1"}, nil)
	s.etvRepo.On("GetByID", ctx, "etv2").Return(&models.EntityTypeVersion{ID: "etv2", EntityTypeID: "et2"}, nil)
	s.etRepo.On("GetByID", ctx, "et1").Return(&models.EntityType{ID: "et1", Name: "mcp-server"}, nil)
	s.etRepo.On("GetByID", ctx, "et2").Return(&models.EntityType{ID: "et2", Name: "mcp-tool"}, nil)
	s.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{{Name: "route_name"}}, nil)
	s.attrRepo.On("ListByVersion", ctx, "etv2").Return([]*models.Attribute{}, nil)
	s.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{
		{Type: "containment", TargetEntityTypeID: "et2"},
	}, nil)
	s.assocRepo.On("ListByVersion", ctx, "etv2").Return([]*models.Association{}, nil)
	s.bindingRepo.On("Create", ctx, mock.AnythingOfType("*models.ExportBinding")).Return(nil)

	b1, err := s.svc.Create(ctx, "my-catalog", "mcp-gateway", map[string]string{
		"server_type": "mcp-server", "tool_type": "mcp-tool",
	})
	require.NoError(t, err)

	b2, err := s.svc.Create(ctx, "my-catalog", "mcp-gateway", map[string]string{
		"server_type": "mcp-server", "tool_type": "mcp-tool",
	})
	require.NoError(t, err)
	assert.NotEqual(t, b1.ID, b2.ID)
}

// T-34.14: Update binding parameters
func TestUpdateBinding_Parameters(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.registry.Register(&stubExporter{
		name: "mcp-gateway",
		params: []export.ParameterDef{
			{Name: "server_type", Type: "string", Required: true},
			{Name: "tool_type", Type: "string", Required: true},
		},
	})

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", CatalogVersionID: "cv1",
	}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{EntityTypeVersionID: "etv1"}, {EntityTypeVersionID: "etv2"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1"}, nil)
	s.etvRepo.On("GetByID", ctx, "etv2").Return(&models.EntityTypeVersion{ID: "etv2", EntityTypeID: "et2"}, nil)
	s.etRepo.On("GetByID", ctx, "et1").Return(&models.EntityType{ID: "et1", Name: "mcp-server"}, nil)
	s.etRepo.On("GetByID", ctx, "et2").Return(&models.EntityType{ID: "et2", Name: "mcp-tool"}, nil)
	s.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{{Name: "route_name"}}, nil)
	s.attrRepo.On("ListByVersion", ctx, "etv2").Return([]*models.Attribute{}, nil)
	s.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{
		{Type: "containment", TargetEntityTypeID: "et2"},
	}, nil)
	s.assocRepo.On("ListByVersion", ctx, "etv2").Return([]*models.Association{}, nil)
	s.bindingRepo.On("GetByID", ctx, "bind1").Return(&models.ExportBinding{
		ID: "bind1", CatalogID: "cat1", ExporterName: "mcp-gateway",
		Parameters: map[string]string{"server_type": "mcp-server", "tool_type": "mcp-tool"},
		Enabled: true, LastRunStatus: "never",
	}, nil)
	s.bindingRepo.On("Update", ctx, mock.AnythingOfType("*models.ExportBinding")).Return(nil)

	updated, err := s.svc.Update(ctx, "my-catalog", "bind1", map[string]string{
		"server_type": "mcp-server", "tool_type": "mcp-tool", "target_namespace": "prod",
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, "prod", updated.Parameters["target_namespace"])
}

// T-34.15: Update binding enabled/disabled toggle
func TestUpdateBinding_EnabledToggle(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", CatalogVersionID: "cv1",
	}, nil)
	disabled := false
	s.bindingRepo.On("GetByID", ctx, "bind1").Return(&models.ExportBinding{
		ID: "bind1", CatalogID: "cat1", ExporterName: "mcp-gateway",
		Parameters: map[string]string{}, Enabled: true, LastRunStatus: "never",
	}, nil)
	s.bindingRepo.On("Update", ctx, mock.AnythingOfType("*models.ExportBinding")).Return(nil)

	updated, err := s.svc.Update(ctx, "my-catalog", "bind1", nil, &disabled)
	require.NoError(t, err)
	assert.False(t, updated.Enabled)
}

// T-34.16: Delete binding
func TestDeleteBinding(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", CatalogVersionID: "cv1",
	}, nil)
	s.bindingRepo.On("GetByID", ctx, "bind1").Return(&models.ExportBinding{
		ID: "bind1", CatalogID: "cat1",
	}, nil)
	s.bindingRepo.On("Delete", ctx, "bind1").Return(nil)

	err := s.svc.Delete(ctx, "my-catalog", "bind1")
	assert.NoError(t, err)
}

// T-34.17: List bindings by catalog returns only that catalog's bindings
func TestListBindings_FilteredByCatalog(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", CatalogVersionID: "cv1",
	}, nil)
	s.bindingRepo.On("ListByCatalog", ctx, "cat1").Return([]*models.ExportBinding{
		{ID: "b1", CatalogID: "cat1", ExporterName: "mcp-gateway"},
		{ID: "b2", CatalogID: "cat1", ExporterName: "mcp-gateway"},
	}, nil)

	bindings, err := s.svc.List(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Len(t, bindings, 2)
}

// T-34.11 (deeper): ValidateSchema — entity type missing required attribute
// Register exporter whose ValidateSchema returns error about missing attribute;
// verify Create binding fails with the validation error.
func TestCreateBinding_ValidateSchema_MissingAttribute(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.registry.Register(&stubExporter{
		name: "strict-exporter",
		params: []export.ParameterDef{
			{Name: "server_type", Type: "string", Required: true},
			{Name: "tool_type", Type: "string", Required: true},
		},
		validateErr: fmt.Errorf("entity type %q is missing required attribute 'route_name'", "mcp-server"),
	})

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{EntityTypeVersionID: "etv1"},
		{EntityTypeVersionID: "etv2"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1"}, nil)
	s.etvRepo.On("GetByID", ctx, "etv2").Return(&models.EntityTypeVersion{ID: "etv2", EntityTypeID: "et2"}, nil)
	s.etRepo.On("GetByID", ctx, "et1").Return(&models.EntityType{ID: "et1", Name: "mcp-server"}, nil)
	s.etRepo.On("GetByID", ctx, "et2").Return(&models.EntityType{ID: "et2", Name: "mcp-tool"}, nil)
	s.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{
		{Name: "description"}, // route_name is missing
	}, nil)
	s.attrRepo.On("ListByVersion", ctx, "etv2").Return([]*models.Attribute{}, nil)
	s.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{
		{Type: "containment", TargetEntityTypeID: "et2"},
	}, nil)
	s.assocRepo.On("ListByVersion", ctx, "etv2").Return([]*models.Association{}, nil)

	_, err := s.svc.Create(ctx, "my-catalog", "strict-exporter", map[string]string{
		"server_type": "mcp-server",
		"tool_type":   "mcp-tool",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "route_name",
		"error should mention the missing attribute")
}

// T-34.12 (deeper): ValidateSchema — missing containment association
// Register exporter whose ValidateSchema returns error about containment;
// verify Create binding fails.
func TestCreateBinding_ValidateSchema_MissingContainment(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.registry.Register(&stubExporter{
		name: "containment-checker",
		params: []export.ParameterDef{
			{Name: "server_type", Type: "string", Required: true},
			{Name: "tool_type", Type: "string", Required: true},
		},
		validateErr: fmt.Errorf("entity type %q has no containment association to %q", "mcp-server", "mcp-tool"),
	})

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{EntityTypeVersionID: "etv1"},
		{EntityTypeVersionID: "etv2"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1"}, nil)
	s.etvRepo.On("GetByID", ctx, "etv2").Return(&models.EntityTypeVersion{ID: "etv2", EntityTypeID: "et2"}, nil)
	s.etRepo.On("GetByID", ctx, "et1").Return(&models.EntityType{ID: "et1", Name: "mcp-server"}, nil)
	s.etRepo.On("GetByID", ctx, "et2").Return(&models.EntityType{ID: "et2", Name: "mcp-tool"}, nil)
	s.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{
		{Name: "route_name"},
	}, nil)
	s.attrRepo.On("ListByVersion", ctx, "etv2").Return([]*models.Attribute{}, nil)
	s.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{}, nil) // no containment
	s.assocRepo.On("ListByVersion", ctx, "etv2").Return([]*models.Association{}, nil)

	_, err := s.svc.Create(ctx, "my-catalog", "containment-checker", map[string]string{
		"server_type": "mcp-server",
		"tool_type":   "mcp-tool",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "containment",
		"error should mention missing containment association")
}

// Item 2: Create does not double-wrap validation errors
// validateBindingParams already returns domainerrors.NewValidation, so Create
// should pass the error through without re-wrapping.
func TestCreate_NoDoubleWrappedValidationError(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.registry.Register(&stubExporter{
		name:   "mcp-gateway",
		params: []export.ParameterDef{{Name: "server_type", Type: "string", Required: true}},
	})

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)

	// Missing required parameter triggers a validation error from validateBindingParams
	_, err := s.svc.Create(ctx, "my-catalog", "mcp-gateway", map[string]string{})
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "VALIDATION: VALIDATION:",
		"validation error should not be double-wrapped with redundant VALIDATION prefix")
}

// Item 2: Update does not double-wrap validation errors
func TestUpdate_NoDoubleWrappedValidationError(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.registry.Register(&stubExporter{
		name:   "mcp-gateway",
		params: []export.ParameterDef{{Name: "server_type", Type: "string", Required: true}},
	})

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.bindingRepo.On("GetByID", ctx, "b1").Return(&models.ExportBinding{
		ID: "b1", CatalogID: "cat1", ExporterName: "mcp-gateway",
		Parameters: map[string]string{"server_type": "mcp-server"}, Enabled: true,
	}, nil)

	// Update with empty params → missing required param → validation error
	_, err := s.svc.Update(ctx, "my-catalog", "b1", map[string]string{}, nil)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "VALIDATION: VALIDATION:",
		"validation error should not be double-wrapped with redundant VALIDATION prefix")
}
