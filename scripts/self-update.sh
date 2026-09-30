#!/usr/bin/env bash
set -Eeuo pipefail
umask 077

CONFIG_FILE="${VENTURERP_UPDATE_CONFIG:-/etc/venturerp/update.env}"
[[ -r "${CONFIG_FILE}" ]] || { printf 'update: configuração ausente: %s\n' "${CONFIG_FILE}" >&2; exit 1; }
# shellcheck disable=SC1090
source "${CONFIG_FILE}"

: "${IMAGE_REPOSITORY:?}"
: "${COMPOSE_FILE:?}"
: "${API_ENV_FILE:?}"
: "${UPDATE_DIR:?}"
: "${BACKUP_DIR:?}"
: "${DATABASE_CONTAINER:?}"
: "${DATABASE_USER:?}"
: "${DATABASE_NAME:?}"
: "${DATABASE_URL:?}"

# O container postgres exige senha até em conexões locais, então todo
# pg_dump/pg_restore/psql via docker exec precisa carregar PGPASSWORD. Deriva do
# DATABASE_URL (fonte única) quando não informado explicitamente.
DATABASE_PASSWORD="${DATABASE_PASSWORD:-$(printf '%s' "${DATABASE_URL}" | sed -nE 's#^[a-z0-9+]+://[^:]+:([^@]+)@.*#\1#p')}"

HEALTH_URL="${HEALTH_URL:-http://127.0.0.1:5070/health/ready}"
# UID/GID do appuser dentro do contêiner da API. A fila é a única ponte
# API↔host: a API (não-root) precisa ler/escrever aqui, então root normaliza a
# posse dos arquivos que produz para esse UID.
UPDATE_UID="${UPDATE_UID:-10001}"
UPDATE_GID="${UPDATE_GID:-10001}"
LEGACY_SERVICE="${LEGACY_SERVICE:-venturerp.service}"
HEALTH_ATTEMPTS="${HEALTH_ATTEMPTS:-30}"
HEALTH_INTERVAL_SECONDS="${HEALTH_INTERVAL_SECONDS:-2}"
REQUEST_FILE="${UPDATE_DIR}/request.json"
STATUS_FILE="${UPDATE_DIR}/status.json"
ACTIVE_LOCK="${UPDATE_DIR}/active.lock"
STATE_FILE="${UPDATE_DIR}/deployed-image"
LOCK_FILE="${LOCK_FILE:-/run/lock/venturerp-update.lock}"
MIGRATIONS_DIR="${UPDATE_DIR}/migrations"

# ── Tenants ───────────────────────────────────────────────────────────────────
#
# Cada empresa cliente é um contêiner de API sobre seu próprio database. Backup,
# migration, health-check e rollback percorrem a lista abaixo, então um cliente
# novo entra no ciclo de atualização apenas por configuração.
#
# O índice 0 é sempre o tenant principal, descrito por DATABASE_URL /
# API_ENV_FILE / HEALTH_URL. Os demais vêm de:
#
#   VENTURERP_TENANTS="usimac treinamento"
#   TENANT_USIMAC_DATABASE_URL=postgres://venturerp_usimac:...@127.0.0.1:5432/venturerp_usimac?sslmode=disable
#   TENANT_USIMAC_API_ENV_FILE=/opt/venturerp/panossoerp/.env.usimac
#   TENANT_USIMAC_HEALTH_URL=http://127.0.0.1:5075/health/ready
#   TENANT_USIMAC_PROFILE=usimac                     # opcional; padrão: o nome
#   TENANT_USIMAC_COMPOSE_ENV_VAR=VENTURERP_USIMAC_API_ENV   # opcional
#
# Usuário, base e senha são derivados do DATABASE_URL quando não informados em
# TENANT_<NOME>_DATABASE_{USER,NAME,PASSWORD}.
TENANT_NAMES=()
TENANT_DB_URLS=()
TENANT_ENV_FILES=()
TENANT_HEALTH_URLS=()
TENANT_PROFILES=()
TENANT_COMPOSE_VARS=()
TENANT_DB_USERS=()
TENANT_DB_NAMES=()
TENANT_DB_PASSWORDS=()
TENANT_BACKUPS=()

url_part() { # $1=url $2=user|password|database
  case "$2" in
    user)     printf '%s' "$1" | sed -nE 's#^[a-z0-9+]+://([^:@/]+).*#\1#p' ;;
    password) printf '%s' "$1" | sed -nE 's#^[a-z0-9+]+://[^:]+:([^@]+)@.*#\1#p' ;;
    database) printf '%s' "$1" | sed -nE 's#^[a-z0-9+]+://[^/]+/([^?]+).*$#\1#p' ;;
  esac
}

