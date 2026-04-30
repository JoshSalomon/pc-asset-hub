package meta

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository/mocks"
)

func newTypeDefSvc() (*TypeDefinitionService, *mocks.MockTypeDefinitionRepo, *mocks.MockTypeDefinitionVersionRepo, *mocks.MockAttributeRepo) {
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	svc := NewTypeDefinitionService(tdRepo, tdvRepo, attrRepo)
	return svc, tdRepo, tdvRepo, attrRepo
}

// === CreateTypeDefinition ===

func TestCreateTypeDefinition_Success(t *testing.T) {
	svc, tdRepo, tdvRepo, _ := newTypeDefSvc()

	tdRepo.On("Create", mock.Anything, mock.MatchedBy(func(td *models.TypeDefinition) bool {
		return td.Name == "guardrailID" && td.BaseType == models.BaseTypeString && !td.System
	})).Return(nil)
	tdvRepo.On("Create", mock.Anything, mock.MatchedBy(func(tdv *models.TypeDefinitionVersion) bool {
		return tdv.VersionNumber == 1 && tdv.Constraints["max_length"] == float64(12)
	})).Return(nil)

	td, tdv, err := svc.CreateTypeDefinition(context.Background(), "guardrailID", "Guardrail ID", models.BaseTypeString, map[string]any{"max_length": float64(12)})
	assert.NoError(t, err)
	assert.Equal(t, "guardrailID", td.Name)
	assert.Equal(t, models.BaseTypeString, td.BaseType)
	assert.False(t, td.System)
	assert.Equal(t, 1, tdv.VersionNumber)
	assert.Equal(t, float64(12), tdv.Constraints["max_length"])
}

