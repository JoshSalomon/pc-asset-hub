//go:build postgres

package database

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// NewDB opens a PostgreSQL database connection.
func NewDB(dsn string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}
