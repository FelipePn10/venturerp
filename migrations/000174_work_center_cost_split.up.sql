BEGIN;

-- ─────────────────────────────────────────────────────────────────────────────
-- Split the work-center hourly cost into machine vs. labor rates (enterprise+ cost).
--
-- The legacy `cost_per_hour` is kept as the blended/machine rate. The standard-cost
-- roll-up now charges each route operation at its OWN work-center rate (no more naive
-- average across all work centers) and uses the rich time model:
--   labor_cost = Σ [ MachineHours(qty) × machine_cost_per_hour
--                  + LaborHours(qty)   × labor_cost_per_hour ]  (per operation, per CT)
-- ─────────────────────────────────────────────────────────────────────────────

ALTER TABLE work_center_costs
    ADD COLUMN IF NOT EXISTS machine_cost_per_hour NUMERIC(15,4),
    ADD COLUMN IF NOT EXISTS labor_cost_per_hour   NUMERIC(15,4);

-- Back-fill: the existing blended rate becomes the machine rate; labor starts at 0
-- (set it explicitly to enable the machine × labor split for a work center).
UPDATE work_center_costs SET machine_cost_per_hour = cost_per_hour WHERE machine_cost_per_hour IS NULL;
UPDATE work_center_costs SET labor_cost_per_hour = 0 WHERE labor_cost_per_hour IS NULL;

COMMIT;
