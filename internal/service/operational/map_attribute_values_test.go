package operational

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
)

// Tests for mapAttributeValues — package-level function, needs internal test.

func ptrF(f float64) *float64 { return &f }

func TestMapAttributeValues_IntegerType(t *testing.T) {
	attrs := []*models.Attribute{
		{ID: "a1", Name: "count", Required: true},
	}
	num := float64(42)
	values := []*models.InstanceAttributeValue{
		{AttributeID: "a1", ValueNumber: &num},
	}
	baseTypes := map[string]string{"a1": "integer"}

	result := mapAttributeValues(attrs, values, baseTypes)
	assert.Len(t, result, 1)
	assert.Equal(t, "count", result[0].Name)
	assert.Equal(t, "integer", result[0].Type)
	assert.Equal(t, &num, result[0].Value)
	assert.True(t, result[0].Required)
}

func TestMapAttributeValues_BooleanType(t *testing.T) {
	attrs := []*models.Attribute{
		{ID: "a1", Name: "enabled"},
	}
	values := []*models.InstanceAttributeValue{
		{AttributeID: "a1", ValueString: "true"},
	}
	baseTypes := map[string]string{"a1": "boolean"}

	result := mapAttributeValues(attrs, values, baseTypes)
	assert.Len(t, result, 1)
	assert.Equal(t, "true", result[0].Value)
}

func TestMapAttributeValues_DateType(t *testing.T) {
	attrs := []*models.Attribute{
		{ID: "a1", Name: "created"},
	}
	values := []*models.InstanceAttributeValue{
		{AttributeID: "a1", ValueString: "2026-01-15"},
	}
	baseTypes := map[string]string{"a1": "date"}

	result := mapAttributeValues(attrs, values, baseTypes)
	assert.Len(t, result, 1)
	assert.Equal(t, "2026-01-15", result[0].Value)
}

func TestMapAttributeValues_URLType(t *testing.T) {
	attrs := []*models.Attribute{
		{ID: "a1", Name: "homepage"},
	}
	values := []*models.InstanceAttributeValue{
		{AttributeID: "a1", ValueString: "https://example.com"},
	}
	baseTypes := map[string]string{"a1": "url"}

	result := mapAttributeValues(attrs, values, baseTypes)
	assert.Len(t, result, 1)
	assert.Equal(t, "https://example.com", result[0].Value)
}

func TestMapAttributeValues_ListType(t *testing.T) {
	attrs := []*models.Attribute{
		{ID: "a1", Name: "tags"},
	}
	values := []*models.InstanceAttributeValue{
		{AttributeID: "a1", ValueJSON: `["alpha","beta"]`},
	}
	baseTypes := map[string]string{"a1": "list"}

	result := mapAttributeValues(attrs, values, baseTypes)
	assert.Len(t, result, 1)
	assert.Equal(t, `["alpha","beta"]`, result[0].Value)
}

func TestMapAttributeValues_ListTypeEmptyJSON(t *testing.T) {
	attrs := []*models.Attribute{
		{ID: "a1", Name: "tags"},
	}
	values := []*models.InstanceAttributeValue{
		{AttributeID: "a1", ValueJSON: ""},
	}
	baseTypes := map[string]string{"a1": "list"}

	result := mapAttributeValues(attrs, values, baseTypes)
	assert.Len(t, result, 1)
	assert.Nil(t, result[0].Value) // Empty JSON → nil
}

func TestMapAttributeValues_JSONType(t *testing.T) {
	attrs := []*models.Attribute{
		{ID: "a1", Name: "metadata"},
	}
	values := []*models.InstanceAttributeValue{
		{AttributeID: "a1", ValueJSON: `{"key":"val"}`},
	}
	baseTypes := map[string]string{"a1": "json"}

	result := mapAttributeValues(attrs, values, baseTypes)
	assert.Len(t, result, 1)
	assert.Equal(t, `{"key":"val"}`, result[0].Value)
}

func TestMapAttributeValues_FallbackToString(t *testing.T) {
	// When attribute ID is not in baseTypeByAttr, falls back to "string"
	attrs := []*models.Attribute{
		{ID: "a-missing", Name: "mystery"},
	}
	values := []*models.InstanceAttributeValue{
		{AttributeID: "a-missing", ValueString: "hello"},
	}
	baseTypes := map[string]string{} // empty — no type info

	result := mapAttributeValues(attrs, values, baseTypes)
	assert.Len(t, result, 1)
	assert.Equal(t, "string", result[0].Type) // fallback
	assert.Equal(t, "hello", result[0].Value) // string branch
}

func TestMapAttributeValues_NoMatchingValue(t *testing.T) {
	attrs := []*models.Attribute{
		{ID: "a1", Name: "hostname"},
	}
	values := []*models.InstanceAttributeValue{} // no values
	baseTypes := map[string]string{"a1": "string"}

	result := mapAttributeValues(attrs, values, baseTypes)
	assert.Len(t, result, 1)
	assert.Nil(t, result[0].Value)
}
