# Repository Guidelines

## Project Structure & Module Organization

This is a Go CLI project for applying database migrations across MySQL, PostgreSQL, and DM. Entry points live in `cmd/migrator` for running migrations and `cmd/mk_migration` for generating SQL files. Reusable implementation code is under `internal/`: `config` loads environment settings, `dialect` contains database-specific drivers, `migcore` defines shared migration models, and `migrator` loads and applies migration files. Example configuration files are `db-migrator-*.yaml`, container packaging is in `Dockerfile`, and SQL migrations are stored under `migrations/`. Runtime loading expects `MIGRATIONS_DIR/DB_INSTANCE_NAME/<dialect>/`.

## Build, Test, and Development Commands

- `go test ./...` runs all Go package tests.
- `go build ./...` verifies every package compiles.
- `make build` builds the migrator binary at `bin/go-multi-db-migrator`.
- `make docker` builds the container image using `IMAGE_NAME` from the Makefile.
- `go run ./cmd/migrator` runs migrations using `DB_DIALECT`, `DB_DSN`, `DB_INSTANCE_NAME`, and `MIGRATIONS_DIR`.
- `go run ./cmd/mk_migration --name add_user_table --all` creates matching `.up.sql` and `.down.sql` files for all supported dialects.

## Coding Style & Naming Conventions

Use standard Go formatting: run `gofmt -w` on changed `.go` files before committing. Package names should stay short and lowercase, matching the existing `internal/config`, `internal/dialect`, and `internal/migrator` style. Exported identifiers should be documented when their purpose is not obvious. Keep dialect-specific behavior behind `migcore.Driver` implementations instead of branching through application code.

Migration files must use `{6-digit-version}_{snake_case}.{up|down}.sql`, for example `000002_add_user_email_index.up.sql`. Do not edit SQL files after they have been applied, because checksums are recorded.

## Testing Guidelines

There are currently no checked-in `*_test.go` files, so add focused unit tests with new behavior. Prefer table-driven tests for parsing, loading, checksum, and dialect-independent runner logic. Name tests after the behavior under test, such as `TestLoadMigrationsSortsByVersion`. Run `go test ./...` before opening a PR.

## Commit & Pull Request Guidelines

Recent history uses Conventional Commit-style messages, including `feat: ...`, `feat(scope): ...`, and `docs(scope): ...`. Follow that pattern and keep subjects imperative and concise.

Pull requests should include a short summary, test results, and any migration/runtime notes. Link related issues when available. For database changes, call out affected dialects and confirm whether both `up` and `down` migrations were added.

## Security & Configuration Tips

Do not commit real DSNs, passwords, or production registry names. Use the sample YAML files for structure only. Keep `LOG_SQL=false` unless SQL logging is needed for local debugging, since migration statements may expose schema or operational details.
