package meta_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository/mocks"
	"github.com/project-catalyst/pc-asset-hub/internal/service/meta"
)

// === Enum Service Tests (T-3.29 through T-3.34) ===

func TestT3_29_CreateEnum(t *testing.T) {
	enumRepo := new(mocks.MockEnumRepo)
	evRepo := new(mocks.MockEnumValueRepo)
	svc := meta.NewEnumService(enumRepo, evRepo, nil)

	enumRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.Enum")).Return(nil)
	evRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EnumValue")).Return(nil)

	e, err := svc.CreateEnum(context.Background(), "Status", "Deploy status", []string{"active", "inactive"})
	require.NoError(t, err)
	assert.NotEmpty(t, e.ID)
	assert.Equal(t, "Status", e.Name)
	assert.Equal(t, "Deploy status", e.Description)
}

func TestT3_30_UpdateEnumAddValue(t *testing.T) {
	enumRepo := new(mocks.MockEnumRepo)
	evRepo := new(mocks.MockEnumValueRepo)
	svc := meta.NewEnumService(enumRepo, evRepo, nil)

	evRepo.On("ListByEnum", mock.Anything, "enum1").Return([]*models.EnumValue{
		{ID: "v1", Value: "active", Ordinal: 0},
	}, nil)
	evRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.EnumValue")).Return(nil)

	err := svc.AddValue(context.Background(), "enum1", "inactive")
	assert.NoError(t, err)
}

func TestT3_31_UpdateEnumRemoveValue(t *testing.T) {
	enumRepo := new(mocks.MockEnumRepo)
	evRepo := new(mocks.MockEnumValueRepo)
	svc := meta.NewEnumService(enumRepo, evRepo, nil)

	evRepo.On("Delete", mock.Anything, "v1").Return(nil)

	err := svc.RemoveValue(context.Background(), "v1")
	assert.NoError(t, err)
}

func TestT3_32_DeleteEnumNoReferences(t *testing.T) {
	enumRepo := new(mocks.MockEnumRepo)
	svc := meta.NewEnumService(enumRepo, nil, nil)

	enumRepo.On("Delete", mock.Anything, "enum1").Return(nil)

	err := svc.DeleteEnum(context.Background(), "enum1")
	assert.NoError(t, err)
}

func TestT3_33_DeleteEnumWithReferences(t *testing.T) {
	enumRepo := new(mocks.MockEnumRepo)
	svc := meta.NewEnumService(enumRepo, nil, nil)

	enumRepo.On("Delete", mock.Anything, "enum1").Return(domainerrors.NewReferencedEnum("Status", []string{"Model.status"}))

	err := svc.DeleteEnum(context.Background(), "enum1")
	assert.True(t, domainerrors.IsReferencedEnum(err))
}

func TestT3_34_GetReferencingAttributes(t *testing.T) {
	svc := meta.NewEnumService(nil, nil, nil)
	refs, err := svc.GetReferencingAttributes(context.Background(), "enum1")
	require.NoError(t, err)
	assert.NotEmpty(t, refs)
}

// === Catalog Version Service Tests (T-3.35 through T-3.47) ===

func TestT3_35_CreateCatalogVersion(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	ltRepo := new(mocks.MockLifecycleTransitionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, ltRepo, nil, "", nil, nil, nil, nil)

	cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
	ltRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.LifecycleTransition")).Return(nil)

	cv, err := svc.CreateCatalogVersion(context.Background(), "v1.0", "Initial version", nil)
	require.NoError(t, err)
	assert.Equal(t, models.LifecycleStageDevelopment, cv.LifecycleStage)
	assert.Equal(t, "Initial version", cv.Description)
}

// TD-2: Duplicate catalog version label returns conflict error
func TestTD2_DuplicateCatalogVersionLabel(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	ltRepo := new(mocks.MockLifecycleTransitionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, ltRepo, nil, "", nil, nil, nil, nil)

	// First create succeeds
	cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil).Once()
	ltRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.LifecycleTransition")).Return(nil).Once()

	_, err := svc.CreateCatalogVersion(context.Background(), "v1.0", "First", nil)
	require.NoError(t, err)

	// Second create with same label — repo returns conflict (DB unique constraint)
	cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(
		domainerrors.NewConflict("CatalogVersion", "version_label already exists: v1.0"),
	).Once()

	_, err = svc.CreateCatalogVersion(context.Background(), "v1.0", "Duplicate", nil)
	require.Error(t, err)
	assert.True(t, domainerrors.IsConflict(err), "expected conflict error for duplicate label")
}

