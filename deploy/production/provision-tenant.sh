#!/usr/bin/env bash
#
# Provisiona uma empresa cliente no VentureERP (idempotente).
#
# Cria o role e o database exclusivos do cliente no PostgreSQL da VPS, fecha o
# acesso cruzado entre as bases, aplica as migrations e registra o cliente no
# ciclo de atualização. Rodar de novo com os mesmos argumentos não altera nada.
#
# O isolamento entre empresas é físico: cada cliente tem seu database, seu role e
# seu contêiner de API. Nenhuma consulta pode cruzar empresas por descuido de
# filtro, e o vazamento de uma credencial não alcança a base da outra empresa.
#
# Uso (como root, na VPS):
#
#   sudo ./provision-tenant.sh usimac
#   sudo ./provision-tenant.sh usimac --senha 'uma-senha-forte'
#   sudo ./provision-tenant.sh usimac --conferir        # só relata, não altera
#
# Opções:
#   --senha <valor>      senha do role; sem ela, é gerada com openssl
#   --base <nome>        nome do database   (padrão: venturerp_<cliente>)
#   --role <nome>        nome do role       (padrão: venturerp_<cliente>)
#   --porta <numero>     porta da API       (padrão: 5075)
#   --conferir           não altera nada; apenas relata o estado atual
#   --sem-blindagem      não mexe nas permissões das bases já existentes
#
# Variáveis de ambiente:
#   DATABASE_CONTAINER   contêiner do PostgreSQL     (padrão: venturerp-postgres)
#   SUPERUSER            superusuário do PostgreSQL  (padrão: venturerp_admin)
#   MIGRATIONS_DIR       diretório das migrations
#                        (padrão: /var/lib/venturerp-update/migrations, que é a
#                         fonte de onde a produção foi migrada; cai no ./migrations
#                         do repositório só quando aquela não existe)
set -Eeuo pipefail
umask 077

TENANT=""
PASSWORD=""
DB_NAME=""
DB_ROLE=""
API_PORT="5075"
DRY_RUN=0
HARDEN=1

die() { printf 'provision-tenant: %s\n' "$1" >&2; exit 1; }
info() { printf '  %s\n' "$1"; }

while [[ $# -gt 0 ]]; do
  case "$1" in
    --senha) PASSWORD="${2:?--senha exige um valor}"; shift 2 ;;
    --base) DB_NAME="${2:?--base exige um valor}"; shift 2 ;;
    --role) DB_ROLE="${2:?--role exige um valor}"; shift 2 ;;
    --porta) API_PORT="${2:?--porta exige um valor}"; shift 2 ;;
    --conferir) DRY_RUN=1; shift ;;
    --sem-blindagem) HARDEN=0; shift ;;
    -h|--help) sed -n '2,34p' "$0"; exit 0 ;;
    -*) die "opção desconhecida: $1" ;;
    *) [[ -z "${TENANT}" ]] || die "informe apenas um cliente"; TENANT="$1"; shift ;;
  esac
done

[[ -n "${TENANT}" ]] || die "informe o nome do cliente (ex.: usimac)"
# O nome vira role, database, perfil do compose e prefixo de variável de ambiente.
[[ "${TENANT}" =~ ^[a-z][a-z0-9_]{1,30}$ ]] || die "nome inválido: use minúsculas, dígitos e _ (ex.: usimac)"

DB_NAME="${DB_NAME:-venturerp_${TENANT}}"
DB_ROLE="${DB_ROLE:-venturerp_${TENANT}}"
DATABASE_CONTAINER="${DATABASE_CONTAINER:-venturerp-postgres}"
SUPERUSER="${SUPERUSER:-venturerp_admin}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Fonte das migrations. O padrão é a fila do atualizador, e não o checkout do
# repositório, porque é DELA que a produção é migrada: o self-update copia as
# migrations de dentro da imagem publicada. O checkout na VPS pode estar dezenas de
# releases atrás — o que aconteceu de fato, e faria um cliente novo nascer com
# schema antigo sem ninguém perceber.
MIGRATIONS_FROM_UPDATER="${MIGRATIONS_FROM_UPDATER:-/var/lib/venturerp-update/migrations}"
if [[ -z "${MIGRATIONS_DIR:-}" ]]; then
  if [[ -d "${MIGRATIONS_FROM_UPDATER}" ]]; then
    MIGRATIONS_DIR="${MIGRATIONS_FROM_UPDATER}"
  else
    MIGRATIONS_DIR="${SCRIPT_DIR}/../../migrations"
  fi
