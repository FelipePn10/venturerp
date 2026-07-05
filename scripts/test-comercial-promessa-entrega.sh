#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

export GOCACHE="${GOCACHE:-/tmp/panossoerp-go-build}"

echo "==> Fase 8 / Promessa de Entrega: testes Go focados"
go test \
  ./internal/application/usecase/delivery_promise_uc \
  ./internal/infrastructure/repository/delivery_promise \
  ./internal/interfaces/http/handler \
  ./api

echo "==> Fase 8 / Promessa de Entrega: validacao estatica"
grep -q "delivery_tank_reservations" migrations/000192_delivery_promise.up.sql
grep -q "DeliveryPromiseUseCase" internal/application/usecase/delivery_promise_uc/delivery_promise_uc.go
grep -q "ReserveTank" internal/application/usecase/delivery_promise_uc/delivery_promise_uc.go
grep -q 'Get("/occupation", deliveryPromiseHandler.Occupation)' api/api.go
grep -q 'Post("/tank-reservations", deliveryPromiseHandler.ReserveTank)' api/api.go
grep -q 'Post("/reschedule", deliveryPromiseHandler.Reschedule)' api/api.go
grep -q "Promessa de Entrega" docs/dev/vendas.md
grep -q "Promessa de entrega" docs/apresentacao/vendas.md

if [[ -n "${BASE_URL:-}" && -n "${TOKEN:-}" ]]; then
  echo "==> Fase 8 / Promessa de Entrega: smoke HTTP"
  AUTH_HEADER="Authorization: Bearer ${TOKEN}"

  curl -fsS "${BASE_URL}/api/delivery-promise/occupation?from_date=2026-07-01&to_date=2026-07-31&daily_capacity=50" \
    -H "$AUTH_HEADER" >/dev/null

  curl -fsS -X POST "${BASE_URL}/api/delivery-promise/tank-reservations" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d '{"customer_code":1,"requested_delivery_date":"2026-07-15","firm_days":3,"daily_capacity":50,"verify_stock":false,"commit":false,"created_by":"00000000-0000-0000-0000-000000000000","lines":[{"item_code":1,"mask":"","quantity":10,"unit_price":100}]}' >/dev/null

  curl -fsS -X POST "${BASE_URL}/api/delivery-promise/reschedule" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d '{"delivery_from":"2026-07-01","delivery_to":"2026-07-31","new_date":"2026-08-05","sales_order_codes":[],"item_codes":[],"reason":"Smoke promessa entrega","created_by":"00000000-0000-0000-0000-000000000000"}' >/dev/null
else
  echo "BASE_URL/TOKEN nao definidos; smoke HTTP ignorado."
fi