func TestCreateTypeDefinition_EmptyName(t *testing.T) {
	svc, _, _, _ := newTypeDefSvc()

	_, _, err := svc.CreateTypeDefinition(context.Background(), "", "desc", models.BaseTypeString, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestCreateTypeDefinition_InvalidBaseType(t *testing.T) {
	svc, _, _, _ := newTypeDefSvc()

	_, _, err := svc.CreateTypeDefinition(context.Background(), "test", "", "invalid", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid base type")
}

func TestCreateTypeDefinition_EnumRequiresValues(t *testing.T) {
	svc, _, _, _ := newTypeDefSvc()

	_, _, err := svc.CreateTypeDefinition(context.Background(), "status", "", models.BaseTypeEnum, map[string]any{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "values")
}

func TestCreateTypeDefinition_EnumWithValues(t *testing.T) {
	svc, tdRepo, tdvRepo, _ := newTypeDefSvc()

	tdRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	tdvRepo.On("Create", mock.Anything, mock.MatchedBy(func(tdv *models.TypeDefinitionVersion) bool {
		vals, ok := tdv.Constraints["values"].([]any)
		return ok && len(vals) == 3
	})).Return(nil)

	td, tdv, err := svc.CreateTypeDefinition(context.Background(), "status", "", models.BaseTypeEnum, map[string]any{
		"values": []any{"active", "inactive", "archived"},
	})
	assert.NoError(t, err)
	assert.Equal(t, models.BaseTypeEnum, td.BaseType)
	assert.NotNil(t, tdv)
}

func TestCreateTypeDefinition_SystemType(t *testing.T) {
	svc, tdRepo, tdvRepo, _ := newTypeDefSvc()

	tdRepo.On("Create", mock.Anything, mock.MatchedBy(func(td *models.TypeDefinition) bool {
		return td.Name == "string" && td.System
	})).Return(nil)
	tdvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	td, err := svc.CreateSystemTypeDefinition(context.Background(), "string", models.BaseTypeString)
	assert.NoError(t, err)
	assert.True(t, td.System)
}

// === GetTypeDefinition ===

func TestGetTypeDefinition_Success(t *testing.T) {
	svc, tdRepo, tdvRepo, _ := newTypeDefSvc()

	td := &models.TypeDefinition{ID: "td-1", Name: "status", BaseType: models.BaseTypeEnum}
	latestVersion := &models.TypeDefinitionVersion{ID: "tdv-1", TypeDefinitionID: "td-1", VersionNumber: 2}

	tdRepo.On("GetByID", mock.Anything, "td-1").Return(td, nil)
	tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-1").Return(latestVersion, nil)

	result, version, err := svc.GetTypeDefinition(context.Background(), "td-1")
	assert.NoError(t, err)
	assert.Equal(t, "status", result.Name)
	assert.Equal(t, 2, version.VersionNumber)
}

// === ListTypeDefinitions ===

func TestListTypeDefinitions_Success(t *testing.T) {
	svc, tdRepo, _, _ := newTypeDefSvc()

	tdRepo.On("List", mock.Anything, mock.Anything).Return([]*models.TypeDefinition{
		{ID: "td-1", Name: "string", BaseType: models.BaseTypeString, System: true},
		{ID: "td-2", Name: "status", BaseType: models.BaseTypeEnum},
	}, 2, nil)

	items, total, err := svc.ListTypeDefinitions(context.Background(), models.ListParams{})
	assert.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, items, 2)
}

// === UpdateTypeDefinition ===

func TestUpdateTypeDefinition_CreatesNewVersion(t *testing.T) {
	svc, tdRepo, tdvRepo, _ := newTypeDefSvc()

	td := &models.TypeDefinition{ID: "td-1", Name: "guardrailID", BaseType: models.BaseTypeString}
	latestTDV := &models.TypeDefinitionVersion{ID: "tdv-1", TypeDefinitionID: "td-1", VersionNumber: 1, Constraints: map[string]any{"max_length": float64(12)}}

	tdRepo.On("GetByID", mock.Anything, "td-1").Return(td, nil)
	tdRepo.On("Update", mock.Anything, mock.Anything).Return(nil)
	tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-1").Return(latestTDV, nil)
	tdvRepo.On("Create", mock.Anything, mock.MatchedBy(func(tdv *models.TypeDefinitionVersion) bool {
		return tdv.VersionNumber == 2 && tdv.Constraints["max_length"] == float64(16)
	})).Return(nil)

	newDesc := "Updated description"
	newTDV, err := svc.UpdateTypeDefinition(context.Background(), "td-1", &newDesc, map[string]any{"max_length": float64(16)})
	assert.NoError(t, err)
	assert.Equal(t, 2, newTDV.VersionNumber)
}

func TestUpdateTypeDefinition_SystemTypeCannotBeUpdated(t *testing.T) {
	svc, tdRepo, _, _ := newTypeDefSvc()

	td := &models.TypeDefinition{ID: "td-1", Name: "string", BaseType: models.BaseTypeString, System: true}
	tdRepo.On("GetByID", mock.Anything, "td-1").Return(td, nil)

	_, err := svc.UpdateTypeDefinition(context.Background(), "td-1", nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "system type")
}

// === DeleteTypeDefinition ===

func TestDeleteTypeDefinition_Success(t *testing.T) {
	svc, tdRepo, tdvRepo, attrRepo, _, _ := newTypeDefSvcWithPins()

	td := &models.TypeDefinition{ID: "td-1", Name: "guardrailID", BaseType: models.BaseTypeString}
	tdRepo.On("GetByID", mock.Anything, "td-1").Return(td, nil)
	tdvRepo.On("ListByTypeDefinition", mock.Anything, "td-1").Return([]*models.TypeDefinitionVersion{
		{ID: "tdv-1", TypeDefinitionID: "td-1", VersionNumber: 1},
	}, nil)
	attrRepo.On("ListByTypeDefinitionVersionIDs", mock.Anything, []string{"tdv-1"}).Return([]*models.Attribute{}, nil)
	tdRepo.On("Delete", mock.Anything, "td-1").Return(nil)

	err := svc.DeleteTypeDefinition(context.Background(), "td-1")
	assert.NoError(t, err)
}

func TestDeleteTypeDefinition_SystemTypeCannotBeDeleted(t *testing.T) {
	svc, tdRepo, _, _ := newTypeDefSvc()

	td := &models.TypeDefinition{ID: "td-1", Name: "string", BaseType: models.BaseTypeString, System: true}
	tdRepo.On("GetByID", mock.Anything, "td-1").Return(td, nil)

	err := svc.DeleteTypeDefinition(context.Background(), "td-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "system type")
}

// === ListVersions ===

func TestListVersions_Success(t *testing.T) {
	svc, _, tdvRepo, _ := newTypeDefSvc()

	tdvRepo.On("ListByTypeDefinition", mock.Anything, "td-1").Return([]*models.TypeDefinitionVersion{
		{ID: "tdv-1", VersionNumber: 1},
		{ID: "tdv-2", VersionNumber: 2},
	}, nil)

	versions, err := svc.ListVersions(context.Background(), "td-1")
	assert.NoError(t, err)
	assert.Len(t, versions, 2)
}

// === Constraint Validation ===

func TestValidateConstraints_StringMaxLength(t *testing.T) {
	svc, _, _, _ := newTypeDefSvc()

	err := svc.ValidateConstraints(models.BaseTypeString, map[string]any{"max_length": float64(100)})
	assert.NoError(t, err)

	err = svc.ValidateConstraints(models.BaseTypeString, map[string]any{"max_length": float64(-1)})
	assert.Error(t, err)
}

func TestValidateConstraints_IntegerMinMax(t *testing.T) {
	svc, _, _, _ := newTypeDefSvc()

	err := svc.ValidateConstraints(models.BaseTypeInteger, map[string]any{"min": float64(0), "max": float64(100)})
	assert.NoError(t, err)

	err = svc.ValidateConstraints(models.BaseTypeInteger, map[string]any{"min": float64(100), "max": float64(0)})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "min")
}

func TestValidateConstraints_EnumValues(t *testing.T) {
	svc, _, _, _ := newTypeDefSvc()

	err := svc.ValidateConstraints(models.BaseTypeEnum, map[string]any{"values": []any{"a", "b"}})
	assert.NoError(t, err)

	err = svc.ValidateConstraints(models.BaseTypeEnum, map[string]any{})
	assert.Error(t, err)
}

func TestValidateConstraints_ListElementType(t *testing.T) {
	svc, _, _, _ := newTypeDefSvc()

	err := svc.ValidateConstraints(models.BaseTypeList, map[string]any{"element_base_type": "string"})
	assert.NoError(t, err)

	err = svc.ValidateConstraints(models.BaseTypeList, map[string]any{"element_base_type": "list"})
	assert.Error(t, err) // list of list not allowed
}

func TestValidateConstraints_StringPattern(t *testing.T) {
	svc, _, _, _ := newTypeDefSvc()

	err := svc.ValidateConstraints(models.BaseTypeString, map[string]any{"pattern": "^[A-Z]+$"})
	assert.NoError(t, err)

	err = svc.ValidateConstraints(models.BaseTypeString, map[string]any{"pattern": "[invalid"})
	assert.Error(t, err)
}

// === CreateTypeDefinition error paths ===

func TestCreateTypeDefinition_TdRepoCreateError(t *testing.T) {
	svc, tdRepo, _, _ := newTypeDefSvc()

	tdRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("db error"))

	_, _, err := svc.CreateTypeDefinition(context.Background(), "test", "desc", models.BaseTypeString, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

func TestCreateTypeDefinition_TdvRepoCreateError(t *testing.T) {
	svc, tdRepo, tdvRepo, _ := newTypeDefSvc()

	tdRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	tdvRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("version db error"))

	_, _, err := svc.CreateTypeDefinition(context.Background(), "test", "desc", models.BaseTypeString, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "version db error")
}

func TestCreateTypeDefinition_InvalidConstraints(t *testing.T) {
	svc, _, _, _ := newTypeDefSvc()

	// String with negative max_length
	_, _, err := svc.CreateTypeDefinition(context.Background(), "test", "", models.BaseTypeString, map[string]any{"max_length": float64(-5)})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max_length")
}

// === CreateSystemTypeDefinition error paths ===

func TestCreateSystemTypeDefinition_TdRepoCreateError(t *testing.T) {
	svc, tdRepo, _, _ := newTypeDefSvc()

	tdRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("system td error"))

	_, err := svc.CreateSystemTypeDefinition(context.Background(), "string", models.BaseTypeString)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "system td error")
}

func TestCreateSystemTypeDefinition_TdvRepoCreateError(t *testing.T) {
	svc, tdRepo, tdvRepo, _ := newTypeDefSvc()

	tdRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	tdvRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("system tdv error"))

	_, err := svc.CreateSystemTypeDefinition(context.Background(), "string", models.BaseTypeString)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "system tdv error")
}

