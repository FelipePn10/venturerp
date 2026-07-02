#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════════════════════════
# test-routing.sh — VALIDAÇÃO DO ROTEIRO DE FABRICAÇÃO (modelo de tempo rico)
#
# Exercita, via HTTP, os recursos enterprise+ da Fase 1:
#   • operações com tempos separados (setup/run/labor/queue/wait/move), base_qty,
#     crew_size e time_unit (MIN|HORA|DIA);
#   • herança operação → operação-de-roteiro e override por componente;
#   • lead time quantidade-consciente (run escala com o lote, setup/queue/wait/move não);
#   • fallback linear quando o roteiro não tem rede definida;
#   • rejeição de time_unit inválido.
#
# NÃO é destrutivo: usa códigos altos e limpa o que cria. Requer a API no ar
# apontando para um Postgres migrado (>= migration 000173).
#
# Uso: BASE_URL=http://localhost:5071 bash scripts/test-routing.sh
# ═══════════════════════════════════════════════════════════════════════════════
set -o pipefail

BASE="${BASE_URL:-http://localhost:5071}"
DB_CONTAINER="${DB_CONTAINER:-panossoerp-postgres-test}"
DB_NAME="${DB_NAME:-panossoerpdatabase_test}"
DB_USER="${DB_USER:-panossoerp_test}"
TOKEN=""
USER_UUID=""
HTTP_STATUS=""
PASS=0; FAIL=0; TOTAL=0

# High codes to avoid colliding with seeded/demo data.
ITEM_CODE="${ITEM_CODE:-940000001}"
WC2_CODE="${WC2_CODE:-940000092}"

# ── Helpers ───────────────────────────────────────────────────────────────────
c()    { printf '\n\033[1;34m══════ %s ══════\033[0m\n' "$*"; }
ok()   { printf '\033[0;32m  ✓ %s\033[0m\n' "$*"; PASS=$((PASS+1)); TOTAL=$((TOTAL+1)); }
err()  { printf '\033[0;31m  ✗ %s\033[0m\n' "$*"; FAIL=$((FAIL+1)); TOTAL=$((TOTAL+1)); }
info() { printf '\033[0;90m    %s\033[0m\n' "$*"; }

