#!/usr/bin/env bash
# Chaos Engineering – injeta falhas e mede impacto na API.
# Roda NO SERVIDOR (ssh erp). Requer: stress-ng, tc, curl, jq.
#
# Uso: sudo bash scripts/loadtest/chaos/chaos.sh
#
# Testes:
#   1. CPU stress — satura CPU, mede latencia
#   2. Memory stress — aloca memoria, mede latencia
#   3. Network latency — adiciona delay, mede latencia
#   4. Network packet loss — dropa pacotes, mede taxa de erro

set -euo pipefail

API_URL="${API_URL:-https://api.venturerp.com}"
HEALTH_URL="${HEALTH_URL:-http://127.0.0.1:5070/health}"
RESULTS_DIR="scripts/loadtest/results"
INTERFACE="${INTERFACE:-eth0}"
CHAOS_DURATION=30
BASELINE_REQUESTS=30

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'
BOLD='\033[1m'

mkdir -p "$RESULTS_DIR"
REPORT_FILE="$RESULTS_DIR/chaos-report-$(date +%Y%m%d-%H%M%S).txt"

log()  { echo -e "${BOLD}[$(date +%H:%M:%S)]${NC} $*"; }
ok()   { echo -e "  ${GREEN}OK${NC}  $*"; }
warn() { echo -e "  ${YELLOW}WARN${NC} $*"; }
fail() { echo -e "  ${RED}FAIL${NC} $*"; }

header() {
  echo ""
  echo "=============================================="
  echo "  $*"
  echo "=============================================="
}

# --- Helpers ---

check_prereqs() {
  local missing=""
  for cmd in curl jq bc; do
    if ! command -v "$cmd" &>/dev/null; then
      missing="$missing $cmd"
    fi
  done
  if ! command -v stress-ng &>/dev/null; then
    warn "stress-ng nao instalado — testes de CPU/Mem serao pulados"
    warn "Instale: apt install stress-ng"
  fi
  if ! command -v tc &>/dev/null; then
    warn "tc nao instalado — testes de rede serao pulados"
    warn "Instale: apt install iproute2"
  fi
  if [ -n "$missing" ]; then
    fail "Dependencias faltando:$missing"
    exit 1
  fi
}

run_baseline() {
  local label="$1"
  local n=${2:-$BASELINE_REQUESTS}
  local total_time=0
  local errors=0
  local min=9999
  local max=0

  for i in $(seq 1 "$n"); do
    local start=$(date +%s%3N 2>/dev/null || echo 0)
    local code
    code=$(curl -s -o /dev/null -w "%{http_code}" "$HEALTH_URL" 2>/dev/null)
    local end=$(date +%s%3N 2>/dev/null || echo 0)
    local elapsed=$((end - start))

    total_time=$((total_time + elapsed))
    [ "$code" != "200" ] && errors=$((errors + 1))
    [ "$elapsed" -lt "$min" ] && min=$elapsed
    [ "$elapsed" -gt "$max" ] && max=$elapsed
  done

  local avg=$((total_time / n))
  local err_rate=$(echo "scale=2; $errors * 100 / $n" | bc)

  echo "$label|$avg|$min|$max|$errors|$err_rate"
  return 0
}

do_healthcheck() {
  local code
  code=$(curl -s -o /dev/null -w "%{http_code}" --max-time 5 "$HEALTH_URL" 2>/dev/null || echo "000")
  if [ "$code" = "200" ]; then
    ok "API respondeu (200)"
    return 0
  else
    fail "API nao respondeu (codigo: $code)"
    return 1
  fi
}

cleanup_tc() {
  tc qdisc del dev "$INTERFACE" root 2>/dev/null || true
}

# --- Testes ---

test_cpu_stress() {
  header "CHAOS: CPU Stress (${CHAOS_DURATION}s)"
  command -v stress-ng &>/dev/null || { warn "stress-ng ausente — pulando"; return; }

  log "Baseline (sem stress)..."
  local baseline
  baseline=$(run_baseline "baseline_cpu")
  log "Baseline: $(echo "$baseline" | cut -d'|' -f2)ms avg"

  log "Injetando CPU stress (todos os cores)..."
  stress-ng --cpu 0 --timeout "${CHAOS_DURATION}s" --quiet &
  local stress_pid=$!

  sleep 5

  log "Medindo latencia sob CPU stress..."
  local under_stress
  under_stress=$(run_baseline "cpu_stress")
  log "CPU stress: $(echo "$under_stress" | cut -d'|' -f2)ms avg"

  wait "$stress_pid" 2>/dev/null || true
  sleep 5

  log "Recuperacao..."
  local recovered
  recovered=$(run_baseline "cpu_recovery")
  log "Recuperado: $(echo "$recovered" | cut -d'|' -f2)ms avg"

  do_healthcheck

  # Analise
  local base_avg stress_avg
  base_avg=$(echo "$baseline" | cut -d'|' -f2)
  stress_avg=$(echo "$under_stress" | cut -d'|' -f2)
  local degradation
  degradation=$(echo "scale=1; ($stress_avg - $base_avg) * 100 / $base_avg" | bc 2>/dev/null || echo "N/A")

  echo ""
  ok "CPU stress concluido — degradacao: ${degradation}% acima da baseline"
}

