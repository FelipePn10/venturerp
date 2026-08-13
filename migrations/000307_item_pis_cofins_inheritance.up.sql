ALTER TABLE items
    ALTER COLUMN accounting_calculate_pis_cofins DROP DEFAULT,
    ALTER COLUMN accounting_calculate_pis_cofins DROP NOT NULL;

COMMENT ON COLUMN items.accounting_calculate_pis_cofins IS
    'NULL herda da classificação fiscal; TRUE/FALSE são sobrescritos pelo item.';
