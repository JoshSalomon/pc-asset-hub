package models

import (
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
	ID                  string `gorm:"primaryKey;size:36"`
	EntityTypeVersionID string `gorm:"not null;size:36;uniqueIndex:idx_attr_version_name"`
	Name                string `gorm:"not null;size:255;uniqueIndex:idx_attr_version_name"`
	Description         string `gorm:"size:1024"`
	Type                string `gorm:"not null;size:20"` // string, number, enum
	EnumID              string `gorm:"size:36"`
	Ordinal             int    `gorm:"not null"`
	Required            bool   `gorm:"not null;default:false"`
}

func (a *Attribute) ToModel() *domain.Attribute {
	return &domain.Attribute{
		ID:                  a.ID,
		EntityTypeVersionID: a.EntityTypeVersionID,
		Name:                a.Name,
		Description:         a.Description,
		Type:                domain.AttributeType(a.Type),
		EnumID:              a.EnumID,
		Ordinal:             a.Ordinal,
		Required:            a.Required,
	}
}

func AttributeFromModel(m *domain.Attribute) *Attribute {
	return &Attribute{
		ID:                  m.ID,
		EntityTypeVersionID: m.EntityTypeVersionID,
		Name:                m.Name,
		Description:         m.Description,
		Type:                string(m.Type),
		EnumID:              m.EnumID,
		Ordinal:             m.Ordinal,
		Required:            m.Required,
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

type Enum struct {
	ID        string `gorm:"primaryKey;size:36"`
	Name      string `gorm:"uniqueIndex;not null;size:255"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Values    []EnumValue `gorm:"foreignKey:EnumID;constraint:OnDelete:CASCADE"`
}

func (e *Enum) ToModel() *domain.Enum {
	return &domain.Enum{
		ID:        e.ID,
		Name:      e.Name,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}

func EnumFromModel(m *domain.Enum) *Enum {
	return &Enum{
		ID:        m.ID,
		Name:      m.Name,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

type EnumValue struct {
	ID      string `gorm:"primaryKey;size:36"`
	EnumID  string `gorm:"not null;size:36;uniqueIndex:idx_enum_value"`
	Value   string `gorm:"not null;size:255;uniqueIndex:idx_enum_value"`
	Ordinal int    `gorm:"not null"`
}

func (e *EnumValue) ToModel() *domain.EnumValue {
	return &domain.EnumValue{
		ID:      e.ID,
		EnumID:  e.EnumID,
		Value:   e.Value,
		Ordinal: e.Ordinal,
	}
}

func EnumValueFromModel(m *domain.EnumValue) *EnumValue {
	return &EnumValue{
		ID:      m.ID,
		EnumID:  m.EnumID,
		Value:   m.Value,
		Ordinal: m.Ordinal,
	}
}

type CatalogVersion struct {
	ID             string `gorm:"primaryKey;size:36"`
	VersionLabel   string `gorm:"uniqueIndex;not null;size:255"`
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
		LifecycleStage: domain.LifecycleStage(c.LifecycleStage),
		CreatedAt:      c.CreatedAt,
		UpdatedAt:      c.UpdatedAt,
	}
}

func CatalogVersionFromModel(m *domain.CatalogVersion) *CatalogVersion {
	return &CatalogVersion{
		ID:             m.ID,
		VersionLabel:   m.VersionLabel,
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

type EntityInstance struct {
	ID               string `gorm:"primaryKey;size:36"`
	EntityTypeID     string `gorm:"not null;size:36;uniqueIndex:idx_instance_scope"`
	CatalogVersionID string `gorm:"not null;size:36;uniqueIndex:idx_instance_scope"`
	ParentInstanceID string `gorm:"size:36;uniqueIndex:idx_instance_scope;default:''"`
	Name             string `gorm:"not null;size:255;uniqueIndex:idx_instance_scope"`
	Description      string `gorm:"size:1024"`
	Version          int    `gorm:"not null;default:1"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        *time.Time `gorm:"index"`
}

func (e *EntityInstance) ToModel() *domain.EntityInstance {
	return &domain.EntityInstance{
		ID:               e.ID,
		EntityTypeID:     e.EntityTypeID,
		CatalogVersionID: e.CatalogVersionID,
		ParentInstanceID: e.ParentInstanceID,
		Name:             e.Name,
		Description:      e.Description,
		Version:          e.Version,
		CreatedAt:        e.CreatedAt,
		UpdatedAt:        e.UpdatedAt,
		DeletedAt:        e.DeletedAt,
	}
}

func EntityInstanceFromModel(m *domain.EntityInstance) *EntityInstance {
	return &EntityInstance{
		ID:               m.ID,
		EntityTypeID:     m.EntityTypeID,
		CatalogVersionID: m.CatalogVersionID,
		ParentInstanceID: m.ParentInstanceID,
		Name:             m.Name,
		Description:      m.Description,
		Version:          m.Version,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
		DeletedAt:        m.DeletedAt,
	}
}

type InstanceAttributeValue struct {
	ID              string `gorm:"primaryKey;size:36"`
	InstanceID      string `gorm:"not null;size:36;uniqueIndex:idx_iav_unique"`
	InstanceVersion int    `gorm:"not null;uniqueIndex:idx_iav_unique"`
	AttributeID     string `gorm:"not null;size:36;uniqueIndex:idx_iav_unique"`
	ValueString     string `gorm:"size:4096"`
	ValueNumber     *float64
	ValueEnum       string `gorm:"size:255"`
}

func (i *InstanceAttributeValue) ToModel() *domain.InstanceAttributeValue {
	return &domain.InstanceAttributeValue{
		ID:              i.ID,
		InstanceID:      i.InstanceID,
		InstanceVersion: i.InstanceVersion,
		AttributeID:     i.AttributeID,
		ValueString:     i.ValueString,
		ValueNumber:     i.ValueNumber,
		ValueEnum:       i.ValueEnum,
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
		ValueEnum:       m.ValueEnum,
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
		&Enum{},
		&EnumValue{},
		&CatalogVersion{},
		&CatalogVersionPin{},
		&LifecycleTransition{},
		&EntityInstance{},
		&InstanceAttributeValue{},
		&AssociationLink{},
	}
}

// InitDB initializes the database with auto-migration and data fixups.
func InitDB(db *gorm.DB) error {
	// Pre-migration: if associations table exists but has no name column, add it
	// as nullable first, populate names, then let AutoMigrate add the NOT NULL constraint.
	if db.Migrator().HasTable(&Association{}) && !db.Migrator().HasColumn(&Association{}, "Name") {
		// Add column as nullable
		if err := db.Exec("ALTER TABLE associations ADD COLUMN name VARCHAR(255) DEFAULT ''").Error; err != nil {
			return err
		}
		// Populate names from target_role, then source_role, then type
		var unnamed []Association
		if err := db.Where("name = ''").Find(&unnamed).Error; err != nil {
			return err
		}
		for i := range unnamed {
			n := unnamed[i].TargetRole
			if n == "" {
				n = unnamed[i].SourceRole
			}
			if n == "" {
				n = unnamed[i].Type + "_assoc"
			}
			unnamed[i].Name = n
			if err := db.Save(&unnamed[i]).Error; err != nil {
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
