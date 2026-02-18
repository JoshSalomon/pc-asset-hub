package controllers

import (
	"fmt"

	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/operator/crdgen"
)

// AssetHubSpec defines the desired state of the AssetHub installation.
type AssetHubSpec struct {
	Replicas     int    `json:"replicas"`
	DBConnection string `json:"db_connection"`
	UIReplicas   int    `json:"uiReplicas"`
	Environment  string `json:"environment"`
	APINodePort  int32  `json:"apiNodePort"`
	UINodePort   int32  `json:"uiNodePort"`
	APIHostname  string `json:"apiHostname"`
	UIHostname   string `json:"uiHostname"`
	CORSOrigins  string `json:"corsOrigins"`
	LogLevel     string `json:"logLevel"`
	ClusterRole  string `json:"clusterRole"`
}

// AssetHubStatus defines the observed state.
type AssetHubStatus struct {
	Ready      bool        `json:"ready"`
	Message    string      `json:"message"`
	Conditions []Condition `json:"conditions,omitempty"`
}

// Condition represents a status condition.
type Condition struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// ReconcileResult describes what resources should exist.
type ReconcileResult struct {
	Deployments []DeploymentSpec `json:"deployments"`
	Services    []ServiceSpec    `json:"services"`
	ConfigMaps  []ConfigMapSpec  `json:"configMaps"`
	Routes      []RouteSpec      `json:"routes"`
}

// DeploymentSpec is a simplified deployment specification.
type DeploymentSpec struct {
	Name               string `json:"name"`
	Image              string `json:"image"`
	Replicas           int    `json:"replicas"`
	ImagePullPolicy    string `json:"imagePullPolicy"`
	EnvFrom            string `json:"envFrom"`
	ContainerPort      int32  `json:"containerPort"`
	ReadinessPath      string `json:"readinessPath"`
	LivenessPath       string `json:"livenessPath"`
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
}

// ServiceSpec is a simplified service specification.
type ServiceSpec struct {
	Name     string `json:"name"`
	Port     int    `json:"port"`
	Type     string `json:"type"`
	NodePort int32  `json:"nodePort"`
}

// ConfigMapSpec is a simplified ConfigMap specification.
type ConfigMapSpec struct {
	Name string            `json:"name"`
	Data map[string]string `json:"data"`
}

// RouteSpec is a simplified OpenShift Route specification.
type RouteSpec struct {
	Name        string `json:"name"`
	Hostname    string `json:"hostname"`
	ServiceName string `json:"serviceName"`
	ServicePort int32  `json:"servicePort"`
	TLS         bool   `json:"tls"`
}

// ReconcileAssetHub determines what resources should be created for an AssetHub CR.
func ReconcileAssetHub(spec AssetHubSpec) *ReconcileResult {
	replicas := spec.Replicas
	if replicas <= 0 {
		replicas = 1
	}
	uiReplicas := spec.UIReplicas
	if uiReplicas <= 0 {
		uiReplicas = 1
	}

	env := spec.Environment
	if env == "" {
		env = "development"
	}
	logLevel := spec.LogLevel
	if logLevel == "" {
		logLevel = "info"
	}

	// Environment-driven settings
	var imagePullPolicy, serviceType, rbacMode string
	var apiNodePort, uiNodePort int32
	switch env {
	case "openshift":
		imagePullPolicy = "IfNotPresent"
		serviceType = "ClusterIP"
		rbacMode = "token"
	default: // development
		imagePullPolicy = "Never"
		serviceType = "NodePort"
		rbacMode = "header"
		apiNodePort = spec.APINodePort
		if apiNodePort == 0 {
			apiNodePort = 30080
		}
		uiNodePort = spec.UINodePort
		if uiNodePort == 0 {
			uiNodePort = 30000
		}
	}

	corsOrigins := spec.CORSOrigins
	if corsOrigins == "" && env == "development" {
		corsOrigins = fmt.Sprintf("http://localhost:%d", uiNodePort)
	}

	clusterRole := spec.ClusterRole
	if clusterRole == "" {
		clusterRole = "development"
	}

	configMapName := "api-server-config"

	result := &ReconcileResult{
		Deployments: []DeploymentSpec{
			{
				Name:               "assethub-api",
				Image:              "assethub/api-server:latest",
				Replicas:           replicas,
				ImagePullPolicy:    imagePullPolicy,
				EnvFrom:            configMapName,
				ContainerPort:      8080,
				ReadinessPath:      "/readyz",
				LivenessPath:       "/healthz",
				ServiceAccountName: "assethub-api-server",
			},
			{
				Name:            "assethub-ui",
				Image:           "assethub/ui:latest",
				Replicas:        uiReplicas,
				ImagePullPolicy: imagePullPolicy,
				ContainerPort:   80,
				ReadinessPath:   "/",
			},
		},
		Services: []ServiceSpec{
			{Name: "assethub-api-svc", Port: 8080, Type: serviceType, NodePort: apiNodePort},
			{Name: "assethub-ui-svc", Port: 80, Type: serviceType, NodePort: uiNodePort},
		},
		ConfigMaps: []ConfigMapSpec{
			{
				Name: configMapName,
				Data: map[string]string{
					"DB_DRIVER":            "postgres",
					"DB_CONNECTION_STRING": spec.DBConnection,
					"API_PORT":             "8080",
					"CORS_ALLOWED_ORIGINS": corsOrigins,
					"LOG_LEVEL":            logLevel,
					"RBAC_MODE":            rbacMode,
					"CLUSTER_ROLE":         clusterRole,
				},
			},
		},
	}

	// Routes only in OpenShift mode
	if env == "openshift" {
		apiHostname := spec.APIHostname
		if apiHostname == "" {
			apiHostname = "api.assethub.example.com"
		}
		uiHostname := spec.UIHostname
		if uiHostname == "" {
			uiHostname = "ui.assethub.example.com"
		}
		result.Routes = []RouteSpec{
			{Name: "assethub-api-route", Hostname: apiHostname, ServiceName: "assethub-api-svc", ServicePort: 8080, TLS: true},
			{Name: "assethub-ui-route", Hostname: uiHostname, ServiceName: "assethub-ui-svc", ServicePort: 80, TLS: true},
		}
	}

	return result
}

