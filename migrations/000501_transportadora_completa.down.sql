ALTER TABLE shipping_carrier_occurrences DROP CONSTRAINT IF EXISTS shipping_carrier_occurrences_carrier_id_fkey;
ALTER TABLE shipping_carrier_occurrences
    ADD CONSTRAINT shipping_carrier_occurrences_carrier_id_fkey FOREIGN KEY (carrier_id) REFERENCES shipping_carriers(id) ON DELETE CASCADE;
ALTER TABLE shipping_carrier_occurrences DROP COLUMN IF EXISTS shipment_id;
SELECT instalar_conferencia_de_empresa();

DROP TABLE IF EXISTS shipping_occurrence_types;

ALTER TABLE shipping_carrier_vehicles
    DROP CONSTRAINT IF EXISTS shipping_carrier_vehicles_mdfe_check,
    DROP COLUMN IF EXISTS vehicle_type_id,
    DROP COLUMN IF EXISTS plate_state,
    DROP COLUMN IF EXISTS renavam,
    DROP COLUMN IF EXISTS tare_kg,
    DROP COLUMN IF EXISTS wheel_type,
    DROP COLUMN IF EXISTS body_type,
    DROP COLUMN IF EXISTS owner_type,
    DROP COLUMN IF EXISTS driver_cnh_category,
    DROP COLUMN IF EXISTS driver_cnh_expiry;
SELECT instalar_conferencia_de_empresa();

DROP TABLE IF EXISTS shipping_vehicle_types;

ALTER TABLE shipping_carriers
    DROP CONSTRAINT IF EXISTS shipping_carriers_cubagem_check,
    DROP COLUMN IF EXISTS cubage_factor,
    DROP COLUMN IF EXISTS dispatch_fee,
    DROP COLUMN IF EXISTS accepts_dangerous_goods;
