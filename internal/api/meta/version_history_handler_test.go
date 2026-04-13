package meta_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	apimeta "github.com/project-catalyst/pc-asset-hub/internal/api/meta"
	apimw "github.com/project-catalyst/pc-asset-hub/internal/api/middleware"
	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/models"
	"github.com/project-catalyst/pc-asset-hub/internal/domain/repository/mocks"
	svcmeta "github.com/project-catalyst/pc-asset-hub/internal/service/meta"
)

func setupVHServer(etvRepo *mocks.MockEntityTypeVersionRepo, attrRepo *mocks.MockAttributeRepo, assocRepo *mocks.MockAssociationRepo) *echo.Echo {
	e := echo.New()
	svc := svcmeta.NewVersionHistoryService(etvRepo, attrRepo, assocRepo)
	handler := apimeta.NewVersionHistoryHandler(svc)

	g := e.Group("/api/meta/v1")
	rbac := &apimw.HeaderRBACProvider{}
	g.Use(apimw.RBACMiddleware(rbac))
	apimeta.RegisterVersionHistoryRoutes(g, handler)

	return e
}

// T-C.26: Get version history
func TestTC26_GetVersionHistory(t *testing.T) {
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	e := setupVHServer(etvRepo, nil, nil)

	now := time.Now()
	etvRepo.On("ListByEntityType", mock.Anything, "et1").Return([]*models.EntityTypeVersion{
		{ID: "v1", EntityTypeID: "et1", Version: 1, Description: "Initial", CreatedAt: now},
		{ID: "v2", EntityTypeID: "et1", Version: 2, Description: "Added attr", CreatedAt: now},
	}, nil)

	rec := doRequest(e, http.MethodGet, "/api/meta/v1/entity-types/et1/versions", "", apimw.RoleRO)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"version":1`)
	assert.Contains(t, rec.Body.String(), `"version":2`)
}

// T-C.27: Compare two versions
func TestTC27_CompareVersions(t *testing.T) {
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	attrRepo := new(mocks.MockAttributeRepo)
	assocRepo := new(mocks.MockAssociationRepo)
	e := setupVHServer(etvRepo, attrRepo, assocRepo)

	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 1).Return(&models.EntityTypeVersion{ID: "v1", EntityTypeID: "et1", Version: 1}, nil)
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 2).Return(&models.EntityTypeVersion{ID: "v2", EntityTypeID: "et1", Version: 2}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Attribute{}, nil)
	attrRepo.On("ListByVersion", mock.Anything, "v2").Return([]*models.Attribute{
		{ID: "a1", Name: "hostname", TypeDefinitionVersionID: "tdv-string"},
	}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v1").Return([]*models.Association{}, nil)
	assocRepo.On("ListByVersion", mock.Anything, "v2").Return([]*models.Association{}, nil)

	rec := doRequest(e, http.MethodGet, "/api/meta/v1/entity-types/et1/versions/diff?v1=1&v2=2", "", apimw.RoleRO)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"added"`)
	assert.Contains(t, rec.Body.String(), `"hostname"`)
}

// T-C.28: Compare nonexistent version → 404
func TestTC28_CompareNonexistentVersion(t *testing.T) {
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	e := setupVHServer(etvRepo, nil, nil)

	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 1).Return(nil, domainerrors.NewNotFound("EntityTypeVersion", "et1:1"))

	rec := doRequest(e, http.MethodGet, "/api/meta/v1/entity-types/et1/versions/diff?v1=1&v2=2", "", apimw.RoleRO)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// T-C.28b: Compare with missing query params → 400
func TestTC28b_CompareMissingParams(t *testing.T) {
	e := setupVHServer(nil, nil, nil)

	rec := doRequest(e, http.MethodGet, "/api/meta/v1/entity-types/et1/versions/diff", "", apimw.RoleRO)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// T-C.28c: Compare with non-integer v1 → 400
func TestTC28c_CompareNonIntegerVersion(t *testing.T) {
	e := setupVHServer(nil, nil, nil)

	rec := doRequest(e, http.MethodGet, "/api/meta/v1/entity-types/et1/versions/diff?v1=abc&v2=2", "", apimw.RoleRO)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// === Coverage: service-error branches ===

func TestVHList_ServiceError(t *testing.T) {
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	e := setupVHServer(etvRepo, nil, nil)
	etvRepo.On("ListByEntityType", mock.Anything, "et1").Return(([]*models.EntityTypeVersion)(nil), domainerrors.NewNotFound("EntityType", "et1"))
	rec := doRequest(e, http.MethodGet, "/api/meta/v1/entity-types/et1/versions", "", apimw.RoleRO)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestVHDiff_ServiceError(t *testing.T) {
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	e := setupVHServer(etvRepo, nil, nil)
	etvRepo.On("GetByEntityTypeAndVersion", mock.Anything, "et1", 1).Return(nil, domainerrors.NewNotFound("EntityTypeVersion", "et1:1"))
	rec := doRequest(e, http.MethodGet, "/api/meta/v1/entity-types/et1/versions/diff?v1=1&v2=2", "", apimw.RoleRO)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// Coverage: Diff v2 parse error (line 53-55)
func TestVHDiff_V2ParseError(t *testing.T) {
	etvRepo := new(mocks.MockEntityTypeVersionRepo)
	e := setupVHServer(etvRepo, nil, nil)

	rec := doRequest(e, http.MethodGet, "/api/meta/v1/entity-types/et1/versions/diff?v1=1&v2=abc", "", apimw.RoleRO)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
