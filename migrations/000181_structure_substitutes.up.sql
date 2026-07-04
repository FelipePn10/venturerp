BEGIN;

-- ─────────────────────────────────────────────────────────────────────────────
-- Alternative / substitute components on the BOM (enterprise+ B1).
--
-- Lines of the same parent sharing a `substitute_group` (> 0) are mutually
-- substitutable. The PRIMARY of a group is the member with the lowest
-- `substitute_priority` (ties → lowest sequence/id). MRP demand and standard cost
-- consider ONLY the primary; the others are alternatives used when the primary is
-- short (shop-floor/manual substitution). substitute_group = 0/NULL → standalone.
-- ─────────────────────────────────────────────────────────────────────────────

ALTER TABLE item_structures
    ADD COLUMN IF NOT EXISTS substitute_group    SMALLINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS substitute_priority SMALLINT NOT NULL DEFAULT 1;

CREATE INDEX IF NOT EXISTS idx_item_structures_subgroup
    ON item_structures(parent_code, substitute_group) WHERE substitute_group > 0;

COMMIT;
