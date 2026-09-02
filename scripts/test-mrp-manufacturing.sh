#!/usr/bin/env bash
set -euo pipefail

# Aceite reproduzível do planejamento de materiais e manufatura.
# Requer: Go, migrate, psql e PostgreSQL de testes já iniciado e migrado.
# Variável obrigatória: TEST_DATABASE_URL.

: "${TEST_DATABASE_URL:?Informe TEST_DATABASE_URL para o PostgreSQL de testes}"
export GOCACHE="${GOCACHE:-/tmp/panossoerp-go-build}"

for command in go migrate psql; do
  command -v "$command" >/dev/null || { echo "dependência ausente: $command" >&2; exit 1; }
done

database_name="$(psql "$TEST_DATABASE_URL" -Atqc 'SELECT current_database()')"
if [[ "$database_name" != *test* ]]; then
  echo "recusado: TEST_DATABASE_URL deve apontar para um banco cujo nome contenha 'test' (atual: $database_name)" >&2
  exit 1
fi

echo "=== Migrações MRP/manufatura ==="
migrate -path migrations -database "$TEST_DATABASE_URL" up
migrate -path migrations -database "$TEST_DATABASE_URL" version

echo "=== Round-trip da última migration ==="
migrate -path migrations -database "$TEST_DATABASE_URL" down 1
migrate -path migrations -database "$TEST_DATABASE_URL" up 1

echo "=== Seed mínimo idempotente ==="
psql "$TEST_DATABASE_URL" -f scripts/seed-mrp-manufacturing-test.sql

echo "=== Testes unitários focados ==="
go test -count=1 \
  ./internal/domain/mrp_calculation/service \
  ./internal/application/usecase/mrp_calculation_uc \
  ./internal/application/usecase/mrp_uc \
  ./internal/application/usecase/mrp_report_uc \
  ./internal/application/usecase/planned_order_uc \
  ./internal/application/usecase/production_plan_uc \
  ./internal/application/usecase/production_order_uc \
  ./internal/application/usecase/drawing_uc \
  ./internal/application/usecase/purchase_order_uc

echo "=== Regressão global ==="
go test -count=1 ./...

echo "=== Análise estática ==="
go vet ./...

echo "=== Testes integrados: tenant, rollback, OF/OCS, manutenção, refugo, WMS, lotes e relatórios ==="
go test -tags=integration -count=1 \
  ./internal/infrastructure/repository/mrp_calculation \
  ./internal/infrastructure/repository/production_plan \
  ./internal/infrastructure/repository/production_order \
  ./internal/infrastructure/repository/mrp_report \
  ./internal/infrastructure/repository/purchase_order \
  ./internal/application/usecase/production_order_uc \
  ./internal/application/usecase/drawing_uc \
  ./internal/application/usecase/purchase_requisition_uc

echo "=== Concorrência e condições de corrida ==="
go test -race -count=1 \
  ./internal/application/usecase/mrp_calculation_uc \
  ./internal/application/usecase/planned_order_uc \
  ./internal/application/usecase/production_order_uc

echo "=== Cobertura focada ==="
go test -cover \
  ./internal/domain/mrp_calculation/service \
  ./internal/application/usecase/mrp_calculation_uc \
  ./internal/application/usecase/mrp_uc \
  ./internal/application/usecase/mrp_report_uc \
  ./internal/application/usecase/planned_order_uc \
  ./internal/application/usecase/production_plan_uc \
  ./internal/application/usecase/production_order_uc \
  ./internal/application/usecase/drawing_uc \
  ./internal/application/usecase/purchase_order_uc

echo "=== Aceite MRP/manufatura concluído ==="
