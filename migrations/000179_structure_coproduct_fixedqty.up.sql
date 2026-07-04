BEGIN;

-- ─────────────────────────────────────────────────────────────────────────────
-- Co-products / by-products and fixed-quantity components on the BOM (enterprise+).
--
-- is_coproduct = TRUE  → the "child" is an OUTPUT of producing the parent (co-product,
--   by-product or returnable scrap), not a consumed input. The MRP does not explode
--   dependent demand for it, and the standard cost CREDITS the parent by its value.
--
-- is_fixed_qty = TRUE  → the component quantity is per production order (lot), not per
--   parent unit. The MRP consumes `quantity` once (loss-adjusted) regardless of the
--   order size, and the standard cost amortizes it over the reference lot.
-- ─────────────────────────────────────────────────────────────────────────────

ALTER TABLE item_structures
    ADD COLUMN IF NOT EXISTS is_coproduct BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS is_fixed_qty BOOLEAN NOT NULL DEFAULT FALSE;

COMMIT;
