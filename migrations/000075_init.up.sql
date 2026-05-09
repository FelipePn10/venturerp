ALTER TABLE machines
    ADD CONSTRAINT chk_efficiency_rate
        CHECK (efficiency_rate >= 0 AND efficiency_rate <= 1);

ALTER TABLE item_machine_times
    ADD COLUMN IF NOT EXISTS production_time_unit capacity_period_enum NOT NULL DEFAULT 'DIA',
    ADD COLUMN IF NOT EXISTS production_base_qty INTEGER NOT NULL;