func TestT3_36_CreateCatalogVersionWithPins(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	ltRepo := new(mocks.MockLifecycleTransitionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, ltRepo, nil, "", nil, nil, nil, nil)

	cvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)
	ltRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	pins := []models.CatalogVersionPin{{EntityTypeVersionID: "etv1"}}
	cv, err := svc.CreateCatalogVersion(context.Background(), "v1.0", "", pins)
	require.NoError(t, err)
	assert.NotEmpty(t, cv.ID)
	pinRepo.AssertCalled(t, "Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin"))
}

func TestT3_37_PromoteDevToTestAsRW(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	ltRepo := new(mocks.MockLifecycleTransitionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, ltRepo, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	cvRepo.On("UpdateLifecycle", mock.Anything, "cv1", models.LifecycleStageTesting).Return(nil)
	ltRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	_, err := svc.Promote(context.Background(), "cv1", meta.RoleRW, "user1")
	assert.NoError(t, err)
}

func TestT3_38_PromoteDevToTestAsRO(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)

	_, err := svc.Promote(context.Background(), "cv1", meta.RoleRO, "user1")
	assert.True(t, domainerrors.IsForbidden(err))
}

func TestT3_39_DemoteTestToDevAsRW(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	ltRepo := new(mocks.MockLifecycleTransitionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, ltRepo, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageTesting,
	}, nil)
	cvRepo.On("UpdateLifecycle", mock.Anything, "cv1", models.LifecycleStageDevelopment).Return(nil)
	ltRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	err := svc.Demote(context.Background(), "cv1", meta.RoleRW, "user1", models.LifecycleStageDevelopment)
	assert.NoError(t, err)
}

func TestT3_40_PromoteTestToProdAsAdmin(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	ltRepo := new(mocks.MockLifecycleTransitionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, ltRepo, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageTesting,
	}, nil)
	cvRepo.On("UpdateLifecycle", mock.Anything, "cv1", models.LifecycleStageProduction).Return(nil)
	ltRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	_, err := svc.Promote(context.Background(), "cv1", meta.RoleAdmin, "admin1")
	assert.NoError(t, err)
}

func TestT3_41_PromoteTestToProdAsRW(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageTesting,
	}, nil)

	_, err := svc.Promote(context.Background(), "cv1", meta.RoleRW, "user1")
	assert.True(t, domainerrors.IsForbidden(err))
}

func TestT3_42_DemoteProdToTestAsSuperAdmin(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	ltRepo := new(mocks.MockLifecycleTransitionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, ltRepo, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageProduction,
	}, nil)
	cvRepo.On("UpdateLifecycle", mock.Anything, "cv1", models.LifecycleStageTesting).Return(nil)
	ltRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	err := svc.Demote(context.Background(), "cv1", meta.RoleSuperAdmin, "superadmin1", models.LifecycleStageTesting)
	assert.NoError(t, err)
}

func TestT3_43_DemoteProdToTestAsAdmin(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageProduction,
	}, nil)

	err := svc.Demote(context.Background(), "cv1", meta.RoleAdmin, "admin1", models.LifecycleStageTesting)
	assert.True(t, domainerrors.IsForbidden(err))
}

func TestT3_44_PromoteDevToProdDirectly(t *testing.T) {
	// The promote method goes one step at a time: dev->test, test->prod.
	// Trying to promote from dev should go to test, not prod.
	cvRepo := new(mocks.MockCatalogVersionRepo)
	ltRepo := new(mocks.MockLifecycleTransitionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, ltRepo, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	cvRepo.On("UpdateLifecycle", mock.Anything, "cv1", models.LifecycleStageTesting).Return(nil)
	ltRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	_, err := svc.Promote(context.Background(), "cv1", meta.RoleAdmin, "admin1")
	assert.NoError(t, err)
	// It should have gone to testing, not production
	cvRepo.AssertCalled(t, "UpdateLifecycle", mock.Anything, "cv1", models.LifecycleStageTesting)
	cvRepo.AssertNotCalled(t, "UpdateLifecycle", mock.Anything, "cv1", models.LifecycleStageProduction)
}

func TestT3_45_TransitionsRecorded(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	ltRepo := new(mocks.MockLifecycleTransitionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, ltRepo, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	cvRepo.On("UpdateLifecycle", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	ltRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.LifecycleTransition")).Return(nil)

	_, err := svc.Promote(context.Background(), "cv1", meta.RoleRW, "user1")
	assert.NoError(t, err)
	ltRepo.AssertCalled(t, "Create", mock.Anything, mock.AnythingOfType("*models.LifecycleTransition"))
}

func TestT3_46_ModifyProductionAsSuperAdmin(t *testing.T) {
	// Super Admin can modify production — this is about the lifecycle service
	// allowing Super Admin to do things that others can't.
	// The Demote method with SuperAdmin role succeeds for production.
	cvRepo := new(mocks.MockCatalogVersionRepo)
	ltRepo := new(mocks.MockLifecycleTransitionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, ltRepo, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageProduction,
	}, nil)
	cvRepo.On("UpdateLifecycle", mock.Anything, "cv1", models.LifecycleStageDevelopment).Return(nil)
	ltRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	err := svc.Demote(context.Background(), "cv1", meta.RoleSuperAdmin, "superadmin", models.LifecycleStageDevelopment)
	assert.NoError(t, err)
}

func TestT3_47_ModifyProductionAsAdmin(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageProduction,
	}, nil)

	err := svc.Demote(context.Background(), "cv1", meta.RoleAdmin, "admin", models.LifecycleStageTesting)
	assert.True(t, domainerrors.IsForbidden(err))
}

