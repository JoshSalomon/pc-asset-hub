package controllers_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	v1alpha1 "github.com/project-catalyst/pc-asset-hub/internal/operator/api/v1alpha1"
	"github.com/project-catalyst/pc-asset-hub/internal/operator/controllers"
)

func testScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(s)
	_ = v1alpha1.AddToScheme(s)
	return s
}

func newTestCR(replicas int, env string) *v1alpha1.AssetHub {
	return &v1alpha1.AssetHub{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-hub",
			Namespace: "default",
		},
		Spec: v1alpha1.AssetHubSpec{
			Replicas:     replicas,
			DBConnection: "host=postgres user=assethub password=assethub dbname=assethub port=5432 sslmode=disable",
			Environment:  env,
			APINodePort:  30080,
			UINodePort:   30000,
		},
	}
}

func TestReconcile_CRExists_CreatesResources(t *testing.T) {
	s := testScheme()
	cr := newTestCR(2, "development")

	cl := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(cr).
		WithStatusSubresource(cr).
		Build()

	r := &controllers.AssetHubReconciler{
		Client: cl,
		Scheme: s,
	}

	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "test-hub", Namespace: "default"}}
	result, err := r.Reconcile(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)

	// Verify Deployments were created
	apiDep := &appsv1.Deployment{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "assethub-api", Namespace: "default"}, apiDep)
	require.NoError(t, err)
	assert.Equal(t, int32(2), *apiDep.Spec.Replicas)

	uiDep := &appsv1.Deployment{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "assethub-ui", Namespace: "default"}, uiDep)
	require.NoError(t, err)
	assert.Equal(t, int32(1), *uiDep.Spec.Replicas)

	// Verify Services were created
	apiSvc := &corev1.Service{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "assethub-api-svc", Namespace: "default"}, apiSvc)
	require.NoError(t, err)
	assert.Equal(t, int32(8080), apiSvc.Spec.Ports[0].Port)

	uiSvc := &corev1.Service{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "assethub-ui-svc", Namespace: "default"}, uiSvc)
	require.NoError(t, err)
	assert.Equal(t, int32(80), uiSvc.Spec.Ports[0].Port)

	// Verify status was updated
	updated := &v1alpha1.AssetHub{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "test-hub", Namespace: "default"}, updated)
	require.NoError(t, err)
	assert.True(t, updated.Status.Ready)
	assert.Equal(t, "all resources reconciled", updated.Status.Message)
}

func TestReconcile_CRDeleted_ReturnsNoError(t *testing.T) {
	s := testScheme()
	cl := fake.NewClientBuilder().WithScheme(s).Build()

	r := &controllers.AssetHubReconciler{
		Client: cl,
		Scheme: s,
	}

	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "missing-hub", Namespace: "default"}}
	result, err := r.Reconcile(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
}

func TestReconcile_UpdateReplicas(t *testing.T) {
	s := testScheme()
	cr := newTestCR(1, "development")

	cl := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(cr).
		WithStatusSubresource(cr).
		Build()

	r := &controllers.AssetHubReconciler{
		Client: cl,
		Scheme: s,
	}

	// First reconcile — creates resources with 1 replica
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "test-hub", Namespace: "default"}}
	_, err := r.Reconcile(context.Background(), req)
	require.NoError(t, err)

	apiDep := &appsv1.Deployment{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "assethub-api", Namespace: "default"}, apiDep)
	require.NoError(t, err)
	assert.Equal(t, int32(1), *apiDep.Spec.Replicas)

	// Update the CR replicas
	updated := &v1alpha1.AssetHub{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "test-hub", Namespace: "default"}, updated)
	require.NoError(t, err)
	updated.Spec.Replicas = 3
	err = cl.Update(context.Background(), updated)
	require.NoError(t, err)

	// Second reconcile — updates replicas to 3
	_, err = r.Reconcile(context.Background(), req)
	require.NoError(t, err)

	err = cl.Get(context.Background(), types.NamespacedName{Name: "assethub-api", Namespace: "default"}, apiDep)
	require.NoError(t, err)
	assert.Equal(t, int32(3), *apiDep.Spec.Replicas)
}

func TestReconcile_StatusUpdated(t *testing.T) {
	s := testScheme()
	cr := newTestCR(1, "development")

	cl := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(cr).
		WithStatusSubresource(cr).
		Build()

	r := &controllers.AssetHubReconciler{
		Client: cl,
		Scheme: s,
	}

	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "test-hub", Namespace: "default"}}
	_, err := r.Reconcile(context.Background(), req)
	require.NoError(t, err)

	updated := &v1alpha1.AssetHub{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "test-hub", Namespace: "default"}, updated)
	require.NoError(t, err)
	assert.True(t, updated.Status.Ready)
	assert.Equal(t, "all resources reconciled", updated.Status.Message)
}

