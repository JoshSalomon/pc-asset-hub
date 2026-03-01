package meta_test

import (
	"context"
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

	e, err := svc.CreateEnum(context.Background(), "Status", []string{"active", "inactive"})
	require.NoError(t, err)
	assert.NotEmpty(t, e.ID)
	assert.Equal(t, "Status", e.Name)
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
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, ltRepo, nil, "", nil, nil, nil)

	cvRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersion")).Return(nil)
	ltRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.LifecycleTransition")).Return(nil)

	cv, err := svc.CreateCatalogVersion(context.Background(), "v1.0", nil)
	require.NoError(t, err)
	assert.Equal(t, models.LifecycleStageDevelopment, cv.LifecycleStage)
}

func TestT3_36_CreateCatalogVersionWithPins(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	ltRepo := new(mocks.MockLifecycleTransitionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, ltRepo, nil, "", nil, nil, nil)

	cvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	pinRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin")).Return(nil)
	ltRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	pins := []models.CatalogVersionPin{{EntityTypeVersionID: "etv1"}}
	cv, err := svc.CreateCatalogVersion(context.Background(), "v1.0", pins)
	require.NoError(t, err)
	assert.NotEmpty(t, cv.ID)
	pinRepo.AssertCalled(t, "Create", mock.Anything, mock.AnythingOfType("*models.CatalogVersionPin"))
}

func TestT3_37_PromoteDevToTestAsRW(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	ltRepo := new(mocks.MockLifecycleTransitionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, ltRepo, nil, "", nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	cvRepo.On("UpdateLifecycle", mock.Anything, "cv1", models.LifecycleStageTesting).Return(nil)
	ltRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	err := svc.Promote(context.Background(), "cv1", meta.RoleRW, "user1")
	assert.NoError(t, err)
}

func TestT3_38_PromoteDevToTestAsRO(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)

	err := svc.Promote(context.Background(), "cv1", meta.RoleRO, "user1")
	assert.True(t, domainerrors.IsForbidden(err))
}

func TestT3_39_DemoteTestToDevAsRW(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	ltRepo := new(mocks.MockLifecycleTransitionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, ltRepo, nil, "", nil, nil, nil)

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
	svc := meta.NewCatalogVersionService(cvRepo, nil, ltRepo, nil, "", nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageTesting,
	}, nil)
	cvRepo.On("UpdateLifecycle", mock.Anything, "cv1", models.LifecycleStageProduction).Return(nil)
	ltRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	err := svc.Promote(context.Background(), "cv1", meta.RoleAdmin, "admin1")
	assert.NoError(t, err)
}

func TestT3_41_PromoteTestToProdAsRW(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageTesting,
	}, nil)

	err := svc.Promote(context.Background(), "cv1", meta.RoleRW, "user1")
	assert.True(t, domainerrors.IsForbidden(err))
}

func TestT3_42_DemoteProdToTestAsSuperAdmin(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	ltRepo := new(mocks.MockLifecycleTransitionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, ltRepo, nil, "", nil, nil, nil)

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
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil)

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
	svc := meta.NewCatalogVersionService(cvRepo, nil, ltRepo, nil, "", nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	cvRepo.On("UpdateLifecycle", mock.Anything, "cv1", models.LifecycleStageTesting).Return(nil)
	ltRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	err := svc.Promote(context.Background(), "cv1", meta.RoleAdmin, "admin1")
	assert.NoError(t, err)
	// It should have gone to testing, not production
	cvRepo.AssertCalled(t, "UpdateLifecycle", mock.Anything, "cv1", models.LifecycleStageTesting)
	cvRepo.AssertNotCalled(t, "UpdateLifecycle", mock.Anything, "cv1", models.LifecycleStageProduction)
}

func TestT3_45_TransitionsRecorded(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	ltRepo := new(mocks.MockLifecycleTransitionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, ltRepo, nil, "", nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	cvRepo.On("UpdateLifecycle", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	ltRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.LifecycleTransition")).Return(nil)

	err := svc.Promote(context.Background(), "cv1", meta.RoleRW, "user1")
	assert.NoError(t, err)
	ltRepo.AssertCalled(t, "Create", mock.Anything, mock.AnythingOfType("*models.LifecycleTransition"))
}

func TestT3_46_ModifyProductionAsSuperAdmin(t *testing.T) {
	// Super Admin can modify production — this is about the lifecycle service
	// allowing Super Admin to do things that others can't.
	// The Demote method with SuperAdmin role succeeds for production.
	cvRepo := new(mocks.MockCatalogVersionRepo)
	ltRepo := new(mocks.MockLifecycleTransitionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, ltRepo, nil, "", nil, nil, nil)

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
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageProduction,
	}, nil)

	err := svc.Demote(context.Background(), "cv1", meta.RoleAdmin, "admin", models.LifecycleStageTesting)
	assert.True(t, domainerrors.IsForbidden(err))
}

// === DeleteCatalogVersion Tests ===

func TestDeleteCatalogVersion_AdminDeletesDev(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	cvRepo.On("Delete", mock.Anything, "cv1").Return(nil)

	err := svc.DeleteCatalogVersion(context.Background(), "cv1", meta.RoleAdmin)
	assert.NoError(t, err)
}

func TestDeleteCatalogVersion_AdminDeletesTesting(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageTesting,
	}, nil)
	cvRepo.On("Delete", mock.Anything, "cv1").Return(nil)

	err := svc.DeleteCatalogVersion(context.Background(), "cv1", meta.RoleAdmin)
	assert.NoError(t, err)
}

func TestDeleteCatalogVersion_AdminCannotDeleteProduction(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageProduction,
	}, nil)

	err := svc.DeleteCatalogVersion(context.Background(), "cv1", meta.RoleAdmin)
	assert.True(t, domainerrors.IsForbidden(err))
}

func TestDeleteCatalogVersion_SuperAdminDeletesProduction(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageProduction,
	}, nil)
	cvRepo.On("Delete", mock.Anything, "cv1").Return(nil)

	err := svc.DeleteCatalogVersion(context.Background(), "cv1", meta.RoleSuperAdmin)
	assert.NoError(t, err)
}

func TestDeleteCatalogVersion_RWForbidden(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)

	err := svc.DeleteCatalogVersion(context.Background(), "cv1", meta.RoleRW)
	assert.True(t, domainerrors.IsForbidden(err))
}

func TestDeleteCatalogVersion_ROForbidden(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)

	err := svc.DeleteCatalogVersion(context.Background(), "cv1", meta.RoleRO)
	assert.True(t, domainerrors.IsForbidden(err))
}

func TestDeleteCatalogVersion_NotFound(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil)

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
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, etRepo, etvRepo)

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
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, nil, nil)

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
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, nil, nil)

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
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, nil, etvRepo)

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
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, nil, nil, "", nil, etRepo, etvRepo)

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
	svc := meta.NewCatalogVersionService(cvRepo, nil, ltRepo, nil, "", nil, nil, nil)

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
	svc := meta.NewCatalogVersionService(cvRepo, nil, ltRepo, nil, "", nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "bad").Return(nil, domainerrors.NewNotFound("CatalogVersion", "bad"))

	_, err := svc.ListTransitions(context.Background(), "bad")
	assert.True(t, domainerrors.IsNotFound(err))
}

func TestTE24_ListTransitionsOrdered(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	ltRepo := new(mocks.MockLifecycleTransitionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, ltRepo, nil, "", nil, nil, nil)

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
