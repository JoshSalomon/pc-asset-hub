package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	m := &domain.Attribute{ID: "a1", EntityTypeVersionID: "v1", Name: "attr", TypeDefinitionVersionID: "tdv-1", Ordinal: 3, Required: true}
	g := AttributeFromModel(m)
	assert.Equal(t, "tdv-1", g.TypeDefinitionVersionID)
	assert.True(t, g.Required)
	back := g.ToModel()
	assert.Equal(t, "tdv-1", back.TypeDefinitionVersionID)
	assert.True(t, back.Required)
	assert.Equal(t, 3, back.Ordinal)
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

func TestTypeDefinitionConversion(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	m := &domain.TypeDefinition{ID: "td1", Name: "Status", Description: "Status type", BaseType: domain.BaseTypeEnum, System: true, CreatedAt: now, UpdatedAt: now}
	g := TypeDefinitionFromModel(m)
	assert.Equal(t, "Status", g.Name)
	assert.Equal(t, "enum", g.BaseType)
	assert.True(t, g.System)
	back := g.ToModel()
	assert.Equal(t, m.ID, back.ID)
	assert.Equal(t, domain.BaseTypeEnum, back.BaseType)
	assert.True(t, back.System)
}

func TestTypeDefinitionVersionConversion(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	m := &domain.TypeDefinitionVersion{ID: "tdv1", TypeDefinitionID: "td1", VersionNumber: 1, Constraints: map[string]any{"allowed_values": []string{"active", "inactive"}}, CreatedAt: now}
	g := TypeDefinitionVersionFromModel(m)
	assert.Equal(t, "td1", g.TypeDefinitionID)
	assert.Equal(t, 1, g.VersionNumber)
	assert.Contains(t, g.Constraints, "allowed_values")
	back := g.ToModel()
	assert.Equal(t, m.ID, back.ID)
	assert.Equal(t, 1, back.VersionNumber)
	assert.Contains(t, back.Constraints, "allowed_values")
}

func TestCatalogVersionTypePinConversion(t *testing.T) {
	m := &domain.CatalogVersionTypePin{ID: "cvtp1", CatalogVersionID: "cv1", TypeDefinitionVersionID: "tdv1"}
	g := CatalogVersionTypePinFromModel(m)
	assert.Equal(t, "cv1", g.CatalogVersionID)
	assert.Equal(t, "tdv1", g.TypeDefinitionVersionID)
	back := g.ToModel()
	assert.Equal(t, m.ID, back.ID)
	assert.Equal(t, "tdv1", back.TypeDefinitionVersionID)
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
	m := &domain.EntityInstance{ID: "i1", EntityTypeID: "et1", CatalogID: "cv1", ParentInstanceID: "p1", Name: "inst", Description: "desc", Version: 2, CreatedAt: now, UpdatedAt: now, DeletedAt: &del}
	g := EntityInstanceFromModel(m)
	assert.Equal(t, "inst", g.Name)
	assert.NotNil(t, g.DeletedAt)
	back := g.ToModel()
	assert.Equal(t, 2, back.Version)
	assert.NotNil(t, back.DeletedAt)
}

