#!/usr/bin/env bash
# Põe no ambiente demo o código DESTA pasta, inclusive o que ainda não foi
# commitado: reconstrói a API, aplica as migrações pendentes (com backup antes)
# e carrega os cenários de beneficiamento e de cadeia de fabricação. Sem release e sem perder os dados.
# Rodar de novo sem mudança nenhuma é rápido e não altera nada.
set -euo pipefail

cd "$(dirname "$0")/.."

# O projeto é compartilhado com a stack local: sem isto, cada comando lista os outros containers.
export COMPOSE_IGNORE_ORPHANS=True
# A atestação de build traz data e muda a imagem a cada build: sem isto, rodar sem
# mudança nenhuma reiniciaria a API.
export BUILDX_NO_DEFAULT_ATTESTATIONS=1
COMPOSE=(docker compose -f docker-compose.demo.yml)
DB_CONTAINER=panossoerp-postgres-demo
API_CONTAINER=panossoerp-api-demo
DB_USER=panossoerp_demo
DB_NAME=panossoerpdatabase_demo

env_do_container() {
  docker inspect -f '{{range .Config.Env}}{{println .}}{{end}}' "$1" 2>/dev/null | sed -n "s/^$2=//p" | head -1
}

# Os segredos são os do demo que já existe: outro valor trancaria o banco.
DEMO_DB_PASSWORD="${DEMO_DB_PASSWORD:-$(env_do_container "$DB_CONTAINER" POSTGRES_PASSWORD)}"
DEMO_JWT_SECRET="${DEMO_JWT_SECRET:-$(env_do_container "$API_CONTAINER" JWT_SECRET)}"
if [ -z "$DEMO_DB_PASSWORD" ] || [ -z "$DEMO_JWT_SECRET" ]; then
  echo "Demo ainda não existe nesta máquina: defina DEMO_DB_PASSWORD, DEMO_JWT_SECRET e DEMO_ADMIN_PASSWORD e rode make demo-bootstrap." >&2
  exit 1
fi
export DEMO_DB_PASSWORD DEMO_JWT_SECRET

psql_demo() {
  docker exec -i "$DB_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1 "$@"
}

aguardar_saude() {
  local container=$1 tentativas=$2
  until [ "$(docker inspect -f '{{.State.Health.Status}}' "$container" 2>/dev/null)" = "healthy" ]; do
    tentativas=$((tentativas - 1))
    if [ "$tentativas" -le 0 ]; then
      echo "O container $container não ficou saudável. Veja: docker logs $container" >&2
      exit 1
    fi
    sleep 2
  done
}

echo "▸ Código: $(pwd) ($(git branch --show-current), $(git rev-parse --short HEAD), $(git status --short | wc -l) arquivo(s) sem commit)"

"${COMPOSE[@]}" up -d postgres-demo
aguardar_saude "$DB_CONTAINER" 30

ultima_migracao=$(find migrations -name '*.up.sql' -printf '%f\n' | sed -E 's/^0*([0-9]+)_.*/\1/' | sort -n | tail -1)
versao_e_dirty=$(psql_demo -Atc "SELECT version, dirty FROM schema_migrations" 2>/dev/null || true)
versao_banco=${versao_e_dirty%%|*}
versao_banco=${versao_banco:-0}

if [ "${versao_e_dirty##*|}" = "t" ]; then
  echo "A migração $versao_banco do demo falhou pela metade. Restaure o backup de backups/demo/ ou rode make demo-reset." >&2
  exit 1
fi
if [ "$versao_banco" -gt "$ultima_migracao" ]; then
  echo "O demo está na migração $versao_banco e esta pasta só vai até a $ultima_migracao: o código daqui é mais antigo que o do demo." >&2
  echo "Rode a partir da pasta com o código mais novo. Nada foi alterado." >&2
  exit 1
fi

if [ "$versao_banco" -lt "$ultima_migracao" ] && [ "$versao_banco" -gt 0 ]; then
  mkdir -p backups/demo
  backup="backups/demo/antes-da-$ultima_migracao-$(date +%Y%m%d-%H%M%S).dump"
  docker exec "$DB_CONTAINER" pg_dump -U "$DB_USER" -d "$DB_NAME" -Fc >"$backup"
  echo "▸ Backup do demo (migração $versao_banco): $backup"
fi

# Se o build falhar, a API anterior continua no ar e o banco não é migrado.
"${COMPOSE[@]}" build api-demo
"${COMPOSE[@]}" up -d api-demo
aguardar_saude "$API_CONTAINER" 60

# Demo carregado antes da correção do seed-demo.sql: acabado como "de terceiro"
# (o MRP não gera ordem para ele), matéria-prima como "fabricado" e serviço com
# o código 4, que não existe. Reclassifica pelo papel do item; não muda nada
# na segunda vez.
psql_demo -q -c "UPDATE items SET engineering_type = CASE
    WHEN engineering_type_struct = 0 AND supplies_type_of_use = 1 THEN 0
    WHEN supplies_type_of_use = 3 THEN 3
    ELSE 1 END
  WHERE (engineering_type = 2 AND engineering_type_struct = 0 AND supplies_type_of_use = 1)
     OR (engineering_type = 0 AND engineering_type_struct = 1 AND supplies_type_of_use = 0)
     OR (engineering_type = 4 AND supplies_type_of_use = 3)"
