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
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", VersionLabel: "v1.0", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)

	cv, err := svc.GetCatalogVersion(context.Background(), "cv1")
	require.NoError(t, err)
	assert.Equal(t, "v1.0", cv.VersionLabel)
}

func TestListCatalogVersions(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil)

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
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageProduction,
	}, nil)

	err := svc.Promote(context.Background(), "cv1", meta.RoleAdmin, "admin")
	assert.Error(t, err)
}

func TestPromote_InvalidStage(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStage("invalid"),
	}, nil)

	err := svc.Promote(context.Background(), "cv1", meta.RoleAdmin, "admin")
	assert.Error(t, err)
}

func TestPromote_ROForbidden(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)

	err := svc.Promote(context.Background(), "cv1", meta.RoleRO, "ro")
	assert.Error(t, err)
}

func TestDemote_DevelopmentFails(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)

	err := svc.Demote(context.Background(), "cv1", meta.RoleAdmin, "admin", models.LifecycleStageDevelopment)
	assert.Error(t, err)
}

func TestDemote_TestingInvalidTarget(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageTesting,
	}, nil)

	err := svc.Demote(context.Background(), "cv1", meta.RoleAdmin, "admin", models.LifecycleStageProduction)
	assert.Error(t, err)
}

func TestDemote_TestingROForbidden(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageTesting,
	}, nil)

	err := svc.Demote(context.Background(), "cv1", meta.RoleRO, "ro", models.LifecycleStageDevelopment)
	assert.Error(t, err)
}

func TestDemote_ProductionInvalidTarget(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", LifecycleStage: models.LifecycleStageProduction,
	}, nil)

	err := svc.Demote(context.Background(), "cv1", meta.RoleSuperAdmin, "sa", models.LifecycleStage("invalid"))
	assert.Error(t, err)
}

func TestDemote_InvalidStage(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", nil, nil, nil)

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

	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, ltRepo, crMgr, "assethub", nil, etRepo, etvRepo)

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

	err := svc.Promote(context.Background(), "cv1", meta.RoleRW, "admin")
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

	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, ltRepo, crMgr, "assethub", nil, etRepo, etvRepo)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", VersionLabel: "Release 1", LifecycleStage: models.LifecycleStageTesting,
	}, nil)
	cvRepo.On("UpdateLifecycle", mock.Anything, "cv1", models.LifecycleStageProduction).Return(nil)
	ltRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	pinRepo.On("ListByCatalogVersion", mock.Anything, "cv1").Return([]*models.CatalogVersionPin{}, nil)
	crMgr.On("CreateOrUpdate", mock.Anything, mock.MatchedBy(func(spec meta.CatalogVersionCRSpec) bool {
		return spec.LifecycleStage == "production"
	})).Return(nil)

	err := svc.Promote(context.Background(), "cv1", meta.RoleAdmin, "admin")
	require.NoError(t, err)
	crMgr.AssertCalled(t, "CreateOrUpdate", mock.Anything, mock.Anything)
}

// T-CV.25: Demote testing→development calls crManager.Delete
func TestTCV25_DemoteTestingToDevCallsDelete(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)
	ltRepo := new(mocks.MockLifecycleTransitionRepo)
	crMgr := new(mockCRManager)

	svc := meta.NewCatalogVersionService(cvRepo, nil, ltRepo, crMgr, "assethub", nil, nil, nil)

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

	svc := meta.NewCatalogVersionService(cvRepo, pinRepo, ltRepo, crMgr, "assethub", nil, etRepo, etvRepo)

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

	svc := meta.NewCatalogVersionService(cvRepo, nil, ltRepo, nil, "", nil, nil, nil)

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{
		ID: "cv1", VersionLabel: "v1", LifecycleStage: models.LifecycleStageDevelopment,
	}, nil)
	cvRepo.On("UpdateLifecycle", mock.Anything, "cv1", models.LifecycleStageTesting).Return(nil)
	ltRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	err := svc.Promote(context.Background(), "cv1", meta.RoleRW, "admin")
	require.NoError(t, err)
	cvRepo.AssertCalled(t, "UpdateLifecycle", mock.Anything, "cv1", models.LifecycleStageTesting)
}

// T-CV.28: ListCatalogVersions with allowedStages=["production"] returns only production versions
func TestTCV28_ListFilteredByAllowedStages(t *testing.T) {
	cvRepo := new(mocks.MockCatalogVersionRepo)

	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", []string{"production"}, nil, nil)

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

	svc := meta.NewCatalogVersionService(cvRepo, nil, nil, nil, "", []string{"production"}, nil, nil)

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