# register_tenant <nome> <database_url> <api_env_file> <health_url> <profile> \
#                 <compose_env_var> [db_user] [db_name] [db_password] [exige_senha]
#
# O tenant principal aceita senha vazia: instalações que autenticam por peer/trust
# ou .pgpass não a carregam no DATABASE_URL. Todo tenant secundário exige senha,
# porque ele só existe a partir de um role dedicado criado com senha.
register_tenant() {
  local name="$1" url="$2" env_file="$3" health="$4" profile="$5" compose_var="$6"
  local db_user="${7:-}" db_name="${8:-}" db_password="${9:-}" require_password="${10:-1}"

  [[ -n "${url}" ]]      || { printf 'update: tenant %s sem DATABASE_URL\n' "${name}" >&2; return 1; }
  [[ -n "${env_file}" ]] || { printf 'update: tenant %s sem API_ENV_FILE\n' "${name}" >&2; return 1; }

  db_user="${db_user:-$(url_part "${url}" user)}"
  db_name="${db_name:-$(url_part "${url}" database)}"
  db_password="${db_password:-$(url_part "${url}" password)}"
  [[ -n "${db_user}" && -n "${db_name}" ]] || {
    printf 'update: não consegui derivar usuário/base do tenant %s\n' "${name}" >&2; return 1; }
  [[ "${require_password}" != "1" || -n "${db_password}" ]] || {
    printf 'update: tenant %s sem senha no DATABASE_URL\n' "${name}" >&2; return 1; }

  TENANT_NAMES+=("${name}")
  TENANT_DB_URLS+=("${url}")
  TENANT_ENV_FILES+=("${env_file}")
  TENANT_HEALTH_URLS+=("${health}")
  TENANT_PROFILES+=("${profile}")
  TENANT_COMPOSE_VARS+=("${compose_var}")
  TENANT_DB_USERS+=("${db_user}")
  TENANT_DB_NAMES+=("${db_name}")
  TENANT_DB_PASSWORDS+=("${db_password}")
}

STAMP="$(date -u +%Y%m%dT%H%M%SZ)"

register_tenant principal "${DATABASE_URL}" "${API_ENV_FILE}" "${HEALTH_URL}" "" \
  VENTURERP_API_ENV "${DATABASE_USER}" "${DATABASE_NAME}" "${DATABASE_PASSWORD}" 0

# Compatibilidade: instalações anteriores descrevem o ambiente de treinamento em
# TRAINING_*. Continua valendo, desde que não esteja também em VENTURERP_TENANTS.
if [[ -n "${TRAINING_DATABASE_URL:-}" ]] && [[ " ${VENTURERP_TENANTS:-} " != *" treinamento "* ]] && [[ " ${VENTURERP_TENANTS:-} " != *" training "* ]]; then
  : "${TRAINING_API_ENV_FILE:?TRAINING_API_ENV_FILE is required when TRAINING_DATABASE_URL is set}"
  register_tenant training "${TRAINING_DATABASE_URL}" "${TRAINING_API_ENV_FILE}" \
    "${TRAINING_HEALTH_URL:-http://127.0.0.1:5071/health/ready}" training VENTURERP_TRAINING_API_ENV \
    "${TRAINING_DATABASE_USER:-}" "${TRAINING_DATABASE_NAME:-}" "${TRAINING_DATABASE_PASSWORD:-}"
fi

for tenant in ${VENTURERP_TENANTS:-}; do
  upper="$(printf '%s' "${tenant}" | tr '[:lower:]-' '[:upper:]_')"
  url_var="TENANT_${upper}_DATABASE_URL"
  env_var="TENANT_${upper}_API_ENV_FILE"
  health_var="TENANT_${upper}_HEALTH_URL"
  profile_var="TENANT_${upper}_PROFILE"
  compose_var_var="TENANT_${upper}_COMPOSE_ENV_VAR"
  user_var="TENANT_${upper}_DATABASE_USER"
  name_var="TENANT_${upper}_DATABASE_NAME"
  pass_var="TENANT_${upper}_DATABASE_PASSWORD"
  register_tenant "${tenant}" "${!url_var:-}" "${!env_var:-}" "${!health_var:-}" \
    "${!profile_var:-${tenant}}" "${!compose_var_var:-VENTURERP_${upper}_API_ENV}" \
    "${!user_var:-}" "${!name_var:-}" "${!pass_var:-}"
done