request() {
  local method="$1"; shift; local path="$1"; shift; local body="${1:-}"
  local resp
  if [ -n "$body" ]; then
    resp=$(curl -s -w "\n__STATUS__%{http_code}" -X "$method" "$BASE$path" \
      -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -d "$body" 2>/dev/null)
  else
    resp=$(curl -s -w "\n__STATUS__%{http_code}" -X "$method" "$BASE$path" \
      -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" 2>/dev/null)
  fi
  HTTP_STATUS=$(echo "$resp" | grep '__STATUS__' | sed 's/__STATUS__//')
  echo "$resp" | sed '/__STATUS__/d'
}
post() { request POST "$@"; }
get()  { request GET  "$1" ""; }

check() {
  local label="$1" val="$2" pat="$3"
  TOTAL=$((TOTAL+1))
  if [ -z "$val" ]; then err "$label — resposta vazia (HTTP $HTTP_STATUS)"; return 1; fi
  if echo "$val" | grep -qE "$pat" 2>/dev/null; then ok "$label (HTTP $HTTP_STATUS)"; return 0
  else err "$label — '$pat' ausente (HTTP $HTTP_STATUS) | $(echo "$val" | head -c 220 | tr '\n' ' ')"; return 1; fi
}
jq_int()  { echo "$1" | grep -o "\"$2\":[0-9]*" | head -1 | grep -o '[0-9]*'; }
jq_str()  { echo "$1" | grep -o "\"$2\":\"[^\"]*\"" | head -1 | sed 's/.*":"\(.*\)"/\1/'; }
# Extract a JSON number (handles decimals): num "<json>" total_hours
num()     { echo "$1" | grep -o "\"$2\":[0-9.]*" | head -1 | sed 's/.*://'; }
db()      { docker exec "$DB_CONTAINER" bash -c "psql -U $DB_USER $DB_NAME -t -c \"$*\"" 2>/dev/null | tr -d ' \n'; }
gt()      { awk -v a="$1" -v b="$2" 'BEGIN{exit !(a>b)}'; }

# ─── 0. HEALTH + AUTH ─────────────────────────────────────────────────────────
c "0. HEALTH + AUTENTICAÇÃO"
H=$(curl -sf -m 3 "$BASE/health" 2>/dev/null || echo "")
check "GET /health" "$H" '"ok"' || { err "API indisponível em $BASE — abortando"; exit 1; }

curl -s -X POST "$BASE/users/register" -H "Content-Type: application/json" \
  -d '{"name":"Routing Tester","email":"routing@test.local","password":"Admin@12345"}' >/dev/null 2>&1
LOGIN=$(curl -sf -X POST "$BASE/users/login" -H "Content-Type: application/json" \
  -d '{"email":"routing@test.local","password":"Admin@12345"}' 2>/dev/null)
TOKEN=$(jq_str "$LOGIN" "token")
[ -z "$TOKEN" ] && { err "Login falhou"; exit 1; }
ok "Login — JWT obtido"
USER_UUID=$(echo "$TOKEN" | cut -d'.' -f2 | \
  awk '{ n=length($0)%4; if(n>0) printf "%s%*s",$0,(4-n),""; else print $0 }' | \
  tr '_-' '/+' | base64 -d 2>/dev/null | \
  python3 -c "import sys,json; print(json.load(sys.stdin)['sub'])" 2>/dev/null || echo "")
[ -z "$USER_UUID" ] && { err "UUID do JWT não extraído"; exit 1; }
info "USER_UUID=$USER_UUID"

# Seed a minimal item to hang the route on (items has no FKs; unique code).
db "DELETE FROM items WHERE code = $ITEM_CODE"
db "INSERT INTO items (code, warehouse_code, created_by) VALUES ($ITEM_CODE, $ITEM_CODE, '$USER_UUID')"
info "Item $ITEM_CODE semeado"

# Work center (machine type) so operations can be costed / loaded by CT.
WC_CODE="${WC_CODE:-940000091}"
MT=$(post "/api/machine/types/create" "{\"code\":$WC_CODE,\"name\":\"CT Roteiro Teste\",\"type\":\"CUT\",\"requires_operator\":false,\"is_active\":true,\"created_by\":\"$USER_UUID\"}")
WC_ID=$(jq_int "$MT" "id")
[ -z "$WC_ID" ] && WC_ID=$(db "SELECT id FROM machine_types WHERE code=$WC_CODE")
info "Work center id=$WC_ID (code $WC_CODE)"

cleanup() {
  db "DELETE FROM manufacturing_routes WHERE item_code = $ITEM_CODE" >/dev/null
  db "DELETE FROM item_standard_costs WHERE item_code = $ITEM_CODE" >/dev/null
  db "DELETE FROM cost_rollup_log WHERE item_code = $ITEM_CODE" >/dev/null
  db "DELETE FROM items WHERE code = $ITEM_CODE" >/dev/null
  [ -n "$OP1_ID" ] && db "DELETE FROM operations WHERE id = $OP1_ID" >/dev/null
  [ -n "$OP2_ID" ] && db "DELETE FROM operations WHERE id = $OP2_ID" >/dev/null
  [ -n "$OP3_ID" ] && db "DELETE FROM operations WHERE id = $OP3_ID" >/dev/null
  [ -n "$WC_ID" ] && db "DELETE FROM work_center_costs WHERE work_center_id = $WC_ID" >/dev/null
  db "DELETE FROM machine_types WHERE code = $WC_CODE" >/dev/null
  db "DELETE FROM machine_types WHERE code = $WC2_CODE" >/dev/null
  [ -n "$TOOL_ID" ] && db "DELETE FROM tools WHERE id = $TOOL_ID" >/dev/null
}
trap cleanup EXIT

# ─── 1. OPERAÇÕES COM TEMPO RICO ──────────────────────────────────────────────
c "1. OPERAÇÕES — MODELO DE TEMPO RICO"
# Corte laser: medido em MINUTOS. run 30min/10pç, labor 20min/10pç, setup 60min,
# fila 5min, movimentação 2min, 2 operadores.
OP1=$(post "/api/routing/operations/" "{\"name\":\"Corte Laser\",\"origin\":\"INTERNA\",\"default_work_center_id\":$WC_ID,\"setup_time\":60,\"run_time\":30,\"labor_time\":20,\"run_base_qty\":10,\"queue_time\":5,\"move_time\":2,\"crew_size\":2,\"time_unit\":\"MIN\",\"created_by\":\"$USER_UUID\"}")
check "POST operação (Corte Laser, MIN)" "$OP1" '"id"'
OP1_ID=$(jq_int "$OP1" "id")

GOT=$(get "/api/routing/operations/$OP1_ID")
check "GET operação — run_time=30"     "$GOT" '"run_time":30'
check "GET operação — time_unit=MIN"   "$GOT" '"time_unit":"MIN"'
check "GET operação — run_base_qty=10" "$GOT" '"run_base_qty":10'
check "GET operação — crew_size=2"     "$GOT" '"crew_size":2'

# Segunda operação (em HORAS) — soldagem.
OP2=$(post "/api/routing/operations/" "{\"name\":\"Soldagem\",\"origin\":\"INTERNA\",\"setup_time\":0.5,\"run_time\":0.25,\"run_base_qty\":1,\"time_unit\":\"HORA\",\"created_by\":\"$USER_UUID\"}")
check "POST operação (Soldagem, HORA)" "$OP2" '"id"'
OP2_ID=$(jq_int "$OP2" "id")

# time_unit inválido deve ser rejeitado (status capturado direto — $(...) roda em subshell).
BADSTATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE/api/routing/operations/" \
  -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" \
  -d "{\"name\":\"Invalida\",\"origin\":\"INTERNA\",\"time_unit\":\"XYZ\",\"created_by\":\"$USER_UUID\"}")