// === DeleteCatalogVersion Tests ===

func TestDeleteCatalogVersion_AdminDeletesDev(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	cvRepo.On("Delete", mock.Anything, "cv1").Return(nil)

	err := svc.DeleteCatalogVersion(context.Background(), "cv1", meta.RoleAdmin)
	assert.NoError(t, err)
}

func TestDeleteCatalogVersion_AdminDeletesTesting(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageTesting,
	}, nil)
	cvRepo.On("Delete", mock.Anything, "cv1").Return(nil)

	err := svc.DeleteCatalogVersion(context.Background(), "cv1", meta.RoleAdmin)
	assert.NoError(t, err)
}

func TestDeleteCatalogVersion_AdminCannotDeleteProduction(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageProduction,
	}, nil)

	err := svc.DeleteCatalogVersion(context.Background(), "cv1", meta.RoleAdmin)
	assert.True(t, domainerrors.IsForbidden(err))
}

func TestDeleteCatalogVersion_SuperAdminDeletesProduction(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageProduction,
	}, nil)
	cvRepo.On("Delete", mock.Anything, "cv1").Return(nil)

	err := svc.DeleteCatalogVersion(context.Background(), "cv1", meta.RoleSuperAdmin)
	assert.NoError(t, err)
}

func TestDeleteCatalogVersion_RWForbidden(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)

	err := svc.DeleteCatalogVersion(context.Background(), "cv1", meta.RoleRW)
	assert.True(t, domainerrors.IsForbidden(err))
}

func TestDeleteCatalogVersion_ROForbidden(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)

	err := svc.DeleteCatalogVersion(context.Background(), "cv1", meta.RoleRO)
	assert.True(t, domainerrors.IsForbidden(err))
}

func TestDeleteCatalogVersion_NotFound(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "bad").Return(nil, domainerrors.NewNotFound("CatalogVersion", "bad"))

	err := svc.DeleteCatalogVersion(context.Background(), "bad", meta.RoleAdmin)
	assert.True(t, domainerrors.IsNotFound(err))
}

// === ListPins and ListTransitions Tests (T-E.22 through T-E.24) ===

func TestTE22_ListPinsResolvedNames(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, etRepo, etvRepo, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "etv1").Return(&models.EntityTypeVersion{
		ID: "etv1", EntityTypeID: "et1", Version: 3,
	}, nil)
	etRepo.On("GetByID", mock.Anything, "et1").Return(&models.EntityType{
		ID: "et1", Name: "Model",
	}, nil)

	pins, err := svc.ListPins(context.Background(), "cv1")
	require.NoError(t, err)
	require.Len(t, pins, 1)
	assert.Equal(t, "Model", pins[0].EntityTypeName)
	assert.Equal(t, "et1", pins[0].EntityTypeID)
	assert.Equal(t, "etv1", pins[0].EntityTypeVersionID)
	assert.Equal(t, 3, pins[0].Version)
}

func TestTE23_ListPinsEmpty(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return([]*models.CatalogVersionPin{}, nil)

	pins, err := svc.ListPins(context.Background(), "cv1")
	require.NoError(t, err)
	assert.Empty(t, pins)
}

// === ListPins and ListTransitions Error Path Tests ===

func TestListPins_PinRepoError(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return(([]*models.CatalogVersionPin)(nil), domainerrors.NewNotFound("Pin", "cv1"))

	_, err := svc.ListPins(context.Background(), "cv1")
	assert.True(t, domainerrors.IsNotFound(err))
}

