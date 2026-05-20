package export_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository/mocks"
	"github.com/project-catalyst/pc-asset-hub/internal/service/operational/export"
)

// --- WithInstanceRepos coverage ---

func TestWithInstanceRepos_SetsRepos(t *testing.T) {
	instRepo := new(mocks.MockEntityInstanceRepo)
	iavRepo := new(mocks.MockInstanceAttributeValueRepo)

	s := setupBindingService()
	svc := export.NewExportBindingService(
		s.bindingRepo, s.catalogRepo, s.registry,
		s.cvRepo, s.pinRepo, s.etvRepo, s.etRepo,
		s.attrRepo, s.assocRepo,
		export.WithInstanceRepos(instRepo, iavRepo),
	)

	// The service should have repos set; we verify by calling Run which needs instRepo
	ctx := context.Background()
	s.registry.Register(&stubExporter{
		name:      "test-exp",
		exportOut: &export.ExportOutput{Artifacts: []export.K8sArtifact{{Name: "a", YAML: "test: true"}}},
	})
	s.catalogRepo.On("GetByName", ctx, "c1").Return(&models.Catalog{
		ID: "cat1", Name: "c1", CatalogVersionID: "cv1",
	}, nil)
	s.bindingRepo.On("GetByID", ctx, "b1").Return(&models.ExportBinding{
		ID: "b1", CatalogID: "cat1", ExporterName: "test-exp",
		Parameters: map[string]string{}, Enabled: true,
	}, nil)
	s.bindingRepo.On("Update", ctx, mock.AnythingOfType("*models.ExportBinding")).Return(nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{}, nil)
	instRepo.On("ListByCatalog", ctx, "cat1").Return([]*models.EntityInstance{}, nil)

	out, err := svc.Run(ctx, "c1", "b1", "")
	require.NoError(t, err)
	require.NotNil(t, out)
}

// --- Get coverage ---

func TestGet_Success(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", CatalogVersionID: "cv1",
	}, nil)
	s.bindingRepo.On("GetByID", ctx, "b1").Return(&models.ExportBinding{
		ID: "b1", CatalogID: "cat1", ExporterName: "mcp-gateway",
	}, nil)

	binding, err := s.svc.Get(ctx, "my-catalog", "b1")
	require.NoError(t, err)
	assert.Equal(t, "b1", binding.ID)
}

func TestGet_CatalogNotFound(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "missing").Return(nil, fmt.Errorf("not found"))

	_, err := s.svc.Get(ctx, "missing", "b1")
	require.Error(t, err)
}

func TestGet_BindingNotFound(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", CatalogVersionID: "cv1",
	}, nil)
	s.bindingRepo.On("GetByID", ctx, "missing").Return(nil, fmt.Errorf("not found"))

	_, err := s.svc.Get(ctx, "my-catalog", "missing")
	require.Error(t, err)
}

func TestGet_BindingWrongCatalog(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", CatalogVersionID: "cv1",
	}, nil)
	s.bindingRepo.On("GetByID", ctx, "b1").Return(&models.ExportBinding{
		ID: "b1", CatalogID: "other-cat", // doesn't match cat1
	}, nil)

	_, err := s.svc.Get(ctx, "my-catalog", "b1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// --- Run coverage ---

func TestRun_Success(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.registry.Register(&stubExporter{
		name:      "test-exp",
		exportOut: &export.ExportOutput{Artifacts: []export.K8sArtifact{{Name: "cm1", Kind: "ConfigMap", YAML: "test: true"}}},
	})

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.bindingRepo.On("GetByID", ctx, "b1").Return(&models.ExportBinding{
		ID: "b1", CatalogID: "cat1", ExporterName: "test-exp",
		Parameters: map[string]string{}, Enabled: true,
	}, nil)
	s.bindingRepo.On("Update", ctx, mock.AnythingOfType("*models.ExportBinding")).Return(nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{}, nil)

	out, err := s.svc.Run(ctx, "my-catalog", "b1", "")
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Len(t, out.Artifacts, 1)
}

func TestRun_CatalogNotFound(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "missing").Return(nil, fmt.Errorf("not found"))

	_, err := s.svc.Run(ctx, "missing", "b1", "")
	require.Error(t, err)
}

func TestRun_ExporterNotFound(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.bindingRepo.On("GetByID", ctx, "b1").Return(&models.ExportBinding{
		ID: "b1", CatalogID: "cat1", ExporterName: "no-such-exporter",
		Parameters: map[string]string{}, Enabled: true,
	}, nil)
	s.bindingRepo.On("Update", ctx, mock.AnythingOfType("*models.ExportBinding")).Return(nil)

	_, err := s.svc.Run(ctx, "my-catalog", "b1", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no-such-exporter")
}

func TestRun_ExportError(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.registry.Register(&stubExporter{
		name:      "fail-exp",
		exportErr: fmt.Errorf("export failed"),
	})

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.bindingRepo.On("GetByID", ctx, "b1").Return(&models.ExportBinding{
		ID: "b1", CatalogID: "cat1", ExporterName: "fail-exp",
		Parameters: map[string]string{}, Enabled: true,
	}, nil)
	s.bindingRepo.On("Update", ctx, mock.AnythingOfType("*models.ExportBinding")).Return(nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{}, nil)

	_, err := s.svc.Run(ctx, "my-catalog", "b1", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "export failed")
}

func TestRun_BuildExportInputError(t *testing.T) {
	instRepo := new(mocks.MockEntityInstanceRepo)
	iavRepo := new(mocks.MockInstanceAttributeValueRepo)

	s := setupBindingService()
	svc := export.NewExportBindingService(
		s.bindingRepo, s.catalogRepo, s.registry,
		s.cvRepo, s.pinRepo, s.etvRepo, s.etRepo,
		s.attrRepo, s.assocRepo,
		export.WithInstanceRepos(instRepo, iavRepo),
	)
	ctx := context.Background()

	s.registry.Register(&stubExporter{
		name:      "test-exp",
		exportOut: &export.ExportOutput{},
	})

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.bindingRepo.On("GetByID", ctx, "b1").Return(&models.ExportBinding{
		ID: "b1", CatalogID: "cat1", ExporterName: "test-exp",
		Parameters: map[string]string{}, Enabled: true,
	}, nil)
	s.bindingRepo.On("Update", ctx, mock.AnythingOfType("*models.ExportBinding")).Return(nil)
	instRepo.On("ListByCatalog", ctx, "cat1").Return(nil, fmt.Errorf("db error"))

	_, err := svc.Run(ctx, "my-catalog", "b1", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

// --- RunAll error paths ---

func TestRunAll_CatalogNotFound(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "missing").Return(nil, fmt.Errorf("not found"))

	_, err := s.svc.RunAll(ctx, "missing")
	require.Error(t, err)
}

func TestRunAll_ListBindingsError(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", CatalogVersionID: "cv1",
	}, nil)
	s.bindingRepo.On("ListByCatalog", ctx, "cat1").Return(nil, fmt.Errorf("db error"))

	_, err := s.svc.RunAll(ctx, "my-catalog")
	require.Error(t, err)
}

