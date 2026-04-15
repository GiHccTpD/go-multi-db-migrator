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
		if !strings.HasSuffix(name, ".up.sql") {
			continue
		}

		version, migName, err := parseFileName(name)
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

func parseFileName(file string) (version, name string, err error) {
	if !strings.HasSuffix(file, ".up.sql") {
		return "", "", fmt.Errorf("not an up migration file: %s", file)
	}

	base := strings.TrimSuffix(file, ".up.sql")
	idx := strings.Index(base, "_")
	if idx <= 0 || idx == len(base)-1 {
		return "", "", fmt.Errorf("invalid migration filename: %s", file)
	}

	version = base[:idx]
	name = base[idx+1:]
	return version, name, nil
}