func TestListPins_EtvGetByIDError(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, nil, etvRepo, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "etv1").Return(nil, domainerrors.NewNotFound("EntityTypeVersion", "etv1"))

	_, err := svc.ListPins(context.Background(), "cv1")
	assert.True(t, domainerrors.IsNotFound(err))
}

func TestListPins_EtGetByIDError(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, etRepo, etvRepo, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "etv1").Return(&models.EntityTypeVersion{
		ID: "etv1", EntityTypeID: "et1", Version: 3,
	}, nil)
	etRepo.On("GetByID", mock.Anything, "et1").Return(nil, domainerrors.NewNotFound("EntityType", "et1"))

	_, err := svc.ListPins(context.Background(), "cv1")
	assert.True(t, domainerrors.IsNotFound(err))
}

func TestListTransitions_LtRepoError(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	ltRepo := new(mocks.MockLifecycleTransitionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, ltRepo, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageTesting,
	}, nil)
	ltRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return(([]*models.LifecycleTransition)(nil), domainerrors.NewNotFound("LifecycleTransition", "cv1"))

	_, err := svc.ListTransitions(context.Background(), "cv1")
	assert.True(t, domainerrors.IsNotFound(err))
}

func TestListTransitions_NotFoundCV(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	ltRepo := new(mocks.MockLifecycleTransitionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, ltRepo, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "bad").Return(nil, domainerrors.NewNotFound("CatalogVersion", "bad"))

	_, err := svc.ListTransitions(context.Background(), "bad")
	assert.True(t, domainerrors.IsNotFound(err))
}

func TestTE24_ListTransitionsOrdered(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	ltRepo := new(mocks.MockLifecycleTransitionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, ltRepo, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageTesting,
	}, nil)
	ltRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return([]*models.LifecycleTransition{
		{ID: "lt1", CatalogVersionID: "cv1", ToStage: "development", PerformedBy: "system"},
		{ID: "lt2", CatalogVersionID: "cv1", FromStage: "development", ToStage: "testing", PerformedBy: "admin"},
	}, nil)

	transitions, err := svc.ListTransitions(context.Background(), "cv1")
	require.NoError(t, err)
	require.Len(t, transitions, 2)
	assert.Equal(t, "development", transitions[0].ToStage)
	assert.Equal(t, "testing", transitions[1].ToStage)
}

// === UpdateCatalogVersion Tests ===

func TestUpdateCatalogVersion_UpdateLabel(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", VersionLabel: "v1.0", Description: "old desc",
	}, nil)
	cvRepo.On("GetByLabel", mock.Anything, "v2.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v2.0"))
	cvRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)

	newLabel := "v2.0"
	cv, err := svc.UpdateCatalogVersion(context.Background(), "cv1", &newLabel, nil, meta.RoleRW)
	require.NoError(t, err)
	assert.Equal(t, "v2.0", cv.VersionLabel)
	assert.Equal(t, "old desc", cv.Description) // unchanged
}

func TestUpdateCatalogVersion_UpdateDescription(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", VersionLabel: "v1.0", Description: "old desc",
	}, nil)
	cvRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)

	newDesc := "new desc"
	cv, err := svc.UpdateCatalogVersion(context.Background(), "cv1", nil, &newDesc, meta.RoleRW)
	require.NoError(t, err)
	assert.Equal(t, "new desc", cv.Description)
	assert.Equal(t, "v1.0", cv.VersionLabel) // unchanged
}

func TestUpdateCatalogVersion_UpdateBoth(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", VersionLabel: "v1.0", Description: "old desc",
	}, nil)
	cvRepo.On("GetByLabel", mock.Anything, "v2.0").Return(nil, domainerrors.NewNotFound("CatalogVersion", "v2.0"))
	cvRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)

	newLabel := "v2.0"
	newDesc := "new desc"
	cv, err := svc.UpdateCatalogVersion(context.Background(), "cv1", &newLabel, &newDesc, meta.RoleRW)
	require.NoError(t, err)
	assert.Equal(t, "v2.0", cv.VersionLabel)
	assert.Equal(t, "new desc", cv.Description)
}

func TestUpdateCatalogVersion_NotFound(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "bad").Return(nil, domainerrors.NewNotFound("CatalogVersion", "bad"))

	_, err := svc.UpdateCatalogVersion(context.Background(), "bad", nil, nil, meta.RoleRW)
	assert.True(t, domainerrors.IsNotFound(err))
}

func TestUpdateCatalogVersion_DuplicateLabel(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", VersionLabel: "v1.0",
	}, nil)
	cvRepo.On("GetByLabel", mock.Anything, "v2.0").Return(&models.CatalogVersion{
		ID: "cv2", VersionLabel: "v2.0",
	}, nil)

	newLabel := "v2.0"
	_, err := svc.UpdateCatalogVersion(context.Background(), "cv1", &newLabel, nil, meta.RoleRW)
	assert.True(t, domainerrors.IsConflict(err))
}

