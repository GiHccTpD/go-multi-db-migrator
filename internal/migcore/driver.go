package migcore

import (
	"context"
	"database/sql"
)

type Driver interface {
	Open(dsn string) (*sql.DB, error)
	Name() string
	NormalizeDialectName() string

	EnsureVersionTable(ctx context.Context, db *sql.DB) error
	GetAppliedMigrations(ctx context.Context, db *sql.DB) (map[string]string, error)
	ApplyMigration(ctx context.Context, db *sql.DB, m Migration, logSQL bool) error

	AcquireLock(ctx context.Context, db *sql.DB, lockKey string) (func() error, error)
}
