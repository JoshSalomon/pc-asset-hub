package validation_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository/mocks"
	"github.com/project-catalyst/pc-asset-hub/internal/service/validation"
)

func TestCheckContainmentCycle_SelfReference(t *testing.T) {
	assocRepo := new(mocks.MockAssociationRepo)
	err := validation.CheckContainmentCycle(context.Background(), assocRepo, "et1", "et1")
	assert.Error(t, err)
}

func TestCheckContainmentCycle_NoCycle(t *testing.T) {
	assocRepo := new(mocks.MockAssociationRepo)
	assocRepo.On("GetContainmentGraph", mock.Anything).Return([]repository.ContainmentEdge{}, nil)

	err := validation.CheckContainmentCycle(context.Background(), assocRepo, "et1", "et2")
	assert.NoError(t, err)
}

func TestCheckContainmentCycle_WithCycle(t *testing.T) {
	assocRepo := new(mocks.MockAssociationRepo)
	assocRepo.On("GetContainmentGraph", mock.Anything).Return([]repository.ContainmentEdge{
		{SourceEntityTypeID: "et2", TargetEntityTypeID: "et1"},
	}, nil)

	err := validation.CheckContainmentCycle(context.Background(), assocRepo, "et1", "et2")
	assert.Error(t, err)
}

func TestCheckContainmentCycle_IndirectCycle(t *testing.T) {
	assocRepo := new(mocks.MockAssociationRepo)
	assocRepo.On("GetContainmentGraph", mock.Anything).Return([]repository.ContainmentEdge{
		{SourceEntityTypeID: "et2", TargetEntityTypeID: "et3"},
		{SourceEntityTypeID: "et3", TargetEntityTypeID: "et1"},
	}, nil)

	err := validation.CheckContainmentCycle(context.Background(), assocRepo, "et1", "et2")
	assert.Error(t, err)
}

func TestCheckContainmentCycle_NoCycleWithExistingEdges(t *testing.T) {
	assocRepo := new(mocks.MockAssociationRepo)
	assocRepo.On("GetContainmentGraph", mock.Anything).Return([]repository.ContainmentEdge{
		{SourceEntityTypeID: "et1", TargetEntityTypeID: "et3"},
		{SourceEntityTypeID: "et3", TargetEntityTypeID: "et4"},
	}, nil)

	err := validation.CheckContainmentCycle(context.Background(), assocRepo, "et1", "et2")
	assert.NoError(t, err)
}

// GetContainmentGraph returns error — propagated to caller (line 18-20)
func TestCheckContainmentCycle_GetContainmentGraphError(t *testing.T) {
	assocRepo := new(mocks.MockAssociationRepo)
	assocRepo.On("GetContainmentGraph", mock.Anything).Return(
		[]repository.ContainmentEdge(nil), fmt.Errorf("database error"),
	)

	err := validation.CheckContainmentCycle(context.Background(), assocRepo, "et1", "et2")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database error")
}

// Diamond-shaped graph where the DFS visits a node twice (line 39-41)
// Graph: et2 -> et3, et2 -> et4, et3 -> et5, et4 -> et5
// Adding et1 -> et2 triggers DFS from et2; et5 is reached via both et3 and et4,
// so the second visit hits visited[et5]=true and returns false early.
func TestCheckContainmentCycle_DiamondGraph_VisitedNodeSkipped(t *testing.T) {
	assocRepo := new(mocks.MockAssociationRepo)
	assocRepo.On("GetContainmentGraph", mock.Anything).Return([]repository.ContainmentEdge{
		{SourceEntityTypeID: "et2", TargetEntityTypeID: "et3"},
		{SourceEntityTypeID: "et2", TargetEntityTypeID: "et4"},
		{SourceEntityTypeID: "et3", TargetEntityTypeID: "et5"},
		{SourceEntityTypeID: "et4", TargetEntityTypeID: "et5"},
	}, nil)

	err := validation.CheckContainmentCycle(context.Background(), assocRepo, "et1", "et2")
	assert.NoError(t, err)
}
