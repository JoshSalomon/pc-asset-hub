package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
)

// MockEntityInstanceRepo mocks EntityInstanceRepository.
type MockEntityInstanceRepo struct{ mock.Mock }

func (m *MockEntityInstanceRepo) Create(ctx context.Context, inst *models.EntityInstance) error {
	return m.Called(ctx, inst).Error(0)
}
func (m *MockEntityInstanceRepo) GetByID(ctx context.Context, id string) (*models.EntityInstance, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.EntityInstance), args.Error(1)
}
func (m *MockEntityInstanceRepo) GetByNameAndParent(ctx context.Context, entityTypeID, catalogID, parentInstanceID, name string) (*models.EntityInstance, error) {
	args := m.Called(ctx, entityTypeID, catalogID, parentInstanceID, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.EntityInstance), args.Error(1)
}
func (m *MockEntityInstanceRepo) List(ctx context.Context, entityTypeID, catalogID string, params models.ListParams) ([]*models.EntityInstance, int, error) {
	args := m.Called(ctx, entityTypeID, catalogID, params)
	return args.Get(0).([]*models.EntityInstance), args.Int(1), args.Error(2)
}
func (m *MockEntityInstanceRepo) ListByParent(ctx context.Context, parentInstanceID string, params models.ListParams) ([]*models.EntityInstance, int, error) {
	args := m.Called(ctx, parentInstanceID, params)
	return args.Get(0).([]*models.EntityInstance), args.Int(1), args.Error(2)
}
func (m *MockEntityInstanceRepo) Update(ctx context.Context, inst *models.EntityInstance) error {
	return m.Called(ctx, inst).Error(0)
}
func (m *MockEntityInstanceRepo) SoftDelete(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockEntityInstanceRepo) DeleteByCatalogID(ctx context.Context, catalogID string) error {
	return m.Called(ctx, catalogID).Error(0)
}

// MockCatalogRepo mocks CatalogRepository.
type MockCatalogRepo struct{ mock.Mock }

func (m *MockCatalogRepo) Create(ctx context.Context, catalog *models.Catalog) error {
	return m.Called(ctx, catalog).Error(0)
}
func (m *MockCatalogRepo) GetByName(ctx context.Context, name string) (*models.Catalog, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Catalog), args.Error(1)
}
func (m *MockCatalogRepo) GetByID(ctx context.Context, id string) (*models.Catalog, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Catalog), args.Error(1)
}
func (m *MockCatalogRepo) List(ctx context.Context, params models.ListParams) ([]*models.Catalog, int, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]*models.Catalog), args.Int(1), args.Error(2)
}
func (m *MockCatalogRepo) Delete(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockCatalogRepo) UpdateValidationStatus(ctx context.Context, id string, status models.ValidationStatus) error {
	return m.Called(ctx, id, status).Error(0)
}

// MockInstanceAttributeValueRepo mocks InstanceAttributeValueRepository.
type MockInstanceAttributeValueRepo struct{ mock.Mock }

func (m *MockInstanceAttributeValueRepo) SetValues(ctx context.Context, values []*models.InstanceAttributeValue) error {
	return m.Called(ctx, values).Error(0)
}
func (m *MockInstanceAttributeValueRepo) GetCurrentValues(ctx context.Context, instanceID string) ([]*models.InstanceAttributeValue, error) {
	args := m.Called(ctx, instanceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.InstanceAttributeValue), args.Error(1)
}
func (m *MockInstanceAttributeValueRepo) GetValuesForVersion(ctx context.Context, instanceID string, version int) ([]*models.InstanceAttributeValue, error) {
	args := m.Called(ctx, instanceID, version)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.InstanceAttributeValue), args.Error(1)
}

// MockAssociationLinkRepo mocks AssociationLinkRepository.
type MockAssociationLinkRepo struct{ mock.Mock }

func (m *MockAssociationLinkRepo) Create(ctx context.Context, link *models.AssociationLink) error {
	return m.Called(ctx, link).Error(0)
}
func (m *MockAssociationLinkRepo) Delete(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockAssociationLinkRepo) GetForwardRefs(ctx context.Context, sourceInstanceID string) ([]*models.AssociationLink, error) {
	args := m.Called(ctx, sourceInstanceID)
	return args.Get(0).([]*models.AssociationLink), args.Error(1)
}
func (m *MockAssociationLinkRepo) GetReverseRefs(ctx context.Context, targetInstanceID string) ([]*models.AssociationLink, error) {
	args := m.Called(ctx, targetInstanceID)
	return args.Get(0).([]*models.AssociationLink), args.Error(1)
}
