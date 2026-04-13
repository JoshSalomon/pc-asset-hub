package operational

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository/mocks"
)

func TestResolveBaseTypes_Success(t *testing.T) {
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	ctx := context.Background()

	attrs := []*models.Attribute{
		{ID: "a1", Name: "name", TypeDefinitionVersionID: "tdv-1"},
		{ID: "a2", Name: "count", TypeDefinitionVersionID: "tdv-2"},
	}

	tdvRepo.On("GetByID", mock.Anything, "tdv-1").Return(&models.TypeDefinitionVersion{
		ID: "tdv-1", TypeDefinitionID: "td-1",
	}, nil)
	tdvRepo.On("GetByID", mock.Anything, "tdv-2").Return(&models.TypeDefinitionVersion{
		ID: "tdv-2", TypeDefinitionID: "td-2",
	}, nil)
	tdRepo.On("GetByID", mock.Anything, "td-1").Return(&models.TypeDefinition{
		ID: "td-1", BaseType: models.BaseTypeString,
	}, nil)
	tdRepo.On("GetByID", mock.Anything, "td-2").Return(&models.TypeDefinition{
		ID: "td-2", BaseType: models.BaseTypeInteger,
	}, nil)

	result, err := ResolveBaseTypes(ctx, attrs, tdvRepo, tdRepo)
	assert.NoError(t, err)
	assert.Equal(t, "string", result["a1"])
	assert.Equal(t, "integer", result["a2"])
}

func TestResolveBaseTypes_TdvGetByIDError(t *testing.T) {
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	ctx := context.Background()

	attrs := []*models.Attribute{
		{ID: "a1", Name: "name", TypeDefinitionVersionID: "tdv-missing"},
	}

	tdvRepo.On("GetByID", mock.Anything, "tdv-missing").Return(nil, errors.New("tdv not found"))

	_, err := ResolveBaseTypes(ctx, attrs, tdvRepo, tdRepo)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve type for attribute name")
}

func TestResolveBaseTypes_TdGetByIDError(t *testing.T) {
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	ctx := context.Background()

	attrs := []*models.Attribute{
		{ID: "a1", Name: "name", TypeDefinitionVersionID: "tdv-1"},
	}

	tdvRepo.On("GetByID", mock.Anything, "tdv-1").Return(&models.TypeDefinitionVersion{
		ID: "tdv-1", TypeDefinitionID: "td-missing",
	}, nil)
	tdRepo.On("GetByID", mock.Anything, "td-missing").Return(nil, errors.New("td not found"))

	_, err := ResolveBaseTypes(ctx, attrs, tdvRepo, tdRepo)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve type definition for attribute name")
}

