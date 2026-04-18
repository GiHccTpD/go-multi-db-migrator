package dialect

import (
	"context"
	"database/sql"
	"fmt"
	"hash/fnv"
	"time"

	"github.com/GiHccTpD/go-multi-db-migrator/internal/migcore"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgresDriver struct{}

var _ Driver = (*PostgresDriver)(nil)

func (d PostgresDriver) Name() string                 { return "postgres" }
func (d PostgresDriver) NormalizeDialectName() string { return "postgres" }

func (d PostgresDriver) Open(dsn string) (*sql.DB, error) {
	return sql.Open("pgx", dsn)
}

func (d PostgresDriver) EnsureVersionTable(ctx context.Context, db *sql.DB) error {
	sqlText := `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version      VARCHAR(64) PRIMARY KEY,
    name         VARCHAR(255) NOT NULL,
    checksum     VARCHAR(128) NOT NULL,
    applied_at   TIMESTAMP NOT NULL,
    success      INTEGER NOT NULL,
    execution_ms BIGINT NOT NULL
);`
	_, err := db.ExecContext(ctx, sqlText)
	return err
}

func (d PostgresDriver) GetAppliedMigrations(ctx context.Context, db *sql.DB) (map[string]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT version, checksum FROM schema_migrations WHERE success = 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]string)
	for rows.Next() {
		var version, checksum string
		if err := rows.Scan(&version, &checksum); err != nil {
			return nil, err
		}
		out[version] = checksum
	}
	return out, rows.Err()
}

func (d PostgresDriver) ApplyMigration(ctx context.Context, db *sql.DB, m migcore.Migration, logSQL bool) error {
	start := time.Now()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, m.SQL); err != nil {
		return fmt.Errorf("exec migration %s failed: %w", m.FileName, err)
	}

	if _, err := tx.ExecContext(ctx, `
INSERT INTO schema_migrations(version, name, checksum, applied_at, success, execution_ms)
VALUES ($1, $2, $3, $4, $5, $6)
`, m.Version, m.Name, m.Checksum, time.Now(), 1, time.Since(start).Milliseconds()); err != nil {
		return fmt.Errorf("insert migration record failed: %w", err)
	}

	return tx.Commit()
}

func (d PostgresDriver) RollbackMigration(ctx context.Context, db *sql.DB, m migcore.Migration, logSQL bool) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, m.SQL); err != nil {
		return fmt.Errorf("exec rollback %s failed: %w", m.FileName, err)
	}

	res, err := tx.ExecContext(ctx, `DELETE FROM schema_migrations WHERE version = $1`, m.Version)
	if err != nil {
		return fmt.Errorf("delete migration record failed: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete migration record rows affected failed: %w", err)
	}
	if affected != 1 {
		return fmt.Errorf("delete migration record affected %d rows, want 1", affected)
	}

	return tx.Commit()
}

func (d PostgresDriver) AcquireLock(ctx context.Context, db *sql.DB, lockKey string) (func() error, error) {
	lockID := hashToInt64(lockKey)

	var ok bool
	if err := db.QueryRowContext(ctx, `SELECT pg_try_advisory_lock($1)`, lockID).Scan(&ok); err != nil {
		return nil, fmt.Errorf("acquire postgres advisory lock failed: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("postgres advisory lock is already held: %s", lockKey)
	}

	unlock := func() error {
		var released bool
		err := db.QueryRowContext(context.Background(), `SELECT pg_advisory_unlock($1)`, lockID).Scan(&released)
		if err != nil {
			return fmt.Errorf("release postgres advisory lock failed: %w", err)
		}
		if !released {
			return fmt.Errorf("postgres advisory lock was not released: %s", lockKey)
		}
		return nil
	}

	return unlock, nil
}

func hashToInt64(s string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	return int64(h.Sum64())
}