// --- executeBinding error paths ---

func TestRunAll_ExecuteBinding_ExporterNotFound(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.bindingRepo.On("ListByCatalog", ctx, "cat1").Return([]*models.ExportBinding{
		{ID: "b1", CatalogID: "cat1", ExporterName: "missing-exporter",
			Parameters: map[string]string{}, Enabled: true},
	}, nil)
	s.bindingRepo.On("Update", ctx, mock.AnythingOfType("*models.ExportBinding")).Return(nil)

	results, err := s.svc.RunAll(ctx, "my-catalog")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "failed", results[0].Status)
	assert.Contains(t, results[0].Error, "missing-exporter")
}

func TestRunAll_ExecuteBinding_BuildInputError(t *testing.T) {
	instRepo := new(mocks.MockEntityInstanceRepo)
	iavRepo := new(mocks.MockInstanceAttributeValueRepo)

	s := setupBindingService()
	svc := export.NewExportBindingService(
		s.bindingRepo, s.catalogRepo, s.registry,
		s.cvRepo, s.pinRepo, s.etvRepo, s.etRepo,
		s.attrRepo, s.assocRepo,
		export.WithInstanceRepos(instRepo, iavRepo),
	)
	ctx := context.Background()

	s.registry.Register(&stubExporter{name: "test-exp", exportOut: &export.ExportOutput{}})

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.bindingRepo.On("ListByCatalog", ctx, "cat1").Return([]*models.ExportBinding{
		{ID: "b1", CatalogID: "cat1", ExporterName: "test-exp",
			Parameters: map[string]string{}, Enabled: true},
	}, nil)
	s.bindingRepo.On("Update", ctx, mock.AnythingOfType("*models.ExportBinding")).Return(nil)
	instRepo.On("ListByCatalog", ctx, "cat1").Return(nil, fmt.Errorf("db error"))

	results, err := svc.RunAll(ctx, "my-catalog")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "failed", results[0].Status)
	assert.Contains(t, results[0].Error, "db error")
}

func TestRunAll_ExecuteBinding_ExportFails(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.registry.Register(&stubExporter{
		name:      "fail-exp",
		exportErr: fmt.Errorf("export failed"),
	})

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.bindingRepo.On("ListByCatalog", ctx, "cat1").Return([]*models.ExportBinding{
		{ID: "b1", CatalogID: "cat1", ExporterName: "fail-exp",
			Parameters: map[string]string{}, Enabled: true},
	}, nil)
	s.bindingRepo.On("Update", ctx, mock.AnythingOfType("*models.ExportBinding")).Return(nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{}, nil)

	results, err := s.svc.RunAll(ctx, "my-catalog")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "failed", results[0].Status)
	assert.Contains(t, results[0].Error, "export failed")
}

// --- Create error paths ---

func TestCreate_CatalogNotFound(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "missing").Return(nil, fmt.Errorf("not found"))

	_, err := s.svc.Create(ctx, "missing", "mcp-gateway", nil)
	require.Error(t, err)
}

func TestCreate_RepoCreateError(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.registry.Register(&stubExporter{
		name: "mcp-gateway",
		params: []export.ParameterDef{
			{Name: "server_type", Type: "string", Required: true},
		},
	})

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", CatalogVersionID: "cv1",
	}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1"}, nil)
	s.etRepo.On("GetByID", ctx, "et1").Return(&models.EntityType{ID: "et1", Name: "mcp-server"}, nil)
	s.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{}, nil)
	s.assocRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Association{}, nil)
	s.bindingRepo.On("Create", ctx, mock.AnythingOfType("*models.ExportBinding")).Return(fmt.Errorf("db error"))

	_, err := s.svc.Create(ctx, "my-catalog", "mcp-gateway", map[string]string{
		"server_type": "mcp-server",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

// --- List error paths ---

func TestList_CatalogNotFound(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "missing").Return(nil, fmt.Errorf("not found"))

	_, err := s.svc.List(ctx, "missing")
	require.Error(t, err)
}

// --- Update error paths ---

func TestUpdate_CatalogNotFound(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "missing").Return(nil, fmt.Errorf("not found"))

	_, err := s.svc.Update(ctx, "missing", "b1", nil, nil)
	require.Error(t, err)
}

