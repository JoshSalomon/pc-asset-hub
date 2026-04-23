package meta_test

import (
	"context"
	"errors"
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
	etvRepo.On("GetByIDs", mock.Anything, []string{"etv1"}).Return([]*models.EntityTypeVersion{
		{ID: "etv1", EntityTypeID: "et1", Version: 1},
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
	// V1 of the same entity type "et1" is already pinned — batch fetch returns it
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1-v1"},
	}, nil)
	etvRepo.On("GetByIDs", mock.Anything, []string{"etv1-v1"}).Return([]*models.EntityTypeVersion{
		{ID: "etv1-v1", EntityTypeID: "et1", Version: 1},
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

	result, err := svc.UpdatePin(context.Background(), "cv1", "pin1", "etv1-v2", meta.RoleAdmin, false)
	require.NoError(t, err)
	assert.Equal(t, "etv1-v2", result.Pin.EntityTypeVersionID)
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

	result, err := svc.UpdatePin(context.Background(), "cv1", "pin1", "etv1-v1", meta.RoleAdmin, false)
	require.NoError(t, err)
	assert.Equal(t, "etv1-v1", result.Pin.EntityTypeVersionID)
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

	_, err := svc.UpdatePin(context.Background(), "cv1", "pin1", "etv2-v1", meta.RoleAdmin, false)
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

	_, err := svc.UpdatePin(context.Background(), "cv1", "bad", "etv1", meta.RoleAdmin, false)
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

	_, err := svc.UpdatePin(context.Background(), "cv1", "pin1", "bad-etv", meta.RoleAdmin, false)
	assert.True(t, domainerrors.IsNotFound(err))
}

// UpdatePin: CV not found
func TestUpdatePin_CVNotFound(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "bad").Return(nil, domainerrors.NewNotFound("CatalogVersion", "bad"))

	_, err := svc.UpdatePin(context.Background(), "bad", "pin1", "etv1", meta.RoleAdmin, false)
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

	_, err := svc.UpdatePin(context.Background(), "cv1", "pin1", "etv1-v2", meta.RoleAdmin, false)
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

	_, err := svc.UpdatePin(context.Background(), "cv1", "pin1", "etv1-v2", meta.RoleAdmin, false)
	assert.Error(t, err)
}

// AddPin: existing pin's ETV batch lookup error
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
	etvRepo.On("GetByIDs", mock.Anything, []string{"etv-bad"}).Return(([]*models.EntityTypeVersion)(nil), domainerrors.NewNotFound("EntityTypeVersion", "etv-bad"))

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

	_, err := svc.UpdatePin(context.Background(), "cv1", "pin-other", "etv1-v2", meta.RoleAdmin, false)
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

	_, err := svc.UpdatePin(context.Background(), "cv1", "pin1", "etv1", meta.RoleRW, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SuperAdmin")
}

// TD-77: AddPin uses batch GetByIDs instead of N+1 GetByID calls for existing pins
func TestTD77_AddPin_BatchFetchExistingPins(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, nil, etvRepo, nil)

	// Set up 5 existing pins, each for a different entity type
	existingPins := make([]*models.CatalogVersionPin, 5)
	existingETVIDs := make([]string, 5)
	batchResult := make([]*models.EntityTypeVersion, 5)
	for i := 0; i < 5; i++ {
		etvID := fmt.Sprintf("etv-existing-%d", i)
		existingPins[i] = &models.CatalogVersionPin{
			ID: fmt.Sprintf("pin-%d", i), CatalogVersionID: "cv1", EntityTypeVersionID: etvID,
		}
		existingETVIDs[i] = etvID
		batchResult[i] = &models.EntityTypeVersion{
			ID: etvID, EntityTypeID: fmt.Sprintf("et-%d", i), Version: 1,
		}
	}

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	// GetByID should only be called once — for the NEW entity type version
	etvRepo.On("GetByID", mock.Anything, "etv-new").Return(&models.EntityTypeVersion{
		ID: "etv-new", EntityTypeID: "et-new", Version: 1,
	}, nil)
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return(existingPins, nil)
	// GetByIDs should be called once for all existing pin ETVs
	etvRepo.On("GetByIDs", mock.Anything, mock.AnythingOfType("[]string")).Return(batchResult, nil)
	pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)

	pin, err := svc.AddPin(context.Background(), "cv1", "etv-new", meta.RoleAdmin)
	require.NoError(t, err)
	assert.NotEmpty(t, pin.ID)

	// Verify GetByID was NOT called for existing pins (only once for the new ETV)
	etvRepo.AssertNumberOfCalls(t, "GetByID", 1)
	// Verify GetByIDs was called exactly once
	etvRepo.AssertNumberOfCalls(t, "GetByIDs", 1)
}

// TD-77: AddPin duplicate detection still works with batch fetch
func TestTD77_AddPin_BatchFetchDuplicateDetected(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, nil, etvRepo, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	// New ETV belongs to entity type "et1"
	etvRepo.On("GetByID", mock.Anything, "etv-new").Return(&models.EntityTypeVersion{
		ID: "etv-new", EntityTypeID: "et1", Version: 2,
	}, nil)
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv-existing"},
	}, nil)
	// Existing pin's ETV also belongs to "et1" — should trigger conflict
	etvRepo.On("GetByIDs", mock.Anything, mock.AnythingOfType("[]string")).Return([]*models.EntityTypeVersion{
		{ID: "etv-existing", EntityTypeID: "et1", Version: 1},
	}, nil)

	_, err := svc.AddPin(context.Background(), "cv1", "etv-new", meta.RoleAdmin)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsConflict(err))
}

// TD-77: AddPin with no existing pins skips GetByIDs call
func TestTD77_AddPin_NoPinsSkipsBatchFetch(t *testing.T) {
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

	// GetByIDs should NOT be called when there are no existing pins
	etvRepo.AssertNotCalled(t, "GetByIDs")
}

// TD-77: AddPin handles GetByIDs error gracefully
func TestTD77_AddPin_BatchFetchError(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, nil, etvRepo, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "etv-new").Return(&models.EntityTypeVersion{
		ID: "etv-new", EntityTypeID: "et1", Version: 1,
	}, nil)
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv-existing"},
	}, nil)
	etvRepo.On("GetByIDs", mock.Anything, mock.AnythingOfType("[]string")).Return(([]*models.EntityTypeVersion)(nil), fmt.Errorf("db error"))

	_, err := svc.AddPin(context.Background(), "cv1", "etv-new", meta.RoleAdmin)
	assert.Error(t, err)
}

