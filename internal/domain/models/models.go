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

type AttributeType string

const (
	AttributeTypeString AttributeType = "string"
	AttributeTypeNumber AttributeType = "number"
	AttributeTypeEnum   AttributeType = "enum"
)

type Attribute struct {
	ID                  string
	EntityTypeVersionID string
	Name                string
	Description         string
	Type                AttributeType
	EnumID              string // empty if not enum type
	Ordinal             int
	Required            bool
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
	TargetEntityTypeID  string
	Type                AssociationType
	SourceRole          string
	TargetRole          string
	CreatedAt           time.Time
}

type Enum struct {
	ID        string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type EnumValue struct {
	ID      string
	EnumID  string
	Value   string
	Ordinal int
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

type EntityInstance struct {
	ID               string
	EntityTypeID     string
	CatalogVersionID string
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
	ValueEnum       string
}

type AssociationLink struct {
	ID               string
	AssociationID    string
	SourceInstanceID string
	TargetInstanceID string
	CreatedAt        time.Time
}

// ListParams holds common parameters for list operations.
type ListParams struct {
	Offset   int
	Limit    int
	SortBy   string
	SortDesc bool
	Filters  map[string]string
}
