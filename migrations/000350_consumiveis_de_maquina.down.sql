ALTER TABLE item_machine_times
    DROP CONSTRAINT IF EXISTS item_machine_times_consumable_fk,
    DROP CONSTRAINT IF EXISTS item_machine_times_consumption_pair,
    DROP CONSTRAINT IF EXISTS item_machine_times_consumption_check;

ALTER TABLE item_machine_times
    DROP COLUMN IF EXISTS consumption_per_hour,
    DROP COLUMN IF EXISTS consumable_id;

DROP TABLE IF EXISTS machine_consumables;
