# `-include` so the makefile still works in CI / containers without a .env file.
-include .env
export

PWD := $(shell pwd)
MIGRATIONS_DIR := $(PWD)/migrations
BIN_DIR := $(PWD)/bin

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

# ── Build & run ──────────────────────────────────────────────────────────────
build:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(BIN_DIR)/erp ./api

run:
	go run ./api

# ── Quality gates ────────────────────────────────────────────────────────────
vet:
	go vet ./...

# Fails if any file is not gofmt-clean (prints the offenders).
fmt-check:
	@gofmt -l . | (! grep . ) || (echo "files need gofmt (run: gofmt -w .)"; exit 1)

# What CI should run on every push: format, vet, build, unit tests + coverage.
ci: fmt-check vet build test-cover

# ── Docker / deploy ──────────────────────────────────────────────────────────
docker-build:
	docker build -t panossoerp/api:latest .

up:
	docker compose up -d --build

down:
	docker compose down

# Bring the stack up including the scheduled backup sidecar.
up-backup:
	docker compose --profile backup up -d --build

logs:
	docker compose logs -f api

# ── Backup / restore ─────────────────────────────────────────────────────────
# One-off logical backup against DATABASE_URL (custom format, into ./backups).
backup:
	DATABASE_URL="$(DATABASE_URL)" BACKUP_DIR="$(PWD)/backups" ./scripts/backup.sh

# Restore a dump: make restore FILE=./backups/<file>.dump
restore:
	@test -n "$(FILE)" || (echo "usage: make restore FILE=./backups/<file>.dump"; exit 2)
	DATABASE_URL="$(DATABASE_URL)" ./scripts/restore.sh "$(FILE)"

.PHONY: create_migration migrate_up migrate_down migrate_force reset print_db sqlc \
	test test-cover test-integration build run vet fmt-check ci \
	docker-build up down up-backup logs backup restore