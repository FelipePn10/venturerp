#!/usr/bin/env bash
# Põe no ambiente demo o código DESTA pasta, inclusive o que ainda não foi
# commitado: reconstrói a API, aplica as migrações pendentes (com backup antes)
# e carrega o cenário de beneficiamento. Sem release e sem perder os dados.
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

bash scripts/seed-demo-beneficiamento.sh
# Só completa o que falta: novos cadastros de apoio chegam sem recriar o demo.
psql_demo -q < scripts/seed-demo-cadastros-fiscais.sql

echo "▸ Demo atualizado: migração $(psql_demo -Atc 'SELECT version FROM schema_migrations'), API em http://localhost:5072"