# Tipo de MRP 2 não existe (0 normal, 1 projeto) e o MRP o lia como projeto:
# a matéria-prima nunca ganhava sugestão de compra.
psql_demo -q -c "UPDATE items SET planning_type_mrp = 0 WHERE planning_type_mrp NOT IN (0, 1)"
# Mesma origem: a estrutura dos acabados saiu toda em UN (chapa em KG, tubo em
# M ficavam sem conversão e o MRP não comprava), o roteiro guardou tempos de
# teste sem unidade (300 "horas" por peça) e um teste deixou o roteiro do 10002
# inativo. Corrige só o que está nesse estado.
psql_demo -q -c "BEGIN;
UPDATE item_structures s SET unit_of_measurement = c.warehouse_unit_of_measurement, conversion_factor = NULL
  FROM items c, items p
 WHERE c.code = s.child_code AND p.code = s.parent_code AND p.engineering_type_struct = 0
   AND s.unit_of_measurement = 'UN' AND c.warehouse_unit_of_measurement IN ('KG','M')
   AND s.child_code BETWEEN 20001 AND 20080;
UPDATE route_operations SET standard_time = NULL, setup_time = NULL, run_time = NULL, labor_time = NULL
 WHERE time_unit IS NULL AND (standard_time >= 100 OR setup_time >= 100 OR run_time >= 100);
UPDATE manufacturing_routes r SET is_active = TRUE
 WHERE r.is_standard AND r.situation = 'APROVADA' AND NOT r.is_active
   AND NOT EXISTS (SELECT 1 FROM manufacturing_routes o WHERE o.item_code = r.item_code AND o.is_active);
-- Centro padrão das operações do demo (o tipo de máquina de mesmo número),
-- como no seed: sem ele a etapa fica sem centro e sem máquina.
UPDATE operations SET default_work_center_id = id WHERE id BETWEEN 1 AND 5 AND default_work_center_id IS NULL;
-- Roteiro de acabado sem nenhuma etapa (demo antigo): sem etapas não há
-- prazo, carga, máquina nem OF. Mesmas etapas do seed-demo.sql.
INSERT INTO route_operations (route_id, sequence, operation_id, situation, is_active, created_at, updated_at)
SELECT r.id, e.seq, e.op, 'APROVADA', TRUE, now(), now()
FROM manufacturing_routes r
CROSS JOIN LATERAL (VALUES (10, 1), (20, 2 + (r.id % 3)::int), (30, 5)) e(seq, op)
WHERE r.is_active AND r.item_code BETWEEN 10001 AND 10050
  AND NOT EXISTS (SELECT 1 FROM route_operations x WHERE x.route_id = r.id);
-- Produtividade por máquina apagada por limpeza antiga: sem ela o MRP não
-- escolhe máquina nem ocupa a agenda. Mesma regra do seed-demo.sql, só para o
-- item com roteiro que ficou sem nenhuma.
INSERT INTO item_machine_times (item_code, mask, production_time, setup_time, priority, machine_code,
  production_time_unit, production_base_qty, efficiency_rate, enterprise_id)
SELECT r.item_code, '', round(o.standard_time * 60, 2), round(o.setup_time * 60, 2),
  row_number() OVER (PARTITION BY r.item_code, m.machine_type_code ORDER BY m.code),
  m.code, 'MINUTO', 1, m.efficiency_rate, m.enterprise_id
FROM manufacturing_routes r
JOIN route_operations ro ON ro.route_id = r.id
JOIN operations o ON o.id = ro.operation_id
JOIN machines m ON m.machine_type_code = o.default_work_center_id
WHERE r.is_active AND r.item_code BETWEEN 10001 AND 10050
  AND NOT EXISTS (SELECT 1 FROM item_machine_times t WHERE t.item_code = r.item_code)
ON CONFLICT DO NOTHING;
-- Produtividade sem empresa (o seed antigo não gravava): a leitura é por
-- empresa e a linha ficava invisível. A empresa é a da máquina.
UPDATE item_machine_times t SET enterprise_id = m.enterprise_id FROM machines m
 WHERE t.enterprise_id IS NULL AND m.code = t.machine_code;
-- Versão de estrutura de teste: obsoleta, nunca aprovada e sem linha. Deixava
-- o item sob controle de versão sem versão vigente e travava a OF.
DELETE FROM bom_headers h WHERE h.status = 'OBSOLETE' AND h.approved_at IS NULL
   AND NOT EXISTS (SELECT 1 FROM bom_header_lines l WHERE l.bom_header_id = h.id);
COMMIT;"
bash scripts/seed-demo-beneficiamento.sh
bash scripts/seed-demo-cadeia.sh
bash scripts/seed-demo-perfis.sh
# Só completa o que falta: novos cadastros de apoio chegam sem recriar o demo.
psql_demo -q < scripts/seed-demo-cadastros-fiscais.sql
# Os 50 clientes do demo têm 2 raízes de CNPJ (um só grupo econômico) e títulos
# vencidos há mais de um ano: a revisão automática de crédito zeraria todos em
# até 1 hora. Desligada aqui; para demonstrá-la, ligue na política de crédito.
psql_demo -q -c "INSERT INTO customer_credit_policies (enterprise_id, auto_review_months)
  SELECT id, 0 FROM enterprise ON CONFLICT (enterprise_id) DO UPDATE SET auto_review_months = 0"

echo "▸ Demo atualizado: migração $(psql_demo -Atc 'SELECT version FROM schema_migrations'), API em http://localhost:5072"
