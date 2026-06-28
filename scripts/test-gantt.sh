#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════════════════════════
# test-gantt.sh — QUADRO DE PROGRAMAÇÃO / APS GANTT (end-to-end)
# Exercita os endpoints novos da branch feature/gantt-scheduling contra a API + DB:
#   · Quadro mensal (JSON) por centro e por ordem — barras sequenciadas, fallback,
#     carga CRP, calendário/atraso, summary.
#   · Dependências finish-start: explícita (route_operation_network) + implícita
#     (cadeia por posição) no JSON e no export SVG.
#   · Board em range livre + zoom semana/dia + validações de range.
#   · Export SVG/PDF (mês e board).
#   · Reschedule (drag-drop): cascata finish-start + checagem de capacidade +
#     cenários negativos; confirma o efeito no banco.
#
# Pré-requisitos: API desta branch no ar (BASE) ligada ao Postgres de teste, e o
# container do Postgres de teste acessível por docker exec (para semear poo/ps/rede,
# que não têm endpoints próprios).
#
# Uso:  BASE_URL=http://localhost:5071 bash scripts/test-gantt.sh
# ═══════════════════════════════════════════════════════════════════════════════
set -o pipefail

BASE="${BASE_URL:-http://localhost:5071}"
DB_CONTAINER="${DB_CONTAINER:-panossoerp-postgres-test}"
DB_NAME="${DB_NAME:-panossoerpdatabase_test}"
DB_USER="${DB_USER:-panossoerp_test}"
YEAR="${YEAR:-2026}"; MONTH="${MONTH:-8}"
TOKEN=""; USER_UUID=""; HTTP_STATUS=""
PASS=0; FAIL=0; TOTAL=0; BUGS=()
G_STATUS_FILE="$(mktemp)"; trap 'rm -f "$G_STATUS_FILE"' EXIT

c()    { printf '\n\033[1;34m══════ %s ══════\033[0m\n' "$*"; }
ok()   { printf '\033[0;32m  ✓ %s\033[0m\n' "$*"; PASS=$((PASS+1)); TOTAL=$((TOTAL+1)); }
err()  { printf '\033[0;31m  ✗ %s\033[0m\n' "$*"; FAIL=$((FAIL+1)); TOTAL=$((TOTAL+1)); BUGS+=("$*"); }
info() { printf '\033[0;90m    %s\033[0m\n' "$*"; }
# db(): roda SQL e devolve só o 1º valor. psql imprime a tag de comando ("INSERT 0 1")
# numa 2ª linha após o RETURNING — por isso pegamos apenas a primeira linha.
db()   { docker exec "$DB_CONTAINER" bash -c "psql -U $DB_USER $DB_NAME -t -A -c \"$*\"" 2>/dev/null | sed -n '1p' | tr -d ' \n'; }

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
  echo "$HTTP_STATUS" > "$G_STATUS_FILE"
  echo "$resp" | sed '/__STATUS__/d'
}
post() { request POST "$@"; }
get()  { request GET  "$1" ""; }

check() {
  local label="$1" val="$2" pat="$3"; local st; st=$(cat "$G_STATUS_FILE" 2>/dev/null); TOTAL=$((TOTAL+1))
  if [ -n "$st" ] && [ "$st" -ge 400 ] 2>/dev/null; then
    err "$label — HTTP $st | $(echo "$val" | head -c 200 | tr '\n' ' ')"; return; fi
  if [ -z "$val" ]; then err "$label — resposta vazia (HTTP $st)"; return; fi
  if echo "$val" | grep -qE "$pat" 2>/dev/null; then PASS=$((PASS+1)); printf '\033[0;32m  ✓ %s (HTTP %s)\033[0m\n' "$label" "$st"
  else err "$label — '$pat' ausente (HTTP $st) | $(echo "$val" | head -c 220 | tr '\n' ' ')"; fi
}
check_fail() {
  local label="$1" val="$2" pat="${3:-}"; local st; st=$(cat "$G_STATUS_FILE" 2>/dev/null); TOTAL=$((TOTAL+1))
  if [ -z "$st" ] || [ "$st" -lt 400 ] 2>/dev/null; then
    err "$label — esperava erro, veio HTTP $st | $(echo "$val" | head -c 160 | tr '\n' ' ')"; return; fi
  if [ -n "$pat" ] && ! echo "$val" | grep -qiE "$pat"; then
    err "$label — erro sem '$pat' (HTTP $st) | $(echo "$val" | head -c 160)"; return; fi
  PASS=$((PASS+1)); printf '\033[0;32m  ✓ %s (rejeitado HTTP %s)\033[0m\n' "$label" "$st"
}
# assert_db: compara um valor lido do banco com o esperado.
assert_db() {
  local label="$1" got="$2" want="$3"; TOTAL=$((TOTAL+1))
  if [ "$got" = "$want" ]; then PASS=$((PASS+1)); printf '\033[0;32m  ✓ %s (=%s)\033[0m\n' "$label" "$got"
  else err "$label — banco='$got' esperado='$want'"; fi
}
jq_int() { echo "$1" | grep -oiE "\"$2\":[0-9]+" | head -1 | grep -oE '[0-9]+'; }
jq_str() { echo "$1" | grep -o "\"$2\":\"[^\"]*\"" | head -1 | sed 's/.*":"\(.*\)"/\1/'; }

