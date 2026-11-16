ALTER TABLE receiving_notices DROP COLUMN IF EXISTS shipping_carrier_id;
ALTER TABLE purchase_orders  DROP COLUMN IF EXISTS shipping_carrier_id,
                             DROP COLUMN IF EXISTS redispatch_shipping_carrier_id;
ALTER TABLE shipment_loads   DROP COLUMN IF EXISTS shipping_carrier_id;
ALTER TABLE shipments        DROP COLUMN IF EXISTS shipping_carrier_id;
SELECT instalar_conferencia_de_empresa();
