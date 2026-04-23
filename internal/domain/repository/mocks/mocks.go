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
func (m *MockEntityTypeVersionRepo) GetByIDs(ctx context.Context, ids []string) ([]*models.EntityTypeVersion, error) {
	args := m.Called(ctx, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.EntityTypeVersion), args.Error(1)
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
func (m *MockAttributeRepo) ListByTypeDefinitionVersionIDs(ctx context.Context, tdvIDs []string) ([]*models.Attribute, error) {
	args := m.Called(ctx, tdvIDs)
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

// MockTypeDefinitionRepo mocks TypeDefinitionRepository.
type MockTypeDefinitionRepo struct{ mock.Mock }

func (m *MockTypeDefinitionRepo) Create(ctx context.Context, td *models.TypeDefinition) error {
	return m.Called(ctx, td).Error(0)
}
func (m *MockTypeDefinitionRepo) GetByID(ctx context.Context, id string) (*models.TypeDefinition, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TypeDefinition), args.Error(1)
}
func (m *MockTypeDefinitionRepo) GetByName(ctx context.Context, name string) (*models.TypeDefinition, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TypeDefinition), args.Error(1)
}
func (m *MockTypeDefinitionRepo) List(ctx context.Context, params models.ListParams) ([]*models.TypeDefinition, int, error) {
	args := m.Called(ctx, params)
	return args.Get(0).([]*models.TypeDefinition), args.Int(1), args.Error(2)
}
func (m *MockTypeDefinitionRepo) Update(ctx context.Context, td *models.TypeDefinition) error {
	return m.Called(ctx, td).Error(0)
}
func (m *MockTypeDefinitionRepo) Delete(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

// MockTypeDefinitionVersionRepo mocks TypeDefinitionVersionRepository.
type MockTypeDefinitionVersionRepo struct{ mock.Mock }

func (m *MockTypeDefinitionVersionRepo) Create(ctx context.Context, tdv *models.TypeDefinitionVersion) error {
	return m.Called(ctx, tdv).Error(0)
}
func (m *MockTypeDefinitionVersionRepo) GetByID(ctx context.Context, id string) (*models.TypeDefinitionVersion, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TypeDefinitionVersion), args.Error(1)
}
func (m *MockTypeDefinitionVersionRepo) GetByIDs(ctx context.Context, ids []string) ([]*models.TypeDefinitionVersion, error) {
	args := m.Called(ctx, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.TypeDefinitionVersion), args.Error(1)
}
func (m *MockTypeDefinitionVersionRepo) GetLatestByTypeDefinition(ctx context.Context, typeDefID string) (*models.TypeDefinitionVersion, error) {
	args := m.Called(ctx, typeDefID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TypeDefinitionVersion), args.Error(1)
}
func (m *MockTypeDefinitionVersionRepo) GetLatestByTypeDefinitions(ctx context.Context, typeDefIDs []string) (map[string]*models.TypeDefinitionVersion, error) {
	args := m.Called(ctx, typeDefIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]*models.TypeDefinitionVersion), args.Error(1)
}
func (m *MockTypeDefinitionVersionRepo) ListByTypeDefinition(ctx context.Context, typeDefID string) ([]*models.TypeDefinitionVersion, error) {
	args := m.Called(ctx, typeDefID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.TypeDefinitionVersion), args.Error(1)
}
func (m *MockTypeDefinitionVersionRepo) GetByVersion(ctx context.Context, typeDefID string, versionNumber int) (*models.TypeDefinitionVersion, error) {
	args := m.Called(ctx, typeDefID, versionNumber)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.TypeDefinitionVersion), args.Error(1)
}

// MockCatalogVersionTypePinRepo mocks CatalogVersionTypePinRepository.
type MockCatalogVersionTypePinRepo struct{ mock.Mock }

func (m *MockCatalogVersionTypePinRepo) Create(ctx context.Context, pin *models.CatalogVersionTypePin) error {
	return m.Called(ctx, pin).Error(0)
}
func (m *MockCatalogVersionTypePinRepo) GetByID(ctx context.Context, id string) (*models.CatalogVersionTypePin, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.CatalogVersionTypePin), args.Error(1)
}
func (m *MockCatalogVersionTypePinRepo) ListByCatalogVersion(ctx context.Context, catalogVersionID string) ([]*models.CatalogVersionTypePin, error) {
	args := m.Called(ctx, catalogVersionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.CatalogVersionTypePin), args.Error(1)
}
func (m *MockCatalogVersionTypePinRepo) Delete(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
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
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
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
