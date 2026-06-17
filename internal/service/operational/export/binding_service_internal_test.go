package export

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
)

func TestExecuteBinding_DeregisteredExporter_NoPanel(t *testing.T) {
	registry := NewExporterRegistry()
	svc := &ExportBindingService{registry: registry}

	catalog := &models.Catalog{ID: "cat1", CatalogVersionID: "cv1"}
	binding := &models.ExportBinding{
		ID: "b1", CatalogID: "cat1", ExporterName: "gone-exporter",
		Parameters: map[string]string{}, Enabled: true,
	}

	result := svc.executeBinding(context.Background(), catalog, binding)
	assert.Equal(t, BindingStatusFailed, result.Status)
	assert.Contains(t, result.Error, "not registered")
}

func TestResolveAttributes_JSONValueParsed(t *testing.T) {
	mockIAV := &mockIAVRepo{}
	mockIAV.vals = []*models.InstanceAttributeValue{
		{AttributeID: "a1", ValueJSON: `["tag1","tag2"]`},
		{AttributeID: "a2", ValueJSON: `{"key":"val"}`},
		{AttributeID: "a3", ValueString: "plain"},
	}
	svc := &ExportBindingService{iavRepo: mockIAV}

	attrs := []*models.Attribute{
		{ID: "a1", Name: "tags"},
		{ID: "a2", Name: "config"},
		{ID: "a3", Name: "name"},
	}
	inst := &models.EntityInstance{ID: "inst1", Version: 1}

	result, err := svc.resolveAttributes(context.Background(), inst, attrs)
	require.NoError(t, err)

	// JSON values must be parsed objects, not raw strings
	// If they're strings, json.Marshal will double-encode them
	data, _ := json.Marshal(result)
	raw := string(data)

	// tags should be a JSON array, not a quoted string
	assert.Contains(t, raw, `"tags":["tag1","tag2"]`)
	assert.NotContains(t, raw, `"tags":"[`)

	// config should be a JSON object, not a quoted string
	assert.Contains(t, raw, `"config":{"key":"val"}`)
	assert.NotContains(t, raw, `"config":"{`)

	// plain string stays as string
	assert.Contains(t, raw, `"name":"plain"`)
}

type mockIAVRepo struct {
	vals []*models.InstanceAttributeValue
}

func (m *mockIAVRepo) SetValues(_ context.Context, _ []*models.InstanceAttributeValue) error { return nil }
func (m *mockIAVRepo) GetValuesForVersion(_ context.Context, _ string, _ int) ([]*models.InstanceAttributeValue, error) {
	return m.vals, nil
}
func (m *mockIAVRepo) DeleteByInstanceID(_ context.Context, _ string) error                      { return nil }
func (m *mockIAVRepo) DeleteByInstanceVersion(_ context.Context, _ string, _ int) error          { return nil }
func (m *mockIAVRepo) RemapAttributeIDs(_ context.Context, _ []string, _ map[string]string) (int64, error) { return 0, nil }
