ALTER TABLE allocation_base_items
DROP CONSTRAINT IF EXISTS allocation_base_items_allocation_base_id_fkey;

ALTER TABLE allocation_base_items
DROP CONSTRAINT IF EXISTS allocation_base_items_cost_center_id_fkey;

ALTER TABLE allocation_bases
ALTER COLUMN code TYPE integer USING code::integer;

ALTER TABLE cost_centers
ALTER COLUMN code TYPE integer USING code::integer;

ALTER TABLE allocation_bases
    ADD CONSTRAINT uq_allocation_bases_code UNIQUE (code);

ALTER TABLE cost_centers
    ADD CONSTRAINT uq_cost_centers_code UNIQUE (code);

ALTER TABLE allocation_base_items
    ADD COLUMN allocation_base_code integer,
ADD COLUMN cost_center_code integer;

UPDATE allocation_base_items abi
SET allocation_base_code = ab.code
    FROM allocation_bases ab
WHERE abi.allocation_base_id = ab.id;

UPDATE allocation_base_items abi
SET cost_center_code = cc.code
    FROM cost_centers cc
WHERE abi.cost_center_id = cc.id;

ALTER TABLE allocation_base_items
    ALTER COLUMN allocation_base_code SET NOT NULL;

ALTER TABLE allocation_base_items
    ALTER COLUMN cost_center_code SET NOT NULL;

ALTER TABLE allocation_base_items
DROP COLUMN allocation_base_id,
DROP COLUMN cost_center_id;

ALTER TABLE allocation_base_items
    ADD CONSTRAINT fk_base_code
        FOREIGN KEY (allocation_base_code)
            REFERENCES allocation_bases(code)
            ON DELETE CASCADE;

ALTER TABLE allocation_base_items
    ADD CONSTRAINT fk_cost_center_code
        FOREIGN KEY (cost_center_code)
            REFERENCES cost_centers(code);