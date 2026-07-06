#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

export GOCACHE="${GOCACHE:-/tmp/panossoerp-go-build}"

echo "==> Fase 13 / Expedição: testes Go focados"
go test \
  ./internal/application/usecase/shipment_uc \
  ./internal/domain/shipment/... \
  ./internal/infrastructure/repository/shipment \
  ./internal/interfaces/http/handler \
  ./api

echo "==> Fase 13 / Expedição: validacao estatica"
test -f migrations/000196_shipment_loads.up.sql
test -f migrations/000196_shipment_loads.down.sql
grep -q "shipment_loads" migrations/000196_shipment_loads.up.sql
grep -q "shipment_load_shipments" migrations/000196_shipment_loads.up.sql
grep -q "shipment_load_fiscal_notes" migrations/000196_shipment_loads.up.sql
grep -q "shipment_delivery_instructions" migrations/000196_shipment_loads.up.sql
grep -q "shipment_dispatch_boxes" migrations/000196_shipment_loads.up.sql
grep -q "CreateLoad" internal/application/usecase/shipment_uc/load_uc.go
grep -q "LoadMonitor" internal/application/usecase/shipment_uc/load_uc.go
grep -q 'Post("/loads"' api/api.go
grep -q "Monitor de expedição" docs/dev/romaneio.md
grep -q "Carga" docs/apresentacao/romaneio.md

if [[ -n "${BASE_URL:-}" && -n "${TOKEN:-}" ]]; then
  echo "==> Fase 13 / Expedição: smoke HTTP"
  AUTH_HEADER="Authorization: Bearer ${TOKEN}"

  curl -fsS -X POST "${BASE_URL}/api/shipments/dispatch-boxes" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d '{"code":"BOX-F13","description":"Box fase 13","zone":"EXP"}' >/dev/null

  LOAD_JSON="$(curl -fsS -X POST "${BASE_URL}/api/shipments/loads" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d '{"description":"Carga fase 13","carrier_code":1,"vehicle_plate":"ABC1D23","driver_name":"Motorista Teste","route_code":"R-SUL","origin":"Fabrica","destination":"Cliente","dispatch_box_code":"BOX-F13","planned_ship_date":"2026-07-10","estimated_delivery":"2026-07-12"}')"
  LOAD_CODE="$(printf '%s' "$LOAD_JSON" | sed -n 's/.*"code":\([0-9][0-9]*\).*/\1/p')"
  if [[ -z "$LOAD_CODE" ]]; then
    echo "Nao foi possivel extrair a carga criada" >&2
    exit 1
  fi

  curl -fsS -X POST "${BASE_URL}/api/shipments/delivery-instructions" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d "{\"load_code\":${LOAD_CODE},\"title\":\"Entrega agendada\",\"instruction\":\"Confirmar doca antes da saida\",\"priority\":1}" >/dev/null

  curl -fsS "${BASE_URL}/api/shipments/loads/monitor?status=PLANNED" -H "$AUTH_HEADER" >/dev/null
  curl -fsS "${BASE_URL}/api/shipments/loads/separation-monitor" -H "$AUTH_HEADER" >/dev/null
  curl -fsS "${BASE_URL}/api/shipments/loads/logistic-panel" -H "$AUTH_HEADER" >/dev/null
else
  echo "BASE_URL/TOKEN nao definidos; smoke HTTP ignorado."
fi
