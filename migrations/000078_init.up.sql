ALTER TABLE configured_item_rules
    ADD COLUMN IF NOT EXISTS code BIGSERIAL;