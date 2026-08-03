-- Compensatory migration for databases that had already advanced beyond 242
-- when 000242_item_master_data_fixes was published.
ALTER TYPE unit_of_measurement_enum ADD VALUE IF NOT EXISTS 'L';
ALTER TYPE unit_of_measurement_enum ADD VALUE IF NOT EXISTS 'CX';
ALTER TYPE unit_of_measurement_enum ADD VALUE IF NOT EXISTS 'PC';
ALTER TYPE unit_of_measurement_enum ADD VALUE IF NOT EXISTS 'GL';
ALTER TYPE unit_of_measurement_enum ADD VALUE IF NOT EXISTS 'PAR';