// AddPin with orphaned pin (GetByIDs returns fewer results than requested) should error
func TestAddPin_OrphanedPinDetected(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, nil, etvRepo, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	// New ETV to add — entity type "et-new"
	etvRepo.On("GetByID", mock.Anything, "etv-new").Return(&models.EntityTypeVersion{
		ID: "etv-new", EntityTypeID: "et-new", Version: 1,
	}, nil)
	// CV has two existing pins, but one points to a deleted ETV
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv-good"},
		{ID: "pin2", CatalogVersionID: "cv1", EntityTypeVersionID: "etv-orphaned"},
	}, nil)
	// GetByIDs returns only 1 of the 2 requested — etv-orphaned is missing
	etvRepo.On("GetByIDs", mock.Anything, mock.AnythingOfType("[]string")).Return([]*models.EntityTypeVersion{
		{ID: "etv-good", EntityTypeID: "et-other", Version: 1},
	}, nil)
	// Mock Create since without the fix, AddPin would proceed to create
	pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)

	_, err := svc.AddPin(context.Background(), "cv1", "etv-new", meta.RoleAdmin)
	assert.Error(t, err, "should detect orphaned pin when GetByIDs returns fewer results than requested")
	assert.Contains(t, err.Error(), "orphaned pin references a deleted entity type version")
}

// === TD-10: Pin changes reset catalog validation status ===

// T-29.26: AddPin resets validation status to draft for all catalogs pinned to the CV
func TestT29_26_AddPinResetsValidationStatus(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	catalogRepo := new(mocks.MockCatalogRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, nil, etvRepo, catalogRepo)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "etv1").Return(&models.EntityTypeVersion{
		ID: "etv1", EntityTypeID: "et1", Version: 1,
	}, nil)
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return([]*models.CatalogVersionPin{}, nil)
	pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)

	// Two catalogs pinned to this CV — one valid, one invalid
	catalogRepo.On("ListByCatalogVersionID", mock.Anything, "cv1").Return([]*models.Catalog{
		{ID: "cat1", Name: "cat-a", CatalogVersionID: "cv1", ValidationStatus: models.ValidationStatusValid},
		{ID: "cat2", Name: "cat-b", CatalogVersionID: "cv1", ValidationStatus: models.ValidationStatusInvalid},
	}, nil)
	catalogRepo.On("UpdateValidationStatus", mock.Anything, "cat1", models.ValidationStatusDraft).Return(nil)
	catalogRepo.On("UpdateValidationStatus", mock.Anything, "cat2", models.ValidationStatusDraft).Return(nil)

	_, err := svc.AddPin(context.Background(), "cv1", "etv1", meta.RoleAdmin)
	require.NoError(t, err)
	catalogRepo.AssertCalled(t, "UpdateValidationStatus", mock.Anything, "cat1", models.ValidationStatusDraft)
	catalogRepo.AssertCalled(t, "UpdateValidationStatus", mock.Anything, "cat2", models.ValidationStatusDraft)
}

// T-29.27: UpdatePin resets validation status to draft
func TestT29_27_UpdatePinResetsValidationStatus(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	catalogRepo := new(mocks.MockCatalogRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, nil, etvRepo, catalogRepo)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	pinRepo.On("GetByID", mock.Anything, "pin1").Return(&models.CatalogVersionPin{
		ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1",
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "etv1").Return(&models.EntityTypeVersion{
		ID: "etv1", EntityTypeID: "et1", Version: 1,
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "etv2").Return(&models.EntityTypeVersion{
		ID: "etv2", EntityTypeID: "et1", Version: 2,
	}, nil)
	pinRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)

	catalogRepo.On("ListByCatalogVersionID", mock.Anything, "cv1").Return([]*models.Catalog{
		{ID: "cat1", Name: "cat-a", CatalogVersionID: "cv1", ValidationStatus: models.ValidationStatusValid},
	}, nil)
	catalogRepo.On("UpdateValidationStatus", mock.Anything, "cat1", models.ValidationStatusDraft).Return(nil)

	_, err := svc.UpdatePin(context.Background(), "cv1", "pin1", "etv2", meta.RoleAdmin, false)
	require.NoError(t, err)
	catalogRepo.AssertCalled(t, "UpdateValidationStatus", mock.Anything, "cat1", models.ValidationStatusDraft)
}

// T-29.28: RemovePin resets validation status to draft
func TestT29_28_RemovePinResetsValidationStatus(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	catalogRepo := new(mocks.MockCatalogRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, nil, etvRepo, catalogRepo)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	pinRepo.On("GetByID", mock.Anything, "pin1").Return(&models.CatalogVersionPin{
		ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1",
	}, nil)
	pinRepo.On("Delete", mock.Anything, "pin1").Return(nil)

	catalogRepo.On("ListByCatalogVersionID", mock.Anything, "cv1").Return([]*models.Catalog{
		{ID: "cat1", Name: "cat-a", CatalogVersionID: "cv1", ValidationStatus: models.ValidationStatusValid},
	}, nil)
	catalogRepo.On("UpdateValidationStatus", mock.Anything, "cat1", models.ValidationStatusDraft).Return(nil)

	err := svc.RemovePin(context.Background(), "cv1", "pin1", meta.RoleAdmin)
	require.NoError(t, err)
	catalogRepo.AssertCalled(t, "UpdateValidationStatus", mock.Anything, "cat1", models.ValidationStatusDraft)
}

// T-29.28b: RemovePin skips already-draft catalogs when resetting validation status
func TestT29_28b_ResetDependentCatalogs_SkipAlreadyDraft(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	catalogRepo := new(mocks.MockCatalogRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, nil, etvRepo, catalogRepo)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	pinRepo.On("GetByID", mock.Anything, "pin1").Return(&models.CatalogVersionPin{
		ID: "pin1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1",
	}, nil)
	pinRepo.On("Delete", mock.Anything, "pin1").Return(nil)

	// cat1 is already draft — should be skipped; cat2 is valid — should be reset
	catalogRepo.On("ListByCatalogVersionID", mock.Anything, "cv1").Return([]*models.Catalog{
		{ID: "cat1", Name: "cat-draft", CatalogVersionID: "cv1", ValidationStatus: models.ValidationStatusDraft},
		{ID: "cat2", Name: "cat-valid", CatalogVersionID: "cv1", ValidationStatus: models.ValidationStatusValid},
	}, nil)
	catalogRepo.On("UpdateValidationStatus", mock.Anything, "cat2", models.ValidationStatusDraft).Return(nil)

	err := svc.RemovePin(context.Background(), "cv1", "pin1", meta.RoleAdmin)
	require.NoError(t, err)

	// cat1 should NOT have UpdateValidationStatus called (already draft)
	catalogRepo.AssertNotCalled(t, "UpdateValidationStatus", mock.Anything, "cat1", mock.Anything)
	// cat2 should have it called
	catalogRepo.AssertCalled(t, "UpdateValidationStatus", mock.Anything, "cat2", models.ValidationStatusDraft)
}

// T-29.29: Pin change on CV with no dependent catalogs → no error
func TestT29_29_PinChangeNoCatalogs(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	catalogRepo := new(mocks.MockCatalogRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, nil, etvRepo, catalogRepo)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "etv1").Return(&models.EntityTypeVersion{
		ID: "etv1", EntityTypeID: "et1", Version: 1,
	}, nil)
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return([]*models.CatalogVersionPin{}, nil)
	pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)

	// No catalogs depend on this CV
	catalogRepo.On("ListByCatalogVersionID", mock.Anything, "cv1").Return([]*models.Catalog{}, nil)

	_, err := svc.AddPin(context.Background(), "cv1", "etv1", meta.RoleAdmin)
	require.NoError(t, err)
	catalogRepo.AssertNotCalled(t, "UpdateValidationStatus")
}

