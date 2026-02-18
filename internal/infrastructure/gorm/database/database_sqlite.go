//go:build !postgres

package database

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// NewDB opens a SQLite database connection.
func NewDB(dsn string) (*gorm.DB, error) {
	return gorm.Open(sqlite.Open(dsn), &gorm.Config{})
}
