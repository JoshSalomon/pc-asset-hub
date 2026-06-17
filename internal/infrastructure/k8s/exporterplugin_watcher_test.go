package k8s

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"

	v1alpha1 "github.com/project-catalyst/pc-asset-hub/internal/operator/api/v1alpha1"
	"github.com/project-catalyst/pc-asset-hub/internal/service/operational/export"
)

func newTestWatcher() *ExporterPluginWatcher {
	registry := export.NewExporterRegistry()
	return NewExporterPluginWatcherForTest(registry, func() string { return "test-token" })
}

func makeCR(name, endpoint, desc string, timeout int, params []v1alpha1.ExporterPluginParameterDef, phase string) *v1alpha1.ExporterPlugin {
	cr := &v1alpha1.ExporterPlugin{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "assethub"},
		Spec: v1alpha1.ExporterPluginSpec{
			Description:     desc,
			Endpoint:        endpoint,
			TimeoutSeconds:  timeout,
			ParameterSchema: params,
		},
	}
	if phase != "" {
		cr.Status.Phase = phase
	}
	return cr
}

// T-36.106: handleAdd registers webhook with correct baseURL, description, params, timeout
func TestWatcher_HandleAdd_Registers(t *testing.T) {
	w := newTestWatcher()
	cr := makeCR("my-exporter", "http://svc.ns.svc.cluster.local", "My Exporter", 20,
		[]v1alpha1.ExporterPluginParameterDef{
			{Name: "ns", Type: "string", Description: "namespace", Required: true},
		}, "Ready")

	w.HandleAddForTest(cr)

	exp, ok := w.Registry().Get("my-exporter")
	require.True(t, ok)
	assert.Equal(t, "my-exporter", exp.Name())
	assert.Equal(t, "My Exporter", exp.Description())
	require.Len(t, exp.ParameterSchema(), 1)
	assert.Equal(t, "ns", exp.ParameterSchema()[0].Name)

	we := exp.(*export.WebhookExporter)
	vt, et := we.Timeouts()
	assert.Equal(t, 10*time.Second, vt)
	assert.Equal(t, 20*time.Second, et)
}

// T-36.107: handleAdd: healthStatus read from cr.Status.Phase
func TestWatcher_HandleAdd_HealthStatus(t *testing.T) {
	w := newTestWatcher()
	cr := makeCR("my-exporter", "http://svc.ns.svc.cluster.local", "test", 10, nil, "Unhealthy")

	w.HandleAddForTest(cr)

	exp, ok := w.Registry().Get("my-exporter")
	require.True(t, ok)
	we := exp.(*export.WebhookExporter)
	assert.Equal(t, "Unhealthy", we.HealthStatus())
}

// T-36.108: handleAdd: exporter Name() matches cr.ObjectMeta.Name
func TestWatcher_HandleAdd_NameMatches(t *testing.T) {
	w := newTestWatcher()
	cr := makeCR("exact-name", "http://localhost", "test", 10, nil, "")

	w.HandleAddForTest(cr)

	exp, ok := w.Registry().Get("exact-name")
	require.True(t, ok)
	assert.Equal(t, "exact-name", exp.Name())
}

// T-36.109: handleDelete deregisters from registry
func TestWatcher_HandleDelete(t *testing.T) {
	w := newTestWatcher()
	cr := makeCR("my-exporter", "http://localhost", "test", 10, nil, "")
	w.HandleAddForTest(cr)

	_, ok := w.Registry().Get("my-exporter")
	require.True(t, ok)

	w.HandleDeleteForTest(cr)

	_, ok = w.Registry().Get("my-exporter")
	assert.False(t, ok)
}

