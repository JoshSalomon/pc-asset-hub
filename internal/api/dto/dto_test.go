package dto_test

import (
	"testing"

	"github.com/project-catalyst/pc-asset-hub/internal/api/dto"
	"github.com/stretchr/testify/assert"
)

func TestIsSystemAttributeName(t *testing.T) {
	assert.True(t, dto.IsSystemAttributeName("name"))
	assert.True(t, dto.IsSystemAttributeName("description"))
	assert.False(t, dto.IsSystemAttributeName("hostname"))
	assert.False(t, dto.IsSystemAttributeName("Name"))
	assert.False(t, dto.IsSystemAttributeName(""))
}
