#!/bin/sh
# backup-loop.sh — entrypoint for the compose `backup` sidecar.
# Runs backup.sh immediately, then every BACKUP_INTERVAL_SECONDS. A crude but
# dependency-free scheduler appropriate for a single-node deployment.
set -eu

INTERVAL="${BACKUP_INTERVAL_SECONDS:-86400}"  # default: daily

echo "[backup-loop] interval=${INTERVAL}s retention=${BACKUP_RETENTION_DAYS:-14}d"
while true; do
    if /bin/sh /scripts/backup.sh; then
        echo "[backup-loop] cycle ok; sleeping ${INTERVAL}s"
    else
        echo "[backup-loop] backup FAILED; retrying after ${INTERVAL}s" >&2
    fi
    sleep "${INTERVAL}"
done
