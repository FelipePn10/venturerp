ALTER TABLE item_machine_times
    ADD COLUMN machine_code BIGINT;

UPDATE item_machine_times imt
SET machine_code = m.code
    FROM machines m
WHERE imt.machine_id = m.id;

ALTER TABLE item_machine_times
    ALTER COLUMN machine_code SET NOT NULL;

ALTER TABLE item_machine_times
    ADD CONSTRAINT item_machine_times_unique_code
        UNIQUE (item_code, mask, machine_code);

ALTER TABLE item_machine_times
DROP COLUMN machine_id;

