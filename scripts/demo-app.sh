#!/usr/bin/env bash
# Atualiza o demo com o código desta pasta e abre o app desktop em cima dele.
# O front vem da pasta do mesmo branch em FRONT_ROOT; FRONT_DIR escolhe outra.
set -euo pipefail

cd "$(dirname "$0")/.."

FRONT_ROOT="${FRONT_ROOT:-$HOME/Documentos/front/erp}"
FRONT_PADRAO=app-venture

if [ -z "${FRONT_DIR:-}" ]; then
  branch=$(git branch --show-current)
  for pasta in "$FRONT_ROOT"/*/; do
    [ -f "$pasta/src-tauri/tauri.demo.conf.json" ] || continue
    if [ "$(git -C "$pasta" branch --show-current 2>/dev/null)" = "$branch" ]; then
      FRONT_DIR=$pasta
      break
    fi
  done
  FRONT_DIR="${FRONT_DIR:-$FRONT_ROOT/$FRONT_PADRAO}"
fi

if ss -ltn | grep -qE '[:.]5173\b'; then
  echo "Já há um app aberto (porta 5173). Feche-o e rode de novo; ele pode estar ligado a outra API." >&2
  exit 1
fi

bash scripts/demo-sync.sh

cd "$FRONT_DIR"
echo "▸ Tela: $(pwd) ($(git branch --show-current), $(git status --short | wc -l) arquivo(s) sem commit)"
echo "▸ Login: admin@panossoerp.demo / Demo@12345"
# Variável herdada do shell passaria por cima do .env.demo e ligaria o app em outra API.
unset VITE_API_PROXY_TARGET VITE_API_URL
exec npm run tauri:demo
