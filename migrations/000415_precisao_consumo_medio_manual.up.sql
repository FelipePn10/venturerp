-- Consumo médio mensal informado no item com a precisão do estoque: item em KG
-- ou M consome 12,5 por mês, e o inteiro truncava o valor que alimenta o ponto
-- de pedido e a reposição. Os valores atuais são inteiros: nada muda neles.
ALTER TABLE items
    ALTER COLUMN warehouse_avg_monthly_consumption_manual TYPE NUMERIC(18,6);