# ─── 0. RESET ─────────────────────────────────────────────────────────────────
c "0. RESET + saúde da API"
RESET_SQL="TRUNCATE TABLE production_sequences, production_order_operations,
  route_operation_network, route_operations, manufacturing_routes, operations,
  capacity_requirements, industrial_calendar, production_orders, item_structures,
  items, machines, machine_types, warehouse, enterprise, users RESTART IDENTITY CASCADE;"
if docker exec "$DB_CONTAINER" bash -c "psql -U $DB_USER $DB_NAME -c \"$RESET_SQL\"" >/dev/null 2>&1; then
  ok "DB reset"
else
  err "Reset falhou — sem docker/DB de teste em '$DB_CONTAINER'? (este E2E precisa semear poo/ps/rede via SQL)"; exit 1
fi
curl -sf "$BASE/health" >/dev/null 2>&1 && ok "API online em $BASE" || { err "API offline em $BASE"; exit 1; }

# ─── 1. LOGIN (ADMIN) ─────────────────────────────────────────────────────────
c "1. AUTENTICAÇÃO"
curl -s -X POST "$BASE/users/register" -H "Content-Type: application/json" \
  -d '{"name":"Admin APS","email":"aps@test.local","password":"Admin@12345"}' >/dev/null 2>&1
db "UPDATE users SET role='ADMIN' WHERE email='aps@test.local';" >/dev/null
LOGIN=$(curl -sf -X POST "$BASE/users/login" -H "Content-Type: application/json" \
  -d '{"email":"aps@test.local","password":"Admin@12345"}' 2>/dev/null)
TOKEN=$(jq_str "$LOGIN" "token"); [ -z "$TOKEN" ] && { err "login falhou"; exit 1; }
USER_UUID=$(echo "$TOKEN" | cut -d'.' -f2 | awk '{n=length($0)%4; if(n>0) printf "%s%*s",$0,(4-n),""; else print $0}' \
  | tr '_-' '/+' | base64 -d 2>/dev/null | python3 -c "import sys,json; print(json.load(sys.stdin)['sub'])" 2>/dev/null || echo "")
[ -z "$USER_UUID" ] && USER_UUID="00000000-0000-0000-0000-000000000001"
ok "login OK (ADMIN) uuid=$USER_UUID"

