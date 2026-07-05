#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

export GOCACHE="${GOCACHE:-/tmp/panossoerp-go-build}"

echo "==> Fase 5 / Representantes: testes Go focados"
go test \
  ./internal/application/usecase/representative_uc \
  ./internal/domain/representative/... \
  ./internal/infrastructure/repository/representative \
  ./internal/interfaces/http/handler \
  ./api

if [[ -n "${BASE_URL:-}" && -n "${TOKEN:-}" ]]; then
  echo "==> Fase 5 / Representantes: smoke HTTP"
  AUTH_HEADER="Authorization: Bearer ${TOKEN}"

  TYPE_JSON="$(curl -fsS -X POST "${BASE_URL}/api/representatives/types/" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d '{"description":"Representante Externo Smoke","is_free":false,"ignores_direct_billing":true}')"
  TYPE_CODE="$(printf '%s' "$TYPE_JSON" | sed -n 's/.*"code":\([0-9][0-9]*\).*/\1/p')"
  if [[ -z "$TYPE_CODE" ]]; then
    echo "Nao foi possivel extrair o tipo de representante criado" >&2
    exit 1
  fi

  REP_JSON="$(curl -fsS -X POST "${BASE_URL}/api/representatives/create" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d "{\"name\":\"Representante Smoke\",\"trade_name\":\"REP Smoke\",\"type_code\":${TYPE_CODE},\"document_number\":\"12345678901\",\"state\":\"rs\",\"city\":\"Caxias do Sul\",\"street\":\"Rua Comercial\",\"street_number\":\"100\",\"device_quantity\":1}")"
  REP_CODE="$(printf '%s' "$REP_JSON" | sed -n 's/.*"code":\([0-9][0-9]*\).*/\1/p')"
  if [[ -z "$REP_CODE" ]]; then
    echo "Nao foi possivel extrair o representante criado" >&2
    exit 1
  fi

  curl -fsS -X POST "${BASE_URL}/api/representatives/enterprises" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d "{\"representative_code\":${REP_CODE},\"enterprise_code\":1,\"enterprise_name\":\"Empresa Smoke\",\"commission_pct\":5,\"is_default\":true,\"is_active\":true}" >/dev/null
  curl -fsS -X POST "${BASE_URL}/api/representatives/accounting" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d "{\"representative_code\":${REP_CODE},\"event_type\":\"GENERATED\",\"debit_account_code\":101,\"credit_account_code\":201,\"history_code\":301}" >/dev/null
  curl -fsS -X POST "${BASE_URL}/api/representatives/regions" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d "{\"representative_code\":${REP_CODE},\"enterprise_code\":1,\"region_code\":10,\"microregion_code\":100,\"is_active\":true}" >/dev/null
  curl -fsS -X POST "${BASE_URL}/api/representatives/segments" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d "{\"representative_code\":${REP_CODE},\"enterprise_code\":1,\"microregion_code\":100,\"market_segment_code\":20,\"is_active\":true}" >/dev/null
  curl -fsS -X POST "${BASE_URL}/api/representatives/sales-plans" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d "{\"representative_code\":${REP_CODE},\"enterprise_code\":1,\"microregion_code\":100,\"sales_plan_code\":30,\"is_active\":true}" >/dev/null
  curl -fsS -X POST "${BASE_URL}/api/representatives/interests" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d "{\"representative_code\":${REP_CODE},\"item_classification_code\":40,\"is_active\":true}" >/dev/null
  curl -fsS -X POST "${BASE_URL}/api/representatives/phones" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d "{\"representative_code\":${REP_CODE},\"ddi\":\"55\",\"ddd\":\"54\",\"phone\":\"3333-3333\",\"phone_type\":\"COMERCIAL\",\"ranking\":1}" >/dev/null
  curl -fsS -X POST "${BASE_URL}/api/representatives/emails" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d "{\"representative_code\":${REP_CODE},\"email\":\"representante.smoke@example.com\",\"ranking\":1}" >/dev/null
  curl -fsS -X POST "${BASE_URL}/api/representatives/correspondence-addresses" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d "{\"representative_code\":${REP_CODE},\"postal_code\":\"95000-000\",\"city\":\"Caxias do Sul\",\"state\":\"RS\",\"street\":\"Rua Correspondencia\",\"street_number\":\"200\",\"district\":\"Centro\",\"is_default\":true}" >/dev/null
  curl -fsS -X POST "${BASE_URL}/api/representatives/contacts" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d "{\"representative_code\":${REP_CODE},\"name\":\"Contato Smoke\",\"role\":\"Preposto\",\"phone\":\"3333-4444\",\"email\":\"contato.smoke@example.com\",\"is_active\":true}" >/dev/null

  curl -fsS "${BASE_URL}/api/representatives/${REP_CODE}" -H "$AUTH_HEADER" >/dev/null
  curl -fsS "${BASE_URL}/api/representatives/report?codes=${REP_CODE}&with_accounts=true&sort_by=REGION" -H "$AUTH_HEADER" >/dev/null
  curl -fsS "${BASE_URL}/api/representatives/follow-up?representative_codes=${REP_CODE}" -H "$AUTH_HEADER" >/dev/null
else
  echo "BASE_URL/TOKEN nao definidos; smoke HTTP ignorado."
fi
