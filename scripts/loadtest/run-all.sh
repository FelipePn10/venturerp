#!/usr/bin/env bash
# Orchestrator – roda todos os testes e gera relatorio consolidado.
#
# Uso:
#   bash scripts/loadtest/run-all.sh          # Testes completos
#   bash scripts/loadtest/run-all.sh quick    # Apenas load test rapido
#   bash scripts/loadtest/run-all.sh chaos    # Apenas chaos engineering
#
# Requisitos:
#   - Docker (para k6) ou k6 instalado localmente
#   - Acesso SSH ao servidor (para chaos tests)
#   - curl, jq

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RESULTS_DIR="$SCRIPT_DIR/results"
TARGET_URL="${TARGET_URL:-https://api.venturerp.com}"
MODE="${1:-full}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

mkdir -p "$RESULTS_DIR"

log()  { echo -e "${BOLD}[$(date +%H:%M:%S)]${NC} $*"; }
ok()   { echo -e "  ${GREEN}OK${NC}  $*"; }
fail() { echo -e "  ${RED}FAIL${NC} $*"; }

banner() {
  echo ""
  echo -e "${CYAN}╔══════════════════════════════════════════════════════════════╗${NC}"
  echo -e "${CYAN}║${NC}     ${BOLD}VENTURERP — API LOAD & CHAOS TESTING SUITE${NC}                ${CYAN}║${NC}"
  echo -e "${CYAN}╚══════════════════════════════════════════════════════════════╝${NC}"
  echo ""
}

check_api() {
  log "Verificando conectividade com $TARGET_URL ..."
  local code
  code=$(curl -s -o /dev/null -w "%{http_code}" --max-time 10 "$TARGET_URL/health" 2>/dev/null || echo "000")
  if [ "$code" = "200" ]; then
    ok "API online — $TARGET_URL ($code)"
    return 0
  else
    fail "API offline ou inacessivel — codigo: $code"
    return 1
  fi
}

ensure_k6() {
  if command -v k6 &>/dev/null; then
    ok "k6 encontrado: $(k6 version 2>&1 | head -1)"
    K6_CMD="k6"
    return
  fi

  if command -v docker &>/dev/null && docker info &>/dev/null 2>&1; then
    ok "Usando k6 via Docker"
    K6_CMD="docker run --rm -i \
      --user $(id -u):$(id -g) \
      -v $SCRIPT_DIR:/scripts:ro \
      -e TARGET_URL=$TARGET_URL \
      --network host \
      grafana/k6"
    return
  fi

  fail "k6 nao encontrado. Instale: https://k6.io/docs/get-started/installation/"
  exit 1
}

run_k6_test() {
  local name="$1"
  local script="$2"
  local result_file="$RESULTS_DIR/${name}-summary.json"

  echo ""
  echo -e "${CYAN}══════════════════════════════════════════════════${NC}"
  echo -e "${CYAN}  $name${NC}"
  echo -e "${CYAN}══════════════════════════════════════════════════${NC}"

  log "Executando $script ..."

  local start=$(date +%s)

  $K6_CMD run "/scripts/k6/$(basename "$script")" \
    --out json="$result_file" \
    2>&1 || true

  local end=$(date +%s)
  local elapsed=$((end - start))

  log "Duracao: ${elapsed}s"

  if [ -f "$result_file" ]; then
    local size
    size=$(du -h "$result_file" | cut -f1)
    ok "Resultado salvo: $result_file ($size)"
  fi
}

run_chaos() {
  echo ""
  echo -e "${CYAN}══════════════════════════════════════════════════${NC}"
  echo -e "${CYAN}  Chaos Engineering${NC}"
  echo -e "${CYAN}══════════════════════════════════════════════════${NC}"

  local chaos_script="$SCRIPT_DIR/chaos/chaos.sh"

  if [ ! -f "$chaos_script" ]; then
    fail "Script de chaos nao encontrado: $chaos_script"
    return
  fi

  log "Chaos Engineering deve rodar NO SERVIDOR."
  echo ""
  echo "  Execute manualmente:"
  echo "  ssh erp"
  echo "  sudo bash /opt/venturerp/panossoerp/scripts/loadtest/chaos/chaos.sh"
  echo ""

  read -rp "  Deseja executar via SSH agora? [s/N] " answer
  if [ "$answer" = "s" ] || [ "$answer" = "S" ]; then
    log "Conectando via SSH..."
    ssh erp "sudo bash /opt/venturerp/panossoerp/scripts/loadtest/chaos/chaos.sh" || fail "Chaos tests falharam"
  fi
}

