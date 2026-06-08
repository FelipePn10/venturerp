#!/bin/sh
# restore.sh — restore a VentureERP database from a pg_dump custom-format file.
#
# DESTRUCTIVE: with --clean it drops existing objects before recreating them.
# Always test restores on a scratch database first (this is the whole point of
# having backups — an untested backup is a hope, not a backup).
#
# Usage:
#   PGHOST=localhost PGUSER=panossoerp PGPASSWORD=*** PGDATABASE=panossoerpdatabase \
#       ./scripts/restore.sh ./backups/panossoerpdatabase-20260608-020000.dump
#   DATABASE_URL=postgres://user:pass@host:5432/db ./scripts/restore.sh <file>
set -eu

DUMP="${1:-}"
if [ -z "${DUMP}" ] || [ ! -f "${DUMP}" ]; then
    echo "usage: restore.sh <path-to.dump>" >&2
    exit 2
fi

echo "[restore] verifying archive ${DUMP}"
pg_restore --list "${DUMP}" >/dev/null

echo "[restore] restoring into ${PGDATABASE:-target db} (this can drop & recreate objects)"
if [ -n "${DATABASE_URL:-}" ]; then
    pg_restore --clean --if-exists --no-owner --no-privileges --exit-on-error \
        --dbname="${DATABASE_URL}" "${DUMP}"
else
    pg_restore --clean --if-exists --no-owner --no-privileges --exit-on-error \
        --dbname="${PGDATABASE:-panossoerpdatabase}" "${DUMP}"
fi

echo "[restore] done"
