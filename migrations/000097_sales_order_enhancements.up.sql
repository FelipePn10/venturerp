BEGIN;

ALTER TABLE public.sales_orders
    ADD COLUMN IF NOT EXISTS representative_order_number  BIGINT,
    ADD COLUMN IF NOT EXISTS is_nfce                       BOOLEAN       NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS street                        TEXT,
    ADD COLUMN IF NOT EXISTS street_number                 VARCHAR(20),
    ADD COLUMN IF NOT EXISTS foreign_document              VARCHAR(20),
    ADD COLUMN IF NOT EXISTS collection_establishment_code BIGINT,
    ADD COLUMN IF NOT EXISTS nf_type_description           VARCHAR(100),
    ADD COLUMN IF NOT EXISTS carrier_code                  BIGINT,
    ADD COLUMN IF NOT EXISTS freight_type                  VARCHAR(20),
    ADD COLUMN IF NOT EXISTS freight_value                 NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS insurance_value               NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS volume_quantity               NUMERIC(15,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS volume_type                   VARCHAR(100),
    ADD COLUMN IF NOT EXISTS net_weight                    NUMERIC(15,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS gross_weight                  NUMERIC(15,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS discount_value                NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS surcharge_value               NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS project_code                  VARCHAR(50),
    ADD COLUMN IF NOT EXISTS project_name                  VARCHAR(200);

COMMIT;
