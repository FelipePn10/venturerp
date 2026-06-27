#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════════════════════════
# test-cutting.sh — PLANO DE CORTE (end-to-end)
# Cobre: Fase 1 (1D) · Fase 2 (firmar/baixa/retalhos + UoM) · Fase 3 (2D guilhotina
# + fita de borda) · Fase 4 (true-shape/raster) · Fase 5 (demanda de OP) ·
# complementos (export SVG/DXF/PDF, programa, agenda na máquina, rateio por OP) ·
# cenários negativos (peça sem encaixe, modo MANUAL sem lote) e reuso de retalho.
# Uso: BASE_URL=http://localhost:5071 bash scripts/test-cutting.sh
# ═══════════════════════════════════════════════════════════════════════════════
set -o pipefail

BASE="${BASE_URL:-http://localhost:5071}"
DB_CONTAINER="${DB_CONTAINER:-panossoerp-postgres-test}"
DB_NAME="${DB_NAME:-panossoerpdatabase_test}"
DB_USER="${DB_USER:-panossoerp_test}"
TOKEN=""; USER_UUID=""; HTTP_STATUS=""
PASS=0; FAIL=0; TOTAL=0; BUGS=()
# request() roda em subshell via $(...); o status é persistido em arquivo para o check ler.
CUT_STATUS_FILE="$(mktemp)"
trap 'rm -f "$CUT_STATUS_FILE"' EXIT

c()    { printf '\n\033[1;34m══════ %s ══════\033[0m\n' "$*"; }
ok()   { printf '\033[0;32m  ✓ %s\033[0m\n' "$*"; PASS=$((PASS+1)); TOTAL=$((TOTAL+1)); }
err()  { printf '\033[0;31m  ✗ %s\033[0m\n' "$*"; FAIL=$((FAIL+1)); TOTAL=$((TOTAL+1)); BUGS+=("$*"); }
info() { printf '\033[0;90m    %s\033[0m\n' "$*"; }
db()   { docker exec "$DB_CONTAINER" bash -c "psql -U $DB_USER $DB_NAME -t -c \"$*\"" 2>/dev/null | tr -d ' \n'; }

