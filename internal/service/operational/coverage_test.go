package operational_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository/mocks"
	"github.com/project-catalyst/pc-asset-hub/internal/service/operational"
)

var dbErr = errors.New("db error")

// ============================================================================
// UpdateInstance error branches
// ============================================================================

func TestUpdateInstance_GetByIDError(t *testing.T) {
	svc, instRepo, _, _ := setupSvc()

	instRepo.On("GetByID", mock.Anything, "inst1").Return(nil, dbErr)

	_, err := svc.UpdateInstance(context.Background(), "inst1", 1)
	assert.ErrorIs(t, err, dbErr)
}

func TestUpdateInstance_UpdateRepoError(t *testing.T) {
	svc, instRepo, _, _ := setupSvc()

	instRepo.On("GetByID", mock.Anything, "inst1").Return(&models.EntityInstance{ID: "inst1", Version: 1}, nil)
	instRepo.On("Update", mock.Anything, mock.Anything).Return(dbErr)

	_, err := svc.UpdateInstance(context.Background(), "inst1", 1)
	assert.ErrorIs(t, err, dbErr)
}

// ============================================================================
// CreateInstance error branches
// ============================================================================

func TestCreateInstance_CreateRepoError(t *testing.T) {
	svc, instRepo, cvRepo, _ := setupSvc()

	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{ID: "cv1"}, nil)
	instRepo.On("Create", mock.Anything, mock.Anything).Return(dbErr)

	_, err := svc.CreateInstance(context.Background(), "et1", "cv1", "", "model-1", "desc", nil)
	assert.ErrorIs(t, err, dbErr)
}

// ============================================================================
// CascadeDelete error branches
// ============================================================================

func TestCascadeDelete_ListByParentError(t *testing.T) {
	svc, instRepo, _, _ := setupSvc()

	instRepo.On("ListByParent", mock.Anything, "parent1", mock.Anything).Return([]*models.EntityInstance(nil), 0, dbErr)

	err := svc.CascadeDelete(context.Background(), "parent1")
	assert.ErrorIs(t, err, dbErr)
}

func TestCascadeDelete_SoftDeleteSelfError(t *testing.T) {
	svc, instRepo, _, _ := setupSvc()

	instRepo.On("ListByParent", mock.Anything, "parent1", mock.Anything).Return([]*models.EntityInstance{}, 0, nil)
	instRepo.On("SoftDelete", mock.Anything, "parent1").Return(dbErr)

	err := svc.CascadeDelete(context.Background(), "parent1")
	assert.ErrorIs(t, err, dbErr)
}

func TestCascadeDelete_RecursiveChildError(t *testing.T) {
	svc, instRepo, _, _ := setupSvc()

	instRepo.On("ListByParent", mock.Anything, "parent1", mock.Anything).Return([]*models.EntityInstance{
		{ID: "child1"},
	}, 1, nil)
	instRepo.On("ListByParent", mock.Anything, "child1", mock.Anything).Return([]*models.EntityInstance(nil), 0, dbErr)

	err := svc.CascadeDelete(context.Background(), "parent1")
	assert.ErrorIs(t, err, dbErr)
}

// ============================================================================
// CreateContainedInstance error branches
// ============================================================================

func TestCreateContainedInstance_CvRepoError(t *testing.T) {
	svc, instRepo, cvRepo, _ := setupSvc()

	instRepo.On("GetByID", mock.Anything, "parent1").Return(&models.EntityInstance{ID: "parent1"}, nil)
	cvRepo.On("GetByID", mock.Anything, "cv1").Return(nil, dbErr)

	_, err := svc.CreateContainedInstance(context.Background(), "parent1", "et1", "cv1", "child-1", "")
	assert.ErrorIs(t, err, dbErr)
}

func TestCreateContainedInstance_InstRepoCreateError(t *testing.T) {
	svc, instRepo, cvRepo, _ := setupSvc()

	instRepo.On("GetByID", mock.Anything, "parent1").Return(&models.EntityInstance{ID: "parent1"}, nil)
	cvRepo.On("GetByID", mock.Anything, "cv1").Return(&models.CatalogVersion{ID: "cv1"}, nil)
	instRepo.On("Create", mock.Anything, mock.Anything).Return(dbErr)

	_, err := svc.CreateContainedInstance(context.Background(), "parent1", "et1", "cv1", "child-1", "")
	assert.ErrorIs(t, err, dbErr)
}

// ============================================================================
// GetForwardReferences / GetReverseReferences error branches
// ============================================================================

func TestGetForwardReferences_Error(t *testing.T) {
	instRepo := new(mocks.MockEntityInstanceRepo)
	iavRepo := new(mocks.MockInstanceAttributeValueRepo)
	cvRepo := new(mocks.MockCatalogVersionRepo)
	linkRepo := new(mocks.MockAssociationLinkRepo)
	svc := operational.NewEntityInstanceService(instRepo, iavRepo, nil, cvRepo, linkRepo)

	linkRepo.On("GetForwardRefs", mock.Anything, "inst1").Return([]*models.AssociationLink(nil), dbErr)

	_, err := svc.GetForwardReferences(context.Background(), "inst1")
	assert.ErrorIs(t, err, dbErr)
}

func TestGetReverseReferences_Error(t *testing.T) {
	instRepo := new(mocks.MockEntityInstanceRepo)
	iavRepo := new(mocks.MockInstanceAttributeValueRepo)
	cvRepo := new(mocks.MockCatalogVersionRepo)
	linkRepo := new(mocks.MockAssociationLinkRepo)
	svc := operational.NewEntityInstanceService(instRepo, iavRepo, nil, cvRepo, linkRepo)

	linkRepo.On("GetReverseRefs", mock.Anything, "inst1").Return([]*models.AssociationLink(nil), dbErr)

	_, err := svc.GetReverseReferences(context.Background(), "inst1")
	assert.ErrorIs(t, err, dbErr)
}

// ============================================================================
// DeleteInstance error branch
// ============================================================================

func TestDeleteInstance_Error(t *testing.T) {
	svc, instRepo, _, _ := setupSvc()

	instRepo.On("SoftDelete", mock.Anything, "inst1").Return(dbErr)

	err := svc.DeleteInstance(context.Background(), "inst1")
	assert.ErrorIs(t, err, dbErr)
}

// ============================================================================
// GetInstance error branch
// ============================================================================

func TestGetInstance_Error(t *testing.T) {
	svc, instRepo, _, _ := setupSvc()

	instRepo.On("GetByID", mock.Anything, "inst1").Return(nil, dbErr)

	_, err := svc.GetInstance(context.Background(), "inst1")
	assert.ErrorIs(t, err, dbErr)
}

// ============================================================================
// ListInstances error branch
// ============================================================================

func TestListInstances_Error(t *testing.T) {
	svc, instRepo, _, _ := setupSvc()

	instRepo.On("List", mock.Anything, "et1", "cv1", mock.Anything).Return([]*models.EntityInstance(nil), 0, dbErr)

	_, _, err := svc.ListInstances(context.Background(), "et1", "cv1", models.ListParams{})
	assert.ErrorIs(t, err, dbErr)
}

// ============================================================================
// ListContainedInstances error branch
// ============================================================================

func TestListContainedInstances_Error(t *testing.T) {
	svc, instRepo, _, _ := setupSvc()

	instRepo.On("ListByParent", mock.Anything, "parent1", mock.Anything).Return([]*models.EntityInstance(nil), 0, dbErr)

	_, _, err := svc.ListContainedInstances(context.Background(), "parent1", models.ListParams{})
	assert.ErrorIs(t, err, dbErr)
}
