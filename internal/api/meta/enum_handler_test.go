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

func setupEnumServer(enumRepo *mocks.MockEnumRepo, evRepo *mocks.MockEnumValueRepo) *echo.Echo {
	e := echo.New()
	svc := svcmeta.NewEnumService(enumRepo, evRepo, nil)
	handler := apimeta.NewEnumHandler(svc)

	g := e.Group("/api/meta/v1")
	rbac := &apimw.HeaderRBACProvider{}
	g.Use(apimw.RBACMiddleware(rbac))
	requireAdmin := apimw.RequireRole(apimw.RoleAdmin)
	apimeta.RegisterEnumRoutes(g, handler, requireAdmin)

	return e
}

// T-C.15: List enums
func TestTC15_ListEnums(t *testing.T) {
	enumRepo := new(mocks.MockEnumRepo)
	e := setupEnumServer(enumRepo, nil)

	now := time.Now()
	enumRepo.On("List", mock.Anything, mock.Anything).Return([]*models.Enum{
		{ID: "e1", Name: "Status", CreatedAt: now, UpdatedAt: now},
		{ID: "e2", Name: "Priority", CreatedAt: now, UpdatedAt: now},
	}, 2, nil)

	rec := doRequest(e, http.MethodGet, "/api/meta/v1/enums", "", apimw.RoleRO)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"Status"`)
	assert.Contains(t, rec.Body.String(), `"Priority"`)
}

// T-C.16: Create enum with values
func TestTC16_CreateEnum(t *testing.T) {
	enumRepo := new(mocks.MockEnumRepo)
	evRepo := new(mocks.MockEnumValueRepo)
	e := setupEnumServer(enumRepo, evRepo)

	enumRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	evRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/enums",
		`{"name":"Status","values":["active","inactive"]}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Contains(t, rec.Body.String(), `"Status"`)
}

