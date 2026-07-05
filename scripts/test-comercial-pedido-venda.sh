#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

export GOCACHE="${GOCACHE:-/tmp/panossoerp-go-build}"

echo "==> Fase 4 / Pedido de Venda: testes Go focados"
go test \
  ./internal/application/usecase/sales_order_uc \
  ./internal/domain/sales_order/... \
  ./internal/infrastructure/repository/sales_order \
  ./internal/interfaces/http/handler \
  ./api

if [[ -n "${BASE_URL:-}" && -n "${TOKEN:-}" ]]; then
  echo "==> Fase 4 / Pedido de Venda: smoke HTTP"
  AUTH_HEADER="Authorization: Bearer ${TOKEN}"
  USER_UUID="${USER_UUID:-00000000-0000-0000-0000-000000000001}"

  SO_JSON="$(curl -fsS -X POST "${BASE_URL}/api/sales-order/create" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d "{\"enterprise_code\":1,\"status\":\"R\",\"origin\":\"NORMAL\",\"currency_code\":\"BRL\",\"commission_pct\":2.5,\"is_nfce\":false,\"freight_type\":\"Cif-Contrat.\",\"freight_value\":12.5,\"insurance_value\":1.5,\"project_code\":\"FASE4\",\"project_name\":\"Smoke Pedido Venda\",\"created_by\":\"${USER_UUID}\"}")"
  SO_CODE="$(printf '%s' "$SO_JSON" | sed -n 's/.*"code":\([0-9][0-9]*\).*/\1/p')"
  if [[ -z "$SO_CODE" ]]; then
    echo "Nao foi possivel extrair o codigo do pedido criado" >&2
    exit 1
  fi

  curl -fsS -X POST "${BASE_URL}/api/sales-order/items/create" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d "{\"sales_order_code\":${SO_CODE},\"sequence\":1,\"item_code\":1,\"requested_qty\":2,\"unit_price\":100,\"discount_pct\":5}" >/dev/null
  curl -fsS "${BASE_URL}/api/sales-order/search?status=R&conference_status=PENDING" -H "$AUTH_HEADER" >/dev/null
  curl -fsS "${BASE_URL}/api/sales-order/report?status=R" -H "$AUTH_HEADER" >/dev/null
  curl -fsS -X POST "${BASE_URL}/api/sales-order/${SO_CODE}/analyze" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d "{\"area\":\"COMMERCIAL\",\"status\":\"APPROVED\",\"reason\":\"Smoke fase 4\",\"created_by\":\"${USER_UUID}\"}" >/dev/null
  curl -fsS -X POST "${BASE_URL}/api/sales-order/${SO_CODE}/release" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d "{\"area\":\"COMMERCIAL\",\"release_status\":\"RELEASED\",\"reason\":\"Smoke fase 4\",\"created_by\":\"${USER_UUID}\"}" >/dev/null
  curl -fsS -X POST "${BASE_URL}/api/sales-order/${SO_CODE}/conference" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d "{\"status\":\"CONFERRED\",\"reason\":\"Conferido no smoke\",\"created_by\":\"${USER_UUID}\"}" >/dev/null
  curl -fsS -X POST "${BASE_URL}/api/sales-order/${SO_CODE}/delay-reason" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d "{\"reason\":\"Sem atraso real - smoke\",\"action\":\"Monitorar\",\"created_by\":\"${USER_UUID}\"}" >/dev/null
  curl -fsS -X DELETE "${BASE_URL}/api/sales-order/${SO_CODE}/cancel" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d '{"reason":"Cancelamento smoke fase 4","complement":"Teste automatizado"}' >/dev/null
else
  echo "BASE_URL/TOKEN nao definidos; smoke HTTP ignorado."
fi
