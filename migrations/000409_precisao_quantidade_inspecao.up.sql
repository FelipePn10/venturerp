-- Quantidades da inspeção de recebimento com a precisão do estoque.
--
-- O recebimento convertido entra na inspeção com 6 casas (100 KG × 0,15530965 =
-- 15,530965 M), mas a ordem de inspeção guardava 15,5310: liberar a quantidade
-- exibida tentava tirar mais do que a inspeção tinha e era recusado. Mesma escala
-- da quantidade do estoque (migração 407).
ALTER TABLE receiving_inspection_orders
    ALTER COLUMN quantity       TYPE NUMERIC(18,6),
    ALTER COLUMN inspected_qty  TYPE NUMERIC(18,6),
    ALTER COLUMN approved_qty   TYPE NUMERIC(18,6),
    ALTER COLUMN rejected_qty   TYPE NUMERIC(18,6),
    ALTER COLUMN rework_qty     TYPE NUMERIC(18,6),
    ALTER COLUMN restricted_qty TYPE NUMERIC(18,6);
ALTER TABLE receiving_inspection_analyses
    ALTER COLUMN conform_qty    TYPE NUMERIC(18,6),
    ALTER COLUMN rejected_qty   TYPE NUMERIC(18,6),
    ALTER COLUMN rework_qty     TYPE NUMERIC(18,6),
    ALTER COLUMN restricted_qty TYPE NUMERIC(18,6);
ALTER TABLE procurement_records
    ALTER COLUMN quantity TYPE NUMERIC(18,6);
ALTER TABLE procurement_inspection_dispositions
    ALTER COLUMN approved_qty TYPE NUMERIC(18,6),
    ALTER COLUMN rejected_qty TYPE NUMERIC(18,6);