// Error path: ListByCatalogVersionID fails during resetDependentCatalogs
func TestT29_ResetDependentCatalogs_ListError(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	catalogRepo := new(mocks.MockCatalogRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, nil, etvRepo, catalogRepo)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "etv1").Return(&models.EntityTypeVersion{
		ID: "etv1", EntityTypeID: "et1", Version: 1,
	}, nil)
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return([]*models.CatalogVersionPin{}, nil)
	pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)

	catalogRepo.On("ListByCatalogVersionID", mock.Anything, "cv1").Return(nil, errors.New("list catalogs error"))

	_, err := svc.AddPin(context.Background(), "cv1", "etv1", meta.RoleAdmin)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "list catalogs error")
}

// Error path: UpdateValidationStatus fails during resetDependentCatalogs
func TestT29_ResetDependentCatalogs_UpdateError(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	catalogRepo := new(mocks.MockCatalogRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, nil, etvRepo, catalogRepo)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "etv1").Return(&models.EntityTypeVersion{
		ID: "etv1", EntityTypeID: "et1", Version: 1,
	}, nil)
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return([]*models.CatalogVersionPin{}, nil)
	pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)

	catalogRepo.On("ListByCatalogVersionID", mock.Anything, "cv1").Return([]*models.Catalog{
		{ID: "cat1", Name: "cat-a", CatalogVersionID: "cv1", ValidationStatus: models.ValidationStatusValid},
	}, nil)
	catalogRepo.On("UpdateValidationStatus", mock.Anything, "cat1", models.ValidationStatusDraft).Return(errors.New("update status error"))

	_, err := svc.AddPin(context.Background(), "cv1", "etv1", meta.RoleAdmin)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update status error")
}

// === TD-114: Instance Attribute Migration on Pin Version Change ===

// migrationTestSetup creates repos + service with migration repos configured.
func migrationTestSetup() (
	*mocks.MockCatalogVersionRepo,
	*mocks.MockCatalogVersionPinRepo,
	*mocks.MockEntityTypeVersionRepo,
	*mocks.MockCatalogRepo,
	*mocks.MockAttributeRepo,
	*mocks.MockEntityInstanceRepo,
	*mocks.MockInstanceAttributeValueRepo,
	*meta.CatalogVersionService,
) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	catalogRepo := new(mocks.MockCatalogRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	instRepo := new(mocks.MockEntityInstanceRepo)
	iavRepo := new(mocks.MockInstanceAttributeValueRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, nil, etvRepo, catalogRepo,
		meta.WithMigrationRepos(attrRepo, instRepo, iavRepo))
	return cvRepo, pinRepo, etvRepo, catalogRepo, attrRepo, instRepo, iavRepo, svc
}

// setupBasicMigrationMocks configures the common mocks for UpdatePin migration tests.
func setupBasicMigrationMocks(cvRepo *mocks.MockCatalogVersionRepo, pinRepo *mocks.MockCatalogVersionPinRepo, etvRepo *mocks.MockEntityTypeVersionRepo) {
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
}

// T-29.38: Match attributes by name — same name in V1 and V2 → mapped
func TestT29_38_MatchAttributesByName(t *testing.T) {
	cvRepo, pinRepo, etvRepo, catalogRepo, attrRepo, instRepo, iavRepo, svc := migrationTestSetup()
	setupBasicMigrationMocks(cvRepo, pinRepo, etvRepo)

	attrRepo.On("ListByVersion", mock.Anything, "etv1-v1").Return([]*models.Attribute{
		{ID: "old-a", Name: "endpoint", Ordinal: 0, TypeDefinitionVersionID: "tdv-str"},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv1-v2").Return([]*models.Attribute{
		{ID: "new-a", Name: "endpoint", Ordinal: 0, TypeDefinitionVersionID: "tdv-str"},
	}, nil)
	catalogRepo.On("ListByCatalogVersionID", mock.Anything, "cv1").Return([]*models.Catalog{
		{ID: "cat1", CatalogVersionID: "cv1", ValidationStatus: models.ValidationStatusValid},
	}, nil)
	catalogRepo.On("UpdateValidationStatus", mock.Anything, "cat1", models.ValidationStatusDraft).Return(nil)
	instRepo.On("List", mock.Anything, "et1", "cat1", mock.Anything).Return([]*models.EntityInstance{
		{ID: "inst1"},
	}, 1, nil)
	iavRepo.On("RemapAttributeIDs", mock.Anything, []string{"inst1"}, map[string]string{"old-a": "new-a"}).Return(1, nil)

	result, err := svc.UpdatePin(context.Background(), "cv1", "pin1", "etv1-v2", meta.RoleAdmin, false)
	require.NoError(t, err)
	require.NotNil(t, result.Migration)
	require.Len(t, result.Migration.AttributeMappings, 1)
	assert.Equal(t, "endpoint", result.Migration.AttributeMappings[0].OldName)
	assert.Equal(t, "endpoint", result.Migration.AttributeMappings[0].NewName)
	assert.Equal(t, "remap", result.Migration.AttributeMappings[0].Action)
	iavRepo.AssertCalled(t, "RemapAttributeIDs", mock.Anything, []string{"inst1"}, map[string]string{"old-a": "new-a"})
}