// T-D.16: Reconcile creates ConfigMap in cluster
func TestTD_16_ReconcileCreatesConfigMap(t *testing.T) {
	s := testScheme()
	cr := newTestCR(1, "development")

	cl := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(cr).
		WithStatusSubresource(cr).
		Build()

	r := &controllers.AssetHubReconciler{Client: cl, Scheme: s}

	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "test-hub", Namespace: "default"}}
	_, err := r.Reconcile(context.Background(), req)
	require.NoError(t, err)

	cm := &corev1.ConfigMap{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "api-server-config", Namespace: "default"}, cm)
	require.NoError(t, err)

	assert.Equal(t, "postgres", cm.Data["DB_DRIVER"])
	assert.Equal(t, "host=postgres user=assethub password=assethub dbname=assethub port=5432 sslmode=disable", cm.Data["DB_CONNECTION_STRING"])
	assert.Equal(t, "8080", cm.Data["API_PORT"])
	assert.Equal(t, "http://localhost:30000", cm.Data["CORS_ALLOWED_ORIGINS"])
	assert.Equal(t, "info", cm.Data["LOG_LEVEL"])
	assert.Equal(t, "header", cm.Data["RBAC_MODE"])
}

// T-D.17: Reconcile creates Deployments with probes and envFrom
func TestTD_17_ReconcileCreatesDeploymentsWithProbesAndEnvFrom(t *testing.T) {
	s := testScheme()
	cr := newTestCR(1, "development")

	cl := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(cr).
		WithStatusSubresource(cr).
		Build()

	r := &controllers.AssetHubReconciler{Client: cl, Scheme: s}

	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "test-hub", Namespace: "default"}}
	_, err := r.Reconcile(context.Background(), req)
	require.NoError(t, err)

	apiDep := &appsv1.Deployment{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "assethub-api", Namespace: "default"}, apiDep)
	require.NoError(t, err)

	container := apiDep.Spec.Template.Spec.Containers[0]
	require.NotNil(t, container.ReadinessProbe, "API should have readiness probe")
	assert.Equal(t, "/readyz", container.ReadinessProbe.HTTPGet.Path)
	require.NotNil(t, container.LivenessProbe, "API should have liveness probe")
	assert.Equal(t, "/healthz", container.LivenessProbe.HTTPGet.Path)
	require.Len(t, container.EnvFrom, 1, "API should have envFrom")
	assert.Equal(t, "api-server-config", container.EnvFrom[0].ConfigMapRef.Name)
}

// T-D.18: Reconcile creates Services with NodePort (dev mode)
func TestTD_18_ReconcileCreatesServicesWithNodePort(t *testing.T) {
	s := testScheme()
	cr := newTestCR(1, "development")

	cl := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(cr).
		WithStatusSubresource(cr).
		Build()

	r := &controllers.AssetHubReconciler{Client: cl, Scheme: s}

	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "test-hub", Namespace: "default"}}
	_, err := r.Reconcile(context.Background(), req)
	require.NoError(t, err)

	apiSvc := &corev1.Service{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "assethub-api-svc", Namespace: "default"}, apiSvc)
	require.NoError(t, err)
	assert.Equal(t, corev1.ServiceTypeNodePort, apiSvc.Spec.Type)
	assert.Equal(t, int32(30080), apiSvc.Spec.Ports[0].NodePort)
}

// T-D.19: Reconcile sets correct container ports (API=8080, UI=80)
func TestTD_19_ReconcileSetsCorrectContainerPorts(t *testing.T) {
	s := testScheme()
	cr := newTestCR(1, "development")

	cl := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(cr).
		WithStatusSubresource(cr).
		Build()

	r := &controllers.AssetHubReconciler{Client: cl, Scheme: s}

	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "test-hub", Namespace: "default"}}
	_, err := r.Reconcile(context.Background(), req)
	require.NoError(t, err)

	apiDep := &appsv1.Deployment{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "assethub-api", Namespace: "default"}, apiDep)
	require.NoError(t, err)
	assert.Equal(t, int32(8080), apiDep.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort)

	uiDep := &appsv1.Deployment{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "assethub-ui", Namespace: "default"}, uiDep)
	require.NoError(t, err)
	assert.Equal(t, int32(80), uiDep.Spec.Template.Spec.Containers[0].Ports[0].ContainerPort)
}

// T-D.20: Reconcile sets imagePullPolicy=Never (dev mode)
func TestTD_20_ReconcileSetsImagePullPolicyNever(t *testing.T) {
	s := testScheme()
	cr := newTestCR(1, "development")

	cl := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(cr).
		WithStatusSubresource(cr).
		Build()

	r := &controllers.AssetHubReconciler{Client: cl, Scheme: s}

	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "test-hub", Namespace: "default"}}
	_, err := r.Reconcile(context.Background(), req)
	require.NoError(t, err)

	apiDep := &appsv1.Deployment{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "assethub-api", Namespace: "default"}, apiDep)
	require.NoError(t, err)
	assert.Equal(t, corev1.PullNever, apiDep.Spec.Template.Spec.Containers[0].ImagePullPolicy)

	uiDep := &appsv1.Deployment{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "assethub-ui", Namespace: "default"}, uiDep)
	require.NoError(t, err)
	assert.Equal(t, corev1.PullNever, uiDep.Spec.Template.Spec.Containers[0].ImagePullPolicy)
}

