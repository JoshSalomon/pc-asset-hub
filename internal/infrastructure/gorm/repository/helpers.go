package repository

import (
	"fmt"
	"strings"
)

func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "unique_violation") ||
		strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "unique") && strings.Contains(msg, "constraint")
}

// allowedSortColumns defines the columns that can be used for sorting.
// This prevents SQL injection via the ORDER BY clause.
var allowedSortColumns = map[string]bool{
	"name":       true,
	"created_at": true,
	"updated_at": true,
	"version":    true,
}

// validateSortBy checks if the sort column is in the allowlist.
func validateSortBy(sortBy string) error {
	if sortBy == "" {
		return nil
	}
	if !allowedSortColumns[sortBy] {
		return fmt.Errorf("invalid sort column: %s", sortBy)
	}
	return nil
}
