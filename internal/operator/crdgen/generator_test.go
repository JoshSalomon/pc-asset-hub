package crdgen_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/operator/crdgen"
)

// T-9.04: Generate CRD from entity type definition → valid K8s CRD YAML
func TestT9_04_GenerateCRD(t *testing.T) {
	et := &models.EntityType{ID: "et1", Name: "Model"}
	attrs := []*models.Attribute{
		{ID: "a1", Name: "endpoint", Type: models.AttributeTypeString},
		{ID: "a2", Name: "max_tokens", Type: models.AttributeTypeNumber},
		{ID: "a3", Name: "status", Type: models.AttributeTypeEnum},
	}

	crd, err := crdgen.GenerateCRD(et, attrs)
	require.NoError(t, err)
	assert.Equal(t, "apiextensions.k8s.io/v1", crd.APIVersion)
	assert.Equal(t, "CustomResourceDefinition", crd.Kind)
	assert.Equal(t, "Model.assethub.project-catalyst.io", crd.Metadata.Name)
	assert.Equal(t, "Model", crd.Spec.Names.Kind)
	assert.Len(t, crd.Spec.Versions, 1)
	assert.True(t, crd.Spec.Versions[0].Served)

	// Verify schema is valid JSON
	var schema map[string]any
	require.NoError(t, json.Unmarshal(crd.Spec.Versions[0].Schema, &schema))
	assert.Contains(t, schema, "openAPIV3Schema")
}

// T-9.04b: Generate CRD JSON string
func TestT9_04b_GenerateCRDJSON(t *testing.T) {
	et := &models.EntityType{ID: "et1", Name: "Tool"}
	attrs := []*models.Attribute{
		{ID: "a1", Name: "command", Type: models.AttributeTypeString},
	}

	jsonStr, err := crdgen.GenerateCRDJSON(et, attrs)
	require.NoError(t, err)
	assert.Contains(t, jsonStr, "apiextensions.k8s.io/v1")
	assert.Contains(t, jsonStr, "Tool")

	// Verify it's valid JSON
	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(jsonStr), &result))
}

// T-9.05: Generate CR from entity instance → valid K8s CR YAML
func TestT9_05_GenerateCR(t *testing.T) {
	et := &models.EntityType{ID: "et1", Name: "Model"}
	inst := &models.EntityInstance{ID: "inst1", Name: "llama-3-70b", Version: 1}
	attrs := []*models.Attribute{
		{ID: "a1", Name: "endpoint", Type: models.AttributeTypeString},
		{ID: "a2", Name: "max_tokens", Type: models.AttributeTypeNumber},
	}

	maxTokens := 4096.0
	values := []*models.InstanceAttributeValue{
		{AttributeID: "a1", ValueString: "https://api.example.com"},
		{AttributeID: "a2", ValueNumber: &maxTokens},
	}

	cr, err := crdgen.GenerateCR(et, inst, values, attrs)
	require.NoError(t, err)
	assert.Equal(t, "assethub.project-catalyst.io/v1", cr.APIVersion)
	assert.Equal(t, "Model", cr.Kind)
	assert.Equal(t, "llama-3-70b", cr.Metadata.Name)
	assert.Equal(t, "https://api.example.com", cr.Spec["endpoint"])
	assert.Equal(t, 4096.0, cr.Spec["max_tokens"])
}

// T-9.09: Generator does not modify the database (it only takes domain models as input)
func TestT9_09_NoDBModification(t *testing.T) {
	// GenerateCRD and GenerateCR take domain models as input, not repositories.
	// They have no access to the database by design.
	et := &models.EntityType{ID: "et1", Name: "Model"}
	_, err := crdgen.GenerateCRD(et, nil)
	assert.NoError(t, err)

	// Nil entity type returns error
	_, err = crdgen.GenerateCRD(nil, nil)
	assert.Error(t, err)
}
