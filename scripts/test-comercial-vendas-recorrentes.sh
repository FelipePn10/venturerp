#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

export GOCACHE="${GOCACHE:-/tmp/panossoerp-go-build}"

echo "==> Fase 12 / Vendas Recorrentes: testes Go focados"
go test \
  ./internal/application/usecase/recurring_sales_uc \
  ./internal/infrastructure/repository/recurring_sales \
  ./internal/interfaces/http/handler \
  ./api

echo "==> Fase 12 / Vendas Recorrentes: validacao estatica"
test -f migrations/000195_recurring_sales.up.sql
test -f migrations/000195_recurring_sales.down.sql
grep -q "recurring_sales_parameters" migrations/000195_recurring_sales.up.sql
grep -q "recurring_sales_adjustment_dates" migrations/000195_recurring_sales.up.sql
grep -q "recurring_sales_representatives" migrations/000195_recurring_sales.up.sql
grep -q "recurring_sales_adjustment_links" migrations/000195_recurring_sales.up.sql
grep -q "CalculateAdjustment" internal/application/usecase/recurring_sales_uc/recurring_sales_uc.go
grep -q "RevenueProjection" internal/application/usecase/recurring_sales_uc/recurring_sales_uc.go
grep -q "CommissionProjection" internal/application/usecase/recurring_sales_uc/recurring_sales_uc.go
grep -q 'Route("/api/recurring-sales"' api/api.go
grep -q "Vendas Recorrentes" docs/dev/vendas.md
grep -q "Vendas Recorrentes" docs/apresentacao/vendas.md

if [[ -n "${BASE_URL:-}" && -n "${TOKEN:-}" ]]; then
  echo "==> Fase 12 / Vendas Recorrentes: smoke HTTP"
  AUTH_HEADER="Authorization: Bearer ${TOKEN}"
  CREATED_BY="${CREATED_BY:-00000000-0000-0000-0000-000000000000}"
  SMOKE_YEAR="${SMOKE_YEAR:-2027}"
  SMOKE_MONTH="$(printf '%02d' $((1 + RANDOM % 12)))"
  SMOKE_DAY="$(printf '%02d' $((1 + RANDOM % 28)))"
  ADJUSTMENT_DATE="${SMOKE_YEAR}-${SMOKE_MONTH}-${SMOKE_DAY}"
  REPRESENTATIVE_CODE="$(curl -fsS "${BASE_URL}/api/representatives/list?active=true" -H "$AUTH_HEADER" | jq -r 'if type == "array" then last.code else (.data // .items // []) | last | .code end // empty')"
  [[ -n "$REPRESENTATIVE_CODE" ]]
  PRICE_TABLE_CODE="$(curl -fsS "${BASE_URL}/api/customers/support/sales-tables/" -H "$AUTH_HEADER" | jq -r 'map(select(.is_active == true)) | last | .code // empty')"
  [[ -n "$PRICE_TABLE_CODE" ]]

  curl -fsS -X PUT "${BASE_URL}/api/recurring-sales/parameters" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d "{\"enterprise_code\":1,\"current_month_billing_limit_day\":10,\"indefinite_delivery_day\":10,\"fixed_term_delivery_day\":10,\"updated_by\":\"${CREATED_BY}\"}" >/dev/null

  curl -fsS -X POST "${BASE_URL}/api/recurring-sales/adjustment-dates" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d "{\"enterprise_code\":1,\"customer_code\":1,\"adjustment_date\":\"${ADJUSTMENT_DATE}\",\"created_by\":\"${CREATED_BY}\"}" >/dev/null

  REC_JSON="$(curl -fsS -X POST "${BASE_URL}/api/recurring-sales/create" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d "{\"enterprise_code\":1,\"customer_code\":1,\"item_code\":\"10001\",\"sales_plan_code\":1,\"price_table_code\":${PRICE_TABLE_CODE},\"movement_type\":\"SALE\",\"term_type\":\"INDEFINITE\",\"sale_date\":\"2026-07-01\",\"next_adjustment_date\":\"${ADJUSTMENT_DATE}\",\"quantity\":1,\"unit_value\":199.90,\"created_by\":\"${CREATED_BY}\",\"representatives\":[{\"representative_code\":${REPRESENTATIVE_CODE},\"is_primary\":true,\"commission_percent\":5,\"commission_base\":\"ADJUSTED\",\"is_lifetime\":true}]}")"
  REC_CODE="$(printf '%s' "$REC_JSON" | jq -r '.code // empty')"
  if [[ -z "$REC_CODE" ]]; then
    echo "Nao foi possivel extrair a venda recorrente criada" >&2
    exit 1
  fi

  curl -fsS -X POST "${BASE_URL}/api/recurring-sales/${REC_CODE}/generate-order" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" -H "Idempotency-Key: recurring-order-${REC_CODE}-${SMOKE_YEAR}" \
    -d "{\"created_by\":\"${CREATED_BY}\",\"sales_uom\":\"UN\",\"price_table_code\":${PRICE_TABLE_CODE},\"confirm_order\":false}" >/dev/null

  curl -fsS "${BASE_URL}/api/recurring-sales/monthly-revenue?from=2026-07-01&to=2026-12-01&customer_code=1" \
    -H "$AUTH_HEADER" >/dev/null

  curl -fsS "${BASE_URL}/api/recurring-sales/future-commissions?from=2026-07-01&to=2026-12-01&representative_code=${REPRESENTATIVE_CODE}" \
    -H "$AUTH_HEADER" >/dev/null

  curl -fsS -X POST "${BASE_URL}/api/recurring-sales/${REC_CODE}/cancel" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d "{\"reason\":\"Encerramento do smoke recorrente\",\"effective_date\":\"${ADJUSTMENT_DATE}\",\"future_orders_policy\":\"NAO_GERAR_NOVOS\",\"correlation_id\":\"smoke-recurring-${REC_CODE}\"}" | jq -e '.lifecycle_status == "CANCELADA" and (.allowed_actions | type == "array")' >/dev/null
else
  echo "BASE_URL/TOKEN nao definidos; smoke HTTP ignorado."
fi
