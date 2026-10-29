UPDATE independent_demands
   SET situacao = 'ABERTA', is_active = TRUE, encerrada_em = NULL, motivo_encerramento = NULL
 WHERE situacao = 'CANCELADA'
   AND motivo_encerramento LIKE 'Gerada pela confirmação do pedido de venda % (linha alterada depois):%';
