-- Sem efeito de propósito. A subida só copiou condições para `payment_conditions`
-- (a tabela antiga ficou intacta); depois disso elas podem estar em cliente,
-- fornecedor, orçamento ou pedido, e apagá-las quebraria esses vínculos.
SELECT 1;
