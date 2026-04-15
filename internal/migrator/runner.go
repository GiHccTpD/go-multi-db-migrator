package migrator

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/GiHccTpD/go-multi-db-migrator/internal/migcore"
)

type Runner struct {
	DB             *sql.DB
	Driver         migcore.Driver
	LogSQL         bool
	RootDir        string
	Dialect        string
	DBInstanceName string
}

func (r *Runner) Run(ctx context.Context) error {
	lockKey := fmt.Sprintf("db-migrator:%s:%s", r.Dialect, r.DBInstanceName)

	unlock, err := r.Driver.AcquireLock(ctx, r.DB, lockKey)
	if err != nil {
		return fmt.Errorf("acquire migration lock failed: %w", err)
	}
	defer func() {
		if err := unlock(); err != nil {
			log.Printf("release migration lock failed: %v", err)
		}
	}()

	if err := r.Driver.EnsureVersionTable(ctx, r.DB); err != nil {
		return fmt.Errorf("ensure version table failed: %w", err)
	}

	applied, err := r.Driver.GetAppliedMigrations(ctx, r.DB)
	if err != nil {
		return fmt.Errorf("load applied migrations failed: %w", err)
	}

	migrations, err := LoadMigrations(r.RootDir, r.Dialect)
	if err != nil {
		return fmt.Errorf("load migration files failed: %w", err)
	}

	if len(migrations) == 0 {
		log.Printf("no migrations found for db_instance=%s dialect=%s", r.DBInstanceName, r.Dialect)
		return nil
	}

	for _, m := range migrations {
		if oldChecksum, ok := applied[m.Version]; ok {
			if oldChecksum != m.Checksum {
				return fmt.Errorf("checksum mismatch for version=%s file=%s", m.Version, m.FileName)
			}
			log.Printf("skip already applied migration version=%s file=%s", m.Version, m.FileName)
			continue
		}

		log.Printf("applying migration version=%s file=%s", m.Version, m.FileName)

		start := time.Now()
		if err := r.Driver.ApplyMigration(ctx, r.DB, m, r.LogSQL); err != nil {
			return fmt.Errorf("apply migration version=%s file=%s failed: %w", m.Version, m.FileName, err)
		}
		log.Printf("applied migration version=%s cost_ms=%d", m.Version, time.Since(start).Milliseconds())
	}

	return nil
}
