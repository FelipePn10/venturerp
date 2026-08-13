UPDATE items SET accounting_calculate_pis_cofins = FALSE
WHERE accounting_calculate_pis_cofins IS NULL;

ALTER TABLE items
    ALTER COLUMN accounting_calculate_pis_cofins SET DEFAULT FALSE,
    ALTER COLUMN accounting_calculate_pis_cofins SET NOT NULL;

COMMENT ON COLUMN items.accounting_calculate_pis_cofins IS NULL;
