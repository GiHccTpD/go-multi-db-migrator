package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/GiHccTpD/go-multi-db-migrator/internal/config"
	"github.com/GiHccTpD/go-multi-db-migrator/internal/dialect"
	"github.com/GiHccTpD/go-multi-db-migrator/internal/migcore"
	"github.com/GiHccTpD/go-multi-db-migrator/internal/migrator"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config failed: %v", err)
	}

	driver := pickDriver(cfg.Dialect)

	db, err := driver.Open(cfg.DSN)
	if err != nil {
		log.Fatalf("open db failed: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	rootDir := filepath.Join(cfg.MigrationsDir, cfg.DBInstanceName)

	r := &migrator.Runner{
		DB:             db,
		Driver:         driver,
		LogSQL:         cfg.LogSQL,
		RootDir:        rootDir,
		Dialect:        normalizeDialect(cfg.Dialect),
		DBInstanceName: cfg.DBInstanceName,
		Direction:      migrator.Direction(cfg.Direction),
		TargetVersion:  cfg.TargetVersion,
	}

	if err := r.Run(ctx); err != nil {
		log.Printf("migration failed: %v", err)
		os.Exit(1)
	}

	log.Println("migration completed successfully")
}

func pickDriver(name string) migcore.Driver {
	switch normalizeDialect(name) {
	case "mysql":
		return dialect.MySQLDriver{}
	case "postgres":
		return dialect.PostgresDriver{}
	case "dm":
		return dialect.DMDriver{}
	default:
		log.Fatalf("unsupported dialect: %s", name)
		return nil
	}
}

func normalizeDialect(name string) string {
	switch name {
	case "mysql", "mariadb":
		return "mysql"
	case "postgres", "postgresql":
		return "postgres"
	case "dm":
		return "dm"
	default:
		return name
	}
}
