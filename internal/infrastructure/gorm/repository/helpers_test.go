package repository

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsUniqueConstraintError_Nil(t *testing.T) {
	assert.False(t, isUniqueConstraintError(nil))
}

func TestIsUniqueConstraintError_NonUnique(t *testing.T) {
	assert.False(t, isUniqueConstraintError(errors.New("some other error")))
}

func TestIsUniqueConstraintError_UniqueViolation(t *testing.T) {
	assert.True(t, isUniqueConstraintError(errors.New("UNIQUE constraint failed")))
}