// T-D.21: Update CR replicas → Deployment updated
func TestTD_21_UpdateCRReplicasUpdatesDeployment(t *testing.T) {
	s := testScheme()
	cr := newTestCR(1, "development")

	cl := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(cr).
		WithStatusSubresource(cr).
		Build()

	r := &controllers.AssetHubReconciler{Client: cl, Scheme: s}

	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "test-hub", Namespace: "default"}}
	_, err := r.Reconcile(context.Background(), req)
	require.NoError(t, err)

	// Update replicas to 5
	updated := &v1alpha1.AssetHub{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "test-hub", Namespace: "default"}, updated)
	require.NoError(t, err)
	updated.Spec.Replicas = 5
	err = cl.Update(context.Background(), updated)
	require.NoError(t, err)

	_, err = r.Reconcile(context.Background(), req)
	require.NoError(t, err)

	apiDep := &appsv1.Deployment{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "assethub-api", Namespace: "default"}, apiDep)
	require.NoError(t, err)
	assert.Equal(t, int32(5), *apiDep.Spec.Replicas)
}

// T-D.22: CR deleted → no error
func TestTD_22_CRDeletedReturnsNoError(t *testing.T) {
	s := testScheme()
	cl := fake.NewClientBuilder().WithScheme(s).Build()

	r := &controllers.AssetHubReconciler{Client: cl, Scheme: s}

	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "nonexistent", Namespace: "default"}}
	result, err := r.Reconcile(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
}

// T-D.23: Status updated to ready after reconcile
func TestTD_23_StatusUpdatedToReady(t *testing.T) {
	s := testScheme()
	cr := newTestCR(1, "development")

	cl := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(cr).
		WithStatusSubresource(cr).
		Build()

	r := &controllers.AssetHubReconciler{Client: cl, Scheme: s}

	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "test-hub", Namespace: "default"}}
	_, err := r.Reconcile(context.Background(), req)
	require.NoError(t, err)

	updated := &v1alpha1.AssetHub{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "test-hub", Namespace: "default"}, updated)
	require.NoError(t, err)
	assert.True(t, updated.Status.Ready)
	assert.Equal(t, "all resources reconciled", updated.Status.Message)
}

// T-CV.12: Reconcile with clusterRole="production" creates ConfigMap with CLUSTER_ROLE=production
func TestTCV12_ReconcileClusterRoleInConfigMap(t *testing.T) {
	s := testScheme()
	cr := newTestCR(1, "development")
	cr.Spec.ClusterRole = "production"

	cl := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(cr).
		WithStatusSubresource(cr).
		Build()

	r := &controllers.AssetHubReconciler{Client: cl, Scheme: s}

	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "test-hub", Namespace: "default"}}
	_, err := r.Reconcile(context.Background(), req)
	require.NoError(t, err)

	cm := &corev1.ConfigMap{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "api-server-config", Namespace: "default"}, cm)
	require.NoError(t, err)
	assert.Equal(t, "production", cm.Data["CLUSTER_ROLE"])
}

// T-CV.13: Reconcile sets owner reference on existing CatalogVersion CR
func TestTCV13_ReconcileSetsOwnerRefOnCatalogVersion(t *testing.T) {
	s := testScheme()
	cr := newTestCR(1, "development")
	cr.UID = "test-uid-123"

	cv := &v1alpha1.CatalogVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "release-1",
			Namespace: "default",
		},
		Spec: v1alpha1.CatalogVersionSpec{
			VersionLabel:   "Release 1",
			LifecycleStage: "testing",
			EntityTypes:    []string{"Device"},
		},
	}

	cl := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(cr, cv).
		WithStatusSubresource(cr, cv).
		Build()

	r := &controllers.AssetHubReconciler{Client: cl, Scheme: s}

	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "test-hub", Namespace: "default"}}
	_, err := r.Reconcile(context.Background(), req)
	require.NoError(t, err)

	updated := &v1alpha1.CatalogVersion{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "release-1", Namespace: "default"}, updated)
	require.NoError(t, err)
	require.NotEmpty(t, updated.OwnerReferences)
	assert.Equal(t, "test-hub", updated.OwnerReferences[0].Name)
}

