package migrator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GiHccTpD/go-multi-db-migrator/internal/migcore"
)

func LoadMigrations(rootDir, dialect string) ([]migcore.Migration, error) {
	return loadMigrationFiles(rootDir, dialect, "up")
}

func LoadDownMigrations(rootDir, dialect string) ([]migcore.Migration, error) {
	return loadMigrationFiles(rootDir, dialect, "down")
}

func loadMigrationFiles(rootDir, dialect, direction string) ([]migcore.Migration, error) {
	dir := filepath.Join(rootDir, dialect)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir failed: %w", err)
	}

	var out []migcore.Migration
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, "."+direction+".sql") {
			continue
		}

		version, migName, err := parseFileName(name, direction)
		if err != nil {
			return nil, fmt.Errorf("parse file %s failed: %w", name, err)
		}

		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("read file %s failed: %w", name, err)
		}

		sqlText := string(b)
		out = append(out, migcore.Migration{
			Version:  version,
			Name:     migName,
			FileName: name,
			SQL:      sqlText,
			Checksum: migcore.Checksum(sqlText),
		})
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Version < out[j].Version
	})

	return out, nil
}

func parseFileName(file, direction string) (version, name string, err error) {
	suffix := "." + direction + ".sql"
	if !strings.HasSuffix(file, suffix) {
		return "", "", fmt.Errorf("not a %s migration file: %s", direction, file)
	}

	base := strings.TrimSuffix(file, suffix)
	idx := strings.Index(base, "_")
	if idx <= 0 || idx == len(base)-1 {
		return "", "", fmt.Errorf("invalid migration filename: %s", file)
	}

	version = base[:idx]
	name = base[idx+1:]
	return version, name, nil
}
