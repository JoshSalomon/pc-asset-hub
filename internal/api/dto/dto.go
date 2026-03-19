package dto

import "time"

// === Entity Type DTOs ===

type CreateEntityTypeRequest struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
}

type EntityTypeResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type EntityTypeVersionResponse struct {
	ID           string    `json:"id"`
	EntityTypeID string    `json:"entity_type_id"`
	Version      int       `json:"version"`
	Description  string    `json:"description"`
	CreatedAt    time.Time `json:"created_at"`
}

type UpdateEntityTypeRequest struct {
	Description string `json:"description"`
}

type CopyEntityTypeRequest struct {
	SourceVersion int    `json:"source_version"`
	NewName       string `json:"new_name" validate:"required"`
}

// === Attribute DTOs ===

type CreateAttributeRequest struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
	Type        string `json:"type" validate:"required"`
	EnumID      string `json:"enum_id"`
	Required    bool   `json:"required"`
}

type CopyAttributesRequest struct {
	SourceEntityTypeID string   `json:"source_entity_type_id" validate:"required"`
	SourceVersion      int      `json:"source_version" validate:"required"`
	AttributeNames     []string `json:"attribute_names" validate:"required"`
}

type ReorderAttributesRequest struct {
	OrderedIDs []string `json:"ordered_ids" validate:"required"`
}

type AttributeResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	EnumID      string `json:"enum_id,omitempty"`
	Ordinal     int    `json:"ordinal"`
	Required    bool   `json:"required"`
	System      bool   `json:"system"`
}

type UpdateAttributeRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Type        *string `json:"type"`
	EnumID      *string `json:"enum_id"`
	Required    *bool   `json:"required"`
}

type RenameEntityTypeRequest struct {
	Name            string `json:"name" validate:"required"`
	DeepCopyAllowed bool   `json:"deep_copy_allowed"`
}

type RenameEntityTypeResponse struct {
	EntityType  EntityTypeResponse `json:"entity_type"`
	WasDeepCopy bool               `json:"was_deep_copy"`
}

// === Association DTOs ===

type CreateAssociationRequest struct {
	TargetEntityTypeID string `json:"target_entity_type_id" validate:"required"`
	Type               string `json:"type" validate:"required"`
	Name               string `json:"name" validate:"required"`
	SourceRole         string `json:"source_role"`
	TargetRole         string `json:"target_role"`
	SourceCardinality  string `json:"source_cardinality"`
	TargetCardinality  string `json:"target_cardinality"`
}

type AssociationResponse struct {
	ID                  string    `json:"id"`
	EntityTypeVersionID string    `json:"entity_type_version_id"`
	Name                string    `json:"name"`
	TargetEntityTypeID  string    `json:"target_entity_type_id"`
	Type                string    `json:"type"`
	SourceRole          string    `json:"source_role"`
	TargetRole          string    `json:"target_role"`
	SourceCardinality   string    `json:"source_cardinality"`
	TargetCardinality   string    `json:"target_cardinality"`
	CreatedAt           time.Time `json:"created_at"`
	// Direction indicates the perspective: "outgoing" (this entity owns the association)
	// or "incoming" (this entity is the target of another entity's association).
	Direction           string    `json:"direction"`
	// SourceEntityTypeID is set for incoming associations to identify the other side.
	SourceEntityTypeID  string    `json:"source_entity_type_id,omitempty"`
}

type UpdateAssociationRequest struct {
	Name              *string `json:"name"`
	Type              *string `json:"type"`
	SourceRole        *string `json:"source_role"`
	TargetRole        *string `json:"target_role"`
	SourceCardinality *string `json:"source_cardinality"`
	TargetCardinality *string `json:"target_cardinality"`
}

// === Enum DTOs ===

type CreateEnumRequest struct {
	Name   string   `json:"name" validate:"required"`
	Values []string `json:"values"`
}

type UpdateEnumRequest struct {
	Name string `json:"name" validate:"required"`
}

type EnumResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type EnumValueResponse struct {
	ID      string `json:"id"`
	Value   string `json:"value"`
	Ordinal int    `json:"ordinal"`
}

