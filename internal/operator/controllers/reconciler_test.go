package controllers_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	v1alpha1 "github.com/project-catalyst/pc-asset-hub/internal/operator/api/v1alpha1"
	"github.com/project-catalyst/pc-asset-hub/internal/operator/controllers"
)

// T-9.01: Create AssetHub CR → Deployment, Service, UI Deployment created
func TestT9_01_ReconcileAssetHub(t *testing.T) {
	spec := controllers.AssetHubSpec{Replicas: 2, DBConnection: "sqlite://test.db"}
	result := controllers.ReconcileAssetHub(spec)

	require.Len(t, result.Deployments, 2)
	assert.Equal(t, "assethub-api", result.Deployments[0].Name)
	assert.Equal(t, 2, result.Deployments[0].Replicas)
	assert.Equal(t, "assethub-ui", result.Deployments[1].Name)

	require.Len(t, result.Services, 2)
	assert.Equal(t, "assethub-api-svc", result.Services[0].Name)
	assert.Equal(t, 8080, result.Services[0].Port)

	// New: verify ConfigMap created
	require.Len(t, result.ConfigMaps, 1)
	assert.Equal(t, "api-server-config", result.ConfigMaps[0].Name)

	// New: verify environment-driven defaults
	assert.Equal(t, "Never", result.Deployments[0].ImagePullPolicy)
	assert.Equal(t, int32(8080), result.Deployments[0].ContainerPort)
	assert.Equal(t, "NodePort", result.Services[0].Type)
}

// T-9.02: Delete AssetHub CR → all managed resources cleaned up
func TestT9_02_CleanupAssetHub(t *testing.T) {
	resources := controllers.CleanupAssetHub()
	assert.Len(t, resources, 7)
	assert.Contains(t, resources, "deployment/assethub-api")
	assert.Contains(t, resources, "deployment/assethub-ui")
	assert.Contains(t, resources, "service/assethub-api-svc")
	assert.Contains(t, resources, "service/assethub-ui-svc")
	assert.Contains(t, resources, "configmap/api-server-config")
	assert.Contains(t, resources, "route/assethub-api-route")
	assert.Contains(t, resources, "route/assethub-ui-route")
}

// T-9.03: AssetHub CR update (change replicas) → Deployment updated
func TestT9_03_UpdateReplicas(t *testing.T) {
	spec1 := controllers.AssetHubSpec{Replicas: 1}
	result1 := controllers.ReconcileAssetHub(spec1)
	assert.Equal(t, 1, result1.Deployments[0].Replicas)

	spec2 := controllers.AssetHubSpec{Replicas: 3}
	result2 := controllers.ReconcileAssetHub(spec2)
	assert.Equal(t, 3, result2.Deployments[0].Replicas)
}

// T-D.01: Development mode produces 2 Deployments, 2 Services, 1 ConfigMap, 0 Routes
func TestTD_01_DevelopmentModeResourceCounts(t *testing.T) {
	spec := controllers.AssetHubSpec{
		Replicas:     1,
		DBConnection: "host=postgres user=assethub password=assethub dbname=assethub port=5432 sslmode=disable",
		Environment:  "development",
	}
	result := controllers.ReconcileAssetHub(spec)

	assert.Len(t, result.Deployments, 2)
	assert.Len(t, result.Services, 2)
	assert.Len(t, result.ConfigMaps, 1)
	assert.Len(t, result.Routes, 0)
}

// T-D.02: Development mode API deployment has port 8080, probes, envFrom, imagePullPolicy=Never
func TestTD_02_DevelopmentAPIDeployment(t *testing.T) {
	spec := controllers.AssetHubSpec{
		Replicas:     1,
		DBConnection: "host=postgres dbname=assethub",
		Environment:  "development",
	}
	result := controllers.ReconcileAssetHub(spec)
	api := result.Deployments[0]

	assert.Equal(t, "assethub-api", api.Name)
	assert.Equal(t, int32(8080), api.ContainerPort)
	assert.Equal(t, "/readyz", api.ReadinessPath)
	assert.Equal(t, "/healthz", api.LivenessPath)
	assert.Equal(t, "api-server-config", api.EnvFrom)
	assert.Equal(t, "Never", api.ImagePullPolicy)
}

// T-D.03: Development mode UI deployment has port 80, readiness probe, imagePullPolicy=Never
func TestTD_03_DevelopmentUIDeployment(t *testing.T) {
	spec := controllers.AssetHubSpec{
		Replicas:    1,
		Environment: "development",
	}
	result := controllers.ReconcileAssetHub(spec)
	ui := result.Deployments[1]

	assert.Equal(t, "assethub-ui", ui.Name)
	assert.Equal(t, int32(80), ui.ContainerPort)
	assert.Equal(t, "/", ui.ReadinessPath)
	assert.Equal(t, "Never", ui.ImagePullPolicy)
}

