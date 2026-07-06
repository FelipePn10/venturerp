#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

export GOCACHE="${GOCACHE:-/tmp/panossoerp-go-build}"

echo "==> Fase 10 / SAC: testes Go focados"
go test \
  ./internal/application/usecase/consumer_service_uc \
  ./internal/infrastructure/repository/consumer_service \
  ./internal/interfaces/http/handler \
  ./api

echo "==> Fase 10 / SAC: validacao estatica"
grep -q "consumer_service_consumers" migrations/000194_consumer_service.up.sql
grep -q "consumer_service_customer_contacts" migrations/000194_consumer_service.up.sql
grep -q "consumer_service_calls" migrations/000194_consumer_service.up.sql
grep -q "consumer_service_call_returns" migrations/000194_consumer_service.up.sql
grep -q "consumer_service_call_attachments" migrations/000194_consumer_service.up.sql
grep -q "consumer_service_call_checklist_items" migrations/000194_consumer_service.up.sql
grep -q "CreateConsumer" internal/application/usecase/consumer_service_uc/consumer_service_uc.go
grep -q "CreateCall" internal/application/usecase/consumer_service_uc/consumer_service_uc.go
grep -q 'Route("/api/consumer-service"' api/api.go
grep -q "Atendimento ao Consumidor" docs/dev/vendas.md
grep -q "Atendimento ao Consumidor" docs/apresentacao/vendas.md

if [[ -n "${BASE_URL:-}" && -n "${TOKEN:-}" ]]; then
  echo "==> Fase 10 / SAC: smoke HTTP"
  AUTH_HEADER="Authorization: Bearer ${TOKEN}"
  CREATED_BY="${CREATED_BY:-00000000-0000-0000-0000-000000000000}"

  curl -fsS -X POST "${BASE_URL}/api/consumer-service/call-types" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d "{\"description\":\"Smoke SAC\",\"is_complaint\":true,\"created_by\":\"${CREATED_BY}\"}" >/dev/null

  curl -fsS "${BASE_URL}/api/consumer-service/call-types" \
    -H "$AUTH_HEADER" >/dev/null

  curl -fsS "${BASE_URL}/api/consumer-service/consumers?active=true" \
    -H "$AUTH_HEADER" >/dev/null

  curl -fsS "${BASE_URL}/api/consumer-service/calls/report" \
    -H "$AUTH_HEADER" >/dev/null
else
  echo "BASE_URL/TOKEN nao definidos; smoke HTTP ignorado."
fi
