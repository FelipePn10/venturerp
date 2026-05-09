DROP TABLE IF EXISTS item_machine_usages;

ALTER TABLE item_machine_times
    ALTER COLUMN code SET NOT NULL;

ALTER TABLE item_machine_times
    ADD CONSTRAINT item_machine_times_machine_code_fkey
        FOREIGN KEY (machine_code)
            REFERENCES machines(code)
            ON DELETE CASCADE;

ALTER TABLE item_machine_times
    ADD CONSTRAINT item_machine_times_item_code_fkey
        FOREIGN KEY (item_code)
            REFERENCES items(code)
            ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_item_machine_times_machine_code
    ON item_machine_times(machine_code);

CREATE INDEX IF NOT EXISTS idx_item_machine_times_priority
    ON item_machine_times(item_code, priority);