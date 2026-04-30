package operational

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository"
)

// === Export file format types ===

type ExportData struct {
	FormatVersion   string              `json:"format_version"`
	ExportedAt      time.Time           `json:"exported_at"`
	SourceSystem    string              `json:"source_system"`
	Catalog         ExportCatalog       `json:"catalog"`
	CatalogVersion  ExportCatalogVersion `json:"catalog_version"`
	TypeDefinitions []ExportTypeDef     `json:"type_definitions"`
	EntityTypes     []ExportEntityType  `json:"entity_types"`
	Instances       []ExportInstance    `json:"instances"`
}

type ExportCatalog struct {
	Name             string `json:"name"`
	Description      string `json:"description"`
	ValidationStatus string `json:"validation_status"`
}

type ExportCatalogVersion struct {
	Label       string `json:"label"`
	Description string `json:"description"`
}

type ExportTypeDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	BaseType    string         `json:"base_type"`
	System      bool           `json:"system"`
	Constraints map[string]any `json:"constraints"`
}

type ExportEntityType struct {
	Name         string              `json:"name"`
	Description  string              `json:"description"`
	Attributes   []ExportAttribute   `json:"attributes"`
	Associations []ExportAssociation `json:"associations"`
}

type ExportAttribute struct {
	Name           string `json:"name"`
	TypeDefinition string `json:"type_definition"`
	Required       bool   `json:"required"`
	Ordinal        int    `json:"ordinal"`
	Description    string `json:"description"`
}

type ExportAssociation struct {
	Name              string `json:"name"`
	Type              string `json:"type"`
	Target            string `json:"target"`
	SourceCardinality string `json:"source_cardinality"`
	TargetCardinality string `json:"target_cardinality"`
	SourceRole        string `json:"source_role"`
	TargetRole        string `json:"target_role"`
}

type ExportLink struct {
	Association string   `json:"association"`
	TargetType  string   `json:"target_type"`
	TargetName  string   `json:"target_name"`
	TargetPath  []string `json:"target_path,omitempty"`
}

type ExportInstance struct {
	EntityType  string            `json:"entity_type"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Attributes  map[string]any    `json:"attributes"`
	Links       []ExportLink      `json:"links,omitempty"`
	Children    map[string][]*ExportInstance `json:"-"`
}

func (ei ExportInstance) MarshalJSON() ([]byte, error) {
	type plain struct {
		EntityType  string         `json:"entity_type"`
		Name        string         `json:"name"`
		Description string         `json:"description"`
		Attributes  map[string]any `json:"attributes"`
		Links       []ExportLink   `json:"links,omitempty"`
	}
	m := make(map[string]any)
	base := plain{
		EntityType:  ei.EntityType,
		Name:        ei.Name,
		Description: ei.Description,
		Attributes:  ei.Attributes,
		Links:       ei.Links,
	}
	b, err := json.Marshal(base)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	for assocName, children := range ei.Children {
		m[assocName] = children
	}
	return json.Marshal(m)
}

func (ed *ExportData) UnmarshalJSON(b []byte) error {
	// First pass: unmarshal everything except instances (which need custom parsing for children).
	type Alias ExportData
	var raw struct {
		Alias
		Instances json.RawMessage `json:"instances"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	*ed = ExportData(raw.Alias)

	if len(raw.Instances) == 0 {
		return nil
	}

	var instanceRaws []json.RawMessage
	if err := json.Unmarshal(raw.Instances, &instanceRaws); err != nil {
		return err
	}

	instances, err := ParseExportInstances(instanceRaws, ed.EntityTypes)
	if err != nil {
		return err
	}
	ed.Instances = instances
	return nil
}

// === Export Service ===

type ExportService struct {
	catalogRepo repository.CatalogRepository
	cvRepo      repository.CatalogVersionRepository
	pinRepo     repository.CatalogVersionPinRepository
	etRepo      repository.EntityTypeRepository
	etvRepo     repository.EntityTypeVersionRepository
	attrRepo    repository.AttributeRepository
	assocRepo   repository.AssociationRepository
	tdRepo      repository.TypeDefinitionRepository
	tdvRepo     repository.TypeDefinitionVersionRepository
	instRepo    repository.EntityInstanceRepository
	iavRepo     repository.InstanceAttributeValueRepository
	linkRepo    repository.AssociationLinkRepository
}

