package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type ExporterPluginAttributeMapping struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Default     string `json:"default,omitempty"`
}

type ExporterPluginParameterDef struct {
	Name              string                           `json:"name"`
	Type              string                           `json:"type"`
	Description       string                           `json:"description,omitempty"`
	Required          bool                             `json:"required,omitempty"`
	AttributeMappings []ExporterPluginAttributeMapping `json:"attributeMappings,omitempty"`
}

type ExporterPluginSpec struct {
	Description     string                       `json:"description,omitempty"`
	Endpoint        string                       `json:"endpoint"`
	TimeoutSeconds  int                          `json:"timeoutSeconds,omitempty"`
	ParameterSchema []ExporterPluginParameterDef `json:"parameterSchema,omitempty"`
	TrustedSubjects []string                     `json:"trustedSubjects,omitempty"`
}

type ExporterPluginStatus struct {
	Phase           string `json:"phase,omitempty"`
	LastHealthCheck string `json:"lastHealthCheck,omitempty"`
	Message         string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

type ExporterPlugin struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ExporterPluginSpec   `json:"spec,omitempty"`
	Status ExporterPluginStatus `json:"status,omitempty"`
}

func (in *ExporterPlugin) DeepCopyInto(out *ExporterPlugin) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	if in.Spec.ParameterSchema != nil {
		out.Spec.ParameterSchema = make([]ExporterPluginParameterDef, len(in.Spec.ParameterSchema))
		for i, p := range in.Spec.ParameterSchema {
			out.Spec.ParameterSchema[i] = p
			if p.AttributeMappings != nil {
				out.Spec.ParameterSchema[i].AttributeMappings = make([]ExporterPluginAttributeMapping, len(p.AttributeMappings))
				copy(out.Spec.ParameterSchema[i].AttributeMappings, p.AttributeMappings)
			}
		}
	}
	if in.Spec.TrustedSubjects != nil {
		out.Spec.TrustedSubjects = make([]string, len(in.Spec.TrustedSubjects))
		copy(out.Spec.TrustedSubjects, in.Spec.TrustedSubjects)
	}
	out.Status = in.Status
}

func (in *ExporterPlugin) DeepCopy() *ExporterPlugin {
	if in == nil {
		return nil
	}
	out := new(ExporterPlugin)
	in.DeepCopyInto(out)
	return out
}

func (in *ExporterPlugin) DeepCopyObject() runtime.Object {
	return in.DeepCopy()
}

// +kubebuilder:object:root=true

type ExporterPluginList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ExporterPlugin `json:"items"`
}

func (in *ExporterPluginList) DeepCopyInto(out *ExporterPluginList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		out.Items = make([]ExporterPlugin, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}
}

func (in *ExporterPluginList) DeepCopy() *ExporterPluginList {
	if in == nil {
		return nil
	}
	out := new(ExporterPluginList)
	in.DeepCopyInto(out)
	return out
}

func (in *ExporterPluginList) DeepCopyObject() runtime.Object {
	return in.DeepCopy()
}
