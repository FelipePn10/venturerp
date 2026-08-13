BEGIN;
ALTER TABLE fiscal_classifications
    ADD COLUMN IF NOT EXISTS enterprise_id BIGINT,
    ADD COLUMN IF NOT EXISTS valid_from DATE NOT NULL DEFAULT CURRENT_DATE,
    ADD COLUMN IF NOT EXISTS valid_until DATE,
    ADD COLUMN IF NOT EXISTS default_origin SMALLINT,
    ADD COLUMN IF NOT EXISTS default_icms_rate NUMERIC(15,4),
    ADD COLUMN IF NOT EXISTS default_calculate_pis_cofins BOOLEAN;
UPDATE fiscal_classifications SET enterprise_id=(SELECT id FROM enterprise ORDER BY id LIMIT 1)
WHERE enterprise_id IS NULL AND (SELECT COUNT(*) FROM enterprise)=1;
DO $$ BEGIN IF EXISTS(SELECT 1 FROM fiscal_classifications WHERE enterprise_id IS NULL) THEN RAISE EXCEPTION 'Classificacoes fiscais legadas ambiguas entre empresas'; END IF; END $$;
ALTER TABLE fiscal_classifications ALTER COLUMN enterprise_id SET NOT NULL;
ALTER TABLE fiscal_classifications DROP CONSTRAINT IF EXISTS fiscal_classifications_code_key;
ALTER TABLE fiscal_classifications ADD CONSTRAINT uq_fiscal_classifications_enterprise_code UNIQUE(enterprise_id,code);
ALTER TABLE fiscal_classifications ADD CONSTRAINT fk_fiscal_classifications_enterprise FOREIGN KEY(enterprise_id) REFERENCES enterprise(id);
ALTER TABLE fiscal_classifications ADD CONSTRAINT chk_fiscal_classification_validity CHECK(valid_until IS NULL OR valid_until>=valid_from);
ALTER TABLE fiscal_classifications ADD CONSTRAINT chk_fiscal_classification_origin CHECK(default_origin IS NULL OR default_origin BETWEEN 0 AND 8);
CREATE INDEX idx_fiscal_classification_tenant_active ON fiscal_classifications(enterprise_id,code) WHERE is_active;
COMMIT;
