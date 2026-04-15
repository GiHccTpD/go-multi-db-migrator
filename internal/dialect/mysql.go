package dialect

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/GiHccTpD/go-multi-db-migrator/internal/migcore"

	_ "github.com/go-sql-driver/mysql"
)

type MySQLDriver struct{}

var _ Driver = (*MySQLDriver)(nil)

func (d MySQLDriver) Name() string                 { return "mysql" }
func (d MySQLDriver) NormalizeDialectName() string { return "mysql" }

func (d MySQLDriver) Open(dsn string) (*sql.DB, error) {
	return sql.Open("mysql", dsn)
}

func (d MySQLDriver) EnsureVersionTable(ctx context.Context, db *sql.DB) error {
	sqlText := `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version      VARCHAR(64) PRIMARY KEY,
    name         VARCHAR(255) NOT NULL,
    checksum     VARCHAR(128) NOT NULL,
    applied_at   TIMESTAMP NOT NULL,
    success      INT NOT NULL,
    execution_ms BIGINT NOT NULL
);`
	_, err := db.ExecContext(ctx, sqlText)
	return err
}

func (d MySQLDriver) GetAppliedMigrations(ctx context.Context, db *sql.DB) (map[string]string, error) {
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

func (d MySQLDriver) ApplyMigration(ctx context.Context, db *sql.DB, m migcore.Migration, logSQL bool) error {
	start := time.Now()

	if _, err := db.ExecContext(ctx, m.SQL); err != nil {
		return fmt.Errorf("exec migration %s failed: %w", m.FileName, err)
	}

	if _, err := db.ExecContext(ctx, `
INSERT INTO schema_migrations(version, name, checksum, applied_at, success, execution_ms)
VALUES (?, ?, ?, ?, ?, ?)
`, m.Version, m.Name, m.Checksum, time.Now(), 1, time.Since(start).Milliseconds()); err != nil {
		return fmt.Errorf("insert migration record failed: %w", err)
	}

	return nil
}

func (d MySQLDriver) AcquireLock(ctx context.Context, db *sql.DB, lockKey string) (func() error, error) {
	var got sql.NullInt64
	if err := db.QueryRowContext(ctx, `SELECT GET_LOCK(?, 0)`, lockKey).Scan(&got); err != nil {
		return nil, fmt.Errorf("acquire mysql lock failed: %w", err)
	}
	if !got.Valid || got.Int64 != 1 {
		return nil, fmt.Errorf("mysql lock is already held: %s", lockKey)
	}

	unlock := func() error {
		var released sql.NullInt64
		if err := db.QueryRowContext(context.Background(), `SELECT RELEASE_LOCK(?)`, lockKey).Scan(&released); err != nil {
			return fmt.Errorf("release mysql lock failed: %w", err)
		}
		return nil
	}
	return unlock, nil
}
