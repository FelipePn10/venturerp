-- O consumo do material consignado dá entrada no estoque; a nota de faturamento do
-- fornecedor vem depois, gera a conta a pagar e não pode entrar de novo. Esta tabela
-- guarda quanto de cada consumo cada nota já faturou.
CREATE TABLE stock_consignment_billings (
    id              bigserial     PRIMARY KEY,
    enterprise_id   bigint        NOT NULL REFERENCES enterprise(id),
    settlement_id   bigint        NOT NULL REFERENCES stock_consignment_settlements(id) ON DELETE CASCADE,
    fiscal_entry_id bigint        NOT NULL,
    quantity        numeric(18,6) NOT NULL CHECK (quantity > 0),
    created_at      timestamptz   NOT NULL DEFAULT NOW(),
    UNIQUE (settlement_id, fiscal_entry_id)
);
CREATE INDEX idx_stock_consignment_billings_nota ON stock_consignment_billings (enterprise_id, fiscal_entry_id);

SELECT instalar_conferencia_de_empresa();
