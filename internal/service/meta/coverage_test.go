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

// === EntityTypeService coverage ===

func TestGetEntityType(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	svc := meta.NewEntityTypeService(etRepo, nil, nil, nil)

	etRepo.On("GetByID", mock.Anything, "et1").Return(&models.EntityType{ID: "et1", Name: "Model"}, nil)

	et, err := svc.GetEntityType(context.Background(), "et1")
	require.NoError(t, err)
	assert.Equal(t, "Model", et.Name)
}

// === CatalogVersionService coverage ===

func TestGetCatalogVersion(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", VersionLabel: "v1.0", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)

	cv, err := svc.GetCatalogVersion(context.Background(), "cv1")
	require.NoError(t, err)
	assert.Equal(t, "v1.0", cv.VersionLabel)
}

func TestListCatalogVersions(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("List", mock.Anything, mock.Anything).Return([]*models.CatalogVersion{
		{ID: "cv1", VersionLabel: "v1.0"},
		{ID: "cv2", VersionLabel: "v2.0"},
	}, 2, nil)

	items, total, err := svc.ListCatalogVersions(context.Background(), models.ListParams{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, items, 2)
}

func TestPromote_ProductionFails(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageProduction,
	}, nil)

	_, err := svc.Promote(context.Background(), "cv1", meta.RoleAdmin, "admin")
	assert.Error(t, err)
}

func TestPromote_InvalidStage(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStage("invalid"),
	}, nil)

	_, err := svc.Promote(context.Background(), "cv1", meta.RoleAdmin, "admin")
	assert.Error(t, err)
}

func TestPromote_ROForbidden(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)

	_, err := svc.Promote(context.Background(), "cv1", meta.RoleRO, "ro")
	assert.Error(t, err)
}

func TestDemote_DevelopmentFails(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)

	err := svc.Demote(context.Background(), "cv1", meta.RoleAdmin, "admin", models.LifecycleStageDevelopment)
	assert.Error(t, err)
}

func TestDemote_TestingInvalidTarget(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageTesting,
	}, nil)

	err := svc.Demote(context.Background(), "cv1", meta.RoleAdmin, "admin", models.LifecycleStageProduction)
	assert.Error(t, err)
}

func TestDemote_TestingROForbidden(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageTesting,
	}, nil)

	err := svc.Demote(context.Background(), "cv1", meta.RoleRO, "ro", models.LifecycleStageDevelopment)
	assert.Error(t, err)
}

func TestDemote_ProductionInvalidTarget(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageProduction,
	}, nil)

	err := svc.Demote(context.Background(), "cv1", meta.RoleSuperAdmin, "sa", models.LifecycleStage("invalid"))
	assert.Error(t, err)
}

func TestDemote_InvalidStage(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStage("unknown"),
	}, nil)

	err := svc.Demote(context.Background(), "cv1", meta.RoleSuperAdmin, "sa", models.LifecycleStageDevelopment)
	assert.Error(t, err)
}

// === MockCRManager for CatalogVersion CR tests ===

type mockCRManager struct{ mock.Mock }

func (m *mockCRManager) CreateOrUpdate(ctx context.Context, spec meta.CatalogVersionCRSpec) error {
	return m.Called(ctx, spec).Error(0)
}
func (m *mockCRManager) Delete(ctx context.Context, name, namespace string) error {
	return m.Called(ctx, name, namespace).Error(0)
}

// T-CV.23: Promote dev→testing calls crManager.CreateOrUpdate with lifecycleStage="testing"
func TestTCV23_PromoteDevToTestingCallsCreateOrUpdate(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	ltRepo := new(mocks.MockLifecycleTransitionRepo)
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	crMgr := new(mockCRManager)

	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, ltRepo, crMgr, "assethub", nil, etRepo, etvRepo, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", VersionLabel: "Release 1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	cvRepo.On("UpdateLifecycle", mock.Anything, "cv1", models.LifecycleStageTesting).Return(nil)
	ltRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return([]*models.CatalogVersionPin{
		{ID: "p1", CatalogVersionID: "cv1", EntityTypeVersionID: "etv1"},
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "etv1").Return(&models.EntityTypeVersion{
		ID: "etv1", EntityTypeID: "et1", Version: 1,
	}, nil)
	etRepo.On("GetByID", mock.Anything, "et1").Return(&models.EntityType{ID: "et1", Name: "Device"}, nil)
	crMgr.On("CreateOrUpdate", mock.Anything, mock.MatchedBy(func(spec meta.CatalogVersionCRSpec) bool {
		return spec.LifecycleStage == "testing" && spec.Name == "release-1"
	})).Return(nil)

	_, err := svc.Promote(context.Background(), "cv1", meta.RoleRW, "admin")
	require.NoError(t, err)
	crMgr.AssertCalled(t, "CreateOrUpdate", mock.Anything, mock.Anything)
}

// T-CV.24: Promote testing→production calls crManager.CreateOrUpdate with lifecycleStage="production"
func TestTCV24_PromoteTestingToProductionCallsCreateOrUpdate(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	ltRepo := new(mocks.MockLifecycleTransitionRepo)
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	crMgr := new(mockCRManager)

	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, ltRepo, crMgr, "assethub", nil, etRepo, etvRepo, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", VersionLabel: "Release 1", LifecycleStage: models.LifecycleStageTesting,
	}, nil)
	cvRepo.On("UpdateLifecycle", mock.Anything, "cv1", models.LifecycleStageProduction).Return(nil)
	ltRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return([]*models.CatalogVersionPin{}, nil)
	crMgr.On("CreateOrUpdate", mock.Anything, mock.MatchedBy(func(spec meta.CatalogVersionCRSpec) bool {
		return spec.LifecycleStage == "production"
	})).Return(nil)

	_, err := svc.Promote(context.Background(), "cv1", meta.RoleAdmin, "admin")
	require.NoError(t, err)
	crMgr.AssertCalled(t, "CreateOrUpdate", mock.Anything, mock.Anything)
}