// CleanupAssetHub returns the list of resources to delete when an AssetHub CR is removed.
func CleanupAssetHub() []string {
	return []string{
		"deployment/assethub-api",
		"deployment/assethub-ui",
		"service/assethub-api-svc",
		"service/assethub-ui-svc",
		"configmap/api-server-config",
		"route/assethub-api-route",
		"route/assethub-ui-route",
	}
}

// PromotionResult describes what happens when a catalog version is promoted.
type PromotionResult struct {
	CRDs   []*crdgen.CRDSpec
	CRs    []*crdgen.CRInstance
	Status AssetHubStatus
}

// ReconcilePromotion generates CRDs/CRs for a promoted catalog version.
func ReconcilePromotion(entityTypes []*models.EntityType, attributesByType map[string][]*models.Attribute) (*PromotionResult, error) {
	result := &PromotionResult{
		Status: AssetHubStatus{Ready: true, Message: "promotion reconciled"},
	}

	for _, et := range entityTypes {
		if et == nil {
			return &PromotionResult{
				Status: AssetHubStatus{
					Ready:   false,
					Message: "nil entity type in promotion list",
					Conditions: []Condition{
						{Type: "CRDGenerationFailed", Status: "True", Message: "nil entity type"},
					},
				},
			}, fmt.Errorf("nil entity type in promotion list")
		}
		attrs := attributesByType[et.ID]
		crd, err := crdgen.GenerateCRD(et, attrs)
		if err != nil {
			return &PromotionResult{
				Status: AssetHubStatus{
					Ready:   false,
					Message: fmt.Sprintf("CRD generation failed for %s: %v", et.Name, err),
					Conditions: []Condition{
						{Type: "CRDGenerationFailed", Status: "True", Message: err.Error()},
					},
				},
			}, err
		}
		result.CRDs = append(result.CRDs, crd)
	}

	return result, nil
}

// ReconcileDemotion returns what to clean up when a catalog version is demoted.
func ReconcileDemotion(entityTypeNames []string) []string {
	var resources []string
	for _, name := range entityTypeNames {
		resources = append(resources, "crd/"+name+".assethub.project-catalyst.io")
	}
	return resources
}

// CatalogVersionStatusResult describes the desired status for a CatalogVersion CR.
type CatalogVersionStatusResult struct {
	Ready   bool   `json:"ready"`
	Message string `json:"message"`
}

// ReconcileCatalogVersionStatus determines the status for a CatalogVersion CR
// based on its lifecycle stage.
func ReconcileCatalogVersionStatus(lifecycleStage string) CatalogVersionStatusResult {
	switch lifecycleStage {
	case "testing":
		return CatalogVersionStatusResult{
			Ready:   true,
			Message: "catalog version available for testing",
		}
	case "production":
		return CatalogVersionStatusResult{
			Ready:   true,
			Message: "catalog version in production",
		}
	default:
		return CatalogVersionStatusResult{
			Ready:   false,
			Message: fmt.Sprintf("unexpected lifecycle stage: %s", lifecycleStage),
		}
	}
}
