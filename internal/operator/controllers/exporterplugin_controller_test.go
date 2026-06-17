package controllers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	v1alpha1 "github.com/project-catalyst/pc-asset-hub/internal/operator/api/v1alpha1"
)

// T-36.65: classifyHealthResult: 200 → Ready
func TestClassifyHealthResult_200_Ready(t *testing.T) {
	phase, msg := classifyHealthResult(200, nil)
	assert.Equal(t, "Ready", phase)
	assert.Empty(t, msg)
}

// T-36.66: classifyHealthResult: 401 → Error "authentication rejected"
func TestClassifyHealthResult_401_Error(t *testing.T) {
	phase, msg := classifyHealthResult(401, nil)
	assert.Equal(t, "Error", phase)
	assert.Contains(t, msg, "authentication rejected")
}

// T-36.67: classifyHealthResult: 403 → Error "authentication rejected"
func TestClassifyHealthResult_403_Error(t *testing.T) {
	phase, msg := classifyHealthResult(403, nil)
	assert.Equal(t, "Error", phase)
	assert.Contains(t, msg, "authentication rejected")
}

// T-36.68: classifyHealthResult: 503 → Unhealthy
func TestClassifyHealthResult_503_Unhealthy(t *testing.T) {
	phase, msg := classifyHealthResult(503, nil)
	assert.Equal(t, "Unhealthy", phase)
	assert.Contains(t, msg, "503")
}

// T-36.69: classifyHealthResult: timeout → Unhealthy
func TestClassifyHealthResult_Timeout_Unhealthy(t *testing.T) {
	phase, msg := classifyHealthResult(0, context.DeadlineExceeded)
	assert.Equal(t, "Unhealthy", phase)
	assert.Contains(t, msg, "timed out")
}

type timeoutError struct{ msg string }

func (e *timeoutError) Error() string   { return e.msg }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return false }

// QR: classifyHealthResult: http.Client timeout (url.Error) → Unhealthy
func TestClassifyHealthResult_HTTPClientTimeout_Unhealthy(t *testing.T) {
	urlErr := &url.Error{Op: "Post", URL: "http://plugin/health", Err: &timeoutError{"Client.Timeout exceeded"}}
	phase, msg := classifyHealthResult(0, urlErr)
	assert.Equal(t, "Unhealthy", phase)
	assert.Contains(t, msg, "timed out")
}

// T-36.70: classifyHealthResult: connection error → Error
func TestClassifyHealthResult_ConnectionError(t *testing.T) {
	phase, msg := classifyHealthResult(0, errors.New("dial tcp: connection refused"))
	assert.Equal(t, "Error", phase)
	assert.Contains(t, msg, "unreachable")
}

func setupReconciler(t *testing.T, objects ...runtime.Object) (*ExporterPluginReconciler, *runtime.Scheme) {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	builder := fake.NewClientBuilder().WithScheme(scheme).
		WithStatusSubresource(&v1alpha1.ExporterPlugin{})
	for _, obj := range objects {
		builder = builder.WithRuntimeObjects(obj)
	}
	c := builder.Build()

	return &ExporterPluginReconciler{
		Client:        c,
		Scheme:        scheme,
		HTTPClient:    &http.Client{Timeout: 2 * time.Second},
		ProbeInterval: 30 * time.Second,
		TokenGetter:   func() string { return "test-token" },
	}, scheme
}

// T-36.71: Reconcile: CR not found → no error
func TestExporterPluginReconcile_NotFound(t *testing.T) {
	r, _ := setupReconciler(t)
	result, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "no-such", Namespace: "assethub"},
	})
	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
}

// T-36.72: Reconcile: healthy plugin (httptest 200) → status.phase=Ready
func TestExporterPluginReconcile_Healthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	ep := &v1alpha1.ExporterPlugin{
		ObjectMeta: metav1.ObjectMeta{Name: "test-plugin", Namespace: "assethub"},
		Spec:       v1alpha1.ExporterPluginSpec{Endpoint: server.URL},
	}
	r, _ := setupReconciler(t, ep)

	result, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-plugin", Namespace: "assethub"},
	})
	require.NoError(t, err)
	assert.Equal(t, 30*time.Second, result.RequeueAfter)

	var updated v1alpha1.ExporterPlugin
	require.NoError(t, r.Get(context.Background(), types.NamespacedName{Name: "test-plugin", Namespace: "assethub"}, &updated))
	assert.Equal(t, "Ready", updated.Status.Phase)
}

// T-36.73: Reconcile: unhealthy plugin (httptest 503) → status.phase=Unhealthy
func TestExporterPluginReconcile_Unhealthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	ep := &v1alpha1.ExporterPlugin{
		ObjectMeta: metav1.ObjectMeta{Name: "test-plugin", Namespace: "assethub"},
		Spec:       v1alpha1.ExporterPluginSpec{Endpoint: server.URL},
	}
	r, _ := setupReconciler(t, ep)

	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-plugin", Namespace: "assethub"},
	})
	require.NoError(t, err)

	var updated v1alpha1.ExporterPlugin
	require.NoError(t, r.Get(context.Background(), types.NamespacedName{Name: "test-plugin", Namespace: "assethub"}, &updated))
	assert.Equal(t, "Unhealthy", updated.Status.Phase)
}

