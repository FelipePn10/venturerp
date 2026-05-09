CREATE TYPE machine_capacity_unit_enum AS ENUM (
    'UN',
    'PEÇAS',
    'KG',
    'T',
    'CHAPAS',
    'M',
    'M2',
    'M3',
    'LITROS'
);

CREATE TYPE capacity_period_enum AS ENUM (
    'MINUTO',
    'HORA',
    'DIA'
);

ALTER TABLE machines
ADD COLUMN IF NOT EXISTS capacity_unit machine_capacity_unit_enum NOT NULL DEFAULT 'UN',
ADD COLUMN IF NOT EXISTS capacity_period capacity_period_enum NOT NULL DEFAULT 'DIA';

ALTER TABLE machines
RENAME COLUMN capacity_per_hour TO capacity;