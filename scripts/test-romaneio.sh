#!/usr/bin/env bash
# ============================================================
# Teste de Romaneio — Geração de PDF e Excel
# Cria dados de teste, gera romaneio, baixa PDF e XLSX
# ============================================================
set -o pipefail

BASE="${BASE_URL:-http://localhost:5071}"
TOKEN="${TOKEN:-}"
USER_UUID="${USER_UUID:-}"
OUTDIR="${OUTDIR:-/tmp/romaneio-test-output}"
PASS=0
FAIL=0

mkdir -p "$OUTDIR"

c()  { printf '\n\033[1;34m═══ %s ═══\033[0m\n' "$*"; }
ok()  { printf '\033[0;32m  ✓ %s\033[0m\n' "$*"; PASS=$((PASS+1)); }
err() { printf '\033[0;31m  ✗ %s\033[0m\n' "$*"; FAIL=$((FAIL+1)); }

request() {
  local method="$1"; shift
  local path="$1"; shift
  local body="${1:-}"
  local url="$BASE$path"
  local response

  if [ -n "$body" ]; then
    response=$(curl -s -w "\n__STATUS__%{http_code}" -X "$method" "$url" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $TOKEN" \
      -d "$body" 2>/dev/null)
  else
    response=$(curl -s -w "\n__STATUS__%{http_code}" -X "$method" "$url" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $TOKEN" 2>/dev/null)
  fi

  HTTP_STATUS=$(echo "$response" | grep '__STATUS__' | sed 's/__STATUS__//')
  echo "$response" | sed '/__STATUS__/d'
}

post() { request POST "$@"; }
get()  { request GET "$1" ""; }

jq_int() { echo "$1" | grep -o "\"$2\":[0-9]*" | head -1 | grep -o '[0-9]*'; }
jq_str() { echo "$1" | grep -o "\"$2\":\"[^\"]*\"" | head -1 | sed 's/.*":"\(.*\)"/\1/'; }

# ─── 1. AUTENTICAÇÃO ─────────────────────────────────────────────────────────
c "1. AUTENTICAÇÃO"

if [ -z "$TOKEN" ]; then
  LOGIN=$(curl -sf -X POST "$BASE/users/login" \
    -H "Content-Type: application/json" \
    -d '{"email":"admin@panossoerp.test","password":"Admin@12345"}' 2>/dev/null)
  TOKEN=$(echo "$LOGIN" | grep -o '"token":"[^"]*"' | head -1 | sed 's/"token":"\(.*\)"/\1/')
  if [ -z "$TOKEN" ]; then
    echo "Erro: nao foi possivel autenticar. Rode o script test-e2e.sh primeiro."
    echo "Ou defina TOKEN=... BASE_URL=..."
    exit 1
  fi
fi

USER_UUID=$(echo "$TOKEN" | cut -d'.' -f2 | \
  awk '{ n=length($0)%4; if(n>0) printf "%s%*s",$0,(4-n),""; else print $0 }' | \
  tr '_-' '/+' | base64 -d 2>/dev/null | \
  python3 -c "import sys,json; print(json.load(sys.stdin)['sub'])" 2>/dev/null || echo "")
ok "Autenticado. User UUID: $USER_UUID"

# ─── 2. HEALTH CHECK ─────────────────────────────────────────────────────────
c "2. HEALTH CHECK"
H=$(curl -sf "$BASE/health" 2>/dev/null)
if echo "$H" | grep -q '"ok"'; then
  ok "API OK"
else
  err "API nao respondeu"; exit 1
fi

# ─── 3. CRIAR ROMANEIO MANUAL ────────────────────────────────────────────────
c "3. CRIAR ROMANEIO MANUAL"

SHIP=$(post "/api/shipments/" "{
  \"reference_type\":\"SALES_ORDER\",
  \"total_volumes\":2,
  \"total_weight\":150.0,
  \"notes\":\"Romaneio Teste PDF/XLSX - $(date +%F_%T)\",
  \"created_by\":\"$USER_UUID\"
}")
SHIP_CODE=$(jq_int "$SHIP" "code")

if [ -z "$SHIP_CODE" ]; then
  err "Falha ao criar romaneio manual"
else
  ok "Romaneio criado: code=$SHIP_CODE"

  # Adicionar itens
  SHIP_I1=$(post "/api/shipments/$SHIP_CODE/items" "{
    \"sequence\":1,\"item_code\":10001,\"quantity\":50.0,\"warehouse_id\":2
  }")
  if echo "$SHIP_I1" | grep -q '"item_code":10001'; then
    ok "Item 10001 adicionado ao romaneio"
  else
    warn "Item 10001 nao encontrado no DB. Criando itens de teste..."

    # Criar items se nao existirem
    post "/api/items/create" "{
      \"code\":10001,\"nature\":2,
      \"pdm\":{\"group_code\":10,\"modifier_code\":0,\"attributes\":[],\"description_technique\":\"Suporte Soldado SS-100\"},
      \"situation\":0,\"health\":\"ATIVO\",
      \"warehouse\":{\"warehouse_code\":1,\"unit_of_measurement\":\"UN\",\"automatic_low\":false,\"minimum_stock\":10},
      \"engineering\":{\"weight\":{\"gross\":1.6,\"net\":1.5,\"unit\":\"KG\"},\"type\":0,\"type_struct\":0,\"oem\":false},
      \"planning\":{\"type_mrp\":0,\"llc\":0,\"ghost\":false},
      \"supplies\":{\"type_of_use\":0},
      \"created_by\":\"$USER_UUID\"
    }" >/dev/null 2>&1

    post "/api/items/create" "{
      \"code\":20001,\"nature\":2,
      \"pdm\":{\"group_code\":10,\"modifier_code\":0,\"attributes\":[],\"description_technique\":\"Chapa de Aco 3mm\"},
      \"situation\":0,\"health\":\"ATIVO\",
      \"warehouse\":{\"warehouse_code\":1,\"unit_of_measurement\":\"KG\",\"automatic_low\":false,\"minimum_stock\":50},
      \"engineering\":{\"weight\":{\"gross\":1.0,\"net\":1.0,\"unit\":\"KG\"},\"type\":0,\"type_struct\":0,\"oem\":false},
      \"planning\":{\"type_mrp\":0,\"llc\":1,\"ghost\":false},
      \"supplies\":{\"type_of_use\":0},
      \"created_by\":\"$USER_UUID\"
    }" >/dev/null 2>&1

    SHIP_I1=$(post "/api/shipments/$SHIP_CODE/items" "{
      \"sequence\":1,\"item_code\":10001,\"quantity\":50.0,\"warehouse_id\":1
    }")
    SHIP_I2=$(post "/api/shipments/$SHIP_CODE/items" "{
      \"sequence\":2,\"item_code\":20001,\"quantity\":100.0,\"warehouse_id\":1
    }")
  fi