func TestUpdateCatalogVersion_GetByLabelDBError(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", VersionLabel: "v1.0",
	}, nil)
	dbErr := fmt.Errorf("connection refused")
	cvRepo.On("GetByLabel", mock.Anything, "v-new").Return(nil, dbErr)

	newLabel := "v-new"
	_, err := svc.UpdateCatalogVersion(context.Background(), "cv1", &newLabel, nil, meta.RoleRW)
	// Should propagate the DB error, not silently swallow it
	assert.ErrorIs(t, err, dbErr)
}

func TestUpdateCatalogVersion_SameLabel(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", VersionLabel: "v1.0", Description: "desc",
	}, nil)
	// GetByLabel returns the SAME CV — should be allowed (no-op rename)
	cvRepo.On("GetByLabel", mock.Anything, "v1.0").Return(&models.CatalogVersion{
		ID: "cv1", VersionLabel: "v1.0",
	}, nil)

	// No actual change, so no Update call needed
	newLabel := "v1.0"
	cv, err := svc.UpdateCatalogVersion(context.Background(), "cv1", &newLabel, nil, meta.RoleRW)
	require.NoError(t, err)
	assert.Equal(t, "v1.0", cv.VersionLabel)
}

func TestUpdateCatalogVersion_NeitherChanged(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", VersionLabel: "v1.0", Description: "desc",
	}, nil)

	cv, err := svc.UpdateCatalogVersion(context.Background(), "cv1", nil, nil, meta.RoleRW)
	require.NoError(t, err)
	assert.Equal(t, "v1.0", cv.VersionLabel)
	// Update should NOT have been called
	cvRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestUpdateCatalogVersion_UpdateRepoError(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", VersionLabel: "v1.0",
	}, nil)
	cvRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(domainerrors.NewValidation("db error"))

	newDesc := "new"
	_, err := svc.UpdateCatalogVersion(context.Background(), "cv1", nil, &newDesc, meta.RoleRW)
	assert.Error(t, err)
}

// === UpdateCatalogVersion Stage Guard Tests (TD-71) ===

func TestUpdateCatalogVersion_ProductionBlocked_SuperAdmin(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", VersionLabel: "v1.0", LifecycleStage: models.LifecycleStageProduction,
	}, nil)

	newDesc := "updated"
	_, err := svc.UpdateCatalogVersion(context.Background(), "cv1", nil, &newDesc, meta.RoleSuperAdmin)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "metadata editing")
	assert.Contains(t, err.Error(), "production")
}

func TestUpdateCatalogVersion_ProductionBlocked_RW(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", VersionLabel: "v1.0", LifecycleStage: models.LifecycleStageProduction,
	}, nil)

	newLabel := "v2.0"
	_, err := svc.UpdateCatalogVersion(context.Background(), "cv1", &newLabel, nil, meta.RoleRW)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "metadata editing")
}

func TestUpdateCatalogVersion_TestingBlocked_RW(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", VersionLabel: "v1.0", LifecycleStage: models.LifecycleStageTesting,
	}, nil)

	newDesc := "updated"
	_, err := svc.UpdateCatalogVersion(context.Background(), "cv1", nil, &newDesc, meta.RoleRW)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "metadata")
	assert.Contains(t, err.Error(), "SuperAdmin")
}

func TestUpdateCatalogVersion_TestingBlocked_Admin(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", VersionLabel: "v1.0", LifecycleStage: models.LifecycleStageTesting,
	}, nil)

	newDesc := "updated"
	_, err := svc.UpdateCatalogVersion(context.Background(), "cv1", nil, &newDesc, meta.RoleAdmin)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SuperAdmin")
}

func TestUpdateCatalogVersion_TestingAllowed_SuperAdmin(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", VersionLabel: "v1.0", Description: "old", LifecycleStage: models.LifecycleStageTesting,
	}, nil)
	cvRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)

	newDesc := "updated"
	cv, err := svc.UpdateCatalogVersion(context.Background(), "cv1", nil, &newDesc, meta.RoleSuperAdmin)
	require.NoError(t, err)
	assert.Equal(t, "updated", cv.Description)
}

func TestUpdateCatalogVersion_DevelopmentAllowed_RW(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", VersionLabel: "v1.0", Description: "old", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	cvRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)

	newDesc := "updated"
	cv, err := svc.UpdateCatalogVersion(context.Background(), "cv1", nil, &newDesc, meta.RoleRW)
	require.NoError(t, err)
	assert.Equal(t, "updated", cv.Description)
}