fi
ENV_FILE="${ENV_FILE:-/opt/venturerp/panossoerp/.env.${TENANT}}"
UPDATE_ENV="${UPDATE_ENV:-/etc/venturerp/update.env}"
COMPOSE_FILE="${COMPOSE_FILE:-/opt/venturerp/updater/compose.yml}"
# A API e o PostgreSQL ficam no mesmo host, em loopback. Configurável para que o
# script possa ser exercitado contra um PostgreSQL descartável antes da VPS.
DB_HOST="${DB_HOST:-127.0.0.1}"
DB_PORT="${DB_PORT:-5432}"

for cmd in docker openssl; do command -v "${cmd}" >/dev/null || die "faltando: ${cmd}"; done
docker inspect "${DATABASE_CONTAINER}" >/dev/null 2>&1 || die "contêiner ${DATABASE_CONTAINER} não existe"

# psql como superusuário, dentro do contêiner. O postgres do contêiner exige
# senha até em conexão local, então a senha do superusuário vem do ambiente dele.
SUPERUSER_PASSWORD="${SUPERUSER_PASSWORD:-$(docker inspect "${DATABASE_CONTAINER}" \
  --format '{{range .Config.Env}}{{println .}}{{end}}' | sed -nE 's/^POSTGRES_PASSWORD=(.*)$/\1/p' | head -1)}"
[[ -n "${SUPERUSER_PASSWORD}" ]] || die "não encontrei a senha do superusuário; exporte SUPERUSER_PASSWORD"

psql_super() { # $1=database, stdin=SQL
  docker exec -i -e PGPASSWORD="${SUPERUSER_PASSWORD}" "${DATABASE_CONTAINER}" \
    psql -v ON_ERROR_STOP=1 -U "${SUPERUSER}" -d "$1" -q
}
psql_super_value() { # $1=database $2=SQL
  docker exec -e PGPASSWORD="${SUPERUSER_PASSWORD}" "${DATABASE_CONTAINER}" \
    psql -tAX -U "${SUPERUSER}" -d "$1" -c "$2"
}

printf 'provision-tenant: cliente=%s base=%s role=%s porta=%s\n' \
  "${TENANT}" "${DB_NAME}" "${DB_ROLE}" "${API_PORT}"

ROLE_EXISTS="$(psql_super_value postgres "SELECT count(*) FROM pg_roles WHERE rolname = '${DB_ROLE}'")"
DB_EXISTS="$(psql_super_value postgres "SELECT count(*) FROM pg_database WHERE datname = '${DB_NAME}'")"

if [[ "${DRY_RUN}" == "1" ]]; then
  echo "provision-tenant: modo conferência — nada será alterado"
  info "role ${DB_ROLE}: $( [[ "${ROLE_EXISTS}" == "1" ]] && echo existe || echo ausente )"
  info "base ${DB_NAME}: $( [[ "${DB_EXISTS}" == "1" ]] && echo existe || echo ausente )"
  info "env  ${ENV_FILE}: $( [[ -f "${ENV_FILE}" ]] && echo existe || echo ausente )"
  info "registro em ${UPDATE_ENV}: $( grep -q "TENANT_${TENANT^^}_DATABASE_URL" "${UPDATE_ENV}" 2>/dev/null && echo presente || echo ausente )"
  echo
  echo "Bases que o role ${DB_ROLE} conseguiria abrir hoje:"
  psql_super_value postgres "
    SELECT d.datname || ' → ' ||
           CASE WHEN has_database_privilege('${DB_ROLE}', d.datname, 'CONNECT')
                THEN 'PODE CONECTAR' ELSE 'bloqueado' END
    FROM pg_database d
    WHERE d.datistemplate = false AND d.datname <> 'postgres'
    ORDER BY d.datname" 2>/dev/null | sed 's/^/  /' || info "(role ainda não existe)"
  exit 0
fi

[[ "${EUID}" -eq 0 ]] || die "rode como root (sudo)"

# ── 1. Role e database exclusivos ─────────────────────────────────────────────
if [[ "${ROLE_EXISTS}" == "1" ]]; then
  info "role ${DB_ROLE} já existe — preservando a senha atual"
  [[ -n "${PASSWORD}" ]] && {
    psql_super postgres <<SQL
ALTER ROLE ${DB_ROLE} WITH LOGIN PASSWORD '${PASSWORD}';
SQL
    info "senha do role ${DB_ROLE} atualizada"
  }
