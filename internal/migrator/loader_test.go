package migrator

import (
	"path/filepath"
	"testing"
)

func TestLoadMigrationsForLocalMakeTargets(t *testing.T) {
	tests := []struct {
		name     string
		rootDir  string
		dialect  string
		wantFile string
	}{
		{
			name:     "postgres migrate_test",
			rootDir:  filepath.Join("..", "..", "migrations", "migrate_test"),
			dialect:  "postgres",
			wantFile: "000001_init_schema.up.sql",
		},
		{
			name:     "mysql test",
			rootDir:  filepath.Join("..", "..", "migrations", "test"),
			dialect:  "mysql",
			wantFile: "000001_init_schema.up.sql",
		},
		{
			name:     "dm test",
			rootDir:  filepath.Join("..", "..", "migrations", "test"),
			dialect:  "dm",
			wantFile: "000001_init_schema.up.sql",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			migrations, err := LoadMigrations(tt.rootDir, tt.dialect)
			if err != nil {
				t.Fatalf("LoadMigrations() error = %v", err)
			}
			if len(migrations) != 1 {
				t.Fatalf("LoadMigrations() returned %d migrations, want 1", len(migrations))
			}
			if migrations[0].FileName != tt.wantFile {
				t.Fatalf("LoadMigrations()[0].FileName = %q, want %q", migrations[0].FileName, tt.wantFile)
			}
		})
	}
}