// === AddPin Tests ===

func TestAddPin_Success(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, nil, etvRepo, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "etv1").Return(&models.EntityTypeVersion{
		ID: "etv1", EntityTypeID: "et1", Version: 1,
	}, nil)
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return([]*models.CatalogVersionPin{}, nil)
	pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)

	pin, err := svc.AddPin(context.Background(), "cv1", "etv1", meta.RoleAdmin)
	require.NoError(t, err)
	assert.NotEmpty(t, pin.ID)
	assert.Equal(t, "cv1", pin.CatalogVersionID)
	assert.Equal(t, "etv1", pin.EntityTypeVersionID)
}

func TestAddPin_CVNotFound(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "bad").Return(nil, domainerrors.NewNotFound("CatalogVersion", "bad"))

	_, err := svc.AddPin(context.Background(), "bad", "etv1", meta.RoleAdmin)
	assert.True(t, domainerrors.IsNotFound(err))
}

func TestAddPin_ETVNotFound(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, etvRepo, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "bad-etv").Return(nil, domainerrors.NewNotFound("EntityTypeVersion", "bad-etv"))

	_, err := svc.AddPin(context.Background(), "cv1", "bad-etv", meta.RoleAdmin)
	assert.True(t, domainerrors.IsNotFound(err))
}

func TestAddPin_DuplicatePin(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, nil, etvRepo, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "etv1").Return(&models.EntityTypeVersion{
		ID: "etv1", EntityTypeID: "et1", Version: 1,
	}, nil)
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
	}, nil)

	_, err := svc.AddPin(context.Background(), "cv1", "etv1", meta.RoleAdmin)
	assert.True(t, domainerrors.IsConflict(err))
}

// T-28.01: AddPin with same entity type (different version) returns 409
func TestAddPin_DuplicateEntityType(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, nil, etvRepo, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	// Requesting to add V2 of entity type "et1"
	etvRepo.On("GetByID", mock.Anything, "etv1-v2").Return(&models.EntityTypeVersion{
		ID: "etv1-v2", EntityTypeID: "et1", Version: 2,
	}, nil)
	// V1 of the same entity type "et1" is already pinned
	etvRepo.On("GetByID", mock.Anything, "etv1-v1").Return(&models.EntityTypeVersion{
		ID: "etv1-v1", EntityTypeID: "et1", Version: 1,
	}, nil)
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1-v1"},
	}, nil)

	_, err := svc.AddPin(context.Background(), "cv1", "etv1-v2", meta.RoleAdmin)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsConflict(err))
	assert.Contains(t, err.Error(), "entity type")
}

func TestAddPin_CreateError(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, nil, etvRepo, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "etv1").Return(&models.EntityTypeVersion{
		ID: "etv1", EntityTypeID: "et1", Version: 1,
	}, nil)
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return([]*models.CatalogVersionPin{}, nil)
	pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(domainerrors.NewValidation("db error"))

	_, err := svc.AddPin(context.Background(), "cv1", "etv1", meta.RoleAdmin)
	assert.Error(t, err)
}

func TestAddPin_ListError(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, nil, etvRepo, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "etv1").Return(&models.EntityTypeVersion{
		ID: "etv1", EntityTypeID: "et1", Version: 1,
	}, nil)
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return(([]*models.CatalogVersionPin)(nil), domainerrors.NewValidation("db error"))

	_, err := svc.AddPin(context.Background(), "cv1", "etv1", meta.RoleAdmin)
	assert.Error(t, err)
}

// === RemovePin Tests ===

func TestRemovePin_Success(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	pinRepo.On("GetByID", mock.Anything, "pin1").Return(&models.CatalogVersionPin{
		ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1",
	}, nil)
	pinRepo.On("Delete", mock.Anything, "pin1").Return(nil)

	err := svc.RemovePin(context.Background(), "cv1", "pin1", meta.RoleAdmin)
	require.NoError(t, err)
}

func TestRemovePin_CVNotFound(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "bad").Return(nil, domainerrors.NewNotFound("CatalogVersion", "bad"))

	err := svc.RemovePin(context.Background(), "bad", "pin1", meta.RoleAdmin)
	assert.True(t, domainerrors.IsNotFound(err))
}

func TestRemovePin_PinNotFound(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	pinRepo.On("GetByID", mock.Anything, "bad-pin").Return(nil, domainerrors.NewNotFound("CatalogVersionPin", "bad-pin"))

	err := svc.RemovePin(context.Background(), "cv1", "bad-pin", meta.RoleAdmin)
	assert.True(t, domainerrors.IsNotFound(err))
}