// T-CV.25: Demote testing→development calls crManager.Delete
func TestTCV25_DemoteTestingToDevCallsDelete(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	ltRepo := new(mocks.MockLifecycleTransitionRepo)
	crMgr := new(mockCRManager)

	svc := meta.NewCatalogVersionService(cvRepo, nil, ltRepo, crMgr, "assethub", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", VersionLabel: "Release 1", LifecycleStage: models.LifecycleStageTesting,
	}, nil)
	cvRepo.On("UpdateLifecycle", mock.Anything, "cv1", models.LifecycleStageDevelopment).Return(nil)
	ltRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	crMgr.On("Delete", mock.Anything, "release-1", "assethub").Return(nil)

	err := svc.Demote(context.Background(), "cv1", meta.RoleRW, "admin", models.LifecycleStageDevelopment)
	require.NoError(t, err)
	crMgr.AssertCalled(t, "Delete", mock.Anything, "release-1", "assethub")
}

// T-CV.26: Demote production→testing calls crManager.CreateOrUpdate (not Delete)
func TestTCV26_DemoteProdToTestingCallsCreateOrUpdate(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	ltRepo := new(mocks.MockLifecycleTransitionRepo)
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	crMgr := new(mockCRManager)

	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, ltRepo, crMgr, "assethub", nil, etRepo, etvRepo, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", VersionLabel: "Release 1", LifecycleStage: models.LifecycleStageProduction,
	}, nil)
	cvRepo.On("UpdateLifecycle", mock.Anything, "cv1", models.LifecycleStageTesting).Return(nil)
	ltRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return([]*models.CatalogVersionPin{}, nil)
	crMgr.On("CreateOrUpdate", mock.Anything, mock.MatchedBy(func(spec meta.CatalogVersionCRSpec) bool {
		return spec.LifecycleStage == "testing"
	})).Return(nil)

	err := svc.Demote(context.Background(), "cv1", meta.RoleSuperAdmin, "sa", models.LifecycleStageTesting)
	require.NoError(t, err)
	crMgr.AssertCalled(t, "CreateOrUpdate", mock.Anything, mock.Anything)
	crMgr.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything, mock.Anything)
}

// T-CV.27: Promote with crManager=nil does not panic, still updates DB
func TestTCV27_PromoteNilCRManagerNoPanic(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	ltRepo := new(mocks.MockLifecycleTransitionRepo)

	svc := meta.NewCatalogVersionService(cvRepo, nil, ltRepo, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", VersionLabel: "v1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	cvRepo.On("UpdateLifecycle", mock.Anything, "cv1", models.LifecycleStageTesting).Return(nil)
	ltRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	_, err := svc.Promote(context.Background(), "cv1", meta.RoleRW, "admin")
	require.NoError(t, err)
	cvRepo.AssertCalled(t, "UpdateLifecycle", mock.Anything, "cv1", models.LifecycleStageTesting)
}

// T-CV.28: ListCatalogVersions with allowedStages=["production"] returns only production versions
func TestTCV28_ListFilteredByAllowedStages(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)

	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", []string{"production"}, nil, nil, nil)

	cvRepo.On("List", mock.Anything, mock.Anything).Return([]*models.CatalogVersion{
		{ID: "cv1", VersionLabel: "v1", LifecycleStage: models.LifecycleStageDevelopment},
		{ID: "cv2", VersionLabel: "v2", LifecycleStage: models.LifecycleStageTesting},
		{ID: "cv3", VersionLabel: "v3", LifecycleStage: models.LifecycleStageProduction},
	}, 3, nil)

	items, total, err := svc.ListCatalogVersions(context.Background(), models.ListParams{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, items, 1)
	assert.Equal(t, "cv3", items[0].ID)
}

// T-CV.29: GetCatalogVersion returns Forbidden when version stage not in allowedStages
func TestTCV29_GetForbiddenWhenStageNotAllowed(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)

	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", []string{"production"}, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", VersionLabel: "v1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)

	_, err := svc.GetCatalogVersion(context.Background(), "cv1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not available")
}

// === EnumService coverage ===

func TestGetEnum(t *testing.T) {
	enumRepo := new(mocks.MockEnumRepo)
	svc := meta.NewEnumService(enumRepo, nil, nil)

	enumRepo.On("GetByID", mock.Anything, "e1").Return(&models.Enum{ID: "e1", Name: "Status"}, nil)

	e, err := svc.GetEnum(context.Background(), "e1")
	require.NoError(t, err)
	assert.Equal(t, "Status", e.Name)
}

