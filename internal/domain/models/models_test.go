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

func TestBaseTypes(t *testing.T) {
	if BaseTypeString != "string" {
		t.Fatal("BaseTypeString should be 'string'")
	}
	if BaseTypeInteger != "integer" {
		t.Fatal("BaseTypeInteger should be 'integer'")
	}
	if BaseTypeNumber != "number" {
		t.Fatal("BaseTypeNumber should be 'number'")
	}
	if BaseTypeBoolean != "boolean" {
		t.Fatal("BaseTypeBoolean should be 'boolean'")
	}
	if BaseTypeDate != "date" {
		t.Fatal("BaseTypeDate should be 'date'")
	}
	if BaseTypeURL != "url" {
		t.Fatal("BaseTypeURL should be 'url'")
	}
	if BaseTypeEnum != "enum" {
		t.Fatal("BaseTypeEnum should be 'enum'")
	}
	if BaseTypeList != "list" {
		t.Fatal("BaseTypeList should be 'list'")
	}
	if BaseTypeJSON != "json" {
		t.Fatal("BaseTypeJSON should be 'json'")
	}
	if !ValidBaseTypes[BaseTypeString] {
		t.Fatal("BaseTypeString should be in ValidBaseTypes")
	}
}

func TestIsCorruptedConstraints(t *testing.T) {
	// Normal constraints — not corrupted
	normal := map[string]any{"max_length": float64(12)}
	if IsCorruptedConstraints(normal) {
		t.Fatal("normal constraints should not be corrupted")
	}

	// Empty constraints — not corrupted
	empty := map[string]any{}
	if IsCorruptedConstraints(empty) {
		t.Fatal("empty constraints should not be corrupted")
	}

	// Corrupted constraints — has _raw key
	corrupted := map[string]any{"_raw": "not{valid json"}
	if !IsCorruptedConstraints(corrupted) {
		t.Fatal("constraints with _raw should be detected as corrupted")
	}
}

func TestExtractRawConstraints(t *testing.T) {
	corrupted := map[string]any{"_raw": "not{valid json"}
	raw := ExtractRawConstraints(corrupted)
	if raw != "not{valid json" {
		t.Fatalf("expected raw string, got %q", raw)
	}

	// Non-corrupted returns empty
	normal := map[string]any{"max_length": float64(12)}
	raw = ExtractRawConstraints(normal)
	if raw != "" {
		t.Fatalf("expected empty string for non-corrupted, got %q", raw)
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
