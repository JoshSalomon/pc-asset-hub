package operational

import (
	"errors"
	"net/http"
	"testing"

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

func TestMapError_Internal(t *testing.T) {
	err := mapError(errors.New("unknown"))
	assert.Equal(t, http.StatusInternalServerError, err.Code)
}
