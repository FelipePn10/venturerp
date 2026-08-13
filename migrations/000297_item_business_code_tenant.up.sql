BEGIN;

CREATE SEQUENCE IF NOT EXISTS items_legacy_code_seq;
SELECT setval(
    'items_legacy_code_seq',
    GREATEST(COALESCE((SELECT MAX(code) FROM items), 0), 1),
    COALESCE((SELECT MAX(code) FROM items), 0) > 0
);

ALTER TABLE items
    ADD COLUMN IF NOT EXISTS enterprise_id BIGINT,
    ADD COLUMN IF NOT EXISTS business_code VARCHAR(60);

UPDATE items
SET enterprise_id = (SELECT id FROM enterprise ORDER BY id LIMIT 1)
WHERE enterprise_id IS NULL
  AND (SELECT COUNT(*) FROM enterprise) = 1;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM items WHERE enterprise_id IS NULL) THEN
        RAISE EXCEPTION 'Nao foi possivel determinar a empresa de todos os itens existentes';
    END IF;
END $$;

UPDATE items
SET business_code = upper(btrim(code::text))
WHERE business_code IS NULL;

ALTER TABLE items
    ALTER COLUMN enterprise_id SET NOT NULL,
    ALTER COLUMN business_code SET NOT NULL,
    ALTER COLUMN code SET DEFAULT nextval('items_legacy_code_seq');

ALTER SEQUENCE items_legacy_code_seq OWNED BY items.code;

ALTER TABLE items
    ADD CONSTRAINT fk_items_enterprise
        FOREIGN KEY (enterprise_id) REFERENCES enterprise(id),
    ADD CONSTRAINT chk_items_business_code_format
        CHECK (business_code ~ '^[A-Z0-9][A-Z0-9._/-]{0,59}$'),
    ADD CONSTRAINT uq_items_enterprise_business_code
        UNIQUE (enterprise_id, business_code);

CREATE INDEX idx_items_enterprise_business_code
    ON items (enterprise_id, business_code);

COMMIT;
