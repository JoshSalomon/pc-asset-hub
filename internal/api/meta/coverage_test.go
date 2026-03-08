package meta

import (
	"errors"
	"net/http"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	domainerrors "github.com/project-catalyst/pc-asset-hub/internal/domain/errors"
)

func TestMapError_NotFound(t *testing.T) {
	err := mapError(domainerrors.NewNotFound("X", "1"))
	assert.Equal(t, http.StatusNotFound, err.Code)
}

func TestMapError_Conflict(t *testing.T) {
	err := mapError(domainerrors.NewConflict("X", "dup"))
	assert.Equal(t, http.StatusConflict, err.Code)
}

func TestMapError_Validation(t *testing.T) {
	err := mapError(domainerrors.NewValidation("bad"))
	assert.Equal(t, http.StatusBadRequest, err.Code)
}

func TestMapError_Forbidden(t *testing.T) {
	err := mapError(domainerrors.NewForbidden("nope"))
	assert.Equal(t, http.StatusForbidden, err.Code)
}

func TestMapError_CycleDetected(t *testing.T) {
	err := mapError(domainerrors.NewCycleDetected("cycle"))
	assert.Equal(t, http.StatusUnprocessableEntity, err.Code)
}

func TestMapError_ReferencedEnum(t *testing.T) {
	err := mapError(domainerrors.NewReferencedEnum("e1", []string{"attr1"}))
	assert.Equal(t, http.StatusUnprocessableEntity, err.Code)
}

func TestMapError_DeepCopyRequired(t *testing.T) {
	err := mapError(domainerrors.NewDeepCopyRequired("version is pinned in a non-development catalog"))
	assert.Equal(t, http.StatusConflict, err.Code)
}

func TestMapError_Internal(t *testing.T) {
	err := mapError(errors.New("unknown error"))
	assert.Equal(t, http.StatusInternalServerError, err.Code)
}

func TestSetupRoutes(t *testing.T) {
	e := echo.New()
	g := e.Group("/api/meta/v1")
	handler := &EntityTypeHandler{}
	SetupRoutes(g, handler)

	routes := e.Routes()
	var paths []string
	for _, r := range routes {
		paths = append(paths, r.Path)
	}
	assert.Contains(t, paths, "/api/meta/v1/entity-types")
	assert.Contains(t, paths, "/api/meta/v1/entity-types/:id")
}
