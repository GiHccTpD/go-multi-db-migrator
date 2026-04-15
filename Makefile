APP=github.com/GiHccTpD/go-multi-db-migrator

build:
	go build -o bin/$(APP) ./cmd/migrator

docker:
	docker build -t your-registry/$(APP):latest .

run-postgres:
	DB_DIALECT=postgres \
	DB_DSN='postgres://postgres:postgres@127.0.0.1:5432/test?sslmode=disable' \
	MIGRATIONS_DIR=./migrations \
	go run ./cmd/migrator

run-mysql:
	DB_DIALECT=mysql \
	DB_DSN='root:123456@tcp(127.0.0.1:3306)/test?parseTime=true&multiStatements=true' \
	MIGRATIONS_DIR=./migrations \
	go run ./cmd/migrator

run-dm:
	DB_DIALECT=dm \
	DB_DSN='dm://SYSDBA:SYSDBA@127.0.0.1:5236?schema=SYSDBA' \
	MIGRATIONS_DIR=./migrations \
	go run ./cmd/migrator