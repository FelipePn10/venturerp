BEGIN;

-- ─────────────────────────────────────────────────────────────────────────────
-- Subcontracting attributes for external / third-party operations (enterprise+ R4).
--
-- Operations with origin EXTERNA / TERCEIROS can name the supplier, the service item
-- to buy, its unit cost and lead time. When a production order is firmed, these drive
-- an automatic service purchase requisition (gancho com compras). Defaults live on
-- `operations`; a route operation may override each on `route_operations`.
--
-- Kept as nullable logical references (no FK) to match the existing loose-coupling
-- style and to allow configuring an operation before its service item/supplier exist.
-- ─────────────────────────────────────────────────────────────────────────────

ALTER TABLE operations
    ADD COLUMN IF NOT EXISTS supplier_id       BIGINT,
    ADD COLUMN IF NOT EXISTS service_item_code BIGINT,
    ADD COLUMN IF NOT EXISTS cost_per_unit     NUMERIC(15,4),
    ADD COLUMN IF NOT EXISTS lead_time_days    INTEGER;

ALTER TABLE route_operations
    ADD COLUMN IF NOT EXISTS supplier_id       BIGINT,
    ADD COLUMN IF NOT EXISTS service_item_code BIGINT,
    ADD COLUMN IF NOT EXISTS cost_per_unit     NUMERIC(15,4),
    ADD COLUMN IF NOT EXISTS lead_time_days    INTEGER;

COMMIT;