func TestRemovePin_PinBelongsToDifferentCV(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	pinRepo.On("GetByID", mock.Anything, "pin1").Return(&models.CatalogVersionPin{
		ID: "pin1", CatalogVersionID: "cv2", EntityTypeVersionID: "etv1",
	}, nil)

	err := svc.RemovePin(context.Background(), "cv1", "pin1", meta.RoleAdmin)
	assert.True(t, domainerrors.IsNotFound(err))
}

func TestRemovePin_DeleteError(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	pinRepo.On("GetByID", mock.Anything, "pin1").Return(&models.CatalogVersionPin{
		ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1",
	}, nil)
	pinRepo.On("Delete", mock.Anything, "pin1").Return(domainerrors.NewValidation("db error"))

	err := svc.RemovePin(context.Background(), "cv1", "pin1", meta.RoleAdmin)
	assert.Error(t, err)
}

// === UpdatePin Tests ===

// T-28.04: UpdatePin changes ETV on existing pin
func TestUpdatePin_Success(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, nil, etvRepo, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	pinRepo.On("GetByID", mock.Anything, "pin1").Return(&models.CatalogVersionPin{
		ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1-v1",
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "etv1-v1").Return(&models.EntityTypeVersion{
		ID: "etv1-v1", EntityTypeID: "et1", Version: 1,
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "etv1-v2").Return(&models.EntityTypeVersion{
		ID: "etv1-v2", EntityTypeID: "et1", Version: 2,
	}, nil)
	pinRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)

	pin, err := svc.UpdatePin(context.Background(), "cv1", "pin1", "etv1-v2", meta.RoleAdmin)
	require.NoError(t, err)
	assert.Equal(t, "etv1-v2", pin.EntityTypeVersionID)
}

// UpdatePin with same version is a valid no-op
func TestUpdatePin_SameVersion(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, nil, etvRepo, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	pinRepo.On("GetByID", mock.Anything, "pin1").Return(&models.CatalogVersionPin{
		ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1-v1",
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "etv1-v1").Return(&models.EntityTypeVersion{
		ID: "etv1-v1", EntityTypeID: "et1", Version: 1,
	}, nil)
	pinRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)

	pin, err := svc.UpdatePin(context.Background(), "cv1", "pin1", "etv1-v1", meta.RoleAdmin)
	require.NoError(t, err)
	assert.Equal(t, "etv1-v1", pin.EntityTypeVersionID)
}

// T-28.05: UpdatePin with ETV from different entity type returns 400
func TestUpdatePin_EntityTypeMismatch(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, nil, etvRepo, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	pinRepo.On("GetByID", mock.Anything, "pin1").Return(&models.CatalogVersionPin{
		ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1-v1",
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "etv1-v1").Return(&models.EntityTypeVersion{
		ID: "etv1-v1", EntityTypeID: "et1", Version: 1,
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "etv2-v1").Return(&models.EntityTypeVersion{
		ID: "etv2-v1", EntityTypeID: "et2", Version: 1,
	}, nil)

	_, err := svc.UpdatePin(context.Background(), "cv1", "pin1", "etv2-v1", meta.RoleAdmin)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
	assert.Contains(t, err.Error(), "entity type mismatch")
}

// T-28.06: UpdatePin with nonexistent pin returns 404
func TestUpdatePin_PinNotFound(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	pinRepo.On("GetByID", mock.Anything, "bad").Return(nil, domainerrors.NewNotFound("CatalogVersionPin", "bad"))

	_, err := svc.UpdatePin(context.Background(), "cv1", "bad", "etv1", meta.RoleAdmin)
	assert.True(t, domainerrors.IsNotFound(err))
}

// T-28.07: UpdatePin with nonexistent ETV returns 404
func TestUpdatePin_ETVNotFound(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, nil, etvRepo, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	pinRepo.On("GetByID", mock.Anything, "pin1").Return(&models.CatalogVersionPin{
		ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1-v1",
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "etv1-v1").Return(&models.EntityTypeVersion{
		ID: "etv1-v1", EntityTypeID: "et1", Version: 1,
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "bad-etv").Return(nil, domainerrors.NewNotFound("EntityTypeVersion", "bad-etv"))

	_, err := svc.UpdatePin(context.Background(), "cv1", "pin1", "bad-etv", meta.RoleAdmin)
	assert.True(t, domainerrors.IsNotFound(err))
}

// UpdatePin: CV not found
func TestUpdatePin_CVNotFound(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "bad").Return(nil, domainerrors.NewNotFound("CatalogVersion", "bad"))

	_, err := svc.UpdatePin(context.Background(), "bad", "pin1", "etv1", meta.RoleAdmin)
	assert.True(t, domainerrors.IsNotFound(err))
}

// UpdatePin: current ETV lookup error
func TestUpdatePin_CurrentETVError(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, nil, etvRepo, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	pinRepo.On("GetByID", mock.Anything, "pin1").Return(&models.CatalogVersionPin{
		ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv-bad",
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "etv-bad").Return(nil, domainerrors.NewNotFound("EntityTypeVersion", "etv-bad"))

	_, err := svc.UpdatePin(context.Background(), "cv1", "pin1", "etv1-v2", meta.RoleAdmin)
	assert.True(t, domainerrors.IsNotFound(err))
}

// UpdatePin: pinRepo.Update error
func TestUpdatePin_UpdateError(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, nil, etvRepo, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	pinRepo.On("GetByID", mock.Anything, "pin1").Return(&models.CatalogVersionPin{
		ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1-v1",
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "etv1-v1").Return(&models.EntityTypeVersion{
		ID: "etv1-v1", EntityTypeID: "et1", Version: 1,
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "etv1-v2").Return(&models.EntityTypeVersion{
		ID: "etv1-v2", EntityTypeID: "et1", Version: 2,
	}, nil)
	pinRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(domainerrors.NewValidation("db error"))

	_, err := svc.UpdatePin(context.Background(), "cv1", "pin1", "etv1-v2", meta.RoleAdmin)
	assert.Error(t, err)
}

// AddPin: existing pin's ETV lookup error
func TestAddPin_ExistingPinETVError(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, nil, etvRepo, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "etv-new").Return(&models.EntityTypeVersion{
		ID: "etv-new", EntityTypeID: "et1", Version: 2,
	}, nil)
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv-bad"},
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "etv-bad").Return(nil, domainerrors.NewNotFound("EntityTypeVersion", "etv-bad"))

	_, err := svc.AddPin(context.Background(), "cv1", "etv-new", meta.RoleAdmin)
	assert.Error(t, err)
}

// T-28.08: UpdatePin verifies pin belongs to specified CV
func TestUpdatePin_PinBelongsToDifferentCV(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	pinRepo.On("GetByID", mock.Anything, "pin-other").Return(&models.CatalogVersionPin{
		ID: "pin-other", CatalogVersionID: "cv-other", EntityTypeVersionID: "etv1",
	}, nil)

	_, err := svc.UpdatePin(context.Background(), "cv1", "pin-other", "etv1-v2", meta.RoleAdmin)
	assert.True(t, domainerrors.IsNotFound(err))
}

// === TD-69: Pin editing stage guards ===

func TestTD69_AddPin_ProductionBlocked(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageProduction,
	}, nil)

	_, err := svc.AddPin(context.Background(), "cv1", "etv1", meta.RoleAdmin)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "production")
}

func TestTD69_AddPin_ProductionBlockedSuperAdmin(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageProduction,
	}, nil)

	_, err := svc.AddPin(context.Background(), "cv1", "etv1", meta.RoleSuperAdmin)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "production")
}

