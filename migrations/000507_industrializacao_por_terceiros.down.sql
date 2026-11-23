DROP INDEX IF EXISTS idx_tps_movements_fiscal_exit;
ALTER TABLE third_party_service_movements DROP COLUMN IF EXISTS fiscal_exit_id, DROP COLUMN IF EXISTS moved_stock;
SELECT instalar_conferencia_de_empresa();
DROP INDEX IF EXISTS uq_warehouse_terceiro_por_fornecedor;
ALTER TABLE warehouse DROP COLUMN IF EXISTS supplier_code;
SELECT instalar_conferencia_de_empresa();
-- O valor TERCEIRO do enum fica: o Postgres não remove valor de enum.