else
  PASSWORD="${PASSWORD:-$(openssl rand -base64 30 | tr -d '/+=' | head -c 32)}"
  psql_super postgres <<SQL
CREATE ROLE ${DB_ROLE} LOGIN PASSWORD '${PASSWORD}';
SQL
  info "role ${DB_ROLE} criado"
fi

if [[ "${DB_EXISTS}" == "1" ]]; then
  info "base ${DB_NAME} já existe"
else
  # Herda encoding e collation do servidor em vez de fixar pt_BR.UTF-8: a imagem
  # alpine do PostgreSQL não traz esse locale, e fixá-lo faria o CREATE falhar.
  psql_super postgres <<SQL
CREATE DATABASE ${DB_NAME} OWNER ${DB_ROLE} ENCODING 'UTF8' TEMPLATE template0;
SQL
  info "base ${DB_NAME} criada"
fi

# ── 2. Fechar a base nova para qualquer role que não seja o dono ──────────────
# Por padrão o PostgreSQL concede CONNECT a PUBLIC: sem este REVOKE, o role de
# uma empresa abriria a base da outra.
psql_super postgres <<SQL
REVOKE ALL ON DATABASE ${DB_NAME} FROM PUBLIC;
GRANT ALL ON DATABASE ${DB_NAME} TO ${DB_ROLE};
GRANT CONNECT ON DATABASE ${DB_NAME} TO ${SUPERUSER};
SQL
psql_super "${DB_NAME}" <<SQL
REVOKE ALL ON SCHEMA public FROM PUBLIC;
GRANT ALL ON SCHEMA public TO ${DB_ROLE};
GRANT USAGE ON SCHEMA public TO ${SUPERUSER};
SQL
info "acesso à base ${DB_NAME} restrito a ${DB_ROLE}"

# ── 3. Blindar as bases que já existiam ───────────────────────────────────────
# Garante o CONNECT explícito do dono ANTES de remover o de PUBLIC, para não
# trancar fora um ambiente que estava em pé.
# Regra: preservar o status quo. Antes de remover o CONNECT de PUBLIC, conceda
# CONNECT nominal a TODO role que hoje consegue abrir a base — inclusive aos que
# só conseguem por herança de PUBLIC. Critérios mais estreitos (ser dono, ter
# CREATE) deixam de fora justamente o role da aplicação, que tem privilégio de
# objeto e nenhum privilégio de banco, e trancariam um ambiente que estava em pé.
if [[ "${HARDEN}" == "1" ]]; then
  while read -r other_db; do
    [[ -z "${other_db}" || "${other_db}" == "${DB_NAME}" ]] && continue

    incumbents="$(psql_super_value postgres "
      SELECT string_agg(quote_ident(rolname), ',' ORDER BY rolname)
      FROM pg_roles
      WHERE rolcanlogin
        AND rolname NOT LIKE 'pg\_%'
        AND rolname <> '${DB_ROLE}'
        AND has_database_privilege(rolname, '${other_db}', 'CONNECT')")"

    if [[ -n "${incumbents}" ]]; then
      psql_super postgres <<SQL
GRANT CONNECT ON DATABASE ${other_db} TO ${incumbents};
SQL
    fi
    psql_super postgres <<SQL
