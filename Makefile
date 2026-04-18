BINARY_NAME=go-multi-db-migrator
IMAGE_NAME=your-registry/go-multi-db-migrator
MIGRATIONS_DIR=./migrations

.PHONY: build docker run-postgres run-mysql run-dm mk-migration clean

build:
	mkdir -p bin
	go build -o bin/$(BINARY_NAME) ./cmd/migrator

docker:
	docker build -t $(IMAGE_NAME):latest .

run-postgres:
	DB_DIALECT=postgres \
	DB_DSN='postgres://onyx:12345678@127.0.0.1:5432/test?sslmode=disable' \
	DB_INSTANCE_NAME=test \
	MIGRATIONS_DIR=$(MIGRATIONS_DIR) \
	go run ./cmd/migrator

run-mysql:
	DB_DIALECT=mysql \
	DB_DSN='root:123456@tcp(127.0.0.1:3306)/test?parseTime=true&multiStatements=true' \
	DB_INSTANCE_NAME=test \
	MIGRATIONS_DIR=$(MIGRATIONS_DIR) \
	go run ./cmd/migrator

run-dm:
	DB_DIALECT=dm \
	DB_DSN='dm://SYSDBA:SYSDBA@127.0.0.1:5236/test?socketTimeout=30&autoCommit=true' \
	DB_INSTANCE_NAME=test \
	MIGRATIONS_DIR=$(MIGRATIONS_DIR) \
	go run ./cmd/migrator

mk-migration:
	@test -n "$(db)" || (echo "usage: make mk-migration db=test name=init_base_schema"; exit 1)
	@test -n "$(name)" || (echo "usage: make mk-migration db=test name=init_base_schema"; exit 1)
	go run ./cmd/mk_migration --db-instance $(db) --name $(name) --all

clean:
	rm -rf bin