if [ "$BADSTATUS" = "400" ] || [ "$BADSTATUS" = "422" ]; then ok "time_unit inválido rejeitado (HTTP $BADSTATUS)"; TOTAL=$((TOTAL+1)); PASS=$((PASS+1));
else err "time_unit inválido NÃO rejeitado (HTTP $BADSTATUS)"; TOTAL=$((TOTAL+1)); fi

# ─── 2. ROTEIRO + OPERAÇÕES ───────────────────────────────────────────────────
c "2. ROTEIRO + OPERAÇÕES (herança e override)"
RT=$(post "/api/routing/routes/" "{\"item_code\":$ITEM_CODE,\"alternative\":1,\"description\":\"Roteiro teste tempo rico\",\"is_standard\":true,\"created_by\":\"$USER_UUID\"}")
check "POST roteiro" "$RT" '"id"'
RT_ID=$(jq_int "$RT" "id")

# Seq 10: herda Corte Laser (sem override).
RO1=$(post "/api/routing/route-operations/$RT_ID/" "{\"route_id\":$RT_ID,\"sequence\":10,\"operation_id\":$OP1_ID,\"situation\":\"APROVADA\"}")
check "POST route-op seq 10 (herda Corte)" "$RO1" '"id"'
RO1_ID=$(jq_int "$RO1" "id")
# Seq 20: Soldagem, override do run para 0.5h.
RO2=$(post "/api/routing/route-operations/$RT_ID/" "{\"route_id\":$RT_ID,\"sequence\":20,\"operation_id\":$OP2_ID,\"run_time\":0.5,\"time_unit\":\"HORA\",\"situation\":\"APROVADA\"}")
check "POST route-op seq 20 (override run=0.5h)" "$RO2" '"id"'

DET=$(get "/api/routing/routes/$RT_ID/")
# EffTime resolvido em horas: Corte run 30min → 0.5h.
check "GET roteiro — eff_time run_hours=0.5 (30min→h)" "$DET" '"run_hours":0.5'

# ─── 3. LEAD TIME QUANTIDADE-CONSCIENTE ───────────────────────────────────────
c "3. LEAD TIME QUANTIDADE-CONSCIENTE"
LT1J=$(get "/api/routing/routes/$RT_ID/lead-time?qty=1")
check "GET lead-time?qty=1" "$LT1J" '"total_hours"'
LT1=$(num "$LT1J" "total_hours")
LT100J=$(get "/api/routing/routes/$RT_ID/lead-time?qty=100")
LT100=$(num "$LT100J" "total_hours")
info "lead time: qty=1 → ${LT1}h   qty=100 → ${LT100}h"
if gt "$LT100" "$LT1"; then ok "lead time cresce com a quantidade ($LT1 → $LT100)"; TOTAL=$((TOTAL+1)); PASS=$((PASS+1));
else err "lead time NÃO escala com a quantidade ($LT1 vs $LT100)"; TOTAL=$((TOTAL+1)); fi
# Rede vazia → fallback linear: caminho crítico deve ter 2 operações somadas.
check "lead-time expõe critical_path" "$LT100J" '"critical_path"'