type AddEnumValueRequest struct {
	Value string `json:"value" validate:"required"`
}

type ReorderEnumValuesRequest struct {
	OrderedIDs []string `json:"ordered_ids" validate:"required"`
}

// === Catalog Version DTOs ===

type CreateCatalogVersionRequest struct {
	VersionLabel string                 `json:"version_label" validate:"required"`
	Pins         []CatalogVersionPinDTO `json:"pins"`
}

type CatalogVersionPinDTO struct {
	EntityTypeVersionID string `json:"entity_type_version_id"`
}

type CatalogVersionResponse struct {
	ID             string    `json:"id"`
	VersionLabel   string    `json:"version_label"`
	LifecycleStage string    `json:"lifecycle_stage"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type CatalogVersionPinResponse struct {
	EntityTypeName      string `json:"entity_type_name"`
	EntityTypeID        string `json:"entity_type_id"`
	EntityTypeVersionID string `json:"entity_type_version_id"`
	Version             int    `json:"version"`
	Description         string `json:"description,omitempty"`
}

type LifecycleTransitionResponse struct {
	ID          string    `json:"id"`
	FromStage   string    `json:"from_stage"`
	ToStage     string    `json:"to_stage"`
	PerformedBy string    `json:"performed_by"`
	PerformedAt time.Time `json:"performed_at"`
	Notes       string    `json:"notes,omitempty"`
}

// === Version History DTOs ===

type VersionDiffResponse struct {
	FromVersion int                  `json:"from_version"`
	ToVersion   int                  `json:"to_version"`
	Changes     []VersionDiffItemDTO `json:"changes"`
}

type VersionDiffItemDTO struct {
	Name       string `json:"name"`
	ChangeType string `json:"change_type"`
	Category   string `json:"category"`
	OldValue   string `json:"old_value,omitempty"`
	NewValue   string `json:"new_value,omitempty"`
}

// === Version Snapshot DTOs ===

type SnapshotAttributeResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	EnumID      string `json:"enum_id,omitempty"`
	EnumName    string `json:"enum_name,omitempty"`
	Ordinal     int    `json:"ordinal"`
	Required    bool   `json:"required"`
	System      bool   `json:"system"`
}

type SnapshotAssociationResponse struct {
	ID                   string `json:"id"`
	Name                 string `json:"name"`
	Type                 string `json:"type"`
	TargetEntityTypeID   string `json:"target_entity_type_id"`
	TargetEntityTypeName string `json:"target_entity_type_name"`
	SourceRole           string `json:"source_role"`
	TargetRole           string `json:"target_role"`
	SourceCardinality    string `json:"source_cardinality"`
	TargetCardinality    string `json:"target_cardinality"`
	Direction            string `json:"direction"`
	SourceEntityTypeID   string `json:"source_entity_type_id,omitempty"`
	SourceEntityTypeName string `json:"source_entity_type_name,omitempty"`
}

type VersionSnapshotResponse struct {
	EntityType   EntityTypeResponse            `json:"entity_type"`
	Version      EntityTypeVersionResponse     `json:"version"`
	Attributes   []SnapshotAttributeResponse   `json:"attributes"`
	Associations []SnapshotAssociationResponse  `json:"associations"`
}

// === Containment Tree DTOs ===

type ContainmentTreeNodeDTO struct {
	EntityType    EntityTypeResponse        `json:"entity_type"`
	Versions      []EntityTypeVersionResponse `json:"versions"`
	LatestVersion int                        `json:"latest_version"`
	Children      []ContainmentTreeNodeDTO   `json:"children"`
}

// === Catalog DTOs ===

type CreateCatalogRequest struct {
	Name             string `json:"name"`
	Description      string `json:"description"`
	CatalogVersionID string `json:"catalog_version_id"`
}

type CatalogResponse struct {
	ID                  string     `json:"id"`
	Name                string     `json:"name"`
	Description         string     `json:"description"`
	CatalogVersionID    string     `json:"catalog_version_id"`
	CatalogVersionLabel string     `json:"catalog_version_label,omitempty"`
	ValidationStatus    string     `json:"validation_status"`
	Published           bool       `json:"published"`
	PublishedAt         *time.Time `json:"published_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

type StatusResponse struct {
	Status string `json:"status"`
}

type CopyCatalogRequest struct {
	Source      string `json:"source"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ReplaceCatalogRequest struct {
	Source      string `json:"source"`
	Target      string `json:"target"`
	ArchiveName string `json:"archive_name"`
}

type CatalogWarningResponse struct {
	CatalogName      string `json:"catalog_name"`
	ValidationStatus string `json:"validation_status"`
}

type PromoteResponse struct {
	Status   string                   `json:"status"`
	Warnings []CatalogWarningResponse `json:"warnings"`
}

// === Validation DTOs ===

type ValidationErrorResponse struct {
	EntityType   string `json:"entity_type"`
	InstanceName string `json:"instance_name"`
	Field        string `json:"field"`
	Violation    string `json:"violation"`
}

type ValidationResultResponse struct {
	Status string                    `json:"status"`
	Errors []ValidationErrorResponse `json:"errors"`
}

// === Instance DTOs ===

type CreateInstanceRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Attributes  map[string]any `json:"attributes"`
}

type UpdateInstanceRequest struct {
	Version     int                    `json:"version"`
	Name        *string                `json:"name,omitempty"`
	Description *string                `json:"description,omitempty"`
	Attributes  map[string]any `json:"attributes,omitempty"`
}

type InstanceResponse struct {
	ID               string                     `json:"id"`
	EntityTypeID     string                     `json:"entity_type_id"`
	CatalogID        string                     `json:"catalog_id"`
	ParentInstanceID string                     `json:"parent_instance_id,omitempty"`
	Name             string                     `json:"name"`
	Description      string                     `json:"description"`
	Version          int                        `json:"version"`
	Attributes       []AttributeValueResponse   `json:"attributes"`
	ParentChain      []ParentChainEntryResponse `json:"parent_chain,omitempty"`
	CreatedAt        time.Time                  `json:"created_at"`
	UpdatedAt        time.Time                  `json:"updated_at"`
}

type AttributeValueResponse struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Value    any    `json:"value"`
	System   bool   `json:"system"`
	Required bool   `json:"required"`
}

// === Association Link DTOs ===

type CreateAssociationLinkRequest struct {
	TargetInstanceID string `json:"target_instance_id" validate:"required"`
	AssociationName  string `json:"association_name" validate:"required"`
}

type AssociationLinkResponse struct {
	ID               string    `json:"id"`
	AssociationID    string    `json:"association_id"`
	SourceInstanceID string    `json:"source_instance_id"`
	TargetInstanceID string    `json:"target_instance_id"`
	CreatedAt        time.Time `json:"created_at"`
}

type ReferenceResponse struct {
	LinkID          string `json:"link_id"`
	AssociationName string `json:"association_name"`
	AssociationType string `json:"association_type"`
	InstanceID      string `json:"instance_id"`
	InstanceName    string `json:"instance_name"`
	EntityTypeName  string `json:"entity_type_name"`
}

// === Set Parent DTO ===

type SetParentRequest struct {
	ParentType       string `json:"parent_type"`
	ParentInstanceID string `json:"parent_instance_id"`
}

// === Catalog Data Viewer DTOs ===

type TreeNodeResponse struct {
	InstanceID     string             `json:"instance_id"`
	InstanceName   string             `json:"instance_name"`
	EntityTypeName string             `json:"entity_type_name"`
	Description    string             `json:"description"`
	Children       []TreeNodeResponse `json:"children"`
}

type ParentChainEntryResponse struct {
	InstanceID     string `json:"instance_id"`
	InstanceName   string `json:"instance_name"`
	EntityTypeName string `json:"entity_type_name"`
}

// === List Response ===

type ListResponse struct {
	Items any `json:"items"`
	Total int         `json:"total"`
}

// === Error Response ===

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