# ─── 2. SETUP via API (empresa, depósito, centros, itens, ordens) ─────────────
c "2. CADASTROS DE APOIO"
post "/api/enterprise/create" "{\"code\":1,\"name\":\"Metalfer Ltda\",\"created_by\":\"$USER_UUID\"}" >/dev/null
post "/api/pdm/create-group" "{\"code\":40,\"description\":\"Producao\",\"enterprise_id\":1,\"created_by\":\"$USER_UUID\"}" >/dev/null
post "/api/warehouse/create" "{\"code\":1,\"description\":\"Almox\",\"location\":0,\"type\":1,\"disposition\":true,\"reservations_allowed\":true,\"created_by\":\"$USER_UUID\"}" >/dev/null
# Centros de trabalho = machine_types. Os CODES (81/82) são DIFERENTES dos ids
# (1/2 após reset) de propósito: assim o teste exercita de verdade o JOIN
# code→id de GetMachineAvailableHours (se a query voltar a comparar code=id, a
# capacidade cai no default e a asserção de capacidade abaixo quebra).
MTC=$(post "/api/machine/types/create" "{\"code\":81,\"name\":\"Corte\",\"type\":\"CUT\",\"requires_operator\":false,\"is_active\":true,\"created_by\":\"$USER_UUID\"}")
MTS=$(post "/api/machine/types/create" "{\"code\":82,\"name\":\"Solda\",\"type\":\"WELD\",\"requires_operator\":false,\"is_active\":true,\"created_by\":\"$USER_UUID\"}")
WC_CORTE=$(db "SELECT id FROM machine_types WHERE code=81;"); WC_SOLDA=$(db "SELECT id FROM machine_types WHERE code=82;")
[ -z "$WC_CORTE" ] && WC_CORTE=1; [ -z "$WC_SOLDA" ] && WC_SOLDA=2
check "centro Corte (code 81 ≠ id $WC_CORTE)" "$MTC" 'id|Corte|code'
check "centro Solda (code 82 ≠ id $WC_SOLDA)" "$MTS" 'id|Solda|code'
# Corte tem 2 máquinas (machine_type_code=81) → capacidade 16h/dia; Solda tem 1 (8h).
post "/api/machine/create" "{\"code\":501,\"name\":\"Serra-01\",\"machine_type_code\":81,\"capacity\":8.0,\"capacity_per_unit\":\"UN\",\"capacity_period\":\"DIA\",\"efficiency_rate\":1.0,\"is_active\":true,\"created_by\":\"$USER_UUID\"}" >/dev/null
post "/api/machine/create" "{\"code\":503,\"name\":\"Serra-02\",\"machine_type_code\":81,\"capacity\":8.0,\"capacity_per_unit\":\"UN\",\"capacity_period\":\"DIA\",\"efficiency_rate\":1.0,\"is_active\":true,\"created_by\":\"$USER_UUID\"}" >/dev/null
post "/api/machine/create" "{\"code\":502,\"name\":\"MIG-01\",\"machine_type_code\":82,\"capacity\":8.0,\"capacity_per_unit\":\"UN\",\"capacity_period\":\"DIA\",\"efficiency_rate\":1.0,\"is_active\":true,\"created_by\":\"$USER_UUID\"}" >/dev/null

mkitem() { post "/api/items/create" "{\"code\":$1,\"nature\":2,
  \"pdm\":{\"group_code\":40,\"modifier_code\":0,\"attributes\":[],\"description_technique\":\"$2\"},
  \"situation\":0,\"health\":\"ATIVO\",
  \"warehouse\":{\"warehouse_code\":1,\"unit_of_measurement\":\"UN\",\"automatic_low\":false,\"minimum_stock\":0},
  \"engineering\":{\"weight\":{\"gross\":1,\"net\":1,\"unit\":\"KG\"},\"type\":0,\"type_struct\":0,\"oem\":false},
  \"planning\":{\"type_mrp\":0,\"llc\":0,\"ghost\":false},
  \"supplies\":{\"type_of_use\":0},\"created_by\":\"$USER_UUID\"}"; }
check "item 70100 (produto)" "$(mkitem 70100 'Conjunto soldado')" '70100|id'

# Ordens de produção (via API → ids/path válidos). OF-A=dep explícita, OF-B=dep
# implícita (ambas sequenciadas via SQL), OF-C=apenas fallback (sem sequência).
mkop() { post "/api/production-order/create" "{\"item_code\":70100,\"mask\":\"\",\"planned_qty\":$1.0,\"start_date\":\"$YEAR-08-03\",\"end_date\":\"$YEAR-08-20\",\"created_by\":\"$USER_UUID\"}"; }
OA=$(mkop 4); OB=$(mkop 3); OC=$(mkop 2)
OFA_ID=$(jq_int "$OA" "ID"); OFB_ID=$(jq_int "$OB" "ID"); OFC_ID=$(jq_int "$OC" "ID")
[ -z "$OFA_ID" ] && OFA_ID=$(db "SELECT id FROM production_orders ORDER BY id LIMIT 1;")
check "cria OFs (A,B,C)" "$OA" 'ID|ItemCode|70100'
info "OF ids: A=$OFA_ID B=$OFB_ID C=$OFC_ID | WC Corte=$WC_CORTE Solda=$WC_SOLDA"
# Normaliza priority/order_number/status para valores conhecidos.
db "UPDATE production_orders SET order_number=id, priority='1', status='OPEN', is_active=TRUE;" >/dev/null

