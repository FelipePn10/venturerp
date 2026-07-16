# `-include` so the makefile still works in CI / containers without a .env file.
-include .env
export

PWD := $(shell pwd)
MIGRATIONS_DIR := $(PWD)/migrations
BIN_DIR := $(PWD)/bin
GO_CACHE ?= /tmp/panossoerp-go-build
GO_PATH ?= /tmp/panossoerp-go

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
	GOCACHE="$(GO_CACHE)" GOPATH="$(GO_PATH)" go test ./...

test-bom-mrp:
	bash scripts/test-bom-mrp.sh

test-purchase-receiving:
	bash scripts/test-purchase-receiving.sh

test-procurement-governance:
	bash scripts/test-procurement-governance.sh

# Coverage report for unit tests.
test-cover:
	GOCACHE="$(GO_CACHE)" GOPATH="$(GO_PATH)" go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

# Integration tests (require a MIGRATED Postgres). Uses TEST_DATABASE_URL when set,
# falling back to DATABASE_URL from .env. Tests create rows with high unique codes
# and clean up after themselves.
test-integration:
	TEST_DATABASE_URL="$${TEST_DATABASE_URL:-$(DATABASE_URL)}" GOCACHE="$(GO_CACHE)" GOPATH="$(GO_PATH)" go test -tags=integration -count=1 ./...

# End-to-end HTTP test of the Plano de Corte module (1D/2D/true-shape, firmar,
# demanda de OP, export, agenda, rateio). Needs the API running + the test DB.
# Override BASE_URL to point at a running server.
test-cutting:
	BASE_URL="$${BASE_URL:-http://localhost:5071}" bash scripts/test-cutting.sh

# Mapas de corte de amostra — roda várias simulações (1D barras, 2D chapa/MDF,
# true-shape laser) pelos otimizadores reais e grava SVG/DXF/PDF em ./cutting-samples
# para inspeção visual. Não precisa de API nem banco.
cutting-samples:
	go run ./cmd/cutting-samples -out cutting-samples

# Romaneio (expedição) — gera PDF e Excel de teste com base em dados reais do banco.
# Requer API rodando + banco de testes com dados (execute test-e2e.sh primeiro).
test-romaneio:
	BASE_URL="$${BASE_URL:-http://localhost:5071}" bash scripts/test-romaneio.sh

# End-to-end HTTP do Quadro de Programação (APS Gantt): quadro mensal, dependências
# finish-start (explícita + implícita), range livre/zoom semana, export SVG/PDF e
# reschedule (cascata + capacidade). Precisa da API desta branch rodando + DB de
# teste acessível por docker exec (o script semeia operações/sequências/rede via SQL).
test-gantt:
	BASE_URL="$${BASE_URL:-http://localhost:5071}" bash scripts/test-gantt.sh

# ── Build & run ──────────────────────────────────────────────────────────────
build:
	GOCACHE="$(GO_CACHE)" GOPATH="$(GO_PATH)" CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(BIN_DIR)/erp ./api

run:
	go run ./api

# ── Quality gates ────────────────────────────────────────────────────────────
vet:
	GOCACHE="$(GO_CACHE)" GOPATH="$(GO_PATH)" go vet ./...

# Fails if any file is not gofmt-clean (prints the offenders).
fmt-check:
	@find . -path './vendor' -prune -o -name '*.go' -print0 | xargs -0 gofmt -l | (! grep . ) || (echo "files need gofmt (run: gofmt -w <files>)"; exit 1)

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

# ── Ambiente de APRESENTAÇÃO (demo) ──────────────────────────────────────────
# Banco isolado (porta 5434) + API (porta 5072) populado com ~1 ano de operação
# fictícia para demonstrações a clientes. Ver docs/dev/demo-environment.md.
DEMO_COMPOSE      := docker-compose.demo.yml
DEMO_DB_CONTAINER := panossoerp-postgres-demo
DEMO_DB_USER      := panossoerp_demo
DEMO_DB_NAME      := panossoerpdatabase_demo
DEMO_DB_PASS      := panossoerp_demo_pass

# Sobe postgres + migra + api do ambiente demo (constrói a imagem se preciso).
demo-up:
	docker compose -f $(DEMO_COMPOSE) up -d --build

# Derruba os containers (mantém o volume/dados).
demo-down:
	docker compose -f $(DEMO_COMPOSE) down

# Derruba E apaga o volume (zera o banco demo).
demo-reset:
	docker compose -f $(DEMO_COMPOSE) down -v

# Reaplica as migrations no banco demo.
demo-migrate:
	docker compose -f $(DEMO_COMPOSE) run --rm migrate-demo

# Popula o banco demo com dados de apresentação (idempotente; recria tudo).
demo-seed:
	docker exec -i -e PGPASSWORD=$(DEMO_DB_PASS) $(DEMO_DB_CONTAINER) \
		psql -U $(DEMO_DB_USER) -d $(DEMO_DB_NAME) -v ON_ERROR_STOP=1 < scripts/seed-demo.sql

# Atalho: sobe a stack e popula em seguida.
demo-bootstrap: demo-up
	@echo "aguardando postgres-demo ficar saudável..."
	@until [ "$$(docker inspect -f '{{.State.Health.Status}}' $(DEMO_DB_CONTAINER) 2>/dev/null)" = "healthy" ]; do sleep 2; done
	$(MAKE) demo-seed

demo-logs:
	docker compose -f $(DEMO_COMPOSE) logs -f api-demo

# ── Backup / restore ─────────────────────────────────────────────────────────
# One-off logical backup against DATABASE_URL (custom format, into ./backups).
backup:
	DATABASE_URL="$(DATABASE_URL)" BACKUP_DIR="$(PWD)/backups" ./scripts/backup.sh

# Restore a dump: make restore FILE=./backups/<file>.dump
restore:
	@test -n "$(FILE)" || (echo "usage: make restore FILE=./backups/<file>.dump"; exit 2)
	DATABASE_URL="$(DATABASE_URL)" ./scripts/restore.sh "$(FILE)"

# ── Release (distribuição versionada) ────────────────────────────────────────
# Um commit comum NÃO gera atualização. Somente a tag SemVer criada por
# `make release VERSION=x.y.z` dispara o pipeline (imagem GHCR + GitHub Release).
# `release-check` roda a mesma validação (go test + docker build) sem criar tag.
# Ver docs/dev/releases-e-atualizacoes.md.
release-check:
	@test -n "$(VERSION)" || (echo "uso: make release VERSION=1.0.0" >&2; exit 2)
	RELEASE_DRY_RUN=1 ./scripts/release.sh "$(VERSION)"

release:
	@test -n "$(VERSION)" || (echo "uso: make release VERSION=1.0.0" >&2; exit 2)
	./scripts/release.sh "$(VERSION)"

.PHONY: cutting-samples create_migration migrate_up migrate_down migrate_force reset print_db sqlc \
	test test-bom-mrp test-purchase-receiving test-procurement-governance test-cover test-integration test-cutting build run vet fmt-check ci \
	docker-build up down up-backup logs backup restore release release-check \
	demo-up demo-down demo-reset demo-migrate demo-seed demo-bootstrap demo-logs