// T-D.04: Development mode API service is NodePort with configured port
func TestTD_04_DevelopmentAPIService(t *testing.T) {
	spec := controllers.AssetHubSpec{
		Replicas:    1,
		Environment: "development",
		APINodePort: 30080,
	}
	result := controllers.ReconcileAssetHub(spec)
	apiSvc := result.Services[0]

	assert.Equal(t, "assethub-api-svc", apiSvc.Name)
	assert.Equal(t, "NodePort", apiSvc.Type)
	assert.Equal(t, int32(30080), apiSvc.NodePort)
}

// T-D.05: Development mode UI service is NodePort with configured port
func TestTD_05_DevelopmentUIService(t *testing.T) {
	spec := controllers.AssetHubSpec{
		Replicas:    1,
		Environment: "development",
		UINodePort:  30000,
	}
	result := controllers.ReconcileAssetHub(spec)
	uiSvc := result.Services[1]

	assert.Equal(t, "assethub-ui-svc", uiSvc.Name)
	assert.Equal(t, "NodePort", uiSvc.Type)
	assert.Equal(t, int32(30000), uiSvc.NodePort)
}

// T-D.06: Development mode ConfigMap has 7 env vars including RBAC_MODE=header and CLUSTER_ROLE=development
func TestTD_06_DevelopmentConfigMap(t *testing.T) {
	spec := controllers.AssetHubSpec{
		Replicas:     1,
		DBConnection: "host=postgres user=assethub password=assethub dbname=assethub port=5432 sslmode=disable",
		Environment:  "development",
	}
	result := controllers.ReconcileAssetHub(spec)
	require.Len(t, result.ConfigMaps, 1)
	cm := result.ConfigMaps[0]

	assert.Equal(t, "api-server-config", cm.Name)
	assert.Len(t, cm.Data, 7)
	assert.Equal(t, "postgres", cm.Data["DB_DRIVER"])
	assert.Equal(t, "host=postgres user=assethub password=assethub dbname=assethub port=5432 sslmode=disable", cm.Data["DB_CONNECTION_STRING"])
	assert.Equal(t, "8080", cm.Data["API_PORT"])
	assert.Equal(t, "http://localhost:30000", cm.Data["CORS_ALLOWED_ORIGINS"])
	assert.Equal(t, "info", cm.Data["LOG_LEVEL"])
	assert.Equal(t, "header", cm.Data["RBAC_MODE"])
	assert.Equal(t, "development", cm.Data["CLUSTER_ROLE"])
}

// T-D.07: OpenShift mode services are ClusterIP, no NodePort
func TestTD_07_OpenShiftServicesClusterIP(t *testing.T) {
	spec := controllers.AssetHubSpec{
		Replicas:    1,
		Environment: "openshift",
	}
	result := controllers.ReconcileAssetHub(spec)

	for _, svc := range result.Services {
		assert.Equal(t, "ClusterIP", svc.Type, "service %s should be ClusterIP", svc.Name)
		assert.Equal(t, int32(0), svc.NodePort, "service %s should have no NodePort", svc.Name)
	}
}

// T-D.08: OpenShift mode produces 2 Routes with TLS edge termination
func TestTD_08_OpenShiftRoutes(t *testing.T) {
	spec := controllers.AssetHubSpec{
		Replicas:    1,
		Environment: "openshift",
		APIHostname: "api.example.com",
		UIHostname:  "ui.example.com",
	}
	result := controllers.ReconcileAssetHub(spec)

	require.Len(t, result.Routes, 2)
	assert.Equal(t, "assethub-api-route", result.Routes[0].Name)
	assert.Equal(t, "api.example.com", result.Routes[0].Hostname)
	assert.Equal(t, "assethub-api-svc", result.Routes[0].ServiceName)
	assert.True(t, result.Routes[0].TLS)

	assert.Equal(t, "assethub-ui-route", result.Routes[1].Name)
	assert.Equal(t, "ui.example.com", result.Routes[1].Hostname)
	assert.Equal(t, "assethub-ui-svc", result.Routes[1].ServiceName)
	assert.True(t, result.Routes[1].TLS)
}

// T-D.09: OpenShift mode ConfigMap has RBAC_MODE=token
func TestTD_09_OpenShiftRBACMode(t *testing.T) {
	spec := controllers.AssetHubSpec{
		Replicas:    1,
		Environment: "openshift",
	}
	result := controllers.ReconcileAssetHub(spec)
	require.Len(t, result.ConfigMaps, 1)
	assert.Equal(t, "token", result.ConfigMaps[0].Data["RBAC_MODE"])
}

