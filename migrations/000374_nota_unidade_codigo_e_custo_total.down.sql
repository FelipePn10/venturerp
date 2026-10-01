BEGIN;
ALTER TABLE item_standard_costs DROP COLUMN IF EXISTS total_cost;
ALTER TABLE item_standard_costs
    ADD COLUMN total_cost NUMERIC(18,6)
    GENERATED ALWAYS AS (material_cost + labor_cost + overhead_cost) STORED;
DROP INDEX IF EXISTS uq_fiscal_exits_rascunho_de_beneficiamento;
ALTER TABLE fiscal_exits DROP COLUMN IF EXISTS customer_material_remittance_id;
ALTER TABLE fiscal_exit_items
    DROP COLUMN IF EXISTS codigo_produto,
    DROP COLUMN IF EXISTS unidade_comercial;
COMMIT;
