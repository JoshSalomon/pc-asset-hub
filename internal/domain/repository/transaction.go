package repository

import "context"

// TransactionManager provides database transaction support for service-layer operations
// that require atomicity across multiple repository calls.
type TransactionManager interface {
	RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
