ALTER TABLE purchase_requisition_items DROP COLUMN mask;
DELETE FROM stock_replenishment_policies WHERE mask <> '';
ALTER TABLE stock_replenishment_policies DROP CONSTRAINT uq_politica_reposicao;
ALTER TABLE stock_replenishment_policies ADD CONSTRAINT stock_replenishment_policies_enterprise_id_item_code_wareho_key UNIQUE (enterprise_id, item_code, warehouse_id);
ALTER TABLE stock_replenishment_policies DROP COLUMN mask;
DROP TABLE physical_inventory_notebooks;
DROP INDEX idx_physical_inventory_items_caderno;
ALTER TABLE physical_inventory_items DROP COLUMN notebook_number;
DROP TABLE stock_tolerance_profiles;
