package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"

	domain "github.com/project-catalyst/pc-asset-hub/internal/domain/models"
)

type EntityType struct {
	ID        string `gorm:"primaryKey;size:36"`
	Name      string `gorm:"uniqueIndex;not null;size:255"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Versions  []EntityTypeVersion `gorm:"foreignKey:EntityTypeID;constraint:OnDelete:CASCADE"`
}

func (e *EntityType) ToModel() *domain.EntityType {
	return &domain.EntityType{
		ID:        e.ID,
		Name:      e.Name,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}

func EntityTypeFromModel(m *domain.EntityType) *EntityType {
	return &EntityType{
		ID:        m.ID,
		Name:      m.Name,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

type EntityTypeVersion struct {
	ID           string `gorm:"primaryKey;size:36"`
	EntityTypeID string `gorm:"not null;size:36;uniqueIndex:idx_etv_type_version"`
	Version      int    `gorm:"not null;uniqueIndex:idx_etv_type_version"`
	Description  string `gorm:"size:1024"`
	CreatedAt    time.Time
	Attributes   []Attribute   `gorm:"foreignKey:EntityTypeVersionID;constraint:OnDelete:CASCADE"`
	Associations []Association `gorm:"foreignKey:EntityTypeVersionID;constraint:OnDelete:CASCADE"`
}

func (e *EntityTypeVersion) ToModel() *domain.EntityTypeVersion {
	return &domain.EntityTypeVersion{
		ID:           e.ID,
		EntityTypeID: e.EntityTypeID,
		Version:      e.Version,
		Description:  e.Description,
		CreatedAt:    e.CreatedAt,
	}
}

func EntityTypeVersionFromModel(m *domain.EntityTypeVersion) *EntityTypeVersion {
	return &EntityTypeVersion{
		ID:           m.ID,
		EntityTypeID: m.EntityTypeID,
		Version:      m.Version,
		Description:  m.Description,
		CreatedAt:    m.CreatedAt,
	}
}

type Attribute struct {
	ID                      string `gorm:"primaryKey;size:36"`
	EntityTypeVersionID     string `gorm:"not null;size:36;uniqueIndex:idx_attr_version_name"`
	Name                    string `gorm:"not null;size:255;uniqueIndex:idx_attr_version_name"`
	Description             string `gorm:"size:1024"`
	TypeDefinitionVersionID string `gorm:"not null;size:36"`
	Ordinal                 int    `gorm:"not null"`
	Required                bool   `gorm:"not null;default:false"`
}

func (a *Attribute) ToModel() *domain.Attribute {
	return &domain.Attribute{
		ID:                      a.ID,
		EntityTypeVersionID:     a.EntityTypeVersionID,
		Name:                    a.Name,
		Description:             a.Description,
		TypeDefinitionVersionID: a.TypeDefinitionVersionID,
		Ordinal:                 a.Ordinal,
		Required:                a.Required,
	}
}

func AttributeFromModel(m *domain.Attribute) *Attribute {
	return &Attribute{
		ID:                      m.ID,
		EntityTypeVersionID:     m.EntityTypeVersionID,
		Name:                    m.Name,
		Description:             m.Description,
		TypeDefinitionVersionID: m.TypeDefinitionVersionID,
		Ordinal:                 m.Ordinal,
		Required:                m.Required,
	}
}

type Association struct {
	ID                  string `gorm:"primaryKey;size:36"`
	EntityTypeVersionID string `gorm:"not null;size:36;uniqueIndex:idx_assoc_version_name"`
	Name                string `gorm:"not null;size:255;uniqueIndex:idx_assoc_version_name"`
	TargetEntityTypeID  string `gorm:"not null;size:36"`
	Type                string `gorm:"not null;size:20"` // containment, directional, bidirectional
	SourceRole          string `gorm:"size:255"`
	TargetRole          string `gorm:"size:255"`
	SourceCardinality   string `gorm:"size:20;default:'0..n'"`
	TargetCardinality   string `gorm:"size:20;default:'0..n'"`
	CreatedAt           time.Time
}

func (a *Association) ToModel() *domain.Association {
	return &domain.Association{
		ID:                  a.ID,
		EntityTypeVersionID: a.EntityTypeVersionID,
		Name:                a.Name,
		TargetEntityTypeID:  a.TargetEntityTypeID,
		Type:                domain.AssociationType(a.Type),
		SourceRole:          a.SourceRole,
		TargetRole:          a.TargetRole,
		SourceCardinality:   a.SourceCardinality,
		TargetCardinality:   a.TargetCardinality,
		CreatedAt:           a.CreatedAt,
	}
}

func AssociationFromModel(m *domain.Association) *Association {
	return &Association{
		ID:                  m.ID,
		EntityTypeVersionID: m.EntityTypeVersionID,
		Name:                m.Name,
		TargetEntityTypeID:  m.TargetEntityTypeID,
		Type:                string(m.Type),
		SourceRole:          m.SourceRole,
		TargetRole:          m.TargetRole,
		SourceCardinality:   m.SourceCardinality,
		TargetCardinality:   m.TargetCardinality,
		CreatedAt:           m.CreatedAt,
	}
}

type TypeDefinition struct {
	ID          string `gorm:"primaryKey;size:36"`
	Name        string `gorm:"uniqueIndex;not null;size:255"`
	Description string `gorm:"size:1024"`
	BaseType    string `gorm:"not null;size:20"`
	System      bool   `gorm:"not null;default:false"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Versions    []TypeDefinitionVersion `gorm:"foreignKey:TypeDefinitionID;constraint:OnDelete:CASCADE"`
}

