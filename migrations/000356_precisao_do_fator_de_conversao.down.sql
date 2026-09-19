-- Voltar para seis casas ARREDONDA os fatores já gravados; as linhas que
-- dependiam da precisão passam a consumir uma quantidade levemente diferente.
ALTER TABLE item_unit_conversions ALTER COLUMN factor TYPE NUMERIC(18,6);
ALTER TABLE item_structures
    ALTER COLUMN quantity_stock_uom TYPE NUMERIC(18,6),
    ALTER COLUMN conversion_factor TYPE NUMERIC(18,6);