// T-36.74: Reconcile: unreachable plugin → status.phase=Error
func TestExporterPluginReconcile_Unreachable(t *testing.T) {
	ep := &v1alpha1.ExporterPlugin{
		ObjectMeta: metav1.ObjectMeta{Name: "test-plugin", Namespace: "assethub"},
		Spec:       v1alpha1.ExporterPluginSpec{Endpoint: "http://127.0.0.1:1"},
	}
	r, _ := setupReconciler(t, ep)

	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-plugin", Namespace: "assethub"},
	})
	require.NoError(t, err)

	var updated v1alpha1.ExporterPlugin
	require.NoError(t, r.Get(context.Background(), types.NamespacedName{Name: "test-plugin", Namespace: "assethub"}, &updated))
	assert.Equal(t, "Error", updated.Status.Phase)
}

// T-36.75: Reconcile: sends Authorization header on health probe
func TestExporterPluginReconcile_AuthHeader(t *testing.T) {
	var capturedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	ep := &v1alpha1.ExporterPlugin{
		ObjectMeta: metav1.ObjectMeta{Name: "test-plugin", Namespace: "assethub"},
		Spec:       v1alpha1.ExporterPluginSpec{Endpoint: server.URL},
	}
	r, _ := setupReconciler(t, ep)

	r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-plugin", Namespace: "assethub"},
	})

	assert.Equal(t, "Bearer test-token", capturedAuth)
}

// T-36.77: Stale-to-Unknown: lastHealthCheck older than 3x probe interval → Unknown
func TestExporterPluginReconcile_StaleToUnknown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	staleTime := time.Now().Add(-5 * time.Minute).UTC().Format(time.RFC3339)
	ep := &v1alpha1.ExporterPlugin{
		ObjectMeta: metav1.ObjectMeta{Name: "test-plugin", Namespace: "assethub"},
		Spec:       v1alpha1.ExporterPluginSpec{Endpoint: server.URL},
		Status: v1alpha1.ExporterPluginStatus{
			Phase:           "Ready",
			LastHealthCheck: staleTime,
		},
	}
	r, _ := setupReconciler(t, ep)

	r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-plugin", Namespace: "assethub"},
	})

	var updated v1alpha1.ExporterPlugin
	require.NoError(t, r.Get(context.Background(), types.NamespacedName{Name: "test-plugin", Namespace: "assethub"}, &updated))
	assert.Equal(t, "Unknown", updated.Status.Phase)
	assert.Contains(t, updated.Status.Message, "stale")
}

// T-36.76: Health transitions: Ready → Unhealthy (503) → Ready (200)
func TestExporterPluginReconcile_HealthTransitions(t *testing.T) {
	var callCount int
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		count := callCount
		callCount++
		mu.Unlock()

		switch count {
		case 0:
			// First probe: healthy
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		case 1:
			// Second probe: unhealthy
			w.WriteHeader(http.StatusServiceUnavailable)
		case 2:
			// Third probe: healthy again
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		default:
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		}
	}))
	defer server.Close()

	ep := &v1alpha1.ExporterPlugin{
		ObjectMeta: metav1.ObjectMeta{Name: "transition-plugin", Namespace: "assethub"},
		Spec:       v1alpha1.ExporterPluginSpec{Endpoint: server.URL},
	}
	r, _ := setupReconciler(t, ep)
	nsName := types.NamespacedName{Name: "transition-plugin", Namespace: "assethub"}
	req := ctrl.Request{NamespacedName: nsName}

	// Step 1: Reconcile → should be Ready (200)
	result, err := r.Reconcile(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, 30*time.Second, result.RequeueAfter)

	var updated v1alpha1.ExporterPlugin
	require.NoError(t, r.Get(context.Background(), nsName, &updated))
	assert.Equal(t, "Ready", updated.Status.Phase, "first reconcile should be Ready")

	// Step 2: Reconcile → should be Unhealthy (503)
	result, err = r.Reconcile(context.Background(), req)
	require.NoError(t, err)

	require.NoError(t, r.Get(context.Background(), nsName, &updated))
	assert.Equal(t, "Unhealthy", updated.Status.Phase, "second reconcile should be Unhealthy")
	assert.Contains(t, updated.Status.Message, "503")

	// Step 3: Reconcile → should be Ready again (200)
	result, err = r.Reconcile(context.Background(), req)
	require.NoError(t, err)

	require.NoError(t, r.Get(context.Background(), nsName, &updated))
	assert.Equal(t, "Ready", updated.Status.Phase, "third reconcile should be Ready again")
}

