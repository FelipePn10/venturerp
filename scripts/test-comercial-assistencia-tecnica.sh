#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

export GOCACHE="${GOCACHE:-/tmp/panossoerp-go-build}"

echo "==> Fase 9 / Assistencia Tecnica: testes Go focados"
go test \
  ./internal/application/usecase/technical_assistance_uc \
  ./internal/infrastructure/repository/technical_assistance \
  ./internal/interfaces/http/handler \
  ./api

echo "==> Fase 9 / Assistencia Tecnica: validacao estatica"
grep -q "technical_assistance_calls" migrations/000193_technical_assistance.up.sql
grep -q "technical_assistance_defect_reasons" migrations/000193_technical_assistance.up.sql
grep -q "technical_assistance_return_notes" migrations/000193_technical_assistance.up.sql
grep -q "technical_assistance_order_links" migrations/000193_technical_assistance.up.sql
grep -q "CanManageTechnicalAssistance" internal/application/ports/auth_service.go
grep -q "GenerateOrders" internal/application/usecase/technical_assistance_uc/technical_assistance_uc.go
grep -q 'Route("/api/technical-assistance"' api/api.go
grep -q "Assistencia Tecnica" docs/dev/vendas.md
grep -q "Assistencia Tecnica" docs/apresentacao/vendas.md

if [[ -n "${BASE_URL:-}" && -n "${TOKEN:-}" ]]; then
  echo "==> Fase 9 / Assistencia Tecnica: smoke HTTP"
  AUTH_HEADER="Authorization: Bearer ${TOKEN}"
  CREATED_BY="${CREATED_BY:-00000000-0000-0000-0000-000000000000}"

  curl -fsS -X POST "${BASE_URL}/api/technical-assistance/defect-groups" \
    -H "$AUTH_HEADER" -H "Content-Type: application/json" \
    -d "{\"description\":\"Smoke garantia\",\"created_by\":\"${CREATED_BY}\"}" >/dev/null

  curl -fsS "${BASE_URL}/api/technical-assistance/defect-groups" \
    -H "$AUTH_HEADER" >/dev/null

  curl -fsS "${BASE_URL}/api/technical-assistance/calls?active=true" \
    -H "$AUTH_HEADER" >/dev/null

  curl -fsS "${BASE_URL}/api/technical-assistance/calls/report" \
    -H "$AUTH_HEADER" >/dev/null
else
  echo "BASE_URL/TOKEN nao definidos; smoke HTTP ignorado."
fi
