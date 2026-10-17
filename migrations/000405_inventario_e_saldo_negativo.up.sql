-- Estoque, fase 1: inventário que corrige o saldo e saldo negativo por almoxarifado.
--
-- Inventário físico. Até aqui ele não mexia no estoque: abrir não gerava os
-- itens, contar atualizava uma linha que não existia (e respondia sucesso), a
-- diferença nunca era calculada e fechar só trocava a situação. Agora a abertura
-- grava a FOTO do saldo (quantidade e custo médio), a contagem calcula a
-- diferença contra a foto e o fechamento lança o ajuste. As colunas abaixo
-- guardam a recontagem e o movimento que cada diferença gerou.
ALTER TABLE physical_inventory_items
    ALTER COLUMN system_qty     TYPE NUMERIC(20,6),
    ALTER COLUMN counted_qty    TYPE NUMERIC(20,6),
    ALTER COLUMN difference_qty TYPE NUMERIC(20,6),
    ALTER COLUMN unit_cost      TYPE NUMERIC(20,6),
    ADD COLUMN IF NOT EXISTS count_number           INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS previous_counted_qty   NUMERIC(20,6),
    ADD COLUMN IF NOT EXISTS adjustment_movement_id BIGINT;

-- Uma linha por item do inventário: a contagem passa a ser um upsert.
CREATE UNIQUE INDEX IF NOT EXISTS uq_physical_inventory_items_item
    ON physical_inventory_items (inventory_id, item_code, mask, warehouse_id);

ALTER TABLE physical_inventories
    ADD COLUMN IF NOT EXISTS closed_by        UUID,
    ADD COLUMN IF NOT EXISTS adjustment_value NUMERIC(18,2);

-- Contagem cíclica aprovada com divergência também passa a ajustar o saldo.
ALTER TABLE stock_cycle_counts
    ADD COLUMN IF NOT EXISTS adjustment_movement_id BIGINT;

-- Saldo negativo. A saída que deixaria o saldo abaixo de zero passa a ser
-- recusada, salvo no almoxarifado que permite (decisão de 17/10/2026). A linha
-- de produção nasce permitindo: a baixa automática (backflush) consome antes de
-- a entrada ser lançada, e bloquear ali pararia o apontamento.
ALTER TABLE warehouse
    ADD COLUMN IF NOT EXISTS allow_negative_stock BOOLEAN NOT NULL DEFAULT FALSE;
UPDATE warehouse SET allow_negative_stock = TRUE WHERE type = 'LINHA_DE_PRODUCAO';

COMMENT ON COLUMN warehouse.allow_negative_stock IS
    'Permite saída que deixa o saldo negativo. Falso por padrão; verdadeiro na linha de produção (baixa automática).';
