DROP TRIGGER IF EXISTS trg_stock_movement_segunda_unidade ON stock_movements;
DROP FUNCTION IF EXISTS stock_movement_segunda_unidade();
ALTER TABLE stock_movements DROP COLUMN secondary_quantity;
DROP TRIGGER IF EXISTS trg_production_order_revisao ON production_orders;
DROP FUNCTION IF EXISTS production_order_revisao_do_item();
ALTER TABLE production_orders DROP COLUMN item_revision;
DROP TABLE IF EXISTS item_revisoes;
DROP TABLE IF EXISTS item_gtins;
ALTER TABLE items
    DROP CONSTRAINT chk_items_secondary_uom, DROP CONSTRAINT chk_items_ex_tipi, DROP CONSTRAINT chk_items_manufacturer_cnpj,
    DROP COLUMN description_en, DROP COLUMN warehouse_secondary_uom, DROP COLUMN warehouse_secondary_factor,
    DROP COLUMN accounting_ex_tipi, DROP COLUMN accounting_relevant_scale, DROP COLUMN accounting_manufacturer_cnpj,
    DROP COLUMN accounting_fci_number;
SELECT instalar_conferencia_de_empresa();