// T-CV.14: Reconcile updates CatalogVersion status to ready=true
func TestTCV14_ReconcileUpdatesCatalogVersionStatus(t *testing.T) {
	s := testScheme()
	cr := newTestCR(1, "development")

	cv := &v1alpha1.CatalogVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "release-1",
			Namespace: "default",
		},
		Spec: v1alpha1.CatalogVersionSpec{
			VersionLabel:   "Release 1",
			LifecycleStage: "production",
			EntityTypes:    []string{"Device"},
		},
	}

	cl := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(cr, cv).
		WithStatusSubresource(cr, cv).
		Build()

	r := &controllers.AssetHubReconciler{Client: cl, Scheme: s}

	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "test-hub", Namespace: "default"}}
	_, err := r.Reconcile(context.Background(), req)
	require.NoError(t, err)

	updated := &v1alpha1.CatalogVersion{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "release-1", Namespace: "default"}, updated)
	require.NoError(t, err)
	assert.True(t, updated.Status.Ready)
	assert.Contains(t, updated.Status.Message, "production")
}

// T-CV.15: Reconcile with no CatalogVersion CRs in namespace succeeds without error
func TestTCV15_ReconcileNoCatalogVersions(t *testing.T) {
	s := testScheme()
	cr := newTestCR(1, "development")

	cl := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(cr).
		WithStatusSubresource(cr).
		Build()

	r := &controllers.AssetHubReconciler{Client: cl, Scheme: s}

	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "test-hub", Namespace: "default"}}
	result, err := r.Reconcile(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)

	updated := &v1alpha1.AssetHub{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "test-hub", Namespace: "default"}, updated)
	require.NoError(t, err)
	assert.True(t, updated.Status.Ready)
}

// Test OpenShift mode creates ClusterIP services with IfNotPresent pull policy
func TestReconcile_OpenShiftMode_CreatesClusterIPServices(t *testing.T) {
	s := testScheme()
	cr := newTestCR(1, "openshift")

	cl := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(cr).
		WithStatusSubresource(cr).
		Build()

	r := &controllers.AssetHubReconciler{Client: cl, Scheme: s}

	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "test-hub", Namespace: "default"}}
	_, err := r.Reconcile(context.Background(), req)
	require.NoError(t, err)

	apiSvc := &corev1.Service{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "assethub-api-svc", Namespace: "default"}, apiSvc)
	require.NoError(t, err)
	assert.Equal(t, corev1.ServiceTypeClusterIP, apiSvc.Spec.Type)
	assert.Equal(t, int32(0), apiSvc.Spec.Ports[0].NodePort)

	apiDep := &appsv1.Deployment{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "assethub-api", Namespace: "default"}, apiDep)
	require.NoError(t, err)
	assert.Equal(t, corev1.PullIfNotPresent, apiDep.Spec.Template.Spec.Containers[0].ImagePullPolicy)
}

// === Catalog CR Reconciliation Tests ===

// T-16.52: Reconciler sets owner reference on Catalog CR
func TestT16_52_CatalogCR_OwnerRefSet(t *testing.T) {
	scheme := testScheme()
	cr := newTestCR(1, "development")
	cr.UID = "test-uid-123"

	cat := &v1alpha1.Catalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prod-catalog",
			Namespace: "default",
		},
		Spec: v1alpha1.CatalogSpec{
			CatalogName:      "prod-catalog",
			ValidationStatus: "valid",
		},
	}

	cl := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(cr, cat).
		WithStatusSubresource(cr, cat).
		Build()
	r := &controllers.AssetHubReconciler{Client: cl, Scheme: scheme}

	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-hub", Namespace: "default"},
	})
	require.NoError(t, err)

	// Verify owner reference set
	updatedCat := &v1alpha1.Catalog{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "prod-catalog", Namespace: "default"}, updatedCat)
	require.NoError(t, err)
	require.Len(t, updatedCat.OwnerReferences, 1)
	assert.Equal(t, "test-hub", updatedCat.OwnerReferences[0].Name)
}

// T-16.53/55: Reconciler sets status.Ready and increments DataVersion
func TestT16_53_CatalogCR_StatusUpdated(t *testing.T) {
	scheme := testScheme()
	cr := newTestCR(1, "development")
	cr.UID = "test-uid-123"

	cat := &v1alpha1.Catalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "prod-catalog",
			Namespace: "default",
		},
		Spec: v1alpha1.CatalogSpec{
			CatalogName:      "prod-catalog",
			ValidationStatus: "valid",
		},
		Status: v1alpha1.CatalogStatus{
			Ready:       false,
			DataVersion: 0,
		},
	}

	cl := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(cr, cat).
		WithStatusSubresource(cr, cat).
		Build()
	r := &controllers.AssetHubReconciler{Client: cl, Scheme: scheme}

	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-hub", Namespace: "default"},
	})
	require.NoError(t, err)

	updatedCat := &v1alpha1.Catalog{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "prod-catalog", Namespace: "default"}, updatedCat)
	require.NoError(t, err)
	assert.True(t, updatedCat.Status.Ready)
	assert.Equal(t, "catalog published", updatedCat.Status.Message)
	assert.Equal(t, 1, updatedCat.Status.DataVersion)
}