request() {
  local method="$1"; shift; local path="$1"; shift; local body="${1:-}"; local resp
  if [ -n "$body" ]; then
    resp=$(curl -s -w "\n__STATUS__%{http_code}" -X "$method" "$BASE$path" \
      -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" -d "$body" 2>/dev/null)
  else
    resp=$(curl -s -w "\n__STATUS__%{http_code}" -X "$method" "$BASE$path" \
      -H "Content-Type: application/json" -H "Authorization: Bearer $TOKEN" 2>/dev/null)
  fi
  HTTP_STATUS=$(echo "$resp" | grep '__STATUS__' | sed 's/__STATUS__//')
  echo "$HTTP_STATUS" > "$CUT_STATUS_FILE"
  echo "$resp" | sed '/__STATUS__/d'
}
post() { request POST "$@"; }
get()  { request GET  "$1" ""; }
put()  { request PUT  "$@"; }

# check: PASS exige HTTP 2xx/3xx E o padrão presente no corpo.
check() {
  local label="$1" val="$2" pat="$3"; local st; st=$(cat "$CUT_STATUS_FILE" 2>/dev/null); TOTAL=$((TOTAL+1))
  if [ -n "$st" ] && [ "$st" -ge 400 ] 2>/dev/null; then
    err "$label — HTTP $st | $(echo "$val" | head -c 200 | tr '\n' ' ')"; return; fi
  if [ -z "$val" ]; then err "$label — resposta vazia (HTTP $st)"; return; fi
  if echo "$val" | grep -qE "$pat" 2>/dev/null; then PASS=$((PASS+1)); printf '\033[0;32m  ✓ %s (HTTP %s)\033[0m\n' "$label" "$st"
  else err "$label — '$pat' ausente (HTTP $st) | $(echo "$val" | head -c 200 | tr '\n' ' ')"; fi
}
# check_fail: cenário negativo — espera HTTP >= 400 e (opcional) padrão no erro.
check_fail() {
  local label="$1" val="$2" pat="${3:-}"; local st; st=$(cat "$CUT_STATUS_FILE" 2>/dev/null); TOTAL=$((TOTAL+1))
  if [ -z "$st" ] || [ "$st" -lt 400 ] 2>/dev/null; then
    err "$label — esperava erro, veio HTTP $st | $(echo "$val" | head -c 160 | tr '\n' ' ')"; return; fi
  if [ -n "$pat" ] && ! echo "$val" | grep -qiE "$pat"; then
    err "$label — erro sem '$pat' (HTTP $st) | $(echo "$val" | head -c 160)"; return; fi
  PASS=$((PASS+1)); printf '\033[0;32m  ✓ %s (rejeitado HTTP %s)\033[0m\n' "$label" "$st"
}
jq_int() { echo "$1" | grep -oiE "\"$2\":[0-9]+" | head -1 | grep -oE '[0-9]+'; }
jq_str() { echo "$1" | grep -o "\"$2\":\"[^\"]*\"" | head -1 | sed 's/.*":"\(.*\)"/\1/'; }

# ─── 0. RESET ─────────────────────────────────────────────────────────────────
c "0. RESET (tabelas de corte + apoio)"
RESET_SQL="TRUNCATE TABLE cutting_plan_order_costs, cutting_plan_consumptions, stock_remnants,
  cutting_pattern_placements, cutting_patterns, cutting_stock_pieces, cutting_plan_parts,
  cutting_plans, machine_schedules, production_orders, item_structures,
  stock_movements, stock_balances, stock_lot_balances, stock_lots,
  items, machines, machine_types, warehouse, enterprise, users RESTART IDENTITY CASCADE;
  INSERT INTO cutting_settings (id) VALUES (1) ON CONFLICT (id) DO NOTHING;"
if docker exec "$DB_CONTAINER" bash -c "psql -U $DB_USER $DB_NAME -c \"$RESET_SQL\"" >/dev/null 2>&1; then
  ok "DB reset"
else
  info "Reset falhou/sem docker — seguindo (códigos podem colidir)"
fi
curl -sf "$BASE/health" >/dev/null 2>&1 && ok "API online" || { err "API offline em $BASE"; exit 1; }

# ─── 1. LOGIN (com promoção a ADMIN p/ editar settings) ───────────────────────
c "1. AUTENTICAÇÃO"
curl -s -X POST "$BASE/users/register" -H "Content-Type: application/json" \
  -d '{"name":"Admin Corte","email":"corte@test.local","password":"Admin@12345"}' >/dev/null 2>&1
db "UPDATE users SET role='ADMIN' WHERE email='corte@test.local';" >/dev/null
LOGIN=$(curl -sf -X POST "$BASE/users/login" -H "Content-Type: application/json" \
  -d '{"email":"corte@test.local","password":"Admin@12345"}' 2>/dev/null)
TOKEN=$(jq_str "$LOGIN" "token"); [ -z "$TOKEN" ] && { err "login falhou"; exit 1; }
ok "login OK (ADMIN)"
USER_UUID=$(echo "$TOKEN" | cut -d'.' -f2 | awk '{n=length($0)%4; if(n>0) printf "%s%*s",$0,(4-n),""; else print $0}' \
  | tr '_-' '/+' | base64 -d 2>/dev/null | python3 -c "import sys,json; print(json.load(sys.stdin)['sub'])" 2>/dev/null || echo "")
info "uuid=$USER_UUID"

# ─── 2. SETUP ─────────────────────────────────────────────────────────────────
c "2. CADASTROS DE APOIO (empresa, depósito, máquina, itens, BOM)"
post "/api/enterprise/create" "{\"code\":1,\"name\":\"Metalmadeira Ltda\",\"created_by\":\"$USER_UUID\"}" >/dev/null
post "/api/pdm/create-group" "{\"code\":30,\"description\":\"Materia Prima\",\"enterprise_id\":1,\"created_by\":\"$USER_UUID\"}" >/dev/null
WH=$(post "/api/warehouse/create" "{\"code\":1,\"description\":\"Almoxarifado MP\",\"location\":0,\"type\":1,\"disposition\":true,\"reservations_allowed\":true,\"created_by\":\"$USER_UUID\"}")
WH_ID=$(jq_int "$WH" "id"); [ -z "$WH_ID" ] && WH_ID=1
check "warehouse" "$WH" 'id|code'
post "/api/machine/types/create" "{\"code\":2,\"name\":\"Seccionadora\",\"type\":\"CUT\",\"requires_operator\":false,\"is_active\":true,\"created_by\":\"$USER_UUID\"}" >/dev/null
M=$(post "/api/machine/create" "{\"code\":1002,\"name\":\"Seccionadora-01\",\"machine_type_code\":2,\"capacity\":16.0,\"capacity_per_unit\":\"UN\",\"capacity_period\":\"DIA\",\"efficiency_rate\":0.95,\"is_active\":true,\"created_by\":\"$USER_UUID\"}")
check "máquina 1002 (Seccionadora)" "$M" 'id|1002|name'

mkitem() { # code uom llc desc dims_extra
  post "/api/items/create" "{\"code\":$1,\"nature\":2,
    \"pdm\":{\"group_code\":30,\"modifier_code\":0,\"attributes\":[],\"description_technique\":\"$4\"},
    \"situation\":0,\"health\":\"ATIVO\",
    \"warehouse\":{\"warehouse_code\":1,\"unit_of_measurement\":\"$2\",\"automatic_low\":false,\"minimum_stock\":0},
    \"engineering\":{\"weight\":{\"gross\":1,\"net\":1,\"unit\":\"KG\"},\"type\":0,\"type_struct\":0,\"oem\":false$5},
    \"planning\":{\"type_mrp\":0,\"llc\":$3,\"ghost\":false},
    \"supplies\":{\"type_of_use\":0},\"created_by\":\"$USER_UUID\"}"
}
check "item 60001 (barra M/LLC9)"  "$(mkitem 60001 "M"  9 "Barra cantoneira 2pol 6m" "")" '60001|id'
check "item 60002 (chapa M2/LLC9)" "$(mkitem 60002 "M2" 9 "Chapa MDF 15mm 2750x1830" "")" '60002|id'
check "item 60100 (produto)"       "$(mkitem 60100 "UN" 0 "Mesa MDF" "")" '60100|id'
check "item 60110 (perna 720, dims)" "$(mkitem 60110 "UN" 1 "Perna 720mm" ",\"dimensions\":{\"length\":720,\"width\":40,\"height\":40}")" '60110|id'
post "/api/items/structure/create" "{\"parent_code\":60100,\"child_code\":60110,\"quantity\":4.0,\"unit_of_measurement\":\"UN\",\"health\":\"ATIVO\",\"loss_percentage\":0.0,\"sequence\":1,\"is_active\":true,\"inherit\":false,\"created_by\":\"$USER_UUID\"}" >/dev/null
S=$(post "/api/items/structure/create" "{\"parent_code\":60110,\"child_code\":60001,\"quantity\":1.0,\"unit_of_measurement\":\"M\",\"health\":\"ATIVO\",\"loss_percentage\":0.0,\"sequence\":1,\"is_active\":true,\"inherit\":false,\"created_by\":\"$USER_UUID\"}")
check "BOM 60100←60110←60001" "$S" 'parent_code|ParentCode|id|ID'