fi

# ─── 4. BAIXAR PDF ───────────────────────────────────────────────────────────
c "4. BAIXAR PDF DO ROMANEIO"

PDF_FILE="$OUTDIR/romaneio_${SHIP_CODE}_$(date +%Y%m%d_%H%M%S).pdf"
HTTP_CODE=$(curl -s -o "$PDF_FILE" -w "%{http_code}" -X GET \
  "$BASE/api/shipments/$SHIP_CODE/export/pdf" \
  -H "Authorization: Bearer $TOKEN" 2>/dev/null)

if [ "$HTTP_CODE" = "200" ] && [ -s "$PDF_FILE" ]; then
  PDF_SIZE=$(ls -lh "$PDF_FILE" | awk '{print $5}')
  ok "PDF gerado: $PDF_FILE ($PDF_SIZE)"

  if file "$PDF_FILE" | grep -q "PDF"; then
    ok "Arquivo e um PDF valido"
  fi

  if grep -q "%PDF-1.4" "$PDF_FILE" 2>/dev/null; then
    ok "Header PDF-1.4 detectado"
  fi
else
  err "Falha ao baixar PDF (HTTP $HTTP_CODE)"
fi

# ─── 5. BAIXAR EXCEL ─────────────────────────────────────────────────────────
c "5. BAIXAR EXCEL DO ROMANEIO"

XLSX_FILE="$OUTDIR/romaneio_${SHIP_CODE}_$(date +%Y%m%d_%H%M%S).xlsx"
HTTP_CODE=$(curl -s -o "$XLSX_FILE" -w "%{http_code}" -X GET \
  "$BASE/api/shipments/$SHIP_CODE/export/xlsx" \
  -H "Authorization: Bearer $TOKEN" 2>/dev/null)

if [ "$HTTP_CODE" = "200" ] && [ -s "$XLSX_FILE" ]; then
  XLSX_SIZE=$(ls -lh "$XLSX_FILE" | awk '{print $5}')
  ok "Excel gerado: $XLSX_FILE ($XLSX_SIZE)"
else
  err "Falha ao baixar Excel (HTTP $HTTP_CODE)"
fi

# ─── 6. AUTO-FILL DE PEDIDO DE VENDA ─────────────────────────────────────────
c "6. AUTO-FILL DE PEDIDO DE VENDA"

# Tenta obter um pedido de venda existente
SO_LIST=$(get "/api/sales-order/list")
SO_CODE=$(echo "$SO_LIST" | python3 -c "
import sys,json
d=json.load(sys.stdin)
rows=d if isinstance(d,list) else []
print(rows[0].get('code','')) if rows else print('')
" 2>/dev/null)

if [ -n "$SO_CODE" ] && [ "$SO_CODE" != "null" ]; then
  AFS=$(post "/api/shipments/auto-fill/sales-order" "{
    \"sales_order_code\":$SO_CODE,
    \"created_by\":\"$USER_UUID\"
  }")
  AFS_CODE=$(jq_int "$AFS" "code")
  if [ -n "$AFS_CODE" ]; then
    ok "Auto-fill do pedido de venda $SO_CODE: romaneio $AFS_CODE criado"

    AF_PDF="$OUTDIR/romaneio_autofill_${AFS_CODE}_$(date +%Y%m%d_%H%M%S).pdf"
    curl -s -o "$AF_PDF" "$BASE/api/shipments/$AFS_CODE/export/pdf" \
      -H "Authorization: Bearer $TOKEN" 2>/dev/null
    if [ -s "$AF_PDF" ]; then
      ok "PDF do auto-fill gerado: $AF_PDF ($(ls -lh "$AF_PDF" | awk '{print $5}'))"
    fi
  else
    err "Falha no auto-fill de pedido de venda"
  fi
else
  warn "Nenhum pedido de venda encontrado. Execute test-e2e.sh primeiro para criar dados de teste."
fi

# ─── 7. RESUMO ───────────────────────────────────────────────────────────────
c "RESUMO"

echo ""
echo "  PASS: $PASS"
echo "  FAIL: $FAIL"
echo ""
echo "  Arquivos gerados em: $OUTDIR"
ls -lh "$OUTDIR"/*.pdf "$OUTDIR"/*.xlsx 2>/dev/null || echo "  (nenhum arquivo gerado)"

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
exit 0