func TestTD69_AddPin_TestingRWBlocked(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageTesting,
	}, nil)

	_, err := svc.AddPin(context.Background(), "cv1", "etv1", meta.RoleRW)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SuperAdmin")
}

func TestTD69_AddPin_TestingAdminBlocked(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageTesting,
	}, nil)

	_, err := svc.AddPin(context.Background(), "cv1", "etv1", meta.RoleAdmin)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SuperAdmin")
}

func TestTD69_AddPin_TestingSuperAdminAllowed(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, nil, etvRepo, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageTesting,
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "etv1").Return(&models.EntityTypeVersion{
		ID: "etv1", EntityTypeID: "et1", Version: 1,
	}, nil)
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return([]*models.CatalogVersionPin{}, nil)
	pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)

	pin, err := svc.AddPin(context.Background(), "cv1", "etv1", meta.RoleSuperAdmin)
	require.NoError(t, err)
	assert.NotEmpty(t, pin.ID)
}

func TestTD69_RemovePin_ProductionBlocked(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageProduction,
	}, nil)

	err := svc.RemovePin(context.Background(), "cv1", "pin1", meta.RoleAdmin)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "production")
}

func TestTD69_UpdatePin_TestingRWBlocked(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageTesting,
	}, nil)

	_, err := svc.UpdatePin(context.Background(), "cv1", "pin1", "etv1", meta.RoleRW)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SuperAdmin")
}
