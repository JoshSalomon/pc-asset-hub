package export_test

import (
	"fmt"
	"sync"
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
			{Name: "server_type", Type: "entity_type", Required: true, Description: "Server entity type"},
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

// T-36.06: Register marks exporter as built-in
func TestRegistry_Register_MarksBuiltIn(t *testing.T) {
	r := export.NewExporterRegistry()
	r.Register(&stubExporter{name: "mcp-gateway"})

	assert.True(t, r.IsBuiltIn("mcp-gateway"))
}

// T-36.07: RegisterWebhook marks exporter as NOT built-in
func TestRegistry_RegisterWebhook_NotBuiltIn(t *testing.T) {
	r := export.NewExporterRegistry()
	r.RegisterWebhook(&stubExporter{name: "my-webhook"})

	got, ok := r.Get("my-webhook")
	require.True(t, ok)
	assert.Equal(t, "my-webhook", got.Name())
	assert.False(t, r.IsBuiltIn("my-webhook"))
}

// T-36.08: Deregister succeeds for webhook exporter
func TestRegistry_Deregister_Webhook(t *testing.T) {
	r := export.NewExporterRegistry()
	r.RegisterWebhook(&stubExporter{name: "my-webhook"})

	ok := r.Deregister("my-webhook")
	assert.True(t, ok)

	_, found := r.Get("my-webhook")
	assert.False(t, found)
}

// T-36.09: Deregister refused for built-in exporter
func TestRegistry_Deregister_BuiltIn_Refused(t *testing.T) {
	r := export.NewExporterRegistry()
	r.Register(&stubExporter{name: "mcp-gateway"})

	ok := r.Deregister("mcp-gateway")
	assert.False(t, ok)

	got, found := r.Get("mcp-gateway")
	assert.True(t, found)
	assert.Equal(t, "mcp-gateway", got.Name())
}

// T-36.10: Deregister returns false for non-existent name
func TestRegistry_Deregister_NotFound(t *testing.T) {
	r := export.NewExporterRegistry()

	ok := r.Deregister("no-such")
	assert.False(t, ok)
}

// T-36.11: RegisterWebhook twice overwrites (last wins)
func TestRegistry_RegisterWebhook_Overwrites(t *testing.T) {
	r := export.NewExporterRegistry()
	r.RegisterWebhook(&stubExporter{name: "my-webhook", desc: "first"})
	r.RegisterWebhook(&stubExporter{name: "my-webhook", desc: "second"})

	got, ok := r.Get("my-webhook")
	require.True(t, ok)
	assert.Equal(t, "second", got.Description())
}

// T-36.12: Concurrent access — 50 goroutines doing Register/Get/List/Deregister — no race
func TestRegistry_ConcurrentAccess(t *testing.T) {
	r := export.NewExporterRegistry()
	r.Register(&stubExporter{name: "built-in"})

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			name := fmt.Sprintf("webhook-%d", i)
			r.RegisterWebhook(&stubExporter{name: name})
			r.Get(name)
			r.List()
			r.IsBuiltIn(name)
			r.Deregister(name)
		}(i)
	}
	wg.Wait()

	// built-in should survive all concurrent operations
	got, ok := r.Get("built-in")
	require.True(t, ok)
	assert.Equal(t, "built-in", got.Name())
}

// T-36.47: List with built-in exporter → source="built-in", health="n/a"
func TestRegistry_List_BuiltIn_SourceHealth(t *testing.T) {
	r := export.NewExporterRegistry()
	r.Register(&stubExporter{name: "mcp-gateway", desc: "MCP Gateway"})

	items := r.List()
	require.Len(t, items, 1)
	assert.Equal(t, "built-in", items[0].Source)
	assert.Equal(t, "n/a", items[0].Health)
}

// T-36.48: List with webhook exporter (health "Ready") → source="webhook", health="Ready"
func TestRegistry_List_Webhook_SourceHealth(t *testing.T) {
	r := export.NewExporterRegistry()
	we, err := export.NewWebhookExporter("my-webhook", "Webhook", "http://localhost", nil, 10, func() string { return "" })
	require.NoError(t, err)
	we.SetHealthStatus("Ready")
	r.RegisterWebhook(we)

	items := r.List()
	require.Len(t, items, 1)
	assert.Equal(t, "webhook", items[0].Source)
	assert.Equal(t, "Ready", items[0].Health)
}

// T-36.49: List with mixed built-in + webhook → correct source/health for each
func TestRegistry_List_Mixed_SourceHealth(t *testing.T) {
	r := export.NewExporterRegistry()
	r.Register(&stubExporter{name: "built-in-exp"})
	we, err := export.NewWebhookExporter("webhook-exp", "Webhook", "http://localhost", nil, 10, func() string { return "" })
	require.NoError(t, err)
	we.SetHealthStatus("Unhealthy")
	r.RegisterWebhook(we)

	items := r.List()
	require.Len(t, items, 2)
	for _, item := range items {
		if item.Name == "built-in-exp" {
			assert.Equal(t, "built-in", item.Source)
			assert.Equal(t, "n/a", item.Health)
		} else {
			assert.Equal(t, "webhook", item.Source)
			assert.Equal(t, "Unhealthy", item.Health)
		}
	}
}

// T-36.100: Register two webhook exporters → both in List
func TestRegistry_TwoWebhooks_BothInList(t *testing.T) {
	r := export.NewExporterRegistry()
	we1, err := export.NewWebhookExporter("webhook-alpha", "Alpha", "http://localhost:8001", nil, 10, func() string { return "" })
	require.NoError(t, err)
	we1.SetHealthStatus("Ready")
	r.RegisterWebhook(we1)

	we2, err := export.NewWebhookExporter("webhook-beta", "Beta", "http://localhost:8002", nil, 10, func() string { return "" })
	require.NoError(t, err)
	we2.SetHealthStatus("Unhealthy")
	r.RegisterWebhook(we2)

	items := r.List()
	require.Len(t, items, 2)

	names := map[string]bool{}
	for _, item := range items {
		names[item.Name] = true
		assert.Equal(t, "webhook", item.Source)
	}
	assert.True(t, names["webhook-alpha"])
	assert.True(t, names["webhook-beta"])
}

// T-36.101: Deregister one webhook → other unaffected
func TestRegistry_DeregisterOne_OtherUnaffected(t *testing.T) {
	r := export.NewExporterRegistry()
	we1, err := export.NewWebhookExporter("webhook-alpha", "Alpha", "http://localhost:8001", nil, 10, func() string { return "" })
	require.NoError(t, err)
	r.RegisterWebhook(we1)

	we2, err := export.NewWebhookExporter("webhook-beta", "Beta", "http://localhost:8002", nil, 10, func() string { return "" })
	require.NoError(t, err)
	r.RegisterWebhook(we2)

	ok := r.Deregister("webhook-alpha")
	assert.True(t, ok)

	_, foundAlpha := r.Get("webhook-alpha")
	assert.False(t, foundAlpha, "webhook-alpha should be deregistered")

	got, foundBeta := r.Get("webhook-beta")
	assert.True(t, foundBeta, "webhook-beta should still be registered")
	assert.Equal(t, "webhook-beta", got.Name())

	items := r.List()
	assert.Len(t, items, 1)
	assert.Equal(t, "webhook-beta", items[0].Name)
}
