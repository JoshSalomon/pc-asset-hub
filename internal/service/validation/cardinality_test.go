package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T-E.68: ValidateCardinality accepts standard options
func TestValidateCardinality_StandardOptions(t *testing.T) {
	for _, v := range []string{"0..1", "0..n", "1", "1..n"} {
		err := ValidateCardinality(v)
		assert.NoError(t, err, "should accept %q", v)
	}
}

// T-E.69: ValidateCardinality accepts custom ranges
func TestValidateCardinality_CustomRanges(t *testing.T) {
	for _, v := range []string{"2..5", "2..n", "3..10", "0..0"} {
		err := ValidateCardinality(v)
		assert.NoError(t, err, "should accept %q", v)
	}
}

// T-E.70: ValidateCardinality accepts exact values
func TestValidateCardinality_ExactValues(t *testing.T) {
	for _, v := range []string{"3", "0", "10"} {
		err := ValidateCardinality(v)
		assert.NoError(t, err, "should accept %q", v)
	}
}

// T-E.71: ValidateCardinality accepts empty string
func TestValidateCardinality_EmptyString(t *testing.T) {
	err := ValidateCardinality("")
	assert.NoError(t, err)
}

// T-E.72: ValidateCardinality rejects invalid formats
func TestValidateCardinality_InvalidFormats(t *testing.T) {
	invalid := []string{
		"-1",       // negative
		"5..2",     // min > max
		"abc",      // non-numeric
		"1.5",      // decimal
		"n..1",     // n as min
		"1..2..3",  // too many parts
		"..5",      // missing min
		"1..",      // missing max
	}
	for _, v := range invalid {
		err := ValidateCardinality(v)
		assert.Error(t, err, "should reject %q", v)
	}
}

// T-E.73: NormalizeCardinality returns "0..n" for empty string
func TestNormalizeCardinality_Empty(t *testing.T) {
	result := NormalizeCardinality("")
	require.Equal(t, "0..n", result)
}

// T-E.74: NormalizeCardinality passes through valid values unchanged
func TestNormalizeCardinality_PassThrough(t *testing.T) {
	for _, v := range []string{"0..1", "1", "1..n", "2..5"} {
		result := NormalizeCardinality(v)
		assert.Equal(t, v, result, "should pass through %q", v)
	}
}

func TestNormalizeSourceCardinality_Containment(t *testing.T) {
	// Empty containment source → "0..1" (not "0..n")
	assert.Equal(t, "0..1", NormalizeSourceCardinality("", true))
	// Explicit values pass through
	assert.Equal(t, "1", NormalizeSourceCardinality("1", true))
	assert.Equal(t, "0..1", NormalizeSourceCardinality("0..1", true))
}

func TestNormalizeSourceCardinality_NonContainment(t *testing.T) {
	// Empty non-containment source → "0..n" (standard default)
	assert.Equal(t, "0..n", NormalizeSourceCardinality("", false))
	assert.Equal(t, "1..n", NormalizeSourceCardinality("1..n", false))
}
