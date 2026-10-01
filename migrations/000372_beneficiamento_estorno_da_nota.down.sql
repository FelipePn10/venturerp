BEGIN;
DROP INDEX IF EXISTS idx_customer_material_movements_nota_viva;
ALTER TABLE customer_material_movements
    DROP CONSTRAINT IF EXISTS customer_material_movements_estorno_completo;
ALTER TABLE customer_material_movements
    DROP COLUMN IF EXISTS reversal_reason,
    DROP COLUMN IF EXISTS reversed_by,
    DROP COLUMN IF EXISTS reversed_at;
COMMIT;