REVOKE CONNECT ON DATABASE ${other_db} FROM PUBLIC;
SQL

    # Conferência: quem conectava antes precisa continuar conectando. Uma perda
    # aqui é um ambiente fora do ar, então falhe em vez de seguir.
    lost=""
    for role in ${incumbents//,/ }; do
      role="${role//\"/}"
      still="$(psql_super_value postgres "SELECT has_database_privilege('${role}', '${other_db}', 'CONNECT')")"
      [[ "${still}" == "t" ]] || lost="${lost} ${role}"
    done
    [[ -z "${lost}" ]] || die "a blindagem de ${other_db} tiraria o acesso de:${lost}"
    info "base ${other_db} fechada para PUBLIC (mantidos:${incumbents:+ ${incumbents}})"
  done < <(psql_super_value postgres "
    SELECT datname FROM pg_database
    WHERE datistemplate = false AND datname NOT IN ('postgres') ORDER BY datname")
fi

# ── 4. Provar o isolamento, em vez de supor ───────────────────────────────────
echo "provision-tenant: conferindo isolamento"
ISOLATION_FAILURES=0
while read -r other_db; do
  [[ -z "${other_db}" || "${other_db}" == "${DB_NAME}" ]] && continue
  can="$(psql_super_value postgres "SELECT has_database_privilege('${DB_ROLE}', '${other_db}', 'CONNECT')")"
  if [[ "${can}" == "t" ]]; then
    printf '  ✗ %s consegue abrir %s\n' "${DB_ROLE}" "${other_db}" >&2
    ISOLATION_FAILURES=$((ISOLATION_FAILURES + 1))
  else
    info "✓ ${DB_ROLE} não abre ${other_db}"
  fi
done < <(psql_super_value postgres "
  SELECT datname FROM pg_database
  WHERE datistemplate = false AND datname NOT IN ('postgres') ORDER BY datname")
[[ "${ISOLATION_FAILURES}" -eq 0 ]] || die "isolamento incompleto; revise os privilégios antes de seguir"

own="$(psql_super_value postgres "SELECT has_database_privilege('${DB_ROLE}', '${DB_NAME}', 'CONNECT')")"
[[ "${own}" == "t" ]] || die "o role ${DB_ROLE} não consegue abrir a própria base ${DB_NAME}"
info "✓ ${DB_ROLE} abre a própria base ${DB_NAME}"

# ── 5. Migrations ─────────────────────────────────────────────────────────────
#
# Antes de aplicar, compara a fonte com o que a produção já tem. Um cliente novo
# nascendo com schema mais antigo que o das empresas existentes é o tipo de defeito
# que só aparece semanas depois, numa tela que falta coluna.
migracao_mais_alta_no_diretorio() {
  local ultima
  ultima="$(find "${MIGRATIONS_DIR}" -maxdepth 1 -name '*.up.sql' -printf '%f\n' 2>/dev/null \
    | sed -nE 's/^0*([0-9]+)_.*$/\1/p' | sort -n | tail -1)"
  printf '%s' "${ultima:-0}"
}

migracao_aplicada_em() { # $1 = database
  psql_super_value "$1" \
    "SELECT COALESCE(MAX(version),0) FROM schema_migrations" 2>/dev/null | tr -d ' \n' || printf '0'
}

if [[ -d "${MIGRATIONS_DIR}" ]]; then
  fonte="$(migracao_mais_alta_no_diretorio)"
  info "fonte das migrations: ${MIGRATIONS_DIR} (até a ${fonte})"

  # Compara com cada base de empresa que já existe. A mais alta manda.
  referencia=0
  referencia_base=""
  while read -r outra_db; do
    [[ -z "${outra_db}" || "${outra_db}" == "${DB_NAME}" ]] && continue
    aplicada="$(migracao_aplicada_em "${outra_db}")"
    [[ "${aplicada}" =~ ^[0-9]+$ ]] || continue
    if (( aplicada > referencia )); then
      referencia="${aplicada}"
      referencia_base="${outra_db}"
    fi
  done < <(psql_super_value postgres "
    SELECT datname FROM pg_database
    WHERE datistemplate = false AND datname NOT IN ('postgres') ORDER BY datname")

  if (( referencia > 0 )) && (( fonte < referencia )); then
    die "as migrations de ${MIGRATIONS_DIR} vão até a ${fonte}, mas ${referencia_base} já está na ${referencia}.
  Um cliente novo nesta fonte nasceria com schema atrasado. Use a fonte da imagem em produção:
    MIGRATIONS_DIR=${MIGRATIONS_FROM_UPDATER} $0 ${TENANT} ..."
  fi
  if (( referencia > 0 )) && (( fonte > referencia )); then
    info "⚠ a fonte está À FRENTE de ${referencia_base} (${fonte} > ${referencia}); confirme que é o que você quer"
  fi

  echo "provision-tenant: aplicando migrations de ${MIGRATIONS_DIR}"
  if [[ -z "${PASSWORD}" ]]; then
    info "role preexistente sem --senha: pulei as migrations (não sei a senha)"
    info "rode de novo com --senha para aplicá-las"
  else
    # O sufixo ,z faz o relabel de SELinux; sem ele o contêinier não lê o volume
    # em hosts com SELinux aplicado.
    docker run --rm --network host -v "$(cd "${MIGRATIONS_DIR}" && pwd):/migrations:ro,z" \
      migrate/migrate:v4.17.1 -path=/migrations \
      -database="postgres://${DB_ROLE}:${PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable" up
    applied="$(psql_super_value "${DB_NAME}" "SELECT version FROM schema_migrations LIMIT 1")"
    info "migrations aplicadas até a versão ${applied}"
  fi
else
  info "diretório de migrations não encontrado em ${MIGRATIONS_DIR} — pulei"
fi

# ── 6. Arquivo de ambiente da API ─────────────────────────────────────────────
if [[ -f "${ENV_FILE}" ]]; then
  info "env ${ENV_FILE} já existe — preservando"
else
  TEMPLATE="${SCRIPT_DIR}/${TENANT}.env.example"
  [[ -f "${TEMPLATE}" ]] || TEMPLATE="${SCRIPT_DIR}/usimac.env.example"
  [[ -f "${TEMPLATE}" ]] || die "não achei um modelo de env em ${SCRIPT_DIR}"
  install -m 0600 /dev/null "${ENV_FILE}"
  # Cada empresa tem segredo próprio: com o mesmo JWT_SECRET, um token emitido
  # pela API de uma empresa seria aceito pela API da outra.
  sed -e "s#^SERVER_ADDR=.*#SERVER_ADDR=${API_PORT}#" \
      -e "s#^DATABASE_URL=.*#DATABASE_URL=postgres://${DB_ROLE}:${PASSWORD}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable#" \
      -e "s#^JWT_SECRET=.*#JWT_SECRET=$(openssl rand -hex 48)#" \
      -e "s#^METRICS_TOKEN=.*#METRICS_TOKEN=$(openssl rand -hex 32)#" \
      "${TEMPLATE}" >"${ENV_FILE}"
  chmod 600 "${ENV_FILE}"
  info "env ${ENV_FILE} gerado com segredos próprios (0600)"
  # Só o que é valor de verdade conta; o cabeçalho do modelo cita CHANGE_ME em
  # comentário e faria o aviso disparar em toda execução.
  if grep -qE '^[A-Z_]+=.*CHANGE_ME' "${ENV_FILE}"; then
    info "⚠ ainda há CHANGE_ME em ${ENV_FILE}:"
    grep -nE '^[A-Z_]+=.*CHANGE_ME' "${ENV_FILE}" | sed 's/^/      /'
  fi
fi

# ── 7. Registro no ciclo de atualização ───────────────────────────────────────
if [[ -f "${UPDATE_ENV}" ]]; then
  if grep -q "TENANT_${TENANT^^}_DATABASE_URL" "${UPDATE_ENV}"; then
    info "cliente já registrado em ${UPDATE_ENV}"
  else
    db_url="$(grep -E '^DATABASE_URL=' "${ENV_FILE}" | head -1 | cut -d= -f2-)"
    cp -a "${UPDATE_ENV}" "${UPDATE_ENV}.bak-$(date -u +%Y%m%dT%H%M%SZ)"
    if grep -qE '^VENTURERP_TENANTS=' "${UPDATE_ENV}"; then
      sed -i -E "s#^VENTURERP_TENANTS=(.*)#VENTURERP_TENANTS=\1 ${TENANT}#" "${UPDATE_ENV}"
    else
      printf '\nVENTURERP_TENANTS=%s\n' "${TENANT}" >>"${UPDATE_ENV}"
    fi
    cat >>"${UPDATE_ENV}" <<EOF
TENANT_${TENANT^^}_DATABASE_URL=${db_url}
TENANT_${TENANT^^}_API_ENV_FILE=${ENV_FILE}
TENANT_${TENANT^^}_HEALTH_URL=http://127.0.0.1:${API_PORT}/health/ready
EOF
    chmod 600 "${UPDATE_ENV}"
    info "cliente registrado em ${UPDATE_ENV} (backup .bak-* criado)"
  fi
else
  info "⚠ ${UPDATE_ENV} não existe — rode provision-updater.sh antes de atualizar a VPS"
fi

echo
echo "provision-tenant: base pronta. Falta o que depende de DNS e do tráfego:"
echo "  1. registro A do subdomínio apontando para esta VPS"
echo "  2. site do nginx + certificado (deploy/production/nginx/)"
echo "  3. subir a API:"
echo "       cd /opt/venturerp/updater"
echo "       COMPOSE_PROFILES=${TENANT} VENTURERP_IMAGE=<imagem-atual> \\"
echo "         VENTURERP_API_ENV=/opt/venturerp/panossoerp/.env \\"
echo "         VENTURERP_${TENANT^^}_API_ENV=${ENV_FILE} \\"
echo "         docker compose -f ${COMPOSE_FILE} up -d api-${TENANT}"
echo "  4. cadastrar a empresa e os usuários (seed-tenant.sql)"
