#!/usr/bin/env bash
#
# Provisiona o trilho de autoatualização do backend na VPS (idempotente).
#
# Instala self-update.sh + compose.yml em /opt/venturerp/updater, gera
# /etc/venturerp/update.env a partir do .env de produção, instala as units
# systemd e habilita o watcher que consome os pedidos do botão do painel admin.
#
# Uso (como root, a partir do checkout do repositório na VPS OU enviando os
# arquivos por scp para um diretório e rodando de lá):
#
#   sudo ./deploy/production/provision-updater.sh
#
# Variáveis de ambiente aceitas (todas têm padrão sensato):
#   PROD_ENV_FILE      .env de produção lido para derivar o DATABASE_URL
#   IMAGE_REPOSITORY   repositório da imagem GHCR
#   DATABASE_CONTAINER nome do container postgres de produção
set -Eeuo pipefail

[[ "${EUID}" -eq 0 ]] || { echo "provision: rode como root (sudo)"; exit 1; }

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

PROD_ENV_FILE="${PROD_ENV_FILE:-/opt/venturerp/panossoerp/.env}"
IMAGE_REPOSITORY="${IMAGE_REPOSITORY:-ghcr.io/felipepn10/panossoerp}"
DATABASE_CONTAINER="${DATABASE_CONTAINER:-venturerp-postgres}"
UPDATER_DIR="/opt/venturerp/updater"
UPDATE_DIR="/var/lib/venturerp-update"
BACKUP_DIR="/var/backups/venturerp/releases"
CONFIG_DIR="/etc/venturerp"
API_ENV_FILE="${PROD_ENV_FILE}"

echo "provision: verificando pré-requisitos"
for cmd in docker jq curl flock systemctl; do command -v "${cmd}" >/dev/null || { echo "faltando: ${cmd}"; exit 1; }; done
docker compose version >/dev/null || { echo "docker compose indisponível"; exit 1; }
[[ -r "${PROD_ENV_FILE}" ]] || { echo "não encontrei o .env de produção em ${PROD_ENV_FILE}"; exit 1; }
docker inspect "${DATABASE_CONTAINER}" >/dev/null || { echo "container ${DATABASE_CONTAINER} não existe"; exit 1; }

DATABASE_URL="$(grep -E '^DATABASE_URL=' "${PROD_ENV_FILE}" | head -1 | cut -d= -f2-)"
[[ -n "${DATABASE_URL}" ]] || { echo "DATABASE_URL ausente em ${PROD_ENV_FILE}"; exit 1; }
DB_USER="$(printf '%s' "${DATABASE_URL}" | sed -nE 's#^[a-z0-9+]+://([^:]+):.*#\1#p')"
DB_NAME="$(printf '%s' "${DATABASE_URL}" | sed -nE 's#^[a-z0-9+]+://[^/]+/([^?]+).*#\1#p')"
[[ -n "${DB_USER}" && -n "${DB_NAME}" ]] || { echo "não consegui derivar usuário/base do DATABASE_URL"; exit 1; }

echo "provision: criando diretórios"
APPUSER_UID="${APPUSER_UID:-10001}"
install -d -m 0755 "${UPDATER_DIR}"
# A fila é a ponte API↔host: o appuser (não-root) do contêiner escreve nela.
install -d -o "${APPUSER_UID}" -g "${APPUSER_UID}" -m 0770 "${UPDATE_DIR}"
install -d -m 0750 "${BACKUP_DIR}"
install -d -m 0750 "${CONFIG_DIR}"

echo "provision: instalando scripts e compose"
install -m 0755 "${SCRIPT_DIR}/../../scripts/self-update.sh" "${UPDATER_DIR}/self-update.sh"
install -m 0755 "${SCRIPT_DIR}/bootstrap-cutover.sh" "${UPDATER_DIR}/bootstrap-cutover.sh"
install -m 0644 "${SCRIPT_DIR}/compose.yml" "${UPDATER_DIR}/compose.yml"

if [[ -f "${CONFIG_DIR}/update.env" ]]; then
  echo "provision: ${CONFIG_DIR}/update.env já existe — preservando"
else
  echo "provision: gerando ${CONFIG_DIR}/update.env"
  umask 077
  cat >"${CONFIG_DIR}/update.env" <<EOF
# Gerado por provision-updater.sh. Mode 0600.
IMAGE_REPOSITORY=${IMAGE_REPOSITORY}
COMPOSE_FILE=${UPDATER_DIR}/compose.yml
API_ENV_FILE=${API_ENV_FILE}
UPDATE_DIR=${UPDATE_DIR}
BACKUP_DIR=${BACKUP_DIR}
DATABASE_CONTAINER=${DATABASE_CONTAINER}
DATABASE_USER=${DB_USER}
DATABASE_NAME=${DB_NAME}
DATABASE_URL=${DATABASE_URL}
HEALTH_URL=http://127.0.0.1:5070/health/ready
LEGACY_SERVICE=venturerp.service
HEALTH_ATTEMPTS=30
HEALTH_INTERVAL_SECONDS=2
EOF
  chmod 600 "${CONFIG_DIR}/update.env"
fi

echo "provision: instalando units systemd"
install -m 0644 "${SCRIPT_DIR}/systemd/venturerp-update.service" /etc/systemd/system/venturerp-update.service
install -m 0644 "${SCRIPT_DIR}/systemd/venturerp-update.path" /etc/systemd/system/venturerp-update.path
# A unit de serviço aponta para /opt/venturerp/updater/self-update.sh (já instalado).
systemctl daemon-reload
systemctl enable --now venturerp-update.path

echo "provision: concluído."
echo "  updater : ${UPDATER_DIR}"
echo "  config  : ${CONFIG_DIR}/update.env (0600)"
echo "  watcher : $(systemctl is-active venturerp-update.path)"
echo
echo "Primeiro cutover (binário nativo -> container) após a imagem ${IMAGE_REPOSITORY}:vX.Y.Z existir:"
echo "  sudo ${UPDATER_DIR}/bootstrap-cutover.sh 1.0.0"