test_memory_stress() {
  header "CHAOS: Memory Stress (${CHAOS_DURATION}s)"
  command -v stress-ng &>/dev/null || { warn "stress-ng ausente — pulando"; return; }

  log "Baseline (sem stress)..."
  local baseline
  baseline=$(run_baseline "baseline_mem")

  log "Injetando memory stress (75% da RAM)..."
  stress-ng --vm 1 --vm-bytes 75% --timeout "${CHAOS_DURATION}s" --quiet &
  local stress_pid=$!

  sleep 5

  log "Medindo latencia sob memory stress..."
  local under_stress
  under_stress=$(run_baseline "mem_stress")

  wait "$stress_pid" 2>/dev/null || true
  sleep 5

  log "Recuperacao..."
  local recovered
  recovered=$(run_baseline "mem_recovery")

  do_healthcheck

  local base_avg stress_avg
  base_avg=$(echo "$baseline" | cut -d'|' -f2)
  stress_avg=$(echo "$under_stress" | cut -d'|' -f2)
  local degradation
  degradation=$(echo "scale=1; ($stress_avg - $base_avg) * 100 / $base_avg" | bc 2>/dev/null || echo "N/A")

  ok "Memory stress concluido — degradacao: ${degradation}%"
}

test_network_latency() {
  header "CHAOS: Network Latency (+200ms, ${CHAOS_DURATION}s)"
  command -v tc &>/dev/null || { warn "tc ausente — pulando"; return; }

  # Usar health diretamente no localhost para nao ser afetado pelo tc
  local direct_health="http://127.0.0.1:5070/health"

  log "Baseline..."
  local baseline
  baseline=$(run_baseline "baseline_net")

  log "Adicionando 200ms de latencia em $INTERFACE..."
  tc qdisc add dev "$INTERFACE" root netem delay 200ms 2>/dev/null || {
    warn "Nao foi possivel adicionar regra tc. Precisa de sudo?"
    return
  }
  trap cleanup_tc EXIT

  sleep 2

  log "Medindo com latencia..."
  local under_stress
  under_stress=$(run_baseline "net_latency")

  cleanup_tc
  trap - EXIT
  sleep 3

  log "Recuperacao..."
  local recovered
  recovered=$(run_baseline "net_recovery")

  do_healthcheck

  ok "Network latency concluido"
}

test_packet_loss() {
  header "CHAOS: Packet Loss (10%, ${CHAOS_DURATION}s)"
  command -v tc &>/dev/null || { warn "tc ausente — pulando"; return; }

  log "Baseline..."
  local baseline
  baseline=$(run_baseline "baseline_pktloss")

  log "Injetando 10%% de packet loss em $INTERFACE..."
  tc qdisc add dev "$INTERFACE" root netem loss 10% 2>/dev/null || {
    warn "Nao foi possivel adicionar regra tc"
    return
  }
  trap cleanup_tc EXIT

  sleep 2

  log "Medindo com packet loss..."
  local under_stress
  under_stress=$(run_baseline "pkt_loss")

  cleanup_tc
  trap - EXIT
  sleep 3

  log "Recuperacao..."
  local recovered
  recovered=$(run_baseline "pkt_recovery")

  do_healthcheck

  local base_err stress_err
  base_err=$(echo "$baseline" | cut -d'|' -f5)
  stress_err=$(echo "$under_stress" | cut -d'|' -f5)

  ok "Packet loss concluido — erros: $base_err (baseline) vs $stress_err (sob perda)"
}

# --- Relatorio ---

generate_report() {
  header "RELATORIO FINAL DE CHAOS ENGINEERING"
  {
    echo "Chaos Engineering Report"
    echo "========================="
    echo "Data: $(date)"
    echo "Alvo: $API_URL"
    echo "Duracao de cada teste: ${CHAOS_DURATION}s"
    echo ""
    echo "Resultados: ver acima"
    echo ""
    echo "Observacoes:"
    echo "- Se a API manteve 200 durante o stress: RESILIENTE"
    echo "- Se houve degradacao mas recuperou: TOLERANTE A FALHAS"
    echo "- Se caiu e nao voltou: FRAGIL — implementar circuit breaker / retry"
    echo ""
    echo "Recomendacoes:"
    echo "1. Adicionar rate limiting se ainda nao tem"
    echo "2. Configurar health checks no systemd (RestartSec)"
    echo "3. Usar connection pooling no banco de dados"
    echo "4. Considerar cache (Redis) para queries repetitivas"
    echo "5. Monitorar memoria com Prometheus (ja configurado)"
  } | tee "$REPORT_FILE"

  echo ""
  ok "Relatorio salvo em: $REPORT_FILE"
}

# --- Main ---

main() {
  echo ""
  echo "╔══════════════════════════════════════════════════════════╗"
  echo "║        CHAOS ENGINEERING — VENTURERP API                ║"
  echo "╚══════════════════════════════════════════════════════════╝"
  echo ""

  check_prereqs

  log "Verificando se a API esta online..."
  if ! do_healthcheck; then
    fail "API offline — abortando"
    exit 1
  fi

  test_cpu_stress
  test_memory_stress
  test_network_latency
  test_packet_loss
  generate_report

  echo ""
  ok "Todos os testes de chaos concluidos!"
}

main "$@"
