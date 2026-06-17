package export_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

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
			{Name: "server_type", Type: "entity_type", Required: true},
			{Name: "tool_type", Type: "entity_type", Required: true},
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
			{Name: "server_type", Type: "entity_type", Required: true},
			{Name: "tool_type", Type: "entity_type", Required: true},
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
			{Name: "server_type", Type: "entity_type", Required: true},
			{Name: "tool_type", Type: "entity_type", Required: true},
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
			{Name: "server_type", Type: "entity_type", Required: true},
			{Name: "tool_type", Type: "entity_type", Required: true},
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
			{Name: "server_type", Type: "entity_type", Required: true},
			{Name: "tool_type", Type: "entity_type", Required: true},
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
			{Name: "server_type", Type: "entity_type", Required: true},
			{Name: "tool_type", Type: "entity_type", Required: true},
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
			{Name: "server_type", Type: "entity_type", Required: true},
			{Name: "tool_type", Type: "entity_type", Required: true},
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
		params: []export.ParameterDef{{Name: "server_type", Type: "entity_type", Required: true}},
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
		params: []export.ParameterDef{{Name: "server_type", Type: "entity_type", Required: true}},
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

// T-36.01: BindingStatusSkipped constant equals "skipped"
func TestBindingStatusSkipped_Value(t *testing.T) {
	assert.Equal(t, "skipped", export.BindingStatusSkipped)
}

// T-36.02: validateParamEntityTypes: param "output_type" with type "string" does NOT trigger entity-type validation
func TestCreateBinding_StringParamWithTypeSuffix_NoEntityTypeValidation(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.registry.Register(&stubExporter{
		name: "test-exporter",
		params: []export.ParameterDef{
			{Name: "output_type", Type: "string", Required: true, Description: "Output format type"},
		},
	})

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1"}, nil)
	s.etRepo.On("GetByID", ctx, "et1").Return(&models.EntityType{ID: "et1", Name: "server"}, nil)
	s.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{}, nil)
	s.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{}, nil)
	s.bindingRepo.On("Create", ctx, mock.AnythingOfType("*models.ExportBinding")).Return(nil)

	// "json" is NOT an entity type — but the param is declared as type "string", so it should NOT be validated
	_, err := s.svc.Create(ctx, "my-catalog", "test-exporter", map[string]string{
		"output_type": "json",
	})
	assert.NoError(t, err, "param 'output_type' of type 'string' should not trigger entity-type validation")
}

// T-36.03: validateParamEntityTypes: param "server_ref" with type "entity_type" validates against CV pins
func TestCreateBinding_EntityTypeParamWithoutSuffix_Validates(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.registry.Register(&stubExporter{
		name: "test-exporter",
		params: []export.ParameterDef{
			{Name: "server_ref", Type: "entity_type", Required: true, Description: "Server entity type ref"},
		},
	})

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1"}, nil)
	s.etRepo.On("GetByID", ctx, "et1").Return(&models.EntityType{ID: "et1", Name: "server"}, nil)
	s.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{}, nil)
	s.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{}, nil)

	// "nonexistent-type" is not a pinned entity type — should fail validation
	_, err := s.svc.Create(ctx, "my-catalog", "test-exporter", map[string]string{
		"server_ref": "nonexistent-type",
	})
	require.Error(t, err, "param 'server_ref' of type 'entity_type' should validate against CV pins")
	assert.Contains(t, err.Error(), "not pinned")
}

// T-36.04: validateParamEntityTypes: param "server_type" with type "entity_type" preserved behavior
func TestCreateBinding_EntityTypeParamWithSuffix_StillValidates(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.registry.Register(&stubExporter{
		name: "test-exporter",
		params: []export.ParameterDef{
			{Name: "server_type", Type: "entity_type", Required: true, Description: "Server entity type"},
		},
	})

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1"}, nil)
	s.etRepo.On("GetByID", ctx, "et1").Return(&models.EntityType{ID: "et1", Name: "server"}, nil)
	s.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{}, nil)
	s.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{}, nil)
	s.bindingRepo.On("Create", ctx, mock.AnythingOfType("*models.ExportBinding")).Return(nil)

	// "server" is a pinned entity type — should pass validation
	_, err := s.svc.Create(ctx, "my-catalog", "test-exporter", map[string]string{
		"server_type": "server",
	})
	assert.NoError(t, err, "param 'server_type' of type 'entity_type' with valid pinned type should succeed")
}