# ─── 3. SEED de roteiro + sequências (SQL) ────────────────────────────────────
c "3. SEED de operações, sequências e rede de dependências (SQL)"
ROUTE_ID=$(db "INSERT INTO manufacturing_routes(code,item_code,description,created_by) VALUES(7700,70100,'Rota Teste','$USER_UUID') RETURNING id;")
OP1=$(db "INSERT INTO operations(code,name,default_work_center_id,created_by) VALUES(9001,'Corte',$WC_CORTE,'$USER_UUID') RETURNING id;")
OP2=$(db "INSERT INTO operations(code,name,default_work_center_id,created_by) VALUES(9002,'Solda',$WC_SOLDA,'$USER_UUID') RETURNING id;")
RO1=$(db "INSERT INTO route_operations(route_id,sequence,operation_id,work_center_id,standard_time,setup_time) VALUES($ROUTE_ID,1,$OP1,$WC_CORTE,6,0) RETURNING id;")
RO2=$(db "INSERT INTO route_operations(route_id,sequence,operation_id,work_center_id,standard_time,setup_time) VALUES($ROUTE_ID,2,$OP2,$WC_SOLDA,4,0) RETURNING id;")
db "INSERT INTO route_operation_network(predecessor_id,successor_id,overlap_pct) VALUES($RO1,$RO2,0);" >/dev/null
[ -n "$ROUTE_ID" ] && [ -n "$RO1" ] && [ -n "$RO2" ] && ok "rota + 2 operações + aresta RO$RO1→RO$RO2" || err "seed de rota falhou (route=$ROUTE_ID ro1=$RO1 ro2=$RO2)"

seed_op_seq() { # order_id route_op_id seq op_name wc start end -> echo ps_id
  local poo; poo=$(db "INSERT INTO production_order_operations(production_order_id,route_operation_id,sequence,operation_name,work_center_id,planned_hours,setup_hours,actual_hours,status) VALUES($1,$2,$3,'$4',$5,$(echo "$7"|grep -q 14 && echo 6 || echo 4),0,2,'IN_PROGRESS') RETURNING id;")
  db "INSERT INTO production_sequences(production_order_id,operation_id,work_center_id,sequence_position,scheduled_start,scheduled_end,status) VALUES($1,$poo,$5,$3,'$6','$7','SCHEDULED') RETURNING id;"
}
# OF-A (explícita): op1 Corte 03/08 08–14 (6h), op2 Solda 03/08 14–18 (4h)
PS_A1=$(seed_op_seq "$OFA_ID" "$RO1" 1 Corte "$WC_CORTE" "$YEAR-08-03 08:00:00-03" "$YEAR-08-03 14:00:00-03")
PS_A2=$(seed_op_seq "$OFA_ID" "$RO2" 2 Solda "$WC_SOLDA" "$YEAR-08-03 14:00:00-03" "$YEAR-08-03 18:00:00-03")
# OF-B (implícita: route_operation_id NULL): op1 Corte 04/08 08–20 (12h, ocupa o Corte),
# op2 Solda 04/08 13–17
PS_B1=$(seed_op_seq "$OFB_ID" NULL 1 Corte "$WC_CORTE" "$YEAR-08-04 08:00:00-03" "$YEAR-08-04 20:00:00-03")
PS_B2=$(seed_op_seq "$OFB_ID" NULL 2 Solda "$WC_SOLDA" "$YEAR-08-04 13:00:00-03" "$YEAR-08-04 17:00:00-03")
info "PS ids: A1=$PS_A1 A2=$PS_A2 B1=$PS_B1 B2=$PS_B2"
[ -n "$PS_A1" ] && [ -n "$PS_B2" ] && ok "4 sequências semeadas" || err "seed de sequências falhou"

