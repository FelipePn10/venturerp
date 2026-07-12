#!/usr/bin/env bash
set -euo pipefail

# Aceite reproduzível do planejamento de materiais e manufatura.
# Requer: Go, migrate e PostgreSQL de testes já iniciado.
# Variável obrigatória: TEST_DATABASE_URL.

: "${TEST_DATABASE_URL:?Informe TEST_DATABASE_URL para o PostgreSQL de testes}"
export GOCACHE="${GOCACHE:-/tmp/panossoerp-go-build}"

echo "=== Migrações MRP/manufatura ==="
migrate -path migrations -database "$TEST_DATABASE_URL" version

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
