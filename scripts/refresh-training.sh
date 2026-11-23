#!/usr/bin/env bash
# Modo simulação (melhoria 2): copia o banco de uma empresa para o banco de
# treino, saneia e sobe a API de treino. Disparado pelo systemd
# (venturerp-simulacao.path) quando a API de produção grava o pedido em
# ${SIMULATION_ROOT}/<empresa>/request.json — ver
# internal/application/usecase/system_update_uc/simulacao.go.
#
# A API não acessa o Docker nem o banco de treino: o pedido traz só o NOME do
# banco de origem, conferido aqui contra a lista do host (SIMULATION_SOURCES).
set -Eeuo pipefail
umask 077

CONFIG_FILE="${VENTURERP_UPDATE_CONFIG:-/etc/venturerp/update.env}"
[[ -r "${CONFIG_FILE}" ]] || { printf 'simulacao: configuração ausente: %s\n' "${CONFIG_FILE}" >&2; exit 1; }
# shellcheck disable=SC1090
source "${CONFIG_FILE}"

: "${DATABASE_CONTAINER:?}"
: "${TRAINING_DATABASE_URL:?TRAINING_DATABASE_URL é obrigatório para o modo simulação}"
SIMULATION_ROOT="${SIMULATION_ROOT:-/var/lib/venturerp-simulacao}"
# Bancos que podem ser copiados para o treino (separados por espaço). A
# identidade do treino (IDENTITY_DATABASE_URL do .env.training) precisa ser a
# mesma empresa da origem, senão ninguém entra na cópia.
: "${SIMULATION_SOURCES:?SIMULATION_SOURCES lista os bancos de origem permitidos}"
SIMULATION_DB_SUPERUSER="${SIMULATION_DB_SUPERUSER:-postgres}"
SIMULATION_DB_SUPERUSER_PASSWORD="${SIMULATION_DB_SUPERUSER_PASSWORD:-}"
# O banco de treino pode ficar noutro servidor PostgreSQL (ex.: o de
# desenvolvimento), para a cópia não disputar recursos com a produção. A origem
# continua lida do DATABASE_CONTAINER (produção).
TRAINING_DATABASE_CONTAINER="${TRAINING_DATABASE_CONTAINER:-${DATABASE_CONTAINER}}"
TRAINING_DB_SUPERUSER="${TRAINING_DB_SUPERUSER:-${SIMULATION_DB_SUPERUSER}}"
TRAINING_DB_SUPERUSER_PASSWORD="${TRAINING_DB_SUPERUSER_PASSWORD:-${SIMULATION_DB_SUPERUSER_PASSWORD}}"
TRAINING_CONTAINER="${TRAINING_CONTAINER:-venturerp-api-training}"
TRAINING_HEALTH_URL="${TRAINING_HEALTH_URL:-http://127.0.0.1:5071/health/ready}"
SIMULATION_TRAINING_URL="${SIMULATION_TRAINING_URL:-}"
SANEAMENTO_SQL="${SANEAMENTO_SQL:-/opt/venturerp/updater/simulacao-saneamento.sql}"
UPDATE_UID="${UPDATE_UID:-10001}"
UPDATE_GID="${UPDATE_GID:-10001}"
HEALTH_ATTEMPTS="${HEALTH_ATTEMPTS:-30}"
HEALTH_INTERVAL_SECONDS="${HEALTH_INTERVAL_SECONDS:-2}"
LOCK_FILE="${SIMULATION_LOCK_FILE:-/run/lock/venturerp-simulacao.lock}"

url_part() { # $1=url $2=user|database
  case "$2" in
    user)     printf '%s' "$1" | sed -nE 's#^[a-z0-9+]+://([^:@/]+).*#\1#p' ;;
    database) printf '%s' "$1" | sed -nE 's#^[a-z0-9+]+://[^/]+/([^?]+).*$#\1#p' ;;
  esac
}
TRAINING_DB_USER="$(url_part "${TRAINING_DATABASE_URL}" user)"
TRAINING_DB_NAME="$(url_part "${TRAINING_DATABASE_URL}" database)"
[[ -n "${TRAINING_DB_USER}" && -n "${TRAINING_DB_NAME}" ]] || { printf 'simulacao: TRAINING_DATABASE_URL inválido\n' >&2; exit 1; }

valid_name() { [[ "$1" =~ ^[a-z_][a-z0-9_]{0,62}$ ]]; }

psql_super() { # $1=database, resto = argumentos (no servidor do TREINO)
  local db="$1"; shift
  docker exec -i -e PGPASSWORD="${TRAINING_DB_SUPERUSER_PASSWORD}" "${TRAINING_DATABASE_CONTAINER}" \
    psql -U "${TRAINING_DB_SUPERUSER}" -d "${db}" -v ON_ERROR_STOP=1 "$@"
}

