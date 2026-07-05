#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

export GOCACHE="${GOCACHE:-/tmp/panossoerp-go-build}"

echo "==> Fase 6 / Metas de Vendas: testes Go focados"
go test \
  ./internal/application/usecase/sales_goal_uc \
  ./internal/domain/sales_goal/... \
  ./internal/infrastructure/repository/sales_goal \
  ./internal/interfaces/http/handler \
  ./api

echo "==> Fase 6 / Metas de Vendas: validacao estatica"
test -f migrations/000191_sales_goals.up.sql
test -f migrations/000191_sales_goals.down.sql
grep -q "sales_goal_periods" migrations/000191_sales_goals.up.sql
grep -q "sales_goal_group_targets" migrations/000191_sales_goals.up.sql
grep -q "/api/sales-goals" api/api.go

if [[ -n "${BASE_URL:-}" && -n "${TOKEN:-}" ]]; then
  echo "==> Fase 6 / Metas de Vendas: smoke HTTP"
  AUTH_HEADER="Authorization: Bearer ${TOKEN}"

  PERIOD_JSON="$(curl -fsS -X POST "${BASE_URL}/api/sales-goals/periods/" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d '{"description":"Julho 2026 Smoke","period_type":"MONTH","start_date":"2026-07-01","end_date":"2026-07-31"}')"
  PERIOD_CODE="$(printf '%s' "$PERIOD_JSON" | sed -n 's/.*"code":\([0-9][0-9]*\).*/\1/p')"
  if [[ -z "$PERIOD_CODE" ]]; then
    echo "Nao foi possivel extrair o periodo criado" >&2
    exit 1
  fi

  GOAL_JSON="$(curl -fsS -X POST "${BASE_URL}/api/sales-goals/create" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d "{\"representative_code\":1,\"period_code\":${PERIOD_CODE},\"analysis_base\":\"SALES\",\"award_pct\":2.5,\"notes\":\"Smoke metas\"}")"
  GOAL_CODE="$(printf '%s' "$GOAL_JSON" | sed -n 's/.*"code":\([0-9][0-9]*\).*/\1/p')"
  if [[ -z "$GOAL_CODE" ]]; then
    echo "Nao foi possivel extrair a meta criada" >&2
    exit 1
  fi

  curl -fsS -X POST "${BASE_URL}/api/sales-goals/items" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d "{\"goal_code\":${GOAL_CODE},\"target_type\":\"ITEM\",\"item_code\":1,\"sales_uom\":\"UN\",\"target_quantity\":10,\"target_value\":10000,\"bonus_pct\":1,\"is_active\":true}" >/dev/null

  GROUP_JSON="$(curl -fsS -X POST "${BASE_URL}/api/sales-goals/group-targets" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d "{\"period_code\":${PERIOD_CODE},\"commercial_group_code\":1,\"goal_type\":\"SALES\",\"minimum_value\":5000,\"minimum_bonus_pct\":0.5,\"probable_value\":8000,\"probable_bonus_pct\":1,\"ideal_value\":10000,\"ideal_bonus_pct\":1.5,\"is_active\":true}")"
  GROUP_ID="$(printf '%s' "$GROUP_JSON" | sed -n 's/.*"id":\([0-9][0-9]*\).*/\1/p')"
  if [[ -z "$GROUP_ID" ]]; then
    echo "Nao foi possivel extrair a meta de grupo criada" >&2
    exit 1
  fi

  curl -fsS -X POST "${BASE_URL}/api/sales-goals/group-customers" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d "{\"group_goal_id\":${GROUP_ID},\"customer_code\":1,\"representative_code\":1,\"minimum_value\":5000,\"minimum_bonus_pct\":0.5,\"probable_value\":8000,\"probable_bonus_pct\":1,\"ideal_value\":10000,\"ideal_bonus_pct\":1.5,\"is_active\":true}" >/dev/null

  curl -fsS -X POST "${BASE_URL}/api/sales-goals/balances" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d "{\"period_code\":${PERIOD_CODE},\"balance_scope\":\"REPRESENTATIVE\",\"representative_code\":1,\"goal_type\":\"SALES\",\"realized_value\":12000,\"ideal_value\":10000,\"balance_value\":2000,\"notes\":\"Excedente smoke\"}" >/dev/null

  curl -fsS "${BASE_URL}/api/sales-goals/${GOAL_CODE}" -H "$AUTH_HEADER" >/dev/null
  curl -fsS "${BASE_URL}/api/sales-goals/report?period_code=${PERIOD_CODE}&analysis_base=SALES" -H "$AUTH_HEADER" >/dev/null
else
  echo "BASE_URL/TOKEN nao definidos; smoke HTTP ignorado."
fi
