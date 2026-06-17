package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// T-36.52: ExporterPlugin DeepCopy independence
func TestExporterPlugin_DeepCopy_Independence(t *testing.T) {
	ep := &ExporterPlugin{
		ObjectMeta: metav1.ObjectMeta{Name: "my-exporter", Namespace: "assethub"},
		Spec: ExporterPluginSpec{
			Description:    "Custom exporter",
			Endpoint:       "https://my-exporter.ns.svc.cluster.local",
			TimeoutSeconds: 15,
			ParameterSchema: []ExporterPluginParameterDef{
				{Name: "ns", Type: "string", Description: "Target namespace", Required: true},
			},
			TrustedSubjects: []string{
				"system:serviceaccount:assethub:assethub-api-server",
			},
		},
		Status: ExporterPluginStatus{
			Phase:           "Ready",
			LastHealthCheck: "2026-06-07T12:00:00Z",
			Message:         "",
		},
	}

	cp := ep.DeepCopy()
	require.NotNil(t, cp)
	assert.Equal(t, "my-exporter", cp.Name)
	assert.Equal(t, "Custom exporter", cp.Spec.Description)
	assert.Equal(t, "https://my-exporter.ns.svc.cluster.local", cp.Spec.Endpoint)
	assert.Equal(t, 15, cp.Spec.TimeoutSeconds)
	require.Len(t, cp.Spec.ParameterSchema, 1)
	assert.Equal(t, "ns", cp.Spec.ParameterSchema[0].Name)
	require.Len(t, cp.Spec.TrustedSubjects, 1)
	assert.Equal(t, "Ready", cp.Status.Phase)

	// Mutations are independent
	cp.Spec.ParameterSchema[0].Name = "mutated"
	assert.Equal(t, "ns", ep.Spec.ParameterSchema[0].Name)

	cp.Spec.TrustedSubjects[0] = "mutated"
	assert.Equal(t, "system:serviceaccount:assethub:assethub-api-server", ep.Spec.TrustedSubjects[0])
}

// DeepCopy preserves and isolates AttributeMappings
func TestExporterPlugin_DeepCopy_AttributeMappings(t *testing.T) {
	ep := &ExporterPlugin{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec: ExporterPluginSpec{
			Endpoint: "http://localhost",
			ParameterSchema: []ExporterPluginParameterDef{
				{
					Name: "server_type", Type: "entity_type", Required: true,
					AttributeMappings: []ExporterPluginAttributeMapping{
						{Name: "route_name_attr", Required: true, Default: "route_name"},
						{Name: "mcp_path_attr", Default: "mcp_path"},
					},
				},
				{Name: "tool_type", Type: "entity_type"},
			},
		},
	}

	cp := ep.DeepCopy()
	require.Len(t, cp.Spec.ParameterSchema[0].AttributeMappings, 2)
	assert.Equal(t, "route_name_attr", cp.Spec.ParameterSchema[0].AttributeMappings[0].Name)
	assert.Equal(t, "route_name", cp.Spec.ParameterSchema[0].AttributeMappings[0].Default)
	assert.Nil(t, cp.Spec.ParameterSchema[1].AttributeMappings)

	// Mutation independence
	cp.Spec.ParameterSchema[0].AttributeMappings[0].Name = "mutated"
	assert.Equal(t, "route_name_attr", ep.Spec.ParameterSchema[0].AttributeMappings[0].Name)
}

// T-36.53: ExporterPlugin DeepCopy nil returns nil
func TestExporterPlugin_DeepCopy_Nil(t *testing.T) {
	var ep *ExporterPlugin
	assert.Nil(t, ep.DeepCopy())
}

// T-36.54: ExporterPlugin DeepCopyObject returns non-nil runtime.Object
func TestExporterPlugin_DeepCopyObject(t *testing.T) {
	ep := &ExporterPlugin{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec:       ExporterPluginSpec{Endpoint: "http://localhost"},
	}
	obj := ep.DeepCopyObject()
	require.NotNil(t, obj)
	_, ok := obj.(*ExporterPlugin)
	assert.True(t, ok)
}

// Copilot review: Default field preserved through DeepCopy
func TestExporterPlugin_DeepCopy_DefaultField(t *testing.T) {
	ep := &ExporterPlugin{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec: ExporterPluginSpec{
			Endpoint: "http://localhost",
			ParameterSchema: []ExporterPluginParameterDef{
				{Name: "ns", Type: "string", Default: "prod"},
			},
		},
	}
	cp := ep.DeepCopy()
	require.Len(t, cp.Spec.ParameterSchema, 1)
	assert.Equal(t, "prod", cp.Spec.ParameterSchema[0].Default)

	cp.Spec.ParameterSchema[0].Default = "dev"
	assert.Equal(t, "prod", ep.Spec.ParameterSchema[0].Default)
}

// T-36.55: ExporterPlugin nil slices handled (ParameterSchema, TrustedSubjects)
func TestExporterPlugin_NilSlices(t *testing.T) {
	ep := &ExporterPlugin{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec: ExporterPluginSpec{
			Endpoint: "http://localhost",
		},
	}

	cp := ep.DeepCopy()
	require.NotNil(t, cp)
	assert.Nil(t, cp.Spec.ParameterSchema)
	assert.Nil(t, cp.Spec.TrustedSubjects)
}

// T-36.56: ExporterPluginList DeepCopy
func TestExporterPluginList_DeepCopy(t *testing.T) {
	list := &ExporterPluginList{
		Items: []ExporterPlugin{
			{ObjectMeta: metav1.ObjectMeta{Name: "a"}, Spec: ExporterPluginSpec{Endpoint: "http://a"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "b"}, Spec: ExporterPluginSpec{Endpoint: "http://b"}},
		},
	}
	cp := list.DeepCopy()
	require.NotNil(t, cp)
	require.Len(t, cp.Items, 2)
	assert.Equal(t, "a", cp.Items[0].Name)

	cp.Items[0].Name = "mutated"
	assert.Equal(t, "a", list.Items[0].Name)
}

// T-36.56b: ExporterPluginList DeepCopy nil returns nil
func TestExporterPluginList_DeepCopy_Nil(t *testing.T) {
	var list *ExporterPluginList
	assert.Nil(t, list.DeepCopy())
}

// T-36.57: ExporterPluginList DeepCopyObject
func TestExporterPluginList_DeepCopyObject(t *testing.T) {
	list := &ExporterPluginList{
		Items: []ExporterPlugin{{ObjectMeta: metav1.ObjectMeta{Name: "a"}}},
	}
	obj := list.DeepCopyObject()
	require.NotNil(t, obj)
	cpList, ok := obj.(*ExporterPluginList)
	require.True(t, ok)
	assert.Len(t, cpList.Items, 1)
}

// T-36.58: AddToScheme registers ExporterPlugin kind
func TestExporterPlugin_RegisteredInScheme(t *testing.T) {
	s := runtime.NewScheme()
	err := AddToScheme(s)
	require.NoError(t, err)

	gvk := GroupVersion.WithKind("ExporterPlugin")
	obj, err := s.New(gvk)
	require.NoError(t, err)
	_, ok := obj.(*ExporterPlugin)
	assert.True(t, ok)

	gvk = GroupVersion.WithKind("ExporterPluginList")
	obj, err = s.New(gvk)
	require.NoError(t, err)
	_, ok = obj.(*ExporterPluginList)
	assert.True(t, ok)
}
