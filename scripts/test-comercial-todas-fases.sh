#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

export GOCACHE="${GOCACHE:-/tmp/panossoerp-go-build}"

scripts=(
  scripts/test-comercial-pricing.sh
  scripts/test-comercial-politicas.sh
  scripts/test-comercial-orcamentos.sh
  scripts/test-comercial-pedido-venda.sh
  scripts/test-comercial-representantes.sh
  scripts/test-comercial-metas.sh
  scripts/test-comercial-previsao-vendas.sh
  scripts/test-comercial-promessa-entrega.sh
  scripts/test-comercial-assistencia-tecnica.sh
  scripts/test-comercial-sac.sh
  scripts/test-comercial-vendas-recorrentes.sh
  scripts/test-comercial-expedicao.sh
  scripts/test-comercial-faturamento.sh
)

for script in "${scripts[@]}"; do
  echo
  echo "============================================================"
  echo "Executando ${script}"
  echo "============================================================"
  bash "$script"
done

echo
echo "==> Suite Go completa"
env GOCACHE="$GOCACHE" go test ./...

echo
echo "==> Varredura de documentação comercial/fiscal/expedição"
if rg -n "Focco|FoccoERP|SAP|Oracle|FFAT|FNFC|FPDV|FCST|FPRV|FREP|FMET|FPRE|FPME|FASS|FATC|CATC|FCVN|FVRE|FPLC|FITE|CPDV|CPRV|CASS|FCTR|FUTL|CFAT" docs SESSION_SUMMARY.md -S; then
  echo "Foram encontradas referências externas/códigos antigos na documentação." >&2
  exit 1
fi

echo "Todas as fases comerciais validadas."