# ─── 3. SETTINGS ──────────────────────────────────────────────────────────────
c "3. CONFIGURAÇÃO DA EMPRESA (cutting-settings)"
check "PUT settings (AUTOMATIC)" "$(put /api/cutting-settings "{\"default_consumption_mode\":\"AUTOMATIC\",\"default_min_remnant_mm\":300,\"default_warehouse_id\":$WH_ID}")" 'AUTOMATIC'
check "GET settings" "$(get /api/cutting-settings)" 'default_consumption_mode|AUTOMATIC'
check_fail "PUT settings inválido (rejeita)" "$(put /api/cutting-settings '{"default_consumption_mode":"XYZ"}')" "invalid"

# ─── 4. FASE 1 — 1D ───────────────────────────────────────────────────────────
c "4. FASE 1 — corte linear 1D (barra em metros)"
P1=$(post "/api/cutting-plans" "{\"material_item_code\":60001,\"cut_type\":\"LINEAR_1D\",\"stock_uom\":\"M\",\"kerf_mm\":3,\"trim_mm\":0,\"min_remnant_mm\":300,\"warehouse_id\":$WH_ID,
  \"parts\":[{\"label\":\"Perna 720\",\"length_mm\":720,\"quantity\":6},{\"label\":\"Travessa 1200\",\"length_mm\":1200,\"quantity\":3}],
  \"stock_pieces\":[{\"length_mm\":6000,\"quantity\":3}],\"created_by\":\"$USER_UUID\"}")
