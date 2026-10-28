-- Quantidades de planejamento do item, da qualidade e da assistência técnica
-- com a precisão do estoque (migração 407).
--
-- Estoque mínimo, lote mínimo, lote múltiplo e estoque de segurança eram
-- inteiros: um item comprado em KG ou M não aceitava 12,5 KG de segurança nem
-- lote múltiplo de 6,35 M (barra), e o valor digitado era truncado. Na
-- qualidade e na assistência a quantidade tinha 4 casas, menos que o saldo
-- (6 casas): a inspeção de 15,530965 M de um recebimento convertido não
-- conferia com o estoque. Os valores atuais são inteiros ou de 4 casas, então
-- a conversão não altera nenhum dado gravado.
ALTER TABLE items
    ALTER COLUMN warehouse_minimum_stock TYPE NUMERIC(18,6),
    ALTER COLUMN planning_minimum_lot    TYPE NUMERIC(18,6),
    ALTER COLUMN planning_multiple_lot   TYPE NUMERIC(18,6),
    ALTER COLUMN planning_safety_stock   TYPE NUMERIC(18,6);

ALTER TABLE quality_records
    ALTER COLUMN inspected_qty TYPE NUMERIC(18,6),
    ALTER COLUMN approved_qty  TYPE NUMERIC(18,6),
    ALTER COLUMN rejected_qty  TYPE NUMERIC(18,6);
ALTER TABLE non_conformances
    ALTER COLUMN nonconform_qty TYPE NUMERIC(18,6);

ALTER TABLE technical_assistance_call_items
    ALTER COLUMN quantity TYPE NUMERIC(18,6);
ALTER TABLE technical_assistance_rma_items
    ALTER COLUMN quantity TYPE NUMERIC(18,6);
