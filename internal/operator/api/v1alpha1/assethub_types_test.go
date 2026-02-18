package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestAssetHub_DeepCopy(t *testing.T) {
	ah := &AssetHub{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec:       AssetHubSpec{Replicas: 3, DBConnection: "sqlite://db", UIReplicas: 1},
		Status:     AssetHubStatus{Ready: true, Message: "ok", Conditions: []metav1.Condition{{Type: "Ready", Status: "True"}}},
	}
	copy := ah.DeepCopy()
	require.NotNil(t, copy)
	assert.Equal(t, "test", copy.Name)
	assert.Equal(t, 3, copy.Spec.Replicas)
	assert.Len(t, copy.Status.Conditions, 1)

	// Mutating copy should not affect original
	copy.Spec.Replicas = 5
	assert.Equal(t, 3, ah.Spec.Replicas)
}

func TestAssetHub_DeepCopyNil(t *testing.T) {
	var ah *AssetHub
	assert.Nil(t, ah.DeepCopy())
}

func TestAssetHub_DeepCopyObject(t *testing.T) {
	ah := &AssetHub{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec:       AssetHubSpec{Replicas: 1},
	}
	obj := ah.DeepCopyObject()
	assert.NotNil(t, obj)
}

func TestAssetHub_DeepCopyInto_NilConditions(t *testing.T) {
	ah := &AssetHub{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Status:     AssetHubStatus{Ready: false},
	}
	out := &AssetHub{}
	ah.DeepCopyInto(out)
	assert.Equal(t, "test", out.Name)
	assert.Nil(t, out.Status.Conditions)
}

func TestAssetHubList_DeepCopy(t *testing.T) {
	list := &AssetHubList{
		Items: []AssetHub{
			{ObjectMeta: metav1.ObjectMeta{Name: "a"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "b"}},
		},
	}
	copy := list.DeepCopy()
	require.NotNil(t, copy)
	assert.Len(t, copy.Items, 2)

	// Mutating copy should not affect original
	copy.Items[0].Name = "mutated"
	assert.Equal(t, "a", list.Items[0].Name)
}

func TestAssetHubList_DeepCopyNil(t *testing.T) {
	var list *AssetHubList
	assert.Nil(t, list.DeepCopy())
}

func TestAssetHubList_DeepCopyObject(t *testing.T) {
	list := &AssetHubList{Items: []AssetHub{{ObjectMeta: metav1.ObjectMeta{Name: "a"}}}}
	obj := list.DeepCopyObject()
	assert.NotNil(t, obj)
}

func TestAssetHubList_DeepCopyInto_NilItems(t *testing.T) {
	list := &AssetHubList{}
	out := &AssetHubList{}
	list.DeepCopyInto(out)
	assert.Nil(t, out.Items)
}

func TestAddKnownTypes(t *testing.T) {
	s := runtime.NewScheme()
	err := AddToScheme(s)
	require.NoError(t, err)

	// Verify types are registered
	gvk := GroupVersion.WithKind("AssetHub")
	obj, err := s.New(gvk)
	require.NoError(t, err)
	_, ok := obj.(*AssetHub)
	assert.True(t, ok)
}
