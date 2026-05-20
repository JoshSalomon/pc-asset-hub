package export

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository"
)

type ExportBindingService struct {
	bindingRepo  repository.ExportBindingRepository
	catalogRepo  repository.CatalogRepository
	registry     *ExporterRegistry
	cvRepo       repository.CatalogVersionRepository
	pinRepo      repository.CatalogVersionPinRepository
	etvRepo      repository.EntityTypeVersionRepository
	etRepo       repository.EntityTypeRepository
	attrRepo     repository.AttributeRepository
	assocRepo    repository.AssociationRepository
	instRepo     repository.EntityInstanceRepository
	iavRepo      repository.InstanceAttributeValueRepository
	linkRepo     repository.AssociationLinkRepository
	previewCache PreviewCache
}

func NewExportBindingService(
	bindingRepo repository.ExportBindingRepository,
	catalogRepo repository.CatalogRepository,
	registry *ExporterRegistry,
	cvRepo repository.CatalogVersionRepository,
	pinRepo repository.CatalogVersionPinRepository,
	etvRepo repository.EntityTypeVersionRepository,
	etRepo repository.EntityTypeRepository,
	attrRepo repository.AttributeRepository,
	assocRepo repository.AssociationRepository,
	opts ...ExportBindingServiceOption,
) *ExportBindingService {
	s := &ExportBindingService{
		bindingRepo: bindingRepo,
		catalogRepo: catalogRepo,
		registry:    registry,
		cvRepo:      cvRepo,
		pinRepo:     pinRepo,
		etvRepo:     etvRepo,
		etRepo:      etRepo,
		attrRepo:    attrRepo,
		assocRepo:   assocRepo,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

type ExportBindingServiceOption func(*ExportBindingService)

func WithInstanceRepos(instRepo repository.EntityInstanceRepository, iavRepo repository.InstanceAttributeValueRepository) ExportBindingServiceOption {
	return func(s *ExportBindingService) {
		s.instRepo = instRepo
		s.iavRepo = iavRepo
	}
}

func WithLinkRepo(linkRepo repository.AssociationLinkRepository) ExportBindingServiceOption {
	return func(s *ExportBindingService) {
		s.linkRepo = linkRepo
	}
}

func WithPreviewCache(cache PreviewCache) ExportBindingServiceOption {
	return func(s *ExportBindingService) {
		s.previewCache = cache
	}
}

func (s *ExportBindingService) getBindingForCatalog(ctx context.Context, catalogName, bindingID string) (*models.Catalog, *models.ExportBinding, error) {
	catalog, err := s.catalogRepo.GetByName(ctx, catalogName)
	if err != nil {
		return nil, nil, err
	}
	binding, err := s.bindingRepo.GetByID(ctx, bindingID)
	if err != nil {
		return nil, nil, err
	}
	if binding.CatalogID != catalog.ID {
		return nil, nil, domainerrors.NewNotFound("ExportBinding", bindingID)
	}
	return catalog, binding, nil
}

func (s *ExportBindingService) validateBindingParams(ctx context.Context, exporterName string, params map[string]string, cvID string) error {
	exporter, ok := s.registry.Get(exporterName)
	if !ok {
		return domainerrors.NewValidation(fmt.Sprintf("exporter %q not found", exporterName))
	}

	if err := s.validateRequiredParams(exporter, params); err != nil {
		return err
	}

	schema, err := s.buildSchemaInfo(ctx, cvID)
	if err != nil {
		return err
	}

	if err := s.validateParamEntityTypes(params, schema); err != nil {
		return err
	}

	return exporter.ValidateSchema(params, schema)
}

func (s *ExportBindingService) Create(ctx context.Context, catalogName, exporterName string, params map[string]string) (*models.ExportBinding, error) {
	catalog, err := s.catalogRepo.GetByName(ctx, catalogName)
	if err != nil {
		return nil, err
	}

	if err := s.validateBindingParams(ctx, exporterName, params, catalog.CatalogVersionID); err != nil {
		return nil, err
	}

	binding := &models.ExportBinding{
		ID:            uuid.Must(uuid.NewV7()).String(),
		CatalogID:     catalog.ID,
		ExporterName:  exporterName,
		Parameters:    params,
		Enabled:       true,
		LastRunStatus: BindingStatusNever,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.bindingRepo.Create(ctx, binding); err != nil {
		return nil, err
	}
	return binding, nil
}

func (s *ExportBindingService) List(ctx context.Context, catalogName string) ([]*models.ExportBinding, error) {
	catalog, err := s.catalogRepo.GetByName(ctx, catalogName)
	if err != nil {
		return nil, err
	}
	return s.bindingRepo.ListByCatalog(ctx, catalog.ID)
}

func (s *ExportBindingService) Get(ctx context.Context, catalogName, bindingID string) (*models.ExportBinding, error) {
	_, binding, err := s.getBindingForCatalog(ctx, catalogName, bindingID)
	return binding, err
}

func (s *ExportBindingService) Update(ctx context.Context, catalogName, bindingID string, params map[string]string, enabled *bool) (*models.ExportBinding, error) {
	catalog, binding, err := s.getBindingForCatalog(ctx, catalogName, bindingID)
	if err != nil {
		return nil, err
	}

	if params != nil {
		if err := s.validateBindingParams(ctx, binding.ExporterName, params, catalog.CatalogVersionID); err != nil {
			return nil, err
		}
		binding.Parameters = params
	}
	if enabled != nil {
		binding.Enabled = *enabled
	}
	binding.UpdatedAt = time.Now()

	if err := s.bindingRepo.Update(ctx, binding); err != nil {
		return nil, err
	}
	return binding, nil
}

func (s *ExportBindingService) Delete(ctx context.Context, catalogName, bindingID string) error {
	_, _, err := s.getBindingForCatalog(ctx, catalogName, bindingID)
	if err != nil {
		return err
	}
	return s.bindingRepo.Delete(ctx, bindingID)
}

func (s *ExportBindingService) Run(ctx context.Context, catalogName, bindingID, vsInstanceName string) (*ExportOutput, error) {
	catalog, binding, err := s.getBindingForCatalog(ctx, catalogName, bindingID)
	if err != nil {
		return nil, err
	}

	if !binding.Enabled {
		return nil, domainerrors.NewValidation("cannot run export on a disabled binding")
	}

	if binding.Parameters["virtual_server_type"] != "" && vsInstanceName == "" {
		return nil, domainerrors.NewValidation("virtual_server_instance is required when binding has virtual_server_type parameter")
	}

	exporter, ok := s.registry.Get(binding.ExporterName)
	if !ok {
		err := domainerrors.NewValidation(fmt.Sprintf("exporter %q not found", binding.ExporterName))
		s.updateBindingStatus(ctx, binding, err)
		return nil, err
	}

	input, err := s.buildExportInput(ctx, catalog, binding)
	if err != nil {
		s.updateBindingStatus(ctx, binding, err)
		return nil, err
	}

	if vsInstanceName != "" {
		allowedToolIDs, vsErr := s.resolveVSInstanceTools(ctx, catalog, vsInstanceName, binding.Parameters["virtual_server_type"])
		if vsErr != nil {
			s.updateBindingStatus(ctx, binding, vsErr)
			return nil, vsErr
		}
		input.VirtualServerInstanceName = vsInstanceName
		input.AllowedToolIDs = allowedToolIDs
	}

	output, err := exporter.Export(ctx, input)
	s.updateBindingStatus(ctx, binding, err)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func (s *ExportBindingService) resolveVSInstanceTools(ctx context.Context, catalog *models.Catalog, vsInstanceName, vsTypeName string) (map[string]bool, error) {
	if s.instRepo == nil || s.linkRepo == nil {
		return nil, domainerrors.NewValidation("instance and link repos required for VS instance selection")
	}

	instances, err := s.instRepo.ListByCatalog(ctx, catalog.ID)
	if err != nil {
		return nil, err
	}

	var vsInstance *models.EntityInstance
	for _, inst := range instances {
		if inst.Name == vsInstanceName {
			vsInstance = inst
			break
		}
	}
	if vsInstance == nil {
		return nil, domainerrors.NewNotFound("VirtualServer instance", vsInstanceName)
	}

	links, err := s.linkRepo.GetForwardRefs(ctx, vsInstance.ID)
	if err != nil {
		return nil, err
	}

	allowed := make(map[string]bool)
	for _, link := range links {
		allowed[link.TargetInstanceID] = true
	}
	return allowed, nil
}

func (s *ExportBindingService) RunAll(ctx context.Context, catalogName string) ([]BindingRunResult, error) {
	catalog, err := s.catalogRepo.GetByName(ctx, catalogName)
	if err != nil {
		return nil, err
	}
	bindings, err := s.bindingRepo.ListByCatalog(ctx, catalog.ID)
	if err != nil {
		return nil, err
	}

	var results []BindingRunResult
	for _, binding := range bindings {
		if !binding.Enabled {
			continue
		}
		results = append(results, s.executeBinding(ctx, catalog, binding))
	}
	return results, nil
}

func (s *ExportBindingService) executeBinding(ctx context.Context, catalog *models.Catalog, binding *models.ExportBinding) BindingRunResult {
	result := BindingRunResult{
		BindingID:    binding.ID,
		ExporterName: binding.ExporterName,
	}

	exporter, ok := s.registry.Get(binding.ExporterName)
	if !ok {
		result.Status = BindingStatusFailed
		result.Error = fmt.Sprintf("exporter %q not found", binding.ExporterName)
		s.updateBindingStatus(ctx, binding, domainerrors.NewValidation(result.Error))
		return result
	}

	input, err := s.buildExportInput(ctx, catalog, binding)
	if err != nil {
		result.Status = BindingStatusFailed
		result.Error = err.Error()
		s.updateBindingStatus(ctx, binding, err)
		return result
	}

	output, err := exporter.Export(ctx, input)
	if err != nil {
		result.Status = BindingStatusFailed
		result.Error = err.Error()
		s.updateBindingStatus(ctx, binding, err)
		return result
	}

	result.Status = BindingStatusSuccess
	result.ArtifactCount = len(output.Artifacts)
	result.Artifacts = output.Artifacts
	s.updateBindingStatus(ctx, binding, nil)
	return result
}

func (s *ExportBindingService) updateBindingStatus(ctx context.Context, binding *models.ExportBinding, execErr error) {
	now := time.Now()
	binding.LastRunAt = &now
	if execErr != nil {
		binding.LastRunStatus = BindingStatusFailed
		binding.LastRunError = execErr.Error()
	} else {
		binding.LastRunStatus = BindingStatusSuccess
		binding.LastRunError = ""
	}
	_ = s.bindingRepo.Update(ctx, binding)
}

type BindingRunResult struct {
	BindingID     string
	ExporterName  string
	Status        string
	Error         string
	ArtifactCount int
	Artifacts     []K8sArtifact
}

func (s *ExportBindingService) BuildSchemaInfo(ctx context.Context, cvID string) (SchemaInfo, error) {
	return s.buildSchemaInfo(ctx, cvID)
}

func (s *ExportBindingService) validateRequiredParams(exporter Exporter, params map[string]string) error {
	schema := exporter.ParameterSchema()
	var missing []string
	for _, p := range schema {
		if p.Required {
			if val, ok := params[p.Name]; !ok || val == "" {
				missing = append(missing, p.Name)
			}
		}
	}
	if len(missing) > 0 {
		return domainerrors.NewValidation(fmt.Sprintf("missing required parameters: %s", strings.Join(missing, ", ")))
	}
	return nil
}

func (s *ExportBindingService) validateParamEntityTypes(params map[string]string, schema SchemaInfo) error {
	typeNames := map[string]bool{}
	for _, et := range schema.EntityTypes {
		typeNames[et.Name] = true
	}
	for key, val := range params {
		if strings.HasSuffix(key, "_type") && val != "" {
			if !typeNames[val] {
				return domainerrors.NewValidation(fmt.Sprintf("entity type %q (parameter %q) is not pinned in the catalog version", val, key))
			}
		}
	}
	return nil
}

type resolvedPin struct {
	etvID  string
	etID   string
	etName string
}

func (s *ExportBindingService) resolvePins(ctx context.Context, cvID string) ([]resolvedPin, map[string]string, error) {
	pins, err := s.pinRepo.ListByCatalogVersion(ctx, cvID)
	if err != nil {
		return nil, nil, err
	}
	etIDToName := map[string]string{}
	var resolved []resolvedPin
	for _, pin := range pins {
		etv, err := s.etvRepo.GetByID(ctx, pin.EntityTypeVersionID)
		if err != nil {
			return nil, nil, err
		}
		et, err := s.etRepo.GetByID(ctx, etv.EntityTypeID)
		if err != nil {
			return nil, nil, err
		}
		etIDToName[et.ID] = et.Name
		resolved = append(resolved, resolvedPin{etvID: pin.EntityTypeVersionID, etID: et.ID, etName: et.Name})
	}
	return resolved, etIDToName, nil
}

func (s *ExportBindingService) buildSchemaInfo(ctx context.Context, cvID string) (SchemaInfo, error) {
	pinInfos, etIDToName, err := s.resolvePins(ctx, cvID)
	if err != nil {
		return SchemaInfo{}, err
	}

	var entityTypes []SchemaEntityType
	for _, pi := range pinInfos {
		attrs, err := s.attrRepo.ListByVersion(ctx, pi.etvID)
		if err != nil {
			return SchemaInfo{}, err
		}
		attrNames := make([]string, len(attrs))
		for i, a := range attrs {
			attrNames[i] = a.Name
		}

		assocs, err := s.assocRepo.ListByVersion(ctx, pi.etvID)
		if err != nil {
			return SchemaInfo{}, err
		}
		schemaAssocs := make([]SchemaAssociation, len(assocs))
		for i, a := range assocs {
			targetName := a.TargetEntityTypeID
			if name, ok := etIDToName[a.TargetEntityTypeID]; ok {
				targetName = name
			}
			schemaAssocs[i] = SchemaAssociation{
				Name:             a.Name,
				Type:             string(a.Type),
				TargetEntityType: targetName,
			}
		}

		entityTypes = append(entityTypes, SchemaEntityType{
			Name:         pi.etName,
			Attributes:   attrNames,
			Associations: schemaAssocs,
		})
	}

	return SchemaInfo{EntityTypes: entityTypes}, nil
}

func (s *ExportBindingService) buildExportInput(ctx context.Context, catalog *models.Catalog, binding *models.ExportBinding) (ExportInput, error) {
	instances, err := s.buildInstancesByType(ctx, catalog)
	if err != nil {
		return ExportInput{}, err
	}

	return ExportInput{
		CatalogName:     catalog.Name,
		CatalogDesc:     catalog.Description,
		Parameters:      binding.Parameters,
		InstancesByType: instances.byType,
		ChildrenOf:      instances.childrenOf,
	}, nil
}

type instanceData struct {
	byType     map[string][]*ExportInstance
	childrenOf map[string][]*ExportInstance
}

func (s *ExportBindingService) buildInstancesByType(ctx context.Context, catalog *models.Catalog) (*instanceData, error) {
	if s.instRepo == nil {
		return &instanceData{
			byType:     make(map[string][]*ExportInstance),
			childrenOf: make(map[string][]*ExportInstance),
		}, nil
	}

	instances, err := s.instRepo.ListByCatalog(ctx, catalog.ID)
	if err != nil {
		return nil, err
	}

	resolved, etIDToName, err := s.resolvePins(ctx, catalog.CatalogVersionID)
	if err != nil {
		return nil, err
	}

	etvIDForET := map[string]string{}
	for _, rp := range resolved {
		etvIDForET[rp.etID] = rp.etvID
	}

	attrsByETV := map[string][]*models.Attribute{}
	for _, etvID := range etvIDForET {
		attrs, err := s.attrRepo.ListByVersion(ctx, etvID)
		if err != nil {
			return nil, err
		}
		attrsByETV[etvID] = attrs
	}

	data := &instanceData{
		byType:     make(map[string][]*ExportInstance),
		childrenOf: make(map[string][]*ExportInstance),
	}

	for _, inst := range instances {
		etName := etIDToName[inst.EntityTypeID]
		etvID := etvIDForET[inst.EntityTypeID]
		attrs := attrsByETV[etvID]

		attrValues := map[string]any{}
		if s.iavRepo != nil {
			vals, err := s.iavRepo.GetValuesForVersion(ctx, inst.ID, inst.Version)
			if err != nil {
				return nil, err
			}
			for _, v := range vals {
				for _, a := range attrs {
					if a.ID == v.AttributeID {
						if v.ValueString != "" {
							attrValues[a.Name] = v.ValueString
						} else if v.ValueJSON != "" {
							attrValues[a.Name] = v.ValueJSON
						}
						break
					}
				}
			}
		}

		ei := &ExportInstance{
			ID:          inst.ID,
			EntityType:  etName,
			Name:        inst.Name,
			Description: inst.Description,
			ParentID:    inst.ParentInstanceID,
			Attributes:  attrValues,
		}

		data.byType[etName] = append(data.byType[etName], ei)
		if inst.ParentInstanceID != "" {
			data.childrenOf[inst.ParentInstanceID] = append(data.childrenOf[inst.ParentInstanceID], ei)
		}
	}

	return data, nil
}
