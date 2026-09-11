-- A trilha de auditoria nascia sem dono: `GET /api/audit-log` devolvia a
-- qualquer usuário autenticado o rastro de TODAS as empresas — caminho, query
-- string, IP e usuário de cada requisição alheia. Passa a ser por empresa.
--
-- A coluna aceita NULL de propósito: evento anterior à autenticação (tentativa
-- de login, rota não casada) não pertence a empresa nenhuma. Esses ficam fora
-- da API por construção, visíveis apenas a quem consulta o banco.
--
-- Não há backfill. A tabela tem gatilho de imutabilidade
-- (trg_audit_log_immutable) que recusa UPDATE, e essa garantia vale mais do que
-- carimbar empresa em registro antigo: o evento histórico foi gravado sem a
-- informação e continuará dizendo isso — NULL, e não um palpite.
ALTER TABLE audit_log ADD COLUMN IF NOT EXISTS enterprise_id BIGINT;

CREATE INDEX IF NOT EXISTS idx_audit_log_enterprise_occurred
    ON audit_log (enterprise_id, occurred_at DESC, id DESC);
