package dialect

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/GiHccTpD/go-multi-db-migrator/internal/migcore"
	_ "github.com/ganl/go-dm"
)

type DMDriver struct{}

func (d DMDriver) NormalizeDialectName() string {
	return "dm"
}

var _ Driver = (*DMDriver)(nil)

func (d DMDriver) Name() string {
	return "dm"
}

func (d DMDriver) Open(dsn string) (*sql.DB, error) {
	return sql.Open("dm", dsn)
}

func (d DMDriver) EnsureVersionTable(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version        VARCHAR(64) PRIMARY KEY,
    name           VARCHAR(255) NOT NULL,
    checksum       VARCHAR(128) NOT NULL,
    applied_at     TIMESTAMP NOT NULL,
    success        INTEGER NOT NULL,
    execution_ms   BIGINT NOT NULL
)`); err != nil {
		return fmt.Errorf("create schema_migrations failed: %w", err)
	}

	if _, err := db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS schema_migration_lock (
    lock_key    VARCHAR(128) PRIMARY KEY,
    holder      VARCHAR(128) NOT NULL,
    acquired_at TIMESTAMP NOT NULL
)`); err != nil {
		return fmt.Errorf("create schema_migration_lock failed: %w", err)
	}

	return nil
}

func (d DMDriver) GetAppliedMigrations(ctx context.Context, db *sql.DB) (map[string]string, error) {
	rows, err := db.QueryContext(ctx, `
SELECT version, checksum
FROM schema_migrations
WHERE success = 1
`)
	if err != nil {
		return nil, fmt.Errorf("query schema_migrations failed: %w", err)
	}
	defer rows.Close()

	out := make(map[string]string)
	for rows.Next() {
		var version, checksum string
		if err := rows.Scan(&version, &checksum); err != nil {
			return nil, fmt.Errorf("scan schema_migrations failed: %w", err)
		}
		out[version] = checksum
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate schema_migrations failed: %w", err)
	}

	return out, nil
}

func (d DMDriver) ApplyMigration(ctx context.Context, db *sql.DB, m migcore.Migration, logSQL bool) error {
	start := time.Now()

	if _, err := db.ExecContext(ctx, m.SQL); err != nil {
		return fmt.Errorf("exec migration %s failed: %w", m.FileName, err)
	}

	_, err := db.ExecContext(ctx, `
INSERT INTO schema_migrations(version, name, checksum, applied_at, success, execution_ms)
VALUES (?, ?, ?, ?, ?, ?)
`, m.Version, m.Name, m.Checksum, time.Now(), 1, time.Since(start).Milliseconds())
	if err != nil {
		return fmt.Errorf("insert migration record failed: %w", err)
	}

	return nil
}

func (d DMDriver) AcquireLock(ctx context.Context, db *sql.DB, lockKey string) (func() error, error) {
	holder, err := currentHolder()
	if err != nil {
		return nil, fmt.Errorf("get current holder failed: %w", err)
	}

	_, err = db.ExecContext(ctx, `
INSERT INTO schema_migration_lock(lock_key, holder, acquired_at)
VALUES (?, ?, ?)
`, lockKey, holder, time.Now())
	if err != nil {
		return nil, fmt.Errorf("acquire dm migration lock failed, lock may already be held, lock_key=%s, err=%w", lockKey, err)
	}

	unlock := func() error {
		res, err := db.ExecContext(context.Background(), `
DELETE FROM schema_migration_lock
WHERE lock_key = ? AND holder = ?
`, lockKey, holder)
		if err != nil {
			return fmt.Errorf("release dm migration lock failed: %w", err)
		}

		affected, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("release dm migration lock rows affected failed: %w", err)
		}
		if affected == 0 {
			return fmt.Errorf("dm migration lock already released or holder mismatch, lock_key=%s, holder=%s", lockKey, holder)
		}
		return nil
	}

	return unlock, nil
}

func currentHolder() (string, error) {
	if podName := os.Getenv("POD_NAME"); podName != "" {
		return podName, nil
	}
	return os.Hostname()
}