func (t *TypeDefinition) ToModel() *domain.TypeDefinition {
	return &domain.TypeDefinition{
		ID:          t.ID,
		Name:        t.Name,
		Description: t.Description,
		BaseType:    domain.BaseType(t.BaseType),
		System:      t.System,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

func TypeDefinitionFromModel(m *domain.TypeDefinition) *TypeDefinition {
	return &TypeDefinition{
		ID:          m.ID,
		Name:        m.Name,
		Description: m.Description,
		BaseType:    string(m.BaseType),
		System:      m.System,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

type TypeDefinitionVersion struct {
	ID               string `gorm:"primaryKey;size:36"`
	TypeDefinitionID string `gorm:"not null;size:36;uniqueIndex:idx_tdv_type_version"`
	VersionNumber    int    `gorm:"not null;uniqueIndex:idx_tdv_type_version"`
	Constraints      string `gorm:"type:text"`
	CreatedAt        time.Time
}

func (t *TypeDefinitionVersion) ToModel() *domain.TypeDefinitionVersion {
	var constraints map[string]any
	if t.Constraints != "" && t.Constraints != "{}" {
		if err := json.Unmarshal([]byte(t.Constraints), &constraints); err != nil {
			// Corrupted JSON: preserve the raw string wrapped in a map so it's
			// visible during validation rather than silently lost.
			constraints = map[string]any{"_raw": t.Constraints}
		}
	}
	if constraints == nil {
		constraints = map[string]any{}
	}
	return &domain.TypeDefinitionVersion{
		ID:               t.ID,
		TypeDefinitionID: t.TypeDefinitionID,
		VersionNumber:    t.VersionNumber,
		Constraints:      constraints,
		CreatedAt:        t.CreatedAt,
	}
}

func TypeDefinitionVersionFromModel(m *domain.TypeDefinitionVersion) *TypeDefinitionVersion {
	constraintsJSON := "{}"
	if len(m.Constraints) > 0 {
		b, err := json.Marshal(m.Constraints)
		if err == nil {
			constraintsJSON = string(b)
		}
	}
	return &TypeDefinitionVersion{
		ID:               m.ID,
		TypeDefinitionID: m.TypeDefinitionID,
		VersionNumber:    m.VersionNumber,
		Constraints:      constraintsJSON,
		CreatedAt:        m.CreatedAt,
	}
}

type CatalogVersionTypePin struct {
	ID                      string `gorm:"primaryKey;size:36"`
	CatalogVersionID        string `gorm:"not null;size:36;uniqueIndex:idx_cvtp_unique"`
	TypeDefinitionVersionID string `gorm:"not null;size:36;uniqueIndex:idx_cvtp_unique"`
}

func (c *CatalogVersionTypePin) ToModel() *domain.CatalogVersionTypePin {
	return &domain.CatalogVersionTypePin{
		ID:                      c.ID,
		CatalogVersionID:        c.CatalogVersionID,
		TypeDefinitionVersionID: c.TypeDefinitionVersionID,
	}
}

func CatalogVersionTypePinFromModel(m *domain.CatalogVersionTypePin) *CatalogVersionTypePin {
	return &CatalogVersionTypePin{
		ID:                      m.ID,
		CatalogVersionID:        m.CatalogVersionID,
		TypeDefinitionVersionID: m.TypeDefinitionVersionID,
	}
}

type CatalogVersion struct {
	ID             string `gorm:"primaryKey;size:36"`
	VersionLabel   string `gorm:"uniqueIndex;not null;size:255"`
	Description    string `gorm:"size:1024"`
	LifecycleStage string `gorm:"not null;size:20;default:development"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Pins           []CatalogVersionPin   `gorm:"foreignKey:CatalogVersionID;constraint:OnDelete:CASCADE"`
	Transitions    []LifecycleTransition `gorm:"foreignKey:CatalogVersionID;constraint:OnDelete:CASCADE"`
}

func (c *CatalogVersion) ToModel() *domain.CatalogVersion {
	return &domain.CatalogVersion{
		ID:             c.ID,
		VersionLabel:   c.VersionLabel,
		Description:    c.Description,
		LifecycleStage: domain.LifecycleStage(c.LifecycleStage),
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
	}
}

func CatalogVersionFromModel(m *domain.CatalogVersion) *CatalogVersion {
	return &CatalogVersion{
		ID:             m.ID,
		VersionLabel:   m.VersionLabel,
		Description:    m.Description,
		LifecycleStage: string(m.LifecycleStage),
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}

type CatalogVersionPin struct {
	ID                  string `gorm:"primaryKey;size:36"`
	CatalogVersionID    string `gorm:"not null;size:36;uniqueIndex:idx_cv_pin"`
	EntityTypeVersionID string `gorm:"not null;size:36;uniqueIndex:idx_cv_pin"`
}

func (c *CatalogVersionPin) ToModel() *domain.CatalogVersionPin {
	return &domain.CatalogVersionPin{
		ID:                  c.ID,
		CatalogVersionID:    c.CatalogVersionID,
		EntityTypeVersionID: c.EntityTypeVersionID,
	}
}

func CatalogVersionPinFromModel(m *domain.CatalogVersionPin) *CatalogVersionPin {
	return &CatalogVersionPin{
		ID:                  m.ID,
		CatalogVersionID:    m.CatalogVersionID,
		EntityTypeVersionID: m.EntityTypeVersionID,
	}
}

type LifecycleTransition struct {
	ID               string `gorm:"primaryKey;size:36"`
	CatalogVersionID string `gorm:"not null;size:36"`
	FromStage        string `gorm:"size:20"`
	ToStage          string `gorm:"not null;size:20"`
	PerformedBy      string `gorm:"not null;size:255"`
	PerformedAt      time.Time
	Notes            string `gorm:"size:1024"`
}

func (l *LifecycleTransition) ToModel() *domain.LifecycleTransition {
	return &domain.LifecycleTransition{
		ID:               l.ID,
		CatalogVersionID: l.CatalogVersionID,
		FromStage:        l.FromStage,
		ToStage:          l.ToStage,
		PerformedBy:      l.PerformedBy,
		PerformedAt:      l.PerformedAt,
		Notes:            l.Notes,
	}
}

func LifecycleTransitionFromModel(m *domain.LifecycleTransition) *LifecycleTransition {
	return &LifecycleTransition{
		ID:               m.ID,
		CatalogVersionID: m.CatalogVersionID,
		FromStage:        m.FromStage,
		ToStage:          m.ToStage,
		PerformedBy:      m.PerformedBy,
		PerformedAt:      m.PerformedAt,
		Notes:            m.Notes,
	}
}

// === Data Table Models ===

type Catalog struct {
	ID               string     `gorm:"primaryKey;size:36"`
	Name             string     `gorm:"uniqueIndex;not null;size:63"`
	Description      string     `gorm:"size:1024"`
	CatalogVersionID string     `gorm:"not null;size:36"`
	ValidationStatus string     `gorm:"not null;size:20;default:draft"`
	Published        bool       `gorm:"not null;default:false"`
	PublishedAt      *time.Time `gorm:"default:null"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (c *Catalog) ToModel() *domain.Catalog {
	return &domain.Catalog{
		ID:               c.ID,
		Name:             c.Name,
		Description:      c.Description,
		CatalogVersionID: c.CatalogVersionID,
		ValidationStatus: domain.ValidationStatus(c.ValidationStatus),
		Published:        c.Published,
		PublishedAt:      c.PublishedAt,
		CreatedAt:        c.CreatedAt,
		UpdatedAt:        c.UpdatedAt,
	}
}

func CatalogFromModel(m *domain.Catalog) *Catalog {
	return &Catalog{
		ID:               m.ID,
		Name:             m.Name,
		Description:      m.Description,
		CatalogVersionID: m.CatalogVersionID,
		ValidationStatus: string(m.ValidationStatus),
		Published:        m.Published,
		PublishedAt:      m.PublishedAt,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}
}

type EntityInstance struct {
	ID               string `gorm:"primaryKey;size:36"`
	EntityTypeID     string `gorm:"not null;size:36;uniqueIndex:idx_instance_scope"`
	CatalogID string `gorm:"column:catalog_id;not null;size:36;uniqueIndex:idx_instance_scope"`
	ParentInstanceID string `gorm:"size:36;uniqueIndex:idx_instance_scope;default:''"`
	Name             string `gorm:"not null;size:255;uniqueIndex:idx_instance_scope"`
	Description      string `gorm:"size:1024"`
	Version          int    `gorm:"not null;default:1"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (e *EntityInstance) ToModel() *domain.EntityInstance {
	return &domain.EntityInstance{
		ID:               e.ID,
		EntityTypeID:     e.EntityTypeID,
		CatalogID:        e.CatalogID,
		ParentInstanceID: e.ParentInstanceID,
		Name:             e.Name,
		Description:      e.Description,
		Version:          e.Version,
		CreatedAt:        e.CreatedAt,
		UpdatedAt:        e.UpdatedAt,
	}
}

func EntityInstanceFromModel(m *domain.EntityInstance) *EntityInstance {
	return &EntityInstance{
		ID:               m.ID,
		EntityTypeID:     m.EntityTypeID,
		CatalogID:        m.CatalogID,
		ParentInstanceID: m.ParentInstanceID,
		Name:             m.Name,
		Description:      m.Description,
		Version:          m.Version,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}
}

type InstanceAttributeValue struct {
	ID              string `gorm:"primaryKey;size:36"`
	InstanceID      string `gorm:"not null;size:36;uniqueIndex:idx_iav_unique"`
	InstanceVersion int    `gorm:"not null;uniqueIndex:idx_iav_unique"`
	AttributeID     string `gorm:"not null;size:36;uniqueIndex:idx_iav_unique"`
	ValueString     string `gorm:"size:4096"`
	ValueNumber     *float64
	ValueJSON       string `gorm:"type:text"`
}

func (i *InstanceAttributeValue) ToModel() *domain.InstanceAttributeValue {
	return &domain.InstanceAttributeValue{
		ID:              i.ID,
		InstanceID:      i.InstanceID,
		InstanceVersion: i.InstanceVersion,
		AttributeID:     i.AttributeID,
		ValueString:     i.ValueString,
		ValueNumber:     i.ValueNumber,
		ValueJSON:       i.ValueJSON,
	}
}

func InstanceAttributeValueFromModel(m *domain.InstanceAttributeValue) *InstanceAttributeValue {
	return &InstanceAttributeValue{
		ID:              m.ID,
		InstanceID:      m.InstanceID,
		InstanceVersion: m.InstanceVersion,
		AttributeID:     m.AttributeID,
		ValueString:     m.ValueString,
		ValueNumber:     m.ValueNumber,
		ValueJSON:       m.ValueJSON,
	}
}

type AssociationLink struct {
	ID               string `gorm:"primaryKey;size:36"`
	AssociationID    string `gorm:"not null;size:36"`
	SourceInstanceID string `gorm:"not null;size:36;index:idx_assoc_link_source"`
	TargetInstanceID string `gorm:"not null;size:36;index:idx_assoc_link_target"`
	CreatedAt        time.Time
}

func (a *AssociationLink) ToModel() *domain.AssociationLink {
	return &domain.AssociationLink{
		ID:               a.ID,
		AssociationID:    a.AssociationID,
		SourceInstanceID: a.SourceInstanceID,
		TargetInstanceID: a.TargetInstanceID,
		CreatedAt:        a.CreatedAt,
	}
}

func AssociationLinkFromModel(m *domain.AssociationLink) *AssociationLink {
	return &AssociationLink{
		ID:               m.ID,
		AssociationID:    m.AssociationID,
		SourceInstanceID: m.SourceInstanceID,
		TargetInstanceID: m.TargetInstanceID,
		CreatedAt:        m.CreatedAt,
	}
}

// AllModels returns all GORM model structs for auto-migration.
func AllModels() []any {
	return []any{
		&EntityType{},
		&EntityTypeVersion{},
		&Attribute{},
		&Association{},
		&TypeDefinition{},
		&TypeDefinitionVersion{},
		&CatalogVersion{},
		&CatalogVersionPin{},
		&CatalogVersionTypePin{},
		&LifecycleTransition{},
		&Catalog{},
		&EntityInstance{},
		&InstanceAttributeValue{},
		&AssociationLink{},
	}
}

// InitDB initializes the database with auto-migration and data fixups.
func InitDB(db *gorm.DB) error {
	// Drop legacy tables that have been replaced by the type system
	for _, table := range []string{"enum_values", "enums"} {
		if db.Migrator().HasTable(table) {
			if err := db.Migrator().DropTable(table); err != nil {
				return err
			}
		}
	}

	// Drop legacy columns from attributes table (replaced by type_definition_version_id)
	if db.Migrator().HasTable("attributes") {
		for _, col := range []string{"type", "enum_id"} {
			if db.Migrator().HasColumn(&Attribute{}, col) {
				if err := db.Migrator().DropColumn(&Attribute{}, col); err != nil {
					return err
				}
			}
		}
	}

	// Drop legacy column from instance_attribute_values table (replaced by value_json)
	if db.Migrator().HasTable("instance_attribute_values") {
		if db.Migrator().HasColumn(&InstanceAttributeValue{}, "value_enum") {
			if err := db.Migrator().DropColumn(&InstanceAttributeValue{}, "value_enum"); err != nil {
				return err
			}
		}
	}

	if err := db.AutoMigrate(AllModels()...); err != nil {
		return err
	}
	// Fix containment associations: source cardinality should be "0..1", not empty or "0..n"
	return db.Model(&Association{}).
		Where("type = ? AND (source_cardinality IS NULL OR source_cardinality = '' OR source_cardinality = '0..n')", "containment").
		Update("source_cardinality", "0..1").Error
}