P1_ID=$(jq_int "$P1" "id"); check "cria plano 1D" "$P1" '"id":[0-9]'
check "otimiza 1D" "$(post /api/cutting-plans/$P1_ID/optimize '')" 'patterns|utilization_pct'

# Cenário negativo: peça maior que qualquer estoque → unplaced
PU=$(post "/api/cutting-plans" "{\"material_item_code\":60001,\"cut_type\":\"LINEAR_1D\",\"stock_uom\":\"M\",\"warehouse_id\":$WH_ID,
  \"parts\":[{\"label\":\"Gigante\",\"length_mm\":7000,\"quantity\":1}],\"stock_pieces\":[{\"length_mm\":6000,\"quantity\":1}],\"created_by\":\"$USER_UUID\"}")
PU_ID=$(jq_int "$PU" "id")
check "otimiza com peça grande → unplaced" "$(post /api/cutting-plans/$PU_ID/optimize '')" 'unplaced'

# ─── 5. FASE 2 — FIRMAR + RETALHO + UoM + MODO MANUAL ─────────────────────────
c "5. FASE 2 — firmar (baixa em metros) + retalho + modo manual"
REL1=$(post "/api/cutting-plans/$P1_ID/release" "")
check "firmar 1D (baixa)" "$REL1" 'FIRMADO|bars_consumed'
check "retalho gerado" "$(get /api/stock-remnants?item_code=60001&only_available=true)" 'length_mm|status'

# Cenário negativo: modo MANUAL sem lote na peça de estoque → rejeita ao firmar
PM=$(post "/api/cutting-plans" "{\"material_item_code\":60001,\"cut_type\":\"LINEAR_1D\",\"stock_uom\":\"M\",\"lot_consumption_mode\":\"MANUAL\",\"warehouse_id\":$WH_ID,
  \"parts\":[{\"label\":\"P\",\"length_mm\":1000,\"quantity\":2}],\"stock_pieces\":[{\"length_mm\":6000,\"quantity\":1}],\"created_by\":\"$USER_UUID\"}")
PM_ID=$(jq_int "$PM" "id"); post "/api/cutting-plans/$PM_ID/optimize" "" >/dev/null
check_fail "firmar MANUAL sem lote (rejeita)" "$(post /api/cutting-plans/$PM_ID/release '')" 'manual|lot'

# Reuso de retalho: novo plano com include_remnants usa a sobra do P1
PR=$(post "/api/cutting-plans" "{\"material_item_code\":60001,\"cut_type\":\"LINEAR_1D\",\"stock_uom\":\"M\",\"min_remnant_mm\":300,\"include_remnants\":true,\"warehouse_id\":$WH_ID,
  \"parts\":[{\"label\":\"Curta\",\"length_mm\":500,\"quantity\":2}],\"stock_pieces\":[{\"length_mm\":6000,\"quantity\":1}],\"created_by\":\"$USER_UUID\"}")
PR_ID=$(jq_int "$PR" "id")
check "otimiza com include_remnants" "$(post /api/cutting-plans/$PR_ID/optimize '')" 'patterns|utilization_pct'

