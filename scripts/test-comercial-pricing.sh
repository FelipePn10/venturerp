#!/usr/bin/env bash
# Focused validation for Comercial fase 1 (Precificacao):
# - pure price formation math
# - optional HTTP smoke for sales table CRUD, table prices and pricing endpoints
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

export GOCACHE="${GOCACHE:-/tmp/panossoerp-go-build}"
export GOPATH="${GOPATH:-/tmp/panossoerp-go}"
export GOFLAGS="${GOFLAGS:--mod=vendor}"

echo "== unit: comercial pricing =="
go test ./internal/domain/customer/entity ./internal/application/usecase/customer_uc

BASE="${BASE_URL:-}"
if [ -z "$BASE" ]; then
  echo "== HTTP smoke skipped (set BASE_URL and a running server to enable) =="
  exit 0
fi

USER_EMAIL="${USER_EMAIL:-admin@panossoerp.demo}"
USER_PASS="${USER_PASS:-admin123}"

echo "== HTTP smoke against $BASE =="
if ! curl -sf "$BASE/health" >/dev/null 2>&1; then
  echo "server not reachable at $BASE/health - aborting smoke"
  exit 1
fi

TOKEN=$(curl -sf -X POST "$BASE/users/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$USER_EMAIL\",\"password\":\"$USER_PASS\"}" | jq -r '.token // empty')
if [ -z "$TOKEN" ]; then
  echo "login failed"
  exit 1
fi
AUTH=(-H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json")

TABLE=$(curl -s -X POST "$BASE/api/customers/sales-tables" "${AUTH[@]}" \
  -d '{"description":"Tabela Comercial Smoke","price_formation":"INFORMADO","decimal_places":2,"composition":"FOB","table_type":"NORMAL","base_date":"PEDIDO"}')
TABLE_CODE=$(echo "$TABLE" | jq -r '.code // empty')
TABLE_ID=$(echo "$TABLE" | jq -r '.id // empty')
if [ -z "$TABLE_CODE" ] || [ -z "$TABLE_ID" ]; then
  echo "failed to create sales table: $TABLE"
  exit 1
fi
echo "  ok  create sales table code=$TABLE_CODE"

PRICE=$(curl -s -X POST "$BASE/api/customers/sales-tables/$TABLE_CODE/prices" "${AUTH[@]}" \
  -d '{"item_code":"SMOKE-PRC-1","price":123.45,"ume":"UN","situation":"ATIVO"}')
PRICE_ID=$(echo "$PRICE" | jq -r '.id // empty')
if [ -z "$PRICE_ID" ]; then
  echo "failed to create table price: $PRICE"
  exit 1
fi
echo "  ok  create table price id=$PRICE_ID"

UNIT=$(curl -s -X POST "$BASE/api/customers/sales-tables/pricing" "${AUTH[@]}" \
  -d "{\"sales_table_code\":$TABLE_CODE,\"item_code\":\"SMOKE-PRC-1\",\"quantity\":2}" | jq -r '.unit_price // empty')
if [ "$UNIT" != "123.45" ]; then
  echo "pricing returned unit_price=$UNIT, want 123.45"
  exit 1
fi
echo "  ok  pricing by table"

SUGGESTED=$(curl -s -X POST "$BASE/api/customers/sales-tables/price-formation" "${AUTH[@]}" \
  -d "{\"sales_table_code\":$TABLE_CODE,\"base_cost\":100,\"margin_pct\":20,\"taxes_pct\":10,\"commission_pct\":5}" | jq -r '.suggested_price // empty')
if [ "$SUGGESTED" != "153.85" ]; then
  echo "price formation returned suggested_price=$SUGGESTED, want 153.85"
  exit 1
fi
echo "  ok  price formation"

POLICY=$(curl -s -X POST "$BASE/api/customers/sales-price-policies" "${AUTH[@]}" \
  -d "{\"description\":\"Politica Smoke\",\"cost_source\":\"INFORMED\",\"margin_pct\":20,\"taxes_pct\":10,\"commission_pct\":5,\"sales_table_code\":$TABLE_CODE}")
POLICY_CODE=$(echo "$POLICY" | jq -r '.code // empty')
if [ -z "$POLICY_CODE" ]; then
  echo "failed to create sales price policy: $POLICY"
  exit 1
fi
echo "  ok  create price policy code=$POLICY_CODE"

POLICY_SUGGESTED=$(curl -s -X POST "$BASE/api/customers/sales-tables/price-formation" "${AUTH[@]}" \
  -d "{\"sales_table_code\":$TABLE_CODE,\"policy_code\":$POLICY_CODE,\"base_cost\":100}" | jq -r '.suggested_price // empty')
if [ "$POLICY_SUGGESTED" != "153.85" ]; then
  echo "policy price formation returned suggested_price=$POLICY_SUGGESTED, want 153.85"
  exit 1
fi
echo "  ok  policy price formation"

GEN=$(curl -s -X POST "$BASE/api/customers/sales-tables/generate-prices" "${AUTH[@]}" \
  -d "{\"sales_table_code\":$TABLE_CODE,\"policy_code\":$POLICY_CODE,\"item_codes\":[\"SMOKE-PRC-1\"],\"reason\":\"smoke\"}")
WARNINGS=$(echo "$GEN" | jq -r '.warnings | length // 0')
if [ "$WARNINGS" = "0" ]; then
  echo "  ok  generate prices"
else
  echo "  note generate-prices returned warnings (expected when policy cost_source requires real cost): $GEN"
fi

echo "== HTTP smoke passed =="
