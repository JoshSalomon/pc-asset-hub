package validation

import (
	"fmt"
	"regexp"
	"strconv"
)

// DefaultCardinality is the default cardinality applied when none is specified.
const DefaultCardinality = "0..n"

var cardinalityRegex = regexp.MustCompile(`^(\d+)(\.\.(\d+|n))?$`)

// ValidateCardinality validates a cardinality string. Empty string is accepted
// (will be normalized to DefaultCardinality). Valid formats: "1", "0..1",
// "0..n", "1..n", "2..5", etc.
func ValidateCardinality(s string) error {
	if s == "" {
		return nil
	}

	matches := cardinalityRegex.FindStringSubmatch(s)
	if matches == nil {
		return fmt.Errorf("invalid cardinality format: %q", s)
	}

	// If there's a max part (e.g., "0..5" or "0..n"), validate min <= max
	if matches[2] != "" {
		min, _ := strconv.Atoi(matches[1])
		maxStr := matches[3]
		if maxStr != "n" {
			max, _ := strconv.Atoi(maxStr)
			if min > max {
				return fmt.Errorf("cardinality min (%d) must be <= max (%d)", min, max)
			}
		}
	}

	return nil
}

// NormalizeCardinality returns DefaultCardinality if s is empty,
// otherwise returns s unchanged.
func NormalizeCardinality(s string) string {
	if s == "" {
		return DefaultCardinality
	}
	return s
}

// NormalizeSourceCardinality normalizes source cardinality based on association type.
// Containment source defaults to "0..1" (an entity is contained by at most one parent).
// Non-containment defaults to "0..n".
func NormalizeSourceCardinality(s string, isContainment bool) string {
	if s == "" {
		if isContainment {
			return "0..1"
		}
		return DefaultCardinality
	}
	return s
}