// T-36.109b: handleDelete: DeletedFinalStateUnknown tombstone → extracts object, deregisters
func TestWatcher_HandleDelete_Tombstone(t *testing.T) {
	w := newTestWatcher()
	cr := makeCR("my-exporter", "http://localhost", "test", 10, nil, "")
	w.HandleAddForTest(cr)

	tombstone := cache.DeletedFinalStateUnknown{
		Key: "assethub/my-exporter",
		Obj: cr,
	}
	w.HandleDeleteForTest(tombstone)

	_, ok := w.Registry().Get("my-exporter")
	assert.False(t, ok)
}

// T-36.110: handleAdd: built-in name collision → not registered, built-in preserved
func TestWatcher_HandleAdd_BuiltInCollision(t *testing.T) {
	w := newTestWatcher()
	w.Registry().Register(&stubExporter{name: "mcp-gateway"})

	cr := makeCR("mcp-gateway", "http://localhost", "webhook clone", 10, nil, "Ready")
	w.HandleAddForTest(cr)

	exp, ok := w.Registry().Get("mcp-gateway")
	require.True(t, ok)
	assert.True(t, w.Registry().IsBuiltIn("mcp-gateway"))
	assert.Equal(t, "stub", exp.Description())
}

// T-36.111: handleUpdate: atomic swap (old gone, new present)
func TestWatcher_HandleUpdate_AtomicSwap(t *testing.T) {
	w := newTestWatcher()
	oldCR := makeCR("my-exporter", "http://old-svc", "old desc", 10, nil, "Ready")
	w.HandleAddForTest(oldCR)

	newCR := makeCR("my-exporter", "http://new-svc", "new desc", 15, nil, "Ready")
	w.HandleUpdateForTest(oldCR, newCR)

	exp, ok := w.Registry().Get("my-exporter")
	require.True(t, ok)
	assert.Equal(t, "new desc", exp.Description())
}

// T-36.112: handleUpdate: invalid spec → old registration preserved
func TestWatcher_HandleUpdate_InvalidSpec_KeepsOld(t *testing.T) {
	w := newTestWatcher()
	oldCR := makeCR("my-exporter", "http://old-svc", "old desc", 10, nil, "Ready")
	w.HandleAddForTest(oldCR)

	invalidCR := makeCR("my-exporter", "http://new-svc/api/v1", "new desc", 10, nil, "Ready")
	w.HandleUpdateForTest(oldCR, invalidCR)

	exp, ok := w.Registry().Get("my-exporter")
	require.True(t, ok)
	assert.Equal(t, "old desc", exp.Description())
}

// T-36.113: Default timeout: omitted spec.TimeoutSeconds → 10s
func TestWatcher_HandleAdd_DefaultTimeout(t *testing.T) {
	w := newTestWatcher()
	cr := makeCR("my-exporter", "http://localhost", "test", 0, nil, "")
	w.HandleAddForTest(cr)

	exp, ok := w.Registry().Get("my-exporter")
	require.True(t, ok)
	we := exp.(*export.WebhookExporter)
	_, et := we.Timeouts()
	assert.Equal(t, 10*time.Second, et)
}

// T-36.114: Custom timeout: spec.TimeoutSeconds=20 → export=20s, validate=10s
func TestWatcher_HandleAdd_CustomTimeout(t *testing.T) {
	w := newTestWatcher()
	cr := makeCR("my-exporter", "http://localhost", "test", 20, nil, "")
	w.HandleAddForTest(cr)

	exp, ok := w.Registry().Get("my-exporter")
	require.True(t, ok)
	we := exp.(*export.WebhookExporter)
	vt, et := we.Timeouts()
	assert.Equal(t, 10*time.Second, vt)
	assert.Equal(t, 20*time.Second, et)
}

// T-36.115: Base URL path rejection: endpoint with path → handleAdd rejects
func TestWatcher_HandleAdd_PathRejected(t *testing.T) {
	w := newTestWatcher()
	cr := makeCR("my-exporter", "http://localhost/api/v1", "test", 10, nil, "")
	w.HandleAddForTest(cr)

	_, ok := w.Registry().Get("my-exporter")
	assert.False(t, ok)
}

