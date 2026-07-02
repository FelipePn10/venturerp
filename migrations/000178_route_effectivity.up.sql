BEGIN;

-- ─────────────────────────────────────────────────────────────────────────────
-- Route effectivity dates (enterprise+ R6).
--
-- A manufacturing route can now be valid only within a date window. NULL bounds are
-- open (valid_from NULL = always valid up to valid_to; valid_to NULL = open-ended).
-- Time-phased revisions use distinct `alternative` numbers, each scoped by its window;
-- the standard-route lookup picks the one effective on the reference date.
-- ─────────────────────────────────────────────────────────────────────────────

ALTER TABLE manufacturing_routes
    ADD COLUMN IF NOT EXISTS valid_from DATE,
    ADD COLUMN IF NOT EXISTS valid_to   DATE;

ALTER TABLE manufacturing_routes
    ADD CONSTRAINT chk_mfg_routes_validity
    CHECK (valid_from IS NULL OR valid_to IS NULL OR valid_to >= valid_from);

CREATE INDEX IF NOT EXISTS idx_mfg_routes_validity ON manufacturing_routes(item_code, valid_from, valid_to);

COMMIT;