func TestUpdate_RepoUpdateError(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", CatalogVersionID: "cv1",
	}, nil)
	s.bindingRepo.On("GetByID", ctx, "b1").Return(&models.ExportBinding{
		ID: "b1", CatalogID: "cat1", ExporterName: "mcp-gateway",
		Parameters: map[string]string{}, Enabled: true,
	}, nil)
	s.bindingRepo.On("Update", ctx, mock.AnythingOfType("*models.ExportBinding")).Return(fmt.Errorf("db error"))

	enabled := false
	_, err := s.svc.Update(ctx, "my-catalog", "b1", nil, &enabled)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

// --- Delete error paths ---

func TestDelete_CatalogNotFound(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "missing").Return(nil, fmt.Errorf("not found"))

	err := s.svc.Delete(ctx, "missing", "b1")
	require.Error(t, err)
}

// --- buildSchemaInfo error paths ---

func TestCreate_BuildSchemaInfo_PinRepoError(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.registry.Register(&stubExporter{name: "test", params: []export.ParameterDef{}})

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", CatalogVersionID: "cv1",
	}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return(nil, fmt.Errorf("pin error"))

	_, err := s.svc.Create(ctx, "my-catalog", "test", map[string]string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pin error")
}

func TestCreate_BuildSchemaInfo_ETVRepoError(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.registry.Register(&stubExporter{name: "test", params: []export.ParameterDef{}})

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", CatalogVersionID: "cv1",
	}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(nil, fmt.Errorf("etv error"))

	_, err := s.svc.Create(ctx, "my-catalog", "test", map[string]string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "etv error")
}

func TestCreate_BuildSchemaInfo_ETRepoError(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.registry.Register(&stubExporter{name: "test", params: []export.ParameterDef{}})

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", CatalogVersionID: "cv1",
	}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1"}, nil)
	s.etRepo.On("GetByID", ctx, "et1").Return(nil, fmt.Errorf("et error"))

	_, err := s.svc.Create(ctx, "my-catalog", "test", map[string]string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "et error")
}

func TestCreate_BuildSchemaInfo_AttrRepoError(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.registry.Register(&stubExporter{name: "test", params: []export.ParameterDef{}})

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", CatalogVersionID: "cv1",
	}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1"}, nil)
	s.etRepo.On("GetByID", ctx, "et1").Return(&models.EntityType{ID: "et1", Name: "server"}, nil)
	s.attrRepo.On("ListByVersion", ctx, "etv1").Return(nil, fmt.Errorf("attr error"))

	_, err := s.svc.Create(ctx, "my-catalog", "test", map[string]string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "attr error")
}

func TestCreate_BuildSchemaInfo_AssocRepoError(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.registry.Register(&stubExporter{name: "test", params: []export.ParameterDef{}})

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", CatalogVersionID: "cv1",
	}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1"}, nil)
	s.etRepo.On("GetByID", ctx, "et1").Return(&models.EntityType{ID: "et1", Name: "server"}, nil)
	s.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{}, nil)
	s.assocRepo.On("ListByVersion", ctx, "etv1").Return(nil, fmt.Errorf("assoc error"))

	_, err := s.svc.Create(ctx, "my-catalog", "test", map[string]string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "assoc error")
}

// --- validateBindingParams error path: buildSchemaInfo fails ---

func TestValidateBindingParams_BuildSchemaError(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.registry.Register(&stubExporter{
		name:   "test",
		params: []export.ParameterDef{{Name: "p1", Required: true}},
	})

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", CatalogVersionID: "cv1",
	}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return(nil, fmt.Errorf("pin error"))

	_, err := s.svc.Create(ctx, "my-catalog", "test", map[string]string{"p1": "v1"})
	require.Error(t, err)
}

// --- buildInstancesByType coverage ---

func setupWithInstanceRepos(t *testing.T) (*export.ExportBindingService, *bindingTestSetup, *mocks.MockEntityInstanceRepo, *mocks.MockInstanceAttributeValueRepo) {
	t.Helper()
	instRepo := new(mocks.MockEntityInstanceRepo)
	iavRepo := new(mocks.MockInstanceAttributeValueRepo)
	s := setupBindingService()

	svc := export.NewExportBindingService(
		s.bindingRepo, s.catalogRepo, s.registry,
		s.cvRepo, s.pinRepo, s.etvRepo, s.etRepo,
		s.attrRepo, s.assocRepo,
		export.WithInstanceRepos(instRepo, iavRepo),
	)
	return svc, s, instRepo, iavRepo
}

func TestRun_BuildInstancesByType_FullPath(t *testing.T) {
	svc, s, instRepo, iavRepo := setupWithInstanceRepos(t)
	ctx := context.Background()

	s.registry.Register(&stubExporter{
		name:      "test-exp",
		exportOut: &export.ExportOutput{Artifacts: []export.K8sArtifact{{Name: "a", YAML: "test: true"}}},
	})

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.bindingRepo.On("GetByID", ctx, "b1").Return(&models.ExportBinding{
		ID: "b1", CatalogID: "cat1", ExporterName: "test-exp",
		Parameters: map[string]string{}, Enabled: true,
	}, nil)
	s.bindingRepo.On("Update", ctx, mock.AnythingOfType("*models.ExportBinding")).Return(nil)

	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1"}, nil)
	s.etRepo.On("GetByID", ctx, "et1").Return(&models.EntityType{ID: "et1", Name: "server"}, nil)
	s.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{
		{ID: "a1", Name: "route_name"},
		{ID: "a2", Name: "config"},
	}, nil)

	instRepo.On("ListByCatalog", ctx, "cat1").Return([]*models.EntityInstance{
		{ID: "i1", EntityTypeID: "et1", Name: "server1", Version: 1, ParentInstanceID: "parent1"},
	}, nil)
	iavRepo.On("GetValuesForVersion", ctx, "i1", 1).Return([]*models.InstanceAttributeValue{
		{AttributeID: "a1", ValueString: "my-route"},
		{AttributeID: "a2", ValueJSON: `{"key":"val"}`},
	}, nil)

	out, err := svc.Run(ctx, "my-catalog", "b1", "")
	require.NoError(t, err)
	require.NotNil(t, out)
}

