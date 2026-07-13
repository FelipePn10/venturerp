#!/usr/bin/env bash
set -euo pipefail

: "${TEST_DATABASE_URL:?Informe TEST_DATABASE_URL para o PostgreSQL de testes}"
export GOCACHE="${GOCACHE:-/tmp/panossoerp-go-build}"

echo "=== Sequenciamento: domínio, calendário, paradas e concorrência ==="
go test -count=1 -race ./internal/application/usecase/aps_uc
go test -tags=integration -count=1 ./internal/infrastructure/repository/aps

echo "=== Segurança: middleware, JWT e isolamento tenant APS ==="
go test -count=1 ./internal/interfaces/middleware ./internal/infrastructure/auth
go test -tags=integration -count=1 -run 'Tenant|Sequencing' ./internal/infrastructure/repository/aps

echo "=== Sequenciamento de produto aprovado ==="