// T-36.78: RunAll: registered runs (success), orphaned returns BindingStatusSkipped
func TestRunAll_SkipsOrphanedExporter(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.registry.Register(&stubExporter{
		name:      "registered-exp",
		exportOut: &export.ExportOutput{Artifacts: []export.K8sArtifact{{Name: "a1"}}},
	})

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.bindingRepo.On("ListByCatalog", ctx, "cat1").Return([]*models.ExportBinding{
		{ID: "b1", CatalogID: "cat1", ExporterName: "registered-exp", Enabled: true, Parameters: map[string]string{}},
		{ID: "b2", CatalogID: "cat1", ExporterName: "orphaned-exp", Enabled: true, Parameters: map[string]string{}},
	}, nil)
	s.bindingRepo.On("Update", ctx, mock.AnythingOfType("*models.ExportBinding")).Return(nil)

	results, err := s.svc.RunAll(ctx, "my-catalog")
	require.NoError(t, err)
	require.Len(t, results, 2)

	for _, r := range results {
		if r.ExporterName == "registered-exp" {
			assert.Equal(t, export.BindingStatusSuccess, r.Status)
		} else {
			assert.Equal(t, export.BindingStatusSkipped, r.Status)
			assert.Contains(t, r.Error, "not registered")
		}
	}
}

// T-36.79: RunAll: orphaned binding's LastRunStatus NOT updated
func TestRunAll_OrphanedBindingStatusNotUpdated(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	orphanedBinding := &models.ExportBinding{
		ID: "b1", CatalogID: "cat1", ExporterName: "orphaned-exp",
		Enabled: true, LastRunStatus: "success", Parameters: map[string]string{},
	}
	s.bindingRepo.On("ListByCatalog", ctx, "cat1").Return([]*models.ExportBinding{orphanedBinding}, nil)

	results, err := s.svc.RunAll(ctx, "my-catalog")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, export.BindingStatusSkipped, results[0].Status)
	assert.Equal(t, "success", orphanedBinding.LastRunStatus, "LastRunStatus should NOT be updated for skipped bindings")
}

// T-36.82: Single RunBinding on orphaned → BindingStatusFailed "exporter not found" (NOT skipped)
func TestRunBinding_Orphaned_FailsNotSkipped(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.bindingRepo.On("GetByID", ctx, "b1").Return(&models.ExportBinding{
		ID: "b1", CatalogID: "cat1", ExporterName: "orphaned-exp",
		Parameters: map[string]string{}, Enabled: true,
	}, nil)
	s.bindingRepo.On("Update", ctx, mock.AnythingOfType("*models.ExportBinding")).Return(nil)

	_, err := s.svc.Run(ctx, "my-catalog", "b1", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// T-36.83a: UpdateBinding on orphaned with enabled=false only (params=nil) → succeeds, skips validation
func TestUpdateBinding_Orphaned_EnabledToggleOnly_Succeeds(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", CatalogVersionID: "cv1",
	}, nil)
	disabled := false
	s.bindingRepo.On("GetByID", ctx, "b1").Return(&models.ExportBinding{
		ID: "b1", CatalogID: "cat1", ExporterName: "orphaned-exp",
		Parameters: map[string]string{}, Enabled: true, LastRunStatus: "never",
	}, nil)
	s.bindingRepo.On("Update", ctx, mock.AnythingOfType("*models.ExportBinding")).Return(nil)

	updated, err := s.svc.Update(ctx, "my-catalog", "b1", nil, &disabled)
	require.NoError(t, err, "toggling enabled on orphaned binding with nil params should succeed")
	assert.False(t, updated.Enabled)
}

