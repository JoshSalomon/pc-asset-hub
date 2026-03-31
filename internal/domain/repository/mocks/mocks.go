package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"

	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository"
)

// MockEntityTypeRepo mocks EntityTypeRepository.
type MockEntityTypeRepo struct{ mock.Mock }

func (m *MockEntityTypeRepo) Create(ctx context.Context, et *models.EntityType) error {
	return m.Called(ctx, et).Error(0)
}
func (m *MockEntityTypeRepo) GetByID(ctx context.Context, id string) (*models.EntityType, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.EntityType), args.Error(1)
}
func (m *MockEntityTypeRepo) GetByName(ctx context.Context, name string) (*models.EntityType, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.EntityType), args.Error(1)
}
func (m *MockEntityTypeRepo) List(ctx context.Context, params models.ListParams) ([]*models.EntityType, int, error) {
	args := m.Called(ctx, params)
	return args.Get(0).([]*models.EntityType), args.Int(1), args.Error(2)
}
func (m *MockEntityTypeRepo) Update(ctx context.Context, et *models.EntityType) error {
	return m.Called(ctx, et).Error(0)
}
func (m *MockEntityTypeRepo) Delete(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

// MockEntityTypeVersionRepo mocks EntityTypeVersionRepository.
type MockEntityTypeVersionRepo struct{ mock.Mock }

func (m *MockEntityTypeVersionRepo) Create(ctx context.Context, etv *models.EntityTypeVersion) error {
	return m.Called(ctx, etv).Error(0)
}
func (m *MockEntityTypeVersionRepo) GetByID(ctx context.Context, id string) (*models.EntityTypeVersion, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.EntityTypeVersion), args.Error(1)
}
func (m *MockEntityTypeVersionRepo) GetByEntityTypeAndVersion(ctx context.Context, entityTypeID string, version int) (*models.EntityTypeVersion, error) {
	args := m.Called(ctx, entityTypeID, version)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.EntityTypeVersion), args.Error(1)
}
func (m *MockEntityTypeVersionRepo) GetLatestByEntityType(ctx context.Context, entityTypeID string) (*models.EntityTypeVersion, error) {
	args := m.Called(ctx, entityTypeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.EntityTypeVersion), args.Error(1)
}
func (m *MockEntityTypeVersionRepo) GetLatestByEntityTypes(ctx context.Context, entityTypeIDs []string) (map[string]*models.EntityTypeVersion, error) {
	args := m.Called(ctx, entityTypeIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]*models.EntityTypeVersion), args.Error(1)
}
func (m *MockEntityTypeVersionRepo) ListByEntityType(ctx context.Context, entityTypeID string) ([]*models.EntityTypeVersion, error) {
	args := m.Called(ctx, entityTypeID)
	return args.Get(0).([]*models.EntityTypeVersion), args.Error(1)
}

// MockAttributeRepo mocks AttributeRepository.
type MockAttributeRepo struct{ mock.Mock }

func (m *MockAttributeRepo) Create(ctx context.Context, attr *models.Attribute) error {
	return m.Called(ctx, attr).Error(0)
}
func (m *MockAttributeRepo) GetByID(ctx context.Context, id string) (*models.Attribute, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Attribute), args.Error(1)
}
func (m *MockAttributeRepo) ListByVersion(ctx context.Context, entityTypeVersionID string) ([]*models.Attribute, error) {
	args := m.Called(ctx, entityTypeVersionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Attribute), args.Error(1)
}
func (m *MockAttributeRepo) Update(ctx context.Context, attr *models.Attribute) error {
	return m.Called(ctx, attr).Error(0)
}
func (m *MockAttributeRepo) Delete(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockAttributeRepo) Reorder(ctx context.Context, entityTypeVersionID string, orderedIDs []string) error {
	return m.Called(ctx, entityTypeVersionID, orderedIDs).Error(0)
}
func (m *MockAttributeRepo) BulkCopyToVersion(ctx context.Context, fromVersionID string, toVersionID string) error {
	return m.Called(ctx, fromVersionID, toVersionID).Error(0)
}

// MockAssociationRepo mocks AssociationRepository.
type MockAssociationRepo struct{ mock.Mock }

func (m *MockAssociationRepo) Create(ctx context.Context, assoc *models.Association) error {
	return m.Called(ctx, assoc).Error(0)
}
func (m *MockAssociationRepo) GetByID(ctx context.Context, id string) (*models.Association, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Association), args.Error(1)
}
func (m *MockAssociationRepo) ListByVersion(ctx context.Context, entityTypeVersionID string) ([]*models.Association, error) {
	args := m.Called(ctx, entityTypeVersionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Association), args.Error(1)
}
func (m *MockAssociationRepo) ListByTargetEntityType(ctx context.Context, targetEntityTypeID string) ([]*models.Association, error) {
	args := m.Called(ctx, targetEntityTypeID)
	return args.Get(0).([]*models.Association), args.Error(1)
}
func (m *MockAssociationRepo) Update(ctx context.Context, assoc *models.Association) error {
	return m.Called(ctx, assoc).Error(0)
}
func (m *MockAssociationRepo) Delete(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockAssociationRepo) BulkCopyToVersion(ctx context.Context, fromVersionID string, toVersionID string) error {
	return m.Called(ctx, fromVersionID, toVersionID).Error(0)
}
func (m *MockAssociationRepo) GetContainmentGraph(ctx context.Context) ([]repository.ContainmentEdge, error) {
	args := m.Called(ctx)
	return args.Get(0).([]repository.ContainmentEdge), args.Error(1)
}

