package meta

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// T-CV.21: SanitizeK8sName("Release 2.3") → "release-2-3"
func TestTCV21_SanitizeK8sNameBasic(t *testing.T) {
	assert.Equal(t, "release-2-3", SanitizeK8sName("Release 2.3"))
}

// T-CV.22: SanitizeK8sName with uppercase, underscores, leading/trailing special chars → valid K8s name
func TestTCV22_SanitizeK8sNameEdgeCases(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"My_Version_1.0", "my-version-1-0"},
		{"---leading-trailing---", "leading-trailing"},
		{"ALLCAPS", "allcaps"},
		{"simple", "simple"},
		{"v1.2.3-beta", "v1-2-3-beta"},
		{"  spaces  ", "spaces"},
		{"mixed__CHARS!!123", "mixed-chars-123"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, SanitizeK8sName(tt.input))
		})
	}
}

func TestSanitizeK8sName_AllSpecialCharsReturnsEmpty(t *testing.T) {
	assert.Equal(t, "", SanitizeK8sName("!!!###$$$"))
	assert.Equal(t, "", SanitizeK8sName(""))
}

func TestSanitizeK8sName_TruncatesLongNames(t *testing.T) {
	long := ""
	for range 300 {
		long += "a"
	}
	result := SanitizeK8sName(long)
	assert.LessOrEqual(t, len(result), maxK8sNameLength)
}