func TestResolveBaseTypes_EmptyAttrs(t *testing.T) {
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	ctx := context.Background()

	result, err := ResolveBaseTypes(ctx, []*models.Attribute{}, tdvRepo, tdRepo)
	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestResolveBaseTypes_CachesLookups(t *testing.T) {
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	ctx := context.Background()

	// Two attributes sharing the same TypeDefinitionVersionID
	attrs := []*models.Attribute{
		{ID: "a1", Name: "name1", TypeDefinitionVersionID: "tdv-1"},
		{ID: "a2", Name: "name2", TypeDefinitionVersionID: "tdv-1"},
	}

	tdvRepo.On("GetByID", mock.Anything, "tdv-1").Return(&models.TypeDefinitionVersion{
		ID: "tdv-1", TypeDefinitionID: "td-1",
	}, nil).Once() // Should only be called once due to cache
	tdRepo.On("GetByID", mock.Anything, "td-1").Return(&models.TypeDefinition{
		ID: "td-1", BaseType: models.BaseTypeString,
	}, nil).Once()

	result, err := ResolveBaseTypes(ctx, attrs, tdvRepo, tdRepo)
	assert.NoError(t, err)
	assert.Equal(t, "string", result["a1"])
	assert.Equal(t, "string", result["a2"])
	tdvRepo.AssertNumberOfCalls(t, "GetByID", 1)
	tdRepo.AssertNumberOfCalls(t, "GetByID", 1)
}

func TestResolveAttrTypeInfo_Success(t *testing.T) {
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	ctx := context.Background()

	attrs := []*models.Attribute{
		{ID: "a1", Name: "status", TypeDefinitionVersionID: "tdv-1"},
	}

	tdvRepo.On("GetByID", mock.Anything, "tdv-1").Return(&models.TypeDefinitionVersion{
		ID: "tdv-1", TypeDefinitionID: "td-1", Constraints: map[string]any{"values": []any{"active", "inactive"}},
	}, nil)
	tdRepo.On("GetByID", mock.Anything, "td-1").Return(&models.TypeDefinition{
		ID: "td-1", BaseType: models.BaseTypeEnum,
	}, nil)

	result, err := ResolveAttrTypeInfo(ctx, attrs, tdvRepo, tdRepo)
	assert.NoError(t, err)
	assert.Equal(t, models.BaseTypeEnum, result["a1"].BaseType)
	assert.Contains(t, result["a1"].Constraints, "values")
}

func TestResolveAttrTypeInfo_TdvGetByIDError(t *testing.T) {
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	ctx := context.Background()

	attrs := []*models.Attribute{
		{ID: "a1", Name: "status", TypeDefinitionVersionID: "tdv-missing"},
	}

	tdvRepo.On("GetByID", mock.Anything, "tdv-missing").Return(nil, errors.New("tdv error"))

	_, err := ResolveAttrTypeInfo(ctx, attrs, tdvRepo, tdRepo)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve type for attribute status")
}

func TestResolveAttrTypeInfo_TdGetByIDError(t *testing.T) {
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	ctx := context.Background()

	attrs := []*models.Attribute{
		{ID: "a1", Name: "status", TypeDefinitionVersionID: "tdv-1"},
	}

	tdvRepo.On("GetByID", mock.Anything, "tdv-1").Return(&models.TypeDefinitionVersion{
		ID: "tdv-1", TypeDefinitionID: "td-missing",
	}, nil)
	tdRepo.On("GetByID", mock.Anything, "td-missing").Return(nil, errors.New("td error"))

	_, err := ResolveAttrTypeInfo(ctx, attrs, tdvRepo, tdRepo)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve type definition for attribute status")
}

func TestResolveAttrTypeInfo_EmptyAttrs(t *testing.T) {
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	ctx := context.Background()

	result, err := ResolveAttrTypeInfo(ctx, []*models.Attribute{}, tdvRepo, tdRepo)
	assert.NoError(t, err)
	assert.Empty(t, result)
}

func TestResolveAttrTypeInfo_CachesLookups(t *testing.T) {
	tdvRepo := new(mocks.MockTypeDefinitionVersionRepo)
	tdRepo := new(mocks.MockTypeDefinitionRepo)
	ctx := context.Background()

	attrs := []*models.Attribute{
		{ID: "a1", Name: "name1", TypeDefinitionVersionID: "tdv-1"},
		{ID: "a2", Name: "name2", TypeDefinitionVersionID: "tdv-1"},
	}

	tdvRepo.On("GetByID", mock.Anything, "tdv-1").Return(&models.TypeDefinitionVersion{
		ID: "tdv-1", TypeDefinitionID: "td-1", Constraints: map[string]any{},
	}, nil).Once()
	tdRepo.On("GetByID", mock.Anything, "td-1").Return(&models.TypeDefinition{
		ID: "td-1", BaseType: models.BaseTypeString,
	}, nil).Once()

	result, err := ResolveAttrTypeInfo(ctx, attrs, tdvRepo, tdRepo)
	assert.NoError(t, err)
	assert.Equal(t, models.BaseTypeString, result["a1"].BaseType)
	assert.Equal(t, models.BaseTypeString, result["a2"].BaseType)
	tdvRepo.AssertNumberOfCalls(t, "GetByID", 1)
	tdRepo.AssertNumberOfCalls(t, "GetByID", 1)
}