// T-D.10: OpenShift mode imagePullPolicy=IfNotPresent
func TestTD_10_OpenShiftImagePullPolicy(t *testing.T) {
	spec := controllers.AssetHubSpec{
		Replicas:    1,
		Environment: "openshift",
	}
	result := controllers.ReconcileAssetHub(spec)

	for _, dep := range result.Deployments {
		assert.Equal(t, "IfNotPresent", dep.ImagePullPolicy, "deployment %s should use IfNotPresent", dep.Name)
	}
}

// T-D.11: Defaults: omit replicas → defaults to 1
func TestTD_11_DefaultReplicas(t *testing.T) {
	spec := controllers.AssetHubSpec{}
	result := controllers.ReconcileAssetHub(spec)

	assert.Equal(t, 1, result.Deployments[0].Replicas)
	assert.Equal(t, 1, result.Deployments[1].Replicas)
}

// T-D.12: Defaults: omit environment → defaults to development
func TestTD_12_DefaultEnvironment(t *testing.T) {
	spec := controllers.AssetHubSpec{}
	result := controllers.ReconcileAssetHub(spec)

	// Development behavior: NodePort services, Never pull policy, no routes
	assert.Equal(t, "NodePort", result.Services[0].Type)
	assert.Equal(t, "Never", result.Deployments[0].ImagePullPolicy)
	assert.Len(t, result.Routes, 0)
}

// T-D.13: Defaults: omit logLevel → defaults to info
func TestTD_13_DefaultLogLevel(t *testing.T) {
	spec := controllers.AssetHubSpec{}
	result := controllers.ReconcileAssetHub(spec)

	require.Len(t, result.ConfigMaps, 1)
	assert.Equal(t, "info", result.ConfigMaps[0].Data["LOG_LEVEL"])
}

// T-D.14: UIReplicas respected
func TestTD_14_UIReplicasRespected(t *testing.T) {
	spec := controllers.AssetHubSpec{
		Replicas:   1,
		UIReplicas: 3,
	}
	result := controllers.ReconcileAssetHub(spec)

	assert.Equal(t, 1, result.Deployments[0].Replicas, "API replicas")
	assert.Equal(t, 3, result.Deployments[1].Replicas, "UI replicas")
}

// T-D.15: Cleanup returns all resource names including ConfigMap and Routes
func TestTD_15_CleanupIncludesAllResources(t *testing.T) {
	resources := controllers.CleanupAssetHub()

	assert.Contains(t, resources, "deployment/assethub-api")
	assert.Contains(t, resources, "deployment/assethub-ui")
	assert.Contains(t, resources, "service/assethub-api-svc")
	assert.Contains(t, resources, "service/assethub-ui-svc")
	assert.Contains(t, resources, "configmap/api-server-config")
	assert.Contains(t, resources, "route/assethub-api-route")
	assert.Contains(t, resources, "route/assethub-ui-route")
}

// T-9.06: Promotion to Testing → CRDs/CRs applied to cluster
func TestT9_06_ReconcilePromotion(t *testing.T) {
	entityTypes := []*models.EntityType{
		{ID: "et1", Name: "Model"},
		{ID: "et2", Name: "Tool"},
	}
	attributesByType := map[string][]*models.Attribute{
		"et1": {{ID: "a1", Name: "endpoint", Type: models.AttributeTypeString}},
		"et2": {{ID: "a2", Name: "command", Type: models.AttributeTypeString}},
	}

	result, err := controllers.ReconcilePromotion(entityTypes, attributesByType)
	require.NoError(t, err)
	assert.True(t, result.Status.Ready)
	assert.Len(t, result.CRDs, 2)
	assert.Equal(t, "Model.assethub.project-catalyst.io", result.CRDs[0].Metadata.Name)
	assert.Equal(t, "Tool.assethub.project-catalyst.io", result.CRDs[1].Metadata.Name)
}

// T-9.07: Demotion → CRDs/CRs removed from cluster
func TestT9_07_ReconcileDemotion(t *testing.T) {
	resources := controllers.ReconcileDemotion([]string{"Model", "Tool"})
	assert.Len(t, resources, 2)
	assert.Contains(t, resources, "crd/Model.assethub.project-catalyst.io")
	assert.Contains(t, resources, "crd/Tool.assethub.project-catalyst.io")
}

// T-9.08: Reconciliation failure → error in CR status conditions
func TestT9_08_ReconciliationFailure(t *testing.T) {
	// Pass a nil entity type to trigger a CRD generation error
	entityTypes := []*models.EntityType{nil}
	attributesByType := map[string][]*models.Attribute{}

	result, err := controllers.ReconcilePromotion(entityTypes, attributesByType)
	assert.Error(t, err)
	assert.False(t, result.Status.Ready)
	assert.NotEmpty(t, result.Status.Conditions)
	assert.Equal(t, "CRDGenerationFailed", result.Status.Conditions[0].Type)
}

