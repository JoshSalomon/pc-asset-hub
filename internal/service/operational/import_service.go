package operational

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository"
)

// === Import request/response types ===

type ImportRequest struct {
	CatalogName         string              `json:"catalog_name"`
	CatalogVersionLabel string              `json:"catalog_version_label"`
	RenameMap           *ImportRenameMap     `json:"rename_map"`
	ReuseExisting       []string            `json:"reuse_existing"`
	Data                *ExportData          `json:"data"`
}

type ImportRenameMap struct {
	EntityTypes     map[string]string `json:"entity_types"`
	TypeDefinitions map[string]string `json:"type_definitions"`
}

type DryRunResponse struct {
	Status     string         `json:"status"`
	Collisions []Collision    `json:"collisions"`
	Summary    DryRunSummary  `json:"summary"`
}

type Collision struct {
	Type       string `json:"type"`
	Name       string `json:"name"`
	Resolution string `json:"resolution"`
	Version    int    `json:"version,omitempty"`
	Detail     string `json:"detail"`
}

type DryRunSummary struct {
	TotalEntities int `json:"total_entities"`
	Conflicts     int `json:"conflicts"`
	Identical     int `json:"identical"`
	New           int `json:"new"`
}

type ImportResponse struct {
	Status         string `json:"status"`
	CatalogName    string `json:"catalog_name"`
	CatalogID      string `json:"catalog_id"`
	TypesCreated   int    `json:"types_created"`
	TypesReused    int    `json:"types_reused"`
	InstancesCreated int  `json:"instances_created"`
	LinksCreated   int    `json:"links_created"`
}

// === Import Service ===

type ImportService struct {
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
	typePinRepo repository.CatalogVersionTypePinRepository
	txManager   repository.TransactionManager
}

type ImportServiceOption func(*ImportService)

func WithImportTransactionManager(txm repository.TransactionManager) ImportServiceOption {
	return func(s *ImportService) {
		s.txManager = txm
	}
}

