package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRunCreatesMigrationsUnderDBInstanceDialectDirs(t *testing.T) {
	t.Chdir(t.TempDir())

	var out bytes.Buffer
	err := run([]string{
		"--db-instance", "migrate_test",
		"--name", "add_user_table",
		"--all",
	}, &out)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	for _, dialect := range []string{"mysql", "postgres", "dm"} {
		for _, direction := range []string{"up", "down"} {
			path := filepath.Join("migrations", "migrate_test", dialect, "000001_add_user_table."+direction+".sql")
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("expected migration file %s to exist: %v", path, err)
			}
		}
	}
}

func TestRunRequiresDBInstance(t *testing.T) {
	t.Chdir(t.TempDir())

	err := run([]string{"--name", "add_user_table", "--all"}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("run() error = nil, want db-instance requirement error")
	}
}