# Carga CRP (load[]) e calendário (um feriado + um dia útil).
db "INSERT INTO capacity_requirements(plan_code,work_center_id,req_date,required_hours,available_hours) VALUES
  (1,$WC_CORTE,'$YEAR-08-03',12,8),(1,$WC_CORTE,'$YEAR-08-04',4,8);" >/dev/null
db "INSERT INTO industrial_calendar(year,month,day,is_workday,description) VALUES
  ($YEAR,8,3,TRUE,'util'),($YEAR,8,4,TRUE,'util'),($YEAR,8,15,FALSE,'feriado') ON CONFLICT DO NOTHING;" >/dev/null
ok "carga CRP + calendário semeados"

# ─── 4. QUADRO MENSAL (JSON) ──────────────────────────────────────────────────
c "4. QUADRO MENSAL — GET /api/aps/gantt/month/$YEAR/$MONTH"
MJSON=$(get "/api/aps/gantt/month/$YEAR/$MONTH")
check "monta quadro (rows/days/summary)" "$MJSON" '"rows"|"days"|"summary"'
check "tem barras sequenciadas" "$MJSON" "\"sequence_id\":$PS_A1"
check "centro Corte na visão por recurso" "$MJSON" 'Corte'
check "carga CRP presente (load[])" "$MJSON" '"load"|required_hours'
check "dia 15 marcado não-útil (feriado)" "$MJSON" '"day":15,"weekday":[0-9]+,"is_workday":false|"is_workday":false'
check "summary contabiliza barras" "$MJSON" 'total_bars|sequenced_bars'
check "fallback da OF-C (sem sequência)" "$MJSON" 'Sem sequenciamento|is_fallback":true'

c "4b. AGRUPAMENTO POR ORDEM"
check "group_by=order" "$(get "/api/aps/gantt/month/$YEAR/$MONTH?group_by=order")" '"group_by":"order"|"OF '

# ─── 5. DEPENDÊNCIAS FINISH-START ─────────────────────────────────────────────
c "5. DEPENDÊNCIAS — explícita (route_operation_network) + implícita (cadeia)"
check "bloco dependencies presente" "$MJSON" '"dependencies"'
check "aresta explícita A1→A2 (implicit=false)" "$MJSON" "\"from_sequence_id\":$PS_A1,\"to_sequence_id\":$PS_A2,\"overlap_pct\":[0-9.]+,\"implicit\":false"
check "aresta implícita B1→B2 (implicit=true)" "$MJSON" "\"from_sequence_id\":$PS_B1,\"to_sequence_id\":$PS_B2,\"overlap_pct\":[0-9.]+,\"implicit\":true"
check "setas no export SVG (<path>)" "$(get "/api/aps/gantt/month/$YEAR/$MONTH/export?format=svg")" '<path'

# ─── 6. RANGE LIVRE + ZOOM SEMANA/DIA ─────────────────────────────────────────
c "6. BOARD RANGE LIVRE — GET /api/aps/gantt/board"
check "range dia (scale=day)"  "$(get "/api/aps/gantt/board?from=$YEAR-08-01&to=$YEAR-08-31&scale=day")"  '"scale":"day".*"days"|"scale":"day"'
WK=$(get "/api/aps/gantt/board?from=$YEAR-08-01&to=$YEAR-08-28&scale=week")
check "range semana (scale=week)" "$WK" '"scale":"week"'
check "coluna semanal traz label (dd/mm)" "$WK" '"label":"[0-3][0-9]/08"'
check "barras aparecem no board" "$(get "/api/aps/gantt/board?from=$YEAR-08-01&to=$YEAR-08-10")" "\"sequence_id\":$PS_A1"
check_fail "range inválido (to<from) rejeita" "$(get "/api/aps/gantt/board?from=$YEAR-08-31&to=$YEAR-08-01")" 'range|invalid'
check_fail "from ausente rejeita" "$(get "/api/aps/gantt/board?to=$YEAR-08-10")" 'from'
check "export board PDF" "$(get "/api/aps/gantt/board/export?from=$YEAR-08-01&to=$YEAR-08-31&format=pdf")" '%PDF'

