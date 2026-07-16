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
LEGACY_SERVICE="${LEGACY_SERVICE:-venturerp.service}"
HEALTH_ATTEMPTS="${HEALTH_ATTEMPTS:-30}"
HEALTH_INTERVAL_SECONDS="${HEALTH_INTERVAL_SECONDS:-2}"
REQUEST_FILE="${UPDATE_DIR}/request.json"
STATUS_FILE="${UPDATE_DIR}/status.json"
ACTIVE_LOCK="${UPDATE_DIR}/active.lock"
STATE_FILE="${UPDATE_DIR}/deployed-image"
LOCK_FILE="${LOCK_FILE:-/run/lock/venturerp-update.lock}"
MIGRATIONS_DIR="${UPDATE_DIR}/migrations"

mkdir -p "${UPDATE_DIR}" "${BACKUP_DIR}" "$(dirname "${LOCK_FILE}")"
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
BACKUP_FILE="${BACKUP_DIR}/pre-update-${VERSION}-$(date -u +%Y%m%dT%H%M%SZ).dump"
SUCCESS=0

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
}

restore_database() {
  [[ -s "${BACKUP_FILE}" ]] || return 1
  docker exec -e PGPASSWORD="${DATABASE_PASSWORD}" "${DATABASE_CONTAINER}" psql -U "${DATABASE_USER}" -d postgres -v ON_ERROR_STOP=1 \
    -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname='${DATABASE_NAME}' AND pid <> pg_backend_pid();" >/dev/null
  docker exec -i -e PGPASSWORD="${DATABASE_PASSWORD}" "${DATABASE_CONTAINER}" pg_restore -U "${DATABASE_USER}" -d "${DATABASE_NAME}" \
    --clean --if-exists --no-owner --no-acl <"${BACKUP_FILE}"
}

rollback() {
  local exit_code="$?"
  trap - ERR
  [[ "${SUCCESS}" == "1" ]] && return 0
  status failed 90 "Falha na implantação; restaurando banco e versão anterior"
  VENTURERP_IMAGE="${TARGET_IMAGE}" VENTURERP_API_ENV="${API_ENV_FILE}" \
    docker compose -f "${COMPOSE_FILE}" down --remove-orphans >/dev/null 2>&1 || true
  restore_database || true
  if [[ -n "${PREVIOUS_IMAGE}" ]]; then
    VENTURERP_IMAGE="${PREVIOUS_IMAGE}" VENTURERP_API_ENV="${API_ENV_FILE}" \
      docker compose -f "${COMPOSE_FILE}" up -d
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

status running 15 "Criando e verificando backup transacional"
docker exec -e PGPASSWORD="${DATABASE_PASSWORD}" "${DATABASE_CONTAINER}" pg_dump -U "${DATABASE_USER}" -d "${DATABASE_NAME}" \
  --format=custom --no-owner --no-acl >"${BACKUP_FILE}"
[[ -s "${BACKUP_FILE}" ]]
docker exec -i "${DATABASE_CONTAINER}" pg_restore --list <"${BACKUP_FILE}" >/dev/null

status running 30 "Baixando imagem assinada pelo pipeline"
docker pull "${TARGET_IMAGE}"
rm -rf "${MIGRATIONS_DIR}"
container_id="$(docker create "${TARGET_IMAGE}")"
trap 'docker rm -f "${container_id}" >/dev/null 2>&1 || true; rollback' ERR INT TERM
docker cp "${container_id}:/app/migrations" "${MIGRATIONS_DIR}"
docker rm "${container_id}" >/dev/null
trap rollback ERR INT TERM

status running 45 "Parando a API e aplicando migrations"
systemctl stop "${LEGACY_SERVICE}" >/dev/null 2>&1 || true
if [[ -n "${PREVIOUS_IMAGE}" ]]; then
  VENTURERP_IMAGE="${PREVIOUS_IMAGE}" VENTURERP_API_ENV="${API_ENV_FILE}" \
    docker compose -f "${COMPOSE_FILE}" down --remove-orphans
fi
docker run --rm --network host -v "${MIGRATIONS_DIR}:/migrations:ro" migrate/migrate:v4.17.1 \
  -path=/migrations -database="${DATABASE_URL}" up

status running 70 "Iniciando a nova versão"
VENTURERP_IMAGE="${TARGET_IMAGE}" VENTURERP_API_ENV="${API_ENV_FILE}" \
  docker compose -f "${COMPOSE_FILE}" up -d

status running 85 "Executando health-check de prontidão"
healthy=0
for ((attempt=1; attempt<=HEALTH_ATTEMPTS; attempt++)); do
  if curl --fail --silent --show-error "${HEALTH_URL}" >/dev/null; then healthy=1; break; fi
  sleep "${HEALTH_INTERVAL_SECONDS}"
done
[[ "${healthy}" == "1" ]]

printf '%s\n' "${TARGET_IMAGE}" >"${STATE_FILE}"
SUCCESS=1
trap - ERR INT TERM
status succeeded 100 "VentureERP atualizado com sucesso para v${VERSION}" "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
rm -f "${REQUEST_FILE}" "${ACTIVE_LOCK}"
find "${BACKUP_DIR}" -type f -name 'pre-update-*.dump' -mtime +30 -delete
