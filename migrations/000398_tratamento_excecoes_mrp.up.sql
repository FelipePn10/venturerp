-- Ação direto na exceção do MRP.
--
-- A exceção dizia "antecipar para 17/10" só no texto; para reprogramar a ordem
-- a partir dela, a data sugerida precisa estar num campo. E o planejador
-- precisa registrar que tratou a mensagem (e como), como no SAP (MD06/MD07,
-- "exception processed") e no Oracle (status da exceção): sem isso, toda
-- mensagem parece pendente até o próximo cálculo.
ALTER TABLE mrp_exception_messages
    ADD COLUMN IF NOT EXISTS suggested_date  date,
    ADD COLUMN IF NOT EXISTS handled_at      timestamptz,
    ADD COLUMN IF NOT EXISTS handled_by      uuid REFERENCES users(id),
    ADD COLUMN IF NOT EXISTS handling_action varchar(20),
    ADD COLUMN IF NOT EXISTS handling_note   text;

ALTER TABLE mrp_exception_messages
    ADD CONSTRAINT ck_mrp_exception_handling_action
    CHECK (handling_action IS NULL OR handling_action IN ('REPROGRAMADA', 'CANCELADA', 'TRATADA'));

COMMENT ON COLUMN mrp_exception_messages.suggested_date IS
    'Data em que o MRP precisa do suprimento; reprogramar a ordem leva a ela.';
COMMENT ON COLUMN mrp_exception_messages.handling_action IS
    'Como o planejador tratou a mensagem: REPROGRAMADA, CANCELADA ou TRATADA (sem alterar a ordem).';
