package v1alpha1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// T-16.54 (partial): Catalog CR type has DataVersion in status, defaults to 0
func TestCatalog_DeepCopyPreservesAllFields(t *testing.T) {
	cat := &Catalog{
		ObjectMeta: metav1.ObjectMeta{Name: "prod-catalog", Namespace: "assethub"},
		Spec: CatalogSpec{
			CatalogName:         "prod-catalog",
			CatalogVersionLabel: "v1.0",
			ValidationStatus:    "valid",
			APIEndpoint:         "/api/data/v1/catalogs/prod-catalog",
		},
		Status: CatalogStatus{
			Ready:       true,
			Message:     "catalog published",
			DataVersion: 3,
			Conditions: []metav1.Condition{
				{Type: "Ready", Status: "True"},
			},
		},
	}

	cp := cat.DeepCopy()
	require.NotNil(t, cp)
	assert.Equal(t, "prod-catalog", cp.Name)
	assert.Equal(t, "assethub", cp.Namespace)
	assert.Equal(t, "prod-catalog", cp.Spec.CatalogName)
	assert.Equal(t, "v1.0", cp.Spec.CatalogVersionLabel)
	assert.Equal(t, "valid", cp.Spec.ValidationStatus)
	assert.Equal(t, "/api/data/v1/catalogs/prod-catalog", cp.Spec.APIEndpoint)
	assert.True(t, cp.Status.Ready)
	assert.Equal(t, "catalog published", cp.Status.Message)
	assert.Equal(t, 3, cp.Status.DataVersion)
	assert.Len(t, cp.Status.Conditions, 1)

	// Mutations are independent
	cp.Status.Conditions[0].Type = "Mutated"
	assert.Equal(t, "Ready", cat.Status.Conditions[0].Type)
}

func TestCatalog_DeepCopyNil(t *testing.T) {
	var cat *Catalog
	assert.Nil(t, cat.DeepCopy())
}

func TestCatalog_DeepCopyObject(t *testing.T) {
	cat := &Catalog{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec:       CatalogSpec{CatalogName: "test"},
	}
	obj := cat.DeepCopyObject()
	require.NotNil(t, obj)
	_, ok := obj.(*Catalog)
	assert.True(t, ok)
}

func TestCatalogList_DeepCopy(t *testing.T) {
	list := &CatalogList{
		Items: []Catalog{
			{ObjectMeta: metav1.ObjectMeta{Name: "a"}, Spec: CatalogSpec{CatalogName: "a"}},
			{ObjectMeta: metav1.ObjectMeta{Name: "b"}, Spec: CatalogSpec{CatalogName: "b"}},
		},
	}
	cp := list.DeepCopy()
	require.NotNil(t, cp)
	assert.Len(t, cp.Items, 2)
	assert.Equal(t, "a", cp.Items[0].Name)

	cp.Items[0].Name = "mutated"
	assert.Equal(t, "a", list.Items[0].Name)
}

func TestCatalogList_DeepCopyNil(t *testing.T) {
	var list *CatalogList
	assert.Nil(t, list.DeepCopy())
}

func TestCatalog_RegisteredInScheme(t *testing.T) {
	s := runtime.NewScheme()
	err := AddToScheme(s)
	require.NoError(t, err)

	gvk := GroupVersion.WithKind("Catalog")
	obj, err := s.New(gvk)
	require.NoError(t, err)
	_, ok := obj.(*Catalog)
	assert.True(t, ok)

	gvk = GroupVersion.WithKind("CatalogList")
	obj, err = s.New(gvk)
	require.NoError(t, err)
	_, ok = obj.(*CatalogList)
	assert.True(t, ok)
}
