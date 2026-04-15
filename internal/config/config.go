package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Dialect       string
	DSN           string
	MigrationsDir string
	LogSQL        bool
}

func Load() (*Config, error) {
	cfg := &Config{
		Dialect:       strings.ToLower(getenv("DB_DIALECT", "")),
		DSN:           getenv("DB_DSN", ""),
		MigrationsDir: getenv("MIGRATIONS_DIR", "/app/migrations"),
		LogSQL:        strings.ToLower(getenv("LOG_SQL", "false")) == "true",
	}

	if cfg.Dialect == "" {
		return nil, fmt.Errorf("DB_DIALECT is required")
	}
	if cfg.DSN == "" {
		return nil, fmt.Errorf("DB_DSN is required")
	}

	switch cfg.Dialect {
	case "mysql", "mariadb", "postgres", "postgresql", "dm":
	default:
		return nil, fmt.Errorf("unsupported DB_DIALECT: %s", cfg.Dialect)
	}

	return cfg, nil
}

func getenv(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}
