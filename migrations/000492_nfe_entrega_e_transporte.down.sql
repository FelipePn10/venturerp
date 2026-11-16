ALTER TABLE fiscal_exits
    DROP CONSTRAINT IF EXISTS fiscal_exits_tipo_frete_chk,
    DROP COLUMN IF EXISTS local_entrega,
    DROP COLUMN IF EXISTS transportador,
    DROP COLUMN IF EXISTS shipping_carrier_id,
    DROP COLUMN IF EXISTS tipo_frete;

-- Coluna removida de tabela com o gatilho de empresa: reinstala a conferência.
SELECT instalar_conferencia_de_empresa();
