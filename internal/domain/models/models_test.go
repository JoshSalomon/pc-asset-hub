package models

import (
	"testing"
	"time"
)

func TestEntityTypeInstantiation(t *testing.T) {
	now := time.Now()
	et := EntityType{
		ID:        "test-id",
		Name:      "Model",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if et.ID != "test-id" || et.Name != "Model" {
		t.Fatal("EntityType fields not set correctly")
	}
}

func TestEntityTypeVersionInstantiation(t *testing.T) {
	etv := EntityTypeVersion{
		ID:           "v-id",
		EntityTypeID: "et-id",
		Version:      1,
		Description:  "Initial version",
		CreatedAt:    time.Now(),
	}
	if etv.Version != 1 || etv.EntityTypeID != "et-id" {
		t.Fatal("EntityTypeVersion fields not set correctly")
	}
}

func TestAttributeTypes(t *testing.T) {
	if AttributeTypeString != "string" {
		t.Fatal("AttributeTypeString should be 'string'")
	}
	if AttributeTypeNumber != "number" {
		t.Fatal("AttributeTypeNumber should be 'number'")
	}
	if AttributeTypeEnum != "enum" {
		t.Fatal("AttributeTypeEnum should be 'enum'")
	}
}

func TestAssociationTypes(t *testing.T) {
	if AssociationTypeContainment != "containment" {
		t.Fatal("AssociationTypeContainment should be 'containment'")
	}
	if AssociationTypeDirectional != "directional" {
		t.Fatal("AssociationTypeDirectional should be 'directional'")
	}
	if AssociationTypeBidirectional != "bidirectional" {
		t.Fatal("AssociationTypeBidirectional should be 'bidirectional'")
	}
}

func TestLifecycleStages(t *testing.T) {
	if LifecycleStageDevelopment != "development" {
		t.Fatal("LifecycleStageDevelopment should be 'development'")
	}
	if LifecycleStageTesting != "testing" {
		t.Fatal("LifecycleStageTesting should be 'testing'")
	}
	if LifecycleStageProduction != "production" {
		t.Fatal("LifecycleStageProduction should be 'production'")
	}
}

func TestEntityInstanceWithNilDeletedAt(t *testing.T) {
	inst := EntityInstance{
		ID:               "inst-id",
		EntityTypeID:     "et-id",
		CatalogID: "cv-id",
		Name:             "test-instance",
		Version:          1,
		DeletedAt:        nil,
	}
	if inst.DeletedAt != nil {
		t.Fatal("DeletedAt should be nil for active instance")
	}
}

func TestInstanceAttributeValueWithNumber(t *testing.T) {
	val := 42.5
	iav := InstanceAttributeValue{
		ID:              "iav-id",
		InstanceID:      "inst-id",
		InstanceVersion: 1,
		AttributeID:     "attr-id",
		ValueNumber:     &val,
	}
	if iav.ValueNumber == nil || *iav.ValueNumber != 42.5 {
		t.Fatal("ValueNumber not set correctly")
	}
}

func TestListParams(t *testing.T) {
	params := ListParams{
		Offset:   0,
		Limit:    10,
		SortBy:   "name",
		SortDesc: false,
		Filters:  map[string]string{"status": "active"},
	}
	if params.Limit != 10 || params.Filters["status"] != "active" {
		t.Fatal("ListParams fields not set correctly")
	}
}

func TestIsSystemAttributeName(t *testing.T) {
	if !IsSystemAttributeName("name") {
		t.Fatal("expected 'name' to be a system attribute")
	}
	if !IsSystemAttributeName("description") {
		t.Fatal("expected 'description' to be a system attribute")
	}
	if IsSystemAttributeName("hostname") {
		t.Fatal("expected 'hostname' to NOT be a system attribute")
	}
	if IsSystemAttributeName("Name") {
		t.Fatal("expected 'Name' (uppercase) to NOT be a system attribute")
	}
	if IsSystemAttributeName("") {
		t.Fatal("expected empty string to NOT be a system attribute")
	}
}