// === GetTypeDefinition error paths ===

func TestGetTypeDefinition_GetByIDError(t *testing.T) {
	svc, tdRepo, _, _ := newTypeDefSvc()

	tdRepo.On("GetByID", mock.Anything, "td-missing").Return(nil, domainerrors.NewNotFound("TypeDefinition", "td-missing"))

	_, _, err := svc.GetTypeDefinition(context.Background(), "td-missing")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
}

func TestGetTypeDefinition_GetLatestByTypeDefinitionError(t *testing.T) {
	svc, tdRepo, tdvRepo, _ := newTypeDefSvc()

	td := &models.TypeDefinition{ID: "td-1", Name: "test", BaseType: models.BaseTypeString}
	tdRepo.On("GetByID", mock.Anything, "td-1").Return(td, nil)
	tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-1").Return(nil, errors.New("version lookup failed"))

	_, _, err := svc.GetTypeDefinition(context.Background(), "td-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "version lookup failed")
}

// === GetLatestVersionNumbers ===

func TestGetLatestVersionNumbers_Success(t *testing.T) {
	svc, _, tdvRepo, _ := newTypeDefSvc()

	tdvRepo.On("GetLatestByTypeDefinitions", mock.Anything, []string{"td-1", "td-2"}).Return(map[string]*models.TypeDefinitionVersion{
		"td-1": {VersionNumber: 3},
		"td-2": {VersionNumber: 1},
	}, nil)

	result, err := svc.GetLatestVersionNumbers(context.Background(), []string{"td-1", "td-2"})
	assert.NoError(t, err)
	assert.Equal(t, 3, result["td-1"])
	assert.Equal(t, 1, result["td-2"])
}

func TestGetLatestVersionNumbers_Error(t *testing.T) {
	svc, _, tdvRepo, _ := newTypeDefSvc()

	tdvRepo.On("GetLatestByTypeDefinitions", mock.Anything, mock.Anything).Return(nil, errors.New("batch error"))

	_, err := svc.GetLatestVersionNumbers(context.Background(), []string{"td-1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "batch error")
}

func TestGetLatestVersionInfo_Success(t *testing.T) {
	svc, _, tdvRepo, _ := newTypeDefSvc()

	tdvRepo.On("GetLatestByTypeDefinitions", mock.Anything, []string{"td-1", "td-2"}).Return(map[string]*models.TypeDefinitionVersion{
		"td-1": {ID: "v-1-3", VersionNumber: 3},
		"td-2": {ID: "v-2-1", VersionNumber: 1},
	}, nil)

	numbers, ids, err := svc.GetLatestVersionInfo(context.Background(), []string{"td-1", "td-2"})
	assert.NoError(t, err)
	assert.Equal(t, 3, numbers["td-1"])
	assert.Equal(t, 1, numbers["td-2"])
	assert.Equal(t, "v-1-3", ids["td-1"])
	assert.Equal(t, "v-2-1", ids["td-2"])
}

func TestGetLatestVersionInfo_Error(t *testing.T) {
	svc, _, tdvRepo, _ := newTypeDefSvc()

	tdvRepo.On("GetLatestByTypeDefinitions", mock.Anything, mock.Anything).Return(nil, errors.New("batch error"))

	_, _, err := svc.GetLatestVersionInfo(context.Background(), []string{"td-1"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "batch error")
}

// === UpdateTypeDefinition error paths ===

func TestUpdateTypeDefinition_GetByIDError(t *testing.T) {
	svc, tdRepo, _, _ := newTypeDefSvc()

	tdRepo.On("GetByID", mock.Anything, "td-missing").Return(nil, domainerrors.NewNotFound("TypeDefinition", "td-missing"))

	_, err := svc.UpdateTypeDefinition(context.Background(), "td-missing", nil, nil)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
}

func TestUpdateTypeDefinition_UpdateRepoError(t *testing.T) {
	svc, tdRepo, _, _ := newTypeDefSvc()

	td := &models.TypeDefinition{ID: "td-1", Name: "test", BaseType: models.BaseTypeString}
	tdRepo.On("GetByID", mock.Anything, "td-1").Return(td, nil)
	tdRepo.On("Update", mock.Anything, mock.Anything).Return(errors.New("update error"))

	desc := "new desc"
	_, err := svc.UpdateTypeDefinition(context.Background(), "td-1", &desc, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update error")
}

func TestUpdateTypeDefinition_GetLatestError(t *testing.T) {
	svc, tdRepo, tdvRepo, _ := newTypeDefSvc()

	td := &models.TypeDefinition{ID: "td-1", Name: "test", BaseType: models.BaseTypeString}
	tdRepo.On("GetByID", mock.Anything, "td-1").Return(td, nil)
	tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-1").Return(nil, errors.New("latest error"))

	_, err := svc.UpdateTypeDefinition(context.Background(), "td-1", nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "latest error")
}

func TestUpdateTypeDefinition_TdvCreateError(t *testing.T) {
	svc, tdRepo, tdvRepo, _ := newTypeDefSvc()

	td := &models.TypeDefinition{ID: "td-1", Name: "test", BaseType: models.BaseTypeString}
	latestTDV := &models.TypeDefinitionVersion{ID: "tdv-1", TypeDefinitionID: "td-1", VersionNumber: 1, Constraints: map[string]any{}}

	tdRepo.On("GetByID", mock.Anything, "td-1").Return(td, nil)
	tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-1").Return(latestTDV, nil)
	tdvRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("create version error"))

	_, err := svc.UpdateTypeDefinition(context.Background(), "td-1", nil, map[string]any{"max_length": float64(10)})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create version error")
}

func TestUpdateTypeDefinition_InvalidConstraints(t *testing.T) {
	svc, tdRepo, tdvRepo, _ := newTypeDefSvc()

	td := &models.TypeDefinition{ID: "td-1", Name: "test", BaseType: models.BaseTypeString}
	latestTDV := &models.TypeDefinitionVersion{ID: "tdv-1", TypeDefinitionID: "td-1", VersionNumber: 1, Constraints: map[string]any{}}

	tdRepo.On("GetByID", mock.Anything, "td-1").Return(td, nil)
	tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-1").Return(latestTDV, nil)

	// Negative max_length should fail validation
	_, err := svc.UpdateTypeDefinition(context.Background(), "td-1", nil, map[string]any{"max_length": float64(-5)})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max_length")
}

func TestUpdateTypeDefinition_NoConstraintsCarryForward(t *testing.T) {
	svc, tdRepo, tdvRepo, _ := newTypeDefSvc()

	td := &models.TypeDefinition{ID: "td-1", Name: "test", BaseType: models.BaseTypeString}
	latestTDV := &models.TypeDefinitionVersion{ID: "tdv-1", TypeDefinitionID: "td-1", VersionNumber: 1, Constraints: map[string]any{"max_length": float64(10)}}

	tdRepo.On("GetByID", mock.Anything, "td-1").Return(td, nil)
	tdvRepo.On("GetLatestByTypeDefinition", mock.Anything, "td-1").Return(latestTDV, nil)
	tdvRepo.On("Create", mock.Anything, mock.MatchedBy(func(tdv *models.TypeDefinitionVersion) bool {
		return tdv.VersionNumber == 2 && tdv.Constraints["max_length"] == float64(10) // carried forward
	})).Return(nil)

	tdv, err := svc.UpdateTypeDefinition(context.Background(), "td-1", nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, 2, tdv.VersionNumber)
}

// === DeleteTypeDefinition error paths ===

func TestDeleteTypeDefinition_GetByIDError(t *testing.T) {
	svc, tdRepo, _, _ := newTypeDefSvc()

	tdRepo.On("GetByID", mock.Anything, "td-missing").Return(nil, domainerrors.NewNotFound("TypeDefinition", "td-missing"))

	err := svc.DeleteTypeDefinition(context.Background(), "td-missing")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
}

// === GetVersion ===

func TestGetVersion_Success(t *testing.T) {
	svc, _, tdvRepo, _ := newTypeDefSvc()

	tdvRepo.On("GetByVersion", mock.Anything, "td-1", 2).Return(&models.TypeDefinitionVersion{
		ID: "tdv-2", TypeDefinitionID: "td-1", VersionNumber: 2, Constraints: map[string]any{"max_length": float64(16)},
	}, nil)

	v, err := svc.GetVersion(context.Background(), "td-1", 2)
	assert.NoError(t, err)
	assert.Equal(t, 2, v.VersionNumber)
	assert.Equal(t, "tdv-2", v.ID)
}

func TestGetVersion_NotFound(t *testing.T) {
	svc, _, tdvRepo, _ := newTypeDefSvc()

	tdvRepo.On("GetByVersion", mock.Anything, "td-1", 99).Return(nil, domainerrors.NewNotFound("TypeDefinitionVersion", "v99"))

	_, err := svc.GetVersion(context.Background(), "td-1", 99)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
}

func TestGetVersion_RepoError(t *testing.T) {
	svc, _, tdvRepo, _ := newTypeDefSvc()

	tdvRepo.On("GetByVersion", mock.Anything, "td-1", 1).Return(nil, errors.New("db error"))

	_, err := svc.GetVersion(context.Background(), "td-1", 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

// === ValidateConstraints - additional base types ===

func TestValidateConstraints_BooleanNoConstraints(t *testing.T) {
	svc, _, _, _ := newTypeDefSvc()

	err := svc.ValidateConstraints(models.BaseTypeBoolean, map[string]any{})
	assert.NoError(t, err)
}

func TestValidateConstraints_DateNoConstraints(t *testing.T) {
	svc, _, _, _ := newTypeDefSvc()

	err := svc.ValidateConstraints(models.BaseTypeDate, map[string]any{})
	assert.NoError(t, err)
}

func TestValidateConstraints_URLNoConstraints(t *testing.T) {
	svc, _, _, _ := newTypeDefSvc()

	err := svc.ValidateConstraints(models.BaseTypeURL, map[string]any{})
	assert.NoError(t, err)
}

func TestValidateConstraints_JSONNoConstraints(t *testing.T) {
	svc, _, _, _ := newTypeDefSvc()

	err := svc.ValidateConstraints(models.BaseTypeJSON, map[string]any{})
	assert.NoError(t, err)
}

func TestValidateConstraints_UnknownBaseType(t *testing.T) {
	svc, _, _, _ := newTypeDefSvc()

	err := svc.ValidateConstraints("unknown_type", map[string]any{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown base type")
}

func TestValidateConstraints_NumberMinMax(t *testing.T) {
	svc, _, _, _ := newTypeDefSvc()

	err := svc.ValidateConstraints(models.BaseTypeNumber, map[string]any{"min": float64(0.5), "max": float64(100.5)})
	assert.NoError(t, err)

	err = svc.ValidateConstraints(models.BaseTypeNumber, map[string]any{"min": float64(100), "max": float64(0)})
	assert.Error(t, err)
}

// === validateStringConstraints - pattern not-a-string ===

func TestValidateStringConstraints_PatternNotAString(t *testing.T) {
	err := validateStringConstraints(map[string]any{"pattern": 42})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pattern must be a string")
}

// === validateMinMaxConstraints - not-a-number cases ===

func TestValidateMinMaxConstraints_MinNotANumber(t *testing.T) {
	err := validateMinMaxConstraints(map[string]any{"min": "not_a_number"}, "integer")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "integer min must be a number")
}

func TestValidateMinMaxConstraints_MaxNotANumber(t *testing.T) {
	err := validateMinMaxConstraints(map[string]any{"max": "not_a_number"}, "integer")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "integer max must be a number")
}

// === validateEnumConstraints - additional cases ===

func TestValidateEnumConstraints_ValuesNotAnArray(t *testing.T) {
	err := validateEnumConstraints(map[string]any{"values": "not_an_array"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "enum values must be a non-empty array")
}

func TestValidateEnumConstraints_ValuesEmpty(t *testing.T) {
	err := validateEnumConstraints(map[string]any{"values": []any{}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "enum values must be a non-empty array")
}

func TestValidateEnumConstraints_ValueNotAString(t *testing.T) {
	err := validateEnumConstraints(map[string]any{"values": []any{42}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "enum values must be strings")
}

func TestValidateEnumConstraints_DuplicateValue(t *testing.T) {
	err := validateEnumConstraints(map[string]any{"values": []any{"a", "b", "a"}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate enum value")
}

// === validateListConstraints - additional cases ===

func TestValidateListConstraints_ElementBaseTypeNotAString(t *testing.T) {
	err := validateListConstraints(map[string]any{"element_base_type": 42})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "element_base_type must be a string")
}

func TestValidateListConstraints_InvalidElementBaseType(t *testing.T) {
	err := validateListConstraints(map[string]any{"element_base_type": "invalid_type"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid element base type")
}

func TestValidateListConstraints_ElementTypeJSON(t *testing.T) {
	err := validateListConstraints(map[string]any{"element_base_type": "json"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "list element type cannot be json")
}

func TestValidateListConstraints_ElementTypeEnum(t *testing.T) {
	err := validateListConstraints(map[string]any{"element_base_type": "enum"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "list element type cannot be enum")
}

func TestValidateListConstraints_MaxLengthNotANumber(t *testing.T) {
	err := validateListConstraints(map[string]any{"max_length": "not_a_number"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "list max_length must be a positive number")
}

func TestValidateListConstraints_MaxLengthNegative(t *testing.T) {
	err := validateListConstraints(map[string]any{"max_length": float64(-1)})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "list max_length must be a positive number")
}

func TestValidateListConstraints_ValidMaxLength(t *testing.T) {
	err := validateListConstraints(map[string]any{"max_length": float64(10)})
	assert.NoError(t, err)
}

func TestValidateListConstraints_ValidStringElementType(t *testing.T) {
	err := validateListConstraints(map[string]any{"element_base_type": "string"})
	assert.NoError(t, err)
}

func TestValidateListConstraints_EmptyConstraints(t *testing.T) {
	err := validateListConstraints(map[string]any{})
	assert.NoError(t, err)
}

// === toFloat64 - additional cases ===

func TestToFloat64_Int(t *testing.T) {
	v, ok := toFloat64(int(42))
	assert.True(t, ok)
	assert.Equal(t, float64(42), v)
}

func TestToFloat64_Int64(t *testing.T) {
	v, ok := toFloat64(int64(42))
	assert.True(t, ok)
	assert.Equal(t, float64(42), v)
}

func TestToFloat64_Float64(t *testing.T) {
	v, ok := toFloat64(float64(3.14))
	assert.True(t, ok)
	assert.Equal(t, float64(3.14), v)
}

func TestToFloat64_String(t *testing.T) {
	_, ok := toFloat64("not a number")
	assert.False(t, ok)
}

func TestToFloat64_Bool(t *testing.T) {
	_, ok := toFloat64(true)
	assert.False(t, ok)
}

// === validateStringConstraints - max_length not a number ===

func TestValidateStringConstraints_MaxLengthNotANumber(t *testing.T) {
	err := validateStringConstraints(map[string]any{"max_length": "not_a_number"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max_length must be a positive number")
}

// === TD-1: Type Definition Deletion Safety ===

func newTypeDefSvcWithPins() (*TypeDefinitionService, *mocks.MockTypeDefinitionRepo, *mocks.MockTypeDefinitionVersionRepo, *mocks.MockAttributeRepo, *mocks.MockCatalogVersionPinRepo, *mocks.MockEntityTypeVersionRepo) {
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := NewTypeDefinitionService(tdRepo, tdvRepo, attrRepo, WithPinRepo(pinRepo), WithETVRepo(etvRepo))
	return svc, tdRepo, tdvRepo, attrRepo, pinRepo, etvRepo
}

// T-29.14: Delete type def used by attribute in CV-pinned entity type version → blocked
func TestT29_14_DeleteTypeDefUsedByPinnedVersion(t *testing.T) {
	svc, tdRepo, tdvRepo, attrRepo, pinRepo, _ := newTypeDefSvcWithPins()

	td := &models.TypeDefinition{ID: "td-1", Name: "guardrailID", BaseType: models.BaseTypeString}
	tdRepo.On("GetByID", mock.Anything, "td-1").Return(td, nil)

	// Type def has one version
	tdvRepo.On("ListByTypeDefinition", mock.Anything, "td-1").Return([]*models.TypeDefinitionVersion{
		{ID: "tdv-1", TypeDefinitionID: "td-1", VersionNumber: 1},
	}, nil)

	// Attribute in etv-1 references tdv-1
	attrRepo.On("ListByTypeDefinitionVersionIDs", mock.Anything, []string{"tdv-1"}).Return([]*models.Attribute{
		{ID: "attr-1", Name: "guard_id", EntityTypeVersionID: "etv-1", TypeDefinitionVersionID: "tdv-1"},
	}, nil)

	// etv-1 is pinned by a CV
	pinRepo.On("ListByEntityTypeVersionIDs", mock.Anything, []string{"etv-1"}).Return([]*models.CatalogVersionPin{
		{ID: "pin-1", CatalogVersionID: "cv-1", EntityTypeVersionID: "etv-1"},
	}, nil)

	err := svc.DeleteTypeDefinition(context.Background(), "td-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "in use")
}

// T-29.15: Delete type def used by attribute in latest entity type version (not pinned) → blocked
func TestT29_15_DeleteTypeDefUsedByLatestVersion(t *testing.T) {
	svc, tdRepo, tdvRepo, attrRepo, pinRepo, etvRepo := newTypeDefSvcWithPins()

	td := &models.TypeDefinition{ID: "td-1", Name: "guardrailID", BaseType: models.BaseTypeString}
	tdRepo.On("GetByID", mock.Anything, "td-1").Return(td, nil)

	tdvRepo.On("ListByTypeDefinition", mock.Anything, "td-1").Return([]*models.TypeDefinitionVersion{
		{ID: "tdv-1", TypeDefinitionID: "td-1", VersionNumber: 1},
	}, nil)

	// Attribute in etv-2 references tdv-1
	attrRepo.On("ListByTypeDefinitionVersionIDs", mock.Anything, []string{"tdv-1"}).Return([]*models.Attribute{
		{ID: "attr-1", Name: "guard_id", EntityTypeVersionID: "etv-2", TypeDefinitionVersionID: "tdv-1"},
	}, nil)

	// etv-2 is NOT pinned
	pinRepo.On("ListByEntityTypeVersionIDs", mock.Anything, []string{"etv-2"}).Return([]*models.CatalogVersionPin{}, nil)

	// But etv-2 IS the latest version of its entity type
	etvRepo.On("GetByID", mock.Anything, "etv-2").Return(&models.EntityTypeVersion{
		ID: "etv-2", EntityTypeID: "et-1", Version: 2,
	}, nil)
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-1").Return(&models.EntityTypeVersion{
		ID: "etv-2", EntityTypeID: "et-1", Version: 2,
	}, nil)

	err := svc.DeleteTypeDefinition(context.Background(), "td-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "in use")
}

// T-29.16: Delete type def used only by old non-pinned, non-latest version → allowed
func TestT29_16_DeleteTypeDefUsedOnlyByOldVersion(t *testing.T) {
	svc, tdRepo, tdvRepo, attrRepo, pinRepo, etvRepo := newTypeDefSvcWithPins()

	td := &models.TypeDefinition{ID: "td-1", Name: "guardrailID", BaseType: models.BaseTypeString}
	tdRepo.On("GetByID", mock.Anything, "td-1").Return(td, nil)

	tdvRepo.On("ListByTypeDefinition", mock.Anything, "td-1").Return([]*models.TypeDefinitionVersion{
		{ID: "tdv-1", TypeDefinitionID: "td-1", VersionNumber: 1},
	}, nil)

	// Attribute in etv-1 references tdv-1
	attrRepo.On("ListByTypeDefinitionVersionIDs", mock.Anything, []string{"tdv-1"}).Return([]*models.Attribute{
		{ID: "attr-1", Name: "guard_id", EntityTypeVersionID: "etv-1", TypeDefinitionVersionID: "tdv-1"},
	}, nil)

	// etv-1 is NOT pinned
	pinRepo.On("ListByEntityTypeVersionIDs", mock.Anything, []string{"etv-1"}).Return([]*models.CatalogVersionPin{}, nil)

	// etv-1 is NOT the latest version (etv-3 is latest)
	etvRepo.On("GetByID", mock.Anything, "etv-1").Return(&models.EntityTypeVersion{
		ID: "etv-1", EntityTypeID: "et-1", Version: 1,
	}, nil)
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-1").Return(&models.EntityTypeVersion{
		ID: "etv-3", EntityTypeID: "et-1", Version: 3,
	}, nil)

	tdRepo.On("Delete", mock.Anything, "td-1").Return(nil)

	err := svc.DeleteTypeDefinition(context.Background(), "td-1")
	assert.NoError(t, err)
}

// T-29.17: Delete type def not used by any attribute → allowed
func TestT29_17_DeleteTypeDefNotUsed(t *testing.T) {
	svc, tdRepo, tdvRepo, attrRepo, _, _ := newTypeDefSvcWithPins()

	td := &models.TypeDefinition{ID: "td-1", Name: "guardrailID", BaseType: models.BaseTypeString}
	tdRepo.On("GetByID", mock.Anything, "td-1").Return(td, nil)

	tdvRepo.On("ListByTypeDefinition", mock.Anything, "td-1").Return([]*models.TypeDefinitionVersion{
		{ID: "tdv-1", TypeDefinitionID: "td-1", VersionNumber: 1},
	}, nil)

	// No attributes reference this type def
	attrRepo.On("ListByTypeDefinitionVersionIDs", mock.Anything, []string{"tdv-1"}).Return([]*models.Attribute{}, nil)

	tdRepo.On("Delete", mock.Anything, "td-1").Return(nil)

	err := svc.DeleteTypeDefinition(context.Background(), "td-1")
	assert.NoError(t, err)
}

// Error path: ListByTypeDefinition fails in checkTypeDefInUse
func TestDeleteTypeDefinition_ListVersionsError(t *testing.T) {
	svc, tdRepo, tdvRepo, _, _, _ := newTypeDefSvcWithPins()

	td := &models.TypeDefinition{ID: "td-1", Name: "guardrailID", BaseType: models.BaseTypeString}
	tdRepo.On("GetByID", mock.Anything, "td-1").Return(td, nil)
	tdvRepo.On("ListByTypeDefinition", mock.Anything, "td-1").Return(nil, errors.New("db error"))

	err := svc.DeleteTypeDefinition(context.Background(), "td-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

// Error path: ListByTypeDefinitionVersionIDs fails
func TestDeleteTypeDefinition_ListAttrsError(t *testing.T) {
	svc, tdRepo, tdvRepo, attrRepo, _, _ := newTypeDefSvcWithPins()

	td := &models.TypeDefinition{ID: "td-1", Name: "guardrailID", BaseType: models.BaseTypeString}
	tdRepo.On("GetByID", mock.Anything, "td-1").Return(td, nil)
	tdvRepo.On("ListByTypeDefinition", mock.Anything, "td-1").Return([]*models.TypeDefinitionVersion{
		{ID: "tdv-1", TypeDefinitionID: "td-1", VersionNumber: 1},
	}, nil)
	attrRepo.On("ListByTypeDefinitionVersionIDs", mock.Anything, []string{"tdv-1"}).Return(nil, errors.New("attr query error"))

	err := svc.DeleteTypeDefinition(context.Background(), "td-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "attr query error")
}

// Error path: ListByEntityTypeVersionIDs fails
func TestDeleteTypeDefinition_ListPinsError(t *testing.T) {
	svc, tdRepo, tdvRepo, attrRepo, pinRepo, _ := newTypeDefSvcWithPins()

	td := &models.TypeDefinition{ID: "td-1", Name: "guardrailID", BaseType: models.BaseTypeString}
	tdRepo.On("GetByID", mock.Anything, "td-1").Return(td, nil)
	tdvRepo.On("ListByTypeDefinition", mock.Anything, "td-1").Return([]*models.TypeDefinitionVersion{
		{ID: "tdv-1", TypeDefinitionID: "td-1", VersionNumber: 1},
	}, nil)
	attrRepo.On("ListByTypeDefinitionVersionIDs", mock.Anything, []string{"tdv-1"}).Return([]*models.Attribute{
		{ID: "attr-1", EntityTypeVersionID: "etv-1", TypeDefinitionVersionID: "tdv-1"},
	}, nil)
	pinRepo.On("ListByEntityTypeVersionIDs", mock.Anything, mock.AnythingOfType("[]string")).Return(nil, errors.New("pin query error"))

	err := svc.DeleteTypeDefinition(context.Background(), "td-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pin query error")
}

// Coverage: checkTypeDefInUse with no versions returns nil (L251 early return)
func TestDeleteTypeDefinition_NoVersions_SkipsInUseCheck(t *testing.T) {
	svc, tdRepo, tdvRepo, _, _, _ := newTypeDefSvcWithPins()

	td := &models.TypeDefinition{ID: "td-1", Name: "guardrailID", BaseType: models.BaseTypeString}
	tdRepo.On("GetByID", mock.Anything, "td-1").Return(td, nil)
	tdvRepo.On("ListByTypeDefinition", mock.Anything, "td-1").Return([]*models.TypeDefinitionVersion{}, nil)
	tdRepo.On("Delete", mock.Anything, "td-1").Return(nil)

	err := svc.DeleteTypeDefinition(context.Background(), "td-1")
	assert.NoError(t, err)
	tdRepo.AssertCalled(t, "Delete", mock.Anything, "td-1")
}
