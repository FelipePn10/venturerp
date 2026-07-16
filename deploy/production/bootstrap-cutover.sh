#!/usr/bin/env bash
#
# Primeiro cutover: migra a produção do binário nativo (venturerp.service) para
# a imagem versionada em container, usando o mesmo self-update.sh das próximas
# atualizações. Requer que a imagem ${IMAGE_REPOSITORY}:vVERSION já exista.
#
#   sudo /opt/venturerp/updater/bootstrap-cutover.sh 1.0.0
#
# Escreve o pedido e executa o updater de forma síncrona para acompanhar a saída.
# O path unit do systemd também dispararia; o flock em self-update.sh garante uma
# única execução.
set -Eeuo pipefail

[[ "${EUID}" -eq 0 ]] || { echo "cutover: rode como root (sudo)"; exit 1; }

VERSION="${1:-}"
[[ "${VERSION}" =~ ^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?$ ]] ||
  { echo "uso: bootstrap-cutover.sh <versao-semver>  (ex.: 1.0.0)"; exit 2; }

CONFIG_FILE="${VENTURERP_UPDATE_CONFIG:-/etc/venturerp/update.env}"
[[ -r "${CONFIG_FILE}" ]] || { echo "cutover: configuração ausente: ${CONFIG_FILE} (rode provision-updater.sh)"; exit 1; }
# shellcheck disable=SC1090
source "${CONFIG_FILE}"
: "${UPDATE_DIR:?}" "${IMAGE_REPOSITORY:?}"

echo "cutover: confirmando que a imagem ${IMAGE_REPOSITORY}:v${VERSION} é acessível"
docker manifest inspect "${IMAGE_REPOSITORY}:v${VERSION}" >/dev/null 2>&1 ||
  docker pull "${IMAGE_REPOSITORY}:v${VERSION}" >/dev/null

install -d -m 0750 "${UPDATE_DIR}"
now="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
umask 077
printf '{"version":"%s","requested_at":"%s"}\n' "${VERSION}" "${now}" >"${UPDATE_DIR}/request.json"
chmod 600 "${UPDATE_DIR}/request.json"

echo "cutover: executando o updater (backup -> pull -> migrate -> up -> health-check)"
UPDATER="$(dirname "$(readlink -f "${BASH_SOURCE[0]}")")/self-update.sh"
[[ -x "${UPDATER}" ]] || UPDATER="/opt/venturerp/updater/self-update.sh"
"${UPDATER}"

echo "cutover: verificando /api/version"
curl -fsS "http://127.0.0.1:5070/api/version" && echo
echo "cutover: concluído. A partir de agora as atualizações saem pelo botão do painel admin."