// MockEnumRepo mocks EnumRepository.
type MockEnumRepo struct{ mock.Mock }

func (m *MockEnumRepo) Create(ctx context.Context, e *models.Enum) error {
	return m.Called(ctx, e).Error(0)
}
func (m *MockEnumRepo) GetByID(ctx context.Context, id string) (*models.Enum, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Enum), args.Error(1)
}
func (m *MockEnumRepo) GetByName(ctx context.Context, name string) (*models.Enum, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Enum), args.Error(1)
}
func (m *MockEnumRepo) List(ctx context.Context, params models.ListParams) ([]*models.Enum, int, error) {
	args := m.Called(ctx, params)
	return args.Get(0).([]*models.Enum), args.Int(1), args.Error(2)
}
func (m *MockEnumRepo) Update(ctx context.Context, e *models.Enum) error {
	return m.Called(ctx, e).Error(0)
}
func (m *MockEnumRepo) Delete(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

// MockEnumValueRepo mocks EnumValueRepository.
type MockEnumValueRepo struct{ mock.Mock }

func (m *MockEnumValueRepo) Create(ctx context.Context, ev *models.EnumValue) error {
	return m.Called(ctx, ev).Error(0)
}
func (m *MockEnumValueRepo) ListByEnum(ctx context.Context, enumID string) ([]*models.EnumValue, error) {
	args := m.Called(ctx, enumID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.EnumValue), args.Error(1)
}
func (m *MockEnumValueRepo) Delete(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockEnumValueRepo) Reorder(ctx context.Context, enumID string, orderedIDs []string) error {
	return m.Called(ctx, enumID, orderedIDs).Error(0)
}

// MockCatalogVersionRepo mocks CatalogVersionRepository.
type MockCatalogVersionRepo struct{ mock.Mock }

func (m *MockCatalogVersionRepo) Create(ctx context.Context, cv *models.CatalogVersion) error {
	return m.Called(ctx, cv).Error(0)
}
func (m *MockCatalogVersionRepo) GetByID(ctx context.Context, id string) (*models.CatalogVersion, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.CatalogVersion), args.Error(1)
}
func (m *MockCatalogVersionRepo) GetByLabel(ctx context.Context, label string) (*models.CatalogVersion, error) {
	args := m.Called(ctx, label)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.CatalogVersion), args.Error(1)
}
func (m *MockCatalogVersionRepo) List(ctx context.Context, params models.ListParams) ([]*models.CatalogVersion, int, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*models.CatalogVersion), args.Int(1), args.Error(2)
}
func (m *MockCatalogVersionRepo) UpdateLifecycle(ctx context.Context, id string, stage models.LifecycleStage) error {
	return m.Called(ctx, id, stage).Error(0)
}
func (m *MockCatalogVersionRepo) Update(ctx context.Context, cv *models.CatalogVersion) error {
	return m.Called(ctx, cv).Error(0)
}
func (m *MockCatalogVersionRepo) Delete(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

// MockCatalogVersionPinRepo mocks CatalogVersionPinRepository.
type MockCatalogVersionPinRepo struct{ mock.Mock }

func (m *MockCatalogVersionPinRepo) Create(ctx context.Context, pin *models.CatalogVersionPin) error {
	return m.Called(ctx, pin).Error(0)
}
func (m *MockCatalogVersionPinRepo) GetByID(ctx context.Context, id string) (*models.CatalogVersionPin, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.CatalogVersionPin), args.Error(1)
}
func (m *MockCatalogVersionPinRepo) Update(ctx context.Context, pin *models.CatalogVersionPin) error {
	return m.Called(ctx, pin).Error(0)
}
func (m *MockCatalogVersionPinRepo) ListByCatalogVersion(ctx context.Context, catalogVersionID string) ([]*models.CatalogVersionPin, error) {
	args := m.Called(ctx, catalogVersionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.CatalogVersionPin), args.Error(1)
}
func (m *MockCatalogVersionPinRepo) ListByEntityTypeVersionIDs(ctx context.Context, etvIDs []string) ([]*models.CatalogVersionPin, error) {
	args := m.Called(ctx, etvIDs)
	return args.Get(0).([]*models.CatalogVersionPin), args.Error(1)
}
func (m *MockCatalogVersionPinRepo) Delete(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

// MockLifecycleTransitionRepo mocks LifecycleTransitionRepository.
type MockLifecycleTransitionRepo struct{ mock.Mock }

func (m *MockLifecycleTransitionRepo) Create(ctx context.Context, lt *models.LifecycleTransition) error {
	return m.Called(ctx, lt).Error(0)
}
func (m *MockLifecycleTransitionRepo) ListByCatalogVersion(ctx context.Context, catalogVersionID string) ([]*models.LifecycleTransition, error) {
	args := m.Called(ctx, catalogVersionID)
	return args.Get(0).([]*models.LifecycleTransition), args.Error(1)
}