// No Catalog CRs — reconciliation succeeds without error
func TestCatalogCR_NoCatalogs(t *testing.T) {
	scheme := testScheme()
	cr := newTestCR(1, "development")
	cr.UID = "test-uid-123"

	cl := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(cr).
		WithStatusSubresource(cr).
		Build()
	r := &controllers.AssetHubReconciler{Client: cl, Scheme: scheme}

	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-hub", Namespace: "default"},
	})
	require.NoError(t, err)
}

// CR being deleted — DeletionTimestamp set (line 46-48)
func TestReconcile_CRBeingDeleted(t *testing.T) {
	s := testScheme()
	cr := newTestCR(1, "development")
	now := metav1.Now()
	cr.DeletionTimestamp = &now
	cr.Finalizers = []string{"test-finalizer"} // Required for fake client to accept DeletionTimestamp

	cl := fake.NewClientBuilder().WithScheme(s).
		WithObjects(cr).
		Build()
	r := &controllers.AssetHubReconciler{Client: cl, Scheme: s}

	result, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-hub", Namespace: "default"},
	})
	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
}

// CR not found — returns no error (line 38-40)
func TestReconcile_CRNotFound(t *testing.T) {
	s := testScheme()
	cl := fake.NewClientBuilder().WithScheme(s).Build()
	r := &controllers.AssetHubReconciler{Client: cl, Scheme: s}

	result, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "nonexistent", Namespace: "default"},
	})
	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{}, result)
}

// === Error Path Tests (interceptor-based) ===

// Line 42: r.Get returns non-NotFound error
func TestReconcile_GetCRError(t *testing.T) {
	s := testScheme()
	cr := newTestCR(1, "development")

	cl := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(cr).
		WithInterceptorFuncs(interceptor.Funcs{
			Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				if _, ok := obj.(*v1alpha1.AssetHub); ok && key.Name == "test-hub" {
					return fmt.Errorf("injected Get error")
				}
				return c.Get(ctx, key, obj, opts...)
			},
		}).
		Build()

	r := &controllers.AssetHubReconciler{Client: cl, Scheme: s}
	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-hub", Namespace: "default"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "injected Get error")
}

// Lines 69-70: reconcileConfigMap fails — error recorded in status
func TestReconcile_ConfigMapError(t *testing.T) {
	s := testScheme()
	cr := newTestCR(1, "development")

	cl := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(cr).
		WithStatusSubresource(cr).
		WithInterceptorFuncs(interceptor.Funcs{
			Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
				if _, ok := obj.(*corev1.ConfigMap); ok {
					return fmt.Errorf("injected ConfigMap create error")
				}
				return c.Create(ctx, obj, opts...)
			},
		}).
		Build()

	r := &controllers.AssetHubReconciler{Client: cl, Scheme: s}
	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-hub", Namespace: "default"},
	})
	// updateStatus records the error in status and returns nil
	require.NoError(t, err)
	updated := &v1alpha1.AssetHub{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "test-hub", Namespace: "default"}, updated)
	require.NoError(t, err)
	assert.False(t, updated.Status.Ready)
	assert.Contains(t, updated.Status.Message, "failed to reconcile configmap")
}

// Lines 76-77: reconcileDeployment fails — error recorded in status
func TestReconcile_DeploymentError(t *testing.T) {
	s := testScheme()
	cr := newTestCR(1, "development")

	cl := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(cr).
		WithStatusSubresource(cr).
		WithInterceptorFuncs(interceptor.Funcs{
			Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
				if _, ok := obj.(*appsv1.Deployment); ok {
					return fmt.Errorf("injected Deployment create error")
				}
				return c.Create(ctx, obj, opts...)
			},
		}).
		Build()

	r := &controllers.AssetHubReconciler{Client: cl, Scheme: s}
	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-hub", Namespace: "default"},
	})
	require.NoError(t, err)
	updated := &v1alpha1.AssetHub{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "test-hub", Namespace: "default"}, updated)
	require.NoError(t, err)
	assert.False(t, updated.Status.Ready)
	assert.Contains(t, updated.Status.Message, "failed to reconcile deployment")
}

// Lines 83-84: reconcileService fails — error recorded in status
func TestReconcile_ServiceError(t *testing.T) {
	s := testScheme()
	cr := newTestCR(1, "development")

	cl := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(cr).
		WithStatusSubresource(cr).
		WithInterceptorFuncs(interceptor.Funcs{
			Create: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
				if _, ok := obj.(*corev1.Service); ok {
					return fmt.Errorf("injected Service create error")
				}
				return c.Create(ctx, obj, opts...)
			},
		}).
		Build()

	r := &controllers.AssetHubReconciler{Client: cl, Scheme: s}
	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-hub", Namespace: "default"},
	})
	require.NoError(t, err)
	updated := &v1alpha1.AssetHub{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "test-hub", Namespace: "default"}, updated)
	require.NoError(t, err)
	assert.False(t, updated.Status.Ready)
	assert.Contains(t, updated.Status.Message, "failed to reconcile service")
}