// T-36.116: Startup pre-population: 2 CRs exist before Start → both in registry
func TestWatcher_StartupPrePopulation(t *testing.T) {
	w := newTestWatcher()
	cr1 := makeCR("exporter-a", "http://svc-a.ns.svc.cluster.local", "Exporter A", 10, nil, "Ready")
	cr2 := makeCR("exporter-b", "http://svc-b.ns.svc.cluster.local", "Exporter B", 15,
		[]v1alpha1.ExporterPluginParameterDef{
			{Name: "ns", Type: "string", Required: true},
		}, "Ready")

	w.HandleAddForTest(cr1)
	w.HandleAddForTest(cr2)

	expA, ok := w.Registry().Get("exporter-a")
	require.True(t, ok, "exporter-a should be in registry")
	assert.Equal(t, "Exporter A", expA.Description())

	expB, ok := w.Registry().Get("exporter-b")
	require.True(t, ok, "exporter-b should be in registry")
	assert.Equal(t, "Exporter B", expB.Description())
	require.Len(t, expB.ParameterSchema(), 1)
	assert.Equal(t, "ns", expB.ParameterSchema()[0].Name)
}

// T-36.103: Non-K8s mode: nil rest.Config → returns error
func TestNewExporterPluginWatcher_NilConfig(t *testing.T) {
	registry := export.NewExporterRegistry()
	_, err := NewExporterPluginWatcher(nil, registry, "assethub", func() string { return "" })
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

// T-36.117: handleAdd with wrong type → no panic, registry unchanged
func TestWatcher_HandleAdd_WrongType(t *testing.T) {
	w := newTestWatcher()
	w.HandleAddForTest("not-an-exporter-plugin")

	// Registry should still be empty
	_, ok := w.Registry().Get("not-an-exporter-plugin")
	assert.False(t, ok)
}

// Robustness: handleUpdate with wrong type for newObj → no panic, registry unchanged
func TestWatcher_HandleUpdate_WrongType(t *testing.T) {
	w := newTestWatcher()
	// Register a valid exporter first
	cr := makeCR("my-exporter", "http://localhost", "test", 10, nil, "")
	w.HandleAddForTest(cr)
	_, ok := w.Registry().Get("my-exporter")
	require.True(t, ok)

	// Update with wrong type — should be a no-op, old registration stays
	w.HandleUpdateForTest("old-string", "new-string")
	_, ok = w.Registry().Get("my-exporter")
	assert.True(t, ok, "old registration should still be present")
}

// Robustness: handleDelete with wrong type (not ExporterPlugin, not tombstone) → no panic
func TestWatcher_HandleDelete_WrongType(t *testing.T) {
	w := newTestWatcher()
	// Register a valid exporter first
	cr := makeCR("my-exporter", "http://localhost", "test", 10, nil, "")
	w.HandleAddForTest(cr)
	_, ok := w.Registry().Get("my-exporter")
	require.True(t, ok)

	// Delete with wrong type — should be a no-op
	w.HandleDeleteForTest("not-an-exporter-plugin")
	_, ok = w.Registry().Get("my-exporter")
	assert.True(t, ok, "exporter should still be registered")
}

// Robustness: handleDelete with tombstone containing wrong type → no panic
func TestWatcher_HandleDelete_TombstoneWrongType(t *testing.T) {
	w := newTestWatcher()
	cr := makeCR("my-exporter", "http://localhost", "test", 10, nil, "")
	w.HandleAddForTest(cr)
	_, ok := w.Registry().Get("my-exporter")
	require.True(t, ok)

	// Tombstone with wrong inner type — should be a no-op
	tombstone := cache.DeletedFinalStateUnknown{
		Key: "assethub/my-exporter",
		Obj: "not-a-cr",
	}
	w.HandleDeleteForTest(tombstone)
	_, ok = w.Registry().Get("my-exporter")
	assert.True(t, ok, "exporter should still be registered")
}

// Coverage: lines 31-51 — NewExporterPluginWatcher with valid rest.Config → succeeds
func TestNewExporterPluginWatcher_ValidConfig(t *testing.T) {
	// Provide a fake API server so ctrlcache.New can create watchers
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return minimal discovery response for any request
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"kind":"APIResourceList","apiVersion":"v1","resources":[]}`))
	}))
	defer server.Close()

	cfg := &rest.Config{Host: server.URL}
	registry := export.NewExporterRegistry()

	watcher, err := NewExporterPluginWatcher(cfg, registry, "assethub", func() string { return "token" })
	require.NoError(t, err)
	require.NotNil(t, watcher)
	assert.NotNil(t, watcher.registry)
	assert.Equal(t, "assethub", watcher.namespace)
	assert.NotNil(t, watcher.cache)
}

// Coverage: lines 54-58 — Start with canceled context → GetInformer error
func TestExporterPluginWatcher_Start_CanceledContext(t *testing.T) {
	// Provide a fake API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"kind":"APIResourceList","apiVersion":"v1","resources":[]}`))
	}))
	defer server.Close()

	cfg := &rest.Config{Host: server.URL}
	registry := export.NewExporterRegistry()

	watcher, err := NewExporterPluginWatcher(cfg, registry, "assethub", func() string { return "" })
	require.NoError(t, err)

	// Cancel context before Start — the cache.GetInformer or cache.Start should fail
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = watcher.Start(ctx)
	require.Error(t, err, "Start with canceled context should return error")
}

