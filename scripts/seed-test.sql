-- ============================================================================
-- PanossoERP — SEMENTE MÍNIMA PARA A SUÍTE DE INTEGRAÇÃO
-- ----------------------------------------------------------------------------
-- A suíte `-tags=integration` roda contra um Postgres de verdade e monta os
-- próprios dados, mas precisa de dois registros que ela não cria: um USUÁRIO
-- (toda linha auditável tem `created_by` com chave estrangeira para `users`) e
-- uma EMPRESA (desde as migrações de isolamento, `enterprise_id` é NOT NULL em
-- quase tudo). `testutil.Actor` e `testutil.TenantContext` leem daqui.
--
-- Sem esta semente os testes falham com "violates not-null constraint" ou
-- "nenhuma empresa selecionada na sessão atual" — que foi o que manteve a
-- suíte vermelha e fora do CI.
--
-- Não há segredo aqui: a suíte nunca faz login, ela lê `users.id` para preencher
-- `created_by`. O campo de senha recebe um texto inerte de propósito — nada neste
-- arquivo serve para autenticar em lugar nenhum.
--
-- Idempotente: pode rodar de novo sobre um banco já semeado.
--
-- Uso: psql "$TEST_DATABASE_URL" -v ON_ERROR_STOP=1 -f scripts/seed-test.sql
-- ============================================================================

\set ON_ERROR_STOP on

BEGIN;

INSERT INTO users (id, name, email, password, role, is_active)
VALUES ('00000000-0000-0000-0000-00000000f1f1'::uuid, 'Integração', 'integracao@venturerp.test',
        'sem-login', 'ADMIN', true)
ON CONFLICT (email) DO NOTHING;

INSERT INTO enterprise (code, name, created_by)
SELECT 1, 'EMPRESA DE INTEGRACAO', u.id FROM users u WHERE u.email = 'integracao@venturerp.test'
ON CONFLICT (code) DO NOTHING;

INSERT INTO user_enterprises (user_id, enterprise_id, role)
SELECT u.id, e.id, 'ADMIN' FROM users u, enterprise e
WHERE u.email = 'integracao@venturerp.test' AND e.code = 1
ON CONFLICT DO NOTHING;

COMMIT;