func TestRun_BuildInstancesByType_PinRepoError(t *testing.T) {
	svc, s, instRepo, _ := setupWithInstanceRepos(t)
	ctx := context.Background()

	s.registry.Register(&stubExporter{name: "test-exp", exportOut: &export.ExportOutput{}})

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.bindingRepo.On("GetByID", ctx, "b1").Return(&models.ExportBinding{
		ID: "b1", CatalogID: "cat1", ExporterName: "test-exp",
		Parameters: map[string]string{}, Enabled: true,
	}, nil)
	s.bindingRepo.On("Update", ctx, mock.AnythingOfType("*models.ExportBinding")).Return(nil)

	instRepo.On("ListByCatalog", ctx, "cat1").Return([]*models.EntityInstance{}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return(nil, fmt.Errorf("pin error"))

	_, err := svc.Run(ctx, "my-catalog", "b1", "")
	require.Error(t, err)
}

func TestRun_BuildInstancesByType_ETVRepoError(t *testing.T) {
	svc, s, instRepo, _ := setupWithInstanceRepos(t)
	ctx := context.Background()

	s.registry.Register(&stubExporter{name: "test-exp", exportOut: &export.ExportOutput{}})

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.bindingRepo.On("GetByID", ctx, "b1").Return(&models.ExportBinding{
		ID: "b1", CatalogID: "cat1", ExporterName: "test-exp",
		Parameters: map[string]string{}, Enabled: true,
	}, nil)
	s.bindingRepo.On("Update", ctx, mock.AnythingOfType("*models.ExportBinding")).Return(nil)

	instRepo.On("ListByCatalog", ctx, "cat1").Return([]*models.EntityInstance{}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(nil, fmt.Errorf("etv error"))

	_, err := svc.Run(ctx, "my-catalog", "b1", "")
	require.Error(t, err)
}

func TestRun_BuildInstancesByType_ETRepoError(t *testing.T) {
	svc, s, instRepo, _ := setupWithInstanceRepos(t)
	ctx := context.Background()

	s.registry.Register(&stubExporter{name: "test-exp", exportOut: &export.ExportOutput{}})

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.bindingRepo.On("GetByID", ctx, "b1").Return(&models.ExportBinding{
		ID: "b1", CatalogID: "cat1", ExporterName: "test-exp",
		Parameters: map[string]string{}, Enabled: true,
	}, nil)
	s.bindingRepo.On("Update", ctx, mock.AnythingOfType("*models.ExportBinding")).Return(nil)

	instRepo.On("ListByCatalog", ctx, "cat1").Return([]*models.EntityInstance{}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1"}, nil)
	s.etRepo.On("GetByID", ctx, "et1").Return(nil, fmt.Errorf("et error"))

	_, err := svc.Run(ctx, "my-catalog", "b1", "")
	require.Error(t, err)
}

func TestRun_BuildInstancesByType_AttrRepoError(t *testing.T) {
	svc, s, instRepo, _ := setupWithInstanceRepos(t)
	ctx := context.Background()

	s.registry.Register(&stubExporter{name: "test-exp", exportOut: &export.ExportOutput{}})

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.bindingRepo.On("GetByID", ctx, "b1").Return(&models.ExportBinding{
		ID: "b1", CatalogID: "cat1", ExporterName: "test-exp",
		Parameters: map[string]string{}, Enabled: true,
	}, nil)
	s.bindingRepo.On("Update", ctx, mock.AnythingOfType("*models.ExportBinding")).Return(nil)

	instRepo.On("ListByCatalog", ctx, "cat1").Return([]*models.EntityInstance{}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1"}, nil)
	s.etRepo.On("GetByID", ctx, "et1").Return(&models.EntityType{ID: "et1", Name: "server"}, nil)
	s.attrRepo.On("ListByVersion", ctx, "etv1").Return(nil, fmt.Errorf("attr error"))

	_, err := svc.Run(ctx, "my-catalog", "b1", "")
	require.Error(t, err)
}

func TestRun_BuildInstancesByType_IAVRepoError(t *testing.T) {
	svc, s, instRepo, iavRepo := setupWithInstanceRepos(t)
	ctx := context.Background()

	s.registry.Register(&stubExporter{name: "test-exp", exportOut: &export.ExportOutput{}})

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1",
	}, nil)
	s.bindingRepo.On("GetByID", ctx, "b1").Return(&models.ExportBinding{
		ID: "b1", CatalogID: "cat1", ExporterName: "test-exp",
		Parameters: map[string]string{}, Enabled: true,
	}, nil)
	s.bindingRepo.On("Update", ctx, mock.AnythingOfType("*models.ExportBinding")).Return(nil)

	instRepo.On("ListByCatalog", ctx, "cat1").Return([]*models.EntityInstance{
		{ID: "i1", EntityTypeID: "et1", Name: "server1", Version: 1},
	}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{
		{EntityTypeVersionID: "etv1"},
	}, nil)
	s.etvRepo.On("GetByID", ctx, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1"}, nil)
	s.etRepo.On("GetByID", ctx, "et1").Return(&models.EntityType{ID: "et1", Name: "server"}, nil)
	s.attrRepo.On("ListByVersion", ctx, "etv1").Return([]*models.Attribute{}, nil)
	iavRepo.On("GetValuesForVersion", ctx, "i1", 1).Return(nil, fmt.Errorf("iav error"))

	_, err := svc.Run(ctx, "my-catalog", "b1", "")
	require.Error(t, err)
}

// --- PublishPreview error paths ---

func TestPublishPreview_CatalogNotFound(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "missing").Return(nil, fmt.Errorf("not found"))

	_, err := s.svc.PublishPreview(ctx, "missing")
	require.Error(t, err)
}

func TestPublishPreview_ListBindingsError(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", CatalogVersionID: "cv1",
	}, nil)
	s.bindingRepo.On("ListByCatalog", ctx, "cat1").Return(nil, fmt.Errorf("db error"))

	_, err := s.svc.PublishPreview(ctx, "my-catalog")
	require.Error(t, err)
}

