package controllers_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

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
