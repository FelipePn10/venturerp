#!/usr/bin/env bash
#
# Backup lógico de TODAS as bases de empresa do VentureERP.
#
# Substitui o backup que cobria uma base só. Com mais de uma empresa na mesma VPS,
# um script que nomeia a base explicitamente protege quem foi lembrado e deixa o
# cliente novo sem backup até alguém notar — normalmente, no dia em que precisa
# restaurar. Aqui as bases são DESCOBERTAS: toda base que não seja de sistema
# entra, então uma empresa nova está protegida no instante em que existe.
#
# Cada base gera um dump próprio (formato custom, comprimido), é verificada com
# pg_restore --list e ganha um sha256. A retenção é aplicada por base, para que o
# volume de uma empresa não expulse o backup da outra.
#
# Instalação (como root, na VPS):
#   install -m 0755 backup-databases.sh /opt/venturerp/database/backup-databases.sh
#   # aponte o ExecStart de venturerp-db-backup.service para este arquivo
#   systemctl daemon-reload && systemctl start venturerp-db-backup.service
#
# Configuração, lida de /opt/venturerp/database/.env (o mesmo de hoje):
#   POSTGRES_ADMIN_USER, POSTGRES_ADMIN_PASSWORD   superusuário do PostgreSQL
#
# Variáveis opcionais:
#   BACKUP_DIR        destino          (padrão: /mnt/HC_Volume_105957343/venturerp-backups)
#   RETENTION_DAYS    retenção         (padrão: 30)
#   DATABASE_CONTAINER contêiner       (padrão: venturerp-postgres)
#   SKIP_DATABASES    bases a ignorar, separadas por espaço
#                     (padrão: postgres template0 template1 venturerp_training
#                      venturerp_development)
set -Eeuo pipefail
umask 077

DB_DIR="${DB_DIR:-/opt/venturerp/database}"
BACKUP_DIR="${BACKUP_DIR:-/mnt/HC_Volume_105957343/venturerp-backups}"
RETENTION_DAYS="${RETENTION_DAYS:-30}"
DATABASE_CONTAINER="${DATABASE_CONTAINER:-venturerp-postgres}"
# ⚠️ Treinamento e desenvolvimento ficam FORA do backup de produção por contrato:
# são ambientes descartáveis, com dados de exercício, e entrar no backup os
# colocaria também na retenção e no restore — um restore de produção não pode
# ressuscitar base de treinamento. Descobrir as bases é o certo (empresa nova entra
# sozinha), mas descobrir sem excluir estas duas traz o que não deveria estar lá.
SKIP_DATABASES="${SKIP_DATABASES:-postgres template0 template1 venturerp_training venturerp_development}"

if [[ -r "${DB_DIR}/.env" ]]; then
  set -a
  # shellcheck disable=SC1091
  . "${DB_DIR}/.env"
  set +a
fi

: "${POSTGRES_ADMIN_USER:?POSTGRES_ADMIN_USER é obrigatório}"
: "${POSTGRES_ADMIN_PASSWORD:?POSTGRES_ADMIN_PASSWORD é obrigatório}"

mkdir -p "${BACKUP_DIR}"
stamp="$(date -u +%Y%m%dT%H%M%SZ)"

psql_admin() {
  docker exec -e PGPASSWORD="${POSTGRES_ADMIN_PASSWORD}" "${DATABASE_CONTAINER}" \
    psql -tAX -U "${POSTGRES_ADMIN_USER}" -d postgres -c "$1"
}

mapfile -t bases < <(psql_admin "SELECT datname FROM pg_database WHERE datistemplate = false ORDER BY datname")

alvos=()
for base in "${bases[@]}"; do
  [[ -z "${base}" ]] && continue
  pular=0
  for excluida in ${SKIP_DATABASES}; do
    [[ "${base}" == "${excluida}" ]] && pular=1
  done
  [[ "${pular}" == "1" ]] || alvos+=("${base}")
done

# Nenhuma base é sinal de que a descoberta falhou, não de que não há o que salvar.
# Sair com sucesso aqui faria o timer reportar backup em dia sem ter salvado nada.
if [[ "${#alvos[@]}" -eq 0 ]]; then
  printf 'backup_erro motivo=nenhuma_base_encontrada container=%s\n' "${DATABASE_CONTAINER}" >&2
  exit 1
fi

falhas=0
for base in "${alvos[@]}"; do
  tmp="${BACKUP_DIR}/.${base}-${stamp}.dump.partial"
  out="${BACKUP_DIR}/${base}-${stamp}.dump"

  # Uma base que falha não interrompe as demais: a empresa B não deve ficar sem
  # backup porque a base da empresa A teve problema.
  if ! docker exec -e PGPASSWORD="${POSTGRES_ADMIN_PASSWORD}" "${DATABASE_CONTAINER}" \
        pg_dump -U "${POSTGRES_ADMIN_USER}" -d "${base}" \
          --format=custom --compress=9 --no-owner --no-privileges >"${tmp}" 2>/dev/null; then
    rm -f "${tmp}"
    printf 'backup_erro base=%s etapa=pg_dump\n' "${base}" >&2
    falhas=$((falhas + 1))
    continue
  fi

  # Um dump que pg_restore não consegue abrir é um arquivo, não um backup.
  if ! docker exec -i "${DATABASE_CONTAINER}" pg_restore --list <"${tmp}" >/dev/null 2>&1; then
    rm -f "${tmp}"
    printf 'backup_erro base=%s etapa=verificacao\n' "${base}" >&2
    falhas=$((falhas + 1))
    continue
  fi

  mv "${tmp}" "${out}"
  sha256sum "${out}" >"${out}.sha256"
  printf 'backup_ok base=%s file=%s bytes=%s\n' "${base}" "${out}" "$(stat -c %s "${out}")"

  # Retenção por base: o prefixo do nome é a chave, então o volume de uma empresa
  # nunca expulsa o backup de outra.
  find "${BACKUP_DIR}" -maxdepth 1 -type f \
    \( -name "${base}-*.dump" -o -name "${base}-*.dump.sha256" \) \
    -mtime "+${RETENTION_DAYS}" -delete
done

# Partials de execuções interrompidas não devem acumular no volume.
find "${BACKUP_DIR}" -maxdepth 1 -type f -name '.*.dump.partial' -mtime +1 -delete 2>/dev/null || true

printf 'backup_resumo bases=%d falhas=%d\n' "${#alvos[@]}" "${falhas}"
[[ "${falhas}" -eq 0 ]]
