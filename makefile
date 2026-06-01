include .env
export

PWD := $(shell pwd)
MIGRATIONS_DIR := $(PWD)/migrations

create_migration:
	migrate create -ext=sql -dir=$(MIGRATIONS_DIR) -seq init

migrate_up:
	migrate -path=$(MIGRATIONS_DIR) \
		-database "$(DATABASE_URL)" \
		-verbose up

migrate_down:
	migrate -path=$(MIGRATIONS_DIR) \
		-database "$(DATABASE_URL)" \
		-verbose down

migrate_force:
	migrate -path=$(MIGRATIONS_DIR) \
		-database "$(DATABASE_URL)" \
		force 1

reset:
	migrate -path=$(MIGRATIONS_DIR) \
		-database "$(DATABASE_URL)" \
		-drop -verbose

print_db:
	@echo $(DATABASE_URL)

sqlc:
	sqlc generate
	go run scripts/fix_sqlc_output.go

# Unit tests (no database) — fast, run on every change.
test:
	go test ./...

# Coverage report for unit tests.
test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

# Integration tests (require a MIGRATED Postgres). Uses TEST_DATABASE_URL when set,
# falling back to DATABASE_URL from .env. Tests create rows with high unique codes
# and clean up after themselves.
test-integration:
	TEST_DATABASE_URL="$${TEST_DATABASE_URL:-$(DATABASE_URL)}" go test -tags=integration -count=1 ./...

.PHONY: create_migration migrate_up migrate_down migrate_force reset print_db sqlc test test-cover test-integration