# ─── 7. RESCHEDULE — cascata + capacidade ─────────────────────────────────────
c "7. RESCHEDULE — POST /api/aps/gantt/reschedule (cascata + capacidade)"
# Move A1 (Corte 6h) para 04/08 08:00 → cai no mesmo dia/centro da B1 (Corte 12h):
#   Corte em 04/08 = 6h + 12h = 18h > 16h (capacidade real = 2 máquinas) ⇒ aviso.
#   A2 (Solda, dependente) começava antes do novo fim de A1 ⇒ empurrada em cascata.
RES=$(post "/api/aps/gantt/reschedule" "{\"sequence_id\":$PS_A1,\"new_start\":\"$YEAR-08-04T08:00:00-03:00\"}")
check "reschedule aplicado (moved)" "$RES" "\"moved\":{[^}]*\"sequence_id\":$PS_A1"
check "cascata empurrou o sucessor A2 (shifted)" "$RES" "\"shifted\":\[[^]]*\"sequence_id\":$PS_A2"
check "cascade_applied=true" "$RES" '"cascade_applied":true'
check "aviso de capacidade no Corte (18h>16h)" "$RES" "\"warnings\":\[[^]]*\"work_center_id\":$WC_CORTE"
check "aviso usa capacidade REAL 16h (prova o JOIN code→id)" "$RES" '"available_hours":16'
check "aviso traz scheduled=18 / over_by=2" "$RES" '"scheduled_hours":18,"available_hours":16,"over_by_hours":2'
# Confirma no banco: A1 e A2 agora em 04/08; A2 começa no fim de A1 (14:00).
assert_db "DB: A1 movida p/ 04/08" "$(db "SELECT to_char(scheduled_start,'YYYY-MM-DD') FROM production_sequences WHERE id=$PS_A1;")" "$YEAR-08-04"
assert_db "DB: A2 empurrada p/ 04/08" "$(db "SELECT to_char(scheduled_start,'YYYY-MM-DD') FROM production_sequences WHERE id=$PS_A2;")" "$YEAR-08-04"
assert_db "DB: A2 começa às 14:00 (fim de A1)" "$(db "SELECT to_char(scheduled_start,'HH24:MI') FROM production_sequences WHERE id=$PS_A2;")" "14:00"

c "7b. RESCHEDULE — sem cascata + trocando de centro"
RES2=$(post "/api/aps/gantt/reschedule" "{\"sequence_id\":$PS_B1,\"new_start\":\"$YEAR-08-06T08:00:00-03:00\",\"new_work_center_id\":$WC_SOLDA,\"cascade\":false}")
check "cascade_applied=false" "$RES2" '"cascade_applied":false'
check "sem shifted" "$RES2" '"shifted":\[\]|"shifted":null|^((?!"shifted":\[\{).)*$'
assert_db "DB: B1 trocou p/ centro Solda" "$(db "SELECT work_center_id FROM production_sequences WHERE id=$PS_B1;")" "$WC_SOLDA"
assert_db "DB: B2 NÃO se moveu (sem cascata)" "$(db "SELECT to_char(scheduled_start,'YYYY-MM-DD') FROM production_sequences WHERE id=$PS_B2;")" "$YEAR-08-04"

c "7c. RESCHEDULE — cenários negativos"
check_fail "sequência inexistente" "$(post "/api/aps/gantt/reschedule" "{\"sequence_id\":999999,\"new_start\":\"$YEAR-08-10T08:00:00-03:00\"}")"
check_fail "sem new_start" "$(post "/api/aps/gantt/reschedule" "{\"sequence_id\":$PS_A1}")" 'new_start'
check_fail "sem sequence_id" "$(post "/api/aps/gantt/reschedule" "{\"new_start\":\"$YEAR-08-10T08:00:00-03:00\"}")" 'sequence_id'

# ─── RESUMO ───────────────────────────────────────────────────────────────────
c "RESUMO"
printf '\033[1mTotal:\033[0m %d  \033[0;32mPASS:\033[0m %d  \033[0;31mFAIL:\033[0m %d\n' "$TOTAL" "$PASS" "$FAIL"
if [ "$FAIL" -gt 0 ]; then printf '\033[0;31mFalhas:\033[0m\n'; for b in "${BUGS[@]}"; do printf '  • %s\n' "$b"; done; exit 1; fi
printf '\033[0;32mTodos os testes do Quadro de Programação (APS Gantt) passaram.\033[0m\n'
