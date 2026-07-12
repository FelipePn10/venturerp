#!/usr/bin/env bash
set -euo pipefail

: "${TEST_DATABASE_URL:?Informe TEST_DATABASE_URL para o PostgreSQL de testes}"
export GOCACHE="${GOCACHE:-/tmp/panossoerp-go-build}"

echo "=== Hardening HTTP: CORS, headers, payload e rate limit ==="
go test -count=1 ./internal/interfaces/middleware

echo "=== Segurança de autenticação/JWT ==="
go test -count=1 ./internal/infrastructure/auth

echo "=== Isolamento tenant em estoque, OF, desenhos e compras ==="
go test -tags=integration -count=1 \
  ./internal/infrastructure/repository/stock \
  ./internal/infrastructure/repository/production_order \
  ./internal/application/usecase/drawing_uc \
  ./internal/infrastructure/repository/purchase_order

echo "=== Segurança MRP/manufatura aprovada ==="