// T-36.83b: UpdateBinding on orphaned with new params → fails "exporter not found"
func TestUpdateBinding_Orphaned_WithParams_Fails(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", CatalogVersionID: "cv1",
	}, nil)
	s.bindingRepo.On("GetByID", ctx, "b1").Return(&models.ExportBinding{
		ID: "b1", CatalogID: "cat1", ExporterName: "orphaned-exp",
		Parameters: map[string]string{}, Enabled: true, LastRunStatus: "never",
	}, nil)

	_, err := s.svc.Update(ctx, "my-catalog", "b1", map[string]string{"key": "val"}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// --- Group B: Webhook /validate flow ---

// webhookTracker tracks calls to /validate and /export endpoints
type webhookTracker struct {
	mu         sync.Mutex
	calls      []string
	validateFn func(w http.ResponseWriter, r *http.Request)
	exportFn   func(w http.ResponseWriter, r *http.Request)
}

func (wt *webhookTracker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	wt.mu.Lock()
	wt.calls = append(wt.calls, r.URL.Path)
	wt.mu.Unlock()

	switch r.URL.Path {
	case "/validate":
		if wt.validateFn != nil {
			wt.validateFn(w, r)
		} else {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(export.WebhookValidateResponse{Valid: true})
		}
	case "/export":
		if wt.exportFn != nil {
			wt.exportFn(w, r)
		} else {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(export.WebhookExportResponse{
				Artifacts: []export.WebhookArtifact{{Kind: "ConfigMap", Name: "test", YAML: "data: test"}},
			})
		}
	case "/health":
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (wt *webhookTracker) getCalls() []string {
	wt.mu.Lock()
	defer wt.mu.Unlock()
	result := make([]string, len(wt.calls))
	copy(result, wt.calls)
	return result
}

// setupWebhookBindingService creates a binding service with a webhook exporter registered
func setupWebhookBindingService(t *testing.T, tracker *webhookTracker) (*bindingTestSetup, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(tracker)
	t.Cleanup(server.Close)

	s := setupBindingService()

	we, err := export.NewWebhookExporter(
		"test-webhook", "Test Webhook Exporter",
		server.URL, nil, 10,
		func() string { return "test-token" },
	)
	require.NoError(t, err)
	s.registry.RegisterWebhook(we)

	return s, server
}

// T-36.86: CreateBinding with webhook exporter → /validate called
func TestCreateBinding_Webhook_ValidateCalled(t *testing.T) {
	tracker := &webhookTracker{}
	s, _ := setupWebhookBindingService(t, tracker)
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{}, nil)
	s.bindingRepo.On("Create", ctx, mock.AnythingOfType("*models.ExportBinding")).Return(nil)

	_, err := s.svc.Create(ctx, "my-catalog", "test-webhook", map[string]string{})
	require.NoError(t, err)

	calls := tracker.getCalls()
	assert.Contains(t, calls, "/validate", "/validate should have been called during Create")
}

// T-36.87: CreateBinding: /validate returns 400 → binding NOT created
func TestCreateBinding_Webhook_ValidateRejects(t *testing.T) {
	tracker := &webhookTracker{
		validateFn: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(export.WebhookValidateResponse{
				Valid: false,
				Error: "invalid parameters",
			})
		},
	}
	s, _ := setupWebhookBindingService(t, tracker)
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{}, nil)

	_, err := s.svc.Create(ctx, "my-catalog", "test-webhook", map[string]string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid parameters")

	// Verify bindingRepo.Create was NOT called
	s.bindingRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

// T-36.88: PublishPreview: /validate called before /export on webhook binding
func TestPublishPreview_Webhook_ValidateBeforeExport(t *testing.T) {
	tracker := &webhookTracker{}
	s, _ := setupWebhookBindingService(t, tracker)
	ctx := context.Background()

	cache := export.NewInMemoryPreviewCache()
	svc := export.NewExportBindingService(
		s.bindingRepo, s.catalogRepo, s.registry,
		s.cvRepo, s.pinRepo, s.etvRepo, s.etRepo,
		s.attrRepo, s.assocRepo,
		export.WithPreviewCache(cache),
	)

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.bindingRepo.On("ListByCatalog", ctx, "cat1").Return([]*models.ExportBinding{
		{ID: "b1", CatalogID: "cat1", ExporterName: "test-webhook",
			Parameters: map[string]string{}, Enabled: true},
	}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{}, nil)
	s.bindingRepo.On("Update", ctx, mock.AnythingOfType("*models.ExportBinding")).Return(nil)

	result, err := svc.PublishPreview(ctx, "my-catalog")
	require.NoError(t, err)
	require.Len(t, result.Bindings, 1)
	assert.Equal(t, export.BindingStatusSuccess, result.Bindings[0].Status)

	calls := tracker.getCalls()
	// Find indices: /validate must appear before /export
	validateIdx := -1
	exportIdx := -1
	for i, c := range calls {
		if c == "/validate" && validateIdx == -1 {
			validateIdx = i
		}
		if c == "/export" && exportIdx == -1 {
			exportIdx = i
		}
	}
	assert.True(t, validateIdx >= 0, "/validate should have been called")
	assert.True(t, exportIdx >= 0, "/export should have been called")
	assert.True(t, validateIdx < exportIdx, "/validate (idx=%d) must be called before /export (idx=%d)", validateIdx, exportIdx)
}

// T-36.89: PublishPreview: /validate timeout → binding marked failed
func TestPublishPreview_Webhook_ValidateTimeout_BindingFailed(t *testing.T) {
	tracker := &webhookTracker{
		validateFn: func(w http.ResponseWriter, r *http.Request) {
			// Delay longer than the validate timeout (3s is min timeout for validate)
			time.Sleep(5 * time.Second)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(export.WebhookValidateResponse{Valid: true})
		},
	}

	s := setupBindingService()
	// Create a webhook exporter with very short timeout
	server := httptest.NewServer(tracker)
	t.Cleanup(server.Close)

	we, err := export.NewWebhookExporter(
		"slow-webhook", "Slow Webhook",
		server.URL, nil, 3, // 3 seconds total, validate gets 3s (clamped)
		func() string { return "" },
	)
	require.NoError(t, err)
	s.registry.RegisterWebhook(we)

	cache := export.NewInMemoryPreviewCache()
	svc := export.NewExportBindingService(
		s.bindingRepo, s.catalogRepo, s.registry,
		s.cvRepo, s.pinRepo, s.etvRepo, s.etRepo,
		s.attrRepo, s.assocRepo,
		export.WithPreviewCache(cache),
	)

	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.bindingRepo.On("ListByCatalog", ctx, "cat1").Return([]*models.ExportBinding{
		{ID: "b1", CatalogID: "cat1", ExporterName: "slow-webhook",
			Parameters: map[string]string{}, Enabled: true},
	}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{}, nil)
	s.bindingRepo.On("Update", ctx, mock.AnythingOfType("*models.ExportBinding")).Return(nil)

	result, err := svc.PublishPreview(ctx, "my-catalog")
	require.NoError(t, err)
	require.Len(t, result.Bindings, 1)
	assert.Equal(t, export.BindingStatusFailed, result.Bindings[0].Status)
	assert.True(t, result.HasFailures)
}

// --- Group C: RunBinding with unhealthy exporter ---

// T-36.91: RunBinding with healthStatus="Unhealthy" → export ATTEMPTED (HTTP call made)
func TestRunBinding_UnhealthyExporter_ExportAttempted(t *testing.T) {
	tracker := &webhookTracker{}
	server := httptest.NewServer(tracker)
	defer server.Close()

	s := setupBindingService()
	we, err := export.NewWebhookExporter(
		"unhealthy-webhook", "Unhealthy Webhook",
		server.URL, nil, 10,
		func() string { return "" },
	)
	require.NoError(t, err)
	we.SetHealthStatus("Unhealthy")
	s.registry.RegisterWebhook(we)

	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.bindingRepo.On("GetByID", ctx, "b1").Return(&models.ExportBinding{
		ID: "b1", CatalogID: "cat1", ExporterName: "unhealthy-webhook",
		Parameters: map[string]string{}, Enabled: true,
	}, nil)
	s.bindingRepo.On("Update", ctx, mock.AnythingOfType("*models.ExportBinding")).Return(nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{}, nil)

	_, err = s.svc.Run(ctx, "my-catalog", "b1", "")
	require.NoError(t, err)

	calls := tracker.getCalls()
	assert.Contains(t, calls, "/export", "export should be attempted even when health=Unhealthy")
}

// T-36.92: RunBinding with healthStatus="Error" → export ATTEMPTED
func TestRunBinding_ErrorHealthExporter_ExportAttempted(t *testing.T) {
	tracker := &webhookTracker{}
	server := httptest.NewServer(tracker)
	defer server.Close()

	s := setupBindingService()
	we, err := export.NewWebhookExporter(
		"error-webhook", "Error Webhook",
		server.URL, nil, 10,
		func() string { return "" },
	)
	require.NoError(t, err)
	we.SetHealthStatus("Error")
	s.registry.RegisterWebhook(we)

	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.bindingRepo.On("GetByID", ctx, "b1").Return(&models.ExportBinding{
		ID: "b1", CatalogID: "cat1", ExporterName: "error-webhook",
		Parameters: map[string]string{}, Enabled: true,
	}, nil)
	s.bindingRepo.On("Update", ctx, mock.AnythingOfType("*models.ExportBinding")).Return(nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{}, nil)

	_, err = s.svc.Run(ctx, "my-catalog", "b1", "")
	require.NoError(t, err)

	calls := tracker.getCalls()
	assert.Contains(t, calls, "/export", "export should be attempted even when health=Error")
}

// T-36.93: Export succeeds despite unhealthy status → binding status=success
func TestRunBinding_UnhealthyExporter_SuccessStatus(t *testing.T) {
	tracker := &webhookTracker{}
	server := httptest.NewServer(tracker)
	defer server.Close()

	s := setupBindingService()
	we, err := export.NewWebhookExporter(
		"unhealthy-ok-webhook", "Unhealthy OK Webhook",
		server.URL, nil, 10,
		func() string { return "" },
	)
	require.NoError(t, err)
	we.SetHealthStatus("Unhealthy")
	s.registry.RegisterWebhook(we)

	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	binding := &models.ExportBinding{
		ID: "b1", CatalogID: "cat1", ExporterName: "unhealthy-ok-webhook",
		Parameters: map[string]string{}, Enabled: true, LastRunStatus: "never",
	}
	s.bindingRepo.On("GetByID", ctx, "b1").Return(binding, nil)
	s.bindingRepo.On("Update", ctx, mock.AnythingOfType("*models.ExportBinding")).Return(nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{}, nil)

	out, err := s.svc.Run(ctx, "my-catalog", "b1", "")
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, export.BindingStatusSuccess, binding.LastRunStatus)
}

// T-36.94: Export fails (connection refused) despite registered → status=failed
func TestRunBinding_ConnectionRefused_FailedStatus(t *testing.T) {
	s := setupBindingService()

	// Create a server that's immediately closed to get a port that refuses connections
	closedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	closedURL := closedServer.URL
	closedServer.Close()

	we, err := export.NewWebhookExporter(
		"closed-webhook", "Closed Webhook",
		closedURL, nil, 3,
		func() string { return "" },
	)
	require.NoError(t, err)
	s.registry.RegisterWebhook(we)

	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.bindingRepo.On("GetByID", ctx, "b1").Return(&models.ExportBinding{
		ID: "b1", CatalogID: "cat1", ExporterName: "closed-webhook",
		Parameters: map[string]string{}, Enabled: true,
	}, nil)
	s.bindingRepo.On("Update", ctx, mock.AnythingOfType("*models.ExportBinding")).Return(nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{}, nil)

	_, err = s.svc.Run(ctx, "my-catalog", "b1", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unreachable")
}

// --- Group D: Zero-param webhook ---

// T-36.95: Zero-param webhook: CreateBinding with no parameters → success
func TestCreateBinding_ZeroParamWebhook_Success(t *testing.T) {
	tracker := &webhookTracker{}
	s, _ := setupWebhookBindingService(t, tracker)
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{}, nil)
	s.bindingRepo.On("Create", ctx, mock.AnythingOfType("*models.ExportBinding")).Return(nil)

	binding, err := s.svc.Create(ctx, "my-catalog", "test-webhook", map[string]string{})
	require.NoError(t, err)
	require.NotNil(t, binding)
	assert.Equal(t, "test-webhook", binding.ExporterName)
}

// T-36.96: Zero-param webhook: RunBinding → /export request has "parameters":{}
func TestRunBinding_ZeroParamWebhook_EmptyParamsInRequest(t *testing.T) {
	var capturedBody []byte
	tracker := &webhookTracker{
		exportFn: func(w http.ResponseWriter, r *http.Request) {
			capturedBody, _ = io.ReadAll(r.Body)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(export.WebhookExportResponse{
				Artifacts: []export.WebhookArtifact{{Kind: "CM", Name: "test", YAML: "data: ok"}},
			})
		},
	}
	s, _ := setupWebhookBindingService(t, tracker)
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.bindingRepo.On("GetByID", ctx, "b1").Return(&models.ExportBinding{
		ID: "b1", CatalogID: "cat1", ExporterName: "test-webhook",
		Parameters: map[string]string{}, Enabled: true,
	}, nil)
	s.bindingRepo.On("Update", ctx, mock.AnythingOfType("*models.ExportBinding")).Return(nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{}, nil)

	out, err := s.svc.Run(ctx, "my-catalog", "b1", "")
	require.NoError(t, err)
	require.NotNil(t, out)

	assert.Contains(t, string(capturedBody), `"parameters":{}`,
		"export request body should contain empty parameters object")
}
