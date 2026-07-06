#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

export GOCACHE="${GOCACHE:-/tmp/panossoerp-go-build}"

echo "==> Fase 14 / Faturamento: testes Go focados"
go test \
  ./internal/application/usecase/fiscal_uc \
  ./internal/domain/fiscal/... \
  ./internal/domain/shipment/... \
  ./internal/interfaces/http/handler \
  ./api

echo "==> Fase 14 / Faturamento: validacao estatica"
test -f migrations/000197_fiscal_exit_sources.up.sql
test -f migrations/000197_fiscal_exit_sources.down.sql
grep -q "shipment_load_code" migrations/000197_fiscal_exit_sources.up.sql
grep -q "fiscal_coupon_number" migrations/000197_fiscal_exit_sources.up.sql
grep -q "CreateFiscalExitFromLoadUseCase" internal/application/usecase/fiscal_uc/create_fiscal_exit_from_load_uc.go
grep -q 'Post("/exits/from-load"' api/api.go
grep -q "emissão por carga" docs/dev/fiscal-financeiro.md
grep -qi "faturamento por carga" docs/apresentacao/fiscal-financeiro.md

if [[ -n "${BASE_URL:-}" && -n "${TOKEN:-}" ]]; then
  echo "==> Fase 14 / Faturamento: smoke HTTP de leitura"
  AUTH_HEADER="Authorization: Bearer ${TOKEN}"

  curl -fsS "${BASE_URL}/api/fiscal/exits/list" -H "$AUTH_HEADER" >/dev/null
  curl -fsS "${BASE_URL}/api/shipments/loads/logistic-panel" -H "$AUTH_HEADER" >/dev/null
else
  echo "BASE_URL/TOKEN nao definidos; smoke HTTP ignorado."
fi
