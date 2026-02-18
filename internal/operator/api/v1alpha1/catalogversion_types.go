package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// CatalogVersionSpec defines the desired state of a CatalogVersion discovery CR.
type CatalogVersionSpec struct {
	VersionLabel   string   `json:"versionLabel"`
	Description    string   `json:"description,omitempty"`
	LifecycleStage string   `json:"lifecycleStage"`
	EntityTypes    []string `json:"entityTypes,omitempty"`
}

// CatalogVersionStatus defines the observed state of CatalogVersion.
type CatalogVersionStatus struct {
	Ready      bool               `json:"ready"`
	Message    string             `json:"message,omitempty"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// CatalogVersion is a lightweight discovery CR for promoted catalog versions.
type CatalogVersion struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CatalogVersionSpec   `json:"spec,omitempty"`
	Status CatalogVersionStatus `json:"status,omitempty"`
}

// DeepCopyInto copies all properties into another CatalogVersion.
func (in *CatalogVersion) DeepCopyInto(out *CatalogVersion) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	if in.Spec.EntityTypes != nil {
		out.Spec.EntityTypes = make([]string, len(in.Spec.EntityTypes))
		copy(out.Spec.EntityTypes, in.Spec.EntityTypes)
	}
	if in.Status.Conditions != nil {
		out.Status.Conditions = make([]metav1.Condition, len(in.Status.Conditions))
		copy(out.Status.Conditions, in.Status.Conditions)
	}
}

// DeepCopy returns a deep copy of CatalogVersion.
func (in *CatalogVersion) DeepCopy() *CatalogVersion {
	if in == nil {
		return nil
	}
	out := new(CatalogVersion)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject returns a deep copy as runtime.Object.
func (in *CatalogVersion) DeepCopyObject() runtime.Object {
	return in.DeepCopy()
}

// +kubebuilder:object:root=true

// CatalogVersionList contains a list of CatalogVersion.
type CatalogVersionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CatalogVersion `json:"items"`
}

// DeepCopyInto copies all properties into another CatalogVersionList.
func (in *CatalogVersionList) DeepCopyInto(out *CatalogVersionList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		out.Items = make([]CatalogVersion, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}
}

// DeepCopy returns a deep copy of CatalogVersionList.
func (in *CatalogVersionList) DeepCopy() *CatalogVersionList {
	if in == nil {
		return nil
	}
	out := new(CatalogVersionList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject returns a deep copy as runtime.Object.
func (in *CatalogVersionList) DeepCopyObject() runtime.Object {
	return in.DeepCopy()
}
