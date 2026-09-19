-- O fator de conversão precisa de mais casas decimais do que seis.
--
-- Um tubo de 6000 mm é cadastrado como "1 barra = 6000 MM" — número inteiro,
-- fácil de conferir na nota do fornecedor. Mas a estrutura consome MILÍMETROS,
-- então o sistema usa a INVERSA: 1/6000 = 0,000166666…
--
-- Em NUMERIC(18,6) isso vira 0,000167. A diferença parece desprezível até
-- multiplicar: 104 mm × 0,000167 = 0,017368 barra, contra os 0,017333 corretos.
-- São 0,2% a mais em cada linha — que reaparecem no consumo da ordem, no custo
-- do produto e no saldo do estoque, sempre para o mesmo lado, e ninguém
-- consegue explicar de onde vem a sobra.
--
-- Dez casas cobrem a inversa de qualquer fator até dez milhões, que é mais do
-- que qualquer conversão real de material.

ALTER TABLE item_structures
    ALTER COLUMN conversion_factor TYPE NUMERIC(20,10),
    -- A quantidade convertida acompanha o fator. Se ela ficasse com menos casas
    -- que ele, ler a coluna e recalcular pelo fator voltariam a dar números
    -- diferentes — só que agora com o recálculo mais preciso que o gravado, o
    -- inverso do problema anterior. Duas respostas para a mesma pergunta é o
    -- que se está eliminando; a precisão é o meio.
    ALTER COLUMN quantity_stock_uom TYPE NUMERIC(20,10);

ALTER TABLE item_unit_conversions
    ALTER COLUMN factor TYPE NUMERIC(20,10);

COMMENT ON COLUMN item_structures.conversion_factor IS
    'Fator aplicado (1 unidade_da_estrutura = fator × unidade_de_estoque), congelado na gravação. Dez casas porque a inversa de um fator grande (1/6000) não cabe em seis.';