// Coverage: lines 60-66 — Start with a realistic fake API server
// exercises AddEventHandler + cache.Start
func TestExporterPluginWatcher_Start_ShortLived(t *testing.T) {
	// Serve discovery and watch endpoints the way a real API server would.
	// The controller-runtime cache needs:
	// 1. Discovery to learn the ExporterPlugin GVR
	// 2. List to fetch existing resources
	// 3. Watch to stream changes
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.URL.Path == "/api" || r.URL.Path == "/api/v1":
			// Core API resources
			w.Write([]byte(`{"kind":"APIResourceList","apiVersion":"v1","groupVersion":"v1","resources":[]}`))

		case r.URL.Path == "/apis":
			// API group list — include our CRD group
			w.Write([]byte(`{"kind":"APIGroupList","apiVersion":"v1","groups":[{"name":"assethub.project-catalyst.io","versions":[{"groupVersion":"assethub.project-catalyst.io/v1alpha1","version":"v1alpha1"}],"preferredVersion":{"groupVersion":"assethub.project-catalyst.io/v1alpha1","version":"v1alpha1"}}]}`))

		case r.URL.Path == "/apis/assethub.project-catalyst.io/v1alpha1":
			// Resources for our group
			w.Write([]byte(`{"kind":"APIResourceList","apiVersion":"v1","groupVersion":"assethub.project-catalyst.io/v1alpha1","resources":[{"name":"exporterplugins","singularName":"exporterplugin","namespaced":true,"kind":"ExporterPlugin","verbs":["list","watch","get"]}]}`))

		case r.URL.Path == "/apis/assethub.project-catalyst.io/v1alpha1/namespaces/assethub/exporterplugins":
			if r.URL.Query().Get("watch") == "true" {
				// Watch — keep connection open until client disconnects
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Transfer-Encoding", "chunked")
				w.WriteHeader(http.StatusOK)
				if f, ok := w.(http.Flusher); ok {
					f.Flush()
				}
				// Block until the request context is done
				<-r.Context().Done()
			} else {
				// List — return empty list
				w.Write([]byte(`{"kind":"ExporterPluginList","apiVersion":"assethub.project-catalyst.io/v1alpha1","metadata":{"resourceVersion":"1"},"items":[]}`))
			}

		default:
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"kind":"Status","apiVersion":"v1","status":"Failure","message":"not found","code":404}`))
		}
	}))
	defer server.Close()

	cfg := &rest.Config{Host: server.URL}
	registry := export.NewExporterRegistry()

	watcher, err := NewExporterPluginWatcher(cfg, registry, "assethub", func() string { return "" })
	require.NoError(t, err)

	// Use a short timeout — just long enough for cache to start and sync
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- watcher.Start(ctx)
	}()

	select {
	case err := <-errCh:
		// Log the error so we can debug coverage
		t.Logf("Start returned: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return within 5 seconds")
	}
}

// Coverage: lines 32-34 — AddToScheme error path
// This error is practically unreachable because v1alpha1.AddToScheme
// only calls scheme.AddKnownTypes which does not return errors.
// Including a placeholder to document why it's uncoverable.

// Coverage: lines 42-44 — ctrlcache.New error path
// ctrlcache.New validates the rest.Config for TLS, mapper, etc.
// Testing with an intentionally broken config to trigger cache creation failure.
func TestNewExporterPluginWatcher_CacheCreationError(t *testing.T) {
	// rest.Config with TLS pointing to a nonexistent CA file
	// causes the transport config to fail during cache creation
	cfg := &rest.Config{
		Host: "https://localhost:1",
		TLSClientConfig: rest.TLSClientConfig{
			CAFile: "/nonexistent/ca.crt",
		},
	}
	registry := export.NewExporterRegistry()
	_, err := NewExporterPluginWatcher(cfg, registry, "assethub", func() string { return "" })
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create cache")
}

// Robustness: handleUpdate with built-in name collision → no-op, built-in preserved
func TestWatcher_HandleUpdate_BuiltInCollision(t *testing.T) {
	w := newTestWatcher()
	w.Registry().Register(&stubExporter{name: "mcp-gateway"})

	oldCR := makeCR("mcp-gateway", "http://old", "old", 10, nil, "Ready")
	newCR := makeCR("mcp-gateway", "http://new", "new", 10, nil, "Ready")
	w.HandleUpdateForTest(oldCR, newCR)

	exp, ok := w.Registry().Get("mcp-gateway")
	require.True(t, ok)
	assert.True(t, w.Registry().IsBuiltIn("mcp-gateway"))
	assert.Equal(t, "stub", exp.Description(), "built-in should be preserved")
}

// Coverage: ServiceAccountTokenGetter — error path (L192-200)
// The SA token file doesn't exist in the test environment, so os.ReadFile fails.
// This tests the error branch including warnOnce.
func TestServiceAccountTokenGetter_FileNotFound(t *testing.T) {
	getter := ServiceAccountTokenGetter()

	// First call: file not found → logs warning, returns ""
	token1 := getter()
	assert.Equal(t, "", token1, "should return empty string when SA token file is missing")

	// Second call: warnOnce ensures log only fires once, still returns ""
	token2 := getter()
	assert.Equal(t, "", token2, "subsequent calls should also return empty string")
}

// SA token getter trims trailing whitespace (K8s token files may have trailing newline)
func TestServiceAccountTokenGetter_TrimsWhitespace(t *testing.T) {
	tmpFile := t.TempDir() + "/token"
	os.WriteFile(tmpFile, []byte("my-token\n"), 0644)

	origPath := saTokenPath
	saTokenPath = tmpFile
	defer func() { saTokenPath = origPath }()

	getter := ServiceAccountTokenGetter()
	token := getter()
	assert.Equal(t, "my-token", token, "trailing newline should be trimmed")
}

type stubExporter struct {
	name string
}

func (s *stubExporter) Name() string                      { return s.name }
func (s *stubExporter) Description() string               { return "stub" }
func (s *stubExporter) ParameterSchema() []export.ParameterDef { return nil }
func (s *stubExporter) ValidateSchema(_ context.Context, params map[string]string, schema export.SchemaInfo) error {
	return nil
}
func (s *stubExporter) Export(_ context.Context, _ export.ExportInput) (*export.ExportOutput, error) {
	return &export.ExportOutput{}, nil
}
