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
		{ID: "a1", Name: "endpoint", TypeDefinitionVersionID: "tdv-string"},
		{ID: "a2", Name: "max_tokens", TypeDefinitionVersionID: "tdv-number"},
		{ID: "a3", Name: "status", TypeDefinitionVersionID: "tdv-enum"},
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
		{ID: "a1", Name: "command", TypeDefinitionVersionID: "tdv-string"},
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
		{ID: "a1", Name: "endpoint", TypeDefinitionVersionID: "tdv-string"},
		{ID: "a2", Name: "max_tokens", TypeDefinitionVersionID: "tdv-number"},
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

// GenerateCRDJSON: nil entity type propagates error (line 106)
func TestGenerateCRDJSON_NilEntityType(t *testing.T) {
	_, err := crdgen.GenerateCRDJSON(nil, nil)
	assert.Error(t, err)
}

// GenerateCR: nil entity type or instance (line 131)
func TestGenerateCR_NilInputs(t *testing.T) {
	_, err := crdgen.GenerateCR(nil, &models.EntityInstance{}, nil, nil)
	assert.Error(t, err)

	_, err = crdgen.GenerateCR(&models.EntityType{}, nil, nil, nil)
	assert.Error(t, err)
}

// GenerateCR: attribute value with unknown ID is skipped (line 144)
func TestGenerateCR_UnknownAttrIDSkipped(t *testing.T) {
	et := &models.EntityType{ID: "et1", Name: "Model"}
	inst := &models.EntityInstance{ID: "i1", Name: "inst"}
	attrs := []*models.Attribute{
		{ID: "a1", Name: "hostname", TypeDefinitionVersionID: "tdv-string"},
	}
	values := []*models.InstanceAttributeValue{
		{AttributeID: "a1", ValueString: "localhost"},
		{AttributeID: "unknown-id", ValueString: "should be skipped"},
	}

	cr, err := crdgen.GenerateCR(et, inst, values, attrs)
	require.NoError(t, err)
	assert.Equal(t, "localhost", cr.Spec["hostname"])
	_, exists := cr.Spec["unknown-id"]
	assert.False(t, exists)
}

// GenerateCR: JSON attribute value (replaces old enum test)
func TestGenerateCR_ValueJSONSerialization(t *testing.T) {
	et := &models.EntityType{ID: "et1", Name: "Model"}
	inst := &models.EntityInstance{ID: "i1", Name: "inst"}
	attrs := []*models.Attribute{
		{ID: "a1", Name: "status", TypeDefinitionVersionID: "tdv-1"},
	}
	values := []*models.InstanceAttributeValue{
		{AttributeID: "a1", ValueJSON: "active"},
	}

	cr, err := crdgen.GenerateCR(et, inst, values, attrs)
	require.NoError(t, err)
	assert.Equal(t, "active", cr.Spec["status"])
}
