package validation_test

import (
	"context"
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
