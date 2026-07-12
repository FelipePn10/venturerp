CREATE UNIQUE INDEX IF NOT EXISTS uq_mrp_single_running_calculation
    ON mrp_calculation_logs ((status))
    WHERE status = 'RUNNING';
