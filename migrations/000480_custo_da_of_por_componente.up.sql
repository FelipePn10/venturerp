-- A liquidação da OF passa a ter os mesmos componentes do custo-padrão
-- (item_standard_costs): máquina (preparação + máquina) e serviço de terceiros
-- ganham colunas próprias. Antes o real só tinha material, mão de obra e
-- indireto, e o padrão comparado deixava máquina, preparação e terceiros de fora.
ALTER TABLE production_order_costs
    ADD COLUMN IF NOT EXISTS machine_cost_real     numeric(20,6) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS machine_cost_std      numeric(20,6) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS machine_variance      numeric(20,6) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS subcontract_cost_real numeric(20,6) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS subcontract_cost_std  numeric(20,6) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS subcontract_variance  numeric(20,6) NOT NULL DEFAULT 0;
