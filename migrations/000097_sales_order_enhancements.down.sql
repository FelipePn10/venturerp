BEGIN;

ALTER TABLE public.sales_orders
    DROP COLUMN IF EXISTS representative_order_number,
    DROP COLUMN IF EXISTS is_nfce,
    DROP COLUMN IF EXISTS street,
    DROP COLUMN IF EXISTS street_number,
    DROP COLUMN IF EXISTS foreign_document,
    DROP COLUMN IF EXISTS collection_establishment_code,
    DROP COLUMN IF EXISTS nf_type_description,
    DROP COLUMN IF EXISTS carrier_code,
    DROP COLUMN IF EXISTS freight_type,
    DROP COLUMN IF EXISTS freight_value,
    DROP COLUMN IF EXISTS insurance_value,
    DROP COLUMN IF EXISTS volume_quantity,
    DROP COLUMN IF EXISTS volume_type,
    DROP COLUMN IF EXISTS net_weight,
    DROP COLUMN IF EXISTS gross_weight,
    DROP COLUMN IF EXISTS discount_value,
    DROP COLUMN IF EXISTS surcharge_value,
    DROP COLUMN IF EXISTS project_code,
    DROP COLUMN IF EXISTS project_name;

COMMIT;
