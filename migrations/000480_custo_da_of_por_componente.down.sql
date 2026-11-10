ALTER TABLE production_order_costs
    DROP COLUMN IF EXISTS machine_cost_real,
    DROP COLUMN IF EXISTS machine_cost_std,
    DROP COLUMN IF EXISTS machine_variance,
    DROP COLUMN IF EXISTS subcontract_cost_real,
    DROP COLUMN IF EXISTS subcontract_cost_std,
    DROP COLUMN IF EXISTS subcontract_variance;
