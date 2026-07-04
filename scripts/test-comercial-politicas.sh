#!/usr/bin/env bash
set -euo pipefail

export GOCACHE="${GOCACHE:-/tmp/panossoerp-go-build}"

go test ./internal/domain/customer/entity

if [[ -z "${BASE_URL:-}" ]]; then
  echo "BASE_URL not set; HTTP smoke skipped."
  exit 0
fi

TOKEN_HEADER=()
if [[ -n "${AUTH_TOKEN:-}" ]]; then
  TOKEN_HEADER=(-H "Authorization: Bearer ${AUTH_TOKEN}")
fi

curl_json() {
  local method="$1"
  local path="$2"
  local body="${3:-}"
  if [[ -n "$body" ]]; then
    curl -fsS -X "$method" "${BASE_URL}${path}" \
      "${TOKEN_HEADER[@]}" \
      -H "Content-Type: application/json" \
      -d "$body"
  else
    curl -fsS -X "$method" "${BASE_URL}${path}" \
      "${TOKEN_HEADER[@]}" \
      -H "Content-Type: application/json"
  fi
}

created=$(curl_json POST /api/customers/support/commercial-policies/ '{
  "description":"Desconto volume smoke",
  "kind":"DISCOUNT",
  "choice_type":"INFORMATION",
  "calc_type":"PERCENT",
  "percent_value":10,
  "min_quantity":5,
  "priority":1,
  "sequence":1,
  "stackable":true,
  "allow_manual_change":true,
  "used_in_commission":true,
  "applies_to_items":true,
  "data_types_json":["ITEM","CUSTOMER","PAYMENT_TERM"],
  "rule_json":{"source":"smoke"}
}')

code=$(printf '%s' "$created" | sed -n 's/.*"code":\([0-9][0-9]*\).*/\1/p')
if [[ -z "$code" ]]; then
  echo "Could not parse created policy code"
  exit 1
fi

curl_json POST "/api/customers/support/commercial-policies/${code}/specific-items" '{
  "item_code":"1001",
  "block_discount":true,
  "block_surcharge":false,
  "ignore_item_policies":false,
  "block_manual_change":true
}' >/dev/null

curl_json POST "/api/customers/support/commercial-policies/${code}/lines" '{
  "line_number":1,
  "sequence_number":1,
  "description":"Faixa principal",
  "calc_type":"PERCENT",
  "percent_value":10,
  "min_value":0,
  "max_value":1000,
  "variables_json":{"item_code":"1001"}
}' >/dev/null

curl_json POST /api/customers/support/commercial-policies/evaluate '{
  "gross_value":1000,
  "quantity":10,
  "item_code":"1001"
}' | grep -q '"discount_value"'

curl_json GET '/api/customers/support/commercial-policies/?kind=DISCOUNT' >/dev/null

echo "Commercial policy smoke passed."
