#!/usr/bin/env bash
# Cenário de beneficiamento do demo: cliente com dados fiscais, configuração em
# homologação SEM token, componente do cliente na estrutura, duas remessas e duas
# OFs com horas. Não refaz nada se a remessa do cenário já existir, para não
# desfazer o que foi testado na tela.
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:5072}"
DEMO_ADMIN_PASSWORD="${DEMO_ADMIN_PASSWORD:-Demo@12345}"
DB_CONTAINER=panossoerp-postgres-demo
DB_USER=panossoerp_demo
DB_NAME="${DB_NAME:-panossoerpdatabase_demo}"
ADMIN='00000000-0000-0000-0000-000000000001'

psql_demo() {
  docker exec -i "$DB_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1 -q "$@"
}

if [ "$(psql_demo -Atc "SELECT count(*) FROM customer_material_remittances WHERE nfe_number IN (90001, 90002)")" != "0" ]; then
  echo "▸ Cenário de beneficiamento já carregado (NF-e 90001 e 90002)."
  exit 0
fi

psql_demo <<SQL
BEGIN;
INSERT INTO customer_addresses(customer_id, address_type, zip_code, street, number, neighborhood, city, uf, is_default)
SELECT c.id, 'ENTREGA', '01310100', 'AV PAULISTA', '1000', 'BELA VISTA', 'SAO PAULO', 'SP', true
FROM customers c
WHERE c.code = 50 AND NOT EXISTS (SELECT 1 FROM customer_addresses a WHERE a.customer_id = c.id);

UPDATE customers SET state_registration = '110042490114' WHERE code = 50;

INSERT INTO fiscal_configs(cnpj_empresa, razao_social, updated_by, enterprise_id, uf_empresa, focus_nfe_ambiente)
SELECT '12345678000195', 'Panosso Metalurgia Ltda', '$ADMIN', 1, 'SP', 'homologacao'
WHERE NOT EXISTS (SELECT 1 FROM fiscal_configs WHERE enterprise_id = 1);

-- Empresa fictícia com endereço de verdade: sem isso a prévia da nota imprime o
-- provisório do cadastro e a conferência impede a emissão.
UPDATE fiscal_configs SET ie_empresa = '110042490114', logradouro = 'Avenida das Indústrias', numero = '1200',
       bairro = 'Distrito Industrial', municipio = 'São Paulo', codigo_municipio = '3550308', cep = '03178200'
 WHERE enterprise_id = 1 AND logradouro = 'A preencher';

UPDATE item_structures SET customer_supplied = TRUE
WHERE id = (
  SELECT s.id FROM item_structures s
  WHERE s.parent_code = (SELECT item_code FROM sales_order_items WHERE sales_order_code = 1500 ORDER BY code LIMIT 1)
  ORDER BY s.id LIMIT 1
);

WITH pai AS (
  SELECT item_code FROM sales_order_items WHERE sales_order_code = 1500 ORDER BY code LIMIT 1
), ordens AS (
  INSERT INTO production_orders(order_number, item_code, planned_qty, produced_qty, status, created_by, enterprise_id, sales_order_code)
  SELECT v.numero, pai.item_code, 5, 0, 'IN_PROGRESS', '$ADMIN', 1, v.pedido
  FROM pai, (VALUES (900001, 1500::bigint), (900002, NULL::bigint)) v(numero, pedido)
  WHERE NOT EXISTS (SELECT 1 FROM production_orders WHERE order_number = v.numero AND enterprise_id = 1)
  RETURNING id
)
INSERT INTO production_appointments(production_order_id, appointment_date, start_time, end_time, created_by)
SELECT id, CURRENT_DATE, '08:00'::time, '11:30'::time, '$ADMIN' FROM ordens;
COMMIT;
SQL

TOKEN=$(curl -sf -X POST "$BASE_URL/users/login" -H 'Content-Type: application/json' \
  -d "{\"email\":\"admin@panossoerp.demo\",\"password\":\"$DEMO_ADMIN_PASSWORD\",\"enterprise_code\":1}" \
  | sed -E 's/.*"token":"([^"]+)".*/\1/') || {
  echo "Não entrei como admin@panossoerp.demo para receber as remessas: confira DEMO_ADMIN_PASSWORD." >&2
  exit 1
}

HOJE=$(date +%F)
receber() {
  curl -sf -o /dev/null -X POST "$BASE_URL/api/customer-material" \
    -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d "$1"
}

receber '{"customer_code":50,"nfe_number":90001,"nfe_series":"1","nfe_key":"35261000000000000000550010000900011000900011","issue_date":"'"$HOJE"'","sales_order_code":1500,"itens":[{"customer_item_code":"CH-3MM","description":"CHAPA 3MM DO CLIENTE","ncm":"72085100","cst":"050","uom":"PC","qty_invoiced":"10","qty_received":"10","unit_value":"120.50"}]}'
receber '{"customer_code":50,"nfe_number":90002,"nfe_series":"1","issue_date":"'"$HOJE"'","itens":[{"customer_item_code":"BR-20","description":"BARRA 20MM DO CLIENTE","ncm":"72149100","cst":"050","uom":"PC","qty_invoiced":"8","qty_received":"7","unit_value":"45.00","divergence_reason":"faltou 1 peca na conferencia"}]}'

echo "▸ Cenário de beneficiamento carregado: NF-e 90001 e 90002, OFs 900001 e 900002."