func NewExportService(
	catalogRepo repository.CatalogRepository,
	cvRepo repository.CatalogVersionRepository,
	pinRepo repository.CatalogVersionPinRepository,
	etRepo repository.EntityTypeRepository,
	etvRepo repository.EntityTypeVersionRepository,
	attrRepo repository.AttributeRepository,
	assocRepo repository.AssociationRepository,
	tdRepo repository.TypeDefinitionRepository,
	tdvRepo repository.TypeDefinitionVersionRepository,
	instRepo repository.EntityInstanceRepository,
	iavRepo repository.InstanceAttributeValueRepository,
	linkRepo repository.AssociationLinkRepository,
) *ExportService {
	return &ExportService{
		catalogRepo: catalogRepo,
		cvRepo:      cvRepo,
		pinRepo:     pinRepo,
		etRepo:      etRepo,
		etvRepo:     etvRepo,
		attrRepo:    attrRepo,
		assocRepo:   assocRepo,
		tdRepo:      tdRepo,
		tdvRepo:     tdvRepo,
		instRepo:    instRepo,
		iavRepo:     iavRepo,
		linkRepo:    linkRepo,
	}
}

func (s *ExportService) ExportCatalog(ctx context.Context, catalogName string, entityFilter []string, sourceSystem string) (*ExportData, error) {
	catalog, err := s.catalogRepo.GetByName(ctx, catalogName)
	if err != nil {
		return nil, err
	}

	cv, err := s.cvRepo.GetByID(ctx, catalog.CatalogVersionID)
	if err != nil {
		return nil, err
	}

	pins, err := s.pinRepo.ListByCatalogVersion(ctx, catalog.CatalogVersionID)
	if err != nil {
		return nil, err
	}

	// Build entity filter set
	filterSet := make(map[string]bool)
	for _, name := range entityFilter {
		filterSet[name] = true
	}

	// Collect type definition IDs to export (custom only)
	tdvIDSet := make(map[string]bool)
	var exportEntityTypes []ExportEntityType

	for _, pin := range pins {
		etv, err := s.etvRepo.GetByID(ctx, pin.EntityTypeVersionID)
		if err != nil {
			return nil, err
		}

		et, err := s.etRepo.GetByID(ctx, etv.EntityTypeID)
		if err != nil {
			return nil, err
		}

		if len(filterSet) > 0 && !filterSet[et.Name] {
			continue
		}

		attrs, err := s.attrRepo.ListByVersion(ctx, etv.ID)
		if err != nil {
			return nil, err
		}

		var exportAttrs []ExportAttribute
		for _, attr := range attrs {
			tdv, err := s.tdvRepo.GetByID(ctx, attr.TypeDefinitionVersionID)
			if err != nil {
				return nil, err
			}
			td, err := s.tdRepo.GetByID(ctx, tdv.TypeDefinitionID)
			if err != nil {
				return nil, err
			}
			if !td.System {
				tdvIDSet[attr.TypeDefinitionVersionID] = true
			}
			exportAttrs = append(exportAttrs, ExportAttribute{
				Name:           attr.Name,
				TypeDefinition: td.Name,
				Required:       attr.Required,
				Ordinal:        attr.Ordinal,
				Description:    attr.Description,
			})
		}

		assocs, err := s.assocRepo.ListByVersion(ctx, etv.ID)
		if err != nil {
			return nil, err
		}

		var exportAssocs []ExportAssociation
		for _, assoc := range assocs {
			targetET, err := s.etRepo.GetByID(ctx, assoc.TargetEntityTypeID)
			if err != nil {
				return nil, err
			}
			if len(filterSet) > 0 && !filterSet[targetET.Name] {
				continue
			}
			exportAssocs = append(exportAssocs, ExportAssociation{
				Name:              assoc.Name,
				Type:              string(assoc.Type),
				Target:            targetET.Name,
				SourceCardinality: assoc.SourceCardinality,
				TargetCardinality: assoc.TargetCardinality,
				SourceRole:        assoc.SourceRole,
				TargetRole:        assoc.TargetRole,
			})
		}

		exportEntityTypes = append(exportEntityTypes, ExportEntityType{
			Name:         et.Name,
			Description:  etv.Description,
			Attributes:   exportAttrs,
			Associations: exportAssocs,
		})
	}

	// Build type definitions list (custom types referenced by attributes)
	var exportTypeDefs []ExportTypeDef
	for tdvID := range tdvIDSet {
		tdv, err := s.tdvRepo.GetByID(ctx, tdvID)
		if err != nil {
			return nil, err
		}
		td, err := s.tdRepo.GetByID(ctx, tdv.TypeDefinitionID)
		if err != nil {
			return nil, err
		}
		exportTypeDefs = append(exportTypeDefs, ExportTypeDef{
			Name:        td.Name,
			Description: td.Description,
			BaseType:    string(td.BaseType),
			System:      td.System,
			Constraints: tdv.Constraints,
		})
	}

	// Build instances
	allInstances, err := s.instRepo.ListByCatalog(ctx, catalog.ID)
	if err != nil {
		return nil, err
	}

	// Build entity type name cache and entity type ID → ETV ID mapping for attribute resolution
	etNameByID := make(map[string]string)
	etvByETID := make(map[string]*models.EntityTypeVersion)
	for _, pin := range pins {
		etv, err := s.etvRepo.GetByID(ctx, pin.EntityTypeVersionID)
		if err != nil {
			return nil, err
		}
		et, err := s.etRepo.GetByID(ctx, etv.EntityTypeID)
		if err != nil {
			return nil, err
		}
		etNameByID[et.ID] = et.Name
		etvByETID[etv.EntityTypeID] = etv
	}

	// Build instance map for containment tree + link resolution
	instanceByID := make(map[string]*models.EntityInstance)
	for _, inst := range allInstances {
		instanceByID[inst.ID] = inst
	}

	// Resolve containment association names: parent ET → association name → child ET
	assocNameByParentChild := make(map[string]map[string]string) // parentETID → childETID → assocName
	for _, pin := range pins {
		etv, err := s.etvRepo.GetByID(ctx, pin.EntityTypeVersionID)
		if err != nil {
			return nil, err
		}
		assocs, err := s.assocRepo.ListByVersion(ctx, etv.ID)
		if err != nil {
			return nil, err
		}
		for _, assoc := range assocs {
			if assoc.Type == models.AssociationTypeContainment {
				if assocNameByParentChild[etv.EntityTypeID] == nil {
					assocNameByParentChild[etv.EntityTypeID] = make(map[string]string)
				}
				assocNameByParentChild[etv.EntityTypeID][assoc.TargetEntityTypeID] = assoc.Name
			}
		}
	}

	// Resolve association ID → name and target entity type name
	assocByID := make(map[string]*models.Association)
	for _, pin := range pins {
		etv, err := s.etvRepo.GetByID(ctx, pin.EntityTypeVersionID)
		if err != nil {
			return nil, err
		}
		assocs, err := s.assocRepo.ListByVersion(ctx, etv.ID)
		if err != nil {
			return nil, err
		}
		for _, a := range assocs {
			assocByID[a.ID] = a
		}
	}

	// Build export instances recursively
	filteredETIDs := make(map[string]bool)
	for _, eet := range exportEntityTypes {
		for id, name := range etNameByID {
			if name == eet.Name {
				filteredETIDs[id] = true
			}
		}
	}

	exportInstances, err := s.buildExportInstances(ctx, allInstances, filteredETIDs, etNameByID, etvByETID, instanceByID, assocNameByParentChild, assocByID)
	if err != nil {
		return nil, err
	}

	return &ExportData{
		FormatVersion:   "1.0",
		ExportedAt:      time.Now(),
		SourceSystem:    sourceSystem,
		Catalog:         ExportCatalog{Name: catalog.Name, Description: catalog.Description, ValidationStatus: string(catalog.ValidationStatus)},
		CatalogVersion:  ExportCatalogVersion{Label: cv.VersionLabel, Description: cv.Description},
		TypeDefinitions: exportTypeDefs,
		EntityTypes:     exportEntityTypes,
		Instances:       exportInstances,
	}, nil
}