func TestListEnums(t *testing.T) {
	enumRepo := new(mocks.MockEnumRepo)
	svc := meta.NewEnumService(enumRepo, nil, nil)

	enumRepo.On("List", mock.Anything, mock.Anything).Return([]*models.Enum{
		{ID: "e1", Name: "Status"},
	}, 1, nil)

	items, total, err := svc.ListEnums(context.Background(), models.ListParams{Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, items, 1)
}

func TestUpdateEnum(t *testing.T) {
	enumRepo := new(mocks.MockEnumRepo)
	svc := meta.NewEnumService(enumRepo, nil, nil)

	enumRepo.On("GetByID", mock.Anything, "e1").Return(&models.Enum{ID: "e1", Name: "Old"}, nil)
	enumRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	err := svc.UpdateEnum(context.Background(), "e1", "Updated")
	assert.NoError(t, err)
}

func TestReorderValues(t *testing.T) {
	evRepo := new(mocks.MockEnumValueRepo)
	svc := meta.NewEnumService(nil, evRepo, nil)

	evRepo.On("Reorder", mock.Anything, "e1", []string{"v2", "v1"}).Return(nil)

	err := svc.ReorderValues(context.Background(), "e1", []string{"v2", "v1"})
	assert.NoError(t, err)
}

// === AssociationService coverage ===

func TestListAssociations(t *testing.T) {
	assocRepo := new(mocks.MockAssociationRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := meta.NewAssociationService(assocRepo, etvRepo, nil)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(
		&models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{
		{ID: "a1", Type: models.AssociationTypeContainment},
	}, nil)

	assocs, err := svc.ListAssociations(context.Background(), "et1")
	require.NoError(t, err)
	assert.Len(t, assocs, 1)
}

func TestListAllAssociations_BothDirections(t *testing.T) {
	assocRepo := new(mocks.MockAssociationRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := meta.NewAssociationService(assocRepo, etvRepo, nil)

	// Entity type A (et-a) has an outgoing containment to B (et-b)
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-b").Return(
		&models.EntityTypeVersion{ID: "v-b1", EntityTypeID: "et-b", Version: 1}, nil)
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(
		&models.EntityTypeVersion{ID: "v-a1", EntityTypeID: "et-a", Version: 1}, nil)

	// B has no outgoing associations
	assocRepo.On("ListByVersion", mock.Anything, "v-b1").Return([]*models.Association{}, nil)

	// B is targeted by A's containment association
	assocRepo.On("ListByTargetEntityType", mock.Anything, "et-b").Return([]*models.Association{
		{ID: "assoc1", EntityTypeVersionID: "v-a1", TargetEntityTypeID: "et-b", Type: models.AssociationTypeContainment, SourceRole: "parent", TargetRole: "child"},
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "v-a1").Return(
		&models.EntityTypeVersion{ID: "v-a1", EntityTypeID: "et-a", Version: 1}, nil)

	result, err := svc.ListAllAssociations(context.Background(), "et-b")
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "incoming", result[0].Direction)
	assert.Equal(t, "et-a", result[0].SourceEntityTypeID)
	assert.Equal(t, models.AssociationTypeContainment, result[0].Type)
}

func TestListAllAssociations_OutgoingAndIncoming(t *testing.T) {
	assocRepo := new(mocks.MockAssociationRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := meta.NewAssociationService(assocRepo, etvRepo, nil)

	// A contains B, B references C
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-b").Return(
		&models.EntityTypeVersion{ID: "v-b1", EntityTypeID: "et-b", Version: 1}, nil)
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(
		&models.EntityTypeVersion{ID: "v-a1", EntityTypeID: "et-a", Version: 1}, nil)

	// B has an outgoing directional reference to C
	assocRepo.On("ListByVersion", mock.Anything, "v-b1").Return([]*models.Association{
		{ID: "assoc-bc", EntityTypeVersionID: "v-b1", TargetEntityTypeID: "et-c", Type: models.AssociationTypeDirectional},
	}, nil)

	// B is also targeted by A
	assocRepo.On("ListByTargetEntityType", mock.Anything, "et-b").Return([]*models.Association{
		{ID: "assoc-ab", EntityTypeVersionID: "v-a1", TargetEntityTypeID: "et-b", Type: models.AssociationTypeContainment},
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "v-a1").Return(
		&models.EntityTypeVersion{ID: "v-a1", EntityTypeID: "et-a", Version: 1}, nil)

	result, err := svc.ListAllAssociations(context.Background(), "et-b")
	require.NoError(t, err)
	require.Len(t, result, 2)

	// One outgoing, one incoming
	var outCount, inCount int
	for _, r := range result {
		if r.Direction == "outgoing" {
			outCount++
		} else {
			inCount++
		}
	}
	assert.Equal(t, 1, outCount)
	assert.Equal(t, 1, inCount)
}

func TestListAllAssociations_ListByTargetError(t *testing.T) {
	assocRepo := new(mocks.MockAssociationRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := meta.NewAssociationService(assocRepo, etvRepo, nil)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-b").Return(
		&models.EntityTypeVersion{ID: "v-b1", EntityTypeID: "et-b", Version: 1}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v-b1").Return([]*models.Association{}, nil)
	assocRepo.On("ListByTargetEntityType", mock.Anything, "et-b").Return(([]*models.Association)(nil), domainerrors.NewNotFound("Association", "et-b"))

	_, err := svc.ListAllAssociations(context.Background(), "et-b")
	assert.True(t, domainerrors.IsNotFound(err))
}

func TestListAllAssociations_GetByIDError(t *testing.T) {
	assocRepo := new(mocks.MockAssociationRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := meta.NewAssociationService(assocRepo, etvRepo, nil)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-b").Return(
		&models.EntityTypeVersion{ID: "v-b1", EntityTypeID: "et-b", Version: 1}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v-b1").Return([]*models.Association{}, nil)
	assocRepo.On("ListByTargetEntityType", mock.Anything, "et-b").Return([]*models.Association{
		{ID: "assoc1", EntityTypeVersionID: "v-a1", TargetEntityTypeID: "et-b", Type: models.AssociationTypeContainment},
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "v-a1").Return(nil, domainerrors.NewNotFound("EntityTypeVersion", "v-a1"))

	_, err := svc.ListAllAssociations(context.Background(), "et-b")
	assert.True(t, domainerrors.IsNotFound(err))
}

func TestListAllAssociations_GetLatestSourceError(t *testing.T) {
	assocRepo := new(mocks.MockAssociationRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := meta.NewAssociationService(assocRepo, etvRepo, nil)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-b").Return(
		&models.EntityTypeVersion{ID: "v-b1", EntityTypeID: "et-b", Version: 1}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v-b1").Return([]*models.Association{}, nil)
	assocRepo.On("ListByTargetEntityType", mock.Anything, "et-b").Return([]*models.Association{
		{ID: "assoc1", EntityTypeVersionID: "v-a1", TargetEntityTypeID: "et-b", Type: models.AssociationTypeContainment},
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "v-a1").Return(
		&models.EntityTypeVersion{ID: "v-a1", EntityTypeID: "et-a", Version: 1}, nil)
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(nil, domainerrors.NewNotFound("EntityTypeVersion", "et-a"))

	_, err := svc.ListAllAssociations(context.Background(), "et-b")
	assert.True(t, domainerrors.IsNotFound(err))
}

func TestListAllAssociations_SkipsOldVersions(t *testing.T) {
	assocRepo := new(mocks.MockAssociationRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	svc := meta.NewAssociationService(assocRepo, etvRepo, nil)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-b").Return(
		&models.EntityTypeVersion{ID: "v-b1", EntityTypeID: "et-b", Version: 1}, nil)
	// A's latest is v-a2, but the incoming association is from v-a1 (old version)
	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-a").Return(
		&models.EntityTypeVersion{ID: "v-a2", EntityTypeID: "et-a", Version: 2}, nil)

	assocRepo.On("ListByVersion", mock.Anything, "v-b1").Return([]*models.Association{}, nil)
	assocRepo.On("ListByTargetEntityType", mock.Anything, "et-b").Return([]*models.Association{
		{ID: "assoc-old", EntityTypeVersionID: "v-a1", TargetEntityTypeID: "et-b", Type: models.AssociationTypeContainment},
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "v-a1").Return(
		&models.EntityTypeVersion{ID: "v-a1", EntityTypeID: "et-a", Version: 1}, nil)

	result, err := svc.ListAllAssociations(context.Background(), "et-b")
	require.NoError(t, err)
	// Old version association is skipped
	assert.Len(t, result, 0)
}

// === Additional coverage: error paths in service/meta ===

// ListAttributes exercises the trivial delegator
func TestListAttributes_Success(t *testing.T) {
	attrRepo := new(mocks.MockAttributeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	etRepo := new(mocks.MockEntityTypeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewAttributeService(attrRepo, etvRepo, etRepo, assocRepo, nil)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "etv1"}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Attribute{
		{ID: "a1", Name: "hostname"},
	}, nil)

	attrs, err := svc.ListAttributes(context.Background(), "et1")
	require.NoError(t, err)
	assert.Len(t, attrs, 1)
}

func TestListAttributes_GetLatestError(t *testing.T) {
	attrRepo := new(mocks.MockAttributeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	etRepo := new(mocks.MockEntityTypeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewAttributeService(attrRepo, etvRepo, etRepo, assocRepo, nil)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(nil, domainerrors.NewNotFound("ETV", "et1"))

	_, err := svc.ListAttributes(context.Background(), "et1")
	assert.Error(t, err)
}

// RenameEntityType: GetByName returns non-NotFound error
func TestRenameEntityType_GetByNameError(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, attrRepo, assocRepo)

	etRepo.On("GetByID", mock.Anything, "et1").Return(&models.EntityType{ID: "et1", Name: "old"}, nil)
	etRepo.On("GetByName", mock.Anything, "new-name").Return(nil, domainerrors.NewValidation("db error"))

	_, err := svc.RenameEntityType(context.Background(), "et1", "new-name", false)
	assert.Error(t, err)
}

// requiresDeepCopy: entity type with no versions
func TestRenameEntityType_NoVersions(t *testing.T) {
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewEntityTypeService(etRepo, etvRepo, attrRepo, assocRepo)

	etRepo.On("GetByID", mock.Anything, "et1").Return(&models.EntityType{ID: "et1", Name: "old"}, nil)
	etRepo.On("GetByName", mock.Anything, "new-name").Return(nil, domainerrors.NewNotFound("ET", "new-name"))
	etvRepo.On("ListByEntityType", mock.Anything, "et1").Return([]*models.EntityTypeVersion{}, nil)
	etRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	result, err := svc.RenameEntityType(context.Background(), "et1", "new-name", false)
	require.NoError(t, err)
	assert.False(t, result.WasDeepCopy)
}

// AddAttribute: BulkCopyToVersion error
func TestAddAttribute_BulkCopyError(t *testing.T) {
	attrRepo := new(mocks.MockAttributeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	etRepo := new(mocks.MockEntityTypeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewAttributeService(attrRepo, etvRepo, etRepo, assocRepo, nil)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Association{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, "etv1", mock.Anything).Return(domainerrors.NewValidation("copy error"))

	_, err := svc.AddAttribute(context.Background(), "et1", "new-attr", "", models.AttributeTypeString, "", false)
	assert.Error(t, err)
}

// === Remaining coverage: all uncovered error paths ===

// CreateAssociation: invalid target cardinality (line 45)
func TestCreateAssociation_InvalidTargetCardinality(t *testing.T) {
	assocRepo := new(mocks.MockAssociationRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	svc := meta.NewAssociationService(assocRepo, etvRepo, attrRepo)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Association{}, nil)

	_, err := svc.CreateAssociation(context.Background(), "et1", "et2", "directional", "assoc1", "", "", "0..n", "invalid!")
	assert.Error(t, err)
}

// AddAttribute: assocRepo.BulkCopyToVersion error (line 72)
func TestAddAttribute_AssocBulkCopyError_Coverage(t *testing.T) {
	attrRepo := new(mocks.MockAttributeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	etRepo := new(mocks.MockEntityTypeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewAttributeService(attrRepo, etvRepo, etRepo, assocRepo, nil)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Association{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, "etv1", mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, "etv1", mock.Anything).Return(domainerrors.NewValidation("assoc copy error"))

	_, err := svc.AddAttribute(context.Background(), "et1", "new-attr", "", models.AttributeTypeString, "", false)
	assert.Error(t, err)
}

// RemoveAttribute: attribute not found after COW (line 153)
func TestRemoveAttribute_NotFound(t *testing.T) {
	attrRepo := new(mocks.MockAttributeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	etRepo := new(mocks.MockEntityTypeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewAttributeService(attrRepo, etvRepo, etRepo, assocRepo, nil)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, "etv1", mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, "etv1", mock.Anything).Return(nil)
	// After COW, the new version's attributes don't include "nonexistent"
	attrRepo.On("ListByVersion", mock.Anything, mock.Anything).Return([]*models.Attribute{
		{ID: "a1", Name: "hostname"},
	}, nil)

	_, err := svc.RemoveAttribute(context.Background(), "et1", "nonexistent")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
}

// CopyAttributesFromType: name conflict (line 186)
func TestCopyAttributes_NameConflict(t *testing.T) {
	attrRepo := new(mocks.MockAttributeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	etRepo := new(mocks.MockEntityTypeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewAttributeService(attrRepo, etvRepo, etRepo, assocRepo, nil)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-target").Return(&models.EntityTypeVersion{ID: "etv-t", EntityTypeID: "et-target", Version: 1}, nil)
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et-source", 1).Return(&models.EntityTypeVersion{ID: "etv-s", EntityTypeID: "et-source", Version: 1}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv-t").Return([]*models.Attribute{
		{ID: "a1", Name: "hostname"},
	}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv-s").Return([]*models.Attribute{
		{ID: "a2", Name: "hostname"}, // same name — conflict
	}, nil)

	_, err := svc.CopyAttributesFromType(context.Background(), "et-target", "et-source", 1, []string{"hostname"})
	assert.Error(t, err)
	assert.True(t, domainerrors.IsConflict(err))
}

// CopyAttributesFromType: attribute not found in source (line 197)
func TestCopyAttributes_SourceAttrNotFound(t *testing.T) {
	attrRepo := new(mocks.MockAttributeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	etRepo := new(mocks.MockEntityTypeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewAttributeService(attrRepo, etvRepo, etRepo, assocRepo, nil)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et-target").Return(&models.EntityTypeVersion{ID: "etv-t", EntityTypeID: "et-target", Version: 1}, nil)
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et-source", 1).Return(&models.EntityTypeVersion{ID: "etv-s", EntityTypeID: "et-source", Version: 1}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv-t").Return([]*models.Attribute{}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv-s").Return([]*models.Attribute{
		{ID: "a1", Name: "port"},
	}, nil)

	_, err := svc.CopyAttributesFromType(context.Background(), "et-target", "et-source", 1, []string{"nonexistent"})
	assert.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
}

// EditAttribute: attribute not found (line 336)
func TestEditAttribute_NotFound(t *testing.T) {
	attrRepo := new(mocks.MockAttributeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	etRepo := new(mocks.MockEntityTypeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewAttributeService(attrRepo, etvRepo, etRepo, assocRepo, nil)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Attribute{
		{ID: "a1", Name: "hostname"},
	}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Association{}, nil)

	newName := "new-name"
	_, err := svc.EditAttribute(context.Background(), "et1", "nonexistent", &newName, nil, nil, nil, nil)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
}

// EditAttribute: BulkCopyToVersion error (line 279)
func TestEditAttribute_BulkCopyError(t *testing.T) {
	attrRepo := new(mocks.MockAttributeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	etRepo := new(mocks.MockEntityTypeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewAttributeService(attrRepo, etvRepo, etRepo, assocRepo, nil)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Attribute{
		{ID: "a1", Name: "hostname", Type: models.AttributeTypeString},
	}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Association{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, "etv1", mock.Anything).Return(domainerrors.NewValidation("copy error"))

	newName := "new-name"
	_, err := svc.EditAttribute(context.Background(), "et1", "hostname", &newName, nil, nil, nil, nil)
	assert.Error(t, err)
}

// EditAssociation: targetRole update (line 198) + not found (line 217)
func TestEditAssociation_TargetRoleUpdate(t *testing.T) {
	assocRepo := new(mocks.MockAssociationRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	svc := meta.NewAssociationService(assocRepo, etvRepo, attrRepo)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Association{
		{ID: "as1", Name: "uses", TargetEntityTypeID: "et2", Type: "directional"},
	}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	// After COW, list the new version's associations (with the one to edit)
	assocRepo.On("ListByVersion", mock.Anything, mock.MatchedBy(func(id string) bool { return id != "etv1" })).Return([]*models.Association{
		{ID: "as1-copy", Name: "uses", TargetEntityTypeID: "et2", Type: "directional"},
	}, nil)
	assocRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	targetRole := "target-role"
	_, err := svc.EditAssociation(context.Background(), "et1", "uses", nil, nil, &targetRole, nil, nil, nil)
	require.NoError(t, err)
}

func TestEditAssociation_NotFound(t *testing.T) {
	assocRepo := new(mocks.MockAssociationRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	svc := meta.NewAssociationService(assocRepo, etvRepo, attrRepo)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Association{}, nil)

	_, err := svc.EditAssociation(context.Background(), "et1", "nonexistent", nil, nil, nil, nil, nil, nil)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
}

// DeleteAssociation: not found after COW (line 285)
func TestDeleteAssociation_NotFound(t *testing.T) {
	assocRepo := new(mocks.MockAssociationRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	svc := meta.NewAssociationService(assocRepo, etvRepo, attrRepo)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Association{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	// After COW, new version also has no matching association
	assocRepo.On("ListByVersion", mock.Anything, mock.MatchedBy(func(id string) bool { return id != "etv1" })).Return([]*models.Association{}, nil)

	_, err := svc.DeleteAssociation(context.Background(), "et1", "nonexistent")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
}

// ListAllAssociations: repo errors (lines 308, 314)
func TestListAllAssociations_GetLatestError(t *testing.T) {
	assocRepo := new(mocks.MockAssociationRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	svc := meta.NewAssociationService(assocRepo, etvRepo, attrRepo)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(nil, domainerrors.NewNotFound("ETV", "et1"))

	_, err := svc.ListAllAssociations(context.Background(), "et1")
	assert.Error(t, err)
}

func TestListAllAssociations_ListByVersionError(t *testing.T) {
	assocRepo := new(mocks.MockAssociationRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	svc := meta.NewAssociationService(assocRepo, etvRepo, attrRepo)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1"}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "etv1").Return(nil, fmt.Errorf("db error"))

	_, err := svc.ListAllAssociations(context.Background(), "et1")
	assert.Error(t, err)
}

// CompareVersions: association diff (lines 114, 129-130)
func TestCompareVersions_AssociationDiff(t *testing.T) {
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewVersionHistoryService(etvRepo, attrRepo, assocRepo)

	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 1).Return(&models.EntityTypeVersion{ID: "etv1"}, nil)
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 2).Return(&models.EntityTypeVersion{ID: "etv2"}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Attribute{}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv2").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Association{
		{ID: "a1", Name: "old-assoc", TargetEntityTypeID: "et2", Type: "directional"},
	}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "etv2").Return([]*models.Association{}, nil)

	diff, err := svc.CompareVersions(context.Background(), "et1", 1, 2)
	require.NoError(t, err)
	// old-assoc was removed in v2
	found := false
	for _, c := range diff.Changes {
		if c.ChangeType == "removed" {
			found = true
		}
	}
	assert.True(t, found, "should detect removed association")
}

// GetCatalogVersion: stage filter (line 108)
func TestGetCatalogVersion_DisallowedStage(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	ltRepo := new(mocks.MockLifecycleTransitionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, ltRepo, nil, "", []string{"production"}, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: "development",
	}, nil)

	_, err := svc.GetCatalogVersion(context.Background(), "cv1")
	assert.Error(t, err)
}

// ListCatalogVersions: repo error (line 119)
func TestListCatalogVersions_RepoError(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	ltRepo := new(mocks.MockLifecycleTransitionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, ltRepo, nil, "", []string{"development"}, nil, nil, nil)

	cvRepo.On("List", mock.Anything, mock.Anything).Return(nil, 0, fmt.Errorf("db error"))

	_, _, err := svc.ListCatalogVersions(context.Background(), models.ListParams{})
	assert.Error(t, err)
}

// ListPins: CV not found (line 347)
func TestListPins_CVNotFound(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	ltRepo := new(mocks.MockLifecycleTransitionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, ltRepo, nil, "", nil, nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(nil, domainerrors.NewNotFound("CV", "cv1"))

	_, err := svc.ListPins(context.Background(), "cv1")
	assert.Error(t, err)
}

// === Remaining catalog_version_service coverage ===

type mockCVCRManager struct {
	createOrUpdateErr error
	deleteErr         error
	deleteCalled      bool
}

func (m *mockCVCRManager) CreateOrUpdate(_ context.Context, _ meta.CatalogVersionCRSpec) error {
	return m.createOrUpdateErr
}
func (m *mockCVCRManager) Delete(_ context.Context, _, _ string) error {
	m.deleteCalled = true
	return m.deleteErr
}

func setupCVSvc(crMgr meta.CatalogVersionCRManager) (*meta.CatalogVersionService, *mocks.MockCatalogVersionRepo, *mocks.MockCatalogVersionPinRepo, *mocks.MockLifecycleTransitionRepo, *mocks.MockEntityTypeRepo, *mocks.MockEntityTypeVersionRepo, *mocks.MockCatalogRepo) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	pinRepo := new(mocks.MockCatalogVersionPinRepo)
	ltRepo := new(mocks.MockLifecycleTransitionRepo)
	etRepo := new(mocks.MockEntityTypeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	catRepo := new(mocks.MockCatalogRepo)
	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, ltRepo, crMgr, "ns", []string{"development", "testing", "production"}, etRepo, etvRepo, catRepo)
	return svc, cvRepo, pinRepo, ltRepo, etRepo, etvRepo, catRepo
}

// GetCatalogVersion: repo error (line 108)
func TestGetCatalogVersion_RepoError(t *testing.T) {
	svc, cvRepo, _, _, _, _, _ := setupCVSvc(nil)
	cvRepo.On("GetByID", mock.Anything, "cv1").Return(nil, fmt.Errorf("db error"))

	_, err := svc.GetCatalogVersion(context.Background(), "cv1")
	assert.Error(t, err)
}

// Promote: empty k8s name (line 190)
func TestPromote_EmptyK8sName(t *testing.T) {
	crMgr := &mockCVCRManager{}
	svc, cvRepo, _, ltRepo, _, _, _ := setupCVSvc(crMgr)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{ID: "cv1", VersionLabel: "---", LifecycleStage: "development"}, nil)
	cvRepo.On("UpdateLifecycle", mock.Anything, "cv1", models.LifecycleStageTesting).Return(nil)
	ltRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	_, err := svc.Promote(context.Background(), "cv1", meta.RoleAdmin, "user")
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

// Promote: getEntityTypeNamesForCV error (line 194)
func TestPromote_GetETNamesError(t *testing.T) {
	crMgr := &mockCVCRManager{}
	svc, cvRepo, pinRepo, ltRepo, _, _, _ := setupCVSvc(crMgr)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{ID: "cv1", VersionLabel: "v1", LifecycleStage: "development"}, nil)
	cvRepo.On("UpdateLifecycle", mock.Anything, "cv1", models.LifecycleStageTesting).Return(nil)
	ltRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return(nil, fmt.Errorf("pin error"))

	_, err := svc.Promote(context.Background(), "cv1", meta.RoleAdmin, "user")
	assert.Error(t, err)
}

// Promote: crManager.CreateOrUpdate error (line 207)
func TestPromote_CRCreateError(t *testing.T) {
	crMgr := &mockCVCRManager{createOrUpdateErr: fmt.Errorf("cr error")}
	svc, cvRepo, pinRepo, ltRepo, _, _, _ := setupCVSvc(crMgr)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{ID: "cv1", VersionLabel: "v1", LifecycleStage: "development"}, nil)
	cvRepo.On("UpdateLifecycle", mock.Anything, "cv1", models.LifecycleStageTesting).Return(nil)
	ltRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return([]*models.CatalogVersionPin{}, nil)

	_, err := svc.Promote(context.Background(), "cv1", meta.RoleAdmin, "user")
	assert.Error(t, err)
}

// Promote: catalog warnings (lines 214-218)
func TestPromote_CatalogWarnings(t *testing.T) {
	crMgr := &mockCVCRManager{}
	svc, cvRepo, pinRepo, ltRepo, _, _, catRepo := setupCVSvc(crMgr)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{ID: "cv1", VersionLabel: "v1", LifecycleStage: "development"}, nil)
	cvRepo.On("UpdateLifecycle", mock.Anything, "cv1", models.LifecycleStageTesting).Return(nil)
	ltRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return([]*models.CatalogVersionPin{}, nil)
	catRepo.On("ListByCatalogVersionID", mock.Anything, "cv1").Return([]*models.Catalog{
		{Name: "draft-cat", ValidationStatus: models.ValidationStatusDraft},
		{Name: "valid-cat", ValidationStatus: models.ValidationStatusValid},
	}, nil)

	result, err := svc.Promote(context.Background(), "cv1", meta.RoleAdmin, "user")
	require.NoError(t, err)
	assert.Len(t, result.Warnings, 1)
	assert.Equal(t, "draft-cat", result.Warnings[0].CatalogName)
}

// Promote: ltRepo.Create error (line 194... actually 184)
func TestPromote_TransitionError(t *testing.T) {
	svc, cvRepo, _, ltRepo, _, _, _ := setupCVSvc(nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{ID: "cv1", VersionLabel: "v1", LifecycleStage: "development"}, nil)
	cvRepo.On("UpdateLifecycle", mock.Anything, "cv1", models.LifecycleStageTesting).Return(nil)
	ltRepo.On("Create", mock.Anything, mock.Anything).Return(fmt.Errorf("lt error"))

	_, err := svc.Promote(context.Background(), "cv1", meta.RoleAdmin, "user")
	assert.Error(t, err)
}

// DeleteCatalogVersion: repo delete error (line 250)
func TestDeleteCV_RepoError(t *testing.T) {
	svc, cvRepo, _, _, _, _, _ := setupCVSvc(nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{ID: "cv1", LifecycleStage: "development"}, nil)
	cvRepo.On("Delete", mock.Anything, "cv1").Return(fmt.Errorf("delete error"))

	err := svc.DeleteCatalogVersion(context.Background(), "cv1", meta.RoleAdmin)
	assert.Error(t, err)
}

// DeleteCatalogVersion: crManager.Delete (line 255)
func TestDeleteCV_WithCRManager(t *testing.T) {
	crMgr := &mockCVCRManager{}
	svc, cvRepo, _, _, _, _, _ := setupCVSvc(crMgr)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{ID: "cv1", VersionLabel: "v1", LifecycleStage: "testing"}, nil)
	cvRepo.On("Delete", mock.Anything, "cv1").Return(nil)

	err := svc.DeleteCatalogVersion(context.Background(), "cv1", meta.RoleAdmin)
	require.NoError(t, err)
	assert.True(t, crMgr.deleteCalled)
}

// Demote: empty k8s name (line 310)
func TestDemote_EmptyK8sName(t *testing.T) {
	crMgr := &mockCVCRManager{}
	svc, cvRepo, _, ltRepo, _, _, _ := setupCVSvc(crMgr)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{ID: "cv1", VersionLabel: "---", LifecycleStage: "testing"}, nil)
	cvRepo.On("UpdateLifecycle", mock.Anything, "cv1", models.LifecycleStageDevelopment).Return(nil)
	ltRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	err := svc.Demote(context.Background(), "cv1", meta.RoleRW, "user", models.LifecycleStageDevelopment)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsValidation(err))
}

// Demote: ltRepo.Create error (line 319... actually 304)
func TestDemote_TransitionError(t *testing.T) {
	svc, cvRepo, _, ltRepo, _, _, _ := setupCVSvc(nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{ID: "cv1", VersionLabel: "v1", LifecycleStage: "testing"}, nil)
	cvRepo.On("UpdateLifecycle", mock.Anything, "cv1", models.LifecycleStageDevelopment).Return(nil)
	ltRepo.On("Create", mock.Anything, mock.Anything).Return(fmt.Errorf("lt error"))

	err := svc.Demote(context.Background(), "cv1", meta.RoleRW, "user", models.LifecycleStageDevelopment)
	assert.Error(t, err)
}

// Demote: getEntityTypeNamesForCV error (line 319 in demote to testing)
func TestDemote_ToTesting_ETNamesError(t *testing.T) {
	crMgr := &mockCVCRManager{}
	svc, cvRepo, pinRepo, ltRepo, _, _, _ := setupCVSvc(crMgr)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{ID: "cv1", VersionLabel: "v1", LifecycleStage: "production"}, nil)
	cvRepo.On("UpdateLifecycle", mock.Anything, "cv1", models.LifecycleStageTesting).Return(nil)
	ltRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return(nil, fmt.Errorf("pin error"))

	err := svc.Demote(context.Background(), "cv1", meta.RoleSuperAdmin, "user", models.LifecycleStageTesting)
	assert.Error(t, err)
}

// getEntityTypeNamesForCV: etvRepo.GetByID error (line 393) — tested via Promote
func TestPromote_ETVResolveError(t *testing.T) {
	crMgr := &mockCVCRManager{}
	svc, cvRepo, pinRepo, ltRepo, _, etvRepo, _ := setupCVSvc(crMgr)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{ID: "cv1", VersionLabel: "v1", LifecycleStage: "development"}, nil)
	cvRepo.On("UpdateLifecycle", mock.Anything, "cv1", models.LifecycleStageTesting).Return(nil)
	ltRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return([]*models.CatalogVersionPin{
		{EntityTypeVersionID: "etv1"},
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "etv1").Return(nil, fmt.Errorf("etv error"))

	_, err := svc.Promote(context.Background(), "cv1", meta.RoleAdmin, "user")
	assert.Error(t, err)
}

// getEntityTypeNamesForCV: etRepo.GetByID error (line 397) — tested via Promote
func TestPromote_ETResolveError(t *testing.T) {
	crMgr := &mockCVCRManager{}
	svc, cvRepo, pinRepo, ltRepo, etRepo, etvRepo, _ := setupCVSvc(crMgr)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{ID: "cv1", VersionLabel: "v1", LifecycleStage: "development"}, nil)
	cvRepo.On("UpdateLifecycle", mock.Anything, "cv1", models.LifecycleStageTesting).Return(nil)
	ltRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return([]*models.CatalogVersionPin{
		{EntityTypeVersionID: "etv1"},
	}, nil)
	etvRepo.On("GetByID", mock.Anything, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1"}, nil)
	etRepo.On("GetByID", mock.Anything, "et1").Return(nil, fmt.Errorf("et error"))

	_, err := svc.Promote(context.Background(), "cv1", meta.RoleAdmin, "user")
	assert.Error(t, err)
}

// EditAssociation: not found in new version (line 217)
// Already covered by TestEditAssociation_NotFound above

// ListAllAssociations: skip self entity type (line 339)
func TestListAllAssociations_SkipSelf(t *testing.T) {
	assocRepo := new(mocks.MockAssociationRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	svc := meta.NewAssociationService(assocRepo, etvRepo, attrRepo)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1"}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Association{}, nil)
	assocRepo.On("ListByTargetEntityType", mock.Anything, "et1").Return([]*models.Association{
		{ID: "a1", EntityTypeVersionID: "etv1", TargetEntityTypeID: "et1"},
	}, nil)
	// The incoming association's ETV belongs to the same entity type — should be skipped
	etvRepo.On("GetByID", mock.Anything, "etv1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1"}, nil)

	result, err := svc.ListAllAssociations(context.Background(), "et1")
	require.NoError(t, err)
	// Self-referencing association from same entity type is skipped
	assert.Len(t, result, 0)
}

// requiresDeepCopy: empty versions (line 226)
// Already covered by TestRenameEntityType_NoVersions above

// EditAttribute: not found (line 336) — already covered by TestEditAttribute_NotFound
// EditAttribute: BulkCopy error (line 279) — already covered by TestEditAttribute_BulkCopyError

// EnumService.ListValues (trivial delegator)
func TestListValues_Coverage(t *testing.T) {
	enumRepo := new(mocks.MockEnumRepo)
	evRepo := new(mocks.MockEnumValueRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	svc := meta.NewEnumService(enumRepo, evRepo, attrRepo)

	evRepo.On("ListByEnum", mock.Anything, "e1").Return([]*models.EnumValue{{ID: "v1", Value: "x"}}, nil)

	vals, err := svc.ListValues(context.Background(), "e1")
	require.NoError(t, err)
	assert.Len(t, vals, 1)
}

// EditAttribute: COW then not-found — mock new version's ListByVersion to return attrs without the target
func TestEditAttribute_NotFoundAfterCOW(t *testing.T) {
	attrRepo := new(mocks.MockAttributeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	etRepo := new(mocks.MockEntityTypeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewAttributeService(attrRepo, etvRepo, etRepo, assocRepo, nil)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	// Original version attrs (used for name conflict check)
	attrRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Attribute{
		{ID: "a1", Name: "hostname", Type: models.AttributeTypeString},
	}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Association{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, "etv1", mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, "etv1", mock.Anything).Return(nil)
	// New version's attrs don't include "hostname" — simulates a race/inconsistency
	attrRepo.On("ListByVersion", mock.Anything, mock.MatchedBy(func(id string) bool { return id != "etv1" })).Return([]*models.Attribute{}, nil)

	newName := "new-name"
	_, err := svc.EditAttribute(context.Background(), "et1", "hostname", &newName, nil, nil, nil, nil)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
}

// EditAttribute: BulkCopy assoc error after attr copy succeeds
func TestEditAttribute_AssocBulkCopyError(t *testing.T) {
	attrRepo := new(mocks.MockAttributeRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	etRepo := new(mocks.MockEntityTypeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	svc := meta.NewAttributeService(attrRepo, etvRepo, etRepo, assocRepo, nil)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Attribute{
		{ID: "a1", Name: "hostname", Type: models.AttributeTypeString},
	}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Association{}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, "etv1", mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, "etv1", mock.Anything).Return(fmt.Errorf("assoc copy error"))

	newName := "new-name"
	_, err := svc.EditAttribute(context.Background(), "et1", "hostname", &newName, nil, nil, nil, nil)
	assert.Error(t, err)
}

// EditAssociation: not found after COW — assoc missing from new version
func TestEditAssociation_NotFoundAfterCOW(t *testing.T) {
	assocRepo := new(mocks.MockAssociationRepo)
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	svc := meta.NewAssociationService(assocRepo, etvRepo, attrRepo)

	etvRepo.On("GetLatestByEntityType", mock.Anything, "et1").Return(&models.EntityTypeVersion{ID: "etv1", EntityTypeID: "et1", Version: 1}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Attribute{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "etv1").Return([]*models.Association{
		{ID: "a1", Name: "uses"},
	}, nil)
	etvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	attrRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	assocRepo.On("BulkCopyToVersion", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	// After COW, new version has no matching assoc
	assocRepo.On("ListByVersion", mock.Anything, mock.MatchedBy(func(id string) bool { return id != "etv1" })).Return([]*models.Association{}, nil)

	_, err := svc.EditAssociation(context.Background(), "et1", "uses", nil, nil, nil, nil, nil, nil)
	assert.Error(t, err)
	assert.True(t, domainerrors.IsNotFound(err))
}

// entity_type_service.go:226 (requiresDeepCopy empty versions) and :207 (deep copy Create error)
// These require WithCatalogRepos + ListByEntityTypeVersionIDs mock setup which is complex.
// The paths ARE tested in entity_type_service_test.go via the full rename flow tests.
// Per-package coverage misses them because the test setup is in a different test function group.