// Lines 90-91 + 285-286: reconcileRoute fails (route Get returns non-NotFound error)
func TestReconcile_RouteGetError(t *testing.T) {
	s := testScheme()
	cr := newTestCR(1, "openshift")
	cr.Spec.APIHostname = "api.example.com"

	cl := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(cr).
		WithStatusSubresource(cr).
		WithInterceptorFuncs(interceptor.Funcs{
			Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				if u, ok := obj.(*unstructured.Unstructured); ok {
					gvk := u.GroupVersionKind()
					if gvk.Kind == "Route" && gvk.Group == "route.openshift.io" {
						return fmt.Errorf("injected Route Get error")
					}
				}
				return c.Get(ctx, key, obj, opts...)
			},
		}).
		Build()

	r := &controllers.AssetHubReconciler{Client: cl, Scheme: s}
	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-hub", Namespace: "default"},
	})
	// updateStatus records route error in status
	require.NoError(t, err)
	updated := &v1alpha1.AssetHub{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "test-hub", Namespace: "default"}, updated)
	require.NoError(t, err)
	assert.False(t, updated.Status.Ready)
	assert.Contains(t, updated.Status.Message, "failed to reconcile route")
}

// Lines 290-293: reconcileRoute — route already exists, update path
func TestReconcile_RouteAlreadyExists_Updates(t *testing.T) {
	s := testScheme()
	cr := newTestCR(1, "openshift")
	cr.Spec.APIHostname = "api.example.com"
	cr.Spec.UIHostname = "ui.example.com"

	// Pre-create route as unstructured object
	existingRoute := &unstructured.Unstructured{}
	existingRoute.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "route.openshift.io",
		Version: "v1",
		Kind:    "Route",
	})
	existingRoute.SetName("assethub-api-route")
	existingRoute.SetNamespace("default")
	existingRoute.Object["spec"] = map[string]any{
		"host": "old-host.example.com",
		"to":   map[string]any{"kind": "Service", "name": "assethub-api-svc"},
		"port": map[string]any{"targetPort": int64(8080)},
	}

	existingRoute2 := &unstructured.Unstructured{}
	existingRoute2.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "route.openshift.io",
		Version: "v1",
		Kind:    "Route",
	})
	existingRoute2.SetName("assethub-ui-route")
	existingRoute2.SetNamespace("default")
	existingRoute2.Object["spec"] = map[string]any{
		"host": "old-ui.example.com",
		"to":   map[string]any{"kind": "Service", "name": "assethub-ui-svc"},
		"port": map[string]any{"targetPort": int64(80)},
	}

	cl := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(cr, existingRoute, existingRoute2).
		WithStatusSubresource(cr).
		Build()

	r := &controllers.AssetHubReconciler{Client: cl, Scheme: s}
	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-hub", Namespace: "default"},
	})
	require.NoError(t, err)

	// Verify route was updated with new hostname
	updated := &unstructured.Unstructured{}
	updated.SetGroupVersionKind(schema.GroupVersionKind{
		Group: "route.openshift.io", Version: "v1", Kind: "Route",
	})
	err = cl.Get(context.Background(), types.NamespacedName{Name: "assethub-api-route", Namespace: "default"}, updated)
	require.NoError(t, err)
	spec := updated.Object["spec"].(map[string]any)
	assert.Equal(t, "api.example.com", spec["host"])
}

// Lines 96-97: reconcileCatalogVersions fails (List error)
func TestReconcile_CatalogVersionsListError(t *testing.T) {
	s := testScheme()
	cr := newTestCR(1, "development")

	cl := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(cr).
		WithStatusSubresource(cr).
		WithInterceptorFuncs(interceptor.Funcs{
			List: func(ctx context.Context, c client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
				if _, ok := list.(*v1alpha1.CatalogVersionList); ok {
					return fmt.Errorf("injected CatalogVersion list error")
				}
				return c.List(ctx, list, opts...)
			},
		}).
		Build()

	r := &controllers.AssetHubReconciler{Client: cl, Scheme: s}
	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-hub", Namespace: "default"},
	})
	require.NoError(t, err)
	updated := &v1alpha1.AssetHub{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "test-hub", Namespace: "default"}, updated)
	require.NoError(t, err)
	assert.False(t, updated.Status.Ready)
	assert.Contains(t, updated.Status.Message, "failed to reconcile catalog versions")
}