func (s *ExportService) buildExportInstances(
	ctx context.Context,
	allInstances []*models.EntityInstance,
	filteredETIDs map[string]bool,
	etNameByID map[string]string,
	etvByETID map[string]*models.EntityTypeVersion,
	instanceByID map[string]*models.EntityInstance,
	assocNameByParentChild map[string]map[string]string,
	assocByID map[string]*models.Association,
) ([]ExportInstance, error) {
	// Group instances by parent
	childrenByParent := make(map[string][]*models.EntityInstance)
	for _, inst := range allInstances {
		childrenByParent[inst.ParentInstanceID] = append(childrenByParent[inst.ParentInstanceID], inst)
	}

	// Build recursively starting from root instances
	var buildInstance func(inst *models.EntityInstance) (*ExportInstance, error)
	buildInstance = func(inst *models.EntityInstance) (*ExportInstance, error) {
		if !filteredETIDs[inst.EntityTypeID] {
			return nil, nil
		}

		etv := etvByETID[inst.EntityTypeID]
		attrs, err := s.attrRepo.ListByVersion(ctx, etv.ID)
		if err != nil {
			return nil, err
		}

		values, err := s.iavRepo.GetValuesForVersion(ctx, inst.ID, inst.Version)
		if err != nil {
			return nil, err
		}

		baseTypeByAttr, err := ResolveBaseTypes(ctx, attrs, s.tdvRepo, s.tdRepo)
		if err != nil {
			return nil, err
		}

		attrMap := make(map[string]any)
		valueByAttrID := make(map[string]*models.InstanceAttributeValue)
		for _, v := range values {
			valueByAttrID[v.AttributeID] = v
		}
		for _, attr := range attrs {
			v, ok := valueByAttrID[attr.ID]
			if !ok {
				continue
			}
			baseType := models.BaseType(baseTypeByAttr[attr.ID])
			switch baseType {
			case models.BaseTypeList, models.BaseTypeJSON:
				if v.ValueJSON != "" {
					var parsed any
					if err := json.Unmarshal([]byte(v.ValueJSON), &parsed); err == nil {
						attrMap[attr.Name] = parsed
					} else {
						attrMap[attr.Name] = v.ValueJSON
					}
				}
			case models.BaseTypeNumber, models.BaseTypeInteger:
				if v.ValueNumber != nil {
					attrMap[attr.Name] = *v.ValueNumber
				} else if v.ValueString != "" {
					attrMap[attr.Name] = v.ValueString
				}
			default:
				if v.ValueString != "" {
					attrMap[attr.Name] = v.ValueString
				}
			}
		}

		// Resolve links (non-containment)
		links, err := s.linkRepo.GetForwardRefs(ctx, inst.ID)
		if err != nil {
			return nil, err
		}

		var exportLinks []ExportLink
		for _, link := range links {
			// Look up association directly by ID — links may reference older ETV associations
			assoc := assocByID[link.AssociationID]
			if assoc == nil {
				// Fallback: look up directly from DB (link references an older ETV's association)
				dbAssoc, err := s.assocRepo.GetByID(ctx, link.AssociationID)
				if err != nil || dbAssoc == nil {
					fmt.Printf("warning: export skipping link %s — association %s not found\n", link.ID, link.AssociationID)
					continue
				}
				assoc = dbAssoc
			}
			if assoc.Type == models.AssociationTypeContainment {
				continue
			}
			targetInst := instanceByID[link.TargetInstanceID]
			if targetInst == nil {
				continue
			}
			targetETName := etNameByID[targetInst.EntityTypeID]
			if !filteredETIDs[targetInst.EntityTypeID] {
				continue
			}

			el := ExportLink{
				Association: assoc.Name,
				TargetType:  targetETName,
				TargetName:  targetInst.Name,
			}

			// Add target_path for contained instances
			if targetInst.ParentInstanceID != "" {
				path := buildParentPath(targetInst, instanceByID)
				if len(path) > 0 {
					el.TargetPath = path
				}
			}

			exportLinks = append(exportLinks, el)
		}

		ei := ExportInstance{
			EntityType:  etNameByID[inst.EntityTypeID],
			Name:        inst.Name,
			Description: inst.Description,
			Attributes:  attrMap,
			Links:       exportLinks,
			Children:    make(map[string][]*ExportInstance),
		}

		// Build children (contained instances)
		children := childrenByParent[inst.ID]
		for _, child := range children {
			childExport, err := buildInstance(child)
			if err != nil {
				return nil, err
			}
			if childExport == nil {
				continue
			}
			// Determine association name
			assocName := ""
			if parentAssocs, ok := assocNameByParentChild[inst.EntityTypeID]; ok {
				assocName = parentAssocs[child.EntityTypeID]
			}
			if assocName == "" {
				assocName = "children"
			}
			ei.Children[assocName] = append(ei.Children[assocName], childExport)
		}

		return &ei, nil
	}

	// Only export root-level instances (no parent)
	var result []ExportInstance
	for _, inst := range allInstances {
		if inst.ParentInstanceID != "" {
			continue
		}
		ei, err := buildInstance(inst)
		if err != nil {
			return nil, err
		}
		if ei != nil {
			result = append(result, *ei)
		}
	}

	return result, nil
}

// buildParentPath walks up the containment tree to build a root-first path.
// Has cycle detection via visited map. No depth limit — containment trees are
// validated at creation time and are expected to be shallow (typically <10 levels).
// If pathological depth becomes an issue, add a maxDepth guard here.
func buildParentPath(inst *models.EntityInstance, instanceByID map[string]*models.EntityInstance) []string {
	var path []string
	visited := make(map[string]bool)
	current := instanceByID[inst.ParentInstanceID]
	for current != nil && !visited[current.ID] {
		visited[current.ID] = true
		path = append([]string{current.Name}, path...)
		current = instanceByID[current.ParentInstanceID]
	}
	return path
}
