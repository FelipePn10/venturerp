#!/usr/bin/env bash
# Perfis de acesso do demo: além do admin, um OPERADOR de posto
# (operador@panossoerp.demo / Operador@12345), para mostrar e testar a
# VPRO0900 como o chão de fábrica a vê (executa etapa e aponta; não cria,
# cancela, entrega nem vê custo). Cadastra pela API, como o admin faria na
# tela; só ajusta o perfil no banco. Não refaz se o usuário já existir.
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:5072}"
DEMO_ADMIN_EMAIL="${DEMO_ADMIN_EMAIL:-admin@panossoerp.demo}"
DEMO_ADMIN_PASSWORD="${DEMO_ADMIN_PASSWORD:-Demo@12345}"
DB_CONTAINER=panossoerp-postgres-demo
DB_USER=panossoerp_demo
DB_NAME="${DB_NAME:-panossoerpdatabase_demo}"
OPERADOR=operador@panossoerp.demo

psql_demo() { docker exec -i "$DB_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1 -qtAX "$@"; }

if [ "$(psql_demo -c "SELECT count(*) FROM users WHERE email = '$OPERADOR'")" = "0" ]; then
  token=$(curl -sf -X POST "$BASE_URL/users/login" -H 'Content-Type: application/json' \
    -d "{\"email\":\"$DEMO_ADMIN_EMAIL\",\"password\":\"$DEMO_ADMIN_PASSWORD\"}" | python3 -c 'import json,sys; d=json.load(sys.stdin); print(d.get("token") or d.get("data",{}).get("token",""))')
  [ -n "$token" ] || { echo "falha no login do admin do demo" >&2; exit 1; }
  empresa=$(psql_demo -c "SELECT code FROM enterprise ORDER BY id LIMIT 1")
  curl -sf -o /dev/null -X POST "$BASE_URL/users/register" -H 'Content-Type: application/json' -H "Authorization: Bearer $token" \
    -d "{\"name\":\"Operador do Posto\",\"email\":\"$OPERADOR\",\"password\":\"Operador@12345\",\"enterprise_code\":$empresa}" \
    || { echo "falha ao cadastrar o operador do demo" >&2; exit 1; }
fi
# O perfil vale por empresa (user_enterprises.role); é dele que o login lê.
psql_demo -c "UPDATE users SET role = 'OPERATOR' WHERE email = '$OPERADOR' AND role IS DISTINCT FROM 'OPERATOR';
  UPDATE user_enterprises ue SET role = 'OPERATOR' FROM users u WHERE u.id = ue.user_id AND u.email = '$OPERADOR' AND ue.role IS DISTINCT FROM 'OPERATOR'"
echo "▸ Perfil de operador do demo: $OPERADOR (OPERATOR)."