func TestPublishPreview_BuildSchemaError(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", CatalogVersionID: "cv1",
	}, nil)
	s.bindingRepo.On("ListByCatalog", ctx, "cat1").Return([]*models.ExportBinding{}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return(nil, fmt.Errorf("schema error"))

	_, err := s.svc.PublishPreview(ctx, "my-catalog")
	require.Error(t, err)
}

func TestPublishPreview_DisabledBindingSkipped(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	cache := export.NewInMemoryPreviewCache()
	svc := export.NewExportBindingService(
		s.bindingRepo, s.catalogRepo, s.registry,
		s.cvRepo, s.pinRepo, s.etvRepo, s.etRepo,
		s.attrRepo, s.assocRepo,
		export.WithPreviewCache(cache),
	)

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", CatalogVersionID: "cv1",
	}, nil)
	s.bindingRepo.On("ListByCatalog", ctx, "cat1").Return([]*models.ExportBinding{
		{ID: "b1", CatalogID: "cat1", ExporterName: "test", Enabled: false},
	}, nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{}, nil)

	result, err := svc.PublishPreview(ctx, "my-catalog")
	require.NoError(t, err)
	assert.Empty(t, result.Bindings) // disabled binding skipped
}

func TestPublishPreview_NilCache(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.registry.Register(&stubExporter{
		name:      "test-exp",
		exportOut: &export.ExportOutput{Artifacts: []export.K8sArtifact{{Name: "a", YAML: "test: true"}}},
	})

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", CatalogVersionID: "cv1",
	}, nil)
	s.bindingRepo.On("ListByCatalog", ctx, "cat1").Return([]*models.ExportBinding{
		{ID: "b1", CatalogID: "cat1", ExporterName: "test-exp", Parameters: map[string]string{}, Enabled: true},
	}, nil)
	s.bindingRepo.On("Update", ctx, mock.AnythingOfType("*models.ExportBinding")).Return(nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{}, nil)

	// Service has no previewCache — should still succeed, just not store
	result, err := s.svc.PublishPreview(ctx, "my-catalog")
	require.NoError(t, err)
	assert.NotEmpty(t, result.SessionToken)
}

// --- GetCachedArtifacts error paths ---