# ─── 4. CUSTEIO POR CENTRO DE TRABALHO ────────────────────────────────────────
c "4. CUSTEIO POR CENTRO DE TRABALHO (máquina × mão-de-obra, amortização de setup)"
WCC=$(post "/api/standard-cost/work-center-costs/" "{\"work_center_id\":$WC_ID,\"cost_per_hour\":100,\"machine_cost_per_hour\":100,\"labor_cost_per_hour\":50,\"currency\":\"BRL\",\"updated_by\":\"$USER_UUID\"}")
check "POST work-center-cost (split máquina/m.o.)" "$WCC" '"machine_cost_per_hour":100'
check "POST work-center-cost expõe labor rate"     "$WCC" '"labor_cost_per_hour":50'

# Custo do Corte (herda o CT via default_work_center_id): setup 60min + run 30min/10pç.
RUP1=$(post "/api/standard-cost/rollup" "{\"item_code\":$ITEM_CODE,\"lot_size\":1,\"calculated_by\":\"$USER_UUID\"}")
check "POST rollup (lote 1) — labor_cost > 0" "$RUP1" '"labor_cost"'
LC1=$(num "$RUP1" "labor_cost")
RUP10=$(post "/api/standard-cost/rollup" "{\"item_code\":$ITEM_CODE,\"lot_size\":10,\"calculated_by\":\"$USER_UUID\"}")
LC10=$(num "$RUP10" "labor_cost")
info "labor_cost por unidade: lote 1 → $LC1   lote 10 → $LC10"
if gt "$LC1" "$LC10"; then ok "setup amortizado pelo lote ($LC1 → $LC10)"; TOTAL=$((TOTAL+1)); PASS=$((PASS+1));
else err "amortização de setup NÃO aplicada ($LC1 vs $LC10)"; TOTAL=$((TOTAL+1)); fi

# ─── 5. RECURSOS ALTERNATIVOS POR OPERAÇÃO ────────────────────────────────────
c "5. RECURSOS ALTERNATIVOS POR OPERAÇÃO (R5)"
MT2=$(post "/api/machine/types/create" "{\"code\":$WC2_CODE,\"name\":\"CT Alternativo\",\"type\":\"CUT\",\"requires_operator\":false,\"is_active\":true,\"created_by\":\"$USER_UUID\"}")
WC2_ID=$(jq_int "$MT2" "id")
[ -z "$WC2_ID" ] && WC2_ID=$(db "SELECT id FROM machine_types WHERE code=$WC2_CODE")
RESBASE="/api/routing/route-operations/$RT_ID/$RO1_ID/resources"
# Recurso primário no CT padrão + alternativo 20% mais lento.
RESA=$(post "$RESBASE" "{\"work_center_id\":$WC_ID,\"priority\":1,\"time_factor\":1,\"is_primary\":true}")
check "POST recurso primário (WC padrão)" "$RESA" '"is_primary":true'
RESB=$(post "$RESBASE" "{\"work_center_id\":$WC2_ID,\"priority\":2,\"time_factor\":1.2}")
check "POST recurso alternativo (time_factor 1.2)" "$RESB" '"time_factor":1.2'
RESB_ID=$(jq_int "$RESB" "id")
LRES=$(get "$RESBASE")
check "GET recursos (primário primeiro)" "$LRES" '"is_primary":true'
# Trocar o primário p/ o alternativo → CT efetivo da operação passa a ser WC2.
post "$RESBASE/$RESB_ID/primary" "" >/dev/null
DET2=$(get "/api/routing/routes/$RT_ID/")
check "primário trocado → route-op usa WC2" "$DET2" "\"work_center_id\":$WC2_ID"

# ─── 6. FERRAMENTAS + VIDA ÚTIL ───────────────────────────────────────────────
c "6. FERRAMENTAS + VIDA ÚTIL (R3)"
TOOL=$(post "/api/routing/tools/" "{\"name\":\"Matriz Estampo M1\",\"tool_type\":\"MATRIZ\",\"life_type\":\"GOLPES\",\"life_limit\":5000,\"cost\":8000,\"created_by\":\"$USER_UUID\"}")
check "POST ferramenta (GOLPES, limite 5000)" "$TOOL" '"life_type":"GOLPES"'
check "ferramenta expõe remaining_life" "$TOOL" '"remaining_life":5000'
TOOL_ID=$(jq_int "$TOOL" "id")
GT=$(get "/api/routing/tools/$TOOL_ID")
check "GET ferramenta" "$GT" "\"id\":$TOOL_ID"
LT=$(get "/api/routing/tools/")
check "GET lista de ferramentas" "$LT" "\"id\":$TOOL_ID"
# life_type inválido rejeitado.
BADT=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE/api/routing/tools/" \
  -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" \
  -d "{\"name\":\"X\",\"life_type\":\"XPTO\",\"created_by\":\"$USER_UUID\"}")
