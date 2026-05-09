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

.PHONY: create_migration migrate_up migrate_down migrate_force reset print_db