func TestGetCachedArtifacts_NilCache(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	_, err := s.svc.GetCachedArtifacts(ctx, "my-catalog", "token", "b1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "preview cache not configured")
}

func TestGetCachedArtifacts_WrongCatalog(t *testing.T) {
	cache := export.NewInMemoryPreviewCache()
	s := setupBindingService()
	svc := export.NewExportBindingService(
		s.bindingRepo, s.catalogRepo, s.registry,
		s.cvRepo, s.pinRepo, s.etvRepo, s.etRepo,
		s.attrRepo, s.assocRepo,
		export.WithPreviewCache(cache),
	)

	require.NoError(t, cache.Store("token1", export.PreviewCacheEntry{
		CatalogName: "catalog-A",
		Artifacts:   map[string][]export.K8sArtifact{"b1": {{Name: "a"}}},
	}, 5*60*1e9)) // 5 minutes as nanoseconds

	ctx := context.Background()
	_, err := svc.GetCachedArtifacts(ctx, "catalog-B", "token1", "b1") // wrong catalog
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetCachedArtifacts_BindingNotFound(t *testing.T) {
	cache := export.NewInMemoryPreviewCache()
	s := setupBindingService()
	svc := export.NewExportBindingService(
		s.bindingRepo, s.catalogRepo, s.registry,
		s.cvRepo, s.pinRepo, s.etvRepo, s.etRepo,
		s.attrRepo, s.assocRepo,
		export.WithPreviewCache(cache),
	)

	require.NoError(t, cache.Store("token1", export.PreviewCacheEntry{
		CatalogName: "my-catalog",
		Artifacts:   map[string][]export.K8sArtifact{"b1": {{Name: "a"}}},
	}, 5*60*1e9))

	ctx := context.Background()
	_, err := svc.GetCachedArtifacts(ctx, "my-catalog", "token1", "b-missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// --- GetPreviewEntry error paths ---

func TestGetPreviewEntry_NilCache(t *testing.T) {
	s := setupBindingService()
	_, err := s.svc.GetPreviewEntry("token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "preview cache not configured")
}

// --- getPreviewTTL coverage ---

func TestGetPreviewTTL_Default(t *testing.T) {
	// Without env var, returns 5 min default
	s := setupBindingService()
	cache := export.NewInMemoryPreviewCache()
	svc := export.NewExportBindingService(
		s.bindingRepo, s.catalogRepo, s.registry,
		s.cvRepo, s.pinRepo, s.etvRepo, s.etRepo,
		s.attrRepo, s.assocRepo,
		export.WithPreviewCache(cache),
	)

	s.registry.Register(&stubExporter{
		name:      "test-exp",
		exportOut: &export.ExportOutput{Artifacts: []export.K8sArtifact{{Name: "a", YAML: "test: true"}}},
	})

	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", CatalogVersionID: "cv1",
	}, nil)
	s.bindingRepo.On("ListByCatalog", ctx, "cat1").Return([]*models.ExportBinding{
		{ID: "b1", CatalogID: "cat1", ExporterName: "test-exp", Parameters: map[string]string{}, Enabled: true},
	}, nil)
	s.bindingRepo.On("Update", ctx, mock.AnythingOfType("*models.ExportBinding")).Return(nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{}, nil)

	result, err := svc.PublishPreview(ctx, "my-catalog")
	require.NoError(t, err)
	// Just verify it runs — the TTL is internal
	assert.NotEmpty(t, result.SessionToken)
}

func TestGetPreviewTTL_FromEnv(t *testing.T) {
	t.Setenv("PUBLISH_PREVIEW_TTL", "120")
	s := setupBindingService()
	cache := export.NewInMemoryPreviewCache()
	svc := export.NewExportBindingService(
		s.bindingRepo, s.catalogRepo, s.registry,
		s.cvRepo, s.pinRepo, s.etvRepo, s.etRepo,
		s.attrRepo, s.assocRepo,
		export.WithPreviewCache(cache),
	)

	s.registry.Register(&stubExporter{
		name:      "test-exp",
		exportOut: &export.ExportOutput{Artifacts: []export.K8sArtifact{{Name: "a", YAML: "test: true"}}},
	})

	ctx := context.Background()
	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", CatalogVersionID: "cv1",
	}, nil)
	s.bindingRepo.On("ListByCatalog", ctx, "cat1").Return([]*models.ExportBinding{
		{ID: "b1", CatalogID: "cat1", ExporterName: "test-exp", Parameters: map[string]string{}, Enabled: true},
	}, nil)
	s.bindingRepo.On("Update", ctx, mock.AnythingOfType("*models.ExportBinding")).Return(nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{}, nil)

	result, err := svc.PublishPreview(ctx, "my-catalog")
	require.NoError(t, err)
	assert.NotEmpty(t, result.SessionToken)
}

// --- MCPGatewayExporter.Description coverage ---

func TestMCPGatewayExporter_Description(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	desc := e.Description()
	assert.NotEmpty(t, desc)
	assert.Contains(t, desc, "MCP")
}

// --- resolveTargetName coverage (non-matching path) ---

func TestMCPGateway_ValidateSchema_ServerTypeNotFound(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	schema := export.SchemaInfo{
		EntityTypes: []export.SchemaEntityType{
			{Name: "other-type", Attributes: []string{"route_name"}},
		},
	}
	err := e.ValidateSchema(map[string]string{"server_type": "mcp-server", "tool_type": "mcp-tool"}, schema)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mcp-server")
	assert.Contains(t, err.Error(), "not found")
}

// --- resolveTargetName fallback path ---

func TestMCPGateway_ValidateSchema_ResolveTargetNameFallback(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	// Use target ID that doesn't match any ET name, exercising the fallback path
	schema := export.SchemaInfo{
		EntityTypes: []export.SchemaEntityType{
			{
				Name:       "mcp-server",
				Attributes: []string{"route_name"},
				Associations: []export.SchemaAssociation{
					{Type: "containment", TargetEntityType: "raw-uuid-that-is-not-a-type-name"},
				},
			},
			{Name: "mcp-tool"},
		},
	}
	err := e.ValidateSchema(map[string]string{
		"server_type": "mcp-server",
		"tool_type":   "mcp-tool",
	}, schema)
	// Should fail because resolved target doesn't match tool_type
	require.Error(t, err)
	assert.Contains(t, err.Error(), "containment")
}

// --- GetCachedArtifacts: Retrieve error (token not found) ---

func TestGetCachedArtifacts_TokenNotFound(t *testing.T) {
	cache := export.NewInMemoryPreviewCache()
	s := setupBindingService()
	svc := export.NewExportBindingService(
		s.bindingRepo, s.catalogRepo, s.registry,
		s.cvRepo, s.pinRepo, s.etvRepo, s.etRepo,
		s.attrRepo, s.assocRepo,
		export.WithPreviewCache(cache),
	)
	ctx := context.Background()
	_, err := svc.GetCachedArtifacts(ctx, "my-catalog", "missing-token", "b1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// --- PreviewCache Store error path ---

type failingCache struct{}

func (c *failingCache) Store(_ string, _ export.PreviewCacheEntry, _ time.Duration) error {
	return fmt.Errorf("cache store error")
}
func (c *failingCache) Retrieve(_ string) (*export.PreviewCacheEntry, error) {
	return nil, fmt.Errorf("not found")
}
func (c *failingCache) Delete(_ string) {}

func TestPublishPreview_StoreError(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	svc := export.NewExportBindingService(
		s.bindingRepo, s.catalogRepo, s.registry,
		s.cvRepo, s.pinRepo, s.etvRepo, s.etRepo,
		s.attrRepo, s.assocRepo,
		export.WithPreviewCache(&failingCache{}),
	)

	s.registry.Register(&stubExporter{
		name:      "test-exp",
		exportOut: &export.ExportOutput{Artifacts: []export.K8sArtifact{{Name: "a", YAML: "test: true"}}},
	})

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", CatalogVersionID: "cv1",
	}, nil)
	s.bindingRepo.On("ListByCatalog", ctx, "cat1").Return([]*models.ExportBinding{
		{ID: "b1", CatalogID: "cat1", ExporterName: "test-exp", Parameters: map[string]string{}, Enabled: true},
	}, nil)
	s.bindingRepo.On("Update", ctx, mock.AnythingOfType("*models.ExportBinding")).Return(nil)
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{}, nil)

	_, err := svc.PublishPreview(ctx, "my-catalog")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cache store error")
}