func TestInstanceAttributeValueConversion(t *testing.T) {
	num := 3.14
	m := &domain.InstanceAttributeValue{ID: "iav1", InstanceID: "i1", InstanceVersion: 1, AttributeID: "a1", ValueString: "hello", ValueNumber: &num, ValueJSON: `{"key":"value"}`}
	g := InstanceAttributeValueFromModel(m)
	assert.Equal(t, "hello", g.ValueString)
	assert.NotNil(t, g.ValueNumber)
	assert.Equal(t, `{"key":"value"}`, g.ValueJSON)
	back := g.ToModel()
	assert.Equal(t, `{"key":"value"}`, back.ValueJSON)
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

func TestCatalogConversion(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	m := &domain.Catalog{
		ID: "c1", Name: "prod-app-a", Description: "Production",
		CatalogVersionID: "cv1", ValidationStatus: domain.ValidationStatusDraft,
		CreatedAt: now, UpdatedAt: now,
	}
	g := CatalogFromModel(m)
	assert.Equal(t, "prod-app-a", g.Name)
	assert.Equal(t, "draft", g.ValidationStatus)
	back := g.ToModel()
	assert.Equal(t, "c1", back.ID)
	assert.Equal(t, domain.ValidationStatusDraft, back.ValidationStatus)
	assert.Equal(t, "cv1", back.CatalogVersionID)
}

func TestAllModels(t *testing.T) {
	models := AllModels()
	assert.Len(t, models, 14)
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

// InitDB skips pre-migration when name column already exists (current state of all databases)
func TestInitDB_SkipsPreMigrationWhenNameExists(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// First InitDB creates all tables including associations with name column
	require.NoError(t, InitDB(db))

	// Insert an association
	err = db.Exec(`INSERT INTO associations (id, entity_type_version_id, target_entity_type_id, type, name, source_cardinality, target_cardinality, created_at)
		VALUES ('a1', 'etv1', 'et2', 'directional', 'uses', '0..n', '0..n', datetime('now'))`).Error
	require.NoError(t, err)

	// Second InitDB should skip the pre-migration (name column exists) and not error
	require.NoError(t, InitDB(db))

	// Verify the association is unchanged
	var assoc Association
	require.NoError(t, db.First(&assoc, "id = ?", "a1").Error)
	assert.Equal(t, "uses", assoc.Name)
}

// InitDB AutoMigrate error path — use an invalid DB to trigger AutoMigrate failure
func TestInitDB_AutoMigrateError(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	// Close the underlying connection to make AutoMigrate fail
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.Close()

	err = InitDB(db)
	assert.Error(t, err)
}

// InitDB is idempotent — running twice should not error
func TestInitDB_Idempotent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, InitDB(db))
	// Second call should also succeed
	require.NoError(t, InitDB(db))
}

// TypeDefinitionVersion ToModel — corrupted JSON in Constraints
func TestTypeDefinitionVersionToModel_CorruptedJSON(t *testing.T) {
	g := &TypeDefinitionVersion{
		ID:               "tdv-corrupt",
		TypeDefinitionID: "td1",
		VersionNumber:    1,
		Constraints:      "not{valid",
	}
	m := g.ToModel()
	require.NotNil(t, m)
	assert.Equal(t, "tdv-corrupt", m.ID)
	// Corrupted JSON: should have _raw key with original string
	assert.Contains(t, m.Constraints, "_raw")
	assert.Equal(t, "not{valid", m.Constraints["_raw"])
}

// TypeDefinitionVersion ToModel — empty Constraints
func TestTypeDefinitionVersionToModel_EmptyConstraints(t *testing.T) {
	g := &TypeDefinitionVersion{
		ID:               "tdv-empty",
		TypeDefinitionID: "td1",
		VersionNumber:    1,
		Constraints:      "",
	}
	m := g.ToModel()
	require.NotNil(t, m)
	assert.Equal(t, map[string]any{}, m.Constraints)
}

// TypeDefinitionVersion ToModel — "{}" Constraints (empty JSON object)
func TestTypeDefinitionVersionToModel_EmptyJSONObject(t *testing.T) {
	g := &TypeDefinitionVersion{
		ID:               "tdv-empty-obj",
		TypeDefinitionID: "td1",
		VersionNumber:    1,
		Constraints:      "{}",
	}
	m := g.ToModel()
	require.NotNil(t, m)
	// Both "" and "{}" hit the same nil-constraints branch
	assert.Equal(t, map[string]any{}, m.Constraints)
}

// InitDB containment cardinality fix — verifies containment associations get "0..1" source cardinality
func TestInitDB_ContainmentCardinalityFix(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, InitDB(db))

	// Insert a containment association with empty source_cardinality
	assoc := Association{
		ID:                  "fix-test",
		EntityTypeVersionID: "etv1",
		TargetEntityTypeID:  "et2",
		Type:                "containment",
		Name:                "contains",
		SourceCardinality:   "",
		TargetCardinality:   "0..n",
	}
	require.NoError(t, db.Create(&assoc).Error)

	// Run InitDB again — should fix source_cardinality to "0..1"
	require.NoError(t, InitDB(db))

	var fixed Association
	require.NoError(t, db.First(&fixed, "id = ?", "fix-test").Error)
	assert.Equal(t, "0..1", fixed.SourceCardinality)
}
