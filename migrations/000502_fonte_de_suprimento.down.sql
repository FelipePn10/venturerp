DROP TRIGGER IF EXISTS trg_fonte_de_suprimento ON purchase_requisition_items;
DROP TRIGGER IF EXISTS trg_fonte_de_suprimento ON third_party_service_movements;
DROP TRIGGER IF EXISTS trg_fonte_de_suprimento ON third_party_service_orders;
DROP TRIGGER IF EXISTS trg_fonte_de_suprimento ON purchase_orders;
DROP TRIGGER IF EXISTS trg_fonte_de_suprimento ON purchase_order_items;
DROP FUNCTION IF EXISTS erp_conferir_fonte_de_suprimento();
DROP FUNCTION IF EXISTS erp_exigir_fonte_da_os(BIGINT, BIGINT, TEXT, BIGINT);
DROP FUNCTION IF EXISTS erp_exigir_fonte_de_suprimento(BIGINT, TEXT, BIGINT);
DROP FUNCTION IF EXISTS erp_fonte_de_suprimento_valida(BIGINT, TEXT, BIGINT, DATE);
DROP FUNCTION IF EXISTS erp_exige_fonte_de_suprimento(BIGINT);

DROP TABLE IF EXISTS item_supplier_block_history;
ALTER TABLE planned_orders DROP COLUMN IF EXISTS supplier_code;

ALTER TABLE item_preferred_suppliers
    DROP CONSTRAINT IF EXISTS item_preferred_suppliers_vigencia_check,
    DROP CONSTRAINT IF EXISTS item_preferred_suppliers_bloqueio_check,
    DROP CONSTRAINT IF EXISTS item_preferred_suppliers_cota_check,
    DROP CONSTRAINT IF EXISTS item_preferred_suppliers_lote_check,
    DROP COLUMN IF EXISTS blocked_by,
    DROP COLUMN IF EXISTS blocked_at,
    DROP COLUMN IF EXISTS blocked_reason,
    DROP COLUMN IF EXISTS blocked,
    DROP COLUMN IF EXISTS quota_since,
    DROP COLUMN IF EXISTS quota_percent,
    DROP COLUMN IF EXISTS order_multiple,
    DROP COLUMN IF EXISTS min_order_qty;

SELECT instalar_conferencia_de_empresa();
