BEGIN;

DROP TABLE IF EXISTS shipment_events;
DROP TABLE IF EXISTS shipment_volumes;

ALTER TABLE shipment_items
    DROP COLUMN IF EXISTS unit_net_weight,
    DROP COLUMN IF EXISTS unit_gross_weight;

ALTER TABLE shipments
    DROP COLUMN IF EXISTS total_net_weight,
    DROP COLUMN IF EXISTS total_gross_weight,
    DROP COLUMN IF EXISTS total_cubage_m3,
    DROP COLUMN IF EXISTS freight_modality,
    DROP COLUMN IF EXISTS freight_value,
    DROP COLUMN IF EXISTS insurance_value,
    DROP COLUMN IF EXISTS vehicle_plate,
    DROP COLUMN IF EXISTS driver_name,
    DROP COLUMN IF EXISTS driver_document,
    DROP COLUMN IF EXISTS antt_code,
    DROP COLUMN IF EXISTS seals,
    DROP COLUMN IF EXISTS estimated_delivery,
    DROP COLUMN IF EXISTS fiscal_exit_id,
    DROP COLUMN IF EXISTS nfe_number,
    DROP COLUMN IF EXISTS nfe_key,
    DROP COLUMN IF EXISTS separated_at,
    DROP COLUMN IF EXISTS conferred_at,
    DROP COLUMN IF EXISTS cancelled_at,
    DROP COLUMN IF EXISTS updated_by;

COMMIT;
