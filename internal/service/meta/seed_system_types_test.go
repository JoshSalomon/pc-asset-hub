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

func TestSeedSystemTypes_AllAlreadyExist(t *testing.T) {
	svc, _, _, _ := newTypeDefSvc()
	tdRepo := new(mocks.MockTypeDefinitionRepo)

	// All types already exist
	for _, st := range systemTypes {
		tdRepo.On("GetByName", mock.Anything, st.Name).Return(&models.TypeDefinition{
			ID: "existing-" + st.Name, Name: st.Name, BaseType: st.BaseType,
		}, nil)
	}

	err := SeedSystemTypes(context.Background(), svc, tdRepo)
	assert.NoError(t, err)
	tdRepo.AssertExpectations(t)
}

func TestSeedSystemTypes_CreatesNew(t *testing.T) {
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	svc := NewTypeDefinitionService(tdRepo, tdvRepo, attrRepo)

	// All types are not found -> need to create
	for _, st := range systemTypes {
		tdRepo.On("GetByName", mock.Anything, st.Name).Return(nil, domainerrors.NewNotFound("TypeDefinition", st.Name))
	}
	tdRepo.On("Create", mock.Anything, mock.MatchedBy(func(td *models.TypeDefinition) bool {
		return td.System == true
	})).Return(nil)
	tdvRepo.On("Create", mock.Anything, mock.MatchedBy(func(tdv *models.TypeDefinitionVersion) bool {
		return tdv.VersionNumber == 1
	})).Return(nil)

	err := SeedSystemTypes(context.Background(), svc, tdRepo)
	assert.NoError(t, err)

	// Should have called Create for each system type
	tdRepo.AssertNumberOfCalls(t, "GetByName", len(systemTypes))
	tdRepo.AssertNumberOfCalls(t, "Create", len(systemTypes))
}

func TestSeedSystemTypes_GetByNameNonNotFoundError(t *testing.T) {
	svc, _, _, _ := newTypeDefSvc()
	tdRepo := new(mocks.MockTypeDefinitionRepo)

	// First type returns a non-NotFound error
	tdRepo.On("GetByName", mock.Anything, systemTypes[0].Name).Return(nil, errors.New("db connection error"))

	err := SeedSystemTypes(context.Background(), svc, tdRepo)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db connection error")
}

func TestSeedSystemTypes_CreateSystemTypeDefinitionError(t *testing.T) {
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	svc := NewTypeDefinitionService(tdRepo, tdvRepo, attrRepo)

	// First type not found, then Create fails
	tdRepo.On("GetByName", mock.Anything, systemTypes[0].Name).Return(nil, domainerrors.NewNotFound("TypeDefinition", systemTypes[0].Name))
	tdRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("create failed"))

	err := SeedSystemTypes(context.Background(), svc, tdRepo)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create failed")
}

func TestSeedSystemTypes_PartialExist(t *testing.T) {
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	svc := NewTypeDefinitionService(tdRepo, tdvRepo, attrRepo)

	// First type exists, rest need creation
	for i, st := range systemTypes {
		if i == 0 {
			tdRepo.On("GetByName", mock.Anything, st.Name).Return(&models.TypeDefinition{
				ID: "existing-" + st.Name, Name: st.Name, BaseType: st.BaseType,
			}, nil)
		} else {
			tdRepo.On("GetByName", mock.Anything, st.Name).Return(nil, domainerrors.NewNotFound("TypeDefinition", st.Name))
		}
	}
	tdRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	tdvRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	err := SeedSystemTypes(context.Background(), svc, tdRepo)
	assert.NoError(t, err)
}
