package export

type WebhookValidateRequest struct {
	Parameters map[string]string  `json:"parameters"`
	Schema     WebhookSchemaInfo  `json:"schema"`
}

type WebhookValidateResponse struct {
	Valid bool   `json:"valid"`
	Error string `json:"error,omitempty"`
}

type WebhookSchemaInfo struct {
	EntityTypes []WebhookSchemaEntityType `json:"entity_types"`
}

type WebhookSchemaEntityType struct {
	Name         string                     `json:"name"`
	Attributes   []string                   `json:"attributes"`
	Associations []WebhookSchemaAssociation `json:"associations"`
}

type WebhookSchemaAssociation struct {
	Name             string `json:"name"`
	Type             string `json:"type"`
	TargetEntityType string `json:"target_entity_type"`
}

type WebhookExportRequest struct {
	CatalogName                string                          `json:"catalog_name"`
	CatalogDescription         string                          `json:"catalog_description"`
	Parameters                 map[string]string               `json:"parameters"`
	InstancesByType            map[string][]WebhookInstance     `json:"instances_by_type"`
	ChildrenOf                 map[string][]string             `json:"children_of"`
	VirtualServerInstanceName  string                          `json:"virtual_server_instance_name,omitempty"`
}

type WebhookInstance struct {
	ID          string                        `json:"id"`
	Name        string                        `json:"name"`
	Description string                        `json:"description"`
	Attributes  map[string]any                `json:"attributes"`
	ParentID    string                        `json:"parent_id"`
	Links       map[string][]WebhookLink      `json:"links,omitempty"`
}

type WebhookLink struct {
	TargetInstanceID   string `json:"target_instance_id"`
	TargetInstanceName string `json:"target_instance_name"`
	TargetEntityType   string `json:"target_entity_type"`
}

type WebhookExportResponse struct {
	Artifacts []WebhookArtifact `json:"artifacts"`
	Warnings  []string          `json:"warnings"`
}

type WebhookArtifact struct {
	APIVersion string `json:"api_version"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
	YAML       string `json:"yaml"`
}

type WebhookErrorResponse struct {
	Error string `json:"error"`
}

type WebhookHealthResponse struct {
	Status string `json:"status"`
}

func SchemaInfoToWebhook(schema SchemaInfo) WebhookSchemaInfo {
	ets := make([]WebhookSchemaEntityType, len(schema.EntityTypes))
	for i, et := range schema.EntityTypes {
		assocs := make([]WebhookSchemaAssociation, len(et.Associations))
		for j, a := range et.Associations {
			assocs[j] = WebhookSchemaAssociation{
				Name:             a.Name,
				Type:             a.Type,
				TargetEntityType: a.TargetEntityType,
			}
		}
		ets[i] = WebhookSchemaEntityType{
			Name:         et.Name,
			Attributes:   et.Attributes,
			Associations: assocs,
		}
	}
	return WebhookSchemaInfo{EntityTypes: ets}
}

func ExportInputToWebhookRequest(input ExportInput) WebhookExportRequest {
	instancesByType := make(map[string][]WebhookInstance, len(input.InstancesByType))
	for typeName, instances := range input.InstancesByType {
		whInstances := make([]WebhookInstance, len(instances))
		for i, inst := range instances {
			var links map[string][]WebhookLink
			if len(inst.LinksByAssoc) > 0 {
				links = make(map[string][]WebhookLink, len(inst.LinksByAssoc))
				for assocName, assocLinks := range inst.LinksByAssoc {
					whLinks := make([]WebhookLink, len(assocLinks))
					for j, l := range assocLinks {
						whLinks[j] = WebhookLink{
							TargetInstanceID:   l.TargetInstanceID,
							TargetInstanceName: l.TargetInstanceName,
							TargetEntityType:   l.TargetEntityType,
						}
					}
					links[assocName] = whLinks
				}
			}
			whInstances[i] = WebhookInstance{
				ID:          inst.ID,
				Name:        inst.Name,
				Description: inst.Description,
				Attributes:  inst.Attributes,
				ParentID:    inst.ParentID,
				Links:       links,
			}
		}
		instancesByType[typeName] = whInstances
	}

	childrenOf := make(map[string][]string, len(input.ChildrenOf))
	for parentID, children := range input.ChildrenOf {
		ids := make([]string, len(children))
		for i, c := range children {
			ids[i] = c.ID
		}
		childrenOf[parentID] = ids
	}

	return WebhookExportRequest{
		CatalogName:               input.CatalogName,
		CatalogDescription:        input.CatalogDesc,
		Parameters:                input.Parameters,
		InstancesByType:           instancesByType,
		ChildrenOf:                childrenOf,
		VirtualServerInstanceName: input.VirtualServerInstanceName,
	}
}

func WebhookExportResponseToOutput(resp WebhookExportResponse) *ExportOutput {
	artifacts := make([]K8sArtifact, len(resp.Artifacts))
	for i, a := range resp.Artifacts {
		artifacts[i] = K8sArtifact{
			APIVersion: a.APIVersion,
			Kind:       a.Kind,
			Name:       a.Name,
			Namespace:  a.Namespace,
			YAML:       a.YAML,
		}
	}
	return &ExportOutput{
		Artifacts: artifacts,
		Warnings:  resp.Warnings,
	}
}
