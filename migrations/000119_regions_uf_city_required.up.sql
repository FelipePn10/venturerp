-- Make UF and city mandatory on regions
UPDATE regions SET uf = '', city = '' WHERE uf IS NULL OR city IS NULL;

ALTER TABLE regions
    ALTER COLUMN uf   SET NOT NULL,
    ALTER COLUMN city SET NOT NULL;
