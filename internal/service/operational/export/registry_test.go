package export_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/project-catalyst/pc-asset-hub/internal/service/operational/export"
)

// T-34.01: Register exporter and retrieve by name
func TestRegistry_RegisterAndGet(t *testing.T) {
	r := export.NewExporterRegistry()
	e := &stubExporter{name: "test-exporter", desc: "A test exporter"}
	r.Register(e)

	got, ok := r.Get("test-exporter")
	require.True(t, ok)
	assert.Equal(t, "test-exporter", got.Name())
	assert.Equal(t, "A test exporter", got.Description())
}

// T-34.02: Get non-existent exporter returns not found
func TestRegistry_GetNonExistent(t *testing.T) {
	r := export.NewExporterRegistry()

	_, ok := r.Get("no-such")
	assert.False(t, ok)
}

// T-34.03: List empty registry returns empty slice
func TestRegistry_ListEmpty(t *testing.T) {
	r := export.NewExporterRegistry()

	items := r.List()
	assert.NotNil(t, items)
	assert.Empty(t, items)
}

// T-34.04: List registry with 2 exporters returns both with name, description, parameter schema
func TestRegistry_ListTwo(t *testing.T) {
	r := export.NewExporterRegistry()
	r.Register(&stubExporter{
		name: "alpha",
		desc: "Alpha exporter",
		params: []export.ParameterDef{
			{Name: "server_type", Type: "string", Required: true, Description: "Server entity type"},
		},
	})
	r.Register(&stubExporter{
		name: "beta",
		desc: "Beta exporter",
	})

	items := r.List()
	require.Len(t, items, 2)

	names := map[string]bool{}
	for _, item := range items {
		names[item.Name] = true
		if item.Name == "alpha" {
			assert.Equal(t, "Alpha exporter", item.Description)
			require.Len(t, item.ParameterSchema, 1)
			assert.Equal(t, "server_type", item.ParameterSchema[0].Name)
			assert.True(t, item.ParameterSchema[0].Required)
		}
	}
	assert.True(t, names["alpha"])
	assert.True(t, names["beta"])
}
