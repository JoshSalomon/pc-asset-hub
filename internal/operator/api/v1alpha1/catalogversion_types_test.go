package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// T-CV.01: CatalogVersion DeepCopy preserves all fields including EntityTypes slice.
func TestTCV01_CatalogVersionDeepCopyPreservesAllFields(t *testing.T) {
	cv := &CatalogVersion{
		ObjectMeta: metav1.ObjectMeta{Name: "release-2-3", Namespace: "assethub"},
		Spec: CatalogVersionSpec{
			VersionLabel:   "Release 2.3",
			Description:    "Q1 release",
			LifecycleStage: "production",
			EntityTypes:    []string{"Device", "Application", "Network"},
		},
		Status: CatalogVersionStatus{
			Ready:   true,
			Message: "catalog version registered",
			Conditions: []metav1.Condition{
				{Type: "Ready", Status: "True"},
			},
		},
	}

	cp := cv.DeepCopy()
	require.NotNil(t, cp)
	assert.Equal(t, "release-2-3", cp.Name)
	assert.Equal(t, "assethub", cp.Namespace)
	assert.Equal(t, "Release 2.3", cp.Spec.VersionLabel)
	assert.Equal(t, "Q1 release", cp.Spec.Description)
	assert.Equal(t, "production", cp.Spec.LifecycleStage)
	assert.Equal(t, []string{"Device", "Application", "Network"}, cp.Spec.EntityTypes)
	assert.True(t, cp.Status.Ready)
	assert.Equal(t, "catalog version registered", cp.Status.Message)
	assert.Len(t, cp.Status.Conditions, 1)

	// Slices are independent copies
	cp.Spec.EntityTypes[0] = "Mutated"
	assert.Equal(t, "Device", cv.Spec.EntityTypes[0])

	cp.Status.Conditions[0].Type = "Mutated"
	assert.Equal(t, "Ready", cv.Status.Conditions[0].Type)
}

// T-CV.02: CatalogVersion DeepCopy of nil returns nil.
func TestTCV02_CatalogVersionDeepCopyNil(t *testing.T) {
	var cv *CatalogVersion
	assert.Nil(t, cv.DeepCopy())
}

// T-CV.03: CatalogVersion DeepCopyObject returns valid runtime.Object.
func TestTCV03_CatalogVersionDeepCopyObject(t *testing.T) {
	cv := &CatalogVersion{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec:       CatalogVersionSpec{VersionLabel: "v1"},
	}
	obj := cv.DeepCopyObject()
	require.NotNil(t, obj)
	_, ok := obj.(*CatalogVersion)
	assert.True(t, ok)
}

// T-CV.04: CatalogVersionList DeepCopy preserves items.
func TestTCV04_CatalogVersionListDeepCopy(t *testing.T) {
	list := &CatalogVersionList{
		Items: []CatalogVersion{
			{ObjectMeta: metav1.ObjectMeta{Name: "a"}, Spec: CatalogVersionSpec{EntityTypes: []string{"Device"}}},
			{ObjectMeta: metav1.ObjectMeta{Name: "b"}, Spec: CatalogVersionSpec{EntityTypes: []string{"App"}}},
		},
	}
	cp := list.DeepCopy()
	require.NotNil(t, cp)
	assert.Len(t, cp.Items, 2)
	assert.Equal(t, "a", cp.Items[0].Name)
	assert.Equal(t, "b", cp.Items[1].Name)

	// Mutation of copy doesn't affect original
	cp.Items[0].Name = "mutated"
	assert.Equal(t, "a", list.Items[0].Name)

	cp.Items[0].Spec.EntityTypes[0] = "Mutated"
	assert.Equal(t, "Device", list.Items[0].Spec.EntityTypes[0])
}

// T-CV.05: CatalogVersionList DeepCopy of nil returns nil.
func TestTCV05_CatalogVersionListDeepCopyNil(t *testing.T) {
	var list *CatalogVersionList
	assert.Nil(t, list.DeepCopy())
}

// T-CV.06: AddToScheme registers CatalogVersion and CatalogVersionList.
func TestTCV06_AddToSchemeRegistersCatalogVersion(t *testing.T) {
	s := runtime.NewScheme()
	err := AddToScheme(s)
	require.NoError(t, err)

	// CatalogVersion is registered
	gvk := GroupVersion.WithKind("CatalogVersion")
	obj, err := s.New(gvk)
	require.NoError(t, err)
	_, ok := obj.(*CatalogVersion)
	assert.True(t, ok)

	// CatalogVersionList is registered
	gvk = GroupVersion.WithKind("CatalogVersionList")
	obj, err = s.New(gvk)
	require.NoError(t, err)
	_, ok = obj.(*CatalogVersionList)
	assert.True(t, ok)
}
