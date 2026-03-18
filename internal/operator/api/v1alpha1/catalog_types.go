package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// CatalogSpec defines the desired state of a Catalog discovery CR.
type CatalogSpec struct {
	CatalogName         string `json:"catalogName"`
	CatalogVersionLabel string `json:"catalogVersionLabel"`
	ValidationStatus    string `json:"validationStatus"`
	APIEndpoint         string `json:"apiEndpoint"`
	SyncVersion         int    `json:"syncVersion"`
}

// CatalogStatus defines the observed state of Catalog.
type CatalogStatus struct {
	Ready              bool               `json:"ready"`
	Message            string             `json:"message,omitempty"`
	DataVersion        int                `json:"dataVersion"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Catalog is a lightweight discovery CR for published catalogs.
type Catalog struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CatalogSpec   `json:"spec,omitempty"`
	Status CatalogStatus `json:"status,omitempty"`
}

// DeepCopyInto copies all properties into another Catalog.
func (in *Catalog) DeepCopyInto(out *Catalog) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status.Ready = in.Status.Ready
	out.Status.Message = in.Status.Message
	out.Status.DataVersion = in.Status.DataVersion
	out.Status.ObservedGeneration = in.Status.ObservedGeneration
	if in.Status.Conditions != nil {
		out.Status.Conditions = make([]metav1.Condition, len(in.Status.Conditions))
		copy(out.Status.Conditions, in.Status.Conditions)
	}
}

// DeepCopy returns a deep copy of Catalog.
func (in *Catalog) DeepCopy() *Catalog {
	if in == nil {
		return nil
	}
	out := new(Catalog)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject returns a deep copy as runtime.Object.
func (in *Catalog) DeepCopyObject() runtime.Object {
	return in.DeepCopy()
}

// +kubebuilder:object:root=true

// CatalogList contains a list of Catalog.
type CatalogList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Catalog `json:"items"`
}

// DeepCopyInto copies all properties into another CatalogList.
func (in *CatalogList) DeepCopyInto(out *CatalogList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		out.Items = make([]Catalog, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}
}

// DeepCopy returns a deep copy of CatalogList.
func (in *CatalogList) DeepCopy() *CatalogList {
	if in == nil {
		return nil
	}
	out := new(CatalogList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject returns a deep copy as runtime.Object.
func (in *CatalogList) DeepCopyObject() runtime.Object {
	return in.DeepCopy()
}