// Lines 101-102: reconcileCatalogs fails (List error)
func TestReconcile_CatalogsListError(t *testing.T) {
	s := testScheme()
	cr := newTestCR(1, "development")

	cl := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(cr).
		WithStatusSubresource(cr).
		WithInterceptorFuncs(interceptor.Funcs{
			List: func(ctx context.Context, c client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
				if _, ok := list.(*v1alpha1.CatalogList); ok {
					return fmt.Errorf("injected Catalog list error")
				}
				return c.List(ctx, list, opts...)
			},
		}).
		Build()

	r := &controllers.AssetHubReconciler{Client: cl, Scheme: s}
	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-hub", Namespace: "default"},
	})
	require.NoError(t, err)
	updated := &v1alpha1.AssetHub{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "test-hub", Namespace: "default"}, updated)
	require.NoError(t, err)
	assert.False(t, updated.Status.Ready)
	assert.Contains(t, updated.Status.Message, "failed to reconcile catalogs")
}

// Line 105: updateStatus final call fails
func TestReconcile_FinalUpdateStatusError(t *testing.T) {
	s := testScheme()
	cr := newTestCR(1, "development")

	statusUpdateCount := 0
	cl := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(cr).
		WithStatusSubresource(cr).
		WithInterceptorFuncs(interceptor.Funcs{
			SubResourceUpdate: func(ctx context.Context, c client.Client, subResourceName string, obj client.Object, opts ...client.SubResourceUpdateOption) error {
				if _, ok := obj.(*v1alpha1.AssetHub); ok {
					statusUpdateCount++
					// Fail the first (and only, in success path) status update
					return fmt.Errorf("injected status update error")
				}
				return c.SubResource(subResourceName).Update(ctx, obj, opts...)
			},
		}).
		Build()

	r := &controllers.AssetHubReconciler{Client: cl, Scheme: s}
	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-hub", Namespace: "default"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "injected status update error")
}

// Line 298: updateStatus — Get fails when fetching latest CR
func TestReconcile_UpdateStatusGetError(t *testing.T) {
	s := testScheme()
	cr := newTestCR(1, "development")

	getCount := 0
	cl := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(cr).
		WithStatusSubresource(cr).
		WithInterceptorFuncs(interceptor.Funcs{
			Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				if _, ok := obj.(*v1alpha1.AssetHub); ok {
					getCount++
					// First Get succeeds (line 37), second Get in updateStatus (line 298) fails
					if getCount >= 2 {
						return fmt.Errorf("injected updateStatus Get error")
					}
				}
				return c.Get(ctx, key, obj, opts...)
			},
		}).
		Build()

	r := &controllers.AssetHubReconciler{Client: cl, Scheme: s}
	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-hub", Namespace: "default"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "injected updateStatus Get error")
}

// Lines 317-320: reconcileCatalogVersions — Update fails after setting owner ref
func TestReconcile_CVUpdateOwnerRefError(t *testing.T) {
	s := testScheme()
	cr := newTestCR(1, "development")
	cr.UID = "test-uid-456"

	cv := &v1alpha1.CatalogVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cv",
			Namespace: "default",
		},
		Spec: v1alpha1.CatalogVersionSpec{
			LifecycleStage: "testing",
		},
	}

	cl := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(cr, cv).
		WithStatusSubresource(cr, cv).
		WithInterceptorFuncs(interceptor.Funcs{
			Update: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
				if _, ok := obj.(*v1alpha1.CatalogVersion); ok {
					return fmt.Errorf("injected CV update error")
				}
				return c.Update(ctx, obj, opts...)
			},
		}).
		Build()

	r := &controllers.AssetHubReconciler{Client: cl, Scheme: s}
	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-hub", Namespace: "default"},
	})
	// reconcileCatalogVersions returns error, updateStatus wraps it in status message
	require.NoError(t, err)
	updated := &v1alpha1.AssetHub{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "test-hub", Namespace: "default"}, updated)
	require.NoError(t, err)
	assert.False(t, updated.Status.Ready)
	assert.Contains(t, updated.Status.Message, "failed to reconcile catalog versions")
}