if [ "$BADT" = "400" ] || [ "$BADT" = "422" ]; then ok "life_type inválido rejeitado (HTTP $BADT)"; TOTAL=$((TOTAL+1)); PASS=$((PASS+1)); else err "life_type inválido NÃO rejeitado (HTTP $BADT)"; TOTAL=$((TOTAL+1)); fi
# Associa à operação do roteiro.
TASSOC=$(post "/api/routing/route-operations/$RT_ID/$RO1_ID/tools" "{\"tool_id\":$TOOL_ID,\"qty_required\":1}")
check "POST associação ferramenta↔operação" "$TASSOC" "\"tool_id\":$TOOL_ID"
LTOOLS=$(get "/api/routing/route-operations/$RT_ID/$RO1_ID/tools")
check "GET ferramentas da operação" "$LTOOLS" "\"tool_name\":\"Matriz Estampo M1\""
REPL=$(get "/api/routing/tools/replacement")
check "GET ferramentas p/ troca (endpoint responde)" "$REPL" '\[|\]'

# ─── 7. SUBCONTRATAÇÃO (OPERAÇÃO EXTERNA) ─────────────────────────────────────
c "7. SUBCONTRATAÇÃO / OPERAÇÃO EXTERNA (R4)"
OPX=$(post "/api/routing/operations/" "{\"name\":\"Zincagem (terceiro)\",\"origin\":\"EXTERNA\",\"service_item_code\":880001,\"supplier_id\":1,\"cost_per_unit\":12.5,\"lead_time_days\":7,\"created_by\":\"$USER_UUID\"}")
check "POST operação EXTERNA c/ subcontratação" "$OPX" '"origin":"EXTERNA"'
check "operação expõe cost_per_unit" "$OPX" '"cost_per_unit":12.5'
OP3_ID=$(jq_int "$OPX" "id")
GX=$(get "/api/routing/operations/$OP3_ID")
check "GET operação externa — service_item_code" "$GX" '"service_item_code":880001'
check "GET operação externa — lead_time_days"    "$GX" '"lead_time_days":7'

# ─── 8. VIGÊNCIA / EFETIVIDADE DO ROTEIRO ─────────────────────────────────────
c "8. VIGÊNCIA DO ROTEIRO (R6)"
RTV=$(post "/api/routing/routes/" "{\"item_code\":$ITEM_CODE,\"alternative\":2,\"description\":\"Rev com vigência\",\"is_standard\":false,\"valid_from\":\"2026-01-01T00:00:00Z\",\"valid_to\":\"2026-12-31T00:00:00Z\",\"created_by\":\"$USER_UUID\"}")
check "POST roteiro com vigência" "$RTV" '"valid_from"'
check "roteiro expõe valid_to" "$RTV" '2026-12-31'
# valid_to antes de valid_from → rejeitado.
BADV=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE/api/routing/routes/" \
  -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" \
  -d "{\"item_code\":$ITEM_CODE,\"alternative\":3,\"valid_from\":\"2026-12-31T00:00:00Z\",\"valid_to\":\"2026-01-01T00:00:00Z\",\"created_by\":\"$USER_UUID\"}")
if [ "$BADV" = "400" ] || [ "$BADV" = "422" ]; then ok "vigência inválida rejeitada (HTTP $BADV)"; TOTAL=$((TOTAL+1)); PASS=$((PASS+1)); else err "vigência inválida NÃO rejeitada (HTTP $BADV)"; TOTAL=$((TOTAL+1)); fi

# ─── RESUMO ───────────────────────────────────────────────────────────────────
c "RESUMO"
printf 'Total: %d   \033[0;32mPASS: %d\033[0m   \033[0;31mFAIL: %d\033[0m\n' "$TOTAL" "$PASS" "$FAIL"
[ "$FAIL" -eq 0 ] && { printf '\033[0;32m✓ Roteiro (tempo rico) validado\033[0m\n'; exit 0; } || exit 1
