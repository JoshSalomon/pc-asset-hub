package export

import "context"

const (
	BindingStatusNever   = "never"
	BindingStatusSuccess = "success"
	BindingStatusFailed  = "failed"
)

type Exporter interface {
	Name() string
	Description() string
	ParameterSchema() []ParameterDef
	ValidateSchema(params map[string]string, schema SchemaInfo) error
	Export(ctx context.Context, input ExportInput) (*ExportOutput, error)
}

type ParameterDef struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Default     string `json:"default,omitempty"`
}

type SchemaInfo struct {
	EntityTypes []SchemaEntityType
}

type SchemaEntityType struct {
	Name         string
	Attributes   []string
	Associations []SchemaAssociation
}

type SchemaAssociation struct {
	Name             string
	Type             string
	TargetEntityType string
}

type ExportInput struct {
	CatalogName              string
	CatalogDesc              string
	CVLabel                  string
	Parameters               map[string]string
	EntityTypes              []ExportEntityType
	InstancesByType          map[string][]*ExportInstance
	ChildrenOf               map[string][]*ExportInstance
	VirtualServerInstanceName string
	AllowedToolIDs           map[string]bool
}

type ExportEntityType struct {
	Name         string
	Attributes   []string
	Associations []SchemaAssociation
}

type ExportInstance struct {
	ID          string
	EntityType  string
	Name        string
	Description string
	ParentID    string
	Attributes  map[string]any
	LinksByAssoc map[string][]ExportLink
}

type ExportLink struct {
	TargetInstanceID   string
	TargetInstanceName string
	TargetEntityType   string
}

type ExportOutput struct {
	Artifacts []K8sArtifact
	Warnings  []string
}

type K8sArtifact struct {
	APIVersion string
	Kind       string
	Name       string
	Namespace  string
	YAML       string
}