// --- Export default namespace coverage ---

func TestMCPGateway_Export_DefaultNamespace(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	input := export.ExportInput{
		CatalogName: "my-catalog",
		Parameters: map[string]string{
			"server_type": "mcp-server",
			"tool_type":   "mcp-tool",
			// target_namespace NOT set — should default to "default"
		},
		InstancesByType: map[string][]*export.ExportInstance{
			"mcp-server": {{
				ID: "s1", EntityType: "mcp-server", Name: "test-server",
				Attributes: map[string]any{"route_name": "test-route"},
			}},
		},
		ChildrenOf: map[string][]*export.ExportInstance{},
	}

	out, err := e.Export(context.Background(), input)
	require.NoError(t, err)
	assert.Equal(t, "default", out.Artifacts[0].Namespace)
}

// --- Run rejects disabled binding ---

func TestRun_DisabledBinding_ReturnsError(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", CatalogVersionID: "cv1",
	}, nil)
	s.bindingRepo.On("GetByID", ctx, "b1").Return(&models.ExportBinding{
		ID: "b1", CatalogID: "cat1", ExporterName: "mcp-gateway", Enabled: false,
	}, nil)

	_, err := s.svc.Run(ctx, "my-catalog", "b1", "")
	require.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
	assert.Contains(t, err.Error(), "disabled")
}

// I7: Cache cleanup goroutine can be stopped
func TestPreviewCache_Stop(t *testing.T) {
	cache := export.NewInMemoryPreviewCacheWithInterval(10 * time.Millisecond)
	cache.Stop()
	// After Stop, storing and retrieving should still work (cache is usable, just no cleanup)
	err := cache.Store("tok", export.PreviewCacheEntry{CatalogName: "cat"}, time.Hour)
	require.NoError(t, err)
	entry, err := cache.Retrieve("tok")
	require.NoError(t, err)
	assert.Equal(t, "cat", entry.CatalogName)
}

// I6: Binding status constants exist and are used
func TestBindingStatusConstants(t *testing.T) {
	assert.Equal(t, "never", export.BindingStatusNever)
	assert.Equal(t, "success", export.BindingStatusSuccess)
	assert.Equal(t, "failed", export.BindingStatusFailed)
}

// C3: executeBinding exporter-not-found passes domain validation error to updateBindingStatus
func TestExecuteBinding_ExporterNotFound_UsesDomainError(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", CatalogVersionID: "cv1",
	}, nil)
	s.bindingRepo.On("ListByCatalog", ctx, "cat1").Return([]*models.ExportBinding{
		{ID: "b1", CatalogID: "cat1", ExporterName: "nonexistent", Parameters: map[string]string{}, Enabled: true},
	}, nil)

	// Capture the binding passed to Update so we can inspect the error stored on it
	var capturedBinding *models.ExportBinding
	s.bindingRepo.On("Update", ctx, mock.AnythingOfType("*models.ExportBinding")).Run(func(args mock.Arguments) {
		capturedBinding = args.Get(1).(*models.ExportBinding)
	}).Return(nil)

	results, err := s.svc.RunAll(ctx, "my-catalog")
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "failed", results[0].Status)
	assert.Contains(t, results[0].Error, "nonexistent")

	// The error stored on the binding should be a VALIDATION prefix, not a raw fmt.Errorf
	require.NotNil(t, capturedBinding)
	assert.Equal(t, "failed", capturedBinding.LastRunStatus)
	assert.Contains(t, capturedBinding.LastRunError, "VALIDATION")
}

// C4: Concurrent Retrieve on expired entry should not race
func TestPreviewCache_ConcurrentRetrieveExpired(t *testing.T) {
	cache := export.NewInMemoryPreviewCacheWithInterval(time.Hour)
	// Store an already-expired entry
	err := cache.Store("token1", export.PreviewCacheEntry{CatalogName: "cat1"}, -1*time.Second)
	require.NoError(t, err)

	// Concurrent retrieves on the expired entry — should not panic or race
	done := make(chan error, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := cache.Retrieve("token1")
			done <- err
		}()
	}
	for i := 0; i < 10; i++ {
		err := <-done
		assert.Error(t, err)
	}
}

// T-34.75i: Run with non-existent VS instance returns NotFound
func TestRun_VSInstanceNotFound(t *testing.T) {
	s := setupBindingService()
	ctx := context.Background()

	s.catalogRepo.On("GetByName", ctx, "my-catalog").Return(&models.Catalog{
		ID: "cat1", CatalogVersionID: "cv1",
	}, nil)
	s.bindingRepo.On("GetByID", ctx, "b1").Return(&models.ExportBinding{
		ID: "b1", CatalogID: "cat1", ExporterName: "mcp-gateway",
		Parameters: map[string]string{"virtual_server_type": "virtual-server"}, Enabled: true,
	}, nil)

	s.registry.Register(&stubExporter{name: "mcp-gateway"})

	instRepo := new(mocks.MockEntityInstanceRepo)
	linkRepo := new(mocks.MockAssociationLinkRepo)
	iavRepo := new(mocks.MockInstanceAttributeValueRepo)

	svc := export.NewExportBindingService(
		s.bindingRepo, s.catalogRepo, s.registry,
		s.cvRepo, s.pinRepo, s.etvRepo, s.etRepo,
		s.attrRepo, s.assocRepo,
		export.WithInstanceRepos(instRepo, iavRepo),
		export.WithLinkRepo(linkRepo),
	)

	// buildExportInput needs these
	s.pinRepo.On("ListByCatalogVersion", ctx, "cv1").Return([]*models.CatalogVersionPin{}, nil)
	// resolveVSInstanceTools needs these
	instRepo.On("ListByCatalog", ctx, "cat1").Return([]*models.EntityInstance{}, nil)
	s.bindingRepo.On("Update", ctx, mock.AnythingOfType("*models.ExportBinding")).Return(nil)

	_, err := svc.Run(ctx, "my-catalog", "b1", "nonexistent-vs")
	require.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
	assert.Contains(t, err.Error(), "nonexistent-vs")
}

