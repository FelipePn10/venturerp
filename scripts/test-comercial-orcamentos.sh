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

if [[ -n "${BASE_URL:-}" ]]; then
  echo "==> Fase 3 / Orçamentos: smoke HTTP"
  if [[ -z "${TOKEN:-}" ]]; then
    USER_EMAIL="${USER_EMAIL:-admin@panossoerp.test}"
    USER_PASS="${USER_PASS:-Admin@12345}"
    TOKEN="$(curl -fsS -X POST "${BASE_URL}/users/login" \
      -H "Content-Type: application/json" \
      -d "{\"email\":\"${USER_EMAIL}\",\"password\":\"${USER_PASS}\"}" | jq -r '.token // empty')"
  fi
  if [[ -z "${TOKEN:-}" ]]; then
    echo "Nao foi possivel autenticar para o smoke HTTP" >&2
    exit 1
  fi
  AUTH_HEADER="Authorization: Bearer ${TOKEN}"
  SMOKE_FILE="$(mktemp)"
  SMOKE_DOWNLOAD="$(mktemp)"
  trap 'rm -f "$SMOKE_FILE" "$SMOKE_DOWNLOAD"' EXIT
  printf 'anexo smoke orçamento\n' >"$SMOKE_FILE"

  curl -fsS -X PUT "${BASE_URL}/api/sales-quotation/parameters" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d '{"purchase_order_prompt":"Ordem de Compra","delivery_authorization_prompt":"Autorização de Entr.","allow_service_items_nfce":true,"default_nfce":false,"minimum_cif_freight":"25.50","add_redelivery_to_freight":true}' >/dev/null
  curl -fsS "${BASE_URL}/api/sales-quotation/parameters" -H "$AUTH_HEADER" | jq -e '.minimum_cif_freight == "25.5" or .minimum_cif_freight == "25.50"' >/dev/null

  QUOTE_JSON="$(curl -fsS -X POST "${BASE_URL}/api/sales-quotation/create" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d '{"enterprise_code":1,"status":"OV","quotation_type":"VENDA","currency_code":"BRL","probability_pct":65,"commission_pct":3.5,"purchase_order_number":"OC-FOCCO-SMOKE","freight_type":"Cif-Contrat.","freight_value":10,"redelivery_freight_value":2,"insurance_value":1,"discount_value":5,"surcharge_value":3,"retained_tax_value":0.5,"release_status":"RELEASED","created_by":"00000000-0000-0000-0000-000000000001"}')"
  QUOTE_CODE="$(printf '%s' "$QUOTE_JSON" | jq -r '.code // empty')"
  if [[ -z "$QUOTE_CODE" ]]; then
    echo "Nao foi possivel extrair o codigo do orçamento criado" >&2
    exit 1
  fi
  ITEM_JSON="$(curl -fsS -X POST "${BASE_URL}/api/sales-quotation/items/create" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d "{\"sales_quotation_code\":${QUOTE_CODE},\"sequence\":1,\"item_code\":1,\"requested_qty\":\"2\",\"unit_price\":\"100\",\"discount_pct\":\"5\"}")"
  ITEM_CODE="$(printf '%s' "$ITEM_JSON" | jq -r '.code // empty')"
  [[ -n "$ITEM_CODE" ]]
  curl -fsS "${BASE_URL}/api/sales-quotation/${QUOTE_CODE}" -H "$AUTH_HEADER" >/dev/null
  curl -fsS "${BASE_URL}/api/sales-quotation/list?purchase_order_number=OC-FOCCO-SMOKE&freight_type=Cif-Contrat.&quotation_type=VENDA&limit=20&offset=0" -H "$AUTH_HEADER" >/dev/null
  curl -fsS "${BASE_URL}/api/sales-quotation/report?quotation_type=VENDA&purchase_order_number=OC-FOCCO-SMOKE" -H "$AUTH_HEADER" >/dev/null
  curl -fsS -X POST "${BASE_URL}/api/sales-quotation/cancellation-reasons" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d '{"code":9001,"description":"Teste automatizado","allow_uncancel":true,"require_complement":true}' >/dev/null
  curl -fsS -X DELETE "${BASE_URL}/api/sales-quotation/items/${ITEM_CODE}/cancel" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d '{"reason_code":9001,"complement":"Cancelamento do item no smoke"}' >/dev/null

  ATTACHMENT_JSON="$(curl -fsS -X POST "${BASE_URL}/api/sales-quotation/${QUOTE_CODE}/attachments" \
    -H "$AUTH_HEADER" -F "file=@${SMOKE_FILE};filename=orcamento-smoke.txt;type=text/plain")"
  ATTACHMENT_ID="$(printf '%s' "$ATTACHMENT_JSON" | jq -r '.id // empty')"
  [[ -n "$ATTACHMENT_ID" ]]
  curl -fsS "${BASE_URL}/api/sales-quotation/${QUOTE_CODE}/attachments" -H "$AUTH_HEADER" | jq -e --argjson id "$ATTACHMENT_ID" 'any(.[]; .id == $id)' >/dev/null
  curl -fsS "${BASE_URL}/api/sales-quotation/${QUOTE_CODE}/attachments/${ATTACHMENT_ID}" -H "$AUTH_HEADER" -o "$SMOKE_DOWNLOAD"
  cmp "$SMOKE_FILE" "$SMOKE_DOWNLOAD"

  curl -fsS -X PATCH "${BASE_URL}/api/sales-quotation/${QUOTE_CODE}/release" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d '{"release_status":"BLOCKED","reason":"Bloqueio smoke"}' >/dev/null
  curl -fsS -X PATCH "${BASE_URL}/api/sales-quotation/${QUOTE_CODE}/release" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d '{"release_status":"RELEASED","reason":"Liberação smoke"}' >/dev/null
  curl -fsS -X DELETE "${BASE_URL}/api/sales-quotation/${QUOTE_CODE}/cancel" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d '{"reason_code":9001,"complement":"Cancelamento com histórico"}' >/dev/null
  curl -fsS -X POST "${BASE_URL}/api/sales-quotation/${QUOTE_CODE}/uncancel" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d '{"reason_code":9001,"complement":"Descancelamento autorizado"}' >/dev/null
  curl -fsS -X POST "${BASE_URL}/api/sales-quotation/${QUOTE_CODE}/attend" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d '{"reason":"Teste de atendimento de orçamento","complement":"Atendimento manual do orçamento"}' >/dev/null
  curl -fsS "${BASE_URL}/api/sales-quotation/${QUOTE_CODE}/events" -H "$AUTH_HEADER" | jq -e 'length >= 6' >/dev/null
  DAV_JSON="$(curl -fsS -X POST "${BASE_URL}/api/sales-quotation/${QUOTE_CODE}/dav" -H "$AUTH_HEADER")"
  printf '%s' "$DAV_JSON" | jq -e '.dav_generated_at != null and .can_print_dav_report == true and .can_print_fiscal_receipt == false and .can_print_sales_order == false and .can_send_email == false' >/dev/null
  curl -fsS -X DELETE "${BASE_URL}/api/sales-quotation/${QUOTE_CODE}/attachments/${ATTACHMENT_ID}" -H "$AUTH_HEADER" >/dev/null
  echo "==> Smoke HTTP de Orçamentos concluído"
else
  echo "BASE_URL nao definida; smoke HTTP ignorado."
fi
