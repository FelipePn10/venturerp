#!/bin/sh
# backup.sh — one logical backup of the VentureERP database.
#
# Uses pg_dump custom format (-Fc): compressed and restorable selectively /
# in parallel with pg_restore. Connection comes from standard libpq env vars
# (PGHOST, PGPORT, PGUSER, PGPASSWORD, PGDATABASE) or a single DATABASE_URL.
#
# Usage:
#   PGHOST=localhost PGUSER=panossoerp PGPASSWORD=*** PGDATABASE=panossoerpdatabase ./scripts/backup.sh
#   DATABASE_URL=postgres://user:pass@host:5432/db ./scripts/backup.sh
#
# Env:
#   BACKUP_DIR             output directory (default: ./backups or /backups in container)
#   BACKUP_RETENTION_DAYS  delete dumps older than N days (default: 14; 0 = keep all)
set -eu

BACKUP_DIR="${BACKUP_DIR:-$( [ -d /backups ] && echo /backups || echo ./backups )}"
RETENTION="${BACKUP_RETENTION_DAYS:-14}"
DB_LABEL="${PGDATABASE:-panossoerpdatabase}"
STAMP="$(date +%Y%m%d-%H%M%S)"
OUT="${BACKUP_DIR}/${DB_LABEL}-${STAMP}.dump"

mkdir -p "${BACKUP_DIR}"

echo "[backup] starting → ${OUT}"
if [ -n "${DATABASE_URL:-}" ]; then
    pg_dump --format=custom --no-owner --no-privileges --file="${OUT}" "${DATABASE_URL}"
else
    pg_dump --format=custom --no-owner --no-privileges --file="${OUT}"
fi

# Integrity check: pg_restore --list must parse the archive.
pg_restore --list "${OUT}" >/dev/null
SIZE="$(wc -c < "${OUT}" 2>/dev/null || echo '?')"
echo "[backup] ok → ${OUT} (${SIZE} bytes)"

if [ "${RETENTION}" -gt 0 ] 2>/dev/null; then
    echo "[backup] pruning dumps older than ${RETENTION} day(s)"
    find "${BACKUP_DIR}" -name "${DB_LABEL}-*.dump" -type f -mtime "+${RETENTION}" -print -delete || true
fi

echo "[backup] done"