# Um tenant secundário nunca pode apontar para a base de outro: um erro de
# copiar-e-colar no update.env faria o rollback restaurar o backup errado sobre
# dados de produção. Falha antes de tocar em qualquer coisa.
for ((i = 0; i < ${#TENANT_NAMES[@]}; i++)); do
  for ((j = i + 1; j < ${#TENANT_NAMES[@]}; j++)); do
    [[ "${TENANT_DB_NAMES[i]}" == "${TENANT_DB_NAMES[j]}" ]] && {
      printf 'update: tenants %s e %s apontam para a mesma base (%s)\n' \
        "${TENANT_NAMES[i]}" "${TENANT_NAMES[j]}" "${TENANT_DB_NAMES[i]}" >&2
      exit 1
    }
  done
  TENANT_BACKUPS+=("")
done

COMPOSE_PROFILES=""
for profile in "${TENANT_PROFILES[@]}"; do
  [[ -z "${profile}" ]] && continue
  COMPOSE_PROFILES="${COMPOSE_PROFILES:+${COMPOSE_PROFILES},}${profile}"
done

mkdir -p "${UPDATE_DIR}" "${BACKUP_DIR}" "$(dirname "${LOCK_FILE}")"
# A API (UID do appuser) escreve request.json/status.json aqui; garanta a posse.
chown "${UPDATE_UID}:${UPDATE_GID}" "${UPDATE_DIR}" 2>/dev/null || true
exec 9>"${LOCK_FILE}"
flock -n 9 || exit 0
[[ -f "${REQUEST_FILE}" ]] || exit 0

VERSION="$(jq -er '.version' "${REQUEST_FILE}")"
[[ "${VERSION}" =~ ^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?$ ]] || {
  rm -f "${REQUEST_FILE}" "${ACTIVE_LOCK}"
  exit 2
}
TARGET_IMAGE="${IMAGE_REPOSITORY}:v${VERSION}"
PREVIOUS_IMAGE=""
[[ -f "${STATE_FILE}" ]] && PREVIOUS_IMAGE="$(<"${STATE_FILE}")"
LEGACY_WAS_ACTIVE=0
systemctl is-active --quiet "${LEGACY_SERVICE}" && LEGACY_WAS_ACTIVE=1
REQUESTED_AT="$(jq -er '.requested_at' "${REQUEST_FILE}")"
STARTED_AT="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
for ((i = 0; i < ${#TENANT_NAMES[@]}; i++)); do
  if [[ "${i}" -eq 0 ]]; then
    TENANT_BACKUPS[i]="${BACKUP_DIR}/pre-update-${VERSION}-${STAMP}.dump"
  else
    TENANT_BACKUPS[i]="${BACKUP_DIR}/pre-update-${TENANT_NAMES[i]}-${VERSION}-${STAMP}.dump"
  fi
done
SUCCESS=0

# Toda invocação do compose precisa exportar o arquivo de env de cada tenant,
# porque o compose.yml referencia um por serviço.
compose() {
  local -a env_assignments=("COMPOSE_PROFILES=${COMPOSE_PROFILES}")
  local i
  for ((i = 0; i < ${#TENANT_NAMES[@]}; i++)); do
    env_assignments+=("${TENANT_COMPOSE_VARS[i]}=${TENANT_ENV_FILES[i]}")
  done
  env "${env_assignments[@]}" VENTURERP_IMAGE="$1" docker compose -f "${COMPOSE_FILE}" "${@:2}"
}

status() {
  local state="$1" progress="$2" message="$3" finished="${4:-}"
  local tmp
  tmp="$(mktemp "${UPDATE_DIR}/.status.XXXXXX")"
  jq -n \
    --arg state "${state}" --arg target "${VERSION}" --arg message "${message}" \
    --arg requested "${REQUESTED_AT}" --arg started "${STARTED_AT}" --arg finished "${finished}" \
    --argjson progress "${progress}" \
    '{state:$state,target_version:$target,progress:$progress,message:$message,requested_at:$requested,started_at:$started}
     + (if $finished == "" then {} else {finished_at:$finished} end)' >"${tmp}"
  chmod 600 "${tmp}"
  mv "${tmp}" "${STATUS_FILE}"
  # A API (não-root) lê o status pelo endpoint; deixe-a como dona do arquivo.
  chown "${UPDATE_UID}:${UPDATE_GID}" "${STATUS_FILE}" 2>/dev/null || true
}

backup_tenant() { # índice
  local i="$1"
  docker exec -e PGPASSWORD="${TENANT_DB_PASSWORDS[i]}" "${DATABASE_CONTAINER}" \
    pg_dump -U "${TENANT_DB_USERS[i]}" -d "${TENANT_DB_NAMES[i]}" \
    --format=custom --no-owner --no-acl >"${TENANT_BACKUPS[i]}"
  [[ -s "${TENANT_BACKUPS[i]}" ]]
  docker exec -i "${DATABASE_CONTAINER}" pg_restore --list <"${TENANT_BACKUPS[i]}" >/dev/null
}

restore_tenant() { # índice
  local i="$1"
  [[ -s "${TENANT_BACKUPS[i]}" ]] || return 1
  docker exec -e PGPASSWORD="${TENANT_DB_PASSWORDS[i]}" "${DATABASE_CONTAINER}" \
    psql -U "${TENANT_DB_USERS[i]}" -d postgres -v ON_ERROR_STOP=1 \
    -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname='${TENANT_DB_NAMES[i]}' AND pid <> pg_backend_pid();" >/dev/null
  docker exec -i -e PGPASSWORD="${TENANT_DB_PASSWORDS[i]}" "${DATABASE_CONTAINER}" \
    pg_restore -U "${TENANT_DB_USERS[i]}" -d "${TENANT_DB_NAMES[i]}" \
    --clean --if-exists --no-owner --no-acl <"${TENANT_BACKUPS[i]}"
}

await_health() { # url
  local url="$1" attempt
  [[ -n "${url}" ]] || return 0
  for ((attempt = 1; attempt <= HEALTH_ATTEMPTS; attempt++)); do
    if curl --fail --silent --show-error "${url}" >/dev/null; then return 0; fi
    sleep "${HEALTH_INTERVAL_SECONDS}"
  done
  return 1
}

rollback() {
  local exit_code="$?"
  trap - ERR
  [[ "${SUCCESS}" == "1" ]] && return 0
  status failed 90 "Falha na implantação; restaurando banco e versão anterior"
  compose "${TARGET_IMAGE}" down --remove-orphans >/dev/null 2>&1 || true
  local i
  for ((i = 0; i < ${#TENANT_NAMES[@]}; i++)); do
    restore_tenant "${i}" || true
  done
  if [[ -n "${PREVIOUS_IMAGE}" ]]; then
    compose "${PREVIOUS_IMAGE}" up -d
  elif [[ "${LEGACY_WAS_ACTIVE}" == "1" ]]; then
    systemctl start "${LEGACY_SERVICE}"
  fi
  status rolled_back 100 "Atualização falhou; versão anterior e backup foram restaurados" "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  rm -f "${REQUEST_FILE}" "${ACTIVE_LOCK}"
  exit "${exit_code}"
}
trap rollback ERR INT TERM

status running 5 "Validando pré-requisitos"
for command in docker jq curl flock; do command -v "${command}" >/dev/null; done
docker inspect "${DATABASE_CONTAINER}" >/dev/null

status running 15 "Criando e verificando backup transacional de ${#TENANT_NAMES[@]} base(s)"
for ((i = 0; i < ${#TENANT_NAMES[@]}; i++)); do
  backup_tenant "${i}"
done

status running 30 "Baixando imagem assinada pelo pipeline"
docker pull "${TARGET_IMAGE}"
rm -rf "${MIGRATIONS_DIR}"
container_id="$(docker create "${TARGET_IMAGE}")"
trap 'docker rm -f "${container_id}" >/dev/null 2>&1 || true; rollback' ERR INT TERM
docker cp "${container_id}:/app/migrations" "${MIGRATIONS_DIR}"
docker rm "${container_id}" >/dev/null
trap rollback ERR INT TERM

status running 45 "Parando as APIs e aplicando migrations"
systemctl stop "${LEGACY_SERVICE}" >/dev/null 2>&1 || true
if [[ -n "${PREVIOUS_IMAGE}" ]]; then
  compose "${PREVIOUS_IMAGE}" down --remove-orphans
fi
for ((i = 0; i < ${#TENANT_NAMES[@]}; i++)); do
  docker run --rm --network host -v "${MIGRATIONS_DIR}:/migrations:ro" migrate/migrate:v4.17.1 \
    -path=/migrations -database="${TENANT_DB_URLS[i]}" up
done

status running 70 "Iniciando a nova versão"
compose "${TARGET_IMAGE}" up -d

status running 85 "Executando health-check de prontidão"
for ((i = 0; i < ${#TENANT_NAMES[@]}; i++)); do
  await_health "${TENANT_HEALTH_URLS[i]}"
done

printf '%s\n' "${TARGET_IMAGE}" >"${STATE_FILE}"
SUCCESS=1
trap - ERR INT TERM
status succeeded 100 "VentureERP atualizado com sucesso para v${VERSION}" "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
rm -f "${REQUEST_FILE}" "${ACTIVE_LOCK}"
find "${BACKUP_DIR}" -type f -name 'pre-update-*.dump' -mtime +30 -delete