// Coverage: line 33 — r.Get returns non-NotFound error → returned to caller
func TestExporterPluginReconcile_GetError(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	cl := fake.NewClientBuilder().WithScheme(scheme).
		WithInterceptorFuncs(interceptor.Funcs{
			Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				if _, ok := obj.(*v1alpha1.ExporterPlugin); ok {
					return fmt.Errorf("injected Get error")
				}
				return c.Get(ctx, key, obj, opts...)
			},
		}).
		Build()

	r := &ExporterPluginReconciler{
		Client:        cl,
		Scheme:        scheme,
		HTTPClient:    &http.Client{Timeout: 2 * time.Second},
		ProbeInterval: 30 * time.Second,
		TokenGetter:   func() string { return "" },
	}
	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-plugin", Namespace: "assethub"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "injected Get error")
}

// Coverage: lines 46-49 — stale status update fails → logged, returns RequeueAfter
func TestExporterPluginReconcile_StaleStatusUpdateError(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	staleTime := time.Now().Add(-5 * time.Minute).UTC().Format(time.RFC3339)
	ep := &v1alpha1.ExporterPlugin{
		ObjectMeta: metav1.ObjectMeta{Name: "test-plugin", Namespace: "assethub"},
		Spec:       v1alpha1.ExporterPluginSpec{Endpoint: "http://localhost"},
		Status: v1alpha1.ExporterPluginStatus{
			Phase:           "Ready",
			LastHealthCheck: staleTime,
		},
	}

	cl := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(ep).
		WithStatusSubresource(ep).
		WithInterceptorFuncs(interceptor.Funcs{
			SubResourceUpdate: func(ctx context.Context, c client.Client, subResourceName string, obj client.Object, opts ...client.SubResourceUpdateOption) error {
				if _, ok := obj.(*v1alpha1.ExporterPlugin); ok {
					return fmt.Errorf("injected status update error")
				}
				return c.SubResource(subResourceName).Update(ctx, obj, opts...)
			},
		}).
		Build()

	r := &ExporterPluginReconciler{
		Client:        cl,
		Scheme:        scheme,
		HTTPClient:    &http.Client{Timeout: 2 * time.Second},
		ProbeInterval: 30 * time.Second,
		TokenGetter:   func() string { return "" },
	}

	result, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-plugin", Namespace: "assethub"},
	})
	require.NoError(t, err, "stale update error is logged, not returned")
	assert.Equal(t, 30*time.Second, result.RequeueAfter)
}

// Coverage: lines 57-59 — http.NewRequestWithContext error (bad endpoint URL)
func TestExporterPluginReconcile_BadEndpointURL(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	ep := &v1alpha1.ExporterPlugin{
		ObjectMeta: metav1.ObjectMeta{Name: "test-plugin", Namespace: "assethub"},
		Spec:       v1alpha1.ExporterPluginSpec{Endpoint: "http://host\x00invalid"},
	}
	r, _ := setupReconciler(t, ep)

	result, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-plugin", Namespace: "assethub"},
	})
	require.NoError(t, err, "bad URL returns RequeueAfter, not error")
	assert.Equal(t, 30*time.Second, result.RequeueAfter)
}

// Coverage: lines 81-84 — status update after health probe fails → logged, returns RequeueAfter
func TestExporterPluginReconcile_HealthStatusUpdateError(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ep := &v1alpha1.ExporterPlugin{
		ObjectMeta: metav1.ObjectMeta{Name: "test-plugin", Namespace: "assethub"},
		Spec:       v1alpha1.ExporterPluginSpec{Endpoint: server.URL},
	}

	cl := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(ep).
		WithStatusSubresource(ep).
		WithInterceptorFuncs(interceptor.Funcs{
			SubResourceUpdate: func(ctx context.Context, c client.Client, subResourceName string, obj client.Object, opts ...client.SubResourceUpdateOption) error {
				if _, ok := obj.(*v1alpha1.ExporterPlugin); ok {
					return fmt.Errorf("injected health status update error")
				}
				return c.SubResource(subResourceName).Update(ctx, obj, opts...)
			},
		}).
		Build()

	r := &ExporterPluginReconciler{
		Client:        cl,
		Scheme:        scheme,
		HTTPClient:    &http.Client{Timeout: 2 * time.Second},
		ProbeInterval: 30 * time.Second,
		TokenGetter:   func() string { return "" },
	}

	result, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-plugin", Namespace: "assethub"},
	})
	require.NoError(t, err, "health status update error is logged, not returned")
	assert.Equal(t, 30*time.Second, result.RequeueAfter)
}

// Coverage: lines 108-113 — SetupWithManager registers controller
func TestExporterPluginReconciler_SetupWithManager(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	// Create a manager with a fake API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"kind":"APIResourceList","apiVersion":"v1","resources":[]}`))
	}))
	defer server.Close()

	mgr, err := ctrl.NewManager(&rest.Config{Host: server.URL}, ctrl.Options{
		Scheme: scheme,
	})
	require.NoError(t, err)

	r := &ExporterPluginReconciler{
		Client:        mgr.GetClient(),
		Scheme:        scheme,
		HTTPClient:    &http.Client{Timeout: 2 * time.Second},
		ProbeInterval: 30 * time.Second,
		TokenGetter:   func() string { return "" },
	}

	err = r.SetupWithManager(mgr)
	assert.NoError(t, err)
}
