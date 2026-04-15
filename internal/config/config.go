package config

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"strings"
)

type Config struct {
	Dialect        string
	DSN            string
	MigrationsDir  string
	DBInstanceName string
	LogSQL         bool
}

func Load() (*Config, error) {
	cfg := &Config{
		Dialect:        strings.ToLower(getenv("DB_DIALECT", "")),
		DSN:            getenv("DB_DSN", ""),
		MigrationsDir:  getenv("MIGRATIONS_DIR", "/app/migrations"),
		DBInstanceName: getenv("DB_INSTANCE_NAME", ""),
		LogSQL:         strings.ToLower(getenv("LOG_SQL", "false")) == "true",
	}

	if cfg.Dialect == "" {
		return nil, fmt.Errorf("DB_DIALECT is required")
	}
	if cfg.DSN == "" {
		return nil, fmt.Errorf("DB_DSN is required")
	}

	if cfg.DBInstanceName == "" {
		name, err := ParseDBInstanceName(cfg.Dialect, cfg.DSN)
		if err != nil {
			return nil, fmt.Errorf("parse DB_INSTANCE_NAME from DSN failed: %w", err)
		}
		cfg.DBInstanceName = name
	}

	return cfg, nil
}

func ParseDBInstanceName(dialect, dsn string) (string, error) {
	switch dialect {
	case "postgres", "postgresql", "dm":
		u, err := url.Parse(dsn)
		if err != nil {
			return "", err
		}
		dbName := strings.TrimPrefix(path.Clean(u.Path), "/")
		if dbName == "" || dbName == "." {
			return "", fmt.Errorf("database name not found in dsn path")
		}
		return dbName, nil

	case "mysql", "mariadb":
		// 兼容：
		// user:pass@tcp(127.0.0.1:3306)/test?parseTime=true
		// 截取最后一个 / 后到 ? 前
		slash := strings.LastIndex(dsn, "/")
		if slash < 0 || slash == len(dsn)-1 {
			return "", fmt.Errorf("database name not found in mysql dsn")
		}
		part := dsn[slash+1:]
		if q := strings.Index(part, "?"); q >= 0 {
			part = part[:q]
		}
		part = strings.TrimSpace(part)
		if part == "" {
			return "", fmt.Errorf("database name empty in mysql dsn")
		}
		return part, nil
	default:
		return "", fmt.Errorf("unsupported dialect: %s", dialect)
	}
}

func getenv(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}