package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GroupVersion is the API group and version for AssetHub resources.
var GroupVersion = schema.GroupVersion{Group: "assethub.project-catalyst.io", Version: "v1alpha1"}

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(GroupVersion,
		&AssetHub{},
		&AssetHubList{},
		&CatalogVersion{},
		&CatalogVersionList{},
	)
	metav1.AddToGroupVersion(scheme, GroupVersion)
	return nil
}

// AssetHubSpec defines the desired state of AssetHub.
type AssetHubSpec struct {
	Replicas     int    `json:"replicas,omitempty"`
	DBConnection string `json:"dbConnection,omitempty"`
	UIReplicas   int    `json:"uiReplicas,omitempty"`
	// Environment drives networking, auth, TLS, and image pull policy.
	// Valid values: "development" (kind cluster), "openshift" (production).
	// Defaults to "development".
	Environment string `json:"environment,omitempty"`
	APINodePort int32  `json:"apiNodePort,omitempty"`
	UINodePort  int32  `json:"uiNodePort,omitempty"`
	APIHostname string `json:"apiHostname,omitempty"`
	UIHostname  string `json:"uiHostname,omitempty"`
	CORSOrigins string `json:"corsOrigins,omitempty"`
	LogLevel    string `json:"logLevel,omitempty"`
	// ClusterRole controls which catalog version lifecycle stages the API server exposes.
	// Valid values: "development" (all stages), "testing" (testing+production), "production" (production only).
	// Defaults to "development".
	ClusterRole string `json:"clusterRole,omitempty"`
}

// AssetHubStatus defines the observed state of AssetHub.
type AssetHubStatus struct {
	Ready      bool               `json:"ready"`
	Message    string             `json:"message,omitempty"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// AssetHub is the Schema for the assethubs API.
type AssetHub struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AssetHubSpec   `json:"spec,omitempty"`
	Status AssetHubStatus `json:"status,omitempty"`
}

// DeepCopyInto copies all properties into another AssetHub.
func (in *AssetHub) DeepCopyInto(out *AssetHub) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	if in.Status.Conditions != nil {
		out.Status.Conditions = make([]metav1.Condition, len(in.Status.Conditions))
		copy(out.Status.Conditions, in.Status.Conditions)
	}
}

// DeepCopy returns a deep copy of AssetHub.
func (in *AssetHub) DeepCopy() *AssetHub {
	if in == nil {
		return nil
	}
	out := new(AssetHub)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject returns a deep copy as runtime.Object.
func (in *AssetHub) DeepCopyObject() runtime.Object {
	return in.DeepCopy()
}

// +kubebuilder:object:root=true

// AssetHubList contains a list of AssetHub.
type AssetHubList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AssetHub `json:"items"`
}

// DeepCopyInto copies all properties into another AssetHubList.
func (in *AssetHubList) DeepCopyInto(out *AssetHubList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		out.Items = make([]AssetHub, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}
}

// DeepCopy returns a deep copy of AssetHubList.
func (in *AssetHubList) DeepCopy() *AssetHubList {
	if in == nil {
		return nil
	}
	out := new(AssetHubList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject returns a deep copy as runtime.Object.
func (in *AssetHubList) DeepCopyObject() runtime.Object {
	return in.DeepCopy()
}