func NewImportService(
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
	typePinRepo repository.CatalogVersionTypePinRepository,
	opts ...ImportServiceOption,
) *ImportService {
	s := &ImportService{
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
		typePinRepo: typePinRepo,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *ImportService) applyRenames(data *ExportData, renameMap *ImportRenameMap) {
	if renameMap == nil {
		return
	}

	// Rename type definitions
	if len(renameMap.TypeDefinitions) > 0 {
		for i, td := range data.TypeDefinitions {
			if newName, ok := renameMap.TypeDefinitions[td.Name]; ok {
				data.TypeDefinitions[i].Name = newName
			}
		}
		// Rename type definition references in attributes
		for i, et := range data.EntityTypes {
			for j, attr := range et.Attributes {
				if newName, ok := renameMap.TypeDefinitions[attr.TypeDefinition]; ok {
					data.EntityTypes[i].Attributes[j].TypeDefinition = newName
				}
			}
		}
	}

	// Rename entity types
	if len(renameMap.EntityTypes) > 0 {
		for i, et := range data.EntityTypes {
			if newName, ok := renameMap.EntityTypes[et.Name]; ok {
				data.EntityTypes[i].Name = newName
			}
		}
		// Rename association targets
		for i, et := range data.EntityTypes {
			for j, assoc := range et.Associations {
				if newName, ok := renameMap.EntityTypes[assoc.Target]; ok {
					data.EntityTypes[i].Associations[j].Target = newName
				}
			}
		}
		// Rename instance entity_type references
		var renameInstances func(instances []ExportInstance)
		renameInstances = func(instances []ExportInstance) {
			for i, inst := range instances {
				if newName, ok := renameMap.EntityTypes[inst.EntityType]; ok {
					instances[i].EntityType = newName
				}
				for j, link := range inst.Links {
					if newName, ok := renameMap.EntityTypes[link.TargetType]; ok {
						instances[i].Links[j].TargetType = newName
					}
				}
				for assocName, children := range inst.Children {
					childSlice := make([]ExportInstance, len(children))
					for k, child := range children {
						childSlice[k] = *child
					}
					renameInstances(childSlice)
					for k := range childSlice {
						instances[i].Children[assocName][k] = &childSlice[k]
					}
				}
			}
		}
		renameInstances(data.Instances)
	}
}

func (s *ImportService) DryRun(ctx context.Context, req *ImportRequest) (*DryRunResponse, error) {
	if req.Data == nil {
		return nil, domainerrors.NewValidation("import data is required")
	}
	if req.Data.FormatVersion != "1.0" {
		return nil, domainerrors.NewValidation(fmt.Sprintf("unsupported format version: %s", req.Data.FormatVersion))
	}

	// Deep copy data and apply renames
	dataCopy := *req.Data
	s.applyRenames(&dataCopy, req.RenameMap)

	catalogName := dataCopy.Catalog.Name
	if req.CatalogName != "" {
		catalogName = req.CatalogName
	}
	if err := ValidateCatalogName(catalogName); err != nil {
		return nil, err
	}

	cvLabel := dataCopy.CatalogVersion.Label
	if req.CatalogVersionLabel != "" {
		cvLabel = req.CatalogVersionLabel
	}
	if cvLabel == "" {
		return nil, domainerrors.NewValidation("catalog version label is required")
	}

	reuseSet := make(map[string]bool)
	for _, name := range req.ReuseExisting {
		reuseSet[name] = true
	}

	var collisions []Collision
	newCount, identicalCount, conflictCount := 0, 0, 0

	// Check catalog name collision
	if _, err := s.catalogRepo.GetByName(ctx, catalogName); err == nil {
		collisions = append(collisions, Collision{Type: "catalog", Name: catalogName, Resolution: "conflict", Detail: "catalog name already exists"})
		conflictCount++
	} else if !domainerrors.IsNotFound(err) {
		return nil, err
	} else {
		newCount++
	}

	// Check CV label collision
	if _, err := s.cvRepo.GetByLabel(ctx, cvLabel); err == nil {
		collisions = append(collisions, Collision{Type: "catalog_version", Name: cvLabel, Resolution: "conflict", Detail: "catalog version label already exists"})
		conflictCount++
	} else if !domainerrors.IsNotFound(err) {
		return nil, err
	} else {
		newCount++
	}

	// Check type definitions
	for _, td := range dataCopy.TypeDefinitions {
		existingTD, err := s.tdRepo.GetByName(ctx, td.Name)
		if err != nil && !domainerrors.IsNotFound(err) {
			return nil, err
		}
		if err == nil {
			// Exists — check if identical
			latestTDV, err := s.tdvRepo.GetLatestByTypeDefinition(ctx, existingTD.ID)
			if err != nil {
				return nil, err
			}
			if isTypeDefIdentical(existingTD, latestTDV, td) {
				resolution := "identical"
				collisions = append(collisions, Collision{Type: "type_definition", Name: td.Name, Resolution: resolution, Version: latestTDV.VersionNumber, Detail: "structurally identical"})
				identicalCount++
			} else {
				collisions = append(collisions, Collision{Type: "type_definition", Name: td.Name, Resolution: "conflict", Detail: "different constraints"})
				conflictCount++
			}
		} else {
			newCount++
		}
	}

	// Check entity types
	for _, et := range dataCopy.EntityTypes {
		if reuseSet[et.Name] {
			continue
		}
		existingET, err := s.etRepo.GetByName(ctx, et.Name)
		if err != nil && !domainerrors.IsNotFound(err) {
			return nil, err
		}
		if err == nil {
			latestETV, err := s.etvRepo.GetLatestByEntityType(ctx, existingET.ID)
			if err != nil {
				return nil, err
			}
			attrs, err := s.attrRepo.ListByVersion(ctx, latestETV.ID)
			if err != nil {
				return nil, err
			}
			assocs, err := s.assocRepo.ListByVersion(ctx, latestETV.ID)
			if err != nil {
				return nil, err
			}
			if isEntityTypeIdentical(attrs, assocs, et, s.tdvRepo, s.tdRepo, s.etRepo, ctx) {
				collisions = append(collisions, Collision{Type: "entity_type", Name: et.Name, Resolution: "identical", Version: latestETV.Version, Detail: "structurally identical"})
				identicalCount++
			} else {
				collisions = append(collisions, Collision{Type: "entity_type", Name: et.Name, Resolution: "conflict", Detail: "exists with different schema"})
				conflictCount++
			}
		} else {
			newCount++
		}
	}

	status := "ready"
	if conflictCount > 0 {
		status = "conflicts_found"
	}

	return &DryRunResponse{
		Status:     status,
		Collisions: collisions,
		Summary: DryRunSummary{
			TotalEntities: newCount + identicalCount + conflictCount,
			Conflicts:     conflictCount,
			Identical:     identicalCount,
			New:           newCount,
		},
	}, nil
}

func (s *ImportService) Import(ctx context.Context, req *ImportRequest) (*ImportResponse, error) {
	if req.Data == nil {
		return nil, domainerrors.NewValidation("import data is required")
	}
	if req.Data.FormatVersion != "1.0" {
		return nil, domainerrors.NewValidation(fmt.Sprintf("unsupported format version: %s", req.Data.FormatVersion))
	}

	dataCopy := *req.Data
	s.applyRenames(&dataCopy, req.RenameMap)

	catalogName := dataCopy.Catalog.Name
	if req.CatalogName != "" {
		catalogName = req.CatalogName
	}
	if err := ValidateCatalogName(catalogName); err != nil {
		return nil, err
	}

	cvLabel := dataCopy.CatalogVersion.Label
	if req.CatalogVersionLabel != "" {
		cvLabel = req.CatalogVersionLabel
	}
	if cvLabel == "" {
		return nil, domainerrors.NewValidation("catalog version label is required")
	}

	reuseSet := make(map[string]bool)
	for _, name := range req.ReuseExisting {
		reuseSet[name] = true
	}

	// Check if catalog name already exists
	if _, err := s.catalogRepo.GetByName(ctx, catalogName); err == nil {
		return nil, domainerrors.NewConflict("Catalog", fmt.Sprintf("catalog %q already exists", catalogName))
	} else if !domainerrors.IsNotFound(err) {
		return nil, err
	}

	// Track created/reused IDs
	typesCreated := 0
	typesReused := 0
	tdNameToVersionID := make(map[string]string)   // type def name → TDV ID
	etNameToID := make(map[string]string)           // entity type name → ET ID
	etNameToVersionID := make(map[string]string)    // entity type name → ETV ID
	createdETNames := make(map[string]bool)         // entity types newly created (not reused)
	allAssocByName := make(map[string]*models.Association) // global assoc name → Association (for cross-ETV link resolution)

	// === Transaction 1: Schema ===
	schemaFn := func(txCtx context.Context) error {
		// 1. Create/reuse type definitions
		for _, td := range dataCopy.TypeDefinitions {
			existingTD, err := s.tdRepo.GetByName(txCtx, td.Name)
			if err != nil && !domainerrors.IsNotFound(err) {
				return err
			}
			if err == nil {
				latestTDV, err := s.tdvRepo.GetLatestByTypeDefinition(txCtx, existingTD.ID)
				if err != nil {
					return err
				}
				if isTypeDefIdentical(existingTD, latestTDV, td) {
					tdNameToVersionID[td.Name] = latestTDV.ID
					typesReused++
					continue
				}
				return domainerrors.NewConflict("TypeDefinition", fmt.Sprintf("type definition %q conflicts with existing — rename or reuse", td.Name))
			}

			tdID := uuid.Must(uuid.NewV7()).String()
			now := time.Now()
			newTD := &models.TypeDefinition{
				ID:          tdID,
				Name:        td.Name,
				Description: td.Description,
				BaseType:    models.BaseType(td.BaseType),
				System:      false,
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			if err := s.tdRepo.Create(txCtx, newTD); err != nil {
				return err
			}

			tdvID := uuid.Must(uuid.NewV7()).String()
			newTDV := &models.TypeDefinitionVersion{
				ID:               tdvID,
				TypeDefinitionID: tdID,
				VersionNumber:    1,
				Constraints:      td.Constraints,
				CreatedAt:        now,
			}
			if err := s.tdvRepo.Create(txCtx, newTDV); err != nil {
				return err
			}
			tdNameToVersionID[td.Name] = tdvID
			typesCreated++
		}

		// Resolve system type definitions by name
		systemTDs, _, err := s.tdRepo.List(txCtx, models.ListParams{Limit: 1000})
		if err != nil {
			return err
		}
		for _, td := range systemTDs {
			if td.System {
				latestTDV, err := s.tdvRepo.GetLatestByTypeDefinition(txCtx, td.ID)
				if err != nil {
					return fmt.Errorf("failed to resolve system type definition %q: %w", td.Name, err)
				}
				tdNameToVersionID[td.Name] = latestTDV.ID
			}
		}

		// 2. Create CV
		cvID := uuid.Must(uuid.NewV7()).String()
		now := time.Now()
		newCV := &models.CatalogVersion{
			ID:             cvID,
			VersionLabel:   cvLabel,
			Description:    dataCopy.CatalogVersion.Description,
			LifecycleStage: models.LifecycleStageDevelopment,
			CreatedAt:      now,
			UpdatedAt:      now,
		}
		if err := s.cvRepo.Create(txCtx, newCV); err != nil {
			return err
		}

		// 3. Create/reuse entity types
		for _, et := range dataCopy.EntityTypes {
			if reuseSet[et.Name] {
				existingET, err := s.etRepo.GetByName(txCtx, et.Name)
				if err != nil {
					return fmt.Errorf("reuse_existing entity type %q not found: %w", et.Name, err)
				}
				latestETV, err := s.etvRepo.GetLatestByEntityType(txCtx, existingET.ID)
				if err != nil {
					return err
				}
				etNameToID[et.Name] = existingET.ID
				etNameToVersionID[et.Name] = latestETV.ID

				// Pin the existing entity type version
				pinID := uuid.Must(uuid.NewV7()).String()
				if err := s.pinRepo.Create(txCtx, &models.CatalogVersionPin{
					ID:                  pinID,
					CatalogVersionID:    cvID,
					EntityTypeVersionID: latestETV.ID,
				}); err != nil {
					return err
				}
				reusedAssocs, err := s.assocRepo.ListByVersion(txCtx, latestETV.ID)
				if err != nil {
					return err
				}
				for _, a := range reusedAssocs {
					allAssocByName[a.Name] = a
				}
				typesReused++
				continue
			}

			// Check for identical existing entity type
			existingET, err := s.etRepo.GetByName(txCtx, et.Name)
			if err == nil {
				latestETV, err := s.etvRepo.GetLatestByEntityType(txCtx, existingET.ID)
				if err != nil {
					return err
				}
				attrs, err := s.attrRepo.ListByVersion(txCtx, latestETV.ID)
				if err != nil {
					return err
				}
				assocs, err := s.assocRepo.ListByVersion(txCtx, latestETV.ID)
				if err != nil {
					return err
				}
				if isEntityTypeIdentical(attrs, assocs, et, s.tdvRepo, s.tdRepo, s.etRepo, txCtx) {
					etNameToID[et.Name] = existingET.ID
					etNameToVersionID[et.Name] = latestETV.ID
					pinID := uuid.Must(uuid.NewV7()).String()
					if err := s.pinRepo.Create(txCtx, &models.CatalogVersionPin{
						ID:                  pinID,
						CatalogVersionID:    cvID,
						EntityTypeVersionID: latestETV.ID,
					}); err != nil {
						return err
					}
					for _, a := range assocs {
						allAssocByName[a.Name] = a
					}
					typesReused++
					continue
				}
				return domainerrors.NewConflict("EntityType", fmt.Sprintf("entity type %q conflicts with existing — rename or add to reuse_existing", et.Name))
			} else if !domainerrors.IsNotFound(err) {
				return err
			}

			// Create new entity type
			etID := uuid.Must(uuid.NewV7()).String()
			now := time.Now()
			newET := &models.EntityType{
				ID:        etID,
				Name:      et.Name,
				CreatedAt: now,
				UpdatedAt: now,
			}
			if err := s.etRepo.Create(txCtx, newET); err != nil {
				return err
			}

			etvID := uuid.Must(uuid.NewV7()).String()
			newETV := &models.EntityTypeVersion{
				ID:           etvID,
				EntityTypeID: etID,
				Version:      1,
				Description:  et.Description,
				CreatedAt:    now,
			}
			if err := s.etvRepo.Create(txCtx, newETV); err != nil {
				return err
			}

			etNameToID[et.Name] = etID
			etNameToVersionID[et.Name] = etvID

			// Create attributes
			for _, attr := range et.Attributes {
				tdvID, ok := tdNameToVersionID[attr.TypeDefinition]
				if !ok {
					return domainerrors.NewValidation(fmt.Sprintf("type definition %q not found for attribute %q", attr.TypeDefinition, attr.Name))
				}
				attrID := uuid.Must(uuid.NewV7()).String()
				if err := s.attrRepo.Create(txCtx, &models.Attribute{
					ID:                      attrID,
					EntityTypeVersionID:     etvID,
					Name:                    attr.Name,
					Description:             attr.Description,
					TypeDefinitionVersionID: tdvID,
					Ordinal:                 attr.Ordinal,
					Required:                attr.Required,
				}); err != nil {
					return err
				}
			}

			// Pin entity type version
			pinID := uuid.Must(uuid.NewV7()).String()
			if err := s.pinRepo.Create(txCtx, &models.CatalogVersionPin{
				ID:                  pinID,
				CatalogVersionID:    cvID,
				EntityTypeVersionID: etvID,
			}); err != nil {
				return err
			}

			typesCreated++
			createdETNames[et.Name] = true
		}

		// 4. Create associations (second pass — need all ET IDs resolved)
		for _, et := range dataCopy.EntityTypes {
			if !createdETNames[et.Name] {
				continue
			}
			etvID := etNameToVersionID[et.Name]

			for _, assoc := range et.Associations {
				targetETID, ok := etNameToID[assoc.Target]
				if !ok {
					return domainerrors.NewValidation(fmt.Sprintf("association target entity type %q not found", assoc.Target))
				}
				assocID := uuid.Must(uuid.NewV7()).String()
				newAssoc := &models.Association{
					ID:                  assocID,
					EntityTypeVersionID: etvID,
					Name:                assoc.Name,
					TargetEntityTypeID:  targetETID,
					Type:                models.AssociationType(assoc.Type),
					SourceRole:          assoc.SourceRole,
					TargetRole:          assoc.TargetRole,
					SourceCardinality:   assoc.SourceCardinality,
					TargetCardinality:   assoc.TargetCardinality,
					CreatedAt:           time.Now(),
				}
				if err := s.assocRepo.Create(txCtx, newAssoc); err != nil {
					return err
				}
				allAssocByName[assoc.Name] = newAssoc
			}
		}

		// Auto-pin type definitions (create CatalogVersionTypePin entries)
		for _, tdvID := range tdNameToVersionID {
			typePinID := uuid.Must(uuid.NewV7()).String()
			if err := s.typePinRepo.Create(txCtx, &models.CatalogVersionTypePin{
				ID:                      typePinID,
				CatalogVersionID:        cvID,
				TypeDefinitionVersionID: tdvID,
			}); err != nil {
				return err
			}
		}

		// Store CV ID for data transaction
		etNameToID["__cv_id__"] = cvID
		return nil
	}

	if s.txManager != nil {
		if err := s.txManager.RunInTransaction(ctx, schemaFn); err != nil {
			return nil, err
		}
	} else {
		if err := schemaFn(ctx); err != nil {
			return nil, err
		}
	}

	cvID := etNameToID["__cv_id__"]
	delete(etNameToID, "__cv_id__")

	// === Transaction 2: Data ===
	instancesCreated := 0
	linksCreated := 0

	dataFn := func(txCtx context.Context) error {
		// Create catalog
		now := time.Now()
		catalogID := uuid.Must(uuid.NewV7()).String()
		newCatalog := &models.Catalog{
			ID:               catalogID,
			Name:             catalogName,
			Description:      dataCopy.Catalog.Description,
			CatalogVersionID: cvID,
			ValidationStatus: models.ValidationStatusDraft,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		if err := s.catalogRepo.Create(txCtx, newCatalog); err != nil {
			return err
		}

		// Store catalog ID for response
		etNameToID["__catalog_id__"] = catalogID

		// Walk instance tree recursively
		instanceNameToID := make(map[string]string) // "entityType/name" → instance ID

		var createInstance func(inst ExportInstance, parentID string) error
		createInstance = func(inst ExportInstance, parentID string) error {
			etID, ok := etNameToID[inst.EntityType]
			if !ok {
				return domainerrors.NewValidation(fmt.Sprintf("entity type %q not found during import", inst.EntityType))
			}
			etvID, ok := etNameToVersionID[inst.EntityType]
			if !ok {
				return domainerrors.NewValidation(fmt.Sprintf("entity type version for %q not found during import", inst.EntityType))
			}

			instID := uuid.Must(uuid.NewV7()).String()
			instNow := time.Now()
			newInst := &models.EntityInstance{
				ID:               instID,
				EntityTypeID:     etID,
				CatalogID:        catalogID,
				ParentInstanceID: parentID,
				Name:             inst.Name,
				Description:      inst.Description,
				Version:          1,
				CreatedAt:        instNow,
				UpdatedAt:        instNow,
			}
			if err := s.instRepo.Create(txCtx, newInst); err != nil {
				return err
			}
			instancesCreated++

			// Key for link resolution
			instanceNameToID[inst.EntityType+"/"+inst.Name] = instID

			// Set attribute values
			if len(inst.Attributes) > 0 {
				attrs, err := s.attrRepo.ListByVersion(txCtx, etvID)
				if err != nil {
					return err
				}
				attrByName := make(map[string]*models.Attribute)
				for _, a := range attrs {
					attrByName[a.Name] = a
				}

				baseTypeByAttr, err := ResolveBaseTypes(txCtx, attrs, s.tdvRepo, s.tdRepo)
				if err != nil {
					return err
				}

				var values []*models.InstanceAttributeValue
				for name, rawVal := range inst.Attributes {
					attr, ok := attrByName[name]
					if !ok {
						continue
					}
					iav := &models.InstanceAttributeValue{
						ID:              uuid.Must(uuid.NewV7()).String(),
						InstanceID:      instID,
						InstanceVersion: 1,
						AttributeID:     attr.ID,
					}
					baseType := models.BaseType(baseTypeByAttr[attr.ID])
					switch baseType {
					case models.BaseTypeList, models.BaseTypeJSON:
						switch v := rawVal.(type) {
						case string:
							iav.ValueJSON = v
						default:
							b, err := json.Marshal(v)
							if err != nil {
								return err
							}
							iav.ValueJSON = string(b)
						}
					case models.BaseTypeNumber, models.BaseTypeInteger:
						switch v := rawVal.(type) {
						case float64:
							iav.ValueNumber = &v
							iav.ValueString = strconv.FormatFloat(v, 'f', -1, 64)
						case string:
							iav.ValueString = v
							if f, err := strconv.ParseFloat(v, 64); err == nil {
								iav.ValueNumber = &f
							}
						default:
							iav.ValueString = fmt.Sprintf("%v", rawVal)
						}
					default:
						iav.ValueString = fmt.Sprintf("%v", rawVal)
					}
					values = append(values, iav)
				}
				if len(values) > 0 {
					if err := s.iavRepo.SetValues(txCtx, values); err != nil {
						return err
					}
				}
			}

			// Recurse into children
			for _, children := range inst.Children {
				for _, child := range children {
					if err := createInstance(*child, instID); err != nil {
						return err
					}
				}
			}

			return nil
		}

		for _, inst := range dataCopy.Instances {
			if err := createInstance(inst, ""); err != nil {
				return err
			}
		}

		// Create association links
		for _, inst := range dataCopy.Instances {
			n, err := s.createLinksRecursive(txCtx, inst, etNameToVersionID, instanceNameToID, allAssocByName)
			if err != nil {
				return err
			}
			linksCreated += n
		}

		return nil
	}

	if s.txManager != nil {
		if err := s.txManager.RunInTransaction(ctx, dataFn); err != nil {
			return nil, fmt.Errorf("schema created successfully, data import failed: %w", err)
		}
	} else {
		if err := dataFn(ctx); err != nil {
			return nil, fmt.Errorf("schema created successfully, data import failed: %w", err)
		}
	}

	catalogID := etNameToID["__catalog_id__"]

	return &ImportResponse{
		Status:           "success",
		CatalogName:      catalogName,
		CatalogID:        catalogID,
		TypesCreated:     typesCreated,
		TypesReused:      typesReused,
		InstancesCreated: instancesCreated,
		LinksCreated:     linksCreated,
	}, nil
}

func (s *ImportService) createLinksRecursive(ctx context.Context, inst ExportInstance, etNameToVersionID map[string]string, instanceNameToID map[string]string, allAssocByName map[string]*models.Association) (int, error) {
	created := 0
	if len(inst.Links) > 0 {
		etvID := etNameToVersionID[inst.EntityType]
		assocs, err := s.assocRepo.ListByVersion(ctx, etvID)
		if err != nil {
			return 0, err
		}
		assocByName := make(map[string]*models.Association)
		for _, a := range assocs {
			assocByName[a.Name] = a
		}

		sourceID := instanceNameToID[inst.EntityType+"/"+inst.Name]
		for _, link := range inst.Links {
			assoc, ok := assocByName[link.Association]
			if !ok {
				// Bidirectional associations may be defined on the target entity type
				assoc, ok = allAssocByName[link.Association]
				if !ok {
					continue
				}
			}

			targetKey := link.TargetType + "/" + link.TargetName
			targetID, ok := instanceNameToID[targetKey]
			if !ok {
				continue
			}

			linkID := uuid.Must(uuid.NewV7()).String()
			if err := s.linkRepo.Create(ctx, &models.AssociationLink{
				ID:               linkID,
				AssociationID:    assoc.ID,
				SourceInstanceID: sourceID,
				TargetInstanceID: targetID,
				CreatedAt:        time.Now(),
			}); err != nil {
				return 0, err
			}
			created++
		}
	}

	for _, children := range inst.Children {
		for _, child := range children {
			n, err := s.createLinksRecursive(ctx, *child, etNameToVersionID, instanceNameToID, allAssocByName)
			if err != nil {
				return 0, err
			}
			created += n
		}
	}

	return created, nil
}

// isTypeDefIdentical compares an existing type definition with its version against an export type def.
func isTypeDefIdentical(existingTD *models.TypeDefinition, existingTDV *models.TypeDefinitionVersion, importTD ExportTypeDef) bool {
	if string(existingTD.BaseType) != importTD.BaseType {
		return false
	}
	if existingTD.Description != importTD.Description {
		return false
	}
	return constraintsEqual(existingTDV.Constraints, importTD.Constraints)
}

// isEntityTypeIdentical compares an existing entity type's schema against an export entity type.
func isEntityTypeIdentical(
	existingAttrs []*models.Attribute,
	existingAssocs []*models.Association,
	importET ExportEntityType,
	tdvRepo repository.TypeDefinitionVersionRepository,
	tdRepo repository.TypeDefinitionRepository,
	etRepo repository.EntityTypeRepository,
	ctx context.Context,
) bool {
	if len(existingAttrs) != len(importET.Attributes) {
		return false
	}
	if len(existingAssocs) != len(importET.Associations) {
		return false
	}

	// Compare attributes by name
	existingAttrByName := make(map[string]*models.Attribute)
	for _, a := range existingAttrs {
		existingAttrByName[a.Name] = a
	}
	for _, importAttr := range importET.Attributes {
		existingAttr, ok := existingAttrByName[importAttr.Name]
		if !ok {
			return false
		}
		if existingAttr.Required != importAttr.Required {
			return false
		}
		// Resolve type definition name for comparison
		tdv, err := tdvRepo.GetByID(ctx, existingAttr.TypeDefinitionVersionID)
		if err != nil {
			return false
		}
		td, err := tdRepo.GetByID(ctx, tdv.TypeDefinitionID)
		if err != nil {
			return false
		}
		if td.Name != importAttr.TypeDefinition {
			return false
		}
	}

	// Compare associations by name
	existingAssocByName := make(map[string]*models.Association)
	for _, a := range existingAssocs {
		existingAssocByName[a.Name] = a
	}
	for _, importAssoc := range importET.Associations {
		existingAssoc, ok := existingAssocByName[importAssoc.Name]
		if !ok {
			return false
		}
		if string(existingAssoc.Type) != importAssoc.Type {
			return false
		}
		// Resolve target entity type name
		targetET, err := etRepo.GetByID(ctx, existingAssoc.TargetEntityTypeID)
		if err != nil {
			return false
		}
		if targetET.Name != importAssoc.Target {
			return false
		}
	}

	return true
}

func constraintsEqual(a, b map[string]any) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	aJSON, err := json.Marshal(a)
	if err != nil {
		return false
	}
	bJSON, err := json.Marshal(b)
	if err != nil {
		return false
	}
	// Normalize by unmarshaling back
	var aNorm, bNorm any
	_ = json.Unmarshal(aJSON, &aNorm)
	_ = json.Unmarshal(bJSON, &bNorm)
	aNormJSON, _ := json.Marshal(aNorm)
	bNormJSON, _ := json.Marshal(bNorm)
	return string(aNormJSON) == string(bNormJSON)
}

// applyMassRename applies prefix/suffix to all entity type and type definition names.
func ApplyMassRename(renameMap *ImportRenameMap, entityTypes []ExportEntityType, typeDefs []ExportTypeDef, prefix, suffix string) *ImportRenameMap {
	if prefix == "" && suffix == "" {
		return renameMap
	}
	if renameMap == nil {
		renameMap = &ImportRenameMap{
			EntityTypes:     make(map[string]string),
			TypeDefinitions: make(map[string]string),
		}
	}
	for _, et := range entityTypes {
		if _, ok := renameMap.EntityTypes[et.Name]; !ok {
			newName := prefix + et.Name + suffix
			if newName != et.Name {
				renameMap.EntityTypes[et.Name] = newName
			}
		}
	}
	for _, td := range typeDefs {
		if _, ok := renameMap.TypeDefinitions[td.Name]; !ok {
			newName := prefix + td.Name + suffix
			if newName != td.Name {
				renameMap.TypeDefinitions[td.Name] = newName
			}
		}
	}
	return renameMap
}

// stripChildren removes child containment keys from instances for JSON unmarshaling.
// Used when parsing the raw JSON export file since children are mixed in with standard fields.
func ParseExportInstances(raw []json.RawMessage, entityTypes []ExportEntityType) ([]ExportInstance, error) {
	// Build set of containment association names
	containmentNames := make(map[string]string) // assoc name → target entity type name
	for _, et := range entityTypes {
		for _, assoc := range et.Associations {
			if assoc.Type == "containment" {
				containmentNames[assoc.Name] = assoc.Target
			}
		}
	}

	var result []ExportInstance
	for _, r := range raw {
		var m map[string]json.RawMessage
		if err := json.Unmarshal(r, &m); err != nil {
			return nil, err
		}

		inst := ExportInstance{
			Attributes: make(map[string]any),
			Children:   make(map[string][]*ExportInstance),
		}

		// Standard fields
		if v, ok := m["entity_type"]; ok {
			_ = json.Unmarshal(v, &inst.EntityType)
		}
		if v, ok := m["name"]; ok {
			_ = json.Unmarshal(v, &inst.Name)
		}
		if v, ok := m["description"]; ok {
			_ = json.Unmarshal(v, &inst.Description)
		}
		if v, ok := m["attributes"]; ok {
			_ = json.Unmarshal(v, &inst.Attributes)
		}
		if v, ok := m["links"]; ok {
			_ = json.Unmarshal(v, &inst.Links)
		}

		// Children — any key matching a containment association name
		for key, val := range m {
			if _, isContainment := containmentNames[key]; isContainment {
				var childRaws []json.RawMessage
				if err := json.Unmarshal(val, &childRaws); err != nil {
					continue
				}
				children, err := ParseExportInstances(childRaws, entityTypes)
				if err != nil {
					return nil, err
				}
				for i := range children {
					// Infer entity type from association target
					if children[i].EntityType == "" {
						children[i].EntityType = containmentNames[key]
					}
					inst.Children[key] = append(inst.Children[key], &children[i])
				}
			}
		}

		result = append(result, inst)
	}
	return result, nil
}
