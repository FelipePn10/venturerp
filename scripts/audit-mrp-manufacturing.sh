#!/usr/bin/env bash
set -euo pipefail

# Auditoria transversal das cinco fases de planejamento de materiais/manufatura.
# Requer: TEST_DATABASE_URL apontando exclusivamente para PostgreSQL de testes,
# migrate, psql, Go e Docker/PostgreSQL já iniciado.

: "${TEST_DATABASE_URL:?Informe TEST_DATABASE_URL para o PostgreSQL de testes}"
export GOCACHE="${GOCACHE:-/tmp/panossoerp-go-build}"

version="$(migrate -path migrations -database "$TEST_DATABASE_URL" version 2>&1)"
if [[ "$version" != "230" ]]; then
  echo "Versão de schema inesperada: $version (esperada: 230)" >&2
  exit 1
fi

echo "=== Reversibilidade das migrations finais ==="
migrate -path migrations -database "$TEST_DATABASE_URL" down 2
migrate -path migrations -database "$TEST_DATABASE_URL" up 2

echo "=== Invariantes de schema e tenant ==="
if command -v psql >/dev/null 2>&1; then
  psql_command=(psql "$TEST_DATABASE_URL")
else
  psql_command=(docker exec "${TEST_POSTGRES_CONTAINER:-panossoerp-postgres-test}" psql
    -U "${TEST_POSTGRES_USER:-panossoerp_test}"
    -d "${TEST_POSTGRES_DB:-panossoerpdatabase_test}")
fi
"${psql_command[@]}" -v ON_ERROR_STOP=1 -Atc "
DO \$\$
DECLARE missing text;
BEGIN
  SELECT string_agg(name, ', ') INTO missing
  FROM (VALUES
    ('production_orders','enterprise_id'),
    ('production_order_materials','enterprise_id'),
    ('production_order_lot_allocations','enterprise_id'),
    ('production_order_scrap_destinations','enterprise_id'),
    ('purchase_order_currency_rates','enterprise_id'),
    ('drawings','enterprise_id'),
    ('mrp_profile_details','enterprise_id')
  ) expected(name,column_name)
  WHERE NOT EXISTS (
    SELECT 1 FROM information_schema.columns c
    WHERE c.table_schema='public' AND c.table_name=expected.name
      AND c.column_name=expected.column_name
  );
  IF missing IS NOT NULL THEN
    RAISE EXCEPTION 'colunas tenant ausentes: %', missing;
  END IF;
  IF EXISTS (
    SELECT 1 FROM production_orders
    GROUP BY enterprise_id,order_number HAVING COUNT(*) > 1
  ) THEN
    RAISE EXCEPTION 'números de OF duplicados por empresa';
  END IF;
END \$\$;
SELECT 'schema-enterprise-ok';"

echo "=== Aceite funcional focado ==="
scripts/test-mrp-manufacturing.sh

echo "=== Segurança e isolamento ==="
scripts/test-mrp-manufacturing-security.sh

echo "=== Regressão e CI globais ==="
make test
make ci
git diff --check

echo "=== Auditoria transversal concluída ==="
