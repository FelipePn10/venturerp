DROP TABLE IF EXISTS item_atributos_livres;
DELETE FROM item_gtins WHERE gtin IS NULL;
ALTER TABLE item_gtins DROP COLUMN IF EXISTS parent_quantity;
ALTER TABLE item_gtins DROP COLUMN IF EXISTS parent_uom;
ALTER TABLE item_gtins ALTER COLUMN gtin SET NOT NULL;