# ─── 6. FASE 3 — 2D + FITA DE BORDA ───────────────────────────────────────────
c "6. FASE 3 — corte 2D de chapa (m²) + fita de borda"
P2=$(post "/api/cutting-plans" "{\"material_item_code\":60002,\"cut_type\":\"GUILLOTINE_2D\",\"stock_uom\":\"M2\",\"kerf_mm\":4,\"trim_mm\":10,\"min_remnant_mm\":200,\"warehouse_id\":$WH_ID,
  \"parts\":[{\"label\":\"Lateral\",\"width_mm\":600,\"height_mm\":700,\"quantity\":4,\"grain\":\"LENGTH\",\"edge_top\":true,\"edge_left\":true,\"band_cost_per_m\":2.5},
            {\"label\":\"Prateleira\",\"width_mm\":564,\"height_mm\":300,\"quantity\":6,\"allow_rotation\":true}],
  \"stock_pieces\":[{\"width_mm\":2750,\"height_mm\":1830,\"quantity\":4}],\"created_by\":\"$USER_UUID\"}")
P2_ID=$(jq_int "$P2" "id"); check "cria plano 2D" "$P2" '"id":[0-9]'
check "otimiza 2D (posições)" "$(post /api/cutting-plans/$P2_ID/optimize '')" 'pos_x_mm|used_area_mm2'
check "detalhe traz fita (banding)" "$(get /api/cutting-plans/$P2_ID)" 'banding_length_mm|"banding"'
check "firmar 2D (baixa por área)" "$(post /api/cutting-plans/$P2_ID/release '')" 'FIRMADO'

# ─── 7. FASE 4 — TRUE-SHAPE (raster) ──────────────────────────────────────────
c "7. FASE 4 — corte true-shape (contorno irregular, raster)"
P3=$(post "/api/cutting-plans" "{\"material_item_code\":60002,\"cut_type\":\"TRUE_SHAPE_2D\",\"stock_uom\":\"M2\",\"kerf_mm\":1,\"warehouse_id\":$WH_ID,
  \"parts\":[{\"label\":\"Flange L\",\"allow_rotation\":true,\"quantity\":2,\"geometry\":[{\"x\":0,\"y\":0},{\"x\":400,\"y\":0},{\"x\":400,\"y\":200},{\"x\":200,\"y\":200},{\"x\":200,\"y\":400},{\"x\":0,\"y\":400}]}],
  \"stock_pieces\":[{\"width_mm\":1000,\"height_mm\":1000,\"quantity\":2}],\"created_by\":\"$USER_UUID\"}")
P3_ID=$(jq_int "$P3" "id"); check "cria plano true-shape" "$P3" '"id":[0-9]'
check "otimiza true-shape (posições)" "$(post /api/cutting-plans/$P3_ID/optimize '')" 'patterns|pos_x_mm|stock_width_mm'

# ─── 8. COMPLEMENTOS — export / programa / agenda ─────────────────────────────
c "8. COMPLEMENTOS — mapa (SVG/DXF/PDF), programa, agenda na máquina"
check "export SVG" "$(get /api/cutting-plans/$P2_ID/export?format=svg)" '<svg|<rect'
check "export DXF" "$(get /api/cutting-plans/$P2_ID/export?format=dxf)" 'LWPOLYLINE|EOF'
check "export PDF" "$(get /api/cutting-plans/$P2_ID/export?format=pdf)" '%PDF'
check "programa de corte" "$(get /api/cutting-plans/$P2_ID/program)" 'patterns|steps|sequence'
# agenda: plano 1D com máquina
PMc=$(post "/api/cutting-plans" "{\"material_item_code\":60001,\"cut_type\":\"LINEAR_1D\",\"stock_uom\":\"M\",\"machine_code\":1002,\"warehouse_id\":$WH_ID,
  \"parts\":[{\"label\":\"P\",\"length_mm\":1000,\"quantity\":2}],\"stock_pieces\":[{\"length_mm\":6000,\"quantity\":1}],\"created_by\":\"$USER_UUID\"}")
PMc_ID=$(jq_int "$PMc" "id"); post "/api/cutting-plans/$PMc_ID/optimize" "" >/dev/null
check "agenda corte na máquina" "$(post /api/cutting-plans/$PMc_ID/schedule '')" 'schedule_code|machine_code|planned_pieces'

# ─── 9. FASE 5 — DEMANDA DE OP + RATEIO POR OP ────────────────────────────────
c "9. FASE 5 — demanda automática a partir de OP (+ rateio de custo por OP)"
O1=$(post "/api/production-order/create" "{\"item_code\":60100,\"mask\":\"\",\"planned_qty\":2.0,\"start_date\":\"2026-07-01\",\"end_date\":\"2026-07-05\",\"created_by\":\"$USER_UUID\"}")
O2=$(post "/api/production-order/create" "{\"item_code\":60100,\"mask\":\"\",\"planned_qty\":1.0,\"start_date\":\"2026-07-01\",\"end_date\":\"2026-07-05\",\"created_by\":\"$USER_UUID\"}")
O1_ID=$(jq_int "$O1" "ID"); O2_ID=$(jq_int "$O2" "ID")
check "cria OPs 60100" "$O1" 'ItemCode|60100'
info "OP ids: $O1_ID, $O2_ID"
GEN=$(post "/api/cutting-plans/from-orders" "{\"production_order_codes\":[$O1_ID,$O2_ID],\"kerf_mm\":3,\"min_remnant_mm\":300,\"warehouse_id\":$WH_ID,\"created_by\":\"$USER_UUID\"}")
check "gera plano a partir das OPs (material 60001)" "$GEN" '60001|material_item_code'
check "agrega ordens (peças/refs)" "$GEN" 'total_pieces|order_refs|OP-'
GEN_PID=$(jq_int "$GEN" "plan_id")
if [ -n "$GEN_PID" ]; then
  post "/api/cutting-plans/$GEN_PID/stock" "{\"length_mm\":6000,\"quantity\":3}" >/dev/null
  post "/api/cutting-plans/$GEN_PID/optimize" "" >/dev/null
  check "firmar plano agregado" "$(post /api/cutting-plans/$GEN_PID/release '')" 'FIRMADO'
  check "rateio de custo por OP (2 ordens)" "$(get /api/cutting-plans/$GEN_PID/order-costs)" 'order_ref|allocated_cost|OP-'
fi

# ─── 10. COMPLEMENTOS — árvore de cortes (seccionadora) + fita no custeio da OP ─
c "10. Árvore de cortes guilhotinados + fita de borda no custeio da OP"
# 10a — o programa do plano 2D traz a árvore de cortes guilhotinados (axis/level/pos)
check "programa 2D traz árvore de cortes (axis)" "$(get /api/cutting-plans/$P2_ID/program)" '"cuts"|"axis"|VERTICAL|HORIZONTAL'
# 10b — fita de borda entra no rateio por OP: 2 peças de MESMA área, só uma com fita →
#       a OP da peça encapada deve custar MAIS (material igual + fita direta).
PFB=$(post "/api/cutting-plans" "{\"material_item_code\":60002,\"cut_type\":\"GUILLOTINE_2D\",\"stock_uom\":\"M2\",\"warehouse_id\":$WH_ID,\"created_by\":\"$USER_UUID\",
  \"parts\":[
    {\"label\":\"Com fita\",\"width_mm\":600,\"height_mm\":400,\"quantity\":2,\"source_ref\":\"OP-A\",\"edge_top\":true,\"edge_bottom\":true,\"band_cost_per_m\":5},
    {\"label\":\"Sem fita\",\"width_mm\":600,\"height_mm\":400,\"quantity\":2,\"source_ref\":\"OP-B\"}],
  \"stock_pieces\":[{\"width_mm\":2440,\"height_mm\":1220,\"quantity\":2}]}")
PFB_ID=$(jq_int "$PFB" "id")
check "cria plano com fita + source_ref" "$PFB" '"id":[0-9]'
post "/api/cutting-plans/$PFB_ID/optimize" "" >/dev/null
check "firma plano com fita" "$(post /api/cutting-plans/$PFB_ID/release '')" 'FIRMADO'
CMP=$(get /api/cutting-plans/$PFB_ID/order-costs | python3 -c "
import sys,json
raw=json.load(sys.stdin)
items=raw if isinstance(raw,list) else raw.get('order_costs',[])
m={c.get('order_ref'):c.get('allocated_cost',0) for c in items}
print('OP-A=%.4f OP-B=%.4f'%(m.get('OP-A',0),m.get('OP-B',0)))" 2>/dev/null)
A=$(echo "$CMP" | sed -n 's/.*OP-A=\([0-9.]*\).*/\1/p'); B=$(echo "$CMP" | sed -n 's/.*OP-B=\([0-9.]*\).*/\1/p')
if [ -n "$A" ] && python3 -c "import sys;sys.exit(0 if float('$A')>float('${B:-0}') else 1)" 2>/dev/null; then
  ok "fita de borda entra no custeio da OP ($CMP)"
else
  err "fita de borda NÃO entrou no custeio da OP ($CMP)"
fi

# ─── RESUMO ───────────────────────────────────────────────────────────────────
c "RESUMO"
printf '\033[1mTotal:\033[0m %d  \033[0;32mPASS:\033[0m %d  \033[0;31mFAIL:\033[0m %d\n' "$TOTAL" "$PASS" "$FAIL"
if [ "$FAIL" -gt 0 ]; then printf '\033[0;31mFalhas:\033[0m\n'; for b in "${BUGS[@]}"; do printf '  • %s\n' "$b"; done; exit 1; fi
printf '\033[0;32mTodos os testes do Plano de Corte passaram.\033[0m\n'
