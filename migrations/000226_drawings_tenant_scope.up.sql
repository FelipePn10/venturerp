ALTER TABLE drawings ADD COLUMN enterprise_id BIGINT REFERENCES enterprise(id);

UPDATE drawings drawing
SET enterprise_id = association.enterprise_id
FROM user_enterprises association
WHERE association.user_id = drawing.created_by
  AND drawing.enterprise_id IS NULL
  AND (SELECT COUNT(*) FROM user_enterprises candidate WHERE candidate.user_id = drawing.created_by) = 1;

UPDATE drawings SET enterprise_id = (SELECT MIN(id) FROM enterprise)
WHERE enterprise_id IS NULL AND (SELECT COUNT(*) FROM enterprise) = 1;

CREATE INDEX idx_drawings_tenant_item ON drawings (enterprise_id, item_code);
CREATE UNIQUE INDEX uq_drawings_tenant_code
    ON drawings (enterprise_id, code, digit, format)
    WHERE enterprise_id IS NOT NULL;

ALTER TABLE drawings DROP CONSTRAINT IF EXISTS drawings_code_digit_format_key;