// T-C.17: Create enum missing name → 400
func TestTC17_CreateEnumMissingName(t *testing.T) {
	e := setupEnumServer(nil, nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/enums",
		`{"values":["a","b"]}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// T-C.18: Create duplicate enum name → 409
func TestTC18_CreateDuplicateEnum(t *testing.T) {
	enumRepo := new(mocks.MockEnumRepo)
	e := setupEnumServer(enumRepo, nil)

	enumRepo.On("Create", mock.Anything, mock.Anything).Return(domainerrors.NewConflict("Enum", "name already exists"))

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/enums",
		`{"name":"Status"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusConflict, rec.Code)
}

// T-C.19: Get enum by ID
func TestTC19_GetEnumByID(t *testing.T) {
	enumRepo := new(mocks.MockEnumRepo)
	e := setupEnumServer(enumRepo, nil)

	now := time.Now()
	enumRepo.On("GetByID", mock.Anything, "e1").Return(&models.Enum{ID: "e1", Name: "Status", CreatedAt: now, UpdatedAt: now}, nil)

	rec := doRequest(e, http.MethodGet, "/api/meta/v1/enums/e1", "", apimw.RoleRO)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"Status"`)
}

// T-C.20: Update enum name
func TestTC20_UpdateEnum(t *testing.T) {
	enumRepo := new(mocks.MockEnumRepo)
	e := setupEnumServer(enumRepo, nil)

	now := time.Now()
	enumRepo.On("GetByID", mock.Anything, "e1").Return(&models.Enum{ID: "e1", Name: "Status", CreatedAt: now, UpdatedAt: now}, nil)
	enumRepo.On("Update", mock.Anything, mock.Anything).Return(nil)

	rec := doRequest(e, http.MethodPut, "/api/meta/v1/enums/e1",
		`{"name":"New Status"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// T-C.21: Delete enum
func TestTC21_DeleteEnum(t *testing.T) {
	enumRepo := new(mocks.MockEnumRepo)
	e := setupEnumServer(enumRepo, nil)

	enumRepo.On("Delete", mock.Anything, "e1").Return(nil)

	rec := doRequest(e, http.MethodDelete, "/api/meta/v1/enums/e1", "", apimw.RoleAdmin)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

// T-C.22: Delete referenced enum → 422
func TestTC22_DeleteReferencedEnum(t *testing.T) {
	enumRepo := new(mocks.MockEnumRepo)
	e := setupEnumServer(enumRepo, nil)

	enumRepo.On("Delete", mock.Anything, "e1").Return(domainerrors.NewReferencedEnum("Status", []string{"attr1"}))

	rec := doRequest(e, http.MethodDelete, "/api/meta/v1/enums/e1", "", apimw.RoleAdmin)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

// T-C.23: List enum values
func TestTC23_ListEnumValues(t *testing.T) {
	evRepo := new(mocks.MockEnumValueRepo)
	e := setupEnumServer(nil, evRepo)

	evRepo.On("ListByEnum", mock.Anything, "e1").Return([]*models.EnumValue{
		{ID: "v1", Value: "active", Ordinal: 0},
		{ID: "v2", Value: "inactive", Ordinal: 1},
	}, nil)

	rec := doRequest(e, http.MethodGet, "/api/meta/v1/enums/e1/values", "", apimw.RoleRO)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"active"`)
	assert.Contains(t, rec.Body.String(), `"inactive"`)
}

// T-C.24: Add enum value
func TestTC24_AddEnumValue(t *testing.T) {
	evRepo := new(mocks.MockEnumValueRepo)
	e := setupEnumServer(nil, evRepo)

	evRepo.On("ListByEnum", mock.Anything, "e1").Return([]*models.EnumValue{}, nil)
	evRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/enums/e1/values",
		`{"value":"pending"}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

// T-C.25: Reorder enum values
func TestTC25_ReorderEnumValues(t *testing.T) {
	evRepo := new(mocks.MockEnumValueRepo)
	e := setupEnumServer(nil, evRepo)

	evRepo.On("Reorder", mock.Anything, "e1", []string{"v2", "v1"}).Return(nil)

	rec := doRequest(e, http.MethodPut, "/api/meta/v1/enums/e1/values/reorder",
		`{"ordered_ids":["v2","v1"]}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusOK, rec.Code)
}

// T-C.19b: Get nonexistent enum → 404
func TestTC19b_GetEnumNotFound(t *testing.T) {
	enumRepo := new(mocks.MockEnumRepo)
	e := setupEnumServer(enumRepo, nil)

	enumRepo.On("GetByID", mock.Anything, "bad").Return(nil, domainerrors.NewNotFound("Enum", "bad"))

	rec := doRequest(e, http.MethodGet, "/api/meta/v1/enums/bad", "", apimw.RoleRO)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// T-C.16b: Create enum as RO → 403
func TestTC16b_CreateEnumAsRO(t *testing.T) {
	e := setupEnumServer(nil, nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/enums",
		`{"name":"Status"}`, apimw.RoleRO)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// T-C.24b: Add enum value missing value → 400
func TestTC24b_AddEnumValueMissingValue(t *testing.T) {
	e := setupEnumServer(nil, nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/enums/e1/values",
		`{}`, apimw.RoleAdmin)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// T-C.25b: Remove enum value
func TestTC25b_RemoveEnumValue(t *testing.T) {
	evRepo := new(mocks.MockEnumValueRepo)
	e := setupEnumServer(nil, evRepo)

	evRepo.On("Delete", mock.Anything, "v1").Return(nil)

	rec := doRequest(e, http.MethodDelete, "/api/meta/v1/enums/e1/values/v1", "", apimw.RoleAdmin)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

// T-C.67: RW cannot create enum → 403
func TestTC67_RWCannotCreateEnum(t *testing.T) {
	e := setupEnumServer(nil, nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/enums",
		`{"name":"Status"}`, apimw.RoleRW)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// T-C.69: SuperAdmin can create enum → 201
func TestTC69_SuperAdminCanCreateEnum(t *testing.T) {
	enumRepo := new(mocks.MockEnumRepo)
	e := setupEnumServer(enumRepo, nil)

	enumRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/enums",
		`{"name":"Priority"}`, apimw.RoleSuperAdmin)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

// T-C.70: RO cannot update enum → 403
func TestTC70_ROCannotUpdateEnum(t *testing.T) {
	e := setupEnumServer(nil, nil)

	rec := doRequest(e, http.MethodPut, "/api/meta/v1/enums/e1",
		`{"name":"Updated"}`, apimw.RoleRO)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// T-C.71: RO cannot delete enum → 403
func TestTC71_ROCannotDeleteEnum(t *testing.T) {
	e := setupEnumServer(nil, nil)

	rec := doRequest(e, http.MethodDelete, "/api/meta/v1/enums/e1", "", apimw.RoleRO)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// T-C.72: RO cannot add enum value → 403
func TestTC72_ROCannotAddEnumValue(t *testing.T) {
	e := setupEnumServer(nil, nil)

	rec := doRequest(e, http.MethodPost, "/api/meta/v1/enums/e1/values",
		`{"value":"pending"}`, apimw.RoleRO)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}