// T-CV.07: ReconcileAssetHub with clusterRole="production" → ConfigMap has CLUSTER_ROLE=production
func TestTCV07_ClusterRoleProduction(t *testing.T) {
	spec := controllers.AssetHubSpec{
		Replicas:    1,
		ClusterRole: "production",
	}
	result := controllers.ReconcileAssetHub(spec)
	require.Len(t, result.ConfigMaps, 1)
	assert.Equal(t, "production", result.ConfigMaps[0].Data["CLUSTER_ROLE"])
}

// T-CV.08: ReconcileAssetHub with clusterRole="" → defaults to CLUSTER_ROLE=development
func TestTCV08_ClusterRoleDefault(t *testing.T) {
	spec := controllers.AssetHubSpec{
		Replicas: 1,
	}
	result := controllers.ReconcileAssetHub(spec)
	require.Len(t, result.ConfigMaps, 1)
	assert.Equal(t, "development", result.ConfigMaps[0].Data["CLUSTER_ROLE"])
}

// T-CV.09: ReconcileAssetHub with clusterRole="testing" → ConfigMap has CLUSTER_ROLE=testing
func TestTCV09_ClusterRoleTesting(t *testing.T) {
	spec := controllers.AssetHubSpec{
		Replicas:    1,
		ClusterRole: "testing",
	}
	result := controllers.ReconcileAssetHub(spec)
	require.Len(t, result.ConfigMaps, 1)
	assert.Equal(t, "testing", result.ConfigMaps[0].Data["CLUSTER_ROLE"])
}

// T-CV.10: ReconcileCatalogVersionStatus with lifecycleStage="testing" → ready=true, message set
func TestTCV10_CatalogVersionStatusTesting(t *testing.T) {
	status := controllers.ReconcileCatalogVersionStatus("testing")
	assert.True(t, status.Ready)
	assert.Contains(t, status.Message, "testing")
}

// T-CV.11: ReconcileCatalogVersionStatus with lifecycleStage="production" → ready=true, message set
func TestTCV11_CatalogVersionStatusProduction(t *testing.T) {
	status := controllers.ReconcileCatalogVersionStatus("production")
	assert.True(t, status.Ready)
	assert.Contains(t, status.Message, "production")
}

// === Catalog CR Status Tests ===

// T-16.55: First reconciliation of new Catalog CR
func TestT16_55_CatalogStatus_FirstReconcile(t *testing.T) {
	status := controllers.ReconcileCatalogStatus(1, v1alpha1.CatalogStatus{
		Ready:       false,
		DataVersion: 0,
	})
	assert.True(t, status.NeedsUpdate)
	assert.True(t, status.Ready)
	assert.Equal(t, "catalog published", status.Message)
	assert.Equal(t, 1, status.DataVersion)
}

// T-16.56: Already reconciled, same generation — no update needed
func TestT16_56_CatalogStatus_NoUpdate(t *testing.T) {
	status := controllers.ReconcileCatalogStatus(1, v1alpha1.CatalogStatus{
		Ready:              true,
		Message:            "catalog published",
		DataVersion:        1,
		ObservedGeneration: 1,
	})
	assert.False(t, status.NeedsUpdate)
}

// Spec changed (generation bumped) — DataVersion should increment
func TestCatalogStatus_SpecChanged_BumpsDataVersion(t *testing.T) {
	status := controllers.ReconcileCatalogStatus(2, v1alpha1.CatalogStatus{
		Ready:              true,
		Message:            "catalog published",
		DataVersion:        1,
		ObservedGeneration: 1,
	})
	assert.True(t, status.NeedsUpdate)
	assert.Equal(t, 2, status.DataVersion)
	assert.Equal(t, int64(2), status.ObservedGeneration)
}

// ReconcileCatalogVersionStatus: invalid lifecycle stage (line 324)
func TestReconcileCatalogVersionStatus_InvalidStage(t *testing.T) {
	result := controllers.ReconcileCatalogVersionStatus("invalid-stage")
	assert.False(t, result.Ready)
	assert.Contains(t, result.Message, "unexpected")
}

// ReconcilePromotion: nil entity type in list (line 226-235)
func TestReconcilePromotion_NilEntityType(t *testing.T) {
	result, err := controllers.ReconcilePromotion([]*models.EntityType{nil}, nil)
	assert.Error(t, err)
	assert.False(t, result.Status.Ready)
	assert.Contains(t, result.Status.Message, "nil entity type")
}

// ReconcilePromotion: empty list produces no error (line 239 not triggered)
func TestReconcilePromotion_EmptyList(t *testing.T) {
	result, err := controllers.ReconcilePromotion([]*models.EntityType{}, nil)
	require.NoError(t, err)
	assert.True(t, result.Status.Ready)
}
