ALTER TABLE fiscal_return_settlements DROP CONSTRAINT IF EXISTS fiscal_return_settlements_origem_chk;
DELETE FROM fiscal_return_settlements WHERE fiscal_exit_id IS NULL;
ALTER TABLE fiscal_return_settlements
    DROP COLUMN IF EXISTS adiantamento_id,
    DROP COLUMN IF EXISTS devolucao_venda_id,
    ALTER COLUMN fiscal_exit_id SET NOT NULL;
ALTER TABLE fiscal_exit_items DROP COLUMN IF EXISTS fiscal_exit_item_origem_id;
ALTER TABLE fiscal_exits
    DROP COLUMN IF EXISTS devolucao_venda_id,
    DROP COLUMN IF EXISTS tipo_documento;
-- adiantamentos.conta_bancaria_id continua aceitando nulo: os créditos de
-- devolução já gravados não têm conta, e apagá-los perderia o saldo do cliente.
DROP TABLE IF EXISTS devolucoes_venda_itens;
DROP TABLE IF EXISTS devolucoes_venda;
SELECT instalar_conferencia_de_empresa();
