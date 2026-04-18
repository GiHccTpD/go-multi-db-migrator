package migrator

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/GiHccTpD/go-multi-db-migrator/internal/migcore"
)

type fakeDriver struct {
	applied   map[string]string
	appliedUp []string
	rolled    []string
	onEnsure  func() error
	onApplied func() error
}

func (d *fakeDriver) Open(string) (*sql.DB, error) { return nil, nil }
func (d *fakeDriver) Name() string                 { return "fake" }
func (d *fakeDriver) NormalizeDialectName() string { return "fake" }
func (d *fakeDriver) EnsureVersionTable(context.Context, *sql.DB) error {
	if d.onEnsure != nil {
		return d.onEnsure()
	}
	return nil
}
func (d *fakeDriver) GetAppliedMigrations(context.Context, *sql.DB) (map[string]string, error) {
	if d.onApplied != nil {
		if err := d.onApplied(); err != nil {
			return nil, err
		}
	}
	out := make(map[string]string, len(d.applied))
	for version, checksum := range d.applied {
		out[version] = checksum
	}
	return out, nil
}
func (d *fakeDriver) ApplyMigration(_ context.Context, _ *sql.DB, m migcore.Migration, _ bool) error {
	d.appliedUp = append(d.appliedUp, m.Version)
	return nil
}
func (d *fakeDriver) RollbackMigration(_ context.Context, _ *sql.DB, m migcore.Migration, _ bool) error {
	d.rolled = append(d.rolled, m.Version)
	return nil
}
func (d *fakeDriver) AcquireLock(context.Context, *sql.DB, string) (func() error, error) {
	return func() error { return nil }, nil
}

func TestRunUpStopsAtTargetVersion(t *testing.T) {
	rootDir := writeTestMigrations(t, map[string]string{
		"000001_create_users": "CREATE TABLE users(id INT);",
		"000002_add_email":    "ALTER TABLE users ADD email VARCHAR(255);",
	})
	driver := &fakeDriver{applied: map[string]string{}}

	r := &Runner{
		Driver:        driver,
		RootDir:       rootDir,
		Dialect:       "postgres",
		Direction:     DirectionUp,
		TargetVersion: "000001",
	}
	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got, want := driver.appliedUp, []string{"000001"}; !equalStrings(got, want) {
		t.Fatalf("applied versions = %v, want %v", got, want)
	}
}

func TestRunDownRollsBackAppliedVersionsAfterTargetInDescendingOrder(t *testing.T) {
	versions := map[string]string{
		"000001_create_users": "CREATE TABLE users(id INT);",
		"000002_add_email":    "ALTER TABLE users ADD email VARCHAR(255);",
		"000003_add_status":   "ALTER TABLE users ADD status VARCHAR(32);",
	}
	rootDir := writeTestMigrations(t, versions)

	applied := make(map[string]string)
	for name, sqlText := range versions {
		applied[name[:6]] = migcore.Checksum(sqlText + "\n")
	}
	driver := &fakeDriver{applied: applied}

	r := &Runner{
		Driver:        driver,
		RootDir:       rootDir,
		Dialect:       "postgres",
		Direction:     DirectionDown,
		TargetVersion: "000001",
	}
	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got, want := driver.rolled, []string{"000003", "000002"}; !equalStrings(got, want) {
		t.Fatalf("rolled back versions = %v, want %v", got, want)
	}
}

func TestRunCreatesCurrentDialectDirectoryAfterEnsuringVersionTable(t *testing.T) {
	rootDir := filepath.Join(t.TempDir(), "migrations", "test")
	driver := &fakeDriver{
		applied: map[string]string{},
		onEnsure: func() error {
			for _, dialect := range []string{"mysql", "postgres", "dm"} {
				if _, err := os.Stat(filepath.Join(rootDir, dialect)); err == nil {
					t.Fatalf("dialect dir %s exists before EnsureVersionTable returns", dialect)
				}
			}
			return nil
		},
		onApplied: func() error {
			if _, err := os.Stat(filepath.Join(rootDir, "postgres")); err != nil {
				t.Fatalf("expected postgres dir to exist before reading applied migrations: %v", err)
			}
			for _, dialect := range []string{"mysql", "dm"} {
				if _, err := os.Stat(filepath.Join(rootDir, dialect)); !os.IsNotExist(err) {
					t.Fatalf("dialect dir %s exists, want only current dialect dir", dialect)
				}
			}
			return nil
		},
	}

	r := &Runner{
		Driver:    driver,
		RootDir:   rootDir,
		Dialect:   "postgres",
		Direction: DirectionUp,
	}
	if err := r.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func writeTestMigrations(t *testing.T, migrations map[string]string) string {
	t.Helper()

	rootDir := filepath.Join(t.TempDir(), "migrations", "test")
	dir := filepath.Join(rootDir, "postgres")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, sqlText := range migrations {
		up := filepath.Join(dir, name+".up.sql")
		down := filepath.Join(dir, name+".down.sql")
		if err := os.WriteFile(up, []byte(sqlText+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(down, []byte("-- rollback "+name+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return rootDir
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
