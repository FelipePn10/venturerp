-- Contagem cíclica por endereço.
--
-- `stock_cycle_counts` já tinha `warehouse_address_id`, mas era um bigint sem
-- destino: os endereços vivem em `manufacturing_warehouse_addresses`, cuja chave
-- é (empresa, almoxarifado, endereço) — não existe `id` para referenciar. A
-- coluna tinha apenas um CHECK (> 0) e nenhuma chave estrangeira.
--
-- Com o endereçamento da migração 344, a contagem precisa do endereço no mesmo
-- formato do saldo (varchar), senão a quantidade esperada é lida do almoxarifado
-- inteiro e toda contagem de um endereço específico acusa divergência falsa.

ALTER TABLE stock_cycle_counts ADD COLUMN IF NOT EXISTS address VARCHAR(100) NOT NULL DEFAULT '';

COMMENT ON COLUMN stock_cycle_counts.address IS
    'Endereço contado; vazio = contagem do almoxarifado inteiro.';
COMMENT ON COLUMN stock_cycle_counts.warehouse_address_id IS
    'Obsoleto: nunca teve tabela-alvo. Use address (migração 345).';

CREATE INDEX IF NOT EXISTS idx_stock_cycle_counts_endereco
    ON stock_cycle_counts (enterprise_id, warehouse_id, address, state);