// Line 330: reconcileCatalogVersions — Status().Update fails
func TestReconcile_CVStatusUpdateError(t *testing.T) {
	s := testScheme()
	cr := newTestCR(1, "development")
	cr.UID = "test-uid-789"

	cv := &v1alpha1.CatalogVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cv",
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{
				{UID: "test-uid-789", Name: "test-hub", APIVersion: "v1alpha1", Kind: "AssetHub"},
			},
		},
		Spec: v1alpha1.CatalogVersionSpec{
			LifecycleStage: "production",
		},
		// Status is empty so it will need an update
	}

	cl := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(cr, cv).
		WithStatusSubresource(cr, cv).
		WithInterceptorFuncs(interceptor.Funcs{
			SubResourceUpdate: func(ctx context.Context, c client.Client, subResourceName string, obj client.Object, opts ...client.SubResourceUpdateOption) error {
				if _, ok := obj.(*v1alpha1.CatalogVersion); ok {
					return fmt.Errorf("injected CV status update error")
				}
				return c.SubResource(subResourceName).Update(ctx, obj, opts...)
			},
		}).
		Build()

	r := &controllers.AssetHubReconciler{Client: cl, Scheme: s}
	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-hub", Namespace: "default"},
	})
	// reconcileCatalogVersions error is wrapped by updateStatus; if the CV status update fails,
	// the error propagates up to reconcileCatalogVersions, then updateStatus is called.
	// But updateStatus also calls SubResourceUpdate on AssetHub, which should succeed.
	require.NoError(t, err)
	updated := &v1alpha1.AssetHub{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "test-hub", Namespace: "default"}, updated)
	require.NoError(t, err)
	assert.False(t, updated.Status.Ready)
	assert.Contains(t, updated.Status.Message, "failed to reconcile catalog versions")
}

// Lines 350-353: reconcileCatalogs — Update fails after setting owner ref
func TestReconcile_CatalogUpdateOwnerRefError(t *testing.T) {
	s := testScheme()
	cr := newTestCR(1, "development")
	cr.UID = "test-uid-cat-1"

	cat := &v1alpha1.Catalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-catalog",
			Namespace: "default",
		},
		Spec: v1alpha1.CatalogSpec{
			CatalogName:      "test-catalog",
			ValidationStatus: "valid",
		},
	}

	cl := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(cr, cat).
		WithStatusSubresource(cr, cat).
		WithInterceptorFuncs(interceptor.Funcs{
			Update: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
				if _, ok := obj.(*v1alpha1.Catalog); ok {
					return fmt.Errorf("injected Catalog update error")
				}
				return c.Update(ctx, obj, opts...)
			},
		}).
		Build()

	r := &controllers.AssetHubReconciler{Client: cl, Scheme: s}
	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-hub", Namespace: "default"},
	})
	require.NoError(t, err)
	updated := &v1alpha1.AssetHub{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "test-hub", Namespace: "default"}, updated)
	require.NoError(t, err)
	assert.False(t, updated.Status.Ready)
	assert.Contains(t, updated.Status.Message, "failed to reconcile catalogs")
}

// Line 366: reconcileCatalogs — Status().Update fails
func TestReconcile_CatalogStatusUpdateError(t *testing.T) {
	s := testScheme()
	cr := newTestCR(1, "development")
	cr.UID = "test-uid-cat-2"

	cat := &v1alpha1.Catalog{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-catalog",
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{
				{UID: "test-uid-cat-2", Name: "test-hub", APIVersion: "v1alpha1", Kind: "AssetHub"},
			},
		},
		Spec: v1alpha1.CatalogSpec{
			CatalogName:      "test-catalog",
			ValidationStatus: "valid",
		},
		// Status is empty so NeedsUpdate will be true
	}

	cl := fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(cr, cat).
		WithStatusSubresource(cr, cat).
		WithInterceptorFuncs(interceptor.Funcs{
			SubResourceUpdate: func(ctx context.Context, c client.Client, subResourceName string, obj client.Object, opts ...client.SubResourceUpdateOption) error {
				if _, ok := obj.(*v1alpha1.Catalog); ok {
					return fmt.Errorf("injected Catalog status update error")
				}
				return c.SubResource(subResourceName).Update(ctx, obj, opts...)
			},
		}).
		Build()

	r := &controllers.AssetHubReconciler{Client: cl, Scheme: s}
	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-hub", Namespace: "default"},
	})
	require.NoError(t, err)
	updated := &v1alpha1.AssetHub{}
	err = cl.Get(context.Background(), types.NamespacedName{Name: "test-hub", Namespace: "default"}, updated)
	require.NoError(t, err)
	assert.False(t, updated.Status.Ready)
	assert.Contains(t, updated.Status.Message, "failed to reconcile catalogs")
}

// CatalogVersion with existing owner ref — hasOwnerRef returns true, skips set (lines 376-377)
func TestReconcile_CVWithExistingOwnerRef(t *testing.T) {
	s := testScheme()
	cr := newTestCR(1, "development")
	cr.UID = "test-uid-123"

	cv := &v1alpha1.CatalogVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cv",
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "assethub.project-catalyst.io/v1alpha1",
					Kind:       "AssetHub",
					Name:       "test-hub",
					UID:        "test-uid-123",
				},
			},
		},
		Spec: v1alpha1.CatalogVersionSpec{
			LifecycleStage: "testing",
		},
	}

	cl := fake.NewClientBuilder().WithScheme(s).
		WithObjects(cr, cv).
		WithStatusSubresource(cr, cv).
		Build()
	r := &controllers.AssetHubReconciler{Client: cl, Scheme: s}

	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "test-hub", Namespace: "default"},
	})
	require.NoError(t, err)
}
