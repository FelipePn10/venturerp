#!/usr/bin/env bash
set -euo pipefail

# Requer: Go 1.25+, PostgreSQL de teste migrado até a versão 236.
: "${TEST_DATABASE_URL:?Informe TEST_DATABASE_URL para o PostgreSQL de testes}"
export GOCACHE="${GOCACHE:-/tmp/panossoerp-go-build}"

echo "=== Serviços de terceiros: domínio, aplicação e HTTP ==="
go test -count=1 -race ./internal/domain/third_party_service
go test -count=1 -race ./internal/application/usecase/third_party_service_uc ./internal/application/usecase/cost_uc ./internal/application/usecase/routing_uc
go test -count=1 ./internal/interfaces/http/handler -run ThirdParty

echo "=== Serviços de terceiros: persistência, tenant e atomicidade ==="
go test -tags=integration -count=1 \
  ./internal/infrastructure/repository/third_party_service \
  ./internal/infrastructure/repository/item_conversion \
  ./internal/infrastructure/repository/routing \
  ./internal/infrastructure/repository/production_order

echo "=== Serviços de terceiros: integração com planejamento e compras ==="
go test -count=1 ./internal/domain/mrp_calculation/service ./internal/application/usecase/mrp_uc ./internal/application/usecase/planned_order_uc

echo "=== Serviços de terceiros aprovado ==="
