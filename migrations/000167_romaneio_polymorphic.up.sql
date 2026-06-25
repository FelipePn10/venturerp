BEGIN;

-- Polymorphic reference: romaneios can now be linked to sales orders,
-- purchase orders, and production orders.
ALTER TABLE shipments
  ADD COLUMN IF NOT EXISTS reference_type        VARCHAR(30),
  ADD COLUMN IF NOT EXISTS purchase_order_code   BIGINT,
  ADD COLUMN IF NOT EXISTS production_order_code BIGINT;

CREATE INDEX IF NOT EXISTS idx_shipments_reference      ON shipments(reference_type, sales_order_code);
CREATE INDEX IF NOT EXISTS idx_shipments_purchase_order ON shipments(purchase_order_code);
CREATE INDEX IF NOT EXISTS idx_shipments_production_order ON shipments(production_order_code);

COMMIT;