status() { # dir state progress message [finished]
  local dir="$1" state="$2" progress="$3" message="$4" finished="${5:-}" tmp
  tmp="$(mktemp "${dir}/.status.XXXXXX")"
  jq -n --arg state "${state}" --arg message "${message}" --arg requested "${REQUESTED_AT:-}" \
    --arg finished "${finished}" --arg url "${SIMULATION_TRAINING_URL}" --argjson progress "${progress}" \
    '{state:$state,progress:$progress,message:$message}
     + (if $requested == "" then {} else {requested_at:$requested} end)
     + (if $finished == "" then {} else {finished_at:$finished} end)
     + (if $url == "" then {} else {training_url:$url} end)' >"${tmp}"
  chmod 640 "${tmp}"
  mv "${tmp}" "${dir}/status.json"
  chown "${UPDATE_UID}:${UPDATE_GID}" "${dir}/status.json" 2>/dev/null || true
}

await_health() {
  local attempt
  for ((attempt = 1; attempt <= HEALTH_ATTEMPTS; attempt++)); do
    if curl --fail --silent --show-error "${TRAINING_HEALTH_URL}" >/dev/null; then return 0; fi
    sleep "${HEALTH_INTERVAL_SECONDS}"
  done
  return 1
}

processar() { # dir
  local dir="$1" request="$1/request.json" source dump
  REQUESTED_AT="$(jq -r '.requested_at // empty' "${request}")"
  source="$(jq -r '.source_database // empty' "${request}")"
  if ! valid_name "${source}" || [[ " ${SIMULATION_SOURCES} " != *" ${source} "* ]]; then
    status "${dir}" failed 100 "Banco de origem não permitido para o treino: ${source}" "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    rm -f "${request}" "${dir}/active.lock"
    return 0
  fi
  if [[ "${source}" == "${TRAINING_DB_NAME}" ]]; then
    status "${dir}" failed 100 "A origem não pode ser o próprio banco de treino" "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    rm -f "${request}" "${dir}/active.lock"
    return 0
  fi
  dump="$(mktemp /tmp/venturerp-simulacao.XXXXXX.dump)"
  falhou() {
    status "${dir}" failed 100 "A cópia para o treino falhou (veja journalctl -u venturerp-simulacao)" "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    rm -f "${request}" "${dir}/active.lock" "${dump}"
    docker start "${TRAINING_CONTAINER}" >/dev/null 2>&1 || true
  }
  trap falhou ERR

  status "${dir}" running 10 "Copiando o banco ${source}"
  docker exec -e PGPASSWORD="${SIMULATION_DB_SUPERUSER_PASSWORD}" "${DATABASE_CONTAINER}" \
    pg_dump -U "${SIMULATION_DB_SUPERUSER}" -d "${source}" --format=custom --no-owner --no-acl >"${dump}"
  [[ -s "${dump}" ]]
  docker exec -i "${TRAINING_DATABASE_CONTAINER}" pg_restore --list <"${dump}" >/dev/null

  status "${dir}" running 40 "Recriando o banco de treino"
  docker stop "${TRAINING_CONTAINER}" >/dev/null 2>&1 || true
  psql_super postgres -c "DROP DATABASE IF EXISTS ${TRAINING_DB_NAME} WITH (FORCE);" -c "CREATE DATABASE ${TRAINING_DB_NAME} OWNER ${TRAINING_DB_USER};" >/dev/null
  docker exec -i -e PGPASSWORD="${TRAINING_DB_SUPERUSER_PASSWORD}" "${TRAINING_DATABASE_CONTAINER}" \
    pg_restore -U "${TRAINING_DB_SUPERUSER}" -d "${TRAINING_DB_NAME}" --no-owner --no-acl --role="${TRAINING_DB_USER}" --exit-on-error <"${dump}"

  status "${dir}" running 75 "Saneando a cópia (homologação, filas de envio, nome da empresa)"
  psql_super "${TRAINING_DB_NAME}" -c "SET ROLE ${TRAINING_DB_USER};" -f - <"${SANEAMENTO_SQL}" >/dev/null
  # Token de HOMOLOGAÇÃO da emissora para o treino emitir NF-e sem valor fiscal.
  # Vai por variável do psql (não aparece na linha de comando nem no log).
  if [[ -n "${TRAINING_FOCUS_TOKEN:-}" ]]; then
    printf '%s\n' "UPDATE fiscal_configs SET focus_nfe_token = :'token';" \
      | psql_super "${TRAINING_DB_NAME}" -v token="${TRAINING_FOCUS_TOKEN}" -f - >/dev/null
  fi

  status "${dir}" running 90 "Subindo a API de treino"
  docker start "${TRAINING_CONTAINER}" >/dev/null
  await_health

  trap - ERR
  rm -f "${dump}" "${request}" "${dir}/active.lock"
  status "${dir}" succeeded 100 "Cópia de ${source} pronta no treino" "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}

exec 9>"${LOCK_FILE}"
flock -n 9 || { printf 'simulacao: outra cópia em andamento\n' >&2; exit 0; }
shopt -s nullglob
for request in "${SIMULATION_ROOT}"/*/request.json; do
  processar "$(dirname "${request}")"
done
