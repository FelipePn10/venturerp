#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

export GOCACHE="${GOCACHE:-/tmp/panossoerp-go-build}"

echo "==> Fase 7 / Previsao de Vendas: testes Go focados"
go test \
  ./internal/application/usecase/sales_forecast_uc \
  ./internal/application/usecase/forecast_uc \
  ./internal/domain/sales_forecast/... \
  ./internal/infrastructure/repository/sales_forecast \
  ./internal/interfaces/http/handler \
  ./api

echo "==> Fase 7 / Previsao de Vendas: validacao estatica"
grep -q "sales_forecasts" migrations/000058_init.up.sql
grep -q "sales_forecast_blocks" migrations/000058_init.up.sql
grep -q "appropriation_tables" migrations/000058_init.up.sql
grep -q "GenerateSalesForecastUseCase" internal/application/usecase/sales_forecast_uc/generate_forecast.go
grep -q "CreateMonthlySalesForecastUseCase" internal/application/usecase/sales_forecast_uc/generate_forecast.go
grep -q "ListForecastSalesOrderHistory" internal/infrastructure/database/queries/sales_forecast.sql
grep -q "ListForecastFiscalHistory" internal/infrastructure/database/queries/sales_forecast.sql
grep -q 'Post("/create-monthly", salesForecastHandler.CreateMonthlyForecast)' api/api.go
grep -q 'Post("/generate", salesForecastHandler.GenerateForecast)' api/api.go
grep -q "Previsão de Vendas" docs/dev/vendas.md
grep -q "Previsão de Vendas" docs/apresentacao/vendas.md

if [[ -n "${BASE_URL:-}" && -n "${TOKEN:-}" ]]; then
  echo "==> Fase 7 / Previsao de Vendas: smoke HTTP"
  AUTH_HEADER="Authorization: Bearer ${TOKEN}"

  curl -fsS -X POST "${BASE_URL}/api/sales-forecast/blocks/create" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d '{"start_date":"2026-08-03","end_date":"2026-08-09","reason":"Smoke bloqueio previsao"}' >/dev/null

  curl -fsS -X POST "${BASE_URL}/api/sales-forecast/appropriation/create" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d '{"description":"Apropriacao smoke","monday_pct":20,"tuesday_pct":20,"wednesday_pct":20,"thursday_pct":20,"friday_pct":20,"saturday_pct":0,"sunday_pct":0,"is_default":true}' >/dev/null

  curl -fsS -X POST "${BASE_URL}/api/sales-forecast/create-monthly" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d '{"item_code":1,"year":2026,"month":7,"quantity":100,"accepts_fraction":true,"update_existing":true}' >/dev/null

  curl -fsS -X POST "${BASE_URL}/api/sales-forecast/generate" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d '{"item_code":1,"start_week":31,"start_year":2026,"periods":3,"model":"MOVING_AVERAGE","ma_window":3,"update_existing":true,"history":[{"period":"2026-04","quantity":90},{"period":"2026-05","quantity":100},{"period":"2026-06","quantity":110},{"period":"2026-07","quantity":120}]}' >/dev/null

  curl -fsS "${BASE_URL}/api/sales-forecast/list/2026" -H "$AUTH_HEADER" >/dev/null
  curl -fsS "${BASE_URL}/api/sales-forecast/item/1" -H "$AUTH_HEADER" >/dev/null
  curl -fsS "${BASE_URL}/api/sales-forecast/blocks/list" -H "$AUTH_HEADER" >/dev/null
  curl -fsS "${BASE_URL}/api/sales-forecast/appropriation/list" -H "$AUTH_HEADER" >/dev/null
else
  echo "BASE_URL/TOKEN nao definidos; smoke HTTP ignorado."
fi
