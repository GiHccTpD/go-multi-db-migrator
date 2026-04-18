package dialect

import (
	"context"
	"database/sql"

	"github.com/GiHccTpD/go-multi-db-migrator/internal/migcore"
)

type Driver interface {
	Open(dsn string) (*sql.DB, error)
	Name() string
	NormalizeDialectName() string

	EnsureVersionTable(ctx context.Context, db *sql.DB) error
	GetAppliedMigrations(ctx context.Context, db *sql.DB) (map[string]string, error)
	ApplyMigration(ctx context.Context, db *sql.DB, m migcore.Migration, logSQL bool) error
	RollbackMigration(ctx context.Context, db *sql.DB, m migcore.Migration, logSQL bool) error
}
