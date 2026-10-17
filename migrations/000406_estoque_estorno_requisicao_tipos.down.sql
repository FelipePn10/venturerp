DROP TABLE IF EXISTS material_requisition_items;
DROP TABLE IF EXISTS material_requisitions;
ALTER TABLE stock_movement_types DROP COLUMN IF EXISTS allows_manual, DROP COLUMN IF EXISTS requires_cost_center;
DROP INDEX IF EXISTS uq_stock_movements_reverses;
ALTER TABLE stock_movements
    DROP COLUMN IF EXISTS cost_center_code,
    DROP COLUMN IF EXISTS movement_type_code,
    DROP COLUMN IF EXISTS reversed_by_movement_id,
    DROP COLUMN IF EXISTS reverses_movement_id;
SELECT instalar_conferencia_de_empresa();