// --- ValidateSchema returns domain validation errors, not generic errors ---

func TestMCPGateway_ValidateSchema_ReturnsValidationError(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	schema := export.SchemaInfo{
		EntityTypes: []export.SchemaEntityType{{Name: "server"}},
	}
	err := e.ValidateSchema(map[string]string{"server_type": "nonexistent", "tool_type": "tool"}, schema)
	require.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err), "ValidateSchema error should be a domain validation error, got: %T", err)
}

// --- ParameterSchema: entity_type params use correct type ---

func TestMCPGateway_ParameterSchema_EntityTypeParams(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	schema := e.ParameterSchema()
	for _, p := range schema {
		if p.Name == "server_type" || p.Name == "tool_type" || p.Name == "virtual_server_type" {
			assert.Equal(t, "entity_type", p.Type, "parameter %q should have type entity_type", p.Name)
		}
		if p.Name == "target_namespace" {
			assert.Equal(t, "string", p.Type, "parameter %q should remain type string", p.Name)
		}
	}
}

// T-34.75a: virtual_server_type parameter is required and type entity_type
func TestMCPGateway_ParameterSchema_HasVirtualServerType(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	schema := e.ParameterSchema()
	var found bool
	for _, p := range schema {
		if p.Name == "virtual_server_type" {
			found = true
			assert.Equal(t, "entity_type", p.Type)
			assert.True(t, p.Required)
			break
		}
	}
	assert.True(t, found, "virtual_server_type parameter must exist")
}

// T-34.12b: ValidateSchema checks virtual_server_type entity type exists in CV
func TestMCPGateway_ValidateSchema_VSTypeNotInCV(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	schema := export.SchemaInfo{
		EntityTypes: []export.SchemaEntityType{
			{
				Name:       "mcp-server",
				Attributes: []string{"route_name"},
				Associations: []export.SchemaAssociation{
					{Type: "containment", TargetEntityType: "mcp-tool"},
				},
			},
			{Name: "mcp-tool"},
			// virtual-server NOT in schema
		},
	}
	err := e.ValidateSchema(map[string]string{
		"server_type":         "mcp-server",
		"tool_type":           "mcp-tool",
		"virtual_server_type": "virtual-server",
	}, schema)
	require.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
	assert.Contains(t, err.Error(), "virtual-server")
	assert.Contains(t, err.Error(), "not found")
}

// T-34.75b/c: ValidateSchema checks virtual_server_type has association to tool_type
func TestMCPGateway_ValidateSchema_VSTypeNoAssocToTool(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	schema := export.SchemaInfo{
		EntityTypes: []export.SchemaEntityType{
			{
				Name:       "mcp-server",
				Attributes: []string{"route_name"},
				Associations: []export.SchemaAssociation{
					{Type: "containment", TargetEntityType: "mcp-tool"},
				},
			},
			{Name: "mcp-tool"},
			{Name: "virtual-server"}, // no association to mcp-tool
		},
	}
	err := e.ValidateSchema(map[string]string{
		"server_type":         "mcp-server",
		"tool_type":           "mcp-tool",
		"virtual_server_type": "virtual-server",
	}, schema)
	require.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
	assert.Contains(t, err.Error(), "virtual-server")
	assert.Contains(t, err.Error(), "mcp-tool")
}

func TestMCPGateway_ValidateSchema_VSTypeWithAssocToTool(t *testing.T) {
	e := export.NewMCPGatewayExporter()
	schema := export.SchemaInfo{
		EntityTypes: []export.SchemaEntityType{
			{
				Name:       "mcp-server",
				Attributes: []string{"route_name"},
				Associations: []export.SchemaAssociation{
					{Type: "containment", TargetEntityType: "mcp-tool"},
				},
			},
			{Name: "mcp-tool"},
			{
				Name: "virtual-server",
				Associations: []export.SchemaAssociation{
					{Type: "directional", TargetEntityType: "mcp-tool"},
				},
			},
		},
	}
	err := e.ValidateSchema(map[string]string{
		"server_type":         "mcp-server",
		"tool_type":           "mcp-tool",
		"virtual_server_type": "virtual-server",
	}, schema)
	require.NoError(t, err)
}

// --- cleanupLoop: expired entries removed by background goroutine ---

func TestInMemoryPreviewCache_CleanupLoop_RemovesExpired(t *testing.T) {
	cache := export.NewInMemoryPreviewCacheWithInterval(10 * time.Millisecond)

	// Store an already-expired entry
	err := cache.Store("expired-token", export.PreviewCacheEntry{CatalogName: "cat1"}, -1*time.Second)
	require.NoError(t, err)

	// Store a valid entry
	err = cache.Store("valid-token", export.PreviewCacheEntry{CatalogName: "cat2"}, 1*time.Hour)
	require.NoError(t, err)

	// Wait for cleanup to fire
	time.Sleep(50 * time.Millisecond)

	// Expired entry should be removed
	_, err = cache.Retrieve("expired-token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Valid entry should still exist
	entry, err := cache.Retrieve("valid-token")
	require.NoError(t, err)
	assert.Equal(t, "cat2", entry.CatalogName)
}
