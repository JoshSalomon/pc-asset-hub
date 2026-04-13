package crdgen

import (
	"encoding/json"
	"fmt"

	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
)

// CRDSpec represents a simplified Kubernetes CRD specification.
type CRDSpec struct {
	APIVersion string            `json:"apiVersion"`
	Kind       string            `json:"kind"`
	Metadata   CRDMetadata       `json:"metadata"`
	Spec       CRDSpecDefinition `json:"spec"`
}

type CRDMetadata struct {
	Name string `json:"name"`
}

type CRDSpecDefinition struct {
	Group    string       `json:"group"`
	Names    CRDNames     `json:"names"`
	Scope    string       `json:"scope"`
	Versions []CRDVersion `json:"versions"`
}

type CRDNames struct {
	Kind     string `json:"kind"`
	Plural   string `json:"plural"`
	Singular string `json:"singular"`
}

type CRDVersion struct {
	Name   string          `json:"name"`
	Served bool            `json:"served"`
	Schema json.RawMessage `json:"schema"`
}

// GenerateCRD creates a Kubernetes CRD YAML representation from an entity type definition.
func GenerateCRD(entityType *models.EntityType, attributes []*models.Attribute) (*CRDSpec, error) {
	if entityType == nil {
		return nil, fmt.Errorf("entity type is nil")
	}

	properties := make(map[string]map[string]string)
	for _, attr := range attributes {
		prop := map[string]string{}
		// Default to string type for CRD schema — the actual base type
		// is resolved via TypeDefinitionVersion at runtime.
		prop["type"] = "string"
		properties[attr.Name] = prop
	}

	schema := map[string]any{
		"openAPIV3Schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"spec": map[string]any{
					"type":       "object",
					"properties": properties,
				},
			},
		},
	}

	schemaJSON, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema: %w", err)
	}

	return &CRDSpec{
		APIVersion: "apiextensions.k8s.io/v1",
		Kind:       "CustomResourceDefinition",
		Metadata: CRDMetadata{
			Name: entityType.Name + ".assethub.project-catalyst.io",
		},
		Spec: CRDSpecDefinition{
			Group: "assethub.project-catalyst.io",
			Names: CRDNames{
				Kind:     entityType.Name,
				Plural:   entityType.Name + "s",
				Singular: entityType.Name,
			},
			Scope: "Namespaced",
			Versions: []CRDVersion{
				{
					Name:   "v1",
					Served: true,
					Schema: schemaJSON,
				},
			},
		},
	}, nil
}

// GenerateCRDJSON returns the CRD as a JSON string.
func GenerateCRDJSON(entityType *models.EntityType, attributes []*models.Attribute) (string, error) {
	crd, err := GenerateCRD(entityType, attributes)
	if err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(crd, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// CRInstance represents a simplified Kubernetes Custom Resource instance.
type CRInstance struct {
	APIVersion string         `json:"apiVersion"`
	Kind       string         `json:"kind"`
	Metadata   CRInstanceMeta `json:"metadata"`
	Spec       map[string]any `json:"spec"`
}

type CRInstanceMeta struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

// GenerateCR creates a Kubernetes CR from an entity instance.
func GenerateCR(entityType *models.EntityType, instance *models.EntityInstance, attributeValues []*models.InstanceAttributeValue, attributes []*models.Attribute) (*CRInstance, error) {
	if entityType == nil || instance == nil {
		return nil, fmt.Errorf("entity type and instance are required")
	}

	// Build attribute name map
	attrMap := make(map[string]*models.Attribute)
	for _, a := range attributes {
		attrMap[a.ID] = a
	}

	spec := make(map[string]any)
	for _, av := range attributeValues {
		attr, ok := attrMap[av.AttributeID]
		if !ok {
			continue
		}
		// Use whichever value column is populated
		if av.ValueNumber != nil {
			spec[attr.Name] = *av.ValueNumber
		} else if av.ValueJSON != "" {
			spec[attr.Name] = av.ValueJSON
		} else {
			spec[attr.Name] = av.ValueString
		}
	}

	return &CRInstance{
		APIVersion: "assethub.project-catalyst.io/v1",
		Kind:       entityType.Name,
		Metadata: CRInstanceMeta{
			Name: instance.Name,
		},
		Spec: spec,
	}, nil
}
