-- A 449 cancelou as demandas que a confirmação do pedido gerava (pedido ×
-- 100000 + sequência), mas só quando a quantidade ainda batia com a da linha.
-- A linha alterada depois da confirmação deixava a demanda duplicada aberta.
-- A demanda gerada pela confirmação se reconhece pelo código, pelo item e pelo
-- autor (o mesmo do pedido), sem depender da quantidade.
UPDATE independent_demands d
   SET situacao = 'CANCELADA', is_active = FALSE, encerrada_em = NOW(),
       motivo_encerramento = 'Gerada pela confirmação do pedido de venda ' || so.order_number
                             || ' (linha alterada depois): duplicava o pedido no MRP, que lê o pedido diretamente.'
  FROM sales_order_items soi
  JOIN sales_orders so ON so.code = soi.sales_order_code
  JOIN enterprise e ON e.code = so.enterprise_code
 WHERE d.situacao = 'ABERTA'
   AND e.id = d.enterprise_id
   AND d.code = so.code * 100000 + soi.sequence
   AND d.item_code = soi.item_code
   AND d.created_by = so.created_by;

-- Encerrada também sai de is_active (o encerramento antes a deixava ativa):
-- quem lê is_active não pode ver demanda fechada como viva.
UPDATE independent_demands SET is_active = FALSE WHERE situacao <> 'ABERTA' AND is_active;