generate_final_report() {
  local report="$RESULTS_DIR/report-$(date +%Y%m%d-%H%M%S).md"

  {
    echo "# VentureRP API — Relatorio de Testes"
    echo ""
    echo "**Data:** $(date '+%d/%m/%Y %H:%M')"
    echo "**Alvo:** $TARGET_URL"
    echo "**Ambiente:** Producao (VPS Hetzner — Ubuntu 26.04)"
    echo ""
    echo "---"
    echo ""
    echo "## 1. Testes Executados"
    echo ""
    echo "| Teste | Tipo | Duracao | Resultado |"
    echo "|-------|------|---------|-----------|"
    echo "| Load Test | Rampa 5→200 VUs | ~5min | $( [ -f "$RESULTS_DIR/load-test-summary.json" ] && echo 'Concluido' || echo 'Pendente') |"
    echo "| Stress Test | Quebra 20→1000 VUs | ~5min | $( [ -f "$RESULTS_DIR/stress-test-summary.json" ] && echo 'Concluido' || echo 'Pendente') |"
    echo "| Soak Test | Sustentado 30 VUs | ~10min | $( [ -f "$RESULTS_DIR/soak-test-summary.json" ] && echo 'Concluido' || echo 'Pendente') |"
    echo "| Chaos | CPU/Mem/Rede | ~5min | $( ls "$RESULTS_DIR"/chaos-report-* 2>/dev/null && echo 'Concluido' || echo 'Pendente') |"
    echo ""
    echo "---"
    echo ""
    echo "## 2. Recomendacoes"
    echo ""
    echo "- Monitorar os dashboards do Grafana durante os testes"
    echo "- Aumentar rate limits se necessario (.env: RATE_LIMIT_RPS)"
    echo "- Verificar conexoes do Postgres (pg_stat_database_numbackends)"
    echo "- Considerar escalar verticalmente a VPS se p95 > 1000ms"
    echo ""
    echo "---"
    echo ""
    echo "## 3. Metricas do Servidor (coletar durante teste)"
    echo ""
    echo '```bash'
    echo '# CPU'
    echo "top -bn1 | head -5"
    echo '# Memoria'
    echo "free -h"
    echo '# Conexoes Postgres'
    echo "ss -tn state established | wc -l"
    echo '# Load'
    echo "uptime"
    echo '```'
  } > "$report"

  ok "Relatorio consolidado: $report"
}

banner

case "$MODE" in
  quick)
    check_api || exit 1
    ensure_k6
    run_k6_test "quick-load-test" "k6/load-test.js"
    generate_final_report
    ;;
  chaos)
    run_chaos
    ;;
  load)
    check_api || exit 1
    ensure_k6
    run_k6_test "load-test" "k6/load-test.js"
    generate_final_report
    ;;
  stress)
    check_api || exit 1
    ensure_k6
    run_k6_test "stress-test" "k6/stress-test.js"
    generate_final_report
    ;;
  full|*)
    check_api || exit 1
    ensure_k6

    run_k6_test "01-load-test"  "k6/load-test.js"

    echo ""
    read -rp "  Continuar para Stress Test? [S/n] " answer
    if [ "$answer" != "n" ] && [ "$answer" != "N" ]; then
      run_k6_test "02-stress-test" "k6/stress-test.js"
    fi

    echo ""
    read -rp "  Continuar para Soak Test (10 min)? [S/n] " answer
    if [ "$answer" != "n" ] && [ "$answer" != "N" ]; then
      run_k6_test "03-soak-test" "k6/soak-test.js"
    fi

    run_chaos
    generate_final_report
    ;;
esac

echo ""
ok "Testes concluidos!"
echo ""
echo "  Resultados em: $RESULTS_DIR/"
echo "  Grafana:       https://grafana.venturerp.com"
echo ""
