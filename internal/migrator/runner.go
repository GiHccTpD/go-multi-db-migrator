package migrator

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sort"
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
	Direction      Direction
	TargetVersion  string
}

type Direction string

const (
	DirectionUp   Direction = "up"
	DirectionDown Direction = "down"
)

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

	switch r.direction() {
	case DirectionUp:
		return r.runUp(ctx, applied)
	case DirectionDown:
		return r.runDown(ctx, applied)
	default:
		return fmt.Errorf("unsupported migration direction: %s", r.Direction)
	}
}

func (r *Runner) direction() Direction {
	if r.Direction == "" {
		return DirectionUp
	}
	return r.Direction
}

func (r *Runner) runUp(ctx context.Context, applied map[string]string) error {
	migrations, err := LoadMigrations(r.RootDir, r.Dialect)
	if err != nil {
		return fmt.Errorf("load migration files failed: %w", err)
	}
	if err := validateTargetVersion(migrations, r.TargetVersion, true); err != nil {
		return err
	}

	if len(migrations) == 0 {
		log.Printf("no migrations found for db_instance=%s dialect=%s", r.DBInstanceName, r.Dialect)
		return nil
	}

	for _, m := range migrations {
		if r.TargetVersion != "" && m.Version > r.TargetVersion {
			break
		}
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

func (r *Runner) runDown(ctx context.Context, applied map[string]string) error {
	if r.TargetVersion == "" {
		return fmt.Errorf("MIGRATION_TARGET_VERSION is required when MIGRATION_DIRECTION=down")
	}

	upMigrations, err := LoadMigrations(r.RootDir, r.Dialect)
	if err != nil {
		return fmt.Errorf("load up migration files failed: %w", err)
	}
	if err := validateTargetVersion(upMigrations, r.TargetVersion, false); err != nil {
		return err
	}
	downMigrations, err := LoadDownMigrations(r.RootDir, r.Dialect)
	if err != nil {
		return fmt.Errorf("load down migration files failed: %w", err)
	}

	upByVersion := migrationsByVersion(upMigrations)
	downByVersion := migrationsByVersion(downMigrations)
	versions := appliedVersionsAfter(applied, r.TargetVersion)
	if len(versions) == 0 {
		log.Printf("no migrations need rollback for db_instance=%s dialect=%s target_version=%s", r.DBInstanceName, r.Dialect, r.TargetVersion)
		return nil
	}

	for _, version := range versions {
		up, ok := upByVersion[version]
		if !ok {
			return fmt.Errorf("applied migration version=%s has no local up file", version)
		}
		if applied[version] != up.Checksum {
			return fmt.Errorf("checksum mismatch for rollback version=%s file=%s", version, up.FileName)
		}

		down, ok := downByVersion[version]
		if !ok {
			return fmt.Errorf("applied migration version=%s has no local down file", version)
		}

		log.Printf("rolling back migration version=%s file=%s", down.Version, down.FileName)
		start := time.Now()
		if err := r.Driver.RollbackMigration(ctx, r.DB, down, r.LogSQL); err != nil {
			return fmt.Errorf("rollback migration version=%s file=%s failed: %w", down.Version, down.FileName, err)
		}
		log.Printf("rolled back migration version=%s cost_ms=%d", down.Version, time.Since(start).Milliseconds())
	}

	return nil
}

func validateTargetVersion(migrations []migcore.Migration, targetVersion string, allowEmpty bool) error {
	if targetVersion == "" && allowEmpty {
		return nil
	}
	if targetVersion == "000000" {
		return nil
	}
	for _, m := range migrations {
		if m.Version == targetVersion {
			return nil
		}
	}
	return fmt.Errorf("target migration version %s not found", targetVersion)
}

func migrationsByVersion(migrations []migcore.Migration) map[string]migcore.Migration {
	out := make(map[string]migcore.Migration, len(migrations))
	for _, m := range migrations {
		out[m.Version] = m
	}
	return out
}

func appliedVersionsAfter(applied map[string]string, targetVersion string) []string {
	var versions []string
	for version := range applied {
		if version > targetVersion {
			versions = append(versions, version)
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(versions)))
	return versions
}
