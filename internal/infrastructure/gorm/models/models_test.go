package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	domain "github.com/project-catalyst/pc-asset-hub/internal/domain/models"
)

func TestEntityTypeConversion(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	m := &domain.EntityType{ID: "id1", Name: "Model", CreatedAt: now, UpdatedAt: now}
	g := EntityTypeFromModel(m)
	assert.Equal(t, "id1", g.ID)
	assert.Equal(t, "Model", g.Name)
	back := g.ToModel()
	assert.Equal(t, m.ID, back.ID)
	assert.Equal(t, m.Name, back.Name)
}

func TestEntityTypeVersionConversion(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	m := &domain.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 2, Description: "desc", CreatedAt: now}
	g := EntityTypeVersionFromModel(m)
	assert.Equal(t, 2, g.Version)
	back := g.ToModel()
	assert.Equal(t, m.ID, back.ID)
	assert.Equal(t, m.Version, back.Version)
}

func TestAttributeConversion(t *testing.T) {
	m := &domain.Attribute{ID: "a1", EntityTypeVersionID: "v1", Name: "attr", Type: domain.AttributeTypeString, EnumID: "e1", Ordinal: 3, Required: true}
	g := AttributeFromModel(m)
	assert.Equal(t, "string", g.Type)
	assert.True(t, g.Required)
	back := g.ToModel()
	assert.Equal(t, domain.AttributeTypeString, back.Type)
	assert.True(t, back.Required)
	assert.Equal(t, "e1", back.EnumID)
}

func TestAssociationConversion(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	m := &domain.Association{ID: "as1", EntityTypeVersionID: "v1", TargetEntityTypeID: "et2", Type: domain.AssociationTypeContainment, SourceRole: "src", TargetRole: "tgt", CreatedAt: now}
	g := AssociationFromModel(m)
	assert.Equal(t, "containment", g.Type)
	back := g.ToModel()
	assert.Equal(t, domain.AssociationTypeContainment, back.Type)
	assert.Equal(t, "src", back.SourceRole)
}

func TestEnumConversion(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	m := &domain.Enum{ID: "e1", Name: "Status", CreatedAt: now, UpdatedAt: now}
	g := EnumFromModel(m)
	assert.Equal(t, "Status", g.Name)
	back := g.ToModel()
	assert.Equal(t, m.ID, back.ID)
}

func TestEnumValueConversion(t *testing.T) {
	m := &domain.EnumValue{ID: "ev1", EnumID: "e1", Value: "active", Ordinal: 0}
	g := EnumValueFromModel(m)
	assert.Equal(t, "active", g.Value)
	back := g.ToModel()
	assert.Equal(t, m.Value, back.Value)
	assert.Equal(t, 0, back.Ordinal)
}

func TestCatalogVersionConversion(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	m := &domain.CatalogVersion{ID: "cv1", VersionLabel: "v1.0", LifecycleStage: domain.LifecycleStageTesting, CreatedAt: now, UpdatedAt: now}
	g := CatalogVersionFromModel(m)
	assert.Equal(t, "testing", g.LifecycleStage)
	back := g.ToModel()
	assert.Equal(t, domain.LifecycleStageTesting, back.LifecycleStage)
}

func TestCatalogVersionPinConversion(t *testing.T) {
	m := &domain.CatalogVersionPin{ID: "p1", CatalogVersionID: "cv1", EntityTypeVersionID: "v1"}
	g := CatalogVersionPinFromModel(m)
	assert.Equal(t, "cv1", g.CatalogVersionID)
	back := g.ToModel()
	assert.Equal(t, m.EntityTypeVersionID, back.EntityTypeVersionID)
}

func TestLifecycleTransitionConversion(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	m := &domain.LifecycleTransition{ID: "lt1", CatalogVersionID: "cv1", FromStage: "dev", ToStage: "test", PerformedBy: "admin", PerformedAt: now, Notes: "note"}
	g := LifecycleTransitionFromModel(m)
	assert.Equal(t, "note", g.Notes)
	back := g.ToModel()
	assert.Equal(t, "admin", back.PerformedBy)
	assert.Equal(t, "note", back.Notes)
}

func TestEntityInstanceConversion(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	del := now.Add(time.Hour)
	m := &domain.EntityInstance{ID: "i1", EntityTypeID: "et1", CatalogVersionID: "cv1", ParentInstanceID: "p1", Name: "inst", Description: "desc", Version: 2, CreatedAt: now, UpdatedAt: now, DeletedAt: &del}
	g := EntityInstanceFromModel(m)
	assert.Equal(t, "inst", g.Name)
	assert.NotNil(t, g.DeletedAt)
	back := g.ToModel()
	assert.Equal(t, 2, back.Version)
	assert.NotNil(t, back.DeletedAt)
}

func TestInstanceAttributeValueConversion(t *testing.T) {
	num := 3.14
	m := &domain.InstanceAttributeValue{ID: "iav1", InstanceID: "i1", InstanceVersion: 1, AttributeID: "a1", ValueString: "hello", ValueNumber: &num, ValueEnum: "active"}
	g := InstanceAttributeValueFromModel(m)
	assert.Equal(t, "hello", g.ValueString)
	assert.NotNil(t, g.ValueNumber)
	back := g.ToModel()
	assert.Equal(t, "active", back.ValueEnum)
	assert.Equal(t, &num, back.ValueNumber)
}

func TestAssociationLinkConversion(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	m := &domain.AssociationLink{ID: "al1", AssociationID: "as1", SourceInstanceID: "i1", TargetInstanceID: "i2", CreatedAt: now}
	g := AssociationLinkFromModel(m)
	assert.Equal(t, "i1", g.SourceInstanceID)
	back := g.ToModel()
	assert.Equal(t, "i2", back.TargetInstanceID)
}

func TestAllModels(t *testing.T) {
	models := AllModels()
	assert.Len(t, models, 12)
}

func TestInitDB(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	assert.NoError(t, err)
	assert.NoError(t, InitDB(db))
	// Verify tables exist by running a query
	var count int64
	assert.NoError(t, db.Table("entity_types").Count(&count).Error)
}