// T-29.39: Attribute in V1 not in V2 → orphaned warning
func TestT29_39_DeletedAttribute(t *testing.T) {
	cvRepo, pinRepo, etvRepo, catalogRepo, attrRepo, instRepo, _, svc := migrationTestSetup()
	setupBasicMigrationMocks(cvRepo, pinRepo, etvRepo)

	attrRepo.On("ListByVersion", mock.Anything, "etv1-v1").Return([]*models.Attribute{
		{ID: "old-a", Name: "old_field", Ordinal: 0, TypeDefinitionVersionID: "tdv-str"},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv1-v2").Return([]*models.Attribute{}, nil)
	catalogRepo.On("ListByCatalogVersionID", mock.Anything, "cv1").Return([]*models.Catalog{
		{ID: "cat1", CatalogVersionID: "cv1", ValidationStatus: models.ValidationStatusValid},
	}, nil)
	catalogRepo.On("UpdateValidationStatus", mock.Anything, "cat1", models.ValidationStatusDraft).Return(nil)
	instRepo.On("List", mock.Anything, "et1", "cat1", mock.Anything).Return([]*models.EntityInstance{
		{ID: "inst1"}, {ID: "inst2"},
	}, 2, nil)

	result, err := svc.UpdatePin(context.Background(), "cv1", "pin1", "etv1-v2", meta.RoleAdmin, false)
	require.NoError(t, err)
	require.NotNil(t, result.Migration)

	require.Len(t, result.Migration.AttributeMappings, 1)
	assert.Equal(t, "orphaned", result.Migration.AttributeMappings[0].Action)
	assert.Equal(t, "old_field", result.Migration.AttributeMappings[0].OldName)

	var deleted *models.MigrationWarning
	for _, w := range result.Migration.Warnings {
		if w.Type == "deleted_attribute" {
			deleted = &w
			break
		}
	}
	require.NotNil(t, deleted)
	assert.Equal(t, "old_field", deleted.Attribute)
	assert.Equal(t, 2, deleted.AffectedInstances)
}

// T-29.40: Attribute in V2 not in V1 → no warning (new attribute, no data to migrate)
func TestT29_40_NewAttributeNoWarning(t *testing.T) {
	cvRepo, pinRepo, etvRepo, catalogRepo, attrRepo, _, _, svc := migrationTestSetup()
	setupBasicMigrationMocks(cvRepo, pinRepo, etvRepo)

	attrRepo.On("ListByVersion", mock.Anything, "etv1-v1").Return([]*models.Attribute{}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv1-v2").Return([]*models.Attribute{
		{ID: "new-a", Name: "new_field", Ordinal: 0, Required: false, TypeDefinitionVersionID: "tdv-str"},
	}, nil)
	catalogRepo.On("ListByCatalogVersionID", mock.Anything, "cv1").Return([]*models.Catalog{}, nil)
	catalogRepo.On("UpdateValidationStatus", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	result, err := svc.UpdatePin(context.Background(), "cv1", "pin1", "etv1-v2", meta.RoleAdmin, false)
	require.NoError(t, err)
	require.NotNil(t, result.Migration)
	assert.Empty(t, result.Migration.Warnings)
}

// T-29.41: Attribute in V2 not in V1 and required → new_required warning
func TestT29_41_NewRequiredAttribute(t *testing.T) {
	cvRepo, pinRepo, etvRepo, catalogRepo, attrRepo, instRepo, iavRepo, svc := migrationTestSetup()
	setupBasicMigrationMocks(cvRepo, pinRepo, etvRepo)

	attrRepo.On("ListByVersion", mock.Anything, "etv1-v1").Return([]*models.Attribute{
		{ID: "old-a", Name: "endpoint", Ordinal: 0, TypeDefinitionVersionID: "tdv-str"},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv1-v2").Return([]*models.Attribute{
		{ID: "new-a", Name: "endpoint", Ordinal: 0, TypeDefinitionVersionID: "tdv-str"},
		{ID: "new-b", Name: "region", Ordinal: 1, Required: true, TypeDefinitionVersionID: "tdv-str"},
	}, nil)
	catalogRepo.On("ListByCatalogVersionID", mock.Anything, "cv1").Return([]*models.Catalog{
		{ID: "cat1", CatalogVersionID: "cv1", ValidationStatus: models.ValidationStatusValid},
	}, nil)
	catalogRepo.On("UpdateValidationStatus", mock.Anything, "cat1", models.ValidationStatusDraft).Return(nil)
	instRepo.On("List", mock.Anything, "et1", "cat1", mock.Anything).Return([]*models.EntityInstance{
		{ID: "inst1"},
	}, 1, nil)
	iavRepo.On("RemapAttributeIDs", mock.Anything, mock.Anything, mock.Anything).Return(1, nil)

	result, err := svc.UpdatePin(context.Background(), "cv1", "pin1", "etv1-v2", meta.RoleAdmin, false)
	require.NoError(t, err)

	var newReq *models.MigrationWarning
	for _, w := range result.Migration.Warnings {
		if w.Type == "new_required" {
			newReq = &w
			break
		}
	}
	require.NotNil(t, newReq, "expected new_required warning")
	assert.Equal(t, "region", newReq.Attribute)
	assert.Equal(t, 1, newReq.AffectedInstances)
}

// T-29.42: Matched attribute changed type definition → type_changed warning
func TestT29_42_TypeChanged(t *testing.T) {
	cvRepo, pinRepo, etvRepo, catalogRepo, attrRepo, instRepo, iavRepo, svc := migrationTestSetup()
	setupBasicMigrationMocks(cvRepo, pinRepo, etvRepo)

	attrRepo.On("ListByVersion", mock.Anything, "etv1-v1").Return([]*models.Attribute{
		{ID: "old-a", Name: "port", Ordinal: 0, TypeDefinitionVersionID: "tdv-str"},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv1-v2").Return([]*models.Attribute{
		{ID: "new-a", Name: "port", Ordinal: 0, TypeDefinitionVersionID: "tdv-int"},
	}, nil)
	catalogRepo.On("ListByCatalogVersionID", mock.Anything, "cv1").Return([]*models.Catalog{
		{ID: "cat1", CatalogVersionID: "cv1", ValidationStatus: models.ValidationStatusValid},
	}, nil)
	catalogRepo.On("UpdateValidationStatus", mock.Anything, "cat1", models.ValidationStatusDraft).Return(nil)
	instRepo.On("List", mock.Anything, "et1", "cat1", mock.Anything).Return([]*models.EntityInstance{
		{ID: "inst1"}, {ID: "inst2"},
	}, 2, nil)
	iavRepo.On("RemapAttributeIDs", mock.Anything, mock.Anything, mock.Anything).Return(2, nil)

	result, err := svc.UpdatePin(context.Background(), "cv1", "pin1", "etv1-v2", meta.RoleAdmin, false)
	require.NoError(t, err)

	var typeChanged *models.MigrationWarning
	for _, w := range result.Migration.Warnings {
		if w.Type == "type_changed" {
			typeChanged = &w
			break
		}
	}
	require.NotNil(t, typeChanged, "expected type_changed warning")
	assert.Equal(t, "port", typeChanged.Attribute)
	assert.Equal(t, "tdv-str", typeChanged.OldType)
	assert.Equal(t, "tdv-int", typeChanged.NewType)
	assert.Equal(t, 2, typeChanged.AffectedInstances)
}

// T-29.43: Same ordinal position, different name → renamed warning
func TestT29_43_RenamedAttribute(t *testing.T) {
	cvRepo, pinRepo, etvRepo, catalogRepo, attrRepo, instRepo, iavRepo, svc := migrationTestSetup()
	setupBasicMigrationMocks(cvRepo, pinRepo, etvRepo)

	attrRepo.On("ListByVersion", mock.Anything, "etv1-v1").Return([]*models.Attribute{
		{ID: "old-a", Name: "hostname", Ordinal: 0, TypeDefinitionVersionID: "tdv-str"},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv1-v2").Return([]*models.Attribute{
		{ID: "new-a", Name: "host", Ordinal: 0, TypeDefinitionVersionID: "tdv-str"},
	}, nil)
	catalogRepo.On("ListByCatalogVersionID", mock.Anything, "cv1").Return([]*models.Catalog{
		{ID: "cat1", CatalogVersionID: "cv1", ValidationStatus: models.ValidationStatusValid},
	}, nil)
	catalogRepo.On("UpdateValidationStatus", mock.Anything, "cat1", models.ValidationStatusDraft).Return(nil)
	instRepo.On("List", mock.Anything, "et1", "cat1", mock.Anything).Return([]*models.EntityInstance{
		{ID: "inst1"},
	}, 1, nil)
	iavRepo.On("RemapAttributeIDs", mock.Anything, mock.Anything, mock.Anything).Return(1, nil)

	result, err := svc.UpdatePin(context.Background(), "cv1", "pin1", "etv1-v2", meta.RoleAdmin, false)
	require.NoError(t, err)

	var renamed *models.MigrationWarning
	for _, w := range result.Migration.Warnings {
		if w.Type == "renamed" {
			renamed = &w
			break
		}
	}
	require.NotNil(t, renamed, "expected renamed warning")
	assert.Equal(t, "host", renamed.Attribute)

	// Verify the mapping was created (old-a → new-a via ordinal match)
	var foundRemap bool
	for _, m := range result.Migration.AttributeMappings {
		if m.OldName == "hostname" && m.NewName == "host" && m.Action == "remap" {
			foundRemap = true
			break
		}
	}
	assert.True(t, foundRemap, "expected remap mapping for renamed attribute")
}

// T-29.44: Same name, same type → clean remap, no warning
func TestT29_44_CleanRemap(t *testing.T) {
	cvRepo, pinRepo, etvRepo, catalogRepo, attrRepo, instRepo, iavRepo, svc := migrationTestSetup()
	setupBasicMigrationMocks(cvRepo, pinRepo, etvRepo)

	attrRepo.On("ListByVersion", mock.Anything, "etv1-v1").Return([]*models.Attribute{
		{ID: "old-a", Name: "endpoint", Ordinal: 0, TypeDefinitionVersionID: "tdv-str"},
		{ID: "old-b", Name: "port", Ordinal: 1, TypeDefinitionVersionID: "tdv-int"},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv1-v2").Return([]*models.Attribute{
		{ID: "new-a", Name: "endpoint", Ordinal: 0, TypeDefinitionVersionID: "tdv-str"},
		{ID: "new-b", Name: "port", Ordinal: 1, TypeDefinitionVersionID: "tdv-int"},
	}, nil)
	catalogRepo.On("ListByCatalogVersionID", mock.Anything, "cv1").Return([]*models.Catalog{
		{ID: "cat1", CatalogVersionID: "cv1", ValidationStatus: models.ValidationStatusValid},
	}, nil)
	catalogRepo.On("UpdateValidationStatus", mock.Anything, "cat1", models.ValidationStatusDraft).Return(nil)
	instRepo.On("List", mock.Anything, "et1", "cat1", mock.Anything).Return([]*models.EntityInstance{
		{ID: "inst1"},
	}, 1, nil)
	iavRepo.On("RemapAttributeIDs", mock.Anything, mock.Anything, mock.Anything).Return(2, nil)

	result, err := svc.UpdatePin(context.Background(), "cv1", "pin1", "etv1-v2", meta.RoleAdmin, false)
	require.NoError(t, err)
	assert.Empty(t, result.Migration.Warnings, "no warnings expected for clean remap")
	assert.Len(t, result.Migration.AttributeMappings, 2)
	for _, m := range result.Migration.AttributeMappings {
		assert.Equal(t, "remap", m.Action)
	}
}

// T-29.45: IAVs with matched attributes have attribute_id updated to new ID
func TestT29_45_IAVRemapped(t *testing.T) {
	cvRepo, pinRepo, etvRepo, catalogRepo, attrRepo, instRepo, iavRepo, svc := migrationTestSetup()
	setupBasicMigrationMocks(cvRepo, pinRepo, etvRepo)

	attrRepo.On("ListByVersion", mock.Anything, "etv1-v1").Return([]*models.Attribute{
		{ID: "old-a", Name: "endpoint", Ordinal: 0, TypeDefinitionVersionID: "tdv-str"},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv1-v2").Return([]*models.Attribute{
		{ID: "new-a", Name: "endpoint", Ordinal: 0, TypeDefinitionVersionID: "tdv-str"},
	}, nil)
	catalogRepo.On("ListByCatalogVersionID", mock.Anything, "cv1").Return([]*models.Catalog{
		{ID: "cat1", CatalogVersionID: "cv1"},
	}, nil)
	catalogRepo.On("UpdateValidationStatus", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	instRepo.On("List", mock.Anything, "et1", "cat1", mock.Anything).Return([]*models.EntityInstance{
		{ID: "inst1"}, {ID: "inst2"},
	}, 2, nil)
	iavRepo.On("RemapAttributeIDs", mock.Anything, []string{"inst1", "inst2"}, map[string]string{"old-a": "new-a"}).Return(2, nil)

	_, err := svc.UpdatePin(context.Background(), "cv1", "pin1", "etv1-v2", meta.RoleAdmin, false)
	require.NoError(t, err)
	iavRepo.AssertCalled(t, "RemapAttributeIDs", mock.Anything, []string{"inst1", "inst2"}, map[string]string{"old-a": "new-a"})
}

// T-29.46: IAVs for orphaned attributes remain in DB (not deleted) — RemapAttributeIDs is not called for orphaned attrs
func TestT29_46_OrphanedIAVsNotDeleted(t *testing.T) {
	cvRepo, pinRepo, etvRepo, catalogRepo, attrRepo, _, _, svc := migrationTestSetup()
	setupBasicMigrationMocks(cvRepo, pinRepo, etvRepo)

	attrRepo.On("ListByVersion", mock.Anything, "etv1-v1").Return([]*models.Attribute{
		{ID: "old-a", Name: "deleted_field", Ordinal: 0, TypeDefinitionVersionID: "tdv-str"},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv1-v2").Return([]*models.Attribute{}, nil)
	catalogRepo.On("ListByCatalogVersionID", mock.Anything, "cv1").Return([]*models.Catalog{}, nil)
	catalogRepo.On("UpdateValidationStatus", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	result, err := svc.UpdatePin(context.Background(), "cv1", "pin1", "etv1-v2", meta.RoleAdmin, false)
	require.NoError(t, err)
	assert.Equal(t, "orphaned", result.Migration.AttributeMappings[0].Action)
	// No RemapAttributeIDs called — mapping is empty so no remap happens
}

// T-29.47: Migration scoped to catalogs pinned to this CV only
func TestT29_47_ScopedToCatalogVersion(t *testing.T) {
	cvRepo, pinRepo, etvRepo, catalogRepo, attrRepo, instRepo, iavRepo, svc := migrationTestSetup()
	setupBasicMigrationMocks(cvRepo, pinRepo, etvRepo)

	attrRepo.On("ListByVersion", mock.Anything, "etv1-v1").Return([]*models.Attribute{
		{ID: "old-a", Name: "endpoint", Ordinal: 0, TypeDefinitionVersionID: "tdv-str"},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv1-v2").Return([]*models.Attribute{
		{ID: "new-a", Name: "endpoint", Ordinal: 0, TypeDefinitionVersionID: "tdv-str"},
	}, nil)
	// Only cat1 is pinned to cv1; cat2 is pinned to a different CV
	catalogRepo.On("ListByCatalogVersionID", mock.Anything, "cv1").Return([]*models.Catalog{
		{ID: "cat1", CatalogVersionID: "cv1"},
	}, nil)
	catalogRepo.On("UpdateValidationStatus", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	instRepo.On("List", mock.Anything, "et1", "cat1", mock.Anything).Return([]*models.EntityInstance{
		{ID: "inst1"},
	}, 1, nil)
	iavRepo.On("RemapAttributeIDs", mock.Anything, []string{"inst1"}, mock.Anything).Return(1, nil)

	result, err := svc.UpdatePin(context.Background(), "cv1", "pin1", "etv1-v2", meta.RoleAdmin, false)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Migration.AffectedCatalogs)
	// instRepo.List was called only for cat1
	instRepo.AssertCalled(t, "List", mock.Anything, "et1", "cat1", mock.Anything)
	instRepo.AssertNotCalled(t, "List", mock.Anything, "et1", "cat2", mock.Anything)
}

// T-29.48: Migration scoped to instances of the affected entity type only
func TestT29_48_ScopedToEntityType(t *testing.T) {
	cvRepo, pinRepo, etvRepo, catalogRepo, attrRepo, instRepo, iavRepo, svc := migrationTestSetup()
	setupBasicMigrationMocks(cvRepo, pinRepo, etvRepo)

	attrRepo.On("ListByVersion", mock.Anything, "etv1-v1").Return([]*models.Attribute{
		{ID: "old-a", Name: "endpoint", Ordinal: 0, TypeDefinitionVersionID: "tdv-str"},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv1-v2").Return([]*models.Attribute{
		{ID: "new-a", Name: "endpoint", Ordinal: 0, TypeDefinitionVersionID: "tdv-str"},
	}, nil)
	catalogRepo.On("ListByCatalogVersionID", mock.Anything, "cv1").Return([]*models.Catalog{
		{ID: "cat1", CatalogVersionID: "cv1"},
	}, nil)
	catalogRepo.On("UpdateValidationStatus", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	// List is called with entity type "et1" — only instances of that type
	instRepo.On("List", mock.Anything, "et1", "cat1", mock.Anything).Return([]*models.EntityInstance{
		{ID: "inst1", EntityTypeID: "et1"},
	}, 1, nil)
	iavRepo.On("RemapAttributeIDs", mock.Anything, mock.Anything, mock.Anything).Return(1, nil)

	_, err := svc.UpdatePin(context.Background(), "cv1", "pin1", "etv1-v2", meta.RoleAdmin, false)
	require.NoError(t, err)
	instRepo.AssertCalled(t, "List", mock.Anything, "et1", "cat1", mock.Anything)
}

// T-29.49: Multiple catalogs pinned to same CV — all migrated
func TestT29_49_MultipleCatalogsMigrated(t *testing.T) {
	cvRepo, pinRepo, etvRepo, catalogRepo, attrRepo, instRepo, iavRepo, svc := migrationTestSetup()
	setupBasicMigrationMocks(cvRepo, pinRepo, etvRepo)

	attrRepo.On("ListByVersion", mock.Anything, "etv1-v1").Return([]*models.Attribute{
		{ID: "old-a", Name: "endpoint", Ordinal: 0, TypeDefinitionVersionID: "tdv-str"},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv1-v2").Return([]*models.Attribute{
		{ID: "new-a", Name: "endpoint", Ordinal: 0, TypeDefinitionVersionID: "tdv-str"},
	}, nil)
	catalogRepo.On("ListByCatalogVersionID", mock.Anything, "cv1").Return([]*models.Catalog{
		{ID: "cat1", CatalogVersionID: "cv1"},
		{ID: "cat2", CatalogVersionID: "cv1"},
	}, nil)
	catalogRepo.On("UpdateValidationStatus", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	instRepo.On("List", mock.Anything, "et1", "cat1", mock.Anything).Return([]*models.EntityInstance{
		{ID: "inst1"},
	}, 1, nil)
	instRepo.On("List", mock.Anything, "et1", "cat2", mock.Anything).Return([]*models.EntityInstance{
		{ID: "inst2"}, {ID: "inst3"},
	}, 2, nil)
	iavRepo.On("RemapAttributeIDs", mock.Anything, []string{"inst1", "inst2", "inst3"}, mock.Anything).Return(3, nil)

	result, err := svc.UpdatePin(context.Background(), "cv1", "pin1", "etv1-v2", meta.RoleAdmin, false)
	require.NoError(t, err)
	assert.Equal(t, 2, result.Migration.AffectedCatalogs)
	assert.Equal(t, 3, result.Migration.AffectedInstances)
}

// T-29.50: Instance with no attribute values — no error
func TestT29_50_InstanceNoValues(t *testing.T) {
	cvRepo, pinRepo, etvRepo, catalogRepo, attrRepo, instRepo, iavRepo, svc := migrationTestSetup()
	setupBasicMigrationMocks(cvRepo, pinRepo, etvRepo)

	attrRepo.On("ListByVersion", mock.Anything, "etv1-v1").Return([]*models.Attribute{
		{ID: "old-a", Name: "endpoint", Ordinal: 0, TypeDefinitionVersionID: "tdv-str"},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv1-v2").Return([]*models.Attribute{
		{ID: "new-a", Name: "endpoint", Ordinal: 0, TypeDefinitionVersionID: "tdv-str"},
	}, nil)
	catalogRepo.On("ListByCatalogVersionID", mock.Anything, "cv1").Return([]*models.Catalog{
		{ID: "cat1", CatalogVersionID: "cv1"},
	}, nil)
	catalogRepo.On("UpdateValidationStatus", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	instRepo.On("List", mock.Anything, "et1", "cat1", mock.Anything).Return([]*models.EntityInstance{
		{ID: "inst1"},
	}, 1, nil)
	iavRepo.On("RemapAttributeIDs", mock.Anything, mock.Anything, mock.Anything).Return(0, nil)

	result, err := svc.UpdatePin(context.Background(), "cv1", "pin1", "etv1-v2", meta.RoleAdmin, false)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Migration.AffectedInstances)
}

// T-29.51: Dry-run returns migration report
func TestT29_51_DryRunReturnsReport(t *testing.T) {
	cvRepo, pinRepo, etvRepo, catalogRepo, attrRepo, instRepo, _, svc := migrationTestSetup()
	setupBasicMigrationMocks(cvRepo, pinRepo, etvRepo)

	attrRepo.On("ListByVersion", mock.Anything, "etv1-v1").Return([]*models.Attribute{
		{ID: "old-a", Name: "endpoint", Ordinal: 0, TypeDefinitionVersionID: "tdv-str"},
		{ID: "old-b", Name: "removed", Ordinal: 1, TypeDefinitionVersionID: "tdv-str"},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv1-v2").Return([]*models.Attribute{
		{ID: "new-a", Name: "endpoint", Ordinal: 0, TypeDefinitionVersionID: "tdv-str"},
	}, nil)
	catalogRepo.On("ListByCatalogVersionID", mock.Anything, "cv1").Return([]*models.Catalog{
		{ID: "cat1", CatalogVersionID: "cv1"},
	}, nil)
	instRepo.On("List", mock.Anything, "et1", "cat1", mock.Anything).Return([]*models.EntityInstance{
		{ID: "inst1"},
	}, 1, nil)

	result, err := svc.UpdatePin(context.Background(), "cv1", "pin1", "etv1-v2", meta.RoleAdmin, true)
	require.NoError(t, err)
	require.NotNil(t, result.Migration)
	assert.Equal(t, 1, result.Migration.AffectedCatalogs)
	assert.Equal(t, 1, result.Migration.AffectedInstances)
	assert.Len(t, result.Migration.AttributeMappings, 2)
	assert.Len(t, result.Migration.Warnings, 1)
	assert.Equal(t, "deleted_attribute", result.Migration.Warnings[0].Type)
}

// T-29.52: Dry-run does not modify IAVs
func TestT29_52_DryRunNoModification(t *testing.T) {
	cvRepo, pinRepo, etvRepo, catalogRepo, attrRepo, instRepo, iavRepo, svc := migrationTestSetup()
	setupBasicMigrationMocks(cvRepo, pinRepo, etvRepo)

	attrRepo.On("ListByVersion", mock.Anything, "etv1-v1").Return([]*models.Attribute{
		{ID: "old-a", Name: "endpoint", Ordinal: 0, TypeDefinitionVersionID: "tdv-str"},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv1-v2").Return([]*models.Attribute{
		{ID: "new-a", Name: "endpoint", Ordinal: 0, TypeDefinitionVersionID: "tdv-str"},
	}, nil)
	catalogRepo.On("ListByCatalogVersionID", mock.Anything, "cv1").Return([]*models.Catalog{
		{ID: "cat1", CatalogVersionID: "cv1"},
	}, nil)
	instRepo.On("List", mock.Anything, "et1", "cat1", mock.Anything).Return([]*models.EntityInstance{
		{ID: "inst1"},
	}, 1, nil)

	_, err := svc.UpdatePin(context.Background(), "cv1", "pin1", "etv1-v2", meta.RoleAdmin, true)
	require.NoError(t, err)
	iavRepo.AssertNotCalled(t, "RemapAttributeIDs", mock.Anything, mock.Anything, mock.Anything)
}

// T-29.53: Dry-run does not change pin version
func TestT29_53_DryRunNoVersionChange(t *testing.T) {
	cvRepo, pinRepo, etvRepo, catalogRepo, attrRepo, _, _, svc := migrationTestSetup()
	setupBasicMigrationMocks(cvRepo, pinRepo, etvRepo)

	attrRepo.On("ListByVersion", mock.Anything, "etv1-v1").Return([]*models.Attribute{}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv1-v2").Return([]*models.Attribute{}, nil)
	catalogRepo.On("ListByCatalogVersionID", mock.Anything, "cv1").Return([]*models.Catalog{}, nil)

	result, err := svc.UpdatePin(context.Background(), "cv1", "pin1", "etv1-v2", meta.RoleAdmin, true)
	require.NoError(t, err)
	// Pin should still be at the old version since dry-run doesn't apply
	assert.Equal(t, "etv1-v1", result.Pin.EntityTypeVersionID)
	pinRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

// === TD-114: Migration Error Path Tests ===

func TestMigration_AttrRepoOldError(t *testing.T) {
	cvRepo, pinRepo, etvRepo, _, attrRepo, _, _, svc := migrationTestSetup()
	setupBasicMigrationMocks(cvRepo, pinRepo, etvRepo)
	attrRepo.On("ListByVersion", mock.Anything, "etv1-v1").Return(nil, errors.New("attr list error"))

	_, err := svc.UpdatePin(context.Background(), "cv1", "pin1", "etv1-v2", meta.RoleAdmin, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "attr list error")
}

func TestMigration_AttrRepoNewError(t *testing.T) {
	cvRepo, pinRepo, etvRepo, _, attrRepo, _, _, svc := migrationTestSetup()
	setupBasicMigrationMocks(cvRepo, pinRepo, etvRepo)
	attrRepo.On("ListByVersion", mock.Anything, "etv1-v1").Return([]*models.Attribute{}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv1-v2").Return(nil, errors.New("new attr error"))

	_, err := svc.UpdatePin(context.Background(), "cv1", "pin1", "etv1-v2", meta.RoleAdmin, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "new attr error")
}

func TestMigration_CatalogListError(t *testing.T) {
	cvRepo, pinRepo, etvRepo, catalogRepo, attrRepo, _, _, svc := migrationTestSetup()
	setupBasicMigrationMocks(cvRepo, pinRepo, etvRepo)
	attrRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Attribute{}, nil)
	catalogRepo.On("ListByCatalogVersionID", mock.Anything, "cv1").Return(nil, errors.New("catalog list error"))

	_, err := svc.UpdatePin(context.Background(), "cv1", "pin1", "etv1-v2", meta.RoleAdmin, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "catalog list error")
}

func TestMigration_InstanceListError(t *testing.T) {
	cvRepo, pinRepo, etvRepo, catalogRepo, attrRepo, instRepo, _, svc := migrationTestSetup()
	setupBasicMigrationMocks(cvRepo, pinRepo, etvRepo)
	attrRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Attribute{}, nil)
	catalogRepo.On("ListByCatalogVersionID", mock.Anything, "cv1").Return([]*models.Catalog{
		{ID: "cat1", CatalogVersionID: "cv1"},
	}, nil)
	instRepo.On("List", mock.Anything, "et1", "cat1", mock.Anything).Return(nil, 0, errors.New("inst list error"))

	_, err := svc.UpdatePin(context.Background(), "cv1", "pin1", "etv1-v2", meta.RoleAdmin, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "inst list error")
}

func TestMigration_RemapError(t *testing.T) {
	cvRepo, pinRepo, etvRepo, catalogRepo, attrRepo, instRepo, iavRepo, svc := migrationTestSetup()
	setupBasicMigrationMocks(cvRepo, pinRepo, etvRepo)
	attrRepo.On("ListByVersion", mock.Anything, "etv1-v1").Return([]*models.Attribute{
		{ID: "old-a", Name: "endpoint", Ordinal: 0, TypeDefinitionVersionID: "tdv-str"},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv1-v2").Return([]*models.Attribute{
		{ID: "new-a", Name: "endpoint", Ordinal: 0, TypeDefinitionVersionID: "tdv-str"},
	}, nil)
	catalogRepo.On("ListByCatalogVersionID", mock.Anything, "cv1").Return([]*models.Catalog{
		{ID: "cat1", CatalogVersionID: "cv1"},
	}, nil)
	instRepo.On("List", mock.Anything, "et1", "cat1", mock.Anything).Return([]*models.EntityInstance{
		{ID: "inst1"},
	}, 1, nil)
	iavRepo.On("RemapAttributeIDs", mock.Anything, mock.Anything, mock.Anything).Return(0, errors.New("remap error"))

	_, err := svc.UpdatePin(context.Background(), "cv1", "pin1", "etv1-v2", meta.RoleAdmin, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "remap error")
}

func TestMigration_ResetCatalogsError(t *testing.T) {
	cvRepo, pinRepo, etvRepo, catalogRepo, attrRepo, instRepo, iavRepo, svc := migrationTestSetup()
	setupBasicMigrationMocks(cvRepo, pinRepo, etvRepo)
	attrRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Attribute{}, nil)
	catalogRepo.On("ListByCatalogVersionID", mock.Anything, "cv1").Return([]*models.Catalog{
		{ID: "cat1", CatalogVersionID: "cv1", ValidationStatus: models.ValidationStatusValid},
	}, nil)
	catalogRepo.On("UpdateValidationStatus", mock.Anything, "cat1", models.ValidationStatusDraft).Return(errors.New("reset error"))
	instRepo.On("List", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return([]*models.EntityInstance{}, 0, nil)
	iavRepo.On("RemapAttributeIDs", mock.Anything, mock.Anything, mock.Anything).Return(0, nil)

	_, err := svc.UpdatePin(context.Background(), "cv1", "pin1", "etv1-v2", meta.RoleAdmin, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reset error")
}

// T-29.66: UpdatePin wraps mutations in a transaction when txManager is set
func TestT29_66_TransactionWrapping(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	catalogRepo := new(mocks.MockCatalogRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	instRepo := new(mocks.MockEntityInstanceRepo)
	iavRepo := new(mocks.MockInstanceAttributeValueRepo)
	txm := &mocks.MockTransactionManager{}
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, nil, etvRepo, catalogRepo,
		meta.WithMigrationRepos(attrRepo, instRepo, iavRepo),
		meta.WithTransactionManager(txm))

	setupBasicMigrationMocks(cvRepo, pinRepo, etvRepo)
	// Override pinRepo.Update to fail — simulates failure after remap
	pinRepo.ExpectedCalls = filterCalls(pinRepo.ExpectedCalls, "Update")
	pinRepo.On("Update", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(errors.New("pin update failed"))

	attrRepo.On("ListByVersion", mock.Anything, "etv1-v1").Return([]*models.Attribute{
		{ID: "old-a", Name: "attr-a", Ordinal: 0},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv1-v2").Return([]*models.Attribute{
		{ID: "new-a", Name: "attr-a", Ordinal: 0},
	}, nil)
	catalogRepo.On("ListByCatalogVersionID", mock.Anything, "cv1").Return([]*models.Catalog{
		{ID: "cat1", CatalogVersionID: "cv1"},
	}, nil)
	instRepo.On("List", mock.Anything, "et1", "cat1", mock.Anything).Return(
		[]*models.EntityInstance{{ID: "inst1"}}, 1, nil)
	iavRepo.On("RemapAttributeIDs", mock.Anything, []string{"inst1"}, map[string]string{"old-a": "new-a"}).Return(1, nil)

	_, err := svc.UpdatePin(context.Background(), "cv1", "pin1", "etv1-v2", meta.RoleAdmin, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pin update failed")
	// RemapAttributeIDs was called (inside the transaction that will roll back)
	iavRepo.AssertCalled(t, "RemapAttributeIDs", mock.Anything, []string{"inst1"}, map[string]string{"old-a": "new-a"})
}

// filterCalls removes mock expectations matching the given method name.
func filterCalls(calls []*mock.Call, method string) []*mock.Call {
	var filtered []*mock.Call
	for _, c := range calls {
		if c.Method != method {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// T-29.67: Migration fails if any catalog has more than 10000 instances of the entity type
func TestT29_67_ExceedsInstanceLimit(t *testing.T) {
	cvRepo, pinRepo, etvRepo, catalogRepo, attrRepo, instRepo, _, svc := migrationTestSetup()
	setupBasicMigrationMocks(cvRepo, pinRepo, etvRepo)

	attrRepo.On("ListByVersion", mock.Anything, "etv1-v1").Return([]*models.Attribute{
		{ID: "old-a", Name: "attr-a", Ordinal: 0},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv1-v2").Return([]*models.Attribute{
		{ID: "new-a", Name: "attr-a", Ordinal: 0},
	}, nil)
	catalogRepo.On("ListByCatalogVersionID", mock.Anything, "cv1").Return([]*models.Catalog{
		{ID: "cat1", CatalogVersionID: "cv1"},
	}, nil)
	// Return 1 instance but total=10001 (exceeds limit)
	instRepo.On("List", mock.Anything, "et1", "cat1", mock.Anything).Return(
		[]*models.EntityInstance{{ID: "inst1"}}, 10001, nil)

	_, err := svc.UpdatePin(context.Background(), "cv1", "pin1", "etv1-v2", meta.RoleAdmin, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds migration limit")
}

// Bug fix: renamed attribute with type change should produce BOTH renamed and type_changed warnings
func TestRenamedAttribute_WithTypeChange(t *testing.T) {
	cvRepo, pinRepo, etvRepo, catalogRepo, attrRepo, instRepo, iavRepo, svc := migrationTestSetup()
	setupBasicMigrationMocks(cvRepo, pinRepo, etvRepo)

	// Old: "hostname" (string), New: "host" (integer) — same ordinal, different name AND type
	attrRepo.On("ListByVersion", mock.Anything, "etv1-v1").Return([]*models.Attribute{
		{ID: "old-a", Name: "hostname", Ordinal: 0, TypeDefinitionVersionID: "tdv-str"},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv1-v2").Return([]*models.Attribute{
		{ID: "new-a", Name: "host", Ordinal: 0, TypeDefinitionVersionID: "tdv-int"},
	}, nil)
	catalogRepo.On("ListByCatalogVersionID", mock.Anything, "cv1").Return([]*models.Catalog{
		{ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1", ValidationStatus: models.ValidationStatusValid},
	}, nil)
	catalogRepo.On("UpdateValidationStatus", mock.Anything, "cat1", models.ValidationStatusDraft).Return(nil)
	instRepo.On("List", mock.Anything, "et1", "cat1", mock.Anything).Return([]*models.EntityInstance{
		{ID: "inst1"},
	}, 1, nil)
	iavRepo.On("RemapAttributeIDs", mock.Anything, mock.Anything, mock.Anything).Return(1, nil)

	result, err := svc.UpdatePin(context.Background(), "cv1", "pin1", "etv1-v2", meta.RoleAdmin, false)
	require.NoError(t, err)

	var hasRenamed, hasTypeChanged bool
	for _, w := range result.Migration.Warnings {
		if w.Type == "renamed" {
			hasRenamed = true
		}
		if w.Type == "type_changed" {
			hasTypeChanged = true
		}
	}
	assert.True(t, hasRenamed, "expected renamed warning")
	assert.True(t, hasTypeChanged, "expected type_changed warning for renamed attribute with different type")
}

// Bug fix: DeleteCatalogVersion should be blocked if catalogs are pinned to the CV
func TestDeleteCatalogVersion_BlockedWhenCatalogsExist(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	catalogRepo := new(mocks.MockCatalogRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, catalogRepo)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	catalogRepo.On("ListByCatalogVersionID", mock.Anything, "cv1").Return([]*models.Catalog{
		{ID: "cat1", Name: "my-catalog", CatalogVersionID: "cv1"},
	}, nil)

	err := svc.DeleteCatalogVersion(context.Background(), "cv1", meta.RoleAdmin)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "catalog")
}

func TestDeleteCatalogVersion_AllowedWhenNoCatalogs(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	catalogRepo := new(mocks.MockCatalogRepo)
	ltRepo := new(mocks.MockLifecycleTransitionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, ltRepo, nil, "", nil, nil, nil, catalogRepo)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	catalogRepo.On("ListByCatalogVersionID", mock.Anything, "cv1").Return([]*models.Catalog{}, nil)
	cvRepo.On("Delete", mock.Anything, "cv1").Return(nil)

	err := svc.DeleteCatalogVersion(context.Background(), "cv1", meta.RoleAdmin)
	assert.NoError(t, err)
}
