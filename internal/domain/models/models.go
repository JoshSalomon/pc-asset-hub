package models

import "time"

type EntityType struct {
	ID        string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type EntityTypeVersion struct {
	ID           string
	EntityTypeID string
	Version      int
	Description  string
	CreatedAt    time.Time
}

// BaseType represents the fundamental data type of a type definition.
type BaseType string

const (
	BaseTypeString  BaseType = "string"
	BaseTypeInteger BaseType = "integer"
	BaseTypeNumber  BaseType = "number"
	BaseTypeBoolean BaseType = "boolean"
	BaseTypeDate    BaseType = "date"
	BaseTypeURL     BaseType = "url"
	BaseTypeEnum    BaseType = "enum"
	BaseTypeList    BaseType = "list"
	BaseTypeJSON    BaseType = "json"
)

// ValidBaseTypes is the set of allowed base types.
var ValidBaseTypes = map[BaseType]bool{
	BaseTypeString: true, BaseTypeInteger: true, BaseTypeNumber: true,
	BaseTypeBoolean: true, BaseTypeDate: true, BaseTypeURL: true,
	BaseTypeEnum: true, BaseTypeList: true, BaseTypeJSON: true,
}

// TypeDefinition is a reusable, versioned type definition (replaces Enum).
type TypeDefinition struct {
	ID          string
	Name        string
	Description string
	BaseType    BaseType
	System      bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// TypeDefinitionVersion is a versioned snapshot of a type definition's constraints.
type TypeDefinitionVersion struct {
	ID               string
	TypeDefinitionID string
	VersionNumber    int
	Constraints      map[string]any
	CreatedAt        time.Time
}

// IsCorruptedConstraints returns true if the constraints map contains a _raw key,
// indicating that the original JSON in the database was malformed.
func IsCorruptedConstraints(constraints map[string]any) bool {
	_, ok := constraints["_raw"]
	return ok
}

// ExtractRawConstraints returns the original raw string from corrupted constraints,
// or an empty string if constraints are not corrupted.
func ExtractRawConstraints(constraints map[string]any) string {
	raw, ok := constraints["_raw"]
	if !ok {
		return ""
	}
	s, _ := raw.(string)
	return s
}

// CatalogVersionTypePin pins a type definition version to a catalog version.
type CatalogVersionTypePin struct {
	ID                      string
	CatalogVersionID        string
	TypeDefinitionVersionID string
}

type Attribute struct {
	ID                      string
	EntityTypeVersionID     string
	Name                    string
	Description             string
	TypeDefinitionVersionID string
	Ordinal                 int
	Required                bool
}

type AssociationType string

const (
	AssociationTypeContainment   AssociationType = "containment"
	AssociationTypeDirectional   AssociationType = "directional"
	AssociationTypeBidirectional AssociationType = "bidirectional"
)

type Association struct {
	ID                  string
	EntityTypeVersionID string
	Name                string
	TargetEntityTypeID  string
	Type                AssociationType
	SourceRole          string
	TargetRole          string
	SourceCardinality   string
	TargetCardinality   string
	CreatedAt           time.Time
}


type LifecycleStage string

const (
	LifecycleStageDevelopment LifecycleStage = "development"
	LifecycleStageTesting     LifecycleStage = "testing"
	LifecycleStageProduction  LifecycleStage = "production"
)

type CatalogVersion struct {
	ID             string
	VersionLabel   string
	Description    string
	LifecycleStage LifecycleStage
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type CatalogVersionPin struct {
	ID                  string
	CatalogVersionID    string
	EntityTypeVersionID string
}

type LifecycleTransition struct {
	ID               string
	CatalogVersionID string
	FromStage        string
	ToStage          string
	PerformedBy      string
	PerformedAt      time.Time
	Notes            string
}

type ValidationStatus string

const (
	ValidationStatusDraft   ValidationStatus = "draft"
	ValidationStatusValid   ValidationStatus = "valid"
	ValidationStatusInvalid ValidationStatus = "invalid"
)

type Catalog struct {
	ID               string
	Name             string
	Description      string
	CatalogVersionID string
	ValidationStatus ValidationStatus
	Published        bool
	PublishedAt      *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type EntityInstance struct {
	ID               string
	EntityTypeID     string
	CatalogID        string
	ParentInstanceID string // empty if top-level
	Name             string
	Description      string
	Version          int
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        *time.Time
}

type InstanceAttributeValue struct {
	ID              string
	InstanceID      string
	InstanceVersion int
	AttributeID     string
	ValueString     string
	ValueNumber     *float64
	ValueJSON       string
}

type AssociationLink struct {
	ID               string
	AssociationID    string
	SourceInstanceID string
	TargetInstanceID string
	CreatedAt        time.Time
}

// === System Attributes ===

const (
	SystemAttrName        = "name"
	SystemAttrDescription = "description"
	SystemAttrNameOrdinal = -2
	SystemAttrDescOrdinal = -1
)

// IsSystemAttributeName returns true if the given name is reserved for a system attribute.
func IsSystemAttributeName(name string) bool {
	return name == SystemAttrName || name == SystemAttrDescription
}

// ListParams holds common parameters for list operations.
type ListParams struct {
	Offset   int
	Limit    int
	SortBy   string
	SortDesc bool
	Filters  map[string]string
}
