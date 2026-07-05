#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

export GOCACHE="${GOCACHE:-/tmp/panossoerp-go-build}"

echo "==> Fase 3 / Orçamentos: testes Go focados"
go test \
  ./internal/application/usecase/sales_quotation_uc \
  ./internal/domain/sales_quotation/... \
  ./internal/infrastructure/repository/sales_quotation \
  ./internal/interfaces/http/handler

if [[ -n "${BASE_URL:-}" && -n "${TOKEN:-}" ]]; then
  echo "==> Fase 3 / Orçamentos: smoke HTTP"
  AUTH_HEADER="Authorization: Bearer ${TOKEN}"
  QUOTE_JSON="$(curl -fsS -X POST "${BASE_URL}/api/sales-quotation/create" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d '{"enterprise_code":1,"status":"OF","quotation_type":"VENDA","currency_code":"BRL","probability_pct":65,"commission_pct":3.5,"purchase_order_number":"OC-FOCCO-SMOKE","freight_type":"Cif-Contrat.","freight_value":10,"redelivery_freight_value":2,"insurance_value":1,"discount_value":5,"surcharge_value":3,"retained_tax_value":0.5,"release_status":"RELEASED","created_by":"00000000-0000-0000-0000-000000000001"}')"
  QUOTE_CODE="$(printf '%s' "$QUOTE_JSON" | sed -n 's/.*"code":\([0-9][0-9]*\).*/\1/p')"
  if [[ -z "$QUOTE_CODE" ]]; then
    echo "Nao foi possivel extrair o codigo do orçamento criado" >&2
    exit 1
  fi
  curl -fsS -X POST "${BASE_URL}/api/sales-quotation/items/create" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d "{\"sales_quotation_code\":${QUOTE_CODE},\"sequence\":1,\"item_code\":1,\"requested_qty\":2,\"unit_price\":100,\"discount_pct\":5}" >/dev/null
  curl -fsS "${BASE_URL}/api/sales-quotation/${QUOTE_CODE}" -H "$AUTH_HEADER" >/dev/null
  curl -fsS "${BASE_URL}/api/sales-quotation/list?purchase_order_number=OC-FOCCO-SMOKE&freight_type=Cif-Contrat." -H "$AUTH_HEADER" >/dev/null
  curl -fsS "${BASE_URL}/api/sales-quotation/report?purchase_order_number=OC-FOCCO-SMOKE" -H "$AUTH_HEADER" >/dev/null
  curl -fsS -X DELETE "${BASE_URL}/api/sales-quotation/${QUOTE_CODE}/cancel" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d '{"reason":"Teste FPDV0205 ORC","complement":"Cancelamento com histórico"}' >/dev/null
  curl -fsS -X POST "${BASE_URL}/api/sales-quotation/${QUOTE_CODE}/uncancel" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d '{"reason":"Teste FPDV0205 ORC","complement":"Descancelamento autorizado"}' >/dev/null
  curl -fsS -X POST "${BASE_URL}/api/sales-quotation/${QUOTE_CODE}/attend" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d '{"reason":"Teste FPDV0205 ORC","complement":"Atendimento manual do orçamento"}' >/dev/null
else
  echo "BASE_URL/TOKEN nao definidos; smoke HTTP ignorado."
fi
