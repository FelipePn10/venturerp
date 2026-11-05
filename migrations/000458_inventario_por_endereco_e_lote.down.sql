DROP TABLE IF EXISTS stock_inconsistency_fixes;

-- Os eventos são imutáveis: só a reversão da migração pode apagá-los.
ALTER TABLE aprovacao_eventos DISABLE TRIGGER trg_aprovacao_eventos_imutaveis;
DELETE FROM aprovacao_eventos e USING aprovacao_solicitacoes s
 WHERE e.enterprise_id = s.enterprise_id AND e.solicitacao_id = s.id AND s.tipo = 'AJUSTE_INVENTARIO';
ALTER TABLE aprovacao_eventos ENABLE TRIGGER trg_aprovacao_eventos_imutaveis;
DELETE FROM aprovacao_solicitacoes WHERE tipo = 'AJUSTE_INVENTARIO';
DELETE FROM aprovacao_politicas WHERE tipo = 'AJUSTE_INVENTARIO';
ALTER TABLE aprovacao_politicas DROP CONSTRAINT IF EXISTS aprovacao_politicas_tipo_check;
ALTER TABLE aprovacao_politicas ADD CONSTRAINT aprovacao_politicas_tipo_check
    CHECK (tipo IN ('LIBERACAO_CREDITO', 'ORCAMENTO_POLITICA_COMERCIAL', 'PEDIDO_POLITICA_COMERCIAL'));

DROP INDEX IF EXISTS uq_physical_inventories_em_andamento;
DROP INDEX IF EXISTS uq_physical_inventory_items_item;

-- Volta a uma linha por item: soma o que foi separado por endereço e lote.
WITH somado AS (
    SELECT MIN(id) AS fica, inventory_id, item_code, mask, warehouse_id,
           SUM(system_qty) AS sistema, SUM(counted_qty) AS contado
      FROM physical_inventory_items
     GROUP BY inventory_id, item_code, mask, warehouse_id
)
UPDATE physical_inventory_items i
   SET system_qty = s.sistema, counted_qty = s.contado, difference_qty = s.contado - s.sistema
  FROM somado s WHERE i.id = s.fica;
DELETE FROM physical_inventory_items i
 WHERE i.id NOT IN (SELECT MIN(id) FROM physical_inventory_items GROUP BY inventory_id, item_code, mask, warehouse_id);
UPDATE physical_inventories SET status = 'OPEN' WHERE status IN ('EM_CONFERENCIA', 'AGUARDANDO_APROVACAO');

ALTER TABLE physical_inventory_items
    DROP COLUMN needs_recount, DROP COLUMN lot, DROP COLUMN address;
CREATE UNIQUE INDEX uq_physical_inventory_items_item
    ON physical_inventory_items (inventory_id, item_code, mask, warehouse_id);

ALTER TABLE physical_inventories
    DROP COLUMN zero_uncounted, DROP COLUMN close_reason, DROP COLUMN tolerance_value, DROP COLUMN tolerance_pct,
    DROP COLUMN approval_mode, DROP COLUMN partial, DROP COLUMN blind_count, DROP COLUMN blocks_movements;
